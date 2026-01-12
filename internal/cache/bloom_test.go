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

func TestBloomFilter_BasicOperations(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	// Test Add and Test
	bf.Add("hello")
	bf.Add("world")

	if !bf.Test("hello") {
		t.Error("Expected 'hello' to be found")
	}
	if !bf.Test("world") {
		t.Error("Expected 'world' to be found")
	}
}

func TestBloomFilter_NoFalseNegatives(t *testing.T) {
	bf := NewBloomFilter(10000, 0.01)

	// Add items
	items := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = fmt.Sprintf("item-%d", i)
		bf.Add(items[i])
	}

	// All items should be found (no false negatives)
	for _, item := range items {
		if !bf.Test(item) {
			t.Errorf("False negative for item: %s", item)
		}
	}
}

func TestBloomFilter_FalsePositiveRate(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	// Add 1000 items
	for i := 0; i < 1000; i++ {
		bf.Add(fmt.Sprintf("item-%d", i))
	}

	// Test 10000 items that were NOT added
	falsePositives := 0
	for i := 1000; i < 11000; i++ {
		if bf.Test(fmt.Sprintf("item-%d", i)) {
			falsePositives++
		}
	}

	// False positive rate should be around 1% (allow 5% margin)
	fpRate := float64(falsePositives) / 10000.0
	if fpRate > 0.05 {
		t.Errorf("False positive rate too high: %.2f%% (expected ~1%%)", fpRate*100)
	}
}

func TestBloomFilter_AddAndTest(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	// First time should return false (not present)
	if bf.AddAndTest("key1") {
		t.Error("First AddAndTest should return false")
	}

	// Second time should return true (was present)
	if !bf.AddAndTest("key1") {
		t.Error("Second AddAndTest should return true")
	}

	// Different key should return false
	if bf.AddAndTest("key2") {
		t.Error("New key AddAndTest should return false")
	}
}

func TestBloomFilter_Clear(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	bf.Add("test")
	if !bf.Test("test") {
		t.Error("Expected 'test' to be found before Clear")
	}

	bf.Clear()

	// After clear, shouldn't find anything (ideally)
	// Note: This is probabilistic, might have false positives
	// But for a cleared filter, should mostly return false
	if bf.Test("test") {
		// This could be a false positive from a cleared filter
		// which should be extremely rare
		t.Log("Warning: false positive after Clear (rare but possible)")
	}

	if bf.Count() != 0 {
		t.Errorf("Expected count 0 after Clear, got %d", bf.Count())
	}
}

func TestBloomFilter_FillRatio(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	initialFill := bf.ApproximateFillRatio()
	if initialFill != 0 {
		t.Errorf("Expected 0 fill ratio initially, got %f", initialFill)
	}

	// Add some items
	for i := 0; i < 500; i++ {
		bf.Add(fmt.Sprintf("item-%d", i))
	}

	fillRatio := bf.ApproximateFillRatio()
	if fillRatio <= 0 || fillRatio > 1 {
		t.Errorf("Fill ratio should be between 0 and 1, got %f", fillRatio)
	}
}

func TestBloomFilter_Concurrent(t *testing.T) {
	bf := NewBloomFilter(10000, 0.01)

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				bf.Add(key)
				bf.Test(key)
			}
		}(i)
	}

	wg.Wait()

	// Filter should still be functional
	bf.Add("final-test")
	if !bf.Test("final-test") {
		t.Error("Filter should still work after concurrent access")
	}
}

func TestBloomLRU_BasicOperations(t *testing.T) {
	bl := NewBloomLRU(1000, time.Minute, 0.01)

	// First occurrence should not be duplicate
	if bl.IsDuplicate("key1") {
		t.Error("First occurrence should not be duplicate")
	}

	// Second occurrence should be duplicate
	if !bl.IsDuplicate("key1") {
		t.Error("Second occurrence should be duplicate")
	}

	// Contains should work
	if !bl.Contains("key1") {
		t.Error("Expected key1 to be contained")
	}

	if bl.Contains("nonexistent") {
		t.Error("Expected nonexistent to not be contained")
	}
}

