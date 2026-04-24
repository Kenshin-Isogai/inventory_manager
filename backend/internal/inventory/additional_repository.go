package inventory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ItemFlow returns chronological inventory events for a single item with running balance.
func (r *Repository) ItemFlow(ctx context.Context, itemID string) (ItemFlowList, error) {
	const q = `
		SELECT
			ie.event_type,
			ie.quantity_delta,
			COALESCE(ie.to_location_code, ie.from_location_code, ie.location_code, '') AS loc,
			COALESCE(ie.source_type, '') AS source_type,
			COALESCE(ie.source_id, '') AS source_id,
			COALESCE(ie.note, '') AS note,
			COALESCE(ie.occurred_at, ie.created_at) AS ts,
			i.canonical_item_number
		FROM inventory_events ie
		JOIN items i ON i.id = ie.item_id
		WHERE ie.item_id = $1
		ORDER BY ts ASC, ie.id ASC
	`
	rows, err := r.db.QueryContext(ctx, q, itemID)
	if err != nil {
		return ItemFlowList{}, fmt.Errorf("query item flow: %w", err)
	}
	defer rows.Close()

	var result ItemFlowList
	result.ItemID = itemID
	balance := 0
	for rows.Next() {
		var e ItemFlowEntry
		var occurredAt time.Time
		if err := rows.Scan(&e.EventType, &e.QuantityDelta, &e.LocationCode, &e.SourceType, &e.SourceRef, &e.Note, &occurredAt, &result.ItemNumber); err != nil {
			return ItemFlowList{}, fmt.Errorf("scan item flow: %w", err)
		}
		balance += e.QuantityDelta
		e.RunningBalance = balance
		e.Date = occurredAt.Format(time.RFC3339)
		result.Rows = append(result.Rows, e)
	}
	if result.Rows == nil {
		result.Rows = []ItemFlowEntry{}
	}
	return result, rows.Err()
}

// ScopeOverview returns scope tree with summary counts for requirements, reservations, shortages.
func (r *Repository) ScopeOverview(ctx context.Context, device string) (ScopeOverviewList, error) {
	const q = `
		WITH req_counts AS (
			SELECT scope_id, COUNT(*) AS cnt
			FROM scope_item_requirements
			GROUP BY scope_id
		),
		res_counts AS (
			SELECT device_scope_id AS scope_id, COUNT(*) AS cnt
			FROM reservations
			WHERE status NOT IN ('cancelled', 'released', 'fulfilled')
			GROUP BY device_scope_id
		),
		shortage_counts AS (
			SELECT
				sir.scope_id,
				COUNT(DISTINCT sir.item_id) AS cnt
			FROM scope_item_requirements sir
			LEFT JOIN (
				SELECT item_id, COALESCE(SUM(available_quantity), 0) AS avail
				FROM inventory_balances
				GROUP BY item_id
			) ib ON ib.item_id = sir.item_id
			LEFT JOIN (
				SELECT item_id, device_scope_id,
					   COALESCE(SUM(quantity), 0) AS reserved
				FROM reservations
				WHERE status NOT IN ('cancelled', 'released')
				GROUP BY item_id, device_scope_id
			) rv ON rv.item_id = sir.item_id AND rv.device_scope_id = sir.scope_id
			WHERE sir.quantity > COALESCE(rv.reserved, 0) + COALESCE(ib.avail, 0)
			GROUP BY sir.scope_id
		)
		SELECT
			COALESCE(d.device_key, ds.device_key) AS device_key,
			COALESCE(d.name, ds.device_key) AS device_name,
			ds.id AS scope_id,
			ds.scope_key,
			COALESCE(ds.scope_name, ds.scope_key) AS scope_name,
			COALESCE(ds.scope_type, '') AS scope_type,
			COALESCE(ds.parent_scope_id, '') AS parent_scope_id,
			COALESCE(ds.status, 'active') AS status,
			ds.planned_start_at,
			COALESCE(rc.cnt, 0) AS requirements_count,
			COALESCE(rsc.cnt, 0) AS reservations_count,
			COALESCE(sc.cnt, 0) AS shortage_count,
			COALESCE(ds.owner_department_key, '') AS owner_dept
		FROM device_scopes ds
		LEFT JOIN devices d ON d.id = ds.device_id
		LEFT JOIN req_counts rc ON rc.scope_id = ds.id
		LEFT JOIN res_counts rsc ON rsc.scope_id = ds.id
		LEFT JOIN shortage_counts sc ON sc.scope_id = ds.id
		WHERE ($1 = '' OR COALESCE(d.device_key, ds.device_key) = $1)
		ORDER BY device_key, ds.scope_key
	`
	rows, err := r.db.QueryContext(ctx, q, device)
	if err != nil {
		return ScopeOverviewList{}, fmt.Errorf("query scope overview: %w", err)
	}
	defer rows.Close()

	var result ScopeOverviewList
	for rows.Next() {
		var row ScopeOverviewRow
		var plannedStart sql.NullTime
		if err := rows.Scan(
			&row.DeviceKey, &row.DeviceName, &row.ScopeID, &row.ScopeKey,
			&row.ScopeName, &row.ScopeType, &row.ParentScopeID, &row.Status,
			&plannedStart,
			&row.RequirementsCount, &row.ReservationsCount, &row.ShortageItemCount,
			&row.OwnerDepartment,
		); err != nil {
			return ScopeOverviewList{}, fmt.Errorf("scan scope overview: %w", err)
		}
		if plannedStart.Valid {
			row.PlannedStartAt = plannedStart.Time.Format(time.RFC3339)
		}
		result.Rows = append(result.Rows, row)
	}
	if result.Rows == nil {
		result.Rows = []ScopeOverviewRow{}
	}
	return result, rows.Err()
}

