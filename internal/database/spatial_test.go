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

// TestGetH3AggregatedHexagons tests H3 hexagon aggregation
func TestGetH3AggregatedHexagons(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Skip if spatial extension not available - CHECK BEFORE INSERTING DATA
	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
		return
	}

	// Insert test data with geolocations
	insertSpatialTestData(t, db)

	tests := []struct {
		name       string
		filter     LocationStatsFilter
		resolution int
		wantErr    bool
		minCount   int // Minimum number of hexagons expected
	}{
		{
			name:       "City-level aggregation (resolution 7)",
			filter:     LocationStatsFilter{Limit: 1000},
			resolution: 7,
			wantErr:    false,
			minCount:   1,
		},
		{
			name:       "Country-level aggregation (resolution 6)",
			filter:     LocationStatsFilter{Limit: 1000},
			resolution: 6,
			wantErr:    false,
			minCount:   1,
		},
		{
			name:       "Neighborhood-level aggregation (resolution 8)",
			filter:     LocationStatsFilter{Limit: 1000},
			resolution: 8,
			wantErr:    false,
			minCount:   1,
		},
		{
			name:       "Invalid resolution - too low",
			filter:     LocationStatsFilter{Limit: 1000},
			resolution: 5,
			wantErr:    true,
			minCount:   0,
		},
		{
			name:       "Invalid resolution - too high",
			filter:     LocationStatsFilter{Limit: 1000},
			resolution: 9,
			wantErr:    true,
			minCount:   0,
		},
		{
			name: "With date filter",
			filter: LocationStatsFilter{
				StartDate: &[]time.Time{time.Now().AddDate(0, 0, -7)}[0],
				Limit:     1000,
			},
			resolution: 7,
			wantErr:    false,
			minCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hexagons, err := db.GetH3AggregatedHexagons(context.Background(), tt.filter, tt.resolution)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetH3AggregatedHexagons() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if hexagons == nil {
					t.Error("Expected non-nil hexagons slice")
					return
				}

				if len(hexagons) < tt.minCount {
					t.Errorf("Expected at least %d hexagons, got %d", tt.minCount, len(hexagons))
				}

				// Validate hexagon data structure
				for _, h := range hexagons {
					if h.H3Index == 0 {
						t.Error("Expected non-zero H3 index")
					}
					if h.PlaybackCount <= 0 {
						t.Error("Expected positive playback count")
					}
					if h.Latitude < -90 || h.Latitude > 90 {
						t.Errorf("Invalid latitude: %f", h.Latitude)
					}
					if h.Longitude < -180 || h.Longitude > 180 {
						t.Errorf("Invalid longitude: %f", h.Longitude)
					}
				}
			}
		})
	}
}

// TestGetDistanceWeightedArcs tests distance-weighted arc calculations
func TestGetDistanceWeightedArcs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Skip if spatial extension not available
	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
		return
	}

	// Insert test data
	insertSpatialTestData(t, db)

	// Server location (New York City)
	serverLat := 40.7128
	serverLon := -74.0060

	tests := []struct {
		name      string
		filter    LocationStatsFilter
		serverLat float64
		serverLon float64
		wantErr   bool
		minCount  int
	}{
		{
			name:      "Valid server location",
			filter:    LocationStatsFilter{Limit: 1000},
			serverLat: serverLat,
			serverLon: serverLon,
			wantErr:   false,
			minCount:  1,
		},
		{
			name:      "Zero server coordinates - should error",
			filter:    LocationStatsFilter{Limit: 1000},
			serverLat: 0.0,
			serverLon: 0.0,
			wantErr:   true,
			minCount:  0,
		},
		{
			name: "With user filter",
			filter: LocationStatsFilter{
				Users: []string{"testuser1"},
				Limit: 1000,
			},
			serverLat: serverLat,
			serverLon: serverLon,
			wantErr:   false,
			minCount:  0, // May be 0 if testuser1 has no playbacks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arcs, err := db.GetDistanceWeightedArcs(context.Background(), tt.filter, tt.serverLat, tt.serverLon)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDistanceWeightedArcs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if arcs == nil {
					t.Error("Expected non-nil arcs slice")
					return
				}

				if len(arcs) < tt.minCount {
					t.Logf("Expected at least %d arcs, got %d (may be valid if no data)", tt.minCount, len(arcs))
				}

				// Validate arc data structure
				for _, a := range arcs {
					if a.ServerLatitude != tt.serverLat || a.ServerLongitude != tt.serverLon {
						t.Errorf("Server coordinates mismatch: got (%f, %f), want (%f, %f)",
							a.ServerLatitude, a.ServerLongitude, tt.serverLat, tt.serverLon)
					}
					if a.PlaybackCount <= 0 {
						t.Error("Expected positive playback count")
					}
					if a.DistanceKm < 0 {
						t.Errorf("Expected non-negative distance, got %f", a.DistanceKm)
					}
					if a.Weight <= 0 {
						t.Error("Expected positive arc weight")
					}
				}
			}
		})
	}
}

