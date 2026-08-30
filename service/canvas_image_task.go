package service

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

type CanvasImageTaskCreateInput struct {
	UserID          string
	UserDisplayName string
	Source          string
	SourceID        string
	NodeID          string
	ClientTaskID    string
	Model           string
	ChannelID       string
	UserChannelID   string
	ChannelName     string
	Prompt          string
	GenerationType  string
	Endpoint        string
	ContentType     string
	RequestBody     string
}

func CreateCanvasImageTask(input CanvasImageTaskCreateInput) (model.CanvasImageTask, bool, error) {
	current := now()
	task := model.CanvasImageTask{
		ID:              firstVideoTaskValue(input.ClientTaskID, "canvas_image_task_"+uuid.NewString()),
		UserID:          strings.TrimSpace(input.UserID),
		UserDisplayName: strings.TrimSpace(input.UserDisplayName),
		Source:          normalizeCanvasImageTaskSource(input.Source),
		SourceID:        strings.TrimSpace(input.SourceID),
		NodeID:          strings.TrimSpace(input.NodeID),
		Model:           strings.TrimSpace(input.Model),
		ChannelID:       strings.TrimSpace(input.ChannelID),
		UserChannelID:   strings.TrimSpace(input.UserChannelID),
		ChannelName:     strings.TrimSpace(input.ChannelName),
		Status:          "queued",
		Progress:        0,
		Prompt:          strings.TrimSpace(input.Prompt),
		GenerationType:  strings.TrimSpace(input.GenerationType),
		Endpoint:        strings.TrimSpace(input.Endpoint),
		ContentType:     strings.TrimSpace(input.ContentType),
		RequestBody:     input.RequestBody,
		CreatedAt:       current,
		UpdatedAt:       current,
	}
	return repository.CreateCanvasImageTaskIfAbsent(task)
}

func GetUserCanvasImageTask(userID string, id string) (model.CanvasImageTask, bool, error) {
	return repository.GetUserCanvasImageTask(strings.TrimSpace(userID), strings.TrimSpace(id))
}

func ListUserCanvasImageTasks(userID string, sources []string, limit int) ([]map[string]any, error) {
	tasks, err := repository.ListUserCanvasImageTasks(strings.TrimSpace(userID), normalizeCanvasImageTaskSources(sources), limit)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, CanvasImageTaskResponse(task))
	}
	return result, nil
}

func BatchUserCanvasImageTasks(userID string, ids []string) ([]map[string]any, error) {
	tasks, err := repository.BatchUserCanvasImageTasks(strings.TrimSpace(userID), ids)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, CanvasImageTaskResponse(task))
	}
	return result, nil
}

func DeleteUserCanvasImageTask(userID string, id string) error {
	userID, id = strings.TrimSpace(userID), strings.TrimSpace(id)
	task, found, err := repository.GetUserCanvasImageTask(userID, id)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	if NormalizeVideoTaskStatus(task.Status) != "completed" && NormalizeVideoTaskStatus(task.Status) != "failed" {
		return errors.New("生成中的任务不能删除，请等待完成或失败")
	}
	return repository.DeleteUserCanvasImageTask(userID, id)
}

func DeleteUserCanvasTasks(userID string, sourceID string, nodeIDs []string) error {
	return repository.DeleteUserCanvasTasks(strings.TrimSpace(userID), strings.TrimSpace(sourceID), nodeIDs)
}

func SaveCanvasImageTask(task model.CanvasImageTask) (model.CanvasImageTask, error) {
	task.UpdatedAt = now()
	return repository.UpdateCanvasImageTask(task)
}

func CanvasImageTaskResponse(task model.CanvasImageTask) map[string]any {
	result := map[string]any{
		"id":             task.ID,
		"object":         "canvas.image.task",
		"source":         task.Source,
		"source_id":      task.SourceID,
		"node_id":        task.NodeID,
		"model":          task.Model,
		"status":         task.Status,
		"progress":       task.Progress,
		"prompt":         task.Prompt,
		"generationType": task.GenerationType,
		"created_at":     task.CreatedAt,
		"updated_at":     task.UpdatedAt,
		"started_at":     task.StartedAt,
		"completed_at":   task.CompletedAt,
		"createdAt":      task.CreatedAt,
		"updatedAt":      task.UpdatedAt,
	}
	if task.ImageURL != "" {
		result["url"] = task.ImageURL
		result["image_url"] = task.ImageURL
		if len(task.ImageURLs) > 0 {
			result["image_urls"] = task.ImageURLs
		}
		result["storageKey"] = task.StorageKey
		result["width"] = task.Width
		result["height"] = task.Height
		result["mimeType"] = task.MimeType
		result["bytes"] = task.Bytes
	}
	if task.Error != "" || task.ErrorDetail != "" {
		result["error"] = map[string]any{"message": userFriendlyTaskError(firstVideoTaskValue(task.Error, task.ErrorDetail), "当前模型生成失败，请稍后重试")}
	}
	return result
}

func normalizeCanvasImageTaskSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "image-workbench":
		return "image-workbench"
	case "workflow":
		return "workflow"
	case "canvas", "":
		return "canvas"
	default:
		return "canvas"
	}
}

func normalizeCanvasImageTaskSources(sources []string) []string {
	result := make([]string, 0, len(sources))
	seen := map[string]bool{}
	for _, source := range sources {
		normalized := normalizeCanvasImageTaskSource(source)
		if normalized != "" && !seen[normalized] {
			result = append(result, normalized)
			seen[normalized] = true
		}
	}
	return result
}
