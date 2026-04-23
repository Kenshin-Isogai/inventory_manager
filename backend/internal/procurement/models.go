package procurement

type ProjectSummary struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	SyncedAt string `json:"syncedAt"`
}

type BudgetCategorySummary struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	SyncedAt  string `json:"syncedAt"`
}

type ProcurementRequestSummary struct {
	ID                   string `json:"id"`
	BatchNumber          string `json:"batchNumber"`
	Title                string `json:"title"`
	ProjectName          string `json:"projectName"`
	BudgetCategoryName   string `json:"budgetCategoryName"`
	SupplierName         string `json:"supplierName"`
	NormalizedStatus     string `json:"normalizedStatus"`
	SourceType           string `json:"sourceType"`
	RequestedItems       int    `json:"requestedItems"`
	DispatchStatus       string `json:"dispatchStatus"`
	ArtifactDeleteStatus string `json:"artifactDeleteStatus"`
	CreatedAt            string `json:"createdAt"`
}

type ProcurementRequestList struct {
	Rows []ProcurementRequestSummary `json:"rows"`
}

type ProcurementLine struct {
	ID                 string `json:"id"`
	ItemNumber         string `json:"itemNumber"`
	Description        string `json:"description"`
	RequestedQuantity  int    `json:"requestedQuantity"`
	DeliveryLocation   string `json:"deliveryLocation"`
	AccountingCategory string `json:"accountingCategory"`
	LeadTimeDays       int    `json:"leadTimeDays"`
	Note               string `json:"note"`
}

type StatusHistoryEntry struct {
	ID               string `json:"id"`
	NormalizedStatus string `json:"normalizedStatus"`
	RawStatus        string `json:"rawStatus"`
	ObservedAt       string `json:"observedAt"`
	Note             string `json:"note"`
}

type ProcurementRequestDetail struct {
	ID                       string                            `json:"id"`
	BatchNumber              string                            `json:"batchNumber"`
	Title                    string                            `json:"title"`
	ProjectName              string                            `json:"projectName"`
	BudgetCategoryName       string                            `json:"budgetCategoryName"`
	SupplierName             string                            `json:"supplierName"`
	QuotationNumber          string                            `json:"quotationNumber"`
	QuotationIssueDate       string                            `json:"quotationIssueDate"`
	ArtifactPath             string                            `json:"artifactPath"`
	ArtifactDeleteStatus     string                            `json:"artifactDeleteStatus"`
	ArtifactDeletedAt        string                            `json:"artifactDeletedAt"`
	NormalizedStatus         string                            `json:"normalizedStatus"`
	RawStatus                string                            `json:"rawStatus"`
	ExternalRequestReference string                            `json:"externalRequestReference"`
	DispatchStatus           string                            `json:"dispatchStatus"`
	DispatchAttempts         int                               `json:"dispatchAttempts"`
	LastDispatchAt           string                            `json:"lastDispatchAt"`
	DispatchErrorCode        string                            `json:"dispatchErrorCode"`
	DispatchErrorMessage     string                            `json:"dispatchErrorMessage"`
	QuantityProgression      string                            `json:"quantityProgression"`
	LastReconciledAt         string                            `json:"lastReconciledAt"`
	SyncSource               string                            `json:"syncSource"`
	SyncError                string                            `json:"syncError"`
	Lines                    []ProcurementLine                 `json:"lines"`
	StatusHistory            []StatusHistoryEntry              `json:"statusHistory"`
	DispatchHistory          []ProcurementDispatchHistoryEntry `json:"dispatchHistory"`
}

type ProcurementRequestCreateInput struct {
	Title            string                         `json:"title"`
	ProjectID        string                         `json:"projectId"`
	BudgetCategoryID string                         `json:"budgetCategoryId"`
	SupplierID       string                         `json:"supplierId"`
	QuotationID      string                         `json:"quotationId"`
	SourceType       string                         `json:"sourceType"`
	CreatedBy        string                         `json:"createdBy"`
	Lines            []ProcurementRequestLineCreate `json:"lines"`
}

type ProcurementRequestLineCreate struct {
	ItemID             string `json:"itemId"`
	QuotationLineID    string `json:"quotationLineId"`
	RequestedQuantity  int    `json:"requestedQuantity"`
	DeliveryLocation   string `json:"deliveryLocation"`
	AccountingCategory string `json:"accountingCategory"`
	Note               string `json:"note"`
}

type ProcurementRequestUpdateInput struct {
	Title            string                         `json:"title"`
	ProjectID        string                         `json:"projectId"`
	BudgetCategoryID string                         `json:"budgetCategoryId"`
	SupplierID       string                         `json:"supplierId"`
	Lines            []ProcurementRequestLineUpdate `json:"lines"`
}

