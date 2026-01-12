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

// Test helpers to reduce cyclomatic complexity

// assertIntEqual checks if two integers are equal
func assertIntEqual(t *testing.T, got, want int, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %d, want %d", msg, got, want)
	}
}

// assertStringEqual checks if two strings are equal
func assertStringEqual(t *testing.T, got, want, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %s, want %s", msg, got, want)
	}
}

// assertFloatEqual checks if two floats are equal
func assertFloatEqual(t *testing.T, got, want float64, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %.2f, want %.2f", msg, got, want)
	}
}

// assertSliceNotEmpty checks if a slice is not empty
func assertSliceNotEmpty(t *testing.T, length int, msg string) {
	t.Helper()
	if length == 0 {
		t.Errorf("%s: expected non-empty slice, got length 0", msg)
	}
}

// assertSliceEmpty checks if a slice is empty
func assertSliceEmpty(t *testing.T, length int, msg string) {
	t.Helper()
	if length != 0 {
		t.Errorf("%s: expected empty slice, got length %d", msg, length)
	}
}

// assertGreaterThan checks if a value is greater than expected
func assertGreaterThan(t *testing.T, got, minValue int, msg string) {
	t.Helper()
	if got <= minValue {
		t.Errorf("%s: got %d, want > %d", msg, got, minValue)
	}
}

// insertBingeEpisodes inserts binge watching test episodes
func insertBingeEpisodes(t *testing.T, db *DB, now time.Time, showName, username string, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		event := &models.PlaybackEvent{
			ID:               uuid.New(),
			SessionKey:       uuid.New().String(),
			StartedAt:        now.Add(time.Duration(i*50) * time.Minute),
			UserID:           1,
			Username:         username,
			IPAddress:        "192.168.1.1",
			MediaType:        "episode",
			Title:            "Episode",
			GrandparentTitle: &showName,
			PercentComplete:  100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert episode: %v", err)
		}
	}
}

// insertMoviePlaybacks inserts movie playback events
func insertMoviePlaybacks(t *testing.T, db *DB, now time.Time, title string, count, percentComplete int) {
	t.Helper()
	for i := 0; i < count; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-i) * time.Hour),
			UserID:          i + 1,
			Username:        "user",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           title,
			PercentComplete: percentComplete,
			PlayDuration:    intPtr(45),
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert movie playback: %v", err)
		}
	}
}

// TestGetBingeAnalytics_WithBingeSessions tests binge watching detection
func TestGetBingeAnalytics_WithBingeSessions(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()
	showName := "Breaking Bad"

	episodes := []struct {
		title      string
		offsetMins int
	}{
		{"Pilot", 0},
		{"Cat's in the Bag...", 50},
		{"...And the Bag's in the River", 100},
		{"Cancer Man", 150},
	}

	for i, ep := range episodes {
		event := &models.PlaybackEvent{
			ID:               uuid.New(),
			SessionKey:       uuid.New().String(),
			StartedAt:        now.Add(time.Duration(ep.offsetMins) * time.Minute),
			UserID:           1,
			Username:         "bingewatcher",
			IPAddress:        "192.168.1.1",
			MediaType:        "episode",
			Title:            ep.title,
			ParentTitle:      strPtr("Season 1"),
			GrandparentTitle: &showName,
			Platform:         "Plex Web",
			Player:           "Chrome",
			PercentComplete:  95,
			PlayDuration:     intPtr(45),
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert episode %d: %v", i, err)
		}
	}

	analytics, err := db.GetBingeAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetBingeAnalytics failed: %v", err)
	}

	assertIntEqual(t, analytics.TotalBingeSessions, 1, "Total binge sessions")
	assertIntEqual(t, analytics.TotalEpisodesBinged, 4, "Total episodes binged")
	assertIntEqual(t, len(analytics.RecentBingeSessions), 1, "Recent binge sessions count")

	if len(analytics.RecentBingeSessions) == 0 {
		return
	}
	session := analytics.RecentBingeSessions[0]
	assertStringEqual(t, session.ShowName, showName, "Show name")
	assertIntEqual(t, session.EpisodeCount, 4, "Episode count in session")
}

// TestGetBingeAnalytics_NoBingeSessions tests when no binge sessions exist
func TestGetBingeAnalytics_NoBingeSessions(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()
	showName := "The Office"
	insertBingeEpisodes(t, db, now, showName, "casualviewer", 2)

	analytics, err := db.GetBingeAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetBingeAnalytics failed: %v", err)
	}

	assertIntEqual(t, analytics.TotalBingeSessions, 0, "Total binge sessions")
	assertIntEqual(t, analytics.TotalEpisodesBinged, 0, "Total episodes binged")
}

