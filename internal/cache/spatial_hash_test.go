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

func TestSpatialHashGrid_BasicOperations(t *testing.T) {
	t.Parallel()

	grid := NewSpatialHashGrid(100) // 100km cells

	// Insert some locations
	now := time.Now()
	grid.Insert("loc1", 40.7128, -74.0060, now, "New York")
	grid.Insert("loc2", 34.0522, -118.2437, now, "Los Angeles")
	grid.Insert("loc3", 51.5074, -0.1278, now, "London")

	if grid.Size() != 3 {
		t.Errorf("Size() = %d, want 3", grid.Size())
	}

	// Get existing
	entry, found := grid.Get("loc1")
	if !found {
		t.Error("Get('loc1') should return true")
	}
	if entry.Lat != 40.7128 {
		t.Errorf("entry.Lat = %f, want 40.7128", entry.Lat)
	}

	// Get non-existing
	_, found = grid.Get("nonexistent")
	if found {
		t.Error("Get('nonexistent') should return false")
	}
}

func TestSpatialHashGrid_Update(t *testing.T) {
	t.Parallel()

	grid := NewSpatialHashGrid(100)
	now := time.Now()

	// Insert initial location
	grid.Insert("user1", 40.7128, -74.0060, now, "Initial")

	// Update location (same ID)
	grid.Insert("user1", 34.0522, -118.2437, now, "Updated")

	// Size should still be 1
	if grid.Size() != 1 {
		t.Errorf("Size() after update = %d, want 1", grid.Size())
	}

	// Get updated location
	entry, _ := grid.Get("user1")
	if entry.Lat != 34.0522 {
		t.Errorf("entry.Lat after update = %f, want 34.0522", entry.Lat)
	}
	if entry.Data != "Updated" {
		t.Errorf("entry.Data after update = %v, want 'Updated'", entry.Data)
	}
}

func TestSpatialHashGrid_Remove(t *testing.T) {
	t.Parallel()

	grid := NewSpatialHashGrid(100)
	now := time.Now()

	grid.Insert("loc1", 40.7128, -74.0060, now, nil)
	grid.Insert("loc2", 34.0522, -118.2437, now, nil)

	// Remove existing
	if !grid.Remove("loc1") {
		t.Error("Remove('loc1') should return true")
	}

	if grid.Size() != 1 {
		t.Errorf("Size() after remove = %d, want 1", grid.Size())
	}

	// Remove non-existing
	if grid.Remove("nonexistent") {
		t.Error("Remove('nonexistent') should return false")
	}
}

func TestSpatialHashGrid_QueryNearby(t *testing.T) {
	t.Parallel()

	grid := NewSpatialHashGrid(50) // 50km cells for more precision
	now := time.Now()

	// Insert NYC and nearby locations
	grid.Insert("nyc", 40.7128, -74.0060, now, "New York")
	grid.Insert("newark", 40.7357, -74.1724, now, "Newark")       // ~15km from NYC
	grid.Insert("philly", 39.9526, -75.1652, now, "Philadelphia") // ~130km from NYC
	grid.Insert("boston", 42.3601, -71.0589, now, "Boston")       // ~300km from NYC

	// Query within 50km of NYC
	results := grid.QueryNearby(40.7128, -74.0060, 50)

	// Should find NYC and Newark
	if len(results) != 2 {
		t.Errorf("QueryNearby(50km) returned %d results, want 2", len(results))
	}

	// Query within 200km of NYC
	results = grid.QueryNearby(40.7128, -74.0060, 200)

	// Should find NYC, Newark, and Philly
	if len(results) != 3 {
		t.Errorf("QueryNearby(200km) returned %d results, want 3", len(results))
	}

	// Query within 500km of NYC
	results = grid.QueryNearby(40.7128, -74.0060, 500)

	// Should find all 4
	if len(results) != 4 {
		t.Errorf("QueryNearby(500km) returned %d results, want 4", len(results))
	}
}

func TestSpatialHashGrid_QueryNearbyWithinTime(t *testing.T) {
	t.Parallel()

	grid := NewSpatialHashGrid(50)
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// Insert with different timestamps
	grid.Insert("recent", 40.7128, -74.0060, now, nil)
	grid.Insert("old", 40.7357, -74.1724, yesterday, nil)

	// Query recent only
	since := now.Add(-1 * time.Hour)
	results := grid.QueryNearbyWithinTime(40.7128, -74.0060, 50, since)

	if len(results) != 1 {
		t.Errorf("QueryNearbyWithinTime returned %d results, want 1", len(results))
	}

	if results[0].ID != "recent" {
		t.Errorf("Expected 'recent' entry, got %s", results[0].ID)
	}
}

