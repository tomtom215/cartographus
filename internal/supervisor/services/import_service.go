// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package services

import (
	"context"
	"fmt"

	"github.com/tomtom215/cartographus/internal/logging"
)

// ImporterInterface defines the interface for the Tautulli importer.
// This interface abstracts the importer's lifecycle for supervisor integration.
type ImporterInterface interface {
	// Import performs the import operation.
	// It blocks until import is complete or context is canceled.
	Import(ctx context.Context) (interface{}, error)

	// IsRunning returns whether an import is currently in progress.
	IsRunning() bool

	// Stop cancels a running import operation.
	Stop() error
}

// ImportService wraps the Tautulli importer as a supervised service.
//
// The service supports two modes:
//  1. AutoStart mode: Runs import automatically when the service starts
//  2. On-demand mode: Waits for external trigger via API endpoint
//
// When AutoStart is enabled, the service will:
//  1. Start the import operation
//  2. Wait for completion or context cancellation
//  3. Return ctx.Err() to indicate normal shutdown
//
// When AutoStart is disabled, the service will:
//  1. Wait for context cancellation
//  2. Return ctx.Err() to indicate normal shutdown
type ImportService struct {
	importer  ImporterInterface
	name      string
	autoStart bool
}

// NewImportService creates a new import service wrapper.
//
// Parameters:
//   - importer: The Tautulli importer instance
//   - autoStart: If true, starts import automatically on service start
//
// Example usage:
//
//	importer := tautulli_import.NewImporter(cfg, publisher, progress)
//	svc := services.NewImportService(importer, true)
//	tree.AddMessagingService(svc)
func NewImportService(importer ImporterInterface, autoStart bool) *ImportService {
	return &ImportService{
		importer:  importer,
		name:      "tautulli-import",
		autoStart: autoStart,
	}
}

// Serve implements suture.Service.
//
// If autoStart is true:
//   - Starts the import operation immediately
//   - Blocks until import completes or context is canceled
//   - Returns any import errors
//
// If autoStart is false:
//   - Blocks until context is canceled
//   - Import must be triggered externally via API
func (s *ImportService) Serve(ctx context.Context) error {
	if s.autoStart {
		logging.Info().Msg("Starting automatic Tautulli database import")
		stats, err := s.importer.Import(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// Context canceled - normal shutdown
				logging.Info().Msg("Import canceled due to shutdown")
				return ctx.Err()
			}
			return fmt.Errorf("import failed: %w", err)
		}
		logging.Info().Interface("stats", stats).Msg("Import completed")

		// Wait for shutdown after import completes
		<-ctx.Done()
		return ctx.Err()
	}

	// On-demand mode - just wait for shutdown
	logging.Info().Msg("Import service started (on-demand mode - use API to trigger)")
	<-ctx.Done()

	// If import is running, stop it
	if s.importer.IsRunning() {
		logging.Info().Msg("Stopping running import due to shutdown")
		if err := s.importer.Stop(); err != nil {
			logging.Warn().Err(err).Msg("Failed to stop import")
		}
	}

	return ctx.Err()
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (s *ImportService) String() string {
	return s.name
}

// Importer returns the underlying importer instance.
// This is useful for triggering imports via API handlers.
func (s *ImportService) Importer() ImporterInterface {
	return s.importer
}
