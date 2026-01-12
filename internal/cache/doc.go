// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package cache provides thread-safe in-memory caching with TTL support.

This package implements a simple but effective caching layer for API responses
and analytics results, reducing database load and improving response times for
frequently accessed data.

# Overview

The cache provides:
  - Thread-safe concurrent access (sync.RWMutex)
  - Time-to-live (TTL) expiration for automatic cleanup
  - Simple key-value storage with any value type (interface{})
  - Lazy expiration checking (on Get operations)
  - Zero external dependencies (stdlib only)

# Use Cases

Primary use cases:
  - Analytics API responses (5-minute TTL)
  - Aggregated statistics (5-minute TTL)
  - Geolocation lookups (permanent until sync)
  - User lists and media type filters (10-minute TTL)
  - Frequently accessed playback data (1-minute TTL)

# Cache Structure

The cache stores items with metadata:

	type Item struct {
	    Value      interface{}  // Cached value (any type)
	    Expiration int64        // Unix timestamp for expiration
	}

# Usage Example

Basic caching:

	import "github.com/tomtom215/cartographus/internal/cache"

	// Create cache with 5-minute default TTL
	c := cache.New(5 * time.Minute)

	// Store value
	c.Set("stats:global", stats)

	// Retrieve value
	if value, ok := c.Get("stats:global"); ok {
	    stats := value.(Stats)
	    // Use cached stats
	}

	// Delete specific key
	c.Delete("stats:global")

	// Clear entire cache
	c.Clear()

API handler caching pattern:

	func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	    cacheKey := "api:stats:v1"

	    // Check cache
	    if cached, ok := h.cache.Get(cacheKey); ok {
	        h.writeJSON(w, http.StatusOK, cached)
	        return
	    }

	    // Cache miss - fetch from database
	    stats, err := h.db.GetStats(r.Context())
	    if err != nil {
	        h.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error())
	        return
	    }

	    // Store in cache
	    h.cache.Set(cacheKey, stats)

	    // Return response
	    h.writeJSON(w, http.StatusOK, stats)
	}

Parameterized cache keys:

	// Build cache key from filter parameters
	func buildCacheKey(endpoint string, filter LocationStatsFilter) string {
	    return fmt.Sprintf("%s:start=%s:end=%s:users=%v:types=%v",
	        endpoint,
	        filter.StartDate.Format("2006-01-02"),
	        filter.EndDate.Format("2006-01-02"),
	        strings.Join(filter.Users, ","),
	        strings.Join(filter.MediaTypes, ","),
	    )
	}

	cacheKey := buildCacheKey("analytics:trends", filter)
	if cached, ok := cache.Get(cacheKey); ok {
	    return cached.(TrendsData), nil
	}

# Cache Invalidation

The cache supports two invalidation strategies:

1. TTL-based expiration (automatic):
  - Items expire after the configured TTL
  - Checked lazily during Get operations
  - No background cleanup goroutine needed

2. Manual invalidation (on data changes):
  - Clear() removes all cache entries
  - Delete(key) removes specific entry
  - OnSyncCompleted triggers full cache clear

Example: Clear cache after sync

	// In sync manager
	func (m *Manager) OnSyncCompleted() {
	    // Clear analytics cache since data changed
	    m.cache.Clear()

	    // Notify frontend
	    m.wsHub.BroadcastSyncCompleted(newRecords, duration)
	}

# Cache Key Conventions

Use consistent key prefixes for organization:

	api:stats:v1                          // Global stats
	api:playbacks:start=2025-01-01:...    // Filtered playbacks
	analytics:trends:...                  // Analytics results
	analytics:distribution:codecs         // Distribution data
	geo:ip=1.2.3.4                       // Geolocation cache
	users:list                           // Available users
	mediatypes:list                      // Available media types

# Performance Characteristics

  - Get operation: O(1) hash map lookup + TTL check (~100ns)
  - Set operation: O(1) hash map insert with lock (~200ns)
  - Delete operation: O(1) hash map delete with lock (~150ns)
  - Clear operation: O(1) map reassignment (~50ns)
  - Memory overhead: ~100 bytes per cached item (key + metadata)

