// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package middleware provides HTTP middleware components for the application.

This package implements infrastructure middleware for compression, performance
monitoring, request ID tracking, and Prometheus metrics integration. These
components work alongside the authentication middleware to create a complete
middleware stack for HTTP request processing.

Key Components:

  - Compression: Gzip compression for responses >1KB
  - Performance Monitor: Request latency tracking with percentile calculations
  - Request ID: UUID-based request tracking for distributed tracing
  - Prometheus Metrics: HTTP request/response instrumentation

Middleware Stack:

The typical middleware stack for an endpoint is:

	http.HandleFunc("/api/v1/endpoint",
	    auth.CORS(                           // Layer 1: CORS headers
	        auth.RateLimit(                  // Layer 2: Rate limiting
	            middleware.PrometheusMetrics( // Layer 3: Metrics
	                middleware.Compression(    // Layer 4: Gzip
	                    middleware.RequestID(  // Layer 5: Request tracking
	                        handler,           // Layer 6: Business logic
	                    ),
	                ),
	            ),
	        ),
	    ),
	)

Usage Example - Compression:

	import "github.com/tomtom215/cartographus/internal/middleware"

	// Wrap handler with gzip compression
	http.HandleFunc("/api/v1/data",
	    middleware.Compression(handler),
	)

	// Responses >1KB are automatically compressed
	// Accept-Encoding: gzip header is required

Usage Example - Performance Monitoring:

	// Create performance monitor
	perfMon := middleware.NewPerformanceMonitor()

	// Wrap handler
	http.HandleFunc("/api/v1/stats",
	    perfMon.Middleware(handler),
	)

	// Get performance statistics
	stats := perfMon.GetStats()
	fmt.Printf("p50: %v, p95: %v, p99: %v\n",
	    stats.P50, stats.P95, stats.P99)

Usage Example - Request ID:

	// Request ID middleware
	http.HandleFunc("/api/v1/logs",
	    middleware.RequestID(handler),
	)

	// Access request ID in handler
	func handler(w http.ResponseWriter, r *http.Request) {
	    requestID := r.Context().Value(middleware.RequestIDKey).(string)
	    log.Printf("[%s] Processing request", requestID)
	}

Performance Characteristics:

  - Compression: 70-90% size reduction for JSON (text/json mime types)
  - Compression overhead: ~1-2ms for typical responses
  - Metrics overhead: <0.1ms per request
  - Request ID overhead: <0.01ms (UUID generation)
  - Performance monitor: Lock-free ring buffer for latency samples

Compression Details:

The compression middleware:
  - Only compresses responses >1KB (configurable threshold)
  - Supports gzip encoding (Accept-Encoding: gzip)
  - Applies to text/json/javascript/xml mime types
  - Automatically sets Content-Encoding header
  - Flushes compressed data for streaming responses

Performance Monitor:

The performance monitor tracks:
  - Request count and error rate
  - Latency percentiles (p50, p95, p99)
  - Rolling window of 1000 most recent requests
  - Thread-safe concurrent access with RWMutex

Thread Safety:

All middleware components are thread-safe:
  - Compression uses per-request gzip writers
  - Performance monitor uses sync.RWMutex
  - Request ID uses context.Context (immutable)
  - Prometheus metrics use atomic operations

See Also:

  - internal/auth: Authentication middleware
  - internal/api: HTTP handlers wrapped by middleware
  - internal/metrics: Prometheus metrics definitions
*/
package middleware
