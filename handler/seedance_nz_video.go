package handler

import (
	"bytes"
	"encoding/json"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
)

func isSeedanceNZChannel(channel model.ModelChannel) bool {
	return strings.EqualFold(strings.TrimSpace(channel.ID), "provider_seedance_nz") ||
		strings.Contains(strings.ToLower(strings.TrimSpace(channel.BaseURL)), "api.seedance.nz")
}

func resolveSeedanceNZVideoModel(upstreamModel string, body []byte, contentType string) string {
	modelName := strings.TrimSpace(upstreamModel)
	if !strings.HasPrefix(strings.ToLower(modelName), "seedance-2.0-") {
		return modelName
	}
	mode := seedanceNZRequestMode(body, contentType)
	for _, suffix := range []string{"-t2v", "-i2v", "-multi"} {
		if strings.HasSuffix(strings.ToLower(modelName), suffix) {
			return modelName[:len(modelName)-len(suffix)] + "-" + mode
		}
	}
	return modelName
}

func seedanceNZRequestMode(body []byte, contentType string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/form-data") {
		return seedanceNZMultipartMode(body, contentType)
	}
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return "t2v"
	}
	if metadata, ok := payload["metadata"].(map[string]any); ok {
		if content, ok := metadata["content"].([]any); ok && len(content) > 0 {
			return "multi"
		}
	}
	if values, ok := payload["images"].([]any); ok && len(values) > 0 {
		return "i2v"
	}
	if value, ok := payload["image"].(string); ok && strings.TrimSpace(value) != "" {
		return "i2v"
	}
	return "t2v"
}

func seedanceNZMultipartMode(body []byte, contentType string) string {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil || strings.TrimSpace(params["boundary"]) == "" {
		return "t2v"
	}
	form, err := multipart.NewReader(bytes.NewReader(body), params["boundary"]).ReadForm(80 << 20)
	if err != nil {
		return "t2v"
	}
	defer form.RemoveAll()
	images := len(form.Value["input_reference[]"]) + len(form.File["input_reference[]"])
	if len(form.Value["first_frame_url"])+len(form.File["first_frame_url"]) > 0 {
		images++
	}
	if len(form.Value["last_frame_url"])+len(form.File["last_frame_url"]) > 0 {
		images++
	}
	videos := len(form.Value["video_reference[]"]) + len(form.File["video_reference[]"])
	audios := len(form.Value["audio_reference[]"]) + len(form.File["audio_reference[]"])
	if videos > 0 || audios > 0 || images > 2 {
		return "multi"
	}
	if images > 0 {
		return "i2v"
	}
	return "t2v"
}
