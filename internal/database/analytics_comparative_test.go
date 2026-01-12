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

// --- Test Helpers ---

// testDBWithData sets up test database with geolocations and returns cleanup function
func testDBWithData(t *testing.T, insertData func(*testing.T, *DB)) *DB {
	db := setupTestDB(t)
	insertTestGeolocations(t, db)
	if insertData != nil {
		insertData(t, db)
	}
	return db
}

// timePtrOffset returns a time pointer offset from now
func timePtrOffset(d time.Duration) *time.Time {
	t := time.Now().Add(d)
	return &t
}

// assertNonNegative checks that a value is not negative
func assertNonNegative(t *testing.T, name string, value int) {
	t.Helper()
	if value < 0 {
		t.Errorf("%s should not be negative, got %d", name, value)
	}
}

// assertNonNegativeFloat checks that a float value is not negative
func assertNonNegativeFloat(t *testing.T, name string, value float64) {
	t.Helper()
	if value < 0 {
		t.Errorf("%s should not be negative, got %f", name, value)
	}
}

// assertNotEmpty checks that a string is not empty
func assertNotEmpty(t *testing.T, name, value string) {
	t.Helper()
	if value == "" {
		t.Errorf("%s should not be empty", name)
	}
}

// assertComparisonType validates comparison type matches expected
func assertComparisonType(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("Expected comparison type '%s', got '%s'", want, got)
	}
}

// TestGetComparativeAnalytics tests the GetComparativeAnalytics function
func TestGetComparativeAnalytics(t *testing.T) {
	db := testDBWithData(t, insertComparativeTestPlaybacks)
	defer db.Close()

	// Test cases: input type -> expected output type (defaults to week for invalid/empty)
	tests := []struct {
		name       string
		filter     LocationStatsFilter
		inputType  string
		expectType string
		checkData  bool // whether to check for metrics data
	}{
		{"week comparison", LocationStatsFilter{}, "week", "week", true},
		{"month comparison", LocationStatsFilter{}, "month", "month", false},
		{"quarter comparison", LocationStatsFilter{}, "quarter", "quarter", false},
		{"year comparison", LocationStatsFilter{}, "year", "year", false},
		{"custom with dates", LocationStatsFilter{StartDate: timePtrOffset(-14 * 24 * time.Hour), EndDate: timePtrOffset(0)}, "custom", "custom", false},
		{"custom without dates", LocationStatsFilter{}, "custom", "custom", false},
		{"invalid type defaults to week", LocationStatsFilter{}, "invalid", "week", false},
		{"empty type defaults to week", LocationStatsFilter{}, "", "week", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := db.GetComparativeAnalytics(context.Background(), tt.filter, tt.inputType)
			if err != nil {
				t.Fatalf("GetComparativeAnalytics() error = %v", err)
			}
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			assertComparisonType(t, result.ComparisonType, tt.expectType)
			if tt.checkData && len(result.MetricsComparison) == 0 {
				t.Error("Expected metrics comparison data")
			}
		})
	}
}

// TestGetPeriodMetrics tests the getPeriodMetrics function
func TestGetPeriodMetrics(t *testing.T) {
	db := testDBWithData(t, insertComparativeTestPlaybacks)
	defer db.Close()

	now := time.Now()
	startDate, endDate := now.Add(-30*24*time.Hour), now

	metrics, err := db.getPeriodMetrics(context.Background(), startDate, endDate, LocationStatsFilter{})
	if err != nil {
		t.Fatalf("getPeriodMetrics failed: %v", err)
	}
	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Verify period boundaries and non-negative values
	if !metrics.StartDate.Equal(startDate) || !metrics.EndDate.Equal(endDate) {
		t.Errorf("Period boundaries mismatch: got %v-%v, want %v-%v", metrics.StartDate, metrics.EndDate, startDate, endDate)
	}
	assertNonNegative(t, "PlaybackCount", metrics.PlaybackCount)
	assertNonNegative(t, "UniqueUsers", metrics.UniqueUsers)
	assertNonNegativeFloat(t, "WatchTimeMinutes", metrics.WatchTimeMinutes)
}

