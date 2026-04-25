// Integration tests for concurrency control:
// Verifies that SELECT FOR UPDATE prevents double-allocation and negative inventory.
//
//go:build integration

package integration

import (
	"context"
	"sync"
	"testing"

	"backend/internal/inventory"
	"backend/internal/testutil"
)

// TestConcurrentAllocation_LastOneItem verifies that when two goroutines
// simultaneously try to allocate the last available item, exactly one succeeds
// and the balance never goes negative.
func TestConcurrentAllocation_LastOneItem(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// Receive exactly 1 unit
	_, err := svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID:       "item-er2",
		LocationCode: "TOKYO-A1",
		Quantity:     1,
	})
	if err != nil {
		t.Fatalf("receive failed: %v", err)
	}

	// Create two reservations
	for i := 0; i < 2; i++ {
		if err := svc.CreateReservation(ctx, inventory.ReservationCreateInput{
			ItemID:        "item-er2",
			DeviceScopeID: "ds-er2-powerboard",
			Quantity:      1,
		}); err != nil {
			t.Fatalf("create reservation %d failed: %v", i, err)
		}
	}

	// Get both reservation IDs
	rows, err := db.QueryContext(ctx, `SELECT id FROM reservations ORDER BY created_at ASC`)
	if err != nil {
		t.Fatalf("query reservations: %v", err)
	}
	var resIDs []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		resIDs = append(resIDs, id)
	}
	rows.Close()
	if len(resIDs) < 2 {
		t.Fatalf("expected 2 reservations, got %d", len(resIDs))
	}

	// Concurrently try to allocate from both reservations
	var wg sync.WaitGroup
	errs := make([]error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = svc.AllocateReservation(ctx, resIDs[idx], inventory.ReservationActionInput{
				LocationCode: "TOKYO-A1",
				Quantity:     1,
			})
		}(i)
	}
	wg.Wait()

	// Exactly one should succeed, one should fail
	successCount := 0
	failCount := 0
	for _, e := range errs {
		if e == nil {
			successCount++
		} else {
			failCount++
		}
	}

	if successCount != 1 || failCount != 1 {
		t.Errorf("expected 1 success + 1 failure, got success=%d failure=%d (err0=%v, err1=%v)",
			successCount, failCount, errs[0], errs[1])
	}

	// Verify balance never went negative
	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	reserved := testutil.MustQueryInt(t, db, `SELECT reserved_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)
	available := testutil.MustQueryInt(t, db, `SELECT available_quantity FROM inventory_balances WHERE item_id = 'item-er2' AND location_code = 'TOKYO-A1'`)

	if onHand < 0 {
		t.Errorf("on_hand went negative: %d", onHand)
	}
	if reserved < 0 {
		t.Errorf("reserved went negative: %d", reserved)
	}
	if available < 0 {
		t.Errorf("available went negative: %d", available)
	}
	if reserved != 1 {
		t.Errorf("reserved: want 1, got %d", reserved)
	}
	if available != 0 {
		t.Errorf("available: want 0, got %d", available)
	}
}

// TestConcurrentAdjust_PreventNegativeOnHand verifies that concurrent negative
// adjustments can't race past the on-hand check.
func TestConcurrentAdjust_PreventNegativeOnHand(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// Receive 5 units
	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID:       "item-mk44",
		LocationCode: "TOKYO-B2",
		Quantity:     5,
	})

	// 10 goroutines each try to adjust -1
	const goroutines = 10
	var wg sync.WaitGroup
	errs := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = svc.AdjustInventory(ctx, inventory.InventoryAdjustInput{
				ItemID:        "item-mk44",
				LocationCode:  "TOKYO-B2",
				QuantityDelta: -1,
			})
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, e := range errs {
		if e == nil {
			successCount++
		}
	}

	// Exactly 5 should succeed (we had 5 in stock)
	if successCount != 5 {
		t.Errorf("expected exactly 5 successes, got %d", successCount)
	}

	// Balance should be exactly 0
	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-mk44' AND location_code = 'TOKYO-B2'`)
	if onHand != 0 {
		t.Errorf("on_hand should be 0, got %d", onHand)
	}
	if onHand < 0 {
		t.Errorf("CRITICAL: on_hand went negative: %d", onHand)
	}
}

// TestConcurrentMoves_PreventOverdraw verifies that concurrent moves from the same
// source location can't overdraw.
func TestConcurrentMoves_PreventOverdraw(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)
	ctx := context.Background()

	// Receive 3 units
	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID:       "item-cn88",
		LocationCode: "TOKYO-C1",
		Quantity:     3,
	})

	// 5 goroutines each try to move 1 unit out
	const goroutines = 5
	var wg sync.WaitGroup
	errs := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = svc.MoveInventory(ctx, inventory.InventoryMoveInput{
				ItemID:           "item-cn88",
				FromLocationCode: "TOKYO-C1",
				ToLocationCode:   "TOKYO-A1",
				Quantity:         1,
			})
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, e := range errs {
		if e == nil {
			successCount++
		}
	}

	if successCount != 3 {
		t.Errorf("expected exactly 3 successful moves, got %d", successCount)
	}

	srcOnHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-cn88' AND location_code = 'TOKYO-C1'`)
	if srcOnHand < 0 {
		t.Errorf("CRITICAL: source on_hand went negative: %d", srcOnHand)
	}
	if srcOnHand != 0 {
		t.Errorf("source on_hand: want 0, got %d", srcOnHand)
	}

	dstOnHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = 'item-cn88' AND location_code = 'TOKYO-A1'`)
	if dstOnHand != 3 {
		t.Errorf("destination on_hand: want 3, got %d", dstOnHand)
	}
}
