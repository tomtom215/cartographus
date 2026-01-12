// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package cache

import (
	"sync"
	"time"
)

// SlidingWindowCounter implements a memory-efficient sliding window counter.
// It divides time into buckets and sums them to get the count within the window.
//
// This is useful for:
//   - Rate limiting (e.g., requests per minute)
//   - Detection rules (e.g., events per hour for a user)
//   - Real-time metrics without database queries
//
// Complexity:
//   - Increment: O(1)
//   - Count: O(k) where k = number of buckets (typically 10-60)
//   - Memory: O(k) per counter
type SlidingWindowCounter struct {
	mu         sync.Mutex
	buckets    []int64       // circular buffer of bucket counts
	bucketSize time.Duration // duration of each bucket
	windowSize time.Duration // total window duration
	numBuckets int           // number of buckets
	current    int           // current bucket index
	lastUpdate time.Time     // last update time
}

// NewSlidingWindowCounter creates a new sliding window counter.
// The window is divided into the specified number of buckets.
//
// Parameters:
//   - windowSize: total duration of the sliding window (e.g., 5 minutes)
//   - numBuckets: number of buckets to divide the window into (e.g., 10)
//
// Example: NewSlidingWindowCounter(5*time.Minute, 10) creates a 5-minute window
// with 30-second buckets.
func NewSlidingWindowCounter(windowSize time.Duration, numBuckets int) *SlidingWindowCounter {
	if numBuckets <= 0 {
		numBuckets = 10
	}
	if windowSize <= 0 {
		windowSize = 5 * time.Minute
	}

	return &SlidingWindowCounter{
		buckets:    make([]int64, numBuckets),
		bucketSize: windowSize / time.Duration(numBuckets),
		windowSize: windowSize,
		numBuckets: numBuckets,
		current:    0,
		lastUpdate: time.Now(),
	}
}

// Increment adds delta to the current bucket.
func (sw *SlidingWindowCounter) Increment(delta int64) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.advance()
	sw.buckets[sw.current] += delta
}

// IncrementOne adds 1 to the current bucket.
func (sw *SlidingWindowCounter) IncrementOne() {
	sw.Increment(1)
}

// Count returns the sum of all buckets in the window.
func (sw *SlidingWindowCounter) Count() int64 {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.advance()

	var total int64
	for _, count := range sw.buckets {
		total += count
	}
	return total
}

// Reset clears all buckets.
func (sw *SlidingWindowCounter) Reset() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	for i := range sw.buckets {
		sw.buckets[i] = 0
	}
	sw.current = 0
	sw.lastUpdate = time.Now()
}

// advance moves the window forward based on elapsed time.
// Must be called with lock held.
func (sw *SlidingWindowCounter) advance() {
	now := time.Now()
	elapsed := now.Sub(sw.lastUpdate)

	// Calculate how many buckets have elapsed
	bucketsElapsed := int(elapsed / sw.bucketSize)

	if bucketsElapsed <= 0 {
		return
	}

	// Clear buckets that have expired
	if bucketsElapsed >= sw.numBuckets {
		// Entire window has elapsed, clear all
		for i := range sw.buckets {
			sw.buckets[i] = 0
		}
		sw.current = 0
	} else {
		// Clear only the elapsed buckets
		for i := 0; i < bucketsElapsed; i++ {
			sw.current = (sw.current + 1) % sw.numBuckets
			sw.buckets[sw.current] = 0
		}
	}

	sw.lastUpdate = now
}

// SlidingWindowStore manages multiple sliding window counters by key.
// This is useful for tracking per-user or per-device metrics.
//
// Example usage:
//
//	store := NewSlidingWindowStore(5*time.Minute, 10)
//	store.Increment("user:123")
//	count := store.Count("user:123")
type SlidingWindowStore struct {
	mu         sync.RWMutex
	counters   map[string]*SlidingWindowCounter
	windowSize time.Duration
	numBuckets int
	maxKeys    int // maximum number of keys (0 = unlimited)
}

// NewSlidingWindowStore creates a new store for sliding window counters.
func NewSlidingWindowStore(windowSize time.Duration, numBuckets, maxKeys int) *SlidingWindowStore {
	return &SlidingWindowStore{
		counters:   make(map[string]*SlidingWindowCounter),
		windowSize: windowSize,
		numBuckets: numBuckets,
		maxKeys:    maxKeys,
	}
}

// Increment adds 1 to the counter for the given key.
func (s *SlidingWindowStore) Increment(key string) {
	s.IncrementBy(key, 1)
}

// IncrementBy adds delta to the counter for the given key.
func (s *SlidingWindowStore) IncrementBy(key string, delta int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	counter, exists := s.counters[key]
	if !exists {
		// Evict oldest if at capacity
		if s.maxKeys > 0 && len(s.counters) >= s.maxKeys {
			s.evictOldest()
		}

		counter = NewSlidingWindowCounter(s.windowSize, s.numBuckets)
		s.counters[key] = counter
	}

	counter.Increment(delta)
}

// Count returns the count for the given key within the window.
func (s *SlidingWindowStore) Count(key string) int64 {
	s.mu.RLock()
	counter, exists := s.counters[key]
	s.mu.RUnlock()

	if !exists {
		return 0
	}
	return counter.Count()
}

// Remove removes the counter for the given key.
func (s *SlidingWindowStore) Remove(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.counters, key)
}

// Keys returns all keys in the store.
func (s *SlidingWindowStore) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.counters))
	for key := range s.counters {
		keys = append(keys, key)
	}
	return keys
}

// Len returns the number of counters in the store.
func (s *SlidingWindowStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.counters)
}

// Clear removes all counters from the store.
func (s *SlidingWindowStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters = make(map[string]*SlidingWindowCounter)
}

