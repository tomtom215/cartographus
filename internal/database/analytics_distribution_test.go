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

// newTestEvent creates a base playback event with sensible defaults for testing.
// The index is used to offset the StartedAt time for ordering.
func newTestEvent(index int) *models.PlaybackEvent {
	return &models.PlaybackEvent{
		ID:              uuid.New(),
		SessionKey:      uuid.New().String(),
		StartedAt:       time.Now().Add(time.Duration(-index) * time.Hour),
		UserID:          1,
		Username:        "testuser",
		IPAddress:       "192.168.1.1",
		MediaType:       "movie",
		Title:           "Test Movie",
		PercentComplete: 100,
	}
}

// insertTestEventsSlice inserts multiple events into the database.
func insertTestEventsSlice(t *testing.T, db *DB, events []*models.PlaybackEvent) {
	t.Helper()
	for _, event := range events {
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}
}

// TestGetTranscodeDistribution_Success tests transcode decision distribution
func TestGetTranscodeDistribution_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	// Insert events with different transcode decisions
	transcodeDecisions := []string{"direct play", "transcode", "copy", "direct play", "transcode"}
	events := make([]*models.PlaybackEvent, len(transcodeDecisions))
	for i, decision := range transcodeDecisions {
		events[i] = newTestEvent(i)
		events[i].TranscodeDecision = &decision
	}
	insertTestEventsSlice(t, db, events)

	stats, err := db.GetTranscodeDistribution(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetTranscodeDistribution failed: %v", err)
	}

	if len(stats) == 0 {
		t.Error("Expected non-empty transcode stats")
	}

	// Verify we have the expected categories
	foundDirectPlay := false
	foundTranscode := false
	for _, stat := range stats {
		if stat.TranscodeDecision == "direct play" {
			foundDirectPlay = true
			if stat.PlaybackCount != 2 {
				t.Errorf("Expected 2 direct play events, got %d", stat.PlaybackCount)
			}
		}
		if stat.TranscodeDecision == "transcode" {
			foundTranscode = true
			if stat.PlaybackCount != 2 {
				t.Errorf("Expected 2 transcode events, got %d", stat.PlaybackCount)
			}
		}
	}

	if !foundDirectPlay {
		t.Error("Expected to find direct play in results")
	}
	if !foundTranscode {
		t.Error("Expected to find transcode in results")
	}
}

// TestGetTranscodeDistribution_WithFilter tests filtering
func TestGetTranscodeDistribution_WithFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	directPlay := "direct play"
	transcode := "transcode"

	// User1: 2 direct play, User2: 1 transcode
	events := []*models.PlaybackEvent{
		func() *models.PlaybackEvent {
			e := newTestEvent(0)
			e.Username = "user1"
			e.TranscodeDecision = &directPlay
			return e
		}(),
		func() *models.PlaybackEvent {
			e := newTestEvent(1)
			e.Username = "user1"
			e.TranscodeDecision = &directPlay
			return e
		}(),
		func() *models.PlaybackEvent {
			e := newTestEvent(2)
			e.UserID = 2
			e.Username = "user2"
			e.IPAddress = "192.168.1.2"
			e.TranscodeDecision = &transcode
			return e
		}(),
	}
	insertTestEventsSlice(t, db, events)

	// Filter by user1
	filter := LocationStatsFilter{
		Users: []string{"user1"},
	}

	stats, err := db.GetTranscodeDistribution(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetTranscodeDistribution failed: %v", err)
	}

	totalCount := 0
	for _, stat := range stats {
		totalCount += stat.PlaybackCount
	}

	if totalCount != 2 {
		t.Errorf("Expected 2 events for user1, got %d", totalCount)
	}
}

// TestGetResolutionDistribution_Success tests video resolution distribution
func TestGetResolutionDistribution_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	resolutions := []string{"1080p", "4K", "720p", "1080p", "4K"}
	events := make([]*models.PlaybackEvent, len(resolutions))
	for i, res := range resolutions {
		events[i] = newTestEvent(i)
		events[i].VideoResolution = &res
	}
	insertTestEventsSlice(t, db, events)

	stats, err := db.GetResolutionDistribution(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetResolutionDistribution failed: %v", err)
	}

	if len(stats) == 0 {
		t.Error("Expected non-empty resolution stats")
	}

	// Verify counts
	resolutionCounts := make(map[string]int)
	for _, stat := range stats {
		resolutionCounts[stat.VideoResolution] = stat.PlaybackCount
	}

	if resolutionCounts["1080p"] != 2 {
		t.Errorf("Expected 2 1080p streams, got %d", resolutionCounts["1080p"])
	}

	if resolutionCounts["4K"] != 2 {
		t.Errorf("Expected 2 4K streams, got %d", resolutionCounts["4K"])
	}
}

