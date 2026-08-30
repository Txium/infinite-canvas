package service

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

type AdminGenerationTaskQuery struct {
	Keyword       string
	Kind          string
	Status        string
	BillingStatus string
	StartedAt     string
	EndedAt       string
	Limit         int
}

func AdminGenerationTasks(query AdminGenerationTaskQuery) ([]model.AdminGenerationTask, error) {
	limit := query.Limit
	if limit <= 0 || limit > 500 { limit = 200 }
	videos, err := repository.ListRecentVideoTasks(500)
	if err != nil { return nil, err }
	images, err := repository.ListRecentCanvasImageTasks(500)
	if err != nil { return nil, err }
	audios, err := repository.ListRecentCanvasAudioTasks(500)
	if err != nil { return nil, err }
	items := make([]model.AdminGenerationTask, 0, len(videos)+len(images)+len(audios))
	ids := make([]string, 0, cap(items))
	for _, task := range videos {
		items = append(items, model.AdminGenerationTask{ID:task.ID,UserID:task.UserID,UserDisplayName:task.UserDisplayName,Kind:"video",Model:task.Model,Status:task.Status,BillingStatus:task.BillingStatus,PriceCents:task.Credits,Source:task.Source,ChannelName:task.ChannelName,UpstreamTaskID:task.UpstreamTaskID,ResultURL:publicTaskResultURL(task.VideoURL),Error:firstVideoTaskValue(task.Error,task.ErrorDetail),CreatedAt:task.CreatedAt,CompletedAt:task.CompletedAt})
		ids = append(ids, task.ID)
	}
	for _, task := range images {
		items = append(items, model.AdminGenerationTask{ID:task.ID,UserID:task.UserID,UserDisplayName:task.UserDisplayName,Kind:"image",Model:task.Model,Status:task.Status,Source:task.Source,ChannelName:task.ChannelName,ResultURL:publicTaskResultURL(task.ImageURL),Error:firstVideoTaskValue(task.Error,task.ErrorDetail),CreatedAt:task.CreatedAt,CompletedAt:task.CompletedAt})
		ids = append(ids, task.ID)
	}
	for _, task := range audios {
		items = append(items, model.AdminGenerationTask{ID:task.ID,UserID:task.UserID,UserDisplayName:task.UserDisplayName,Kind:"audio",Model:task.Model,Status:task.Status,Source:task.Source,ChannelName:task.ChannelName,ResultURL:publicTaskResultURL(task.AudioURL),Error:firstVideoTaskValue(task.Error,task.ErrorDetail),CreatedAt:task.CreatedAt,CompletedAt:task.CompletedAt})
		ids = append(ids, task.ID)
	}
	items = appendMissingVideoCallLogs(items, videos)
	logs, err := repository.ListTaskCreditLogs(ids)
	if err != nil { return nil, err }
	byTask := map[string][]model.CreditLog{}
	for _, item := range logs { byTask[item.RelatedID] = append(byTask[item.RelatedID], item) }
	for i := range items {
		for _, entry := range byTask[items[i].ID] {
			if entry.Type == model.CreditLogTypeAIFreeze { items[i].PriceCents = -entry.Amount; items[i].BillingStatus = "frozen" }
			if entry.Type == model.CreditLogTypeAISettle { items[i].BillingStatus = "settled" }
			if entry.Type == model.CreditLogTypeAIRelease { items[i].BillingStatus = "released" }
		}
		items[i].BillingStatus = strings.TrimSpace(items[i].BillingStatus)
	}
	items = filterAdminGenerationTasks(items, query)
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
	if limit > 0 && len(items) > limit { items = items[:limit] }
	return items, nil
}

