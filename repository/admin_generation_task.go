package repository

import "github.com/tigerowo/infinite-canvas/model"

func ListRecentVideoTasks(limit int) ([]model.VideoTask, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var items []model.VideoTask
	err = db.Order("created_at DESC").Limit(normalizeTaskLimit(limit)).Find(&items).Error
	return items, err
}

func ListRecentCanvasImageTasks(limit int) ([]model.CanvasImageTask, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var items []model.CanvasImageTask
	err = db.Order("created_at DESC").Limit(normalizeTaskLimit(limit)).Find(&items).Error
	return items, err
}

func ListRecentCanvasAudioTasks(limit int) ([]model.CanvasAudioTask, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var items []model.CanvasAudioTask
	err = db.Order("created_at DESC").Limit(normalizeTaskLimit(limit)).Find(&items).Error
	return items, err
}

func ListUserVideoTasksForHistory(userID string, limit int) ([]model.VideoTask, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var items []model.VideoTask
	err = db.Where("user_id = ?", userID).Order("created_at DESC").Limit(normalizeTaskLimit(limit)).Find(&items).Error
	return items, err
}

func ListUserImageTasksForHistory(userID string, limit int) ([]model.CanvasImageTask, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var items []model.CanvasImageTask
	err = db.Where("user_id = ?", userID).Order("created_at DESC").Limit(normalizeTaskLimit(limit)).Find(&items).Error
	return items, err
}

func ListUserAudioTasksForHistory(userID string, limit int) ([]model.CanvasAudioTask, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var items []model.CanvasAudioTask
	err = db.Where("user_id = ?", userID).Order("created_at DESC").Limit(normalizeTaskLimit(limit)).Find(&items).Error
	return items, err
}

func ListTaskCreditLogs(relatedIDs []string) ([]model.CreditLog, error) {
	if len(relatedIDs) == 0 {
		return []model.CreditLog{}, nil
	}
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var items []model.CreditLog
	err = db.Where("related_id IN ? AND type IN ?", relatedIDs, []model.CreditLogType{model.CreditLogTypeAIFreeze, model.CreditLogTypeAISettle, model.CreditLogTypeAIRelease}).Order("created_at ASC").Find(&items).Error
	return items, err
}

func normalizeTaskLimit(limit int) int {
	if limit <= 0 || limit > 500 {
		return 200
	}
	return limit
}
