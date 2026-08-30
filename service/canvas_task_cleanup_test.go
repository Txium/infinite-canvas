package service

import "testing"

func TestStaleCanvasTaskDisposition(t *testing.T) {
	if staleCanvasTaskNeedsReconciliation("queued", "") {
		t.Fatal("unsubmitted queued task should be refundable")
	}
	for _, test := range []struct{ status, started string }{{"processing", ""}, {"running", ""}, {"queued", "2026-08-30T00:00:00Z"}} {
		if !staleCanvasTaskNeedsReconciliation(test.status, test.started) {
			t.Fatalf("submitted task should reconcile: %+v", test)
		}
	}
}
