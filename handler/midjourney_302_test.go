package handler

import (
	"encoding/json"
	"testing"
)

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
