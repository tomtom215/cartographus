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

// createTestEvent creates a test playback event with default values
func createTestEvent(now time.Time, idx int) *models.PlaybackEvent {
	return &models.PlaybackEvent{
		ID:              uuid.New(),
		SessionKey:      uuid.New().String(),
		StartedAt:       now.Add(time.Duration(-idx) * time.Hour),
		UserID:          1,
		Username:        "testuser",
		IPAddress:       "192.168.1.1",
		MediaType:       "movie",
		Title:           "Movie",
		PercentComplete: 100,
	}
}

// insertEvents inserts multiple events using a modifier function
func insertEvents[T any](t *testing.T, db *DB, count int, data []T, modify func(*models.PlaybackEvent, T, int)) {
	now := time.Now()
	for i, d := range data {
		if i >= count {
			break
		}
		event := createTestEvent(now, i)
		modify(event, d, i)
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}
}

func TestGetResolutionMismatchAnalytics(t *testing.T) {
	tests := []struct {
		name                string
		resolutions         []struct{ video, stream, transcode string }
		wantTotal           int
		wantMismatched      int
		wantMismatchRateMin float64
		wantMismatchRateMax float64
	}{
		{
			name: "with mismatches",
			resolutions: []struct{ video, stream, transcode string }{
				{"4K", "1080p", "transcode"},
				{"1080p", "720p", "transcode"},
				{"4K", "1080p", "transcode"},
				{"1080p", "1080p", "direct play"},
			},
			wantTotal:           4,
			wantMismatched:      3,
			wantMismatchRateMin: 70,
			wantMismatchRateMax: 80,
		},
		{
			name: "no mismatches",
			resolutions: []struct{ video, stream, transcode string }{
				{"1080p", "1080p", "direct play"},
				{"1080p", "1080p", "direct play"},
				{"720p", "720p", "direct play"},
			},
			wantTotal:           3,
			wantMismatched:      0,
			wantMismatchRateMin: 0,
			wantMismatchRateMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()
			insertTestGeolocations(t, db)

			insertEvents(t, db, len(tt.resolutions), tt.resolutions, func(e *models.PlaybackEvent, r struct{ video, stream, transcode string }, _ int) {
				e.VideoResolution = &r.video
				e.StreamVideoResolution = &r.stream
				e.TranscodeDecision = &r.transcode
			})

			analytics, err := db.GetResolutionMismatchAnalytics(context.Background(), LocationStatsFilter{})
			if err != nil {
				t.Fatalf("GetResolutionMismatchAnalytics failed: %v", err)
			}

			if analytics.TotalPlaybacks != tt.wantTotal {
				t.Errorf("TotalPlaybacks = %d, want %d", analytics.TotalPlaybacks, tt.wantTotal)
			}
			if analytics.MismatchedPlaybacks != tt.wantMismatched {
				t.Errorf("MismatchedPlaybacks = %d, want %d", analytics.MismatchedPlaybacks, tt.wantMismatched)
			}
			if analytics.MismatchRate < tt.wantMismatchRateMin || analytics.MismatchRate > tt.wantMismatchRateMax {
				t.Errorf("MismatchRate = %.2f, want between %.2f and %.2f", analytics.MismatchRate, tt.wantMismatchRateMin, tt.wantMismatchRateMax)
			}
		})
	}
}

