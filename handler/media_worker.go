package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const mediaWorkerDefaultMaxMB int64 = 25

type mediaWorkerHealth struct {
	Ready bool `json:"ready"`
}

func mediaWorkerMaxBytes() int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(os.Getenv("MEDIA_WORKER_MAX_MB")), 10, 64)
	if err != nil || value < 1 || value > (1<<63-1)/(1<<20) {
		value = mediaWorkerDefaultMaxMB
	}
	return value << 20
}

// Media processing is isolated from provider routing and all wallet operations.
func MediaWorkerStatus(w http.ResponseWriter, r *http.Request) {
	base, token, err := mediaWorkerConfig()
	status := map[string]any{"configured": err == nil && token != "", "ready": false, "maxBytes": mediaWorkerMaxBytes()}
	if err != nil || token == "" {
		status["msg"] = "媒体 Worker 尚未配置"
		OK(w, status)
		return
	}
	healthURL := *base
	healthURL.Path = strings.TrimRight(healthURL.Path, "/") + "/health"
	healthURL.RawQuery, healthURL.Fragment = "", ""
	ctx, cancel := context.WithTimeout(r.Context(), 70*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, healthURL.String(), nil)
	response, requestErr := (&http.Client{Timeout: 70 * time.Second}).Do(request)
	if requestErr != nil {
		status["msg"] = "媒体 Worker 唤醒失败或连接超时，请稍后重试"
		OK(w, status)
		return
	}
	defer response.Body.Close()
	var health mediaWorkerHealth
	decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&health)
	if response.StatusCode != http.StatusOK || decodeErr != nil || !health.Ready {
		status["msg"] = "媒体 Worker 尚未就绪，请稍后重试"
		OK(w, status)
		return
	}
	status["ready"] = true
	OK(w, status)
}

func mediaWorkerConfig() (*url.URL, string, error) {
	base, err := url.Parse(strings.TrimRight(os.Getenv("MEDIA_WORKER_URL"), "/"))
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return nil, "", errors.New("invalid MEDIA_WORKER_URL")
	}
	return base, os.Getenv("MEDIA_WORKER_TOKEN"), nil
}

func ProcessMedia(w http.ResponseWriter, r *http.Request) {
	base, token, err := mediaWorkerConfig()
	maxBytes := mediaWorkerMaxBytes()
	requestMaxBytes := maxBytes + (1 << 20) // multipart headers and boundaries
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") || token == "" {
		FailWithStatus(w, http.StatusServiceUnavailable, "媒体处理 Worker 尚未配置，请管理员部署独立 Worker；不会调用付费模型")
		return
	}
	if r.ContentLength > requestMaxBytes {
		FailWithStatus(w, http.StatusRequestEntityTooLarge, "视频超过当前媒体处理上传限制")
		return
	}
	// The public request reaches Go through the Next.js streaming proxy. Its
	// request context can be cancelled as soon as the proxy finishes forwarding
	// the upload body, before the worker has returned its result. Give the
	// bounded worker call its own lifetime so a completed upload is not aborted
	// by the transport layer. The 240-second timeout still prevents orphaned
	// media work from running indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	// Next.js forwards request bodies with chunked encoding. Spool a bounded
	// upload so the private worker receives a verified Content-Length.
	upload, err := os.CreateTemp("", "canvas-media-upload-*")
	if err != nil {
		Fail(w, "媒体临时存储不可用")
		return
	}
	defer os.Remove(upload.Name())
	defer upload.Close()
	size, err := io.Copy(upload, http.MaxBytesReader(w, r.Body, requestMaxBytes))
	if err != nil || size == 0 {
		FailWithStatus(w, http.StatusRequestEntityTooLarge, "上传失败或文件超过当前媒体处理限制")
		return
	}
	if _, err = upload.Seek(0, io.SeekStart); err != nil {
		Fail(w, "无法读取上传文件")
		return
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/process"
	base.RawQuery, base.Fragment = "", ""
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base.String(), upload)
	if err != nil {
		Fail(w, "无法创建媒体处理请求")
		return
	}
	request.ContentLength = size
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	client := &http.Client{Timeout: 240 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		log.Printf("media worker request failed host=%s error=%T %v", base.Host, err, err)
		FailWithStatus(w, http.StatusBadGateway, "媒体Worker连接失败或处理超时，原视频未修改")
		return
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
		if readErr != nil {
			log.Printf("media worker error response host=%s status=%d content_type=%q read_error=%T", base.Host, response.StatusCode, response.Header.Get("Content-Type"), readErr)
		} else {
			log.Printf("media worker error response host=%s status=%d content_type=%q body=%q", base.Host, response.StatusCode, response.Header.Get("Content-Type"), strings.TrimSpace(string(body)))
		}
		w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(response.StatusCode)
		_, _ = w.Write(body)
		return
	}
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(response.Body, maxBytes+1))
}
