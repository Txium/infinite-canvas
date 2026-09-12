package handler

import (
	"encoding/json"
	"net/http"
	"testing"

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

func TestUnique302MidjourneyURLs(t *testing.T) {
	urls := unique302MidjourneyURLs(midjourney302TaskResponse{ImageURL: "https://example.com/grid.png", ImageURLs: []string{"https://example.com/1.png", "https://example.com/1.png"}})
	if len(urls) != 2 {
		t.Fatalf("expected two unique urls, got %#v", urls)
	}
}
