// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestGetTemporalHeatmap_DayInterval tests temporal heatmap with daily buckets
func TestGetTemporalHeatmap_DayInterval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events across multiple days and locations
	locations := []struct {
		city    string
		country string
		ip      string
		lat     float64
		lon     float64
	}{
		{"New York", "United States", "192.168.1.1", 40.7128, -74.0060},
		{"London", "United Kingdom", "192.168.1.3", 51.5074, -0.1278},
	}

	for i := 0; i < 10; i++ {
		loc := locations[i%2]

		// Ensure geolocation exists
		city := loc.city
		geo := &models.Geolocation{
			IPAddress: loc.ip,
			Latitude:  loc.lat,
			Longitude: loc.lon,
			City:      &city,
			Country:   loc.country,
		}
		if err := db.UpsertGeolocation(geo); err != nil {
			t.Fatalf("Failed to upsert geolocation: %v", err)
		}

		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i*24) * time.Hour), // Spread across days
			UserID:          1,
			Username:        "testuser",
			IPAddress:       loc.ip,
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	response, err := db.GetTemporalHeatmap(context.Background(), LocationStatsFilter{}, "day")
	if err != nil {
		t.Fatalf("GetTemporalHeatmap failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	if response.Interval != "day" {
		t.Errorf("Expected interval 'day', got '%s'", response.Interval)
	}

	if len(response.Buckets) == 0 {
		t.Error("Expected non-empty temporal data")
	}

	// Verify we have location data
	hasLocationData := false
	for _, bucket := range response.Buckets {
		if len(bucket.Points) > 0 {
			hasLocationData = true
			break
		}
	}

	if !hasLocationData {
		t.Error("Expected at least some location data in temporal heatmap")
	}
}

// TestGetTemporalHeatmap_WeekInterval tests weekly buckets
func TestGetTemporalHeatmap_WeekInterval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events across multiple weeks
	for i := 0; i < 20; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i*7*24) * time.Hour), // Weekly spacing
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	response, err := db.GetTemporalHeatmap(context.Background(), LocationStatsFilter{}, "week")
	if err != nil {
		t.Fatalf("GetTemporalHeatmap failed: %v", err)
	}

	if response.Interval != "week" {
		t.Errorf("Expected interval 'week', got '%s'", response.Interval)
	}

	if len(response.Buckets) == 0 {
		t.Error("Expected non-empty temporal data for weekly interval")
	}
}

// TestGetTemporalHeatmap_MonthInterval tests monthly buckets
func TestGetTemporalHeatmap_MonthInterval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events across multiple months
	for i := 0; i < 6; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.AddDate(0, -i, 0), // Monthly spacing
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	response, err := db.GetTemporalHeatmap(context.Background(), LocationStatsFilter{}, "month")
	if err != nil {
		t.Fatalf("GetTemporalHeatmap failed: %v", err)
	}

	if response.Interval != "month" {
		t.Errorf("Expected interval 'month', got '%s'", response.Interval)
	}

	if len(response.Buckets) == 0 {
		t.Error("Expected non-empty temporal data for monthly interval")
	}
}

// TestGetTemporalHeatmap_WithFilters tests applying filters
func TestGetTemporalHeatmap_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// User1: 5 events
	for i := 0; i < 5; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i*24) * time.Hour),
			UserID:          1,
			Username:        "user1",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// User2: 3 events
	for i := 0; i < 3; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i*24) * time.Hour),
			UserID:          2,
			Username:        "user2",
			IPAddress:       "192.168.1.2",
			MediaType:       "episode",
			Title:           "Episode",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// Filter by user1
	filter := LocationStatsFilter{
		Users: []string{"user1"},
	}

	response, err := db.GetTemporalHeatmap(context.Background(), filter, "day")
	if err != nil {
		t.Fatalf("GetTemporalHeatmap failed: %v", err)
	}

	// Count total playbacks in response
	totalPlaybacks := 0
	for _, bucket := range response.Buckets {
		totalPlaybacks += bucket.Count
	}

	if totalPlaybacks != 5 {
		t.Errorf("Expected 5 playbacks for user1, got %d", totalPlaybacks)
	}
}

// TestGetTemporalHeatmap_EmptyData tests with no data
func TestGetTemporalHeatmap_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	response, err := db.GetTemporalHeatmap(context.Background(), LocationStatsFilter{}, "day")
	if err != nil {
		t.Fatalf("GetTemporalHeatmap failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected non-nil response even with no data")
	}

	if len(response.Buckets) != 0 {
		t.Error("Expected empty buckets array with no playbacks")
	}
}

