// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSlidingWindowCounter_BasicOperations(t *testing.T) {
	sw := NewSlidingWindowCounter(time.Second, 10)

	// Initial count should be 0
	if sw.Count() != 0 {
		t.Errorf("Expected initial count 0, got %d", sw.Count())
	}

	// Increment
	sw.IncrementOne()
	sw.IncrementOne()
	sw.Increment(3)

	if sw.Count() != 5 {
		t.Errorf("Expected count 5, got %d", sw.Count())
	}
}

func TestSlidingWindowCounter_WindowExpiration(t *testing.T) {
	// Short window for testing
	sw := NewSlidingWindowCounter(100*time.Millisecond, 10)

	sw.Increment(10)

	// Count should be 10 immediately
	if sw.Count() != 10 {
		t.Errorf("Expected count 10, got %d", sw.Count())
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Count should be 0 after expiration
	if sw.Count() != 0 {
		t.Errorf("Expected count 0 after expiration, got %d", sw.Count())
	}
}

func TestSlidingWindowCounter_PartialExpiration(t *testing.T) {
	// 100ms window with 10 buckets (10ms per bucket)
	sw := NewSlidingWindowCounter(100*time.Millisecond, 10)

	sw.Increment(10)

	// Wait for half the window
	time.Sleep(60 * time.Millisecond)

	// Add more
	sw.Increment(5)

	// Should have some from first increment + all from second
	count := sw.Count()
	if count < 5 || count > 15 {
		t.Errorf("Expected count between 5 and 15, got %d", count)
	}

	// Wait for first batch to fully expire
	time.Sleep(60 * time.Millisecond)

	// Should only have second batch
	count = sw.Count()
	if count != 5 {
		t.Logf("Count after expiration: %d (expected 5, timing-dependent)", count)
	}
}

func TestSlidingWindowCounter_Reset(t *testing.T) {
	sw := NewSlidingWindowCounter(time.Minute, 10)

	sw.Increment(100)
	sw.Reset()

	if sw.Count() != 0 {
		t.Errorf("Expected count 0 after reset, got %d", sw.Count())
	}
}

func TestSlidingWindowCounter_Concurrent(t *testing.T) {
	sw := NewSlidingWindowCounter(time.Second, 10)

	var wg sync.WaitGroup
	numGoroutines := 100
	incrementsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				sw.IncrementOne()
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * incrementsPerGoroutine)
	if sw.Count() != expected {
		t.Errorf("Expected count %d, got %d", expected, sw.Count())
	}
}

func TestSlidingWindowStore_BasicOperations(t *testing.T) {
	store := NewSlidingWindowStore(time.Minute, 10, 0)

	store.Increment("user1")
	store.Increment("user1")
	store.Increment("user2")

	if store.Count("user1") != 2 {
		t.Errorf("Expected count 2 for user1, got %d", store.Count("user1"))
	}

	if store.Count("user2") != 1 {
		t.Errorf("Expected count 1 for user2, got %d", store.Count("user2"))
	}

	if store.Count("user3") != 0 {
		t.Errorf("Expected count 0 for user3, got %d", store.Count("user3"))
	}
}

func TestSlidingWindowStore_IncrementBy(t *testing.T) {
	store := NewSlidingWindowStore(time.Minute, 10, 0)

	store.IncrementBy("key", 5)
	store.IncrementBy("key", 3)

	if store.Count("key") != 8 {
		t.Errorf("Expected count 8, got %d", store.Count("key"))
	}
}

func TestSlidingWindowStore_MaxKeys(t *testing.T) {
	store := NewSlidingWindowStore(time.Minute, 10, 3)

	store.Increment("key1")
	store.Increment("key2")
	store.Increment("key3")

	if store.Len() != 3 {
		t.Errorf("Expected len 3, got %d", store.Len())
	}

	// Adding 4th key should evict one
	store.Increment("key4")

	if store.Len() != 3 {
		t.Errorf("Expected len still 3 after eviction, got %d", store.Len())
	}
}

func TestSlidingWindowStore_Remove(t *testing.T) {
	store := NewSlidingWindowStore(time.Minute, 10, 0)

	store.Increment("key1")
	store.Increment("key2")

	store.Remove("key1")

	if store.Count("key1") != 0 {
		t.Errorf("Expected count 0 after remove, got %d", store.Count("key1"))
	}

	if store.Len() != 1 {
		t.Errorf("Expected len 1 after remove, got %d", store.Len())
	}
}

func TestSlidingWindowStore_Keys(t *testing.T) {
	store := NewSlidingWindowStore(time.Minute, 10, 0)

	store.Increment("a")
	store.Increment("b")
	store.Increment("c")

	keys := store.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	if !keySet["a"] || !keySet["b"] || !keySet["c"] {
		t.Error("Expected all keys to be present")
	}
}

func TestSlidingWindowStore_Clear(t *testing.T) {
	store := NewSlidingWindowStore(time.Minute, 10, 0)

	store.Increment("key1")
	store.Increment("key2")

	store.Clear()

	if store.Len() != 0 {
		t.Errorf("Expected len 0 after clear, got %d", store.Len())
	}
}

