// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
plex_friends.go - Plex.tv Friends and Sharing API Client

This file provides the PlexTVClient for managing Plex friends, library sharing,
and managed users through the plex.tv API endpoints.

Features:
  - List/invite/remove friends
  - Manage library sharing with friends
  - List/create/delete managed users (Plex Home)
  - Update content restrictions for managed users

API Endpoints:
  - GET  https://plex.tv/api/v2/friends - List friends
  - POST https://plex.tv/api/v2/friends/invite - Send friend invite
  - DELETE https://plex.tv/api/v2/friends/{id} - Remove friend
  - POST https://plex.tv/api/servers/{machineId}/shared_servers - Share libraries
  - DELETE https://plex.tv/api/servers/{machineId}/shared_servers/{id} - Revoke sharing
  - GET  https://plex.tv/api/v2/home/users - List managed users
  - POST https://plex.tv/api/v2/home/users/restricted - Create managed user

All requests require a valid Plex authentication token obtained via OAuth.
*/

package sync

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/goccy/go-json"
)

const (
	// PlexTVBaseURL is the base URL for plex.tv API endpoints
	PlexTVBaseURL = "https://plex.tv"

	// PlexTVClientTimeout is the HTTP client timeout for plex.tv requests
	PlexTVClientTimeout = 30 * time.Second
)

// PlexTVClient handles communication with plex.tv API for friends and sharing
type PlexTVClient struct {
	token      string
	machineID  string // Server's machineIdentifier for sharing operations
	httpClient *http.Client
	clientID   string // X-Plex-Client-Identifier
	clientName string // X-Plex-Product
}

// PlexTVClientConfig contains configuration for creating a PlexTVClient
type PlexTVClientConfig struct {
	Token      string // Plex authentication token
	MachineID  string // Server's machineIdentifier (from /identity)
	ClientID   string // Application client identifier
	ClientName string // Application name (e.g., "Cartographus")
}

// NewPlexTVClient creates a new client for plex.tv API
func NewPlexTVClient(cfg PlexTVClientConfig) *PlexTVClient {
	clientID := cfg.ClientID
	if clientID == "" {
		clientID = "cartographus"
	}
	clientName := cfg.ClientName
	if clientName == "" {
		clientName = "Cartographus"
	}

	return &PlexTVClient{
		token:      cfg.Token,
		machineID:  cfg.MachineID,
		clientID:   clientID,
		clientName: clientName,
		httpClient: &http.Client{
			Timeout: PlexTVClientTimeout,
		},
	}
}

// PlexFriend represents a friend from the Plex friends list
type PlexFriend struct {
	ID                int64  `json:"id"`
	UUID              string `json:"uuid"`
	Username          string `json:"username"`
	Email             string `json:"email"`
	Thumb             string `json:"thumb"`
	Server            bool   `json:"server"`            // Has server access
	Home              bool   `json:"home"`              // Is Plex Home member
	AllowSync         bool   `json:"allowSync"`         // Can sync content
	AllowCameraUpload bool   `json:"allowCameraUpload"` // Can upload camera photos
	AllowChannels     bool   `json:"allowChannels"`     // Has channel access
	SharedSections    []int  `json:"sharedSections"`    // Library section IDs
	FilterMovies      string `json:"filterMovies"`      // Movie filter (label-based)
	FilterTelevision  string `json:"filterTelevision"`  // TV filter
	FilterMusic       string `json:"filterMusic"`       // Music filter
	Status            string `json:"status"`            // "accepted", "pending", "pending_received"
	Title             string `json:"title"`             // Display name
	RestrictedProfile string `json:"restrictedProfile"` // Content restriction profile
}

// PlexFriendsResponse is the response from GET /api/v2/friends
type PlexFriendsResponse []PlexFriend

// PlexInviteRequest represents a friend invitation request
type PlexInviteRequest struct {
	Email             string `json:"usernameOrEmail"`
	AllowSync         bool   `json:"allowSync"`
	AllowCameraUpload bool   `json:"allowCameraUpload"`
	AllowChannels     bool   `json:"allowChannels"`
}

