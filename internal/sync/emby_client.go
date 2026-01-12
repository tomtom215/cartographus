// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
emby_client.go - Emby REST API Client

This file implements a REST API client for Emby media server.
It provides methods to fetch session data, user information, and system info.

API Reference: https://dev.emby.media/doc/restapi/index.html
*/

package sync

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/models"
)

// EmbyClientInterface defines the interface for Emby API operations.
// Both EmbyClient and EmbyCircuitBreakerClient implement this interface.
type EmbyClientInterface interface {
	Ping(ctx context.Context) error
	GetSessions(ctx context.Context) ([]models.EmbySession, error)
	GetActiveSessions(ctx context.Context) ([]models.EmbySession, error)
	GetSystemInfo(ctx context.Context) (*EmbySystemInfo, error)
	GetUsers(ctx context.Context) ([]EmbyUser, error)
	StopSession(ctx context.Context, sessionID string) error
	GetWebSocketURL() (string, error)
}

// Ensure EmbyClient implements EmbyClientInterface
var _ EmbyClientInterface = (*EmbyClient)(nil)

// EmbyClient provides access to Emby REST API
type EmbyClient struct {
	baseURL    string
	apiKey     string
	userID     string // Optional: for user-scoped API operations
	httpClient *http.Client
}

// EmbySystemInfo represents Emby server system information
type EmbySystemInfo struct {
	ServerName         string `json:"ServerName"`
	Version            string `json:"Version"`
	ID                 string `json:"Id"`
	OperatingSystem    string `json:"OperatingSystem"`
	HasUpdateAvailable bool   `json:"HasUpdateAvailable"`
}

// EmbyUser represents an Emby user
type EmbyUser struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

// NewEmbyClient creates a new Emby API client
//
// Parameters:
//   - baseURL: Emby server URL (e.g., http://localhost:8096)
//   - apiKey: Emby API key from Admin Dashboard > API Keys
//   - userID: Optional user ID for user-scoped operations
func NewEmbyClient(baseURL, apiKey, userID string) *EmbyClient {
	// Normalize URL (remove trailing slash)
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &EmbyClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		userID:  userID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetSessions retrieves all active sessions from Emby
//
// Returns a list of all active playback sessions including:
//   - User information
//   - Device/client details
//   - Currently playing content (if any)
//   - Playback state and progress
//   - Transcode information (if transcoding)
func (c *EmbyClient) GetSessions(ctx context.Context) ([]models.EmbySession, error) {
	endpoint := "/Sessions"

	resp, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("emby sessions request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("emby sessions returned status %d (failed to read body)", resp.StatusCode)
		}
		return nil, fmt.Errorf("emby sessions returned status %d: %s", resp.StatusCode, string(body))
	}

	var sessions []models.EmbySession
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode emby sessions: %w", err)
	}

	return sessions, nil
}

// GetActiveSessions retrieves only sessions with active playback
//
// Filters sessions to return only those with NowPlayingItem set,
// indicating active playback (playing or paused).
func (c *EmbyClient) GetActiveSessions(ctx context.Context) ([]models.EmbySession, error) {
	sessions, err := c.GetSessions(ctx)
	if err != nil {
		return nil, err
	}

	// Filter to active sessions only
	active := make([]models.EmbySession, 0, len(sessions))
	for i := range sessions {
		if sessions[i].NowPlayingItem != nil {
			active = append(active, sessions[i])
		}
	}

	return active, nil
}

// GetSystemInfo retrieves Emby server system information
func (c *EmbyClient) GetSystemInfo(ctx context.Context) (*EmbySystemInfo, error) {
	endpoint := "/System/Info"

	resp, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("emby system info request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("emby system info returned status %d (failed to read body)", resp.StatusCode)
		}
		return nil, fmt.Errorf("emby system info returned status %d: %s", resp.StatusCode, string(body))
	}

	var info EmbySystemInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode emby system info: %w", err)
	}

	return &info, nil
}

// GetUsers retrieves all users from Emby
func (c *EmbyClient) GetUsers(ctx context.Context) ([]EmbyUser, error) {
	endpoint := "/Users"

	resp, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("emby users request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("emby users returned status %d (failed to read body)", resp.StatusCode)
		}
		return nil, fmt.Errorf("emby users returned status %d: %s", resp.StatusCode, string(body))
	}

	var users []EmbyUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode emby users: %w", err)
	}

	return users, nil
}

