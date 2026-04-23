package httpapi

import (
	"encoding/json"
	"mime"
	"net/http"
	"path/filepath"

	ocr "backend/internal/ocr"
)

func (h Handlers) OCRJobList(w http.ResponseWriter, r *http.Request) {
	data, err := h.phaseThree.Jobs(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[ocr.OCRJobList]{Data: data})
}

func (h Handlers) CreateOCRJob(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse multipart form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file field is required"})
		return
	}
	defer file.Close()

	result, err := h.phaseThree.CreateJob(r.Context(), header.Filename, inferContentType(header.Filename, header.Header.Get("Content-Type")), r.FormValue("createdBy"), file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[ocr.OCRJobCreateResult]{Data: result})
}

func (h Handlers) OCRJobDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	data, err := h.phaseThree.JobDetail(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[ocr.OCRJobDetail]{Data: data})
}

func (h Handlers) UpdateOCRReview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var input ocr.OCRReviewUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	if err := h.phaseThree.UpdateReview(r.Context(), id, input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reviewed"})
}

func (h Handlers) AssistOCRLine(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var input ocr.OCRLineAssistInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	suggestion, err := h.phaseThree.AssistLine(r.Context(), id, input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[ocr.OCRLineAssistSuggestion]{Data: suggestion})
}

func (h Handlers) RegisterOCRItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var input ocr.OCRRegisterItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	itemID, err := h.phaseThree.RegisterItem(r.Context(), id, input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"itemId": itemID, "status": "registered"})
}

func (h Handlers) CreateOCRProcurementDraft(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.phaseThree.CreateProcurementDraft(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, APIEnvelope[ocr.OCRProcurementDraftCreateResult]{Data: result})
}

func (h Handlers) RetryOCRJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.phaseThree.RetryJob(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIEnvelope[ocr.OCRRetryResult]{Data: result})
}

func inferContentType(fileName, contentType string) string {
	if contentType != "" && contentType != "application/octet-stream" {
		return contentType
	}
	byExtension := mime.TypeByExtension(filepath.Ext(fileName))
	if byExtension != "" {
		return byExtension
	}
	return "application/octet-stream"
}
