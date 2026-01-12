// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
	"time"
)

// TestAppendInClause tests the appendInClause helper function
func TestAppendInClause(t *testing.T) {

	t.Run("string slice with question marks", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 1

		appendInClause("username", []string{"alice", "bob", "charlie"}, &whereClauses, &args, &argPos, false)

		if len(whereClauses) != 1 {
			t.Errorf("Expected 1 where clause, got %d", len(whereClauses))
		}
		if whereClauses[0] != "username IN (?, ?, ?)" {
			t.Errorf("Expected 'username IN (?, ?, ?)', got %q", whereClauses[0])
		}
		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
		if argPos != 4 {
			t.Errorf("Expected argPos to be 4, got %d", argPos)
		}
	})

	t.Run("string slice with positional params", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 1

		appendInClause("username", []string{"alice", "bob"}, &whereClauses, &args, &argPos, true)

		if whereClauses[0] != "username IN ($1, $2)" {
			t.Errorf("Expected 'username IN ($1, $2)', got %q", whereClauses[0])
		}
		if argPos != 3 {
			t.Errorf("Expected argPos to be 3, got %d", argPos)
		}
	})

	t.Run("int slice with question marks", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 1

		appendInClause("year", []int{2020, 2021, 2022}, &whereClauses, &args, &argPos, false)

		if len(whereClauses) != 1 {
			t.Errorf("Expected 1 where clause, got %d", len(whereClauses))
		}
		if whereClauses[0] != "year IN (?, ?, ?)" {
			t.Errorf("Expected 'year IN (?, ?, ?)', got %q", whereClauses[0])
		}
		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
		// Verify args contain the actual values
		if args[0] != 2020 || args[1] != 2021 || args[2] != 2022 {
			t.Errorf("Args values incorrect: %v", args)
		}
	})

	t.Run("int slice with positional params", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 5 // Start at position 5

		appendInClause("year", []int{2023}, &whereClauses, &args, &argPos, true)

		if whereClauses[0] != "year IN ($5)" {
			t.Errorf("Expected 'year IN ($5)', got %q", whereClauses[0])
		}
		if argPos != 6 {
			t.Errorf("Expected argPos to be 6, got %d", argPos)
		}
	})

	t.Run("empty string slice does nothing", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 1

		appendInClause("username", []string{}, &whereClauses, &args, &argPos, false)

		if len(whereClauses) != 0 {
			t.Errorf("Expected 0 where clauses, got %d", len(whereClauses))
		}
		if len(args) != 0 {
			t.Errorf("Expected 0 args, got %d", len(args))
		}
		if argPos != 1 {
			t.Errorf("Expected argPos to remain 1, got %d", argPos)
		}
	})

	t.Run("empty int slice does nothing", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 1

		appendInClause("year", []int{}, &whereClauses, &args, &argPos, false)

		if len(whereClauses) != 0 {
			t.Errorf("Expected 0 where clauses, got %d", len(whereClauses))
		}
		if len(args) != 0 {
			t.Errorf("Expected 0 args, got %d", len(args))
		}
	})

	t.Run("unknown type does nothing", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 1

		// Pass an unsupported type ([]float64)
		appendInClause("value", []float64{1.5, 2.5}, &whereClauses, &args, &argPos, false)

		if len(whereClauses) != 0 {
			t.Errorf("Expected 0 where clauses for unknown type, got %d", len(whereClauses))
		}
		if len(args) != 0 {
			t.Errorf("Expected 0 args for unknown type, got %d", len(args))
		}
		if argPos != 1 {
			t.Errorf("Expected argPos to remain 1 for unknown type, got %d", argPos)
		}
	})

	t.Run("nil value does nothing", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 1

		appendInClause("username", nil, &whereClauses, &args, &argPos, false)

		if len(whereClauses) != 0 {
			t.Errorf("Expected 0 where clauses for nil, got %d", len(whereClauses))
		}
	})

	t.Run("single item string slice", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 1

		appendInClause("platform", []string{"iOS"}, &whereClauses, &args, &argPos, false)

		if whereClauses[0] != "platform IN (?)" {
			t.Errorf("Expected 'platform IN (?)', got %q", whereClauses[0])
		}
		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
		if args[0] != "iOS" {
			t.Errorf("Expected 'iOS', got %v", args[0])
		}
	})

	t.Run("multiple calls accumulate", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 1

		appendInClause("username", []string{"alice"}, &whereClauses, &args, &argPos, false)
		appendInClause("platform", []string{"iOS", "Android"}, &whereClauses, &args, &argPos, false)
		appendInClause("year", []int{2024}, &whereClauses, &args, &argPos, false)

		if len(whereClauses) != 3 {
			t.Errorf("Expected 3 where clauses, got %d", len(whereClauses))
		}
		if len(args) != 4 { // 1 + 2 + 1
			t.Errorf("Expected 4 args, got %d", len(args))
		}
		if argPos != 5 {
			t.Errorf("Expected argPos to be 5, got %d", argPos)
		}
	})

	t.Run("special characters in strings", func(t *testing.T) {
		var whereClauses []string
		var args []interface{}
		argPos := 1

		appendInClause("username", []string{"user@domain.com", "user'quote", "user\"double"}, &whereClauses, &args, &argPos, false)

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
		// Verify values are passed through unchanged (parameterized queries handle escaping)
		if args[0] != "user@domain.com" {
			t.Errorf("First arg incorrect: %v", args[0])
		}
		if args[1] != "user'quote" {
			t.Errorf("Second arg incorrect: %v", args[1])
		}
		if args[2] != "user\"double" {
			t.Errorf("Third arg incorrect: %v", args[2])
		}
	})
}

