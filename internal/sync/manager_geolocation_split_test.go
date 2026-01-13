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

	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestManager_FetchAndCacheGeolocation_TautulliSuccess(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	geoCached := false
	mockDB := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			geoCached = true
			if geo.IPAddress != "203.0.113.1" {
				t.Errorf("Expected IP 203.0.113.1, got %s", geo.IPAddress)
			}
			if geo.Country != "Japan" {
				t.Errorf("Expected country Japan, got %s", geo.Country)
			}
			// City field is not set by fetchAndCacheGeolocation
			return nil
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Country:   "Japan",
						City:      "Tokyo",
						Region:    "Tokyo",
						Latitude:  35.6762,
						Longitude: 139.6503,
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	geo, err := manager.fetchAndCacheGeolocation(context.Background(), "203.0.113.1")
	if err != nil {
		t.Fatalf("fetchAndCacheGeolocation failed: %v", err)
	}

	if !geoCached {
		t.Error("Expected geolocation to be cached")
	}

	if geo.Country != "Japan" {
		t.Errorf("Expected country Japan, got %s", geo.Country)
	}
}

func TestManager_FetchAndCacheGeolocation_TautulliRetry(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	attemptCount := 0
	mockDB := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			return nil
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
			attemptCount++
			if attemptCount < 2 {
				return nil, errors.New("temporary network error")
			}
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Country:   "United States",
						Latitude:  37.7749,
						Longitude: -122.4194,
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	_, err := manager.fetchAndCacheGeolocation(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("fetchAndCacheGeolocation should succeed after retry: %v", err)
	}

	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts (1 failure + 1 success), got %d", attemptCount)
	}
}

func TestManager_FetchAndCacheGeolocation_DatabaseError(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	mockDB := &mockDB{
		upsertGeolocation: func(geo *models.Geolocation) error {
			return errors.New("database write failed")
		},
	}

	mockClient := &mockTautulliClient{
		getGeoIPLookup: func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Country:   "Canada",
						Latitude:  43.65107,
						Longitude: -79.347015,
					},
				},
			}, nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	// Use a public IP address (not private like 10.0.0.1) to test the database caching error path
	// Private IPs are now handled specially and don't reach the caching step
	_, err := manager.fetchAndCacheGeolocation(context.Background(), "203.0.113.50")
	if err == nil {
		t.Error("Expected error when database write fails")
	}

	if !strings.Contains(err.Error(), "failed to cache geolocation") {
		t.Errorf("Expected 'failed to cache geolocation' error, got: %v", err)
	}
}
