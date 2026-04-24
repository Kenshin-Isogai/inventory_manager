package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"backend/internal/platform/storage"
	procurement "backend/internal/procurement"
)

type Service struct {
	repo        *Repository
	store       storage.Store
	provider    Provider
	procurement *procurement.Service
}

const (
	maxOCRRetries   = 3
	ocrStaleTimeout = 10 * time.Minute
)

func NewService(repo *Repository, store storage.Store, provider Provider, procurement *procurement.Service) *Service {
	return &Service{repo: repo, store: store, provider: provider, procurement: procurement}
}

func (s *Service) CreateJob(ctx context.Context, fileName, contentType, createdBy string, body io.Reader) (OCRJobCreateResult, error) {
	id := fmt.Sprintf("ocr-%d", time.Now().UnixNano())
	safeName := strings.ReplaceAll(filepath.Base(fileName), " ", "_")
	artifactPath, err := s.store.Save(ctx, filepath.Join("ocr", id, safeName), body)
	if err != nil {
		return OCRJobCreateResult{}, fmt.Errorf("save artifact: %w", err)
	}

	if err := s.repo.CreateJob(ctx, id, fileName, contentType, artifactPath, providerName(s.provider), defaultString(createdBy, "local-user")); err != nil {
		if cleanupErr := s.cleanupArtifact(ctx, artifactPath); cleanupErr != nil {
			return OCRJobCreateResult{}, fmt.Errorf("create ocr job: %w (artifact cleanup failed: %v)", err, cleanupErr)
		}
		return OCRJobCreateResult{}, err
	}

	doc, err := s.provider.Extract(ctx, artifactPath, contentType)
	if err != nil {
		_ = s.repo.MarkFailed(ctx, id, err.Error())
		if cleanupErr := s.cleanupArtifact(ctx, artifactPath); cleanupErr != nil {
			return OCRJobCreateResult{}, fmt.Errorf("extract ocr: %w (artifact cleanup failed: %v)", err, cleanupErr)
		}
		return OCRJobCreateResult{}, fmt.Errorf("extract ocr: %w", err)
	}

	if err := s.autofillMatches(ctx, &doc); err != nil {
		_ = s.repo.MarkFailed(ctx, id, err.Error())
		if cleanupErr := s.cleanupArtifact(ctx, artifactPath); cleanupErr != nil {
			return OCRJobCreateResult{}, fmt.Errorf("autofill OCR matches: %w (artifact cleanup failed: %v)", err, cleanupErr)
		}
		return OCRJobCreateResult{}, fmt.Errorf("autofill OCR matches: %w", err)
	}

	if err := s.repo.SaveResult(ctx, id, doc); err != nil {
		_ = s.repo.MarkFailed(ctx, id, err.Error())
		if cleanupErr := s.cleanupArtifact(ctx, artifactPath); cleanupErr != nil {
			return OCRJobCreateResult{}, fmt.Errorf("save ocr result: %w (artifact cleanup failed: %v)", err, cleanupErr)
		}
		return OCRJobCreateResult{}, err
	}

	return OCRJobCreateResult{ID: id, Status: "ready_for_review"}, nil
}

func (s *Service) Jobs(ctx context.Context, createdBy string) (OCRJobList, error) {
	if err := s.repo.RecoverStaleProcessingJobs(ctx, ocrStaleTimeout); err != nil {
		return OCRJobList{}, err
	}
	return s.repo.Jobs(ctx, createdBy)
}

func (s *Service) JobDetail(ctx context.Context, id string) (OCRJobDetail, error) {
	if id == "" {
		return OCRJobDetail{}, fmt.Errorf("job id is required")
	}
	if err := s.repo.RecoverStaleProcessingJobs(ctx, ocrStaleTimeout); err != nil {
		return OCRJobDetail{}, err
	}
	detail, err := s.repo.JobDetail(ctx, id)
	if err != nil {
		return OCRJobDetail{}, err
	}
	if err := s.attachCandidates(ctx, &detail); err != nil {
		return OCRJobDetail{}, err
	}
	return detail, nil
}

func (s *Service) UpdateReview(ctx context.Context, id string, input OCRReviewUpdateInput) error {
	if id == "" {
		return fmt.Errorf("job id is required")
	}
	return s.repo.UpdateReview(ctx, id, input)
}

