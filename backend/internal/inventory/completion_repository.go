package inventory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (r *Repository) Requirements(ctx context.Context, device, scope string) (RequirementList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			sir.id,
			ds.device_key,
			ds.scope_key,
			i.id,
			i.canonical_item_number,
			i.description,
			sir.quantity,
			sir.note
		FROM scope_item_requirements sir
		JOIN device_scopes ds ON ds.id = sir.scope_id
		JOIN items i ON i.id = sir.item_id
		WHERE ($1 = '' OR ds.device_key = $1)
		  AND ($2 = '' OR ds.scope_key = $2)
		ORDER BY ds.device_key, ds.scope_key, i.canonical_item_number
	`, device, scope)
	if err != nil {
		return RequirementList{}, fmt.Errorf("query requirements: %w", err)
	}
	defer rows.Close()

	result := RequirementList{Rows: []RequirementSummary{}}
	for rows.Next() {
		var row RequirementSummary
		if err := rows.Scan(
			&row.ID,
			&row.Device,
			&row.Scope,
			&row.ItemID,
			&row.ItemNumber,
			&row.Description,
			&row.Quantity,
			&row.Note,
		); err != nil {
			return RequirementList{}, fmt.Errorf("scan requirement: %w", err)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) UpsertRequirement(ctx context.Context, input RequirementUpsertInput) (RequirementSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return RequirementSummary{}, fmt.Errorf("begin requirement tx: %w", err)
	}
	defer tx.Rollback()

	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = fmt.Sprintf("req-%d", time.Now().UnixNano())
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO scope_item_requirements (id, scope_id, item_id, quantity, note, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		`, id, input.DeviceScopeID, input.ItemID, input.Quantity, input.Note); err != nil {
			return RequirementSummary{}, fmt.Errorf("insert requirement: %w", err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
			UPDATE scope_item_requirements
			SET scope_id = $2,
			    item_id = $3,
			    quantity = $4,
			    note = $5,
			    updated_at = NOW()
			WHERE id = $1
		`, id, input.DeviceScopeID, input.ItemID, input.Quantity, input.Note); err != nil {
			return RequirementSummary{}, fmt.Errorf("update requirement: %w", err)
		}
	}

	row := tx.QueryRowContext(ctx, `
		SELECT
			sir.id,
			ds.device_key,
			ds.scope_key,
			i.id,
			i.canonical_item_number,
			i.description,
			sir.quantity,
			sir.note
		FROM scope_item_requirements sir
		JOIN device_scopes ds ON ds.id = sir.scope_id
		JOIN items i ON i.id = sir.item_id
		WHERE sir.id = $1
	`, id)
	var result RequirementSummary
	if err := row.Scan(
		&result.ID,
		&result.Device,
		&result.Scope,
		&result.ItemID,
		&result.ItemNumber,
		&result.Description,
		&result.Quantity,
		&result.Note,
	); err != nil {
		return RequirementSummary{}, fmt.Errorf("reload requirement: %w", err)
	}

	if err := recordAuditEventTx(ctx, tx, "", "requirement.upserted", "scope_item_requirement", id, map[string]any{
		"scopeId":  input.DeviceScopeID,
		"itemId":   input.ItemID,
		"quantity": input.Quantity,
	}); err != nil {
		return RequirementSummary{}, err
	}

	if err := tx.Commit(); err != nil {
		return RequirementSummary{}, fmt.Errorf("commit requirement tx: %w", err)
	}
	return result, nil
}

func (r *Repository) ReservationDetail(ctx context.Context, id string) (ReservationDetail, error) {
	return r.reservationDetail(ctx, r.db, id)
}

func (r *Repository) UpdateReservation(ctx context.Context, id string, input ReservationUpdateInput) (ReservationDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ReservationDetail{}, fmt.Errorf("begin reservation update tx: %w", err)
	}
	defer tx.Rollback()

	current, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return ReservationDetail{}, err
	}
	if current.Status == "fulfilled" || current.Status == "cancelled" {
		return ReservationDetail{}, fmt.Errorf("reservation cannot be updated from status %s", current.Status)
	}
	if current.AllocatedQuantity > input.Quantity {
		return ReservationDetail{}, fmt.Errorf("updated quantity cannot be lower than allocated quantity")
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE reservations
		SET item_id = $2,
		    device_scope_id = $3,
		    quantity = $4,
		    requested_by = $5,
		    purpose = $6,
		    priority = $7,
		    needed_by_at = NULLIF($8, '')::timestamptz,
		    planned_use_at = NULLIF($9, '')::timestamptz,
		    hold_until_at = NULLIF($10, '')::timestamptz,
		    note = $11,
		    updated_at = NOW()
		WHERE id = $1
	`, id, input.ItemID, input.DeviceScopeID, input.Quantity, emptyDefault(input.RequestedBy, current.RequestedBy), input.Purpose, defaultString(input.Priority, "normal"), input.NeededByAt, input.PlannedUseAt, input.HoldUntilAt, input.Note); err != nil {
		return ReservationDetail{}, fmt.Errorf("update reservation: %w", err)
	}
	if err := r.refreshReservationStatusTx(ctx, tx, id); err != nil {
		return ReservationDetail{}, err
	}
	if err := r.recordReservationEventTx(ctx, tx, id, "updated", input.Quantity, input.RequestedBy, map[string]any{
		"itemId":        input.ItemID,
		"deviceScopeId": input.DeviceScopeID,
		"purpose":       input.Purpose,
		"priority":      defaultString(input.Priority, "normal"),
	}); err != nil {
		return ReservationDetail{}, err
	}
	if err := recordAuditEventTx(ctx, tx, input.RequestedBy, "reservation.updated", "reservation", id, map[string]any{
		"itemId":        input.ItemID,
		"deviceScopeId": input.DeviceScopeID,
		"quantity":      input.Quantity,
	}); err != nil {
		return ReservationDetail{}, err
	}

	detail, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return ReservationDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReservationDetail{}, fmt.Errorf("commit reservation update tx: %w", err)
	}
	return detail, nil
}

func (r *Repository) DeleteReservation(ctx context.Context, id, actorID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reservation delete tx: %w", err)
	}
	defer tx.Rollback()

	detail, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return err
	}
	if detail.Status == "fulfilled" {
		return fmt.Errorf("fulfilled reservations cannot be deleted")
	}
	if detail.AllocatedQuantity > 0 {
		return fmt.Errorf("release allocated quantity before deleting reservation")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM reservations WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete reservation: %w", err)
	}
	if err := recordAuditEventTx(ctx, tx, actorID, "reservation.deleted", "reservation", id, map[string]any{
		"itemId":   detail.ItemID,
		"quantity": detail.Quantity,
		"status":   detail.Status,
	}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reservation delete tx: %w", err)
	}
	return nil
}

func (r *Repository) AllocateReservation(ctx context.Context, id string, input ReservationActionInput) (ReservationDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ReservationDetail{}, fmt.Errorf("begin allocation tx: %w", err)
	}
	defer tx.Rollback()

	reservation, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return ReservationDetail{}, err
	}
	if reservation.Status == "cancelled" || reservation.Status == "fulfilled" {
		return ReservationDetail{}, fmt.Errorf("reservation cannot be allocated from status %s", reservation.Status)
	}

	if err := ensureBalanceRowTx(ctx, tx, reservation.ItemID, input.LocationCode); err != nil {
		return ReservationDetail{}, err
	}

	var available int
	if err := tx.QueryRowContext(ctx, `
		SELECT available_quantity
		FROM inventory_balances
		WHERE item_id = $1 AND location_code = $2
		FOR UPDATE
	`, reservation.ItemID, input.LocationCode).Scan(&available); err != nil {
		return ReservationDetail{}, fmt.Errorf("load inventory balance: %w", err)
	}
	if available < input.Quantity {
		return ReservationDetail{}, fmt.Errorf("insufficient available quantity at %s", input.LocationCode)
	}

	allocationID := fmt.Sprintf("alloc-%d", time.Now().UnixNano())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO reservation_allocations (id, reservation_id, item_id, location_code, quantity, status, allocated_at, note)
		VALUES ($1, $2, $3, $4, $5, 'allocated', NOW(), $6)
	`, allocationID, id, reservation.ItemID, input.LocationCode, input.Quantity, input.Note); err != nil {
		return ReservationDetail{}, fmt.Errorf("insert reservation allocation: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO inventory_events (
			id, item_id, location_code, from_location_code, to_location_code, event_type, quantity_delta, note, device_scope_id,
			actor_id, source_type, source_id, correlation_id, occurred_at
		) VALUES ($1, $2, $3, $4, '', 'reserve_allocate', 0, $5, $6, $7, 'reservation', $8, $9, NOW())
	`, fmt.Sprintf("evt-%d", time.Now().UnixNano()), reservation.ItemID, input.LocationCode, input.LocationCode, input.Note, reservation.DeviceScopeID, emptyDefault(input.ActorID, "local-user"), id, allocationID); err != nil {
		return ReservationDetail{}, fmt.Errorf("insert reservation allocation event: %w", err)
	}

	if err := adjustReservedQuantityTx(ctx, tx, reservation.ItemID, input.LocationCode, input.Quantity); err != nil {
		return ReservationDetail{}, err
	}
	if err := r.recordReservationEventTx(ctx, tx, id, "allocated", input.Quantity, input.ActorID, map[string]any{
		"locationCode": input.LocationCode,
		"allocationId": allocationID,
		"note":         input.Note,
	}); err != nil {
		return ReservationDetail{}, err
	}
	if err := r.refreshReservationStatusTx(ctx, tx, id); err != nil {
		return ReservationDetail{}, err
	}
	if err := recordAuditEventTx(ctx, tx, input.ActorID, "reservation.allocated", "reservation", id, map[string]any{
		"locationCode": input.LocationCode,
		"quantity":     input.Quantity,
	}); err != nil {
		return ReservationDetail{}, err
	}

	detail, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return ReservationDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReservationDetail{}, fmt.Errorf("commit allocation tx: %w", err)
	}
	return detail, nil
}

