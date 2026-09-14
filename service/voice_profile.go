package service

import (
	"context"
	"errors"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
	"math"
	"strings"
	"time"
)

func CurrentVoiceProfiles(ctx context.Context) ([]model.VoiceProfile, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return nil, errors.New("请先登录")
	}
	return repository.ListVoiceProfiles(user.ID)
}

func ValidateVoiceProfile(p *model.VoiceProfile) error {
	p.CharacterID = strings.TrimSpace(p.CharacterID)
	if p.CharacterID == "" || len(p.CharacterID) > 128 || len(p.CharacterName) > 200 || len(p.VoicePrompt) > 4000 || len(p.VoiceProvider) > 128 || len(p.VoiceID) > 256 || len(p.DefaultEmotion) > 200 || len(p.DefaultStyle) > 500 {
		return errors.New("角色或音色字段长度无效")
	}
	if p.DefaultSpeed == 0 {
		p.DefaultSpeed = 1
	}
	if math.IsNaN(p.DefaultSpeed) || math.IsInf(p.DefaultSpeed, 0) || p.DefaultSpeed < 0.25 || p.DefaultSpeed > 4 {
		return errors.New("语速必须在0.25至4之间")
	}
	return nil
}

func SaveCurrentVoiceProfile(ctx context.Context, p model.VoiceProfile) (model.VoiceProfile, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return p, errors.New("请先登录")
	}
	if err := ValidateVoiceProfile(&p); err != nil {
		return p, err
	}
	p.UserID = user.ID
	p.UpdatedAt = time.Now().UTC()
	return p, repository.SaveVoiceProfile(p)
}
