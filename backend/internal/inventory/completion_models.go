package inventory

type RequirementSummary struct {
	ID          string `json:"id"`
	Device      string `json:"device"`
	Scope       string `json:"scope"`
	ItemID      string `json:"itemId"`
	ItemNumber  string `json:"itemNumber"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	Note        string `json:"note"`
}

type RequirementList struct {
	Rows []RequirementSummary `json:"rows"`
}

type RequirementUpsertInput struct {
	ID            string `json:"id"`
	DeviceScopeID string `json:"deviceScopeId"`
	ItemID        string `json:"itemId"`
	Quantity      int    `json:"quantity"`
	Note          string `json:"note"`
}

type ReservationAllocation struct {
	ID           string `json:"id"`
	LocationCode string `json:"locationCode"`
	Quantity     int    `json:"quantity"`
	Status       string `json:"status"`
	AllocatedAt  string `json:"allocatedAt"`
	ReleasedAt   string `json:"releasedAt"`
	Note         string `json:"note"`
}

type ReservationDetail struct {
	ID                 string                  `json:"id"`
	ItemID             string                  `json:"itemId"`
	ItemNumber         string                  `json:"itemNumber"`
	Description        string                  `json:"description"`
	DeviceScopeID      string                  `json:"deviceScopeId"`
	Device             string                  `json:"device"`
	Scope              string                  `json:"scope"`
	Quantity           int                     `json:"quantity"`
	AllocatedQuantity  int                     `json:"allocatedQuantity"`
	Status             string                  `json:"status"`
	Purpose            string                  `json:"purpose"`
	Priority           string                  `json:"priority"`
	NeededByAt         string                  `json:"neededByAt"`
	PlannedUseAt       string                  `json:"plannedUseAt"`
	HoldUntilAt        string                  `json:"holdUntilAt"`
	FulfilledAt        string                  `json:"fulfilledAt"`
	ReleasedAt         string                  `json:"releasedAt"`
	CancellationReason string                  `json:"cancellationReason"`
	RequestedBy        string                  `json:"requestedBy"`
	Note               string                  `json:"note"`
	Allocations        []ReservationAllocation `json:"allocations"`
}

type ReservationActionInput struct {
	LocationCode string `json:"locationCode"`
	Quantity     int    `json:"quantity"`
	Reason       string `json:"reason"`
	ActorID      string `json:"actorId"`
	Note         string `json:"note"`
}

type ReservationUpdateInput struct {
	ItemID        string `json:"itemId"`
	DeviceScopeID string `json:"deviceScopeId"`
	Quantity      int    `json:"quantity"`
	RequestedBy   string `json:"requestedBy"`
	Purpose       string `json:"purpose"`
	Priority      string `json:"priority"`
	NeededByAt    string `json:"neededByAt"`
	PlannedUseAt  string `json:"plannedUseAt"`
	HoldUntilAt   string `json:"holdUntilAt"`
	Note          string `json:"note"`
}

type InventoryItemSummary struct {
	ItemID            string `json:"itemId"`
	ItemNumber        string `json:"itemNumber"`
	Description       string `json:"description"`
	Manufacturer      string `json:"manufacturer"`
	Category          string `json:"category"`
	OnHandQuantity    int    `json:"onHandQuantity"`
	ReservedQuantity  int    `json:"reservedQuantity"`
	AvailableQuantity int    `json:"availableQuantity"`
}

type InventoryItemList struct {
	Rows []InventoryItemSummary `json:"rows"`
}

type LocationSummary struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	LocationType      string `json:"locationType"`
	IsActive          bool   `json:"isActive"`
	OnHandQuantity    int    `json:"onHandQuantity"`
	ReservedQuantity  int    `json:"reservedQuantity"`
	AvailableQuantity int    `json:"availableQuantity"`
}

type LocationList struct {
	Rows []LocationSummary `json:"rows"`
}

type InventoryEventEntry struct {
	ID                string `json:"id"`
	EventType         string `json:"eventType"`
	ItemID            string `json:"itemId"`
	ItemNumber        string `json:"itemNumber"`
	FromLocationCode  string `json:"fromLocationCode"`
	ToLocationCode    string `json:"toLocationCode"`
	QuantityDelta     int    `json:"quantityDelta"`
	ActorID           string `json:"actorId"`
	SourceType        string `json:"sourceType"`
	SourceID          string `json:"sourceId"`
	CorrelationID     string `json:"correlationId"`
	ReversedByEventID string `json:"reversedByEventId"`
	Note              string `json:"note"`
	OccurredAt        string `json:"occurredAt"`
}

type InventoryEventList struct {
	Rows []InventoryEventEntry `json:"rows"`
}

type InventoryReceiveInput struct {
	ItemID        string `json:"itemId"`
	LocationCode  string `json:"locationCode"`
	Quantity      int    `json:"quantity"`
	DeviceScopeID string `json:"deviceScopeId"`
	ActorID       string `json:"actorId"`
	SourceType    string `json:"sourceType"`
	SourceID      string `json:"sourceId"`
	Note          string `json:"note"`
}

type InventoryMoveInput struct {
	ItemID           string `json:"itemId"`
	FromLocationCode string `json:"fromLocationCode"`
	ToLocationCode   string `json:"toLocationCode"`
	Quantity         int    `json:"quantity"`
	DeviceScopeID    string `json:"deviceScopeId"`
	ActorID          string `json:"actorId"`
	SourceType       string `json:"sourceType"`
	SourceID         string `json:"sourceId"`
	Note             string `json:"note"`
}

type InventoryUndoInput struct {
	ActorID string `json:"actorId"`
	Reason  string `json:"reason"`
}

type ReceiptLineInput struct {
	PurchaseOrderLineID string `json:"purchaseOrderLineId"`
	ItemID              string `json:"itemId"`
	LocationCode        string `json:"locationCode"`
	Quantity            int    `json:"quantity"`
	Note                string `json:"note"`
}

type ReceiptCreateInput struct {
	PurchaseOrderID string             `json:"purchaseOrderId"`
	SourceType      string             `json:"sourceType"`
	ReceivedBy      string             `json:"receivedBy"`
	ReceivedAt      string             `json:"receivedAt"`
	Note            string             `json:"note"`
	Lines           []ReceiptLineInput `json:"lines"`
}

type ReceiptSummary struct {
	ID              string `json:"id"`
	ReceiptNumber   string `json:"receiptNumber"`
	PurchaseOrderID string `json:"purchaseOrderId"`
	SourceType      string `json:"sourceType"`
	ReceivedBy      string `json:"receivedBy"`
	ReceivedAt      string `json:"receivedAt"`
	Note            string `json:"note"`
}

type ArrivalSummary struct {
	PurchaseOrderLineID string `json:"purchaseOrderLineId"`
	PurchaseOrderID     string `json:"purchaseOrderId"`
	OrderNumber         string `json:"orderNumber"`
	ItemID              string `json:"itemId"`
	ItemNumber          string `json:"itemNumber"`
	Description         string `json:"description"`
	OrderedQuantity     int    `json:"orderedQuantity"`
	ReceivedQuantity    int    `json:"receivedQuantity"`
	PendingQuantity     int    `json:"pendingQuantity"`
	SupplierName        string `json:"supplierName"`
	ExpectedArrivalDate string `json:"expectedArrivalDate"`
	Status              string `json:"status"`
}

type ArrivalList struct {
	Rows []ArrivalSummary `json:"rows"`
}

type ImportPreviewRow struct {
	RowNumber int               `json:"rowNumber"`
	Status    string            `json:"status"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Raw       map[string]string `json:"raw"`
}

