// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package cache

import (
	"hash/fnv"
	"sync"
	"time"
)

// DeduplicationCache is the interface for event deduplication caches.
// Implementations may use different strategies (exact match, probabilistic, etc.)
// but must provide the same deduplication semantics.
//
// CRITICAL (v2.3): For zero data loss requirements, use ExactLRU (zero false positives).
// BloomLRU may be used when ~1% false positive rate is acceptable for better performance.
type DeduplicationCache interface {
	// IsDuplicate checks if a key has been seen before.
	// If not a duplicate, records the key for future checks.
	// Returns true if the key is a duplicate.
	IsDuplicate(key string) bool

	// Contains checks if a key exists without modifying the cache.
	Contains(key string) bool

	// Record records a key as seen without checking for duplicates.
	Record(key string)

	// CleanupExpired removes expired entries from the cache.
	// Returns the number of entries removed.
	CleanupExpired() int

	// Clear removes all entries from the cache.
	Clear()

	// Len returns the current number of entries in the cache.
	Len() int

	// Stats returns performance statistics.
	// Returns (bloomNegatives, lruChecks, duplicates, lruSize).
	Stats() (bloomNegatives, lruChecks, duplicates int64, lruSize int)
}

// Compile-time interface verification
var _ DeduplicationCache = (*BloomLRU)(nil)
var _ DeduplicationCache = (*ExactLRU)(nil)

// BloomFilter is a probabilistic data structure for set membership testing.
// It provides O(1) operations with configurable false positive rate.
//
// Key characteristics:
//   - No false negatives: if Test() returns false, the item definitely wasn't added
//   - Possible false positives: if Test() returns true, the item might have been added
//   - Space efficient: uses ~10 bits per element for 1% false positive rate
//   - Cannot remove items (use for caches that don't need deletion)
//
// Usage pattern for deduplication:
//
//	if !bloom.Test(key) {
//	    // Definitely not seen before
//	    return false
//	}
//	// Might have been seen, verify with LRU cache
//	return lru.Contains(key)
type BloomFilter struct {
	mu       sync.RWMutex
	bits     []uint64 // bit array
	size     uint64   // number of bits
	hashFns  int      // number of hash functions to use
	count    int      // number of items added
	capacity int      // expected capacity
}

// NewBloomFilter creates a new Bloom filter with the specified expected capacity
// and target false positive rate.
//
// Parameters:
//   - expectedItems: expected number of unique items to add
//   - falsePositiveRate: target false positive probability (e.g., 0.01 for 1%)
//
// Example: NewBloomFilter(10000, 0.01) creates a filter for 10k items with 1% FP rate.
func NewBloomFilter(expectedItems int, falsePositiveRate float64) *BloomFilter {
	if expectedItems <= 0 {
		expectedItems = 10000
	}
	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		falsePositiveRate = 0.01
	}

	// Calculate optimal size and hash functions
	// m = -n * ln(p) / (ln(2)^2) where m = bits, n = items, p = false positive rate
	// k = (m/n) * ln(2) where k = number of hash functions
	ln2 := 0.693147
	ln2Squared := ln2 * ln2

	// Natural log approximation for false positive rate
	lnP := approximateLn(falsePositiveRate)

	// Calculate optimal bit array size
	m := int(-float64(expectedItems) * lnP / ln2Squared)
	if m < 64 {
		m = 64
	}

	// Calculate optimal number of hash functions
	k := int(float64(m) / float64(expectedItems) * ln2)
	if k < 1 {
		k = 1
	}
	if k > 10 {
		k = 10 // Cap to prevent excessive hashing
	}

	// Round up to multiple of 64 for efficient storage
	words := (m + 63) / 64

	return &BloomFilter{
		bits:     make([]uint64, words),
		size:     uint64(words * 64),
		hashFns:  k,
		capacity: expectedItems,
	}
}

// Add adds an item to the Bloom filter.
func (bf *BloomFilter) Add(key string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	hashes := bf.getHashes(key)
	for _, h := range hashes {
		idx := h % bf.size
		bf.bits[idx/64] |= 1 << (idx % 64)
	}
	bf.count++
}

// Test checks if an item might be in the Bloom filter.
// Returns:
//   - false: item definitely NOT in the filter
//   - true: item might be in the filter (verify with authoritative source)
func (bf *BloomFilter) Test(key string) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	hashes := bf.getHashes(key)
	for _, h := range hashes {
		idx := h % bf.size
		if bf.bits[idx/64]&(1<<(idx%64)) == 0 {
			return false // Definitely not present
		}
	}
	return true // Might be present
}

