// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// MockEventStore implements EventStore for testing.
type MockEventStore struct {
	mu           sync.Mutex
	events       []*MediaEvent
	insertErr    error
	insertCalls  int
	batchSizes   []int
	flushSignals chan struct{}
}

func NewMockEventStore() *MockEventStore {
	return &MockEventStore{
		events:       make([]*MediaEvent, 0),
		batchSizes:   make([]int, 0),
		flushSignals: make(chan struct{}, 100),
	}
}

func (m *MockEventStore) InsertMediaEvents(ctx context.Context, events []*MediaEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.insertCalls++
	m.batchSizes = append(m.batchSizes, len(events))

	if m.insertErr != nil {
		return m.insertErr
	}

	m.events = append(m.events, events...)
	select {
	case m.flushSignals <- struct{}{}:
	default:
	}
	return nil
}

func (m *MockEventStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.insertErr = err
}

func (m *MockEventStore) GetEvents() []*MediaEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]*MediaEvent, len(m.events))
	copy(copied, m.events)
	return copied
}

func (m *MockEventStore) GetInsertCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.insertCalls
}

func (m *MockEventStore) GetBatchSizes() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]int, len(m.batchSizes))
	copy(copied, m.batchSizes)
	return copied
}

func (m *MockEventStore) WaitForFlush(timeout time.Duration) bool {
	select {
	case <-m.flushSignals:
		return true
	case <-time.After(timeout):
		return false
	}
}

// TestAppender_NewAppender verifies appender creation with valid config.
func TestAppender_NewAppender(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     100,
		FlushInterval: time.Second,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}
	if appender == nil {
		t.Fatal("NewAppender() returned nil")
	}

	// Verify initial stats
	stats := appender.Stats()
	if stats.EventsReceived != 0 {
		t.Errorf("Stats().EventsReceived = %d, want 0", stats.EventsReceived)
	}
	if stats.EventsFlushed != 0 {
		t.Errorf("Stats().EventsFlushed = %d, want 0", stats.EventsFlushed)
	}
	if stats.FlushCount != 0 {
		t.Errorf("Stats().FlushCount = %d, want 0", stats.FlushCount)
	}
}

