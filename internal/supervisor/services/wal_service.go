// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package services

import (
	"context"
	"fmt"
)

// WALStartStopper interface matches the WAL RetryLoop and Compactor lifecycle.
//
// This interface allows the WAL services to work with the actual WAL components
// without importing the wal package, avoiding circular dependencies.
//
// Satisfied by:
//   - *wal.RetryLoop from internal/wal/retry.go:41-80
//   - *wal.Compactor from internal/wal/compaction.go:45-83
type WALStartStopper interface {
	Start(ctx context.Context) error
	Stop()
	IsRunning() bool
}

// WALRetryLoopService wraps the WAL retry loop as a supervised service.
//
// The retry loop handles background retry of failed WAL entries, attempting
// to publish them to NATS with exponential backoff.
//
// It adapts the Start/Stop lifecycle pattern to suture's Serve pattern:
//  1. Calls Start(ctx) to begin the retry loop
//  2. Waits for context cancellation
//  3. Calls Stop() for graceful shutdown (waits for goroutines via WaitGroup)
//
// Example usage:
//
//	retryLoop := wal.NewRetryLoop(w, walPub)
//	svc := services.NewWALRetryLoopService(retryLoop)
//	tree.AddDataService(svc)
type WALRetryLoopService struct {
	retryLoop WALStartStopper
	name      string
}

// NewWALRetryLoopService creates a new WAL retry loop service wrapper.
func NewWALRetryLoopService(retryLoop WALStartStopper) *WALRetryLoopService {
	return &WALRetryLoopService{
		retryLoop: retryLoop,
		name:      "wal-retry-loop",
	}
}

// Serve implements suture.Service.
//
// This method:
//  1. Starts the retry loop (which spawns its background goroutine)
//  2. Blocks until the context is canceled
//  3. Stops the retry loop (which waits for the goroutine to complete)
//
// If Start() fails, the error is returned immediately, causing suture to
// restart the service according to its backoff policy.
func (s *WALRetryLoopService) Serve(ctx context.Context) error {
	// Start the retry loop
	if err := s.retryLoop.Start(ctx); err != nil {
		return fmt.Errorf("WAL retry loop start failed: %w", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop the retry loop - this blocks until the background goroutine exits
	// per retry.go:71 (r.wg.Wait())
	s.retryLoop.Stop()

	return ctx.Err()
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (s *WALRetryLoopService) String() string {
	return s.name
}

// WALCompactorService wraps the WAL compactor as a supervised service.
//
// The compactor handles periodic cleanup of confirmed WAL entries and
// triggers BadgerDB garbage collection.
//
// It adapts the Start/Stop lifecycle pattern to suture's Serve pattern:
//  1. Calls Start(ctx) to begin the compaction loop
//  2. Waits for context cancellation
//  3. Calls Stop() for graceful shutdown (waits for goroutines via WaitGroup)
//
// Example usage:
//
//	compactor := wal.NewCompactor(w)
//	svc := services.NewWALCompactorService(compactor)
//	tree.AddDataService(svc)
type WALCompactorService struct {
	compactor WALStartStopper
	name      string
}

// NewWALCompactorService creates a new WAL compactor service wrapper.
func NewWALCompactorService(compactor WALStartStopper) *WALCompactorService {
	return &WALCompactorService{
		compactor: compactor,
		name:      "wal-compactor",
	}
}

// Serve implements suture.Service.
//
// This method:
//  1. Starts the compactor (which spawns its background goroutine)
//  2. Blocks until the context is canceled
//  3. Stops the compactor (which waits for the goroutine to complete)
//
// If Start() fails, the error is returned immediately, causing suture to
// restart the service according to its backoff policy.
func (s *WALCompactorService) Serve(ctx context.Context) error {
	// Start the compactor
	if err := s.compactor.Start(ctx); err != nil {
		return fmt.Errorf("WAL compactor start failed: %w", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop the compactor - this blocks until the background goroutine exits
	// per compaction.go:74 (c.wg.Wait())
	s.compactor.Stop()

	return ctx.Err()
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (s *WALCompactorService) String() string {
	return s.name
}
