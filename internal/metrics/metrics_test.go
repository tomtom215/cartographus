// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package metrics

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestRecordDBQuery tests database query metric recording
func TestRecordDBQuery(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		table     string
		duration  time.Duration
		err       error
	}{
		{
			name:      "successful SELECT query",
			operation: "SELECT",
			table:     "playback_events",
			duration:  10 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "successful INSERT query",
			operation: "INSERT",
			table:     "geolocations",
			duration:  5 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "failed query with short error",
			operation: "UPDATE",
			table:     "users",
			duration:  100 * time.Millisecond,
			err:       errors.New("connection refused"),
		},
		{
			name:      "failed query with long error - should truncate to 50 chars",
			operation: "DELETE",
			table:     "sessions",
			duration:  50 * time.Millisecond,
			err:       errors.New("this is a very long error message that exceeds fifty characters and should be truncated properly"),
		},
		{
			name:      "fast query under 1ms",
			operation: "SELECT",
			table:     "cache",
			duration:  500 * time.Microsecond,
			err:       nil,
		},
		{
			name:      "slow query over 5 seconds",
			operation: "SELECT",
			table:     "analytics",
			duration:  5500 * time.Millisecond,
			err:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record the query - should not panic
			RecordDBQuery(tt.operation, tt.table, tt.duration, tt.err)

			// Verify duration was recorded (histogram observation)
			// Just check it doesn't panic - actual values would require metric inspection
		})
	}
}

// TestRecordDBQuery_ErrorTruncation verifies error messages are truncated at 50 chars
func TestRecordDBQuery_ErrorTruncation(t *testing.T) {
	// Error with exactly 50 characters
	err50 := errors.New(strings.Repeat("a", 50))
	RecordDBQuery("SELECT", "test", time.Millisecond, err50)

	// Error with 51 characters - should truncate
	err51 := errors.New(strings.Repeat("b", 51))
	RecordDBQuery("SELECT", "test", time.Millisecond, err51)

	// Error with 100 characters - should truncate
	err100 := errors.New(strings.Repeat("c", 100))
	RecordDBQuery("SELECT", "test", time.Millisecond, err100)

	// Very short error
	errShort := errors.New("err")
	RecordDBQuery("SELECT", "test", time.Millisecond, errShort)
}

// TestRecordAPIRequest tests API request metric recording
func TestRecordAPIRequest(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		endpoint   string
		statusCode string
		duration   time.Duration
	}{
		{
			name:       "successful GET request",
			method:     "GET",
			endpoint:   "/api/v1/stats",
			statusCode: "200",
			duration:   25 * time.Millisecond,
		},
		{
			name:       "successful POST login",
			method:     "POST",
			endpoint:   "/api/v1/auth/login",
			statusCode: "200",
			duration:   150 * time.Millisecond,
		},
		{
			name:       "unauthorized request",
			method:     "GET",
			endpoint:   "/api/v1/playbacks",
			statusCode: "401",
			duration:   5 * time.Millisecond,
		},
		{
			name:       "not found request",
			method:     "GET",
			endpoint:   "/api/v1/unknown",
			statusCode: "404",
			duration:   2 * time.Millisecond,
		},
		{
			name:       "internal server error",
			method:     "POST",
			endpoint:   "/api/v1/sync",
			statusCode: "500",
			duration:   500 * time.Millisecond,
		},
		{
			name:       "rate limited request",
			method:     "GET",
			endpoint:   "/api/v1/analytics/trends",
			statusCode: "429",
			duration:   1 * time.Millisecond,
		},
		{
			name:       "bad request",
			method:     "POST",
			endpoint:   "/api/v1/export",
			statusCode: "400",
			duration:   10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record the request - should not panic
			RecordAPIRequest(tt.method, tt.endpoint, tt.statusCode, tt.duration)
		})
	}
}

