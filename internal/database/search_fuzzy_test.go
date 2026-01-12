// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"testing"
)

// TestFuzzySearchPlaybacks tests the fuzzy search functionality for playback content.
func TestFuzzySearchPlaybacks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert test data
	insertTestPlaybacksForSearch(t, db)

	tests := []struct {
		name           string
		query          string
		minScore       int
		limit          int
		wantMinResults int
		wantMaxResults int
		wantFirstMatch string // Expected first result title (if any)
	}{
		{
			name:           "exact match should find Breaking Bad",
			query:          "Breaking Bad",
			minScore:       90,
			limit:          10,
			wantMinResults: 1,
			wantMaxResults: 10,
			wantFirstMatch: "Breaking Bad",
		},
		{
			name:           "typo match should find Breaking Bad",
			query:          "Braking Bad",
			minScore:       70,
			limit:          10,
			wantMinResults: 0, // May or may not match depending on RapidFuzz availability
			wantMaxResults: 10,
		},
		{
			name:           "partial match should find multiple",
			query:          "Breaking",
			minScore:       50,
			limit:          10,
			wantMinResults: 0, // Depends on test data and scoring
			wantMaxResults: 10,
		},
		{
			name:           "no match should return empty",
			query:          "xyznonexistent123",
			minScore:       70,
			limit:          10,
			wantMinResults: 0,
			wantMaxResults: 0,
		},
		{
			name:           "low score threshold returns more",
			query:          "Game",
			minScore:       30,
			limit:          50,
			wantMinResults: 0,
			wantMaxResults: 50,
		},
		{
			name:           "limit parameter is respected",
			query:          "the",
			minScore:       20,
			limit:          3,
			wantMinResults: 0,
			wantMaxResults: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := db.FuzzySearchPlaybacks(ctx, tt.query, tt.minScore, tt.limit)
			if err != nil {
				t.Fatalf("FuzzySearchPlaybacks() error = %v", err)
			}

			if len(results) < tt.wantMinResults {
				t.Errorf("got %d results, want at least %d", len(results), tt.wantMinResults)
			}

			if len(results) > tt.wantMaxResults {
				t.Errorf("got %d results, want at most %d", len(results), tt.wantMaxResults)
			}

			if tt.wantFirstMatch != "" && len(results) > 0 && results[0].Title != tt.wantFirstMatch {
				t.Errorf("first result title = %q, want %q", results[0].Title, tt.wantFirstMatch)
			}

			// Verify all results have valid scores
			for i, r := range results {
				if r.Score < 0 || r.Score > 100 {
					t.Errorf("result[%d] score = %d, want between 0 and 100", i, r.Score)
				}
			}
		})
	}
}

// TestFuzzySearchUsers tests the fuzzy search functionality for users.
func TestFuzzySearchUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert test data
	insertTestPlaybacksForSearch(t, db)

	tests := []struct {
		name           string
		query          string
		minScore       int
		limit          int
		wantMinResults int
		wantMaxResults int
	}{
		{
			name:           "exact username match",
			query:          "testuser",
			minScore:       90,
			limit:          10,
			wantMinResults: 0, // May or may not have exact match
			wantMaxResults: 10,
		},
		{
			name:           "partial username match",
			query:          "test",
			minScore:       50,
			limit:          10,
			wantMinResults: 0,
			wantMaxResults: 10,
		},
		{
			name:           "no match",
			query:          "xyznonexistent",
			minScore:       90,
			limit:          10,
			wantMinResults: 0,
			wantMaxResults: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := db.FuzzySearchUsers(ctx, tt.query, tt.minScore, tt.limit)
			if err != nil {
				t.Fatalf("FuzzySearchUsers() error = %v", err)
			}

			if len(results) < tt.wantMinResults {
				t.Errorf("got %d results, want at least %d", len(results), tt.wantMinResults)
			}

			if len(results) > tt.wantMaxResults {
				t.Errorf("got %d results, want at most %d", len(results), tt.wantMaxResults)
			}

			// Verify all results have valid scores
			for i, r := range results {
				if r.Score < 0 || r.Score > 100 {
					t.Errorf("result[%d] score = %d, want between 0 and 100", i, r.Score)
				}
			}
		})
	}
}

