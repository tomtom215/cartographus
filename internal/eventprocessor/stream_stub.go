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

// StreamManager is a stub when NATS dependencies are not available.
// Build with -tags=nats to enable full JetStream stream support.
type StreamManager struct {
	// stub - no fields needed
}

// NewStreamManager returns an error when NATS dependencies are not available.
// Build with -tags=nats to enable full JetStream stream support.
func NewStreamManager(nc interface{}, cfg *StreamConfig) (*StreamManager, error) {
	return nil, fmt.Errorf("NATS stream manager not available: build with -tags=nats")
}

// EnsureStream is a stub that returns an error.
func (m *StreamManager) EnsureStream(ctx context.Context) (interface{}, error) {
	return nil, fmt.Errorf("NATS stream manager not available: build with -tags=nats")
}

// GetStreamInfo is a stub that returns an error.
func (m *StreamManager) GetStreamInfo(ctx context.Context) (interface{}, error) {
	return nil, fmt.Errorf("NATS stream manager not available: build with -tags=nats")
}

// PurgeStream is a stub that returns an error.
func (m *StreamManager) PurgeStream(ctx context.Context) error {
	return fmt.Errorf("NATS stream manager not available: build with -tags=nats")
}

// DeleteStream is a stub that returns an error.
func (m *StreamManager) DeleteStream(ctx context.Context) error {
	return fmt.Errorf("NATS stream manager not available: build with -tags=nats")
}
