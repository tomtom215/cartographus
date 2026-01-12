// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// ============================================================================
// Session Poller Deduplication Stress Tests
// ============================================================================

func TestJellyfinSessionPoller_DeduplicationStress(t *testing.T) {
	t.Parallel()

	const numSessions = 100
	const numPollCycles = 10

	var callCount int32

	// Create mock server that returns many sessions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Generate JSON for many sessions
		sessions := "["
		for i := 0; i < numSessions; i++ {
			if i > 0 {
				sessions += ","
			}
			sessions += fmt.Sprintf(`{
				"Id": "jellyfin-stress-session-%d",
				"UserName": "User%d",
				"NowPlayingItem": {
					"Id": "item-%d",
					"Name": "Movie %d",
					"Type": "Movie"
				},
				"PlayState": {
					"IsPaused": false
				}
			}`, i, i, i, i)
		}
		sessions += "]"
		_, _ = w.Write([]byte(sessions))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour, // Long interval - manual polling
		PublishAll:     false,         // Deduplication enabled
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// Poll multiple times - should only see each session once
	for i := 0; i < numPollCycles; i++ {
		poller.poll(ctx)
	}

	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != int32(numSessions) {
		t.Errorf("callback called %d times, want %d (each session should be seen once)", finalCount, numSessions)
	}

	// Verify all sessions are tracked
	// LRUCache is thread-safe, no external lock needed
	trackedCount := poller.seenSessions.Len()

	if trackedCount != numSessions {
		t.Errorf("tracked sessions = %d, want %d", trackedCount, numSessions)
	}
}

func TestEmbySessionPoller_DeduplicationStress(t *testing.T) {
	t.Parallel()

	const numSessions = 100
	const numPollCycles = 10

	var callCount int32

	// Create mock server that returns many sessions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Generate JSON for many sessions
		sessions := "["
		for i := 0; i < numSessions; i++ {
			if i > 0 {
				sessions += ","
			}
			sessions += fmt.Sprintf(`{
				"Id": "emby-stress-session-%d",
				"UserName": "User%d",
				"NowPlayingItem": {
					"Id": "item-%d",
					"Name": "Movie %d",
					"Type": "Movie"
				},
				"PlayState": {
					"IsPaused": false
				}
			}`, i, i, i, i)
		}
		sessions += "]"
		_, _ = w.Write([]byte(sessions))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour, // Long interval - manual polling
		PublishAll:     false,         // Deduplication enabled
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewEmbyClient(server.URL, "test-key", "")
	poller := NewEmbySessionPoller(client, config)

	poller.SetOnSession(func(_ *models.EmbySession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// Poll multiple times - should only see each session once
	for i := 0; i < numPollCycles; i++ {
		poller.poll(ctx)
	}

	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != int32(numSessions) {
		t.Errorf("callback called %d times, want %d (each session should be seen once)", finalCount, numSessions)
	}
}

// ============================================================================
// Concurrent Access Stress Tests
// ============================================================================

func TestJellyfinSessionPoller_ConcurrentDeduplication(t *testing.T) {
	t.Parallel()

	const numGoroutines = 10
	const sessionsPerGoroutine = 50

	var callCount int32

	// Create a mock server that returns sessions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		sessions := "["
		for i := 0; i < sessionsPerGoroutine; i++ {
			if i > 0 {
				sessions += ","
			}
			sessions += fmt.Sprintf(`{
				"Id": "concurrent-session-%d",
				"UserName": "User%d",
				"NowPlayingItem": {
					"Id": "item-%d",
					"Name": "Movie %d",
					"Type": "Movie"
				}
			}`, i, i, i, i)
		}
		sessions += "]"
		_, _ = w.Write([]byte(sessions))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     false,
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()
	var wg sync.WaitGroup

	// Start multiple goroutines polling concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			poller.poll(ctx)
		}()
	}

	wg.Wait()

	// Each unique session should only be seen exactly once across all goroutines
	// Using atomic IsDuplicate ensures no race conditions in concurrent polling
	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != int32(sessionsPerGoroutine) {
		t.Errorf("callback called %d times, want exactly %d (atomic dedup)", finalCount, sessionsPerGoroutine)
	}
}

