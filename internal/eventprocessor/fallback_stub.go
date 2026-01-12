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

// FallbackReader is a stub when NATS dependencies are not available.
// Build with -tags=nats to enable full NATS client fallback support.
type FallbackReader struct {
	// stub - no fields needed
}

// NewFallbackReader returns an error when NATS dependencies are not available.
// Build with -tags=nats to enable full NATS client fallback support.
func NewFallbackReader(natsURL string) (*FallbackReader, error) {
	return nil, fmt.Errorf("NATS fallback reader not available: build with -tags=nats")
}

// Query is a stub that returns an error.
func (r *FallbackReader) Query(ctx context.Context, streamName string, opts *QueryOptions) ([]StreamMessage, error) {
	return nil, fmt.Errorf("NATS fallback reader not available: build with -tags=nats")
}

// GetMessage is a stub that returns an error.
func (r *FallbackReader) GetMessage(ctx context.Context, streamName string, seq uint64) (*StreamMessage, error) {
	return nil, fmt.Errorf("NATS fallback reader not available: build with -tags=nats")
}

// GetLastSequence is a stub that returns an error.
func (r *FallbackReader) GetLastSequence(ctx context.Context, streamName string) (uint64, error) {
	return 0, fmt.Errorf("NATS fallback reader not available: build with -tags=nats")
}

// Health is a stub that returns an error.
func (r *FallbackReader) Health(ctx context.Context) error {
	return fmt.Errorf("NATS fallback reader not available: build with -tags=nats")
}

// Close is a no-op stub.
func (r *FallbackReader) Close() error {
	return nil
}
