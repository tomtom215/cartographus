// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package logging

import (
	"context"

	"github.com/rs/zerolog"
)

// EventLogger provides specialized logging for event processing.
// This is a stub implementation for non-NATS builds.
type EventLogger struct{}

// NewEventLogger creates a logger configured for event processing.
func NewEventLogger() *EventLogger {
	return &EventLogger{}
}

// NewEventLoggerWithLogger creates an EventLogger with a custom logger.
//
//nolint:gocritic // zerolog.Logger is designed to be passed by value
func NewEventLoggerWithLogger(_ zerolog.Logger) *EventLogger {
	return &EventLogger{}
}

// WithFields returns a new EventLogger with additional default fields.
func (e *EventLogger) WithFields(_ map[string]interface{}) *EventLogger {
	return e
}

// Debug logs a debug message (no-op).
func (e *EventLogger) Debug(_ string, _ ...interface{}) {}

// Info logs an info message (no-op).
func (e *EventLogger) Info(_ string, _ ...interface{}) {}

// Warn logs a warning message (no-op).
func (e *EventLogger) Warn(_ string, _ ...interface{}) {}

// Error logs an error message (no-op).
func (e *EventLogger) Error(_ string, _ ...interface{}) {}

// DebugContext logs a debug message with context (no-op).
func (e *EventLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}

// InfoContext logs an info message with context (no-op).
func (e *EventLogger) InfoContext(_ context.Context, _ string, _ ...interface{}) {}

// WarnContext logs a warning message with context (no-op).
func (e *EventLogger) WarnContext(_ context.Context, _ string, _ ...interface{}) {}

// ErrorContext logs an error message with context (no-op).
func (e *EventLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}

// LogEventReceived logs when an event is received (no-op).
func (e *EventLogger) LogEventReceived(_ context.Context, _, _, _ string) {}

// LogEventProcessed logs when an event is successfully processed (no-op).
func (e *EventLogger) LogEventProcessed(_ context.Context, _ string, _ int64) {}

// LogEventFailed logs when event processing fails (no-op).
func (e *EventLogger) LogEventFailed(_ context.Context, _ string, _ error) {}

// LogDuplicate logs when a duplicate event is detected (no-op).
func (e *EventLogger) LogDuplicate(_ context.Context, _, _ string) {}

// LogDLQEntry logs when an event is sent to the DLQ (no-op).
func (e *EventLogger) LogDLQEntry(_ context.Context, _ string, _ error, _ int) {}

// LogBatchFlush logs batch flush operations (no-op).
func (e *EventLogger) LogBatchFlush(_ context.Context, _ int, _ int64) {}

// LogEventPublished logs when an event is published to NATS (no-op).
func (e *EventLogger) LogEventPublished(_ context.Context, _, _ string) {}

// LogSubscriptionStarted logs when a subscription is started (no-op).
func (e *EventLogger) LogSubscriptionStarted(_, _ string) {}

// LogSubscriptionStopped logs when a subscription is stopped (no-op).
func (e *EventLogger) LogSubscriptionStopped(_ string) {}

// LogRouterStarted logs when the Watermill router starts (no-op).
func (e *EventLogger) LogRouterStarted() {}

// LogRouterStopped logs when the Watermill router stops (no-op).
func (e *EventLogger) LogRouterStopped() {}
