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

func TestProcessingAndFailedURLsAreNotCompletedArtifacts(t *testing.T) {
	for _, payload := range []string{`{"status":"processing","url":"https://example.com/poll"}`, `{"status":"failed","url":"https://example.com/preview.png","error":"rejected"}`} {
		result := parseVideoTaskPayload([]byte(payload), "public-model")
		if result.Status == "completed" || result.VideoURL != "" {
			t.Fatalf("non-result incorrectly completed: %+v", result)
		}
	}
}

func TestVideoResultPrefersOriginalOverPreview(t *testing.T) {
	payload := []byte(`{"data":{"status":"finished","preview_url":"https://cdn.example.com/preview-480p.mp4","original_url":"https://cdn.example.com/original-1080p.mp4","width":1920,"height":1080}}`)
	result := parseVideoTaskPayload(payload, "hailuo")
	if result.Status != "completed" || result.VideoURL != "https://cdn.example.com/original-1080p.mp4" {
		t.Fatalf("did not select original result: %#v", result)
	}
	if result.FinalWidth != 1920 || result.FinalHeight != 1080 {
		t.Fatalf("final dimensions missing: %#v", result)
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
