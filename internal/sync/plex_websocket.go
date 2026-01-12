// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"

	"github.com/tomtom215/cartographus/internal/models"
)

// PlexWebSocketClient handles real-time event stream from Plex Media Server
//
// This client connects to Plex's WebSocket endpoint (/:/websockets/notifications)
// to receive instant notifications about playback state changes, library updates,
// and server status changes.
//
// Key Features:
//   - Automatic reconnection with exponential backoff
//   - Thread-safe callback registration
//   - Graceful shutdown handling
//   - Ping/pong keepalive (30-second interval)
//
// Architecture:
//   - Primary data source: Tautulli (has IP addresses, geolocation, quality metrics)
//   - Plex WebSocket: Fills gaps with instant updates for missed sessions
//
// Deduplication Strategy:
//   - Check sessionKey exists in database before processing
//   - If Tautulli already has the session, skip Plex data
//   - Only use Plex data for sessions Tautulli missed
type PlexWebSocketClient struct {
	baseURL string // Plex server URL (http://localhost:32400)
	token   string // X-Plex-Token for authentication

	conn     *websocket.Conn
	connMu   sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Callbacks for different event types (thread-safe)
	callbackMu sync.RWMutex
	onPlaying  func(models.PlexPlayingNotification)
	onTimeline func(models.PlexTimelineNotification)
	onActivity func(models.PlexActivityNotification)
	onStatus   func(models.PlexStatusNotification)
}

// NewPlexWebSocketClient creates a new Plex WebSocket client
//
// Parameters:
//   - baseURL: Plex Media Server URL (e.g., "http://localhost:32400")
//   - token: X-Plex-Token for authentication
//
// Returns initialized client (not yet connected - call Connect)
func NewPlexWebSocketClient(baseURL, token string) *PlexWebSocketClient {
	return &PlexWebSocketClient{
		baseURL:  baseURL,
		token:    token,
		stopChan: make(chan struct{}),
	}
}

// Connect establishes WebSocket connection to Plex server
//
// This method:
//  1. Constructs WebSocket URL with authentication token
//  2. Establishes connection with 10-second timeout
//  3. Starts background goroutines for reading messages and ping/pong
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - error: Connection errors, authentication failures, or dial errors
//
// Thread Safety: Safe for concurrent calls (uses mutex)
func (c *PlexWebSocketClient) Connect(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	// If already connected, return
	if c.conn != nil {
		return nil
	}

	// Build WebSocket URL
	wsURL, err := c.buildWebSocketURL()
	if err != nil {
		return fmt.Errorf("build websocket url: %w", err)
	}

	// Configure WebSocket dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		// Enable compression for large payloads
		EnableCompression: true,
	}

	// Establish connection
	conn, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			return fmt.Errorf("websocket dial failed (HTTP %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("websocket dial: %w", err)
	}

	c.conn = conn
	logging.Info().Msg("Plex WebSocket connected to")

	// Start message listener goroutine
	c.wg.Add(1)
	go c.listen(ctx)

	// Start ping/pong keepalive goroutine
	c.wg.Add(1)
	go c.pingLoop(ctx)

	return nil
}

// buildWebSocketURL constructs the Plex WebSocket URL with authentication
//
// Format: ws://{host}:{port}/:/websockets/notifications?X-Plex-Token={token}
//
// Handles:
//   - HTTP → WS protocol conversion
//   - HTTPS → WSS protocol conversion
//   - Authentication token injection
func (c *PlexWebSocketClient) buildWebSocketURL() (string, error) {
	// Parse base URL
	parsedURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}

	// Convert HTTP(S) to WS(S)
	scheme := "ws"
	if parsedURL.Scheme == "https" {
		scheme = "wss"
	}

	// Build WebSocket URL
	wsURL := fmt.Sprintf("%s://%s/:/websockets/notifications", scheme, parsedURL.Host)

	// Add authentication token as query parameter
	parsedWS, err := url.Parse(wsURL)
	if err != nil {
		return "", fmt.Errorf("parse ws url: %w", err)
	}

	q := parsedWS.Query()
	q.Set("X-Plex-Token", c.token)
	parsedWS.RawQuery = q.Encode()

	return parsedWS.String(), nil
}

