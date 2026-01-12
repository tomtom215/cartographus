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

// mockJellyfinWebSocketServer creates a test WebSocket server that simulates Jellyfin
type mockJellyfinWebSocketServer struct {
	server   *httptest.Server
	upgrader websocket.Upgrader
	connChan chan *websocket.Conn
}

// newMockJellyfinWebSocketServer creates a new mock Jellyfin WebSocket server
func newMockJellyfinWebSocketServer() *mockJellyfinWebSocketServer {
	mock := &mockJellyfinWebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
		connChan: make(chan *websocket.Conn, 1),
	}

	// Create HTTP test server
	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key
		apiKey := r.URL.Query().Get("api_key")
		if apiKey != "test-api-key" {
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
func (m *mockJellyfinWebSocketServer) close() {
	m.server.Close()
}

// sendMessage sends a message to connected client
func (m *mockJellyfinWebSocketServer) sendMessage(conn *websocket.Conn, msgType string, data interface{}) error {
	msg := JellyfinWSMessage{
		MessageType: msgType,
	}

	if data != nil {
		dataBytes, err := json.Marshal(data)
		if err != nil {
			return err
		}
		msg.Data = dataBytes
	}

	return conn.WriteJSON(msg)
}

// getWebSocketURL returns the WebSocket URL for testing
func (m *mockJellyfinWebSocketServer) getWebSocketURL() string {
	// Replace http:// with ws://
	return "ws" + strings.TrimPrefix(m.server.URL, "http") + "/socket?api_key=test-api-key&deviceId=cartographus"
}

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNewJellyfinWebSocketClient(t *testing.T) {
	client := NewJellyfinWebSocketClient("ws://localhost:8096/socket", "test-api-key")

	if client == nil {
		t.Fatal("NewJellyfinWebSocketClient returned nil")
	}
	checkStringEqual(t, "wsURL", client.wsURL, "ws://localhost:8096/socket")
	checkStringEqual(t, "apiKey", client.apiKey, "test-api-key")
	checkTrue(t, "stopChan not nil", client.stopChan != nil)
}

// ============================================================================
// Connection Tests
// ============================================================================

func TestJellyfinWebSocketClient_Connect(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	client := NewJellyfinWebSocketClient(mock.getWebSocketURL(), "test-api-key")

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

func TestJellyfinWebSocketClient_ConnectAlreadyConnected(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	client := NewJellyfinWebSocketClient(mock.getWebSocketURL(), "test-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connection
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("First Connect() failed: %v", err)
	}
	defer client.Close()

	// Drain the connection channel
	<-mock.connChan

	// Second connection should be a no-op
	err = client.Connect(ctx)
	if err != nil {
		t.Errorf("Second Connect() failed: %v", err)
	}
}

func TestJellyfinWebSocketClient_AuthenticationFailure(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	// Use wrong API key
	wsURL := "ws" + strings.TrimPrefix(mock.server.URL, "http") + "/socket?api_key=wrong-key&deviceId=cartographus"
	client := NewJellyfinWebSocketClient(wsURL, "wrong-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err == nil {
		t.Error("Connect() succeeded with invalid API key, want error")
	}
}

// ============================================================================
// Message Handling Tests
// ============================================================================

func TestJellyfinWebSocketClient_SessionsMessage(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	client := NewJellyfinWebSocketClient(mock.getWebSocketURL(), "test-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Track received sessions
	var receivedCount int32
	var mu stdsync.Mutex
	var receivedSessions []models.JellyfinSession

	client.SetCallbacks(
		func(sessions []models.JellyfinSession) {
			atomic.AddInt32(&receivedCount, 1)
			mu.Lock()
			receivedSessions = sessions
			mu.Unlock()
		},
		nil, nil,
	)

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	// Get server connection
	var serverConn *websocket.Conn
	select {
	case serverConn = <-mock.connChan:
		defer serverConn.Close()
	case <-time.After(1 * time.Second):
		t.Fatal("Server did not receive connection")
	}

	// Send sessions message
	sessions := []models.JellyfinSession{
		{
			ID:         "session-123",
			UserName:   "TestUser",
			DeviceName: "Test Device",
			Client:     "Jellyfin Web",
			NowPlayingItem: &models.JellyfinNowPlayingItem{
				ID:   "item-456",
				Name: "Test Movie",
				Type: "Movie",
			},
		},
		{
			ID:         "session-456",
			UserName:   "AnotherUser",
			DeviceName: "Phone",
			Client:     "Jellyfin Mobile",
		},
	}

	if err := mock.sendMessage(serverConn, "Sessions", sessions); err != nil {
		t.Fatalf("Failed to send sessions: %v", err)
	}

	// Wait for message processing
	time.Sleep(100 * time.Millisecond)

	// Verify sessions received
	if atomic.LoadInt32(&receivedCount) != 1 {
		t.Errorf("Received %d session updates, want 1", atomic.LoadInt32(&receivedCount))
	}

	mu.Lock()
	sessCount := len(receivedSessions)
	mu.Unlock()

	if sessCount != 2 {
		t.Errorf("Received %d sessions, want 2", sessCount)
	}
}

func TestJellyfinWebSocketClient_UserDataChangedMessage(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	client := NewJellyfinWebSocketClient(mock.getWebSocketURL(), "test-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mu stdsync.Mutex
	var userDataReceived bool
	var receivedUserID string

	client.SetCallbacks(
		nil,
		func(userID string, _ any) {
			mu.Lock()
			userDataReceived = true
			receivedUserID = userID
			mu.Unlock()
		},
		nil,
	)

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	serverConn := <-mock.connChan
	defer serverConn.Close()

	// Send user data changed message
	userData := map[string]interface{}{
		"UserId": "user-abc-123",
		"UserDataList": []map[string]interface{}{
			{
				"ItemId":           "item-123",
				"PlayedPercentage": 45.5,
			},
		},
	}

	if err := mock.sendMessage(serverConn, "UserDataChanged", userData); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	received := userDataReceived
	userID := receivedUserID
	mu.Unlock()

	if !received {
		t.Error("UserDataChanged not received")
	}
	if userID != "user-abc-123" {
		t.Errorf("UserID = %s, want user-abc-123", userID)
	}
}

func TestJellyfinWebSocketClient_PlaystateMessage(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	client := NewJellyfinWebSocketClient(mock.getWebSocketURL(), "test-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mu stdsync.Mutex
	var playstateReceived bool
	var receivedSessionID string
	var receivedCommand string

	client.SetCallbacks(
		nil, nil,
		func(sessionID, command string) {
			mu.Lock()
			playstateReceived = true
			receivedSessionID = sessionID
			receivedCommand = command
			mu.Unlock()
		},
	)

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	serverConn := <-mock.connChan
	defer serverConn.Close()

	// Send playstate message
	playstate := map[string]interface{}{
		"SessionId": "session-123",
		"Command":   "Pause",
	}

	if err := mock.sendMessage(serverConn, "Playstate", playstate); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	received := playstateReceived
	sessID := receivedSessionID
	cmd := receivedCommand
	mu.Unlock()

	if !received {
		t.Error("Playstate not received")
	}
	if sessID != "session-123" {
		t.Errorf("SessionID = %s, want session-123", sessID)
	}
	if cmd != "Pause" {
		t.Errorf("Command = %s, want Pause", cmd)
	}
}

func TestJellyfinWebSocketClient_KeepAliveMessage(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	client := NewJellyfinWebSocketClient(mock.getWebSocketURL(), "test-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// No callbacks - just verify no crash on KeepAlive message
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	serverConn := <-mock.connChan
	defer serverConn.Close()

	// Send KeepAlive message (should be ignored without error)
	if err := mock.sendMessage(serverConn, "KeepAlive", nil); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Send ForceKeepAlive message (should be handled gracefully)
	if err := mock.sendMessage(serverConn, "ForceKeepAlive", nil); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// If we get here, the messages were handled without crashing
}

func TestJellyfinWebSocketClient_UnknownMessageType(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	client := NewJellyfinWebSocketClient(mock.getWebSocketURL(), "test-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	serverConn := <-mock.connChan
	defer serverConn.Close()

	// Send unknown message type (should be logged and ignored)
	if err := mock.sendMessage(serverConn, "UnknownType", map[string]string{"foo": "bar"}); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// If we get here, the unknown message was handled without crashing
}

// ============================================================================
// Close and Lifecycle Tests
// ============================================================================

func TestJellyfinWebSocketClient_Close(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	client := NewJellyfinWebSocketClient(mock.getWebSocketURL(), "test-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// Verify connected
	if !client.IsConnected() {
		t.Error("IsConnected() = false after Connect()")
	}

	// Drain connection
	<-mock.connChan

	// Close client
	if err := client.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Verify disconnected
	if client.IsConnected() {
		t.Error("IsConnected() = true after Close()")
	}
}

func TestJellyfinWebSocketClient_IsConnectedNotConnected(t *testing.T) {
	client := NewJellyfinWebSocketClient("ws://localhost:8096/socket", "test-api-key")

	if client.IsConnected() {
		t.Error("IsConnected() = true before Connect()")
	}
}

// ============================================================================
// SendMessage Tests
// ============================================================================

func TestJellyfinWebSocketClient_SendMessage(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	client := NewJellyfinWebSocketClient(mock.getWebSocketURL(), "test-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	serverConn := <-mock.connChan
	defer serverConn.Close()

	// Send a message
	msg := JellyfinWSMessage{
		MessageType: "SessionsStart",
	}

	err := client.SendMessage(msg)
	checkNoError(t, err)

	// Read message on server side
	var received JellyfinWSMessage
	if err := serverConn.ReadJSON(&received); err != nil {
		t.Fatalf("Server failed to read message: %v", err)
	}

	checkStringEqual(t, "MessageType", received.MessageType, "SessionsStart")
}

func TestJellyfinWebSocketClient_SendMessageNotConnected(t *testing.T) {
	client := NewJellyfinWebSocketClient("ws://localhost:8096/socket", "test-api-key")

	msg := JellyfinWSMessage{
		MessageType: "KeepAlive",
	}

	err := client.SendMessage(msg)
	checkError(t, err)
	checkErrorContains(t, err, "not connected")
}

// ============================================================================
// SetCallbacks Tests
// ============================================================================

func TestJellyfinWebSocketClient_SetCallbacks(t *testing.T) {
	client := NewJellyfinWebSocketClient("ws://localhost:8096/socket", "test-api-key")

	// Initially no callbacks
	client.callbackMu.RLock()
	hasOnSession := client.onSession != nil
	client.callbackMu.RUnlock()

	if hasOnSession {
		t.Error("onSession should be nil initially")
	}

	// Set callbacks
	client.SetCallbacks(
		func(_ []models.JellyfinSession) {},
		func(_ string, _ any) {},
		func(_, _ string) {},
	)

	client.callbackMu.RLock()
	hasOnSession = client.onSession != nil
	hasOnUserData := client.onUserDataChanged != nil
	hasOnPlayState := client.onPlayStateChange != nil
	client.callbackMu.RUnlock()

	if !hasOnSession {
		t.Error("onSession should be set")
	}
	if !hasOnUserData {
		t.Error("onUserDataChanged should be set")
	}
	if !hasOnPlayState {
		t.Error("onPlayStateChange should be set")
	}
}

// ============================================================================
// Concurrent Access Tests
// ============================================================================

func TestJellyfinWebSocketClient_ConcurrentCallbacks(t *testing.T) {
	mock := newMockJellyfinWebSocketServer()
	defer mock.close()

	client := NewJellyfinWebSocketClient(mock.getWebSocketURL(), "test-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var receivedCount int32

	client.SetCallbacks(
		func(_ []models.JellyfinSession) {
			atomic.AddInt32(&receivedCount, 1)
		},
		nil, nil,
	)

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	serverConn := <-mock.connChan
	defer serverConn.Close()

	// Concurrently update callbacks while sending messages
	done := make(chan bool)
	go func() {
		for i := 0; i < 10; i++ {
			client.SetCallbacks(
				func(_ []models.JellyfinSession) {
					atomic.AddInt32(&receivedCount, 1)
				},
				nil, nil,
			)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Send multiple sessions messages
	for i := 0; i < 10; i++ {
		sessions := []models.JellyfinSession{{ID: "test"}}
		if err := mock.sendMessage(serverConn, "Sessions", sessions); err != nil {
			t.Errorf("Failed to send message: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	<-done
	time.Sleep(100 * time.Millisecond)

	// Verify no race conditions (just check > 0)
	if count := atomic.LoadInt32(&receivedCount); count == 0 {
		t.Error("No sessions received")
	}
}

// ============================================================================
// Message Parsing Tests
// ============================================================================

func TestJellyfinWebSocketClient_HandleMessageInvalidJSON(t *testing.T) {
	client := NewJellyfinWebSocketClient("ws://localhost:8096/socket", "test-api-key")

	// Call handleMessage with invalid JSON (should not panic)
	client.handleMessage([]byte(`{invalid json}`))
	// If we get here, the invalid JSON was handled without crashing
}

func TestJellyfinWebSocketClient_HandleMessageNilCallbacks(t *testing.T) {
	client := NewJellyfinWebSocketClient("ws://localhost:8096/socket", "test-api-key")

	// No callbacks set - should handle messages without panic
	sessions := []models.JellyfinSession{{ID: "test"}}
	data, _ := json.Marshal(JellyfinWSMessage{
		MessageType: "Sessions",
		Data:        mustMarshalJSON(sessions),
	})

	client.handleMessage(data)
	// If we get here, message was handled without crashing
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkJellyfinWebSocketClient_MessageHandling(b *testing.B) {
	client := NewJellyfinWebSocketClient("ws://localhost:8096/socket", "test-api-key")
	client.SetCallbacks(
		func(_ []models.JellyfinSession) {},
		nil, nil,
	)

	sessions := []models.JellyfinSession{
		{ID: "session-1", UserName: "User1"},
		{ID: "session-2", UserName: "User2"},
	}
	msg := JellyfinWSMessage{
		MessageType: "Sessions",
		Data:        mustMarshalJSON(sessions),
	}
	data, _ := json.Marshal(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.handleMessage(data)
	}
}

// mustMarshalJSON marshals to JSON or panics (for test data only)
func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
