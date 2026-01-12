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

// TestInitializeCompletionBuckets tests the initializeCompletionBuckets function
func TestInitializeCompletionBuckets(t *testing.T) {

	buckets := initializeCompletionBuckets()

	// Verify we have exactly 5 buckets
	if len(buckets) != 5 {
		t.Errorf("Expected 5 buckets, got %d", len(buckets))
	}

	// Expected bucket definitions
	expected := []struct {
		bucket     string
		minPercent int
		maxPercent int
	}{
		{"0-25%", 0, 25},
		{"25-50%", 25, 50},
		{"50-75%", 50, 75},
		{"75-99%", 75, 99},
		{"100%", 100, 100},
	}

	for i, exp := range expected {
		if i >= len(buckets) {
			t.Fatalf("Missing bucket at index %d", i)
		}

		bucket := buckets[i]
		if bucket.Bucket != exp.bucket {
			t.Errorf("Bucket[%d].Bucket = %q, expected %q", i, bucket.Bucket, exp.bucket)
		}
		if bucket.MinPercent != exp.minPercent {
			t.Errorf("Bucket[%d].MinPercent = %d, expected %d", i, bucket.MinPercent, exp.minPercent)
		}
		if bucket.MaxPercent != exp.maxPercent {
			t.Errorf("Bucket[%d].MaxPercent = %d, expected %d", i, bucket.MaxPercent, exp.maxPercent)
		}

		// Verify initial values are zero
		if bucket.PlaybackCount != 0 {
			t.Errorf("Bucket[%d].PlaybackCount = %d, expected 0", i, bucket.PlaybackCount)
		}
		if bucket.AvgCompletion != 0 {
			t.Errorf("Bucket[%d].AvgCompletion = %f, expected 0", i, bucket.AvgCompletion)
		}
	}
}

// TestCalculateCompletionStats tests the calculateCompletionStats function
func TestCalculateCompletionStats(t *testing.T) {

	tests := []struct {
		name                 string
		totalPlaybacks       int
		totalCompletion      float64
		fullyWatched         int
		expectedAvg          float64
		expectedFullyWatched float64
	}{
		{
			name:                 "Zero playbacks",
			totalPlaybacks:       0,
			totalCompletion:      0,
			fullyWatched:         0,
			expectedAvg:          0,
			expectedFullyWatched: 0,
		},
		{
			name:                 "All fully watched",
			totalPlaybacks:       10,
			totalCompletion:      1000, // 10 * 100%
			fullyWatched:         10,
			expectedAvg:          100,
			expectedFullyWatched: 100,
		},
		{
			name:                 "Half fully watched",
			totalPlaybacks:       100,
			totalCompletion:      7500, // Average of 75%
			fullyWatched:         50,
			expectedAvg:          75,
			expectedFullyWatched: 50,
		},
		{
			name:                 "None fully watched",
			totalPlaybacks:       20,
			totalCompletion:      1000, // Average of 50%
			fullyWatched:         0,
			expectedAvg:          50,
			expectedFullyWatched: 0,
		},
		{
			name:                 "Single playback fully watched",
			totalPlaybacks:       1,
			totalCompletion:      100,
			fullyWatched:         1,
			expectedAvg:          100,
			expectedFullyWatched: 100,
		},
		{
			name:                 "Large dataset",
			totalPlaybacks:       10000,
			totalCompletion:      850000, // Average of 85%
			fullyWatched:         7500,
			expectedAvg:          85,
			expectedFullyWatched: 75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avgCompletion, fullyWatchedPct := calculateCompletionStats(tt.totalPlaybacks, tt.totalCompletion, tt.fullyWatched)

			if avgCompletion != tt.expectedAvg {
				t.Errorf("avgCompletion = %f, expected %f", avgCompletion, tt.expectedAvg)
			}

			if fullyWatchedPct != tt.expectedFullyWatched {
				t.Errorf("fullyWatchedPct = %f, expected %f", fullyWatchedPct, tt.expectedFullyWatched)
			}
		})
	}
}

