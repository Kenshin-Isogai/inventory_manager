package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/auth"
	"backend/internal/config"
	inventory "backend/internal/inventory"
	ocr "backend/internal/ocr"
	"backend/internal/platform/logging"
	procurement "backend/internal/procurement"
	"backend/internal/testutil"
)

func TestProtectedEndpointRequiresAuthentication(t *testing.T) {
	router := newAuthenticatedRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operator/dashboard", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRBACEnforcesRoleAssignments(t *testing.T) {
	router := newAuthenticatedRouter(t)

	tests := []struct {
		name       string
		token      string
		method     string
		target     string
		body       string
		wantStatus int
	}{
		{
			name:       "operator can read operator dashboard",
			token:      "local-operator-token",
			method:     http.MethodGet,
			target:     "/api/v1/operator/dashboard",
			wantStatus: http.StatusOK,
		},
		{
			name:       "inventory cannot read operator only dashboard",
			token:      "local-inventory-token",
			method:     http.MethodGet,
			target:     "/api/v1/operator/dashboard",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "inventory can read inventory overview",
			token:      "local-inventory-token",
			method:     http.MethodGet,
			target:     "/api/v1/inventory/overview",
			wantStatus: http.StatusOK,
		},
		{
			name:       "operator cannot read inventory overview",
			token:      "local-operator-token",
			method:     http.MethodGet,
			target:     "/api/v1/inventory/overview",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "inventory can read shared reservation list",
			token:      "local-inventory-token",
			method:     http.MethodGet,
			target:     "/api/v1/operator/reservations",
			wantStatus: http.StatusOK,
		},
		{
			name:       "procurement cannot read reservation list",
			token:      "local-procurement-token",
			method:     http.MethodGet,
			target:     "/api/v1/operator/reservations",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "procurement can read procurement requests",
			token:      "local-procurement-token",
			method:     http.MethodGet,
			target:     "/api/v1/procurement/requests",
			wantStatus: http.StatusOK,
		},
		{
			name:       "inventory cannot read procurement requests",
			token:      "local-inventory-token",
			method:     http.MethodGet,
			target:     "/api/v1/procurement/requests",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin can read admin master summary",
			token:      "local-admin-token",
			method:     http.MethodGet,
			target:     "/api/v1/admin/master-data",
			wantStatus: http.StatusOK,
		},
		{
			name:       "operator cannot read admin master summary",
			token:      "local-operator-token",
			method:     http.MethodGet,
			target:     "/api/v1/admin/master-data",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "inspector can access arrival calendar",
			token:      "local-inspector-token",
			method:     http.MethodGet,
			target:     "/api/v1/inventory/arrivals/calendar?yearMonth=2025-04",
			wantStatus: http.StatusOK,
		},
		{
			name:       "procurement cannot access arrival calendar",
			token:      "local-procurement-token",
			method:     http.MethodGet,
			target:     "/api/v1/inventory/arrivals/calendar?yearMonth=2025-04",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "inventory can use receipt endpoint",
			token:      "local-inventory-token",
			method:     http.MethodPost,
			target:     "/api/v1/inventory/receipts",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "inspector can use receipt endpoint",
			token:      "local-inspector-token",
			method:     http.MethodPost,
			target:     "/api/v1/inspector/receipts",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "procurement cannot use receipt endpoint",
			token:      "local-procurement-token",
			method:     http.MethodPost,
			target:     "/api/v1/inspector/receipts",
			body:       `{}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin bypasses procurement role checks",
			token:      "local-admin-token",
			method:     http.MethodGet,
			target:     "/api/v1/procurement/requests",
			wantStatus: http.StatusOK,
		},
		{
			name:       "pending user is rejected even with requested role",
			token:      "local-pending-token",
			method:     http.MethodGet,
			target:     "/api/v1/procurement/requests",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, bytes.NewBufferString(tc.body))
			req.Header.Set("Authorization", "Bearer "+tc.token)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("%s %s: expected %d, got %d", tc.method, tc.target, tc.wantStatus, rec.Code)
			}
		})
	}
}

func newAuthenticatedRouter(t *testing.T) http.Handler {
	t.Helper()

	db := testutil.SetupTestDB(t)
	repo := auth.NewRepository(db)
	ctx := context.Background()

	seedActiveLocalUser(t, ctx, repo, "admin@example.local", "Local Admin", "seed-admin", []string{"admin"})
	seedActiveLocalUser(t, ctx, repo, "operator@example.local", "Local Operator", "seed-operator", []string{"operator"})
	seedActiveLocalUser(t, ctx, repo, "inventory@example.local", "Local Inventory", "seed-inventory", []string{"inventory"})
	seedActiveLocalUser(t, ctx, repo, "procurement@example.local", "Local Procurement", "seed-procurement", []string{"procurement"})
	seedActiveLocalUser(t, ctx, repo, "inspector@example.local", "Local Inspector", "seed-inspector", []string{"receiving_inspector"})
	seedPendingLocalUser(t, ctx, repo, "pending@example.local", "Pending Procurement", "seed-pending", "procurement")

	cfg := config.Config{
		App: config.AppConfig{Name: "inventory-manager-api", Env: "test", Mode: "test"},
		HTTP: config.HTTPConfig{
			AllowedOrigins: []string{"http://localhost:5173"},
		},
		Storage: config.StorageConfig{Mode: "local"},
		Auth: config.AuthConfig{
			Mode:           "enforced",
			RBAC:           "enforced",
			Verifier:       "local",
			LocalTokenSpec: "local-pending-token=pending@example.local|Pending Procurement|procurement",
		},
	}
	authService, err := auth.NewService(cfg.Auth, repo)
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	phaseOne := inventory.NewService(inventory.NewRepository(db))
	phaseTwo := procurement.NewService(procurement.NewRepository(db), nil, nil, nil)
	phaseThree := ocr.NewService(ocr.NewRepository(db), nil, nil, phaseTwo)

	return NewRouter(cfg, logging.NewJSONLogger("debug"), authService, phaseOne, phaseTwo, phaseThree, nil)
}

func seedActiveLocalUser(t *testing.T, ctx context.Context, repo *auth.Repository, email, displayName, subject string, roles []string) {
	t.Helper()

	user, err := repo.UpsertPendingRegistration(ctx, "local", subject, email, displayName, "", "", "", "")
	if err != nil {
		t.Fatalf("seed pending user %s: %v", email, err)
	}
	if _, err := repo.UpdateUserStatus(ctx, user.ID, "active", "", roles); err != nil {
		t.Fatalf("activate user %s: %v", email, err)
	}
}

func seedPendingLocalUser(t *testing.T, ctx context.Context, repo *auth.Repository, email, displayName, subject, requestedRole string) {
	t.Helper()

	if _, err := repo.UpsertPendingRegistration(ctx, "local", subject, email, displayName, "", requestedRole, "", ""); err != nil {
		t.Fatalf("seed pending user %s: %v", email, err)
	}
}
