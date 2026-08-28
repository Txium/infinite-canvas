package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
)

func isWaveSpeedChannel(channel model.ModelChannel) bool {
	return strings.EqualFold(strings.TrimSpace(channel.Protocol), "wavespeed") ||
		strings.Contains(strings.ToLower(strings.TrimSpace(channel.BaseURL)), "api.wavespeed.ai")
}

func normalizeWaveSpeedImageBody(body []byte, contentType string, variantID string) ([]byte, string, error) {
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return body, contentType, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, contentType, err
	}
	delete(payload, "model")
	delete(payload, "n")
	aspectRatio, _ := payload["aspect_ratio"].(string)
	if size, ok := payload["size"].(string); ok {
		if converted := waveSpeedAspectRatio(size); converted != "" {
			aspectRatio = converted
			payload["aspect_ratio"] = converted
		}
	}
	delete(payload, "size")
	if quality, resolution, ok := waveSpeedGPTImageTier(variantID); ok {
		// GPT Image 2 currently accepts only these documented inputs. Strip
		// OpenAI-only response and streaming options so a valid request is not
		// rejected after the upstream has accepted a paid task.
		cleaned := map[string]any{
			"prompt":     payload["prompt"],
			"quality":    quality,
			"resolution": resolution,
		}
		if strings.TrimSpace(aspectRatio) != "" {
			cleaned["aspect_ratio"] = aspectRatio
		}
		payload = cleaned
	}
	encoded, err := json.Marshal(payload)
	return encoded, "application/json", err
}

func waveSpeedAspectRatio(size string) string {
	switch strings.ToLower(strings.TrimSpace(size)) {
	case "1024x1024", "1:1":
		return "1:1"
	case "1536x1024", "3:2":
		return "3:2"
	case "1024x1536", "2:3":
		return "2:3"
	default:
		return ""
	}
}

func waveSpeedGPTImageTier(variantID string) (string, string, bool) {
	tiers := map[string][2]string{
		"gpt_image_2__01": {"low", "1k"},
		"gpt_image_2__02": {"low", "2k"},
		"gpt_image_2__03": {"low", "4k"},
		"gpt_image_2__04": {"medium", "1k"},
		"gpt_image_2__05": {"medium", "2k"},
		"gpt_image_2__06": {"medium", "4k"},
		"gpt_image_2__07": {"high", "1k"},
		"gpt_image_2__08": {"high", "2k"},
		"gpt_image_2__09": {"high", "4k"},
	}
	tier, ok := tiers[strings.ToLower(strings.TrimSpace(variantID))]
	return tier[0], tier[1], ok
}

func copyWaveSpeedImageResponse(w http.ResponseWriter, response *http.Response, request *http.Request, channel model.ModelChannel, logContext aiLogContext, onFailure func()) bool {
	if !isWaveSpeedChannel(channel) || (!strings.Contains(request.URL.Path, "/text-to-image") && !strings.Contains(request.URL.Path, "/image-to-image")) {
		return false
	}
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 512*1024))
	taskID, outputs, status, errorMessage := readWaveSpeedTask(payload)
	if errorMessage != "" {
		if onFailure != nil { onFailure() }
		writeWaveSpeedImageError(w, errorMessage, logContext)
		return true
	}
	if len(outputs) == 0 && taskID != "" && !waveSpeedDone(status) {
		outputs, errorMessage = pollWaveSpeedImageTask(request, channel, taskID)
	}
	if errorMessage != "" || len(outputs) == 0 {
		if onFailure != nil { onFailure() }
		writeWaveSpeedImageError(w, firstNonEmpty(errorMessage, "WaveSpeed 图片任务完成但没有返回图片"), logContext)
		return true
	}
	writeWaveSpeedImagesResponse(w, outputs, logContext)
	return true
}

func pollWaveSpeedImageTask(request *http.Request, channel model.ModelChannel, taskID string) ([]string, string) {
	pollURL := service.BuildModelChannelURL(channel, "/predictions/"+url.PathEscape(taskID)+"/result")
	for attempt := 0; attempt < 300; attempt++ {
		if attempt > 0 {
			select {
			case <-request.Context().Done():
				return nil, request.Context().Err().Error()
			case <-time.After(2 * time.Second):
			}
		}
		pollRequest, err := http.NewRequestWithContext(request.Context(), http.MethodGet, pollURL, nil)
		if err != nil {
			return nil, err.Error()
		}
		service.SetModelChannelAuthHeader(pollRequest, channel)
		pollResponse, err := service.HTTPClientForChannel(channel).Do(pollRequest)
		if err != nil {
			return nil, err.Error()
		}
		body, _ := io.ReadAll(io.LimitReader(pollResponse.Body, 512*1024))
		_ = pollResponse.Body.Close()
		if pollResponse.StatusCode >= http.StatusBadRequest {
			return nil, readUpstreamAIErrorMessage(body, pollResponse.StatusCode)
		}
		_, outputs, status, errorMessage := readWaveSpeedTask(body)
		if errorMessage != "" { return nil, errorMessage }
		if len(outputs) > 0 { return outputs, "" }
		if waveSpeedFailed(status) { return nil, "WaveSpeed 图片生成失败" }
		if waveSpeedDone(status) { return nil, "WaveSpeed 图片任务完成但没有返回图片" }
	}
	return nil, "WaveSpeed 图片任务超时"
}

func readWaveSpeedTask(payload []byte) (string, []string, string, string) {
	var result struct {
		Code int `json:"code"`
		Message string `json:"message"`
		Data struct {
			ID string `json:"id"`
			Status string `json:"status"`
			Outputs []string `json:"outputs"`
			Error string `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil { return "", nil, "", "WaveSpeed 响应无法解析" }
	if result.Code != 0 && result.Code != 200 { return "", nil, result.Data.Status, firstNonEmpty(result.Message, result.Data.Error, "WaveSpeed 请求失败") }
	if waveSpeedFailed(result.Data.Status) { return result.Data.ID, result.Data.Outputs, result.Data.Status, firstNonEmpty(result.Data.Error, result.Message, "WaveSpeed 图片生成失败") }
	return strings.TrimSpace(result.Data.ID), result.Data.Outputs, strings.TrimSpace(result.Data.Status), ""
}

func waveSpeedDone(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) { case "completed", "succeeded", "success": return true }
	return false
}

func waveSpeedFailed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) { case "failed", "error", "cancelled", "canceled": return true }
	return false
}

func writeWaveSpeedImagesResponse(w http.ResponseWriter, outputs []string, logContext aiLogContext) {
	data := make([]map[string]string, 0, len(outputs))
	for _, output := range outputs { if output = strings.TrimSpace(output); output != "" { data = append(data, map[string]string{"url": output}) } }
	payload, _ := json.Marshal(map[string]any{"created":time.Now().Unix(), "data":data})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
	saveAIProxyLog(logContext, http.StatusOK, string(payload), "")
}

func writeWaveSpeedImageError(w http.ResponseWriter, message string, logContext aiLogContext) {
	payload, _ := json.Marshal(map[string]any{"code":1, "data":nil, "msg":message})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Upstream-Status", "502")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
	saveAIProxyLog(logContext, http.StatusBadGateway, string(payload), message)
}