// TestBuildFilterConditionsComprehensive tests buildFilterConditions with all filter types
func TestBuildFilterConditionsComprehensive(t *testing.T) {

	t.Run("start date only with question marks", func(t *testing.T) {
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		filter := LocationStatsFilter{
			StartDate: &start,
		}

		whereClauses, args := buildFilterConditions(filter, false, 1)

		if len(whereClauses) != 1 {
			t.Errorf("Expected 1 clause, got %d", len(whereClauses))
		}
		if whereClauses[0] != "started_at >= ?" {
			t.Errorf("Expected 'started_at >= ?', got %q", whereClauses[0])
		}
		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})

	t.Run("start date only with positional params", func(t *testing.T) {
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		filter := LocationStatsFilter{
			StartDate: &start,
		}

		whereClauses, args := buildFilterConditions(filter, true, 1)

		if whereClauses[0] != "started_at >= $1" {
			t.Errorf("Expected 'started_at >= $1', got %q", whereClauses[0])
		}
		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})

	t.Run("all multi-value filters", func(t *testing.T) {
		filter := LocationStatsFilter{
			Users:              []string{"user1"},
			MediaTypes:         []string{"movie"},
			Platforms:          []string{"Roku"},
			Players:            []string{"Plex"},
			TranscodeDecisions: []string{"direct play"},
			VideoResolutions:   []string{"1080p"},
			VideoCodecs:        []string{"h264"},
			AudioCodecs:        []string{"aac"},
			Libraries:          []string{"Movies"},
			ContentRatings:     []string{"PG-13"},
			Years:              []int{2024},
			LocationTypes:      []string{"WAN"},
		}

		whereClauses, args := buildFilterConditions(filter, false, 1)

		// Should have 12 clauses (all multi-value filters)
		if len(whereClauses) != 12 {
			t.Errorf("Expected 12 clauses, got %d", len(whereClauses))
		}
		// Should have 12 args (one for each single-value filter)
		if len(args) != 12 {
			t.Errorf("Expected 12 args, got %d", len(args))
		}
	})

	t.Run("positional params start at custom position", func(t *testing.T) {
		filter := LocationStatsFilter{
			Users: []string{"alice", "bob"},
		}

		whereClauses, _ := buildFilterConditions(filter, true, 10)

		if whereClauses[0] != "username IN ($10, $11)" {
			t.Errorf("Expected 'username IN ($10, $11)', got %q", whereClauses[0])
		}
	})

	t.Run("empty multi-value slices are ignored", func(t *testing.T) {
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		filter := LocationStatsFilter{
			StartDate:  &start,
			Users:      []string{},
			MediaTypes: []string{},
			Platforms:  []string{},
		}

		whereClauses, args := buildFilterConditions(filter, false, 1)

		// Only the date filter should be present
		if len(whereClauses) != 1 {
			t.Errorf("Expected 1 clause (only date), got %d", len(whereClauses))
		}
		if len(args) != 1 {
			t.Errorf("Expected 1 arg (only date), got %d", len(args))
		}
	})

	t.Run("nil pointer fields are ignored", func(t *testing.T) {
		filter := LocationStatsFilter{
			StartDate: nil,
			EndDate:   nil,
			Users:     []string{"alice"},
		}

		whereClauses, args := buildFilterConditions(filter, false, 1)

		if len(whereClauses) != 1 {
			t.Errorf("Expected 1 clause, got %d", len(whereClauses))
		}
		if whereClauses[0] != "username IN (?)" {
			t.Errorf("Expected 'username IN (?)', got %q", whereClauses[0])
		}
		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})
}

