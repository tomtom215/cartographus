// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package supervisor

import (
	"context"
	"log/slog"
	"time"

	"github.com/thejerf/suture/v4"
	"github.com/thejerf/sutureslog"
)

// TreeConfig holds supervisor tree configuration.
type TreeConfig struct {
	// FailureThreshold is the number of failures before entering backoff.
	// Default: 5
	FailureThreshold float64

	// FailureDecay is the rate at which failures decay in seconds.
	// Default: 30
	FailureDecay float64

	// FailureBackoff is the duration to wait when threshold is exceeded.
	// Default: 15s
	FailureBackoff time.Duration

	// ShutdownTimeout is the maximum time to wait for graceful shutdown.
	// Default: 10s
	ShutdownTimeout time.Duration
}

// DefaultTreeConfig returns production-ready defaults.
// These values match suture's built-in defaults per pkg.go.dev documentation.
func DefaultTreeConfig() TreeConfig {
	return TreeConfig{
		FailureThreshold: 5.0,
		FailureDecay:     30.0,
		FailureBackoff:   15 * time.Second,
		ShutdownTimeout:  10 * time.Second,
	}
}

// SupervisorTree manages the hierarchical supervisor structure for Cartographus.
//
// The tree is organized into three layers:
//   - data: WAL services (if enabled)
//   - messaging: WebSocket hub, sync manager, NATS (if enabled)
//   - api: HTTP server
//
// This structure provides failure isolation - a crash in the messaging layer
// won't affect the API layer's ability to serve cached responses.
type SupervisorTree struct {
	root      *suture.Supervisor
	data      *suture.Supervisor
	messaging *suture.Supervisor
	api       *suture.Supervisor
	logger    *slog.Logger
	config    TreeConfig
}

// NewSupervisorTree creates a new supervisor tree with the given configuration.
func NewSupervisorTree(logger *slog.Logger, config TreeConfig) (*SupervisorTree, error) {
	// Apply defaults for zero values
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5.0
	}
	if config.FailureDecay == 0 {
		config.FailureDecay = 30.0
	}
	if config.FailureBackoff == 0 {
		config.FailureBackoff = 15 * time.Second
	}
	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = 10 * time.Second
	}

	// Create event hook using sutureslog.
	// IMPORTANT: The correct API is (&Handler{Logger: logger}).MustHook()
	// NOT sutureslog.EventHook(logger) which does not exist.
	// MustHook has a pointer receiver, so we need to take the address.
	handler := &sutureslog.Handler{Logger: logger}
	eventHook := handler.MustHook()

	rootSpec := suture.Spec{
		EventHook:        eventHook,
		FailureThreshold: config.FailureThreshold,
		FailureDecay:     config.FailureDecay,
		FailureBackoff:   config.FailureBackoff,
		Timeout:          config.ShutdownTimeout,
	}

	// Child supervisors use the same failure parameters.
	// They will inherit the EventHook when added to the root.
	childSpec := suture.Spec{
		FailureThreshold: config.FailureThreshold,
		FailureDecay:     config.FailureDecay,
		FailureBackoff:   config.FailureBackoff,
		Timeout:          config.ShutdownTimeout,
	}

	root := suture.New("cartographus", rootSpec)
	data := suture.New("data-layer", childSpec)
	messaging := suture.New("messaging-layer", childSpec)
	api := suture.New("api-layer", childSpec)

	// Build tree hierarchy
	root.Add(data)
	root.Add(messaging)
	root.Add(api)

	return &SupervisorTree{
		root:      root,
		data:      data,
		messaging: messaging,
		api:       api,
		logger:    logger,
		config:    config,
	}, nil
}

// Root returns the root supervisor for direct access if needed.
func (t *SupervisorTree) Root() *suture.Supervisor {
	return t.root
}

// AddDataService adds a service to the data layer supervisor.
// Use this for WAL-related services (RetryLoop, Compactor).
func (t *SupervisorTree) AddDataService(svc suture.Service) suture.ServiceToken {
	return t.data.Add(svc)
}

// AddMessagingService adds a service to the messaging layer supervisor.
// Use this for WebSocket hub, sync manager, and NATS components.
func (t *SupervisorTree) AddMessagingService(svc suture.Service) suture.ServiceToken {
	return t.messaging.Add(svc)
}

// AddAPIService adds a service to the API layer supervisor.
// Use this for the HTTP server.
func (t *SupervisorTree) AddAPIService(svc suture.Service) suture.ServiceToken {
	return t.api.Add(svc)
}

// RemoveMessagingService removes a service from the messaging layer supervisor.
// Use this to remove services that were added with AddMessagingService.
func (t *SupervisorTree) RemoveMessagingService(token suture.ServiceToken) error {
	return t.messaging.Remove(token)
}

// Serve starts the supervisor tree and blocks until the context is canceled.
// This is the main entry point for running the supervised application.
func (t *SupervisorTree) Serve(ctx context.Context) error {
	return t.root.Serve(ctx)
}

// ServeBackground starts the supervisor tree in a background goroutine.
// Returns a channel that receives the error (or nil) when the supervisor stops.
func (t *SupervisorTree) ServeBackground(ctx context.Context) <-chan error {
	return t.root.ServeBackground(ctx)
}

// UnstoppedServiceReport returns information about services that failed to stop
// within the configured shutdown timeout. Useful for debugging shutdown issues.
func (t *SupervisorTree) UnstoppedServiceReport() ([]suture.UnstoppedService, error) {
	return t.root.UnstoppedServiceReport()
}

// Remove removes a service from the tree by its token.
// The service will be stopped and removed.
func (t *SupervisorTree) Remove(token suture.ServiceToken) error {
	return t.root.Remove(token)
}

// RemoveAndWait removes a service and waits for it to fully stop.
// Use this when you need to ensure a service has completely terminated
// before proceeding (e.g., during configuration reload).
func (t *SupervisorTree) RemoveAndWait(token suture.ServiceToken, timeout time.Duration) error {
	return t.root.RemoveAndWait(token, timeout)
}
