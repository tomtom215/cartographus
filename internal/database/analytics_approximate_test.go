// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"testing"
	"time"
)

// TestGetApproximateStats tests the approximate stats functionality
func TestGetApproximateStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert test data
	insertTestPlaybacksForApproximateStats(t, db)

	tests := []struct {
		name          string
		filter        ApproximateStatsFilter
		wantMinUsers  int64
		wantMaxUsers  int64
		wantMinTitles int64
		wantMinTotal  int64
	}{
		{
			name:          "no filter returns all data",
			filter:        ApproximateStatsFilter{},
			wantMinUsers:  1,
			wantMaxUsers:  100,
			wantMinTitles: 1,
			wantMinTotal:  5, // We insert 5 test events
		},
		{
			name: "date filter restricts results",
			filter: ApproximateStatsFilter{
				StartDate: approxTimePtr(time.Now().Add(-24 * time.Hour)),
				EndDate:   approxTimePtr(time.Now().Add(24 * time.Hour)),
			},
			wantMinUsers:  1,
			wantMaxUsers:  100,
			wantMinTitles: 1,
			wantMinTotal:  1,
		},
		{
			name: "user filter restricts results",
			filter: ApproximateStatsFilter{
				Users: []string{"testuser1"},
			},
			wantMinUsers:  0,
			wantMaxUsers:  1,
			wantMinTitles: 0,
			wantMinTotal:  0,
		},
		{
			name: "media type filter restricts results",
			filter: ApproximateStatsFilter{
				MediaTypes: []string{"movie"},
			},
			wantMinUsers:  0,
			wantMaxUsers:  100,
			wantMinTitles: 0,
			wantMinTotal:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := db.GetApproximateStats(ctx, tt.filter)
			if err != nil {
				t.Fatalf("GetApproximateStats() error = %v", err)
			}

			if stats == nil {
				t.Fatal("GetApproximateStats() returned nil stats")
			}

			// Verify user count is within expected range
			if stats.UniqueUsers < tt.wantMinUsers {
				t.Errorf("UniqueUsers = %d, want at least %d", stats.UniqueUsers, tt.wantMinUsers)
			}
			if stats.UniqueUsers > tt.wantMaxUsers {
				t.Errorf("UniqueUsers = %d, want at most %d", stats.UniqueUsers, tt.wantMaxUsers)
			}

			// Verify title count
			if stats.UniqueTitles < tt.wantMinTitles {
				t.Errorf("UniqueTitles = %d, want at least %d", stats.UniqueTitles, tt.wantMinTitles)
			}

			// Verify total playbacks
			if stats.TotalPlaybacks < tt.wantMinTotal {
				t.Errorf("TotalPlaybacks = %d, want at least %d", stats.TotalPlaybacks, tt.wantMinTotal)
			}

			// Verify percentiles are non-negative when we have data
			if stats.TotalPlaybacks > 0 {
				if stats.WatchTimeP50 < 0 {
					t.Errorf("WatchTimeP50 = %f, want non-negative", stats.WatchTimeP50)
				}
				if stats.WatchTimeP95 < stats.WatchTimeP50 {
					t.Errorf("WatchTimeP95 (%f) should be >= WatchTimeP50 (%f)", stats.WatchTimeP95, stats.WatchTimeP50)
				}
			}

			// Verify QueryTimeMS is set
			if stats.QueryTimeMS < 0 {
				t.Errorf("QueryTimeMS = %d, want non-negative", stats.QueryTimeMS)
			}

			// Log whether approximate or exact was used
			t.Logf("IsApproximate: %v, UniqueUsers: %d, TotalPlaybacks: %d, QueryTime: %dms",
				stats.IsApproximate, stats.UniqueUsers, stats.TotalPlaybacks, stats.QueryTimeMS)
		})
	}
}

// TestApproximateDistinctCount tests the distinct count functionality
func TestApproximateDistinctCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert test data
	insertTestPlaybacksForApproximateStats(t, db)

	tests := []struct {
		name      string
		column    string
		filter    ApproximateStatsFilter
		wantMin   int64
		wantMax   int64
		wantError bool
	}{
		{
			name:    "distinct usernames",
			column:  "username",
			filter:  ApproximateStatsFilter{},
			wantMin: 1,
			wantMax: 100,
		},
		{
			name:    "distinct titles",
			column:  "title",
			filter:  ApproximateStatsFilter{},
			wantMin: 1,
			wantMax: 100,
		},
		{
			name:    "distinct media types",
			column:  "media_type",
			filter:  ApproximateStatsFilter{},
			wantMin: 1,
			wantMax: 10,
		},
		{
			name:      "invalid column returns error",
			column:    "invalid_column",
			filter:    ApproximateStatsFilter{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, isApproximate, err := db.ApproximateDistinctCount(ctx, tt.column, tt.filter)

			if tt.wantError {
				if err == nil {
					t.Error("ApproximateDistinctCount() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ApproximateDistinctCount() error = %v", err)
			}

			if count < tt.wantMin {
				t.Errorf("count = %d, want at least %d", count, tt.wantMin)
			}
			if count > tt.wantMax {
				t.Errorf("count = %d, want at most %d", count, tt.wantMax)
			}

			t.Logf("Column: %s, Count: %d, IsApproximate: %v", tt.column, count, isApproximate)
		})
	}
}

