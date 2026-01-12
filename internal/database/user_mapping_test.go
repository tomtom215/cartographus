// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
user_mapping_test.go - Tests for User ID Mapping Operations

This file tests the user mapping functionality that enables cross-source
user tracking by mapping external IDs (Jellyfin/Emby UUIDs, Plex integers)
to internal integer IDs.

Test Coverage:
  - GetOrCreateUserMapping: Create and retrieve mappings
  - GetUserMappingByExternal: Lookup by source+server+external_id
  - GetUserMappingByInternal: Lookup by internal user ID
  - ResolveUserID: Convenience method for event processing
  - Thread safety: Concurrent access to mapping operations
  - Edge cases: Empty strings, special characters, Unicode
*/

package database

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"github.com/tomtom215/cartographus/internal/models"
)

// TestGetOrCreateUserMapping tests the atomic lookup-or-create operation
func TestGetOrCreateUserMapping(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("creates new mapping when none exists", func(t *testing.T) {
		lookup := &models.UserMappingLookup{
			Source:         "jellyfin",
			ServerID:       "jellyfin-server-1",
			ExternalUserID: "550e8400-e29b-41d4-a716-446655440000",
			Username:       strPtr("testuser"),
			FriendlyName:   strPtr("Test User"),
		}

		mapping, created, err := db.GetOrCreateUserMapping(ctx, lookup)
		if err != nil {
			t.Fatalf("GetOrCreateUserMapping failed: %v", err)
		}

		if !created {
			t.Error("Expected created=true for new mapping")
		}

		if mapping.Source != lookup.Source {
			t.Errorf("Expected Source=%s, got %s", lookup.Source, mapping.Source)
		}
		if mapping.ServerID != lookup.ServerID {
			t.Errorf("Expected ServerID=%s, got %s", lookup.ServerID, mapping.ServerID)
		}
		if mapping.ExternalUserID != lookup.ExternalUserID {
			t.Errorf("Expected ExternalUserID=%s, got %s", lookup.ExternalUserID, mapping.ExternalUserID)
		}
		if mapping.InternalUserID < 1 {
			t.Errorf("Expected InternalUserID >= 1, got %d", mapping.InternalUserID)
		}
		if mapping.Username == nil || *mapping.Username != "testuser" {
			t.Error("Username not set correctly")
		}
	})

	t.Run("returns existing mapping when found", func(t *testing.T) {
		lookup := &models.UserMappingLookup{
			Source:         "jellyfin",
			ServerID:       "jellyfin-server-1",
			ExternalUserID: "550e8400-e29b-41d4-a716-446655440001",
		}

		// Create first
		mapping1, created1, err := db.GetOrCreateUserMapping(ctx, lookup)
		if err != nil {
			t.Fatalf("First GetOrCreateUserMapping failed: %v", err)
		}
		if !created1 {
			t.Error("Expected created=true for first call")
		}

		// Get existing
		mapping2, created2, err := db.GetOrCreateUserMapping(ctx, lookup)
		if err != nil {
			t.Fatalf("Second GetOrCreateUserMapping failed: %v", err)
		}
		if created2 {
			t.Error("Expected created=false for existing mapping")
		}

		if mapping1.ID != mapping2.ID {
			t.Errorf("Expected same ID, got %d vs %d", mapping1.ID, mapping2.ID)
		}
		if mapping1.InternalUserID != mapping2.InternalUserID {
			t.Errorf("Expected same InternalUserID, got %d vs %d", mapping1.InternalUserID, mapping2.InternalUserID)
		}
	})

	t.Run("updates metadata when changed", func(t *testing.T) {
		lookup := &models.UserMappingLookup{
			Source:         "jellyfin",
			ServerID:       "jellyfin-server-1",
			ExternalUserID: "550e8400-e29b-41d4-a716-446655440002",
			Username:       strPtr("olduser"),
		}

		// Create with original username
		mapping1, _, err := db.GetOrCreateUserMapping(ctx, lookup)
		if err != nil {
			t.Fatalf("First GetOrCreateUserMapping failed: %v", err)
		}

		// Update lookup with new username
		lookup.Username = strPtr("newuser")
		mapping2, _, err := db.GetOrCreateUserMapping(ctx, lookup)
		if err != nil {
			t.Fatalf("Second GetOrCreateUserMapping failed: %v", err)
		}

		// Verify same mapping
		if mapping1.ID != mapping2.ID {
			t.Errorf("Expected same ID after update")
		}
	})

	t.Run("generates unique internal IDs", func(t *testing.T) {
		internalIDs := make(map[int]bool)

		for i := 0; i < 10; i++ {
			lookup := &models.UserMappingLookup{
				Source:         "plex",
				ServerID:       "plex-server-1",
				ExternalUserID: string(rune('a' + i)), // a, b, c, ...
			}

			mapping, _, err := db.GetOrCreateUserMapping(ctx, lookup)
			if err != nil {
				t.Fatalf("GetOrCreateUserMapping failed: %v", err)
			}

			if internalIDs[mapping.InternalUserID] {
				t.Errorf("Duplicate InternalUserID: %d", mapping.InternalUserID)
			}
			internalIDs[mapping.InternalUserID] = true
		}
	})
}

