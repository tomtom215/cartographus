// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package cache provides high-performance data structures for caching and deduplication.
// These implementations are optimized for the specific access patterns used in Cartographus.
package cache

import (
	"sync"
	"time"
)

// LRUEntry represents an entry in the LRU cache with TTL support.
type LRUEntry struct {
	key       string
	value     time.Time
	prev      *LRUEntry
	next      *LRUEntry
	expiresAt time.Time
}

// LRUCache implements a thread-safe Least Recently Used cache with TTL support.
// It provides O(1) operations for Get, Add, and eviction (vs O(n) for map-based eviction).
//
// Key features:
//   - O(1) Get, Add, Remove operations
//   - O(1) LRU eviction when capacity is reached
//   - TTL support with lazy expiration
//   - Thread-safe operations
//
// This implementation uses a doubly-linked list for ordering and a hashmap for lookups,
// following the pattern from TheAlgorithms/Go LRU implementation.
type LRUCache struct {
	mu sync.RWMutex

	// capacity is the maximum number of entries
	capacity int

	// ttl is the time-to-live for entries
	ttl time.Duration

	// items maps keys to linked list nodes for O(1) lookup
	items map[string]*LRUEntry

	// head and tail are sentinel nodes for the doubly-linked list
	// head.next is the most recently used, tail.prev is the least recently used
	head *LRUEntry
	tail *LRUEntry

	// stats
	hits   int64
	misses int64
}

// NewLRUCache creates a new LRU cache with the specified capacity and TTL.
func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	if capacity <= 0 {
		capacity = 10000 // Default capacity
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute // Default TTL
	}

	c := &LRUCache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*LRUEntry, capacity),
		head:     &LRUEntry{},
		tail:     &LRUEntry{},
	}

	// Initialize linked list sentinels
	c.head.next = c.tail
	c.tail.prev = c.head

	return c
}

// Get retrieves an entry from the cache.
// Returns the timestamp and true if found and not expired, false otherwise.
// Found entries are moved to the front (most recently used).
func (c *LRUCache) Get(key string) (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.items[key]; exists {
		// Check if expired
		if time.Now().After(entry.expiresAt) {
			c.removeEntry(entry)
			c.misses++
			return time.Time{}, false
		}

		// Move to front (most recently used)
		c.moveToFront(entry)
		c.hits++
		return entry.value, true
	}

	c.misses++
	return time.Time{}, false
}

// Contains checks if a key exists in the cache without updating access order.
func (c *LRUCache) Contains(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if entry, exists := c.items[key]; exists {
		return !time.Now().After(entry.expiresAt)
	}
	return false
}

// Add adds or updates an entry in the cache.
// If the cache is at capacity, the least recently used entry is evicted.
func (c *LRUCache) Add(key string, value time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiresAt := now.Add(c.ttl)

	// Check if key already exists
	if entry, exists := c.items[key]; exists {
		entry.value = value
		entry.expiresAt = expiresAt
		c.moveToFront(entry)
		return
	}

	// Create new entry
	entry := &LRUEntry{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
	}

	// Add to front of list and map
	c.addToFront(entry)
	c.items[key] = entry

	// Evict if over capacity
	for len(c.items) > c.capacity {
		c.evictOldest()
	}
}

// Remove removes an entry from the cache.
// Returns true if the entry was found and removed.
func (c *LRUCache) Remove(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.items[key]; exists {
		c.removeEntry(entry)
		return true
	}
	return false
}

// IsDuplicate checks if a key exists and is not expired.
// If not a duplicate, records the key with current timestamp.
// This is a convenience method for deduplication use cases.
func (c *LRUCache) IsDuplicate(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	if entry, exists := c.items[key]; exists {
		if !now.After(entry.expiresAt) {
			// Not expired, this is a duplicate
			c.moveToFront(entry)
			c.hits++
			return true
		}
		// Expired, remove and treat as new
		c.removeEntry(entry)
	}

	// Not a duplicate, record it
	expiresAt := now.Add(c.ttl)
	entry := &LRUEntry{
		key:       key,
		value:     now,
		expiresAt: expiresAt,
	}
	c.addToFront(entry)
	c.items[key] = entry

	// Evict if over capacity
	for len(c.items) > c.capacity {
		c.evictOldest()
	}

	c.misses++
	return false
}

// Len returns the current number of entries in the cache.
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all entries from the cache.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*LRUEntry, c.capacity)
	c.head.next = c.tail
	c.tail.prev = c.head
}

// CleanupExpired removes all expired entries from the cache.
// Returns the number of entries removed.
func (c *LRUCache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	// Walk from tail (oldest) to head (newest)
	for entry := c.tail.prev; entry != c.head; {
		prev := entry.prev
		if now.After(entry.expiresAt) {
			c.removeEntry(entry)
			removed++
		}
		entry = prev
	}

	return removed
}

// Stats returns cache hit/miss statistics.
func (c *LRUCache) Stats() (hits, misses int64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, len(c.items)
}

// Internal methods (must be called with lock held)

// addToFront adds an entry to the front of the list (most recently used).
func (c *LRUCache) addToFront(entry *LRUEntry) {
	entry.prev = c.head
	entry.next = c.head.next
	c.head.next.prev = entry
	c.head.next = entry
}

// moveToFront moves an existing entry to the front of the list.
func (c *LRUCache) moveToFront(entry *LRUEntry) {
	// Remove from current position
	entry.prev.next = entry.next
	entry.next.prev = entry.prev

	// Add to front
	c.addToFront(entry)
}

// removeEntry removes an entry from both the list and the map.
func (c *LRUCache) removeEntry(entry *LRUEntry) {
	// Remove from list
	entry.prev.next = entry.next
	entry.next.prev = entry.prev

	// Remove from map
	delete(c.items, entry.key)
}

// evictOldest removes the least recently used entry.
func (c *LRUCache) evictOldest() {
	oldest := c.tail.prev
	if oldest == c.head {
		return // List is empty
	}
	c.removeEntry(oldest)
}
