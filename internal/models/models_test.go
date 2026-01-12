// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// testJSONRoundTrip is a generic helper that tests JSON marshal/unmarshal for any type.
// It marshals the input, unmarshals it back, and calls the verification function.
func testJSONRoundTrip[T any](t *testing.T, name string, input T, verify func(t *testing.T, decoded T)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		data, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("Failed to marshal %s: %v", name, err)
		}

		var decoded T
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal %s: %v", name, err)
		}

		if verify != nil {
			verify(t, decoded)
		}
	})
}

// Test fixtures - reusable test data
var (
	testTime       = time.Now()
	testUUID       = uuid.New()
	testCity       = "Philadelphia"
	testRegion     = "Pennsylvania"
	testPostalCode = "19102"
	testTimezone   = "America/New_York"
	testRadius     = 10
)

func TestJSONMarshaling(t *testing.T) {
	t.Parallel()

	// PlaybackEvent with populated fields
	testJSONRoundTrip(t, "PlaybackEvent", createTestPlaybackEvent(), func(t *testing.T, decoded PlaybackEvent) {
		if decoded.ID != testUUID {
			t.Errorf("Expected ID %v, got %v", testUUID, decoded.ID)
		}
		if decoded.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", decoded.Username)
		}
		if decoded.PercentComplete != 75 {
			t.Errorf("Expected percent complete 75, got %d", decoded.PercentComplete)
		}
		if decoded.ParentTitle == nil || *decoded.ParentTitle != "Season 1" {
			t.Error("ParentTitle not properly marshaled/unmarshaled")
		}
	})

	// Geolocation
	testJSONRoundTrip(t, "Geolocation", createTestGeolocation(), func(t *testing.T, decoded Geolocation) {
		if decoded.IPAddress != "192.168.1.100" {
			t.Errorf("Expected IP 192.168.1.100, got %s", decoded.IPAddress)
		}
		if decoded.Latitude != 39.9526 {
			t.Errorf("Expected latitude 39.9526, got %f", decoded.Latitude)
		}
		if decoded.City == nil || *decoded.City != "Philadelphia" {
			t.Error("City not properly marshaled/unmarshaled")
		}
	})

	// APIResponse
	testJSONRoundTrip(t, "APIResponse", createTestAPIResponse(), func(t *testing.T, decoded APIResponse) {
		if decoded.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", decoded.Status)
		}
		if decoded.Error != nil {
			t.Error("Expected error to be nil")
		}
	})

	// APIError
	testJSONRoundTrip(t, "APIError", createTestAPIError(), func(t *testing.T, decoded APIError) {
		if decoded.Code != "VALIDATION_ERROR" {
			t.Errorf("Expected code 'VALIDATION_ERROR', got '%s'", decoded.Code)
		}
		if decoded.Message != "Invalid input" {
			t.Errorf("Expected message 'Invalid input', got '%s'", decoded.Message)
		}
	})

	// LoginRequest
	testJSONRoundTrip(t, "LoginRequest", LoginRequest{Username: "admin", Password: "secret123", RememberMe: true}, func(t *testing.T, decoded LoginRequest) {
		if decoded.Username != "admin" {
			t.Errorf("Expected username 'admin', got '%s'", decoded.Username)
		}
		if !decoded.RememberMe {
			t.Error("Expected RememberMe to be true")
		}
	})

	// Stats
	testJSONRoundTrip(t, "Stats", createTestStats(), func(t *testing.T, decoded Stats) {
		if decoded.TotalPlaybacks != 1000 {
			t.Errorf("Expected 1000 playbacks, got %d", decoded.TotalPlaybacks)
		}
		if len(decoded.TopCountries) != 1 {
			t.Errorf("Expected 1 top country, got %d", len(decoded.TopCountries))
		}
	})

	// TautulliHistory
	testJSONRoundTrip(t, "TautulliHistory", createTestTautulliHistory(), func(t *testing.T, decoded tautulli.TautulliHistory) {
		if decoded.Response.Result != "success" {
			t.Errorf("Expected result 'success', got '%s'", decoded.Response.Result)
		}
		if len(decoded.Response.Data.Data) != 1 {
			t.Errorf("Expected 1 record, got %d", len(decoded.Response.Data.Data))
		}
	})

	// TautulliGeoIP
	testJSONRoundTrip(t, "TautulliGeoIP", createTestTautulliGeoIP(), func(t *testing.T, decoded tautulli.TautulliGeoIP) {
		if decoded.Response.Data.City != "Philadelphia" {
			t.Errorf("Expected city 'Philadelphia', got '%s'", decoded.Response.Data.City)
		}
		if decoded.Response.Data.Latitude != 39.9526 {
			t.Errorf("Expected latitude 39.9526, got %f", decoded.Response.Data.Latitude)
		}
	})

	// BingeAnalytics
	testJSONRoundTrip(t, "BingeAnalytics", createTestBingeAnalytics(), func(t *testing.T, decoded BingeAnalytics) {
		if decoded.TotalBingeSessions != 50 {
			t.Errorf("Expected 50 binge sessions, got %d", decoded.TotalBingeSessions)
		}
		if decoded.AvgEpisodesPerBinge != 6.0 {
			t.Errorf("Expected 6.0 avg episodes, got %f", decoded.AvgEpisodesPerBinge)
		}
	})

	// PopularContent
	testJSONRoundTrip(t, "PopularContent", createTestPopularContent(), func(t *testing.T, decoded PopularContent) {
		if decoded.PlaybackCount != 100 {
			t.Errorf("Expected 100 playbacks, got %d", decoded.PlaybackCount)
		}
		if decoded.AvgCompletion != 85.5 {
			t.Errorf("Expected 85.5 avg completion, got %f", decoded.AvgCompletion)
		}
	})

	// WatchParty
	testJSONRoundTrip(t, "WatchParty", createTestWatchParty(), func(t *testing.T, decoded WatchParty) {
		if decoded.ParticipantCount != 4 {
			t.Errorf("Expected 4 participants, got %d", decoded.ParticipantCount)
		}
		if !decoded.SameLocation {
			t.Error("Expected SameLocation to be true")
		}
	})

	// UserEngagement
	testJSONRoundTrip(t, "UserEngagement", createTestUserEngagement(), func(t *testing.T, decoded UserEngagement) {
		if decoded.TotalSessions != 100 {
			t.Errorf("Expected 100 sessions, got %d", decoded.TotalSessions)
		}
		if decoded.ActivityScore != 85.5 {
			t.Errorf("Expected activity score 85.5, got %f", decoded.ActivityScore)
		}
	})

	// ComparativeAnalytics
	testJSONRoundTrip(t, "ComparativeAnalytics", createTestComparativeAnalytics(), func(t *testing.T, decoded ComparativeAnalytics) {
		if decoded.ComparisonType != "week" {
			t.Errorf("Expected comparison type 'week', got '%s'", decoded.ComparisonType)
		}
		if decoded.CurrentPeriod.PlaybackCount != 100 {
			t.Errorf("Expected 100 playbacks, got %d", decoded.CurrentPeriod.PlaybackCount)
		}
	})

	// TemporalHeatmapResponse
	testJSONRoundTrip(t, "TemporalHeatmapResponse", createTestTemporalHeatmap(), func(t *testing.T, decoded TemporalHeatmapResponse) {
		if decoded.Interval != "hour" {
			t.Errorf("Expected interval 'hour', got '%s'", decoded.Interval)
		}
		if len(decoded.Buckets) != 1 {
			t.Errorf("Expected 1 bucket, got %d", len(decoded.Buckets))
		}
	})

	// HealthStatus
	testJSONRoundTrip(t, "HealthStatus", createTestHealthStatus(), func(t *testing.T, decoded HealthStatus) {
		if decoded.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", decoded.Status)
		}
		if !decoded.DatabaseConnected {
			t.Error("Expected DatabaseConnected to be true")
		}
	})

	// BandwidthAnalytics
	testJSONRoundTrip(t, "BandwidthAnalytics", createTestBandwidthAnalytics(), func(t *testing.T, decoded BandwidthAnalytics) {
		if decoded.TotalBandwidthGB != 1000.5 {
			t.Errorf("Expected 1000.5 GB, got %f", decoded.TotalBandwidthGB)
		}
		if decoded.AvgBandwidthMbps != 50.5 {
			t.Errorf("Expected 50.5 Mbps, got %f", decoded.AvgBandwidthMbps)
		}
	})
}

// TestPlaybackEvent_NilPointers verifies that nil pointer fields marshal/unmarshal correctly
func TestPlaybackEvent_NilPointers(t *testing.T) {
	t.Parallel()

	event := PlaybackEvent{
		ID:              testUUID,
		SessionKey:      "test",
		Username:        "user",
		Title:           "Title",
		MediaType:       "movie",
		StartedAt:       testTime,
		CreatedAt:       testTime,
		UserID:          1,
		IPAddress:       "127.0.0.1",
		Platform:        "web",
		Player:          "Chrome",
		PercentComplete: 50,
		// All pointer fields are nil by default
	}

	testJSONRoundTrip(t, "NilPointers", event, func(t *testing.T, decoded PlaybackEvent) {
		if decoded.ParentTitle != nil {
			t.Error("Expected ParentTitle to be nil")
		}
		if decoded.StoppedAt != nil {
			t.Error("Expected StoppedAt to be nil")
		}
	})
}

// Factory functions for creating test fixtures

func createTestPlaybackEvent() PlaybackEvent {
	parentTitle := "Season 1"
	grandparentTitle := "Show Name"
	return PlaybackEvent{
		ID:               testUUID,
		SessionKey:       "test-session-123",
		StartedAt:        testTime,
		UserID:           42,
		Username:         "testuser",
		IPAddress:        "192.168.1.100",
		MediaType:        "episode",
		Title:            "Episode 1",
		ParentTitle:      &parentTitle,
		GrandparentTitle: &grandparentTitle,
		Platform:         "web",
		Player:           "Chrome",
		LocationType:     "wan",
		PercentComplete:  75,
		PausedCounter:    2,
		CreatedAt:        testTime,
	}
}

