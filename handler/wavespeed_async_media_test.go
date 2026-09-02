package handler

import (
	"encoding/json"
	"testing"
)

func TestNormalizeWaveSpeedInfiniteTalkBody(t *testing.T) {
	body, _, err := normalizeWaveSpeedInfiniteTalkBody([]byte(`{"model":"infinitetalk__01","image":"https://example.com/a.png","audio":"https://example.com/a.mp3","seconds":"5","resolution":"480p"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["model"] != nil || payload["seconds"] != nil || payload["image"] == nil || payload["audio"] == nil {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestNormalizeWaveSpeedSpeechBody(t *testing.T) {
	body, _, err := normalizeWaveSpeedSpeechBody([]byte(`{"model":"speech_26_hd__01","input":"你好"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["text"] != "你好" || payload["voice_id"] == nil || payload["model"] != nil {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestNormalizeWaveSpeedSeed3DBody(t *testing.T) {
	body, _, err := normalizeWaveSpeedSeed3DBody([]byte(`{"model":"seed3d_20__01","image":"https://example.com/object.png","subdivision_level":"medium"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["model"] != nil || payload["image"] == nil {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}
