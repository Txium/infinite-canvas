package model

// PaymentReceipt prevents the same third-party trade number from crediting
// more than one recharge order. TradeNo is only inserted after a verified
// payment, so pending orders do not compete on an empty unique value.
type PaymentReceipt struct {
	TradeNo     string `json:"tradeNo" gorm:"primaryKey"`
	OrderID     string `json:"orderId" gorm:"uniqueIndex"`
	Provider    string `json:"provider" gorm:"index"`
	SellerID    string `json:"sellerId"`
	AmountCents int    `json:"amountCents"`
	CreatedAt   string `json:"createdAt"`
}

type ProviderLedgerType string

const (
	ProviderLedgerCost          ProviderLedgerType = "provider_cost"
	ProviderLedgerTopup         ProviderLedgerType = "provider_topup"
	ProviderLedgerRefund        ProviderLedgerType = "provider_refund"
	ProviderLedgerBalanceAdjust ProviderLedgerType = "manual_balance_adjustment"
)

// ProviderLedger is append-only and deliberately separate from user credit_logs.
type ProviderLedger struct {
	ID             string             `json:"id" gorm:"primaryKey"`
	ProviderID     string             `json:"providerId" gorm:"index"`
	Type           ProviderLedgerType `json:"type" gorm:"index"`
	Amount         int64              `json:"amount"`
	Currency       string             `json:"currency"`
	AmountCNY      int64              `json:"amountCny"`
	TaskID         string             `json:"taskId" gorm:"index"`
	UpstreamTaskID string             `json:"upstreamTaskId" gorm:"index"`
	BalanceBefore  int64              `json:"balanceBefore"`
	BalanceAfter   int64              `json:"balanceAfter"`
	OperatorID     string             `json:"operatorId" gorm:"index"`
	Reason         string             `json:"reason"`
	Reference      string             `json:"reference"`
	IdempotencyKey string             `json:"idempotencyKey" gorm:"uniqueIndex"`
	CreatedAt      string             `json:"createdAt" gorm:"index"`
}

type OperatingExpense struct {
	ID          string `json:"id" gorm:"primaryKey"`
	Category    string `json:"category" gorm:"index"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	AmountCNY   int64  `json:"amountCny"`
	ExpenseDate string `json:"date" gorm:"index"`
	OperatorID  string `json:"operatorId" gorm:"index"`
	Reason      string `json:"reason"`
	Reference   string `json:"reference"`
	CreatedAt   string `json:"createdAt"`
}
