// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package cache provides high-performance data structures for caching and deduplication.
package cache

import "time"

// Cacher defines the interface for cache implementations.
// Both Cache (TTL-based) and LFUCache implement this interface,
// allowing for easy switching between caching strategies.
//
// Usage:
//
//	// Use standard TTL cache
//	var c Cacher = NewTTL(5 * time.Minute)
//
//	// Or use LFU cache for better hit rates with skewed access patterns
//	var c Cacher = NewLFU(10000, 5 * time.Minute)
//
//	c.Set("key", value)
//	if val, ok := c.Get("key"); ok {
//	    // Use cached value
//	}
type Cacher interface {
	// Get retrieves a value from the cache.
	// Returns the value and true if found and not expired.
	Get(key string) (interface{}, bool)

	// Set stores a value in the cache with the default TTL.
	Set(key string, value interface{})

	// SetWithTTL stores a value with a custom TTL.
	SetWithTTL(key string, value interface{}, ttl time.Duration)

	// Delete removes a value from the cache.
	Delete(key string)

	// Clear removes all entries from the cache.
	Clear()

	// GetStats returns cache statistics.
	GetStats() Stats

	// HitRate returns the cache hit rate as a percentage.
	HitRate() float64
}

// CacheType represents the type of cache to create.
type CacheType string

const (
	// CacheTypeTTL is a simple TTL-based cache (default).
	// Best for: General purpose caching, when access patterns are uniform.
	CacheTypeTTL CacheType = "ttl"

	// CacheTypeLFU is a Least Frequently Used cache.
	// Best for: Analytics queries with highly skewed access patterns (80/20 rule).
	// Provides 40-60% better hit rates than TTL for frequently accessed data.
	CacheTypeLFU CacheType = "lfu"
)

// CacheConfig holds configuration for creating a cache.
type CacheConfig struct {
	// Type specifies the cache implementation (ttl or lfu)
	Type CacheType

	// TTL is the default time-to-live for cache entries
	TTL time.Duration

	// Capacity is the maximum number of entries (only used for LFU)
	// Default: 10000 for LFU, unlimited for TTL
	Capacity int
}

// NewCacher creates a cache based on the configuration.
// This factory function allows easy switching between cache strategies.
//
// Example:
//
//	// Create TTL cache (default behavior)
//	cache := NewCacher(CacheConfig{Type: CacheTypeTTL, TTL: 5 * time.Minute})
//
//	// Create LFU cache for analytics
//	cache := NewCacher(CacheConfig{Type: CacheTypeLFU, TTL: 5 * time.Minute, Capacity: 10000})
func NewCacher(cfg CacheConfig) Cacher {
	if cfg.TTL <= 0 {
		cfg.TTL = 5 * time.Minute
	}

	switch cfg.Type {
	case CacheTypeLFU:
		capacity := cfg.Capacity
		if capacity <= 0 {
			capacity = 10000
		}
		return &lfuCacheAdapter{LFUCache: NewLFUCache(capacity, cfg.TTL)}
	default:
		return New(cfg.TTL)
	}
}

// NewTTL creates a new TTL-based cache (same as New).
// Convenience function for explicit cache type selection.
func NewTTL(ttl time.Duration) Cacher {
	return New(ttl)
}

// NewLFU creates a new LFU cache.
// Convenience function for explicit cache type selection.
func NewLFU(capacity int, ttl time.Duration) Cacher {
	return &lfuCacheAdapter{LFUCache: NewLFUCache(capacity, ttl)}
}

// lfuCacheAdapter adapts LFUCache to implement the Cacher interface.
// This is needed because LFUCache has slightly different method signatures.
type lfuCacheAdapter struct {
	*LFUCache
}

// Delete implements Cacher.Delete for LFUCache.
func (a *lfuCacheAdapter) Delete(key string) {
	a.LFUCache.Delete(key)
}

// GetStats implements Cacher.GetStats for LFUCache.
func (a *lfuCacheAdapter) GetStats() Stats {
	hits, misses, size := a.Stats()
	return Stats{
		Hits:      hits,
		Misses:    misses,
		TotalKeys: int64(size),
	}
}

// Verify interface implementations at compile time
var (
	_ Cacher = (*Cache)(nil)
	_ Cacher = (*lfuCacheAdapter)(nil)
)