func (r *Repository) ReleaseReservation(ctx context.Context, id string, input ReservationActionInput) (ReservationDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ReservationDetail{}, fmt.Errorf("begin release tx: %w", err)
	}
	defer tx.Rollback()

	detail, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return ReservationDetail{}, err
	}
	remaining := input.Quantity
	if remaining <= 0 {
		remaining = detail.AllocatedQuantity
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, location_code, quantity
		FROM reservation_allocations
		WHERE reservation_id = $1 AND status = 'allocated'
		ORDER BY allocated_at
		FOR UPDATE
	`, id)
	if err != nil {
		return ReservationDetail{}, fmt.Errorf("query reservation allocations: %w", err)
	}
	defer rows.Close()

	type allocationRow struct {
		ID           string
		LocationCode string
		Quantity     int
	}
	allocations := []allocationRow{}
	for rows.Next() {
		var row allocationRow
		if err := rows.Scan(&row.ID, &row.LocationCode, &row.Quantity); err != nil {
			return ReservationDetail{}, fmt.Errorf("scan reservation allocation: %w", err)
		}
		allocations = append(allocations, row)
	}

	released := 0
	for _, allocation := range allocations {
		if remaining <= 0 {
			break
		}
		portion := allocation.Quantity
		if portion > remaining {
			portion = remaining
		}

		if portion == allocation.Quantity {
			if _, err := tx.ExecContext(ctx, `
				UPDATE reservation_allocations
				SET status = 'released', released_at = NOW(), note = $2
				WHERE id = $1
			`, allocation.ID, input.Note); err != nil {
				return ReservationDetail{}, fmt.Errorf("release allocation: %w", err)
			}
		} else {
			if _, err := tx.ExecContext(ctx, `
				UPDATE reservation_allocations
				SET quantity = quantity - $2
				WHERE id = $1
			`, allocation.ID, portion); err != nil {
				return ReservationDetail{}, fmt.Errorf("shrink allocation: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO reservation_allocations (id, reservation_id, item_id, location_code, quantity, status, allocated_at, released_at, note)
				SELECT $1, reservation_id, item_id, location_code, $3, 'released', allocated_at, NOW(), $4
				FROM reservation_allocations
				WHERE id = $2
			`, fmt.Sprintf("alloc-%d", time.Now().UnixNano()), allocation.ID, portion, input.Note); err != nil {
				return ReservationDetail{}, fmt.Errorf("split release allocation: %w", err)
			}
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO inventory_events (
				id, item_id, location_code, from_location_code, to_location_code, event_type, quantity_delta, note, device_scope_id,
				actor_id, source_type, source_id, correlation_id, occurred_at
			) VALUES ($1, $2, $3, $4, '', 'reserve_release', 0, $5, $6, $7, 'reservation', $8, $9, NOW())
		`, fmt.Sprintf("evt-%d", time.Now().UnixNano()), detail.ItemID, allocation.LocationCode, allocation.LocationCode, input.Note, detail.DeviceScopeID, emptyDefault(input.ActorID, "local-user"), id, allocation.ID); err != nil {
			return ReservationDetail{}, fmt.Errorf("insert reservation release event: %w", err)
		}

		if err := adjustReservedQuantityTx(ctx, tx, detail.ItemID, allocation.LocationCode, -portion); err != nil {
			return ReservationDetail{}, err
		}
		released += portion
		remaining -= portion
	}

	if released == 0 {
		return ReservationDetail{}, fmt.Errorf("no allocated quantity available to release")
	}
	if err := r.recordReservationEventTx(ctx, tx, id, "released", released, input.ActorID, map[string]any{
		"note": input.Note,
	}); err != nil {
		return ReservationDetail{}, err
	}
	if err := r.refreshReservationStatusTx(ctx, tx, id); err != nil {
		return ReservationDetail{}, err
	}
	if err := recordAuditEventTx(ctx, tx, input.ActorID, "reservation.released", "reservation", id, map[string]any{
		"quantity": released,
	}); err != nil {
		return ReservationDetail{}, err
	}

	result, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return ReservationDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReservationDetail{}, fmt.Errorf("commit release tx: %w", err)
	}
	return result, nil
}

func (r *Repository) FulfillReservation(ctx context.Context, id string, input ReservationActionInput) (ReservationDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ReservationDetail{}, fmt.Errorf("begin fulfill tx: %w", err)
	}
	defer tx.Rollback()

	detail, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return ReservationDetail{}, err
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, location_code, quantity
		FROM reservation_allocations
		WHERE reservation_id = $1 AND status = 'allocated'
		ORDER BY allocated_at
		FOR UPDATE
	`, id)
	if err != nil {
		return ReservationDetail{}, fmt.Errorf("query allocations for fulfillment: %w", err)
	}
	defer rows.Close()

	fulfilled := 0
	for rows.Next() {
		var allocationID, locationCode string
		var quantity int
		if err := rows.Scan(&allocationID, &locationCode, &quantity); err != nil {
			return ReservationDetail{}, fmt.Errorf("scan fulfillment allocation: %w", err)
		}

		if err := decrementBalanceTx(ctx, tx, detail.ItemID, locationCode, quantity); err != nil {
			return ReservationDetail{}, err
		}
		if err := adjustReservedQuantityTx(ctx, tx, detail.ItemID, locationCode, -quantity); err != nil {
			return ReservationDetail{}, err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO inventory_events (
				id, item_id, location_code, from_location_code, to_location_code, event_type, quantity_delta, note, device_scope_id,
				actor_id, source_type, source_id, correlation_id, occurred_at
			) VALUES ($1, $2, $3, $4, '', 'consume', $5, $6, $7, $8, 'reservation', $9, $10, NOW())
		`, fmt.Sprintf("evt-%d", time.Now().UnixNano()), detail.ItemID, locationCode, locationCode, -quantity, input.Note, detail.DeviceScopeID, emptyDefault(input.ActorID, "local-user"), id, allocationID); err != nil {
			return ReservationDetail{}, fmt.Errorf("insert fulfillment inventory event: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE reservation_allocations
			SET status = 'released', released_at = NOW(), note = $2
			WHERE id = $1
		`, allocationID, input.Note); err != nil {
			return ReservationDetail{}, fmt.Errorf("mark allocation fulfilled: %w", err)
		}
		fulfilled += quantity
	}

	if fulfilled == 0 {
		return ReservationDetail{}, fmt.Errorf("reservation has no allocated quantity to fulfill")
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE reservations
		SET status = 'fulfilled',
		    fulfilled_at = NOW(),
		    released_at = COALESCE(released_at, NOW()),
		    updated_at = NOW()
		WHERE id = $1
	`, id); err != nil {
		return ReservationDetail{}, fmt.Errorf("mark reservation fulfilled: %w", err)
	}
	if err := r.recordReservationEventTx(ctx, tx, id, "fulfilled", fulfilled, input.ActorID, map[string]any{
		"note": input.Note,
	}); err != nil {
		return ReservationDetail{}, err
	}
	if err := recordAuditEventTx(ctx, tx, input.ActorID, "reservation.fulfilled", "reservation", id, map[string]any{
		"quantity": fulfilled,
	}); err != nil {
		return ReservationDetail{}, err
	}

	result, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return ReservationDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReservationDetail{}, fmt.Errorf("commit fulfillment tx: %w", err)
	}
	return result, nil
}

func (r *Repository) CancelReservation(ctx context.Context, id string, input ReservationActionInput) (ReservationDetail, error) {
	if _, err := r.ReleaseReservation(ctx, id, ReservationActionInput{Quantity: input.Quantity, ActorID: input.ActorID, Note: input.Note}); err != nil && !strings.Contains(err.Error(), "no allocated quantity") {
		return ReservationDetail{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ReservationDetail{}, fmt.Errorf("begin cancellation tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE reservations
		SET status = 'cancelled',
		    cancellation_reason = $2,
		    released_at = COALESCE(released_at, NOW()),
		    updated_at = NOW()
		WHERE id = $1
	`, id, defaultString(input.Reason, input.Note)); err != nil {
		return ReservationDetail{}, fmt.Errorf("cancel reservation: %w", err)
	}
	if err := r.recordReservationEventTx(ctx, tx, id, "cancelled", 0, input.ActorID, map[string]any{
		"reason": input.Reason,
		"note":   input.Note,
	}); err != nil {
		return ReservationDetail{}, err
	}
	if err := recordAuditEventTx(ctx, tx, input.ActorID, "reservation.cancelled", "reservation", id, map[string]any{
		"reason": input.Reason,
	}); err != nil {
		return ReservationDetail{}, err
	}
	result, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return ReservationDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReservationDetail{}, fmt.Errorf("commit cancellation tx: %w", err)
	}
	return result, nil
}

