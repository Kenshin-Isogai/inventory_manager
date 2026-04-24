package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	inventory "backend/internal/inventory"
)

func (h Handlers) Requirements(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "inventory") {
		return
	}
	data, err := h.phaseOne.Requirements(r.Context(), r.URL.Query().Get("device"), r.URL.Query().Get("scope"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.RequirementList]{Data: data})
}

func (h Handlers) UpsertRequirement(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
		return
	}
	var input inventory.RequirementUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UpsertRequirement(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.RequirementSummary]{Data: data})
}

func (h Handlers) BatchUpsertRequirements(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator") {
		return
	}
	var input inventory.RequirementBatchUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.BatchUpsertRequirements(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.RequirementBatchUpsertResult]{Data: data})
}

func (h Handlers) ReservationDetail(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "inventory") {
		return
	}
	data, err := h.phaseOne.ReservationDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ReservationDetail]{Data: data})
}

func (h Handlers) UpdateReservation(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "inventory") {
		return
	}
	var input inventory.ReservationUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UpdateReservation(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ReservationDetail]{Data: data})
}

func (h Handlers) DeleteReservation(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "inventory") {
		return
	}
	var payload map[string]string
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	if err := h.phaseOne.DeleteReservation(r.Context(), r.PathValue("id"), payload["actorId"]); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h Handlers) AllocateReservation(w http.ResponseWriter, r *http.Request) {
	h.handleReservationAction(w, r, h.phaseOne.AllocateReservation)
}

func (h Handlers) ReleaseReservation(w http.ResponseWriter, r *http.Request) {
	h.handleReservationAction(w, r, h.phaseOne.ReleaseReservation)
}

func (h Handlers) FulfillReservation(w http.ResponseWriter, r *http.Request) {
	h.handleReservationAction(w, r, h.phaseOne.FulfillReservation)
}

func (h Handlers) CancelReservation(w http.ResponseWriter, r *http.Request) {
	h.handleReservationAction(w, r, h.phaseOne.CancelReservation)
}

func (h Handlers) UndoReservation(w http.ResponseWriter, r *http.Request) {
	h.handleReservationAction(w, r, h.phaseOne.UndoReservation)
}

func (h Handlers) InventoryItems(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory", "operator") {
		return
	}
	data, err := h.phaseOne.InventoryItems(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.InventoryItemList]{Data: data})
}

func (h Handlers) InventoryLocations(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory") {
		return
	}
	data, err := h.phaseOne.InventoryLocations(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.LocationList]{Data: data})
}

func (h Handlers) InventoryEvents(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory") {
		return
	}
	data, err := h.phaseOne.InventoryEvents(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.InventoryEventList]{Data: data})
}

func (h Handlers) InventorySnapshot(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory", "operator") {
		return
	}
	data, err := h.phaseOne.InventorySnapshot(r.Context(), r.URL.Query().Get("device"), r.URL.Query().Get("scope"), r.URL.Query().Get("itemId"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.InventorySnapshot]{Data: data})
}

func (h Handlers) ReceiveInventory(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory", "receiving_inspector") {
		return
	}
	var input inventory.InventoryReceiveInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.ReceiveInventory(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.InventoryEventEntry]{Data: data})
}

func (h Handlers) MoveInventory(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory") {
		return
	}
	var input inventory.InventoryMoveInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.MoveInventory(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.InventoryEventEntry]{Data: data})
}

func (h Handlers) UndoInventoryEvent(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "inventory") {
		return
	}
	var input inventory.InventoryUndoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UndoInventoryEvent(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.InventoryEventEntry]{Data: data})
}

func (h Handlers) Arrivals(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "receiving_inspector", "inventory") {
		return
	}
	data, err := h.phaseOne.Arrivals(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ArrivalList]{Data: data})
}

func (h Handlers) CreateReceipt(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "receiving_inspector", "inventory") {
		return
	}
	var input inventory.ReceiptCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.CreateReceipt(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[inventory.ReceiptSummary]{Data: data})
}

func (h Handlers) ImportPreview(w http.ResponseWriter, r *http.Request) {
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
	body, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read uploaded file"})
		return
	}
	data, err := h.phaseOne.ImportPreview(r.Context(), r.URL.Query().Get("type"), header.Filename, body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ImportPreviewResult]{Data: data})
}

func (h Handlers) ImportDetail(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "admin") {
		return
	}
	data, err := h.phaseOne.ImportDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ImportDetail]{Data: data})
}

func (h Handlers) UndoImport(w http.ResponseWriter, r *http.Request) {
	if !h.requireActiveRole(w, r, "operator", "admin") {
		return
	}
	var payload map[string]string
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := h.phaseOne.UndoImport(r.Context(), r.PathValue("id"), payload["actorId"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ImportDetail]{Data: data})
}

func (h Handlers) handleReservationAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(rctx context.Context, id string, input inventory.ReservationActionInput) (inventory.ReservationDetail, error),
) {
	if !h.requireActiveRole(w, r, "operator", "inventory") {
		return
	}
	var input inventory.ReservationActionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	data, err := action(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[inventory.ReservationDetail]{Data: data})
}
