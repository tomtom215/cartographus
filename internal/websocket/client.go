// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package websocket

import (
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tomtom215/cartographus/internal/logging"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512 KB
)

// clientIDCounter generates unique, monotonically increasing IDs for clients.
// DETERMINISM: This ensures clients can be sorted in a consistent order for
// broadcast operations, eliminating non-deterministic map iteration order.
var clientIDCounter atomic.Uint64

// Client is a middleman between the websocket connection and the hub
type Client struct {
	// id is a unique identifier for this client, used for deterministic ordering.
	// DETERMINISM: Assigned from an atomic counter to ensure consistent sorting.
	id   uint64
	hub  *Hub
	conn *websocket.Conn
	send chan Message
}

// NewClient creates a new Client with a unique deterministic ID
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		id:   clientIDCounter.Add(1),
		hub:  hub,
		conn: conn,
		send: make(chan Message, 256),
	}
}

// ID returns the client's unique identifier for deterministic ordering
func (c *Client) ID() uint64 {
	return c.id
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister <- c
		_ = c.conn.Close() // Explicitly ignore error - best-effort cleanup
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		logging.Error().Err(err).Msg("failed to set read deadline")
		return
	}

	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.Error().Err(err).Msg("unexpected websocket close error")
			}
			break
		}

		// Handle client messages (ping/pong, etc.)
		if msg.Type == MessageTypePing {
			pong := Message{
				Type: MessageTypePong,
				Data: nil,
			}
			select {
			case c.send <- pong:
			default:
			}
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close() // Explicitly ignore error - best-effort cleanup
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logging.Error().Err(err).Msg("failed to set write deadline")
				return
			}

			if !ok {
				// The hub closed the channel
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					logging.Error().Err(err).Msg("failed to write close message")
				}
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				logging.Error().Err(err).Msg("failed to write JSON message")
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logging.Error().Err(err).Msg("failed to set write deadline for ping")
				return
			}

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Start begins reading and writing for the client
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}