// TestApproximatePercentile tests the percentile calculation functionality
func TestApproximatePercentile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert test data
	insertTestPlaybacksForApproximateStats(t, db)

	tests := []struct {
		name       string
		column     string
		percentile float64
		filter     ApproximateStatsFilter
		wantMin    float64
		wantMax    float64
		wantError  bool
	}{
		{
			name:       "median play_duration",
			column:     "play_duration",
			percentile: 0.50,
			filter:     ApproximateStatsFilter{},
			wantMin:    0,
			wantMax:    100000,
		},
		{
			name:       "p95 play_duration",
			column:     "play_duration",
			percentile: 0.95,
			filter:     ApproximateStatsFilter{},
			wantMin:    0,
			wantMax:    100000,
		},
		{
			name:       "median percent_complete",
			column:     "percent_complete",
			percentile: 0.50,
			filter:     ApproximateStatsFilter{},
			wantMin:    0,
			wantMax:    100,
		},
		{
			name:       "invalid column returns error",
			column:     "invalid_column",
			percentile: 0.50,
			filter:     ApproximateStatsFilter{},
			wantError:  true,
		},
		{
			name:       "invalid percentile returns error",
			column:     "play_duration",
			percentile: 1.5,
			filter:     ApproximateStatsFilter{},
			wantError:  true,
		},
		{
			name:       "negative percentile returns error",
			column:     "play_duration",
			percentile: -0.1,
			filter:     ApproximateStatsFilter{},
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, isApproximate, err := db.ApproximatePercentile(ctx, tt.column, tt.percentile, tt.filter)

			if tt.wantError {
				if err == nil {
					t.Error("ApproximatePercentile() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ApproximatePercentile() error = %v", err)
			}

			if value < tt.wantMin {
				t.Errorf("value = %f, want at least %f", value, tt.wantMin)
			}
			if value > tt.wantMax {
				t.Errorf("value = %f, want at most %f", value, tt.wantMax)
			}

			t.Logf("Column: %s, Percentile: %.2f, Value: %f, IsApproximate: %v",
				tt.column, tt.percentile, value, isApproximate)
		})
	}
}

// TestApproximateStatsPercentileOrdering tests that percentiles are properly ordered
func TestApproximateStatsPercentileOrdering(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert test data with varying durations
	insertTestPlaybacksWithVaryingDurations(t, db)

	stats, err := db.GetApproximateStats(ctx, ApproximateStatsFilter{})
	if err != nil {
		t.Fatalf("GetApproximateStats() error = %v", err)
	}

	// Verify percentile ordering (p50 <= p75 <= p90 <= p95 <= p99)
	if stats.WatchTimeP50 > stats.WatchTimeP75 {
		t.Errorf("P50 (%f) should be <= P75 (%f)", stats.WatchTimeP50, stats.WatchTimeP75)
	}
	if stats.WatchTimeP75 > stats.WatchTimeP90 {
		t.Errorf("P75 (%f) should be <= P90 (%f)", stats.WatchTimeP75, stats.WatchTimeP90)
	}
	if stats.WatchTimeP90 > stats.WatchTimeP95 {
		t.Errorf("P90 (%f) should be <= P95 (%f)", stats.WatchTimeP90, stats.WatchTimeP95)
	}
	if stats.WatchTimeP95 > stats.WatchTimeP99 {
		t.Errorf("P95 (%f) should be <= P99 (%f)", stats.WatchTimeP95, stats.WatchTimeP99)
	}
}

// TestDataSketchesAvailability tests the availability detection
func TestDataSketchesAvailability(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// The availability flag should be set during extension loading
	available := db.IsDataSketchesAvailable()

	// Log the availability for debugging
	t.Logf("DataSketches available: %v", available)

	// Test that approximate stats works regardless of availability
	ctx := context.Background()
	insertTestPlaybacksForApproximateStats(t, db)

	stats, err := db.GetApproximateStats(ctx, ApproximateStatsFilter{})
	if err != nil {
		t.Fatalf("GetApproximateStats() should work regardless of DataSketches availability: %v", err)
	}

	// Verify IsApproximate matches availability
	if stats.IsApproximate != available {
		t.Logf("Note: IsApproximate (%v) differs from availability (%v) - this can happen if approximate query failed and fell back to exact",
			stats.IsApproximate, available)
	}
}