func (r *Repository) UndoReservation(ctx context.Context, id string, input ReservationActionInput) (ReservationDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ReservationDetail{}, fmt.Errorf("begin reservation undo tx: %w", err)
	}
	defer tx.Rollback()

	var eventType string
	var metadataRaw []byte
	if err := tx.QueryRowContext(ctx, `
		SELECT event_type, metadata
		FROM reservation_events
		WHERE reservation_id = $1
		ORDER BY occurred_at DESC
		LIMIT 1
	`, id).Scan(&eventType, &metadataRaw); err != nil {
		return ReservationDetail{}, fmt.Errorf("load reservation event for undo: %w", err)
	}

	switch eventType {
	case "allocated":
		var payload map[string]any
		_ = json.Unmarshal(metadataRaw, &payload)
		qty, _ := payload["quantity"].(float64)
		locationCode, _ := payload["locationCode"].(string)
		if locationCode == "" {
			return ReservationDetail{}, fmt.Errorf("latest allocation event cannot be undone without locationCode")
		}
		return r.ReleaseReservation(ctx, id, ReservationActionInput{
			LocationCode: locationCode,
			Quantity:     int(qty),
			ActorID:      input.ActorID,
			Note:         defaultString(input.Note, defaultString(input.Reason, "Undo allocation")),
		})
	case "cancelled":
		if _, err := tx.ExecContext(ctx, `
			UPDATE reservations
			SET status = 'requested',
			    cancellation_reason = '',
			    updated_at = NOW()
			WHERE id = $1
		`, id); err != nil {
			return ReservationDetail{}, fmt.Errorf("undo cancellation: %w", err)
		}
	case "fulfilled":
		return ReservationDetail{}, fmt.Errorf("undo reservation fulfillment should be executed through inventory event undo")
	default:
		return ReservationDetail{}, fmt.Errorf("unsupported reservation undo event type: %s", eventType)
	}

	if err := r.recordReservationEventTx(ctx, tx, id, "undo", 0, input.ActorID, map[string]any{
		"reason": input.Reason,
	}); err != nil {
		return ReservationDetail{}, err
	}
	result, err := r.reservationDetailTx(ctx, tx, id)
	if err != nil {
		return ReservationDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReservationDetail{}, fmt.Errorf("commit reservation undo tx: %w", err)
	}
	return result, nil
}

