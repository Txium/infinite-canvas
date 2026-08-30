package handler

import (
	"net/http"
	"strconv"

	"github.com/tigerowo/infinite-canvas/service"
)

func UserGenerationTasks(w http.ResponseWriter, r *http.Request) {
	user, _ := service.UserFromContext(r.Context())
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := service.UserGenerationTasks(user, service.UserGenerationTaskQuery{Kind: r.URL.Query().Get("kind"), Status: r.URL.Query().Get("status"), Days: days, Limit: limit})
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, items)
}
