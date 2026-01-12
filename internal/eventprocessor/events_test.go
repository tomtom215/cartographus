// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package eventprocessor

import (
	"testing"
	"time"
)

func TestNewMediaEvent(t *testing.T) {
	event := NewMediaEvent("plex")

	if event.EventID == "" {
		t.Error("Expected EventID to be set")
	}
	if event.Source != "plex" {
		t.Errorf("Expected Source=plex, got %s", event.Source)
	}
	if event.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set")
	}
}

func TestMediaEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   *MediaEvent
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid event",
			event: &MediaEvent{
				EventID:   "test-id",
				Source:    "plex",
				UserID:    1,
				MediaType: "movie",
				Title:     "Test Movie",
			},
			wantErr: false,
		},
		{
			name: "missing event_id",
			event: &MediaEvent{
				Source:    "plex",
				UserID:    1,
				MediaType: "movie",
				Title:     "Test Movie",
			},
			wantErr: true,
			errMsg:  "event_id: required",
		},
		{
			name: "missing source",
			event: &MediaEvent{
				EventID:   "test-id",
				UserID:    1,
				MediaType: "movie",
				Title:     "Test Movie",
			},
			wantErr: true,
			errMsg:  "source: required",
		},
		{
			name: "missing user_id",
			event: &MediaEvent{
				EventID:   "test-id",
				Source:    "plex",
				MediaType: "movie",
				Title:     "Test Movie",
			},
			wantErr: true,
			errMsg:  "user_id: required",
		},
		{
			name: "missing media_type",
			event: &MediaEvent{
				EventID: "test-id",
				Source:  "plex",
				UserID:  1,
				Title:   "Test Movie",
			},
			wantErr: true,
			errMsg:  "media_type: required",
		},
		{
			name: "missing title",
			event: &MediaEvent{
				EventID:   "test-id",
				Source:    "plex",
				UserID:    1,
				MediaType: "movie",
			},
			wantErr: true,
			errMsg:  "title: required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("Expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMediaEvent_Topic(t *testing.T) {
	tests := []struct {
		source    string
		mediaType string
		expected  string
	}{
		{"plex", "movie", "playback.plex.movie"},
		{"tautulli", "episode", "playback.tautulli.episode"},
		{"jellyfin", "track", "playback.jellyfin.track"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			event := &MediaEvent{
				Source:    tt.source,
				MediaType: tt.mediaType,
			}
			if got := event.Topic(); got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestMediaEvent_IsComplete(t *testing.T) {
	t.Run("incomplete event", func(t *testing.T) {
		event := &MediaEvent{}
		if event.IsComplete() {
			t.Error("Expected IsComplete=false for event without StoppedAt")
		}
	})

	t.Run("complete event", func(t *testing.T) {
		now := time.Now()
		event := &MediaEvent{StoppedAt: &now}
		if !event.IsComplete() {
			t.Error("Expected IsComplete=true for event with StoppedAt")
		}
	})
}

func TestMediaEvent_Duration(t *testing.T) {
	now := time.Now()

	t.Run("uses PlayDuration if set", func(t *testing.T) {
		event := &MediaEvent{
			StartedAt:    now.Add(-10 * time.Minute),
			PlayDuration: 300, // 5 minutes
		}
		if got := event.Duration(); got != 300 {
			t.Errorf("Expected 300, got %d", got)
		}
	})

	t.Run("calculates from StoppedAt", func(t *testing.T) {
		stopped := now.Add(5 * time.Minute)
		event := &MediaEvent{
			StartedAt: now,
			StoppedAt: &stopped,
		}
		got := event.Duration()
		if got != 300 {
			t.Errorf("Expected 300, got %d", got)
		}
	})

	t.Run("calculates from now for in-progress", func(t *testing.T) {
		event := &MediaEvent{
			StartedAt: time.Now().Add(-10 * time.Second),
		}
		got := event.Duration()
		if got < 9 || got > 11 {
			t.Errorf("Expected ~10, got %d", got)
		}
	})
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "test_field", Message: "test message"}
	expected := "test_field: test message"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestMediaEvent_GenerateCorrelationKey(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 32, 45, 0, time.UTC)

	// v2.0 correlation key format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
	// When Source is empty, defaults to "unknown"
	// When ServerID is empty, defaults to "default"
	// When SessionKey is empty, trailing colon is still present (empty session_key)
	// Time bucket uses exact second-precision timestamps (not 5-minute buckets)

	tests := []struct {
		name     string
		event    *MediaEvent
		expected string
	}{
		{
			name: "basic correlation key with rating key and machine id",
			event: &MediaEvent{
				UserID:    12345,
				RatingKey: "54321",
				MachineID: "device123",
				StartedAt: baseTime,
			},
			expected: "unknown:default:12345:54321:device123:2024-01-15T10:32:45:", // Trailing colon for empty session_key
		},
		{
			name: "correlation key with unknown machine id",
			event: &MediaEvent{
				UserID:    12345,
				RatingKey: "54321",
				MachineID: "", // Empty MachineID defaults to "unknown"
				StartedAt: baseTime,
			},
			expected: "unknown:default:12345:54321:unknown:2024-01-15T10:32:45:",
		},
		{
			name: "correlation key falls back to title",
			event: &MediaEvent{
				UserID:    12345,
				Title:     "Test Movie",
				RatingKey: "",
				MachineID: "device123",
				StartedAt: baseTime,
			},
			expected: "unknown:default:12345:Test Movie:device123:2024-01-15T10:32:45:",
		},
		{
			name: "different user same content same time same device",
			event: &MediaEvent{
				UserID:    99999,
				RatingKey: "54321",
				MachineID: "device123",
				StartedAt: baseTime,
			},
			expected: "unknown:default:99999:54321:device123:2024-01-15T10:32:45:",
		},
		{
			name: "same user different timestamp",
			event: &MediaEvent{
				UserID:    12345,
				RatingKey: "54321",
				MachineID: "device123",
				StartedAt: baseTime.Add(6 * time.Minute), // 10:38:45
			},
			expected: "unknown:default:12345:54321:device123:2024-01-15T10:38:45:",
		},
		{
			name: "time differs by seconds creates different key",
			event: &MediaEvent{
				UserID:    12345,
				RatingKey: "54321",
				MachineID: "device123",
				StartedAt: baseTime.Add(2 * time.Minute), // 10:34:45 - different second-level timestamp
			},
			expected: "unknown:default:12345:54321:device123:2024-01-15T10:34:45:",
		},
		{
			name: "zero user id",
			event: &MediaEvent{
				UserID:    0,
				RatingKey: "54321",
				MachineID: "device123",
				StartedAt: baseTime,
			},
			expected: "unknown:default:0:54321:device123:2024-01-15T10:32:45:",
		},
		{
			name: "same user same content different devices",
			event: &MediaEvent{
				UserID:    12345,
				RatingKey: "54321",
				MachineID: "iphone",
				StartedAt: baseTime,
			},
			expected: "unknown:default:12345:54321:iphone:2024-01-15T10:32:45:",
		},
		{
			name: "with explicit source and server_id",
			event: &MediaEvent{
				Source:    "plex",
				ServerID:  "plex-server-1",
				UserID:    12345,
				RatingKey: "54321",
				MachineID: "device123",
				StartedAt: baseTime,
			},
			expected: "plex:plex-server-1:12345:54321:device123:2024-01-15T10:32:45:",
		},
		{
			name: "different servers same content same user",
			event: &MediaEvent{
				Source:    "plex",
				ServerID:  "plex-server-2",
				UserID:    12345,
				RatingKey: "54321",
				MachineID: "device123",
				StartedAt: baseTime,
			},
			expected: "plex:plex-server-2:12345:54321:device123:2024-01-15T10:32:45:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.GenerateCorrelationKey()
			if got != tt.expected {
				t.Errorf("GenerateCorrelationKey() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMediaEvent_SetCorrelationKey(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 32, 45, 0, time.UTC)

	event := &MediaEvent{
		UserID:    12345,
		RatingKey: "54321",
		MachineID: "device123",
		StartedAt: baseTime,
	}

	if event.CorrelationKey != "" {
		t.Error("CorrelationKey should be empty initially")
	}

	event.SetCorrelationKey()

	// v2.0 format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
	// Source defaults to "unknown", ServerID defaults to "default"
	// Time bucket uses exact second-precision timestamps
	// Trailing colon present when session_key is empty
	expected := "unknown:default:12345:54321:device123:2024-01-15T10:32:45:"
	if event.CorrelationKey != expected {
		t.Errorf("SetCorrelationKey() set CorrelationKey = %q, want %q", event.CorrelationKey, expected)
	}
}

func TestCorrelationKey_SourceIsolation(t *testing.T) {
	// v2.0 Behavior: Different sources get DIFFERENT correlation keys
	// This is intentional to prevent accidental cross-source data corruption.
	// For cross-source deduplication (e.g., Plex webhook + Tautulli sync),
	// events must have the SAME StartedAt timestamp (second-precision).

	baseTime := time.Date(2024, 1, 15, 10, 32, 0, 0, time.UTC)

	// Plex webhook arrives first
	plexEvent := &MediaEvent{
		Source:     SourcePlex,
		UserID:     12345,
		RatingKey:  "54321",
		MachineID:  "device123",
		StartedAt:  baseTime,
		SessionKey: "webhook-device123-54321", // No colons - colons are delimiters in correlation key
	}

	// Tautulli sync arrives later but reports the SAME StartedAt
	// (cross-source dedup requires matching timestamps)
	tautulliEvent := &MediaEvent{
		Source:     SourceTautulli,
		UserID:     12345,
		RatingKey:  "54321",
		MachineID:  "device123",
		StartedAt:  baseTime, // Same timestamp for cross-source matching
		SessionKey: "tautulli-session-abc123",
	}

	plexKey := plexEvent.GenerateCorrelationKey()
	tautulliKey := tautulliEvent.GenerateCorrelationKey()

	// v2.0: Different sources should have DIFFERENT correlation keys (due to source prefix)
	// This prevents accidental data corruption when sources have different data quality
	if plexKey == tautulliKey {
		t.Errorf("Different sources should have different correlation keys: %q == %q", plexKey, tautulliKey)
	}

	// Verify the keys contain their respective source identifiers and session keys
	// v2.3 format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
	// Note: SessionKeys should not contain colons since colons are used as delimiters
	expectedPlexKey := "plex:default:12345:54321:device123:2024-01-15T10:32:00:webhook-device123-54321"
	expectedTautulliKey := "tautulli:default:12345:54321:device123:2024-01-15T10:32:00:tautulli-session-abc123"
	if plexKey != expectedPlexKey {
		t.Errorf("Plex key format incorrect: got %q, want %q", plexKey, expectedPlexKey)
	}
	if tautulliKey != expectedTautulliKey {
		t.Errorf("Tautulli key format incorrect: got %q, want %q", tautulliKey, expectedTautulliKey)
	}

	// Cross-source deduplication works via getCrossSourceKey() in handlers.go,
	// which extracts the content-based portion (parts 2-6: server_id through time_bucket).
	// The session_key is intentionally excluded to allow matching across sources.
	// This allows "plex:default:...:timestamp:session1" and "tautulli:default:...:timestamp:session2" to match
	// on their content portion (server_id:user_id:rating_key:machine_id:time_bucket).
}

func TestCorrelationKey_SameSourceSameServerDeduplication(t *testing.T) {
	// Same source + same server + same user/content/device/time = SAME key
	// This is the correct deduplication behavior within a single source
	// NOTE: Exact timestamp matching is required (second-precision)

	baseTime := time.Date(2024, 1, 15, 10, 32, 0, 0, time.UTC)

	event1 := &MediaEvent{
		Source:    SourcePlex,
		ServerID:  "plex-server-1",
		UserID:    12345,
		RatingKey: "54321",
		MachineID: "device123",
		StartedAt: baseTime,
	}

	// Same playback reported again (e.g., from retry or duplicate notification)
	// Must have EXACT same StartedAt for deduplication to work
	event2 := &MediaEvent{
		Source:    SourcePlex,
		ServerID:  "plex-server-1",
		UserID:    12345,
		RatingKey: "54321",
		MachineID: "device123",
		StartedAt: baseTime, // Same exact timestamp for deduplication
	}

	key1 := event1.GenerateCorrelationKey()
	key2 := event2.GenerateCorrelationKey()

	// Same source + same server + same timestamp = same correlation key (deduplication works)
	if key1 != key2 {
		t.Errorf("Same source+server events should have same correlation key: %q != %q", key1, key2)
	}
}

func TestCorrelationKey_MultiServerDeduplication(t *testing.T) {
	// Different servers should have DIFFERENT correlation keys
	// even for the same content/user (prevents false deduplication)

	baseTime := time.Date(2024, 1, 15, 10, 32, 0, 0, time.UTC)

	// User watching on Plex Server A
	serverAEvent := &MediaEvent{
		Source:    SourcePlex,
		ServerID:  "plex-server-A",
		UserID:    12345,
		RatingKey: "54321",
		MachineID: "device123",
		StartedAt: baseTime,
	}

	// Same user, same content on Plex Server B (different library instance)
	serverBEvent := &MediaEvent{
		Source:    SourcePlex,
		ServerID:  "plex-server-B",
		UserID:    12345,
		RatingKey: "54321", // Could be same rating key across servers
		MachineID: "device123",
		StartedAt: baseTime,
	}

	keyA := serverAEvent.GenerateCorrelationKey()
	keyB := serverBEvent.GenerateCorrelationKey()

	// Different servers = different correlation keys
	if keyA == keyB {
		t.Errorf("Different servers should have different correlation keys: %q == %q", keyA, keyB)
	}

	// Verify correct format (exact timestamp, trailing colon for empty session_key)
	if keyA != "plex:plex-server-A:12345:54321:device123:2024-01-15T10:32:00:" {
		t.Errorf("Server A key format incorrect: %q", keyA)
	}
	if keyB != "plex:plex-server-B:12345:54321:device123:2024-01-15T10:32:00:" {
		t.Errorf("Server B key format incorrect: %q", keyB)
	}
}

func TestCorrelationKey_DifferentPlaybacksSameContent(t *testing.T) {
	// Verify that different playbacks of the same content get different keys
	// (separated by time bucket)

	user1Morning := &MediaEvent{
		UserID:    12345,
		RatingKey: "54321",
		MachineID: "device123",
		StartedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	user1Evening := &MediaEvent{
		UserID:    12345,
		RatingKey: "54321",
		MachineID: "device123",
		StartedAt: time.Date(2024, 1, 15, 20, 30, 0, 0, time.UTC), // 10 hours later
	}

	key1 := user1Morning.GenerateCorrelationKey()
	key2 := user1Evening.GenerateCorrelationKey()

	if key1 == key2 {
		t.Errorf("Different playback times should have different correlation keys: %q == %q", key1, key2)
	}
}

func TestCorrelationKey_SharedAccountMultiDevice(t *testing.T) {
	// CRITICAL: Verify that the same user watching same content on different devices
	// generates DIFFERENT correlation keys (no false positive deduplication)

	baseTime := time.Date(2024, 1, 15, 10, 32, 0, 0, time.UTC)

	// Same user watching same movie on iPhone
	iphoneEvent := &MediaEvent{
		UserID:    12345,
		RatingKey: "54321",
		MachineID: "iphone-abc123",
		StartedAt: baseTime,
	}

	// Same user watching same movie on Apple TV (shared account scenario)
	appletvEvent := &MediaEvent{
		UserID:    12345,
		RatingKey: "54321",
		MachineID: "appletv-xyz789",
		StartedAt: baseTime, // Same time
	}

	iphoneKey := iphoneEvent.GenerateCorrelationKey()
	appletvKey := appletvEvent.GenerateCorrelationKey()

	// Keys should be DIFFERENT because devices are different
	if iphoneKey == appletvKey {
		t.Errorf("Multi-device playbacks should have different correlation keys: %q == %q", iphoneKey, appletvKey)
	}

	// v2.0 format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
	// Source defaults to "unknown", ServerID defaults to "default"
	// Time bucket uses exact second-precision timestamps
	// Trailing colon present when session_key is empty
	if iphoneKey != "unknown:default:12345:54321:iphone-abc123:2024-01-15T10:32:00:" {
		t.Errorf("iPhone key format unexpected: %q", iphoneKey)
	}
	if appletvKey != "unknown:default:12345:54321:appletv-xyz789:2024-01-15T10:32:00:" {
		t.Errorf("AppleTV key format unexpected: %q", appletvKey)
	}
}

func TestFormatInt(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{12345, "12345"},
		{999999999, "999999999"},
		{-1, "-1"},
		{-42, "-42"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatInt(tt.input)
			if got != tt.expected {
				t.Errorf("formatInt(%d) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEventConstants(t *testing.T) {
	// Verify constants are defined correctly
	if EventTypePlaybackStart != "start" {
		t.Errorf("Expected EventTypePlaybackStart=start, got %s", EventTypePlaybackStart)
	}
	if EventTypePlaybackStop != "stop" {
		t.Errorf("Expected EventTypePlaybackStop=stop, got %s", EventTypePlaybackStop)
	}
	if SourcePlex != "plex" {
		t.Errorf("Expected SourcePlex=plex, got %s", SourcePlex)
	}
	if SourceTautulli != "tautulli" {
		t.Errorf("Expected SourceTautulli=tautulli, got %s", SourceTautulli)
	}
	if SourceJellyfin != "jellyfin" {
		t.Errorf("Expected SourceJellyfin=jellyfin, got %s", SourceJellyfin)
	}
	if MediaTypeMovie != "movie" {
		t.Errorf("Expected MediaTypeMovie=movie, got %s", MediaTypeMovie)
	}
	if MediaTypeEpisode != "episode" {
		t.Errorf("Expected MediaTypeEpisode=episode, got %s", MediaTypeEpisode)
	}
	if MediaTypeTrack != "track" {
		t.Errorf("Expected MediaTypeTrack=track, got %s", MediaTypeTrack)
	}
	if TranscodeDecisionDirectPlay != "direct play" {
		t.Errorf("Expected TranscodeDecisionDirectPlay='direct play', got %s", TranscodeDecisionDirectPlay)
	}
	if LocationTypeWAN != "wan" {
		t.Errorf("Expected LocationTypeWAN=wan, got %s", LocationTypeWAN)
	}
	if LocationTypeLAN != "lan" {
		t.Errorf("Expected LocationTypeLAN=lan, got %s", LocationTypeLAN)
	}
}
