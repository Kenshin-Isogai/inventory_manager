package inventory

import (
	"context"
	"fmt"
)

// ItemFlow returns chronological inventory events for a single item.
func (s *Service) ItemFlow(ctx context.Context, itemID string) (ItemFlowList, error) {
	if itemID == "" {
		return ItemFlowList{}, fmt.Errorf("item id is required")
	}
	return s.repo.ItemFlow(ctx, itemID)
}

// ScopeOverview returns scope tree with summary counts.
func (s *Service) ScopeOverview(ctx context.Context, device string) (ScopeOverviewList, error) {
	return s.repo.ScopeOverview(ctx, device)
}

// ShortageTimeline returns shortage broken down by scope start date timing.
func (s *Service) ShortageTimeline(ctx context.Context, device, scope string) (ShortageTimeline, error) {
	if device == "" || scope == "" {
		return ShortageTimeline{}, fmt.Errorf("device and scope are required")
	}
	return s.repo.ShortageTimeline(ctx, device, scope)
}

// EnhancedShortages returns shortages with procurement pipeline info.
func (s *Service) EnhancedShortages(ctx context.Context, device, scope, coverageRule string) (EnhancedShortageList, error) {
	return s.repo.EnhancedShortages(ctx, device, scope, coverageRule)
}

// ReservationsExportCSV returns reservation data as CSV bytes.
func (s *Service) ReservationsExportCSV(ctx context.Context, device, scope string) (string, error) {
	return s.repo.ReservationsExportCSV(ctx, device, scope)
}

// RequirementsExportCSV returns requirements data as CSV bytes.
func (s *Service) RequirementsExportCSV(ctx context.Context, device, scope string) (string, error) {
	return s.repo.RequirementsExportCSV(ctx, device, scope)
}

// RequirementsImportPreview previews a requirements CSV import.
func (s *Service) RequirementsImportPreview(ctx context.Context, fileName string, data []byte) (RequirementsImportPreview, error) {
	if fileName == "" {
		return RequirementsImportPreview{}, fmt.Errorf("fileName is required")
	}
	return s.repo.RequirementsImportPreview(ctx, fileName, data)
}

// RequirementsImportApply applies a previewed requirements CSV import.
func (s *Service) RequirementsImportApply(ctx context.Context, fileName string, data []byte) (RequirementsImportResult, error) {
	if fileName == "" {
		return RequirementsImportResult{}, fmt.Errorf("fileName is required")
	}
	return s.repo.RequirementsImportApply(ctx, fileName, data)
}

// BulkReservationPreview generates a preview of bulk reservations from requirements.
func (s *Service) BulkReservationPreview(ctx context.Context, scopeID string) (BulkReservationPreview, error) {
	if scopeID == "" {
		return BulkReservationPreview{}, fmt.Errorf("scope id is required")
	}
	return s.repo.BulkReservationPreview(ctx, scopeID)
}

// BulkReservationConfirm creates reservations from a confirmed bulk preview.
func (s *Service) BulkReservationConfirm(ctx context.Context, input BulkReservationConfirmInput) (BulkReservationResult, error) {
	if input.ScopeID == "" || len(input.Rows) == 0 {
		return BulkReservationResult{}, fmt.Errorf("scopeId and at least one row are required")
	}
	return s.repo.BulkReservationConfirm(ctx, input)
}

// ArrivalCalendar returns expected arrivals grouped by date for a given month.
func (s *Service) ArrivalCalendar(ctx context.Context, yearMonth string) (ArrivalCalendar, error) {
	if yearMonth == "" {
		return ArrivalCalendar{}, fmt.Errorf("yearMonth is required (YYYY-MM format)")
	}
	return s.repo.ArrivalCalendar(ctx, yearMonth)
}

// ItemSuggest returns items matching a search query for typeahead.
func (s *Service) ItemSuggest(ctx context.Context, query string) (ItemSuggestionList, error) {
	if query == "" {
		return ItemSuggestionList{Rows: []ItemSuggestion{}}, nil
	}
	return s.repo.ItemSuggest(ctx, query)
}

// CategorySuggest returns categories matching a search query for typeahead.
func (s *Service) CategorySuggest(ctx context.Context, query string) (CategorySuggestionList, error) {
	if query == "" {
		return CategorySuggestionList{Rows: []CategorySuggestion{}}, nil
	}
	return s.repo.CategorySuggest(ctx, query)
}

// InventorySnapshotAtDate returns projected inventory at a future date.
func (s *Service) InventorySnapshotAtDate(ctx context.Context, device, scope, itemID, targetDate string) (InventorySnapshot, error) {
	if targetDate == "" {
		return s.repo.InventorySnapshot(ctx, device, scope, itemID)
	}
	return s.repo.InventorySnapshotAtDate(ctx, device, scope, itemID, targetDate)
}
