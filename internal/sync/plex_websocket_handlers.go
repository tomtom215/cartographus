// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"

	"github.com/tomtom215/cartographus/internal/logging"

	"github.com/tomtom215/cartographus/internal/models"
)

// ========================================
// Plex WebSocket Integration (v1.39)
// Manager Methods for WebSocket Handlers
// ========================================

// StartPlexWebSocket initializes and starts the Plex WebSocket client for real-time updates
//
// This method establishes a WebSocket connection to Plex Media Server to receive instant
// notifications about playback state changes, library updates, and server status changes.
//
// Key Features:
//   - Sub-second playback notifications (vs 30-60s Tautulli polling)
//   - Automatic reconnection on connection loss
//   - Deduplication: Only processes sessions Tautulli missed
//   - Broadcasts updates to frontend via WebSocket hub
//
// Deduplication Strategy:
//
//	Priority Order: Tautulli > Plex WebSocket
//	- Checks if sessionKey exists in database
//	- Only uses Plex data if Tautulli doesn't have it
//	- Prevents duplicate events and maintains data quality
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - error: Connection errors, authentication failures, or initialization errors
//
// Thread Safety: Safe for concurrent calls
func (m *Manager) StartPlexWebSocket(ctx context.Context) error {
	if m.plexClient == nil {
		return fmt.Errorf("plex client not initialized")
	}

	// Create Plex WebSocket client
	m.plexWSClient = NewPlexWebSocketClient(m.cfg.Plex.URL, m.cfg.Plex.Token)

	// Register callbacks for different notification types
	m.plexWSClient.SetCallbacks(
		// Playing state changes (CRITICAL: Real-time playback tracking)
		func(notif models.PlexPlayingNotification) {
			m.handleRealtimePlayback(ctx, &notif)
		},
		// Timeline changes (library scans, new content)
		func(notif models.PlexTimelineNotification) {
			m.handleTimelineNotification(&notif)
		},
		// Activity updates (background tasks)
		func(notif models.PlexActivityNotification) {
			m.handleActivityNotification(&notif)
		},
		// Server status changes
		func(notif models.PlexStatusNotification) {
			m.handleStatusNotification(notif)
		},
	)

	// Connect to Plex WebSocket
	if err := m.plexWSClient.Connect(ctx); err != nil {
		return fmt.Errorf("connect plex websocket: %w", err)
	}

	logging.Info().Msg("Plex WebSocket started successfully")
	return nil
}

// handleRealtimePlayback processes real-time playback notifications from Plex
//
// This method:
//  1. Checks if Tautulli already has this session (deduplication)
//  2. If new session, fetches full details from /status/sessions
//  3. Extracts IP address for geolocation
//  4. Publishes event to NATS for detection processing
//  5. Broadcasts update to frontend WebSocket
//
// Deduplication:
//   - Tautulli data always wins (has IP, geolocation, quality metrics)
//   - Plex WebSocket only fills gaps for missed sessions
//
// Detection Integration (ADR-0020):
//   - All playback events are published to NATS for anomaly detection
//   - Detection engine runs rules like impossible travel, concurrent streams
//
// Thread Safety: Safe for concurrent calls (uses database mutex)
func (m *Manager) handleRealtimePlayback(ctx context.Context, notif *models.PlexPlayingNotification) {
	// Log all playback state changes for debugging
	logging.Debug().Str("playback_state", notif.GetPlaybackState()).Str("session", notif.SessionKey).Str("state", notif.State).Str("rating_key", notif.RatingKey).Msg("Plex WebSocket playback")

	// CRITICAL: Buffering detection for performance monitoring
	if notif.IsBuffering() {
		logging.Info().Msg("BUFFERING DETECTED: SessionKey=, RatingKey=")
	}

	// Deduplication: Check if Tautulli already has this session
	exists, err := m.db.SessionKeyExists(ctx, notif.SessionKey)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to check session existence")
		return
	}

	if exists {
		// Tautulli owns this data - skip Plex data for database
		logging.Info().Msg("Session  already exists in Tautulli data - skipping Plex update")

		// Still broadcast state change to frontend for instant UI updates
		if m.wsHub != nil {
			m.wsHub.BroadcastJSON("plex_realtime_playback", map[string]interface{}{
				"session_key":  notif.SessionKey,
				"state":        notif.State,
				"rating_key":   notif.RatingKey,
				"view_offset":  notif.ViewOffset,
				"is_buffering": notif.IsBuffering(),
			})
		}
		return
	}

	// NEW SESSION: Tautulli missed this - use Plex data
	logging.Info().Msg("New session  detected from Plex WebSocket (Tautulli missed)")

	// Fetch full session details from Plex for IP address and user info
	// This is needed for detection rules (impossible travel, geo restrictions)
	m.fetchAndPublishSession(ctx, notif.SessionKey)

	// Broadcast to frontend for instant map update
	if m.wsHub != nil {
		m.wsHub.BroadcastJSON("plex_realtime_playback", map[string]interface{}{
			"session_key":       notif.SessionKey,
			"state":             notif.State,
			"rating_key":        notif.RatingKey,
			"view_offset":       notif.ViewOffset,
			"transcode_session": notif.TranscodeSession,
			"is_buffering":      notif.IsBuffering(),
			"is_new_session":    true, // Flag for frontend to handle differently
		})
	}
}

