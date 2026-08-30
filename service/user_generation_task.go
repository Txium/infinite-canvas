package service

import (
	"sort"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

type UserGenerationTaskQuery struct {
	Kind   string
	Status string
	Days   int
	Limit  int
}

func UserGenerationTasks(user model.AuthUser, query UserGenerationTaskQuery) ([]model.UserGenerationTask, error) {
	limit := query.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	videos, err := repository.ListUserVideoTasksForHistory(user.ID, limit)
	if err != nil {
		return nil, err
	}
	images, err := repository.ListUserImageTasksForHistory(user.ID, limit)
	if err != nil {
		return nil, err
	}
	audios, err := repository.ListUserAudioTasksForHistory(user.ID, limit)
	if err != nil {
		return nil, err
	}
	items := make([]model.UserGenerationTask, 0, len(videos)+len(images)+len(audios))
	logKeys := make([]string, 0, len(videos)+len(images)+len(audios))
	byLogKey := map[string]*model.UserGenerationTask{}
	for _, task := range videos {
		price := task.SalePriceCents
		if price == 0 {
			price = int64(task.Credits)
		}
		items = append(items, userTask(task.ID, task.Model, "video", task.Status, task.BillingStatus, price, task.Progress, task.CreatedAt, task.CompletedAt, "", task.VideoURL))
		key := strings.TrimSpace(task.BillingID)
		if key == "" {
			key = task.ID
		}
		logKeys = append(logKeys, key)
		byLogKey[key] = &items[len(items)-1]
	}
	for _, task := range images {
		items = append(items, userTask(task.ID, task.Model, "image", task.Status, "", task.SalePriceCents, task.Progress, task.CreatedAt, task.CompletedAt, task.Prompt, task.ImageURL))
		logKeys = append(logKeys, task.ID)
		byLogKey[task.ID] = &items[len(items)-1]
	}
	for _, task := range audios {
		items = append(items, userTask(task.ID, task.Model, "audio", task.Status, "", task.SalePriceCents, task.Progress, task.CreatedAt, task.CompletedAt, task.Prompt, task.AudioURL))
		logKeys = append(logKeys, task.ID)
		byLogKey[task.ID] = &items[len(items)-1]
	}
	logs, err := repository.ListTaskCreditLogs(logKeys)
	if err != nil {
		return nil, err
	}
	for _, entry := range logs {
		item := byLogKey[entry.RelatedID]
		if item == nil {
			continue
		}
		if entry.Type == model.CreditLogTypeAIFreeze && item.SalePriceCents == 0 {
			item.SalePriceCents = int64(-entry.Amount)
		}
		if entry.Type == model.CreditLogTypeAIRelease {
			item.RefundAmountCents = int64(entry.Amount)
			item.Status = "refunded"
			item.UserFriendlyError = friendlyTaskError(item.Status)
		}
	}
	cutoff := time.Time{}
	if query.Days > 0 {
		cutoff = time.Now().AddDate(0, 0, -query.Days)
	}
	filtered := items[:0]
	for _, item := range items {
		if query.Kind != "" && item.TaskType != query.Kind {
			continue
		}
		if query.Status != "" && item.Status != query.Status {
			continue
		}
		if !cutoff.IsZero() {
			created, _ := time.Parse(time.RFC3339, item.CreatedAt)
			if !created.IsZero() && created.Before(cutoff) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt > filtered[j].CreatedAt })
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func userTask(id, modelName, kind, status, billing string, price int64, progress int, created, completed, input, result string) model.UserGenerationTask {
	normalized := NormalizeVideoTaskStatus(status)
	if normalized == "processing" && progress <= 0 {
		normalized = "queued"
	}
	refund := int64(0)
	if billing == "released" {
		refund = price
		normalized = "refunded"
	}
	if price < 0 {
		price = -price
	}
	input = strings.TrimSpace(input)
	if len([]rune(input)) > 120 {
		input = string([]rune(input)[:120]) + "…"
	}
	return model.UserGenerationTask{TaskID: id, DisplayModelName: modelName, VariantName: modelName, TaskType: kind, Status: normalized, SalePriceCents: price, RefundAmountCents: refund, Progress: progress, CreatedAt: created, CompletedAt: completed, DurationSeconds: taskDuration(created, completed), InputSummary: input, ResultURL: publicTaskResultURL(result), UserFriendlyError: friendlyTaskError(normalized)}
}

func taskDuration(created, completed string) int64 {
	start, e1 := time.Parse(time.RFC3339, created)
	end, e2 := time.Parse(time.RFC3339, completed)
	if e1 != nil || e2 != nil || end.Before(start) {
		return 0
	}
	return int64(end.Sub(start).Seconds())
}
func friendlyTaskError(status string) string {
	if status == "failed" || status == "refunded" {
		return "当前模型生成失败，请稍后重试"
	}
	return ""
}
