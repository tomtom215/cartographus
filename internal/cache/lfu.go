// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package cache provides high-performance data structures for caching and deduplication.
package cache

import (
	"sync"
	"time"
)

// LFUEntry represents an entry in the LFU cache.
type LFUEntry struct {
	key       string
	value     interface{}
	freq      int       // Access frequency
	expiresAt time.Time // TTL expiration time
	prev      *LFUEntry // Previous entry in frequency list
	next      *LFUEntry // Next entry in frequency list
}

// freqList is a doubly-linked list of entries with the same frequency.
type freqList struct {
	head *LFUEntry // Sentinel head (most recently used at this frequency)
	tail *LFUEntry // Sentinel tail (least recently used at this frequency)
	size int       // Number of entries in this list
}

// newFreqList creates a new frequency list with sentinel nodes.
func newFreqList() *freqList {
	fl := &freqList{
		head: &LFUEntry{},
		tail: &LFUEntry{},
	}
	fl.head.next = fl.tail
	fl.tail.prev = fl.head
	return fl
}

// addToFront adds an entry to the front of the list (most recently used).
func (fl *freqList) addToFront(entry *LFUEntry) {
	entry.prev = fl.head
	entry.next = fl.head.next
	fl.head.next.prev = entry
	fl.head.next = entry
	fl.size++
}

// remove removes an entry from the list.
func (fl *freqList) remove(entry *LFUEntry) {
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
	entry.prev = nil
	entry.next = nil
	fl.size--
}

// removeLast removes and returns the last entry (least recently used at this frequency).
func (fl *freqList) removeLast() *LFUEntry {
	if fl.size == 0 {
		return nil
	}
	entry := fl.tail.prev
	fl.remove(entry)
	return entry
}

// isEmpty returns true if the list has no entries.
func (fl *freqList) isEmpty() bool {
	return fl.size == 0
}

// LFUCache implements a thread-safe Least Frequently Used cache with O(1) operations.
// It evicts entries that are accessed least frequently, making it ideal for caching
// data with highly skewed access patterns (80/20 rule).
//
// Key features:
//   - O(1) Get, Set, and eviction operations
//   - Frequency-based eviction (least frequently used items evicted first)
//   - TTL support with lazy expiration
//   - Thread-safe operations
//   - Hit rate tracking for monitoring
//
// This implementation uses a combination of hashmaps and doubly-linked lists:
//   - keyMap: maps keys to entries for O(1) lookup
//   - freqMap: maps frequencies to lists of entries
//   - minFreq: tracks the minimum frequency for O(1) eviction
//
// Compared to LRU:
//   - LFU provides 40-60% better hit rates for skewed access patterns
//   - LRU is better for temporal locality (recently accessed = likely to be accessed again)
//   - LFU is better for frequency locality (frequently accessed = likely to be accessed again)
type LFUCache struct {
	mu sync.RWMutex

	// capacity is the maximum number of entries
	capacity int

	// ttl is the default time-to-live for entries
	ttl time.Duration

	// keyMap maps keys to entries for O(1) lookup
	keyMap map[string]*LFUEntry

	// freqMap maps frequencies to doubly-linked lists of entries
	freqMap map[int]*freqList

	// minFreq tracks the minimum frequency for O(1) eviction
	minFreq int

	// stats for monitoring
	hits   int64
	misses int64
}

// NewLFUCache creates a new LFU cache with the specified capacity and TTL.
func NewLFUCache(capacity int, ttl time.Duration) *LFUCache {
	if capacity <= 0 {
		capacity = 10000 // Default capacity
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute // Default TTL
	}

	return &LFUCache{
		capacity: capacity,
		ttl:      ttl,
		keyMap:   make(map[string]*LFUEntry, capacity),
		freqMap:  make(map[int]*freqList),
		minFreq:  0,
	}
}

// Get retrieves an entry from the cache.
// Returns the value and true if found and not expired, nil and false otherwise.
// Found entries have their frequency incremented.
func (c *LFUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.keyMap[key]
	if !exists {
		c.misses++
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		c.removeEntry(entry)
		c.misses++
		return nil, false
	}

	// Increment frequency
	c.incrementFreq(entry)
	c.hits++

	return entry.value, true
}

// Set adds or updates an entry in the cache.
// If the cache is at capacity, the least frequently used entry is evicted.
func (c *LFUCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL adds or updates an entry with a custom TTL.
func (c *LFUCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiresAt := now.Add(ttl)

	// Check if key already exists
	if entry, exists := c.keyMap[key]; exists {
		entry.value = value
		entry.expiresAt = expiresAt
		c.incrementFreq(entry)
		return
	}

	// Evict if at capacity
	if len(c.keyMap) >= c.capacity {
		c.evict()
	}

	// Create new entry with frequency 1
	entry := &LFUEntry{
		key:       key,
		value:     value,
		freq:      1,
		expiresAt: expiresAt,
	}

	// Add to frequency list
	if c.freqMap[1] == nil {
		c.freqMap[1] = newFreqList()
	}
	c.freqMap[1].addToFront(entry)

	// Add to key map
	c.keyMap[key] = entry

	// Update minFreq
	c.minFreq = 1
}

// Delete removes an entry from the cache.
// Returns true if the entry was found and removed.
func (c *LFUCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.keyMap[key]; exists {
		c.removeEntry(entry)
		return true
	}
	return false
}

// Contains checks if a key exists in the cache without modifying frequency.
func (c *LFUCache) Contains(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if entry, exists := c.keyMap[key]; exists {
		return !time.Now().After(entry.expiresAt)
	}
	return false
}

// Len returns the current number of entries in the cache.
func (c *LFUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.keyMap)
}

