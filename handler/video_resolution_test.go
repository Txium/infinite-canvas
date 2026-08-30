package handler

import (
	"bytes"
	"mime/multipart"
	"testing"

	"github.com/tigerowo/infinite-canvas/model"
)

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
