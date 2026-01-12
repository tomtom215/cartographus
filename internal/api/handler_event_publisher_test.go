// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goccy/go-json"
	ws "github.com/tomtom215/cartographus/internal/websocket"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// mockWebhookEventPublisher implements EventPublisher for testing webhook NATS integration.
// Thread-safe for concurrent access testing.
type mockWebhookEventPublisher struct {
	publishCalls  atomic.Int32
	mu            sync.Mutex
	lastEvent     *models.PlaybackEvent
	shouldError   bool
	errorToReturn error
}

func (m *mockWebhookEventPublisher) PublishPlaybackEvent(_ context.Context, event *models.PlaybackEvent) error {
	m.publishCalls.Add(1)
	m.mu.Lock()
	m.lastEvent = event
	m.mu.Unlock()
	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return context.DeadlineExceeded
	}
	return nil
}

func (m *mockWebhookEventPublisher) getLastEvent() *models.PlaybackEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastEvent
}

func (m *mockWebhookEventPublisher) getPublishCount() int32 {
	return m.publishCalls.Load()
}

// TestHandler_SetEventPublisher verifies event publisher injection.
func TestHandler_SetEventPublisher(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	pub := &mockWebhookEventPublisher{}

	h.SetEventPublisher(pub)

	if h.eventPublisher == nil {
		t.Error("eventPublisher should be set")
	}
}

// TestHandler_SetEventPublisher_Nil verifies nil publisher is allowed.
func TestHandler_SetEventPublisher_Nil(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	pub := &mockWebhookEventPublisher{}

	// First set a publisher
	h.SetEventPublisher(pub)

	// Then clear it
	h.SetEventPublisher(nil)

	if h.eventPublisher != nil {
		t.Error("eventPublisher should be nil after setting nil")
	}
}

// TestHandler_publishWebhookEvent_NoPublisher verifies no-op when publisher is nil.
func TestHandler_publishWebhookEvent_NoPublisher(t *testing.T) {
	t.Parallel()

	h := &Handler{}

	webhook := &models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    12345,
			Title: "TestUser",
		},
		Player: models.PlexWebhookPlayer{
			PublicAddress: "192.168.1.100",
			Title:         "Test Player",
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:  "movie",
			Title: "Test Movie",
		},
	}

	// Should not panic when publisher is nil
	h.publishWebhookEvent(context.Background(), webhook)
}

// TestHandler_publishWebhookEvent_WithPublisher verifies event publishing.
func TestHandler_publishWebhookEvent_WithPublisher(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	pub := &mockWebhookEventPublisher{}
	h.SetEventPublisher(pub)

	webhook := &models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    42,
			Title: "TestUser",
		},
		Player: models.PlexWebhookPlayer{
			PublicAddress: "10.0.0.1",
			Title:         "Test Player",
			Local:         true,
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:               "movie",
			Title:              "Test Movie",
			LibrarySectionType: "movie",
			RatingKey:          "12345",
		},
	}

	ctx := context.Background()
	h.publishWebhookEvent(ctx, webhook)

	// Wait for async publish with polling (more reliable in CI under load)
	var lastEvent *models.PlaybackEvent
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		lastEvent = pub.getLastEvent()
		if lastEvent != nil {
			break
		}
	}

	if pub.getPublishCount() != 1 {
		t.Errorf("publishCalls = %d, want 1", pub.getPublishCount())
	}

	if lastEvent == nil {
		t.Fatal("lastEvent should not be nil")
	}
	if lastEvent.UserID != 42 {
		t.Errorf("lastEvent.UserID = %d, want 42", lastEvent.UserID)
	}
	if lastEvent.Username != "TestUser" {
		t.Errorf("lastEvent.Username = %s, want TestUser", lastEvent.Username)
	}
	if lastEvent.Title != "Test Movie" {
		t.Errorf("lastEvent.Title = %s, want Test Movie", lastEvent.Title)
	}
	if lastEvent.MediaType != "movie" {
		t.Errorf("lastEvent.MediaType = %s, want movie", lastEvent.MediaType)
	}
	if lastEvent.IPAddress != "10.0.0.1" {
		t.Errorf("lastEvent.IPAddress = %s, want 10.0.0.1", lastEvent.IPAddress)
	}
}

