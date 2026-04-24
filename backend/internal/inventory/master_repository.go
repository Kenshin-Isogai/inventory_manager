package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (r *Repository) MasterItems(ctx context.Context) (MasterItemList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, canonical_item_number, description, manufacturer_key, category_key, COALESCE(default_supplier_id, ''), note, lifecycle_status
		FROM items
		ORDER BY canonical_item_number
	`)
	if err != nil {
		return MasterItemList{}, fmt.Errorf("query master items: %w", err)
	}
	defer rows.Close()
	result := MasterItemList{Rows: []MasterItemRecord{}}
	for rows.Next() {
		var row MasterItemRecord
		if err := rows.Scan(&row.ID, &row.ItemNumber, &row.Description, &row.ManufacturerKey, &row.CategoryKey, &row.DefaultSupplierID, &row.Note, &row.LifecycleStatus); err != nil {
			return MasterItemList{}, fmt.Errorf("scan master item: %w", err)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) UpsertMasterItem(ctx context.Context, input MasterItemUpsertInput) (MasterItemRecord, error) {
	id := input.ID
	if id == "" {
		id = fmt.Sprintf("item-%d", time.Now().UnixNano())
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description, default_supplier_id, note, lifecycle_status, updated_at)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, NOW())
		ON CONFLICT (id) DO UPDATE
		SET manufacturer_key = EXCLUDED.manufacturer_key,
		    category_key = EXCLUDED.category_key,
		    canonical_item_number = EXCLUDED.canonical_item_number,
		    description = EXCLUDED.description,
		    default_supplier_id = EXCLUDED.default_supplier_id,
		    note = EXCLUDED.note,
		    lifecycle_status = EXCLUDED.lifecycle_status,
		    updated_at = NOW()
	`, id, input.ManufacturerKey, input.CategoryKey, input.ItemNumber, input.Description, input.DefaultSupplierID, input.Note, defaultString(input.LifecycleStatus, "active")); err != nil {
		return MasterItemRecord{}, fmt.Errorf("upsert master item: %w", err)
	}
	var result MasterItemRecord
	if err := r.db.QueryRowContext(ctx, `
		SELECT id, canonical_item_number, description, manufacturer_key, category_key, COALESCE(default_supplier_id, ''), note, lifecycle_status
		FROM items
		WHERE id = $1
	`, id).Scan(&result.ID, &result.ItemNumber, &result.Description, &result.ManufacturerKey, &result.CategoryKey, &result.DefaultSupplierID, &result.Note, &result.LifecycleStatus); err != nil {
		return MasterItemRecord{}, fmt.Errorf("reload master item: %w", err)
	}
	return result, nil
}

func (r *Repository) MasterItemDetail(ctx context.Context, id string) (MasterItemRecord, error) {
	var result MasterItemRecord
	if err := r.db.QueryRowContext(ctx, `
		SELECT id, canonical_item_number, description, manufacturer_key, category_key, COALESCE(default_supplier_id, ''), note, lifecycle_status
		FROM items
		WHERE id = $1
	`, id).Scan(&result.ID, &result.ItemNumber, &result.Description, &result.ManufacturerKey, &result.CategoryKey, &result.DefaultSupplierID, &result.Note, &result.LifecycleStatus); err != nil {
		return MasterItemRecord{}, fmt.Errorf("query master item detail: %w", err)
	}
	return result, nil
}

func (r *Repository) DeleteMasterItem(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete master item: %w", err)
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return fmt.Errorf("item not found: %s", id)
	}
	return nil
}

func (r *Repository) Suppliers(ctx context.Context) (SupplierList, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, COALESCE(contact_name, ''), COALESCE(contact_email, '') FROM suppliers ORDER BY name`)
	if err != nil {
		return SupplierList{}, fmt.Errorf("query suppliers: %w", err)
	}
	defer rows.Close()
	result := SupplierList{Rows: []SupplierRecord{}}
	for rows.Next() {
		var row SupplierRecord
		if err := rows.Scan(&row.ID, &row.Name, &row.ContactName, &row.ContactEmail); err != nil {
			return SupplierList{}, fmt.Errorf("scan supplier: %w", err)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) UpsertSupplier(ctx context.Context, input SupplierUpsertInput) (SupplierRecord, error) {
	id := input.ID
	if id == "" {
		id = fmt.Sprintf("sup-%d", time.Now().UnixNano())
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO suppliers (id, name, contact_name, contact_email, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
		    contact_name = EXCLUDED.contact_name,
		    contact_email = EXCLUDED.contact_email,
		    updated_at = NOW()
	`, id, input.Name, input.ContactName, input.ContactEmail); err != nil {
		return SupplierRecord{}, fmt.Errorf("upsert supplier: %w", err)
	}
	var result SupplierRecord
	if err := r.db.QueryRowContext(ctx, `SELECT id, name, COALESCE(contact_name, ''), COALESCE(contact_email, '') FROM suppliers WHERE id = $1`, id).Scan(&result.ID, &result.Name, &result.ContactName, &result.ContactEmail); err != nil {
		return SupplierRecord{}, fmt.Errorf("reload supplier: %w", err)
	}
	return result, nil
}

