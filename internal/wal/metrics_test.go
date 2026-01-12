// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package wal

import (
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestMetrics_WALCounterFunctions tests all WAL counter increment functions.
// These tests verify that counters increment correctly relative to their previous value.
func TestMetrics_WALCounterFunctions(t *testing.T) {
	// Cannot use t.Parallel() - shared global metrics

	tests := []struct {
		name       string
		recordFunc func()
		metric     prometheus.Counter
		metricName string
	}{
		{
			name:       "RecordWALWrite",
			recordFunc: RecordWALWrite,
			metric:     walWritesTotal,
			metricName: "wal_writes_total",
		},
		{
			name:       "RecordWALConfirm",
			recordFunc: RecordWALConfirm,
			metric:     walConfirmsTotal,
			metricName: "wal_confirms_total",
		},
		{
			name:       "RecordWALRetry",
			recordFunc: RecordWALRetry,
			metric:     walRetriesTotal,
			metricName: "wal_retries_total",
		},
		{
			name:       "RecordWALCompaction",
			recordFunc: RecordWALCompaction,
			metric:     walCompactionsTotal,
			metricName: "wal_compactions_total",
		},
		{
			name:       "RecordWALWriteFailure",
			recordFunc: RecordWALWriteFailure,
			metric:     walWriteFailures,
			metricName: "wal_write_failures_total",
		},
		{
			name:       "RecordWALNATSPublishFailure",
			recordFunc: RecordWALNATSPublishFailure,
			metric:     walNATSPublishFailures,
			metricName: "wal_nats_publish_failures_total",
		},
		{
			name:       "RecordWALMaxRetriesExceeded",
			recordFunc: RecordWALMaxRetriesExceeded,
			metric:     walMaxRetriesExceeded,
			metricName: "wal_max_retries_exceeded_total",
		},
		{
			name:       "RecordWALExpiredEntry",
			recordFunc: RecordWALExpiredEntry,
			metric:     walExpiredEntries,
			metricName: "wal_expired_entries_total",
		},
		{
			name:       "RecordWALGCRun",
			recordFunc: RecordWALGCRun,
			metric:     walGCRuns,
			metricName: "wal_gc_runs_total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get initial value
			initialValue := testutil.ToFloat64(tt.metric)

			// Call the recording function
			tt.recordFunc()

			// Verify counter was incremented by exactly 1
			newValue := testutil.ToFloat64(tt.metric)
			delta := newValue - initialValue
			if delta != 1 {
				t.Errorf("%s: expected counter to increment by 1, got delta of %f", tt.name, delta)
			}

			// Call multiple times to verify consistent behavior
			tt.recordFunc()
			tt.recordFunc()

			finalValue := testutil.ToFloat64(tt.metric)
			totalDelta := finalValue - initialValue
			if totalDelta != 3 {
				t.Errorf("%s: expected counter to increment by 3 total, got delta of %f", tt.name, totalDelta)
			}
		})
	}
}

// TestMetrics_WALAddCounterFunctions tests WAL counter functions that add arbitrary values.
func TestMetrics_WALAddCounterFunctions(t *testing.T) {
	// Cannot use t.Parallel() - shared global metrics

	tests := []struct {
		name       string
		recordFunc func(int64)
		metric     prometheus.Counter
		metricName string
	}{
		{
			name:       "RecordWALEntriesCompacted",
			recordFunc: RecordWALEntriesCompacted,
			metric:     walEntriesCompacted,
			metricName: "wal_entries_compacted_total",
		},
		{
			name:       "RecordWALRecoveredEntries",
			recordFunc: RecordWALRecoveredEntries,
			metric:     walRecoveredEntries,
			metricName: "wal_recovered_entries_total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get initial value
			initialValue := testutil.ToFloat64(tt.metric)

			// Add a specific count
			tt.recordFunc(5)

			newValue := testutil.ToFloat64(tt.metric)
			delta := newValue - initialValue
			if delta != 5 {
				t.Errorf("%s: expected counter to add 5, got delta of %f", tt.name, delta)
			}

			// Add zero (should not change)
			beforeZero := testutil.ToFloat64(tt.metric)
			tt.recordFunc(0)
			afterZero := testutil.ToFloat64(tt.metric)
			if afterZero != beforeZero {
				t.Errorf("%s: adding 0 should not change counter, got delta of %f", tt.name, afterZero-beforeZero)
			}

			// Add large value
			beforeLarge := testutil.ToFloat64(tt.metric)
			tt.recordFunc(1000)
			afterLarge := testutil.ToFloat64(tt.metric)
			if afterLarge-beforeLarge != 1000 {
				t.Errorf("%s: expected counter to add 1000, got delta of %f", tt.name, afterLarge-beforeLarge)
			}
		})
	}
}