// fetchAndPublishSession fetches full session data from Plex and publishes to NATS.
// This enables detection of security anomalies (impossible travel, concurrent streams)
// for sessions that Tautulli missed.
//
// The method runs asynchronously to avoid blocking WebSocket event handling.
func (m *Manager) fetchAndPublishSession(ctx context.Context, sessionKey string) {
	// Run asynchronously to not block WebSocket handler
	go func() {
		if m.plexClient == nil {
			return
		}

		// Fetch all active sessions from Plex
		sessions, err := m.plexClient.GetTranscodeSessions(ctx)
		if err != nil {
			logging.Error().Err(err).Msg("Failed to fetch sessions from Plex")
			return
		}

		// Find the session we're looking for
		var targetSession *models.PlexSession
		for i := range sessions {
			if sessions[i].SessionKey == sessionKey {
				targetSession = &sessions[i]
				break
			}
		}

		if targetSession == nil {
			logging.Info().Msg("Session  not found in Plex sessions - may have ended")
			return
		}

		// Create PlaybackEvent from session data
		event := m.plexSessionToPlaybackEvent(ctx, targetSession)
		if event == nil {
			return
		}

		// Publish to NATS for detection processing
		m.publishEvent(ctx, event)
		logging.Info().Msg("Published Plex session  to NATS for detection")
	}()
}

// plexSessionToPlaybackEvent converts a PlexSession to PlaybackEvent for NATS publishing.
// Returns nil if the session doesn't have sufficient data.
//
// This method:
//  1. Creates a PlaybackEvent from Plex session data
//  2. Sets ServerID from configuration for multi-server deduplication
//  3. Resolves Plex user ID to internal user ID via UserResolver (if available)
//  4. Populates media, user, and player information
//
// The context is used for user ID resolution timeouts and cancellation.
func (m *Manager) plexSessionToPlaybackEvent(ctx context.Context, session *models.PlexSession) *models.PlaybackEvent {
	if session == nil {
		return nil
	}

	// Create ServerID pointer if configured
	var serverIDPtr *string
	if m.cfg.Plex.ServerID != "" {
		serverID := m.cfg.Plex.ServerID
		serverIDPtr = &serverID
	}

	event := &models.PlaybackEvent{
		Source:     "plex",
		ServerID:   serverIDPtr,
		SessionKey: session.SessionKey,
		MediaType:  session.Type,
		Title:      session.Title,
	}

	m.populateMediaInfo(event, session)
	m.populateUserInfo(event, session)
	m.populatePlayerInfo(event, session)
	m.populateQualityMetrics(event, session)

	// Resolve Plex user ID to internal user ID for cross-source consistency
	// Plex user IDs are integers but may differ across servers, so we map them
	if m.userResolver != nil && session.User != nil && session.User.ID > 0 {
		externalUserID := fmt.Sprintf("%d", session.User.ID)
		internalUserID, err := m.userResolver.ResolveUserID(
			ctx,
			"plex",
			m.cfg.Plex.ServerID,
			externalUserID,
			&session.User.Title,
			&session.User.Title, // Use title as friendly name
		)
		if err != nil {
			logging.Info().Str("user_id", externalUserID).Err(err).Msg("Warning: Failed to resolve user ID")
			// Continue with original Plex user ID as fallback
		} else {
			event.UserID = internalUserID
		}
	}

	return event
}

// populateMediaInfo adds media metadata to the event.
func (m *Manager) populateMediaInfo(event *models.PlaybackEvent, session *models.PlexSession) {
	if session.RatingKey != "" {
		event.RatingKey = &session.RatingKey
	}
	if session.ParentTitle != "" {
		event.ParentTitle = &session.ParentTitle
	}
	if session.GrandparentTitle != "" {
		event.GrandparentTitle = &session.GrandparentTitle
	}
}

// populateUserInfo adds user information to the event.
func (m *Manager) populateUserInfo(event *models.PlaybackEvent, session *models.PlexSession) {
	if session.User == nil {
		return
	}
	event.UserID = session.User.ID
	event.Username = session.User.Title
	if session.User.Thumb != "" {
		event.UserThumb = &session.User.Thumb
	}
}

