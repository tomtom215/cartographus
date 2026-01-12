// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
	"time"
)

func TestBuildFilterConditions_EmptyFilter(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	filter := LocationStatsFilter{}

	whereClauses, args := buildFilterConditions(filter, false, 1)

	if len(whereClauses) != 0 {
		t.Errorf("Expected 0 where clauses, got %d", len(whereClauses))
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestBuildFilterConditions_DateRange(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	filter := LocationStatsFilter{
		StartDate: &start,
		EndDate:   &end,
	}

	whereClauses, args := buildFilterConditions(filter, false, 1)

	if len(whereClauses) != 2 {
		t.Errorf("Expected 2 where clauses, got %d", len(whereClauses))
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if whereClauses[0] != "started_at >= ?" {
		t.Errorf("Expected 'started_at >= ?', got '%s'", whereClauses[0])
	}

	if whereClauses[1] != "started_at <= ?" {
		t.Errorf("Expected 'started_at <= ?', got '%s'", whereClauses[1])
	}
}

func TestBuildFilterConditions_Users(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	filter := LocationStatsFilter{
		Users: []string{"alice", "bob", "charlie"},
	}

	whereClauses, args := buildFilterConditions(filter, false, 1)

	if len(whereClauses) != 1 {
		t.Errorf("Expected 1 where clause, got %d", len(whereClauses))
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}

	if whereClauses[0] != "username IN (?, ?, ?)" {
		t.Errorf("Expected 'username IN (?, ?, ?)', got '%s'", whereClauses[0])
	}
}

func TestBuildFilterConditions_PositionalParams(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	filter := LocationStatsFilter{
		Users: []string{"alice", "bob"},
	}

	whereClauses, args := buildFilterConditions(filter, true, 1)

	if len(whereClauses) != 1 {
		t.Errorf("Expected 1 where clause, got %d", len(whereClauses))
	}

	if whereClauses[0] != "username IN ($1, $2)" {
		t.Errorf("Expected 'username IN ($1, $2)', got '%s'", whereClauses[0])
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

func TestBuildFilterConditions_MultipleFilters(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	filter := LocationStatsFilter{
		StartDate:  &start,
		Users:      []string{"alice"},
		MediaTypes: []string{"movie", "episode"},
		Platforms:  []string{"Roku"},
	}

	whereClauses, args := buildFilterConditions(filter, false, 1)

	if len(whereClauses) != 4 {
		t.Errorf("Expected 4 where clauses, got %d", len(whereClauses))
	}

	if len(args) != 5 {
		t.Errorf("Expected 5 args (1 date + 1 user + 2 media + 1 platform), got %d", len(args))
	}
}

func TestBuildFilterConditions_AllFilters(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	filter := LocationStatsFilter{
		StartDate:          &start,
		EndDate:            &end,
		Users:              []string{"alice"},
		MediaTypes:         []string{"movie"},
		Platforms:          []string{"Roku"},
		Players:            []string{"Plex for Roku"},
		TranscodeDecisions: []string{"direct play"},
		VideoResolutions:   []string{"1080p"},
		VideoCodecs:        []string{"H.264"},
		AudioCodecs:        []string{"AAC"},
		Libraries:          []string{"Movies"},
		ContentRatings:     []string{"PG-13"},
		Years:              []int{2023},
		LocationTypes:      []string{"LAN"},
	}

	whereClauses, args := buildFilterConditions(filter, false, 1)

	// Should have 14 where clauses (2 dates + 12 multi-value filters)
	if len(whereClauses) != 14 {
		t.Errorf("Expected 14 where clauses, got %d", len(whereClauses))
	}

	// Should have 14 args (2 dates + 12 single-value filters)
	if len(args) != 14 {
		t.Errorf("Expected 14 args, got %d", len(args))
	}
}

func TestBuildFilterConditions_StartArgPosition(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	filter := LocationStatsFilter{
		Users: []string{"alice", "bob"},
	}

	// Start at position 5
	whereClauses, args := buildFilterConditions(filter, true, 5)

	if len(whereClauses) != 1 {
		t.Errorf("Expected 1 where clause, got %d", len(whereClauses))
	}

	// Should use $5 and $6 instead of $1 and $2
	if whereClauses[0] != "username IN ($5, $6)" {
		t.Errorf("Expected 'username IN ($5, $6)', got '%s'", whereClauses[0])
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

func TestJoin(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	tests := []struct {
		name     string
		input    []string
		sep      string
		expected string
	}{
		{"empty", []string{}, ",", ""},
		{"single", []string{"a"}, ",", "a"},
		{"multiple", []string{"a", "b", "c"}, ",", "a,b,c"},
		{"space separator", []string{"x", "y"}, " ", "x y"},
		{"custom separator", []string{"foo", "bar", "baz"}, " | ", "foo | bar | baz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := join(tt.input, tt.sep)
			if result != tt.expected {
				t.Errorf("join() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildFilterConditions_MediaTypes(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	filter := LocationStatsFilter{
		MediaTypes: []string{"movie", "episode", "track"},
	}

	whereClauses, args := buildFilterConditions(filter, false, 1)

	if len(whereClauses) != 1 {
		t.Errorf("Expected 1 where clause, got %d", len(whereClauses))
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}

	if whereClauses[0] != "media_type IN (?, ?, ?)" {
		t.Errorf("Expected 'media_type IN (?, ?, ?)', got '%s'", whereClauses[0])
	}
}

func TestBuildFilterConditions_Platforms(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	filter := LocationStatsFilter{
		Platforms: []string{"Roku", "Android"},
	}

	whereClauses, args := buildFilterConditions(filter, false, 1)

	if len(whereClauses) != 1 {
		t.Errorf("Expected 1 where clause, got %d", len(whereClauses))
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if whereClauses[0] != "platform IN (?, ?)" {
		t.Errorf("Expected 'platform IN (?, ?)', got '%s'", whereClauses[0])
	}
}

func TestBuildFilterConditions_Years(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	filter := LocationStatsFilter{
		Years: []int{2020, 2021, 2022, 2023},
	}

	whereClauses, args := buildFilterConditions(filter, false, 1)

	if len(whereClauses) != 1 {
		t.Errorf("Expected 1 where clause, got %d", len(whereClauses))
	}

	if len(args) != 4 {
		t.Errorf("Expected 4 args, got %d", len(args))
	}

	if whereClauses[0] != "year IN (?, ?, ?, ?)" {
		t.Errorf("Expected 'year IN (?, ?, ?, ?)', got '%s'", whereClauses[0])
	}
}

func TestBuildFilterConditions_LocationTypes(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	filter := LocationStatsFilter{
		LocationTypes: []string{"LAN", "WAN"},
	}

	whereClauses, args := buildFilterConditions(filter, false, 1)

	if len(whereClauses) != 1 {
		t.Errorf("Expected 1 where clause, got %d", len(whereClauses))
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if whereClauses[0] != "location_type IN (?, ?)" {
		t.Errorf("Expected 'location_type IN (?, ?)', got '%s'", whereClauses[0])
	}
}
