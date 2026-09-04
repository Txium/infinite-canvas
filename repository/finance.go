package repository

import (
	"sort"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
)

func AdminFinanceSummary(todayStart, period, periodStart, periodEnd string) (model.AdminFinanceSummary, error) {
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
	if err := db.Model(&model.RefundOrder{}).Where("status IN ?", []model.RefundOrderStatus{model.RefundOrderPending, model.RefundOrderProcessing}).
		Select("COALESCE(SUM(amount_cents), 0)").Scan(&result.RefundReserveCents).Error; err != nil {
		return result, err
	}
	result.UnconsumedBalanceCents = wallet.AvailableBalanceCents + wallet.FrozenBalanceCents + result.RefundReserveCents
	if err := scanFinancePeriod(db.Model(&model.CreditLog{}), &result.AllTime); err != nil {
		return result, err
	}
	if err := scanRefundPeriod(db, &result.AllTime, "", ""); err != nil {
		return result, err
	}
	if err := scanFinancePeriod(db.Model(&model.CreditLog{}).Where("created_at >= ?", todayStart), &result.Today); err != nil {
		return result, err
	}
	if err := scanRefundPeriod(db, &result.Today, todayStart, ""); err != nil {
		return result, err
	}
	result.RealizedRevenueCents = result.AllTime.RevenueCents
	if err := db.Model(&model.ProviderLedger{}).
		Select("COALESCE(SUM(CASE WHEN type = ? THEN amount_cny WHEN type = ? THEN -amount_cny ELSE 0 END), 0)", model.ProviderLedgerCost, model.ProviderLedgerRefund).
		Scan(&result.ActualProviderCostCents).Error; err != nil {
		return result, err
	}
	var videoReserve int64
	if err := db.Model(&model.VideoTask{}).Where("billing_status IN ?", []string{"frozen", "reconciling"}).
		Select("COALESCE(SUM(estimated_provider_cost_cents), 0)").Scan(&videoReserve).Error; err != nil {
		return result, err
	}
	result.ProviderReserveCents += videoReserve
	for _, taskModel := range []any{&model.CanvasImageTask{}, &model.CanvasAudioTask{}} {
		var reserve int64
		if err := db.Model(taskModel).Where("status IN ?", []string{"queued", "pending", "processing", "running", "submitted", "reconciling"}).
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
	result.Period = period
	periodLogs := db.Model(&model.CreditLog{})
	if periodStart != "" {
		periodLogs = periodLogs.Where("created_at >= ?", periodStart)
	}
	if periodEnd != "" {
		periodLogs = periodLogs.Where("created_at < ?", periodEnd)
	}
	if err := scanFinancePeriod(periodLogs, &result.Selected); err != nil {
		return result, err
	}
	if err := scanRefundPeriod(db, &result.Selected, periodStart, periodEnd); err != nil {
		return result, err
	}
	providerCosts, selectedProviderCost, err := providerCostSummaries(db, todayStart, periodStart, periodEnd)
	if err != nil {
		return result, err
	}
	result.ProviderCosts, result.SelectedProviderCostCents = providerCosts, selectedProviderCost
	result.SelectedGrossProfitCents = result.Selected.RevenueCents - selectedProviderCost
	selectedExpenses := db.Model(&model.OperatingExpense{})
	selectedCompensations := db.Model(&model.CreditLog{}).Where("type = ?", model.CreditLogTypeCompensation)
	if periodStart != "" {
		selectedExpenses = selectedExpenses.Where("created_at >= ?", periodStart)
		selectedCompensations = selectedCompensations.Where("created_at >= ?", periodStart)
	}
	if periodEnd != "" {
		selectedExpenses = selectedExpenses.Where("created_at < ?", periodEnd)
		selectedCompensations = selectedCompensations.Where("created_at < ?", periodEnd)
	}
	var selectedExpenseTotals struct{ PaymentFeeCents, OperatingCostCents int64 }
	if err := selectedExpenses.Select(`
		COALESCE(SUM(CASE WHEN category = 'payment_fee' THEN amount_cny ELSE 0 END), 0) AS payment_fee_cents,
		COALESCE(SUM(CASE WHEN category <> 'payment_fee' THEN amount_cny ELSE 0 END), 0) AS operating_cost_cents`).Scan(&selectedExpenseTotals).Error; err != nil {
		return result, err
	}
	result.SelectedPaymentFeeCents = selectedExpenseTotals.PaymentFeeCents
	result.SelectedOperatingCostCents = selectedExpenseTotals.OperatingCostCents
	if err := selectedCompensations.Select("COALESCE(SUM(amount), 0)").Scan(&result.SelectedCompensationCents).Error; err != nil {
		return result, err
	}
	result.SelectedNetProfitCents = result.SelectedGrossProfitCents - result.SelectedPaymentFeeCents - result.SelectedCompensationCents - result.SelectedOperatingCostCents
	profits, err := modelProfitSummaries(db, periodStart, periodEnd)
	if err != nil {
		return result, err
	}
	result.ModelProfits = profits
	var providerCostEntries int64
	if err := db.Model(&model.ProviderLedger{}).Where("type = ?", model.ProviderLedgerCost).Count(&providerCostEntries).Error; err != nil {
		return result, err
	}
	result.UpstreamCostReady = result.AllTime.SettledTasks == 0 || providerCostEntries >= result.AllTime.SettledTasks
	return result, nil
}

func providerCostSummaries(db *gorm.DB, todayStart, start, end string) ([]model.ProviderCostSummary, int64, error) {
	type row struct {
		Provider                                                string
		TodayCents, Last7DaysCents, AllTimeCents, SelectedCents int64
	}
	sevenDays := timeNowDate(todayStart, -6)
	query := db.Model(&model.ProviderLedger{}).Where("type IN ?", []model.ProviderLedgerType{model.ProviderLedgerCost, model.ProviderLedgerRefund})
	selectSQL := `provider_id AS provider,
		COALESCE(SUM(CASE WHEN created_at >= ? THEN CASE WHEN type = ? THEN amount_cny ELSE -amount_cny END ELSE 0 END),0) AS today_cents,
		COALESCE(SUM(CASE WHEN created_at >= ? THEN CASE WHEN type = ? THEN amount_cny ELSE -amount_cny END ELSE 0 END),0) AS last7_days_cents,
		COALESCE(SUM(CASE WHEN type = ? THEN amount_cny ELSE -amount_cny END),0) AS all_time_cents`
	args := []any{todayStart, model.ProviderLedgerCost, sevenDays, model.ProviderLedgerCost, model.ProviderLedgerCost}
	if start != "" {
		selectSQL += `, COALESCE(SUM(CASE WHEN created_at >= ?`
		args = append(args, start)
		if end != "" {
			selectSQL += ` AND created_at < ?`
			args = append(args, end)
		}
		selectSQL += ` THEN CASE WHEN type = ? THEN amount_cny ELSE -amount_cny END ELSE 0 END),0) AS selected_cents`
		args = append(args, model.ProviderLedgerCost)
	} else {
		selectSQL += `, COALESCE(SUM(CASE WHEN type = ? THEN amount_cny ELSE -amount_cny END),0) AS selected_cents`
		args = append(args, model.ProviderLedgerCost)
	}
	var rows []row
	if err := query.Select(selectSQL, args...).Group("provider_id").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	known := []string{"302", "wavespeed", "lec", "seedance_nz"}
	by := map[string]row{}
	for _, item := range rows {
		key := canonicalFinanceProvider(item.Provider)
		current := by[key]
		current.Provider = key
		current.TodayCents += item.TodayCents
		current.Last7DaysCents += item.Last7DaysCents
		current.AllTimeCents += item.AllTimeCents
		current.SelectedCents += item.SelectedCents
		by[key] = current
	}
	result := make([]model.ProviderCostSummary, 0, len(known))
	var selected int64
	for _, id := range known {
		item := by[id]
		result = append(result, model.ProviderCostSummary{Provider: id, TodayCents: item.TodayCents, Last7DaysCents: item.Last7DaysCents, AllTimeCents: item.AllTimeCents})
		selected += item.SelectedCents
	}
	return result, selected, nil
}

func canonicalFinanceProvider(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(normalized, "wavespeed"):
		return "wavespeed"
	case strings.Contains(normalized, "seedance"):
		return "seedance_nz"
	case strings.Contains(normalized, "lec") || strings.Contains(normalized, "paipu"):
		return "lec"
	case strings.Contains(normalized, "302"):
		return "302"
	default:
		return normalized
	}
}