// TestGetCodecDistribution_Success tests codec distribution
func TestGetCodecDistribution_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	codecs := []struct {
		video string
		audio string
	}{
		{"h264", "aac"},
		{"hevc", "eac3"},
		{"h264", "aac"},
		{"vp9", "opus"},
	}

	events := make([]*models.PlaybackEvent, len(codecs))
	for i, codec := range codecs {
		events[i] = newTestEvent(i)
		events[i].VideoCodec = &codec.video
		events[i].AudioCodec = &codec.audio
	}
	insertTestEventsSlice(t, db, events)

	stats, err := db.GetCodecDistribution(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetCodecDistribution failed: %v", err)
	}

	if len(stats) == 0 {
		t.Error("Expected non-empty codec stats")
	}

	// Verify we have at least video codec stats
	hasVideoStats := false
	hasAudioStats := false
	for _, stat := range stats {
		if stat.VideoCodec != "" {
			hasVideoStats = true
		}
		if stat.AudioCodec != "" {
			hasAudioStats = true
		}
	}

	if !hasVideoStats {
		t.Error("Expected video codec statistics")
	}
	if !hasAudioStats {
		t.Error("Expected audio codec statistics")
	}
}

// TestGetLibraryStats_Success tests library statistics
func TestGetLibraryStats_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	libraries := []struct {
		name      string
		sectionID int
	}{
		{"Movies", 1},
		{"TV Shows", 2},
		{"Movies", 1},
		{"Movies", 1},
		{"TV Shows", 2},
	}

	events := make([]*models.PlaybackEvent, len(libraries))
	for i, lib := range libraries {
		events[i] = newTestEvent(i)
		events[i].LibraryName = &lib.name
		events[i].SectionID = &lib.sectionID
	}
	insertTestEventsSlice(t, db, events)

	stats, err := db.GetLibraryStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetLibraryStats failed: %v", err)
	}

	if len(stats) != 2 {
		t.Errorf("Expected 2 libraries, got %d", len(stats))
	}

	// Verify counts
	for _, stat := range stats {
		if stat.LibraryName == "Movies" {
			if stat.PlaybackCount != 3 {
				t.Errorf("Expected 3 plays for Movies, got %d", stat.PlaybackCount)
			}
		}
		if stat.LibraryName == "TV Shows" {
			if stat.PlaybackCount != 2 {
				t.Errorf("Expected 2 plays for TV Shows, got %d", stat.PlaybackCount)
			}
		}
	}
}

// TestGetRatingDistribution_Success tests content rating distribution
func TestGetRatingDistribution_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	ratings := []string{"PG", "PG-13", "R", "PG", "PG-13", "PG-13"}
	events := make([]*models.PlaybackEvent, len(ratings))
	for i, rating := range ratings {
		events[i] = newTestEvent(i)
		events[i].ContentRating = &rating
	}
	insertTestEventsSlice(t, db, events)

	stats, err := db.GetRatingDistribution(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetRatingDistribution failed: %v", err)
	}

	if len(stats) != 3 {
		t.Errorf("Expected 3 unique ratings, got %d", len(stats))
	}

	// Verify counts
	ratingCounts := make(map[string]int)
	for _, stat := range stats {
		ratingCounts[stat.ContentRating] = stat.PlaybackCount
	}

	if ratingCounts["PG-13"] != 3 {
		t.Errorf("Expected 3 PG-13 items, got %d", ratingCounts["PG-13"])
	}
	if ratingCounts["PG"] != 2 {
		t.Errorf("Expected 2 PG items, got %d", ratingCounts["PG"])
	}
	if ratingCounts["R"] != 1 {
		t.Errorf("Expected 1 R item, got %d", ratingCounts["R"])
	}
}

// TestGetDurationStats_Success tests playback duration statistics
func TestGetDurationStats_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	// Insert events with different durations (in minutes)
	durations := []int{45, 90, 120, 60, 150}
	events := make([]*models.PlaybackEvent, len(durations))
	for i, duration := range durations {
		events[i] = newTestEvent(i)
		events[i].PlayDuration = &duration
	}
	insertTestEventsSlice(t, db, events)

	stats, err := db.GetDurationStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetDurationStats failed: %v", err)
	}

	if stats.TotalDuration != 465 { // 45+90+120+60+150 = 465
		t.Errorf("Expected 465 total duration minutes, got %d", stats.TotalDuration)
	}

	if stats.AvgDuration != 93 { // 465/5 = 93
		t.Errorf("Expected 93 avg duration minutes, got %d", stats.AvgDuration)
	}

	if stats.MedianDuration == 0 {
		t.Error("Expected non-zero median duration")
	}
}

