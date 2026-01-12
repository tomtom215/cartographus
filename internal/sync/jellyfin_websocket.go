// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
jellyfin_websocket.go - Jellyfin WebSocket Client

This file implements a WebSocket client for receiving real-time playback
notifications from Jellyfin media server.

WebSocket Endpoint: ws://{jellyfin_url}/socket?api_key={api_key}
*/

package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"

	"github.com/tomtom215/cartographus/internal/models"
)

// JellyfinWebSocketClient manages WebSocket connection to Jellyfin for real-time notifications
type JellyfinWebSocketClient struct {
	// Connection configuration
	wsURL  string
	apiKey string

	// WebSocket connection
	conn   *websocket.Conn
	connMu sync.RWMutex

	// Lifecycle management
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Callbacks (protected by mutex)
	callbackMu        sync.RWMutex
	onSession         func([]models.JellyfinSession)
	onUserDataChanged func(string, any)
	onPlayStateChange func(string, string) // sessionID, command
}

// JellyfinWSMessage represents a generic WebSocket message
type JellyfinWSMessage struct {
	MessageType string          `json:"MessageType"`
	Data        json.RawMessage `json:"Data,omitempty"`
}

// JellyfinSessionsStartRequest is sent to subscribe to session updates
type JellyfinSessionsStartRequest struct {
	MessageType string `json:"MessageType"`
	Data        string `json:"Data"` // "0,1500" = initial data, update interval in ms
}

// NewJellyfinWebSocketClient creates a new WebSocket client for Jellyfin
func NewJellyfinWebSocketClient(wsURL, apiKey string) *JellyfinWebSocketClient {
	return &JellyfinWebSocketClient{
		wsURL:    wsURL,
		apiKey:   apiKey,
		stopChan: make(chan struct{}),
	}
}

// SetCallbacks registers callback functions for different notification types
func (c *JellyfinWebSocketClient) SetCallbacks(
	onSession func([]models.JellyfinSession),
	onUserDataChanged func(string, any),
	onPlayStateChange func(string, string),
) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()

	c.onSession = onSession
	c.onUserDataChanged = onUserDataChanged
	c.onPlayStateChange = onPlayStateChange
}

// Connect establishes WebSocket connection to Jellyfin
func (c *JellyfinWebSocketClient) Connect(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		return nil // Already connected
	}

	logging.Info().Str("url", c.wsURL).Msg("Connecting")

	dialer := websocket.Dialer{
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: true,
	}

	conn, resp, err := dialer.DialContext(ctx, c.wsURL, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("websocket dial failed (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	if resp != nil && resp.Body != nil {
		if cerr := resp.Body.Close(); cerr != nil {
			logging.Info().Err(cerr).Msg("Warning: failed to close response body")
		}
	}

	c.conn = conn
	logging.Info().Msg("[jellyfin-ws] Connected successfully")

	// Subscribe to sessions
	if err := c.subscribeToSessions(); err != nil {
		logging.Info().Err(err).Msg("Warning: failed to subscribe to sessions")
	}

	// Start background goroutines
	c.wg.Add(2)
	go c.listen(ctx)
	go c.pingLoop(ctx)

	return nil
}

// subscribeToSessions sends the SessionsStart message to receive session updates
func (c *JellyfinWebSocketClient) subscribeToSessions() error {
	// Subscribe with initial data and 1500ms update interval
	msg := JellyfinSessionsStartRequest{
		MessageType: "SessionsStart",
		Data:        "0,1500",
	}

	return c.conn.WriteJSON(msg)
}

// listen processes incoming WebSocket messages
func (c *JellyfinWebSocketClient) listen(ctx context.Context) {
	defer c.wg.Done()

	reconnectDelay := 1 * time.Second
	maxReconnectDelay := 32 * time.Second

	for {
		select {
		case <-ctx.Done():
			logging.Info().Msg("[jellyfin-ws] Listener stopping (context canceled)")
			return
		case <-c.stopChan:
			logging.Info().Msg("[jellyfin-ws] Listener stopping (stop signal)")
			return
		default:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				// Connection lost - attempt reconnect with cancellable wait
				logging.Info().Dur("delay", reconnectDelay).Msg("Connection lost, reconnecting...")
				select {
				case <-time.After(reconnectDelay):
					// Continue with reconnection
				case <-ctx.Done():
					return
				}
				reconnectDelay *= 2
				if reconnectDelay > maxReconnectDelay {
					reconnectDelay = maxReconnectDelay
				}

				if err := c.Connect(ctx); err != nil {
					logging.Info().Err(err).Msg("Reconnection failed")
					continue
				}
				reconnectDelay = 1 * time.Second // Reset on success
				continue
			}

			// Set read deadline
			if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
				logging.Info().Err(err).Msg("Failed to set read deadline")
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					logging.Info().Msg("[jellyfin-ws] Connection closed normally")
				} else if ctx.Err() != nil {
					return
				} else {
					logging.Info().Err(err).Msg("Read error")
				}
				c.closeConnection()
				continue
			}

			reconnectDelay = 1 * time.Second // Reset on successful read
			c.handleMessage(message)
		}
	}
}

