package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
)

func Test302MidjourneyKeepsGatewayAndModelAuthentication(t *testing.T) {
	for _, endpoint := range []string{"/mj/submit/imagine", "/mj-turbo/submit/imagine"} {
		request, _ := http.NewRequest(http.MethodPost, "https://api.302.ai"+endpoint, nil)
		request.Header.Set("Authorization", "Bearer stale-test-value")
		set302MidjourneyAuthHeader(request, model.ModelChannel{APIKey: "test-secret"}, endpoint)
		if request.Header.Get("Authorization") != "Bearer test-secret" || request.Header.Get("mj-api-secret") != "test-secret" {
			t.Fatal("gateway and Midjourney authentication must use the resolved channel key")
		}
	}
}

func TestNormalize302MidjourneyRequest(t *testing.T) {
	body, contentType, err := normalize302MidjourneyRequest([]byte(`{"model":"midjourney__01","prompt":"stormy coast","size":"1280x720"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "application/json" {
		t.Fatalf("unexpected content type %s", contentType)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["prompt"] != "stormy coast --ar 16:9" {
		t.Fatalf("unexpected prompt %#v", payload["prompt"])
	}
	if payload["botType"] != "MID_JOURNEY" {
		t.Fatalf("unexpected bot type %#v", payload["botType"])
	}
}

func TestNormalize302MidjourneyRejectsMultipart(t *testing.T) {
	if _, _, err := normalize302MidjourneyRequest(nil, "multipart/form-data; boundary=x"); err == nil {
		t.Fatal("expected reference image request to be rejected")
	}
}

func Test302MidjourneyResolvesSubmitPath(t *testing.T) {
	channel := model.ModelChannel{ID: "provider_302", BaseURL: "https://api.302.ai"}
	for _, endpoint := range []string{"/mj/submit/imagine", "/mj-turbo/submit/imagine"} {
		resolvedPath := resolveAIProxyPath(channel, endpoint, "/images/generations")
		if resolvedPath != endpoint {
			t.Fatalf("expected %s, got %s", endpoint, resolvedPath)
		}
		resolvedURL := resolveAIProxyURL(channel, endpoint, resolvedPath)
		if resolvedURL != channel.BaseURL+endpoint {
			t.Fatalf("unexpected Midjourney URL %s", resolvedURL)
		}
	}
}

func TestCanvasMediaReconciliationKeepsPersistedMidjourneyUpstreamModel(t *testing.T) {
	task := model.VideoTask{Model: "midjourney__01", UpstreamModel: "/mj/submit/imagine"}
	if upstream := videoTaskUpstreamModel(task); upstream != "/mj/submit/imagine" || !is302MidjourneyModel(upstream) {
		t.Fatalf("persisted Midjourney route was lost: %q", upstream)
	}
}

func TestUnique302MidjourneyURLs(t *testing.T) {
	urls := unique302MidjourneyURLs(midjourney302TaskResponse{ImageURL: "https://example.com/grid.png", ImageURLs: []string{"https://example.com/1.png", "https://example.com/1.png"}})
	if len(urls) != 2 {
		t.Fatalf("expected two unique urls, got %#v", urls)
	}
}

func Test302MidjourneyPersistsAcceptedTaskBeforePolling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"code":1,"result":"mj-task-persisted"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"mj-task-persisted","status":"SUCCESS","imageUrl":"https://example.com/result.png"}`))
	}))
	defer server.Close()

	taskID := "canvas_image_task_mj_persistence"
	persistedID := ""
	previousRemember := rememberCanvasUpstreamTask
	rememberCanvasUpstreamTask = func(userID, localTaskID, upstreamID, channelID, channelName string, audio bool) error {
		persistedID = upstreamID
		return nil
	}
	defer func() { rememberCanvasUpstreamTask = previousRemember }()
	request, _ := http.NewRequestWithContext(context.WithValue(context.Background(), canvasBillingContextKey{}, taskID), http.MethodPost, server.URL+"/mj/submit/imagine", nil)
	recorder := httptest.NewRecorder()
	previousInterval := midjourney302PollInterval
	midjourney302PollInterval = time.Millisecond
	defer func() { midjourney302PollInterval = previousInterval }()
	channel := model.ModelChannel{ID: "provider_302", Name: "302.AI", BaseURL: server.URL, APIKey: "test", Enabled: true}
	copy302MidjourneyImageResponse(recorder, request, channel, aiLogContext{UserID: "mj-owner", Model: "midjourney__01", Channel: channel, Endpoint: "/images/generations"}, nil, nil)
	if persistedID != "mj-task-persisted" {
		t.Fatalf("accepted MJ task was not persisted before polling: %q", persistedID)
	}
}
