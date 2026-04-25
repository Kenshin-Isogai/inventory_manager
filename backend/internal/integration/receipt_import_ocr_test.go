//go:build integration

package integration

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"backend/internal/inventory"
	ocrpkg "backend/internal/ocr"
	procurement "backend/internal/procurement"
	"backend/internal/testutil"
)

type testArtifactStore struct {
	savedKey    string
	deletedPath string
}

func (s *testArtifactStore) Save(_ context.Context, key string, body io.Reader) (string, error) {
	if _, err := io.ReadAll(body); err != nil {
		return "", err
	}
	s.savedKey = key
	return "artifacts/" + key, nil
}

func (s *testArtifactStore) Delete(_ context.Context, path string) error {
	s.deletedPath = path
	return nil
}

type scriptedOCRProvider struct {
	responses []scriptedOCRResponse
	calls     int
}

type scriptedOCRResponse struct {
	doc ocrpkg.ExtractedDocument
	err error
}

func (p *scriptedOCRProvider) Extract(context.Context, string, string) (ocrpkg.ExtractedDocument, error) {
	if p.calls >= len(p.responses) {
		return ocrpkg.ExtractedDocument{}, errors.New("unexpected OCR extract call")
	}
	response := p.responses[p.calls]
	p.calls++
	if response.err != nil {
		return ocrpkg.ExtractedDocument{}, response.err
	}
	return response.doc, nil
}

func (p *scriptedOCRProvider) SuggestLineResolution(context.Context, ocrpkg.LineResolutionInput) (ocrpkg.OCRLineAssistSuggestion, error) {
	return ocrpkg.OCRLineAssistSuggestion{}, nil
}

func (p *scriptedOCRProvider) Name() string {
	return "scripted"
}

