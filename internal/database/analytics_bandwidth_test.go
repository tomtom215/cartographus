// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

func TestGetBitrateAnalytics_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert test data with bitrate information
	events := []models.PlaybackEvent{
		{
			SessionKey:       "session1",
			Username:         "user1",
			Title:            "Test Movie 4K",
			MediaType:        "movie",
			SourceBitrate:    intPtr(25000), // 25 Mbps source (4K)
			TranscodeBitrate: intPtr(8000),  // 8 Mbps transcode (1080p)
			NetworkBandwidth: intPtr(10000), // 10 Mbps network
			VideoResolution:  strPtr("4k"),
			StartedAt:        time.Now().Add(-24 * time.Hour),
		},
		{
			SessionKey:       "session2",
			Username:         "user2",
			Title:            "Test Movie 1080p",
			MediaType:        "movie",
			SourceBitrate:    intPtr(8000), // 8 Mbps source
			TranscodeBitrate: intPtr(4000), // 4 Mbps transcode
			NetworkBandwidth: intPtr(6000), // 6 Mbps network
			VideoResolution:  strPtr("1080p"),
			StartedAt:        time.Now().Add(-12 * time.Hour),
		},
		{
			SessionKey:       "session3",
			Username:         "user1",
			Title:            "Test Movie 720p",
			MediaType:        "movie",
			SourceBitrate:    intPtr(4000), // 4 Mbps source
			TranscodeBitrate: intPtr(2000), // 2 Mbps transcode
			NetworkBandwidth: intPtr(3000), // 3 Mbps network
			VideoResolution:  strPtr("720p"),
			StartedAt:        time.Now().Add(-6 * time.Hour),
		},
	}

	for _, event := range events {
		if err := db.InsertPlaybackEvent(&event); err != nil {
			t.Fatalf("Failed to insert test event: %v", err)
		}
	}

	// Test GetBitrateAnalytics with no filter
	filter := LocationStatsFilter{}
	analytics, err := db.GetBitrateAnalytics(ctx, filter)
	if err != nil {
		t.Fatalf("GetBitrateAnalytics failed: %v", err)
	}

	// Verify analytics structure
	if analytics == nil {
		t.Fatal("Expected analytics, got nil")
	}

	// Verify average bitrates
	if analytics.AverageSourceBitrate == 0 {
		t.Error("Expected non-zero average source bitrate")
	}
	if analytics.AverageTranscodeBitrate == 0 {
		t.Error("Expected non-zero average transcode bitrate")
	}

	// Verify peak bitrate (should be highest source bitrate)
	if analytics.PeakBitrate != 25000 {
		t.Errorf("Expected peak bitrate 25000, got %d", analytics.PeakBitrate)
	}

	// Verify median bitrate exists
	if analytics.MedianBitrate == 0 {
		t.Error("Expected non-zero median bitrate")
	}

	// Verify bandwidth utilization percentage
	if analytics.BandwidthUtilization < 0 || analytics.BandwidthUtilization > 100 {
		t.Errorf("Expected bandwidth utilization 0-100%%, got %.2f", analytics.BandwidthUtilization)
	}

	// Verify constrained sessions count
	if analytics.ConstrainedSessions < 0 {
		t.Errorf("Expected non-negative constrained sessions, got %d", analytics.ConstrainedSessions)
	}

	// Verify bitrate by resolution array
	if len(analytics.BitrateByResolution) == 0 {
		t.Error("Expected non-empty bitrate by resolution array")
	}

	// Verify resolution tiers exist (should have 4K, 1080p, 720p)
	resolutionMap := make(map[string]bool)
	for _, item := range analytics.BitrateByResolution {
		resolutionMap[item.Resolution] = true

		// Verify each item has valid data
		if item.AverageBitrate == 0 {
			t.Errorf("Expected non-zero average bitrate for resolution %s", item.Resolution)
		}
		if item.SessionCount == 0 {
			t.Errorf("Expected non-zero session count for resolution %s", item.Resolution)
		}
		if item.TranscodeRate < 0 || item.TranscodeRate > 100 {
			t.Errorf("Expected transcode rate 0-100%% for resolution %s, got %.2f", item.Resolution, item.TranscodeRate)
		}
	}

	// Verify expected resolutions are present
	expectedResolutions := []string{"4K", "1080p", "720p"}
	for _, res := range expectedResolutions {
		if !resolutionMap[res] {
			t.Errorf("Expected resolution %s in results", res)
		}
	}

	// Verify time series array
	if len(analytics.BitrateTimeSeries) == 0 {
		t.Error("Expected non-empty bitrate time series array")
	}

	// Verify time series items have valid data
	for i, item := range analytics.BitrateTimeSeries {
		if item.Timestamp == "" {
			t.Errorf("Expected non-empty timestamp for time series item %d", i)
		}
		if item.AverageBitrate == 0 {
			t.Errorf("Expected non-zero average bitrate for time series item %d", i)
		}
		if item.PeakBitrate == 0 {
			t.Errorf("Expected non-zero peak bitrate for time series item %d", i)
		}
		if item.ActiveSessions == 0 {
			t.Errorf("Expected non-zero active sessions for time series item %d", i)
		}
	}
}