// AddAndTest adds an item and returns whether it was possibly already present.
// This is a convenience method combining Test and Add for deduplication.
func (bf *BloomFilter) AddAndTest(key string) bool {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	hashes := bf.getHashes(key)

	// First check if all bits are set
	allSet := true
	for _, h := range hashes {
		idx := h % bf.size
		if bf.bits[idx/64]&(1<<(idx%64)) == 0 {
			allSet = false
			break
		}
	}

	// Set all bits
	for _, h := range hashes {
		idx := h % bf.size
		bf.bits[idx/64] |= 1 << (idx % 64)
	}
	bf.count++

	return allSet
}

// Clear resets the Bloom filter.
func (bf *BloomFilter) Clear() {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	for i := range bf.bits {
		bf.bits[i] = 0
	}
	bf.count = 0
}

// Count returns the number of items added (may include duplicates).
func (bf *BloomFilter) Count() int {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	return bf.count
}

// Capacity returns the expected capacity of the filter.
func (bf *BloomFilter) Capacity() int {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	return bf.capacity
}

// ApproximateFillRatio returns the approximate fill ratio of the bit array.
func (bf *BloomFilter) ApproximateFillRatio() float64 {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	setBits := 0
	for _, word := range bf.bits {
		setBits += popcount(word)
	}
	return float64(setBits) / float64(bf.size)
}

// getHashes generates multiple hash values for a key using double hashing technique.
// This is more efficient than computing k independent hash functions.
func (bf *BloomFilter) getHashes(key string) []uint64 {
	// Use FNV-1a for first hash
	h1 := fnv.New64a()
	h1.Write([]byte(key))
	hash1 := h1.Sum64()

	// Use FNV-1 (non-a variant) for second hash by modifying input
	h2 := fnv.New64()
	h2.Write([]byte(key))
	h2.Write([]byte{0xff}) // Salt to differentiate
	hash2 := h2.Sum64()

	// Generate k hashes using double hashing: h(i) = h1 + i*h2
	hashes := make([]uint64, bf.hashFns)
	for i := 0; i < bf.hashFns; i++ {
		hashes[i] = hash1 + uint64(i)*hash2
	}
	return hashes
}

// popcount returns the number of set bits in a uint64 (population count).
func popcount(x uint64) int {
	// Brian Kernighan's algorithm - efficient for sparse bit patterns
	count := 0
	for x != 0 {
		x &= x - 1
		count++
	}
	return count
}

// approximateLn computes natural logarithm approximation for small values.
// Used for Bloom filter sizing calculations.
func approximateLn(x float64) float64 {
	// For values between 0 and 1, use series expansion: ln(x) = -ln(1/x)
	// ln(1/x) = (1/x - 1) - (1/x - 1)^2/2 + (1/x - 1)^3/3 - ...
	// But simpler: use lookup table approximation for common false positive rates

	switch {
	case x >= 0.1:
		return -2.303 // ln(0.1)
	case x >= 0.05:
		return -2.996 // ln(0.05)
	case x >= 0.01:
		return -4.605 // ln(0.01)
	case x >= 0.005:
		return -5.298 // ln(0.005)
	case x >= 0.001:
		return -6.908 // ln(0.001)
	default:
		return -9.210 // ln(0.0001)
	}
}

// BloomLRU combines a Bloom filter with an LRU cache for efficient deduplication.
// The Bloom filter provides fast negative lookups, while the LRU provides
// accurate verification and TTL-based expiration.
//
// Usage:
//   - ~90%+ of unique items short-circuit at Bloom filter (fast path)
//   - Only potential duplicates check the LRU cache (slow path)
//   - LRU provides TTL expiration and accurate duplicate tracking
type BloomLRU struct {
	bloom *BloomFilter
	lru   *LRUCache
	mu    sync.RWMutex

	// Stats
	bloomNegatives int64 // Items definitely not in cache (bloom said no)
	lruChecks      int64 // Items that needed LRU verification
	duplicates     int64 // Confirmed duplicates
}

// NewBloomLRU creates a new combined Bloom filter + LRU cache.
func NewBloomLRU(capacity int, ttl time.Duration, falsePositiveRate float64) *BloomLRU {
	return &BloomLRU{
		bloom: NewBloomFilter(capacity, falsePositiveRate),
		lru:   NewLRUCache(capacity, ttl),
	}
}

// IsDuplicate checks if a key has been seen before.
// Fast path: Bloom filter says definitely not seen.
// Slow path: Bloom says maybe, verify with LRU.
func (bl *BloomLRU) IsDuplicate(key string) bool {
	// Fast path: Bloom filter says definitely not seen
	if !bl.bloom.Test(key) {
		bl.mu.Lock()
		bl.bloomNegatives++
		bl.mu.Unlock()

		// Add to both structures
		bl.bloom.Add(key)
		bl.lru.Add(key, time.Now())
		return false
	}

	// Slow path: might be a duplicate, verify with LRU
	bl.mu.Lock()
	bl.lruChecks++
	bl.mu.Unlock()

	if bl.lru.IsDuplicate(key) {
		bl.mu.Lock()
		bl.duplicates++
		bl.mu.Unlock()
		return true
	}

	// Not in LRU (expired or false positive from bloom), add it
	bl.bloom.Add(key) // Re-add in case bloom state drifted
	return false
}

