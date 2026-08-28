package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeDockerSQLiteDSNUsesMountedDataDir(t *testing.T) {
	root := t.TempDir()
	appDataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	Cfg = Config{StorageDriver: "sqlite", DatabaseDSN: "data/infinite-canvas.db?_pragma=busy_timeout(5000)"}

	normalizeDockerSQLiteDSN(appDataDir)

	want := filepath.Join(root, "data", "infinite-canvas.db") + "?_pragma=busy_timeout(5000)"
	if Cfg.DatabaseDSN != want {
		t.Fatalf("DatabaseDSN = %q, want %q", Cfg.DatabaseDSN, want)
	}
}

func TestNormalizeDockerSQLiteDSNLeavesLocalPathWithoutMountedDataDir(t *testing.T) {
	Cfg = Config{StorageDriver: "sqlite", DatabaseDSN: "data/infinite-canvas.db"}

	normalizeDockerSQLiteDSN(filepath.Join(t.TempDir(), "missing-data"))

	if Cfg.DatabaseDSN != "data/infinite-canvas.db" {
		t.Fatalf("DatabaseDSN = %q, want relative local path", Cfg.DatabaseDSN)
	}
}

func TestDatabasePersistentRejectsRenderEphemeralSQLite(t *testing.T) {
	t.Setenv("RENDER_SERVICE_ID", "srv-test")
	t.Setenv("PERSISTENT_DISK_PATH", filepath.Join(t.TempDir(), "persistent"))
	Cfg = Config{StorageDriver: "sqlite", DatabaseDSN: "data/infinite-canvas.db"}
	if DatabasePersistent() {
		t.Fatal("relative Render SQLite path must be treated as ephemeral")
	}
}

func TestDatabasePersistentAcceptsMountedRenderSQLite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "persistent")
	t.Setenv("RENDER_SERVICE_ID", "srv-test")
	t.Setenv("PERSISTENT_DISK_PATH", root)
	Cfg = Config{StorageDriver: "sqlite", DatabaseDSN: filepath.Join(root, "infinite-canvas.db")}
	if !DatabasePersistent() {
		t.Fatal("SQLite under the configured persistent disk must be treated as durable")
	}
}

func TestPaymentConfiguredRejectsLegacyEpay(t *testing.T) {
	Cfg = Config{EpayAPIURL: "https://pay.example.com", EpayMerchantID: "merchant", EpayMerchantKey: "secret", PublicBaseURL: "https://canvas.example.com"}
	if PaymentConfigured() {
		t.Fatal("legacy Epay variables must not enable the official Alipay wallet")
	}
}

func TestPaymentConfiguredAcceptsOfficialAlipay(t *testing.T) {
	Cfg = Config{AlipayAppID: "2021000000000000", AlipayAppPrivateKey: "private", AlipayPublicKey: "public", AlipayGatewayURL: "https://openapi.alipay.com/gateway.do", AlipayPaymentEnabled: true, PublicBaseURL: "https://canvas.example.com"}
	if !PaymentConfigured() { t.Fatal("official Alipay configuration should be ready") }
	Cfg.AlipayPublicKey = ""
	if PaymentConfigured() { t.Fatal("Alipay without public key must not be ready") }
}
