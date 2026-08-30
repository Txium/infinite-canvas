package service

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

const videoTaskPollInterval = 5 * time.Second

const videoTaskMaxAge = 20 * time.Minute
const videoTaskRecoveryAge = 48 * time.Hour
const videoTaskFinishedRetention = 30 * 24 * time.Hour
const videoTaskCleanupInterval = 10 * time.Minute

var (
	videoTaskPollerOnce  sync.Once
	videoTaskPollWake    = make(chan struct{}, 1)
	videoTaskPollerMu    sync.RWMutex
	videoTaskPoller      VideoTaskPollFunc
	videoTaskRunningMu   sync.Mutex
	videoTaskRunning     bool
	videoTaskWakePending bool
)

type VideoTaskCreateInput struct {
	UserID                     string
	UserDisplayName            string
	Model                      string
	UpstreamModel              string
	ChannelID                  string
	UserChannelID              string
	ChannelName                string
	Source                     string
	SourceID                   string
	ClientTaskID               string
	UpstreamTaskID             string
	UpstreamVideoID            string
	Status                     string
	Progress                   int
	Seconds                    string
	Size                       string
	VideoURL                   string
	Error                      string
	ErrorDetail                string
	RequestBody                string
	ResponseBody               string
	Credits                    int
	BillingID                  string
	BillingStatus              string
	BillingPath                string
	SalePriceCents             int64
	EstimatedProviderCostCents int64
	UpstreamRefundStatus       string
}

type VideoTaskPollUpdate struct {
	Status       string
	Progress     int
	Seconds      string
	Size         string
	VideoURL     string
	Error        string
	ErrorDetail  string
	ResponseBody string
}

type VideoTaskPollFunc func(model.VideoTask) (VideoTaskPollUpdate, error)

func CreateVideoTask(input VideoTaskCreateInput) (model.VideoTask, error) {
	current := now()
	status := NormalizeVideoTaskStatus(input.Status)
	if status == "" {
		status = "queued"
	}
	task := model.VideoTask{
		ID:                         firstVideoTaskValue(input.ClientTaskID, input.UpstreamTaskID, input.UpstreamVideoID, "video-task-"+uuid.NewString()),
		UserID:                     strings.TrimSpace(input.UserID),
		UserDisplayName:            strings.TrimSpace(input.UserDisplayName),
		Model:                      strings.TrimSpace(input.Model),
		UpstreamModel:              strings.TrimSpace(input.UpstreamModel),
		ChannelID:                  strings.TrimSpace(input.ChannelID),
		UserChannelID:              strings.TrimSpace(input.UserChannelID),
		ChannelName:                strings.TrimSpace(input.ChannelName),
		Source:                     normalizeVideoTaskSource(input.Source),
		SourceID:                   strings.TrimSpace(input.SourceID),
		UpstreamTaskID:             strings.TrimSpace(input.UpstreamTaskID),
		UpstreamVideoID:            strings.TrimSpace(input.UpstreamVideoID),
		Status:                     status,
		Progress:                   clampProgress(input.Progress),
		Seconds:                    strings.TrimSpace(input.Seconds),
		Size:                       strings.TrimSpace(input.Size),
		VideoURL:                   strings.TrimSpace(input.VideoURL),
		Error:                      strings.TrimSpace(input.Error),
		ErrorDetail:                strings.TrimSpace(input.ErrorDetail),
		RequestBody:                input.RequestBody,
		ResponseBody:               input.ResponseBody,
		LastResponse:               input.ResponseBody,
		Credits:                    input.Credits,
		BillingID:                  strings.TrimSpace(input.BillingID),
		BillingStatus:              strings.TrimSpace(input.BillingStatus),
		BillingPath:                strings.TrimSpace(input.BillingPath),
		SalePriceCents:             input.SalePriceCents,
		EstimatedProviderCostCents: input.EstimatedProviderCostCents,
		UpstreamRefundStatus:       strings.TrimSpace(input.UpstreamRefundStatus),
		CreatedAt:                  current,
		UpdatedAt:                  current,
	}
	if IsCompletedVideoTaskStatus(task.Status) || task.VideoURL != "" {
		task.Status = "completed"
		task.Progress = 100
		task.CompletedAt = current
	} else if IsFailedVideoTaskStatus(task.Status) || task.Error != "" {
		task.Status = "failed"
		task.CompletedAt = current
	}
	saved, err := repository.SaveVideoTask(task)
	if err == nil && (IsCompletedVideoTaskStatus(saved.Status) || IsFailedVideoTaskStatus(saved.Status)) {
		if err = finalizeVideoTaskBilling(&saved); err == nil {
			saved, err = repository.SaveVideoTask(saved)
		}
	}
	if err == nil && !IsCompletedVideoTaskStatus(saved.Status) && !IsFailedVideoTaskStatus(saved.Status) {
		WakeVideoTaskPoller()
	}
	return saved, err
}

func GetUserVideoTask(userID string, id string) (model.VideoTask, bool, error) {
	return repository.GetUserVideoTask(strings.TrimSpace(userID), strings.TrimSpace(id))
}

