// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package main

import (
	"context"
	"net/http"

	"github.com/dgraph-io/badger/v4"
	"github.com/tomtom215/cartographus/internal/api"
	"github.com/tomtom215/cartographus/internal/config"
	tautulliimport "github.com/tomtom215/cartographus/internal/import"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/supervisor"
	"github.com/tomtom215/cartographus/internal/supervisor/services"
)

// importerAdapter wraps tautulliimport.Importer to implement services.ImporterInterface.
// This adapter is necessary because the Importer returns *ImportStats while the
// interface expects interface{} for flexibility.
type importerAdapter struct {
	importer *tautulliimport.Importer
}

// Import implements services.ImporterInterface.
func (a *importerAdapter) Import(ctx context.Context) (interface{}, error) {
	return a.importer.Import(ctx)
}

// IsRunning implements services.ImporterInterface.
func (a *importerAdapter) IsRunning() bool {
	return a.importer.IsRunning()
}

// Stop implements services.ImporterInterface.
func (a *importerAdapter) Stop() error {
	return a.importer.Stop()
}

// ImportComponents holds import-related components for lifecycle management.
type ImportComponents struct {
	importer *tautulliimport.Importer
	progress tautulliimport.ProgressTracker
	service  *services.ImportService
	handlers *api.ImportHandlers
}

// InitImport initializes the Tautulli database import functionality.
// It creates the importer, progress tracker, supervisor service, and API handlers.
//
// Parameters:
//   - cfg: Application configuration with import settings
//   - natsComponents: NATS components providing the event publisher
//   - tree: Supervisor tree for adding the import service
//   - router: API router for registering import endpoints
//
// Returns nil if import is disabled in configuration.
func InitImport(cfg *config.Config, natsComponents *NATSComponents, tree *supervisor.SupervisorTree, router *api.Router) (*ImportComponents, error) {
	if !cfg.Import.Enabled {
		logging.Info().Msg("Tautulli database import disabled (IMPORT_ENABLED=false)")
		return nil, nil
	}

	if natsComponents == nil {
		logging.Info().Msg("Tautulli database import requires NATS to be enabled")
		return nil, nil
	}

	logging.Info().Msg("Initializing Tautulli database import...")

	components := &ImportComponents{}

	// Create progress tracker
	// Use BadgerDB if WAL is enabled for persistent progress across restarts
	// Otherwise fall back to in-memory progress
	var progress tautulliimport.ProgressTracker
	if badgerDB := natsComponents.BadgerDB(); badgerDB != nil {
		if db, ok := badgerDB.(*badger.DB); ok {
			progress = tautulliimport.NewBadgerProgress(db)
			components.progress = progress
			logging.Info().Msg("Import progress tracker created (BadgerDB - persistent)")
		} else {
			progress = tautulliimport.NewInMemoryProgress()
			components.progress = progress
			logging.Warn().Msg("Import progress tracker created (in-memory - BadgerDB type assertion failed)")
		}
	} else {
		progress = tautulliimport.NewInMemoryProgress()
		components.progress = progress
		logging.Info().Msg("Import progress tracker created (in-memory - WAL not enabled)")
	}

	// Get event publisher from NATS components
	// The publisher implements tautulliimport.EventPublisher interface
	publisher := natsComponents.publisher

	// Create the importer
	importer := tautulliimport.NewImporter(&cfg.Import, publisher, progress)
	components.importer = importer
	logging.Info().
		Str("db_path", cfg.Import.DBPath).
		Int("batch_size", cfg.Import.BatchSize).
		Bool("dry_run", cfg.Import.DryRun).
		Msg("Importer created")

	// Create import service for supervisor
	// Use adapter to satisfy ImporterInterface which expects interface{} return type
	adapter := &importerAdapter{importer: importer}
	importService := services.NewImportService(adapter, cfg.Import.AutoStart)
	components.service = importService

	// Add to supervisor tree (messaging layer)
	tree.AddMessagingService(importService)
	if cfg.Import.AutoStart {
		logging.Info().Msg("Import service added to supervisor tree (auto-start enabled)")
	} else {
		logging.Info().Msg("Import service added to supervisor tree (on-demand mode)")
	}

	// Create API handlers
	handlers := api.NewImportHandlers(importer, progress)
	components.handlers = handlers

	// Register import routes via the route registrar
	router.SetImportRouteRegistrar(func(mux *http.ServeMux) {
		router.RegisterImportRoutes(mux, handlers)
		logging.Info().Msg("Import API routes registered")
	})

	logging.Info().Msg("Tautulli database import initialized successfully")
	return components, nil
}

// Importer returns the underlying importer instance.
func (c *ImportComponents) Importer() *tautulliimport.Importer {
	if c == nil {
		return nil
	}
	return c.importer
}

// Progress returns the progress tracker.
func (c *ImportComponents) Progress() tautulliimport.ProgressTracker {
	if c == nil {
		return nil
	}
	return c.progress
}