type ImportPreviewResult struct {
	ImportType string             `json:"importType"`
	FileName   string             `json:"fileName"`
	Status     string             `json:"status"`
	Rows       []ImportPreviewRow `json:"rows"`
}

type ImportDetail struct {
	ID             string                `json:"id"`
	ImportType     string                `json:"importType"`
	Status         string                `json:"status"`
	LifecycleState string                `json:"lifecycleState"`
	FileName       string                `json:"fileName"`
	Summary        string                `json:"summary"`
	CreatedAt      string                `json:"createdAt"`
	UndoneAt       string                `json:"undoneAt"`
	Rows           []ImportPreviewRow    `json:"rows"`
	Effects        []ImportEffectSummary `json:"effects"`
}

type CSVImportApplyResult struct {
	JobID   string       `json:"jobId"`
	Created int          `json:"created"`
	Updated int          `json:"updated"`
	Skipped int          `json:"skipped"`
	Errored int          `json:"errored"`
	Detail  ImportDetail `json:"detail"`
}

type ImportEffectSummary struct {
	ID               string `json:"id"`
	TargetEntityType string `json:"targetEntityType"`
	TargetEntityID   string `json:"targetEntityId"`
	EffectType       string `json:"effectType"`
}

type MasterItemRecord struct {
	ID                string `json:"id"`
	ItemNumber        string `json:"itemNumber"`
	Description       string `json:"description"`
	ManufacturerKey   string `json:"manufacturerKey"`
	CategoryKey       string `json:"categoryKey"`
	DefaultSupplierID string `json:"defaultSupplierId"`
	Note              string `json:"note"`
	LifecycleStatus   string `json:"lifecycleStatus"`
}

