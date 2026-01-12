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
	"github.com/tomtom215/cartographus/internal/models"
)

// TestBuildTrendsWhereClause tests the buildTrendsWhereClause function
func TestBuildTrendsWhereClause(t *testing.T) {

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	tests := []struct {
		name           string
		filter         LocationStatsFilter
		expectedClause string
		expectedArgs   int
	}{
		{
			name:           "Empty filter",
			filter:         LocationStatsFilter{},
			expectedClause: "",
			expectedArgs:   0,
		},
		{
			name: "StartDate only",
			filter: LocationStatsFilter{
				StartDate: &yesterday,
			},
			expectedClause: " AND started_at >= ?",
			expectedArgs:   1,
		},
		{
			name: "EndDate only",
			filter: LocationStatsFilter{
				EndDate: &now,
			},
			expectedClause: " AND started_at <= ?",
			expectedArgs:   1,
		},
		{
			name: "StartDate and EndDate",
			filter: LocationStatsFilter{
				StartDate: &yesterday,
				EndDate:   &now,
			},
			expectedClause: " AND started_at >= ? AND started_at <= ?",
			expectedArgs:   2,
		},
		{
			name: "Single user",
			filter: LocationStatsFilter{
				Users: []string{"user1"},
			},
			expectedClause: " AND username IN (?)",
			expectedArgs:   1,
		},
		{
			name: "Multiple users",
			filter: LocationStatsFilter{
				Users: []string{"user1", "user2", "user3"},
			},
			expectedClause: " AND username IN (?,?,?)",
			expectedArgs:   3,
		},
		{
			name: "Single media type",
			filter: LocationStatsFilter{
				MediaTypes: []string{"movie"},
			},
			expectedClause: " AND media_type IN (?)",
			expectedArgs:   1,
		},
		{
			name: "Multiple media types",
			filter: LocationStatsFilter{
				MediaTypes: []string{"movie", "episode", "track"},
			},
			expectedClause: " AND media_type IN (?,?,?)",
			expectedArgs:   3,
		},
		{
			name: "All filters combined",
			filter: LocationStatsFilter{
				StartDate:  &yesterday,
				EndDate:    &now,
				Users:      []string{"user1", "user2"},
				MediaTypes: []string{"movie", "episode"},
			},
			expectedClause: " AND started_at >= ? AND started_at <= ? AND username IN (?,?) AND media_type IN (?,?)",
			expectedArgs:   6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args := buildTrendsWhereClause(tt.filter)

			if whereClause != tt.expectedClause {
				t.Errorf("buildTrendsWhereClause() whereClause = %q, expected %q", whereClause, tt.expectedClause)
			}

			if len(args) != tt.expectedArgs {
				t.Errorf("buildTrendsWhereClause() args count = %d, expected %d", len(args), tt.expectedArgs)
			}

			// Verify argument values for date filters
			if tt.filter.StartDate != nil && len(args) > 0 {
				if args[0] != *tt.filter.StartDate {
					t.Error("Expected first argument to be StartDate")
				}
			}
		})
	}
}

