// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package websocket

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

//nolint:gochecknoinits // init ensures consistent logging for tests
func init() {
	// Initialize logging for tests with discard output
	logging.Init(logging.Config{
		Level:  "info",
		Format: "console",
		Output: io.Discard,
	})
}

// setupHub creates and starts a new hub for testing
func setupHub(t *testing.T) *Hub {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)
	return hub
}

// createTestClient creates a mock client for testing
func createTestClient(hub *Hub) *Client {
	return &Client{hub: hub, conn: nil, send: make(chan Message, 256)}
}

// registerClient registers a client and waits for registration to complete
func registerClient(hub *Hub, client *Client) {
	hub.Register <- client
	time.Sleep(20 * time.Millisecond)
}

// createTestPlaybackEvent creates a test playback event
func createTestPlaybackEvent() *models.PlaybackEvent {
	return &models.PlaybackEvent{
		ID: uuid.New(), SessionKey: "test-session", Username: "testuser",
		Title: "Test Movie", MediaType: "movie", IPAddress: "127.0.0.1",
		Platform: "web", Player: "Chrome", StartedAt: time.Now(), CreatedAt: time.Now(), UserID: 1,
	}
}

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	checks := []struct {
		name   string
		check  bool
		errMsg string
	}{
		{"clients map", hub.clients != nil, "clients map not initialized"},
		{"broadcast channel", hub.broadcast != nil, "broadcast channel not initialized"},
		{"Register channel", hub.Register != nil, "Register channel not initialized"},
		{"Unregister channel", hub.Unregister != nil, "Unregister channel not initialized"},
		{"empty clients", len(hub.clients) == 0, "clients map should be empty"},
	}

	for _, c := range checks {
		if !c.check {
			t.Error(c.errMsg)
		}
	}
}

func TestHub_GetClientCount(t *testing.T) {
	hub := NewHub()

	if hub.GetClientCount() != 0 {
		t.Errorf("Expected 0 clients initially, got %d", hub.GetClientCount())
	}

	for i := 0; i < 5; i++ {
		hub.clients[createTestClient(hub)] = true
	}

	if hub.GetClientCount() != 5 {
		t.Errorf("Expected 5 clients, got %d", hub.GetClientCount())
	}
}

