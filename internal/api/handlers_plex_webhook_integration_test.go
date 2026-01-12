// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build integration

package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/goccy/go-json"
	ws "github.com/tomtom215/cartographus/internal/websocket"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// mockEventPublisher implements EventPublisher for testing NATS integration.
type mockEventPublisher struct {
	mu             sync.Mutex
	publishedEvent *models.PlaybackEvent
	publishCalled  bool
	publishError   error
}

func (m *mockEventPublisher) PublishPlaybackEvent(_ context.Context, event *models.PlaybackEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishCalled = true
	m.publishedEvent = event
	return m.publishError
}

func (m *mockEventPublisher) getPublishedEvent() *models.PlaybackEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publishedEvent
}

func (m *mockEventPublisher) wasPublishCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publishCalled
}

// TestPlexWebhook_Integration_EventPublishing tests the full webhook → NATS flow.
func TestPlexWebhook_Integration_EventPublishing(t *testing.T) {
	t.Parallel()

	// Create handler with mock event publisher
	cfg := &config.Config{
		Plex: config.PlexConfig{
			WebhooksEnabled: true,
			WebhookSecret:   "",
		},
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	publisher := &mockEventPublisher{}

	handler := &Handler{
		cache:          cache.New(5 * time.Minute),
		config:         cfg,
		wsHub:          wsHub,
		eventPublisher: publisher,
		startTime:      time.Now(),
	}

	// Create webhook payload for media.play event
	webhook := models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    42,
			Title: "TestUser",
		},
		Server: models.PlexWebhookServer{
			Title: "Plex Server",
			UUID:  "server-uuid-123",
		},
		Player: models.PlexWebhookPlayer{
			Title:         "iPhone",
			PublicAddress: "203.0.113.42",
			Local:         false,
			UUID:          "player-uuid-456",
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:                "movie",
			Title:               "Inception",
			LibrarySectionTitle: "Movies",
			LibrarySectionType:  "movie",
		},
	}

	payload, err := json.Marshal(webhook)
	if err != nil {
		t.Fatalf("failed to marshal webhook: %v", err)
	}

	// Send webhook request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Wait for async event publishing
	time.Sleep(100 * time.Millisecond)

	// Verify event was published
	if !publisher.wasPublishCalled() {
		t.Error("expected event to be published to NATS")
	}

	event := publisher.getPublishedEvent()
	if event == nil {
		t.Fatal("expected published event to be non-nil")
	}

	// Verify event fields
	if event.Source != "plex_webhook" {
		t.Errorf("event Source = %q, want %q", event.Source, "plex_webhook")
	}
	if event.Username != "TestUser" {
		t.Errorf("event Username = %q, want %q", event.Username, "TestUser")
	}
	if event.Title != "Inception" {
		t.Errorf("event Title = %q, want %q", event.Title, "Inception")
	}
	if event.MediaType != "movie" {
		t.Errorf("event MediaType = %q, want %q", event.MediaType, "movie")
	}
	if event.IPAddress != "203.0.113.42" {
		t.Errorf("event IPAddress = %q, want %q", event.IPAddress, "203.0.113.42")
	}
	if event.Player != "iPhone" {
		t.Errorf("event Player = %q, want %q", event.Player, "iPhone")
	}
}

// TestPlexWebhook_Integration_MultipleEventTypes tests publishing different event types.
func TestPlexWebhook_Integration_MultipleEventTypes(t *testing.T) {
	t.Parallel()

	eventTypes := []string{
		"media.play",
		"media.pause",
		"media.resume",
		"media.stop",
		"media.scrobble",
	}

	for _, eventType := range eventTypes {
		eventType := eventType // capture for parallel
		t.Run(eventType, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				Plex: config.PlexConfig{
					WebhooksEnabled: true,
					WebhookSecret:   "",
				},
				API: config.APIConfig{
					DefaultPageSize: 100,
					MaxPageSize:     1000,
				},
			}

			wsHub := ws.NewHub()
			go wsHub.Run()

			publisher := &mockEventPublisher{}

			handler := &Handler{
				cache:          cache.New(5 * time.Minute),
				config:         cfg,
				wsHub:          wsHub,
				eventPublisher: publisher,
				startTime:      time.Now(),
			}

			webhook := models.PlexWebhook{
				Event: eventType,
				Account: models.PlexWebhookAccount{
					ID:    1,
					Title: "User",
				},
				Server: models.PlexWebhookServer{
					Title: "Server",
					UUID:  "uuid",
				},
				Player: models.PlexWebhookPlayer{
					Title:         "Player",
					PublicAddress: "10.0.0.1",
					Local:         true,
					UUID:          "player",
				},
				Metadata: &models.PlexWebhookMetadata{
					Type:  "movie",
					Title: "Movie",
				},
			}

			payload, _ := json.Marshal(webhook)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.PlexWebhook(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			// Wait for async publishing
			time.Sleep(100 * time.Millisecond)

			if !publisher.wasPublishCalled() {
				t.Errorf("event type %s not published", eventType)
			}
		})
	}
}