// ShortageTimeline returns shortage items split by scope start date availability.
func (r *Repository) ShortageTimeline(ctx context.Context, device, scope string) (ShortageTimeline, error) {
	// First get the scope's planned_start_at
	var scopeID string
	var plannedStart sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT ds.id, ds.planned_start_at
		FROM device_scopes ds
		LEFT JOIN devices d ON d.id = ds.device_id
		WHERE COALESCE(d.device_key, ds.device_key) = $1 AND ds.scope_key = $2
		LIMIT 1
	`, device, scope).Scan(&scopeID, &plannedStart)
	if err != nil {
		return ShortageTimeline{}, fmt.Errorf("find scope: %w", err)
	}

	result := ShortageTimeline{
		Device: device,
		Scope:  scope,
	}
	if plannedStart.Valid {
		result.PlannedStartAt = plannedStart.Time.Format(time.RFC3339)
	}

	// Get requirements for this scope
	const q = `
		SELECT
			sir.item_id,
			i.canonical_item_number,
			COALESCE(m.name, '') AS manufacturer,
			COALESCE(i.description, '') AS description,
			sir.quantity AS required_qty,
			COALESCE(ib.avail, 0) AS current_available,
			COALESCE(po_before.incoming, 0) AS incoming_before_start
		FROM scope_item_requirements sir
		JOIN items i ON i.id = sir.item_id
		LEFT JOIN manufacturers m ON m.key = i.manufacturer_key
		LEFT JOIN (
			SELECT item_id, SUM(available_quantity) AS avail
			FROM inventory_balances
			GROUP BY item_id
		) ib ON ib.item_id = sir.item_id
		LEFT JOIN (
			SELECT pl.item_id, SUM(pol.ordered_quantity - pol.received_quantity) AS incoming
			FROM purchase_order_lines pol
			JOIN procurement_lines pl ON pl.id = pol.procurement_line_id
			WHERE pol.status NOT IN ('cancelled', 'received')
			  AND pol.expected_arrival_date IS NOT NULL
			  AND ($3::timestamptz IS NULL OR pol.expected_arrival_date <= $3::date)
			GROUP BY pl.item_id
		) po_before ON po_before.item_id = sir.item_id
		WHERE sir.scope_id = $1
		  AND sir.quantity > COALESCE(ib.avail, 0) + COALESCE(po_before.incoming, 0)
		ORDER BY i.canonical_item_number
	`

	var startParam interface{} = nil
	if plannedStart.Valid {
		startParam = plannedStart.Time
	}

	rows, err := r.db.QueryContext(ctx, q, scopeID, device, startParam)
	if err != nil {
		return ShortageTimeline{}, fmt.Errorf("query shortage timeline: %w", err)
	}
	defer rows.Close()

	itemIDs := []string{}
	entryMap := map[string]*ShortageTimelineEntry{}
	for rows.Next() {
		var e ShortageTimelineEntry
		var currentAvail, incomingBefore int
		if err := rows.Scan(&e.ItemID, &e.ItemNumber, &e.Manufacturer, &e.Description,
			&e.RequiredQuantity, &currentAvail, &incomingBefore); err != nil {
			return ShortageTimeline{}, fmt.Errorf("scan shortage timeline: %w", err)
		}
		e.AvailableByStart = currentAvail + incomingBefore
		e.ShortageAtStart = e.RequiredQuantity - e.AvailableByStart
		if e.ShortageAtStart < 0 {
			e.ShortageAtStart = 0
		}
		e.DelayedArrivals = []DelayedArrival{}
		entryMap[e.ItemID] = &e
		itemIDs = append(itemIDs, e.ItemID)
	}
	if err := rows.Err(); err != nil {
		return ShortageTimeline{}, err
	}

	// Get delayed arrivals (after scope start)
	if len(itemIDs) > 0 && plannedStart.Valid {
		placeholders := make([]string, len(itemIDs))
		args := make([]interface{}, 0, len(itemIDs)+1)
		args = append(args, plannedStart.Time)
		for i, id := range itemIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
			args = append(args, id)
		}
		delayQ := fmt.Sprintf(`
			SELECT
				pl.item_id,
				pol.expected_arrival_date,
				pol.ordered_quantity - pol.received_quantity AS pending_qty,
				COALESCE(po.order_number, '') AS po_number,
				pol.id AS pol_id
			FROM purchase_order_lines pol
			JOIN purchase_orders po ON po.id = pol.purchase_order_id
			JOIN procurement_lines pl ON pl.id = pol.procurement_line_id
			WHERE pol.status NOT IN ('cancelled', 'received')
			  AND pol.expected_arrival_date > $1::date
			  AND pl.item_id IN (%s)
			ORDER BY pol.expected_arrival_date
		`, strings.Join(placeholders, ","))

		drows, err := r.db.QueryContext(ctx, delayQ, args...)
		if err != nil {
			return ShortageTimeline{}, fmt.Errorf("query delayed arrivals: %w", err)
		}
		defer drows.Close()
		for drows.Next() {
			var itemID, poNumber, polID string
			var arrDate time.Time
			var qty int
			if err := drows.Scan(&itemID, &arrDate, &qty, &poNumber, &polID); err != nil {
				return ShortageTimeline{}, fmt.Errorf("scan delayed arrival: %w", err)
			}
			if entry, ok := entryMap[itemID]; ok {
				entry.DelayedArrivals = append(entry.DelayedArrivals, DelayedArrival{
					ExpectedDate:        arrDate.Format("2006-01-02"),
					Quantity:            qty,
					PurchaseOrderNumber: poNumber,
					PurchaseOrderLineID: polID,
				})
			}
		}
	}

	for _, id := range itemIDs {
		if e, ok := entryMap[id]; ok {
			result.Rows = append(result.Rows, *e)
		}
	}
	if result.Rows == nil {
		result.Rows = []ShortageTimelineEntry{}
	}
	return result, nil
}

// EnhancedShortages returns shortages with procurement pipeline info.
func (r *Repository) EnhancedShortages(ctx context.Context, device, scope, coverageRule string) (EnhancedShortageList, error) {
	if coverageRule == "" {
		coverageRule = "approved"
	}
	const q = `
		WITH req AS (
			SELECT
				ds.device_key,
				ds.scope_key,
				sir.item_id,
				SUM(sir.quantity) AS required_qty
			FROM scope_item_requirements sir
			JOIN device_scopes ds ON ds.id = sir.scope_id
			LEFT JOIN devices d ON d.id = ds.device_id
			WHERE ($1 = '' OR COALESCE(d.device_key, ds.device_key) = $1)
			  AND ($2 = '' OR ds.scope_key = $2)
			GROUP BY ds.device_key, ds.scope_key, sir.item_id
		),
		res AS (
			SELECT
				ds.device_key,
				ds.scope_key,
				r.item_id,
				SUM(r.quantity) AS reserved_qty
			FROM reservations r
			JOIN device_scopes ds ON ds.id = r.device_scope_id
			LEFT JOIN devices d ON d.id = ds.device_id
			WHERE r.status NOT IN ('cancelled', 'released')
			  AND ($1 = '' OR COALESCE(d.device_key, ds.device_key) = $1)
			  AND ($2 = '' OR ds.scope_key = $2)
			GROUP BY ds.device_key, ds.scope_key, r.item_id
		),
		inv AS (
			SELECT item_id, SUM(available_quantity) AS avail
			FROM inventory_balances
			GROUP BY item_id
		),
		proc AS (
			SELECT
				pl.item_id,
				SUM(CASE WHEN psp.normalized_status IN ('submitted','under_review','approved','ordered','partially_received')
					THEN pl.requested_quantity ELSE 0 END) AS in_flow,
				SUM(COALESCE(po_qty.ordered, 0)) AS ordered,
				SUM(COALESCE(po_qty.received, 0)) AS received,
				STRING_AGG(DISTINCT NULLIF(COALESCE(psp.external_request_reference, pb.batch_number), ''), ',') AS related_refs
			FROM procurement_lines pl
			JOIN procurement_batches pb ON pb.id = pl.batch_id
			LEFT JOIN procurement_status_projections psp ON psp.batch_id = pb.id
			LEFT JOIN (
				SELECT procurement_line_id,
				       SUM(ordered_quantity) AS ordered,
				       SUM(received_quantity) AS received
				FROM purchase_order_lines
				WHERE status <> 'cancelled'
				GROUP BY procurement_line_id
			) po_qty ON po_qty.procurement_line_id = pl.id
			GROUP BY pl.item_id
		)
		SELECT
			req.device_key,
			req.scope_key,
			COALESCE(m.name, '') AS manufacturer,
			i.canonical_item_number,
			COALESCE(i.description, '') AS description,
			i.id AS item_id,
			req.required_qty,
			COALESCE(res.reserved_qty, 0) AS reserved_qty,
			COALESCE(inv.avail, 0) AS available_qty,
			COALESCE(proc.in_flow, 0) AS in_flow,
			COALESCE(proc.ordered, 0) AS ordered,
			COALESCE(proc.received, 0) AS received,
			COALESCE(proc.related_refs, '') AS related_refs
		FROM req
		JOIN items i ON i.id = req.item_id
		LEFT JOIN manufacturers m ON m.key = i.manufacturer_key
		LEFT JOIN res ON res.item_id = req.item_id AND res.device_key = req.device_key AND res.scope_key = req.scope_key
		LEFT JOIN inv ON inv.item_id = req.item_id
		LEFT JOIN proc ON proc.item_id = req.item_id
		ORDER BY req.device_key, req.scope_key, i.canonical_item_number
	`
	rows, err := r.db.QueryContext(ctx, q, device, scope)
	if err != nil {
		return EnhancedShortageList{}, fmt.Errorf("query enhanced shortages: %w", err)
	}
	defer rows.Close()

	result := EnhancedShortageList{CoverageRule: coverageRule}
	for rows.Next() {
		var row EnhancedShortageRow
		var relatedRefs string
		if err := rows.Scan(
			&row.Device, &row.Scope, &row.Manufacturer, &row.ItemNumber, &row.Description,
			&row.ItemID, &row.RequiredQuantity, &row.ReservedQuantity, &row.AvailableQuantity,
			&row.InRequestFlowQuantity, &row.OrderedQuantity, &row.ReceivedQuantity, &relatedRefs,
		); err != nil {
			return EnhancedShortageList{}, fmt.Errorf("scan enhanced shortage: %w", err)
		}
		row.RawShortage = row.RequiredQuantity - row.ReservedQuantity - row.AvailableQuantity
		if row.RawShortage < 0 {
			row.RawShortage = 0
		}
		// Apply coverage rule
		var covered int
		switch coverageRule {
		case "none":
			covered = 0
		case "submitted":
			covered = row.InRequestFlowQuantity
		case "approved":
			covered = row.OrderedQuantity + row.ReceivedQuantity
		case "ordered":
			covered = row.OrderedQuantity + row.ReceivedQuantity
		case "received":
			covered = row.ReceivedQuantity
		default:
			covered = row.OrderedQuantity + row.ReceivedQuantity
		}
		row.ActionableShortage = row.RawShortage - covered
		if row.ActionableShortage < 0 {
			row.ActionableShortage = 0
		}
		row.RelatedProcurementRequests = splitCSVRefs(relatedRefs)
		if row.RawShortage > 0 {
			result.Rows = append(result.Rows, row)
		}
	}
	if result.Rows == nil {
		result.Rows = []EnhancedShortageRow{}
	}
	return result, rows.Err()
}

// ReservationsExportCSV returns reservation data as CSV string.
func (r *Repository) ReservationsExportCSV(ctx context.Context, device, scope string) (string, error) {
	const q = `
		SELECT
			COALESCE(d.device_key, ds.device_key) AS device,
			ds.scope_key AS scope,
			COALESCE(m.name, '') AS manufacturer,
			i.canonical_item_number,
			COALESCE(i.description, '') AS description,
			rv.quantity,
			rv.status,
			rv.priority,
			rv.needed_by_at,
			COALESCE((
				SELECT STRING_AGG(DISTINCT ra.source_type, ',')
				FROM reservation_allocations ra WHERE ra.reservation_id = rv.id AND ra.status = 'allocated'
			), '') AS source_types
		FROM reservations rv
		JOIN items i ON i.id = rv.item_id
		JOIN device_scopes ds ON ds.id = rv.device_scope_id
		LEFT JOIN devices d ON d.id = ds.device_id
		LEFT JOIN manufacturers m ON m.key = i.manufacturer_key
		WHERE rv.status NOT IN ('cancelled', 'released')
		  AND ($1 = '' OR COALESCE(d.device_key, ds.device_key) = $1)
		  AND ($2 = '' OR ds.scope_key = $2)
		ORDER BY device, scope, i.canonical_item_number
	`
	rows, err := r.db.QueryContext(ctx, q, device, scope)
	if err != nil {
		return "", fmt.Errorf("query reservations export: %w", err)
	}
	defer rows.Close()

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"device", "scope", "manufacturer", "item_number", "description", "quantity", "status", "priority", "needed_by", "source_type"})
	for rows.Next() {
		var dev, sc, mfr, item, desc, status, priority, srcTypes string
		var qty int
		var neededBy sql.NullTime
		if err := rows.Scan(&dev, &sc, &mfr, &item, &desc, &qty, &status, &priority, &neededBy, &srcTypes); err != nil {
			return "", fmt.Errorf("scan reservations export: %w", err)
		}
		nb := ""
		if neededBy.Valid {
			nb = neededBy.Time.Format("2006-01-02")
		}
		_ = w.Write([]string{dev, sc, mfr, item, desc, strconv.Itoa(qty), status, priority, nb, srcTypes})
	}
	w.Flush()
	return buf.String(), rows.Err()
}

// RequirementsExportCSV returns requirements data as CSV string.
func (r *Repository) RequirementsExportCSV(ctx context.Context, device, scope string) (string, error) {
	const q = `
		SELECT
			COALESCE(d.device_key, ds.device_key) AS device,
			ds.scope_key AS scope,
			COALESCE(m.name, '') AS manufacturer,
			i.canonical_item_number,
			COALESCE(i.description, '') AS description,
			sir.quantity,
			sir.needed_by_at,
			COALESCE(sir.note, '') AS note
		FROM scope_item_requirements sir
		JOIN device_scopes ds ON ds.id = sir.scope_id
		JOIN items i ON i.id = sir.item_id
		LEFT JOIN devices d ON d.id = ds.device_id
		LEFT JOIN manufacturers m ON m.key = i.manufacturer_key
		WHERE ($1 = '' OR COALESCE(d.device_key, ds.device_key) = $1)
		  AND ($2 = '' OR ds.scope_key = $2)
		ORDER BY device, scope, i.canonical_item_number
	`
	rows, err := r.db.QueryContext(ctx, q, device, scope)
	if err != nil {
		return "", fmt.Errorf("query requirements export: %w", err)
	}
	defer rows.Close()

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"device", "scope", "manufacturer", "item_number", "description", "quantity", "needed_by_at", "note"})
	for rows.Next() {
		var dev, sc, mfr, item, desc, note string
		var qty int
		var neededBy sql.NullTime
		if err := rows.Scan(&dev, &sc, &mfr, &item, &desc, &qty, &neededBy, &note); err != nil {
			return "", fmt.Errorf("scan requirements export: %w", err)
		}
		_ = w.Write([]string{dev, sc, mfr, item, desc, strconv.Itoa(qty), nullableDateString(neededBy), note})
	}
	w.Flush()
	return buf.String(), rows.Err()
}

// RequirementsImportPreview parses a requirements CSV and checks for issues.
func (r *Repository) RequirementsImportPreview(ctx context.Context, fileName string, data []byte) (RequirementsImportPreview, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		return RequirementsImportPreview{}, fmt.Errorf("parse CSV: %w", err)
	}
	if len(records) < 2 {
		return RequirementsImportPreview{}, fmt.Errorf("CSV must have a header row and at least one data row")
	}

	result := RequirementsImportPreview{FileName: fileName}
	headerIndex := map[string]int{}
	for idx, header := range records[0] {
		headerIndex[strings.ToLower(strings.TrimSpace(header))] = idx
	}
	field := func(rec []string, names ...string) string {
		for _, name := range names {
			if idx, ok := headerIndex[name]; ok && idx < len(rec) {
				return strings.TrimSpace(rec[idx])
			}
		}
		return ""
	}
	for i, rec := range records[1:] {
		row := RequirementsImportPreviewRow{RowNumber: i + 2}
		if len(rec) == 0 {
			row.Status = "error"
			row.Message = "empty row"
			result.Rows = append(result.Rows, row)
			continue
		}
		row.DeviceKey = field(rec, "device", "device_key")
		row.ScopeKey = field(rec, "scope", "scope_key")
		row.Manufacturer = field(rec, "manufacturer", "manufacturer_name")
		row.ItemNumber = field(rec, "item_number", "canonical_item_number")
		row.Description = field(rec, "description", "item_description")
		row.NeededByAt = field(rec, "needed_by_at", "needed_by", "neededByAt")
		qtyRaw := field(rec, "quantity", "required_quantity")
		if row.DeviceKey == "" || row.ScopeKey == "" || row.ItemNumber == "" || qtyRaw == "" {
			row.Status = "error"
			row.Message = "required columns are device, scope, item_number, quantity"
			result.Rows = append(result.Rows, row)
			continue
		}
		qty, err := strconv.Atoi(qtyRaw)
		if err != nil || qty <= 0 {
			row.Status = "error"
			row.Message = "invalid quantity"
			result.Rows = append(result.Rows, row)
			continue
		}
		row.Quantity = qty
		if row.NeededByAt != "" {
			if _, err := time.Parse("2006-01-02", row.NeededByAt); err != nil {
				row.Status = "error"
				row.Message = "invalid needed_by_at (expected YYYY-MM-DD)"
				result.Rows = append(result.Rows, row)
				continue
			}
		}

		// Check scope exists
		var scopeID string
		err = r.db.QueryRowContext(ctx, `
			SELECT ds.id FROM device_scopes ds
			LEFT JOIN devices d ON d.id = ds.device_id
			WHERE COALESCE(d.device_key, ds.device_key) = $1 AND ds.scope_key = $2
		`, row.DeviceKey, row.ScopeKey).Scan(&scopeID)
		if err != nil {
			row.Status = "error"
			row.Message = fmt.Sprintf("scope not found: %s/%s", row.DeviceKey, row.ScopeKey)
			result.Rows = append(result.Rows, row)
			continue
		}
		row.ScopeID = scopeID

		// Check item exists
		var itemID string
		err = r.db.QueryRowContext(ctx, `
			SELECT id FROM items WHERE canonical_item_number = $1
		`, row.ItemNumber).Scan(&itemID)
		if err != nil {
			row.Status = "warning"
			row.Message = "item not registered in master"
			row.ItemRegistered = false
		} else {
			row.ItemID = itemID
			row.ItemRegistered = true
			row.Status = "valid"
		}
		result.Rows = append(result.Rows, row)
	}
	if result.Rows == nil {
		result.Rows = []RequirementsImportPreviewRow{}
	}
	return result, nil
}

// RequirementsImportApply applies a requirements CSV import.
func (r *Repository) RequirementsImportApply(ctx context.Context, fileName string, data []byte) (RequirementsImportResult, error) {
	preview, err := r.RequirementsImportPreview(ctx, fileName, data)
	if err != nil {
		return RequirementsImportResult{}, err
	}

	var result RequirementsImportResult
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return RequirementsImportResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, row := range preview.Rows {
		if row.Status == "error" || !row.ItemRegistered {
			result.Skipped++
			continue
		}
		id := uuid.New().String()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO scope_item_requirements (id, scope_id, item_id, quantity, needed_by_at, note)
			VALUES ($1, $2, $3, $4, NULLIF($5, '')::timestamptz, '')
			ON CONFLICT (scope_id, item_id)
			DO UPDATE SET quantity = scope_item_requirements.quantity + EXCLUDED.quantity,
			             needed_by_at = COALESCE(EXCLUDED.needed_by_at, scope_item_requirements.needed_by_at),
			             updated_at = NOW()
		`, id, row.ScopeID, row.ItemID, row.Quantity, row.NeededByAt)
		if err != nil {
			result.Errored++
			continue
		}
		result.Created++
	}

	if err := tx.Commit(); err != nil {
		return RequirementsImportResult{}, fmt.Errorf("commit: %w", err)
	}
	return result, nil
}

