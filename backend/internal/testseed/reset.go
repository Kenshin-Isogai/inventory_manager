package testseed

import (
	"context"
	"database/sql"
	"fmt"
)

var resetTables = []string{
	"import_effects", "import_rows", "import_jobs",
	"procurement_status_history", "procurement_status_projections",
	"purchase_order_lines", "purchase_orders",
	"procurement_lines", "procurement_batches",
	"quotation_lines", "supplier_quotations",
	"ocr_result_lines", "ocr_job_results", "ocr_jobs",
	"receipt_lines", "receipts",
	"external_project_budget_categories", "external_projects",
	"reservation_events", "reservation_allocations", "reservations",
	"inventory_event_links", "inventory_events", "inventory_balances",
	"scope_assembly_usage", "scope_assembly_requirements",
	"scope_item_requirements", "assembly_components", "assemblies",
	"scope_departments", "supplier_item_aliases",
	"items",
	"device_scopes", "devices", "locations",
	"suppliers", "departments", "categories", "manufacturers",
	"scope_systems",
	"user_status_history", "user_roles", "audit_events", "app_users",
}

var masterSeedStatements = []string{
	"INSERT INTO manufacturers (key, name) VALUES ('omron', 'Omron'), ('phoenix', 'Phoenix Contact'), ('molex', 'Molex') ON CONFLICT DO NOTHING",
	"INSERT INTO categories (key, name) VALUES ('relay', 'Relay'), ('terminal', 'Terminal Block'), ('connector', 'Connector') ON CONFLICT DO NOTHING",
	"INSERT INTO departments (key, name) VALUES ('production', 'Production'), ('procurement', 'Procurement'), ('optics', 'Optics'), ('mechanical', 'Mechanical'), ('controls', 'Controls') ON CONFLICT DO NOTHING",
	"INSERT INTO suppliers (id, name, contact_name, contact_email) VALUES ('sup-thorlabs', 'Thorlabs Japan', 'Sales Desk', 'sales@thorlabs.example'), ('sup-misumi', 'MISUMI', 'Support', 'support@misumi.example') ON CONFLICT DO NOTHING",
	"INSERT INTO locations (code, name, location_type, is_active) VALUES ('TOKYO-A1', 'Tokyo Warehouse A1', 'stockroom', TRUE), ('TOKYO-B2', 'Tokyo Warehouse B2', 'stockroom', TRUE), ('TOKYO-C1', 'Tokyo Warehouse C1', 'stockroom', TRUE), ('INCOMING', 'Incoming staging', 'receiving', TRUE) ON CONFLICT DO NOTHING",
	"INSERT INTO scope_systems (key, name, description, status) VALUES ('controls', 'Control System', 'Control engineering', 'active'), ('mechanical', 'Mechanical System', 'Mechanical engineering', 'active') ON CONFLICT DO NOTHING",
	"INSERT INTO devices (id, device_key, name, device_type, status) VALUES ('dev-er2', 'ER2', 'ER2 Device', 'standard', 'active'), ('dev-mk4', 'MK4', 'MK4 Device', 'standard', 'active') ON CONFLICT DO NOTHING",
	"INSERT INTO device_scopes (id, device_id, device_key, parent_scope_id, system_key, scope_key, scope_name, scope_type, owner_department_key, status, description) VALUES ('ds-er2-controls', 'dev-er2', 'ER2', NULL, 'controls', 'controls', 'Control System', 'system', 'controls', 'active', 'ER2 controls system') ON CONFLICT (device_key, scope_key) DO NOTHING",
	"INSERT INTO device_scopes (id, device_id, device_key, parent_scope_id, system_key, scope_key, scope_name, scope_type, owner_department_key, status, description) VALUES ('ds-mk4-mechanical', 'dev-mk4', 'MK4', NULL, 'mechanical', 'mechanical', 'Mechanical System', 'system', 'mechanical', 'active', 'MK4 mechanical system') ON CONFLICT (device_key, scope_key) DO NOTHING",
	"INSERT INTO device_scopes (id, device_id, device_key, parent_scope_id, system_key, scope_key, scope_name, scope_type, owner_department_key, status, description) VALUES ('ds-er2-powerboard', 'dev-er2', 'ER2', 'ds-er2-controls', 'controls', 'powerboard', 'ER2 Powerboard', 'assembly', 'controls', 'active', 'ER2 powerboard assembly') ON CONFLICT (device_key, scope_key) DO NOTHING",
	"INSERT INTO device_scopes (id, device_id, device_key, parent_scope_id, system_key, scope_key, scope_name, scope_type, owner_department_key, status, description) VALUES ('ds-mk4-cabinet', 'dev-mk4', 'MK4', 'ds-mk4-mechanical', 'mechanical', 'cabinet', 'MK4 Cabinet', 'assembly', 'mechanical', 'active', 'MK4 cabinet assembly') ON CONFLICT (device_key, scope_key) DO NOTHING",
	"INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description, default_supplier_id, note) VALUES ('item-er2', 'omron', 'relay', 'ER2', 'Control relay', 'sup-misumi', 'Standard relay') ON CONFLICT DO NOTHING",
	"INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description, default_supplier_id, note) VALUES ('item-mk44', 'phoenix', 'terminal', 'MK-44', 'Terminal block 4P', 'sup-thorlabs', 'Common terminal block') ON CONFLICT DO NOTHING",
	"INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description, default_supplier_id, note) VALUES ('item-cn88', 'molex', 'connector', 'CN-88', 'I/O connector housing', 'sup-misumi', 'Used in cabinet harness') ON CONFLICT DO NOTHING",
}

func ResetDatabase(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reset tx: %w", err)
	}
	defer tx.Rollback()

	for _, table := range resetTables {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}

	if err := SeedMasterData(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reset tx: %w", err)
	}
	return nil
}

func SeedMasterData(ctx context.Context, execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}) error {
	for _, stmt := range masterSeedStatements {
		if _, err := execer.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("seed master data: %w", err)
		}
	}
	return nil
}
