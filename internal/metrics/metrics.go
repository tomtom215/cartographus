// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus Metrics Integration for Production Observability
// This package provides comprehensive instrumentation for:
// - Database query performance (DuckDB)
// - API endpoint latency and throughput
// - Sync operation metrics
// - Cache efficiency
// - WebSocket connections

var (
	// Database Metrics
	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "duckdb_query_duration_seconds",
			Help:    "Duration of DuckDB queries in seconds",
			Buckets: prometheus.DefBuckets, // 0.005s, 0.01s, 0.025s, 0.05s, 0.1s, 0.25s, 0.5s, 1s, 2.5s, 5s, 10s
		},
		[]string{"operation", "table"},
	)

	DBQueryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "duckdb_query_errors_total",
			Help: "Total number of DuckDB query errors",
		},
		[]string{"operation", "table", "error_type"},
	)

	DBConnectionPoolSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "duckdb_connection_pool_size",
			Help: "Current number of database connections in use",
		},
	)

	DBSpatialOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "duckdb_spatial_operations_total",
			Help: "Total number of spatial operations (ST_* functions)",
		},
		[]string{"operation_type"}, // "point", "distance", "hilbert", "mvt", "envelope"
	)

	// Vector Tile Cache Metrics (MEDIUM-1)
	TileCacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tile_cache_hits_total",
			Help: "Total number of vector tile cache hits",
		},
	)

	TileCacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tile_cache_misses_total",
			Help: "Total number of vector tile cache misses",
		},
	)

	TileCacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tile_cache_entries",
			Help: "Current number of cached vector tiles",
		},
	)

	TileCacheDataVersion = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tile_cache_data_version",
			Help: "Current tile cache data version (increments on data changes)",
		},
	)

	// API Endpoint Metrics
	APIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}, // Optimized for API latency
		},
		[]string{"method", "endpoint"},
	)

	APIActiveRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "api_active_requests",
			Help: "Current number of active API requests",
		},
	)

	APIRateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_rate_limit_hits_total",
			Help: "Total number of rate limit rejections",
		},
		[]string{"endpoint"},
	)

	// Sync Operation Metrics
	SyncDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "sync_duration_seconds",
			Help:    "Duration of sync operations in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600}, // Sync operations can take minutes
		},
	)

	SyncRecordsProcessed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sync_records_processed_total",
			Help: "Total number of playback records processed during sync",
		},
	)

	SyncErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sync_errors_total",
			Help: "Total number of sync errors",
		},
		[]string{"error_type"}, // "tautulli_api", "database", "geolocation", "validation"
	)

	SyncLastSuccess = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sync_last_success_timestamp",
			Help: "Unix timestamp of last successful sync",
		},
	)

	SyncBatchSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "sync_batch_size",
			Help:    "Number of records in sync batches",
			Buckets: []float64{10, 50, 100, 250, 500, 1000, 5000, 10000},
		},
	)

	// Geolocation Metrics (MEDIUM-2)
	GeolocationBatchSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "geolocation_batch_size",
			Help:    "Number of IPs in geolocation batch lookups",
			Buckets: []float64{1, 5, 10, 20, 50, 100, 200, 500},
		},
	)

	GeolocationCacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "geolocation_cache_hits_total",
			Help: "Total number of geolocation cache hits (DB)",
		},
	)

	GeolocationCacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "geolocation_cache_misses_total",
			Help: "Total number of geolocation cache misses (API fetch required)",
		},
	)

	GeolocationAPICallDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "geolocation_api_call_duration_seconds",
			Help:    "Duration of Tautulli geolocation API calls",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Cache Metrics (General)
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"}, // "analytics", "tile", "geolocation"
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	CacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_entries",
			Help: "Current number of cached entries",
		},
		[]string{"cache_type"},
	)

	CacheEvictions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_evictions_total",
			Help: "Total number of cache evictions (TTL expiry)",
		},
		[]string{"cache_type"},
	)

	// WebSocket Metrics
	WSConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_connections",
			Help: "Current number of active WebSocket connections",
		},
	)

	WSMessagesSent = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "websocket_messages_sent_total",
			Help: "Total number of WebSocket messages sent",
		},
	)

	WSMessagesReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "websocket_messages_received_total",
			Help: "Total number of WebSocket messages received",
		},
	)

	WSErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_errors_total",
			Help: "Total number of WebSocket errors",
		},
		[]string{"error_type"},
	)

	// Circuit Breaker Metrics
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
		},
		[]string{"name"},
	)

	CircuitBreakerRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "Total number of requests through circuit breaker",
		},
		[]string{"name", "result"}, // result: "success", "failure", "rejected"
	)

	CircuitBreakerConsecutiveFailures = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_consecutive_failures",
			Help: "Current number of consecutive failures",
		},
		[]string{"name"},
	)

	CircuitBreakerTransitions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_state_transitions_total",
			Help: "Total number of circuit breaker state transitions",
		},
		[]string{"name", "from_state", "to_state"},
	)

	// Dead Letter Queue Metrics (Phase 5)
	DLQEntriesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "dlq_entries_total",
			Help: "Current number of entries in the Dead Letter Queue",
		},
	)

	DLQEntriesByCategory = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dlq_entries_by_category",
			Help: "Current number of DLQ entries by error category",
		},
		[]string{"category"}, // connection, timeout, validation, database, capacity, unknown
	)

	DLQMessagesAdded = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "dlq_messages_added_total",
			Help: "Total number of messages added to the DLQ",
		},
	)

	DLQMessagesRemoved = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "dlq_messages_removed_total",
			Help: "Total number of messages removed from the DLQ (successfully reprocessed)",
		},
	)

	DLQMessagesExpired = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "dlq_messages_expired_total",
			Help: "Total number of messages expired from the DLQ",
		},
	)

	DLQRetryAttempts = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "dlq_retry_attempts_total",
			Help: "Total number of retry attempts for DLQ messages",
		},
	)

	DLQRetrySuccesses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "dlq_retry_successes_total",
			Help: "Total number of successful DLQ message retries",
		},
	)

	DLQRetryFailures = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "dlq_retry_failures_total",
			Help: "Total number of failed DLQ message retries",
		},
	)

	DLQOldestEntryAge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "dlq_oldest_entry_age_seconds",
			Help: "Age of the oldest entry in the DLQ in seconds",
		},
	)

	// NATS Event Processing Metrics (Phase 6)
	NATSMessagesPublished = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nats_messages_published_total",
			Help: "Total number of messages published to NATS",
		},
	)

	NATSMessagesConsumed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nats_messages_consumed_total",
			Help: "Total number of messages consumed from NATS",
		},
	)

	NATSMessagesProcessed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nats_messages_processed_total",
			Help: "Total number of messages successfully processed",
		},
	)

	NATSMessagesDeduplicated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nats_messages_deduplicated_total",
			Help: "Total number of messages skipped due to deduplication",
		},
	)

	NATSMessagesParseFailed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nats_messages_parse_failed_total",
			Help: "Total number of messages that failed to parse",
		},
	)

	NATSProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nats_processing_duration_seconds",
			Help:    "Duration of NATS message processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	NATSBatchFlushDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nats_batch_flush_duration_seconds",
			Help:    "Duration of batch flush operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	NATSBatchSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nats_batch_size",
			Help:    "Number of events in each batch flush",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
	)

	NATSQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nats_queue_depth",
			Help: "Current depth of the NATS message queue",
		},
	)

	NATSConsumerLag = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nats_consumer_lag",
			Help: "Number of pending messages in NATS consumer",
		},
	)

	// System Metrics
	AppInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_info",
			Help: "Application version and build information",
		},
		[]string{"version", "go_version"},
	)

	AppUptime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "app_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)
)

