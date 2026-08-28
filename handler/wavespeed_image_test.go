package handler

import "testing"

func TestReadWaveSpeedTask(t *testing.T) {
	id, outputs, status, message := readWaveSpeedTask([]byte(`{"code":200,"data":{"id":"task-1","status":"completed","outputs":["https://example.com/a.png"]}}`))
	if id != "task-1" || status != "completed" || message != "" || len(outputs) != 1 {
		t.Fatalf("unexpected task parse: %q %#v %q %q", id, outputs, status, message)
	}
}

func TestNormalizeWaveSpeedImageBody(t *testing.T) {
	body, _, err := normalizeWaveSpeedImageBody([]byte(`{"model":"openai/gpt-image-2/text-to-image","prompt":"hi","n":1}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if text != `{"prompt":"hi"}` {
		t.Fatalf("normalized body = %s", text)
	}
}
