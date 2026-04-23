package ocr

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateJob(ctx context.Context, id, fileName, contentType, artifactPath, provider, createdBy string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ocr_jobs (id, file_name, content_type, artifact_path, status, provider, created_by, processing_started_at)
		VALUES ($1, $2, $3, $4, 'processing', $5, $6, NOW())
	`, id, fileName, contentType, artifactPath, provider, createdBy)
	if err != nil {
		return fmt.Errorf("insert ocr job: %w", err)
	}
	return nil
}

func (r *Repository) MarkFailed(ctx context.Context, jobID, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE ocr_jobs
		SET status = 'failed', error_message = $2, processing_started_at = NULL, updated_at = NOW()
		WHERE id = $1
	`, jobID, message)
	if err != nil {
		return fmt.Errorf("mark ocr job failed: %w", err)
	}
	return nil
}

func (r *Repository) SaveResult(ctx context.Context, jobID string, doc ExtractedDocument) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ocr_job_results (job_id, supplier_id, quotation_number, issue_date, raw_payload)
		VALUES (
			$1,
			CASE
				WHEN NULLIF($2, '') IS NULL THEN NULL
				WHEN EXISTS (SELECT 1 FROM suppliers WHERE id = $2) THEN $2
				ELSE NULL
			END,
			$3,
			NULLIF($4, '')::date,
			$5::jsonb
		)
		ON CONFLICT (job_id) DO UPDATE SET
			supplier_id = EXCLUDED.supplier_id,
			quotation_number = EXCLUDED.quotation_number,
			issue_date = EXCLUDED.issue_date,
			raw_payload = EXCLUDED.raw_payload,
			updated_at = NOW()
	`, jobID, doc.SupplierID, doc.QuotationNumber, doc.IssueDate, doc.RawPayload); err != nil {
		return fmt.Errorf("upsert ocr result: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM ocr_result_lines WHERE job_id = $1`, jobID); err != nil {
		return fmt.Errorf("clear ocr lines: %w", err)
	}

	for index, line := range doc.Lines {
		lineID := fmt.Sprintf("%s-line-%d", jobID, index+1)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO ocr_result_lines (
				id, job_id, item_id, manufacturer_name, item_number, item_description, quantity,
				lead_time_days, delivery_location, budget_category_id, accounting_category,
				supplier_contact, is_user_confirmed
			) VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8, $9, NULLIF($10, ''), $11, $12, $13)
		`,
			lineID, jobID, line.ItemID, line.ManufacturerName, line.ItemNumber, line.ItemDescription,
			line.Quantity, line.LeadTimeDays, line.DeliveryLocation, line.BudgetCategoryID,
			line.AccountingCategory, line.SupplierContact, line.IsUserConfirmed,
		); err != nil {
			return fmt.Errorf("insert ocr line: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE ocr_jobs
		SET status = 'ready_for_review', error_message = '', processing_started_at = NULL, updated_at = NOW()
		WHERE id = $1
	`, jobID); err != nil {
		return fmt.Errorf("mark ocr job ready: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) Jobs(ctx context.Context) (OCRJobList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, file_name, content_type, status, provider, retry_count, created_at, updated_at
		FROM ocr_jobs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return OCRJobList{}, fmt.Errorf("query ocr jobs: %w", err)
	}
	defer rows.Close()

	out := OCRJobList{Rows: []OCRJobSummary{}}
	for rows.Next() {
		var row OCRJobSummary
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&row.ID, &row.FileName, &row.ContentType, &row.Status, &row.Provider, &row.RetryCount, &createdAt, &updatedAt); err != nil {
			return OCRJobList{}, fmt.Errorf("scan ocr job: %w", err)
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		row.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		out.Rows = append(out.Rows, row)
	}
	return out, rows.Err()
}

