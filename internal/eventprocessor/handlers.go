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

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/metrics"
	"github.com/tomtom215/cartographus/internal/models"
)

// DedupeAuditStore is the interface for logging deduplication decisions.
// This allows the handler to work with any storage implementation.
type DedupeAuditStore interface {
	// InsertDedupeAuditEntry logs a deduplication decision for audit and recovery.
	InsertDedupeAuditEntry(ctx context.Context, entry *models.DedupeAuditEntry) error
}

// DuckDBHandler processes media events for DuckDB persistence.
// It handles deserialization, cross-source deduplication, and batch appending.
//
// This handler is designed to work with the Watermill Router's middleware stack:
//   - Recoverer handles panics
//   - Retry handles transient failures
//   - PoisonQueue routes permanent failures to DLQ
//
// Cross-source deduplication (CorrelationKey) is handled internally because
// it requires application-level knowledge that simple middleware can't provide.
//
// Performance: Uses ExactLRU for O(1) deduplication with ZERO false positives.
// (v2.3: Changed from BloomLRU to ExactLRU to eliminate 1% false positive rate)
type DuckDBHandler struct {
	appender   *Appender
	config     DuckDBHandlerConfig
	logger     watermill.LoggerAdapter
	auditStore DedupeAuditStore // Optional: for logging dedupe decisions (ADR-0022)

	// Cross-source deduplication cache using ExactLRU (v2.3)
	// CRITICAL: Uses exact-match LRU for ZERO false positives
	// This prevents data loss from incorrectly marking unique events as duplicates
	dedupCache cache.DeduplicationCache

	// Metrics
	messagesReceived  atomic.Int64
	messagesProcessed atomic.Int64
	duplicatesSkipped atomic.Int64
	parseErrors       atomic.Int64
	lastMessageTime   atomic.Value // stores time.Time
}

// DuckDBHandlerConfig holds configuration for the DuckDB handler.
type DuckDBHandlerConfig struct {
	// EnableCrossSourceDedup enables correlation key deduplication
	// for detecting duplicate events from different sources (Plex, Tautulli, Jellyfin).
	EnableCrossSourceDedup bool

	// DeduplicationWindow is how long to remember correlation keys.
	DeduplicationWindow time.Duration

	// MaxDeduplicationEntries is the maximum cache size.
	MaxDeduplicationEntries int

	// EnableDedupeAudit enables logging of deduplication decisions (ADR-0022).
	// When enabled, each dedupe decision is recorded for visibility and recovery.
	EnableDedupeAudit bool

	// StoreRawPayload enables storing the full event payload in audit entries.
	// This allows restoration of incorrectly deduplicated events.
	// Uses more storage but enables full recovery capability.
	StoreRawPayload bool

	// SyncFlush forces synchronous flush after each append.
	// DETERMINISM: When true, ensures the database write completes before ACKing
	// the NATS message. This prevents data loss if the consumer crashes between
	// ACK and async flush, at the cost of higher latency.
	// Default: false (async batching for better performance)
	SyncFlush bool
}

// DefaultDuckDBHandlerConfig returns production defaults.
func DefaultDuckDBHandlerConfig() DuckDBHandlerConfig {
	return DuckDBHandlerConfig{
		EnableCrossSourceDedup:  true,
		DeduplicationWindow:     5 * time.Minute,
		MaxDeduplicationEntries: 10000,
		EnableDedupeAudit:       true, // Enable audit logging by default
		StoreRawPayload:         true, // Store full payload for recovery
	}
}

// NewDuckDBHandler creates a new handler for DuckDB persistence.
func NewDuckDBHandler(appender *Appender, cfg DuckDBHandlerConfig, logger watermill.LoggerAdapter) (*DuckDBHandler, error) {
	if appender == nil {
		return nil, fmt.Errorf("appender required")
	}
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	// Initialize ExactLRU for ZERO false positives (v2.3)
	// ExactLRU provides:
	// - O(1) operations (same as BloomLRU)
	// - ZERO false positives (vs 1% with BloomLRU)
	// - Exact string matching guarantees no data loss from incorrect dedup
	//
	// CRITICAL: For zero data loss requirements, ExactLRU is mandatory.
	// The slight memory overhead is justified by the guarantee of no false positives.
	dedupCache := cache.NewExactLRU(
		cfg.MaxDeduplicationEntries,
		cfg.DeduplicationWindow,
	)

	h := &DuckDBHandler{
		appender:   appender,
		config:     cfg,
		logger:     logger,
		dedupCache: dedupCache,
	}
	h.lastMessageTime.Store(time.Time{})

	return h, nil
}