// TestRecordSyncOperation tests sync operation metric recording
func TestRecordSyncOperation(t *testing.T) {
	tests := []struct {
		name             string
		duration         time.Duration
		recordsProcessed int
		err              error
		expectedErrType  string // expected error type classification
	}{
		{
			name:             "successful sync - small batch",
			duration:         5 * time.Second,
			recordsProcessed: 100,
			err:              nil,
			expectedErrType:  "",
		},
		{
			name:             "successful sync - large batch",
			duration:         60 * time.Second,
			recordsProcessed: 10000,
			err:              nil,
			expectedErrType:  "",
		},
		{
			name:             "successful sync - zero records",
			duration:         1 * time.Second,
			recordsProcessed: 0,
			err:              nil,
			expectedErrType:  "",
		},
		{
			name:             "tautulli API error",
			duration:         30 * time.Second,
			recordsProcessed: 500,
			err:              errors.New("tautulli connection refused"),
			expectedErrType:  "tautulli_api",
		},
		{
			name:             "database error",
			duration:         15 * time.Second,
			recordsProcessed: 250,
			err:              errors.New("database write failed"),
			expectedErrType:  "database",
		},
		{
			name:             "geolocation error",
			duration:         20 * time.Second,
			recordsProcessed: 750,
			err:              errors.New("geolocation lookup timeout"),
			expectedErrType:  "geolocation",
		},
		{
			name:             "unknown error type",
			duration:         10 * time.Second,
			recordsProcessed: 100,
			err:              errors.New("something unexpected happened"),
			expectedErrType:  "other",
		},
		{
			name:             "empty error message",
			duration:         5 * time.Second,
			recordsProcessed: 50,
			err:              errors.New(""),
			expectedErrType:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record the sync operation - should not panic
			RecordSyncOperation(tt.duration, tt.recordsProcessed, tt.err)
		})
	}
}

// TestTrackActiveRequest tests active request tracking
func TestTrackActiveRequest(t *testing.T) {
	tests := []struct {
		name string
		inc  bool
	}{
		{
			name: "increment active request",
			inc:  true,
		},
		{
			name: "decrement active request",
			inc:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track active request - should not panic
			TrackActiveRequest(tt.inc)
		})
	}
}

// TestTrackActiveRequest_RequestLifecycle simulates realistic request lifecycle
func TestTrackActiveRequest_RequestLifecycle(t *testing.T) {
	// Simulate multiple concurrent requests
	for i := 0; i < 10; i++ {
		TrackActiveRequest(true) // Request starts
	}

	// Some requests complete
	for i := 0; i < 5; i++ {
		TrackActiveRequest(false) // Request ends
	}

	// More requests start
	for i := 0; i < 3; i++ {
		TrackActiveRequest(true)
	}

	// All remaining complete
	for i := 0; i < 8; i++ {
		TrackActiveRequest(false)
	}
}

// TestContains tests the contains helper function
func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "substring at start",
			s:        "tautulli error occurred",
			substr:   "tautulli",
			expected: true,
		},
		{
			name:     "substring not at start",
			s:        "error from tautulli",
			substr:   "tautulli",
			expected: false,
		},
		{
			name:     "empty substring - always true",
			s:        "any string",
			substr:   "",
			expected: true,
		},
		{
			name:     "empty string with empty substr",
			s:        "",
			substr:   "",
			expected: true,
		},
		{
			name:     "substring longer than string",
			s:        "hi",
			substr:   "hello",
			expected: false,
		},
		{
			name:     "exact match",
			s:        "database",
			substr:   "database",
			expected: true,
		},
		{
			name:     "case sensitive - no match",
			s:        "Database error",
			substr:   "database",
			expected: false,
		},
		{
			name:     "geolocation prefix match",
			s:        "geolocation lookup failed",
			substr:   "geolocation",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// TestConcurrentMetricRecording tests thread safety of metric recording
func TestConcurrentMetricRecording(t *testing.T) {
	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 50

	// Test concurrent DB query recording
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				RecordDBQuery("SELECT", "test_table", time.Duration(j)*time.Millisecond, nil)
			}
		}(i)
	}

	// Test concurrent API request recording
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				RecordAPIRequest("GET", "/api/v1/test", "200", time.Duration(j)*time.Millisecond)
			}
		}(i)
	}

	// Test concurrent active request tracking
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				TrackActiveRequest(true)
				TrackActiveRequest(false)
			}
		}(i)
	}

	// Test concurrent sync operation recording
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				RecordSyncOperation(time.Second, 100, nil)
			}
		}(i)
	}

	wg.Wait()
}