func (r *Repository) JobDetail(ctx context.Context, id string) (OCRJobDetail, error) {
	var detail OCRJobDetail
	var rawPayload []byte
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			j.id,
			j.file_name,
			j.content_type,
			j.artifact_path,
			j.status,
			j.provider,
			j.error_message,
			j.retry_count,
			COALESCE(res.supplier_id, ''),
			COALESCE(sq.id, ''),
			COALESCE(res.quotation_number, ''),
			COALESCE(res.issue_date::text, ''),
			COALESCE(pb.id, ''),
			COALESCE(pb.batch_number, ''),
			COALESCE(res.raw_payload, '{}'::jsonb)
		FROM ocr_jobs j
		LEFT JOIN ocr_job_results res ON res.job_id = j.id
		LEFT JOIN supplier_quotations sq ON sq.source_ocr_job_id = j.id
		LEFT JOIN procurement_batches pb ON pb.source_ocr_job_id = j.id
		WHERE j.id = $1
	`, id).Scan(
		&detail.ID,
		&detail.FileName,
		&detail.ContentType,
		&detail.ArtifactPath,
		&detail.Status,
		&detail.Provider,
		&detail.ErrorMessage,
		&detail.RetryCount,
		&detail.SupplierID,
		&detail.QuotationID,
		&detail.QuotationNumber,
		&detail.IssueDate,
		&detail.ProcurementRequestID,
		&detail.ProcurementBatchNumber,
		&rawPayload,
	); err != nil {
		return OCRJobDetail{}, fmt.Errorf("query ocr job detail: %w", err)
	}
	detail.RawPayload = string(rawPayload)

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(item_id, ''), manufacturer_name, item_number, item_description, quantity,
		       lead_time_days, delivery_location, COALESCE(budget_category_id, ''), accounting_category,
		       supplier_contact, is_user_confirmed
		FROM ocr_result_lines
		WHERE job_id = $1
		ORDER BY created_at
	`, id)
	if err != nil {
		return OCRJobDetail{}, fmt.Errorf("query ocr lines: %w", err)
	}
	defer rows.Close()

	detail.Lines = []OCRResultLine{}
	for rows.Next() {
		var line OCRResultLine
		if err := rows.Scan(
			&line.ID,
			&line.ItemID,
			&line.ManufacturerName,
			&line.ItemNumber,
			&line.ItemDescription,
			&line.Quantity,
			&line.LeadTimeDays,
			&line.DeliveryLocation,
			&line.BudgetCategoryID,
			&line.AccountingCategory,
			&line.SupplierContact,
			&line.IsUserConfirmed,
		); err != nil {
			return OCRJobDetail{}, fmt.Errorf("scan ocr line: %w", err)
		}
		detail.Lines = append(detail.Lines, line)
	}
	return detail, rows.Err()
}

