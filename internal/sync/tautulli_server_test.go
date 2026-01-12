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

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// mockServerResult represents a mock server setup for testing
type mockServerResult struct {
	server *httptest.Server
	client *TautulliClient
}

// setupMockServer creates a test server with the given handler and returns client
func setupMockServer(handler http.HandlerFunc) *mockServerResult {
	server := httptest.NewServer(handler)
	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)
	return &mockServerResult{server: server, client: client}
}

// close closes the mock server
func (m *mockServerResult) close() {
	m.server.Close()
}

// jsonHandler creates a handler that returns JSON-encoded data
func jsonHandler(data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(data)
	}
}

// errorHandler creates a handler that returns an HTTP error status
func errorHandler(status int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(message))
	}
}

// TestGetServerInfo tests the GetServerInfo endpoint using table-driven tests
func TestGetServerInfo(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantErr     bool
		errContains string
		validate    func(*testing.T, *tautulli.TautulliServerInfo)
	}{
		{
			name: "success",
			handler: jsonHandler(tautulli.TautulliServerInfo{
				Response: tautulli.TautulliServerInfoResponse{
					Result: "success",
					Data: tautulli.TautulliServerInfoData{
						PMSName:    "Plex Media Server",
						PMSVersion: "1.40.0.8395",
					},
				},
			}),
			wantErr: false,
			validate: func(t *testing.T, info *tautulli.TautulliServerInfo) {
				if info.Response.Data.PMSName != "Plex Media Server" {
					t.Errorf("PMSName = %q, want 'Plex Media Server'", info.Response.Data.PMSName)
				}
			},
		},
		{
			name:        "HTTP error",
			handler:     errorHandler(http.StatusServiceUnavailable, "Service temporarily unavailable"),
			wantErr:     true,
			errContains: "503",
		},
		{
			name: "API error with message",
			handler: jsonHandler(tautulli.TautulliServerInfo{
				Response: tautulli.TautulliServerInfoResponse{
					Result:  "error",
					Message: stringPtr("Server not configured"),
				},
			}),
			wantErr:     true,
			errContains: "Server not configured",
		},
		{
			name: "API error without message",
			handler: jsonHandler(tautulli.TautulliServerInfo{
				Response: tautulli.TautulliServerInfoResponse{
					Result:  "error",
					Message: nil,
				},
			}),
			wantErr:     true,
			errContains: "unknown error",
		},
		{
			name: "JSON decode error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{invalid json`))
			},
			wantErr:     true,
			errContains: "decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := setupMockServer(tt.handler)
			defer mock.close()

			info, err := mock.client.GetServerInfo(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain %q, got: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, info)
			}
		})
	}
}

