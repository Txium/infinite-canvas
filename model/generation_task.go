package model

type AdminGenerationTask struct {
	ID              string `json:"id"`
	UserID          string `json:"userId"`
	UserDisplayName string `json:"userDisplayName"`
	Kind            string `json:"kind"`
	Model           string `json:"model"`
	Status          string `json:"status"`
	BillingStatus   string `json:"billingStatus"`
	PriceCents      int    `json:"priceCents"`
	Source          string `json:"source"`
	ResultURL       string `json:"resultUrl"`
	Error           string `json:"error"`
	CreatedAt       string `json:"createdAt"`
	CompletedAt     string `json:"completedAt"`
}