type batchCSVRow struct {
	RowNumber int
	Raw       map[string]string
}

func parseBatchCSV(data []byte) ([]batchCSVRow, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must have a header row and at least one data row")
	}
	headers := normalizeCSVHeaders(records[0])
	rows := make([]batchCSVRow, 0, len(records)-1)
	for index, record := range records[1:] {
		rows = append(rows, batchCSVRow{
			RowNumber: index + 2,
			Raw:       csvRecord(headers, record),
		})
	}
	return rows, nil
}

func rowValue(row map[string]string, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(row[name]); value != "" {
			return value
		}
	}
	return ""
}

func positiveRowInt(row map[string]string, names ...string) (int, bool) {
	raw := rowValue(row, names...)
	value, err := strconv.Atoi(raw)
	return value, err == nil && value > 0
}

func anyRowInt(row map[string]string, names ...string) (int, bool) {
	raw := rowValue(row, names...)
	value, err := strconv.Atoi(raw)
	return value, err == nil
}

func (r *Repository) resolveBatchScopeID(ctx context.Context, raw map[string]string) string {
	if id := rowValue(raw, "device_scope_id", "scope_id"); id != "" {
		return id
	}
	device := rowValue(raw, "device", "device_key")
	scope := rowValue(raw, "scope", "scope_key")
	if device == "" || scope == "" {
		return ""
	}
	var id string
	_ = r.db.QueryRowContext(ctx, `
		SELECT ds.id
		FROM device_scopes ds
		LEFT JOIN devices d ON d.id = ds.device_id
		WHERE COALESCE(d.device_key, ds.device_key) = $1 AND ds.scope_key = $2
	`, device, scope).Scan(&id)
	return id
}

