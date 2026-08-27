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
}

type RechargeOrderList struct {
	Items []RechargeOrder `json:"items"`
	Total int             `json:"total"`
}
