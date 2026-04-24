package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/httpapi"
	inventory "backend/internal/inventory"
	ocr "backend/internal/ocr"
	"backend/internal/platform/database"
	"backend/internal/platform/logging"
	"backend/internal/platform/storage"
	procurement "backend/internal/procurement"
)

type App struct {
	cfg       config.Config
	logger    *slog.Logger
	server    *http.Server
	db        *sql.DB
	ocrService *ocr.Service
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	logger := logging.NewJSONLogger(cfg.Logging.Level)

	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}

	store, err := storage.New(cfg.Storage)
	if err != nil {
		db.Close()
		return nil, err
	}

	phaseOneService := inventory.NewService(inventory.NewRepository(db))
	authService, err := auth.NewService(cfg.Auth, auth.NewRepository(db))
	if err != nil {
		db.Close()
		return nil, err
	}
	phaseTwoService := procurement.NewService(
		procurement.NewRepository(db),
		store,
		procurement.NewMockDispatcher(),
		procurement.NewMockSyncAdapter(cfg.Integration.ProcurementWebhookSecret),
	)
	ocrProvider, err := buildOCRProvider(cfg)
	if err != nil {
		db.Close()
		return nil, err
	}
	phaseThreeService := ocr.NewService(ocr.NewRepository(db), store, ocrProvider, phaseTwoService)
	router := httpapi.NewRouter(cfg, logger, authService, phaseOneService, phaseTwoService, phaseThreeService, db.PingContext)

	server := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	return &App{
		cfg:        cfg,
		logger:     logger,
		server:     server,
		db:         db,
		ocrService: phaseThreeService,
	}, nil
}

func buildOCRProvider(cfg config.Config) (ocr.Provider, error) {
	switch strings.TrimSpace(strings.ToLower(cfg.OCR.Provider)) {
	case "vertex_ai", "vertex-ai":
		return ocr.NewVertexAIProvider(cfg.OCR)
	case "mock", "":
		return ocr.NewMockProvider("mock"), nil
	default:
		return nil, fmt.Errorf("unsupported OCR provider: %s", cfg.OCR.Provider)
	}
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("starting http server",
			slog.String("addr", a.server.Addr),
			slog.String("env", a.cfg.App.Env),
			slog.String("mode", a.cfg.App.Mode),
		)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
		close(errCh)
	}()

	go a.runOCRCleanupLoop(ctx)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.WriteTimeout)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		if err := a.db.Close(); err != nil {
			return fmt.Errorf("close db: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}
}

func (a *App) runOCRCleanupLoop(ctx context.Context) {
	const interval = 1 * time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	a.logger.Info("ocr cleanup loop started", slog.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := a.ocrService.CleanupExpiredJobs(ctx)
			if err != nil {
				a.logger.Error("ocr cleanup failed", slog.String("error", err.Error()))
			} else if count > 0 {
				a.logger.Info("ocr cleanup completed", slog.Int("deleted", count))
			}
		}
	}
}