// TestHandler_publishWebhookEvent_ErrorHandling verifies errors don't block.
func TestHandler_publishWebhookEvent_ErrorHandling(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	pub := &mockWebhookEventPublisher{shouldError: true}
	h.SetEventPublisher(pub)

	webhook := &models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    1,
			Title: "User",
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:  "movie",
			Title: "Test",
		},
	}

	ctx := context.Background()

	// Should not block even if publisher errors
	done := make(chan struct{})
	go func() {
		h.publishWebhookEvent(ctx, webhook)
		close(done)
	}()

	select {
	case <-done:
		// Good - publishWebhookEvent returned
	case <-time.After(100 * time.Millisecond):
		t.Error("publishWebhookEvent blocked despite error")
	}

	// Wait for async publish with polling (more reliable in CI under load)
	var count int32
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		count = pub.getPublishCount()
		if count >= 1 {
			break
		}
	}

	if count != 1 {
		t.Errorf("publishCalls = %d, want 1", count)
	}
}

// TestHandler_publishWebhookEvent_ConcurrentAccess verifies thread safety.
func TestHandler_publishWebhookEvent_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	pub := &mockWebhookEventPublisher{}
	h.SetEventPublisher(pub)

	ctx := context.Background()
	const numGoroutines = 100

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			webhook := &models.PlexWebhook{
				Event: "media.play",
				Account: models.PlexWebhookAccount{
					ID:    id,
					Title: "User",
				},
				Metadata: &models.PlexWebhookMetadata{
					Type:  "movie",
					Title: "Test",
				},
			}
			h.publishWebhookEvent(ctx, webhook)
		}(i)
	}
	wg.Wait()

	// Wait for all async publishes with polling (more reliable in CI under load)
	var count int32
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		count = pub.getPublishCount()
		if count >= numGoroutines {
			break
		}
	}

	if count != numGoroutines {
		t.Errorf("publishCalls = %d, want %d", count, numGoroutines)
	}
}

// TestHandler_publishWebhookEvent_NilMetadata verifies handling of nil metadata.
func TestHandler_publishWebhookEvent_NilMetadata(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	pub := &mockWebhookEventPublisher{}
	h.SetEventPublisher(pub)

	webhook := &models.PlexWebhook{
		Event: "admin.database.backup", // No metadata for admin events
		Account: models.PlexWebhookAccount{
			ID:    1,
			Title: "Admin",
		},
		Metadata: nil, // Explicitly nil
	}

	ctx := context.Background()
	h.publishWebhookEvent(ctx, webhook)

	// Wait for async publish
	time.Sleep(50 * time.Millisecond)

	// Should not publish if no meaningful media data
	if pub.getPublishCount() != 0 {
		t.Errorf("publishCalls = %d, want 0 for nil metadata", pub.getPublishCount())
	}
}

// TestHandler_publishWebhookEvent_TVEpisode verifies TV episode event conversion.
func TestHandler_publishWebhookEvent_TVEpisode(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	pub := &mockWebhookEventPublisher{}
	h.SetEventPublisher(pub)

	webhook := &models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    99,
			Title: "TVWatcher",
		},
		Player: models.PlexWebhookPlayer{
			PublicAddress: "203.0.113.5",
			Title:         "Living Room TV",
			Local:         false,
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:             "episode",
			Title:            "Pilot",
			ParentTitle:      "Season 1",
			GrandparentTitle: "Breaking Bad",
			Index:            1,
			ParentIndex:      1,
			RatingKey:        "54321",
		},
	}

	ctx := context.Background()
	h.publishWebhookEvent(ctx, webhook)

	// Wait for async publish with polling (more reliable in CI under load)
	var lastEvent *models.PlaybackEvent
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		lastEvent = pub.getLastEvent()
		if lastEvent != nil {
			break
		}
	}

	if lastEvent == nil {
		t.Fatal("lastEvent should not be nil")
	}
	if lastEvent.MediaType != "episode" {
		t.Errorf("lastEvent.MediaType = %s, want episode", lastEvent.MediaType)
	}
	if lastEvent.Title != "Pilot" {
		t.Errorf("lastEvent.Title = %s, want Pilot", lastEvent.Title)
	}
	if lastEvent.ParentTitle == nil || *lastEvent.ParentTitle != "Season 1" {
		t.Error("lastEvent.ParentTitle should be 'Season 1'")
	}
	if lastEvent.GrandparentTitle == nil || *lastEvent.GrandparentTitle != "Breaking Bad" {
		t.Error("lastEvent.GrandparentTitle should be 'Breaking Bad'")
	}
}