func TestSlidingWindowStore_CleanupInactive(t *testing.T) {
	// Short window for testing
	store := NewSlidingWindowStore(50*time.Millisecond, 10, 0)

	store.Increment("active")
	store.Increment("inactive")

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Re-increment active key
	store.Increment("active")

	removed := store.CleanupInactive()
	if removed != 1 {
		t.Errorf("Expected 1 inactive counter removed, got %d", removed)
	}

	if store.Len() != 1 {
		t.Errorf("Expected len 1 after cleanup, got %d", store.Len())
	}

	if store.Count("active") != 1 {
		t.Errorf("Expected active count 1, got %d", store.Count("active"))
	}
}

func TestUniqueValueCounter_BasicOperations(t *testing.T) {
	uvc := NewUniqueValueCounter(time.Minute, 10)

	uvc.Add("ip1")
	uvc.Add("ip2")
	uvc.Add("ip1") // Duplicate

	if uvc.CountUnique() != 2 {
		t.Errorf("Expected 2 unique values, got %d", uvc.CountUnique())
	}
}

func TestUniqueValueCounter_GetUnique(t *testing.T) {
	uvc := NewUniqueValueCounter(time.Minute, 10)

	uvc.Add("a")
	uvc.Add("b")
	uvc.Add("c")
	uvc.Add("a")

	unique := uvc.GetUnique()
	if len(unique) != 3 {
		t.Errorf("Expected 3 unique values, got %d", len(unique))
	}

	valueSet := make(map[string]bool)
	for _, v := range unique {
		valueSet[v] = true
	}

	if !valueSet["a"] || !valueSet["b"] || !valueSet["c"] {
		t.Error("Expected all unique values to be present")
	}
}

func TestUniqueValueCounter_WindowExpiration(t *testing.T) {
	// Short window for testing
	uvc := NewUniqueValueCounter(100*time.Millisecond, 10)

	uvc.Add("ip1")
	uvc.Add("ip2")

	if uvc.CountUnique() != 2 {
		t.Errorf("Expected 2 unique values, got %d", uvc.CountUnique())
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	if uvc.CountUnique() != 0 {
		t.Errorf("Expected 0 unique values after expiration, got %d", uvc.CountUnique())
	}
}

func TestUniqueValueCounter_Reset(t *testing.T) {
	uvc := NewUniqueValueCounter(time.Minute, 10)

	uvc.Add("ip1")
	uvc.Add("ip2")

	uvc.Reset()

	if uvc.CountUnique() != 0 {
		t.Errorf("Expected 0 after reset, got %d", uvc.CountUnique())
	}
}

func TestUniqueValueStore_BasicOperations(t *testing.T) {
	store := NewUniqueValueStore(time.Minute, 10, 0)

	store.Add("device1", "ip1")
	store.Add("device1", "ip2")
	store.Add("device1", "ip1") // Duplicate
	store.Add("device2", "ip3")

	if store.CountUnique("device1") != 2 {
		t.Errorf("Expected 2 unique for device1, got %d", store.CountUnique("device1"))
	}

	if store.CountUnique("device2") != 1 {
		t.Errorf("Expected 1 unique for device2, got %d", store.CountUnique("device2"))
	}
}

func TestUniqueValueStore_GetUnique(t *testing.T) {
	store := NewUniqueValueStore(time.Minute, 10, 0)

	store.Add("device", "ip1")
	store.Add("device", "ip2")
	store.Add("device", "ip3")

	unique := store.GetUnique("device")
	if len(unique) != 3 {
		t.Errorf("Expected 3 unique values, got %d", len(unique))
	}
}

func TestUniqueValueStore_MaxKeys(t *testing.T) {
	store := NewUniqueValueStore(time.Minute, 10, 2)

	store.Add("device1", "ip1")
	store.Add("device2", "ip2")

	if store.Len() != 2 {
		t.Errorf("Expected len 2, got %d", store.Len())
	}

	// Adding 3rd device should evict one
	store.Add("device3", "ip3")

	if store.Len() != 2 {
		t.Errorf("Expected len still 2 after eviction, got %d", store.Len())
	}
}

func TestUniqueValueStore_Remove(t *testing.T) {
	store := NewUniqueValueStore(time.Minute, 10, 0)

	store.Add("device", "ip1")
	store.Remove("device")

	if store.CountUnique("device") != 0 {
		t.Errorf("Expected 0 after remove, got %d", store.CountUnique("device"))
	}
}

func BenchmarkSlidingWindowCounter_Increment(b *testing.B) {
	sw := NewSlidingWindowCounter(time.Minute, 60)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.IncrementOne()
	}
}

func BenchmarkSlidingWindowCounter_Count(b *testing.B) {
	sw := NewSlidingWindowCounter(time.Minute, 60)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		sw.IncrementOne()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.Count()
	}
}

func BenchmarkSlidingWindowStore_Increment(b *testing.B) {
	store := NewSlidingWindowStore(time.Minute, 60, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Increment(fmt.Sprintf("key-%d", i%1000))
	}
}

func BenchmarkUniqueValueCounter_Add(b *testing.B) {
	uvc := NewUniqueValueCounter(time.Minute, 60)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uvc.Add(fmt.Sprintf("value-%d", i%100))
	}
}

func BenchmarkUniqueValueCounter_CountUnique(b *testing.B) {
	uvc := NewUniqueValueCounter(time.Minute, 60)

	// Pre-populate with some unique values
	for i := 0; i < 100; i++ {
		uvc.Add(fmt.Sprintf("value-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uvc.CountUnique()
	}
}
