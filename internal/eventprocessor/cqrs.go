// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// EventBusConfig holds configuration for the CQRS Event Bus.
type EventBusConfig struct {
	// GeneratePublishTopic returns the topic name for an event type.
	// Default: "playback.<source>.<media_type>" based on MediaEvent fields.
	GeneratePublishTopic func(params cqrs.GenerateEventPublishTopicParams) (string, error)

	// Marshaler handles event serialization.
	// Default: JSON marshaler.
	Marshaler cqrs.CommandEventMarshaler
}

// DefaultEventBusConfig returns production defaults for the Event Bus.
func DefaultEventBusConfig() EventBusConfig {
	return EventBusConfig{
		GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
			// Default to playback topic for media events
			return "playback." + params.EventName, nil
		},
		Marshaler: cqrs.JSONMarshaler{
			GenerateName: cqrs.StructName,
		},
	}
}

// EventBus provides type-safe event publishing using Watermill's CQRS component.
// It automatically marshals Go structs to messages and routes them to appropriate topics.
//
// Benefits over raw Publisher:
//   - Type-safe event publishing (compile-time checks)
//   - Automatic JSON marshaling
//   - Event name generation from struct type
//   - Consistent topic routing
type EventBus struct {
	bus    *cqrs.EventBus
	config EventBusConfig
	logger watermill.LoggerAdapter
}

// NewEventBus creates a new CQRS Event Bus.
func NewEventBus(publisher message.Publisher, cfg EventBusConfig, logger watermill.LoggerAdapter) (*EventBus, error) {
	if publisher == nil {
		return nil, fmt.Errorf("publisher required")
	}
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}
	if cfg.Marshaler == nil {
		cfg.Marshaler = cqrs.JSONMarshaler{GenerateName: cqrs.StructName}
	}

	eventBus, err := cqrs.NewEventBusWithConfig(publisher, cqrs.EventBusConfig{
		GeneratePublishTopic: cfg.GeneratePublishTopic,
		Marshaler:            cfg.Marshaler,
		Logger:               logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create event bus: %w", err)
	}

	return &EventBus{
		bus:    eventBus,
		config: cfg,
		logger: logger,
	}, nil
}

// Publish sends an event to the appropriate topic.
// The event type determines the topic via GeneratePublishTopic.
func (b *EventBus) Publish(ctx context.Context, event interface{}) error {
	return b.bus.Publish(ctx, event)
}

// PublishMediaEvent publishes a MediaEvent with automatic topic routing.
// Topic is determined by event.Topic() method.
func (b *EventBus) PublishMediaEvent(ctx context.Context, event *MediaEvent) error {
	return b.bus.Publish(ctx, event)
}

// EventProcessorConfig holds configuration for the CQRS Event Processor.
type EventProcessorConfig struct {
	// GenerateSubscribeTopic returns the topic to subscribe to for an event type.
	GenerateSubscribeTopic func(eventName string) string

	// SubscriberConstructor creates subscribers for each handler.
	SubscriberConstructor cqrs.EventsSubscriberConstructor

	// Marshaler handles event deserialization.
	Marshaler cqrs.CommandEventMarshaler

	// Router is the Watermill router for handler registration.
	Router *message.Router

	// AckOnUnknownEvent determines behavior for unknown event types.
	AckOnUnknownEvent bool
}

// EventProcessor provides type-safe event handling using Watermill's CQRS component.
// It automatically deserializes messages to the appropriate Go struct type.
//
// Benefits over raw Subscriber:
//   - Type-safe event handlers (handler receives concrete type, not raw message)
//   - Automatic JSON unmarshaling
//   - Multiple handlers per event type
//   - Clean handler registration
type EventProcessor struct {
	config   EventProcessorConfig
	logger   watermill.LoggerAdapter
	handlers []cqrs.EventHandler
}

// NewEventProcessor creates a new CQRS Event Processor.
// The processor must have AddHandlers called before running.
func NewEventProcessor(
	router *message.Router,
	subscriberConstructor cqrs.EventsSubscriberConstructor,
	cfg EventProcessorConfig,
	logger watermill.LoggerAdapter,
) (*EventProcessor, error) {
	if router == nil {
		return nil, fmt.Errorf("router required")
	}
	if subscriberConstructor == nil {
		return nil, fmt.Errorf("subscriber constructor required")
	}
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}
	if cfg.Marshaler == nil {
		cfg.Marshaler = cqrs.JSONMarshaler{GenerateName: cqrs.StructName}
	}
	if cfg.GenerateSubscribeTopic == nil {
		cfg.GenerateSubscribeTopic = func(eventName string) string {
			return "playback." + eventName
		}
	}

	return &EventProcessor{
		config:   cfg,
		logger:   logger,
		handlers: make([]cqrs.EventHandler, 0),
	}, nil
}

// MediaEventHandler is a type-safe handler for MediaEvent events.
// Implement this interface for custom MediaEvent processing.
type MediaEventHandler interface {
	// Handle processes a MediaEvent.
	// Return error to trigger retry/DLQ, nil to acknowledge.
	Handle(ctx context.Context, event *MediaEvent) error
}

