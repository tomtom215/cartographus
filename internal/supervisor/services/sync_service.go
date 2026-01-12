// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package services

import (
	"context"
	"fmt"
)

// StartStopManager interface matches the existing internal/sync.Manager lifecycle.
//
// This interface abstracts the sync manager's Start/Stop pattern, allowing the
// SyncService wrapper to adapt it to suture's Serve pattern without modifying
// the existing manager code.
//
// The interface is satisfied by *sync.Manager from internal/sync/manager.go:
//   - Start(ctx context.Context) error - lines 123-153
//   - Stop() error - lines 205-229
type StartStopManager interface {
	Start(ctx context.Context) error
	Stop() error
}

// SyncService wraps the sync manager as a supervised service.
//
// It adapts the Start/Stop lifecycle pattern to suture's Serve pattern:
//  1. Calls Start(ctx) to begin the sync manager
//  2. Waits for context cancellation
//  3. Calls Stop() for graceful shutdown
//
// The sync manager handles its own goroutines internally via WaitGroup,
// so this wrapper simply orchestrates the lifecycle transitions.
type SyncService struct {
	manager StartStopManager
	name    string
}

// NewSyncService creates a new sync service wrapper.
//
// Example usage:
//
//	syncManager := sync.NewManager(db, tautulliClient, cfg, wsHub)
//	svc := services.NewSyncService(syncManager)
//	tree.AddMessagingService(svc)
func NewSyncService(manager StartStopManager) *SyncService {
	return &SyncService{
		manager: manager,
		name:    "sync-manager",
	}
}

// Serve implements suture.Service.
//
// This method:
//  1. Starts the sync manager (which spawns its internal goroutines)
//  2. Blocks until the context is canceled
//  3. Stops the sync manager (which waits for its goroutines to complete)
//
// If Start() fails, the error is returned immediately, causing suture to
// restart the service according to its backoff policy.
func (s *SyncService) Serve(ctx context.Context) error {
	// Start the manager - this spawns internal goroutines but returns immediately
	if err := s.manager.Start(ctx); err != nil {
		return fmt.Errorf("sync manager start failed: %w", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop the manager - this blocks until all internal goroutines complete
	// per manager.go:225 (m.wg.Wait())
	if err := s.manager.Stop(); err != nil {
		// Log but don't return the error - we're shutting down anyway
		// and the context error is the primary cause
		return fmt.Errorf("sync manager stop failed: %w", err)
	}

	return ctx.Err()
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (s *SyncService) String() string {
	return s.name
}
