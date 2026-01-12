// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package websocket

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

// mockNATSHandler implements NATSMessageHandler for testing.
type mockNATSHandler struct {
	mu       sync.Mutex
	messages chan []byte
	closed   bool
}

func newMockNATSHandler() *mockNATSHandler {
	return &mockNATSHandler{
		messages: make(chan []byte, 100),
	}
}

func (m *mockNATSHandler) Subscribe(_ context.Context, _ string) (<-chan []byte, error) {
	return m.messages, nil
}

func (m *mockNATSHandler) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.messages)
	}
	return nil
}

func (m *mockNATSHandler) Send(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.messages <- data
	}
}

// TestNATSSubscriber_NewNATSSubscriber verifies subscriber creation.
func TestNATSSubscriber_NewNATSSubscriber(t *testing.T) {
	hub := NewHub()
	handler := newMockNATSHandler()

	sub := NewNATSSubscriber(hub, handler)
	if sub == nil {
		t.Fatal("NewNATSSubscriber returned nil")
	}
	if sub.hub != hub {
		t.Error("hub not set correctly")
	}
	if sub.handler != handler {
		t.Error("handler not set correctly")
	}
}

// TestNATSSubscriber_Start verifies subscriber starts correctly.
func TestNATSSubscriber_Start(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	handler := newMockNATSHandler()
	sub := NewNATSSubscriber(hub, handler)

	ctx := context.Background()
	if err := sub.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify running state
	sub.mu.Lock()
	running := sub.running
	sub.mu.Unlock()

	if !running {
		t.Error("subscriber should be running")
	}

	sub.Stop()
	handler.Close()
}

// TestNATSSubscriber_Start_Idempotent verifies multiple Start calls are safe.
func TestNATSSubscriber_Start_Idempotent(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	handler := newMockNATSHandler()
	sub := NewNATSSubscriber(hub, handler)

	ctx := context.Background()

	// Start multiple times should not error
	for i := 0; i < 3; i++ {
		if err := sub.Start(ctx); err != nil {
			t.Errorf("Start() call %d error = %v", i+1, err)
		}
	}

	sub.Stop()
	handler.Close()
}

// TestNATSSubscriber_HandleMessage verifies message processing.
func TestNATSSubscriber_HandleMessage(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create a test client to receive broadcasts
	client := &Client{
		hub:  hub,
		send: make(chan Message, 10),
	}
	hub.Register <- client

	// Wait for registration (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)

	handler := newMockNATSHandler()
	sub := NewNATSSubscriber(hub, handler)

	ctx := context.Background()
	if err := sub.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Send a test event
	event := MediaEvent{
		EventID:   "test-event-123",
		Source:    "plex",
		UserID:    42,
		Username:  "testuser",
		MediaType: "movie",
		Title:     "Test Movie",
		StartedAt: time.Now(),
		Platform:  "Windows",
		Player:    "Plex",
	}
	data, _ := json.Marshal(event)
	handler.Send(data)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check if client received the broadcast
	select {
	case msg := <-client.send:
		if msg.Type != MessageTypePlayback {
			t.Errorf("Message type = %s, want %s", msg.Type, MessageTypePlayback)
		}
	default:
		t.Error("Client did not receive broadcast")
	}

	sub.Stop()
	handler.Close()
}

// TestNATSSubscriber_HandleInvalidMessage verifies invalid message handling.
func TestNATSSubscriber_HandleInvalidMessage(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	handler := newMockNATSHandler()
	sub := NewNATSSubscriber(hub, handler)

	ctx := context.Background()
	if err := sub.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Send invalid JSON
	handler.Send([]byte("not valid json"))

	// Wait for processing - should not panic (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)

	sub.Stop()
	handler.Close()
}

// TestNATSSubscriber_Stop verifies clean shutdown.
func TestNATSSubscriber_Stop(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	handler := newMockNATSHandler()
	sub := NewNATSSubscriber(hub, handler)

	ctx := context.Background()
	if err := sub.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stop should complete without blocking
	done := make(chan struct{})
	go func() {
		sub.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good
	case <-time.After(time.Second):
		t.Error("Stop() blocked for too long")
	}

	// Verify stopped state
	sub.mu.Lock()
	running := sub.running
	sub.mu.Unlock()

	if running {
		t.Error("subscriber should not be running after Stop")
	}

	handler.Close()
}

// TestNATSSubscriber_Stop_Idempotent verifies multiple Stop calls are safe.
func TestNATSSubscriber_Stop_Idempotent(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	handler := newMockNATSHandler()
	sub := NewNATSSubscriber(hub, handler)

	ctx := context.Background()
	if err := sub.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Multiple Stop calls should not panic
	for i := 0; i < 3; i++ {
		sub.Stop()
	}

	handler.Close()
}

// TestNATSSubscriber_MediaEventConversion verifies event conversion.
func TestNATSSubscriber_MediaEventConversion(t *testing.T) {
	sub := &NATSSubscriber{}

	now := time.Now()
	stoppedAt := now.Add(2 * time.Hour)

	event := &MediaEvent{
		EventID:           "test-123",
		Source:            "plex",
		UserID:            42,
		Username:          "testuser",
		MediaType:         "movie",
		Title:             "Test Movie",
		ParentTitle:       "Season 1",
		GrandparentTitle:  "Test Show",
		RatingKey:         "12345",
		StartedAt:         now,
		StoppedAt:         &stoppedAt,
		PercentComplete:   95,
		PlayDuration:      7200,
		PausedCounter:     3,
		Platform:          "Windows",
		Player:            "Plex",
		IPAddress:         "192.168.1.100",
		LocationType:      "lan",
		TranscodeDecision: "direct play",
		VideoResolution:   "1080",
		VideoCodec:        "hevc",
		VideoDynamicRange: "HDR10",
		AudioCodec:        "eac3",
		AudioChannels:     6,
		StreamBitrate:     25000,
		Secure:            true,
		Local:             true,
	}

	playback := sub.mediaEventToPlaybackEvent(event)

	// Verify core fields
	if playback.Source != "plex" {
		t.Errorf("Source = %s, want plex", playback.Source)
	}
	if playback.UserID != 42 {
		t.Errorf("UserID = %d, want 42", playback.UserID)
	}
	if playback.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", playback.Username)
	}
	if playback.MediaType != "movie" {
		t.Errorf("MediaType = %s, want movie", playback.MediaType)
	}
	if playback.Title != "Test Movie" {
		t.Errorf("Title = %s, want Test Movie", playback.Title)
	}

	// Verify optional fields
	if playback.ParentTitle == nil || *playback.ParentTitle != "Season 1" {
		t.Errorf("ParentTitle = %v, want Season 1", playback.ParentTitle)
	}
	if playback.TranscodeDecision == nil || *playback.TranscodeDecision != "direct play" {
		t.Errorf("TranscodeDecision = %v, want direct play", playback.TranscodeDecision)
	}
	if playback.PlayDuration == nil || *playback.PlayDuration != 7200 {
		t.Errorf("PlayDuration = %v, want 7200", playback.PlayDuration)
	}

	// Verify boolean conversion
	if playback.Secure == nil || *playback.Secure != 1 {
		t.Errorf("Secure = %v, want 1", playback.Secure)
	}
	if playback.Local == nil || *playback.Local != 1 {
		t.Errorf("Local = %v, want 1", playback.Local)
	}
}
