// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetPlexBandwidthStatistics retrieves bandwidth statistics from Plex Media Server
//
// This method wraps the PlexClient.GetBandwidthStatistics call and provides
// proper error handling when Plex integration is not configured.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - timespan: Optional time aggregation period in seconds (nil for all data)
//
// Returns:
//   - *models.PlexBandwidthResponse: Bandwidth statistics
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexBandwidthStatistics(ctx context.Context, timespan *int) (*models.PlexBandwidthResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetBandwidthStatistics(ctx, timespan)
}

// GetPlexLibrarySections retrieves all library sections from Plex Media Server
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - *models.PlexLibrarySectionsResponse: List of library sections
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexLibrarySections(ctx context.Context) (*models.PlexLibrarySectionsResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetLibrarySections(ctx)
}

// GetPlexLibrarySectionContent retrieves content from a specific library section
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - sectionKey: Library section key (from GetPlexLibrarySections)
//   - start: Pagination start offset (nil for 0)
//   - size: Number of items to return (nil for default)
//
// Returns:
//   - *models.PlexLibrarySectionContentResponse: Media items in the section
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexLibrarySectionContent(ctx context.Context, sectionKey string, start, size *int) (*models.PlexLibrarySectionContentResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetLibrarySectionContent(ctx, sectionKey, start, size)
}

// GetPlexLibrarySectionRecentlyAdded retrieves recently added content from a library section
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - sectionKey: Library section key (from GetPlexLibrarySections)
//   - size: Number of items to return (nil for default)
//
// Returns:
//   - *models.PlexLibrarySectionContentResponse: Recently added media items
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexLibrarySectionRecentlyAdded(ctx context.Context, sectionKey string, size *int) (*models.PlexLibrarySectionContentResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetLibrarySectionRecentlyAdded(ctx, sectionKey, size)
}

// GetPlexActivities retrieves current server activities from Plex Media Server
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - *models.PlexActivitiesResponse: List of server activities
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexActivities(ctx context.Context) (*models.PlexActivitiesResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetActivities(ctx)
}

// IsPlexEnabled returns true if Plex integration is configured and enabled
func (m *Manager) IsPlexEnabled() bool {
	return m.plexClient != nil
}

// GetPlexSessions retrieves active playback sessions from Plex Media Server
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - *models.PlexSessionsResponse: Active playback sessions with transcode details
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexSessions(ctx context.Context) (*models.PlexSessionsResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetSessions(ctx)
}

// GetPlexIdentity retrieves server identity from Plex Media Server
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - *models.PlexIdentityResponse: Server identity information
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexIdentity(ctx context.Context) (*models.PlexIdentityResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetIdentity(ctx)
}

// GetPlexMetadata retrieves detailed metadata for a specific media item
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - ratingKey: Unique identifier for the media item
//
// Returns:
//   - *models.PlexMetadataResponse: Detailed metadata including media info
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexMetadata(ctx context.Context, ratingKey string) (*models.PlexMetadataResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetMetadata(ctx, ratingKey)
}

// GetPlexDevices retrieves connected devices from Plex Media Server
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - *models.PlexDevicesResponse: List of connected devices
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexDevices(ctx context.Context) (*models.PlexDevicesResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetDevices(ctx)
}

// GetPlexAccounts retrieves user accounts from Plex Media Server
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - *models.PlexAccountsResponse: List of user accounts
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexAccounts(ctx context.Context) (*models.PlexAccountsResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetAccounts(ctx)
}

// GetPlexOnDeck retrieves on-deck content from Plex Media Server
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - *models.PlexOnDeckResponse: On-deck items (partially watched content)
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexOnDeck(ctx context.Context) (*models.PlexOnDeckResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetOnDeck(ctx)
}

// GetPlexPlaylists retrieves playlists from Plex Media Server
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - *models.PlexPlaylistsResponse: List of user playlists
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexPlaylists(ctx context.Context) (*models.PlexPlaylistsResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetPlaylists(ctx)
}

// GetPlexSearch performs a search within a library section
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - sectionKey: Library section key
//   - query: Search query string
//   - mediaType: Optional media type filter (1=movie, 4=episode, etc.)
//
// Returns:
//   - *models.PlexSearchResponse: Search results
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexSearch(ctx context.Context, sectionKey, query string, mediaType *int) (*models.PlexSearchResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.Search(ctx, sectionKey, query, mediaType)
}

// GetPlexTranscodeSessions retrieves active transcode sessions from Plex Media Server
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - *models.PlexTranscodeSessionsResponse: Active transcode sessions
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexTranscodeSessions(ctx context.Context) (*models.PlexTranscodeSessionsResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetTranscodeSessionsDetailed(ctx)
}

// CancelPlexTranscode cancels an active transcode session
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - sessionKey: Transcode session key to cancel
//
// Returns:
//   - error: If Plex is not enabled, session not found, or network errors occur
func (m *Manager) CancelPlexTranscode(ctx context.Context, sessionKey string) error {
	if m.plexClient == nil {
		return fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.CancelTranscode(ctx, sessionKey)
}

// GetPlexServerCapabilities retrieves comprehensive server capabilities from Plex Media Server
//
// This endpoint returns full server capabilities including feature flags,
// transcoder support, Plex Pass status, and available directories.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - *models.PlexServerCapabilitiesResponse: Server capabilities and features
//   - error: If Plex is not enabled or network errors occur
func (m *Manager) GetPlexServerCapabilities(ctx context.Context) (*models.PlexServerCapabilitiesResponse, error) {
	if m.plexClient == nil {
		return nil, fmt.Errorf("plex client not initialized")
	}
	return m.plexClient.GetServerCapabilities(ctx)
}