func (s *Service) AssistLine(ctx context.Context, jobID string, input OCRLineAssistInput) (OCRLineAssistSuggestion, error) {
	if jobID == "" || input.LineID == "" {
		return OCRLineAssistSuggestion{}, fmt.Errorf("job id and line id are required")
	}
	detail, err := s.JobDetail(ctx, jobID)
	if err != nil {
		return OCRLineAssistSuggestion{}, err
	}
	var target *OCRResultLine
	for index := range detail.Lines {
		if detail.Lines[index].ID == input.LineID {
			target = &detail.Lines[index]
			break
		}
	}
	if target == nil {
		return OCRLineAssistSuggestion{}, fmt.Errorf("ocr line not found: %s", input.LineID)
	}
	categories, err := s.repo.CategoryKeys(ctx)
	if err != nil {
		return OCRLineAssistSuggestion{}, err
	}
	suggestion, err := s.provider.SuggestLineResolution(ctx, LineResolutionInput{
		SupplierName: detail.SupplierName,
		Line:         *target,
		Candidates:   target.MatchCandidates,
		Categories:   categories,
	})
	if err != nil {
		return OCRLineAssistSuggestion{}, err
	}
	if suggestion.LineID == "" {
		suggestion.LineID = input.LineID
	}
	if len(suggestion.Candidates) == 0 {
		suggestion.Candidates = target.MatchCandidates
	}
	return suggestion, nil
}

func (s *Service) RegisterItem(ctx context.Context, jobID string, input OCRRegisterItemInput) (string, error) {
	if jobID == "" {
		return "", fmt.Errorf("job id is required")
	}
	if input.LineID == "" {
		return "", fmt.Errorf("lineId is required")
	}
	if input.CanonicalItemNumber == "" {
		return "", fmt.Errorf("canonicalItemNumber is required")
	}
	if input.ManufacturerName == "" {
		return "", fmt.Errorf("manufacturerName is required")
	}
	if input.Description == "" {
		return "", fmt.Errorf("description is required")
	}
	if input.DefaultSupplierID == "" {
		detail, err := s.JobDetail(ctx, jobID)
		if err == nil {
			input.DefaultSupplierID = detail.SupplierID
		}
	}
	return s.repo.RegisterItemFromOCR(ctx, jobID, input)
}

func (s *Service) RetryJob(ctx context.Context, jobID string) (OCRRetryResult, error) {
	if jobID == "" {
		return OCRRetryResult{}, fmt.Errorf("job id is required")
	}
	job, err := s.repo.RetryJob(ctx, jobID)
	if err != nil {
		return OCRRetryResult{}, err
	}
	if job.Status == "processing" {
		if err := s.repo.RecoverStaleProcessingJobs(ctx, ocrStaleTimeout); err != nil {
			return OCRRetryResult{}, err
		}
		job, err = s.repo.RetryJob(ctx, jobID)
		if err != nil {
			return OCRRetryResult{}, err
		}
		if job.Status == "processing" {
			return OCRRetryResult{}, fmt.Errorf("ocr job is still processing")
		}
	}
	if job.RetryCount >= maxOCRRetries {
		return OCRRetryResult{}, fmt.Errorf("ocr retry limit reached")
	}
	if job.Status != "failed" && job.Status != "ready_for_review" && job.Status != "reviewed" {
		return OCRRetryResult{}, fmt.Errorf("ocr job cannot be retried from status %s", job.Status)
	}
	if err := s.repo.MarkRetrying(ctx, jobID); err != nil {
		return OCRRetryResult{}, err
	}
	doc, err := s.provider.Extract(ctx, job.ArtifactPath, job.ContentType)
	if err != nil {
		_ = s.repo.MarkFailed(ctx, jobID, err.Error())
		return OCRRetryResult{}, fmt.Errorf("extract ocr: %w", err)
	}
	if err := s.autofillMatches(ctx, &doc); err != nil {
		_ = s.repo.MarkFailed(ctx, jobID, err.Error())
		return OCRRetryResult{}, fmt.Errorf("autofill OCR matches: %w", err)
	}
	if err := s.repo.SaveResult(ctx, jobID, doc); err != nil {
		_ = s.repo.MarkFailed(ctx, jobID, err.Error())
		return OCRRetryResult{}, err
	}
	retriedJob, err := s.repo.RetryJob(ctx, jobID)
	if err != nil {
		return OCRRetryResult{}, err
	}
	return OCRRetryResult{ID: jobID, Status: "ready_for_review", RetryCount: retriedJob.RetryCount}, nil
}

