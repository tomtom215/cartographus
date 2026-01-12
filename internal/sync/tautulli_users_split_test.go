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

func TestTautulliClient_GetUser(t *testing.T) {
	tests := []struct {
		name        string
		userID      int
		expectError bool
	}{
		{
			name:        "valid user ID",
			userID:      123,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cmd") != "get_user" {
					t.Errorf("Expected cmd=get_user, got %s", r.URL.Query().Get("cmd"))
				}
				mockData := tautulli.TautulliUser{
					Response: tautulli.TautulliUserResponse{
						Result: "success",
						Data: tautulli.TautulliUserData{
							UserID:       tt.userID,
							Username:     "testuser",
							FriendlyName: "Test User",
							Email:        "test@example.com",
							IsHomeUser:   1,
						},
					},
				}
				json.NewEncoder(w).Encode(mockData)
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
			client := NewTautulliClient(cfg)

			user, err := client.GetUser(context.Background(), tt.userID)
			if (err != nil) != tt.expectError {
				t.Errorf("GetUser() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError && user.Response.Data.Username != "testuser" {
				t.Errorf("Expected username='testuser', got '%s'", user.Response.Data.Username)
			}
		})
	}
}

// TestTautulliClient_GetUsers tests the bulk user fetch endpoint
func TestTautulliClient_GetUsers(t *testing.T) {
	tests := []struct {
		name          string
		response      tautulli.TautulliUsers
		expectedCount int
		expectError   bool
	}{
		{
			name: "multiple users",
			response: tautulli.TautulliUsers{
				Response: tautulli.TautulliUsersResponse{
					Result: "success",
					Data: []tautulli.TautulliUserData{
						{
							UserID:       1,
							Username:     "user1",
							FriendlyName: "User One",
							Email:        "user1@example.com",
						},
						{
							UserID:       2,
							Username:     "user2",
							FriendlyName: "User Two",
							Email:        "user2@example.com",
						},
					},
				},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "empty user list",
			response: tautulli.TautulliUsers{
				Response: tautulli.TautulliUsersResponse{
					Result: "success",
					Data:   []tautulli.TautulliUserData{},
				},
			},
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cmd") != "get_users" {
					t.Errorf("Expected cmd=get_users, got %s", r.URL.Query().Get("cmd"))
				}
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
			client := NewTautulliClient(cfg)

			users, err := client.GetUsers(context.Background())
			if (err != nil) != tt.expectError {
				t.Fatalf("GetUsers() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError {
				if users.Response.Result != "success" {
					t.Errorf("Expected result='success', got '%s'", users.Response.Result)
				}
				if len(users.Response.Data) != tt.expectedCount {
					t.Errorf("Expected %d users, got %d", tt.expectedCount, len(users.Response.Data))
				}
			}
		})
	}
}