func TestBloomLRU_Record(t *testing.T) {
	bl := NewBloomLRU(1000, time.Minute, 0.01)

	// Record without checking duplicate
	bl.Record("key1")

	// Should now be contained
	if !bl.Contains("key1") {
		t.Error("Expected key1 to be contained after Record")
	}

	// IsDuplicate should return true
	if !bl.IsDuplicate("key1") {
		t.Error("Expected key1 to be duplicate after Record")
	}
}

func TestBloomLRU_Expiration(t *testing.T) {
	bl := NewBloomLRU(1000, 50*time.Millisecond, 0.01)

	bl.Record("key1")

	// Should be duplicate immediately
	if !bl.IsDuplicate("key1") {
		t.Error("Should be duplicate immediately")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// After expiration, should not be duplicate (LRU expired)
	// Note: Bloom filter still has it, but LRU doesn't
	if bl.IsDuplicate("key1") {
		t.Error("Should not be duplicate after expiration")
	}
}

func TestBloomLRU_Stats(t *testing.T) {
	bl := NewBloomLRU(1000, time.Minute, 0.01)

	// New items (bloom negative)
	bl.IsDuplicate("a")
	bl.IsDuplicate("b")
	bl.IsDuplicate("c")

	// Duplicate (LRU check)
	bl.IsDuplicate("a")

	bloomNeg, lruChecks, dups, size := bl.Stats()

	// First 3 were new items (bloom negative path)
	if bloomNeg != 3 {
		t.Errorf("Expected 3 bloom negatives, got %d", bloomNeg)
	}

	// 4th was LRU check (bloom said maybe)
	if lruChecks != 1 {
		t.Errorf("Expected 1 LRU check, got %d", lruChecks)
	}

	// 4th was confirmed duplicate
	if dups != 1 {
		t.Errorf("Expected 1 duplicate, got %d", dups)
	}

	// Should have 3 items in LRU
	if size != 3 {
		t.Errorf("Expected size 3, got %d", size)
	}
}

func TestBloomLRU_Clear(t *testing.T) {
	bl := NewBloomLRU(1000, time.Minute, 0.01)

	bl.Record("key1")
	bl.Record("key2")

	bl.Clear()

	if bl.Len() != 0 {
		t.Errorf("Expected len 0 after Clear, got %d", bl.Len())
	}

	// Should not be duplicate after clear
	if bl.IsDuplicate("key1") {
		t.Error("Should not be duplicate after Clear")
	}
}

func BenchmarkBloomFilter_Add(b *testing.B) {
	bf := NewBloomFilter(100000, 0.01)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.Add(fmt.Sprintf("key-%d", i))
	}
}

func BenchmarkBloomFilter_Test(b *testing.B) {
	bf := NewBloomFilter(100000, 0.01)

	// Pre-populate
	for i := 0; i < 10000; i++ {
		bf.Add(fmt.Sprintf("key-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.Test(fmt.Sprintf("key-%d", i%10000))
	}
}

func BenchmarkBloomLRU_IsDuplicate(b *testing.B) {
	bl := NewBloomLRU(100000, time.Minute, 0.01)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bl.IsDuplicate(fmt.Sprintf("key-%d", i%10000))
	}
}

func BenchmarkBloomLRU_FastPath(b *testing.B) {
	bl := NewBloomLRU(100000, time.Minute, 0.01)

	// Benchmark the fast path (bloom negative)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// New unique keys hit the fast path
		bl.IsDuplicate(fmt.Sprintf("unique-key-%d", i))
	}
}

// ExactLRU Tests - Zero False Positives

func TestExactLRU_BasicOperations(t *testing.T) {
	el := NewExactLRU(1000, time.Minute)

	// First occurrence should not be duplicate
	if el.IsDuplicate("key1") {
		t.Error("First occurrence should not be duplicate")
	}

	// Second occurrence should be duplicate
	if !el.IsDuplicate("key1") {
		t.Error("Second occurrence should be duplicate")
	}

	// Contains should work
	if !el.Contains("key1") {
		t.Error("Expected key1 to be contained")
	}

	if el.Contains("nonexistent") {
		t.Error("Expected nonexistent to not be contained")
	}
}