func HasActiveVideoTaskForChannel(userID string, channelID string, excludeID string) (bool, error) {
	return repository.HasActiveVideoTaskForChannel(strings.TrimSpace(userID), strings.TrimSpace(channelID), strings.TrimSpace(excludeID))
}

func ListUserVideoTasks(userID string, source string, limit int) ([]map[string]any, error) {
	tasks, err := repository.ListUserVideoTasks(strings.TrimSpace(userID), normalizeVideoTaskSource(source), limit)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, VideoTaskResponse(task))
	}
	return result, nil
}

func DeleteUserVideoTask(userID string, id string) error {
	return repository.DeleteUserVideoTask(strings.TrimSpace(userID), strings.TrimSpace(id))
}

func VideoTaskResponse(task model.VideoTask) map[string]any {
	result := map[string]any{
		"id":            task.ID,
		"object":        "video",
		"model":         task.Model,
		"source":        task.Source,
		"source_id":     task.SourceID,
		"status":        task.Status,
		"progress":      task.Progress,
		"task_id":       firstVideoTaskValue(task.UpstreamTaskID, task.ID),
		"video_id":      task.UpstreamVideoID,
		"seconds":       task.Seconds,
		"size":          task.Size,
		"created_at":    task.CreatedAt,
		"updated_at":    task.UpdatedAt,
		"started_at":    task.StartedAt,
		"completed_at":  task.CompletedAt,
		"billingStatus": task.BillingStatus,
		"createdAt":     task.CreatedAt,
		"updatedAt":     task.UpdatedAt,
	}
	if strings.TrimSpace(task.UpstreamModel) == "" {
		result["channelId"] = task.ChannelID
		result["userChannelId"] = task.UserChannelID
		result["channelName"] = task.ChannelName
		result["request_body"] = task.RequestBody
	}
	if task.VideoURL != "" {
		result["url"] = task.VideoURL
		result["video_url"] = task.VideoURL
		result["data"] = []map[string]any{{"url": task.VideoURL}}
	}
	if IsFailedVideoTaskStatus(task.Status) && (task.Error != "" || task.ErrorDetail != "") {
		result["error"] = map[string]any{"message": firstVideoTaskValue(task.Error, task.ErrorDetail)}
		result["error_detail"] = task.ErrorDetail
	}
	return result
}

func StartVideoTaskPoller(poll VideoTaskPollFunc) {
	if poll == nil {
		return
	}
	videoTaskPollerMu.Lock()
	videoTaskPoller = poll
	videoTaskPollerMu.Unlock()
	videoTaskPollerOnce.Do(func() {
		go runVideoTaskPoller()
	})
	WakeVideoTaskPoller()
}

func WakeVideoTaskPoller() {
	videoTaskRunningMu.Lock()
	if videoTaskRunning {
		videoTaskWakePending = true
		videoTaskRunningMu.Unlock()
		return
	}
	videoTaskRunning = true
	videoTaskWakePending = false
	videoTaskRunningMu.Unlock()
	select {
	case videoTaskPollWake <- struct{}{}:
	default:
		videoTaskRunningMu.Lock()
		videoTaskRunning = false
		videoTaskRunningMu.Unlock()
	}
}

func runVideoTaskPoller() {
	inFlight := sync.Map{}
	lastCleanupAt := time.Time{}
	for range videoTaskPollWake {
		for {
			current := time.Now()
			tasks, err := repository.ListDueVideoTasks(200)
			if err != nil {
				log.Printf("list due video tasks failed err=%v", err)
				waitForNextVideoTaskPoll()
				continue
			}
			recoverable, recoveryErr := repository.ListRecoverableTimedOutVideoTasks(videoTaskTime(current.Add(-videoTaskRecoveryAge)), 100)
			if recoveryErr != nil {
				log.Printf("list recoverable video tasks failed err=%v", recoveryErr)
			} else {
				tasks = append(tasks, recoverable...)
			}
			if len(tasks) == 0 {
				videoTaskRunningMu.Lock()
				if videoTaskWakePending {
					videoTaskWakePending = false
					videoTaskRunningMu.Unlock()
					continue
				}
				videoTaskRunning = false
				videoTaskRunningMu.Unlock()
				break
			}
			if lastCleanupAt.IsZero() || current.Sub(lastCleanupAt) >= videoTaskCleanupInterval {
				if err := repository.DeleteFinishedVideoTasksBefore(videoTaskTime(current.Add(-videoTaskFinishedRetention))); err != nil {
					log.Printf("cleanup finished video tasks failed err=%v", err)
				}
				lastCleanupAt = current
			}
			for _, task := range tasks {
				if videoTaskExpired(task, current) && !recoverableVideoTask(task) {
					if err := UpdateVideoTaskFromPoll(task, VideoTaskPollUpdate{
						Status:       "reconciling",
						ResponseBody: task.LastResponse,
					}); err != nil {
						log.Printf("mark video task reconciling failed id=%s err=%v", task.ID, err)
					}
					continue
				}
				if NormalizeVideoTaskStatus(task.Status) == "reconciling" && videoTaskPolledRecently(task, current, time.Minute) {
					continue
				}
				if _, loaded := inFlight.LoadOrStore(task.ID, true); loaded {
					continue
				}
				go func(task model.VideoTask) {
					defer inFlight.Delete(task.ID)
					poll := currentVideoTaskPoller()
					if poll == nil {
						return
					}
					update, err := poll(task)
					if err != nil {
						update = VideoTaskPollUpdate{Status: task.Status, ErrorDetail: err.Error()}
					}
					if err := UpdateVideoTaskFromPoll(task, update); err != nil {
						log.Printf("update video task failed id=%s err=%v", task.ID, err)
					}
				}(task)
			}
			waitForNextVideoTaskPoll()
		}
	}
}