// RecordDBQuery records a database query metric
func RecordDBQuery(operation, table string, duration time.Duration, err error) {
	DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
	if err != nil {
		errorType := err.Error()
		// Truncate long error messages
		if len(errorType) > 50 {
			errorType = errorType[:50]
		}
		DBQueryErrors.WithLabelValues(operation, table, errorType).Inc()
	}
}

// RecordAPIRequest records an API request metric
func RecordAPIRequest(method, endpoint, statusCode string, duration time.Duration) {
	APIRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	APIRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordSyncOperation records a sync operation metric
func RecordSyncOperation(duration time.Duration, recordsProcessed int, err error) {
	SyncDuration.Observe(duration.Seconds())
	SyncRecordsProcessed.Add(float64(recordsProcessed))
	if err != nil {
		errorType := "unknown"
		// Categorize error types
		errorMsg := err.Error()
		if len(errorMsg) > 0 {
			switch {
			case contains(errorMsg, "tautulli"):
				errorType = "tautulli_api"
			case contains(errorMsg, "database"):
				errorType = "database"
			case contains(errorMsg, "geolocation"):
				errorType = "geolocation"
			default:
				errorType = "other"
			}
		}
		SyncErrors.WithLabelValues(errorType).Inc()
	} else {
		// Update last success timestamp
		SyncLastSuccess.Set(float64(time.Now().Unix()))
	}
}

// TrackActiveRequest tracks active API requests
func TrackActiveRequest(inc bool) {
	if inc {
		APIActiveRequests.Inc()
	} else {
		APIActiveRequests.Dec()
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// RecordDLQEntry records a message being added to the DLQ
func RecordDLQEntry(category string) {
	DLQMessagesAdded.Inc()
	DLQEntriesByCategory.WithLabelValues(category).Inc()
}

// RecordDLQRemoval records a message being successfully removed from the DLQ
func RecordDLQRemoval(category string) {
	DLQMessagesRemoved.Inc()
	DLQEntriesByCategory.WithLabelValues(category).Dec()
}

// RecordDLQExpiry records a message expiring from the DLQ
func RecordDLQExpiry(category string) {
	DLQMessagesExpired.Inc()
	DLQEntriesByCategory.WithLabelValues(category).Dec()
}

// RecordDLQRetry records a retry attempt and its outcome
func RecordDLQRetry(success bool) {
	DLQRetryAttempts.Inc()
	if success {
		DLQRetrySuccesses.Inc()
	} else {
		DLQRetryFailures.Inc()
	}
}

// UpdateDLQGauges updates DLQ gauge metrics with current stats
func UpdateDLQGauges(totalEntries int64, oldestEntryAge float64, entriesByCategory map[string]int64) {
	DLQEntriesTotal.Set(float64(totalEntries))
	DLQOldestEntryAge.Set(oldestEntryAge)
	for category, count := range entriesByCategory {
		DLQEntriesByCategory.WithLabelValues(category).Set(float64(count))
	}
}

// RecordNATSPublish records a message being published to NATS
func RecordNATSPublish() {
	NATSMessagesPublished.Inc()
}

// RecordNATSConsume records a message being consumed from NATS
func RecordNATSConsume() {
	NATSMessagesConsumed.Inc()
}

// RecordNATSProcessed records a message being successfully processed
func RecordNATSProcessed() {
	NATSMessagesProcessed.Inc()
}

// RecordNATSDeduplicated records a message being skipped due to deduplication
func RecordNATSDeduplicated() {
	NATSMessagesDeduplicated.Inc()
}

// RecordNATSParseFailed records a message that failed to parse
func RecordNATSParseFailed() {
	NATSMessagesParseFailed.Inc()
}

// RecordNATSProcessingDuration records the duration of message processing
func RecordNATSProcessingDuration(duration time.Duration) {
	NATSProcessingDuration.Observe(duration.Seconds())
}

// RecordNATSBatchFlush records a batch flush operation
func RecordNATSBatchFlush(duration time.Duration, batchSize int) {
	NATSBatchFlushDuration.Observe(duration.Seconds())
	NATSBatchSize.Observe(float64(batchSize))
}

// UpdateNATSQueueDepth updates the NATS queue depth gauge
func UpdateNATSQueueDepth(depth int64) {
	NATSQueueDepth.Set(float64(depth))
}

// UpdateNATSConsumerLag updates the NATS consumer lag gauge
func UpdateNATSConsumerLag(lag int64) {
	NATSConsumerLag.Set(float64(lag))
}

// Wrapped Report Metrics (Annual Wrapped / Year-in-Review Feature)
var (
	// WrappedReportGenerationDuration tracks the time to generate wrapped reports
	WrappedReportGenerationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wrapped_report_generation_duration_seconds",
			Help:    "Duration of wrapped report generation in seconds",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}, // Up to 1 minute for complex reports
		},
		[]string{"year", "scope"}, // scope: "user", "server", "batch"
	)

	// WrappedReportsGenerated counts total wrapped reports generated
	WrappedReportsGenerated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wrapped_reports_generated_total",
			Help: "Total number of wrapped reports generated",
		},
		[]string{"year", "scope"}, // scope: "user", "server"
	)

	// WrappedReportGenerationErrors counts errors during report generation
	WrappedReportGenerationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wrapped_report_generation_errors_total",
			Help: "Total number of wrapped report generation errors",
		},
		[]string{"year", "error_type"}, // error_type: "no_data", "query_failed", "save_failed"
	)

	// WrappedReportBatchSize tracks the number of reports generated in batch operations
	WrappedReportBatchSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wrapped_report_batch_size",
			Help:    "Number of wrapped reports generated in batch operations",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
	)

	// WrappedReportCacheHits counts successful cache hits for wrapped reports
	WrappedReportCacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wrapped_report_cache_hits_total",
			Help: "Total number of wrapped report cache hits",
		},
		[]string{"year"},
	)

	// WrappedReportCacheMisses counts cache misses for wrapped reports
	WrappedReportCacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wrapped_report_cache_misses_total",
			Help: "Total number of wrapped report cache misses",
		},
		[]string{"year"},
	)

	// WrappedShareTokensCreated counts share tokens created for wrapped reports
	WrappedShareTokensCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wrapped_share_tokens_created_total",
			Help: "Total number of share tokens created for wrapped reports",
		},
	)

	// WrappedShareTokenAccess counts accesses via share tokens
	WrappedShareTokenAccess = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wrapped_share_token_access_total",
			Help: "Total number of wrapped report accesses via share tokens",
		},
	)

	// WrappedLeaderboardQueries counts leaderboard queries
	WrappedLeaderboardQueries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wrapped_leaderboard_queries_total",
			Help: "Total number of wrapped leaderboard queries",
		},
		[]string{"year"},
	)

	// WrappedActiveYear tracks which years have active reports
	WrappedActiveYear = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wrapped_active_year_reports",
			Help: "Number of reports available for each year",
		},
		[]string{"year"},
	)
)

