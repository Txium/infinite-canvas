package model

type VideoTask struct {
	ID                          string `json:"id" gorm:"primaryKey"`
	UserID                      string `json:"userId" gorm:"index"`
	UserDisplayName             string `json:"userDisplayName"`
	Model                       string `json:"model" gorm:"index"`
	UpstreamModel               string `json:"-" gorm:"column:upstream_model"`
	Provider                    string `json:"provider"`
	Adapter                     string `json:"adapter"`
	ProviderEndpoint            string `json:"providerEndpoint"`
	UpstreamRequestSent         bool   `json:"upstreamRequestSent"`
	UpstreamRequestStartedAt    string `json:"upstreamRequestStartedAt"`
	UpstreamHTTPStatus          int    `json:"upstreamHttpStatus"`
	ProviderTaskStatus          string `json:"providerTaskStatus"`
	ChannelID                   string `json:"channelId" gorm:"index"`
	UserChannelID               string `json:"userChannelId" gorm:"index"`
	ChannelName                 string `json:"channelName"`
	Source                      string `json:"source" gorm:"index"`
	SourceID                    string `json:"source_id" gorm:"index"`
	UpstreamTaskID              string `json:"upstreamTaskId" gorm:"index"`
	UpstreamVideoID             string `json:"upstreamVideoId" gorm:"index"`
	Status                      string `json:"status" gorm:"index:idx_video_tasks_status_created_at,priority:1"`
	Progress                    int    `json:"progress"`
	Seconds                     string `json:"seconds"`
	Size                        string `json:"size"`
	VideoURL                    string `json:"videoUrl" gorm:"type:text"`
	ProviderOriginalResultURL   string `json:"providerOriginalResultUrl" gorm:"type:text"`
	CanvasResultURL             string `json:"canvasResultUrl" gorm:"type:text"`
	FrontendSelectedResolution  string `json:"frontendSelectedResolution"`
	BackendResolvedResolution   string `json:"backendResolvedResolution"`
	ProviderRequestedResolution string `json:"providerRequestedResolution"`
	ProviderFinalResolution     string `json:"providerFinalResolution"`
	ProviderFinalWidth          int    `json:"providerFinalWidth"`
	ProviderFinalHeight         int    `json:"providerFinalHeight"`
	Error                       string `json:"error" gorm:"type:text"`
	ErrorDetail                 string `json:"errorDetail" gorm:"type:text"`
	ErrorCode                   string `json:"errorCode" gorm:"index"`
	RequestBody                 string `json:"requestBody" gorm:"type:text"`
	ResponseBody                string `json:"responseBody" gorm:"type:text"`
	LastResponse                string `json:"lastResponse" gorm:"type:text"`
	Credits                     int    `json:"credits"`
	SalePriceCents              int64  `json:"salePriceCents"`
	EstimatedProviderCostCents  int64  `json:"estimatedProviderCostCents"`
	ActualProviderCostCents     int64  `json:"actualProviderCostCents"`
	GrossProfitCents            int64  `json:"grossProfitCents"`
	UpstreamRefundStatus        string `json:"upstreamRefundStatus"`
	ProviderCostSource          string `json:"providerCostSource"`
	ProviderCostConfirmedAt     string `json:"providerCostConfirmedAt"`
	BillingID                   string `json:"billingId" gorm:"index"`
	BillingStatus               string `json:"billingStatus" gorm:"index"`
	BillingPath                 string `json:"-"`
	CreatedAt                   string `json:"createdAt" gorm:"index;index:idx_video_tasks_status_created_at,priority:2"`
	UpdatedAt                   string `json:"updatedAt" gorm:"index"`
	StartedAt                   string `json:"startedAt"`
	CompletedAt                 string `json:"completedAt"`
	LastPolledAt                string `json:"lastPolledAt" gorm:"index"`
}
