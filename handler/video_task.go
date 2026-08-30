package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
)

func StartVideoTaskPoller() {
	service.StartVideoTaskPoller(pollVideoTaskFromUpstream)
}

func UserVideoTasks(w http.ResponseWriter, r *http.Request) {
	user, ok := service.UserFromContext(r.Context())
	if !ok {
		Fail(w, "未登录或权限不足")
		return
	}
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		source = "video-workbench"
	}
	if source != "video-workbench" && source != "canvas" && source != "all" {
		Fail(w, "视频任务来源无效")
		return
	}
	tasks, err := service.ListUserVideoTasks(user.ID, source, 200)
	if err != nil {
		log.Printf("list video tasks failed: user=%s err=%v", user.ID, err)
		Fail(w, "AI 接口请求失败")
		return
	}
	OK(w, tasks)
}

func DeleteUserVideoTask(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := service.UserFromContext(r.Context())
	if !ok {
		Fail(w, "未登录或权限不足")
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		Fail(w, "视频任务不存在")
		return
	}
	if err := service.DeleteUserVideoTask(user.ID, id); err != nil {
		log.Printf("delete video task failed: user=%s id=%s err=%v", user.ID, id, err)
		Fail(w, "AI 接口请求失败")
		return
	}
	OK(w, map[string]any{"deleted": true})
}