// TestGetBingeAnalytics_WithUserFilter tests filtering by user
func TestGetBingeAnalytics_WithUserFilter(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()
	showName := "Stranger Things"

	insertBingeEpisodes(t, db, now, showName, "user1", 4)

	for i := 0; i < 3; i++ {
		event := &models.PlaybackEvent{
			ID:               uuid.New(),
			SessionKey:       uuid.New().String(),
			StartedAt:        now.Add(time.Duration(i*50) * time.Minute),
			UserID:           2,
			Username:         "user2",
			IPAddress:        "192.168.1.2",
			MediaType:        "episode",
			Title:            "Episode",
			GrandparentTitle: &showName,
			PercentComplete:  100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	filter := LocationStatsFilter{Users: []string{"user1"}}

	analytics, err := db.GetBingeAnalytics(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetBingeAnalytics failed: %v", err)
	}

	assertIntEqual(t, analytics.TotalBingeSessions, 1, "Binge sessions for user1")
	assertIntEqual(t, analytics.TotalEpisodesBinged, 4, "Episodes binged for user1")
}

// TestGetBandwidthAnalytics_Success tests bandwidth analytics
func TestGetBandwidthAnalytics_Success(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()

	events := []struct {
		transcodeDecision string
		videoResolution   string
	}{
		{"direct play", "1080p"},
		{"direct play", "4k"},
		{"transcode", "720p"},
		{"transcode", "480p"},
		{"copy", "1080p"},
	}

	for i, e := range events {
		duration := 45
		event := &models.PlaybackEvent{
			ID:                uuid.New(),
			SessionKey:        uuid.New().String(),
			StartedAt:         now.Add(time.Duration(-i) * time.Hour),
			UserID:            1,
			Username:          "testuser",
			IPAddress:         "192.168.1.1",
			MediaType:         "movie",
			Title:             "Test Movie",
			Platform:          "Plex Web",
			Player:            "Chrome",
			TranscodeDecision: &e.transcodeDecision,
			VideoResolution:   &e.videoResolution,
			PlayDuration:      &duration,
			PercentComplete:   100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	analytics, err := db.GetBandwidthAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetBandwidthAnalytics failed: %v", err)
	}

	assertSliceNotEmpty(t, len(analytics.ByTranscode), "Transcode categories")

	totalPlaybacks := 0
	for _, bt := range analytics.ByTranscode {
		totalPlaybacks += bt.PlaybackCount
	}
	assertGreaterThan(t, totalPlaybacks, 0, "Total playback counts")
}

// TestGetPopularContent_Success tests popular content analytics
func TestGetPopularContent_Success(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()

	insertMoviePlaybacks(t, db, now, "Popular Movie", 5, 100)

	event := &models.PlaybackEvent{
		ID:              uuid.New(),
		SessionKey:      uuid.New().String(),
		StartedAt:       now,
		UserID:          10,
		Username:        "user",
		IPAddress:       "192.168.1.1",
		MediaType:       "movie",
		Title:           "Unpopular Movie",
		PercentComplete: 50,
	}
	if err := db.InsertPlaybackEvent(event); err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	analytics, err := db.GetPopularContent(context.Background(), LocationStatsFilter{}, 10)
	if err != nil {
		t.Fatalf("GetPopularContent failed: %v", err)
	}

	assertSliceNotEmpty(t, len(analytics.TopMovies), "Popular movies")

	if len(analytics.TopMovies) == 0 {
		return
	}
	topMovie := analytics.TopMovies[0]
	assertStringEqual(t, topMovie.Title, "Popular Movie", "Most popular movie title")
	assertIntEqual(t, topMovie.PlaybackCount, 5, "Most popular movie playback count")
}

// TestGetWatchParties_DetectsSimultaneousViewing tests watch party detection
func TestGetWatchParties_DetectsSimultaneousViewing(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()
	title := "Avengers: Endgame"

	for i := 1; i <= 3; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now,
			UserID:          i,
			Username:        "user",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           title,
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	analytics, err := db.GetWatchParties(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetWatchParties failed: %v", err)
	}

	assertGreaterThan(t, analytics.TotalWatchParties, 0, "Total watch parties")
}

// TestGetUserEngagement_Success tests user engagement metrics
func TestGetUserEngagement_Success(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()

	for userID := 1; userID <= 3; userID++ {
		for i := 0; i < userID*2; i++ {
			duration := 45
			event := &models.PlaybackEvent{
				ID:              uuid.New(),
				SessionKey:      uuid.New().String(),
				StartedAt:       now.Add(time.Duration(-i) * time.Hour),
				UserID:          userID,
				Username:        "user",
				IPAddress:       "192.168.1.1",
				MediaType:       "movie",
				Title:           "Movie",
				Platform:        "Plex Web",
				Player:          "Chrome",
				PlayDuration:    &duration,
				PercentComplete: 90,
			}
			if err := db.InsertPlaybackEvent(event); err != nil {
				t.Fatalf("Failed to insert: %v", err)
			}
		}
	}

	analytics, err := db.GetUserEngagement(context.Background(), LocationStatsFilter{}, 10)
	if err != nil {
		t.Fatalf("GetUserEngagement failed: %v", err)
	}

	assertGreaterThan(t, analytics.Summary.ActiveUsers, 0, "Active users")
	assertSliceNotEmpty(t, len(analytics.TopUsers), "Top users")
}

// TestGetComparativeAnalytics_WeekOverWeek tests comparative analytics
func TestGetComparativeAnalytics_WeekOverWeek(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()

	// Current week: 5 playbacks
	insertMoviePlaybacks(t, db, now, "Current Movie", 5, 100)

	// Previous week: 3 playbacks
	for i := 0; i < 3; i++ {
		duration := 45
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(-7*24-i) * time.Hour),
			UserID:          1,
			Username:        "user",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Previous Movie",
			Platform:        "Plex Web",
			Player:          "Chrome",
			PlayDuration:    &duration,
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert previous week event: %v", err)
		}
	}

	startDate := now.Add(-7 * 24 * time.Hour)
	endDate := now
	filter := LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	analytics, err := db.GetComparativeAnalytics(context.Background(), filter, "week")
	if err != nil {
		t.Fatalf("GetComparativeAnalytics failed: %v", err)
	}

	assertStringEqual(t, analytics.ComparisonType, "week", "Comparison type")
	assertGreaterThan(t, analytics.CurrentPeriod.PlaybackCount, 0, "Current period playback count")
	assertGreaterThan(t, analytics.PreviousPeriod.PlaybackCount, 0, "Previous period playback count")
}

// TestGetComparativeAnalytics_MonthOverMonth tests month-over-month comparison
func TestGetComparativeAnalytics_MonthOverMonth(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()
	insertMoviePlaybacks(t, db, now, "Movie", 10, 100)

	startDate := now.Add(-30 * 24 * time.Hour)
	endDate := now
	filter := LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	analytics, err := db.GetComparativeAnalytics(context.Background(), filter, "month")
	if err != nil {
		t.Fatalf("GetComparativeAnalytics failed: %v", err)
	}

	assertStringEqual(t, analytics.ComparisonType, "month", "Comparison type")
}

// TestGetContentAbandonmentAnalytics_HighAbandonmentContent tests content with high abandonment rates
func TestGetContentAbandonmentAnalytics_HighAbandonmentContent(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()
	movieTitle := "Abandoned Movie"

	abandonmentRates := []int{25, 40, 60, 75, 95}
	for i, percentComplete := range abandonmentRates {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(i) * time.Hour),
			UserID:          i + 1,
			Username:        "user" + string(rune('A'+i)),
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           movieTitle,
			Year:            intPtr(2023),
			ContentRating:   strPtr("PG-13"),
			Genres:          strPtr("Action, Thriller"),
			Platform:        "Plex Web",
			Player:          "Chrome",
			PercentComplete: percentComplete,
			PlayDuration:    intPtr(45),
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event %d: %v", i, err)
		}
	}

	analytics, err := db.GetContentAbandonmentAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetContentAbandonmentAnalytics failed: %v", err)
	}

	assertIntEqual(t, analytics.Summary.TotalPlaybacks, 5, "Total playbacks")
	assertIntEqual(t, analytics.Summary.CompletedPlaybacks, 1, "Completed playbacks")
	assertIntEqual(t, analytics.Summary.AbandonedPlaybacks, 4, "Abandoned playbacks")
	assertFloatEqual(t, analytics.Summary.CompletionRate, 20.0, "Completion rate")

	assertSliceNotEmpty(t, len(analytics.TopAbandoned), "Top abandoned content")
	assertSliceNotEmpty(t, len(analytics.CompletionByMediaType), "Media type completion")
	assertSliceNotEmpty(t, len(analytics.DropOffDistribution), "Drop-off distribution")
}

