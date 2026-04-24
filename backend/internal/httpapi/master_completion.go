package httpapi

import (
	"encoding/json"
	"net/http"

	inventory "backend/internal/inventory"
)

func (h Handlers) MasterItems(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	data, err := h.phaseOne.MasterItems(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.MasterItemList]{Data: data})
}

func (h Handlers) MasterItemDetail(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	data, err := h.phaseOne.MasterItemDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.MasterItemRecord]{Data: data})
}

func (h Handlers) UpsertMasterItem(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	var input inventory.MasterItemUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UpsertMasterItem(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.MasterItemRecord]{Data: data})
}

func (h Handlers) DeleteMasterItem(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	if err := h.phaseOne.DeleteMasterItem(r.Context(), r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h Handlers) Suppliers(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "procurement") {
		return
	}
	data, err := h.phaseOne.Suppliers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.SupplierList]{Data: data})
}

func (h Handlers) UpsertSupplier(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "procurement") {
		return
	}
	var input inventory.SupplierUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UpsertSupplier(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.SupplierRecord]{Data: data})
}

func (h Handlers) Aliases(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "procurement") {
		return
	}
	data, err := h.phaseOne.Aliases(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.AliasList]{Data: data})
}

func (h Handlers) UpsertAlias(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "procurement") {
		return
	}
	var input inventory.AliasUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UpsertAlias(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.SupplierAliasSummary]{Data: data})
}

func (h Handlers) Devices(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "operator") {
		return
	}
	data, err := h.phaseOne.Devices(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.DeviceList]{Data: data})
}

func (h Handlers) UpsertDevice(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	var input inventory.DeviceUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UpsertDevice(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.DeviceRecord]{Data: data})
}

func (h Handlers) DeviceScopes(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "operator") {
		return
	}
	data, err := h.phaseOne.DeviceScopes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.DeviceScopeList]{Data: data})
}

func (h Handlers) UpsertDeviceScope(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "operator") {
		return
	}
	var input inventory.DeviceScopeUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UpsertDeviceScope(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.DeviceScopeRecord]{Data: data})
}

func (h Handlers) ScopeSystems(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "operator") {
		return
	}
	data, err := h.phaseOne.ScopeSystems(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ScopeSystemList]{Data: data})
}

func (h Handlers) UpsertScopeSystem(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	var input inventory.ScopeSystemUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UpsertScopeSystem(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.ScopeSystemRecord]{Data: data})
}

func (h Handlers) DeleteScopeSystem(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	if err := h.phaseOne.DeleteScopeSystem(r.Context(), r.PathValue("key")); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h Handlers) UpsertLocation(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "inventory") {
		return
	}
	var input inventory.LocationUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UpsertLocation(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.LocationSummary]{Data: data})
}