func TestGetBitrateAnalytics_WithDateRangeFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert events with different dates
	now := time.Now()
	events := []models.PlaybackEvent{
		{
			SessionKey:       "old_session",
			Username:         "user1",
			SourceBitrate:    intPtr(10000),
			TranscodeBitrate: intPtr(5000),
			NetworkBandwidth: intPtr(6000),
			VideoResolution:  strPtr("1080p"),
			StartedAt:        now.Add(-60 * 24 * time.Hour), // 60 days ago (outside 30-day window)
		},
		{
			SessionKey:       "recent_session",
			Username:         "user1",
			SourceBitrate:    intPtr(8000),
			TranscodeBitrate: intPtr(4000),
			NetworkBandwidth: intPtr(5000),
			VideoResolution:  strPtr("1080p"),
			StartedAt:        now.Add(-5 * 24 * time.Hour), // 5 days ago (inside filter)
		},
	}

	for _, event := range events {
		if err := db.InsertPlaybackEvent(&event); err != nil {
			t.Fatalf("Failed to insert test event: %v", err)
		}
	}

	// Test with date range filter (last 7 days)
	startDate := now.Add(-7 * 24 * time.Hour)
	endDate := now
	filter := LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	analytics, err := db.GetBitrateAnalytics(ctx, filter)
	if err != nil {
		t.Fatalf("GetBitrateAnalytics with date filter failed: %v", err)
	}

	// Should only include recent session
	if analytics.AverageSourceBitrate != 8000 {
		t.Errorf("Expected average source bitrate 8000 (recent session only), got %d", analytics.AverageSourceBitrate)
	}
}

func TestGetBitrateAnalytics_WithUserFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert events for different users
	events := []models.PlaybackEvent{
		{
			SessionKey:       "user1_session",
			Username:         "user1",
			SourceBitrate:    intPtr(15000),
			TranscodeBitrate: intPtr(7000),
			NetworkBandwidth: intPtr(8000),
			VideoResolution:  strPtr("4k"),
			StartedAt:        time.Now(),
		},
		{
			SessionKey:       "user2_session",
			Username:         "user2",
			SourceBitrate:    intPtr(5000),
			TranscodeBitrate: intPtr(3000),
			NetworkBandwidth: intPtr(4000),
			VideoResolution:  strPtr("720p"),
			StartedAt:        time.Now(),
		},
	}

	for _, event := range events {
		if err := db.InsertPlaybackEvent(&event); err != nil {
			t.Fatalf("Failed to insert test event: %v", err)
		}
	}

	// Test with user filter
	filter := LocationStatsFilter{
		Users: []string{"user1"},
	}

	analytics, err := db.GetBitrateAnalytics(ctx, filter)
	if err != nil {
		t.Fatalf("GetBitrateAnalytics with user filter failed: %v", err)
	}

	// Should only include user1's session
	if analytics.AverageSourceBitrate != 15000 {
		t.Errorf("Expected average source bitrate 15000 (user1 only), got %d", analytics.AverageSourceBitrate)
	}
}

