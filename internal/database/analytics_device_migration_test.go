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

// TestGetDeviceMigrationAnalytics tests the main GetDeviceMigrationAnalytics function
func TestGetDeviceMigrationAnalytics(t *testing.T) {
	db := testDBWithData(t, insertDeviceMigrationTestData)
	defer db.Close()

	result, err := db.GetDeviceMigrationAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetDeviceMigrationAnalytics() error = %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Validate summary structure
	assertNonNegative(t, "TotalUsers", result.Summary.TotalUsers)
	assertNonNegative(t, "MultiDeviceUsers", result.Summary.MultiDeviceUsers)
	assertNonNegative(t, "TotalMigrations", result.Summary.TotalMigrations)
	assertNonNegativeFloat(t, "AvgPlatformsPerUser", result.Summary.AvgPlatformsPerUser)

	// Validate metadata
	if result.Metadata.ExecutionTimeMS < 0 {
		t.Error("ExecutionTimeMS should not be negative")
	}
	if result.Metadata.QueryHash == "" {
		t.Error("QueryHash should not be empty")
	}

	// Validate that we have data from our test setup
	if result.Summary.TotalUsers == 0 {
		t.Error("Expected TotalUsers > 0 with test data")
	}

	// Validate components are populated
	if result.TopUserProfiles == nil {
		t.Error("TopUserProfiles should be initialized")
	}
	if result.AdoptionTrends == nil {
		t.Error("AdoptionTrends should be initialized")
	}
	if result.CommonTransitions == nil {
		t.Error("CommonTransitions should be initialized")
	}
	if result.PlatformDistribution == nil {
		t.Error("PlatformDistribution should be initialized")
	}
}

// TestGetDeviceMigrationSummary tests the getDeviceMigrationSummary function
func TestGetDeviceMigrationSummary(t *testing.T) {
	db := testDBWithData(t, insertDeviceMigrationTestData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{"empty filter", LocationStatsFilter{}},
		{"with date filter", LocationStatsFilter{StartDate: timePtrOffset(-14 * 24 * time.Hour)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, err := db.getDeviceMigrationSummary(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("getDeviceMigrationSummary() error = %v", err)
			}
			if summary == nil {
				t.Fatal("Expected non-nil summary")
			}

			assertNonNegative(t, "TotalUsers", summary.TotalUsers)
			assertNonNegative(t, "MultiDeviceUsers", summary.MultiDeviceUsers)
			assertNonNegative(t, "TotalMigrations", summary.TotalMigrations)
			assertNonNegativeFloat(t, "AvgPlatformsPerUser", summary.AvgPlatformsPerUser)

			// MultiDevicePercentage should be 0-100
			if summary.MultiDevicePercentage < 0 || summary.MultiDevicePercentage > 100 {
				t.Errorf("MultiDevicePercentage should be 0-100, got %f", summary.MultiDevicePercentage)
			}
		})
	}
}

// TestGetTopUserDeviceProfiles tests the getTopUserDeviceProfiles function
func TestGetTopUserDeviceProfiles(t *testing.T) {
	db := testDBWithData(t, insertDeviceMigrationTestData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
		limit  int
	}{
		{"default limit", LocationStatsFilter{}, 10},
		{"small limit", LocationStatsFilter{}, 5},
		{"zero limit", LocationStatsFilter{}, 0},
		{"large limit", LocationStatsFilter{}, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profiles, err := db.getTopUserDeviceProfiles(context.Background(), tt.filter, tt.limit)
			if err != nil {
				t.Fatalf("getTopUserDeviceProfiles() error = %v", err)
			}

			// Validate each profile
			for _, profile := range profiles {
				if profile.UserID == 0 && profile.Username == "" {
					t.Error("Profile should have UserID or Username")
				}
				assertNonNegative(t, "TotalPlatformsUsed", profile.TotalPlatformsUsed)
				assertNonNegative(t, "TotalSessions", profile.TotalSessions)
				assertNonNegative(t, "TotalMigrations", profile.TotalMigrations)
				assertNonNegative(t, "DaysSinceFirstSeen", profile.DaysSinceFirstSeen)
				assertNonNegative(t, "DaysSinceLastSeen", profile.DaysSinceLastSeen)

				// IsMultiDevice should be consistent
				if profile.IsMultiDevice && profile.TotalPlatformsUsed < 2 {
					t.Error("IsMultiDevice should require at least 2 platforms")
				}

				// PrimaryPlatformPercentage should be 0-100
				if profile.PrimaryPlatformPercentage < 0 || profile.PrimaryPlatformPercentage > 100 {
					t.Errorf("PrimaryPlatformPercentage should be 0-100, got %f", profile.PrimaryPlatformPercentage)
				}
			}
		})
	}
}

