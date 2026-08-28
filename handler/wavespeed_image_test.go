package handler

import "testing"

func TestReadWaveSpeedTask(t *testing.T) {
	id, outputs, status, message := readWaveSpeedTask([]byte(`{"code":200,"data":{"id":"task-1","status":"completed","outputs":["https://example.com/a.png"]}}`))
	if id != "task-1" || status != "completed" || message != "" || len(outputs) != 1 {
		t.Fatalf("unexpected task parse: %q %#v %q %q", id, outputs, status, message)
	}
}

func TestNormalizeWaveSpeedImageBody(t *testing.T) {
	body, _, err := normalizeWaveSpeedImageBody([]byte(`{"model":"gpt_image_2__01","prompt":"hi","n":1,"size":"1024x1024","quality":"high"}`), "application/json", "gpt_image_2__01")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if text != `{"aspect_ratio":"1:1","prompt":"hi","quality":"low","resolution":"1k"}` {
		t.Fatalf("normalized body = %s", text)
	}
}

func TestWaveSpeedGPTImageTier(t *testing.T) {
	quality, resolution, ok := waveSpeedGPTImageTier("gpt_image_2__09")
	if !ok || quality != "high" || resolution != "4k" {
		t.Fatalf("unexpected tier: %q %q %v", quality, resolution, ok)
	}
}
