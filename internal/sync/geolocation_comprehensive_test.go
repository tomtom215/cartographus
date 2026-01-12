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

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// TestResolveGeolocation tests the geolocation resolution flow
func TestResolveGeolocation(t *testing.T) {
	tests := []struct {
		name          string
		cachedGeo     *models.Geolocation
		getCachedErr  error
		fetchedGeo    *tautulli.TautulliGeoIP
		fetchErr      error
		expectSuccess bool
		expectLat     float64
		expectLon     float64
		expectCountry string
	}{
		{
			name: "cached geolocation returned",
			cachedGeo: &models.Geolocation{
				IPAddress: "203.0.113.1", // Use public IP (TEST-NET-3), private IPs get handled specially
				Latitude:  40.7128,
				Longitude: -74.0060,
				Country:   "United States",
			},
			getCachedErr:  nil,
			expectSuccess: true,
			expectLat:     40.7128,
			expectLon:     -74.0060,
			expectCountry: "United States",
		},
		{
			name:         "not cached, fetch succeeds",
			cachedGeo:    nil,
			getCachedErr: nil,
			fetchedGeo: &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Latitude:  51.5074,
						Longitude: -0.1278,
						Country:   "United Kingdom",
						City:      "London",
					},
				},
			},
			expectSuccess: true,
			expectLat:     51.5074,
			expectLon:     -0.1278,
			expectCountry: "United Kingdom",
		},
		{
			name:         "cache error, fallback to fetch",
			cachedGeo:    nil,
			getCachedErr: errors.New("cache error"),
			fetchedGeo: &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Latitude:  48.8566,
						Longitude: 2.3522,
						Country:   "France",
					},
				},
			},
			expectSuccess: true,
			expectLat:     48.8566,
			expectLon:     2.3522,
			expectCountry: "France",
		},
		// Note: "fetch error creates fallback" test case removed since v2.0 multi-provider
		// architecture. When Tautulli fails, ip-api.com fallback is tried automatically.
		// The fallback generally succeeds for public IPs, so the old behavior of returning
		// Unknown coordinates no longer applies. The fallback behavior is tested in
		// TestFetchAndCacheGeolocation_FallbackToExternalProvider instead.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Sync: config.SyncConfig{
					BatchSize:     10,
					RetryAttempts: 1,
					RetryDelay:    1 * time.Millisecond,
				},
			}

			mockDb := &mockDB{
				getGeolocation: func(ctx context.Context, ip string) (*models.Geolocation, error) {
					return tt.cachedGeo, tt.getCachedErr
				},
				upsertGeolocation: func(geo *models.Geolocation) error {
					return nil
				},
			}

			mockClient := &mockTautulliClient{
				getGeoIPLookup: func(ctx context.Context, ip string) (*tautulli.TautulliGeoIP, error) {
					if tt.fetchErr != nil {
						return nil, tt.fetchErr
					}
					return tt.fetchedGeo, nil
				},
			}

			manager := NewManager(mockDb, nil, mockClient, cfg, nil)

			record := &tautulli.TautulliHistoryRecord{
				SessionKey: stringPtr("test-session"),
				IPAddress:  "203.0.113.1", // Use public IP (TEST-NET-3), private IPs get handled specially
			}

			geo, err := manager.resolveGeolocation(context.Background(), record)

			// resolveGeolocation always returns success (with fallback if needed)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if geo == nil {
				t.Fatal("expected non-nil geolocation")
			}

			if geo.Latitude != tt.expectLat {
				t.Errorf("latitude: expected %.4f, got %.4f", tt.expectLat, geo.Latitude)
			}
			if geo.Longitude != tt.expectLon {
				t.Errorf("longitude: expected %.4f, got %.4f", tt.expectLon, geo.Longitude)
			}
			if geo.Country != tt.expectCountry {
				t.Errorf("country: expected %q, got %q", tt.expectCountry, geo.Country)
			}
		})
	}
}

