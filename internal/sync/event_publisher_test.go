// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
)

// mockEventPublisher implements EventPublisher for testing.
type mockEventPublisher struct {
	publishCalls atomic.Int32
	mu           sync.Mutex
	lastEvent    *models.PlaybackEvent
	shouldError  bool
}

func (m *mockEventPublisher) PublishPlaybackEvent(_ context.Context, event *models.PlaybackEvent) error {
	m.publishCalls.Add(1)
	m.mu.Lock()
	m.lastEvent = event
	m.mu.Unlock()
	if m.shouldError {
		return context.DeadlineExceeded
	}
	return nil
}

func (m *mockEventPublisher) getLastEvent() *models.PlaybackEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastEvent
}

// TestManager_SetEventPublisher verifies event publisher injection.
func TestManager_SetEventPublisher(t *testing.T) {
	m := &Manager{}
	pub := &mockEventPublisher{}

	m.SetEventPublisher(pub)

	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.eventPublisher == nil {
		t.Error("eventPublisher should be set")
	}
}

// TestManager_SetEventPublisher_Nil verifies nil publisher is allowed.
func TestManager_SetEventPublisher_Nil(t *testing.T) {
	m := &Manager{}
	pub := &mockEventPublisher{}

	// First set a publisher
	m.SetEventPublisher(pub)

	// Then clear it
	m.SetEventPublisher(nil)

	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.eventPublisher != nil {
		t.Error("eventPublisher should be nil after setting nil")
	}
}

// TestManager_publishEvent_NoPublisher verifies no-op when publisher is nil.
func TestManager_publishEvent_NoPublisher(t *testing.T) {
	m := &Manager{}

	event := &models.PlaybackEvent{
		ID:        uuid.New(),
		UserID:    1,
		MediaType: "movie",
		Title:     "Test",
	}

	// Should not panic when publisher is nil
	m.publishEvent(context.Background(), event)
}

// TestManager_publishEvent_WithPublisher verifies event publishing.
func TestManager_publishEvent_WithPublisher(t *testing.T) {
	m := &Manager{}
	pub := &mockEventPublisher{}
	m.SetEventPublisher(pub)

	event := &models.PlaybackEvent{
		ID:        uuid.New(),
		UserID:    42,
		MediaType: "movie",
		Title:     "Test Movie",
	}

	ctx := context.Background()
	m.publishEvent(ctx, event)

	// Wait for async publish
	time.Sleep(50 * time.Millisecond)

	if pub.publishCalls.Load() != 1 {
		t.Errorf("publishCalls = %d, want 1", pub.publishCalls.Load())
	}
	lastEvent := pub.getLastEvent()
	if lastEvent == nil {
		t.Fatal("lastEvent should not be nil")
	}
	if lastEvent.UserID != 42 {
		t.Errorf("lastEvent.UserID = %d, want 42", lastEvent.UserID)
	}
}

// TestManager_publishEvent_ErrorHandling verifies errors don't block.
func TestManager_publishEvent_ErrorHandling(t *testing.T) {
	m := &Manager{}
	pub := &mockEventPublisher{shouldError: true}
	m.SetEventPublisher(pub)

	event := &models.PlaybackEvent{
		ID:        uuid.New(),
		UserID:    1,
		MediaType: "movie",
		Title:     "Test",
	}

	ctx := context.Background()

	// Should not block even if publisher errors
	done := make(chan struct{})
	go func() {
		m.publishEvent(ctx, event)
		close(done)
	}()

	select {
	case <-done:
		// Good - publishEvent returned
	case <-time.After(100 * time.Millisecond):
		t.Error("publishEvent blocked despite error")
	}

	// Wait for async publish to complete
	time.Sleep(50 * time.Millisecond)

	if pub.publishCalls.Load() != 1 {
		t.Errorf("publishCalls = %d, want 1", pub.publishCalls.Load())
	}
}

// TestManager_publishEvent_ConcurrentAccess verifies thread safety.
func TestManager_publishEvent_ConcurrentAccess(t *testing.T) {
	m := &Manager{}
	pub := &mockEventPublisher{}
	m.SetEventPublisher(pub)

	ctx := context.Background()
	const numGoroutines = 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			event := &models.PlaybackEvent{
				ID:        uuid.New(),
				UserID:    id,
				MediaType: "movie",
				Title:     "Test",
			}
			m.publishEvent(ctx, event)
		}(i)
	}

	// Wait for all async publishes
	time.Sleep(200 * time.Millisecond)

	if pub.publishCalls.Load() != numGoroutines {
		t.Errorf("publishCalls = %d, want %d", pub.publishCalls.Load(), numGoroutines)
	}
}