// TestGetUserPlatformHistory tests the getUserPlatformHistory function
func TestGetUserPlatformHistory(t *testing.T) {
	db := testDBWithData(t, insertDeviceMigrationTestData)
	defer db.Close()

	// Test with a user that has multiple platforms
	history, err := db.getUserPlatformHistory(context.Background(), LocationStatsFilter{}, 1)
	if err != nil {
		t.Fatalf("getUserPlatformHistory() error = %v", err)
	}

	// Validate history structure
	isPrimarySet := false
	for _, usage := range history {
		assertNotEmpty(t, "Platform", usage.Platform)
		assertNonNegative(t, "SessionCount", usage.SessionCount)
		assertNonNegativeFloat(t, "TotalWatchTimeMinutes", usage.TotalWatchTimeMinutes)

		if usage.Percentage < 0 || usage.Percentage > 100 {
			t.Errorf("Percentage should be 0-100, got %f", usage.Percentage)
		}

		// Only one platform should be marked as primary
		if usage.IsPrimary {
			if isPrimarySet {
				t.Error("Multiple platforms marked as primary")
			}
			isPrimarySet = true
		}
	}
}

// TestCountUserMigrations tests the countUserMigrations function
func TestCountUserMigrations(t *testing.T) {
	db := testDBWithData(t, insertDeviceMigrationTestData)
	defer db.Close()

	tests := []struct {
		name   string
		userID int
	}{
		{"user with migrations", 1},
		{"user without migrations", 5},
		{"non-existent user", 999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := db.countUserMigrations(context.Background(), LocationStatsFilter{}, tt.userID)
			if err != nil {
				t.Fatalf("countUserMigrations() error = %v", err)
			}
			assertNonNegative(t, "migration count", count)
		})
	}
}

// TestGetRecentMigrations tests the getRecentMigrations function
func TestGetRecentMigrations(t *testing.T) {
	db := testDBWithData(t, insertDeviceMigrationTestData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
		limit  int
	}{
		{"default limit", LocationStatsFilter{}, 50},
		{"small limit", LocationStatsFilter{}, 5},
		{"zero limit", LocationStatsFilter{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrations, err := db.getRecentMigrations(context.Background(), tt.filter, tt.limit)
			if err != nil {
				t.Fatalf("getRecentMigrations() error = %v", err)
			}

			// Validate each migration
			for _, m := range migrations {
				assertNotEmpty(t, "FromPlatform", m.FromPlatform)
				assertNotEmpty(t, "ToPlatform", m.ToPlatform)

				if m.FromPlatform == m.ToPlatform {
					t.Error("FromPlatform and ToPlatform should be different")
				}

				assertNonNegative(t, "SessionsBeforeMigration", m.SessionsBeforeMigration)
				assertNonNegative(t, "SessionsAfterMigration", m.SessionsAfterMigration)
			}
		})
	}
}

