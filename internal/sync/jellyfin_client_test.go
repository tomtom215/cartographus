// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// ============================================================================
// Jellyfin Client Constructor Tests
// ============================================================================

func TestNewJellyfinClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		apiKey  string
		userID  string
		wantURL string
	}{
		{
			name:    "basic URL",
			baseURL: "http://localhost:8096",
			apiKey:  "test-api-key",
			userID:  "",
			wantURL: "http://localhost:8096",
		},
		{
			name:    "URL with trailing slash",
			baseURL: "http://localhost:8096/",
			apiKey:  "test-api-key",
			userID:  "",
			wantURL: "http://localhost:8096",
		},
		{
			name:    "HTTPS URL",
			baseURL: "https://jellyfin.example.com/",
			apiKey:  "test-api-key",
			userID:  "user-123",
			wantURL: "https://jellyfin.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewJellyfinClient(tt.baseURL, tt.apiKey, tt.userID)
			checkStringEqual(t, "baseURL", client.baseURL, tt.wantURL)
			checkStringEqual(t, "apiKey", client.apiKey, tt.apiKey)
			checkStringEqual(t, "userID", client.userID, tt.userID)
			checkTrue(t, "httpClient not nil", client.httpClient != nil)
		})
	}
}

// ============================================================================
// GetSessions Tests
// ============================================================================

func TestJellyfinClientGetSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/Sessions")
		checkStringEqual(t, "method", r.Method, "GET")
		verifyJellyfinHeaders(t, r)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jellyfinSessionsResponse))
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-api-key", "")
	sessions, err := client.GetSessions(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "sessions", len(sessions), 2)

	// Verify first session (active playback)
	session := sessions[0]
	checkStringEqual(t, "session.ID", session.ID, "session-123")
	checkStringEqual(t, "session.UserName", session.UserName, "TestUser")
	checkStringEqual(t, "session.DeviceName", session.DeviceName, "Living Room TV")
	checkStringEqual(t, "session.Client", session.Client, "Jellyfin Web")
	checkTrue(t, "NowPlayingItem not nil", session.NowPlayingItem != nil)
	checkStringEqual(t, "NowPlayingItem.Name", session.NowPlayingItem.Name, "Inception")
	checkStringEqual(t, "NowPlayingItem.Type", session.NowPlayingItem.Type, "Movie")

	// Verify second session (idle)
	idleSession := sessions[1]
	checkStringEqual(t, "idleSession.ID", idleSession.ID, "session-456")
	checkNil(t, "idleSession.NowPlayingItem", idleSession.NowPlayingItem == nil)
}

const jellyfinSessionsResponse = `[
	{
		"Id": "session-123",
		"UserId": "user-abc",
		"UserName": "TestUser",
		"Client": "Jellyfin Web",
		"DeviceId": "device-xyz",
		"DeviceName": "Living Room TV",
		"DeviceType": "Browser",
		"ApplicationVersion": "10.8.0",
		"RemoteEndPoint": "192.168.1.100:52345",
		"LastActivityDate": "2024-01-15T10:30:00.0000000Z",
		"NowPlayingItem": {
			"Id": "item-12345",
			"Name": "Inception",
			"Type": "Movie",
			"MediaType": "Video",
			"ProductionYear": 2010,
			"RunTimeTicks": 88800000000,
			"ProviderIds": {
				"Imdb": "tt1375666",
				"Tmdb": "27205"
			},
			"MediaStreams": [
				{
					"Type": "Video",
					"Codec": "h264",
					"Width": 1920,
					"Height": 1080,
					"Index": 0
				},
				{
					"Type": "Audio",
					"Codec": "aac",
					"Channels": 6,
					"Index": 1
				}
			]
		},
		"PlayState": {
			"PositionTicks": 36000000000,
			"CanSeek": true,
			"IsPaused": false,
			"IsMuted": false,
			"PlayMethod": "DirectPlay"
		},
		"Capabilities": {
			"SupportsMediaControl": true,
			"SupportsPersistentIdentifier": true
		}
	},
	{
		"Id": "session-456",
		"UserId": "user-def",
		"UserName": "AnotherUser",
		"Client": "Jellyfin Mobile",
		"DeviceId": "device-mobile",
		"DeviceName": "iPhone",
		"RemoteEndPoint": "10.0.0.50",
		"LastActivityDate": "2024-01-15T09:00:00.0000000Z"
	}
]`

