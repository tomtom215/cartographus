// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/metrics"
)

// EventStore defines the interface for persisting media events.
// Implementations include DuckDB database and mock stores for testing.
type EventStore interface {
	// InsertMediaEvents inserts a batch of media events.
	// Implementations should handle deduplication and validation.
	InsertMediaEvents(ctx context.Context, events []*MediaEvent) error
}

// AppenderStats holds runtime statistics for monitoring.
type AppenderStats struct {
	EventsReceived int64         // Total events received via Append
	EventsFlushed  int64         // Total events successfully written to store
	FlushCount     int64         // Number of flush operations
	ErrorCount     int64         // Number of failed flushes
	LastFlushTime  time.Time     // Time of last successful flush
	LastError      string        // Last error message
	BufferSize     int           // Current buffer size
	AvgFlushTime   time.Duration // Average flush duration
}

// Appender provides batch buffering and periodic flushing of media events.
// It buffers incoming events and writes them to the store in batches,
// either when the batch size is reached or the flush interval elapses.
//
// Key features:
//   - Thread-safe concurrent appends
//   - Configurable batch size and flush interval
//   - Graceful shutdown with pending event flush
//   - Error retry with buffer retention
//   - Metrics for monitoring
//
// DETERMINISM: Flush operations are serialized via flushMu to ensure consistent
// insert ordering. Without this, timer-based flushes and batch-triggered flushes
// could overlap, causing non-deterministic insert sequences in the database.
type Appender struct {
	store  EventStore
	config AppenderConfig

	// Buffer management
	mu     sync.Mutex
	buffer []*MediaEvent

	// DETERMINISM: Flush serialization mutex ensures only one flush runs at a time.
	// This prevents race conditions between timer-based and batch-triggered flushes
	// that could cause non-deterministic event ordering in the database.
	flushMu sync.Mutex

	// State management
	closed   atomic.Bool
	started  atomic.Bool
	stopChan chan struct{}
	doneChan chan struct{}
	flushWg  sync.WaitGroup // Tracks in-flight async flushes for graceful shutdown

	// Metrics (atomic for thread-safe reads)
	eventsReceived atomic.Int64
	eventsFlushed  atomic.Int64
	flushCount     atomic.Int64
	errorCount     atomic.Int64
	lastFlushTime  atomic.Value // stores time.Time
	lastError      atomic.Value // stores string
	totalFlushTime atomic.Int64 // nanoseconds for averaging
}

// NewAppender creates a new Appender with the given store and configuration.
// Returns an error if the store is nil or configuration is invalid.
func NewAppender(store EventStore, cfg AppenderConfig) (*Appender, error) {
	if store == nil {
		return nil, fmt.Errorf("store required")
	}
	if cfg.BatchSize <= 0 {
		return nil, fmt.Errorf("batch size must be positive")
	}
	if cfg.FlushInterval <= 0 {
		return nil, fmt.Errorf("flush interval must be positive")
	}

	a := &Appender{
		store:    store,
		config:   cfg,
		buffer:   make([]*MediaEvent, 0, cfg.BatchSize),
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}

	// Initialize atomic values
	a.lastFlushTime.Store(time.Time{})
	a.lastError.Store("")

	return a, nil
}

// Start begins the periodic flush timer.
// Must be called to enable interval-based flushing.
// Safe to call multiple times (idempotent).
func (a *Appender) Start(ctx context.Context) error {
	if a.closed.Load() {
		return fmt.Errorf("appender is closed")
	}
	if a.started.Swap(true) {
		return nil // Already started
	}

	go a.flushLoop(ctx)
	return nil
}

