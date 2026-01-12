// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
plex_sessions.go - Plex Session and Transcode Monitoring

This file provides methods for monitoring active playback sessions and
transcode operations on Plex Media Server.

Session Monitoring (v1.39):
  - GetTranscodeSessions(): Active sessions with transcode details
  - GetSessions(): Full session response for API endpoints
  - GetTranscodeSessionsDetailed(): Dedicated transcode endpoint

Buffer Health Monitoring (v1.41):
  - GetSessionTimeline(): Real-time timeline data including:
  - Buffer availability (maxOffsetAvailable, minOffsetAvailable)
  - Current playback position (viewOffset)
  - Transcode progress and speed
  - Session state (playing, paused, buffering)

Transcode Control:
  - CancelTranscode(): Cancel an active transcode session

Hardware Acceleration Detection:
The transcode sessions include hardware acceleration status:
  - Quick Sync (Intel)
  - NVENC (NVIDIA)
  - VAAPI (Linux)
  - VideoToolbox (macOS)

Polling Recommendations:
  - Session monitoring: 5-10 seconds
  - Buffer health: 5 seconds
  - Transcode progress: 5-10 seconds
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"net/http"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetTranscodeSessions retrieves active transcode sessions from Plex Media Server
//
// This endpoint returns currently active playback sessions with comprehensive transcode details:
//   - Transcode progress and speed (real-time monitoring)
//   - Hardware acceleration status (Quick Sync, NVENC, VAAPI, VideoToolbox)
//   - Quality metrics (source vs delivered resolution/codec)
//   - User and device information
//   - Transcode decision reasoning (bandwidth, device, codec support)
//
// Endpoint: GET /status/sessions
// Polling Interval: Recommended 5-10 seconds (configurable)
func (c *PlexClient) GetTranscodeSessions(ctx context.Context) ([]models.PlexSession, error) {
	var sessionsResp models.PlexSessionsResponse
	if err := c.doJSONRequest(ctx, "/status/sessions", &sessionsResp); err != nil {
		return nil, err
	}

	// Return sessions array (empty array if no active sessions)
	if sessionsResp.MediaContainer.Metadata == nil {
		return []models.PlexSession{}, nil
	}

	return sessionsResp.MediaContainer.Metadata, nil
}

// GetSessionTimeline retrieves timeline data for a specific session (Phase 2.1: Buffer Health Monitoring v1.41)
//
// This endpoint provides real-time playback timeline information including critical buffer health data:
//   - Buffer availability (maxOffsetAvailable, minOffsetAvailable)
//   - Current playback position (viewOffset)
//   - Transcode progress and speed
//   - Session state (playing, paused, buffering)
//
// Endpoint: GET /status/sessions (filtered by sessionKey in response)
// Polling Interval: Recommended 5 seconds for buffer health monitoring
func (c *PlexClient) GetSessionTimeline(ctx context.Context, sessionKey string) (*models.PlexSessionTimelineResponse, error) {
	// Plex API doesn't have a direct /status/sessions/{sessionKey}/timeline endpoint
	// Instead, we fetch all sessions and filter by sessionKey
	var sessionsResp models.PlexSessionsResponse
	if err := c.doJSONRequest(ctx, "/status/sessions", &sessionsResp); err != nil {
		return nil, err
	}

	// Filter sessions by sessionKey
	var matchingMetadata []models.PlexTimelineMetadata
	for i := range sessionsResp.MediaContainer.Metadata {
		session := &sessionsResp.MediaContainer.Metadata[i] // Use pointer to avoid copying 224 bytes per iteration
		if session.SessionKey == sessionKey {
			// Convert PlexSession to PlexTimelineMetadata
			timeline := models.PlexTimelineMetadata{
				SessionKey: session.SessionKey,
				Key:        session.Key,
				Type:       session.Type,
				Title:      session.Title,
				ViewOffset: session.ViewOffset,
				Duration:   session.Duration,
			}

			// Copy transcode session data if present
			if session.TranscodeSession != nil {
				timeline.TranscodeSession = &models.PlexTranscodeSessionTimeline{
					MaxOffsetAvailable: session.TranscodeSession.MaxOffsetAvailable,
					MinOffsetAvailable: session.TranscodeSession.MinOffsetAvailable,
					Progress:           session.TranscodeSession.Progress,
					Speed:              session.TranscodeSession.Speed,
					Throttled:          session.TranscodeSession.Throttled,
					Complete:           session.TranscodeSession.Complete,
					Key:                session.TranscodeSession.Key,
				}
			}

			matchingMetadata = append(matchingMetadata, timeline)
			break // Found the session, stop searching
		}
	}

	// Return timeline response
	return &models.PlexSessionTimelineResponse{
		MediaContainer: models.PlexTimelineContainer{
			Size:     len(matchingMetadata),
			Metadata: matchingMetadata,
		},
	}, nil
}

// GetSessions retrieves active playback sessions from Plex Media Server
//
// This is an alias for GetTranscodeSessions, returning the full session response.
// Useful for the /api/v1/plex/sessions endpoint.
//
// Endpoint: GET /status/sessions
func (c *PlexClient) GetSessions(ctx context.Context) (*models.PlexSessionsResponse, error) {
	var sessionsResp models.PlexSessionsResponse
	if err := c.doJSONRequest(ctx, "/status/sessions", &sessionsResp); err != nil {
		return nil, err
	}
	return &sessionsResp, nil
}

// GetTranscodeSessionsDetailed retrieves active transcode sessions with detailed metrics
//
// Endpoint: GET /transcode/sessions
func (c *PlexClient) GetTranscodeSessionsDetailed(ctx context.Context) (*models.PlexTranscodeSessionsResponse, error) {
	var transcodeResp models.PlexTranscodeSessionsResponse
	if err := c.doJSONRequest(ctx, "/transcode/sessions", &transcodeResp); err != nil {
		return nil, err
	}
	return &transcodeResp, nil
}

// CancelTranscode cancels an active transcode session
//
// Endpoint: DELETE /transcode/sessions/{sessionKey}
func (c *PlexClient) CancelTranscode(ctx context.Context, sessionKey string) error {
	return c.doRequest(ctx, requestConfig{
		method:      http.MethodDelete,
		path:        "/transcode/sessions/" + sessionKey,
		expectNoErr: true,
	}, nil)
}
