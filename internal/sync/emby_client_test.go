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
// Emby Client Constructor Tests
// ============================================================================

func TestNewEmbyClient(t *testing.T) {
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
			baseURL: "https://emby.example.com/",
			apiKey:  "test-api-key",
			userID:  "user-123",
			wantURL: "https://emby.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewEmbyClient(tt.baseURL, tt.apiKey, tt.userID)
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

func TestEmbyClientGetSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/Sessions")
		checkStringEqual(t, "method", r.Method, "GET")
		verifyEmbyHeaders(t, r)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(embySessionsResponse))
	}))
	defer server.Close()

	client := NewEmbyClient(server.URL, "test-api-key", "")
	sessions, err := client.GetSessions(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "sessions", len(sessions), 2)

	// Verify first session (active playback)
	session := sessions[0]
	checkStringEqual(t, "session.ID", session.ID, "emby-session-123")
	checkStringEqual(t, "session.UserName", session.UserName, "EmbyUser")
	checkStringEqual(t, "session.DeviceName", session.DeviceName, "Living Room Emby")
	checkStringEqual(t, "session.Client", session.Client, "Emby Web")
	checkTrue(t, "NowPlayingItem not nil", session.NowPlayingItem != nil)
	checkStringEqual(t, "NowPlayingItem.Name", session.NowPlayingItem.Name, "The Matrix")
	checkStringEqual(t, "NowPlayingItem.Type", session.NowPlayingItem.Type, "Movie")

	// Verify second session (idle)
	idleSession := sessions[1]
	checkStringEqual(t, "idleSession.ID", idleSession.ID, "emby-session-456")
	checkNil(t, "idleSession.NowPlayingItem", idleSession.NowPlayingItem == nil)
}

const embySessionsResponse = `[
	{
		"Id": "emby-session-123",
		"UserId": "user-emby-abc",
		"UserName": "EmbyUser",
		"Client": "Emby Web",
		"DeviceId": "device-emby-xyz",
		"DeviceName": "Living Room Emby",
		"DeviceType": "Browser",
		"ApplicationVersion": "4.7.0",
		"RemoteEndPoint": "192.168.1.200:54321",
		"LastActivityDate": "2024-01-15T10:30:00.0000000Z",
		"NowPlayingItem": {
			"Id": "emby-item-12345",
			"Name": "The Matrix",
			"Type": "Movie",
			"MediaType": "Video",
			"ProductionYear": 1999,
			"RunTimeTicks": 81600000000,
			"ProviderIds": {
				"Imdb": "tt0133093",
				"Tmdb": "603"
			},
			"MediaStreams": [
				{
					"Type": "Video",
					"Codec": "hevc",
					"Width": 3840,
					"Height": 2160,
					"Index": 0
				},
				{
					"Type": "Audio",
					"Codec": "truehd",
					"Channels": 8,
					"Index": 1
				}
			]
		},
		"PlayState": {
			"PositionTicks": 40800000000,
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
		"Id": "emby-session-456",
		"UserId": "user-emby-def",
		"UserName": "AnotherEmbyUser",
		"Client": "Emby Mobile",
		"DeviceId": "device-emby-mobile",
		"DeviceName": "Android Phone",
		"RemoteEndPoint": "10.0.0.100",
		"LastActivityDate": "2024-01-15T09:00:00.0000000Z"
	}
]`

func TestEmbyClientGetSessionsError(t *testing.T) {
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

			client := NewEmbyClient(server.URL, "test-api-key", "")
			_, err := client.GetSessions(context.Background())

			checkError(t, err)
		})
	}
}

func TestEmbyClientGetSessionsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewEmbyClient(server.URL, "test-api-key", "")
	_, err := client.GetSessions(context.Background())

	checkError(t, err)
	checkErrorContains(t, err, "failed to decode")
}

func TestEmbyClientGetSessionsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewEmbyClient(server.URL, "test-api-key", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetSessions(ctx)
	checkError(t, err)
}

// ============================================================================
// GetActiveSessions Tests
// ============================================================================

func TestEmbyClientGetActiveSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(embySessionsResponse))
	}))
	defer server.Close()

	client := NewEmbyClient(server.URL, "test-api-key", "")
	sessions, err := client.GetActiveSessions(context.Background())

	checkNoError(t, err)
	// Only one session has NowPlayingItem
	checkSliceLen(t, "active sessions", len(sessions), 1)
	checkStringEqual(t, "active session ID", sessions[0].ID, "emby-session-123")
}

func TestEmbyClientGetActiveSessionsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewEmbyClient(server.URL, "test-api-key", "")
	sessions, err := client.GetActiveSessions(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "active sessions", len(sessions), 0)
}

// ============================================================================
// GetSystemInfo Tests
// ============================================================================

func TestEmbyClientGetSystemInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/System/Info")
		verifyEmbyHeaders(t, r)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(embySystemInfoResponse))
	}))
	defer server.Close()

	client := NewEmbyClient(server.URL, "test-api-key", "")
	info, err := client.GetSystemInfo(context.Background())

	checkNoError(t, err)
	checkStringEqual(t, "ServerName", info.ServerName, "My Emby Server")
	checkStringEqual(t, "Version", info.Version, "4.7.14.0")
	checkStringEqual(t, "ID", info.ID, "emby-server-id-12345")
	checkStringEqual(t, "OperatingSystem", info.OperatingSystem, "Linux")
	checkTrue(t, "HasUpdateAvailable should be true", info.HasUpdateAvailable)
}

const embySystemInfoResponse = `{
	"ServerName": "My Emby Server",
	"Version": "4.7.14.0",
	"Id": "emby-server-id-12345",
	"OperatingSystem": "Linux",
	"HasUpdateAvailable": true
}`

func TestEmbyClientGetSystemInfoError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Access denied"))
	}))
	defer server.Close()

	client := NewEmbyClient(server.URL, "test-api-key", "")
	_, err := client.GetSystemInfo(context.Background())

	checkError(t, err)
	checkErrorContains(t, err, "403")
}

// ============================================================================
// GetUsers Tests
// ============================================================================

func TestEmbyClientGetUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/Users")
		verifyEmbyHeaders(t, r)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(embyUsersResponse))
	}))
	defer server.Close()

	client := NewEmbyClient(server.URL, "test-api-key", "")
	users, err := client.GetUsers(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "users", len(users), 2)
	checkStringEqual(t, "user[0].ID", users[0].ID, "emby-user-abc-123")
	checkStringEqual(t, "user[0].Name", users[0].Name, "Admin")
	checkStringEqual(t, "user[1].ID", users[1].ID, "emby-user-def-456")
	checkStringEqual(t, "user[1].Name", users[1].Name, "Guest")
}

const embyUsersResponse = `[
	{"Id": "emby-user-abc-123", "Name": "Admin"},
	{"Id": "emby-user-def-456", "Name": "Guest"}
]`

func TestEmbyClientGetUsersError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewEmbyClient(server.URL, "test-api-key", "")
	_, err := client.GetUsers(context.Background())

	checkError(t, err)
}

// ============================================================================
// Ping Tests
// ============================================================================

func TestEmbyClientPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/System/Ping")
		verifyEmbyHeaders(t, r)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewEmbyClient(server.URL, "test-api-key", "")
	err := client.Ping(context.Background())

	checkNoError(t, err)
}

func TestEmbyClientPingError(t *testing.T) {
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

			client := NewEmbyClient(server.URL, "test-api-key", "")
			err := client.Ping(context.Background())

			checkError(t, err)
		})
	}
}

func TestEmbyClientPingConnectionError(t *testing.T) {
	// Use a URL that won't connect
	client := NewEmbyClient("http://localhost:1", "test-api-key", "")
	err := client.Ping(context.Background())

	checkError(t, err)
}

// ============================================================================
// StopSession Tests
// ============================================================================