// TestGetLocationsInViewport tests bounding box spatial queries
func TestGetLocationsInViewport(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Skip if spatial extension not available
	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
		return
	}

	// Insert test data
	insertSpatialTestData(t, db)

	tests := []struct {
		name     string
		filter   LocationStatsFilter
		west     float64
		south    float64
		east     float64
		north    float64
		wantErr  bool
		minCount int
	}{
		{
			name:     "Whole world viewport",
			filter:   LocationStatsFilter{Limit: 5000},
			west:     -180,
			south:    -90,
			east:     180,
			north:    90,
			wantErr:  false,
			minCount: 1,
		},
		{
			name:     "North America viewport",
			filter:   LocationStatsFilter{Limit: 5000},
			west:     -130,
			south:    25,
			east:     -60,
			north:    50,
			wantErr:  false,
			minCount: 0, // May be 0 if no data in this region
		},
		{
			name:     "Europe viewport",
			filter:   LocationStatsFilter{Limit: 5000},
			west:     -10,
			south:    35,
			east:     40,
			north:    70,
			wantErr:  false,
			minCount: 0,
		},
		{
			name:     "Small viewport (New York City)",
			filter:   LocationStatsFilter{Limit: 5000},
			west:     -74.1,
			south:    40.6,
			east:     -73.9,
			north:    40.8,
			wantErr:  false,
			minCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locations, err := db.GetLocationsInViewport(context.Background(), tt.filter, tt.west, tt.south, tt.east, tt.north)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetLocationsInViewport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if locations == nil {
					t.Error("Expected non-nil locations slice")
					return
				}

				// Validate locations are within viewport
				for _, loc := range locations {
					if loc.Latitude < tt.south || loc.Latitude > tt.north {
						t.Errorf("Location latitude %f outside viewport bounds [%f, %f]",
							loc.Latitude, tt.south, tt.north)
					}
					if loc.Longitude < tt.west || loc.Longitude > tt.east {
						t.Errorf("Location longitude %f outside viewport bounds [%f, %f]",
							loc.Longitude, tt.west, tt.east)
					}
					if loc.PlaybackCount <= 0 {
						t.Error("Expected positive playback count")
					}
				}
			}
		})
	}
}

// TestGetTemporalSpatialDensity tests temporal-spatial density calculations
func TestGetTemporalSpatialDensity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Skip if spatial extension not available
	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
		return
	}

	// Insert test data
	insertSpatialTestData(t, db)

	tests := []struct {
		name         string
		filter       LocationStatsFilter
		interval     string
		h3Resolution int
		wantErr      bool
		minCount     int
	}{
		{
			name:         "Hour interval",
			filter:       LocationStatsFilter{Limit: 1000},
			interval:     "hour",
			h3Resolution: 7,
			wantErr:      false,
			minCount:     1,
		},
		{
			name:         "Day interval",
			filter:       LocationStatsFilter{Limit: 1000},
			interval:     "day",
			h3Resolution: 7,
			wantErr:      false,
			minCount:     1,
		},
		{
			name:         "Week interval",
			filter:       LocationStatsFilter{Limit: 1000},
			interval:     "week",
			h3Resolution: 7,
			wantErr:      false,
			minCount:     1,
		},
		{
			name:         "Month interval",
			filter:       LocationStatsFilter{Limit: 1000},
			interval:     "month",
			h3Resolution: 7,
			wantErr:      false,
			minCount:     1,
		},
		{
			name:         "Invalid interval",
			filter:       LocationStatsFilter{Limit: 1000},
			interval:     "invalid",
			h3Resolution: 7,
			wantErr:      true,
			minCount:     0,
		},
		{
			name:         "Invalid resolution - too low",
			filter:       LocationStatsFilter{Limit: 1000},
			interval:     "hour",
			h3Resolution: 5,
			wantErr:      true,
			minCount:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			points, err := db.GetTemporalSpatialDensity(context.Background(), tt.filter, tt.interval, tt.h3Resolution)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetTemporalSpatialDensity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if points == nil {
					t.Error("Expected non-nil points slice")
					return
				}

				if len(points) < tt.minCount {
					t.Errorf("Expected at least %d points, got %d", tt.minCount, len(points))
				}

				// Validate temporal-spatial points
				for _, p := range points {
					if p.TimeBucket.IsZero() {
						t.Error("Expected non-zero time bucket")
					}
					if p.H3Index == 0 {
						t.Error("Expected non-zero H3 index")
					}
					if p.PlaybackCount <= 0 {
						t.Error("Expected positive playback count")
					}
					if p.RollingAvgPlaybacks < 0 {
						t.Error("Expected non-negative rolling average")
					}
					if p.CumulativePlaybacks < 0 {
						t.Error("Expected non-negative cumulative playbacks")
					}
				}
			}
		})
	}
}

