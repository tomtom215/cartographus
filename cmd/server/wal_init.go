// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal && nats

package main

import (
	"context"

	"github.com/tomtom215/cartographus/internal/eventprocessor"
	"github.com/tomtom215/cartographus/internal/logging"
	intsync "github.com/tomtom215/cartographus/internal/sync"
	"github.com/tomtom215/cartographus/internal/wal"
)

// WALComponents holds WAL-related components for lifecycle management.
type WALComponents struct {
	wal       *wal.BadgerWAL
	retryLoop *wal.RetryLoop
	compactor *wal.Compactor
	publisher *eventprocessor.WALEnabledPublisher
}

// InitWAL initializes the Write-Ahead Log for event durability.
// It returns WAL components that should be managed alongside NATS components.
//
// The WAL ensures no event loss by persisting events to BadgerDB before NATS publishing.
// If WAL is enabled, it wraps the SyncEventPublisher with WAL durability.
func InitWAL(ctx context.Context, syncPublisher *eventprocessor.SyncEventPublisher) (*WALComponents, error) {
	cfg := wal.LoadConfig()

	if !cfg.Enabled {
		logging.Warn().Msg("WAL disabled (WAL_ENABLED=false). Events may be lost if NATS fails.")
		return nil, nil
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	logging.Info().Str("path", cfg.Path).Bool("sync_writes", cfg.SyncWrites).Msg("Initializing WAL...")

	// Open BadgerDB WAL
	w, err := wal.Open(&cfg)
	if err != nil {
		return nil, err
	}

	components := &WALComponents{
		wal: w,
	}

	// Create WAL-enabled publisher
	walPublisher, err := eventprocessor.NewWALEnabledPublisher(syncPublisher, w)
	if err != nil {
		if closeErr := w.Close(); closeErr != nil {
			logging.Error().Err(closeErr).Msg("Error closing WAL after publisher creation failure")
		}
		return nil, err
	}
	components.publisher = walPublisher
	logging.Info().Msg("WAL-enabled event publisher created")

	// Create WAL publisher adapter for recovery and retry
	walPub := walPublisher.CreateWALPublisher()

	// Run recovery of pending entries from previous run
	logging.Info().Msg("Running WAL recovery for pending entries...")
	result, err := w.RecoverPending(ctx, walPub)
	if err != nil {
		logging.Warn().Err(err).Msg("WAL recovery error")
		// Don't fail initialization - recovery is best-effort
	} else if result != nil && result.TotalPending > 0 {
		logging.Info().
			Int("total", result.TotalPending).
			Int("recovered", result.Recovered).
			Int("failed", result.Failed).
			Int("expired", result.Expired).
			Msg("WAL recovery completed")
	}

	// Create and start background retry loop
	retryLoop := wal.NewRetryLoop(w, walPub)
	if err := retryLoop.Start(ctx); err != nil {
		if closeErr := w.Close(); closeErr != nil {
			logging.Error().Err(closeErr).Msg("Error closing WAL after retry loop start failure")
		}
		return nil, err
	}
	components.retryLoop = retryLoop
	logging.Info().Msg("WAL retry loop started")

	// Create and start compactor
	compactor := wal.NewCompactor(w)
	if err := compactor.Start(ctx); err != nil {
		retryLoop.Stop()
		if closeErr := w.Close(); closeErr != nil {
			logging.Error().Err(closeErr).Msg("Error closing WAL after compactor start failure")
		}
		return nil, err
	}
	components.compactor = compactor
	logging.Info().Msg("WAL compactor started")

	logging.Info().Msg("WAL initialized successfully")
	return components, nil
}

// Publisher returns the WAL-enabled publisher.
// Use this instead of the direct SyncEventPublisher when WAL is enabled.
func (c *WALComponents) Publisher() *eventprocessor.WALEnabledPublisher {
	if c == nil {
		return nil
	}
	return c.publisher
}

// EventPublisher returns the WAL-enabled publisher as a sync.EventPublisher interface.
// This provides a unified interface for callers regardless of WAL build tags.
// Returns nil if WAL is not initialized or disabled.
func (c *WALComponents) EventPublisher() intsync.EventPublisher {
	if c == nil || c.publisher == nil {
		return nil
	}
	return c.publisher
}

// BadgerDB returns the underlying BadgerDB instance from the WAL.
// This allows other components (like import progress tracking) to share
// the same BadgerDB instance. Returns nil if WAL is not initialized.
func (c *WALComponents) BadgerDB() interface{} {
	if c == nil || c.wal == nil {
		return nil
	}
	return c.wal.DB()
}

// Shutdown gracefully stops all WAL components.
func (c *WALComponents) Shutdown() {
	if c == nil {
		return
	}

	logging.Info().Msg("Shutting down WAL components...")

	// Stop retry loop first
	if c.retryLoop != nil {
		c.retryLoop.Stop()
		logging.Info().Msg("WAL retry loop stopped")
	}

	// Stop compactor
	if c.compactor != nil {
		c.compactor.Stop()
		logging.Info().Msg("WAL compactor stopped")
	}

	// Close WAL
	if c.wal != nil {
		if err := c.wal.Close(); err != nil {
			logging.Error().Err(err).Msg("Error closing WAL")
		}
		logging.Info().Msg("WAL closed")
	}

	logging.Info().Msg("WAL shutdown complete")
}

// Stats returns current WAL statistics.
func (c *WALComponents) Stats() wal.Stats {
	if c == nil || c.wal == nil {
		return wal.Stats{}
	}
	return c.wal.Stats()
}
