// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package websocket

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// ShutdownReason identifies why the hub is shutting down.
// This enables clear observability in logs and metrics.
type ShutdownReason string

const (
	// ShutdownReasonContextCanceled indicates the parent context was canceled.
	// This is the normal graceful shutdown path (e.g., SIGTERM).
	ShutdownReasonContextCanceled ShutdownReason = "context_canceled"

	// ShutdownReasonContextDeadline indicates the context deadline was exceeded.
	// This may indicate a hung operation during shutdown.
	ShutdownReasonContextDeadline ShutdownReason = "context_deadline"
)

// Message types for WebSocket communication
const (
	MessageTypePlayback       = "playback"
	MessageTypePing           = "ping"
	MessageTypePong           = "pong"
	MessageTypeSyncCompleted  = "sync_completed"
	MessageTypeStatsUpdate    = "stats_update"
	MessageTypeDetectionAlert = "detection_alert"
	MessageTypeSyncProgress   = "sync_progress"
)

// Message represents a WebSocket message
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan Message, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run starts the hub (blocks forever, no context support).
//
// Deprecated: Use RunWithContext for supervised operation.
//
// DETERMINISM: Uses priority-based selection to ensure predictable behavior:
// - Priority 1: Client lifecycle events (Register/Unregister)
// - Priority 2: Broadcast messages
// This ensures client state is always consistent before processing messages.
func (h *Hub) Run() {
	for {
		// DETERMINISM: Priority-based selection prevents non-deterministic
		// ordering when multiple channels are ready simultaneously.
		// When Go's select has multiple ready channels, it picks randomly.
		// Priority selection ensures consistent, predictable behavior.

		// Priority 1: Handle client lifecycle events first (non-blocking check)
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logging.Info().Int("total_clients", len(h.clients)).Msg("websocket client connected")
			continue
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			logging.Info().Int("total_clients", len(h.clients)).Msg("websocket client disconnected")
			continue
		default:
			// No lifecycle events pending, proceed to broadcast
		}

		// Priority 2: Handle broadcast messages (blocking wait)
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logging.Info().Int("total_clients", len(h.clients)).Msg("websocket client connected")

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			logging.Info().Int("total_clients", len(h.clients)).Msg("websocket client disconnected")

		case message := <-h.broadcast:
			h.broadcastToClients(message)
		}
	}
}

// RunWithContext starts the hub with context support for graceful shutdown.
// This method is designed for use with suture supervision.
//
// When the context is canceled:
//  1. All connected clients are gracefully closed
//  2. The method returns ctx.Err()
//
// This allows the hub to be restarted by a supervisor without leaving
// orphaned connections.
//
// DETERMINISM: Uses priority-based selection to ensure predictable behavior:
// - Priority 1: Context cancellation (shutdown)
// - Priority 2: Client lifecycle events (Register/Unregister)
// - Priority 3: Broadcast messages
//
// OBSERVABILITY: Shutdown logs include:
// - Component identification ("websocket-hub")
// - Shutdown reason (context_canceled, context_deadline)
// - Client count at shutdown time
// - Duration is not logged as the hub runs indefinitely
func (h *Hub) RunWithContext(ctx context.Context) error {
	for {
		// DETERMINISM: Priority-based selection prevents non-deterministic
		// ordering when multiple channels are ready simultaneously.

		// Priority 1: Check for shutdown (highest priority, non-blocking)
		select {
		case <-ctx.Done():
			h.logGracefulShutdown(ctx)
			return ctx.Err()
		default:
			// Context not canceled, continue
		}

		// Priority 2: Handle client lifecycle events (non-blocking check)
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logging.Info().Int("total_clients", len(h.clients)).Msg("websocket client connected")
			continue
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			logging.Info().Int("total_clients", len(h.clients)).Msg("websocket client disconnected")
			continue
		default:
			// No lifecycle events pending
		}

		// Priority 3: Handle broadcast messages or wait for any event (blocking)
		select {
		case <-ctx.Done():
			h.logGracefulShutdown(ctx)
			return ctx.Err()

		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logging.Info().Int("total_clients", len(h.clients)).Msg("websocket client connected")

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			logging.Info().Int("total_clients", len(h.clients)).Msg("websocket client disconnected")

		case message := <-h.broadcast:
			h.broadcastToClients(message)
		}
	}
}