func proxyAIVideoTaskRequest(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	body, contentType, modelName, err := readAIRequest(r)
	if err != nil {
		log.Printf("AI video request read failed: %v", err)
		Fail(w, "AI 接口请求失败")
		return
	}
	user, ok := service.UserFromContext(r.Context())
	if !ok {
		Fail(w, "未登录或权限不足")
		return
	}
	requestedModel := modelName
	if err := validateFixedMarketVideoResolution(requestedModel, body, contentType); err != nil {
		Fail(w, err.Error())
		return
	}
	if err := validateRequiredVideoReferences(requestedModel, body, contentType); err != nil {
		Fail(w, err.Error())
		return
	}
	clientTaskID := readClientVideoTaskID(r)
	if clientTaskID != "" {
		if existing, found, lookupErr := service.GetUserVideoTask(user.ID, clientTaskID); lookupErr == nil && found {
			OK(w, service.VideoTaskResponse(existing))
			return
		}
	}
	billingID := firstNonEmpty(clientTaskID, "video_"+uuid.NewString())
	candidates, routed, err := service.ResolveMarketRoutes(requestedModel)
	userChannelID := ""
	if err == nil && !routed {
		channel, selectedUserChannelID, selectErr := selectAIRequestChannel(user, modelName, r.Header.Get("X-Model-Channel-ID"), r.Header.Get(userModelChannelHeader))
		userChannelID, err = selectedUserChannelID, selectErr
		if err == nil {
			candidates = []service.MarketRouteCandidate{{Channel: channel, UpstreamModel: modelName}}
		}
	}
	if err != nil {
		log.Printf("AI video select channel failed: model=%s err=%v", requestedModel, err)
		failAIChannelSelect(w, err, "AI 接口请求失败")
		return
	}
	credits := 0
	estimatedProviderCost := int64(0)
	if userChannelID == "" {
		var marketCost bool
		var billingUnit string
		credits, billingUnit, marketCost, err = service.MarketModelPricingForRequest(requestedModel, body)
		if err == nil && !marketCost {
			credits, err = service.ModelCost(requestedModel)
		}
		if err != nil {
			log.Printf("AI video read model cost failed: model=%s err=%v", requestedModel, err)
			Fail(w, err.Error())
			return
		}
		billingUnits := readAIRequestBillingUnits(body, contentType, billingUnit)
		credits *= billingUnits
		if estimatedProviderCost, err = service.MarketModelEstimatedProviderCost(requestedModel, billingUnits); err != nil {
			log.Printf("AI video read provider cost failed: model=%s err=%v", requestedModel, err)
			Fail(w, "模型成本配置无效")
			return
		}
	}
	if credits > 0 {
		if err := service.FreezeUserCredits(user.ID, requestedModel, credits, "/videos", billingID); err != nil {
			FailError(w, err)
			return
		}
	}
	// Persist before contacting the provider. A repeated client task ID now
	// returns this same record instead of creating another paid upstream task.
	task, claimed, err := service.ClaimVideoTask(service.VideoTaskCreateInput{
		UserID: user.ID, UserDisplayName: firstNonEmpty(user.DisplayName, user.Username),
		Model: requestedModel, Source: readVideoTaskSource(r), SourceID: readVideoTaskSourceID(r),
		ClientTaskID: firstNonEmpty(clientTaskID, billingID), Status: "queued",
		Credits: credits, BillingID: billingID,
		BillingStatus: map[bool]string{true: "frozen", false: ""}[credits > 0],
		BillingPath:   "/videos", SalePriceCents: int64(credits),
		EstimatedProviderCostCents: estimatedProviderCost, UpstreamRefundStatus: "not_required",
	})
	if err != nil {
		if credits > 0 {
			releaseVideoCredits(user.ID, requestedModel, credits, "/videos", billingID)
		}
		log.Printf("save video task before provider submission failed: model=%s err=%v", requestedModel, err)
		Fail(w, "视频任务创建失败，未提交上游")
		return
	}
	if !claimed {
		OK(w, service.VideoTaskResponse(task))
		return
	}
	var channel model.ModelChannel
	var request *http.Request
	var payload []byte
	var status int
	var upstreamPath string
	var logContext aiLogContext
	for index, candidate := range candidates {
		channel, modelName = candidate.Channel, candidate.UpstreamModel
		attemptBody, attemptContentType := append([]byte(nil), body...), contentType
		if routed && isSeedanceNZChannel(channel) {
			modelName = resolveSeedanceNZVideoModel(modelName, attemptBody, attemptContentType)
		}
		if routed {
			attemptBody, err = replaceAIRequestModel(attemptBody, attemptContentType, modelName)
		}
		upstreamPath = resolveAIProxyPath(channel, modelName, "/videos")
		if err == nil {
			attemptBody, attemptContentType, err = normalizeVideoCreateBody(attemptBody, attemptContentType, modelName, channel, upstreamPath)
		}
		if err == nil {
			request, err = http.NewRequest(http.MethodPost, service.BuildModelChannelURL(channel, upstreamPath), bytes.NewReader(attemptBody))
		}
		logContext = aiLogContext{StartedAt: startedAt, Endpoint: "/videos", Method: http.MethodPost, Model: requestedModel, Channel: channel, UserID: user.ID, UserDisplayName: firstNonEmpty(user.DisplayName, user.Username), Credits: credits, RequestBody: summarizeAIRequest(attemptBody, attemptContentType)}
		if err == nil {
			service.SetModelChannelAuthHeader(request, channel)
			if attemptContentType != "" {
				request.Header.Set("Content-Type", attemptContentType)
			}
			payload, status, err = doAIRequest(request, channel)
		}
		if index+1 < len(candidates) && retryableMarketRouteFailure(status, err) {
			saveAIProxyLog(logContext, status, string(payload), firstNonEmpty(errorString(err), strings.TrimSpace(string(payload))))
			err, payload, status = nil, nil, 0
			continue
		}
		break
	}
	if err != nil {
		// A transport error can happen after the provider received the request.
		// Keep funds frozen and require reconciliation; refunding here would make
		// a retry capable of charging the provider twice.
		markVideoTaskSubmissionUncertain(task, err.Error())
		saveAIProxyLog(logContext, 0, "", err.Error())
		Fail(w, "AI 接口请求失败")
		return
	}
	if status >= http.StatusBadRequest {
		message := readUpstreamAIErrorMessage(payload, status)
		failVideoTaskBeforeUpstreamAcceptance(task, message, strings.TrimSpace(string(payload)))
		saveAIProxyLog(logContext, status, string(payload), strings.TrimSpace(string(payload)))
		Fail(w, message)
		return
	}
	transformed := transformVideoCreatePayload(payload, request, channel, modelName)
	if message := readVideoCreateErrorMessage(payload, transformed, channel, modelName); message != "" {
		failVideoTaskBeforeUpstreamAcceptance(task, message, message)
		saveAIProxyLog(logContext, status, string(payload), message)
		Fail(w, message)
		return
	}
	parsed := parseVideoTaskPayload(transformed, modelName)
	if parsed.UpstreamTaskID == "" && parsed.UpstreamVideoID == "" {
		markVideoTaskSubmissionUncertain(task, "上游响应未包含任务 ID: "+string(transformed))
		saveAIProxyLog(logContext, status, string(transformed), "视频接口没有返回任务 ID")
		Fail(w, "视频接口没有返回任务 ID")
		return
	}
	task, err = completeVideoTaskSubmissionWithRetry(task, service.VideoTaskCreateInput{
		UserID:                     user.ID,
		UserDisplayName:            firstNonEmpty(user.DisplayName, user.Username),
		Model:                      requestedModel,
		UpstreamModel:              modelName,
		ChannelID:                  channel.ID,
		UserChannelID:              userChannelID,
		ChannelName:                channel.Name,
		Source:                     readVideoTaskSource(r),
		SourceID:                   readVideoTaskSourceID(r),
		ClientTaskID:               task.ID,
		UpstreamTaskID:             parsed.UpstreamTaskID,
		UpstreamVideoID:            parsed.UpstreamVideoID,
		Status:                     parsed.Status,
		Progress:                   parsed.Progress,
		Seconds:                    parsed.Seconds,
		Size:                       parsed.Size,
		VideoURL:                   parsed.VideoURL,
		Error:                      parsed.Error,
		ErrorDetail:                parsed.ErrorDetail,
		RequestBody:                logContext.RequestBody,
		ResponseBody:               string(transformed),
		Credits:                    credits,
		BillingID:                  billingID,
		BillingStatus:              map[bool]string{true: "frozen", false: ""}[credits > 0],
		BillingPath:                upstreamPath,
		SalePriceCents:             int64(credits),
		EstimatedProviderCostCents: estimatedProviderCost,
		UpstreamRefundStatus:       "not_required",
	})
	if err != nil {
		// The provider has already accepted the task. Never release here: doing so
		// would let the user retry while the first paid task continues upstream.
		log.Printf("save accepted video task failed: local_task=%s upstream_task=%s model=%s err=%v", task.ID, parsed.UpstreamTaskID, modelName, err)
		Fail(w, "上游已接收任务，本地正在对账，请勿重复提交")
		return
	}
	saveAIProxyLog(logContext, status, string(transformed), "")
	OK(w, service.VideoTaskResponse(task))
}

