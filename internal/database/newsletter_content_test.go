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

func TestGetRecentlyAddedMovies(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Insert test data
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type": "movie",
		"title":      "Test Movie 1",
		"rating_key": "movie1",
		"added_at":   time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		"year":       2024,
		"genres":     "Action, Comedy",
	})
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type": "movie",
		"title":      "Test Movie 2",
		"rating_key": "movie2",
		"added_at":   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		"year":       2023,
	})
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type": "episode",
		"title":      "Test Episode",
		"rating_key": "ep1",
		"added_at":   time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	})

	// Test getting recently added movies
	since := time.Now().Add(-24 * time.Hour)
	movies, err := db.GetRecentlyAddedMovies(ctx, since, 10)
	if err != nil {
		t.Fatalf("GetRecentlyAddedMovies failed: %v", err)
	}

	if len(movies) < 1 {
		t.Errorf("Expected at least 1 movie, got %d", len(movies))
	}

	// Verify it only returns movies (not episodes)
	for _, movie := range movies {
		if movie.MediaType != "movie" {
			t.Errorf("Expected media_type 'movie', got '%s'", movie.MediaType)
		}
	}
}

func TestGetRecentlyAddedShows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Insert test episode data
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type":             "episode",
		"title":                  "Episode 1",
		"grandparent_title":      "Test Show",
		"rating_key":             "ep1",
		"grandparent_rating_key": "show1",
		"added_at":               time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		"year":                   2024,
	})
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type":             "episode",
		"title":                  "Episode 2",
		"grandparent_title":      "Test Show",
		"rating_key":             "ep2",
		"grandparent_rating_key": "show1",
		"added_at":               time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		"year":                   2024,
	})

	// Test getting recently added shows
	since := time.Now().Add(-24 * time.Hour)
	shows, err := db.GetRecentlyAddedShows(ctx, since, 10)
	if err != nil {
		t.Fatalf("GetRecentlyAddedShows failed: %v", err)
	}

	if len(shows) < 1 {
		t.Errorf("Expected at least 1 show, got %d", len(shows))
	}

	// Verify show data
	for _, show := range shows {
		if show.Title == "" {
			t.Error("Show title should not be empty")
		}
		if show.RatingKey == "" {
			t.Error("Show rating key should not be empty")
		}
	}
}

func TestGetTopMovies(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Insert multiple playbacks for the same movie
	for i := 0; i < 5; i++ {
		insertTestPlaybackEvent(t, db, map[string]interface{}{
			"media_type": "movie",
			"title":      "Popular Movie",
			"rating_key": "pop_movie",
			"started_at": time.Now().Add(-time.Duration(i) * time.Hour),
			"stopped_at": time.Now().Add(-time.Duration(i)*time.Hour + 2*time.Hour),
			"user_id":    i + 1,
		})
	}

	// Insert fewer playbacks for another movie
	for i := 0; i < 2; i++ {
		insertTestPlaybackEvent(t, db, map[string]interface{}{
			"media_type": "movie",
			"title":      "Less Popular Movie",
			"rating_key": "less_pop_movie",
			"started_at": time.Now().Add(-time.Duration(i) * time.Hour),
			"stopped_at": time.Now().Add(-time.Duration(i)*time.Hour + time.Hour),
			"user_id":    i + 1,
		})
	}

	// Test getting top movies
	since := time.Now().Add(-7 * 24 * time.Hour)
	movies, err := db.GetTopMovies(ctx, since, 10)
	if err != nil {
		t.Fatalf("GetTopMovies failed: %v", err)
	}

	if len(movies) < 1 {
		t.Errorf("Expected at least 1 movie, got %d", len(movies))
	}

	// Verify the most popular movie is first
	if len(movies) >= 2 {
		if movies[0].WatchCount < movies[1].WatchCount {
			t.Error("Movies should be ordered by watch count descending")
		}
	}
}

