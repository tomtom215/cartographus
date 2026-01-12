// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package cache

import (
	"sync"
	"time"
)

// HeapEntry represents an entry in the min-heap, keyed by timestamp.
type HeapEntry[T any] struct {
	Key       string
	Value     T
	Timestamp time.Time
	index     int // index in the heap array, used for O(log n) updates
}

// MinHeap implements a min-heap ordered by timestamp.
// It provides O(log n) operations for Push and Pop, and O(1) for Peek.
//
// This is used for:
//   - DLQ entry management (evict oldest when at capacity)
//   - Retry scheduling (get entries due for retry)
//
// The heap maintains a parallel map for O(1) key lookup.
type MinHeap[T any] struct {
	mu     sync.RWMutex
	heap   []*HeapEntry[T]
	byKey  map[string]*HeapEntry[T]
	maxLen int // maximum entries (0 = unlimited)
}

// NewMinHeap creates a new min-heap with optional maximum length.
func NewMinHeap[T any](maxLen int) *MinHeap[T] {
	return &MinHeap[T]{
		heap:   make([]*HeapEntry[T], 0),
		byKey:  make(map[string]*HeapEntry[T]),
		maxLen: maxLen,
	}
}

// Push adds an entry to the heap.
// If an entry with the same key exists, it updates the existing entry.
// Returns the evicted entry if the heap was at capacity, nil otherwise.
func (h *MinHeap[T]) Push(key string, value T, timestamp time.Time) *HeapEntry[T] {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if key already exists
	if existing, exists := h.byKey[key]; exists {
		existing.Value = value
		existing.Timestamp = timestamp
		h.fix(existing.index)
		return nil
	}

	// Create new entry
	entry := &HeapEntry[T]{
		Key:       key,
		Value:     value,
		Timestamp: timestamp,
		index:     len(h.heap),
	}

	// Add to heap and map
	h.heap = append(h.heap, entry)
	h.byKey[key] = entry
	h.bubbleUp(entry.index)

	// Evict if over capacity
	if h.maxLen > 0 && len(h.heap) > h.maxLen {
		return h.popOldest()
	}

	return nil
}

// Pop removes and returns the oldest entry (minimum timestamp).
// Returns nil if the heap is empty.
func (h *MinHeap[T]) Pop() *HeapEntry[T] {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.popOldest()
}

// Peek returns the oldest entry without removing it.
// Returns nil if the heap is empty.
func (h *MinHeap[T]) Peek() *HeapEntry[T] {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.heap) == 0 {
		return nil
	}
	return h.heap[0]
}

// Get retrieves an entry by key without removing it.
// Returns nil if not found.
func (h *MinHeap[T]) Get(key string) *HeapEntry[T] {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.byKey[key]
}

// Remove removes an entry by key.
// Returns the removed entry, or nil if not found.
func (h *MinHeap[T]) Remove(key string) *HeapEntry[T] {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, exists := h.byKey[key]
	if !exists {
		return nil
	}

	return h.removeAt(entry.index)
}

// Update updates an entry's timestamp and reorders the heap.
// Returns false if the key doesn't exist.
func (h *MinHeap[T]) Update(key string, timestamp time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, exists := h.byKey[key]
	if !exists {
		return false
	}

	entry.Timestamp = timestamp
	h.fix(entry.index)
	return true
}

// Len returns the number of entries in the heap.
func (h *MinHeap[T]) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.heap)
}

// GetBefore returns all entries with timestamps before the given time.
// Entries are not removed from the heap.
func (h *MinHeap[T]) GetBefore(t time.Time) []*HeapEntry[T] {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var entries []*HeapEntry[T]
	for _, entry := range h.heap {
		if entry.Timestamp.Before(t) {
			entries = append(entries, entry)
		}
	}
	return entries
}

// PopBefore removes and returns all entries with timestamps before the given time.
func (h *MinHeap[T]) PopBefore(t time.Time) []*HeapEntry[T] {
	h.mu.Lock()
	defer h.mu.Unlock()

	var entries []*HeapEntry[T]
	for len(h.heap) > 0 && h.heap[0].Timestamp.Before(t) {
		entries = append(entries, h.popOldest())
	}
	return entries
}

// Clear removes all entries from the heap.
func (h *MinHeap[T]) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.heap = make([]*HeapEntry[T], 0)
	h.byKey = make(map[string]*HeapEntry[T])
}

// All returns all entries in no particular order.
func (h *MinHeap[T]) All() []*HeapEntry[T] {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entries := make([]*HeapEntry[T], len(h.heap))
	copy(entries, h.heap)
	return entries
}

// Internal heap operations (must be called with lock held)

// popOldest removes and returns the minimum element.
func (h *MinHeap[T]) popOldest() *HeapEntry[T] {
	if len(h.heap) == 0 {
		return nil
	}

	return h.removeAt(0)
}

// removeAt removes the element at the given index.
func (h *MinHeap[T]) removeAt(i int) *HeapEntry[T] {
	n := len(h.heap) - 1
	entry := h.heap[i]

	// Remove from map
	delete(h.byKey, entry.Key)

	if i == n {
		// Removing last element
		h.heap = h.heap[:n]
		return entry
	}

	// Move last element to position i
	h.heap[i] = h.heap[n]
	h.heap[i].index = i
	h.heap = h.heap[:n]

	// Fix heap property
	h.fix(i)

	return entry
}

// fix maintains heap property after a timestamp change at index i.
func (h *MinHeap[T]) fix(i int) {
	// Try bubbling up
	if h.bubbleUp(i) {
		return
	}
	// If didn't bubble up, try bubbling down
	h.bubbleDown(i)
}

// bubbleUp moves element at index i up to its correct position.
// Returns true if the element moved.
func (h *MinHeap[T]) bubbleUp(i int) bool {
	moved := false
	for i > 0 {
		parent := (i - 1) / 2
		if !h.heap[i].Timestamp.Before(h.heap[parent].Timestamp) {
			break
		}
		h.swap(i, parent)
		i = parent
		moved = true
	}
	return moved
}

// bubbleDown moves element at index i down to its correct position.
func (h *MinHeap[T]) bubbleDown(i int) {
	n := len(h.heap)
	for {
		smallest := i
		left := 2*i + 1
		right := 2*i + 2

		if left < n && h.heap[left].Timestamp.Before(h.heap[smallest].Timestamp) {
			smallest = left
		}
		if right < n && h.heap[right].Timestamp.Before(h.heap[smallest].Timestamp) {
			smallest = right
		}

		if smallest == i {
			break
		}

		h.swap(i, smallest)
		i = smallest
	}
}

// swap swaps elements at indices i and j.
func (h *MinHeap[T]) swap(i, j int) {
	h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
	h.heap[i].index = i
	h.heap[j].index = j
}
