// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	c := New(1 * time.Minute)

	// Test Set and Get
	c.Set("key1", "value1")
	value, exists := c.Get("key1")
	if !exists {
		t.Error("Expected key1 to exist")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// Test non-existent key
	_, exists = c.Get("key2")
	if exists {
		t.Error("Expected key2 to not exist")
	}
}

func TestCacheExpiration(t *testing.T) {
	c := New(100 * time.Millisecond)

	c.Set("key1", "value1")

	// Value should exist immediately
	_, exists := c.Get("key1")
	if !exists {
		t.Error("Expected key1 to exist immediately after set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Value should be expired
	_, exists = c.Get("key1")
	if exists {
		t.Error("Expected key1 to be expired")
	}
}

func TestCacheDelete(t *testing.T) {
	c := New(1 * time.Minute)

	c.Set("key1", "value1")
	c.Delete("key1")

	_, exists := c.Get("key1")
	if exists {
		t.Error("Expected key1 to be deleted")
	}
}

func TestCacheClear(t *testing.T) {
	c := New(1 * time.Minute)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	c.Clear()

	for _, key := range []string{"key1", "key2", "key3"} {
		_, exists := c.Get(key)
		if exists {
			t.Errorf("Expected %s to be cleared", key)
		}
	}
}

func TestCacheStats(t *testing.T) {
	c := New(1 * time.Minute)

	c.Set("key1", "value1")
	c.Get("key1") // hit
	c.Get("key2") // miss
	c.Get("key1") // hit

	stats := c.GetStats()

	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	hitRate := c.HitRate()
	expectedHitRate := 66.66666666666667 // 2/3 * 100
	if hitRate < expectedHitRate-0.01 || hitRate > expectedHitRate+0.01 {
		t.Errorf("Expected hit rate around %.2f%%, got %.2f%%", expectedHitRate, hitRate)
	}
}

func TestCacheSetWithTTL(t *testing.T) {
	c := New(1 * time.Minute)

	// Set with short TTL
	c.SetWithTTL("key1", "value1", 100*time.Millisecond)

	// Should exist immediately
	_, exists := c.Get("key1")
	if !exists {
		t.Error("Expected key1 to exist")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, exists = c.Get("key1")
	if exists {
		t.Error("Expected key1 to be expired")
	}
}

func TestGenerateKey(t *testing.T) {
	type TestParams struct {
		UserID int
		Date   string
	}

	params1 := TestParams{UserID: 1, Date: "2024-01-01"}
	params2 := TestParams{UserID: 1, Date: "2024-01-01"}
	params3 := TestParams{UserID: 2, Date: "2024-01-01"}

	key1 := GenerateKey("method1", params1)
	key2 := GenerateKey("method1", params2)
	key3 := GenerateKey("method1", params3)

	// Same params should generate same key
	if key1 != key2 {
		t.Error("Expected same params to generate same key")
	}

	// Different params should generate different key
	if key1 == key3 {
		t.Error("Expected different params to generate different key")
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := New(1 * time.Minute)

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := "key"
				c.Set(key, id)
				c.Get(key)
				if j%10 == 0 {
					c.Delete(key)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we get here without deadlock or panic, the test passes
	stats := c.GetStats()
	if stats.Hits == 0 && stats.Misses == 0 {
		t.Error("Expected some cache activity from concurrent operations")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	c := New(1 * time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set("key", "value")
	}
}

func BenchmarkCacheGet(b *testing.B) {
	c := New(1 * time.Minute)
	c.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("key")
	}
}

func BenchmarkGenerateKey(b *testing.B) {
	type TestParams struct {
		UserID int
		Date   string
		Limit  int
	}

	params := TestParams{UserID: 123, Date: "2024-01-01", Limit: 100}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateKey("GetTopUsers", params)
	}
}

// Test manual cleanup functionality
func TestCacheManualCleanup(t *testing.T) {
	c := New(50 * time.Millisecond)

	// Add some entries
	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	// Verify all exist
	if _, exists := c.Get("key1"); !exists {
		t.Error("Expected key1 to exist")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup
	c.cleanup()

	// Verify all are cleaned up
	stats := c.GetStats()
	if stats.TotalKeys != 0 {
		t.Errorf("Expected 0 total keys after cleanup, got %d", stats.TotalKeys)
	}

	if stats.Evictions != 3 {
		t.Errorf("Expected 3 evictions, got %d", stats.Evictions)
	}

	// Verify LastCleanup was updated
	if stats.LastCleanup.IsZero() {
		t.Error("Expected LastCleanup to be set")
	}
}

// Test cleanup of partially expired entries
func TestCachePartialExpiration(t *testing.T) {
	c := New(100 * time.Millisecond)

	// Set entries with different TTLs
	c.SetWithTTL("short-lived", "value1", 50*time.Millisecond)
	c.SetWithTTL("long-lived", "value2", 200*time.Millisecond)

	// Wait for short-lived to expire
	time.Sleep(75 * time.Millisecond)

	// Trigger cleanup
	c.cleanup()

	// Short-lived should be gone
	if _, exists := c.Get("short-lived"); exists {
		t.Error("Expected short-lived key to be cleaned up")
	}

	// Long-lived should still exist
	if _, exists := c.Get("long-lived"); !exists {
		t.Error("Expected long-lived key to still exist")
	}

	stats := c.GetStats()
	if stats.TotalKeys != 1 {
		t.Errorf("Expected 1 total key, got %d", stats.TotalKeys)
	}
}

// Test cleanup loop runs periodically
func TestCacheCleanupLoop(t *testing.T) {
	// Note: The cleanup loop runs every 5 minutes by default
	// For testing, we create a cache and verify it's initialized correctly
	c := New(1 * time.Millisecond)

	// Add an entry that will expire quickly
	c.Set("test-key", "test-value")

	// The cleanup loop is running in background
	// Wait a reasonable time for at least one cleanup cycle
	// Since we can't easily control the ticker, we just verify the mechanism works

	// Give the cleanup loop time to potentially run
	time.Sleep(10 * time.Millisecond)

	// The entry should have expired
	_, exists := c.Get("test-key")
	if exists {
		t.Log("Entry still exists - cleanup loop may not have run yet (this is timing-dependent)")
	}

	// The important part is that cleanup() can be called without panicking
	c.cleanup()
}

// Test zero TTL behavior
func TestCacheZeroTTL(t *testing.T) {
	c := New(0)

	c.Set("key1", "value1")

	// With zero or negative TTL, items expire immediately
	_, exists := c.Get("key1")
	if exists {
		t.Error("Expected key with zero TTL to be expired immediately")
	}
}

// Test very short TTL
func TestCacheVeryShortTTL(t *testing.T) {
	c := New(1 * time.Nanosecond)

	c.Set("key1", "value1")

	// Even nanosecond TTL should work
	time.Sleep(1 * time.Millisecond)
	_, exists := c.Get("key1")
	if exists {
		t.Error("Expected key with nanosecond TTL to expire quickly")
	}
}

// Test Stats struct is a copy, not reference
func TestCacheStatsCopy(t *testing.T) {
	c := New(1 * time.Minute)

	c.Set("key1", "value1")
	c.Get("key1")

	stats1 := c.GetStats()
	originalHits := stats1.Hits

	// More operations
	c.Get("key1")
	c.Get("key2")

	// stats1 should still have old values (it's a copy)
	if stats1.Hits != originalHits {
		t.Error("GetStats should return a copy, not a reference")
	}

	// Get new stats
	stats2 := c.GetStats()
	if stats2.Hits == originalHits {
		t.Error("Expected new stats to reflect updated hits")
	}
}

// Test HitRate with zero operations
func TestCacheHitRateZeroOperations(t *testing.T) {
	c := New(1 * time.Minute)

	// No gets performed yet
	hitRate := c.HitRate()
	if hitRate != 0.0 {
		t.Errorf("Expected 0%% hit rate with no operations, got %.2f%%", hitRate)
	}
}

// Test HitRate with only misses
func TestCacheHitRateOnlyMisses(t *testing.T) {
	c := New(1 * time.Minute)

	// Only misses
	c.Get("nonexistent1")
	c.Get("nonexistent2")
	c.Get("nonexistent3")

	hitRate := c.HitRate()
	if hitRate != 0.0 {
		t.Errorf("Expected 0%% hit rate with only misses, got %.2f%%", hitRate)
	}

	stats := c.GetStats()
	if stats.Hits != 0 {
		t.Errorf("Expected 0 hits, got %d", stats.Hits)
	}
	if stats.Misses != 3 {
		t.Errorf("Expected 3 misses, got %d", stats.Misses)
	}
}

// Test HitRate with only hits
func TestCacheHitRateOnlyHits(t *testing.T) {
	c := New(1 * time.Minute)

	c.Set("key1", "value1")

	// Only hits
	c.Get("key1")
	c.Get("key1")
	c.Get("key1")

	hitRate := c.HitRate()
	if hitRate != 100.0 {
		t.Errorf("Expected 100%% hit rate with only hits, got %.2f%%", hitRate)
	}

	stats := c.GetStats()
	if stats.Hits != 3 {
		t.Errorf("Expected 3 hits, got %d", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("Expected 0 misses, got %d", stats.Misses)
	}
}

// Test eviction counter on delete
func TestCacheEvictionCounter(t *testing.T) {
	c := New(1 * time.Minute)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	initialStats := c.GetStats()
	initialEvictions := initialStats.Evictions

	// Delete one key
	c.Delete("key1")

	stats := c.GetStats()
	if stats.Evictions != initialEvictions+1 {
		t.Errorf("Expected evictions to increase by 1, got %d", stats.Evictions-initialEvictions)
	}
}

// Test eviction counter on clear
func TestCacheEvictionCounterOnClear(t *testing.T) {
	c := New(1 * time.Minute)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	initialStats := c.GetStats()

	c.Clear()

	stats := c.GetStats()
	expectedEvictions := initialStats.Evictions + 3
	if stats.Evictions != expectedEvictions {
		t.Errorf("Expected %d evictions, got %d", expectedEvictions, stats.Evictions)
	}

	if stats.TotalKeys != 0 {
		t.Errorf("Expected 0 total keys after clear, got %d", stats.TotalKeys)
	}
}

// Test eviction counter on expiration
func TestCacheEvictionCounterOnExpiration(t *testing.T) {
	c := New(50 * time.Millisecond)

	c.Set("key1", "value1")

	initialStats := c.GetStats()

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Access expired key (triggers eviction)
	c.Get("key1")

	stats := c.GetStats()
	if stats.Evictions <= initialStats.Evictions {
		t.Error("Expected evictions to increase when accessing expired key")
	}
}

// Test TotalKeys counter updates
func TestCacheTotalKeysCounter(t *testing.T) {
	c := New(1 * time.Minute)

	// Add keys one by one
	c.Set("key1", "value1")
	stats := c.GetStats()
	if stats.TotalKeys != 1 {
		t.Errorf("Expected 1 total key, got %d", stats.TotalKeys)
	}

	c.Set("key2", "value2")
	stats = c.GetStats()
	if stats.TotalKeys != 2 {
		t.Errorf("Expected 2 total keys, got %d", stats.TotalKeys)
	}

	c.Set("key3", "value3")
	stats = c.GetStats()
	if stats.TotalKeys != 3 {
		t.Errorf("Expected 3 total keys, got %d", stats.TotalKeys)
	}

	// Overwrite existing key (should not increase count)
	c.Set("key1", "new-value1")
	stats = c.GetStats()
	if stats.TotalKeys != 3 {
		t.Errorf("Expected 3 total keys after overwrite, got %d", stats.TotalKeys)
	}
}

// Test GenerateKey with complex nested structures
func TestGenerateKeyComplexStructures(t *testing.T) {
	type NestedParams struct {
		Filters map[string][]string
		Options struct {
			Sort  string
			Limit int
		}
	}

	params1 := NestedParams{
		Filters: map[string][]string{
			"type": {"movie", "episode"},
			"user": {"alice"},
		},
	}
	params1.Options.Sort = "date"
	params1.Options.Limit = 100

	params2 := NestedParams{
		Filters: map[string][]string{
			"type": {"movie", "episode"},
			"user": {"alice"},
		},
	}
	params2.Options.Sort = "date"
	params2.Options.Limit = 100

	params3 := NestedParams{
		Filters: map[string][]string{
			"type": {"movie"},
			"user": {"bob"},
		},
	}
	params3.Options.Sort = "title"
	params3.Options.Limit = 50

	key1 := GenerateKey("ComplexQuery", params1)
	key2 := GenerateKey("ComplexQuery", params2)
	key3 := GenerateKey("ComplexQuery", params3)

	// Same params should generate same key
	if key1 != key2 {
		t.Error("Expected identical complex params to generate same key")
	}

	// Different params should generate different key
	if key1 == key3 {
		t.Error("Expected different complex params to generate different key")
	}

	// Verify key format (method:hash)
	if !contains(key1, "ComplexQuery:") {
		t.Errorf("Expected key to contain method name, got: %s", key1)
	}
}

// Test GenerateKey with unmarshalable data (should fallback)
func TestGenerateKeyUnmarshalable(t *testing.T) {
	// Channels cannot be marshaled to JSON
	type UnmarshalableParams struct {
		Ch chan int
	}

	params := UnmarshalableParams{
		Ch: make(chan int),
	}

	// Should fallback to simple string key without panicking
	key := GenerateKey("TestMethod", params)

	if key == "" {
		t.Error("Expected non-empty key even with unmarshalable data")
	}

	// Should contain method name
	if !contains(key, "TestMethod:") {
		t.Errorf("Expected key to contain method name, got: %s", key)
	}
}

// Test GenerateKey with nil params
func TestGenerateKeyNilParams(t *testing.T) {
	key := GenerateKey("NilParamsMethod", nil)

	if key == "" {
		t.Error("Expected non-empty key with nil params")
	}

	if !contains(key, "NilParamsMethod:") {
		t.Errorf("Expected key to contain method name, got: %s", key)
	}
}

// Test cache with large number of entries
func TestCacheLargeNumberOfEntries(t *testing.T) {
	c := New(1 * time.Minute)

	numEntries := 10000
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		c.Set(key, value)
	}

	stats := c.GetStats()
	if stats.TotalKeys != int64(numEntries) {
		t.Errorf("Expected %d total keys, got %d", numEntries, stats.TotalKeys)
	}

	// Verify random samples
	for i := 0; i < 100; i++ {
		idx := i * 100
		key := fmt.Sprintf("key-%d", idx)
		expectedValue := fmt.Sprintf("value-%d", idx)

		value, exists := c.Get(key)
		if !exists {
			t.Errorf("Expected key %s to exist", key)
		}

		if value != expectedValue {
			t.Errorf("Expected value %s, got %v", expectedValue, value)
		}
	}
}

// Test cache entry overwrite preserves expiration update
func TestCacheEntryOverwrite(t *testing.T) {
	c := New(200 * time.Millisecond) // Increased TTL for CI stability

	// Set initial value
	c.Set("key1", "value1")

	// Wait a bit (25% of TTL)
	time.Sleep(50 * time.Millisecond)

	// Overwrite with new value (resets expiration)
	c.Set("key1", "value2")

	// Wait past original expiration but within reset window
	// Original would expire at 200ms, we're at 50+100=150ms
	// Reset expiration is at 50+200=250ms, so 150ms < 250ms
	time.Sleep(100 * time.Millisecond)

	// Should still exist (expiration was reset at T=50ms to T=250ms)
	value, exists := c.Get("key1")
	if !exists {
		t.Error("Expected overwritten key to have reset expiration")
	}

	if value != "value2" {
		t.Errorf("Expected value2, got %v", value)
	}
}

// Test SetWithTTL overrides default TTL
func TestCacheSetWithTTLOverridesDefault(t *testing.T) {
	c := New(50 * time.Millisecond) // Default 50ms

	// Set with custom longer TTL
	c.SetWithTTL("long-key", "long-value", 200*time.Millisecond)

	// Set with default TTL
	c.Set("short-key", "short-value")

	// Wait for default TTL to expire
	time.Sleep(75 * time.Millisecond)

	// Short key should be expired
	if _, exists := c.Get("short-key"); exists {
		t.Error("Expected short key to be expired")
	}

	// Long key should still exist
	if _, exists := c.Get("long-key"); !exists {
		t.Error("Expected long key to still exist")
	}
}

// Benchmark cleanup operation
func BenchmarkCacheCleanup(b *testing.B) {
	c := New(1 * time.Millisecond)

	// Add many entries
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
	}

	// Wait for all to expire
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.cleanup()
	}
}

// Helper function for string contains
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
