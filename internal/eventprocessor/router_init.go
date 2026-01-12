// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RouterComponents holds all Router-based event processing components.
// This provides a cleaner alternative to the current manual subscribe loop approach.
type RouterComponents struct {
	Router           *Router
	DuckDBHandler    *DuckDBHandler
	WebSocketHandler *WebSocketHandler
	EventBus         *EventBus
	Forwarder        *Forwarder
	Logger           watermill.LoggerAdapter
}

// RouterComponentsConfig holds configuration for creating RouterComponents.
type RouterComponentsConfig struct {
	// RouterConfig for the Watermill Router (pointer to avoid copy overhead)
	RouterConfig *RouterConfig

	// DuckDBHandlerConfig for event persistence
	DuckDBHandlerConfig DuckDBHandlerConfig

	// ForwarderConfig for transactional outbox (optional)
	ForwarderConfig ForwarderConfig

	// EnableForwarder controls whether the Forwarder is created
	EnableForwarder bool

	// PoisonQueuePublisher is used for routing failed messages
	// Set to nil to disable poison queue
	PoisonQueuePublisher message.Publisher

	// WebSocketBroadcaster for real-time updates (optional)
	WebSocketBroadcaster WebSocketBroadcaster
}

// DefaultRouterComponentsConfig returns production defaults.
func DefaultRouterComponentsConfig() RouterComponentsConfig {
	defaultRouterCfg := DefaultRouterConfig()
	return RouterComponentsConfig{
		RouterConfig:        &defaultRouterCfg,
		DuckDBHandlerConfig: DefaultDuckDBHandlerConfig(),
		ForwarderConfig:     DefaultForwarderConfig(),
		EnableForwarder:     false,
	}
}

// NewRouterComponents creates all Router-based components wired together.
// This is the recommended way to initialize the event processing system.
//
// Usage example:
//
//	cfg := DefaultRouterComponentsConfig()
//	cfg.WebSocketBroadcaster = wsHub // Your WebSocket hub
//
//	components, err := NewRouterComponents(
//	    cfg,
//	    appender,              // Your Appender instance
//	    publisher,             // Watermill Publisher for events
//	    duckdbSubscriber,      // Subscriber for DuckDB path
//	    websocketSubscriber,   // Subscriber for WebSocket path (optional)
//	    nil,                   // Logger (uses default if nil)
//	)
//
//	// Start processing
//	ctx := context.Background()
//	if err := components.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Cleanup
//	defer components.Stop()
func NewRouterComponents(
	cfg *RouterComponentsConfig,
	appender *Appender,
	publisher message.Publisher,
	duckdbSubscriber message.Subscriber,
	websocketSubscriber message.Subscriber,
	logger watermill.LoggerAdapter,
) (*RouterComponents, error) {
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}
	if cfg == nil {
		defaultCfg := DefaultRouterComponentsConfig()
		cfg = &defaultCfg
	}

	components := &RouterComponents{
		Logger: logger,
	}

	// Create Router with middleware stack
	router, err := NewRouter(cfg.RouterConfig, cfg.PoisonQueuePublisher, logger)
	if err != nil {
		return nil, fmt.Errorf("create router: %w", err)
	}
	components.Router = router

	// Create DuckDB handler if appender provided
	if appender != nil && duckdbSubscriber != nil {
		duckdbHandler, err := NewDuckDBHandler(appender, cfg.DuckDBHandlerConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("create duckdb handler: %w", err)
		}
		components.DuckDBHandler = duckdbHandler

		// Register with router
		router.AddConsumerHandler(
			"duckdb-consumer",
			"playback.>",
			duckdbSubscriber,
			duckdbHandler.Handle,
		)
	}

	// Create WebSocket handler if broadcaster provided
	if cfg.WebSocketBroadcaster != nil && websocketSubscriber != nil {
		wsHandler, err := NewWebSocketHandler(cfg.WebSocketBroadcaster, logger)
		if err != nil {
			return nil, fmt.Errorf("create websocket handler: %w", err)
		}
		components.WebSocketHandler = wsHandler

		// Register with router
		router.AddConsumerHandler(
			"websocket-broadcaster",
			"playback.>",
			websocketSubscriber,
			wsHandler.Handle,
		)
	}

	// Create EventBus for type-safe publishing
	if publisher != nil {
		eventBusCfg := DefaultEventBusConfig()
		// Use default topic generation - already produces "playback.<eventName>"

		eventBus, err := NewEventBus(publisher, eventBusCfg, logger)
		if err != nil {
			return nil, fmt.Errorf("create event bus: %w", err)
		}
		components.EventBus = eventBus
	}

	return components, nil
}

