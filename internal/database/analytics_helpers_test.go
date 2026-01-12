// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
	"time"
)

func TestBuildWhereClauseWithArgs(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	tests := []struct {
		name           string
		filter         LocationStatsFilter
		baseConditions []string
		wantClause     string
		wantArgsCount  int
	}{
		{
			name:           "empty filter with no base conditions",
			filter:         LocationStatsFilter{},
			baseConditions: nil,
			wantClause:     "",
			wantArgsCount:  0,
		},
		{
			name:           "empty filter with base condition",
			filter:         LocationStatsFilter{},
			baseConditions: []string{"media_type = 'episode'"},
			wantClause:     "media_type = 'episode'",
			wantArgsCount:  0,
		},
		{
			name: "start date filter",
			filter: LocationStatsFilter{
				StartDate: ptrTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			baseConditions: nil,
			wantClause:     "started_at >= ?",
			wantArgsCount:  1,
		},
		{
			name: "end date filter",
			filter: LocationStatsFilter{
				EndDate: ptrTime(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)),
			},
			baseConditions: nil,
			wantClause:     "started_at <= ?",
			wantArgsCount:  1,
		},
		{
			name: "date range filter",
			filter: LocationStatsFilter{
				StartDate: ptrTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				EndDate:   ptrTime(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)),
			},
			baseConditions: nil,
			wantClause:     "started_at >= ? AND started_at <= ?",
			wantArgsCount:  2,
		},
		{
			name: "single user filter",
			filter: LocationStatsFilter{
				Users: []string{"testuser"},
			},
			baseConditions: nil,
			wantClause:     "username IN (?)",
			wantArgsCount:  1,
		},
		{
			name: "multiple users filter",
			filter: LocationStatsFilter{
				Users: []string{"user1", "user2", "user3"},
			},
			baseConditions: nil,
			wantClause:     "username IN (?, ?, ?)",
			wantArgsCount:  3,
		},
		{
			name: "media types filter",
			filter: LocationStatsFilter{
				MediaTypes: []string{"movie", "episode"},
			},
			baseConditions: nil,
			wantClause:     "media_type IN (?, ?)",
			wantArgsCount:  2,
		},
		{
			name: "platforms filter",
			filter: LocationStatsFilter{
				Platforms: []string{"Roku", "Apple TV"},
			},
			baseConditions: nil,
			wantClause:     "platform IN (?, ?)",
			wantArgsCount:  2,
		},
		{
			name: "players filter",
			filter: LocationStatsFilter{
				Players: []string{"Plex Web", "Plex for iOS"},
			},
			baseConditions: nil,
			wantClause:     "player IN (?, ?)",
			wantArgsCount:  2,
		},
		{
			name: "combined filters with base condition",
			filter: LocationStatsFilter{
				StartDate:  ptrTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				EndDate:    ptrTime(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)),
				Users:      []string{"user1", "user2"},
				MediaTypes: []string{"episode"},
				Platforms:  []string{"Roku"},
				Players:    []string{"Plex for Roku"},
			},
			baseConditions: []string{"percent_complete > 90"},
			wantClause:     "percent_complete > 90 AND started_at >= ? AND started_at <= ? AND username IN (?, ?) AND media_type IN (?) AND platform IN (?) AND player IN (?)",
			wantArgsCount:  7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClause, gotArgs := buildWhereClauseWithArgs(tt.filter, tt.baseConditions...)

			if gotClause != tt.wantClause {
				t.Errorf("buildWhereClauseWithArgs() clause = %v, want %v", gotClause, tt.wantClause)
			}

			if len(gotArgs) != tt.wantArgsCount {
				t.Errorf("buildWhereClauseWithArgs() args count = %d, want %d", len(gotArgs), tt.wantArgsCount)
			}
		})
	}
}

func TestBuildBingeSQLWithWhere(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	sql := buildBingeSQLWithWhere()

	// Verify SQL contains essential components
	if sql == "" {
		t.Fatal("buildBingeSQLWithWhere() returned empty string")
	}

	// Check for required CTEs
	requiredPatterns := []string{
		"WITH episode_sessions AS",
		"session_markers AS",
		"session_groups AS",
		"binge_sessions AS",
		"WHERE %s", // Placeholder for WHERE clause
		"LIMIT 50",
	}

	for _, pattern := range requiredPatterns {
		if !containsSubstring(sql, pattern) {
			t.Errorf("buildBingeSQLWithWhere() missing pattern: %s", pattern)
		}
	}
}

func TestBuildBingeSummarySQL(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	sql := buildBingeSummarySQL()

	// Verify SQL contains essential components
	if sql == "" {
		t.Fatal("buildBingeSummarySQL() returned empty string")
	}

	// Check for required elements
	requiredPatterns := []string{
		"WITH episode_sessions AS",
		"COUNT(*) as total_binge_sessions",
		"PERCENTILE_CONT(0.5)",
		"WHERE %s",
	}

	for _, pattern := range requiredPatterns {
		if !containsSubstring(sql, pattern) {
			t.Errorf("buildBingeSummarySQL() missing pattern: %s", pattern)
		}
	}
}

func TestBuildTopShowsSQL(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	sql := buildTopShowsSQL()

	// Verify SQL contains essential components
	if sql == "" {
		t.Fatal("buildTopShowsSQL() returned empty string")
	}

	// Check for required elements
	requiredPatterns := []string{
		"WITH episode_sessions AS",
		"binge_sessions AS",
		"GROUP BY show_name",
		"ORDER BY session_count DESC",
		"LIMIT 20",
	}

	for _, pattern := range requiredPatterns {
		if !containsSubstring(sql, pattern) {
			t.Errorf("buildTopShowsSQL() missing pattern: %s", pattern)
		}
	}
}

// Helper functions for tests

func ptrTime(t time.Time) *time.Time {
	return &t
}

func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(s) < len(substr) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