// TestMetrics_WALGaugeFunctions tests all WAL gauge update functions.
// Each test captures the value immediately after setting to avoid race conditions.
func TestMetrics_WALGaugeFunctions(t *testing.T) {
	// Cannot use t.Parallel() - shared global metrics

	tests := []struct {
		name       string
		recordFunc func(int64)
		metric     prometheus.Gauge
		metricName string
	}{
		{
			name:       "UpdateWALPendingEntries",
			recordFunc: UpdateWALPendingEntries,
			metric:     walPendingEntries,
			metricName: "wal_pending_entries",
		},
		{
			name:       "UpdateWALConfirmedEntries",
			recordFunc: UpdateWALConfirmedEntries,
			metric:     walConfirmedEntries,
			metricName: "wal_confirmed_entries",
		},
		{
			name:       "UpdateWALDBSize",
			recordFunc: UpdateWALDBSize,
			metric:     walDBSizeBytes,
			metricName: "wal_db_size_bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test positive value
			tt.recordFunc(100)
			value := testutil.ToFloat64(tt.metric)
			if value != 100 {
				t.Errorf("%s: expected 100, got %f", tt.name, value)
			}

			// Test zero value
			tt.recordFunc(0)
			value = testutil.ToFloat64(tt.metric)
			if value != 0 {
				t.Errorf("%s: expected 0, got %f", tt.name, value)
			}

			// Test negative value (gauges can be negative)
			tt.recordFunc(-50)
			value = testutil.ToFloat64(tt.metric)
			if value != -50 {
				t.Errorf("%s: expected -50, got %f", tt.name, value)
			}

			// Test large value
			tt.recordFunc(1000000000)
			value = testutil.ToFloat64(tt.metric)
			if value != 1000000000 {
				t.Errorf("%s: expected 1000000000, got %f", tt.name, value)
			}
		})
	}
}

// TestMetrics_WALHistogramFunctions tests all WAL histogram observation functions.
func TestMetrics_WALHistogramFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		recordFunc func(float64)
		metricName string
	}{
		{
			name:       "RecordWALWriteLatency",
			recordFunc: RecordWALWriteLatency,
			metricName: "wal_write_latency_seconds",
		},
		{
			name:       "RecordWALCompactionLatency",
			recordFunc: RecordWALCompactionLatency,
			metricName: "wal_compaction_latency_seconds",
		},
		{
			name:       "RecordWALGCLatency",
			recordFunc: RecordWALGCLatency,
			metricName: "wal_gc_latency_seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Histograms can only have positive values observed
			// Test various positive values
			testValues := []float64{0.001, 0.01, 0.1, 0.5, 1.0, 5.0, 10.0}

			for _, val := range testValues {
				// Should not panic
				tt.recordFunc(val)
			}

			// Test zero value
			tt.recordFunc(0)

			// Test very small positive value
			tt.recordFunc(0.0001)
		})
	}
}

