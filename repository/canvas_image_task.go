package repository

import (
	"errors"
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
)

func ListStaleCanvasImageTasks(before string, limit int) ([]model.CanvasImageTask, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var tasks []model.CanvasImageTask
	err = db.Where("status IN ? AND created_at < ?", []string{"queued", "processing", "running", "in_progress"}, before).
		Order("created_at ASC").Limit(normalizeTaskLimit(limit)).Find(&tasks).Error
	return tasks, err
}

func SaveCanvasImageTask(task model.CanvasImageTask) (model.CanvasImageTask, error) {
	db, err := DB()
	if err != nil {
		return task, err
	}
	return task, db.Save(&task).Error
}

func CreateCanvasImageTaskIfAbsent(task model.CanvasImageTask) (model.CanvasImageTask, bool, error) {
	db, err := DB()
	if err != nil {
		return task, false, err
	}
	err = db.Create(&task).Error
	if err == nil {
		return task, true, nil
	}
	existing, found, lookupErr := GetUserCanvasImageTask(task.UserID, task.ID)
	if lookupErr != nil {
		return task, false, lookupErr
	}
	if found {
		return existing, false, nil
	}
	return task, false, err
}

func UpdateCanvasImageTask(task model.CanvasImageTask) (model.CanvasImageTask, error) {
	db, err := DB()
	if err != nil {
		return task, err
	}

	query := db.Model(&model.CanvasImageTask{}).
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
		latest, found, err := GetUserCanvasImageTask(task.UserID, task.ID)
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

func GetUserCanvasImageTask(userID string, id string) (model.CanvasImageTask, bool, error) {
	db, err := DB()
	if err != nil {
		return model.CanvasImageTask{}, false, err
	}
	var task model.CanvasImageTask
	err = db.First(&task, "user_id = ? AND id = ?", userID, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.CanvasImageTask{}, false, nil
	}
	return task, err == nil, err
}

func GetCanvasImageTaskByID(id string) (model.CanvasImageTask, bool, error) {
	db, err := DB()
	if err != nil {
		return model.CanvasImageTask{}, false, err
	}
	var task model.CanvasImageTask
	err = db.First(&task, "id = ?", strings.TrimSpace(id)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.CanvasImageTask{}, false, nil
	}
	return task, err == nil, err
}

func ReopenCanvasImageTaskForReconciliation(id string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	return db.Model(&model.CanvasImageTask{}).
		Where("id = ? AND upstream_task_id <> '' AND status = 'failed'", strings.TrimSpace(id)).
		Updates(map[string]any{
			"status": "timed_out_unknown", "error": "",
			"error_detail": "管理员正在重新查询已提交的上游任务",
			"error_code": model.GenerationErrorTimedOutUnknown,
			"completed_at": "",
		}).Error
}

func ListUserCanvasImageTasks(userID string, sources []string, limit int) ([]model.CanvasImageTask, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	var tasks []model.CanvasImageTask
	query := db.Where("user_id = ?", userID)
	if len(sources) > 0 {
		query = query.Where("source IN ?", sources)
	}
	err = query.
		Where("status IN ?", []string{"queued", "processing", "running", "in_progress", "reconciling", "timed_out_unknown"}).
		Order("created_at DESC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

func BatchUserCanvasImageTasks(userID string, ids []string) ([]model.CanvasImageTask, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	keys := uniqueTrimmedValues(ids...)
	if len(keys) == 0 {
		return []model.CanvasImageTask{}, nil
	}
	var tasks []model.CanvasImageTask
	err = db.Where("user_id = ? AND id IN ?", userID, keys).Find(&tasks).Error
	return tasks, err
}

func DeleteUserCanvasImageTask(userID string, id string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	return db.Where("user_id = ? AND id = ?", userID, strings.TrimSpace(id)).
		Where("status IN ?", []string{"completed", "failed", "cancelled", "canceled"}).Delete(&model.CanvasImageTask{}).Error
}

func DeleteUserCanvasTasks(userID string, sourceID string, nodeIDs []string) error {
	db, err := DB()
	if err != nil {
		return err
	}

	nodeIDs = uniqueTrimmedValues(nodeIDs...)

	return db.Transaction(func(tx *gorm.DB) error {
		deleteTasks := func(task any) error {
			query := tx.Where(
				"user_id = ? AND source = ? AND source_id = ?",
				userID,
				"canvas",
				sourceID,
			)
			if len(nodeIDs) > 0 {
				query = query.Where("node_id IN ?", nodeIDs)
			}
			query = query.Where("status IN ?", []string{"completed", "failed", "cancelled", "canceled"})
			return query.Delete(task).Error
		}

		if err := deleteTasks(&model.CanvasImageTask{}); err != nil {
			return err
		}
		return deleteTasks(&model.CanvasAudioTask{})
	})
}

func uniqueTrimmedValues(values ...string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			result = append(result, value)
			seen[value] = true
		}
	}
	return result
}
