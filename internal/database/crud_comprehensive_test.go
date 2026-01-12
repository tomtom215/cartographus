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

// TestInsertPlaybackEvent tests inserting playback events with various scenarios
func TestInsertPlaybackEvent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert geolocations first
	insertTestGeolocations(t, db)

	tests := []struct {
		name    string
		event   *models.PlaybackEvent
		wantErr bool
	}{
		{
			name: "valid movie playback",
			event: &models.PlaybackEvent{
				ID:              uuid.New(),
				SessionKey:      "session123",
				UserID:          1,
				Username:        "testuser",
				IPAddress:       "192.168.1.1",
				MediaType:       "movie",
				Title:           "Test Movie",
				Platform:        "Chrome",
				Player:          "Plex Web",
				PercentComplete: 100,
				PlayDuration:    intPtr(7200),
				StartedAt:       time.Now().Add(-2 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "valid episode playback",
			event: &models.PlaybackEvent{
				ID:               uuid.New(),
				SessionKey:       "session456",
				UserID:           2,
				Username:         "testuser2",
				IPAddress:        "192.168.1.2",
				MediaType:        "episode",
				Title:            "Test Episode",
				ParentTitle:      stringPtr("Season 1"),
				GrandparentTitle: stringPtr("Test Show"),
				Platform:         "Firefox",
				Player:           "Plex Web",
				PercentComplete:  50,
				PlayDuration:     intPtr(2400),
				StartedAt:        time.Now().Add(-1 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "track playback",
			event: &models.PlaybackEvent{
				ID:               uuid.New(),
				SessionKey:       "session789",
				UserID:           3,
				Username:         "testuser3",
				IPAddress:        "192.168.1.3",
				MediaType:        "track",
				Title:            "Test Song",
				ParentTitle:      stringPtr("Test Album"),
				GrandparentTitle: stringPtr("Test Artist"),
				Platform:         "Android",
				Player:           "Plex Mobile",
				PercentComplete:  100,
				PlayDuration:     intPtr(180),
				StartedAt:        time.Now().Add(-5 * time.Minute),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.InsertPlaybackEvent(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertPlaybackEvent() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify the event was inserted
				result, err := db.conn.Query(`
					SELECT id, session_key, username, media_type
					FROM playback_events
					WHERE session_key = ?
				`, tt.event.SessionKey)
				if err != nil {
					t.Fatalf("Failed to query inserted event: %v", err)
				}
				defer result.Close()

				if !result.Next() {
					t.Error("Event was not inserted")
				}

				if err := result.Err(); err != nil {
					t.Fatalf("Error iterating query results: %v", err)
				}

				var (
					id         string
					sessionKey string
					username   string
					mediaType  string
				)

				if err := result.Scan(&id, &sessionKey, &username, &mediaType); err != nil {
					t.Fatalf("Failed to scan result: %v", err)
				}

				if sessionKey != tt.event.SessionKey {
					t.Errorf("Expected session_key %s, got %s", tt.event.SessionKey, sessionKey)
				}
				if username != tt.event.Username {
					t.Errorf("Expected username %s, got %s", tt.event.Username, username)
				}
				if mediaType != tt.event.MediaType {
					t.Errorf("Expected media_type %s, got %s", tt.event.MediaType, mediaType)
				}
			}
		})
	}
}

// TestInsertGeolocation tests inserting geolocation data
func TestInsertGeolocation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name    string
		geo     *models.Geolocation
		wantErr bool
	}{
		{
			name: "valid US location",
			geo: &models.Geolocation{
				IPAddress: "8.8.8.8",
				Latitude:  40.7128,
				Longitude: -74.0060,
				City:      stringPtr("New York"),
				Region:    stringPtr("New York"),
				Country:   "United States",
			},
			wantErr: false,
		},
		{
			name: "valid UK location",
			geo: &models.Geolocation{
				IPAddress: "8.8.4.4",
				Latitude:  51.5074,
				Longitude: -0.1278,
				City:      stringPtr("London"),
				Region:    stringPtr("England"),
				Country:   "United Kingdom",
			},
			wantErr: false,
		},
		{
			name: "location with empty city",
			geo: &models.Geolocation{
				IPAddress: "1.1.1.1",
				Latitude:  0.0,
				Longitude: 0.0,
				City:      stringPtr(""),
				Region:    stringPtr(""),
				Country:   "Unknown",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.UpsertGeolocation(tt.geo)
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertGeolocation() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify the geolocation was inserted
				result, err := db.conn.Query(`
					SELECT ip_address, city, country
					FROM geolocations
					WHERE ip_address = ?
				`, tt.geo.IPAddress)
				if err != nil {
					t.Fatalf("Failed to query inserted geolocation: %v", err)
				}
				defer result.Close()

				if !result.Next() {
					t.Error("Geolocation was not inserted")
				}

				if err := result.Err(); err != nil {
					t.Fatalf("Error iterating query results: %v", err)
				}
			}
		})
	}
}