// SetAuditStore sets the audit store for logging deduplication decisions.
// This is optional - if not set, dedupe decisions are not logged to the database.
func (h *DuckDBHandler) SetAuditStore(store DedupeAuditStore) {
	h.auditStore = store
}

// Handle processes a single media event message.
// This is the handler function passed to Router.AddNoPublisherHandler.
//
// Error handling:
//   - Parse errors return PermanentError (no retry, goes to DLQ)
//   - Append errors return error (triggers retry)
//   - Duplicates return nil (ack without processing)
func (h *DuckDBHandler) Handle(msg *message.Message) error {
	startTime := time.Now()
	msgCount := h.messagesReceived.Add(1)
	h.lastMessageTime.Store(startTime)
	metrics.RecordNATSConsume()

	// Store raw payload for potential audit logging (before parsing)
	rawPayload := msg.Payload

	// Deserialize event
	var event MediaEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		h.parseErrors.Add(1)
		metrics.RecordNATSParseFailed()
		h.logger.Error("Failed to parse message", err, watermill.LogFields{
			"message_uuid": msg.UUID,
		})
		// Return permanent error - no point retrying malformed JSON
		return NewPermanentError("JSON parse error", err)
	}

	// TRACING: Log every event received for end-to-end data loss investigation
	logging.Trace().
		Int64("msg_count", msgCount).
		Str("session_key", event.SessionKey).
		Str("event_id", event.EventID).
		Str("correlation_key", event.CorrelationKey).
		Str("username", event.Username).
		Msg("HANDLER: RECEIVED")

	// Cross-source deduplication (EventID, SessionKey, CorrelationKey)
	// Pass raw payload for audit logging if enabled
	if h.config.EnableCrossSourceDedup && h.isDuplicateWithAudit(&event, rawPayload) {
		dupCount := h.duplicatesSkipped.Add(1)
		metrics.RecordNATSDeduplicated()
		logging.Trace().
			Int64("msg_count", msgCount).
			Str("session_key", event.SessionKey).
			Str("event_id", event.EventID).
			Int64("total_dupes", dupCount).
			Msg("HANDLER: DEDUPLICATED")
		// Return nil to acknowledge - this is expected behavior, not an error
		return nil
	}

	// Append to buffer for batch write
	ctx := context.Background() // Router provides message context via msg.Context()
	if msgCtx := msg.Context(); msgCtx != nil {
		ctx = msgCtx
	}

	if err := h.appender.Append(ctx, &event); err != nil {
		h.logger.Error("Failed to append event", err, watermill.LogFields{
			"event_id": event.EventID,
		})
		// Return retryable error - append might succeed later
		return NewRetryableError("append failed", err)
	}

	// DETERMINISM: If SyncFlush is enabled, flush immediately to ensure
	// the database write completes before we ACK the NATS message.
	// This prevents data loss if the consumer crashes between ACK and async flush.
	if h.config.SyncFlush {
		if err := h.appender.Flush(ctx); err != nil {
			h.logger.Error("Failed to flush event synchronously", err, watermill.LogFields{
				"event_id": event.EventID,
			})
			// Return retryable error - flush might succeed later
			return NewRetryableError("sync flush failed", err)
		}
	}

	// Record for deduplication
	if h.config.EnableCrossSourceDedup {
		h.recordEvent(&event)
	}

	processedCount := h.messagesProcessed.Add(1)
	metrics.RecordNATSProcessed()
	metrics.RecordNATSProcessingDuration(time.Since(startTime))

	// TRACING: Log every successful append for end-to-end data loss investigation
	logging.Trace().
		Int64("msg_count", msgCount).
		Str("session_key", event.SessionKey).
		Str("event_id", event.EventID).
		Int64("processed", processedCount).
		Msg("HANDLER: APPENDED")

	return nil
}

