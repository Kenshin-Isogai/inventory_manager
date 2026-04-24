package inventory

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) Requirements(ctx context.Context, device, scope string) (RequirementList, error) {
	return s.repo.Requirements(ctx, device, scope)
}

func (s *Service) UpsertRequirement(ctx context.Context, input RequirementUpsertInput) (RequirementSummary, error) {
	if input.DeviceScopeID == "" || input.ItemID == "" || input.Quantity <= 0 {
		return RequirementSummary{}, fmt.Errorf("deviceScopeId, itemId, and positive quantity are required")
	}
	return s.repo.UpsertRequirement(ctx, input)
}

func (s *Service) BatchUpsertRequirements(ctx context.Context, input RequirementBatchUpsertInput) (RequirementBatchUpsertResult, error) {
	if input.DeviceScopeID == "" {
		return RequirementBatchUpsertResult{}, fmt.Errorf("deviceScopeId is required")
	}
	if len(input.Rows) == 0 {
		return RequirementBatchUpsertResult{}, fmt.Errorf("at least one row is required")
	}
	for i, row := range input.Rows {
		if row.ItemID == "" || row.Quantity <= 0 {
			return RequirementBatchUpsertResult{}, fmt.Errorf("row %d: itemId and positive quantity are required", i)
		}
	}
	return s.repo.BatchUpsertRequirements(ctx, input)
}

func (s *Service) ReservationDetail(ctx context.Context, id string) (ReservationDetail, error) {
	if id == "" {
		return ReservationDetail{}, fmt.Errorf("reservation id is required")
	}
	return s.repo.ReservationDetail(ctx, id)
}

func (s *Service) UpdateReservation(ctx context.Context, id string, input ReservationUpdateInput) (ReservationDetail, error) {
	if id == "" {
		return ReservationDetail{}, fmt.Errorf("reservation id is required")
	}
	if input.ItemID == "" || input.DeviceScopeID == "" || input.Quantity <= 0 {
		return ReservationDetail{}, fmt.Errorf("itemId, deviceScopeId, and positive quantity are required")
	}
	return s.repo.UpdateReservation(ctx, id, input)
}

func (s *Service) DeleteReservation(ctx context.Context, id, actorID string) error {
	if id == "" {
		return fmt.Errorf("reservation id is required")
	}
	return s.repo.DeleteReservation(ctx, id, actorID)
}

func (s *Service) AllocateReservation(ctx context.Context, id string, input ReservationActionInput) (ReservationDetail, error) {
	if id == "" || input.LocationCode == "" || input.Quantity <= 0 {
		return ReservationDetail{}, fmt.Errorf("reservation id, locationCode, and positive quantity are required")
	}
	return s.repo.AllocateReservation(ctx, id, input)
}

func (s *Service) ReleaseReservation(ctx context.Context, id string, input ReservationActionInput) (ReservationDetail, error) {
	if id == "" || input.Quantity <= 0 {
		return ReservationDetail{}, fmt.Errorf("reservation id and positive quantity are required")
	}
	return s.repo.ReleaseReservation(ctx, id, input)
}

func (s *Service) FulfillReservation(ctx context.Context, id string, input ReservationActionInput) (ReservationDetail, error) {
	if id == "" {
		return ReservationDetail{}, fmt.Errorf("reservation id is required")
	}
	return s.repo.FulfillReservation(ctx, id, input)
}

func (s *Service) CancelReservation(ctx context.Context, id string, input ReservationActionInput) (ReservationDetail, error) {
	if id == "" {
		return ReservationDetail{}, fmt.Errorf("reservation id is required")
	}
	return s.repo.CancelReservation(ctx, id, input)
}

func (s *Service) UndoReservation(ctx context.Context, id string, input ReservationActionInput) (ReservationDetail, error) {
	if id == "" {
		return ReservationDetail{}, fmt.Errorf("reservation id is required")
	}
	return s.repo.UndoReservation(ctx, id, input)
}

func (s *Service) InventoryItems(ctx context.Context) (InventoryItemList, error) {
	return s.repo.InventoryItems(ctx)
}