// MediaEventHandlerFunc is a function type that implements MediaEventHandler.
type MediaEventHandlerFunc func(ctx context.Context, event *MediaEvent) error

// Handle implements MediaEventHandler.
func (f MediaEventHandlerFunc) Handle(ctx context.Context, event *MediaEvent) error {
	return f(ctx, event)
}

// NewMediaEventHandler creates a cqrs.EventHandler for MediaEvent.
// This bridges the type-safe handler to Watermill's CQRS interface.
func NewMediaEventHandler(handlerName string, handler MediaEventHandler) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		handlerName,
		func(ctx context.Context, event *MediaEvent) error {
			return handler.Handle(ctx, event)
		},
	)
}

// GenericEventHandler wraps any typed handler function into a cqrs.EventHandler.
// T should be the struct type (not a pointer). The handler receives *T.
func GenericEventHandler[T any](handlerName string, handler func(ctx context.Context, event *T) error) cqrs.EventHandler {
	return cqrs.NewEventHandler(handlerName, handler)
}

// EventHandlerGroup manages multiple handlers for different event types.
// Use this to register handlers that share a subscriber.
type EventHandlerGroup struct {
	name       string
	subscriber message.Subscriber
	handlers   []cqrs.EventHandler
	marshaler  cqrs.CommandEventMarshaler
	logger     watermill.LoggerAdapter
}

// NewEventHandlerGroup creates a new handler group.
func NewEventHandlerGroup(name string, subscriber message.Subscriber, logger watermill.LoggerAdapter) *EventHandlerGroup {
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}
	return &EventHandlerGroup{
		name:       name,
		subscriber: subscriber,
		handlers:   make([]cqrs.EventHandler, 0),
		marshaler:  cqrs.JSONMarshaler{GenerateName: cqrs.StructName},
		logger:     logger,
	}
}

// AddHandler adds a handler to the group.
func (g *EventHandlerGroup) AddHandler(handler cqrs.EventHandler) *EventHandlerGroup {
	g.handlers = append(g.handlers, handler)
	return g
}

// AddMediaEventHandler adds a typed MediaEvent handler.
func (g *EventHandlerGroup) AddMediaEventHandler(name string, handler MediaEventHandler) *EventHandlerGroup {
	g.handlers = append(g.handlers, NewMediaEventHandler(name, handler))
	return g
}

// AddMediaEventFunc adds a MediaEvent handler function.
func (g *EventHandlerGroup) AddMediaEventFunc(name string, fn MediaEventHandlerFunc) *EventHandlerGroup {
	return g.AddMediaEventHandler(name, fn)
}

// Handlers returns all registered handlers.
func (g *EventHandlerGroup) Handlers() []cqrs.EventHandler {
	return g.handlers
}

// Subscriber returns the group's subscriber.
func (g *EventHandlerGroup) Subscriber() message.Subscriber {
	return g.subscriber
}

// Name returns the group name.
func (g *EventHandlerGroup) Name() string {
	return g.name
}

// MediaEventMarshaler provides custom marshaling for MediaEvent.
// It uses the event's Topic() method for routing.
type MediaEventMarshaler struct {
	cqrs.JSONMarshaler
}

// Name returns the event name for topic routing.
func (m MediaEventMarshaler) Name(event interface{}) string {
	if me, ok := event.(*MediaEvent); ok {
		// Use Source.MediaType as the event name for topic routing
		return me.Source + "." + me.MediaType
	}
	// Fallback to struct name
	return reflect.TypeOf(event).Elem().Name()
}

// NewMediaEventMarshaler creates a marshaler optimized for MediaEvent.
func NewMediaEventMarshaler() MediaEventMarshaler {
	return MediaEventMarshaler{
		JSONMarshaler: cqrs.JSONMarshaler{
			GenerateName: cqrs.StructName,
		},
	}
}

// EventRegistry tracks registered event types for documentation and validation.
type EventRegistry struct {
	eventTypes map[string]reflect.Type
}

// NewEventRegistry creates a new event registry.
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		eventTypes: make(map[string]reflect.Type),
	}
}

// Register adds an event type to the registry.
func (r *EventRegistry) Register(name string, eventType interface{}) {
	r.eventTypes[name] = reflect.TypeOf(eventType)
}

// RegisterMediaEvent registers MediaEvent with standard naming.
func (r *EventRegistry) RegisterMediaEvent() {
	r.eventTypes["MediaEvent"] = reflect.TypeOf(MediaEvent{})
}

// Get returns the type for an event name.
func (r *EventRegistry) Get(name string) (reflect.Type, bool) {
	t, ok := r.eventTypes[name]
	return t, ok
}

// Names returns all registered event names.
func (r *EventRegistry) Names() []string {
	names := make([]string, 0, len(r.eventTypes))
	for name := range r.eventTypes {
		names = append(names, name)
	}
	return names
}
