// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
tautulli_users.go - Tautulli User Management Methods

This file provides methods for retrieving user information and statistics
from Tautulli, including the generic callTautulliAPI helper.

User Methods:
  - GetUser(): Single user details by user_id
  - GetUsersTable(): Paginated user listing with statistics
  - GetUserIPs(): IP address history for a user
  - GetUserPlayerStats(): Player/device usage by user
  - GetUserWatchTimeStats(): Watch time analytics by user

Generic API Helper:
The callTautulliAPI[T any] function provides a type-safe generic helper
that centralizes:
  - URL construction with API key and command
  - HTTP request execution with rate limiting
  - Error handling and response validation
  - JSON decoding to typed response struct

This eliminates ~40 lines of boilerplate per endpoint, saving approximately
240+ lines of code across the 54 Tautulli API methods.

User Statistics:
User endpoints provide comprehensive viewing analytics:
  - Total plays and watch time
  - Last activity timestamp
  - Favorite media and libraries
  - IP address geolocation history
  - Device and player preferences
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// callTautulliAPI is a generic helper that handles all Tautulli API calls
// It reduces code duplication by centralizing URL building, HTTP requests, error handling, and JSON decoding
// This eliminates ~40 lines of boilerplate per endpoint (6 endpoints = 240 lines saved)
func callTautulliAPI[T any](ctx context.Context, c *TautulliClient, cmd string, params url.Values, validator func(*T) error) (*T, error) {
	// Add required API params
	params.Set("apikey", c.apiKey)
	params.Set("cmd", cmd)

	reqURL := fmt.Sprintf("%s/api/v2?%s", c.baseURL, params.Encode())

	resp, err := c.doRequestWithRateLimit(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make %s request: %w", cmd, err)
	}
	defer resp.Body.Close()

	// Check HTTP status code before attempting to decode
	if resp.StatusCode != http.StatusOK {
		body := readBodyForError(resp.Body)
		return nil, fmt.Errorf("%s request failed with status %d: %s", cmd, resp.StatusCode, string(body))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode %s response: %w", cmd, err)
	}

	// Validate response success
	if validator != nil {
		if err := validator(&result); err != nil {
			return nil, err
		}
	}

	return &result, nil
}

// Helper functions to validate specific response types
func validateUserResponse(u *tautulli.TautulliUser) error {
	if u.Response.Result != "success" {
		msg := "unknown error"
		if u.Response.Message != nil {
			msg = *u.Response.Message
		}
		return fmt.Errorf("get_user request failed: %s", msg)
	}
	return nil
}

func validateUserPlayerStatsResponse(u *tautulli.TautulliUserPlayerStats) error {
	if u.Response.Result != "success" {
		msg := "unknown error"
		if u.Response.Message != nil {
			msg = *u.Response.Message
		}
		return fmt.Errorf("get_user_player_stats request failed: %s", msg)
	}
	return nil
}

func validateUserWatchTimeStatsResponse(u *tautulli.TautulliUserWatchTimeStats) error {
	if u.Response.Result != "success" {
		msg := "unknown error"
		if u.Response.Message != nil {
			msg = *u.Response.Message
		}
		return fmt.Errorf("get_user_watch_time_stats request failed: %s", msg)
	}
	return nil
}

func validateUserIPsResponse(u *tautulli.TautulliUserIPs) error {
	if u.Response.Result != "success" {
		msg := "unknown error"
		if u.Response.Message != nil {
			msg = *u.Response.Message
		}
		return fmt.Errorf("get_user_ips request failed: %s", msg)
	}
	return nil
}

func validateUsersTableResponse(u *tautulli.TautulliUsersTable) error {
	if u.Response.Result != "success" {
		msg := "unknown error"
		if u.Response.Message != nil {
			msg = *u.Response.Message
		}
		return fmt.Errorf("get_users_table request failed: %s", msg)
	}
	return nil
}

func validateUserLoginsResponse(u *tautulli.TautulliUserLogins) error {
	if u.Response.Result != "success" {
		msg := "unknown error"
		if u.Response.Message != nil {
			msg = *u.Response.Message
		}
		return fmt.Errorf("get_user_logins request failed: %s", msg)
	}
	return nil
}