// TestGetStats tests retrieving database statistics
func TestGetStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert test data
	insertTestGeolocations(t, db)
	insertTestPlaybacks(t, db)

	stats, err := db.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.TotalPlaybacks < 0 {
		t.Errorf("Expected non-negative total playbacks, got %d", stats.TotalPlaybacks)
	}

	if stats.UniqueUsers < 0 {
		t.Errorf("Expected non-negative unique users, got %d", stats.UniqueUsers)
	}

	if stats.UniqueLocations < 0 {
		t.Errorf("Expected non-negative unique locations, got %d", stats.UniqueLocations)
	}
}

// TestGetLocationStats tests retrieving location statistics with filters
func TestGetLocationStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert test data
	insertTestGeolocations(t, db)
	insertTestPlaybacks(t, db)

	tests := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{
			name:   "no filter",
			filter: LocationStatsFilter{},
		},
		{
			name: "with limit",
			filter: LocationStatsFilter{
				Limit: 5,
			},
		},
		{
			name: "with user filter",
			filter: LocationStatsFilter{
				Users: []string{"user1"},
			},
		},
		{
			name: "with media type filter",
			filter: LocationStatsFilter{
				MediaTypes: []string{"movie"},
			},
		},
		{
			name: "with date range",
			filter: func() LocationStatsFilter {
				now := time.Now()
				start := now.Add(-7 * 24 * time.Hour)
				return LocationStatsFilter{
					StartDate: &start,
					EndDate:   &now,
				}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := db.GetLocationStatsFiltered(context.Background(), tt.filter)
			if err != nil {
				t.Errorf("GetLocationStatsFiltered() error = %v", err)
			}

			// Validate result structure
			if stats == nil {
				t.Error("Expected non-nil stats")
			}
		})
	}
}

// TestFilterBuilder tests the filter builder functionality
func TestFilterBuilder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name   string
		filter LocationStatsFilter
	}{
		{
			name: "multiple filters",
			filter: func() LocationStatsFilter {
				now := time.Now()
				start := now.Add(-7 * 24 * time.Hour)
				return LocationStatsFilter{
					Users:      []string{"testuser"},
					MediaTypes: []string{"movie"},
					Platforms:  []string{"Chrome"},
					StartDate:  &start,
					EndDate:    &now,
				}
			}(),
		},
		{
			name: "only date range",
			filter: func() LocationStatsFilter {
				now := time.Now()
				start := now.Add(-30 * 24 * time.Hour)
				return LocationStatsFilter{
					StartDate: &start,
					EndDate:   &now,
				}
			}(),
		},
		{
			name: "only user",
			filter: LocationStatsFilter{
				Users: []string{"testuser"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			_, err := db.GetLocationStatsFiltered(context.Background(), tt.filter)
			if err != nil {
				t.Errorf("GetLocationStatsFiltered() with filter error = %v", err)
			}
		})
	}
}