// Append adds an event to the buffer.
// Returns an error if the appender is closed.
// If the buffer reaches batch size, an async flush is triggered.
func (a *Appender) Append(ctx context.Context, event *MediaEvent) error {
	if a.closed.Load() {
		return fmt.Errorf("appender is closed")
	}

	a.mu.Lock()
	a.buffer = append(a.buffer, event)
	bufferSize := len(a.buffer)
	received := a.eventsReceived.Add(1)
	needsFlush := bufferSize >= a.config.BatchSize
	a.mu.Unlock()

	// TRACING: Log every event appended to buffer
	logging.Trace().
		Int64("received", received).
		Str("session_key", event.SessionKey).
		Str("event_id", event.EventID).
		Int("buffer_size", bufferSize).
		Int("batch_size", a.config.BatchSize).
		Msg("APPENDER: BUFFERED")

	if needsFlush {
		a.flushWg.Add(1)
		go func() {
			defer a.flushWg.Done()
			// CRITICAL FIX: Use a detached context with timeout for async flush.
			// The caller's context (e.g., Watermill message context) may be canceled
			// when the message handler returns, but the flush must complete to persist
			// data. Using the caller's context caused "context canceled" errors when
			// the goroutine attempted to begin a transaction after ctx was canceled.
			flushCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			a.doFlush(flushCtx)
		}()
	}

	return nil
}

// Flush manually flushes all buffered events.
// Blocks until the flush completes or errors.
// Also waits for any in-flight async flushes to complete first.
func (a *Appender) Flush(ctx context.Context) error {
	// Wait for any in-flight async flushes to complete first
	a.flushWg.Wait()
	return a.doFlushSync(ctx)
}

// Close stops the appender and flushes any pending events.
// Safe to call multiple times (idempotent).
func (a *Appender) Close() error {
	if a.closed.Swap(true) {
		return nil // Already closed
	}

	// Stop flush loop if running
	if a.started.Load() {
		close(a.stopChan)
		<-a.doneChan
	}

	// Wait for any in-flight async flushes to complete
	// This ensures all batch-triggered flushes finish before final flush
	a.flushWg.Wait()

	// Final flush of pending events
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.doFlushSync(ctx)
}

// Stats returns current runtime statistics.
func (a *Appender) Stats() AppenderStats {
	a.mu.Lock()
	bufferSize := len(a.buffer)
	a.mu.Unlock()

	var avgFlushTime time.Duration
	if count := a.flushCount.Load(); count > 0 {
		avgFlushTime = time.Duration(a.totalFlushTime.Load() / count)
	}

	var lastFlushTime time.Time
	if t, ok := a.lastFlushTime.Load().(time.Time); ok {
		lastFlushTime = t
	}
	var lastError string
	if e, ok := a.lastError.Load().(string); ok {
		lastError = e
	}

	return AppenderStats{
		EventsReceived: a.eventsReceived.Load(),
		EventsFlushed:  a.eventsFlushed.Load(),
		FlushCount:     a.flushCount.Load(),
		ErrorCount:     a.errorCount.Load(),
		LastFlushTime:  lastFlushTime,
		LastError:      lastError,
		BufferSize:     bufferSize,
		AvgFlushTime:   avgFlushTime,
	}
}

// flushLoop runs the periodic flush timer.
//
// CRITICAL: Timer-based flushes use a fresh context with 30s timeout, NOT the
// parent context. This prevents "context deadline exceeded" errors when the
// parent context (from suture supervisor) has issues or when flush operations
// take longer than expected. The parent context is ONLY used to detect shutdown.
func (a *Appender) flushLoop(ctx context.Context) {
	defer close(a.doneChan)

	ticker := time.NewTicker(a.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopChan:
			return
		case <-ticker.C:
			// CRITICAL FIX: Use a fresh context with timeout for timer-based flushes.
			// The parent context (from suture supervisor) should only control shutdown,
			// not impose deadlines on individual flush operations. This matches the
			// behavior of async flushes in Append() which use context.Background().
			flushCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			a.doFlush(flushCtx)
			cancel()
		}
	}
}

// doFlush performs an async flush (non-blocking).
// Error is logged but not returned since this is async.
func (a *Appender) doFlush(ctx context.Context) {
	if err := a.doFlushSync(ctx); err != nil {
		// Error already tracked in stats, just ensure it's handled
		a.lastError.Store(err.Error())
		logging.Debug().Err(err).Msg("APPENDER: Async flush error")
	}
}

