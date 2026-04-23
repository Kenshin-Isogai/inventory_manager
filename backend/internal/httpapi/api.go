package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"backend/internal/auth"
	"backend/internal/config"
	inventory "backend/internal/inventory"
	ocr "backend/internal/ocr"
	procurement "backend/internal/procurement"
)

type ServiceStatus struct {
	FrontendBaseURL string   `json:"frontendBaseUrl"`
	AuthMode        string   `json:"authMode"`
	AuthProvider    string   `json:"authProvider"`
	RBACMode        string   `json:"rbacMode"`
	StorageMode     string   `json:"storageMode"`
	Capabilities    []string `json:"capabilities"`
}

type Handlers struct {
	cfg        config.Config
	logger     *slog.Logger
	auth       *auth.Service
	phaseOne   *inventory.Service
	phaseTwo   *procurement.Service
	phaseThree *ocr.Service
}

func NewHandlers(cfg config.Config, logger *slog.Logger, authService *auth.Service, phaseOne *inventory.Service, phaseTwo *procurement.Service, phaseThree *ocr.Service) Handlers {
	return Handlers{cfg: cfg, logger: logger, auth: authService, phaseOne: phaseOne, phaseTwo: phaseTwo, phaseThree: phaseThree}
}

func (h Handlers) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": h.cfg.App.Name,
	})
}

func (h Handlers) Ready(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
		"mode":   h.cfg.App.Mode,
	})
}

func (h Handlers) APIHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h Handlers) Bootstrap(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, ServiceStatus{
		FrontendBaseURL: h.cfg.HTTP.AllowedOrigins[0],
		AuthMode:        h.cfg.Auth.Mode,
		AuthProvider:    h.cfg.Auth.Verifier,
		RBACMode:        h.cfg.Auth.RBAC,
		StorageMode:     h.cfg.Storage.Mode,
		Capabilities: []string{
			"local-mode",
			"mock-read-models",
			"migration-runner",
			"cloud-storage-extension-point",
			"auth-session",
			"user-approval-flow",
		},
	})
}

func (h Handlers) requireAuthenticated(w http.ResponseWriter, r *http.Request) (auth.Principal, bool) {
	principal := auth.PrincipalFromContext(r.Context())
	if h.auth == nil || h.cfg.Auth.Mode != "enforced" {
		return principal, true
	}
	if !principal.Authenticated {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return principal, false
	}
	return principal, true
}

func (h Handlers) requireVerifiedIdentity(w http.ResponseWriter, principal auth.Principal) bool {
	if h.auth == nil || h.cfg.Auth.Mode != "enforced" || !h.cfg.Auth.RequireEmailVerified {
		return true
	}
	if !principal.EmailVerified {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "email verification is required"})
		return false
	}
	return true
}

func (h Handlers) requireActiveRole(w http.ResponseWriter, r *http.Request, roles ...string) bool {
	principal, ok := h.requireAuthenticated(w, r)
	if !ok {
		return false
	}
	if h.auth == nil || h.cfg.Auth.Mode != "enforced" {
		return true
	}
	if principal.Status != "active" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "active account required"})
		return false
	}
	if h.cfg.Auth.RBAC != "enforced" {
		return true
	}
	for _, role := range roles {
		if auth.Allowed(principal, role) {
			return true
		}
	}
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	return false
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