func completeVideoTaskSubmissionWithRetry(task model.VideoTask, input service.VideoTaskCreateInput) (model.VideoTask, error) {
	var saved model.VideoTask
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		saved, err = service.CompleteVideoTaskSubmission(task, input)
		if err == nil {
			return saved, nil
		}
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}
	return saved, err
}

func failVideoTaskBeforeUpstreamAcceptance(task model.VideoTask, message string, detail string) {
	if err := service.UpdateVideoTaskFromPoll(task, service.VideoTaskPollUpdate{
		Status: "failed", Error: message, ErrorDetail: detail,
	}); err != nil {
		log.Printf("mark unaccepted video task failed: id=%s err=%v", task.ID, err)
	}
}

func markVideoTaskSubmissionUncertain(task model.VideoTask, detail string) {
	if err := service.UpdateVideoTaskFromPoll(task, service.VideoTaskPollUpdate{
		Status: "reconciling", ErrorDetail: detail,
	}); err != nil {
		log.Printf("mark uncertain video submission failed: id=%s err=%v", task.ID, err)
	}
}

func isLECVideoChannel(channel model.ModelChannel) bool {
	id := strings.ToLower(strings.TrimSpace(channel.ID))
	name := strings.ToLower(strings.TrimSpace(channel.Name))
	baseURL := strings.ToLower(strings.TrimSpace(channel.BaseURL))
	return id == "provider_lec" || name == "lec" || strings.Contains(baseURL, "paipu.net")
}

func validateFixedMarketVideoResolution(modelName string, body []byte, contentType string) error {
	fixed := fixedMarketVideoResolution(modelName)
	if fixed == "" || !strings.Contains(strings.ToLower(contentType), "application/json") {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	requested := normalizeMarketVideoResolution(firstNonEmpty(
		toStringSafe(payload["resolution"]),
		toStringSafe(payload["resolution_name"]),
		toStringSafe(payload["vquality"]),
		toStringSafe(payload["quality"]),
	))
	if requested == "" || requested == fixed {
		return nil
	}
	return fmt.Errorf("当前视频档位固定输出 %s，不能按 %s 生成；请选择名称明确标注 %s 的高清档位", strings.ToUpper(fixed), strings.ToUpper(requested), strings.ToUpper(requested))
}

func fixedMarketVideoResolution(modelName string) string {
	value := strings.ToLower(strings.TrimSpace(modelName))
	if strings.Contains(value, "_480p") || strings.Contains(value, "-480p") {
		return "480p"
	}
	if strings.Contains(value, "_720p") || strings.Contains(value, "-720p") {
		return "720p"
	}
	switch value {
	case "seedance_2__01", "lec_seedance_2_0":
		return "720p"
	}
	if strings.HasPrefix(value, "kling_3__") {
		var index int
		if _, err := fmt.Sscanf(value, "kling_3__%d", &index); err == nil {
			if index >= 1 && index <= 8 {
				return "720p"
			}
			if index >= 9 && index <= 16 {
				return "1080p"
			}
		}
	}
	return ""
}

func normalizeMarketVideoResolution(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "480", "480p", "low", "sd":
		return "480p"
	case "720", "720p", "medium", "high", "hd", "auto":
		return "720p"
	case "1080", "1080p", "fhd", "pro":
		return "1080p"
	default:
		return ""
	}
}