func (s *Service) CreateProcurementDraft(ctx context.Context, jobID string) (OCRProcurementDraftCreateResult, error) {
	if jobID == "" {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("job id is required")
	}
	if s.procurement == nil {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("procurement service is not configured")
	}

	detail, err := s.JobDetail(ctx, jobID)
	if err != nil {
		return OCRProcurementDraftCreateResult{}, err
	}

	if detail.Status != "ready_for_review" && detail.Status != "reviewed" {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("ocr job status must be ready_for_review or reviewed")
	}
	if detail.SupplierID == "" {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("supplierId is required before creating a procurement draft")
	}
	if detail.QuotationNumber == "" {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("quotationNumber is required before creating a procurement draft")
	}
	if detail.IssueDate == "" {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("issueDate is required before creating a procurement draft")
	}
	if len(detail.Lines) == 0 {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("at least one OCR line is required")
	}

	lines := make([]procurement.OCRProcurementDraftLineCreate, 0, len(detail.Lines))
	for _, line := range detail.Lines {
		if line.ItemID == "" {
			return OCRProcurementDraftCreateResult{}, fmt.Errorf("ocr line %s must be resolved before creating a procurement draft", line.ID)
		}
		if !line.IsUserConfirmed {
			return OCRProcurementDraftCreateResult{}, fmt.Errorf("ocr line %s must be user-confirmed before creating a procurement draft", line.ID)
		}
		lines = append(lines, procurement.OCRProcurementDraftLineCreate{
			ItemID:             line.ItemID,
			ManufacturerName:   line.ManufacturerName,
			ItemNumber:         line.ItemNumber,
			ItemDescription:    line.ItemDescription,
			Quantity:           line.Quantity,
			LeadTimeDays:       line.LeadTimeDays,
			DeliveryLocation:   line.DeliveryLocation,
			BudgetCategoryID:   line.BudgetCategoryID,
			AccountingCategory: line.AccountingCategory,
			SupplierContact:    line.SupplierContact,
			Note:               fmt.Sprintf("Created from OCR quotation line %s", defaultString(line.ItemNumber, line.ID)),
		})
	}

	result, err := s.procurement.CreateDraftFromOCR(ctx, procurement.OCRProcurementDraftCreateInput{
		SourceOCRJobID:  jobID,
		Title:           defaultOCRDraftTitle(detail),
		SupplierID:      detail.SupplierID,
		QuotationNumber: detail.QuotationNumber,
		IssueDate:       detail.IssueDate,
		ArtifactPath:    detail.ArtifactPath,
		CreatedBy:       "local-user",
		Lines:           lines,
	})
	if err != nil {
		return OCRProcurementDraftCreateResult{}, err
	}

	return OCRProcurementDraftCreateResult{
		ProcurementRequestID:   result.ProcurementRequestID,
		ProcurementBatchNumber: result.ProcurementBatchNumber,
		QuotationID:            result.QuotationID,
		Status:                 result.Status,
	}, nil
}

func (s *Service) DeleteJob(ctx context.Context, jobID string) error {
	if jobID == "" {
		return fmt.Errorf("job id is required")
	}
	artifactPath, err := s.repo.JobArtifactPath(ctx, jobID)
	if err != nil {
		return err
	}
	if err := s.repo.SoftDeleteJob(ctx, jobID); err != nil {
		return err
	}
	if artifactPath != "" {
		_ = s.cleanupArtifact(ctx, artifactPath)
	}
	return nil
}

const ocrJobTTL = 10 * 24 * time.Hour // 10 days

func (s *Service) CleanupExpiredJobs(ctx context.Context) (int, error) {
	expired, err := s.repo.ListExpiredJobs(ctx, ocrJobTTL)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, job := range expired {
		if err := s.repo.SoftDeleteJob(ctx, job.ID); err != nil {
			continue
		}
		if job.ArtifactPath != "" {
			_ = s.cleanupArtifact(ctx, job.ArtifactPath)
		}
		count++
	}
	return count, nil
}

