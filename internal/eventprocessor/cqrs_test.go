// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

func TestDefaultEventBusConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultEventBusConfig()

	if cfg.GeneratePublishTopic == nil {
		t.Error("GeneratePublishTopic should not be nil")
	}

	// Test default topic generation
	topic, err := cfg.GeneratePublishTopic(cqrs.GenerateEventPublishTopicParams{
		EventName: "plex.movie",
	})
	if err != nil {
		t.Errorf("GeneratePublishTopic error: %v", err)
	}
	if topic != "playback.plex.movie" {
		t.Errorf("GeneratePublishTopic = %q, want %q", topic, "playback.plex.movie")
	}
}

func TestMediaEventMarshaler_Name(t *testing.T) {
	t.Parallel()

	marshaler := NewMediaEventMarshaler()

	event := &MediaEvent{
		Source:    "plex",
		MediaType: "movie",
	}

	name := marshaler.Name(event)
	if name != "plex.movie" {
		t.Errorf("Name = %q, want %q", name, "plex.movie")
	}
}

func TestMediaEventMarshaler_Name_Episode(t *testing.T) {
	t.Parallel()

	marshaler := NewMediaEventMarshaler()

	event := &MediaEvent{
		Source:    "tautulli",
		MediaType: "episode",
	}

	name := marshaler.Name(event)
	if name != "tautulli.episode" {
		t.Errorf("Name = %q, want %q", name, "tautulli.episode")
	}
}

func TestMediaEventHandlerFunc(t *testing.T) {
	t.Parallel()

	var called bool
	var receivedEvent *MediaEvent

	handler := MediaEventHandlerFunc(func(ctx context.Context, event *MediaEvent) error {
		called = true
		receivedEvent = event
		return nil
	})

	event := &MediaEvent{
		EventID:   "test-event",
		Source:    "plex",
		MediaType: "movie",
		Title:     "Test Movie",
	}

	err := handler.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("Handle error: %v", err)
	}
	if !called {
		t.Error("Handler was not called")
	}
	if receivedEvent != event {
		t.Error("Handler received wrong event")
	}
}

func TestNewMediaEventHandler(t *testing.T) {
	t.Parallel()

	var handleCount int
	handler := NewMediaEventHandler("test-handler", MediaEventHandlerFunc(func(ctx context.Context, event *MediaEvent) error {
		handleCount++
		return nil
	}))

	// Verify it creates a valid cqrs.EventHandler
	if handler == nil {
		t.Fatal("NewMediaEventHandler returned nil")
	}
}

func TestEventHandlerGroup(t *testing.T) {
	t.Parallel()

	logger := watermill.NewStdLogger(false, false)
	group := NewEventHandlerGroup("test-group", nil, logger)

	if group.Name() != "test-group" {
		t.Errorf("Name = %q, want %q", group.Name(), "test-group")
	}

	// Add handlers
	group.AddMediaEventFunc("handler1", func(ctx context.Context, event *MediaEvent) error {
		return nil
	})
	group.AddMediaEventFunc("handler2", func(ctx context.Context, event *MediaEvent) error {
		return nil
	})

	handlers := group.Handlers()
	if len(handlers) != 2 {
		t.Errorf("Handlers count = %d, want 2", len(handlers))
	}
}

func TestEventHandlerGroup_ChainedAddition(t *testing.T) {
	t.Parallel()

	group := NewEventHandlerGroup("chained", nil, nil)

	// Chained addition should work
	result := group.
		AddMediaEventFunc("h1", func(ctx context.Context, event *MediaEvent) error { return nil }).
		AddMediaEventFunc("h2", func(ctx context.Context, event *MediaEvent) error { return nil }).
		AddMediaEventFunc("h3", func(ctx context.Context, event *MediaEvent) error { return nil })

	if result != group {
		t.Error("Chained methods should return same group")
	}
	if len(group.Handlers()) != 3 {
		t.Errorf("Handlers count = %d, want 3", len(group.Handlers()))
	}
}

func TestEventRegistry(t *testing.T) {
	t.Parallel()

	registry := NewEventRegistry()

	// Register MediaEvent
	registry.RegisterMediaEvent()

	// Verify registration
	typ, ok := registry.Get("MediaEvent")
	if !ok {
		t.Error("MediaEvent not found in registry")
	}
	if typ != reflect.TypeOf(MediaEvent{}) {
		t.Error("MediaEvent type mismatch")
	}

	// Test custom registration
	type CustomEvent struct {
		ID   string
		Name string
	}
	registry.Register("CustomEvent", CustomEvent{})

	_, ok = registry.Get("CustomEvent")
	if !ok {
		t.Error("CustomEvent not found in registry")
	}

	// Test names
	names := registry.Names()
	if len(names) != 2 {
		t.Errorf("Names count = %d, want 2", len(names))
	}
}

func TestEventRegistry_NotFound(t *testing.T) {
	t.Parallel()

	registry := NewEventRegistry()

	_, ok := registry.Get("NonExistent")
	if ok {
		t.Error("Should not find non-existent event")
	}
}

// MockPublisher implements message.Publisher for testing EventBus.
type MockPublisher struct{}

func (p *MockPublisher) Publish(_ string, _ ...*struct {
	UUID     string
	Metadata map[string]string
	Payload  []byte
}) error {
	return nil
}

func (p *MockPublisher) Close() error {
	return nil
}

func TestGenericEventHandler(t *testing.T) {
	t.Parallel()

	// Test with MediaEvent - T should be the struct type, handler receives *T
	handler := GenericEventHandler[MediaEvent]("media-handler", func(ctx context.Context, event *MediaEvent) error {
		if event.Title != "Test" {
			return ErrNATSNotEnabled
		}
		return nil
	})

	if handler == nil {
		t.Fatal("GenericEventHandler returned nil")
	}
}

// TestEventBusConfig_CustomTopicGenerator verifies custom topic generation.
func TestEventBusConfig_CustomTopicGenerator(t *testing.T) {
	t.Parallel()

	cfg := EventBusConfig{
		GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
			return "custom." + params.EventName, nil
		},
	}

	topic, err := cfg.GeneratePublishTopic(cqrs.GenerateEventPublishTopicParams{
		EventName: "test",
	})
	if err != nil {
		t.Errorf("GeneratePublishTopic error: %v", err)
	}
	if topic != "custom.test" {
		t.Errorf("Custom topic = %q, want %q", topic, "custom.test")
	}
}

// TestCQRSJSONMarshaler verifies JSON marshaling works with MediaEvent.
func TestCQRSJSONMarshaler(t *testing.T) {
	t.Parallel()

	marshaler := cqrs.JSONMarshaler{
		GenerateName: cqrs.StructName,
	}

	event := &MediaEvent{
		EventID:   "marshal-test",
		Source:    "plex",
		MediaType: "movie",
		Title:     "Test Movie",
		UserID:    1,
		Username:  "testuser",
		StartedAt: time.Now().UTC(),
	}

	// Marshal
	msg, err := marshaler.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if msg == nil {
		t.Fatal("Marshal returned nil message")
	}

	// Unmarshal
	target := &MediaEvent{}
	err = marshaler.Unmarshal(msg, target)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if target.EventID != event.EventID {
		t.Errorf("EventID = %q, want %q", target.EventID, event.EventID)
	}
	if target.Source != event.Source {
		t.Errorf("Source = %q, want %q", target.Source, event.Source)
	}
	if target.Title != event.Title {
		t.Errorf("Title = %q, want %q", target.Title, event.Title)
	}
}
