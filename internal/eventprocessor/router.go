// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"github.com/tomtom215/cartographus/internal/cache"
)

// RouterConfig holds configuration for the Watermill Router.
type RouterConfig struct {
	// CloseTimeout is how long to wait for handlers to finish when closing.
	CloseTimeout time.Duration

	// Retry configuration
	RetryMaxRetries      int
	RetryInitialInterval time.Duration
	RetryMaxInterval     time.Duration
	RetryMultiplier      float64

	// Throttle configuration (messages per second, 0 = disabled)
	ThrottlePerSecond int64

	// PoisonQueue configuration
	PoisonQueueTopic string

	// Deduplication configuration
	DeduplicationEnabled bool
	DeduplicationTTL     time.Duration
}

// DefaultRouterConfig returns production defaults for the Router.
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		CloseTimeout:         30 * time.Second,
		RetryMaxRetries:      5,
		RetryInitialInterval: time.Second,
		RetryMaxInterval:     time.Minute,
		RetryMultiplier:      2.0,
		ThrottlePerSecond:    0, // Disabled by default
		PoisonQueueTopic:     "dlq.playback",
		DeduplicationEnabled: false, // DISABLED: Uses msg.UUID which may be regenerated, causing data loss (see commit 8e98500)
		DeduplicationTTL:     5 * time.Minute,
	}
}

// Router wraps the Watermill Router with pre-configured middleware.
// It provides automatic Ack/Nack handling, retry logic, circuit breaker,
// panic recovery, and poison queue routing for failed messages.
type Router struct {
	router       *message.Router
	config       RouterConfig
	logger       watermill.LoggerAdapter
	poisonPub    message.Publisher
	running      bool
	handlers     map[string]*message.Handler
	dedupRepo    *InMemoryDeduplicator
	metricsStore *RouterMetrics
}

// RouterMetrics holds runtime metrics for the Router.
type RouterMetrics struct {
	MessagesReceived     int64
	MessagesProcessed    int64
	MessagesFailed       int64
	MessagesRetried      int64
	MessagesPoisoned     int64
	MessagesDeduplicated int64
}

// InMemoryDeduplicator implements middleware.ExpiringKeyRepository for message deduplication.
// This is used for simple deduplication (exact message ID matching).
// Cross-source deduplication is handled separately in DuckDBConsumer.
//
// Performance: Uses LRUCache for O(1) operations with automatic capacity management.
type InMemoryDeduplicator struct {
	cache *cache.LRUCache
}

// NewInMemoryDeduplicator creates a new in-memory deduplicator.
// Uses LRU cache with 10000 entry capacity for bounded memory usage.
func NewInMemoryDeduplicator(ttl time.Duration) *InMemoryDeduplicator {
	return &InMemoryDeduplicator{
		cache: cache.NewLRUCache(10000, ttl),
	}
}

// IsDuplicate checks if a key exists and hasn't expired.
// Returns true if duplicate, false if new.
// Implements middleware.ExpiringKeyRepository interface.
//
// Performance: O(1) operations vs O(n) eviction with map-based approach.
func (d *InMemoryDeduplicator) IsDuplicate(_ context.Context, key string) (bool, error) {
	return d.cache.IsDuplicate(key), nil
}

// NewRouter creates a new Watermill Router with pre-configured middleware.
// The router handles:
//   - Automatic Ack/Nack based on handler success/failure
//   - Panic recovery with stack trace logging
//   - Exponential backoff retry for transient failures
//   - Poison queue routing for permanent failures
//   - Optional rate limiting (throttling)
//   - Optional simple deduplication (for exact message matches)
func NewRouter(
	cfg *RouterConfig,
	poisonPublisher message.Publisher,
	logger watermill.LoggerAdapter,
) (*Router, error) {
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	if cfg == nil {
		defaultCfg := DefaultRouterConfig()
		cfg = &defaultCfg
	}

	// Create base router
	routerCfg := message.RouterConfig{
		CloseTimeout: cfg.CloseTimeout,
	}

	wmRouter, err := message.NewRouter(routerCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("create watermill router: %w", err)
	}

	r := &Router{
		router:       wmRouter,
		config:       *cfg,
		logger:       logger,
		poisonPub:    poisonPublisher,
		handlers:     make(map[string]*message.Handler),
		metricsStore: &RouterMetrics{},
	}

	// Add signal handler plugin for graceful shutdown
	wmRouter.AddPlugin(plugin.SignalsHandler)

	// Add middleware in order (outer to inner):
	// 1. Recoverer - catch panics and convert to errors
	// 2. Retry - handle transient failures with backoff
	// 3. Throttle - rate limiting (if enabled)
	// 4. Deduplicator - simple dedup (if enabled)
	// 5. Poison Queue - route permanent failures to DLQ

	// Recoverer: Convert panics to errors
	wmRouter.AddMiddleware(middleware.Recoverer)

	// Retry: Exponential backoff for transient failures
	retryMiddleware := middleware.Retry{
		MaxRetries:      cfg.RetryMaxRetries,
		InitialInterval: cfg.RetryInitialInterval,
		MaxInterval:     cfg.RetryMaxInterval,
		Multiplier:      cfg.RetryMultiplier,
		Logger:          logger,
	}
	wmRouter.AddMiddleware(retryMiddleware.Middleware)

	// Throttle: Rate limiting (if enabled)
	if cfg.ThrottlePerSecond > 0 {
		throttle := middleware.NewThrottle(cfg.ThrottlePerSecond, time.Second)
		wmRouter.AddMiddleware(throttle.Middleware)
	}

	// Deduplicator: Simple message ID deduplication (if enabled)
	if cfg.DeduplicationEnabled {
		r.dedupRepo = NewInMemoryDeduplicator(cfg.DeduplicationTTL)
		dedup := middleware.Deduplicator{
			KeyFactory: func(msg *message.Message) (string, error) {
				return msg.UUID, nil
			},
			Repository: r.dedupRepo,
		}
		wmRouter.AddMiddleware(dedup.Middleware)
	}

	// Poison Queue: Route messages that fail after all retries
	if poisonPublisher != nil && cfg.PoisonQueueTopic != "" {
		poisonQueue, err := middleware.PoisonQueue(poisonPublisher, cfg.PoisonQueueTopic)
		if err != nil {
			return nil, fmt.Errorf("create poison queue middleware: %w", err)
		}
		wmRouter.AddMiddleware(poisonQueue)
	}

	return r, nil
}

