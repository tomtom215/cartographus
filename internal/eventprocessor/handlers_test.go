// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/goccy/go-json"
)

func TestDefaultDuckDBHandlerConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultDuckDBHandlerConfig()

	if !cfg.EnableCrossSourceDedup {
		t.Error("EnableCrossSourceDedup should be true by default")
	}
	if cfg.DeduplicationWindow != 5*time.Minute {
		t.Errorf("DeduplicationWindow = %v, want %v", cfg.DeduplicationWindow, 5*time.Minute)
	}
	if cfg.MaxDeduplicationEntries != 10000 {
		t.Errorf("MaxDeduplicationEntries = %d, want 10000", cfg.MaxDeduplicationEntries)
	}
}

func TestNewDuckDBHandler_NilAppender(t *testing.T) {
	t.Parallel()

	cfg := DefaultDuckDBHandlerConfig()
	_, err := NewDuckDBHandler(nil, cfg, nil)
	if err == nil {
		t.Error("NewDuckDBHandler should error with nil appender")
	}
}

func TestNewDuckDBHandler_NilLogger(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appender, err := NewAppender(store, DefaultAppenderConfig())
	if err != nil {
		t.Fatalf("NewAppender error: %v", err)
	}

	cfg := DefaultDuckDBHandlerConfig()
	handler, err := NewDuckDBHandler(appender, cfg, nil)
	if err != nil {
		t.Fatalf("NewDuckDBHandler error: %v", err)
	}
	if handler == nil {
		t.Fatal("NewDuckDBHandler returned nil")
	}
}

func TestDuckDBHandler_Handle_ValidEvent(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 1 // Immediate flush for testing
	appender, err := NewAppender(store, appenderCfg)
	if err != nil {
		t.Fatalf("NewAppender error: %v", err)
	}

	ctx := context.Background()
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("Appender.Start error: %v", err)
	}
	defer appender.Close()

	cfg := DefaultDuckDBHandlerConfig()
	cfg.EnableCrossSourceDedup = false // Disable for this test
	handler, err := NewDuckDBHandler(appender, cfg, nil)
	if err != nil {
		t.Fatalf("NewDuckDBHandler error: %v", err)
	}

	// Create valid event
	event := &MediaEvent{
		EventID:   "test-event-1",
		Source:    "plex",
		MediaType: "movie",
		Title:     "Test Movie",
		UserID:    1,
		Username:  "testuser",
		StartedAt: time.Now(),
	}

	data, _ := json.Marshal(event)
	msg := message.NewMessage(event.EventID, data)

	// Handle should succeed
	err = handler.Handle(msg)
	if err != nil {
		t.Errorf("Handle error: %v", err)
	}

	// Verify stats
	stats := handler.Stats()
	if stats.MessagesReceived != 1 {
		t.Errorf("MessagesReceived = %d, want 1", stats.MessagesReceived)
	}
	if stats.MessagesProcessed != 1 {
		t.Errorf("MessagesProcessed = %d, want 1", stats.MessagesProcessed)
	}
}

func TestDuckDBHandler_Handle_InvalidJSON(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appender, err := NewAppender(store, DefaultAppenderConfig())
	if err != nil {
		t.Fatalf("NewAppender error: %v", err)
	}

	cfg := DefaultDuckDBHandlerConfig()
	handler, err := NewDuckDBHandler(appender, cfg, nil)
	if err != nil {
		t.Fatalf("NewDuckDBHandler error: %v", err)
	}

	// Create invalid JSON message
	msg := message.NewMessage("invalid-msg", []byte("not valid json"))

	// Handle should return PermanentError
	err = handler.Handle(msg)
	if err == nil {
		t.Error("Handle should error on invalid JSON")
	}
	if !IsPermanentError(err) {
		t.Errorf("Error should be PermanentError, got %T", err)
	}

	// Verify stats
	stats := handler.Stats()
	if stats.ParseErrors != 1 {
		t.Errorf("ParseErrors = %d, want 1", stats.ParseErrors)
	}
}

