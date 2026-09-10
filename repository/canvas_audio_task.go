package repository

import (
	"errors"
	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
)

func ListStaleCanvasAudioTasks(before string, limit int) ([]model.CanvasAudioTask, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var tasks []model.CanvasAudioTask
	err = db.Where("status IN ? AND created_at < ?", []string{"queued", "processing", "running", "in_progress"}, before).
		Order("created_at ASC").Limit(normalizeTaskLimit(limit)).Find(&tasks).Error
	return tasks, err
}

func SaveCanvasAudioTask(task model.CanvasAudioTask) (model.CanvasAudioTask, error) {
	db, err := DB()
	if err != nil {
		return task, err
	}
	return task, db.Save(&task).Error
}

func CreateCanvasAudioTaskIfAbsent(task model.CanvasAudioTask) (model.CanvasAudioTask, bool, error) {
	db, err := DB()
	if err != nil {
		return task, false, err
	}
	err = db.Create(&task).Error
	if err == nil {
		return task, true, nil
	}
	existing, found, lookupErr := GetUserCanvasAudioTask(task.UserID, task.ID)
	if lookupErr != nil {
		return task, false, lookupErr
	}
	if found {
		return existing, false, nil
	}
	return task, false, err
}

func UpdateCanvasAudioTask(task model.CanvasAudioTask) (model.CanvasAudioTask, error) {
	db, err := DB()
	if err != nil {
		return task, err
	}

	query := db.Model(&model.CanvasAudioTask{}).
		Where("user_id = ? AND id = ?", task.UserID, task.ID).
		Select("*")
	if task.UpstreamTaskID == "" {
		query = query.Omit("upstream_task_id", "channel_id", "channel_name")
	}
	if task.Status != "completed" {
		query = query.Where("status NOT IN ?", []string{"completed", "failed", "cancelled", "canceled"})
	}
	if task.Status == "completed" && task.StorageKey == "" {
		query = query.Where("(status <> ? OR storage_key IS NULL OR storage_key = '')", "completed")
	}
	result := query.Updates(&task)
	if result.Error != nil {
		return task, result.Error
	}
	if result.RowsAffected == 0 {
		latest, found, err := GetUserCanvasAudioTask(task.UserID, task.ID)
		if err != nil {
			return task, err
		}
		if !found {
			return task, gorm.ErrRecordNotFound
		}
		return latest, nil
	}
	return task, nil
}

func GetUserCanvasAudioTask(userID string, id string) (model.CanvasAudioTask, bool, error) {
	db, err := DB()
	if err != nil {
		return model.CanvasAudioTask{}, false, err
	}
	var task model.CanvasAudioTask
	err = db.First(&task, "user_id = ? AND id = ?", userID, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.CanvasAudioTask{}, false, nil
	}
	return task, err == nil, err
}
