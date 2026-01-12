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

func TestLFUCache_BasicOperations(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 5*time.Minute)

	// Test Set and Get
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	if val, found := cache.Get("key1"); !found || val != "value1" {
		t.Errorf("Get('key1') = %v, %v, want 'value1', true", val, found)
	}

	if val, found := cache.Get("key2"); !found || val != "value2" {
		t.Errorf("Get('key2') = %v, %v, want 'value2', true", val, found)
	}

	// Test non-existent key
	if _, found := cache.Get("nonexistent"); found {
		t.Error("Get('nonexistent') should return false")
	}
}

func TestLFUCache_FrequencyTracking(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 5*time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Access key1 multiple times to increase frequency
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key1")

	// Access key2 once
	cache.Get("key2")

	// key1 should have higher frequency
	freq1 := cache.GetFrequency("key1")
	freq2 := cache.GetFrequency("key2")

	if freq1 <= freq2 {
		t.Errorf("key1 frequency (%d) should be > key2 frequency (%d)", freq1, freq2)
	}
}

func TestLFUCache_Eviction(t *testing.T) {
	t.Parallel()

	// Small cache to test eviction
	cache := NewLFUCache(3, 5*time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Access key1 and key2 to increase their frequency
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key2")

	// key3 has the lowest frequency (only initial insert)
	// Adding key4 should evict key3
	cache.Set("key4", "value4")

	if cache.Contains("key3") {
		t.Error("key3 should have been evicted (lowest frequency)")
	}

	if !cache.Contains("key1") {
		t.Error("key1 should still exist (higher frequency)")
	}

	if !cache.Contains("key2") {
		t.Error("key2 should still exist (higher frequency)")
	}

	if !cache.Contains("key4") {
		t.Error("key4 should exist (just added)")
	}
}

func TestLFUCache_EvictionSameFrequency(t *testing.T) {
	t.Parallel()

	// When multiple entries have the same frequency,
	// evict the least recently used among them
	cache := NewLFUCache(3, 5*time.Minute)

	cache.Set("key1", "value1")
	time.Sleep(1 * time.Millisecond)
	cache.Set("key2", "value2")
	time.Sleep(1 * time.Millisecond)
	cache.Set("key3", "value3")

	// All have frequency 1 (only inserted, never accessed)
	// key1 is the oldest, should be evicted first
	cache.Set("key4", "value4")

	if cache.Contains("key1") {
		t.Error("key1 should have been evicted (oldest with same frequency)")
	}

	if !cache.Contains("key2") {
		t.Error("key2 should still exist")
	}

	if !cache.Contains("key3") {
		t.Error("key3 should still exist")
	}

	if !cache.Contains("key4") {
		t.Error("key4 should exist (just added)")
	}
}

func TestLFUCache_Update(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 5*time.Minute)

	cache.Set("key1", "original")

	// Get to increase frequency
	cache.Get("key1")
	freq1 := cache.GetFrequency("key1")

	// Update value
	cache.Set("key1", "updated")

	if val, found := cache.Get("key1"); !found || val != "updated" {
		t.Errorf("Get('key1') after update = %v, %v, want 'updated', true", val, found)
	}

	// Frequency should have increased (Set also increments frequency for existing keys)
	freq2 := cache.GetFrequency("key1")
	if freq2 <= freq1 {
		t.Errorf("Frequency after update (%d) should be > before (%d)", freq2, freq1)
	}
}

func TestLFUCache_Delete(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 5*time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Delete existing key
	if !cache.Delete("key1") {
		t.Error("Delete('key1') should return true")
	}

	if cache.Contains("key1") {
		t.Error("key1 should not exist after delete")
	}

	// Delete non-existent key
	if cache.Delete("nonexistent") {
		t.Error("Delete('nonexistent') should return false")
	}

	// key2 should still exist
	if !cache.Contains("key2") {
		t.Error("key2 should still exist")
	}
}

func TestLFUCache_TTL(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 50*time.Millisecond)

	cache.Set("key1", "value1")

	// Should be found immediately
	if _, found := cache.Get("key1"); !found {
		t.Error("key1 should be found before expiration")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Should not be found after expiration
	if _, found := cache.Get("key1"); found {
		t.Error("key1 should not be found after expiration")
	}
}

