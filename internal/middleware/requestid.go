// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/logging"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// RequestID middleware generates a unique ID for each request
// and adds it to both the response header and request context.
// It also integrates with the logging package for distributed tracing
// by populating both request_id and correlation_id in the context.
func RequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if request already has an ID (from upstream proxy)
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			// Generate new UUID v4
			requestID = uuid.New().String()
		}

		// Add to response header for client visibility
		w.Header().Set("X-Request-ID", requestID)

		// Add to request context for logging and tracing
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Integrate with logging package for structured logging with request tracing
		ctx = logging.ContextWithRequestID(ctx, requestID)
		ctx = logging.ContextWithNewCorrelationID(ctx)

		next(w, r.WithContext(ctx))
	}
}

// GetRequestID extracts the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