// handleMessage processes a single WebSocket message
//
//nolint:gocyclo // Switch statement with multiple message types - complexity is inherent
func (c *JellyfinWebSocketClient) handleMessage(data []byte) {
	var msg JellyfinWSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		logging.Info().Err(err).Msg("Failed to parse message")
		return
	}

	c.callbackMu.RLock()
	defer c.callbackMu.RUnlock()

	switch msg.MessageType {
	case "Sessions":
		// Session updates
		var sessions []models.JellyfinSession
		if err := json.Unmarshal(msg.Data, &sessions); err != nil {
			logging.Info().Err(err).Msg("Failed to parse sessions")
			return
		}
		logging.Info().Int("count", len(sessions)).Msg("Received sessions")
		if c.onSession != nil {
			c.onSession(sessions)
		}

	case "UserDataChanged":
		// User data updates (ratings, watch status)
		if c.onUserDataChanged != nil {
			var data map[string]interface{}
			if err := json.Unmarshal(msg.Data, &data); err == nil {
				userID, ok := data["UserId"].(string)
				if !ok {
					userID = ""
				}
				c.onUserDataChanged(userID, data)
			}
		}

	case "Playstate":
		// Playback state changes (play, pause, stop)
		if c.onPlayStateChange != nil {
			var data map[string]interface{}
			if err := json.Unmarshal(msg.Data, &data); err == nil {
				sessionID, ok := data["SessionId"].(string)
				if !ok {
					sessionID = ""
				}
				command, ok := data["Command"].(string)
				if !ok {
					command = ""
				}
				c.onPlayStateChange(sessionID, command)
			}
		}

	case "ForceKeepAlive":
		// Server requesting keep-alive response
		// The ping loop handles this automatically

	case "KeepAlive":
		// Keep-alive acknowledgment - ignore

	default:
		// Log unknown message types for debugging
		logging.Info().Str("type", msg.MessageType).Msg("Unknown message type")
	}
}

// pingLoop sends periodic keep-alive messages
func (c *JellyfinWebSocketClient) pingLoop(ctx context.Context) {
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

			// Send keep-alive message
			msg := JellyfinWSMessage{
				MessageType: "KeepAlive",
			}

			c.connMu.Lock()
			err := conn.WriteJSON(msg)
			c.connMu.Unlock()

			if err != nil {
				logging.Info().Err(err).Msg("Keep-alive failed")
				c.closeConnection()
				continue
			}
		}
	}
}

// closeConnection safely closes the WebSocket connection
func (c *JellyfinWebSocketClient) closeConnection() {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		// Send close message
		if err := c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(1*time.Second),
		); err != nil {
			logging.Info().Err(err).Msg("Failed to send close message")
		}

		if err := c.conn.Close(); err != nil {
			logging.Info().Err(err).Msg("Failed to close connection")
		}
		c.conn = nil
	}
}

// Close gracefully closes the WebSocket client
func (c *JellyfinWebSocketClient) Close() error {
	logging.Info().Msg("[jellyfin-ws] Closing WebSocket client...")

	// Signal all goroutines to stop
	close(c.stopChan)

	// Close the connection
	c.closeConnection()

	// Wait for goroutines to complete
	c.wg.Wait()

	logging.Info().Msg("[jellyfin-ws] WebSocket client closed")
	return nil
}

// IsConnected returns true if the WebSocket is connected
func (c *JellyfinWebSocketClient) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn != nil
}

// SendMessage sends a message to the WebSocket server
func (c *JellyfinWebSocketClient) SendMessage(msg interface{}) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	return c.conn.WriteJSON(msg)
}
