package handler

import (
	"net/http"

	"github.com/tigerowo/infinite-canvas/service"
)

func AdminFinanceSummary(w http.ResponseWriter, r *http.Request) {
	result, err := service.AdminFinanceSummary()
	if err != nil { FailError(w, err); return }
	OK(w, result)
}
