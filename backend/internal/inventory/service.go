package inventory

import (
	"context"
	"fmt"
	"io"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Dashboard(ctx context.Context) (DashboardData, error) {
	return s.repo.Dashboard(ctx)
}

func (s *Service) Reservations(ctx context.Context, device, scope string) (ReservationList, error) {
	return s.repo.Reservations(ctx, device, scope)
}

func (s *Service) InventoryOverview(ctx context.Context) (InventoryOverview, error) {
	return s.repo.InventoryOverview(ctx)
}

func (s *Service) Shortages(ctx context.Context, device, scope string) (ShortageList, error) {
	return s.repo.Shortages(ctx, device, scope)
}

func (s *Service) Imports(ctx context.Context) (ImportHistory, error) {
	return s.repo.Imports(ctx)
}

func (s *Service) MasterSummary(ctx context.Context) (MasterDataSummary, error) {
	return s.repo.MasterSummary(ctx)
}

func (s *Service) ShortageCSV(ctx context.Context, device, scope string) (string, error) {
	return s.repo.ShortageCSV(ctx, device, scope)
}

func (s *Service) ExportMasterCSV(ctx context.Context, exportType string) (string, error) {
	return s.repo.ExportMasterCSV(ctx, exportType)
}

func (s *Service) ImportMasterCSV(ctx context.Context, importType, fileName string, body io.Reader) (ImportJob, error) {
	if fileName == "" {
		return ImportJob{}, fmt.Errorf("file name is required")
	}
	return s.repo.ImportMasterCSV(ctx, importType, fileName, body)
}

func (s *Service) CreateReservation(ctx context.Context, input ReservationCreateInput) error {
	if input.ItemID == "" || input.DeviceScopeID == "" || input.Quantity <= 0 {
		return fmt.Errorf("itemId, deviceScopeId, and positive quantity are required")
	}
	return s.repo.CreateReservation(ctx, input)
}

func (s *Service) AdjustInventory(ctx context.Context, input InventoryAdjustInput) error {
	if input.ItemID == "" || input.LocationCode == "" || input.QuantityDelta == 0 {
		return fmt.Errorf("itemId, locationCode, and non-zero quantityDelta are required")
	}
	return s.repo.AdjustInventory(ctx, input)
}