// TestGetContentAbandonmentAnalytics_HighCompletionContent tests content with high completion rates
func TestGetContentAbandonmentAnalytics_HighCompletionContent(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()
	movieTitle := "Popular Completed Movie"

	for i := 0; i < 5; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(i) * time.Hour),
			UserID:          i + 1,
			Username:        "completionist" + string(rune('A'+i)),
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           movieTitle,
			Platform:        "Plex Web",
			Player:          "Chrome",
			PercentComplete: 95 + i,
			PlayDuration:    intPtr(90),
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event %d: %v", i, err)
		}
	}

	analytics, err := db.GetContentAbandonmentAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetContentAbandonmentAnalytics failed: %v", err)
	}

	assertIntEqual(t, analytics.Summary.TotalPlaybacks, 5, "Total playbacks")
	assertIntEqual(t, analytics.Summary.CompletedPlaybacks, 5, "Completed playbacks")
	assertIntEqual(t, analytics.Summary.AbandonedPlaybacks, 0, "Abandoned playbacks")
	assertFloatEqual(t, analytics.Summary.CompletionRate, 100.0, "Completion rate")
}

// TestGetContentAbandonmentAnalytics_TVShowFirstEpisodeDropOff tests first episode abandonment
func TestGetContentAbandonmentAnalytics_TVShowFirstEpisodeDropOff(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()
	showName := "Test Show"

	for i := 0; i < 5; i++ {
		percentComplete := 50
		if i < 2 {
			percentComplete = 95
		}

		event := &models.PlaybackEvent{
			ID:                   uuid.New(),
			SessionKey:           uuid.New().String(),
			StartedAt:            now.Add(time.Duration(i) * time.Hour),
			UserID:               i + 1,
			Username:             "viewer" + string(rune('A'+i)),
			IPAddress:            "192.168.1.1",
			MediaType:            "episode",
			Title:                "Pilot",
			ParentTitle:          strPtr("Season 1"),
			GrandparentTitle:     &showName,
			MediaIndex:           intPtr(1),
			ParentMediaIndex:     intPtr(1),
			GrandparentRatingKey: strPtr("show123"),
			Platform:             "Plex Web",
			Player:               "Chrome",
			PercentComplete:      percentComplete,
			PlayDuration:         intPtr(45),
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert pilot event %d: %v", i, err)
		}
	}

	for i := 0; i < 2; i++ {
		event := &models.PlaybackEvent{
			ID:                   uuid.New(),
			SessionKey:           uuid.New().String(),
			StartedAt:            now.Add(time.Duration(i+10) * time.Hour),
			UserID:               i + 1,
			Username:             "viewer" + string(rune('A'+i)),
			IPAddress:            "192.168.1.1",
			MediaType:            "episode",
			Title:                "Episode 2",
			ParentTitle:          strPtr("Season 1"),
			GrandparentTitle:     &showName,
			MediaIndex:           intPtr(2),
			ParentMediaIndex:     intPtr(1),
			GrandparentRatingKey: strPtr("show123"),
			Platform:             "Plex Web",
			Player:               "Chrome",
			PercentComplete:      95,
			PlayDuration:         intPtr(45),
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert continuation event %d: %v", i, err)
		}
	}

	analytics, err := db.GetContentAbandonmentAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetContentAbandonmentAnalytics failed: %v", err)
	}

	assertSliceNotEmpty(t, len(analytics.FirstEpisodeAbandonment), "First episode abandonment data")
}