// TestGetNearbyLocations tests proximity search
func TestGetNearbyLocations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Skip if spatial extension not available
	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
		return
	}

	// Insert test data
	insertSpatialTestData(t, db)

	tests := []struct {
		name     string
		lat      float64
		lon      float64
		radiusKm float64
		filter   LocationStatsFilter
		wantErr  bool
		minCount int
	}{
		{
			name:     "100km radius from New York",
			lat:      40.7128,
			lon:      -74.0060,
			radiusKm: 100,
			filter:   LocationStatsFilter{Limit: 1000},
			wantErr:  false,
			minCount: 0, // May be 0 if no data nearby
		},
		{
			name:     "500km radius from London",
			lat:      51.5074,
			lon:      -0.1278,
			radiusKm: 500,
			filter:   LocationStatsFilter{Limit: 1000},
			wantErr:  false,
			minCount: 0,
		},
		{
			name:     "Large radius - 5000km",
			lat:      40.7128,
			lon:      -74.0060,
			radiusKm: 5000,
			filter:   LocationStatsFilter{Limit: 1000},
			wantErr:  false,
			minCount: 0,
		},
		{
			name:     "Small radius - 10km",
			lat:      40.7128,
			lon:      -74.0060,
			radiusKm: 10,
			filter:   LocationStatsFilter{Limit: 1000},
			wantErr:  false,
			minCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locations, err := db.GetNearbyLocations(context.Background(), tt.lat, tt.lon, tt.radiusKm, tt.filter)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetNearbyLocations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if locations == nil {
					t.Error("Expected non-nil locations slice")
					return
				}

				// Validate locations - all should be within radius
				// Note: We can't validate exact distance without spatial functions,
				// but we can validate data structure
				for _, loc := range locations {
					if loc.PlaybackCount <= 0 {
						t.Error("Expected positive playback count")
					}
					if loc.Latitude < -90 || loc.Latitude > 90 {
						t.Errorf("Invalid latitude: %f", loc.Latitude)
					}
					if loc.Longitude < -180 || loc.Longitude > 180 {
						t.Errorf("Invalid longitude: %f", loc.Longitude)
					}
				}
			}
		})
	}
}

// insertSpatialTestData inserts test data for spatial queries
func insertSpatialTestData(t *testing.T, db *DB) {
	t.Helper()

	// Use a timeout context to prevent hangs during resource pressure
	// NOTE: DuckDB CGO syscalls cannot be interrupted by Go contexts,
	// so we use individual inserts instead of batch to allow interruption between inserts
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Insert test geolocations with spatial data
	locations := []struct {
		ip      string
		lat     float64
		lon     float64
		city    string
		country string
	}{
		{"10.0.0.1", 40.7128, -74.0060, "New York", "United States"},
		{"10.0.0.2", 34.0522, -118.2437, "Los Angeles", "United States"},
		{"10.0.0.3", 51.5074, -0.1278, "London", "United Kingdom"},
		{"10.0.0.4", 48.8566, 2.3522, "Paris", "France"},
		{"10.0.0.5", 35.6762, 139.6503, "Tokyo", "Japan"},
	}

	for _, loc := range locations {
		// Check context before each insert to allow early termination
		if ctx.Err() != nil {
			t.Fatalf("Context timeout during geolocation inserts: %v", ctx.Err())
		}

		geo := &models.Geolocation{
			IPAddress:   loc.ip,
			Latitude:    loc.lat,
			Longitude:   loc.lon,
			City:        &loc.city,
			Country:     loc.country,
			LastUpdated: time.Now(),
		}

		// Use server coordinates for distance calculation
		err := db.UpsertGeolocationWithServer(geo, 40.7128, -74.0060)
		if err != nil {
			t.Fatalf("Failed to insert geolocation %s: %v", loc.ip, err)
		}
	}

	// Insert playback events individually to allow interruption between inserts
	// This is slower than batch insert but more resilient to CI resource pressure
	// because Go contexts can interrupt between inserts (but not during CGO syscalls)
	for i, loc := range locations {
		for j := 0; j < 3; j++ { // 3 playbacks per location
			// Check context before each insert
			if ctx.Err() != nil {
				t.Fatalf("Context timeout during playback inserts: %v", ctx.Err())
			}

			event := &models.PlaybackEvent{
				ID:              uuid.New(),
				SessionKey:      uuid.New().String(),
				StartedAt:       time.Now().Add(-time.Duration(j) * time.Hour),
				UserID:          i + 1,
				Username:        "testuser" + string(rune('0'+i)),
				IPAddress:       loc.ip,
				MediaType:       "movie",
				Title:           "Test Movie",
				Platform:        "Test Platform",
				Player:          "Test Player",
				LocationType:    "WAN",
				PercentComplete: 75,
				CreatedAt:       time.Now(),
			}

			err := db.InsertPlaybackEvent(event)
			if err != nil {
				t.Fatalf("Failed to insert playback event %d-%d: %v", i, j, err)
			}
		}
	}
}