func (r *Repository) resolveBatchItemID(ctx context.Context, raw map[string]string) string {
	if id := rowValue(raw, "item_id"); id != "" {
		return id
	}
	itemNumber := rowValue(raw, "item_number", "canonical_item_number")
	if itemNumber == "" {
		return ""
	}
	var id string
	_ = r.db.QueryRowContext(ctx, `SELECT id FROM items WHERE canonical_item_number = $1`, itemNumber).Scan(&id)
	return id
}

func batchPreviewResult(importType, fileName string, rows []ImportPreviewRow) ImportPreviewResult {
	status := "ready"
	for _, row := range rows {
		if row.Status == "invalid" {
			status = "has_errors"
			break
		}
	}
	return ImportPreviewResult{ImportType: importType, FileName: fileName, Status: status, Rows: rows}
}

func rawPayload(raw map[string]string) string {
	payload, _ := json.Marshal(raw)
	return string(payload)
}

func (r *Repository) createCSVImportJobTx(ctx context.Context, tx *sql.Tx, importType, fileName, actor string, preview ImportPreviewResult, summary map[string]int) (string, map[int]string, error) {
	jobID := fmt.Sprintf("imp-%d", time.Now().UnixNano())
	summaryBytes, _ := json.Marshal(summary)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO import_jobs (id, import_type, status, lifecycle_state, file_name, summary, created_by, created_at, updated_at)
		VALUES ($1, $2, 'completed', 'applied', $3, $4::jsonb, $5, NOW(), NOW())
	`, jobID, importType, fileName, string(summaryBytes), defaultString(actor, "local-user")); err != nil {
		return "", nil, fmt.Errorf("insert import job: %w", err)
	}
	rowIDs := map[int]string{}
	for _, row := range preview.Rows {
		rowID := fmt.Sprintf("import-row-%d", time.Now().UnixNano())
		rowIDs[row.RowNumber] = rowID
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO import_rows (id, import_job_id, row_number, raw_payload, normalized_payload, status, code, message, created_at)
			VALUES ($1, $2, $3, $4::jsonb, $4::jsonb, $5, $6, $7, NOW())
		`, rowID, jobID, row.RowNumber, rawPayload(row.Raw), row.Status, row.Code, row.Message); err != nil {
			return "", nil, fmt.Errorf("insert import row: %w", err)
		}
	}
	return jobID, rowIDs, nil
}

func insertCSVImportEffectTx(ctx context.Context, tx *sql.Tx, jobID, rowID, entityType, entityID, effectType string, beforeState, afterState any) error {
	beforeBytes, _ := json.Marshal(beforeState)
	afterBytes, _ := json.Marshal(afterState)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO import_effects (id, import_job_id, import_row_id, target_entity_type, target_entity_id, effect_type, before_state, after_state, created_at)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7::jsonb, $8::jsonb, NOW())
	`, fmt.Sprintf("import-effect-%d", time.Now().UnixNano()), jobID, rowID, entityType, entityID, effectType, string(beforeBytes), string(afterBytes)); err != nil {
		return fmt.Errorf("insert import effect: %w", err)
	}
	return nil
}

func importSummaryPayload(summary map[string]int) map[string]any {
	payload := map[string]any{}
	for key, value := range summary {
		payload[key] = value
	}
	return payload
}

func (r *Repository) ReservationImportPreview(ctx context.Context, fileName string, data []byte) (ImportPreviewResult, error) {
	rows, err := parseBatchCSV(data)
	if err != nil {
		return ImportPreviewResult{}, err
	}
	previewRows := make([]ImportPreviewRow, 0, len(rows))
	for _, parsed := range rows {
		row := ImportPreviewRow{RowNumber: parsed.RowNumber, Status: "valid", Code: "reservation_create", Raw: parsed.Raw}
		itemID := r.resolveBatchItemID(ctx, parsed.Raw)
		scopeID := r.resolveBatchScopeID(ctx, parsed.Raw)
		qty, qtyOK := positiveRowInt(parsed.Raw, "quantity")
		switch {
		case itemID == "":
			row.Status, row.Code, row.Message = "invalid", "item_not_found", "item_id or item_number must resolve to an item"
		case scopeID == "":
			row.Status, row.Code, row.Message = "invalid", "scope_not_found", "device_scope_id or device/scope must resolve to a scope"
		case !qtyOK:
			row.Status, row.Code, row.Message = "invalid", "invalid_quantity", "quantity must be a positive integer"
		default:
			row.Raw["item_id"] = itemID
			row.Raw["device_scope_id"] = scopeID
			row.Raw["quantity"] = strconv.Itoa(qty)
		}
		previewRows = append(previewRows, row)
	}
	return batchPreviewResult("reservations", fileName, previewRows), nil
}

func (r *Repository) ReservationImportApply(ctx context.Context, fileName string, data []byte, actor string) (CSVImportApplyResult, error) {
	preview, err := r.ReservationImportPreview(ctx, fileName, data)
	if err != nil {
		return CSVImportApplyResult{}, err
	}
	if preview.Status == "has_errors" {
		return CSVImportApplyResult{}, fmt.Errorf("CSV contains invalid reservation rows")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CSVImportApplyResult{}, fmt.Errorf("begin reservation import tx: %w", err)
	}
	defer tx.Rollback()

	summary := map[string]int{"created": 0, "updated": 0, "skipped": 0, "errored": 0}
	jobID, rowIDs, err := r.createCSVImportJobTx(ctx, tx, "reservations", fileName, actor, preview, summary)
	if err != nil {
		return CSVImportApplyResult{}, err
	}
	for _, row := range preview.Rows {
		qty, _ := positiveRowInt(row.Raw, "quantity")
		reservationID := fmt.Sprintf("res-%d", time.Now().UnixNano())
		requestedBy := defaultString(rowValue(row.Raw, "requested_by"), actor)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reservations (
				id, item_id, device_scope_id, quantity, status, requested_by, note, purpose, priority,
				needed_by_at, planned_use_at, hold_until_at, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, 'requested', $5, $6, $7, $8, NULLIF($9, '')::timestamptz, NULLIF($10, '')::timestamptz, NULLIF($11, '')::timestamptz, NOW(), NOW())
		`, reservationID, row.Raw["item_id"], row.Raw["device_scope_id"], qty, requestedBy, rowValue(row.Raw, "note"), rowValue(row.Raw, "purpose"), defaultString(rowValue(row.Raw, "priority"), "normal"), rowValue(row.Raw, "needed_by_at"), rowValue(row.Raw, "planned_use_at"), rowValue(row.Raw, "hold_until_at")); err != nil {
			return CSVImportApplyResult{}, fmt.Errorf("insert reservation row %d: %w", row.RowNumber, err)
		}
		if err := r.recordReservationEventTx(ctx, tx, reservationID, "requested", qty, requestedBy, map[string]any{"note": rowValue(row.Raw, "note")}); err != nil {
			return CSVImportApplyResult{}, err
		}
		if err := insertCSVImportEffectTx(ctx, tx, jobID, rowIDs[row.RowNumber], "reservation", reservationID, "insert", map[string]any{}, row.Raw); err != nil {
			return CSVImportApplyResult{}, err
		}
		summary["created"]++
	}
	summaryBytes, _ := json.Marshal(summary)
	if _, err := tx.ExecContext(ctx, `UPDATE import_jobs SET summary = $2::jsonb WHERE id = $1`, jobID, string(summaryBytes)); err != nil {
		return CSVImportApplyResult{}, fmt.Errorf("update import summary: %w", err)
	}
	if err := recordAuditEventTx(ctx, tx, actor, "reservations.imported", "import_job", jobID, importSummaryPayload(summary)); err != nil {
		return CSVImportApplyResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return CSVImportApplyResult{}, fmt.Errorf("commit reservation import tx: %w", err)
	}
	detail, err := r.ImportDetail(ctx, jobID)
	if err != nil {
		return CSVImportApplyResult{}, err
	}
	return CSVImportApplyResult{JobID: jobID, Created: summary["created"], Detail: detail}, nil
}

