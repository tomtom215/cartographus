// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"time"
)

// DuckDBHandler is a stub for non-NATS builds.
type DuckDBHandler struct{}

// DuckDBHandlerConfig holds configuration for the DuckDB handler.
type DuckDBHandlerConfig struct {
	EnableCrossSourceDedup  bool
	DeduplicationWindow     time.Duration
	MaxDeduplicationEntries int
}

// DefaultDuckDBHandlerConfig returns production defaults.
func DefaultDuckDBHandlerConfig() DuckDBHandlerConfig {
	return DuckDBHandlerConfig{
		EnableCrossSourceDedup:  true,
		DeduplicationWindow:     5 * time.Minute,
		MaxDeduplicationEntries: 10000,
	}
}

// DuckDBHandlerStats holds runtime statistics.
type DuckDBHandlerStats struct {
	MessagesReceived  int64
	MessagesProcessed int64
	DuplicatesSkipped int64
	ParseErrors       int64
	LastMessageTime   time.Time
}

// Stats is a stub for non-NATS builds.
func (h *DuckDBHandler) Stats() DuckDBHandlerStats {
	return DuckDBHandlerStats{}
}

// StartCleanup is a stub for non-NATS builds.
func (h *DuckDBHandler) StartCleanup(_ context.Context) {}

// WebSocketHandler is a stub for non-NATS builds.
type WebSocketHandler struct{}

// WebSocketBroadcaster defines the interface for broadcasting to WebSocket clients.
type WebSocketBroadcaster interface {
	BroadcastRaw(data []byte)
}

// WebSocketHandlerStats holds runtime statistics.
type WebSocketHandlerStats struct {
	MessagesReceived  int64
	MessagesBroadcast int64
}

// Stats is a stub for non-NATS builds.
func (h *WebSocketHandler) Stats() WebSocketHandlerStats {
	return WebSocketHandlerStats{}
}