// TestJoinHelper tests the join helper function
func TestJoinHelper(t *testing.T) {

	tests := []struct {
		name     string
		strs     []string
		sep      string
		expected string
	}{
		{
			name:     "empty slice",
			strs:     []string{},
			sep:      ", ",
			expected: "",
		},
		{
			name:     "single element",
			strs:     []string{"a"},
			sep:      ", ",
			expected: "a",
		},
		{
			name:     "two elements",
			strs:     []string{"a", "b"},
			sep:      ", ",
			expected: "a, b",
		},
		{
			name:     "multiple elements",
			strs:     []string{"x", "y", "z"},
			sep:      "-",
			expected: "x-y-z",
		},
		{
			name:     "empty separator",
			strs:     []string{"a", "b", "c"},
			sep:      "",
			expected: "abc",
		},
		{
			name:     "long separator",
			strs:     []string{"foo", "bar"},
			sep:      " AND ",
			expected: "foo AND bar",
		},
		{
			name:     "elements with spaces",
			strs:     []string{"hello world", "foo bar"},
			sep:      ", ",
			expected: "hello world, foo bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := join(tt.strs, tt.sep)
			if result != tt.expected {
				t.Errorf("join(%v, %q) = %q, want %q", tt.strs, tt.sep, result, tt.expected)
			}
		})
	}
}

// TestFilterFieldOrderConsistency tests that filters are applied in consistent order
func TestFilterFieldOrderConsistency(t *testing.T) {

	// Run multiple times to verify consistent ordering
	for i := 0; i < 5; i++ {
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

		filter := LocationStatsFilter{
			StartDate:  &start,
			EndDate:    &end,
			Users:      []string{"alice"},
			MediaTypes: []string{"movie"},
		}

		whereClauses, _ := buildFilterConditions(filter, false, 1)

		// Order should be: start date, end date, users, media types
		expectedOrder := []string{
			"started_at >= ?",
			"started_at <= ?",
			"username IN (?)",
			"media_type IN (?)",
		}

		if len(whereClauses) != len(expectedOrder) {
			t.Fatalf("Iteration %d: Expected %d clauses, got %d", i, len(expectedOrder), len(whereClauses))
		}

		for j, expected := range expectedOrder {
			if whereClauses[j] != expected {
				t.Errorf("Iteration %d, clause %d: Expected %q, got %q", i, j, expected, whereClauses[j])
			}
		}
	}
}

// BenchmarkBuildFilterConditions benchmarks filter condition building
func BenchmarkBuildFilterConditions(b *testing.B) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	b.Run("minimal filter", func(b *testing.B) {
		filter := LocationStatsFilter{
			Users: []string{"alice"},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buildFilterConditions(filter, false, 1)
		}
	})

	b.Run("full filter", func(b *testing.B) {
		filter := LocationStatsFilter{
			StartDate:          &start,
			EndDate:            &end,
			Users:              []string{"alice", "bob", "charlie"},
			MediaTypes:         []string{"movie", "episode"},
			Platforms:          []string{"Roku", "iOS", "Android"},
			Players:            []string{"Plex Web", "Plex App"},
			TranscodeDecisions: []string{"direct play", "transcode"},
			VideoResolutions:   []string{"1080p", "4k"},
			VideoCodecs:        []string{"h264", "hevc"},
			AudioCodecs:        []string{"aac", "ac3"},
			Libraries:          []string{"Movies", "TV Shows"},
			ContentRatings:     []string{"PG", "PG-13", "R"},
			Years:              []int{2020, 2021, 2022, 2023, 2024},
			LocationTypes:      []string{"LAN", "WAN"},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buildFilterConditions(filter, false, 1)
		}
	})

	b.Run("positional params", func(b *testing.B) {
		filter := LocationStatsFilter{
			Users:      []string{"alice", "bob"},
			MediaTypes: []string{"movie"},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buildFilterConditions(filter, true, 1)
		}
	})
}

// BenchmarkAppendInClause benchmarks the appendInClause helper
func BenchmarkAppendInClause(b *testing.B) {
	b.Run("small string slice", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var whereClauses []string
			var args []interface{}
			argPos := 1
			appendInClause("username", []string{"alice", "bob"}, &whereClauses, &args, &argPos, false)
		}
	})

	b.Run("large string slice", func(b *testing.B) {
		users := make([]string, 100)
		for i := range users {
			users[i] = "user"
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var whereClauses []string
			var args []interface{}
			argPos := 1
			appendInClause("username", users, &whereClauses, &args, &argPos, false)
		}
	})

	b.Run("int slice", func(b *testing.B) {
		years := []int{2020, 2021, 2022, 2023, 2024}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var whereClauses []string
			var args []interface{}
			argPos := 1
			appendInClause("year", years, &whereClauses, &args, &argPos, false)
		}
	})
}
