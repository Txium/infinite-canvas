package model

const (
	GenerationErrorModelNotFound                = "MODEL_NOT_FOUND"
	GenerationErrorModelNotMapped               = "MODEL_NOT_MAPPED"
	GenerationErrorProviderNotFound             = "PROVIDER_NOT_FOUND"
	GenerationErrorProviderDisabled             = "PROVIDER_DISABLED"
	GenerationErrorProviderKeyMissing           = "PROVIDER_KEY_MISSING"
	GenerationErrorProviderEndpointMissing      = "PROVIDER_ENDPOINT_MISSING"
	GenerationErrorAdapterNotFound              = "ADAPTER_NOT_FOUND"
	GenerationErrorInvalidParams                = "INVALID_PARAMS"
	GenerationErrorReferenceImageInvalid        = "REFERENCE_IMAGE_INVALID"
	GenerationErrorPayloadBuildFailed           = "PAYLOAD_BUILD_FAILED"
	GenerationErrorUpstreamRequestNotSent       = "UPSTREAM_REQUEST_NOT_SENT"
	GenerationErrorUpstreamAuthFailed           = "UPSTREAM_AUTH_FAILED"
	GenerationErrorUpstreamRejected             = "UPSTREAM_REJECTED"
	GenerationErrorUpstreamRateLimit            = "UPSTREAM_RATE_LIMIT"
	GenerationErrorUpstreamTaskFailed           = "UPSTREAM_TASK_FAILED"
	GenerationErrorPollFailed                   = "POLL_FAILED"
	GenerationErrorResultURLMissing             = "RESULT_URL_MISSING"
	GenerationErrorTimedOutUnknown              = "TIMED_OUT_UNKNOWN"
	GenerationErrorProviderResolutionDowngraded = "PROVIDER_RESOLUTION_DOWNGRADED"
)
