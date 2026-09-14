package handler

import (
	"encoding/json"
	"testing"
)

func TestNormalizeWaveSpeedH3VideoBodyExtractsPromptFromContent(t *testing.T) {
	body := []byte(`{"model":"hailuo_h3__01","content":[{"type":"text","text":"a quiet seaside railway"}],"resolution":"480P","duration":4,"ratio":"16:9"}`)
	normalized, contentType, err := normalizeWaveSpeedH3VideoBody(body, "application/json", "/wavespeed-ai/minimax-h3/text-to-video")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "application/json" {
		t.Fatalf("unexpected content type %q", contentType)
	}
	var payload map[string]any
	if err := json.Unmarshal(normalized, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["prompt"] != "a quiet seaside railway" {
		t.Fatalf("prompt was not extracted: %#v", payload)
	}
	if payload["resolution"] != "480p" {
		t.Fatalf("resolution changed unexpectedly: %#v", payload)
	}
	if payload["duration"] != float64(4) {
		t.Fatalf("duration changed unexpectedly: %#v", payload)
	}
}