type ProcurementRequestLineUpdate struct {
	ID                 string `json:"id"`
	ItemID             string `json:"itemId"`
	QuotationLineID    string `json:"quotationLineId"`
	RequestedQuantity  int    `json:"requestedQuantity"`
	DeliveryLocation   string `json:"deliveryLocation"`
	AccountingCategory string `json:"accountingCategory"`
	BudgetCategoryID   string `json:"budgetCategoryId"`
	SupplierContact    string `json:"supplierContact"`
	LeadTimeDays       int    `json:"leadTimeDays"`
	Status             string `json:"status"`
	Note               string `json:"note"`
}

type OCRProcurementDraftCreateInput struct {
	SourceOCRJobID  string                          `json:"sourceOcrJobId"`
	Title           string                          `json:"title"`
	SupplierID      string                          `json:"supplierId"`
	QuotationNumber string                          `json:"quotationNumber"`
	IssueDate       string                          `json:"issueDate"`
	ArtifactPath    string                          `json:"artifactPath"`
	CreatedBy       string                          `json:"createdBy"`
	Lines           []OCRProcurementDraftLineCreate `json:"lines"`
}

type OCRProcurementDraftLineCreate struct {
	ItemID             string `json:"itemId"`
	ManufacturerName   string `json:"manufacturerName"`
	ItemNumber         string `json:"itemNumber"`
	ItemDescription    string `json:"itemDescription"`
	Quantity           int    `json:"quantity"`
	LeadTimeDays       int    `json:"leadTimeDays"`
	DeliveryLocation   string `json:"deliveryLocation"`
	BudgetCategoryID   string `json:"budgetCategoryId"`
	AccountingCategory string `json:"accountingCategory"`
	SupplierContact    string `json:"supplierContact"`
	Note               string `json:"note"`
}

type OCRProcurementDraftCreateResult struct {
	ProcurementRequestID   string `json:"procurementRequestId"`
	ProcurementBatchNumber string `json:"procurementBatchNumber"`
	QuotationID            string `json:"quotationId"`
	Status                 string `json:"status"`
}

type ProcurementDispatchHistoryEntry struct {
	ID                       string `json:"id"`
	DispatchStatus           string `json:"dispatchStatus"`
	ExternalRequestReference string `json:"externalRequestReference"`
	Retryable                bool   `json:"retryable"`
	ErrorCode                string `json:"errorCode"`
	ErrorMessage             string `json:"errorMessage"`
	ObservedAt               string `json:"observedAt"`
}

type ProcurementSubmitResult struct {
	RequestID                string `json:"requestId"`
	ExternalRequestReference string `json:"externalRequestReference"`
	DispatchStatus           string `json:"dispatchStatus"`
	ArtifactDeleteStatus     string `json:"artifactDeleteStatus"`
}

type ProcurementQuantityProgression struct {
	Requested int `json:"requested"`
	Ordered   int `json:"ordered"`
	Received  int `json:"received"`
}

type ProcurementReconcileResult struct {
	RequestID           string `json:"requestId"`
	NormalizedStatus    string `json:"normalizedStatus"`
	RawStatus           string `json:"rawStatus"`
	QuantityProgression string `json:"quantityProgression"`
	LastReconciledAt    string `json:"lastReconciledAt"`
	SyncSource          string `json:"syncSource"`
}

type PurchaseOrderSummary struct {
	ID                 string `json:"id"`
	ProcurementBatchID string `json:"procurementBatchId"`
	BatchNumber        string `json:"batchNumber"`
	OrderNumber        string `json:"orderNumber"`
	Status             string `json:"status"`
	SupplierName       string `json:"supplierName"`
	OrderedQuantity    int    `json:"orderedQuantity"`
	ReceivedQuantity   int    `json:"receivedQuantity"`
	OpenQuantity       int    `json:"openQuantity"`
	IssuedAt           string `json:"issuedAt"`
}

type PurchaseOrderList struct {
	Rows []PurchaseOrderSummary `json:"rows"`
}

type PurchaseOrderLine struct {
	ID                  string `json:"id"`
	ProcurementLineID   string `json:"procurementLineId"`
	ItemID              string `json:"itemId"`
	ItemNumber          string `json:"itemNumber"`
	Description         string `json:"description"`
	OrderedQuantity     int    `json:"orderedQuantity"`
	ReceivedQuantity    int    `json:"receivedQuantity"`
	OpenQuantity        int    `json:"openQuantity"`
	ExpectedArrivalDate string `json:"expectedArrivalDate"`
	Status              string `json:"status"`
	DeliveryLocation    string `json:"deliveryLocation"`
	Note                string `json:"note"`
}

