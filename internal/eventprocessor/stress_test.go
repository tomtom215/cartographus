// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Stress tests for the EventProcessor components.
// These tests verify behavior under high load conditions.

// TestStress_HighMessageVolume tests appender with 10,000+ events.
func TestStress_HighMessageVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     1000, // Large batch for throughput
		FlushInterval: time.Second,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}
	defer appender.Close()

	ctx := context.Background()
	const totalEvents = 10000

	start := time.Now()

	for i := 0; i < totalEvents; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i
		event.MediaType = MediaTypeMovie
		event.Title = "Stress Test Movie"
		event.StartedAt = time.Now()

		if err := appender.Append(ctx, event); err != nil {
			t.Fatalf("Append() event %d error = %v", i, err)
		}
	}

	// Force flush remaining
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	elapsed := time.Since(start)

	events := store.GetEvents()
	if len(events) != totalEvents {
		t.Errorf("Store events = %d, want %d", len(events), totalEvents)
	}

	throughput := float64(totalEvents) / elapsed.Seconds()
	t.Logf("High volume test: %d events in %v (%.0f events/sec)", totalEvents, elapsed, throughput)

	// Minimum throughput expectation (should handle >1000 events/sec)
	if throughput < 1000 {
		t.Errorf("Throughput %.0f events/sec is below minimum expectation of 1000", throughput)
	}
}

// TestStress_ConcurrentConsumerGroups simulates multiple consumer groups.
func TestStress_ConcurrentConsumerGroups(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Shared store simulates multiple consumers writing to same database
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     100,
		FlushInterval: 100 * time.Millisecond,
	}

	const numConsumerGroups = 10
	const eventsPerGroup = 1000

	var wg sync.WaitGroup
	var totalAppended atomic.Int64
	var errors atomic.Int64

	ctx := context.Background()

	// Simulate multiple consumer groups
	for group := 0; group < numConsumerGroups; group++ {
		wg.Add(1)
		go func(groupID int) {
			defer wg.Done()

			// Each consumer group has its own appender
			appender, err := NewAppender(store, cfg)
			if err != nil {
				errors.Add(1)
				t.Errorf("Consumer group %d: NewAppender() error = %v", groupID, err)
				return
			}
			defer appender.Close()

			// Start the appender timer
			if err := appender.Start(ctx); err != nil {
				errors.Add(1)
				t.Errorf("Consumer group %d: Start() error = %v", groupID, err)
				return
			}

			// Process events for this consumer group
			for i := 0; i < eventsPerGroup; i++ {
				event := NewMediaEvent(SourcePlex)
				event.UserID = groupID*10000 + i
				event.MediaType = MediaTypeMovie
				event.Title = "Consumer Group Test"
				event.StartedAt = time.Now()

				if err := appender.Append(ctx, event); err != nil {
					errors.Add(1)
					t.Errorf("Consumer group %d: Append() event %d error = %v", groupID, i, err)
					return
				}
				totalAppended.Add(1)
			}
		}(group)
	}

	wg.Wait()

	if errors.Load() > 0 {
		t.Fatalf("Had %d errors during concurrent consumer test", errors.Load())
	}

	// Wait for all async flushes
	time.Sleep(500 * time.Millisecond)

	totalExpected := numConsumerGroups * eventsPerGroup
	events := store.GetEvents()

	t.Logf("Concurrent consumer groups: %d groups x %d events = %d total, stored %d",
		numConsumerGroups, eventsPerGroup, totalExpected, len(events))

	if len(events) != totalExpected {
		t.Errorf("Store events = %d, want %d", len(events), totalExpected)
	}

	// Verify no duplicate user IDs (unique events)
	userIDs := make(map[int]bool)
	for _, e := range events {
		if userIDs[e.UserID] {
			t.Errorf("Duplicate event with UserID %d found", e.UserID)
		}
		userIDs[e.UserID] = true
	}
}

// SlowMockEventStore simulates a slow consumer (backpressure scenario).
type SlowMockEventStore struct {
	MockEventStore
	delay time.Duration
}

func NewSlowMockEventStore(delay time.Duration) *SlowMockEventStore {
	return &SlowMockEventStore{
		MockEventStore: *NewMockEventStore(),
		delay:          delay,
	}
}

func (s *SlowMockEventStore) InsertMediaEvents(ctx context.Context, events []*MediaEvent) error {
	// Simulate slow database/network
	time.Sleep(s.delay)
	return s.MockEventStore.InsertMediaEvents(ctx, events)
}

// TestStress_Backpressure tests appender behavior when store is slow.
func TestStress_Backpressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Slow store that takes 50ms per batch insert
	store := NewSlowMockEventStore(50 * time.Millisecond)
	cfg := AppenderConfig{
		BatchSize:     100,
		FlushInterval: time.Hour, // Only flush on batch size
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}
	defer appender.Close()

	ctx := context.Background()
	const totalEvents = 500 // 5 batches of 100

	start := time.Now()

	// Rapidly add events (faster than store can process)
	for i := 0; i < totalEvents; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i
		event.MediaType = MediaTypeMovie
		event.Title = "Backpressure Test"
		event.StartedAt = time.Now()

		if err := appender.Append(ctx, event); err != nil {
			t.Fatalf("Append() event %d error = %v", i, err)
		}
	}

	// Force flush and wait for slow store
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	elapsed := time.Since(start)

	events := store.GetEvents()
	if len(events) != totalEvents {
		t.Errorf("Store events = %d, want %d", len(events), totalEvents)
	}

	stats := appender.Stats()
	t.Logf("Backpressure test: %d events, %d flushes in %v (store delay: %v)",
		totalEvents, stats.FlushCount, elapsed, store.delay)

	// With 5 batches at 50ms each, minimum time should be ~250ms
	expectedMinTime := time.Duration(stats.FlushCount) * store.delay
	if elapsed < expectedMinTime {
		t.Errorf("Elapsed %v is less than expected minimum %v (backpressure may not be working)",
			elapsed, expectedMinTime)
	}
}

