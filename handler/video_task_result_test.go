package handler

import (
	"testing"

	"github.com/tigerowo/infinite-canvas/service"
)

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

func TestUnavailablePersistedRouteKeepsAcceptedTaskReconciling(t *testing.T) {
	update := reconcilingVideoPollUpdate("route disabled", `{"error":"temporary"}`)
	if update.Status != "reconciling" || update.Error != "" {
		t.Fatalf("temporary route failure became terminal: %#v", update)
	}
	if service.IsFailedVideoTaskStatus(update.Status) {
		t.Fatal("temporary route failure must not release frozen funds")
	}
	if update.ErrorDetail != "route disabled" || update.ResponseBody == "" {
		t.Fatalf("reconciliation evidence missing: %#v", update)
	}
}
