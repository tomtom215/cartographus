// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
plex.go - Plex Media Server API Client

This file provides the core PlexClient struct and primary Plex API types for
communicating with Plex Media Server's REST API.

PlexClient Features:
  - HTTP client with 30-second timeout
  - X-Plex-Token authentication
  - Automatic rate limit handling with exponential backoff
  - JSON response parsing

Data Limitations:
Plex history is INCOMPLETE compared to Tautulli. Missing fields:
  - IP addresses (no geolocation source)
  - Platform/Player information
  - Quality metrics (resolution, codec, transcode decision)
  - Stream details (bitrate, audio channels, HDR)
  - Connection security flags

This makes Tautulli the primary data source, with Plex providing supplementary
real-time session data and historical backfill capabilities.

API Methods in this file:
  - NewPlexClient(): Create authenticated client
  - GetHistoryAll(): Fetch complete playback history
  - doRequestWithRateLimit(): HTTP 429 retry logic

Related Files:
  - plex_request.go: HTTP request helpers
  - plex_sessions.go: Session and transcode monitoring
  - plex_library.go: Library content methods
  - plex_server.go: Server information methods
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// PlexClient handles communication with Plex Media Server API
// Implements hybrid data architecture (v1.37) for historical playback data backfill
type PlexClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Plex API Response Structures
// Based on Plex Media Server API: /status/sessions/history/all

// PlexHistoryResponse represents the top-level response from /status/sessions/history/all
type PlexHistoryResponse struct {
	MediaContainer PlexMediaContainer `json:"MediaContainer"`
}

// PlexMediaContainer wraps the history metadata array
type PlexMediaContainer struct {
	Size     int            `json:"size"`     // Total number of records returned
	Metadata []PlexMetadata `json:"Metadata"` // Array of playback history records
}

// PlexMetadata represents a single playback history entry from Plex
// NOTE: Plex history is INCOMPLETE compared to Tautulli - missing IPs, geolocations, quality metrics
type PlexMetadata struct {
	// Primary identifiers
	RatingKey            string `json:"ratingKey"`                      // Plex unique content identifier
	Key                  string `json:"key"`                            // Plex metadata key path
	ParentRatingKey      string `json:"parentRatingKey,omitempty"`      // Season/Album rating key
	GrandparentRatingKey string `json:"grandparentRatingKey,omitempty"` // Show/Artist rating key

	// Media type and titles
	Type             string `json:"type"`                       // "movie", "episode", "track"
	Title            string `json:"title"`                      // Episode/Movie/Song title
	GrandparentTitle string `json:"grandparentTitle,omitempty"` // TV show or artist name
	ParentTitle      string `json:"parentTitle,omitempty"`      // Season or album name
	OriginalTitle    string `json:"originalTitle,omitempty"`    // Original non-localized title

	// Playback metadata
	ViewedAt   int64 `json:"viewedAt"`             // Unix timestamp (seconds since epoch)
	Duration   int64 `json:"duration,omitempty"`   // Total duration in milliseconds
	ViewOffset int64 `json:"viewOffset,omitempty"` // Watch position in milliseconds (how far watched)

	// User information
	AccountID int       `json:"accountID"` // Plex user account ID
	User      *PlexUser `json:"User,omitempty"`

	// Episode numbering (for TV shows)
	Index       int `json:"index,omitempty"`       // Episode number (media_index)
	ParentIndex int `json:"parentIndex,omitempty"` // Season number (parent_media_index)

	// Additional metadata
	Year                  int    `json:"year,omitempty"`
	Guid                  string `json:"guid,omitempty"`                  // External IDs (IMDB, TVDB, TMDB)
	OriginallyAvailableAt string `json:"originallyAvailableAt,omitempty"` // Release date (ISO 8601)
	Thumb                 string `json:"thumb,omitempty"`                 // Thumbnail path

	// MISSING FROM PLEX (compared to Tautulli):
	// - IP address (geolocation source)
	// - Platform/Player (device information)
	// - Quality metrics (resolution, codec, transcode decision)
	// - Stream details (bitrate, audio channels, HDR)
	// - Connection security (secure, relayed, local flags)
}

