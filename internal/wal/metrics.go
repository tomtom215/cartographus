// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package wal

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for WAL operations
var (
	// walWritesTotal counts total WAL write operations.
	walWritesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_writes_total",
		Help: "Total number of WAL write operations",
	})

	// walConfirmsTotal counts total WAL confirm operations.
	walConfirmsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_confirms_total",
		Help: "Total number of WAL confirm operations",
	})

	// walRetriesTotal counts total WAL retry attempts.
	walRetriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_retries_total",
		Help: "Total number of WAL retry attempts",
	})

	// walPendingEntries is the current number of pending WAL entries.
	walPendingEntries = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "wal_pending_entries",
		Help: "Current number of pending WAL entries",
	})

	// walConfirmedEntries is the current number of confirmed WAL entries awaiting compaction.
	walConfirmedEntries = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "wal_confirmed_entries",
		Help: "Current number of confirmed WAL entries awaiting compaction",
	})

	// walWriteLatency measures WAL write latency.
	walWriteLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "wal_write_latency_seconds",
		Help:    "WAL write latency in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// walDBSizeBytes is the current BadgerDB database size.
	walDBSizeBytes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "wal_db_size_bytes",
		Help: "BadgerDB database size in bytes",
	})

	// walCompactionsTotal counts total compaction runs.
	walCompactionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_compactions_total",
		Help: "Total number of WAL compaction runs",
	})

	// walEntriesCompacted counts entries removed during compaction.
	walEntriesCompacted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_entries_compacted_total",
		Help: "Total number of entries removed during compaction",
	})

	// walRecoveredEntries counts entries recovered on startup.
	walRecoveredEntries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_recovered_entries_total",
		Help: "Total number of entries recovered on startup",
	})

	// walWriteFailures counts failed WAL writes.
	walWriteFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_write_failures_total",
		Help: "Total number of failed WAL write operations",
	})

	// walNATSPublishFailures counts failed NATS publish attempts from WAL.
	walNATSPublishFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_nats_publish_failures_total",
		Help: "Total number of NATS publish failures from WAL entries",
	})

	// walMaxRetriesExceeded counts entries that exceeded max retry attempts.
	walMaxRetriesExceeded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_max_retries_exceeded_total",
		Help: "Total number of entries that exceeded maximum retry attempts",
	})

	// walExpiredEntries counts entries that expired before confirmation.
	walExpiredEntries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_expired_entries_total",
		Help: "Total number of entries that expired before NATS confirmation",
	})

	// walCompactionLatency measures compaction latency.
	walCompactionLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "wal_compaction_latency_seconds",
		Help:    "WAL compaction latency in seconds",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~100s
	})

	// walGCLatency measures BadgerDB value log GC latency.
	walGCLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "wal_gc_latency_seconds",
		Help:    "BadgerDB value log GC latency in seconds",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 12), // 0.01s to ~40s
	})

	// walGCRuns counts total GC runs.
	walGCRuns = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wal_gc_runs_total",
		Help: "Total number of BadgerDB value log GC runs",
	})

	// Consumer WAL metrics (ADR-0023: Exactly-Once Delivery)

	// consumerWALWritesTotal counts total Consumer WAL write operations.
	consumerWALWritesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consumer_wal_writes_total",
		Help: "Total number of Consumer WAL write operations",
	})

	// consumerWALConfirmsTotal counts total Consumer WAL confirm operations.
	consumerWALConfirmsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consumer_wal_confirms_total",
		Help: "Total number of Consumer WAL confirm operations (successful DuckDB inserts)",
	})

	// consumerWALRetriesTotal counts total Consumer WAL retry attempts.
	consumerWALRetriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consumer_wal_retries_total",
		Help: "Total number of Consumer WAL retry attempts",
	})

	// consumerWALFailuresTotal counts permanently failed entries.
	consumerWALFailuresTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consumer_wal_failures_total",
		Help: "Total number of Consumer WAL entries that permanently failed",
	})

	// consumerWALRecoveriesTotal counts entries recovered on startup.
	consumerWALRecoveriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "consumer_wal_recoveries_total",
		Help: "Total number of Consumer WAL entries recovered on startup",
	})

	// consumerWALPendingEntries is the current number of pending Consumer WAL entries.
	consumerWALPendingEntries = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "consumer_wal_pending_entries",
		Help: "Current number of pending Consumer WAL entries awaiting DuckDB insert",
	})
)

