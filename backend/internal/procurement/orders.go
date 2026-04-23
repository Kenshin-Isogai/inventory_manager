package procurement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (r *Repository) Orders(ctx context.Context) (PurchaseOrderList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			po.id,
			po.procurement_batch_id,
			pb.batch_number,
			po.order_number,
			po.status,
			COALESCE(s.name, ''),
			COALESCE(SUM(pol.ordered_quantity), 0),
			COALESCE(SUM(pol.received_quantity), 0),
			po.issued_at
		FROM purchase_orders po
		JOIN procurement_batches pb ON pb.id = po.procurement_batch_id
		LEFT JOIN suppliers s ON s.id = pb.supplier_id
		LEFT JOIN purchase_order_lines pol ON pol.purchase_order_id = po.id
		GROUP BY po.id, pb.batch_number, s.name
		ORDER BY COALESCE(po.issued_at, po.created_at) DESC, po.order_number DESC
	`)
	if err != nil {
		return PurchaseOrderList{}, fmt.Errorf("query purchase orders: %w", err)
	}
	defer rows.Close()

	result := PurchaseOrderList{Rows: []PurchaseOrderSummary{}}
	for rows.Next() {
		var row PurchaseOrderSummary
		var issuedAt sql.NullTime
		if err := rows.Scan(&row.ID, &row.ProcurementBatchID, &row.BatchNumber, &row.OrderNumber, &row.Status, &row.SupplierName, &row.OrderedQuantity, &row.ReceivedQuantity, &issuedAt); err != nil {
			return PurchaseOrderList{}, fmt.Errorf("scan purchase order: %w", err)
		}
		row.OpenQuantity = row.OrderedQuantity - row.ReceivedQuantity
		row.IssuedAt = nullableTimeString(issuedAt)
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) OrderDetail(ctx context.Context, id string) (PurchaseOrderDetail, error) {
	var detail PurchaseOrderDetail
	var issuedAt sql.NullTime
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			po.id,
			po.procurement_batch_id,
			pb.batch_number,
			pb.title,
			po.order_number,
			po.status,
			COALESCE(s.name, ''),
			po.issued_at
		FROM purchase_orders po
		JOIN procurement_batches pb ON pb.id = po.procurement_batch_id
		LEFT JOIN suppliers s ON s.id = pb.supplier_id
		WHERE po.id = $1
	`, id).Scan(&detail.ID, &detail.ProcurementBatchID, &detail.BatchNumber, &detail.Title, &detail.OrderNumber, &detail.Status, &detail.SupplierName, &issuedAt); err != nil {
		return PurchaseOrderDetail{}, fmt.Errorf("query purchase order detail: %w", err)
	}
	detail.IssuedAt = nullableTimeString(issuedAt)

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			pol.id,
			pol.procurement_line_id,
			COALESCE(pl.item_id, ''),
			COALESCE(i.canonical_item_number, ''),
			COALESCE(i.description, ql.item_description),
			pol.ordered_quantity,
			COALESCE(pol.received_quantity, 0),
			COALESCE(pol.expected_arrival_date::text, ''),
			COALESCE(pol.status, 'ordered'),
			COALESCE(pl.delivery_location, ''),
			COALESCE(pol.note, '')
		FROM purchase_order_lines pol
		JOIN procurement_lines pl ON pl.id = pol.procurement_line_id
		LEFT JOIN items i ON i.id = pl.item_id
		LEFT JOIN quotation_lines ql ON ql.id = pl.quotation_line_id
		WHERE pol.purchase_order_id = $1
		ORDER BY pol.created_at
	`, id)
	if err != nil {
		return PurchaseOrderDetail{}, fmt.Errorf("query purchase order lines: %w", err)
	}
	defer rows.Close()

	detail.Lines = []PurchaseOrderLine{}
	for rows.Next() {
		var row PurchaseOrderLine
		if err := rows.Scan(&row.ID, &row.ProcurementLineID, &row.ItemID, &row.ItemNumber, &row.Description, &row.OrderedQuantity, &row.ReceivedQuantity, &row.ExpectedArrivalDate, &row.Status, &row.DeliveryLocation, &row.Note); err != nil {
			return PurchaseOrderDetail{}, fmt.Errorf("scan purchase order line: %w", err)
		}
		row.OpenQuantity = row.OrderedQuantity - row.ReceivedQuantity
		detail.Lines = append(detail.Lines, row)
	}
	return detail, rows.Err()
}

func (r *Repository) CreateOrder(ctx context.Context, input PurchaseOrderCreateInput) (PurchaseOrderDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return PurchaseOrderDetail{}, fmt.Errorf("begin purchase order create tx: %w", err)
	}
	defer tx.Rollback()

	orderID, batchID, err := r.upsertOrderTx(ctx, tx, "", input.ProcurementBatchID, input.OrderNumber, input.Status, input.IssuedAt, input.Lines)
	if err != nil {
		return PurchaseOrderDetail{}, err
	}
	if err := r.refreshBatchProgressionTx(ctx, tx, batchID); err != nil {
		return PurchaseOrderDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return PurchaseOrderDetail{}, fmt.Errorf("commit purchase order create tx: %w", err)
	}
	return r.OrderDetail(ctx, orderID)
}

func (r *Repository) UpdateOrder(ctx context.Context, id string, input PurchaseOrderUpdateInput) (PurchaseOrderDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return PurchaseOrderDetail{}, fmt.Errorf("begin purchase order update tx: %w", err)
	}
	defer tx.Rollback()

	var batchID string
	if err := tx.QueryRowContext(ctx, `SELECT procurement_batch_id FROM purchase_orders WHERE id = $1 FOR UPDATE`, id).Scan(&batchID); err != nil {
		return PurchaseOrderDetail{}, fmt.Errorf("load purchase order: %w", err)
	}
	var receivedCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM purchase_order_lines WHERE purchase_order_id = $1 AND received_quantity > 0`, id).Scan(&receivedCount); err != nil {
		return PurchaseOrderDetail{}, fmt.Errorf("query received order lines: %w", err)
	}
	if receivedCount > 0 {
		return PurchaseOrderDetail{}, fmt.Errorf("received purchase orders cannot be replaced")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM purchase_order_lines WHERE purchase_order_id = $1`, id); err != nil {
		return PurchaseOrderDetail{}, fmt.Errorf("clear purchase order lines: %w", err)
	}
	if _, _, err := r.upsertOrderTx(ctx, tx, id, batchID, input.OrderNumber, input.Status, input.IssuedAt, input.Lines); err != nil {
		return PurchaseOrderDetail{}, err
	}
	if err := r.refreshBatchProgressionTx(ctx, tx, batchID); err != nil {
		return PurchaseOrderDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return PurchaseOrderDetail{}, fmt.Errorf("commit purchase order update tx: %w", err)
	}
	return r.OrderDetail(ctx, id)
}

func (r *Repository) DeleteOrder(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin purchase order delete tx: %w", err)
	}
	defer tx.Rollback()

	var batchID string
	if err := tx.QueryRowContext(ctx, `SELECT procurement_batch_id FROM purchase_orders WHERE id = $1 FOR UPDATE`, id).Scan(&batchID); err != nil {
		return fmt.Errorf("load purchase order for delete: %w", err)
	}
	var receiptCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM receipt_lines rl
		JOIN purchase_order_lines pol ON pol.id = rl.purchase_order_line_id
		WHERE pol.purchase_order_id = $1
	`, id).Scan(&receiptCount); err != nil {
		return fmt.Errorf("query purchase order receipts: %w", err)
	}
	if receiptCount > 0 {
		return fmt.Errorf("purchase orders with receipts cannot be deleted")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM purchase_orders WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete purchase order: %w", err)
	}
	if err := r.refreshBatchProgressionTx(ctx, tx, batchID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) upsertOrderTx(ctx context.Context, tx *sql.Tx, id, procurementBatchID, orderNumber, status, issuedAtRaw string, lines []PurchaseOrderLineInput) (string, string, error) {
	if procurementBatchID == "" {
		return "", "", fmt.Errorf("procurementBatchId is required")
	}
	var batchExists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM procurement_batches WHERE id = $1`, procurementBatchID).Scan(&batchExists); err != nil {
		return "", "", fmt.Errorf("query procurement batch: %w", err)
	}
	if batchExists == 0 {
		return "", "", fmt.Errorf("procurement batch not found: %s", procurementBatchID)
	}

	now := time.Now().UTC()
	if id == "" {
		id = fmt.Sprintf("po-%d", now.UnixNano())
	}
	if orderNumber == "" {
		orderNumber = fmt.Sprintf("PO-%s-%03d", now.Format("20060102"), now.Nanosecond()%1000)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO purchase_orders (id, procurement_batch_id, order_number, status, issued_at, created_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::timestamptz, NOW())
		ON CONFLICT (id) DO UPDATE
		SET order_number = EXCLUDED.order_number,
		    status = EXCLUDED.status,
		    issued_at = EXCLUDED.issued_at
	`, id, procurementBatchID, orderNumber, defaultString(status, "ordered"), issuedAtRaw); err != nil {
		return "", "", fmt.Errorf("upsert purchase order: %w", err)
	}

	for index, line := range lines {
		var lineBatchID string
		if err := tx.QueryRowContext(ctx, `SELECT batch_id FROM procurement_lines WHERE id = $1`, line.ProcurementLineID).Scan(&lineBatchID); err != nil {
			return "", "", fmt.Errorf("load procurement line: %w", err)
		}
		if lineBatchID != procurementBatchID {
			return "", "", fmt.Errorf("procurement line %s does not belong to batch %s", line.ProcurementLineID, procurementBatchID)
		}
		lineID := line.ID
		if lineID == "" {
			lineID = fmt.Sprintf("po-line-%d-%d", now.UnixNano(), index+1)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO purchase_order_lines (
				id, purchase_order_id, procurement_line_id, ordered_quantity, received_quantity, expected_arrival_date, status, note, created_at
			) VALUES ($1, $2, $3, $4, 0, NULLIF($5, '')::date, $6, $7, NOW())
		`, lineID, id, line.ProcurementLineID, line.OrderedQuantity, line.ExpectedArrivalDate, defaultString(line.Status, "ordered"), line.Note); err != nil {
			return "", "", fmt.Errorf("insert purchase order line: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_status_history (id, batch_id, normalized_status, raw_status, observed_at, note)
		VALUES ($1, $2, 'ordered', $3, NOW(), $4)
	`, fmt.Sprintf("psh-%d", now.UnixNano()), procurementBatchID, defaultString(status, "ordered"), "Purchase order changed via backend API"); err != nil {
		return "", "", fmt.Errorf("insert purchase order history: %w", err)
	}
	return id, procurementBatchID, nil
}

