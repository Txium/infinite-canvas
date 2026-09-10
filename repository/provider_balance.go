package repository

import "github.com/tigerowo/infinite-canvas/model"

// Only a verified balance updates the value. Failed or out-of-order requests
// cannot erase the last successful reading or alter the RMB ledger.
func RecordProviderBalanceSnapshot(id, baseURL, attemptedAt, amount, currency, detail string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	changes := map[string]any{"upstream_balance_attempted_at": attemptedAt, "upstream_balance_error": detail}
	if detail == "" && amount != "" && currency != "" {
		changes["upstream_balance_amount"] = amount
		changes["upstream_balance_currency"] = currency
		changes["upstream_balance_checked_at"] = attemptedAt
	}
	return db.Model(&model.ModelProvider{}).Where("id = ? AND base_url = ?", id, baseURL).
		Where("upstream_balance_attempted_at IS NULL OR upstream_balance_attempted_at = '' OR upstream_balance_attempted_at <= ?", attemptedAt).
		Updates(changes).Error
}
