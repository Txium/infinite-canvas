package repository

import (
	"sync"
	"testing"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
)

func TestCreateVideoTaskIfAbsentConcurrentClaim(t *testing.T) {
	useFinanceTestDB(t)
	const attempts = 8
	var wg sync.WaitGroup
	claims := make(chan bool, attempts)
	errs := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, claimed, err := CreateVideoTaskIfAbsent(model.VideoTask{ID: "client_video_task_same", UserID: "user-a", Status: "queued"})
			claims <- claimed
			errs <- err
		}()
	}
	wg.Wait()
	close(claims)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	for claimed := range claims {
		if claimed {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("claims=%d, want exactly 1", count)
	}
}

func TestDistinctVideoTasksAreIndependentlyClaimed(t *testing.T) {
	useFinanceTestDB(t)
	for _, id := range []string{"client_video_task_a", "client_video_task_b"} {
		_, claimed, err := CreateVideoTaskIfAbsent(model.VideoTask{ID: id, UserID: "user-a", ChannelID: "provider_lec", Status: "queued"})
		if err != nil {
			t.Fatal(err)
		}
		if !claimed {
			t.Fatalf("distinct task %s was incorrectly blocked by another LEC task", id)
		}
	}
}

func TestDueVideoTasksKeepRecentReconciliationAndDropExpiredAuditRecords(t *testing.T) {
	useFinanceTestDB(t)
	now := time.Now().UTC()
	tasks := []model.VideoTask{
		{ID: "recent-reconciling", UserID: "user-a", Status: "reconciling", CreatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339Nano)},
		{ID: "expired-reconciling", UserID: "user-a", Status: "reconciling", CreatedAt: now.Add(-72 * time.Hour).Format(time.RFC3339Nano)},
	}
	database, _ := DB()
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	due, err := ListDueVideoTasks(now.Add(-48*time.Hour).Format(time.RFC3339Nano), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].ID != "recent-reconciling" {
		t.Fatalf("unexpected due reconciliation tasks: %+v", due)
	}
}

func TestCanvasMediaTasksConcurrentClaim(t *testing.T) {
	useFinanceTestDB(t)
	tests := []struct {
		name  string
		claim func() (bool, error)
	}{
		{"image", func() (bool, error) {
			_, claimed, err := CreateCanvasImageTaskIfAbsent(model.CanvasImageTask{ID: "client_image_same", UserID: "user-a", Status: "queued"})
			return claimed, err
		}},
		{"audio", func() (bool, error) {
			_, claimed, err := CreateCanvasAudioTaskIfAbsent(model.CanvasAudioTask{ID: "client_audio_same", UserID: "user-a", Status: "queued"})
			return claimed, err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const attempts = 6
			var wg sync.WaitGroup
			claims := make(chan bool, attempts)
			errs := make(chan error, attempts)
			for i := 0; i < attempts; i++ {
				wg.Add(1)
				go func() { defer wg.Done(); claimed, err := test.claim(); claims <- claimed; errs <- err }()
			}
			wg.Wait()
			close(claims)
			close(errs)
			for err := range errs {
				if err != nil {
					t.Fatal(err)
				}
			}
			count := 0
			for claimed := range claims {
				if claimed {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("claims=%d, want exactly 1", count)
			}
		})
	}
}

func TestUserTaskCreditLogsAreOwnerScoped(t *testing.T) {
	useFinanceTestDB(t)
	database, _ := DB()
	logs := []model.CreditLog{
		{ID: "credit_release_a", UserID: "user-a", RelatedID: "same-related", Type: model.CreditLogTypeAIRelease, Amount: 169},
		{ID: "credit_release_b", UserID: "user-b", RelatedID: "same-related", Type: model.CreditLogTypeAIRelease, Amount: 999},
	}
	if err := database.Create(&logs).Error; err != nil {
		t.Fatal(err)
	}
	items, err := ListUserTaskCreditLogs("user-a", []string{"same-related"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].UserID != "user-a" || items[0].Amount != 169 {
		t.Fatalf("cross-user logs leaked: %+v", items)
	}
}
