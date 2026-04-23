package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/config"
	"backend/internal/httpapi"
	inventory "backend/internal/inventory"
	ocr "backend/internal/ocr"
	"backend/internal/platform/logging"
	procurement "backend/internal/procurement"
)

func TestHealthEndpoint(t *testing.T) {
	cfg := config.Config{
		App: config.AppConfig{Name: "inventory-manager-api", Env: "test", Mode: "local"},
		HTTP: config.HTTPConfig{
			AllowedOrigins: []string{"http://localhost:5173"},
		},
		Storage: config.StorageConfig{Mode: "local"},
		Auth:    config.AuthConfig{Mode: "none", RBAC: "dry_run"},
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router := httpapi.NewRouter(cfg, logging.NewJSONLogger("debug"), nil, inventory.NewService(nil), procurement.NewService(nil, nil, nil, nil), ocr.NewService(nil, nil, nil, nil))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
