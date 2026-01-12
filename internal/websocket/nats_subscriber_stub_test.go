// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package websocket

import (
	"context"
	"testing"
)

// Tests for NATS subscriber stub (non-NATS builds)

// TestNATSSubscriberStub_NewNATSSubscriber tests that NewNATSSubscriber returns nil
func TestNATSSubscriberStub_NewNATSSubscriber(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	sub := NewNATSSubscriber(hub, nil)
	if sub != nil {
		t.Error("NewNATSSubscriber() should return nil in non-NATS build")
	}
}

// TestNATSSubscriberStub_Start tests that Start returns error
func TestNATSSubscriberStub_Start(t *testing.T) {
	t.Parallel()

	sub := &NATSSubscriber{}
	err := sub.Start(context.Background())
	if err == nil {
		t.Error("Start() should return error in non-NATS build")
	}
}

// TestNATSSubscriberStub_Stop tests that Stop is a no-op
func TestNATSSubscriberStub_Stop(t *testing.T) {
	t.Parallel()

	sub := &NATSSubscriber{}
	// Should not panic
	sub.Stop()
}