// appendMissingVideoCallLogs restores historical provider calls that predate
// persisted video_tasks. This keeps the admin ledger useful without inventing
// billing entries or attaching an upstream result to an arbitrary canvas node.
func appendMissingVideoCallLogs(items []model.AdminGenerationTask, videos []model.VideoTask) []model.AdminGenerationTask {
	logs, err := ListAICallLogs(model.Query{Page: 1, PageSize: 500})
	if err != nil { return items }
	known := map[string]bool{}
	for _, task := range videos {
		known[strings.TrimSpace(task.ID)] = true
		known[strings.TrimSpace(task.UpstreamTaskID)] = true
		known[strings.TrimSpace(task.UpstreamVideoID)] = true
	}
	recovered := map[string]model.AdminGenerationTask{}
	for _, entry := range logs.Items {
		if !looksLikeVideoCallLog(entry) { continue }
		taskID, status, resultURL, message := parseGenerationCallPayload(entry.ResponseBody)
		if taskID == "" { taskID = strings.TrimSpace(entry.ID) }
		if known[taskID] { continue }
		if status == "" {
			if entry.Status >= 400 || strings.TrimSpace(entry.Error) != "" { status = "failed" } else { status = "processing" }
		}
		message = firstVideoTaskValue(message, entry.Error)
		candidate := model.AdminGenerationTask{ID:"log-"+taskID,UserID:entry.UserID,UserDisplayName:entry.UserDisplayName,Kind:"video",Model:entry.Model,Status:NormalizeVideoTaskStatus(status),Source:"历史调用日志",ChannelName:entry.ChannelName,UpstreamTaskID:taskID,ResultURL:publicTaskResultURL(resultURL),Error:message,CreatedAt:entry.CreatedAt}
		current, exists := recovered[taskID]
		if !exists || generationTaskLogScore(candidate) > generationTaskLogScore(current) { recovered[taskID] = candidate }
	}
	for _, item := range recovered { items = append(items, item) }
	return items
}

func looksLikeVideoCallLog(entry model.AICallLog) bool {
	value := strings.ToLower(strings.Join([]string{entry.Endpoint, entry.Model, entry.ChannelName}, " "))
	return strings.Contains(value, "video") || strings.Contains(value, "hailuo") || strings.Contains(value, "minimax-h3") || strings.Contains(value, "seedance")
}

func parseGenerationCallPayload(raw string) (string, string, string, string) {
	var payload any
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload) != nil { return "", "", "", "" }
	return generationPayloadValues(payload)
}

func generationPayloadValues(value any) (string, string, string, string) {
	var taskID, status, resultURL, message string
	var walk func(any)
	walk = func(node any) {
		switch typed := node.(type) {
		case map[string]any:
			for key, child := range typed {
				name := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
				text, _ := child.(string); text = strings.TrimSpace(text)
				if taskID == "" && (name == "taskid" || name == "predictionid" || name == "videoid" || name == "id") { taskID = text }
				if status == "" && (name == "status" || name == "state") { status = text }
				if resultURL == "" && (name == "url" || name == "videourl" || name == "outputurl" || name == "downloadurl") { resultURL = text }
				if message == "" && (name == "error" || name == "errormessage" || name == "message" || name == "failreason") { message = text }
				walk(child)
			}
		case []any:
			for _, child := range typed { walk(child) }
		case string:
			if resultURL == "" && (strings.HasPrefix(typed, "https://") || strings.HasPrefix(typed, "http://")) { resultURL = strings.TrimSpace(typed) }
		}
	}
	walk(value)
	return taskID, status, resultURL, message
}

func generationTaskLogScore(item model.AdminGenerationTask) int {
	score := 0
	if item.Status == "completed" { score += 100 }
	if item.Status == "failed" { score += 80 }
	if item.ResultURL != "" { score += 40 }
	if item.Error != "" { score += 20 }
	return score
}

func filterAdminGenerationTasks(items []model.AdminGenerationTask, query AdminGenerationTaskQuery) []model.AdminGenerationTask {
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	result := make([]model.AdminGenerationTask, 0, len(items))
	for _, item := range items {
		searchText := strings.ToLower(strings.Join([]string{item.ID, item.UserID, item.UserDisplayName, item.Model, item.Source, item.Error}, " "))
		if keyword != "" && !strings.Contains(searchText, keyword) { continue }
		if query.Kind != "" && item.Kind != query.Kind { continue }
		if query.Status != "" && NormalizeVideoTaskStatus(item.Status) != NormalizeVideoTaskStatus(query.Status) { continue }
		if query.BillingStatus != "" && item.BillingStatus != query.BillingStatus { continue }
		createdDate := item.CreatedAt
		if len(createdDate) >= 10 { createdDate = createdDate[:10] }
		if query.StartedAt != "" && createdDate < query.StartedAt { continue }
		if query.EndedAt != "" && createdDate > query.EndedAt { continue }
		result = append(result, item)
	}
	return result
}

func publicTaskResultURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") { return value }
	return ""
}