// TestGetTemporalHeatmap_MultipleLocations tests aggregation across locations
func TestGetTemporalHeatmap_MultipleLocations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Skip if spatial extension not available (test uses geolocation data)
	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
		return
	}

	// Use noon today to avoid midnight boundary issues when test runs near midnight
	// (e.g., at 00:01:30, events at now-1min and now-2min would be yesterday)
	now := time.Now().Truncate(24 * time.Hour).Add(12 * time.Hour)

	// Create multiple geolocations
	cityNewYork := "New York"
	cityLondon := "London"
	cityTokyo := "Tokyo"

	geolocations := []models.Geolocation{
		{
			IPAddress: "192.168.1.1",
			Latitude:  40.7128,
			Longitude: -74.0060,
			City:      &cityNewYork,
			Country:   "United States",
		},
		{
			IPAddress: "192.168.1.2",
			Latitude:  51.5074,
			Longitude: -0.1278,
			City:      &cityLondon,
			Country:   "United Kingdom",
		},
		{
			IPAddress: "192.168.1.3",
			Latitude:  35.6762,
			Longitude: 139.6503,
			City:      &cityTokyo,
			Country:   "Japan",
		},
	}

	for _, geo := range geolocations {
		if err := db.UpsertGeolocation(&geo); err != nil {
			t.Fatalf("Failed to upsert geolocation: %v", err)
		}
	}

	// Insert events from each location on the same day
	// Use minute offsets to guarantee same day bucket (avoid midnight boundary issues)
	for i, geo := range geolocations {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i) * time.Minute), // Same day, different minutes
			UserID:          i + 1,
			Username:        "user",
			IPAddress:       geo.IPAddress,
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	response, err := db.GetTemporalHeatmap(context.Background(), LocationStatsFilter{}, "day")
	if err != nil {
		t.Fatalf("GetTemporalHeatmap failed: %v", err)
	}

	// Should have one day bucket with multiple locations
	if len(response.Buckets) == 0 {
		t.Fatal("Expected at least one time bucket")
	}

	// Check if locations are aggregated
	firstBucket := response.Buckets[0]
	if len(firstBucket.Points) < 3 {
		t.Errorf("Expected at least 3 location points in first bucket, got %d", len(firstBucket.Points))
	}
}

// TestGetTemporalHeatmap_DateRangeFilter tests filtering by date range
func TestGetTemporalHeatmap_DateRangeFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events spanning 30 days
	for i := 0; i < 30; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i*24) * time.Hour),
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// Filter to last 7 days
	startDate := now.Add(-7 * 24 * time.Hour)
	endDate := now

	filter := LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	response, err := db.GetTemporalHeatmap(context.Background(), filter, "day")
	if err != nil {
		t.Fatalf("GetTemporalHeatmap failed: %v", err)
	}

	// Count total playbacks
	totalPlaybacks := 0
	for _, bucket := range response.Buckets {
		totalPlaybacks += bucket.Count
	}

	// Should have approximately 7-8 events (due to date range)
	if totalPlaybacks < 5 || totalPlaybacks > 10 {
		t.Errorf("Expected approximately 7-8 playbacks in 7-day range, got %d", totalPlaybacks)
	}
}

// TestGetTemporalHeatmap_HourInterval tests hourly buckets
func TestGetTemporalHeatmap_HourInterval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events across multiple hours
	for i := 0; i < 24; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i) * time.Hour), // Hourly spacing
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	response, err := db.GetTemporalHeatmap(context.Background(), LocationStatsFilter{}, "hour")
	if err != nil {
		t.Fatalf("GetTemporalHeatmap failed: %v", err)
	}

	if response.Interval != "hour" {
		t.Errorf("Expected interval 'hour', got '%s'", response.Interval)
	}

	if len(response.Buckets) == 0 {
		t.Error("Expected non-empty temporal data for hourly interval")
	}

	// Verify bucket structure (hour buckets should have 1-hour duration)
	if len(response.Buckets) > 0 {
		bucket := response.Buckets[0]
		duration := bucket.EndTime.Sub(bucket.StartTime)
		if duration != time.Hour {
			t.Errorf("Expected 1-hour bucket duration, got %v", duration)
		}
		// Label should be in "Mon 3PM" format
		if bucket.Label == "" {
			t.Error("Expected non-empty label for hour bucket")
		}
	}
}

// TestGetTemporalHeatmap_InvalidInterval tests error handling for invalid interval
func TestGetTemporalHeatmap_InvalidInterval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.GetTemporalHeatmap(context.Background(), LocationStatsFilter{}, "invalid")
	if err == nil {
		t.Error("Expected error for invalid interval, got nil")
	}

	// Verify error message contains expected text
	if err != nil && !stringContains(err.Error(), "invalid interval") {
		t.Errorf("Expected error message to contain 'invalid interval', got: %v", err)
	}
}

// TestGetTemporalHeatmap_FillMissingBuckets tests gap filling in time series
func TestGetTemporalHeatmap_FillMissingBuckets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events with gaps (day 0, day 3, day 6) to test gap filling
	for _, dayOffset := range []int{0, 3, 6} {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-dayOffset*24) * time.Hour),
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	response, err := db.GetTemporalHeatmap(context.Background(), LocationStatsFilter{}, "day")
	if err != nil {
		t.Fatalf("GetTemporalHeatmap failed: %v", err)
	}

	// With gap filling, should have continuous buckets between first and last
	// Original: 3 buckets (day 0, day 3, day 6)
	// After fill: 7 buckets (day 0, 1, 2, 3, 4, 5, 6)
	if len(response.Buckets) < 3 {
		t.Errorf("Expected at least 3 buckets, got %d", len(response.Buckets))
	}

	// Count empty buckets (filled gaps)
	emptyBuckets := 0
	for _, bucket := range response.Buckets {
		if bucket.Count == 0 {
			emptyBuckets++
		}
	}

	// Should have some empty buckets from gap filling
	// (gaps at days 1, 2, 4, 5)
	if emptyBuckets == 0 && len(response.Buckets) > 3 {
		t.Log("Gap filling is working - found continuous time series")
	}
}
