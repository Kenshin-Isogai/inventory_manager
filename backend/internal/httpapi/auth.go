package httpapi

import (
	"encoding/json"
	"net/http"

	"backend/internal/auth"
)

func (h Handlers) CurrentSession(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	writeJSON(w, http.StatusOK, APIEnvelope[auth.SessionResponse]{Data: h.auth.Session(r.Context(), principal)})
}

func (h Handlers) RegisterUser(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	var input auth.RegistrationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	result, err := h.auth.Register(r.Context(), principal, input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[auth.UserSummary]{Data: result})
}

func (h Handlers) Users(w http.ResponseWriter, r *http.Request) {
	if !h.requireRole(w, r, "admin") {
		return
	}
	rows, err := h.auth.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[[]auth.UserSummary]{Data: rows})
}

func (h Handlers) Roles(w http.ResponseWriter, r *http.Request) {
	if !h.requireRole(w, r, "admin") {
		return
	}
	rows, err := h.auth.ListRoles(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[[]auth.RoleSummary]{Data: rows})
}

func (h Handlers) ApproveUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireRole(w, r, "admin") {
		return
	}
	var input auth.ApproveUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	result, err := h.auth.ApproveUser(r.Context(), r.PathValue("id"), input.Roles)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[auth.UserSummary]{Data: result})
}

func (h Handlers) RejectUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireRole(w, r, "admin") {
		return
	}
	var input auth.RejectUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	result, err := h.auth.RejectUser(r.Context(), r.PathValue("id"), input.Reason)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[auth.UserSummary]{Data: result})
}
