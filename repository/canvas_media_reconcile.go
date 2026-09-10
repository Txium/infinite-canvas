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
		Updates(map[string]any{"upstream_task_id": upstreamID, "channel_id": channelID, "channel_name": channelName})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func ListCanvasMediaToReconcile(before, retryBefore string) ([]model.CanvasImageTask, []model.CanvasAudioTask, error) {
	database, err := DB()
	if err != nil {
		return nil, nil, err
	}
	var images []model.CanvasImageTask
	var audios []model.CanvasAudioTask
	query := "upstream_task_id <> '' AND ((status = 'processing' AND updated_at < ?) OR (status = 'reconciling' AND updated_at < ?))"
	if err := database.Where(query, before, retryBefore).Order("updated_at ASC").Limit(50).Find(&images).Error; err != nil {
		return nil, nil, err
	}
	err = database.Where(query, before, retryBefore).Order("updated_at ASC").Limit(50).Find(&audios).Error
	return images, audios, err
}