// RecordWrappedGeneration records a wrapped report generation
func RecordWrappedGeneration(year int, scope string, duration time.Duration, err error) {
	yearStr := strconv.Itoa(year)
	WrappedReportGenerationDuration.WithLabelValues(yearStr, scope).Observe(duration.Seconds())
	if err != nil {
		errorType := "unknown"
		errorMsg := err.Error()
		switch {
		case contains(errorMsg, "no playback"):
			errorType = "no_data"
		case contains(errorMsg, "query"), contains(errorMsg, "database"):
			errorType = "query_failed"
		case contains(errorMsg, "save"), contains(errorMsg, "insert"):
			errorType = "save_failed"
		}
		WrappedReportGenerationErrors.WithLabelValues(yearStr, errorType).Inc()
	} else {
		WrappedReportsGenerated.WithLabelValues(yearStr, scope).Inc()
	}
}

// RecordWrappedBatch records a batch generation of wrapped reports
func RecordWrappedBatch(batchSize int) {
	WrappedReportBatchSize.Observe(float64(batchSize))
}

// RecordWrappedCacheHit records a cache hit for wrapped report retrieval
func RecordWrappedCacheHit(year int) {
	WrappedReportCacheHits.WithLabelValues(strconv.Itoa(year)).Inc()
}