func TestEmbySessionPoller_ConcurrentDeduplication(t *testing.T) {
	t.Parallel()

	const numGoroutines = 10
	const sessionsPerGoroutine = 50

	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		sessions := "["
		for i := 0; i < sessionsPerGoroutine; i++ {
			if i > 0 {
				sessions += ","
			}
			sessions += fmt.Sprintf(`{
				"Id": "emby-concurrent-%d",
				"UserName": "User%d",
				"NowPlayingItem": {
					"Id": "item-%d",
					"Name": "Movie %d",
					"Type": "Movie"
				}
			}`, i, i, i, i)
		}
		sessions += "]"
		_, _ = w.Write([]byte(sessions))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     false,
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewEmbyClient(server.URL, "test-key", "")
	poller := NewEmbySessionPoller(client, config)

	poller.SetOnSession(func(_ *models.EmbySession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			poller.poll(ctx)
		}()
	}

	wg.Wait()

	// Each unique session should only be seen exactly once across all goroutines
	// Using atomic IsDuplicate ensures no race conditions in concurrent polling
	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != int32(sessionsPerGoroutine) {
		t.Errorf("callback called %d times, want exactly %d (atomic dedup)", finalCount, sessionsPerGoroutine)
	}
}

// ============================================================================
// TTL Expiration Stress Tests
// ============================================================================