func createTestGeolocation() Geolocation {
	return Geolocation{
		IPAddress:      "192.168.1.100",
		Latitude:       39.9526,
		Longitude:      -75.1652,
		City:           &testCity,
		Region:         &testRegion,
		Country:        "United States",
		PostalCode:     &testPostalCode,
		Timezone:       &testTimezone,
		AccuracyRadius: &testRadius,
		LastUpdated:    testTime,
	}
}

func createTestAPIResponse() APIResponse {
	return APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"count": 42,
			"items": []string{"item1", "item2"},
		},
		Metadata: Metadata{Timestamp: testTime, QueryTimeMS: 23},
		Error:    nil,
	}
}

func createTestAPIError() APIError {
	return APIError{
		Code:    "VALIDATION_ERROR",
		Message: "Invalid input",
		Details: map[string]interface{}{"field": "username", "constraint": "min_length"},
	}
}

func createTestStats() Stats {
	lastSync := testTime
	return Stats{
		TotalPlaybacks:    1000,
		UniqueLocations:   50,
		UniqueUsers:       25,
		TopCountries:      []CountryStats{{Country: "US", PlaybackCount: 500, UniqueUsers: 15}},
		RecentActivity:    10,
		LastSyncTime:      &lastSync,
		DatabaseSizeBytes: 1024000,
	}
}

func createTestTautulliHistory() tautulli.TautulliHistory {
	return tautulli.TautulliHistory{
		Response: tautulli.TautulliHistoryResponse{
			Result:  "success",
			Message: nil,
			Data: tautulli.TautulliHistoryData{
				RecordsFiltered: 10,
				RecordsTotal:    100,
				Data: []tautulli.TautulliHistoryRecord{
					{SessionKey: func() *string { s := "session-1"; return &s }(), Date: 1700000000, UserID: func() *int { i := 1; return &i }(), User: "testuser", MediaType: "movie", Title: "Test Movie", Platform: "web", Player: "Chrome"},
				},
			},
		},
	}
}

func createTestTautulliGeoIP() tautulli.TautulliGeoIP {
	return tautulli.TautulliGeoIP{
		Response: tautulli.TautulliGeoIPResponse{
			Result:  "success",
			Message: nil,
			Data: tautulli.TautulliGeoIPData{
				City: "Philadelphia", Region: "Pennsylvania", Country: "United States",
				PostalCode: "19102", Timezone: "America/New_York",
				Latitude: 39.9526, Longitude: -75.1652, AccuracyRadius: 10,
			},
		},
	}
}

func createTestBingeAnalytics() BingeAnalytics {
	return BingeAnalytics{
		TotalBingeSessions:  50,
		TotalEpisodesBinged: 300,
		AvgEpisodesPerBinge: 6.0,
		AvgBingeDuration:    180.5,
		TopBingeShows:       []BingeShowStats{{ShowName: "Test Show", BingeCount: 10, TotalEpisodes: 60, UniqueWatchers: 5, AvgEpisodes: 6.0}},
		TopBingeWatchers:    []BingeUserStats{},
		RecentBingeSessions: []BingeSession{},
		BingesByDay:         []BingesByDayOfWeek{},
	}
}

func createTestPopularContent() PopularContent {
	parent := "Season 1"
	year := 2023
	rating := "PG-13"
	return PopularContent{
		MediaType: "episode", Title: "Test Episode", ParentTitle: &parent, GrandparentTitle: nil,
		PlaybackCount: 100, UniqueUsers: 25, AvgCompletion: 85.5,
		FirstPlayed: testTime.Add(-30 * 24 * time.Hour), LastPlayed: testTime,
		Year: &year, ContentRating: &rating, TotalWatchTime: 500,
	}
}

func createTestWatchParty() WatchParty {
	location := "Philadelphia"
	return WatchParty{
		MediaType: "movie", Title: "Test Movie", PartyTime: testTime, ParticipantCount: 4,
		Participants: []WatchPartyParticipant{
			{UserID: 1, Username: "user1", IPAddress: "192.168.1.100", StartedAt: testTime, PercentComplete: 50},
		},
		SameLocation: true, LocationName: &location, AvgCompletion: 75.5, TotalDuration: 120,
	}
}

func createTestUserEngagement() UserEngagement {
	mostWatchedType := "movie"
	mostWatchedTitle := "Favorite Movie"
	return UserEngagement{
		UserID: 42, Username: "testuser",
		TotalWatchTimeMinutes: 5000.5, TotalSessions: 100, AverageSessionMinutes: 50.0,
		FirstSeenAt: testTime.Add(-365 * 24 * time.Hour), LastSeenAt: testTime,
		DaysSinceFirstSeen: 365, DaysSinceLastSeen: 0,
		TotalContentItems: 200, UniqueContentItems: 150,
		MostWatchedType: &mostWatchedType, MostWatchedTitle: &mostWatchedTitle,
		ActivityScore: 85.5, AvgCompletionRate: 90.0, FullyWatchedCount: 140,
		ReturnVisitorRate: 95.0, UniqueLocations: 3, UniquePlatforms: 2,
	}
}

func createTestComparativeAnalytics() ComparativeAnalytics {
	return ComparativeAnalytics{
		CurrentPeriod:  PeriodMetrics{StartDate: testTime.Add(-7 * 24 * time.Hour), EndDate: testTime, PlaybackCount: 100, UniqueUsers: 25, WatchTimeMinutes: 5000.0},
		PreviousPeriod: PeriodMetrics{StartDate: testTime.Add(-14 * 24 * time.Hour), EndDate: testTime.Add(-7 * 24 * time.Hour), PlaybackCount: 80, UniqueUsers: 20, WatchTimeMinutes: 4000.0},
		ComparisonType: "week",
		MetricsComparison: []ComparativeMetrics{
			{Metric: "playback_count", CurrentValue: 100, PreviousValue: 80, AbsoluteChange: 20, PercentageChange: 25.0, GrowthDirection: "up", IsImprovement: true},
		},
		TopContentComparison: []TopContentComparison{},
		TopUserComparison:    []TopUserComparison{},
		OverallTrend:         "growing",
		KeyInsights:          []string{"Playback count increased by 25%"},
	}
}

func createTestTemporalHeatmap() TemporalHeatmapResponse {
	return TemporalHeatmapResponse{
		Interval: "hour", TotalCount: 1000,
		StartDate: testTime.Add(-24 * time.Hour), EndDate: testTime,
		Buckets: []TemporalHeatmapBucket{
			{StartTime: testTime.Add(-2 * time.Hour), EndTime: testTime.Add(-1 * time.Hour), Label: "2PM - 3PM", Count: 50,
				Points: []TemporalHeatmapPoint{{Latitude: 39.9526, Longitude: -75.1652, Weight: 10}}},
		},
	}
}

func createTestHealthStatus() HealthStatus {
	lastSync := testTime.Add(-5 * time.Minute)
	return HealthStatus{
		Status: "healthy", Version: "1.0.0", DatabaseConnected: true, TautulliConnected: true,
		LastSyncTime: &lastSync, Uptime: 3600.5,
	}
}

func createTestBandwidthAnalytics() BandwidthAnalytics {
	return BandwidthAnalytics{
		TotalBandwidthGB: 1000.5, DirectPlayBandwidthGB: 600.0, TranscodeBandwidthGB: 400.5,
		AvgBandwidthMbps: 50.5, PeakBandwidthMbps: 100.0,
		ByTranscode: []BandwidthByTranscode{}, ByResolution: []BandwidthByResolution{},
		ByCodec: []BandwidthByCodec{}, Trends: []BandwidthTrend{}, TopUsers: []BandwidthByUser{},
	}
}

// =============================================================================
// BOUNDARY AND EDGE CASE TESTS
// =============================================================================

// TestPlaybackEvent_BoundaryValues tests numeric boundary conditions
func TestPlaybackEvent_BoundaryValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		event  PlaybackEvent
		verify func(t *testing.T, decoded PlaybackEvent)
	}{
		{
			name: "MaxInt values",
			event: PlaybackEvent{
				ID:              testUUID,
				SessionKey:      "max-int-test",
				Username:        "user",
				Title:           "Title",
				MediaType:       "movie",
				StartedAt:       testTime,
				CreatedAt:       testTime,
				UserID:          1<<31 - 1, // Max int32 value
				IPAddress:       "127.0.0.1",
				Platform:        "web",
				Player:          "Chrome",
				LocationType:    "wan",
				PercentComplete: 100,
				PausedCounter:   1<<31 - 1,
			},
			verify: func(t *testing.T, decoded PlaybackEvent) {
				if decoded.UserID != 1<<31-1 {
					t.Errorf("Expected UserID %d, got %d", 1<<31-1, decoded.UserID)
				}
				if decoded.PausedCounter != 1<<31-1 {
					t.Errorf("Expected PausedCounter %d, got %d", 1<<31-1, decoded.PausedCounter)
				}
			},
		},
		{
			name: "Zero values",
			event: PlaybackEvent{
				ID:              testUUID,
				SessionKey:      "zero-test",
				Username:        "user",
				Title:           "Title",
				MediaType:       "movie",
				StartedAt:       testTime,
				CreatedAt:       testTime,
				UserID:          0,
				IPAddress:       "127.0.0.1",
				Platform:        "web",
				Player:          "Chrome",
				LocationType:    "wan",
				PercentComplete: 0,
				PausedCounter:   0,
			},
			verify: func(t *testing.T, decoded PlaybackEvent) {
				if decoded.UserID != 0 {
					t.Errorf("Expected UserID 0, got %d", decoded.UserID)
				}
				if decoded.PercentComplete != 0 {
					t.Errorf("Expected PercentComplete 0, got %d", decoded.PercentComplete)
				}
			},
		},
		{
			name: "Negative values where allowed",
			event: PlaybackEvent{
				ID:              testUUID,
				SessionKey:      "negative-test",
				Username:        "user",
				Title:           "Title",
				MediaType:       "movie",
				StartedAt:       testTime,
				CreatedAt:       testTime,
				UserID:          -1, // Some systems use -1 for special users
				IPAddress:       "127.0.0.1",
				Platform:        "web",
				Player:          "Chrome",
				LocationType:    "wan",
				PercentComplete: 0,
				PausedCounter:   0,
			},
			verify: func(t *testing.T, decoded PlaybackEvent) {
				if decoded.UserID != -1 {
					t.Errorf("Expected UserID -1, got %d", decoded.UserID)
				}
			},
		},
		{
			name: "MaxInt64 for file size",
			event: func() PlaybackEvent {
				maxSize := int64(1<<63 - 1)
				return PlaybackEvent{
					ID:              testUUID,
					SessionKey:      "max-filesize-test",
					Username:        "user",
					Title:           "Title",
					MediaType:       "movie",
					StartedAt:       testTime,
					CreatedAt:       testTime,
					UserID:          1,
					IPAddress:       "127.0.0.1",
					Platform:        "web",
					Player:          "Chrome",
					LocationType:    "wan",
					PercentComplete: 50,
					FileSize:        &maxSize,
				}
			}(),
			verify: func(t *testing.T, decoded PlaybackEvent) {
				if decoded.FileSize == nil {
					t.Fatal("Expected FileSize to be non-nil")
				}
				if *decoded.FileSize != 1<<63-1 {
					t.Errorf("Expected FileSize %d, got %d", int64(1<<63-1), *decoded.FileSize)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.event)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded PlaybackEvent
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			tc.verify(t, decoded)
		})
	}
}

