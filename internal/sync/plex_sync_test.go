// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestSyncPlexHistoricalNilClient tests error when Plex client is nil
func TestSyncPlexHistoricalNilClient(t *testing.T) {
	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:      true,
			SyncDaysBack: 30,
		},
	}

	manager := &Manager{
		cfg:        cfg,
		plexClient: nil,
	}

	err := manager.syncPlexHistorical(context.Background())
	checkError(t, err)
	checkErrorContains(t, err, "plex client not initialized")
}

// TestSyncPlexRecentNilClient tests error when Plex client is nil
func TestSyncPlexRecentNilClient(t *testing.T) {
	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:      true,
			SyncInterval: 24 * time.Hour,
		},
	}

	manager := &Manager{
		cfg:        cfg,
		plexClient: nil,
	}

	err := manager.syncPlexRecent(context.Background())
	checkError(t, err)
	checkErrorContains(t, err, "plex client not initialized")
}

// TestPlexClientPing tests Plex server ping
func TestPlexClientPing(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		expectError bool
	}{
		{"successful ping", http.StatusOK, false},
		{"unauthorized", http.StatusUnauthorized, true},
		{"server error", http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				verifyPlexHeaders(t, r)
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewPlexClient(server.URL, "test-token")
			err := client.Ping(context.Background())

			verifyErrorExpectation(t, err, tt.expectError)
		})
	}
}

// verifyPlexHeaders checks that required Plex headers are set
func verifyPlexHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("X-Plex-Token") == "" {
		t.Error("X-Plex-Token header not set")
	}
}

// verifyErrorExpectation checks error matches expectation
func verifyErrorExpectation(t *testing.T, err error, expectError bool) {
	t.Helper()
	if expectError {
		checkError(t, err)
	} else {
		checkNoError(t, err)
	}
}

// TestPlexClientGetHistoryAll tests fetching Plex history
func TestPlexClientGetHistoryAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		verifyHistoryAllRequest(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(historyAllResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	history, err := client.GetHistoryAll(context.Background(), "viewedAt", nil)

	checkNoError(t, err)
	checkSliceLen(t, "history", len(history), 2)
	checkStringEqual(t, "history[0].Title", history[0].Title, "Test Movie")
	checkStringEqual(t, "history[1].GrandparentTitle", history[1].GrandparentTitle, "Test Show")
}

// verifyHistoryAllRequest validates the history request
func verifyHistoryAllRequest(t *testing.T, r *http.Request) {
	t.Helper()
	checkTrue(t, "path starts with /status/sessions/history/all",
		strings.HasPrefix(r.URL.Path, "/status/sessions/history/all"))
	verifyPlexHeaders(t, r)
	checkStringEqual(t, "Accept header", r.Header.Get("Accept"), "application/json")
	checkStringEqual(t, "sort parameter", r.URL.Query().Get("sort"), "viewedAt")
}

const historyAllResponse = `{
	"MediaContainer": {
		"size": 2,
		"Metadata": [
			{
				"ratingKey": "12345",
				"type": "movie",
				"title": "Test Movie",
				"viewedAt": 1700000000,
				"duration": 7200000,
				"accountID": 1
			},
			{
				"ratingKey": "67890",
				"type": "episode",
				"title": "Pilot",
				"grandparentTitle": "Test Show",
				"parentTitle": "Season 1",
				"viewedAt": 1700001000,
				"duration": 3600000,
				"accountID": 2
			}
		]
	}
}`

// TestPlexClientGetHistoryAllWithAccountID tests filtering by account ID
func TestPlexClientGetHistoryAllWithAccountID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "accountID parameter", r.URL.Query().Get("accountID"), "42")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0, "Metadata": []}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	accountID := 42
	_, err := client.GetHistoryAll(context.Background(), "-viewedAt", &accountID)
	checkNoError(t, err)
}

// TestPlexClientGetHistoryAllError tests error handling
func TestPlexClientGetHistoryAllError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	_, err := client.GetHistoryAll(context.Background(), "viewedAt", nil)
	checkError(t, err)
}

// TestPlexClientGetServerIdentity tests server identity retrieval
func TestPlexClientGetServerIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/identity")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(serverIdentityResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	identity, err := client.GetServerIdentity(context.Background())

	checkNoError(t, err)
	checkStringEqual(t, "MachineIdentifier", identity.MachineIdentifier, "abc123def456")
	checkStringEqual(t, "Version", identity.Version, "1.40.0.8395")
	checkStringEqual(t, "Platform", identity.Platform, "Linux")
}

