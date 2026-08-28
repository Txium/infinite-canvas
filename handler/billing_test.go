package handler

import "testing"

func TestReadAIRequestBillingUnits(t *testing.T) {
	if got := readAIRequestBillingUnits([]byte(`{"duration":10}`), "application/json", "/秒"); got != 10 { t.Fatalf("duration units = %d", got) }
	if got := readAIRequestBillingUnits([]byte(`{"seconds":"5"}`), "application/json", "/秒"); got != 5 { t.Fatalf("seconds units = %d", got) }
	if got := readAIRequestBillingUnits([]byte(`{"duration":-1}`), "application/json", "/秒"); got != 15 { t.Fatalf("automatic duration units = %d", got) }
	if got := readAIRequestBillingUnits([]byte(`{"n":3}`), "application/json", "/张"); got != 3 { t.Fatalf("image units = %d", got) }
}