// CleanupInactive removes counters that have no counts in the window.
// Returns the number of counters removed.
func (s *SlidingWindowStore) CleanupInactive() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for key, counter := range s.counters {
		if counter.Count() == 0 {
			delete(s.counters, key)
			removed++
		}
	}
	return removed
}

// evictOldest removes a random counter when at capacity.
// Must be called with lock held.
func (s *SlidingWindowStore) evictOldest() {
	// Simple eviction: remove the first key found (random due to map iteration)
	for key := range s.counters {
		delete(s.counters, key)
		return
	}
}

// UniqueValueCounter tracks unique values within a sliding window.
// This is useful for counting unique IPs, devices, or users within a time period.
type UniqueValueCounter struct {
	mu         sync.Mutex
	buckets    []map[string]struct{} // circular buffer of value sets
	bucketSize time.Duration
	windowSize time.Duration
	numBuckets int
	current    int
	lastUpdate time.Time
}

// NewUniqueValueCounter creates a new unique value counter.
func NewUniqueValueCounter(windowSize time.Duration, numBuckets int) *UniqueValueCounter {
	if numBuckets <= 0 {
		numBuckets = 10
	}
	if windowSize <= 0 {
		windowSize = 5 * time.Minute
	}

	buckets := make([]map[string]struct{}, numBuckets)
	for i := range buckets {
		buckets[i] = make(map[string]struct{})
	}

	return &UniqueValueCounter{
		buckets:    buckets,
		bucketSize: windowSize / time.Duration(numBuckets),
		windowSize: windowSize,
		numBuckets: numBuckets,
		current:    0,
		lastUpdate: time.Now(),
	}
}

// Add records a value in the current bucket.
func (u *UniqueValueCounter) Add(value string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.advance()
	u.buckets[u.current][value] = struct{}{}
}

// CountUnique returns the count of unique values across all buckets.
func (u *UniqueValueCounter) CountUnique() int {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.advance()

	// Merge all buckets
	merged := make(map[string]struct{})
	for _, bucket := range u.buckets {
		for value := range bucket {
			merged[value] = struct{}{}
		}
	}
	return len(merged)
}

// GetUnique returns all unique values across all buckets.
func (u *UniqueValueCounter) GetUnique() []string {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.advance()

	// Merge all buckets
	merged := make(map[string]struct{})
	for _, bucket := range u.buckets {
		for value := range bucket {
			merged[value] = struct{}{}
		}
	}

	values := make([]string, 0, len(merged))
	for value := range merged {
		values = append(values, value)
	}
	return values
}

// Reset clears all buckets.
func (u *UniqueValueCounter) Reset() {
	u.mu.Lock()
	defer u.mu.Unlock()

	for i := range u.buckets {
		u.buckets[i] = make(map[string]struct{})
	}
	u.current = 0
	u.lastUpdate = time.Now()
}

// advance moves the window forward based on elapsed time.
func (u *UniqueValueCounter) advance() {
	now := time.Now()
	elapsed := now.Sub(u.lastUpdate)
	bucketsElapsed := int(elapsed / u.bucketSize)

	if bucketsElapsed <= 0 {
		return
	}

	if bucketsElapsed >= u.numBuckets {
		// Entire window has elapsed, clear all
		for i := range u.buckets {
			u.buckets[i] = make(map[string]struct{})
		}
		u.current = 0
	} else {
		// Clear only the elapsed buckets
		for i := 0; i < bucketsElapsed; i++ {
			u.current = (u.current + 1) % u.numBuckets
			u.buckets[u.current] = make(map[string]struct{})
		}
	}

	u.lastUpdate = now
}

// UniqueValueStore manages multiple unique value counters by key.
type UniqueValueStore struct {
	mu         sync.RWMutex
	counters   map[string]*UniqueValueCounter
	windowSize time.Duration
	numBuckets int
	maxKeys    int
}

// NewUniqueValueStore creates a new store for unique value counters.
func NewUniqueValueStore(windowSize time.Duration, numBuckets, maxKeys int) *UniqueValueStore {
	return &UniqueValueStore{
		counters:   make(map[string]*UniqueValueCounter),
		windowSize: windowSize,
		numBuckets: numBuckets,
		maxKeys:    maxKeys,
	}
}

// Add records a value for the given key.
func (s *UniqueValueStore) Add(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	counter, exists := s.counters[key]
	if !exists {
		if s.maxKeys > 0 && len(s.counters) >= s.maxKeys {
			s.evictOldest()
		}
		counter = NewUniqueValueCounter(s.windowSize, s.numBuckets)
		s.counters[key] = counter
	}

	counter.Add(value)
}

// CountUnique returns the count of unique values for the given key.
func (s *UniqueValueStore) CountUnique(key string) int {
	s.mu.RLock()
	counter, exists := s.counters[key]
	s.mu.RUnlock()

	if !exists {
		return 0
	}
	return counter.CountUnique()
}

// GetUnique returns all unique values for the given key.
func (s *UniqueValueStore) GetUnique(key string) []string {
	s.mu.RLock()
	counter, exists := s.counters[key]
	s.mu.RUnlock()

	if !exists {
		return nil
	}
	return counter.GetUnique()
}

// Remove removes the counter for the given key.
func (s *UniqueValueStore) Remove(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.counters, key)
}

// Len returns the number of counters in the store.
func (s *UniqueValueStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.counters)
}

// Clear removes all counters from the store.
func (s *UniqueValueStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters = make(map[string]*UniqueValueCounter)
}

// evictOldest removes a random counter when at capacity.
func (s *UniqueValueStore) evictOldest() {
	for key := range s.counters {
		delete(s.counters, key)
		return
	}
}
