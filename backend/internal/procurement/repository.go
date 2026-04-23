package procurement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Projects(ctx context.Context) ([]ProjectSummary, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, project_key, name, synced_at FROM external_projects ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	out := []ProjectSummary{}
	for rows.Next() {
		var row ProjectSummary
		var syncedAt time.Time
		if err := rows.Scan(&row.ID, &row.Key, &row.Name, &syncedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		row.SyncedAt = syncedAt.UTC().Format(time.RFC3339)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) BudgetCategories(ctx context.Context, projectID string) ([]BudgetCategorySummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, category_key, name, synced_at
		FROM external_project_budget_categories
		WHERE ($1 = '' OR project_id = $1)
		ORDER BY name
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query budget categories: %w", err)
	}
	defer rows.Close()

	out := []BudgetCategorySummary{}
	for rows.Next() {
		var row BudgetCategorySummary
		var syncedAt time.Time
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.Key, &row.Name, &syncedAt); err != nil {
			return nil, fmt.Errorf("scan budget category: %w", err)
		}
		row.SyncedAt = syncedAt.UTC().Format(time.RFC3339)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) ProjectByID(ctx context.Context, id string) (ProjectSummary, error) {
	var project ProjectSummary
	var syncedAt time.Time
	if err := r.db.QueryRowContext(ctx, `
		SELECT id, project_key, name, synced_at
		FROM external_projects
		WHERE id = $1
	`, id).Scan(&project.ID, &project.Key, &project.Name, &syncedAt); err != nil {
		return ProjectSummary{}, fmt.Errorf("query project: %w", err)
	}
	project.SyncedAt = syncedAt.UTC().Format(time.RFC3339)
	return project, nil
}

func (r *Repository) Requests(ctx context.Context) (ProcurementRequestList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			pb.id,
			pb.batch_number,
			pb.title,
			COALESCE(ep.name, ''),
			COALESCE(bc.name, ''),
			COALESCE(s.name, ''),
			COALESCE(psp.normalized_status, pb.normalized_status),
			pb.source_type,
			COUNT(pl.id) AS requested_items,
			COALESCE(pdo.status, 'not_submitted'),
			COALESCE(sq.artifact_delete_status, 'retained'),
			pb.created_at
		FROM procurement_batches pb
		LEFT JOIN external_projects ep ON ep.id = pb.project_id
		LEFT JOIN external_project_budget_categories bc ON bc.id = pb.budget_category_id
		LEFT JOIN suppliers s ON s.id = pb.supplier_id
		LEFT JOIN supplier_quotations sq ON sq.id = pb.quotation_id
		LEFT JOIN procurement_dispatch_outbox pdo ON pdo.batch_id = pb.id
		LEFT JOIN procurement_status_projections psp ON psp.batch_id = pb.id
		LEFT JOIN procurement_lines pl ON pl.batch_id = pb.id
		GROUP BY pb.id, ep.name, bc.name, s.name, psp.normalized_status, pdo.status, sq.artifact_delete_status
		ORDER BY pb.created_at DESC
	`)
	if err != nil {
		return ProcurementRequestList{}, fmt.Errorf("query requests: %w", err)
	}
	defer rows.Close()

	out := ProcurementRequestList{Rows: []ProcurementRequestSummary{}}
	for rows.Next() {
		var row ProcurementRequestSummary
		var createdAt time.Time
		if err := rows.Scan(
			&row.ID,
			&row.BatchNumber,
			&row.Title,
			&row.ProjectName,
			&row.BudgetCategoryName,
			&row.SupplierName,
			&row.NormalizedStatus,
			&row.SourceType,
			&row.RequestedItems,
			&row.DispatchStatus,
			&row.ArtifactDeleteStatus,
			&createdAt,
		); err != nil {
			return ProcurementRequestList{}, fmt.Errorf("scan request: %w", err)
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out.Rows = append(out.Rows, row)
	}
	return out, rows.Err()
}

func (r *Repository) RequestDetail(ctx context.Context, id string) (ProcurementRequestDetail, error) {
	var detail ProcurementRequestDetail
	var quantityProgression []byte

	if err := r.db.QueryRowContext(ctx, `
		SELECT
			pb.id,
			pb.batch_number,
			pb.title,
			COALESCE(ep.name, ''),
			COALESCE(bc.name, ''),
			COALESCE(s.name, ''),
			COALESCE(sq.quotation_number, ''),
			COALESCE(sq.issue_date::text, ''),
			COALESCE(sq.artifact_path, ''),
			COALESCE(sq.artifact_delete_status, 'retained'),
			COALESCE(sq.artifact_deleted_at::text, ''),
			COALESCE(psp.normalized_status, pb.normalized_status),
			COALESCE(psp.raw_status, pb.status),
			COALESCE(psp.external_request_reference, ''),
			COALESCE(pdo.status, 'not_submitted'),
			COALESCE(pdo.attempt_count, 0),
			COALESCE(pdo.last_attempt_at::text, ''),
			COALESCE(pdh.normalized_error_code, ''),
			COALESCE(pdh.error_message, ''),
			COALESCE(psp.quantity_progression, '{}'::jsonb),
			COALESCE(psp.last_reconciled_at::text, ''),
			COALESCE(psp.sync_source, ''),
			COALESCE(psp.sync_error, '')
		FROM procurement_batches pb
		LEFT JOIN external_projects ep ON ep.id = pb.project_id
		LEFT JOIN external_project_budget_categories bc ON bc.id = pb.budget_category_id
		LEFT JOIN suppliers s ON s.id = pb.supplier_id
		LEFT JOIN supplier_quotations sq ON sq.id = pb.quotation_id
		LEFT JOIN procurement_status_projections psp ON psp.batch_id = pb.id
		LEFT JOIN procurement_dispatch_outbox pdo ON pdo.batch_id = pb.id
		LEFT JOIN LATERAL (
			SELECT normalized_error_code, error_message
			FROM procurement_dispatch_history
			WHERE batch_id = pb.id
			ORDER BY observed_at DESC
			LIMIT 1
		) pdh ON TRUE
		WHERE pb.id = $1
	`, id).Scan(
		&detail.ID,
		&detail.BatchNumber,
		&detail.Title,
		&detail.ProjectName,
		&detail.BudgetCategoryName,
		&detail.SupplierName,
		&detail.QuotationNumber,
		&detail.QuotationIssueDate,
		&detail.ArtifactPath,
		&detail.ArtifactDeleteStatus,
		&detail.ArtifactDeletedAt,
		&detail.NormalizedStatus,
		&detail.RawStatus,
		&detail.ExternalRequestReference,
		&detail.DispatchStatus,
		&detail.DispatchAttempts,
		&detail.LastDispatchAt,
		&detail.DispatchErrorCode,
		&detail.DispatchErrorMessage,
		&quantityProgression,
		&detail.LastReconciledAt,
		&detail.SyncSource,
		&detail.SyncError,
	); err != nil {
		return ProcurementRequestDetail{}, fmt.Errorf("query request detail: %w", err)
	}
	detail.QuantityProgression = string(quantityProgression)

	lineRows, err := r.db.QueryContext(ctx, `
		SELECT
			pl.id,
			COALESCE(i.canonical_item_number, ''),
			COALESCE(i.description, ql.item_description),
			pl.requested_quantity,
			pl.delivery_location,
			pl.accounting_category,
			COALESCE(ql.lead_time_days, 0),
			pl.note
		FROM procurement_lines pl
		LEFT JOIN items i ON i.id = pl.item_id
		LEFT JOIN quotation_lines ql ON ql.id = pl.quotation_line_id
		WHERE pl.batch_id = $1
		ORDER BY pl.created_at
	`, id)
	if err != nil {
		return ProcurementRequestDetail{}, fmt.Errorf("query request lines: %w", err)
	}
	defer lineRows.Close()
	detail.Lines = []ProcurementLine{}
	for lineRows.Next() {
		var row ProcurementLine
		if err := lineRows.Scan(
			&row.ID,
			&row.ItemNumber,
			&row.Description,
			&row.RequestedQuantity,
			&row.DeliveryLocation,
			&row.AccountingCategory,
			&row.LeadTimeDays,
			&row.Note,
		); err != nil {
			return ProcurementRequestDetail{}, fmt.Errorf("scan request line: %w", err)
		}
		detail.Lines = append(detail.Lines, row)
	}

	historyRows, err := r.db.QueryContext(ctx, `
		SELECT id, normalized_status, raw_status, observed_at, note
		FROM procurement_status_history
		WHERE batch_id = $1
		ORDER BY observed_at DESC
	`, id)
	if err != nil {
		return ProcurementRequestDetail{}, fmt.Errorf("query request history: %w", err)
	}
	defer historyRows.Close()
	detail.StatusHistory = []StatusHistoryEntry{}
	for historyRows.Next() {
		var row StatusHistoryEntry
		var observedAt time.Time
		if err := historyRows.Scan(&row.ID, &row.NormalizedStatus, &row.RawStatus, &observedAt, &row.Note); err != nil {
			return ProcurementRequestDetail{}, fmt.Errorf("scan request history: %w", err)
		}
		row.ObservedAt = observedAt.UTC().Format(time.RFC3339)
		detail.StatusHistory = append(detail.StatusHistory, row)
	}

	dispatchRows, err := r.db.QueryContext(ctx, `
		SELECT id, normalized_status, external_request_reference, retryable, normalized_error_code, error_message, observed_at
		FROM procurement_dispatch_history
		WHERE batch_id = $1
		ORDER BY observed_at DESC
	`, id)
	if err != nil {
		return ProcurementRequestDetail{}, fmt.Errorf("query request dispatch history: %w", err)
	}
	defer dispatchRows.Close()
	detail.DispatchHistory = []ProcurementDispatchHistoryEntry{}
	for dispatchRows.Next() {
		var row ProcurementDispatchHistoryEntry
		var observedAt time.Time
		if err := dispatchRows.Scan(&row.ID, &row.DispatchStatus, &row.ExternalRequestReference, &row.Retryable, &row.ErrorCode, &row.ErrorMessage, &observedAt); err != nil {
			return ProcurementRequestDetail{}, fmt.Errorf("scan request dispatch history: %w", err)
		}
		row.ObservedAt = observedAt.UTC().Format(time.RFC3339)
		detail.DispatchHistory = append(detail.DispatchHistory, row)
	}

	return detail, nil
}

func (r *Repository) CreateRequest(ctx context.Context, input ProcurementRequestCreateInput) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	id, _, err := r.createRequestTx(ctx, tx, createRequestTxInput{
		Title:            input.Title,
		ProjectID:        input.ProjectID,
		BudgetCategoryID: input.BudgetCategoryID,
		SupplierID:       input.SupplierID,
		QuotationID:      input.QuotationID,
		SourceType:       defaultString(input.SourceType, "manual"),
		CreatedBy:        defaultString(input.CreatedBy, "local-user"),
		HistoryNote:      "Created from local Phase 2 flow",
		Lines:            toCreateRequestLines(input.Lines),
	})
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return id, nil
}

func (r *Repository) CreateDraftFromOCR(ctx context.Context, input OCRProcurementDraftCreateInput) (OCRProcurementDraftCreateResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	batchID, batchNumber, quotationID, err := r.findOCRDraftTx(ctx, tx, input.SourceOCRJobID)
	if err != nil {
		return OCRProcurementDraftCreateResult{}, err
	}
	if batchID != "" {
		return OCRProcurementDraftCreateResult{
			ProcurementRequestID:   batchID,
			ProcurementBatchNumber: batchNumber,
			QuotationID:            quotationID,
			Status:                 "existing",
		}, nil
	}

	projectID, budgetCategoryID, err := r.deriveBatchContextTx(ctx, tx, input.Lines)
	if err != nil {
		return OCRProcurementDraftCreateResult{}, err
	}

	now := time.Now().UTC()
	quotationID = fmt.Sprintf("quote-%d", now.UnixNano())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO supplier_quotations (
			id, supplier_id, quotation_number, issue_date, artifact_path, status, source_ocr_job_id
		) VALUES ($1, $2, $3, NULLIF($4, '')::date, $5, 'reviewed', $6)
	`, quotationID, input.SupplierID, input.QuotationNumber, input.IssueDate, input.ArtifactPath, input.SourceOCRJobID); err != nil {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("insert supplier quotation: %w", err)
	}

	requestLines := make([]createRequestLineInput, 0, len(input.Lines))
	for index, line := range input.Lines {
		quotationLineID := fmt.Sprintf("quote-line-%d-%d", now.UnixNano(), index+1)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO quotation_lines (
				id, quotation_id, item_id, manufacturer_name, item_number, item_description, quantity,
				lead_time_days, delivery_location, accounting_category, supplier_contact
			) VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8, $9, $10, $11)
		`, quotationLineID, quotationID, line.ItemID, line.ManufacturerName, line.ItemNumber, line.ItemDescription, line.Quantity, line.LeadTimeDays, line.DeliveryLocation, line.AccountingCategory, line.SupplierContact); err != nil {
			return OCRProcurementDraftCreateResult{}, fmt.Errorf("insert quotation line: %w", err)
		}
		requestLines = append(requestLines, createRequestLineInput{
			ItemID:             line.ItemID,
			QuotationLineID:    quotationLineID,
			RequestedQuantity:  line.Quantity,
			DeliveryLocation:   line.DeliveryLocation,
			AccountingCategory: line.AccountingCategory,
			Note:               line.Note,
		})
	}

	batchID, batchNumber, err = r.createRequestTx(ctx, tx, createRequestTxInput{
		Title:            input.Title,
		ProjectID:        projectID,
		BudgetCategoryID: budgetCategoryID,
		SupplierID:       input.SupplierID,
		QuotationID:      quotationID,
		SourceType:       "ocr",
		CreatedBy:        defaultString(input.CreatedBy, "local-user"),
		SourceOCRJobID:   input.SourceOCRJobID,
		HistoryNote:      fmt.Sprintf("Created from OCR job %s", input.SourceOCRJobID),
		Lines:            requestLines,
	})
	if err != nil {
		return OCRProcurementDraftCreateResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("commit tx: %w", err)
	}

	return OCRProcurementDraftCreateResult{
		ProcurementRequestID:   batchID,
		ProcurementBatchNumber: batchNumber,
		QuotationID:            quotationID,
		Status:                 "created",
	}, nil
}

type createRequestTxInput struct {
	Title            string
	ProjectID        string
	BudgetCategoryID string
	SupplierID       string
	QuotationID      string
	SourceType       string
	CreatedBy        string
	SourceOCRJobID   string
	HistoryNote      string
	Lines            []createRequestLineInput
}

type createRequestLineInput struct {
	ItemID             string
	QuotationLineID    string
	RequestedQuantity  int
	DeliveryLocation   string
	AccountingCategory string
	Note               string
}

func (r *Repository) createRequestTx(ctx context.Context, tx *sql.Tx, input createRequestTxInput) (string, string, error) {
	now := time.Now().UTC()
	id := fmt.Sprintf("batch-%d", now.UnixNano())
	batchNumber := fmt.Sprintf("PR-%s-%03d", now.Format("20060102"), now.Nanosecond()%1000)

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_batches (
			id, batch_number, title, project_id, budget_category_id, supplier_id, quotation_id,
			status, normalized_status, source_type, created_by, source_ocr_job_id
		) VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), 'draft', 'draft', $8, $9, NULLIF($10, ''))
	`, id, batchNumber, input.Title, input.ProjectID, input.BudgetCategoryID, input.SupplierID, input.QuotationID, input.SourceType, input.CreatedBy, input.SourceOCRJobID); err != nil {
		return "", "", fmt.Errorf("insert batch: %w", err)
	}

	for index, line := range input.Lines {
		lineID := fmt.Sprintf("pline-%d-%d", now.UnixNano(), index+1)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO procurement_lines (
				id, batch_id, item_id, quotation_line_id, requested_quantity, unit, delivery_location, accounting_category, note
			) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, 'pcs', $6, $7, $8)
		`, lineID, id, line.ItemID, line.QuotationLineID, line.RequestedQuantity, line.DeliveryLocation, line.AccountingCategory, line.Note); err != nil {
			return "", "", fmt.Errorf("insert line: %w", err)
		}
	}

	progression, _ := json.Marshal(map[string]int{"requested": totalRequestedFromCreateLines(input.Lines), "ordered": 0, "received": 0})
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_status_projections (batch_id, normalized_status, raw_status, quantity_progression, external_request_reference, last_observed_at, updated_at)
		VALUES ($1, 'draft', 'draft', $2::jsonb, '', NOW(), NOW())
	`, id, string(progression)); err != nil {
		return "", "", fmt.Errorf("insert status projection: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_status_history (id, batch_id, normalized_status, raw_status, observed_at, note)
		VALUES ($1, $2, 'draft', 'draft', NOW(), $3)
	`, fmt.Sprintf("psh-%d", now.UnixNano()), id, defaultString(input.HistoryNote, "Created from local Phase 2 flow")); err != nil {
		return "", "", fmt.Errorf("insert status history: %w", err)
	}

	return id, batchNumber, nil
}

func (r *Repository) findOCRDraftTx(ctx context.Context, tx *sql.Tx, sourceOCRJobID string) (string, string, string, error) {
	if sourceOCRJobID == "" {
		return "", "", "", nil
	}
	var batchID, batchNumber, quotationID string
	err := tx.QueryRowContext(ctx, `
		SELECT id, batch_number, COALESCE(quotation_id, '')
		FROM procurement_batches
		WHERE source_ocr_job_id = $1
	`, sourceOCRJobID).Scan(&batchID, &batchNumber, &quotationID)
	if err == sql.ErrNoRows {
		return "", "", "", nil
	}
	if err != nil {
		return "", "", "", fmt.Errorf("query ocr procurement draft: %w", err)
	}
	return batchID, batchNumber, quotationID, nil
}

func (r *Repository) deriveBatchContextTx(ctx context.Context, tx *sql.Tx, lines []OCRProcurementDraftLineCreate) (string, string, error) {
	budgetIDs := map[string]struct{}{}
	for _, line := range lines {
		if line.BudgetCategoryID != "" {
			budgetIDs[line.BudgetCategoryID] = struct{}{}
		}
	}
	if len(budgetIDs) != 1 {
		return "", "", nil
	}
	var budgetCategoryID string
	for id := range budgetIDs {
		budgetCategoryID = id
	}
	var projectID string
	if err := tx.QueryRowContext(ctx, `
		SELECT project_id
		FROM external_project_budget_categories
		WHERE id = $1
	`, budgetCategoryID).Scan(&projectID); err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("budgetCategoryId not found: %s", budgetCategoryID)
		}
		return "", "", fmt.Errorf("query budget category project: %w", err)
	}
	return projectID, budgetCategoryID, nil
}

func totalRequested(lines []ProcurementRequestLineCreate) int {
	total := 0
	for _, line := range lines {
		total += line.RequestedQuantity
	}
	return total
}

func totalRequestedFromCreateLines(lines []createRequestLineInput) int {
	total := 0
	for _, line := range lines {
		total += line.RequestedQuantity
	}
	return total
}

func toCreateRequestLines(lines []ProcurementRequestLineCreate) []createRequestLineInput {
	out := make([]createRequestLineInput, 0, len(lines))
	for _, line := range lines {
		out = append(out, createRequestLineInput{
			ItemID:             line.ItemID,
			QuotationLineID:    line.QuotationLineID,
			RequestedQuantity:  line.RequestedQuantity,
			DeliveryLocation:   line.DeliveryLocation,
			AccountingCategory: line.AccountingCategory,
			Note:               line.Note,
		})
	}
	return out
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

type submissionState struct {
	DispatchStatus           string
	ExternalRequestReference string
	ArtifactDeleteStatus     string
}

func (r *Repository) SubmissionPayload(ctx context.Context, id string) (ProcurementSubmissionPayload, submissionState, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			pb.id,
			pb.batch_number,
			pb.title,
			COALESCE(ep.project_key, ''),
			COALESCE(ep.name, ''),
			COALESCE(bc.category_key, ''),
			COALESCE(bc.name, ''),
			COALESCE(s.id, ''),
			COALESCE(s.name, ''),
			COALESCE(sq.id, ''),
			COALESCE(sq.quotation_number, ''),
			COALESCE(sq.issue_date::text, ''),
			COALESCE(sq.artifact_path, ''),
			COALESCE(sq.artifact_delete_status, 'retained'),
			COALESCE(psp.external_request_reference, ''),
			COALESCE(pdo.status, 'not_submitted'),
			COALESCE(pl.item_id, ''),
			COALESCE(i.canonical_item_number, ''),
			COALESCE(i.description, ql.item_description),
			pl.requested_quantity,
			pl.delivery_location,
			pl.accounting_category,
			COALESCE(ql.lead_time_days, 0),
			COALESCE(ql.supplier_contact, '')
		FROM procurement_batches pb
		LEFT JOIN external_projects ep ON ep.id = pb.project_id
		LEFT JOIN external_project_budget_categories bc ON bc.id = pb.budget_category_id
		LEFT JOIN suppliers s ON s.id = pb.supplier_id
		LEFT JOIN supplier_quotations sq ON sq.id = pb.quotation_id
		LEFT JOIN procurement_status_projections psp ON psp.batch_id = pb.id
		LEFT JOIN procurement_dispatch_outbox pdo ON pdo.batch_id = pb.id
		JOIN procurement_lines pl ON pl.batch_id = pb.id
		LEFT JOIN items i ON i.id = pl.item_id
		LEFT JOIN quotation_lines ql ON ql.id = pl.quotation_line_id
		WHERE pb.id = $1
		ORDER BY pl.created_at
	`, id)
	if err != nil {
		return ProcurementSubmissionPayload{}, submissionState{}, fmt.Errorf("query submission payload: %w", err)
	}
	defer rows.Close()

	payload := ProcurementSubmissionPayload{Lines: []ProcurementPayloadLine{}}
	state := submissionState{}
	for rows.Next() {
		var line ProcurementPayloadLine
		if err := rows.Scan(
			&payload.BatchID,
			&payload.BatchNumber,
			&payload.Title,
			&payload.ProjectKey,
			&payload.ProjectName,
			&payload.BudgetCategoryKey,
			&payload.BudgetCategoryName,
			&payload.SupplierID,
			&payload.SupplierName,
			&payload.QuotationID,
			&payload.QuotationNumber,
			&payload.QuotationIssueDate,
			&payload.ArtifactPath,
			&state.ArtifactDeleteStatus,
			&state.ExternalRequestReference,
			&state.DispatchStatus,
			&line.ItemID,
			&line.ItemNumber,
			&line.Description,
			&line.RequestedQuantity,
			&line.DeliveryLocation,
			&line.AccountingCategory,
			&line.LeadTimeDays,
			&line.SupplierContact,
		); err != nil {
			return ProcurementSubmissionPayload{}, submissionState{}, fmt.Errorf("scan submission payload: %w", err)
		}
		payload.Lines = append(payload.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return ProcurementSubmissionPayload{}, submissionState{}, err
	}
	if payload.BatchID == "" {
		return ProcurementSubmissionPayload{}, submissionState{}, fmt.Errorf("procurement request not found: %s", id)
	}
	payload.IdempotencyKey = fmt.Sprintf("procurement:%s:quotation:%s", payload.BatchID, payload.QuotationID)
	return payload, state, nil
}

func (r *Repository) StartDispatchAttempt(ctx context.Context, batchID string, payload ProcurementSubmissionPayload) error {
	payloadJSON, _ := json.Marshal(payload)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO procurement_dispatch_outbox (
			id, batch_id, event_type, status, idempotency_key, payload, attempt_count, last_attempt_at, updated_at
		) VALUES ($1, $2, 'submit_procurement_request', 'processing', $3, $4::jsonb, 0, NULL, NOW())
		ON CONFLICT (batch_id) DO UPDATE SET
			status = 'processing',
			idempotency_key = EXCLUDED.idempotency_key,
			payload = EXCLUDED.payload,
			updated_at = NOW()
	`, fmt.Sprintf("outbox-%s", batchID), batchID, payload.IdempotencyKey, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("upsert dispatch outbox: %w", err)
	}
	return nil
}

func (r *Repository) RecordDispatchSuccess(ctx context.Context, batchID string, payload ProcurementSubmissionPayload, result DispatchResult) error {
	payloadJSON, _ := json.Marshal(payload)
	rawResponseJSON, _ := json.Marshal(result.RawResponse)
	evidenceJSON, _ := json.Marshal(result.EvidenceFileReferences)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin dispatch success tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE procurement_dispatch_outbox
		SET status = 'submitted',
		    attempt_count = attempt_count + 1,
		    last_attempt_at = NOW(),
		    next_attempt_at = NULL,
		    updated_at = NOW()
		WHERE batch_id = $1
	`, batchID); err != nil {
		return fmt.Errorf("update dispatch outbox success: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_dispatch_history (
			id, batch_id, normalized_status, external_request_reference, idempotency_key, payload, raw_response, evidence_file_references
		) VALUES ($1, $2, 'submitted', $3, $4, $5::jsonb, $6::jsonb, $7::jsonb)
	`, fmt.Sprintf("dispatch-%d", time.Now().UnixNano()), batchID, result.ExternalRequestReference, payload.IdempotencyKey, string(payloadJSON), string(rawResponseJSON), string(evidenceJSON)); err != nil {
		return fmt.Errorf("insert dispatch history success: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE procurement_status_projections
		SET normalized_status = 'submitted',
		    raw_status = $2,
		    external_request_reference = $3,
		    updated_at = NOW(),
		    last_observed_at = NOW()
		WHERE batch_id = $1
	`, batchID, defaultString(result.RawStatus, "submitted_to_internal_flow"), result.ExternalRequestReference); err != nil {
		return fmt.Errorf("update procurement status projection: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_status_history (id, batch_id, normalized_status, raw_status, observed_at, note)
		VALUES ($1, $2, 'submitted', $3, NOW(), $4)
	`, fmt.Sprintf("psh-%d", time.Now().UnixNano()), batchID, defaultString(result.RawStatus, "submitted_to_internal_flow"), "Submitted to local dispatch adapter"); err != nil {
		return fmt.Errorf("insert procurement status history success: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) RecordDispatchFailure(ctx context.Context, batchID string, payload ProcurementSubmissionPayload, dispatchErr NormalizedDispatchError) error {
	payloadJSON, _ := json.Marshal(payload)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin dispatch failure tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE procurement_dispatch_outbox
		SET status = 'failed',
		    attempt_count = attempt_count + 1,
		    last_attempt_at = NOW(),
		    next_attempt_at = CASE WHEN $2 THEN NOW() + INTERVAL '5 minutes' ELSE NULL END,
		    updated_at = NOW()
		WHERE batch_id = $1
	`, batchID, dispatchErr.Retryable); err != nil {
		return fmt.Errorf("update dispatch outbox failure: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_dispatch_history (
			id, batch_id, normalized_status, external_request_reference, idempotency_key, payload, raw_response,
			evidence_file_references, retryable, normalized_error_code, error_message
		) VALUES ($1, $2, 'failed', '', $3, $4::jsonb, '{}'::jsonb, '[]'::jsonb, $5, $6, $7)
	`, fmt.Sprintf("dispatch-%d", time.Now().UnixNano()), batchID, payload.IdempotencyKey, string(payloadJSON), dispatchErr.Retryable, dispatchErr.Code, dispatchErr.Message); err != nil {
		return fmt.Errorf("insert dispatch history failure: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) MarkArtifactDeleted(ctx context.Context, batchID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE supplier_quotations sq
		SET artifact_delete_status = 'deleted',
		    artifact_deleted_at = NOW(),
		    artifact_delete_error = ''
		FROM procurement_batches pb
		WHERE pb.id = $1 AND pb.quotation_id = sq.id
	`, batchID)
	if err != nil {
		return fmt.Errorf("mark artifact deleted: %w", err)
	}
	return nil
}

func (r *Repository) MarkArtifactDeleteFailure(ctx context.Context, batchID, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE supplier_quotations sq
		SET artifact_delete_status = 'delete_failed',
		    artifact_delete_error = $2
		FROM procurement_batches pb
		WHERE pb.id = $1 AND pb.quotation_id = sq.id
	`, batchID, message)
	if err != nil {
		return fmt.Errorf("mark artifact delete failure: %w", err)
	}
	return nil
}

type reconciliationContext struct {
	RequestID                string
	ExternalRequestReference string
	NormalizedStatus         string
	QuantityProgression      ProcurementQuantityProgression
}

func (r *Repository) ReconciliationContext(ctx context.Context, requestID string) (reconciliationContext, error) {
	var (
		out reconciliationContext
		raw []byte
	)
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			pb.id,
			COALESCE(psp.external_request_reference, ''),
			COALESCE(psp.normalized_status, pb.normalized_status),
			COALESCE(psp.quantity_progression, '{}'::jsonb)
		FROM procurement_batches pb
		LEFT JOIN procurement_status_projections psp ON psp.batch_id = pb.id
		WHERE pb.id = $1
	`, requestID).Scan(&out.RequestID, &out.ExternalRequestReference, &out.NormalizedStatus, &raw); err != nil {
		return reconciliationContext{}, fmt.Errorf("query reconciliation context: %w", err)
	}
	out.QuantityProgression = decodeQuantityProgression(raw)
	return out, nil
}

