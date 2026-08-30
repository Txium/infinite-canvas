package service

import (
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

func AdminFinanceSummary() (model.AdminFinanceSummary, error) {
	current := time.Now()
	start := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location()).Format(time.RFC3339)
	return repository.AdminFinanceSummary(start)
}

func AdminProviderLedgers(admin model.AuthUser) ([]model.ProviderLedger, error) {
	if !model.IsSuperAdminRole(admin.Role) {
		return nil, safeMessageError{message: "只有 super_admin 可以查看上游资金账本"}
	}
	return repository.ListProviderLedgers(200)
}

func AdminRecordProviderTopup(admin model.AuthUser, providerID string, amountCents int64, reason, reference string) (model.ProviderLedger, error) {
	if !model.IsSuperAdminRole(admin.Role) {
		return model.ProviderLedger{}, safeMessageError{message: "只有 super_admin 可以登记上游充值"}
	}
	if amountCents <= 0 {
		return model.ProviderLedger{}, safeMessageError{message: "充值金额必须大于 0"}
	}
	if strings.TrimSpace(reason) == "" {
		return model.ProviderLedger{}, safeMessageError{message: "充值原因不能为空"}
	}
	return repository.RecordProviderTopup(strings.TrimSpace(providerID), amountCents, admin.ID, reason, reference, now())
}

func AdminOperatingExpenses(admin model.AuthUser) ([]model.OperatingExpense, error) {
	if !model.IsSuperAdminRole(admin.Role) {
		return nil, safeMessageError{message: "只有 super_admin 可以查看运营费用"}
	}
	return repository.ListOperatingExpenses(200)
}

func AdminRecordOperatingExpense(admin model.AuthUser, category string, amountCents int64, date, reason, reference string) (model.OperatingExpense, error) {
	if !model.IsSuperAdminRole(admin.Role) {
		return model.OperatingExpense{}, safeMessageError{message: "只有 super_admin 可以登记运营费用"}
	}
	allowed := map[string]bool{"server": true, "database": true, "storage": true, "cdn": true, "domain": true, "payment_fee": true, "other": true}
	category = strings.TrimSpace(category)
	if !allowed[category] || amountCents <= 0 || strings.TrimSpace(reason) == "" {
		return model.OperatingExpense{}, safeMessageError{message: "运营费用参数无效"}
	}
	if strings.TrimSpace(date) == "" {
		date = time.Now().Format("2006-01-02")
	}
	return repository.SaveOperatingExpense(model.OperatingExpense{Category: category, Amount: amountCents, Currency: "CNY", AmountCNY: amountCents, ExpenseDate: date, OperatorID: admin.ID, Reason: strings.TrimSpace(reason), Reference: strings.TrimSpace(reference), CreatedAt: now()})
}
