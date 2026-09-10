package service

import (
	"encoding/json"
	"testing"
)

func TestProviderBalanceKeepsOriginalPrecisionAndRejectsMissing(t *testing.T) {
	for _, raw := range []string{`0`, `"0.000000000000123456789"`, `123456789.987654321`, `"-0.01"`} {
		got, err := providerBalanceAmount(json.RawMessage(raw))
		if err != nil || got == "" {
			t.Fatalf("valid balance %s: %q %v", raw, got, err)
		}
		if raw == `"0.000000000000123456789"` && got != "0.000000000000123456789" {
			t.Fatal("balance precision lost")
		}
	}
	for _, raw := range []string{"", `null`, `{}`, `true`, `"NaN"`, `"Infinity"`, `1e10000`, `"1 USD"`, `""`} {
		if got, err := providerBalanceAmount(json.RawMessage(raw)); err == nil {
			t.Fatalf("invalid balance %s became %q", raw, got)
		}
	}
}