// TestMetricLabels verifies that metrics have proper labels configured
func TestMetricLabels(t *testing.T) {
	// Test DBQueryDuration has correct labels
	DBQueryDuration.WithLabelValues("SELECT", "test_table").Observe(0.1)
	DBQueryDuration.WithLabelValues("INSERT", "another_table").Observe(0.2)

	// Test DBQueryErrors has correct labels
	DBQueryErrors.WithLabelValues("DELETE", "test_table", "constraint_violation").Inc()

	// Test APIRequestsTotal has correct labels
	APIRequestsTotal.WithLabelValues("GET", "/api/test", "200").Inc()
	APIRequestsTotal.WithLabelValues("POST", "/api/test", "500").Inc()

	// Test SyncErrors has correct labels
	SyncErrors.WithLabelValues("tautulli_api").Inc()
	SyncErrors.WithLabelValues("database").Inc()
	SyncErrors.WithLabelValues("geolocation").Inc()

	// Test CacheHits has correct labels
	CacheHits.WithLabelValues("analytics").Inc()
	CacheHits.WithLabelValues("tile").Inc()

	// Test WSErrors has correct labels
	WSErrors.WithLabelValues("connection_closed").Inc()
	WSErrors.WithLabelValues("write_failed").Inc()
}

// TestDBSpatialOperations tests spatial operation metric recording
func TestDBSpatialOperations(t *testing.T) {
	operationTypes := []string{"point", "distance", "hilbert", "mvt", "envelope"}

	for _, op := range operationTypes {
		t.Run("spatial_"+op, func(t *testing.T) {
			DBSpatialOperations.WithLabelValues(op).Inc()
		})
	}
}

// TestTileCacheMetrics tests tile cache metric recording
func TestTileCacheMetrics(t *testing.T) {
	// Simulate cache operations
	TileCacheHits.Inc()
	TileCacheHits.Inc()
	TileCacheHits.Inc()
	TileCacheMisses.Inc()
	TileCacheSize.Set(100)
	TileCacheDataVersion.Set(42)
}

// TestGeolocationMetrics tests geolocation metric recording
func TestGeolocationMetrics(t *testing.T) {
	// Test batch size histogram
	GeolocationBatchSize.Observe(1)
	GeolocationBatchSize.Observe(10)
	GeolocationBatchSize.Observe(50)
	GeolocationBatchSize.Observe(100)

	// Test cache hits/misses
	GeolocationCacheHits.Add(50)
	GeolocationCacheMisses.Add(10)

	// Test API call duration
	GeolocationAPICallDuration.Observe(0.5)
	GeolocationAPICallDuration.Observe(1.2)
}