func TestJellyfinClientGetSessionsError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       "Invalid API key",
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			body:       "Server error",
		},
		{
			name:       "service unavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.body != "" {
					_, _ = w.Write([]byte(tt.body))
				}
			}))
			defer server.Close()

			client := NewJellyfinClient(server.URL, "test-api-key", "")
			_, err := client.GetSessions(context.Background())

			checkError(t, err)
		})
	}
}

func TestJellyfinClientGetSessionsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-api-key", "")
	_, err := client.GetSessions(context.Background())

	checkError(t, err)
	checkErrorContains(t, err, "failed to decode")
}

func TestJellyfinClientGetSessionsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-api-key", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetSessions(ctx)
	checkError(t, err)
}

// ============================================================================
// GetActiveSessions Tests
// ============================================================================

func TestJellyfinClientGetActiveSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jellyfinSessionsResponse))
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-api-key", "")
	sessions, err := client.GetActiveSessions(context.Background())

	checkNoError(t, err)
	// Only one session has NowPlayingItem
	checkSliceLen(t, "active sessions", len(sessions), 1)
	checkStringEqual(t, "active session ID", sessions[0].ID, "session-123")
}

func TestJellyfinClientGetActiveSessionsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-api-key", "")
	sessions, err := client.GetActiveSessions(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "active sessions", len(sessions), 0)
}

// ============================================================================
// GetSystemInfo Tests
// ============================================================================

func TestJellyfinClientGetSystemInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/System/Info")
		verifyJellyfinHeaders(t, r)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jellyfinSystemInfoResponse))
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-api-key", "")
	info, err := client.GetSystemInfo(context.Background())

	checkNoError(t, err)
	checkStringEqual(t, "ServerName", info.ServerName, "My Jellyfin Server")
	checkStringEqual(t, "Version", info.Version, "10.8.13")
	checkStringEqual(t, "ID", info.ID, "server-id-12345")
	checkStringEqual(t, "OperatingSystem", info.OperatingSystem, "Linux")
	checkTrue(t, "HasUpdateAvailable should be false", !info.HasUpdateAvailable)
}

const jellyfinSystemInfoResponse = `{
	"ServerName": "My Jellyfin Server",
	"Version": "10.8.13",
	"Id": "server-id-12345",
	"OperatingSystem": "Linux",
	"HasUpdateAvailable": false
}`

func TestJellyfinClientGetSystemInfoError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Access denied"))
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-api-key", "")
	_, err := client.GetSystemInfo(context.Background())

	checkError(t, err)
	checkErrorContains(t, err, "403")
}

// ============================================================================
// GetUsers Tests
// ============================================================================

func TestJellyfinClientGetUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/Users")
		verifyJellyfinHeaders(t, r)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jellyfinUsersResponse))
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-api-key", "")
	users, err := client.GetUsers(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "users", len(users), 2)
	checkStringEqual(t, "user[0].ID", users[0].ID, "user-abc-123")
	checkStringEqual(t, "user[0].Name", users[0].Name, "Admin")
	checkStringEqual(t, "user[1].ID", users[1].ID, "user-def-456")
	checkStringEqual(t, "user[1].Name", users[1].Name, "Guest")
}

const jellyfinUsersResponse = `[
	{"Id": "user-abc-123", "Name": "Admin"},
	{"Id": "user-def-456", "Name": "Guest"}
]`

func TestJellyfinClientGetUsersError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-api-key", "")
	_, err := client.GetUsers(context.Background())

	checkError(t, err)
}

// ============================================================================
// Ping Tests
// ============================================================================

func TestJellyfinClientPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/System/Ping")
		verifyJellyfinHeaders(t, r)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewJellyfinClient(server.URL, "test-api-key", "")
	err := client.Ping(context.Background())

	checkNoError(t, err)
}

func TestJellyfinClientPingError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"service unavailable", http.StatusServiceUnavailable},
		{"not found", http.StatusNotFound},
		{"internal error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewJellyfinClient(server.URL, "test-api-key", "")
			err := client.Ping(context.Background())

			checkError(t, err)
		})
	}
}

