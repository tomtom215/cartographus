// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"testing"
	"time"
)

// TestContentMappingCreateAndLookup tests basic content mapping operations.
func TestContentMappingCreateAndLookup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize cross-platform schema
	if err := db.InitCrossPlatformSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize cross-platform schema: %v", err)
	}

	// Test creating a content mapping with IMDb ID
	imdbID := "tt1234567"
	tmdbID := 12345
	lookup := &ContentMappingLookup{
		IMDbID:    &imdbID,
		TMDbID:    &tmdbID,
		Title:     "Test Movie",
		MediaType: "movie",
	}

	mapping, created, err := db.GetOrCreateContentMapping(ctx, lookup)
	if err != nil {
		t.Fatalf("Failed to create content mapping: %v", err)
	}
	if !created {
		t.Error("Expected new mapping to be created")
	}
	if mapping.Title != "Test Movie" {
		t.Errorf("Expected title 'Test Movie', got '%s'", mapping.Title)
	}
	if mapping.IMDbID == nil || *mapping.IMDbID != imdbID {
		t.Error("IMDb ID not set correctly")
	}

	// Test lookup by IMDb ID
	found, err := db.GetContentMappingByExternalID(ctx, "imdb", imdbID)
	if err != nil {
		t.Fatalf("Failed to lookup by IMDb ID: %v", err)
	}
	if found.ID != mapping.ID {
		t.Error("Lookup returned different mapping")
	}

	// Test idempotent creation (should return existing)
	mapping2, created2, err := db.GetOrCreateContentMapping(ctx, lookup)
	if err != nil {
		t.Fatalf("Failed on second create call: %v", err)
	}
	if created2 {
		t.Error("Expected existing mapping, not new creation")
	}
	if mapping2.ID != mapping.ID {
		t.Error("Second call returned different mapping")
	}
}

// TestContentMappingLinkPlatforms tests linking platform-specific IDs to a mapping.
func TestContentMappingLinkPlatforms(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.InitCrossPlatformSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize cross-platform schema: %v", err)
	}

	// Create initial mapping
	imdbID := "tt9999999"
	lookup := &ContentMappingLookup{
		IMDbID:    &imdbID,
		Title:     "Test Show",
		MediaType: "show",
	}

	mapping, _, err := db.GetOrCreateContentMapping(ctx, lookup)
	if err != nil {
		t.Fatalf("Failed to create mapping: %v", err)
	}

	// Link Plex content
	if err := db.LinkPlexContent(ctx, mapping.ID, "12345"); err != nil {
		t.Fatalf("Failed to link Plex content: %v", err)
	}

	// Verify Plex link
	found, err := db.GetContentMappingByExternalID(ctx, "plex", "12345")
	if err != nil {
		t.Fatalf("Failed to lookup by Plex ID: %v", err)
	}
	if found.ID != mapping.ID {
		t.Error("Plex lookup returned different mapping")
	}

	// Link Jellyfin content
	if err := db.LinkJellyfinContent(ctx, mapping.ID, "jellyfin-uuid-123"); err != nil {
		t.Fatalf("Failed to link Jellyfin content: %v", err)
	}

	// Verify Jellyfin link
	found, err = db.GetContentMappingByExternalID(ctx, "jellyfin", "jellyfin-uuid-123")
	if err != nil {
		t.Fatalf("Failed to lookup by Jellyfin ID: %v", err)
	}
	if found.ID != mapping.ID {
		t.Error("Jellyfin lookup returned different mapping")
	}

	// Link Emby content
	if err := db.LinkEmbyContent(ctx, mapping.ID, "emby-item-456"); err != nil {
		t.Fatalf("Failed to link Emby content: %v", err)
	}

	// Verify all links are present
	updated, err := db.GetContentMappingByID(ctx, mapping.ID)
	if err != nil {
		t.Fatalf("Failed to get updated mapping: %v", err)
	}

	if updated.PlexRatingKey == nil || *updated.PlexRatingKey != "12345" {
		t.Error("Plex rating key not persisted")
	}
	if updated.JellyfinItemID == nil || *updated.JellyfinItemID != "jellyfin-uuid-123" {
		t.Error("Jellyfin item ID not persisted")
	}
	if updated.EmbyItemID == nil || *updated.EmbyItemID != "emby-item-456" {
		t.Error("Emby item ID not persisted")
	}
}

// TestContentMappingRequiresExternalID tests that creation fails without external IDs.
func TestContentMappingRequiresExternalID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.InitCrossPlatformSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize cross-platform schema: %v", err)
	}

	// Try to create without any external IDs
	lookup := &ContentMappingLookup{
		Title:     "No External ID",
		MediaType: "movie",
	}

	_, _, err := db.GetOrCreateContentMapping(ctx, lookup)
	if err == nil {
		t.Error("Expected error when creating mapping without external ID")
	}
}

// TestUserLinkCreate tests creating user links.
func TestUserLinkCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.InitCrossPlatformSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize cross-platform schema: %v", err)
	}

	// Create link between two users
	createdBy := "admin"
	link, err := db.CreateUserLink(ctx, 1, 2, "manual", &createdBy)
	if err != nil {
		t.Fatalf("Failed to create user link: %v", err)
	}

	if link.PrimaryUserID != 1 || link.LinkedUserID != 2 {
		t.Error("Link user IDs not set correctly")
	}
	if link.LinkType != "manual" {
		t.Errorf("Expected link type 'manual', got '%s'", link.LinkType)
	}
	if link.CreatedBy == nil || *link.CreatedBy != "admin" {
		t.Error("CreatedBy not set correctly")
	}

	// Test idempotent creation
	link2, err := db.CreateUserLink(ctx, 1, 2, "manual", nil)
	if err != nil {
		t.Fatalf("Failed on second create call: %v", err)
	}
	if link2.ID != link.ID {
		t.Error("Second call should return existing link")
	}
}