// TestGetDurationStats_EmptyData tests duration stats with no data
func TestGetDurationStats_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stats, err := db.GetDurationStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetDurationStats failed: %v", err)
	}

	if stats.TotalDuration != 0 {
		t.Errorf("Expected 0 total duration, got %d", stats.TotalDuration)
	}

	if stats.AvgDuration != 0 {
		t.Errorf("Expected 0 avg duration, got %d", stats.AvgDuration)
	}
}

// TestGetYearDistribution_Success tests content year distribution
func TestGetYearDistribution_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	years := []int{2024, 2023, 2022, 2024, 2023, 2024}
	events := make([]*models.PlaybackEvent, len(years))
	for i, year := range years {
		events[i] = newTestEvent(i)
		events[i].Year = &year
	}
	insertTestEventsSlice(t, db, events)

	stats, err := db.GetYearDistribution(context.Background(), LocationStatsFilter{}, 10)
	if err != nil {
		t.Fatalf("GetYearDistribution failed: %v", err)
	}

	if len(stats) != 3 {
		t.Errorf("Expected 3 unique years, got %d", len(stats))
	}

	// Verify counts (should be ordered by count DESC)
	if len(stats) > 0 {
		topYear := stats[0]
		if topYear.Year != 2024 {
			t.Errorf("Expected 2024 to be most popular year, got %d", topYear.Year)
		}
		if topYear.PlaybackCount != 3 {
			t.Errorf("Expected 3 plays for 2024, got %d", topYear.PlaybackCount)
		}
	}
}

// TestGetYearDistribution_WithLimit tests limiting results
func TestGetYearDistribution_WithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	// Insert 5 different years
	events := make([]*models.PlaybackEvent, 5)
	for i := 0; i < 5; i++ {
		year := 2020 + i
		events[i] = newTestEvent(i)
		events[i].Year = &year
	}
	insertTestEventsSlice(t, db, events)

	// Limit to top 3
	stats, err := db.GetYearDistribution(context.Background(), LocationStatsFilter{}, 3)
	if err != nil {
		t.Fatalf("GetYearDistribution failed: %v", err)
	}

	if len(stats) > 3 {
		t.Errorf("Expected at most 3 years, got %d", len(stats))
	}
}

// TestGetPlatformDistribution_Success tests platform distribution
func TestGetPlatformDistribution_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	// Insert events with different platforms
	platforms := []string{"Windows", "macOS", "iOS", "Android", "Windows", "iOS", "Windows"}
	events := make([]*models.PlaybackEvent, len(platforms))
	for i, platform := range platforms {
		events[i] = newTestEvent(i)
		events[i].UserID = i%3 + 1
		events[i].Platform = platform
	}
	insertTestEventsSlice(t, db, events)

	stats, err := db.GetPlatformDistribution(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetPlatformDistribution failed: %v", err)
	}

	if len(stats) != 4 {
		t.Errorf("Expected 4 unique platforms, got %d", len(stats))
	}

	// Verify counts by platform
	platformCounts := make(map[string]int)
	for _, stat := range stats {
		platformCounts[stat.Platform] = stat.PlaybackCount
	}

	if platformCounts["Windows"] != 3 {
		t.Errorf("Expected 3 Windows plays, got %d", platformCounts["Windows"])
	}
	if platformCounts["iOS"] != 2 {
		t.Errorf("Expected 2 iOS plays, got %d", platformCounts["iOS"])
	}
}

// TestGetPlatformDistribution_EmptyDatabase tests empty database
func TestGetPlatformDistribution_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stats, err := db.GetPlatformDistribution(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetPlatformDistribution on empty DB failed: %v", err)
	}

	if len(stats) != 0 {
		t.Errorf("Expected empty result on empty DB, got %d", len(stats))
	}
}

// TestGetPlatformDistribution_WithFilter tests filtering
func TestGetPlatformDistribution_WithFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	platforms := []string{"Windows", "macOS", "iOS"}
	events := make([]*models.PlaybackEvent, len(platforms))
	for i, platform := range platforms {
		events[i] = newTestEvent(i)
		events[i].Username = "user1"
		events[i].Platform = platform
	}
	insertTestEventsSlice(t, db, events)

	// Filter by user
	stats, err := db.GetPlatformDistribution(context.Background(), LocationStatsFilter{
		Users: []string{"user1"},
	})
	if err != nil {
		t.Fatalf("GetPlatformDistribution with filter failed: %v", err)
	}

	if len(stats) != 3 {
		t.Errorf("Expected 3 platforms for user1, got %d", len(stats))
	}
}

