package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	procurement "backend/internal/procurement"
)

func (h Handlers) ProcurementProjects(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	rows, err := h.phaseTwo.Projects(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[[]procurement.ProjectSummary]{Data: rows})
}

func (h Handlers) RefreshProcurementProjects(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	result, err := h.phaseTwo.RefreshProjects(r.Context(), "manual")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[procurement.MasterSyncResult]{Data: result})
}

func (h Handlers) ProcurementBudgetCategories(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	rows, err := h.phaseTwo.BudgetCategories(r.Context(), r.URL.Query().Get("projectId"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[[]procurement.BudgetCategorySummary]{Data: rows})
}

func (h Handlers) RefreshProcurementBudgetCategories(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	result, err := h.phaseTwo.RefreshBudgetCategories(r.Context(), r.URL.Query().Get("projectId"), "manual")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[procurement.MasterSyncResult]{Data: result})
}

func (h Handlers) ProcurementSyncRuns(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	rows, err := h.phaseTwo.MasterSyncRuns(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[[]procurement.MasterSyncRunEntry]{Data: rows})
}

func (h Handlers) ProcurementWebhookEvents(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	rows, err := h.phaseTwo.WebhookEvents(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[[]procurement.WebhookEventEntry]{Data: rows})
}

func (h Handlers) ProcurementRequests(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	rows, err := h.phaseTwo.Requests(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[procurement.ProcurementRequestList]{Data: rows})
}

func (h Handlers) ProcurementRequestDetail(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/procurement/requests/")
	detail, err := h.phaseTwo.RequestDetail(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[procurement.ProcurementRequestDetail]{Data: detail})
}

func (h Handlers) CreateProcurementRequest(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	var input procurement.ProcurementRequestCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	id, err := h.phaseTwo.CreateRequest(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "status": "created"})
}

func (h Handlers) ReconcileProcurementRequest(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	id := r.PathValue("id")
	result, err := h.phaseTwo.ReconcileRequest(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[procurement.ProcurementReconcileResult]{Data: result})
}

func (h Handlers) SubmitProcurementRequest(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "procurement") {
		return
	}
	id := r.PathValue("id")
	result, err := h.phaseTwo.SubmitRequest(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[procurement.ProcurementSubmitResult]{Data: result})
}

func (h Handlers) ProcurementWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read request body"})
		return
	}
	result, err := h.phaseTwo.HandleWebhook(r.Context(), r.Header, body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[procurement.WebhookProcessResult]{Data: result})
}