// TestGetSyncedItems tests the GetSyncedItems endpoint
func TestGetSyncedItems(t *testing.T) {
	tests := []struct {
		name        string
		machineID   string
		userID      int
		handler     http.HandlerFunc
		wantErr     bool
		wantLen     int
		errContains string
	}{
		{
			name:      "success with machineID",
			machineID: "device123",
			userID:    0,
			handler: jsonHandler(tautulli.TautulliSyncedItems{
				Response: tautulli.TautulliSyncedItemsResponse{
					Result: "success",
					Data: []tautulli.TautulliSyncedItem{
						{SyncID: "1", RatingKey: "12345", SyncTitle: "Synced Movie", State: "complete"},
					},
				},
			}),
			wantErr: false,
			wantLen: 1,
		},
		{
			name:      "success with userID",
			machineID: "",
			userID:    42,
			handler: jsonHandler(tautulli.TautulliSyncedItems{
				Response: tautulli.TautulliSyncedItemsResponse{Result: "success", Data: []tautulli.TautulliSyncedItem{}},
			}),
			wantErr: false,
			wantLen: 0,
		},
		{
			name:      "API error",
			machineID: "invalid-device",
			userID:    0,
			handler: jsonHandler(tautulli.TautulliSyncedItems{
				Response: tautulli.TautulliSyncedItemsResponse{
					Result:  "error",
					Message: stringPtr("Device not found"),
				},
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := setupMockServer(tt.handler)
			defer mock.close()

			items, err := mock.client.GetSyncedItems(context.Background(), tt.machineID, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(items.Response.Data) != tt.wantLen {
				t.Errorf("len(Data) = %d, want %d", len(items.Response.Data), tt.wantLen)
			}
		})
	}

	t.Run("network error", func(t *testing.T) {
		cfg := &config.TautulliConfig{URL: "http://localhost:9999", APIKey: "test-key"}
		client := NewTautulliClient(cfg)

		_, err := client.GetSyncedItems(context.Background(), "device", 0)
		if err == nil {
			t.Fatal("Expected error for network failure")
		}
	})
}

// TestTerminateSession tests the TerminateSession endpoint
func TestTerminateSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		message     string
		handler     http.HandlerFunc
		wantErr     bool
		errContains string
	}{
		{
			name:      "success with message",
			sessionID: "session123",
			message:   "Server maintenance",
			handler: jsonHandler(tautulli.TautulliTerminateSession{
				Response: tautulli.TautulliTerminateSessionResponse{
					Result:  "success",
					Message: stringPtr("Session terminated successfully"),
				},
			}),
			wantErr: false,
		},
		{
			name:      "success without message",
			sessionID: "session456",
			message:   "",
			handler: jsonHandler(tautulli.TautulliTerminateSession{
				Response: tautulli.TautulliTerminateSessionResponse{Result: "success"},
			}),
			wantErr: false,
		},
		{
			name:      "API error",
			sessionID: "invalid-session",
			message:   "",
			handler: jsonHandler(tautulli.TautulliTerminateSession{
				Response: tautulli.TautulliTerminateSessionResponse{
					Result:  "error",
					Message: stringPtr("Session not found"),
				},
			}),
			wantErr:     true,
			errContains: "Session not found",
		},
		{
			name:      "JSON decode error",
			sessionID: "session",
			message:   "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`not valid json`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := setupMockServer(tt.handler)
			defer mock.close()

			result, err := mock.client.TerminateSession(context.Background(), tt.sessionID, tt.message)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain %q, got: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.Response.Result != "success" {
				t.Errorf("Result = %q, want 'success'", result.Response.Result)
			}
		})
	}
}

