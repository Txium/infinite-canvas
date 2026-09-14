package service

import (
	"context"
	"errors"
	"github.com/tigerowo/infinite-canvas/model"
	"math"
	"testing"
)

func TestMediaAIReservedCapabilitiesNeverSubmit(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range MediaAICapabilities() {
		if seen[c.Operation] || c.Status != "COMING_SOON" {
			t.Fatal("invalid capability", c)
		}
		seen[c.Operation] = true
		a, err := ResolveMediaAIAdapter(c.Operation)
		if err != nil {
			t.Fatal(err)
		}
		id, err := a.Submit(context.Background(), model.MediaAIRequest{Operation: c.Operation})
		if id != "" || !errors.Is(err, ErrMediaAIDisabled) {
			t.Fatal("disabled adapter accepted request")
		}
	}
	if _, err := ResolveMediaAIAdapter("unknown"); err == nil {
		t.Fatal("unknown operation accepted")
	}
}

func TestVoiceProfileValidation(t *testing.T) {
	p := model.VoiceProfile{CharacterID: " character "}
	if err := ValidateVoiceProfile(&p); err != nil || p.DefaultSpeed != 1 || p.CharacterID != "character" {
		t.Fatal("invalid defaults")
	}
	for _, speed := range []float64{-.1, 5, math.NaN(), math.Inf(1)} {
		p.DefaultSpeed = speed
		if ValidateVoiceProfile(&p) == nil {
			t.Fatal("invalid speed accepted")
		}
	}
}