// TestUserLinkSelfLinkPrevented tests that users cannot link to themselves.
func TestUserLinkSelfLinkPrevented(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.InitCrossPlatformSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize cross-platform schema: %v", err)
	}

	// Try to link user to themselves
	_, err := db.CreateUserLink(ctx, 1, 1, "manual", nil)
	if err == nil {
		t.Error("Expected error when linking user to themselves")
	}
}

// TestGetLinkedUserIDs tests retrieving all linked user IDs.
func TestGetLinkedUserIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.InitCrossPlatformSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize cross-platform schema: %v", err)
	}

	// Create links: 1 -> 2, 1 -> 3
	if _, err := db.CreateUserLink(ctx, 1, 2, "email", nil); err != nil {
		t.Fatalf("Failed to create first link: %v", err)
	}
	if _, err := db.CreateUserLink(ctx, 1, 3, "manual", nil); err != nil {
		t.Fatalf("Failed to create second link: %v", err)
	}

	// Get all linked IDs for user 1
	ids, err := db.GetAllLinkedUserIDs(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get linked user IDs: %v", err)
	}

	// Should include 1, 2, 3
	if len(ids) != 3 {
		t.Errorf("Expected 3 IDs, got %d", len(ids))
	}

	// Verify user 1 is included (the original user)
	found1 := false
	for _, id := range ids {
		if id == 1 {
			found1 = true
			break
		}
	}
	if !found1 {
		t.Error("Original user ID should be in the result")
	}
}

// TestDeleteUserLink tests user link deletion.
func TestDeleteUserLink(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.InitCrossPlatformSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize cross-platform schema: %v", err)
	}

	// Create a link
	if _, err := db.CreateUserLink(ctx, 10, 20, "manual", nil); err != nil {
		t.Fatalf("Failed to create link: %v", err)
	}

	// Verify link exists
	ids, _ := db.GetAllLinkedUserIDs(ctx, 10)
	if len(ids) != 2 {
		t.Error("Link should exist")
	}

	// Delete the link
	if err := db.DeleteUserLink(ctx, 10, 20); err != nil {
		t.Fatalf("Failed to delete link: %v", err)
	}

	// Verify link is gone
	ids, _ = db.GetAllLinkedUserIDs(ctx, 10)
	if len(ids) != 1 { // Only the user itself
		t.Error("Link should be deleted")
	}
}

// TestDeleteUserLinkBidirectional tests that deletion works in either direction.
func TestDeleteUserLinkBidirectional(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.InitCrossPlatformSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize cross-platform schema: %v", err)
	}

	// Create link: 100 -> 200
	if _, err := db.CreateUserLink(ctx, 100, 200, "email", nil); err != nil {
		t.Fatalf("Failed to create link: %v", err)
	}

	// Delete in reverse direction (200 -> 100) should still work
	if err := db.DeleteUserLink(ctx, 200, 100); err != nil {
		t.Fatalf("Failed to delete link in reverse direction: %v", err)
	}

	// Verify link is gone
	ids, _ := db.GetAllLinkedUserIDs(ctx, 100)
	if len(ids) != 1 {
		t.Error("Link should be deleted even when specified in reverse")
	}
}

// TestContentMappingByTMDbWithMediaType tests that TMDb lookup considers media type.
func TestContentMappingByTMDbWithMediaType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.InitCrossPlatformSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize cross-platform schema: %v", err)
	}

	// TMDb can have same ID for movie and show
	tmdbID := 550

	// Create movie mapping
	movieLookup := &ContentMappingLookup{
		TMDbID:    &tmdbID,
		Title:     "Fight Club",
		MediaType: "movie",
	}
	movieMapping, _, err := db.GetOrCreateContentMapping(ctx, movieLookup)
	if err != nil {
		t.Fatalf("Failed to create movie mapping: %v", err)
	}

	// Create show mapping with same TMDb ID
	showLookup := &ContentMappingLookup{
		TMDbID:    &tmdbID,
		Title:     "Some TV Show",
		MediaType: "show",
	}
	showMapping, created, err := db.GetOrCreateContentMapping(ctx, showLookup)
	if err != nil {
		t.Fatalf("Failed to create show mapping: %v", err)
	}

	// Should create new mapping since media type differs
	if !created {
		t.Error("Expected new show mapping to be created")
	}
	if showMapping.ID == movieMapping.ID {
		t.Error("Show and movie should have different mappings")
	}
}

// TestJoinOrHelper tests the joinOr helper function.
func TestJoinOrHelper(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{}, "1=0"},
		{[]string{"a = 1"}, "a = 1"},
		{[]string{"a = 1", "b = 2"}, "a = 1 OR b = 2"},
		{[]string{"a = 1", "b = 2", "c = 3"}, "a = 1 OR b = 2 OR c = 3"},
	}

	for _, tt := range tests {
		result := joinOr(tt.input)
		if result != tt.expected {
			t.Errorf("joinOr(%v) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}