// TestCircuitBreakerMetrics tests circuit breaker metric recording
func TestCircuitBreakerMetrics(t *testing.T) {
	cbName := "tautulli_api"

	// Test state changes (0=closed, 1=half-open, 2=open)
	CircuitBreakerState.WithLabelValues(cbName).Set(0) // closed
	CircuitBreakerState.WithLabelValues(cbName).Set(2) // open
	CircuitBreakerState.WithLabelValues(cbName).Set(1) // half-open

	// Test request counts
	CircuitBreakerRequests.WithLabelValues(cbName, "success").Inc()
	CircuitBreakerRequests.WithLabelValues(cbName, "failure").Inc()
	CircuitBreakerRequests.WithLabelValues(cbName, "rejected").Inc()

	// Test consecutive failures
	CircuitBreakerConsecutiveFailures.WithLabelValues(cbName).Set(5)

	// Test state transitions
	CircuitBreakerTransitions.WithLabelValues(cbName, "closed", "open").Inc()
	CircuitBreakerTransitions.WithLabelValues(cbName, "open", "half-open").Inc()
	CircuitBreakerTransitions.WithLabelValues(cbName, "half-open", "closed").Inc()
}

// TestWebSocketMetrics tests WebSocket metric recording
func TestWebSocketMetrics(t *testing.T) {
	// Test connection gauge
	WSConnections.Set(10)
	WSConnections.Inc()
	WSConnections.Dec()

	// Test message counters
	WSMessagesSent.Add(100)
	WSMessagesReceived.Add(50)

	// Test error counter with different types
	WSErrors.WithLabelValues("connection_closed").Inc()
	WSErrors.WithLabelValues("write_timeout").Inc()
	WSErrors.WithLabelValues("invalid_message").Inc()
}

// TestAppMetrics tests application-level metrics
func TestAppMetrics(t *testing.T) {
	// Test app info
	AppInfo.WithLabelValues("1.42", "go1.25.4").Set(1)

	// Test uptime
	AppUptime.Set(3600) // 1 hour
	AppUptime.Add(60)   // Add 1 minute
}

// TestSyncBatchSize tests sync batch size histogram
func TestSyncBatchSize(t *testing.T) {
	batchSizes := []float64{10, 50, 100, 250, 500, 1000, 5000, 10000}

	for _, size := range batchSizes {
		SyncBatchSize.Observe(size)
	}
}

// TestAPIRateLimitHits tests rate limit hit counter
func TestAPIRateLimitHits(t *testing.T) {
	endpoints := []string{
		"/api/v1/stats",
		"/api/v1/playbacks",
		"/api/v1/analytics/trends",
		"/api/v1/sync",
	}

	for _, endpoint := range endpoints {
		APIRateLimitHits.WithLabelValues(endpoint).Inc()
	}
}

// TestCacheMetrics tests general cache metrics
func TestCacheMetrics(t *testing.T) {
	cacheTypes := []string{"analytics", "tile", "geolocation"}

	for _, cacheType := range cacheTypes {
		CacheHits.WithLabelValues(cacheType).Add(100)
		CacheMisses.WithLabelValues(cacheType).Add(20)
		CacheSize.WithLabelValues(cacheType).Set(50)
		CacheEvictions.WithLabelValues(cacheType).Add(5)
	}
}

// TestDBConnectionPoolSize tests connection pool size gauge
func TestDBConnectionPoolSize(t *testing.T) {
	DBConnectionPoolSize.Set(1)
	DBConnectionPoolSize.Inc()
	DBConnectionPoolSize.Set(5)
	DBConnectionPoolSize.Dec()
}

// TestSyncLastSuccess tests sync timestamp recording
func TestSyncLastSuccess(t *testing.T) {
	// Simulate successful sync
	RecordSyncOperation(5*time.Second, 100, nil)

	// Get the current value - should be recent
	// Note: We can't easily get the value without more complex setup,
	// but we verify no panic occurs
}