// listen processes incoming WebSocket messages in a background goroutine
//
// This method:
//  1. Reads messages from WebSocket connection
//  2. Parses JSON notification container
//  3. Routes notifications to registered callbacks
//  4. Implements automatic reconnection on errors
//
// Reconnection Strategy:
//   - Exponential backoff: 1s, 2s, 4s, 8s, 16s, max 32s
//   - Unlimited retry attempts (runs until context canceled or stopChan closed)
//   - Cleans up old connection before reconnecting
func (c *PlexWebSocketClient) listen(ctx context.Context) {
	defer c.wg.Done()

	reconnectDelay := 1 * time.Second
	maxReconnectDelay := 32 * time.Second

	for {
		select {
		case <-ctx.Done():
			logging.Info().Msg("Plex WebSocket listener stopping (context canceled)")
			return
		case <-c.stopChan:
			logging.Info().Msg("Plex WebSocket listener stopping (stop signal received)")
			return
		default:
			// Read message from WebSocket
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				// Connection lost - attempt reconnect with cancellable wait
				logging.Info().Msg("Plex WebSocket connection lost, reconnecting in ...")
				select {
				case <-time.After(reconnectDelay):
					// Continue with reconnection
				case <-ctx.Done():
					return
				}

				// Exponential backoff
				reconnectDelay *= 2
				if reconnectDelay > maxReconnectDelay {
					reconnectDelay = maxReconnectDelay
				}

				// Attempt reconnection
				if err := c.Connect(ctx); err != nil {
					logging.Error().Err(err).Msg("Plex WebSocket reconnection failed")
					continue
				}

				// Reset reconnect delay on successful connection
				reconnectDelay = 1 * time.Second
				continue
			}

			// Read message with timeout
			if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
				logging.Info().Msg("Plex WebSocket: failed to set read deadline:")
				// Continue reading despite deadline error
			}
			_, message, err := conn.ReadMessage()
			if err != nil {
				// Check if error is due to normal closure
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					logging.Info().Msg("Plex WebSocket closed normally")
					c.closeConnection()
					continue
				}

				// Check if error is due to context cancellation
				if ctx.Err() != nil {
					return
				}

				// Connection error - close and reconnect
				logging.Info().Msg("Plex WebSocket read error:")
				c.closeConnection()
				continue
			}

			// Reset reconnect delay on successful message read
			reconnectDelay = 1 * time.Second

			// Handle message
			c.handleMessage(message)
		}
	}
}

// handleMessage parses and routes a WebSocket message to appropriate callbacks
//
// Message Format:
//
//	{
//	  "NotificationContainer": {
//	    "type": "playing",
//	    "PlaySessionStateNotification": [...]
//	  }
//	}
//
// Routes to callbacks based on notification type:
//   - "playing" → onPlaying callback
//   - "timeline" → onTimeline callback
//   - "activity" → onActivity callback
//   - "status" → onStatus callback
//
//nolint:gocyclo // Message routing requires checking multiple notification types
func (c *PlexWebSocketClient) handleMessage(data []byte) {
	var wrapper models.PlexNotificationWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		logging.Error().Err(err).Msg("Failed to parse Plex notification")
		return
	}

	container := wrapper.NotificationContainer

	// Route notification to appropriate callback
	c.callbackMu.RLock()
	defer c.callbackMu.RUnlock()

	switch container.Type {
	case "playing":
		if c.onPlaying != nil && len(container.PlaySessionStateNotification) > 0 {
			for i := range container.PlaySessionStateNotification {
				c.onPlaying(container.PlaySessionStateNotification[i])
			}
		}

	case "timeline":
		if c.onTimeline != nil && len(container.TimelineEntry) > 0 {
			for i := range container.TimelineEntry {
				c.onTimeline(container.TimelineEntry[i])
			}
		}

	case "activity":
		if c.onActivity != nil && len(container.ActivityNotification) > 0 {
			for i := range container.ActivityNotification {
				c.onActivity(container.ActivityNotification[i])
			}
		}

	case "status":
		if c.onStatus != nil && len(container.StatusNotification) > 0 {
			for _, notif := range container.StatusNotification {
				c.onStatus(notif)
			}
		}

	default:
		// Unknown notification type - log and ignore
		logging.Info().Msg("Unknown Plex notification type:")
	}
}

