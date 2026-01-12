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

// Helper to insert playback events with hardware transcode data
func insertHardwareTranscodeTestData(t *testing.T, db *DB) {
	t.Helper()
	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events with various hardware transcode configurations
	testEvents := []struct {
		hwEncoding     int
		hwDecoding     int
		hwFullPipeline int
		hwDecode       string
		hwDecodeTitle  string
		hwEncode       string
		hwEncodeTitle  string
		throttled      int
		decision       string
	}{
		// Full hardware pipeline (NVIDIA)
		{1, 1, 1, "nvdec", "NVIDIA NVDEC", "nvenc", "NVIDIA NVENC", 0, "transcode"},
		{1, 1, 1, "nvdec", "NVIDIA NVDEC", "nvenc", "NVIDIA NVENC", 0, "transcode"},
		// Hardware decode only (Intel QSV)
		{0, 1, 0, "qsv", "Intel Quick Sync", "", "", 0, "transcode"},
		{0, 1, 0, "qsv", "Intel Quick Sync", "", "", 0, "transcode"},
		// Hardware encode only (VAAPI)
		{1, 0, 0, "", "", "vaapi", "VA-API", 0, "transcode"},
		// Software transcode (throttled)
		{0, 0, 0, "", "", "", "", 1, "transcode"},
		{0, 0, 0, "", "", "", "", 1, "transcode"},
		// Direct play sessions
		{0, 0, 0, "", "", "", "", 0, "direct play"},
		{0, 0, 0, "", "", "", "", 0, "direct play"},
		{0, 0, 0, "", "", "", "", 0, "direct play"},
	}

	for i, te := range testEvents {
		event := &models.PlaybackEvent{
			ID:                      uuid.New(),
			SessionKey:              uuid.New().String(),
			StartedAt:               now.Add(time.Duration(-i) * time.Hour),
			UserID:                  1,
			Username:                "testuser",
			IPAddress:               "192.168.1.1",
			MediaType:               "movie",
			Title:                   "Test Movie",
			TranscodeDecision:       &te.decision,
			TranscodeHWEncoding:     &te.hwEncoding,
			TranscodeHWDecoding:     &te.hwDecoding,
			TranscodeHWFullPipeline: &te.hwFullPipeline,
			TranscodeHWDecode:       &te.hwDecode,
			TranscodeHWDecodeTitle:  &te.hwDecodeTitle,
			TranscodeHWEncode:       &te.hwEncode,
			TranscodeHWEncodeTitle:  &te.hwEncodeTitle,
			TranscodeThrottled:      &te.throttled,
			PercentComplete:         100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}
}

// TestGetHardwareTranscodeStats_Success tests basic hardware transcode statistics
func TestGetHardwareTranscodeStats_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertHardwareTranscodeTestData(t, db)

	stats, err := db.GetHardwareTranscodeStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetHardwareTranscodeStats failed: %v", err)
	}

	// Verify total sessions
	if stats.TotalSessions != 10 {
		t.Errorf("Expected 10 total sessions, got %d", stats.TotalSessions)
	}

	// Verify HW transcode sessions (5 with HW encoding or decoding)
	if stats.HWTranscodeSessions != 5 {
		t.Errorf("Expected 5 HW transcode sessions, got %d", stats.HWTranscodeSessions)
	}

	// Verify SW transcode sessions (2)
	if stats.SWTranscodeSessions != 2 {
		t.Errorf("Expected 2 SW transcode sessions, got %d", stats.SWTranscodeSessions)
	}

	// Verify direct play sessions (3)
	if stats.DirectPlaySessions != 3 {
		t.Errorf("Expected 3 direct play sessions, got %d", stats.DirectPlaySessions)
	}

	// Verify full pipeline count (2)
	if stats.FullPipelineCount != 2 {
		t.Errorf("Expected 2 full pipeline sessions, got %d", stats.FullPipelineCount)
	}

	// Verify throttled sessions (2)
	if stats.ThrottledSessions != 2 {
		t.Errorf("Expected 2 throttled sessions, got %d", stats.ThrottledSessions)
	}

	// Verify percentages
	expectedHWPct := 50.0 // 5/10 * 100
	if stats.HWPercentage != expectedHWPct {
		t.Errorf("Expected HW percentage %.1f%%, got %.1f%%", expectedHWPct, stats.HWPercentage)
	}

	expectedThrottledPct := 20.0 // 2/10 * 100
	if stats.ThrottledPercentage != expectedThrottledPct {
		t.Errorf("Expected throttled percentage %.1f%%, got %.1f%%", expectedThrottledPct, stats.ThrottledPercentage)
	}
}

