package httpapi

import (
	"log/slog"
	"net/http"

	"backend/internal/auth"
	"backend/internal/config"
	inventory "backend/internal/inventory"
	ocr "backend/internal/ocr"
	procurement "backend/internal/procurement"
)

func NewRouter(cfg config.Config, logger *slog.Logger, authService *auth.Service, phaseOne *inventory.Service, phaseTwo *procurement.Service, phaseThree *ocr.Service) http.Handler {
	handlers := NewHandlers(cfg, logger, authService, phaseOne, phaseTwo, phaseThree)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("GET /ready", handlers.Ready)
	mux.HandleFunc("GET /api/health", handlers.APIHealth)
	mux.HandleFunc("GET /api/v1/bootstrap", handlers.Bootstrap)
	mux.HandleFunc("GET /api/v1/auth/me", handlers.CurrentSession)
	mux.HandleFunc("POST /api/v1/auth/register", handlers.RegisterUser)
	mux.HandleFunc("GET /api/v1/operator/dashboard", handlers.OperatorDashboard)
	mux.HandleFunc("GET /api/v1/operator/reservations", handlers.ReservationList)
	mux.HandleFunc("GET /api/v1/operator/shortages", handlers.ShortageList)
	mux.HandleFunc("GET /api/v1/operator/shortages/export", handlers.ShortageCSVExport)
	mux.HandleFunc("GET /api/v1/operator/imports", handlers.ImportHistory)
	mux.HandleFunc("POST /api/v1/operator/reservations", handlers.CreateReservation)
	mux.HandleFunc("GET /api/v1/inventory/overview", handlers.InventoryOverview)
	mux.HandleFunc("POST /api/v1/inventory/adjustments", handlers.AdjustInventory)
	mux.HandleFunc("GET /api/v1/admin/master-data", handlers.MasterSummary)
	mux.HandleFunc("GET /api/v1/admin/master-data/export", handlers.MasterDataExport)
	mux.HandleFunc("POST /api/v1/admin/master-data/import", handlers.MasterDataImport)
	mux.HandleFunc("GET /api/v1/admin/users", handlers.Users)
	mux.HandleFunc("GET /api/v1/admin/roles", handlers.Roles)
	mux.HandleFunc("POST /api/v1/admin/users/{id}/approve", handlers.ApproveUser)
	mux.HandleFunc("POST /api/v1/admin/users/{id}/reject", handlers.RejectUser)
	mux.HandleFunc("GET /api/v1/procurement/projects", handlers.ProcurementProjects)
	mux.HandleFunc("POST /api/v1/procurement/projects/refresh", handlers.RefreshProcurementProjects)
	mux.HandleFunc("GET /api/v1/procurement/budget-categories", handlers.ProcurementBudgetCategories)
	mux.HandleFunc("POST /api/v1/procurement/budget-categories/refresh", handlers.RefreshProcurementBudgetCategories)
	mux.HandleFunc("GET /api/v1/procurement/sync-runs", handlers.ProcurementSyncRuns)
	mux.HandleFunc("GET /api/v1/procurement/webhooks/external", handlers.ProcurementWebhookEvents)
	mux.HandleFunc("GET /api/v1/procurement/requests", handlers.ProcurementRequests)
	mux.HandleFunc("POST /api/v1/procurement/requests", handlers.CreateProcurementRequest)
	mux.HandleFunc("GET /api/v1/procurement/requests/", handlers.ProcurementRequestDetail)
	mux.HandleFunc("POST /api/v1/procurement/requests/{id}/submit", handlers.SubmitProcurementRequest)
	mux.HandleFunc("POST /api/v1/procurement/requests/{id}/reconcile", handlers.ReconcileProcurementRequest)
	mux.HandleFunc("POST /api/v1/procurement/webhooks/external", handlers.ProcurementWebhook)
	mux.HandleFunc("GET /api/v1/procurement/ocr-jobs", handlers.OCRJobList)
	mux.HandleFunc("POST /api/v1/procurement/ocr-jobs", handlers.CreateOCRJob)
	mux.HandleFunc("GET /api/v1/procurement/ocr-jobs/{id}", handlers.OCRJobDetail)
	mux.HandleFunc("PATCH /api/v1/procurement/ocr-jobs/{id}/review", handlers.UpdateOCRReview)
	mux.HandleFunc("POST /api/v1/procurement/ocr-jobs/{id}/assist", handlers.AssistOCRLine)
	mux.HandleFunc("POST /api/v1/procurement/ocr-jobs/{id}/create-draft", handlers.CreateOCRProcurementDraft)
	mux.HandleFunc("POST /api/v1/procurement/ocr-jobs/{id}/retry", handlers.RetryOCRJob)
	mux.HandleFunc("PUT /api/v1/procurement/ocr-jobs/{id}/register-item", handlers.RegisterOCRItem)

	var handler http.Handler = mux
	if authService != nil {
		handler = authService.Middleware(handler)
	}
	handler = WithCORS(handler, cfg.HTTP.AllowedOrigins)
	handler = WithAccessLog(handler, logger)
	handler = WithRecover(handler, logger)
	return handler
}