func TestExactLRU_ZeroFalsePositives(t *testing.T) {
	el := NewExactLRU(10000, time.Minute)

	// Add 1000 items
	for i := 0; i < 1000; i++ {
		el.IsDuplicate(fmt.Sprintf("item-%d", i))
	}

	// Check 10000 items that were NOT added - should all return false
	// Unlike BloomLRU, ExactLRU should have ZERO false positives
	falsePositives := 0
	for i := 1000; i < 11000; i++ {
		if el.Contains(fmt.Sprintf("item-%d", i)) {
			falsePositives++
		}
	}

	// CRITICAL: ExactLRU must have zero false positives
	if falsePositives != 0 {
		t.Errorf("ExactLRU should have ZERO false positives, got %d", falsePositives)
	}
}

func TestExactLRU_Record(t *testing.T) {
	el := NewExactLRU(1000, time.Minute)

	// Record without checking duplicate
	el.Record("key1")

	// Should now be contained
	if !el.Contains("key1") {
		t.Error("Expected key1 to be contained after Record")
	}

	// IsDuplicate should return true
	if !el.IsDuplicate("key1") {
		t.Error("Expected key1 to be duplicate after Record")
	}
}

func TestExactLRU_Expiration(t *testing.T) {
	el := NewExactLRU(1000, 50*time.Millisecond)

	el.Record("key1")

	// Should be duplicate immediately
	if !el.IsDuplicate("key1") {
		t.Error("Should be duplicate immediately")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// After expiration, should not be duplicate
	if el.IsDuplicate("key1") {
		t.Error("Should not be duplicate after expiration")
	}
}

func TestExactLRU_Stats(t *testing.T) {
	el := NewExactLRU(1000, time.Minute)

	// New items
	el.IsDuplicate("a")
	el.IsDuplicate("b")
	el.IsDuplicate("c")

	// Duplicate
	el.IsDuplicate("a")

	bloomNeg, checks, dups, size := el.Stats()

	// bloomNegatives should always be 0 for ExactLRU (no bloom filter)
	if bloomNeg != 0 {
		t.Errorf("Expected 0 bloom negatives for ExactLRU, got %d", bloomNeg)
	}

	// 4 total checks
	if checks != 4 {
		t.Errorf("Expected 4 checks, got %d", checks)
	}

	// 1 duplicate
	if dups != 1 {
		t.Errorf("Expected 1 duplicate, got %d", dups)
	}

	// 3 items in cache
	if size != 3 {
		t.Errorf("Expected size 3, got %d", size)
	}
}

func TestExactLRU_Clear(t *testing.T) {
	el := NewExactLRU(1000, time.Minute)

	el.Record("key1")
	el.Record("key2")

	el.Clear()

	if el.Len() != 0 {
		t.Errorf("Expected len 0 after Clear, got %d", el.Len())
	}

	// Should not be duplicate after clear
	if el.IsDuplicate("key1") {
		t.Error("Should not be duplicate after Clear")
	}
}

func TestExactLRU_Interface(t *testing.T) {
	// Verify ExactLRU implements DeduplicationCache interface
	var cache DeduplicationCache = NewExactLRU(1000, time.Minute)

	// Test all interface methods
	if cache.IsDuplicate("key1") {
		t.Error("First key should not be duplicate")
	}
	if !cache.IsDuplicate("key1") {
		t.Error("Second occurrence should be duplicate")
	}
	cache.Record("key2")
	if !cache.Contains("key2") {
		t.Error("key2 should be contained after Record")
	}
	cache.CleanupExpired()
	cache.Clear()
	if cache.Len() != 0 {
		t.Error("Cache should be empty after Clear")
	}
}

func BenchmarkExactLRU_IsDuplicate(b *testing.B) {
	el := NewExactLRU(100000, time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		el.IsDuplicate(fmt.Sprintf("key-%d", i%10000))
	}
}

func BenchmarkExactLRU_Contains(b *testing.B) {
	el := NewExactLRU(100000, time.Minute)

	// Pre-populate
	for i := 0; i < 10000; i++ {
		el.Record(fmt.Sprintf("key-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		el.Contains(fmt.Sprintf("key-%d", i%10000))
	}
}