func TestGetTopShows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Insert multiple episode playbacks for the same show
	for i := 0; i < 5; i++ {
		insertTestPlaybackEvent(t, db, map[string]interface{}{
			"media_type":             "episode",
			"title":                  "Episode " + string(rune('A'+i)),
			"grandparent_title":      "Popular Show",
			"rating_key":             "ep" + string(rune('A'+i)),
			"grandparent_rating_key": "pop_show",
			"started_at":             time.Now().Add(-time.Duration(i) * time.Hour),
			"stopped_at":             time.Now().Add(-time.Duration(i)*time.Hour + time.Hour),
			"user_id":                i + 1,
		})
	}

	// Test getting top shows
	since := time.Now().Add(-7 * 24 * time.Hour)
	shows, err := db.GetTopShows(ctx, since, 10)
	if err != nil {
		t.Fatalf("GetTopShows failed: %v", err)
	}

	if len(shows) < 1 {
		t.Errorf("Expected at least 1 show, got %d", len(shows))
	}

	// Verify show data includes watch count
	for _, show := range shows {
		if show.WatchCount <= 0 {
			t.Error("Show watch count should be positive")
		}
	}
}

func TestGetPeriodStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now

	// Insert test playbacks
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type": "movie",
		"title":      "Test Movie",
		"rating_key": "movie1",
		"started_at": now.Add(-2 * time.Hour),
		"stopped_at": now.Add(-1 * time.Hour),
		"user_id":    1,
		"platform":   "Chrome",
	})
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type": "episode",
		"title":      "Test Episode",
		"rating_key": "ep1",
		"started_at": now.Add(-3 * time.Hour),
		"stopped_at": now.Add(-2 * time.Hour),
		"user_id":    2,
		"platform":   "Plex Web",
	})

	// Test getting period stats
	stats, err := db.GetPeriodStats(ctx, start, end)
	if err != nil {
		t.Fatalf("GetPeriodStats failed: %v", err)
	}

	if stats.TotalPlaybacks < 2 {
		t.Errorf("Expected at least 2 playbacks, got %d", stats.TotalPlaybacks)
	}

	if stats.UniqueUsers < 1 {
		t.Errorf("Expected at least 1 unique user, got %d", stats.UniqueUsers)
	}

	if stats.MoviesWatched < 1 {
		t.Errorf("Expected at least 1 movie watched, got %d", stats.MoviesWatched)
	}

	if stats.EpisodesWatched < 1 {
		t.Errorf("Expected at least 1 episode watched, got %d", stats.EpisodesWatched)
	}
}

func TestGetUserStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now

	// Insert test playbacks for user
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type": "movie",
		"title":      "Test Movie",
		"rating_key": "movie1",
		"started_at": now.Add(-2 * time.Hour),
		"stopped_at": now.Add(-1 * time.Hour),
		"user_id":    100,
		"username":   "testuser",
		"genres":     "Action",
	})
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type":        "episode",
		"title":             "Test Episode",
		"grandparent_title": "Test Show",
		"rating_key":        "ep1",
		"started_at":        now.Add(-4 * time.Hour),
		"stopped_at":        now.Add(-3 * time.Hour),
		"user_id":           100,
		"username":          "testuser",
	})

	// Test getting user stats
	userStats, err := db.GetUserStats(ctx, "100", start, end)
	if err != nil {
		t.Fatalf("GetUserStats failed: %v", err)
	}

	if userStats.UserID != "100" {
		t.Errorf("Expected user ID '100', got '%s'", userStats.UserID)
	}

	if userStats.PlaybackCount < 2 {
		t.Errorf("Expected at least 2 playbacks, got %d", userStats.PlaybackCount)
	}
}

func TestGetServerHealth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Insert some test data
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type": "movie",
		"title":      "Test Movie",
		"rating_key": "movie1",
		"section_id": 1,
		"started_at": time.Now().Add(-1 * time.Hour),
	})

	// Test getting server health
	health, err := db.GetServerHealth(ctx)
	if err != nil {
		t.Fatalf("GetServerHealth failed: %v", err)
	}

	if health.ServerStatus == "" {
		t.Error("Server status should not be empty")
	}

	if health.UptimePercent <= 0 {
		t.Error("Uptime percent should be positive")
	}
}

