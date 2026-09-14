package service

import (
	"github.com/tigerowo/infinite-canvas/model"
	"testing"
)

func TestVideoPollingLogIsNotGenerationAttempt(t *testing.T) {
	for _, method := range []string{"GET", "get"} {
		if looksLikeVideoCallLog(model.AICallLog{Method: method, Endpoint: "/videos/task_123", Model: "seedance_2__01", Status: 429}) {
			t.Fatal("polling errors must remain diagnostics, not generation attempts")
		}
	}
	if !looksLikeVideoCallLog(model.AICallLog{Method: "POST", Endpoint: "/videos", Model: "seedance_2__01"}) {
		t.Fatal("historical creation logs must still be recognized")
	}
}