# Memory Management

Cache memory grows with stored items:

	Estimated memory per item:
	  - Key string: len(key) bytes
	  - Item metadata: ~48 bytes (struct overhead)
	  - Value data: depends on cached type
	  - Total: ~100 bytes + value size

	Example with 1000 cached analytics results:
	  - 1000 items × (100 bytes + 5KB per result)
	  - = ~5MB cache memory usage

# Thread Safety

All cache methods are thread-safe using sync.RWMutex:

  - Get: Acquires read lock (concurrent reads allowed)
  - Set: Acquires write lock (exclusive access)
  - Delete: Acquires write lock (exclusive access)
  - Clear: Acquires write lock (exclusive access)

Multiple goroutines can safely access the cache concurrently.

# TTL Configuration

Recommended TTL values by use case:

	Analytics endpoints: 5 minutes
	  - Balances freshness and performance
	  - Automatically invalidated on sync

	Statistics endpoints: 5 minutes
	  - Global stats don't change frequently
	  - Reduces database load significantly

	Geolocation data: No expiration
	  - IP geolocation is static
	  - Manually cleared on sync

	User/media type lists: 10 minutes
	  - Changes infrequently
	  - Small memory footprint

	Real-time data: 30 seconds
	  - Current active sessions
	  - Buffer health status

# Cache Hit Rate

Monitor cache effectiveness:

	hits := cacheHits.Load()
	misses := cacheMisses.Load()
	hitRate := float64(hits) / float64(hits + misses)

	if hitRate < 0.5 {
	    // Cache hit rate too low
	    // Consider increasing TTL or reviewing cache keys
	}

Target hit rates:
  - Analytics endpoints: 80-90% (high query cost)
  - Statistics endpoints: 70-80% (moderate query cost)
  - Real-time data: 40-60% (low query cost, high churn)

# Limitations

The current implementation has intentional limitations for simplicity:

  - No maximum cache size limit (grows unbounded)
  - No LRU eviction policy (only TTL-based)
  - No background cleanup (lazy expiration)
  - No cache persistence (in-memory only)
  - No distributed caching (single instance)

These limitations are acceptable for the application's scale:
  - Small dataset (10k-100k playbacks)
  - Single instance deployment
  - Predictable cache size (~10-50MB)
  - Automatic clearing on sync

# Future Enhancements

Potential improvements for larger scale:

 1. LRU eviction: Add size limit with least-recently-used eviction
 2. Background cleanup: Periodic goroutine to remove expired items
 3. Cache metrics: Track hit/miss rates, size, eviction counts
 4. Distributed cache: Redis integration for multi-instance deployments
 5. Cache warming: Pre-populate cache on startup
 6. Compression: Compress large cached values to save memory

# Testing

The package includes comprehensive tests:
  - Basic operations (Get, Set, Delete, Clear)
  - TTL expiration behavior
  - Concurrent access with race detector
  - Thread safety validation
  - Memory usage benchmarks

Test coverage: 98.7% (as of 2025-11-23)

Run tests with race detector:

	go test -race ./internal/cache

# Example: Analytics Cache

Full example with cache wrapper:

	type AnalyticsCache struct {
	    cache *cache.Cache
	    db    *database.Database
	}

	func (ac *AnalyticsCache) GetTrends(ctx context.Context, filter LocationStatsFilter) (TrendsData, error) {
	    cacheKey := buildTrendsCacheKey(filter)

	    // Check cache
	    if cached, ok := ac.cache.Get(cacheKey); ok {
	        return cached.(TrendsData), nil
	    }

	    // Cache miss - query database
	    trends, err := ac.db.GetPlaybackTrends(ctx, filter)
	    if err != nil {
	        return TrendsData{}, err
	    }

	    // Store in cache
	    ac.cache.Set(cacheKey, trends)

	    return trends, nil
	}

	func (ac *AnalyticsCache) Invalidate() {
	    ac.cache.Clear()
	}

# See Also

  - internal/api: API handlers that use caching
  - internal/middleware: HTTP middleware integration
  - internal/database: Database layer cached by this package
*/
package cache
