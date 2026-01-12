// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package metrics provides Prometheus metrics collection and export for observability.

This package implements comprehensive application instrumentation using the Prometheus
client library, exposing metrics for monitoring performance, errors, and system health.

# Overview

The package provides metrics for:
  - HTTP request latency and throughput
  - Database query performance
  - Sync operation statistics
  - Circuit breaker state transitions
  - Cache hit/miss rates
  - WebSocket connection counts
  - Plex integration performance (v1.39+)

# Metrics Endpoint

Metrics are exposed at the /metrics endpoint in Prometheus text format:

	curl http://localhost:3857/metrics

# Available Metrics

HTTP Metrics:
  - http_requests_total: Total HTTP requests (counter)
    Labels: method, endpoint, status
  - http_request_duration_seconds: Request latency (histogram)
    Labels: method, endpoint
    Buckets: .001, .005, .01, .05, .1, .5, 1, 5, 10
  - http_requests_in_flight: Active requests (gauge)

Database Metrics:
  - db_query_duration_seconds: Query execution time (histogram)
    Labels: operation (select, insert, update, delete)
  - db_connections_active: Active database connections (gauge)
  - db_query_errors_total: Failed queries (counter)
    Labels: operation, error_type

Sync Metrics:
  - sync_duration_seconds: Sync operation duration (histogram)
    Buckets: 1, 5, 10, 30, 60, 120, 300
  - sync_records_total: Records processed per sync (counter)
    Labels: source (tautulli, plex)
  - sync_errors_total: Failed syncs (counter)
    Labels: source, error_type
  - sync_last_success_timestamp: Unix timestamp of last successful sync (gauge)

Circuit Breaker Metrics:
  - circuit_breaker_state: Current state (gauge)
    Labels: name
    Values: 0=closed, 1=open, 2=half-open
  - circuit_breaker_failures_total: Failure counts (counter)
    Labels: name, state
  - circuit_breaker_successes_total: Success counts (counter)
    Labels: name, state

Cache Metrics:
  - cache_hits_total: Cache hits (counter)
    Labels: cache_type
  - cache_misses_total: Cache misses (counter)
    Labels: cache_type
  - cache_evictions_total: Cache evictions (counter)
    Labels: cache_type
  - cache_size_bytes: Current cache size (gauge)
    Labels: cache_type

WebSocket Metrics:
  - websocket_connections_active: Active connections (gauge)
  - websocket_messages_sent_total: Messages sent (counter)
    Labels: message_type
  - websocket_messages_received_total: Messages received (counter)
    Labels: message_type

Plex Metrics (v1.39+):
  - plex_websocket_state: WebSocket connection state (gauge)
    Values: 0=disconnected, 1=connected
  - plex_transcode_sessions_active: Active transcode sessions (gauge)
  - plex_buffer_health_critical_sessions: Sessions with critical buffer (gauge)
  - plex_api_calls_total: Plex API calls (counter)
    Labels: endpoint, status

# Usage Example

Basic setup in main.go:

	import (
	    "github.com/tomtom215/cartographus/internal/metrics"
	    "github.com/prometheus/client_golang/prometheus/promhttp"
	)

	func main() {
	    // Initialize metrics
	    metrics.Init()

	    // Register metrics endpoint
	    http.Handle("/metrics", promhttp.Handler())

	    // Record metrics
	    metrics.RecordHTTPRequest("GET", "/api/v1/stats", 200, 0.023)
	    metrics.RecordDatabaseQuery("select", 0.005)
	    metrics.RecordSyncOperation("tautulli", 1500, 32.5)
	}

Recording HTTP metrics with middleware:

	func MetricsMiddleware(next http.Handler) http.Handler {
	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	        start := time.Now()

	        // Wrap ResponseWriter to capture status code
	        rw := &responseWriter{ResponseWriter: w, statusCode: 200}

	        next.ServeHTTP(rw, r)

	        // Record metrics
	        duration := time.Since(start).Seconds()
	        metrics.RecordHTTPRequest(r.Method, r.URL.Path, rw.statusCode, duration)
	    })
	}

Recording database query metrics:

	func (db *Database) Query(ctx context.Context, sql string, args ...interface{}) (*sql.Rows, error) {
	    start := time.Now()
	    rows, err := db.conn.QueryContext(ctx, sql, args...)
	    duration := time.Since(start).Seconds()

	    metrics.RecordDatabaseQuery("select", duration)
	    if err != nil {
	        metrics.IncrementDatabaseErrors("select", "query_error")
	    }

	    return rows, err
	}

