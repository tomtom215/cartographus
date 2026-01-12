// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package cache

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// Entry represents a cached item with expiration
type Entry struct {
	Data      interface{}
	ExpiresAt time.Time
}

// Cache provides a thread-safe in-memory cache with TTL support
type Cache struct {
	mu      sync.RWMutex
	entries map[string]Entry
	ttl     time.Duration
	stats   Stats
}

// Stats tracks cache performance metrics
type Stats struct {
	mu          sync.RWMutex
	Hits        int64
	Misses      int64
	Evictions   int64
	TotalKeys   int64
	LastCleanup time.Time
}

// New creates a new thread-safe in-memory cache with automatic expiration.
//
// This constructor initializes a cache with the specified time-to-live (TTL) for all entries.
// It starts a background goroutine that performs cleanup every 5 minutes to remove expired entries.
//
// Parameters:
//   - ttl: Default expiration duration for cache entries (e.g., 5 * time.Minute)
//
// Returns:
//   - Pointer to initialized Cache with background cleanup goroutine running
//
// Thread Safety:
//   - Safe for concurrent access from multiple goroutines
//   - Uses sync.RWMutex for read/write locking
//   - Background cleanup goroutine runs for cache lifetime
//
// Performance:
//   - O(1) lookups with Go map
//   - Cleanup runs every 5 minutes (minimal overhead)
//   - Tracks hit rate, misses, evictions for monitoring
//
// Example:
//
//	cache := cache.New(5 * time.Minute)
//	cache.Set("key", value)
//	if data, ok := cache.Get("key"); ok {
//	    // Use cached data
//	}
func New(ttl time.Duration) *Cache {
	c := &Cache{
		entries: make(map[string]Entry),
		ttl:     ttl,
		stats: Stats{
			LastCleanup: time.Now(),
		},
	}

	// Start background cleanup goroutine
	go c.cleanupLoop()

	return c
}

// Get retrieves a value from the cache by key with automatic expiration checking.
//
// This method performs atomic read-lock access to retrieve cached data. If the entry
// has expired, it's automatically removed and counted as a cache miss.
//
// Parameters:
//   - key: Cache key string (use GenerateKey() for consistent key generation)
//
// Returns:
//   - interface{}: Cached data if found and not expired
//   - bool: true if entry exists and is valid, false otherwise
//
// Behavior:
//   - Returns (nil, false) if key doesn't exist
//   - Returns (nil, false) if entry has expired (entry is deleted)
//   - Returns (data, true) if entry is valid
//
// Statistics:
//   - Increments Hits counter on successful retrieval
//   - Increments Misses counter on miss or expiration
//   - Increments Evictions counter when removing expired entry
//
// Thread Safety: Uses RLock for concurrent read access, upgrades to Lock for deletion.
//
// Example:
//
//	if data, ok := cache.Get("analytics:stats"); ok {
//	    return data.(*models.Stats), nil
//	}
//	// Cache miss, fetch from database
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		c.recordMiss()
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		// Entry expired, remove it
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		c.recordMiss()
		c.recordEviction()
		return nil, false
	}

	c.recordHit()
	return entry.Data, true
}

// Set stores a value in the cache with the default TTL configured at cache creation.
//
// This is a convenience method that wraps SetWithTTL using the cache's default TTL.
// The entry will expire after the configured duration (e.g., 5 minutes).
//
// Parameters:
//   - key: Cache key string (use GenerateKey() for consistent keys)
//   - value: Data to cache (any type, typically JSON-serializable structs)
//
// Behavior:
//   - Overwrites existing entry with same key
//   - Sets expiration to now + default TTL
//   - Updates TotalKeys statistic
//
// Thread Safety: Uses write lock for safe concurrent access.
//
// Example:
//
//	cache.Set("analytics:stats", stats)
//	cache.Set("user:alice", userData)
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in the cache with a custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = Entry{
		Data:      value,
		ExpiresAt: time.Now().Add(ttl),
	}

	c.stats.mu.Lock()
	c.stats.TotalKeys = int64(len(c.entries))
	c.stats.mu.Unlock()
}