// RecordWALWrite increments the write counter.
func RecordWALWrite() {
	walWritesTotal.Inc()
}

// RecordWALConfirm increments the confirm counter.
func RecordWALConfirm() {
	walConfirmsTotal.Inc()
}

// RecordWALRetry increments the retry counter.
func RecordWALRetry() {
	walRetriesTotal.Inc()
}

// UpdateWALPendingEntries sets the pending entries gauge.
func UpdateWALPendingEntries(count int64) {
	walPendingEntries.Set(float64(count))
}

// UpdateWALConfirmedEntries sets the confirmed entries gauge.
func UpdateWALConfirmedEntries(count int64) {
	walConfirmedEntries.Set(float64(count))
}

// RecordWALWriteLatency records a write latency measurement.
func RecordWALWriteLatency(seconds float64) {
	walWriteLatency.Observe(seconds)
}

// UpdateWALDBSize sets the database size gauge.
func UpdateWALDBSize(bytes int64) {
	walDBSizeBytes.Set(float64(bytes))
}

// RecordWALCompaction increments the compaction counter.
func RecordWALCompaction() {
	walCompactionsTotal.Inc()
}

// RecordWALEntriesCompacted adds to the compacted entries counter.
func RecordWALEntriesCompacted(count int64) {
	walEntriesCompacted.Add(float64(count))
}

// RecordWALRecoveredEntries adds to the recovered entries counter.
func RecordWALRecoveredEntries(count int64) {
	walRecoveredEntries.Add(float64(count))
}

// RecordWALWriteFailure increments the write failure counter.
func RecordWALWriteFailure() {
	walWriteFailures.Inc()
}

// RecordWALNATSPublishFailure increments the NATS publish failure counter.
func RecordWALNATSPublishFailure() {
	walNATSPublishFailures.Inc()
}

// RecordWALMaxRetriesExceeded increments the max retries exceeded counter.
func RecordWALMaxRetriesExceeded() {
	walMaxRetriesExceeded.Inc()
}

// RecordWALExpiredEntry increments the expired entries counter.
func RecordWALExpiredEntry() {
	walExpiredEntries.Inc()
}

// RecordWALCompactionLatency records a compaction latency measurement.
func RecordWALCompactionLatency(seconds float64) {
	walCompactionLatency.Observe(seconds)
}

// RecordWALGCLatency records a GC latency measurement.
func RecordWALGCLatency(seconds float64) {
	walGCLatency.Observe(seconds)
}

// RecordWALGCRun increments the GC run counter.
func RecordWALGCRun() {
	walGCRuns.Inc()
}

// Consumer WAL metric recording functions (ADR-0023)

// RecordConsumerWALWrite increments the consumer WAL write counter.
func RecordConsumerWALWrite() {
	consumerWALWritesTotal.Inc()
}

// RecordConsumerWALConfirm increments the consumer WAL confirm counter.
func RecordConsumerWALConfirm() {
	consumerWALConfirmsTotal.Inc()
}

// RecordConsumerWALRetry increments the consumer WAL retry counter.
func RecordConsumerWALRetry() {
	consumerWALRetriesTotal.Inc()
}

// RecordConsumerWALFailure increments the consumer WAL failure counter.
func RecordConsumerWALFailure() {
	consumerWALFailuresTotal.Inc()
}

// RecordConsumerWALRecovery adds to the consumer WAL recovery counter.
func RecordConsumerWALRecovery(count int64) {
	consumerWALRecoveriesTotal.Add(float64(count))
}

// UpdateConsumerWALPendingEntries sets the consumer WAL pending entries gauge.
func UpdateConsumerWALPendingEntries(count int64) {
	consumerWALPendingEntries.Set(float64(count))
}
