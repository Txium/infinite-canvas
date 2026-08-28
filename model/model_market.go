package model

type ModelProvider struct {
	ID                       string `json:"id" gorm:"primaryKey"`
	Name                     string `json:"name"`
	Code                     string `json:"code" gorm:"uniqueIndex"`
	BaseURL                  string `json:"baseUrl"`
	APIKey                   string `json:"apiKey,omitempty"`
	HasAPIKey                bool   `json:"hasApiKey" gorm:"-"`
	Enabled                  bool   `json:"enabled"`
	Priority                 int    `json:"priority"`
	Timeout                  int    `json:"timeout"`
	BalanceCents             *int64 `json:"balanceCents"`
	BalanceCheckedAt         string `json:"balanceCheckedAt"`
	WarningBalanceCents      int64  `json:"warningBalanceCents"`
	CriticalBalanceCents     int64  `json:"criticalBalanceCents"`
	LowBalanceCents          int64  `json:"lowBalanceCents"`
	Ready                    bool   `json:"ready" gorm:"-"`
	RouteCount               int    `json:"routeCount" gorm:"-"`
	EnabledRouteCount        int    `json:"enabledRouteCount" gorm:"-"`
	BalanceStatus            string `json:"balanceStatus" gorm:"-"`
	BalanceMessage           string `json:"balanceMessage" gorm:"-"`
	Remark                   string `json:"remark"`
	CreatedAt                string `json:"createdAt"`
	UpdatedAt                string `json:"updatedAt"`
}

type MarketModel struct {
	ID                    string   `json:"id" gorm:"primaryKey"`
	Name                  string   `json:"name"`
	Category              string   `json:"category" gorm:"index"`
	Icon                  string   `json:"icon"`
	Description           string   `json:"description"`
	Modes                 []string `json:"modes" gorm:"serializer:json"`
	Resolutions           []string `json:"resolutions" gorm:"serializer:json"`
	Durations             []string `json:"durations" gorm:"serializer:json"`
	Ratios                []string `json:"ratios" gorm:"serializer:json"`
	MaxReferenceImages    int      `json:"maxReferenceImages"`
	SupportsPerson        bool     `json:"supportsPerson"`
	SupportsFirstLastFrame bool    `json:"supportsFirstLastFrame"`
	SupportsAudioReference bool    `json:"supportsAudioReference"`
	Speed                 string   `json:"speed"`
	Featured              bool     `json:"featured" gorm:"index"`
	Status                string   `json:"status" gorm:"index"`
	Enabled               bool     `json:"enabled" gorm:"index"`
	Sort                   int      `json:"sort"`
	CreatedAt              string   `json:"createdAt"`
	UpdatedAt              string   `json:"updatedAt"`
}

type ModelRoute struct {
	ID              string `json:"id" gorm:"primaryKey"`
	ModelID         string `json:"modelId" gorm:"index"`
	VariantID       string `json:"variantId" gorm:"index"`
	ProviderID      string `json:"providerId" gorm:"index"`
	UpstreamModelID string `json:"upstreamModelId"`
	Protocol        string `json:"protocol"`
	Priority        int    `json:"priority" gorm:"index"`
	Enabled         bool   `json:"enabled"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type ModelVariant struct {
	ID              string `json:"id" gorm:"primaryKey"`
	ModelID         string `json:"modelId" gorm:"index"`
	Name            string `json:"name"`
	ProviderCode    string `json:"providerCode" gorm:"index"`
	UpstreamModelID string `json:"upstreamModelId"`
	CostCents       *int64 `json:"costCents"`
	CostText        string `json:"costText"`
	PriceCents      *int64 `json:"priceCents"`
	PriceText       string `json:"priceText"`
	BillingUnit     string `json:"billingUnit"`
	PricingMode     string `json:"pricingMode" gorm:"index"`
	PriceFormula    string `json:"priceFormula"`
	MarginText      string `json:"marginText"`
	PersonNote      string `json:"personNote"`
	RefundPolicy    string `json:"refundPolicy"`
	SourceURL       string `json:"sourceUrl"`
	Remark          string `json:"remark"`
	Enabled         bool   `json:"enabled" gorm:"index"`
	Sort            int    `json:"sort"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type PublicModelVariant struct {
	ID           string `json:"id"`
	ModelID      string `json:"modelId"`
	Name         string `json:"name"`
	PriceCents   *int64 `json:"priceCents"`
	PriceText    string `json:"priceText"`
	BillingUnit  string `json:"billingUnit"`
	PricingMode  string `json:"pricingMode"`
	PriceFormula string `json:"priceFormula"`
	PersonNote   string `json:"personNote"`
	Remark       string `json:"remark"`
	Enabled      bool   `json:"enabled"`
	Sort         int    `json:"sort"`
}

type MarketModelCard struct {
	MarketModel
	Variants            []PublicModelVariant `json:"variants"`
	AvailableVariantIDs []string       `json:"availableVariantIds"`
	Available           bool           `json:"available"`
}

type ModelCatalogVersion struct {
	Key       string `json:"key" gorm:"primaryKey"`
	Version   int    `json:"version"`
	UpdatedAt string `json:"updatedAt"`
}
