package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
)

const (
	auditRequestReceived       = "REQUEST_RECEIVED"
	auditModelResolved         = "MODEL_RESOLVED"
	auditProviderResolved      = "PROVIDER_RESOLVED"
	auditProviderConfigOK      = "PROVIDER_CONFIG_OK"
	auditAdapterSelected       = "ADAPTER_SELECTED"
	auditParamValidationPassed = "PARAM_VALIDATION_PASSED"
	auditPayloadBuilt          = "PAYLOAD_BUILT"
	auditUpstreamRequestStart  = "UPSTREAM_REQUEST_START"
	auditUpstreamResponse      = "UPSTREAM_RESPONSE_RECEIVED"
	auditUpstreamTaskIDSaved   = "UPSTREAM_TASK_ID_SAVED"
)

// logGenerationAudit deliberately records routing and lifecycle metadata only.
// Request bodies, provider responses and credentials must never be added here.
func logGenerationAudit(stage, localTaskID, modelID, provider, upstreamModelID, adapter, endpoint string, fields ...any) {
	message := fmt.Sprintf("generation_audit stage=%s local_task_id=%q model_id=%q provider=%q upstream_model_id=%q adapter=%q endpoint=%q",
		stage, localTaskID, modelID, provider, upstreamModelID, adapter, endpoint)
	for index := 0; index+1 < len(fields); index += 2 {
		message += fmt.Sprintf(" %s=%q", fmt.Sprint(fields[index]), fmt.Sprint(fields[index+1]))
	}
	log.Print(message)
}

func readGenerationResolution(body []byte, contentType string) string {
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if mediaType != "application/json" {
		return ""
	}
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	for _, key := range []string{"resolution", "resolution_name", "vquality", "quality", "size", "image_size"} {
		if value := strings.TrimSpace(toStringSafe(payload[key])); value != "" {
			return value
		}
	}
	return ""
}

func auditNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func classifyGenerationError(status int, message string, requestSent bool) string {
	if !requestSent {
		return model.GenerationErrorUpstreamRequestNotSent
	}
	lower := strings.ToLower(strings.TrimSpace(message))
	switch {
	case status == 401 || status == 403 || strings.Contains(lower, "apikey") || strings.Contains(lower, "api key") || strings.Contains(lower, "unauthorized"):
		return model.GenerationErrorUpstreamAuthFailed
	case status == 429:
		return model.GenerationErrorUpstreamRateLimit
	case status >= 400:
		return model.GenerationErrorUpstreamRejected
	case strings.Contains(lower, "没有返回") && strings.Contains(lower, "url"):
		return model.GenerationErrorResultURLMissing
	case strings.Contains(lower, "参数") || strings.Contains(lower, "参考图"):
		return model.GenerationErrorInvalidParams
	default:
		return model.GenerationErrorUpstreamTaskFailed
	}
}