func TestJellyfinSessionPoller_TTLExpirationStress(t *testing.T) {
	t.Parallel()

	const numSessions = 20
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		sessions := "["
		for i := 0; i < numSessions; i++ {
			if i > 0 {
				sessions += ","
			}
			sessions += fmt.Sprintf(`{
				"Id": "ttl-session-%d",
				"UserName": "User%d",
				"NowPlayingItem": {
					"Id": "item-%d",
					"Name": "Movie %d",
					"Type": "Movie"
				}
			}`, i, i, i, i)
		}
		sessions += "]"
		_, _ = w.Write([]byte(sessions))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     false,
		SeenSessionTTL: 50 * time.Millisecond, // Very short TTL for testing
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// First poll - see all sessions
	poller.poll(ctx)

	firstCount := atomic.LoadInt32(&callCount)
	if firstCount != int32(numSessions) {
		t.Errorf("first poll: callback called %d times, want %d", firstCount, numSessions)
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Cleanup expired sessions
	poller.cleanupSeenSessions()

	// Second poll - should see all sessions again after TTL expiry
	poller.poll(ctx)

	finalCount := atomic.LoadInt32(&callCount)
	expectedTotal := numSessions * 2
	if finalCount != int32(expectedTotal) {
		t.Errorf("after TTL expiry: callback called %d times, want %d", finalCount, expectedTotal)
	}
}

func TestEmbySessionPoller_TTLExpirationStress(t *testing.T) {
	t.Parallel()

	const numSessions = 20
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		sessions := "["
		for i := 0; i < numSessions; i++ {
			if i > 0 {
				sessions += ","
			}
			sessions += fmt.Sprintf(`{
				"Id": "emby-ttl-session-%d",
				"UserName": "User%d",
				"NowPlayingItem": {
					"Id": "item-%d",
					"Name": "Movie %d",
					"Type": "Movie"
				}
			}`, i, i, i, i)
		}
		sessions += "]"
		_, _ = w.Write([]byte(sessions))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     false,
		SeenSessionTTL: 50 * time.Millisecond,
	}

	client := NewEmbyClient(server.URL, "test-key", "")
	poller := NewEmbySessionPoller(client, config)

	poller.SetOnSession(func(_ *models.EmbySession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// First poll
	poller.poll(ctx)

	firstCount := atomic.LoadInt32(&callCount)
	if firstCount != int32(numSessions) {
		t.Errorf("first poll: callback called %d times, want %d", firstCount, numSessions)
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Cleanup expired sessions
	poller.cleanupSeenSessions()

	// Second poll - should see all sessions again
	poller.poll(ctx)

	finalCount := atomic.LoadInt32(&callCount)
	expectedTotal := numSessions * 2
	if finalCount != int32(expectedTotal) {
		t.Errorf("after TTL expiry: callback called %d times, want %d", finalCount, expectedTotal)
	}
}

// ============================================================================
// Memory Pressure Tests (Large Session Counts)
// ============================================================================

func TestJellyfinSessionPoller_HighSessionCount(t *testing.T) {
	t.Parallel()

	const numSessions = 1000 // High session count

	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		sessions := "["
		for i := 0; i < numSessions; i++ {
			if i > 0 {
				sessions += ","
			}
			sessions += fmt.Sprintf(`{
				"Id": "high-volume-session-%d",
				"UserName": "User%d",
				"NowPlayingItem": {
					"Id": "item-%d",
					"Name": "Movie %d",
					"Type": "Movie"
				}
			}`, i, i, i, i)
		}
		sessions += "]"
		_, _ = w.Write([]byte(sessions))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     false,
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// Poll once
	poller.poll(ctx)

	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != int32(numSessions) {
		t.Errorf("callback called %d times, want %d", finalCount, numSessions)
	}

	// Verify memory tracking
	// LRUCache is thread-safe, no external lock needed
	trackedCount := poller.seenSessions.Len()

	if trackedCount != numSessions {
		t.Errorf("tracked sessions = %d, want %d", trackedCount, numSessions)
	}
}

// ============================================================================
// Database Deduplication Tests
// ============================================================================

func TestDatabaseSessionKeyDeduplication(t *testing.T) {
	t.Parallel()

	// Track which session keys have been "inserted"
	var mu sync.Mutex
	insertedKeys := make(map[string]bool)
	var insertCount int32
	var duplicateAttempts int32

	db := &mockDB{
		sessionKeyExists: func(_ context.Context, sessionKey string) (bool, error) {
			mu.Lock()
			defer mu.Unlock()
			return insertedKeys[sessionKey], nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			mu.Lock()
			defer mu.Unlock()
			if insertedKeys[event.SessionKey] {
				atomic.AddInt32(&duplicateAttempts, 1)
				return fmt.Errorf("duplicate key")
			}
			insertedKeys[event.SessionKey] = true
			atomic.AddInt32(&insertCount, 1)
			return nil
		},
	}

	// Simulate processing events - some duplicates
	events := []string{
		"session-1", "session-2", "session-3",
		"session-1", // duplicate
		"session-4",
		"session-2", // duplicate
		"session-5",
		"session-1", // duplicate
	}

	ctx := context.Background()
	for _, sessionKey := range events {
		exists, _ := db.SessionKeyExists(ctx, sessionKey)
		if !exists {
			event := &models.PlaybackEvent{SessionKey: sessionKey}
			_ = db.InsertPlaybackEvent(event)
		}
	}

	// Should have inserted only unique keys
	if insertCount != 5 {
		t.Errorf("insert count = %d, want 5 unique keys", insertCount)
	}

	// No duplicate attempts should have happened (we check before insert)
	if duplicateAttempts != 0 {
		t.Errorf("duplicate attempts = %d, want 0", duplicateAttempts)
	}
}

func TestDatabaseSessionKeyDeduplication_Concurrent(t *testing.T) {
	t.Parallel()

	// Track which session keys have been "inserted"
	var mu sync.Mutex
	insertedKeys := make(map[string]bool)
	var insertCount int32

	db := &mockDB{
		sessionKeyExists: func(_ context.Context, sessionKey string) (bool, error) {
			mu.Lock()
			defer mu.Unlock()
			return insertedKeys[sessionKey], nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			mu.Lock()
			defer mu.Unlock()
			if insertedKeys[event.SessionKey] {
				return fmt.Errorf("duplicate key")
			}
			insertedKeys[event.SessionKey] = true
			atomic.AddInt32(&insertCount, 1)
			return nil
		},
	}

	const numGoroutines = 50
	const keysPerGoroutine = 20

	var wg sync.WaitGroup
	ctx := context.Background()

	// All goroutines try to insert the same set of keys
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < keysPerGoroutine; i++ {
				sessionKey := fmt.Sprintf("concurrent-key-%d", i)
				exists, _ := db.SessionKeyExists(ctx, sessionKey)
				if !exists {
					event := &models.PlaybackEvent{SessionKey: sessionKey}
					_ = db.InsertPlaybackEvent(event)
				}
			}
		}()
	}

	wg.Wait()

	// Should have inserted exactly keysPerGoroutine unique keys
	if insertCount != int32(keysPerGoroutine) {
		t.Errorf("insert count = %d, want %d unique keys", insertCount, keysPerGoroutine)
	}
}

// ============================================================================
// User Resolver Deduplication Tests
// ============================================================================

// mockUserResolverWithTracking tracks all calls for verification
type mockUserResolverWithTracking struct {
	mu          sync.Mutex
	calls       []userResolverCall
	nextUserID  int
	userMapping map[string]int // source:serverID:externalID -> userID
}

type userResolverCall struct {
	source         string
	serverID       string
	externalUserID string
}

func newMockUserResolverWithTracking() *mockUserResolverWithTracking {
	return &mockUserResolverWithTracking{
		nextUserID:  1,
		userMapping: make(map[string]int),
	}
}

func (m *mockUserResolverWithTracking) ResolveUserID(_ context.Context, source, serverID, externalUserID string, _, _ *string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, userResolverCall{
		source:         source,
		serverID:       serverID,
		externalUserID: externalUserID,
	})

	key := fmt.Sprintf("%s:%s:%s", source, serverID, externalUserID)
	if userID, exists := m.userMapping[key]; exists {
		return userID, nil
	}

	// Create new mapping
	userID := m.nextUserID
	m.nextUserID++
	m.userMapping[key] = userID
	return userID, nil
}