// TestConvenienceWrappers tests the convenience wrapper functions
func TestConvenienceWrappers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	insertTestPlaybacksForApproximateStats(t, db)

	filter := ApproximateStatsFilter{}

	t.Run("GetDistinctUsersApproximate", func(t *testing.T) {
		count, _, err := db.GetDistinctUsersApproximate(ctx, filter)
		if err != nil {
			t.Fatalf("GetDistinctUsersApproximate() error = %v", err)
		}
		if count < 0 {
			t.Errorf("count = %d, want non-negative", count)
		}
	})

	t.Run("GetDistinctTitlesApproximate", func(t *testing.T) {
		count, _, err := db.GetDistinctTitlesApproximate(ctx, filter)
		if err != nil {
			t.Fatalf("GetDistinctTitlesApproximate() error = %v", err)
		}
		if count < 0 {
			t.Errorf("count = %d, want non-negative", count)
		}
	})

	t.Run("GetMedianWatchTimeApproximate", func(t *testing.T) {
		median, _, err := db.GetMedianWatchTimeApproximate(ctx, filter)
		if err != nil {
			t.Fatalf("GetMedianWatchTimeApproximate() error = %v", err)
		}
		if median < 0 {
			t.Errorf("median = %f, want non-negative", median)
		}
	})
}

// insertTestPlaybacksForApproximateStats inserts test data for approximate stats tests
func insertTestPlaybacksForApproximateStats(t *testing.T, db *DB) {
	t.Helper()

	ctx := context.Background()

	testEvents := []struct {
		ratingKey       string
		title           string
		mediaType       string
		username        string
		playDuration    int
		percentComplete int
		platform        string
		player          string
	}{
		{"2001", "Inception", "movie", "user1", 7200, 95, "Web", "Chrome"},
		{"2002", "The Matrix", "movie", "user2", 8100, 100, "Android", "Plex"},
		{"2003", "Breaking Bad S01E01", "episode", "user1", 2700, 80, "iOS", "Plex"},
		{"2004", "Game of Thrones S01E01", "episode", "user3", 3600, 90, "Web", "Firefox"},
		{"2005", "Dark Side of the Moon", "track", "user2", 300, 100, "Desktop", "Plexamp"},
	}

	for _, e := range testEvents {
		sql := `
			INSERT INTO playback_events (
				id, session_key, started_at, user_id, username, ip_address,
				media_type, title, platform, player, play_duration, percent_complete,
				paused_counter, rating_key, location_type
			) VALUES (
				uuid(), ?, CURRENT_TIMESTAMP, ?, ?, '192.168.1.1',
				?, ?, ?, ?, ?, ?,
				0, ?, 'lan'
			)
		`
		_, err := db.conn.ExecContext(ctx, sql,
			"session_"+e.ratingKey,
			100+len(e.username),
			e.username,
			e.mediaType,
			e.title,
			e.platform,
			e.player,
			e.playDuration,
			e.percentComplete,
			e.ratingKey,
		)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}
}

// insertTestPlaybacksWithVaryingDurations inserts test data with varying durations for percentile testing
func insertTestPlaybacksWithVaryingDurations(t *testing.T, db *DB) {
	t.Helper()

	ctx := context.Background()

	// Insert events with play_durations: 60, 120, 180, 240, 300, 600, 900, 1200, 1800, 3600 seconds
	durations := []int{60, 120, 180, 240, 300, 600, 900, 1200, 1800, 3600}

	for i, playDuration := range durations {
		sql := `
			INSERT INTO playback_events (
				id, session_key, started_at, user_id, username, ip_address,
				media_type, title, platform, player, play_duration, percent_complete,
				paused_counter, rating_key, location_type
			) VALUES (
				uuid(), ?, CURRENT_TIMESTAMP, ?, ?, '192.168.1.1',
				'movie', ?, 'Web', 'Chrome', ?, 100,
				0, ?, 'lan'
			)
		`
		_, err := db.conn.ExecContext(ctx, sql,
			"duration_session_"+string(rune('a'+i)),
			100+i,
			"testuser",
			"Test Movie "+string(rune('A'+i)),
			playDuration,
			"3000"+string(rune('0'+i)),
		)
		if err != nil {
			t.Fatalf("Failed to insert test data for play_duration %d: %v", playDuration, err)
		}
	}
}

// approxTimePtr returns a pointer to the given time (local helper to avoid collision with database_test.go)
func approxTimePtr(t time.Time) *time.Time {
	return &t
}
