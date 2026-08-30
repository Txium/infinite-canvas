package service

import (
	"reflect"
	"testing"
)

func TestCatalogModelIDsSupportsProviderCatalogs(t *testing.T) {
	payload := map[string]any{"data": []any{
		map[string]any{"id": "kwaivgi/kling-v3.0-std"},
		map[string]any{"id": "lec-model"},
		map[string]any{"model_id": "wavespeed/model/variant"},
	}}
	want := []string{"kwaivgi/kling-v3.0-std", "lec-model", "wavespeed/model/variant"}
	if got := catalogModelIDs(payload); !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog ids = %#v, want %#v", got, want)
	}
}

func TestCatalogModelIDsRejectsEmptyPayload(t *testing.T) {
	if got := catalogModelIDs(map[string]any{"data": []any{}}); len(got) != 0 {
		t.Fatalf("expected empty catalog, got %#v", got)
	}
}
