// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/metrics"
)

// MessageSource defines the interface for receiving messages.
// This abstraction allows the consumer to work with different message sources.
type MessageSource interface {
	// Subscribe subscribes to a topic and returns a channel of messages.
	Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error)
	// Close closes the message source.
	Close() error
}

// ConsumerConfig holds configuration for the DuckDB consumer.
type ConsumerConfig struct {
	// Topic is the NATS subject pattern to subscribe to (default: "playback.>")
	Topic string

	// EnableDeduplication enables event deduplication based on EventID
	EnableDeduplication bool

	// DeduplicationWindow is how long to remember event IDs for deduplication
	DeduplicationWindow time.Duration

	// MaxDeduplicationEntries is the maximum number of entries in the dedup cache
	MaxDeduplicationEntries int

	// WorkerCount is the number of concurrent message processors
	WorkerCount int

	// EnableDLQ enables the Dead Letter Queue for failed messages
	EnableDLQ bool

	// DLQConfig holds DLQ configuration when EnableDLQ is true
	DLQConfig DLQConfig
}

// DefaultConsumerConfig returns a ConsumerConfig with sensible defaults.
func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		Topic:                   "playback.>",
		EnableDeduplication:     true,
		DeduplicationWindow:     5 * time.Minute,
		MaxDeduplicationEntries: 10000,
		WorkerCount:             1,
	}
}

// ConsumerStats holds runtime statistics for monitoring.
type ConsumerStats struct {
	MessagesReceived  int64     // Total messages received
	MessagesProcessed int64     // Successfully processed messages
	ParseErrors       int64     // JSON parse failures
	DuplicatesSkipped int64     // Messages skipped due to deduplication
	MessagesSentToDLQ int64     // Messages sent to Dead Letter Queue
	LastMessageTime   time.Time // Time of last received message
}

// DuckDBConsumer consumes media events from JetStream and writes them to DuckDB.
// It handles deserialization, deduplication, and batch buffering through the Appender.
//
// Deprecated: DuckDBConsumer has been replaced by the Router-based approach using
// DuckDBHandler. New code should use Router.AddNoPublisherHandler with DuckDBHandler
// instead of creating a DuckDBConsumer directly. The Router-based approach provides
// automatic retry, poison queue routing, and middleware support.
//
// Migration example:
//
//	// Old approach (deprecated):
//	consumer, _ := NewDuckDBConsumer(subscriber, appender, &cfg)
//	consumer.Start(ctx)
//
//	// New approach:
//	handler, _ := NewDuckDBHandler(appender, handlerCfg, logger)
//	router.AddNoPublisherHandler("duckdb-handler", "playback.>", subscriber, handler.Handle)
//	router.Run(ctx)
//
// Performance: Uses BloomLRU for O(1) deduplication with ~90%+ fast-path rejections.
type DuckDBConsumer struct {
	source   MessageSource
	appender *Appender
	config   ConsumerConfig

	// Dead Letter Queue handler (nil if DLQ disabled)
	dlqHandler *DLQHandler

	// Deduplication cache using BloomLRU
	// Provides O(1) operations vs O(n) eviction with map-based approach
	dedupCache *cache.BloomLRU

	// State
	running atomic.Bool
	stopCh  chan struct{}
	doneCh  chan struct{}

	// Metrics
	messagesReceived  atomic.Int64
	messagesProcessed atomic.Int64
	parseErrors       atomic.Int64
	duplicatesSkipped atomic.Int64
	messagesSentToDLQ atomic.Int64
	lastMessageTime   atomic.Value // stores time.Time
}

// NewDuckDBConsumer creates a new DuckDB consumer.
// The appender should be started separately to enable batch flushing.
//
// Deprecated: Use NewDuckDBHandler with Router.AddNoPublisherHandler instead.
// See DuckDBConsumer type documentation for migration guide.
func NewDuckDBConsumer(source MessageSource, appender *Appender, cfg *ConsumerConfig) (*DuckDBConsumer, error) {
	if source == nil {
		return nil, fmt.Errorf("message source required")
	}
	if appender == nil {
		return nil, fmt.Errorf("appender required")
	}

	// Initialize BloomLRU with config values
	// BloomLRU provides O(1) operations vs O(n) eviction with map
	dedupCache := cache.NewBloomLRU(
		cfg.MaxDeduplicationEntries,
		cfg.DeduplicationWindow,
		0.01, // 1% false positive rate
	)

	c := &DuckDBConsumer{
		source:     source,
		appender:   appender,
		config:     *cfg,
		dedupCache: dedupCache,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}

	c.lastMessageTime.Store(time.Time{})

	// Create DLQ handler if enabled
	if cfg.EnableDLQ {
		dlqHandler, err := NewDLQHandler(cfg.DLQConfig)
		if err != nil {
			return nil, fmt.Errorf("create DLQ handler: %w", err)
		}
		c.dlqHandler = dlqHandler
	}

	return c, nil
}

