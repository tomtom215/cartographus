// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
jellyfin_manager.go - Jellyfin Integration Manager

This file provides integration between the sync manager and Jellyfin services.
It handles initialization, lifecycle management, and event processing for
Jellyfin WebSocket and session polling services.
*/

package sync

import (
	"context"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// JellyfinManager orchestrates Jellyfin integration services
type JellyfinManager struct {
	client         JellyfinClientInterface
	wsClient       *JellyfinWebSocketClient
	poller         *JellyfinSessionPoller
	cfg            *config.JellyfinConfig
	eventPublisher EventPublisher
	wsHub          WebSocketHub
	userResolver   UserResolver // For resolving external UUIDs to internal user IDs
}

// NewJellyfinManager creates a new Jellyfin integration manager
//
// Parameters:
//   - cfg: Jellyfin configuration (URL, API key, server_id, etc.)
//   - wsHub: WebSocket hub for real-time frontend updates
//   - userResolver: For mapping Jellyfin UUIDs to internal user IDs (optional, can be nil)
//
// The userResolver enables proper user tracking across multiple servers and sources.
// If nil, user IDs will default to 0 (legacy behavior for backward compatibility).
func NewJellyfinManager(cfg *config.JellyfinConfig, wsHub WebSocketHub, userResolver UserResolver) *JellyfinManager {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	// Use circuit breaker client for resilience against API failures
	client := NewJellyfinCircuitBreakerClient(JellyfinCircuitBreakerConfig{
		BaseURL: cfg.URL,
		APIKey:  cfg.APIKey,
		UserID:  cfg.UserID,
	})

	return &JellyfinManager{
		client:       client,
		cfg:          cfg,
		wsHub:        wsHub,
		userResolver: userResolver,
	}
}

// SetEventPublisher sets the NATS event publisher
func (m *JellyfinManager) SetEventPublisher(publisher EventPublisher) {
	m.eventPublisher = publisher
}

// ServerID returns the configured server ID for this manager.
// Used for multi-server support to identify which server events originate from.
func (m *JellyfinManager) ServerID() string {
	if m == nil || m.cfg == nil {
		return ""
	}
	return m.cfg.ServerID
}

// Start initializes and starts all enabled Jellyfin services
func (m *JellyfinManager) Start(ctx context.Context) error {
	if m == nil {
		return nil
	}

	logging.Info().Msg("[jellyfin] Starting Jellyfin integration...")

	// Test connection
	if err := m.client.Ping(ctx); err != nil {
		logging.Info().Err(err).Msg("WARNING: Ping failed")
		// Don't fail startup - server may become available later
	} else {
		info, err := m.client.GetSystemInfo(ctx)
		if err != nil {
			logging.Info().Err(err).Msg("WARNING: Failed to get system info")
		} else {
			logging.Info().Str("server", info.ServerName).Str("version", info.Version).Msg("Connected")
		}
	}

	// Start WebSocket client if enabled
	if m.cfg.RealtimeEnabled {
		if err := m.startWebSocket(ctx); err != nil {
			logging.Info().Err(err).Msg("WARNING: Failed to start WebSocket")
		}
	}

	// Start session poller if enabled
	if m.cfg.SessionPollingEnabled {
		if err := m.startSessionPoller(ctx); err != nil {
			logging.Info().Err(err).Msg("WARNING: Failed to start session poller")
		}
	}

	logging.Info().Msg("[jellyfin] Jellyfin integration started")
	return nil
}

// startWebSocket initializes and starts the WebSocket client
func (m *JellyfinManager) startWebSocket(ctx context.Context) error {
	wsURL, err := m.client.GetWebSocketURL()
	if err != nil {
		return err
	}

	m.wsClient = NewJellyfinWebSocketClient(wsURL, m.cfg.APIKey)
	m.wsClient.SetCallbacks(
		m.handleSessionUpdate,
		m.handleUserDataChanged,
		m.handlePlayStateChange,
	)

	return m.wsClient.Connect(ctx)
}

// startSessionPoller initializes and starts the session poller
func (m *JellyfinManager) startSessionPoller(ctx context.Context) error {
	interval := m.cfg.SessionPollingInterval
	if interval < 10*time.Second {
		logging.Info().Dur("interval", interval).Msg("WARNING: Polling interval too low, using 10s")
		interval = 10 * time.Second
	}

	config := SessionPollerConfig{
		Interval:       interval,
		PublishAll:     false,
		SeenSessionTTL: 1 * time.Hour,
	}

	m.poller = NewJellyfinSessionPoller(m.client, config)
	m.poller.SetOnSession(m.handleNewSession)

	return m.poller.Start(ctx)
}

// handleSessionUpdate processes session updates from WebSocket
func (m *JellyfinManager) handleSessionUpdate(sessions []models.JellyfinSession) {
	for i := range sessions {
		session := &sessions[i]

		// Only process active sessions
		if !session.IsActive() {
			continue
		}

		// Broadcast to frontend for instant UI updates
		if m.wsHub != nil {
			m.wsHub.BroadcastJSON("jellyfin_session", map[string]interface{}{
				"session_id": session.ID,
				"user":       session.UserName,
				"title":      session.GetContentTitle(),
				"state":      m.getSessionState(session),
				"progress":   session.GetPercentComplete(),
			})
		}

		// Publish to NATS for event processing
		m.publishSession(session)
	}
}

// handleUserDataChanged processes user data changes
func (m *JellyfinManager) handleUserDataChanged(userID string, data any) {
	logging.Info().Str("user_id", userID).Msg("User data changed for user")
	// Can be extended to handle watch status updates
}

// handlePlayStateChange processes playback state changes
func (m *JellyfinManager) handlePlayStateChange(sessionID, command string) {
	logging.Info().Str("session_id", sessionID).Str("command", command).Msg("Playstate change")

	// Broadcast to frontend
	if m.wsHub != nil {
		m.wsHub.BroadcastJSON("jellyfin_playstate", map[string]interface{}{
			"session_id": sessionID,
			"command":    command,
		})
	}
}

// handleNewSession processes new sessions from the poller
func (m *JellyfinManager) handleNewSession(session *models.JellyfinSession) {
	logging.Info().Str("user", session.UserName).Str("title", session.GetContentTitle()).Msg("New session")
	m.publishSession(session)
}

// publishSession converts a session to a PlaybackEvent and publishes to NATS
//
// This method:
//  1. Converts Jellyfin session to PlaybackEvent
//  2. Sets ServerID from configuration for multi-server support
//  3. Resolves external Jellyfin UUID to internal user ID via UserResolver
//  4. Publishes to NATS for event processing and detection
func (m *JellyfinManager) publishSession(session *models.JellyfinSession) {
	if m.eventPublisher == nil {
		return
	}

	event := SessionToPlaybackEvent(session, "")
	if event == nil {
		return
	}

	ctx := context.Background()

	// Set ServerID from configuration for multi-server deduplication
	if m.cfg.ServerID != "" {
		serverID := m.cfg.ServerID
		event.ServerID = &serverID
	}

	// Resolve external Jellyfin UUID to internal user ID
	// This enables consistent user tracking across all sources (Plex, Jellyfin, Emby)
	if m.userResolver != nil && session.UserID != "" {
		internalUserID, err := m.userResolver.ResolveUserID(
			ctx,
			"jellyfin",
			m.cfg.ServerID,
			session.UserID, // Jellyfin uses UUID strings
			&session.UserName,
			&session.UserName, // Use username as friendly name if not available
		)
		if err != nil {
			logging.Info().Str("user_id", session.UserID).Err(err).Msg("Warning: Failed to resolve user ID")
			// Continue with UserID=0 as fallback
		} else {
			event.UserID = internalUserID
		}
	}

	if err := m.eventPublisher.PublishPlaybackEvent(ctx, event); err != nil {
		logging.Info().Err(err).Msg("Failed to publish event")
	}
}

// getSessionState returns the session state string
func (m *JellyfinManager) getSessionState(session *models.JellyfinSession) string {
	if session.IsPaused() {
		return "paused"
	}
	if session.IsPlaying() {
		return "playing"
	}
	return "stopped"
}

// Stop gracefully stops all Jellyfin services
func (m *JellyfinManager) Stop() error {
	if m == nil {
		return nil
	}

	logging.Info().Msg("[jellyfin] Stopping Jellyfin integration...")

	if m.wsClient != nil {
		if err := m.wsClient.Close(); err != nil {
			logging.Info().Err(err).Msg("Error closing WebSocket")
		}
	}

	if m.poller != nil {
		m.poller.Stop()
	}

	logging.Info().Msg("[jellyfin] Jellyfin integration stopped")
	return nil
}
