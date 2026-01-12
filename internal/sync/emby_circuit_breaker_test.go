// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gobreaker "github.com/sony/gobreaker/v2"
)

// TestEmbyCircuitBreaker_OpensAfterFailures verifies circuit opens after exceeding failure threshold
func TestEmbyCircuitBreaker_OpensAfterFailures(t *testing.T) {
	// Create a mock server that always fails
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	cfg := EmbyCircuitBreakerConfig{
		BaseURL: failServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewEmbyCircuitBreakerClient(cfg)

	// Initial state should be closed
	if cbc.State() != gobreaker.StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", cbc.State())
	}

	// Simulate 11 API calls with 100% failure rate
	for i := 0; i < 11; i++ {
		_ = cbc.Ping(context.Background())
	}

	// After 100% failure rate with 10+ requests, circuit should be open
	if cbc.State() != gobreaker.StateOpen {
		t.Errorf("Expected circuit to be Open after 100%% failure rate, got %v", cbc.State())
	}

	// Verify next request is rejected with ErrOpenState
	err := cbc.Ping(context.Background())
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Errorf("Expected ErrOpenState when circuit is open, got %v", err)
	}
}

// TestEmbyCircuitBreaker_DoesNotOpenBelowThreshold verifies circuit stays closed below failure threshold
func TestEmbyCircuitBreaker_DoesNotOpenBelowThreshold(t *testing.T) {
	requestCount := 0
	// Create a mock server that succeeds 60% of the time (below 60% failure threshold)
	mixedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		// Fail only 50% of requests (below threshold)
		if requestCount%2 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mixedServer.Close()

	cfg := EmbyCircuitBreakerConfig{
		BaseURL: mixedServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewEmbyCircuitBreakerClient(cfg)

	// Make 10 requests (50% failure rate - below threshold)
	for i := 0; i < 10; i++ {
		_ = cbc.Ping(context.Background())
	}

	// Circuit should still be closed (50% < 60% threshold)
	if cbc.State() != gobreaker.StateClosed {
		t.Errorf("Expected circuit to remain Closed with 50%% failure rate, got %v", cbc.State())
	}
}

// TestEmbyCircuitBreaker_RequiresMinimumRequests verifies circuit requires minimum 10 requests
func TestEmbyCircuitBreaker_RequiresMinimumRequests(t *testing.T) {
	// Create a mock server that always fails
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	cfg := EmbyCircuitBreakerConfig{
		BaseURL: failServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewEmbyCircuitBreakerClient(cfg)

	// Simulate only 5 API calls with 100% failure rate
	// Circuit should NOT open because we need minimum 10 requests
	for i := 0; i < 5; i++ {
		_ = cbc.Ping(context.Background())
	}

	// Circuit should still be closed despite 100% failure rate (< 10 requests)
	if cbc.State() != gobreaker.StateClosed {
		t.Errorf("Expected circuit to remain Closed with <10 requests, got %v", cbc.State())
	}
}

// TestEmbyCircuitBreaker_GetSessionsWithCircuitBreaker tests GetSessions method
func TestEmbyCircuitBreaker_GetSessionsWithCircuitBreaker(t *testing.T) {
	// Create a mock server that returns valid sessions
	sessionsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer sessionsServer.Close()

	cfg := EmbyCircuitBreakerConfig{
		BaseURL: sessionsServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewEmbyCircuitBreakerClient(cfg)

	sessions, err := cbc.GetSessions(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if sessions == nil {
		t.Error("Expected sessions slice, got nil")
	}
}

// TestEmbyCircuitBreaker_GetSystemInfoWithCircuitBreaker tests GetSystemInfo method
func TestEmbyCircuitBreaker_GetSystemInfoWithCircuitBreaker(t *testing.T) {
	// Create a mock server that returns valid system info
	infoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		info := EmbySystemInfo{
			ServerName:      "Test Emby",
			Version:         "4.7.0",
			ID:              "test-id-456",
			OperatingSystem: "Linux",
		}
		_ = json.NewEncoder(w).Encode(info)
	}))
	defer infoServer.Close()

	cfg := EmbyCircuitBreakerConfig{
		BaseURL: infoServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewEmbyCircuitBreakerClient(cfg)

	info, err := cbc.GetSystemInfo(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if info == nil {
		t.Fatal("Expected system info, got nil")
	}

	if info.ServerName != "Test Emby" {
		t.Errorf("Expected ServerName 'Test Emby', got '%s'", info.ServerName)
	}
}

// TestEmbyCircuitBreaker_GetWebSocketURL tests that GetWebSocketURL is a passthrough
func TestEmbyCircuitBreaker_GetWebSocketURL(t *testing.T) {
	cfg := EmbyCircuitBreakerConfig{
		BaseURL: "http://localhost:8096",
		APIKey:  "test-key",
		UserID:  "user123",
	}

	cbc := NewEmbyCircuitBreakerClient(cfg)

	wsURL, err := cbc.GetWebSocketURL()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if wsURL == "" {
		t.Error("Expected WebSocket URL, got empty string")
	}

	// Should contain ws:// or wss://
	if len(wsURL) < 5 || (wsURL[:5] != "ws://" && wsURL[:6] != "wss://") {
		t.Errorf("Expected WebSocket URL to start with ws:// or wss://, got %s", wsURL)
	}
}

// TestEmbyCircuitBreaker_CountsAndName tests Counts() and Name() methods
func TestEmbyCircuitBreaker_CountsAndName(t *testing.T) {
	cfg := EmbyCircuitBreakerConfig{
		BaseURL: "http://localhost:8096",
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewEmbyCircuitBreakerClient(cfg)

	// Check name
	if cbc.Name() != "emby-api" {
		t.Errorf("Expected name 'emby-api', got '%s'", cbc.Name())
	}

	// Check initial counts
	counts := cbc.Counts()
	if counts.Requests != 0 {
		t.Errorf("Expected 0 requests initially, got %d", counts.Requests)
	}
}

// TestEmbyCircuitBreaker_TransitionsToHalfOpen tests circuit transition to half-open after timeout
func TestEmbyCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	// Create a failing server
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	client := NewEmbyClient(failServer.URL, "test-key", "")
	cbName := "emby-test-circuit"

	// Create circuit breaker with 100ms timeout for testing
	cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        cbName,
		MaxRequests: 3,
		Interval:    time.Second,
		Timeout:     100 * time.Millisecond, // Short timeout for testing

		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < 10 {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= 0.6
		},
	})

	cbc := &EmbyCircuitBreakerClient{
		client: client,
		cb:     cb,
		name:   cbName,
	}

	// Force circuit to open state
	for i := 0; i < 11; i++ {
		_ = cbc.Ping(context.Background())
	}

	// Verify circuit is open
	if cbc.State() != gobreaker.StateOpen {
		t.Fatalf("Expected circuit to be Open, got %v", cbc.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Create a success server
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer successServer.Close()

	// Update client to point to success server
	cbc.client = NewEmbyClient(successServer.URL, "test-key", "")

	// Next request should trigger transition to half-open then closed on success
	_ = cbc.Ping(context.Background())

	// State should not be open anymore
	if cbc.State() == gobreaker.StateOpen {
		t.Errorf("Expected circuit to transition from Open after timeout, still Open")
	}
}

// TestEmbyCircuitBreaker_GetUsersWithCircuitBreaker tests GetUsers method
func TestEmbyCircuitBreaker_GetUsersWithCircuitBreaker(t *testing.T) {
	// Create a mock server that returns valid users
	usersServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		users := []EmbyUser{
			{ID: "user1", Name: "Test User 1"},
			{ID: "user2", Name: "Test User 2"},
		}
		_ = json.NewEncoder(w).Encode(users)
	}))
	defer usersServer.Close()

	cfg := EmbyCircuitBreakerConfig{
		BaseURL: usersServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewEmbyCircuitBreakerClient(cfg)

	users, err := cbc.GetUsers(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	if users[0].Name != "Test User 1" {
		t.Errorf("Expected first user name 'Test User 1', got '%s'", users[0].Name)
	}
}
