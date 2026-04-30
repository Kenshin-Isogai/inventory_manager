package procurement

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"backend/internal/platform/storage"
)

type Service struct {
	repo        *Repository
	store       storage.Store
	dispatcher  Dispatcher
	syncAdapter SyncAdapter
}

func NewService(repo *Repository, store storage.Store, dispatcher Dispatcher, syncAdapter SyncAdapter) *Service {
	return &Service{repo: repo, store: store, dispatcher: dispatcher, syncAdapter: syncAdapter}
}

func (s *Service) Projects(ctx context.Context) ([]ProjectSummary, error) {
	return s.repo.Projects(ctx)
}

func (s *Service) BudgetCategories(ctx context.Context, projectID string) ([]BudgetCategorySummary, error) {
	return s.repo.BudgetCategories(ctx, projectID)
}

func (s *Service) Requests(ctx context.Context) (ProcurementRequestList, error) {
	return s.repo.Requests(ctx)
}

func (s *Service) RequestDetail(ctx context.Context, id string) (ProcurementRequestDetail, error) {
	if id == "" {
		return ProcurementRequestDetail{}, fmt.Errorf("request id is required")
	}
	return s.repo.RequestDetail(ctx, id)
}

func (s *Service) Orders(ctx context.Context) (PurchaseOrderList, error) {
	return s.repo.Orders(ctx)
}

func (s *Service) OrderDetail(ctx context.Context, id string) (PurchaseOrderDetail, error) {
	if id == "" {
		return PurchaseOrderDetail{}, fmt.Errorf("order id is required")
	}
	return s.repo.OrderDetail(ctx, id)
}

func (s *Service) CreateOrder(ctx context.Context, input PurchaseOrderCreateInput) (PurchaseOrderDetail, error) {
	if input.ProcurementBatchID == "" {
		return PurchaseOrderDetail{}, fmt.Errorf("procurementBatchId is required")
	}
	if len(input.Lines) == 0 {
		return PurchaseOrderDetail{}, fmt.Errorf("at least one purchase order line is required")
	}
	for _, line := range input.Lines {
		if line.ProcurementLineID == "" || line.OrderedQuantity <= 0 {
			return PurchaseOrderDetail{}, fmt.Errorf("procurementLineId and positive orderedQuantity are required")
		}
	}
	return s.repo.CreateOrder(ctx, input)
}

func (s *Service) UpdateOrder(ctx context.Context, id string, input PurchaseOrderUpdateInput) (PurchaseOrderDetail, error) {
	if id == "" {
		return PurchaseOrderDetail{}, fmt.Errorf("order id is required")
	}
	if len(input.Lines) == 0 {
		return PurchaseOrderDetail{}, fmt.Errorf("at least one purchase order line is required")
	}
	for _, line := range input.Lines {
		if line.ProcurementLineID == "" || line.OrderedQuantity <= 0 {
			return PurchaseOrderDetail{}, fmt.Errorf("procurementLineId and positive orderedQuantity are required")
		}
	}
	return s.repo.UpdateOrder(ctx, id, input)
}

func (s *Service) DeleteOrder(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("order id is required")
	}
	return s.repo.DeleteOrder(ctx, id)
}

func (s *Service) CreateRequest(ctx context.Context, input ProcurementRequestCreateInput) (string, error) {
	if input.Title == "" {
		return "", fmt.Errorf("title is required")
	}
	if len(input.Lines) == 0 {
		return "", fmt.Errorf("at least one procurement line is required")
	}
	for _, line := range input.Lines {
		if line.RequestedQuantity <= 0 {
			return "", fmt.Errorf("requestedQuantity must be positive")
		}
	}
	return s.repo.CreateRequest(ctx, input)
}

