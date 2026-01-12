// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
plex_server.go - Plex Server Information Methods

This file provides methods for accessing Plex Media Server system information,
capabilities, devices, and statistics.

Server Information:
  - Ping(): Verify connectivity and authentication
  - GetServerIdentity(): Machine identifier, version, platform
  - GetIdentity(): Server identity using models types
  - GetServerCapabilities(): Full server capabilities and feature flags

System Monitoring:
  - GetActivities(): Background tasks and server activities
  - GetBandwidthStatistics(): Historical bandwidth usage data

Connected Devices:
  - GetDevices(): All connected Plex clients and players
  - GetAccounts(): User accounts with access to server

Server Capabilities Response:
The capabilities endpoint (/) returns comprehensive server info:
  - Feature flags (sync, camera upload, etc.)
  - Transcoder support and settings
  - Account status and restrictions
  - Available directories and services

Bandwidth Statistics:
Returns aggregated data by configurable timespan:
  - Device information (name, platform, client identifier)
  - Account information (user name, preferences)
  - Bandwidth records (bytes, timestamp, LAN vs WAN)
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tomtom215/cartographus/internal/models"
)

// Ping checks connectivity and authentication to Plex server
//
// This method verifies that:
//   - Plex server is reachable at the configured URL
//   - Authentication token is valid
//   - Server responds with HTTP 200
func (c *PlexClient) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Plex-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// GetServerIdentity retrieves Plex server machine identifier and version info
//
// This endpoint returns unique server identification and platform details:
//   - MachineIdentifier: Unique server UUID
//   - Version: Plex Media Server version (e.g., "1.40.0.8395")
//   - Platform: Operating system (e.g., "Linux", "Windows", "MacOSX")
func (c *PlexClient) GetServerIdentity(ctx context.Context) (*PlexIdentityContainer, error) {
	var identityResp PlexIdentityResponse
	if err := c.doJSONRequest(ctx, "/identity", &identityResp); err != nil {
		return nil, err
	}
	return &identityResp.MediaContainer, nil
}

// GetIdentity retrieves server identity from Plex Media Server
//
// Endpoint: GET /identity
func (c *PlexClient) GetIdentity(ctx context.Context) (*models.PlexIdentityResponse, error) {
	var identityResp models.PlexIdentityResponse
	if err := c.doJSONRequest(ctx, "/identity", &identityResp); err != nil {
		return nil, err
	}
	return &identityResp, nil
}

// GetServerCapabilities retrieves comprehensive server capabilities from Plex Media Server
//
// This endpoint returns the full server capabilities including feature flags,
// transcoder support, account status, and available directories. This is the
// primary server information endpoint that provides more detail than /identity.
//
// Endpoint: GET /
func (c *PlexClient) GetServerCapabilities(ctx context.Context) (*models.PlexServerCapabilitiesResponse, error) {
	var capabilitiesResp models.PlexServerCapabilitiesResponse
	if err := c.doJSONRequest(ctx, "/", &capabilitiesResp); err != nil {
		return nil, err
	}
	return &capabilitiesResp, nil
}

// GetActivities retrieves current server activities from Plex Media Server
//
// This endpoint returns background tasks and server activities:
//   - Library scans and refreshes
//   - Media analysis and optimization
//   - Database maintenance tasks
//   - Scheduled jobs progress
//
// Endpoint: GET /activities
func (c *PlexClient) GetActivities(ctx context.Context) (*models.PlexActivitiesResponse, error) {
	var activitiesResp models.PlexActivitiesResponse
	if err := c.doJSONRequest(ctx, "/activities", &activitiesResp); err != nil {
		return nil, err
	}
	return &activitiesResp, nil
}

// GetBandwidthStatistics retrieves bandwidth usage statistics from Plex Media Server
//
// This endpoint returns historical bandwidth consumption data aggregated by timespan:
//   - Device information (name, platform, client identifier)
//   - Account information (user name, preferences)
//   - Bandwidth records (bytes, timestamp, LAN vs WAN)
//
// Endpoint: GET /statistics/bandwidth
func (c *PlexClient) GetBandwidthStatistics(ctx context.Context, timespan *int) (*models.PlexBandwidthResponse, error) {
	path := "/statistics/bandwidth"
	if timespan != nil {
		path = fmt.Sprintf("/statistics/bandwidth?timespan=%d", *timespan)
	}

	var bandwidthResp models.PlexBandwidthResponse
	if err := c.doJSONRequest(ctx, path, &bandwidthResp); err != nil {
		return nil, err
	}
	return &bandwidthResp, nil
}

// GetDevices retrieves connected devices from Plex Media Server
//
// Endpoint: GET /devices
func (c *PlexClient) GetDevices(ctx context.Context) (*models.PlexDevicesResponse, error) {
	var devicesResp models.PlexDevicesResponse
	if err := c.doJSONRequest(ctx, "/devices", &devicesResp); err != nil {
		return nil, err
	}
	return &devicesResp, nil
}

// GetAccounts retrieves user accounts from Plex Media Server
//
// Endpoint: GET /accounts
func (c *PlexClient) GetAccounts(ctx context.Context) (*models.PlexAccountsResponse, error) {
	var accountsResp models.PlexAccountsResponse
	if err := c.doJSONRequest(ctx, "/accounts", &accountsResp); err != nil {
		return nil, err
	}
	return &accountsResp, nil
}
