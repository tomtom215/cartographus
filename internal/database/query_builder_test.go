// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"strings"
	"testing"
	"time"
)

// TestBuildInClause tests the buildInClause function
func TestBuildInClause(t *testing.T) {
	tests := []struct {
		name                 string
		items                []string
		expectedPlaceholders string
		expectedArgsLen      int
	}{
		{
			name:                 "single item",
			items:                []string{"user1"},
			expectedPlaceholders: "?",
			expectedArgsLen:      1,
		},
		{
			name:                 "multiple items",
			items:                []string{"user1", "user2", "user3"},
			expectedPlaceholders: "?,?,?",
			expectedArgsLen:      3,
		},
		{
			name:                 "empty slice",
			items:                []string{},
			expectedPlaceholders: "",
			expectedArgsLen:      0,
		},
		{
			name:                 "special characters in items",
			items:                []string{"user@domain.com", "user with spaces", "user'quote"},
			expectedPlaceholders: "?,?,?",
			expectedArgsLen:      3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			placeholders, args := buildInClause(tt.items)

			if placeholders != tt.expectedPlaceholders {
				t.Errorf("placeholders: expected %q, got %q", tt.expectedPlaceholders, placeholders)
			}

			if len(args) != tt.expectedArgsLen {
				t.Errorf("args length: expected %d, got %d", tt.expectedArgsLen, len(args))
			}

			// Verify args contain the original items
			for i, item := range tt.items {
				if args[i] != item {
					t.Errorf("args[%d]: expected %q, got %q", i, item, args[i])
				}
			}
		})
	}
}

// TestLocationStatsFilterBuildFilterConditions tests filter condition building
func TestLocationStatsFilterBuildFilterConditions(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	tests := []struct {
		name             string
		filter           LocationStatsFilter
		expectConditions bool
		expectArgs       int
		checkContains    []string
	}{
		{
			name:             "empty filter",
			filter:           LocationStatsFilter{},
			expectConditions: false,
			expectArgs:       0,
		},
		{
			name: "start date only",
			filter: LocationStatsFilter{
				StartDate: &yesterday,
			},
			expectConditions: true,
			expectArgs:       1,
			checkContains:    []string{"p.started_at >= ?"},
		},
		{
			name: "end date only",
			filter: LocationStatsFilter{
				EndDate: &now,
			},
			expectConditions: true,
			expectArgs:       1,
			checkContains:    []string{"p.started_at <= ?"},
		},
		{
			name: "date range",
			filter: LocationStatsFilter{
				StartDate: &yesterday,
				EndDate:   &now,
			},
			expectConditions: true,
			expectArgs:       2,
			checkContains:    []string{"p.started_at >= ?", "p.started_at <= ?"},
		},
		{
			name: "users filter",
			filter: LocationStatsFilter{
				Users: []string{"user1", "user2"},
			},
			expectConditions: true,
			expectArgs:       2,
			checkContains:    []string{"p.username IN (?,?)"},
		},
		{
			name: "media types filter",
			filter: LocationStatsFilter{
				MediaTypes: []string{"movie", "episode"},
			},
			expectConditions: true,
			expectArgs:       2,
			checkContains:    []string{"p.media_type IN (?,?)"},
		},
		{
			name: "all filters combined",
			filter: LocationStatsFilter{
				StartDate:  &yesterday,
				EndDate:    &now,
				Users:      []string{"user1"},
				MediaTypes: []string{"movie"},
			},
			expectConditions: true,
			expectArgs:       4,
			checkContains: []string{
				"p.started_at >= ?",
				"p.started_at <= ?",
				"p.username IN (?)",
				"p.media_type IN (?)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions, args := tt.filter.buildFilterConditions()

			if tt.expectConditions {
				if conditions == "" {
					t.Error("expected conditions but got empty string")
				}
				if !strings.HasPrefix(conditions, " AND ") {
					t.Error("conditions should start with ' AND '")
				}
			} else {
				if conditions != "" {
					t.Errorf("expected empty conditions, got %q", conditions)
				}
			}

			if len(args) != tt.expectArgs {
				t.Errorf("args: expected %d, got %d", tt.expectArgs, len(args))
			}

			for _, substr := range tt.checkContains {
				if !strings.Contains(conditions, substr) {
					t.Errorf("conditions should contain %q, got %q", substr, conditions)
				}
			}
		})
	}
}

