package model

import "time"

const (
	MediaAIQueued          = "queued"
	MediaAIProcessing      = "processing"
	MediaAICompleted       = "completed"
	MediaAIFailed          = "failed"
	MediaAITimedOutUnknown = "timed_out_unknown"
	MediaAICancelled       = "cancelled"
)

// VoiceProfile belongs to one user and one character; provider credentials never belong here.
type VoiceProfile struct {
	UserID         string    `json:"-" gorm:"primaryKey;size:64"`
	CharacterID    string    `json:"character_id" gorm:"primaryKey;size:128"`
	CharacterName  string    `json:"character_name"`
	VoiceProvider  string    `json:"voice_provider"`
	VoiceID        string    `json:"voice_id"`
	VoicePrompt    string    `json:"voice_prompt"`
	DefaultSpeed   float64   `json:"default_speed"`
	DefaultEmotion string    `json:"default_emotion"`
	DefaultStyle   string    `json:"default_style"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SpeechSegment struct {
	SpeakerID string  `json:"speaker_id,omitempty"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
	Text      string  `json:"text,omitempty"`
}

// Contract for future asynchronous adapters; no task is queued while disabled.
type MediaAIRequest struct {
	Operation    string  `json:"operation"`
	SourceNodeID string  `json:"source_node_id"`
	SourceURL    string  `json:"source_url"`
	Text         string  `json:"text"`
	VoiceID      string  `json:"voice_id"`
	Speed        float64 `json:"speed"`
	Emotion      string  `json:"emotion"`
	Style        string  `json:"style"`
	VoicePrompt  string  `json:"voice_prompt"`
}

type MediaAIOutput struct {
	Kind     string  `json:"kind"`
	Label    string  `json:"label"`
	URL      string  `json:"output_url"`
	VoiceID  string  `json:"voice_id,omitempty"`
	Duration float64 `json:"duration"`
	Codec    string  `json:"codec"`
}

type MediaAITask struct {
	ID             string          `json:"id" gorm:"primaryKey;size:64"`
	UserID         string          `json:"-" gorm:"index;size:64"`
	Operation      string          `json:"operation"`
	SourceNodeID   string          `json:"source_node_id"`
	Adapter        string          `json:"adapter"`
	ProviderTaskID string          `json:"provider_task_id"`
	Status         string          `json:"status" gorm:"index"`
	Outputs        []MediaAIOutput `json:"outputs" gorm:"serializer:json"`
	Segments       []SpeechSegment `json:"segments" gorm:"serializer:json"`
	ErrorCode      string          `json:"error_code"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
