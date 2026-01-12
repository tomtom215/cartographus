// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// HTTPServer interface matches *http.Server lifecycle methods.
//
// This interface allows the HTTPServerService to work with http.Server
// without direct dependency, enabling testing with mocks.
//
// Satisfied by *http.Server from net/http:
//   - ListenAndServe() error
//   - Shutdown(ctx context.Context) error
type HTTPServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// HTTPServerService wraps an HTTP server as a supervised service.
//
// This wrapper handles the translation between http.Server's blocking
// ListenAndServe pattern and suture's context-aware Serve pattern:
//
//  1. Starts ListenAndServe in a goroutine
//  2. Waits for either context cancellation or server error
//  3. On shutdown, calls Shutdown with the provided timeout
//
// Example usage:
//
//	server := &http.Server{Addr: ":3857", Handler: router}
//	svc := services.NewHTTPServerService(server, 10*time.Second)
//	tree.AddAPIService(svc)
type HTTPServerService struct {
	server          HTTPServer
	shutdownTimeout time.Duration
	name            string
}

// NewHTTPServerService creates a new HTTP server service wrapper.
//
// The shutdownTimeout determines how long to wait for active connections
// to close during graceful shutdown. A typical value is 10-30 seconds.
func NewHTTPServerService(server HTTPServer, shutdownTimeout time.Duration) *HTTPServerService {
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}
	return &HTTPServerService{
		server:          server,
		shutdownTimeout: shutdownTimeout,
		name:            "http-server",
	}
}

// Serve implements suture.Service.
//
// This method:
//  1. Starts the HTTP server in a goroutine (blocks on ListenAndServe)
//  2. Waits for context cancellation or server error
//  3. On shutdown, calls server.Shutdown for graceful termination
//
// Returns nil on graceful shutdown, or an error if the server fails.
// http.ErrServerClosed is converted to nil since it's expected on shutdown.
func (h *HTTPServerService) Serve(ctx context.Context) error {
	// Start server in goroutine since ListenAndServe blocks
	errCh := make(chan error, 1)
	go func() {
		if err := h.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-errCh:
		// Server failed to start or crashed
		if err != nil {
			return fmt.Errorf("http server failed: %w", err)
		}
		// Server closed normally (shouldn't happen unless externally triggered)
		return nil

	case <-ctx.Done():
		// Graceful shutdown requested
		// Use a new context for shutdown since the original is canceled
		shutdownCtx, cancel := context.WithTimeout(context.Background(), h.shutdownTimeout)
		defer cancel()

		// Attempt graceful shutdown
		if err := h.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http server shutdown failed: %w", err)
		}

		// Wait for the server goroutine to finish
		<-errCh
		return ctx.Err()
	}
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (h *HTTPServerService) String() string {
	return h.name
}
