// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// configurableMockEventHistory is a mock that allows configuring responses and tracking calls
type configurableMockEventHistory struct {
	mu sync.Mutex

	// Return values
	lastEvent             *DetectionEvent
	lastEventErr          error
	activeStreams         []DetectionEvent
	activeStreamsErr      error
	recentIPs             []string
	recentIPsErr          error
	simultaneousLocations []DetectionEvent
	simultaneousErr       error
	geolocation           *Geolocation
	geolocationErr        error

	// Call tracking
	lastEventCalls     int
	activeStreamsCalls int
	recentIPsCalls     int
	simultaneousCalls  int
	geolocationCalls   int
}

func (m *configurableMockEventHistory) GetLastEventForUser(ctx context.Context, userID int, serverID string) (*DetectionEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastEventCalls++
	return m.lastEvent, m.lastEventErr
}

func (m *configurableMockEventHistory) GetActiveStreamsForUser(ctx context.Context, userID int, serverID string) ([]DetectionEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeStreamsCalls++
	return m.activeStreams, m.activeStreamsErr
}

func (m *configurableMockEventHistory) GetRecentIPsForDevice(ctx context.Context, machineID string, serverID string, window time.Duration) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recentIPsCalls++
	return m.recentIPs, m.recentIPsErr
}

func (m *configurableMockEventHistory) GetSimultaneousLocations(ctx context.Context, userID int, serverID string, window time.Duration) ([]DetectionEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simultaneousCalls++
	return m.simultaneousLocations, m.simultaneousErr
}

func (m *configurableMockEventHistory) GetGeolocation(ctx context.Context, ipAddress string) (*Geolocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.geolocationCalls++
	return m.geolocation, m.geolocationErr
}

func TestDefaultCachedEventHistoryConfig(t *testing.T) {
	config := DefaultCachedEventHistoryConfig()

	if config.DeviceIPWindow != 5*time.Minute {
		t.Errorf("DeviceIPWindow = %v, want 5m", config.DeviceIPWindow)
	}
	if config.EventCacheTTL != 30*time.Second {
		t.Errorf("EventCacheTTL = %v, want 30s", config.EventCacheTTL)
	}
	if config.GeoCacheTTL != time.Hour {
		t.Errorf("GeoCacheTTL = %v, want 1h", config.GeoCacheTTL)
	}
	if config.MaxDevices != 10000 {
		t.Errorf("MaxDevices = %v, want 10000", config.MaxDevices)
	}
	if config.MaxGeoEntries != 10000 {
		t.Errorf("MaxGeoEntries = %v, want 10000", config.MaxGeoEntries)
	}
}

func TestNewCachedEventHistory(t *testing.T) {
	mock := &configurableMockEventHistory{}
	config := DefaultCachedEventHistoryConfig()

	cached := NewCachedEventHistory(mock, config)

	if cached == nil {
		t.Fatal("cached should not be nil")
	}
	if cached.wrapped != mock {
		t.Error("wrapped should be the mock")
	}
}

func TestNewCachedEventHistory_ZeroConfig(t *testing.T) {
	mock := &configurableMockEventHistory{}
	config := CachedEventHistoryConfig{} // All zeros

	cached := NewCachedEventHistory(mock, config)

	// Should use defaults for zero values
	if cached.deviceIPWindow != 5*time.Minute {
		t.Errorf("deviceIPWindow = %v, want 5m", cached.deviceIPWindow)
	}
	if cached.eventCacheTTL != 30*time.Second {
		t.Errorf("eventCacheTTL = %v, want 30s", cached.eventCacheTTL)
	}
	if cached.geoCacheTTL != time.Hour {
		t.Errorf("geoCacheTTL = %v, want 1h", cached.geoCacheTTL)
	}
}

func TestCachedEventHistory_RecordEvent_NilEvent(t *testing.T) {
	mock := &configurableMockEventHistory{}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// Should not panic
	cached.RecordEvent(nil)
}

func TestCachedEventHistory_RecordEvent(t *testing.T) {
	mock := &configurableMockEventHistory{}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	event := &DetectionEvent{
		UserID:    42,
		Username:  "testuser",
		MachineID: "machine123",
		ServerID:  "server1",
		IPAddress: "1.2.3.4",
		Timestamp: time.Now(),
	}

	cached.RecordEvent(event)

	// Verify the event is cached for GetLastEventForUser
	cachedEvent, err := cached.GetLastEventForUser(context.Background(), 42, "server1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cachedEvent != event {
		t.Error("cached event should match recorded event")
	}

	// Should not call wrapped (cache hit)
	if mock.lastEventCalls != 0 {
		t.Errorf("expected 0 calls to wrapped, got %d", mock.lastEventCalls)
	}

	// Check stats
	stats := cached.Stats()
	if stats.LastEventHits != 1 {
		t.Errorf("LastEventHits = %d, want 1", stats.LastEventHits)
	}
}

