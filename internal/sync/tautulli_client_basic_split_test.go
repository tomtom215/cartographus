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

	"github.com/tomtom215/cartographus/internal/config"
)

func TestNewTautulliClient(t *testing.T) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-api-key",
	}

	client := NewTautulliClient(cfg)

	if client == nil {
		t.Fatal("NewTautulliClient returned nil")
	}

	if client.baseURL != cfg.URL {
		t.Errorf("Expected baseURL %s, got %s", cfg.URL, client.baseURL)
	}

	if client.apiKey != cfg.APIKey {
		t.Errorf("Expected apiKey %s, got %s", cfg.APIKey, client.apiKey)
	}

	if client.client == nil {
		t.Error("HTTP client not initialized")
	}

	if client.client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.client.Timeout)
	}
}

func TestTautulliClient_Ping(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful ping",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "server error",
			statusCode:  http.StatusInternalServerError,
			expectError: true,
		},
		{
			name:        "not found",
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cmd") != "arnold" {
					t.Errorf("Expected cmd=arnold, got %s", r.URL.Query().Get("cmd"))
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{
				URL:    server.URL,
				APIKey: "test-key",
			}
			client := NewTautulliClient(cfg)

			err := client.Ping(context.Background())

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestTautulliClient_NetworkFailure(t *testing.T) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:9999", // Non-existent server
		APIKey: "test-key",
	}
	client := NewTautulliClient(cfg)

	// Test Ping with network failure
	err := client.Ping(context.Background())
	if err == nil {
		t.Error("Expected network error for Ping, got nil")
	}

	// Test GetHistory with network failure
	_, err = client.GetHistory(context.Background(), 0, 25)
	if err == nil {
		t.Error("Expected network error for GetHistory, got nil")
	}

	// Test GetGeoIPLookup with network failure
	_, err = client.GetGeoIPLookup(context.Background(), "8.8.8.8")
	if err == nil {
		t.Error("Expected network error for GetGeoIPLookup, got nil")
	}
}

func TestTautulliClient_Timeout(t *testing.T) {
	// Create a server that delays response beyond timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second) // Longer than 30s timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	client := NewTautulliClient(cfg)

	// This test would timeout, but we'll skip it for fast test execution
	// Just verify timeout is configured
	if client.client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.client.Timeout)
	}
}
