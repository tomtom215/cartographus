// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"

	"github.com/tomtom215/cartographus/internal/models"
)

// SyncEventPublisher is a stub for non-NATS builds.
type SyncEventPublisher struct{}

// NewSyncEventPublisher returns an error in non-NATS builds.
func NewSyncEventPublisher(_ *Publisher) (*SyncEventPublisher, error) {
	return nil, ErrNATSNotEnabled
}

// PublishPlaybackEvent is a no-op stub.
func (p *SyncEventPublisher) PublishPlaybackEvent(_ context.Context, _ *models.PlaybackEvent) error {
	return ErrNATSNotEnabled
}