// isDuplicateWithAudit checks if an event has been seen recently and logs audit entries.
// Checks EventID, SessionKey, and CorrelationKey for cross-source deduplication.
//
// Performance: Uses ExactLRU for O(1) lookups with ZERO false positives.
// (v2.3: Changed from BloomLRU to ExactLRU to eliminate false positive data loss)
//
// When audit logging is enabled, each duplicate detection is recorded to the
// dedupe_audit_log table for visibility and potential recovery (ADR-0022).
func (h *DuckDBHandler) isDuplicateWithAudit(event *MediaEvent, rawPayload []byte) bool {
	// Check EventID (primary dedup key)
	// NOTE: IsDuplicate both checks AND records if new - this is the expected behavior
	if h.dedupCache.IsDuplicate(event.EventID) {
		logging.Debug().
			Str("event_id", event.EventID).
			Str("session_key", event.SessionKey).
			Msg("DEDUP: DUPLICATE by EventID")
		h.logger.Debug("Duplicate detected by EventID", watermill.LogFields{
			"event_id":    event.EventID,
			"session_key": event.SessionKey,
		})
		h.logDedupeDecision(event, rawPayload, "event_id")
		return true
	}

	// Check SessionKey if different from EventID
	if event.SessionKey != "" && event.SessionKey != event.EventID {
		if h.dedupCache.Contains(event.SessionKey) {
			logging.Debug().
				Str("session_key", event.SessionKey).
				Str("event_id", event.EventID).
				Msg("DEDUP: DUPLICATE by SessionKey")
			h.logger.Debug("Duplicate detected by SessionKey", watermill.LogFields{
				"event_id":    event.EventID,
				"session_key": event.SessionKey,
			})
			h.logDedupeDecision(event, rawPayload, "session_key")
			return true
		}
	}

	// Check CorrelationKey for same-source deduplication
	if event.CorrelationKey != "" {
		if h.dedupCache.Contains("corr:" + event.CorrelationKey) {
			logging.Debug().
				Str("correlation_key", event.CorrelationKey).
				Str("event_id", event.EventID).
				Str("session_key", event.SessionKey).
				Msg("DEDUP: DUPLICATE by CorrelationKey")
			h.logger.Debug("Duplicate detected by CorrelationKey", watermill.LogFields{
				"event_id":        event.EventID,
				"correlation_key": event.CorrelationKey,
			})
			h.logDedupeDecision(event, rawPayload, "correlation_key")
			return true
		}

		// Check cross-source key for cross-source deduplication
		// This strips the source prefix to allow dedup between e.g., Plex webhook + Tautulli sync
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
					if h.dedupCache.Contains("xsrc:" + otherSource + ":" + crossSourceKey) {
						logging.Debug().
							Str("cross_source_key", crossSourceKey).
							Str("event_id", event.EventID).
							Str("matched_source", otherSource).
							Msg("DEDUP: DUPLICATE by CrossSourceKey")
						h.logger.Debug("Duplicate detected by CrossSourceKey", watermill.LogFields{
							"event_id":         event.EventID,
							"correlation_key":  event.CorrelationKey,
							"cross_source_key": crossSourceKey,
							"event_source":     eventSource,
							"matched_source":   otherSource,
						})
						h.logDedupeDecision(event, rawPayload, "cross_source_key")
						return true
					}
				}
			}
		}
	}

	return false
}

// logDedupeDecision logs a deduplication decision to the audit store.
// This is called when an event is detected as a duplicate and discarded at the bloom cache layer.
func (h *DuckDBHandler) logDedupeDecision(event *MediaEvent, rawPayload []byte, reason string) {
	// Skip if audit logging is disabled or no store configured
	if !h.config.EnableDedupeAudit || h.auditStore == nil {
		return
	}

	// Build audit entry
	// Layer is always "bloom_cache" since this function is called from the bloom cache dedup check
	entry := &models.DedupeAuditEntry{
		ID:                      uuid.New(),
		Timestamp:               time.Now(),
		DiscardedEventID:        event.EventID,
		DiscardedSessionKey:     event.SessionKey,
		DiscardedCorrelationKey: event.CorrelationKey,
		DiscardedSource:         event.Source,
		DedupeReason:            reason,
		DedupeLayer:             "bloom_cache",
		UserID:                  event.UserID,
		Username:                event.Username,
		MediaType:               event.MediaType,
		Title:                   event.Title,
		RatingKey:               event.RatingKey,
		Status:                  "auto_dedupe",
		CreatedAt:               time.Now(),
	}

	// Set started_at if available
	if !event.StartedAt.IsZero() {
		entry.DiscardedStartedAt = &event.StartedAt
	}

	// Store raw payload for recovery if enabled
	if h.config.StoreRawPayload && len(rawPayload) > 0 {
		entry.DiscardedRawPayload = rawPayload
	}

	// Insert audit entry asynchronously (don't block message processing)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.auditStore.InsertDedupeAuditEntry(ctx, entry); err != nil {
			h.logger.Error("Failed to insert dedupe audit entry", err, watermill.LogFields{
				"event_id":      event.EventID,
				"dedupe_reason": reason,
			})
		}
	}()
}

// recordEvent adds event keys to the deduplication cache.
// Uses BloomLRU which handles capacity limits and TTL automatically.
func (h *DuckDBHandler) recordEvent(event *MediaEvent) {
	// Record EventID (already recorded by IsDuplicate, but record to refresh TTL)
	h.dedupCache.Record(event.EventID)

	// Record SessionKey if different from EventID
	if event.SessionKey != "" && event.SessionKey != event.EventID {
		h.dedupCache.Record(event.SessionKey)
	}

	// Record CorrelationKey for same-source dedup
	if event.CorrelationKey != "" {
		h.dedupCache.Record("corr:" + event.CorrelationKey)

		// Record cross-source key for cross-source dedup
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
				h.dedupCache.Record("xsrc:" + eventSource + ":" + crossSourceKey)
			}
		}
	}
	// Note: BloomLRU handles capacity limits automatically with O(1) LRU eviction
}

