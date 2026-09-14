package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
	"github.com/tigerowo/infinite-canvas/service"
)

var rememberCanvasUpstreamTask = repository.RememberCanvasUpstreamTask

// Persist acceptance before synchronous polling; a restart must never require
// submitting another paid request. Only opaque upstream IDs stay on the server.
func rememberWaveSpeedTask(r *http.Request, ctx aiLogContext, upstreamID string) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = rememberCanvasUpstreamTask(ctx.UserID, internalBillingID(r), upstreamID, ctx.Channel.ID, ctx.Channel.Name, ctx.Endpoint == "/audio/speech")
		if err == nil {
			logGenerationAudit(auditUpstreamTaskIDSaved, internalBillingID(r), ctx.Model, ctx.Channel.Name, firstNonEmpty(ctx.Channel.Models...), ctx.Channel.Protocol, ctx.Endpoint, "upstream_task_id", upstreamID)
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}

func saveReconcilingCanvasImageTask(task model.CanvasImageTask, detail string) {
	if latest, found, err := service.GetUserCanvasImageTask(task.UserID, task.ID); err == nil && found {
		task = latest
	}
	if task.Status == "completed" || task.Status == "failed" {
		return
	}
	task.Status, task.Error, task.ErrorDetail = "reconciling", "", detail
	if strings.Contains(strings.ToLower(detail), "timed out") || strings.Contains(detail, "暂未确认") {
		task.Status = "timed_out_unknown"
		task.ErrorCode = model.GenerationErrorTimedOutUnknown
	} else if task.ErrorCode == "" {
		task.ErrorCode = model.GenerationErrorPollFailed
	}
	if _, err := saveCanvasImageTaskWithRetry(task); err != nil {
		log.Printf("save image reconciliation: %v", err)
	}
}

func saveReconcilingCanvasAudioTask(task model.CanvasAudioTask, detail string) {
	if latest, found, err := service.GetUserCanvasAudioTask(task.UserID, task.ID); err == nil && found {
		task = latest
	}
	if task.Status == "completed" || task.Status == "failed" {
		return
	}
	task.Status, task.Error, task.ErrorDetail = "reconciling", "", detail
	if _, err := saveCanvasAudioTaskWithRetry(task); err != nil {
		log.Printf("save audio reconciliation: %v", err)
	}
}

var canvasMediaReconcileOnce sync.Once

func StartCanvasMediaReconciler() {
	canvasMediaReconcileOnce.Do(func() {
		go func() {
			for {
				reconcileCanvasMedia()
				time.Sleep(time.Minute)
			}
		}()
	})
}

func reconcileCanvasMedia() {
	before := time.Now().Add(-12 * time.Minute).Format(time.RFC3339)
	images, audios, err := repository.ListCanvasMediaToReconcile(before, time.Now().Add(-time.Minute).Format(time.RFC3339))
	if err != nil {
		log.Printf("list media reconciliation: %v", err)
		return
	}
	for _, task := range images {
		outputs, status, detail := pollAcceptedCanvasMedia(task.UserID, task.Model, task.ChannelID, task.UpstreamTaskID)
		if providerStatus := strings.TrimSpace(detail); providerStatus != "" {
			task.ProviderTaskStatus = providerStatus
		}
		if status == "failed" {
			task.ErrorCode = model.GenerationErrorUpstreamTaskFailed
			saveFailedCanvasImageTask(task, "图片生成失败", detail)
			continue
		}
		if status != "completed" || len(outputs) == 0 {
			task.Status, task.UpdatedAt = "reconciling", taskTime()
			if strings.TrimSpace(detail) != "" {
				task.ErrorDetail = detail
			}
			if _, err := saveCanvasImageTaskWithRetry(task); err != nil {
				log.Printf("save pending image: %v", err)
			}
			continue
		}
		task.ImageURL, task.ImageURLs = outputs[0], outputs
		task.ProviderOriginalResultURL = outputs[0]
		if stored, ok := persistGeneratedMedia(task.UserID, task.ImageURL, "generated-image-"+task.ID, 40<<20); ok {
			task.ImageURL, task.StorageKey, task.MimeType, task.Bytes = stored.URL, stored.StorageKey, stored.MimeType, stored.Bytes
			task.Width, task.Height = stored.Width, stored.Height
			task.ProviderFinalWidth, task.ProviderFinalHeight = stored.Width, stored.Height
			if stored.Width > 0 && stored.Height > 0 {
				task.ProviderFinalResolution = fmt.Sprintf("%dx%d", stored.Width, stored.Height)
			}
		}
		task.CanvasResultURL = task.ImageURL
		if err := settleAcceptedCanvasMedia(task.UserID, task.Model, task.Endpoint, task.ID); err != nil {
			log.Printf("settle recovered image: %v", err)
			continue
		}
		task.Status, task.Progress, task.CompletedAt, task.Error, task.ErrorDetail = "completed", 100, taskTime(), "", ""
		task.ProviderTaskStatus = "SUCCESS"
		task.ErrorCode = ""
		if _, err := saveCanvasImageTaskWithRetry(task); err != nil {
			log.Printf("save recovered image: %v", err)
		}
	}
	for _, task := range audios {
		outputs, status, detail := pollAcceptedCanvasMedia(task.UserID, task.Model, task.ChannelID, task.UpstreamTaskID)
		if status == "failed" {
			saveFailedCanvasAudioTask(task, "音频生成失败", detail)
			continue
		}
		if status != "completed" || len(outputs) == 0 {
			task.Status, task.UpdatedAt = "reconciling", taskTime()
			if _, err := saveCanvasAudioTaskWithRetry(task); err != nil {
				log.Printf("save pending audio: %v", err)
			}
			continue
		}
		task.AudioURL = outputs[0]
		if stored, ok := persistGeneratedMedia(task.UserID, task.AudioURL, "generated-audio-"+task.ID, 32<<20); ok {
			task.AudioURL, task.StorageKey, task.MimeType, task.Bytes = stored.URL, stored.StorageKey, stored.MimeType, stored.Bytes
		}
		if err := settleAcceptedCanvasMedia(task.UserID, task.Model, task.Endpoint, task.ID); err != nil {
			log.Printf("settle recovered audio: %v", err)
			continue
		}
		task.Status, task.Progress, task.CompletedAt, task.Error, task.ErrorDetail = "completed", 100, taskTime(), "", ""
		if _, err := saveCanvasAudioTaskWithRetry(task); err != nil {
			log.Printf("save recovered audio: %v", err)
		}
	}
}

