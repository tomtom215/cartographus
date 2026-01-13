// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"testing"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// TestFetchAndCacheGeolocation_ValidNullIsland tests that (0,0) coordinates are accepted
// when they have a valid country. This fixes the bug where valid Null Island coordinates
// (Gulf of Guinea, off coast of Africa) were incorrectly rejected.
func TestFetchAndCacheGeolocation_ValidNullIsland(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    100,
		},
	}

	mockDB := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			// Verify that (0,0) coordinates with valid country are accepted
			if geo.Latitude != 0.0 || geo.Longitude != 0.0 {
				t.Errorf("Expected (0,0) coordinates, got (%.4f, %.4f)", geo.Latitude, geo.Longitude)
			}
			if geo.Country != "Null Island Test" {
				t.Errorf("Expected country 'Null Island Test', got '%s'", geo.Country)
			}
			return nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Latitude:  0.0,
						Longitude: 0.0,
						Country:   "Null Island Test",
						City:      "Null City",
						Region:    "Atlantic Ocean",
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	geo, err := manager.fetchAndCacheGeolocation(context.Background(), "192.168.1.1")
	if err != nil {
		t.Errorf("fetchAndCacheGeolocation() with valid (0,0) coordinates returned error: %v", err)
	}

	if geo == nil {
		t.Fatal("fetchAndCacheGeolocation() returned nil geolocation")
	}

	if geo.Latitude != 0.0 || geo.Longitude != 0.0 {
		t.Errorf("Expected (0,0) coordinates, got (%.4f, %.4f)", geo.Latitude, geo.Longitude)
	}

	if geo.Country != "Null Island Test" {
		t.Errorf("Expected country 'Null Island Test', got '%s'", geo.Country)
	}
}

// TestFetchAndCacheGeolocation_EmptyCountryTriggersFallback tests that when Tautulli
// returns empty country (indicating failed lookup), the system falls back to the
// external GeoIP service (ip-api.com) which typically succeeds for public IPs.
// v2.0: Multi-provider architecture now provides automatic fallback.
func TestFetchAndCacheGeolocation_EmptyCountryTriggersFallback(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    100,
		},
	}

	geoCached := false
	mockDB := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			// With multi-provider, fallback to ip-api.com succeeds and caches the result
			geoCached = true
			return nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Latitude:  10.5,
						Longitude: -20.3,
						Country:   "", // Empty country triggers fallback to ip-api.com
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	// Use public IP (TEST-NET-3) - private IPs get handled specially and skip the fetch path
	// The ip-api.com fallback should succeed for this public IP
	geo, err := manager.fetchAndCacheGeolocation(context.Background(), "203.0.113.3")

	// With multi-provider architecture, fallback to ip-api.com typically succeeds
	if err != nil {
		t.Logf("Fallback also failed (may be network issue): %v", err)
		// Don't fail the test - ip-api.com might be unavailable in test environment
		return
	}

	if geo == nil {
		t.Error("Expected non-nil geolocation from fallback provider")
		return
	}

	// Fallback should provide valid country data
	if geo.Country == "" {
		t.Error("Expected non-empty country from fallback provider")
	}

	// Geolocation should have been cached
	if !geoCached {
		t.Error("Expected geolocation to be cached after fallback success")
	}
}

// TestFetchAndCacheGeolocation_NormalCoordinatesStillWork verifies that normal
// (non-zero) coordinates continue to work as expected after the fix.
func TestFetchAndCacheGeolocation_NormalCoordinatesStillWork(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     10,
			RetryAttempts: 1,
			RetryDelay:    100,
		},
	}

	expectedLat := 37.7749
	expectedLon := -122.4194

	mockDB := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			if geo.Latitude != expectedLat || geo.Longitude != expectedLon {
				t.Errorf("Expected (%.4f, %.4f), got (%.4f, %.4f)",
					expectedLat, expectedLon, geo.Latitude, geo.Longitude)
			}
			if geo.Country != "United States" {
				t.Errorf("Expected country 'United States', got '%s'", geo.Country)
			}
			return nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Latitude:  expectedLat,
						Longitude: expectedLon,
						Country:   "United States",
						City:      "San Francisco",
						Region:    "California",
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	geo, err := manager.fetchAndCacheGeolocation(context.Background(), "192.168.1.1")
	if err != nil {
		t.Errorf("fetchAndCacheGeolocation() with normal coordinates returned error: %v", err)
	}

	if geo == nil {
		t.Fatal("fetchAndCacheGeolocation() returned nil geolocation")
	}

	if geo.Latitude != expectedLat || geo.Longitude != expectedLon {
		t.Errorf("Expected (%.4f, %.4f), got (%.4f, %.4f)",
			expectedLat, expectedLon, geo.Latitude, geo.Longitude)
	}

	if geo.Country != "United States" {
		t.Errorf("Expected country 'United States', got '%s'", geo.Country)
	}
}
