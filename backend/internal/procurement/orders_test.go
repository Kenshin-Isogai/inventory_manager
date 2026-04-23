package procurement

import (
	"context"
	"testing"
)

func TestCreateOrderValidatesInput(t *testing.T) {
	service := NewService(nil, nil, nil, nil)

	if _, err := service.CreateOrder(context.Background(), PurchaseOrderCreateInput{}); err == nil {
		t.Fatalf("expected missing batch validation error")
	}
	if _, err := service.CreateOrder(context.Background(), PurchaseOrderCreateInput{
		ProcurementBatchID: "batch-1",
	}); err == nil {
		t.Fatalf("expected missing lines validation error")
	}
	if _, err := service.CreateOrder(context.Background(), PurchaseOrderCreateInput{
		ProcurementBatchID: "batch-1",
		Lines: []PurchaseOrderLineInput{
			{OrderedQuantity: 1},
		},
	}); err == nil {
		t.Fatalf("expected missing procurement line validation error")
	}
}

func TestUpdateOrderValidatesInput(t *testing.T) {
	service := NewService(nil, nil, nil, nil)

	if _, err := service.UpdateOrder(context.Background(), "", PurchaseOrderUpdateInput{}); err == nil {
		t.Fatalf("expected missing order id validation error")
	}
	if _, err := service.UpdateOrder(context.Background(), "po-1", PurchaseOrderUpdateInput{}); err == nil {
		t.Fatalf("expected missing lines validation error")
	}
	if _, err := service.UpdateOrder(context.Background(), "po-1", PurchaseOrderUpdateInput{
		Lines: []PurchaseOrderLineInput{
			{ProcurementLineID: "pline-1", OrderedQuantity: 0},
		},
	}); err == nil {
		t.Fatalf("expected invalid ordered quantity validation error")
	}
}
