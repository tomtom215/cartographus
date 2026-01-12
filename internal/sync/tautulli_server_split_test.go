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

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestTautulliClient_GetServerInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_server_info" {
			t.Errorf("Expected cmd=get_server_info, got %s", r.URL.Query().Get("cmd"))
		}
		mockData := tautulli.TautulliServerInfo{
			Response: tautulli.TautulliServerInfoResponse{
				Result: "success",
				Data: tautulli.TautulliServerInfoData{
					PMSName:            "Test Server",
					PMSVersion:         "1.40.0.7775",
					PMSPlatform:        "Linux",
					PMSPlatformVersion: "6.5.0",
				},
			},
		}
		json.NewEncoder(w).Encode(mockData)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	info, err := client.GetServerInfo(context.Background())
	if err != nil {
		t.Errorf("GetServerInfo() error = %v", err)
	}
	if info.Response.Data.PMSName != "Test Server" {
		t.Errorf("Expected plex_name='Test Server', got '%s'", info.Response.Data.PMSName)
	}
}

func TestTautulliClient_GetSyncedItems(t *testing.T) {
	tests := []struct {
		name        string
		machineID   string
		userID      int
		expectError bool
	}{
		{
			name:        "filter by machine ID",
			machineID:   "device123",
			userID:      0,
			expectError: false,
		},
		{
			name:        "filter by user ID",
			machineID:   "",
			userID:      5,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cmd") != "get_synced_items" {
					t.Errorf("Expected cmd=get_synced_items, got %s", r.URL.Query().Get("cmd"))
				}
				mockData := tautulli.TautulliSyncedItems{
					Response: tautulli.TautulliSyncedItemsResponse{
						Result: "success",
						Data:   []tautulli.TautulliSyncedItem{},
					},
				}
				json.NewEncoder(w).Encode(mockData)
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
			client := NewTautulliClient(cfg)

			items, err := client.GetSyncedItems(context.Background(), tt.machineID, tt.userID)
			if (err != nil) != tt.expectError {
				t.Errorf("GetSyncedItems() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && items == nil {
				t.Error("Expected non-nil items response")
			}
		})
	}
}

func TestTautulliClient_TerminateSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		message     string
		expectError bool
	}{
		{
			name:        "terminate with message",
			sessionID:   "session123",
			message:     "Server maintenance",
			expectError: false,
		},
		{
			name:        "terminate without message",
			sessionID:   "session456",
			message:     "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cmd") != "terminate_session" {
					t.Errorf("Expected cmd=terminate_session, got %s", r.URL.Query().Get("cmd"))
				}
				if r.URL.Query().Get("session_id") != tt.sessionID {
					t.Errorf("Expected session_id=%s, got %s", tt.sessionID, r.URL.Query().Get("session_id"))
				}
				mockData := tautulli.TautulliTerminateSession{
					Response: tautulli.TautulliTerminateSessionResponse{
						Result:  "success",
						Message: stringPtr("Session terminated"),
					},
				}
				json.NewEncoder(w).Encode(mockData)
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
			client := NewTautulliClient(cfg)

			result, err := client.TerminateSession(context.Background(), tt.sessionID, tt.message)
			if (err != nil) != tt.expectError {
				t.Errorf("TerminateSession() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && result.Response.Result != "success" {
				t.Errorf("Expected result='success', got '%s'", result.Response.Result)
			}
		})
	}
}

// Note: Helper functions (stringPtr, contains, indexContains) are defined in test_helpers.go
// and geolocation_validation_test.go, shared across all test files in this package
