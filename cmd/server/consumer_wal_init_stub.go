// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !wal || !nats

package main

import (
	"context"

	"github.com/tomtom215/cartographus/internal/database"
)

// ConsumerWALComponents is a stub when WAL is not enabled.
type ConsumerWALComponents struct{}

// InitConsumerWAL is a no-op when WAL is not enabled.
// Consumer-side WAL protection is not available without the wal build tag.
func InitConsumerWAL(ctx context.Context, db *database.DB) (*ConsumerWALComponents, error) {
	return nil, nil
}

// WAL returns nil when WAL is not enabled.
func (c *ConsumerWALComponents) WAL() interface{} {
	return nil
}

// Shutdown is a no-op when WAL is not enabled.
func (c *ConsumerWALComponents) Shutdown() {}

// Stats returns empty stats when WAL is not enabled.
func (c *ConsumerWALComponents) Stats() interface{} {
	return nil
}