func (r *Repository) refreshBatchProgressionTx(ctx context.Context, tx *sql.Tx, batchID string) error {
	var requested, ordered, received int
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(requested_quantity), 0)
		FROM procurement_lines
		WHERE batch_id = $1
	`, batchID).Scan(&requested); err != nil {
		return fmt.Errorf("query requested quantity progression: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(pol.ordered_quantity), 0), COALESCE(SUM(pol.received_quantity), 0)
		FROM purchase_order_lines pol
		JOIN purchase_orders po ON po.id = pol.purchase_order_id
		WHERE po.procurement_batch_id = $1
	`, batchID).Scan(&ordered, &received); err != nil {
		return fmt.Errorf("query ordered quantity progression: %w", err)
	}

	normalizedStatus := "draft"
	rawStatus := "draft"
	if ordered > 0 {
		normalizedStatus = "ordered"
		rawStatus = "ordered"
	}
	if received > 0 {
		normalizedStatus = "partially_received"
		rawStatus = "partially_received"
	}
	if ordered > 0 && received >= ordered {
		normalizedStatus = "received"
		rawStatus = "received"
	}

	progression, _ := json.Marshal(map[string]int{
		"requested": requested,
		"ordered":   ordered,
		"received":  received,
	})
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO procurement_status_projections (batch_id, normalized_status, raw_status, quantity_progression, external_request_reference, last_observed_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, '', NOW(), NOW())
		ON CONFLICT (batch_id) DO UPDATE
		SET normalized_status = EXCLUDED.normalized_status,
		    raw_status = EXCLUDED.raw_status,
		    quantity_progression = EXCLUDED.quantity_progression,
		    updated_at = NOW(),
		    last_observed_at = NOW()
	`, batchID, normalizedStatus, rawStatus, string(progression)); err != nil {
		return fmt.Errorf("upsert procurement status projection: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE procurement_batches
		SET normalized_status = $2,
		    status = $3
		WHERE id = $1
	`, batchID, normalizedStatus, rawStatus); err != nil {
		return fmt.Errorf("update procurement batch status: %w", err)
	}
	return nil
}

func nullableTimeString(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339)
}