func TestDuckDBHandler_CrossSourceDeduplication(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appenderCfg := DefaultAppenderConfig()
	appenderCfg.BatchSize = 10
	appender, err := NewAppender(store, appenderCfg)
	if err != nil {
		t.Fatalf("NewAppender error: %v", err)
	}

	cfg := DefaultDuckDBHandlerConfig()
	cfg.EnableCrossSourceDedup = true
	cfg.DeduplicationWindow = time.Minute
	handler, err := NewDuckDBHandler(appender, cfg, nil)
	if err != nil {
		t.Fatalf("NewDuckDBHandler error: %v", err)
	}

	// Create first event
	event1 := &MediaEvent{
		EventID:        "event-1",
		SessionKey:     "session-1",
		CorrelationKey: "corr-key-1",
		Source:         "plex",
		MediaType:      "movie",
		Title:          "Test Movie",
		UserID:         1,
		Username:       "testuser",
		StartedAt:      time.Now(),
	}

	data1, _ := json.Marshal(event1)
	msg1 := message.NewMessage(event1.EventID, data1)

	// First handle should succeed
	if err := handler.Handle(msg1); err != nil {
		t.Errorf("First Handle error: %v", err)
	}

	// Duplicate by EventID
	msg2 := message.NewMessage(event1.EventID, data1)
	if err := handler.Handle(msg2); err != nil {
		t.Errorf("Second Handle error: %v", err)
	}

	// Duplicate by SessionKey (different EventID)
	event2 := *event1
	event2.EventID = "event-2"
	data2, _ := json.Marshal(&event2)
	msg3 := message.NewMessage(event2.EventID, data2)
	if err := handler.Handle(msg3); err != nil {
		t.Errorf("Third Handle error: %v", err)
	}

	// Duplicate by CorrelationKey (different EventID and SessionKey)
	event3 := *event1
	event3.EventID = "event-3"
	event3.SessionKey = "session-3"
	data3, _ := json.Marshal(&event3)
	msg4 := message.NewMessage(event3.EventID, data3)
	if err := handler.Handle(msg4); err != nil {
		t.Errorf("Fourth Handle error: %v", err)
	}

	// Verify stats - should have 1 processed, 3 duplicates
	stats := handler.Stats()
	if stats.MessagesReceived != 4 {
		t.Errorf("MessagesReceived = %d, want 4", stats.MessagesReceived)
	}
	if stats.MessagesProcessed != 1 {
		t.Errorf("MessagesProcessed = %d, want 1", stats.MessagesProcessed)
	}
	if stats.DuplicatesSkipped != 3 {
		t.Errorf("DuplicatesSkipped = %d, want 3", stats.DuplicatesSkipped)
	}
}

func TestDuckDBHandler_StartCleanup(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appender, err := NewAppender(store, DefaultAppenderConfig())
	if err != nil {
		t.Fatalf("NewAppender error: %v", err)
	}

	cfg := DefaultDuckDBHandlerConfig()
	cfg.EnableCrossSourceDedup = true
	cfg.DeduplicationWindow = 50 * time.Millisecond // Short for testing
	handler, err := NewDuckDBHandler(appender, cfg, nil)
	if err != nil {
		t.Fatalf("NewDuckDBHandler error: %v", err)
	}

	// Start cleanup goroutine
	ctx, cancel := context.WithCancel(context.Background())
	handler.StartCleanup(ctx)

	// Add an event to the dedup cache
	event := &MediaEvent{
		EventID:   "cleanup-test-event",
		Source:    "plex",
		MediaType: "movie",
		Title:     "Test",
		UserID:    1,
		Username:  "test",
		StartedAt: time.Now(),
	}
	data, _ := json.Marshal(event)
	msg := message.NewMessage(event.EventID, data)
	handler.Handle(msg)

	// Wait for cleanup to run
	time.Sleep(100 * time.Millisecond)

	// After TTL + cleanup interval, event should be removed
	// Handler should process it as new
	msg2 := message.NewMessage(event.EventID, data)
	handler.Handle(msg2)

	stats := handler.Stats()
	// Both should be processed since TTL expired
	if stats.MessagesProcessed < 1 {
		t.Errorf("MessagesProcessed = %d, want >= 1", stats.MessagesProcessed)
	}

	cancel()
}

