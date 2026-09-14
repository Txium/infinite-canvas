package repository

import (
	"github.com/glebarez/sqlite"
	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
	"testing"
)

func TestVoiceProfileCompositeOwnershipSchema(t *testing.T) {
	local, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = local.AutoMigrate(&model.VoiceProfile{}, &model.MediaAITask{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"user-a", "user-b"} {
		if err = local.Create(&model.VoiceProfile{UserID: id, CharacterID: "same-character", VoiceID: id, DefaultSpeed: 1}).Error; err != nil {
			t.Fatal(err)
		}
	}
	var result []model.VoiceProfile
	if err = local.Where("user_id = ?", "user-a").Find(&result).Error; err != nil || len(result) != 1 || result[0].VoiceID != "user-a" {
		t.Fatal("ownership schema failed", err, result)
	}
}