// populatePlayerInfo adds player/device information to the event.
func (m *Manager) populatePlayerInfo(event *models.PlaybackEvent, session *models.PlexSession) {
	if session.Player == nil {
		return
	}

	player := session.Player
	event.IPAddress = player.Address
	event.Platform = player.Platform
	event.Player = player.Title

	// Optional device identifiers
	if player.MachineID != "" {
		event.MachineID = &player.MachineID
	}
	if player.Device != "" {
		event.Device = &player.Device
	}
	if player.Product != "" {
		event.Product = &player.Product
	}
	if player.Version != "" {
		event.ProductVersion = &player.Version
	}
	if player.PlatformVersion != "" {
		event.PlatformVersion = &player.PlatformVersion
	}

	// Connection type
	if player.Local {
		event.LocationType = "lan"
		local := 1
		event.Local = &local
	} else {
		event.LocationType = "wan"
		local := 0
		event.Local = &local
	}

	// Security flags
	if player.Relayed {
		relayed := 1
		event.Relayed = &relayed
	}
	if player.Secure {
		secure := 1
		event.Secure = &secure
	}
}

// populateQualityMetrics adds quality and streaming metrics to the event.
// This includes transcode decisions, video/audio codecs, resolution, and hardware acceleration.
// CRITICAL for standalone mode analytics where Tautulli is not available.
func (m *Manager) populateQualityMetrics(event *models.PlaybackEvent, session *models.PlexSession) {
	// Extract source media quality from Media array
	if len(session.Media) > 0 {
		populateSourceMediaQuality(event, &session.Media[0])
	}

	// Extract transcode session details (if transcoding)
	ts := session.TranscodeSession
	if ts == nil {
		// Direct play - no transcode session
		directPlay := "direct play"
		event.TranscodeDecision = &directPlay
		return
	}

	// Populate transcode-related fields
	populateTranscodeDecisions(event, ts)
	populateTranscodeOutputQuality(event, ts)
	populateTranscodeHardwareAccel(event, ts)
	populateTranscodeProgressMetrics(event, ts)
}

// populateSourceMediaQuality extracts source media quality from Plex Media object.
func populateSourceMediaQuality(event *models.PlaybackEvent, media *models.PlexMedia) {
	if media.VideoCodec != "" {
		event.VideoCodec = &media.VideoCodec
	}
	if media.AudioCodec != "" {
		event.AudioCodec = &media.AudioCodec
	}
	if media.VideoResolution != "" {
		event.VideoResolution = &media.VideoResolution
	}
	if media.Bitrate > 0 {
		event.Bitrate = &media.Bitrate
		event.SourceBitrate = &media.Bitrate
	}
	if media.Width > 0 {
		event.VideoWidth = &media.Width
	}
	if media.Height > 0 {
		event.VideoHeight = &media.Height
	}
	if media.AudioChannels > 0 {
		channels := fmt.Sprintf("%d", media.AudioChannels)
		event.AudioChannels = &channels
	}
	if media.Container != "" {
		event.Container = &media.Container
	}
	if media.VideoFrameRate != "" {
		event.VideoFrameRate = &media.VideoFrameRate
	}
	if media.VideoProfile != "" {
		event.VideoProfile = &media.VideoProfile
	}
	if media.AudioProfile != "" {
		event.AudioProfile = &media.AudioProfile
	}
}

// populateTranscodeDecisions extracts transcode decisions and source codecs.
func populateTranscodeDecisions(event *models.PlaybackEvent, ts *models.PlexTranscodeSession) {
	// Individual decisions
	if ts.VideoDecision != "" {
		event.VideoDecision = &ts.VideoDecision
	}
	if ts.AudioDecision != "" {
		event.AudioDecision = &ts.AudioDecision
	}
	if ts.SubtitleDecision != "" {
		event.SubtitleDecision = &ts.SubtitleDecision
	}

	// Determine overall transcode decision
	decision := determineOverallTranscodeDecision(ts.VideoDecision, ts.AudioDecision)
	event.TranscodeDecision = &decision

	// Source codecs (override media codecs if present in transcode session)
	if ts.SourceVideoCodec != "" {
		event.VideoCodec = &ts.SourceVideoCodec
	}
	if ts.SourceAudioCodec != "" {
		event.AudioCodec = &ts.SourceAudioCodec
	}
}

// determineOverallTranscodeDecision returns the overall transcode decision based on video/audio decisions.
func determineOverallTranscodeDecision(videoDecision, audioDecision string) string {
	if videoDecision == "transcode" || audioDecision == "transcode" {
		return "transcode"
	}
	if videoDecision == "copy" || audioDecision == "copy" {
		return "copy"
	}
	return "direct play"
}

