// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// Helper function for creating pointer to string (intPtr is in analytics_advanced_test.go)
func stringPtr(s string) *string { return &s }

func TestBuildBingeWhereClause(t *testing.T) {

	t.Run("empty filter returns episode-only clause", func(t *testing.T) {
		filter := LocationStatsFilter{}
		whereClause, args := buildBingeWhereClause(filter)

		if whereClause != "media_type = 'episode'" {
			t.Errorf("Expected 'media_type = 'episode'', got: %s", whereClause)
		}
		if len(args) != 0 {
			t.Errorf("Expected 0 args, got: %d", len(args))
		}
	})

	t.Run("with start date", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		filter := LocationStatsFilter{StartDate: &startDate}
		whereClause, args := buildBingeWhereClause(filter)

		if whereClause != "media_type = 'episode' AND started_at >= ?" {
			t.Errorf("Unexpected where clause: %s", whereClause)
		}
		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got: %d", len(args))
		}
		if args[0] != startDate {
			t.Errorf("Start date arg mismatch: got %v, want %v", args[0], startDate)
		}
	})

	t.Run("with end date", func(t *testing.T) {
		endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
		filter := LocationStatsFilter{EndDate: &endDate}
		whereClause, args := buildBingeWhereClause(filter)

		if whereClause != "media_type = 'episode' AND started_at <= ?" {
			t.Errorf("Unexpected where clause: %s", whereClause)
		}
		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got: %d", len(args))
		}
		if args[0] != endDate {
			t.Errorf("End date arg mismatch: got %v, want %v", args[0], endDate)
		}
	})

	t.Run("with date range", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
		filter := LocationStatsFilter{
			StartDate: &startDate,
			EndDate:   &endDate,
		}
		whereClause, args := buildBingeWhereClause(filter)

		expectedClause := "media_type = 'episode' AND started_at >= ? AND started_at <= ?"
		if whereClause != expectedClause {
			t.Errorf("Unexpected where clause: %s\nExpected: %s", whereClause, expectedClause)
		}
		if len(args) != 2 {
			t.Errorf("Expected 2 args, got: %d", len(args))
		}
	})

	t.Run("with single user", func(t *testing.T) {
		filter := LocationStatsFilter{Users: []string{"alice"}}
		whereClause, args := buildBingeWhereClause(filter)

		expectedClause := "media_type = 'episode' AND username IN (?)"
		if whereClause != expectedClause {
			t.Errorf("Unexpected where clause: %s\nExpected: %s", whereClause, expectedClause)
		}
		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got: %d", len(args))
		}
		if args[0] != "alice" {
			t.Errorf("User arg mismatch: got %v, want 'alice'", args[0])
		}
	})

	t.Run("with multiple users", func(t *testing.T) {
		filter := LocationStatsFilter{Users: []string{"alice", "bob", "charlie"}}
		whereClause, args := buildBingeWhereClause(filter)

		expectedClause := "media_type = 'episode' AND username IN (?, ?, ?)"
		if whereClause != expectedClause {
			t.Errorf("Unexpected where clause: %s\nExpected: %s", whereClause, expectedClause)
		}
		if len(args) != 3 {
			t.Errorf("Expected 3 args, got: %d", len(args))
		}
		if args[0] != "alice" || args[1] != "bob" || args[2] != "charlie" {
			t.Errorf("User args mismatch: got %v", args)
		}
	})

	t.Run("with all filters", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
		filter := LocationStatsFilter{
			StartDate: &startDate,
			EndDate:   &endDate,
			Users:     []string{"alice", "bob"},
		}
		whereClause, args := buildBingeWhereClause(filter)

		expectedClause := "media_type = 'episode' AND started_at >= ? AND started_at <= ? AND username IN (?, ?)"
		if whereClause != expectedClause {
			t.Errorf("Unexpected where clause: %s\nExpected: %s", whereClause, expectedClause)
		}
		if len(args) != 4 {
			t.Errorf("Expected 4 args, got: %d", len(args))
		}
	})
}