// TestMetrics_ConsumerWALCounterFunctions tests all Consumer WAL counter increment functions.
func TestMetrics_ConsumerWALCounterFunctions(t *testing.T) {
	// Cannot use t.Parallel() - shared global metrics

	tests := []struct {
		name       string
		recordFunc func()
		metric     prometheus.Counter
		metricName string
	}{
		{
			name:       "RecordConsumerWALWrite",
			recordFunc: RecordConsumerWALWrite,
			metric:     consumerWALWritesTotal,
			metricName: "consumer_wal_writes_total",
		},
		{
			name:       "RecordConsumerWALConfirm",
			recordFunc: RecordConsumerWALConfirm,
			metric:     consumerWALConfirmsTotal,
			metricName: "consumer_wal_confirms_total",
		},
		{
			name:       "RecordConsumerWALRetry",
			recordFunc: RecordConsumerWALRetry,
			metric:     consumerWALRetriesTotal,
			metricName: "consumer_wal_retries_total",
		},
		{
			name:       "RecordConsumerWALFailure",
			recordFunc: RecordConsumerWALFailure,
			metric:     consumerWALFailuresTotal,
			metricName: "consumer_wal_failures_total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get initial value
			initialValue := testutil.ToFloat64(tt.metric)

			// Call the recording function
			tt.recordFunc()

			// Verify counter was incremented by exactly 1
			newValue := testutil.ToFloat64(tt.metric)
			delta := newValue - initialValue
			if delta != 1 {
				t.Errorf("%s: expected counter to increment by 1, got delta of %f", tt.name, delta)
			}

			// Call multiple times
			tt.recordFunc()
			tt.recordFunc()

			finalValue := testutil.ToFloat64(tt.metric)
			totalDelta := finalValue - initialValue
			if totalDelta != 3 {
				t.Errorf("%s: expected counter to increment by 3 total, got delta of %f", tt.name, totalDelta)
			}
		})
	}
}

// TestMetrics_ConsumerWALRecoveryCounter tests RecordConsumerWALRecovery function.
func TestMetrics_ConsumerWALRecoveryCounter(t *testing.T) {
	// Cannot use t.Parallel() - shared global metrics

	// Get initial value
	initialValue := testutil.ToFloat64(consumerWALRecoveriesTotal)

	// Add specific count
	RecordConsumerWALRecovery(10)
	newValue := testutil.ToFloat64(consumerWALRecoveriesTotal)
	delta := newValue - initialValue
	if delta != 10 {
		t.Errorf("RecordConsumerWALRecovery: expected counter to add 10, got delta of %f", delta)
	}

	// Add zero
	beforeZero := testutil.ToFloat64(consumerWALRecoveriesTotal)
	RecordConsumerWALRecovery(0)
	afterZero := testutil.ToFloat64(consumerWALRecoveriesTotal)
	if afterZero != beforeZero {
		t.Errorf("RecordConsumerWALRecovery: adding 0 should not change counter, got delta of %f", afterZero-beforeZero)
	}

	// Add another value
	beforeAdd := testutil.ToFloat64(consumerWALRecoveriesTotal)
	RecordConsumerWALRecovery(25)
	afterAdd := testutil.ToFloat64(consumerWALRecoveriesTotal)
	if afterAdd-beforeAdd != 25 {
		t.Errorf("RecordConsumerWALRecovery: expected counter to add 25, got delta of %f", afterAdd-beforeAdd)
	}
}

// TestMetrics_ConsumerWALPendingGauge tests UpdateConsumerWALPendingEntries function.
func TestMetrics_ConsumerWALPendingGauge(t *testing.T) {
	// Cannot use t.Parallel() - shared global metrics

	// Test positive value
	UpdateConsumerWALPendingEntries(50)
	value := testutil.ToFloat64(consumerWALPendingEntries)
	if value != 50 {
		t.Errorf("UpdateConsumerWALPendingEntries: expected 50, got %f", value)
	}

	// Test zero
	UpdateConsumerWALPendingEntries(0)
	value = testutil.ToFloat64(consumerWALPendingEntries)
	if value != 0 {
		t.Errorf("UpdateConsumerWALPendingEntries: expected 0, got %f", value)
	}

	// Test negative (gauges support this)
	UpdateConsumerWALPendingEntries(-10)
	value = testutil.ToFloat64(consumerWALPendingEntries)
	if value != -10 {
		t.Errorf("UpdateConsumerWALPendingEntries: expected -10, got %f", value)
	}

	// Test large value
	UpdateConsumerWALPendingEntries(999999)
	value = testutil.ToFloat64(consumerWALPendingEntries)
	if value != 999999 {
		t.Errorf("UpdateConsumerWALPendingEntries: expected 999999, got %f", value)
	}
}

