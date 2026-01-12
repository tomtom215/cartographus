// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"reflect"
)

// EventBusConfig holds configuration for the CQRS Event Bus.
type EventBusConfig struct {
	GeneratePublishTopic func(eventName string) string
	Marshaler            interface{}
}

// DefaultEventBusConfig returns production defaults for the Event Bus.
func DefaultEventBusConfig() EventBusConfig {
	return EventBusConfig{}
}

// EventBus is a stub for non-NATS builds.
type EventBus struct{}

// Publish is a stub for non-NATS builds.
func (b *EventBus) Publish(_ context.Context, _ interface{}) error {
	return ErrNATSNotEnabled
}

// PublishMediaEvent is a stub for non-NATS builds.
func (b *EventBus) PublishMediaEvent(_ context.Context, _ *MediaEvent) error {
	return ErrNATSNotEnabled
}

// EventProcessorConfig holds configuration for the CQRS Event Processor.
type EventProcessorConfig struct {
	GenerateSubscribeTopic func(eventName string) string
	SubscriberConstructor  interface{}
	Marshaler              interface{}
	Router                 interface{}
	AckOnUnknownEvent      bool
}

// EventProcessor is a stub for non-NATS builds.
type EventProcessor struct{}

// MediaEventHandler is a type-safe handler for MediaEvent events.
type MediaEventHandler interface {
	Handle(ctx context.Context, event *MediaEvent) error
}

// MediaEventHandlerFunc is a function type that implements MediaEventHandler.
type MediaEventHandlerFunc func(ctx context.Context, event *MediaEvent) error

// Handle implements MediaEventHandler.
func (f MediaEventHandlerFunc) Handle(ctx context.Context, event *MediaEvent) error {
	return f(ctx, event)
}

// EventHandlerGroup is a stub for non-NATS builds.
type EventHandlerGroup struct{}

// AddMediaEventFunc is a stub for non-NATS builds.
func (g *EventHandlerGroup) AddMediaEventFunc(_ string, _ MediaEventHandlerFunc) *EventHandlerGroup {
	return g
}

// Name is a stub for non-NATS builds.
func (g *EventHandlerGroup) Name() string {
	return ""
}

// MediaEventMarshaler is a stub for non-NATS builds.
type MediaEventMarshaler struct{}

// EventRegistry tracks registered event types.
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