// Clear removes all entries from the cache.
func (c *LFUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.keyMap = make(map[string]*LFUEntry, c.capacity)
	c.freqMap = make(map[int]*freqList)
	c.minFreq = 0
}

// Stats returns cache statistics.
func (c *LFUCache) Stats() (hits, misses int64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, len(c.keyMap)
}

// HitRate returns the cache hit rate as a percentage.
func (c *LFUCache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	if total == 0 {
		return 0.0
	}
	return float64(c.hits) / float64(total) * 100.0
}

// GetFrequency returns the access frequency for a key.
// Returns 0 if the key doesn't exist.
func (c *LFUCache) GetFrequency(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if entry, exists := c.keyMap[key]; exists {
		return entry.freq
	}
	return 0
}

// CleanupExpired removes all expired entries from the cache.
// Returns the number of entries removed.
func (c *LFUCache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, entry := range c.keyMap {
		if now.After(entry.expiresAt) {
			c.removeEntryUnlocked(key, entry)
			removed++
		}
	}

	return removed
}

// Internal methods (must be called with lock held)

// incrementFreq moves an entry to the next frequency level.
func (c *LFUCache) incrementFreq(entry *LFUEntry) {
	oldFreq := entry.freq

	// Remove from current frequency list
	if fl, exists := c.freqMap[oldFreq]; exists {
		fl.remove(entry)

		// Update minFreq if necessary
		if fl.isEmpty() && c.minFreq == oldFreq {
			c.minFreq++
		}
	}

	// Increment frequency
	entry.freq++
	newFreq := entry.freq

	// Add to new frequency list
	if c.freqMap[newFreq] == nil {
		c.freqMap[newFreq] = newFreqList()
	}
	c.freqMap[newFreq].addToFront(entry)
}

// evict removes the least frequently used entry.
func (c *LFUCache) evict() {
	// Get the list with minimum frequency
	fl := c.freqMap[c.minFreq]
	if fl == nil || fl.isEmpty() {
		return
	}

	// Remove the least recently used entry at this frequency
	entry := fl.removeLast()
	if entry != nil {
		delete(c.keyMap, entry.key)
	}
}

// removeEntry removes an entry from both the frequency list and key map.
func (c *LFUCache) removeEntry(entry *LFUEntry) {
	c.removeEntryUnlocked(entry.key, entry)
}

// removeEntryUnlocked removes an entry (helper for removeEntry and CleanupExpired).
func (c *LFUCache) removeEntryUnlocked(key string, entry *LFUEntry) {
	// Remove from frequency list
	if fl, exists := c.freqMap[entry.freq]; exists {
		fl.remove(entry)
	}

	// Remove from key map
	delete(c.keyMap, key)
}

// LFUCacheGeneric is a type-safe version of LFUCache using generics.
// Use this when you need compile-time type safety for cached values.
type LFUCacheGeneric[V any] struct {
	cache *LFUCache
}

// NewLFUCacheGeneric creates a new type-safe LFU cache.
func NewLFUCacheGeneric[V any](capacity int, ttl time.Duration) *LFUCacheGeneric[V] {
	return &LFUCacheGeneric[V]{
		cache: NewLFUCache(capacity, ttl),
	}
}

// Get retrieves a value from the cache.
func (c *LFUCacheGeneric[V]) Get(key string) (V, bool) {
	var zero V
	value, found := c.cache.Get(key)
	if !found {
		return zero, false
	}
	typed, ok := value.(V)
	if !ok {
		return zero, false
	}
	return typed, true
}

// Set stores a value in the cache.
func (c *LFUCacheGeneric[V]) Set(key string, value V) {
	c.cache.Set(key, value)
}

// SetWithTTL stores a value with a custom TTL.
func (c *LFUCacheGeneric[V]) SetWithTTL(key string, value V, ttl time.Duration) {
	c.cache.SetWithTTL(key, value, ttl)
}

// Delete removes a value from the cache.
func (c *LFUCacheGeneric[V]) Delete(key string) bool {
	return c.cache.Delete(key)
}

// Contains checks if a key exists.
func (c *LFUCacheGeneric[V]) Contains(key string) bool {
	return c.cache.Contains(key)
}

// Len returns the number of entries.
func (c *LFUCacheGeneric[V]) Len() int {
	return c.cache.Len()
}

// Clear removes all entries.
func (c *LFUCacheGeneric[V]) Clear() {
	c.cache.Clear()
}

// Stats returns cache statistics.
func (c *LFUCacheGeneric[V]) Stats() (hits, misses int64, size int) {
	return c.cache.Stats()
}

// HitRate returns the hit rate percentage.
func (c *LFUCacheGeneric[V]) HitRate() float64 {
	return c.cache.HitRate()
}