func TestLFUCache_CustomTTL(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 1*time.Hour)

	cache.SetWithTTL("key1", "value1", 50*time.Millisecond)

	// Should be found immediately
	if _, found := cache.Get("key1"); !found {
		t.Error("key1 should be found before expiration")
	}

	// Wait for custom TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Should not be found after expiration
	if _, found := cache.Get("key1"); found {
		t.Error("key1 should not be found after custom TTL expiration")
	}
}

func TestLFUCache_Clear(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 5*time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", cache.Len())
	}

	if cache.Contains("key1") {
		t.Error("key1 should not exist after clear")
	}
}

func TestLFUCache_Len(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 5*time.Minute)

	if cache.Len() != 0 {
		t.Errorf("Len() = %d, want 0 for empty cache", cache.Len())
	}

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	if cache.Len() != 2 {
		t.Errorf("Len() = %d, want 2", cache.Len())
	}

	cache.Delete("key1")

	if cache.Len() != 1 {
		t.Errorf("Len() after delete = %d, want 1", cache.Len())
	}
}

func TestLFUCache_Stats(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 5*time.Minute)

	cache.Set("key1", "value1")

	// Hit
	cache.Get("key1")
	cache.Get("key1")

	// Miss
	cache.Get("nonexistent")

	hits, misses, size := cache.Stats()

	if hits != 2 {
		t.Errorf("Hits = %d, want 2", hits)
	}
	if misses != 1 {
		t.Errorf("Misses = %d, want 1", misses)
	}
	if size != 1 {
		t.Errorf("Size = %d, want 1", size)
	}
}

func TestLFUCache_HitRate(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 5*time.Minute)

	// Empty cache hit rate should be 0
	if cache.HitRate() != 0.0 {
		t.Errorf("HitRate() for empty cache = %f, want 0.0", cache.HitRate())
	}

	cache.Set("key1", "value1")

	// 3 hits, 1 miss = 75% hit rate
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("nonexistent")

	hitRate := cache.HitRate()
	if hitRate < 74.9 || hitRate > 75.1 {
		t.Errorf("HitRate() = %f, want ~75.0", hitRate)
	}
}

func TestLFUCache_CleanupExpired(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 50*time.Millisecond)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	removed := cache.CleanupExpired()

	if removed != 3 {
		t.Errorf("CleanupExpired() removed %d, want 3", removed)
	}

	if cache.Len() != 0 {
		t.Errorf("Len() after cleanup = %d, want 0", cache.Len())
	}
}

func TestLFUCache_Contains(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(100, 5*time.Minute)

	cache.Set("key1", "value1")

	if !cache.Contains("key1") {
		t.Error("Contains('key1') should be true")
	}

	if cache.Contains("nonexistent") {
		t.Error("Contains('nonexistent') should be false")
	}

	// Contains should not affect frequency
	initialFreq := cache.GetFrequency("key1")
	cache.Contains("key1")
	cache.Contains("key1")
	afterFreq := cache.GetFrequency("key1")

	if initialFreq != afterFreq {
		t.Error("Contains should not affect frequency")
	}
}

func TestLFUCache_Concurrent(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(1000, 5*time.Minute)

	var wg sync.WaitGroup
	numGoroutines := 100
	numOps := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := string(rune('a' + (id+j)%26))
				cache.Set(key, id*1000+j)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := string(rune('a' + (id+j)%26))
				cache.Get(key)
				cache.Contains(key)
			}
		}(i)
	}

	wg.Wait()

	// Cache should be in a consistent state
	if cache.Len() > 1000 {
		t.Errorf("Len() = %d, should not exceed capacity 1000", cache.Len())
	}
}

func TestLFUCache_LargeScale(t *testing.T) {
	t.Parallel()

	cache := NewLFUCache(1000, 5*time.Minute)

	// Insert more items than capacity
	for i := 0; i < 2000; i++ {
		cache.Set(string(rune(i)), i)
	}

	// Cache should maintain capacity
	if cache.Len() > 1000 {
		t.Errorf("Len() = %d, should not exceed capacity 1000", cache.Len())
	}
}