// TestPlexWebhook_Integration_WebSocketBroadcast tests WebSocket broadcasting.
func TestPlexWebhook_Integration_WebSocketBroadcast(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			WebhooksEnabled: true,
			WebhookSecret:   "",
		},
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	// Give hub time to start
	time.Sleep(50 * time.Millisecond)

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		config:    cfg,
		wsHub:     wsHub,
		startTime: time.Now(),
	}

	webhook := models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    1,
			Title: "User",
		},
		Server: models.PlexWebhookServer{
			Title: "Server",
			UUID:  "uuid",
		},
		Player: models.PlexWebhookPlayer{
			Title:         "Player",
			PublicAddress: "10.0.0.1",
			Local:         true,
			UUID:          "player",
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:  "movie",
			Title: "Movie",
		},
	}

	payload, _ := json.Marshal(webhook)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// WebSocket broadcast happens asynchronously
	// In a real integration test, we'd connect a WebSocket client and verify the message
	// For this test, we just verify the handler didn't panic
}

// TestPlexWebhook_Integration_TVShowMetadata tests TV show webhook processing.
func TestPlexWebhook_Integration_TVShowMetadata(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			WebhooksEnabled: true,
			WebhookSecret:   "",
		},
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	publisher := &mockEventPublisher{}

	handler := &Handler{
		cache:          cache.New(5 * time.Minute),
		config:         cfg,
		wsHub:          wsHub,
		eventPublisher: publisher,
		startTime:      time.Now(),
	}

	webhook := models.PlexWebhook{
		Event: "media.play",
		Account: models.PlexWebhookAccount{
			ID:    1,
			Title: "User",
		},
		Server: models.PlexWebhookServer{
			Title: "Server",
			UUID:  "uuid",
		},
		Player: models.PlexWebhookPlayer{
			Title:         "Apple TV",
			PublicAddress: "172.16.0.100",
			Local:         true,
			UUID:          "player",
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:             "episode",
			Title:            "The Pilot",
			ParentTitle:      "Season 1",
			GrandparentTitle: "Breaking Bad",
		},
	}

	payload, _ := json.Marshal(webhook)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Wait for async publishing
	time.Sleep(100 * time.Millisecond)

	event := publisher.getPublishedEvent()
	if event == nil {
		t.Fatal("expected event to be published")
	}

	// Verify TV show fields
	if event.MediaType != "episode" {
		t.Errorf("MediaType = %q, want %q", event.MediaType, "episode")
	}
	if event.Title != "The Pilot" {
		t.Errorf("Title = %q, want %q", event.Title, "The Pilot")
	}
	if event.ParentTitle == nil || *event.ParentTitle != "Season 1" {
		t.Errorf("ParentTitle = %v, want %q", event.ParentTitle, "Season 1")
	}
	if event.GrandparentTitle == nil || *event.GrandparentTitle != "Breaking Bad" {
		t.Errorf("GrandparentTitle = %v, want %q", event.GrandparentTitle, "Breaking Bad")
	}
}

// TestPlexWebhook_Integration_ConcurrentRequests tests handling concurrent webhooks.
func TestPlexWebhook_Integration_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			WebhooksEnabled: true,
			WebhookSecret:   "",
		},
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		config:    cfg,
		wsHub:     wsHub,
		startTime: time.Now(),
	}

	const numRequests = 50
	var wg sync.WaitGroup
	wg.Add(numRequests)

	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer wg.Done()

			webhook := models.PlexWebhook{
				Event: "media.play",
				Account: models.PlexWebhookAccount{
					ID:    id,
					Title: "User",
				},
				Server: models.PlexWebhookServer{
					Title: "Server",
					UUID:  "uuid",
				},
				Player: models.PlexWebhookPlayer{
					Title:         "Player",
					PublicAddress: "10.0.0.1",
					Local:         true,
					UUID:          "player",
				},
				Metadata: &models.PlexWebhookMetadata{
					Type:  "movie",
					Title: "Movie",
				},
			}

			payload, _ := json.Marshal(webhook)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.PlexWebhook(w, req)

			if w.Code != http.StatusOK {
				errors <- nil // Unexpected error handled elsewhere
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Count any errors
	errorCount := 0
	for range errors {
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("%d concurrent requests failed", errorCount)
	}
}