// PlexSharedServer represents a shared server entry
type PlexSharedServer struct {
	ID                int64  `json:"id"`
	UserID            int64  `json:"userID"`
	Username          string `json:"username"`
	Email             string `json:"email"`
	Thumb             string `json:"thumb"`
	InvitedEmail      string `json:"invitedEmail"`
	AcceptedAt        int64  `json:"acceptedAt"`
	DeletedAt         int64  `json:"deletedAt"`
	LeftAt            int64  `json:"leftAt"`
	AllowSync         bool   `json:"allowSync"`
	AllowCameraUpload bool   `json:"allowCameraUpload"`
	AllowChannels     bool   `json:"allowChannels"`
	FilterMovies      string `json:"filterMovies"`
	FilterTelevision  string `json:"filterTelevision"`
	FilterMusic       string `json:"filterMusic"`
}

// PlexSharedServersResponse is the response from shared_servers endpoints
type PlexSharedServersResponse struct {
	MediaContainer struct {
		Size          int                `json:"size"`
		SharedServers []PlexSharedServer `json:"SharedServer"`
	} `json:"MediaContainer"`
}

// PlexShareRequest represents a library sharing request
type PlexShareRequest struct {
	InvitedEmail      string `json:"invitedEmail"`
	LibrarySectionIDs []int  `json:"librarySectionIds"`
	AllowSync         bool   `json:"allowSync"`
	AllowCameraUpload bool   `json:"allowCameraUpload"`
	AllowChannels     bool   `json:"allowChannels"`
	FilterMovies      string `json:"filterMovies,omitempty"`
	FilterTelevision  string `json:"filterTelevision,omitempty"`
	FilterMusic       string `json:"filterMusic,omitempty"`
}

// PlexManagedUser represents a managed user in Plex Home
type PlexManagedUser struct {
	ID                 int64  `json:"id"`
	UUID               string `json:"uuid"`
	Username           string `json:"username"`
	Title              string `json:"title"`
	Thumb              string `json:"thumb"`
	Restricted         bool   `json:"restricted"`
	RestrictionProfile string `json:"restrictionProfile"` // "little_kid", "older_kid", "teen"
	Home               bool   `json:"home"`
	HomeAdmin          bool   `json:"homeAdmin"`
	Guest              bool   `json:"guest"`
	Protected          bool   `json:"protected"` // Has PIN
}

// PlexHomeUsersResponse is the response from GET /api/v2/home/users
type PlexHomeUsersResponse struct {
	Users []PlexManagedUser `json:"users"`
}

// PlexCreateManagedUserRequest represents the request to create a managed user
type PlexCreateManagedUserRequest struct {
	Name               string `json:"name"`
	RestrictionProfile string `json:"restrictionProfile"` // "little_kid", "older_kid", "teen"
}

// doRequest executes an HTTP request against plex.tv API
func (c *PlexTVClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := PlexTVBaseURL + path

	var reqBody *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	var req *http.Request
	var err error
	if reqBody != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, reqBody)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, http.NoBody)
	}
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Set Plex headers
	req.Header.Set("X-Plex-Token", c.token)
	req.Header.Set("X-Plex-Client-Identifier", c.clientID)
	req.Header.Set("X-Plex-Product", c.clientName)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %d %s", resp.StatusCode, resp.Status)
	}

	// Decode response if result pointer provided
	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// ListFriends returns the list of friends from plex.tv
func (c *PlexTVClient) ListFriends(ctx context.Context) ([]PlexFriend, error) {
	var friends PlexFriendsResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/v2/friends", nil, &friends); err != nil {
		return nil, fmt.Errorf("list friends: %w", err)
	}
	return friends, nil
}

// InviteFriend sends a friend invitation to the specified email
func (c *PlexTVClient) InviteFriend(ctx context.Context, req PlexInviteRequest) error {
	if err := c.doRequest(ctx, http.MethodPost, "/api/v2/friends/invite", req, nil); err != nil {
		return fmt.Errorf("invite friend: %w", err)
	}
	return nil
}

// RemoveFriend removes a friend by their ID
func (c *PlexTVClient) RemoveFriend(ctx context.Context, friendID int64) error {
	path := fmt.Sprintf("/api/v2/friends/%d", friendID)
	if err := c.doRequest(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("remove friend: %w", err)
	}
	return nil
}

// ListSharedServers returns the list of users the server is shared with
func (c *PlexTVClient) ListSharedServers(ctx context.Context) ([]PlexSharedServer, error) {
	if c.machineID == "" {
		return nil, fmt.Errorf("machine ID required for shared servers operations")
	}

	path := fmt.Sprintf("/api/servers/%s/shared_servers", c.machineID)
	var resp PlexSharedServersResponse
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("list shared servers: %w", err)
	}
	return resp.MediaContainer.SharedServers, nil
}

