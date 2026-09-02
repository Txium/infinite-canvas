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

func TestLECLatestPricesAndBillingUnits(t *testing.T) {
	_, variants, _, _, _, err := defaultModelCatalog()
	if err != nil {
		t.Fatalf("load default model catalog: %v", err)
	}

	tests := map[string]struct {
		costCents   int64
		priceCents  int64
		billingUnit string
	}{
		"seedance_2__01":               {130, 149, "/次"},
		"lec_seed_2_0_900":             {130, 149, "/次"},
		"lec_md_seedance_2_0_900_720p": {120, 139, "/次"},
		"lec_seed_2_5_900":             {300, 329, "/次"},
		"lec_ac_seedance_2_5_480p":     {45, 51, "/秒"},
		"lec_ac_seedance_2_5_900":      {300, 339, "/次"},
		"lec_ac_seedance_2_5":          {82, 92, "/秒"},
	}

	for _, variant := range variants {
		want, ok := tests[variant.ID]
		if !ok {
			continue
		}
		if variant.CostCents == nil || *variant.CostCents != want.costCents {
			t.Errorf("%s cost = %v, want %d", variant.ID, variant.CostCents, want.costCents)
		}
		if variant.PriceCents == nil || *variant.PriceCents != want.priceCents {
			t.Errorf("%s price = %v, want %d", variant.ID, variant.PriceCents, want.priceCents)
		}
		if variant.BillingUnit != want.billingUnit {
			t.Errorf("%s billing unit = %q, want %q", variant.ID, variant.BillingUnit, want.billingUnit)
		}
		delete(tests, variant.ID)
	}
	for id := range tests {
		t.Errorf("variant %s not found", id)
	}
}