// RecordWrappedCacheMiss records a cache miss for wrapped report retrieval
func RecordWrappedCacheMiss(year int) {
	WrappedReportCacheMisses.WithLabelValues(strconv.Itoa(year)).Inc()
}

// RecordWrappedShareTokenCreated records creation of a share token
func RecordWrappedShareTokenCreated() {
	WrappedShareTokensCreated.Inc()
}

// RecordWrappedShareAccess records access via a share token
func RecordWrappedShareAccess() {
	WrappedShareTokenAccess.Inc()
}

// RecordWrappedLeaderboardQuery records a leaderboard query
func RecordWrappedLeaderboardQuery(year int) {
	WrappedLeaderboardQueries.WithLabelValues(strconv.Itoa(year)).Inc()
}

// UpdateWrappedActiveYearReports updates the count of active reports for a year
func UpdateWrappedActiveYearReports(year int, count int64) {
	WrappedActiveYear.WithLabelValues(strconv.Itoa(year)).Set(float64(count))
}

// =============================================================================
// Personal Access Token (PAT) Metrics (v2.5)
// =============================================================================

var (
	// PATOperationsTotal counts PAT operations
	PATOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pat_operations_total",
			Help: "Total number of PAT operations",
		},
		[]string{"operation", "success"},
	)

	// PATValidationsTotal counts PAT validation attempts
	PATValidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pat_validations_total",
			Help: "Total number of PAT validation attempts",
		},
		[]string{"result"},
	)

	// PATActiveTokens tracks the number of active tokens
	PATActiveTokens = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pat_active_tokens",
			Help: "Current number of active (non-revoked, non-expired) PATs",
		},
	)
)

// RecordPATOperation records a PAT operation (create, revoke, regenerate, etc.)
func RecordPATOperation(operation string, success bool) {
	successStr := "true"
	if !success {
		successStr = "false"
	}
	PATOperationsTotal.WithLabelValues(operation, successStr).Inc()
}

// RecordPATValidation records a PAT validation attempt
func RecordPATValidation(result string) {
	PATValidationsTotal.WithLabelValues(result).Inc()
}

// SetPATActiveTokens sets the current count of active tokens
func SetPATActiveTokens(count int64) {
	PATActiveTokens.Set(float64(count))
}
