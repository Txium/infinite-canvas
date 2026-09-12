package handler

import (
	"github.com/tigerowo/infinite-canvas/model"
	"testing"
)

func TestVideoRetryRequiresConfirmedFailureAndReleasedFunds(t *testing.T) {
	for _, task := range []model.VideoTask{
		{Status: "queued", BillingStatus: "frozen", Credits: 169},
		{Status: "processing", BillingStatus: "frozen", Credits: 169},
		{Status: "failed", BillingStatus: "frozen", Credits: 169},
		{Status: "completed", BillingStatus: "settled", Credits: 169},
	} {
		if canRetryFailedVideoTask(task) {
			t.Fatalf("unsafe retry accepted: %#v", task)
		}
	}
	if !canRetryFailedVideoTask(model.VideoTask{Status: "failed", BillingStatus: "released", Credits: 169}) {
		t.Fatal("released failure cannot retry")
	}
}

func TestVideoRetryIDIsStableAndDistinct(t *testing.T) {
	one := videoRetryTaskID("user", "old")
	if one == "old" || one != videoRetryTaskID("user", "old") {
		t.Fatal("retry is not idempotent")
	}
	if one == videoRetryTaskID("other", "old") || one == videoRetryTaskID("user", one) {
		t.Fatal("retry identities overlap")
	}
}