type MasterItemList struct {
	Rows []MasterItemRecord `json:"rows"`
}

type InventorySnapshotScopeSummary struct {
	Device              string `json:"device"`
	Scope               string `json:"scope"`
	RequirementQuantity int    `json:"requirementQuantity"`
	ReservationQuantity int    `json:"reservationQuantity"`
	AllocatedQuantity   int    `json:"allocatedQuantity"`
	RemainingDemand     int    `json:"remainingDemand"`
}

type InventorySnapshotRow struct {
	ItemID                  string                          `json:"itemId"`
	ItemNumber              string                          `json:"itemNumber"`
	Description             string                          `json:"description"`
	Manufacturer            string                          `json:"manufacturer"`
	Category                string                          `json:"category"`
	OnHandQuantity          int                             `json:"onHandQuantity"`
	AllocatedReservedQty    int                             `json:"allocatedReservedQuantity"`
	FreeQuantity            int                             `json:"freeQuantity"`
	RequirementQuantity     int                             `json:"requirementQuantity"`
	ReservationQuantity     int                             `json:"reservationQuantity"`
	UncoveredDemandQuantity int                             `json:"uncoveredDemandQuantity"`
	IncomingQuantity        int                             `json:"incomingQuantity"`
	NetAvailableQuantity    int                             `json:"netAvailableQuantity"`
	ScopeSummaries          []InventorySnapshotScopeSummary `json:"scopeSummaries"`
}

type InventorySnapshot struct {
	GeneratedAt       string                 `json:"generatedAt"`
	SnapshotSignature string                 `json:"snapshotSignature"`
	DeviceFilter      string                 `json:"deviceFilter"`
	ScopeFilter       string                 `json:"scopeFilter"`
	ItemIDFilter      string                 `json:"itemIdFilter"`
	Rows              []InventorySnapshotRow `json:"rows"`
}

type MasterItemUpsertInput struct {
	ID                string `json:"id"`
	ItemNumber        string `json:"itemNumber"`
	Description       string `json:"description"`
	ManufacturerKey   string `json:"manufacturerKey"`
	CategoryKey       string `json:"categoryKey"`
	DefaultSupplierID string `json:"defaultSupplierId"`
	Note              string `json:"note"`
	LifecycleStatus   string `json:"lifecycleStatus"`
}

