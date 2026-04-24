package ocr

type OCRJobSummary struct {
	ID          string `json:"id"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Status      string `json:"status"`
	Provider    string `json:"provider"`
	RetryCount  int    `json:"retryCount"`
	CreatedBy   string `json:"createdBy"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type OCRJobList struct {
	Rows []OCRJobSummary `json:"rows"`
}

type OCRResultLine struct {
	ID                 string             `json:"id"`
	ItemID             string             `json:"itemId"`
	ManufacturerName   string             `json:"manufacturerName"`
	ItemNumber         string             `json:"itemNumber"`
	ItemDescription    string             `json:"itemDescription"`
	Quantity           int                `json:"quantity"`
	LeadTimeDays       int                `json:"leadTimeDays"`
	DeliveryLocation   string             `json:"deliveryLocation"`
	BudgetCategoryID   string             `json:"budgetCategoryId"`
	AccountingCategory string             `json:"accountingCategory"`
	SupplierContact    string             `json:"supplierContact"`
	IsUserConfirmed    bool               `json:"isUserConfirmed"`
	MatchCandidates    []OCRItemCandidate `json:"matchCandidates,omitempty"`
}

type OCRJobDetail struct {
	ID                     string                 `json:"id"`
	FileName               string                 `json:"fileName"`
	ContentType            string                 `json:"contentType"`
	ArtifactPath           string                 `json:"artifactPath"`
	Status                 string                 `json:"status"`
	Provider               string                 `json:"provider"`
	ErrorMessage           string                 `json:"errorMessage"`
	RetryCount             int                    `json:"retryCount"`
	SupplierName           string                 `json:"supplierName"`
	SupplierID             string                 `json:"supplierId"`
	SupplierMatch          []OCRSupplierCandidate `json:"supplierMatch"`
	QuotationID            string                 `json:"quotationId"`
	QuotationNumber        string                 `json:"quotationNumber"`
	IssueDate              string                 `json:"issueDate"`
	ProcurementRequestID   string                 `json:"procurementRequestId"`
	ProcurementBatchNumber string                 `json:"procurementBatchNumber"`
	RawPayload             string                 `json:"rawPayload"`
	Lines                  []OCRResultLine        `json:"lines"`
}

type OCRSupplierCandidate struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	MatchReason string  `json:"matchReason"`
}

type OCRItemCandidate struct {
	ItemID              string  `json:"itemId"`
	CanonicalItemNumber string  `json:"canonicalItemNumber"`
	Description         string  `json:"description"`
	ManufacturerName    string  `json:"manufacturerName"`
	DefaultSupplierID   string  `json:"defaultSupplierId"`
	SupplierAlias       string  `json:"supplierAlias"`
	Score               float64 `json:"score"`
	MatchReason         string  `json:"matchReason"`
}

type OCRLineAssistSuggestion struct {
	LineID                   string             `json:"lineId"`
	MatchedItemID            string             `json:"matchedItemId"`
	SuggestedCanonicalNumber string             `json:"suggestedCanonicalNumber"`
	SuggestedManufacturer    string             `json:"suggestedManufacturer"`
	SuggestedCategoryKey     string             `json:"suggestedCategoryKey"`
	SuggestedAliasNumber     string             `json:"suggestedAliasNumber"`
	Confidence               float64            `json:"confidence"`
	Rationale                string             `json:"rationale"`
	Candidates               []OCRItemCandidate `json:"candidates"`
}

type OCRLineAssistInput struct {
	LineID string `json:"lineId"`
}

type OCRRegisterItemInput struct {
	LineID              string `json:"lineId"`
	CanonicalItemNumber string `json:"canonicalItemNumber"`
	Description         string `json:"description"`
	ManufacturerName    string `json:"manufacturerName"`
	CategoryKey         string `json:"categoryKey"`
	CategoryName        string `json:"categoryName"`
	DefaultSupplierID   string `json:"defaultSupplierId"`
	SupplierAliasNumber string `json:"supplierAliasNumber"`
	UnitsPerOrder       int    `json:"unitsPerOrder"`
}

type OCRJobCreateResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type OCRProcurementDraftCreateResult struct {
	ProcurementRequestID   string `json:"procurementRequestId"`
	ProcurementBatchNumber string `json:"procurementBatchNumber"`
	QuotationID            string `json:"quotationId"`
	Status                 string `json:"status"`
}

type OCRRetryResult struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	RetryCount int    `json:"retryCount"`
}

type OCRLineUpdate struct {
	ID                 string `json:"id"`
	ItemID             string `json:"itemId"`
	DeliveryLocation   string `json:"deliveryLocation"`
	BudgetCategoryID   string `json:"budgetCategoryId"`
	AccountingCategory string `json:"accountingCategory"`
	SupplierContact    string `json:"supplierContact"`
	IsUserConfirmed    bool   `json:"isUserConfirmed"`
}

type OCRReviewUpdateInput struct {
	SupplierID      string          `json:"supplierId"`
	QuotationNumber string          `json:"quotationNumber"`
	IssueDate       string          `json:"issueDate"`
	Lines           []OCRLineUpdate `json:"lines"`
}

type ExtractedDocument struct {
	SupplierName    string
	SupplierID      string
	QuotationNumber string
	IssueDate       string
	RawPayload      string
	Lines           []OCRResultLine
}