func (s *Service) InventoryLocations(ctx context.Context) (LocationList, error) {
	return s.repo.InventoryLocations(ctx)
}

func (s *Service) InventoryEvents(ctx context.Context) (InventoryEventList, error) {
	return s.repo.InventoryEvents(ctx)
}

func (s *Service) InventorySnapshot(ctx context.Context, device, scope, itemID string) (InventorySnapshot, error) {
	return s.repo.InventorySnapshot(ctx, device, scope, itemID)
}

func (s *Service) ReceiveInventory(ctx context.Context, input InventoryReceiveInput) (InventoryEventEntry, error) {
	if input.ItemID == "" || input.LocationCode == "" || input.Quantity <= 0 {
		return InventoryEventEntry{}, fmt.Errorf("itemId, locationCode, and positive quantity are required")
	}
	return s.repo.ReceiveInventory(ctx, input)
}

func (s *Service) MoveInventory(ctx context.Context, input InventoryMoveInput) (InventoryEventEntry, error) {
	if input.ItemID == "" || input.FromLocationCode == "" || input.ToLocationCode == "" || input.Quantity <= 0 {
		return InventoryEventEntry{}, fmt.Errorf("itemId, fromLocationCode, toLocationCode, and positive quantity are required")
	}
	return s.repo.MoveInventory(ctx, input)
}

func (s *Service) UndoInventoryEvent(ctx context.Context, id string, input InventoryUndoInput) (InventoryEventEntry, error) {
	if id == "" {
		return InventoryEventEntry{}, fmt.Errorf("inventory event id is required")
	}
	return s.repo.UndoInventoryEvent(ctx, id, input)
}

func (s *Service) Arrivals(ctx context.Context) (ArrivalList, error) {
	return s.repo.Arrivals(ctx)
}

func (s *Service) CreateReceipt(ctx context.Context, input ReceiptCreateInput) (ReceiptSummary, error) {
	if len(input.Lines) == 0 {
		return ReceiptSummary{}, fmt.Errorf("at least one receipt line is required")
	}
	return s.repo.CreateReceipt(ctx, input)
}

func (s *Service) ImportPreview(ctx context.Context, importType, fileName string, records []byte) (ImportPreviewResult, error) {
	if importType == "" || fileName == "" {
		return ImportPreviewResult{}, fmt.Errorf("importType and fileName are required")
	}
	return s.repo.ImportPreview(ctx, importType, fileName, records)
}

func (s *Service) ImportDetail(ctx context.Context, id string) (ImportDetail, error) {
	if id == "" {
		return ImportDetail{}, fmt.Errorf("import id is required")
	}
	return s.repo.ImportDetail(ctx, id)
}

func (s *Service) UndoImport(ctx context.Context, id, actor string) (ImportDetail, error) {
	if id == "" {
		return ImportDetail{}, fmt.Errorf("import id is required")
	}
	return s.repo.UndoImport(ctx, id, actor)
}

func (s *Service) MasterItems(ctx context.Context) (MasterItemList, error) {
	return s.repo.MasterItems(ctx)
}

func (s *Service) UpsertMasterItem(ctx context.Context, input MasterItemUpsertInput) (MasterItemRecord, error) {
	if input.ItemNumber == "" || input.Description == "" || input.ManufacturerKey == "" || input.CategoryKey == "" {
		return MasterItemRecord{}, fmt.Errorf("itemNumber, description, manufacturerKey, and categoryKey are required")
	}
	return s.repo.UpsertMasterItem(ctx, input)
}

func (s *Service) MasterItemDetail(ctx context.Context, id string) (MasterItemRecord, error) {
	if id == "" {
		return MasterItemRecord{}, fmt.Errorf("item id is required")
	}
	return s.repo.MasterItemDetail(ctx, id)
}

func (s *Service) DeleteMasterItem(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("item id is required")
	}
	return s.repo.DeleteMasterItem(ctx, id)
}

func (s *Service) Suppliers(ctx context.Context) (SupplierList, error) {
	return s.repo.Suppliers(ctx)
}

