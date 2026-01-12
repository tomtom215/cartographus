// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package websocket provides real-time bidirectional communication for live updates.

This package implements WebSocket support for broadcasting sync completion events,
statistics updates, and live playback notifications to connected frontend clients.
It uses the gorilla/websocket library with a hub-client architecture for efficient
message broadcasting.

Key Components:

  - Hub: Central message broker that manages client connections and broadcasts
  - Client: Represents a single WebSocket connection with read/write goroutines
  - Message: Typed message structure for different event types

Architecture:

The package implements a hub-and-spoke pattern:

	┌──────────┐
	│   Hub    │ ← Broadcasts to all clients
	└────┬─────┘
	     │
	┌────┴─────┬─────────┬─────────┐
	│          │         │         │
	│ Client1  │ Client2 │ Client3 │ Client4
	│          │         │         │
	└──────────┴─────────┴─────────┘

Each client has two goroutines:
  - readPump: Reads from WebSocket, handles pings
  - writePump: Writes to WebSocket, sends pongs

Message Types:

The following message types are supported:

  - sync_completed: Sync operation finished (newRecords, durationMs)
  - stats_update: Global statistics changed (totalPlaybacks, uniqueUsers, etc.)
  - live_activity: Real-time playback notifications
  - error: Error messages for debugging

Usage Example - Server:

	import (
	    "github.com/tomtom215/cartographus/internal/websocket"
	    "net/http"
	)

	// Create hub
	hub := websocket.NewHub()
	go hub.Run()

	// WebSocket upgrade endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
	    websocket.ServeWS(hub, w, r)
	})

	// Broadcast sync completion
	hub.BroadcastSyncCompleted(150, 2340) // 150 new records, 2340ms

	// Broadcast stats update
	hub.BroadcastStatsUpdate(map[string]interface{}{
	    "total_playbacks": 10000,
	    "unique_users": 45,
	})

Usage Example - Client (JavaScript):

	// Connect to WebSocket
	const ws = new WebSocket('ws://localhost:3857/ws');

	ws.onmessage = (event) => {
	    const msg = JSON.parse(event.data);

	    if (msg.type === 'sync_completed') {
	        console.log(`Sync: ${msg.data.new_records} new records`);
	        refreshData(); // Reload charts and map
	    }

	    if (msg.type === 'stats_update') {
	        updateStatsDisplay(msg.data);
	    }
	};

Performance Characteristics:

  - Broadcast latency: <10ms for typical payloads
  - Max clients: 1000+ concurrent connections tested
  - Ping interval: 30 seconds (keeps connection alive)
  - Write deadline: 10 seconds per message
  - Message size limit: 512KB (configurable)

Connection Lifecycle:

1. Client connects via HTTP upgrade
2. Hub registers client
3. Client starts read/write goroutines
4. Hub broadcasts messages to all clients
5. Client disconnects (network error or explicit close)
6. Hub unregisters client and cleans up

Thread Safety:

The package is fully thread-safe:
  - Hub uses mutex for client map access
  - Channels coordinate goroutine communication
  - Each client has separate read/write goroutines
  - No shared mutable state between clients

Error Handling:

The package handles:
  - Connection upgrades failures: Returns HTTP 400
  - Read errors: Closes connection gracefully
  - Write errors: Removes client from hub
  - Ping/pong timeout: Detects dead connections (60s timeout)

Configuration:

WebSocket settings:
  - writeWait: 10 seconds (time allowed to write message)
  - pongWait: 60 seconds (time allowed to read pong)
  - pingPeriod: 30 seconds (ping interval, must be < pongWait)
  - maxMessageSize: 512 KB (max message size)

See Also:

  - github.com/gorilla/websocket: Underlying WebSocket library
  - internal/api: WebSocket endpoint handler
  - internal/sync: Sync completion broadcasts
*/
package websocket