func (r *Repository) UpdateReview(ctx context.Context, jobID string, input OCRReviewUpdateInput) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE ocr_job_results
		SET supplier_id = NULLIF($2, ''),
			quotation_number = $3,
			issue_date = NULLIF($4, '')::date,
			updated_at = NOW()
		WHERE job_id = $1
	`, jobID, input.SupplierID, input.QuotationNumber, input.IssueDate); err != nil {
		return fmt.Errorf("update ocr header: %w", err)
	}

	for _, line := range input.Lines {
		if _, err := tx.ExecContext(ctx, `
			UPDATE ocr_result_lines
			SET item_id = NULLIF($2, ''),
				delivery_location = $3,
				budget_category_id = NULLIF($4, ''),
				accounting_category = $5,
				supplier_contact = $6,
				is_user_confirmed = $7,
				updated_at = NOW()
			WHERE id = $1
		`, line.ID, line.ItemID, line.DeliveryLocation, line.BudgetCategoryID, line.AccountingCategory, line.SupplierContact, line.IsUserConfirmed); err != nil {
			return fmt.Errorf("update ocr line: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE ocr_jobs
		SET status = 'reviewed', updated_at = NOW()
		WHERE id = $1
	`, jobID); err != nil {
		return fmt.Errorf("mark ocr reviewed: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) CategoryKeys(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key FROM categories ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("query category keys: %w", err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan category key: %w", err)
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

type retryableJob struct {
	ID           string
	FileName     string
	ContentType  string
	ArtifactPath string
	Status       string
	RetryCount   int
}

func (r *Repository) RecoverStaleProcessingJobs(ctx context.Context, olderThan time.Duration) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE ocr_jobs
		SET status = 'failed',
		    error_message = 'OCR processing timed out and requires retry',
		    processing_started_at = NULL,
		    updated_at = NOW()
		WHERE status = 'processing'
		  AND COALESCE(processing_started_at, updated_at) < NOW() - $1::interval
	`, fmt.Sprintf("%d seconds", int(olderThan.Seconds())))
	if err != nil {
		return fmt.Errorf("recover stale ocr jobs: %w", err)
	}
	return nil
}

func (r *Repository) RetryJob(ctx context.Context, jobID string) (retryableJob, error) {
	var job retryableJob
	if err := r.db.QueryRowContext(ctx, `
		SELECT id, file_name, content_type, artifact_path, status, retry_count
		FROM ocr_jobs
		WHERE id = $1
	`, jobID).Scan(&job.ID, &job.FileName, &job.ContentType, &job.ArtifactPath, &job.Status, &job.RetryCount); err != nil {
		if err == sql.ErrNoRows {
			return retryableJob{}, fmt.Errorf("ocr job not found: %s", jobID)
		}
		return retryableJob{}, fmt.Errorf("query retryable ocr job: %w", err)
	}
	return job, nil
}

func (r *Repository) MarkRetrying(ctx context.Context, jobID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE ocr_jobs
		SET status = 'processing',
		    error_message = '',
		    retry_count = retry_count + 1,
		    last_retry_at = NOW(),
		    processing_started_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`, jobID)
	if err != nil {
		return fmt.Errorf("mark ocr job retrying: %w", err)
	}
	return nil
}

type supplierRecord struct {
	ID   string
	Name string
}

