package inventory

import "testing"

func TestSnapshotSignatureChangesWithSnapshotContent(t *testing.T) {
	base := []InventorySnapshotRow{
		{
			ItemID:               "item-1",
			ItemNumber:           "ER2",
			NetAvailableQuantity: 5,
			ScopeSummaries: []InventorySnapshotScopeSummary{
				{Device: "ER2", Scope: "powerboard", RequirementQuantity: 4, ReservationQuantity: 2, RemainingDemand: 2},
			},
		},
	}

	sigA := snapshotSignature(base)
	sigB := snapshotSignature(base)
	if sigA == "" {
		t.Fatalf("expected non-empty signature")
	}
	if sigA != sigB {
		t.Fatalf("expected deterministic signature, got %q and %q", sigA, sigB)
	}

	changed := []InventorySnapshotRow{
		{
			ItemID:               "item-1",
			ItemNumber:           "ER2",
			NetAvailableQuantity: 4,
			ScopeSummaries: []InventorySnapshotScopeSummary{
				{Device: "ER2", Scope: "powerboard", RequirementQuantity: 4, ReservationQuantity: 2, RemainingDemand: 2},
			},
		},
	}
	if sigA == snapshotSignature(changed) {
		t.Fatalf("expected signature to change when snapshot content changes")
	}
}