// TestGetPlatformAdoptionTrends tests the getPlatformAdoptionTrends function
func TestGetPlatformAdoptionTrends(t *testing.T) {
	db := testDBWithData(t, insertDeviceMigrationTestData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{"empty filter", LocationStatsFilter{}},
		{"with date range", LocationStatsFilter{
			StartDate: timePtrOffset(-30 * 24 * time.Hour),
			EndDate:   timePtrOffset(0),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trends, err := db.getPlatformAdoptionTrends(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("getPlatformAdoptionTrends() error = %v", err)
			}

			for _, trend := range trends {
				assertNotEmpty(t, "Date", trend.Date)
				assertNotEmpty(t, "Platform", trend.Platform)
				assertNonNegative(t, "NewUsers", trend.NewUsers)
				assertNonNegative(t, "ActiveUsers", trend.ActiveUsers)
				assertNonNegative(t, "SessionCount", trend.SessionCount)

				if trend.MarketShare < 0 || trend.MarketShare > 100 {
					t.Errorf("MarketShare should be 0-100, got %f", trend.MarketShare)
				}
			}
		})
	}
}

// TestGetCommonPlatformTransitions tests the getCommonPlatformTransitions function
func TestGetCommonPlatformTransitions(t *testing.T) {
	db := testDBWithData(t, insertDeviceMigrationTestData)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
		limit  int
	}{
		{"default limit", LocationStatsFilter{}, 10},
		{"small limit", LocationStatsFilter{}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transitions, err := db.getCommonPlatformTransitions(context.Background(), tt.filter, tt.limit)
			if err != nil {
				t.Fatalf("getCommonPlatformTransitions() error = %v", err)
			}

			for _, trans := range transitions {
				assertNotEmpty(t, "FromPlatform", trans.FromPlatform)
				assertNotEmpty(t, "ToPlatform", trans.ToPlatform)
				assertNonNegative(t, "TransitionCount", trans.TransitionCount)
				assertNonNegative(t, "UniqueUsers", trans.UniqueUsers)
				assertNonNegativeFloat(t, "AvgDaysBeforeSwitch", trans.AvgDaysBeforeSwitch)

				if trans.ReturnRate < 0 || trans.ReturnRate > 100 {
					t.Errorf("ReturnRate should be 0-100, got %f", trans.ReturnRate)
				}
			}
		})
	}
}

// TestDetermineTrendInterval tests the determineTrendInterval function
func TestDetermineTrendInterval(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		filter   LocationStatsFilter
		expected string
	}{
		{
			name:     "nil dates defaults to week",
			filter:   LocationStatsFilter{},
			expected: "week",
		},
		{
			name: "7 days or less returns day",
			filter: LocationStatsFilter{
				StartDate: timePtr(now.Add(-5 * 24 * time.Hour)),
				EndDate:   timePtr(now),
			},
			expected: "day",
		},
		{
			name: "8-60 days returns week",
			filter: LocationStatsFilter{
				StartDate: timePtr(now.Add(-30 * 24 * time.Hour)),
				EndDate:   timePtr(now),
			},
			expected: "week",
		},
		{
			name: "61-365 days returns month",
			filter: LocationStatsFilter{
				StartDate: timePtr(now.Add(-180 * 24 * time.Hour)),
				EndDate:   timePtr(now),
			},
			expected: "month",
		},
		{
			name: "more than 365 days returns quarter",
			filter: LocationStatsFilter{
				StartDate: timePtr(now.Add(-400 * 24 * time.Hour)),
				EndDate:   timePtr(now),
			},
			expected: "quarter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interval := determineTrendInterval(tt.filter)
			if interval != tt.expected {
				t.Errorf("determineTrendInterval() = %s, want %s", interval, tt.expected)
			}
		})
	}
}

// TestBuildDeviceMigrationMetadata tests the buildDeviceMigrationMetadata function
func TestBuildDeviceMigrationMetadata(t *testing.T) {
	db := testDBWithData(t, insertDeviceMigrationTestData)
	defer db.Close()

	startTime := time.Now()
	filter := LocationStatsFilter{}

	metadata := db.buildDeviceMigrationMetadata(context.Background(), filter, startTime)

	if metadata.ExecutionTimeMS < 0 {
		t.Error("ExecutionTimeMS should not be negative")
	}
	if metadata.MigrationWindowDays != migrationWindowDays {
		t.Errorf("MigrationWindowDays should be %d, got %d", migrationWindowDays, metadata.MigrationWindowDays)
	}
	if metadata.QueryHash == "" {
		t.Error("QueryHash should not be empty")
	}
	assertNonNegative(t, "TotalEventsAnalyzed", metadata.TotalEventsAnalyzed)
	assertNonNegative(t, "UniquePlatformsFound", metadata.UniquePlatformsFound)
}