func (m *mockUserResolverWithTracking) getUniqueUserCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.userMapping)
}

func (m *mockUserResolverWithTracking) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func TestUserResolver_MultiServerDeduplication(t *testing.T) {
	t.Parallel()

	resolver := newMockUserResolverWithTracking()
	ctx := context.Background()

	// Same user ID on different servers - should get different internal IDs
	users := []struct {
		source         string
		serverID       string
		externalUserID string
	}{
		{"jellyfin", "server-1", "user-abc"},
		{"jellyfin", "server-2", "user-abc"}, // Same external ID, different server
		{"emby", "server-1", "user-abc"},     // Same external ID, different source
		{"jellyfin", "server-1", "user-abc"}, // Duplicate - should return same ID
		{"plex", "server-1", "12345"},
		{"plex", "server-2", "12345"}, // Same external ID, different server
	}

	results := make(map[string]int)
	for _, u := range users {
		key := fmt.Sprintf("%s:%s:%s", u.source, u.serverID, u.externalUserID)
		userID, err := resolver.ResolveUserID(ctx, u.source, u.serverID, u.externalUserID, nil, nil)
		if err != nil {
			t.Errorf("ResolveUserID failed: %v", err)
		}
		results[key] = userID
	}

	// Should have 5 unique users (6 calls, 1 duplicate)
	if resolver.getUniqueUserCount() != 5 {
		t.Errorf("unique user count = %d, want 5", resolver.getUniqueUserCount())
	}

	// Should have made 6 calls total
	if resolver.getCallCount() != 6 {
		t.Errorf("call count = %d, want 6", resolver.getCallCount())
	}

	// Verify duplicate returns same ID
	if results["jellyfin:server-1:user-abc"] != 1 {
		t.Error("first jellyfin user should have ID 1")
	}
}

func TestUserResolver_ConcurrentResolution(t *testing.T) {
	t.Parallel()

	resolver := newMockUserResolverWithTracking()
	ctx := context.Background()

	const numGoroutines = 20
	const usersPerGoroutine = 10

	var wg sync.WaitGroup

	// All goroutines try to resolve the same set of users
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < usersPerGoroutine; i++ {
				externalID := fmt.Sprintf("user-%d", i)
				_, _ = resolver.ResolveUserID(ctx, "jellyfin", "server-1", externalID, nil, nil)
			}
		}(g)
	}

	wg.Wait()

	// Should have exactly usersPerGoroutine unique users
	if resolver.getUniqueUserCount() != usersPerGoroutine {
		t.Errorf("unique user count = %d, want %d", resolver.getUniqueUserCount(), usersPerGoroutine)
	}

	// Should have made numGoroutines * usersPerGoroutine calls
	expectedCalls := numGoroutines * usersPerGoroutine
	if resolver.getCallCount() != expectedCalls {
		t.Errorf("call count = %d, want %d", resolver.getCallCount(), expectedCalls)
	}
}

// ============================================================================
// Cross-Source Event Flow Tests
// ============================================================================

// crossSourceMockPublisher tracks all published events for deduplication testing
type crossSourceMockPublisher struct {
	mu             sync.Mutex
	publishedKeys  map[string]bool
	publishedCount int32
}

func (m *crossSourceMockPublisher) PublishPlaybackEvent(_ context.Context, event *models.PlaybackEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishedKeys == nil {
		m.publishedKeys = make(map[string]bool)
	}
	if !m.publishedKeys[event.SessionKey] {
		m.publishedKeys[event.SessionKey] = true
		atomic.AddInt32(&m.publishedCount, 1)
	}
	return nil
}

func TestCrossSourceEventDeduplication(t *testing.T) {
	t.Parallel()

	publisher := &crossSourceMockPublisher{
		publishedKeys: make(map[string]bool),
	}

	// Simulate concurrent event publishing from multiple sources
	var wg sync.WaitGroup
	sources := []string{"jellyfin", "emby", "plex"}

	for _, source := range sources {
		wg.Add(1)
		go func(src string) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				// Each source generates unique session keys with source prefix
				sessionKey := fmt.Sprintf("%s-session-%d", src, i)
				state := "playing"
				event := &models.PlaybackEvent{
					SessionKey: sessionKey,
					State:      &state,
				}
				_ = publisher.PublishPlaybackEvent(context.Background(), event)
			}
		}(source)
	}

	wg.Wait()

	// Should have 300 unique events (100 per source)
	if publisher.publishedCount != 300 {
		t.Errorf("unique published events = %d, want 300", publisher.publishedCount)
	}
}

