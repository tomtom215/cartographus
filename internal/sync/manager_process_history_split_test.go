// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestManager_ProcessHistoryRecord_SessionExists(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()
	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return true, nil // Session already exists
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("existing-session"),
		UserID:     intPtr(1),
		User:       "testuser",
		IPAddress:  "192.168.1.1",
		MediaType:  "movie",
		Title:      "Test Movie",
		Started:    time.Now().Unix(),
	}

	// Should return nil (no error) without processing
	err := manager.processHistoryRecord(context.Background(), record)
	if err != nil {
		t.Errorf("Expected no error for existing session, got: %v", err)
	}
}

func TestManager_ProcessHistoryRecord_InvalidIPAddress(t *testing.T) {
	t.Parallel() // Safe - subtests with isolated mocks

	tests := []struct {
		name      string
		ipAddress string
	}{
		{"empty IP", ""},
		{"N/A IP", "N/A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // Subtests can also run in parallel

			cfg := newTestConfig()

			mockDb := &mockDB{
				sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
					return false, nil
				},
			}

			manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

			record := &tautulli.TautulliHistoryRecord{
				SessionKey: stringPtr("test-session"),
				IPAddress:  tt.ipAddress,
				User:       "testuser",
				MediaType:  "movie",
				Title:      "Test Movie",
				Started:    time.Now().Unix(),
			}

			err := manager.processHistoryRecord(context.Background(), record)
			if err == nil {
				t.Error("Expected error for invalid IP address")
			}

			if !strings.Contains(err.Error(), "invalid IP address") {
				t.Errorf("Expected 'invalid IP address' error, got: %v", err)
			}
		})
	}
}

func TestManager_ProcessHistoryRecord_GeolocationCached(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()
	cachedGeo := &models.Geolocation{
		IPAddress: "8.8.8.8",
		Latitude:  37.7749,
		Longitude: -122.4194,
		Country:   "United States",
	}

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			if ipAddress == "8.8.8.8" {
				return cachedGeo, nil
			}
			return nil, nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			if event.IPAddress != "8.8.8.8" {
				t.Errorf("Expected IP 8.8.8.8, got %s", event.IPAddress)
			}
			if event.SessionKey != "test-session-1" {
				t.Errorf("Expected session key test-session-1, got %s", event.SessionKey)
			}
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("test-session-1"),
		UserID:     intPtr(1),
		User:       "testuser",
		IPAddress:  "8.8.8.8",
		MediaType:  "movie",
		Title:      "Test Movie",
		Platform:   "Chrome",
		Player:     "Plex Web",
		Started:    time.Now().Unix(),
	}

	err := manager.processHistoryRecord(context.Background(), record)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestManager_ProcessHistoryRecord_FetchGeolocation(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()
	geoLookupCalled := false
	// Use public IP address (TEST-NET-2) to test the geolocation fetch path
	// Private IPs (10.0.0.0/8, etc.) are now handled specially and don't trigger GeoIP lookup
	testIP := "198.51.100.1"

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return nil, nil // Not cached
		},
		upsertGeolocation: func(geo *models.Geolocation) error {
			if geo.IPAddress != testIP {
				t.Errorf("Expected IP %s, got %s", testIP, geo.IPAddress)
			}
			if geo.Country != "Canada" {
				t.Errorf("Expected country Canada, got %s", geo.Country)
			}
			return nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			return nil
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
			geoLookupCalled = true
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Country:   "Canada",
						City:      "Toronto",
						Latitude:  43.65107,
						Longitude: -79.347015,
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("test-session-2"),
		UserID:     intPtr(2),
		User:       "user2",
		IPAddress:  testIP,
		MediaType:  "episode",
		Title:      "Test Episode",
		Started:    time.Now().Unix(),
	}

	err := manager.processHistoryRecord(context.Background(), record)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !geoLookupCalled {
		t.Error("Expected GeoIP lookup to be called")
	}
}

