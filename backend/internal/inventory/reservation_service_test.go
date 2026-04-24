package inventory

import (
	"context"
	"testing"
)

func TestUpdateReservationValidatesInput(t *testing.T) {
	service := NewService(nil)

	if _, err := service.UpdateReservation(context.Background(), "", ReservationUpdateInput{}); err == nil {
		t.Fatalf("expected missing reservation id validation error")
	}
	if _, err := service.UpdateReservation(context.Background(), "res-1", ReservationUpdateInput{}); err == nil {
		t.Fatalf("expected missing reservation fields validation error")
	}
}

func TestUpsertDeviceScopeValidatesHierarchyShape(t *testing.T) {
	service := NewService(nil)

	if _, err := service.UpsertDeviceScope(context.Background(), DeviceScopeUpsertInput{
		DeviceKey:     "ER2",
		ScopeKey:      "powerboard",
		ScopeType:     "assembly",
		ParentScopeID: "",
	}); err == nil {
		t.Fatalf("expected non-system scope parent validation error")
	}

	if _, err := service.UpsertDeviceScope(context.Background(), DeviceScopeUpsertInput{
		DeviceKey:     "ER2",
		ScopeKey:      "optics",
		SystemKey:     "optics",
		ScopeType:     "system",
		ParentScopeID: "scope-parent",
	}); err == nil {
		t.Fatalf("expected system scope parent validation error")
	}

	if _, err := service.UpsertDeviceScope(context.Background(), DeviceScopeUpsertInput{
		DeviceKey: "ER2",
		ScopeKey:  "optics",
		SystemKey: "mechanical",
		ScopeType: "system",
	}); err == nil {
		t.Fatalf("expected system scope key coherence validation error")
	}
}
