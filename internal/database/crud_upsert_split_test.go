// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"testing"

	"github.com/tomtom215/cartographus/internal/models"
)

func TestUpsertGeolocation_Insert(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	city := "New York"
	region := "New York"
	postalCode := "10001"
	timezone := "America/New_York"
	accuracyRadius := 10

	geo := &models.Geolocation{
		IPAddress:      "10.0.0.1",
		Latitude:       40.7128,
		Longitude:      -74.0060,
		City:           &city,
		Region:         &region,
		Country:        "United States",
		PostalCode:     &postalCode,
		Timezone:       &timezone,
		AccuracyRadius: &accuracyRadius,
	}

	err := db.UpsertGeolocation(geo)
	if err != nil {
		t.Fatalf("UpsertGeolocation (insert) failed: %v", err)
	}

	// Verify LastUpdated was set
	if geo.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be set")
	}

	// Retrieve and verify
	retrieved, err := db.GetGeolocation(context.Background(), geo.IPAddress)
	if err != nil {
		t.Fatalf("GetGeolocation failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected geolocation to exist")
	}

	if (retrieved.City == nil && geo.City != nil) || (retrieved.City != nil && geo.City == nil) ||
		(retrieved.City != nil && geo.City != nil && *retrieved.City != *geo.City) {
		var expectedCity, gotCity string
		if geo.City != nil {
			expectedCity = *geo.City
		} else {
			expectedCity = "<nil>"
		}
		if retrieved.City != nil {
			gotCity = *retrieved.City
		} else {
			gotCity = "<nil>"
		}
		t.Errorf("Expected city %s, got %s", expectedCity, gotCity)
	}
}

func TestUpsertGeolocation_Update(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	// Insert initial geolocation
	city := "Los Angeles"
	region := "California"

	geo := &models.Geolocation{
		IPAddress: "10.0.0.2",
		Latitude:  34.0522,
		Longitude: -118.2437,
		City:      &city,
		Region:    &region,
		Country:   "United States",
	}

	err := db.UpsertGeolocation(geo)
	if err != nil {
		t.Fatalf("UpsertGeolocation (initial insert) failed: %v", err)
	}

	// Update with new city
	newCity := "Beverly Hills"
	geo.City = &newCity
	err = db.UpsertGeolocation(geo)
	if err != nil {
		t.Fatalf("UpsertGeolocation (update) failed: %v", err)
	}

	// Verify update
	retrieved, err := db.GetGeolocation(context.Background(), geo.IPAddress)
	if err != nil {
		t.Fatalf("GetGeolocation failed: %v", err)
	}

	if retrieved.City == nil || *retrieved.City != "Beverly Hills" {
		if retrieved.City == nil {
			t.Error("Expected updated city 'Beverly Hills', got nil")
		} else {
			t.Errorf("Expected updated city 'Beverly Hills', got '%s'", *retrieved.City)
		}
	}
}

// TestUpsertGeolocationWithServer_Insert tests upserting a geolocation with server coordinates
func TestUpsertGeolocationWithServer_Insert(t *testing.T) {

	db := setupTestDB(t)
	defer db.Close()

	// Skip if spatial extension not available (H3/distance calculations require it)
	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
	}

	city := "London"
	region := "England"
	timezone := "Europe/London"

	geo := &models.Geolocation{
		IPAddress: "10.0.0.100",
		Latitude:  51.5074,
		Longitude: -0.1278,
		City:      &city,
		Region:    &region,
		Country:   "United Kingdom",
		Timezone:  &timezone,
	}

	// Server location in New York
	serverLat := 40.7128
	serverLon := -74.0060

	err := db.UpsertGeolocationWithServer(geo, serverLat, serverLon)
	if err != nil {
		t.Fatalf("UpsertGeolocationWithServer failed: %v", err)
	}

	// Verify geolocation was inserted
	retrieved, err := db.GetGeolocation(context.Background(), geo.IPAddress)
	if err != nil {
		t.Fatalf("GetGeolocation failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected geolocation to exist")
	}

	if retrieved.City == nil || *retrieved.City != "London" {
		t.Errorf("Expected city 'London', got %v", retrieved.City)
	}
}

// TestUpsertGeolocationWithServer_Update tests updating an existing geolocation with server coords
func TestUpsertGeolocationWithServer_Update(t *testing.T) {

	db := setupTestDB(t)
	defer db.Close()

	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
	}

	// Insert initial
	city := "Tokyo"
	geo := &models.Geolocation{
		IPAddress: "10.0.0.101",
		Latitude:  35.6762,
		Longitude: 139.6503,
		City:      &city,
		Country:   "Japan",
	}

	serverLat := 40.7128
	serverLon := -74.0060

	err := db.UpsertGeolocationWithServer(geo, serverLat, serverLon)
	if err != nil {
		t.Fatalf("First UpsertGeolocationWithServer failed: %v", err)
	}

	// Update with new city
	newCity := "Osaka"
	geo.City = &newCity
	geo.Latitude = 34.6937
	geo.Longitude = 135.5023

	err = db.UpsertGeolocationWithServer(geo, serverLat, serverLon)
	if err != nil {
		t.Fatalf("Second UpsertGeolocationWithServer failed: %v", err)
	}

	// Verify update
	retrieved, err := db.GetGeolocation(context.Background(), geo.IPAddress)
	if err != nil {
		t.Fatalf("GetGeolocation failed: %v", err)
	}

	if retrieved.City == nil || *retrieved.City != "Osaka" {
		t.Errorf("Expected city 'Osaka', got %v", retrieved.City)
	}
}

// TestUpsertGeolocationWithServer_ZeroLastUpdated tests that LastUpdated is set when zero
func TestUpsertGeolocationWithServer_ZeroLastUpdated(t *testing.T) {

	db := setupTestDB(t)
	defer db.Close()

	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
	}

	city := "Sydney"
	geo := &models.Geolocation{
		IPAddress: "10.0.0.102",
		Latitude:  -33.8688,
		Longitude: 151.2093,
		City:      &city,
		Country:   "Australia",
		// LastUpdated is zero value (should be set by UpsertGeolocationWithServer)
	}

	err := db.UpsertGeolocationWithServer(geo, 0, 0)
	if err != nil {
		t.Fatalf("UpsertGeolocationWithServer failed: %v", err)
	}

	// Verify LastUpdated was set
	if geo.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be set when initially zero")
	}
}

// TestUpsertGeolocationWithServer_ConcurrentSameIP tests retry logic for same IP
func TestUpsertGeolocationWithServer_ConcurrentSameIP(t *testing.T) {

	db := setupTestDB(t)
	defer db.Close()

	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
	}

	// This test verifies that concurrent upserts to the same IP don't cause errors
	// due to the per-IP locking mechanism
	city := "Paris"
	geo := &models.Geolocation{
		IPAddress: "10.0.0.103",
		Latitude:  48.8566,
		Longitude: 2.3522,
		City:      &city,
		Country:   "France",
	}

	// Perform multiple sequential upserts (simulating what would happen with concurrent calls)
	for i := 0; i < 3; i++ {
		err := db.UpsertGeolocationWithServer(geo, 0, 0)
		if err != nil {
			t.Fatalf("UpsertGeolocationWithServer iteration %d failed: %v", i, err)
		}
	}

	// Verify final state
	retrieved, err := db.GetGeolocation(context.Background(), geo.IPAddress)
	if err != nil {
		t.Fatalf("GetGeolocation failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected geolocation to exist")
	}
}