// Delete removes a specific cache entry by key.
//
// This method performs immediate removal of a cache entry, incrementing the
// Evictions counter. It's typically used for manual cache invalidation.
//
// Parameters:
//   - key: Cache key to remove
//
// Behavior:
//   - No-op if key doesn't exist (safe to call with non-existent keys)
//   - Increments Evictions counter regardless of existence
//   - Does NOT decrement TotalKeys counter (updated on next cleanup)
//
// Thread Safety: Uses write lock for safe concurrent access.
//
// Example:
//
//	// Invalidate specific cache entry after data update
//	cache.Delete("analytics:stats")
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()

	c.recordEviction()
}

// Clear removes all entries from the cache in a single atomic operation.
//
// This method performs complete cache invalidation, typically called after data
// synchronization completes to ensure clients receive fresh data.
//
// Behavior:
//   - Removes all cache entries immediately
//   - Increments Evictions counter by number of entries removed
//   - Resets TotalKeys counter to 0
//   - Creates new empty map (old map eligible for garbage collection)
//
// Thread Safety: Uses write lock for safe concurrent access.
//
// Performance: O(1) operation (map replacement, not per-entry deletion).
//
// Example:
//
//	// Invalidate all cached analytics after sync
//	handler.cache.Clear()
//	log.Println("Analytics cache cleared after sync")
func (c *Cache) Clear() {
	c.mu.Lock()
	evictions := int64(len(c.entries))
	c.entries = make(map[string]Entry)
	c.mu.Unlock()

	c.stats.mu.Lock()
	c.stats.Evictions += evictions
	c.stats.TotalKeys = 0
	c.stats.mu.Unlock()
}

// GetStats returns a snapshot of current cache performance statistics.
//
// This method provides read-only access to cache metrics for monitoring and debugging.
// The returned Stats struct is a copy, safe to read without holding locks.
//
// Returns:
//   - Stats struct with current values:
//   - Hits: Number of successful cache retrievals
//   - Misses: Number of cache misses (key not found or expired)
//   - Evictions: Number of entries removed (manual + automatic)
//   - TotalKeys: Current number of entries in cache
//   - LastCleanup: Timestamp of most recent background cleanup
//
// Thread Safety: Uses read lock, returns copy of stats.
//
// Derived Metrics:
//   - Use HitRate() method for hit percentage calculation
//   - Hit Rate = Hits / (Hits + Misses) * 100
//
// Example:
//
//	stats := cache.GetStats()
//	log.Printf("Cache: %d keys, %.2f%% hit rate, %d evictions",
//	    stats.TotalKeys, cache.HitRate(), stats.Evictions)
func (c *Cache) GetStats() Stats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	return Stats{
		Hits:        c.stats.Hits,
		Misses:      c.stats.Misses,
		Evictions:   c.stats.Evictions,
		TotalKeys:   c.stats.TotalKeys,
		LastCleanup: c.stats.LastCleanup,
	}
}

// HitRate returns the cache hit rate as a percentage
func (c *Cache) HitRate() float64 {
	stats := c.GetStats()
	total := stats.Hits + stats.Misses
	if total == 0 {
		return 0.0
	}
	return float64(stats.Hits) / float64(total) * 100.0
}

// cleanupLoop periodically removes expired entries
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes all expired entries
func (c *Cache) cleanup() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	evictions := int64(0)
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
			evictions++
		}
	}

	c.stats.mu.Lock()
	c.stats.Evictions += evictions
	c.stats.TotalKeys = int64(len(c.entries))
	c.stats.LastCleanup = now
	c.stats.mu.Unlock()
}

// recordHit increments the hit counter
func (c *Cache) recordHit() {
	c.stats.mu.Lock()
	c.stats.Hits++
	c.stats.mu.Unlock()
}

// recordMiss increments the miss counter
func (c *Cache) recordMiss() {
	c.stats.mu.Lock()
	c.stats.Misses++
	c.stats.mu.Unlock()
}

// recordEviction increments the eviction counter
func (c *Cache) recordEviction() {
	c.stats.mu.Lock()
	c.stats.Evictions++
	c.stats.mu.Unlock()
}

// GenerateKey creates a cache key from the method name and parameters
func GenerateKey(method string, params interface{}) string {
	// Serialize parameters to JSON
	data, err := json.Marshal(params)
	if err != nil {
		// Fallback to simple string key
		return fmt.Sprintf("%s:%v", method, params)
	}

	// Hash the JSON data for a compact key
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%s:%x", method, hash[:16])
}
