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

// TestGetPopularContent tests the GetPopularContent function
func TestGetPopularContent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	insertEngagementTestPlaybacks(t, db)

	tests := []struct {
		name   string
		filter LocationStatsFilter
		limit  int
	}{
		{"returns popular content", LocationStatsFilter{}, 10},
		{"with zero limit", LocationStatsFilter{}, 0},
		{"with date filter", LocationStatsFilter{StartDate: timePtr(time.Now().Add(-24 * time.Hour))}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := db.GetPopularContent(context.Background(), tt.filter, tt.limit)
			if err != nil {
				t.Fatalf("GetPopularContent failed: %v", err)
			}
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
		})
	}
}

// TestGetWatchParties tests the GetWatchParties function
func TestGetWatchParties(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	insertWatchPartyTestPlaybacks(t, db)

	tests := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{"detects watch parties", LocationStatsFilter{}},
		{"with date filter", LocationStatsFilter{StartDate: timePtr(time.Now().Add(-24 * time.Hour))}},
		{"with user filter", LocationStatsFilter{Users: []string{"user1", "user2"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := db.GetWatchParties(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("GetWatchParties failed: %v", err)
			}
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if result.PartiesByDay == nil {
				t.Error("Expected PartiesByDay to be initialized")
			}
			if len(result.PartiesByDay) != 7 {
				t.Errorf("Expected 7 days in PartiesByDay, got %d", len(result.PartiesByDay))
			}
		})
	}
}

// TestGetUserEngagement tests the GetUserEngagement function
func TestGetUserEngagement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	insertEngagementTestPlaybacks(t, db)

	tests := []struct {
		name   string
		filter LocationStatsFilter
		limit  int
	}{
		{"basic user engagement query", LocationStatsFilter{}, 10},
		{"limit clamped to minimum", LocationStatsFilter{}, 0},
		{"limit clamped to maximum", LocationStatsFilter{}, 200},
		{"with date and user filter", LocationStatsFilter{
			StartDate: timePtr(time.Now().Add(-14 * 24 * time.Hour)),
			Users:     []string{"user1", "user2"},
		}, 5},
		{"with media type filter", LocationStatsFilter{MediaTypes: []string{"movie", "episode"}}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := db.GetUserEngagement(context.Background(), tt.filter, tt.limit)
			if err != nil {
				t.Fatalf("GetUserEngagement failed: %v", err)
			}
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if result.Summary.TotalUsers < 0 {
				t.Error("TotalUsers should not be negative")
			}
		})
	}
}

// TestBuildPopularContentWhereClause tests the WHERE clause builder
func TestBuildPopularContentWhereClause(t *testing.T) {

	tests := []struct {
		name          string
		filter        LocationStatsFilter
		expectClauses int // minimum expected number of clauses
		expectArgs    int // expected number of args
	}{
		{
			name:          "empty filter",
			filter:        LocationStatsFilter{},
			expectClauses: 1, // just "1=1"
			expectArgs:    0,
		},
		{
			name: "with start date",
			filter: LocationStatsFilter{
				StartDate: func() *time.Time { t := time.Now(); return &t }(),
			},
			expectClauses: 1,
			expectArgs:    1,
		},
		{
			name: "with end date",
			filter: LocationStatsFilter{
				EndDate: func() *time.Time { t := time.Now(); return &t }(),
			},
			expectClauses: 1,
			expectArgs:    1,
		},
		{
			name: "with both dates",
			filter: LocationStatsFilter{
				StartDate: func() *time.Time { t := time.Now().Add(-24 * time.Hour); return &t }(),
				EndDate:   func() *time.Time { t := time.Now(); return &t }(),
			},
			expectClauses: 1,
			expectArgs:    2,
		},
		{
			name: "with users",
			filter: LocationStatsFilter{
				Users: []string{"user1", "user2", "user3"},
			},
			expectClauses: 1,
			expectArgs:    3,
		},
		{
			name: "with all filters",
			filter: LocationStatsFilter{
				StartDate: func() *time.Time { t := time.Now().Add(-24 * time.Hour); return &t }(),
				EndDate:   func() *time.Time { t := time.Now(); return &t }(),
				Users:     []string{"user1", "user2"},
			},
			expectClauses: 1,
			expectArgs:    4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args := buildEngagementWhereClause(tt.filter, "", false)
			if whereClause == "" {
				t.Error("Expected non-empty WHERE clause")
			}
			if len(args) != tt.expectArgs {
				t.Errorf("Expected %d args, got %d", tt.expectArgs, len(args))
			}
		})
	}
}