// doFlushSync performs a synchronous flush.
// Returns nil if buffer is empty or flush succeeds.
// On error, events are retained in buffer for retry.
//
// CRITICAL: Flushes events in chunks of BatchSize to prevent OOM errors.
// Large buffers (e.g., 300 events) would cause DuckDB memory exhaustion
// if flushed all at once. Chunking ensures each insert stays within limits.
//
// DETERMINISM: Uses flushMu to serialize flush operations, ensuring
// consistent insert ordering regardless of concurrent flush triggers.
func (a *Appender) doFlushSync(ctx context.Context) error {
	// DETERMINISM: Serialize flush operations to ensure consistent ordering.
	// Without this, timer-based and batch-triggered flushes could interleave,
	// causing events to be inserted in non-deterministic order.
	a.flushMu.Lock()
	defer a.flushMu.Unlock()

	a.mu.Lock()
	bufferSize := len(a.buffer)
	if bufferSize == 0 {
		a.mu.Unlock()
		return nil
	}

	// Take ownership of buffer
	events := a.buffer
	a.buffer = make([]*MediaEvent, 0, a.config.BatchSize)
	a.mu.Unlock()

	logging.Debug().
		Int("count", len(events)).
		Int("batch_size", a.config.BatchSize).
		Msg("APPENDER: Flushing events to store")

	// CRITICAL FIX: Chunk events into batch-sized pieces to prevent OOM.
	// Previously, we flushed ALL events at once (e.g., 299 events), which
	// caused DuckDB to run out of memory during transaction processing.
	// Now we flush in chunks of BatchSize, allowing memory to be released
	// between batches.
	totalFlushed := 0
	totalStart := time.Now()

	for start := 0; start < len(events); start += a.config.BatchSize {
		end := start + a.config.BatchSize
		if end > len(events) {
			end = len(events)
		}
		chunk := events[start:end]

		logging.Debug().
			Int("start", start).
			Int("end", end).
			Int("total", len(events)).
			Msg("APPENDER: Flushing chunk")
		chunkStart := time.Now()
		err := a.store.InsertMediaEvents(ctx, chunk)
		chunkElapsed := time.Since(chunkStart)

		if err != nil {
			// Restore ONLY unflushed events to buffer for retry
			unflushed := events[start:]
			logging.Debug().
				Int("start", start).
				Err(err).
				Int("unflushed", len(unflushed)).
				Msg("APPENDER: Chunk insert failed, restoring unflushed events to buffer")
			a.mu.Lock()
			a.buffer = append(unflushed, a.buffer...)
			a.mu.Unlock()

			a.errorCount.Add(1)
			a.lastError.Store(err.Error())
			// Record partial success metrics
			if totalFlushed > 0 {
				a.eventsFlushed.Add(int64(totalFlushed))
				a.flushCount.Add(1)
				logging.Debug().
					Int("succeeded", totalFlushed).
					Msg("APPENDER: Partial flush - events succeeded before failure")
			}
			return fmt.Errorf("flush events (chunk %d-%d): %w", start, end, err)
		}

		logging.Debug().
			Int("start", start).
			Int("end", end).
			Dur("elapsed", chunkElapsed).
			Msg("APPENDER: Chunk flushed successfully")
		totalFlushed += len(chunk)

		// Record per-chunk metrics
		metrics.RecordNATSBatchFlush(chunkElapsed, len(chunk))
	}

	totalElapsed := time.Since(totalStart)
	logging.Debug().
		Int("count", totalFlushed).
		Dur("elapsed", totalElapsed).
		Int("chunks", (len(events)+a.config.BatchSize-1)/a.config.BatchSize).
		Msg("APPENDER: Successfully flushed all events")

	// Update success metrics
	a.eventsFlushed.Add(int64(totalFlushed))
	a.flushCount.Add(1)
	a.totalFlushTime.Add(totalElapsed.Nanoseconds())
	a.lastFlushTime.Store(time.Now())
	a.lastError.Store("")

	return nil
}