// Start begins consuming messages from the source.
// Returns immediately - consumption happens in a goroutine.
func (c *DuckDBConsumer) Start(ctx context.Context) error {
	if c.running.Swap(true) {
		return nil // Already running
	}

	messages, err := c.source.Subscribe(ctx, c.config.Topic)
	if err != nil {
		c.running.Store(false)
		return fmt.Errorf("subscribe to %s: %w", c.config.Topic, err)
	}

	go c.consumeLoop(ctx, messages)

	if c.config.EnableDeduplication {
		go c.dedupCleanupLoop(ctx)
	}

	logging.Info().
		Str("topic", c.config.Topic).
		Bool("dedup", c.config.EnableDeduplication).
		Msg("DuckDB consumer started")
	return nil
}

// Stop gracefully stops the consumer.
func (c *DuckDBConsumer) Stop() {
	if !c.running.Swap(false) {
		return // Already stopped
	}

	close(c.stopCh)
	<-c.doneCh

	logging.Info().Msg("DuckDB consumer stopped")
}

// IsRunning returns whether the consumer is currently running.
func (c *DuckDBConsumer) IsRunning() bool {
	return c.running.Load()
}

// Stats returns current runtime statistics.
func (c *DuckDBConsumer) Stats() ConsumerStats {
	var lastTime time.Time
	if t, ok := c.lastMessageTime.Load().(time.Time); ok {
		lastTime = t
	}
	return ConsumerStats{
		MessagesReceived:  c.messagesReceived.Load(),
		MessagesProcessed: c.messagesProcessed.Load(),
		ParseErrors:       c.parseErrors.Load(),
		DuplicatesSkipped: c.duplicatesSkipped.Load(),
		MessagesSentToDLQ: c.messagesSentToDLQ.Load(),
		LastMessageTime:   lastTime,
	}
}

// DLQStats returns current DLQ statistics.
// Returns empty stats if DLQ is disabled.
func (c *DuckDBConsumer) DLQStats() DLQStats {
	if c.dlqHandler == nil {
		return DLQStats{}
	}
	return c.dlqHandler.Stats()
}

// consumeLoop processes messages from the subscription.
// DETERMINISM: Implements graceful shutdown with message draining to prevent data loss.
// When shutdown is signaled, it drains all pending messages before returning.
func (c *DuckDBConsumer) consumeLoop(ctx context.Context, messages <-chan *message.Message) {
	defer func() {
		c.running.Store(false)
		close(c.doneCh)
	}()

	for {
		select {
		case <-ctx.Done():
			// DETERMINISM: Drain remaining messages before shutdown to prevent data loss.
			// This ensures all messages received before shutdown are processed.
			c.drainMessages(messages)
			return
		case <-c.stopCh:
			// DETERMINISM: Drain remaining messages before shutdown to prevent data loss.
			c.drainMessages(messages)
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			c.processMessage(ctx, msg)
		}
	}
}

// drainMessages processes all remaining messages in the channel before shutdown.
// DETERMINISM: This ensures no messages are lost during graceful shutdown.
// Uses a timeout to prevent blocking indefinitely if the channel keeps receiving.
func (c *DuckDBConsumer) drainMessages(messages <-chan *message.Message) {
	// Use a short timeout to prevent blocking indefinitely
	// 100ms is enough to process buffered messages without significant delay
	drainTimeout := time.After(100 * time.Millisecond)
	drainedCount := 0

	for {
		select {
		case <-drainTimeout:
			if drainedCount > 0 {
				logging.Info().Int("count", drainedCount).Msg("DuckDB consumer drained messages during shutdown")
			}
			return
		case msg, ok := <-messages:
			if !ok {
				if drainedCount > 0 {
					logging.Info().Int("count", drainedCount).Msg("DuckDB consumer drained messages during shutdown (channel closed)")
				}
				return
			}
			// Use a background context since the original context is canceled
			c.processMessage(context.Background(), msg)
			drainedCount++
		default:
			// No more messages in buffer
			if drainedCount > 0 {
				logging.Info().Int("count", drainedCount).Msg("DuckDB consumer drained messages during shutdown")
			}
			return
		}
	}
}

// processMessage handles a single message.
func (c *DuckDBConsumer) processMessage(ctx context.Context, msg *message.Message) {
	startTime := time.Now()
	c.messagesReceived.Add(1)
	c.lastMessageTime.Store(startTime)

	// Record message consumption
	metrics.RecordNATSConsume()

	// Deserialize event
	var event MediaEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		c.parseErrors.Add(1)
		metrics.RecordNATSParseFailed()
		logging.Warn().
			Str("message_uuid", msg.UUID).
			Err(err).
			Msg("Failed to parse message")

		// Route parse errors to DLQ as permanent errors (malformed data)
		if c.dlqHandler != nil {
			// Create a placeholder event for DLQ tracking
			dlqEvent := &MediaEvent{EventID: msg.UUID}
			c.dlqHandler.AddEntry(dlqEvent, NewPermanentError("JSON parse error", err), msg.UUID)
			c.messagesSentToDLQ.Add(1)
		}

		msg.Ack() // Ack to prevent redelivery of malformed messages
		return
	}

	// Check for deduplication (both EventID and SessionKey)
	if c.config.EnableDeduplication && c.isDuplicate(&event) {
		c.duplicatesSkipped.Add(1)
		metrics.RecordNATSDeduplicated()
		msg.Ack()
		return
	}

	// Append to buffer for batch write
	if err := c.appender.Append(ctx, &event); err != nil {
		logging.Warn().
			Str("event_id", event.EventID).
			Err(err).
			Msg("Failed to append event")

		// Route append errors to DLQ
		if c.dlqHandler != nil {
			c.dlqHandler.AddEntry(&event, NewRetryableError("append failed", err), msg.UUID)
			c.messagesSentToDLQ.Add(1)
		}

		msg.Nack() // Nack for retry by NATS (in addition to DLQ tracking)
		return
	}

	// Record for deduplication (both EventID and SessionKey)
	if c.config.EnableDeduplication {
		c.recordEvent(&event)
	}

	c.messagesProcessed.Add(1)
	metrics.RecordNATSProcessed()
	metrics.RecordNATSProcessingDuration(time.Since(startTime))
	msg.Ack()
}

