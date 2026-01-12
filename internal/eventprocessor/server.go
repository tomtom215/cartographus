// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

// EmbeddedServer wraps the NATS server with lifecycle management.
// The embedded server provides a self-contained NATS JetStream instance
// for single-instance deployments without external dependencies.
type EmbeddedServer struct {
	server    *server.Server
	config    ServerConfig
	clientURL string
}

// NewEmbeddedServer creates and starts an embedded NATS server.
// The server is configured for JetStream with the specified limits.
// Returns an error if the server fails to start within 30 seconds.
func NewEmbeddedServer(cfg *ServerConfig) (*EmbeddedServer, error) {
	opts := &server.Options{
		ServerName:         "media-events",
		Host:               cfg.Host,
		Port:               cfg.Port,
		JetStream:          true,
		StoreDir:           cfg.StoreDir,
		JetStreamMaxMemory: cfg.JetStreamMaxMem,
		JetStreamMaxStore:  cfg.JetStreamMaxStore,
		// CRITICAL: Set to false for hybrid deployments
		// nats_js extension requires TCP access
		DontListen: false,
		// Logging
		Debug:      false,
		Trace:      false,
		NoLog:      false,
		MaxPayload: 8 * 1024 * 1024, // 8MB max message size
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("create NATS server: %w", err)
	}

	// Configure structured logging
	ns.ConfigureLogger()

	// Start in background
	go ns.Start()

	// Wait for ready with timeout
	if !ns.ReadyForConnections(30 * time.Second) {
		ns.Shutdown()
		return nil, fmt.Errorf("NATS server not ready within timeout")
	}

	return &EmbeddedServer{
		server:    ns,
		config:    *cfg,
		clientURL: ns.ClientURL(),
	}, nil
}

// ClientURL returns the connection URL for clients.
func (s *EmbeddedServer) ClientURL() string {
	return s.clientURL
}

// Shutdown gracefully stops the server.
// Waits for in-flight messages to complete or context cancellation.
func (s *EmbeddedServer) Shutdown(ctx context.Context) error {
	// Allow in-flight messages to complete
	s.server.Shutdown()

	// Wait for shutdown or context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.server.WaitForShutdown()
		return nil
	}
}

// IsRunning returns server health status.
func (s *EmbeddedServer) IsRunning() bool {
	return s.server.Running()
}

// JetStreamEnabled returns whether JetStream is enabled.
func (s *EmbeddedServer) JetStreamEnabled() bool {
	return s.server.JetStreamEnabled()
}