// Record records a key as seen without checking for duplicates.
func (bl *BloomLRU) Record(key string) {
	bl.bloom.Add(key)
	bl.lru.Add(key, time.Now())
}

// Contains checks if a key might be in the cache without modifying it.
func (bl *BloomLRU) Contains(key string) bool {
	if !bl.bloom.Test(key) {
		return false
	}
	return bl.lru.Contains(key)
}

// CleanupExpired removes expired entries from the LRU cache.
// Note: Bloom filter cannot be cleaned (items are permanent).
func (bl *BloomLRU) CleanupExpired() int {
	return bl.lru.CleanupExpired()
}

// Clear resets both the Bloom filter and LRU cache.
func (bl *BloomLRU) Clear() {
	bl.bloom.Clear()
	bl.lru.Clear()

	bl.mu.Lock()
	bl.bloomNegatives = 0
	bl.lruChecks = 0
	bl.duplicates = 0
	bl.mu.Unlock()
}

// Stats returns performance statistics.
func (bl *BloomLRU) Stats() (bloomNegatives, lruChecks, duplicates int64, lruSize int) {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	return bl.bloomNegatives, bl.lruChecks, bl.duplicates, bl.lru.Len()
}

// Len returns the number of items in the LRU cache.
func (bl *BloomLRU) Len() int {
	return bl.lru.Len()
}

// ExactLRU provides a deduplication cache with ZERO false positives.
// Unlike BloomLRU which uses a probabilistic Bloom filter (1% false positive rate),
// ExactLRU uses only the exact-match LRU cache.
//
// Trade-offs vs BloomLRU:
//   - ZERO false positives (vs 1% with BloomLRU)
//   - Slightly higher memory usage (full keys stored)
//   - Same O(1) performance for all operations
//
// CRITICAL (v2.3): For applications requiring zero data loss, ExactLRU is
// the recommended choice as it eliminates the risk of incorrectly marking
// unique events as duplicates.
type ExactLRU struct {
	lru *LRUCache
	mu  sync.RWMutex

	// Stats for compatibility with BloomLRU interface
	checks     int64 // Total duplicate checks
	duplicates int64 // Confirmed duplicates
}

// NewExactLRU creates a new exact-match LRU cache for deduplication.
// Unlike BloomLRU, this has ZERO false positives - only exact string matches
// are considered duplicates.
//
// Parameters:
//   - capacity: maximum number of entries in the cache
//   - ttl: time-to-live for entries before they expire
func NewExactLRU(capacity int, ttl time.Duration) *ExactLRU {
	return &ExactLRU{
		lru: NewLRUCache(capacity, ttl),
	}
}

// IsDuplicate checks if a key has been seen before using exact matching.
// Returns true if the key exists and hasn't expired.
// If not a duplicate, records the key for future checks.
//
// GUARANTEED: Zero false positives. If this returns true, the key was
// definitely seen before (within TTL window).
func (el *ExactLRU) IsDuplicate(key string) bool {
	el.mu.Lock()
	el.checks++
	el.mu.Unlock()

	isDup := el.lru.IsDuplicate(key)
	if isDup {
		el.mu.Lock()
		el.duplicates++
		el.mu.Unlock()
	}
	return isDup
}

// Record records a key as seen without checking for duplicates.
func (el *ExactLRU) Record(key string) {
	el.lru.Add(key, time.Now())
}

// Contains checks if a key might be in the cache without modifying it.
// Unlike BloomLRU, this is an exact check with zero false positives.
func (el *ExactLRU) Contains(key string) bool {
	return el.lru.Contains(key)
}

// CleanupExpired removes expired entries from the LRU cache.
func (el *ExactLRU) CleanupExpired() int {
	return el.lru.CleanupExpired()
}

// Clear resets the cache.
func (el *ExactLRU) Clear() {
	el.lru.Clear()

	el.mu.Lock()
	el.checks = 0
	el.duplicates = 0
	el.mu.Unlock()
}

// Stats returns performance statistics compatible with BloomLRU interface.
// Returns (bloomNegatives=0, lruChecks=checks, duplicates, lruSize).
func (el *ExactLRU) Stats() (bloomNegatives, lruChecks, duplicates int64, lruSize int) {
	el.mu.RLock()
	defer el.mu.RUnlock()

	// bloomNegatives is always 0 since we don't use a bloom filter
	return 0, el.checks, el.duplicates, el.lru.Len()
}

// Len returns the number of items in the LRU cache.
func (el *ExactLRU) Len() int {
	return el.lru.Len()
}