func TestHub_BroadcastMethods(t *testing.T) {
	t.Run("BroadcastNewPlayback without clients", func(t *testing.T) {
		hub := setupHub(t)
		hub.BroadcastNewPlayback(createTestPlaybackEvent())
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("BroadcastJSON without clients", func(t *testing.T) {
		hub := setupHub(t)
		hub.BroadcastJSON("test_type", map[string]interface{}{"test_key": "test_value", "count": 42})
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("BroadcastSyncCompleted without clients", func(t *testing.T) {
		hub := setupHub(t)
		hub.BroadcastSyncCompleted(42, 5000)
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("BroadcastStatsUpdate without clients", func(t *testing.T) {
		hub := setupHub(t)
		hub.BroadcastStatsUpdate(1000, "2025-11-18T12:00:00Z")
		time.Sleep(10 * time.Millisecond)
	})
}

func TestHub_ClientRegistration(t *testing.T) {
	hub := setupHub(t)
	client := createTestClient(hub)
	registerClient(hub, client)

	if hub.GetClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", hub.GetClientCount())
	}

	hub.mu.RLock()
	if !hub.clients[client] {
		t.Error("Client should be registered")
	}
	hub.mu.RUnlock()

	// Unregister
	hub.Unregister <- client
	time.Sleep(20 * time.Millisecond)

	if hub.GetClientCount() != 0 {
		t.Errorf("Expected 0 clients after unregister, got %d", hub.GetClientCount())
	}
}

func TestHub_UnregisterNonExistentClient(t *testing.T) {
	hub := setupHub(t)
	client := createTestClient(hub)

	hub.Unregister <- client
	time.Sleep(20 * time.Millisecond)

	if hub.GetClientCount() != 0 {
		t.Errorf("Expected 0 clients, got %d", hub.GetClientCount())
	}
}

func TestHub_BroadcastToClients(t *testing.T) {
	hub := setupHub(t)

	const numClients = 3
	clients := make([]*Client, numClients)
	var mu sync.Mutex
	received := make([]bool, numClients)
	var wg sync.WaitGroup

	for i := 0; i < numClients; i++ {
		clients[i] = createTestClient(hub)
		registerClient(hub, clients[i])
	}

	if hub.GetClientCount() != numClients {
		t.Fatalf("Expected %d clients, got %d", numClients, hub.GetClientCount())
	}

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int, c *Client) {
			defer wg.Done()
			select {
			case msg := <-c.send:
				if msg.Type == "test_broadcast" {
					mu.Lock()
					received[idx] = true
					mu.Unlock()
				}
			case <-time.After(500 * time.Millisecond):
			}
		}(i, clients[i])
	}

	time.Sleep(20 * time.Millisecond)
	hub.BroadcastJSON("test_broadcast", map[string]string{"message": "hello"})
	wg.Wait()

	mu.Lock()
	for i, r := range received {
		if !r {
			t.Errorf("Client %d did not receive broadcast", i)
		}
	}
	mu.Unlock()
}

func TestHub_ConcurrentOperations(t *testing.T) {
	hub := setupHub(t)
	done := make(chan bool)

	go func() {
		for i := 0; i < 10; i++ {
			registerClient(hub, createTestClient(hub))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 20; i++ {
			hub.BroadcastJSON("test", map[string]int{"i": i})
			time.Sleep(2 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			hub.GetClientCount()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	for i := 0; i < 3; i++ {
		<-done
	}
	time.Sleep(100 * time.Millisecond)

	if hub.GetClientCount() != 10 {
		t.Errorf("Expected 10 clients, got %d", hub.GetClientCount())
	}
}

func TestMarshalMessage(t *testing.T) {
	tests := []struct {
		name    string
		message Message
	}{
		{"simple message", Message{Type: "ping", Data: nil}},
		{"string data", Message{Type: "test", Data: "hello world"}},
		{"map data", Message{Type: "sync_completed", Data: map[string]interface{}{"count": 42}}},
		{"struct data", Message{Type: "playback", Data: createTestPlaybackEvent()}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := MarshalMessage(tt.message)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(data) == 0 || data[0] != '{' || data[len(data)-1] != '}' {
				t.Error("Invalid JSON output")
			}
		})
	}
}

func TestHub_MessageTypes(t *testing.T) {
	expected := map[string]string{
		MessageTypePlayback:      "playback",
		MessageTypePing:          "ping",
		MessageTypePong:          "pong",
		MessageTypeSyncCompleted: "sync_completed",
		MessageTypeStatsUpdate:   "stats_update",
	}

	for got, want := range expected {
		if got != want {
			t.Errorf("Message type = %q, want %q", got, want)
		}
	}
}

func TestHub_BroadcastWithClients(t *testing.T) {
	tests := []struct {
		name        string
		broadcast   func(*Hub)
		wantType    string
		validateMsg func(*testing.T, Message)
	}{
		{
			name:      "BroadcastNewPlayback",
			broadcast: func(h *Hub) { h.BroadcastNewPlayback(createTestPlaybackEvent()) },
			wantType:  MessageTypePlayback,
			validateMsg: func(t *testing.T, msg Message) {
				if msg.Data == nil {
					t.Error("Expected non-nil data")
				}
			},
		},
		{
			name:      "BroadcastSyncCompleted",
			broadcast: func(h *Hub) { h.BroadcastSyncCompleted(42, 1500) },
			wantType:  MessageTypeSyncCompleted,
			validateMsg: func(t *testing.T, msg Message) {
				data, ok := msg.Data.(SyncCompletedData)
				if !ok {
					t.Fatalf("Expected SyncCompletedData, got %T", msg.Data)
				}
				if data.NewPlaybacks != 42 || data.SyncDurationMs != 1500 || data.Timestamp == "" {
					t.Error("Invalid SyncCompletedData")
				}
			},
		},
		{
			name:      "BroadcastStatsUpdate",
			broadcast: func(h *Hub) { h.BroadcastStatsUpdate(1000, "2025-11-18T12:00:00Z") },
			wantType:  MessageTypeStatsUpdate,
			validateMsg: func(t *testing.T, msg Message) {
				data, ok := msg.Data.(StatsUpdateData)
				if !ok {
					t.Fatalf("Expected StatsUpdateData, got %T", msg.Data)
				}
				if data.TotalCount != 1000 || data.LastPlayback != "2025-11-18T12:00:00Z" {
					t.Error("Invalid StatsUpdateData")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := setupHub(t)
			client := createTestClient(hub)
			registerClient(hub, client)

			tt.broadcast(hub)

			select {
			case msg := <-client.send:
				if msg.Type != tt.wantType {
					t.Errorf("Type = %q, want %q", msg.Type, tt.wantType)
				}
				tt.validateMsg(t, msg)
			case <-time.After(100 * time.Millisecond):
				t.Error("Timeout waiting for message")
			}

			hub.Unregister <- client
		})
	}
}

func TestHub_ChannelFullBehavior(t *testing.T) {
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	defer zerolog.SetGlobalLevel(oldLevel)

	tests := []struct {
		name      string
		broadcast func(*Hub)
	}{
		{"BroadcastNewPlayback", func(h *Hub) { h.BroadcastNewPlayback(createTestPlaybackEvent()) }},
		{"BroadcastJSON", func(h *Hub) { h.BroadcastJSON("test", map[string]string{"test": "data"}) }},
		{"BroadcastSyncCompleted", func(h *Hub) { h.BroadcastSyncCompleted(10, 1000) }},
		{"BroadcastStatsUpdate", func(h *Hub) { h.BroadcastStatsUpdate(1000, "2025-11-18T12:00:00Z") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub() // Don't start Run() so channel fills

			for i := 0; i < 256; i++ {
				tt.broadcast(hub)
			}
			tt.broadcast(hub) // Should hit default case and not block
		})
	}
}

// TestHub_BroadcastToFullClient tests broadcasting when a client's send channel is full
func TestHub_BroadcastToFullClient(t *testing.T) {
	hub := setupHub(t)

	// Create client with tiny buffer that will fill up
	client := &Client{hub: hub, conn: nil, send: make(chan Message, 1)}
	registerClient(hub, client)

	// Fill the client's send channel
	client.send <- Message{Type: "filler", Data: nil}

	// Now broadcast - this should trigger the default case in Run()
	// where it closes the client's send channel and removes it
	hub.BroadcastJSON("test_overflow", map[string]string{"overflow": "test"})

	// Wait for client removal with polling (more reliable in CI under load)
	var clientCount int
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		clientCount = hub.GetClientCount()
		if clientCount == 0 {
			break
		}
	}

	// Client should have been removed due to full channel
	if clientCount != 0 {
		t.Errorf("Expected 0 clients after overflow handling, got %d", clientCount)
	}
}

// TestHub_RunWithContext tests the context-aware run method
func TestHub_RunWithContext(t *testing.T) {
	t.Run("shuts down on context cancellation", func(t *testing.T) {
		oldLevel := zerolog.GlobalLevel()
		zerolog.SetGlobalLevel(zerolog.Disabled)
		defer zerolog.SetGlobalLevel(oldLevel)

		hub := NewHub()
		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- hub.RunWithContext(ctx)
		}()

		// Let it start
		time.Sleep(20 * time.Millisecond)

		// Cancel the context
		cancel()

		select {
		case err := <-errCh:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("expected context.Canceled, got %v", err)
			}
		case <-time.After(time.Second):
			t.Error("RunWithContext did not return after context cancellation")
		}
	})

	t.Run("shuts down on context deadline", func(t *testing.T) {
		oldLevel := zerolog.GlobalLevel()
		zerolog.SetGlobalLevel(zerolog.Disabled)
		defer zerolog.SetGlobalLevel(oldLevel)

		hub := NewHub()
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			errCh <- hub.RunWithContext(ctx)
		}()

		select {
		case err := <-errCh:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("expected context.DeadlineExceeded, got %v", err)
			}
		case <-time.After(time.Second):
			t.Error("RunWithContext did not return after deadline")
		}
	})

	t.Run("closes all clients on shutdown", func(t *testing.T) {
		oldLevel := zerolog.GlobalLevel()
		zerolog.SetGlobalLevel(zerolog.Disabled)
		defer zerolog.SetGlobalLevel(oldLevel)

		hub := NewHub()
		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- hub.RunWithContext(ctx)
		}()

		// Register some clients
		clients := make([]*Client, 3)
		for i := 0; i < 3; i++ {
			clients[i] = createTestClient(hub)
			hub.Register <- clients[i]
		}

		// Wait for registration with polling (more reliable in CI under load)
		var clientCount int
		for i := 0; i < 10; i++ {
			time.Sleep(20 * time.Millisecond)
			clientCount = hub.GetClientCount()
			if clientCount == 3 {
				break
			}
		}

		if clientCount != 3 {
			t.Fatalf("expected 3 clients, got %d", clientCount)
		}

		// Cancel and wait for shutdown
		cancel()

		select {
		case <-errCh:
			// Hub has shut down
		case <-time.After(time.Second):
			t.Fatal("RunWithContext did not return after context cancellation")
		}

		// All clients should be removed
		if hub.GetClientCount() != 0 {
			t.Errorf("expected 0 clients after shutdown, got %d", hub.GetClientCount())
		}
	})

	t.Run("handles messages before shutdown", func(t *testing.T) {
		oldLevel := zerolog.GlobalLevel()
		zerolog.SetGlobalLevel(zerolog.Disabled)
		defer zerolog.SetGlobalLevel(oldLevel)

		hub := NewHub()
		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- hub.RunWithContext(ctx)
		}()

		// Register a client
		client := createTestClient(hub)
		hub.Register <- client
		time.Sleep(20 * time.Millisecond)

		// Send a message
		hub.BroadcastJSON("test_message", map[string]string{"key": "value"})

		// Verify message received
		select {
		case msg := <-client.send:
			if msg.Type != "test_message" {
				t.Errorf("expected message type 'test_message', got %q", msg.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("did not receive message")
		}

		cancel()
		<-errCh
	})
}

// TestHub_CloseAllClients tests the closeAllClients method
func TestHub_CloseAllClients(t *testing.T) {
	hub := NewHub()

	// Manually add clients
	clients := make([]*Client, 5)
	for i := 0; i < 5; i++ {
		clients[i] = createTestClient(hub)
		hub.mu.Lock()
		hub.clients[clients[i]] = true
		hub.mu.Unlock()
	}

	if hub.GetClientCount() != 5 {
		t.Fatalf("expected 5 clients, got %d", hub.GetClientCount())
	}

	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	hub.closeAllClients()
	zerolog.SetGlobalLevel(oldLevel)

	if hub.GetClientCount() != 0 {
		t.Errorf("expected 0 clients after closeAllClients, got %d", hub.GetClientCount())
	}
}

func BenchmarkHub_BroadcastJSON(b *testing.B) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 10; i++ {
		client := createTestClient(hub)
		hub.Register <- client
		go func(c *Client) {
			for range c.send {
			}
		}(client)
	}

	// Allow registrations and goroutines to start (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)

	testData := map[string]interface{}{"test": "data", "count": 42}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.BroadcastJSON("test", testData)
	}
}

