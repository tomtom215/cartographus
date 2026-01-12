// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

func TestNewJellyfinSessionPoller(t *testing.T) {
	t.Parallel()

	config := DefaultSessionPollerConfig()
	client := NewJellyfinClient("http://localhost:8096", "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

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

func TestJellyfinSessionPoller_StartStop(t *testing.T) {
	t.Parallel()

	config := SessionPollerConfig{
		Interval:       100 * time.Millisecond,
		PublishAll:     false,
		SeenSessionTTL: 1 * time.Second,
	}

	// Create a mock server that returns empty sessions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start poller
	err := poller.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Starting again should be a no-op
	err = poller.Start(ctx)
	if err != nil {
		t.Errorf("second start should not error: %v", err)
	}

	// Let it run for a bit
	time.Sleep(150 * time.Millisecond)

	// Stop poller
	poller.Stop()

	// Stopping again should be a no-op
	poller.Stop()
}

func TestJellyfinSessionPoller_SeenSessions(t *testing.T) {
	t.Parallel()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour, // Long interval - we don't want automatic polling
		PublishAll:     false,
		SeenSessionTTL: 100 * time.Millisecond,
	}
	client := NewJellyfinClient("http://localhost:8096", "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

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

func TestJellyfinSessionPoller_CleanupSeenSessions(t *testing.T) {
	t.Parallel()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     false,
		SeenSessionTTL: 50 * time.Millisecond,
	}
	client := NewJellyfinClient("http://localhost:8096", "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	// Add some sessions
	poller.markSessionSeen("session-1")
	poller.markSessionSeen("session-2")
	poller.markSessionSeen("session-3")

	// Verify all three are tracked
	// LRUCache is thread-safe, no external lock needed
	trackedCount := poller.seenSessions.Len()

	if trackedCount != 3 {
		t.Errorf("tracked sessions = %d, want 3", trackedCount)
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Cleanup expired sessions
	poller.cleanupSeenSessions()

	// LRUCache is thread-safe, no external lock needed
	trackedCount = poller.seenSessions.Len()

	if trackedCount != 0 {
		t.Errorf("tracked sessions after cleanup = %d, want 0", trackedCount)
	}
}

func TestJellyfinSessionPoller_ContextCancellation(t *testing.T) {
	t.Parallel()

	config := SessionPollerConfig{
		Interval:       50 * time.Millisecond,
		PublishAll:     false,
		SeenSessionTTL: 1 * time.Second,
	}

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

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

	// The Stop() method will ensure clean shutdown
	poller.Stop()
}

func TestJellyfinSessionPoller_SetOnSession(t *testing.T) {
	t.Parallel()

	config := DefaultSessionPollerConfig()
	client := NewJellyfinClient("http://localhost:8096", "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	// Initially no callback
	poller.mu.RLock()
	hasCallback := poller.onSession != nil
	poller.mu.RUnlock()

	if hasCallback {
		t.Error("expected onSession to be nil initially")
	}

	// Set callback
	called := false
	poller.SetOnSession(func(_ *models.JellyfinSession) {
		called = true
	})

	poller.mu.RLock()
	hasCallback = poller.onSession != nil
	poller.mu.RUnlock()

	if !hasCallback {
		t.Error("expected onSession to be set")
	}

	// Test calling it
	poller.mu.RLock()
	cb := poller.onSession
	poller.mu.RUnlock()
	cb(&models.JellyfinSession{})

	if !called {
		t.Error("callback was not invoked")
	}
}

func TestJellyfinSessionPoller_Poll(t *testing.T) {
	t.Parallel()

	var callCount int32

	// Create a mock server that returns active sessions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{
				"Id": "session-123",
				"UserName": "TestUser",
				"NowPlayingItem": {
					"Id": "item-456",
					"Name": "Test Movie",
					"Type": "Movie"
				}
			},
			{
				"Id": "session-456",
				"UserName": "AnotherUser",
				"NowPlayingItem": {
					"Id": "item-789",
					"Name": "Another Movie",
					"Type": "Movie"
				}
			}
		]`))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       100 * time.Millisecond,
		PublishAll:     false,
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// Call poll directly
	poller.poll(ctx)

	// Should have received both sessions
	if count := atomic.LoadInt32(&callCount); count != 2 {
		t.Errorf("callback called %d times, want 2", count)
	}

	// Poll again - should not call callback again (sessions are seen)
	poller.poll(ctx)

	if count := atomic.LoadInt32(&callCount); count != 2 {
		t.Errorf("callback called %d times, want 2 (no duplicates)", count)
	}
}

func TestJellyfinSessionPoller_PollWithPublishAll(t *testing.T) {
	t.Parallel()

	var callCount int32

	// Create a mock server that returns active sessions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{
				"Id": "session-123",
				"UserName": "TestUser",
				"NowPlayingItem": {
					"Id": "item-456",
					"Name": "Test Movie",
					"Type": "Movie"
				}
			}
		]`))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       100 * time.Millisecond,
		PublishAll:     true, // Publish all sessions every time
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// Call poll multiple times
	poller.poll(ctx)
	poller.poll(ctx)
	poller.poll(ctx)

	// With PublishAll=true, should be called 3 times
	if count := atomic.LoadInt32(&callCount); count != 3 {
		t.Errorf("callback called %d times, want 3 (PublishAll enabled)", count)
	}
}

func TestJellyfinSessionPoller_PollError(t *testing.T) {
	t.Parallel()

	var callCount int32

	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       100 * time.Millisecond,
		PublishAll:     false,
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// Call poll - should handle error gracefully
	poller.poll(ctx)

	// Callback should not be called
	if count := atomic.LoadInt32(&callCount); count != 0 {
		t.Errorf("callback called %d times on error, want 0", count)
	}
}

func TestJellyfinSessionPoller_PollNoCallback(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns active sessions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{
				"Id": "session-123",
				"NowPlayingItem": {
					"Id": "item-456",
					"Name": "Test",
					"Type": "Movie"
				}
			}
		]`))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       100 * time.Millisecond,
		PublishAll:     true,
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	// Don't set a callback

	ctx := context.Background()

	// Poll should not panic even without callback
	poller.poll(ctx)
	// If we get here, no panic occurred
}

func TestJellyfinSessionPoller_IntegrationPolling(t *testing.T) {
	t.Parallel()

	var callCount int32

	// Create a mock server that returns active sessions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{
				"Id": "session-123",
				"UserName": "TestUser",
				"NowPlayingItem": {
					"Id": "item-456",
					"Name": "Test Movie",
					"Type": "Movie"
				}
			}
		]`))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       50 * time.Millisecond,
		PublishAll:     true,
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the poller
	err := poller.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Let it run for a bit (should poll ~3-4 times including initial)
	time.Sleep(175 * time.Millisecond)

	// Stop the poller
	poller.Stop()

	// Should have been called at least 3 times (initial + 2 interval polls)
	if count := atomic.LoadInt32(&callCount); count < 3 {
		t.Errorf("callback called %d times, want at least 3", count)
	}
}
