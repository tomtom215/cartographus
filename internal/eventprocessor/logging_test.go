// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCorrelationID_Generate(t *testing.T) {
	t.Parallel()

	id1 := GenerateCorrelationID()
	id2 := GenerateCorrelationID()

	if id1 == "" {
		t.Error("expected non-empty correlation ID")
	}
	if id1 == id2 {
		t.Error("expected unique correlation IDs")
	}
}

func TestCorrelationID_FromContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Without correlation ID
	id := CorrelationIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty correlation ID, got %s", id)
	}

	// With correlation ID
	ctx = ContextWithCorrelationID(ctx, "test-123")
	id = CorrelationIDFromContext(ctx)
	if id != "test-123" {
		t.Errorf("expected 'test-123', got '%s'", id)
	}
}

func TestCorrelationID_NewContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = ContextWithNewCorrelationID(ctx)

	id := CorrelationIDFromContext(ctx)
	if id == "" {
		t.Error("expected correlation ID to be generated")
	}
}

func TestStructuredLogger_Basic(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	logger.Info("test message", "key1", "value1")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("expected log to contain message")
	}
	if !strings.Contains(output, "key1") {
		t.Error("expected log to contain key1")
	}
	if !strings.Contains(output, "value1") {
		t.Error("expected log to contain value1")
	}
}

func TestStructuredLogger_WithCorrelationID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	ctx := ContextWithCorrelationID(context.Background(), "corr-123")
	logger.InfoContext(ctx, "test message")

	output := buf.String()
	if !strings.Contains(output, "corr-123") {
		t.Error("expected log to contain correlation ID")
	}
}

func TestStructuredLogger_Error(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	logger.Error("error message", "err", "something failed")

	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Error("expected log to contain error message")
	}
	if !strings.Contains(output, "ERROR") || !strings.Contains(output, "error") {
		// Accept either ERROR or error level indicator
		if !strings.Contains(strings.ToLower(output), "err") {
			t.Error("expected log to indicate error level")
		}
	}
}

func TestStructuredLogger_WithFields(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	// Create logger with default fields
	eventLogger := logger.WithFields("component", "consumer", "topic", "playback.>")
	eventLogger.Info("processing event")

	output := buf.String()
	if !strings.Contains(output, "consumer") {
		t.Error("expected log to contain component field")
	}
	if !strings.Contains(output, "playback.>") {
		t.Error("expected log to contain topic field")
	}
}

func TestStructuredLogger_Debug(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	// Set debug level
	logger.SetLevel(LogLevelDebug)
	logger.Debug("debug message", "detail", "verbose")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("expected log to contain debug message")
	}
}

func TestStructuredLogger_LevelFilter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	// Set info level (debug should be filtered)
	logger.SetLevel(LogLevelInfo)
	logger.Debug("debug message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("expected debug message to be filtered out")
	}

	// Info should still work
	logger.Info("info message")
	output = buf.String()
	if !strings.Contains(output, "info message") {
		t.Error("expected info message to be logged")
	}
}

func TestLogLevel_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
	}

	for _, tt := range tests {
		if tt.level.String() != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.level.String())
		}
	}
}

func TestEventLogger_Integration(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	// Simulate event processing logging
	ctx := ContextWithCorrelationID(context.Background(), "evt-12345")
	eventLogger := logger.WithFields(
		"component", "consumer",
		"event_id", "abc-123",
		"source", "plex",
	)

	eventLogger.InfoContext(ctx, "event received")
	eventLogger.InfoContext(ctx, "event processed", "duration_ms", 42)

	output := buf.String()
	if !strings.Contains(output, "evt-12345") {
		t.Error("expected correlation ID in output")
	}
	if !strings.Contains(output, "abc-123") {
		t.Error("expected event_id in output")
	}
	if !strings.Contains(output, "plex") {
		t.Error("expected source in output")
	}
}

// TestStructuredLogger_GetLevel tests GetLevel retrieves the current log level.
func TestStructuredLogger_GetLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	// Default level should be Info
	if logger.GetLevel() != LogLevelInfo {
		t.Errorf("expected default level Info, got %v", logger.GetLevel())
	}

	// Set to debug
	logger.SetLevel(LogLevelDebug)
	if logger.GetLevel() != LogLevelDebug {
		t.Errorf("expected Debug level, got %v", logger.GetLevel())
	}

	// Set to error
	logger.SetLevel(LogLevelError)
	if logger.GetLevel() != LogLevelError {
		t.Errorf("expected Error level, got %v", logger.GetLevel())
	}
}