// TestFetchAndCacheGeolocationAllOptionalFields tests that all optional fields are mapped
func TestFetchAndCacheGeolocationAllOptionalFields(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	var savedGeo *models.Geolocation
	mockDb := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			savedGeo = geo
			return nil
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ip string) (*tautulli.TautulliGeoIP, error) {
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Latitude:       37.7749,
						Longitude:      -122.4194,
						Country:        "United States",
						City:           "San Francisco",
						Region:         "California",
						PostalCode:     "94102",
						Timezone:       "America/Los_Angeles",
						AccuracyRadius: 10,
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	geo, err := manager.fetchAndCacheGeolocation(context.Background(), "192.168.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields are set
	if geo.IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress: expected 192.168.1.1, got %s", geo.IPAddress)
	}
	if geo.Latitude != 37.7749 {
		t.Errorf("Latitude: expected 37.7749, got %f", geo.Latitude)
	}
	if geo.Longitude != -122.4194 {
		t.Errorf("Longitude: expected -122.4194, got %f", geo.Longitude)
	}
	if geo.Country != "United States" {
		t.Errorf("Country: expected United States, got %s", geo.Country)
	}

	// Verify optional fields
	if geo.City == nil || *geo.City != "San Francisco" {
		t.Error("City not set correctly")
	}
	if geo.Region == nil || *geo.Region != "California" {
		t.Error("Region not set correctly")
	}
	if geo.PostalCode == nil || *geo.PostalCode != "94102" {
		t.Error("PostalCode not set correctly")
	}
	if geo.Timezone == nil || *geo.Timezone != "America/Los_Angeles" {
		t.Error("Timezone not set correctly")
	}
	if geo.AccuracyRadius == nil || *geo.AccuracyRadius != 10 {
		t.Error("AccuracyRadius not set correctly")
	}

	// Verify LastUpdated is recent
	if time.Since(geo.LastUpdated) > time.Minute {
		t.Error("LastUpdated should be recent")
	}

	// Verify it was saved
	if savedGeo == nil {
		t.Error("geolocation should have been saved to database")
	}
}

// TestFetchAndCacheGeolocationMinimalFields tests that minimal fields work
func TestFetchAndCacheGeolocationMinimalFields(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	mockDb := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			return nil
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ip string) (*tautulli.TautulliGeoIP, error) {
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Latitude:  10.0,
						Longitude: 20.0,
						Country:   "TestCountry",
						// All optional fields empty/zero
						City:           "",
						Region:         "",
						PostalCode:     "",
						Timezone:       "",
						AccuracyRadius: 0,
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	geo, err := manager.fetchAndCacheGeolocation(context.Background(), "192.168.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Optional fields should be nil when empty
	if geo.City != nil {
		t.Error("City should be nil for empty string")
	}
	if geo.Region != nil {
		t.Error("Region should be nil for empty string")
	}
	if geo.PostalCode != nil {
		t.Error("PostalCode should be nil for empty string")
	}
	if geo.Timezone != nil {
		t.Error("Timezone should be nil for empty string")
	}
	if geo.AccuracyRadius != nil {
		t.Error("AccuracyRadius should be nil for zero")
	}
}

// TestFetchAndCacheGeolocationCacheError tests cache upsert error handling
func TestFetchAndCacheGeolocationCacheError(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	cacheErr := errors.New("database error")
	mockDb := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			return cacheErr
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ip string) (*tautulli.TautulliGeoIP, error) {
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Latitude:  10.0,
						Longitude: 20.0,
						Country:   "TestCountry",
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	// Use public IP (TEST-NET-3) - private IPs get handled specially and skip the cache path
	_, err := manager.fetchAndCacheGeolocation(context.Background(), "203.0.113.2")

	if err == nil {
		t.Error("expected error when cache fails")
	}

	if !strings.Contains(err.Error(), "failed to cache geolocation") {
		t.Errorf("expected cache error message, got: %v", err)
	}
}

// TestFetchAndCacheGeolocationAPIError tests API error handling with retry
func TestFetchAndCacheGeolocationAPIError(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 2,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	mockDb := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			return nil
		},
	}

	callCount := 0
	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ip string) (*tautulli.TautulliGeoIP, error) {
			callCount++
			return nil, errors.New("API unavailable")
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	_, err := manager.fetchAndCacheGeolocation(context.Background(), "192.168.1.1")

	if err == nil {
		t.Error("expected error when API fails")
	}

	// Should have retried
	if callCount != 2 {
		t.Errorf("expected 2 API calls (with retry), got %d", callCount)
	}
}

