package model

type AdminGenerationTask struct {
	ID                          string `json:"id"`
	UserID                      string `json:"userId"`
	UserDisplayName             string `json:"userDisplayName"`
	Kind                        string `json:"kind"`
	Model                       string `json:"model"`
	Status                      string `json:"status"`
	BillingStatus               string `json:"billingStatus"`
	PriceCents                  int    `json:"priceCents"`
	Source                      string `json:"source"`
	ChannelName                 string `json:"channelName"`
	Provider                    string `json:"provider"`
	UpstreamModelID             string `json:"upstreamModelId"`
	Adapter                     string `json:"adapter"`
	ProviderEndpoint            string `json:"providerEndpoint"`
	UpstreamRequestSent         bool   `json:"upstreamRequestSent"`
	UpstreamHTTPStatus          int    `json:"upstreamHttpStatus"`
	UpstreamTaskID              string `json:"upstreamTaskId"`
	ProviderTaskStatus          string `json:"providerTaskStatus"`
	FrontendSelectedResolution  string `json:"frontendSelectedResolution"`
	ProviderRequestedResolution string `json:"providerRequestedResolution"`
	ProviderFinalResolution     string `json:"providerFinalResolution"`
	ProviderOriginalResultURL   string `json:"providerOriginalResultUrl"`
	CanvasResultURL             string `json:"canvasResultUrl"`
	ErrorCode                   string `json:"errorCode"`
	ResultURL                   string `json:"resultUrl"`
	Error                       string `json:"error"`
	CreatedAt                   string `json:"createdAt"`
	CompletedAt                 string `json:"completedAt"`
}