func TestLFUCache_FrequencyPreservation(t *testing.T) {
	t.Parallel()

	// Capacity 4 means eviction happens when adding 5th item
	cache := NewLFUCache(4, 5*time.Minute)

	// Add items with varying frequencies
	cache.Set("low", "value")     // freq 1
	cache.Set("medium", "value")  // freq 1
	cache.Set("high", "value")    // freq 1
	cache.Set("highest", "value") // freq 1

	// Increase frequencies
	for i := 0; i < 10; i++ {
		cache.Get("highest")
	}
	for i := 0; i < 5; i++ {
		cache.Get("high")
	}
	for i := 0; i < 2; i++ {
		cache.Get("medium")
	}
	// "low" stays at freq 1

	// Add new item to trigger eviction - should evict "low" (lowest frequency)
	cache.Set("new1", "value")

	if cache.Contains("low") {
		t.Error("'low' should have been evicted (lowest frequency)")
	}

	// highest and high should definitely still exist
	if !cache.Contains("highest") {
		t.Error("'highest' should still exist")
	}
	if !cache.Contains("high") {
		t.Error("'high' should still exist")
	}
	if !cache.Contains("medium") {
		t.Error("'medium' should still exist")
	}
	if !cache.Contains("new1") {
		t.Error("'new1' should exist (just added)")
	}
}

func TestLFUCacheGeneric_BasicOperations(t *testing.T) {
	t.Parallel()

	cache := NewLFUCacheGeneric[string](100, 5*time.Minute)

	cache.Set("key1", "value1")

	val, found := cache.Get("key1")
	if !found {
		t.Error("Get should find key1")
	}
	if val != "value1" {
		t.Errorf("Get = %q, want 'value1'", val)
	}

	// Type safety check - this wouldn't compile with wrong type
	_, _ = cache.Get("key1") // val is string type
}

func TestLFUCacheGeneric_StructValues(t *testing.T) {
	t.Parallel()

	type User struct {
		ID   int
		Name string
	}

	cache := NewLFUCacheGeneric[User](100, 5*time.Minute)

	user := User{ID: 1, Name: "Alice"}
	cache.Set("user:1", user)

	retrieved, found := cache.Get("user:1")
	if !found {
		t.Error("Get should find user:1")
	}
	if retrieved.ID != 1 || retrieved.Name != "Alice" {
		t.Errorf("Get = %+v, want %+v", retrieved, user)
	}
}

func TestLFUCacheGeneric_AllMethods(t *testing.T) {
	t.Parallel()

	cache := NewLFUCacheGeneric[int](100, 5*time.Minute)

	cache.Set("a", 1)
	cache.SetWithTTL("b", 2, 1*time.Hour)

	if !cache.Contains("a") {
		t.Error("Contains('a') should be true")
	}

	if cache.Len() != 2 {
		t.Errorf("Len() = %d, want 2", cache.Len())
	}

	if !cache.Delete("a") {
		t.Error("Delete('a') should return true")
	}

	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", cache.Len())
	}

	hits, misses, size := cache.Stats()
	if hits < 0 || misses < 0 || size != 0 {
		t.Errorf("Stats() = %d, %d, %d, unexpected values", hits, misses, size)
	}

	hitRate := cache.HitRate()
	if hitRate < 0 {
		t.Errorf("HitRate() = %f, should be >= 0", hitRate)
	}
}

// Benchmark tests
func BenchmarkLFUCache_Set(b *testing.B) {
	cache := NewLFUCache(10000, 5*time.Minute)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set("key", "value")
	}
}

func BenchmarkLFUCache_Get(b *testing.B) {
	cache := NewLFUCache(10000, 5*time.Minute)
	cache.Set("key", "value")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Get("key")
	}
}

func BenchmarkLFUCache_SetEviction(b *testing.B) {
	cache := NewLFUCache(100, 5*time.Minute)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(string(rune(i%200)), "value")
	}
}

func BenchmarkLFUCache_MixedOperations(b *testing.B) {
	cache := NewLFUCache(1000, 5*time.Minute)

	// Pre-populate
	for i := 0; i < 500; i++ {
		cache.Set(string(rune(i)), "value")
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if i%3 == 0 {
			cache.Set(string(rune(i%1000)), "value")
		} else {
			cache.Get(string(rune(i % 500)))
		}
	}
}

func BenchmarkLFUCache_ConcurrentAccess(b *testing.B) {
	cache := NewLFUCache(1000, 5*time.Minute)

	// Pre-populate
	for i := 0; i < 100; i++ {
		cache.Set(string(rune(i)), "value")
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%3 == 0 {
				cache.Set(string(rune(i%200)), "value")
			} else {
				cache.Get(string(rune(i % 100)))
			}
			i++
		}
	})
}
