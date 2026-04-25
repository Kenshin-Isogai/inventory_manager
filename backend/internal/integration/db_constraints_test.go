// Integration tests for database constraints: CHECK, UNIQUE, FK.
//
// Run with:
//   TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/inventory_manager_test?sslmode=disable \
//     go test -v -tags=integration ./internal/integration/...
//
//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	"backend/internal/testutil"
)

// ============================================================
// CHECK constraints
// ============================================================

func TestCheckConstraint_ReservationQuantityMustBePositive(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx := context.Background()

	// quantity = 0 should violate CHECK (quantity > 0)
	_, err := db.ExecContext(ctx, `
		INSERT INTO reservations (id, item_id, device_scope_id, quantity, status)
		VALUES ('res-zero', 'item-er2', 'ds-er2-powerboard', 0, 'requested')
	`)
	if err == nil {
		t.Fatal("expected CHECK violation for quantity=0, got nil")
	}
	if !strings.Contains(err.Error(), "check") && !strings.Contains(err.Error(), "violates") {
		t.Logf("error: %v", err)
	}

	// quantity = -1 should also violate
	_, err = db.ExecContext(ctx, `
		INSERT INTO reservations (id, item_id, device_scope_id, quantity, status)
		VALUES ('res-neg', 'item-er2', 'ds-er2-powerboard', -1, 'requested')
	`)
	if err == nil {
		t.Fatal("expected CHECK violation for quantity=-1, got nil")
	}

	// quantity = 1 should succeed
	_, err = db.ExecContext(ctx, `
		INSERT INTO reservations (id, item_id, device_scope_id, quantity, status)
		VALUES ('res-ok', 'item-er2', 'ds-er2-powerboard', 1, 'requested')
	`)
	if err != nil {
		t.Fatalf("expected success for quantity=1, got: %v", err)
	}
}

func TestCheckConstraint_SupplierAliasUnitsPerOrderMustBePositive(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx := context.Background()

	// units_per_order = 0 should violate CHECK (units_per_order > 0)
	_, err := db.ExecContext(ctx, `
		INSERT INTO supplier_item_aliases (id, item_id, supplier_id, supplier_item_number, units_per_order)
		VALUES ('alias-bad', 'item-er2', 'sup-misumi', 'BAD-ALIAS', 0)
	`)
	if err == nil {
		t.Fatal("expected CHECK violation for units_per_order=0, got nil")
	}

	// units_per_order = 1 should succeed
	_, err = db.ExecContext(ctx, `
		INSERT INTO supplier_item_aliases (id, item_id, supplier_id, supplier_item_number, units_per_order)
		VALUES ('alias-good', 'item-er2', 'sup-misumi', 'GOOD-ALIAS', 1)
	`)
	if err != nil {
		t.Fatalf("expected success for units_per_order=1, got: %v", err)
	}
}

// ============================================================
// UNIQUE constraints
// ============================================================

func TestUniqueConstraint_ItemCanonicalNumber(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx := context.Background()

	// item-er2 already uses canonical_item_number='ER2'; inserting another with same number should fail
	_, err := db.ExecContext(ctx, `
		INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description)
		VALUES ('item-dup', 'omron', 'relay', 'ER2', 'Duplicate relay')
	`)
	if err == nil {
		t.Fatal("expected UNIQUE violation for duplicate canonical_item_number, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "duplicate") && !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Logf("unexpected error message: %v", err)
	}
}

func TestUniqueConstraint_DeviceScopeKey(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx := context.Background()

	// (device_key, scope_key) = ('ER2', 'powerboard') already exists
	_, err := db.ExecContext(ctx, `
		INSERT INTO device_scopes (id, device_key, scope_key, description)
		VALUES ('ds-dup', 'ER2', 'powerboard', 'Duplicate scope')
	`)
	if err == nil {
		t.Fatal("expected UNIQUE violation for duplicate device_scope, got nil")
	}
}

func TestUniqueConstraint_UserEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx := context.Background()

	testutil.MustExec(t, db, `
		INSERT INTO app_users (id, email, display_name, status)
		VALUES ('00000000-0000-0000-0000-000000000001', 'alice@example.com', 'Alice', 'active')
	`)

	_, err := db.ExecContext(ctx, `
		INSERT INTO app_users (id, email, display_name, status)
		VALUES ('00000000-0000-0000-0000-000000000002', 'alice@example.com', 'Alice Clone', 'active')
	`)
	if err == nil {
		t.Fatal("expected UNIQUE violation for duplicate email, got nil")
	}
}

// ============================================================
// FK constraints
// ============================================================

func TestFKConstraint_ItemRequiresValidManufacturer(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `
		INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description)
		VALUES ('item-fk-test', 'nonexistent-mfg', 'relay', 'FK-TEST', 'Should fail')
	`)
	if err == nil {
		t.Fatal("expected FK violation for nonexistent manufacturer_key, got nil")
	}
}

func TestFKConstraint_ItemRequiresValidCategory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `
		INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description)
		VALUES ('item-fk-test2', 'omron', 'nonexistent-cat', 'FK-TEST-2', 'Should fail')
	`)
	if err == nil {
		t.Fatal("expected FK violation for nonexistent category_key, got nil")
	}
}

func TestFKConstraint_ReservationRequiresValidItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `
		INSERT INTO reservations (id, item_id, device_scope_id, quantity, status)
		VALUES ('res-fk', 'nonexistent-item', 'ds-er2-powerboard', 5, 'requested')
	`)
	if err == nil {
		t.Fatal("expected FK violation for nonexistent item_id, got nil")
	}
}

func TestFKConstraint_ReservationRequiresValidDeviceScope(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `
		INSERT INTO reservations (id, item_id, device_scope_id, quantity, status)
		VALUES ('res-fk2', 'item-er2', 'nonexistent-scope', 5, 'requested')
	`)
	if err == nil {
		t.Fatal("expected FK violation for nonexistent device_scope_id, got nil")
	}
}

func TestFKConstraint_CascadeDeleteItem(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Create an item + alias
	testutil.MustExec(t, db, `
		INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description)
		VALUES ('item-cascade', 'omron', 'relay', 'CASCADE-TEST', 'For cascade test')
	`)
	testutil.MustExec(t, db, `
		INSERT INTO supplier_item_aliases (id, item_id, supplier_id, supplier_item_number, units_per_order)
		VALUES ('alias-cascade', 'item-cascade', 'sup-misumi', 'CASCADE-ALIAS', 1)
	`)

	// Verify alias exists
	count := testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM supplier_item_aliases WHERE id = 'alias-cascade'`)
	if count != 1 {
		t.Fatalf("expected 1 alias, got %d", count)
	}

	// Delete item → alias should be cascade-deleted
	testutil.MustExec(t, db, `DELETE FROM items WHERE id = 'item-cascade'`)

	count = testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM supplier_item_aliases WHERE id = 'alias-cascade'`)
	if count != 0 {
		t.Fatalf("expected alias to be cascade-deleted, got %d", count)
	}
}

func TestFKConstraint_InventoryEventRequiresValidItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx := context.Background()
	_ = ctx

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO inventory_events (id, item_id, location_code, event_type, quantity_delta)
		VALUES ('evt-fk', 'nonexistent-item', 'TOKYO-A1', 'receive', 10)
	`)
	if err == nil {
		t.Fatal("expected FK violation for nonexistent item_id in inventory_events, got nil")
	}
}