func (r *Repository) FindRequestIDByExternalReference(ctx context.Context, externalRequestReference string) (string, error) {
	var requestID string
	if err := r.db.QueryRowContext(ctx, `
		SELECT batch_id
		FROM procurement_status_projections
		WHERE external_request_reference = $1
	`, externalRequestReference).Scan(&requestID); err != nil {
		return "", fmt.Errorf("query request by external reference: %w", err)
	}
	return requestID, nil
}

func (r *Repository) ApplyReconciliation(ctx context.Context, requestID string, result ReconciliationResult) error {
	progressionJSON := encodeQuantityProgression(result.QuantityProgression)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reconciliation tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE procurement_batches
		SET normalized_status = $2,
		    status = $3
		WHERE id = $1
	`, requestID, result.NormalizedStatus, result.RawStatus); err != nil {
		return fmt.Errorf("update procurement batch reconciliation: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE procurement_status_projections
		SET normalized_status = $2,
		    raw_status = $3,
		    quantity_progression = $4::jsonb,
		    external_request_reference = $5,
		    last_observed_at = $6,
		    updated_at = NOW(),
		    last_reconciled_at = $6,
		    sync_source = $7,
		    sync_error = ''
		WHERE batch_id = $1
	`, requestID, result.NormalizedStatus, result.RawStatus, progressionJSON, result.ExternalRequestReference, result.ObservedAt, result.SyncSource); err != nil {
		return fmt.Errorf("update procurement status projection reconciliation: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_status_history (id, batch_id, normalized_status, raw_status, observed_at, note)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, fmt.Sprintf("psh-%d", time.Now().UnixNano()), requestID, result.NormalizedStatus, result.RawStatus, result.ObservedAt, defaultString(result.Note, fmt.Sprintf("Reconciled via %s", result.SyncSource))); err != nil {
		return fmt.Errorf("insert procurement status history reconciliation: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) RecordReconciliationFailure(ctx context.Context, requestID, message, source string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE procurement_status_projections
		SET sync_error = $2,
		    sync_source = $3,
		    updated_at = NOW()
		WHERE batch_id = $1
	`, requestID, message, source)
	if err != nil {
		return fmt.Errorf("record reconciliation failure: %w", err)
	}
	return nil
}

func (r *Repository) UpsertProjectMaster(ctx context.Context, rows []ProjectMasterRow, source, triggeredBy string) (MasterSyncResult, error) {
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return MasterSyncResult{}, fmt.Errorf("begin project master sync tx: %w", err)
	}
	defer tx.Rollback()

	for index, row := range rows {
		id := row.ID
		if id == "" {
			id = fmt.Sprintf("sync-project-%d-%d", now.UnixNano(), index+1)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO external_projects (id, project_key, name, status, synced_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (project_key) DO UPDATE SET
				name = EXCLUDED.name,
				status = EXCLUDED.status,
				synced_at = EXCLUDED.synced_at
		`, id, row.Key, row.Name, defaultString(row.Status, "active"), now); err != nil {
			return MasterSyncResult{}, fmt.Errorf("upsert project master row: %w", err)
		}
	}

	if err := insertMasterSyncRunTx(ctx, tx, masterSyncRunInput{
		ID:          fmt.Sprintf("sync-%d", now.UnixNano()),
		SyncType:    "projects",
		Status:      "completed",
		RowCount:    len(rows),
		Source:      source,
		TriggeredBy: triggeredBy,
		FinishedAt:  now,
	}); err != nil {
		return MasterSyncResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return MasterSyncResult{}, fmt.Errorf("commit project master sync tx: %w", err)
	}

	return MasterSyncResult{
		SyncType:    "projects",
		Status:      "completed",
		RowCount:    len(rows),
		Source:      source,
		TriggeredBy: triggeredBy,
		SyncedAt:    now.Format(time.RFC3339),
	}, nil
}