// TestFetchAndCacheGeolocationRetrySuccess tests retry succeeding on second attempt
func TestFetchAndCacheGeolocationRetrySuccess(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 3,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	mockDb := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			return nil
		},
	}

	callCount := 0
	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ip string) (*tautulli.TautulliGeoIP, error) {
			callCount++
			if callCount < 2 {
				return nil, errors.New("temporary failure")
			}
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Latitude:  10.0,
						Longitude: 20.0,
						Country:   "TestCountry",
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	geo, err := manager.fetchAndCacheGeolocation(context.Background(), "192.168.1.1")

	if err != nil {
		t.Errorf("unexpected error after retry success: %v", err)
	}

	if geo == nil {
		t.Fatal("expected non-nil geolocation after retry success")
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

// TestValidateRecordSessionExists tests session existence check
func TestValidateRecordSessionExists(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return true, nil // Session already exists
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("existing-session"),
		IPAddress:  "192.168.1.1",
	}

	err := manager.validateRecord(context.Background(), record)

	if err == nil {
		t.Error("expected error for existing session")
	}

	if err.Error() != "session already processed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestValidateRecordInvalidIP tests IP validation
func TestValidateRecordInvalidIP(t *testing.T) {
	tests := []struct {
		name      string
		ipAddress string
		expectErr bool
	}{
		{
			name:      "empty IP",
			ipAddress: "",
			expectErr: true,
		},
		{
			name:      "N/A IP",
			ipAddress: "N/A",
			expectErr: true,
		},
		{
			name:      "valid IP",
			ipAddress: "192.168.1.1",
			expectErr: false,
		},
		{
			name:      "IPv6",
			ipAddress: "::1",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Sync: config.SyncConfig{
					BatchSize:     10,
					RetryAttempts: 1,
					RetryDelay:    1 * time.Millisecond,
				},
			}

			mockDb := &mockDB{
				sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
					return false, nil // Session doesn't exist
				},
			}

			manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

			record := &tautulli.TautulliHistoryRecord{
				SessionKey: stringPtr("new-session"),
				IPAddress:  tt.ipAddress,
			}

			err := manager.validateRecord(context.Background(), record)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error for invalid IP")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateRecordDatabaseError tests database error handling
func TestValidateRecordDatabaseError(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	dbErr := errors.New("database connection error")
	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, dbErr
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("test-session"),
		IPAddress:  "192.168.1.1",
	}

	err := manager.validateRecord(context.Background(), record)

	if err == nil {
		t.Error("expected error for database failure")
	}

	if !strings.Contains(err.Error(), "failed to check if session exists") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestProcessHistoryRecordSuccess tests successful record processing
func TestProcessHistoryRecordSuccess(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	var insertedEvent *models.PlaybackEvent
	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ip string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ip,
				Latitude:  37.7749,
				Longitude: -122.4194,
				Country:   "United States",
			}, nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			insertedEvent = event
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:      stringPtr("test-session-123"),
		Started:         time.Now().Unix(),
		UserID:          intPtr(42),
		User:            "testuser",
		IPAddress:       "192.168.1.100",
		MediaType:       "movie",
		Title:           "Test Movie",
		Platform:        "Chrome",
		Player:          "Plex Web",
		Location:        "lan",
		PercentComplete: intPtr(100),
	}

	err := manager.processHistoryRecord(context.Background(), record)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if insertedEvent == nil {
		t.Fatal("expected event to be inserted")
	}

	if insertedEvent.SessionKey != "test-session-123" {
		t.Errorf("SessionKey: expected test-session-123, got %s", insertedEvent.SessionKey)
	}
	if insertedEvent.Username != "testuser" {
		t.Errorf("Username: expected testuser, got %s", insertedEvent.Username)
	}
	if insertedEvent.Title != "Test Movie" {
		t.Errorf("Title: expected Test Movie, got %s", insertedEvent.Title)
	}
}

// TestProcessHistoryRecordSkipsExisting tests skipping existing sessions
func TestProcessHistoryRecordSkipsExisting(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	insertCalled := false
	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return true, nil // Session exists
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			insertCalled = true
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("existing-session"),
		Started:    time.Now().Unix(),
		IPAddress:  "192.168.1.1",
	}

	err := manager.processHistoryRecord(context.Background(), record)

	// Should return nil (skip gracefully)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if insertCalled {
		t.Error("insert should not be called for existing session")
	}
}

