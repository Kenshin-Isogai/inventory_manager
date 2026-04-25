// Integration tests for transaction management: rollback on partial failure, undo operations.
//
//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	"backend/internal/inventory"
	"backend/internal/testutil"
)

// ============================================================
// Undo operations (transactional reversal)
// ============================================================

func TestUndoReceive_RollsBackBalance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// Receive 10 units
	entry, err := svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID:       "item-er2",
		LocationCode: "TOKYO-A1",
		Quantity:     10,
	})
	if err != nil {
		t.Fatalf("receive failed: %v", err)
	}

	// Verify balance is 10
	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	if onHand != 10 {
		t.Fatalf("on_hand before undo: want 10, got %d", onHand)
	}

	// Undo the receive
	_, err = svc.UndoInventoryEvent(ctx, entry.ID, inventory.InventoryUndoInput{
		ActorID: "test-user",
		Reason:  "test undo",
	})
	if err != nil {
		t.Fatalf("UndoInventoryEvent failed: %v", err)
	}

	// Balance should be back to 0
	onHand = testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	if onHand != 0 {
		t.Errorf("on_hand after undo: want 0, got %d", onHand)
	}
}

func TestUndoAdjust_RollsBackBalance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// Receive then adjust
	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 20})
	svc.AdjustInventory(ctx, inventory.InventoryAdjustInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", QuantityDelta: -5})

	onHandAfterAdjust := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	if onHandAfterAdjust != 15 {
		t.Fatalf("on_hand after adjust: want 15, got %d", onHandAfterAdjust)
	}

	// Find the adjust event
	var adjustEventID string
	err := db.QueryRowContext(ctx, `SELECT id FROM inventory_events WHERE item_id = 'item-er2' AND event_type = 'adjust' ORDER BY created_at DESC LIMIT 1`).Scan(&adjustEventID)
	if err != nil {
		t.Fatalf("find adjust event: %v", err)
	}

	// Undo the negative adjustment
	_, err = svc.UndoInventoryEvent(ctx, adjustEventID, inventory.InventoryUndoInput{
		ActorID: "test-user",
		Reason:  "undo adjustment",
	})
	if err != nil {
		t.Fatalf("UndoInventoryEvent failed: %v", err)
	}

	// Should be back to 20
	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	if onHand != 20 {
		t.Errorf("on_hand after undo adjust: want 20, got %d", onHand)
	}
}

func TestUndoSameEvent_Twice_Fails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	entry, _ := svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID:       "item-er2",
		LocationCode: "TOKYO-A1",
		Quantity:     10,
	})

	// First undo: should succeed
	_, err := svc.UndoInventoryEvent(ctx, entry.ID, inventory.InventoryUndoInput{ActorID: "test"})
	if err != nil {
		t.Fatalf("first undo failed: %v", err)
	}

	// Second undo of same event: should fail
	_, err = svc.UndoInventoryEvent(ctx, entry.ID, inventory.InventoryUndoInput{ActorID: "test"})
	if err == nil {
		t.Fatal("expected error on double undo, got nil")
	}
}

// ============================================================
// Import CSV: transactional atomicity
// ============================================================

func TestImportCSV_PartialFailure_RollsBack(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// CSV with row 1 valid, row 2 referencing nonexistent supplier
	csvData := "canonical_item_number,description,manufacturer,category,default_supplier_id\n" +
		"IMPORT-OK,Good Item,Omron,Relay,sup-misumi\n" +
		"IMPORT-BAD,Bad Item,Phoenix Contact,Terminal Block,sup-nonexistent\n"

	_, err := svc.ImportMasterCSV(ctx, "items", "test.csv", strings.NewReader(csvData))
	if err == nil {
		t.Fatal("expected import to fail due to nonexistent supplier")
	}

	// Neither item should have been created (transaction rolled back)
	count := testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM items WHERE canonical_item_number IN ('IMPORT-OK', 'IMPORT-BAD')`)
	if count != 0 {
		t.Errorf("expected 0 items after rollback, got %d", count)
	}
}

// ============================================================
// Multi-table transaction: reservation create writes both reservations + reservation_events
// ============================================================

func TestCreateReservation_WritesReservationAndEvent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	err := svc.CreateReservation(ctx, inventory.ReservationCreateInput{
		ItemID:        "item-er2",
		DeviceScopeID: "ds-er2-powerboard",
		Quantity:      3,
	})
	if err != nil {
		t.Fatalf("CreateReservation failed: %v", err)
	}

	// Both reservation and reservation_event should exist
	resCount := testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM reservations WHERE item_id = 'item-er2' AND quantity = 3`)
	if resCount != 1 {
		t.Errorf("expected 1 reservation, got %d", resCount)
	}

	eventCount := testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM reservation_events WHERE event_type = 'requested'`)
	if eventCount < 1 {
		t.Errorf("expected at least 1 reservation_event, got %d", eventCount)
	}

	// Audit event should also be recorded
	auditCount := testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM audit_events WHERE event_type = 'reservation.created'`)
	if auditCount < 1 {
		t.Errorf("expected at least 1 audit_event, got %d", auditCount)
	}
}

// ============================================================
// Move: source decrement + target increment are atomic
// ============================================================

func TestMoveInventory_AtomicDecAndInc(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 10})

	_, err := svc.MoveInventory(ctx, inventory.InventoryMoveInput{
		ItemID:           "item-er2",
		FromLocationCode: "TOKYO-A1",
		ToLocationCode:   "TOKYO-B2",
		Quantity:         7,
	})
	if err != nil {
		t.Fatalf("MoveInventory failed: %v", err)
	}

	srcOnHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	dstOnHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-B2'`)

	// Total should remain 10
	if srcOnHand+dstOnHand != 10 {
		t.Errorf("total on_hand should remain 10, got src=%d + dst=%d = %d", srcOnHand, dstOnHand, srcOnHand+dstOnHand)
	}
}