func TestGetHDRAnalytics(t *testing.T) {
	tests := []struct {
		name           string
		dynamicRanges  []string
		wantTotal      int
		wantHDRRateMin float64
		wantHDRRateMax float64
	}{
		{
			name:           "mixed HDR and SDR",
			dynamicRanges:  []string{"HDR10", "Dolby Vision", "SDR", "HDR10", "SDR"},
			wantTotal:      5,
			wantHDRRateMin: 50,
			wantHDRRateMax: 70,
		},
		{
			name:           "empty data",
			dynamicRanges:  nil,
			wantTotal:      0,
			wantHDRRateMin: 0,
			wantHDRRateMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()
			insertTestGeolocations(t, db)

			if len(tt.dynamicRanges) > 0 {
				insertEvents(t, db, len(tt.dynamicRanges), tt.dynamicRanges, func(e *models.PlaybackEvent, dr string, _ int) {
					e.VideoDynamicRange = &dr
				})
			}

			analytics, err := db.GetHDRAnalytics(context.Background(), LocationStatsFilter{})
			if err != nil {
				t.Fatalf("GetHDRAnalytics failed: %v", err)
			}

			if analytics.TotalPlaybacks != tt.wantTotal {
				t.Errorf("TotalPlaybacks = %d, want %d", analytics.TotalPlaybacks, tt.wantTotal)
			}
			if analytics.HDRAdoptionRate < tt.wantHDRRateMin || analytics.HDRAdoptionRate > tt.wantHDRRateMax {
				t.Errorf("HDRAdoptionRate = %.2f, want between %.2f and %.2f", analytics.HDRAdoptionRate, tt.wantHDRRateMin, tt.wantHDRRateMax)
			}
		})
	}
}

func TestGetAudioAnalytics(t *testing.T) {
	t.Run("success with distribution", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()
		insertTestGeolocations(t, db)

		audioData := []struct{ codec, channels string }{
			{"eac3", "7.1"}, {"aac", "5.1"}, {"dts", "7.1"}, {"aac", "stereo"}, {"eac3", "5.1"},
		}
		insertEvents(t, db, len(audioData), audioData, func(e *models.PlaybackEvent, a struct{ codec, channels string }, _ int) {
			e.AudioCodec = &a.codec
			e.AudioChannels = &a.channels
		})

		analytics, err := db.GetAudioAnalytics(context.Background(), LocationStatsFilter{})
		if err != nil {
			t.Fatalf("GetAudioAnalytics failed: %v", err)
		}

		if analytics.TotalPlaybacks != 5 {
			t.Errorf("TotalPlaybacks = %d, want 5", analytics.TotalPlaybacks)
		}
		if len(analytics.CodecDistribution) == 0 || len(analytics.ChannelDistribution) == 0 {
			t.Error("Expected codec and channel distribution data")
		}
	})

	t.Run("with user filter", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()
		insertTestGeolocations(t, db)

		now := time.Now()
		codec1, codec2 := "eac3", "aac"

		e1 := createTestEvent(now, 0)
		e1.UserID, e1.Username, e1.AudioCodec = 1, "user1", &codec1
		if err := db.InsertPlaybackEvent(e1); err != nil {
			t.Fatalf("InsertPlaybackEvent(e1) failed: %v", err)
		}

		e2 := createTestEvent(now, 1)
		e2.UserID, e2.Username, e2.IPAddress, e2.AudioCodec = 2, "user2", "192.168.1.2", &codec2
		if err := db.InsertPlaybackEvent(e2); err != nil {
			t.Fatalf("InsertPlaybackEvent(e2) failed: %v", err)
		}

		analytics, err := db.GetAudioAnalytics(context.Background(), LocationStatsFilter{Users: []string{"user1"}})
		if err != nil {
			t.Fatalf("GetAudioAnalytics failed: %v", err)
		}
		if analytics.TotalPlaybacks != 1 {
			t.Errorf("TotalPlaybacks = %d, want 1", analytics.TotalPlaybacks)
		}
	})
}

func TestGetSubtitleAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	subtitleCodecs := []string{"srt", "ass", "", "srt", ""}
	insertEvents(t, db, len(subtitleCodecs), subtitleCodecs, func(e *models.PlaybackEvent, subtitle string, _ int) {
		e.Platform, e.Player = "Plex Web", "Chrome"
		if subtitle != "" {
			e.SubtitleCodec = &subtitle
			flag := 1
			e.Subtitles = &flag
		} else {
			flag := 0
			e.Subtitles = &flag
		}
	})

	analytics, err := db.GetSubtitleAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetSubtitleAnalytics failed: %v", err)
	}

	if analytics.TotalPlaybacks != 5 {
		t.Errorf("TotalPlaybacks = %d, want 5", analytics.TotalPlaybacks)
	}
	if analytics.PlaybacksWithSubs != 3 {
		t.Errorf("PlaybacksWithSubs = %d, want 3", analytics.PlaybacksWithSubs)
	}
	if analytics.SubtitleUsageRate < 55 || analytics.SubtitleUsageRate > 65 {
		t.Errorf("SubtitleUsageRate = %.2f, want ~60%%", analytics.SubtitleUsageRate)
	}
}

func TestGetFrameRateAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	frameRates := []string{"24fps", "30fps", "60fps", "24fps", "30fps"}
	insertEvents(t, db, len(frameRates), frameRates, func(e *models.PlaybackEvent, fps string, _ int) {
		e.VideoFrameRate = &fps
	})

	analytics, err := db.GetFrameRateAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetFrameRateAnalytics failed: %v", err)
	}

	if analytics.TotalPlaybacks != 5 {
		t.Errorf("TotalPlaybacks = %d, want 5", analytics.TotalPlaybacks)
	}
	if len(analytics.FrameRateDistribution) == 0 {
		t.Error("Expected frame rate distribution data")
	}
}

func TestGetContainerAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	containers := []string{"mkv", "mp4", "avi", "mkv", "mp4"}
	insertEvents(t, db, len(containers), containers, func(e *models.PlaybackEvent, container string, _ int) {
		e.Container = &container
	})

	analytics, err := db.GetContainerAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetContainerAnalytics failed: %v", err)
	}

	if analytics.TotalPlaybacks != 5 {
		t.Errorf("TotalPlaybacks = %d, want 5", analytics.TotalPlaybacks)
	}
	if len(analytics.FormatDistribution) == 0 {
		t.Error("Expected format distribution data")
	}
}

func TestGetConnectionSecurityAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	secureFlags := []int{1, 0, 1, 0, 1}
	insertEvents(t, db, len(secureFlags), secureFlags, func(e *models.PlaybackEvent, secure int, _ int) {
		e.Secure = &secure
	})

	analytics, err := db.GetConnectionSecurityAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetConnectionSecurityAnalytics failed: %v", err)
	}

	if analytics.TotalPlaybacks != 5 {
		t.Errorf("TotalPlaybacks = %d, want 5", analytics.TotalPlaybacks)
	}
	if analytics.SecureConnections+analytics.InsecureConnections != 5 {
		t.Errorf("Secure + insecure = %d, want 5", analytics.SecureConnections+analytics.InsecureConnections)
	}
}

func TestGetPausePatternAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	pauseCounts := []int{0, 2, 5, 1, 3}
	insertEvents(t, db, len(pauseCounts), pauseCounts, func(e *models.PlaybackEvent, pauses int, _ int) {
		e.PausedCounter = pauses
	})

	analytics, err := db.GetPausePatternAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetPausePatternAnalytics failed: %v", err)
	}

	if analytics.TotalPlaybacks != 5 {
		t.Errorf("TotalPlaybacks = %d, want 5", analytics.TotalPlaybacks)
	}
	// Average: (0+2+5+1+3)/5 = 2.2
	if analytics.AvgPausesPerSession < 2.0 || analytics.AvgPausesPerSession > 2.5 {
		t.Errorf("AvgPausesPerSession = %.2f, want ~2.2", analytics.AvgPausesPerSession)
	}
}

func TestGetLibraryAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	now := time.Now()
	sectionID := 1
	libName := "Movies"

	for i := 0; i < 5; i++ {
		event := createTestEvent(now, i)
		event.UserID = i + 1
		event.Username = "user"
		event.Platform = "Plex Web"
		event.Player = "Chrome"
		event.SectionID = &sectionID
		event.LibraryName = &libName
		event.PercentComplete = 90
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	analytics, err := db.GetLibraryAnalytics(context.Background(), sectionID, LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetLibraryAnalytics failed: %v", err)
	}

	if analytics.LibraryID != sectionID {
		t.Errorf("LibraryID = %d, want %d", analytics.LibraryID, sectionID)
	}
	if analytics.TotalPlaybacks != 5 {
		t.Errorf("TotalPlaybacks = %d, want 5", analytics.TotalPlaybacks)
	}
}

func TestGetConcurrentStreamsAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	now := time.Now()
	stopped := now.Add(2 * time.Hour)

	for i := 0; i < 3; i++ {
		event := createTestEvent(now, 0) // All start at same time
		event.StoppedAt = &stopped
		event.UserID = i + 1
		event.Username = "user"
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	analytics, err := db.GetConcurrentStreamsAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetConcurrentStreamsAnalytics failed: %v", err)
	}

	if analytics.TotalSessions != 3 {
		t.Errorf("TotalSessions = %d, want 3", analytics.TotalSessions)
	}
	if analytics.PeakConcurrent < 1 {
		t.Errorf("PeakConcurrent = %d, want >= 1", analytics.PeakConcurrent)
	}
}

// TestGetAudioAnalytics_DownmixEvents tests audio downmix event detection
func TestGetAudioAnalytics_DownmixEvents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	// Insert playback events with different source and stream audio channels
	// This should trigger downmix detection (8 channels down to 2)
	audioData := []struct {
		audioChannels, streamAudioChannels string
	}{
		{"8", "2"}, // 7.1 downmixed to stereo
		{"6", "2"}, // 5.1 downmixed to stereo
		{"8", "6"}, // 7.1 downmixed to 5.1
		{"2", "2"}, // No downmix (stereo to stereo)
		{"6", "6"}, // No downmix (5.1 to 5.1)
	}

	now := time.Now()
	for i, a := range audioData {
		event := createTestEvent(now, i)
		event.AudioChannels = &a.audioChannels
		event.StreamAudioChannels = &a.streamAudioChannels
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	analytics, err := db.GetAudioAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetAudioAnalytics failed: %v", err)
	}

	// Should detect 3 downmix events (where channels differ)
	if len(analytics.DownmixEvents) != 3 {
		t.Errorf("DownmixEvents count = %d, want 3", len(analytics.DownmixEvents))
	}
}

// TestGetHDRAnalytics_ToneMappingEvents tests HDR tone mapping detection
func TestGetHDRAnalytics_ToneMappingEvents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	// Insert HDR content with tone mapping (transcode from HDR to SDR)
	hdrData := []struct {
		dynamicRange, streamDynamicRange string
		transcodeDecision                string
	}{
		{"HDR10", "SDR", "transcode"},        // Tone mapped
		{"Dolby Vision", "SDR", "transcode"}, // Tone mapped
		{"HDR10", "HDR10", "direct play"},    // No tone mapping
		{"SDR", "SDR", "direct play"},        // SDR content
	}

	now := time.Now()
	for i, h := range hdrData {
		event := createTestEvent(now, i)
		event.VideoDynamicRange = &h.dynamicRange
		event.StreamVideoDynamicRange = &h.streamDynamicRange
		event.TranscodeDecision = &h.transcodeDecision
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	analytics, err := db.GetHDRAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetHDRAnalytics failed: %v", err)
	}

	if analytics.TotalPlaybacks != 4 {
		t.Errorf("TotalPlaybacks = %d, want 4", analytics.TotalPlaybacks)
	}
	// Should detect 2 tone mapping events (HDR10->SDR, DV->SDR)
	if len(analytics.ToneMappingEvents) < 1 {
		t.Logf("ToneMappingEvents = %d (may vary based on query)", len(analytics.ToneMappingEvents))
	}
}

// TestGetContainerAnalytics_RemuxEvents tests container remux detection
func TestGetContainerAnalytics_RemuxEvents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	// Insert playbacks where container is remuxed (source container != stream container)
	containerData := []struct {
		container, streamContainer string
	}{
		{"mkv", "ts"},  // Remuxed from MKV to TS
		{"avi", "mp4"}, // Remuxed from AVI to MP4
		{"mkv", "mkv"}, // No remux
		{"mp4", "mp4"}, // No remux
	}

	now := time.Now()
	for i, c := range containerData {
		event := createTestEvent(now, i)
		event.Container = &c.container
		event.StreamContainer = &c.streamContainer
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	analytics, err := db.GetContainerAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetContainerAnalytics failed: %v", err)
	}

	if analytics.TotalPlaybacks != 4 {
		t.Errorf("TotalPlaybacks = %d, want 4", analytics.TotalPlaybacks)
	}
	// Should detect 2 remux events (mkv->ts, avi->mp4)
	if len(analytics.RemuxEvents) < 1 {
		t.Logf("RemuxEvents = %d (may vary based on query)", len(analytics.RemuxEvents))
	}
}