// TestStress_BurstTraffic tests handling of traffic bursts.
func TestStress_BurstTraffic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     100,
		FlushInterval: 50 * time.Millisecond,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}
	defer appender.Close()

	ctx := context.Background()
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Simulate burst pattern: 5 bursts of 200 events with pauses
	const numBursts = 5
	const eventsPerBurst = 200
	const burstPause = 100 * time.Millisecond

	for burst := 0; burst < numBursts; burst++ {
		// Rapid burst of events
		for i := 0; i < eventsPerBurst; i++ {
			event := NewMediaEvent(SourcePlex)
			event.UserID = burst*10000 + i
			event.MediaType = MediaTypeMovie
			event.Title = "Burst Test"
			event.StartedAt = time.Now()

			if err := appender.Append(ctx, event); err != nil {
				t.Fatalf("Burst %d: Append() event %d error = %v", burst, i, err)
			}
		}

		// Pause between bursts
		if burst < numBursts-1 {
			time.Sleep(burstPause)
		}
	}

	// Close and flush remaining
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	totalExpected := numBursts * eventsPerBurst
	events := store.GetEvents()

	t.Logf("Burst traffic test: %d bursts x %d events = %d total, stored %d",
		numBursts, eventsPerBurst, totalExpected, len(events))

	if len(events) != totalExpected {
		t.Errorf("Store events = %d, want %d", len(events), totalExpected)
	}
}

// TestStress_LongRunning tests appender over extended period with timer flushes.
func TestStress_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     1000, // Large batch (won't trigger by size)
		FlushInterval: 50 * time.Millisecond,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}
	defer appender.Close()

	ctx := context.Background()
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Simulate steady stream of events over 1 second
	const testDuration = 1 * time.Second
	const eventsPerSecond = 100

	start := time.Now()
	eventCount := 0
	eventInterval := time.Second / time.Duration(eventsPerSecond)

	ticker := time.NewTicker(eventInterval)
	defer ticker.Stop()

	timeout := time.After(testDuration)

	for {
		select {
		case <-timeout:
			goto done
		case <-ticker.C:
			event := NewMediaEvent(SourcePlex)
			event.UserID = eventCount
			event.MediaType = MediaTypeMovie
			event.Title = "Long Running Test"
			event.StartedAt = time.Now()

			if err := appender.Append(ctx, event); err != nil {
				t.Fatalf("Append() event %d error = %v", eventCount, err)
			}
			eventCount++
		}
	}

done:
	// Wait for final flush
	time.Sleep(100 * time.Millisecond)

	elapsed := time.Since(start)
	stats := appender.Stats()

	t.Logf("Long running test: %d events in %v, %d flushes (interval-based)",
		eventCount, elapsed, stats.FlushCount)

	// Should have had multiple timer-based flushes
	if stats.FlushCount < 5 {
		t.Errorf("FlushCount = %d, expected at least 5 interval-based flushes in 1 second", stats.FlushCount)
	}

	events := store.GetEvents()
	if len(events) != eventCount {
		t.Errorf("Store events = %d, want %d", len(events), eventCount)
	}
}

// TestStress_RaceConditions runs with -race to detect race conditions.
// Note: Flush() must not be called concurrently with Append() due to
// sync.WaitGroup semantics (Add must not be called concurrently with Wait).
func TestStress_RaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     10,
		FlushInterval: 10 * time.Millisecond,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}

	ctx := context.Background()
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	const numWriters = 20
	const eventsPerWriter = 100

	var wg sync.WaitGroup

	// Concurrent writers
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < eventsPerWriter; i++ {
				event := NewMediaEvent(SourcePlex)
				event.UserID = writerID*1000 + i
				event.MediaType = MediaTypeMovie
				event.Title = "Race Condition Test"
				event.StartedAt = time.Now()
				_ = appender.Append(ctx, event)
			}
		}(w)
	}

	// Concurrent stat readers (safe to call concurrently)
	for r := 0; r < 5; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = appender.Stats()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Wait for all writers to complete before testing flush
	wg.Wait()

	// Now test sequential flushes (safe after writers complete)
	for f := 0; f < 5; f++ {
		_ = appender.Flush(ctx)
		time.Sleep(5 * time.Millisecond)
	}

	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// All events should be stored (test passes if no race detected)
	totalExpected := numWriters * eventsPerWriter
	events := store.GetEvents()
	if len(events) != totalExpected {
		t.Errorf("Store events = %d, want %d", len(events), totalExpected)
	}
}

// BenchmarkStress_Throughput measures maximum throughput under load.
func BenchmarkStress_Throughput(b *testing.B) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     10000,
		FlushInterval: time.Hour,
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
	event.Title = "Throughput Benchmark"
	event.StartedAt = time.Now()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localEvent := *event // Copy to avoid contention on shared event
		for pb.Next() {
			_ = appender.Append(ctx, &localEvent)
		}
	})
}
