package model

type ModelProvider struct {
	ID        string `json:"id" gorm:"primaryKey"`
	Name      string `json:"name"`
	Code      string `json:"code" gorm:"uniqueIndex"`
	BaseURL   string `json:"baseUrl"`
	APIKey    string `json:"apiKey,omitempty"`
	Enabled   bool   `json:"enabled"`
	Priority  int    `json:"priority"`
	Timeout   int    `json:"timeout"`
	Remark    string `json:"remark"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
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
	ProviderID      string `json:"providerId" gorm:"index"`
	UpstreamModelID string `json:"upstreamModelId"`
	Protocol        string `json:"protocol"`
	Priority        int    `json:"priority" gorm:"index"`
	Enabled         bool   `json:"enabled"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type ModelPrice struct {
	ID           string `json:"id" gorm:"primaryKey"`
	ModelID      string `json:"modelId" gorm:"index"`
	Variant      string `json:"variant"`
	BillingMode  string `json:"billingMode"`
	Unit         string `json:"unit"`
	CostFen      int64  `json:"costFen"`
	PriceCredits int    `json:"priceCredits"`
	Currency     string `json:"currency"`
	Enabled      bool   `json:"enabled"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type MarketModelCard struct {
	MarketModel
	Prices    []ModelPrice `json:"prices"`
	Available bool         `json:"available"`
}

type ModelCatalogVersion struct {
	Key       string `json:"key" gorm:"primaryKey"`
	Version   int    `json:"version"`
	UpdatedAt string `json:"updatedAt"`
}
