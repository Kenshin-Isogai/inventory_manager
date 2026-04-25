// Integration tests for inventory calculation logic: on_hand, reserved, available quantities.
//
//go:build integration

package integration

import (
	"context"
	"testing"

	"backend/internal/inventory"
	"backend/internal/testutil"
)

// ============================================================
// Inventory receive / adjust / move and balance verification
// ============================================================

func TestInventoryReceive_IncreasesOnHandAndAvailable(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// Receive 10 units of item-er2 at TOKYO-A1
	_, err := svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID:       "item-er2",
		LocationCode: "TOKYO-A1",
		Quantity:     10,
		Note:         "Initial receive",
	})
	if err != nil {
		t.Fatalf("ReceiveInventory failed: %v", err)
	}

	// Verify balance
	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	available := testutil.MustQueryInt(t, db, `SELECT available_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	reserved := testutil.MustQueryInt(t, db, `SELECT reserved_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)

	if onHand != 10 {
		t.Errorf("on_hand_quantity: want 10, got %d", onHand)
	}
	if available != 10 {
		t.Errorf("available_quantity: want 10, got %d", available)
	}
	if reserved != 0 {
		t.Errorf("reserved_quantity: want 0, got %d", reserved)
	}
}

func TestInventoryReceive_MultipleReceivesAccumulate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
			ItemID:       "item-er2",
			LocationCode: "TOKYO-A1",
			Quantity:     5,
		}); err != nil {
			t.Fatalf("receive %d failed: %v", i, err)
		}
	}

	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	if onHand != 15 {
		t.Errorf("on_hand_quantity after 3 receives of 5: want 15, got %d", onHand)
	}
}

func TestInventoryAdjust_PositiveDelta(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// First receive to establish balance
	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 10})

	// Positive adjustment
	err := svc.AdjustInventory(ctx, inventory.InventoryAdjustInput{
		ItemID:        "item-er2",
		LocationCode:  "TOKYO-A1",
		QuantityDelta: 5,
	})
	if err != nil {
		t.Fatalf("AdjustInventory(+5) failed: %v", err)
	}

	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	if onHand != 15 {
		t.Errorf("on_hand after +5 adjust: want 15, got %d", onHand)
	}
}

func TestInventoryAdjust_NegativeDelta(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 10})

	err := svc.AdjustInventory(ctx, inventory.InventoryAdjustInput{
		ItemID:        "item-er2",
		LocationCode:  "TOKYO-A1",
		QuantityDelta: -3,
	})
	if err != nil {
		t.Fatalf("AdjustInventory(-3) failed: %v", err)
	}

	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	if onHand != 7 {
		t.Errorf("on_hand after -3 adjust: want 7, got %d", onHand)
	}
}

func TestInventoryAdjust_NegativeDeltaExceedingOnHand_Fails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 5})

	err := svc.AdjustInventory(ctx, inventory.InventoryAdjustInput{
		ItemID:        "item-er2",
		LocationCode:  "TOKYO-A1",
		QuantityDelta: -10,
	})
	if err == nil {
		t.Fatal("expected error when adjusting below 0, got nil")
	}

	// on_hand should remain unchanged
	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	if onHand != 5 {
		t.Errorf("on_hand should remain 5 after failed adjust, got %d", onHand)
	}
}

func TestInventoryMove_DecreasesSourceIncreasesTarget(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// Seed source
	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 20})

	_, err := svc.MoveInventory(ctx, inventory.InventoryMoveInput{
		ItemID:           "item-er2",
		FromLocationCode: "TOKYO-A1",
		ToLocationCode:   "TOKYO-B2",
		Quantity:         8,
	})
	if err != nil {
		t.Fatalf("MoveInventory failed: %v", err)
	}

	srcOnHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	dstOnHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-B2'`)

	if srcOnHand != 12 {
		t.Errorf("source on_hand: want 12, got %d", srcOnHand)
	}
	if dstOnHand != 8 {
		t.Errorf("destination on_hand: want 8, got %d", dstOnHand)
	}
}

func TestInventoryMove_InsufficientStock_Fails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 5})

	_, err := svc.MoveInventory(ctx, inventory.InventoryMoveInput{
		ItemID:           "item-er2",
		FromLocationCode: "TOKYO-A1",
		ToLocationCode:   "TOKYO-B2",
		Quantity:         10,
	})
	if err == nil {
		t.Fatal("expected error moving more than available, got nil")
	}
}

// ============================================================
// Allocation flow: reserve → allocate → verify reserved_quantity
// ============================================================

