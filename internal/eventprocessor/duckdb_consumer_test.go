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

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/goccy/go-json"
)

// MockMessageSource implements a mock message source for testing.
type MockMessageSource struct {
	messages chan *message.Message
	closed   bool
	mu       sync.Mutex
}

func NewMockMessageSource() *MockMessageSource {
	return &MockMessageSource{
		messages: make(chan *message.Message, 100),
	}
}

func (m *MockMessageSource) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return m.messages, nil
}

func (m *MockMessageSource) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.messages)
	}
	return nil
}

func (m *MockMessageSource) SendMessage(event *MediaEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	msg := message.NewMessage(event.EventID, data)
	m.messages <- msg
	return nil
}

// CountingEventStore wraps MockEventStore and counts inserts with a hook.
type CountingEventStore struct {
	*MockEventStore
	insertCount *atomic.Int32
}

func (c *CountingEventStore) InsertMediaEvents(ctx context.Context, events []*MediaEvent) error {
	c.insertCount.Add(int32(len(events)))
	return c.MockEventStore.InsertMediaEvents(ctx, events)
}

// TestDuckDBConsumer_NewDuckDBConsumer tests consumer creation.
func TestDuckDBConsumer_NewDuckDBConsumer(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appender, err := NewAppender(store, DefaultAppenderConfig())
	if err != nil {
		t.Fatalf("failed to create appender: %v", err)
	}

	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()

	consumer, err := NewDuckDBConsumer(source, appender, &cfg)
	if err != nil {
		t.Fatalf("NewDuckDBConsumer() error = %v", err)
	}
	if consumer == nil {
		t.Fatal("NewDuckDBConsumer() returned nil")
	}

	// Verify initial state
	if consumer.IsRunning() {
		t.Error("Consumer should not be running before Start()")
	}
}