// TestGetUserMappingByExternal tests lookup by source+server+external_id
func TestGetUserMappingByExternal(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("returns mapping when exists", func(t *testing.T) {
		lookup := &models.UserMappingLookup{
			Source:         "emby",
			ServerID:       "emby-server-1",
			ExternalUserID: "emby-user-uuid-123",
		}

		created, _, err := db.GetOrCreateUserMapping(ctx, lookup)
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		found, err := db.GetUserMappingByExternal(ctx, lookup.Source, lookup.ServerID, lookup.ExternalUserID)
		if err != nil {
			t.Fatalf("GetUserMappingByExternal failed: %v", err)
		}

		if found.ID != created.ID {
			t.Errorf("Expected ID=%d, got %d", created.ID, found.ID)
		}
	})

	t.Run("returns error when not found", func(t *testing.T) {
		_, err := db.GetUserMappingByExternal(ctx, "nonexistent", "server", "user")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("Expected sql.ErrNoRows, got %v", err)
		}
	})

	t.Run("distinguishes between servers", func(t *testing.T) {
		externalID := "same-external-id"

		// Create mapping for server 1
		lookup1 := &models.UserMappingLookup{
			Source:         "plex",
			ServerID:       "plex-server-A",
			ExternalUserID: externalID,
		}
		mapping1, _, _ := db.GetOrCreateUserMapping(ctx, lookup1)

		// Create mapping for server 2 (same external ID, different server)
		lookup2 := &models.UserMappingLookup{
			Source:         "plex",
			ServerID:       "plex-server-B",
			ExternalUserID: externalID,
		}
		mapping2, _, _ := db.GetOrCreateUserMapping(ctx, lookup2)

		// Should be different mappings
		if mapping1.InternalUserID == mapping2.InternalUserID {
			t.Error("Different servers with same external ID should have different internal IDs")
		}

		// Verify lookups return correct mapping
		found1, _ := db.GetUserMappingByExternal(ctx, "plex", "plex-server-A", externalID)
		found2, _ := db.GetUserMappingByExternal(ctx, "plex", "plex-server-B", externalID)

		if found1.InternalUserID != mapping1.InternalUserID {
			t.Error("Server A lookup returned wrong mapping")
		}
		if found2.InternalUserID != mapping2.InternalUserID {
			t.Error("Server B lookup returned wrong mapping")
		}
	})
}