// getCrossSourceKey is an internal wrapper around GetCrossSourceKey for handlers.
// See GetCrossSourceKey in events.go for full documentation.
func getCrossSourceKey(corrKey string) string {
	return GetCrossSourceKey(corrKey)
}

// knownCrossSources lists media server sources that participate in cross-source deduplication.
// Only events from these sources will be checked/recorded for cross-source matching.
// This prevents false positive deduplication from unknown or custom sources.
var knownCrossSources = []string{SourcePlex, SourceTautulli, SourceJellyfin, SourceEmby}

// getSourceFromCorrelationKey extracts the source (first part) from a correlation key.
// Returns empty string if the key is invalid.
//
// Format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
// Example: "plex:default:12345:54321:device123:2024-01-15T10:32:00:session-abc" -> "plex"
func getSourceFromCorrelationKey(corrKey string) string {
	if corrKey == "" {
		return ""
	}
	for i := 0; i < len(corrKey); i++ {
		if corrKey[i] == ':' {
			return corrKey[:i]
		}
	}
	return ""
}

// isKnownCrossSource checks if a source participates in cross-source deduplication.
func isKnownCrossSource(source string) bool {
	for _, s := range knownCrossSources {
		if s == source {
			return true
		}
	}
	return false
}

// StartCleanup starts a goroutine to periodically clean up expired entries.
// The BloomLRU handles LRU eviction automatically, but this provides
// periodic cleanup of expired entries from the LRU portion.
func (h *DuckDBHandler) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(h.config.DeduplicationWindow / 2)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.dedupCache.CleanupExpired()
			}
		}
	}()
}

// Stats returns current handler statistics.
func (h *DuckDBHandler) Stats() DuckDBHandlerStats {
	var lastTime time.Time
	if t, ok := h.lastMessageTime.Load().(time.Time); ok {
		lastTime = t
	}

	return DuckDBHandlerStats{
		MessagesReceived:  h.messagesReceived.Load(),
		MessagesProcessed: h.messagesProcessed.Load(),
		DuplicatesSkipped: h.duplicatesSkipped.Load(),
		ParseErrors:       h.parseErrors.Load(),
		LastMessageTime:   lastTime,
	}
}

// DuckDBHandlerStats holds runtime statistics.
type DuckDBHandlerStats struct {
	MessagesReceived  int64
	MessagesProcessed int64
	DuplicatesSkipped int64
	ParseErrors       int64
	LastMessageTime   time.Time
}

// WebSocketHandler broadcasts media events to WebSocket clients.
// It converts MediaEvent to the format expected by the WebSocket hub.
type WebSocketHandler struct {
	hub    WebSocketBroadcaster
	logger watermill.LoggerAdapter

	messagesReceived  atomic.Int64
	messagesBroadcast atomic.Int64
}

// WebSocketBroadcaster defines the interface for broadcasting to WebSocket clients.
// This allows the handler to work with any WebSocket implementation.
type WebSocketBroadcaster interface {
	// BroadcastRaw sends raw JSON bytes to all connected clients.
	BroadcastRaw(data []byte)
}

// NewWebSocketHandler creates a new handler for WebSocket broadcasting.
func NewWebSocketHandler(hub WebSocketBroadcaster, logger watermill.LoggerAdapter) (*WebSocketHandler, error) {
	if hub == nil {
		return nil, fmt.Errorf("hub required")
	}
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
	}, nil
}

// Handle broadcasts a message to WebSocket clients.
// This handler always succeeds (returns nil) because broadcast failure
// shouldn't stop message processing or trigger retries.
func (h *WebSocketHandler) Handle(msg *message.Message) error {
	h.messagesReceived.Add(1)

	// Broadcast raw payload - let the hub handle formatting
	h.hub.BroadcastRaw(msg.Payload)
	h.messagesBroadcast.Add(1)

	return nil
}

// Stats returns current handler statistics.
func (h *WebSocketHandler) Stats() WebSocketHandlerStats {
	return WebSocketHandlerStats{
		MessagesReceived:  h.messagesReceived.Load(),
		MessagesBroadcast: h.messagesBroadcast.Load(),
	}
}

// WebSocketHandlerStats holds runtime statistics.
type WebSocketHandlerStats struct {
	MessagesReceived  int64
	MessagesBroadcast int64
}