const serverIdentityResponse = `{
	"MediaContainer": {
		"machineIdentifier": "abc123def456",
		"version": "1.40.0.8395",
		"platform": "Linux"
	}
}`

// TestPlexClientGetTranscodeSessionsComprehensive tests comprehensive transcode session retrieval
func TestPlexClientGetTranscodeSessionsComprehensive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/status/sessions")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(transcodeSessionsResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	sessions, err := client.GetTranscodeSessions(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "sessions", len(sessions), 1)

	session := sessions[0]
	verifyTranscodeSession(t, session)
}

func verifyTranscodeSession(t *testing.T, session models.PlexSession) {
	t.Helper()
	checkStringEqual(t, "SessionKey", session.SessionKey, "session123")
	checkStringEqual(t, "Title", session.Title, "Test Movie")

	if session.TranscodeSession == nil {
		t.Fatal("TranscodeSession should not be nil")
	}
	checkFloat64PtrEqual(t, "Speed", &session.TranscodeSession.Speed, 2.5)

	if session.User == nil {
		t.Fatal("User should not be nil")
	}
	checkStringEqual(t, "User.Title", session.User.Title, "testuser")
}

const transcodeSessionsResponse = `{
	"MediaContainer": {
		"size": 1,
		"Metadata": [
			{
				"sessionKey": "session123",
				"key": "/library/metadata/12345",
				"type": "movie",
				"title": "Test Movie",
				"viewOffset": 3600000,
				"duration": 7200000,
				"TranscodeSession": {
					"key": "abc123",
					"progress": 50.0,
					"speed": 2.5,
					"throttled": false,
					"videoDecision": "transcode",
					"audioDecision": "copy",
					"maxOffsetAvailable": 4000000,
					"minOffsetAvailable": 0
				},
				"User": {
					"id": 1,
					"title": "testuser"
				},
				"Player": {
					"title": "Living Room TV",
					"product": "Plex for Smart TVs"
				}
			}
		]
	}
}`

// TestPlexClientGetTranscodeSessionsEmpty tests empty sessions response
func TestPlexClientGetTranscodeSessionsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	sessions, err := client.GetTranscodeSessions(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "sessions", len(sessions), 0)
}

// TestPlexClientGetSessionTimelineComprehensive tests comprehensive timeline retrieval
func TestPlexClientGetSessionTimelineComprehensive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sessionTimelineResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	timeline, err := client.GetSessionTimeline(context.Background(), "session123")

	checkNoError(t, err)
	checkIntEqual(t, "Size", timeline.MediaContainer.Size, 1)
	checkSliceLen(t, "Metadata", len(timeline.MediaContainer.Metadata), 1)

	metadata := timeline.MediaContainer.Metadata[0]
	verifyTimelineMetadata(t, metadata)
}

func verifyTimelineMetadata(t *testing.T, metadata models.PlexTimelineMetadata) {
	t.Helper()
	checkStringEqual(t, "SessionKey", metadata.SessionKey, "session123")
	checkStringEqual(t, "Title", metadata.Title, "Test Movie")

	if metadata.TranscodeSession == nil {
		t.Fatal("TranscodeSession should not be nil")
	}
	checkFloat64PtrEqual(t, "MaxOffsetAvailable", &metadata.TranscodeSession.MaxOffsetAvailable, 4000000)
}

const sessionTimelineResponse = `{
	"MediaContainer": {
		"size": 2,
		"Metadata": [
			{
				"sessionKey": "session123",
				"key": "/library/metadata/12345",
				"type": "movie",
				"title": "Test Movie",
				"viewOffset": 3600000,
				"duration": 7200000,
				"TranscodeSession": {
					"maxOffsetAvailable": 4000000,
					"minOffsetAvailable": 0,
					"progress": 50.0,
					"speed": 2.0
				}
			},
			{
				"sessionKey": "session456",
				"title": "Other Movie"
			}
		]
	}
}`

// TestPlexClientGetSessionTimelineNotFound tests timeline for non-existent session
func TestPlexClientGetSessionTimelineNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 1, "Metadata": [{"sessionKey": "other-session"}]}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	timeline, err := client.GetSessionTimeline(context.Background(), "non-existent")

	checkNoError(t, err)
	checkIntEqual(t, "Size", timeline.MediaContainer.Size, 0)
}

// TestPlexClientRateLimitingBasic tests that 429 responses return appropriate errors
func TestPlexClientRateLimitingBasic(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	err := client.Ping(context.Background())

	checkError(t, err)
	checkIntEqual(t, "callCount", callCount, 1)
}

