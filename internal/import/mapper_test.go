// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulliimport

import (
	"strings"
	"testing"
	"time"
)

func TestMapper_ToPlaybackEvent(t *testing.T) {
	mapper := NewMapper()

	t.Run("converts core fields correctly", func(t *testing.T) {
		now := time.Now()
		rec := TautulliRecord{
			ID:              123,
			SessionKey:      "session-key-123",
			StartedAt:       now,
			StoppedAt:       now.Add(time.Hour),
			UserID:          42,
			Username:        "testuser",
			IPAddress:       "192.168.1.100",
			Platform:        "Chrome",
			Player:          "Plex Web",
			PercentComplete: 75,
			PausedCounter:   120,
			MediaType:       "movie",
			Title:           "Test Movie",
			LocationType:    "lan",
		}

		event := mapper.ToPlaybackEvent(&rec)

		if event.SessionKey != rec.SessionKey {
			t.Errorf("SessionKey = %s, want %s", event.SessionKey, rec.SessionKey)
		}
		if event.UserID != rec.UserID {
			t.Errorf("UserID = %d, want %d", event.UserID, rec.UserID)
		}
		if event.Username != rec.Username {
			t.Errorf("Username = %s, want %s", event.Username, rec.Username)
		}
		if event.IPAddress != rec.IPAddress {
			t.Errorf("IPAddress = %s, want %s", event.IPAddress, rec.IPAddress)
		}
		if event.MediaType != rec.MediaType {
			t.Errorf("MediaType = %s, want %s", event.MediaType, rec.MediaType)
		}
		if event.Title != rec.Title {
			t.Errorf("Title = %s, want %s", event.Title, rec.Title)
		}
		if event.PercentComplete != rec.PercentComplete {
			t.Errorf("PercentComplete = %d, want %d", event.PercentComplete, rec.PercentComplete)
		}
		if event.Source != "tautulli-import" {
			t.Errorf("Source = %s, want tautulli-import", event.Source)
		}
	})

	t.Run("generates deterministic ID", func(t *testing.T) {
		rec := TautulliRecord{
			SessionKey: "session-key-123",
			StartedAt:  time.Unix(1700000000, 0),
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
		}

		event1 := mapper.ToPlaybackEvent(&rec)
		event2 := mapper.ToPlaybackEvent(&rec)

		if event1.ID != event2.ID {
			t.Errorf("IDs should be deterministic: got %s and %s", event1.ID, event2.ID)
		}
	})

	t.Run("generates different IDs for different records", func(t *testing.T) {
		rec1 := TautulliRecord{
			SessionKey: "session-key-1",
			StartedAt:  time.Unix(1700000000, 0),
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie 1",
		}
		rec2 := TautulliRecord{
			SessionKey: "session-key-2",
			StartedAt:  time.Unix(1700000000, 0),
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie 2",
		}

		event1 := mapper.ToPlaybackEvent(&rec1)
		event2 := mapper.ToPlaybackEvent(&rec2)

		if event1.ID == event2.ID {
			t.Error("Different records should have different IDs")
		}
	})

	t.Run("generates v2.3 correlation key", func(t *testing.T) {
		ratingKey := "12345"
		machineID := "machine-abc"
		rec := TautulliRecord{
			SessionKey: "session-key-123",
			StartedAt:  time.Date(2024, 1, 15, 10, 33, 0, 0, time.UTC),
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  &machineID,
		}

		event := mapper.ToPlaybackEvent(&rec)

		if event.CorrelationKey == nil {
			t.Fatal("CorrelationKey should not be nil")
		}

		// v2.3 format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
		key := *event.CorrelationKey

		// Should start with source
		if !strings.HasPrefix(key, "tautulli-import:") {
			t.Errorf("CorrelationKey should start with source: %s", key)
		}
		// Should contain server_id (default)
		if !strings.Contains(key, ":default:") {
			t.Errorf("CorrelationKey should contain server_id 'default': %s", key)
		}
		// Should contain user ID
		if !strings.Contains(key, ":42:") {
			t.Errorf("CorrelationKey should contain user ID: %s", key)
		}
		// Should contain rating key
		if !strings.Contains(key, ":12345:") {
			t.Errorf("CorrelationKey should contain rating key: %s", key)
		}
		// Should contain machine ID
		if !strings.Contains(key, ":machine-abc:") {
			t.Errorf("CorrelationKey should contain machine ID: %s", key)
		}
		// Time should use second precision (10:33:00)
		if !strings.Contains(key, "2024-01-15T10:33:00") {
			t.Errorf("CorrelationKey should contain second-precision timestamp: %s", key)
		}
		// Should end with session key
		if !strings.HasSuffix(key, ":session-key-123") {
			t.Errorf("CorrelationKey should end with session key: %s", key)
		}
	})

	t.Run("handles optional fields", func(t *testing.T) {
		parentTitle := "Season 1"
		grandparentTitle := "Test Show"
		year := 2024
		rec := TautulliRecord{
			SessionKey:       "session-key-123",
			StartedAt:        time.Now(),
			UserID:           42,
			Username:         "testuser",
			IPAddress:        "192.168.1.100",
			MediaType:        "episode",
			Title:            "Pilot",
			ParentTitle:      &parentTitle,
			GrandparentTitle: &grandparentTitle,
			Year:             &year,
		}

		event := mapper.ToPlaybackEvent(&rec)

		if event.ParentTitle == nil || *event.ParentTitle != parentTitle {
			t.Errorf("ParentTitle = %v, want %s", event.ParentTitle, parentTitle)
		}
		if event.GrandparentTitle == nil || *event.GrandparentTitle != grandparentTitle {
			t.Errorf("GrandparentTitle = %v, want %s", event.GrandparentTitle, grandparentTitle)
		}
		if event.Year == nil || *event.Year != year {
			t.Errorf("Year = %v, want %d", event.Year, year)
		}
	})
}