func (r *Repository) AllocationImportPreview(ctx context.Context, fileName string, data []byte) (ImportPreviewResult, error) {
	rows, err := parseBatchCSV(data)
	if err != nil {
		return ImportPreviewResult{}, err
	}
	previewRows := make([]ImportPreviewRow, 0, len(rows))
	for _, parsed := range rows {
		row := ImportPreviewRow{RowNumber: parsed.RowNumber, Status: "valid", Code: "reservation_allocate", Raw: parsed.Raw}
		reservationID := rowValue(parsed.Raw, "reservation_id")
		location := rowValue(parsed.Raw, "location_code")
		qty, qtyOK := positiveRowInt(parsed.Raw, "quantity")
		var itemID string
		var available int
		if reservationID != "" && location != "" {
			_ = r.db.QueryRowContext(ctx, `
				SELECT r.item_id, COALESCE(ib.available_quantity, 0)
				FROM reservations r
				LEFT JOIN inventory_balances ib ON ib.item_id = r.item_id AND ib.location_code = $2
				WHERE r.id = $1
			`, reservationID, location).Scan(&itemID, &available)
		}
		switch {
		case reservationID == "":
			row.Status, row.Code, row.Message = "invalid", "missing_reservation_id", "reservation_id is required"
		case itemID == "":
			row.Status, row.Code, row.Message = "invalid", "reservation_not_found", "reservation_id was not found"
		case location == "":
			row.Status, row.Code, row.Message = "invalid", "missing_location", "location_code is required"
		case !qtyOK:
			row.Status, row.Code, row.Message = "invalid", "invalid_quantity", "quantity must be a positive integer"
		case available < qty:
			row.Status, row.Code, row.Message = "invalid", "insufficient_available_quantity", fmt.Sprintf("available quantity at %s is %d", location, available)
		}
		previewRows = append(previewRows, row)
	}
	return batchPreviewResult("reservation_allocations", fileName, previewRows), nil
}

func (r *Repository) AllocationImportApply(ctx context.Context, fileName string, data []byte, actor string) (CSVImportApplyResult, error) {
	preview, err := r.AllocationImportPreview(ctx, fileName, data)
	if err != nil {
		return CSVImportApplyResult{}, err
	}
	if preview.Status == "has_errors" {
		return CSVImportApplyResult{}, fmt.Errorf("CSV contains invalid allocation rows")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CSVImportApplyResult{}, fmt.Errorf("begin allocation import tx: %w", err)
	}
	defer tx.Rollback()
	summary := map[string]int{"created": 0, "updated": 0, "skipped": 0, "errored": 0}
	jobID, rowIDs, err := r.createCSVImportJobTx(ctx, tx, "reservation_allocations", fileName, actor, preview, summary)
	if err != nil {
		return CSVImportApplyResult{}, err
	}
	for _, row := range preview.Rows {
		qty, _ := positiveRowInt(row.Raw, "quantity")
		detail, err := r.reservationDetailTx(ctx, tx, rowValue(row.Raw, "reservation_id"))
		if err != nil {
			return CSVImportApplyResult{}, err
		}
		allocationID := fmt.Sprintf("alloc-%d", time.Now().UnixNano())
		location := rowValue(row.Raw, "location_code")
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reservation_allocations (id, reservation_id, item_id, location_code, quantity, status, allocated_at, note)
			VALUES ($1, $2, $3, $4, $5, 'allocated', NOW(), $6)
		`, allocationID, detail.ID, detail.ItemID, location, qty, rowValue(row.Raw, "note")); err != nil {
			return CSVImportApplyResult{}, fmt.Errorf("insert allocation row %d: %w", row.RowNumber, err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO inventory_events (
				id, item_id, location_code, from_location_code, to_location_code, event_type, quantity_delta, note, device_scope_id,
				actor_id, source_type, source_id, correlation_id, occurred_at
			) VALUES ($1, $2, $3, $4, '', 'reserve_allocate', 0, $5, $6, $7, 'reservation', $8, $9, NOW())
		`, fmt.Sprintf("evt-%d", time.Now().UnixNano()), detail.ItemID, location, location, rowValue(row.Raw, "note"), detail.DeviceScopeID, defaultString(rowValue(row.Raw, "actor_id"), actor), detail.ID, allocationID); err != nil {
			return CSVImportApplyResult{}, fmt.Errorf("insert allocation inventory event row %d: %w", row.RowNumber, err)
		}
		if err := adjustReservedQuantityTx(ctx, tx, detail.ItemID, location, qty); err != nil {
			return CSVImportApplyResult{}, err
		}
		if err := r.recordReservationEventTx(ctx, tx, detail.ID, "allocated", qty, actor, map[string]any{"locationCode": location, "allocationId": allocationID, "note": rowValue(row.Raw, "note")}); err != nil {
			return CSVImportApplyResult{}, err
		}
		if err := r.refreshReservationStatusTx(ctx, tx, detail.ID); err != nil {
			return CSVImportApplyResult{}, err
		}
		if err := insertCSVImportEffectTx(ctx, tx, jobID, rowIDs[row.RowNumber], "reservation_allocation", allocationID, "insert", map[string]any{}, row.Raw); err != nil {
			return CSVImportApplyResult{}, err
		}
		summary["created"]++
	}
	summaryBytes, _ := json.Marshal(summary)
	if _, err := tx.ExecContext(ctx, `UPDATE import_jobs SET summary = $2::jsonb WHERE id = $1`, jobID, string(summaryBytes)); err != nil {
		return CSVImportApplyResult{}, fmt.Errorf("update import summary: %w", err)
	}
	if err := recordAuditEventTx(ctx, tx, actor, "reservation_allocations.imported", "import_job", jobID, importSummaryPayload(summary)); err != nil {
		return CSVImportApplyResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return CSVImportApplyResult{}, fmt.Errorf("commit allocation import tx: %w", err)
	}
	detail, err := r.ImportDetail(ctx, jobID)
	if err != nil {
		return CSVImportApplyResult{}, err
	}
	return CSVImportApplyResult{JobID: jobID, Created: summary["created"], Detail: detail}, nil
}

func (r *Repository) InventoryOperationImportPreview(ctx context.Context, operation, fileName string, data []byte) (ImportPreviewResult, error) {
	rows, err := parseBatchCSV(data)
	if err != nil {
		return ImportPreviewResult{}, err
	}
	previewRows := make([]ImportPreviewRow, 0, len(rows))
	for _, parsed := range rows {
		row := ImportPreviewRow{RowNumber: parsed.RowNumber, Status: "valid", Code: "inventory_" + operation, Raw: parsed.Raw}
		itemID := r.resolveBatchItemID(ctx, parsed.Raw)
		scopeID := r.resolveBatchScopeID(ctx, parsed.Raw)
		if itemID == "" {
			row.Status, row.Code, row.Message = "invalid", "item_not_found", "item_id or item_number must resolve to an item"
		} else if scopeID == "" {
			row.Status, row.Code, row.Message = "invalid", "scope_not_found", "device_scope_id or device/scope must resolve to a scope"
		} else {
			row.Raw["item_id"] = itemID
			row.Raw["device_scope_id"] = scopeID
			switch operation {
			case "adjust":
				if _, ok := anyRowInt(parsed.Raw, "quantity_delta"); !ok {
					row.Status, row.Code, row.Message = "invalid", "invalid_quantity_delta", "quantity_delta must be an integer"
				} else if rowValue(parsed.Raw, "location_code") == "" {
					row.Status, row.Code, row.Message = "invalid", "missing_location", "location_code is required"
				}
			case "receive":
				if _, ok := positiveRowInt(parsed.Raw, "quantity"); !ok {
					row.Status, row.Code, row.Message = "invalid", "invalid_quantity", "quantity must be a positive integer"
				} else if rowValue(parsed.Raw, "location_code") == "" {
					row.Status, row.Code, row.Message = "invalid", "missing_location", "location_code is required"
				}
			case "move":
				if _, ok := positiveRowInt(parsed.Raw, "quantity"); !ok {
					row.Status, row.Code, row.Message = "invalid", "invalid_quantity", "quantity must be a positive integer"
				} else if rowValue(parsed.Raw, "from_location_code") == "" || rowValue(parsed.Raw, "to_location_code") == "" {
					row.Status, row.Code, row.Message = "invalid", "missing_location", "from_location_code and to_location_code are required"
				}
			default:
				return ImportPreviewResult{}, fmt.Errorf("unsupported inventory operation: %s", operation)
			}
		}
		previewRows = append(previewRows, row)
	}
	return batchPreviewResult("inventory_"+operation, fileName, previewRows), nil
}

