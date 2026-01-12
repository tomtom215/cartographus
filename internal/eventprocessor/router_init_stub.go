// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
)

// RouterComponents is a stub for non-NATS builds.
type RouterComponents struct{}

// RouterComponentsConfig is a stub for non-NATS builds.
type RouterComponentsConfig struct {
	RouterConfig         *RouterConfig
	DuckDBHandlerConfig  DuckDBHandlerConfig
	ForwarderConfig      ForwarderConfig
	EnableForwarder      bool
	PoisonQueuePublisher interface{}
	WebSocketBroadcaster WebSocketBroadcaster
}

// DefaultRouterComponentsConfig returns production defaults.
func DefaultRouterComponentsConfig() RouterComponentsConfig {
	defaultRouterCfg := DefaultRouterConfig()
	return RouterComponentsConfig{
		RouterConfig:        &defaultRouterCfg,
		DuckDBHandlerConfig: DefaultDuckDBHandlerConfig(),
		ForwarderConfig:     DefaultForwarderConfig(),
	}
}

// Start is a stub for non-NATS builds.
func (c *RouterComponents) Start(_ context.Context) error {
	return ErrNATSNotEnabled
}

// Stop is a stub for non-NATS builds.
func (c *RouterComponents) Stop() error {
	return nil
}

// IsRunning is a stub for non-NATS builds.
func (c *RouterComponents) IsRunning() bool {
	return false
}

// Stats returns empty statistics for non-NATS builds.
func (c *RouterComponents) Stats() RouterComponentsStats {
	return RouterComponentsStats{}
}

// RouterComponentsStats holds combined statistics.
type RouterComponentsStats struct {
	Router    *RouterMetrics
	DuckDB    DuckDBHandlerStats
	WebSocket WebSocketHandlerStats
}

// MigrationGuide provides documentation stub for non-NATS builds.
type MigrationGuide struct{}
