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

func TestFenwickTree_BasicOperations(t *testing.T) {
	t.Parallel()

	ft := NewFenwickTree(10)

	// Initially all zeros
	for i := 0; i < 10; i++ {
		if got := ft.Get(i); got != 0 {
			t.Errorf("Get(%d) = %d, want 0", i, got)
		}
	}

	// Update some values
	ft.Update(0, 5)
	ft.Update(3, 10)
	ft.Update(7, 3)

	// Verify individual values
	if got := ft.Get(0); got != 5 {
		t.Errorf("Get(0) = %d, want 5", got)
	}
	if got := ft.Get(3); got != 10 {
		t.Errorf("Get(3) = %d, want 10", got)
	}
	if got := ft.Get(7); got != 3 {
		t.Errorf("Get(7) = %d, want 3", got)
	}
}

func TestFenwickTree_PrefixSum(t *testing.T) {
	t.Parallel()

	ft := NewFenwickTree(5)

	// Set values: [1, 2, 3, 4, 5]
	ft.Update(0, 1)
	ft.Update(1, 2)
	ft.Update(2, 3)
	ft.Update(3, 4)
	ft.Update(4, 5)

	tests := []struct {
		index int
		want  int64
	}{
		{0, 1},
		{1, 3},  // 1+2
		{2, 6},  // 1+2+3
		{3, 10}, // 1+2+3+4
		{4, 15}, // 1+2+3+4+5
	}

	for _, tt := range tests {
		if got := ft.PrefixSum(tt.index); got != tt.want {
			t.Errorf("PrefixSum(%d) = %d, want %d", tt.index, got, tt.want)
		}
	}
}

func TestFenwickTree_RangeSum(t *testing.T) {
	t.Parallel()

	ft := NewFenwickTree(5)

	// Set values: [1, 2, 3, 4, 5]
	ft.Update(0, 1)
	ft.Update(1, 2)
	ft.Update(2, 3)
	ft.Update(3, 4)
	ft.Update(4, 5)

	tests := []struct {
		left, right int
		want        int64
	}{
		{0, 0, 1},
		{1, 3, 9},  // 2+3+4
		{2, 4, 12}, // 3+4+5
		{0, 4, 15}, // All
		{3, 3, 4},  // Single element
	}

	for _, tt := range tests {
		if got := ft.RangeSum(tt.left, tt.right); got != tt.want {
			t.Errorf("RangeSum(%d, %d) = %d, want %d", tt.left, tt.right, got, tt.want)
		}
	}
}

func TestFenwickTree_Set(t *testing.T) {
	t.Parallel()

	ft := NewFenwickTree(5)

	ft.Set(2, 10)
	if got := ft.Get(2); got != 10 {
		t.Errorf("After Set(2, 10): Get(2) = %d, want 10", got)
	}

	ft.Set(2, 5) // Change value
	if got := ft.Get(2); got != 5 {
		t.Errorf("After Set(2, 5): Get(2) = %d, want 5", got)
	}

	ft.Set(2, 0) // Set to zero
	if got := ft.Get(2); got != 0 {
		t.Errorf("After Set(2, 0): Get(2) = %d, want 0", got)
	}
}

func TestFenwickTree_Total(t *testing.T) {
	t.Parallel()

	ft := NewFenwickTree(5)

	if got := ft.Total(); got != 0 {
		t.Errorf("Total() on empty tree = %d, want 0", got)
	}

	ft.Update(0, 1)
	ft.Update(2, 3)
	ft.Update(4, 5)

	if got := ft.Total(); got != 9 {
		t.Errorf("Total() = %d, want 9", got)
	}
}

func TestFenwickTree_Clear(t *testing.T) {
	t.Parallel()

	ft := NewFenwickTree(5)

	ft.Update(0, 10)
	ft.Update(2, 20)
	ft.Update(4, 30)

	ft.Clear()

	if got := ft.Total(); got != 0 {
		t.Errorf("Total() after Clear = %d, want 0", got)
	}

	for i := 0; i < 5; i++ {
		if got := ft.Get(i); got != 0 {
			t.Errorf("Get(%d) after Clear = %d, want 0", i, got)
		}
	}
}