// isDuplicate checks if an event has been seen recently.
// It checks EventID, SessionKey (if present), and CorrelationKey (for cross-source dedup).
//
// Cross-source deduplication is CRITICAL for event sourcing mode where events from
// multiple sources (Tautulli sync, Plex webhooks, Jellyfin) may represent the same
// playback. The CorrelationKey uses (user_id, rating_key, time_bucket) to identify
// equivalent events across sources.
//
// Performance: Uses BloomLRU for O(1) lookups. ~90% of unique events
// short-circuit at the Bloom filter without touching the LRU cache.
func (c *DuckDBConsumer) isDuplicate(event *MediaEvent) bool {
	// Check EventID (same exact event)
	if c.dedupCache.IsDuplicate(event.EventID) {
		return true
	}

	// Check SessionKey if present (source-specific deduplication)
	if event.SessionKey != "" && event.SessionKey != event.EventID {
		if c.dedupCache.Contains(event.SessionKey) {
			return true
		}
	}

	// Check CorrelationKey if present (same-source deduplication)
	if event.CorrelationKey != "" {
		if c.dedupCache.Contains("corr:" + event.CorrelationKey) {
			return true
		}

		// Check cross-source key for CROSS-SOURCE deduplication
		// This strips the source prefix to allow dedup between different sources
		// e.g., Plex webhook and Tautulli sync for the same playback session
		// CRITICAL: Only check OTHER sources to prevent false positives within the same source
		crossSourceKey := getCrossSourceKey(event.CorrelationKey)
		if crossSourceKey != "" {
			// Extract source from this event's correlation key
			eventSource := getSourceFromCorrelationKey(event.CorrelationKey)

			// Only check cross-source dedup if this source participates
			if isKnownCrossSource(eventSource) {
				// Check if ANY OTHER source has recorded this same playback
				for _, otherSource := range knownCrossSources {
					if otherSource == eventSource {
						continue // Skip same source - this prevents false positives
					}
					if c.dedupCache.Contains("xsrc:" + otherSource + ":" + crossSourceKey) {
						return true
					}
				}
			}
		}
	}

	return false
}

// recordEvent adds event keys to the deduplication cache.
// It records EventID, SessionKey (if different), and CorrelationKey for comprehensive
// cross-source deduplication.
//
// Uses BloomLRU which handles capacity limits and TTL automatically with O(1) eviction.
func (c *DuckDBConsumer) recordEvent(event *MediaEvent) {
	// Record EventID (already recorded by IsDuplicate, but record to refresh TTL)
	c.dedupCache.Record(event.EventID)

	// Record SessionKey if different from EventID (source-specific)
	if event.SessionKey != "" && event.SessionKey != event.EventID {
		c.dedupCache.Record(event.SessionKey)
	}

	// Record CorrelationKey for same-source deduplication
	// Prefixed with "corr:" to avoid collision with EventID/SessionKey
	if event.CorrelationKey != "" {
		c.dedupCache.Record("corr:" + event.CorrelationKey)

		// Record cross-source key for cross-source deduplication
		// CRITICAL: Store with source prefix so other sources can check against it
		// This prevents false positives within the same source
		crossSourceKey := getCrossSourceKey(event.CorrelationKey)
		if crossSourceKey != "" {
			// Extract source from correlation key
			eventSource := getSourceFromCorrelationKey(event.CorrelationKey)

			// Only record for known cross-sources
			if isKnownCrossSource(eventSource) {
				// Record with source prefix: "xsrc:{source}:{crossSourceKey}"
				// Other sources will check "xsrc:{otherSource}:{crossSourceKey}"
				c.dedupCache.Record("xsrc:" + eventSource + ":" + crossSourceKey)
			}
		}
	}
	// Note: BloomLRU handles capacity limits automatically with O(1) LRU eviction
}

// dedupCleanupLoop periodically cleans up expired deduplication entries.
// The BloomLRU handles LRU eviction automatically, but this provides
// periodic cleanup of expired entries from the LRU portion.
func (c *DuckDBConsumer) dedupCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(c.config.DeduplicationWindow / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.dedupCache.CleanupExpired()
		}
	}
}