func (r *Repository) Aliases(ctx context.Context) (AliasList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sia.id, sia.supplier_id, s.name, sia.item_id, i.canonical_item_number, sia.supplier_item_number, sia.units_per_order
		FROM supplier_item_aliases sia
		JOIN suppliers s ON s.id = sia.supplier_id
		JOIN items i ON i.id = sia.item_id
		ORDER BY s.name, sia.supplier_item_number
	`)
	if err != nil {
		return AliasList{}, fmt.Errorf("query aliases: %w", err)
	}
	defer rows.Close()
	result := AliasList{Rows: []SupplierAliasSummary{}}
	for rows.Next() {
		var row SupplierAliasSummary
		if err := rows.Scan(&row.ID, &row.SupplierID, &row.SupplierName, &row.ItemID, &row.CanonicalItemNumber, &row.SupplierItemNumber, &row.UnitsPerOrder); err != nil {
			return AliasList{}, fmt.Errorf("scan alias: %w", err)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) UpsertAlias(ctx context.Context, input AliasUpsertInput) (SupplierAliasSummary, error) {
	id := input.ID
	if id == "" {
		id = fmt.Sprintf("alias-%d", time.Now().UnixNano())
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO supplier_item_aliases (id, item_id, supplier_id, supplier_item_number, units_per_order, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (id) DO UPDATE
		SET item_id = EXCLUDED.item_id,
		    supplier_id = EXCLUDED.supplier_id,
		    supplier_item_number = EXCLUDED.supplier_item_number,
		    units_per_order = EXCLUDED.units_per_order,
		    updated_at = NOW()
	`, id, input.ItemID, input.SupplierID, input.SupplierItemNumber, input.UnitsPerOrder); err != nil {
		return SupplierAliasSummary{}, fmt.Errorf("upsert alias: %w", err)
	}
	var result SupplierAliasSummary
	if err := r.db.QueryRowContext(ctx, `
		SELECT sia.id, sia.supplier_id, s.name, sia.item_id, i.canonical_item_number, sia.supplier_item_number, sia.units_per_order
		FROM supplier_item_aliases sia
		JOIN suppliers s ON s.id = sia.supplier_id
		JOIN items i ON i.id = sia.item_id
		WHERE sia.id = $1
	`, id).Scan(&result.ID, &result.SupplierID, &result.SupplierName, &result.ItemID, &result.CanonicalItemNumber, &result.SupplierItemNumber, &result.UnitsPerOrder); err != nil {
		return SupplierAliasSummary{}, fmt.Errorf("reload alias: %w", err)
	}
	return result, nil
}

func (r *Repository) Devices(ctx context.Context) (DeviceList, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, device_key, name, device_type, status FROM devices ORDER BY device_key`)
	if err != nil {
		return DeviceList{}, fmt.Errorf("query devices: %w", err)
	}
	defer rows.Close()
	result := DeviceList{Rows: []DeviceRecord{}}
	for rows.Next() {
		var row DeviceRecord
		if err := rows.Scan(&row.ID, &row.DeviceKey, &row.Name, &row.DeviceType, &row.Status); err != nil {
			return DeviceList{}, fmt.Errorf("scan device: %w", err)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) UpsertDevice(ctx context.Context, input DeviceUpsertInput) (DeviceRecord, error) {
	id := input.ID
	if id == "" {
		id = fmt.Sprintf("device-%s", strings.ToLower(input.DeviceKey))
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO devices (id, device_key, name, device_type, status, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (id) DO UPDATE
		SET device_key = EXCLUDED.device_key,
		    name = EXCLUDED.name,
		    device_type = EXCLUDED.device_type,
		    status = EXCLUDED.status,
		    updated_at = NOW()
	`, id, input.DeviceKey, input.Name, input.DeviceType, defaultString(input.Status, "active")); err != nil {
		return DeviceRecord{}, fmt.Errorf("upsert device: %w", err)
	}
	var result DeviceRecord
	if err := r.db.QueryRowContext(ctx, `SELECT id, device_key, name, device_type, status FROM devices WHERE id = $1`, id).Scan(&result.ID, &result.DeviceKey, &result.Name, &result.DeviceType, &result.Status); err != nil {
		return DeviceRecord{}, fmt.Errorf("reload device: %w", err)
	}
	return result, nil
}

