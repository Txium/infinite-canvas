package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMissingClientVideoTaskDoesNotRemainQueuedForever(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/videos/client_video_task_missing", nil)

	AIVideo(recorder, request, "client_video_task_missing")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for an unpersisted client task, got %d", recorder.Code)
	}
	var payload response
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code == 0 {
		t.Fatalf("missing client task must not be reported as queued success: %s", recorder.Body.String())
	}
}