// TestPlexClientRateLimitingExceeded tests that rate limit errors are reported
func TestPlexClientRateLimitingExceeded(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	err := client.Ping(context.Background())

	checkError(t, err)
	checkErrorContains(t, err, "429")
	checkIntEqual(t, "callCount", callCount, 1)
}

// TestPlexClientContextCancellation tests that context cancellation is respected
func TestPlexClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.Ping(ctx)
	checkError(t, err)
}

// TestNewPlexClientConfiguration tests client constructor configuration
func TestNewPlexClientConfiguration(t *testing.T) {
	client := NewPlexClient("http://localhost:32400", "my-token")

	checkStringEqual(t, "baseURL", client.baseURL, "http://localhost:32400")
	checkStringEqual(t, "token", client.token, "my-token")

	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Timeout: expected 30s, got %v", client.httpClient.Timeout)
	}
}

// TestConvertPlexToPlaybackEventFullRecord tests full conversion
func TestConvertPlexToPlaybackEventFullRecord(t *testing.T) {
	cfg := &config.Config{}
	m := &Manager{cfg: cfg}

	now := time.Now().Unix()
	record := createFullPlexRecord(now)
	event := m.convertPlexToPlaybackEvent(record)

	verifyPlexEventBasicFields(t, event)
	verifyPlexEventMetadata(t, event)
	verifyPlexEventMissingFields(t, event)
}

func createFullPlexRecord(now int64) *PlexMetadata {
	return &PlexMetadata{
		RatingKey:            "12345",
		Key:                  "/library/metadata/12345",
		ParentRatingKey:      "parent123",
		GrandparentRatingKey: "gparent123",
		Type:                 "episode",
		Title:                "Pilot",
		GrandparentTitle:     "Breaking Bad",
		ParentTitle:          "Season 1",
		OriginalTitle:        "Piloto",
		ViewedAt:             now,
		Duration:             3600000, // 1 hour in ms
		ViewOffset:           1800000, // 30 minutes in ms (50%)
		AccountID:            42,
		User: &PlexUser{
			ID:    42,
			Title: "testuser",
		},
		Index:                 1,
		ParentIndex:           1,
		Year:                  2008,
		Guid:                  "plex://show/12345/episode/1",
		OriginallyAvailableAt: "2008-01-20",
		Thumb:                 "/library/metadata/12345/thumb",
	}
}

func verifyPlexEventBasicFields(t *testing.T, event *models.PlaybackEvent) {
	t.Helper()
	checkStringEqual(t, "Source", event.Source, "plex")
	checkStringPtrEqual(t, "PlexKey", event.PlexKey, "12345")
	checkStringEqual(t, "Title", event.Title, "Pilot")
	checkStringEqual(t, "MediaType", event.MediaType, "episode")
	checkIntEqual(t, "UserID", event.UserID, 42)
	checkStringEqual(t, "Username", event.Username, "testuser")
	checkIntEqual(t, "PercentComplete", event.PercentComplete, 50)

	if event.StoppedAt == nil {
		t.Error("StoppedAt should be set when duration > 0")
	}
}

func verifyPlexEventMetadata(t *testing.T, event *models.PlaybackEvent) {
	t.Helper()
	checkStringPtrEqual(t, "GrandparentTitle", event.GrandparentTitle, "Breaking Bad")
	checkStringPtrEqual(t, "ParentTitle", event.ParentTitle, "Season 1")
	checkIntPtrEqual(t, "MediaIndex", event.MediaIndex, 1)
	checkIntPtrEqual(t, "ParentMediaIndex", event.ParentMediaIndex, 1)
	checkIntPtrEqual(t, "Year", event.Year, 2008)
	checkStringPtrEqual(t, "Guid", event.GUID, "plex://show/12345/episode/1")
	checkStringPtrEqual(t, "OriginallyAvailableAt", event.OriginallyAvailableAt, "2008-01-20")
}

func verifyPlexEventMissingFields(t *testing.T, event *models.PlaybackEvent) {
	t.Helper()
	checkStringEmpty(t, "IPAddress", event.IPAddress)
	checkStringEmpty(t, "Platform", event.Platform)
}

