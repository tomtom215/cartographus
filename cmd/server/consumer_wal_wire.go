// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal && nats

package main

import (
	"context"

	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/eventprocessor"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/wal"
)

// CreateEventStore creates an EventStore implementation for the appender.
// When consumer WAL is available, it wraps the DuckDB store with WAL protection.
// This ensures exactly-once delivery between NATS and DuckDB.
func CreateEventStore(ctx context.Context, db *database.DB, consumerWAL *wal.ConsumerWAL) (eventprocessor.EventStore, error) {
	// Create base DuckDB store
	duckdbStore, err := eventprocessor.NewDuckDBStore(db)
	if err != nil {
		return nil, err
	}

	// If consumer WAL is available, wrap with WAL protection
	if consumerWAL != nil {
		walStore, err := eventprocessor.NewWALStore(duckdbStore, consumerWAL)
		if err != nil {
			logging.Warn().Err(err).Msg("Failed to create WAL store, falling back to direct DuckDB")
			return duckdbStore, nil
		}
		logging.Info().Msg("Using WAL-protected event store for exactly-once delivery")
		return walStore, nil
	}

	logging.Info().Msg("Consumer WAL not available, using direct DuckDB store")
	return duckdbStore, nil
}

// InitAndWireConsumerWAL initializes the consumer WAL and returns the store to use.
// This is called during NATS initialization when a database is provided.
func InitAndWireConsumerWAL(ctx context.Context, db *database.DB) (*ConsumerWALComponents, eventprocessor.EventStore, error) {
	// Initialize consumer WAL
	consumerWALComponents, err := InitConsumerWAL(ctx, db)
	if err != nil {
		logging.Warn().Err(err).Msg("Consumer WAL initialization failed, proceeding without WAL protection")
		// Fall back to direct DuckDB store
		store, storeErr := eventprocessor.NewDuckDBStore(db)
		if storeErr != nil {
			return nil, nil, storeErr
		}
		return nil, store, nil
	}

	// Get the consumer WAL instance
	var consumerWAL *wal.ConsumerWAL
	if consumerWALComponents != nil {
		consumerWAL = consumerWALComponents.WAL()
	}

	// Create event store (WAL-protected if available)
	store, err := CreateEventStore(ctx, db, consumerWAL)
	if err != nil {
		if consumerWALComponents != nil {
			consumerWALComponents.Shutdown()
		}
		return nil, nil, err
	}

	return consumerWALComponents, store, nil
}