# Prometheus Configuration

Example prometheus.yml configuration:

	scrape_configs:
	  - job_name: 'cartographus'
	    static_configs:
	      - targets: ['localhost:3857']
	    metrics_path: '/metrics'
	    scrape_interval: 15s

# Grafana Dashboards

The metrics support Grafana dashboards with panels for:

  - Request rate (queries per second)
  - Request latency (p50, p95, p99 percentiles)
  - Error rate (errors per second by endpoint)
  - Database query performance (duration distribution)
  - Sync operation statistics (records/sec, duration trends)
  - Circuit breaker state visualization
  - Cache hit rate and efficiency

Example PromQL queries:

	# HTTP request rate
	rate(http_requests_total[5m])

	# HTTP p95 latency
	histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

	# Database query rate
	rate(db_query_duration_seconds_count[5m])

	# Cache hit rate
	sum(rate(cache_hits_total[5m])) / (sum(rate(cache_hits_total[5m])) + sum(rate(cache_misses_total[5m])))

	# Sync records per minute
	rate(sync_records_total[1m]) * 60

# Performance Impact

Metrics collection overhead:
  - Counter increment: ~100ns per operation
  - Histogram observation: ~500ns per operation
  - Memory overhead: ~5KB per metric time series
  - Total overhead: <1% CPU, <10MB RAM for typical workloads

# Thread Safety

All metric recording functions are thread-safe and designed for concurrent use
from multiple goroutines. The Prometheus client library handles synchronization
internally.

# Cardinality Management

To prevent high cardinality issues:

  - Endpoint labels are normalized (no query parameters)
  - Status codes are grouped (2xx, 3xx, 4xx, 5xx)
  - Error types are limited to predefined constants
  - User-specific labels are avoided

Maximum cardinality per metric:
  - http_requests_total: ~500 series (10 methods × 50 endpoints × 5 statuses)
  - db_query_duration_seconds: ~20 series (4 operations × 5 buckets)
  - circuit_breaker_state: ~10 series (5 breakers × 3 states)

# Alerting Rules

Example Prometheus alerting rules:

	groups:
	  - name: cartographus
	    rules:
	      - alert: HighErrorRate
	        expr: |
	          sum(rate(http_requests_total{status=~"5.."}[5m]))
	          /
	          sum(rate(http_requests_total[5m]))
	          > 0.05
	        for: 5m
	        annotations:
	          summary: "High error rate: {{ $value }}%"

	      - alert: SlowDatabaseQueries
	        expr: |
	          histogram_quantile(0.95,
	            rate(db_query_duration_seconds_bucket[5m]))
	          > 1
	        for: 5m
	        annotations:
	          summary: "p95 query latency: {{ $value }}s"

	      - alert: CircuitBreakerOpen
	        expr: circuit_breaker_state > 0
	        for: 2m
	        annotations:
	          summary: "Circuit breaker open for {{ $labels.name }}"

# Debugging

Enable metrics debugging with LOG_LEVEL=debug:

	# View all registered metrics
	curl http://localhost:3857/metrics | grep "# HELP"

	# Check specific metric
	curl http://localhost:3857/metrics | grep http_requests_total

	# Validate Prometheus format
	promtool check metrics http://localhost:3857/metrics

# Best Practices

When adding new metrics:

 1. Use appropriate metric types:
    - Counter: Monotonically increasing values (requests, errors)
    - Gauge: Point-in-time values (connections, queue size)
    - Histogram: Distribution of values (latency, size)

 2. Choose descriptive names:
    - Use underscore separation: http_request_duration_seconds
    - Include units: _seconds, _bytes, _total
    - Follow Prometheus naming conventions

 3. Add helpful documentation:
    - Include HELP text describing the metric
    - Document all label dimensions
    - Specify units in metric name

 4. Minimize cardinality:
    - Avoid high-cardinality labels (user IDs, timestamps)
    - Normalize endpoint paths
    - Use fixed error type constants

 5. Test performance impact:
    - Benchmark metric recording overhead
    - Monitor memory usage with many time series
    - Validate scrape duration <1s

# See Also

  - internal/middleware: HTTP middleware with metrics integration
  - internal/database: Database metrics recording
  - internal/sync: Sync operation metrics
  - https://prometheus.io/docs/practices/naming/: Metric naming conventions
  - https://prometheus.io/docs/practices/instrumentation/: Instrumentation guide
*/
package metrics