func currentVideoTaskPoller() VideoTaskPollFunc {
	videoTaskPollerMu.RLock()
	defer videoTaskPollerMu.RUnlock()
	return videoTaskPoller
}

func waitForNextVideoTaskPoll() {
	time.Sleep(videoTaskPollInterval)
}

func UpdateVideoTaskFromPoll(task model.VideoTask, update VideoTaskPollUpdate) error {
	current := now()
	task.Status = NormalizeVideoTaskStatus(firstVideoTaskValue(update.Status, task.Status))
	if task.Status == "" {
		task.Status = "processing"
	}
	if update.Progress > 0 || task.Progress == 0 {
		task.Progress = clampProgress(update.Progress)
	}
	if strings.TrimSpace(update.Seconds) != "" {
		task.Seconds = strings.TrimSpace(update.Seconds)
	}
	if strings.TrimSpace(update.Size) != "" {
		task.Size = strings.TrimSpace(update.Size)
	}
	if strings.TrimSpace(update.VideoURL) != "" {
		task.VideoURL = strings.TrimSpace(update.VideoURL)
	}
	if strings.TrimSpace(update.Error) != "" {
		task.Error = strings.TrimSpace(update.Error)
	}
	if strings.TrimSpace(update.ErrorDetail) != "" {
		task.ErrorDetail = strings.TrimSpace(update.ErrorDetail)
	}
	if update.ResponseBody != "" {
		task.LastResponse = update.ResponseBody
	}
	task.UpdatedAt = current
	task.LastPolledAt = videoTaskTime(time.Now())
	if !IsFailedVideoTaskStatus(task.Status) {
		task.Error = ""
		task.ErrorDetail = ""
	}
	if task.VideoURL != "" || IsCompletedVideoTaskStatus(task.Status) {
		task.Status = "completed"
		task.Progress = 100
		task.CompletedAt = current
		task.Error = ""
		task.ErrorDetail = ""
	} else if task.Error != "" || IsFailedVideoTaskStatus(task.Status) {
		task.Status = "failed"
		task.CompletedAt = current
	}
	if err := finalizeVideoTaskBilling(&task); err != nil {
		return err
	}
	_, err := repository.SaveVideoTask(task)
	return err
}

func recoverableVideoTask(task model.VideoTask) bool {
	return task.Status == "failed" && task.BillingStatus == "released" && task.Error == "任务处理超时，已自动解冻" && task.ErrorDetail == "任务超过 30 分钟仍未完成"
}

func videoTaskExpired(task model.VideoTask, current time.Time) bool {
	createdAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(task.CreatedAt))
	return err == nil && current.Sub(createdAt) >= videoTaskMaxAge
}

func videoTaskPolledRecently(task model.VideoTask, current time.Time, interval time.Duration) bool {
	last, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(task.LastPolledAt))
	return err == nil && current.Sub(last) < interval
}

func finalizeVideoTaskBilling(task *model.VideoTask) error {
	if task == nil || task.Credits <= 0 || strings.TrimSpace(task.BillingID) == "" || task.BillingStatus != "frozen" {
		return nil
	}
	if IsCompletedVideoTaskStatus(task.Status) {
		return repository.SettleVideoTaskFinancials(task, now())
	} else if IsFailedVideoTaskStatus(task.Status) {
		if err := ReleaseUserCredits(task.UserID, task.Model, task.Credits, task.BillingPath, task.BillingID); err != nil {
			return err
		}
		task.BillingStatus = "released"
		if task.UpstreamRefundStatus == "" {
			task.UpstreamRefundStatus = "pending"
		}
	}
	return nil
}

func NormalizeVideoTaskStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "complete", "done", "succeeded", "success":
		return "completed"
	case "failed", "fail", "error", "cancelled", "canceled", "timeout", "deleted":
		return "failed"
	case "running", "processing", "in_progress", "in-progress":
		return "processing"
	case "reconciling", "waiting_upstream", "waiting-upstream", "manual_review":
		return "reconciling"
	case "queued", "queue", "pending", "":
		return "queued"
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func IsCompletedVideoTaskStatus(status string) bool {
	return NormalizeVideoTaskStatus(status) == "completed"
}

func IsFailedVideoTaskStatus(status string) bool {
	return NormalizeVideoTaskStatus(status) == "failed"
}

func videoTaskTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func firstVideoTaskValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeVideoTaskSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "canvas":
		return "canvas"
	case "video-workbench", "":
		return "video-workbench"
	default:
		return "video-workbench"
	}
}

func clampProgress(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
