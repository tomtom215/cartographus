// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package cache

import (
	"sync"
	"testing"
	"time"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	cache := NewLRUCache(3, time.Minute)

	// Test Add and Get
	cache.Add("a", time.Now())
	cache.Add("b", time.Now())
	cache.Add("c", time.Now())

	if _, found := cache.Get("a"); !found {
		t.Error("Expected to find key 'a'")
	}
	if _, found := cache.Get("b"); !found {
		t.Error("Expected to find key 'b'")
	}
	if _, found := cache.Get("c"); !found {
		t.Error("Expected to find key 'c'")
	}

	// Test Len
	if cache.Len() != 3 {
		t.Errorf("Expected len 3, got %d", cache.Len())
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	cache := NewLRUCache(3, time.Minute)

	// Fill cache
	cache.Add("a", time.Now())
	cache.Add("b", time.Now())
	cache.Add("c", time.Now())

	// Access 'a' to make it most recently used
	cache.Get("a")

	// Add new item, should evict 'b' (least recently used)
	cache.Add("d", time.Now())

	// 'b' should be evicted (was LRU after 'a' was accessed)
	if _, found := cache.Get("b"); found {
		t.Error("Expected 'b' to be evicted")
	}

	// 'a', 'c', 'd' should still be present
	if _, found := cache.Get("a"); !found {
		t.Error("Expected 'a' to be present")
	}
	if _, found := cache.Get("c"); !found {
		t.Error("Expected 'c' to be present")
	}
	if _, found := cache.Get("d"); !found {
		t.Error("Expected 'd' to be present")
	}
}

func TestLRUCache_TTLExpiration(t *testing.T) {
	cache := NewLRUCache(10, 50*time.Millisecond)

	cache.Add("a", time.Now())

	// Should be found immediately
	if _, found := cache.Get("a"); !found {
		t.Error("Expected to find key 'a' immediately")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Should not be found after expiration
	if _, found := cache.Get("a"); found {
		t.Error("Expected key 'a' to be expired")
	}
}

func TestLRUCache_IsDuplicate(t *testing.T) {
	cache := NewLRUCache(100, time.Minute)

	// First time should not be duplicate
	if cache.IsDuplicate("key1") {
		t.Error("First occurrence should not be duplicate")
	}

	// Second time should be duplicate
	if !cache.IsDuplicate("key1") {
		t.Error("Second occurrence should be duplicate")
	}

	// Different key should not be duplicate
	if cache.IsDuplicate("key2") {
		t.Error("Different key should not be duplicate")
	}
}

func TestLRUCache_Remove(t *testing.T) {
	cache := NewLRUCache(10, time.Minute)

	cache.Add("a", time.Now())
	cache.Add("b", time.Now())

	if !cache.Remove("a") {
		t.Error("Expected Remove to return true for existing key")
	}

	if cache.Remove("a") {
		t.Error("Expected Remove to return false for non-existing key")
	}

	if _, found := cache.Get("a"); found {
		t.Error("Expected key 'a' to be removed")
	}

	if _, found := cache.Get("b"); !found {
		t.Error("Expected key 'b' to still be present")
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache := NewLRUCache(10, time.Minute)

	cache.Add("a", time.Now())
	cache.Add("b", time.Now())
	cache.Add("c", time.Now())

	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Expected empty cache after Clear, got len %d", cache.Len())
	}

	if _, found := cache.Get("a"); found {
		t.Error("Expected no items after Clear")
	}
}

func TestLRUCache_CleanupExpired(t *testing.T) {
	cache := NewLRUCache(10, 50*time.Millisecond)

	cache.Add("a", time.Now())
	cache.Add("b", time.Now())
	cache.Add("c", time.Now())

	// Wait for some items to expire
	time.Sleep(60 * time.Millisecond)

	// Add a new item that shouldn't expire
	cache.Add("d", time.Now())

	removed := cache.CleanupExpired()
	if removed != 3 {
		t.Errorf("Expected 3 expired items removed, got %d", removed)
	}

	if cache.Len() != 1 {
		t.Errorf("Expected 1 item remaining, got %d", cache.Len())
	}

	if _, found := cache.Get("d"); !found {
		t.Error("Expected 'd' to still be present")
	}
}

func TestLRUCache_Stats(t *testing.T) {
	cache := NewLRUCache(10, time.Minute)

	cache.Add("a", time.Now())
	cache.Get("a")        // hit
	cache.Get("a")        // hit
	cache.Get("nonexist") // miss

	hits, misses, size := cache.Stats()
	if hits != 2 {
		t.Errorf("Expected 2 hits, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("Expected 1 miss, got %d", misses)
	}
	if size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}
}

func TestLRUCache_Concurrent(t *testing.T) {
	cache := NewLRUCache(1000, time.Minute)

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune('a' + (id+j)%26))
				cache.Add(key, time.Now())
				cache.Get(key)
				cache.Contains(key)
				cache.IsDuplicate(key)
			}
		}(i)
	}

	wg.Wait()

	// Cache should still be functional
	cache.Add("test", time.Now())
	if _, found := cache.Get("test"); !found {
		t.Error("Cache should still work after concurrent access")
	}
}

func TestLRUCache_UpdateExisting(t *testing.T) {
	cache := NewLRUCache(3, time.Minute)

	t1 := time.Now()
	cache.Add("a", t1)

	// Update with new time
	t2 := t1.Add(time.Second)
	cache.Add("a", t2)

	// Should still have only 1 entry
	if cache.Len() != 1 {
		t.Errorf("Expected len 1 after update, got %d", cache.Len())
	}

	// Should return updated time
	if val, found := cache.Get("a"); !found || !val.Equal(t2) {
		t.Error("Expected updated time value")
	}
}

func BenchmarkLRUCache_Add(b *testing.B) {
	cache := NewLRUCache(10000, time.Minute)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune('a' + i%26))
		cache.Add(key, now)
	}
}

func BenchmarkLRUCache_Get(b *testing.B) {
	cache := NewLRUCache(10000, time.Minute)
	now := time.Now()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		key := string(rune('a' + i%26))
		cache.Add(key, now)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune('a' + i%26))
		cache.Get(key)
	}
}

func BenchmarkLRUCache_IsDuplicate(b *testing.B) {
	cache := NewLRUCache(10000, time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune('a' + i%26))
		cache.IsDuplicate(key)
	}
}

func BenchmarkLRUCache_Eviction(b *testing.B) {
	cache := NewLRUCache(100, time.Minute)
	now := time.Now()

	// Pre-fill cache to capacity
	for i := 0; i < 100; i++ {
		cache.Add(string(rune(i)), now)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Add new item (triggers eviction)
		cache.Add(string(rune(1000+i)), now)
	}
}