// TestPlaybackEvent_EmptyStrings tests empty string handling
func TestPlaybackEvent_EmptyStrings(t *testing.T) {
	t.Parallel()

	event := PlaybackEvent{
		ID:              testUUID,
		SessionKey:      "",
		Username:        "",
		Title:           "",
		MediaType:       "",
		StartedAt:       testTime,
		CreatedAt:       testTime,
		UserID:          1,
		IPAddress:       "",
		Platform:        "",
		Player:          "",
		LocationType:    "",
		PercentComplete: 0,
	}

	testJSONRoundTrip(t, "EmptyStrings", event, func(t *testing.T, decoded PlaybackEvent) {
		if decoded.SessionKey != "" {
			t.Errorf("Expected empty SessionKey, got '%s'", decoded.SessionKey)
		}
		if decoded.Username != "" {
			t.Errorf("Expected empty Username, got '%s'", decoded.Username)
		}
		if decoded.Title != "" {
			t.Errorf("Expected empty Title, got '%s'", decoded.Title)
		}
	})
}

// TestPlaybackEvent_SpecialCharacters tests special character handling in strings
func TestPlaybackEvent_SpecialCharacters(t *testing.T) {
	t.Parallel()

	specialChars := "Test <script>alert('xss')</script> & \"quotes\" 'apostrophe' \n\t\r unicode: \u0000\u001F"
	event := PlaybackEvent{
		ID:              testUUID,
		SessionKey:      specialChars,
		Username:        specialChars,
		Title:           specialChars,
		MediaType:       "movie",
		StartedAt:       testTime,
		CreatedAt:       testTime,
		UserID:          1,
		IPAddress:       "127.0.0.1",
		Platform:        "web",
		Player:          "Chrome",
		LocationType:    "wan",
		PercentComplete: 50,
	}

	testJSONRoundTrip(t, "SpecialCharacters", event, func(t *testing.T, decoded PlaybackEvent) {
		if decoded.Title != specialChars {
			t.Errorf("Expected Title '%s', got '%s'", specialChars, decoded.Title)
		}
	})
}

// TestPlaybackEvent_UnicodeStrings tests Unicode string handling
func TestPlaybackEvent_UnicodeStrings(t *testing.T) {
	t.Parallel()

	unicodeTitle := "日本語タイトル 中文标题 한국어 제목 Ελληνικά العربية עברית"
	event := PlaybackEvent{
		ID:              testUUID,
		SessionKey:      "unicode-test",
		Username:        "用户名",
		Title:           unicodeTitle,
		MediaType:       "movie",
		StartedAt:       testTime,
		CreatedAt:       testTime,
		UserID:          1,
		IPAddress:       "127.0.0.1",
		Platform:        "web",
		Player:          "Chrome",
		LocationType:    "wan",
		PercentComplete: 50,
	}

	testJSONRoundTrip(t, "UnicodeStrings", event, func(t *testing.T, decoded PlaybackEvent) {
		if decoded.Title != unicodeTitle {
			t.Errorf("Expected Title '%s', got '%s'", unicodeTitle, decoded.Title)
		}
		if decoded.Username != "用户名" {
			t.Errorf("Expected Username '用户名', got '%s'", decoded.Username)
		}
	})
}

// =============================================================================
// DISTRIBUTION MODEL TESTS
// =============================================================================

func TestDistributionModels(t *testing.T) {
	t.Parallel()

	// PlatformStats
	testJSONRoundTrip(t, "PlatformStats", PlatformStats{
		Platform:      "iOS",
		PlaybackCount: 500,
		UniqueUsers:   50,
	}, func(t *testing.T, decoded PlatformStats) {
		if decoded.Platform != "iOS" {
			t.Errorf("Expected Platform 'iOS', got '%s'", decoded.Platform)
		}
		if decoded.PlaybackCount != 500 {
			t.Errorf("Expected PlaybackCount 500, got %d", decoded.PlaybackCount)
		}
	})

	// PlayerStats
	testJSONRoundTrip(t, "PlayerStats", PlayerStats{
		Player:        "Plex Web",
		PlaybackCount: 300,
		UniqueUsers:   30,
	}, func(t *testing.T, decoded PlayerStats) {
		if decoded.Player != "Plex Web" {
			t.Errorf("Expected Player 'Plex Web', got '%s'", decoded.Player)
		}
	})

	// CompletionBucket
	testJSONRoundTrip(t, "CompletionBucket", CompletionBucket{
		Bucket:        "90-100%",
		MinPercent:    90,
		MaxPercent:    100,
		PlaybackCount: 150,
		AvgCompletion: 95.5,
	}, func(t *testing.T, decoded CompletionBucket) {
		if decoded.Bucket != "90-100%" {
			t.Errorf("Expected Bucket '90-100%%', got '%s'", decoded.Bucket)
		}
		if decoded.MinPercent != 90 {
			t.Errorf("Expected MinPercent 90, got %d", decoded.MinPercent)
		}
	})

	// ContentCompletionStats
	testJSONRoundTrip(t, "ContentCompletionStats", ContentCompletionStats{
		Buckets: []CompletionBucket{
			{Bucket: "0-25%", MinPercent: 0, MaxPercent: 25, PlaybackCount: 10, AvgCompletion: 12.5},
		},
		TotalPlaybacks:  100,
		AvgCompletion:   75.0,
		FullyWatched:    60,
		FullyWatchedPct: 60.0,
	}, func(t *testing.T, decoded ContentCompletionStats) {
		if len(decoded.Buckets) != 1 {
			t.Errorf("Expected 1 bucket, got %d", len(decoded.Buckets))
		}
		if decoded.TotalPlaybacks != 100 {
			t.Errorf("Expected TotalPlaybacks 100, got %d", decoded.TotalPlaybacks)
		}
	})

	// TranscodeStats
	testJSONRoundTrip(t, "TranscodeStats", TranscodeStats{
		TranscodeDecision: "transcode",
		PlaybackCount:     200,
		Percentage:        40.0,
	}, func(t *testing.T, decoded TranscodeStats) {
		if decoded.TranscodeDecision != "transcode" {
			t.Errorf("Expected TranscodeDecision 'transcode', got '%s'", decoded.TranscodeDecision)
		}
	})

	// ResolutionStats
	testJSONRoundTrip(t, "ResolutionStats", ResolutionStats{
		VideoResolution: "1080p",
		PlaybackCount:   400,
		Percentage:      80.0,
	}, func(t *testing.T, decoded ResolutionStats) {
		if decoded.VideoResolution != "1080p" {
			t.Errorf("Expected VideoResolution '1080p', got '%s'", decoded.VideoResolution)
		}
	})

	// CodecStats
	testJSONRoundTrip(t, "CodecStats", CodecStats{
		VideoCodec:    "h264",
		AudioCodec:    "aac",
		PlaybackCount: 350,
		Percentage:    70.0,
	}, func(t *testing.T, decoded CodecStats) {
		if decoded.VideoCodec != "h264" {
			t.Errorf("Expected VideoCodec 'h264', got '%s'", decoded.VideoCodec)
		}
		if decoded.AudioCodec != "aac" {
			t.Errorf("Expected AudioCodec 'aac', got '%s'", decoded.AudioCodec)
		}
	})

	// LibraryStats
	testJSONRoundTrip(t, "LibraryStats", LibraryStats{
		SectionID:     1,
		LibraryName:   "Movies",
		PlaybackCount: 250,
		UniqueUsers:   25,
		TotalDuration: 15000,
		AvgCompletion: 85.5,
	}, func(t *testing.T, decoded LibraryStats) {
		if decoded.LibraryName != "Movies" {
			t.Errorf("Expected LibraryName 'Movies', got '%s'", decoded.LibraryName)
		}
		if decoded.SectionID != 1 {
			t.Errorf("Expected SectionID 1, got %d", decoded.SectionID)
		}
	})

	// RatingStats
	testJSONRoundTrip(t, "RatingStats", RatingStats{
		ContentRating: "PG-13",
		PlaybackCount: 180,
		Percentage:    36.0,
	}, func(t *testing.T, decoded RatingStats) {
		if decoded.ContentRating != "PG-13" {
			t.Errorf("Expected ContentRating 'PG-13', got '%s'", decoded.ContentRating)
		}
	})

	// DurationStats
	testJSONRoundTrip(t, "DurationStats", DurationStats{
		AvgDuration:      90,
		MedianDuration:   85,
		TotalDuration:    45000,
		AvgCompletion:    82.5,
		FullyWatched:     300,
		FullyWatchedPct:  60.0,
		PartiallyWatched: 200,
		DurationByType: []DurationByMediaType{
			{MediaType: "movie", AvgDuration: 120, TotalDuration: 24000, PlaybackCount: 200, AvgCompletion: 85.0},
		},
	}, func(t *testing.T, decoded DurationStats) {
		if decoded.AvgDuration != 90 {
			t.Errorf("Expected AvgDuration 90, got %d", decoded.AvgDuration)
		}
		if len(decoded.DurationByType) != 1 {
			t.Errorf("Expected 1 DurationByType, got %d", len(decoded.DurationByType))
		}
	})

	// YearStats
	testJSONRoundTrip(t, "YearStats", YearStats{
		Year:          2024,
		PlaybackCount: 500,
	}, func(t *testing.T, decoded YearStats) {
		if decoded.Year != 2024 {
			t.Errorf("Expected Year 2024, got %d", decoded.Year)
		}
	})
}

