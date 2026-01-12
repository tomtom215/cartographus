// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	stdsync "sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"

	"github.com/tomtom215/cartographus/internal/models"
)

// mockPlexWebSocketServer creates a test WebSocket server that simulates Plex
type mockPlexWebSocketServer struct {
	server   *httptest.Server
	upgrader websocket.Upgrader
	connChan chan *websocket.Conn
}

// newMockPlexWebSocketServer creates a new mock Plex WebSocket server
func newMockPlexWebSocketServer() *mockPlexWebSocketServer {
	mock := &mockPlexWebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		connChan: make(chan *websocket.Conn, 1),
	}

	// Create HTTP test server
	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authentication token
		token := r.URL.Query().Get("X-Plex-Token")
		if token != "test-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Upgrade to WebSocket
		conn, err := mock.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		// Send connection to test
		mock.connChan <- conn
	}))

	return mock
}

// close shuts down the mock server
func (m *mockPlexWebSocketServer) close() {
	m.server.Close()
}

// sendNotification sends a notification to connected client
func (m *mockPlexWebSocketServer) sendNotification(conn *websocket.Conn, notif interface{}) error {
	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

// plexWSTestSetup holds common test setup components
type plexWSTestSetup struct {
	mock   *mockPlexWebSocketServer
	client *PlexWebSocketClient
	ctx    context.Context
	cancel context.CancelFunc
}

// setupPlexWSTest creates a mock server and client for testing.
// Returns setup struct. Caller should defer setup.cleanup().
func setupPlexWSTest(t *testing.T) *plexWSTestSetup {
	t.Helper()
	mock := newMockPlexWebSocketServer()
	baseURL := "http" + strings.TrimPrefix(mock.server.URL, "http")
	client := NewPlexWebSocketClient(baseURL, "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	return &plexWSTestSetup{
		mock:   mock,
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}
}

// cleanup closes all resources
func (s *plexWSTestSetup) cleanup() {
	s.cancel()
	s.client.Close()
	s.mock.close()
}

// connectAndGetServerConn connects the client and returns the server-side connection.
func (s *plexWSTestSetup) connectAndGetServerConn(t *testing.T) *websocket.Conn {
	t.Helper()
	if err := s.client.Connect(s.ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	select {
	case conn := <-s.mock.connChan:
		return conn
	case <-time.After(1 * time.Second):
		t.Fatal("Server did not receive connection")
		return nil
	}
}

// TestPlexWebSocketClient_Connect tests successful WebSocket connection
func TestPlexWebSocketClient_Connect(t *testing.T) {
	mock := newMockPlexWebSocketServer()
	defer mock.close()

	// Replace http:// with ws://
	baseURL := "http" + strings.TrimPrefix(mock.server.URL, "http")

	client := NewPlexWebSocketClient(baseURL, "test-token")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	// Verify connection established
	if !client.IsConnected() {
		t.Error("IsConnected() = false, want true")
	}

	// Wait for server to receive connection
	select {
	case conn := <-mock.connChan:
		conn.Close()
	case <-time.After(1 * time.Second):
		t.Error("Server did not receive connection")
	}
}

// TestPlexWebSocketClient_PlayingNotification tests playing state notification handling
func TestPlexWebSocketClient_PlayingNotification(t *testing.T) {
	setup := setupPlexWSTest(t)
	defer setup.cleanup()

	// Track received notifications (protected by mutex for race-free access)
	var receivedCount int32
	var mu stdsync.Mutex
	var receivedNotif models.PlexPlayingNotification

	setup.client.SetCallbacks(
		func(notif models.PlexPlayingNotification) {
			atomic.AddInt32(&receivedCount, 1)
			mu.Lock()
			receivedNotif = notif
			mu.Unlock()
		},
		nil, nil, nil,
	)

	serverConn := setup.connectAndGetServerConn(t)
	defer serverConn.Close()

	// Send playing notification
	notification := models.PlexNotificationWrapper{
		NotificationContainer: models.PlexNotificationContainer{
			Type: "playing",
			Size: 1,
			PlaySessionStateNotification: []models.PlexPlayingNotification{
				{
					SessionKey:       "test-session-123",
					ClientIdentifier: "test-client",
					State:            "playing",
					RatingKey:        "12345",
					ViewOffset:       300000, // 5 minutes
					TranscodeSession: "transcode-abc",
				},
			},
		},
	}

	if err := setup.mock.sendNotification(serverConn, notification); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	// Wait for notification to be received
	time.Sleep(100 * time.Millisecond)

	// Verify notification was received
	if atomic.LoadInt32(&receivedCount) != 1 {
		t.Errorf("Received %d notifications, want 1", atomic.LoadInt32(&receivedCount))
	}

	// Verify notification content (mutex-protected read)
	mu.Lock()
	sessionKey := receivedNotif.SessionKey
	state := receivedNotif.State
	viewOffset := receivedNotif.ViewOffset
	mu.Unlock()

	if sessionKey != "test-session-123" {
		t.Errorf("SessionKey = %s, want test-session-123", sessionKey)
	}
	if state != "playing" {
		t.Errorf("State = %s, want playing", state)
	}
	if viewOffset != 300000 {
		t.Errorf("ViewOffset = %d, want 300000", viewOffset)
	}
}

// TestPlexWebSocketClient_BufferingDetection tests buffering state detection
func TestPlexWebSocketClient_BufferingDetection(t *testing.T) {
	setup := setupPlexWSTest(t)
	defer setup.cleanup()

	// Track buffering notifications (protected by mutex for race-free access)
	var mu stdsync.Mutex
	var bufferingDetected bool

	setup.client.SetCallbacks(
		func(notif models.PlexPlayingNotification) {
			if notif.IsBuffering() {
				mu.Lock()
				bufferingDetected = true
				mu.Unlock()
			}
		},
		nil, nil, nil,
	)

	serverConn := setup.connectAndGetServerConn(t)
	defer serverConn.Close()

	// Send buffering notification
	notification := models.PlexNotificationWrapper{
		NotificationContainer: models.PlexNotificationContainer{
			Type: "playing",
			PlaySessionStateNotification: []models.PlexPlayingNotification{
				{
					SessionKey: "test-session-456",
					State:      "buffering", // CRITICAL: Buffering state
					RatingKey:  "67890",
				},
			},
		},
	}

	if err := setup.mock.sendNotification(serverConn, notification); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify buffering was detected (mutex-protected read)
	mu.Lock()
	detected := bufferingDetected
	mu.Unlock()

	if !detected {
		t.Error("Buffering state not detected")
	}
}

// TestPlexWebSocketClient_MultipleNotifications tests batch notification handling
func TestPlexWebSocketClient_MultipleNotifications(t *testing.T) {
	setup := setupPlexWSTest(t)
	defer setup.cleanup()

	var receivedCount int32

	setup.client.SetCallbacks(
		func(notif models.PlexPlayingNotification) {
			atomic.AddInt32(&receivedCount, 1)
		},
		nil, nil, nil,
	)

	serverConn := setup.connectAndGetServerConn(t)
	defer serverConn.Close()

	// Send notification with multiple sessions
	notification := models.PlexNotificationWrapper{
		NotificationContainer: models.PlexNotificationContainer{
			Type: "playing",
			Size: 3,
			PlaySessionStateNotification: []models.PlexPlayingNotification{
				{SessionKey: "session-1", State: "playing"},
				{SessionKey: "session-2", State: "paused"},
				{SessionKey: "session-3", State: "playing"},
			},
		},
	}

	if err := setup.mock.sendNotification(serverConn, notification); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify all 3 notifications were received
	if count := atomic.LoadInt32(&receivedCount); count != 3 {
		t.Errorf("Received %d notifications, want 3", count)
	}
}

// TestPlexWebSocketClient_TimelineNotification tests timeline event handling
func TestPlexWebSocketClient_TimelineNotification(t *testing.T) {
	setup := setupPlexWSTest(t)
	defer setup.cleanup()

	var mu stdsync.Mutex
	var timelineReceived bool

	setup.client.SetCallbacks(
		nil,
		func(notif models.PlexTimelineNotification) {
			mu.Lock()
			timelineReceived = true
			mu.Unlock()
			if notif.Type != 4 { // episode
				t.Errorf("Type = %d, want 4 (episode)", notif.Type)
			}
			if notif.State != 6 { // analyzing
				t.Errorf("State = %d, want 6 (analyzing)", notif.State)
			}
		},
		nil, nil,
	)

	serverConn := setup.connectAndGetServerConn(t)
	defer serverConn.Close()

	// Send timeline notification
	notification := models.PlexNotificationWrapper{
		NotificationContainer: models.PlexNotificationContainer{
			Type: "timeline",
			TimelineEntry: []models.PlexTimelineNotification{
				{
					Identifier: "com.plexapp.plugins.library",
					ItemID:     123,
					Type:       4, // episode
					State:      6, // analyzing
					Title:      "Test Episode",
					SectionID:  2,
					UpdatedAt:  time.Now().Unix(),
				},
			},
		},
	}

	if err := setup.mock.sendNotification(serverConn, notification); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	received := timelineReceived
	mu.Unlock()

	if !received {
		t.Error("Timeline notification not received")
	}
}

// TestPlexWebSocketClient_ActivityNotification tests background task progress
func TestPlexWebSocketClient_ActivityNotification(t *testing.T) {
	setup := setupPlexWSTest(t)
	defer setup.cleanup()

	var mu stdsync.Mutex
	var activityReceived bool
	var progress int

	setup.client.SetCallbacks(
		nil, nil,
		func(notif models.PlexActivityNotification) {
			mu.Lock()
			activityReceived = true
			progress = notif.GetActivityProgress()
			mu.Unlock()
		},
		nil,
	)

	serverConn := setup.connectAndGetServerConn(t)
	defer serverConn.Close()

	// Send activity notification
	notification := models.PlexNotificationWrapper{
		NotificationContainer: models.PlexNotificationContainer{
			Type: "activity",
			ActivityNotification: []models.PlexActivityNotification{
				{
					Event: "progress",
					UUID:  "scan-123",
					Activity: models.PlexActivityData{
						UUID:     "scan-123",
						Type:     "library.scan",
						Title:    "Scanning Library",
						Subtitle: "Movies",
						Progress: 45, // 45% complete
						Context: models.PlexActivityContext{
							LibrarySectionID:    "1",
							LibrarySectionTitle: "Movies",
							LibrarySectionType:  "movie",
						},
					},
				},
			},
		},
	}

	if err := setup.mock.sendNotification(serverConn, notification); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	received := activityReceived
	progressVal := progress
	mu.Unlock()

	if !received {
		t.Error("Activity notification not received")
	}
	if progressVal != 45 {
		t.Errorf("Progress = %d, want 45", progressVal)
	}
}

// TestPlexWebSocketClient_StatusNotification tests server status changes
func TestPlexWebSocketClient_StatusNotification(t *testing.T) {
	setup := setupPlexWSTest(t)
	defer setup.cleanup()

	var mu stdsync.Mutex
	var statusReceived bool
	var shutdownDetected bool

	setup.client.SetCallbacks(
		nil, nil, nil,
		func(notif models.PlexStatusNotification) {
			mu.Lock()
			statusReceived = true
			if notif.NotificationName == "SERVER_SHUTDOWN" {
				shutdownDetected = true
			}
			mu.Unlock()
		},
	)

	serverConn := setup.connectAndGetServerConn(t)
	defer serverConn.Close()

	// Send server shutdown notification
	notification := models.PlexNotificationWrapper{
		NotificationContainer: models.PlexNotificationContainer{
			Type: "status",
			StatusNotification: []models.PlexStatusNotification{
				{
					Title:            "Server Shutting Down",
					Description:      "Plex Media Server is shutting down for maintenance",
					NotificationName: "SERVER_SHUTDOWN",
				},
			},
		},
	}

	if err := setup.mock.sendNotification(serverConn, notification); err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	received := statusReceived
	shutdown := shutdownDetected
	mu.Unlock()

	if !received {
		t.Error("Status notification not received")
	}
	if !shutdown {
		t.Error("Server shutdown not detected")
	}
}

// TestPlexWebSocketClient_AuthenticationFailure tests invalid token handling
func TestPlexWebSocketClient_AuthenticationFailure(t *testing.T) {
	mock := newMockPlexWebSocketServer()
	defer mock.close()

	baseURL := "http" + strings.TrimPrefix(mock.server.URL, "http")
	client := NewPlexWebSocketClient(baseURL, "invalid-token") // Wrong token

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt connection with invalid token
	err := client.Connect(ctx)
	if err == nil {
		t.Error("Connect() succeeded with invalid token, want error")
	}

	// Verify error contains HTTP 401
	if !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "Unauthorized") {
		t.Errorf("Expected 401 Unauthorized error, got: %v", err)
	}
}

// TestPlexWebSocketClient_Close tests graceful shutdown
func TestPlexWebSocketClient_Close(t *testing.T) {
	// Don't use setupPlexWSTest here since we're testing Close behavior explicitly
	mock := newMockPlexWebSocketServer()
	defer mock.close()

	baseURL := "http" + strings.TrimPrefix(mock.server.URL, "http")
	client := NewPlexWebSocketClient(baseURL, "test-token")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// Verify connected
	if !client.IsConnected() {
		t.Error("IsConnected() = false after Connect()")
	}

	// Close client
	if err := client.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Verify disconnected
	if client.IsConnected() {
		t.Error("IsConnected() = true after Close()")
	}
}

// TestPlexWebSocketClient_ConcurrentCallbacks tests thread safety
func TestPlexWebSocketClient_ConcurrentCallbacks(t *testing.T) {
	setup := setupPlexWSTest(t)
	defer setup.cleanup()

	var receivedCount int32

	// Initial callback
	setup.client.SetCallbacks(
		func(notif models.PlexPlayingNotification) {
			atomic.AddInt32(&receivedCount, 1)
		},
		nil, nil, nil,
	)

	serverConn := setup.connectAndGetServerConn(t)
	defer serverConn.Close()

	// Send notification
	notification := models.PlexNotificationWrapper{
		NotificationContainer: models.PlexNotificationContainer{
			Type: "playing",
			PlaySessionStateNotification: []models.PlexPlayingNotification{
				{SessionKey: "test", State: "playing"},
			},
		},
	}

	// Concurrently update callbacks while sending notifications
	done := make(chan bool)
	go func() {
		for i := 0; i < 10; i++ {
			setup.client.SetCallbacks(
				func(notif models.PlexPlayingNotification) {
					atomic.AddInt32(&receivedCount, 1)
				},
				nil, nil, nil,
			)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Send multiple notifications
	for i := 0; i < 10; i++ {
		if err := setup.mock.sendNotification(serverConn, notification); err != nil {
			t.Errorf("Failed to send notification: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	<-done
	time.Sleep(100 * time.Millisecond)

	// Verify no race conditions (just check > 0)
	if count := atomic.LoadInt32(&receivedCount); count == 0 {
		t.Error("No notifications received")
	}
}

// TestPlexNotificationHelpers tests notification helper methods
func TestPlexNotificationHelpers(t *testing.T) {
	tests := []struct {
		name      string
		notif     models.PlexPlayingNotification
		wantState string
		isBuffer  bool
		isActive  bool
	}{
		{
			name:      "playing state",
			notif:     models.PlexPlayingNotification{State: "playing"},
			wantState: "Playing",
			isBuffer:  false,
			isActive:  true,
		},
		{
			name:      "paused state",
			notif:     models.PlexPlayingNotification{State: "paused"},
			wantState: "Paused",
			isBuffer:  false,
			isActive:  true,
		},
		{
			name:      "buffering state",
			notif:     models.PlexPlayingNotification{State: "buffering"},
			wantState: "Buffering",
			isBuffer:  true,
			isActive:  true,
		},
		{
			name:      "stopped state",
			notif:     models.PlexPlayingNotification{State: "stopped"},
			wantState: "Stopped",
			isBuffer:  false,
			isActive:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.notif.GetPlaybackState(); got != tt.wantState {
				t.Errorf("GetPlaybackState() = %s, want %s", got, tt.wantState)
			}
			if got := tt.notif.IsBuffering(); got != tt.isBuffer {
				t.Errorf("IsBuffering() = %v, want %v", got, tt.isBuffer)
			}
			if got := tt.notif.IsActive(); got != tt.isActive {
				t.Errorf("IsActive() = %v, want %v", got, tt.isActive)
			}
		})
	}
}

// Benchmark tests

// BenchmarkPlexWebSocketClient_MessageHandling benchmarks notification processing
func BenchmarkPlexWebSocketClient_MessageHandling(b *testing.B) {
	notification := models.PlexNotificationWrapper{
		NotificationContainer: models.PlexNotificationContainer{
			Type: "playing",
			PlaySessionStateNotification: []models.PlexPlayingNotification{
				{SessionKey: "test", State: "playing"},
			},
		},
	}

	data, _ := json.Marshal(notification)

	client := NewPlexWebSocketClient("http://localhost:32400", "test-token")
	client.SetCallbacks(
		func(notif models.PlexPlayingNotification) {},
		nil, nil, nil,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.handleMessage(data)
	}
}

// TestPopulateQualityMetrics tests extraction of quality metrics from Plex sessions
func TestPopulateQualityMetrics(t *testing.T) {
	m := &Manager{}

	t.Run("direct_play_session", func(t *testing.T) {
		session := &models.PlexSession{
			SessionKey: "test-session-1",
			Media: []models.PlexMedia{{
				VideoCodec:      "hevc",
				AudioCodec:      "eac3",
				VideoResolution: "4k",
				Bitrate:         25000,
				Width:           3840,
				Height:          2160,
				AudioChannels:   6,
				Container:       "mkv",
				VideoFrameRate:  "24p",
				VideoProfile:    "main 10",
				AudioProfile:    "lc",
			}},
			// No TranscodeSession = direct play
		}

		event := &models.PlaybackEvent{}
		m.populateQualityMetrics(event, session)

		// Verify direct play is detected
		if event.TranscodeDecision == nil || *event.TranscodeDecision != "direct play" {
			t.Errorf("TranscodeDecision = %v, want 'direct play'", event.TranscodeDecision)
		}

		// Verify source media quality
		if event.VideoCodec == nil || *event.VideoCodec != "hevc" {
			t.Errorf("VideoCodec = %v, want 'hevc'", event.VideoCodec)
		}
		if event.VideoResolution == nil || *event.VideoResolution != "4k" {
			t.Errorf("VideoResolution = %v, want '4k'", event.VideoResolution)
		}
		if event.Bitrate == nil || *event.Bitrate != 25000 {
			t.Errorf("Bitrate = %v, want 25000", event.Bitrate)
		}
		if event.VideoWidth == nil || *event.VideoWidth != 3840 {
			t.Errorf("VideoWidth = %v, want 3840", event.VideoWidth)
		}
		if event.AudioChannels == nil || *event.AudioChannels != "6" {
			t.Errorf("AudioChannels = %v, want '6'", event.AudioChannels)
		}
	})

	t.Run("transcode_session", func(t *testing.T) {
		session := &models.PlexSession{
			SessionKey: "test-session-2",
			Media: []models.PlexMedia{{
				VideoCodec:      "hevc",
				AudioCodec:      "truehd",
				VideoResolution: "4k",
				Bitrate:         50000,
			}},
			TranscodeSession: &models.PlexTranscodeSession{
				VideoDecision:        "transcode",
				AudioDecision:        "transcode",
				SourceVideoCodec:     "hevc",
				SourceAudioCodec:     "truehd",
				VideoCodec:           "h264",
				AudioCodec:           "aac",
				Width:                1920,
				Height:               1080,
				AudioChannels:        2,
				Container:            "ts",
				Protocol:             "hls",
				TranscodeHwRequested: true,
				TranscodeHwDecoding:  "videotoolbox",
				TranscodeHwEncoding:  "videotoolbox",
				TranscodeHwFullPipe:  true,
				Progress:             45.5,
				Speed:                2.3,
				Throttled:            false,
				Key:                  "/transcode/sessions/abc123",
			},
		}

		event := &models.PlaybackEvent{}
		m.populateQualityMetrics(event, session)

		// Verify transcode decision
		if event.TranscodeDecision == nil || *event.TranscodeDecision != "transcode" {
			t.Errorf("TranscodeDecision = %v, want 'transcode'", event.TranscodeDecision)
		}

		// Verify source codec is set from TranscodeSession (overrides Media)
		if event.VideoCodec == nil || *event.VideoCodec != "hevc" {
			t.Errorf("VideoCodec = %v, want 'hevc'", event.VideoCodec)
		}

		// Verify transcode output
		if event.TranscodeVideoCodec == nil || *event.TranscodeVideoCodec != "h264" {
			t.Errorf("TranscodeVideoCodec = %v, want 'h264'", event.TranscodeVideoCodec)
		}
		if event.TranscodeAudioCodec == nil || *event.TranscodeAudioCodec != "aac" {
			t.Errorf("TranscodeAudioCodec = %v, want 'aac'", event.TranscodeAudioCodec)
		}
		if event.TranscodeVideoWidth == nil || *event.TranscodeVideoWidth != 1920 {
			t.Errorf("TranscodeVideoWidth = %v, want 1920", event.TranscodeVideoWidth)
		}
		if event.TranscodeVideoHeight == nil || *event.TranscodeVideoHeight != 1080 {
			t.Errorf("TranscodeVideoHeight = %v, want 1080", event.TranscodeVideoHeight)
		}

		// Verify hardware acceleration
		if event.TranscodeHWRequested == nil || *event.TranscodeHWRequested != 1 {
			t.Errorf("TranscodeHWRequested = %v, want 1", event.TranscodeHWRequested)
		}
		if event.TranscodeHWDecode == nil || *event.TranscodeHWDecode != "videotoolbox" {
			t.Errorf("TranscodeHWDecode = %v, want 'videotoolbox'", event.TranscodeHWDecode)
		}
		if event.TranscodeHWFullPipeline == nil || *event.TranscodeHWFullPipeline != 1 {
			t.Errorf("TranscodeHWFullPipeline = %v, want 1", event.TranscodeHWFullPipeline)
		}

		// Verify transcode progress
		if event.TranscodeProgress == nil || *event.TranscodeProgress != 45 {
			t.Errorf("TranscodeProgress = %v, want 45", event.TranscodeProgress)
		}
		if event.TranscodeSpeed == nil || *event.TranscodeSpeed != "2.3x" {
			t.Errorf("TranscodeSpeed = %v, want '2.3x'", event.TranscodeSpeed)
		}
	})

	t.Run("copy_decision_session", func(t *testing.T) {
		session := &models.PlexSession{
			SessionKey: "test-session-3",
			TranscodeSession: &models.PlexTranscodeSession{
				VideoDecision:    "copy",
				AudioDecision:    "copy",
				SourceVideoCodec: "h264",
				SourceAudioCodec: "aac",
			},
		}

		event := &models.PlaybackEvent{}
		m.populateQualityMetrics(event, session)

		// Verify copy decision
		if event.TranscodeDecision == nil || *event.TranscodeDecision != "copy" {
			t.Errorf("TranscodeDecision = %v, want 'copy'", event.TranscodeDecision)
		}
	})

	t.Run("empty_session", func(t *testing.T) {
		session := &models.PlexSession{
			SessionKey: "test-session-4",
		}

		event := &models.PlaybackEvent{}
		m.populateQualityMetrics(event, session)

		// Direct play when no media or transcode
		if event.TranscodeDecision == nil || *event.TranscodeDecision != "direct play" {
			t.Errorf("TranscodeDecision = %v, want 'direct play'", event.TranscodeDecision)
		}
	})
}