// TestGetUserMappingByInternal tests lookup by internal user ID
func TestGetUserMappingByInternal(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("returns all mappings for internal ID", func(t *testing.T) {
		// Create multiple mappings that would have the same internal ID
		// (This tests the case where mappings are linked across sources)
		lookup := &models.UserMappingLookup{
			Source:         "jellyfin",
			ServerID:       "jellyfin-lookup-test",
			ExternalUserID: "lookup-test-user",
		}
		mapping, _, _ := db.GetOrCreateUserMapping(ctx, lookup)

		found, err := db.GetUserMappingByInternal(ctx, mapping.InternalUserID)
		if err != nil {
			t.Fatalf("GetUserMappingByInternal failed: %v", err)
		}

		if len(found) < 1 {
			t.Error("Expected at least one mapping")
		}

		foundMatch := false
		for _, m := range found {
			if m.ID == mapping.ID {
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			t.Error("Did not find expected mapping in results")
		}
	})

	t.Run("returns empty slice for unknown internal ID", func(t *testing.T) {
		found, err := db.GetUserMappingByInternal(ctx, 999999)
		if err != nil {
			t.Fatalf("GetUserMappingByInternal failed: %v", err)
		}

		if len(found) != 0 {
			t.Errorf("Expected empty slice, got %d mappings", len(found))
		}
	})
}

// TestResolveUserID tests the convenience method for event processing
func TestResolveUserID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("resolves and returns consistent ID", func(t *testing.T) {
		username := "resolvetest"
		friendlyName := "Resolve Test"

		// First resolution
		id1, err := db.ResolveUserID(ctx, "plex", "plex-resolve-test", "12345", &username, &friendlyName)
		if err != nil {
			t.Fatalf("First ResolveUserID failed: %v", err)
		}

		// Second resolution (same user)
		id2, err := db.ResolveUserID(ctx, "plex", "plex-resolve-test", "12345", &username, &friendlyName)
		if err != nil {
			t.Fatalf("Second ResolveUserID failed: %v", err)
		}

		if id1 != id2 {
			t.Errorf("Expected same ID for same user, got %d vs %d", id1, id2)
		}
	})

	t.Run("handles nil username and friendlyName", func(t *testing.T) {
		id, err := db.ResolveUserID(ctx, "jellyfin", "jellyfin-nil-test", "uuid-nil-test", nil, nil)
		if err != nil {
			t.Fatalf("ResolveUserID with nils failed: %v", err)
		}

		if id < 1 {
			t.Errorf("Expected valid ID, got %d", id)
		}
	})
}

// TestUserMappingConcurrency tests thread safety of mapping operations
func TestUserMappingConcurrency(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()
	const numGoroutines = 10
	const numUsers = 5

	t.Run("concurrent creates for same user return same ID", func(t *testing.T) {
		lookup := &models.UserMappingLookup{
			Source:         "concurrent-test",
			ServerID:       "server-concurrent",
			ExternalUserID: "concurrent-user-1",
		}

		results := make(chan int, numGoroutines)
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				mapping, _, err := db.GetOrCreateUserMapping(ctx, lookup)
				if err != nil {
					t.Errorf("Concurrent GetOrCreateUserMapping failed: %v", err)
					return
				}
				results <- mapping.InternalUserID
			}()
		}

		wg.Wait()
		close(results)

		// All results should be the same ID
		var firstID int
		first := true
		for id := range results {
			if first {
				firstID = id
				first = false
			} else if id != firstID {
				t.Errorf("Got different IDs in concurrent test: %d vs %d", firstID, id)
			}
		}
	})

	t.Run("concurrent creates for different users get unique IDs", func(t *testing.T) {
		results := make(chan int, numUsers)
		var wg sync.WaitGroup

		for i := 0; i < numUsers; i++ {
			wg.Add(1)
			go func(userNum int) {
				defer wg.Done()
				lookup := &models.UserMappingLookup{
					Source:         "concurrent-unique",
					ServerID:       "server-unique",
					ExternalUserID: string(rune('A' + userNum)),
				}
				mapping, _, err := db.GetOrCreateUserMapping(ctx, lookup)
				if err != nil {
					t.Errorf("Concurrent GetOrCreateUserMapping failed: %v", err)
					return
				}
				results <- mapping.InternalUserID
			}(i)
		}

		wg.Wait()
		close(results)

		// All results should be unique
		seen := make(map[int]bool)
		for id := range results {
			if seen[id] {
				t.Errorf("Got duplicate ID in unique user test: %d", id)
			}
			seen[id] = true
		}
	})
}