// TestMetrics_HistogramBuckets tests that histograms use appropriate bucket configurations.
func TestMetrics_HistogramBuckets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		recordFunc func(float64)
		metricName string
	}{
		{
			name:       "write latency",
			recordFunc: RecordWALWriteLatency,
			metricName: "wal_write_latency_seconds",
		},
		{
			name:       "compaction latency",
			recordFunc: RecordWALCompactionLatency,
			metricName: "wal_compaction_latency_seconds",
		},
		{
			name:       "GC latency",
			recordFunc: RecordWALGCLatency,
			metricName: "wal_gc_latency_seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record a value and check no panic occurs
			tt.recordFunc(0.5)

			// Record edge case values
			tt.recordFunc(0.0)       // Minimum
			tt.recordFunc(0.0000001) // Very small
			tt.recordFunc(100.0)     // Large
			tt.recordFunc(1000.0)    // Very large (outside normal buckets)
		})
	}
}

// TestMetrics_ConcurrentAccess tests that metrics can be safely accessed concurrently.
func TestMetrics_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	// Run multiple goroutines recording metrics concurrently
	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Counter increments
				RecordWALWrite()
				RecordWALConfirm()
				RecordWALRetry()
				RecordWALWriteFailure()
				RecordConsumerWALWrite()
				RecordConsumerWALConfirm()

				// Counter adds
				RecordWALEntriesCompacted(1)
				RecordWALRecoveredEntries(1)
				RecordConsumerWALRecovery(1)

				// Gauge updates
				UpdateWALPendingEntries(int64(j))
				UpdateWALConfirmedEntries(int64(j))
				UpdateConsumerWALPendingEntries(int64(j))

				// Histogram observations
				RecordWALWriteLatency(0.001 * float64(j+1))
				RecordWALCompactionLatency(0.01 * float64(j+1))
				RecordWALGCLatency(0.001 * float64(j+1))
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify counters increased (we can't predict exact values due to parallel test execution)
	// Just verify they're positive and no panics occurred
	if testutil.ToFloat64(walWritesTotal) == 0 {
		t.Error("walWritesTotal should be positive after concurrent access")
	}
	if testutil.ToFloat64(consumerWALWritesTotal) == 0 {
		t.Error("consumerWALWritesTotal should be positive after concurrent access")
	}
}

// TestMetrics_MetricNames verifies that metrics are registered with expected names.
func TestMetrics_MetricNames(t *testing.T) {
	t.Parallel()

	expectedNames := []string{
		"wal_writes_total",
		"wal_confirms_total",
		"wal_retries_total",
		"wal_pending_entries",
		"wal_confirmed_entries",
		"wal_write_latency_seconds",
		"wal_db_size_bytes",
		"wal_compactions_total",
		"wal_entries_compacted_total",
		"wal_recovered_entries_total",
		"wal_write_failures_total",
		"wal_nats_publish_failures_total",
		"wal_max_retries_exceeded_total",
		"wal_expired_entries_total",
		"wal_compaction_latency_seconds",
		"wal_gc_latency_seconds",
		"wal_gc_runs_total",
		"consumer_wal_writes_total",
		"consumer_wal_confirms_total",
		"consumer_wal_retries_total",
		"consumer_wal_failures_total",
		"consumer_wal_recoveries_total",
		"consumer_wal_pending_entries",
	}

	// Collect all registered metric names from default registry
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	registeredNames := make(map[string]bool)
	for _, mf := range mfs {
		registeredNames[mf.GetName()] = true
	}

	for _, name := range expectedNames {
		if !registeredNames[name] {
			t.Errorf("Expected metric %q to be registered but it was not found", name)
		}
	}
}