func TestEmbyClientStopSession(t *testing.T) {
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
				checkStringEqual(t, "path", r.URL.Path, "/Sessions/emby-session-123/Playing/Stop")
				verifyEmbyHeaders(t, r)
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewEmbyClient(server.URL, "test-api-key", "")
			err := client.StopSession(context.Background(), "emby-session-123")

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

func TestEmbyClientGetWebSocketURL(t *testing.T) {
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
			wantURL: "ws://localhost:8096/embywebsocket?api_key=test-key&deviceId=cartographus",
		},
		{
			name:    "https to wss",
			baseURL: "https://emby.example.com",
			apiKey:  "api-key-123",
			userID:  "",
			wantURL: "wss://emby.example.com/embywebsocket?api_key=api-key-123&deviceId=cartographus",
		},
		{
			name:    "with user ID",
			baseURL: "http://localhost:8096",
			apiKey:  "test-key",
			userID:  "user-abc",
			wantURL: "ws://localhost:8096/embywebsocket?api_key=test-key&deviceId=cartographus-user-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewEmbyClient(tt.baseURL, tt.apiKey, tt.userID)
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
// EmbySessionToPlaybackEvent Tests
// ============================================================================

func TestEmbySessionToPlaybackEventNilSession(t *testing.T) {
	event := EmbySessionToPlaybackEvent(nil, "server-1")
	checkNil(t, "event should be nil for nil session", event == nil)
}

func TestEmbySessionToPlaybackEventNoNowPlayingItem(t *testing.T) {
	session := &models.EmbySession{
		ID:       "emby-session-123",
		UserName: "TestUser",
	}
	event := EmbySessionToPlaybackEvent(session, "server-1")
	checkNil(t, "event should be nil when no NowPlayingItem", event == nil)
}

func TestEmbySessionToPlaybackEventMovie(t *testing.T) {
	session := &models.EmbySession{
		ID:             "emby-session-123",
		UserID:         "user-emby-abc",
		UserName:       "EmbyUser",
		Client:         "Emby Web",
		DeviceID:       "device-emby-xyz",
		DeviceName:     "Living Room TV",
		RemoteEndPoint: "192.168.1.200:54321",
		NowPlayingItem: &models.EmbyNowPlayingItem{
			ID:             "emby-item-12345",
			Name:           "The Matrix",
			Type:           "Movie",
			MediaType:      "Video",
			ProductionYear: 1999,
			RunTimeTicks:   81600000000, // 2h 16m
			ProviderIDs: map[string]string{
				"Imdb": "tt0133093",
				"Tmdb": "603",
			},
			MediaStreams: []models.EmbyMediaStream{
				{Type: "Video", Codec: "hevc", Width: 3840, Height: 2160},
			},
		},
		PlayState: &models.EmbyPlayState{
			PositionTicks: 40800000000, // 1h 8m
			IsPaused:      false,
			PlayMethod:    "DirectPlay",
		},
	}

	event := EmbySessionToPlaybackEvent(session, "server-1")

	checkTrue(t, "event should not be nil", event != nil)
	checkStringEqual(t, "Source", event.Source, "emby")
	checkStringEqual(t, "SessionKey", event.SessionKey, "emby-session-123")
	checkStringEqual(t, "Username", event.Username, "EmbyUser")
	checkStringEqual(t, "Platform", event.Platform, "Emby Web")
	checkStringEqual(t, "Player", event.Player, "Living Room TV")
	checkStringEqual(t, "MediaType", event.MediaType, "movie")
	checkStringEqual(t, "Title", event.Title, "The Matrix")
	checkIntPtrEqual(t, "Year", event.Year, 1999)
	checkStringEqual(t, "IPAddress", event.IPAddress, "192.168.1.200")
	checkStringPtrEqual(t, "TranscodeDecision", event.TranscodeDecision, "direct play")
	checkStringPtrEqual(t, "RatingKey", event.RatingKey, "emby-item-12345")
	checkStringPtrEqual(t, "Guid", event.GUID, "imdb://tt0133093")
	checkIntPtrEqual(t, "VideoHeight", event.VideoHeight, 2160)
	checkIntPtrEqual(t, "VideoWidth", event.VideoWidth, 3840)
	checkStringPtrEqual(t, "VideoFullResolution", event.VideoFullResolution, "4K")
	checkStringPtrEqual(t, "State", event.State, "playing")

	// Percent complete: (1h8m / 2h16m) * 100 = 50%
	checkIntEqual(t, "PercentComplete", event.PercentComplete, 50)
}

func TestEmbySessionToPlaybackEventTVShow(t *testing.T) {
	session := &models.EmbySession{
		ID:             "emby-session-456",
		UserName:       "TVFan",
		DeviceID:       "device-123",
		DeviceName:     "Bedroom TV",
		Client:         "Emby Android",
		RemoteEndPoint: "10.0.0.100",
		NowPlayingItem: &models.EmbyNowPlayingItem{
			ID:                "episode-789",
			Name:              "Ozymandias",
			Type:              "Episode",
			MediaType:         "Video",
			SeriesID:          "series-bb",
			SeriesName:        "Breaking Bad",
			SeasonID:          "season-5",
			SeasonName:        "Season 5",
			IndexNumber:       14,
			ParentIndexNumber: 5,
			RunTimeTicks:      28800000000, // 48 minutes
			MediaStreams: []models.EmbyMediaStream{
				{Type: "Video", Codec: "h264", Width: 1920, Height: 1080},
			},
		},
		PlayState: &models.EmbyPlayState{
			PositionTicks: 14400000000, // 24 minutes
			IsPaused:      true,
			PlayMethod:    "Transcode",
		},
		TranscodingInfo: &models.EmbyTranscodingInfo{
			VideoCodec:    "h264",
			AudioCodec:    "aac",
			AudioChannels: 2,
			Width:         1280,
			Height:        720,
		},
	}

	event := EmbySessionToPlaybackEvent(session, "server-1")

	checkTrue(t, "event should not be nil", event != nil)
	checkStringEqual(t, "MediaType", event.MediaType, "episode")
	checkStringEqual(t, "Title", event.Title, "Ozymandias")
	checkStringPtrEqual(t, "GrandparentTitle", event.GrandparentTitle, "Breaking Bad")
	checkStringPtrEqual(t, "ParentTitle", event.ParentTitle, "Season 5")
	checkIntPtrEqual(t, "MediaIndex", event.MediaIndex, 14)
	checkIntPtrEqual(t, "ParentMediaIndex", event.ParentMediaIndex, 5)
	checkStringPtrEqual(t, "TranscodeDecision", event.TranscodeDecision, "transcode")
	checkStringPtrEqual(t, "State", event.State, "paused")
	checkStringPtrEqual(t, "VideoCodec", event.VideoCodec, "h264")
	checkStringPtrEqual(t, "AudioCodec", event.AudioCodec, "aac")
	checkIntPtrEqual(t, "VideoWidth", event.VideoWidth, 1920)
	checkIntPtrEqual(t, "VideoHeight", event.VideoHeight, 1080)

	// Percent: 24/48 * 100 = 50%
	checkIntEqual(t, "PercentComplete", event.PercentComplete, 50)
}

func TestEmbySessionToPlaybackEventMusic(t *testing.T) {
	session := &models.EmbySession{
		ID:             "emby-session-789",
		UserName:       "MusicLover",
		DeviceID:       "device-audio",
		DeviceName:     "HomePod",
		Client:         "Emby Music",
		RemoteEndPoint: "[::1]:8096",
		NowPlayingItem: &models.EmbyNowPlayingItem{
			ID:           "track-123",
			Name:         "Hotel California",
			Type:         "Audio",
			MediaType:    "Audio",
			Album:        "Hotel California",
			AlbumArtist:  "Eagles",
			Artists:      []string{"Eagles"},
			RunTimeTicks: 3910000000, // 6:31
			MediaStreams: []models.EmbyMediaStream{
				{Type: "Video", Height: 720, Width: 1280}, // Album art video
			},
		},
		PlayState: &models.EmbyPlayState{
			PositionTicks: 1955000000, // ~3:15
			IsPaused:      false,
			PlayMethod:    "DirectStream",
		},
	}

	event := EmbySessionToPlaybackEvent(session, "server-1")

	checkTrue(t, "event should not be nil", event != nil)
	checkStringEqual(t, "MediaType", event.MediaType, "track")
	checkStringEqual(t, "Title", event.Title, "Hotel California")
	checkStringPtrEqual(t, "GrandparentTitle", event.GrandparentTitle, "Eagles") // AlbumArtist
	checkStringPtrEqual(t, "ParentTitle", event.ParentTitle, "Hotel California") // Album
	checkStringPtrEqual(t, "TranscodeDecision", event.TranscodeDecision, "direct stream")
}

func TestEmbySessionToPlaybackEventResolutionMapping(t *testing.T) {
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
			session := &models.EmbySession{
				ID:         "emby-session-test",
				DeviceID:   "device-test",
				DeviceName: "Test",
				NowPlayingItem: &models.EmbyNowPlayingItem{
					ID:           "item-test",
					Name:         "Test",
					Type:         "Movie",
					RunTimeTicks: 1000000000,
					MediaStreams: []models.EmbyMediaStream{
						{Type: "Video", Height: tt.height, Width: tt.height * 16 / 9},
					},
				},
				PlayState: &models.EmbyPlayState{
					PlayMethod: "DirectPlay",
				},
			}

			event := EmbySessionToPlaybackEvent(session, "server-1")
			checkStringPtrEqual(t, "VideoFullResolution", event.VideoFullResolution, tt.wantResol)
		})
	}
}