// TestDetermineTrendsInterval tests the determineTrendsInterval function
func TestDetermineTrendsInterval(t *testing.T) {

	now := time.Now()

	tests := []struct {
		name             string
		minDate          *time.Time
		maxDate          *time.Time
		expectedInterval string
		expectedFormat   string
	}{
		{
			name:             "Nil dates returns day interval",
			minDate:          nil,
			maxDate:          nil,
			expectedInterval: "day",
			expectedFormat:   "strftime(started_at, '%Y-%m-%d')",
		},
		{
			name:             "Nil minDate returns day interval",
			minDate:          nil,
			maxDate:          &now,
			expectedInterval: "day",
			expectedFormat:   "strftime(started_at, '%Y-%m-%d')",
		},
		{
			name:             "Nil maxDate returns day interval",
			minDate:          &now,
			maxDate:          nil,
			expectedInterval: "day",
			expectedFormat:   "strftime(started_at, '%Y-%m-%d')",
		},
		{
			name:             "Same day returns day interval",
			minDate:          &now,
			maxDate:          &now,
			expectedInterval: "day",
			expectedFormat:   "strftime(started_at, '%Y-%m-%d')",
		},
		{
			name:             "30 days apart returns day interval",
			minDate:          func() *time.Time { t := now.Add(-30 * 24 * time.Hour); return &t }(),
			maxDate:          &now,
			expectedInterval: "day",
			expectedFormat:   "strftime(started_at, '%Y-%m-%d')",
		},
		{
			name:             "90 days apart returns day interval",
			minDate:          func() *time.Time { t := now.Add(-90 * 24 * time.Hour); return &t }(),
			maxDate:          &now,
			expectedInterval: "day",
			expectedFormat:   "strftime(started_at, '%Y-%m-%d')",
		},
		{
			name:             "91 days apart returns week interval",
			minDate:          func() *time.Time { t := now.Add(-91 * 24 * time.Hour); return &t }(),
			maxDate:          &now,
			expectedInterval: "week",
			expectedFormat:   "strftime(started_at, '%Y-W%V')",
		},
		{
			name:             "180 days apart returns week interval",
			minDate:          func() *time.Time { t := now.Add(-180 * 24 * time.Hour); return &t }(),
			maxDate:          &now,
			expectedInterval: "week",
			expectedFormat:   "strftime(started_at, '%Y-W%V')",
		},
		{
			name:             "365 days apart returns week interval",
			minDate:          func() *time.Time { t := now.Add(-365 * 24 * time.Hour); return &t }(),
			maxDate:          &now,
			expectedInterval: "week",
			expectedFormat:   "strftime(started_at, '%Y-W%V')",
		},
		{
			name:             "366 days apart returns month interval",
			minDate:          func() *time.Time { t := now.Add(-366 * 24 * time.Hour); return &t }(),
			maxDate:          &now,
			expectedInterval: "month",
			expectedFormat:   "strftime(started_at, '%Y-%m')",
		},
		{
			name:             "730 days (2 years) apart returns month interval",
			minDate:          func() *time.Time { t := now.Add(-730 * 24 * time.Hour); return &t }(),
			maxDate:          &now,
			expectedInterval: "month",
			expectedFormat:   "strftime(started_at, '%Y-%m')",
		},
		{
			name:             "1825 days (5 years) apart returns month interval",
			minDate:          func() *time.Time { t := now.Add(-1825 * 24 * time.Hour); return &t }(),
			maxDate:          &now,
			expectedInterval: "month",
			expectedFormat:   "strftime(started_at, '%Y-%m')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interval, dateFormat := determineTrendsInterval(tt.minDate, tt.maxDate)

			if interval != tt.expectedInterval {
				t.Errorf("determineTrendsInterval() interval = %q, expected %q", interval, tt.expectedInterval)
			}

			if dateFormat != tt.expectedFormat {
				t.Errorf("determineTrendsInterval() dateFormat = %q, expected %q", dateFormat, tt.expectedFormat)
			}
		})
	}
}

// TestDetermineTrendsInterval_BoundaryConditions tests exact boundary conditions
func TestDetermineTrendsInterval_BoundaryConditions(t *testing.T) {

	now := time.Now()

	// Test exact boundary: 90 days should be "day", 90.1 days should be "week"
	tests := []struct {
		name     string
		daysDiff float64
		expected string
	}{
		{"Exactly 90 days", 90, "day"},
		{"Just over 90 days", 90.5, "week"},
		{"Exactly 365 days", 365, "week"},
		{"Just over 365 days", 365.5, "month"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minDate := now.Add(-time.Duration(tt.daysDiff*24) * time.Hour)
			interval, _ := determineTrendsInterval(&minDate, &now)

			if interval != tt.expected {
				t.Errorf("At %.1f days, expected interval %q, got %q", tt.daysDiff, tt.expected, interval)
			}
		})
	}
}

