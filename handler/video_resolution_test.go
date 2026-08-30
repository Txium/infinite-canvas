package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tigerowo/infinite-canvas/model"
)

// TestVideoReferenceMockProviderChain exercises the paid boundary without
// calling a real provider: the provider must receive the exact public image
// URL, return a real upstream task ID, and later return the completed video.
func TestVideoReferenceMockProviderChain(t *testing.T) {
	const referenceURL = "https://canvas.example.test/api/media/references/image-123.png"
	const upstreamTaskID = "provider-task-123"
	const videoURL = "https://cdn.example.test/video-123.mp4"

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/videos":
			if err := r.ParseMultipartForm(2 << 20); err != nil {
				t.Fatalf("provider could not parse multipart request: %v", err)
			}
			if got := r.FormValue("input_reference[]"); got != referenceURL {
				t.Fatalf("provider reference = %q, want %q", got, referenceURL)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"task_id":"`+upstreamTaskID+`","status":"queued"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/videos/"+upstreamTaskID:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": upstreamTaskID, "status": "completed", "progress": 100, "video_url": videoURL})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	_ = writer.WriteField("model", "seedance_2__01")
	_ = writer.WriteField("input_reference[]", referenceURL)
	_ = writer.Close()
	contentType := writer.FormDataContentType()
	if err := validateRequiredVideoReferences("seedance_2__01", requestBody.Bytes(), contentType); err != nil {
		t.Fatalf("valid reference rejected: %v", err)
	}

	channel := model.ModelChannel{BaseURL: provider.URL}
	createRequest, err := http.NewRequest(http.MethodPost, provider.URL+"/videos", bytes.NewReader(requestBody.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	createRequest.Header.Set("Content-Type", contentType)
	createPayload, createStatus, err := doAIRequest(createRequest, channel)
	if err != nil || createStatus != http.StatusOK {
		t.Fatalf("create request failed: status=%d err=%v payload=%s", createStatus, err, createPayload)
	}
	created := parseVideoTaskPayload(createPayload, "seedance_2__01")
	if created.UpstreamTaskID != upstreamTaskID {
		t.Fatalf("upstream task id = %q, want %q", created.UpstreamTaskID, upstreamTaskID)
	}

	pollRequest, err := http.NewRequest(http.MethodGet, provider.URL+"/videos/"+created.UpstreamTaskID, nil)
	if err != nil {
		t.Fatal(err)
	}
	pollPayload, pollStatus, err := doAIRequest(pollRequest, channel)
	if err != nil || pollStatus != http.StatusOK {
		t.Fatalf("poll request failed: status=%d err=%v payload=%s", pollStatus, err, pollPayload)
	}
	completed := parseVideoTaskPayload(pollPayload, "seedance_2__01")
	if completed.Status != "completed" || completed.VideoURL != videoURL {
		t.Fatalf("completed task = %+v", completed)
	}
}

func TestRequiredLECVideoReferenceRejectedBeforeProvider(t *testing.T) {
	if err := validateRequiredVideoReferences("seedance_2__01", []byte(`{"prompt":"scene"}`), "application/json"); err == nil {
		t.Fatal("missing reference should be rejected")
	}
	if err := validateRequiredVideoReferences("seedance_2__01", []byte(`{"images":["https://example.com/a.png"]}`), "application/json"); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("input_reference[]", "https://example.com/a.png")
	_ = writer.Close()
	if err := validateRequiredVideoReferences("seedance_2__01", body.Bytes(), writer.FormDataContentType()); err != nil {
		t.Fatal(err)
	}
}

func TestExplicitLECImageReferenceRules(t *testing.T) {
	for modelName, rule := range lecVideoReferenceRules {
		if err := validateRequiredVideoReferences(modelName, []byte(`{"prompt":"scene"}`), "application/json"); err == nil {
			t.Fatalf("%s accepted an empty reference", modelName)
		}
		images := make([]string, rule.MaxImages)
		for index := range images {
			images[index] = fmt.Sprintf("https://canvas.example.test/reference-%d.png", index)
		}
		payload, _ := json.Marshal(map[string]any{"images": images})
		if err := validateRequiredVideoReferences(modelName, payload, "application/json"); err != nil {
			t.Fatalf("%s rejected its documented max image count: %v", modelName, err)
		}
		images = append(images, "https://canvas.example.test/too-many.png")
		payload, _ = json.Marshal(map[string]any{"images": images})
		if err := validateRequiredVideoReferences(modelName, payload, "application/json"); err == nil {
			t.Fatalf("%s accepted more than %d images", modelName, rule.MaxImages)
		}
	}
	if err := validateRequiredVideoReferences("lec_seedance_2_0_fast_431_720p", []byte(`{"prompt":"text-to-video remains allowed"}`), "application/json"); err != nil {
		t.Fatalf("optional-reference LEC variant was incorrectly forced to image-to-video: %v", err)
	}
}

func TestValidateFixedMarketVideoResolution(t *testing.T) {
	if err := validateFixedMarketVideoResolution("lec_seedance_2_0_full_933_480p", []byte(`{"resolution":"720p"}`), "application/json"); err == nil {
		t.Fatal("expected 720p request to be rejected for fixed 480p variant")
	}
	if err := validateFixedMarketVideoResolution("lec_seedance_2_0_full_933_480p", []byte(`{"resolution":"480p"}`), "application/json"); err != nil {
		t.Fatalf("fixed 480p request rejected: %v", err)
	}
	if err := validateFixedMarketVideoResolution("lec_seedance_2_0_fast_431_720p", []byte(`{"resolution":"720p"}`), "application/json"); err != nil {
		t.Fatalf("fixed 720p request rejected: %v", err)
	}
}

func TestFixedMarketVideoResolution(t *testing.T) {
	for model, expected := range map[string]string{
		"lec_seedance_2_0_full_933_480p": "480p",
		"lec_seedance_2_0_fast_431_720p": "720p",
		"kling_3__09":                    "1080p",
		"kling_3__01":                    "720p",
	} {
		if actual := fixedMarketVideoResolution(model); actual != expected {
			t.Fatalf("fixedMarketVideoResolution(%q) = %q, want %q", model, actual, expected)
		}
	}
}

func TestIsLECVideoChannel(t *testing.T) {
	for _, channel := range []model.ModelChannel{
		{ID: "provider_lec"},
		{Name: "LEC"},
		{BaseURL: "https://api.paipu.net/v1"},
	} {
		if !isLECVideoChannel(channel) {
			t.Fatalf("expected LEC channel: %#v", channel)
		}
	}
	if isLECVideoChannel(model.ModelChannel{ID: "provider_wavespeed", Name: "WaveSpeed", BaseURL: "https://api.wavespeed.ai/api/v3"}) {
		t.Fatal("did not expect WaveSpeed to be treated as LEC")
	}
}
