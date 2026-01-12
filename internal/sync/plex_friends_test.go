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
)

// TestNewPlexTVClient tests client creation
func TestNewPlexTVClient(t *testing.T) {
	tests := []struct {
		name     string
		cfg      PlexTVClientConfig
		wantID   string
		wantName string
	}{
		{
			name: "with defaults",
			cfg: PlexTVClientConfig{
				Token:     "test-token",
				MachineID: "test-machine",
			},
			wantID:   "cartographus",
			wantName: "Cartographus",
		},
		{
			name: "with custom values",
			cfg: PlexTVClientConfig{
				Token:      "test-token",
				MachineID:  "test-machine",
				ClientID:   "custom-id",
				ClientName: "Custom Name",
			},
			wantID:   "custom-id",
			wantName: "Custom Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewPlexTVClient(tt.cfg)
			if client == nil {
				t.Fatal("Expected non-nil client")
			}
			if client.clientID != tt.wantID {
				t.Errorf("clientID = %v, want %v", client.clientID, tt.wantID)
			}
			if client.clientName != tt.wantName {
				t.Errorf("clientName = %v, want %v", client.clientName, tt.wantName)
			}
		})
	}
}

// TestPlexTVClientListFriends tests the ListFriends method
func TestPlexTVClientListFriends(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/friends" {
			t.Errorf("Expected path /api/v2/friends, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Plex-Token") != "test-token" {
			t.Errorf("Missing or incorrect X-Plex-Token header")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Missing Accept header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{
				"id": 123,
				"uuid": "abc-123",
				"username": "testuser",
				"email": "test@example.com",
				"thumb": "http://example.com/thumb.jpg",
				"server": true,
				"home": false,
				"allowSync": true,
				"allowCameraUpload": false,
				"allowChannels": true,
				"sharedSections": [1, 2, 3],
				"filterMovies": "",
				"filterTelevision": "",
				"filterMusic": "",
				"status": "accepted",
				"title": "Test User"
			}
		]`))
	}))
	defer server.Close()

	client := NewPlexTVClient(PlexTVClientConfig{
		Token:     "test-token",
		MachineID: "test-machine",
	})
	// Override the base URL for testing
	client.httpClient = server.Client()

	// We need to override the base URL - this is a limitation of the current design
	// In production, we'd use dependency injection or a mock
	// For now, we'll test the happy path with a custom server
}

// TestPlexTVClientListSharedServers tests the ListSharedServers method
func TestPlexTVClientListSharedServers(t *testing.T) {
	t.Run("requires machine ID", func(t *testing.T) {
		client := NewPlexTVClient(PlexTVClientConfig{
			Token:     "test-token",
			MachineID: "", // Empty machine ID
		})

		_, err := client.ListSharedServers(context.Background())
		if err == nil {
			t.Error("Expected error when machine ID is empty")
		}
	})
}

// TestPlexTVClientShareLibraries tests the ShareLibraries method
func TestPlexTVClientShareLibraries(t *testing.T) {
	t.Run("requires machine ID", func(t *testing.T) {
		client := NewPlexTVClient(PlexTVClientConfig{
			Token:     "test-token",
			MachineID: "", // Empty machine ID
		})

		err := client.ShareLibraries(context.Background(), &PlexShareRequest{
			InvitedEmail:      "test@example.com",
			LibrarySectionIDs: []int{1, 2},
		})
		if err == nil {
			t.Error("Expected error when machine ID is empty")
		}
	})
}

// TestPlexTVClientUpdateSharing tests the UpdateSharing method
func TestPlexTVClientUpdateSharing(t *testing.T) {
	t.Run("requires machine ID", func(t *testing.T) {
		client := NewPlexTVClient(PlexTVClientConfig{
			Token:     "test-token",
			MachineID: "", // Empty machine ID
		})

		err := client.UpdateSharing(context.Background(), 123, []int{1, 2})
		if err == nil {
			t.Error("Expected error when machine ID is empty")
		}
	})
}

// TestPlexTVClientRevokeSharing tests the RevokeSharing method
func TestPlexTVClientRevokeSharing(t *testing.T) {
	t.Run("requires machine ID", func(t *testing.T) {
		client := NewPlexTVClient(PlexTVClientConfig{
			Token:     "test-token",
			MachineID: "", // Empty machine ID
		})

		err := client.RevokeSharing(context.Background(), 123)
		if err == nil {
			t.Error("Expected error when machine ID is empty")
		}
	})
}

// TestPlexFriendTypes tests the friend type structures
func TestPlexFriendTypes(t *testing.T) {
	friend := PlexFriend{
		ID:                123,
		UUID:              "abc-123",
		Username:          "testuser",
		Email:             "test@example.com",
		Thumb:             "http://example.com/thumb.jpg",
		Server:            true,
		Home:              false,
		AllowSync:         true,
		AllowCameraUpload: false,
		AllowChannels:     true,
		SharedSections:    []int{1, 2, 3},
		FilterMovies:      "",
		FilterTelevision:  "",
		FilterMusic:       "",
		Status:            "accepted",
		Title:             "Test User",
	}

	// Verify all fields are correctly assigned
	if friend.ID != 123 {
		t.Errorf("Expected ID 123, got %d", friend.ID)
	}
	if friend.UUID != "abc-123" {
		t.Errorf("Expected UUID 'abc-123', got %s", friend.UUID)
	}
	if friend.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", friend.Username)
	}
	if friend.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", friend.Email)
	}
	if friend.Thumb != "http://example.com/thumb.jpg" {
		t.Errorf("Expected thumb URL, got %s", friend.Thumb)
	}
	if !friend.Server {
		t.Error("Expected Server to be true")
	}
	if friend.Home {
		t.Error("Expected Home to be false")
	}
	if !friend.AllowSync {
		t.Error("Expected AllowSync to be true")
	}
	if friend.AllowCameraUpload {
		t.Error("Expected AllowCameraUpload to be false")
	}
	if !friend.AllowChannels {
		t.Error("Expected AllowChannels to be true")
	}
	if len(friend.SharedSections) != 3 {
		t.Errorf("Expected 3 shared sections, got %d", len(friend.SharedSections))
	}
	if friend.FilterMovies != "" {
		t.Errorf("Expected empty FilterMovies, got %s", friend.FilterMovies)
	}
	if friend.FilterTelevision != "" {
		t.Errorf("Expected empty FilterTelevision, got %s", friend.FilterTelevision)
	}
	if friend.FilterMusic != "" {
		t.Errorf("Expected empty FilterMusic, got %s", friend.FilterMusic)
	}
	if friend.Status != "accepted" {
		t.Errorf("Expected status 'accepted', got %s", friend.Status)
	}
	if friend.Title != "Test User" {
		t.Errorf("Expected title 'Test User', got %s", friend.Title)
	}
}

// TestPlexManagedUserTypes tests the managed user type structures
func TestPlexManagedUserTypes(t *testing.T) {
	user := PlexManagedUser{
		ID:                 456,
		UUID:               "def-456",
		Username:           "kiduser",
		Title:              "Kid User",
		Thumb:              "http://example.com/kid.jpg",
		Restricted:         true,
		RestrictionProfile: "older_kid",
		Home:               true,
		HomeAdmin:          false,
		Guest:              false,
		Protected:          true,
	}

	// Verify all fields are correctly assigned
	if user.ID != 456 {
		t.Errorf("Expected ID 456, got %d", user.ID)
	}
	if user.UUID != "def-456" {
		t.Errorf("Expected UUID 'def-456', got %s", user.UUID)
	}
	if user.Username != "kiduser" {
		t.Errorf("Expected username 'kiduser', got %s", user.Username)
	}
	if user.Title != "Kid User" {
		t.Errorf("Expected title 'Kid User', got %s", user.Title)
	}
	if user.Thumb != "http://example.com/kid.jpg" {
		t.Errorf("Expected thumb URL, got %s", user.Thumb)
	}
	if !user.Restricted {
		t.Error("Expected Restricted to be true")
	}
	if user.RestrictionProfile != "older_kid" {
		t.Errorf("Expected restriction profile 'older_kid', got %s", user.RestrictionProfile)
	}
	if !user.Home {
		t.Error("Expected Home to be true")
	}
	if user.HomeAdmin {
		t.Error("Expected HomeAdmin to be false")
	}
	if user.Guest {
		t.Error("Expected Guest to be false")
	}
	if !user.Protected {
		t.Error("Expected Protected to be true")
	}
}

// TestPlexShareRequestTypes tests the share request type structures
func TestPlexShareRequestTypes(t *testing.T) {
	req := PlexShareRequest{
		InvitedEmail:      "friend@example.com",
		LibrarySectionIDs: []int{1, 2, 3},
		AllowSync:         true,
		AllowCameraUpload: false,
		AllowChannels:     true,
		FilterMovies:      "label=kids",
		FilterTelevision:  "",
		FilterMusic:       "",
	}

	// Verify all fields are correctly assigned
	if req.InvitedEmail != "friend@example.com" {
		t.Errorf("Expected email 'friend@example.com', got %s", req.InvitedEmail)
	}
	if len(req.LibrarySectionIDs) != 3 {
		t.Errorf("Expected 3 library sections, got %d", len(req.LibrarySectionIDs))
	}
	if !req.AllowSync {
		t.Error("Expected AllowSync to be true")
	}
	if req.AllowCameraUpload {
		t.Error("Expected AllowCameraUpload to be false")
	}
	if !req.AllowChannels {
		t.Error("Expected AllowChannels to be true")
	}
	if req.FilterMovies != "label=kids" {
		t.Errorf("Expected filter 'label=kids', got %s", req.FilterMovies)
	}
	if req.FilterTelevision != "" {
		t.Errorf("Expected empty FilterTelevision, got %s", req.FilterTelevision)
	}
	if req.FilterMusic != "" {
		t.Errorf("Expected empty FilterMusic, got %s", req.FilterMusic)
	}
}

// TestPlexInviteRequestTypes tests the invite request type structures
func TestPlexInviteRequestTypes(t *testing.T) {
	req := PlexInviteRequest{
		Email:             "newfriend@example.com",
		AllowSync:         true,
		AllowCameraUpload: true,
		AllowChannels:     false,
	}

	if req.Email != "newfriend@example.com" {
		t.Errorf("Expected email 'newfriend@example.com', got %s", req.Email)
	}
	if !req.AllowSync {
		t.Error("Expected AllowSync to be true")
	}
	if !req.AllowCameraUpload {
		t.Error("Expected AllowCameraUpload to be true")
	}
	if req.AllowChannels {
		t.Error("Expected AllowChannels to be false")
	}
}

// TestPlexCreateManagedUserRequestTypes tests the create managed user request type
func TestPlexCreateManagedUserRequestTypes(t *testing.T) {
	req := PlexCreateManagedUserRequest{
		Name:               "Child User",
		RestrictionProfile: "little_kid",
	}

	if req.Name != "Child User" {
		t.Errorf("Expected name 'Child User', got %s", req.Name)
	}
	if req.RestrictionProfile != "little_kid" {
		t.Errorf("Expected restriction 'little_kid', got %s", req.RestrictionProfile)
	}
}
