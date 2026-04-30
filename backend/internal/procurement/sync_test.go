package procurement

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestMockSyncAdapterAcceptsPubSubWrappedRakurakuWebhook(t *testing.T) {
	adapter := NewMockSyncAdapter("")
	data := base64.StdEncoding.EncodeToString([]byte(`{"dbSchemaId":"101251","keyId":"00012","responseType":"1"}`))

	result, err := adapter.VerifyWebhook(context.Background(), WebhookVerificationInput{
		Headers: http.Header{},
		Body:    []byte(`{"message":{"data":"` + data + `"},"subscription":"projects/test/subscriptions/rakuraku"}`),
	})
	if err != nil {
		t.Fatalf("expected pubsub webhook to verify, got error: %v", err)
	}
	if result.Event.EventType != "procurement.status_changed" {
		t.Fatalf("expected procurement status event, got %s", result.Event.EventType)
	}
	if result.Event.ExternalRequestReference != "00012" || result.Event.DBSchemaID != "101251" {
		t.Fatalf("unexpected event metadata: %+v", result.Event)
	}
}

func TestRakurakuSyncAdapterFetchesViewAPI(t *testing.T) {
	var gotToken string
	var gotPayload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-HD-apitoken")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"success",
			"code":"200",
			"items":{
				"購買依頼ID":"00012",
				"申請状態":"承認",
				"details":[
					{"数量":"2"},
					{"数量":"3"}
				]
			},
			"accessTime":"2026-04-29 20:10:55 +0900"
		}`))
	}))
	defer server.Close()

	adapter := NewRakurakuSyncAdapter(RakurakuSyncAdapterConfig{
		ViewAPIURL:   server.URL,
		APIToken:     "token-1",
		ResponseType: "1",
	})
	result, err := adapter.FetchProcurementReconciliation(context.Background(), ReconciliationInput{
		ExternalRequestReference: "00012",
		DBSchemaID:               "101251",
		QuantityProgression: ProcurementQuantityProgression{
			Requested: 4,
		},
		Trigger: "webhook",
	})
	if err != nil {
		t.Fatalf("expected reconciliation result, got error: %v", err)
	}
	if gotToken != "token-1" {
		t.Fatalf("expected api token header, got %q", gotToken)
	}
	if gotPayload["dbSchemaId"] != "101251" || gotPayload["keyId"] != "00012" || gotPayload["responseType"] != "1" {
		t.Fatalf("unexpected request payload: %+v", gotPayload)
	}
	if result.NormalizedStatus != "ordered" || result.RawStatus != "承認" {
		t.Fatalf("unexpected status: %+v", result)
	}
	if result.QuantityProgression.Requested != 5 || result.QuantityProgression.Ordered != 5 {
		t.Fatalf("unexpected quantity progression: %+v", result.QuantityProgression)
	}
}
