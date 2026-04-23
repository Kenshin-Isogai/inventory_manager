package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	inventory "backend/internal/inventory"
)

func (h Handlers) OperatorDashboard(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
		return
	}
	data, err := h.phaseOne.Dashboard(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.DashboardData]{Data: data})
}

func (h Handlers) ReservationList(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "inventory") {
		return
	}
	data, err := h.phaseOne.Reservations(r.Context(), r.URL.Query().Get("device"), r.URL.Query().Get("scope"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ReservationList]{Data: data})
}

func (h Handlers) InventoryOverview(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory") {
		return
	}
	data, err := h.phaseOne.InventoryOverview(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.InventoryOverview]{Data: data})
}

func (h Handlers) ShortageList(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "inventory") {
		return
	}
	data, err := h.phaseOne.Shortages(r.Context(), r.URL.Query().Get("device"), r.URL.Query().Get("scope"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ShortageList]{Data: data})
}

func (h Handlers) ShortageCSVExport(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "inventory") {
		return
	}
	data, err := h.phaseOne.ShortageCSV(r.Context(), r.URL.Query().Get("device"), r.URL.Query().Get("scope"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"shortages.csv\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(data))
}

func (h Handlers) ImportHistory(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
		return
	}
	data, err := h.phaseOne.Imports(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ImportHistory]{Data: data})
}

func (h Handlers) MasterSummary(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	data, err := h.phaseOne.MasterSummary(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.MasterDataSummary]{Data: data})
}

func (h Handlers) MasterDataExport(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin") {
		return
	}
	exportType := r.URL.Query().Get("type")
	data, err := h.phaseOne.ExportMasterCSV(r.Context(), exportType)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.csv\"", exportType))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(data))
}

func (h Handlers) MasterDataImport(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "operator") {
		return
	}
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse multipart form"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file field is required"})
		return
	}
	defer file.Close()

	job, err := h.phaseOne.ImportMasterCSV(r.Context(), r.URL.Query().Get("type"), header.Filename, file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error(), "jobId": job.ID})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.ImportJob]{Data: job})
}

func (h Handlers) CreateReservation(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
		return
	}
	var input inventory.ReservationCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	if err := h.phaseOne.CreateReservation(r.Context(), input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

func (h Handlers) AdjustInventory(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory") {
		return
	}
	var input inventory.InventoryAdjustInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	if err := h.phaseOne.AdjustInventory(r.Context(), input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}