// Start begins processing events.
func (c *RouterComponents) Start(ctx context.Context) error {
	// Start dedup cleanup for DuckDB handler
	if c.DuckDBHandler != nil {
		c.DuckDBHandler.StartCleanup(ctx)
	}

	// Start forwarder if configured
	if c.Forwarder != nil {
		if err := c.Forwarder.Start(ctx); err != nil {
			return fmt.Errorf("start forwarder: %w", err)
		}
	}

	// Start router (runs handlers)
	go func() {
		if err := c.Router.Run(ctx); err != nil {
			c.Logger.Error("Router error", err, nil)
		}
	}()

	// Wait for router to be running
	<-c.Router.Running()

	c.Logger.Info("Router components started", watermill.LogFields{
		"handlers": len(c.Router.handlers),
	})

	return nil
}

// Stop gracefully stops all components.
func (c *RouterComponents) Stop() error {
	var errs []error

	if c.Forwarder != nil {
		if err := c.Forwarder.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop forwarder: %w", err))
		}
	}

	if c.Router != nil {
		if err := c.Router.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close router: %w", err))
		}
	}

	c.Logger.Info("Router components stopped", nil)

	if len(errs) > 0 {
		return errs[0] // Return first error
	}
	return nil
}

// IsRunning returns whether components are active.
func (c *RouterComponents) IsRunning() bool {
	return c.Router != nil && c.Router.IsRunning()
}

// Stats returns combined statistics from all components.
func (c *RouterComponents) Stats() RouterComponentsStats {
	stats := RouterComponentsStats{}

	if c.Router != nil {
		stats.Router = c.Router.Metrics()
	}
	if c.DuckDBHandler != nil {
		stats.DuckDB = c.DuckDBHandler.Stats()
	}
	if c.WebSocketHandler != nil {
		stats.WebSocket = c.WebSocketHandler.Stats()
	}

	return stats
}

// RouterComponentsStats holds combined statistics.
type RouterComponentsStats struct {
	Router    *RouterMetrics
	DuckDB    DuckDBHandlerStats
	WebSocket WebSocketHandlerStats
}

// MigrationGuide provides documentation for migrating from manual loops to Router.
//
// Current approach (manual loops):
//
//	subscriber.NewMessageHandler("playback.>").
//	    Handle(func(ctx context.Context, msg *message.Message) error {
//	        // Manual parsing, ack/nack, error handling
//	        return nil
//	    }).
//	    Run(ctx)
//
// New approach (Router-based):
//
//	cfg := DefaultRouterComponentsConfig()
//	components, _ := NewRouterComponents(cfg, appender, pub, sub, wsSub, nil)
//	components.Start(ctx)
//
// Benefits of Router-based approach:
//  1. Automatic Ack/Nack - No manual msg.Ack()/msg.Nack() calls
//  2. Built-in retry - Exponential backoff on transient errors
//  3. Poison queue - Failed messages automatically routed to DLQ
//  4. Panic recovery - Handlers won't crash the entire system
//  5. Rate limiting - Throttle high-volume streams
//  6. Deduplication - Simple message ID dedup at middleware level
//  7. Type safety - CQRS handlers receive concrete types
//  8. Transactional - Forwarder ensures atomic DB + publish
//
// Migration steps:
//  1. Create RouterComponentsConfig with your settings
//  2. Call NewRouterComponents with existing appender/subscribers
//  3. Replace Start calls with components.Start()
//  4. Remove manual MessageHandler usage
//  5. Use EventBus for type-safe publishing
type MigrationGuide struct{}
