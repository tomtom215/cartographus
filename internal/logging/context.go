// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package logging

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Context keys for logging.
type contextKey string

const (
	// correlationIDKey is the context key for correlation IDs.
	correlationIDKey contextKey = "correlation_id"

	// requestIDKey is the context key for HTTP request IDs.
	requestIDKey contextKey = "request_id"

	// loggerKey is the context key for storing a logger instance.
	loggerKey contextKey = "logger"
)

// GenerateCorrelationID creates a new unique correlation ID.
// Returns the first 8 characters of a UUID for readability.
func GenerateCorrelationID() string {
	return uuid.New().String()[:8]
}

// GenerateRequestID creates a new unique request ID.
// Returns a full UUID for uniqueness across distributed systems.
func GenerateRequestID() string {
	return uuid.New().String()
}

// ContextWithCorrelationID returns a new context with the given correlation ID.
//
//	ctx = logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())
func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// ContextWithNewCorrelationID returns a context with a newly generated correlation ID.
//
//	ctx = logging.ContextWithNewCorrelationID(ctx)
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

// ContextWithRequestID returns a new context with the given request ID.
//
//	ctx = logging.ContextWithRequestID(ctx, requestID)
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// ContextWithNewRequestID returns a context with a newly generated request ID.
func ContextWithNewRequestID(ctx context.Context) context.Context {
	return ContextWithRequestID(ctx, GenerateRequestID())
}

// RequestIDFromContext retrieves the request ID from context.
// Returns empty string if not present.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// ContextWithLogger stores a logger in the context.
// This is useful for passing pre-configured loggers through middleware.
//
//nolint:gocritic // zerolog.Logger is designed to be passed by value
func ContextWithLogger(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext retrieves a logger from context.
// Returns the global logger if no logger is stored in context.
func LoggerFromContext(ctx context.Context) zerolog.Logger {
	if logger, ok := ctx.Value(loggerKey).(zerolog.Logger); ok {
		return logger
	}
	return Logger()
}

// Ctx returns a logger with context values (correlation_id, request_id) automatically added.
// This is the recommended way to log with context in handlers and services.
//
//	logging.Ctx(ctx).Info().Msg("Processing request")
//	// Output: {"level":"info","correlation_id":"abc12345","request_id":"uuid","message":"Processing request"}
func Ctx(ctx context.Context) *zerolog.Logger {
	// Check if a logger is stored in context
	logger := LoggerFromContext(ctx)

	// Create a new logger with context fields
	contextLogger := logger.With().Logger()

	// Add correlation ID if present
	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		contextLogger = contextLogger.With().Str("correlation_id", correlationID).Logger()
	}

	// Add request ID if present
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		contextLogger = contextLogger.With().Str("request_id", requestID).Logger()
	}

	return &contextLogger
}

// CtxWith returns a logger context builder with context values pre-populated.
// Use this when you need to add additional fields beyond the standard context fields.
//
//	logger := logging.CtxWith(ctx).Str("user_id", uid).Logger()
//	logger.Info().Msg("User action")
func CtxWith(ctx context.Context) zerolog.Context {
	logger := LoggerFromContext(ctx)
	logCtx := logger.With()

	// Add correlation ID if present
	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		logCtx = logCtx.Str("correlation_id", correlationID)
	}

	// Add request ID if present
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		logCtx = logCtx.Str("request_id", requestID)
	}

	return logCtx
}

// CtxDebug starts a debug level message with context fields.
// Shorthand for Ctx(ctx).Debug().
func CtxDebug(ctx context.Context) *zerolog.Event {
	return Ctx(ctx).Debug()
}

// CtxInfo starts an info level message with context fields.
// Shorthand for Ctx(ctx).Info().
func CtxInfo(ctx context.Context) *zerolog.Event {
	return Ctx(ctx).Info()
}

// CtxWarn starts a warn level message with context fields.
// Shorthand for Ctx(ctx).Warn().
func CtxWarn(ctx context.Context) *zerolog.Event {
	return Ctx(ctx).Warn()
}

// CtxError starts an error level message with context fields.
// Shorthand for Ctx(ctx).Error().
func CtxError(ctx context.Context) *zerolog.Event {
	return Ctx(ctx).Error()
}

// CtxErr starts an error level message with context fields and the error.
// Shorthand for Ctx(ctx).Err(err).
func CtxErr(ctx context.Context, err error) *zerolog.Event {
	return Ctx(ctx).Err(err)
}

// WithComponent creates a child logger with a component field.
// Use this to create component-specific loggers.
//
//	syncLogger := logging.WithComponent("sync")
//	syncLogger.Info().Msg("Sync started")
func WithComponent(component string) zerolog.Logger {
	return With().Str("component", component).Logger()
}

// WithService creates a child logger with a service field.
// Use this to identify the service in distributed systems.
//
//	serviceLogger := logging.WithService("api")
func WithService(service string) zerolog.Logger {
	return With().Str("service", service).Logger()
}