// PlexUser represents user information in Plex history responses
type PlexUser struct {
	ID    int    `json:"id"`
	Title string `json:"title"` // Username/friendly name
}

// PlexIdentityResponse represents the response from /identity endpoint
type PlexIdentityResponse struct {
	MediaContainer PlexIdentityContainer `json:"MediaContainer"`
}

// PlexIdentityContainer wraps server identity information
type PlexIdentityContainer struct {
	MachineIdentifier string `json:"machineIdentifier"`
	Version           string `json:"version"`
	Platform          string `json:"platform"`
}

// NewPlexClient creates a new Plex API client with authentication token
//
// Parameters:
//   - baseURL: Plex Media Server URL (e.g., "http://localhost:32400")
//   - token: X-Plex-Token for authentication (find in Settings → Network → Show Advanced)
//
// Returns initialized PlexClient with 30-second HTTP timeout
func NewPlexClient(baseURL, token string) *PlexClient {
	return &PlexClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetHistoryAll fetches complete playback history from Plex Media Server
//
// This endpoint returns ALL playback history available in Plex's database.
// Unlike Tautulli, Plex history is INCOMPLETE:
//   - Has: Timestamps, titles, users, episode numbers
//   - Missing: IP addresses, geolocations, quality metrics, platform/player info
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - sort: Sort order - "viewedAt" (oldest first) or "-viewedAt" (newest first)
//   - accountID: Optional filter by specific Plex user account ID (nil for all users)
//
// Returns:
//   - []PlexMetadata: Array of playback history records
//   - error: Network errors, authentication failures, or JSON parsing errors
func (c *PlexClient) GetHistoryAll(ctx context.Context, sort string, accountID *int) ([]PlexMetadata, error) {
	query := url.Values{}
	if sort != "" {
		query.Add("sort", sort)
	}
	if accountID != nil {
		query.Add("accountID", fmt.Sprintf("%d", *accountID))
	}

	var historyResp PlexHistoryResponse
	if err := c.doJSONRequestWithQuery(ctx, "/status/sessions/history/all", query, &historyResp); err != nil {
		return nil, err
	}

	return historyResp.MediaContainer.Metadata, nil
}

// doRequestWithRateLimit executes HTTP request with automatic retry on rate limiting (HTTP 429)
//
// This method implements exponential backoff retry logic to handle Plex API rate limits:
//   - Max 5 retry attempts
//   - Exponential backoff: 1s, 2s, 4s, 8s, 16s
//   - Respects Retry-After header (RFC 6585) if present
//   - Only retries on HTTP 429 (Too Many Requests)
//
// Parameters:
//   - req: Prepared *http.Request with context, headers, and authentication
//
// Returns:
//   - *http.Response: Successful response (caller must close Body)
//   - error: Network errors or exceeded retry attempts
func (c *PlexClient) doRequestWithRateLimit(req *http.Request) (*http.Response, error) {
	const maxRetries = 5
	baseDelay := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("execute request: %w", err)
		}

		// Success - return response
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Rate limited - close response and retry
		resp.Body.Close()

		// Last attempt failed - return error
		if attempt == maxRetries {
			return nil, fmt.Errorf("rate limit exceeded after %d retries", maxRetries)
		}

		// Calculate retry delay (exponential backoff)
		retryDelay := baseDelay * (1 << attempt) // 1s, 2s, 4s, 8s, 16s

		// Check for Retry-After header (RFC 6585)
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Try parsing as seconds (integer)
			if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
				retryDelay = seconds
			}
		}

		logging.Warn().Dur("retry_delay", retryDelay).Int("attempt", attempt+1).Int("max_retries", maxRetries).Msg("Plex API rate limited (HTTP 429), retrying")

		// Wait before retrying
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(retryDelay):
			// Continue to next retry
		}
	}

	return nil, fmt.Errorf("unreachable code: retry loop should return or error")
}
