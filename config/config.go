package config

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Port                string `env:"PORT" envDefault:"8080"`
	AdminUsername       string `env:"ADMIN_USERNAME" envDefault:"admin"`
	AdminPassword       string `env:"ADMIN_PASSWORD" envDefault:"infinite-canvas"`
	JWTSecret           string `env:"JWT_SECRET" envDefault:"infinite-canvas"`
	JWTExpireHours      int    `env:"JWT_EXPIRE_HOURS" envDefault:"168"`
	StorageDriver       string `env:"STORAGE_DRIVER" envDefault:"sqlite"`
	DatabaseDSN         string `env:"DATABASE_DSN" envDefault:"data/infinite-canvas.db"`
	PublicBaseURL       string `env:"PUBLIC_BASE_URL"`
	LinuxDoAuthorizeURL string `env:"LINUX_DO_AUTHORIZE_URL" envDefault:"https://connect.linux.do/oauth2/authorize"`
	LinuxDoTokenURL     string `env:"LINUX_DO_TOKEN_URL" envDefault:"https://connect.linux.do/oauth2/token"`
	LinuxDoUserInfoURL  string `env:"LINUX_DO_USERINFO_URL" envDefault:"https://connect.linux.do/api/user"`
	AILogDir            string `env:"AI_LOG_DIR" envDefault:"data/logs/ai-calls"`
	EpayAPIURL          string `env:"EPAY_API_URL"`
	EpayMerchantID      string `env:"EPAY_MERCHANT_ID"`
	EpayMerchantKey     string `env:"EPAY_MERCHANT_KEY"`
	AlipayAppID          string `env:"ALIPAY_APP_ID"`
	AlipayAppPrivateKey  string `env:"ALIPAY_APP_PRIVATE_KEY"`
	AlipayPublicKey      string `env:"ALIPAY_PUBLIC_KEY"`
	AlipayGatewayURL     string `env:"ALIPAY_GATEWAY_URL" envDefault:"https://openapi.alipay.com/gateway.do"`
	AlipayPaymentEnabled bool   `env:"ALIPAY_PAYMENT_ENABLED" envDefault:"false"`
	ManagedPlatformMode bool   `env:"MANAGED_PLATFORM_MODE" envDefault:"true"`
	GenerationRPM      int    `env:"GENERATION_REQUESTS_PER_MINUTE" envDefault:"30"`
}

var Cfg Config

func Load() error {
	_ = godotenv.Load()
	if err := env.Parse(&Cfg); err != nil {
		return err
	}
	normalizeDockerSQLiteDSN("/app/data")
	if strings.TrimSpace(Cfg.JWTSecret) == "" || Cfg.JWTSecret == "infinite-canvas" {
		secret, err := persistentJWTSecret()
		if err != nil {
			return err
		}
		Cfg.JWTSecret = secret
	}
	return nil
}

// persistentJWTSecret keeps issued login tokens valid across service restarts.
// Production deployments should still prefer setting JWT_SECRET explicitly.
func persistentJWTSecret() (string, error) {
	secretPath := filepath.Join("data", ".jwt-secret")
	dsn := strings.TrimSpace(Cfg.DatabaseDSN)
	if dsn != "" && dsn != ":memory:" && !strings.HasPrefix(dsn, "file:") {
		pathPart := dsn
		if index := strings.Index(pathPart, "?"); index >= 0 {
			pathPart = pathPart[:index]
		}
		if dir := filepath.Dir(pathPart); dir != "." && dir != "" {
			secretPath = filepath.Join(dir, ".jwt-secret")
		}
	}
	if data, err := os.ReadFile(secretPath); err == nil {
		if secret := strings.TrimSpace(string(data)); secret != "" {
			return secret, nil
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	secret, err := randomSecret()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(secretPath), 0700); err != nil {
		return "", err
	}
	if err := os.WriteFile(secretPath, []byte(secret+"\n"), 0600); err != nil {
		return "", err
	}
	return secret, nil
}

func normalizeDockerSQLiteDSN(appDataDir string) {
	driver := strings.ToLower(strings.TrimSpace(Cfg.StorageDriver))
	if driver != "" && driver != "sqlite" {
		return
	}
	dsn := strings.TrimSpace(Cfg.DatabaseDSN)
	if dsn == "" || dsn == ":memory:" || strings.HasPrefix(dsn, "file:") {
		return
	}
	pathPart, suffix := dsn, ""
	if index := strings.Index(dsn, "?"); index >= 0 {
		pathPart = dsn[:index]
		suffix = dsn[index:]
	}
	if filepath.IsAbs(pathPart) {
		return
	}
	slashPath := filepath.ToSlash(pathPart)
	if slashPath != "data" && !strings.HasPrefix(slashPath, "data/") {
		return
	}
	if _, err := os.Stat(appDataDir); err != nil {
		return
	}
	Cfg.DatabaseDSN = filepath.Join(filepath.Dir(appDataDir), filepath.FromSlash(slashPath)) + suffix
}

func randomSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// PaymentConfigured reports whether the server has all credentials required to
// create and verify an online payment order. It never exposes those credentials.
func PaymentConfigured() bool {
	if AlipayConfigured() {
		return true
	}
	return strings.TrimSpace(Cfg.EpayAPIURL) != "" &&
		strings.TrimSpace(Cfg.EpayMerchantID) != "" &&
		strings.TrimSpace(Cfg.EpayMerchantKey) != "" &&
		strings.TrimSpace(Cfg.PublicBaseURL) != ""
}

// AlipayConfigured reports whether official Alipay RSA2 payment can be used.
func AlipayConfigured() bool {
	return Cfg.AlipayPaymentEnabled && AlipayCredentialsConfigured()
}

func AlipayCredentialsConfigured() bool {
	return strings.TrimSpace(Cfg.AlipayAppID) != "" &&
		strings.TrimSpace(Cfg.AlipayAppPrivateKey) != "" &&
		strings.TrimSpace(Cfg.AlipayPublicKey) != "" &&
		strings.TrimSpace(Cfg.AlipayGatewayURL) != "" &&
		strings.TrimSpace(Cfg.PublicBaseURL) != ""
}

// DatabasePersistent is deliberately conservative on Render: a SQLite file is
// considered durable only when it is placed under a mounted persistent-disk
// path. Managed databases are durable by definition.
func DatabasePersistent() bool {
	driver := strings.ToLower(strings.TrimSpace(Cfg.StorageDriver))
	if driver != "" && driver != "sqlite" {
		return true
	}
	if strings.TrimSpace(os.Getenv("RENDER_SERVICE_ID")) == "" {
		return true
	}
	dsn := strings.TrimSpace(Cfg.DatabaseDSN)
	if index := strings.Index(dsn, "?"); index >= 0 {
		dsn = dsn[:index]
	}
	if !filepath.IsAbs(dsn) {
		return false
	}
	root := strings.TrimSpace(os.Getenv("PERSISTENT_DISK_PATH"))
	if root == "" {
		root = "/var/data"
	}
	path, err := filepath.Abs(dsn)
	if err != nil {
		return false
	}
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootPath, path)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
