package handler

import "testing"

func TestParseVideoTaskPayloadReadsWaveSpeedOutputs(t *testing.T) {
	payload := []byte(`{"code":200,"data":{"id":"prediction-1","status":"completed","outputs":["https://cdn.example.com/video.mp4"]}}`)
	result := parseVideoTaskPayload(payload, "wavespeed-ai/minimax-h3/image-to-video")
	if result.Status != "completed" || result.VideoURL != "https://cdn.example.com/video.mp4" {
		t.Fatalf("unexpected parsed result: %#v", result)
	}
}

func TestParseVideoTaskPayloadReadsNestedContentURL(t *testing.T) {
	payload := []byte(`{"data":{"id":"task-1","status":"success","content":{"url":"https://cdn.example.com/lec-video.mp4"}}}`)
	result := parseVideoTaskPayload(payload, "seedance")
	if result.Status != "completed" || result.VideoURL != "https://cdn.example.com/lec-video.mp4" {
		t.Fatalf("unexpected parsed result: %#v", result)
	}
}
