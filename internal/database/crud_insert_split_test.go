// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
)

func TestInsertPlaybackEvent_GeneratesID(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	event := &models.PlaybackEvent{
		// ID not set - should be generated
		SessionKey:      "test-session-" + uuid.New().String(),
		StartedAt:       time.Now(),
		UserID:          1,
		Username:        "testuser",
		IPAddress:       "192.168.1.100",
		MediaType:       "movie",
		Title:           "Test Movie",
		Platform:        "Test Platform",
		Player:          "Test Player",
		LocationType:    "LAN",
		PercentComplete: 100,
	}

	err := db.InsertPlaybackEvent(event)
	if err != nil {
		t.Fatalf("InsertPlaybackEvent failed: %v", err)
	}

	// Verify ID was generated
	if event.ID == uuid.Nil {
		t.Error("Expected ID to be generated, but it's nil")
	}

	// Verify CreatedAt was set
	if event.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set, but it's zero")
	}
}

func TestInsertPlaybackEvent_PreservesExistingID(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	existingID := uuid.New()
	existingTime := time.Now().Add(-1 * time.Hour)

	event := &models.PlaybackEvent{
		ID:              existingID,
		SessionKey:      "test-session-" + uuid.New().String(),
		StartedAt:       time.Now(),
		UserID:          1,
		Username:        "testuser",
		IPAddress:       "192.168.1.100",
		MediaType:       "movie",
		Title:           "Test Movie",
		Platform:        "Test Platform",
		Player:          "Test Player",
		LocationType:    "LAN",
		PercentComplete: 100,
		CreatedAt:       existingTime,
	}

	err := db.InsertPlaybackEvent(event)
	if err != nil {
		t.Fatalf("InsertPlaybackEvent failed: %v", err)
	}

	// Verify ID was preserved
	if event.ID != existingID {
		t.Errorf("Expected ID %s, got %s", existingID, event.ID)
	}

	// Verify CreatedAt was preserved
	if !event.CreatedAt.Equal(existingTime) {
		t.Errorf("Expected CreatedAt %v, got %v", existingTime, event.CreatedAt)
	}
}

