// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package services

import (
	"context"
	"fmt"
	"time"
)

// NATSComponentsRunner interface matches the existing NATSComponents lifecycle.
//
// This interface allows the NATSComponentsService to work with NATSComponents
// without importing the main package, avoiding circular dependencies.
//
// Satisfied by *NATSComponents from cmd/server/nats_init.go:
//   - Start(ctx context.Context) error - starts Router and DuckDB appender
//   - Shutdown(ctx context.Context) - stops Router and all components
//   - IsRunning() bool - returns running state
type NATSComponentsRunner interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context)
	IsRunning() bool
}

// NATSComponentsService wraps NATSComponents as a supervised service.
//
// It adapts the Start/Shutdown lifecycle pattern to suture's Serve pattern:
//  1. Calls Start(ctx) to begin all NATS components
//  2. Waits for context cancellation
//  3. Calls Shutdown(ctx) for graceful cleanup
//
// The service manages multiple NATS subsystems including:
//   - Embedded NATS server (if configured)
//   - JetStream connection and publisher
//   - Watermill Router (handles WebSocket and DuckDB message processing)
//   - DuckDB appender (batch writes)
//   - Health checker
//
// Example usage:
//
//	natsComponents, _ := InitNATS(cfg, syncManager, wsHub, handler, db)
//	svc := services.NewNATSComponentsService(natsComponents)
//	tree.AddMessagingService(svc)
type NATSComponentsService struct {
	components      NATSComponentsRunner
	shutdownTimeout time.Duration
	name            string
}

// NewNATSComponentsService creates a new NATS components service wrapper.
//
// Uses a default shutdown timeout of 10 seconds, matching the existing
// shutdown behavior in cmd/server/main.go:308-310.
func NewNATSComponentsService(components NATSComponentsRunner) *NATSComponentsService {
	return &NATSComponentsService{
		components:      components,
		shutdownTimeout: 10 * time.Second,
		name:            "nats-components",
	}
}

// NewNATSComponentsServiceWithTimeout creates a NATS service with custom shutdown timeout.
func NewNATSComponentsServiceWithTimeout(components NATSComponentsRunner, shutdownTimeout time.Duration) *NATSComponentsService {
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}
	return &NATSComponentsService{
		components:      components,
		shutdownTimeout: shutdownTimeout,
		name:            "nats-components",
	}
}

// Serve implements suture.Service.
//
// This method:
//  1. Starts all NATS components (Router, DuckDB appender)
//  2. Blocks until the context is canceled
//  3. Shuts down all components with the configured timeout
//
// If Start() fails, the error is returned immediately, causing suture to
// restart the service according to its backoff policy.
func (s *NATSComponentsService) Serve(ctx context.Context) error {
	// Start all NATS components
	if err := s.components.Start(ctx); err != nil {
		return fmt.Errorf("NATS components start failed: %w", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Shutdown with timeout - use fresh context since original is canceled
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	s.components.Shutdown(shutdownCtx)

	return ctx.Err()
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (s *NATSComponentsService) String() string {
	return s.name
}
