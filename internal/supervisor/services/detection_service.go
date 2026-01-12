// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package services

import (
	"context"
)

// DetectionEngine interface matches the detection.Engine's RunWithContext method.
//
// This interface allows the DetectionService to work with the detection engine
// without importing the detection package, avoiding circular dependencies.
//
// Satisfied by *detection.Engine from internal/detection/engine.go.
type DetectionEngine interface {
	// RunWithContext starts the detection engine's background processing.
	// It returns when the context is canceled.
	RunWithContext(ctx context.Context) error
}

// DetectionService wraps a detection engine as a supervised service.
//
// The engine's background processing monitors for events and generates alerts.
// The supervisor will restart the service if it crashes.
//
// Example usage:
//
//	engine := detection.NewEngine(store, eventHistory, wsHub)
//	svc := services.NewDetectionService(engine)
//	tree.AddMessagingService(svc)
type DetectionService struct {
	engine DetectionEngine
	name   string
}

// NewDetectionService creates a new detection engine service wrapper.
func NewDetectionService(engine DetectionEngine) *DetectionService {
	return &DetectionService{
		engine: engine,
		name:   "detection-engine",
	}
}

// Serve implements suture.Service.
//
// This method delegates to engine.RunWithContext which:
//  1. Processes incoming detection events
//  2. Runs detection rules against events
//  3. Generates and broadcasts alerts
//  4. Returns when the context is canceled
//
// The method returns ctx.Err() on normal shutdown.
func (d *DetectionService) Serve(ctx context.Context) error {
	return d.engine.RunWithContext(ctx)
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (d *DetectionService) String() string {
	return d.name
}
