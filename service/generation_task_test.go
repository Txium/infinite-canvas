package service

import "testing"

func TestParseGenerationCallPayloadWaveSpeedCompleted(t *testing.T) {
	id, status, url, message := parseGenerationCallPayload(`{"code":200,"data":{"id":"pred-1","status":"completed","outputs":["https://cdn.example.com/hailuo.mp4"]}}`)
	if id != "pred-1" || status != "completed" || url != "https://cdn.example.com/hailuo.mp4" || message != "" {
		t.Fatalf("unexpected values: id=%q status=%q url=%q message=%q", id, status, url, message)
	}
}

func TestParseGenerationCallPayloadFailure(t *testing.T) {
	id, status, url, message := parseGenerationCallPayload(`{"data":{"task_id":"task-2","status":"failed","error_message":"REQUEST_INVALID"}}`)
	if id != "task-2" || status != "failed" || url != "" || message != "REQUEST_INVALID" {
		t.Fatalf("unexpected values: id=%q status=%q url=%q message=%q", id, status, url, message)
	}
}