type PurchaseOrderDetail struct {
	ID                 string              `json:"id"`
	ProcurementBatchID string              `json:"procurementBatchId"`
	BatchNumber        string              `json:"batchNumber"`
	Title              string              `json:"title"`
	OrderNumber        string              `json:"orderNumber"`
	Status             string              `json:"status"`
	SupplierName       string              `json:"supplierName"`
	IssuedAt           string              `json:"issuedAt"`
	Lines              []PurchaseOrderLine `json:"lines"`
}

type PurchaseOrderLineInput struct {
	ID                  string `json:"id"`
	ProcurementLineID   string `json:"procurementLineId"`
	OrderedQuantity     int    `json:"orderedQuantity"`
	ExpectedArrivalDate string `json:"expectedArrivalDate"`
	Status              string `json:"status"`
	Note                string `json:"note"`
}

type PurchaseOrderCreateInput struct {
	ProcurementBatchID string                   `json:"procurementBatchId"`
	OrderNumber        string                   `json:"orderNumber"`
	Status             string                   `json:"status"`
	IssuedAt           string                   `json:"issuedAt"`
	Lines              []PurchaseOrderLineInput `json:"lines"`
}

type PurchaseOrderUpdateInput struct {
	OrderNumber string                   `json:"orderNumber"`
	Status      string                   `json:"status"`
	IssuedAt    string                   `json:"issuedAt"`
	Lines       []PurchaseOrderLineInput `json:"lines"`
}

type MasterSyncResult struct {
	SyncType    string `json:"syncType"`
	ProjectID   string `json:"projectId"`
	ProjectKey  string `json:"projectKey"`
	Status      string `json:"status"`
	RowCount    int    `json:"rowCount"`
	Source      string `json:"source"`
	TriggeredBy string `json:"triggeredBy"`
	SyncedAt    string `json:"syncedAt"`
}

type WebhookProcessResult struct {
	EventType  string `json:"eventType"`
	Status     string `json:"status"`
	RequestID  string `json:"requestId"`
	ProjectKey string `json:"projectKey"`
	SyncedAt   string `json:"syncedAt"`
}

type MasterSyncRunEntry struct {
	ID           string `json:"id"`
	SyncType     string `json:"syncType"`
	ProjectID    string `json:"projectId"`
	ProjectKey   string `json:"projectKey"`
	Status       string `json:"status"`
	RowCount     int    `json:"rowCount"`
	Source       string `json:"source"`
	TriggeredBy  string `json:"triggeredBy"`
	ErrorMessage string `json:"errorMessage"`
	StartedAt    string `json:"startedAt"`
	FinishedAt   string `json:"finishedAt"`
}

type WebhookEventEntry struct {
	ID                       string `json:"id"`
	EventType                string `json:"eventType"`
	ExternalRequestReference string `json:"externalRequestReference"`
	ProjectKey               string `json:"projectKey"`
	NormalizedStatus         string `json:"normalizedStatus"`
	RawStatus                string `json:"rawStatus"`
	ReceivedAt               string `json:"receivedAt"`
	ProcessedAt              string `json:"processedAt"`
	ProcessingError          string `json:"processingError"`
}

type ProcurementSubmissionPayload struct {
	BatchID            string                   `json:"batchId"`
	BatchNumber        string                   `json:"batchNumber"`
	Title              string                   `json:"title"`
	IdempotencyKey     string                   `json:"idempotencyKey"`
	ProjectKey         string                   `json:"projectKey"`
	ProjectName        string                   `json:"projectName"`
	BudgetCategoryKey  string                   `json:"budgetCategoryKey"`
	BudgetCategoryName string                   `json:"budgetCategoryName"`
	SupplierID         string                   `json:"supplierId"`
	SupplierName       string                   `json:"supplierName"`
	QuotationID        string                   `json:"quotationId"`
	QuotationNumber    string                   `json:"quotationNumber"`
	QuotationIssueDate string                   `json:"quotationIssueDate"`
	ArtifactPath       string                   `json:"artifactPath"`
	Lines              []ProcurementPayloadLine `json:"lines"`
}

type ProcurementPayloadLine struct {
	ItemID             string `json:"itemId"`
	ItemNumber         string `json:"itemNumber"`
	Description        string `json:"description"`
	RequestedQuantity  int    `json:"requestedQuantity"`
	DeliveryLocation   string `json:"deliveryLocation"`
	AccountingCategory string `json:"accountingCategory"`
	LeadTimeDays       int    `json:"leadTimeDays"`
	SupplierContact    string `json:"supplierContact"`
}
