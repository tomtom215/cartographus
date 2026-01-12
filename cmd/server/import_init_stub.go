// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package main

import (
	"github.com/tomtom215/cartographus/internal/api"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/supervisor"
)

// ImportComponents is a stub for non-NATS builds.
type ImportComponents struct{}

// InitImport is a no-op when NATS is not enabled.
// Import functionality requires NATS for event publishing.
func InitImport(_ *config.Config, _ *NATSComponents, _ *supervisor.SupervisorTree, _ *api.Router) (*ImportComponents, error) {
	// Import requires NATS - no-op when NATS is not compiled in
	return nil, nil
}
