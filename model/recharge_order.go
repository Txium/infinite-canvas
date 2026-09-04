package model

type RechargeOrderStatus string

const (
	RechargeOrderPending  RechargeOrderStatus = "pending"
	RechargeOrderApproved RechargeOrderStatus = "approved"
	RechargeOrderRejected RechargeOrderStatus = "rejected"
)

// RechargeOrder 用户充值申请。支付平台接入后 ProviderTradeID 保存第三方订单号。
type RechargeOrder struct {
	ID              string              `json:"id" gorm:"primaryKey"`
	UserID          string              `json:"userId" gorm:"index"`
	Username        string              `json:"username" gorm:"-"`
	AmountCents     int                 `json:"amountCents"`
	Credits         int                 `json:"credits"`
	Status          RechargeOrderStatus `json:"status" gorm:"index"`
	PaymentMethod   string              `json:"paymentMethod"`
	PaymentNote     string              `json:"paymentNote"`
	ProviderTradeID string              `json:"providerTradeId" gorm:"index"`
	AdminRemark     string              `json:"adminRemark"`
	ReviewedBy      string              `json:"reviewedBy"`
	ReviewedAt      string              `json:"reviewedAt"`
	CreatedAt       string              `json:"createdAt"`
	UpdatedAt       string              `json:"updatedAt"`
	RefundableCents int                 `json:"refundableCents" gorm:"-"`
}

type RechargeOrderList struct {
	Items []RechargeOrder `json:"items"`
	Total int             `json:"total"`
}

type RechargePayment struct {
	Order  RechargeOrder `json:"order"`
	PayURL string        `json:"payUrl"`
}

type RefundOrderStatus string

const (
	RefundOrderPending    RefundOrderStatus = "pending"
	RefundOrderProcessing RefundOrderStatus = "processing"
	RefundOrderSucceeded  RefundOrderStatus = "succeeded"
	RefundOrderRejected   RefundOrderStatus = "rejected"
	RefundOrderFailed     RefundOrderStatus = "failed"
)

// RefundOrder records a buyer request to return unused wallet balance to the
// original Alipay trade. Amounts are integer CNY cents.
type RefundOrder struct {
	ID                        string            `json:"id" gorm:"primaryKey"`
	RechargeOrderID           string            `json:"rechargeOrderId" gorm:"index"`
	UserID                    string            `json:"userId" gorm:"index"`
	Username                  string            `json:"username" gorm:"-"`
	AmountCents               int               `json:"amountCents"`
	Reason                    string            `json:"reason"`
	Status                    RefundOrderStatus `json:"status" gorm:"index"`
	ProviderTradeID           string            `json:"-"`
	ProviderRefundAmountCents int               `json:"providerRefundAmountCents"`
	AdminRemark               string            `json:"adminRemark"`
	FailureMessage            string            `json:"failureMessage"`
	ReviewedBy                string            `json:"reviewedBy" gorm:"index"`
	ReviewedAt                string            `json:"reviewedAt"`
	CreatedAt                 string            `json:"createdAt"`
	UpdatedAt                 string            `json:"updatedAt"`
}

type RefundOrderList struct {
	Items []RefundOrder `json:"items"`
	Total int           `json:"total"`
}
