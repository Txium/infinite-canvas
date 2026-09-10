package repository

import (
	"github.com/tigerowo/infinite-canvas/model"
	"testing"
)

func TestProviderBalanceSnapshotPreservesCurrencyLedgerAndLatestSuccess(t *testing.T) {
	useFinanceTestDB(t)
	db, _ := DB()
	manual := int64(12345)
	provider := model.ModelProvider{ID: "snapshot", Code: "wavespeed", BaseURL: "https://example.com/api/v3", BalanceCents: &manual}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatal(err)
	}
	if err := RecordProviderBalanceSnapshot(provider.ID, provider.BaseURL, "2026-09-10T01:00:00.000000000Z", "2.000000123456789", "USD", ""); err != nil {
		t.Fatal(err)
	}
	if err := RecordProviderBalanceSnapshot(provider.ID, provider.BaseURL, "2026-09-10T01:02:00.000000000Z", "", "", "query failed"); err != nil {
		t.Fatal(err)
	}
	if err := RecordProviderBalanceSnapshot(provider.ID, provider.BaseURL, "2026-09-10T01:01:00.000000000Z", "999", "USD", ""); err != nil {
		t.Fatal(err)
	}
	if err := RecordProviderBalanceSnapshot(provider.ID, "https://wrong-provider.example", "2026-09-10T01:03:00.000000000Z", "999", "USD", ""); err != nil {
		t.Fatal(err)
	}
	saved, err := SavedModelProviderByID(provider.ID)
	if err != nil || saved.UpstreamBalanceAmount != "2.000000123456789" || saved.UpstreamBalanceCurrency != "USD" || saved.UpstreamBalanceError != "query failed" || saved.UpstreamBalanceCheckedAt != "2026-09-10T01:00:00.000000000Z" || *saved.BalanceCents != manual {
		t.Fatalf("snapshot corrupted: %+v %v", saved, err)
	}
	// An old settings form cannot overwrite a new remote snapshot or financial ledger.
	provider.Name = "edited name"
	provider.UpstreamBalanceAmount = "888"
	if err := SaveModelProvider(provider); err != nil {
		t.Fatal(err)
	}
	saved, _ = SavedModelProviderByID(provider.ID)
	if saved.Name != "edited name" || saved.UpstreamBalanceAmount != "2.000000123456789" || *saved.BalanceCents != manual {
		t.Fatalf("config overwrote balance: %+v", saved)
	}
}
