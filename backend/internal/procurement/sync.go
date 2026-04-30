package procurement

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	DBSchemaID               string
	KeyID                    string
	ResponseType             string
	Payload                  map[string]any
}

type WebhookVerificationResult struct {
	Accepted bool
	Event    WebhookEvent
}

type ReconciliationInput struct {
	BatchID                  string
	ExternalRequestReference string
	DBSchemaID               string
	KeyID                    string
	ResponseType             string
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

type RakurakuSyncAdapterConfig struct {
	WebhookSecret     string
	ViewAPIURL        string
	APIToken          string
	ResponseType      string
	DefaultDBSchemaID string
	RequestTimeout    time.Duration
}

type RakurakuSyncAdapter struct {
	webhookSecret     string
	viewAPIURL        string
	apiToken          string
	responseType      string
	defaultDBSchemaID string
	client            *http.Client
}

func NewRakurakuSyncAdapter(cfg RakurakuSyncAdapterConfig) *RakurakuSyncAdapter {
	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &RakurakuSyncAdapter{
		webhookSecret:     strings.TrimSpace(cfg.WebhookSecret),
		viewAPIURL:        strings.TrimSpace(cfg.ViewAPIURL),
		apiToken:          strings.TrimSpace(cfg.APIToken),
		responseType:      defaultString(strings.TrimSpace(cfg.ResponseType), "1"),
		defaultDBSchemaID: strings.TrimSpace(cfg.DefaultDBSchemaID),
		client:            &http.Client{Timeout: timeout},
	}
}

func (a *RakurakuSyncAdapter) Name() string {
	return "rakuraku_sync_adapter"
}

func (a *RakurakuSyncAdapter) VerifyWebhook(_ context.Context, input WebhookVerificationInput) (WebhookVerificationResult, error) {
	if a.webhookSecret != "" && input.Headers.Get("X-Webhook-Secret") != a.webhookSecret {
		return WebhookVerificationResult{}, fmt.Errorf("webhook secret verification failed")
	}

	body, err := unwrapPubSubMessage(input.Body)
	if err != nil {
		return WebhookVerificationResult{}, err
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return WebhookVerificationResult{}, fmt.Errorf("invalid webhook payload: %w", err)
	}

	keyID := firstNonEmpty(stringValue(payload["keyId"]), stringValue(payload["keyID"]))
	dbSchemaID := stringValue(payload["dbSchemaId"])
	responseType := defaultString(stringValue(payload["responseType"]), a.responseType)
	externalReference := firstNonEmpty(stringValue(payload["externalRequestReference"]), keyID)
	eventType := stringValue(payload["eventType"])
	if eventType == "" && dbSchemaID != "" && keyID != "" {
		eventType = "procurement.status_changed"
	}
	if eventType == "" {
		eventType = "master.projects_changed"
	}

	return WebhookVerificationResult{
		Accepted: true,
		Event: WebhookEvent{
			EventType:                eventType,
			ExternalRequestReference: externalReference,
			ProjectKey:               stringValue(payload["projectKey"]),
			NormalizedStatus:         normalizeStatusName(stringValue(payload["normalizedStatus"])),
			RawStatus:                stringValue(payload["rawStatus"]),
			DBSchemaID:               dbSchemaID,
			KeyID:                    keyID,
			ResponseType:             responseType,
			Payload:                  payload,
		},
	}, nil
}

func (a *RakurakuSyncAdapter) FetchProcurementReconciliation(ctx context.Context, input ReconciliationInput) (ReconciliationResult, error) {
	if a.viewAPIURL == "" {
		return ReconciliationResult{}, fmt.Errorf("rakuraku view api url is not configured")
	}
	keyID := firstNonEmpty(input.KeyID, input.ExternalRequestReference)
	if keyID == "" {
		return ReconciliationResult{}, fmt.Errorf("keyId or external request reference is required")
	}
	dbSchemaID := firstNonEmpty(input.DBSchemaID, a.defaultDBSchemaID)
	if dbSchemaID == "" {
		return ReconciliationResult{}, fmt.Errorf("dbSchemaId is required for rakuraku reconciliation")
	}

	requestPayload := map[string]string{
		"dbSchemaId":   dbSchemaID,
		"keyId":        keyID,
		"responseType": defaultString(strings.TrimSpace(input.ResponseType), a.responseType),
	}
	body, _ := json.Marshal(requestPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.viewAPIURL, bytes.NewReader(body))
	if err != nil {
		return ReconciliationResult{}, fmt.Errorf("build rakuraku request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if a.apiToken != "" {
		req.Header.Set("X-HD-apitoken", a.apiToken)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return ReconciliationResult{}, fmt.Errorf("call rakuraku view api: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReconciliationResult{}, fmt.Errorf("read rakuraku response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ReconciliationResult{}, fmt.Errorf("rakuraku view api returned %d: %s", resp.StatusCode, string(respBody))
	}

	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return ReconciliationResult{}, fmt.Errorf("decode rakuraku response: %w", err)
	}
	if status := strings.ToLower(stringValue(raw["status"])); status != "" && status != "success" {
		return ReconciliationResult{}, fmt.Errorf("rakuraku view api status is %s", status)
	}

	items, _ := raw["items"].(map[string]any)
	rawStatus := firstNonEmpty(stringValue(items["申請状態"]), input.EventRawStatus)
	normalizedStatus := normalizeRakurakuStatus(firstNonEmpty(input.EventNormalizedStatus, rawStatus, input.CurrentNormalizedStatus))
	progression := rakurakuQuantityProgression(normalizedStatus, input.QuantityProgression, items)
	observedAt := parseRakurakuAccessTime(stringValue(raw["accessTime"]))

	return ReconciliationResult{
		NormalizedStatus:         normalizedStatus,
		RawStatus:                rawStatus,
		ExternalRequestReference: keyID,
		QuantityProgression:      progression,
		ObservedAt:               observedAt,
		RawResponse:              raw,
		SyncSource:               a.Name(),
		Note:                     fmt.Sprintf("Reconciled via %s (%s dbSchemaId=%s keyId=%s)", a.Name(), defaultString(input.Trigger, "manual"), dbSchemaID, keyID),
	}, nil
}

func (a *RakurakuSyncAdapter) FetchProjectMaster(_ context.Context) (ProjectMasterFetchResult, error) {
	return ProjectMasterFetchResult{}, fmt.Errorf("rakuraku project master sync is not configured")
}

func (a *RakurakuSyncAdapter) FetchBudgetCategories(_ context.Context, projectKey string) (BudgetCategoryFetchResult, error) {
	return BudgetCategoryFetchResult{}, fmt.Errorf("rakuraku budget category sync is not configured for project %s", projectKey)
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
	body, err := unwrapPubSubMessage(input.Body)
	if err != nil {
		return WebhookVerificationResult{}, err
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return WebhookVerificationResult{}, fmt.Errorf("invalid webhook payload: %w", err)
	}

	keyID := firstNonEmpty(stringValue(payload["keyId"]), stringValue(payload["keyID"]))
	externalReference := firstNonEmpty(stringValue(payload["externalRequestReference"]), keyID)
	event := WebhookEvent{
		EventType:                stringValue(payload["eventType"]),
		ExternalRequestReference: externalReference,
		ProjectKey:               stringValue(payload["projectKey"]),
		NormalizedStatus:         normalizeStatusName(stringValue(payload["normalizedStatus"])),
		RawStatus:                stringValue(payload["rawStatus"]),
		DBSchemaID:               stringValue(payload["dbSchemaId"]),
		KeyID:                    keyID,
		ResponseType:             stringValue(payload["responseType"]),
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

func normalizeRakurakuStatus(value string) string {
	normalized := normalizeStatusName(value)
	switch normalized {
	case "draft", "submitted", "ordered", "partially_received", "received", "rejected":
		return normalized
	}

	switch strings.TrimSpace(value) {
	case "下書き", "未申請":
		return "draft"
	case "申請中", "承認待ち", "確認中":
		return "submitted"
	case "承認", "承認済", "発注", "発注済":
		return "ordered"
	case "一部入荷", "一部納品", "分納":
		return "partially_received"
	case "入荷済", "納品済", "検収済", "完了":
		return "received"
	case "却下", "否認", "差戻", "差し戻し", "取消", "キャンセル":
		return "rejected"
	default:
		if normalized != "" {
			return normalized
		}
		return "submitted"
	}
}

func rakurakuQuantityProgression(status string, current ProcurementQuantityProgression, items map[string]any) ProcurementQuantityProgression {
	next := current
	total := totalRakurakuDetailQuantity(items)
	if total > 0 {
		next.Requested = total
	}
	return nextQuantityProgression(status, next)
}

func totalRakurakuDetailQuantity(items map[string]any) int {
	if items == nil {
		return 0
	}
	details, ok := items["details"].([]any)
	if !ok {
		return 0
	}
	total := 0
	for _, item := range details {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		total += parseDisplayInt(stringValue(row["数量"]))
	}
	return total
}

func parseDisplayInt(value string) int {
	cleaned := strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	if cleaned == "" {
		return 0
	}
	var out int
	if _, err := fmt.Sscanf(cleaned, "%d", &out); err != nil {
		return 0
	}
	return out
}

func parseRakurakuAccessTime(value string) time.Time {
	if value != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05 -0700", value); err == nil {
			return parsed.UTC()
		}
	}
	return time.Now().UTC()
}

func unwrapPubSubMessage(body []byte) ([]byte, error) {
	var envelope struct {
		Message *struct {
			Data string `json:"data"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Message != nil && strings.TrimSpace(envelope.Message.Data) != "" {
		decoded, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
		if err != nil {
			return nil, fmt.Errorf("invalid pubsub message data: %w", err)
		}
		return decoded, nil
	}
	return body, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
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
