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

// EventStore defines the interface for persisting media events.
// This is a stub for non-NATS builds.
type EventStore interface {
	InsertMediaEvents(ctx context.Context, events []*MediaEvent) error
}

// AppenderStats holds runtime statistics for monitoring.
// This is a stub for non-NATS builds.
type AppenderStats struct {
	EventsReceived int64
	EventsFlushed  int64
	FlushCount     int64
	ErrorCount     int64
	LastFlushTime  time.Time
	LastError      string
	BufferSize     int
	AvgFlushTime   time.Duration
}

// Appender is a stub for non-NATS builds.
type Appender struct{}

// NewAppender returns an error in non-NATS builds.
func NewAppender(_ EventStore, _ AppenderConfig) (*Appender, error) {
	return nil, ErrNATSNotEnabled
}

// Start is a no-op stub.
func (a *Appender) Start(_ context.Context) error {
	return ErrNATSNotEnabled
}

// Append is a no-op stub.
func (a *Appender) Append(_ context.Context, _ *MediaEvent) error {
	return ErrNATSNotEnabled
}

// Flush is a no-op stub.
func (a *Appender) Flush(_ context.Context) error {
	return ErrNATSNotEnabled
}

// Close is a no-op stub.
func (a *Appender) Close() error {
	return nil
}

// Stats returns empty stats in non-NATS builds.
func (a *Appender) Stats() AppenderStats {
	return AppenderStats{}
}