func (r *Repository) Suppliers(ctx context.Context) ([]supplierRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM suppliers ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query suppliers: %w", err)
	}
	defer rows.Close()

	out := []supplierRecord{}
	for rows.Next() {
		var row supplierRecord
		if err := rows.Scan(&row.ID, &row.Name); err != nil {
			return nil, fmt.Errorf("scan supplier: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type itemRecord struct {
	ItemID              string
	CanonicalItemNumber string
	Description         string
	ManufacturerName    string
	DefaultSupplierID   string
	SupplierAlias       string
}

func (r *Repository) ItemRecords(ctx context.Context) ([]itemRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			i.id,
			i.canonical_item_number,
			i.description,
			m.name,
			COALESCE(i.default_supplier_id, ''),
			COALESCE(sia.supplier_item_number, '')
		FROM items i
		JOIN manufacturers m ON m.key = i.manufacturer_key
		LEFT JOIN supplier_item_aliases sia ON sia.item_id = i.id
		WHERE i.active = TRUE
		ORDER BY i.canonical_item_number, sia.supplier_item_number
	`)
	if err != nil {
		return nil, fmt.Errorf("query item records: %w", err)
	}
	defer rows.Close()

	out := []itemRecord{}
	for rows.Next() {
		var row itemRecord
		if err := rows.Scan(
			&row.ItemID,
			&row.CanonicalItemNumber,
			&row.Description,
			&row.ManufacturerName,
			&row.DefaultSupplierID,
			&row.SupplierAlias,
		); err != nil {
			return nil, fmt.Errorf("scan item record: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) RegisterItemFromOCR(ctx context.Context, jobID string, input OCRRegisterItemInput) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	categoryKey := normalizeKey(input.CategoryKey)
	categoryName := strings.TrimSpace(input.CategoryName)
	if categoryKey == "" {
		categoryKey = "misc"
	}
	if categoryName == "" {
		categoryName = strings.Title(strings.ReplaceAll(categoryKey, "-", " "))
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO categories (key, name)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET name = EXCLUDED.name
	`, categoryKey, categoryName); err != nil {
		return "", fmt.Errorf("upsert category: %w", err)
	}

	manufacturerName := strings.TrimSpace(input.ManufacturerName)
	manufacturerKey := normalizeKey(manufacturerName)
	if manufacturerKey == "" {
		return "", fmt.Errorf("manufacturerName is required")
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO manufacturers (key, name)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET name = EXCLUDED.name
	`, manufacturerKey, manufacturerName); err != nil {
		return "", fmt.Errorf("upsert manufacturer: %w", err)
	}

	itemNumber := strings.TrimSpace(input.CanonicalItemNumber)
	if itemNumber == "" {
		return "", fmt.Errorf("canonicalItemNumber is required")
	}

	itemID := ""
	if err := tx.QueryRowContext(ctx, `SELECT id FROM items WHERE canonical_item_number = $1`, itemNumber).Scan(&itemID); err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("query existing item: %w", err)
	}
	if itemID == "" {
		itemID = fmt.Sprintf("item-%d", time.Now().UnixNano())
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description, default_supplier_id, note, active)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), 'Created from OCR review', TRUE)
		`, itemID, manufacturerKey, categoryKey, itemNumber, strings.TrimSpace(input.Description), input.DefaultSupplierID); err != nil {
			return "", fmt.Errorf("insert item: %w", err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
			UPDATE items
			SET manufacturer_key = $2,
			    category_key = $3,
			    description = $4,
			    default_supplier_id = NULLIF($5, ''),
			    note = 'Updated from OCR review'
			WHERE id = $1
		`, itemID, manufacturerKey, categoryKey, strings.TrimSpace(input.Description), input.DefaultSupplierID); err != nil {
			return "", fmt.Errorf("update item: %w", err)
		}
	}

	if alias := strings.TrimSpace(input.SupplierAliasNumber); alias != "" && strings.TrimSpace(input.DefaultSupplierID) != "" {
		unitsPerOrder := input.UnitsPerOrder
		if unitsPerOrder <= 0 {
			unitsPerOrder = 1
		}
		existingAliasID, existingAliasItemID := "", ""
		err := tx.QueryRowContext(ctx, `
			SELECT id, item_id
			FROM supplier_item_aliases
			WHERE supplier_id = $1 AND supplier_item_number = $2
		`, input.DefaultSupplierID, alias).Scan(&existingAliasID, &existingAliasItemID)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("query existing alias: %w", err)
		}
		if existingAliasID == "" {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO supplier_item_aliases (id, item_id, supplier_id, supplier_item_number, units_per_order)
				VALUES ($1, $2, $3, $4, $5)
			`, fmt.Sprintf("alias-%d", time.Now().UnixNano()), itemID, input.DefaultSupplierID, alias, unitsPerOrder); err != nil {
				return "", fmt.Errorf("insert alias: %w", err)
			}
		} else if existingAliasItemID != itemID {
			return "", fmt.Errorf("supplier alias already belongs to another item: %s", alias)
		} else {
			if _, err := tx.ExecContext(ctx, `
				UPDATE supplier_item_aliases
				SET units_per_order = $2
				WHERE id = $1
			`, existingAliasID, unitsPerOrder); err != nil {
				return "", fmt.Errorf("update alias units: %w", err)
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE ocr_result_lines
		SET item_id = $3, is_user_confirmed = TRUE, updated_at = NOW()
		WHERE job_id = $1 AND id = $2
	`, jobID, input.LineID, itemID); err != nil {
		return "", fmt.Errorf("update ocr line item: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return itemID, nil
}

func normalizeKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	value = reg.ReplaceAllString(value, "")
	value = strings.Trim(value, "-")
	return value
}
