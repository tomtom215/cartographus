// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package main

import (
	"context"

	"github.com/tomtom215/cartographus/internal/api"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/detection"
	"github.com/tomtom215/cartographus/internal/logging"
	intsync "github.com/tomtom215/cartographus/internal/sync"
	ws "github.com/tomtom215/cartographus/internal/websocket"
)

// NATSComponents is a stub for non-NATS builds.
type NATSComponents struct{}

// InitNATS is a no-op stub for non-NATS builds.
// Returns nil to indicate NATS is not available.
func InitNATS(cfg *config.Config, syncManager *intsync.Manager, wsHub *ws.Hub, handler *api.Handler, db *database.DB, _ *detection.Engine) (*NATSComponents, error) {
	if cfg.NATS.Enabled {
		logging.Warn().Msg("NATS_ENABLED=true but NATS support not compiled (build with -tags nats)")
	}
	return nil, nil
}

// Start is a no-op stub for non-NATS builds.
func (c *NATSComponents) Start(_ context.Context) error {
	return nil
}

// Shutdown is a no-op stub for non-NATS builds.
func (c *NATSComponents) Shutdown(_ context.Context) {}

// IsRunning returns false for non-NATS builds.
func (c *NATSComponents) IsRunning() bool {
	return false
}

// BadgerDB returns nil for non-NATS builds.
func (c *NATSComponents) BadgerDB() interface{} {
	return nil
}

// EventPublisher returns nil for non-NATS builds.
func (c *NATSComponents) EventPublisher() intsync.EventPublisher {
	return nil
}