func (r *Repository) InventoryOperationImportApply(ctx context.Context, operation, fileName string, data []byte, actor string) (CSVImportApplyResult, error) {
	preview, err := r.InventoryOperationImportPreview(ctx, operation, fileName, data)
	if err != nil {
		return CSVImportApplyResult{}, err
	}
	if preview.Status == "has_errors" {
		return CSVImportApplyResult{}, fmt.Errorf("CSV contains invalid inventory operation rows")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CSVImportApplyResult{}, fmt.Errorf("begin inventory operation import tx: %w", err)
	}
	defer tx.Rollback()
	summary := map[string]int{"created": 0, "updated": 0, "skipped": 0, "errored": 0}
	jobID, rowIDs, err := r.createCSVImportJobTx(ctx, tx, "inventory_"+operation, fileName, actor, preview, summary)
	if err != nil {
		return CSVImportApplyResult{}, err
	}
	for _, row := range preview.Rows {
		var event InventoryEventEntry
		switch operation {
		case "adjust":
			delta, _ := anyRowInt(row.Raw, "quantity_delta")
			if delta > 0 {
				if err := incrementBalanceTx(ctx, tx, row.Raw["item_id"], rowValue(row.Raw, "location_code"), delta); err != nil {
					return CSVImportApplyResult{}, err
				}
			} else {
				if err := decrementBalanceTx(ctx, tx, row.Raw["item_id"], rowValue(row.Raw, "location_code"), -delta); err != nil {
					return CSVImportApplyResult{}, err
				}
			}
			event, err = insertInventoryEventTx(ctx, tx, inventoryEventInsert{ItemID: row.Raw["item_id"], LocationCode: rowValue(row.Raw, "location_code"), FromLocationCode: rowValue(row.Raw, "location_code"), ToLocationCode: rowValue(row.Raw, "location_code"), EventType: "adjust", QuantityDelta: delta, DeviceScopeID: row.Raw["device_scope_id"], ActorID: actor, SourceType: "manual", SourceID: rowValue(row.Raw, "source_id"), Note: rowValue(row.Raw, "note")})
		case "receive":
			qty, _ := positiveRowInt(row.Raw, "quantity")
			event, err = insertInventoryEventTx(ctx, tx, inventoryEventInsert{ItemID: row.Raw["item_id"], LocationCode: rowValue(row.Raw, "location_code"), ToLocationCode: rowValue(row.Raw, "location_code"), EventType: "receive", QuantityDelta: qty, DeviceScopeID: row.Raw["device_scope_id"], ActorID: actor, SourceType: defaultString(rowValue(row.Raw, "source_type"), "manual"), SourceID: rowValue(row.Raw, "source_id"), Note: rowValue(row.Raw, "note")})
			if err == nil {
				err = incrementBalanceTx(ctx, tx, row.Raw["item_id"], rowValue(row.Raw, "location_code"), qty)
			}
		case "move":
			qty, _ := positiveRowInt(row.Raw, "quantity")
			if err = decrementBalanceTx(ctx, tx, row.Raw["item_id"], rowValue(row.Raw, "from_location_code"), qty); err == nil {
				err = incrementBalanceTx(ctx, tx, row.Raw["item_id"], rowValue(row.Raw, "to_location_code"), qty)
			}
			if err == nil {
				event, err = insertInventoryEventTx(ctx, tx, inventoryEventInsert{ItemID: row.Raw["item_id"], LocationCode: rowValue(row.Raw, "to_location_code"), FromLocationCode: rowValue(row.Raw, "from_location_code"), ToLocationCode: rowValue(row.Raw, "to_location_code"), EventType: "move", QuantityDelta: qty, DeviceScopeID: row.Raw["device_scope_id"], ActorID: actor, SourceType: defaultString(rowValue(row.Raw, "source_type"), "manual"), SourceID: rowValue(row.Raw, "source_id"), CorrelationSource: fmt.Sprintf("%s->%s", rowValue(row.Raw, "from_location_code"), rowValue(row.Raw, "to_location_code")), Note: rowValue(row.Raw, "note")})
			}
		}
		if err != nil {
			return CSVImportApplyResult{}, fmt.Errorf("apply inventory %s row %d: %w", operation, row.RowNumber, err)
		}
		if err := insertCSVImportEffectTx(ctx, tx, jobID, rowIDs[row.RowNumber], "inventory_event", event.ID, "insert", map[string]any{}, row.Raw); err != nil {
			return CSVImportApplyResult{}, err
		}
		summary["created"]++
	}
	summaryBytes, _ := json.Marshal(summary)
	if _, err := tx.ExecContext(ctx, `UPDATE import_jobs SET summary = $2::jsonb WHERE id = $1`, jobID, string(summaryBytes)); err != nil {
		return CSVImportApplyResult{}, fmt.Errorf("update import summary: %w", err)
	}
	if err := recordAuditEventTx(ctx, tx, actor, "inventory_operations.imported", "import_job", jobID, importSummaryPayload(summary)); err != nil {
		return CSVImportApplyResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return CSVImportApplyResult{}, fmt.Errorf("commit inventory operation import tx: %w", err)
	}
	detail, err := r.ImportDetail(ctx, jobID)
	if err != nil {
		return CSVImportApplyResult{}, err
	}
	return CSVImportApplyResult{JobID: jobID, Created: summary["created"], Detail: detail}, nil
}

// BulkReservationPreview generates allocation plan from requirements.
func (r *Repository) BulkReservationPreview(ctx context.Context, scopeID string) (BulkReservationPreview, error) {
	const q = `
		SELECT
			sir.item_id,
			i.canonical_item_number,
			COALESCE(m.name, '') AS manufacturer,
			COALESCE(i.description, '') AS description,
			sir.quantity AS required_qty,
			sir.needed_by_at,
			COALESCE(existing_res.reserved, 0) AS already_reserved
		FROM scope_item_requirements sir
		JOIN items i ON i.id = sir.item_id
		LEFT JOIN manufacturers m ON m.key = i.manufacturer_key
		LEFT JOIN (
			SELECT item_id, device_scope_id, SUM(quantity) AS reserved
			FROM reservations
			WHERE status NOT IN ('cancelled', 'released')
			GROUP BY item_id, device_scope_id
		) existing_res ON existing_res.item_id = sir.item_id AND existing_res.device_scope_id = sir.scope_id
		WHERE sir.scope_id = $1
		  AND sir.quantity > COALESCE(existing_res.reserved, 0)
		ORDER BY i.canonical_item_number
	`
	rows, err := r.db.QueryContext(ctx, q, scopeID)
	if err != nil {
		return BulkReservationPreview{}, fmt.Errorf("query bulk preview: %w", err)
	}
	defer rows.Close()

	result := BulkReservationPreview{ScopeID: scopeID}
	for rows.Next() {
		var row BulkReservationPreviewRow
		var alreadyReserved int
		var neededBy sql.NullTime
		if err := rows.Scan(&row.ItemID, &row.ItemNumber, &row.Manufacturer, &row.Description,
			&row.RequiredQuantity, &neededBy, &alreadyReserved); err != nil {
			return BulkReservationPreview{}, fmt.Errorf("scan bulk preview: %w", err)
		}
		row.NeededByAt = nullableDateString(neededBy)
		needed := row.RequiredQuantity - alreadyReserved
		row.RequiredQuantity = needed
		row.AllocFromStockLocs = []StockAllocation{}
		row.AllocFromOrderLocs = []OrderAllocation{}

		// Check available stock
		balRows, err := r.db.QueryContext(ctx, `
			SELECT location_code, available_quantity
			FROM inventory_balances
			WHERE item_id = $1 AND available_quantity > 0
			ORDER BY available_quantity DESC
		`, row.ItemID)
		if err != nil {
			return BulkReservationPreview{}, fmt.Errorf("query balance for %s: %w", row.ItemID, err)
		}
		remaining := needed
		for balRows.Next() {
			var loc string
			var avail int
			if err := balRows.Scan(&loc, &avail); err != nil {
				balRows.Close()
				return BulkReservationPreview{}, err
			}
			alloc := avail
			if alloc > remaining {
				alloc = remaining
			}
			if alloc > 0 {
				row.AllocFromStock += alloc
				row.AllocFromStockLocs = append(row.AllocFromStockLocs, StockAllocation{
					LocationCode: loc,
					Quantity:     alloc,
				})
				remaining -= alloc
			}
			if remaining <= 0 {
				break
			}
		}
		balRows.Close()

		// Check incoming orders
		if remaining > 0 && neededBy.Valid {
			polRows, err := r.db.QueryContext(ctx, `
				SELECT pol.id, po.order_number, pol.expected_arrival_date,
				       pol.ordered_quantity - pol.received_quantity - COALESCE(existing_alloc.allocated, 0) AS pending
				FROM purchase_order_lines pol
				JOIN purchase_orders po ON po.id = pol.purchase_order_id
				JOIN procurement_lines pl ON pl.id = pol.procurement_line_id
				LEFT JOIN (
					SELECT ra.purchase_order_line_id, SUM(ra.quantity) AS allocated
					FROM reservation_allocations ra
					JOIN reservations rv ON rv.id = ra.reservation_id
					WHERE ra.status = 'allocated'
					  AND ra.source_type = 'incoming_order'
					  AND rv.status NOT IN ('cancelled', 'released')
					GROUP BY ra.purchase_order_line_id
				) existing_alloc ON existing_alloc.purchase_order_line_id = pol.id
				WHERE pl.item_id = $1
				  AND pol.status NOT IN ('cancelled', 'received')
				  AND pol.expected_arrival_date IS NOT NULL
				  AND pol.expected_arrival_date <= $2::date
				  AND (pol.ordered_quantity - pol.received_quantity - COALESCE(existing_alloc.allocated, 0)) > 0
				ORDER BY pol.expected_arrival_date NULLS LAST
			`, row.ItemID, neededBy.Time)
			if err != nil {
				return BulkReservationPreview{}, fmt.Errorf("query PO lines for %s: %w", row.ItemID, err)
			}
			for polRows.Next() {
				var polID, poNum string
				var arrDate sql.NullTime
				var pending int
				if err := polRows.Scan(&polID, &poNum, &arrDate, &pending); err != nil {
					polRows.Close()
					return BulkReservationPreview{}, err
				}
				alloc := pending
				if alloc > remaining {
					alloc = remaining
				}
				arr := ""
				if arrDate.Valid {
					arr = arrDate.Time.Format("2006-01-02")
				}
				row.AllocFromOrders += alloc
				row.AllocFromOrderLocs = append(row.AllocFromOrderLocs, OrderAllocation{
					PurchaseOrderLineID: polID,
					PurchaseOrderNumber: poNum,
					ExpectedArrival:     arr,
					Quantity:            alloc,
				})
				remaining -= alloc
				if remaining <= 0 {
					break
				}
			}
			polRows.Close()
		}

		row.Unallocated = remaining
		if row.Unallocated < 0 {
			row.Unallocated = 0
		}
		result.Rows = append(result.Rows, row)
	}
	if result.Rows == nil {
		result.Rows = []BulkReservationPreviewRow{}
	}
	return result, rows.Err()
}