// TestCompareMetric tests the compareMetric helper function
func TestCompareMetric(t *testing.T) {

	tests := []struct {
		name            string
		metricName      string
		current         float64
		previous        float64
		higherIsBetter  bool
		expectDirection string
		expectImprove   bool
	}{
		{
			name:            "increase when higher is better",
			metricName:      "Playbacks",
			current:         100,
			previous:        80,
			higherIsBetter:  true,
			expectDirection: "up",
			expectImprove:   true,
		},
		{
			name:            "decrease when higher is better",
			metricName:      "Playbacks",
			current:         60,
			previous:        80,
			higherIsBetter:  true,
			expectDirection: "down",
			expectImprove:   false,
		},
		{
			name:            "increase when lower is better",
			metricName:      "Errors",
			current:         100,
			previous:        80,
			higherIsBetter:  false,
			expectDirection: "up",
			expectImprove:   false,
		},
		{
			name:            "decrease when lower is better",
			metricName:      "Errors",
			current:         60,
			previous:        80,
			higherIsBetter:  false,
			expectDirection: "down",
			expectImprove:   true,
		},
		{
			name:            "stable values",
			metricName:      "Constant",
			current:         100,
			previous:        100,
			higherIsBetter:  true,
			expectDirection: "stable",
			expectImprove:   false,
		},
		{
			name:            "from zero",
			metricName:      "New Metric",
			current:         50,
			previous:        0,
			higherIsBetter:  true,
			expectDirection: "up",
			expectImprove:   true,
		},
		{
			name:            "to zero",
			metricName:      "Declining",
			current:         0,
			previous:        50,
			higherIsBetter:  true,
			expectDirection: "down",
			expectImprove:   false,
		},
		{
			name:            "small change within threshold",
			metricName:      "Small",
			current:         100.005,
			previous:        100,
			higherIsBetter:  true,
			expectDirection: "stable",
			expectImprove:   true, // technically > 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareMetric(tt.metricName, tt.current, tt.previous, tt.higherIsBetter)

			if result.Metric != tt.metricName {
				t.Errorf("Expected metric name '%s', got '%s'", tt.metricName, result.Metric)
			}
			if result.CurrentValue != tt.current {
				t.Errorf("Expected current value %f, got %f", tt.current, result.CurrentValue)
			}
			if result.PreviousValue != tt.previous {
				t.Errorf("Expected previous value %f, got %f", tt.previous, result.PreviousValue)
			}
			if result.GrowthDirection != tt.expectDirection {
				t.Errorf("Expected direction '%s', got '%s'", tt.expectDirection, result.GrowthDirection)
			}
			if result.IsImprovement != tt.expectImprove {
				t.Errorf("Expected improvement %v, got %v", tt.expectImprove, result.IsImprovement)
			}
		})
	}
}

// TestGenerateKeyInsights tests the generateKeyInsights function
func TestGenerateKeyInsights(t *testing.T) {

	// Helper to create metrics with given values
	makeMetrics := func(playbacks, users int, watchTime, sessionMins, completion float64, content int) *models.PeriodMetrics {
		return &models.PeriodMetrics{
			PlaybackCount: playbacks, UniqueUsers: users, WatchTimeMinutes: watchTime,
			AvgSessionMins: sessionMins, AvgCompletion: completion, UniqueContent: content,
		}
	}

	tests := []struct {
		name     string
		current  *models.PeriodMetrics
		previous *models.PeriodMetrics
	}{
		{"growing activity", makeMetrics(120, 15, 2000, 50, 85, 50), makeMetrics(100, 10, 1500, 40, 75, 30)},
		{"declining activity", makeMetrics(80, 8, 1000, 30, 60, 20), makeMetrics(100, 10, 1500, 40, 75, 30)},
		{"stable activity", makeMetrics(100, 10, 1500, 40, 75, 30), makeMetrics(100, 10, 1500, 40, 75, 30)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := []models.ComparativeMetrics{
				compareMetric("Playbacks", float64(tt.current.PlaybackCount), float64(tt.previous.PlaybackCount), true),
			}
			insights := generateKeyInsights(tt.current, tt.previous, metrics)
			if len(insights) == 0 {
				t.Error("Expected at least one insight")
			}
		})
	}
}