func TestGetUserRecommendations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Insert test data - user 1 and user 2 both watched movie1
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type": "movie",
		"title":      "Shared Movie",
		"rating_key": "movie1",
		"user_id":    1,
		"started_at": time.Now().Add(-2 * time.Hour),
	})
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type": "movie",
		"title":      "Shared Movie",
		"rating_key": "movie1",
		"user_id":    2,
		"started_at": time.Now().Add(-2 * time.Hour),
	})

	// User 2 also watched movie2, which user 1 hasn't seen
	insertTestPlaybackEvent(t, db, map[string]interface{}{
		"media_type": "movie",
		"title":      "Recommended Movie",
		"rating_key": "movie2",
		"user_id":    2,
		"started_at": time.Now().Add(-1 * time.Hour),
	})

	// Test getting recommendations for user 1
	recs, err := db.GetUserRecommendations(ctx, "1", 10)
	if err != nil {
		t.Fatalf("GetUserRecommendations failed: %v", err)
	}

	// User 1 should be recommended movie2 (watched by similar user 2)
	// Note: This test may have varying results depending on data
	t.Logf("Got %d recommendations for user 1", len(recs))
}

func TestGetRecentlyAddedMovies_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Query with future time should return no results
	since := time.Now().Add(24 * time.Hour)
	movies, err := db.GetRecentlyAddedMovies(ctx, since, 10)
	if err != nil {
		t.Fatalf("GetRecentlyAddedMovies failed: %v", err)
	}

	if len(movies) != 0 {
		t.Errorf("Expected 0 movies, got %d", len(movies))
	}
}

func TestGetPeriodStats_EmptyPeriod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Query empty time period
	start := time.Now().Add(24 * time.Hour)
	end := time.Now().Add(48 * time.Hour)

	stats, err := db.GetPeriodStats(ctx, start, end)
	if err != nil {
		t.Fatalf("GetPeriodStats failed: %v", err)
	}

	if stats.TotalPlaybacks != 0 {
		t.Errorf("Expected 0 playbacks in empty period, got %d", stats.TotalPlaybacks)
	}
}

// Helper function to insert test playback events
func insertTestPlaybackEvent(t *testing.T, db *DB, fields map[string]interface{}) {
	t.Helper()

	// Set defaults
	if _, ok := fields["started_at"]; !ok {
		fields["started_at"] = time.Now()
	}
	if _, ok := fields["user_id"]; !ok {
		fields["user_id"] = 1
	}
	if _, ok := fields["username"]; !ok {
		fields["username"] = "testuser"
	}
	if _, ok := fields["ip_address"]; !ok {
		fields["ip_address"] = "127.0.0.1"
	}
	if _, ok := fields["session_key"]; !ok {
		fields["session_key"] = "session-" + fields["rating_key"].(string)
	}

	query := `
		INSERT INTO playback_events (
			id, session_key, started_at, stopped_at, user_id, username,
			ip_address, media_type, title, rating_key, year, genres,
			grandparent_title, grandparent_rating_key, added_at, section_id, platform
		) VALUES (
			gen_random_uuid(), ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?
		)
	`

	_, err := db.conn.Exec(query,
		fields["session_key"],
		fields["started_at"],
		fields["stopped_at"],
		fields["user_id"],
		fields["username"],
		fields["ip_address"],
		fields["media_type"],
		fields["title"],
		fields["rating_key"],
		fields["year"],
		fields["genres"],
		fields["grandparent_title"],
		fields["grandparent_rating_key"],
		fields["added_at"],
		fields["section_id"],
		fields["platform"],
	)
	if err != nil {
		t.Fatalf("Failed to insert test playback event: %v", err)
	}
}
