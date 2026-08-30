package repository

import "github.com/tigerowo/infinite-canvas/model"

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
	if err := db.Create(&task).Error; err == nil {
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

	return task, db.Model(&model.CanvasAudioTask{}).
		Where("user_id = ? AND id = ?", task.UserID, task.ID).
		Select("*").
		Updates(&task).Error
}

func GetUserCanvasAudioTask(userID string, id string) (model.CanvasAudioTask, bool, error) {
	db, err := DB()
	if err != nil {
		return model.CanvasAudioTask{}, false, err
	}
	var task model.CanvasAudioTask
	err = db.First(&task, "user_id = ? AND id = ?", userID, id).Error
	if err != nil {
		return model.CanvasAudioTask{}, false, nil
	}
	return task, true, nil
}
