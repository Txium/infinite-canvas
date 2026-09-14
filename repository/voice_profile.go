package repository

import (
	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm/clause"
)

func ListVoiceProfiles(userID string) ([]model.VoiceProfile, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	items := []model.VoiceProfile{}
	err = db.Where("user_id = ?", userID).Order("updated_at desc").Limit(200).Find(&items).Error
	return items, err
}

func SaveVoiceProfile(item model.VoiceProfile) error {
	db, err := DB()
	if err != nil {
		return err
	}
	return db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "character_id"}}, DoUpdates: clause.AssignmentColumns([]string{"character_name", "voice_provider", "voice_id", "voice_prompt", "default_speed", "default_emotion", "default_style", "updated_at"})}).Create(&item).Error
}