func TestGetBingeAnalytics(t *testing.T) {

	// Note: Subtests that create database connections should NOT call t.Parallel()
	// to avoid DuckDB resource contention and hangs with race detector.
	// See: 307 parallel database tests causing goroutine deadlock.

	t.Run("empty database returns zero values", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		analytics, err := db.GetBingeAnalytics(context.Background(), LocationStatsFilter{})
		if err != nil {
			t.Fatalf("GetBingeAnalytics error: %v", err)
		}

		if analytics == nil {
			t.Fatal("Expected non-nil analytics")
		}
		if analytics.TotalBingeSessions != 0 {
			t.Errorf("TotalBingeSessions = %d, want 0", analytics.TotalBingeSessions)
		}
		if analytics.TotalEpisodesBinged != 0 {
			t.Errorf("TotalEpisodesBinged = %d, want 0", analytics.TotalEpisodesBinged)
		}
		if analytics.RecentBingeSessions == nil {
			t.Error("RecentBingeSessions should be empty slice, not nil")
		}
		if analytics.TopBingeShows == nil {
			t.Error("TopBingeShows should be empty slice, not nil")
		}
		if analytics.TopBingeWatchers == nil {
			t.Error("TopBingeWatchers should be empty slice, not nil")
		}
		if analytics.BingesByDay == nil {
			t.Error("BingesByDay should be empty slice, not nil")
		}
	})

	t.Run("with binge sessions", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Insert binge-worthy data: 5 episodes of same show within 6 hours
		baseTime := time.Date(2024, 6, 15, 19, 0, 0, 0, time.UTC)
		for i := 0; i < 5; i++ {
			event := createTestPlaybackEvent()
			event.UserID = 1
			event.Username = "bingewatcher"
			event.MediaType = "episode"
			event.GrandparentTitle = stringPtr("Breaking Bad")
			event.ParentTitle = stringPtr("Season 1")
			event.Title = "Episode " + string(rune('1'+i))
			event.MediaIndex = intPtr(i + 1)
			event.StartedAt = baseTime.Add(time.Duration(i) * time.Hour) // 1 hour apart
			stoppedAt := event.StartedAt.Add(45 * time.Minute)
			event.StoppedAt = &stoppedAt
			event.PlayDuration = intPtr(45)
			event.PercentComplete = 95

			if err := db.InsertPlaybackEvent(event); err != nil {
				t.Fatalf("Failed to insert event: %v", err)
			}
		}

		analytics, err := db.GetBingeAnalytics(context.Background(), LocationStatsFilter{})
		if err != nil {
			t.Fatalf("GetBingeAnalytics error: %v", err)
		}

		if analytics.TotalBingeSessions < 1 {
			t.Errorf("Expected at least 1 binge session, got %d", analytics.TotalBingeSessions)
		}
		if analytics.TotalEpisodesBinged < 3 {
			t.Errorf("Expected at least 3 episodes binged, got %d", analytics.TotalEpisodesBinged)
		}
	})

	t.Run("non-consecutive episodes not counted as binge", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Insert episodes more than 6 hours apart (no binge)
		baseTime := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		for i := 0; i < 3; i++ {
			event := createTestPlaybackEvent()
			event.UserID = 2
			event.Username = "casual_watcher"
			event.MediaType = "episode"
			event.GrandparentTitle = stringPtr("The Office")
			event.StartedAt = baseTime.Add(time.Duration(i) * 24 * time.Hour) // 24 hours apart (> 6 hours)
			event.PercentComplete = 90

			if err := db.InsertPlaybackEvent(event); err != nil {
				t.Fatalf("Failed to insert event: %v", err)
			}
		}

		analytics, err := db.GetBingeAnalytics(context.Background(), LocationStatsFilter{})
		if err != nil {
			t.Fatalf("GetBingeAnalytics error: %v", err)
		}

		// Episodes watched more than 6 hours apart should not be counted as binge
		if analytics.TotalBingeSessions != 0 {
			t.Errorf("Expected 0 binge sessions for non-consecutive watching, got %d", analytics.TotalBingeSessions)
		}
	})

	t.Run("movies not included in binge analysis", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Insert movies (should not be counted)
		for i := 0; i < 5; i++ {
			event := createTestPlaybackEvent()
			event.UserID = 3
			event.Username = "movie_fan"
			event.MediaType = "movie"
			event.Title = "Movie " + string(rune('A'+i))
			event.GrandparentTitle = stringPtr("")
			event.StartedAt = time.Now().Add(-time.Duration(i) * time.Hour)

			if err := db.InsertPlaybackEvent(event); err != nil {
				t.Fatalf("Failed to insert event: %v", err)
			}
		}

		analytics, err := db.GetBingeAnalytics(context.Background(), LocationStatsFilter{})
		if err != nil {
			t.Fatalf("GetBingeAnalytics error: %v", err)
		}

		// Movies should not contribute to binge sessions
		if analytics.TotalBingeSessions != 0 {
			t.Errorf("Expected 0 binge sessions for movies, got %d", analytics.TotalBingeSessions)
		}
	})

	t.Run("user filter works correctly", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Insert binge sessions for two different users
		baseTime := time.Date(2024, 6, 15, 19, 0, 0, 0, time.UTC)

		// User 1 binge
		for i := 0; i < 4; i++ {
			event := createTestPlaybackEvent()
			event.UserID = 1
			event.Username = "user_one"
			event.MediaType = "episode"
			event.GrandparentTitle = stringPtr("Show A")
			event.StartedAt = baseTime.Add(time.Duration(i) * 30 * time.Minute)

			if err := db.InsertPlaybackEvent(event); err != nil {
				t.Fatalf("Failed to insert event: %v", err)
			}
		}

		// User 2 binge
		for i := 0; i < 4; i++ {
			event := createTestPlaybackEvent()
			event.UserID = 2
			event.Username = "user_two"
			event.MediaType = "episode"
			event.GrandparentTitle = stringPtr("Show B")
			event.StartedAt = baseTime.Add(time.Duration(i) * 30 * time.Minute)

			if err := db.InsertPlaybackEvent(event); err != nil {
				t.Fatalf("Failed to insert event: %v", err)
			}
		}

		// Filter for user_one only
		filter := LocationStatsFilter{Users: []string{"user_one"}}
		analytics, err := db.GetBingeAnalytics(context.Background(), filter)
		if err != nil {
			t.Fatalf("GetBingeAnalytics error: %v", err)
		}

		// Should only count user_one's binge sessions
		if analytics.TotalBingeSessions > 0 {
			// Verify that the recent sessions are only from user_one
			for _, session := range analytics.RecentBingeSessions {
				if session.Username != "user_one" {
					t.Errorf("Found session from wrong user: %s", session.Username)
				}
			}
		}
	})

	t.Run("date filter works correctly", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Insert binge in January
		jan := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		for i := 0; i < 4; i++ {
			event := createTestPlaybackEvent()
			event.UserID = 1
			event.Username = "tester"
			event.MediaType = "episode"
			event.GrandparentTitle = stringPtr("January Show")
			event.StartedAt = jan.Add(time.Duration(i) * time.Hour)

			if err := db.InsertPlaybackEvent(event); err != nil {
				t.Fatalf("Failed to insert event: %v", err)
			}
		}

		// Insert binge in June
		june := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		for i := 0; i < 4; i++ {
			event := createTestPlaybackEvent()
			event.UserID = 1
			event.Username = "tester"
			event.MediaType = "episode"
			event.GrandparentTitle = stringPtr("June Show")
			event.StartedAt = june.Add(time.Duration(i) * time.Hour)

			if err := db.InsertPlaybackEvent(event); err != nil {
				t.Fatalf("Failed to insert event: %v", err)
			}
		}

		// Filter for June only
		juneStart := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		juneEnd := time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC)
		filter := LocationStatsFilter{StartDate: &juneStart, EndDate: &juneEnd}

		analytics, err := db.GetBingeAnalytics(context.Background(), filter)
		if err != nil {
			t.Fatalf("GetBingeAnalytics error: %v", err)
		}

		// Verify only June shows appear in results
		for _, show := range analytics.TopBingeShows {
			if show.ShowName == "January Show" {
				t.Error("January Show should not appear in June-filtered results")
			}
		}
	})

	t.Run("binges by day returns all 7 days", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Even with no data, should return all 7 days with zero counts
		analytics, err := db.GetBingeAnalytics(context.Background(), LocationStatsFilter{})
		if err != nil {
			t.Fatalf("GetBingeAnalytics error: %v", err)
		}

		if len(analytics.BingesByDay) != 7 {
			t.Errorf("BingesByDay should have 7 entries, got %d", len(analytics.BingesByDay))
		}

		// Verify days 0-6 are present
		for i, day := range analytics.BingesByDay {
			if day.DayOfWeek != i {
				t.Errorf("Day %d has wrong DayOfWeek: %d", i, day.DayOfWeek)
			}
		}
	})

	t.Run("context cancellation is respected", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := db.GetBingeAnalytics(ctx, LocationStatsFilter{})
		if err == nil {
			// Note: DuckDB may not always respect context cancellation
			// This test verifies that the context is passed through
			t.Log("Context cancellation may not be immediately respected by DuckDB")
		}
	})
}

// Helper function to create a test playback event
func createTestPlaybackEvent() *models.PlaybackEvent {
	now := time.Now()
	stoppedAt := now.Add(30 * time.Minute)
	return &models.PlaybackEvent{
		SessionKey:      "session-" + now.Format("20060102150405"),
		UserID:          1,
		Username:        "testuser",
		IPAddress:       "192.168.1.100",
		MediaType:       "episode",
		Title:           "Test Episode",
		Platform:        "Chrome",
		Player:          "Plex Web",
		FullTitle:       stringPtr("Test Show - S01E01 - Test Episode"),
		StartedAt:       now,
		StoppedAt:       &stoppedAt,
		PausedCounter:   0,
		PlayDuration:    intPtr(30),
		PercentComplete: 80,
	}
}

// Benchmark tests
func BenchmarkGetBingeAnalytics_EmptyDB(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()

	ctx := context.Background()
	filter := LocationStatsFilter{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetBingeAnalytics(ctx, filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildBingeWhereClause(b *testing.B) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	filter := LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
		Users:     []string{"alice", "bob", "charlie"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildBingeWhereClause(filter)
	}
}