func TestSpatialHashGrid_QueryCell(t *testing.T) {
	t.Parallel()

	grid := NewSpatialHashGrid(100)
	now := time.Now()

	// Insert locations in same cell
	grid.Insert("loc1", 40.7128, -74.0060, now, nil)
	grid.Insert("loc2", 40.7200, -74.0100, now, nil) // Same cell

	// Insert in different cell
	grid.Insert("loc3", 34.0522, -118.2437, now, nil)

	// Query cell containing NYC
	results := grid.QueryCell(40.7128, -74.0060)

	if len(results) != 2 {
		t.Errorf("QueryCell returned %d results, want 2", len(results))
	}
}

func TestSpatialHashGrid_Clear(t *testing.T) {
	t.Parallel()

	grid := NewSpatialHashGrid(100)
	now := time.Now()

	grid.Insert("loc1", 40.7128, -74.0060, now, nil)
	grid.Insert("loc2", 34.0522, -118.2437, now, nil)

	grid.Clear()

	if grid.Size() != 0 {
		t.Errorf("Size() after Clear = %d, want 0", grid.Size())
	}

	if grid.NumCells() != 0 {
		t.Errorf("NumCells() after Clear = %d, want 0", grid.NumCells())
	}
}

func TestSpatialHashGrid_CleanupBefore(t *testing.T) {
	t.Parallel()

	grid := NewSpatialHashGrid(100)
	now := time.Now()
	old := now.Add(-2 * time.Hour)

	grid.Insert("recent", 40.7128, -74.0060, now, nil)
	grid.Insert("old1", 34.0522, -118.2437, old, nil)
	grid.Insert("old2", 51.5074, -0.1278, old, nil)

	cutoff := now.Add(-1 * time.Hour)
	removed := grid.CleanupBefore(cutoff)

	if removed != 2 {
		t.Errorf("CleanupBefore removed %d, want 2", removed)
	}

	if grid.Size() != 1 {
		t.Errorf("Size() after cleanup = %d, want 1", grid.Size())
	}

	// Recent should still exist
	_, found := grid.Get("recent")
	if !found {
		t.Error("'recent' should still exist after cleanup")
	}
}

func TestSpatialHashGrid_Concurrent(t *testing.T) {
	t.Parallel()

	grid := NewSpatialHashGrid(100)

	var wg sync.WaitGroup
	numGoroutines := 50
	numOps := 100

	// Concurrent inserts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				lat := float64(id%90) - 45
				lon := float64(j%180) - 90
				grid.Insert(string(rune(id*100+j)), lat, lon, time.Now(), nil)
			}
		}(i)
	}

	// Concurrent queries
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				lat := float64(id%90) - 45
				lon := float64(j%180) - 90
				grid.QueryNearby(lat, lon, 100)
			}
		}(i)
	}

	wg.Wait()
}

func TestHaversineDistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		expectedKm             float64
		tolerance              float64
	}{
		{
			name:       "NYC to LA",
			lat1:       40.7128,
			lon1:       -74.0060,
			lat2:       34.0522,
			lon2:       -118.2437,
			expectedKm: 3935, // ~3935km
			tolerance:  50,
		},
		{
			name:       "NYC to Newark",
			lat1:       40.7128,
			lon1:       -74.0060,
			lat2:       40.7357,
			lon2:       -74.1724,
			expectedKm: 14, // ~14km
			tolerance:  2,
		},
		{
			name:       "London to Paris",
			lat1:       51.5074,
			lon1:       -0.1278,
			lat2:       48.8566,
			lon2:       2.3522,
			expectedKm: 344, // ~344km
			tolerance:  10,
		},
		{
			name:       "Same point",
			lat1:       40.7128,
			lon1:       -74.0060,
			lat2:       40.7128,
			lon2:       -74.0060,
			expectedKm: 0,
			tolerance:  0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := haversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			diff := dist - tt.expectedKm
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("haversineDistance = %.2f km, want ~%.2f km (diff: %.2f)", dist, tt.expectedKm, diff)
			}
		})
	}
}

func TestUserLocationTracker_BasicOperations(t *testing.T) {
	t.Parallel()

	tracker := NewUserLocationTracker(100)

	// Record first location
	prev := tracker.RecordLocation("user1", 40.7128, -74.0060, time.Now(), nil)
	if prev != nil {
		t.Error("First location should have no previous")
	}

	// Record second location
	prev = tracker.RecordLocation("user1", 34.0522, -118.2437, time.Now(), nil)
	if prev == nil {
		t.Fatal("Second location should have previous")
	}
	if prev.Lat != 40.7128 {
		t.Errorf("Previous lat = %f, want 40.7128", prev.Lat)
	}

	// Get last location
	last, found := tracker.GetLastLocation("user1")
	if !found {
		t.Error("GetLastLocation should find user1")
	}
	if last.Lat != 34.0522 {
		t.Errorf("Last location lat = %f, want 34.0522", last.Lat)
	}
}