func validateRequiredVideoReferences(modelName string, body []byte, contentType string) error {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	rule, constrained := lecVideoReferenceRules[modelName]
	if !constrained {
		return nil
	}
	count := countVideoReferenceImages(body, contentType)
	if count < rule.MinImages {
		return fmt.Errorf("当前视频档位必须连接至少 %d 张有效图片；请先等待图片生成和保存完成，再提交视频", rule.MinImages)
	}
	if rule.MaxImages > 0 && count > rule.MaxImages {
		return fmt.Errorf("当前视频档位最多支持 %d 张参考图片", rule.MaxImages)
	}
	return nil
}

type videoReferenceRule struct {
	MinImages int
	MaxImages int
}

// lecVideoReferenceRules is the single backend source of truth for LEC
// variants whose catalog explicitly requires image input. Variants described
// only as "supports references" remain optional and are deliberately omitted.
var lecVideoReferenceRules = map[string]videoReferenceRule{
	"seedance_2__01":               {MinImages: 1, MaxImages: 9},
	"lec_seedance_2_0":             {MinImages: 1, MaxImages: 9},
	"lec_seed_2_0_900":             {MinImages: 1, MaxImages: 9},
	"lec_seedance_2_5_ht_30s":      {MinImages: 1, MaxImages: 9},
	"lec_ac_seedance_2_5_10_image": {MinImages: 1, MaxImages: 10},
	"lec_ac_seedance_2_5_900":      {MinImages: 1, MaxImages: 9},
}

func countVideoReferenceImages(body []byte, contentType string) int {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType == "multipart/form-data" {
		form, readErr := multipart.NewReader(bytes.NewReader(body), params["boundary"]).ReadForm(80 << 20)
		if readErr != nil {
			return 0
		}
		defer form.RemoveAll()
		count := 0
		for _, key := range []string{"input_reference[]", "image", "images", "image_url", "image_urls", "first_frame_url", "start_image_url"} {
			for _, value := range form.Value[key] {
				if strings.TrimSpace(value) != "" {
					count++
				}
			}
			count += len(form.File[key])
		}
		return count
	}
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return 0
	}
	count := 0
	for _, key := range []string{"input_reference", "input_reference[]", "image", "images", "image_url", "image_urls", "reference_image", "reference_images", "reference_image_urls", "first_frame_url", "start_image_url"} {
		if !isEmptyValue(payload[key]) {
			switch value := payload[key].(type) {
			case []any:
				count += len(value)
			case []string:
				count += len(value)
			default:
				count++
			}
		}
	}
	return count
}

func readClientVideoTaskID(r *http.Request) string {
	id := strings.TrimSpace(r.Header.Get("X-Client-Video-Task-ID"))
	if isClientVideoTaskID(id) {
		return id
	}
	return ""
}

func readVideoTaskSource(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Video-Task-Source"))
}

func readVideoTaskSourceID(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Video-Task-Source-ID"))
}

func isClientVideoTaskID(id string) bool {
	return strings.HasPrefix(strings.TrimSpace(id), "client_video_task_")
}

func serveAIVideoTask(w http.ResponseWriter, r *http.Request, id string) bool {
	user, ok := service.UserFromContext(r.Context())
	if !ok {
		return false
	}
	task, found, err := service.GetUserVideoTask(user.ID, id)
	if err != nil {
		log.Printf("read video task failed: id=%s user=%s err=%v", id, user.ID, err)
		Fail(w, "AI 接口请求失败")
		return true
	}
	if !found {
		return false
	}
	OK(w, service.VideoTaskResponse(task))
	return true
}

