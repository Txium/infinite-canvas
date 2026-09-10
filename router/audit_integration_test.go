package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tigerowo/infinite-canvas/config"
	"github.com/tigerowo/infinite-canvas/handler"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
	"github.com/tigerowo/infinite-canvas/service"
)

func TestAuditHTTPWorkflow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Chdir(t.TempDir())
	config.Cfg = config.Config{StorageDriver: "sqlite", DatabaseDSN: filepath.Join(t.TempDir(), "audit.db"), JWTSecret: "audit-test-only-secret", JWTExpireHours: 1, ManagedPlatformMode: true}
	for _, key := range []string{"MODEL_PROVIDER_302_API_KEY", "MODEL_PROVIDER_LEC_API_KEY", "MODEL_PROVIDER_SEEDANCE_NZ_API_KEY"} {
		t.Setenv(key, "")
	}
	t.Setenv("MODEL_PROVIDER_WAVESPEED_API_KEY", "audit-mock-key")
	app := New()
	request := func(method, endpoint, token string, payload any) (int, map[string]any) {
		t.Helper()
		body, _ := json.Marshal(payload)
		r := httptest.NewRequest(method, endpoint, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		var result map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("%s invalid JSON: %v", endpoint, err)
		}
		return w.Code, result
	}
	register := func(name string) (string, string) {
		t.Helper()
		_, result := request("POST", "/api/auth/register", "", map[string]string{"username": name, "password": "audit-password-123"})
		if result["code"] != float64(0) {
			t.Fatalf("registration failed: %v", result)
		}
		data := result["data"].(map[string]any)
		return data["token"].(string), data["user"].(map[string]any)["id"].(string)
	}
	a, userID := register("audit_a")
	b, _ := register("audit_b")
	database, err := repository.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	t.Run("auth_and_admin_isolation", func(t *testing.T) {
		if status, _ := request("GET", "/api/v1/wallet/credit-logs", "", nil); status != 401 {
			t.Fatalf("anonymous wallet status %d", status)
		}
		if status, _ := request("GET", "/api/admin/finance-summary", a, nil); status != 403 {
			t.Fatalf("user admin status %d", status)
		}
		if status, result := request("GET", "/api/auth/me", a, nil); status != 200 || result["code"] != float64(0) {
			t.Fatal("current user failed")
		}
	})
	t.Run("canvas_cloud_save_and_user_isolation", func(t *testing.T) {
		_, result := request("POST", "/api/v1/canvas/projects", a, map[string]any{"data": map[string]any{"id": "canvas-a", "name": "审计画布", "createdAt": "2026-09-10T00:00:00Z", "updatedAt": "2026-09-10T00:00:00Z", "nodes": []any{}}})
		if result["code"] != float64(0) {
			t.Fatalf("save failed: %v", result)
		}
		_, own := request("GET", "/api/v1/canvas/projects", a, nil)
		_, other := request("GET", "/api/v1/canvas/projects", b, nil)
		if len(own["data"].([]any)) != 1 || len(other["data"].([]any)) != 0 {
			t.Fatal("cloud projects not isolated")
		}
	})
	t.Run("mock_image_generation_billing_and_idempotency", func(t *testing.T) {
		var creates, polls atomic.Int32
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "POST" {
				creates.Add(1)
				fmt.Fprint(w, `{"code":200,"data":{"id":"mock-prediction","status":"processing"}}`)
				return
			}
			if polls.Add(1) == 1 {
				w.WriteHeader(503)
				fmt.Fprint(w, `{"message":"temporary"}`)
				return
			}
			fmt.Fprint(w, `{"code":200,"data":{"id":"mock-prediction","status":"completed","outputs":["https://example.com/result.png"]}}`)
		}))
		defer upstream.Close()
		if _, err := service.ListMarketModels("", false); err != nil {
			t.Fatal(err)
		}
		database, _ := repository.DB()
		database.Model(&model.ModelProvider{}).Where("id = ?", "provider_wavespeed").Updates(map[string]any{"base_url": upstream.URL + "/api/v3", "enabled": true})
		database.Model(&model.ModelRoute{}).Where("variant_id = ?", "flux_2_klein__01").Update("enabled", true)
		database.Model(&model.User{}).Where("id = ?", userID).Update("credits", 500)
		payload := map[string]any{"clientTaskId": "audit-image", "sourceId": "canvas-a", "nodeId": "node-a", "requestBody": map[string]any{"model": "flux_2_klein__01", "prompt": "test", "n": 1}}
		_, created := request("POST", "/api/v1/canvas/image-tasks", a, payload)
		if created["code"] != float64(0) {
			t.Fatalf("create failed: %v", created)
		}
		var task model.CanvasImageTask
		for deadline := time.Now().Add(8 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			task, _, _ = repository.GetUserCanvasImageTask(userID, "audit-image")
			if task.Status == "completed" || task.Status == "failed" {
				break
			}
		}
		if task.Status != "completed" || task.ImageURL == "" || task.UpstreamTaskID != "mock-prediction" {
			t.Fatalf("task did not recover transient poll failure: %+v", task)
		}
		request("POST", "/api/v1/canvas/image-tasks", a, payload)
		if creates.Load() != 1 {
			t.Fatalf("upstream submitted %d times", creates.Load())
		}
		user, _, _ := repository.GetUserByID(userID)
		if user.Credits != 491 || user.FrozenCredits != 0 {
			t.Fatalf("wallet mismatch: available=%d frozen=%d", user.Credits, user.FrozenCredits)
		}
		_, history := request("GET", "/api/v1/generation-tasks", a, nil)
		encoded, _ := json.Marshal(history)
		for _, secret := range []string{"mock-prediction", "audit-mock-key", "provider_wavespeed", "upstreamModel"} {
			if strings.Contains(string(encoded), secret) {
				t.Fatalf("public history leaked %s", secret)
			}
		}
		_, hidden := request("GET", "/api/v1/canvas/image-tasks/audit-image", b, nil)
		if hidden["code"] == float64(0) {
			t.Fatal("other user read task")
		}
	})
	t.Run("accepted_media_recovers_without_original_worker", func(t *testing.T) {
		var posts atomic.Int32
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method != http.MethodGet {
				posts.Add(1)
				w.WriteHeader(500)
				return
			}
			if strings.Contains(r.URL.Path, "failed-result") {
				fmt.Fprint(w, `{"code":200,"data":{"status":"failed","error":"generation rejected"}}`)
				return
			}
			fmt.Fprint(w, `{"code":200,"data":{"status":"completed","outputs":["https://example.com/recovered-output"]}}`)
		}))
		defer upstream.Close()
		database.Model(&model.ModelProvider{}).Where("id = ?", "provider_wavespeed").Update("base_url", upstream.URL+"/api/v3")
		database.Model(&model.ModelRoute{}).Where("variant_id = ?", "speech_26_hd__01").Update("enabled", true)
		old := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
		image := model.CanvasImageTask{ID: "recover-image", UserID: userID, Model: "flux_2_klein__01", ChannelID: "provider_wavespeed", Endpoint: "/images/generations", UpstreamTaskID: "image-result", Status: "reconciling", CreatedAt: old, UpdatedAt: old}
		audio := model.CanvasAudioTask{ID: "recover-audio", UserID: userID, Model: "speech_26_hd__01", ChannelID: "provider_wavespeed", Endpoint: "/audio/speech", UpstreamTaskID: "audio-result", Status: "reconciling", CreatedAt: old, UpdatedAt: old}
		failed := image
		failed.ID, failed.UpstreamTaskID = "recover-failed", "failed-result"
		for _, task := range []model.CanvasImageTask{image, failed} {
			if err := database.Create(&task).Error; err != nil {
				t.Fatal(err)
			}
			if err := service.FreezeUserCredits(userID, task.Model, 9, task.Endpoint, task.ID); err != nil {
				t.Fatal(err)
			}
		}
		if err := database.Create(&audio).Error; err != nil {
			t.Fatal(err)
		}
		if err := service.FreezeUserCredits(userID, audio.Model, 20, audio.Endpoint, audio.ID); err != nil {
			t.Fatal(err)
		}
		handler.StartCanvasMediaReconciler()
		var recoveredImage model.CanvasImageTask
		var recoveredAudio model.CanvasAudioTask
		var rejected model.CanvasImageTask
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			recoveredImage, _, _ = repository.GetUserCanvasImageTask(userID, image.ID)
			recoveredAudio, _, _ = repository.GetUserCanvasAudioTask(userID, audio.ID)
			rejected, _, _ = repository.GetUserCanvasImageTask(userID, failed.ID)
			if recoveredImage.Status == "completed" && recoveredAudio.Status == "completed" && rejected.Status == "failed" {
				break
			}
		}
		if recoveredImage.ImageURL == "" || recoveredAudio.AudioURL == "" || rejected.Status != "failed" {
			t.Fatalf("durable recovery failed: image=%s audio=%s rejected=%s", recoveredImage.Status, recoveredAudio.Status, rejected.Status)
		}
		if posts.Load() != 0 {
			t.Fatal("recovery resubmitted a paid request")
		}
		user, _, _ := repository.GetUserByID(userID)
		if user.Credits != 462 || user.FrozenCredits != 0 {
			t.Fatalf("recovered wallet mismatch: %d frozen=%d", user.Credits, user.FrozenCredits)
		}
	})
	t.Run("provider_balance_survives_query_failure", func(t *testing.T) {
		var invalid atomic.Bool
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/balance") {
				t.Errorf("unexpected balance request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			if invalid.Load() {
				fmt.Fprint(w, `{"code":200,"data":{}}`)
				return
			}
			fmt.Fprint(w, `{"code":200,"data":{"balance":"0.01234567890123456789"}}`)
		}))
		defer upstream.Close()
		database.Model(&model.ModelProvider{}).Where("id = ?", "provider_wavespeed").Update("base_url", upstream.URL+"/api/v3")
		result, err := service.TestModelProviderConnection("provider_wavespeed")
		if err != nil || result.BalanceAmount != "0.01234567890123456789" || result.BalanceCurrency != "USD" || result.BalanceCheckedAt == "" {
			t.Fatalf("balance check failed: %+v %v", result, err)
		}
		invalid.Store(true)
		if _, err := service.TestModelProviderConnection("provider_wavespeed"); err == nil {
			t.Fatal("missing upstream balance treated as zero success")
		}
		saved, err := repository.SavedModelProviderByID("provider_wavespeed")
		if err != nil || saved.UpstreamBalanceAmount != result.BalanceAmount || saved.UpstreamBalanceCheckedAt != result.BalanceCheckedAt || saved.UpstreamBalanceError == "" {
			t.Fatalf("last successful balance lost: %+v %v", saved, err)
		}
	})
}
