package procurement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (r *Repository) UpdateRequest(ctx context.Context, id string, input ProcurementRequestUpdateInput) (ProcurementRequestDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ProcurementRequestDetail{}, fmt.Errorf("begin procurement update tx: %w", err)
	}
	defer tx.Rollback()

	var dispatchStatus string
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(dispatch_status, '')
		FROM procurement_dispatch_outbox
		WHERE batch_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, id).Scan(&dispatchStatus); err != nil && err != sql.ErrNoRows {
		return ProcurementRequestDetail{}, fmt.Errorf("query dispatch status: %w", err)
	}
	if dispatchStatus == "submitted" {
		return ProcurementRequestDetail{}, fmt.Errorf("submitted procurement requests cannot be edited")
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE procurement_batches
		SET title = $2,
		    project_id = NULLIF($3, ''),
		    budget_category_id = NULLIF($4, ''),
		    supplier_id = NULLIF($5, ''),
		    updated_at = NOW()
		WHERE id = $1
	`, id, input.Title, input.ProjectID, input.BudgetCategoryID, input.SupplierID); err != nil {
		return ProcurementRequestDetail{}, fmt.Errorf("update procurement batch: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM procurement_lines WHERE batch_id = $1`, id); err != nil {
		return ProcurementRequestDetail{}, fmt.Errorf("clear procurement lines: %w", err)
	}

	now := time.Now().UTC()
	for index, line := range input.Lines {
		lineID := line.ID
		if lineID == "" {
			lineID = fmt.Sprintf("pline-%d-%d", now.UnixNano(), index+1)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO procurement_lines (
				id, batch_id, item_id, quotation_line_id, requested_quantity, unit, delivery_location, accounting_category, note,
				status, lead_time_days, budget_category_id, supplier_contact, updated_at
			) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, 'pcs', $6, $7, $8, $9, $10, NULLIF($11, ''), $12, NOW())
		`, lineID, id, line.ItemID, line.QuotationLineID, line.RequestedQuantity, line.DeliveryLocation, line.AccountingCategory, line.Note, defaultString(line.Status, "draft"), line.LeadTimeDays, line.BudgetCategoryID, line.SupplierContact); err != nil {
			return ProcurementRequestDetail{}, fmt.Errorf("insert updated procurement line: %w", err)
		}
	}

	progression, _ := json.Marshal(map[string]int{
		"requested": totalRequestedUpdateLines(input.Lines),
		"ordered":   0,
		"received":  0,
	})
	if _, err := tx.ExecContext(ctx, `
		UPDATE procurement_status_projections
		SET quantity_progression = $2::jsonb,
		    updated_at = NOW()
		WHERE batch_id = $1
	`, id, string(progression)); err != nil {
		return ProcurementRequestDetail{}, fmt.Errorf("update procurement quantity progression: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_status_history (id, batch_id, normalized_status, raw_status, observed_at, note)
		VALUES ($1, $2, 'draft', 'draft_updated', NOW(), $3)
	`, fmt.Sprintf("psh-%d", now.UnixNano()), id, "Draft updated via backend API"); err != nil {
		return ProcurementRequestDetail{}, fmt.Errorf("insert procurement update history: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return ProcurementRequestDetail{}, fmt.Errorf("commit procurement update tx: %w", err)
	}
	return r.RequestDetail(ctx, id)
}

func totalRequestedUpdateLines(lines []ProcurementRequestLineUpdate) int {
	total := 0
	for _, line := range lines {
		total += line.RequestedQuantity
	}
	return total
}
