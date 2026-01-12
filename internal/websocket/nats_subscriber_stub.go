// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package websocket

import (
	"context"
	"fmt"
)

// NATSMessageHandler defines the interface for receiving NATS messages.
// This is a stub for non-NATS builds.
type NATSMessageHandler interface {
	Subscribe(ctx context.Context, topic string) (<-chan []byte, error)
	Close() error
}

// NATSSubscriber is a stub for non-NATS builds.
type NATSSubscriber struct{}

// NewNATSSubscriber returns nil in non-NATS builds.
func NewNATSSubscriber(_ *Hub, _ NATSMessageHandler) *NATSSubscriber {
	return nil
}

// Start returns an error in non-NATS builds.
func (s *NATSSubscriber) Start(_ context.Context) error {
	return fmt.Errorf("NATS support not enabled (build with -tags nats)")
}

// Stop is a no-op stub.
func (s *NATSSubscriber) Stop() {}