func serveGeminiVideoTaskContent(w http.ResponseWriter, r *http.Request, id string) bool {
	user, ok := service.UserFromContext(r.Context())
	if !ok {
		return false
	}
	task, found, err := service.GetUserVideoTask(user.ID, strings.TrimSpace(id))
	if err != nil || !found {
		return false
	}
	channel, _, err := selectPersistedVideoTaskChannel(task)
	if err != nil || !service.IsGeminiChannel(channel) {
		return false
	}
	if strings.TrimSpace(task.VideoURL) == "" {
		Fail(w, "Gemini Veo 任务完成但没有返回视频地址")
		return true
	}
	request, err := http.NewRequest(http.MethodGet, task.VideoURL, nil)
	if err != nil {
		Fail(w, "视频内容下载失败")
		return true
	}
	service.SetModelChannelAuthHeader(request, channel)
	response, err := service.HTTPClientForChannel(channel).Do(request)
	if err != nil {
		Fail(w, "视频内容下载失败")
		return true
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		Fail(w, readUpstreamAIErrorMessage(nil, response.StatusCode))
		return true
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
	return true
}

func pollVideoTaskFromUpstream(task model.VideoTask) (service.VideoTaskPollUpdate, error) {
	channel, upstreamModel, err := selectPersistedVideoTaskChannel(task)
	if err != nil {
		// The provider has already accepted this task. A missing/temporarily
		// disabled local route does not prove the upstream task failed, so keep
		// funds frozen and retry reconciliation after the route is restored.
		return reconcilingVideoPollUpdate("暂时无法读取原提交线路："+err.Error(), ""), nil
	}
	// A local task ID is never a provider task ID. Sending it upstream creates
	// false 404 failures and can incorrectly release frozen funds.
	pollID := firstNonEmpty(task.UpstreamTaskID, task.UpstreamVideoID)
	if isAgnesVideoModel(upstreamModel) && strings.HasPrefix(task.UpstreamVideoID, "video_") {
		pollID = task.UpstreamVideoID
	}
	if strings.TrimSpace(pollID) == "" {
		return service.VideoTaskPollUpdate{}, errors.New("视频任务缺少上游任务 ID")
	}
	endpoint := "/videos/" + pollID
	upstreamPath := resolveAIProxyPath(channel, upstreamModel, endpoint)
	request, err := http.NewRequest(http.MethodGet, resolveAIProxyURL(channel, upstreamModel, upstreamPath), nil)
	if err != nil {
		return service.VideoTaskPollUpdate{}, err
	}
	service.SetModelChannelAuthHeader(request, channel)
	startedAt := time.Now()
	logContext := aiLogContext{
		StartedAt:       startedAt,
		Endpoint:        endpoint,
		Method:          http.MethodGet,
		Model:           task.Model,
		Channel:         channel,
		UserID:          task.UserID,
		UserDisplayName: task.UserDisplayName,
		RequestBody:     fmt.Sprintf(`{"taskId":%q}`, pollID),
	}
	payload, status, err := doAIRequest(request, channel)
	if err != nil {
		saveAIProxyLog(logContext, 0, "", err.Error())
		return service.VideoTaskPollUpdate{}, err
	}
	if status >= http.StatusBadRequest {
		message := readUpstreamAIErrorMessage(payload, status)
		saveAIProxyLog(logContext, status, string(payload), strings.TrimSpace(string(payload)))
		// HTTP errors while querying an already accepted task are not terminal
		// generation results. LEC/WaveSpeed can transiently return 404/429/5xx
		// while the paid task continues and later succeeds. Only an explicit
		// provider task payload with a failed state may release user funds.
		return reconcilingVideoPollUpdate(message, string(payload)), nil
	}
	transformed := transformVideoStatusPayload(payload, request, channel, upstreamModel)
	parsed := parseVideoTaskPayload(transformed, upstreamModel)
	if parsed.Status == "failed" && parsed.Error == "" {
		parsed.Error = firstNonEmpty(parsed.ErrorDetail, "视频任务生成失败")
	}
	if errMessage := readVideoStatusErrorMessage(payload, transformed, channel, upstreamModel); errMessage != "" {
		if parsed.Error == "" {
			parsed.Error = errMessage
		}
		parsed.Status = "failed"
	}
	if parsed.ErrorDetail == "" && len(payload) > 0 && parsed.Error != "" {
		parsed.ErrorDetail = string(payload)
	}
	if parsed.VideoURL != "" && (service.IsCompletedVideoTaskStatus(parsed.Status) || parsed.Progress >= 100) {
		if uploaded, ok := persistGeneratedMedia(task.UserID, parsed.VideoURL, "generated-video-"+task.ID, 300<<20); ok {
			parsed.VideoURL = uploaded.URL
		}
	}
	saveAIProxyLog(logContext, status, string(transformed), firstNonEmpty(parsed.Error, ""))
	return service.VideoTaskPollUpdate{
		Status:       parsed.Status,
		Progress:     parsed.Progress,
		Seconds:      parsed.Seconds,
		Size:         parsed.Size,
		VideoURL:     parsed.VideoURL,
		Error:        parsed.Error,
		ErrorDetail:  parsed.ErrorDetail,
		ResponseBody: string(transformed),
	}, nil
}

func reconcilingVideoPollUpdate(detail string, responseBody string) service.VideoTaskPollUpdate {
	return service.VideoTaskPollUpdate{
		Status:       "reconciling",
		ErrorDetail:  strings.TrimSpace(detail),
		ResponseBody: responseBody,
	}
}

// selectPersistedVideoTaskChannel restores the same managed-market provider
// used when the asynchronous task was created. Managed providers are not part
// of the legacy settings channel list, so polling them through
// SelectModelChannelForModel incorrectly reported "没有可用模型渠道".
func selectPersistedVideoTaskChannel(task model.VideoTask) (model.ModelChannel, string, error) {
	upstreamModel := videoTaskUpstreamModel(task)
	if strings.TrimSpace(task.UserChannelID) != "" {
		channel, err := service.SelectUserLocalModelChannelForModel(task.UserID, upstreamModel, task.UserChannelID)
		return channel, upstreamModel, err
	}
	candidates, routed, err := service.ResolveMarketRoutes(task.Model)
	if err != nil {
		return model.ModelChannel{}, upstreamModel, err
	}
	if routed {
		for _, candidate := range candidates {
			if candidate.Channel.ID == task.ChannelID && candidate.UpstreamModel == upstreamModel {
				return candidate.Channel, candidate.UpstreamModel, nil
			}
		}
		for _, candidate := range candidates {
			if candidate.Channel.ID == task.ChannelID {
				// A seedance.nz route stores the tier's t2v model in the catalog,
				// while request-time routing may have selected i2v or multi. Poll the
				// persisted upstream model instead of silently changing it back.
				return candidate.Channel, upstreamModel, nil
			}
		}
		if len(candidates) > 0 {
			return candidates[0].Channel, candidates[0].UpstreamModel, nil
		}
	}
	channel, err := service.SelectModelChannelForModel(upstreamModel, task.ChannelID)
	return channel, upstreamModel, err
}

func videoTaskUpstreamModel(task model.VideoTask) string {
	return firstNonEmpty(task.UpstreamModel, task.Model)
}

func normalizeVideoCreateBody(body []byte, contentType string, modelName string, channel model.ModelChannel, upstreamPath string) ([]byte, string, error) {
	if isWaveSpeedChannel(channel) && strings.Contains(strings.ToLower(upstreamPath), "/minimax-h3/") {
		return normalizeWaveSpeedH3VideoBody(body, contentType, upstreamPath)
	}
	if service.IsGeminiChannel(channel) {
		normalized, err := service.StripGeminiModelField(body, contentType)
		return normalized, contentType, err
	}
	if isKIEChannel(channel, modelName) && upstreamPath == "/jobs/createTask" {
		return normalizeKIEVideoBody(body, contentType, modelName, channel)
	}
	if isAPIMartChannel(channel, modelName) && upstreamPath == "/videos/generations" {
		return normalizeAPIMartVideoBody(body, contentType, modelName, channel)
	}
	return body, contentType, nil
}

func doAIRequest(request *http.Request, channel model.ModelChannel) ([]byte, int, error) {
	response, err := service.HTTPClientForChannel(channel).Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	return payload, response.StatusCode, nil
}

func transformVideoCreatePayload(payload []byte, request *http.Request, channel model.ModelChannel, modelName string) []byte {
	if service.IsGeminiChannel(channel) {
		if transformed, ok := transformGeminiVideoTaskResponse(payload); ok {
			return transformed
		}
	}
	if isKIEChannel(channel, modelName) && strings.Contains(request.URL.Path, "/jobs/createTask") {
		if transformed, ok := transformKIECreateVideoResponse(payload, modelName); ok {
			return transformed
		}
	}
	if isAPIMartChannel(channel, modelName) && strings.Contains(request.URL.Path, "/videos/generations") {
		if transformed, ok := transformAPIMartCreateVideoResponse(payload, modelName); ok {
			return transformed
		}
	}
	return payload
}

func transformVideoStatusPayload(payload []byte, request *http.Request, channel model.ModelChannel, modelName string) []byte {
	if service.IsGeminiChannel(channel) {
		if transformed, ok := transformGeminiVideoTaskResponse(payload); ok {
			return transformed
		}
	}
	if isMiniMaxH3Channel(channel, modelName) && strings.Contains(request.URL.Path, "/v2/query/video_generation/") {
		if transformed, ok := transformMiniMaxVideoTaskResponse(payload); ok {
			return transformed
		}
	}
	if isKIEChannel(channel, modelName) && strings.Contains(request.URL.Path, "/jobs/recordInfo") {
		if transformed, ok := transformKIETaskResponse(payload, modelName); ok {
			return transformed
		}
	}
	if isAPIMartChannel(channel, modelName) && strings.Contains(request.URL.Path, "/tasks/") {
		if transformed, ok := transformAPIMartTaskResponse(payload, modelName); ok {
			return transformed
		}
	}
	return payload
}

func transformGeminiVideoTaskResponse(payload []byte) ([]byte, bool) {
	var root map[string]any
	if len(payload) == 0 || json.Unmarshal(payload, &root) != nil {
		return nil, false
	}
	name := readStringPath(root, "name")
	done, _ := root["done"].(bool)
	videoURL := findFirstHTTPURL(root)
	errorMessage := firstNonEmpty(readStringPath(root, "error.message"))
	status := "processing"
	progress := 0
	if errorMessage != "" {
		status = "failed"
	} else if done && videoURL != "" {
		status = "completed"
		progress = 100
	} else if done {
		status = "failed"
		errorMessage = "Gemini Veo 任务完成但没有返回视频地址"
	}
	transformed, err := json.Marshal(map[string]any{
		"id":        name,
		"task_id":   name,
		"status":    status,
		"progress":  progress,
		"video_url": videoURL,
		"error":     map[string]any{"message": errorMessage},
	})
	return transformed, err == nil
}

func readVideoCreateErrorMessage(raw []byte, transformed []byte, channel model.ModelChannel, modelName string) string {
	if isKIEChannel(channel, modelName) {
		return firstNonEmpty(readKIECreateTaskErrorMessage(raw), readProviderPayloadError(raw), readNormalizedVideoError(transformed))
	}
	return firstNonEmpty(readProviderPayloadError(raw), readNormalizedVideoError(transformed))
}

func readVideoStatusErrorMessage(raw []byte, transformed []byte, channel model.ModelChannel, modelName string) string {
	if isKIEChannel(channel, modelName) {
		return firstNonEmpty(readKIERecordInfoErrorMessage(raw), readProviderPayloadError(raw), readNormalizedVideoError(transformed))
	}
	return firstNonEmpty(readProviderPayloadError(raw), readNormalizedVideoError(transformed))
}

type parsedVideoTaskPayload struct {
	UpstreamTaskID  string
	UpstreamVideoID string
	Status          string
	Progress        int
	Seconds         string
	Size            string
	VideoURL        string
	Error           string
	ErrorDetail     string
}

func parseVideoTaskPayload(payload []byte, modelName string) parsedVideoTaskPayload {
	var root any
	if len(payload) == 0 || json.Unmarshal(payload, &root) != nil {
		return parsedVideoTaskPayload{Status: "processing"}
	}
	data := normalizeVideoPayloadMap(root)
	result := parsedVideoTaskPayload{
		UpstreamTaskID:  firstNonEmpty(readStringPath(data, "task_id"), readStringPath(data, "taskId"), readStringPath(data, "id"), readStringPath(data, "request_id")),
		UpstreamVideoID: firstNonEmpty(readStringPath(data, "video_id"), readStringPath(data, "videoId")),
		Status:          service.NormalizeVideoTaskStatus(firstNonEmpty(readStringPath(data, "status"), readStringPath(data, "state"), readStringPath(data, "task_status"))),
		Progress:        readIntPath(data, "progress"),
		Seconds:         firstNonEmpty(readStringPath(data, "seconds"), readStringPath(data, "duration")),
		Size:            firstNonEmpty(readStringPath(data, "size"), readSizeFromDimensions(data)),
		VideoURL:        firstNonEmpty(readStringPath(data, "video_url"), readStringPath(data, "url"), readStringPath(data, "remixed_from_video_id"), readStringPath(data, "output_url"), readStringPath(data, "download_url"), findFirstHTTPURL(data)),
		Error:           firstNonEmpty(readStringPath(data, "error.message"), readStringPath(data, "error")),
		ErrorDetail:     "",
	}
	if result.UpstreamTaskID == result.UpstreamVideoID && strings.HasPrefix(result.UpstreamVideoID, "video_") {
		result.UpstreamTaskID = ""
	}
	if result.Status == "" {
		result.Status = "processing"
	}
	if result.VideoURL != "" {
		result.Status = "completed"
		result.Progress = 100
	}
	if result.Status == "failed" && result.Error == "" {
		result.Error = firstNonEmpty(readStringPath(data, "message"), readStringPath(data, "msg"), "视频任务生成失败")
	}
	if result.UpstreamVideoID == "" && isAgnesVideoModel(modelName) && strings.HasPrefix(result.VideoURL, "video_") {
		result.UpstreamVideoID = result.VideoURL
	}
	if result.Error != "" {
		result.ErrorDetail = string(payload)
	}
	return result
}

func normalizeVideoPayloadMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		if data, ok := typed["data"].(map[string]any); ok {
			for key, item := range typed {
				if _, exists := data[key]; !exists {
					data[key] = item
				}
			}
			return data
		}
		if data, ok := typed["data"].([]any); ok && len(data) > 0 {
			if item, ok := data[0].(map[string]any); ok {
				for key, value := range typed {
					if _, exists := item[key]; !exists {
						item[key] = value
					}
				}
				return item
			}
		}
		return typed
	default:
		return map[string]any{}
	}
}

