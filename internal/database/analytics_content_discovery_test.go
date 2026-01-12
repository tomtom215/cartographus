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
)

// TestGetContentDiscoveryAnalytics tests the main GetContentDiscoveryAnalytics function
func TestGetContentDiscoveryAnalytics(t *testing.T) {
	db := testDBWithData(t, insertContentDiscoveryTestData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{"default filter", LocationStatsFilter{}},
		{"with date filter", LocationStatsFilter{StartDate: timePtrOffset(-30 * 24 * time.Hour)}},
		{"with user filter", LocationStatsFilter{Users: []string{"user1", "user2"}}},
		{"with media type filter", LocationStatsFilter{MediaTypes: []string{"movie"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := db.GetContentDiscoveryAnalytics(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("GetContentDiscoveryAnalytics() error = %v", err)
			}
			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			// Validate summary structure
			assertNonNegative(t, "TotalContentWithAddedAt", result.Summary.TotalContentWithAddedAt)
			assertNonNegative(t, "TotalDiscovered", result.Summary.TotalDiscovered)
			assertNonNegative(t, "TotalNeverWatched", result.Summary.TotalNeverWatched)
			assertNonNegativeFloat(t, "AvgTimeToDiscoveryHours", result.Summary.AvgTimeToDiscoveryHours)

			// Validate metadata
			if result.Metadata.ExecutionTimeMS < 0 {
				t.Error("ExecutionTimeMS should not be negative")
			}
			if result.Metadata.QueryHash == "" {
				t.Error("QueryHash should not be empty")
			}
			if result.Metadata.EarlyDiscoveryThresholdHours != earlyDiscoveryThresholdHours {
				t.Errorf("EarlyDiscoveryThresholdHours should be %d", earlyDiscoveryThresholdHours)
			}
		})
	}
}

// TestGetContentDiscoverySummary tests the getContentDiscoverySummary function
func TestGetContentDiscoverySummary(t *testing.T) {
	db := testDBWithData(t, insertContentDiscoveryTestData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{"empty filter", LocationStatsFilter{}},
		{"with date filter", LocationStatsFilter{StartDate: timePtrOffset(-14 * 24 * time.Hour)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, err := db.getContentDiscoverySummary(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("getContentDiscoverySummary() error = %v", err)
			}
			if summary == nil {
				t.Fatal("Expected non-nil summary")
			}

			assertNonNegative(t, "TotalContentWithAddedAt", summary.TotalContentWithAddedAt)
			assertNonNegative(t, "TotalDiscovered", summary.TotalDiscovered)
			assertNonNegative(t, "TotalNeverWatched", summary.TotalNeverWatched)
			assertNonNegativeFloat(t, "AvgTimeToDiscoveryHours", summary.AvgTimeToDiscoveryHours)
			assertNonNegativeFloat(t, "MedianTimeToDiscoveryHours", summary.MedianTimeToDiscoveryHours)
			assertNonNegativeFloat(t, "FastestDiscoveryHours", summary.FastestDiscoveryHours)
			assertNonNegative(t, "SlowestDiscoveryDays", summary.SlowestDiscoveryDays)

			// Rates should be 0-100
			if summary.OverallDiscoveryRate < 0 || summary.OverallDiscoveryRate > 100 {
				t.Errorf("OverallDiscoveryRate should be 0-100, got %f", summary.OverallDiscoveryRate)
			}
			if summary.EarlyDiscoveryRate < 0 || summary.EarlyDiscoveryRate > 100 {
				t.Errorf("EarlyDiscoveryRate should be 0-100, got %f", summary.EarlyDiscoveryRate)
			}
		})
	}
}

// TestGetDiscoveryTimeBuckets tests the getDiscoveryTimeBuckets function
func TestGetDiscoveryTimeBuckets(t *testing.T) {
	db := testDBWithData(t, insertContentDiscoveryTestData)
	defer db.Close()

	buckets, err := db.getDiscoveryTimeBuckets(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("getDiscoveryTimeBuckets() error = %v", err)
	}

	// Should have 5 buckets: 0-24h, 1-7d, 7-30d, 30-90d, 90d+
	expectedBuckets := []string{"0-24h", "1-7d", "7-30d", "30-90d", "90d+"}
	bucketNames := make(map[string]bool)
	for _, b := range buckets {
		bucketNames[b.Bucket] = true
		assertNonNegative(t, "ContentCount", b.ContentCount)
		if b.Percentage < 0 || b.Percentage > 100 {
			t.Errorf("Percentage should be 0-100, got %f", b.Percentage)
		}
	}

	for _, expected := range expectedBuckets {
		if !bucketNames[expected] {
			t.Errorf("Missing expected bucket: %s", expected)
		}
	}
}

// TestGetEarlyAdopters tests the getEarlyAdopters function
func TestGetEarlyAdopters(t *testing.T) {
	db := testDBWithData(t, insertContentDiscoveryTestData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
		limit  int
	}{
		{"default limit", LocationStatsFilter{}, 20},
		{"small limit", LocationStatsFilter{}, 5},
		{"zero limit", LocationStatsFilter{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adopters, err := db.getEarlyAdopters(context.Background(), tt.filter, tt.limit)
			if err != nil {
				t.Fatalf("getEarlyAdopters() error = %v", err)
			}

			for _, a := range adopters {
				if a.UserID == 0 && a.Username == "" {
					t.Error("EarlyAdopter should have UserID or Username")
				}
				assertNonNegative(t, "EarlyDiscoveryCount", a.EarlyDiscoveryCount)
				assertNonNegative(t, "TotalDiscoveries", a.TotalDiscoveries)
				assertNonNegativeFloat(t, "AvgTimeToDiscoveryHours", a.AvgTimeToDiscoveryHours)

				if a.EarlyDiscoveryRate < 0 || a.EarlyDiscoveryRate > 100 {
					t.Errorf("EarlyDiscoveryRate should be 0-100, got %f", a.EarlyDiscoveryRate)
				}
			}
		})
	}
}

// TestGetRecentlyDiscoveredContent tests the getRecentlyDiscoveredContent function
func TestGetRecentlyDiscoveredContent(t *testing.T) {
	db := testDBWithData(t, insertContentDiscoveryTestData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
		limit  int
	}{
		{"default limit", LocationStatsFilter{}, 20},
		{"small limit", LocationStatsFilter{}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := db.getRecentlyDiscoveredContent(context.Background(), tt.filter, tt.limit)
			if err != nil {
				t.Fatalf("getRecentlyDiscoveredContent() error = %v", err)
			}

			for _, item := range content {
				assertNotEmpty(t, "RatingKey", item.RatingKey)
				assertNotEmpty(t, "Title", item.Title)
				assertNonNegative(t, "TotalPlaybacks", item.TotalPlaybacks)
				assertNonNegative(t, "UniqueViewers", item.UniqueViewers)

				// DiscoveryVelocity should be "fast", "medium", or "slow"
				validVelocities := map[string]bool{"fast": true, "medium": true, "slow": true}
				if !validVelocities[item.DiscoveryVelocity] {
					t.Errorf("Invalid DiscoveryVelocity: %s", item.DiscoveryVelocity)
				}
			}
		})
	}
}