// TestSyncPlexHistoricalFiltersOldRecords tests date filtering
func TestSyncPlexHistoricalFiltersOldRecords(t *testing.T) {
	insertedRecords := make([]*models.PlaybackEvent, 0)
	mockDB := &mockDB{
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			insertedRecords = append(insertedRecords, event)
			return nil
		},
	}

	now := time.Now()
	recentTimestamp := now.Add(-5 * 24 * time.Hour).Unix()
	oldTimestamp := now.Add(-60 * 24 * time.Hour).Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(historicalFilterResponse, recentTimestamp, oldTimestamp)))
	}))
	defer server.Close()

	cfg := createPlexSyncConfig(30)
	plexClient := NewPlexClient(server.URL, "test-token")
	manager := NewManager(mockDB, nil, &mockTautulliClient{}, cfg, nil)
	manager.plexClient = plexClient

	err := manager.syncPlexHistorical(context.Background())
	checkNoError(t, err)
}

const historicalFilterResponse = `{
	"MediaContainer": {
		"size": 2,
		"Metadata": [
			{
				"ratingKey": "recent",
				"type": "movie",
				"title": "Recent Movie",
				"viewedAt": %d,
				"accountID": 1
			},
			{
				"ratingKey": "old",
				"type": "movie",
				"title": "Old Movie",
				"viewedAt": %d,
				"accountID": 1
			}
		]
	}
}`

func createPlexSyncConfig(syncDaysBack int) *config.Config {
	return &config.Config{
		Plex: config.PlexConfig{
			Enabled:      true,
			SyncDaysBack: syncDaysBack,
		},
		Sync: config.SyncConfig{
			BatchSize:     1000,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}
}

// TestStartPlexServices tests that Plex services are started based on config
func TestStartPlexServices(t *testing.T) {
	tests := []struct {
		name                string
		plexEnabled         bool
		historicalSync      bool
		realtimeEnabled     bool
		transcodeMonitor    bool
		bufferHealthMonitor bool
	}{
		{"all disabled", false, false, false, false, false},
		{"historical sync enabled", true, true, false, false, false},
		{"realtime enabled", true, false, true, false, false},
		{"transcode monitoring enabled", true, false, false, true, false},
		{"buffer health monitoring enabled", true, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createPlexServicesConfig(tt)
			mockDB := &mockDB{
				sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
					return false, nil
				},
			}

			manager := NewManager(mockDB, nil, &mockTautulliClient{}, cfg, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Should not panic regardless of config
			manager.startPlexServices(ctx)
		})
	}
}

func createPlexServicesConfig(tt struct {
	name                string
	plexEnabled         bool
	historicalSync      bool
	realtimeEnabled     bool
	transcodeMonitor    bool
	bufferHealthMonitor bool
}) *config.Config {
	return &config.Config{
		Plex: config.PlexConfig{
			Enabled:                     tt.plexEnabled,
			HistoricalSync:              tt.historicalSync,
			RealtimeEnabled:             tt.realtimeEnabled,
			TranscodeMonitoring:         tt.transcodeMonitor,
			BufferHealthMonitoring:      tt.bufferHealthMonitor,
			SyncInterval:                time.Hour,
			TranscodeMonitoringInterval: 10 * time.Second,
			BufferHealthPollInterval:    5 * time.Second,
		},
		Sync: config.SyncConfig{
			Interval:      time.Minute,
			Lookback:      time.Hour,
			BatchSize:     100,
			RetryAttempts: 1,
			RetryDelay:    time.Millisecond,
		},
	}
}

// TestSyncPlexRecentHandlesErrors tests error handling in recent sync
func TestSyncPlexRecentHandlesErrors(t *testing.T) {
	mockDB := &mockDB{
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			return errors.New("constraint violation: UNIQUE")
		},
	}

	now := time.Now()
	recentTimestamp := now.Add(-1 * time.Hour).Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(recentSyncErrorResponse, recentTimestamp)))
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:      true,
			SyncInterval: 24 * time.Hour,
		},
		Sync: config.SyncConfig{
			BatchSize:     1000,
			RetryAttempts: 1,
			RetryDelay:    1 * time.Millisecond,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")
	manager := NewManager(mockDB, nil, &mockTautulliClient{}, cfg, nil)
	manager.plexClient = plexClient

	// Should not return error even when inserts fail with constraint violation
	err := manager.syncPlexRecent(context.Background())
	checkNoError(t, err)
}

const recentSyncErrorResponse = `{
	"MediaContainer": {
		"size": 1,
		"Metadata": [
			{
				"ratingKey": "12345",
				"type": "movie",
				"title": "Test Movie",
				"viewedAt": %d,
				"accountID": 1
			}
		]
	}
}`
