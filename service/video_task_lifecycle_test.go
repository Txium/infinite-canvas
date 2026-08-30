package service

import (
	"testing"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
)

func TestNormalizeVideoTaskStatusReconciling(t *testing.T) {
	for _, status := range []string{"reconciling", "waiting_upstream", "manual_review"} {
		if got := NormalizeVideoTaskStatus(status); got != "reconciling" { t.Fatalf("%s normalized to %s", status, got) }
	}
}

func TestVideoTaskPolledRecently(t *testing.T) {
	now := time.Now().UTC()
	task := model.VideoTask{LastPolledAt: videoTaskTime(now.Add(-30 * time.Second))}
	if !videoTaskPolledRecently(task, now, time.Minute) { t.Fatal("expected recent poll") }
	if videoTaskPolledRecently(task, now, 10*time.Second) { t.Fatal("expected poll to be due") }
}