// TestGetStaleContent tests the getStaleContent function
func TestGetStaleContent(t *testing.T) {
	db := testDBWithData(t, insertContentDiscoveryStaleData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
		limit  int
	}{
		{"default limit", LocationStatsFilter{}, 50},
		{"small limit", LocationStatsFilter{}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := db.getStaleContent(context.Background(), tt.filter, tt.limit)
			if err != nil {
				t.Fatalf("getStaleContent() error = %v", err)
			}

			for _, item := range content {
				assertNotEmpty(t, "RatingKey", item.RatingKey)
				assertNotEmpty(t, "Title", item.Title)
				assertNonNegative(t, "DaysSinceAdded", item.DaysSinceAdded)

				// Stale content should be at least staleContentThresholdDays old
				if item.DaysSinceAdded < staleContentThresholdDays {
					t.Errorf("Stale content should be >=%d days old, got %d", staleContentThresholdDays, item.DaysSinceAdded)
				}
			}
		})
	}
}

// TestGetLibraryDiscoveryStats tests the getLibraryDiscoveryStats function
func TestGetLibraryDiscoveryStats(t *testing.T) {
	db := testDBWithData(t, insertContentDiscoveryTestData)
	defer db.Close()

	stats, err := db.getLibraryDiscoveryStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("getLibraryDiscoveryStats() error = %v", err)
	}

	for _, s := range stats {
		assertNotEmpty(t, "LibraryName", s.LibraryName)
		assertNonNegative(t, "TotalItems", s.TotalItems)
		assertNonNegative(t, "WatchedItems", s.WatchedItems)
		assertNonNegative(t, "UnwatchedItems", s.UnwatchedItems)
		assertNonNegativeFloat(t, "AvgTimeToDiscoveryHours", s.AvgTimeToDiscoveryHours)

		if s.DiscoveryRate < 0 || s.DiscoveryRate > 100 {
			t.Errorf("DiscoveryRate should be 0-100, got %f", s.DiscoveryRate)
		}
		if s.EarlyDiscoveryRate < 0 || s.EarlyDiscoveryRate > 100 {
			t.Errorf("EarlyDiscoveryRate should be 0-100, got %f", s.EarlyDiscoveryRate)
		}

		// TotalItems = WatchedItems + UnwatchedItems
		if s.TotalItems != s.WatchedItems+s.UnwatchedItems {
			t.Errorf("TotalItems (%d) should equal WatchedItems (%d) + UnwatchedItems (%d)",
				s.TotalItems, s.WatchedItems, s.UnwatchedItems)
		}
	}
}