// TestGetPlatformDistribution_UniqueUsers tests unique user counting
func TestGetPlatformDistribution_UniqueUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	// Multiple users on same platform (3 unique users, 5 events)
	events := make([]*models.PlaybackEvent, 5)
	for i := 0; i < 5; i++ {
		events[i] = newTestEvent(i)
		events[i].UserID = i%3 + 1 // 3 unique users
		events[i].Platform = "Windows"
	}
	insertTestEventsSlice(t, db, events)

	stats, err := db.GetPlatformDistribution(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetPlatformDistribution failed: %v", err)
	}

	if len(stats) != 1 {
		t.Fatalf("Expected 1 platform, got %d", len(stats))
	}

	if stats[0].PlaybackCount != 5 {
		t.Errorf("Expected 5 playbacks, got %d", stats[0].PlaybackCount)
	}

	if stats[0].UniqueUsers != 3 {
		t.Errorf("Expected 3 unique users, got %d", stats[0].UniqueUsers)
	}
}

// TestGetPlayerDistribution_Success tests player distribution
func TestGetPlayerDistribution_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	// Insert events with different players
	players := []string{"Plex Web", "Plex for iOS", "Plex for Android", "Plex Web", "Plex for iOS", "Plex Web"}
	events := make([]*models.PlaybackEvent, len(players))
	for i, player := range players {
		events[i] = newTestEvent(i)
		events[i].UserID = i%3 + 1
		events[i].Player = player
	}
	insertTestEventsSlice(t, db, events)

	stats, err := db.GetPlayerDistribution(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetPlayerDistribution failed: %v", err)
	}

	if len(stats) != 3 {
		t.Errorf("Expected 3 unique players, got %d", len(stats))
	}

	// Verify counts by player
	playerCounts := make(map[string]int)
	for _, stat := range stats {
		playerCounts[stat.Player] = stat.PlaybackCount
	}

	if playerCounts["Plex Web"] != 3 {
		t.Errorf("Expected 3 Plex Web plays, got %d", playerCounts["Plex Web"])
	}
	if playerCounts["Plex for iOS"] != 2 {
		t.Errorf("Expected 2 Plex for iOS plays, got %d", playerCounts["Plex for iOS"])
	}
}

// TestGetPlayerDistribution_EmptyDatabase tests empty database
func TestGetPlayerDistribution_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stats, err := db.GetPlayerDistribution(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetPlayerDistribution on empty DB failed: %v", err)
	}

	if len(stats) != 0 {
		t.Errorf("Expected empty result on empty DB, got %d", len(stats))
	}
}

// TestGetPlayerDistribution_WithDateFilter tests date filtering
func TestGetPlayerDistribution_WithDateFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events at different times (days apart instead of hours)
	events := make([]*models.PlaybackEvent, 10)
	for i := 0; i < 10; i++ {
		events[i] = newTestEvent(0)
		events[i].StartedAt = now.Add(time.Duration(-i) * 24 * time.Hour)
		events[i].Player = "Plex Web"
	}
	insertTestEventsSlice(t, db, events)

	// Filter to last 3 days
	startDate := now.Add(-3 * 24 * time.Hour)
	endDate := now

	stats, err := db.GetPlayerDistribution(context.Background(), LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	})
	if err != nil {
		t.Fatalf("GetPlayerDistribution with filter failed: %v", err)
	}

	if len(stats) != 1 {
		t.Fatalf("Expected 1 player, got %d", len(stats))
	}

	// Should have fewer plays with date filter (4 days: today, -1d, -2d, -3d)
	if stats[0].PlaybackCount > 4 {
		t.Errorf("Expected at most 4 plays in 3-day window, got %d", stats[0].PlaybackCount)
	}
}

// TestGetPlayerDistribution_EmptyPlayerExcluded tests that empty players are excluded
func TestGetPlayerDistribution_EmptyPlayerExcluded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert one with player, one without (empty string)
	events := []*models.PlaybackEvent{
		{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now,
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Test Movie",
			Player:          "Plex Web",
			PercentComplete: 100,
		},
		{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(-1 * time.Hour),
			UserID:          1,
			Username:        "testuser",
			IPAddress:       "192.168.1.1",
			MediaType:       "movie",
			Title:           "Test Movie 2",
			Player:          "", // Empty player
			PercentComplete: 100,
		},
	}

	for _, event := range events {
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	stats, err := db.GetPlayerDistribution(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetPlayerDistribution failed: %v", err)
	}

	// Should only return the one with a non-empty player
	if len(stats) != 1 {
		t.Errorf("Expected 1 player (empty excluded), got %d", len(stats))
	}

	if stats[0].PlaybackCount != 1 {
		t.Errorf("Expected 1 playback, got %d", stats[0].PlaybackCount)
	}
}