func TestTautulliClient_GetUserPlayerStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_user_player_stats" {
			t.Errorf("Expected cmd=get_user_player_stats, got %s", r.URL.Query().Get("cmd"))
		}
		if r.URL.Query().Get("user_id") != "1" {
			t.Errorf("Expected user_id=1, got %s", r.URL.Query().Get("user_id"))
		}

		response := tautulli.TautulliUserPlayerStats{
			Response: tautulli.TautulliUserPlayerStatsResponse{
				Result: "success",
				Data: []tautulli.TautulliUserPlayerStatRow{
					{PlayerName: "Chrome", TotalPlays: 100, PlatformName: "Chrome"},
					{PlayerName: "Roku", TotalPlays: 50, PlatformName: "Roku"},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetUserPlayerStats(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetUserPlayerStats() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data) != 2 {
		t.Errorf("Expected 2 players, got %d", len(stats.Response.Data))
	}
}

func TestTautulliClient_GetUserWatchTimeStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_user_watch_time_stats" {
			t.Errorf("Expected cmd=get_user_watch_time_stats, got %s", r.URL.Query().Get("cmd"))
		}
		if r.URL.Query().Get("user_id") != "1" {
			t.Errorf("Expected user_id=1, got %s", r.URL.Query().Get("user_id"))
		}

		response := tautulli.TautulliUserWatchTimeStats{
			Response: tautulli.TautulliUserWatchTimeStatsResponse{
				Result: "success",
				Data: []tautulli.TautulliUserWatchTimeStatRow{
					{QueryDays: 7, TotalTime: 3600, TotalPlays: 10},
					{QueryDays: 30, TotalTime: 14400, TotalPlays: 40},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetUserWatchTimeStats(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("GetUserWatchTimeStats() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data) != 2 {
		t.Errorf("Expected 2 time periods, got %d", len(stats.Response.Data))
	}
}

func TestTautulliClient_GetUserIPs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_user_ips" {
			t.Errorf("Expected cmd=get_user_ips, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliUserIPs{
			Response: tautulli.TautulliUserIPsResponse{
				Result: "success",
				Data: []tautulli.TautulliUserIPData{
					{
						UserID:       1,
						FriendlyName: "Test User",
						IPAddress:    "192.168.1.100",
						LastSeen:     1700000000,
						LastPlayed:   "2023-11-15",
						PlayCount:    25,
						PlatformName: "Chrome",
						PlayerName:   "Plex Web",
					},
					{
						UserID:       1,
						FriendlyName: "Test User",
						IPAddress:    "10.0.0.5",
						LastSeen:     1699000000,
						LastPlayed:   "2023-11-10",
						PlayCount:    10,
						PlatformName: "Android",
						PlayerName:   "Plex for Android",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	userIPs, err := client.GetUserIPs(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetUserIPs() error = %v", err)
	}
	if userIPs.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", userIPs.Response.Result)
	}
	if len(userIPs.Response.Data) != 2 {
		t.Errorf("Expected 2 IP records, got %d", len(userIPs.Response.Data))
	}
	if userIPs.Response.Data[0].IPAddress != "192.168.1.100" {
		t.Errorf("Expected IP='192.168.1.100', got '%s'", userIPs.Response.Data[0].IPAddress)
	}
	if userIPs.Response.Data[0].PlayCount != 25 {
		t.Errorf("Expected PlayCount=25, got %d", userIPs.Response.Data[0].PlayCount)
	}
}

func TestTautulliClient_GetUsersTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_users_table" {
			t.Errorf("Expected cmd=get_users_table, got %s", r.URL.Query().Get("cmd"))
		}

		// Verify pagination parameters
		if r.URL.Query().Get("start") != "0" {
			t.Errorf("Expected start=0, got %s", r.URL.Query().Get("start"))
		}
		if r.URL.Query().Get("length") != "10" {
			t.Errorf("Expected length=10, got %s", r.URL.Query().Get("length"))
		}

		response := tautulli.TautulliUsersTable{
			Response: tautulli.TautulliUsersTableResponse{
				Result: "success",
				Data: tautulli.TautulliUsersTableData{
					RecordsTotal:    100,
					RecordsFiltered: 50,
					Draw:            1,
					Data: []tautulli.TautulliUsersTableRow{
						{
							UserID:       1,
							Username:     "testuser",
							FriendlyName: "Test User",
							UserThumb:    "/thumb/user1.jpg",
							Plays:        150,
							Duration:     36000,
							LastSeen:     1700000000,
							LastPlayed:   "2023-11-15",
							IPAddress:    "192.168.1.100",
							PlatformName: "Chrome",
							PlayerName:   "Plex Web",
						},
						{
							UserID:       2,
							Username:     "user2",
							FriendlyName: "User Two",
							UserThumb:    "/thumb/user2.jpg",
							Plays:        75,
							Duration:     18000,
							LastSeen:     1699000000,
							LastPlayed:   "2023-11-10",
							IPAddress:    "10.0.0.5",
							PlatformName: "Android",
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	usersTable, err := client.GetUsersTable(context.Background(), 0, "last_seen", "desc", 0, 10, "")
	if err != nil {
		t.Fatalf("GetUsersTable() error = %v", err)
	}
	if usersTable.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", usersTable.Response.Result)
	}
	if usersTable.Response.Data.RecordsTotal != 100 {
		t.Errorf("Expected RecordsTotal=100, got %d", usersTable.Response.Data.RecordsTotal)
	}
	if len(usersTable.Response.Data.Data) != 2 {
		t.Errorf("Expected 2 users, got %d", len(usersTable.Response.Data.Data))
	}
	if usersTable.Response.Data.Data[0].Username != "testuser" {
		t.Errorf("Expected username='testuser', got '%s'", usersTable.Response.Data.Data[0].Username)
	}
	if usersTable.Response.Data.Data[0].Plays != 150 {
		t.Errorf("Expected Plays=150, got %d", usersTable.Response.Data.Data[0].Plays)
	}
}

func TestTautulliClient_GetUserLogins(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_user_logins" {
			t.Errorf("Expected cmd=get_user_logins, got %s", r.URL.Query().Get("cmd"))
		}

		// Verify user_id parameter
		if r.URL.Query().Get("user_id") != "1" {
			t.Errorf("Expected user_id=1, got %s", r.URL.Query().Get("user_id"))
		}

		response := tautulli.TautulliUserLogins{
			Response: tautulli.TautulliUserLoginsResponse{
				Result: "success",
				Data: tautulli.TautulliUserLoginsData{
					RecordsTotal:    500,
					RecordsFiltered: 250,
					Draw:            1,
					Data: []tautulli.TautulliUserLoginsRow{
						{
							Timestamp:    1700000000,
							Time:         "2023-11-15 10:00:00",
							UserID:       1,
							Username:     "testuser",
							FriendlyName: "Test User",
							IPAddress:    "192.168.1.100",
							Host:         "home-network",
							UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
							OS:           "Windows 10",
							Browser:      "Chrome 119",
							Success:      1,
						},
						{
							Timestamp:    1699000000,
							Time:         "2023-11-10 15:30:00",
							UserID:       1,
							Username:     "testuser",
							FriendlyName: "Test User",
							IPAddress:    "10.0.0.5",
							Host:         "mobile-network",
							UserAgent:    "Plex/8.5.0 (Android 13)",
							OS:           "Android",
							Browser:      "Plex App",
							Success:      1,
						},
						{
							Timestamp:    1698000000,
							Time:         "2023-11-05 08:00:00",
							UserID:       1,
							Username:     "testuser",
							FriendlyName: "Test User",
							IPAddress:    "203.0.113.50",
							Host:         "unknown",
							UserAgent:    "curl/7.68.0",
							OS:           "Linux",
							Browser:      "curl",
							Success:      0,
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	userLogins, err := client.GetUserLogins(context.Background(), 1, "timestamp", "desc", 0, 10, "")
	if err != nil {
		t.Fatalf("GetUserLogins() error = %v", err)
	}
	if userLogins.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", userLogins.Response.Result)
	}
	if userLogins.Response.Data.RecordsTotal != 500 {
		t.Errorf("Expected RecordsTotal=500, got %d", userLogins.Response.Data.RecordsTotal)
	}
	if len(userLogins.Response.Data.Data) != 3 {
		t.Errorf("Expected 3 login records, got %d", len(userLogins.Response.Data.Data))
	}
	// Test successful login
	if userLogins.Response.Data.Data[0].Success != 1 {
		t.Errorf("Expected Success=1 for first record, got %d", userLogins.Response.Data.Data[0].Success)
	}
	if userLogins.Response.Data.Data[0].OS != "Windows 10" {
		t.Errorf("Expected OS='Windows 10', got '%s'", userLogins.Response.Data.Data[0].OS)
	}
	// Test failed login
	if userLogins.Response.Data.Data[2].Success != 0 {
		t.Errorf("Expected Success=0 for failed login, got %d", userLogins.Response.Data.Data[2].Success)
	}
	if userLogins.Response.Data.Data[2].Browser != "curl" {
		t.Errorf("Expected Browser='curl', got '%s'", userLogins.Response.Data.Data[2].Browser)
	}
}