// ShareLibraries shares libraries with a user
func (c *PlexTVClient) ShareLibraries(ctx context.Context, req *PlexShareRequest) error {
	if c.machineID == "" {
		return fmt.Errorf("machine ID required for sharing operations")
	}

	// Build the request body as form data (Plex API expects form-encoded for this endpoint)
	path := fmt.Sprintf("/api/servers/%s/shared_servers", c.machineID)

	// Convert section IDs to comma-separated string
	sectionIDs := ""
	for i, id := range req.LibrarySectionIDs {
		if i > 0 {
			sectionIDs += ","
		}
		sectionIDs += strconv.Itoa(id)
	}

	// Build form data
	formData := map[string]interface{}{
		"invitedEmail":      req.InvitedEmail,
		"librarySectionIds": sectionIDs,
		"allowSync":         req.AllowSync,
		"allowCameraUpload": req.AllowCameraUpload,
		"allowChannels":     req.AllowChannels,
	}
	if req.FilterMovies != "" {
		formData["filterMovies"] = req.FilterMovies
	}
	if req.FilterTelevision != "" {
		formData["filterTelevision"] = req.FilterTelevision
	}
	if req.FilterMusic != "" {
		formData["filterMusic"] = req.FilterMusic
	}

	if err := c.doRequest(ctx, http.MethodPost, path, formData, nil); err != nil {
		return fmt.Errorf("share libraries: %w", err)
	}
	return nil
}

// UpdateSharing updates library sharing settings for a shared server entry
func (c *PlexTVClient) UpdateSharing(ctx context.Context, sharedServerID int64, sectionIDs []int) error {
	if c.machineID == "" {
		return fmt.Errorf("machine ID required for sharing operations")
	}

	path := fmt.Sprintf("/api/servers/%s/shared_servers/%d", c.machineID, sharedServerID)

	// Convert section IDs to comma-separated string
	sectionIDsStr := ""
	for i, id := range sectionIDs {
		if i > 0 {
			sectionIDsStr += ","
		}
		sectionIDsStr += strconv.Itoa(id)
	}

	formData := map[string]interface{}{
		"librarySectionIds": sectionIDsStr,
	}

	if err := c.doRequest(ctx, http.MethodPut, path, formData, nil); err != nil {
		return fmt.Errorf("update sharing: %w", err)
	}
	return nil
}

// RevokeSharing removes library sharing for a shared server entry
func (c *PlexTVClient) RevokeSharing(ctx context.Context, sharedServerID int64) error {
	if c.machineID == "" {
		return fmt.Errorf("machine ID required for sharing operations")
	}

	path := fmt.Sprintf("/api/servers/%s/shared_servers/%d", c.machineID, sharedServerID)
	if err := c.doRequest(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("revoke sharing: %w", err)
	}
	return nil
}

// ListManagedUsers returns the list of managed users in Plex Home
func (c *PlexTVClient) ListManagedUsers(ctx context.Context) ([]PlexManagedUser, error) {
	var resp PlexHomeUsersResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/v2/home/users", nil, &resp); err != nil {
		return nil, fmt.Errorf("list managed users: %w", err)
	}
	return resp.Users, nil
}

// CreateManagedUser creates a new managed user in Plex Home
func (c *PlexTVClient) CreateManagedUser(ctx context.Context, req PlexCreateManagedUserRequest) (*PlexManagedUser, error) {
	var user PlexManagedUser
	if err := c.doRequest(ctx, http.MethodPost, "/api/v2/home/users/restricted", req, &user); err != nil {
		return nil, fmt.Errorf("create managed user: %w", err)
	}
	return &user, nil
}

// DeleteManagedUser removes a managed user from Plex Home
func (c *PlexTVClient) DeleteManagedUser(ctx context.Context, userID int64) error {
	path := fmt.Sprintf("/api/v2/home/users/%d", userID)
	if err := c.doRequest(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("delete managed user: %w", err)
	}
	return nil
}

// UpdateManagedUserRestrictions updates the restriction profile for a managed user
func (c *PlexTVClient) UpdateManagedUserRestrictions(ctx context.Context, userID int64, profile string) error {
	path := fmt.Sprintf("/api/v2/home/users/%d", userID)
	body := map[string]interface{}{
		"restrictionProfile": profile,
	}
	if err := c.doRequest(ctx, http.MethodPut, path, body, nil); err != nil {
		return fmt.Errorf("update user restrictions: %w", err)
	}
	return nil
}