func readNormalizedVideoError(payload []byte) string {
	parsed := parseVideoTaskPayload(payload, "")
	if parsed.Status == "failed" || parsed.Error != "" {
		return firstNonEmpty(parsed.Error, "视频任务生成失败")
	}
	return ""
}

func readProviderPayloadError(payload []byte) string {
	var value map[string]any
	if len(payload) == 0 || json.Unmarshal(payload, &value) != nil {
		return ""
	}
	code, hasCode := value["code"]
	if !hasCode {
		return ""
	}
	successCode := false
	switch typed := code.(type) {
	case float64:
		successCode = typed == 0 || typed == 200
	case string:
		text := strings.TrimSpace(strings.ToLower(typed))
		successCode = text == "" || text == "0" || text == "200" || text == "success" || text == "ok"
	default:
		successCode = false
	}
	if successCode {
		return ""
	}
	return firstNonEmpty(readStringPath(value, "error.message"), readStringPath(value, "error"), readStringPath(value, "message"), readStringPath(value, "msg"), fmt.Sprint(code))
}

func readStringPath(data map[string]any, path string) string {
	var current any = data
	for _, part := range strings.Split(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[part]
	}
	return strings.TrimSpace(toStringSafe(current))
}

func readIntPath(data map[string]any, key string) int {
	value := data[key]
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		number, _ := typed.Int64()
		return int(number)
	case string:
		var number int
		_, _ = fmt.Sscanf(strings.TrimSpace(typed), "%d", &number)
		return number
	default:
		return 0
	}
}

