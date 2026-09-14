package handler

import (
	"encoding/json"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
	"net/http"
)

func MediaAICapabilities(w http.ResponseWriter, r *http.Request) {
	OK(w, service.MediaAICapabilities())
}
func SubmitMediaAI(w http.ResponseWriter, r *http.Request) {
	var request model.MediaAIRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request) != nil {
		FailWithStatus(w, 400, "任务参数无效")
		return
	}
	adapter, err := service.ResolveMediaAIAdapter(request.Operation)
	if err != nil {
		FailWithStatus(w, 400, err.Error())
		return
	}
	_, err = adapter.Submit(r.Context(), request)
	if err != nil {
		FailWithStatus(w, 503, err.Error())
		return
	}
	FailWithStatus(w, 503, "任务执行器未启用")
}
