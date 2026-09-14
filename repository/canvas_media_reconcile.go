package repository

import (
	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
)

func RememberCanvasUpstreamTask(userID, taskID, upstreamID, channelID, channelName string, audio bool) error {
	database, err := DB()
	if err != nil {
		return err
	}
	var taskModel any = &model.CanvasImageTask{}
	if audio {
		taskModel = &model.CanvasAudioTask{}
	}
	result := database.Model(taskModel).Where("id = ? AND user_id = ?", taskID, userID).
		Updates(map[string]any{"upstream_task_id": upstreamID, "channel_id": channelID, "channel_name": channelName, "provider_task_status": "submitted"})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func MarkCanvasImageUpstreamRequest(userID, taskID, provider, upstreamModelID, adapter, endpoint, startedAt, requestedResolution string) error {
	database, err := DB()
	if err != nil {
		return err
	}
	result := database.Model(&model.CanvasImageTask{}).Where("id = ? AND user_id = ?", taskID, userID).Updates(map[string]any{
		"provider": provider, "upstream_model_id": upstreamModelID, "adapter": adapter,
		"provider_endpoint": endpoint, "upstream_request_sent": true,
		"upstream_request_started_at": startedAt, "provider_requested_resolution": requestedResolution,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func RecordCanvasImageUpstreamResponse(userID, taskID string, status int, providerStatus string) error {
	database, err := DB()
	if err != nil {
		return err
	}
	updates := map[string]any{"upstream_http_status": status}
	if providerStatus != "" {
		updates["provider_task_status"] = providerStatus
	}
	return database.Model(&model.CanvasImageTask{}).Where("id = ? AND user_id = ?", taskID, userID).Updates(updates).Error
}

func ListCanvasMediaToReconcile(before, _ string) ([]model.CanvasImageTask, []model.CanvasAudioTask, error) {
	database, err := DB()
	if err != nil {
		return nil, nil, err
	}
	var images []model.CanvasImageTask
	var audios []model.CanvasAudioTask
	// Reconciliation is already invoked by a one-minute scheduler. Once a task
	// enters a durable reconciliation state, updated_at must not gate polling:
	// stale workers and status observers can legitimately refresh that field and
	// would otherwise starve an accepted upstream task forever.
	query := "upstream_task_id <> '' AND ((status = 'processing' AND updated_at < ?) OR status IN ('reconciling', 'timed_out_unknown'))"
	if err := database.Where(query, before).Order("updated_at ASC").Limit(50).Find(&images).Error; err != nil {
		return nil, nil, err
	}
	err = database.Where(query, before).Order("updated_at ASC").Limit(50).Find(&audios).Error
	return images, audios, err
}