// TestNewQueryBuilder tests the query builder constructor
func TestNewQueryBuilder(t *testing.T) {
	baseQuery := "SELECT * FROM playback_events WHERE 1=1"
	qb := newQueryBuilder(baseQuery)

	if qb.baseQuery != baseQuery {
		t.Errorf("baseQuery: expected %q, got %q", baseQuery, qb.baseQuery)
	}

	if len(qb.args) != 0 {
		t.Error("args should be empty initially")
	}

	if len(qb.filters) != 0 {
		t.Error("filters should be empty initially")
	}
}

// TestQueryBuilderAddDateRangeFilter tests date range filtering
func TestQueryBuilderAddDateRangeFilter(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	tests := []struct {
		name          string
		filter        LocationStatsFilter
		expectedArgs  int
		expectStartAt bool
		expectEndAt   bool
	}{
		{
			name:         "empty date range",
			filter:       LocationStatsFilter{},
			expectedArgs: 0,
		},
		{
			name: "start date only",
			filter: LocationStatsFilter{
				StartDate: &yesterday,
			},
			expectedArgs:  1,
			expectStartAt: true,
		},
		{
			name: "end date only",
			filter: LocationStatsFilter{
				EndDate: &now,
			},
			expectedArgs: 1,
			expectEndAt:  true,
		},
		{
			name: "both dates",
			filter: LocationStatsFilter{
				StartDate: &yesterday,
				EndDate:   &now,
			},
			expectedArgs:  2,
			expectStartAt: true,
			expectEndAt:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := newQueryBuilder("SELECT * FROM events WHERE 1=1")
			qb.addDateRangeFilter(tt.filter)

			if len(qb.args) != tt.expectedArgs {
				t.Errorf("args: expected %d, got %d", tt.expectedArgs, len(qb.args))
			}

			query, _ := qb.build("")

			if tt.expectStartAt && !strings.Contains(query, "started_at >= ?") {
				t.Error("query should contain 'started_at >= ?'")
			}

			if tt.expectEndAt && !strings.Contains(query, "started_at <= ?") {
				t.Error("query should contain 'started_at <= ?'")
			}
		})
	}
}

// TestQueryBuilderAddUsersFilter tests user filtering
func TestQueryBuilderAddUsersFilter(t *testing.T) {
	tests := []struct {
		name         string
		users        []string
		expectedArgs int
		expectedIn   string
	}{
		{
			name:         "empty users",
			users:        []string{},
			expectedArgs: 0,
		},
		{
			name:         "single user",
			users:        []string{"user1"},
			expectedArgs: 1,
			expectedIn:   "username IN (?)",
		},
		{
			name:         "multiple users",
			users:        []string{"user1", "user2", "user3"},
			expectedArgs: 3,
			expectedIn:   "username IN (?,?,?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := newQueryBuilder("SELECT * FROM events WHERE 1=1")
			qb.addUsersFilter(tt.users)

			if len(qb.args) != tt.expectedArgs {
				t.Errorf("args: expected %d, got %d", tt.expectedArgs, len(qb.args))
			}

			if tt.expectedIn != "" {
				query, _ := qb.build("")
				if !strings.Contains(query, tt.expectedIn) {
					t.Errorf("query should contain %q", tt.expectedIn)
				}
			}
		})
	}
}

