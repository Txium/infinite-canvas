package handler

import "testing"

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