func timeNowDate(todayStart string, days int) string {
	value, err := time.Parse(time.RFC3339, todayStart)
	if err != nil {
		return todayStart
	}
	return value.AddDate(0, 0, days).Format(time.RFC3339)
}

func modelProfitSummaries(db *gorm.DB, start, end string) ([]model.ModelProfitSummary, error) {
	type row struct {
		Model                                                                                        string
		TaskCount, RevenueCents, ProviderCostCents, EstimatedCostTaskCount, UnconfirmedCostTaskCount int64
	}
	combined := map[string]*model.ModelProfitSummary{}
	for _, table := range []string{"video_tasks", "canvas_image_tasks", "canvas_audio_tasks"} {
		query := db.Table(table).Where("status IN ?", []string{"completed", "succeeded", "success"})
		if start != "" {
			query = query.Where("created_at >= ?", start)
		}
		if end != "" {
			query = query.Where("created_at < ?", end)
		}
		var rows []row
		if err := query.Select(`model, COUNT(*) AS task_count, COALESCE(SUM(sale_price_cents),0) AS revenue_cents, COALESCE(SUM(actual_provider_cost_cents),0) AS provider_cost_cents, COALESCE(SUM(CASE WHEN provider_cost_source = 'estimated' THEN 1 ELSE 0 END),0) AS estimated_cost_task_count, COALESCE(SUM(CASE WHEN provider_cost_source IS NULL OR provider_cost_source = '' THEN 1 ELSE 0 END),0) AS unconfirmed_cost_task_count`).Group("model").Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, item := range rows {
			current := combined[item.Model]
			if current == nil {
				current = &model.ModelProfitSummary{Model: item.Model}
				combined[item.Model] = current
			}
			current.TaskCount += item.TaskCount
			current.RevenueCents += item.RevenueCents
			current.ProviderCostCents += item.ProviderCostCents
			current.EstimatedCostTaskCount += item.EstimatedCostTaskCount
			current.UnconfirmedCostTaskCount += item.UnconfirmedCostTaskCount
		}
	}
	result := make([]model.ModelProfitSummary, 0, len(combined))
	for _, item := range combined {
		item.GrossProfitCents = item.RevenueCents - item.ProviderCostCents
		result = append(result, *item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].GrossProfitCents > result[j].GrossProfitCents })
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

func scanRefundPeriod(db *gorm.DB, result *model.FinancePeriodSummary, start, end string) error {
	query := db.Model(&model.RefundOrder{}).Where("status = ?", model.RefundOrderSucceeded)
	if start != "" {
		query = query.Where("updated_at >= ?", start)
	}
	if end != "" {
		query = query.Where("updated_at < ?", end)
	}
	if err := query.Select("COALESCE(SUM(amount_cents), 0)").Scan(&result.RefundCents).Error; err != nil {
		return err
	}
	result.NetRechargeCents = result.RechargeCents - result.RefundCents
	return nil
}
