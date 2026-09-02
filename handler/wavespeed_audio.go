package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
)

func normalizeWaveSpeedSpeechBody(body []byte, contentType string) ([]byte, string, error) {
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return nil, contentType, errors.New("语音模型仅支持 JSON 请求")
	}
	var source map[string]any
	if err := json.Unmarshal(body, &source); err != nil {
		return nil, contentType, err
	}
	text := strings.TrimSpace(waveSpeedString(source["input"]))
	if text == "" {
		return nil, contentType, errors.New("请输入需要合成的文字")
	}
	if len([]rune(text)) > 1000 {
		return nil, contentType, errors.New("MiniMax Speech 2.6 HD 每次最多合成 1000 个字符")
	}
	payload := map[string]any{
		"text":                  text,
		"voice_id":              "Wise_Woman",
		"format":                "mp3",
		"sample_rate":           44100,
		"bitrate":               128000,
		"channel":               "mono",
		"english_normalization": true,
	}
	encoded, err := json.Marshal(payload)
	return encoded, "application/json", err
}

func copyWaveSpeedAudioResponse(w http.ResponseWriter, response *http.Response, request *http.Request, channel model.ModelChannel, logContext aiLogContext, onFailure func()) bool {
	if !isWaveSpeedChannel(channel) || logContext.Endpoint != "/audio/speech" {
		return false
	}
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 512*1024))
	taskID, outputs, status, message := readWaveSpeedTask(payload)
	if len(outputs) == 0 && taskID != "" && !waveSpeedDone(status) && message == "" {
		outputs, message = pollWaveSpeedTask(request, channel, taskID, "音频")
	}
	if message != "" || len(outputs) == 0 {
		if onFailure != nil {
			onFailure()
		}
		writeWaveSpeedImageError(w, firstNonEmpty(message, "WaveSpeed 音频任务完成但没有返回音频"), logContext)
		return true
	}
	download, err := http.NewRequestWithContext(request.Context(), http.MethodGet, outputs[0], nil)
	if err != nil {
		if onFailure != nil {
			onFailure()
		}
		writeWaveSpeedImageError(w, "WaveSpeed 音频地址无效", logContext)
		return true
	}
	result, err := service.HTTPClientForChannel(channel).Do(download)
	if err != nil || result.StatusCode >= http.StatusBadRequest {
		if result != nil {
			_ = result.Body.Close()
		}
		if onFailure != nil {
			onFailure()
		}
		writeWaveSpeedImageError(w, "WaveSpeed 音频下载失败", logContext)
		return true
	}
	defer result.Body.Close()
	audio, _ := io.ReadAll(io.LimitReader(result.Body, 32*1024*1024))
	contentType := strings.TrimSpace(strings.Split(result.Header.Get("Content-Type"), ";")[0])
	if !strings.HasPrefix(contentType, "audio/") {
		contentType = http.DetectContentType(audio)
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(audio)
	saveAIProxyLog(logContext, http.StatusOK, "[binary audio]", "")
	return true
}
