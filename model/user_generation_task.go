package model

type UserGenerationTask struct {
	TaskID            string `json:"task_id"`
	DisplayModelName  string `json:"display_model_name"`
	VariantName       string `json:"variant_name"`
	TaskType          string `json:"task_type"`
	Status            string `json:"status"`
	SalePriceCents    int64  `json:"sale_price_cents"`
	RefundAmountCents int64  `json:"refund_amount_cents"`
	Progress          int    `json:"progress"`
	CreatedAt         string `json:"created_at"`
	CompletedAt       string `json:"completed_at"`
	DurationSeconds   int64  `json:"duration_seconds"`
	InputSummary      string `json:"input_summary"`
	ResultURL         string `json:"result_url"`
	UserFriendlyError string `json:"user_friendly_error"`
}
