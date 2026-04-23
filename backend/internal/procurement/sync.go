package procurement

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SyncAdapter interface {
	Name() string
	VerifyWebhook(ctx context.Context, input WebhookVerificationInput) (WebhookVerificationResult, error)
	FetchProcurementReconciliation(ctx context.Context, input ReconciliationInput) (ReconciliationResult, error)
	FetchProjectMaster(ctx context.Context) (ProjectMasterFetchResult, error)
	FetchBudgetCategories(ctx context.Context, projectKey string) (BudgetCategoryFetchResult, error)
}

type WebhookVerificationInput struct {
	Headers http.Header
	Body    []byte
}

type WebhookEvent struct {
	EventType                string
	ExternalRequestReference string
	ProjectKey               string
	NormalizedStatus         string
	RawStatus                string
	Payload                  map[string]any
}

type WebhookVerificationResult struct {
	Accepted bool
	Event    WebhookEvent
}

type ReconciliationInput struct {
	BatchID                  string
	ExternalRequestReference string
	CurrentNormalizedStatus  string
	QuantityProgression      ProcurementQuantityProgression
	Trigger                  string
	EventNormalizedStatus    string
	EventRawStatus           string
}

type ReconciliationResult struct {
	NormalizedStatus         string
	RawStatus                string
	ExternalRequestReference string
	QuantityProgression      ProcurementQuantityProgression
	ObservedAt               time.Time
	RawResponse              map[string]any
	Note                     string
	SyncSource               string
}

type ProjectMasterRow struct {
	ID     string
	Key    string
	Name   string
	Status string
}

type ProjectMasterFetchResult struct {
	Rows        []ProjectMasterRow
	SyncedAt    time.Time
	RawResponse map[string]any
	Source      string
}

type BudgetCategoryMasterRow struct {
	ID         string
	ProjectKey string
	Key        string
	Name       string
}

type BudgetCategoryFetchResult struct {
	ProjectKey  string
	Rows        []BudgetCategoryMasterRow
	SyncedAt    time.Time
	RawResponse map[string]any
	Source      string
}

type MockSyncAdapter struct {
	webhookSecret string
}

func NewMockSyncAdapter(webhookSecret string) *MockSyncAdapter {
	return &MockSyncAdapter{webhookSecret: strings.TrimSpace(webhookSecret)}
}

func (a *MockSyncAdapter) Name() string {
	return "mock_sync_adapter"
}

func (a *MockSyncAdapter) VerifyWebhook(_ context.Context, input WebhookVerificationInput) (WebhookVerificationResult, error) {
	if a.webhookSecret != "" && input.Headers.Get("X-Webhook-Secret") != a.webhookSecret {
		return WebhookVerificationResult{}, fmt.Errorf("webhook secret verification failed")
	}

	var payload map[string]any
	if err := json.Unmarshal(input.Body, &payload); err != nil {
		return WebhookVerificationResult{}, fmt.Errorf("invalid webhook payload: %w", err)
	}

	event := WebhookEvent{
		EventType:                stringValue(payload["eventType"]),
		ExternalRequestReference: stringValue(payload["externalRequestReference"]),
		ProjectKey:               stringValue(payload["projectKey"]),
		NormalizedStatus:         normalizeStatusName(stringValue(payload["normalizedStatus"])),
		RawStatus:                stringValue(payload["rawStatus"]),
		Payload:                  payload,
	}
	if event.EventType == "" {
		switch {
		case event.ExternalRequestReference != "":
			event.EventType = "procurement.status_changed"
		case event.ProjectKey != "":
			event.EventType = "master.budget_categories_changed"
		default:
			event.EventType = "master.projects_changed"
		}
	}

	return WebhookVerificationResult{
		Accepted: true,
		Event:    event,
	}, nil
}

