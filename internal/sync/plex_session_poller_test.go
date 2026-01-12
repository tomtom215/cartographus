// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"testing"
	"time"
)

func TestNewPlexSessionPoller(t *testing.T) {
	t.Parallel()

	config := DefaultSessionPollerConfig()
	poller := NewPlexSessionPoller(nil, config)

	if poller == nil {
		t.Fatal("expected non-nil poller")
	}
	if poller.config.Interval != 30*time.Second {
		t.Errorf("interval = %v, want %v", poller.config.Interval, 30*time.Second)
	}
	if poller.config.PublishAll {
		t.Error("expected PublishAll to be false by default")
	}
}

func TestPlexSessionPoller_StartStop(t *testing.T) {
	t.Parallel()

	config := SessionPollerConfig{
		Interval:       100 * time.Millisecond,
		PublishAll:     false,
		SeenSessionTTL: 1 * time.Second,
	}
	poller := NewPlexSessionPoller(nil, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start poller
	err := poller.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	if !poller.IsRunning() {
		t.Error("expected poller to be running")
	}

	// Starting again should be a no-op
	err = poller.Start(ctx)
	if err != nil {
		t.Errorf("second start should not error: %v", err)
	}

	// Stop poller
	poller.Stop()

	if poller.IsRunning() {
		t.Error("expected poller to be stopped")
	}

	// Stopping again should be a no-op
	poller.Stop()
}

func TestPlexSessionPoller_SeenSessions(t *testing.T) {
	t.Parallel()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour, // Long interval - we don't want automatic polling
		PublishAll:     false,
		SeenSessionTTL: 100 * time.Millisecond,
	}
	poller := NewPlexSessionPoller(nil, config)

	// Initially, session should not be seen
	if poller.hasSeenSession("session-1") {
		t.Error("expected session-1 to not be seen initially")
	}

	// Mark session as seen
	poller.markSessionSeen("session-1")

	if !poller.hasSeenSession("session-1") {
		t.Error("expected session-1 to be seen after marking")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	if poller.hasSeenSession("session-1") {
		t.Error("expected session-1 to expire after TTL")
	}
}

func TestPlexSessionPoller_CleanupSeenSessions(t *testing.T) {
	t.Parallel()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     false,
		SeenSessionTTL: 50 * time.Millisecond,
	}
	poller := NewPlexSessionPoller(nil, config)

	// Add some sessions
	poller.markSessionSeen("session-1")
	poller.markSessionSeen("session-2")
	poller.markSessionSeen("session-3")

	stats := poller.Stats()
	if stats.TrackedSessions != 3 {
		t.Errorf("tracked sessions = %d, want 3", stats.TrackedSessions)
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Cleanup expired sessions
	poller.cleanupSeenSessions()

	stats = poller.Stats()
	if stats.TrackedSessions != 0 {
		t.Errorf("tracked sessions after cleanup = %d, want 0", stats.TrackedSessions)
	}
}

func TestPlexSessionPoller_Stats(t *testing.T) {
	t.Parallel()

	config := SessionPollerConfig{
		Interval:       42 * time.Second,
		PublishAll:     true,
		SeenSessionTTL: 5 * time.Minute,
	}
	poller := NewPlexSessionPoller(nil, config)

	stats := poller.Stats()

	if stats.Running {
		t.Error("expected Running to be false initially")
	}
	if stats.TrackedSessions != 0 {
		t.Errorf("TrackedSessions = %d, want 0", stats.TrackedSessions)
	}
	if stats.PollInterval != 42*time.Second {
		t.Errorf("PollInterval = %v, want %v", stats.PollInterval, 42*time.Second)
	}
	if stats.SeenSessionTTL != 5*time.Minute {
		t.Errorf("SeenSessionTTL = %v, want %v", stats.SeenSessionTTL, 5*time.Minute)
	}
}

func TestDefaultSessionPollerConfig(t *testing.T) {
	t.Parallel()

	config := DefaultSessionPollerConfig()

	if config.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want %v", config.Interval, 30*time.Second)
	}
	if config.PublishAll {
		t.Error("expected PublishAll to be false by default")
	}
	if config.SeenSessionTTL != 5*time.Minute {
		t.Errorf("SeenSessionTTL = %v, want %v", config.SeenSessionTTL, 5*time.Minute)
	}
}

func TestPlexSessionPoller_ContextCancellation(t *testing.T) {
	t.Parallel()

	config := SessionPollerConfig{
		Interval:       50 * time.Millisecond,
		PublishAll:     false,
		SeenSessionTTL: 1 * time.Second,
	}
	poller := NewPlexSessionPoller(nil, config)

	ctx, cancel := context.WithCancel(context.Background())

	// Start poller
	err := poller.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Cancel context
	cancel()

	// Give some time for the poller to notice and stop
	time.Sleep(100 * time.Millisecond)

	// Poller should still be marked as running (internal state)
	// but its goroutine should have exited
	// The Stop() method will reset the running flag
	poller.Stop()

	if poller.IsRunning() {
		t.Error("expected poller to be stopped after context cancellation")
	}
}