// pingLoop sends ping messages to keep connection alive
//
// Plex WebSocket requires periodic ping messages to detect dead connections.
// This goroutine sends pings every 30 seconds and waits for pong responses.
//
// Timeout: 60 seconds (if no pong received, connection is considered dead)
func (c *PlexWebSocketClient) pingLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				continue
			}

			// Send ping with 10-second write deadline
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				logging.Info().Msg("Plex WebSocket ping failed:")
				c.closeConnection()
				continue
			}

			logging.Info().Msg("Plex WebSocket ping sent")
		}
	}
}

// closeConnection closes the WebSocket connection and cleans up resources
//
// Thread Safety: Safe for concurrent calls (uses mutex)
func (c *PlexWebSocketClient) closeConnection() {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		// Send close message
		if err := c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(1*time.Second),
		); err != nil {
			logging.Info().Msg("Plex WebSocket: failed to send close message:")
		}
		if err := c.conn.Close(); err != nil {
			logging.Info().Msg("Plex WebSocket: failed to close connection:")
		}
		c.conn = nil
		logging.Info().Msg("Plex WebSocket connection closed")
	}
}

// SetCallbacks registers event handlers for different notification types
//
// Callbacks are invoked when matching notifications are received.
// All callbacks are optional (pass nil to ignore notification type).
//
// Parameters:
//   - onPlaying: Called for playback state changes (playing, paused, stopped, buffering)
//   - onTimeline: Called for content metadata changes (library scans, new content)
//   - onActivity: Called for background task updates (progress, completion)
//   - onStatus: Called for server status changes (shutdown, restart)
//
// Thread Safety: Safe for concurrent calls (uses mutex)
//
// Example:
//
//	client.SetCallbacks(
//	    func(notif models.PlexPlayingNotification) {
//	        log.Printf("Playback: %s - %s", notif.SessionKey, notif.State)
//	    },
//	    nil, // Ignore timeline
//	    nil, // Ignore activity
//	    nil, // Ignore status
//	)
func (c *PlexWebSocketClient) SetCallbacks(
	onPlaying func(models.PlexPlayingNotification),
	onTimeline func(models.PlexTimelineNotification),
	onActivity func(models.PlexActivityNotification),
	onStatus func(models.PlexStatusNotification),
) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()

	c.onPlaying = onPlaying
	c.onTimeline = onTimeline
	c.onActivity = onActivity
	c.onStatus = onStatus
}

// Close gracefully shuts down the WebSocket client
//
// This method:
//  1. Signals all goroutines to stop
//  2. Closes WebSocket connection
//  3. Waits for goroutines to finish
//
// Thread Safety: Safe for concurrent calls
//
// Returns:
//   - error: Always returns nil (reserved for future use)
func (c *PlexWebSocketClient) Close() error {
	// Signal goroutines to stop
	close(c.stopChan)

	// Close connection
	c.closeConnection()

	// Wait for goroutines to finish
	c.wg.Wait()

	logging.Info().Msg("Plex WebSocket client shut down")
	return nil
}

// IsConnected returns true if WebSocket connection is established
//
// Thread Safety: Safe for concurrent calls (uses RLock)
func (c *PlexWebSocketClient) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn != nil
}