// logGracefulShutdown logs the shutdown with structured fields for observability.
// This method:
//  1. Closes all connected clients
//  2. Logs structured shutdown information without error field
//
// The log includes:
//   - component: "websocket-hub" for filtering
//   - reason: shutdown trigger (context_canceled, context_deadline)
//   - clients_closed: number of clients that were connected
//
// Note: ctx.Err() is NOT logged as an error because context cancellation
// is expected behavior during graceful shutdown. Logging it as .Err() would
// confuse operators monitoring error logs.
func (h *Hub) logGracefulShutdown(ctx context.Context) {
	// Count clients before closing (for logging)
	clientCount := h.GetClientCount()

	// Close all clients gracefully
	h.closeAllClients()

	// Determine shutdown reason from context error
	reason := getShutdownReason(ctx)

	// Log shutdown with structured fields (no error field - this is expected behavior)
	logging.Info().
		Str("component", "websocket-hub").
		Str("reason", string(reason)).
		Int("clients_closed", clientCount).
		Msg("websocket hub stopped")
}

// getShutdownReason determines the shutdown reason from the context error.
// This provides clear observability for operators monitoring logs.
func getShutdownReason(ctx context.Context) ShutdownReason {
	switch ctx.Err() {
	case context.Canceled:
		return ShutdownReasonContextCanceled
	case context.DeadlineExceeded:
		return ShutdownReasonContextDeadline
	default:
		// Fallback for any future context error types
		return ShutdownReasonContextCanceled
	}
}

// broadcastToClients sends a message to all connected clients in a deterministic order.
// DETERMINISM: Sorts clients by their pointer address to ensure consistent iteration order.
// This prevents non-deterministic message delivery order which could cause:
// - Inconsistent client behavior in tests
// - Non-reproducible race conditions
// - Unpredictable message acknowledgment sequences
func (h *Hub) broadcastToClients(message Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// DETERMINISM: Extract client pointers and sort for consistent ordering.
	// Pointer addresses provide a stable sort key within a single process run.
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}

	// Sort by client ID for deterministic ordering
	// Using pointer address as a stable sort key
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].id < clients[j].id
	})

	// Track clients to remove (can't modify map during iteration)
	var toRemove []*Client

	for _, client := range clients {
		select {
		case client.send <- message:
			// Message sent successfully
		default:
			// Channel full or closed, mark for removal
			toRemove = append(toRemove, client)
		}
	}

	// Remove failed clients
	for _, client := range toRemove {
		close(client.send)
		delete(h.clients, client)
	}
}

// closeAllClients gracefully closes all connected WebSocket clients.
// Called during shutdown to ensure clean termination.
// DETERMINISM: Closes clients in ID order to ensure consistent shutdown behavior.
func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// DETERMINISM: Sort clients by ID for consistent close order
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].id < clients[j].id
	})

	for _, client := range clients {
		close(client.send)
		delete(h.clients, client)
	}
	logging.Info().Msg("closed all websocket clients during shutdown")
}

// BroadcastNewPlayback sends a new playback event to all connected clients
func (h *Hub) BroadcastNewPlayback(event *models.PlaybackEvent) {
	message := Message{
		Type: MessageTypePlayback,
		Data: event,
	}

	select {
	case h.broadcast <- message:
	default:
		logging.Warn().Msg("broadcast channel full, dropping playback message")
	}
}

// BroadcastJSON sends a JSON message to all connected clients
func (h *Hub) BroadcastJSON(messageType string, data interface{}) {
	message := Message{
		Type: messageType,
		Data: data,
	}

	select {
	case h.broadcast <- message:
	default:
		logging.Warn().Str("message_type", messageType).Msg("broadcast channel full, dropping JSON message")
	}
}

// SyncCompletedData represents data sent with sync_completed message
type SyncCompletedData struct {
	Timestamp      string `json:"timestamp"`
	NewPlaybacks   int    `json:"new_playbacks"`
	SyncDurationMs int64  `json:"sync_duration_ms"`
}