// TestGetStreamData tests the GetStreamData endpoint
func TestGetStreamData(t *testing.T) {
	tests := []struct {
		name        string
		rowID       int
		sessionKey  string
		handler     http.HandlerFunc
		wantErr     bool
		errContains string
		wantBitrate int
	}{
		{
			name:       "success with rowID",
			rowID:      123,
			sessionKey: "",
			handler: jsonHandler(tautulli.TautulliStreamData{
				Response: tautulli.TautulliStreamDataResponse{
					Result: "success",
					Data:   tautulli.TautulliStreamDataInfo{Bitrate: 15000, VideoCodec: "h264"},
				},
			}),
			wantErr:     false,
			wantBitrate: 15000,
		},
		{
			name:       "success with sessionKey",
			rowID:      0,
			sessionKey: "session-abc",
			handler: jsonHandler(tautulli.TautulliStreamData{
				Response: tautulli.TautulliStreamDataResponse{Result: "success", Data: tautulli.TautulliStreamDataInfo{}},
			}),
			wantErr: false,
		},
		{
			name:       "API error",
			rowID:      999,
			sessionKey: "",
			handler: jsonHandler(tautulli.TautulliStreamData{
				Response: tautulli.TautulliStreamDataResponse{
					Result:  "error",
					Message: stringPtr("Stream not found"),
				},
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := setupMockServer(tt.handler)
			defer mock.close()

			data, err := mock.client.GetStreamData(context.Background(), tt.rowID, tt.sessionKey)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.wantBitrate > 0 && data.Response.Data.Bitrate != tt.wantBitrate {
				t.Errorf("Bitrate = %d, want %d", data.Response.Data.Bitrate, tt.wantBitrate)
			}
		})
	}

	t.Run("error no parameters", func(t *testing.T) {
		cfg := &config.TautulliConfig{URL: "http://localhost", APIKey: "test-key"}
		client := NewTautulliClient(cfg)

		_, err := client.GetStreamData(context.Background(), 0, "")
		if err == nil {
			t.Fatal("Expected error when no parameters provided")
		}
		if !strings.Contains(err.Error(), "either row_id or session_key must be provided") {
			t.Errorf("Error should mention missing parameters, got: %v", err)
		}
	})
}

// TestGetUser tests the GetUser endpoint
func TestGetUser(t *testing.T) {
	tests := []struct {
		name     string
		userID   int
		handler  http.HandlerFunc
		wantErr  bool
		validate func(*testing.T, *tautulli.TautulliUser)
	}{
		{
			name:   "success",
			userID: 42,
			handler: jsonHandler(tautulli.TautulliUser{
				Response: tautulli.TautulliUserResponse{
					Result: "success",
					Data: tautulli.TautulliUserData{
						UserID: 42, Username: "testuser", FriendlyName: "Test User",
					},
				},
			}),
			wantErr: false,
			validate: func(t *testing.T, user *tautulli.TautulliUser) {
				if user.Response.Data.Username != "testuser" {
					t.Errorf("Username = %q, want 'testuser'", user.Response.Data.Username)
				}
			},
		},
		{
			name:   "API error",
			userID: 999,
			handler: jsonHandler(tautulli.TautulliUser{
				Response: tautulli.TautulliUserResponse{
					Result:  "error",
					Message: stringPtr("User not found"),
				},
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := setupMockServer(tt.handler)
			defer mock.close()

			user, err := mock.client.GetUser(context.Background(), tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, user)
			}
		})
	}
}

// TestGetUserPlayerStats tests the GetUserPlayerStats endpoint
func TestGetUserPlayerStats(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
		wantLen int
	}{
		{
			name: "success",
			handler: jsonHandler(tautulli.TautulliUserPlayerStats{
				Response: tautulli.TautulliUserPlayerStatsResponse{
					Result: "success",
					Data:   []tautulli.TautulliUserPlayerStatRow{{PlayerName: "Plex Web", TotalPlays: 150}},
				},
			}),
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "API error",
			handler: jsonHandler(tautulli.TautulliUserPlayerStats{
				Response: tautulli.TautulliUserPlayerStatsResponse{
					Result:  "error",
					Message: stringPtr("Invalid user"),
				},
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := setupMockServer(tt.handler)
			defer mock.close()

			stats, err := mock.client.GetUserPlayerStats(context.Background(), 42)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stats.Response.Data) != tt.wantLen {
				t.Errorf("len(Data) = %d, want %d", len(stats.Response.Data), tt.wantLen)
			}
		})
	}
}

// TestGetUserWatchTimeStats tests the GetUserWatchTimeStats endpoint
func TestGetUserWatchTimeStats(t *testing.T) {
	mock := setupMockServer(jsonHandler(tautulli.TautulliUserWatchTimeStats{
		Response: tautulli.TautulliUserWatchTimeStatsResponse{
			Result: "success",
			Data:   []tautulli.TautulliUserWatchTimeStatRow{{QueryDays: 30, TotalPlays: 50}},
		},
	}))
	defer mock.close()

	stats, err := mock.client.GetUserWatchTimeStats(context.Background(), 42, "30")
	if err != nil {
		t.Fatalf("GetUserWatchTimeStats() error = %v", err)
	}
	if len(stats.Response.Data) != 1 {
		t.Errorf("len(Data) = %d, want 1", len(stats.Response.Data))
	}
}