func TestMapper_ValidateRecord(t *testing.T) {
	mapper := NewMapper()

	tests := []struct {
		name    string
		record  TautulliRecord
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid record",
			record: TautulliRecord{
				SessionKey: "session-key-123",
				StartedAt:  time.Now(),
				UserID:     42,
				Username:   "testuser",
				IPAddress:  "192.168.1.100",
				MediaType:  "movie",
				Title:      "Test Movie",
			},
			wantErr: false,
		},
		{
			name: "missing session key",
			record: TautulliRecord{
				StartedAt: time.Now(),
				UserID:    42,
				Username:  "testuser",
				IPAddress: "192.168.1.100",
				MediaType: "movie",
				Title:     "Test Movie",
			},
			wantErr: true,
			errMsg:  "missing session_key",
		},
		{
			name: "missing started timestamp",
			record: TautulliRecord{
				SessionKey: "session-key-123",
				UserID:     42,
				Username:   "testuser",
				IPAddress:  "192.168.1.100",
				MediaType:  "movie",
				Title:      "Test Movie",
			},
			wantErr: true,
			errMsg:  "started timestamp",
		},
		{
			name: "invalid user ID",
			record: TautulliRecord{
				SessionKey: "session-key-123",
				StartedAt:  time.Now(),
				UserID:     0,
				Username:   "testuser",
				IPAddress:  "192.168.1.100",
				MediaType:  "movie",
				Title:      "Test Movie",
			},
			wantErr: true,
			errMsg:  "invalid user_id",
		},
		{
			name: "missing username",
			record: TautulliRecord{
				SessionKey: "session-key-123",
				StartedAt:  time.Now(),
				UserID:     42,
				IPAddress:  "192.168.1.100",
				MediaType:  "movie",
				Title:      "Test Movie",
			},
			wantErr: true,
			errMsg:  "missing username",
		},
		{
			name: "missing IP address",
			record: TautulliRecord{
				SessionKey: "session-key-123",
				StartedAt:  time.Now(),
				UserID:     42,
				Username:   "testuser",
				MediaType:  "movie",
				Title:      "Test Movie",
			},
			wantErr: true,
			errMsg:  "ip_address",
		},
		{
			name: "N/A IP address",
			record: TautulliRecord{
				SessionKey: "session-key-123",
				StartedAt:  time.Now(),
				UserID:     42,
				Username:   "testuser",
				IPAddress:  "N/A",
				MediaType:  "movie",
				Title:      "Test Movie",
			},
			wantErr: true,
			errMsg:  "ip_address",
		},
		{
			name: "missing media type",
			record: TautulliRecord{
				SessionKey: "session-key-123",
				StartedAt:  time.Now(),
				UserID:     42,
				Username:   "testuser",
				IPAddress:  "192.168.1.100",
				Title:      "Test Movie",
			},
			wantErr: true,
			errMsg:  "missing media_type",
		},
		{
			name: "missing title",
			record: TautulliRecord{
				SessionKey: "session-key-123",
				StartedAt:  time.Now(),
				UserID:     42,
				Username:   "testuser",
				IPAddress:  "192.168.1.100",
				MediaType:  "movie",
			},
			wantErr: true,
			errMsg:  "missing title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapper.ValidateRecord(&tt.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecord() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateRecord() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestMapper_FilterValidRecords(t *testing.T) {
	mapper := NewMapper()

	records := []TautulliRecord{
		// Valid record
		{
			SessionKey: "session-1",
			StartedAt:  time.Now(),
			UserID:     1,
			Username:   "user1",
			IPAddress:  "192.168.1.1",
			MediaType:  "movie",
			Title:      "Movie 1",
		},
		// Invalid: missing session key
		{
			StartedAt: time.Now(),
			UserID:    2,
			Username:  "user2",
			IPAddress: "192.168.1.2",
			MediaType: "movie",
			Title:     "Movie 2",
		},
		// Valid record
		{
			SessionKey: "session-3",
			StartedAt:  time.Now(),
			UserID:     3,
			Username:   "user3",
			IPAddress:  "192.168.1.3",
			MediaType:  "episode",
			Title:      "Episode 3",
		},
		// Invalid: N/A IP address
		{
			SessionKey: "session-4",
			StartedAt:  time.Now(),
			UserID:     4,
			Username:   "user4",
			IPAddress:  "N/A",
			MediaType:  "movie",
			Title:      "Movie 4",
		},
	}

	valid, skipped := mapper.FilterValidRecords(records)

	if len(valid) != 2 {
		t.Errorf("FilterValidRecords() got %d valid records, want 2", len(valid))
	}
	if skipped != 2 {
		t.Errorf("FilterValidRecords() skipped %d records, want 2", skipped)
	}
}

func TestMapper_ToPlaybackEvents(t *testing.T) {
	mapper := NewMapper()

	records := []TautulliRecord{
		{
			SessionKey: "session-1",
			StartedAt:  time.Now(),
			UserID:     1,
			Username:   "user1",
			IPAddress:  "192.168.1.1",
			MediaType:  "movie",
			Title:      "Movie 1",
		},
		{
			SessionKey: "session-2",
			StartedAt:  time.Now(),
			UserID:     2,
			Username:   "user2",
			IPAddress:  "192.168.1.2",
			MediaType:  "episode",
			Title:      "Episode 2",
		},
	}

	events := mapper.ToPlaybackEvents(records)

	if len(events) != 2 {
		t.Fatalf("ToPlaybackEvents() returned %d events, want 2", len(events))
	}

	if events[0].SessionKey != "session-1" {
		t.Errorf("First event SessionKey = %s, want session-1", events[0].SessionKey)
	}
	if events[1].SessionKey != "session-2" {
		t.Errorf("Second event SessionKey = %s, want session-2", events[1].SessionKey)
	}
}
