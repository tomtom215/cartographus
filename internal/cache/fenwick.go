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

// FenwickTree (Binary Indexed Tree) provides O(log n) range sum queries and updates.
// It's ideal for temporal analytics where we need efficient:
//   - Point updates: Update count for a specific time bucket
//   - Range queries: Sum counts across a time range
//   - Prefix sums: Total count up to a specific time
//
// Time Complexity:
//   - Update: O(log n)
//   - Range Query: O(log n)
//   - Point Query: O(log n)
//
// Compared to array-based aggregation:
//   - Array update: O(1), but range query: O(n)
//   - FenwickTree: O(log n) for both
//
// Use cases:
//   - Temporal heatmaps (playback events per hour/day)
//   - Trend analysis (cumulative views over time)
//   - Rolling window statistics
type FenwickTree struct {
	mu   sync.RWMutex
	tree []int64 // 1-indexed for cleaner bit manipulation
	n    int     // Number of elements (buckets)
}

// NewFenwickTree creates a new Fenwick Tree with n buckets.
// Each bucket can represent a time unit (hour, day, etc.).
func NewFenwickTree(n int) *FenwickTree {
	if n <= 0 {
		n = 1
	}
	return &FenwickTree{
		tree: make([]int64, n+1), // 1-indexed
		n:    n,
	}
}

// Update adds delta to the value at index i (0-indexed).
// Time complexity: O(log n)
func (ft *FenwickTree) Update(i int, delta int64) {
	if i < 0 || i >= ft.n {
		return
	}

	ft.mu.Lock()
	defer ft.mu.Unlock()

	i++ // Convert to 1-indexed
	for i <= ft.n {
		ft.tree[i] += delta
		i += i & (-i) // Add last set bit
	}
}

// PrefixSum returns the sum of elements from index 0 to i (inclusive, 0-indexed).
// Time complexity: O(log n)
func (ft *FenwickTree) PrefixSum(i int) int64 {
	if i < 0 {
		return 0
	}
	if i >= ft.n {
		i = ft.n - 1
	}

	ft.mu.RLock()
	defer ft.mu.RUnlock()

	i++ // Convert to 1-indexed
	var sum int64
	for i > 0 {
		sum += ft.tree[i]
		i -= i & (-i) // Remove last set bit
	}
	return sum
}

// RangeSum returns the sum of elements from index left to right (inclusive, 0-indexed).
// Time complexity: O(log n)
func (ft *FenwickTree) RangeSum(left, right int) int64 {
	if left < 0 {
		left = 0
	}
	if right >= ft.n {
		right = ft.n - 1
	}
	if left > right {
		return 0
	}

	if left == 0 {
		return ft.PrefixSum(right)
	}
	return ft.PrefixSum(right) - ft.PrefixSum(left-1)
}

// Get returns the value at index i (0-indexed).
// Time complexity: O(log n)
func (ft *FenwickTree) Get(i int) int64 {
	if i < 0 || i >= ft.n {
		return 0
	}
	return ft.RangeSum(i, i)
}

// Set sets the value at index i to val (0-indexed).
// Time complexity: O(log n)
func (ft *FenwickTree) Set(i int, val int64) {
	current := ft.Get(i)
	ft.Update(i, val-current)
}

// Size returns the number of buckets.
func (ft *FenwickTree) Size() int {
	return ft.n
}

// Total returns the sum of all elements.
// Time complexity: O(log n)
func (ft *FenwickTree) Total() int64 {
	return ft.PrefixSum(ft.n - 1)
}

// Clear resets all values to zero.
func (ft *FenwickTree) Clear() {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	for i := range ft.tree {
		ft.tree[i] = 0
	}
}

// TemporalFenwickTree extends FenwickTree with time-bucket mapping.
// It automatically maps timestamps to bucket indices based on a configurable
// bucket size (e.g., 1 hour, 1 day).
type TemporalFenwickTree struct {
	tree       *FenwickTree
	bucketSize time.Duration // Size of each bucket (e.g., 1 hour)
	startTime  time.Time     // Start time for index calculation
}