// TestGetHardwareTranscodeStats_DecoderStats tests decoder statistics
func TestGetHardwareTranscodeStats_DecoderStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertHardwareTranscodeTestData(t, db)

	stats, err := db.GetHardwareTranscodeStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetHardwareTranscodeStats failed: %v", err)
	}

	// Should have decoder stats for nvdec and qsv
	if len(stats.DecoderStats) != 2 {
		t.Errorf("Expected 2 decoder types, got %d", len(stats.DecoderStats))
	}

	// Verify NVDEC is present
	foundNvdec := false
	for _, d := range stats.DecoderStats {
		if d.CodecName == "nvdec" {
			foundNvdec = true
			if d.SessionCount != 2 {
				t.Errorf("Expected 2 nvdec sessions, got %d", d.SessionCount)
			}
		}
	}
	if !foundNvdec {
		t.Error("Expected to find nvdec in decoder stats")
	}
}

// TestGetHardwareTranscodeStats_EncoderStats tests encoder statistics
func TestGetHardwareTranscodeStats_EncoderStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertHardwareTranscodeTestData(t, db)

	stats, err := db.GetHardwareTranscodeStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetHardwareTranscodeStats failed: %v", err)
	}

	// Should have encoder stats for nvenc and vaapi
	if len(stats.EncoderStats) != 2 {
		t.Errorf("Expected 2 encoder types, got %d", len(stats.EncoderStats))
	}
}

// TestGetHardwareTranscodeStats_WithFilter tests filtering by date
func TestGetHardwareTranscodeStats_WithFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertHardwareTranscodeTestData(t, db)

	now := time.Now()
	startDate := now.Add(-3 * time.Hour)
	endDate := now

	stats, err := db.GetHardwareTranscodeStats(context.Background(), LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	})
	if err != nil {
		t.Fatalf("GetHardwareTranscodeStats with filter failed: %v", err)
	}

	// With 3-hour window, should have fewer sessions
	if stats.TotalSessions >= 10 {
		t.Errorf("Expected fewer than 10 sessions with date filter, got %d", stats.TotalSessions)
	}
}

// TestGetHardwareTranscodeStats_EmptyDatabase tests empty database case
func TestGetHardwareTranscodeStats_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stats, err := db.GetHardwareTranscodeStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetHardwareTranscodeStats on empty DB failed: %v", err)
	}

	if stats.TotalSessions != 0 {
		t.Errorf("Expected 0 sessions on empty DB, got %d", stats.TotalSessions)
	}
	if stats.HWPercentage != 0 {
		t.Errorf("Expected 0%% HW on empty DB, got %.1f%%", stats.HWPercentage)
	}
}

// Helper to insert HDR content test data
func insertHDRTestData(t *testing.T, db *DB) {
	t.Helper()
	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events with various HDR configurations
	testEvents := []struct {
		streamVideoFullResolution string
		videoFullResolution       string
		videoColorSpace           string
		videoColorPrimaries       string
	}{
		// HDR10 content
		{"3840x2160", "3840x2160 (4K HDR10)", "bt2020nc", "bt2020"},
		{"3840x2160", "3840x2160 (4K HDR10)", "bt2020nc", "bt2020"},
		// Dolby Vision content
		{"3840x2160", "3840x2160 (4K Dolby Vision)", "bt2020nc", "bt2020"},
		// HLG content
		{"3840x2160", "3840x2160 (4K HLG)", "bt2020nc", "bt2020"},
		// SDR content
		{"1920x1080", "1920x1080 (1080p)", "bt709", "bt709"},
		{"1920x1080", "1920x1080 (1080p)", "bt709", "bt709"},
		{"1280x720", "1280x720 (720p)", "bt709", "bt709"},
		{"1280x720", "1280x720 (720p)", "", ""},
	}

	for i, te := range testEvents {
		colorSpace := te.videoColorSpace
		colorPrimaries := te.videoColorPrimaries
		var colorSpacePtr, colorPrimariesPtr *string
		if colorSpace != "" {
			colorSpacePtr = &colorSpace
		}
		if colorPrimaries != "" {
			colorPrimariesPtr = &colorPrimaries
		}
		streamRes := te.streamVideoFullResolution
		videoRes := te.videoFullResolution
		event := &models.PlaybackEvent{
			ID:                        uuid.New(),
			SessionKey:                uuid.New().String(),
			StartedAt:                 now.Add(time.Duration(-i) * time.Hour),
			UserID:                    1,
			Username:                  "testuser",
			IPAddress:                 "192.168.1.1",
			MediaType:                 "movie",
			Title:                     "Test Movie",
			StreamVideoFullResolution: &streamRes,
			VideoFullResolution:       &videoRes,
			VideoColorSpace:           colorSpacePtr,
			VideoColorPrimaries:       colorPrimariesPtr,
			PercentComplete:           100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}
}

// TestGetHDRContentStats_Success tests HDR content statistics
func TestGetHDRContentStats_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertHDRTestData(t, db)

	stats, err := db.GetHDRContentStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetHDRContentStats failed: %v", err)
	}

	// Verify total sessions
	if stats.TotalSessions != 8 {
		t.Errorf("Expected 8 total sessions, got %d", stats.TotalSessions)
	}

	// Verify HDR sessions (4 with bt2020 color space)
	if stats.HDRSessions != 4 {
		t.Errorf("Expected 4 HDR sessions, got %d", stats.HDRSessions)
	}

	// Verify SDR sessions
	if stats.SDRSessions != 4 {
		t.Errorf("Expected 4 SDR sessions, got %d", stats.SDRSessions)
	}

	// Verify HDR percentage
	expectedHDRPct := 50.0 // 4/8 * 100
	if stats.HDRPercentage != expectedHDRPct {
		t.Errorf("Expected HDR percentage %.1f%%, got %.1f%%", expectedHDRPct, stats.HDRPercentage)
	}
}