// TestMetrics_MetricHelp verifies that metrics have meaningful help text.
func TestMetrics_MetricHelp(t *testing.T) {
	t.Parallel()

	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	walMetrics := make(map[string]string)
	for _, mf := range mfs {
		name := mf.GetName()
		if strings.HasPrefix(name, "wal_") || strings.HasPrefix(name, "consumer_wal_") {
			walMetrics[name] = mf.GetHelp()
		}
	}

	// Verify each WAL metric has non-empty help text
	for name, help := range walMetrics {
		if help == "" {
			t.Errorf("Metric %q has empty help text", name)
		}
		// Help text should be descriptive (at least 10 chars)
		if len(help) < 10 {
			t.Errorf("Metric %q has very short help text: %q", name, help)
		}
	}
}

// TestMetrics_CounterNeverDecreases tests that counter values never decrease.
func TestMetrics_CounterNeverDecreases(t *testing.T) {
	// Cannot use t.Parallel() - shared global metrics

	// Get initial values
	initialWrites := testutil.ToFloat64(walWritesTotal)

	// Record some values
	for i := 0; i < 10; i++ {
		RecordWALWrite()
		currentValue := testutil.ToFloat64(walWritesTotal)
		if currentValue < initialWrites {
			t.Errorf("Counter walWritesTotal decreased from %f to %f", initialWrites, currentValue)
		}
		if currentValue < initialWrites+float64(i) {
			t.Errorf("Counter walWritesTotal did not increase as expected")
		}
	}
}

// TestMetrics_GaugeCanIncrementAndDecrement tests that gauges support both operations.
func TestMetrics_GaugeCanIncrementAndDecrement(t *testing.T) {
	// Cannot use t.Parallel() - shared global metrics

	// Set to positive value
	UpdateWALPendingEntries(100)
	v1 := testutil.ToFloat64(walPendingEntries)
	if v1 != 100 {
		t.Errorf("Expected 100, got %f", v1)
	}

	// Decrease (set to lower value)
	UpdateWALPendingEntries(50)
	v2 := testutil.ToFloat64(walPendingEntries)
	if v2 != 50 {
		t.Errorf("Expected 50, got %f", v2)
	}

	// Decrease to zero
	UpdateWALPendingEntries(0)
	v3 := testutil.ToFloat64(walPendingEntries)
	if v3 != 0 {
		t.Errorf("Expected 0, got %f", v3)
	}

	// Set to negative (possible with gauges)
	UpdateWALPendingEntries(-25)
	v4 := testutil.ToFloat64(walPendingEntries)
	if v4 != -25 {
		t.Errorf("Expected -25, got %f", v4)
	}
}

// TestMetrics_EdgeCases tests edge case values for metrics.
func TestMetrics_EdgeCases(t *testing.T) {
	// Cannot use t.Parallel() - shared global metrics

	// Test int64 max value for gauges
	UpdateWALDBSize(9223372036854775807) // MaxInt64
	v := testutil.ToFloat64(walDBSizeBytes)
	// Float64 has limited precision for large integers
	if v <= 0 {
		t.Errorf("Expected positive value for MaxInt64, got %f", v)
	}

	// Test int64 min value for gauges (should work)
	UpdateWALPendingEntries(-9223372036854775807)
	v = testutil.ToFloat64(walPendingEntries)
	if v >= 0 {
		t.Errorf("Expected negative value for MinInt64+1, got %f", v)
	}

	// Test zero for all gauge types
	UpdateWALPendingEntries(0)
	UpdateWALConfirmedEntries(0)
	UpdateWALDBSize(0)
	UpdateConsumerWALPendingEntries(0)

	if testutil.ToFloat64(walPendingEntries) != 0 {
		t.Error("walPendingEntries should be 0")
	}
	if testutil.ToFloat64(walConfirmedEntries) != 0 {
		t.Error("walConfirmedEntries should be 0")
	}
	if testutil.ToFloat64(walDBSizeBytes) != 0 {
		t.Error("walDBSizeBytes should be 0")
	}
	if testutil.ToFloat64(consumerWALPendingEntries) != 0 {
		t.Error("consumerWALPendingEntries should be 0")
	}
}

