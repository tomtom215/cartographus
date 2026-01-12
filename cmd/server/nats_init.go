// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/tomtom215/cartographus/internal/api"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/detection"
	"github.com/tomtom215/cartographus/internal/eventprocessor"
	"github.com/tomtom215/cartographus/internal/logging"
	intsync "github.com/tomtom215/cartographus/internal/sync"
	ws "github.com/tomtom215/cartographus/internal/websocket"
)

// NATSComponents holds all NATS-related components for lifecycle management.
type NATSComponents struct {
	server            *eventprocessor.EmbeddedServer
	natsConn          *natsgo.Conn
	streamInitializer *eventprocessor.StreamInitializer
	publisher         *eventprocessor.Publisher

	// Router-based message processing (replaces manual consume loops)
	router           *eventprocessor.Router
	duckdbHandler    *eventprocessor.DuckDBHandler
	wsHandler        *eventprocessor.WebSocketHandler
	detectionHandler *eventprocessor.DetectionHandler

	// Subscribers for Router handlers
	duckdbSubscriber    *eventprocessor.Subscriber
	wsSubscriber        *eventprocessor.Subscriber
	detectionSubscriber *eventprocessor.Subscriber

	// DuckDB persistence
	duckdbAppender *eventprocessor.Appender
	duckdbStore    eventprocessor.EventStore // EventStore interface (may be DuckDBStore or WALStore)

	// WAL for event durability (optional, requires -tags wal,nats)
	walComponents *WALComponents

	// Consumer-side WAL for exactly-once delivery (optional, requires -tags wal,nats)
	consumerWALComponents *ConsumerWALComponents

	// Event publisher for Jellyfin/Emby managers
	eventPublisher intsync.EventPublisher

	// Health checking
	healthChecker *eventprocessor.HealthChecker

	shutdownComplete chan struct{}
	mu               sync.Mutex
	running          bool
}

