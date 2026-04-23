package httpapi

import (
	"encoding/json"
	"net/http"

	"backend/internal/auth"
)

func (h Handlers) BootstrapRegisterUser(w http.ResponseWriter, r *http.Request) {
	var input auth.RegistrationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	result, err := h.auth.BootstrapRegister(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[auth.UserSummary]{Data: result})
}

func (h Handlers) Permissions(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	rows, err := h.auth.Permissions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[[]auth.PermissionSummary]{Data: rows})
}

func (h Handlers) UpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	var input auth.RolePermissionUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	result, err := h.auth.UpdateRolePermissions(r.Context(), r.PathValue("key"), input.Permissions)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[auth.RoleSummary]{Data: result})
}

func (h Handlers) UserStatusHistory(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	rows, err := h.auth.UserStatusHistory(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[[]auth.UserStatusHistoryEntry]{Data: rows})
}
