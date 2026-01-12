// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package services

import (
	"context"
	"fmt"
)

// EmbyStartStopManager interface matches the Emby manager lifecycle
type EmbyStartStopManager interface {
	Start(ctx context.Context) error
	Stop() error
}

// EmbyService wraps the Emby manager as a supervised service.
//
// It adapts the Start/Stop lifecycle pattern to suture's Serve pattern:
//  1. Calls Start(ctx) to begin the Emby manager
//  2. Waits for context cancellation
//  3. Calls Stop() for graceful shutdown
type EmbyService struct {
	manager EmbyStartStopManager
	name    string
}

// NewEmbyService creates a new Emby service wrapper.
//
// Example usage:
//
//	embyManager := sync.NewEmbyManager(cfg.Emby, wsHub)
//	svc := services.NewEmbyService(embyManager)
//	tree.AddMessagingService(svc)
func NewEmbyService(manager EmbyStartStopManager) *EmbyService {
	return &EmbyService{
		manager: manager,
		name:    "emby-manager",
	}
}

// NewEmbyServiceWithName creates a new Emby service wrapper with a custom name.
// This is useful when running multiple Emby servers to differentiate them in logs.
//
// Example usage:
//
//	embyManager := sync.NewEmbyManager(&cfg, wsHub, userResolver)
//	svc := services.NewEmbyServiceWithName(embyManager, "emby-home")
//	tree.AddMessagingService(svc)
func NewEmbyServiceWithName(manager EmbyStartStopManager, name string) *EmbyService {
	return &EmbyService{
		manager: manager,
		name:    name,
	}
}

// Serve implements suture.Service.
//
// This method:
//  1. Starts the Emby manager (which spawns its internal goroutines)
//  2. Blocks until the context is canceled
//  3. Stops the Emby manager (which waits for its goroutines to complete)
//
// If Start() fails, the error is returned immediately, causing suture to
// restart the service according to its backoff policy.
func (s *EmbyService) Serve(ctx context.Context) error {
	// Start the manager
	if err := s.manager.Start(ctx); err != nil {
		return fmt.Errorf("emby manager start failed: %w", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop the manager
	if err := s.manager.Stop(); err != nil {
		return fmt.Errorf("emby manager stop failed: %w", err)
	}

	return ctx.Err()
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (s *EmbyService) String() string {
	return s.name
}