// TestGetContentAbandonmentAnalytics_FilterByMediaType tests filtering by media type
func TestGetContentAbandonmentAnalytics_FilterByMediaType(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	now := time.Now()

	for i := 0; i < 3; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(time.Duration(i) * time.Hour),
			UserID:          i + 1,
			Username:        "moviefan" + string(rune('A'+i)),
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Movie " + string(rune('A'+i)),
			Platform:        "Plex Web",
			Player:          "Chrome",
			PercentComplete: 50,
			PlayDuration:    intPtr(90),
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert movie event %d: %v", i, err)
		}
	}

	for i := 0; i < 2; i++ {
		event := &models.PlaybackEvent{
			ID:               uuid.New(),
			SessionKey:       uuid.New().String(),
			StartedAt:        now.Add(time.Duration(i+10) * time.Hour),
			UserID:           i + 1,
			Username:         "tvfan" + string(rune('A'+i)),
			IPAddress:        "192.168.1.1",
			MediaType:        "episode",
			Title:            "Episode " + string(rune('A'+i)),
			GrandparentTitle: strPtr("TV Show"),
			Platform:         "Plex Web",
			Player:           "Chrome",
			PercentComplete:  95,
			PlayDuration:     intPtr(45),
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert episode event %d: %v", i, err)
		}
	}

	filter := LocationStatsFilter{MediaTypes: []string{"movie"}}

	analytics, err := db.GetContentAbandonmentAnalytics(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetContentAbandonmentAnalytics with filter failed: %v", err)
	}

	assertIntEqual(t, analytics.Summary.TotalPlaybacks, 3, "Total playbacks (movies only)")
}

// TestGetContentAbandonmentAnalytics_EmptyDataset tests with no playback events
func TestGetContentAbandonmentAnalytics_EmptyDataset(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	insertTestGeolocations(t, db)

	analytics, err := db.GetContentAbandonmentAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetContentAbandonmentAnalytics with empty dataset failed: %v", err)
	}

	assertIntEqual(t, analytics.Summary.TotalPlaybacks, 0, "Total playbacks")
	assertIntEqual(t, analytics.Summary.CompletedPlaybacks, 0, "Completed playbacks")
	assertSliceEmpty(t, len(analytics.TopAbandoned), "Top abandoned list")
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
