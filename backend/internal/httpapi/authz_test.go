package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/auth"
	"backend/internal/config"
	inventory "backend/internal/inventory"
	ocr "backend/internal/ocr"
	"backend/internal/platform/logging"
	procurement "backend/internal/procurement"
)

func TestProtectedEndpointRequiresAuthentication(t *testing.T) {
	cfg := config.Config{
		App: config.AppConfig{Name: "inventory-manager-api", Env: "test", Mode: "cloud"},
		HTTP: config.HTTPConfig{
			AllowedOrigins: []string{"http://localhost:5173"},
		},
		Storage: config.StorageConfig{Mode: "local"},
		Auth: config.AuthConfig{
			Mode:     "enforced",
			RBAC:     "enforced",
			Verifier: "local",
		},
	}
	authService, err := auth.NewService(cfg.Auth, auth.NewRepository(nil))
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operator/dashboard", nil)
	rec := httptest.NewRecorder()

	router := NewRouter(cfg, logging.NewJSONLogger("debug"), authService, inventory.NewService(nil), procurement.NewService(nil, nil, nil, nil), ocr.NewService(nil, nil, nil, nil))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
