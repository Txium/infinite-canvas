package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tigerowo/infinite-canvas/config"
	"github.com/tigerowo/infinite-canvas/model"
)

func TestImageReferenceIsPersistedAndProviderReadable(t *testing.T) {
	previous := config.Cfg
	t.Cleanup(func() { config.Cfg = previous })
	config.Cfg.StorageDriver = "sqlite"
	config.Cfg.DatabaseDSN = filepath.Join(t.TempDir(), "chain.db")

	var app *httptest.Server
	app = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/media/references":
			UploadReferenceMedia(w, r)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/media/references/"):
			ReferenceMedia(w, r, strings.TrimPrefix(r.URL.Path, "/api/media/references/"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer app.Close()
	config.Cfg.PublicBaseURL = app.URL

	// The bytes are deliberately not recompressed or transformed by the
	// reference store; the Provider receives the same persisted object.
	imageBytes := []byte("mock-generated-png-bytes")
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "generated.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(imageBytes)
	_ = writer.Close()
	request, _ := http.NewRequest(http.MethodPost, app.URL+"/api/v1/media/references", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var envelope struct {
		Code int                        `json:"code"`
		Data referenceMediaUploadResult `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil || envelope.Code != 0 {
		t.Fatalf("reference upload failed: code=%d err=%v", envelope.Code, err)
	}
	if !strings.HasPrefix(envelope.Data.URL, app.URL+"/api/media/references/") {
		t.Fatalf("unexpected persistent reference URL: %s", envelope.Data.URL)
	}

	stored, err := http.Get(envelope.Data.URL)
	if err != nil {
		t.Fatal(err)
	}
	storedBytes, _ := io.ReadAll(stored.Body)
	_ = stored.Body.Close()
	if stored.StatusCode != http.StatusOK || !bytes.Equal(storedBytes, imageBytes) {
		t.Fatalf("stored image unavailable or changed: status=%d bytes=%q", stored.StatusCode, storedBytes)
	}

	const upstreamTaskID = "mock-provider-video-task"
	const finalVideoURL = "https://cdn.example.test/generated-video.mp4"
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/videos":
			if err := r.ParseMultipartForm(2 << 20); err != nil {
				t.Fatal(err)
			}
			if received := r.FormValue("input_reference[]"); received != envelope.Data.URL {
				t.Fatalf("provider received reference %q, want %q", received, envelope.Data.URL)
			}
			_, _ = io.WriteString(w, `{"task_id":"`+upstreamTaskID+`","status":"queued"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/videos/"+upstreamTaskID:
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": upstreamTaskID, "status": "completed", "video_url": finalVideoURL})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()

	var videoBody bytes.Buffer
	videoWriter := multipart.NewWriter(&videoBody)
	_ = videoWriter.WriteField("model", "seedance_2__01")
	_ = videoWriter.WriteField("input_reference[]", envelope.Data.URL)
	_ = videoWriter.Close()
	createRequest, _ := http.NewRequest(http.MethodPost, provider.URL+"/videos", bytes.NewReader(videoBody.Bytes()))
	createRequest.Header.Set("Content-Type", videoWriter.FormDataContentType())
	createPayload, createStatus, err := doAIRequest(createRequest, model.ModelChannel{BaseURL: provider.URL})
	if err != nil || createStatus != http.StatusOK {
		t.Fatalf("video create failed: status=%d err=%v payload=%s", createStatus, err, createPayload)
	}
	created := parseVideoTaskPayload(createPayload, "seedance_2__01")
	if created.UpstreamTaskID != upstreamTaskID {
		t.Fatalf("real upstream task id was not mapped: %+v", created)
	}
	pollRequest, _ := http.NewRequest(http.MethodGet, provider.URL+"/videos/"+created.UpstreamTaskID, nil)
	pollPayload, pollStatus, err := doAIRequest(pollRequest, model.ModelChannel{BaseURL: provider.URL})
	completed := parseVideoTaskPayload(pollPayload, "seedance_2__01")
	if err != nil || pollStatus != http.StatusOK || completed.Status != "completed" || completed.VideoURL != finalVideoURL {
		t.Fatalf("video poll failed: status=%d err=%v task=%+v", pollStatus, err, completed)
	}

	missing, err := http.Get(app.URL + "/api/media/references/missing.png")
	if err != nil {
		t.Fatal(err)
	}
	_ = missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("expired/missing reference status=%d, want 404", missing.StatusCode)
	}
}
