package model

type FinancePeriodSummary struct {
	RechargeCents    int64 `json:"rechargeCents"`
	RefundCents      int64 `json:"refundCents"`
	NetRechargeCents int64 `json:"netRechargeCents"`
	RevenueCents     int64 `json:"revenueCents"`
	ReleasedCents    int64 `json:"releasedCents"`
	SettledTasks     int64 `json:"settledTasks"`
	ReleasedTasks    int64 `json:"releasedTasks"`
}

type AdminFinanceSummary struct {
	UserCount                  int64                 `json:"userCount"`
	AvailableBalanceCents      int64                 `json:"availableBalanceCents"`
	FrozenBalanceCents         int64                 `json:"frozenBalanceCents"`
	AllTime                    FinancePeriodSummary  `json:"allTime"`
	Today                      FinancePeriodSummary  `json:"today"`
	UpstreamCostReady          bool                  `json:"upstreamCostReady"`
	UnconsumedBalanceCents     int64                 `json:"unconsumedBalanceCents"`
	RealizedRevenueCents       int64                 `json:"realizedRevenueCents"`
	ActualProviderCostCents    int64                 `json:"actualProviderCostCents"`
	ProviderReserveCents       int64                 `json:"providerReserveCents"`
	RefundReserveCents         int64                 `json:"refundReserveCents"`
	GrossProfitCents           int64                 `json:"grossProfitCents"`
	PaymentFeeCents            int64                 `json:"paymentFeeCents"`
	CompensationCents          int64                 `json:"compensationCents"`
	OperatingCostCents         int64                 `json:"operatingCostCents"`
	EstimatedNetProfitCents    int64                 `json:"estimatedNetProfitCents"`
	Period                     string                `json:"period"`
	Selected                   FinancePeriodSummary  `json:"selected"`
	SelectedProviderCostCents  int64                 `json:"selectedProviderCostCents"`
	SelectedGrossProfitCents   int64                 `json:"selectedGrossProfitCents"`
	SelectedPaymentFeeCents    int64                 `json:"selectedPaymentFeeCents"`
	SelectedCompensationCents  int64                 `json:"selectedCompensationCents"`
	SelectedOperatingCostCents int64                 `json:"selectedOperatingCostCents"`
	SelectedNetProfitCents     int64                 `json:"selectedNetProfitCents"`
	ModelProfits               []ModelProfitSummary  `json:"modelProfits"`
	ProviderCosts              []ProviderCostSummary `json:"providerCosts"`
}

type ModelProfitSummary struct {
	Model                    string `json:"model"`
	TaskCount                int64  `json:"taskCount"`
	RevenueCents             int64  `json:"revenueCents"`
	ProviderCostCents        int64  `json:"providerCostCents"`
	GrossProfitCents         int64  `json:"grossProfitCents"`
	EstimatedCostTaskCount   int64  `json:"estimatedCostTaskCount"`
	UnconfirmedCostTaskCount int64  `json:"unconfirmedCostTaskCount"`
}

type ProviderCostSummary struct {
	Provider       string `json:"provider"`
	TodayCents     int64  `json:"todayCents"`
	Last7DaysCents int64  `json:"last7DaysCents"`
	AllTimeCents   int64  `json:"allTimeCents"`
}
