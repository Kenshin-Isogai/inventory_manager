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
