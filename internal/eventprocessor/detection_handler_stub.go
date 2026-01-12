// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"time"

	"github.com/tomtom215/cartographus/internal/detection"
)

// DetectionProcessor is the interface that the detection engine implements.
// Stub for non-NATS builds.
type DetectionProcessor interface {
	Process(ctx context.Context, event *detection.DetectionEvent) ([]*detection.Alert, error)
	Enabled() bool
}

// DetectionHandler is a stub for non-NATS builds.
type DetectionHandler struct{}

// DetectionHandlerConfig is a stub for non-NATS builds.
type DetectionHandlerConfig struct {
	ContinueOnError bool
}

// DefaultDetectionHandlerConfig returns production defaults (stub).
func DefaultDetectionHandlerConfig() DetectionHandlerConfig {
	return DetectionHandlerConfig{ContinueOnError: true}
}

// NewDetectionHandler returns nil for non-NATS builds.
func NewDetectionHandler(_ DetectionProcessor, _ interface{}) (*DetectionHandler, error) {
	return nil, nil
}

// Stats returns empty statistics (stub).
func (h *DetectionHandler) Stats() DetectionHandlerStats {
	return DetectionHandlerStats{}
}

// DetectionHandlerStats holds runtime statistics (stub).
type DetectionHandlerStats struct {
	MessagesReceived  int64
	MessagesProcessed int64
	DetectionErrors   int64
	AlertsGenerated   int64
	SkippedDisabled   int64
	ParseErrors       int64
	LastMessageTime   time.Time
}