func TestManager_ProcessHistoryRecord_GeolocationFailure_UsesFallback(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()
	geoCached := false

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return nil, nil // Not cached
		},
		upsertGeolocation: func(geo *models.Geolocation) error {
			// Should cache either fallback provider data OR unknown location
			// With multi-provider architecture (v2.0), ip-api.com fallback may succeed
			// for public IPs. The important thing is geolocation gets cached.
			geoCached = true
			return nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			return nil
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
			return nil, errors.New("geolocation service unavailable")
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("test-session-fail"),
		UserID:     intPtr(3),
		User:       "user3",
		IPAddress:  "203.0.113.0",
		MediaType:  "movie",
		Title:      "Test Movie",
		Started:    time.Now().Unix(),
	}

	// Should not fail - either fallback provider (ip-api.com) succeeds
	// or Unknown location is cached. Multi-provider GeoIP (v2.0) ensures
	// graceful degradation when primary source fails.
	err := manager.processHistoryRecord(context.Background(), record)
	if err != nil {
		t.Errorf("Expected no error with geolocation fallback, got: %v", err)
	}

	if !geoCached {
		t.Error("Expected geolocation to be cached (either from fallback provider or Unknown)")
	}
}

func TestManager_ProcessHistoryRecord_CompleteFields(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()
	var capturedEvent *models.PlaybackEvent

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ipAddress,
				Latitude:  40.7128,
				Longitude: -74.0060,
				Country:   "United States",
			}, nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			capturedEvent = event
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	stoppedTime := time.Now().Unix()
	record := &tautulli.TautulliHistoryRecord{
		SessionKey:        stringPtr("complete-session"),
		UserID:            intPtr(5),
		User:              "completeuser",
		IPAddress:         "192.168.100.50",
		MediaType:         "episode",
		Title:             "Episode Title",
		ParentTitle:       stringPtr("Season 1"),
		GrandparentTitle:  stringPtr("Show Name"),
		Platform:          "Android",
		Player:            "Plex for Android",
		Location:          "wan",
		Started:           time.Now().Unix() - 3600,
		Stopped:           stoppedTime,
		PercentComplete:   intPtr(100),
		PausedCounter:     intPtr(2),
		TranscodeDecision: "transcode",
		VideoResolution:   "1080p",
		VideoCodec:        "h264",
		AudioCodec:        "aac",
		SectionID:         intPtr(1),
		LibraryName:       "TV Shows",
		ContentRating:     "TV-14",
		Duration:          intPtr(3600),
		Year:              intPtr(2024),
	}

	err := manager.processHistoryRecord(context.Background(), record)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capturedEvent == nil {
		t.Fatal("Expected event to be captured")
	}

	// Verify all fields are populated
	if record.SessionKey == nil || capturedEvent.SessionKey != *record.SessionKey {
		t.Errorf("SessionKey mismatch: expected %v, got %s", record.SessionKey, capturedEvent.SessionKey)
	}

	if capturedEvent.Username != record.User {
		t.Errorf("Username mismatch: expected %s, got %s", record.User, capturedEvent.Username)
	}

	if record.ParentTitle != nil && (capturedEvent.ParentTitle == nil || *capturedEvent.ParentTitle != *record.ParentTitle) {
		t.Error("ParentTitle not set correctly")
	}

	if record.GrandparentTitle != nil && (capturedEvent.GrandparentTitle == nil || *capturedEvent.GrandparentTitle != *record.GrandparentTitle) {
		t.Error("GrandparentTitle not set correctly")
	}

	if capturedEvent.TranscodeDecision == nil || *capturedEvent.TranscodeDecision != record.TranscodeDecision {
		t.Error("TranscodeDecision not set correctly")
	}

	if capturedEvent.VideoResolution == nil || *capturedEvent.VideoResolution != record.VideoResolution {
		t.Error("VideoResolution not set correctly")
	}

	if capturedEvent.VideoCodec == nil || *capturedEvent.VideoCodec != record.VideoCodec {
		t.Error("VideoCodec not set correctly")
	}

	if capturedEvent.AudioCodec == nil || *capturedEvent.AudioCodec != record.AudioCodec {
		t.Error("AudioCodec not set correctly")
	}

	if capturedEvent.SectionID == nil || record.SectionID == nil || *capturedEvent.SectionID != *record.SectionID {
		t.Error("SectionID not set correctly")
	}

	if capturedEvent.LibraryName == nil || *capturedEvent.LibraryName != record.LibraryName {
		t.Error("LibraryName not set correctly")
	}

	if capturedEvent.ContentRating == nil || *capturedEvent.ContentRating != record.ContentRating {
		t.Error("ContentRating not set correctly")
	}

	expectedDuration := *record.Duration / 60
	if capturedEvent.PlayDuration == nil || *capturedEvent.PlayDuration != expectedDuration {
		t.Errorf("PlayDuration mismatch: expected %d, got %v", expectedDuration, capturedEvent.PlayDuration)
	}

	if capturedEvent.Year == nil || record.Year == nil || *capturedEvent.Year != *record.Year {
		t.Error("Year not set correctly")
	}

	if capturedEvent.StoppedAt == nil {
		t.Error("StoppedAt should be set")
	}

	// Verify UUID was generated
	if capturedEvent.ID == uuid.Nil {
		t.Error("Expected non-nil UUID for ID")
	}
}