// TestGetTopComparisons tests both content and user comparison functions
func TestGetTopComparisons(t *testing.T) {
	db := testDBWithData(t, insertComparativeTestPlaybacks)
	defer db.Close()

	now := time.Now()
	currentStart, currentEnd := now.Add(-7*24*time.Hour), now
	previousStart, previousEnd := currentStart.Add(-7*24*time.Hour), currentStart
	ctx := context.Background()

	validTrending := map[string]bool{"new": true, "rising": true, "falling": true, "stable": true}

	t.Run("content comparison", func(t *testing.T) {
		comparison, err := db.getTopContentComparison(ctx, currentStart, currentEnd, previousStart, previousEnd, LocationStatsFilter{}, 10)
		if err != nil {
			t.Fatalf("getTopContentComparison failed: %v", err)
		}
		for _, item := range comparison {
			assertNotEmpty(t, "Title", item.Title)
			if item.CurrentRank < 1 {
				t.Error("Current rank should be at least 1")
			}
			if !validTrending[item.Trending] {
				t.Errorf("Invalid trending value: %s", item.Trending)
			}
		}
	})

	t.Run("user comparison", func(t *testing.T) {
		comparison, err := db.getTopUserComparison(ctx, currentStart, currentEnd, previousStart, previousEnd, LocationStatsFilter{}, 10)
		if err != nil {
			t.Fatalf("getTopUserComparison failed: %v", err)
		}
		for _, item := range comparison {
			assertNotEmpty(t, "Username", item.Username)
			if item.CurrentRank < 1 {
				t.Error("Current rank should be at least 1")
			}
		}
	})
}

// TestGetTopForPeriod tests both content and user period queries
func TestGetTopForPeriod(t *testing.T) {
	db := testDBWithData(t, insertComparativeTestPlaybacks)
	defer db.Close()

	now := time.Now()
	startDate, endDate := now.Add(-30*24*time.Hour), now
	ctx := context.Background()

	t.Run("content for period", func(t *testing.T) {
		content, err := db.getTopContentForPeriod(ctx, startDate, endDate, LocationStatsFilter{}, 10)
		if err != nil {
			t.Fatalf("getTopContentForPeriod failed: %v", err)
		}
		for _, item := range content {
			assertNotEmpty(t, "Title", item.Title)
			if item.Count < 1 {
				t.Error("Count should be at least 1")
			}
		}
	})

	t.Run("users for period", func(t *testing.T) {
		users, err := db.getTopUsersForPeriod(ctx, startDate, endDate, LocationStatsFilter{}, 10)
		if err != nil {
			t.Fatalf("getTopUsersForPeriod failed: %v", err)
		}
		for _, user := range users {
			assertNotEmpty(t, "Username", user.Username)
			assertNonNegativeFloat(t, "WatchTime", user.WatchTime)
		}
	})
}

// TestGetContentAbandonmentAnalytics tests the GetContentAbandonmentAnalytics function
func TestGetContentAbandonmentAnalytics(t *testing.T) {
	db := testDBWithData(t, insertAbandonmentTestPlaybacks)
	defer db.Close()

	tests := []struct {
		name              string
		filter            LocationStatsFilter
		checkDropOff      bool
		expectEmptyGenre  bool
		expectEmptyEpisod bool
	}{
		{"basic abandonment query", LocationStatsFilter{}, true, false, false},
		{"with date filter", LocationStatsFilter{StartDate: timePtrOffset(-30 * 24 * time.Hour)}, false, false, false},
		{"movie media type filter", LocationStatsFilter{MediaTypes: []string{"movie"}}, false, false, false},
		{"episode media type filter", LocationStatsFilter{MediaTypes: []string{"episode"}}, false, false, false},
		{"track media type skips genre/episode", LocationStatsFilter{MediaTypes: []string{"track"}}, false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := db.GetContentAbandonmentAnalytics(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("GetContentAbandonmentAnalytics() error = %v", err)
			}
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			assertNonNegative(t, "TotalPlaybacks", result.Summary.TotalPlaybacks)
			if tt.checkDropOff && result.DropOffDistribution == nil {
				t.Error("DropOffDistribution should be initialized")
			}
			if tt.expectEmptyGenre && len(result.AbandonmentByGenre) > 0 {
				t.Error("Expected empty genre analysis for tracks")
			}
			if tt.expectEmptyEpisod && len(result.FirstEpisodeAbandonment) > 0 {
				t.Error("Expected empty first episode analysis for tracks")
			}
		})
	}
}

