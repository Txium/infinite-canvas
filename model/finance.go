package model

type FinancePeriodSummary struct {
	RechargeCents int64 `json:"rechargeCents"`
	RevenueCents  int64 `json:"revenueCents"`
	ReleasedCents int64 `json:"releasedCents"`
	SettledTasks  int64 `json:"settledTasks"`
	ReleasedTasks int64 `json:"releasedTasks"`
}

type AdminFinanceSummary struct {
	UserCount               int64                `json:"userCount"`
	AvailableBalanceCents   int64                `json:"availableBalanceCents"`
	FrozenBalanceCents      int64                `json:"frozenBalanceCents"`
	AllTime                 FinancePeriodSummary `json:"allTime"`
	Today                   FinancePeriodSummary `json:"today"`
	UpstreamCostReady       bool                 `json:"upstreamCostReady"`
	UnconsumedBalanceCents  int64                `json:"unconsumedBalanceCents"`
	RealizedRevenueCents    int64                `json:"realizedRevenueCents"`
	ActualProviderCostCents int64                `json:"actualProviderCostCents"`
	ProviderReserveCents    int64                `json:"providerReserveCents"`
	GrossProfitCents        int64                `json:"grossProfitCents"`
	PaymentFeeCents         int64                `json:"paymentFeeCents"`
	CompensationCents       int64                `json:"compensationCents"`
	OperatingCostCents      int64                `json:"operatingCostCents"`
	EstimatedNetProfitCents int64                `json:"estimatedNetProfitCents"`
}