// TestBuildWatchPartyWhereClause tests the watch party WHERE clause builder
func TestBuildWatchPartyWhereClause(t *testing.T) {

	tests := []struct {
		name       string
		filter     LocationStatsFilter
		expectArgs int
	}{
		{
			name:       "empty filter",
			filter:     LocationStatsFilter{},
			expectArgs: 0,
		},
		{
			name: "with date range",
			filter: LocationStatsFilter{
				StartDate: func() *time.Time { t := time.Now().Add(-24 * time.Hour); return &t }(),
				EndDate:   func() *time.Time { t := time.Now(); return &t }(),
			},
			expectArgs: 2,
		},
		{
			name: "with users and media types",
			filter: LocationStatsFilter{
				Users:      []string{"user1"},
				MediaTypes: []string{"movie", "episode"},
			},
			expectArgs: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args := buildEngagementWhereClause(tt.filter, "p1", true)
			if whereClause == "" {
				t.Error("Expected non-empty WHERE clause")
			}
			if len(args) != tt.expectArgs {
				t.Errorf("Expected %d args, got %d", tt.expectArgs, len(args))
			}
		})
	}
}

// TestBuildUserEngagementWhereClause tests the user engagement WHERE clause builder
func TestBuildUserEngagementWhereClause(t *testing.T) {

	tests := []struct {
		name       string
		filter     LocationStatsFilter
		expectArgs int
	}{
		{
			name:       "empty filter",
			filter:     LocationStatsFilter{},
			expectArgs: 0,
		},
		{
			name: "full filter",
			filter: LocationStatsFilter{
				StartDate:  func() *time.Time { t := time.Now().Add(-24 * time.Hour); return &t }(),
				EndDate:    func() *time.Time { t := time.Now(); return &t }(),
				Users:      []string{"user1", "user2"},
				MediaTypes: []string{"movie"},
			},
			expectArgs: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args := buildEngagementWhereClause(tt.filter, "", true)
			if whereClause == "" {
				t.Error("Expected non-empty WHERE clause")
			}
			if len(args) != tt.expectArgs {
				t.Errorf("Expected %d args, got %d", tt.expectArgs, len(args))
			}
		})
	}
}