// TestGetDiscoveryTrends tests the getDiscoveryTrends function
func TestGetDiscoveryTrends(t *testing.T) {
	db := testDBWithData(t, insertContentDiscoveryTestData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{"empty filter", LocationStatsFilter{}},
		{"with date range", LocationStatsFilter{
			StartDate: timePtrOffset(-30 * 24 * time.Hour),
			EndDate:   timePtrOffset(0),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trends, err := db.getDiscoveryTrends(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("getDiscoveryTrends() error = %v", err)
			}

			for _, trend := range trends {
				assertNotEmpty(t, "Date", trend.Date)
				assertNonNegative(t, "ContentAdded", trend.ContentAdded)
				assertNonNegative(t, "ContentDiscovered", trend.ContentDiscovered)
				assertNonNegativeFloat(t, "AvgTimeToDiscoveryHours", trend.AvgTimeToDiscoveryHours)

				if trend.DiscoveryRate < 0 || trend.DiscoveryRate > 100 {
					t.Errorf("DiscoveryRate should be 0-100, got %f", trend.DiscoveryRate)
				}
			}
		})
	}
}

// TestBuildContentDiscoveryMetadata tests the buildContentDiscoveryMetadata function
func TestBuildContentDiscoveryMetadata(t *testing.T) {
	db := testDBWithData(t, insertContentDiscoveryTestData)
	defer db.Close()

	startTime := time.Now()
	filter := LocationStatsFilter{}

	metadata := db.buildContentDiscoveryMetadata(context.Background(), filter, startTime)

	if metadata.ExecutionTimeMS < 0 {
		t.Error("ExecutionTimeMS should not be negative")
	}
	if metadata.EarlyDiscoveryThresholdHours != earlyDiscoveryThresholdHours {
		t.Errorf("EarlyDiscoveryThresholdHours should be %d, got %d", earlyDiscoveryThresholdHours, metadata.EarlyDiscoveryThresholdHours)
	}
	if metadata.StaleContentThresholdDays != staleContentThresholdDays {
		t.Errorf("StaleContentThresholdDays should be %d, got %d", staleContentThresholdDays, metadata.StaleContentThresholdDays)
	}
	if metadata.QueryHash == "" {
		t.Error("QueryHash should not be empty")
	}
	assertNonNegative(t, "TotalEventsAnalyzed", metadata.TotalEventsAnalyzed)
	assertNonNegative(t, "UniqueContentAnalyzed", metadata.UniqueContentAnalyzed)
}

// TestContextCancellationContentDiscovery tests context cancellation handling
func TestContextCancellationContentDiscovery(t *testing.T) {
	db := testDBWithData(t, insertContentDiscoveryTestData)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := db.GetContentDiscoveryAnalytics(ctx, LocationStatsFilter{})
	// Context cancellation may or may not cause an error depending on timing
	// This test ensures no panic
	_ = err
}

// insertContentDiscoveryTestData inserts test data for content discovery tests
func insertContentDiscoveryTestData(t *testing.T, db *DB) {
	now := time.Now()

	// Content with various discovery times
	contentData := []struct {
		ratingKey      string
		title          string
		libraryName    string
		mediaType      string
		addedDaysAgo   int
		watchedDaysAgo int // -1 means never watched
		userID         int
		username       string
	}{
		// Fast discovery (within 24h)
		{"rk1", "New Movie 1", "Movies", "movie", 10, 10, 1, "user1"},
		{"rk2", "New Movie 2", "Movies", "movie", 5, 5, 1, "user1"},

		// Medium discovery (1-7 days)
		{"rk3", "Movie 3", "Movies", "movie", 20, 17, 2, "user2"},
		{"rk4", "Movie 4", "Movies", "movie", 15, 12, 2, "user2"},

		// Slow discovery (7-30 days)
		{"rk5", "Old Movie 5", "Movies", "movie", 60, 40, 1, "user1"},

		// Very slow discovery (30-90 days)
		{"rk6", "Classic Movie", "Movies", "movie", 100, 50, 3, "user3"},

		// TV Shows - mixed discovery
		{"rk7", "Show S01E01", "TV Shows", "episode", 30, 28, 1, "user1"},
		{"rk8", "Show S01E02", "TV Shows", "episode", 30, 25, 1, "user1"},
		{"rk9", "Show2 S01E01", "TV Shows", "episode", 20, 15, 2, "user2"},
	}

	for _, cd := range contentData {
		addedAt := now.Add(-time.Duration(cd.addedDaysAgo) * 24 * time.Hour)
		var watchedAt *time.Time
		if cd.watchedDaysAgo >= 0 {
			t := now.Add(-time.Duration(cd.watchedDaysAgo) * 24 * time.Hour)
			watchedAt = &t
		}
		insertContentWithDiscoveryData(t, db, cd.ratingKey, cd.title, cd.libraryName, cd.mediaType, addedAt, watchedAt, cd.userID, cd.username)
	}
}