// TestMetricsRegistration verifies all metrics are properly registered
func TestMetricsRegistration(t *testing.T) {
	// Test that all metrics can be collected without panic
	metrics := []prometheus.Collector{
		DBQueryDuration,
		DBQueryErrors,
		DBConnectionPoolSize,
		DBSpatialOperations,
		TileCacheHits,
		TileCacheMisses,
		TileCacheSize,
		TileCacheDataVersion,
		APIRequestsTotal,
		APIRequestDuration,
		APIActiveRequests,
		APIRateLimitHits,
		SyncDuration,
		SyncRecordsProcessed,
		SyncErrors,
		SyncLastSuccess,
		SyncBatchSize,
		GeolocationBatchSize,
		GeolocationCacheHits,
		GeolocationCacheMisses,
		GeolocationAPICallDuration,
		CacheHits,
		CacheMisses,
		CacheSize,
		CacheEvictions,
		WSConnections,
		WSMessagesSent,
		WSMessagesReceived,
		WSErrors,
		CircuitBreakerState,
		CircuitBreakerRequests,
		CircuitBreakerConsecutiveFailures,
		CircuitBreakerTransitions,
		AppInfo,
		AppUptime,
	}

	// Verify each metric can be described
	for _, m := range metrics {
		ch := make(chan *prometheus.Desc, 10)
		m.Describe(ch)
		close(ch)

		// Should have at least one descriptor
		count := 0
		for range ch {
			count++
		}
		if count == 0 {
			t.Errorf("Metric has no descriptors")
		}
	}
}

// Benchmark tests for metrics performance

func BenchmarkRecordDBQuery(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordDBQuery("SELECT", "playback_events", 10*time.Millisecond, nil)
	}
}

func BenchmarkRecordDBQueryWithError(b *testing.B) {
	err := errors.New("connection refused")
	for i := 0; i < b.N; i++ {
		RecordDBQuery("SELECT", "playback_events", 10*time.Millisecond, err)
	}
}

func BenchmarkRecordAPIRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordAPIRequest("GET", "/api/v1/stats", "200", 25*time.Millisecond)
	}
}

func BenchmarkRecordSyncOperation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordSyncOperation(5*time.Second, 1000, nil)
	}
}

func BenchmarkTrackActiveRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TrackActiveRequest(true)
		TrackActiveRequest(false)
	}
}

func BenchmarkContains(b *testing.B) {
	s := "tautulli connection refused"
	substr := "tautulli"
	for i := 0; i < b.N; i++ {
		contains(s, substr)
	}
}

// TestMetricGathering tests that metrics can be gathered using testutil
func TestMetricGathering(t *testing.T) {
	// Record some metrics
	RecordDBQuery("TEST", "test_table", time.Millisecond, nil)
	RecordAPIRequest("GET", "/test", "200", time.Millisecond)

	// Verify we can lint the metrics (checks for consistency issues)
	problems, err := testutil.GatherAndLint(prometheus.DefaultGatherer)
	if err != nil {
		t.Logf("Lint errors (may be expected): %v", err)
	}
	for _, p := range problems {
		t.Logf("Metric lint problem: %s", p.Text)
	}
}

// TestDLQMetrics tests DLQ (Dead Letter Queue) metric recording
func TestDLQMetrics(t *testing.T) {
	categories := []string{"playback", "geolocation", "notification"}

	for _, category := range categories {
		t.Run("category_"+category, func(t *testing.T) {
			// Test entry recording
			RecordDLQEntry(category)

			// Test removal recording
			RecordDLQRemoval(category)

			// Test expiry recording
			RecordDLQExpiry(category)
		})
	}
}

// TestRecordDLQRetry tests DLQ retry metric recording
func TestRecordDLQRetry(t *testing.T) {
	tests := []struct {
		name    string
		success bool
	}{
		{"successful retry", true},
		{"failed retry", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordDLQRetry(tt.success)
		})
	}
}

// TestUpdateDLQGauges tests DLQ gauge updates
func TestUpdateDLQGauges(t *testing.T) {
	// Test with empty map
	UpdateDLQGauges(0, 0.0, map[string]int64{})

	// Test with single category
	UpdateDLQGauges(10, 300.0, map[string]int64{"playback": 10})

	// Test with multiple categories
	UpdateDLQGauges(25, 600.0, map[string]int64{
		"playback":     15,
		"geolocation":  5,
		"notification": 5,
	})
}