func providerName(provider Provider) string {
	if provider == nil {
		return "unknown"
	}
	return provider.Name()
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultOCRDraftTitle(detail OCRJobDetail) string {
	switch {
	case strings.TrimSpace(detail.SupplierName) != "" && strings.TrimSpace(detail.QuotationNumber) != "":
		return fmt.Sprintf("%s quotation %s", strings.TrimSpace(detail.SupplierName), strings.TrimSpace(detail.QuotationNumber))
	case strings.TrimSpace(detail.QuotationNumber) != "":
		return fmt.Sprintf("OCR quotation %s", strings.TrimSpace(detail.QuotationNumber))
	default:
		return fmt.Sprintf("OCR procurement draft %s", detail.ID)
	}
}

func (s *Service) cleanupArtifact(ctx context.Context, artifactPath string) error {
	if s.store == nil || artifactPath == "" {
		return nil
	}
	return s.store.Delete(ctx, artifactPath)
}

func (s *Service) autofillMatches(ctx context.Context, doc *ExtractedDocument) error {
	if doc == nil {
		return nil
	}
	supplierCandidates, err := s.findSupplierCandidates(ctx, coalesceSupplierQuery(doc.SupplierName, doc.RawPayload))
	if err != nil {
		return err
	}
	if doc.SupplierID == "" && len(supplierCandidates) > 0 && supplierCandidates[0].Score >= 0.9 {
		doc.SupplierID = supplierCandidates[0].ID
	}

	for index := range doc.Lines {
		line := &doc.Lines[index]
		itemCandidates, err := s.findItemCandidates(ctx, *line, doc.SupplierID)
		if err != nil {
			return err
		}
		if line.ItemID == "" && len(itemCandidates) > 0 && itemCandidates[0].Score >= 0.95 {
			line.ItemID = itemCandidates[0].ItemID
		}
	}
	return nil
}

func (s *Service) attachCandidates(ctx context.Context, detail *OCRJobDetail) error {
	if detail == nil {
		return nil
	}

	if detail.SupplierName == "" {
		detail.SupplierName = extractSupplierName(detail.RawPayload)
	}

	supplierCandidates, err := s.findSupplierCandidates(ctx, coalesceSupplierQuery(detail.SupplierName, detail.RawPayload))
	if err != nil {
		return err
	}
	detail.SupplierMatch = supplierCandidates
	if detail.SupplierID == "" && len(supplierCandidates) > 0 && supplierCandidates[0].Score >= 0.9 {
		detail.SupplierID = supplierCandidates[0].ID
	}

	for index := range detail.Lines {
		itemCandidates, err := s.findItemCandidates(ctx, detail.Lines[index], detail.SupplierID)
		if err != nil {
			return err
		}
		detail.Lines[index].MatchCandidates = itemCandidates
		if detail.Lines[index].ItemID == "" && len(itemCandidates) > 0 && itemCandidates[0].Score >= 0.95 {
			detail.Lines[index].ItemID = itemCandidates[0].ItemID
		}
	}
	return nil
}

func (s *Service) findSupplierCandidates(ctx context.Context, query string) ([]OCRSupplierCandidate, error) {
	if s.repo == nil || strings.TrimSpace(query) == "" {
		return []OCRSupplierCandidate{}, nil
	}
	records, err := s.repo.Suppliers(ctx)
	if err != nil {
		return nil, err
	}
	normalizedQuery := normalizeMatchKey(query)
	candidates := make([]OCRSupplierCandidate, 0, len(records))
	for _, record := range records {
		score, reason := scoreSupplierCandidate(normalizedQuery, record.Name)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, OCRSupplierCandidate{
			ID:          record.ID,
			Name:        record.Name,
			Score:       score,
			MatchReason: reason,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Name < candidates[j].Name
		}
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}
	return candidates, nil
}

func (s *Service) findItemCandidates(ctx context.Context, line OCRResultLine, supplierID string) ([]OCRItemCandidate, error) {
	if s.repo == nil {
		return []OCRItemCandidate{}, nil
	}
	records, err := s.repo.ItemRecords(ctx)
	if err != nil {
		return nil, err
	}
	normalizedManufacturer := normalizeMatchKey(line.ManufacturerName)
	normalizedItemNumber := normalizeMatchKey(line.ItemNumber)
	normalizedDescription := normalizeMatchKey(line.ItemDescription)
	candidates := make([]OCRItemCandidate, 0, len(records))
	for _, record := range records {
		score, reason := scoreItemCandidate(normalizedManufacturer, normalizedItemNumber, normalizedDescription, supplierID, record)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, OCRItemCandidate{
			ItemID:              record.ItemID,
			CanonicalItemNumber: record.CanonicalItemNumber,
			Description:         record.Description,
			ManufacturerName:    record.ManufacturerName,
			DefaultSupplierID:   record.DefaultSupplierID,
			SupplierAlias:       record.SupplierAlias,
			Score:               score,
			MatchReason:         reason,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].CanonicalItemNumber < candidates[j].CanonicalItemNumber
		}
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}
	return candidates, nil
}

