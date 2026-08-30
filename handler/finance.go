package handler

import (
	"encoding/json"
	"net/http"

	"github.com/tigerowo/infinite-canvas/service"
)

func AdminFinanceSummary(w http.ResponseWriter, r *http.Request) {
	result, err := service.AdminFinanceSummary(r.URL.Query().Get("period"))
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

type providerTopupRequest struct {
	ProviderID  string `json:"providerId"`
	AmountCents int64  `json:"amountCents"`
	Reason      string `json:"reason"`
	Reference   string `json:"reference"`
}
type operatingExpenseRequest struct {
	Category    string `json:"category"`
	AmountCents int64  `json:"amountCents"`
	Date        string `json:"date"`
	Reason      string `json:"reason"`
	Reference   string `json:"reference"`
}

func AdminProviderLedgers(w http.ResponseWriter, r *http.Request) {
	admin, _ := service.UserFromContext(r.Context())
	result, err := service.AdminProviderLedgers(admin)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

func AdminRecordProviderTopup(w http.ResponseWriter, r *http.Request) {
	admin, _ := service.UserFromContext(r.Context())
	var request providerTopupRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	result, err := service.AdminRecordProviderTopup(admin, request.ProviderID, request.AmountCents, request.Reason, request.Reference)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

func AdminOperatingExpenses(w http.ResponseWriter, r *http.Request) {
	admin, _ := service.UserFromContext(r.Context())
	result, err := service.AdminOperatingExpenses(admin)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

func AdminRecordOperatingExpense(w http.ResponseWriter, r *http.Request) {
	admin, _ := service.UserFromContext(r.Context())
	var request operatingExpenseRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	result, err := service.AdminRecordOperatingExpense(admin, request.Category, request.AmountCents, request.Date, request.Reason, request.Reference)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}