// TestUserMappingEdgeCases tests edge cases and special characters
func TestUserMappingEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("handles special characters in external ID", func(t *testing.T) {
		specialIDs := []string{
			"user@domain.com",
			"user+tag@example.com",
			"user/with/slashes",
			"user with spaces",
			"user\twith\ttabs",
		}

		for _, extID := range specialIDs {
			lookup := &models.UserMappingLookup{
				Source:         "edge-case",
				ServerID:       "server-special",
				ExternalUserID: extID,
			}

			mapping, _, err := db.GetOrCreateUserMapping(ctx, lookup)
			if err != nil {
				t.Errorf("Failed for external ID %q: %v", extID, err)
				continue
			}

			// Verify round-trip
			found, err := db.GetUserMappingByExternal(ctx, lookup.Source, lookup.ServerID, extID)
			if err != nil {
				t.Errorf("Lookup failed for external ID %q: %v", extID, err)
				continue
			}

			if found.InternalUserID != mapping.InternalUserID {
				t.Errorf("Round-trip failed for %q", extID)
			}
		}
	})

	t.Run("handles Unicode in username", func(t *testing.T) {
		lookup := &models.UserMappingLookup{
			Source:         "unicode-test",
			ServerID:       "server-unicode",
			ExternalUserID: "unicode-user",
			Username:       strPtr("用户名"),
			FriendlyName:   strPtr("Пользователь"),
		}

		mapping, _, err := db.GetOrCreateUserMapping(ctx, lookup)
		if err != nil {
			t.Fatalf("Failed with Unicode: %v", err)
		}

		if mapping.Username == nil || *mapping.Username != "用户名" {
			t.Error("Unicode username not preserved")
		}
		if mapping.FriendlyName == nil || *mapping.FriendlyName != "Пользователь" {
			t.Error("Unicode friendly name not preserved")
		}
	})

	t.Run("handles very long external IDs", func(t *testing.T) {
		// Jellyfin/Emby UUIDs are 36 chars, but test longer
		longID := "very-long-external-user-id-that-exceeds-normal-uuid-length-" +
			"0123456789012345678901234567890123456789012345678901234567890123456789"

		lookup := &models.UserMappingLookup{
			Source:         "long-id-test",
			ServerID:       "server-long",
			ExternalUserID: longID,
		}

		mapping, _, err := db.GetOrCreateUserMapping(ctx, lookup)
		if err != nil {
			t.Fatalf("Failed with long ID: %v", err)
		}

		if mapping.ExternalUserID != longID {
			t.Error("Long external ID not preserved")
		}
	})
}

// TestGetUserMappingStats tests the statistics method
func TestGetUserMappingStats(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("returns stats for empty database", func(t *testing.T) {
		stats, err := db.GetUserMappingStats(ctx)
		if err != nil {
			t.Fatalf("GetUserMappingStats failed: %v", err)
		}

		if stats.TotalMappings != 0 {
			t.Errorf("Expected 0 total mappings, got %d", stats.TotalMappings)
		}
		if stats.UniqueInternalIDs != 0 {
			t.Errorf("Expected 0 unique IDs, got %d", stats.UniqueInternalIDs)
		}
	})

	t.Run("returns correct stats after creating mappings", func(t *testing.T) {
		// Create mappings from different sources
		sources := []string{"plex", "jellyfin", "emby"}
		for i, src := range sources {
			lookup := &models.UserMappingLookup{
				Source:         src,
				ServerID:       src + "-stats-server",
				ExternalUserID: string(rune('1' + i)),
			}
			_, _, _ = db.GetOrCreateUserMapping(ctx, lookup)
		}

		stats, err := db.GetUserMappingStats(ctx)
		if err != nil {
			t.Fatalf("GetUserMappingStats failed: %v", err)
		}

		if stats.TotalMappings < 3 {
			t.Errorf("Expected at least 3 total mappings, got %d", stats.TotalMappings)
		}

		// Check counts by source
		for _, src := range sources {
			if stats.BySource[src] < 1 {
				t.Errorf("Expected at least 1 mapping for source %s", src)
			}
		}
	})
}

// Helper functions
// Note: strPtr is defined in analytics_advanced_test.go
// Note: setupTestDB is defined in database_test.go
// Note: cleanupTestDB is defined in concurrent_test.go
