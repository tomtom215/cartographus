// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

// This file provides a no-op stub for NATS supervisor integration.
// It is only compiled when the "nats" build tag is NOT enabled.
//
// Build without NATS support (default):
//
//	go build -o tautulli-maps ./cmd/server

package main

import (
	"github.com/tomtom215/cartographus/internal/supervisor"
)

// AddNATSToSupervisor is a no-op stub for non-NATS builds.
//
// When NATS support is not compiled in (no -tags nats), this function
// does nothing. This allows main.go to call AddNATSToSupervisor
// unconditionally without build tag conditionals.
//
// The NATSComponents parameter will be nil from the stub InitNATS
// function in nats_init_stub.go.
func AddNATSToSupervisor(_ *supervisor.SupervisorTree, _ *NATSComponents) {
	// No-op: NATS not compiled in
}
