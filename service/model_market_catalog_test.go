package service

import "testing"

func TestSeedanceSpecialVariantUsesAvailableLEC900Route(t *testing.T) {
	_, variants, _, routes, _, err := defaultModelCatalog()
	if err != nil {
		t.Fatalf("load default model catalog: %v", err)
	}

	const variantID = "seedance_2__01"
	const upstreamModelID = "lec-seed-2-0-900"
	foundVariant := false
	for _, variant := range variants {
		if variant.ID != variantID {
			continue
		}
		foundVariant = true
		if variant.UpstreamModelID != upstreamModelID {
			t.Fatalf("variant upstream model = %q, want %q", variant.UpstreamModelID, upstreamModelID)
		}
	}
	if !foundVariant {
		t.Fatalf("variant %q not found", variantID)
	}

	foundRoute := false
	for _, route := range routes {
		if route.VariantID != variantID {
			continue
		}
		foundRoute = true
		if route.UpstreamModelID != upstreamModelID {
			t.Fatalf("route upstream model = %q, want %q", route.UpstreamModelID, upstreamModelID)
		}
	}
	if !foundRoute {
		t.Fatalf("route for %q not found", variantID)
	}
}
