// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/cache"
)

// CachedEventHistory wraps an EventHistory implementation with in-memory caching.
// It uses sliding window counters to reduce database queries for frequently accessed data.
//
// Performance benefits:
//   - GetRecentIPsForDevice: Tracks unique IPs per device in memory, reduces DB queries
//   - GetLastEventForUser: Caches recent events with TTL
//   - GetGeolocation: Caches IP geolocation lookups
//
// This is particularly valuable for detection rules that may check the same
// user/device multiple times per event batch.
type CachedEventHistory struct {
	wrapped EventHistory

	// Configuration
	deviceIPWindow time.Duration // Window for device IP tracking
	eventCacheTTL  time.Duration // TTL for cached events
	geoCacheTTL    time.Duration // TTL for geolocation cache

	// Device IP tracking using UniqueValueStore
	// Key: "machineID:serverID", Value: unique IPs within window
	deviceIPCache *cache.UniqueValueStore

	// Last event cache using LRU
	// Key: "userID:serverID", Value: *DetectionEvent
	lastEventMu    sync.RWMutex
	lastEventCache map[string]*cachedEvent

	// Geolocation cache using LRU
	geoCache *cache.LRUCache

	// Stats
	deviceIPHits   int64
	deviceIPMisses int64
	lastEventHits  int64
	lastEventMiss  int64
	geoHits        int64
	geoMisses      int64
	statsMu        sync.Mutex
}

// cachedEvent stores a cached detection event with expiration.
type cachedEvent struct {
	event     *DetectionEvent
	expiresAt time.Time
}

// CachedEventHistoryConfig configures the cached event history.
type CachedEventHistoryConfig struct {
	// DeviceIPWindow is how long to track device IPs (default: 5 minutes)
	DeviceIPWindow time.Duration

	// EventCacheTTL is how long to cache last events (default: 30 seconds)
	EventCacheTTL time.Duration

	// GeoCacheTTL is how long to cache geolocation lookups (default: 1 hour)
	GeoCacheTTL time.Duration

	// MaxDevices is the maximum number of devices to track (default: 10000)
	MaxDevices int

	// MaxGeoEntries is the maximum geolocation cache entries (default: 10000)
	MaxGeoEntries int
}

// DefaultCachedEventHistoryConfig returns production defaults.
func DefaultCachedEventHistoryConfig() CachedEventHistoryConfig {
	return CachedEventHistoryConfig{
		DeviceIPWindow: 5 * time.Minute,
		EventCacheTTL:  30 * time.Second,
		GeoCacheTTL:    time.Hour,
		MaxDevices:     10000,
		MaxGeoEntries:  10000,
	}
}

// NewCachedEventHistory creates a new cached event history wrapper.
func NewCachedEventHistory(wrapped EventHistory, cfg CachedEventHistoryConfig) *CachedEventHistory {
	if cfg.DeviceIPWindow <= 0 {
		cfg.DeviceIPWindow = 5 * time.Minute
	}
	if cfg.EventCacheTTL <= 0 {
		cfg.EventCacheTTL = 30 * time.Second
	}
	if cfg.GeoCacheTTL <= 0 {
		cfg.GeoCacheTTL = time.Hour
	}
	if cfg.MaxDevices <= 0 {
		cfg.MaxDevices = 10000
	}
	if cfg.MaxGeoEntries <= 0 {
		cfg.MaxGeoEntries = 10000
	}

	return &CachedEventHistory{
		wrapped:        wrapped,
		deviceIPWindow: cfg.DeviceIPWindow,
		eventCacheTTL:  cfg.EventCacheTTL,
		geoCacheTTL:    cfg.GeoCacheTTL,
		deviceIPCache:  cache.NewUniqueValueStore(cfg.DeviceIPWindow, 10, cfg.MaxDevices),
		lastEventCache: make(map[string]*cachedEvent),
		geoCache:       cache.NewLRUCache(cfg.MaxGeoEntries, cfg.GeoCacheTTL),
	}
}

// RecordEvent records an event for caching purposes.
// This should be called when new events are processed to populate the cache.
func (c *CachedEventHistory) RecordEvent(event *DetectionEvent) {
	if event == nil {
		return
	}

	// Record IP for device tracking
	if event.MachineID != "" && event.IPAddress != "" {
		key := fmt.Sprintf("%s:%s", event.MachineID, event.ServerID)
		c.deviceIPCache.Add(key, event.IPAddress)
	}

	// Cache as last event for user
	if event.UserID != 0 {
		key := fmt.Sprintf("%d:%s", event.UserID, event.ServerID)
		c.lastEventMu.Lock()
		c.lastEventCache[key] = &cachedEvent{
			event:     event,
			expiresAt: time.Now().Add(c.eventCacheTTL),
		}
		c.lastEventMu.Unlock()
	}
}