type SupplierRecord struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ContactName  string `json:"contactName"`
	ContactEmail string `json:"contactEmail"`
}

type SupplierList struct {
	Rows []SupplierRecord `json:"rows"`
}

type SupplierUpsertInput struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ContactName  string `json:"contactName"`
	ContactEmail string `json:"contactEmail"`
}

type AliasList struct {
	Rows []SupplierAliasSummary `json:"rows"`
}

type AliasUpsertInput struct {
	ID                 string `json:"id"`
	ItemID             string `json:"itemId"`
	SupplierID         string `json:"supplierId"`
	SupplierItemNumber string `json:"supplierItemNumber"`
	UnitsPerOrder      int    `json:"unitsPerOrder"`
}

type DeviceRecord struct {
	ID         string `json:"id"`
	DeviceKey  string `json:"deviceKey"`
	Name       string `json:"name"`
	DeviceType string `json:"deviceType"`
	Status     string `json:"status"`
}

type DeviceList struct {
	Rows []DeviceRecord `json:"rows"`
}

type DeviceUpsertInput struct {
	ID         string `json:"id"`
	DeviceKey  string `json:"deviceKey"`
	Name       string `json:"name"`
	DeviceType string `json:"deviceType"`
	Status     string `json:"status"`
}

type DeviceScopeRecord struct {
	ID                 string `json:"id"`
	DeviceID           string `json:"deviceId"`
	DeviceKey          string `json:"deviceKey"`
	ParentScopeID      string `json:"parentScopeId"`
	ParentScopeKey     string `json:"parentScopeKey"`
	SystemKey          string `json:"systemKey"`
	SystemName         string `json:"systemName"`
	ScopeKey           string `json:"scopeKey"`
	ScopeName          string `json:"scopeName"`
	ScopeType          string `json:"scopeType"`
	OwnerDepartmentKey string `json:"ownerDepartmentKey"`
	Status             string `json:"status"`
}

type DeviceScopeList struct {
	Rows []DeviceScopeRecord `json:"rows"`
}

type DeviceScopeUpsertInput struct {
	ID                 string `json:"id"`
	DeviceID           string `json:"deviceId"`
	DeviceKey          string `json:"deviceKey"`
	ParentScopeID      string `json:"parentScopeId"`
	SystemKey          string `json:"systemKey"`
	ScopeKey           string `json:"scopeKey"`
	ScopeName          string `json:"scopeName"`
	ScopeType          string `json:"scopeType"`
	OwnerDepartmentKey string `json:"ownerDepartmentKey"`
	Status             string `json:"status"`
}

type ScopeSystemRecord struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	InUseCount  int    `json:"inUseCount"`
}

type ScopeSystemList struct {
	Rows []ScopeSystemRecord `json:"rows"`
}

type ScopeSystemUpsertInput struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type LocationUpsertInput struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	LocationType string `json:"locationType"`
	IsActive     bool   `json:"isActive"`
}

// --- Additional spec 042401 models ---

// ItemFlow: per-item chronological inventory movement history
type ItemFlowEntry struct {
	Date           string `json:"date"`
	EventType      string `json:"eventType"`
	QuantityDelta  int    `json:"quantityDelta"`
	RunningBalance int    `json:"runningBalance"`
	SourceType     string `json:"sourceType"`
	SourceRef      string `json:"sourceRef"`
	Note           string `json:"note"`
	LocationCode   string `json:"locationCode"`
}

type ItemFlowList struct {
	ItemID     string          `json:"itemId"`
	ItemNumber string          `json:"itemNumber"`
	Rows       []ItemFlowEntry `json:"rows"`
}