// TestContextCancellationDeviceMigration tests context cancellation handling
func TestContextCancellationDeviceMigration(t *testing.T) {
	db := testDBWithData(t, insertDeviceMigrationTestData)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := db.GetDeviceMigrationAnalytics(ctx, LocationStatsFilter{})
	// Context cancellation may or may not cause an error depending on timing
	// This test ensures no panic
	_ = err
}

// insertDeviceMigrationTestData inserts test data for device migration tests
func insertDeviceMigrationTestData(t *testing.T, db *DB) {
	now := time.Now()

	// User 1: Multiple platform user with migrations
	platformSequence1 := []struct {
		platform   string
		offsetDays int
	}{
		{"Roku", 30},
		{"Roku", 28},
		{"AppleTV", 25}, // Migration from Roku to AppleTV
		{"AppleTV", 23},
		{"AppleTV", 20},
		{"iOS", 15}, // Migration to iOS
		{"iOS", 10},
		{"AppleTV", 5}, // Back to AppleTV
		{"AppleTV", 2},
	}

	for i, ps := range platformSequence1 {
		insertPlaybackWithPlatform(t, db, 1, "user1", ps.platform, now.Add(-time.Duration(ps.offsetDays)*24*time.Hour), i)
	}

	// User 2: Multi-platform user
	platforms2 := []struct {
		platform   string
		offsetDays int
	}{
		{"Android", 20},
		{"Web", 18},
		{"Android", 15},
		{"Chromecast", 10},
		{"Web", 5},
	}

	for i, ps := range platforms2 {
		insertPlaybackWithPlatform(t, db, 2, "user2", ps.platform, now.Add(-time.Duration(ps.offsetDays)*24*time.Hour), 100+i)
	}

	// User 3: Single platform user
	for i := 0; i < 5; i++ {
		insertPlaybackWithPlatform(t, db, 3, "user3", "Roku", now.Add(-time.Duration(i*2)*24*time.Hour), 200+i)
	}

	// User 4: Recent adopter
	platforms4 := []struct {
		platform   string
		offsetDays int
	}{
		{"iOS", 5},
		{"iOS", 4},
		{"iOS", 3},
	}

	for i, ps := range platforms4 {
		insertPlaybackWithPlatform(t, db, 4, "user4", ps.platform, now.Add(-time.Duration(ps.offsetDays)*24*time.Hour), 300+i)
	}

	// User 5: No migrations, single platform
	insertPlaybackWithPlatform(t, db, 5, "user5", "Web", now.Add(-1*24*time.Hour), 400)
}

// insertPlaybackWithPlatform inserts a playback event with specified platform
func insertPlaybackWithPlatform(t *testing.T, db *DB, userID int, username, platform string, startedAt time.Time, idx int) {
	t.Helper()

	_, err := db.conn.Exec(`
		INSERT INTO playback_events (
			id, session_key, started_at, stopped_at, user_id, username,
			ip_address, media_type, title, percent_complete, play_duration, platform
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.New().String(), uuid.New().String(), startedAt,
		startedAt.Add(60*time.Minute), userID, username, "192.168.1.1",
		"movie", "Test Movie "+string(rune('A'+idx%26)), 100, 60, platform)
	if err != nil {
		t.Fatalf("Failed to insert playback with platform: %v", err)
	}
}

// TestEmptyDatabaseDeviceMigration tests behavior with no data
func TestEmptyDatabaseDeviceMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	result, err := db.GetDeviceMigrationAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetDeviceMigrationAnalytics() with empty db error = %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result even with empty database")
	}

	// Summary should have zeros
	if result.Summary.TotalUsers != 0 {
		t.Errorf("Expected TotalUsers=0, got %d", result.Summary.TotalUsers)
	}
	if result.Summary.TotalMigrations != 0 {
		t.Errorf("Expected TotalMigrations=0, got %d", result.Summary.TotalMigrations)
	}
}