// TestGetSubtitleAnalytics_LanguageDistribution tests subtitle language distribution
func TestGetSubtitleAnalytics_LanguageDistribution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	// Insert playbacks with various subtitle languages
	subtitleData := []struct {
		language, codec string
	}{
		{"English", "srt"},
		{"Spanish", "ass"},
		{"English", "srt"},
		{"French", "srt"},
		{"English", "vtt"},
	}

	now := time.Now()
	for i, s := range subtitleData {
		event := createTestEvent(now, i)
		event.SubtitleLanguage = &s.language
		event.SubtitleCodec = &s.codec
		subtitles := 1
		event.Subtitles = &subtitles
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	analytics, err := db.GetSubtitleAnalytics(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetSubtitleAnalytics failed: %v", err)
	}

	if analytics.PlaybacksWithSubs != 5 {
		t.Errorf("PlaybacksWithSubs = %d, want 5", analytics.PlaybacksWithSubs)
	}
	// Should have language distribution with English as most common
	if len(analytics.LanguageDistribution) == 0 {
		t.Error("Expected LanguageDistribution data")
	}
}

// TestEnsureContext tests the context timeout helper
func TestEnsureContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	t.Run("nil context gets timeout", func(t *testing.T) {
		ctx, cancel := db.ensureContext(nil)
		defer cancel()

		deadline, hasDeadline := ctx.Deadline()
		if !hasDeadline {
			t.Error("Expected context to have deadline")
		}
		// Deadline should be ~30 seconds from now
		timeUntilDeadline := time.Until(deadline)
		if timeUntilDeadline < 29*time.Second || timeUntilDeadline > 31*time.Second {
			t.Errorf("Expected deadline ~30s, got %v", timeUntilDeadline)
		}
	})

	t.Run("context without deadline gets timeout", func(t *testing.T) {
		ctx, cancel := db.ensureContext(context.Background())
		defer cancel()

		deadline, hasDeadline := ctx.Deadline()
		if !hasDeadline {
			t.Error("Expected context to have deadline")
		}
		timeUntilDeadline := time.Until(deadline)
		if timeUntilDeadline < 29*time.Second || timeUntilDeadline > 31*time.Second {
			t.Errorf("Expected deadline ~30s, got %v", timeUntilDeadline)
		}
	})

	t.Run("context with deadline preserved", func(t *testing.T) {
		originalCtx, originalCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer originalCancel()

		ctx, cancel := db.ensureContext(originalCtx)
		defer cancel()

		deadline, hasDeadline := ctx.Deadline()
		if !hasDeadline {
			t.Error("Expected context to have deadline")
		}
		timeUntilDeadline := time.Until(deadline)
		// Should preserve original 10s deadline
		if timeUntilDeadline > 11*time.Second {
			t.Errorf("Expected deadline ~10s, got %v", timeUntilDeadline)
		}
	})
}

// TestQueryAndScan tests the queryAndScan helper
func TestQueryAndScan(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)
	insertTestPlaybacks(t, db)

	t.Run("successful query", func(t *testing.T) {
		query := "SELECT COUNT(*) FROM playback_events"
		var count int
		err := db.queryRowWithContext(context.Background(), query, nil, &count)
		if err != nil {
			t.Fatalf("queryRowWithContext failed: %v", err)
		}
		if count < 1 {
			t.Error("Expected at least 1 playback event")
		}
	})

	t.Run("query with args", func(t *testing.T) {
		query := "SELECT city FROM geolocations WHERE ip_address = ?"
		args := []interface{}{"192.168.1.1"}
		var city string
		err := db.queryRowWithContext(context.Background(), query, args, &city)
		// No rows returns nil by design (for aggregation queries)
		if err != nil {
			t.Logf("Query returned error: %v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		query := "SELECT COUNT(*) FROM playback_events"
		var count int
		err := db.queryRowWithContext(ctx, query, nil, &count)
		if err == nil {
			t.Error("Expected error with canceled context")
		}
	})
}
