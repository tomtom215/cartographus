// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"fmt"
)

// Subscriber is a stub when NATS dependencies are not available.
// Build with -tags=nats to enable full Watermill subscriber support.
type Subscriber struct {
	// stub - no fields needed
}

// NewSubscriber returns an error when NATS dependencies are not available.
// Build with -tags=nats to enable full Watermill subscriber support.
func NewSubscriber(cfg *SubscriberConfig, logger interface{}) (*Subscriber, error) {
	return nil, fmt.Errorf("NATS subscriber not available: build with -tags=nats")
}

// Subscribe is a stub that returns an error.
func (s *Subscriber) Subscribe(ctx context.Context, topic string) (<-chan interface{}, error) {
	return nil, fmt.Errorf("NATS subscriber not available: build with -tags=nats")
}

// Close is a no-op stub.
func (s *Subscriber) Close() error {
	return nil
}

// MessageHandler is a stub when NATS dependencies are not available.
type MessageHandler struct {
	topic string
}

// NewMessageHandler creates a stub handler.
func (s *Subscriber) NewMessageHandler(topic string) *MessageHandler {
	return &MessageHandler{topic: topic}
}

// Handle is a stub that does nothing.
func (h *MessageHandler) Handle(fn func(ctx context.Context, msg interface{}) error) *MessageHandler {
	return h
}

// Run is a stub that returns an error.
func (h *MessageHandler) Run(ctx context.Context) error {
	return fmt.Errorf("NATS subscriber not available: build with -tags=nats")
}

// EventHandler is a stub when NATS dependencies are not available.
type EventHandler struct{}

// NewEventHandler creates a stub handler.
func (s *Subscriber) NewEventHandler(topic string) *EventHandler {
	return &EventHandler{}
}

// Handle is a stub that does nothing.
func (h *EventHandler) Handle(fn func(ctx context.Context, event *MediaEvent) error) *EventHandler {
	return h
}

// Run is a stub that returns an error.
func (h *EventHandler) Run(ctx context.Context) error {
	return fmt.Errorf("NATS subscriber not available: build with -tags=nats")
}