func TestInsertPlaybackEvent_MetadataEnrichmentFields(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	// Create test data for new metadata enrichment fields
	ratingKey := "123456"
	parentRatingKey := "123450"
	grandparentRatingKey := "123400"
	mediaIndex := 5       // Episode 5
	parentMediaIndex := 2 // Season 2
	guid := "plex://episode/5d9c0852f647b40020cae1f7"
	originalTitle := "Breaking Bad (Original Title)"
	fullTitle := "S02E05 - Breakage"
	originallyAvailableAt := "2009-04-05"
	watchedStatus := 1
	thumb := "/library/metadata/123456/thumb/1234567890"
	directors := "Vince Gilligan,Michelle MacLaren"
	writers := "Vince Gilligan"
	actors := "Bryan Cranston,Aaron Paul,Anna Gunn"
	genres := "Crime,Drama,Thriller"

	event := &models.PlaybackEvent{
		SessionKey:      "test-session-" + uuid.New().String(),
		StartedAt:       time.Now(),
		UserID:          1,
		Username:        "testuser",
		IPAddress:       "192.168.1.100",
		MediaType:       "episode",
		Title:           "Breakage",
		Platform:        "Plex Web",
		Player:          "Chrome",
		LocationType:    "LAN",
		PercentComplete: 100,

		// New metadata enrichment fields
		RatingKey:             &ratingKey,
		ParentRatingKey:       &parentRatingKey,
		GrandparentRatingKey:  &grandparentRatingKey,
		MediaIndex:            &mediaIndex,
		ParentMediaIndex:      &parentMediaIndex,
		GUID:                  &guid,
		OriginalTitle:         &originalTitle,
		FullTitle:             &fullTitle,
		OriginallyAvailableAt: &originallyAvailableAt,
		WatchedStatus:         &watchedStatus,
		Thumb:                 &thumb,
		Directors:             &directors,
		Writers:               &writers,
		Actors:                &actors,
		Genres:                &genres,
	}

	// Insert the event
	err := db.InsertPlaybackEvent(event)
	if err != nil {
		t.Fatalf("InsertPlaybackEvent failed: %v", err)
	}

	// Verify ID and CreatedAt were set
	if event.ID == uuid.Nil {
		t.Error("Expected ID to be generated, but it's nil")
	}
	if event.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set, but it's zero")
	}

	// Query the event back to verify fields were stored correctly
	query := `SELECT
		rating_key, parent_rating_key, grandparent_rating_key,
		media_index, parent_media_index,
		guid, original_title, full_title, originally_available_at,
		watched_status, thumb,
		directors, writers, actors, genres
	FROM playback_events WHERE session_key = ?`

	var (
		dbRatingKey, dbParentRatingKey, dbGrandparentRatingKey                 *string
		dbMediaIndex, dbParentMediaIndex, dbWatchedStatus                      *int
		dbGUID, dbOriginalTitle, dbFullTitle, dbOriginallyAvailableAt, dbThumb *string
		dbDirectors, dbWriters, dbActors, dbGenres                             *string
	)

	err = db.conn.QueryRow(query, event.SessionKey).Scan(
		&dbRatingKey, &dbParentRatingKey, &dbGrandparentRatingKey,
		&dbMediaIndex, &dbParentMediaIndex,
		&dbGUID, &dbOriginalTitle, &dbFullTitle, &dbOriginallyAvailableAt,
		&dbWatchedStatus, &dbThumb,
		&dbDirectors, &dbWriters, &dbActors, &dbGenres,
	)
	if err != nil {
		t.Fatalf("Failed to query playback event: %v", err)
	}

	// Verify all fields match
	if dbRatingKey == nil || *dbRatingKey != ratingKey {
		t.Errorf("Expected rating_key %s, got %v", ratingKey, dbRatingKey)
	}
	if dbParentRatingKey == nil || *dbParentRatingKey != parentRatingKey {
		t.Errorf("Expected parent_rating_key %s, got %v", parentRatingKey, dbParentRatingKey)
	}
	if dbGrandparentRatingKey == nil || *dbGrandparentRatingKey != grandparentRatingKey {
		t.Errorf("Expected grandparent_rating_key %s, got %v", grandparentRatingKey, dbGrandparentRatingKey)
	}
	if dbMediaIndex == nil || *dbMediaIndex != mediaIndex {
		t.Errorf("Expected media_index %d, got %v", mediaIndex, dbMediaIndex)
	}
	if dbParentMediaIndex == nil || *dbParentMediaIndex != parentMediaIndex {
		t.Errorf("Expected parent_media_index %d, got %v", parentMediaIndex, dbParentMediaIndex)
	}
	if dbGUID == nil || *dbGUID != guid {
		t.Errorf("Expected guid %s, got %v", guid, dbGUID)
	}
	if dbOriginalTitle == nil || *dbOriginalTitle != originalTitle {
		t.Errorf("Expected original_title %s, got %v", originalTitle, dbOriginalTitle)
	}
	if dbFullTitle == nil || *dbFullTitle != fullTitle {
		t.Errorf("Expected full_title %s, got %v", fullTitle, dbFullTitle)
	}
	if dbOriginallyAvailableAt == nil || *dbOriginallyAvailableAt != originallyAvailableAt {
		t.Errorf("Expected originally_available_at %s, got %v", originallyAvailableAt, dbOriginallyAvailableAt)
	}
	if dbWatchedStatus == nil || *dbWatchedStatus != watchedStatus {
		t.Errorf("Expected watched_status %d, got %v", watchedStatus, dbWatchedStatus)
	}
	if dbThumb == nil || *dbThumb != thumb {
		t.Errorf("Expected thumb %s, got %v", thumb, dbThumb)
	}
	if dbDirectors == nil || *dbDirectors != directors {
		t.Errorf("Expected directors %s, got %v", directors, dbDirectors)
	}
	if dbWriters == nil || *dbWriters != writers {
		t.Errorf("Expected writers %s, got %v", writers, dbWriters)
	}
	if dbActors == nil || *dbActors != actors {
		t.Errorf("Expected actors %s, got %v", actors, dbActors)
	}
	if dbGenres == nil || *dbGenres != genres {
		t.Errorf("Expected genres %s, got %v", genres, dbGenres)
	}
}