func (s *Service) UpsertSupplier(ctx context.Context, input SupplierUpsertInput) (SupplierRecord, error) {
	if input.Name == "" {
		return SupplierRecord{}, fmt.Errorf("name is required")
	}
	return s.repo.UpsertSupplier(ctx, input)
}

func (s *Service) Aliases(ctx context.Context) (AliasList, error) {
	return s.repo.Aliases(ctx)
}

func (s *Service) UpsertAlias(ctx context.Context, input AliasUpsertInput) (SupplierAliasSummary, error) {
	if input.ItemID == "" || input.SupplierID == "" || input.SupplierItemNumber == "" || input.UnitsPerOrder <= 0 {
		return SupplierAliasSummary{}, fmt.Errorf("itemId, supplierId, supplierItemNumber, and positive unitsPerOrder are required")
	}
	return s.repo.UpsertAlias(ctx, input)
}

func (s *Service) Devices(ctx context.Context) (DeviceList, error) {
	return s.repo.Devices(ctx)
}

func (s *Service) UpsertDevice(ctx context.Context, input DeviceUpsertInput) (DeviceRecord, error) {
	if input.DeviceKey == "" || input.Name == "" {
		return DeviceRecord{}, fmt.Errorf("deviceKey and name are required")
	}
	return s.repo.UpsertDevice(ctx, input)
}

func (s *Service) DeviceScopes(ctx context.Context) (DeviceScopeList, error) {
	return s.repo.DeviceScopes(ctx)
}

func (s *Service) ScopeSystems(ctx context.Context) (ScopeSystemList, error) {
	return s.repo.ScopeSystems(ctx)
}

func (s *Service) UpsertScopeSystem(ctx context.Context, input ScopeSystemUpsertInput) (ScopeSystemRecord, error) {
	if input.Key == "" || input.Name == "" {
		return ScopeSystemRecord{}, fmt.Errorf("key and name are required")
	}
	return s.repo.UpsertScopeSystem(ctx, input)
}

func (s *Service) DeleteScopeSystem(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("scope system key is required")
	}
	return s.repo.DeleteScopeSystem(ctx, key)
}

func (s *Service) UpsertDeviceScope(ctx context.Context, input DeviceScopeUpsertInput) (DeviceScopeRecord, error) {
	if input.DeviceKey == "" || input.ScopeKey == "" {
		return DeviceScopeRecord{}, fmt.Errorf("deviceKey and scopeKey are required")
	}
	scopeType := normalizeDeviceScopeType(defaultString(input.ScopeType, "assembly"))
	if !isValidDeviceScopeType(scopeType) {
		return DeviceScopeRecord{}, fmt.Errorf("unsupported scopeType: %s", input.ScopeType)
	}
	if scopeType == "system" {
		if strings.TrimSpace(input.ParentScopeID) != "" {
			return DeviceScopeRecord{}, fmt.Errorf("system scopes cannot have a parentScopeId")
		}
		scopeKey := normalizeLookupKey(input.ScopeKey)
		systemKey := normalizeLookupKey(defaultString(input.SystemKey, input.ScopeKey))
		if scopeKey == "" || systemKey == "" {
			return DeviceScopeRecord{}, fmt.Errorf("system scopes require a valid systemKey")
		}
		if scopeKey != systemKey {
			return DeviceScopeRecord{}, fmt.Errorf("system scope key must match systemKey")
		}
	} else if strings.TrimSpace(input.ParentScopeID) == "" {
		return DeviceScopeRecord{}, fmt.Errorf("non-system scopes require a parentScopeId")
	}
	input.ScopeType = scopeType
	return s.repo.UpsertDeviceScope(ctx, input)
}

func (s *Service) UpsertLocation(ctx context.Context, input LocationUpsertInput) (LocationSummary, error) {
	if input.Code == "" || input.Name == "" {
		return LocationSummary{}, fmt.Errorf("code and name are required")
	}
	return s.repo.UpsertLocation(ctx, input)
}
