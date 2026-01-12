// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"fmt"
)

// StreamInitializer is a stub when NATS is not enabled.
type StreamInitializer struct{}

// NewStreamInitializer returns an error when NATS is not compiled in.
func NewStreamInitializer(js interface{}, cfg *StreamConfig) (*StreamInitializer, error) {
	return nil, fmt.Errorf("NATS not available: build with -tags=nats")
}

// EnsureStream is a no-op stub.
func (s *StreamInitializer) EnsureStream(ctx context.Context) (interface{}, error) {
	return nil, ErrNATSNotEnabled
}

// GetStreamInfo is a no-op stub.
func (s *StreamInitializer) GetStreamInfo(ctx context.Context) (interface{}, error) {
	return nil, ErrNATSNotEnabled
}

// IsHealthy always returns false when NATS is not enabled.
func (s *StreamInitializer) IsHealthy(ctx context.Context) bool {
	return false
}

// Config returns an empty configuration.
func (s *StreamInitializer) Config() StreamConfig {
	return StreamConfig{}
}