// AddHandler registers a handler for processing messages from a topic.
// The handler function should process the message and return any output messages.
// Errors trigger retry logic; permanent failures go to the poison queue.
//
// Parameters:
//   - name: Unique handler name for logging and metrics
//   - subscribeTopic: NATS subject to subscribe to (supports wildcards like "playback.>")
//   - subscriber: Watermill Subscriber for the input topic
//   - publishTopic: NATS subject to publish output messages (empty string if no output)
//   - publisher: Watermill Publisher for output (nil if no output)
//   - handler: Function that processes messages
func (r *Router) AddHandler(
	name string,
	subscribeTopic string,
	subscriber message.Subscriber,
	publishTopic string,
	publisher message.Publisher,
	handler message.HandlerFunc,
) *message.Handler {
	h := r.router.AddHandler(
		name,
		subscribeTopic,
		subscriber,
		publishTopic,
		publisher,
		handler,
	)
	r.handlers[name] = h
	return h
}

// AddConsumerHandler registers a handler that doesn't produce output messages.
// This is a convenience wrapper for consumers that only read and process messages.
// Note: This replaces the deprecated AddNoPublisherHandler.
func (r *Router) AddConsumerHandler(
	name string,
	subscribeTopic string,
	subscriber message.Subscriber,
	handler message.NoPublishHandlerFunc,
) *message.Handler {
	h := r.router.AddConsumerHandler(
		name,
		subscribeTopic,
		subscriber,
		handler,
	)
	r.handlers[name] = h
	return h
}

// AddHandlerMiddleware adds middleware to a specific handler.
// Handler-level middleware runs after router-level middleware.
func (r *Router) AddHandlerMiddleware(handlerName string, m ...message.HandlerMiddleware) error {
	h, exists := r.handlers[handlerName]
	if !exists {
		return fmt.Errorf("handler %q not found", handlerName)
	}
	h.AddMiddleware(m...)
	return nil
}

// Run starts the router and blocks until context cancellation or Close().
// All registered handlers begin processing messages.
func (r *Router) Run(ctx context.Context) error {
	r.running = true
	defer func() { r.running = false }()
	return r.router.Run(ctx)
}

// RunAsync starts the router in a goroutine and returns immediately.
// Returns a channel that will be closed when the router is running.
func (r *Router) RunAsync(ctx context.Context) <-chan struct{} {
	running := make(chan struct{})

	go func() {
		// Start router in background
		go func() {
			r.running = true
			defer func() { r.running = false }()
			if err := r.router.Run(ctx); err != nil {
				r.logger.Error("Router error", err, nil)
			}
		}()

		// Wait for router to be running, then signal
		<-r.router.Running()
		close(running)
	}()

	return running
}

// Running returns a channel that closes when the router is running.
func (r *Router) Running() <-chan struct{} {
	return r.router.Running()
}

// Close gracefully stops the router.
// Waits for in-flight messages to complete up to CloseTimeout.
func (r *Router) Close() error {
	return r.router.Close()
}

// IsRunning returns whether the router is currently processing messages.
func (r *Router) IsRunning() bool {
	return r.running
}

// Metrics returns current router metrics.
func (r *Router) Metrics() *RouterMetrics {
	return r.metricsStore
}

// HealthCheck implements HealthCheckable interface.
// It returns the health status of the Router based on whether it's running.
func (r *Router) HealthCheck(ctx context.Context) ComponentHealth {
	health := ComponentHealth{
		Name:      "router",
		LastCheck: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if r.running {
		health.Healthy = true
		health.Message = "Router is running"
		health.Details["handlers"] = len(r.handlers)
		health.Details["messages_received"] = r.metricsStore.MessagesReceived
		health.Details["messages_processed"] = r.metricsStore.MessagesProcessed
		health.Details["messages_failed"] = r.metricsStore.MessagesFailed
	} else {
		health.Healthy = false
		health.Error = "Router is not running"
	}

	return health
}
