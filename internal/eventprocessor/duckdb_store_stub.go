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

// PlaybackEventInserter defines the interface for inserting playback events.
// This is a stub for non-NATS builds.
type PlaybackEventInserter interface {
	InsertPlaybackEvent(event *models.PlaybackEvent) error
}

// DuckDBStore is a stub for non-NATS builds.
type DuckDBStore struct{}

// NewDuckDBStore returns an error in non-NATS builds.
func NewDuckDBStore(_ PlaybackEventInserter) (*DuckDBStore, error) {
	return nil, ErrNATSNotEnabled
}

// InsertMediaEvents is a no-op stub.
func (s *DuckDBStore) InsertMediaEvents(_ context.Context, _ []*MediaEvent) error {
	return ErrNATSNotEnabled
}