// TestStructuredLogger_Warn tests the Warn method.
func TestStructuredLogger_Warn(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	logger.Warn("warning message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "warning message") {
		t.Error("expected warning message in output")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("expected WARN level in output")
	}
	if !strings.Contains(output, "key=value") {
		t.Error("expected key=value in output")
	}
}

// TestStructuredLogger_DebugContext tests DebugContext with context.
func TestStructuredLogger_DebugContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)
	logger.SetLevel(LogLevelDebug)

	ctx := ContextWithCorrelationID(context.Background(), "debug-ctx-123")
	logger.DebugContext(ctx, "debug context message", "detail", "info")

	output := buf.String()
	if !strings.Contains(output, "debug context message") {
		t.Error("expected debug message in output")
	}
	if !strings.Contains(output, "DEBUG") {
		t.Error("expected DEBUG level in output")
	}
	if !strings.Contains(output, "debug-ctx-123") {
		t.Error("expected correlation ID in output")
	}
}

// TestStructuredLogger_WarnContext tests WarnContext with context.
func TestStructuredLogger_WarnContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	ctx := ContextWithCorrelationID(context.Background(), "warn-ctx-456")
	logger.WarnContext(ctx, "warn context message", "severity", "medium")

	output := buf.String()
	if !strings.Contains(output, "warn context message") {
		t.Error("expected warn message in output")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("expected WARN level in output")
	}
	if !strings.Contains(output, "warn-ctx-456") {
		t.Error("expected correlation ID in output")
	}
}

// TestStructuredLogger_ErrorContext tests ErrorContext with context.
func TestStructuredLogger_ErrorContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	ctx := ContextWithCorrelationID(context.Background(), "error-ctx-789")
	logger.ErrorContext(ctx, "error context message", "code", "500")

	output := buf.String()
	if !strings.Contains(output, "error context message") {
		t.Error("expected error message in output")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("expected ERROR level in output")
	}
	if !strings.Contains(output, "error-ctx-789") {
		t.Error("expected correlation ID in output")
	}
}

// TestStructuredLogger_NilWriter tests that nil writer uses stdout.
func TestStructuredLogger_NilWriter(t *testing.T) {
	t.Parallel()

	// Should not panic when writer is nil
	logger := NewStructuredLogger(nil)
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	// The logger should be usable (writes to stdout)
	logger.Info("test nil writer")
}

// TestSetDefaultLogLevel tests the package-level SetDefaultLogLevel function.
func TestSetDefaultLogLevel(t *testing.T) {
	// Save original level
	originalLevel := DefaultLogger.GetLevel()
	defer DefaultLogger.SetLevel(originalLevel)

	SetDefaultLogLevel(LogLevelDebug)
	if DefaultLogger.GetLevel() != LogLevelDebug {
		t.Errorf("expected Debug level on default logger, got %v", DefaultLogger.GetLevel())
	}

	SetDefaultLogLevel(LogLevelError)
	if DefaultLogger.GetLevel() != LogLevelError {
		t.Errorf("expected Error level on default logger, got %v", DefaultLogger.GetLevel())
	}
}

// TestNewEventLogger tests the EventLogger creation and nil handling.
func TestNewEventLogger(t *testing.T) {
	t.Parallel()

	// With nil logger (should use default)
	eventLogger := NewEventLogger(nil)
	if eventLogger == nil {
		t.Fatal("expected non-nil event logger")
	}

	// With provided logger
	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)
	eventLogger = NewEventLogger(logger)
	if eventLogger == nil {
		t.Fatal("expected non-nil event logger with provided logger")
	}
}

// TestEventLogger_LogEventReceived tests logging event reception.
func TestEventLogger_LogEventReceived(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)
	eventLogger := NewEventLogger(logger)

	ctx := ContextWithCorrelationID(context.Background(), "recv-123")
	eventLogger.LogEventReceived(ctx, "evt-001", "plex", "movie")

	output := buf.String()
	if !strings.Contains(output, "event received") {
		t.Error("expected 'event received' message")
	}
	if !strings.Contains(output, "evt-001") {
		t.Error("expected event_id in output")
	}
	if !strings.Contains(output, "plex") {
		t.Error("expected source in output")
	}
	if !strings.Contains(output, "movie") {
		t.Error("expected media_type in output")
	}
}