func (s *Service) UpdateRequest(ctx context.Context, id string, input ProcurementRequestUpdateInput) (ProcurementRequestDetail, error) {
	if id == "" {
		return ProcurementRequestDetail{}, fmt.Errorf("request id is required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return ProcurementRequestDetail{}, fmt.Errorf("title is required")
	}
	if len(input.Lines) == 0 {
		return ProcurementRequestDetail{}, fmt.Errorf("at least one procurement line is required")
	}
	for _, line := range input.Lines {
		if line.RequestedQuantity <= 0 {
			return ProcurementRequestDetail{}, fmt.Errorf("requestedQuantity must be positive")
		}
	}
	return s.repo.UpdateRequest(ctx, id, input)
}

func (s *Service) CreateDraftFromOCR(ctx context.Context, input OCRProcurementDraftCreateInput) (OCRProcurementDraftCreateResult, error) {
	if input.SourceOCRJobID == "" {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("sourceOcrJobId is required")
	}
	if input.SupplierID == "" {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("supplierId is required")
	}
	if input.QuotationNumber == "" {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("quotationNumber is required")
	}
	if input.IssueDate == "" {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("issueDate is required")
	}
	if len(input.Lines) == 0 {
		return OCRProcurementDraftCreateResult{}, fmt.Errorf("at least one OCR line is required")
	}
	if input.Title == "" {
		input.Title = fmt.Sprintf("OCR quotation %s", input.QuotationNumber)
	}
	for _, line := range input.Lines {
		if line.ItemID == "" {
			return OCRProcurementDraftCreateResult{}, fmt.Errorf("itemId is required for all OCR lines")
		}
		if line.Quantity <= 0 {
			return OCRProcurementDraftCreateResult{}, fmt.Errorf("quantity must be positive")
		}
	}
	return s.repo.CreateDraftFromOCR(ctx, input)
}

func (s *Service) SubmitRequest(ctx context.Context, id string) (ProcurementSubmitResult, error) {
	if id == "" {
		return ProcurementSubmitResult{}, fmt.Errorf("request id is required")
	}
	if s.dispatcher == nil {
		return ProcurementSubmitResult{}, fmt.Errorf("dispatcher is not configured")
	}

	payload, currentState, err := s.repo.SubmissionPayload(ctx, id)
	if err != nil {
		return ProcurementSubmitResult{}, err
	}
	if currentState.DispatchStatus == "submitted" && currentState.ExternalRequestReference != "" {
		return ProcurementSubmitResult{
			RequestID:                id,
			ExternalRequestReference: currentState.ExternalRequestReference,
			DispatchStatus:           currentState.DispatchStatus,
			ArtifactDeleteStatus:     currentState.ArtifactDeleteStatus,
		}, nil
	}

	if err := s.repo.StartDispatchAttempt(ctx, id, payload); err != nil {
		return ProcurementSubmitResult{}, err
	}

	result, err := s.dispatcher.SubmitProcurementRequest(ctx, payload)
	if err != nil {
		normalized := normalizeDispatchError(err)
		if recordErr := s.repo.RecordDispatchFailure(ctx, id, payload, normalized); recordErr != nil {
			return ProcurementSubmitResult{}, recordErr
		}
		return ProcurementSubmitResult{}, err
	}

	if err := s.repo.RecordDispatchSuccess(ctx, id, payload, result); err != nil {
		return ProcurementSubmitResult{}, err
	}

	artifactStatus := currentState.ArtifactDeleteStatus
	if payload.ArtifactPath != "" && s.store != nil {
		if err := s.store.Delete(ctx, payload.ArtifactPath); err != nil {
			if markErr := s.repo.MarkArtifactDeleteFailure(ctx, id, err.Error()); markErr != nil {
				return ProcurementSubmitResult{}, markErr
			}
			return ProcurementSubmitResult{}, fmt.Errorf("procurement dispatch succeeded but artifact cleanup failed: %w", err)
		}
		if err := s.repo.MarkArtifactDeleted(ctx, id); err != nil {
			return ProcurementSubmitResult{}, err
		}
		artifactStatus = "deleted"
	}

	return ProcurementSubmitResult{
		RequestID:                id,
		ExternalRequestReference: result.ExternalRequestReference,
		DispatchStatus:           "submitted",
		ArtifactDeleteStatus:     artifactStatus,
	}, nil
}

func (s *Service) ReconcileRequest(ctx context.Context, id string) (ProcurementReconcileResult, error) {
	if id == "" {
		return ProcurementReconcileResult{}, fmt.Errorf("request id is required")
	}
	if s.syncAdapter == nil {
		return ProcurementReconcileResult{}, fmt.Errorf("sync adapter is not configured")
	}

	current, err := s.repo.ReconciliationContext(ctx, id)
	if err != nil {
		return ProcurementReconcileResult{}, err
	}
	if current.ExternalRequestReference == "" {
		return ProcurementReconcileResult{}, fmt.Errorf("request must be submitted before reconciliation")
	}

	result, err := s.syncAdapter.FetchProcurementReconciliation(ctx, ReconciliationInput{
		BatchID:                  current.RequestID,
		ExternalRequestReference: current.ExternalRequestReference,
		CurrentNormalizedStatus:  current.NormalizedStatus,
		QuantityProgression:      current.QuantityProgression,
		Trigger:                  "manual",
	})
	if err != nil {
		_ = s.repo.RecordReconciliationFailure(ctx, id, err.Error(), s.syncAdapter.Name())
		return ProcurementReconcileResult{}, err
	}
	if err := s.repo.ApplyReconciliation(ctx, id, result); err != nil {
		return ProcurementReconcileResult{}, err
	}

	return ProcurementReconcileResult{
		RequestID:           id,
		NormalizedStatus:    result.NormalizedStatus,
		RawStatus:           result.RawStatus,
		QuantityProgression: encodeQuantityProgression(result.QuantityProgression),
		LastReconciledAt:    result.ObservedAt.UTC().Format(time.RFC3339),
		SyncSource:          result.SyncSource,
	}, nil
}

func (s *Service) RefreshProjects(ctx context.Context, triggeredBy string) (MasterSyncResult, error) {
	if s.syncAdapter == nil {
		return MasterSyncResult{}, fmt.Errorf("sync adapter is not configured")
	}
	result, err := s.syncAdapter.FetchProjectMaster(ctx)
	if err != nil {
		return MasterSyncResult{}, err
	}
	return s.repo.UpsertProjectMaster(ctx, result.Rows, defaultString(result.Source, s.syncAdapter.Name()), defaultString(triggeredBy, "manual"))
}

func (s *Service) RefreshBudgetCategories(ctx context.Context, projectID, triggeredBy string) (MasterSyncResult, error) {
	if s.syncAdapter == nil {
		return MasterSyncResult{}, fmt.Errorf("sync adapter is not configured")
	}
	if projectID == "" {
		projects, err := s.repo.Projects(ctx)
		if err != nil {
			return MasterSyncResult{}, err
		}
		total := 0
		var syncedAt string
		for _, project := range projects {
			result, err := s.refreshBudgetCategoriesByProjectKey(ctx, project.Key, triggeredBy)
			if err != nil {
				return MasterSyncResult{}, err
			}
			total += result.RowCount
			if result.SyncedAt > syncedAt {
				syncedAt = result.SyncedAt
			}
		}
		return MasterSyncResult{
			SyncType:    "budget_categories",
			Status:      "completed",
			RowCount:    total,
			Source:      s.syncAdapter.Name(),
			TriggeredBy: defaultString(triggeredBy, "manual"),
			SyncedAt:    syncedAt,
		}, nil
	}

	project, err := s.repo.ProjectByID(ctx, projectID)
	if err != nil {
		return MasterSyncResult{}, err
	}
	return s.refreshBudgetCategoriesByProjectKey(ctx, project.Key, triggeredBy)
}

func (s *Service) MasterSyncRuns(ctx context.Context) ([]MasterSyncRunEntry, error) {
	return s.repo.MasterSyncRuns(ctx)
}

func (s *Service) WebhookEvents(ctx context.Context) ([]WebhookEventEntry, error) {
	return s.repo.WebhookEvents(ctx)
}

func (s *Service) HandleWebhook(ctx context.Context, headers map[string][]string, body []byte) (WebhookProcessResult, error) {
	if s.syncAdapter == nil {
		return WebhookProcessResult{}, fmt.Errorf("sync adapter is not configured")
	}

	result, err := s.syncAdapter.VerifyWebhook(ctx, WebhookVerificationInput{
		Headers: http.Header(headers),
		Body:    body,
	})
	if err != nil {
		return WebhookProcessResult{}, err
	}
	if !result.Accepted {
		return WebhookProcessResult{}, fmt.Errorf("webhook rejected")
	}

	eventID, err := s.repo.RecordWebhookReceived(ctx, result.Event)
	if err != nil {
		return WebhookProcessResult{}, err
	}

	processResult, processErr := s.processWebhookEvent(ctx, result.Event)
	if processErr != nil {
		_ = s.repo.MarkWebhookFailed(ctx, eventID, processErr.Error())
		return WebhookProcessResult{}, processErr
	}
	if err := s.repo.MarkWebhookProcessed(ctx, eventID); err != nil {
		return WebhookProcessResult{}, err
	}
	return processResult, nil
}

func (s *Service) processWebhookEvent(ctx context.Context, event WebhookEvent) (WebhookProcessResult, error) {
	switch event.EventType {
	case "procurement.status_changed":
		requestID, err := s.repo.FindRequestIDByExternalReference(ctx, event.ExternalRequestReference)
		if err != nil {
			return WebhookProcessResult{}, err
		}
		current, err := s.repo.ReconciliationContext(ctx, requestID)
		if err != nil {
			return WebhookProcessResult{}, err
		}
		result, err := s.syncAdapter.FetchProcurementReconciliation(ctx, ReconciliationInput{
			BatchID:                  requestID,
			ExternalRequestReference: current.ExternalRequestReference,
			DBSchemaID:               event.DBSchemaID,
			KeyID:                    event.KeyID,
			ResponseType:             event.ResponseType,
			CurrentNormalizedStatus:  current.NormalizedStatus,
			QuantityProgression:      current.QuantityProgression,
			Trigger:                  "webhook",
			EventNormalizedStatus:    event.NormalizedStatus,
			EventRawStatus:           event.RawStatus,
		})
		if err != nil {
			_ = s.repo.RecordReconciliationFailure(ctx, requestID, err.Error(), s.syncAdapter.Name())
			return WebhookProcessResult{}, err
		}
		if err := s.repo.ApplyReconciliation(ctx, requestID, result); err != nil {
			return WebhookProcessResult{}, err
		}
		return WebhookProcessResult{
			EventType: event.EventType,
			Status:    "processed",
			RequestID: requestID,
			SyncedAt:  result.ObservedAt.UTC().Format(time.RFC3339),
		}, nil
	case "master.projects_changed":
		result, err := s.RefreshProjects(ctx, "webhook")
		if err != nil {
			return WebhookProcessResult{}, err
		}
		return WebhookProcessResult{
			EventType: event.EventType,
			Status:    result.Status,
			SyncedAt:  result.SyncedAt,
		}, nil
	case "master.budget_categories_changed":
		result, err := s.refreshBudgetCategoriesByProjectKey(ctx, event.ProjectKey, "webhook")
		if err != nil {
			return WebhookProcessResult{}, err
		}
		return WebhookProcessResult{
			EventType:  event.EventType,
			Status:     result.Status,
			ProjectKey: result.ProjectKey,
			SyncedAt:   result.SyncedAt,
		}, nil
	default:
		return WebhookProcessResult{}, fmt.Errorf("unsupported webhook event type: %s", event.EventType)
	}
}

func (s *Service) refreshBudgetCategoriesByProjectKey(ctx context.Context, projectKey, triggeredBy string) (MasterSyncResult, error) {
	if strings.TrimSpace(projectKey) == "" {
		return s.RefreshBudgetCategories(ctx, "", triggeredBy)
	}
	result, err := s.syncAdapter.FetchBudgetCategories(ctx, projectKey)
	if err != nil {
		return MasterSyncResult{}, err
	}
	return s.repo.UpsertBudgetCategories(ctx, result.ProjectKey, result.Rows, defaultString(result.Source, s.syncAdapter.Name()), defaultString(triggeredBy, "manual"))
}
