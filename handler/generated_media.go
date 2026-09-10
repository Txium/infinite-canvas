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
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
	"github.com/tigerowo/infinite-canvas/service"
)

// persistGeneratedMedia copies provider output bytes unchanged. It never
// resizes, recompresses, or transcodes customer media.
func persistGeneratedMedia(userID, remoteURL, prefix string, maxBytes int64) (service.UploadedStorageObject, bool) {
	remoteURL = strings.TrimSpace(remoteURL)
	if userID == "" || !strings.HasPrefix(remoteURL, "https://") {
		return service.UploadedStorageObject{}, false
	}
	ctx, cancel, ok := generatedMediaStorageContext(userID)
	if !ok {
		return service.UploadedStorageObject{}, false
	}
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
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
	uploaded, err := service.UploadStorageObject(ctx, fmt.Sprintf("%s%s", prefix, extension), contentType, data)
	if err != nil {
		log.Printf("persist generated media failed: user=%s err=%v", userID, err)
		return service.UploadedStorageObject{}, false
	}
	return uploaded, true
}

// Background jobs have no request session. Resolve the actual owner instead of
// dropping their role or granting every job administrator storage privileges.
func generatedMediaStorageContext(userID string) (context.Context, context.CancelFunc, bool) {
	user, found, err := repository.GetUserByID(userID)
	if err != nil || !found || user.Status != model.UserStatusActive || user.Role == model.UserRoleGuest {
		return nil, nil, false
	}
	ctx, cancel := context.WithTimeout(service.WithUser(context.Background(), model.PublicUser(user)), 2*time.Minute)
	active, err := service.HasActiveCloudStorage(ctx)
	if err != nil || !active {
		cancel()
		return nil, nil, false
	}
	return ctx, cancel, true
}

// Audio is already downloaded and validated; store those exact bytes rather
// than retaining a large base64 payload when platform storage is available.
func persistGeneratedAudio(userID, filename, mimeType string, data []byte) (service.UploadedStorageObject, bool) {
	if len(data) == 0 || len(data) > 32<<20 {
		return service.UploadedStorageObject{}, false
	}
	ctx, cancel, ok := generatedMediaStorageContext(userID)
	if !ok {
		return service.UploadedStorageObject{}, false
	}
	defer cancel()
	uploaded, err := service.UploadStorageObject(ctx, filename, mimeType, data)
	if err != nil {
		log.Printf("persist generated audio failed: user=%s err=%v", userID, err)
		return service.UploadedStorageObject{}, false
	}
	return uploaded, true
}