func TestEmbySessionToPlaybackEventIPAddressParsing(t *testing.T) {
	tests := []struct {
		name           string
		remoteEndPoint string
		wantIP         string
	}{
		{"IPv4 with port", "192.168.1.200:54321", "192.168.1.200"},
		{"IPv4 without port", "192.168.1.200", "192.168.1.200"},
		{"IPv6 with port", "[::1]:8096", "::1"},
		{"IPv6 without port", "[2001:db8::1]", "2001:db8::1"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &models.EmbySession{
				ID:             "emby-session-test",
				DeviceID:       "device-test",
				DeviceName:     "Test",
				RemoteEndPoint: tt.remoteEndPoint,
				NowPlayingItem: &models.EmbyNowPlayingItem{
					ID:           "item-test",
					Name:         "Test",
					Type:         "Movie",
					RunTimeTicks: 1000000000,
				},
				PlayState: &models.EmbyPlayState{
					PlayMethod: "DirectPlay",
				},
			}

			event := EmbySessionToPlaybackEvent(session, "server-1")
			checkStringEqual(t, "IPAddress", event.IPAddress, tt.wantIP)
		})
	}
}

func TestEmbySessionToPlaybackEventGuidFromTMDB(t *testing.T) {
	session := &models.EmbySession{
		ID:         "emby-session-test",
		DeviceID:   "device-test",
		DeviceName: "Test",
		NowPlayingItem: &models.EmbyNowPlayingItem{
			ID:           "item-test",
			Name:         "Test Movie",
			Type:         "Movie",
			RunTimeTicks: 1000000000,
			ProviderIDs: map[string]string{
				"Tmdb": "12345", // No IMDB, should fall back to TMDB
			},
		},
		PlayState: &models.EmbyPlayState{
			PlayMethod: "DirectPlay",
		},
	}

	event := EmbySessionToPlaybackEvent(session, "server-1")
	checkStringPtrEqual(t, "Guid", event.GUID, "tmdb://12345")
}