func (a *MockSyncAdapter) FetchProcurementReconciliation(_ context.Context, input ReconciliationInput) (ReconciliationResult, error) {
	if input.ExternalRequestReference == "" {
		return ReconciliationResult{}, fmt.Errorf("external request reference is required")
	}

	targetStatus := normalizeStatusName(input.EventNormalizedStatus)
	if targetStatus == "" {
		targetStatus = nextNormalizedStatus(input.CurrentNormalizedStatus)
	}
	progression := nextQuantityProgression(targetStatus, input.QuantityProgression)
	rawStatus := input.EventRawStatus
	if rawStatus == "" {
		rawStatus = defaultRawStatus(targetStatus)
	}

	observedAt := time.Now().UTC()
	return ReconciliationResult{
		NormalizedStatus:         targetStatus,
		RawStatus:                rawStatus,
		ExternalRequestReference: input.ExternalRequestReference,
		QuantityProgression:      progression,
		ObservedAt:               observedAt,
		SyncSource:               a.Name(),
		Note:                     fmt.Sprintf("Reconciled via %s (%s)", a.Name(), defaultString(input.Trigger, "manual")),
		RawResponse: map[string]any{
			"adapter":                  a.Name(),
			"batchId":                  input.BatchID,
			"trigger":                  defaultString(input.Trigger, "manual"),
			"normalizedStatus":         targetStatus,
			"rawStatus":                rawStatus,
			"externalRequestReference": input.ExternalRequestReference,
			"observedAt":               observedAt.Format(time.RFC3339),
		},
	}, nil
}

func (a *MockSyncAdapter) FetchProjectMaster(_ context.Context) (ProjectMasterFetchResult, error) {
	now := time.Now().UTC()
	rows := []ProjectMasterRow{
		{Key: "ER2-UPGRADE", Name: "ER2 Production Upgrade", Status: "active"},
		{Key: "MK4-REFRESH", Name: "MK4 Cabinet Refresh", Status: "active"},
		{Key: "LAB-AUTOMATION", Name: "Lab Automation Expansion", Status: "active"},
	}
	return ProjectMasterFetchResult{
		Rows:     rows,
		SyncedAt: now,
		Source:   a.Name(),
		RawResponse: map[string]any{
			"adapter": a.Name(),
			"count":   len(rows),
		},
	}, nil
}

func (a *MockSyncAdapter) FetchBudgetCategories(_ context.Context, projectKey string) (BudgetCategoryFetchResult, error) {
	now := time.Now().UTC()
	key := strings.TrimSpace(projectKey)
	if key == "" {
		return BudgetCategoryFetchResult{}, fmt.Errorf("project key is required")
	}

	rows := []BudgetCategoryMasterRow{
		{ProjectKey: key, Key: "material", Name: "Material Cost"},
		{ProjectKey: key, Key: "maintenance", Name: "Maintenance"},
	}
	if key == "LAB-AUTOMATION" {
		rows = append(rows, BudgetCategoryMasterRow{ProjectKey: key, Key: "capex", Name: "Capital Expenditure"})
	}

	return BudgetCategoryFetchResult{
		ProjectKey: key,
		Rows:       rows,
		SyncedAt:   now,
		Source:     a.Name(),
		RawResponse: map[string]any{
			"adapter":    a.Name(),
			"projectKey": key,
			"count":      len(rows),
		},
	}, nil
}

func nextNormalizedStatus(current string) string {
	switch normalizeStatusName(current) {
	case "draft":
		return "submitted"
	case "submitted":
		return "ordered"
	case "ordered":
		return "partially_received"
	case "partially_received":
		return "received"
	case "received":
		return "received"
	default:
		return "ordered"
	}
}

func nextQuantityProgression(status string, current ProcurementQuantityProgression) ProcurementQuantityProgression {
	next := current
	switch normalizeStatusName(status) {
	case "draft":
		next.Ordered = 0
		next.Received = 0
	case "submitted":
	case "ordered":
		next.Ordered = maxInt(next.Ordered, next.Requested)
		next.Received = 0
	case "partially_received":
		next.Ordered = maxInt(next.Ordered, next.Requested)
		next.Received = maxInt(next.Received, minInt(next.Requested, maxInt(1, next.Requested/2)))
	case "received":
		next.Ordered = maxInt(next.Ordered, next.Requested)
		next.Received = next.Requested
	}
	return next
}

func defaultRawStatus(normalized string) string {
	switch normalizeStatusName(normalized) {
	case "submitted":
		return "submitted_to_external_flow"
	case "ordered":
		return "external_order_confirmed"
	case "partially_received":
		return "external_partial_receipt"
	case "received":
		return "external_receipt_completed"
	default:
		return defaultString(normalized, "draft")
	}
}

func normalizeStatusName(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "draft", "submitted", "ordered", "partially_received", "received", "rejected":
		return normalized
	default:
		return normalized
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