func TestJellyfinClientPingConnectionError(t *testing.T) {
	// Use a URL that won't connect
	client := NewJellyfinClient("http://localhost:1", "test-api-key", "")
	err := client.Ping(context.Background())

	checkError(t, err)
}

// ============================================================================
// StopSession Tests
// ============================================================================

func TestJellyfinClientStopSession(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"success with 204", http.StatusNoContent, false},
		{"success with 200", http.StatusOK, false},
		{"forbidden", http.StatusForbidden, true},
		{"not found", http.StatusNotFound, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				checkStringEqual(t, "method", r.Method, "POST")
				checkStringEqual(t, "path", r.URL.Path, "/Sessions/session-123/Playing/Stop")
				verifyJellyfinHeaders(t, r)
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewJellyfinClient(server.URL, "test-api-key", "")
			err := client.StopSession(context.Background(), "session-123")

			if tt.wantErr {
				checkError(t, err)
			} else {
				checkNoError(t, err)
			}
		})
	}
}

// ============================================================================
// GetWebSocketURL Tests
// ============================================================================

func TestJellyfinClientGetWebSocketURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		apiKey  string
		userID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "http to ws",
			baseURL: "http://localhost:8096",
			apiKey:  "test-key",
			userID:  "",
			wantURL: "ws://localhost:8096/socket?api_key=test-key&deviceId=cartographus",
		},
		{
			name:    "https to wss",
			baseURL: "https://jellyfin.example.com",
			apiKey:  "api-key-123",
			userID:  "",
			wantURL: "wss://jellyfin.example.com/socket?api_key=api-key-123&deviceId=cartographus",
		},
		{
			name:    "with user ID",
			baseURL: "http://localhost:8096",
			apiKey:  "test-key",
			userID:  "user-abc",
			wantURL: "ws://localhost:8096/socket?api_key=test-key&deviceId=cartographus-user-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewJellyfinClient(tt.baseURL, tt.apiKey, tt.userID)
			url, err := client.GetWebSocketURL()

			if tt.wantErr {
				checkError(t, err)
				return
			}

			checkNoError(t, err)
			checkStringEqual(t, "WebSocket URL", url, tt.wantURL)
		})
	}
}

// ============================================================================
// SessionToPlaybackEvent Tests
// ============================================================================

func TestSessionToPlaybackEventNilSession(t *testing.T) {
	event := SessionToPlaybackEvent(nil, "server-1")
	checkNil(t, "event should be nil for nil session", event == nil)
}

func TestSessionToPlaybackEventNoNowPlayingItem(t *testing.T) {
	session := &models.JellyfinSession{
		ID:       "session-123",
		UserName: "TestUser",
	}
	event := SessionToPlaybackEvent(session, "server-1")
	checkNil(t, "event should be nil when no NowPlayingItem", event == nil)
}

