// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !wal || !nats

package main

import (
	"context"

	"github.com/tomtom215/cartographus/internal/logging"
	intsync "github.com/tomtom215/cartographus/internal/sync"
	"github.com/tomtom215/cartographus/internal/wal"
)

// WALComponents is a stub for builds without WAL support.
type WALComponents struct{}

// InitWAL returns nil when WAL is disabled via build tags.
func InitWAL(_ context.Context, _ interface{}) (*WALComponents, error) {
	logging.Info().Msg("WAL not available (built without -tags wal,nats)")
	return nil, nil
}

// EventPublisher returns nil when WAL is disabled.
// This allows callers to check if WAL is providing an EventPublisher.
func (c *WALComponents) EventPublisher() intsync.EventPublisher {
	return nil
}

// Shutdown does nothing when WAL is disabled.
func (c *WALComponents) Shutdown() {}

// Stats returns empty stats when WAL is disabled.
func (c *WALComponents) Stats() wal.Stats {
	return wal.Stats{}
}

// BadgerDB returns nil when WAL is disabled.
func (c *WALComponents) BadgerDB() interface{} {
	return nil
}