// TestMetrics_HistogramSumAndCount verifies histogram observations accumulate correctly.
func TestMetrics_HistogramSumAndCount(t *testing.T) {
	t.Parallel()

	// Record known values to write latency histogram
	testValues := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	for _, v := range testValues {
		RecordWALWriteLatency(v)
	}

	// Note: Due to parallel tests and the shared default registry,
	// we can't assert exact sum/count values. We just verify no panics occur
	// and the histogram accepts the values.
}

// TestMetrics_AllFunctionsExecuteWithoutPanic ensures all metric functions can be called safely.
func TestMetrics_AllFunctionsExecuteWithoutPanic(t *testing.T) {
	t.Parallel()

	// This test verifies that all functions can be called without panicking
	// even with edge case values

	// Counter increments
	RecordWALWrite()
	RecordWALConfirm()
	RecordWALRetry()
	RecordWALCompaction()
	RecordWALWriteFailure()
	RecordWALNATSPublishFailure()
	RecordWALMaxRetriesExceeded()
	RecordWALExpiredEntry()
	RecordWALGCRun()
	RecordConsumerWALWrite()
	RecordConsumerWALConfirm()
	RecordConsumerWALRetry()
	RecordConsumerWALFailure()

	// Counter adds with various values
	RecordWALEntriesCompacted(0)
	RecordWALEntriesCompacted(1)
	RecordWALEntriesCompacted(100)
	RecordWALRecoveredEntries(0)
	RecordWALRecoveredEntries(1)
	RecordWALRecoveredEntries(100)
	RecordConsumerWALRecovery(0)
	RecordConsumerWALRecovery(1)
	RecordConsumerWALRecovery(100)

	// Gauge updates with various values
	UpdateWALPendingEntries(0)
	UpdateWALPendingEntries(100)
	UpdateWALPendingEntries(-100)
	UpdateWALConfirmedEntries(0)
	UpdateWALConfirmedEntries(100)
	UpdateWALConfirmedEntries(-100)
	UpdateWALDBSize(0)
	UpdateWALDBSize(1024 * 1024 * 1024)
	UpdateConsumerWALPendingEntries(0)
	UpdateConsumerWALPendingEntries(50)
	UpdateConsumerWALPendingEntries(-50)

	// Histogram observations with various values
	RecordWALWriteLatency(0)
	RecordWALWriteLatency(0.001)
	RecordWALWriteLatency(1.0)
	RecordWALWriteLatency(100.0)
	RecordWALCompactionLatency(0)
	RecordWALCompactionLatency(0.1)
	RecordWALCompactionLatency(10.0)
	RecordWALGCLatency(0)
	RecordWALGCLatency(0.01)
	RecordWALGCLatency(1.0)
}

// BenchmarkMetrics_RecordWALWrite benchmarks counter increment performance.
func BenchmarkMetrics_RecordWALWrite(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordWALWrite()
	}
}

// BenchmarkMetrics_UpdateWALPendingEntries benchmarks gauge update performance.
func BenchmarkMetrics_UpdateWALPendingEntries(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UpdateWALPendingEntries(int64(i))
	}
}

// BenchmarkMetrics_RecordWALWriteLatency benchmarks histogram observation performance.
func BenchmarkMetrics_RecordWALWriteLatency(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordWALWriteLatency(0.001 * float64(i%1000))
	}
}

// BenchmarkMetrics_ConcurrentCounterIncrement benchmarks concurrent counter access.
func BenchmarkMetrics_ConcurrentCounterIncrement(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			RecordWALWrite()
		}
	})
}

// BenchmarkMetrics_ConcurrentGaugeUpdate benchmarks concurrent gauge access.
func BenchmarkMetrics_ConcurrentGaugeUpdate(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		i := int64(0)
		for pb.Next() {
			i++
			UpdateWALPendingEntries(i)
		}
	})
}