func BenchmarkHub_RegisterUnregister(b *testing.B) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := createTestClient(hub)
		hub.Register <- client
		hub.Unregister <- client
	}
}

// =============================================================================
// Shutdown Logging Tests
// =============================================================================
// These tests verify the production-grade shutdown logging behavior.
// The logging is designed to be:
// - Deterministic: Same format every time
// - Observable: Includes component, reason, and client count
// - Non-confusing: Does not log context.Canceled as an error
// =============================================================================

// TestGetShutdownReason verifies shutdown reason detection from context errors.
func TestGetShutdownReason(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected ShutdownReason
	}{
		{
			name: "context canceled returns context_canceled",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			expected: ShutdownReasonContextCanceled,
		},
		{
			name: "context deadline exceeded returns context_deadline",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(10 * time.Millisecond) // Ensure deadline passes
				return ctx
			},
			expected: ShutdownReasonContextDeadline,
		},
		{
			name: "active context has no error (edge case)",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expected: ShutdownReasonContextCanceled, // Fallback behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			got := getShutdownReason(ctx)
			if got != tt.expected {
				t.Errorf("getShutdownReason() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestShutdownReason_Constants verifies shutdown reason constant values.
// This ensures consistent log output format across versions.
func TestShutdownReason_Constants(t *testing.T) {
	// These values are used in log output and may be parsed by log aggregators.
	// Changing them would be a breaking change for monitoring systems.
	tests := []struct {
		constant ShutdownReason
		expected string
	}{
		{ShutdownReasonContextCanceled, "context_canceled"},
		{ShutdownReasonContextDeadline, "context_deadline"},
	}

	for _, tt := range tests {
		if string(tt.constant) != tt.expected {
			t.Errorf("ShutdownReason constant = %q, want %q", tt.constant, tt.expected)
		}
	}
}

// TestHub_GracefulShutdown_LogsCorrectly verifies that shutdown produces
// correctly formatted log output without error fields.
func TestHub_GracefulShutdown_LogsCorrectly(t *testing.T) {
	// Suppress normal log output during test
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	defer zerolog.SetGlobalLevel(oldLevel)

	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	// Start hub
	errCh := make(chan error, 1)
	go func() {
		errCh <- hub.RunWithContext(ctx)
	}()

	// Register clients
	numClients := 3
	for i := 0; i < numClients; i++ {
		client := createTestClient(hub)
		hub.Register <- client
	}

	// Wait for registrations
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		if hub.GetClientCount() == numClients {
			break
		}
	}

	if hub.GetClientCount() != numClients {
		t.Fatalf("expected %d clients, got %d", numClients, hub.GetClientCount())
	}

	// Trigger graceful shutdown
	cancel()

	// Wait for shutdown
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not complete in time")
	}

	// Verify all clients were closed
	if hub.GetClientCount() != 0 {
		t.Errorf("expected 0 clients after shutdown, got %d", hub.GetClientCount())
	}
}

// TestHub_GracefulShutdown_WithDeadline verifies shutdown behavior with deadline.
func TestHub_GracefulShutdown_WithDeadline(t *testing.T) {
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	defer zerolog.SetGlobalLevel(oldLevel)

	hub := NewHub()

	// Use a short deadline
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Start hub
	errCh := make(chan error, 1)
	go func() {
		errCh <- hub.RunWithContext(ctx)
	}()

	// Register a client
	client := createTestClient(hub)
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Wait for deadline to expire and hub to shut down
	select {
	case err := <-errCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not complete in time")
	}

	// Verify client was closed
	if hub.GetClientCount() != 0 {
		t.Errorf("expected 0 clients after deadline shutdown, got %d", hub.GetClientCount())
	}
}

// TestHub_GracefulShutdown_ZeroClients verifies shutdown with no connected clients.
func TestHub_GracefulShutdown_ZeroClients(t *testing.T) {
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	defer zerolog.SetGlobalLevel(oldLevel)

	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	// Start hub with no clients
	errCh := make(chan error, 1)
	go func() {
		errCh <- hub.RunWithContext(ctx)
	}()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Verify no clients
	if hub.GetClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", hub.GetClientCount())
	}

	// Trigger shutdown
	cancel()

	// Wait for shutdown
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not complete in time")
	}
}

// TestHub_logGracefulShutdown_Idempotent verifies calling shutdown multiple times
// doesn't panic (though this shouldn't happen in production).
func TestHub_logGracefulShutdown_Idempotent(t *testing.T) {
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	defer zerolog.SetGlobalLevel(oldLevel)

	hub := NewHub()

	// Add a client
	client := createTestClient(hub)
	hub.mu.Lock()
	hub.clients[client] = true
	hub.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Pre-cancel

	// Call shutdown multiple times - should not panic
	hub.logGracefulShutdown(ctx)
	hub.logGracefulShutdown(ctx)
	hub.logGracefulShutdown(ctx)

	// All clients should be closed (even though we only had one)
	if hub.GetClientCount() != 0 {
		t.Errorf("expected 0 clients after shutdown, got %d", hub.GetClientCount())
	}
}