// TestCalculateCompletionStats_DivisionByZero tests that division by zero is handled
func TestCalculateCompletionStats_DivisionByZero(t *testing.T) {

	// Ensure no panic occurs with zero totalPlaybacks
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("calculateCompletionStats panicked with zero totalPlaybacks: %v", r)
		}
	}()

	avgCompletion, fullyWatchedPct := calculateCompletionStats(0, 100, 50)

	if avgCompletion != 0 {
		t.Errorf("Expected avgCompletion = 0 with zero totalPlaybacks, got %f", avgCompletion)
	}

	if fullyWatchedPct != 0 {
		t.Errorf("Expected fullyWatchedPct = 0 with zero totalPlaybacks, got %f", fullyWatchedPct)
	}
}

// TestBuildDurationWhereClause tests the buildDurationWhereClause function
func TestBuildDurationWhereClause(t *testing.T) {

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	tests := []struct {
		name           string
		filter         LocationStatsFilter
		expectedPrefix string
		expectedArgs   int
	}{
		{
			name:           "Empty filter still has duration check",
			filter:         LocationStatsFilter{},
			expectedPrefix: " AND play_duration IS NOT NULL AND play_duration > 0",
			expectedArgs:   0,
		},
		{
			name: "StartDate only",
			filter: LocationStatsFilter{
				StartDate: &yesterday,
			},
			expectedPrefix: " AND play_duration IS NOT NULL AND play_duration > 0 AND started_at >= ?",
			expectedArgs:   1,
		},
		{
			name: "EndDate only",
			filter: LocationStatsFilter{
				EndDate: &now,
			},
			expectedPrefix: " AND play_duration IS NOT NULL AND play_duration > 0 AND started_at <= ?",
			expectedArgs:   1,
		},
		{
			name: "StartDate and EndDate",
			filter: LocationStatsFilter{
				StartDate: &yesterday,
				EndDate:   &now,
			},
			expectedPrefix: " AND play_duration IS NOT NULL AND play_duration > 0 AND started_at >= ? AND started_at <= ?",
			expectedArgs:   2,
		},
		{
			name: "Single user",
			filter: LocationStatsFilter{
				Users: []string{"user1"},
			},
			expectedPrefix: " AND play_duration IS NOT NULL AND play_duration > 0 AND username IN (?)",
			expectedArgs:   1,
		},
		{
			name: "Multiple users",
			filter: LocationStatsFilter{
				Users: []string{"user1", "user2", "user3"},
			},
			expectedPrefix: " AND play_duration IS NOT NULL AND play_duration > 0 AND username IN (?,?,?)",
			expectedArgs:   3,
		},
		{
			name: "Single media type",
			filter: LocationStatsFilter{
				MediaTypes: []string{"movie"},
			},
			expectedPrefix: " AND play_duration IS NOT NULL AND play_duration > 0 AND media_type IN (?)",
			expectedArgs:   1,
		},
		{
			name: "Multiple media types",
			filter: LocationStatsFilter{
				MediaTypes: []string{"movie", "episode"},
			},
			expectedPrefix: " AND play_duration IS NOT NULL AND play_duration > 0 AND media_type IN (?,?)",
			expectedArgs:   2,
		},
		{
			name: "All filters combined",
			filter: LocationStatsFilter{
				StartDate:  &yesterday,
				EndDate:    &now,
				Users:      []string{"user1", "user2"},
				MediaTypes: []string{"movie", "episode"},
			},
			expectedPrefix: " AND play_duration IS NOT NULL AND play_duration > 0 AND started_at >= ? AND started_at <= ? AND username IN (?,?) AND media_type IN (?,?)",
			expectedArgs:   6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args := buildDurationWhereClause(tt.filter)

			if whereClause != tt.expectedPrefix {
				t.Errorf("buildDurationWhereClause() whereClause = %q, expected %q", whereClause, tt.expectedPrefix)
			}

			if len(args) != tt.expectedArgs {
				t.Errorf("buildDurationWhereClause() args count = %d, expected %d", len(args), tt.expectedArgs)
			}
		})
	}
}