// =============================================================================
// USER MAPPING MODEL TESTS
// =============================================================================

func TestUserMappingModels(t *testing.T) {
	t.Parallel()

	username := "testuser"
	friendlyName := "Test User"
	email := "test@example.com"
	thumb := "https://example.com/avatar.jpg"

	// UserMapping with all fields
	testJSONRoundTrip(t, "UserMapping_Full", UserMapping{
		ID:             1,
		Source:         "plex",
		ServerID:       "server-123",
		ExternalUserID: "ext-456",
		InternalUserID: 42,
		Username:       &username,
		FriendlyName:   &friendlyName,
		Email:          &email,
		UserThumb:      &thumb,
		CreatedAt:      testTime,
		UpdatedAt:      testTime,
	}, func(t *testing.T, decoded UserMapping) {
		if decoded.ID != 1 {
			t.Errorf("Expected ID 1, got %d", decoded.ID)
		}
		if decoded.Source != "plex" {
			t.Errorf("Expected Source 'plex', got '%s'", decoded.Source)
		}
		if decoded.Username == nil || *decoded.Username != "testuser" {
			t.Error("Username not properly marshaled/unmarshaled")
		}
	})

	// UserMapping with nil optional fields
	testJSONRoundTrip(t, "UserMapping_Minimal", UserMapping{
		ID:             2,
		Source:         "jellyfin",
		ServerID:       "server-789",
		ExternalUserID: "uuid-abc-def",
		InternalUserID: 100,
		CreatedAt:      testTime,
		UpdatedAt:      testTime,
	}, func(t *testing.T, decoded UserMapping) {
		if decoded.Username != nil {
			t.Error("Expected Username to be nil")
		}
		if decoded.Email != nil {
			t.Error("Expected Email to be nil")
		}
	})

	// UserMappingStats
	lastCreated := testTime.Add(-1 * time.Hour)
	lastUpdated := testTime
	testJSONRoundTrip(t, "UserMappingStats", UserMappingStats{
		TotalMappings:     100,
		BySource:          map[string]int{"plex": 50, "jellyfin": 30, "emby": 20},
		ByServer:          map[string]int{"server-1": 60, "server-2": 40},
		UniqueInternalIDs: 75,
		LastCreatedAt:     &lastCreated,
		LastUpdatedAt:     &lastUpdated,
	}, func(t *testing.T, decoded UserMappingStats) {
		if decoded.TotalMappings != 100 {
			t.Errorf("Expected TotalMappings 100, got %d", decoded.TotalMappings)
		}
		if decoded.BySource["plex"] != 50 {
			t.Errorf("Expected BySource['plex'] 50, got %d", decoded.BySource["plex"])
		}
		if decoded.UniqueInternalIDs != 75 {
			t.Errorf("Expected UniqueInternalIDs 75, got %d", decoded.UniqueInternalIDs)
		}
	})
}

// =============================================================================
// FAILED EVENT AND DEDUPE AUDIT TESTS
// =============================================================================

func TestFailedEventModel(t *testing.T) {
	t.Parallel()

	lastRetry := testTime.Add(-5 * time.Minute)
	testJSONRoundTrip(t, "FailedEvent_Full", FailedEvent{
		ID:                 testUUID,
		TransactionID:      "txn-123",
		EventID:            "evt-456",
		SessionKey:         "sess-789",
		CorrelationKey:     "corr-abc",
		Source:             "plex",
		EventPayload:       []byte(`{"test": "data"}`),
		FailedAt:           testTime,
		FailureReason:      "database_error",
		FailureLayer:       "duckdb_insert",
		LastError:          "unique constraint violation",
		RetryCount:         3,
		LastRetryAt:        &lastRetry,
		MaxRetriesExceeded: true,
		Status:             "abandoned",
		CreatedAt:          testTime,
		UpdatedAt:          testTime,
	}, func(t *testing.T, decoded FailedEvent) {
		if decoded.TransactionID != "txn-123" {
			t.Errorf("Expected TransactionID 'txn-123', got '%s'", decoded.TransactionID)
		}
		if decoded.FailureLayer != "duckdb_insert" {
			t.Errorf("Expected FailureLayer 'duckdb_insert', got '%s'", decoded.FailureLayer)
		}
		if decoded.RetryCount != 3 {
			t.Errorf("Expected RetryCount 3, got %d", decoded.RetryCount)
		}
		if !decoded.MaxRetriesExceeded {
			t.Error("Expected MaxRetriesExceeded to be true")
		}
	})
}

func TestDedupeAuditEntryModel(t *testing.T) {
	t.Parallel()

	discardedStart := testTime.Add(-10 * time.Minute)
	similarity := 0.95
	resolvedAt := testTime

	testJSONRoundTrip(t, "DedupeAuditEntry_Full", DedupeAuditEntry{
		ID:                      testUUID,
		Timestamp:               testTime,
		DiscardedEventID:        "disc-123",
		DiscardedSessionKey:     "sess-disc",
		DiscardedCorrelationKey: "corr-disc",
		DiscardedSource:         "tautulli",
		DiscardedStartedAt:      &discardedStart,
		DiscardedRawPayload:     []byte(`{"discarded": true}`),
		MatchedEventID:          "match-456",
		MatchedSessionKey:       "sess-match",
		MatchedCorrelationKey:   "corr-match",
		MatchedSource:           "plex",
		DedupeReason:            "correlation_key",
		DedupeLayer:             "bloom_cache",
		SimilarityScore:         &similarity,
		UserID:                  42,
		Username:                "testuser",
		MediaType:               "movie",
		Title:                   "Test Movie",
		RatingKey:               "rating-123",
		Status:                  "auto_dedupe",
		ResolvedBy:              "admin",
		ResolvedAt:              &resolvedAt,
		ResolutionNotes:         "Verified duplicate",
		CreatedAt:               testTime,
	}, func(t *testing.T, decoded DedupeAuditEntry) {
		if decoded.DedupeReason != "correlation_key" {
			t.Errorf("Expected DedupeReason 'correlation_key', got '%s'", decoded.DedupeReason)
		}
		if decoded.DedupeLayer != "bloom_cache" {
			t.Errorf("Expected DedupeLayer 'bloom_cache', got '%s'", decoded.DedupeLayer)
		}
		if decoded.SimilarityScore == nil || *decoded.SimilarityScore != 0.95 {
			t.Error("SimilarityScore not properly marshaled/unmarshaled")
		}
	})
}

func TestDedupeAuditStatsModel(t *testing.T) {
	t.Parallel()

	testJSONRoundTrip(t, "DedupeAuditStats", DedupeAuditStats{
		TotalDeduped:   1000,
		PendingReview:  50,
		UserRestored:   10,
		UserConfirmed:  940,
		AccuracyRate:   98.9,
		DedupeByReason: map[string]int64{"correlation_key": 600, "session_key": 300, "event_id": 100},
		DedupeByLayer:  map[string]int64{"bloom_cache": 800, "nats_dedup": 150, "db_unique": 50},
		DedupeBySource: map[string]int64{"plex": 500, "tautulli": 400, "jellyfin": 100},
		Last24Hours:    50,
		Last7Days:      300,
		Last30Days:     1000,
	}, func(t *testing.T, decoded DedupeAuditStats) {
		if decoded.TotalDeduped != 1000 {
			t.Errorf("Expected TotalDeduped 1000, got %d", decoded.TotalDeduped)
		}
		if decoded.AccuracyRate != 98.9 {
			t.Errorf("Expected AccuracyRate 98.9, got %f", decoded.AccuracyRate)
		}
		if decoded.DedupeByReason["correlation_key"] != 600 {
			t.Errorf("Expected DedupeByReason['correlation_key'] 600, got %d", decoded.DedupeByReason["correlation_key"])
		}
	})
}

// =============================================================================
// ANALYTICS MODELS TESTS
// =============================================================================

func TestEngagementModels(t *testing.T) {
	t.Parallel()

	// ViewingPatternByHour
	testJSONRoundTrip(t, "ViewingPatternByHour", ViewingPatternByHour{
		HourOfDay:        14,
		SessionCount:     50,
		WatchTimeMinutes: 2500.5,
		UniqueUsers:      25,
		AvgCompletion:    85.0,
	}, func(t *testing.T, decoded ViewingPatternByHour) {
		if decoded.HourOfDay != 14 {
			t.Errorf("Expected HourOfDay 14, got %d", decoded.HourOfDay)
		}
	})

	// ViewingPatternByDay
	testJSONRoundTrip(t, "ViewingPatternByDay", ViewingPatternByDay{
		DayOfWeek:        6, // Saturday
		DayName:          "Saturday",
		SessionCount:     100,
		WatchTimeMinutes: 5000.0,
		UniqueUsers:      40,
		AvgCompletion:    90.0,
	}, func(t *testing.T, decoded ViewingPatternByDay) {
		if decoded.DayName != "Saturday" {
			t.Errorf("Expected DayName 'Saturday', got '%s'", decoded.DayName)
		}
	})

	// UserEngagementSummary
	testJSONRoundTrip(t, "UserEngagementSummary", UserEngagementSummary{
		TotalUsers:            100,
		ActiveUsers:           75,
		TotalWatchTimeMinutes: 50000.0,
		TotalSessions:         500,
		AvgSessionMinutes:     100.0,
		AvgUserWatchTime:      500.0,
		AvgCompletionRate:     85.0,
		ReturnVisitorRate:     80.0,
	}, func(t *testing.T, decoded UserEngagementSummary) {
		if decoded.TotalUsers != 100 {
			t.Errorf("Expected TotalUsers 100, got %d", decoded.TotalUsers)
		}
	})

	// UserEngagementAnalytics
	mostActiveHour := 20
	mostActiveDay := 6
	testJSONRoundTrip(t, "UserEngagementAnalytics", UserEngagementAnalytics{
		Summary: UserEngagementSummary{TotalUsers: 100, ActiveUsers: 75},
		TopUsers: []UserEngagement{
			{UserID: 1, Username: "topuser", TotalSessions: 50, ActivityScore: 95.0},
		},
		ViewingPatternsByHour: []ViewingPatternByHour{
			{HourOfDay: 20, SessionCount: 50},
		},
		ViewingPatternsByDay: []ViewingPatternByDay{
			{DayOfWeek: 6, DayName: "Saturday", SessionCount: 100},
		},
		MostActiveHour: &mostActiveHour,
		MostActiveDay:  &mostActiveDay,
	}, func(t *testing.T, decoded UserEngagementAnalytics) {
		if decoded.MostActiveHour == nil || *decoded.MostActiveHour != 20 {
			t.Error("MostActiveHour not properly marshaled/unmarshaled")
		}
		if len(decoded.TopUsers) != 1 {
			t.Errorf("Expected 1 TopUser, got %d", len(decoded.TopUsers))
		}
	})
}

