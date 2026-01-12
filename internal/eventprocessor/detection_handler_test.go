// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/detection"
)

// mockDetectionProcessor implements DetectionProcessor for testing.
type mockDetectionProcessor struct {
	mu             sync.Mutex
	enabled        bool
	eventsReceived []*detection.DetectionEvent
	alertsToReturn []*detection.Alert
	errorToReturn  error
}

func (m *mockDetectionProcessor) Process(_ context.Context, event *detection.DetectionEvent) ([]*detection.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventsReceived = append(m.eventsReceived, event)
	return m.alertsToReturn, m.errorToReturn
}

func (m *mockDetectionProcessor) Enabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.enabled
}

func (m *mockDetectionProcessor) getEventsReceived() []*detection.DetectionEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*detection.DetectionEvent, len(m.eventsReceived))
	copy(result, m.eventsReceived)
	return result
}

func TestNewDetectionHandler(t *testing.T) {
	t.Parallel()

	t.Run("nil processor", func(t *testing.T) {
		_, err := NewDetectionHandler(nil, nil)
		if err == nil {
			t.Error("expected error for nil processor")
		}
	})

	t.Run("valid processor", func(t *testing.T) {
		proc := &mockDetectionProcessor{enabled: true}
		h, err := NewDetectionHandler(proc, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if h == nil {
			t.Error("expected non-nil handler")
		}
	})
}

func TestDetectionHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("processes valid message", func(t *testing.T) {
		proc := &mockDetectionProcessor{enabled: true}
		h, err := NewDetectionHandler(proc, watermill.NewStdLogger(false, false))
		if err != nil {
			t.Fatalf("failed to create handler: %v", err)
		}

		event := &MediaEvent{
			EventID:   "test-event-1",
			UserID:    42,
			Username:  "testuser",
			MediaType: "movie",
			Title:     "Test Movie",
			Source:    "plex",
			IPAddress: "192.168.1.100",
			Timestamp: time.Now(),
		}

		payload, _ := json.Marshal(event)
		msg := message.NewMessage("msg-1", payload)

		err = h.Handle(msg)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		events := proc.getEventsReceived()
		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		received := events[0]
		if received.EventID != "test-event-1" {
			t.Errorf("event ID = %q, want %q", received.EventID, "test-event-1")
		}
		if received.UserID != 42 {
			t.Errorf("user ID = %d, want %d", received.UserID, 42)
		}
		if received.Username != "testuser" {
			t.Errorf("username = %q, want %q", received.Username, "testuser")
		}
	})

	t.Run("skips when disabled", func(t *testing.T) {
		proc := &mockDetectionProcessor{enabled: false}
		h, _ := NewDetectionHandler(proc, nil)

		event := &MediaEvent{
			EventID:   "test-event-2",
			UserID:    1,
			Username:  "user",
			MediaType: "movie",
			Title:     "Movie",
			Source:    "plex",
		}

		payload, _ := json.Marshal(event)
		msg := message.NewMessage("msg-2", payload)

		err := h.Handle(msg)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		events := proc.getEventsReceived()
		if len(events) != 0 {
			t.Errorf("expected 0 events (disabled), got %d", len(events))
		}

		stats := h.Stats()
		if stats.SkippedDisabled != 1 {
			t.Errorf("skipped disabled = %d, want 1", stats.SkippedDisabled)
		}
	})

	t.Run("handles malformed JSON", func(t *testing.T) {
		proc := &mockDetectionProcessor{enabled: true}
		h, _ := NewDetectionHandler(proc, nil)

		msg := message.NewMessage("msg-3", []byte("not valid json"))

		err := h.Handle(msg)
		if err != nil {
			t.Errorf("expected nil error for malformed JSON, got: %v", err)
		}

		stats := h.Stats()
		if stats.ParseErrors != 1 {
			t.Errorf("parse errors = %d, want 1", stats.ParseErrors)
		}
	})

	t.Run("tracks alerts generated", func(t *testing.T) {
		proc := &mockDetectionProcessor{
			enabled: true,
			alertsToReturn: []*detection.Alert{
				{ID: 1, RuleType: detection.RuleTypeImpossibleTravel},
				{ID: 2, RuleType: detection.RuleTypeConcurrentStreams},
			},
		}
		h, _ := NewDetectionHandler(proc, nil)

		event := &MediaEvent{
			EventID:   "test-event-3",
			UserID:    1,
			Username:  "user",
			MediaType: "movie",
			Title:     "Movie",
			Source:    "plex",
		}

		payload, _ := json.Marshal(event)
		msg := message.NewMessage("msg-4", payload)

		_ = h.Handle(msg)

		stats := h.Stats()
		if stats.AlertsGenerated != 2 {
			t.Errorf("alerts generated = %d, want 2", stats.AlertsGenerated)
		}
	})
}

