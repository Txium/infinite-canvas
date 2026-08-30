package service

import (
	"testing"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
)

func TestNormalizeVideoTaskStatusReconciling(t *testing.T) {
	for _, status := range []string{"reconciling", "waiting_upstream", "manual_review"} {
		if got := NormalizeVideoTaskStatus(status); got != "reconciling" {
			t.Fatalf("%s normalized to %s", status, got)
		}
	}
}

func TestVideoTaskPolledRecently(t *testing.T) {
	now := time.Now().UTC()
	task := model.VideoTask{LastPolledAt: videoTaskTime(now.Add(-30 * time.Second))}
	if !videoTaskPolledRecently(task, now, time.Minute) {
		t.Fatal("expected recent poll")
	}
	if videoTaskPolledRecently(task, now, 10*time.Second) {
		t.Fatal("expected poll to be due")
	}
}

func TestVideoTaskEntersReconciliationAfterTwentyMinutes(t *testing.T) {
	now := time.Now().UTC()
	if !videoTaskExpired(model.VideoTask{CreatedAt: videoTaskTime(now.Add(-21 * time.Minute))}, now) {
		t.Fatal("expected 21 minute task to expire")
	}
	if videoTaskExpired(model.VideoTask{CreatedAt: videoTaskTime(now.Add(-19 * time.Minute))}, now) {
		t.Fatal("did not expect 19 minute task to expire")
	}
}

func TestExpiredVideoTaskRemainsPollableAfterReconciliation(t *testing.T) {
	now := time.Now().UTC()
	task := model.VideoTask{
		CreatedAt:    videoTaskTime(now.Add(-25 * time.Minute)),
		LastPolledAt: videoTaskTime(now.Add(-2 * time.Minute)),
		Status:       "reconciling",
	}
	if !videoTaskExpired(task, now) {
		t.Fatal("expected task to be in slow reconciliation mode")
	}
	if videoTaskPolledRecently(task, now, time.Minute) {
		t.Fatal("expected reconciliation poll to be due")
	}
}