// TestPlexWebhook_WithEventPublisher tests full webhook flow with NATS publishing.
func TestPlexWebhook_WithEventPublisher(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			WebhooksEnabled: true,
			WebhookSecret:   "",
		},
	}

	wsHub := ws.NewHub()
	go wsHub.RunWithContext(context.Background()) //nolint:errcheck // Test only

	pub := &mockWebhookEventPublisher{}

	handler := &Handler{
		cache:          cache.New(5 * time.Minute),
		config:         cfg,
		wsHub:          wsHub,
		startTime:      time.Now(),
		eventPublisher: pub,
	}

	payload := createPlexWebhookPayload("media.play")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Wait for async publish with polling (more reliable in CI under load)
	var lastEvent *models.PlaybackEvent
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		lastEvent = pub.getLastEvent()
		if lastEvent != nil {
			break
		}
	}

	if pub.getPublishCount() != 1 {
		t.Errorf("publishCalls = %d, want 1", pub.getPublishCount())
	}

	if lastEvent == nil {
		t.Fatal("Event should have been published")
	}
	if lastEvent.Source != "plex" {
		t.Errorf("lastEvent.Source = %s, want plex", lastEvent.Source)
	}
}

// TestPlexWebhook_NoPublish_NonMediaEvents tests that non-media events don't publish.
func TestPlexWebhook_NoPublish_NonMediaEvents(t *testing.T) {
	t.Parallel()

	nonMediaEvents := []string{
		"admin.database.backup",
		"admin.database.corrupted",
		"device.new",
	}

	for _, eventType := range nonMediaEvents {
		t.Run(eventType, func(t *testing.T) {
			cfg := &config.Config{
				Plex: config.PlexConfig{
					WebhooksEnabled: true,
					WebhookSecret:   "",
				},
			}

			wsHub := ws.NewHub()
			go wsHub.RunWithContext(context.Background())

			pub := &mockWebhookEventPublisher{}

			handler := &Handler{
				cache:          cache.New(5 * time.Minute),
				config:         cfg,
				wsHub:          wsHub,
				startTime:      time.Now(),
				eventPublisher: pub,
			}

			// Create webhook without metadata (typical for admin events)
			webhook := models.PlexWebhook{
				Event: eventType,
				Account: models.PlexWebhookAccount{
					ID:    12345,
					Title: "Admin",
				},
				Server: models.PlexWebhookServer{
					Title: "TestServer",
					UUID:  "test-uuid",
				},
				Metadata: nil,
			}
			payload, _ := json.Marshal(webhook)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.PlexWebhook(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// Wait for any async operations
			time.Sleep(50 * time.Millisecond)

			if pub.getPublishCount() != 0 {
				t.Errorf("publishCalls = %d, want 0 for %s", pub.getPublishCount(), eventType)
			}
		})
	}
}

// TestPlexWebhook_MediaEvents_AllPublish tests that all media events trigger publish.
func TestPlexWebhook_MediaEvents_AllPublish(t *testing.T) {
	t.Parallel()

	mediaEvents := []string{
		"media.play",
		"media.pause",
		"media.resume",
		"media.stop",
		"media.scrobble",
	}

	for _, eventType := range mediaEvents {
		t.Run(eventType, func(t *testing.T) {
			cfg := &config.Config{
				Plex: config.PlexConfig{
					WebhooksEnabled: true,
					WebhookSecret:   "",
				},
			}

			wsHub := ws.NewHub()
			go wsHub.RunWithContext(context.Background())

			pub := &mockWebhookEventPublisher{}

			handler := &Handler{
				cache:          cache.New(5 * time.Minute),
				config:         cfg,
				wsHub:          wsHub,
				startTime:      time.Now(),
				eventPublisher: pub,
			}

			payload := createPlexWebhookPayload(eventType)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.PlexWebhook(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// Wait for async publish with polling (more reliable in CI under load)
			var count int32
			for i := 0; i < 10; i++ {
				time.Sleep(20 * time.Millisecond)
				count = pub.getPublishCount()
				if count >= 1 {
					break
				}
			}

			if count != 1 {
				t.Errorf("publishCalls = %d, want 1 for %s", count, eventType)
			}
		})
	}
}

// TestHandler_publishWebhookEvent_MachineID verifies MachineID is set from Player.UUID.
// CRITICAL (v1.47): MachineID is essential for cross-source deduplication in multi-device
// shared account scenarios. Without this, different devices would generate the same
// CorrelationKey, causing false positive deduplication.
func TestHandler_publishWebhookEvent_MachineID(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	pub := &mockWebhookEventPublisher{}
	h.SetEventPublisher(pub)

	// Test with Player.UUID set
	webhook := &models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    12345,
			Title: "SharedUser",
		},
		Player: models.PlexWebhookPlayer{
			UUID:          "iphone-abc123", // Device identifier
			PublicAddress: "192.168.1.100",
			Title:         "iPhone",
			Local:         true,
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:      "movie",
			Title:     "Test Movie",
			RatingKey: "54321",
		},
	}

	ctx := context.Background()
	h.publishWebhookEvent(ctx, webhook)

	// Wait for async publish with polling (more reliable in CI under load)
	var lastEvent *models.PlaybackEvent
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		lastEvent = pub.getLastEvent()
		if lastEvent != nil {
			break
		}
	}

	if lastEvent == nil {
		t.Fatal("lastEvent should not be nil")
	}

	// CRITICAL: Verify MachineID is set from Player.UUID
	if lastEvent.MachineID == nil {
		t.Fatal("MachineID should be set from Player.UUID")
	}
	if *lastEvent.MachineID != "iphone-abc123" {
		t.Errorf("MachineID = %s, want iphone-abc123", *lastEvent.MachineID)
	}
}