// TestGetUserIPs tests the GetUserIPs endpoint
func TestGetUserIPs(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
		wantLen int
	}{
		{
			name: "success",
			handler: jsonHandler(tautulli.TautulliUserIPs{
				Response: tautulli.TautulliUserIPsResponse{
					Result: "success",
					Data: []tautulli.TautulliUserIPData{
						{IPAddress: "192.168.1.100", PlayCount: 50},
						{IPAddress: "10.0.0.50", PlayCount: 25},
					},
				},
			}),
			wantErr: false,
			wantLen: 2,
		},
		{
			name:    "HTTP error",
			handler: errorHandler(http.StatusBadRequest, "Bad request"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := setupMockServer(tt.handler)
			defer mock.close()

			ips, err := mock.client.GetUserIPs(context.Background(), 42)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(ips.Response.Data) != tt.wantLen {
				t.Errorf("len(Data) = %d, want %d", len(ips.Response.Data), tt.wantLen)
			}
		})
	}
}

// TestGetUsersTable tests the GetUsersTable endpoint
func TestGetUsersTable(t *testing.T) {
	tests := []struct {
		name             string
		handler          http.HandlerFunc
		wantErr          bool
		errContains      string
		wantRecordsTotal int
	}{
		{
			name: "success",
			handler: jsonHandler(tautulli.TautulliUsersTable{
				Response: tautulli.TautulliUsersTableResponse{
					Result: "success",
					Data: tautulli.TautulliUsersTableData{
						RecordsTotal:    100,
						RecordsFiltered: 50,
						Data:            []tautulli.TautulliUsersTableRow{{UserID: 1, Username: "user1"}},
					},
				},
			}),
			wantErr:          false,
			wantRecordsTotal: 100,
		},
		{
			name: "API error",
			handler: jsonHandler(tautulli.TautulliUsersTable{
				Response: tautulli.TautulliUsersTableResponse{
					Result:  "error",
					Message: stringPtr("Database connection failed"),
				},
			}),
			wantErr:     true,
			errContains: "Database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := setupMockServer(tt.handler)
			defer mock.close()

			table, err := mock.client.GetUsersTable(context.Background(), 1, "friendly_name", "asc", 0, 25, "")

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain %q, got: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if table.Response.Data.RecordsTotal != tt.wantRecordsTotal {
				t.Errorf("RecordsTotal = %d, want %d", table.Response.Data.RecordsTotal, tt.wantRecordsTotal)
			}
		})
	}
}

// TestGetUserLogins tests the GetUserLogins endpoint
func TestGetUserLogins(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMockServer(jsonHandler(tautulli.TautulliUserLogins{
			Response: tautulli.TautulliUserLoginsResponse{
				Result: "success",
				Data: tautulli.TautulliUserLoginsData{
					RecordsTotal: 10,
					Data:         []tautulli.TautulliUserLoginsRow{{Timestamp: 1732483200, IPAddress: "192.168.1.100"}},
				},
			},
		}))
		defer mock.close()

		logins, err := mock.client.GetUserLogins(context.Background(), 42, "timestamp", "desc", 0, 25, "")
		if err != nil {
			t.Fatalf("GetUserLogins() error = %v", err)
		}
		if logins.Response.Data.RecordsTotal != 10 {
			t.Errorf("RecordsTotal = %d, want 10", logins.Response.Data.RecordsTotal)
		}
	})

	t.Run("network failure", func(t *testing.T) {
		cfg := &config.TautulliConfig{URL: "http://localhost:9999", APIKey: "test-key"}
		client := NewTautulliClient(cfg)

		_, err := client.GetUserLogins(context.Background(), 42, "", "", 0, 25, "")
		if err == nil {
			t.Fatal("Expected error for network failure")
		}
	})
}

// BenchmarkTautulliServerEndpoints benchmarks the GetServerInfo endpoint
func BenchmarkTautulliServerEndpoints(b *testing.B) {
	mock := setupMockServer(jsonHandler(tautulli.TautulliServerInfo{
		Response: tautulli.TautulliServerInfoResponse{
			Result: "success",
			Data:   tautulli.TautulliServerInfoData{PMSName: "Plex", PMSVersion: "1.40.0"},
		},
	}))
	defer mock.close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := mock.client.GetServerInfo(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}
