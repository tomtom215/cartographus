// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestTautulliUser_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetUserFunc: func(ctx context.Context, userID int) (*tautulli.TautulliUser, error) {
			return &tautulli.TautulliUser{
				Response: tautulli.TautulliUserResponse{
					Result: "success",
					Data: tautulli.TautulliUserData{
						UserID:   userID,
						Username: "testuser",
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/user?user_id=123", nil)
	w := httptest.NewRecorder()

	handler.TautulliUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestTautulliUsers_Success tests the bulk user fetch endpoint (get_users)
func TestTautulliUsers_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetUsersFunc: func(ctx context.Context) (*tautulli.TautulliUsers, error) {
			return &tautulli.TautulliUsers{
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
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/users", nil)
	w := httptest.NewRecorder()

	handler.TautulliUsers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify response contains user data
	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	// Data should be an array of users
	dataSlice, ok := response.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array, got %T", response.Data)
	}

	if len(dataSlice) != 2 {
		t.Errorf("Expected 2 users, got %d", len(dataSlice))
	}
}

// TestTautulliUsers_EmptyResult tests when no users are returned
func TestTautulliUsers_EmptyResult(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetUsersFunc: func(ctx context.Context) (*tautulli.TautulliUsers, error) {
			return &tautulli.TautulliUsers{
				Response: tautulli.TautulliUsersResponse{
					Result: "success",
					Data:   []tautulli.TautulliUserData{},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/users", nil)
	w := httptest.NewRecorder()

	handler.TautulliUsers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliUserPlayerStats_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetUserPlayerStatsFunc: func(ctx context.Context, userID int) (*tautulli.TautulliUserPlayerStats, error) {
			return &tautulli.TautulliUserPlayerStats{
				Response: tautulli.TautulliUserPlayerStatsResponse{
					Result: "success",
					Data: []tautulli.TautulliUserPlayerStatRow{
						{PlayerName: "Chrome", TotalPlays: 100, PlatformName: "Chrome"},
						{PlayerName: "Roku", TotalPlays: 50, PlatformName: "Roku"},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/user-player-stats?user_id=1", nil)
	w := httptest.NewRecorder()

	handler.TautulliUserPlayerStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliUserWatchTimeStats_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetUserWatchTimeStatsFunc: func(ctx context.Context, userID int, queryDays string) (*tautulli.TautulliUserWatchTimeStats, error) {
			return &tautulli.TautulliUserWatchTimeStats{
				Response: tautulli.TautulliUserWatchTimeStatsResponse{
					Result: "success",
					Data: []tautulli.TautulliUserWatchTimeStatRow{
						{QueryDays: 7, TotalTime: 3600, TotalPlays: 10},
						{QueryDays: 30, TotalTime: 14400, TotalPlays: 40},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/user-watch-time-stats?user_id=1", nil)
	w := httptest.NewRecorder()

	handler.TautulliUserWatchTimeStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliItemUserStats_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetItemUserStatsFunc: func(ctx context.Context, ratingKey string, grouping int) (*tautulli.TautulliItemUserStats, error) {
			return &tautulli.TautulliItemUserStats{
				Response: tautulli.TautulliItemUserStatsResponse{
					Result: "success",
					Data: []tautulli.TautulliItemUserStatRow{
						{UserID: 1, Username: "user1", TotalPlays: 5},
						{UserID: 2, Username: "user2", TotalPlays: 3},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/item-user-stats?rating_key=12345", nil)
	w := httptest.NewRecorder()

	handler.TautulliItemUserStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Priority 2: Library-Specific Analytics - Handler Tests (4 endpoints)

func TestTautulliUserIPs_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetUserIPsFunc: func(ctx context.Context, userID int) (*tautulli.TautulliUserIPs, error) {
			return &tautulli.TautulliUserIPs{
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
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/user-ips?user_id=1", nil)
	w := httptest.NewRecorder()

	handler.TautulliUserIPs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

func TestTautulliUsersTable_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetUsersTableFunc: func(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUsersTable, error) {
			return &tautulli.TautulliUsersTable{
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
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/users-table?start=0&length=10", nil)
	w := httptest.NewRecorder()

	handler.TautulliUsersTable(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}