func (r *Repository) InventoryItems(ctx context.Context) (InventoryItemList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			i.id,
			i.canonical_item_number,
			i.description,
			m.name,
			c.name,
			COALESCE(SUM(ib.on_hand_quantity), 0),
			COALESCE(SUM(ib.reserved_quantity), 0),
			COALESCE(SUM(ib.available_quantity), 0)
		FROM items i
		JOIN manufacturers m ON m.key = i.manufacturer_key
		JOIN categories c ON c.key = i.category_key
		LEFT JOIN inventory_balances ib ON ib.item_id = i.id
		GROUP BY i.id, i.canonical_item_number, i.description, m.name, c.name
		ORDER BY i.canonical_item_number
	`)
	if err != nil {
		return InventoryItemList{}, fmt.Errorf("query inventory items: %w", err)
	}
	defer rows.Close()

	result := InventoryItemList{Rows: []InventoryItemSummary{}}
	for rows.Next() {
		var row InventoryItemSummary
		if err := rows.Scan(
			&row.ItemID,
			&row.ItemNumber,
			&row.Description,
			&row.Manufacturer,
			&row.Category,
			&row.OnHandQuantity,
			&row.ReservedQuantity,
			&row.AvailableQuantity,
		); err != nil {
			return InventoryItemList{}, fmt.Errorf("scan inventory item: %w", err)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) InventoryLocations(ctx context.Context) (LocationList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			l.code,
			l.name,
			l.location_type,
			l.is_active,
			COALESCE(SUM(ib.on_hand_quantity), 0),
			COALESCE(SUM(ib.reserved_quantity), 0),
			COALESCE(SUM(ib.available_quantity), 0)
		FROM locations l
		LEFT JOIN inventory_balances ib ON ib.location_code = l.code
		GROUP BY l.code, l.name, l.location_type, l.is_active
		ORDER BY l.code
	`)
	if err != nil {
		return LocationList{}, fmt.Errorf("query inventory locations: %w", err)
	}
	defer rows.Close()

	result := LocationList{Rows: []LocationSummary{}}
	for rows.Next() {
		var row LocationSummary
		if err := rows.Scan(
			&row.Code,
			&row.Name,
			&row.LocationType,
			&row.IsActive,
			&row.OnHandQuantity,
			&row.ReservedQuantity,
			&row.AvailableQuantity,
		); err != nil {
			return LocationList{}, fmt.Errorf("scan inventory location: %w", err)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) InventoryEvents(ctx context.Context) (InventoryEventList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			e.id,
			e.event_type,
			e.item_id,
			i.canonical_item_number,
			COALESCE(e.from_location_code, ''),
			COALESCE(e.to_location_code, ''),
			e.quantity_delta,
			COALESCE(e.actor_id, ''),
			COALESCE(e.source_type, ''),
			COALESCE(e.source_id, ''),
			COALESCE(e.correlation_id, ''),
			COALESCE(e.reversed_by_event_id, ''),
			e.note,
			e.occurred_at
		FROM inventory_events e
		JOIN items i ON i.id = e.item_id
		ORDER BY e.occurred_at DESC
		LIMIT 200
	`)
	if err != nil {
		return InventoryEventList{}, fmt.Errorf("query inventory events: %w", err)
	}
	defer rows.Close()

	result := InventoryEventList{Rows: []InventoryEventEntry{}}
	for rows.Next() {
		var row InventoryEventEntry
		var occurredAt time.Time
		if err := rows.Scan(
			&row.ID,
			&row.EventType,
			&row.ItemID,
			&row.ItemNumber,
			&row.FromLocationCode,
			&row.ToLocationCode,
			&row.QuantityDelta,
			&row.ActorID,
			&row.SourceType,
			&row.SourceID,
			&row.CorrelationID,
			&row.ReversedByEventID,
			&row.Note,
			&occurredAt,
		); err != nil {
			return InventoryEventList{}, fmt.Errorf("scan inventory event: %w", err)
		}
		row.OccurredAt = occurredAt.UTC().Format(time.RFC3339)
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) InventorySnapshot(ctx context.Context, device, scope, itemID string) (InventorySnapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, i.canonical_item_number, i.description, m.name, c.name
		FROM items i
		JOIN manufacturers m ON m.key = i.manufacturer_key
		JOIN categories c ON c.key = i.category_key
		WHERE ($1 = '' OR i.id = $1)
		ORDER BY i.canonical_item_number
	`, itemID)
	if err != nil {
		return InventorySnapshot{}, fmt.Errorf("query snapshot items: %w", err)
	}
	defer rows.Close()

	type snapshotAccumulator struct {
		row InventorySnapshotRow
	}
	accumulators := map[string]*snapshotAccumulator{}
	for rows.Next() {
		var manufacturerKey, categoryKey string
		var row InventorySnapshotRow
		if err := rows.Scan(&row.ItemID, &row.ItemNumber, &row.Description, &manufacturerKey, &categoryKey); err != nil {
			return InventorySnapshot{}, fmt.Errorf("scan snapshot item: %w", err)
		}
		row.Manufacturer = manufacturerKey
		row.Category = categoryKey
		row.ScopeSummaries = []InventorySnapshotScopeSummary{}
		accumulators[row.ItemID] = &snapshotAccumulator{row: row}
	}
	if err := rows.Err(); err != nil {
		return InventorySnapshot{}, err
	}

	if len(accumulators) == 0 {
		return InventorySnapshot{
			GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
			SnapshotSignature: snapshotSignature(nil),
			DeviceFilter:      device,
			ScopeFilter:       scope,
			ItemIDFilter:      itemID,
			Rows:              []InventorySnapshotRow{},
		}, nil
	}

	itemFilter := func(id string) bool {
		return itemID == "" || id == itemID
	}

	inventoryRows, err := r.db.QueryContext(ctx, `
		SELECT item_id, COALESCE(SUM(on_hand_quantity), 0), COALESCE(SUM(reserved_quantity), 0), COALESCE(SUM(available_quantity), 0)
		FROM inventory_balances
		WHERE ($1 = '' OR item_id = $1)
		GROUP BY item_id
	`, itemID)
	if err != nil {
		return InventorySnapshot{}, fmt.Errorf("query snapshot inventory balances: %w", err)
	}
	defer inventoryRows.Close()
	for inventoryRows.Next() {
		var key string
		var onHand, reserved, available int
		if err := inventoryRows.Scan(&key, &onHand, &reserved, &available); err != nil {
			return InventorySnapshot{}, fmt.Errorf("scan snapshot inventory balances: %w", err)
		}
		if current := accumulators[key]; current != nil {
			current.row.OnHandQuantity = onHand
			current.row.AllocatedReservedQty = reserved
			current.row.FreeQuantity = available
		}
	}
	if err := inventoryRows.Err(); err != nil {
		return InventorySnapshot{}, err
	}

	requirementRows, err := r.db.QueryContext(ctx, `
		SELECT sir.item_id, ds.device_key, ds.scope_key, SUM(sir.quantity)
		FROM scope_item_requirements sir
		JOIN device_scopes ds ON ds.id = sir.scope_id
		WHERE ($1 = '' OR sir.item_id = $1)
		  AND ($2 = '' OR ds.device_key = $2)
		  AND ($3 = '' OR ds.scope_key = $3)
		GROUP BY sir.item_id, ds.device_key, ds.scope_key
	`, itemID, device, scope)
	if err != nil {
		return InventorySnapshot{}, fmt.Errorf("query snapshot requirements: %w", err)
	}
	defer requirementRows.Close()
	scopeSummaries := map[string]map[string]*InventorySnapshotScopeSummary{}
	for requirementRows.Next() {
		var itemKey, deviceKey, scopeKey string
		var quantity int
		if err := requirementRows.Scan(&itemKey, &deviceKey, &scopeKey, &quantity); err != nil {
			return InventorySnapshot{}, fmt.Errorf("scan snapshot requirement: %w", err)
		}
		if !itemFilter(itemKey) {
			continue
		}
		current := accumulators[itemKey]
		if current == nil {
			continue
		}
		current.row.RequirementQuantity += quantity
		if scopeSummaries[itemKey] == nil {
			scopeSummaries[itemKey] = map[string]*InventorySnapshotScopeSummary{}
		}
		scopeMapKey := deviceKey + "|" + scopeKey
		scopeSummaries[itemKey][scopeMapKey] = &InventorySnapshotScopeSummary{
			Device:              deviceKey,
			Scope:               scopeKey,
			RequirementQuantity: quantity,
		}
	}
	if err := requirementRows.Err(); err != nil {
		return InventorySnapshot{}, err
	}

	reservationRows, err := r.db.QueryContext(ctx, `
		SELECT
			r.item_id,
			ds.device_key,
			ds.scope_key,
			COALESCE(SUM(r.quantity), 0) AS reservation_quantity,
			COALESCE(SUM(
				CASE WHEN r.status IN ('allocated', 'partially_allocated')
				THEN (SELECT COALESCE(SUM(ra.quantity), 0) FROM reservation_allocations ra WHERE ra.reservation_id = r.id AND ra.status = 'allocated')
				ELSE 0 END
			), 0) AS allocated_quantity
		FROM reservations r
		JOIN device_scopes ds ON ds.id = r.device_scope_id
		WHERE r.status IN ('requested', 'awaiting_stock', 'partially_allocated', 'allocated')
		  AND ($1 = '' OR r.item_id = $1)
		  AND ($2 = '' OR ds.device_key = $2)
		  AND ($3 = '' OR ds.scope_key = $3)
		GROUP BY r.item_id, ds.device_key, ds.scope_key
	`, itemID, device, scope)
	if err != nil {
		return InventorySnapshot{}, fmt.Errorf("query snapshot reservations: %w", err)
	}
	defer reservationRows.Close()
	for reservationRows.Next() {
		var itemKey, deviceKey, scopeKey string
		var reservationQty, allocatedQty int
		if err := reservationRows.Scan(&itemKey, &deviceKey, &scopeKey, &reservationQty, &allocatedQty); err != nil {
			return InventorySnapshot{}, fmt.Errorf("scan snapshot reservation: %w", err)
		}
		current := accumulators[itemKey]
		if current == nil {
			continue
		}
		current.row.ReservationQuantity += reservationQty
		if scopeSummaries[itemKey] == nil {
			scopeSummaries[itemKey] = map[string]*InventorySnapshotScopeSummary{}
		}
		scopeMapKey := deviceKey + "|" + scopeKey
		summary := scopeSummaries[itemKey][scopeMapKey]
		if summary == nil {
			summary = &InventorySnapshotScopeSummary{Device: deviceKey, Scope: scopeKey}
			scopeSummaries[itemKey][scopeMapKey] = summary
		}
		summary.ReservationQuantity += reservationQty
		summary.AllocatedQuantity += allocatedQty
	}
	if err := reservationRows.Err(); err != nil {
		return InventorySnapshot{}, err
	}

	incomingRows, err := r.db.QueryContext(ctx, `
		SELECT pl.item_id, COALESCE(SUM(GREATEST(pol.ordered_quantity - COALESCE(pol.received_quantity, 0), 0)), 0)
		FROM purchase_order_lines pol
		JOIN purchase_orders po ON po.id = pol.purchase_order_id
		JOIN procurement_lines pl ON pl.id = pol.procurement_line_id
		WHERE ($1 = '' OR pl.item_id = $1)
		  AND po.status <> 'cancelled'
		  AND COALESCE(pol.status, 'ordered') <> 'cancelled'
		GROUP BY pl.item_id
	`, itemID)
	if err != nil {
		return InventorySnapshot{}, fmt.Errorf("query snapshot incoming orders: %w", err)
	}
	defer incomingRows.Close()
	for incomingRows.Next() {
		var itemKey string
		var incoming int
		if err := incomingRows.Scan(&itemKey, &incoming); err != nil {
			return InventorySnapshot{}, fmt.Errorf("scan snapshot incoming order: %w", err)
		}
		if current := accumulators[itemKey]; current != nil {
			current.row.IncomingQuantity = incoming
		}
	}
	if err := incomingRows.Err(); err != nil {
		return InventorySnapshot{}, err
	}

	result := InventorySnapshot{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		DeviceFilter: device,
		ScopeFilter:  scope,
		ItemIDFilter: itemID,
		Rows:         make([]InventorySnapshotRow, 0, len(accumulators)),
	}
	for itemKey, current := range accumulators {
		for _, summary := range scopeSummaries[itemKey] {
			summary.RemainingDemand = maxInt(summary.RequirementQuantity-summary.ReservationQuantity, 0)
			current.row.ScopeSummaries = append(current.row.ScopeSummaries, *summary)
		}
		sort.Slice(current.row.ScopeSummaries, func(i, j int) bool {
			if current.row.ScopeSummaries[i].Device == current.row.ScopeSummaries[j].Device {
				return current.row.ScopeSummaries[i].Scope < current.row.ScopeSummaries[j].Scope
			}
			return current.row.ScopeSummaries[i].Device < current.row.ScopeSummaries[j].Device
		})
		current.row.UncoveredDemandQuantity = maxInt(current.row.RequirementQuantity-current.row.ReservationQuantity, 0)
		current.row.NetAvailableQuantity = current.row.FreeQuantity + current.row.IncomingQuantity - current.row.UncoveredDemandQuantity
		result.Rows = append(result.Rows, current.row)
	}
	sort.Slice(result.Rows, func(i, j int) bool {
		return result.Rows[i].ItemNumber < result.Rows[j].ItemNumber
	})
	result.SnapshotSignature = snapshotSignature(result.Rows)
	return result, nil
}

func (r *Repository) ReceiveInventory(ctx context.Context, input InventoryReceiveInput) (InventoryEventEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return InventoryEventEntry{}, fmt.Errorf("begin receive tx: %w", err)
	}
	defer tx.Rollback()

	entry, err := insertInventoryEventTx(ctx, tx, inventoryEventInsert{
		ItemID:         input.ItemID,
		LocationCode:   input.LocationCode,
		ToLocationCode: input.LocationCode,
		EventType:      "receive",
		QuantityDelta:  input.Quantity,
		DeviceScopeID:  input.DeviceScopeID,
		ActorID:        input.ActorID,
		SourceType:     defaultString(input.SourceType, "manual"),
		SourceID:       input.SourceID,
		Note:           input.Note,
	})
	if err != nil {
		return InventoryEventEntry{}, err
	}
	if err := incrementBalanceTx(ctx, tx, input.ItemID, input.LocationCode, input.Quantity); err != nil {
		return InventoryEventEntry{}, err
	}
	if err := recordAuditEventTx(ctx, tx, input.ActorID, "inventory.received", "item", input.ItemID, map[string]any{
		"locationCode": input.LocationCode,
		"quantity":     input.Quantity,
	}); err != nil {
		return InventoryEventEntry{}, err
	}
	if err := tx.Commit(); err != nil {
		return InventoryEventEntry{}, fmt.Errorf("commit receive tx: %w", err)
	}
	return entry, nil
}