func TestSessionToPlaybackEventMovie(t *testing.T) {
	session := &models.JellyfinSession{
		ID:             "session-123",
		UserID:         "user-abc",
		UserName:       "TestUser",
		Client:         "Jellyfin Web",
		DeviceID:       "device-xyz",
		DeviceName:     "Living Room TV",
		RemoteEndPoint: "192.168.1.100:52345",
		NowPlayingItem: &models.JellyfinNowPlayingItem{
			ID:             "item-12345",
			Name:           "Inception",
			Type:           "Movie",
			MediaType:      "Video",
			ProductionYear: 2010,
			RunTimeTicks:   88800000000, // 2h 28m
			ProviderIDs: map[string]string{
				"Imdb": "tt1375666",
				"Tmdb": "27205",
			},
			MediaStreams: []models.JellyfinMediaStream{
				{Type: "Video", Codec: "h264", Width: 1920, Height: 1080},
			},
		},
		PlayState: &models.JellyfinPlayState{
			PositionTicks: 36000000000, // 1 hour
			IsPaused:      false,
			PlayMethod:    "DirectPlay",
		},
	}

	event := SessionToPlaybackEvent(session, "server-1")

	checkTrue(t, "event should not be nil", event != nil)
	checkStringEqual(t, "Source", event.Source, "jellyfin")
	checkStringEqual(t, "SessionKey", event.SessionKey, "session-123")
	checkStringEqual(t, "Username", event.Username, "TestUser")
	checkStringEqual(t, "Platform", event.Platform, "Jellyfin Web")
	checkStringEqual(t, "Player", event.Player, "Living Room TV")
	checkStringEqual(t, "MediaType", event.MediaType, "movie")
	checkStringEqual(t, "Title", event.Title, "Inception")
	checkIntPtrEqual(t, "Year", event.Year, 2010)
	checkStringEqual(t, "IPAddress", event.IPAddress, "192.168.1.100")
	checkStringPtrEqual(t, "TranscodeDecision", event.TranscodeDecision, "direct play")
	checkStringPtrEqual(t, "RatingKey", event.RatingKey, "item-12345")
	checkStringPtrEqual(t, "Guid", event.Guid, "imdb://tt1375666")
	checkIntPtrEqual(t, "VideoHeight", event.VideoHeight, 1080)
	checkIntPtrEqual(t, "VideoWidth", event.VideoWidth, 1920)
	checkStringPtrEqual(t, "VideoFullResolution", event.VideoFullResolution, "1080p")
	checkStringPtrEqual(t, "State", event.State, "playing")

	// Percent complete: (1 hour / 2h28m) * 100 = ~40%
	checkIntEqual(t, "PercentComplete", event.PercentComplete, 40)
}

func TestSessionToPlaybackEventTVShow(t *testing.T) {
	session := &models.JellyfinSession{
		ID:             "session-456",
		UserName:       "TVFan",
		DeviceID:       "device-123",
		DeviceName:     "Bedroom TV",
		Client:         "Jellyfin Android",
		RemoteEndPoint: "10.0.0.100",
		NowPlayingItem: &models.JellyfinNowPlayingItem{
			ID:                "episode-789",
			Name:              "Pilot",
			Type:              "Episode",
			MediaType:         "Video",
			SeriesID:          "series-123",
			SeriesName:        "Breaking Bad",
			SeasonID:          "season-1",
			SeasonName:        "Season 1",
			IndexNumber:       1,
			ParentIndexNumber: 1,
			RunTimeTicks:      35400000000, // 59 minutes
			MediaStreams: []models.JellyfinMediaStream{
				{Type: "Video", Codec: "hevc", Width: 3840, Height: 2160},
			},
		},
		PlayState: &models.JellyfinPlayState{
			PositionTicks: 17700000000, // ~29.5 minutes
			IsPaused:      true,
			PlayMethod:    "Transcode",
		},
		TranscodingInfo: &models.JellyfinTranscodingInfo{
			VideoCodec:    "h264",
			AudioCodec:    "aac",
			AudioChannels: 2,
			Width:         1920,
			Height:        1080,
		},
	}

	event := SessionToPlaybackEvent(session, "server-1")

	checkTrue(t, "event should not be nil", event != nil)
	checkStringEqual(t, "MediaType", event.MediaType, "episode")
	checkStringEqual(t, "Title", event.Title, "Pilot")
	checkStringPtrEqual(t, "GrandparentTitle", event.GrandparentTitle, "Breaking Bad")
	checkStringPtrEqual(t, "ParentTitle", event.ParentTitle, "Season 1")
	checkIntPtrEqual(t, "MediaIndex", event.MediaIndex, 1)
	checkIntPtrEqual(t, "ParentMediaIndex", event.ParentMediaIndex, 1)
	checkStringPtrEqual(t, "TranscodeDecision", event.TranscodeDecision, "transcode")
	checkStringPtrEqual(t, "State", event.State, "paused")
	checkStringPtrEqual(t, "VideoCodec", event.VideoCodec, "h264")
	checkStringPtrEqual(t, "AudioCodec", event.AudioCodec, "aac")
	checkIntPtrEqual(t, "VideoWidth", event.VideoWidth, 3840)
	checkIntPtrEqual(t, "VideoHeight", event.VideoHeight, 2160)

	// Percent: 29.5/59 * 100 = 50%
	checkIntEqual(t, "PercentComplete", event.PercentComplete, 50)
}

