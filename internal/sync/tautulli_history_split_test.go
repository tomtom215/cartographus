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
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestTautulliClient_GetHistory(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		if r.URL.Query().Get("cmd") != "get_history" {
			t.Errorf("Expected cmd=get_history, got %s", r.URL.Query().Get("cmd"))
		}
		if r.URL.Query().Get("start") != "0" {
			t.Errorf("Expected start=0, got %s", r.URL.Query().Get("start"))
		}
		if r.URL.Query().Get("length") != "25" {
			t.Errorf("Expected length=25, got %s", r.URL.Query().Get("length"))
		}
		if r.URL.Query().Get("order_column") != "started" {
			t.Errorf("Expected order_column=started, got %s", r.URL.Query().Get("order_column"))
		}
		if r.URL.Query().Get("order_dir") != "desc" {
			t.Errorf("Expected order_dir=desc, got %s", r.URL.Query().Get("order_dir"))
		}

		// Return mock history response
		response := tautulli.TautulliHistory{
			Response: tautulli.TautulliHistoryResponse{
				Result: "success",
				Data: tautulli.TautulliHistoryData{
					RecordsTotal: 1,
					Data: []tautulli.TautulliHistoryRecord{
						{
							SessionKey:      stringPtr("test-session-1"),
							Started:         time.Now().Unix(),
							Stopped:         time.Now().Unix() + 3600,
							UserID:          intPtr(1),
							User:            "testuser",
							IPAddress:       "192.168.1.100",
							MediaType:       "movie",
							Title:           "Test Movie",
							Platform:        "Chrome",
							Player:          "Plex Web",
							Location:        "lan",
							PercentComplete: intPtr(95),
						},
					},
				},
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	client := NewTautulliClient(cfg)

	history, err := client.GetHistory(context.Background(), 0, 25)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if history == nil {
		t.Fatal("Expected non-nil history")
	}

	if history.Response.Result != "success" {
		t.Errorf("Expected result=success, got %s", history.Response.Result)
	}

	if len(history.Response.Data.Data) != 1 {
		t.Errorf("Expected 1 record, got %d", len(history.Response.Data.Data))
	}

	record := history.Response.Data.Data[0]
	if record.SessionKey == nil || *record.SessionKey != "test-session-1" {
		t.Errorf("Expected session key test-session-1, got %v", record.SessionKey)
	}
	if record.User != "testuser" {
		t.Errorf("Expected user testuser, got %s", record.User)
	}
}

func TestTautulliClient_GetHistorySince(t *testing.T) {
	sinceTime := time.Now().Add(-24 * time.Hour)

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		if r.URL.Query().Get("cmd") != "get_history" {
			t.Errorf("Expected cmd=get_history, got %s", r.URL.Query().Get("cmd"))
		}
		if r.URL.Query().Get("after") == "" {
			t.Error("Expected after parameter to be set")
		}

		// Return mock response
		response := tautulli.TautulliHistory{
			Response: tautulli.TautulliHistoryResponse{
				Result: "success",
				Data: tautulli.TautulliHistoryData{
					RecordsTotal: 2,
					Data: []tautulli.TautulliHistoryRecord{
						{
							SessionKey: stringPtr("session-1"),
							Started:    time.Now().Unix(),
							User:       "user1",
							IPAddress:  "10.0.0.1",
							MediaType:  "episode",
							Title:      "Episode 1",
						},
						{
							SessionKey: stringPtr("session-2"),
							Started:    time.Now().Unix() - 3600,
							User:       "user2",
							IPAddress:  "10.0.0.2",
							MediaType:  "movie",
							Title:      "Movie 1",
						},
					},
				},
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	client := NewTautulliClient(cfg)

	history, err := client.GetHistorySince(context.Background(), sinceTime, 0, 50)
	if err != nil {
		t.Fatalf("GetHistorySince failed: %v", err)
	}

	if len(history.Response.Data.Data) != 2 {
		t.Errorf("Expected 2 records, got %d", len(history.Response.Data.Data))
	}
}

func TestTautulliClient_GeoIPLookup_Errors(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:          "HTTP 500 error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  "Internal Server Error",
			expectError:   true,
			errorContains: "geoip request failed with status 500",
		},
		{
			name:       "API failure response",
			statusCode: http.StatusOK,
			responseBody: tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result:  "error",
					Message: stringPtr("Invalid IP address"),
				},
			},
			expectError:   true,
			errorContains: "geoip lookup failed",
		},
		{
			name:          "Invalid JSON response",
			statusCode:    http.StatusOK,
			responseBody:  "invalid json {",
			expectError:   true,
			errorContains: "failed to decode geoip response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if str, ok := tt.responseBody.(string); ok {
					w.Write([]byte(str))
				} else {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{
				URL:    server.URL,
				APIKey: "test-key",
			}
			client := NewTautulliClient(cfg)

			_, err := client.GetGeoIPLookup(context.Background(), "8.8.8.8")

			if !tt.expectError {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
			}
		})
	}
}

func TestTautulliClient_GetHistory_Errors(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:          "HTTP 401 unauthorized",
			statusCode:    http.StatusUnauthorized,
			responseBody:  "Unauthorized",
			expectError:   true,
			errorContains: "history request failed with status 401",
		},
		{
			name:       "API error response",
			statusCode: http.StatusOK,
			responseBody: tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result:  "error",
					Message: stringPtr("Database connection failed"),
				},
			},
			expectError:   true,
			errorContains: "history request failed",
		},
		{
			name:          "Invalid JSON",
			statusCode:    http.StatusOK,
			responseBody:  "not valid json",
			expectError:   true,
			errorContains: "failed to decode history response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if str, ok := tt.responseBody.(string); ok {
					w.Write([]byte(str))
				} else {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{
				URL:    server.URL,
				APIKey: "test-key",
			}
			client := NewTautulliClient(cfg)

			_, err := client.GetHistory(context.Background(), 0, 25)

			if !tt.expectError {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
			}
		})
	}
}

// Note: stringPtr helper is defined in test_helpers.go, shared across test files in this package
// String comparison now uses standard library strings.Contains
