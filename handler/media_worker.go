package handler

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Media processing is isolated from provider routing and all wallet operations.
func MediaWorkerStatus(w http.ResponseWriter, r *http.Request) {
	OK(w, map[string]any{"configured": os.Getenv("MEDIA_WORKER_URL") != "" && os.Getenv("MEDIA_WORKER_TOKEN") != "", "maxBytes": 100 << 20})
}

func ProcessMedia(w http.ResponseWriter, r *http.Request) {
	base, err := url.Parse(strings.TrimRight(os.Getenv("MEDIA_WORKER_URL"), "/"))
	token := os.Getenv("MEDIA_WORKER_TOKEN")
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") || token == "" {
		FailWithStatus(w, http.StatusServiceUnavailable, "媒体处理 Worker 尚未配置，请管理员部署独立 Worker；不会调用付费模型")
		return
	}
	if r.ContentLength > 100<<20 {
		FailWithStatus(w, http.StatusRequestEntityTooLarge, "请选择小于100MB的视频")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 150*time.Second)
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
	size, err := io.Copy(upload, http.MaxBytesReader(w, r.Body, 100<<20))
	if err != nil || size == 0 {
		FailWithStatus(w, http.StatusRequestEntityTooLarge, "上传失败或文件超过100MB")
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
	client := &http.Client{Timeout: 150 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		FailWithStatus(w, http.StatusBadGateway, "媒体Worker连接失败或处理超时，原视频未修改")
		return
	}
	defer response.Body.Close()
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(response.Body, 110<<20))
}