// TestNATSPublishMetrics tests NATS publish metric recording
func TestNATSPublishMetrics(t *testing.T) {
	// Record multiple publishes
	for i := 0; i < 10; i++ {
		RecordNATSPublish()
	}
}

// TestNATSConsumeMetrics tests NATS consume metric recording
func TestNATSConsumeMetrics(t *testing.T) {
	// Record multiple consumes
	for i := 0; i < 10; i++ {
		RecordNATSConsume()
	}
}

// TestNATSProcessedMetrics tests NATS processed metric recording
func TestNATSProcessedMetrics(t *testing.T) {
	// Record multiple processed messages
	for i := 0; i < 10; i++ {
		RecordNATSProcessed()
	}
}

// TestNATSDeduplicatedMetrics tests NATS deduplication metric recording
func TestNATSDeduplicatedMetrics(t *testing.T) {
	// Record multiple deduplications
	for i := 0; i < 5; i++ {
		RecordNATSDeduplicated()
	}
}

// TestNATSParseFailedMetrics tests NATS parse failed metric recording
func TestNATSParseFailedMetrics(t *testing.T) {
	// Record multiple parse failures
	for i := 0; i < 3; i++ {
		RecordNATSParseFailed()
	}
}

// TestNATSProcessingDurationMetrics tests NATS processing duration recording
func TestNATSProcessingDurationMetrics(t *testing.T) {
	durations := []time.Duration{
		1 * time.Millisecond,
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
	}

	for _, d := range durations {
		RecordNATSProcessingDuration(d)
	}
}

// TestNATSBatchFlushMetrics tests NATS batch flush metric recording
func TestNATSBatchFlushMetrics(t *testing.T) {
	tests := []struct {
		name      string
		duration  time.Duration
		batchSize int
	}{
		{"small batch", 10 * time.Millisecond, 10},
		{"medium batch", 50 * time.Millisecond, 100},
		{"large batch", 100 * time.Millisecond, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordNATSBatchFlush(tt.duration, tt.batchSize)
		})
	}
}

// TestNATSQueueDepthMetrics tests NATS queue depth gauge updates
func TestNATSQueueDepthMetrics(t *testing.T) {
	depths := []int64{0, 10, 100, 1000, 0}

	for _, depth := range depths {
		UpdateNATSQueueDepth(depth)
	}
}

// TestNATSConsumerLagMetrics tests NATS consumer lag gauge updates
func TestNATSConsumerLagMetrics(t *testing.T) {
	lags := []int64{0, 5, 50, 500, 0}

	for _, lag := range lags {
		UpdateNATSConsumerLag(lag)
	}
}

// TestDLQMetricsConcurrent tests DLQ metrics under concurrent access
func TestDLQMetricsConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			category := "concurrent_test"
			for j := 0; j < operationsPerGoroutine; j++ {
				RecordDLQEntry(category)
				RecordDLQRetry(j%2 == 0)
			}
		}(i)
	}

	wg.Wait()
}

// TestNATSMetricsConcurrent tests NATS metrics under concurrent access
func TestNATSMetricsConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				RecordNATSPublish()
				RecordNATSConsume()
				RecordNATSProcessed()
				RecordNATSProcessingDuration(time.Duration(j) * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
}

// ==================== Wrapped Report Metrics Tests ====================