// TestQueryBuilderAddMediaTypesFilter tests media type filtering
func TestQueryBuilderAddMediaTypesFilter(t *testing.T) {
	tests := []struct {
		name         string
		mediaTypes   []string
		expectedArgs int
		expectedIn   string
	}{
		{
			name:         "empty media types",
			mediaTypes:   []string{},
			expectedArgs: 0,
		},
		{
			name:         "single type",
			mediaTypes:   []string{"movie"},
			expectedArgs: 1,
			expectedIn:   "media_type IN (?)",
		},
		{
			name:         "multiple types",
			mediaTypes:   []string{"movie", "episode", "track"},
			expectedArgs: 3,
			expectedIn:   "media_type IN (?,?,?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := newQueryBuilder("SELECT * FROM events WHERE 1=1")
			qb.addMediaTypesFilter(tt.mediaTypes)

			if len(qb.args) != tt.expectedArgs {
				t.Errorf("args: expected %d, got %d", tt.expectedArgs, len(qb.args))
			}

			if tt.expectedIn != "" {
				query, _ := qb.build("")
				if !strings.Contains(query, tt.expectedIn) {
					t.Errorf("query should contain %q", tt.expectedIn)
				}
			}
		})
	}
}

// TestQueryBuilderAddStandardFilters tests combined standard filtering
func TestQueryBuilderAddStandardFilters(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	filter := LocationStatsFilter{
		StartDate:  &yesterday,
		EndDate:    &now,
		Users:      []string{"user1", "user2"},
		MediaTypes: []string{"movie"},
	}

	qb := newQueryBuilder("SELECT * FROM events WHERE 1=1")
	qb.addStandardFilters(filter)

	// Should have 5 args: start date, end date, 2 users, 1 media type
	if len(qb.args) != 5 {
		t.Errorf("args: expected 5, got %d", len(qb.args))
	}

	query, _ := qb.build("")

	expectedParts := []string{
		"started_at >= ?",
		"started_at <= ?",
		"username IN (?,?)",
		"media_type IN (?)",
	}

	for _, part := range expectedParts {
		if !strings.Contains(query, part) {
			t.Errorf("query should contain %q, got %q", part, query)
		}
	}
}

// TestQueryBuilderAddFilter tests custom filter addition
func TestQueryBuilderAddFilter(t *testing.T) {
	qb := newQueryBuilder("SELECT * FROM events WHERE 1=1")
	qb.addFilter("location_type = ?", "lan")
	qb.addFilter("percent_complete > ?", 50)

	if len(qb.args) != 2 {
		t.Errorf("args: expected 2, got %d", len(qb.args))
	}

	if len(qb.filters) != 2 {
		t.Errorf("filters: expected 2, got %d", len(qb.filters))
	}

	query, _ := qb.build("")

	if !strings.Contains(query, "location_type = ?") {
		t.Error("query should contain 'location_type = ?'")
	}

	if !strings.Contains(query, "percent_complete > ?") {
		t.Error("query should contain 'percent_complete > ?'")
	}
}

// TestQueryBuilderAddLimit tests limit addition
func TestQueryBuilderAddLimit(t *testing.T) {
	qb := newQueryBuilder("SELECT * FROM events WHERE 1=1")
	qb.addLimit(100)

	if len(qb.args) != 1 {
		t.Errorf("args: expected 1, got %d", len(qb.args))
	}

	if qb.args[0] != 100 {
		t.Errorf("limit arg: expected 100, got %v", qb.args[0])
	}
}

// TestQueryBuilderBuild tests query building
func TestQueryBuilderBuild(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*queryBuilder)
		suffix        string
		expectedQuery string
		expectedArgs  int
	}{
		{
			name:          "empty builder",
			setup:         func(qb *queryBuilder) {},
			suffix:        "",
			expectedQuery: "SELECT * FROM events WHERE 1=1",
			expectedArgs:  0,
		},
		{
			name: "with filters",
			setup: func(qb *queryBuilder) {
				qb.addFilter("status = ?", "active")
			},
			suffix:        "",
			expectedQuery: "SELECT * FROM events WHERE 1=1 AND status = ?",
			expectedArgs:  1,
		},
		{
			name: "with suffix",
			setup: func(qb *queryBuilder) {
				qb.addLimit(10)
			},
			suffix:        "ORDER BY id DESC LIMIT ?",
			expectedQuery: "SELECT * FROM events WHERE 1=1 ORDER BY id DESC LIMIT ?",
			expectedArgs:  1,
		},
		{
			name: "multiple filters with suffix",
			setup: func(qb *queryBuilder) {
				qb.addFilter("status = ?", "active")
				qb.addFilter("type = ?", "movie")
			},
			suffix:        "ORDER BY created_at DESC",
			expectedQuery: "SELECT * FROM events WHERE 1=1 AND status = ? AND type = ? ORDER BY created_at DESC",
			expectedArgs:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := newQueryBuilder("SELECT * FROM events WHERE 1=1")
			tt.setup(qb)

			query, args := qb.build(tt.suffix)

			if query != tt.expectedQuery {
				t.Errorf("query: expected %q, got %q", tt.expectedQuery, query)
			}

			if len(args) != tt.expectedArgs {
				t.Errorf("args: expected %d, got %d", tt.expectedArgs, len(args))
			}
		})
	}
}