func TestManager_ProcessHistoryRecord_NAValues(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	var capturedEvent *models.PlaybackEvent

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ipAddress,
				Latitude:  0,
				Longitude: 0,
				Country:   "Unknown",
			}, nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			capturedEvent = event
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:        stringPtr("na-values-session"),
		UserID:            intPtr(6),
		User:              "nauser",
		IPAddress:         "10.0.0.1",
		MediaType:         "movie",
		Title:             "Movie with NA values",
		TranscodeDecision: "N/A",
		VideoResolution:   "N/A",
		VideoCodec:        "N/A",
		AudioCodec:        "N/A",
		LibraryName:       "N/A",
		ContentRating:     "N/A",
		Started:           time.Now().Unix(),
	}

	err := manager.processHistoryRecord(context.Background(), record)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capturedEvent == nil {
		t.Fatal("Expected event to be captured")
	}

	// Verify N/A values are not set (should be nil)
	if capturedEvent.TranscodeDecision != nil {
		t.Error("TranscodeDecision should be nil for N/A value")
	}

	if capturedEvent.VideoResolution != nil {
		t.Error("VideoResolution should be nil for N/A value")
	}

	if capturedEvent.VideoCodec != nil {
		t.Error("VideoCodec should be nil for N/A value")
	}

	if capturedEvent.AudioCodec != nil {
		t.Error("AudioCodec should be nil for N/A value")
	}

	if capturedEvent.LibraryName != nil {
		t.Error("LibraryName should be nil for N/A value")
	}

	if capturedEvent.ContentRating != nil {
		t.Error("ContentRating should be nil for N/A value")
	}
}

func TestManager_ProcessHistoryRecord_DatabaseInsertError(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ipAddress,
				Latitude:  40.7128,
				Longitude: -74.0060,
				Country:   "United States",
			}, nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			return errors.New("database insert failed")
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("test-session"),
		UserID:     intPtr(1),
		User:       "testuser",
		IPAddress:  "192.168.1.1",
		MediaType:  "movie",
		Title:      "Test Movie",
		Started:    time.Now().Unix(),
	}

	err := manager.processHistoryRecord(context.Background(), record)
	if err == nil {
		t.Error("Expected error from database insert failure")
	}

	if !strings.Contains(err.Error(), "failed to insert playback event") {
		t.Errorf("Expected 'failed to insert playback event' error, got: %v", err)
	}
}

func TestManager_ProcessHistoryRecord_SessionKeyExistsError(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, errors.New("database query failed")
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("test-session"),
		UserID:     intPtr(1),
		User:       "testuser",
		IPAddress:  "192.168.1.1",
		MediaType:  "movie",
		Title:      "Test Movie",
		Started:    time.Now().Unix(),
	}

	err := manager.processHistoryRecord(context.Background(), record)
	if err == nil {
		t.Error("Expected error from session key check failure")
	}
}

func TestManager_ProcessHistoryRecord_GetGeolocationError(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return nil, errors.New("database read failed")
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
			// Also fail the lookup
			return nil, errors.New("geoip service unavailable")
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("test-session"),
		UserID:     intPtr(1),
		User:       "testuser",
		IPAddress:  "192.168.1.1",
		MediaType:  "movie",
		Title:      "Test Movie",
		Started:    time.Now().Unix(),
	}

	err := manager.processHistoryRecord(context.Background(), record)
	// Should still succeed using unknown location
	if err != nil {
		t.Errorf("Expected success with unknown location fallback, got error: %v", err)
	}
}
