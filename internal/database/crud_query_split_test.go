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

func TestSessionKeyExists_NotExists(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	exists, err := db.SessionKeyExists(context.Background(), "nonexistent-session-key")
	if err != nil {
		t.Fatalf("SessionKeyExists failed: %v", err)
	}

	if exists {
		t.Error("Expected session key to not exist")
	}
}

func TestSessionKeyExists_Exists(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	sessionKey := "test-session-" + uuid.New().String()

	// Insert playback event
	event := &models.PlaybackEvent{
		SessionKey:      sessionKey,
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

	// Check if session key exists
	exists, err := db.SessionKeyExists(context.Background(), sessionKey)
	if err != nil {
		t.Fatalf("SessionKeyExists failed: %v", err)
	}

	if !exists {
		t.Error("Expected session key to exist")
	}
}

func TestGetGeolocation_NotFound(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	geo, err := db.GetGeolocation(context.Background(), "192.168.99.99")
	if err != nil {
		t.Fatalf("GetGeolocation failed: %v", err)
	}

	if geo != nil {
		t.Error("Expected geolocation to be nil for non-existent IP")
	}
}

func TestGetLastPlaybackTime_NoPlaybacks(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	lastTime, err := db.GetLastPlaybackTime(context.Background())
	if err != nil {
		t.Fatalf("GetLastPlaybackTime failed: %v", err)
	}

	if lastTime != nil {
		t.Error("Expected nil for database with no playbacks")
	}
}

func TestGetLastPlaybackTime_WithPlaybacks(t *testing.T) {
	// Safe to parallelize - each test uses isolated setupTestDB(t)

	db := setupTestDB(t)
	defer db.Close()

	// Insert a playback event
	now := time.Now()
	event := &models.PlaybackEvent{
		SessionKey:      "test-session-" + uuid.New().String(),
		StartedAt:       now,
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

	lastTime, err := db.GetLastPlaybackTime(context.Background())
	if err != nil {
		t.Fatalf("GetLastPlaybackTime failed: %v", err)
	}

	if lastTime == nil {
		t.Fatal("Expected non-nil lastTime")
	}

	// Times should be close (within 1 second due to database precision)
	if lastTime.Sub(now).Abs() > time.Second {
		t.Errorf("Expected lastTime close to %v, got %v", now, *lastTime)
	}
}