func TestSessionToPlaybackEventMusic(t *testing.T) {
	session := &models.JellyfinSession{
		ID:             "session-789",
		UserName:       "MusicLover",
		DeviceID:       "device-audio",
		DeviceName:     "Sonos Speaker",
		Client:         "Jellyfin Music",
		RemoteEndPoint: "[::1]:8096",
		NowPlayingItem: &models.JellyfinNowPlayingItem{
			ID:           "track-123",
			Name:         "Bohemian Rhapsody",
			Type:         "Audio",
			MediaType:    "Audio",
			Album:        "A Night at the Opera",
			AlbumArtist:  "Queen",
			Artists:      []string{"Queen"},
			RunTimeTicks: 3540000000, // 5:54
			MediaStreams: []models.JellyfinMediaStream{
				{Type: "Video", Height: 720, Width: 1280}, // Some audio files have embedded video
			},
		},
		PlayState: &models.JellyfinPlayState{
			PositionTicks: 1770000000, // ~3 minutes
			IsPaused:      false,
			PlayMethod:    "DirectStream",
		},
	}

	event := SessionToPlaybackEvent(session, "server-1")

	checkTrue(t, "event should not be nil", event != nil)
	checkStringEqual(t, "MediaType", event.MediaType, "track")
	checkStringEqual(t, "Title", event.Title, "Bohemian Rhapsody")
	checkStringPtrEqual(t, "GrandparentTitle", event.GrandparentTitle, "Queen")      // AlbumArtist
	checkStringPtrEqual(t, "ParentTitle", event.ParentTitle, "A Night at the Opera") // Album
	checkStringPtrEqual(t, "TranscodeDecision", event.TranscodeDecision, "direct stream")
}

func TestSessionToPlaybackEventResolutionMapping(t *testing.T) {
	tests := []struct {
		name      string
		height    int
		wantResol string
	}{
		{"4K", 2160, "4K"},
		{"1080p", 1080, "1080p"},
		{"720p", 720, "720p"},
		{"SD", 480, "SD"},
		{"SD low", 360, "SD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &models.JellyfinSession{
				ID:         "session-test",
				DeviceID:   "device-test",
				DeviceName: "Test",
				NowPlayingItem: &models.JellyfinNowPlayingItem{
					ID:           "item-test",
					Name:         "Test",
					Type:         "Movie",
					RunTimeTicks: 1000000000,
					MediaStreams: []models.JellyfinMediaStream{
						{Type: "Video", Height: tt.height, Width: tt.height * 16 / 9},
					},
				},
				PlayState: &models.JellyfinPlayState{
					PlayMethod: "DirectPlay",
				},
			}

			event := SessionToPlaybackEvent(session, "server-1")
			checkStringPtrEqual(t, "VideoFullResolution", event.VideoFullResolution, tt.wantResol)
		})
	}
}

func TestSessionToPlaybackEventIPAddressParsing(t *testing.T) {
	tests := []struct {
		name           string
		remoteEndPoint string
		wantIP         string
	}{
		{"IPv4 with port", "192.168.1.100:52345", "192.168.1.100"},
		{"IPv4 without port", "192.168.1.100", "192.168.1.100"},
		{"IPv6 with port", "[::1]:8096", "::1"},
		{"IPv6 without port", "[2001:db8::1]", "2001:db8::1"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &models.JellyfinSession{
				ID:             "session-test",
				DeviceID:       "device-test",
				DeviceName:     "Test",
				RemoteEndPoint: tt.remoteEndPoint,
				NowPlayingItem: &models.JellyfinNowPlayingItem{
					ID:           "item-test",
					Name:         "Test",
					Type:         "Movie",
					RunTimeTicks: 1000000000,
				},
				PlayState: &models.JellyfinPlayState{
					PlayMethod: "DirectPlay",
				},
			}

			event := SessionToPlaybackEvent(session, "server-1")
			checkStringEqual(t, "IPAddress", event.IPAddress, tt.wantIP)
		})
	}
}