// ScopeOverview: scope tree with summary counts
type ScopeOverviewRow struct {
	DeviceKey         string `json:"deviceKey"`
	DeviceName        string `json:"deviceName"`
	ScopeID           string `json:"scopeId"`
	ScopeKey          string `json:"scopeKey"`
	ScopeName         string `json:"scopeName"`
	ScopeType         string `json:"scopeType"`
	ParentScopeID     string `json:"parentScopeId"`
	Status            string `json:"status"`
	PlannedStartAt    string `json:"plannedStartAt"`
	RequirementsCount int    `json:"requirementsCount"`
	ReservationsCount int    `json:"reservationsCount"`
	ShortageItemCount int    `json:"shortageItemCount"`
	OwnerDepartment   string `json:"ownerDepartment"`
}

type ScopeOverviewList struct {
	Rows []ScopeOverviewRow `json:"rows"`
}

// ShortageTimeline: shortage broken down by scope start date
type DelayedArrival struct {
	ExpectedDate        string `json:"expectedDate"`
	Quantity            int    `json:"quantity"`
	PurchaseOrderNumber string `json:"purchaseOrderNumber"`
	PurchaseOrderLineID string `json:"purchaseOrderLineId"`
}

type ShortageTimelineEntry struct {
	ItemID           string           `json:"itemId"`
	ItemNumber       string           `json:"itemNumber"`
	Manufacturer     string           `json:"manufacturer"`
	Description      string           `json:"description"`
	RequiredQuantity int              `json:"requiredQuantity"`
	AvailableByStart int              `json:"availableByStart"`
	ShortageAtStart  int              `json:"shortageAtStart"`
	DelayedArrivals  []DelayedArrival `json:"delayedArrivals"`
}

type ShortageTimeline struct {
	Device         string                  `json:"device"`
	Scope          string                  `json:"scope"`
	PlannedStartAt string                  `json:"plannedStartAt"`
	Rows           []ShortageTimelineEntry `json:"rows"`
}

// Enhanced shortage with procurement pipeline info
type EnhancedShortageRow struct {
	Device                     string   `json:"device"`
	Scope                      string   `json:"scope"`
	Manufacturer               string   `json:"manufacturer"`
	ItemNumber                 string   `json:"itemNumber"`
	Description                string   `json:"description"`
	ItemID                     string   `json:"itemId"`
	RequiredQuantity           int      `json:"requiredQuantity"`
	ReservedQuantity           int      `json:"reservedQuantity"`
	AvailableQuantity          int      `json:"availableQuantity"`
	RawShortage                int      `json:"rawShortage"`
	InRequestFlowQuantity      int      `json:"inRequestFlowQuantity"`
	OrderedQuantity            int      `json:"orderedQuantity"`
	ReceivedQuantity           int      `json:"receivedQuantity"`
	ActionableShortage         int      `json:"actionableShortage"`
	RelatedProcurementRequests []string `json:"relatedProcurementRequests"`
}

type EnhancedShortageList struct {
	CoverageRule string                `json:"coverageRule"`
	Rows         []EnhancedShortageRow `json:"rows"`
}

// Bulk reservation from requirements
type StockAllocation struct {
	LocationCode string `json:"locationCode"`
	Quantity     int    `json:"quantity"`
}

type OrderAllocation struct {
	PurchaseOrderLineID string `json:"purchaseOrderLineId"`
	PurchaseOrderNumber string `json:"purchaseOrderNumber"`
	ExpectedArrival     string `json:"expectedArrival"`
	Quantity            int    `json:"quantity"`
}

type BulkReservationPreviewRow struct {
	ItemID             string            `json:"itemId"`
	ItemNumber         string            `json:"itemNumber"`
	Manufacturer       string            `json:"manufacturer"`
	Description        string            `json:"description"`
	RequiredQuantity   int               `json:"requiredQuantity"`
	AllocFromStock     int               `json:"allocFromStock"`
	AllocFromStockLocs []StockAllocation `json:"allocFromStockLocs"`
	AllocFromOrders    int               `json:"allocFromOrders"`
	AllocFromOrderLocs []OrderAllocation `json:"allocFromOrderLocs"`
	Unallocated        int               `json:"unallocated"`
}