func TestCohortRetentionModels(t *testing.T) {
	t.Parallel()

	// WeekRetention
	testJSONRoundTrip(t, "WeekRetention", WeekRetention{
		WeekOffset:    1,
		ActiveUsers:   80,
		RetentionRate: 80.0,
		WeekDate:      testTime,
	}, func(t *testing.T, decoded WeekRetention) {
		if decoded.RetentionRate != 80.0 {
			t.Errorf("Expected RetentionRate 80.0, got %f", decoded.RetentionRate)
		}
	})

	// CohortData
	testJSONRoundTrip(t, "CohortData", CohortData{
		CohortWeek:      "2024-W01",
		CohortStartDate: testTime,
		InitialUsers:    100,
		Retention: []WeekRetention{
			{WeekOffset: 0, ActiveUsers: 100, RetentionRate: 100.0, WeekDate: testTime},
			{WeekOffset: 1, ActiveUsers: 80, RetentionRate: 80.0, WeekDate: testTime.Add(7 * 24 * time.Hour)},
		},
		AverageRetention: 80.0,
		ChurnRate:        20.0,
	}, func(t *testing.T, decoded CohortData) {
		if decoded.CohortWeek != "2024-W01" {
			t.Errorf("Expected CohortWeek '2024-W01', got '%s'", decoded.CohortWeek)
		}
		if len(decoded.Retention) != 2 {
			t.Errorf("Expected 2 retention entries, got %d", len(decoded.Retention))
		}
	})

	// CohortRetentionSummary
	testJSONRoundTrip(t, "CohortRetentionSummary", CohortRetentionSummary{
		TotalCohorts:            12,
		TotalUsersTracked:       500,
		Week1Retention:          75.0,
		Week4Retention:          50.0,
		Week12Retention:         30.0,
		MedianRetentionWeek1:    78.0,
		BestPerformingCohort:    "2024-W05",
		WorstPerformingCohort:   "2024-W02",
		OverallAverageRetention: 55.0,
		RetentionTrend:          "improving",
	}, func(t *testing.T, decoded CohortRetentionSummary) {
		if decoded.TotalCohorts != 12 {
			t.Errorf("Expected TotalCohorts 12, got %d", decoded.TotalCohorts)
		}
		if decoded.RetentionTrend != "improving" {
			t.Errorf("Expected RetentionTrend 'improving', got '%s'", decoded.RetentionTrend)
		}
	})

	// RetentionPoint
	testJSONRoundTrip(t, "RetentionPoint", RetentionPoint{
		WeekOffset:       4,
		AverageRetention: 50.0,
		MedianRetention:  52.0,
		MinRetention:     30.0,
		MaxRetention:     70.0,
		CohortsWithData:  10,
	}, func(t *testing.T, decoded RetentionPoint) {
		if decoded.WeekOffset != 4 {
			t.Errorf("Expected WeekOffset 4, got %d", decoded.WeekOffset)
		}
	})

	// CohortQueryMetadata
	testJSONRoundTrip(t, "CohortQueryMetadata", CohortQueryMetadata{
		QueryHash:         "abc123def456",
		DataRangeStart:    testTime.Add(-90 * 24 * time.Hour),
		DataRangeEnd:      testTime,
		CohortGranularity: "week",
		MaxWeeksTracked:   12,
		EventCount:        50000,
		GeneratedAt:       testTime,
		QueryTimeMs:       150,
		Cached:            false,
	}, func(t *testing.T, decoded CohortQueryMetadata) {
		if decoded.CohortGranularity != "week" {
			t.Errorf("Expected CohortGranularity 'week', got '%s'", decoded.CohortGranularity)
		}
		if decoded.EventCount != 50000 {
			t.Errorf("Expected EventCount 50000, got %d", decoded.EventCount)
		}
	})
}

func TestQualityAnalyticsModels(t *testing.T) {
	t.Parallel()

	// ResolutionMismatch
	testJSONRoundTrip(t, "ResolutionMismatch", ResolutionMismatch{
		SourceResolution:  "4K",
		StreamResolution:  "1080p",
		PlaybackCount:     50,
		Percentage:        10.0,
		AffectedUsers:     []string{"user1", "user2"},
		CommonPlatforms:   []string{"iOS", "Android"},
		TranscodeRequired: true,
	}, func(t *testing.T, decoded ResolutionMismatch) {
		if decoded.SourceResolution != "4K" {
			t.Errorf("Expected SourceResolution '4K', got '%s'", decoded.SourceResolution)
		}
		if len(decoded.AffectedUsers) != 2 {
			t.Errorf("Expected 2 AffectedUsers, got %d", len(decoded.AffectedUsers))
		}
	})

	// HDRAnalytics
	testJSONRoundTrip(t, "HDRAnalytics", HDRAnalytics{
		TotalPlaybacks: 1000,
		FormatDistribution: []DynamicRangeDistribution{
			{DynamicRange: "HDR10", PlaybackCount: 200, Percentage: 20.0, UniqueUsers: 30},
			{DynamicRange: "SDR", PlaybackCount: 800, Percentage: 80.0, UniqueUsers: 80},
		},
		HDRAdoptionRate: 20.0,
		ToneMappingEvents: []ToneMappingEvent{
			{SourceFormat: "HDR10", StreamFormat: "SDR", OccurrenceCount: 50},
		},
		HDRCapableDevices: []HDRDeviceStats{
			{Platform: "Apple TV", HDRPlaybacks: 100, SDRPlaybacks: 50, HDRCapable: true},
		},
		ContentByFormat: []ContentFormatStats{
			{DynamicRange: "HDR10", UniqueContent: 50, AvgCompletion: 90.0, MostWatchedTitle: "Dune"},
		},
	}, func(t *testing.T, decoded HDRAnalytics) {
		if decoded.HDRAdoptionRate != 20.0 {
			t.Errorf("Expected HDRAdoptionRate 20.0, got %f", decoded.HDRAdoptionRate)
		}
		if len(decoded.FormatDistribution) != 2 {
			t.Errorf("Expected 2 FormatDistribution entries, got %d", len(decoded.FormatDistribution))
		}
	})

	// AudioAnalytics
	testJSONRoundTrip(t, "AudioAnalytics", AudioAnalytics{
		TotalPlaybacks: 1000,
		ChannelDistribution: []AudioChannelDistribution{
			{Channels: "5.1", Layout: "5.1(side)", PlaybackCount: 300, Percentage: 30.0, AvgCompletion: 85.0},
		},
		CodecDistribution: []AudioCodecDistribution{
			{Codec: "eac3", PlaybackCount: 400, Percentage: 40.0, IsLossless: false, AvgBitrate: 640.0},
		},
		DownmixEvents: []AudioDownmixEvent{
			{SourceChannels: "7.1", StreamChannels: "stereo", OccurrenceCount: 20, Platforms: []string{"iOS"}},
		},
		SurroundSoundAdoption: 45.0,
		LosslessAdoption:      10.0,
		AvgBitrate:            500.0,
		AtmosPlaybacks:        50,
	}, func(t *testing.T, decoded AudioAnalytics) {
		if decoded.SurroundSoundAdoption != 45.0 {
			t.Errorf("Expected SurroundSoundAdoption 45.0, got %f", decoded.SurroundSoundAdoption)
		}
		if decoded.AtmosPlaybacks != 50 {
			t.Errorf("Expected AtmosPlaybacks 50, got %d", decoded.AtmosPlaybacks)
		}
	})

	// SubtitleAnalytics
	testJSONRoundTrip(t, "SubtitleAnalytics", SubtitleAnalytics{
		TotalPlaybacks:       1000,
		SubtitleUsageRate:    35.0,
		PlaybacksWithSubs:    350,
		PlaybacksWithoutSubs: 650,
		LanguageDistribution: []SubtitleLanguageStats{
			{Language: "English", PlaybackCount: 200, Percentage: 57.1, UniqueUsers: 40},
		},
		CodecDistribution: []SubtitleCodecStats{
			{Codec: "srt", PlaybackCount: 300, Percentage: 85.7},
		},
		UserPreferences: []UserSubtitlePreference{
			{Username: "user1", TotalPlaybacks: 100, SubtitleUsageCount: 50, SubtitleUsageRate: 50.0, PreferredLanguages: []string{"English", "Spanish"}},
		},
	}, func(t *testing.T, decoded SubtitleAnalytics) {
		if decoded.SubtitleUsageRate != 35.0 {
			t.Errorf("Expected SubtitleUsageRate 35.0, got %f", decoded.SubtitleUsageRate)
		}
	})

	// FrameRateAnalytics
	testJSONRoundTrip(t, "FrameRateAnalytics", FrameRateAnalytics{
		TotalPlaybacks: 1000,
		FrameRateDistribution: []FrameRateDistribution{
			{FrameRate: "24", PlaybackCount: 600, Percentage: 60.0, AvgCompletion: 88.0},
			{FrameRate: "60", PlaybackCount: 100, Percentage: 10.0, AvgCompletion: 92.0},
		},
		ByMediaType: map[string][]FrameRateDistribution{
			"movie": {{FrameRate: "24", PlaybackCount: 500}},
		},
		HighFrameRateAdoption: 10.0,
		ConversionEvents: []FrameRateConversion{
			{SourceFPS: "60", StreamFPS: "30", OccurrenceCount: 20, Platforms: []string{"Chrome"}},
		},
	}, func(t *testing.T, decoded FrameRateAnalytics) {
		if decoded.HighFrameRateAdoption != 10.0 {
			t.Errorf("Expected HighFrameRateAdoption 10.0, got %f", decoded.HighFrameRateAdoption)
		}
	})

	// ContainerAnalytics
	testJSONRoundTrip(t, "ContainerAnalytics", ContainerAnalytics{
		TotalPlaybacks: 1000,
		FormatDistribution: []ContainerDistribution{
			{Container: "mkv", PlaybackCount: 400, Percentage: 40.0, DirectPlayRate: 70.0},
		},
		DirectPlayRates: map[string]float64{"mkv": 70.0, "mp4": 95.0},
		RemuxEvents: []ContainerRemux{
			{SourceContainer: "mkv", StreamContainer: "mp4", OccurrenceCount: 100, Platforms: []string{"Safari"}},
		},
		PlatformCompatibility: []PlatformContainer{
			{Platform: "iOS", SupportedFormats: []string{"mp4", "mov"}, TranscodeRequired: []string{"mkv", "avi"}, DirectPlayRate: 60.0},
		},
	}, func(t *testing.T, decoded ContainerAnalytics) {
		if decoded.DirectPlayRates["mkv"] != 70.0 {
			t.Errorf("Expected DirectPlayRates['mkv'] 70.0, got %f", decoded.DirectPlayRates["mkv"])
		}
	})
}

