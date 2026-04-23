package procurement

import (
	"context"
	"fmt"
	"time"
)

type Dispatcher interface {
	SubmitProcurementRequest(ctx context.Context, payload ProcurementSubmissionPayload) (DispatchResult, error)
	Name() string
}

type DispatchResult struct {
	ExternalRequestReference string
	RawStatus                string
	RawResponse              map[string]any
	EvidenceFileReferences   []string
}

type NormalizedDispatchError struct {
	Code      string
	Retryable bool
	Message   string
}

func (e NormalizedDispatchError) Error() string {
	return e.Message
}

type MockDispatcher struct{}

func NewMockDispatcher() *MockDispatcher {
	return &MockDispatcher{}
}

func (d *MockDispatcher) Name() string {
	return "mock_dispatcher"
}

func (d *MockDispatcher) SubmitProcurementRequest(_ context.Context, payload ProcurementSubmissionPayload) (DispatchResult, error) {
	if payload.BatchID == "" || payload.QuotationID == "" {
		return DispatchResult{}, NormalizedDispatchError{
			Code:      "invalid_payload",
			Retryable: false,
			Message:   "batchId and quotationId are required for dispatch",
		}
	}
	now := time.Now().UTC()
	externalRef := fmt.Sprintf("LOCAL-SUBMIT-%s-%03d", now.Format("20060102"), now.Nanosecond()%1000)
	return DispatchResult{
		ExternalRequestReference: externalRef,
		RawStatus:                "submitted_to_mock_adapter",
		RawResponse: map[string]any{
			"adapter":        d.Name(),
			"batchId":        payload.BatchID,
			"idempotencyKey": payload.IdempotencyKey,
			"acceptedAt":     now.Format(time.RFC3339),
		},
		EvidenceFileReferences: []string{fmt.Sprintf("local-evidence:%s", payload.QuotationID)},
	}, nil
}

func normalizeDispatchError(err error) NormalizedDispatchError {
	if err == nil {
		return NormalizedDispatchError{}
	}
	if normalized, ok := err.(NormalizedDispatchError); ok {
		return normalized
	}
	return NormalizedDispatchError{
		Code:      "dispatch_failed",
		Retryable: false,
		Message:   err.Error(),
	}
}
