package repository

import (
	"testing"

	"github.com/tigerowo/infinite-canvas/model"
)

func TestCatalogV16UpdatesExistingSeedanceSpecialRoute(t *testing.T) {
	useFinanceTestDB(t)
	database, _ := DB()
	old := model.ModelRoute{ID: "route_seedance_2__01", ModelID: "seedance_2", VariantID: "seedance_2__01", ProviderID: "provider_lec", UpstreamModelID: "lec-ac-seedance-900-720p", Protocol: "custom", Priority: 1, Enabled: true}
	if err := database.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.ModelCatalogVersion{Key: "default", Version: 15}).Error; err != nil {
		t.Fatal(err)
	}

	cost, price := int64(150), int64(169)
	err := SyncDefaultModelCatalog(
		16,
		[]model.MarketModel{{ID: "seedance_2", Name: "Seedance 2.0", Category: "video", Enabled: true}},
		[]model.ModelVariant{{ID: "seedance_2__01", ModelID: "seedance_2", Name: "人物特惠 720P / 900", ProviderCode: "lec", UpstreamModelID: "lec-seed-2-0-900", CostCents: &cost, PriceCents: &price, Enabled: true}},
		[]model.ModelProvider{{ID: "provider_lec", Code: "lec", Name: "LEC", BaseURL: "https://api.paipu.net", Enabled: true}},
		[]model.ModelRoute{{ID: "route_seedance_2__01", ModelID: "seedance_2", VariantID: "seedance_2__01", ProviderID: "provider_lec", UpstreamModelID: "lec-seed-2-0-900", Protocol: "custom", Priority: 1, Enabled: true}},
		"2026-08-30T00:00:00Z",
	)
	if err != nil {
		t.Fatal(err)
	}

	var saved model.ModelRoute
	if err := database.First(&saved, "id = ?", old.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.UpstreamModelID != "lec-seed-2-0-900" {
		t.Fatalf("upstream model = %q", saved.UpstreamModelID)
	}
}