// BulkReservationConfirm creates reservations from confirmed bulk preview.
func (r *Repository) BulkReservationConfirm(ctx context.Context, input BulkReservationConfirmInput) (BulkReservationResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return BulkReservationResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var result BulkReservationResult
	for _, row := range input.Rows {
		totalQty := 0
		for _, sa := range row.StockAllocations {
			totalQty += sa.Quantity
		}
		for _, oa := range row.OrderAllocations {
			totalQty += oa.Quantity
		}
		if totalQty <= 0 {
			continue
		}

		resID := uuid.New().String()
		priority := row.Priority
		if priority == "" {
			priority = "normal"
		}

		var neededBy interface{} = nil
		if row.NeededByAt != "" {
			if _, err := time.Parse("2006-01-02", row.NeededByAt); err != nil {
				return BulkReservationResult{}, fmt.Errorf("invalid neededByAt for item %s: expected YYYY-MM-DD", row.ItemID)
			}
			neededBy = row.NeededByAt
		}

		_, err := tx.ExecContext(ctx, `
			INSERT INTO reservations (id, item_id, device_scope_id, quantity, status, requested_by, purpose, priority, needed_by_at, note)
			VALUES ($1, $2, $3, $4, 'requested', $5, $6, $7, $8, '')
		`, resID, row.ItemID, input.ScopeID, totalQty, input.ActorID, row.Purpose, priority, neededBy)
		if err != nil {
			return BulkReservationResult{}, fmt.Errorf("insert reservation: %w", err)
		}

		// Create stock allocations
		for _, sa := range row.StockAllocations {
			if sa.Quantity <= 0 {
				continue
			}
			var available int
			if err := tx.QueryRowContext(ctx, `
				SELECT available_quantity
				FROM inventory_balances
				WHERE item_id = $1 AND location_code = $2
				FOR UPDATE
			`, row.ItemID, sa.LocationCode).Scan(&available); err != nil {
				return BulkReservationResult{}, fmt.Errorf("lock stock allocation: %w", err)
			}
			if available < sa.Quantity {
				return BulkReservationResult{}, fmt.Errorf("insufficient stock for item %s at %s: requested %d, available %d", row.ItemID, sa.LocationCode, sa.Quantity, available)
			}
			allocID := uuid.New().String()
			_, err := tx.ExecContext(ctx, `
				INSERT INTO reservation_allocations (id, reservation_id, item_id, location_code, quantity, status, source_type)
				VALUES ($1, $2, $3, $4, $5, 'allocated', 'stock')
			`, allocID, resID, row.ItemID, sa.LocationCode, sa.Quantity)
			if err != nil {
				return BulkReservationResult{}, fmt.Errorf("insert stock allocation: %w", err)
			}
			// Update inventory balance
			if err := adjustReservedQuantityTx(ctx, tx, row.ItemID, sa.LocationCode, sa.Quantity); err != nil {
				return BulkReservationResult{}, fmt.Errorf("adjust reserved qty: %w", err)
			}
			// Record inventory event
			eventID := uuid.New().String()
			_, err = tx.ExecContext(ctx, `
				INSERT INTO inventory_events (id, event_type, item_id, location_code, to_location_code, quantity_delta, source_type, source_id, occurred_at)
				VALUES ($1, 'reserve_allocate', $2, $3, $3, $4, 'reservation', $5, NOW())
			`, eventID, row.ItemID, sa.LocationCode, 0, resID)
			if err != nil {
				return BulkReservationResult{}, fmt.Errorf("insert inventory event: %w", err)
			}
		}

		// Create order allocations
		for _, oa := range row.OrderAllocations {
			if oa.Quantity <= 0 {
				continue
			}
			var pending int
			var expected sql.NullTime
			if err := tx.QueryRowContext(ctx, `
				SELECT pol.ordered_quantity - pol.received_quantity - COALESCE(existing_alloc.allocated, 0) AS pending,
				       pol.expected_arrival_date
				FROM purchase_order_lines pol
				JOIN procurement_lines pl ON pl.id = pol.procurement_line_id
				LEFT JOIN (
					SELECT ra.purchase_order_line_id, SUM(ra.quantity) AS allocated
					FROM reservation_allocations ra
					JOIN reservations rv ON rv.id = ra.reservation_id
					WHERE ra.status = 'allocated'
					  AND ra.source_type = 'incoming_order'
					  AND ra.purchase_order_line_id IS NOT NULL
					  AND rv.status NOT IN ('cancelled', 'released')
					GROUP BY ra.purchase_order_line_id
				) existing_alloc ON existing_alloc.purchase_order_line_id = pol.id
				WHERE pol.id = $1
				  AND pl.item_id = $2
				  AND pol.status NOT IN ('cancelled', 'received')
				FOR UPDATE OF pol
			`, oa.PurchaseOrderLineID, row.ItemID).Scan(&pending, &expected); err != nil {
				return BulkReservationResult{}, fmt.Errorf("lock incoming allocation: %w", err)
			}
			if !expected.Valid {
				return BulkReservationResult{}, fmt.Errorf("purchase order line %s has no expected arrival date", oa.PurchaseOrderLineID)
			}
			if row.NeededByAt != "" {
				neededDate, _ := time.Parse("2006-01-02", row.NeededByAt)
				if expected.Time.After(neededDate) {
					return BulkReservationResult{}, fmt.Errorf("purchase order line %s arrives after neededByAt for item %s", oa.PurchaseOrderLineID, row.ItemID)
				}
			}
			if pending < oa.Quantity {
				return BulkReservationResult{}, fmt.Errorf("insufficient incoming order quantity for line %s: requested %d, available %d", oa.PurchaseOrderLineID, oa.Quantity, pending)
			}
			allocID := uuid.New().String()
			_, err := tx.ExecContext(ctx, `
				INSERT INTO reservation_allocations (id, reservation_id, item_id, location_code, quantity, status, source_type, purchase_order_line_id)
				VALUES ($1, $2, $3, 'INCOMING', $4, 'allocated', 'incoming_order', $5)
			`, allocID, resID, row.ItemID, oa.Quantity, oa.PurchaseOrderLineID)
			if err != nil {
				return BulkReservationResult{}, fmt.Errorf("insert order allocation: %w", err)
			}
		}

		// Record reservation event
		reEventID := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO reservation_events (id, reservation_id, event_type, quantity, actor_id, occurred_at)
			VALUES ($1, $2, 'created', $3, $4, NOW())
		`, reEventID, resID, totalQty, input.ActorID)
		if err != nil {
			return BulkReservationResult{}, fmt.Errorf("insert reservation event: %w", err)
		}

		result.Created++
		result.IDs = append(result.IDs, resID)
	}

	if err := tx.Commit(); err != nil {
		return BulkReservationResult{}, fmt.Errorf("commit: %w", err)
	}
	if result.IDs == nil {
		result.IDs = []string{}
	}
	return result, nil
}

// ArrivalCalendar returns PO line arrivals grouped by date for a month.
func (r *Repository) ArrivalCalendar(ctx context.Context, yearMonth string) (ArrivalCalendar, error) {
	const q = `
		SELECT
			pol.expected_arrival_date,
			i.id AS item_id,
			i.canonical_item_number,
			COALESCE(m.name, '') AS manufacturer,
			COALESCE(i.description, '') AS description,
			pol.ordered_quantity - pol.received_quantity AS pending_qty,
			COALESCE(po.order_number, '') AS po_number,
			pol.id AS pol_id,
			COALESCE(sq.quotation_number, '') AS quot_number,
			COALESCE(s.name, '') AS supplier_name
		FROM purchase_order_lines pol
		JOIN purchase_orders po ON po.id = pol.purchase_order_id
		JOIN procurement_lines pl ON pl.id = pol.procurement_line_id
		JOIN procurement_batches pb ON pb.id = po.procurement_batch_id
		JOIN items i ON i.id = pl.item_id
		LEFT JOIN manufacturers m ON m.key = i.manufacturer_key
		LEFT JOIN suppliers s ON s.id = pb.supplier_id
		LEFT JOIN quotation_lines ql ON ql.id = pl.quotation_line_id
		LEFT JOIN supplier_quotations sq ON sq.id = ql.quotation_id
		WHERE pol.expected_arrival_date IS NOT NULL
		  AND pol.status NOT IN ('cancelled', 'received')
		  AND TO_CHAR(pol.expected_arrival_date, 'YYYY-MM') = $1
		  AND (pol.ordered_quantity - pol.received_quantity) > 0
		ORDER BY pol.expected_arrival_date, i.canonical_item_number
	`
	rows, err := r.db.QueryContext(ctx, q, yearMonth)
	if err != nil {
		return ArrivalCalendar{}, fmt.Errorf("query arrival calendar: %w", err)
	}
	defer rows.Close()

	dayMap := map[string]*ArrivalCalendarDay{}
	dayOrder := []string{}
	for rows.Next() {
		var arrDate time.Time
		var item ArrivalCalendarItem
		if err := rows.Scan(&arrDate, &item.ItemID, &item.ItemNumber, &item.Manufacturer,
			&item.Description, &item.Quantity, &item.PurchaseOrderNumber,
			&item.PurchaseOrderLineID, &item.QuotationNumber, &item.SupplierName); err != nil {
			return ArrivalCalendar{}, fmt.Errorf("scan arrival calendar: %w", err)
		}
		dateStr := arrDate.Format("2006-01-02")
		if _, ok := dayMap[dateStr]; !ok {
			dayMap[dateStr] = &ArrivalCalendarDay{Date: dateStr}
			dayOrder = append(dayOrder, dateStr)
		}
		dayMap[dateStr].Items = append(dayMap[dateStr].Items, item)
	}

	result := ArrivalCalendar{YearMonth: yearMonth}
	for _, d := range dayOrder {
		result.Days = append(result.Days, *dayMap[d])
	}
	if result.Days == nil {
		result.Days = []ArrivalCalendarDay{}
	}
	return result, rows.Err()
}

// ItemSuggest returns items matching a fuzzy search query.
func (r *Repository) ItemSuggest(ctx context.Context, query string) (ItemSuggestionList, error) {
	const q = `
		SELECT i.id, i.canonical_item_number, COALESCE(i.description, '') AS description,
		       COALESCE(m.name, '') AS manufacturer, COALESCE(c.name, '') AS category
		FROM items i
		LEFT JOIN manufacturers m ON m.key = i.manufacturer_key
		LEFT JOIN categories c ON c.key = i.category_key
		WHERE i.canonical_item_number ILIKE '%' || $1 || '%'
		   OR i.description ILIKE '%' || $1 || '%'
		   OR m.name ILIKE '%' || $1 || '%'
		ORDER BY
		    CASE WHEN i.canonical_item_number ILIKE $1 || '%' THEN 0 ELSE 1 END,
		    i.canonical_item_number
		LIMIT 20
	`
	rows, err := r.db.QueryContext(ctx, q, query)
	if err != nil {
		return ItemSuggestionList{}, fmt.Errorf("query item suggest: %w", err)
	}
	defer rows.Close()

	var result ItemSuggestionList
	for rows.Next() {
		var s ItemSuggestion
		if err := rows.Scan(&s.ID, &s.ItemNumber, &s.Description, &s.Manufacturer, &s.Category); err != nil {
			return ItemSuggestionList{}, fmt.Errorf("scan item suggest: %w", err)
		}
		result.Rows = append(result.Rows, s)
	}
	if result.Rows == nil {
		result.Rows = []ItemSuggestion{}
	}
	return result, rows.Err()
}

// CategorySuggest returns categories matching a search query.
func (r *Repository) CategorySuggest(ctx context.Context, query string) (CategorySuggestionList, error) {
	const q = `
		SELECT key, name FROM categories
		WHERE name ILIKE '%' || $1 || '%'
		ORDER BY
		    CASE WHEN name ILIKE $1 || '%' THEN 0 ELSE 1 END,
		    name
		LIMIT 20
	`
	rows, err := r.db.QueryContext(ctx, q, query)
	if err != nil {
		return CategorySuggestionList{}, fmt.Errorf("query category suggest: %w", err)
	}
	defer rows.Close()

	var result CategorySuggestionList
	for rows.Next() {
		var s CategorySuggestion
		if err := rows.Scan(&s.Key, &s.Name); err != nil {
			return CategorySuggestionList{}, fmt.Errorf("scan category suggest: %w", err)
		}
		result.Rows = append(result.Rows, s)
	}
	if result.Rows == nil {
		result.Rows = []CategorySuggestion{}
	}
	return result, rows.Err()
}

// InventorySnapshotAtDate returns projected inventory at a future date.
func (r *Repository) InventorySnapshotAtDate(ctx context.Context, device, scope, itemID, targetDate string) (InventorySnapshot, error) {
	// First get the standard snapshot
	snap, err := r.InventorySnapshot(ctx, device, scope, itemID)
	if err != nil {
		return InventorySnapshot{}, err
	}

	// Parse target date
	td, err := time.Parse("2006-01-02", targetDate)
	if err != nil {
		return InventorySnapshot{}, fmt.Errorf("invalid target_date format (expected YYYY-MM-DD): %w", err)
	}

	// Get incoming quantities per item by target date
	const q = `
		SELECT pl.item_id, SUM(pol.ordered_quantity - pol.received_quantity) AS incoming
		FROM purchase_order_lines pol
		JOIN procurement_lines pl ON pl.id = pol.procurement_line_id
		WHERE pol.status NOT IN ('cancelled', 'received')
		  AND pol.expected_arrival_date IS NOT NULL
		  AND pol.expected_arrival_date <= $1
		  AND (pol.ordered_quantity - pol.received_quantity) > 0
		GROUP BY pl.item_id
	`
	rows, err := r.db.QueryContext(ctx, q, td)
	if err != nil {
		return InventorySnapshot{}, fmt.Errorf("query projected incoming: %w", err)
	}
	defer rows.Close()

	projected := map[string]int{}
	for rows.Next() {
		var itemID string
		var incoming int
		if err := rows.Scan(&itemID, &incoming); err != nil {
			return InventorySnapshot{}, fmt.Errorf("scan projected incoming: %w", err)
		}
		projected[itemID] = incoming
	}

	// Update snapshot rows with projected values
	for i := range snap.Rows {
		row := &snap.Rows[i]
		if inc, ok := projected[row.ItemID]; ok {
			row.IncomingQuantity = inc
		}
		row.NetAvailableQuantity = row.FreeQuantity + row.IncomingQuantity - row.UncoveredDemandQuantity
	}

	// Regenerate signature
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v-%s", snap.Rows, targetDate)))
	snap.SnapshotSignature = fmt.Sprintf("%x", h.Sum(nil))
	snap.GeneratedAt = time.Now().UTC().Format(time.RFC3339)

	return snap, nil
}

func splitCSVRefs(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	refs := make([]string, 0, len(parts))
	for _, part := range parts {
		ref := strings.TrimSpace(part)
		if ref != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}
