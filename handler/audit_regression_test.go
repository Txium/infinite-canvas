package handler

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func TestBillingHeadersCannotImpersonateInternalTask(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/images/generations", nil)
	r.Header.Set("X-Billing-Task-ID", "victim-task")
	r.Header.Set(deferredBillingReleaseHeader, "1")
	if internalBillingID(r) != "" {
		t.Fatal("untrusted billing header accepted")
	}
	r = r.WithContext(context.WithValue(r.Context(), canvasBillingContextKey{}, "real-task"))
	if internalBillingID(r) != "real-task" {
		t.Fatal("server billing context missing")
	}
}

func TestMultipartRouteReplacesModelAndPreservesReference(t *testing.T) {
	var raw bytes.Buffer
	w := multipart.NewWriter(&raw)
	w.WriteField("model", "public-variant")
	w.WriteField("prompt", "reference test")
	f, _ := w.CreateFormFile("image", "reference.png")
	f.Write([]byte("original-image-bytes"))
	w.Close()
	body, err := replaceAIRequestModel(raw.Bytes(), w.FormDataContentType(), "real-provider-model")
	if err != nil {
		t.Fatal(err)
	}
	_, params, _ := mime.ParseMediaType(w.FormDataContentType())
	form, err := multipart.NewReader(bytes.NewReader(body), params["boundary"]).ReadForm(1024)
	if err != nil {
		t.Fatal(err)
	}
	defer form.RemoveAll()
	if len(form.Value["model"]) != 1 || form.Value["model"][0] != "real-provider-model" {
		t.Fatalf("model not routed: %v", form.Value)
	}
	file, err := form.File["image"][0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	content, _ := io.ReadAll(file)
	if string(content) != "original-image-bytes" {
		t.Fatal("reference image changed")
	}
}

func TestBillingRejectsOverflowAndInvalidCounts(t *testing.T) {
	for _, body := range []string{`{"n":1e100}`, `{"n":1.5}`, `{"n":999999}`, `{"n":0}`} {
		if _, err := checkedBillingTotal(100, readAIRequestCount([]byte(body), "application/json")); err == nil {
			t.Fatalf("invalid count allowed: %s", body)
		}
	}
	if _, err := checkedBillingTotal(int(^uint(0)>>1), 2); err == nil {
		t.Fatal("billing overflow allowed")
	}
	if got := readAIRequestBillingUnits([]byte(`{"seconds":5.5}`), "application/json", "/秒"); got != 6 {
		t.Fatalf("fractional seconds undercharged: %d", got)
	}
	if _, err := checkedBillingTotal(109, readAIRequestBillingUnits([]byte(`{"seconds":1e100}`), "application/json", "/秒")); err == nil {
		t.Fatal("unbounded duration accepted")
	}
}

func TestCanvasEndpointIsScopedToMediaType(t *testing.T) {
	for _, endpoint := range []string{"/videos", "/admin", "../billing", "https://example.com"} {
		if validCanvasTaskEndpoint(endpoint, "/images/generations") {
			t.Fatalf("arbitrary endpoint allowed: %s", endpoint)
		}
	}
	if validCanvasTaskEndpoint("/images/generations", "/audio/speech") {
		t.Fatal("image endpoint allowed for audio")
	}
}

func TestMultipartBillingValidatesWholeNumericField(t *testing.T) {
	for _, value := range []string{"1.5", "1 garbage", "0", "NaN", "1e100"} {
		var raw bytes.Buffer
		w := multipart.NewWriter(&raw)
		w.WriteField("n", value)
		w.Close()
		if readAIRequestCount(raw.Bytes(), w.FormDataContentType()) != 0 {
			t.Fatalf("invalid count accepted: %s", value)
		}
	}
	var raw bytes.Buffer
	w := multipart.NewWriter(&raw)
	w.WriteField("seconds", "5.5")
	w.Close()
	if got := readAIRequestBillingUnits(raw.Bytes(), w.FormDataContentType(), "/秒"); got != 6 {
		t.Fatalf("duration truncated: %d", got)
	}
}

func TestAudioRejectsEmptyJSONAndHTMLResponses(t *testing.T) {
	for _, body := range []string{"", `{"error":"service unavailable"}`, "<html>502 Bad Gateway</html>"} {
		if validCanvasAudioPayload("audio/mpeg", []byte(body)) {
			t.Fatalf("invalid media accepted: %s", body)
		}
	}
	if !validCanvasAudioPayload("audio/mpeg", []byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0, 0}) {
		t.Fatal("MP3 rejected")
	}
}