// TestProcessHistoryRecordWithGeoSuccess tests batch processing with pre-fetched geo
func TestProcessHistoryRecordWithGeoSuccess(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	var insertedEvent *models.PlaybackEvent
	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			insertedEvent = event
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:      stringPtr("test-session"),
		Started:         time.Now().Unix(),
		UserID:          intPtr(1),
		User:            "user1",
		IPAddress:       "10.0.0.1",
		MediaType:       "episode",
		Title:           "Pilot",
		Platform:        "Roku",
		Player:          "Roku Ultra",
		Location:        "wan",
		PercentComplete: intPtr(50),
	}

	geoMap := map[string]*models.Geolocation{
		"10.0.0.1": {
			IPAddress: "10.0.0.1",
			Latitude:  40.7128,
			Longitude: -74.0060,
			Country:   "United States",
		},
	}

	err := manager.processHistoryRecordWithGeo(context.Background(), record, geoMap)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if insertedEvent == nil {
		t.Fatal("expected event to be inserted")
	}

	if insertedEvent.SessionKey != "test-session" {
		t.Errorf("SessionKey mismatch")
	}
}

// TestProcessHistoryRecordWithGeoMissingCountry tests validation of geo with missing country
func TestProcessHistoryRecordWithGeoMissingCountry(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("test-session"),
		Started:    time.Now().Unix(),
		IPAddress:  "10.0.0.1",
	}

	geoMap := map[string]*models.Geolocation{
		"10.0.0.1": {
			IPAddress: "10.0.0.1",
			Latitude:  40.7128,
			Longitude: -74.0060,
			Country:   "", // Empty country
		},
	}

	err := manager.processHistoryRecordWithGeo(context.Background(), record, geoMap)

	if err == nil {
		t.Error("expected error for missing country")
	}

	if !strings.Contains(err.Error(), "invalid geolocation") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestProcessBatchEmptyRecords tests processing empty batch