// ============================================================================
// PublishAll Mode Tests
// ============================================================================

func TestJellyfinSessionPoller_PublishAllMode(t *testing.T) {
	t.Parallel()

	const numSessions = 10
	const numPollCycles = 5

	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		sessions := "["
		for i := 0; i < numSessions; i++ {
			if i > 0 {
				sessions += ","
			}
			sessions += fmt.Sprintf(`{
				"Id": "publishall-session-%d",
				"UserName": "User%d",
				"NowPlayingItem": {
					"Id": "item-%d",
					"Name": "Movie %d",
					"Type": "Movie"
				}
			}`, i, i, i, i)
		}
		sessions += "]"
		_, _ = w.Write([]byte(sessions))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     true, // Publish all sessions every time
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// Poll multiple times - should see all sessions each time
	for i := 0; i < numPollCycles; i++ {
		poller.poll(ctx)
	}

	expectedTotal := numSessions * numPollCycles
	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != int32(expectedTotal) {
		t.Errorf("callback called %d times, want %d (PublishAll mode)", finalCount, expectedTotal)
	}
}

func TestEmbySessionPoller_PublishAllMode(t *testing.T) {
	t.Parallel()

	const numSessions = 10
	const numPollCycles = 5

	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		sessions := "["
		for i := 0; i < numSessions; i++ {
			if i > 0 {
				sessions += ","
			}
			sessions += fmt.Sprintf(`{
				"Id": "emby-publishall-%d",
				"UserName": "User%d",
				"NowPlayingItem": {
					"Id": "item-%d",
					"Name": "Movie %d",
					"Type": "Movie"
				}
			}`, i, i, i, i)
		}
		sessions += "]"
		_, _ = w.Write([]byte(sessions))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     true,
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewEmbyClient(server.URL, "test-key", "")
	poller := NewEmbySessionPoller(client, config)

	poller.SetOnSession(func(_ *models.EmbySession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	for i := 0; i < numPollCycles; i++ {
		poller.poll(ctx)
	}

	expectedTotal := numSessions * numPollCycles
	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != int32(expectedTotal) {
		t.Errorf("callback called %d times, want %d (PublishAll mode)", finalCount, expectedTotal)
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestSessionPoller_EmptySessionID(t *testing.T) {
	t.Parallel()

	var callCount int32

	// Server returns session with empty ID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{
			"Id": "",
			"UserName": "User",
			"NowPlayingItem": {
				"Id": "item-1",
				"Name": "Movie",
				"Type": "Movie"
			}
		}]`))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     false,
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// Poll multiple times
	for i := 0; i < 3; i++ {
		poller.poll(ctx)
	}

	// Empty ID should still be tracked (even if unusual)
	// Behavior depends on implementation - verify it doesn't panic
	if callCount < 1 {
		t.Error("expected at least one callback for session with empty ID")
	}
}

func TestSessionPoller_RapidPolling(t *testing.T) {
	t.Parallel()

	const numPolls = 100
	var callCount int32
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{
			"Id": "rapid-session-1",
			"UserName": "User",
			"NowPlayingItem": {
				"Id": "item-1",
				"Name": "Movie",
				"Type": "Movie"
			}
		}]`))
	}))
	defer server.Close()

	config := SessionPollerConfig{
		Interval:       1 * time.Hour,
		PublishAll:     false,
		SeenSessionTTL: 5 * time.Minute,
	}

	client := NewJellyfinClient(server.URL, "test-key", "")
	poller := NewJellyfinSessionPoller(client, config)

	poller.SetOnSession(func(_ *models.JellyfinSession) {
		atomic.AddInt32(&callCount, 1)
	})

	ctx := context.Background()

	// Rapid-fire polling
	for i := 0; i < numPolls; i++ {
		poller.poll(ctx)
	}

	// Should have made numPolls requests
	if requestCount != int32(numPolls) {
		t.Errorf("request count = %d, want %d", requestCount, numPolls)
	}

	// But only one callback (deduplication)
	if callCount != 1 {
		t.Errorf("callback count = %d, want 1 (deduplication should work)", callCount)
	}
}
