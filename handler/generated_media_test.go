package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tigerowo/infinite-canvas/config"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
	"github.com/tigerowo/infinite-canvas/service"
)

type generatedMediaTransport func(*http.Request) (*http.Response, error)

func (f generatedMediaTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGeneratedMediaStoragePreservesOwnerAndBytes(t *testing.T) {
	previous := config.Cfg
	config.Cfg = config.Config{StorageDriver: "sqlite", DatabaseDSN: filepath.Join(t.TempDir(), "generated.db")}
	t.Cleanup(func() { config.Cfg = previous })
	db, err := repository.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	for _, user := range []model.User{
		{ID: "admin", Username: "storage-admin", AffCode: "storage-admin", Role: model.UserRoleSuperAdmin, Status: model.UserStatusActive},
		{ID: "user", Username: "storage-user", AffCode: "storage-user", Role: model.UserRoleUser, Status: model.UserStatusActive},
		{ID: "banned", Username: "storage-banned", AffCode: "storage-banned", Role: model.UserRoleAdmin, Status: model.UserStatusBan},
	} {
		if _, err := repository.SaveUser(user); err != nil {
			t.Fatal(err)
		}
	}
	settings := model.Settings{Private: model.PrivateSetting{Storage: model.PrivateStorageSetting{
		Mode:                    "server_sqlite_s3",
		AllowUserGlobalProvider: false,
		Providers:               []model.StorageProvider{{ID: "test-s3", Type: "s3", Enabled: true, Endpoint: "https://storage.example.test/storage/v1/s3", Bucket: "assets", PathPrefix: "canvas", Region: "test", AccessKeyID: "test-key", SecretAccessKey: "test-secret"}},
	}}}
	if _, err := repository.SaveSettings(settings, "2026-09-10T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	client := service.SafeProxyHTTPClient()
	original := client.Transport
	t.Cleanup(func() { client.Transport = original })
	audioBytes := []byte{'I', 'D', '3', 0, 1, 2, 3, 4}
	var puts int
	client.Transport = generatedMediaTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method == "PUT" {
			puts++
			payload, _ := io.ReadAll(r.Body)
			if !bytes.Equal(payload, audioBytes) || !strings.Contains(r.URL.Path, "/assets/canvas/admin/") {
				t.Fatalf("media bytes or owner path changed: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "audio/mpeg" {
				t.Fatal("audio type lost")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		if r.Method != "GET" {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"audio/mpeg"}}, Body: io.NopCloser(bytes.NewReader(audioBytes))}, nil
	})
	stored, ok := persistGeneratedAudio("admin", "generated-audio", "audio/mpeg", audioBytes)
	if !ok || !strings.HasPrefix(stored.StorageKey, "server:") {
		t.Fatal("administrator output did not persist")
	}
	object, err := repository.GetStorageObject(stored.ID)
	sum := sha256.Sum256(audioBytes)
	if err != nil || object.CreatedBy != "admin" || object.SHA256 != hex.EncodeToString(sum[:]) || object.Bytes != int64(len(audioBytes)) {
		t.Fatalf("incorrect stored metadata: %+v %v", object, err)
	}
	if _, ok := persistGeneratedMedia("admin", "https://media.example.test/audio.mp3", "recovered", 1024); !ok {
		t.Fatal("recovered output did not persist")
	}
	for _, id := range []string{"user", "banned", "missing", ""} {
		if _, ok := persistGeneratedAudio(id, "unauthorized", "audio/mpeg", audioBytes); ok {
			t.Fatalf("unauthorized owner persisted: %q", id)
		}
	}
	if _, ok := persistGeneratedAudio("admin", "empty", "audio/mpeg", nil); ok {
		t.Fatal("empty audio persisted")
	}
	if puts != 2 {
		t.Fatalf("unauthorized requests reached storage: %d", puts)
	}
}