// insertContentDiscoveryStaleData inserts test data for stale content tests
func insertContentDiscoveryStaleData(t *testing.T, db *DB) {
	now := time.Now()

	// Content that was added long ago but never watched (stale)
	staleContent := []struct {
		ratingKey    string
		title        string
		libraryName  string
		mediaType    string
		addedDaysAgo int
	}{
		{"stale1", "Forgotten Movie 1", "Movies", "movie", 120},
		{"stale2", "Forgotten Movie 2", "Movies", "movie", 150},
		{"stale3", "Old Show", "TV Shows", "episode", 180},
	}

	for _, sc := range staleContent {
		addedAt := now.Add(-time.Duration(sc.addedDaysAgo) * 24 * time.Hour)
		insertContentWithDiscoveryData(t, db, sc.ratingKey, sc.title, sc.libraryName, sc.mediaType, addedAt, nil, 1, "user1")
	}

	// Also insert some non-stale content for comparison
	insertContentDiscoveryTestData(t, db)
}

// insertContentWithDiscoveryData inserts a playback event with discovery-related data
func insertContentWithDiscoveryData(t *testing.T, db *DB, ratingKey, title, libraryName, mediaType string, addedAt time.Time, watchedAt *time.Time, userID int, username string) {
	t.Helper()

	var startedAt, stoppedAt time.Time
	var percentComplete int

	if watchedAt != nil {
		startedAt = *watchedAt
		stoppedAt = watchedAt.Add(90 * time.Minute)
		percentComplete = 85
	} else {
		// For never-watched/stale content, use a placeholder started_at
		// with very low percent_complete (<=5) so it counts as "not really watched"
		startedAt = addedAt.Add(time.Hour) // Just after added_at
		stoppedAt = startedAt.Add(time.Minute)
		percentComplete = 1 // Very low completion = counts as "stale"
	}

	_, err := db.conn.Exec(`
		INSERT INTO playback_events (
			id, session_key, started_at, stopped_at, user_id, username,
			ip_address, media_type, title, rating_key, library_name,
			added_at, percent_complete, play_duration
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.New().String(), uuid.New().String(), startedAt, stoppedAt,
		userID, username, "192.168.1.1", mediaType, title, ratingKey, libraryName,
		addedAt.Format(time.RFC3339), percentComplete, 1)
	if err != nil {
		t.Fatalf("Failed to insert content with discovery data: %v", err)
	}
}

// TestEmptyDatabaseContentDiscovery tests behavior with no data
func TestEmptyDatabaseContentDiscovery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	result, err := db.GetContentDiscoveryAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetContentDiscoveryAnalytics() with empty db error = %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result even with empty database")
	}

	// Summary should have zeros
	if result.Summary.TotalContentWithAddedAt != 0 {
		t.Errorf("Expected TotalContentWithAddedAt=0, got %d", result.Summary.TotalContentWithAddedAt)
	}
	if result.Summary.TotalDiscovered != 0 {
		t.Errorf("Expected TotalDiscovered=0, got %d", result.Summary.TotalDiscovered)
	}
}

// TestDiscoveryWithInvalidAddedAt tests handling of invalid added_at values
func TestDiscoveryWithInvalidAddedAt(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	// Insert content with invalid added_at
	_, err := db.conn.Exec(`
		INSERT INTO playback_events (
			id, session_key, started_at, stopped_at, user_id, username,
			ip_address, media_type, title, rating_key, added_at, percent_complete, play_duration
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.New().String(), uuid.New().String(), time.Now(), time.Now().Add(time.Hour),
		1, "user1", "192.168.1.1", "movie", "Test Movie", "rk_invalid", "not-a-date", 100, 60)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Should not crash with invalid data
	_, err = db.GetContentDiscoveryAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetContentDiscoveryAnalytics() should handle invalid added_at: %v", err)
	}
}