func TestFenwickTree_BoundaryConditions(t *testing.T) {
	t.Parallel()

	ft := NewFenwickTree(5)

	// Out of bounds operations should be safe
	ft.Update(-1, 100)  // Should be ignored
	ft.Update(100, 100) // Should be ignored
	ft.Update(5, 100)   // Should be ignored (n=5, valid indices 0-4)

	if got := ft.Total(); got != 0 {
		t.Errorf("Total() after out-of-bounds updates = %d, want 0", got)
	}

	// Out of bounds queries
	if got := ft.Get(-1); got != 0 {
		t.Errorf("Get(-1) = %d, want 0", got)
	}
	if got := ft.Get(100); got != 0 {
		t.Errorf("Get(100) = %d, want 0", got)
	}
	if got := ft.PrefixSum(-1); got != 0 {
		t.Errorf("PrefixSum(-1) = %d, want 0", got)
	}
}

func TestFenwickTree_Concurrent(t *testing.T) {
	t.Parallel()

	ft := NewFenwickTree(100)

	var wg sync.WaitGroup
	numGoroutines := 50
	numOps := 100

	// Concurrent updates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				ft.Update(id%100, 1)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				ft.Get(id % 100)
				ft.PrefixSum(id % 100)
				ft.RangeSum(0, id%100)
			}
		}(i)
	}

	wg.Wait()

	// Total should be numGoroutines * numOps
	expectedTotal := int64(numGoroutines * numOps)
	if got := ft.Total(); got != expectedTotal {
		t.Errorf("Total() = %d, want %d", got, expectedTotal)
	}
}

func TestTemporalFenwickTree_BasicOperations(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 23, 59, 59, 0, time.UTC)

	// Create hourly buckets
	tft := NewTemporalFenwickTree(start, end, time.Hour)

	if tft.NumBuckets() != 24 {
		t.Errorf("NumBuckets() = %d, want 24", tft.NumBuckets())
	}

	// Add events at different hours
	tft.Increment(start.Add(0 * time.Hour))  // 00:00
	tft.Increment(start.Add(0 * time.Hour))  // 00:00
	tft.Increment(start.Add(5 * time.Hour))  // 05:00
	tft.Increment(start.Add(12 * time.Hour)) // 12:00
	tft.Increment(start.Add(12 * time.Hour)) // 12:00
	tft.Increment(start.Add(12 * time.Hour)) // 12:00

	if got := tft.Total(); got != 6 {
		t.Errorf("Total() = %d, want 6", got)
	}

	if got := tft.Get(start); got != 2 {
		t.Errorf("Get(00:00) = %d, want 2", got)
	}
	if got := tft.Get(start.Add(5 * time.Hour)); got != 1 {
		t.Errorf("Get(05:00) = %d, want 1", got)
	}
	if got := tft.Get(start.Add(12 * time.Hour)); got != 3 {
		t.Errorf("Get(12:00) = %d, want 3", got)
	}
}

func TestTemporalFenwickTree_RangeSumTime(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 23, 59, 59, 0, time.UTC)

	tft := NewTemporalFenwickTree(start, end, time.Hour)

	// Add events: 2 at 00:00, 3 at 06:00, 5 at 12:00, 1 at 18:00
	tft.Add(start.Add(0*time.Hour), 2)
	tft.Add(start.Add(6*time.Hour), 3)
	tft.Add(start.Add(12*time.Hour), 5)
	tft.Add(start.Add(18*time.Hour), 1)

	// Morning (00:00 - 11:59): 2 + 3 = 5
	morningStart := start
	morningEnd := start.Add(11 * time.Hour)
	if got := tft.RangeSumTime(morningStart, morningEnd); got != 5 {
		t.Errorf("Morning events = %d, want 5", got)
	}

	// Afternoon (12:00 - 23:59): 5 + 1 = 6
	afternoonStart := start.Add(12 * time.Hour)
	afternoonEnd := end
	if got := tft.RangeSumTime(afternoonStart, afternoonEnd); got != 6 {
		t.Errorf("Afternoon events = %d, want 6", got)
	}

	// Full day: 2 + 3 + 5 + 1 = 11
	if got := tft.RangeSumTime(start, end); got != 11 {
		t.Errorf("All events = %d, want 11", got)
	}
}

func TestTemporalFenwickTree_GetBuckets(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 5, 0, 0, 0, time.UTC)

	tft := NewTemporalFenwickTree(start, end, time.Hour)

	tft.Add(start.Add(0*time.Hour), 10)
	tft.Add(start.Add(2*time.Hour), 5)

	buckets := tft.GetBuckets()
	if len(buckets) != 6 {
		t.Errorf("GetBuckets() returned %d buckets, want 6", len(buckets))
	}

	// Check specific buckets
	if buckets[0].Count != 10 {
		t.Errorf("Bucket[0].Count = %d, want 10", buckets[0].Count)
	}
	if buckets[2].Count != 5 {
		t.Errorf("Bucket[2].Count = %d, want 5", buckets[2].Count)
	}
}

