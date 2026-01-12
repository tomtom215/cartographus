// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"fmt"
	"time"
)

// ResilientReaderConfig configures the resilient reader behavior.
type ResilientReaderConfig struct {
	NATSURL             string
	CircuitBreakerName  string
	MaxRequests         uint32
	Interval            time.Duration
	Timeout             time.Duration
	FailureThreshold    uint32
	HealthCheckInterval time.Duration
	EnablePrimaryReader bool
}

// DefaultResilientReaderConfig returns production defaults.
func DefaultResilientReaderConfig(natsURL string) ResilientReaderConfig {
	return ResilientReaderConfig{
		NATSURL:             natsURL,
		CircuitBreakerName:  "stream-reader",
		MaxRequests:         3,
		Interval:            30 * time.Second,
		Timeout:             10 * time.Second,
		FailureThreshold:    5,
		HealthCheckInterval: 30 * time.Second,
		EnablePrimaryReader: false,
	}
}

// ResilientReader is a stub when NATS dependencies are not available.
type ResilientReader struct{}

// NewResilientReader returns an error when NATS dependencies are not available.
func NewResilientReader(cfg *ResilientReaderConfig) (*ResilientReader, error) {
	return nil, fmt.Errorf("resilient reader not available: build with -tags=nats")
}

// SetPrimaryReader is a no-op stub.
func (r *ResilientReader) SetPrimaryReader(primary StreamReader) {}

// Query is a stub that returns an error.
func (r *ResilientReader) Query(ctx context.Context, stream string, opts *QueryOptions) ([]StreamMessage, error) {
	return nil, fmt.Errorf("resilient reader not available: build with -tags=nats")
}

// GetMessage is a stub that returns an error.
func (r *ResilientReader) GetMessage(ctx context.Context, stream string, seq uint64) (*StreamMessage, error) {
	return nil, fmt.Errorf("resilient reader not available: build with -tags=nats")
}

// GetLastSequence is a stub that returns an error.
func (r *ResilientReader) GetLastSequence(ctx context.Context, stream string) (uint64, error) {
	return 0, fmt.Errorf("resilient reader not available: build with -tags=nats")
}

// Health is a stub that returns an error.
func (r *ResilientReader) Health(ctx context.Context) error {
	return fmt.Errorf("resilient reader not available: build with -tags=nats")
}

// Close is a no-op stub.
func (r *ResilientReader) Close() error {
	return nil
}

// Stats returns empty statistics.
func (r *ResilientReader) Stats() ReaderStats {
	return ReaderStats{}
}

// FallbackCount returns 0.
func (r *ResilientReader) FallbackCount() int64 {
	return 0
}
