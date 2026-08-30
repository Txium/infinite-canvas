package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
)

const midjourney302PollInterval = 2 * time.Second
const midjourney302MaxWait = 10 * time.Minute

type midjourney302SubmitResponse struct {
	Code        int             `json:"code"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

type midjourney302TaskResponse struct {
	ID          string   `json:"id"`
	Status      string   `json:"status"`
	Progress    string   `json:"progress"`
	Description string   `json:"description"`
	FailReason  string   `json:"failReason"`
	ImageURL    string   `json:"imageUrl"`
	ImageURLs   []string `json:"imageUrls"`
}

func is302MidjourneyModel(modelName string) bool {
	name := strings.ToLower(strings.TrimSpace(modelName))
	return strings.HasPrefix(name, "/mj/submit/") || strings.HasPrefix(name, "/mj-turbo/submit/")
}

func is302MidjourneyRequest(channel model.ModelChannel, modelName string, path string) bool {
	return channel.ID == "provider_302" && path == "/images/generations" && is302MidjourneyModel(modelName)
}

func set302MidjourneyAuthHeader(request *http.Request, channel model.ModelChannel, modelName string) {
	if request == nil || !is302MidjourneyModel(modelName) {
		return
	}
	request.Header.Del("Authorization")
	request.Header.Set("mj-api-secret", channel.APIKey)
}

func normalize302MidjourneyRequest(body []byte, contentType string) ([]byte, string, error) {
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return nil, "", errors.New("Midjourney 当前仅支持文字生图，请移除参考图片后重试")
	}
	var input struct {
		Prompt string `json:"prompt"`
		Size   string `json:"size"`
	}
	if err := json.Unmarshal(body, &input); err != nil {
		return nil, "", errors.New("Midjourney 请求参数无效")
	}
	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		return nil, "", errors.New("Midjourney 提示词不能为空")
	}
	if ratio := midjourney302AspectRatio(input.Size); ratio != "" && !strings.Contains(prompt, "--ar ") {
		prompt += " --ar " + ratio
	}
	payload := map[string]any{
		"base64Array": []string{},
		"botType":     "MID_JOURNEY",
		"notifyHook":  "",
		"prompt":      prompt,
		"state":       "",
	}
	encoded, err := json.Marshal(payload)
	return encoded, "application/json", err
}

func midjourney302AspectRatio(size string) string {
	size = strings.TrimSpace(strings.ToLower(size))
	if strings.Contains(size, ":") {
		parts := strings.SplitN(size, ":", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0] + ":" + parts[1]
		}
	}
	var width, height int
	if _, err := fmt.Sscanf(size, "%dx%d", &width, &height); err != nil || width <= 0 || height <= 0 {
		return ""
	}
	switch {
	case width*9 == height*16:
		return "16:9"
	case width*16 == height*9:
		return "9:16"
	case width*2 == height*3:
		return "3:2"
	case width*3 == height*2:
		return "2:3"
	case width == height:
		return "1:1"
	default:
		return ""
	}
}

func copy302MidjourneyImageResponse(w http.ResponseWriter, submitRequest *http.Request, channel model.ModelChannel, logContext aiLogContext, onSuccess func(), onFailure func()) {
	fail := func(message string, status int, payload string) {
		if onFailure != nil {
			onFailure()
		}
		saveAIProxyLog(logContext, status, payload, message)
		Fail(w, message)
	}
	response, err := service.HTTPClientForChannel(channel).Do(submitRequest)
	if err != nil {
		fail("Midjourney 上游连接失败", 0, err.Error())
		return
	}
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 512*1024))
	response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		fail(readUpstreamAIErrorMessage(payload, response.StatusCode), response.StatusCode, string(payload))
		return
	}
	var submitted midjourney302SubmitResponse
	if err := json.Unmarshal(payload, &submitted); err != nil || submitted.Code != 1 {
		message := firstNonEmpty(submitted.Description, "Midjourney 任务提交失败")
		fail(message, response.StatusCode, string(payload))
		return
	}
	taskID := strings.Trim(strings.TrimSpace(string(submitted.Result)), `"`)
	if taskID == "" || taskID == "null" {
		fail("Midjourney 没有返回任务 ID", response.StatusCode, string(payload))
		return
	}

	deadline := time.Now().Add(midjourney302MaxWait)
	for time.Now().Before(deadline) {
		select {
		case <-submitRequest.Context().Done():
			fail("Midjourney 请求已取消，本次费用已退回", 0, submitRequest.Context().Err().Error())
			return
		case <-time.After(midjourney302PollInterval):
		}
		result, status, raw, err := fetch302MidjourneyTask(channel, submitRequest.URL.Path, taskID)
		if err != nil {
			if status == http.StatusTooManyRequests || status >= http.StatusInternalServerError {
				continue
			}
			fail(err.Error(), status, raw)
			return
		}
		switch strings.ToUpper(strings.TrimSpace(result.Status)) {
		case "SUCCESS":
			urls := unique302MidjourneyURLs(result)
			if len(urls) == 0 {
				fail("Midjourney 任务完成但没有返回图片", status, raw)
				return
			}
			data := make([]map[string]string, 0, len(urls))
			for _, imageURL := range urls {
				data = append(data, map[string]string{"url": imageURL})
			}
			encoded, _ := json.Marshal(map[string]any{"created": time.Now().Unix(), "data": data, "task_id": taskID})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(encoded)
			saveAIProxyLog(logContext, http.StatusOK, string(encoded), "")
			if onSuccess != nil {
				onSuccess()
			}
			return
		case "FAILURE", "FAILED", "CANCEL", "CANCELLED":
			fail(firstNonEmpty(result.FailReason, result.Description, "Midjourney 生成失败"), status, raw)
			return
		}
	}
	fail("Midjourney 超过 10 分钟仍未返回结果，本次费用已退回", http.StatusGatewayTimeout, "")
}

func fetch302MidjourneyTask(channel model.ModelChannel, submitPath string, taskID string) (midjourney302TaskResponse, int, string, error) {
	prefix := "/mj"
	if strings.HasPrefix(strings.ToLower(submitPath), "/mj-turbo/") {
		prefix = "/mj-turbo"
	}
	taskURL := strings.TrimRight(strings.TrimSpace(channel.BaseURL), "/") + prefix + "/task/" + url.PathEscape(taskID) + "/fetch"
	request, err := http.NewRequest(http.MethodGet, taskURL, nil)
	if err != nil {
		return midjourney302TaskResponse{}, 0, "", err
	}
	request.Header.Set("mj-api-secret", channel.APIKey)
	response, err := service.HTTPClientForChannel(channel).Do(request)
	if err != nil {
		return midjourney302TaskResponse{}, 0, "", err
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 512*1024))
	var result midjourney302TaskResponse
	if response.StatusCode >= http.StatusBadRequest {
		return result, response.StatusCode, string(payload), errors.New(readUpstreamAIErrorMessage(payload, response.StatusCode))
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return result, response.StatusCode, string(payload), errors.New("Midjourney 任务状态无法解析")
	}
	return result, response.StatusCode, string(payload), nil
}

func unique302MidjourneyURLs(result midjourney302TaskResponse) []string {
	values := append([]string{}, result.ImageURLs...)
	values = append(values, result.ImageURL)
	seen := map[string]bool{}
	urls := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			urls = append(urls, value)
		}
	}
	return urls
}