func coalesceSupplierQuery(supplierName, rawPayload string) string {
	if strings.TrimSpace(supplierName) != "" {
		return supplierName
	}
	return extractSupplierName(rawPayload)
}

func extractSupplierName(rawPayload string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		return ""
	}
	if supplierName, ok := payload["supplier_name"].(string); ok {
		return strings.TrimSpace(supplierName)
	}
	return ""
}

func scoreSupplierCandidate(query, supplierName string) (float64, string) {
	normalizedName := normalizeMatchKey(supplierName)
	switch {
	case query == "" || normalizedName == "":
		return 0, ""
	case normalizedName == query:
		return 1, "exact supplier name match"
	case strings.Contains(normalizedName, query) || strings.Contains(query, normalizedName):
		return 0.92, "supplier name contains match"
	case sharedTokenCount(query, normalizedName) >= 2:
		return 0.75, "supplier name token overlap"
	default:
		return 0, ""
	}
}

func scoreItemCandidate(manufacturer, itemNumber, description, supplierID string, record itemRecord) (float64, string) {
	normalizedCanonical := normalizeMatchKey(record.CanonicalItemNumber)
	normalizedAlias := normalizeMatchKey(record.SupplierAlias)
	normalizedManufacturer := normalizeMatchKey(record.ManufacturerName)
	normalizedDescription := normalizeMatchKey(record.Description)

	score := 0.0
	reasons := []string{}

	if itemNumber != "" && normalizedAlias != "" && normalizedAlias == itemNumber {
		score += 0.78
		reasons = append(reasons, "exact supplier alias match")
	}
	if itemNumber != "" && normalizedCanonical == itemNumber {
		score += 0.72
		reasons = append(reasons, "exact canonical item match")
	}
	if itemNumber != "" && normalizedAlias != "" && strings.Contains(normalizedAlias, itemNumber) {
		score += 0.45
		reasons = append(reasons, "partial supplier alias match")
	}
	if itemNumber != "" && strings.Contains(itemNumber, normalizedCanonical) {
		score += 0.36
		reasons = append(reasons, "canonical item contained in OCR item number")
	}
	if manufacturer != "" && normalizedManufacturer == manufacturer {
		score += 0.18
		reasons = append(reasons, "manufacturer match")
	}
	if supplierID != "" && record.DefaultSupplierID == supplierID {
		score += 0.12
		reasons = append(reasons, "default supplier match")
	}
	if description != "" && normalizedDescription != "" && sharedTokenCount(description, normalizedDescription) >= 2 {
		score += 0.12
		reasons = append(reasons, "description token overlap")
	}

	if score > 1 {
		score = 1
	}
	if score < 0.3 {
		return 0, ""
	}
	return score, strings.Join(reasons, ", ")
}

func normalizeMatchKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func sharedTokenCount(left, right string) int {
	leftTokens := tokenSet(left)
	rightTokens := tokenSet(right)
	count := 0
	for token := range leftTokens {
		if _, ok := rightTokens[token]; ok {
			count++
		}
	}
	return count
}

func tokenSet(value string) map[string]struct{} {
	normalized := strings.ToLower(value)
	replacer := strings.NewReplacer("/", " ", "-", " ", "_", " ", ",", " ", ".", " ", "(", " ", ")", " ")
	fields := strings.Fields(replacer.Replace(normalized))
	out := map[string]struct{}{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, err := strconv.Atoi(field); err == nil {
			continue
		}
		out[field] = struct{}{}
	}
	return out
}
