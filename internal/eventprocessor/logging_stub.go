// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"io"
	"os"
)

// LogLevel represents logging severity.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	return "INFO"
}

func GenerateCorrelationID() string {
	return ""
}

func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	return ctx
}

func ContextWithNewCorrelationID(ctx context.Context) context.Context {
	return ctx
}

func CorrelationIDFromContext(ctx context.Context) string {
	return ""
}

type StructuredLogger struct {
	writer io.Writer
}

func NewStructuredLogger(w io.Writer) *StructuredLogger {
	if w == nil {
		w = os.Stdout
	}
	return &StructuredLogger{writer: w}
}

func (l *StructuredLogger) SetLevel(level LogLevel)                                             {}
func (l *StructuredLogger) GetLevel() LogLevel                                                  { return LogLevelInfo }
func (l *StructuredLogger) WithFields(fields ...interface{}) *StructuredLogger                  { return l }
func (l *StructuredLogger) Debug(msg string, fields ...interface{})                             {}
func (l *StructuredLogger) Info(msg string, fields ...interface{})                              {}
func (l *StructuredLogger) Warn(msg string, fields ...interface{})                              {}
func (l *StructuredLogger) Error(msg string, fields ...interface{})                             {}
func (l *StructuredLogger) DebugContext(ctx context.Context, msg string, fields ...interface{}) {}
func (l *StructuredLogger) InfoContext(ctx context.Context, msg string, fields ...interface{})  {}
func (l *StructuredLogger) WarnContext(ctx context.Context, msg string, fields ...interface{})  {}
func (l *StructuredLogger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {}

var DefaultLogger = NewStructuredLogger(os.Stdout)

func SetDefaultLogLevel(level LogLevel) {}

type EventLogger struct{}

func NewEventLogger(logger *StructuredLogger) *EventLogger {
	return &EventLogger{}
}

func (e *EventLogger) LogEventReceived(ctx context.Context, eventID, source, mediaType string)    {}
func (e *EventLogger) LogEventProcessed(ctx context.Context, eventID string, durationMs int64)    {}
func (e *EventLogger) LogEventFailed(ctx context.Context, eventID string, err error)              {}
func (e *EventLogger) LogDuplicate(ctx context.Context, eventID, reason string)                   {}
func (e *EventLogger) LogDLQEntry(ctx context.Context, eventID string, err error, retryCount int) {}
func (e *EventLogger) LogBatchFlush(ctx context.Context, count int, durationMs int64)             {}