func TestProcessBatchEmptyRecords(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	manager := NewManager(&mockDB{}, nil, &mockTautulliClient{}, cfg, nil)

	processed, err := manager.processBatch(context.Background(), []tautulli.TautulliHistoryRecord{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if processed != 0 {
		t.Errorf("expected 0 processed, got %d", processed)
	}
}

// TestProcessBatchWithRecords tests batch processing multiple records
func TestProcessBatchWithRecords(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	insertedCount := 0
	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocations: func(ctx context.Context, ips []string) (map[string]*models.Geolocation, error) {
			result := make(map[string]*models.Geolocation)
			for _, ip := range ips {
				result[ip] = &models.Geolocation{
					IPAddress: ip,
					Latitude:  37.7749,
					Longitude: -122.4194,
					Country:   "United States",
				}
			}
			return result, nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			insertedCount++
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	records := []tautulli.TautulliHistoryRecord{
		{SessionKey: stringPtr("session1"), Started: time.Now().Unix(), IPAddress: "192.168.1.1"},
		{SessionKey: stringPtr("session2"), Started: time.Now().Unix(), IPAddress: "192.168.1.2"},
		{SessionKey: stringPtr("session3"), Started: time.Now().Unix(), IPAddress: "192.168.1.3"},
	}

	processed, err := manager.processBatch(context.Background(), records)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if processed != 3 {
		t.Errorf("expected 3 processed, got %d", processed)
	}

	if insertedCount != 3 {
		t.Errorf("expected 3 inserts, got %d", insertedCount)
	}
}

// TestProcessBatchFallbackToIndividual tests fallback when batch geolocation fails
func TestProcessBatchFallbackToIndividual(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	insertedCount := 0
	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocations: func(ctx context.Context, ips []string) (map[string]*models.Geolocation, error) {
			return nil, errors.New("batch lookup failed") // Force fallback
		},
		getGeolocation: func(ctx context.Context, ip string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ip,
				Latitude:  37.7749,
				Longitude: -122.4194,
				Country:   "United States",
			}, nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			insertedCount++
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	records := []tautulli.TautulliHistoryRecord{
		{SessionKey: stringPtr("session1"), Started: time.Now().Unix(), IPAddress: "192.168.1.1"},
		{SessionKey: stringPtr("session2"), Started: time.Now().Unix(), IPAddress: "192.168.1.2"},
	}

	processed, err := manager.processBatch(context.Background(), records)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if processed != 2 {
		t.Errorf("expected 2 processed (fallback mode), got %d", processed)
	}
}

// TestProcessBatchFiltersNAIPs tests that N/A IPs are filtered out
func TestProcessBatchFiltersNAIPs(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	geoLookupIPs := make([]string, 0)
	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocations: func(ctx context.Context, ips []string) (map[string]*models.Geolocation, error) {
			geoLookupIPs = ips
			result := make(map[string]*models.Geolocation)
			for _, ip := range ips {
				result[ip] = &models.Geolocation{
					IPAddress: ip,
					Latitude:  37.7749,
					Longitude: -122.4194,
					Country:   "United States",
				}
			}
			return result, nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	records := []tautulli.TautulliHistoryRecord{
		{SessionKey: stringPtr("session1"), Started: time.Now().Unix(), IPAddress: "192.168.1.1"},
		{SessionKey: stringPtr("session2"), Started: time.Now().Unix(), IPAddress: "N/A"},      // Should be filtered
		{SessionKey: stringPtr("session3"), Started: time.Now().Unix(), IPAddress: ""},         // Should be filtered
		{SessionKey: stringPtr("session4"), Started: time.Now().Unix(), IPAddress: "10.0.0.1"}, // Valid
	}

	_, _ = manager.processBatch(context.Background(), records)

	// Only valid IPs should be looked up
	if len(geoLookupIPs) != 2 {
		t.Errorf("expected 2 IPs in geolocation lookup, got %d: %v", len(geoLookupIPs), geoLookupIPs)
	}
}

// TestProcessBatchDeduplicatesIPs tests that duplicate IPs are deduplicated
func TestProcessBatchDeduplicatesIPs(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	geoLookupIPs := make([]string, 0)
	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocations: func(ctx context.Context, ips []string) (map[string]*models.Geolocation, error) {
			geoLookupIPs = ips
			result := make(map[string]*models.Geolocation)
			for _, ip := range ips {
				result[ip] = &models.Geolocation{
					IPAddress: ip,
					Latitude:  37.7749,
					Longitude: -122.4194,
					Country:   "United States",
				}
			}
			return result, nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			return nil
		},
	}

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	// Multiple records with same IP
	records := []tautulli.TautulliHistoryRecord{
		{SessionKey: stringPtr("session1"), Started: time.Now().Unix(), IPAddress: "192.168.1.1"},
		{SessionKey: stringPtr("session2"), Started: time.Now().Unix(), IPAddress: "192.168.1.1"}, // Duplicate
		{SessionKey: stringPtr("session3"), Started: time.Now().Unix(), IPAddress: "192.168.1.1"}, // Duplicate
	}

	_, _ = manager.processBatch(context.Background(), records)

	// Only 1 unique IP should be looked up
	if len(geoLookupIPs) != 1 {
		t.Errorf("expected 1 unique IP in geolocation lookup, got %d: %v", len(geoLookupIPs), geoLookupIPs)
	}
}