func (r *Repository) MoveInventory(ctx context.Context, input InventoryMoveInput) (InventoryEventEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return InventoryEventEntry{}, fmt.Errorf("begin move tx: %w", err)
	}
	defer tx.Rollback()

	if err := decrementBalanceTx(ctx, tx, input.ItemID, input.FromLocationCode, input.Quantity); err != nil {
		return InventoryEventEntry{}, err
	}
	if err := incrementBalanceTx(ctx, tx, input.ItemID, input.ToLocationCode, input.Quantity); err != nil {
		return InventoryEventEntry{}, err
	}
	entry, err := insertInventoryEventTx(ctx, tx, inventoryEventInsert{
		ItemID:            input.ItemID,
		LocationCode:      input.ToLocationCode,
		FromLocationCode:  input.FromLocationCode,
		ToLocationCode:    input.ToLocationCode,
		EventType:         "move",
		QuantityDelta:     input.Quantity,
		DeviceScopeID:     input.DeviceScopeID,
		ActorID:           input.ActorID,
		SourceType:        defaultString(input.SourceType, "manual"),
		SourceID:          input.SourceID,
		CorrelationSource: fmt.Sprintf("%s->%s", input.FromLocationCode, input.ToLocationCode),
		Note:              input.Note,
	})
	if err != nil {
		return InventoryEventEntry{}, err
	}
	if err := recordAuditEventTx(ctx, tx, input.ActorID, "inventory.moved", "item", input.ItemID, map[string]any{
		"from":     input.FromLocationCode,
		"to":       input.ToLocationCode,
		"quantity": input.Quantity,
	}); err != nil {
		return InventoryEventEntry{}, err
	}
	if err := tx.Commit(); err != nil {
		return InventoryEventEntry{}, fmt.Errorf("commit move tx: %w", err)
	}
	return entry, nil
}

func (r *Repository) UndoInventoryEvent(ctx context.Context, id string, input InventoryUndoInput) (InventoryEventEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return InventoryEventEntry{}, fmt.Errorf("begin inventory undo tx: %w", err)
	}
	defer tx.Rollback()

	original, err := inventoryEventByIDTx(ctx, tx, id)
	if err != nil {
		return InventoryEventEntry{}, err
	}
	if original.ReversedByEventID != "" {
		return InventoryEventEntry{}, fmt.Errorf("inventory event already reversed")
	}

	var undo InventoryEventEntry
	switch original.EventType {
	case "receive":
		if err := decrementBalanceTx(ctx, tx, original.ItemID, original.ToLocationCode, original.QuantityDelta); err != nil {
			return InventoryEventEntry{}, err
		}
		undo, err = insertInventoryEventTx(ctx, tx, inventoryEventInsert{
			ItemID:           original.ItemID,
			LocationCode:     original.ToLocationCode,
			FromLocationCode: original.ToLocationCode,
			EventType:        "undo",
			QuantityDelta:    -original.QuantityDelta,
			ActorID:          input.ActorID,
			SourceType:       "undo",
			SourceID:         id,
			Note:             defaultString(input.Reason, "Undo receive"),
		})
	case "adjust", "consume":
		if original.QuantityDelta > 0 {
			if err := decrementBalanceTx(ctx, tx, original.ItemID, original.ToLocationCode, original.QuantityDelta); err != nil {
				return InventoryEventEntry{}, err
			}
		} else {
			if err := incrementBalanceTx(ctx, tx, original.ItemID, defaultString(original.ToLocationCode, original.FromLocationCode), -original.QuantityDelta); err != nil {
				return InventoryEventEntry{}, err
			}
		}
		undo, err = insertInventoryEventTx(ctx, tx, inventoryEventInsert{
			ItemID:           original.ItemID,
			LocationCode:     defaultString(original.ToLocationCode, original.FromLocationCode),
			FromLocationCode: original.FromLocationCode,
			ToLocationCode:   original.ToLocationCode,
			EventType:        "undo",
			QuantityDelta:    -original.QuantityDelta,
			ActorID:          input.ActorID,
			SourceType:       "undo",
			SourceID:         id,
			Note:             defaultString(input.Reason, "Undo inventory adjustment"),
		})
	case "move":
		if err := decrementBalanceTx(ctx, tx, original.ItemID, original.ToLocationCode, original.QuantityDeltaOrTransfer()); err != nil {
			return InventoryEventEntry{}, err
		}
		if err := incrementBalanceTx(ctx, tx, original.ItemID, original.FromLocationCode, original.QuantityDeltaOrTransfer()); err != nil {
			return InventoryEventEntry{}, err
		}
		undo, err = insertInventoryEventTx(ctx, tx, inventoryEventInsert{
			ItemID:            original.ItemID,
			LocationCode:      original.FromLocationCode,
			FromLocationCode:  original.ToLocationCode,
			ToLocationCode:    original.FromLocationCode,
			EventType:         "undo",
			QuantityDelta:     0,
			ActorID:           input.ActorID,
			SourceType:        "undo",
			SourceID:          id,
			CorrelationSource: fmt.Sprintf("%s->%s", original.ToLocationCode, original.FromLocationCode),
			Note:              defaultString(input.Reason, "Undo inventory move"),
		})
	default:
		return InventoryEventEntry{}, fmt.Errorf("unsupported inventory undo for event type %s", original.EventType)
	}
	if err != nil {
		return InventoryEventEntry{}, err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE inventory_events SET reversed_by_event_id = $2 WHERE id = $1`, id, undo.ID); err != nil {
		return InventoryEventEntry{}, fmt.Errorf("mark inventory event reversed: %w", err)
	}
	if err := recordAuditEventTx(ctx, tx, input.ActorID, "inventory.undone", "inventory_event", id, map[string]any{
		"undoEventId": undo.ID,
	}); err != nil {
		return InventoryEventEntry{}, err
	}

	if err := tx.Commit(); err != nil {
		return InventoryEventEntry{}, fmt.Errorf("commit inventory undo tx: %w", err)
	}
	return undo, nil
}

func (r *Repository) Arrivals(ctx context.Context) (ArrivalList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			pol.id,
			po.id,
			po.order_number,
			pl.item_id,
			i.canonical_item_number,
			i.description,
			pol.ordered_quantity,
			pol.received_quantity,
			(pol.ordered_quantity - pol.received_quantity) AS pending_quantity,
			s.name,
			COALESCE(pol.expected_arrival_date::text, ''),
			pol.status
		FROM purchase_order_lines pol
		JOIN purchase_orders po ON po.id = pol.purchase_order_id
		JOIN procurement_lines pl ON pl.id = pol.procurement_line_id
		JOIN procurement_batches pb ON pb.id = po.procurement_batch_id
		JOIN suppliers s ON s.id = pb.supplier_id
		JOIN items i ON i.id = pl.item_id
		WHERE pol.ordered_quantity > pol.received_quantity
		ORDER BY po.created_at DESC, pol.id
	`)
	if err != nil {
		return ArrivalList{}, fmt.Errorf("query arrivals: %w", err)
	}
	defer rows.Close()

	result := ArrivalList{Rows: []ArrivalSummary{}}
	for rows.Next() {
		var row ArrivalSummary
		if err := rows.Scan(
			&row.PurchaseOrderLineID,
			&row.PurchaseOrderID,
			&row.OrderNumber,
			&row.ItemID,
			&row.ItemNumber,
			&row.Description,
			&row.OrderedQuantity,
			&row.ReceivedQuantity,
			&row.PendingQuantity,
			&row.SupplierName,
			&row.ExpectedArrivalDate,
			&row.Status,
		); err != nil {
			return ArrivalList{}, fmt.Errorf("scan arrival: %w", err)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) CreateReceipt(ctx context.Context, input ReceiptCreateInput) (ReceiptSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ReceiptSummary{}, fmt.Errorf("begin receipt tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	receiptID := fmt.Sprintf("receipt-%d", now.UnixNano())
	receiptNumber := fmt.Sprintf("RCPT-%s-%03d", now.Format("20060102"), now.Nanosecond()%1000)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO receipts (id, purchase_order_id, receipt_number, source_type, received_by, received_at, note, created_at)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, COALESCE(NULLIF($6, '')::timestamptz, NOW()), $7, NOW())
	`, receiptID, input.PurchaseOrderID, receiptNumber, defaultString(input.SourceType, "manual"), emptyDefault(input.ReceivedBy, "local-user"), input.ReceivedAt, input.Note); err != nil {
		return ReceiptSummary{}, fmt.Errorf("insert receipt: %w", err)
	}

	for index, line := range input.Lines {
		lineID := fmt.Sprintf("receipt-line-%d-%d", now.UnixNano(), index+1)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO receipt_lines (id, receipt_id, purchase_order_line_id, item_id, location_code, received_quantity, note, created_at)
			VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, NOW())
		`, lineID, receiptID, line.PurchaseOrderLineID, line.ItemID, line.LocationCode, line.Quantity, line.Note); err != nil {
			return ReceiptSummary{}, fmt.Errorf("insert receipt line: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE purchase_order_lines
			SET received_quantity = received_quantity + $2,
			    status = CASE WHEN received_quantity + $2 >= ordered_quantity THEN 'received' ELSE 'partially_received' END
			WHERE id = NULLIF($1, '')
		`, line.PurchaseOrderLineID, line.Quantity); err != nil {
			return ReceiptSummary{}, fmt.Errorf("update purchase order line receipt quantity: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO inventory_events (
				id, item_id, location_code, from_location_code, to_location_code, event_type, quantity_delta, note,
				actor_id, source_type, source_id, correlation_id, occurred_at
			) VALUES ($1, $2, $3, '', $3, 'receive', $4, $5, $6, 'receipt', $7, $8, NOW())
		`, fmt.Sprintf("evt-%d-%d", now.UnixNano(), index+1), line.ItemID, line.LocationCode, line.Quantity, line.Note, emptyDefault(input.ReceivedBy, "local-user"), receiptID, lineID); err != nil {
			return ReceiptSummary{}, fmt.Errorf("insert receipt inventory event: %w", err)
		}
		if err := incrementBalanceTx(ctx, tx, line.ItemID, line.LocationCode, line.Quantity); err != nil {
			return ReceiptSummary{}, err
		}
	}

	if err := recordAuditEventTx(ctx, tx, input.ReceivedBy, "receipt.created", "receipt", receiptID, map[string]any{
		"lineCount": len(input.Lines),
	}); err != nil {
		return ReceiptSummary{}, err
	}

	var summary ReceiptSummary
	var receivedAt time.Time
	if err := tx.QueryRowContext(ctx, `
		SELECT id, receipt_number, COALESCE(purchase_order_id, ''), source_type, received_by, received_at, note
		FROM receipts
		WHERE id = $1
	`, receiptID).Scan(&summary.ID, &summary.ReceiptNumber, &summary.PurchaseOrderID, &summary.SourceType, &summary.ReceivedBy, &receivedAt, &summary.Note); err != nil {
		return ReceiptSummary{}, fmt.Errorf("reload receipt: %w", err)
	}
	summary.ReceivedAt = receivedAt.UTC().Format(time.RFC3339)

	if err := tx.Commit(); err != nil {
		return ReceiptSummary{}, fmt.Errorf("commit receipt tx: %w", err)
	}
	return summary, nil
}

