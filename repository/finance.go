package repository

import (
	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
)

func AdminFinanceSummary(todayStart string) (model.AdminFinanceSummary, error) {
	db, err := DB()
	if err != nil {
		return model.AdminFinanceSummary{}, err
	}
	result := model.AdminFinanceSummary{}
	var wallet struct {
		UserCount             int64
		AvailableBalanceCents int64
		FrozenBalanceCents    int64
	}
	if err := db.Model(&model.User{}).Select("COUNT(*) AS user_count, COALESCE(SUM(credits), 0) AS available_balance_cents, COALESCE(SUM(frozen_credits), 0) AS frozen_balance_cents").Scan(&wallet).Error; err != nil {
		return result, err
	}
	result.UserCount = wallet.UserCount
	result.AvailableBalanceCents = wallet.AvailableBalanceCents
	result.FrozenBalanceCents = wallet.FrozenBalanceCents
	result.UnconsumedBalanceCents = wallet.AvailableBalanceCents + wallet.FrozenBalanceCents
	if err := scanFinancePeriod(db.Model(&model.CreditLog{}), &result.AllTime); err != nil {
		return result, err
	}
	if err := scanFinancePeriod(db.Model(&model.CreditLog{}).Where("created_at >= ?", todayStart), &result.Today); err != nil {
		return result, err
	}
	result.RealizedRevenueCents = result.AllTime.RevenueCents
	if err := db.Model(&model.ProviderLedger{}).
		Select("COALESCE(SUM(CASE WHEN type = ? THEN amount_cny WHEN type = ? THEN -amount_cny ELSE 0 END), 0)", model.ProviderLedgerCost, model.ProviderLedgerRefund).
		Scan(&result.ActualProviderCostCents).Error; err != nil {
		return result, err
	}
	for _, taskModel := range []any{&model.VideoTask{}, &model.CanvasImageTask{}, &model.CanvasAudioTask{}} {
		var reserve int64
		if err := db.Model(taskModel).Where("billing_status IN ?", []string{"frozen", "reconciling"}).
			Select("COALESCE(SUM(estimated_provider_cost_cents), 0)").Scan(&reserve).Error; err != nil {
			return result, err
		}
		result.ProviderReserveCents += reserve
	}
	var expenses struct{ PaymentFeeCents, OperatingCostCents int64 }
	if err := db.Model(&model.OperatingExpense{}).Select(`
		COALESCE(SUM(CASE WHEN category = 'payment_fee' THEN amount_cny ELSE 0 END), 0) AS payment_fee_cents,
		COALESCE(SUM(CASE WHEN category <> 'payment_fee' THEN amount_cny ELSE 0 END), 0) AS operating_cost_cents`).Scan(&expenses).Error; err != nil {
		return result, err
	}
	result.PaymentFeeCents = expenses.PaymentFeeCents
	result.OperatingCostCents = expenses.OperatingCostCents
	if err := db.Model(&model.CreditLog{}).Where("type = ?", model.CreditLogTypeCompensation).
		Select("COALESCE(SUM(amount), 0)").Scan(&result.CompensationCents).Error; err != nil {
		return result, err
	}
	result.GrossProfitCents = result.RealizedRevenueCents - result.ActualProviderCostCents
	result.EstimatedNetProfitCents = result.GrossProfitCents - result.PaymentFeeCents - result.CompensationCents - result.OperatingCostCents
	result.UpstreamCostReady = true
	return result, nil
}

func scanFinancePeriod(query *gorm.DB, result *model.FinancePeriodSummary) error {
	return query.Select(`
		COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS recharge_cents,
		COALESCE(SUM(CASE WHEN type = ? THEN -frozen_amount ELSE 0 END), 0) AS revenue_cents,
		COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS released_cents,
		COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END), 0) AS settled_tasks,
		COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END), 0) AS released_tasks`,
		model.CreditLogTypeRecharge,
		model.CreditLogTypeAISettle,
		model.CreditLogTypeAIRelease,
		model.CreditLogTypeAISettle,
		model.CreditLogTypeAIRelease,
	).Scan(result).Error
}