func validateUsersResponse(u *tautulli.TautulliUsers) error {
	if u.Response.Result != "success" {
		msg := "unknown error"
		if u.Response.Message != nil {
			msg = *u.Response.Message
		}
		return fmt.Errorf("get_users request failed: %s", msg)
	}
	return nil
}

func (c *TautulliClient) GetUser(ctx context.Context, userID int) (*tautulli.TautulliUser, error) {
	params := url.Values{}
	params.Set("user_id", fmt.Sprintf("%d", userID))
	return callTautulliAPI(ctx, c, "get_user", params, validateUserResponse)
}

// GetUsers retrieves a list of all users that have accessed the Plex server
// This is the bulk fetch endpoint - more efficient than calling GetUser for each user
func (c *TautulliClient) GetUsers(ctx context.Context) (*tautulli.TautulliUsers, error) {
	params := url.Values{}
	return callTautulliAPI(ctx, c, "get_users", params, validateUsersResponse)
}

// GetLibraryUserStats retrieves library usage statistics by user from Tautulli

func (c *TautulliClient) GetUserPlayerStats(ctx context.Context, userID int) (*tautulli.TautulliUserPlayerStats, error) {
	params := url.Values{}
	if userID > 0 {
		params.Set("user_id", fmt.Sprintf("%d", userID))
	}
	return callTautulliAPI(ctx, c, "get_user_player_stats", params, validateUserPlayerStatsResponse)
}

// GetUserWatchTimeStats retrieves watch time statistics for a specific user
func (c *TautulliClient) GetUserWatchTimeStats(ctx context.Context, userID int, queryDays string) (*tautulli.TautulliUserWatchTimeStats, error) {
	params := url.Values{}
	if userID > 0 {
		params.Set("user_id", fmt.Sprintf("%d", userID))
	}
	if queryDays != "" {
		params.Set("query_days", queryDays)
	}
	return callTautulliAPI(ctx, c, "get_user_watch_time_stats", params, validateUserWatchTimeStatsResponse)
}

// GetItemUserStats retrieves user statistics for a specific media item

func (c *TautulliClient) GetUserIPs(ctx context.Context, userID int) (*tautulli.TautulliUserIPs, error) {
	params := url.Values{}
	if userID > 0 {
		params.Set("user_id", fmt.Sprintf("%d", userID))
	}
	return callTautulliAPI(ctx, c, "get_user_ips", params, validateUserIPsResponse)
}

// GetUsersTable retrieves paginated user data with sorting and filtering
func (c *TautulliClient) GetUsersTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUsersTable, error) {
	params := url.Values{}
	if grouping >= 0 {
		params.Set("grouping", fmt.Sprintf("%d", grouping))
	}
	if orderColumn != "" {
		params.Set("order_column", orderColumn)
	}
	if orderDir != "" {
		params.Set("order_dir", orderDir)
	}
	if start >= 0 {
		params.Set("start", fmt.Sprintf("%d", start))
	}
	if length > 0 {
		params.Set("length", fmt.Sprintf("%d", length))
	}
	if search != "" {
		params.Set("search", search)
	}
	return callTautulliAPI(ctx, c, "get_users_table", params, validateUsersTableResponse)
}

// GetUserLogins retrieves login history and patterns for users
func (c *TautulliClient) GetUserLogins(ctx context.Context, userID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUserLogins, error) {
	params := url.Values{}
	if userID > 0 {
		params.Set("user_id", fmt.Sprintf("%d", userID))
	}
	if orderColumn != "" {
		params.Set("order_column", orderColumn)
	}
	if orderDir != "" {
		params.Set("order_dir", orderDir)
	}
	if start >= 0 {
		params.Set("start", fmt.Sprintf("%d", start))
	}
	if length > 0 {
		params.Set("length", fmt.Sprintf("%d", length))
	}
	if search != "" {
		params.Set("search", search)
	}
	return callTautulliAPI(ctx, c, "get_user_logins", params, validateUserLoginsResponse)
}