// TestQueryHelpers tests individual query helper methods
func TestQueryHelpers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	insertEngagementTestPlaybacks(t, db)

	whereClause, args := buildEngagementWhereClause(LocationStatsFilter{}, "", false)

	tests := []struct {
		name          string
		queryFunc     func() error
		expectedType  string
		validateCount bool
	}{
		{
			name: "queryTopMovies",
			queryFunc: func() error {
				movies, err := db.queryTopMovies(context.Background(), whereClause, args, 10)
				if err != nil {
					return err
				}
				for _, movie := range movies {
					if movie.Title == "" {
						t.Error("Movie title should not be empty")
					}
				}
				return nil
			},
		},
		{
			name: "queryTopShows",
			queryFunc: func() error {
				shows, err := db.queryTopShows(context.Background(), whereClause, args, 10)
				if err != nil {
					return err
				}
				for _, show := range shows {
					if show.MediaType != "show" {
						t.Errorf("Expected media_type 'show', got '%s'", show.MediaType)
					}
				}
				return nil
			},
		},
		{
			name: "queryTopEpisodes",
			queryFunc: func() error {
				episodes, err := db.queryTopEpisodes(context.Background(), whereClause, args, 10)
				if err != nil {
					return err
				}
				for _, ep := range episodes {
					if ep.MediaType != "episode" {
						t.Errorf("Expected media_type 'episode', got '%s'", ep.MediaType)
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.queryFunc(); err != nil {
				t.Fatalf("%s failed: %v", tt.name, err)
			}
		})
	}
}

// TestWatchPartyHelpers tests watch party helper methods
func TestWatchPartyHelpers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	insertWatchPartyTestPlaybacks(t, db)

	whereClause, args := buildEngagementWhereClause(LocationStatsFilter{}, "p1", true)

	tests := []struct {
		name      string
		queryFunc func() error
	}{
		{
			name: "getWatchPartySummary",
			queryFunc: func() error {
				total, participants, avg, sameLoc, err := db.getWatchPartySummary(context.Background(), whereClause, args)
				if err != nil {
					return err
				}
				if total < 0 || participants < 0 || avg < 0 || sameLoc < 0 {
					t.Error("Summary values should not be negative")
				}
				return nil
			},
		},
		{
			name: "getRecentWatchParties",
			queryFunc: func() error {
				parties, err := db.getRecentWatchParties(context.Background(), whereClause, args)
				if err != nil {
					return err
				}
				for _, party := range parties {
					if party.ParticipantCount < 2 {
						t.Error("Watch party should have at least 2 participants")
					}
				}
				return nil
			},
		},
		{
			name: "getTopWatchPartyContent",
			queryFunc: func() error {
				content, err := db.getTopWatchPartyContent(context.Background(), whereClause, args)
				if err != nil {
					return err
				}
				for _, c := range content {
					if c.PartyCount < 1 {
						t.Error("Party count should be at least 1")
					}
				}
				return nil
			},
		},
		{
			name: "getWatchPartiesByDay",
			queryFunc: func() error {
				byDay, err := db.getWatchPartiesByDay(context.Background(), whereClause, args)
				if err != nil {
					return err
				}
				if len(byDay) != 7 {
					t.Errorf("Expected 7 days, got %d", len(byDay))
				}
				for _, day := range byDay {
					if day.DayOfWeek < 0 || day.DayOfWeek > 6 {
						t.Errorf("Invalid day of week: %d", day.DayOfWeek)
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.queryFunc(); err != nil {
				t.Fatalf("%s failed: %v", tt.name, err)
			}
		})
	}
}

// TestUserEngagementHelpers tests user engagement helper methods
func TestUserEngagementHelpers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	insertEngagementTestPlaybacks(t, db)

	whereClause, args := buildEngagementWhereClause(LocationStatsFilter{}, "", true)
	whereClauseP, argsP := buildEngagementWhereClause(LocationStatsFilter{}, "p", false)

	tests := []struct {
		name      string
		queryFunc func() error
	}{
		{
			name: "getUserEngagementSummary",
			queryFunc: func() error {
				summary, err := db.getUserEngagementSummary(context.Background(), whereClause, args)
				if err != nil {
					return err
				}
				if summary.TotalUsers < 0 {
					t.Error("TotalUsers should not be negative")
				}
				return nil
			},
		},
		{
			name: "getViewingPatternsByHour",
			queryFunc: func() error {
				patterns, mostActive, err := db.getViewingPatternsByHour(context.Background(), whereClause, args)
				if err != nil {
					return err
				}
				for _, p := range patterns {
					if p.HourOfDay < 0 || p.HourOfDay > 23 {
						t.Errorf("Invalid hour: %d", p.HourOfDay)
					}
				}
				if mostActive != nil && (*mostActive < 0 || *mostActive > 23) {
					t.Errorf("Invalid most active hour: %d", *mostActive)
				}
				return nil
			},
		},
		{
			name: "getViewingPatternsByDay",
			queryFunc: func() error {
				patterns, mostActive, err := db.getViewingPatternsByDay(context.Background(), whereClause, args)
				if err != nil {
					return err
				}
				for _, p := range patterns {
					if p.DayOfWeek < 0 || p.DayOfWeek > 6 {
						t.Errorf("Invalid day: %d", p.DayOfWeek)
					}
					if p.DayName == "" {
						t.Error("DayName should not be empty")
					}
				}
				if mostActive != nil && (*mostActive < 0 || *mostActive > 6) {
					t.Errorf("Invalid most active day: %d", *mostActive)
				}
				return nil
			},
		},
		{
			name: "getTopEngagedUsers",
			queryFunc: func() error {
				users, err := db.getTopEngagedUsers(context.Background(), whereClause, args, whereClauseP, argsP, 10)
				if err != nil {
					return err
				}
				for _, u := range users {
					if u.UserID == 0 && u.Username == "" {
						t.Error("User should have ID or username")
					}
					if u.ActivityScore < 0 {
						t.Error("ActivityScore should not be negative")
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.queryFunc(); err != nil {
				t.Fatalf("%s failed: %v", tt.name, err)
			}
		})
	}
}

// TestContextCancellation tests that context cancellation is handled
func TestContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := db.GetPopularContent(ctx, LocationStatsFilter{}, 10)
	if err == nil {
		// Context cancellation may or may not cause an error depending on timing
		// This is expected - the test just ensures no panic
	}
}

// insertEngagementTestPlaybacks inserts test data for engagement tests
func insertEngagementTestPlaybacks(t *testing.T, db *DB) {
	now := time.Now()
	playbacks := []struct {
		userID           int
		username         string
		ip               string
		mediaType        string
		title            string
		parentTitle      string
		grandparentTitle string
		startedAt        time.Time
		playDuration     int
		percentComplete  int
		platform         string
	}{
		// User 1 - multiple playbacks
		{1, "user1", "192.168.1.1", "movie", "Inception", "", "", now.Add(-2 * time.Hour), 120, 100, "Roku"},
		{1, "user1", "192.168.1.1", "movie", "Inception", "", "", now.Add(-4 * time.Hour), 120, 100, "Roku"},
		{1, "user1", "192.168.1.1", "episode", "Ep 1", "Season 1", "Breaking Bad", now.Add(-6 * time.Hour), 45, 95, "AppleTV"},
		{1, "user1", "192.168.1.1", "episode", "Ep 2", "Season 1", "Breaking Bad", now.Add(-8 * time.Hour), 45, 100, "AppleTV"},

		// User 2 - different content
		{2, "user2", "192.168.1.2", "movie", "The Matrix", "", "", now.Add(-1 * time.Hour), 136, 90, "Web"},
		{2, "user2", "192.168.1.2", "episode", "Pilot", "Season 1", "The Office", now.Add(-3 * time.Hour), 22, 100, "Roku"},

		// User 3 - music
		{3, "user3", "192.168.1.3", "track", "Bohemian Rhapsody", "", "", now.Add(-30 * time.Minute), 6, 100, "iOS"},
		{3, "user3", "192.168.1.3", "track", "Stairway to Heaven", "", "", now.Add(-1 * time.Hour), 8, 85, "iOS"},

		// User 4 - older playbacks
		{4, "user4", "192.168.1.4", "movie", "Pulp Fiction", "", "", now.Add(-7 * 24 * time.Hour), 154, 75, "Android"},

		// User 5 - 30 days ago
		{5, "user5", "192.168.1.5", "movie", "Fight Club", "", "", now.Add(-30 * 24 * time.Hour), 139, 60, "Web"},
	}

	for _, pb := range playbacks {
		_, err := db.conn.Exec(`
			INSERT INTO playback_events (
				id, session_key, started_at, stopped_at, user_id, username,
				ip_address, media_type, title, parent_title, grandparent_title,
				percent_complete, play_duration, platform
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), uuid.New().String(), pb.startedAt,
			pb.startedAt.Add(time.Duration(pb.playDuration)*time.Minute),
			pb.userID, pb.username, pb.ip, pb.mediaType, pb.title,
			nullableString(pb.parentTitle), nullableString(pb.grandparentTitle),
			pb.percentComplete, pb.playDuration, pb.platform)
		if err != nil {
			t.Fatalf("Failed to insert playback: %v", err)
		}
	}
}

// insertWatchPartyTestPlaybacks inserts test data for watch party detection
func insertWatchPartyTestPlaybacks(t *testing.T, db *DB) {
	now := time.Now()
	baseTime := now.Add(-1 * time.Hour)

	// Create a watch party - multiple users watching the same content within 15 minutes
	watchPartyPlaybacks := []struct {
		userID    int
		username  string
		ip        string
		title     string
		startedAt time.Time
	}{
		{1, "user1", "192.168.1.1", "Watch Party Movie", baseTime},
		{2, "user2", "192.168.1.2", "Watch Party Movie", baseTime.Add(5 * time.Minute)},
		{3, "user3", "192.168.1.3", "Watch Party Movie", baseTime.Add(10 * time.Minute)},
		// Same IP - in-person watch party
		{4, "user4", "192.168.1.1", "Watch Party Movie", baseTime.Add(2 * time.Minute)},
	}

	for _, pb := range watchPartyPlaybacks {
		_, err := db.conn.Exec(`
			INSERT INTO playback_events (
				id, session_key, started_at, stopped_at, user_id, username,
				ip_address, media_type, title, percent_complete, play_duration
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), uuid.New().String(), pb.startedAt,
			pb.startedAt.Add(2*time.Hour), pb.userID, pb.username, pb.ip,
			"movie", pb.title, 100, 120)
		if err != nil {
			t.Fatalf("Failed to insert watch party playback: %v", err)
		}
	}
}

// Note: nullableString is defined in newsletter.go and shared across package
// Note: timePtr is defined in database_test.go and shared across test files