func TestGetBitrateAnalytics_ConstrainedSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert sessions with various bandwidth constraints
	events := []models.PlaybackEvent{
		{
			SessionKey:       "constrained1",
			Username:         "user1",
			SourceBitrate:    intPtr(10000), // 10 Mbps bitrate
			TranscodeBitrate: intPtr(10000),
			NetworkBandwidth: intPtr(8000), // 8 Mbps bandwidth (constrained: 10 > 8 * 0.8)
			VideoResolution:  strPtr("1080p"),
			StartedAt:        time.Now(),
		},
		{
			SessionKey:       "constrained2",
			Username:         "user2",
			SourceBitrate:    intPtr(15000), // 15 Mbps bitrate
			TranscodeBitrate: intPtr(15000),
			NetworkBandwidth: intPtr(10000), // 10 Mbps bandwidth (constrained: 15 > 10 * 0.8)
			VideoResolution:  strPtr("4k"),
			StartedAt:        time.Now(),
		},
		{
			SessionKey:       "unconstrained",
			Username:         "user3",
			SourceBitrate:    intPtr(5000), // 5 Mbps bitrate
			TranscodeBitrate: intPtr(5000),
			NetworkBandwidth: intPtr(10000), // 10 Mbps bandwidth (not constrained: 5 <= 10 * 0.8)
			VideoResolution:  strPtr("720p"),
			StartedAt:        time.Now(),
		},
	}

	for _, event := range events {
		if err := db.InsertPlaybackEvent(&event); err != nil {
			t.Fatalf("Failed to insert test event: %v", err)
		}
	}

	filter := LocationStatsFilter{}
	analytics, err := db.GetBitrateAnalytics(ctx, filter)
	if err != nil {
		t.Fatalf("GetBitrateAnalytics failed: %v", err)
	}

	// Should detect 2 constrained sessions
	if analytics.ConstrainedSessions != 2 {
		t.Errorf("Expected 2 constrained sessions, got %d", analytics.ConstrainedSessions)
	}
}

func TestGetBitrateAnalytics_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	filter := LocationStatsFilter{}

	analytics, err := db.GetBitrateAnalytics(ctx, filter)
	if err != nil {
		t.Fatalf("GetBitrateAnalytics on empty DB failed: %v", err)
	}

	// Should return zero values for empty database
	if analytics.AverageSourceBitrate != 0 {
		t.Errorf("Expected zero average source bitrate, got %d", analytics.AverageSourceBitrate)
	}
	if analytics.ConstrainedSessions != 0 {
		t.Errorf("Expected zero constrained sessions, got %d", analytics.ConstrainedSessions)
	}
	if len(analytics.BitrateByResolution) != 0 {
		t.Errorf("Expected empty resolution array, got %d items", len(analytics.BitrateByResolution))
	}
	if len(analytics.BitrateTimeSeries) != 0 {
		t.Errorf("Expected empty time series array, got %d items", len(analytics.BitrateTimeSeries))
	}
}