var deviceScopeTypes = map[string]struct{}{
	"system":       {},
	"assembly":     {},
	"module":       {},
	"area":         {},
	"work_package": {},
}

type deviceScopeParentRecord struct {
	DeviceKey string
	SystemKey string
}

func normalizeDeviceScopeType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func isValidDeviceScopeType(value string) bool {
	_, ok := deviceScopeTypes[value]
	return ok
}

func (r *Repository) deviceScopeParent(ctx context.Context, parentScopeID string) (deviceScopeParentRecord, error) {
	var parent deviceScopeParentRecord
	if err := r.db.QueryRowContext(ctx, `
		SELECT device_key, COALESCE(system_key, '')
		FROM device_scopes
		WHERE id = $1
	`, parentScopeID).Scan(&parent.DeviceKey, &parent.SystemKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return deviceScopeParentRecord{}, fmt.Errorf("parent scope not found: %s", parentScopeID)
		}
		return deviceScopeParentRecord{}, fmt.Errorf("query parent scope: %w", err)
	}
	return parent, nil
}

func (r *Repository) deviceScopeWouldCreateCycle(ctx context.Context, scopeID, parentScopeID string) (bool, error) {
	var cycle bool
	if err := r.db.QueryRowContext(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_scope_id
			FROM device_scopes
			WHERE id = $1
			UNION ALL
			SELECT ds.id, ds.parent_scope_id
			FROM device_scopes ds
			JOIN ancestors a ON ds.id = a.parent_scope_id
		)
		SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = $2)
	`, parentScopeID, scopeID).Scan(&cycle); err != nil {
		return false, fmt.Errorf("check scope hierarchy cycle: %w", err)
	}
	return cycle, nil
}

func (r *Repository) DeviceScopes(ctx context.Context) (DeviceScopeList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ds.id,
			COALESCE(ds.device_id, ''),
			ds.device_key,
			COALESCE(ds.parent_scope_id, ''),
			COALESCE(parent.scope_key, ''),
			COALESCE(ds.system_key, ''),
			COALESCE(ss.name, ''),
			ds.scope_key,
			ds.scope_name,
			ds.scope_type,
			COALESCE(ds.owner_department_key, ''),
			ds.status
		FROM device_scopes ds
		LEFT JOIN device_scopes parent ON parent.id = ds.parent_scope_id
		LEFT JOIN scope_systems ss ON ss.key = ds.system_key
		ORDER BY ds.device_key, COALESCE(ds.system_key, ''), COALESCE(parent.scope_key, ''), ds.scope_type, ds.scope_key
	`)
	if err != nil {
		return DeviceScopeList{}, fmt.Errorf("query device scopes: %w", err)
	}
	defer rows.Close()
	result := DeviceScopeList{Rows: []DeviceScopeRecord{}}
	for rows.Next() {
		var row DeviceScopeRecord
		if err := rows.Scan(
			&row.ID, &row.DeviceID, &row.DeviceKey, &row.ParentScopeID, &row.ParentScopeKey,
			&row.SystemKey, &row.SystemName, &row.ScopeKey, &row.ScopeName, &row.ScopeType, &row.OwnerDepartmentKey, &row.Status,
		); err != nil {
			return DeviceScopeList{}, fmt.Errorf("scan device scope: %w", err)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) UpsertDeviceScope(ctx context.Context, input DeviceScopeUpsertInput) (DeviceScopeRecord, error) {
	deviceKey := strings.TrimSpace(input.DeviceKey)
	scopeKey := normalizeLookupKey(input.ScopeKey)
	scopeType := normalizeDeviceScopeType(defaultString(input.ScopeType, "assembly"))
	parentScopeID := strings.TrimSpace(input.ParentScopeID)
	systemKey := normalizeLookupKey(input.SystemKey)
	ownerDepartmentKey := normalizeLookupKey(input.OwnerDepartmentKey)
	if deviceKey == "" || scopeKey == "" {
		return DeviceScopeRecord{}, fmt.Errorf("deviceKey and scopeKey are required")
	}
	if !isValidDeviceScopeType(scopeType) {
		return DeviceScopeRecord{}, fmt.Errorf("unsupported scopeType: %s", input.ScopeType)
	}
	deviceID := input.DeviceID
	if deviceID == "" {
		if err := r.db.QueryRowContext(ctx, `SELECT id FROM devices WHERE device_key = $1`, deviceKey).Scan(&deviceID); err != nil {
			return DeviceScopeRecord{}, fmt.Errorf("resolve device id: %w", err)
		}
	}
	id := input.ID
	if id == "" {
		id = fmt.Sprintf("scope-%d", time.Now().UnixNano())
	}
	if scopeType == "system" {
		if parentScopeID != "" {
			return DeviceScopeRecord{}, fmt.Errorf("system scopes cannot have a parentScopeId")
		}
		if systemKey == "" {
			systemKey = scopeKey
		}
		if scopeKey != systemKey {
			return DeviceScopeRecord{}, fmt.Errorf("system scope key must match systemKey")
		}
		if ownerDepartmentKey == "" {
			ownerDepartmentKey = systemKey
		}
	} else {
		if parentScopeID == "" {
			return DeviceScopeRecord{}, fmt.Errorf("non-system scopes require a parentScopeId")
		}
		parent, err := r.deviceScopeParent(ctx, parentScopeID)
		if err != nil {
			return DeviceScopeRecord{}, err
		}
		if parent.DeviceKey != deviceKey {
			return DeviceScopeRecord{}, fmt.Errorf("parent scope %s belongs to device %s, expected %s", parentScopeID, parent.DeviceKey, deviceKey)
		}
		if parent.SystemKey == "" {
			return DeviceScopeRecord{}, fmt.Errorf("parent scope %s is missing a systemKey", parentScopeID)
		}
		if systemKey != "" && systemKey != parent.SystemKey {
			return DeviceScopeRecord{}, fmt.Errorf("child scope systemKey must match parent systemKey %s", parent.SystemKey)
		}
		if input.ID != "" {
			cycle, err := r.deviceScopeWouldCreateCycle(ctx, input.ID, parentScopeID)
			if err != nil {
				return DeviceScopeRecord{}, err
			}
			if cycle {
				return DeviceScopeRecord{}, fmt.Errorf("parent scope would create a hierarchy cycle")
			}
		}
		systemKey = parent.SystemKey
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO device_scopes (id, device_id, device_key, parent_scope_id, system_key, scope_key, scope_name, scope_type, owner_department_key, status, description, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8, NULLIF($9, ''), $10, '', NOW())
		ON CONFLICT (id) DO UPDATE
		SET device_id = EXCLUDED.device_id,
		    device_key = EXCLUDED.device_key,
		    parent_scope_id = EXCLUDED.parent_scope_id,
		    system_key = EXCLUDED.system_key,
		    scope_key = EXCLUDED.scope_key,
		    scope_name = EXCLUDED.scope_name,
		    scope_type = EXCLUDED.scope_type,
		    owner_department_key = EXCLUDED.owner_department_key,
		    status = EXCLUDED.status,
		    updated_at = NOW()
	`, id, deviceID, deviceKey, parentScopeID, systemKey, scopeKey, defaultString(input.ScopeName, scopeKey), scopeType, ownerDepartmentKey, defaultString(input.Status, "active")); err != nil {
		return DeviceScopeRecord{}, fmt.Errorf("upsert device scope: %w", err)
	}
	var result DeviceScopeRecord
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			ds.id,
			COALESCE(ds.device_id, ''),
			ds.device_key,
			COALESCE(ds.parent_scope_id, ''),
			COALESCE(parent.scope_key, ''),
			COALESCE(ds.system_key, ''),
			COALESCE(ss.name, ''),
			ds.scope_key,
			ds.scope_name,
			ds.scope_type,
			COALESCE(ds.owner_department_key, ''),
			ds.status
		FROM device_scopes ds
		LEFT JOIN device_scopes parent ON parent.id = ds.parent_scope_id
		LEFT JOIN scope_systems ss ON ss.key = ds.system_key
		WHERE ds.id = $1
	`, id).Scan(
		&result.ID, &result.DeviceID, &result.DeviceKey, &result.ParentScopeID, &result.ParentScopeKey,
		&result.SystemKey, &result.SystemName, &result.ScopeKey, &result.ScopeName, &result.ScopeType, &result.OwnerDepartmentKey, &result.Status,
	); err != nil {
		return DeviceScopeRecord{}, fmt.Errorf("reload device scope: %w", err)
	}
	return result, nil
}

func (r *Repository) ScopeSystems(ctx context.Context) (ScopeSystemList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ss.key, ss.name, ss.description, ss.status, COUNT(ds.id) AS in_use_count
		FROM scope_systems ss
		LEFT JOIN device_scopes ds ON ds.system_key = ss.key
		GROUP BY ss.key, ss.name, ss.description, ss.status
		ORDER BY ss.key
	`)
	if err != nil {
		return ScopeSystemList{}, fmt.Errorf("query scope systems: %w", err)
	}
	defer rows.Close()
	result := ScopeSystemList{Rows: []ScopeSystemRecord{}}
	for rows.Next() {
		var row ScopeSystemRecord
		if err := rows.Scan(&row.Key, &row.Name, &row.Description, &row.Status, &row.InUseCount); err != nil {
			return ScopeSystemList{}, fmt.Errorf("scan scope system: %w", err)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Repository) UpsertScopeSystem(ctx context.Context, input ScopeSystemUpsertInput) (ScopeSystemRecord, error) {
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO scope_systems (key, name, description, status, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (key) DO UPDATE
		SET name = EXCLUDED.name,
		    description = EXCLUDED.description,
		    status = EXCLUDED.status,
		    updated_at = NOW()
	`, input.Key, input.Name, input.Description, defaultString(input.Status, "active")); err != nil {
		return ScopeSystemRecord{}, fmt.Errorf("upsert scope system: %w", err)
	}
	var result ScopeSystemRecord
	if err := r.db.QueryRowContext(ctx, `
		SELECT ss.key, ss.name, ss.description, ss.status, COUNT(ds.id) AS in_use_count
		FROM scope_systems ss
		LEFT JOIN device_scopes ds ON ds.system_key = ss.key
		WHERE ss.key = $1
		GROUP BY ss.key, ss.name, ss.description, ss.status
	`, input.Key).Scan(&result.Key, &result.Name, &result.Description, &result.Status, &result.InUseCount); err != nil {
		return ScopeSystemRecord{}, fmt.Errorf("reload scope system: %w", err)
	}
	return result, nil
}