func TestCreateReceipt_ConvertsIncomingAllocationAndUpdatesInventory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	inventoryService := inventory.NewService(inventory.NewRepository(db))
	procurementService := procurement.NewService(procurement.NewRepository(db), nil, nil, nil)
	ctx := context.Background()

	requestID, err := procurementService.CreateRequest(ctx, procurement.ProcurementRequestCreateInput{
		Title:      "Receipt conversion coverage",
		SupplierID: "sup-misumi",
		Lines: []procurement.ProcurementRequestLineCreate{
			{
				ItemID:            "item-er2",
				RequestedQuantity: 4,
				DeliveryLocation:  "TOKYO-A1",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}

	requestDetail, err := procurementService.RequestDetail(ctx, requestID)
	if err != nil {
		t.Fatalf("RequestDetail failed: %v", err)
	}
	if len(requestDetail.Lines) != 1 {
		t.Fatalf("expected 1 procurement line, got %d", len(requestDetail.Lines))
	}

	orderDetail, err := procurementService.CreateOrder(ctx, procurement.PurchaseOrderCreateInput{
		ProcurementBatchID: requestID,
		Lines: []procurement.PurchaseOrderLineInput{
			{
				ProcurementLineID:   requestDetail.Lines[0].ID,
				OrderedQuantity:     4,
				ExpectedArrivalDate: "2026-05-01",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}
	if len(orderDetail.Lines) != 1 {
		t.Fatalf("expected 1 purchase order line, got %d", len(orderDetail.Lines))
	}

	_, err = inventoryService.BulkReservationConfirm(ctx, inventory.BulkReservationConfirmInput{
		ScopeID: "ds-er2-powerboard",
		ActorID: "receipt-tester",
		Rows: []inventory.BulkReservationConfirmRow{
			{
				ItemID:   "item-er2",
				Purpose:  "receipt conversion coverage",
				Priority: "normal",
				OrderAllocations: []inventory.OrderAllocation{
					{
						PurchaseOrderLineID: orderDetail.Lines[0].ID,
						Quantity:            3,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BulkReservationConfirm failed: %v", err)
	}

	receipt, err := inventoryService.CreateReceipt(ctx, inventory.ReceiptCreateInput{
		PurchaseOrderID: orderDetail.ID,
		ReceivedBy:      "receipt-tester",
		Lines: []inventory.ReceiptLineInput{
			{
				PurchaseOrderLineID: orderDetail.Lines[0].ID,
				ItemID:              "item-er2",
				LocationCode:        "TOKYO-A1",
				Quantity:            3,
				Note:                "partial receipt",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateReceipt failed: %v", err)
	}
	if receipt.PurchaseOrderID != orderDetail.ID {
		t.Fatalf("receipt purchase order id: want %s, got %s", orderDetail.ID, receipt.PurchaseOrderID)
	}

	receivedQty := testutil.MustQueryInt(t, db, `SELECT received_quantity FROM purchase_order_lines WHERE id = $1`, orderDetail.Lines[0].ID)
	if receivedQty != 3 {
		t.Fatalf("received quantity: want 3, got %d", receivedQty)
	}

	var poLineStatus string
	if err := db.QueryRowContext(ctx, `SELECT status FROM purchase_order_lines WHERE id = $1`, orderDetail.Lines[0].ID).Scan(&poLineStatus); err != nil {
		t.Fatalf("query purchase order line status: %v", err)
	}
	if poLineStatus != "partially_received" {
		t.Fatalf("purchase order line status: want partially_received, got %s", poLineStatus)
	}

	onHand := testutil.MustQueryInt(t, db, `SELECT on_hand_quantity FROM inventory_balances WHERE item_id = $1 AND location_code = $2`, "item-er2", "TOKYO-A1")
	reserved := testutil.MustQueryInt(t, db, `SELECT reserved_quantity FROM inventory_balances WHERE item_id = $1 AND location_code = $2`, "item-er2", "TOKYO-A1")
	available := testutil.MustQueryInt(t, db, `SELECT available_quantity FROM inventory_balances WHERE item_id = $1 AND location_code = $2`, "item-er2", "TOKYO-A1")
	if onHand != 3 || reserved != 3 || available != 0 {
		t.Fatalf("balance mismatch: onHand=%d reserved=%d available=%d", onHand, reserved, available)
	}

	convertedCount := testutil.MustQueryInt(t, db, `
		SELECT COUNT(*)
		FROM reservation_allocations
		WHERE purchase_order_line_id IS NULL
		  AND source_type = 'stock'
		  AND location_code = 'TOKYO-A1'
	`)
	if convertedCount != 1 {
		t.Fatalf("converted stock allocation count: want 1, got %d", convertedCount)
	}

	incomingCount := testutil.MustQueryInt(t, db, `
		SELECT COUNT(*)
		FROM reservation_allocations
		WHERE purchase_order_line_id = $1
		  AND source_type = 'incoming_order'
	`, orderDetail.Lines[0].ID)
	if incomingCount != 0 {
		t.Fatalf("incoming allocation count after receipt: want 0, got %d", incomingCount)
	}

	receiveEventCount := testutil.MustQueryInt(t, db, `
		SELECT COUNT(*)
		FROM inventory_events
		WHERE source_type = 'receipt'
		  AND source_id = $1
		  AND event_type = 'receive'
	`, receipt.ID)
	if receiveEventCount != 1 {
		t.Fatalf("receipt inventory event count: want 1, got %d", receiveEventCount)
	}

	convertedEventCount := testutil.MustQueryInt(t, db, `
		SELECT COUNT(*)
		FROM inventory_events
		WHERE event_type = 'reserve_allocate'
		  AND correlation_id LIKE $1
	`, receipt.ID+":%")
	if convertedEventCount != 1 {
		t.Fatalf("converted allocation event count: want 1, got %d", convertedEventCount)
	}

	auditCount := testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM audit_events WHERE event_type = 'receipt.created'`)
	if auditCount != 1 {
		t.Fatalf("receipt audit count: want 1, got %d", auditCount)
	}
}

func TestReservationImport_UndoAndReapply(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := inventory.NewService(inventory.NewRepository(db))
	ctx := context.Background()

	csvData := []byte("item_id,device_scope_id,quantity,purpose\nitem-er2,ds-er2-powerboard,2,Undo coverage\n")

	firstApply, err := service.ReservationImportApply(ctx, "reservations.csv", csvData, "import-tester")
	if err != nil {
		t.Fatalf("ReservationImportApply failed: %v", err)
	}
	if firstApply.Created != 1 {
		t.Fatalf("first import created count: want 1, got %d", firstApply.Created)
	}
	if firstApply.Detail.LifecycleState != "applied" {
		t.Fatalf("first import lifecycle state: want applied, got %s", firstApply.Detail.LifecycleState)
	}
	if len(firstApply.Detail.Effects) != 1 || firstApply.Detail.Effects[0].TargetEntityType != "reservation" {
		t.Fatalf("unexpected import effects: %+v", firstApply.Detail.Effects)
	}

	reservationCount := testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM reservations WHERE item_id = 'item-er2'`)
	if reservationCount != 1 {
		t.Fatalf("reservation count after import: want 1, got %d", reservationCount)
	}

	undone, err := service.UndoImport(ctx, firstApply.JobID, "import-tester")
	if err != nil {
		t.Fatalf("UndoImport failed: %v", err)
	}
	if undone.LifecycleState != "undone" {
		t.Fatalf("undo lifecycle state: want undone, got %s", undone.LifecycleState)
	}
	if undone.UndoneAt == "" {
		t.Fatal("expected undoneAt to be set")
	}

	reservationCount = testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM reservations WHERE item_id = 'item-er2'`)
	if reservationCount != 0 {
		t.Fatalf("reservation count after undo: want 0, got %d", reservationCount)
	}

	secondApply, err := service.ReservationImportApply(ctx, "reservations.csv", csvData, "import-tester")
	if err != nil {
		t.Fatalf("second ReservationImportApply failed: %v", err)
	}
	if secondApply.Created != 1 {
		t.Fatalf("second import created count: want 1, got %d", secondApply.Created)
	}

	reservationCount = testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM reservations WHERE item_id = 'item-er2'`)
	if reservationCount != 1 {
		t.Fatalf("reservation count after reapply: want 1, got %d", reservationCount)
	}
}

func TestOCRRetryAndCreateProcurementDraft(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &testArtifactStore{}
	provider := &scriptedOCRProvider{
		responses: []scriptedOCRResponse{
			{err: errors.New("temporary OCR outage")},
			{
				doc: ocrpkg.ExtractedDocument{
					SupplierName:    "MISUMI",
					SupplierID:      "sup-misumi",
					QuotationNumber: "Q-2026-001",
					IssueDate:       "2026-04-25",
					RawPayload:      `{"supplier_name":"MISUMI","quotation_number":"Q-2026-001"}`,
					Lines: []ocrpkg.OCRResultLine{
						{
							ItemID:             "item-er2",
							ManufacturerName:   "Omron",
							ItemNumber:         "ER2-P4",
							ItemDescription:    "Control relay pack",
							Quantity:           4,
							LeadTimeDays:       7,
							DeliveryLocation:   "TOKYO-A1",
							AccountingCategory: "capex",
						},
					},
				},
			},
		},
	}
	procurementService := procurement.NewService(procurement.NewRepository(db), nil, nil, nil)
	ocrService := ocrpkg.NewService(ocrpkg.NewRepository(db), store, provider, procurementService)
	ctx := context.Background()

	if _, err := ocrService.CreateJob(ctx, "quote.pdf", "application/pdf", "ocr-tester", strings.NewReader("fake pdf")); err == nil {
		t.Fatal("expected CreateJob to fail on first OCR extract attempt")
	}

	var jobID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM ocr_jobs WHERE file_name = 'quote.pdf'`).Scan(&jobID); err != nil {
		t.Fatalf("query failed OCR job id: %v", err)
	}

	jobStatusCount := testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM ocr_jobs WHERE id = $1 AND status = 'failed'`, jobID)
	if jobStatusCount != 1 {
		t.Fatalf("failed OCR job count: want 1, got %d", jobStatusCount)
	}

	retryResult, err := ocrService.RetryJob(ctx, jobID)
	if err != nil {
		t.Fatalf("RetryJob failed: %v", err)
	}
	if retryResult.Status != "ready_for_review" || retryResult.RetryCount != 1 {
		t.Fatalf("unexpected retry result: %+v", retryResult)
	}

	detail, err := ocrService.JobDetail(ctx, jobID)
	if err != nil {
		t.Fatalf("JobDetail failed: %v", err)
	}
	if len(detail.Lines) != 1 {
		t.Fatalf("expected 1 OCR line, got %d", len(detail.Lines))
	}

	if err := ocrService.UpdateReview(ctx, jobID, ocrpkg.OCRReviewUpdateInput{
		SupplierID:      "sup-misumi",
		QuotationNumber: "Q-2026-001",
		IssueDate:       "2026-04-25",
		Lines: []ocrpkg.OCRLineUpdate{
			{
				ID:                 detail.Lines[0].ID,
				ItemID:             "item-er2",
				DeliveryLocation:   "TOKYO-A1",
				AccountingCategory: "capex",
				IsUserConfirmed:    true,
			},
		},
	}); err != nil {
		t.Fatalf("UpdateReview failed: %v", err)
	}

	draft, err := ocrService.CreateProcurementDraft(ctx, jobID)
	if err != nil {
		t.Fatalf("CreateProcurementDraft failed: %v", err)
	}
	if draft.Status != "created" {
		t.Fatalf("draft status: want created, got %s", draft.Status)
	}

	requestDetail, err := procurementService.RequestDetail(ctx, draft.ProcurementRequestID)
	if err != nil {
		t.Fatalf("RequestDetail for OCR draft failed: %v", err)
	}
	if len(requestDetail.Lines) != 1 {
		t.Fatalf("expected 1 OCR procurement line, got %d", len(requestDetail.Lines))
	}

	batchCount := testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM procurement_batches WHERE source_ocr_job_id = $1`, jobID)
	if batchCount != 1 {
		t.Fatalf("procurement batch count from OCR job: want 1, got %d", batchCount)
	}

	quotationCount := testutil.MustQueryInt(t, db, `SELECT COUNT(*) FROM supplier_quotations WHERE source_ocr_job_id = $1`, jobID)
	if quotationCount != 1 {
		t.Fatalf("quotation count from OCR job: want 1, got %d", quotationCount)
	}

	reusedDraft, err := ocrService.CreateProcurementDraft(ctx, jobID)
	if err != nil {
		t.Fatalf("second CreateProcurementDraft failed: %v", err)
	}
	if reusedDraft.Status != "existing" {
		t.Fatalf("second draft status: want existing, got %s", reusedDraft.Status)
	}
}