// TestGetContentCompletionStats_Success tests the full GetContentCompletionStats method
func TestGetContentCompletionStats_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events with various completion percentages
	completions := []int{10, 30, 60, 85, 100, 100, 50, 25, 75, 100}

	for i, completion := range completions {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i) * time.Hour),
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: completion,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	stats, err := db.GetContentCompletionStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetContentCompletionStats failed: %v", err)
	}

	// Verify bucket count
	if len(stats.Buckets) != 5 {
		t.Errorf("Expected 5 buckets, got %d", len(stats.Buckets))
	}

	// Verify total playbacks
	if stats.TotalPlaybacks != 10 {
		t.Errorf("Expected 10 total playbacks, got %d", stats.TotalPlaybacks)
	}

	// Verify fully watched count (3 at 100%)
	if stats.FullyWatched != 3 {
		t.Errorf("Expected 3 fully watched, got %d", stats.FullyWatched)
	}

	// Verify average completion is reasonable (sum = 10+30+60+85+100+100+50+25+75+100 = 635)
	expectedAvg := 63.5
	if stats.AvgCompletion < expectedAvg-1 || stats.AvgCompletion > expectedAvg+1 {
		t.Errorf("Expected average completion around %.1f, got %.1f", expectedAvg, stats.AvgCompletion)
	}

	// Verify fully watched percentage (3/10 = 30%)
	expectedPct := 30.0
	if stats.FullyWatchedPct < expectedPct-1 || stats.FullyWatchedPct > expectedPct+1 {
		t.Errorf("Expected fully watched percentage around %.1f, got %.1f", expectedPct, stats.FullyWatchedPct)
	}
}

// TestGetContentCompletionStats_EmptyData tests with no playback data
func TestGetContentCompletionStats_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	// No playbacks inserted

	stats, err := db.GetContentCompletionStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetContentCompletionStats failed: %v", err)
	}

	if stats.TotalPlaybacks != 0 {
		t.Errorf("Expected 0 total playbacks, got %d", stats.TotalPlaybacks)
	}

	if stats.FullyWatched != 0 {
		t.Errorf("Expected 0 fully watched, got %d", stats.FullyWatched)
	}

	if stats.AvgCompletion != 0 {
		t.Errorf("Expected 0 average completion, got %f", stats.AvgCompletion)
	}
}

// TestGetContentCompletionStats_WithFilters tests filtering for completion stats
func TestGetContentCompletionStats_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// User1: 3 events at 100%
	for i := 0; i < 3; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now,
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

	// User2: 2 events at 50%
	for i := 0; i < 2; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now,
			UserID:          2,
			Username:        "user2",
			IPAddress:       "192.168.1.2",
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: 50,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// Filter by user1 only
	filter := LocationStatsFilter{
		Users: []string{"user1"},
	}

	stats, err := db.GetContentCompletionStats(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetContentCompletionStats failed: %v", err)
	}

	if stats.TotalPlaybacks != 3 {
		t.Errorf("Expected 3 playbacks for user1, got %d", stats.TotalPlaybacks)
	}

	if stats.FullyWatched != 3 {
		t.Errorf("Expected 3 fully watched for user1, got %d", stats.FullyWatched)
	}

	if stats.FullyWatchedPct != 100 {
		t.Errorf("Expected 100%% fully watched for user1, got %.1f%%", stats.FullyWatchedPct)
	}
}

// TestGetDurationStats_Success_Helpers tests the GetDurationStats method
func TestGetDurationStats_Success_Helpers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events with various durations
	durations := []int{30, 60, 90, 120, 150}
	mediaTypes := []string{"movie", "movie", "episode", "episode", "track"}

	for i, duration := range durations {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i) * time.Hour),
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       mediaTypes[i],
			Title:           "Content",
			PlayDuration:    &duration,
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	stats, err := db.GetDurationStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetDurationStats failed: %v", err)
	}

	// Verify total duration (30+60+90+120+150 = 450)
	if stats.TotalDuration != 450 {
		t.Errorf("Expected total duration 450, got %d", stats.TotalDuration)
	}

	// Verify average duration (450/5 = 90)
	if stats.AvgDuration != 90 {
		t.Errorf("Expected average duration 90, got %d", stats.AvgDuration)
	}

	// Verify median duration (90 is the middle value)
	if stats.MedianDuration != 90 {
		t.Errorf("Expected median duration 90, got %d", stats.MedianDuration)
	}

	// Verify we have duration by media type
	if len(stats.DurationByType) == 0 {
		t.Error("Expected duration by type data")
	}
}

