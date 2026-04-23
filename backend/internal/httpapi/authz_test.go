package httpapi

import (
	"bytes"
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

	router := NewRouter(cfg, logging.NewJSONLogger("debug"), authService, inventory.NewService(nil), procurement.NewService(nil, nil, nil, nil), ocr.NewService(nil, nil, nil, nil), nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAdditionalProtectedRoutesRequireAuthentication(t *testing.T) {
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

	router := NewRouter(cfg, logging.NewJSONLogger("debug"), authService, inventory.NewService(nil), procurement.NewService(nil, nil, nil, nil), ocr.NewService(nil, nil, nil, nil), nil)
	tests := []struct {
		method string
		target string
		body   string
	}{
		{method: http.MethodGet, target: "/api/v1/inventory/snapshot"},
		{method: http.MethodDelete, target: "/api/v1/operator/reservations/res-1", body: `{}`},
		{method: http.MethodGet, target: "/api/v1/procurement/orders"},
		{method: http.MethodPost, target: "/api/v1/procurement/orders", body: `{"procurementBatchId":"batch-1","lines":[{"procurementLineId":"line-1","orderedQuantity":1}]}`},
		{method: http.MethodDelete, target: "/api/v1/admin/master-data/items/item-1", body: `{}`},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.target, bytes.NewBufferString(tc.body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d", tc.method, tc.target, rec.Code)
		}
	}
}