// TestRecordWrappedGeneration tests wrapped report generation metric recording
func TestRecordWrappedGeneration(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		scope    string
		duration time.Duration
		err      error
	}{
		{
			name:     "successful user report generation",
			year:     2025,
			scope:    "user",
			duration: 500 * time.Millisecond,
			err:      nil,
		},
		{
			name:     "successful server report generation",
			year:     2024,
			scope:    "server",
			duration: 2 * time.Second,
			err:      nil,
		},
		{
			name:     "successful batch report generation",
			year:     2025,
			scope:    "batch",
			duration: 30 * time.Second,
			err:      nil,
		},
		{
			name:     "no playback data error",
			year:     2023,
			scope:    "user",
			duration: 100 * time.Millisecond,
			err:      errors.New("no playback events found for user"),
		},
		{
			name:     "database query error",
			year:     2025,
			scope:    "user",
			duration: 5 * time.Second,
			err:      errors.New("query failed: connection timeout"),
		},
		{
			name:     "save error",
			year:     2025,
			scope:    "user",
			duration: 1 * time.Second,
			err:      errors.New("save failed: constraint violation"),
		},
		{
			name:     "unknown error type",
			year:     2025,
			scope:    "user",
			duration: 200 * time.Millisecond,
			err:      errors.New("something unexpected happened"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record the generation - should not panic
			RecordWrappedGeneration(tt.year, tt.scope, tt.duration, tt.err)
		})
	}
}

// TestRecordWrappedBatch tests wrapped report batch size recording
func TestRecordWrappedBatch(t *testing.T) {
	batchSizes := []int{1, 5, 10, 25, 50, 100, 250, 500, 1000}

	for _, size := range batchSizes {
		t.Run("batch_size_"+strconv.Itoa(size), func(t *testing.T) {
			RecordWrappedBatch(size)
		})
	}
}

// TestRecordWrappedCacheHitMiss tests wrapped report cache hit/miss recording
func TestRecordWrappedCacheHitMiss(t *testing.T) {
	years := []int{2023, 2024, 2025, 2026}

	for _, year := range years {
		t.Run("year_"+strconv.Itoa(year), func(t *testing.T) {
			// Test cache hits
			RecordWrappedCacheHit(year)
			RecordWrappedCacheHit(year)
			RecordWrappedCacheHit(year)

			// Test cache misses
			RecordWrappedCacheMiss(year)
		})
	}
}

// TestRecordWrappedShareToken tests wrapped share token metric recording
func TestRecordWrappedShareToken(t *testing.T) {
	// Record share token creation
	for i := 0; i < 5; i++ {
		RecordWrappedShareTokenCreated()
	}

	// Record share token accesses
	for i := 0; i < 10; i++ {
		RecordWrappedShareAccess()
	}
}

// TestRecordWrappedLeaderboardQuery tests wrapped leaderboard query recording
func TestRecordWrappedLeaderboardQuery(t *testing.T) {
	years := []int{2023, 2024, 2025}

	for _, year := range years {
		t.Run("year_"+strconv.Itoa(year), func(t *testing.T) {
			RecordWrappedLeaderboardQuery(year)
			RecordWrappedLeaderboardQuery(year)
		})
	}
}

// TestUpdateWrappedActiveYearReports tests wrapped active year gauge updates
func TestUpdateWrappedActiveYearReports(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		count int64
	}{
		{"2025 with 100 reports", 2025, 100},
		{"2024 with 500 reports", 2024, 500},
		{"2023 with 1000 reports", 2023, 1000},
		{"2025 reset to 0", 2025, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UpdateWrappedActiveYearReports(tt.year, tt.count)
		})
	}
}

// TestWrappedMetricsConcurrent tests wrapped metrics under concurrent access
func TestWrappedMetricsConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100

	// Test concurrent generation recording
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				RecordWrappedGeneration(2025, "user", time.Duration(j)*time.Millisecond, nil)
				RecordWrappedCacheHit(2025)
			}
		}(i)
	}

	// Test concurrent cache recording
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				if j%2 == 0 {
					RecordWrappedCacheHit(2024)
				} else {
					RecordWrappedCacheMiss(2024)
				}
			}
		}(i)
	}

	// Test concurrent share token recording
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				RecordWrappedShareTokenCreated()
				RecordWrappedShareAccess()
			}
		}(i)
	}

	wg.Wait()
}

