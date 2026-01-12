// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !wal

package wal

import (
	"context"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// WAL provides durable write-ahead logging before NATS publish.
// This stub implementation does nothing when WAL is disabled via build tags.
type WAL interface {
	Write(ctx context.Context, event interface{}) (entryID string, err error)
	Confirm(ctx context.Context, entryID string) error
	GetPending(ctx context.Context) ([]*Entry, error)
	Stats() Stats
	Close() error
}

// Entry represents a single WAL entry (stub).
type Entry struct {
	ID            string          `json:"id"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
	Attempts      int             `json:"attempts"`
	LastAttemptAt time.Time       `json:"last_attempt_at,omitempty"`
	LastError     string          `json:"last_error,omitempty"`
	Confirmed     bool            `json:"confirmed"`
	ConfirmedAt   *time.Time      `json:"confirmed_at,omitempty"`
}

// UnmarshalPayload deserializes the payload into the given type.
func (e *Entry) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal(e.Payload, v)
}

// Stats contains WAL metrics (stub).
type Stats struct {
	PendingCount   int64
	ConfirmedCount int64
	TotalWrites    int64
	TotalConfirms  int64
	TotalRetries   int64
	LastCompaction time.Time
	DBSizeBytes    int64
}

// NoOpWAL is a stub implementation that does nothing.
// Used when the application is built without the 'wal' build tag.
type NoOpWAL struct{}

// Open returns a no-op WAL stub.
// This is used when WAL is disabled via build tags.
func Open(cfg *Config) (*NoOpWAL, error) {
	logging.Info().Msg("WAL disabled (build without -tags wal). Events are not durably stored.")
	return &NoOpWAL{}, nil
}

// OpenForTesting returns a no-op WAL stub for testing.
func OpenForTesting(cfg *Config) (*NoOpWAL, error) {
	return &NoOpWAL{}, nil
}

// Write does nothing and returns an empty entry ID.
func (w *NoOpWAL) Write(ctx context.Context, event interface{}) (string, error) {
	return "", nil
}

// Confirm does nothing.
func (w *NoOpWAL) Confirm(ctx context.Context, entryID string) error {
	return nil
}

// GetPending returns an empty slice.
func (w *NoOpWAL) GetPending(ctx context.Context) ([]*Entry, error) {
	return nil, nil
}

// UpdateAttempt does nothing.
func (w *NoOpWAL) UpdateAttempt(ctx context.Context, entryID string, lastError string) error {
	return nil
}

// DeleteEntry does nothing.
func (w *NoOpWAL) DeleteEntry(ctx context.Context, entryID string) error {
	return nil
}

// Stats returns empty stats.
func (w *NoOpWAL) Stats() Stats {
	return Stats{}
}

// Close does nothing.
func (w *NoOpWAL) Close() error {
	return nil
}

// GetConfig returns the configuration (stub).
func (w *NoOpWAL) GetConfig() Config {
	return Config{}
}

// SetMetricsCallback does nothing.
func (w *NoOpWAL) SetMetricsCallback(cb func(Stats)) {}

// RunGC does nothing.
func (w *NoOpWAL) RunGC() error {
	return nil
}

// Publisher is the interface for publishing WAL entries (stub).
type Publisher interface {
	PublishEntry(ctx context.Context, entry *Entry) error
}

// PublisherFunc is a function type that implements Publisher (stub).
type PublisherFunc func(ctx context.Context, entry *Entry) error

// PublishEntry implements Publisher.
func (f PublisherFunc) PublishEntry(ctx context.Context, entry *Entry) error {
	return f(ctx, entry)
}

// Errors (stub)
var (
	ErrWALClosed     = stubError("WAL is closed")
	ErrNilEvent      = stubError("event cannot be nil")
	ErrEmptyEntryID  = stubError("entry ID cannot be empty")
	ErrEntryNotFound = stubError("entry not found")
)

type stubError string

func (e stubError) Error() string { return string(e) }
