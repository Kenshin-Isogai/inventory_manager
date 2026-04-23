package procurement

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestMockSyncAdapterReconciliationProgressesStatus(t *testing.T) {
	adapter := NewMockSyncAdapter("")

	result, err := adapter.FetchProcurementReconciliation(context.Background(), ReconciliationInput{
		BatchID:                  "batch-001",
		ExternalRequestReference: "LOCAL-SUBMIT-001",
		CurrentNormalizedStatus:  "submitted",
		QuantityProgression: ProcurementQuantityProgression{
			Requested: 8,
			Ordered:   0,
			Received:  0,
		},
		Trigger: "manual",
	})
	if err != nil {
		t.Fatalf("expected reconciliation result, got error: %v", err)
	}
	if result.NormalizedStatus != "ordered" {
		t.Fatalf("expected ordered status, got %s", result.NormalizedStatus)
	}
	if result.QuantityProgression.Ordered != 8 {
		t.Fatalf("expected ordered quantity to advance to 8, got %d", result.QuantityProgression.Ordered)
	}
}

func TestMockSyncAdapterVerifyWebhookRequiresSecretWhenConfigured(t *testing.T) {
	adapter := NewMockSyncAdapter("top-secret")

	_, err := adapter.VerifyWebhook(context.Background(), WebhookVerificationInput{
		Headers: http.Header{},
		Body:    []byte(`{"eventType":"master.projects_changed"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("expected secret verification failure, got %v", err)
	}
}