// TestQueryBuilderChaining tests method chaining
func TestQueryBuilderChaining(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	filter := LocationStatsFilter{
		StartDate: &yesterday,
		EndDate:   &now,
	}

	qb := newQueryBuilder("SELECT * FROM events WHERE 1=1").
		addDateRangeFilter(filter).
		addUsersFilter([]string{"user1"}).
		addMediaTypesFilter([]string{"movie", "episode"}).
		addLimit(50)

	// Verify chaining works correctly
	if len(qb.args) != 6 { // 2 dates + 1 user + 2 media types + 1 limit
		t.Errorf("args: expected 6, got %d", len(qb.args))
	}

	if len(qb.filters) != 4 { // 2 date filters + 1 user filter + 1 media type filter
		t.Errorf("filters: expected 4, got %d", len(qb.filters))
	}
}

// TestQueryBuilderEdgeCases tests edge cases
func TestQueryBuilderEdgeCases(t *testing.T) {
	t.Run("nil slice handling", func(t *testing.T) {
		qb := newQueryBuilder("SELECT * FROM events")
		qb.addUsersFilter(nil)
		qb.addMediaTypesFilter(nil)

		if len(qb.args) != 0 {
			t.Error("nil slices should not add args")
		}

		if len(qb.filters) != 0 {
			t.Error("nil slices should not add filters")
		}
	})

	t.Run("special characters in values", func(t *testing.T) {
		qb := newQueryBuilder("SELECT * FROM events")
		qb.addUsersFilter([]string{"user@domain.com", "user'quote", "user\"double"})

		if len(qb.args) != 3 {
			t.Errorf("args: expected 3, got %d", len(qb.args))
		}
	})

	t.Run("empty base query", func(t *testing.T) {
		qb := newQueryBuilder("")
		qb.addFilter("status = ?", "active")

		query, _ := qb.build("LIMIT 10")

		if query != " AND status = ? LIMIT 10" {
			t.Errorf("unexpected query: %q", query)
		}
	})
}

// TestQueryBuilderRealWorldExample tests a real-world query scenario
func TestQueryBuilderRealWorldExample(t *testing.T) {
	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)

	// Build a realistic query
	baseQuery := `
		SELECT p.id, p.title, p.username, p.started_at
		FROM playback_events p
		WHERE 1=1
	`

	filter := LocationStatsFilter{
		StartDate:  &weekAgo,
		EndDate:    &now,
		Users:      []string{"admin", "user1", "user2"},
		MediaTypes: []string{"movie", "episode"},
	}

	qb := newQueryBuilder(strings.TrimSpace(baseQuery)).
		addStandardFilters(filter).
		addLimit(100)

	query, args := qb.build("ORDER BY started_at DESC LIMIT ?")

	// Verify the query structure
	expectedParts := []string{
		"playback_events p",
		"WHERE 1=1",
		"started_at >= ?",
		"started_at <= ?",
		"username IN (?,?,?)",
		"media_type IN (?,?)",
		"ORDER BY started_at DESC",
		"LIMIT ?",
	}

	for _, part := range expectedParts {
		if !strings.Contains(query, part) {
			t.Errorf("query should contain %q", part)
		}
	}

	// 2 dates + 3 users + 2 media types + 1 limit = 8 args
	if len(args) != 8 {
		t.Errorf("args: expected 8, got %d", len(args))
	}
}