// TestGetDurationStats_EmptyData_Helpers tests duration stats with no data
func TestGetDurationStats_EmptyData_Helpers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	// No playbacks inserted

	stats, err := db.GetDurationStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetDurationStats failed: %v", err)
	}

	if stats.TotalDuration != 0 {
		t.Errorf("Expected 0 total duration, got %d", stats.TotalDuration)
	}

	if stats.AvgDuration != 0 {
		t.Errorf("Expected 0 average duration, got %d", stats.AvgDuration)
	}
}

// TestGetDurationStats_NullDurations tests handling of NULL play_duration values
func TestGetDurationStats_NullDurations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events with NULL durations (PlayDuration = nil)
	for i := 0; i < 3; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i) * time.Hour),
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Movie",
			PlayDuration:    nil, // NULL duration
			PercentComplete: 50,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	// Insert one event with valid duration
	validDuration := 60
	event := &models.PlaybackEvent{
		ID:              uuid.New(),
		SessionKey:      uuid.New().String(),
		StartedAt:       now,
		UserID:          1,
		Username:        "testuser",
		IPAddress:       "192.168.1.1",
		MediaType:       "movie",
		Title:           "Movie",
		PlayDuration:    &validDuration,
		PercentComplete: 100,
	}
	if err := db.InsertPlaybackEvent(event); err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	stats, err := db.GetDurationStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetDurationStats failed: %v", err)
	}

	// Should only count the one with valid duration due to buildDurationWhereClause
	if stats.TotalDuration != 60 {
		t.Errorf("Expected total duration 60 (only valid duration), got %d", stats.TotalDuration)
	}
}

// Sink variables to prevent dead code elimination in benchmarks.
// The compiler cannot optimize away assignments to package-level variables.
var (
	bucketsSink      []models.CompletionBucket
	avgCompSink      float64
	fullyWatchedSink float64
)

// BenchmarkInitializeCompletionBuckets benchmarks bucket initialization
func BenchmarkInitializeCompletionBuckets(b *testing.B) {
	var result []models.CompletionBucket
	for i := 0; i < b.N; i++ {
		result = initializeCompletionBuckets()
	}
	// Prevent dead code elimination by assigning to package-level sink
	bucketsSink = result
}

// BenchmarkCalculateCompletionStats benchmarks completion stats calculation
func BenchmarkCalculateCompletionStats(b *testing.B) {
	testCases := []struct {
		name           string
		totalPlaybacks int
		totalCompl     float64
		fullyWatched   int
	}{
		{"Small", 10, 750, 5},
		{"Medium", 1000, 75000, 500},
		{"Large", 100000, 7500000, 50000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			var avgComp, fullyWatchedPct float64
			for i := 0; i < b.N; i++ {
				avgComp, fullyWatchedPct = calculateCompletionStats(tc.totalPlaybacks, tc.totalCompl, tc.fullyWatched)
			}
			// Prevent dead code elimination
			avgCompSink = avgComp
			fullyWatchedSink = fullyWatchedPct
		})
	}
}

// Additional sink variables for WHERE clause benchmarks
var (
	whereClauseSink string
	whereArgsSink   []interface{}
)

// BenchmarkBuildDurationWhereClause benchmarks the buildDurationWhereClause function
func BenchmarkBuildDurationWhereClause(b *testing.B) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	filters := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{"Empty", LocationStatsFilter{}},
		{"DateOnly", LocationStatsFilter{StartDate: &yesterday, EndDate: &now}},
		{"WithUsers", LocationStatsFilter{
			StartDate: &yesterday,
			EndDate:   &now,
			Users:     []string{"user1", "user2", "user3"},
		}},
		{"FullFilter", LocationStatsFilter{
			StartDate:  &yesterday,
			EndDate:    &now,
			Users:      []string{"user1", "user2", "user3", "user4", "user5"},
			MediaTypes: []string{"movie", "episode", "track"},
		}},
	}

	for _, f := range filters {
		b.Run(f.name, func(b *testing.B) {
			var clause string
			var args []interface{}
			for i := 0; i < b.N; i++ {
				clause, args = buildDurationWhereClause(f.filter)
			}
			// Prevent dead code elimination
			whereClauseSink = clause
			whereArgsSink = args
		})
	}
}