// Ping tests connectivity to the Emby server
func (c *EmbyClient) Ping(ctx context.Context) error {
	endpoint := "/System/Ping"

	resp, err := c.doRequest(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("emby ping failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("emby ping returned status %d", resp.StatusCode)
	}

	return nil
}

// StopSession stops/terminates a playback session
//
// Parameters:
//   - sessionID: The ID of the session to stop (from EmbySession.ID)
//
// This sends a stop command to the specified session, which will halt playback
// on the client device. The session must be an active playback session.
func (c *EmbyClient) StopSession(ctx context.Context, sessionID string) error {
	endpoint := fmt.Sprintf("/Sessions/%s/Playing/Stop", sessionID)

	resp, err := c.doPostRequest(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("emby stop session request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Emby returns 204 No Content on success
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("emby stop session returned status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("emby stop session returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetWebSocketURL returns the WebSocket URL for real-time notifications
func (c *EmbyClient) GetWebSocketURL() (string, error) {
	parsedURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Convert http(s) to ws(s)
	switch parsedURL.Scheme {
	case "http":
		parsedURL.Scheme = "ws"
	case "https":
		parsedURL.Scheme = "wss"
	default:
		parsedURL.Scheme = "ws"
	}

	// Build WebSocket URL with API key
	// Emby uses /embywebsocket instead of /socket
	parsedURL.Path = "/embywebsocket"
	query := parsedURL.Query()
	query.Set("api_key", c.apiKey)
	if c.userID != "" {
		query.Set("deviceId", "cartographus-"+c.userID)
	} else {
		query.Set("deviceId", "cartographus")
	}
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// doRequest performs an HTTP GET request to the Emby API
func (c *EmbyClient) doRequest(ctx context.Context, endpoint string) (*http.Response, error) {
	fullURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authorization header using API key
	req.Header.Set("X-Emby-Token", c.apiKey)
	req.Header.Set("X-Emby-Client", "Cartographus")
	req.Header.Set("X-Emby-Device-Name", "Cartographus")
	req.Header.Set("X-Emby-Device-Id", "cartographus")
	req.Header.Set("X-Emby-Client-Version", "1.0.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// doPostRequest performs an HTTP POST request to the Emby API
func (c *EmbyClient) doPostRequest(ctx context.Context, endpoint string) (*http.Response, error) {
	fullURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authorization header using API key
	req.Header.Set("X-Emby-Token", c.apiKey)
	req.Header.Set("X-Emby-Client", "Cartographus")
	req.Header.Set("X-Emby-Device-Name", "Cartographus")
	req.Header.Set("X-Emby-Device-Id", "cartographus")
	req.Header.Set("X-Emby-Client-Version", "1.0.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// EmbySessionToPlaybackEvent converts an Emby session to a PlaybackEvent
//
//nolint:gocyclo // Data mapping function with many field assignments - complexity is inherent
func EmbySessionToPlaybackEvent(session *models.EmbySession, _ string) *models.PlaybackEvent {
	if session == nil || session.NowPlayingItem == nil {
		return nil
	}

	item := session.NowPlayingItem

	// Get transcode decision as pointer
	transcodeDecision := session.GetTranscodeDecision()
	machineID := session.DeviceID

	event := &models.PlaybackEvent{
		SessionKey:        session.ID,
		Source:            "emby",
		TranscodeDecision: &transcodeDecision,
		Platform:          session.Client,
		Player:            session.DeviceName,
		MachineID:         &machineID,
		StartedAt:         time.Now(), // Will be updated from session activity
		CreatedAt:         time.Now(),
	}

	// User information
	event.UserID = 0 // Emby uses string UUIDs; we'll need a mapping strategy
	event.Username = session.UserName
	friendlyName := session.UserName
	event.FriendlyName = &friendlyName

	// Content information
	event.MediaType = item.GetMediaType()
	event.Title = item.Name
	if item.ProductionYear > 0 {
		event.Year = &item.ProductionYear
	}

	// TV Show specific
	if item.SeriesName != "" {
		event.GrandparentTitle = &item.SeriesName
		if item.SeasonName != "" {
			event.ParentTitle = &item.SeasonName
		}
		if item.IndexNumber > 0 {
			event.MediaIndex = &item.IndexNumber
		}
		if item.ParentIndexNumber > 0 {
			event.ParentMediaIndex = &item.ParentIndexNumber
		}
	}

	// Music specific
	if item.Album != "" {
		if item.AlbumArtist != "" {
			event.GrandparentTitle = &item.AlbumArtist
		}
		event.ParentTitle = &item.Album
	}

	// Playback state - calculate percent complete
	if session.PlayState != nil {
		if item.RunTimeTicks > 0 && session.PlayState.PositionTicks > 0 {
			event.PercentComplete = int((session.PlayState.PositionTicks * 100) / item.RunTimeTicks)
		}
		if session.PlayState.IsPaused {
			state := "paused"
			event.State = &state
		} else {
			state := "playing"
			event.State = &state
		}
	}

	// IP address
	event.IPAddress = session.GetIPAddress()

	// External IDs (for correlation)
	if item.ProviderIDs != nil {
		if imdb, ok := item.ProviderIDs["Imdb"]; ok {
			guid := "imdb://" + imdb
			event.Guid = &guid
		} else if tmdb, ok := item.ProviderIDs["Tmdb"]; ok {
			guid := "tmdb://" + tmdb
			event.Guid = &guid
		}
	}

	// Transcode information
	if session.TranscodingInfo != nil {
		if session.TranscodingInfo.VideoCodec != "" {
			event.VideoCodec = &session.TranscodingInfo.VideoCodec
		}
		if session.TranscodingInfo.AudioCodec != "" {
			event.AudioCodec = &session.TranscodingInfo.AudioCodec
		}
		if session.TranscodingInfo.AudioChannels > 0 {
			audioChannels := fmt.Sprintf("%d", session.TranscodingInfo.AudioChannels)
			event.AudioChannels = &audioChannels
		}
		if session.TranscodingInfo.Width > 0 {
			event.VideoWidth = &session.TranscodingInfo.Width
			event.VideoHeight = &session.TranscodingInfo.Height
		}
	}

	// Video resolution from media streams
	if item.MediaStreams != nil {
		for i := range item.MediaStreams {
			stream := &item.MediaStreams[i]
			if stream.Type != "Video" || stream.Height <= 0 {
				continue
			}
			event.VideoHeight = &stream.Height
			event.VideoWidth = &stream.Width
			var resolution string
			if stream.Height >= 2160 {
				resolution = "4K"
			} else if stream.Height >= 1080 {
				resolution = "1080p"
			} else if stream.Height >= 720 {
				resolution = "720p"
			} else {
				resolution = "SD"
			}
			event.VideoFullResolution = &resolution
			break
		}
	}

	// Rating key (use item ID)
	event.RatingKey = &item.ID

	return event
}
