package handler

import (
	"encoding/json"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
	"net/http"
)

func VoiceProfiles(w http.ResponseWriter, r *http.Request) {
	items, err := service.CurrentVoiceProfiles(r.Context())
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, items)
}
func SaveVoiceProfile(w http.ResponseWriter, r *http.Request) {
	var p model.VoiceProfile
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&p) != nil {
		FailWithStatus(w, 400, "音色参数无效")
		return
	}
	saved, err := service.SaveCurrentVoiceProfile(r.Context(), p)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, saved)
}
