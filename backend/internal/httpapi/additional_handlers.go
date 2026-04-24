package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	inventory "backend/internal/inventory"
)

// ItemFlow returns chronological inventory events for a single item.
func (h Handlers) ItemFlow(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory", "operator") {
		return
	}
	data, err := h.phaseOne.ItemFlow(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ItemFlowList]{Data: data})
}

// ScopeOverview returns scope tree with summary counts.
func (h Handlers) ScopeOverview(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "inventory") {
		return
	}
	data, err := h.phaseOne.ScopeOverview(r.Context(), r.URL.Query().Get("device"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ScopeOverviewList]{Data: data})
}

// ShortageTimeline returns shortage broken down by scope start date timing.
func (h Handlers) ShortageTimeline(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
		return
	}
	data, err := h.phaseOne.ShortageTimeline(r.Context(), r.URL.Query().Get("device"), r.URL.Query().Get("scope"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ShortageTimeline]{Data: data})
}

// EnhancedShortages returns shortages with procurement pipeline info.
func (h Handlers) EnhancedShortages(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "inventory") {
		return
	}
	q := r.URL.Query()
	data, err := h.phaseOne.EnhancedShortages(r.Context(), q.Get("device"), q.Get("scope"), q.Get("coverageRule"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.EnhancedShortageList]{Data: data})
}

// ReservationsExportCSV returns reservations as a CSV download.
func (h Handlers) ReservationsExportCSV(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
		return
	}
	csv, err := h.phaseOne.ReservationsExportCSV(r.Context(), r.URL.Query().Get("device"), r.URL.Query().Get("scope"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=reservations.csv")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(csv))
}

// RequirementsExportCSV returns requirements as a CSV download.
func (h Handlers) RequirementsExportCSV(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
		return
	}
	csv, err := h.phaseOne.RequirementsExportCSV(r.Context(), r.URL.Query().Get("device"), r.URL.Query().Get("scope"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=requirements.csv")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(csv))
}

// RequirementsImportPreview previews a requirements CSV import.
func (h Handlers) RequirementsImportPreview(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
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
	body, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read uploaded file"})
		return
	}
	data, err := h.phaseOne.RequirementsImportPreview(r.Context(), header.Filename, body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.RequirementsImportPreview]{Data: data})
}

// RequirementsImportApply applies a requirements CSV import.
func (h Handlers) RequirementsImportApply(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
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
	body, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read uploaded file"})
		return
	}
	data, err := h.phaseOne.RequirementsImportApply(r.Context(), header.Filename, body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.RequirementsImportResult]{Data: data})
}

// BulkReservationPreview generates a preview of bulk reservations.
func (h Handlers) BulkReservationPreview(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
		return
	}
	scopeID := r.URL.Query().Get("scopeId")
	if scopeID == "" && r.Method == http.MethodPost {
		var input struct {
			ScopeID string `json:"scopeId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
			return
		}
		scopeID = input.ScopeID
	}
	data, err := h.phaseOne.BulkReservationPreview(r.Context(), scopeID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.BulkReservationPreview]{Data: data})
}

// BulkReservationConfirm creates reservations from confirmed preview.
func (h Handlers) BulkReservationConfirm(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
		return
	}
	var input inventory.BulkReservationConfirmInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.BulkReservationConfirm(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.BulkReservationResult]{Data: data})
}

// ArrivalCalendar returns expected arrivals grouped by date.
func (h Handlers) ArrivalCalendar(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory", "operator", "receiving_inspector") {
		return
	}
	ym := r.URL.Query().Get("yearMonth")
	data, err := h.phaseOne.ArrivalCalendar(r.Context(), ym)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ArrivalCalendar]{Data: data})
}

// ItemSuggest returns items matching a search query for typeahead.
func (h Handlers) ItemSuggest(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "operator", "inventory") {
		return
	}
	data, err := h.phaseOne.ItemSuggest(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ItemSuggestionList]{Data: data})
}

// CategorySuggest returns categories matching a search query for typeahead.
func (h Handlers) CategorySuggest(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "admin", "operator", "inventory") {
		return
	}
	data, err := h.phaseOne.CategorySuggest(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.CategorySuggestionList]{Data: data})
}

// InventorySnapshotAtDate extends InventorySnapshot with target_date support.
func (h Handlers) InventorySnapshotAtDate(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory", "operator") {
		return
	}
	q := r.URL.Query()
	data, err := h.phaseOne.InventorySnapshotAtDate(r.Context(), q.Get("device"), q.Get("scope"), q.Get("itemId"), q.Get("target_date"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.InventorySnapshot]{Data: data})
}
