// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const correlationIDKey contextKey = "correlation_id"

// LogLevel represents logging severity.
type LogLevel int

const (
	// LogLevelDebug logs all messages including debug.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo logs informational messages and above.
	LogLevelInfo
	// LogLevelWarn logs warnings and errors.
	LogLevelWarn
	// LogLevelError logs only errors.
	LogLevelError
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// GenerateCorrelationID creates a new unique correlation ID.
func GenerateCorrelationID() string {
	return uuid.New().String()[:8] // Short ID for readability
}

// ContextWithCorrelationID returns a context with the given correlation ID.
func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// ContextWithNewCorrelationID returns a context with a new generated correlation ID.
func ContextWithNewCorrelationID(ctx context.Context) context.Context {
	return ContextWithCorrelationID(ctx, GenerateCorrelationID())
}

// CorrelationIDFromContext retrieves the correlation ID from context.
// Returns empty string if not present.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// StructuredLogger provides structured logging with context support.
type StructuredLogger struct {
	mu       sync.RWMutex
	writer   io.Writer
	level    LogLevel
	fields   []interface{}
	timeFunc func() time.Time
}

// NewStructuredLogger creates a new structured logger writing to the given writer.
func NewStructuredLogger(w io.Writer) *StructuredLogger {
	if w == nil {
		w = os.Stdout
	}
	return &StructuredLogger{
		writer:   w,
		level:    LogLevelInfo,
		fields:   make([]interface{}, 0),
		timeFunc: time.Now,
	}
}

// SetLevel sets the minimum log level.
func (l *StructuredLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level.
func (l *StructuredLogger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// WithFields returns a new logger with additional default fields.
func (l *StructuredLogger) WithFields(fields ...interface{}) *StructuredLogger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make([]interface{}, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &StructuredLogger{
		writer:   l.writer,
		level:    l.level,
		fields:   newFields,
		timeFunc: l.timeFunc,
	}
}

// Debug logs a debug message.
func (l *StructuredLogger) Debug(msg string, fields ...interface{}) {
	l.log(context.Background(), LogLevelDebug, msg, fields...)
}

// Info logs an info message.
func (l *StructuredLogger) Info(msg string, fields ...interface{}) {
	l.log(context.Background(), LogLevelInfo, msg, fields...)
}

// Warn logs a warning message.
func (l *StructuredLogger) Warn(msg string, fields ...interface{}) {
	l.log(context.Background(), LogLevelWarn, msg, fields...)
}

// Error logs an error message.
func (l *StructuredLogger) Error(msg string, fields ...interface{}) {
	l.log(context.Background(), LogLevelError, msg, fields...)
}

// DebugContext logs a debug message with context (for correlation ID).
func (l *StructuredLogger) DebugContext(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, LogLevelDebug, msg, fields...)
}

// InfoContext logs an info message with context.
func (l *StructuredLogger) InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, LogLevelInfo, msg, fields...)
}

// WarnContext logs a warning message with context.
func (l *StructuredLogger) WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, LogLevelWarn, msg, fields...)
}

// ErrorContext logs an error message with context.
func (l *StructuredLogger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	l.log(ctx, LogLevelError, msg, fields...)
}

// log writes a log entry if the level is enabled.
func (l *StructuredLogger) log(ctx context.Context, level LogLevel, msg string, fields ...interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	writer := l.writer
	defaultFields := l.fields
	l.mu.RUnlock()

	if level < currentLevel {
		return
	}

	// Build log entry
	timestamp := l.timeFunc().UTC().Format(time.RFC3339)
	correlationID := CorrelationIDFromContext(ctx)

	// Build fields string
	var fieldsStr string

	// Add default fields first
	for i := 0; i < len(defaultFields); i += 2 {
		if i+1 < len(defaultFields) {
			fieldsStr += fmt.Sprintf(" %v=%v", defaultFields[i], defaultFields[i+1])
		}
	}

	// Add correlation ID if present
	if correlationID != "" {
		fieldsStr += fmt.Sprintf(" correlation_id=%s", correlationID)
	}

	// Add message-specific fields
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			fieldsStr += fmt.Sprintf(" %v=%v", fields[i], fields[i+1])
		}
	}

	// Write log entry - errors are intentionally ignored for logging
	entry := fmt.Sprintf("%s [%s] %s%s\n", timestamp, level.String(), msg, fieldsStr)
	//nolint:errcheck // Log write errors are intentionally ignored
	writer.Write([]byte(entry))
}

// DefaultLogger is the package-level default logger.
var DefaultLogger = NewStructuredLogger(os.Stdout)

// SetDefaultLogLevel sets the default logger's level.
func SetDefaultLogLevel(level LogLevel) {
	DefaultLogger.SetLevel(level)
}

// EventLogger provides specialized logging for event processing.
type EventLogger struct {
	logger *StructuredLogger
}

// NewEventLogger creates a logger configured for event processing.
func NewEventLogger(logger *StructuredLogger) *EventLogger {
	if logger == nil {
		logger = DefaultLogger
	}
	return &EventLogger{
		logger: logger.WithFields("component", "eventprocessor"),
	}
}

// LogEventReceived logs when an event is received.
func (e *EventLogger) LogEventReceived(ctx context.Context, eventID, source, mediaType string) {
	e.logger.InfoContext(ctx, "event received",
		"event_id", eventID,
		"source", source,
		"media_type", mediaType,
	)
}

// LogEventProcessed logs when an event is successfully processed.
func (e *EventLogger) LogEventProcessed(ctx context.Context, eventID string, durationMs int64) {
	e.logger.InfoContext(ctx, "event processed",
		"event_id", eventID,
		"duration_ms", durationMs,
	)
}

// LogEventFailed logs when event processing fails.
func (e *EventLogger) LogEventFailed(ctx context.Context, eventID string, err error) {
	e.logger.ErrorContext(ctx, "event processing failed",
		"event_id", eventID,
		"error", err.Error(),
	)
}

// LogDuplicate logs when a duplicate event is detected.
func (e *EventLogger) LogDuplicate(ctx context.Context, eventID, reason string) {
	e.logger.DebugContext(ctx, "duplicate event skipped",
		"event_id", eventID,
		"reason", reason,
	)
}

// LogDLQEntry logs when an event is sent to the DLQ.
func (e *EventLogger) LogDLQEntry(ctx context.Context, eventID string, err error, retryCount int) {
	e.logger.WarnContext(ctx, "event sent to DLQ",
		"event_id", eventID,
		"error", err.Error(),
		"retry_count", retryCount,
	)
}

// LogBatchFlush logs batch flush operations.
func (e *EventLogger) LogBatchFlush(ctx context.Context, count int, durationMs int64) {
	e.logger.InfoContext(ctx, "batch flush completed",
		"event_count", count,
		"duration_ms", durationMs,
	)
}