func (r *Repository) UpsertBudgetCategories(ctx context.Context, projectKey string, rows []BudgetCategoryMasterRow, source, triggeredBy string) (MasterSyncResult, error) {
	now := time.Now().UTC()
	var projectID string
	if err := r.db.QueryRowContext(ctx, `
		SELECT id
		FROM external_projects
		WHERE project_key = $1
	`, projectKey).Scan(&projectID); err != nil {
		return MasterSyncResult{}, fmt.Errorf("query project id for budget category sync: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return MasterSyncResult{}, fmt.Errorf("begin budget category sync tx: %w", err)
	}
	defer tx.Rollback()

	for index, row := range rows {
		id := row.ID
		if id == "" {
			id = fmt.Sprintf("sync-budget-%d-%d", now.UnixNano(), index+1)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO external_project_budget_categories (id, project_id, category_key, name, synced_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (project_id, category_key) DO UPDATE SET
				name = EXCLUDED.name,
				synced_at = EXCLUDED.synced_at
		`, id, projectID, row.Key, row.Name, now); err != nil {
			return MasterSyncResult{}, fmt.Errorf("upsert budget category row: %w", err)
		}
	}

	if err := insertMasterSyncRunTx(ctx, tx, masterSyncRunInput{
		ID:          fmt.Sprintf("sync-%d", now.UnixNano()),
		SyncType:    "budget_categories",
		ProjectID:   projectID,
		ProjectKey:  projectKey,
		Status:      "completed",
		RowCount:    len(rows),
		Source:      source,
		TriggeredBy: triggeredBy,
		FinishedAt:  now,
	}); err != nil {
		return MasterSyncResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return MasterSyncResult{}, fmt.Errorf("commit budget category sync tx: %w", err)
	}

	return MasterSyncResult{
		SyncType:    "budget_categories",
		ProjectID:   projectID,
		ProjectKey:  projectKey,
		Status:      "completed",
		RowCount:    len(rows),
		Source:      source,
		TriggeredBy: triggeredBy,
		SyncedAt:    now.Format(time.RFC3339),
	}, nil
}

func (r *Repository) RecordWebhookReceived(ctx context.Context, event WebhookEvent) (string, error) {
	now := time.Now().UTC()
	id := fmt.Sprintf("webhook-%d", now.UnixNano())
	payloadJSON, _ := json.Marshal(event.Payload)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO procurement_webhook_events (
			id, event_type, external_request_reference, project_key, normalized_status, raw_status, payload, received_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
	`, id, event.EventType, event.ExternalRequestReference, event.ProjectKey, event.NormalizedStatus, event.RawStatus, string(payloadJSON), now)
	if err != nil {
		return "", fmt.Errorf("insert procurement webhook event: %w", err)
	}
	return id, nil
}

func (r *Repository) MarkWebhookProcessed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE procurement_webhook_events
		SET processed_at = NOW(),
		    processing_error = ''
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("mark procurement webhook processed: %w", err)
	}
	return nil
}

func (r *Repository) MarkWebhookFailed(ctx context.Context, id, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE procurement_webhook_events
		SET processed_at = NOW(),
		    processing_error = $2
		WHERE id = $1
	`, id, message)
	if err != nil {
		return fmt.Errorf("mark procurement webhook failed: %w", err)
	}
	return nil
}

func (r *Repository) MasterSyncRuns(ctx context.Context) ([]MasterSyncRunEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, sync_type, COALESCE(project_id, ''), project_key, status, row_count, source, triggered_by, error_message, started_at, COALESCE(finished_at, started_at)
		FROM procurement_master_sync_runs
		ORDER BY started_at DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("query master sync runs: %w", err)
	}
	defer rows.Close()

	result := []MasterSyncRunEntry{}
	for rows.Next() {
		var entry MasterSyncRunEntry
		var startedAt time.Time
		var finishedAt time.Time
		if err := rows.Scan(
			&entry.ID,
			&entry.SyncType,
			&entry.ProjectID,
			&entry.ProjectKey,
			&entry.Status,
			&entry.RowCount,
			&entry.Source,
			&entry.TriggeredBy,
			&entry.ErrorMessage,
			&startedAt,
			&finishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan master sync run: %w", err)
		}
		entry.StartedAt = startedAt.UTC().Format(time.RFC3339)
		entry.FinishedAt = finishedAt.UTC().Format(time.RFC3339)
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (r *Repository) WebhookEvents(ctx context.Context) ([]WebhookEventEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, event_type, external_request_reference, project_key, normalized_status, raw_status, received_at, COALESCE(processed_at, received_at), processing_error
		FROM procurement_webhook_events
		ORDER BY received_at DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("query webhook events: %w", err)
	}
	defer rows.Close()

	result := []WebhookEventEntry{}
	for rows.Next() {
		var entry WebhookEventEntry
		var receivedAt time.Time
		var processedAt time.Time
		if err := rows.Scan(
			&entry.ID,
			&entry.EventType,
			&entry.ExternalRequestReference,
			&entry.ProjectKey,
			&entry.NormalizedStatus,
			&entry.RawStatus,
			&receivedAt,
			&processedAt,
			&entry.ProcessingError,
		); err != nil {
			return nil, fmt.Errorf("scan webhook event: %w", err)
		}
		entry.ReceivedAt = receivedAt.UTC().Format(time.RFC3339)
		entry.ProcessedAt = processedAt.UTC().Format(time.RFC3339)
		result = append(result, entry)
	}
	return result, rows.Err()
}

type masterSyncRunInput struct {
	ID          string
	SyncType    string
	ProjectID   string
	ProjectKey  string
	Status      string
	RowCount    int
	Source      string
	TriggeredBy string
	FinishedAt  time.Time
}

func insertMasterSyncRunTx(ctx context.Context, tx *sql.Tx, input masterSyncRunInput) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_master_sync_runs (
			id, sync_type, project_id, project_key, status, row_count, source, triggered_by, started_at, finished_at
		) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, $8, $9, $9)
	`, input.ID, input.SyncType, input.ProjectID, input.ProjectKey, input.Status, input.RowCount, input.Source, input.TriggeredBy, input.FinishedAt); err != nil {
		return fmt.Errorf("insert master sync run: %w", err)
	}
	return nil
}

func decodeQuantityProgression(raw []byte) ProcurementQuantityProgression {
	if len(raw) == 0 {
		return ProcurementQuantityProgression{}
	}
	var progression ProcurementQuantityProgression
	if err := json.Unmarshal(raw, &progression); err != nil {
		return ProcurementQuantityProgression{}
	}
	return progression
}

func encodeQuantityProgression(progression ProcurementQuantityProgression) string {
	payload, err := json.Marshal(progression)
	if err != nil {
		return "{}"
	}
	return string(payload)
}