func TestSessionToPlaybackEventGuidFromTMDB(t *testing.T) {
	session := &models.JellyfinSession{
		ID:         "session-test",
		DeviceID:   "device-test",
		DeviceName: "Test",
		NowPlayingItem: &models.JellyfinNowPlayingItem{
			ID:           "item-test",
			Name:         "Test Movie",
			Type:         "Movie",
			RunTimeTicks: 1000000000,
			ProviderIDs: map[string]string{
				"Tmdb": "12345", // No IMDB, should fall back to TMDB
			},
		},
		PlayState: &models.JellyfinPlayState{
			PlayMethod: "DirectPlay",
		},
	}

	event := SessionToPlaybackEvent(session, "server-1")
	checkStringPtrEqual(t, "Guid", event.Guid, "tmdb://12345")
}

// ============================================================================
// Jellyfin Session Helper Method Tests
// ============================================================================

func TestJellyfinSessionIsPlaying(t *testing.T) {
	tests := []struct {
		name    string
		session *models.JellyfinSession
		want    bool
	}{
		{
			name: "playing",
			session: &models.JellyfinSession{
				NowPlayingItem: &models.JellyfinNowPlayingItem{Name: "Test"},
				PlayState:      &models.JellyfinPlayState{IsPaused: false},
			},
			want: true,
		},
		{
			name: "paused",
			session: &models.JellyfinSession{
				NowPlayingItem: &models.JellyfinNowPlayingItem{Name: "Test"},
				PlayState:      &models.JellyfinPlayState{IsPaused: true},
			},
			want: false,
		},
		{
			name: "no playstate",
			session: &models.JellyfinSession{
				NowPlayingItem: &models.JellyfinNowPlayingItem{Name: "Test"},
			},
			want: false,
		},
		{
			name:    "idle session",
			session: &models.JellyfinSession{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.session.IsPlaying()
			if got != tt.want {
				t.Errorf("IsPlaying() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJellyfinSessionGetTranscodeDecision(t *testing.T) {
	tests := []struct {
		name       string
		playMethod string
		want       string
	}{
		{"DirectPlay", "DirectPlay", "direct play"},
		{"DirectStream", "DirectStream", "direct stream"},
		{"Transcode", "Transcode", "transcode"},
		{"unknown", "Custom", "Custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &models.JellyfinSession{
				PlayState: &models.JellyfinPlayState{
					PlayMethod: tt.playMethod,
				},
			}
			got := session.GetTranscodeDecision()
			checkStringEqual(t, "GetTranscodeDecision", got, tt.want)
		})
	}
}

func TestJellyfinSessionGetPercentComplete(t *testing.T) {
	tests := []struct {
		name          string
		positionTicks int64
		runTimeTicks  int64
		wantPercent   int
	}{
		{"50%", 50000000000, 100000000000, 50},
		{"0% no progress", 0, 100000000000, 0},
		{"0% no duration", 50000000000, 0, 0},
		{"100%", 100000000000, 100000000000, 100},
		{"33%", 33000000000, 100000000000, 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &models.JellyfinSession{
				NowPlayingItem: &models.JellyfinNowPlayingItem{
					RunTimeTicks: tt.runTimeTicks,
				},
				PlayState: &models.JellyfinPlayState{
					PositionTicks: tt.positionTicks,
				},
			}
			got := session.GetPercentComplete()
			checkIntEqual(t, "GetPercentComplete", got, tt.wantPercent)
		})
	}
}

func TestJellyfinNowPlayingItemGetMediaType(t *testing.T) {
	tests := []struct {
		itemType string
		want     string
	}{
		{"Movie", "movie"},
		{"Episode", "episode"},
		{"Audio", "track"},
		{"MusicVideo", "track"},
		{"Unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.itemType, func(t *testing.T) {
			item := &models.JellyfinNowPlayingItem{Type: tt.itemType}
			got := item.GetMediaType()
			checkStringEqual(t, "GetMediaType", got, tt.want)
		})
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func verifyJellyfinHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	checkStringEqual(t, "X-Emby-Token header", r.Header.Get("X-Emby-Token"), "test-api-key")
	checkStringEqual(t, "X-Emby-Client header", r.Header.Get("X-Emby-Client"), "Cartographus")
	checkStringEqual(t, "Accept header", r.Header.Get("Accept"), "application/json")
}
