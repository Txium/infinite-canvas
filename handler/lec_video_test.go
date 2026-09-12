package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"testing"

	"github.com/tigerowo/infinite-canvas/model"
)

func TestLEC900MultipartURLsBecomeJSON(t *testing.T) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for key, value := range map[string]string{"model": "seedance_2__01", "prompt": "sea breeze", "seconds": "15", "size": "1280x720"} {
		_ = w.WriteField(key, value)
	}
	_ = w.WriteField("input_reference[]", "https://canvas.example/a.jpg")
	_ = w.WriteField("input_reference[]", "https://canvas.example/b.jpg")
	_ = w.Close()
	result, contentType, err := normalizeVideoCreateBody(body.Bytes(), w.FormDataContentType(), "lec-seed-2-0-900", model.ModelChannel{ID: "provider_lec"}, "/videos")
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if contentType != "application/json" || json.Unmarshal(result, &payload) != nil {
		t.Fatalf("invalid JSON response %s", result)
	}
	if payload["model"] != "lec-seed-2-0-900" || payload["aspect_ratio"] != "16:9" || len(payload["images"].([]any)) != 2 {
		t.Fatalf("wrong mapping %s", result)
	}
	if len(payload) != 4 {
		t.Fatalf("unexpected multipart-only fields %s", result)
	}
}

func TestLEC900RejectsInvalidReferencesAndSettings(t *testing.T) {
	for _, body := range []string{
		`{"prompt":"test","images":["blob:local"]}`,
		`{"prompt":"test","images":["http://example.com/a.jpg"]}`,
		`{"prompt":"test","size":"1024x1024"}`,
		`{"prompt":"test","seconds":10}`,
		`{"prompt":"test","last_frame_url":"https://example.com/end.jpg"}`,
		`{"prompt":"test","video_reference[]":"https://example.com/a.mp4"}`,
		`not json`,
	} {
		if _, _, err := normalizeLEC900VideoBody([]byte(body), "application/json"); err == nil {
			t.Fatalf("accepted invalid request %s", body)
		}
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("input_reference[]", "local.png")
	_, _ = part.Write([]byte("image"))
	_ = w.Close()
	if _, _, err := normalizeLEC900VideoBody(body.Bytes(), w.FormDataContentType()); err == nil {
		t.Fatal("accepted raw multipart file")
	}
}

func TestLEC900JSONAndOtherProviders(t *testing.T) {
	body := []byte(`{"prompt":"test","aspect_ratio":"9:16","images":["https://example.com/a.jpg"]}`)
	result, _, err := normalizeLEC900VideoBody(body, "application/json")
	if err != nil || !bytes.Contains(result, []byte(`"aspect_ratio":"9:16"`)) {
		t.Fatalf("JSON input failed %s %v", result, err)
	}
	result, contentType, err := normalizeVideoCreateBody(body, "application/json", "other-model", model.ModelChannel{ID: "other"}, "/videos")
	if err != nil || contentType != "application/json" || !bytes.Equal(body, result) {
		t.Fatal("unrelated provider changed")
	}
}