func TestDetectionHandler_EventConversion(t *testing.T) {
	t.Parallel()

	proc := &mockDetectionProcessor{enabled: true}
	h, _ := NewDetectionHandler(proc, nil)

	event := &MediaEvent{
		EventID:          "conv-test-1",
		SessionKey:       "session-123",
		CorrelationKey:   "corr-456",
		Source:           "plex",
		Timestamp:        time.Now(),
		UserID:           100,
		Username:         "testuser",
		FriendlyName:     "Test User",
		MachineID:        "device-abc",
		Platform:         "iOS",
		Player:           "Plex for iOS",
		Device:           "iPhone",
		MediaType:        "episode",
		Title:            "Episode Title",
		GrandparentTitle: "Show Name",
		IPAddress:        "10.0.0.1",
		LocationType:     "lan",
	}

	payload, _ := json.Marshal(event)
	msg := message.NewMessage("msg-conv", payload)

	_ = h.Handle(msg)

	events := proc.getEventsReceived()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	det := events[0]

	// Verify all fields are mapped correctly
	if det.EventID != "conv-test-1" {
		t.Errorf("EventID = %q, want %q", det.EventID, "conv-test-1")
	}
	if det.SessionKey != "session-123" {
		t.Errorf("SessionKey = %q, want %q", det.SessionKey, "session-123")
	}
	if det.CorrelationKey != "corr-456" {
		t.Errorf("CorrelationKey = %q, want %q", det.CorrelationKey, "corr-456")
	}
	if det.Source != "plex" {
		t.Errorf("Source = %q, want %q", det.Source, "plex")
	}
	if det.UserID != 100 {
		t.Errorf("UserID = %d, want %d", det.UserID, 100)
	}
	if det.Username != "testuser" {
		t.Errorf("Username = %q, want %q", det.Username, "testuser")
	}
	if det.FriendlyName != "Test User" {
		t.Errorf("FriendlyName = %q, want %q", det.FriendlyName, "Test User")
	}
	if det.MachineID != "device-abc" {
		t.Errorf("MachineID = %q, want %q", det.MachineID, "device-abc")
	}
	if det.Platform != "iOS" {
		t.Errorf("Platform = %q, want %q", det.Platform, "iOS")
	}
	if det.Player != "Plex for iOS" {
		t.Errorf("Player = %q, want %q", det.Player, "Plex for iOS")
	}
	if det.Device != "iPhone" {
		t.Errorf("Device = %q, want %q", det.Device, "iPhone")
	}
	if det.MediaType != "episode" {
		t.Errorf("MediaType = %q, want %q", det.MediaType, "episode")
	}
	if det.Title != "Episode Title" {
		t.Errorf("Title = %q, want %q", det.Title, "Episode Title")
	}
	if det.GrandparentTitle != "Show Name" {
		t.Errorf("GrandparentTitle = %q, want %q", det.GrandparentTitle, "Show Name")
	}
	if det.IPAddress != "10.0.0.1" {
		t.Errorf("IPAddress = %q, want %q", det.IPAddress, "10.0.0.1")
	}
	if det.LocationType != "lan" {
		t.Errorf("LocationType = %q, want %q", det.LocationType, "lan")
	}
	if det.EventType != EventTypePlaybackStart {
		t.Errorf("EventType = %q, want %q", det.EventType, EventTypePlaybackStart)
	}
}

func TestDetectionHandler_StopEventType(t *testing.T) {
	t.Parallel()

	proc := &mockDetectionProcessor{enabled: true}
	h, _ := NewDetectionHandler(proc, nil)

	now := time.Now()
	stoppedAt := now.Add(time.Hour)

	event := &MediaEvent{
		EventID:   "stop-test-1",
		UserID:    1,
		Username:  "user",
		MediaType: "movie",
		Title:     "Movie",
		Source:    "plex",
		StartedAt: now,
		StoppedAt: &stoppedAt,
	}

	payload, _ := json.Marshal(event)
	msg := message.NewMessage("msg-stop", payload)

	_ = h.Handle(msg)

	events := proc.getEventsReceived()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].EventType != EventTypePlaybackStop {
		t.Errorf("EventType = %q, want %q", events[0].EventType, EventTypePlaybackStop)
	}
}

func TestDetectionHandler_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	proc := &mockDetectionProcessor{enabled: true}
	h, _ := NewDetectionHandler(proc, nil)

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			event := &MediaEvent{
				EventID:   "concurrent-" + string(rune(id)),
				UserID:    id,
				Username:  "user",
				MediaType: "movie",
				Title:     "Movie",
				Source:    "plex",
			}

			payload, _ := json.Marshal(event)
			msg := message.NewMessage("msg-"+string(rune(id)), payload)

			_ = h.Handle(msg)
		}(i)
	}

	wg.Wait()

	stats := h.Stats()
	if stats.MessagesReceived != numGoroutines {
		t.Errorf("messages received = %d, want %d", stats.MessagesReceived, numGoroutines)
	}
}