// InitNATS initializes all NATS components when NATS_ENABLED=true.
// It returns a cleanup function that should be called during shutdown.
//
// Parameters:
//   - cfg: Application configuration with NATS settings
//   - syncManager: Sync manager for Tautulli event publishing
//   - wsHub: WebSocket hub for real-time broadcasts
//   - handler: API handler for Plex webhook event publishing (optional, can be nil)
//   - db: Database for DuckDB consumer path (optional, can be nil to disable DuckDB path)
//   - detectionEngine: Detection engine for anomaly detection (optional, can be nil)
//
//nolint:gocyclo // Complex initialization with multiple components is inherently multi-step
func InitNATS(cfg *config.Config, syncManager *intsync.Manager, wsHub *ws.Hub, handler *api.Handler, db *database.DB, detectionEngine *detection.Engine) (*NATSComponents, error) {
	if !cfg.NATS.Enabled {
		logging.Info().Msg("NATS event processing disabled (NATS_ENABLED=false)")
		return nil, nil
	}

	logging.Info().Msg("Initializing NATS event processing...")

	components := &NATSComponents{
		shutdownComplete: make(chan struct{}),
	}

	var natsURL string

	// Step 1: Initialize embedded NATS server if enabled
	if cfg.NATS.EmbeddedServer {
		serverCfg := eventprocessor.ServerConfig{
			Host:              "127.0.0.1",
			Port:              4222,
			StoreDir:          cfg.NATS.StoreDir,
			JetStreamMaxMem:   cfg.NATS.MaxMemory,
			JetStreamMaxStore: cfg.NATS.MaxStore,
		}

		server, err := eventprocessor.NewEmbeddedServer(&serverCfg)
		if err != nil {
			return nil, err
		}
		components.server = server
		natsURL = server.ClientURL()
		logging.Info().Str("url", natsURL).Msg("Embedded NATS server started")
	} else {
		natsURL = cfg.NATS.URL
		logging.Info().Str("url", natsURL).Msg("Using external NATS server")
	}

	// Step 2: Connect to NATS and initialize stream
	nc, err := natsgo.Connect(natsURL,
		natsgo.RetryOnFailedConnect(true),
		natsgo.MaxReconnects(-1),
		natsgo.ReconnectWait(2*time.Second),
	)
	if err != nil {
		components.Shutdown(context.Background())
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}
	components.natsConn = nc
	logging.Info().Msg("NATS connection established")

	// Step 3: Initialize JetStream and ensure stream exists
	js, err := jetstream.New(nc)
	if err != nil {
		components.Shutdown(context.Background())
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	streamCfg := eventprocessor.DefaultStreamConfig()
	streamCfg.MaxAge = time.Duration(cfg.NATS.StreamRetentionDays) * 24 * time.Hour

	streamInitializer, err := eventprocessor.NewStreamInitializer(js, &streamCfg)
	if err != nil {
		components.Shutdown(context.Background())
		return nil, fmt.Errorf("create stream initializer: %w", err)
	}
	components.streamInitializer = streamInitializer

	ctx := context.Background()
	stream, err := streamInitializer.EnsureStream(ctx)
	if err != nil {
		components.Shutdown(context.Background())
		return nil, fmt.Errorf("ensure stream exists: %w", err)
	}
	streamInfo := stream.CachedInfo()
	logging.Info().
		Str("name", streamInfo.Config.Name).
		Strs("subjects", streamInfo.Config.Subjects).
		Dur("max_age", streamInfo.Config.MaxAge).
		Msg("JetStream stream ready")

	// Step 4: Create Publisher
	publisherCfg := eventprocessor.DefaultPublisherConfig(natsURL)
	publisher, err := eventprocessor.NewPublisher(publisherCfg, nil)
	if err != nil {
		components.Shutdown(context.Background())
		return nil, err
	}
	components.publisher = publisher
	logging.Info().Msg("NATS publisher created")

	// Step 5: Create SyncEventPublisher
	syncPublisher, err := eventprocessor.NewSyncEventPublisher(publisher)
	if err != nil {
		components.Shutdown(context.Background())
		return nil, err
	}
	logging.Info().Msg("NATS sync event publisher created")

	// Step 5b: Initialize WAL for event durability (if enabled)
	// WAL wraps the sync publisher to ensure no event loss on NATS failures
	walComponents, err := InitWAL(ctx, syncPublisher)
	if err != nil {
		components.Shutdown(context.Background())
		return nil, fmt.Errorf("initialize WAL: %w", err)
	}
	components.walComponents = walComponents

	// Determine which event publisher to use
	// If WAL is enabled, use WAL-enabled publisher for durability
	// Otherwise, use direct sync publisher
	var eventPublisher intsync.EventPublisher = syncPublisher
	if walComponents != nil {
		if walPub := walComponents.EventPublisher(); walPub != nil {
			eventPublisher = walPub
			logging.Info().Msg("Using WAL-enabled event publisher for durability")
		}
	}

	// Wire event publisher to sync manager and handler
	syncManager.SetEventPublisher(eventPublisher)
	logging.Info().Msg("Event publisher wired to sync manager")

	// Wire handler with same publisher for Plex webhooks
	if handler != nil {
		handler.SetEventPublisher(eventPublisher)
		logging.Info().Msg("Event publisher wired to API handler for webhooks")
	}

	// Store event publisher for external access (Jellyfin/Emby managers)
	components.eventPublisher = eventPublisher

	// Step 6: Create Router with middleware from config
	routerCfg := eventprocessor.RouterConfig{
		RetryMaxRetries:      cfg.NATS.RouterRetryCount,
		RetryInitialInterval: cfg.NATS.RouterRetryInitialInterval,
		RetryMaxInterval:     cfg.NATS.RouterRetryInitialInterval * 10, // 10x initial
		ThrottlePerSecond:    int64(cfg.NATS.RouterThrottlePerSecond),
		DeduplicationEnabled: cfg.NATS.RouterDeduplicationEnabled,
		DeduplicationTTL:     cfg.NATS.RouterDeduplicationTTL,
		CloseTimeout:         cfg.NATS.RouterCloseTimeout,
	}
	if cfg.NATS.RouterPoisonQueueEnabled {
		routerCfg.PoisonQueueTopic = cfg.NATS.RouterPoisonQueueTopic
	}

	// Create poison queue publisher if enabled
	// Use the underlying Watermill publisher for poison queue middleware
	var poisonPub message.Publisher
	if cfg.NATS.RouterPoisonQueueEnabled && publisher != nil {
		poisonPub = publisher.WatermillPublisher()
	}

	router, err := eventprocessor.NewRouter(&routerCfg, poisonPub, nil)
	if err != nil {
		components.Shutdown(context.Background())
		return nil, fmt.Errorf("create router: %w", err)
	}
	components.router = router
	logging.Info().
		Int("retry", routerCfg.RetryMaxRetries).
		Bool("dedup", routerCfg.DeduplicationEnabled).
		Bool("poison", cfg.NATS.RouterPoisonQueueEnabled).
		Msg("Watermill Router created")

	// Step 7: Create WebSocket handler and subscriber
	wsHandler, err := eventprocessor.NewWebSocketHandler(wsHub, nil)
	if err != nil {
		components.Shutdown(context.Background())
		return nil, fmt.Errorf("create WebSocket handler: %w", err)
	}
	components.wsHandler = wsHandler

	wsSubscriberCfg := eventprocessor.SubscriberConfig{
		URL:              natsURL,
		DurableName:      cfg.NATS.DurableName + "-websocket",
		QueueGroup:       cfg.NATS.QueueGroup + "-websocket",
		SubscribersCount: 1,
		AckWaitTimeout:   30 * time.Second,
		MaxDeliver:       3,
		MaxAckPending:    100,
		CloseTimeout:     10 * time.Second,
		MaxReconnects:    -1,
		ReconnectWait:    2 * time.Second,
		// Bind to existing stream to avoid AutoProvision trying to create
		// a stream from the wildcard topic name (playback.>)
		StreamName: streamCfg.Name,
	}
	wsSubscriber, err := eventprocessor.NewSubscriber(&wsSubscriberCfg, nil)
	if err != nil {
		components.Shutdown(context.Background())
		return nil, fmt.Errorf("create WebSocket subscriber: %w", err)
	}
	components.wsSubscriber = wsSubscriber

	// Register WebSocket handler with Router (no output publishing)
	router.AddConsumerHandler(
		"websocket-handler",
		"playback.>",
		wsSubscriber,
		wsHandler.Handle,
	)
	logging.Info().Msg("WebSocket handler registered with Router")

	// Step 8: Create DuckDB path with Consumer WAL protection (if database is provided)
	if db != nil {
		// Initialize Consumer WAL and create event store
		// Consumer WAL provides exactly-once delivery between NATS and DuckDB
		consumerWALComponents, eventStore, err := InitAndWireConsumerWAL(ctx, db)
		if err != nil {
			components.Shutdown(context.Background())
			return nil, fmt.Errorf("create event store: %w", err)
		}
		components.consumerWALComponents = consumerWALComponents
		components.duckdbStore = eventStore

		// Create appender for batch writes
		appenderCfg := eventprocessor.DefaultAppenderConfig()
		appenderCfg.BatchSize = cfg.NATS.BatchSize
		appenderCfg.FlushInterval = cfg.NATS.FlushInterval
		duckdbAppender, err := eventprocessor.NewAppender(eventStore, appenderCfg)
		if err != nil {
			components.Shutdown(context.Background())
			return nil, fmt.Errorf("create DuckDB appender: %w", err)
		}
		components.duckdbAppender = duckdbAppender
		logging.Info().
			Int("batch_size", appenderCfg.BatchSize).
			Dur("flush_interval", appenderCfg.FlushInterval).
			Msg("DuckDB appender created")

		// Wire appender to sync publisher for flush support
		// This enables deterministic sync completion by flushing events before reporting
		syncPublisher.SetAppender(duckdbAppender)

		// Create DuckDB handler with cross-source deduplication
		duckdbHandlerCfg := eventprocessor.DuckDBHandlerConfig{
			DeduplicationWindow:     cfg.NATS.RouterDeduplicationTTL,
			MaxDeduplicationEntries: 10000,
			EnableCrossSourceDedup:  true, // Dedup across Plex/Tautulli/Jellyfin
		}
		duckdbHandler, err := eventprocessor.NewDuckDBHandler(duckdbAppender, duckdbHandlerCfg, nil)
		if err != nil {
			components.Shutdown(context.Background())
			return nil, fmt.Errorf("create DuckDB handler: %w", err)
		}
		components.duckdbHandler = duckdbHandler

		// Create subscriber for DuckDB handler
		duckdbSubscriberCfg := eventprocessor.SubscriberConfig{
			URL:              natsURL,
			DurableName:      cfg.NATS.DurableName + "-duckdb",
			QueueGroup:       cfg.NATS.QueueGroup + "-duckdb",
			SubscribersCount: 1,
			AckWaitTimeout:   60 * time.Second,
			MaxDeliver:       5,
			MaxAckPending:    1000,
			CloseTimeout:     30 * time.Second,
			MaxReconnects:    -1,
			ReconnectWait:    2 * time.Second,
			// Bind to existing stream to avoid AutoProvision trying to create
			// a stream from the wildcard topic name (playback.>)
			StreamName: streamCfg.Name,
		}
		duckdbSubscriber, err := eventprocessor.NewSubscriber(&duckdbSubscriberCfg, nil)
		if err != nil {
			components.Shutdown(context.Background())
			return nil, fmt.Errorf("create DuckDB subscriber: %w", err)
		}
		components.duckdbSubscriber = duckdbSubscriber

		// Register DuckDB handler with Router (no output publishing)
		router.AddConsumerHandler(
			"duckdb-handler",
			"playback.>",
			duckdbSubscriber,
			duckdbHandler.Handle,
		)
		logging.Info().Msg("DuckDB handler registered with Router (cross-source dedup enabled)")
	} else {
		logging.Info().Msg("DuckDB path disabled (no database provided)")
	}

	// Step 9: Create detection handler (if detection engine is provided)
	if detectionEngine != nil {
		// Create detection handler
		detectionHandler, err := eventprocessor.NewDetectionHandler(detectionEngine, nil)
		if err != nil {
			components.Shutdown(context.Background())
			return nil, fmt.Errorf("create detection handler: %w", err)
		}
		components.detectionHandler = detectionHandler

		// Create subscriber for detection handler
		detectionSubscriberCfg := eventprocessor.SubscriberConfig{
			URL:              natsURL,
			DurableName:      cfg.NATS.DurableName + "-detection",
			QueueGroup:       cfg.NATS.QueueGroup + "-detection",
			SubscribersCount: 1,
			AckWaitTimeout:   30 * time.Second,
			MaxDeliver:       3,
			MaxAckPending:    500,
			CloseTimeout:     10 * time.Second,
			MaxReconnects:    -1,
			ReconnectWait:    2 * time.Second,
			StreamName:       streamCfg.Name,
		}
		detectionSubscriber, err := eventprocessor.NewSubscriber(&detectionSubscriberCfg, nil)
		if err != nil {
			components.Shutdown(context.Background())
			return nil, fmt.Errorf("create detection subscriber: %w", err)
		}
		components.detectionSubscriber = detectionSubscriber

		// Register detection handler with Router
		router.AddConsumerHandler(
			"detection-handler",
			"playback.>",
			detectionSubscriber,
			detectionHandler.Handle,
		)
		logging.Info().Msg("Detection handler registered with Router for anomaly detection")
	} else {
		logging.Info().Msg("Detection path disabled (no detection engine provided)")
	}

	// Step 10: Create and register health checker
	healthChecker := eventprocessor.NewHealthChecker(eventprocessor.DefaultHealthConfig())
	components.healthChecker = healthChecker

	// Register components for health checking
	if components.publisher != nil {
		healthChecker.RegisterComponent("publisher", components.publisher)
	}
	if components.router != nil {
		healthChecker.RegisterComponent("router", components.router)
	}
	if components.duckdbAppender != nil {
		healthChecker.RegisterComponent("appender", components.duckdbAppender)
	}
	logging.Info().Msg("Health checker initialized with NATS components")

	// Wire health checker to API
	api.SetNATSHealthChecker(healthChecker)
	logging.Info().Msg("Health checker wired to API endpoints")

	components.mu.Lock()
	components.running = true
	components.mu.Unlock()

	logging.Info().Msg("NATS event processing initialized successfully")
	return components, nil
}

// Start begins the NATS Router and all message processing.
// This should be called after InitNATS and after the server is ready.
//
// The Router starts all registered handlers (WebSocket, DuckDB) in a single
// coordinated lifecycle. The DuckDB appender is started first to ensure
// batch writes are ready before message consumption begins.
func (c *NATSComponents) Start(ctx context.Context) error {
	if c == nil {
		return nil
	}

	// Start DuckDB appender first (handles batch writes)
	if c.duckdbAppender != nil {
		logging.Info().Msg("Starting DuckDB appender...")
		if err := c.duckdbAppender.Start(ctx); err != nil {
			return fmt.Errorf("start DuckDB appender: %w", err)
		}
	}

	// Start the Router (runs all handlers: WebSocket + DuckDB)
	if c.router != nil {
		logging.Info().Msg("Starting Watermill Router...")
		running := c.router.RunAsync(ctx)
		// Wait for router to be running
		select {
		case <-running:
			logging.Info().Msg("Watermill Router started successfully")
		case <-ctx.Done():
			return fmt.Errorf("context canceled while starting router: %w", ctx.Err())
		}
	}

	logging.Info().Msg("All NATS components started")
	return nil
}

// Shutdown gracefully stops all NATS components.
//
// Shutdown order is critical for clean termination:
//  1. Stop Router first (stops all message handlers)
//  2. Close DuckDB appender (flushes remaining buffer)
//  3. Close subscribers (Watermill JetStream subscribers)
//  4. Close publisher
//  5. Shutdown WAL components
//  6. Close NATS connection
//  7. Shutdown embedded server last
func (c *NATSComponents) Shutdown(ctx context.Context) {
	if c == nil {
		return
	}

	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()

	logging.Info().Msg("Shutting down NATS components...")

	// Shutdown components in order using helper methods
	c.shutdownRouter()
	c.shutdownDuckDB()
	c.shutdownSubscribers()
	c.shutdownPublisher()
	c.shutdownWAL()
	c.shutdownConnection(ctx)

	close(c.shutdownComplete)
	logging.Info().Msg("NATS shutdown complete")
}

// shutdownRouter stops the Watermill Router.
func (c *NATSComponents) shutdownRouter() {
	if c.router == nil {
		return
	}
	if err := c.router.Close(); err != nil {
		logging.Error().Err(err).Msg("Error closing Router")
	}
	logging.Info().Msg("Watermill Router stopped")
}

// shutdownDuckDB closes DuckDB appender and flushes remaining buffer.
func (c *NATSComponents) shutdownDuckDB() {
	if c.duckdbAppender == nil {
		return
	}
	if err := c.duckdbAppender.Close(); err != nil {
		logging.Error().Err(err).Msg("Error closing DuckDB appender")
	}
	logging.Info().Msg("DuckDB appender closed")
}

// shutdownSubscribers closes all JetStream subscribers.
func (c *NATSComponents) shutdownSubscribers() {
	if c.duckdbSubscriber != nil {
		if err := c.duckdbSubscriber.Close(); err != nil {
			logging.Error().Err(err).Msg("Error closing DuckDB subscriber")
		}
		logging.Info().Msg("DuckDB subscriber closed")
	}
	if c.wsSubscriber != nil {
		if err := c.wsSubscriber.Close(); err != nil {
			logging.Error().Err(err).Msg("Error closing WebSocket subscriber")
		}
		logging.Info().Msg("WebSocket subscriber closed")
	}
	if c.detectionSubscriber != nil {
		if err := c.detectionSubscriber.Close(); err != nil {
			logging.Error().Err(err).Msg("Error closing detection subscriber")
		}
		logging.Info().Msg("Detection subscriber closed")
	}
}

// shutdownPublisher closes the NATS publisher.
func (c *NATSComponents) shutdownPublisher() {
	if c.publisher == nil {
		return
	}
	if err := c.publisher.Close(); err != nil {
		logging.Error().Err(err).Msg("Error closing publisher")
	}
	logging.Info().Msg("Publisher closed")
}

// shutdownWAL stops WAL components (retry loop, compactor, BadgerDB).
func (c *NATSComponents) shutdownWAL() {
	// Shutdown consumer WAL first (before producer WAL)
	if c.consumerWALComponents != nil {
		c.consumerWALComponents.Shutdown()
	}
	if c.walComponents == nil {
		return
	}
	c.walComponents.Shutdown()
}

// shutdownConnection closes NATS connection and embedded server.
func (c *NATSComponents) shutdownConnection(ctx context.Context) {
	if c.natsConn != nil {
		c.natsConn.Close()
		logging.Info().Msg("NATS connection closed")
	}
	if c.server != nil {
		if err := c.server.Shutdown(ctx); err != nil {
			logging.Error().Err(err).Msg("Error shutting down NATS server")
		}
		logging.Info().Msg("Embedded NATS server stopped")
	}
}

// IsRunning returns whether NATS components are active.
func (c *NATSComponents) IsRunning() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// BadgerDB returns the underlying BadgerDB instance from WAL.
// This allows other components (like import progress tracking) to share
// the same BadgerDB instance. Returns nil if WAL is not initialized.
func (c *NATSComponents) BadgerDB() interface{} {
	if c == nil || c.walComponents == nil {
		return nil
	}
	return c.walComponents.BadgerDB()
}

// EventPublisher returns the event publisher for wiring to additional managers.
// Returns nil if NATS is not initialized.
func (c *NATSComponents) EventPublisher() intsync.EventPublisher {
	if c == nil {
		return nil
	}
	return c.eventPublisher
}