func (r *Repository) ImportPreview(ctx context.Context, importType, fileName string, records []byte) (ImportPreviewResult, error) {
	reader := csv.NewReader(bytes.NewReader(records))
	data, err := reader.ReadAll()
	if err != nil {
		return ImportPreviewResult{}, fmt.Errorf("read preview csv: %w", err)
	}
	if len(data) == 0 {
		return ImportPreviewResult{}, fmt.Errorf("csv header is required")
	}

	headers := normalizeCSVHeaders(data[0])
	required := requiredImportHeaders(importType)
	rows := make([]ImportPreviewRow, 0, len(data)-1)
	for index, values := range data[1:] {
		raw := map[string]string{}
		for headerIndex, header := range headers {
			if headerIndex >= len(values) {
				raw[header] = ""
				continue
			}
			raw[header] = values[headerIndex]
		}

		row := ImportPreviewRow{RowNumber: index + 1, Status: "valid", Raw: raw}
		for _, header := range required {
			if strings.TrimSpace(raw[header]) == "" {
				row.Status = "invalid"
				row.Code = "missing_required_column"
				row.Message = fmt.Sprintf("%s is required", header)
				break
			}
		}
		if row.Status == "valid" {
			row.Code, row.Message = previewImportRow(importType, raw)
			if strings.HasPrefix(row.Code, "invalid_") {
				row.Status = "invalid"
			}
		}
		rows = append(rows, row)
	}

	status := "ready"
	for _, row := range rows {
		if row.Status == "invalid" {
			status = "has_errors"
			break
		}
	}

	_ = ctx
	return ImportPreviewResult{
		ImportType: importType,
		FileName:   fileName,
		Status:     status,
		Rows:       rows,
	}, nil
}

func (r *Repository) ImportDetail(ctx context.Context, id string) (ImportDetail, error) {
	var detail ImportDetail
	var createdAt time.Time
	var undoneAt sql.NullTime
	if err := r.db.QueryRowContext(ctx, `
		SELECT id, import_type, status, COALESCE(lifecycle_state, ''), file_name, summary::text, created_at, undone_at
		FROM import_jobs
		WHERE id = $1
	`, id).Scan(&detail.ID, &detail.ImportType, &detail.Status, &detail.LifecycleState, &detail.FileName, &detail.Summary, &createdAt, &undoneAt); err != nil {
		return ImportDetail{}, fmt.Errorf("query import detail: %w", err)
	}
	detail.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if undoneAt.Valid {
		detail.UndoneAt = undoneAt.Time.UTC().Format(time.RFC3339)
	}

	rowRecords, err := r.db.QueryContext(ctx, `
		SELECT row_number, status, code, message, raw_payload::text
		FROM import_rows
		WHERE import_job_id = $1
		ORDER BY row_number
	`, id)
	if err != nil {
		return ImportDetail{}, fmt.Errorf("query import rows: %w", err)
	}
	defer rowRecords.Close()
	detail.Rows = []ImportPreviewRow{}
	for rowRecords.Next() {
		var row ImportPreviewRow
		var rawJSON string
		if err := rowRecords.Scan(&row.RowNumber, &row.Status, &row.Code, &row.Message, &rawJSON); err != nil {
			return ImportDetail{}, fmt.Errorf("scan import row: %w", err)
		}
		row.Raw = map[string]string{}
		_ = json.Unmarshal([]byte(rawJSON), &row.Raw)
		detail.Rows = append(detail.Rows, row)
	}

	effectRows, err := r.db.QueryContext(ctx, `
		SELECT id, target_entity_type, target_entity_id, effect_type
		FROM import_effects
		WHERE import_job_id = $1
		ORDER BY created_at
	`, id)
	if err != nil {
		return ImportDetail{}, fmt.Errorf("query import effects: %w", err)
	}
	defer effectRows.Close()
	detail.Effects = []ImportEffectSummary{}
	for effectRows.Next() {
		var effect ImportEffectSummary
		if err := effectRows.Scan(&effect.ID, &effect.TargetEntityType, &effect.TargetEntityID, &effect.EffectType); err != nil {
			return ImportDetail{}, fmt.Errorf("scan import effect: %w", err)
		}
		detail.Effects = append(detail.Effects, effect)
	}
	return detail, nil
}

func (r *Repository) UndoImport(ctx context.Context, id, actor string) (ImportDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ImportDetail{}, fmt.Errorf("begin import undo tx: %w", err)
	}
	defer tx.Rollback()

	var lifecycleState string
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(lifecycle_state, 'applied') FROM import_jobs WHERE id = $1`, id).Scan(&lifecycleState); err != nil {
		return ImportDetail{}, fmt.Errorf("query import lifecycle state: %w", err)
	}
	if lifecycleState == "undone" {
		return ImportDetail{}, fmt.Errorf("import already undone")
	}

	effects, err := tx.QueryContext(ctx, `
		SELECT target_entity_type, target_entity_id, effect_type, before_state::text
		FROM import_effects
		WHERE import_job_id = $1
		ORDER BY created_at DESC
	`, id)
	if err != nil {
		return ImportDetail{}, fmt.Errorf("query undo import effects: %w", err)
	}
	defer effects.Close()
	for effects.Next() {
		var entityType, entityID, effectType, beforeState string
		if err := effects.Scan(&entityType, &entityID, &effectType, &beforeState); err != nil {
			return ImportDetail{}, fmt.Errorf("scan undo import effect: %w", err)
		}
		switch {
		case effectType == "insert" && entityType == "item":
			if _, err := tx.ExecContext(ctx, `DELETE FROM items WHERE id = $1`, entityID); err != nil {
				return ImportDetail{}, fmt.Errorf("undo imported item insert: %w", err)
			}
		case effectType == "insert" && entityType == "alias":
			if _, err := tx.ExecContext(ctx, `DELETE FROM supplier_item_aliases WHERE id = $1`, entityID); err != nil {
				return ImportDetail{}, fmt.Errorf("undo imported alias insert: %w", err)
			}
		case effectType == "update" && beforeState != "":
			var payload map[string]any
			if err := json.Unmarshal([]byte(beforeState), &payload); err != nil {
				return ImportDetail{}, fmt.Errorf("decode undo import state: %w", err)
			}
			if entityType == "item" {
				if _, err := tx.ExecContext(ctx, `
					UPDATE items
					SET description = $2,
					    manufacturer_key = $3,
					    category_key = $4,
					    default_supplier_id = NULLIF($5, ''),
					    note = $6,
					    updated_at = NOW()
					WHERE id = $1
				`, entityID, stringValue(payload["description"]), stringValue(payload["manufacturer_key"]), stringValue(payload["category_key"]), stringValue(payload["default_supplier_id"]), stringValue(payload["note"])); err != nil {
					return ImportDetail{}, fmt.Errorf("restore imported item: %w", err)
				}
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE import_jobs
		SET lifecycle_state = 'undone',
		    undone_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`, id); err != nil {
		return ImportDetail{}, fmt.Errorf("mark import undone: %w", err)
	}
	if err := recordAuditEventTx(ctx, tx, actor, "import.undone", "import_job", id, map[string]any{}); err != nil {
		return ImportDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return ImportDetail{}, fmt.Errorf("commit import undo tx: %w", err)
	}
	return r.ImportDetail(ctx, id)
}