func pollAcceptedCanvasMedia(userID, modelName, channelID, taskID string) ([]string, string, string) {
	channel, upstreamModel, err := selectPersistedVideoTaskChannel(model.VideoTask{UserID: userID, Model: modelName, ChannelID: channelID})
	if err != nil {
		return nil, "reconciling", "原供应商暂不可用"
	}
	if is302MidjourneyModel(upstreamModel) {
		result, status, raw, fetchErr := fetch302MidjourneyTask(channel, upstreamModel, taskID)
		if fetchErr != nil {
			if status == http.StatusTooManyRequests || status >= http.StatusInternalServerError || status == 0 {
				return nil, "reconciling", firstNonEmpty(fetchErr.Error(), raw)
			}
			return nil, "failed", firstNonEmpty(fetchErr.Error(), raw)
		}
		switch strings.ToUpper(strings.TrimSpace(result.Status)) {
		case "SUCCESS", "SUCCEEDED", "COMPLETED", "FINISHED":
			if outputs := unique302MidjourneyURLs(result); len(outputs) > 0 {
				return outputs, "completed", ""
			}
			return nil, "reconciling", "上游任务完成但结果地址暂不可用"
		case "FAILURE", "FAILED", "CANCEL", "CANCELLED":
			return nil, "failed", firstNonEmpty(result.FailReason, result.Description, "Midjourney 上游任务失败")
		default:
			return nil, "reconciling", result.Status
		}
	}
	if !isWaveSpeedChannel(channel) {
		return nil, "reconciling", "原供应商暂不支持自动对账"
	}
	r, err := http.NewRequest(http.MethodGet, service.BuildModelChannelURL(channel, "/predictions/"+url.PathEscape(taskID)+"/result"), nil)
	if err != nil {
		return nil, "reconciling", err.Error()
	}
	service.SetModelChannelAuthHeader(r, channel)
	response, err := service.HTTPClientForChannel(channel).Do(r)
	if err != nil {
		return nil, "reconciling", err.Error()
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 512<<10))
	if err != nil || response.StatusCode != http.StatusOK {
		return nil, "reconciling", "结果查询暂不可用"
	}
	_, outputs, status, detail := readWaveSpeedTask(data)
	if waveSpeedFailed(status) {
		return nil, "failed", detail
	}
	if waveSpeedDone(status) && len(outputs) > 0 && detail == "" {
		valid := make([]string, 0, len(outputs))
		for _, output := range outputs {
			parsed, err := url.Parse(output)
			if err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Hostname() != "" && parsed.User == nil {
				valid = append(valid, output)
			}
		}
		if len(valid) > 0 {
			return valid, "completed", ""
		}
	}
	return nil, "reconciling", detail
}

func settleAcceptedCanvasMedia(userID, modelName, endpoint, taskID string) error {
	logs, err := repository.ListUserTaskCreditLogs(userID, []string{taskID})
	if err != nil {
		return err
	}
	credits := 0
	for _, entry := range logs {
		if entry.Type == model.CreditLogTypeAISettle || entry.Type == model.CreditLogTypeAIRelease {
			return nil
		}
		if entry.Type == model.CreditLogTypeAIFreeze {
			credits = -entry.Amount
		}
	}
	if credits <= 0 {
		return nil
	}
	return service.SettleUserCredits(userID, strings.TrimSpace(modelName), credits, endpoint, taskID)
}
