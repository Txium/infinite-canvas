package model

type FinancePeriodSummary struct {
	RechargeCents int64 `json:"rechargeCents"`
	RevenueCents  int64 `json:"revenueCents"`
	ReleasedCents int64 `json:"releasedCents"`
	SettledTasks  int64 `json:"settledTasks"`
	ReleasedTasks int64 `json:"releasedTasks"`
}

type AdminFinanceSummary struct {
	UserCount           int64                `json:"userCount"`
	AvailableBalanceCents int64              `json:"availableBalanceCents"`
	FrozenBalanceCents  int64                `json:"frozenBalanceCents"`
	AllTime             FinancePeriodSummary `json:"allTime"`
	Today               FinancePeriodSummary `json:"today"`
	UpstreamCostReady   bool                 `json:"upstreamCostReady"`
}