// populateTranscodeOutputQuality extracts transcoded output quality metrics.
func populateTranscodeOutputQuality(event *models.PlaybackEvent, ts *models.PlexTranscodeSession) {
	if ts.VideoCodec != "" {
		event.TranscodeVideoCodec = &ts.VideoCodec
	}
	if ts.AudioCodec != "" {
		event.TranscodeAudioCodec = &ts.AudioCodec
	}
	if ts.Width > 0 {
		event.TranscodeVideoWidth = &ts.Width
	}
	if ts.Height > 0 {
		event.TranscodeVideoHeight = &ts.Height
	}
	if ts.AudioChannels > 0 {
		event.TranscodeAudioChannels = &ts.AudioChannels
	}
	if ts.Container != "" {
		event.TranscodeContainer = &ts.Container
	}
	if ts.Protocol != "" {
		event.StreamContainer = &ts.Protocol
	}

	// Calculate stream bitrate from size and duration
	if ts.Size > 0 && ts.Duration > 0 {
		bitrateKbps := int(ts.Size * 8 / ts.Duration)
		event.StreamBitrate = &bitrateKbps
		event.TranscodeBitrate = &bitrateKbps
	}
}

// populateTranscodeHardwareAccel extracts hardware acceleration status.
func populateTranscodeHardwareAccel(event *models.PlaybackEvent, ts *models.PlexTranscodeSession) {
	if ts.TranscodeHwRequested {
		hw := 1
		event.TranscodeHWRequested = &hw
	}
	if ts.TranscodeHwDecoding != "" {
		hw := 1
		event.TranscodeHWDecoding = &hw
		event.TranscodeHWDecode = &ts.TranscodeHwDecoding
	}
	if ts.TranscodeHwEncoding != "" {
		hw := 1
		event.TranscodeHWEncoding = &hw
		event.TranscodeHWEncode = &ts.TranscodeHwEncoding
	}
	if ts.TranscodeHwFullPipe {
		hw := 1
		event.TranscodeHWFullPipeline = &hw
	}
}

// populateTranscodeProgressMetrics extracts transcode progress and performance metrics.
func populateTranscodeProgressMetrics(event *models.PlaybackEvent, ts *models.PlexTranscodeSession) {
	if ts.Progress > 0 {
		progress := int(ts.Progress)
		event.TranscodeProgress = &progress
	}
	if ts.Speed > 0 {
		speed := fmt.Sprintf("%.1fx", ts.Speed)
		event.TranscodeSpeed = &speed
	}
	if ts.Throttled {
		throttled := 1
		event.TranscodeThrottled = &throttled
	}
	if ts.Key != "" {
		event.TranscodeKey = &ts.Key
	}
}

// handleTimelineNotification processes library content changes
//
// Use Cases:
//   - Library scan progress monitoring
//   - New content detection alerts
//   - Metadata update tracking
func (m *Manager) handleTimelineNotification(notif *models.PlexTimelineNotification) {
	// Only log significant state changes (not every update)
	if notif.State == 6 { // Analyzing
		logging.Info().Msg("Library update:  (section ) - analyzing")
	} else if notif.State == 9 { // Deleted
		logging.Info().Msg("Content deleted:  (section )")
	}

	// Broadcast to frontend if WebSocket hub available
	if m.wsHub != nil && notif.State == 6 {
		m.wsHub.BroadcastJSON("library_scan_progress", map[string]interface{}{
			"section_id":     notif.SectionID,
			"title":          notif.Title,
			"state":          notif.State,
			"metadata_state": notif.MetadataState,
		})
	}
}

// handleActivityNotification processes background task updates
//
// Use Cases:
//   - Library scan progress (0-100%)
//   - Database optimization progress
//   - Scheduled maintenance tracking
func (m *Manager) handleActivityNotification(notif *models.PlexActivityNotification) {
	// Log task progress
	if notif.Activity.Progress > 0 {
		logging.Info().Msg("Plex activity:  -  complete")
	}

	// Broadcast progress updates to frontend
	if m.wsHub != nil && notif.Activity.Progress > 0 {
		m.wsHub.BroadcastJSON("background_task_progress", map[string]interface{}{
			"uuid":     notif.UUID,
			"type":     notif.Activity.Type,
			"title":    notif.Activity.Title,
			"subtitle": notif.Activity.Subtitle,
			"progress": notif.Activity.Progress,
			"context":  notif.Activity.Context,
		})
	}
}

// handleStatusNotification processes server status changes
//
// Use Cases:
//   - Server shutdown alerts (maintenance windows)
//   - Server restart detection
//   - Uptime monitoring
func (m *Manager) handleStatusNotification(notif models.PlexStatusNotification) {
	// Log server status changes
	logging.Info().Msg("Plex server status:  -")

	// Broadcast critical alerts to frontend
	if m.wsHub != nil && notif.NotificationName == "SERVER_SHUTDOWN" {
		m.wsHub.BroadcastJSON("server_alert", map[string]interface{}{
			"severity":    "critical",
			"title":       notif.Title,
			"description": notif.Description,
			"type":        notif.NotificationName,
		})
	}
}