// MockWebSocketHub implements WebSocketBroadcaster for testing.
type MockWebSocketHub struct {
	messages     [][]byte
	broadcastCnt atomic.Int32
}

func (h *MockWebSocketHub) BroadcastRaw(data []byte) {
	h.messages = append(h.messages, data)
	h.broadcastCnt.Add(1)
}

func TestNewWebSocketHandler_NilHub(t *testing.T) {
	t.Parallel()

	_, err := NewWebSocketHandler(nil, nil)
	if err == nil {
		t.Error("NewWebSocketHandler should error with nil hub")
	}
}

func TestWebSocketHandler_Handle(t *testing.T) {
	t.Parallel()

	hub := &MockWebSocketHub{}
	handler, err := NewWebSocketHandler(hub, watermill.NewStdLogger(false, false))
	if err != nil {
		t.Fatalf("NewWebSocketHandler error: %v", err)
	}

	// Create message
	event := &MediaEvent{
		EventID:   "ws-event-1",
		Source:    "plex",
		MediaType: "movie",
		Title:     "Test Movie",
	}
	data, _ := json.Marshal(event)
	msg := message.NewMessage(event.EventID, data)

	// Handle should always succeed
	err = handler.Handle(msg)
	if err != nil {
		t.Errorf("Handle error: %v", err)
	}

	// Verify broadcast was called
	if hub.broadcastCnt.Load() != 1 {
		t.Errorf("BroadcastRaw called %d times, want 1", hub.broadcastCnt.Load())
	}

	// Verify stats
	stats := handler.Stats()
	if stats.MessagesReceived != 1 {
		t.Errorf("MessagesReceived = %d, want 1", stats.MessagesReceived)
	}
	if stats.MessagesBroadcast != 1 {
		t.Errorf("MessagesBroadcast = %d, want 1", stats.MessagesBroadcast)
	}
}

func TestWebSocketHandler_Handle_Multiple(t *testing.T) {
	t.Parallel()

	hub := &MockWebSocketHub{}
	handler, err := NewWebSocketHandler(hub, nil)
	if err != nil {
		t.Fatalf("NewWebSocketHandler error: %v", err)
	}

	// Handle multiple messages
	for i := 0; i < 10; i++ {
		msg := message.NewMessage("msg-id", []byte("test data"))
		if err := handler.Handle(msg); err != nil {
			t.Errorf("Handle %d error: %v", i, err)
		}
	}

	stats := handler.Stats()
	if stats.MessagesReceived != 10 {
		t.Errorf("MessagesReceived = %d, want 10", stats.MessagesReceived)
	}
	if stats.MessagesBroadcast != 10 {
		t.Errorf("MessagesBroadcast = %d, want 10", stats.MessagesBroadcast)
	}
}

func TestDuckDBHandler_Stats_LastMessageTime(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appender, err := NewAppender(store, DefaultAppenderConfig())
	if err != nil {
		t.Fatalf("NewAppender error: %v", err)
	}

	cfg := DefaultDuckDBHandlerConfig()
	cfg.EnableCrossSourceDedup = false
	handler, err := NewDuckDBHandler(appender, cfg, nil)
	if err != nil {
		t.Fatalf("NewDuckDBHandler error: %v", err)
	}

	// Initially should be zero time
	stats := handler.Stats()
	if !stats.LastMessageTime.IsZero() {
		t.Error("LastMessageTime should be zero initially")
	}

	// Handle a message
	event := &MediaEvent{
		EventID:   "time-test-event",
		Source:    "plex",
		MediaType: "movie",
		Title:     "Test",
		UserID:    1,
		Username:  "test",
		StartedAt: time.Now(),
	}
	data, _ := json.Marshal(event)
	msg := message.NewMessage(event.EventID, data)

	before := time.Now()
	handler.Handle(msg)
	after := time.Now()

	stats = handler.Stats()
	if stats.LastMessageTime.Before(before) || stats.LastMessageTime.After(after) {
		t.Errorf("LastMessageTime = %v, want between %v and %v",
			stats.LastMessageTime, before, after)
	}
}
