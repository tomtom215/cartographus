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

// BenchmarkGetH3AggregatedHexagons benchmarks H3 hexagon aggregation at different resolutions
func BenchmarkGetH3AggregatedHexagons(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer db.Close()

	if !db.spatialAvailable {
		b.Skip("Spatial extension not available")
	}

	// Insert large test dataset
	insertLargeSpatialDataset(b, db, 1000)

	filter := LocationStatsFilter{Limit: 10000}

	benchmarks := []struct {
		name       string
		resolution int
	}{
		{"Resolution6_Country", 6},
		{"Resolution7_City", 7},
		{"Resolution8_Neighborhood", 8},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := db.GetH3AggregatedHexagons(context.Background(), filter, bm.resolution)
				if err != nil {
					b.Fatalf("GetH3AggregatedHexagons failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkGetDistanceWeightedArcs benchmarks arc distance calculations
func BenchmarkGetDistanceWeightedArcs(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer db.Close()

	if !db.spatialAvailable {
		b.Skip("Spatial extension not available")
	}

	// Insert large test dataset
	insertLargeSpatialDataset(b, db, 1000)

	serverLat := 40.7128
	serverLon := -74.0060

	benchmarks := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{"NoFilter", LocationStatsFilter{Limit: 1000}},
		{"WithDateFilter", LocationStatsFilter{
			StartDate: &[]time.Time{time.Now().AddDate(0, 0, -30)}[0],
			Limit:     1000,
		}},
		{"WithUserFilter", LocationStatsFilter{
			Users: []string{"testuser1", "testuser2"},
			Limit: 1000,
		}},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := db.GetDistanceWeightedArcs(context.Background(), bm.filter, serverLat, serverLon)
				if err != nil {
					b.Fatalf("GetDistanceWeightedArcs failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkGetLocationsInViewport benchmarks spatial index viewport queries
func BenchmarkGetLocationsInViewport(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer db.Close()

	if !db.spatialAvailable {
		b.Skip("Spatial extension not available")
	}

	// Insert large test dataset
	insertLargeSpatialDataset(b, db, 1000)

	filter := LocationStatsFilter{Limit: 5000}

	benchmarks := []struct {
		name  string
		west  float64
		south float64
		east  float64
		north float64
	}{
		{"WholeWorld", -180, -90, 180, 90},
		{"NorthAmerica", -130, 25, -60, 50},
		{"Europe", -10, 35, 40, 70},
		{"Asia", 60, -10, 180, 70},
		{"SmallViewport_NYC", -74.1, 40.6, -73.9, 40.8},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := db.GetLocationsInViewport(context.Background(), filter, bm.west, bm.south, bm.east, bm.north)
				if err != nil {
					b.Fatalf("GetLocationsInViewport failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkGetTemporalSpatialDensity benchmarks temporal-spatial aggregations
func BenchmarkGetTemporalSpatialDensity(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer db.Close()

	if !db.spatialAvailable {
		b.Skip("Spatial extension not available")
	}

	// Insert large test dataset with temporal spread
	insertTemporalSpatialDataset(b, db, 1000, 30) // 1000 locations, 30 days

	filter := LocationStatsFilter{
		StartDate: &[]time.Time{time.Now().AddDate(0, 0, -30)}[0],
		Limit:     10000,
	}

	benchmarks := []struct {
		name         string
		interval     string
		h3Resolution int
	}{
		{"Hourly_Resolution7", "hour", 7},
		{"Daily_Resolution7", "day", 7},
		{"Weekly_Resolution7", "week", 7},
		{"Monthly_Resolution7", "month", 7},
		{"Hourly_Resolution6", "hour", 6},
		{"Daily_Resolution8", "day", 8},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := db.GetTemporalSpatialDensity(context.Background(), filter, bm.interval, bm.h3Resolution)
				if err != nil {
					b.Fatalf("GetTemporalSpatialDensity failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkGetNearbyLocations benchmarks proximity search
func BenchmarkGetNearbyLocations(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer db.Close()

	if !db.spatialAvailable {
		b.Skip("Spatial extension not available")
	}

	// Insert large test dataset
	insertLargeSpatialDataset(b, db, 1000)

	filter := LocationStatsFilter{Limit: 1000}
	lat := 40.7128 // New York
	lon := -74.0060

	benchmarks := []struct {
		name     string
		radiusKm float64
	}{
		{"Radius10km", 10},
		{"Radius100km", 100},
		{"Radius500km", 500},
		{"Radius1000km", 1000},
		{"Radius5000km", 5000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := db.GetNearbyLocations(context.Background(), lat, lon, bm.radiusKm, filter)
				if err != nil {
					b.Fatalf("GetNearbyLocations failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkSpatialIndexPerformance compares viewport queries with and without spatial index
func BenchmarkSpatialIndexPerformance(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer db.Close()

	if !db.spatialAvailable {
		b.Skip("Spatial extension not available")
	}

	// Insert large test dataset
	insertLargeSpatialDataset(b, db, 5000) // Larger dataset for index benefits

	filter := LocationStatsFilter{Limit: 5000}

	// Benchmark with spatial index (current implementation)
	b.Run("WithSpatialIndex", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := db.GetLocationsInViewport(context.Background(), filter, -74.1, 40.6, -73.9, 40.8)
			if err != nil {
				b.Fatalf("GetLocationsInViewport failed: %v", err)
			}
		}
	})
}

// setupBenchmarkDB creates a test database with optimized settings for benchmarking
func setupBenchmarkDB(b *testing.B) *DB {
	db := setupTestDB(&testing.T{})
	return db
}

// insertLargeSpatialDataset inserts a large dataset for performance benchmarking
func insertLargeSpatialDataset(b *testing.B, db *DB, numLocations int) {
	// Generate distributed locations across the globe
	latMin, latMax := -60.0, 70.0
	lonMin, lonMax := -180.0, 180.0
	latStep := (latMax - latMin) / float64(numLocations/10)
	lonStep := (lonMax - lonMin) / float64(numLocations/10)

	locationCount := 0
	for lat := latMin; lat < latMax && locationCount < numLocations; lat += latStep {
		for lon := lonMin; lon < lonMax && locationCount < numLocations; lon += lonStep {
			ip := generateIP(locationCount)
			city := "City" + string(rune('A'+locationCount%26))
			country := "Country" + string(rune('A'+locationCount%10))

			geo := &models.Geolocation{
				IPAddress:   ip,
				Latitude:    lat,
				Longitude:   lon,
				City:        &city,
				Country:     country,
				LastUpdated: time.Now(),
			}

			// Use New York as server location for distance calculations
			err := db.UpsertGeolocationWithServer(geo, 40.7128, -74.0060)
			if err != nil {
				b.Fatalf("Failed to insert geolocation: %v", err)
			}

			// Insert 5 playback events per location
			for i := 0; i < 5; i++ {
				event := &models.PlaybackEvent{
					ID:              uuid.New(),
					SessionKey:      uuid.New().String(),
					StartedAt:       time.Now().Add(-time.Duration(i) * time.Hour),
					UserID:          (locationCount % 100) + 1,
					Username:        "testuser" + string(rune('0'+(locationCount%10))),
					IPAddress:       ip,
					MediaType:       []string{"movie", "episode", "track"}[i%3],
					Title:           "Test Media " + string(rune('A'+i)),
					Platform:        "Test Platform",
					Player:          "Test Player",
					LocationType:    "WAN",
					PercentComplete: 50 + i*10,
					CreatedAt:       time.Now(),
				}

				err := db.InsertPlaybackEvent(event)
				if err != nil {
					b.Fatalf("Failed to insert playback event: %v", err)
				}
			}

			locationCount++
		}
	}

	b.Logf("Inserted %d locations with %d playback events", locationCount, locationCount*5)
}

// insertTemporalSpatialDataset inserts data spread across time and space
func insertTemporalSpatialDataset(b *testing.B, db *DB, numLocations int, daysBack int) {
	// Generate distributed locations
	latMin, latMax := -60.0, 70.0
	lonMin, lonMax := -180.0, 180.0
	latStep := (latMax - latMin) / float64(numLocations/10)
	lonStep := (lonMax - lonMin) / float64(numLocations/10)

	locationCount := 0
	for lat := latMin; lat < latMax && locationCount < numLocations; lat += latStep {
		for lon := lonMin; lon < lonMax && locationCount < numLocations; lon += lonStep {
			ip := generateIP(locationCount)
			city := "City" + string(rune('A'+locationCount%26))
			country := "Country" + string(rune('A'+locationCount%10))

			geo := &models.Geolocation{
				IPAddress:   ip,
				Latitude:    lat,
				Longitude:   lon,
				City:        &city,
				Country:     country,
				LastUpdated: time.Now(),
			}

			err := db.UpsertGeolocationWithServer(geo, 40.7128, -74.0060)
			if err != nil {
				b.Fatalf("Failed to insert geolocation: %v", err)
			}

			// Insert playback events spread across time
			for day := 0; day < daysBack; day++ {
				// Vary number of playbacks per day
				playbacksPerDay := 1 + (day % 5)
				for i := 0; i < playbacksPerDay; i++ {
					event := &models.PlaybackEvent{
						ID:              uuid.New(),
						SessionKey:      uuid.New().String(),
						StartedAt:       time.Now().AddDate(0, 0, -day).Add(-time.Duration(i) * time.Hour),
						UserID:          (locationCount % 100) + 1,
						Username:        "testuser" + string(rune('0'+(locationCount%10))),
						IPAddress:       ip,
						MediaType:       []string{"movie", "episode", "track"}[i%3],
						Title:           "Test Media",
						Platform:        "Test Platform",
						Player:          "Test Player",
						LocationType:    "WAN",
						PercentComplete: 75,
						CreatedAt:       time.Now().AddDate(0, 0, -day),
					}

					err := db.InsertPlaybackEvent(event)
					if err != nil {
						b.Fatalf("Failed to insert playback event: %v", err)
					}
				}
			}

			locationCount++
		}
	}

	b.Logf("Inserted %d locations with temporal data spanning %d days", locationCount, daysBack)
}

// generateIP generates a test IP address from an index
func generateIP(index int) string {
	// Generate IP in range 10.x.x.x for testing
	octet2 := (index / 65536) % 256
	octet3 := (index / 256) % 256
	octet4 := index % 256
	return "10." + string(rune('0'+octet2)) + "." + string(rune('0'+octet3)) + "." + string(rune('0'+octet4))
}