func TestGetBitrateAnalytics_TranscodeRateCalculation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert 4K sessions: 3 transcoded, 1 direct play
	events := []models.PlaybackEvent{
		{
			SessionKey:       "4k_transcode1",
			Username:         "user1",
			SourceBitrate:    intPtr(25000),
			TranscodeBitrate: intPtr(8000), // Transcoded (different from source)
			NetworkBandwidth: intPtr(10000),
			VideoResolution:  strPtr("4k"),
			StartedAt:        time.Now(),
		},
		{
			SessionKey:       "4k_transcode2",
			Username:         "user2",
			SourceBitrate:    intPtr(25000),
			TranscodeBitrate: intPtr(8000), // Transcoded
			NetworkBandwidth: intPtr(10000),
			VideoResolution:  strPtr("4k"),
			StartedAt:        time.Now(),
		},
		{
			SessionKey:       "4k_transcode3",
			Username:         "user3",
			SourceBitrate:    intPtr(25000),
			TranscodeBitrate: intPtr(8000), // Transcoded
			NetworkBandwidth: intPtr(10000),
			VideoResolution:  strPtr("4k"),
			StartedAt:        time.Now(),
		},
		{
			SessionKey:       "4k_direct",
			Username:         "user4",
			SourceBitrate:    intPtr(25000),
			TranscodeBitrate: intPtr(25000), // Direct play (same as source)
			NetworkBandwidth: intPtr(30000),
			VideoResolution:  strPtr("4k"),
			StartedAt:        time.Now(),
		},
	}

	for _, event := range events {
		if err := db.InsertPlaybackEvent(&event); err != nil {
			t.Fatalf("Failed to insert test event: %v", err)
		}
	}

	filter := LocationStatsFilter{}
	analytics, err := db.GetBitrateAnalytics(ctx, filter)
	if err != nil {
		t.Fatalf("GetBitrateAnalytics failed: %v", err)
	}

	// Find 4K resolution in results
	var fourKItem *models.BitrateByResolutionItem
	for i, item := range analytics.BitrateByResolution {
		if item.Resolution == "4K" {
			fourKItem = &analytics.BitrateByResolution[i]
			break
		}
	}

	if fourKItem == nil {
		t.Fatal("Expected 4K resolution in results")
	}

	// Verify session count
	if fourKItem.SessionCount != 4 {
		t.Errorf("Expected 4 sessions for 4K, got %d", fourKItem.SessionCount)
	}

	// Verify transcode rate (3 out of 4 = 75%)
	expectedRate := 75.0
	tolerance := 1.0 // Allow 1% tolerance for floating point
	if fourKItem.TranscodeRate < expectedRate-tolerance || fourKItem.TranscodeRate > expectedRate+tolerance {
		t.Errorf("Expected transcode rate ~%.1f%%, got %.2f%%", expectedRate, fourKItem.TranscodeRate)
	}
}

func TestGetBitrateAnalytics_TimeSeriesOrdering(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()

	// Insert events across multiple days
	for i := 0; i < 5; i++ {
		event := models.PlaybackEvent{
			SessionKey:       "session_day_" + string(rune(i)),
			Username:         "user1",
			SourceBitrate:    intPtr(10000),
			TranscodeBitrate: intPtr(5000),
			NetworkBandwidth: intPtr(6000),
			VideoResolution:  strPtr("1080p"),
			StartedAt:        now.Add(-time.Duration(i*24) * time.Hour),
		}
		if err := db.InsertPlaybackEvent(&event); err != nil {
			t.Fatalf("Failed to insert test event: %v", err)
		}
	}

	filter := LocationStatsFilter{}
	analytics, err := db.GetBitrateAnalytics(ctx, filter)
	if err != nil {
		t.Fatalf("GetBitrateAnalytics failed: %v", err)
	}

	// Verify time series is ordered chronologically (oldest to newest)
	if len(analytics.BitrateTimeSeries) < 2 {
		t.Skip("Need at least 2 time series items to test ordering")
	}

	for i := 1; i < len(analytics.BitrateTimeSeries); i++ {
		prevTime, err1 := time.Parse(time.RFC3339, analytics.BitrateTimeSeries[i-1].Timestamp)
		currTime, err2 := time.Parse(time.RFC3339, analytics.BitrateTimeSeries[i].Timestamp)

		if err1 != nil || err2 != nil {
			t.Fatalf("Failed to parse timestamps: %v, %v", err1, err2)
		}

		if currTime.Before(prevTime) {
			t.Errorf("Time series not in chronological order: %s before %s",
				analytics.BitrateTimeSeries[i].Timestamp,
				analytics.BitrateTimeSeries[i-1].Timestamp)
		}
	}
}