func readSizeFromDimensions(data map[string]any) string {
	width := readIntPath(data, "width")
	height := readIntPath(data, "height")
	if width > 0 && height > 0 {
		return fmt.Sprintf("%dx%d", width, height)
	}
	return ""
}

func findFirstHTTPURL(value any) string {
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
			return text
		}
		var parsed any
		if json.Unmarshal([]byte(text), &parsed) == nil {
			return findFirstHTTPURL(parsed)
		}
	case []any:
		for _, item := range typed {
			if url := findFirstHTTPURL(item); url != "" {
				return url
			}
		}
	case map[string]any:
		for _, key := range []string{"uri", "url", "video_url", "videoUrl", "download_url", "downloadUrl", "output_url", "outputUrl", "outputs", "output", "content", "resultUrls", "result_urls", "videoUrls", "video_urls", "urls", "videos", "video_result", "video", "generatedSamples", "generateVideoResponse", "response", "data", "result", "metadata"} {
			if url := findFirstHTTPURL(typed[key]); url != "" {
				return url
			}
		}
	}
	return ""
}

func releaseVideoCredits(userID string, modelName string, credits int, endpoint string, billingID string) {
	if err := service.ReleaseUserCredits(userID, modelName, credits, endpoint, billingID); err != nil {
		log.Printf("AI video release credits failed: user=%s model=%s credits=%d err=%v", userID, modelName, credits, err)
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