// NewTemporalFenwickTree creates a tree for time-based aggregation.
//
// Parameters:
//   - startTime: The earliest time to track
//   - endTime: The latest time to track
//   - bucketSize: Duration of each bucket (e.g., time.Hour for hourly buckets)
//
// Example:
//
//	// Create hourly buckets for the last 7 days
//	tree := NewTemporalFenwickTree(
//	    time.Now().AddDate(0, 0, -7),
//	    time.Now(),
//	    time.Hour,
//	)
//	tree.Increment(eventTime) // Add event
//	count := tree.RangeSumTime(dayStart, dayEnd) // Query day's events
func NewTemporalFenwickTree(startTime, endTime time.Time, bucketSize time.Duration) *TemporalFenwickTree {
	if bucketSize <= 0 {
		bucketSize = time.Hour
	}

	// Calculate number of buckets
	duration := endTime.Sub(startTime)
	numBuckets := int(duration/bucketSize) + 1

	if numBuckets <= 0 {
		numBuckets = 1
	}

	return &TemporalFenwickTree{
		tree:       NewFenwickTree(numBuckets),
		bucketSize: bucketSize,
		startTime:  startTime,
	}
}

// timeToIndex converts a timestamp to a bucket index.
func (tft *TemporalFenwickTree) timeToIndex(t time.Time) int {
	if t.Before(tft.startTime) {
		return 0
	}
	return int(t.Sub(tft.startTime) / tft.bucketSize)
}

// indexToTime converts a bucket index to the start time of that bucket.
func (tft *TemporalFenwickTree) indexToTime(i int) time.Time {
	return tft.startTime.Add(time.Duration(i) * tft.bucketSize)
}

// Increment adds 1 to the bucket containing the given timestamp.
func (tft *TemporalFenwickTree) Increment(t time.Time) {
	tft.Add(t, 1)
}

// Add adds delta to the bucket containing the given timestamp.
func (tft *TemporalFenwickTree) Add(t time.Time, delta int64) {
	idx := tft.timeToIndex(t)
	if idx >= 0 && idx < tft.tree.Size() {
		tft.tree.Update(idx, delta)
	}
}

// Get returns the count for the bucket containing the given timestamp.
func (tft *TemporalFenwickTree) Get(t time.Time) int64 {
	idx := tft.timeToIndex(t)
	return tft.tree.Get(idx)
}

// RangeSumTime returns the sum of counts between start and end (inclusive).
func (tft *TemporalFenwickTree) RangeSumTime(start, end time.Time) int64 {
	startIdx := tft.timeToIndex(start)
	endIdx := tft.timeToIndex(end)
	return tft.tree.RangeSum(startIdx, endIdx)
}

// PrefixSumTime returns the sum of all counts up to and including time t.
func (tft *TemporalFenwickTree) PrefixSumTime(t time.Time) int64 {
	idx := tft.timeToIndex(t)
	return tft.tree.PrefixSum(idx)
}

// Total returns the sum of all counts.
func (tft *TemporalFenwickTree) Total() int64 {
	return tft.tree.Total()
}

// BucketSize returns the size of each bucket.
func (tft *TemporalFenwickTree) BucketSize() time.Duration {
	return tft.bucketSize
}

// StartTime returns the start time of the tree.
func (tft *TemporalFenwickTree) StartTime() time.Time {
	return tft.startTime
}

// NumBuckets returns the number of buckets.
func (tft *TemporalFenwickTree) NumBuckets() int {
	return tft.tree.Size()
}

// GetBucketCounts returns all bucket values as a slice.
// Useful for generating heatmaps or histograms.
func (tft *TemporalFenwickTree) GetBucketCounts() []int64 {
	counts := make([]int64, tft.tree.Size())
	for i := 0; i < tft.tree.Size(); i++ {
		counts[i] = tft.tree.Get(i)
	}
	return counts
}

// TimeBucket represents a time bucket with its value.
type TimeBucket struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Count int64     `json:"count"`
}

// GetBuckets returns all buckets with their time ranges and values.
func (tft *TemporalFenwickTree) GetBuckets() []TimeBucket {
	buckets := make([]TimeBucket, tft.tree.Size())
	for i := 0; i < tft.tree.Size(); i++ {
		start := tft.indexToTime(i)
		buckets[i] = TimeBucket{
			Start: start,
			End:   start.Add(tft.bucketSize),
			Count: tft.tree.Get(i),
		}
	}
	return buckets
}

// GetNonZeroBuckets returns only buckets with non-zero values.
func (tft *TemporalFenwickTree) GetNonZeroBuckets() []TimeBucket {
	var buckets []TimeBucket
	for i := 0; i < tft.tree.Size(); i++ {
		count := tft.tree.Get(i)
		if count != 0 {
			start := tft.indexToTime(i)
			buckets = append(buckets, TimeBucket{
				Start: start,
				End:   start.Add(tft.bucketSize),
				Count: count,
			})
		}
	}
	return buckets
}

// Clear resets all buckets to zero.
func (tft *TemporalFenwickTree) Clear() {
	tft.tree.Clear()
}
