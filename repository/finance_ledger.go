package repository

import (
	"strings"

	"github.com/google/uuid"
	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
)

func SaveProviderLedger(entry model.ProviderLedger) (model.ProviderLedger, error) {
	db, err := DB()
	if err != nil {
		return entry, err
	}
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = "provider_ledger_" + uuid.NewString()
	}
	return entry, db.Create(&entry).Error
}

func SaveOperatingExpense(entry model.OperatingExpense) (model.OperatingExpense, error) {
	db, err := DB()
	if err != nil {
		return entry, err
	}
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = "operating_expense_" + uuid.NewString()
	}
	return entry, db.Create(&entry).Error
}

func ListProviderLedgers(limit int) ([]model.ProviderLedger, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var items []model.ProviderLedger
	err = db.Order("created_at desc").Limit(limit).Find(&items).Error
	return items, err
}

func ListOperatingExpenses(limit int) ([]model.OperatingExpense, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var items []model.OperatingExpense
	err = db.Order("expense_date desc, created_at desc").Limit(limit).Find(&items).Error
	return items, err
}

// RecordProviderTopup atomically updates a manually tracked CNY balance and
// appends the separate upstream-funds ledger entry.
func RecordProviderTopup(providerID string, amountCNY int64, operatorID, reason, reference, current string) (model.ProviderLedger, error) {
	if amountCNY <= 0 || strings.TrimSpace(reason) == "" {
		return model.ProviderLedger{}, gorm.ErrInvalidValue
	}
	db, err := DB()
	if err != nil {
		return model.ProviderLedger{}, err
	}
	var result model.ProviderLedger
	err = db.Transaction(func(tx *gorm.DB) error {
		var provider model.ModelProvider
		if err := tx.Where("id = ?", providerID).First(&provider).Error; err != nil {
			return err
		}
		before := int64(0)
		if provider.BalanceCents != nil {
			before = *provider.BalanceCents
		}
		after := before + amountCNY
		if err := tx.Model(&provider).Updates(map[string]any{"balance_cents": after, "balance_checked_at": current, "updated_at": current}).Error; err != nil {
			return err
		}
		result = model.ProviderLedger{
			ID: "provider_ledger_" + uuid.NewString(), ProviderID: providerID, Type: model.ProviderLedgerTopup,
			Amount: amountCNY, Currency: "CNY", AmountCNY: amountCNY, BalanceBefore: before, BalanceAfter: after,
			OperatorID: operatorID, Reason: strings.TrimSpace(reason), Reference: strings.TrimSpace(reference),
			IdempotencyKey: "provider_topup:" + uuid.NewString(), CreatedAt: current,
		}
		return tx.Create(&result).Error
	})
	return result, err
}

// SettleVideoTaskFinancials atomically settles the frozen customer price,
// records provider cost once, and persists the realized gross profit.
func SettleVideoTaskFinancials(task *model.VideoTask, current string) error {
	if task == nil {
		return gorm.ErrInvalidValue
	}
	db, err := DB()
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var saved model.VideoTask
		if err := tx.Where("id = ?", task.ID).First(&saved).Error; err != nil {
			return err
		}
		if saved.BillingStatus == "settled" {
			*task = saved
			return nil
		}
		if saved.BillingStatus != "frozen" {
			return gorm.ErrInvalidValue
		}

		updated := tx.Model(&model.User{}).
			Where("id = ? AND frozen_credits >= ?", saved.UserID, saved.Credits).
			Updates(map[string]any{"frozen_credits": gorm.Expr("frozen_credits - ?", saved.Credits), "updated_at": current})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return gorm.ErrInvalidValue
		}
		var user model.User
		if err := tx.Where("id = ?", saved.UserID).First(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.CreditLog{
			ID: "credit_settle_" + saved.BillingID, UserID: saved.UserID,
			Type: model.CreditLogTypeAISettle, FrozenAmount: -saved.Credits,
			Balance: user.Credits, FrozenBalance: user.FrozenCredits,
			RelatedID: saved.BillingID, Remark: "结算模型费用 " + saved.Model, CreatedAt: current,
		}).Error; err != nil {
			return err
		}

		if saved.SalePriceCents == 0 {
			saved.SalePriceCents = int64(saved.Credits)
		}
		if saved.ActualProviderCostCents == 0 && saved.EstimatedProviderCostCents > 0 {
			saved.ActualProviderCostCents = saved.EstimatedProviderCostCents
			saved.ProviderCostSource = "estimated"
		}
		saved.GrossProfitCents = saved.SalePriceCents - saved.ActualProviderCostCents
		if saved.ActualProviderCostCents > 0 {
			entry := model.ProviderLedger{
				ID: "provider_ledger_" + uuid.NewString(), ProviderID: firstLedgerValue(saved.ChannelID, saved.ChannelName),
				Type: model.ProviderLedgerCost, Amount: saved.ActualProviderCostCents, Currency: "CNY",
				AmountCNY: saved.ActualProviderCostCents, TaskID: saved.ID, UpstreamTaskID: saved.UpstreamTaskID,
				Reason: "生成任务上游成本", IdempotencyKey: "provider_cost:" + saved.ID, CreatedAt: current,
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}
		saved.BillingStatus = "settled"
		saved.UpstreamRefundStatus = "not_required"
		saved.ProviderCostConfirmedAt = current
		saved.UpdatedAt = current
		if err := tx.Save(&saved).Error; err != nil {
			return err
		}
		*task = saved
		return nil
	})
}

func firstLedgerValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "unknown"
}
