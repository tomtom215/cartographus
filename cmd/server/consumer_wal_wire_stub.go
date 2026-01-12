// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !wal || !nats

package main

import (
	"context"

	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/eventprocessor"
	"github.com/tomtom215/cartographus/internal/logging"
)

// CreateEventStore creates an EventStore implementation for the appender.
// Without the wal build tag, this returns the DuckDB store directly.
// Note: This configuration provides at-least-once delivery, not exactly-once.
func CreateEventStore(ctx context.Context, db *database.DB, _ interface{}) (eventprocessor.EventStore, error) {
	logging.Info().Msg("Consumer WAL not available (build without -tags wal), using direct DuckDB store")
	return eventprocessor.NewDuckDBStore(db)
}

// InitAndWireConsumerWAL is a stub when WAL is not enabled.
// It returns nil for consumer WAL components and creates a direct DuckDB store.
func InitAndWireConsumerWAL(ctx context.Context, db *database.DB) (*ConsumerWALComponents, eventprocessor.EventStore, error) {
	store, err := eventprocessor.NewDuckDBStore(db)
	if err != nil {
		return nil, nil, err
	}
	logging.Info().Msg("Using direct DuckDB store (consumer WAL not available)")
	return nil, store, nil
}
