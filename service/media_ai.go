package service

import (
	"context"
	"errors"
	"github.com/tigerowo/infinite-canvas/model"
)

type MediaAICapability struct {
	Operation string `json:"operation"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

func MediaAICapabilities() []MediaAICapability {
	return []MediaAICapability{
		{"asr", "语音转文字", "COMING_SOON"},
		{"tts", "AI配音", "COMING_SOON"},
		{"voice_design", "音色设计", "COMING_SOON"},
		{"voice_clone", "音色克隆", "COMING_SOON"},
		{"bgm_separation", "人声/BGM分离", "COMING_SOON"},
		{"speaker_diarization", "说话人时间段识别", "COMING_SOON"},
		{"speech_separation", "重叠语音分离", "COMING_SOON"},
		{"remove_burned_subtitle", "AI去画面字幕", "COMING_SOON"},
		{"video_repair", "视频高清修复", "COMING_SOON"},
		{"video_inpainting", "视频主体消除/补绘", "COMING_SOON"},
	}
}

// Separate operation keys allow independent self-hosted or external implementations.
// Future registration must include billing, task persistence and result ownership checks.
type MediaAIAdapter interface {
	Submit(context.Context, model.MediaAIRequest) (string, error)
	Poll(context.Context, string) (model.MediaAITask, error)
}

var ErrMediaAIDisabled = errors.New("能力尚未配置，不会创建任务或调用付费模型")

type DisabledMediaAIAdapter struct{}

func (DisabledMediaAIAdapter) Submit(context.Context, model.MediaAIRequest) (string, error) {
	return "", ErrMediaAIDisabled
}
func (DisabledMediaAIAdapter) Poll(context.Context, string) (model.MediaAITask, error) {
	return model.MediaAITask{}, ErrMediaAIDisabled
}

func ResolveMediaAIAdapter(operation string) (MediaAIAdapter, error) {
	for _, c := range MediaAICapabilities() {
		if c.Operation == operation {
			return DisabledMediaAIAdapter{}, nil
		}
	}
	return nil, errors.New("不支持的媒体AI任务类型")
}
