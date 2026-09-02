package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(variantID)), "krea_v2__") {
		if err := normalizeWaveSpeedKreaReferences(payload, variantID); err != nil {
			return nil, contentType, err
		}
	}
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

func normalizeWaveSpeedKreaReferences(payload map[string]any, variantID string) error {
	variantID = strings.ToLower(strings.TrimSpace(variantID))
	images := waveSpeedImageReferences(payload)
	delete(payload, "image")
	delete(payload, "image_url")
	delete(payload, "images")
	delete(payload, "reference")

	switch variantID {
	case "krea_v2__01":
		// Turbo accepts one optional source image under `image`.
		if len(images) > 0 {
			payload["image"] = images[0]
		}
	case "krea_v2__02":
		// The current Medium text-to-image endpoint documents prompt and
		// aspect ratio only. Reject references before a paid upstream task.
		if len(images) > 0 {
			return errors.New("Krea 2 Medium 当前不支持参考图，请选择 Turbo 或 Large")
		}
	case "krea_v2__03":
		// Large accepts up to ten style references as objects.
		if len(images) > 10 {
			images = images[:10]
		}
		if len(images) > 0 {
			references := make([]map[string]any, 0, len(images))
			for _, image := range images {
				references = append(references, map[string]any{"image_url": image, "strength": 1})
			}
			payload["reference"] = references
		}
	}
	return nil
}

func waveSpeedImageReferences(payload map[string]any) []string {
	result := make([]string, 0, 10)
	appendImage := func(value any) {
		image := strings.TrimSpace(fmt.Sprint(value))
		if image == "" || image == "<nil>" {
			return
		}
		for _, saved := range result {
			if saved == image {
				return
			}
		}
		result = append(result, image)
	}
	for _, name := range []string{"image", "image_url"} {
		if value, ok := payload[name]; ok {
			appendImage(value)
		}
	}
	if values, ok := payload["images"].([]any); ok {
		for _, value := range values {
			appendImage(value)
		}
	}
	if values, ok := payload["reference"].([]any); ok {
		for _, value := range values {
			if item, ok := value.(map[string]any); ok {
				appendImage(item["image_url"])
			} else {
				appendImage(value)
			}
		}
	}
	return result
}

func waveSpeedAspectRatio(size string) string {
	value := strings.ToLower(strings.TrimSpace(size))
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == 'x' || r == ':' })
	if len(parts) != 2 {
		return ""
	}
	width, widthErr := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, heightErr := strconv.Atoi(strings.TrimSpace(parts[1]))
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return ""
	}
	divisor := waveSpeedGreatestCommonDivisor(width, height)
	ratio := fmt.Sprintf("%d:%d", width/divisor, height/divisor)
	// WaveSpeed GPT Image 2 accepts these aspect ratios.  Returning an empty
	// value for any other ratio lets the upstream use its documented default
	// instead of sending an unsupported value.
	switch ratio {
	case "1:1", "1:2", "2:1", "1:3", "3:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "9:21", "21:9":
		return ratio
	default:
		return ""
	}
}

func waveSpeedGreatestCommonDivisor(a int, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
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
		if onFailure != nil {
			onFailure()
		}
		writeWaveSpeedImageError(w, errorMessage, logContext)
		return true
	}
	if len(outputs) == 0 && taskID != "" && !waveSpeedDone(status) {
		outputs, errorMessage = pollWaveSpeedTask(request, channel, taskID, "图片")
	}
	if errorMessage != "" || len(outputs) == 0 {
		if onFailure != nil {
			onFailure()
		}
		writeWaveSpeedImageError(w, firstNonEmpty(errorMessage, "WaveSpeed 图片任务完成但没有返回图片"), logContext)
		return true
	}
	writeWaveSpeedImagesResponse(w, outputs, logContext)
	return true
}

func pollWaveSpeedTask(request *http.Request, channel model.ModelChannel, taskID string, mediaLabel string) ([]string, string) {
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
		if errorMessage != "" {
			return nil, errorMessage
		}
		if len(outputs) > 0 {
			return outputs, ""
		}
		if waveSpeedFailed(status) {
			return nil, "WaveSpeed " + mediaLabel + "生成失败"
		}
		if waveSpeedDone(status) {
			return nil, "WaveSpeed " + mediaLabel + "任务完成但没有返回结果"
		}
	}
	return nil, "WaveSpeed " + mediaLabel + "任务超时"
}

func readWaveSpeedTask(payload []byte) (string, []string, string, string) {
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			ID      string   `json:"id"`
			Status  string   `json:"status"`
			Outputs []string `json:"outputs"`
			Error   string   `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return "", nil, "", "WaveSpeed 响应无法解析"
	}
	if result.Code != 0 && result.Code != 200 {
		return "", nil, result.Data.Status, firstNonEmpty(result.Message, result.Data.Error, "WaveSpeed 请求失败")
	}
	if waveSpeedFailed(result.Data.Status) {
		return result.Data.ID, result.Data.Outputs, result.Data.Status, firstNonEmpty(result.Data.Error, result.Message, "WaveSpeed 图片生成失败")
	}
	return strings.TrimSpace(result.Data.ID), result.Data.Outputs, strings.TrimSpace(result.Data.Status), ""
}

func waveSpeedDone(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "succeeded", "success":
		return true
	}
	return false
}

func waveSpeedFailed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "error", "cancelled", "canceled":
		return true
	}
	return false
}

func writeWaveSpeedImagesResponse(w http.ResponseWriter, outputs []string, logContext aiLogContext) {
	data := make([]map[string]string, 0, len(outputs))
	for _, output := range outputs {
		if output = strings.TrimSpace(output); output != "" {
			data = append(data, map[string]string{"url": output})
		}
	}
	payload, _ := json.Marshal(map[string]any{"created": time.Now().Unix(), "data": data})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
	saveAIProxyLog(logContext, http.StatusOK, string(payload), "")
}

func writeWaveSpeedImageError(w http.ResponseWriter, message string, logContext aiLogContext) {
	payload, _ := json.Marshal(map[string]any{"code": 1, "data": nil, "msg": message})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Upstream-Status", "502")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
	saveAIProxyLog(logContext, http.StatusBadGateway, string(payload), message)
}
