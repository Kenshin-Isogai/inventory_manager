package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App         AppConfig
	HTTP        HTTPConfig
	Database    DatabaseConfig
	Storage     StorageConfig
	Auth        AuthConfig
	Integration IntegrationConfig
	Logging     LoggingConfig
	OCR         OCRConfig
}

type AppConfig struct {
	Name string
	Env  string
	Mode string
}

type HTTPConfig struct {
	Port           string
	BaseURL        string
	AllowedOrigins []string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type StorageConfig struct {
	Mode       string
	Artifacts  string
	BucketName string
}

type AuthConfig struct {
	Mode                 string
	RBAC                 string
	Verifier             string
	LocalTokenSpec       string
	ExpectedIssuer       string
	ExpectedAudience     string
	JWKSURL              string
	SigningAlgorithms    []string
	RequireEmailVerified bool
}

type IntegrationConfig struct {
	ProcurementWebhookSecret string
}

type LoggingConfig struct {
	Level string
}

type OCRConfig struct {
	Provider                     string
	GoogleCloudProject           string
	VertexAILocation             string
	GeminiModel                  string
	GoogleApplicationCredentials string
	StorageBucket                string
}

func Load() (Config, error) {
	loadDotEnvFiles(".env", ".env.local")

	cfg := Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "inventory-manager-api"),
			Env:  getEnv("APP_ENV", "local"),
			Mode: getEnv("APP_MODE", "local"),
		},
		HTTP: HTTPConfig{
			Port:           getEnv("HTTP_PORT", "8080"),
			BaseURL:        getEnv("APP_BASE_URL", "http://localhost:8080"),
			AllowedOrigins: splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:5173")),
			ReadTimeout:    getDurationEnv("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:   getDurationEnv("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:    getDurationEnv("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Storage: StorageConfig{
			Mode:       getEnv("STORAGE_MODE", "local"),
			Artifacts:  getEnv("ARTIFACTS_DIR", "./artifacts"),
			BucketName: getEnv("CLOUD_STORAGE_BUCKET", ""),
		},
		Auth: AuthConfig{
			Mode:                 normalizeAuthMode(getEnv("AUTH_MODE", "none")),
			RBAC:                 normalizeRBACMode(getEnv("RBAC_MODE", "dry_run")),
			Verifier:             getEnv("JWT_VERIFIER", getEnv("AUTH_PROVIDER", "local")),
			LocalTokenSpec:       getEnv("LOCAL_AUTH_TOKENS", ""),
			ExpectedIssuer:       getEnv("OIDC_EXPECTED_ISSUER", getEnv("JWT_ISSUER", "")),
			ExpectedAudience:     getEnv("OIDC_EXPECTED_AUDIENCE", getEnv("JWT_AUDIENCE", "")),
			JWKSURL:              getEnv("OIDC_JWKS_URL", getEnv("JWKS_URL", "")),
			SigningAlgorithms:    splitCSV(getEnv("JWT_SIGNING_ALGORITHMS", "RS256")),
			RequireEmailVerified: getBoolEnv("OIDC_REQUIRE_EMAIL_VERIFIED", false),
		},
		Integration: IntegrationConfig{
			ProcurementWebhookSecret: getEnv("PROCUREMENT_WEBHOOK_SECRET", ""),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		OCR: OCRConfig{
			Provider:                     getEnv("OCR_PROVIDER", "mock"),
			GoogleCloudProject:           getEnv("GOOGLE_CLOUD_PROJECT", ""),
			VertexAILocation:             getEnv("VERTEX_AI_LOCATION", "asia-northeast1"),
			GeminiModel:                  getEnv("GEMINI_MODEL", "gemini-3-flash-preview"),
			GoogleApplicationCredentials: normalizeOptionalEnv(getEnv("GOOGLE_APPLICATION_CREDENTIALS", "")),
			StorageBucket:                getEnv("OCR_STORAGE_BUCKET", ""),
		},
	}

	if cfg.Database.URL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func loadDotEnvFiles(names ...string) {
	for _, name := range names {
		loadDotEnvFile(name)
	}
}

func loadDotEnvFile(name string) {
	path := filepath.Clean(name)
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(getEnv(key, "")))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	case "":
		return fallback
	default:
		return fallback
	}
}

func normalizeAuthMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "oidc_dry_run":
		return "dry_run"
	case "oidc_enforced":
		return "enforced"
	default:
		return strings.TrimSpace(strings.ToLower(value))
	}
}

func normalizeRBACMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "rbac_dry_run":
		return "dry_run"
	case "rbac_enforced":
		return "enforced"
	default:
		return strings.TrimSpace(strings.ToLower(value))
	}
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalizeOptionalEnv(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasPrefix(trimmed, "__FILL_ME") {
		return ""
	}
	return trimmed
}
