package repository

import (
	"github.com/tigerowo/infinite-canvas/model"
	"testing"
)

func TestSettlementPreservesFreshPollResult(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 500, 169)
	database, _ := DB()
	task := model.VideoTask{ID: "fresh", UserID: "user-1", Model: "public-model", Status: "reconciling", Credits: 169, BillingID: "fresh", BillingStatus: "frozen", EstimatedProviderCostCents: 150}
	if err := database.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	task.Status, task.Progress, task.VideoURL = "completed", 100, "https://example.com/result.mp4"
	task.CompletedAt, task.LastPolledAt, task.LastResponse = "2026-09-10T01:00:00Z", "2026-09-10T01:00:00Z", `{"status":"completed"}`
	if err := SettleVideoTaskFinancials(&task, task.CompletedAt); err != nil {
		t.Fatal(err)
	}
	var saved model.VideoTask
	database.First(&saved, "id = ?", task.ID)
	if saved.Status != "completed" || saved.VideoURL != task.VideoURL || saved.VideoURL == "" || saved.CompletedAt == "" || saved.LastResponse == "" || saved.BillingStatus != "settled" {
		t.Fatalf("fresh result lost: %+v", saved)
	}
}

func TestWalletRejectsOppositeTerminalOperation(t *testing.T) {
	for _, first := range []string{"settle", "release"} {
		t.Run(first, func(t *testing.T) {
			useFinanceTestDB(t)
			seedFinanceUser(t, 0, 200)
			settle := model.CreditLog{ID: "settle", UserID: "user-1", RelatedID: "task", Type: model.CreditLogTypeAISettle, FrozenAmount: -100}
			release := model.CreditLog{ID: "release", UserID: "user-1", RelatedID: "task", Type: model.CreditLogTypeAIRelease, Amount: 100, FrozenAmount: -100}
			var err error
			if first == "settle" {
				_, _, err = SettleUserCredits("user-1", 100, settle, "now")
				if err != nil {
					t.Fatal(err)
				}
				_, _, err = ReleaseUserCredits("user-1", 100, release, "now")
			} else {
				_, _, err = ReleaseUserCredits("user-1", 100, release, "now")
				if err != nil {
					t.Fatal(err)
				}
				_, _, err = SettleUserCredits("user-1", 100, settle, "now")
			}
			if err == nil {
				t.Fatal("opposite finalization consumed another task's hold")
			}
			user, _, _ := GetUserByID("user-1")
			if user.FrozenCredits != 100 {
				t.Fatalf("other reservation changed: %d", user.FrozenCredits)
			}
		})
	}
}

func TestWalletRejectsCrossUserBillingReplay(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 500, 0)
	database, _ := DB()
	if err := database.Create(&model.User{ID: "user-2", Username: "other", AffCode: "other-aff", Credits: 500}).Error; err != nil {
		t.Fatal(err)
	}
	entry := model.CreditLog{ID: "shared", UserID: "user-1", RelatedID: "shared", Type: model.CreditLogTypeAIFreeze, Amount: -100, FrozenAmount: 100}
	if _, _, err := FreezeUserCredits("user-1", 100, entry, "now"); err != nil {
		t.Fatal(err)
	}
	entry.UserID = "user-2"
	if _, _, err := FreezeUserCredits("user-2", 100, entry, "now"); err == nil {
		t.Fatal("cross-user billing replay accepted")
	}
}

func TestLateSuccessAfterRefundPreservesOtherFrozenCredits(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 0, 200)
	database, _ := DB()
	task := model.VideoTask{ID: "late", UserID: "user-1", Status: "processing", Credits: 100, BillingID: "late", BillingStatus: "frozen"}
	database.Create(&task)
	entry := model.CreditLog{ID: "credit_release_late", UserID: "user-1", RelatedID: "late", Type: model.CreditLogTypeAIRelease, Amount: 100, FrozenAmount: -100}
	if _, _, err := ReleaseUserCredits("user-1", 100, entry, "now"); err != nil {
		t.Fatal(err)
	}
	task.Status, task.VideoURL = "completed", "https://example.com/late.mp4"
	if err := SettleVideoTaskFinancials(&task, "now"); err != nil {
		t.Fatal(err)
	}
	user, _, _ := GetUserByID("user-1")
	if task.BillingStatus != "released" || task.VideoURL == "" || user.Credits != 100 || user.FrozenCredits != 100 {
		t.Fatalf("late result incorrectly charged: task=%+v user=%+v", task, user)
	}
}

