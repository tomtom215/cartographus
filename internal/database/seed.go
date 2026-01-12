// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// SeedMockData seeds the database with realistic mock data for screenshot tests
// This is intended for CI/CD screenshot capture and demo purposes only
func (db *DB) SeedMockData(ctx context.Context) error {
	logging.Info().Msg("Seeding database with mock data for screenshots...")

	// Seed random number generator for consistent but varied data
	rand.Seed(time.Now().UnixNano())

	// Define mock data parameters
	const (
		numUsers        = 15
		numLocations    = 50
		numPlaybacks    = 250
		daysOfHistory   = 30
		playbacksPerDay = numPlaybacks / daysOfHistory
	)

	// Mock user names
	users := []string{
		"Alice", "Bob", "Charlie", "David", "Emma",
		"Frank", "Grace", "Henry", "Isabella", "Jack",
		"Kate", "Liam", "Mia", "Noah", "Olivia",
	}

	// Mock cities with coordinates (realistic global distribution)
	locations := []struct {
		city    string
		region  string
		country string
		lat     float64
		lon     float64
	}{
		{"New York", "NY", "United States", 40.7128, -74.0060},
		{"Los Angeles", "CA", "United States", 34.0522, -118.2437},
		{"Chicago", "IL", "United States", 41.8781, -87.6298},
		{"London", "England", "United Kingdom", 51.5074, -0.1278},
		{"Paris", "Île-de-France", "France", 48.8566, 2.3522},
		{"Berlin", "Berlin", "Germany", 52.5200, 13.4050},
		{"Tokyo", "Tokyo", "Japan", 35.6762, 139.6503},
		{"Sydney", "NSW", "Australia", -33.8688, 151.2093},
		{"Toronto", "ON", "Canada", 43.6532, -79.3832},
		{"Amsterdam", "North Holland", "Netherlands", 52.3676, 4.9041},
		{"Barcelona", "Catalonia", "Spain", 41.3851, 2.1734},
		{"Singapore", "Singapore", "Singapore", 1.3521, 103.8198},
		{"Mumbai", "Maharashtra", "India", 19.0760, 72.8777},
		{"São Paulo", "SP", "Brazil", -23.5505, -46.6333},
		{"Mexico City", "CDMX", "Mexico", 19.4326, -99.1332},
		{"Moscow", "Moscow", "Russia", 55.7558, 37.6173},
		{"Dubai", "Dubai", "UAE", 25.2048, 55.2708},
		{"Seoul", "Seoul", "South Korea", 37.5665, 126.9780},
		{"Hong Kong", "Hong Kong", "Hong Kong", 22.3193, 114.1694},
		{"Istanbul", "Istanbul", "Turkey", 41.0082, 28.9784},
		{"Bangkok", "Bangkok", "Thailand", 13.7563, 100.5018},
		{"Cairo", "Cairo", "Egypt", 30.0444, 31.2357},
		{"Buenos Aires", "Buenos Aires", "Argentina", -34.6037, -58.3816},
		{"Lagos", "Lagos", "Nigeria", 6.5244, 3.3792},
		{"Johannesburg", "Gauteng", "South Africa", -26.2041, 28.0473},
		{"Montreal", "QC", "Canada", 45.5017, -73.5673},
		{"Vancouver", "BC", "Canada", 49.2827, -123.1207},
		{"Miami", "FL", "United States", 25.7617, -80.1918},
		{"Seattle", "WA", "United States", 47.6062, -122.3321},
		{"San Francisco", "CA", "United States", 37.7749, -122.4194},
		{"Boston", "MA", "United States", 42.3601, -71.0589},
		{"Washington DC", "DC", "United States", 38.9072, -77.0369},
		{"Atlanta", "GA", "United States", 33.7490, -84.3880},
		{"Dallas", "TX", "United States", 32.7767, -96.7970},
		{"Houston", "TX", "United States", 29.7604, -95.3698},
		{"Phoenix", "AZ", "United States", 33.4484, -112.0740},
		{"Philadelphia", "PA", "United States", 39.9526, -75.1652},
		{"Denver", "CO", "United States", 39.7392, -104.9903},
		{"Portland", "OR", "United States", 45.5152, -122.6784},
		{"Austin", "TX", "United States", 30.2672, -97.7431},
		{"Rome", "Lazio", "Italy", 41.9028, 12.4964},
		{"Madrid", "Madrid", "Spain", 40.4168, -3.7038},
		{"Vienna", "Vienna", "Austria", 48.2082, 16.3738},
		{"Prague", "Prague", "Czech Republic", 50.0755, 14.4378},
		{"Copenhagen", "Copenhagen", "Denmark", 55.6761, 12.5683},
		{"Stockholm", "Stockholm", "Sweden", 59.3293, 18.0686},
		{"Oslo", "Oslo", "Norway", 59.9139, 10.7522},
		{"Helsinki", "Helsinki", "Finland", 60.1699, 24.9384},
		{"Warsaw", "Mazovia", "Poland", 52.2297, 21.0122},
		{"Brussels", "Brussels", "Belgium", 50.8503, 4.3517},
	}

	// Mock movie/show titles
	titles := []string{
		"The Matrix", "Inception", "Interstellar", "The Dark Knight",
		"Pulp Fiction", "Fight Club", "Forrest Gump", "The Shawshank Redemption",
		"Breaking Bad S01E01", "Game of Thrones S01E01", "Stranger Things S01E01",
		"The Office S01E01", "Friends S01E01", "The Mandalorian S01E01",
		"Avatar", "Titanic", "The Avengers", "Star Wars: A New Hope",
		"Blade Runner 2049", "Mad Max: Fury Road", "Dunkirk", "1917",
	}

	// Mock platforms and players
	platforms := []string{"Plex Web", "iOS", "Android", "Roku", "Fire TV", "Apple TV", "Windows", "macOS", "Linux", "Smart TV"}
	players := []string{"Chrome", "Firefox", "Safari", "Edge", "Plex App", "VLC", "Direct Play", "Transcoded"}

	// 1. Seed geolocations first
	logging.Info().Int("count", len(locations)).Msg("Creating mock geolocations...")
	for i, loc := range locations {
		ip := fmt.Sprintf("192.168.%d.%d", i/256, i%256)

		// Helper to convert string to *string
		strPtr := func(s string) *string { return &s }
		intPtr := func(n int) *int { return &n }

		geo := &models.Geolocation{
			IPAddress:      ip,
			Latitude:       loc.lat,
			Longitude:      loc.lon,
			City:           strPtr(loc.city),
			Region:         strPtr(loc.region),
			Country:        loc.country,
			PostalCode:     strPtr(fmt.Sprintf("%05d", 10000+i)),
			Timezone:       strPtr("UTC"),
			AccuracyRadius: intPtr(10),
			LastUpdated:    time.Now(),
		}

		if err := db.UpsertGeolocationWithServer(geo, 40.7128, -74.0060); err != nil {
			return fmt.Errorf("failed to seed geolocation %s: %w", loc.city, err)
		}
	}
	logging.Info().Int("count", len(locations)).Msg("Created geolocations")

	// 2. Seed playback events
	logging.Info().Int("count", numPlaybacks).Msg("Creating mock playback events...")
	startDate := time.Now().AddDate(0, 0, -daysOfHistory)

	for i := 0; i < numPlaybacks; i++ {
		// Random distribution across time
		dayOffset := rand.Intn(daysOfHistory)
		hourOffset := rand.Intn(24)
		minuteOffset := rand.Intn(60)
		timestamp := startDate.AddDate(0, 0, dayOffset).Add(time.Hour * time.Duration(hourOffset)).Add(time.Minute * time.Duration(minuteOffset))

		// Random user and title
		user := users[rand.Intn(len(users))]
		title := titles[rand.Intn(len(titles))]
		platform := platforms[rand.Intn(len(platforms))]
		player := players[rand.Intn(len(players))]

		// Determine media type from title
		mediaType := "movie"
		if stringContains(title, "S0") && stringContains(title, "E0") {
			mediaType = "episode"
		}

		// Helper to convert time.Time to *time.Time
		timePtr := func(t time.Time) *time.Time { return &t }
		strPtr := func(s string) *string { return &s }
		intPtr := func(n int) *int { return &n }

		// Random watch duration (30 minutes to 3 hours)
		durationMinutes := 30 + rand.Intn(150)
		stoppedTime := timestamp.Add(time.Duration(durationMinutes) * time.Minute)

		event := &models.PlaybackEvent{
			ID:               uuid.New(),
			SessionKey:       fmt.Sprintf("mock-session-%d", i),
			StartedAt:        timestamp,
			StoppedAt:        timePtr(stoppedTime),
			UserID:           indexOf(users, user),
			Username:         user,
			IPAddress:        fmt.Sprintf("192.168.%d.%d", rand.Intn(len(locations))/256, rand.Intn(len(locations))%256),
			MediaType:        mediaType,
			Title:            title,
			ParentTitle:      strPtr(""),
			GrandparentTitle: strPtr(""),
			Platform:         platform,
			Player:           player,
			LocationType:     "lan",
			PercentComplete:  85 + rand.Intn(15), // 85-100%
			PausedCounter:    rand.Intn(5),
			PlayDuration:     intPtr(durationMinutes), // CRITICAL: Required for analytics queries
			CreatedAt:        time.Now(),
		}

		if err := db.InsertPlaybackEvent(event); err != nil {
			return fmt.Errorf("failed to seed playback event %d: %w", i, err)
		}
	}
	logging.Info().Int("count", numPlaybacks).Msg("Created playback events")

	logging.Info().
		Int("users", numUsers).
		Int("locations", len(locations)).
		Int("playbacks", numPlaybacks).
		Int("days", daysOfHistory).
		Msg("Mock data seeded successfully")

	return nil
}

// Helper function for indexOf (note: contains and indexContainsStr are already defined in database.go)
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return 0
}
