// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package services

import (
	"context"
	"fmt"
)

// JellyfinStartStopManager interface matches the Jellyfin manager lifecycle
type JellyfinStartStopManager interface {
	Start(ctx context.Context) error
	Stop() error
}

// JellyfinService wraps the Jellyfin manager as a supervised service.
//
// It adapts the Start/Stop lifecycle pattern to suture's Serve pattern:
//  1. Calls Start(ctx) to begin the Jellyfin manager
//  2. Waits for context cancellation
//  3. Calls Stop() for graceful shutdown
type JellyfinService struct {
	manager JellyfinStartStopManager
	name    string
}

// NewJellyfinService creates a new Jellyfin service wrapper.
//
// Example usage:
//
//	jellyfinManager := sync.NewJellyfinManager(cfg.Jellyfin, wsHub)
//	svc := services.NewJellyfinService(jellyfinManager)
//	tree.AddMessagingService(svc)
func NewJellyfinService(manager JellyfinStartStopManager) *JellyfinService {
	return &JellyfinService{
		manager: manager,
		name:    "jellyfin-manager",
	}
}

// NewJellyfinServiceWithName creates a new Jellyfin service wrapper with a custom name.
// This is useful when running multiple Jellyfin servers to differentiate them in logs.
//
// Example usage:
//
//	jellyfinManager := sync.NewJellyfinManager(&cfg, wsHub, userResolver)
//	svc := services.NewJellyfinServiceWithName(jellyfinManager, "jellyfin-home")
//	tree.AddMessagingService(svc)
func NewJellyfinServiceWithName(manager JellyfinStartStopManager, name string) *JellyfinService {
	return &JellyfinService{
		manager: manager,
		name:    name,
	}
}

// Serve implements suture.Service.
//
// This method:
//  1. Starts the Jellyfin manager (which spawns its internal goroutines)
//  2. Blocks until the context is canceled
//  3. Stops the Jellyfin manager (which waits for its goroutines to complete)
//
// If Start() fails, the error is returned immediately, causing suture to
// restart the service according to its backoff policy.
func (s *JellyfinService) Serve(ctx context.Context) error {
	// Start the manager
	if err := s.manager.Start(ctx); err != nil {
		return fmt.Errorf("jellyfin manager start failed: %w", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop the manager
	if err := s.manager.Stop(); err != nil {
		return fmt.Errorf("jellyfin manager stop failed: %w", err)
	}

	return ctx.Err()
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (s *JellyfinService) String() string {
	return s.name
}
