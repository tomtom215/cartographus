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

func TestMinHeap_BasicOperations(t *testing.T) {
	h := NewMinHeap[string](0)

	// Push items
	h.Push("c", "third", time.Now().Add(3*time.Second))
	h.Push("a", "first", time.Now().Add(1*time.Second))
	h.Push("b", "second", time.Now().Add(2*time.Second))

	if h.Len() != 3 {
		t.Errorf("Expected len 3, got %d", h.Len())
	}

	// Peek should return the oldest (smallest timestamp)
	oldest := h.Peek()
	if oldest == nil || oldest.Key != "a" {
		t.Errorf("Expected peek to return 'a', got %v", oldest)
	}

	// Pop should return items in timestamp order
	first := h.Pop()
	if first == nil || first.Key != "a" {
		t.Errorf("Expected pop to return 'a', got %v", first)
	}

	second := h.Pop()
	if second == nil || second.Key != "b" {
		t.Errorf("Expected pop to return 'b', got %v", second)
	}

	third := h.Pop()
	if third == nil || third.Key != "c" {
		t.Errorf("Expected pop to return 'c', got %v", third)
	}

	// Pop from empty heap
	empty := h.Pop()
	if empty != nil {
		t.Error("Expected nil from empty heap")
	}
}

func TestMinHeap_Get(t *testing.T) {
	h := NewMinHeap[int](0)

	h.Push("key1", 100, time.Now())
	h.Push("key2", 200, time.Now())

	entry := h.Get("key1")
	if entry == nil || entry.Value != 100 {
		t.Errorf("Expected to get key1 with value 100, got %v", entry)
	}

	notFound := h.Get("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for nonexistent key")
	}
}

func TestMinHeap_Remove(t *testing.T) {
	h := NewMinHeap[string](0)

	h.Push("a", "first", time.Now().Add(1*time.Second))
	h.Push("b", "second", time.Now().Add(2*time.Second))
	h.Push("c", "third", time.Now().Add(3*time.Second))

	// Remove middle item
	removed := h.Remove("b")
	if removed == nil || removed.Key != "b" {
		t.Errorf("Expected to remove 'b', got %v", removed)
	}

	if h.Len() != 2 {
		t.Errorf("Expected len 2 after remove, got %d", h.Len())
	}

	// Verify remaining items are correct
	first := h.Pop()
	if first == nil || first.Key != "a" {
		t.Errorf("Expected 'a' first, got %v", first)
	}

	second := h.Pop()
	if second == nil || second.Key != "c" {
		t.Errorf("Expected 'c' second, got %v", second)
	}
}

func TestMinHeap_Update(t *testing.T) {
	h := NewMinHeap[string](0)

	baseTime := time.Now()
	h.Push("a", "first", baseTime.Add(1*time.Second))
	h.Push("b", "second", baseTime.Add(2*time.Second))
	h.Push("c", "third", baseTime.Add(3*time.Second))

	// Update 'c' to have the oldest timestamp
	if !h.Update("c", baseTime) {
		t.Error("Expected Update to return true for existing key")
	}

	// 'c' should now be the oldest
	oldest := h.Peek()
	if oldest == nil || oldest.Key != "c" {
		t.Errorf("Expected 'c' to be oldest after update, got %v", oldest)
	}

	// Update nonexistent key should return false
	if h.Update("nonexistent", baseTime) {
		t.Error("Expected Update to return false for nonexistent key")
	}
}

func TestMinHeap_MaxLen(t *testing.T) {
	h := NewMinHeap[string](3)

	baseTime := time.Now()
	h.Push("a", "first", baseTime.Add(1*time.Second))
	h.Push("b", "second", baseTime.Add(2*time.Second))
	h.Push("c", "third", baseTime.Add(3*time.Second))

	// Adding 4th item should evict the oldest (a)
	evicted := h.Push("d", "fourth", baseTime.Add(4*time.Second))
	if evicted == nil || evicted.Key != "a" {
		t.Errorf("Expected 'a' to be evicted, got %v", evicted)
	}

	if h.Len() != 3 {
		t.Errorf("Expected len 3, got %d", h.Len())
	}

	// 'a' should not be found
	if h.Get("a") != nil {
		t.Error("Expected 'a' to be evicted")
	}
}