// TestGetHDRContentStats_ColorSpaces tests color space distribution
func TestGetHDRContentStats_ColorSpaces(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertHDRTestData(t, db)

	stats, err := db.GetHDRContentStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetHDRContentStats failed: %v", err)
	}

	// Should have color space entries
	if len(stats.ColorSpaces) == 0 {
		t.Error("Expected non-empty color spaces")
	}

	// Verify bt2020nc is present
	foundBt2020 := false
	for _, cs := range stats.ColorSpaces {
		if cs.ColorSpace == "bt2020nc" {
			foundBt2020 = true
			if cs.SessionCount != 4 {
				t.Errorf("Expected 4 bt2020nc sessions, got %d", cs.SessionCount)
			}
		}
	}
	if !foundBt2020 {
		t.Error("Expected to find bt2020nc in color spaces")
	}
}

// TestGetHDRContentStats_EmptyDatabase tests empty database case
func TestGetHDRContentStats_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stats, err := db.GetHDRContentStats(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetHDRContentStats on empty DB failed: %v", err)
	}

	if stats.TotalSessions != 0 {
		t.Errorf("Expected 0 sessions on empty DB, got %d", stats.TotalSessions)
	}
}

// TestGetHDRContentStats_WithFilter tests filtering
func TestGetHDRContentStats_WithFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertHDRTestData(t, db)

	now := time.Now()
	startDate := now.Add(-4 * time.Hour)
	endDate := now

	stats, err := db.GetHDRContentStats(context.Background(), LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	})
	if err != nil {
		t.Fatalf("GetHDRContentStats with filter failed: %v", err)
	}

	// Should have fewer sessions with date filter
	if stats.TotalSessions >= 8 {
		t.Errorf("Expected fewer than 8 sessions with date filter, got %d", stats.TotalSessions)
	}
}

// TestGetHardwareTranscodeTrends_Success tests hardware transcode trends over time
func TestGetHardwareTranscodeTrends_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	now := time.Now()

	// Insert events spread over 7 days with varying HW usage
	for i := 0; i < 7; i++ {
		for j := 0; j < 5; j++ {
			hwEncoding := 0
			decision := "direct play"
			if j < 3 { // 3 out of 5 use HW encoding
				hwEncoding = 1
				decision = "transcode"
			}

			event := &models.PlaybackEvent{
				ID:                  uuid.New(),
				SessionKey:          uuid.New().String(),
				StartedAt:           now.Add(time.Duration(-i*24) * time.Hour),
				UserID:              1,
				Username:            "testuser",
				IPAddress:           "192.168.1.1",
				MediaType:           "movie",
				Title:               "Test Movie",
				TranscodeDecision:   &decision,
				TranscodeHWEncoding: &hwEncoding,
				PercentComplete:     100,
			}
			if err := db.InsertPlaybackEvent(event); err != nil {
				t.Fatalf("Failed to insert event: %v", err)
			}
		}
	}

	startDate := now.Add(-7 * 24 * time.Hour)
	endDate := now

	trends, err := db.GetHardwareTranscodeTrends(context.Background(), LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	})
	if err != nil {
		t.Fatalf("GetHardwareTranscodeTrends failed: %v", err)
	}

	// Should have trend data
	if len(trends) == 0 {
		t.Error("Expected non-empty trends data")
	}
}

// TestGetHardwareTranscodeTrends_EmptyDatabase tests empty database
func TestGetHardwareTranscodeTrends_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	startDate := now.Add(-7 * 24 * time.Hour)
	endDate := now

	trends, err := db.GetHardwareTranscodeTrends(context.Background(), LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	})
	if err != nil {
		t.Fatalf("GetHardwareTranscodeTrends on empty DB failed: %v", err)
	}

	if len(trends) != 0 {
		t.Errorf("Expected empty trends on empty DB, got %d entries", len(trends))
	}
}

// TestCalcPercentage tests the percentage calculation helper
func TestCalcPercentage(t *testing.T) {

	tests := []struct {
		name     string
		part     int
		total    int
		expected float64
	}{
		{"zero total", 5, 0, 0.0},
		{"zero part", 0, 10, 0.0},
		{"50 percent", 5, 10, 50.0},
		{"100 percent", 10, 10, 100.0},
		{"25 percent", 25, 100, 25.0},
		{"33.33 percent", 1, 3, 33.33333333333333},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcPercentage(tt.part, tt.total)
			if got != tt.expected {
				t.Errorf("calcPercentage(%d, %d) = %v, want %v", tt.part, tt.total, got, tt.expected)
			}
		})
	}
}