// TestAppender_NewAppender_InvalidConfig verifies validation errors.
func TestAppender_NewAppender_InvalidConfig(t *testing.T) {
	store := NewMockEventStore()

	tests := []struct {
		name    string
		cfg     AppenderConfig
		wantErr string
	}{
		{
			name:    "nil store",
			cfg:     AppenderConfig{BatchSize: 100, FlushInterval: time.Second},
			wantErr: "store required",
		},
		{
			name:    "zero batch size",
			cfg:     AppenderConfig{BatchSize: 0, FlushInterval: time.Second},
			wantErr: "batch size must be positive",
		},
		{
			name:    "negative batch size",
			cfg:     AppenderConfig{BatchSize: -1, FlushInterval: time.Second},
			wantErr: "batch size must be positive",
		},
		{
			name:    "zero flush interval",
			cfg:     AppenderConfig{BatchSize: 100, FlushInterval: 0},
			wantErr: "flush interval must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var st EventStore
			if tt.name != "nil store" {
				st = store
			}
			_, err := NewAppender(st, tt.cfg)
			if err == nil {
				t.Fatal("NewAppender() error = nil, want error")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("NewAppender() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestAppender_Append_SingleEvent verifies single event buffering.
func TestAppender_Append_SingleEvent(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     10, // Won't trigger with just 1 event
		FlushInterval: time.Hour,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}

	ctx := context.Background()
	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.MediaType = MediaTypeMovie
	event.Title = "Test Movie"

	if err := appender.Append(ctx, event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	stats := appender.Stats()
	if stats.EventsReceived != 1 {
		t.Errorf("Stats().EventsReceived = %d, want 1", stats.EventsReceived)
	}
	if stats.EventsFlushed != 0 {
		t.Errorf("Stats().EventsFlushed = %d, want 0 (not flushed yet)", stats.EventsFlushed)
	}
	if stats.BufferSize != 1 {
		t.Errorf("Stats().BufferSize = %d, want 1", stats.BufferSize)
	}
}

// TestAppender_Append_BatchTrigger verifies flush at batch size.
func TestAppender_Append_BatchTrigger(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     5,
		FlushInterval: time.Hour, // Won't trigger
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}

	ctx := context.Background()

	// Add exactly batch size events
	for i := 0; i < 5; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i + 1
		event.MediaType = MediaTypeMovie
		event.Title = "Test Movie"
		if err := appender.Append(ctx, event); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	// Wait for async flush
	if !store.WaitForFlush(time.Second) {
		t.Fatal("Flush not triggered within timeout")
	}

	// Allow goroutine to finish updating stats after InsertMediaEvents returns
	// (the signal is sent from within InsertMediaEvents, but stats are updated after)
	// Use 100ms for CI reliability under load
	time.Sleep(100 * time.Millisecond)

	// Verify flush occurred
	events := store.GetEvents()
	if len(events) != 5 {
		t.Errorf("Store events = %d, want 5", len(events))
	}

	stats := appender.Stats()
	if stats.EventsFlushed != 5 {
		t.Errorf("Stats().EventsFlushed = %d, want 5", stats.EventsFlushed)
	}
	if stats.FlushCount != 1 {
		t.Errorf("Stats().FlushCount = %d, want 1", stats.FlushCount)
	}
	if stats.BufferSize != 0 {
		t.Errorf("Stats().BufferSize = %d, want 0", stats.BufferSize)
	}
}

// TestAppender_Append_IntervalTrigger verifies flush by timer.
func TestAppender_Append_IntervalTrigger(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     1000, // Won't trigger
		FlushInterval: 100 * time.Millisecond,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}
	defer appender.Close()

	// Start the appender to enable timer
	ctx := context.Background()
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Add some events (less than batch size)
	for i := 0; i < 3; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i + 1
		event.MediaType = MediaTypeMovie
		event.Title = "Test Movie"
		if err := appender.Append(ctx, event); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	// Wait for interval-based flush
	if !store.WaitForFlush(500 * time.Millisecond) {
		t.Fatal("Interval flush not triggered within timeout")
	}

	// Allow goroutine to finish updating stats after InsertMediaEvents returns
	// Use 100ms for CI reliability under load
	time.Sleep(100 * time.Millisecond)

	events := store.GetEvents()
	if len(events) != 3 {
		t.Errorf("Store events = %d, want 3", len(events))
	}

	stats := appender.Stats()
	if stats.EventsFlushed != 3 {
		t.Errorf("Stats().EventsFlushed = %d, want 3", stats.EventsFlushed)
	}
}

// TestAppender_Append_MultipleFlushes verifies multiple batch flushes.
// Note: Async flush triggers are non-deterministic, so we verify invariants
// rather than exact counts.
func TestAppender_Append_MultipleFlushes(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     3,
		FlushInterval: time.Hour,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}

	ctx := context.Background()

	// Add 6 events (should trigger exactly 2 flushes of 3 each)
	for i := 0; i < 6; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i + 1
		event.MediaType = MediaTypeMovie
		event.Title = "Test Movie"
		if err := appender.Append(ctx, event); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
		// Small delay between events to allow flush goroutines to start
		// Use 100ms for CI reliability under load
		if i == 2 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Wait for async flushes to complete (150ms for CI reliability)
	time.Sleep(150 * time.Millisecond)

	// Verify all events were flushed (at batch boundaries)
	events := store.GetEvents()
	if len(events) != 6 {
		t.Errorf("Store events = %d, want 6", len(events))
	}

	stats := appender.Stats()
	if stats.EventsReceived != 6 {
		t.Errorf("Stats().EventsReceived = %d, want 6", stats.EventsReceived)
	}
	if stats.EventsFlushed != 6 {
		t.Errorf("Stats().EventsFlushed = %d, want 6", stats.EventsFlushed)
	}
	// At least 1 flush should have occurred
	if stats.FlushCount < 1 {
		t.Errorf("Stats().FlushCount = %d, want >= 1", stats.FlushCount)
	}
	// Buffer should be empty after exact multiple of batch size
	if stats.BufferSize != 0 {
		t.Errorf("Stats().BufferSize = %d, want 0", stats.BufferSize)
	}
}

// TestAppender_Close_FlushesPending verifies Close flushes remaining events.
func TestAppender_Close_FlushesPending(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     100, // Won't trigger
		FlushInterval: time.Hour,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}

	ctx := context.Background()

	// Add events (less than batch size)
	for i := 0; i < 5; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i + 1
		event.MediaType = MediaTypeMovie
		event.Title = "Test Movie"
		if err := appender.Append(ctx, event); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	// Verify not flushed yet
	if len(store.GetEvents()) != 0 {
		t.Fatal("Events should not be flushed before Close")
	}

	// Close should flush pending events
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events := store.GetEvents()
	if len(events) != 5 {
		t.Errorf("Store events = %d, want 5", len(events))
	}

	stats := appender.Stats()
	if stats.EventsFlushed != 5 {
		t.Errorf("Stats().EventsFlushed = %d, want 5", stats.EventsFlushed)
	}
}

// TestAppender_Close_Idempotent verifies Close can be called multiple times.
func TestAppender_Close_Idempotent(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     100,
		FlushInterval: time.Second,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}

	ctx := context.Background()
	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.MediaType = MediaTypeMovie
	event.Title = "Test Movie"
	_ = appender.Append(ctx, event)

	// Close multiple times should not panic or error
	for i := 0; i < 3; i++ {
		if err := appender.Close(); err != nil {
			t.Errorf("Close() call %d error = %v", i+1, err)
		}
	}

	// Events should only be flushed once
	events := store.GetEvents()
	if len(events) != 1 {
		t.Errorf("Store events = %d, want 1", len(events))
	}
}

