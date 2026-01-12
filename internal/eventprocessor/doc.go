// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package eventprocessor provides an event-sourced architecture using Watermill,
// NATS JetStream, and DuckDB for multi-source media server analytics.
//
// This package enables multiple deployment scenarios:
//   - Plex-only: Direct Plex WebSocket + API integration without Tautulli
//   - Tautulli: Enhanced analytics on top of existing Tautulli data
//   - Hybrid: Both Plex and Tautulli for comprehensive coverage
//   - Migration: Gradual migration from Tautulli to standalone Plex support
//   - Jellyfin: Future support for Jellyfin with the same event-driven pattern
//   - Multi-Server: Support for users with synced Plex + Jellyfin instances
//
// # Architecture Decision: NATS-First (Event Sourcing)
//
// This package implements a unified event-sourced architecture where ALL playback
// events flow through NATS JetStream before reaching DuckDB:
//
//	┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
//	│  Tautulli   │   │ Plex Webhook│   │Plex WebSocket│  │  Jellyfin   │
//	│    Sync     │   │   Handler   │   │   Handler   │   │   (Future)  │
//	└──────┬──────┘   └──────┬──────┘   └──────┬──────┘   └──────┬──────┘
//	       │                 │                 │                 │
//	       └────────────────┬┴─────────────────┴─────────────────┘
//	                        │
//	                        ▼
//	              ┌─────────────────────┐
//	              │   NATS JetStream    │  ← Single Source of Truth
//	              │   (Event Store)     │
//	              └─────────┬───────────┘
//	                        │
//	          ┌─────────────┼─────────────┐
//	          ▼             ▼             ▼
//	   ┌────────────┐ ┌───────────┐ ┌────────────┐
//	   │DuckDBConsumer│ │WebSocket │ │  Future    │
//	   │(Materialized)│ │  Bridge  │ │ Consumers  │
//	   └──────┬───────┘ └───────────┘ └────────────┘
//	          │
//	          ▼
//	   ┌────────────┐
//	   │   DuckDB   │  ← Materialized View (derived state)
//	   │ (Analytics)│
//	   └────────────┘
//
// # Why Event Sourcing?
//
//   - Multi-Source Deduplication: Users with synced Plex/Jellyfin watch history
//     need cross-source deduplication at the event level
//   - Single Source of Truth: NATS JetStream holds the authoritative event log
//   - Replay & Audit: Full event history enables debugging and replay
//   - Scalability: Adding Jellyfin becomes "just another event publisher"
//   - Real-Time: WebSocket consumers get events immediately
//   - Testability: Centralized event tests work for all sources
//
// # Cross-Source Deduplication Strategy
//
// Multi-layer deduplication handles the complexity of users with multiple servers:
//
//  1. Event Correlation Key: (source, user_id, rating_key, started_at_bucket)
//     - Groups events representing the same playback across sources
//     - 5-minute time bucket handles clock skew between servers
//
//  2. In-Memory Cache: Recent EventIDs and CorrelationKeys (5-minute window)
//     - Fast path for duplicate webhook deliveries
//     - Handles rapid-fire events from same source
//
//  3. Database Check: Query for existing correlated events before INSERT
//     - Cross-source dedup: Tautulli event vs Plex webhook for same playback
//     - Handles Plex/Jellyfin sync tools that duplicate watch history
//
//  4. Database Constraint: UNIQUE INDEX as final safety net
//     - Prevents duplicates that slip through other layers
//
// # Data Flow
//
// All sources publish to NATS, DuckDBConsumer is the ONLY writer to DuckDB:
//
//	Sync Manager:    Tautulli API → MediaEvent → NATS publish
//	Webhook Handler: Plex Webhook → MediaEvent → NATS publish
//	WebSocket:       Plex WS      → MediaEvent → NATS publish
//	Future:          Jellyfin API → MediaEvent → NATS publish
//
//	DuckDBConsumer:  NATS subscribe → Deduplicate → Batch → DuckDB INSERT
//
// # Key Components
//
//   - EmbeddedServer: Optional embedded NATS JetStream server for single-instance deployments
//   - Publisher: Watermill publisher with circuit breaker and reconnection handling
//   - Subscriber: Durable JetStream consumer with exactly-once delivery
//   - DuckDBConsumer: Event consumer with cross-source deduplication
//   - EventAppender: Batch appender for high-throughput DuckDB writes
//   - StreamReader: Unified interface for reading from streams
//
// # Usage Example
//
//	// Create embedded NATS server (optional)
//	server, err := eventprocessor.NewEmbeddedServer(eventprocessor.DefaultServerConfig())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer server.Shutdown(ctx)
//
//	// Create publisher
//	pub, err := eventprocessor.NewPublisher(
//	    eventprocessor.DefaultPublisherConfig(server.ClientURL()),
//	    nil, // logger
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer pub.Close()
//
//	// Publish event
//	event := eventprocessor.NewMediaEvent("plex")
//	event.UserID = 1
//	event.Username = "user"
//	event.MediaType = "movie"
//	event.Title = "Movie Title"
//	event.StartedAt = time.Now()
//
//	msg, _ := eventprocessor.SerializeEvent(event)
//	pub.Publish(ctx, event.Topic(), msg)
//
// # Configuration
//
// The package uses configuration structs with sensible defaults:
//
//	cfg := eventprocessor.DefaultNATSConfig()
//	cfg.StoreDir = "/data/nats/jetstream"
//	cfg.MaxMemory = 1 << 30 // 1GB
//
// # Fallback Pattern
//
// The package implements a resilient reader pattern that automatically falls back
// to the Go NATS client when the DuckDB nats_js extension is unavailable:
//
//	reader, err := eventprocessor.NewResilientReader(cfg)
//	// Uses nats_js extension if available, otherwise Go NATS client
//	messages, err := reader.Query(ctx, "MEDIA_EVENTS", opts)
//
// # Integration
//
// The event processor integrates with the existing Cartographus architecture:
//
//   - Sync Manager publishes events to NATS JetStream
//   - Subscriber processes events and writes to DuckDB
//   - WebSocket Hub subscribes to NATS for real-time frontend updates
//
// See docs/WATERMILL_NATS_ARCHITECTURE.md for detailed architecture documentation.
package eventprocessor
