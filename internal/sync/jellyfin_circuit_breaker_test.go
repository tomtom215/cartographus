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

// TestJellyfinCircuitBreaker_OpensAfterFailures verifies circuit opens after exceeding failure threshold
func TestJellyfinCircuitBreaker_OpensAfterFailures(t *testing.T) {
	// Create a mock server that always fails
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	cfg := JellyfinCircuitBreakerConfig{
		BaseURL: failServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewJellyfinCircuitBreakerClient(cfg)

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

// TestJellyfinCircuitBreaker_DoesNotOpenBelowThreshold verifies circuit stays closed below failure threshold
func TestJellyfinCircuitBreaker_DoesNotOpenBelowThreshold(t *testing.T) {
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

	cfg := JellyfinCircuitBreakerConfig{
		BaseURL: mixedServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewJellyfinCircuitBreakerClient(cfg)

	// Make 10 requests (50% failure rate - below threshold)
	for i := 0; i < 10; i++ {
		_ = cbc.Ping(context.Background())
	}

	// Circuit should still be closed (50% < 60% threshold)
	if cbc.State() != gobreaker.StateClosed {
		t.Errorf("Expected circuit to remain Closed with 50%% failure rate, got %v", cbc.State())
	}
}

// TestJellyfinCircuitBreaker_RequiresMinimumRequests verifies circuit requires minimum 10 requests
func TestJellyfinCircuitBreaker_RequiresMinimumRequests(t *testing.T) {
	// Create a mock server that always fails
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	cfg := JellyfinCircuitBreakerConfig{
		BaseURL: failServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewJellyfinCircuitBreakerClient(cfg)

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

// TestJellyfinCircuitBreaker_GetSessionsWithCircuitBreaker tests GetSessions method
func TestJellyfinCircuitBreaker_GetSessionsWithCircuitBreaker(t *testing.T) {
	// Create a mock server that returns valid sessions
	sessionsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer sessionsServer.Close()

	cfg := JellyfinCircuitBreakerConfig{
		BaseURL: sessionsServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewJellyfinCircuitBreakerClient(cfg)

	sessions, err := cbc.GetSessions(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if sessions == nil {
		t.Error("Expected sessions slice, got nil")
	}
}

// TestJellyfinCircuitBreaker_GetSystemInfoWithCircuitBreaker tests GetSystemInfo method
func TestJellyfinCircuitBreaker_GetSystemInfoWithCircuitBreaker(t *testing.T) {
	// Create a mock server that returns valid system info
	infoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		info := JellyfinSystemInfo{
			ServerName:      "Test Jellyfin",
			Version:         "10.8.0",
			ID:              "test-id-123",
			OperatingSystem: "Linux",
		}
		_ = json.NewEncoder(w).Encode(info)
	}))
	defer infoServer.Close()

	cfg := JellyfinCircuitBreakerConfig{
		BaseURL: infoServer.URL,
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewJellyfinCircuitBreakerClient(cfg)

	info, err := cbc.GetSystemInfo(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if info == nil {
		t.Fatal("Expected system info, got nil")
	}

	if info.ServerName != "Test Jellyfin" {
		t.Errorf("Expected ServerName 'Test Jellyfin', got '%s'", info.ServerName)
	}
}

// TestJellyfinCircuitBreaker_GetWebSocketURL tests that GetWebSocketURL is a passthrough
func TestJellyfinCircuitBreaker_GetWebSocketURL(t *testing.T) {
	cfg := JellyfinCircuitBreakerConfig{
		BaseURL: "http://localhost:8096",
		APIKey:  "test-key",
		UserID:  "user123",
	}

	cbc := NewJellyfinCircuitBreakerClient(cfg)

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

// TestJellyfinCircuitBreaker_CountsAndName tests Counts() and Name() methods
func TestJellyfinCircuitBreaker_CountsAndName(t *testing.T) {
	cfg := JellyfinCircuitBreakerConfig{
		BaseURL: "http://localhost:8096",
		APIKey:  "test-key",
		UserID:  "",
	}

	cbc := NewJellyfinCircuitBreakerClient(cfg)

	// Check name
	if cbc.Name() != "jellyfin-api" {
		t.Errorf("Expected name 'jellyfin-api', got '%s'", cbc.Name())
	}

	// Check initial counts
	counts := cbc.Counts()
	if counts.Requests != 0 {
		t.Errorf("Expected 0 requests initially, got %d", counts.Requests)
	}
}

// TestJellyfinCircuitBreaker_TransitionsToHalfOpen tests circuit transition to half-open after timeout
func TestJellyfinCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	// Create a failing server
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	client := NewJellyfinClient(failServer.URL, "test-key", "")
	cbName := "jellyfin-test-circuit"

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

	cbc := &JellyfinCircuitBreakerClient{
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
	cbc.client = NewJellyfinClient(successServer.URL, "test-key", "")

	// Next request should trigger transition to half-open then closed on success
	_ = cbc.Ping(context.Background())

	// State should not be open anymore
	if cbc.State() == gobreaker.StateOpen {
		t.Errorf("Expected circuit to transition from Open after timeout, still Open")
	}
}