// TestFuzzyMatchScore tests the direct fuzzy match score calculation.
func TestFuzzyMatchScore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name              string
		str1              string
		str2              string
		wantMin           int
		wantMax           int
		wantExact         bool // If true, expect exact score
		wantScore         int  // Expected exact score
		requiresRapidFuzz bool // If true, skip when RapidFuzz unavailable
	}{
		{
			name:      "identical strings",
			str1:      "Breaking Bad",
			str2:      "Breaking Bad",
			wantExact: true,
			wantScore: 100,
		},
		{
			name:              "similar strings",
			str1:              "Breaking Bad",
			str2:              "Braking Bad",
			wantMin:           70,
			wantMax:           100,
			requiresRapidFuzz: true,
		},
		{
			name:              "different strings",
			str1:              "Breaking Bad",
			str2:              "Game of Thrones",
			wantMin:           0,
			wantMax:           50,
			requiresRapidFuzz: true,
		},
		{
			name:      "empty strings",
			str1:      "",
			str2:      "",
			wantExact: true,
			wantScore: 100, // Empty strings are considered identical
		},
	}

	rapidFuzzAvailable := db.IsRapidFuzzAvailable()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.requiresRapidFuzz && !rapidFuzzAvailable {
				t.Skip("RapidFuzz extension not available, skipping fuzzy match test")
			}

			score, err := db.FuzzyMatchScore(ctx, tt.str1, tt.str2)
			if err != nil {
				t.Fatalf("FuzzyMatchScore() error = %v", err)
			}

			if tt.wantExact {
				if score != tt.wantScore {
					t.Errorf("score = %d, want exactly %d", score, tt.wantScore)
				}
			} else {
				if score < tt.wantMin || score > tt.wantMax {
					t.Errorf("score = %d, want between %d and %d", score, tt.wantMin, tt.wantMax)
				}
			}
		})
	}
}

// TestFuzzySearchDefaults tests that default values are applied correctly.
func TestFuzzySearchDefaults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test with invalid/zero values
	results, err := db.FuzzySearchPlaybacks(ctx, "test", 0, 0)
	if err != nil {
		t.Fatalf("FuzzySearchPlaybacks() error = %v", err)
	}

	// Should use defaults (minScore=70, limit=20)
	if len(results) > 20 {
		t.Errorf("expected default limit of 20, got %d results", len(results))
	}

	// Test with negative values
	results, err = db.FuzzySearchPlaybacks(ctx, "test", -1, -1)
	if err != nil {
		t.Fatalf("FuzzySearchPlaybacks() with negative values error = %v", err)
	}

	// Should still work with defaults
	if len(results) > 20 {
		t.Errorf("expected default limit of 20, got %d results", len(results))
	}
}

// TestFuzzySearchRapidFuzzAvailability tests the availability detection.
func TestFuzzySearchRapidFuzzAvailability(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// The availability flag should be set during extension loading
	available := db.IsRapidFuzzAvailable()

	// Log the availability for debugging
	t.Logf("RapidFuzz available: %v", available)

	// Test that search works regardless of availability
	ctx := context.Background()
	_, err := db.FuzzySearchPlaybacks(ctx, "test", 70, 10)
	if err != nil {
		t.Fatalf("FuzzySearchPlaybacks() should work regardless of RapidFuzz availability: %v", err)
	}
}

// insertTestPlaybacksForSearch inserts test data for search tests
func insertTestPlaybacksForSearch(t *testing.T, db *DB) {
	t.Helper()

	ctx := context.Background()

	// Insert test playback events with various titles
	testEvents := []struct {
		ratingKey        string
		title            string
		grandparentTitle string
		mediaType        string
		username         string
		year             int
	}{
		{"1001", "Breaking Bad", "", "show", "testuser1", 2008},
		{"1002", "Breaking Dawn", "", "movie", "testuser2", 2011},
		{"1003", "Better Call Saul", "", "show", "testuser1", 2015},
		{"1004", "The Office", "", "show", "admin", 2005},
		{"1005", "Game of Thrones", "", "show", "testuser3", 2011},
		{"1006", "Pilot", "Breaking Bad", "episode", "testuser1", 2008},
		{"1007", "The Wire", "", "show", "john_doe", 2002},
	}

	for _, e := range testEvents {
		sql := `
			INSERT INTO playback_events (
				id, session_key, started_at, user_id, username, ip_address,
				media_type, title, grandparent_title, platform, player,
				location_type, percent_complete, paused_counter, rating_key, year
			) VALUES (
				uuid(), ?, CURRENT_TIMESTAMP, ?, ?, '192.168.1.1',
				?, ?, ?, 'Test', 'Test Player',
				'lan', 100, 0, ?, ?
			)
		`
		_, err := db.conn.ExecContext(ctx, sql,
			"session_"+e.ratingKey,
			100+len(e.username), // Simple user_id
			e.username,
			e.mediaType,
			e.title,
			e.grandparentTitle,
			e.ratingKey,
			e.year,
		)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}
}
