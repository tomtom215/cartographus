// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package logging

import (
	"context"

	"github.com/rs/zerolog"
)

// EventLogger provides specialized logging for event processing.
// This logger is designed for NATS/Watermill event handlers with
// domain-specific methods for common event processing scenarios.
type EventLogger struct {
	logger zerolog.Logger
}

// NewEventLogger creates a logger configured for event processing.
// If logger is nil, uses the global logger with component field.
func NewEventLogger() *EventLogger {
	return &EventLogger{
		logger: With().Str("component", "eventprocessor").Logger(),
	}
}

// NewEventLoggerWithLogger creates an EventLogger with a custom logger.
//
//nolint:gocritic // zerolog.Logger is designed to be passed by value (copy-on-write semantics)
func NewEventLoggerWithLogger(logger zerolog.Logger) *EventLogger {
	return &EventLogger{
		logger: logger.With().Str("component", "eventprocessor").Logger(),
	}
}

// WithFields returns a new EventLogger with additional default fields.
func (e *EventLogger) WithFields(fields map[string]interface{}) *EventLogger {
	ctx := e.logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &EventLogger{logger: ctx.Logger()}
}

// Debug logs a debug message.
func (e *EventLogger) Debug(msg string, fields ...interface{}) {
	event := e.logger.Debug()
	event = addFieldPairs(event, fields)
	event.Msg(msg)
}

// Info logs an info message.
func (e *EventLogger) Info(msg string, fields ...interface{}) {
	event := e.logger.Info()
	event = addFieldPairs(event, fields)
	event.Msg(msg)
}

// Warn logs a warning message.
func (e *EventLogger) Warn(msg string, fields ...interface{}) {
	event := e.logger.Warn()
	event = addFieldPairs(event, fields)
	event.Msg(msg)
}

// Error logs an error message.
func (e *EventLogger) Error(msg string, fields ...interface{}) {
	event := e.logger.Error()
	event = addFieldPairs(event, fields)
	event.Msg(msg)
}

// DebugContext logs a debug message with context (for correlation ID).
func (e *EventLogger) DebugContext(ctx context.Context, msg string, fields ...interface{}) {
	logger := e.loggerWithContext(ctx)
	event := logger.Debug()
	event = addFieldPairs(event, fields)
	event.Msg(msg)
}

// InfoContext logs an info message with context.
func (e *EventLogger) InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	logger := e.loggerWithContext(ctx)
	event := logger.Info()
	event = addFieldPairs(event, fields)
	event.Msg(msg)
}

// WarnContext logs a warning message with context.
func (e *EventLogger) WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	logger := e.loggerWithContext(ctx)
	event := logger.Warn()
	event = addFieldPairs(event, fields)
	event.Msg(msg)
}

// ErrorContext logs an error message with context.
func (e *EventLogger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	logger := e.loggerWithContext(ctx)
	event := logger.Error()
	event = addFieldPairs(event, fields)
	event.Msg(msg)
}

// loggerWithContext returns a logger with context fields added.
func (e *EventLogger) loggerWithContext(ctx context.Context) zerolog.Logger {
	logCtx := e.logger.With()

	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		logCtx = logCtx.Str("correlation_id", correlationID)
	}

	if requestID := RequestIDFromContext(ctx); requestID != "" {
		logCtx = logCtx.Str("request_id", requestID)
	}

	return logCtx.Logger()
}

// ============================================================
// Domain-Specific Event Logging Methods
// ============================================================

// LogEventReceived logs when an event is received.
func (e *EventLogger) LogEventReceived(ctx context.Context, eventID, source, mediaType string) {
	e.InfoContext(ctx, "event received",
		"event_id", eventID,
		"source", source,
		"media_type", mediaType,
	)
}

// LogEventProcessed logs when an event is successfully processed.
func (e *EventLogger) LogEventProcessed(ctx context.Context, eventID string, durationMs int64) {
	e.InfoContext(ctx, "event processed",
		"event_id", eventID,
		"duration_ms", durationMs,
	)
}

// LogEventFailed logs when event processing fails.
func (e *EventLogger) LogEventFailed(ctx context.Context, eventID string, err error) {
	logger := e.loggerWithContext(ctx)
	event := logger.Error().
		Str("event_id", eventID).
		Err(err)
	event.Msg("event processing failed")
}

// LogDuplicate logs when a duplicate event is detected.
func (e *EventLogger) LogDuplicate(ctx context.Context, eventID, reason string) {
	e.DebugContext(ctx, "duplicate event skipped",
		"event_id", eventID,
		"reason", reason,
	)
}

// LogDLQEntry logs when an event is sent to the DLQ.
func (e *EventLogger) LogDLQEntry(ctx context.Context, eventID string, err error, retryCount int) {
	logger := e.loggerWithContext(ctx)
	event := logger.Warn().
		Str("event_id", eventID).
		Err(err).
		Int("retry_count", retryCount)
	event.Msg("event sent to DLQ")
}

// LogBatchFlush logs batch flush operations.
func (e *EventLogger) LogBatchFlush(ctx context.Context, count int, durationMs int64) {
	e.InfoContext(ctx, "batch flush completed",
		"event_count", count,
		"duration_ms", durationMs,
	)
}

// LogEventPublished logs when an event is published to NATS.
func (e *EventLogger) LogEventPublished(ctx context.Context, eventID, topic string) {
	e.DebugContext(ctx, "event published",
		"event_id", eventID,
		"topic", topic,
	)
}

// LogSubscriptionStarted logs when a subscription is started.
func (e *EventLogger) LogSubscriptionStarted(topic, queue string) {
	e.Info("subscription started",
		"topic", topic,
		"queue", queue,
	)
}

// LogSubscriptionStopped logs when a subscription is stopped.
func (e *EventLogger) LogSubscriptionStopped(topic string) {
	e.Info("subscription stopped",
		"topic", topic,
	)
}

// LogRouterStarted logs when the Watermill router starts.
func (e *EventLogger) LogRouterStarted() {
	e.Info("router started")
}

// LogRouterStopped logs when the Watermill router stops.
func (e *EventLogger) LogRouterStopped() {
	e.Info("router stopped")
}
