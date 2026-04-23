package inventory

type DashboardMetric struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Delta string `json:"delta"`
}

type DashboardData struct {
	GeneratedAt string            `json:"generatedAt"`
	Metrics     []DashboardMetric `json:"metrics"`
	Alerts      []string          `json:"alerts"`
}

type ReservationSummary struct {
	ID          string `json:"id"`
	ItemNumber  string `json:"itemNumber"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	Device      string `json:"device"`
	Scope       string `json:"scope"`
	Status      string `json:"status"`
}

type ReservationList struct {
	Rows []ReservationSummary `json:"rows"`
}

type InventoryBalance struct {
	ItemID            string `json:"itemId"`
	ItemNumber        string `json:"itemNumber"`
	Description       string `json:"description"`
	Manufacturer      string `json:"manufacturer"`
	Category          string `json:"category"`
	LocationCode      string `json:"locationCode"`
	OnHandQuantity    int    `json:"onHandQuantity"`
	ReservedQuantity  int    `json:"reservedQuantity"`
	AvailableQuantity int    `json:"availableQuantity"`
}

type InventoryOverview struct {
	Balances []InventoryBalance `json:"balances"`
}

type ShortageRow struct {
	Device       string `json:"device"`
	Scope        string `json:"scope"`
	Manufacturer string `json:"manufacturer"`
	ItemNumber   string `json:"itemNumber"`
	Description  string `json:"description"`
	Quantity     int    `json:"quantity"`
}

type ShortageList struct {
	Rows []ShortageRow `json:"rows"`
}

type ImportJob struct {
	ID         string `json:"id"`
	ImportType string `json:"importType"`
	Status     string `json:"status"`
	FileName   string `json:"fileName"`
	Summary    string `json:"summary"`
	CreatedAt  string `json:"createdAt"`
}

type ImportHistory struct {
	Rows []ImportJob `json:"rows"`
}

type MasterDataSummary struct {
	ItemCount         int                    `json:"itemCount"`
	SupplierCount     int                    `json:"supplierCount"`
	AliasCount        int                    `json:"aliasCount"`
	Manufacturers     []string               `json:"manufacturers"`
	Categories        []CategorySummary      `json:"categories"`
	Suppliers         []SupplierSummary      `json:"suppliers"`
	Aliases           []SupplierAliasSummary `json:"aliases"`
	RecentItems       []MasterItem           `json:"recentItems"`
	RecentImportFiles []string               `json:"recentImportFiles"`
}

type MasterItem struct {
	ItemNumber   string `json:"itemNumber"`
	Description  string `json:"description"`
	Manufacturer string `json:"manufacturer"`
	Category     string `json:"category"`
	Supplier     string `json:"supplier"`
}

type CategorySummary struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type SupplierSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SupplierAliasSummary struct {
	ID                  string `json:"id"`
	SupplierID          string `json:"supplierId"`
	SupplierName        string `json:"supplierName"`
	ItemID              string `json:"itemId"`
	CanonicalItemNumber string `json:"canonicalItemNumber"`
	SupplierItemNumber  string `json:"supplierItemNumber"`
	UnitsPerOrder       int    `json:"unitsPerOrder"`
}

type ReservationCreateInput struct {
	ItemID        string `json:"itemId"`
	DeviceScopeID string `json:"deviceScopeId"`
	Quantity      int    `json:"quantity"`
	RequestedBy   string `json:"requestedBy"`
	Note          string `json:"note"`
}

type InventoryAdjustInput struct {
	ItemID        string `json:"itemId"`
	LocationCode  string `json:"locationCode"`
	QuantityDelta int    `json:"quantityDelta"`
	DeviceScopeID string `json:"deviceScopeId"`
	Note          string `json:"note"`
}