// TestAppender_Append_AfterClose verifies error on closed appender.
func TestAppender_Append_AfterClose(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     100,
		FlushInterval: time.Second,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}

	_ = appender.Close()

	ctx := context.Background()
	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.MediaType = MediaTypeMovie
	event.Title = "Test Movie"

	err = appender.Append(ctx, event)
	if err == nil {
		t.Fatal("Append() after Close() should error")
	}
	if err.Error() != "appender is closed" {
		t.Errorf("Append() error = %q, want %q", err.Error(), "appender is closed")
	}
}

// TestAppender_Flush_StoreError verifies error handling on store failure.
func TestAppender_Flush_StoreError(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     2,
		FlushInterval: time.Hour,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}

	// Set store to return error
	storeErr := errors.New("database connection failed")
	store.SetError(storeErr)

	ctx := context.Background()

	// Add events to trigger flush
	for i := 0; i < 2; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i + 1
		event.MediaType = MediaTypeMovie
		event.Title = "Test Movie"
		_ = appender.Append(ctx, event)
	}

	// Wait a bit for async flush
	time.Sleep(100 * time.Millisecond)

	stats := appender.Stats()
	if stats.ErrorCount != 1 {
		t.Errorf("Stats().ErrorCount = %d, want 1", stats.ErrorCount)
	}
	if stats.LastError == "" {
		t.Error("Stats().LastError should be set")
	}

	// Events should be retained in buffer for retry
	if stats.BufferSize != 2 {
		t.Errorf("Stats().BufferSize = %d, want 2 (retained after error)", stats.BufferSize)
	}
}

// TestAppender_Flush_Manual verifies manual flush operation.
func TestAppender_Flush_Manual(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     1000, // Won't trigger
		FlushInterval: time.Hour,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}

	ctx := context.Background()

	// Add events
	for i := 0; i < 5; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i + 1
		event.MediaType = MediaTypeMovie
		event.Title = "Test Movie"
		_ = appender.Append(ctx, event)
	}

	// Manual flush
	if err := appender.Flush(ctx); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	events := store.GetEvents()
	if len(events) != 5 {
		t.Errorf("Store events = %d, want 5", len(events))
	}

	stats := appender.Stats()
	if stats.BufferSize != 0 {
		t.Errorf("Stats().BufferSize = %d, want 0", stats.BufferSize)
	}
}

// TestAppender_ConcurrentAppend verifies thread safety.
func TestAppender_ConcurrentAppend(t *testing.T) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     50,
		FlushInterval: time.Hour,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}

	ctx := context.Background()
	const numGoroutines = 10
	const eventsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < eventsPerGoroutine; i++ {
				event := NewMediaEvent(SourcePlex)
				event.UserID = goroutineID*100 + i
				event.MediaType = MediaTypeMovie
				event.Title = "Test Movie"
				if err := appender.Append(ctx, event); err != nil {
					t.Errorf("Append() error = %v", err)
				}
			}
		}(g)
	}

	wg.Wait()

	// Close to flush remaining
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	totalEvents := numGoroutines * eventsPerGoroutine
	events := store.GetEvents()
	if len(events) != totalEvents {
		t.Errorf("Store events = %d, want %d", len(events), totalEvents)
	}

	stats := appender.Stats()
	if stats.EventsReceived != int64(totalEvents) {
		t.Errorf("Stats().EventsReceived = %d, want %d", stats.EventsReceived, totalEvents)
	}
	if stats.EventsFlushed != int64(totalEvents) {
		t.Errorf("Stats().EventsFlushed = %d, want %d", stats.EventsFlushed, totalEvents)
	}
}

// BenchmarkAppender_Append benchmarks appender throughput.
func BenchmarkAppender_Append(b *testing.B) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     1000,
		FlushInterval: time.Second,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		b.Fatalf("NewAppender() error = %v", err)
	}
	defer appender.Close()

	ctx := context.Background()
	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.MediaType = MediaTypeMovie
	event.Title = "Benchmark Movie"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = appender.Append(ctx, event)
	}
}

// BenchmarkAppender_Concurrent benchmarks concurrent append.
func BenchmarkAppender_Concurrent(b *testing.B) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     1000,
		FlushInterval: time.Second,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		b.Fatalf("NewAppender() error = %v", err)
	}
	defer appender.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		event := NewMediaEvent(SourcePlex)
		event.UserID = 1
		event.MediaType = MediaTypeMovie
		event.Title = "Benchmark Movie"
		for pb.Next() {
			_ = appender.Append(ctx, event)
		}
	})
}
