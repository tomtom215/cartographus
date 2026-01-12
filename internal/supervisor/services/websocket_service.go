// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package services

import (
	"context"
)

// ContextHub interface matches *websocket.Hub's RunWithContext method.
//
// This interface allows the WebSocketHubService to work with the Hub
// without importing the websocket package, avoiding circular dependencies.
//
// Satisfied by *websocket.Hub from internal/websocket/hub.go:
//   - RunWithContext(ctx context.Context) error - lines 91-127
type ContextHub interface {
	RunWithContext(ctx context.Context) error
}

// WebSocketHubService wraps a WebSocket hub as a supervised service.
//
// The hub's RunWithContext method already implements the suture.Service
// pattern, so this wrapper simply delegates to it and provides a name
// for logging.
//
// Example usage:
//
//	hub := websocket.NewHub()
//	svc := services.NewWebSocketHubService(hub)
//	tree.AddMessagingService(svc)
type WebSocketHubService struct {
	hub  ContextHub
	name string
}

// NewWebSocketHubService creates a new WebSocket hub service wrapper.
func NewWebSocketHubService(hub ContextHub) *WebSocketHubService {
	return &WebSocketHubService{
		hub:  hub,
		name: "websocket-hub",
	}
}

// Serve implements suture.Service.
//
// This method delegates to hub.RunWithContext which:
//  1. Processes client registration/unregistration and broadcasts
//  2. Returns when the context is canceled
//  3. Gracefully closes all clients on shutdown
//
// The method returns ctx.Err() on normal shutdown.
func (w *WebSocketHubService) Serve(ctx context.Context) error {
	return w.hub.RunWithContext(ctx)
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (w *WebSocketHubService) String() string {
	return w.name
}