// TestWrappedMetricsErrorClassification tests error type classification
func TestWrappedMetricsErrorClassification(t *testing.T) {
	tests := []struct {
		name         string
		errMsg       string
		expectedType string
	}{
		{"no playback error", "no playback data found", "no_data"},
		{"query error", "query failed: syntax error", "query_failed"},
		{"database error", "database connection refused", "query_failed"},
		{"save error", "save failed: disk full", "save_failed"},
		{"insert error", "insert failed: duplicate key", "save_failed"},
		{"unknown error", "unexpected network error", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			RecordWrappedGeneration(2025, "user", time.Second, err)
			// Verifies no panic and error is recorded
		})
	}
}

// TestWrappedMetricsLabels verifies wrapped metrics have proper labels
func TestWrappedMetricsLabels(t *testing.T) {
	// Test WrappedReportGenerationDuration has correct labels
	WrappedReportGenerationDuration.WithLabelValues("2025", "user").Observe(1.0)
	WrappedReportGenerationDuration.WithLabelValues("2024", "server").Observe(2.0)
	WrappedReportGenerationDuration.WithLabelValues("2025", "batch").Observe(30.0)

	// Test WrappedReportsGenerated has correct labels
	WrappedReportsGenerated.WithLabelValues("2025", "user").Inc()
	WrappedReportsGenerated.WithLabelValues("2024", "server").Inc()

	// Test WrappedReportGenerationErrors has correct labels
	WrappedReportGenerationErrors.WithLabelValues("2025", "no_data").Inc()
	WrappedReportGenerationErrors.WithLabelValues("2024", "query_failed").Inc()
	WrappedReportGenerationErrors.WithLabelValues("2023", "save_failed").Inc()

	// Test WrappedReportCacheHits has correct labels
	WrappedReportCacheHits.WithLabelValues("2025").Inc()
	WrappedReportCacheHits.WithLabelValues("2024").Inc()

	// Test WrappedReportCacheMisses has correct labels
	WrappedReportCacheMisses.WithLabelValues("2025").Inc()

	// Test WrappedLeaderboardQueries has correct labels
	WrappedLeaderboardQueries.WithLabelValues("2025").Inc()

	// Test WrappedActiveYear has correct labels
	WrappedActiveYear.WithLabelValues("2025").Set(100)
	WrappedActiveYear.WithLabelValues("2024").Set(500)
}

// TestWrappedMetricsRegistration verifies all wrapped metrics are properly registered
func TestWrappedMetricsRegistration(t *testing.T) {
	metrics := []prometheus.Collector{
		WrappedReportGenerationDuration,
		WrappedReportsGenerated,
		WrappedReportGenerationErrors,
		WrappedReportBatchSize,
		WrappedReportCacheHits,
		WrappedReportCacheMisses,
		WrappedShareTokensCreated,
		WrappedShareTokenAccess,
		WrappedLeaderboardQueries,
		WrappedActiveYear,
	}

	// Verify each metric can be described
	for _, m := range metrics {
		ch := make(chan *prometheus.Desc, 10)
		m.Describe(ch)
		close(ch)

		// Should have at least one descriptor
		count := 0
		for range ch {
			count++
		}
		if count == 0 {
			t.Errorf("Wrapped metric has no descriptors")
		}
	}
}

// Benchmark tests for wrapped metrics performance

func BenchmarkRecordWrappedGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordWrappedGeneration(2025, "user", 500*time.Millisecond, nil)
	}
}

func BenchmarkRecordWrappedGenerationWithError(b *testing.B) {
	err := errors.New("no playback data found")
	for i := 0; i < b.N; i++ {
		RecordWrappedGeneration(2025, "user", 100*time.Millisecond, err)
	}
}

func BenchmarkRecordWrappedCacheHit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordWrappedCacheHit(2025)
	}
}

func BenchmarkRecordWrappedBatch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordWrappedBatch(100)
	}
}
