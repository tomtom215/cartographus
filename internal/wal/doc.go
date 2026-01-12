// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package wal provides a durable Write-Ahead Log (WAL) using BadgerDB.
//
// The WAL guarantees no event loss by persisting events to disk before
// publishing to NATS. Events survive process crashes, power failures,
// and NATS outages.
//
// # Architecture
//
// The WAL sits between event generation and NATS publishing:
//
//	Event → WAL Write (ACID, fsync) → NATS Publish → WAL Confirm
//	                                              ↓ (on failure)
//	                                        Entry preserved for retry
//
// # Components
//
//   - BadgerWAL: Core WAL implementation using BadgerDB
//   - RetryLoop: Background goroutine that retries failed publishes
//   - Compactor: Background goroutine that cleans up confirmed entries
//
// # Usage
//
// Basic usage:
//
//	// Create configuration
//	cfg := wal.LoadConfig()
//
//	// Open WAL
//	w, err := wal.Open(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer w.Close()
//
//	// Write event before NATS publish
//	entryID, err := w.Write(ctx, event)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Publish to NATS
//	if err := publisher.Publish(event); err != nil {
//	    // Entry preserved in WAL for retry
//	    return err
//	}
//
//	// Confirm successful publish
//	if err := w.Confirm(ctx, entryID); err != nil {
//	    log.Printf("WAL confirm failed: %v", err)
//	}
//
// # Recovery
//
// On startup, recover pending entries from previous runs:
//
//	result, err := w.RecoverPending(ctx, publisher)
//	if err != nil {
//	    log.Printf("Recovery error: %v", err)
//	}
//	log.Printf("Recovered %d events", result.Recovered)
//
// # Background Processing
//
// Start retry loop and compactor for automatic handling:
//
//	// Start retry loop
//	retryLoop := wal.NewRetryLoop(w, publisher)
//	retryLoop.Start(ctx)
//	defer retryLoop.Stop()
//
//	// Start compactor
//	compactor := wal.NewCompactor(w)
//	compactor.Start(ctx)
//	defer compactor.Stop()
//
// # Build Tags
//
// The WAL is optional and can be disabled via build tags:
//
//	# Build with WAL
//	go build -tags wal ./cmd/server
//
//	# Build without WAL (no-op stub)
//	go build ./cmd/server
//
// # Configuration
//
// Configuration is loaded from environment variables:
//
//	WAL_ENABLED=true         # Enable WAL (default: true)
//	WAL_PATH=/data/wal       # Storage directory
//	WAL_SYNC_WRITES=true     # Force fsync (durability)
//	WAL_RETRY_INTERVAL=30s   # Retry loop interval
//	WAL_MAX_RETRIES=100      # Max attempts before giving up
//	WAL_RETRY_BACKOFF=5s     # Initial backoff duration
//	WAL_COMPACT_INTERVAL=1h  # Compaction interval
//	WAL_ENTRY_TTL=168h       # Entry time-to-live (7 days)
//
// # Why BadgerDB
//
// BadgerDB was chosen for:
//   - Pure Go (no CGO required)
//   - ACID compliance with checksums
//   - Concurrent writes (LSM-tree)
//   - Designed for write-heavy workloads
//   - Built-in TTL support
//
// Alternatives considered:
//   - bbolt: Single-writer limitation
//   - Append-only file: Corruption risk on power loss
//   - NATS KV: Requires network connection
//
// # Metrics
//
// Prometheus metrics are exported for monitoring:
//
//	wal_writes_total           # Total write operations
//	wal_confirms_total         # Total confirm operations
//	wal_retries_total          # Total retry attempts
//	wal_pending_entries        # Current pending count
//	wal_db_size_bytes          # Database size
//	wal_write_latency_seconds  # Write latency histogram
//
// # Thread Safety
//
// All WAL operations are thread-safe. Multiple goroutines can
// call Write, Confirm, and other methods concurrently.
package wal
