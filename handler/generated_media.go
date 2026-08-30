package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
)

// persistGeneratedMedia copies provider output bytes unchanged. It never
// resizes, recompresses, or transcodes customer media.
func persistGeneratedMedia(userID, remoteURL, prefix string, maxBytes int64) (service.UploadedStorageObject, bool) {
	remoteURL = strings.TrimSpace(remoteURL)
	if userID == "" || !strings.HasPrefix(remoteURL, "https://") {
		return service.UploadedStorageObject{}, false
	}
	storage, err := service.PublicStorageConfig()
	if err != nil || storage.Mode == "local_indexeddb" {
		return service.UploadedStorageObject{}, false
	}
	request, err := http.NewRequest(http.MethodGet, remoteURL, nil)
	if err != nil {
		return service.UploadedStorageObject{}, false
	}
	response, err := service.SafeProxyHTTPClient().Do(request)
	if err != nil {
		log.Printf("download generated media for persistence failed: %v", err)
		return service.UploadedStorageObject{}, false
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return service.UploadedStorageObject{}, false
	}
	contentType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	extensions, _ := mime.ExtensionsByType(contentType)
	extension := ".bin"
	if len(extensions) > 0 {
		extension = extensions[0]
	} else if parsed := filepath.Ext(request.URL.Path); parsed != "" && len(parsed) <= 8 {
		extension = parsed
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil || int64(len(data)) == 0 || int64(len(data)) > maxBytes {
		return service.UploadedStorageObject{}, false
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	ctx := service.WithUser(context.Background(), model.AuthUser{ID: userID})
	uploaded, err := service.UploadStorageObject(ctx, fmt.Sprintf("%s%s", prefix, extension), contentType, data)
	if err != nil {
		log.Printf("persist generated media failed: user=%s err=%v", userID, err)
		return service.UploadedStorageObject{}, false
	}
	return uploaded, true
}