func TestAllocation_ReservedQuantityReducesAvailable(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// Receive 20 units
	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 20})

	// Create reservation
	err := svc.CreateReservation(ctx, inventory.ReservationCreateInput{
		ItemID:        "item-er2",
		DeviceScopeID: "ds-er2-powerboard",
		Quantity:      5,
	})
	if err != nil {
		t.Fatalf("CreateReservation failed: %v", err)
	}

	// Find the reservation ID
	var resID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM reservations ORDER BY created_at DESC LIMIT 1`).Scan(&resID); err != nil {
		t.Fatalf("get reservation id: %v", err)
	}

	// Allocate from TOKYO-A1
	_, err = svc.AllocateReservation(ctx, resID, inventory.ReservationActionInput{
		LocationCode: "TOKYO-A1",
		Quantity:     5,
	})
	if err != nil {
		t.Fatalf("AllocateReservation failed: %v", err)
	}

	// Verify balance: reserved should be 5, available should be 15
	reserved := testutil.MustQueryInt(t, db, `SELECT reserved_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	available := testutil.MustQueryInt(t, db, `SELECT available_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)

	if reserved != 5 {
		t.Errorf("reserved: want 5, got %d", reserved)
	}
	if available != 15 {
		t.Errorf("available: want 15, got %d", available)
	}
	if onHand != 20 {
		t.Errorf("on_hand should remain 20, got %d", onHand)
	}
}

func TestAllocation_ExceedingAvailable_Fails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 5})

	err := svc.CreateReservation(ctx, inventory.ReservationCreateInput{
		ItemID:        "item-er2",
		DeviceScopeID: "ds-er2-powerboard",
		Quantity:      10,
	})
	if err != nil {
		t.Fatalf("CreateReservation failed: %v", err)
	}

	var resID string
	db.QueryRowContext(ctx, `SELECT id FROM reservations ORDER BY created_at DESC LIMIT 1`).Scan(&resID)

	_, err = svc.AllocateReservation(ctx, resID, inventory.ReservationActionInput{
		LocationCode: "TOKYO-A1",
		Quantity:     10,
	})
	if err == nil {
		t.Fatal("expected error allocating more than available, got nil")
	}
}

// ============================================================
// Shortage detection logic
// ============================================================

func TestShortages_DetectedWhenReservedExceedsOnHand(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// Receive only 5 units
	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 5})

	// Reserve 10 (exceeds on-hand)
	svc.CreateReservation(ctx, inventory.ReservationCreateInput{
		ItemID:        "item-er2",
		DeviceScopeID: "ds-er2-powerboard",
		Quantity:      10,
	})

	shortages, err := svc.Shortages(ctx, "", "")
	if err != nil {
		t.Fatalf("Shortages query failed: %v", err)
	}

	if len(shortages.Rows) == 0 {
		t.Fatal("expected at least 1 shortage row, got 0")
	}

	found := false
	for _, row := range shortages.Rows {
		if row.ItemNumber == "ER2" {
			found = true
			if row.Quantity != 5 {
				t.Errorf("shortage quantity: want 5, got %d", row.Quantity)
			}
		}
	}
	if !found {
		t.Error("expected shortage for ER2 item, not found")
	}
}

func TestShortages_NoShortageWhenSufficientStock(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 100})

	svc.CreateReservation(ctx, inventory.ReservationCreateInput{
		ItemID:        "item-er2",
		DeviceScopeID: "ds-er2-powerboard",
		Quantity:      5,
	})

	shortages, err := svc.Shortages(ctx, "", "")
	if err != nil {
		t.Fatalf("Shortages query failed: %v", err)
	}

	for _, row := range shortages.Rows {
		if row.ItemNumber == "ER2" {
			t.Errorf("unexpected shortage for ER2 when stock is sufficient")
		}
	}
}

// ============================================================
// Validation: service-layer input checks
// ============================================================

func TestValidation_CreateReservation_MissingFields(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	tests := []struct {
		name  string
		input inventory.ReservationCreateInput
	}{
		{"missing itemId", inventory.ReservationCreateInput{DeviceScopeID: "ds-er2-powerboard", Quantity: 1}},
		{"missing deviceScopeId", inventory.ReservationCreateInput{ItemID: "item-er2", Quantity: 1}},
		{"zero quantity", inventory.ReservationCreateInput{ItemID: "item-er2", DeviceScopeID: "ds-er2-powerboard", Quantity: 0}},
		{"negative quantity", inventory.ReservationCreateInput{ItemID: "item-er2", DeviceScopeID: "ds-er2-powerboard", Quantity: -1}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.CreateReservation(ctx, tc.input)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestValidation_AdjustInventory_MissingFields(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	tests := []struct {
		name  string
		input inventory.InventoryAdjustInput
	}{
		{"missing itemId", inventory.InventoryAdjustInput{LocationCode: "TOKYO-A1", QuantityDelta: 1}},
		{"missing locationCode", inventory.InventoryAdjustInput{ItemID: "item-er2", QuantityDelta: 1}},
		{"zero delta", inventory.InventoryAdjustInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", QuantityDelta: 0}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.AdjustInventory(ctx, tc.input)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

// ============================================================
// Inventory overview: balance aggregation
// ============================================================

func TestInventoryOverview_AggregatesCorrectly(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// Receive into two locations
	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 10})
	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{ItemID: "item-er2", LocationCode: "TOKYO-B2", Quantity: 5})

	overview, err := svc.InventoryOverview(ctx)
	if err != nil {
		t.Fatalf("InventoryOverview failed: %v", err)
	}

	if len(overview.Balances) < 2 {
		t.Fatalf("expected at least 2 balance rows, got %d", len(overview.Balances))
	}

	totalOnHand := 0
	for _, b := range overview.Balances {
		if b.ItemNumber == "ER2" {
			totalOnHand += b.OnHandQuantity
		}
	}
	if totalOnHand != 15 {
		t.Errorf("total on_hand for ER2: want 15, got %d", totalOnHand)
	}
}