func TestCachedEventHistory_GetLastEventForUser_CacheMiss(t *testing.T) {
	expectedEvent := &DetectionEvent{
		UserID:    42,
		Username:  "testuser",
		Timestamp: time.Now(),
	}

	mock := &configurableMockEventHistory{
		lastEvent: expectedEvent,
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// First call should be cache miss
	event, err := cached.GetLastEventForUser(context.Background(), 42, "server1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event != expectedEvent {
		t.Error("returned event should match mock event")
	}
	if mock.lastEventCalls != 1 {
		t.Errorf("expected 1 call to wrapped, got %d", mock.lastEventCalls)
	}

	// Second call should be cache hit
	event2, err := cached.GetLastEventForUser(context.Background(), 42, "server1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event2 != expectedEvent {
		t.Error("cached event should match")
	}
	if mock.lastEventCalls != 1 {
		t.Errorf("expected still 1 call to wrapped (cache hit), got %d", mock.lastEventCalls)
	}

	stats := cached.Stats()
	if stats.LastEventMisses != 1 {
		t.Errorf("LastEventMisses = %d, want 1", stats.LastEventMisses)
	}
	if stats.LastEventHits != 1 {
		t.Errorf("LastEventHits = %d, want 1", stats.LastEventHits)
	}
}

func TestCachedEventHistory_GetLastEventForUser_Error(t *testing.T) {
	mock := &configurableMockEventHistory{
		lastEventErr: errors.New("database error"),
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	event, err := cached.GetLastEventForUser(context.Background(), 42, "server1")
	if err == nil {
		t.Error("expected error")
	}
	if event != nil {
		t.Error("event should be nil on error")
	}
}

func TestCachedEventHistory_GetLastEventForUser_NilResult(t *testing.T) {
	mock := &configurableMockEventHistory{
		lastEvent: nil, // No event found
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	event, err := cached.GetLastEventForUser(context.Background(), 42, "server1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event != nil {
		t.Error("event should be nil when not found")
	}
}

func TestCachedEventHistory_GetActiveStreamsForUser(t *testing.T) {
	expectedStreams := []DetectionEvent{
		{SessionKey: "session1", UserID: 42},
		{SessionKey: "session2", UserID: 42},
	}

	mock := &configurableMockEventHistory{
		activeStreams: expectedStreams,
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// Active streams are not cached - always calls wrapped
	streams, err := cached.GetActiveStreamsForUser(context.Background(), 42, "server1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streams) != 2 {
		t.Errorf("expected 2 streams, got %d", len(streams))
	}
	if mock.activeStreamsCalls != 1 {
		t.Errorf("expected 1 call, got %d", mock.activeStreamsCalls)
	}

	// Second call should also call wrapped (no caching)
	cached.GetActiveStreamsForUser(context.Background(), 42, "server1")
	if mock.activeStreamsCalls != 2 {
		t.Errorf("expected 2 calls (no caching), got %d", mock.activeStreamsCalls)
	}
}

func TestCachedEventHistory_GetRecentIPsForDevice_CacheHit(t *testing.T) {
	mock := &configurableMockEventHistory{
		recentIPs: []string{"1.2.3.4", "5.6.7.8"},
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// Record some events to populate the device IP cache
	cached.RecordEvent(&DetectionEvent{
		MachineID: "machine123",
		ServerID:  "server1",
		IPAddress: "1.2.3.4",
		Timestamp: time.Now(),
	})
	cached.RecordEvent(&DetectionEvent{
		MachineID: "machine123",
		ServerID:  "server1",
		IPAddress: "5.6.7.8",
		Timestamp: time.Now(),
	})

	// Should get from cache
	ips, err := cached.GetRecentIPsForDevice(context.Background(), "machine123", "server1", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 2 {
		t.Errorf("expected 2 IPs, got %d", len(ips))
	}

	// Should not call wrapped (cache hit)
	if mock.recentIPsCalls != 0 {
		t.Errorf("expected 0 calls to wrapped (cache hit), got %d", mock.recentIPsCalls)
	}

	stats := cached.Stats()
	if stats.DeviceIPHits != 1 {
		t.Errorf("DeviceIPHits = %d, want 1", stats.DeviceIPHits)
	}
}

func TestCachedEventHistory_GetRecentIPsForDevice_CacheMiss(t *testing.T) {
	expectedIPs := []string{"1.2.3.4", "5.6.7.8"}
	mock := &configurableMockEventHistory{
		recentIPs: expectedIPs,
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// No events recorded, should be cache miss
	ips, err := cached.GetRecentIPsForDevice(context.Background(), "machine123", "server1", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 2 {
		t.Errorf("expected 2 IPs, got %d", len(ips))
	}

	if mock.recentIPsCalls != 1 {
		t.Errorf("expected 1 call to wrapped (cache miss), got %d", mock.recentIPsCalls)
	}

	stats := cached.Stats()
	if stats.DeviceIPMisses != 1 {
		t.Errorf("DeviceIPMisses = %d, want 1", stats.DeviceIPMisses)
	}
}

func TestCachedEventHistory_GetRecentIPsForDevice_Error(t *testing.T) {
	mock := &configurableMockEventHistory{
		recentIPsErr: errors.New("database error"),
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	ips, err := cached.GetRecentIPsForDevice(context.Background(), "machine123", "server1", 5*time.Minute)
	if err == nil {
		t.Error("expected error")
	}
	if ips != nil {
		t.Error("ips should be nil on error")
	}
}

func TestCachedEventHistory_GetSimultaneousLocations(t *testing.T) {
	expectedLocations := []DetectionEvent{
		{SessionKey: "session1", Latitude: 40.7128, Longitude: -74.0060},
	}

	mock := &configurableMockEventHistory{
		simultaneousLocations: expectedLocations,
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// Simultaneous locations are not cached - always calls wrapped
	locations, err := cached.GetSimultaneousLocations(context.Background(), 42, "server1", 30*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locations) != 1 {
		t.Errorf("expected 1 location, got %d", len(locations))
	}

	if mock.simultaneousCalls != 1 {
		t.Errorf("expected 1 call, got %d", mock.simultaneousCalls)
	}

	// Second call should also call wrapped (no caching)
	cached.GetSimultaneousLocations(context.Background(), 42, "server1", 30*time.Minute)
	if mock.simultaneousCalls != 2 {
		t.Errorf("expected 2 calls (no caching), got %d", mock.simultaneousCalls)
	}
}

func TestCachedEventHistory_GetGeolocation(t *testing.T) {
	expectedGeo := &Geolocation{
		IPAddress: "1.2.3.4",
		Latitude:  40.7128,
		Longitude: -74.0060,
		City:      "New York",
		Country:   "US",
	}

	mock := &configurableMockEventHistory{
		geolocation: expectedGeo,
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// First call - cache miss
	geo, err := cached.GetGeolocation(context.Background(), "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if geo != expectedGeo {
		t.Error("returned geo should match mock")
	}
	if mock.geolocationCalls != 1 {
		t.Errorf("expected 1 call, got %d", mock.geolocationCalls)
	}

	stats := cached.Stats()
	if stats.GeoMisses != 1 {
		t.Errorf("GeoMisses = %d, want 1", stats.GeoMisses)
	}
}

func TestCachedEventHistory_GetGeolocation_Error(t *testing.T) {
	mock := &configurableMockEventHistory{
		geolocationErr: errors.New("lookup failed"),
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	geo, err := cached.GetGeolocation(context.Background(), "1.2.3.4")
	if err == nil {
		t.Error("expected error")
	}
	if geo != nil {
		t.Error("geo should be nil on error")
	}
}

func TestCachedEventHistory_Stats(t *testing.T) {
	mock := &configurableMockEventHistory{
		lastEvent: &DetectionEvent{UserID: 42, Timestamp: time.Now()},
	}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// Generate some stats
	cached.GetLastEventForUser(context.Background(), 42, "server1") // miss
	cached.GetLastEventForUser(context.Background(), 42, "server1") // hit
	cached.GetLastEventForUser(context.Background(), 42, "server1") // hit

	cached.RecordEvent(&DetectionEvent{
		MachineID: "m1",
		ServerID:  "s1",
		IPAddress: "1.1.1.1",
	})
	cached.GetRecentIPsForDevice(context.Background(), "m1", "s1", 5*time.Minute) // hit

	stats := cached.Stats()

	if stats.LastEventMisses != 1 {
		t.Errorf("LastEventMisses = %d, want 1", stats.LastEventMisses)
	}
	if stats.LastEventHits != 2 {
		t.Errorf("LastEventHits = %d, want 2", stats.LastEventHits)
	}
	if stats.DeviceIPHits != 1 {
		t.Errorf("DeviceIPHits = %d, want 1", stats.DeviceIPHits)
	}
	if stats.CachedLastEvents < 1 {
		t.Errorf("CachedLastEvents = %d, want >= 1", stats.CachedLastEvents)
	}
}

func TestCachedEventHistory_Cleanup(t *testing.T) {
	mock := &configurableMockEventHistory{}
	config := CachedEventHistoryConfig{
		DeviceIPWindow: 5 * time.Minute,
		EventCacheTTL:  50 * time.Millisecond, // Very short TTL for testing
		GeoCacheTTL:    time.Hour,
		MaxDevices:     1000,
		MaxGeoEntries:  1000,
	}
	cached := NewCachedEventHistory(mock, config)

	// Record an event
	cached.RecordEvent(&DetectionEvent{
		UserID:    42,
		ServerID:  "server1",
		Timestamp: time.Now(),
	})

	// Verify it's cached
	stats := cached.Stats()
	if stats.CachedLastEvents != 1 {
		t.Errorf("CachedLastEvents = %d, want 1", stats.CachedLastEvents)
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Run cleanup
	cached.Cleanup()

	// Verify expired entry was removed
	stats = cached.Stats()
	if stats.CachedLastEvents != 0 {
		t.Errorf("CachedLastEvents after cleanup = %d, want 0", stats.CachedLastEvents)
	}
}

func TestCachedEventHistory_Clear(t *testing.T) {
	mock := &configurableMockEventHistory{}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// Record some events
	cached.RecordEvent(&DetectionEvent{
		UserID:    42,
		MachineID: "m1",
		ServerID:  "s1",
		IPAddress: "1.2.3.4",
	})
	cached.RecordEvent(&DetectionEvent{
		UserID:    43,
		MachineID: "m2",
		ServerID:  "s1",
		IPAddress: "5.6.7.8",
	})

	// Verify something is cached
	stats := cached.Stats()
	if stats.CachedLastEvents == 0 {
		t.Error("expected some cached events")
	}

	// Clear all caches
	cached.Clear()

	// Verify everything is cleared
	stats = cached.Stats()
	if stats.CachedLastEvents != 0 {
		t.Errorf("CachedLastEvents after Clear = %d, want 0", stats.CachedLastEvents)
	}
	if stats.TrackedDevices != 0 {
		t.Errorf("TrackedDevices after Clear = %d, want 0", stats.TrackedDevices)
	}
}

func TestCachedEventHistory_CacheExpiration(t *testing.T) {
	mock := &configurableMockEventHistory{
		lastEvent: &DetectionEvent{UserID: 42, Timestamp: time.Now()},
	}
	config := CachedEventHistoryConfig{
		DeviceIPWindow: 5 * time.Minute,
		EventCacheTTL:  50 * time.Millisecond, // Very short TTL
		GeoCacheTTL:    time.Hour,
		MaxDevices:     1000,
		MaxGeoEntries:  1000,
	}
	cached := NewCachedEventHistory(mock, config)

	// First call - cache miss
	cached.GetLastEventForUser(context.Background(), 42, "server1")
	if mock.lastEventCalls != 1 {
		t.Errorf("expected 1 call, got %d", mock.lastEventCalls)
	}

	// Second call - cache hit
	cached.GetLastEventForUser(context.Background(), 42, "server1")
	if mock.lastEventCalls != 1 {
		t.Errorf("expected still 1 call (cache hit), got %d", mock.lastEventCalls)
	}

	// Wait for cache to expire
	time.Sleep(60 * time.Millisecond)

	// Third call - cache miss (expired)
	cached.GetLastEventForUser(context.Background(), 42, "server1")
	if mock.lastEventCalls != 2 {
		t.Errorf("expected 2 calls (cache expired), got %d", mock.lastEventCalls)
	}
}

func TestCachedEventHistory_RecordEvent_NoMachineID(t *testing.T) {
	mock := &configurableMockEventHistory{}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// Event without machine ID should still cache for user lookup
	event := &DetectionEvent{
		UserID:    42,
		ServerID:  "server1",
		IPAddress: "1.2.3.4",
		MachineID: "", // No machine ID
		Timestamp: time.Now(),
	}

	cached.RecordEvent(event)

	// Should be cached for user lookup
	cachedEvent, err := cached.GetLastEventForUser(context.Background(), 42, "server1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cachedEvent != event {
		t.Error("event should be cached for user lookup")
	}
}

func TestCachedEventHistory_RecordEvent_NoUserID(t *testing.T) {
	mock := &configurableMockEventHistory{}
	cached := NewCachedEventHistory(mock, DefaultCachedEventHistoryConfig())

	// Event without user ID should still cache for device IP tracking
	event := &DetectionEvent{
		UserID:    0, // No user ID
		MachineID: "machine123",
		ServerID:  "server1",
		IPAddress: "1.2.3.4",
		Timestamp: time.Now(),
	}

	cached.RecordEvent(event)

	// Device IP should still be tracked
	ips, err := cached.GetRecentIPsForDevice(context.Background(), "machine123", "server1", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 1 || ips[0] != "1.2.3.4" {
		t.Errorf("expected [1.2.3.4], got %v", ips)
	}
}