// TestDuckDBConsumer_NewDuckDBConsumer_InvalidConfig tests error cases.
func TestDuckDBConsumer_NewDuckDBConsumer_InvalidConfig(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appender, _ := NewAppender(store, DefaultAppenderConfig())
	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()

	tests := []struct {
		name     string
		source   MessageSource
		appender *Appender
		wantErr  bool
	}{
		{
			name:     "nil source",
			source:   nil,
			appender: appender,
			wantErr:  true,
		},
		{
			name:     "nil appender",
			source:   source,
			appender: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDuckDBConsumer(tt.source, tt.appender, &cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDuckDBConsumer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDuckDBConsumer_ProcessMessages tests basic message processing.
func TestDuckDBConsumer_ProcessMessages(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 2
	appenderCfg.FlushInterval = 50 * time.Millisecond
	appender, err := NewAppender(store, appenderCfg)
	if err != nil {
		t.Fatalf("failed to create appender: %v", err)
	}

	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()
	cfg.Topic = "playback.>"

	consumer, err := NewDuckDBConsumer(source, appender, &cfg)
	if err != nil {
		t.Fatalf("NewDuckDBConsumer() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start consumer
	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("appender.Start() error = %v", err)
	}

	// Send test events
	event1 := NewMediaEvent("plex")
	event1.UserID = 1
	event1.Username = "user1"
	event1.MediaType = "movie"
	event1.Title = "Test Movie 1"
	if err := source.SendMessage(event1); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	event2 := NewMediaEvent("plex")
	event2.UserID = 2
	event2.Username = "user2"
	event2.MediaType = "episode"
	event2.Title = "Test Episode"
	if err := source.SendMessage(event2); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	// Wait for batch flush
	time.Sleep(200 * time.Millisecond)

	// Stop consumer
	consumer.Stop()
	appender.Close()

	// Verify events were stored
	events := store.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
}

// TestDuckDBConsumer_Deduplication tests session key deduplication.
func TestDuckDBConsumer_Deduplication(t *testing.T) {
	t.Parallel()

	var insertCount atomic.Int32
	baseStore := NewMockEventStore()
	store := &CountingEventStore{
		MockEventStore: baseStore,
		insertCount:    &insertCount,
	}

	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 10
	appenderCfg.FlushInterval = 50 * time.Millisecond
	appender, err := NewAppender(store, appenderCfg)
	if err != nil {
		t.Fatalf("failed to create appender: %v", err)
	}

	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()
	cfg.EnableDeduplication = true
	cfg.DeduplicationWindow = 1 * time.Minute

	consumer, err := NewDuckDBConsumer(source, appender, &cfg)
	if err != nil {
		t.Fatalf("NewDuckDBConsumer() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("appender.Start() error = %v", err)
	}

	// Send same event ID multiple times (simulating duplicates)
	event := NewMediaEvent("plex")
	event.EventID = "duplicate-event-id"
	event.UserID = 1
	event.Username = "user1"
	event.MediaType = "movie"
	event.Title = "Test Movie"

	// Send 3 copies of same event
	for i := 0; i < 3; i++ {
		if err := source.SendMessage(event); err != nil {
			t.Fatalf("SendMessage() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	consumer.Stop()
	appender.Close()

	// Only 1 event should be stored due to deduplication
	events := store.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event (deduplicated), got %d", len(events))
	}

	// Verify stats show duplicates detected
	stats := consumer.Stats()
	if stats.DuplicatesSkipped != 2 {
		t.Errorf("Expected 2 duplicates skipped, got %d", stats.DuplicatesSkipped)
	}
}

// TestDuckDBConsumer_Stop tests graceful shutdown.
func TestDuckDBConsumer_Stop(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appender, _ := NewAppender(store, DefaultAppenderConfig())
	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()

	consumer, _ := NewDuckDBConsumer(source, appender, &cfg)

	ctx := context.Background()
	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !consumer.IsRunning() {
		t.Error("Consumer should be running after Start()")
	}

	consumer.Stop()

	if consumer.IsRunning() {
		t.Error("Consumer should not be running after Stop()")
	}

	// Calling Stop again should be safe
	consumer.Stop()
}

// TestDuckDBConsumer_Stats tests statistics collection.
func TestDuckDBConsumer_Stats(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 10
	appenderCfg.FlushInterval = 50 * time.Millisecond
	appender, _ := NewAppender(store, appenderCfg)

	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()

	consumer, _ := NewDuckDBConsumer(source, appender, &cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	consumer.Start(ctx)
	appender.Start(ctx)

	// Send test events
	for i := 0; i < 5; i++ {
		event := NewMediaEvent("plex")
		event.UserID = i
		event.Username = "user"
		event.MediaType = "movie"
		event.Title = "Movie"
		source.SendMessage(event)
	}

	time.Sleep(200 * time.Millisecond)

	stats := consumer.Stats()
	if stats.MessagesReceived != 5 {
		t.Errorf("Expected 5 messages received, got %d", stats.MessagesReceived)
	}
	if stats.MessagesProcessed != 5 {
		t.Errorf("Expected 5 messages processed, got %d", stats.MessagesProcessed)
	}

	consumer.Stop()
	appender.Close()
}

// TestDuckDBConsumer_InvalidMessage tests handling of invalid JSON.
func TestDuckDBConsumer_InvalidMessage(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appender, _ := NewAppender(store, DefaultAppenderConfig())
	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()

	consumer, _ := NewDuckDBConsumer(source, appender, &cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	consumer.Start(ctx)
	appender.Start(ctx)

	// Send invalid JSON
	invalidMsg := message.NewMessage("invalid-id", []byte("not json"))
	source.messages <- invalidMsg

	time.Sleep(100 * time.Millisecond)

	stats := consumer.Stats()
	if stats.MessagesReceived != 1 {
		t.Errorf("Expected 1 message received, got %d", stats.MessagesReceived)
	}
	if stats.ParseErrors != 1 {
		t.Errorf("Expected 1 parse error, got %d", stats.ParseErrors)
	}

	consumer.Stop()
	appender.Close()
}

// TestDuckDBConsumer_ContextCancellation tests proper cancellation handling.
func TestDuckDBConsumer_ContextCancellation(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appender, _ := NewAppender(store, DefaultAppenderConfig())
	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()

	consumer, _ := NewDuckDBConsumer(source, appender, &cfg)

	ctx, cancel := context.WithCancel(context.Background())

	consumer.Start(ctx)

	// Cancel context
	cancel()

	// Wait for shutdown - must be longer than drainMessages timeout (100ms)
	// DETERMINISM: drainMessages ensures no data loss during graceful shutdown
	time.Sleep(150 * time.Millisecond)

	if consumer.IsRunning() {
		t.Error("Consumer should stop when context is canceled")
	}
}

// TestConsumerConfig_Defaults tests default configuration values.
func TestConsumerConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg := DefaultConsumerConfig()

	if cfg.Topic != "playback.>" {
		t.Errorf("Topic = %s, want playback.>", cfg.Topic)
	}
	if !cfg.EnableDeduplication {
		t.Error("EnableDeduplication should be true by default")
	}
	if cfg.DeduplicationWindow != 5*time.Minute {
		t.Errorf("DeduplicationWindow = %v, want 5m", cfg.DeduplicationWindow)
	}
	if cfg.MaxDeduplicationEntries != 10000 {
		t.Errorf("MaxDeduplicationEntries = %d, want 10000", cfg.MaxDeduplicationEntries)
	}
}

// TestDuckDBConsumer_DeduplicationExpiry tests that old dedup entries expire.
func TestDuckDBConsumer_DeduplicationExpiry(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 10
	appenderCfg.FlushInterval = 20 * time.Millisecond
	appender, _ := NewAppender(store, appenderCfg)

	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()
	cfg.EnableDeduplication = true
	cfg.DeduplicationWindow = 50 * time.Millisecond // Very short for testing

	consumer, _ := NewDuckDBConsumer(source, appender, &cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	consumer.Start(ctx)
	appender.Start(ctx)

	// Send event
	event := NewMediaEvent("plex")
	event.EventID = "expiry-test-id"
	event.UserID = 1
	event.Username = "user"
	event.MediaType = "movie"
	event.Title = "Movie"
	source.SendMessage(event)

	time.Sleep(30 * time.Millisecond)

	// Wait for deduplication window to expire
	time.Sleep(100 * time.Millisecond)

	// Send same event again - should NOT be deduplicated since window expired
	source.SendMessage(event)

	time.Sleep(100 * time.Millisecond)

	consumer.Stop()
	appender.Close()

	// Both events should be stored
	events := store.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events (dedup expired), got %d", len(events))
	}
}

// TestDuckDBConsumer_WithDLQ tests consumer with DLQ integration.
func TestDuckDBConsumer_WithDLQ(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 10
	appenderCfg.FlushInterval = 50 * time.Millisecond
	appender, err := NewAppender(store, appenderCfg)
	if err != nil {
		t.Fatalf("failed to create appender: %v", err)
	}

	source := NewMockMessageSource()

	// Create consumer with DLQ enabled
	cfg := DefaultConsumerConfig()
	cfg.EnableDLQ = true
	cfg.DLQConfig = DefaultDLQConfig()

	consumer, err := NewDuckDBConsumer(source, appender, &cfg)
	if err != nil {
		t.Fatalf("NewDuckDBConsumer() error = %v", err)
	}

	// Verify DLQ handler is created
	if consumer.dlqHandler == nil {
		t.Fatal("DLQ handler should be created when EnableDLQ=true")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("appender.Start() error = %v", err)
	}

	// Send valid event
	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.Username = "user1"
	event.MediaType = MediaTypeMovie
	event.Title = "Test Movie"
	if err := source.SendMessage(event); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	consumer.Stop()
	appender.Close()

	// Verify event was stored
	events := store.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	// Verify DLQ is empty (no failures)
	dlqStats := consumer.DLQStats()
	if dlqStats.TotalEntries != 0 {
		t.Errorf("DLQ should be empty, got %d entries", dlqStats.TotalEntries)
	}
}

// TestDuckDBConsumer_ParseErrorsToDLQ tests that parse errors go to DLQ.
func TestDuckDBConsumer_ParseErrorsToDLQ(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 10
	appenderCfg.FlushInterval = 50 * time.Millisecond
	appender, err := NewAppender(store, appenderCfg)
	if err != nil {
		t.Fatalf("failed to create appender: %v", err)
	}

	source := NewMockMessageSource()

	// Create consumer with DLQ enabled
	cfg := DefaultConsumerConfig()
	cfg.EnableDLQ = true
	cfg.DLQConfig = DefaultDLQConfig()

	consumer, err := NewDuckDBConsumer(source, appender, &cfg)
	if err != nil {
		t.Fatalf("NewDuckDBConsumer() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("appender.Start() error = %v", err)
	}

	// Send malformed message (will fail to parse)
	invalidMsg := message.NewMessage("invalid-msg-id", []byte("not valid json"))
	source.messages <- invalidMsg

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	consumer.Stop()
	appender.Close()

	// Verify parse error was sent to DLQ
	stats := consumer.Stats()
	if stats.ParseErrors != 1 {
		t.Errorf("Expected 1 parse error, got %d", stats.ParseErrors)
	}
	if stats.MessagesSentToDLQ != 1 {
		t.Errorf("Expected 1 message sent to DLQ, got %d", stats.MessagesSentToDLQ)
	}

	// Verify DLQ entry exists
	dlqStats := consumer.DLQStats()
	if dlqStats.TotalEntries != 1 {
		t.Errorf("Expected 1 DLQ entry, got %d", dlqStats.TotalEntries)
	}
	if dlqStats.EntriesByCategory[ErrorCategoryValidation] != 1 {
		t.Errorf("Expected 1 validation error in DLQ, got %d", dlqStats.EntriesByCategory[ErrorCategoryValidation])
	}
}

// TestDuckDBConsumer_DLQDisabled tests consumer works without DLQ.
func TestDuckDBConsumer_DLQDisabled(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 10
	appenderCfg.FlushInterval = 50 * time.Millisecond
	appender, _ := NewAppender(store, appenderCfg)

	source := NewMockMessageSource()

	// Create consumer without DLQ
	cfg := DefaultConsumerConfig()
	cfg.EnableDLQ = false

	consumer, err := NewDuckDBConsumer(source, appender, &cfg)
	if err != nil {
		t.Fatalf("NewDuckDBConsumer() error = %v", err)
	}

	// Verify DLQ handler is nil
	if consumer.dlqHandler != nil {
		t.Error("DLQ handler should be nil when EnableDLQ=false")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	consumer.Start(ctx)
	appender.Start(ctx)

	// Send event
	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.Username = "user1"
	event.MediaType = MediaTypeMovie
	event.Title = "Test Movie"
	source.SendMessage(event)

	time.Sleep(200 * time.Millisecond)

	consumer.Stop()
	appender.Close()

	events := store.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

// TestDuckDBConsumer_CorrelationKeyDeduplication tests cross-source deduplication via correlation key.
// This is CRITICAL for event sourcing mode where events from multiple sources
// (Tautulli sync, Plex webhooks, Jellyfin) may represent the same playback.
func TestDuckDBConsumer_CorrelationKeyDeduplication(t *testing.T) {
	t.Parallel()

	var insertCount atomic.Int32
	baseStore := NewMockEventStore()
	store := &CountingEventStore{
		MockEventStore: baseStore,
		insertCount:    &insertCount,
	}

	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 10
	appenderCfg.FlushInterval = 50 * time.Millisecond
	appender, err := NewAppender(store, appenderCfg)
	if err != nil {
		t.Fatalf("failed to create appender: %v", err)
	}

	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()
	cfg.EnableDeduplication = true
	cfg.DeduplicationWindow = 1 * time.Minute

	consumer, err := NewDuckDBConsumer(source, appender, &cfg)
	if err != nil {
		t.Fatalf("NewDuckDBConsumer() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("appender.Start() error = %v", err)
	}

	baseTime := time.Date(2024, 1, 15, 10, 32, 0, 0, time.UTC)

	// Simulate Plex webhook arriving first
	plexEvent := NewMediaEvent(SourcePlex)
	plexEvent.EventID = "plex-event-123"
	plexEvent.SessionKey = "webhook-device123-54321" // No colons - colons are delimiters in correlation key
	plexEvent.UserID = 12345
	plexEvent.Username = "testuser"
	plexEvent.MediaType = MediaTypeMovie
	plexEvent.Title = "Test Movie"
	plexEvent.RatingKey = "54321"
	plexEvent.MachineID = "device123" // Same device for cross-source dedup
	plexEvent.StartedAt = baseTime
	plexEvent.SetCorrelationKey() // Generate correlation key

	if err := source.SendMessage(plexEvent); err != nil {
		t.Fatalf("SendMessage(plex) error = %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Simulate Tautulli sync arriving later with different SessionKey but same content/device
	// NOTE: Cross-source deduplication requires EXACT same StartedAt timestamp
	// (correlation keys use second-precision timestamps, not 5-minute buckets)
	tautulliEvent := NewMediaEvent(SourceTautulli)
	tautulliEvent.EventID = "tautulli-event-456"         // Different EventID
	tautulliEvent.SessionKey = "tautulli-session-abc123" // Different SessionKey
	tautulliEvent.UserID = 12345                         // Same user
	tautulliEvent.Username = "testuser"
	tautulliEvent.MediaType = MediaTypeMovie
	tautulliEvent.Title = "Test Movie"
	tautulliEvent.RatingKey = "54321"     // Same content
	tautulliEvent.MachineID = "device123" // Same device - critical for cross-source dedup
	tautulliEvent.StartedAt = baseTime    // Same StartedAt for cross-source dedup
	tautulliEvent.SetCorrelationKey()     // Should generate matching cross-source key

	if err := source.SendMessage(tautulliEvent); err != nil {
		t.Fatalf("SendMessage(tautulli) error = %v", err)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	consumer.Stop()
	appender.Close()

	// Only 1 event should be stored due to cross-source deduplication
	events := store.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event (cross-source deduplicated), got %d", len(events))
	}

	// Verify stats show duplicate detected
	stats := consumer.Stats()
	if stats.DuplicatesSkipped != 1 {
		t.Errorf("Expected 1 duplicate skipped, got %d", stats.DuplicatesSkipped)
	}

	// v2.3: Correlation keys are intentionally DIFFERENT (include source prefix AND session_key)
	// Cross-source dedup works via getCrossSourceKey() which strips both source and session_key
	if plexEvent.CorrelationKey == tautulliEvent.CorrelationKey {
		t.Error("Correlation keys should be different (v2.3 includes source prefix and session_key)")
	}

	// Verify the cross-source keys match (this is what enables cross-source dedup)
	// getCrossSourceKey strips source prefix AND session_key suffix to get content-based portion
	plexCrossKey := getCrossSourceKey(plexEvent.CorrelationKey)
	tautulliCrossKey := getCrossSourceKey(tautulliEvent.CorrelationKey)
	if plexCrossKey != tautulliCrossKey {
		t.Errorf("Cross-source keys should match: %q != %q", plexCrossKey, tautulliCrossKey)
	}
}

// TestDuckDBConsumer_CorrelationKeyDifferentContent tests that different content is NOT deduplicated.
func TestDuckDBConsumer_CorrelationKeyDifferentContent(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 10
	appenderCfg.FlushInterval = 50 * time.Millisecond
	appender, err := NewAppender(store, appenderCfg)
	if err != nil {
		t.Fatalf("failed to create appender: %v", err)
	}

	source := NewMockMessageSource()
	cfg := DefaultConsumerConfig()
	cfg.EnableDeduplication = true
	cfg.DeduplicationWindow = 1 * time.Minute

	consumer, err := NewDuckDBConsumer(source, appender, &cfg)
	if err != nil {
		t.Fatalf("NewDuckDBConsumer() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("appender.Start() error = %v", err)
	}

	baseTime := time.Date(2024, 1, 15, 10, 32, 0, 0, time.UTC)

	// Event 1: User watching Movie A
	event1 := NewMediaEvent(SourcePlex)
	event1.UserID = 12345
	event1.Username = "testuser"
	event1.MediaType = MediaTypeMovie
	event1.Title = "Movie A"
	event1.RatingKey = "11111"
	event1.MachineID = "device123"
	event1.StartedAt = baseTime
	event1.SetCorrelationKey()

	// Event 2: Same user watching Movie B (different content)
	event2 := NewMediaEvent(SourcePlex)
	event2.UserID = 12345
	event2.Username = "testuser"
	event2.MediaType = MediaTypeMovie
	event2.Title = "Movie B"
	event2.RatingKey = "22222" // Different RatingKey
	event2.MachineID = "device123"
	event2.StartedAt = baseTime
	event2.SetCorrelationKey()

	if err := source.SendMessage(event1); err != nil {
		t.Fatalf("SendMessage(event1) error = %v", err)
	}
	if err := source.SendMessage(event2); err != nil {
		t.Fatalf("SendMessage(event2) error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	consumer.Stop()
	appender.Close()

	// Both events should be stored (different content)
	events := store.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events (different content), got %d", len(events))
	}

	// Verify correlation keys are different
	if event1.CorrelationKey == event2.CorrelationKey {
		t.Errorf("Correlation keys should be different for different content: %q", event1.CorrelationKey)
	}
}