// =============================================================================
// LOCATION AND GEOLOCATION TESTS
// =============================================================================

func TestLocationModels(t *testing.T) {
	t.Parallel()

	// LocationStats with nil optional fields
	testJSONRoundTrip(t, "LocationStats_Minimal", LocationStats{
		Country:       "United States",
		Latitude:      39.9526,
		Longitude:     -75.1652,
		PlaybackCount: 100,
		UniqueUsers:   25,
		FirstSeen:     testTime.Add(-30 * 24 * time.Hour),
		LastSeen:      testTime,
		AvgCompletion: 85.0,
	}, func(t *testing.T, decoded LocationStats) {
		if decoded.Country != "United States" {
			t.Errorf("Expected Country 'United States', got '%s'", decoded.Country)
		}
		if decoded.Region != nil {
			t.Error("Expected Region to be nil")
		}
	})

	// LocationStats with all fields
	region := "Pennsylvania"
	city := "Philadelphia"
	testJSONRoundTrip(t, "LocationStats_Full", LocationStats{
		Country:       "United States",
		Region:        &region,
		City:          &city,
		Latitude:      39.9526,
		Longitude:     -75.1652,
		PlaybackCount: 100,
		UniqueUsers:   25,
		FirstSeen:     testTime.Add(-30 * 24 * time.Hour),
		LastSeen:      testTime,
		AvgCompletion: 85.0,
	}, func(t *testing.T, decoded LocationStats) {
		if decoded.Region == nil || *decoded.Region != "Pennsylvania" {
			t.Error("Region not properly marshaled/unmarshaled")
		}
		if decoded.City == nil || *decoded.City != "Philadelphia" {
			t.Error("City not properly marshaled/unmarshaled")
		}
	})

	// Geolocation boundary tests
	testJSONRoundTrip(t, "Geolocation_BoundaryCoords", Geolocation{
		IPAddress:   "192.168.1.1",
		Latitude:    90.0,  // North Pole
		Longitude:   180.0, // Date line
		Country:     "Antarctica",
		LastUpdated: testTime,
	}, func(t *testing.T, decoded Geolocation) {
		if decoded.Latitude != 90.0 {
			t.Errorf("Expected Latitude 90.0, got %f", decoded.Latitude)
		}
		if decoded.Longitude != 180.0 {
			t.Errorf("Expected Longitude 180.0, got %f", decoded.Longitude)
		}
	})

	// Negative coordinates
	testJSONRoundTrip(t, "Geolocation_NegativeCoords", Geolocation{
		IPAddress:   "192.168.1.1",
		Latitude:    -90.0,  // South Pole
		Longitude:   -180.0, // Date line
		Country:     "Antarctica",
		LastUpdated: testTime,
	}, func(t *testing.T, decoded Geolocation) {
		if decoded.Latitude != -90.0 {
			t.Errorf("Expected Latitude -90.0, got %f", decoded.Latitude)
		}
		if decoded.Longitude != -180.0 {
			t.Errorf("Expected Longitude -180.0, got %f", decoded.Longitude)
		}
	})
}

// =============================================================================
// SETUP STATUS MODELS TESTS
// =============================================================================

func TestSetupStatusModels(t *testing.T) {
	t.Parallel()

	// SetupStatus full
	testJSONRoundTrip(t, "SetupStatus_Full", SetupStatus{
		Ready: true,
		Database: SetupDatabaseStatus{
			Connected: true,
		},
		DataSources: SetupDataSources{
			Tautulli: SetupDataSourceStatus{Configured: true, Connected: true, URL: "http://localhost:8181"},
			Plex:     SetupMediaServerStatus{Configured: true, Connected: true, ServerCount: 2},
			Jellyfin: SetupMediaServerStatus{Configured: false},
			Emby:     SetupMediaServerStatus{Configured: false},
			NATS:     SetupOptionalFeature{Enabled: true, Connected: true},
		},
		DataAvailable: SetupDataAvailability{
			HasPlaybacks:    true,
			PlaybackCount:   5000,
			HasGeolocations: true,
		},
		Recommendations: []string{"Configure Jellyfin for additional media", "Enable NATS for real-time updates"},
	}, func(t *testing.T, decoded SetupStatus) {
		if !decoded.Ready {
			t.Error("Expected Ready to be true")
		}
		if decoded.DataSources.Plex.ServerCount != 2 {
			t.Errorf("Expected Plex ServerCount 2, got %d", decoded.DataSources.Plex.ServerCount)
		}
		if len(decoded.Recommendations) != 2 {
			t.Errorf("Expected 2 recommendations, got %d", len(decoded.Recommendations))
		}
	})

	// SetupStatus with errors
	testJSONRoundTrip(t, "SetupStatus_WithErrors", SetupStatus{
		Ready: false,
		Database: SetupDatabaseStatus{
			Connected: false,
		},
		DataSources: SetupDataSources{
			Tautulli: SetupDataSourceStatus{Configured: true, Connected: false, Error: "Connection refused"},
			Plex:     SetupMediaServerStatus{Configured: true, Connected: false, Error: "Invalid token"},
			Jellyfin: SetupMediaServerStatus{Configured: false},
			Emby:     SetupMediaServerStatus{Configured: false},
			NATS:     SetupOptionalFeature{Enabled: false},
		},
		DataAvailable: SetupDataAvailability{
			HasPlaybacks:    false,
			PlaybackCount:   0,
			HasGeolocations: false,
		},
	}, func(t *testing.T, decoded SetupStatus) {
		if decoded.Ready {
			t.Error("Expected Ready to be false")
		}
		if decoded.DataSources.Tautulli.Error != "Connection refused" {
			t.Errorf("Expected Tautulli error 'Connection refused', got '%s'", decoded.DataSources.Tautulli.Error)
		}
	})
}

// =============================================================================
// PAGINATION MODELS TESTS
// =============================================================================

func TestPaginationModels(t *testing.T) {
	t.Parallel()

	// PaginationInfo with all fields
	nextCursor := "eyJzdGFydGVkX2F0IjoiMjAyNS0wMS0wMVQxMjowMDowMFoiLCJpZCI6ImFiYzEyMyJ9"
	prevCursor := "eyJzdGFydGVkX2F0IjoiMjAyNS0wMS0wMVQxMTowMDowMFoiLCJpZCI6InByZXYxMjMifQ=="
	totalCount := 500
	testJSONRoundTrip(t, "PaginationInfo_Full", PaginationInfo{
		Limit:      100,
		HasMore:    true,
		NextCursor: &nextCursor,
		PrevCursor: &prevCursor,
		TotalCount: &totalCount,
	}, func(t *testing.T, decoded PaginationInfo) {
		if decoded.Limit != 100 {
			t.Errorf("Expected Limit 100, got %d", decoded.Limit)
		}
		if !decoded.HasMore {
			t.Error("Expected HasMore to be true")
		}
		if decoded.NextCursor == nil || *decoded.NextCursor != nextCursor {
			t.Error("NextCursor not properly marshaled/unmarshaled")
		}
	})

	// PaginationInfo minimal (first page)
	testJSONRoundTrip(t, "PaginationInfo_FirstPage", PaginationInfo{
		Limit:   50,
		HasMore: false,
	}, func(t *testing.T, decoded PaginationInfo) {
		if decoded.NextCursor != nil {
			t.Error("Expected NextCursor to be nil")
		}
		if decoded.PrevCursor != nil {
			t.Error("Expected PrevCursor to be nil")
		}
	})

	// PlaybackCursor
	testJSONRoundTrip(t, "PlaybackCursor", PlaybackCursor{
		StartedAt: testTime,
		ID:        "abc123",
	}, func(t *testing.T, decoded PlaybackCursor) {
		if decoded.ID != "abc123" {
			t.Errorf("Expected ID 'abc123', got '%s'", decoded.ID)
		}
	})

	// PlaybacksResponse
	testJSONRoundTrip(t, "PlaybacksResponse", PlaybacksResponse{
		Events: []PlaybackEvent{
			createTestPlaybackEvent(),
		},
		Pagination: PaginationInfo{
			Limit:   100,
			HasMore: true,
		},
	}, func(t *testing.T, decoded PlaybacksResponse) {
		if len(decoded.Events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(decoded.Events))
		}
	})
}

