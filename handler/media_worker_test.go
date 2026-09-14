package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMediaWorkerUnconfiguredDoesNotCallProvider(t *testing.T) {
	t.Setenv("MEDIA_WORKER_URL", "")
	t.Setenv("MEDIA_WORKER_TOKEN", "")
	response := httptest.NewRecorder()
	ProcessMedia(response, httptest.NewRequest("POST", "/", strings.NewReader("media")))
	if response.Code != 503 {
		t.Fatalf("got %d", response.Code)
	}
}

func TestMediaWorkerAcceptsChunkedUploadWithoutForwardingUserToken(t *testing.T) {
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "original-media" || r.ContentLength != 14 || r.Header.Get("Authorization") != "Bearer worker-test" {
			t.Error("incorrect worker request")
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("processed-output"))
	}))
	defer worker.Close()
	t.Setenv("MEDIA_WORKER_URL", worker.URL)
	t.Setenv("MEDIA_WORKER_TOKEN", "worker-test")
	t.Setenv("MEDIA_WORKER_MAX_MB", "1")
	request := httptest.NewRequest("POST", "/", strings.NewReader("original-media"))
	request.ContentLength = -1
	request.Header.Set("Authorization", "Bearer user-token")
	response := httptest.NewRecorder()
	ProcessMedia(response, request)
	if response.Code != 200 || response.Body.String() != "processed-output" {
		t.Fatalf("bad response %d %s", response.Code, response.Body.String())
	}
}

func TestMediaWorkerLimitComesFromEnvironment(t *testing.T) {
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ready":true}`))
	}))
	defer worker.Close()
	t.Setenv("MEDIA_WORKER_URL", worker.URL)
	t.Setenv("MEDIA_WORKER_TOKEN", "worker-test")
	t.Setenv("MEDIA_WORKER_MAX_MB", "500")
	status := httptest.NewRecorder()
	MediaWorkerStatus(status, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(status.Body.String(), `"maxBytes":524288000`) {
		t.Fatalf("unexpected status %s", status.Body.String())
	}
	if !strings.Contains(status.Body.String(), `"ready":true`) {
		t.Fatalf("worker was not reported ready: %s", status.Body.String())
	}

	t.Setenv("MEDIA_WORKER_MAX_MB", "invalid")
	if mediaWorkerMaxBytes() != 25<<20 {
		t.Fatal("invalid limit did not use safe default")
	}
}