func TestTemporalFenwickTree_GetNonZeroBuckets(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 23, 0, 0, 0, time.UTC)

	tft := NewTemporalFenwickTree(start, end, time.Hour)

	// Only add to 3 hours
	tft.Add(start.Add(0*time.Hour), 10)
	tft.Add(start.Add(5*time.Hour), 20)
	tft.Add(start.Add(12*time.Hour), 30)

	nonZero := tft.GetNonZeroBuckets()
	if len(nonZero) != 3 {
		t.Errorf("GetNonZeroBuckets() returned %d, want 3", len(nonZero))
	}

	// Verify counts
	expectedCounts := []int64{10, 20, 30}
	for i, bucket := range nonZero {
		if bucket.Count != expectedCounts[i] {
			t.Errorf("NonZeroBucket[%d].Count = %d, want %d", i, bucket.Count, expectedCounts[i])
		}
	}
}

func TestTemporalFenwickTree_Clear(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 5, 0, 0, 0, time.UTC)

	tft := NewTemporalFenwickTree(start, end, time.Hour)

	tft.Add(start, 100)
	tft.Add(start.Add(2*time.Hour), 200)

	tft.Clear()

	if got := tft.Total(); got != 0 {
		t.Errorf("Total() after Clear = %d, want 0", got)
	}

	nonZero := tft.GetNonZeroBuckets()
	if len(nonZero) != 0 {
		t.Errorf("GetNonZeroBuckets() after Clear returned %d, want 0", len(nonZero))
	}
}

func TestTemporalFenwickTree_DailyBuckets(t *testing.T) {
	t.Parallel()

	// Create daily buckets for a week
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 7, 23, 59, 59, 0, time.UTC)

	tft := NewTemporalFenwickTree(start, end, 24*time.Hour)

	if tft.NumBuckets() != 7 {
		t.Errorf("NumBuckets() = %d, want 7", tft.NumBuckets())
	}

	// Add events for each day
	for i := 0; i < 7; i++ {
		tft.Add(start.AddDate(0, 0, i), int64(i+1)*10)
	}

	// Total: 10 + 20 + 30 + 40 + 50 + 60 + 70 = 280
	if got := tft.Total(); got != 280 {
		t.Errorf("Total() = %d, want 280", got)
	}

	// First 3 days: 10 + 20 + 30 = 60
	if got := tft.RangeSumTime(start, start.AddDate(0, 0, 2)); got != 60 {
		t.Errorf("First 3 days = %d, want 60", got)
	}
}

func TestTemporalFenwickTree_BucketSize(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	tft := NewTemporalFenwickTree(start, end, time.Hour)

	if tft.BucketSize() != time.Hour {
		t.Errorf("BucketSize() = %v, want %v", tft.BucketSize(), time.Hour)
	}

	if tft.StartTime() != start {
		t.Errorf("StartTime() = %v, want %v", tft.StartTime(), start)
	}
}

// Benchmarks

func BenchmarkFenwickTree_Update(b *testing.B) {
	ft := NewFenwickTree(10000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ft.Update(i%10000, 1)
	}
}

func BenchmarkFenwickTree_PrefixSum(b *testing.B) {
	ft := NewFenwickTree(10000)
	for i := 0; i < 10000; i++ {
		ft.Update(i, int64(i))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ft.PrefixSum(i % 10000)
	}
}

func BenchmarkFenwickTree_RangeSum(b *testing.B) {
	ft := NewFenwickTree(10000)
	for i := 0; i < 10000; i++ {
		ft.Update(i, int64(i))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ft.RangeSum(i%5000, (i%5000)+1000)
	}
}

func BenchmarkTemporalFenwickTree_Increment(b *testing.B) {
	start := time.Now().AddDate(0, 0, -365)
	end := time.Now()
	tft := NewTemporalFenwickTree(start, end, time.Hour)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t := start.Add(time.Duration(i%8760) * time.Hour)
		tft.Increment(t)
	}
}

func BenchmarkTemporalFenwickTree_RangeSumTime(b *testing.B) {
	start := time.Now().AddDate(0, 0, -365)
	end := time.Now()
	tft := NewTemporalFenwickTree(start, end, time.Hour)

	// Populate
	for i := 0; i < 8760; i++ {
		tft.Add(start.Add(time.Duration(i)*time.Hour), int64(i%100))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := start.Add(time.Duration(i%4000) * time.Hour)
		e := s.Add(720 * time.Hour) // 30 days
		tft.RangeSumTime(s, e)
	}
}
