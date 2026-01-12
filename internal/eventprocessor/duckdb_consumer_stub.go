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

// ConsumerConfig holds configuration for the DuckDB consumer.
// This is a stub for non-NATS builds.
type ConsumerConfig struct {
	Topic                   string
	EnableDeduplication     bool
	DeduplicationWindow     time.Duration
	MaxDeduplicationEntries int
	WorkerCount             int
}

// DefaultConsumerConfig returns default configuration.
// This is a stub for non-NATS builds.
func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		Topic:                   "playback.>",
		EnableDeduplication:     true,
		DeduplicationWindow:     5 * time.Minute,
		MaxDeduplicationEntries: 10000,
		WorkerCount:             1,
	}
}

// ConsumerStats holds runtime statistics.
// This is a stub for non-NATS builds.
type ConsumerStats struct {
	MessagesReceived  int64
	MessagesProcessed int64
	ParseErrors       int64
	DuplicatesSkipped int64
	LastMessageTime   time.Time
}

// DuckDBConsumer is a stub for non-NATS builds.
//
// Deprecated: DuckDBConsumer has been replaced by the Router-based approach using
// DuckDBHandler. See duckdb_consumer.go for migration guide.
type DuckDBConsumer struct{}

// NewDuckDBConsumer returns an error in non-NATS builds.
func NewDuckDBConsumer(_ interface{}, _ *Appender, _ *ConsumerConfig) (*DuckDBConsumer, error) {
	return nil, ErrNATSNotEnabled
}

// Start is a stub for non-NATS builds.
func (c *DuckDBConsumer) Start(_ context.Context) error {
	return ErrNATSNotEnabled
}

// Stop is a stub for non-NATS builds.
func (c *DuckDBConsumer) Stop() {}

// IsRunning is a stub for non-NATS builds.
func (c *DuckDBConsumer) IsRunning() bool {
	return false
}

// Stats is a stub for non-NATS builds.
func (c *DuckDBConsumer) Stats() ConsumerStats {
	return ConsumerStats{}
}
