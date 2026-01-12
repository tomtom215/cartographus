// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tomtom215/cartographus/internal/metrics"
)

// PrometheusMetrics creates middleware for recording Prometheus metrics
// Comprehensive API request instrumentation for Prometheus metrics
func PrometheusMetrics(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Track active requests
		metrics.TrackActiveRequest(true)
		defer metrics.TrackActiveRequest(false)

		// Record start time
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapper := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call next handler
		next(wrapper, r)

		// Calculate duration
		duration := time.Since(start)

		// Record metrics
		metrics.RecordAPIRequest(
			r.Method,
			r.URL.Path,
			strconv.Itoa(wrapper.statusCode),
			duration,
		)
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