type inventoryEventInsert struct {
	ItemID            string
	LocationCode      string
	FromLocationCode  string
	ToLocationCode    string
	EventType         string
	QuantityDelta     int
	DeviceScopeID     string
	ActorID           string
	SourceType        string
	SourceID          string
	CorrelationSource string
	Note              string
}

func insertInventoryEventTx(ctx context.Context, tx *sql.Tx, input inventoryEventInsert) (InventoryEventEntry, error) {
	id := fmt.Sprintf("evt-%d", time.Now().UnixNano())
	occurredAt := time.Now().UTC()
	correlationID := input.CorrelationSource
	if correlationID == "" {
		correlationID = fmt.Sprintf("%s:%s:%d", input.EventType, input.ItemID, occurredAt.UnixNano())
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO inventory_events (
			id, item_id, location_code, from_location_code, to_location_code, event_type, quantity_delta, note, device_scope_id,
			actor_id, source_type, source_id, correlation_id, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), $10, $11, $12, $13, $14)
	`,
		id,
		input.ItemID,
		defaultString(input.LocationCode, defaultString(input.ToLocationCode, input.FromLocationCode)),
		input.FromLocationCode,
		input.ToLocationCode,
		input.EventType,
		input.QuantityDelta,
		input.Note,
		input.DeviceScopeID,
		emptyDefault(input.ActorID, "local-user"),
		defaultString(input.SourceType, "manual"),
		input.SourceID,
		correlationID,
		occurredAt,
	); err != nil {
		return InventoryEventEntry{}, fmt.Errorf("insert inventory event: %w", err)
	}

	var itemNumber string
	if err := tx.QueryRowContext(ctx, `SELECT canonical_item_number FROM items WHERE id = $1`, input.ItemID).Scan(&itemNumber); err != nil {
		return InventoryEventEntry{}, fmt.Errorf("load item number for inventory event: %w", err)
	}
	return InventoryEventEntry{
		ID:               id,
		EventType:        input.EventType,
		ItemID:           input.ItemID,
		ItemNumber:       itemNumber,
		FromLocationCode: input.FromLocationCode,
		ToLocationCode:   input.ToLocationCode,
		QuantityDelta:    input.QuantityDelta,
		ActorID:          emptyDefault(input.ActorID, "local-user"),
		SourceType:       defaultString(input.SourceType, "manual"),
		SourceID:         input.SourceID,
		CorrelationID:    correlationID,
		Note:             input.Note,
		OccurredAt:       occurredAt.Format(time.RFC3339),
	}, nil
}

func inventoryEventByIDTx(ctx context.Context, tx *sql.Tx, id string) (InventoryEventEntry, error) {
	var row InventoryEventEntry
	var occurredAt time.Time
	if err := tx.QueryRowContext(ctx, `
		SELECT
			e.id,
			e.event_type,
			e.item_id,
			i.canonical_item_number,
			COALESCE(e.from_location_code, ''),
			COALESCE(e.to_location_code, ''),
			e.quantity_delta,
			COALESCE(e.actor_id, ''),
			COALESCE(e.source_type, ''),
			COALESCE(e.source_id, ''),
			COALESCE(e.correlation_id, ''),
			COALESCE(e.reversed_by_event_id, ''),
			e.note,
			e.occurred_at
		FROM inventory_events e
		JOIN items i ON i.id = e.item_id
		WHERE e.id = $1
		FOR UPDATE
	`, id).Scan(
		&row.ID,
		&row.EventType,
		&row.ItemID,
		&row.ItemNumber,
		&row.FromLocationCode,
		&row.ToLocationCode,
		&row.QuantityDelta,
		&row.ActorID,
		&row.SourceType,
		&row.SourceID,
		&row.CorrelationID,
		&row.ReversedByEventID,
		&row.Note,
		&occurredAt,
	); err != nil {
		return InventoryEventEntry{}, fmt.Errorf("query inventory event: %w", err)
	}
	row.OccurredAt = occurredAt.UTC().Format(time.RFC3339)
	return row, nil
}

func ensureBalanceRowTx(ctx context.Context, tx *sql.Tx, itemID, locationCode string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO locations (code, name, location_type)
		VALUES ($1, $1, 'stockroom')
		ON CONFLICT (code) DO NOTHING
	`, locationCode); err != nil {
		return fmt.Errorf("ensure inventory location: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO inventory_balances (item_id, location_code, on_hand_quantity, reserved_quantity, available_quantity, updated_at)
		VALUES ($1, $2, 0, 0, 0, NOW())
		ON CONFLICT (item_id, location_code) DO NOTHING
	`, itemID, locationCode); err != nil {
		return fmt.Errorf("ensure inventory balance row: %w", err)
	}
	return nil
}

func incrementBalanceTx(ctx context.Context, tx *sql.Tx, itemID, locationCode string, quantity int) error {
	if err := ensureBalanceRowTx(ctx, tx, itemID, locationCode); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE inventory_balances
		SET on_hand_quantity = on_hand_quantity + $3,
		    available_quantity = (on_hand_quantity + $3) - reserved_quantity,
		    updated_at = NOW()
		WHERE item_id = $1 AND location_code = $2
	`, itemID, locationCode, quantity); err != nil {
		return fmt.Errorf("increment inventory balance: %w", err)
	}
	return nil
}

func decrementBalanceTx(ctx context.Context, tx *sql.Tx, itemID, locationCode string, quantity int) error {
	if err := ensureBalanceRowTx(ctx, tx, itemID, locationCode); err != nil {
		return err
	}
	var onHand, reserved int
	if err := tx.QueryRowContext(ctx, `
		SELECT on_hand_quantity, reserved_quantity
		FROM inventory_balances
		WHERE item_id = $1 AND location_code = $2
		FOR UPDATE
	`, itemID, locationCode).Scan(&onHand, &reserved); err != nil {
		return fmt.Errorf("lock inventory balance: %w", err)
	}
	if onHand < quantity {
		return fmt.Errorf("insufficient on-hand quantity at %s", locationCode)
	}
	if onHand-quantity < reserved {
		return fmt.Errorf("insufficient free stock at %s", locationCode)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE inventory_balances
		SET on_hand_quantity = on_hand_quantity - $3,
		    available_quantity = (on_hand_quantity - $3) - reserved_quantity,
		    updated_at = NOW()
		WHERE item_id = $1 AND location_code = $2
	`, itemID, locationCode, quantity); err != nil {
		return fmt.Errorf("decrement inventory balance: %w", err)
	}
	return nil
}

func adjustReservedQuantityTx(ctx context.Context, tx *sql.Tx, itemID, locationCode string, delta int) error {
	if err := ensureBalanceRowTx(ctx, tx, itemID, locationCode); err != nil {
		return err
	}
	var onHand, reserved int
	if err := tx.QueryRowContext(ctx, `
		SELECT on_hand_quantity, reserved_quantity
		FROM inventory_balances
		WHERE item_id = $1 AND location_code = $2
		FOR UPDATE
	`, itemID, locationCode).Scan(&onHand, &reserved); err != nil {
		return fmt.Errorf("lock inventory balance for reservation: %w", err)
	}
	nextReserved := reserved + delta
	if nextReserved < 0 {
		return fmt.Errorf("reserved quantity cannot be negative")
	}
	if nextReserved > onHand {
		return fmt.Errorf("reserved quantity cannot exceed on-hand quantity")
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE inventory_balances
		SET reserved_quantity = $3,
		    available_quantity = on_hand_quantity - $3,
		    updated_at = NOW()
		WHERE item_id = $1 AND location_code = $2
	`, itemID, locationCode, nextReserved); err != nil {
		return fmt.Errorf("update reserved inventory quantity: %w", err)
	}
	return nil
}