func TestUserLocationTracker_MultipleUsers(t *testing.T) {
	t.Parallel()

	tracker := NewUserLocationTracker(100)
	now := time.Now()

	tracker.RecordLocation("user1", 40.7128, -74.0060, now, nil)
	tracker.RecordLocation("user2", 34.0522, -118.2437, now, nil)
	tracker.RecordLocation("user3", 51.5074, -0.1278, now, nil)

	if tracker.NumUsers() != 3 {
		t.Errorf("NumUsers() = %d, want 3", tracker.NumUsers())
	}

	// Each user should have their own last location
	loc1, _ := tracker.GetLastLocation("user1")
	loc2, _ := tracker.GetLastLocation("user2")

	if loc1.Lat == loc2.Lat {
		t.Error("Different users should have different locations")
	}
}

func TestUserLocationTracker_GetNearbyUsers(t *testing.T) {
	t.Parallel()

	tracker := NewUserLocationTracker(50)
	now := time.Now()

	// Users near NYC
	tracker.RecordLocation("user1", 40.7128, -74.0060, now, nil) // NYC
	tracker.RecordLocation("user2", 40.7357, -74.1724, now, nil) // Newark

	// User far away
	tracker.RecordLocation("user3", 34.0522, -118.2437, now, nil) // LA

	nearby := tracker.GetNearbyUsers(40.7128, -74.0060, 50)

	if len(nearby) != 2 {
		t.Errorf("GetNearbyUsers returned %d, want 2", len(nearby))
	}
}

func TestUserLocationTracker_Cleanup(t *testing.T) {
	t.Parallel()

	tracker := NewUserLocationTracker(100)
	now := time.Now()
	old := now.Add(-2 * time.Hour)

	tracker.RecordLocation("user1", 40.7128, -74.0060, now, nil)
	tracker.RecordLocation("user2", 34.0522, -118.2437, old, nil)

	removed := tracker.CleanupOldLocations(1 * time.Hour)

	if removed != 1 {
		t.Errorf("CleanupOldLocations removed %d, want 1", removed)
	}

	if tracker.Size() != 1 {
		t.Errorf("Size() after cleanup = %d, want 1", tracker.Size())
	}
}

func TestUserLocationTracker_Clear(t *testing.T) {
	t.Parallel()

	tracker := NewUserLocationTracker(100)
	now := time.Now()

	tracker.RecordLocation("user1", 40.7128, -74.0060, now, nil)
	tracker.RecordLocation("user2", 34.0522, -118.2437, now, nil)

	tracker.Clear()

	if tracker.Size() != 0 {
		t.Errorf("Size() after Clear = %d, want 0", tracker.Size())
	}

	if tracker.NumUsers() != 0 {
		t.Errorf("NumUsers() after Clear = %d, want 0", tracker.NumUsers())
	}
}

// Benchmarks

func BenchmarkSpatialHashGrid_Insert(b *testing.B) {
	grid := NewSpatialHashGrid(100)
	now := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lat := float64(i%180) - 90
		lon := float64(i%360) - 180
		grid.Insert(string(rune(i)), lat, lon, now, nil)
	}
}

func BenchmarkSpatialHashGrid_QueryNearby(b *testing.B) {
	grid := NewSpatialHashGrid(100)
	now := time.Now()

	// Populate with 10000 entries
	for i := 0; i < 10000; i++ {
		lat := float64(i%180) - 90
		lon := float64(i%360) - 180
		grid.Insert(string(rune(i)), lat, lon, now, nil)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		grid.QueryNearby(40.7128, -74.0060, 100)
	}
}

func BenchmarkSpatialHashGrid_QueryNearbyLargeRadius(b *testing.B) {
	grid := NewSpatialHashGrid(100)
	now := time.Now()

	// Populate with 10000 entries
	for i := 0; i < 10000; i++ {
		lat := float64(i%180) - 90
		lon := float64(i%360) - 180
		grid.Insert(string(rune(i)), lat, lon, now, nil)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		grid.QueryNearby(40.7128, -74.0060, 500)
	}
}

func BenchmarkUserLocationTracker_RecordLocation(b *testing.B) {
	tracker := NewUserLocationTracker(100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lat := float64(i%180) - 90
		lon := float64(i%360) - 180
		tracker.RecordLocation("user1", lat, lon, time.Now(), nil)
	}
}