// TestDatabaseClose tests database closure
func TestDatabaseClose(t *testing.T) {
	db := setupTestDB(t)

	err := db.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Subsequent operations should fail
	_, err = db.GetStats(context.Background())
	if err == nil {
		t.Error("Expected error after close, got nil")
	}
}

// TestConcurrentInserts tests concurrent playback event insertions
func TestConcurrentInserts(t *testing.T) {
	db := setupConcurrentTestDB(t)
	defer db.Close()

	// Insert geolocations first
	insertTestGeolocations(t, db)

	// Create multiple events concurrently
	const numEvents = 10
	done := make(chan error, numEvents)

	for i := 0; i < numEvents; i++ {
		go func(index int) {
			event := &models.PlaybackEvent{
				ID:              uuid.New(),
				SessionKey:      uuid.New().String(),
				UserID:          index,
				Username:        "user" + string(rune(index)),
				IPAddress:       "192.168.1.1",
				MediaType:       "movie",
				Title:           "Concurrent Test Movie",
				Platform:        "Chrome",
				Player:          "Plex Web",
				PercentComplete: 100,
				PlayDuration:    intPtr(7200),
				StartedAt:       time.Now(),
			}

			done <- db.InsertPlaybackEvent(event)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numEvents; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent insert %d failed: %v", i, err)
		}
	}
}

// TestEmptyDatabase tests operations on an empty database
func TestEmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Stats on empty database should return zero values
	stats, err := db.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats() on empty database error = %v", err)
	}

	if stats.TotalPlaybacks != 0 {
		t.Errorf("Expected 0 total playbacks on empty database, got %d", stats.TotalPlaybacks)
	}

	// Location stats should return empty slice
	locationStats, err := db.GetLocationStatsFiltered(context.Background(), LocationStatsFilter{})
	if err != nil {
		t.Fatalf("GetLocationStatsFiltered() on empty database error = %v", err)
	}

	if len(locationStats) != 0 {
		t.Errorf("Expected 0 location stats on empty database, got %d", len(locationStats))
	}
}

// TestInvalidIPAddress tests handling of invalid IP addresses
func TestInvalidIPAddress(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	geo := &models.Geolocation{
		IPAddress: "invalid_ip",
		Latitude:  0.0,
		Longitude: 0.0,
		City:      stringPtr("Unknown"),
		Region:    stringPtr("Unknown"),
		Country:   "Unknown",
	}

	// Should insert without error (validation happens elsewhere)
	err := db.UpsertGeolocation(geo)
	if err != nil {
		t.Errorf("UpsertGeolocation() with invalid IP error = %v", err)
	}
}

// TestLargeDataset tests performance with a larger dataset
func TestLargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	// Insert test geolocations
	insertTestGeolocations(t, db)

	// Insert many events
	const numEvents = 1000
	for i := 0; i < numEvents; i++ {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			UserID:          i % 10, // 10 different users
			Username:        "user" + string(rune(i%10)),
			IPAddress:       "192.168.1.1",
			MediaType:       []string{"movie", "episode", "track"}[i%3],
			Title:           "Test Title",
			Platform:        "Chrome",
			Player:          "Plex Web",
			PercentComplete: 100,
			PlayDuration:    intPtr(7200),
			StartedAt:       time.Now().Add(-time.Duration(i) * time.Minute),
		}

		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event %d: %v", i, err)
		}
	}

	// Query stats - should be fast even with 1000 events
	start := time.Now()
	stats, err := db.GetStats(context.Background())
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.TotalPlaybacks < numEvents {
		t.Errorf("Expected at least %d playbacks, got %d", numEvents, stats.TotalPlaybacks)
	}

	// Should complete in under 1 second
	if duration > time.Second {
		t.Errorf("GetStats() took too long: %v", duration)
	}
}
