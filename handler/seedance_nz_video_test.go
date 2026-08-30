package handler

import (
	"encoding/json"
	"testing"
)

func TestResolveSeedanceNZVideoModelFromJSON(t *testing.T) {
	tests := []struct {
		name string
		body map[string]any
		want string
	}{
		{name: "text", body: map[string]any{"prompt": "hello"}, want: "seedance-2.0-mini-t2v"},
		{name: "image", body: map[string]any{"images": []string{"https://example.com/a.png"}}, want: "seedance-2.0-mini-i2v"},
		{name: "multi", body: map[string]any{"metadata": map[string]any{"content": []any{map[string]any{"type": "video_url"}}}}, want: "seedance-2.0-mini-multi"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, _ := json.Marshal(test.body)
			if got := resolveSeedanceNZVideoModel("seedance-2.0-mini-t2v", body, "application/json"); got != test.want {
				t.Fatalf("resolveSeedanceNZVideoModel() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestResolveSeedanceNZVideoModelLeavesOtherModelsAlone(t *testing.T) {
	body := []byte(`{"images":["https://example.com/a.png"]}`)
	if got := resolveSeedanceNZVideoModel("kling-v3", body, "application/json"); got != "kling-v3" {
		t.Fatalf("resolveSeedanceNZVideoModel() = %q", got)
	}
}
