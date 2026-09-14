package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMediaAIRejectsDisabledAndUnknown(t *testing.T) {
	for _, tc := range []struct {
		body string
		code int
	}{{`{"operation":"asr"}`, 503}, {`{"operation":"speaker_diarization"}`, 503}, {`{"operation":"unknown"}`, 400}, {`{`, 400}} {
		w := httptest.NewRecorder()
		SubmitMediaAI(w, httptest.NewRequest("POST", "/", strings.NewReader(tc.body)))
		if w.Code != tc.code {
			t.Fatalf("status %d expected %d", w.Code, tc.code)
		}
	}
}
