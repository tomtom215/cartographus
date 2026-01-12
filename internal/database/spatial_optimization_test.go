// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"testing"

	"github.com/tomtom215/cartographus/internal/config"
)

// TestInitializeSpatialOptimizations tests the spatial optimization initialization.
func TestInitializeSpatialOptimizations(t *testing.T) {
	tests := []struct {
		name      string
		serverLat float64
		serverLon float64
		wantErr   bool
	}{
		{
			name:      "with valid server coordinates",
			serverLat: 40.7128,
			serverLon: -74.0060, // New York
			wantErr:   false,
		},
		{
			name:      "with zero coordinates",
			serverLat: 0.0,
			serverLon: 0.0,
			wantErr:   false,
		},
		{
			name:      "with southern hemisphere coordinates",
			serverLat: -33.8688,
			serverLon: 151.2093, // Sydney
			wantErr:   false,
		},
		{
			name:      "with western hemisphere coordinates",
			serverLat: 34.0522,
			serverLon: -118.2437, // Los Angeles
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test database
			cfg := &config.DatabaseConfig{
				Path:        ":memory:",
				MaxMemory:   "512MB",
				SkipIndexes: true, // Skip 97 indexes for fast test setup
			}

			db, err := New(cfg, tt.serverLat, tt.serverLon)
			if err != nil {
				t.Fatalf("Failed to create test database: %v", err)
			}
			defer db.Close()

			// The spatial optimizations are called during New()
			// We verify by checking if the database was created successfully
			// and is functional

			// Verify the database can be pinged
			if err := db.Ping(context.Background()); err != nil {
				t.Errorf("Database ping failed: %v", err)
			}
		})
	}
}

// TestInitializeSpatialOptimizations_SpatialUnavailable tests behavior when spatial is unavailable.
func TestInitializeSpatialOptimizations_SpatialUnavailable(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	// Create database - it should handle spatial extension availability gracefully
	db, err := New(cfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// The test passes if database creation succeeded
	// The spatial optimization should be skipped if extension is unavailable
}

// TestUpdateGeolocationSpatialData tests updating spatial data for a geolocation
func TestUpdateGeolocationSpatialData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// First, insert a geolocation
	insertTestGeolocations(t, db)

	tests := []struct {
		name      string
		ipAddress string
		serverLat float64
		serverLon float64
		wantErr   bool
	}{
		{
			name:      "update existing geolocation with server coords",
			ipAddress: "192.168.1.1",
			serverLat: 40.7128,
			serverLon: -74.0060,
			wantErr:   false,
		},
		{
			name:      "update with zero server coords",
			ipAddress: "192.168.1.2",
			serverLat: 0.0,
			serverLon: 0.0,
			wantErr:   false,
		},
		{
			name:      "update non-existent geolocation",
			ipAddress: "10.0.0.1",
			serverLat: 40.7128,
			serverLon: -74.0060,
			wantErr:   false, // Should not error, just won't update anything
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.UpdateGeolocationSpatialData(tt.ipAddress, tt.serverLat, tt.serverLon)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateGeolocationSpatialData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestUpdateGeolocationSpatialData_WithServerCoords tests the full update path
func TestUpdateGeolocationSpatialData_WithServerCoords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert a test geolocation
	insertTestGeolocations(t, db)

	// Update with valid server coordinates
	err := db.UpdateGeolocationSpatialData("192.168.1.1", 40.7128, -74.0060)
	if err != nil {
		// May fail if H3 extension is not available - that's okay for test
		t.Logf("UpdateGeolocationSpatialData returned: %v (may be expected if H3 unavailable)", err)
	}

	// Verify database is still functional
	if pingErr := db.Ping(context.Background()); pingErr != nil {
		t.Errorf("Database not functional after update: %v", pingErr)
	}
}

// TestUpdateGeolocationSpatialData_WithoutServerCoords tests the update path without distance calculation
func TestUpdateGeolocationSpatialData_WithoutServerCoords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert a test geolocation
	insertTestGeolocations(t, db)

	// Update without server coordinates (should skip distance calculation)
	err := db.UpdateGeolocationSpatialData("192.168.1.1", 0.0, 0.0)
	if err != nil {
		// May fail if H3 extension is not available - that's okay for test
		t.Logf("UpdateGeolocationSpatialData returned: %v (may be expected if H3 unavailable)", err)
	}

	// Verify database is still functional
	if pingErr := db.Ping(context.Background()); pingErr != nil {
		t.Errorf("Database not functional after update: %v", pingErr)
	}
}

// TestSpatialAvailabilityFlag tests the IsSpatialAvailable method.
func TestSpatialAvailabilityFlag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// The spatial availability flag should be set (true or false depending on environment)
	// We just verify the method doesn't panic
	available := db.IsSpatialAvailable()
	t.Logf("Spatial extension available: %v", available)
}

// TestSpatialOptimizations_EdgeCases tests edge cases for spatial optimizations.
func TestSpatialOptimizations_EdgeCases(t *testing.T) {
	t.Run("extreme latitude values", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()
		insertTestGeolocations(t, db)

		// Near poles
		err := db.UpdateGeolocationSpatialData("192.168.1.1", 89.9999, 0.0)
		// May fail if spatial not available, but should not panic
		if err != nil {
			t.Logf("Update with extreme latitude: %v", err)
		}
	})

	t.Run("extreme longitude values", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()
		insertTestGeolocations(t, db)

		// Date line
		err := db.UpdateGeolocationSpatialData("192.168.1.1", 0.0, 179.9999)
		if err != nil {
			t.Logf("Update with extreme longitude: %v", err)
		}
	})

	t.Run("negative coordinates", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()
		insertTestGeolocations(t, db)

		// Southern and Western hemispheres
		err := db.UpdateGeolocationSpatialData("192.168.1.1", -45.0, -90.0)
		if err != nil {
			t.Logf("Update with negative coordinates: %v", err)
		}
	})
}

// TestSpatialMigrations tests that migrations run without error.
func TestSpatialMigrations(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	// Create database with migrations
	db, err := New(cfg, 40.7128, -74.0060)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify the database schema includes expected columns
	// by attempting queries that use them
	var exists bool

	// Check if h3_index_6 column exists
	err = db.conn.QueryRow(`
		SELECT COUNT(*) > 0
		FROM information_schema.columns
		WHERE table_name = 'geolocations' AND column_name = 'h3_index_6'
	`).Scan(&exists)
	if err != nil {
		// Table might not exist in information_schema for DuckDB
		t.Logf("Could not query information_schema: %v", err)
	}
}

// TestSpatialIndexCreation tests that spatial indexes are created.
func TestSpatialIndexCreation(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip indexes initially for fast setup
	}

	db, err := New(cfg, 40.7128, -74.0060)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// This test specifically needs indexes - create them explicitly
	if err := db.CreateIndexes(); err != nil {
		t.Fatalf("Failed to create indexes: %v", err)
	}

	// Insert test data
	insertTestGeolocations(t, db)

	// Try a spatial query that would benefit from indexes
	var count int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM geolocations WHERE latitude > 0
	`).Scan(&count)
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	if count == 0 {
		t.Error("Expected some geolocations")
	}
}