// TestGetPlaybackTrends_WeekInterval tests week interval determination
func TestGetPlaybackTrends_WeekInterval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	// Insert playbacks spanning > 90 days but <= 365 days to trigger week interval
	now := time.Now()
	testData := []struct {
		userID    int
		username  string
		ip        string
		mediaType string
		title     string
		startedAt time.Time
	}{
		{1, "user1", "192.168.1.1", "movie", "Movie 1", now.Add(-150 * 24 * time.Hour)},
		{1, "user1", "192.168.1.1", "movie", "Movie 2", now.Add(-120 * 24 * time.Hour)},
		{2, "user2", "192.168.1.2", "episode", "Show S01E01", now.Add(-100 * 24 * time.Hour)},
		{3, "user3", "192.168.1.3", "movie", "Movie 3", now.Add(-50 * 24 * time.Hour)},
		{4, "user4", "192.168.1.4", "episode", "Show S01E02", now.Add(-7 * 24 * time.Hour)},
		{5, "user5", "192.168.1.5", "movie", "Movie 4", now.Add(-1 * 24 * time.Hour)},
	}

	for _, pb := range testData {
		stoppedAt := pb.startedAt.Add(2 * time.Hour)
		_, err := db.conn.Exec(`
			INSERT INTO playback_events (
				id, session_key, started_at, stopped_at, user_id, username,
				ip_address, media_type, title, percent_complete
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), uuid.New().String(), pb.startedAt, stoppedAt,
			pb.userID, pb.username, pb.ip, pb.mediaType, pb.title, 100)
		if err != nil {
			t.Fatalf("Failed to insert playback: %v", err)
		}
	}

	startDate := now.Add(-150 * 24 * time.Hour) // 150 days to trigger week interval
	filter := LocationStatsFilter{
		StartDate: &startDate,
	}

	trends, interval, err := db.GetPlaybackTrends(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetPlaybackTrends failed: %v", err)
	}

	if interval != "week" {
		t.Errorf("Expected interval 'week', got '%s'", interval)
	}

	if len(trends) == 0 {
		t.Error("Expected trends data, got empty slice")
	}

	// Verify week format in dates (YYYY-W##)
	for _, trend := range trends {
		if len(trend.Date) > 0 && trend.Date[4] != '-' {
			t.Errorf("Expected date in week format, got: %s", trend.Date)
		}
	}
}

// TestGetPlaybackTrends_CanceledContext tests context cancellation
func TestGetPlaybackTrends_CanceledContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	insertTestPlaybacks(t, db)

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	now := time.Now()
	startDate := now.Add(-24 * time.Hour)
	filter := LocationStatsFilter{
		StartDate: &startDate,
	}

	_, _, err := db.GetPlaybackTrends(ctx, filter)
	if err == nil {
		t.Error("Expected error with canceled context")
	}
}

// TestGetPlaybackTrends_NilContext tests nil context handling
func TestGetPlaybackTrends_NilContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	insertTestPlaybacks(t, db)

	now := time.Now()
	startDate := now.Add(-24 * time.Hour)
	filter := LocationStatsFilter{
		StartDate: &startDate,
	}

	// nil context should be handled by ensureContext
	// Note: This tests the internal ensureContext fallback behavior
	trends, interval, err := db.GetPlaybackTrends(nil, filter)
	if err != nil {
		t.Logf("GetPlaybackTrends with nil context returned error (may be expected): %v", err)
	}

	// If no error, verify results
	if err == nil {
		if interval == "" {
			t.Error("Expected non-empty interval")
		}
		t.Logf("Got %d trends with interval %s", len(trends), interval)
	}
}

// TestGetViewingHoursHeatmap_WithMediaTypeFilter tests media type filtering for heatmap
func TestGetViewingHoursHeatmap_WithMediaTypeFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert different media types
	mediaTypes := []string{"movie", "movie", "episode", "track"}
	for i, mt := range mediaTypes {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i) * time.Hour),
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       mt,
			Title:           "Test Content",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	filter := LocationStatsFilter{
		MediaTypes: []string{"movie"},
	}

	heatmap, err := db.GetViewingHoursHeatmap(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetViewingHoursHeatmap failed: %v", err)
	}

	// Calculate total playback count
	totalCount := 0
	for _, entry := range heatmap {
		totalCount += entry.PlaybackCount
	}

	// Should have 2 movie playbacks only
	if totalCount != 2 {
		t.Errorf("Expected 2 playbacks (movies only), got %d", totalCount)
	}
}

// TestGetViewingHoursHeatmap_EmptyData tests heatmap with no matching data
func TestGetViewingHoursHeatmap_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	// No playbacks inserted

	filter := LocationStatsFilter{}

	heatmap, err := db.GetViewingHoursHeatmap(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetViewingHoursHeatmap failed: %v", err)
	}

	if len(heatmap) != 0 {
		t.Errorf("Expected empty heatmap, got %d entries", len(heatmap))
	}
}

// TestGetViewingHoursHeatmap_DateRangeFilter tests date range filtering for heatmap
func TestGetViewingHoursHeatmap_DateRangeFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	// Insert one event yesterday and one last week
	events := []struct {
		startedAt time.Time
	}{
		{yesterday},
		{lastWeek},
	}

	for i, e := range events {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       e.startedAt,
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Movie",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event %d: %v", i, err)
		}
	}

	// Filter to only include yesterday
	startDate := now.Add(-48 * time.Hour)
	endDate := now
	filter := LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	heatmap, err := db.GetViewingHoursHeatmap(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetViewingHoursHeatmap failed: %v", err)
	}

	// Count total playbacks in heatmap
	totalCount := 0
	for _, entry := range heatmap {
		totalCount += entry.PlaybackCount
	}

	// Should have 1 playback (yesterday only, not last week)
	if totalCount != 1 {
		t.Errorf("Expected 1 playback in date range, got %d", totalCount)
	}
}

// TestGetViewingHoursHeatmap_VerifyHourAndDayValues tests that hour/day values are valid
func TestGetViewingHoursHeatmap_VerifyHourAndDayValues(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	insertTestPlaybacks(t, db)

	now := time.Now()
	startDate := now.Add(-365 * 24 * time.Hour)
	filter := LocationStatsFilter{
		StartDate: &startDate,
	}

	heatmap, err := db.GetViewingHoursHeatmap(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetViewingHoursHeatmap failed: %v", err)
	}

	for _, entry := range heatmap {
		// Hour should be 0-23
		if entry.Hour < 0 || entry.Hour > 23 {
			t.Errorf("Invalid hour value: %d (expected 0-23)", entry.Hour)
		}

		// Day of week should be 0-6 (Sunday-Saturday)
		if entry.DayOfWeek < 0 || entry.DayOfWeek > 6 {
			t.Errorf("Invalid day of week value: %d (expected 0-6)", entry.DayOfWeek)
		}

		// Playback count should be positive
		if entry.PlaybackCount <= 0 {
			t.Errorf("Invalid playback count: %d (expected > 0)", entry.PlaybackCount)
		}
	}
}

// Sink variables to prevent dead code elimination in benchmarks.
// The compiler cannot optimize away assignments to package-level variables.
var (
	trendsClauseSink   string
	trendsArgsSink     []interface{}
	trendsIntervalSink string
	trendsDateFmtSink  string
)

// BenchmarkBuildTrendsWhereClause benchmarks the buildTrendsWhereClause function
func BenchmarkBuildTrendsWhereClause(b *testing.B) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	filters := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{"Empty", LocationStatsFilter{}},
		{"DateOnly", LocationStatsFilter{StartDate: &yesterday, EndDate: &now}},
		{"WithUsers", LocationStatsFilter{
			StartDate: &yesterday,
			EndDate:   &now,
			Users:     []string{"user1", "user2", "user3"},
		}},
		{"FullFilter", LocationStatsFilter{
			StartDate:  &yesterday,
			EndDate:    &now,
			Users:      []string{"user1", "user2", "user3", "user4", "user5"},
			MediaTypes: []string{"movie", "episode", "track"},
		}},
	}

	for _, f := range filters {
		b.Run(f.name, func(b *testing.B) {
			var clause string
			var args []interface{}
			for i := 0; i < b.N; i++ {
				clause, args = buildTrendsWhereClause(f.filter)
			}
			// Prevent dead code elimination
			trendsClauseSink = clause
			trendsArgsSink = args
		})
	}
}

// BenchmarkDetermineTrendsInterval benchmarks the determineTrendsInterval function
func BenchmarkDetermineTrendsInterval(b *testing.B) {
	now := time.Now()

	testCases := []struct {
		name    string
		minDate *time.Time
		maxDate *time.Time
	}{
		{"NilDates", nil, nil},
		{"SameDay", &now, &now},
		{"30Days", func() *time.Time { t := now.Add(-30 * 24 * time.Hour); return &t }(), &now},
		{"180Days", func() *time.Time { t := now.Add(-180 * 24 * time.Hour); return &t }(), &now},
		{"730Days", func() *time.Time { t := now.Add(-730 * 24 * time.Hour); return &t }(), &now},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			var interval, dateFormat string
			for i := 0; i < b.N; i++ {
				interval, dateFormat = determineTrendsInterval(tc.minDate, tc.maxDate)
			}
			// Prevent dead code elimination
			trendsIntervalSink = interval
			trendsDateFmtSink = dateFormat
		})
	}
}