// =============================================================================
// EMPTY SLICE VS NIL SLICE TESTS
// =============================================================================

func TestEmptyVsNilSlices(t *testing.T) {
	t.Parallel()

	// Test that empty slices marshal to [] not null
	stats := Stats{
		TotalPlaybacks:    0,
		UniqueLocations:   0,
		UniqueUsers:       0,
		TopCountries:      []CountryStats{}, // Empty slice
		RecentActivity:    0,
		DatabaseSizeBytes: 0,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Check that TopCountries is [] not null
	if string(data) == "" {
		t.Fatal("Marshaled data is empty")
	}

	var decoded Stats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Empty slice should remain a slice (not nil) after round-trip
	if decoded.TopCountries == nil {
		t.Error("Expected TopCountries to be empty slice, got nil")
	}
	if len(decoded.TopCountries) != 0 {
		t.Errorf("Expected 0 TopCountries, got %d", len(decoded.TopCountries))
	}
}

// =============================================================================
// VIEWING HOURS HEATMAP TESTS
// =============================================================================

func TestViewingHoursHeatmapModels(t *testing.T) {
	t.Parallel()

	testJSONRoundTrip(t, "ViewingHoursHeatmap", ViewingHoursHeatmap{
		DayOfWeek:     5, // Friday
		Hour:          20,
		PlaybackCount: 150,
	}, func(t *testing.T, decoded ViewingHoursHeatmap) {
		if decoded.Hour != 20 {
			t.Errorf("Expected Hour 20, got %d", decoded.Hour)
		}
		if decoded.DayOfWeek != 5 {
			t.Errorf("Expected DayOfWeek 5, got %d", decoded.DayOfWeek)
		}
	})
}

// =============================================================================
// RESPONSE WRAPPER TESTS
// =============================================================================

func TestResponseWrappers(t *testing.T) {
	t.Parallel()

	// TrendsResponse
	testJSONRoundTrip(t, "TrendsResponse", TrendsResponse{
		PlaybackTrends: []PlaybackTrend{
			{Date: "2024-01-01", PlaybackCount: 100, UniqueUsers: 25},
			{Date: "2024-01-02", PlaybackCount: 120, UniqueUsers: 30},
		},
		Interval: "day",
	}, func(t *testing.T, decoded TrendsResponse) {
		if decoded.Interval != "day" {
			t.Errorf("Expected Interval 'day', got '%s'", decoded.Interval)
		}
		if len(decoded.PlaybackTrends) != 2 {
			t.Errorf("Expected 2 PlaybackTrends, got %d", len(decoded.PlaybackTrends))
		}
	})

	// UsersResponse
	testJSONRoundTrip(t, "UsersResponse", UsersResponse{
		TopUsers: []UserActivity{
			{Username: "user1", PlaybackCount: 100, TotalDuration: 5000, AvgCompletion: 85.0, UniqueMedia: 50},
		},
	}, func(t *testing.T, decoded UsersResponse) {
		if len(decoded.TopUsers) != 1 {
			t.Errorf("Expected 1 TopUser, got %d", len(decoded.TopUsers))
		}
	})

	// GeographicResponse
	testJSONRoundTrip(t, "GeographicResponse", GeographicResponse{
		TopCities:    []CityStats{{City: "Philadelphia", Country: "US", PlaybackCount: 100, UniqueUsers: 25}},
		TopCountries: []CountryStats{{Country: "US", PlaybackCount: 500, UniqueUsers: 100}},
		MediaTypeDistribution: []MediaTypeStats{
			{MediaType: "movie", PlaybackCount: 300, UniqueUsers: 80},
		},
		ViewingHoursHeatmap: []ViewingHoursHeatmap{
			{DayOfWeek: 5, Hour: 20, PlaybackCount: 50},
		},
		PlatformDistribution:   []PlatformStats{{Platform: "iOS", PlaybackCount: 200, UniqueUsers: 40}},
		PlayerDistribution:     []PlayerStats{{Player: "Plex Web", PlaybackCount: 150, UniqueUsers: 30}},
		ContentCompletionStats: ContentCompletionStats{TotalPlaybacks: 500, AvgCompletion: 80.0},
		TranscodeDistribution:  []TranscodeStats{{TranscodeDecision: "direct play", PlaybackCount: 400, Percentage: 80.0}},
		ResolutionDistribution: []ResolutionStats{{VideoResolution: "1080p", PlaybackCount: 350, Percentage: 70.0}},
		CodecDistribution:      []CodecStats{{VideoCodec: "h264", AudioCodec: "aac", PlaybackCount: 300, Percentage: 60.0}},
		LibraryDistribution:    []LibraryStats{{SectionID: 1, LibraryName: "Movies", PlaybackCount: 250, UniqueUsers: 50}},
		RatingDistribution:     []RatingStats{{ContentRating: "PG-13", PlaybackCount: 180, Percentage: 36.0}},
		DurationStats:          DurationStats{AvgDuration: 90, MedianDuration: 85, TotalDuration: 45000},
		YearDistribution:       []YearStats{{Year: 2024, PlaybackCount: 500}},
	}, func(t *testing.T, decoded GeographicResponse) {
		if len(decoded.TopCities) != 1 {
			t.Errorf("Expected 1 TopCity, got %d", len(decoded.TopCities))
		}
		if decoded.DurationStats.AvgDuration != 90 {
			t.Errorf("Expected AvgDuration 90, got %d", decoded.DurationStats.AvgDuration)
		}
	})
}

// =============================================================================
// SPATIAL ANALYTICS MODELS TESTS
// =============================================================================

func TestSpatialModels(t *testing.T) {
	t.Parallel()

	// H3HexagonStats
	testJSONRoundTrip(t, "H3HexagonStats", H3HexagonStats{
		H3Index:           0x8928308280fffff,
		Latitude:          39.9526,
		Longitude:         -75.1652,
		PlaybackCount:     100,
		UniqueUsers:       25,
		AvgCompletion:     85.5,
		TotalWatchMinutes: 5000,
	}, func(t *testing.T, decoded H3HexagonStats) {
		if decoded.H3Index != 0x8928308280fffff {
			t.Errorf("Expected H3Index 0x8928308280fffff, got %x", decoded.H3Index)
		}
		if decoded.PlaybackCount != 100 {
			t.Errorf("Expected PlaybackCount 100, got %d", decoded.PlaybackCount)
		}
	})

	// ArcStats
	testJSONRoundTrip(t, "ArcStats", ArcStats{
		UserLatitude:    39.9526,
		UserLongitude:   -75.1652,
		ServerLatitude:  37.7749,
		ServerLongitude: -122.4194,
		City:            "Philadelphia",
		Country:         "United States",
		DistanceKm:      3900.5,
		PlaybackCount:   50,
		UniqueUsers:     10,
		AvgCompletion:   90.0,
		Weight:          195025.0,
	}, func(t *testing.T, decoded ArcStats) {
		if decoded.DistanceKm != 3900.5 {
			t.Errorf("Expected DistanceKm 3900.5, got %f", decoded.DistanceKm)
		}
		if decoded.City != "Philadelphia" {
			t.Errorf("Expected City 'Philadelphia', got '%s'", decoded.City)
		}
	})

	// TemporalSpatialPoint
	testJSONRoundTrip(t, "TemporalSpatialPoint", TemporalSpatialPoint{
		TimeBucket:          testTime,
		H3Index:             0x8928308280fffff,
		Latitude:            39.9526,
		Longitude:           -75.1652,
		PlaybackCount:       50,
		UniqueUsers:         15,
		RollingAvgPlaybacks: 45.5,
		CumulativePlaybacks: 500,
	}, func(t *testing.T, decoded TemporalSpatialPoint) {
		if decoded.RollingAvgPlaybacks != 45.5 {
			t.Errorf("Expected RollingAvgPlaybacks 45.5, got %f", decoded.RollingAvgPlaybacks)
		}
		if decoded.CumulativePlaybacks != 500 {
			t.Errorf("Expected CumulativePlaybacks 500, got %d", decoded.CumulativePlaybacks)
		}
	})

	// ViewportBounds - normal case
	testJSONRoundTrip(t, "ViewportBounds_Normal", ViewportBounds{
		West:  -122.5,
		South: 37.5,
		East:  -122.0,
		North: 38.0,
	}, func(t *testing.T, decoded ViewportBounds) {
		if decoded.West != -122.5 {
			t.Errorf("Expected West -122.5, got %f", decoded.West)
		}
	})

	// ViewportBounds - cross date line
	testJSONRoundTrip(t, "ViewportBounds_CrossDateLine", ViewportBounds{
		West:  170.0,
		South: -10.0,
		East:  -170.0,
		North: 10.0,
	}, func(t *testing.T, decoded ViewportBounds) {
		if decoded.West != 170.0 || decoded.East != -170.0 {
			t.Error("Date line crossing bounds not preserved")
		}
	})

	// ProximityQuery
	testJSONRoundTrip(t, "ProximityQuery", ProximityQuery{
		Latitude:  39.9526,
		Longitude: -75.1652,
		RadiusKm:  50.0,
	}, func(t *testing.T, decoded ProximityQuery) {
		if decoded.RadiusKm != 50.0 {
			t.Errorf("Expected RadiusKm 50.0, got %f", decoded.RadiusKm)
		}
	})
}

// =============================================================================
// CONCURRENT STREAMS MODELS TESTS
// =============================================================================

func TestConcurrentStreamsModels(t *testing.T) {
	t.Parallel()

	// ConcurrentStreamsTimeBucket
	testJSONRoundTrip(t, "ConcurrentStreamsTimeBucket", ConcurrentStreamsTimeBucket{
		Timestamp:       testTime,
		ConcurrentCount: 10,
		DirectPlay:      5,
		DirectStream:    3,
		Transcode:       2,
	}, func(t *testing.T, decoded ConcurrentStreamsTimeBucket) {
		if decoded.ConcurrentCount != 10 {
			t.Errorf("Expected ConcurrentCount 10, got %d", decoded.ConcurrentCount)
		}
		if decoded.DirectPlay+decoded.DirectStream+decoded.Transcode != 10 {
			t.Error("Stream counts don't add up to concurrent count")
		}
	})

	// ConcurrentStreamsByType
	testJSONRoundTrip(t, "ConcurrentStreamsByType", ConcurrentStreamsByType{
		TranscodeDecision: "transcode",
		AvgConcurrent:     3.5,
		MaxConcurrent:     8,
		Percentage:        35.0,
	}, func(t *testing.T, decoded ConcurrentStreamsByType) {
		if decoded.TranscodeDecision != "transcode" {
			t.Errorf("Expected TranscodeDecision 'transcode', got '%s'", decoded.TranscodeDecision)
		}
	})

	// ConcurrentStreamsByDayOfWeek
	testJSONRoundTrip(t, "ConcurrentStreamsByDayOfWeek", ConcurrentStreamsByDayOfWeek{
		DayOfWeek:      6, // Saturday
		AvgConcurrent:  5.5,
		PeakConcurrent: 12,
	}, func(t *testing.T, decoded ConcurrentStreamsByDayOfWeek) {
		if decoded.DayOfWeek != 6 {
			t.Errorf("Expected DayOfWeek 6, got %d", decoded.DayOfWeek)
		}
		if decoded.PeakConcurrent != 12 {
			t.Errorf("Expected PeakConcurrent 12, got %d", decoded.PeakConcurrent)
		}
	})

	// ConcurrentStreamsByHour
	testJSONRoundTrip(t, "ConcurrentStreamsByHour", ConcurrentStreamsByHour{
		Hour:           20, // 8 PM
		AvgConcurrent:  8.5,
		PeakConcurrent: 15,
	}, func(t *testing.T, decoded ConcurrentStreamsByHour) {
		if decoded.Hour != 20 {
			t.Errorf("Expected Hour 20, got %d", decoded.Hour)
		}
	})

	// ConcurrentStreamsAnalytics - full
	testJSONRoundTrip(t, "ConcurrentStreamsAnalytics", ConcurrentStreamsAnalytics{
		PeakConcurrent: 15,
		PeakTime:       testTime,
		AvgConcurrent:  5.5,
		TotalSessions:  500,
		TimeSeriesData: []ConcurrentStreamsTimeBucket{
			{Timestamp: testTime, ConcurrentCount: 10, DirectPlay: 5, DirectStream: 3, Transcode: 2},
		},
		ByTranscodeDecision: []ConcurrentStreamsByType{
			{TranscodeDecision: "direct play", AvgConcurrent: 2.5, MaxConcurrent: 5, Percentage: 50.0},
		},
		ByDayOfWeek: []ConcurrentStreamsByDayOfWeek{
			{DayOfWeek: 6, AvgConcurrent: 5.5, PeakConcurrent: 12},
		},
		ByHourOfDay: []ConcurrentStreamsByHour{
			{Hour: 20, AvgConcurrent: 8.5, PeakConcurrent: 15},
		},
		CapacityRecommendation: "Current capacity is sufficient for 20 concurrent streams",
	}, func(t *testing.T, decoded ConcurrentStreamsAnalytics) {
		if decoded.PeakConcurrent != 15 {
			t.Errorf("Expected PeakConcurrent 15, got %d", decoded.PeakConcurrent)
		}
		if decoded.CapacityRecommendation == "" {
			t.Error("Expected CapacityRecommendation to be non-empty")
		}
	})
}

// =============================================================================
// DATA QUALITY MODELS TESTS
// =============================================================================

func TestDataQualityModels(t *testing.T) {
	t.Parallel()

	// DataQualitySummary
	testJSONRoundTrip(t, "DataQualitySummary", DataQualitySummary{
		TotalEvents:        100000,
		OverallScore:       92.5,
		Grade:              "A",
		CompletenessScore:  95.0,
		ValidityScore:      90.0,
		ConsistencyScore:   92.5,
		NullFieldRate:      2.5,
		InvalidValueRate:   1.5,
		DuplicateRate:      0.5,
		FutureDateRate:     0.1,
		OrphanedGeoRate:    3.0,
		IssueCount:         15,
		CriticalIssueCount: 2,
		TrendDirection:     "improving",
	}, func(t *testing.T, decoded DataQualitySummary) {
		if decoded.OverallScore != 92.5 {
			t.Errorf("Expected OverallScore 92.5, got %f", decoded.OverallScore)
		}
		if decoded.Grade != "A" {
			t.Errorf("Expected Grade 'A', got '%s'", decoded.Grade)
		}
		if decoded.TrendDirection != "improving" {
			t.Errorf("Expected TrendDirection 'improving', got '%s'", decoded.TrendDirection)
		}
	})

	// FieldQualityMetric
	testJSONRoundTrip(t, "FieldQualityMetric", FieldQualityMetric{
		FieldName:    "ip_address",
		Category:     "network",
		TotalRecords: 100000,
		NullCount:    500,
		NullRate:     0.5,
		InvalidCount: 100,
		InvalidRate:  0.1,
		UniqueCount:  50000,
		Cardinality:  0.5,
		QualityScore: 95.0,
		IsRequired:   true,
		Status:       "healthy",
	}, func(t *testing.T, decoded FieldQualityMetric) {
		if decoded.FieldName != "ip_address" {
			t.Errorf("Expected FieldName 'ip_address', got '%s'", decoded.FieldName)
		}
		if !decoded.IsRequired {
			t.Error("Expected IsRequired to be true")
		}
	})

	// DailyQualityTrend
	testJSONRoundTrip(t, "DailyQualityTrend", DailyQualityTrend{
		Date:         testTime,
		EventCount:   5000,
		OverallScore: 93.5,
		NullRate:     2.0,
		InvalidRate:  1.2,
		NewIssues:    2,
	}, func(t *testing.T, decoded DailyQualityTrend) {
		if decoded.EventCount != 5000 {
			t.Errorf("Expected EventCount 5000, got %d", decoded.EventCount)
		}
	})

	// DataQualityIssue
	testJSONRoundTrip(t, "DataQualityIssue", DataQualityIssue{
		ID:               "issue-001",
		Type:             "null_required",
		Severity:         "critical",
		Field:            "user_id",
		Title:            "Missing user IDs detected",
		Description:      "500 records have null user_id values",
		AffectedRecords:  500,
		ImpactPercentage: 0.5,
		FirstDetected:    testTime.Add(-24 * time.Hour),
		LastSeen:         testTime,
		ExampleValues:    []string{"null", "", "undefined"},
		Recommendation:   "Verify data pipeline is correctly extracting user IDs",
		AutoResolvable:   false,
	}, func(t *testing.T, decoded DataQualityIssue) {
		if decoded.Type != "null_required" {
			t.Errorf("Expected Type 'null_required', got '%s'", decoded.Type)
		}
		if decoded.Severity != "critical" {
			t.Errorf("Expected Severity 'critical', got '%s'", decoded.Severity)
		}
		if len(decoded.ExampleValues) != 3 {
			t.Errorf("Expected 3 ExampleValues, got %d", len(decoded.ExampleValues))
		}
	})

	// SourceQuality
	testJSONRoundTrip(t, "SourceQuality", SourceQuality{
		Source:          "plex",
		ServerID:        "server-123",
		EventCount:      50000,
		EventPercentage: 50.0,
		QualityScore:    94.5,
		NullRate:        2.0,
		InvalidRate:     1.0,
		Status:          "healthy",
		TopIssue:        "Missing geolocation data",
	}, func(t *testing.T, decoded SourceQuality) {
		if decoded.Source != "plex" {
			t.Errorf("Expected Source 'plex', got '%s'", decoded.Source)
		}
		if decoded.QualityScore != 94.5 {
			t.Errorf("Expected QualityScore 94.5, got %f", decoded.QualityScore)
		}
	})

	// DataQualityMetadata
	testJSONRoundTrip(t, "DataQualityMetadata", DataQualityMetadata{
		QueryHash:      "abc123def456",
		DataRangeStart: testTime.Add(-30 * 24 * time.Hour),
		DataRangeEnd:   testTime,
		AnalyzedTables: []string{"playback_events", "geolocations"},
		RulesApplied:   []string{"null_check", "type_validation", "range_check"},
		GeneratedAt:    testTime,
		QueryTimeMs:    250,
		Cached:         false,
	}, func(t *testing.T, decoded DataQualityMetadata) {
		if len(decoded.AnalyzedTables) != 2 {
			t.Errorf("Expected 2 AnalyzedTables, got %d", len(decoded.AnalyzedTables))
		}
		if len(decoded.RulesApplied) != 3 {
			t.Errorf("Expected 3 RulesApplied, got %d", len(decoded.RulesApplied))
		}
	})

	// DataQualityReport - full
	testJSONRoundTrip(t, "DataQualityReport", DataQualityReport{
		Summary: DataQualitySummary{
			TotalEvents:  100000,
			OverallScore: 92.5,
			Grade:        "A",
		},
		FieldQuality: []FieldQualityMetric{
			{FieldName: "user_id", Category: "identity", QualityScore: 98.0, Status: "healthy"},
		},
		DailyTrends: []DailyQualityTrend{
			{Date: testTime, EventCount: 5000, OverallScore: 93.5},
		},
		Issues: []DataQualityIssue{
			{ID: "issue-001", Type: "null_required", Severity: "warning"},
		},
		SourceBreakdown: []SourceQuality{
			{Source: "plex", EventCount: 50000, QualityScore: 94.5},
		},
		Metadata: DataQualityMetadata{
			QueryHash:   "abc123",
			GeneratedAt: testTime,
		},
	}, func(t *testing.T, decoded DataQualityReport) {
		if decoded.Summary.Grade != "A" {
			t.Errorf("Expected Summary.Grade 'A', got '%s'", decoded.Summary.Grade)
		}
		if len(decoded.FieldQuality) != 1 {
			t.Errorf("Expected 1 FieldQuality entry, got %d", len(decoded.FieldQuality))
		}
		if len(decoded.Issues) != 1 {
			t.Errorf("Expected 1 Issue, got %d", len(decoded.Issues))
		}
	})
}
