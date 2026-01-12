// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

// This file provides NATS integration with the supervisor tree.
// It is only compiled when the "nats" build tag is enabled.
//
// Build with NATS support:
//
//	go build -tags nats -o tautulli-maps ./cmd/server

package main

import (
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/supervisor"
	"github.com/tomtom215/cartographus/internal/supervisor/services"
)

// AddNATSToSupervisor adds the NATS components service to the supervisor tree's
// messaging layer for automatic lifecycle management.
//
// The NATS components include:
//   - Embedded NATS server (if configured)
//   - JetStream publisher for event distribution
//   - WebSocket subscriber for real-time client notifications
//   - DuckDB consumer for event persistence
//   - WAL components for event durability (if WAL build tag enabled)
//
// When added to the supervisor tree:
//   - Start() is called when the supervisor starts
//   - Shutdown() is called when the supervisor stops
//   - The service is automatically restarted on failure
//
// This function is a no-op if natsComponents is nil (NATS disabled via config).
//
// Example usage in main.go:
//
//	natsComponents, _ := InitNATS(cfg, syncManager, wsHub, handler, db)
//	AddNATSToSupervisor(tree, natsComponents)
func AddNATSToSupervisor(tree *supervisor.SupervisorTree, natsComponents *NATSComponents) {
	if natsComponents == nil {
		return
	}
	tree.AddMessagingService(services.NewNATSComponentsService(natsComponents))
	logging.Info().Msg("NATS components added to supervisor tree (messaging layer)")
}
