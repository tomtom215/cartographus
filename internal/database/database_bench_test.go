// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// BenchmarkGetStats benchmarks the stats query
func BenchmarkGetStats(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()

	// Insert test data
	insertBenchPlaybacks(b, db, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetStats(context.Background())
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetPlaybackTrends benchmarks playback trends query
func BenchmarkGetPlaybackTrends(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()

	insertBenchPlaybacks(b, db, 1000)

	filter := LocationStatsFilter{
		Limit: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := db.GetPlaybackTrends(context.Background(), filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetMediaTypeDistribution benchmarks media type distribution query
func BenchmarkGetMediaTypeDistribution(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()

	insertBenchPlaybacks(b, db, 1000)

	filter := LocationStatsFilter{
		Limit: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetMediaTypeDistribution(context.Background(), filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetTopCities benchmarks top cities query
func BenchmarkGetTopCities(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()

	insertBenchPlaybacks(b, db, 1000)
	insertBenchGeolocations(b, db, 10)

	filter := LocationStatsFilter{
		Limit: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetTopCities(context.Background(), filter, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetLibraryStats benchmarks library stats query
func BenchmarkGetLibraryStats(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()

	insertBenchPlaybacks(b, db, 1000)

	filter := LocationStatsFilter{
		Limit: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetLibraryStats(context.Background(), filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetDurationStats benchmarks duration stats query
func BenchmarkGetDurationStats(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()

	insertBenchPlaybacks(b, db, 1000)

	filter := LocationStatsFilter{
		Limit: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetDurationStats(context.Background(), filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParallelQueries benchmarks concurrent query execution
func BenchmarkParallelQueries(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()

	insertBenchPlaybacks(b, db, 1000)

	filter := LocationStatsFilter{
		Limit: 1000,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := db.GetMediaTypeDistribution(context.Background(), filter)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Benchmark helpers

func setupBenchDB(b *testing.B) *DB {
	b.Helper()

	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := New(cfg, 0.0, 0.0)
	if err != nil {
		b.Fatalf("Failed to create test database: %v", err)
	}

	return db
}

func insertBenchPlaybacks(b *testing.B, db *DB, count int) {
	b.Helper()

	baseTime := time.Now().Add(-24 * time.Hour)
	mediaTypes := []string{"movie", "episode", "track"}
	platforms := []string{"Chrome", "Roku", "iOS"}
	libraries := []string{"Movies", "TV Shows", "Music"}

	for i := 0; i < count; i++ {
		sectionID := (i % 3) + 1
		libraryName := libraries[i%3]
		playDuration := 30 + (i % 120)

		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      fmt.Sprintf("test-session-%d", i),
			StartedAt:       baseTime.Add(time.Duration(i) * time.Minute),
			UserID:          (i % 10) + 1,
			Username:        fmt.Sprintf("user%d", (i%10)+1),
			IPAddress:       fmt.Sprintf("192.168.1.%d", (i%254)+1),
			MediaType:       mediaTypes[i%3],
			Title:           fmt.Sprintf("Test Title %d", i),
			Platform:        platforms[i%3],
			Player:          "Plex Web",
			LocationType:    "lan",
			PercentComplete: 50 + (i % 50),
			PausedCounter:   i % 5,
			SectionID:       &sectionID,
			LibraryName:     &libraryName,
			PlayDuration:    &playDuration,
		}

		if err := db.InsertPlaybackEvent(event); err != nil {
			b.Fatalf("Failed to insert test playback: %v", err)
		}
	}
}

func insertBenchGeolocations(b *testing.B, db *DB, count int) {
	b.Helper()

	cities := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix"}
	countries := []string{"US", "CA", "GB", "DE", "FR"}

	for i := 0; i < count; i++ {
		city := cities[i%len(cities)]
		country := countries[i%len(countries)]

		geo := &models.Geolocation{
			IPAddress: fmt.Sprintf("192.168.1.%d", i+1),
			Latitude:  40.0 + float64(i)*0.1,
			Longitude: -74.0 + float64(i)*0.1,
			City:      &city,
			Country:   country,
		}

		if err := db.UpsertGeolocation(geo); err != nil {
			b.Fatalf("Failed to upsert test geolocation: %v", err)
		}
	}
}
