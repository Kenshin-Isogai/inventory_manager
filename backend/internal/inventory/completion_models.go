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
	ScopeKey           string `json:"scopeKey"`
	ScopeName          string `json:"scopeName"`
	ScopeType          string `json:"scopeType"`
	OwnerDepartmentKey string `json:"ownerDepartmentKey"`
	Status             string `json:"status"`
}

type LocationUpsertInput struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	LocationType string `json:"locationType"`
	IsActive     bool   `json:"isActive"`
}