// TestAbandonmentHelpers tests abandonment helper methods
func TestAbandonmentHelpers(t *testing.T) {
	db := testDBWithData(t, insertAbandonmentTestPlaybacks)
	defer db.Close()

	whereClause, args := buildAbandonmentWhereClause(LocationStatsFilter{})
	ctx := context.Background()

	// Test getAbandonmentSummary
	summary, err := db.getAbandonmentSummary(ctx, whereClause, args)
	if err != nil {
		t.Fatalf("getAbandonmentSummary failed: %v", err)
	}
	assertNonNegative(t, "TotalPlaybacks", summary.TotalPlaybacks)
	if summary.CompletionRate < 0 || summary.CompletionRate > 100 {
		t.Errorf("CompletionRate should be 0-100, got %f", summary.CompletionRate)
	}

	// Test getTopAbandonedContent
	abandoned, err := db.getTopAbandonedContent(ctx, whereClause, args)
	if err != nil {
		t.Fatalf("getTopAbandonedContent failed: %v", err)
	}
	for _, item := range abandoned {
		assertNotEmpty(t, "Title", item.Title)
	}

	// Test getCompletionByMediaType
	byType, err := db.getCompletionByMediaType(ctx, whereClause, args)
	if err != nil {
		t.Fatalf("getCompletionByMediaType failed: %v", err)
	}
	for _, item := range byType {
		assertNotEmpty(t, "MediaType", item.MediaType)
	}

	// Test getDropOffDistribution
	dist, err := db.getDropOffDistribution(ctx, whereClause, args, 100)
	if err != nil {
		t.Fatalf("getDropOffDistribution failed: %v", err)
	}
	for _, bucket := range dist {
		assertNotEmpty(t, "Bucket", bucket.Bucket)
		if bucket.MinPercent < 0 || bucket.MaxPercent > 100 {
			t.Error("Bucket percentages out of range")
		}
	}
}

// TestContainsHelper tests the contains helper function
func TestContainsHelper(t *testing.T) {

	tests := []struct {
		name   string
		slice  []string
		value  string
		expect bool
	}{
		{"value present", []string{"a", "b", "c"}, "b", true},
		{"value absent", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"nil slice", nil, "a", false},
		{"first element", []string{"a", "b", "c"}, "a", true},
		{"last element", []string{"a", "b", "c"}, "c", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.slice, tt.value); got != tt.expect {
				t.Errorf("contains(%v, %s) = %v, want %v", tt.slice, tt.value, got, tt.expect)
			}
		})
	}
}