// ============================================================================
// Emby Session Helper Method Tests
// ============================================================================

func TestEmbySessionIsPlaying(t *testing.T) {
	tests := []struct {
		name    string
		session *models.EmbySession
		want    bool
	}{
		{
			name: "playing",
			session: &models.EmbySession{
				NowPlayingItem: &models.EmbyNowPlayingItem{Name: "Test"},
				PlayState:      &models.EmbyPlayState{IsPaused: false},
			},
			want: true,
		},
		{
			name: "paused",
			session: &models.EmbySession{
				NowPlayingItem: &models.EmbyNowPlayingItem{Name: "Test"},
				PlayState:      &models.EmbyPlayState{IsPaused: true},
			},
			want: false,
		},
		{
			name: "no playstate",
			session: &models.EmbySession{
				NowPlayingItem: &models.EmbyNowPlayingItem{Name: "Test"},
			},
			want: false,
		},
		{
			name:    "idle session",
			session: &models.EmbySession{},
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

func TestEmbySessionGetTranscodeDecision(t *testing.T) {
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
			session := &models.EmbySession{
				PlayState: &models.EmbyPlayState{
					PlayMethod: tt.playMethod,
				},
			}
			got := session.GetTranscodeDecision()
			checkStringEqual(t, "GetTranscodeDecision", got, tt.want)
		})
	}
}

func TestEmbySessionGetPercentComplete(t *testing.T) {
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
			session := &models.EmbySession{
				NowPlayingItem: &models.EmbyNowPlayingItem{
					RunTimeTicks: tt.runTimeTicks,
				},
				PlayState: &models.EmbyPlayState{
					PositionTicks: tt.positionTicks,
				},
			}
			got := session.GetPercentComplete()
			checkIntEqual(t, "GetPercentComplete", got, tt.wantPercent)
		})
	}
}

func TestEmbyNowPlayingItemGetMediaType(t *testing.T) {
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
			item := &models.EmbyNowPlayingItem{Type: tt.itemType}
			got := item.GetMediaType()
			checkStringEqual(t, "GetMediaType", got, tt.want)
		})
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func verifyEmbyHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	checkStringEqual(t, "X-Emby-Token header", r.Header.Get("X-Emby-Token"), "test-api-key")
	checkStringEqual(t, "X-Emby-Client header", r.Header.Get("X-Emby-Client"), "Cartographus")
	checkStringEqual(t, "Accept header", r.Header.Get("Accept"), "application/json")
}
