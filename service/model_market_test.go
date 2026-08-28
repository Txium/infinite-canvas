package service

import "testing"

func TestProviderOriginURL(t *testing.T) {
	got, err := providerOriginURL("https://api.seedance.nz/v1?debug=1", "/api/usage/wallet/")
	if err != nil {
		t.Fatalf("providerOriginURL returned error: %v", err)
	}
	if want := "https://api.seedance.nz/api/usage/wallet/"; got != want {
		t.Fatalf("providerOriginURL = %q, want %q", got, want)
	}
}

func TestModelIDsFromPayloadAcceptsNestedData(t *testing.T) {
	payload := map[string]any{
		"models": []any{
			map[string]any{"id": "model-a"},
			map[string]any{"id": "model-b"},
		},
	}
	got := modelIDsFromPayload(payload)
	if len(got) != 2 || got[0] != "model-a" || got[1] != "model-b" {
		t.Fatalf("modelIDsFromPayload = %#v", got)
	}
}