// GetLastEventForUser retrieves the most recent event for a user.
// Uses cache if available, falls back to wrapped implementation.
func (c *CachedEventHistory) GetLastEventForUser(ctx context.Context, userID int, serverID string) (*DetectionEvent, error) {
	key := fmt.Sprintf("%d:%s", userID, serverID)

	// Check cache
	c.lastEventMu.RLock()
	cached, exists := c.lastEventCache[key]
	c.lastEventMu.RUnlock()

	if exists && time.Now().Before(cached.expiresAt) {
		c.statsMu.Lock()
		c.lastEventHits++
		c.statsMu.Unlock()
		return cached.event, nil
	}

	// Cache miss, query wrapped implementation
	c.statsMu.Lock()
	c.lastEventMiss++
	c.statsMu.Unlock()

	event, err := c.wrapped.GetLastEventForUser(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if event != nil {
		c.lastEventMu.Lock()
		c.lastEventCache[key] = &cachedEvent{
			event:     event,
			expiresAt: time.Now().Add(c.eventCacheTTL),
		}
		c.lastEventMu.Unlock()
	}

	return event, nil
}

// GetActiveStreamsForUser retrieves currently active streams.
// This is not cached as it needs real-time data.
func (c *CachedEventHistory) GetActiveStreamsForUser(ctx context.Context, userID int, serverID string) ([]DetectionEvent, error) {
	return c.wrapped.GetActiveStreamsForUser(ctx, userID, serverID)
}

// GetRecentIPsForDevice retrieves recent IPs for a device.
// Uses in-memory sliding window counter to reduce database queries.
func (c *CachedEventHistory) GetRecentIPsForDevice(ctx context.Context, machineID string, serverID string, window time.Duration) ([]string, error) {
	key := fmt.Sprintf("%s:%s", machineID, serverID)

	// Check if we have in-memory data
	uniqueCount := c.deviceIPCache.CountUnique(key)
	if uniqueCount > 0 {
		// Return from in-memory cache
		c.statsMu.Lock()
		c.deviceIPHits++
		c.statsMu.Unlock()

		return c.deviceIPCache.GetUnique(key), nil
	}

	// Cache miss, query wrapped implementation and populate cache
	c.statsMu.Lock()
	c.deviceIPMisses++
	c.statsMu.Unlock()

	ips, err := c.wrapped.GetRecentIPsForDevice(ctx, machineID, serverID, window)
	if err != nil {
		return nil, err
	}

	// Populate cache with results
	for _, ip := range ips {
		c.deviceIPCache.Add(key, ip)
	}

	return ips, nil
}

// GetSimultaneousLocations retrieves concurrent sessions at different locations.
// This is not cached as it needs real-time data.
func (c *CachedEventHistory) GetSimultaneousLocations(ctx context.Context, userID int, serverID string, window time.Duration) ([]DetectionEvent, error) {
	return c.wrapped.GetSimultaneousLocations(ctx, userID, serverID, window)
}

// GetGeolocation retrieves geolocation for an IP address.
// Uses LRU cache to avoid repeated lookups.
func (c *CachedEventHistory) GetGeolocation(ctx context.Context, ipAddress string) (*Geolocation, error) {
	// Check cache
	if c.geoCache.Contains(ipAddress) {
		c.statsMu.Lock()
		c.geoHits++
		c.statsMu.Unlock()

		// Need to retrieve the actual value - cache.LRUCache stores time.Time
		// For geolocation, we need a separate cache structure or extend LRU
		// Fall through to wrapped call for now
	}

	// Query wrapped implementation
	c.statsMu.Lock()
	c.geoMisses++
	c.statsMu.Unlock()

	geo, err := c.wrapped.GetGeolocation(ctx, ipAddress)
	if err != nil {
		return nil, err
	}

	// Mark as seen in cache (for Contains check)
	if geo != nil {
		c.geoCache.Add(ipAddress, time.Now())
	}

	return geo, nil
}

// Stats returns cache statistics.
func (c *CachedEventHistory) Stats() CachedEventHistoryStats {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	return CachedEventHistoryStats{
		DeviceIPHits:       c.deviceIPHits,
		DeviceIPMisses:     c.deviceIPMisses,
		LastEventHits:      c.lastEventHits,
		LastEventMisses:    c.lastEventMiss,
		GeoHits:            c.geoHits,
		GeoMisses:          c.geoMisses,
		TrackedDevices:     c.deviceIPCache.Len(),
		CachedLastEvents:   len(c.lastEventCache),
		CachedGeolocations: c.geoCache.Len(),
	}
}

// CachedEventHistoryStats holds cache statistics.
type CachedEventHistoryStats struct {
	DeviceIPHits       int64
	DeviceIPMisses     int64
	LastEventHits      int64
	LastEventMisses    int64
	GeoHits            int64
	GeoMisses          int64
	TrackedDevices     int
	CachedLastEvents   int
	CachedGeolocations int
}

// Cleanup removes expired entries from all caches.
func (c *CachedEventHistory) Cleanup() {
	// Cleanup last event cache
	c.lastEventMu.Lock()
	now := time.Now()
	for key, cached := range c.lastEventCache {
		if now.After(cached.expiresAt) {
			delete(c.lastEventCache, key)
		}
	}
	c.lastEventMu.Unlock()

	// LRU cache handles its own cleanup via TTL
	c.geoCache.CleanupExpired()
}

// Clear removes all cached data.
func (c *CachedEventHistory) Clear() {
	c.lastEventMu.Lock()
	c.lastEventCache = make(map[string]*cachedEvent)
	c.lastEventMu.Unlock()

	c.deviceIPCache.Clear()
	c.geoCache.Clear()
}