// BroadcastSyncCompleted notifies all clients that a sync has completed
func (h *Hub) BroadcastSyncCompleted(newPlaybacks int, durationMs int64) {
	data := SyncCompletedData{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		NewPlaybacks:   newPlaybacks,
		SyncDurationMs: durationMs,
	}

	message := Message{
		Type: MessageTypeSyncCompleted,
		Data: data,
	}

	select {
	case h.broadcast <- message:
		logging.Info().Int("clients", h.GetClientCount()).Int("new_playbacks", newPlaybacks).Msg("broadcast sync_completed")
	default:
		logging.Warn().Msg("broadcast channel full, dropping sync_completed message")
	}
}

// StatsUpdateData represents data sent with stats_update message
type StatsUpdateData struct {
	Timestamp    string `json:"timestamp"`
	TotalCount   int    `json:"total_count"`
	LastPlayback string `json:"last_playback,omitempty"`
}

// BroadcastStatsUpdate notifies all clients that stats have been updated
func (h *Hub) BroadcastStatsUpdate(totalCount int, lastPlayback string) {
	data := StatsUpdateData{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		TotalCount:   totalCount,
		LastPlayback: lastPlayback,
	}

	message := Message{
		Type: MessageTypeStatsUpdate,
		Data: data,
	}

	select {
	case h.broadcast <- message:
		logging.Info().Int("clients", h.GetClientCount()).Int("total_count", totalCount).Msg("broadcast stats_update")
	default:
		logging.Warn().Msg("broadcast channel full, dropping stats_update message")
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SyncProgressData represents data sent with sync_progress message
type SyncProgressData struct {
	Operation     string      `json:"operation"` // tautulli_import, plex_historical, server_sync
	Status        string      `json:"status"`    // running, completed, error, canceled
	ServerID      string      `json:"server_id,omitempty"`
	Progress      interface{} `json:"progress"`
	Message       string      `json:"message,omitempty"`
	Error         string      `json:"error,omitempty"`
	CorrelationID string      `json:"correlation_id"`
}

// BroadcastSyncProgress notifies all clients of sync operation progress
func (h *Hub) BroadcastSyncProgress(operation, status, correlationID string, progress interface{}) {
	data := SyncProgressData{
		Operation:     operation,
		Status:        status,
		Progress:      progress,
		CorrelationID: correlationID,
	}

	message := Message{
		Type: MessageTypeSyncProgress,
		Data: data,
	}

	select {
	case h.broadcast <- message:
		logging.Debug().
			Int("clients", h.GetClientCount()).
			Str("operation", operation).
			Str("status", status).
			Msg("broadcast sync_progress")
	default:
		logging.Warn().Msg("broadcast channel full, dropping sync_progress message")
	}
}

// BroadcastSyncProgressWithDetails notifies all clients of sync operation progress with full details
func (h *Hub) BroadcastSyncProgressWithDetails(data *SyncProgressData) {
	message := Message{
		Type: MessageTypeSyncProgress,
		Data: data,
	}

	select {
	case h.broadcast <- message:
		logging.Debug().
			Int("clients", h.GetClientCount()).
			Str("operation", data.Operation).
			Str("status", data.Status).
			Msg("broadcast sync_progress")
	default:
		logging.Warn().Msg("broadcast channel full, dropping sync_progress message")
	}
}

// BroadcastRaw parses raw JSON bytes as a playback event and broadcasts to clients.
// This method implements the eventprocessor.WebSocketBroadcaster interface.
// It expects the JSON to match the MediaEvent structure from eventprocessor.
func (h *Hub) BroadcastRaw(data []byte) {
	var rawEvent map[string]interface{}
	if err := json.Unmarshal(data, &rawEvent); err != nil {
		logging.Warn().Err(err).Msg("failed to unmarshal raw event for broadcast")
		return
	}

	// Broadcast the raw event data as a playback message
	// This preserves all fields from the original event
	message := Message{
		Type: MessageTypePlayback,
		Data: rawEvent,
	}

	select {
	case h.broadcast <- message:
	default:
		logging.Warn().Msg("broadcast channel full, dropping raw message")
	}
}

// MarshalMessage converts a message to JSON
func MarshalMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}