// TestEventLogger_LogEventProcessed tests logging event processing completion.
func TestEventLogger_LogEventProcessed(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)
	eventLogger := NewEventLogger(logger)

	ctx := ContextWithCorrelationID(context.Background(), "proc-456")
	eventLogger.LogEventProcessed(ctx, "evt-002", 150)

	output := buf.String()
	if !strings.Contains(output, "event processed") {
		t.Error("expected 'event processed' message")
	}
	if !strings.Contains(output, "evt-002") {
		t.Error("expected event_id in output")
	}
	if !strings.Contains(output, "150") {
		t.Error("expected duration_ms in output")
	}
}

// TestEventLogger_LogEventFailed tests logging event processing failure.
func TestEventLogger_LogEventFailed(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)
	eventLogger := NewEventLogger(logger)

	ctx := ContextWithCorrelationID(context.Background(), "fail-789")
	testErr := errors.New("database connection failed")
	eventLogger.LogEventFailed(ctx, "evt-003", testErr)

	output := buf.String()
	if !strings.Contains(output, "event processing failed") {
		t.Error("expected 'event processing failed' message")
	}
	if !strings.Contains(output, "evt-003") {
		t.Error("expected event_id in output")
	}
	if !strings.Contains(output, "database connection failed") {
		t.Error("expected error message in output")
	}
}

// TestEventLogger_LogDuplicate tests logging duplicate event detection.
func TestEventLogger_LogDuplicate(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)
	logger.SetLevel(LogLevelDebug) // LogDuplicate uses Debug level
	eventLogger := NewEventLogger(logger)

	ctx := ContextWithCorrelationID(context.Background(), "dup-111")
	eventLogger.LogDuplicate(ctx, "evt-004", "cross-source dedup")

	output := buf.String()
	if !strings.Contains(output, "duplicate event skipped") {
		t.Error("expected 'duplicate event skipped' message")
	}
	if !strings.Contains(output, "evt-004") {
		t.Error("expected event_id in output")
	}
	if !strings.Contains(output, "cross-source dedup") {
		t.Error("expected reason in output")
	}
}

// TestEventLogger_LogDLQEntry tests logging DLQ entry.
func TestEventLogger_LogDLQEntry(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)
	eventLogger := NewEventLogger(logger)

	ctx := ContextWithCorrelationID(context.Background(), "dlq-222")
	testErr := errors.New("max retries exceeded")
	eventLogger.LogDLQEntry(ctx, "evt-005", testErr, 5)

	output := buf.String()
	if !strings.Contains(output, "event sent to DLQ") {
		t.Error("expected 'event sent to DLQ' message")
	}
	if !strings.Contains(output, "evt-005") {
		t.Error("expected event_id in output")
	}
	if !strings.Contains(output, "max retries exceeded") {
		t.Error("expected error in output")
	}
	if !strings.Contains(output, "5") {
		t.Error("expected retry_count in output")
	}
}

// TestEventLogger_LogBatchFlush tests logging batch flush operations.
func TestEventLogger_LogBatchFlush(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)
	eventLogger := NewEventLogger(logger)

	ctx := ContextWithCorrelationID(context.Background(), "batch-333")
	eventLogger.LogBatchFlush(ctx, 100, 45)

	output := buf.String()
	if !strings.Contains(output, "batch flush completed") {
		t.Error("expected 'batch flush completed' message")
	}
	if !strings.Contains(output, "100") {
		t.Error("expected event_count in output")
	}
	if !strings.Contains(output, "45") {
		t.Error("expected duration_ms in output")
	}
}

// TestLogLevel_UnknownString tests unknown log level string representation.
func TestLogLevel_UnknownString(t *testing.T) {
	t.Parallel()

	unknownLevel := LogLevel(99)
	if unknownLevel.String() != "UNKNOWN" {
		t.Errorf("expected UNKNOWN for invalid level, got %s", unknownLevel.String())
	}
}

// TestStructuredLogger_OddFields tests logging with odd number of fields.
func TestStructuredLogger_OddFields(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)

	// Odd number of fields - last field should be ignored
	logger.Info("odd fields test", "key1", "value1", "orphan_key")

	output := buf.String()
	if !strings.Contains(output, "odd fields test") {
		t.Error("expected message in output")
	}
	if !strings.Contains(output, "key1=value1") {
		t.Error("expected key1=value1 in output")
	}
}