// insertComparativeTestPlaybacks inserts test data spanning multiple periods
func insertComparativeTestPlaybacks(t *testing.T, db *DB) {
	now := time.Now()
	periods := []struct {
		offset time.Duration
		count  int
	}{
		{-1 * 24 * time.Hour, 5},   // Yesterday
		{-7 * 24 * time.Hour, 3},   // 1 week ago
		{-14 * 24 * time.Hour, 4},  // 2 weeks ago
		{-30 * 24 * time.Hour, 2},  // 1 month ago
		{-60 * 24 * time.Hour, 2},  // 2 months ago
		{-90 * 24 * time.Hour, 1},  // 3 months ago
		{-365 * 24 * time.Hour, 1}, // 1 year ago
	}

	playbackID := 0
	for _, period := range periods {
		for i := 0; i < period.count; i++ {
			playbackID++
			userID := (playbackID % 5) + 1
			username := []string{"user1", "user2", "user3", "user4", "user5"}[userID-1]
			ip := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4", "192.168.1.5"}[userID-1]
			startedAt := now.Add(period.offset).Add(time.Duration(i) * time.Hour)

			_, err := db.conn.Exec(`
				INSERT INTO playback_events (
					id, session_key, started_at, stopped_at, user_id, username,
					ip_address, media_type, title, percent_complete, play_duration
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, uuid.New().String(), uuid.New().String(), startedAt,
				startedAt.Add(90*time.Minute), userID, username, ip,
				"movie", "Comparative Movie "+string(rune('A'+playbackID%10)), 85, 90)
			if err != nil {
				t.Fatalf("Failed to insert comparative playback: %v", err)
			}
		}
	}
}

// TestDetermineOverallTrend tests the determineOverallTrend helper function
func TestDetermineOverallTrend(t *testing.T) {

	tests := []struct {
		name          string
		currentCount  int
		previousCount int
		expectedTrend string
	}{
		{
			name:          "growing - current 20% higher than previous",
			currentCount:  120,
			previousCount: 100,
			expectedTrend: "growing",
		},
		{
			name:          "growing - current exactly 10% higher (boundary)",
			currentCount:  111,
			previousCount: 100,
			expectedTrend: "growing",
		},
		{
			name:          "declining - current 20% lower than previous",
			currentCount:  80,
			previousCount: 100,
			expectedTrend: "declining",
		},
		{
			name:          "declining - current exactly 10% lower (boundary)",
			currentCount:  89,
			previousCount: 100,
			expectedTrend: "declining",
		},
		{
			name:          "stable - current equal to previous",
			currentCount:  100,
			previousCount: 100,
			expectedTrend: "stable",
		},
		{
			name:          "stable - current slightly higher (within 10%)",
			currentCount:  109,
			previousCount: 100,
			expectedTrend: "stable",
		},
		{
			name:          "stable - current slightly lower (within 10%)",
			currentCount:  91,
			previousCount: 100,
			expectedTrend: "stable",
		},
		{
			name:          "growing from zero previous",
			currentCount:  10,
			previousCount: 0,
			expectedTrend: "growing",
		},
		{
			name:          "stable when both zero",
			currentCount:  0,
			previousCount: 0,
			expectedTrend: "stable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := &models.PeriodMetrics{PlaybackCount: tt.currentCount}
			previous := &models.PeriodMetrics{PlaybackCount: tt.previousCount}

			trend := determineOverallTrend(current, previous)
			if trend != tt.expectedTrend {
				t.Errorf("determineOverallTrend() = %s, want %s", trend, tt.expectedTrend)
			}
		})
	}
}

// insertAbandonmentTestPlaybacks inserts test data for abandonment analysis
func insertAbandonmentTestPlaybacks(t *testing.T, db *DB) {
	now := time.Now()

	playbacks := []struct {
		userID           int
		username         string
		ip               string
		mediaType        string
		title            string
		grandparentTitle string
		parentTitle      string
		mediaIndex       int
		parentMediaIndex int
		percentComplete  int
		genres           string
	}{
		// Completed content (>= 90%)
		{1, "user1", "192.168.1.1", "movie", "Complete Movie 1", "", "", 0, 0, 100, "Action, Drama"},
		{1, "user1", "192.168.1.1", "movie", "Complete Movie 1", "", "", 0, 0, 95, "Action, Drama"},
		{1, "user1", "192.168.1.1", "movie", "Complete Movie 1", "", "", 0, 0, 92, "Action, Drama"},

		// Abandoned content (< 90%)
		{2, "user2", "192.168.1.2", "movie", "Abandoned Movie 1", "", "", 0, 0, 25, "Horror"},
		{2, "user2", "192.168.1.2", "movie", "Abandoned Movie 1", "", "", 0, 0, 30, "Horror"},
		{2, "user2", "192.168.1.2", "movie", "Abandoned Movie 1", "", "", 0, 0, 35, "Horror"},
		{3, "user3", "192.168.1.3", "movie", "Abandoned Movie 2", "", "", 0, 0, 50, "Comedy"},
		{3, "user3", "192.168.1.3", "movie", "Abandoned Movie 2", "", "", 0, 0, 55, "Comedy"},
		{3, "user3", "192.168.1.3", "movie", "Abandoned Movie 2", "", "", 0, 0, 60, "Comedy"},

		// First episode data for TV show analysis
		{4, "user4", "192.168.1.4", "episode", "Pilot", "Breaking Bad", "Season 1", 1, 1, 100, "Drama"},
		{4, "user4", "192.168.1.4", "episode", "Cat's in the Bag", "Breaking Bad", "Season 1", 2, 1, 95, "Drama"},
		{5, "user5", "192.168.1.5", "episode", "Pilot", "Breaking Bad", "Season 1", 1, 1, 40, "Drama"},
		// User 5 didn't continue watching

		// Music tracks
		{1, "user1", "192.168.1.1", "track", "Song 1", "", "", 0, 0, 100, ""},
		{1, "user1", "192.168.1.1", "track", "Song 2", "", "", 0, 0, 50, ""},
	}

	for _, pb := range playbacks {
		startedAt := now.Add(-time.Duration(len(playbacks)) * time.Hour)
		_, err := db.conn.Exec(`
			INSERT INTO playback_events (
				id, session_key, started_at, stopped_at, user_id, username,
				ip_address, media_type, title, grandparent_title, parent_title,
				media_index, parent_media_index, percent_complete, genres, play_duration
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), uuid.New().String(), startedAt,
			startedAt.Add(90*time.Minute), pb.userID, pb.username, pb.ip,
			pb.mediaType, pb.title,
			nullableString(pb.grandparentTitle), nullableString(pb.parentTitle),
			pb.mediaIndex, pb.parentMediaIndex, pb.percentComplete,
			nullableString(pb.genres), 90)
		if err != nil {
			t.Fatalf("Failed to insert abandonment playback: %v", err)
		}
	}
}
