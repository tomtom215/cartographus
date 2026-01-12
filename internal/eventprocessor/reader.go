// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package eventprocessor

import (
	"context"
	"time"
)

// StreamMessage represents a message from the NATS JetStream stream.
type StreamMessage struct {
	Sequence  uint64
	Subject   string
	Data      []byte
	Timestamp time.Time
	Headers   map[string][]string
}

// QueryOptions defines stream query parameters.
type QueryOptions struct {
	// StartSeq is the starting sequence number (inclusive).
	// If 0, starts from the first available message.
	StartSeq uint64

	// EndSeq is the ending sequence number (inclusive).
	// If 0, continues to the last available message.
	EndSeq uint64

	// StartTime filters messages after this time.
	StartTime time.Time

	// EndTime filters messages before this time.
	EndTime time.Time

	// Subject filters messages by subject pattern.
	// Supports NATS wildcards (*, >).
	Subject string

	// Limit is the maximum number of messages to return.
	// Default safety limit is 10000 if not specified.
	Limit int

	// JSONExtract specifies JSON fields to extract.
	// Map of field name to JSON path.
	JSONExtract map[string]string
}

// StreamReader provides a unified interface for reading from JetStream streams.
// Implementations include nats_js extension (SQL-based) and Go NATS client fallback.
type StreamReader interface {
	// Query returns messages matching the options.
	Query(ctx context.Context, stream string, opts *QueryOptions) ([]StreamMessage, error)

	// GetMessage retrieves a single message by sequence number.
	GetMessage(ctx context.Context, stream string, seq uint64) (*StreamMessage, error)

	// GetLastSequence returns the latest sequence number in the stream.
	GetLastSequence(ctx context.Context, stream string) (uint64, error)

	// Health checks reader availability.
	Health(ctx context.Context) error

	// Close releases resources.
	Close() error
}

// ReaderType identifies the StreamReader implementation.
type ReaderType string

const (
	// ReaderTypeNatsJS indicates the DuckDB nats_js extension reader.
	ReaderTypeNatsJS ReaderType = "natsjs"
	// ReaderTypeFallback indicates the Go NATS client fallback reader.
	ReaderTypeFallback ReaderType = "fallback"
)

// ReaderStats holds reader statistics for monitoring.
type ReaderStats struct {
	CurrentReader       ReaderType
	CircuitBreakerState string
	PrimaryAvailable    bool
	QueriesTotal        int64
	ErrorsTotal         int64
	LastQueryTime       time.Time
}
