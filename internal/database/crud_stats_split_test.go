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

func TestGetUniqueUsers_EmptyDatabase(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	users, err := db.GetUniqueUsers(context.Background())
	if err != nil {
		t.Fatalf("GetUniqueUsers failed: %v", err)
	}

	if len(users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(users))
	}
}

func TestGetUniqueMediaTypes_EmptyDatabase(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	mediaTypes, err := db.GetUniqueMediaTypes(context.Background())
	if err != nil {
		t.Fatalf("GetUniqueMediaTypes failed: %v", err)
	}

	if len(mediaTypes) != 0 {
		t.Errorf("Expected 0 media types, got %d", len(mediaTypes))
	}
}

func TestGetStats_Success(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	// Insert test playback events from multiple users and locations
	now := time.Now()

	events := []struct {
		userID   int
		username string
		ip       string
		offset   time.Duration
	}{
		{1, "user1", "192.168.1.1", 0},
		{2, "user2", "192.168.1.2", -1 * time.Hour},
		{1, "user1", "192.168.1.1", -2 * time.Hour},
		{3, "user3", "192.168.1.3", -30 * time.Hour}, // Older than 24h (increased margin for timing stability)
	}

	for _, e := range events {
		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       now.Add(e.offset),
			UserID:          e.userID,
			Username:        e.username,
			IPAddress:       e.ip,
			MediaType:       "movie",
			Title:           "Test Movie",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	stats, err := db.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalPlaybacks != 4 {
		t.Errorf("Expected 4 total playbacks, got %d", stats.TotalPlaybacks)
	}

	if stats.UniqueUsers != 3 {
		t.Errorf("Expected 3 unique users, got %d", stats.UniqueUsers)
	}

	if stats.UniqueLocations < 3 {
		t.Errorf("Expected at least 3 unique locations, got %d", stats.UniqueLocations)
	}

	// Recent activity should be 3 (only events within last 24h)
	if stats.RecentActivity != 3 {
		t.Errorf("Expected 3 recent playbacks, got %d", stats.RecentActivity)
	}

	// Should have top countries
	if len(stats.TopCountries) == 0 {
		t.Error("Expected at least one country in top countries")
	}
}

func TestGetStats_EmptyDatabase(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	stats, err := db.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalPlaybacks != 0 {
		t.Errorf("Expected 0 total playbacks, got %d", stats.TotalPlaybacks)
	}

	if stats.UniqueUsers != 0 {
		t.Errorf("Expected 0 unique users, got %d", stats.UniqueUsers)
	}

	if stats.RecentActivity != 0 {
		t.Errorf("Expected 0 recent activity, got %d", stats.RecentActivity)
	}
}
