// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"fmt"
)

// EmbeddedServer is a stub when NATS dependencies are not available.
// Build with -tags=nats to enable full NATS server support.
type EmbeddedServer struct {
	clientURL string
}

// NewEmbeddedServer returns an error when NATS dependencies are not available.
// Build with -tags=nats to enable full NATS server support.
func NewEmbeddedServer(cfg *ServerConfig) (*EmbeddedServer, error) {
	return nil, fmt.Errorf("NATS server not available: build with -tags=nats")
}

// ClientURL returns the connection URL for clients.
func (s *EmbeddedServer) ClientURL() string {
	return s.clientURL
}

// Shutdown is a no-op stub.
func (s *EmbeddedServer) Shutdown(ctx context.Context) error {
	return nil
}

// IsRunning always returns false for the stub.
func (s *EmbeddedServer) IsRunning() bool {
	return false
}

// JetStreamEnabled always returns false for the stub.
func (s *EmbeddedServer) JetStreamEnabled() bool {
	return false
}
