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
	"github.com/tomtom215/cartographus/internal/models"
)

func TestGetPlaybackEvents_Pagination(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	// Insert multiple playback events
	for i := 0; i < 5; i++ {
		event := &models.PlaybackEvent{
			SessionKey:      uuid.New().String(),
			StartedAt:       time.Now().Add(time.Duration(-i) * time.Hour),
			UserID:          i + 1,
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
	}

	// Get first 3 events
	events, err := db.GetPlaybackEvents(context.Background(), 3, 0)
	if err != nil {
		t.Fatalf("GetPlaybackEvents failed: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// Get next 2 events
	events, err = db.GetPlaybackEvents(context.Background(), 3, 3)
	if err != nil {
		t.Fatalf("GetPlaybackEvents (offset) failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events with offset, got %d", len(events))
	}
}

func TestGetPlaybackEventsWithCursor_Pagination(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	// Insert 10 playback events with distinct timestamps
	for i := 0; i < 10; i++ {
		event := &models.PlaybackEvent{
			SessionKey:      uuid.New().String(),
			StartedAt:       time.Now().Add(time.Duration(-i) * time.Hour),
			UserID:          i + 1,
			Username:        "testuser",
			IPAddress:       "192.168.1.100",
			MediaType:       "movie",
			Title:           fmt.Sprintf("Test Movie %d", i),
			Platform:        "Test Platform",
			Player:          "Test Player",
			LocationType:    "LAN",
			PercentComplete: 100,
		}

		err := db.InsertPlaybackEvent(event)
		if err != nil {
			t.Fatalf("InsertPlaybackEvent failed: %v", err)
		}
	}

	// First page - no cursor
	events, nextCursor, hasMore, err := db.GetPlaybackEventsWithCursor(context.Background(), 3, nil)
	if err != nil {
		t.Fatalf("GetPlaybackEventsWithCursor (first page) failed: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events on first page, got %d", len(events))
	}

	if !hasMore {
		t.Error("Expected hasMore=true for first page with 10 total events")
	}

	if nextCursor == nil {
		t.Fatal("Expected nextCursor to be non-nil for first page")
	}

	// Verify ordering (most recent first)
	if events[0].Title != "Test Movie 0" {
		t.Errorf("Expected first event to be 'Test Movie 0', got '%s'", events[0].Title)
	}

	// Second page - use cursor from first page
	events2, nextCursor2, hasMore2, err := db.GetPlaybackEventsWithCursor(context.Background(), 3, nextCursor)
	if err != nil {
		t.Fatalf("GetPlaybackEventsWithCursor (second page) failed: %v", err)
	}

	if len(events2) != 3 {
		t.Errorf("Expected 3 events on second page, got %d", len(events2))
	}

	if !hasMore2 {
		t.Error("Expected hasMore=true for second page with 10 total events")
	}

	// Verify no overlap with first page
	for _, e1 := range events {
		for _, e2 := range events2 {
			if e1.ID == e2.ID {
				t.Errorf("Found duplicate event ID %s across pages", e1.ID)
			}
		}
	}

	// Third page
	events3, nextCursor3, hasMore3, err := db.GetPlaybackEventsWithCursor(context.Background(), 3, nextCursor2)
	if err != nil {
		t.Fatalf("GetPlaybackEventsWithCursor (third page) failed: %v", err)
	}

	if len(events3) != 3 {
		t.Errorf("Expected 3 events on third page, got %d", len(events3))
	}

	if !hasMore3 {
		t.Error("Expected hasMore=true for third page with 10 total events")
	}

	// Fourth page - should have only 1 event and hasMore=false
	events4, nextCursor4, hasMore4, err := db.GetPlaybackEventsWithCursor(context.Background(), 3, nextCursor3)
	if err != nil {
		t.Fatalf("GetPlaybackEventsWithCursor (fourth page) failed: %v", err)
	}

	if len(events4) != 1 {
		t.Errorf("Expected 1 event on fourth page, got %d", len(events4))
	}

	if hasMore4 {
		t.Error("Expected hasMore=false for last page")
	}

	if nextCursor4 != nil {
		t.Error("Expected nextCursor to be nil on last page")
	}

	// Verify total unique events across all pages
	allEvents := append(append(append(events, events2...), events3...), events4...)
	if len(allEvents) != 10 {
		t.Errorf("Expected 10 total events across all pages, got %d", len(allEvents))
	}
}

func TestGetPlaybackEventsWithCursor_EmptyDatabase(t *testing.T) {

	db := setupTestDB(t)
	defer db.Close()

	events, nextCursor, hasMore, err := db.GetPlaybackEventsWithCursor(context.Background(), 10, nil)
	if err != nil {
		t.Fatalf("GetPlaybackEventsWithCursor failed: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events for empty database, got %d", len(events))
	}

	if hasMore {
		t.Error("Expected hasMore=false for empty database")
	}

	if nextCursor != nil {
		t.Error("Expected nextCursor=nil for empty database")
	}
}
