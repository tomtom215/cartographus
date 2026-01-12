// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package eventprocessor

import (
	"testing"
	"time"

	"github.com/goccy/go-json"
)

func TestSerializer_Marshal(t *testing.T) {
	serializer := NewSerializer()

	t.Run("valid event", func(t *testing.T) {
		event := &MediaEvent{
			EventID:   "test-id",
			Source:    "plex",
			UserID:    1,
			Username:  "testuser",
			MediaType: "movie",
			Title:     "Test Movie",
			StartedAt: time.Now(),
		}

		data, err := serializer.Marshal(event)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Error("Expected non-empty data")
		}

		// Verify JSON structure
		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}
		if decoded["event_id"] != "test-id" {
			t.Errorf("Expected event_id=test-id, got %v", decoded["event_id"])
		}
		if decoded["source"] != "plex" {
			t.Errorf("Expected source=plex, got %v", decoded["source"])
		}
	})

	t.Run("invalid event - missing required field", func(t *testing.T) {
		event := &MediaEvent{
			// Missing required fields
		}

		_, err := serializer.Marshal(event)
		if err == nil {
			t.Error("Expected validation error")
		}
	})
}

func TestSerializer_Unmarshal(t *testing.T) {
	serializer := NewSerializer()

	t.Run("valid JSON", func(t *testing.T) {
		data := []byte(`{
			"event_id": "test-id",
			"source": "plex",
			"user_id": 1,
			"username": "testuser",
			"media_type": "movie",
			"title": "Test Movie",
			"timestamp": "2025-01-01T12:00:00Z"
		}`)

		event, err := serializer.Unmarshal(data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if event.EventID != "test-id" {
			t.Errorf("Expected EventID=test-id, got %s", event.EventID)
		}
		if event.Source != "plex" {
			t.Errorf("Expected Source=plex, got %s", event.Source)
		}
		if event.UserID != 1 {
			t.Errorf("Expected UserID=1, got %d", event.UserID)
		}
		if event.Title != "Test Movie" {
			t.Errorf("Expected Title='Test Movie', got %s", event.Title)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		data := []byte(`{invalid json}`)

		_, err := serializer.Unmarshal(data)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("optional fields", func(t *testing.T) {
		data := []byte(`{
			"event_id": "test-id",
			"source": "plex",
			"user_id": 1,
			"media_type": "movie",
			"title": "Test Movie",
			"parent_title": "Season 1",
			"grandparent_title": "Test Show",
			"transcode_decision": "transcode",
			"video_resolution": "1080p",
			"secure": true,
			"local": false
		}`)

		event, err := serializer.Unmarshal(data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if event.ParentTitle != "Season 1" {
			t.Errorf("Expected ParentTitle='Season 1', got %s", event.ParentTitle)
		}
		if event.GrandparentTitle != "Test Show" {
			t.Errorf("Expected GrandparentTitle='Test Show', got %s", event.GrandparentTitle)
		}
		if event.TranscodeDecision != "transcode" {
			t.Errorf("Expected TranscodeDecision='transcode', got %s", event.TranscodeDecision)
		}
		if !event.Secure {
			t.Error("Expected Secure=true")
		}
		if event.Local {
			t.Error("Expected Local=false")
		}
	})
}

func TestSerializeEvent(t *testing.T) {
	event := &MediaEvent{
		EventID:   "test-id",
		Source:    "plex",
		UserID:    1,
		MediaType: "movie",
		Title:     "Test Movie",
	}

	data, err := SerializeEvent(event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty data")
	}
}

func TestDeserializeEvent(t *testing.T) {
	data := []byte(`{
		"event_id": "test-id",
		"source": "plex",
		"user_id": 1,
		"media_type": "movie",
		"title": "Test Movie"
	}`)

	event, err := DeserializeEvent(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if event.EventID != "test-id" {
		t.Errorf("Expected EventID=test-id, got %s", event.EventID)
	}
}

func TestRoundTrip(t *testing.T) {
	serializer := NewSerializer()

	now := time.Now().UTC().Truncate(time.Second)
	stopped := now.Add(30 * time.Minute)

	original := &MediaEvent{
		EventID:           "round-trip-test",
		Source:            "tautulli",
		Timestamp:         now,
		UserID:            42,
		Username:          "roundtrip",
		MediaType:         "episode",
		Title:             "Test Episode",
		ParentTitle:       "Season 1",
		GrandparentTitle:  "Test Show",
		RatingKey:         "12345",
		StartedAt:         now,
		StoppedAt:         &stopped,
		PercentComplete:   75,
		PlayDuration:      1800,
		PausedCounter:     2,
		Platform:          "Roku",
		Player:            "Roku Ultra",
		IPAddress:         "192.168.1.100",
		LocationType:      "lan",
		TranscodeDecision: "direct play",
		VideoResolution:   "1080p",
		VideoCodec:        "h264",
		VideoDynamicRange: "SDR",
		AudioCodec:        "aac",
		AudioChannels:     2,
		StreamBitrate:     5000,
		Secure:            true,
		Local:             true,
		Relayed:           false,
	}

	// Marshal
	data, err := serializer.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal
	decoded, err := serializer.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Compare fields
	if decoded.EventID != original.EventID {
		t.Errorf("EventID mismatch: %s != %s", decoded.EventID, original.EventID)
	}
	if decoded.Source != original.Source {
		t.Errorf("Source mismatch: %s != %s", decoded.Source, original.Source)
	}
	if decoded.UserID != original.UserID {
		t.Errorf("UserID mismatch: %d != %d", decoded.UserID, original.UserID)
	}
	if decoded.Username != original.Username {
		t.Errorf("Username mismatch: %s != %s", decoded.Username, original.Username)
	}
	if decoded.MediaType != original.MediaType {
		t.Errorf("MediaType mismatch: %s != %s", decoded.MediaType, original.MediaType)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title mismatch: %s != %s", decoded.Title, original.Title)
	}
	if decoded.ParentTitle != original.ParentTitle {
		t.Errorf("ParentTitle mismatch: %s != %s", decoded.ParentTitle, original.ParentTitle)
	}
	if decoded.GrandparentTitle != original.GrandparentTitle {
		t.Errorf("GrandparentTitle mismatch: %s != %s", decoded.GrandparentTitle, original.GrandparentTitle)
	}
	if decoded.PercentComplete != original.PercentComplete {
		t.Errorf("PercentComplete mismatch: %d != %d", decoded.PercentComplete, original.PercentComplete)
	}
	if decoded.PlayDuration != original.PlayDuration {
		t.Errorf("PlayDuration mismatch: %d != %d", decoded.PlayDuration, original.PlayDuration)
	}
	if decoded.TranscodeDecision != original.TranscodeDecision {
		t.Errorf("TranscodeDecision mismatch: %s != %s", decoded.TranscodeDecision, original.TranscodeDecision)
	}
	if decoded.Secure != original.Secure {
		t.Errorf("Secure mismatch: %v != %v", decoded.Secure, original.Secure)
	}
	if decoded.Local != original.Local {
		t.Errorf("Local mismatch: %v != %v", decoded.Local, original.Local)
	}
}