func (r *Repository) DeleteScopeSystem(ctx context.Context, key string) error {
	var inUse int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM device_scopes WHERE system_key = $1`, key).Scan(&inUse); err != nil {
		return fmt.Errorf("count scope system usage: %w", err)
	}
	if inUse > 0 {
		return fmt.Errorf("scope system is in use by %d scope(s)", inUse)
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM scope_systems WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("delete scope system: %w", err)
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return fmt.Errorf("scope system not found: %s", key)
	}
	return nil
}

func (r *Repository) UpsertLocation(ctx context.Context, input LocationUpsertInput) (LocationSummary, error) {
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO locations (code, name, location_type, is_active, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (code) DO UPDATE
		SET name = EXCLUDED.name,
		    location_type = EXCLUDED.location_type,
		    is_active = EXCLUDED.is_active,
		    updated_at = NOW()
	`, input.Code, input.Name, defaultString(input.LocationType, "stockroom"), input.IsActive); err != nil {
		return LocationSummary{}, fmt.Errorf("upsert location: %w", err)
	}
	var result LocationSummary
	if err := r.db.QueryRowContext(ctx, `
		SELECT l.code, l.name, l.location_type, l.is_active, COALESCE(SUM(ib.on_hand_quantity), 0), COALESCE(SUM(ib.reserved_quantity), 0), COALESCE(SUM(ib.available_quantity), 0)
		FROM locations l
		LEFT JOIN inventory_balances ib ON ib.location_code = l.code
		WHERE l.code = $1
		GROUP BY l.code, l.name, l.location_type, l.is_active
	`, input.Code).Scan(&result.Code, &result.Name, &result.LocationType, &result.IsActive, &result.OnHandQuantity, &result.ReservedQuantity, &result.AvailableQuantity); err != nil {
		return LocationSummary{}, fmt.Errorf("reload location: %w", err)
	}
	return result, nil
}