func TestMediaAcceptanceSurvivesStaleWorkerSave(t *testing.T) {
	useFinanceTestDB(t)
	old := model.CanvasImageTask{ID: "recover-image", UserID: "owner", ChannelID: "original", Status: "processing", UpdatedAt: "2026-01-01T00:00:00Z"}
	if _, _, err := CreateCanvasImageTaskIfAbsent(old); err != nil {
		t.Fatal(err)
	}
	if err := RememberCanvasUpstreamTask("owner", old.ID, "prediction-id", "actual-provider", "provider", false); err != nil {
		t.Fatal(err)
	}
	old.Status = "reconciling"
	if _, err := UpdateCanvasImageTask(old); err != nil {
		t.Fatal(err)
	}
	saved, found, err := GetUserCanvasImageTask("owner", old.ID)
	if err != nil || !found || saved.UpstreamTaskID != "prediction-id" || saved.ChannelID != "actual-provider" {
		t.Fatalf("accepted mapping lost: %+v err=%v", saved, err)
	}
	// A stale worker may refresh updated_at after the task entered durable
	// reconciliation. That must not prevent the scheduler from polling the
	// already-paid upstream task.
	saved.UpdatedAt = "2099-01-01T00:00:00Z"
	if _, err := UpdateCanvasImageTask(saved); err != nil {
		t.Fatal(err)
	}
	images, _, err := ListCanvasMediaToReconcile("2026-09-10T00:00:00Z", "2026-09-10T00:00:00Z")
	if err != nil || len(images) != 1 {
		t.Fatalf("restart recovery missing: %d %v", len(images), err)
	}
	other := old
	other.UserID = "someone-else"
	if _, _, err := CreateCanvasImageTaskIfAbsent(other); err == nil {
		t.Fatal("task collision silently reported success")
	}
}

func TestStaleMediaWorkerCannotUndoCompletion(t *testing.T) {
	useFinanceTestDB(t)
	image := model.CanvasImageTask{ID: "terminal-image", UserID: "owner", Status: "completed", ImageURL: "https://example.com/result.png"}
	if _, err := SaveCanvasImageTask(image); err != nil {
		t.Fatal(err)
	}
	image.Status, image.ImageURL = "reconciling", ""
	savedImage, err := UpdateCanvasImageTask(image)
	if err != nil || savedImage.Status != "completed" || savedImage.ImageURL == "" {
		t.Fatalf("completed image regressed: %+v %v", savedImage, err)
	}
	audio := model.CanvasAudioTask{ID: "terminal-audio", UserID: "owner", Status: "completed", AudioURL: "https://example.com/result.mp3"}
	if _, err := SaveCanvasAudioTask(audio); err != nil {
		t.Fatal(err)
	}
	audio.Status, audio.AudioURL = "failed", ""
	savedAudio, err := UpdateCanvasAudioTask(audio)
	if err != nil || savedAudio.Status != "completed" || savedAudio.AudioURL == "" {
		t.Fatalf("completed audio regressed: %+v %v", savedAudio, err)
	}
}

func TestCompletedPaymentRejectsDifferentTradeReplay(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 0, 0)
	order := model.RechargeOrder{ID: "receipt", UserID: "user-1", Credits: 100, AmountCents: 100, Status: model.RechargeOrderPending}
	if _, err := SaveRechargeOrder(order); err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteRechargeOrder(order.ID, "original-trade", "alipay", "seller", "now"); err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteRechargeOrder(order.ID, "other-trade", "alipay", "seller", "now"); err == nil {
		t.Fatal("mismatched payment accepted")
	}
}

func TestStaleCompletionCannotReplaceStoredMediaWithTemporaryURL(t *testing.T) {
	useFinanceTestDB(t)
	image := model.CanvasImageTask{ID: "stored-image", UserID: "owner", Status: "completed", ImageURL: "/api/files/image/content", StorageKey: "server:image"}
	if _, err := SaveCanvasImageTask(image); err != nil {
		t.Fatal(err)
	}
	image.ImageURL, image.StorageKey = "https://example.com/temporary.png", ""
	savedImage, err := UpdateCanvasImageTask(image)
	if err != nil || savedImage.StorageKey != "server:image" || savedImage.ImageURL != "/api/files/image/content" {
		t.Fatalf("stored image overwritten: %+v %v", savedImage, err)
	}
	audio := model.CanvasAudioTask{ID: "stored-audio", UserID: "owner", Status: "completed", AudioURL: "/api/files/audio/content", StorageKey: "server:audio"}
	if _, err := SaveCanvasAudioTask(audio); err != nil {
		t.Fatal(err)
	}
	audio.AudioURL, audio.StorageKey = "data:audio/mpeg;base64,SUQz", ""
	savedAudio, err := UpdateCanvasAudioTask(audio)
	if err != nil || savedAudio.StorageKey != "server:audio" || savedAudio.AudioURL != "/api/files/audio/content" {
		t.Fatalf("stored audio overwritten: %+v %v", savedAudio, err)
	}
}

func TestEstimatedCostIsNotReportedAsVerified(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 0, 100)
	database, _ := DB()
	task := model.VideoTask{ID: "estimated", UserID: "user-1", Status: "completed", BillingID: "estimated", BillingStatus: "frozen", Credits: 100, EstimatedProviderCostCents: 50}
	if err := database.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := SettleVideoTaskFinancials(&task, "2026-09-10T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	summary, err := AdminFinanceSummary("2026-09-10T00:00:00Z", "all", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if summary.UpstreamCostReady {
		t.Fatal("estimated costs reported as verified")
	}
}