func TestMinHeap_GetBefore(t *testing.T) {
	h := NewMinHeap[string](0)

	baseTime := time.Now()
	h.Push("a", "first", baseTime)
	h.Push("b", "second", baseTime.Add(1*time.Minute))
	h.Push("c", "third", baseTime.Add(2*time.Minute))

	// Get entries before 90 seconds from base
	cutoff := baseTime.Add(90 * time.Second)
	entries := h.GetBefore(cutoff)

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries before cutoff, got %d", len(entries))
	}

	// Should still have all items (GetBefore doesn't remove)
	if h.Len() != 3 {
		t.Errorf("Expected len 3 after GetBefore, got %d", h.Len())
	}
}

func TestMinHeap_PopBefore(t *testing.T) {
	h := NewMinHeap[string](0)

	baseTime := time.Now()
	h.Push("a", "first", baseTime)
	h.Push("b", "second", baseTime.Add(1*time.Minute))
	h.Push("c", "third", baseTime.Add(2*time.Minute))

	// Pop entries before 90 seconds from base
	cutoff := baseTime.Add(90 * time.Second)
	entries := h.PopBefore(cutoff)

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries popped, got %d", len(entries))
	}

	// Should only have 'c' remaining
	if h.Len() != 1 {
		t.Errorf("Expected len 1 after PopBefore, got %d", h.Len())
	}

	remaining := h.Pop()
	if remaining == nil || remaining.Key != "c" {
		t.Errorf("Expected 'c' remaining, got %v", remaining)
	}
}

func TestMinHeap_All(t *testing.T) {
	h := NewMinHeap[int](0)

	h.Push("a", 1, time.Now())
	h.Push("b", 2, time.Now())
	h.Push("c", 3, time.Now())

	all := h.All()
	if len(all) != 3 {
		t.Errorf("Expected 3 entries from All, got %d", len(all))
	}

	// Verify all keys are present
	keys := make(map[string]bool)
	for _, entry := range all {
		keys[entry.Key] = true
	}

	if !keys["a"] || !keys["b"] || !keys["c"] {
		t.Error("Expected all keys to be present in All()")
	}
}

func TestMinHeap_Clear(t *testing.T) {
	h := NewMinHeap[string](0)

	h.Push("a", "first", time.Now())
	h.Push("b", "second", time.Now())

	h.Clear()

	if h.Len() != 0 {
		t.Errorf("Expected len 0 after Clear, got %d", h.Len())
	}

	if h.Get("a") != nil {
		t.Error("Expected no entries after Clear")
	}
}

func TestMinHeap_Concurrent(t *testing.T) {
	h := NewMinHeap[int](0)

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				h.Push(key, id*j, time.Now().Add(time.Duration(j)*time.Millisecond))
				h.Get(key)
				h.Len()
			}
		}(i)
	}

	wg.Wait()

	// Heap should still be functional
	h.Push("final", 999, time.Now())
	if h.Get("final") == nil {
		t.Error("Heap should still work after concurrent access")
	}
}

func TestMinHeap_UpdateExisting(t *testing.T) {
	h := NewMinHeap[string](0)

	baseTime := time.Now()
	h.Push("a", "value1", baseTime)

	// Push same key with different value and time
	evicted := h.Push("a", "value2", baseTime.Add(time.Hour))
	if evicted != nil {
		t.Error("Expected no eviction when updating existing key")
	}

	if h.Len() != 1 {
		t.Errorf("Expected len 1 after update, got %d", h.Len())
	}

	entry := h.Get("a")
	if entry == nil || entry.Value != "value2" {
		t.Errorf("Expected updated value 'value2', got %v", entry)
	}
}

func BenchmarkMinHeap_Push(b *testing.B) {
	h := NewMinHeap[int](0)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Push(fmt.Sprintf("key-%d", i), i, now.Add(time.Duration(i)*time.Millisecond))
	}
}

func BenchmarkMinHeap_Pop(b *testing.B) {
	h := NewMinHeap[int](0)
	now := time.Now()

	// Pre-populate
	for i := 0; i < b.N; i++ {
		h.Push(fmt.Sprintf("key-%d", i), i, now.Add(time.Duration(i)*time.Millisecond))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Pop()
	}
}

func BenchmarkMinHeap_PushWithEviction(b *testing.B) {
	h := NewMinHeap[int](100)
	now := time.Now()

	// Pre-fill to capacity
	for i := 0; i < 100; i++ {
		h.Push(fmt.Sprintf("key-%d", i), i, now.Add(time.Duration(i)*time.Millisecond))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Push(fmt.Sprintf("new-key-%d", i), i, now.Add(time.Duration(i)*time.Millisecond))
	}
}