func (r *Repository) reservationDetail(ctx context.Context, db queryable, id string) (ReservationDetail, error) {
	var detail ReservationDetail
	var neededBy, plannedUse, holdUntil, fulfilledAt, releasedAt sql.NullTime
	if err := db.QueryRowContext(ctx, `
		SELECT
			r.id,
			r.item_id,
			i.canonical_item_number,
			i.description,
			r.device_scope_id,
			ds.device_key,
			ds.scope_key,
			r.quantity,
			COALESCE((SELECT SUM(quantity) FROM reservation_allocations WHERE reservation_id = r.id AND status = 'allocated'), 0),
			r.status,
			r.purpose,
			r.priority,
			r.needed_by_at,
			r.planned_use_at,
			r.hold_until_at,
			r.fulfilled_at,
			r.released_at,
			r.cancellation_reason,
			r.requested_by,
			r.note
		FROM reservations r
		JOIN items i ON i.id = r.item_id
		JOIN device_scopes ds ON ds.id = r.device_scope_id
		WHERE r.id = $1
	`, id).Scan(
		&detail.ID,
		&detail.ItemID,
		&detail.ItemNumber,
		&detail.Description,
		&detail.DeviceScopeID,
		&detail.Device,
		&detail.Scope,
		&detail.Quantity,
		&detail.AllocatedQuantity,
		&detail.Status,
		&detail.Purpose,
		&detail.Priority,
		&neededBy,
		&plannedUse,
		&holdUntil,
		&fulfilledAt,
		&releasedAt,
		&detail.CancellationReason,
		&detail.RequestedBy,
		&detail.Note,
	); err != nil {
		return ReservationDetail{}, fmt.Errorf("query reservation detail: %w", err)
	}
	detail.NeededByAt = nullableTimeString(neededBy)
	detail.PlannedUseAt = nullableTimeString(plannedUse)
	detail.HoldUntilAt = nullableTimeString(holdUntil)
	detail.FulfilledAt = nullableTimeString(fulfilledAt)
	detail.ReleasedAt = nullableTimeString(releasedAt)

	rows, err := db.QueryContext(ctx, `
		SELECT id, location_code, quantity, status, allocated_at, released_at, note
		FROM reservation_allocations
		WHERE reservation_id = $1
		ORDER BY allocated_at
	`, id)
	if err != nil {
		return ReservationDetail{}, fmt.Errorf("query reservation allocations detail: %w", err)
	}
	defer rows.Close()
	detail.Allocations = []ReservationAllocation{}
	for rows.Next() {
		var allocation ReservationAllocation
		var allocatedAt time.Time
		var released sql.NullTime
		if err := rows.Scan(&allocation.ID, &allocation.LocationCode, &allocation.Quantity, &allocation.Status, &allocatedAt, &released, &allocation.Note); err != nil {
			return ReservationDetail{}, fmt.Errorf("scan reservation allocation detail: %w", err)
		}
		allocation.AllocatedAt = allocatedAt.UTC().Format(time.RFC3339)
		allocation.ReleasedAt = nullableTimeString(released)
		detail.Allocations = append(detail.Allocations, allocation)
	}
	return detail, rows.Err()
}

func (r *Repository) reservationDetailTx(ctx context.Context, tx *sql.Tx, id string) (ReservationDetail, error) {
	return r.reservationDetail(ctx, tx, id)
}

func (r *Repository) recordReservationEventTx(ctx context.Context, tx *sql.Tx, reservationID, eventType string, quantity int, actorID string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["quantity"] = quantity
	payload, _ := json.Marshal(metadata)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO reservation_events (id, reservation_id, event_type, quantity, actor_id, metadata, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, NOW())
	`, fmt.Sprintf("reservation-event-%d", time.Now().UnixNano()), reservationID, eventType, quantity, emptyDefault(actorID, "local-user"), string(payload)); err != nil {
		return fmt.Errorf("insert reservation event: %w", err)
	}
	return nil
}

func (r *Repository) refreshReservationStatusTx(ctx context.Context, tx *sql.Tx, reservationID string) error {
	var totalQuantity int
	if err := tx.QueryRowContext(ctx, `SELECT quantity FROM reservations WHERE id = $1 FOR UPDATE`, reservationID).Scan(&totalQuantity); err != nil {
		return fmt.Errorf("lock reservation for status refresh: %w", err)
	}
	var allocated int
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(quantity), 0)
		FROM reservation_allocations
		WHERE reservation_id = $1 AND status = 'allocated'
	`, reservationID).Scan(&allocated); err != nil {
		return fmt.Errorf("query reservation allocated quantity: %w", err)
	}
	status := "requested"
	releasedAt := sql.NullTime{}
	if allocated >= totalQuantity && totalQuantity > 0 {
		status = "allocated"
	} else if allocated > 0 {
		status = "partially_allocated"
	} else {
		status = "awaiting_stock"
		releasedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE reservations
		SET status = $2,
		    released_at = CASE WHEN $3::timestamptz IS NULL THEN released_at ELSE $3 END,
		    updated_at = NOW()
		WHERE id = $1
	`, reservationID, status, nullTimeArg(releasedAt)); err != nil {
		return fmt.Errorf("update reservation status: %w", err)
	}
	return nil
}

func requiredImportHeaders(importType string) []string {
	switch strings.TrimSpace(importType) {
	case "items_with_aliases":
		return []string{"canonical_item_number"}
	case "items":
		return []string{"canonical_item_number", "description", "manufacturer", "category"}
	case "aliases":
		return []string{"supplier_id", "canonical_item_number", "supplier_item_number", "units_per_order"}
	default:
		return []string{}
	}
}

func previewImportRow(importType string, raw map[string]string) (string, string) {
	switch strings.TrimSpace(importType) {
	case "items_with_aliases":
		description := strings.TrimSpace(raw["description"])
		manufacturer := strings.TrimSpace(raw["manufacturer"])
		category := strings.TrimSpace(raw["category"])
		defaultSupplierID := strings.TrimSpace(raw["default_supplier_id"])
		supplierID := strings.TrimSpace(raw["supplier_id"])
		aliasNumber := strings.TrimSpace(raw["supplier_item_number"])
		unitsPerOrder := strings.TrimSpace(raw["units_per_order"])

		hasDescription := description != ""
		hasManufacturer := manufacturer != ""
		hasCategory := category != ""
		hasItemFields := hasDescription || hasManufacturer || hasCategory
		hasCompleteItem := hasDescription && hasManufacturer && hasCategory
		hasAliasFields := aliasNumber != "" || supplierID != "" || unitsPerOrder != ""
		aliasSupplierID := supplierID
		if aliasSupplierID == "" {
			aliasSupplierID = defaultSupplierID
		}

		if hasItemFields && !hasCompleteItem {
			return "invalid_partial_item", "description, manufacturer, and category are required together for item rows"
		}
		if !hasCompleteItem && !hasAliasFields {
			return "invalid_empty_master_row", "row must include item fields or supplier alias fields"
		}
		if hasAliasFields && (aliasSupplierID == "" || aliasNumber == "") {
			return "invalid_alias", "supplier_id or default_supplier_id, and supplier_item_number are required for alias rows"
		}
		if hasCompleteItem && hasAliasFields {
			return "preview_item_with_alias", "Row is ready to import as item with supplier alias"
		}
		if hasCompleteItem {
			return "preview_item", "Row is ready to import as master item"
		}
		return "preview_alias_for_existing_item", "Row is ready to import as alias for an existing canonical item"
	case "items":
		return "preview_item", "Row is ready to import as master item"
	case "aliases":
		return "preview_alias", "Row is ready to import as supplier alias"
	default:
		return "invalid_import_type", "unsupported import type"
	}
}

func recordAuditEventTx(ctx context.Context, tx *sql.Tx, actorID, eventType, entityType, entityID string, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	body, _ := json.Marshal(payload)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_events (actor_id, event_type, entity_type, entity_id, payload, created_at)
		VALUES (NULLIF($1, ''), $2, $3, $4, $5::jsonb, NOW())
	`, actorID, eventType, entityType, entityID, string(body)); err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

type queryable interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func nullableTimeString(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339)
}

func nullTimeArg(value sql.NullTime) any {
	if !value.Valid {
		return nil
	}
	return value.Time
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func (e InventoryEventEntry) QuantityDeltaOrTransfer() int {
	if e.QuantityDelta < 0 {
		return -e.QuantityDelta
	}
	return e.QuantityDelta
}

func snapshotSignature(rows []InventorySnapshotRow) string {
	if rows == nil {
		rows = []InventorySnapshotRow{}
	}
	payload, _ := json.Marshal(rows)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:16])
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