type BulkReservationPreview struct {
	ScopeID string                      `json:"scopeId"`
	Rows    []BulkReservationPreviewRow `json:"rows"`
}

type BulkReservationConfirmInput struct {
	ScopeID string                      `json:"scopeId"`
	ActorID string                      `json:"actorId"`
	Rows    []BulkReservationConfirmRow `json:"rows"`
}

type BulkReservationConfirmRow struct {
	ItemID           string            `json:"itemId"`
	StockAllocations []StockAllocation `json:"stockAllocations"`
	OrderAllocations []OrderAllocation `json:"orderAllocations"`
	Purpose          string            `json:"purpose"`
	Priority         string            `json:"priority"`
	NeededByAt       string            `json:"neededByAt"`
}

type BulkReservationResult struct {
	Created int      `json:"created"`
	IDs     []string `json:"ids"`
}

// Arrival calendar
type ArrivalCalendarItem struct {
	ItemID              string `json:"itemId"`
	ItemNumber          string `json:"itemNumber"`
	Manufacturer        string `json:"manufacturer"`
	Description         string `json:"description"`
	Quantity            int    `json:"quantity"`
	PurchaseOrderNumber string `json:"purchaseOrderNumber"`
	PurchaseOrderLineID string `json:"purchaseOrderLineId"`
	QuotationNumber     string `json:"quotationNumber"`
	SupplierName        string `json:"supplierName"`
}

type ArrivalCalendarDay struct {
	Date  string                `json:"date"`
	Items []ArrivalCalendarItem `json:"items"`
}

type ArrivalCalendar struct {
	YearMonth string               `json:"yearMonth"`
	Days      []ArrivalCalendarDay `json:"days"`
}

// Item suggest (typeahead)
type ItemSuggestion struct {
	ID           string `json:"id"`
	ItemNumber   string `json:"itemNumber"`
	Description  string `json:"description"`
	Manufacturer string `json:"manufacturer"`
	Category     string `json:"category"`
}

type ItemSuggestionList struct {
	Rows []ItemSuggestion `json:"rows"`
}

type CategorySuggestion struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type CategorySuggestionList struct {
	Rows []CategorySuggestion `json:"rows"`
}

// Requirements CSV import
type RequirementsImportPreviewRow struct {
	RowNumber      int    `json:"rowNumber"`
	DeviceKey      string `json:"deviceKey"`
	ScopeKey       string `json:"scopeKey"`
	ItemNumber     string `json:"itemNumber"`
	Manufacturer   string `json:"manufacturer"`
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	Status         string `json:"status"`
	Message        string `json:"message"`
	ItemID         string `json:"itemId"`
	ScopeID        string `json:"scopeId"`
	ItemRegistered bool   `json:"itemRegistered"`
}

type RequirementsImportPreview struct {
	FileName string                         `json:"fileName"`
	Rows     []RequirementsImportPreviewRow `json:"rows"`
}

type RequirementsImportResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
	Errored int `json:"errored"`
}

// Reservation/Requirements CSV export
type ReservationExportRow struct {
	Device       string `json:"device"`
	Scope        string `json:"scope"`
	Manufacturer string `json:"manufacturer"`
	ItemNumber   string `json:"itemNumber"`
	Description  string `json:"description"`
	Quantity     int    `json:"quantity"`
	Status       string `json:"status"`
	Priority     string `json:"priority"`
	NeededByAt   string `json:"neededByAt"`
	SourceType   string `json:"sourceType"`
}

type RequirementExportRow struct {
	Device       string `json:"device"`
	Scope        string `json:"scope"`
	Manufacturer string `json:"manufacturer"`
	ItemNumber   string `json:"itemNumber"`
	Description  string `json:"description"`
	Quantity     int    `json:"quantity"`
	Note         string `json:"note"`
}