// TestHandler_publishWebhookEvent_MachineID_Empty verifies empty UUID handling.
func TestHandler_publishWebhookEvent_MachineID_Empty(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	pub := &mockWebhookEventPublisher{}
	h.SetEventPublisher(pub)

	// Test with empty Player.UUID
	webhook := &models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    12345,
			Title: "User",
		},
		Player: models.PlexWebhookPlayer{
			UUID:          "", // Empty UUID
			PublicAddress: "192.168.1.100",
			Title:         "Unknown Device",
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:  "movie",
			Title: "Test Movie",
		},
	}

	ctx := context.Background()
	h.publishWebhookEvent(ctx, webhook)

	// Wait for async publish with polling (more reliable in CI under load)
	var lastEvent *models.PlaybackEvent
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		lastEvent = pub.getLastEvent()
		if lastEvent != nil {
			break
		}
	}

	if lastEvent == nil {
		t.Fatal("lastEvent should not be nil")
	}

	// MachineID should be nil when Player.UUID is empty
	if lastEvent.MachineID != nil {
		t.Errorf("MachineID should be nil for empty UUID, got %s", *lastEvent.MachineID)
	}
}

// TestHandler_publishWebhookEvent_MultiDevice verifies different devices get different MachineIDs.
// This test ensures that shared account multi-device scenarios don't cause false positive deduplication.
func TestHandler_publishWebhookEvent_MultiDevice(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	pub := &mockWebhookEventPublisher{}
	h.SetEventPublisher(pub)

	ctx := context.Background()

	// Device 1: iPhone
	webhook1 := &models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    12345,
			Title: "SharedUser",
		},
		Player: models.PlexWebhookPlayer{
			UUID:          "iphone-abc123",
			PublicAddress: "192.168.1.100",
			Title:         "iPhone",
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:      "movie",
			Title:     "Same Movie",
			RatingKey: "54321",
		},
	}

	h.publishWebhookEvent(ctx, webhook1)

	// Wait for async publish with polling (more reliable in CI under load)
	var event1 *models.PlaybackEvent
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		event1 = pub.getLastEvent()
		if event1 != nil {
			break
		}
	}

	// Device 2: Apple TV
	webhook2 := &models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    12345,
			Title: "SharedUser", // Same user
		},
		Player: models.PlexWebhookPlayer{
			UUID:          "appletv-xyz789", // Different device
			PublicAddress: "192.168.1.101",
			Title:         "Apple TV",
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:      "movie",
			Title:     "Same Movie", // Same content
			RatingKey: "54321",
		},
	}

	h.publishWebhookEvent(ctx, webhook2)

	// Wait for async publish with polling (more reliable in CI under load)
	var event2 *models.PlaybackEvent
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		event2 = pub.getLastEvent()
		// Need to check if it's a different event (different MachineID means event2 arrived)
		if event2 != nil && event2.MachineID != nil && *event2.MachineID == "appletv-xyz789" {
			break
		}
	}

	// Verify both events have MachineID set
	if event1 == nil || event1.MachineID == nil || event2 == nil || event2.MachineID == nil {
		t.Fatal("Both events should have MachineID set")
	}

	// CRITICAL: Verify MachineIDs are different
	if *event1.MachineID == *event2.MachineID {
		t.Errorf("MachineIDs should be different: %s vs %s", *event1.MachineID, *event2.MachineID)
	}

	// Verify session keys use MachineID (Player.UUID)
	if event1.SessionKey == event2.SessionKey {
		t.Errorf("SessionKeys should be different for different devices")
	}
}

// BenchmarkHandler_publishWebhookEvent benchmarks async event publishing.
func BenchmarkHandler_publishWebhookEvent(b *testing.B) {
	h := &Handler{}
	pub := &mockWebhookEventPublisher{}
	h.SetEventPublisher(pub)

	webhook := &models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    1,
			Title: "User",
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:  "movie",
			Title: "Test Movie",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.publishWebhookEvent(ctx, webhook)
	}
}
