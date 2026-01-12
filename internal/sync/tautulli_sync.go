// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/rs/zerolog"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/metrics"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// errSkipRecord is a sentinel error indicating that a record should be skipped gracefully
var errSkipRecord = errors.New("skip record")

// performInitialSync performs the first synchronization on startup
//
// RACE CONDITION FIX (v2.4): Acquires syncMu to prevent concurrent execution
// with syncLoop or TriggerSync. Without this lock, the initial sync could run
// concurrently with periodic syncs if they started before initial sync completed.
func (m *Manager) performInitialSync() error {
	logging.Info().Msg("Performing initial sync...")

	// RACE CONDITION FIX: Acquire sync mutex to prevent concurrent sync execution
	m.syncMu.Lock()
	defer m.syncMu.Unlock()

	since := m.getSyncStartTime()
	return m.syncDataSince(context.Background(), since)
}

// getSyncStartTime returns the appropriate start time for sync operations.
// When SyncAll is enabled, returns Y2K (2000-01-01) to fetch all data.
// This is deterministic and works regardless of when the sync runs.
// Note: We use Y2K instead of Unix epoch because some APIs don't handle
// pre-2000 dates correctly.
func (m *Manager) getSyncStartTime() time.Time {
	logging.Debug().
		Bool("sync_all", m.cfg.Sync.SyncAll).
		Dur("lookback", m.cfg.Sync.Lookback).
		Msg("getSyncStartTime: checking config")

	if m.cfg.Sync.SyncAll {
		logging.Info().Msg("SYNC_ALL enabled: syncing all historical data from 2000-01-01")
		// Y2K date - far enough back to capture all data, but compatible with all APIs
		return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	since := time.Now().Add(-m.cfg.Sync.Lookback)
	logging.Debug().Time("since", since).Msg("getSyncStartTime: using lookback")
	return since
}

// syncLoop runs the periodic synchronization
func (m *Manager) syncLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.cfg.Sync.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			// Prevent concurrent sync execution
			m.syncMu.Lock()
			err := m.syncData()
			m.syncMu.Unlock()

			if err != nil {
				logging.Error().Err(err).Msg("Sync failed")
			}
		}
	}
}

// syncData synchronizes new data from Tautulli
func (m *Manager) syncData() error {
	// Note: syncMu should be locked by caller (TriggerSync or syncLoop)
	since := m.LastSyncTime()
	if since.IsZero() {
		since = m.getSyncStartTime()
	}
	return m.syncDataSince(context.Background(), since)
}

// syncDataSince synchronizes data from a specific point in time
func (m *Manager) syncDataSince(ctx context.Context, since time.Time) error {
	syncStartTime := time.Now()
	totalProcessed, err := m.fetchAndProcessBatches(ctx, since)
	if err != nil {
		return err
	}

	m.finalizeSyncOperation(ctx, syncStartTime, totalProcessed)
	return nil
}

// fetchAndProcessBatches fetches and processes all batches from Tautulli API
func (m *Manager) fetchAndProcessBatches(ctx context.Context, since time.Time) (int, error) {
	start := 0
	totalProcessed := 0

	for {
		history, shouldContinue, err := m.fetchHistoryBatch(ctx, since, start)
		if err != nil {
			return totalProcessed, err
		}
		if !shouldContinue {
			break
		}

		processed := m.processBatchWithMetrics(ctx, history.Response.Data.Data)
		totalProcessed += processed

		logging.Info().
			Int("processed", processed).
			Int("batch_size", len(history.Response.Data.Data)).
			Int("total", totalProcessed).
			Msg("Processed batch")

		if len(history.Response.Data.Data) < m.cfg.Sync.BatchSize {
			break
		}

		start += m.cfg.Sync.BatchSize
	}

	return totalProcessed, nil
}

// fetchHistoryBatch fetches a single batch from Tautulli with retry logic
func (m *Manager) fetchHistoryBatch(ctx context.Context, since time.Time, start int) (*tautulli.TautulliHistory, bool, error) {
	var history *tautulli.TautulliHistory
	var err error

	err = m.retryWithBackoff(ctx, func() error {
		history, err = m.client.GetHistorySince(ctx, since, start, m.cfg.Sync.BatchSize)
		return err
	})

	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch history (start=%d): %w", start, err)
	}

	if len(history.Response.Data.Data) == 0 {
		return nil, false, nil
	}

	return history, true, nil
}

// processBatchWithMetrics processes a batch and records metrics
func (m *Manager) processBatchWithMetrics(ctx context.Context, records []tautulli.TautulliHistoryRecord) int {
	metrics.SyncBatchSize.Observe(float64(len(records)))

	processed, err := m.processBatch(ctx, records)
	if err != nil {
		logging.Warn().Err(err).Msg("Batch processing had errors, but continuing")
		metrics.SyncErrors.WithLabelValues("database").Inc()
	}

	return processed
}

// finalizeSyncOperation completes the sync operation with metrics, callbacks, and logging
func (m *Manager) finalizeSyncOperation(ctx context.Context, syncStartTime time.Time, totalProcessed int) {
	m.mu.Lock()
	m.lastSync = syncStartTime
	callback := m.onSyncCompleted
	m.mu.Unlock()

	// Flush pending events to database before reporting completion
	m.flushPublisherWithVerification(ctx, totalProcessed)

	// Calculate sync duration and record metrics
	syncDuration := time.Since(syncStartTime)
	durationMs := syncDuration.Milliseconds()
	metrics.RecordSyncOperation(syncDuration, totalProcessed, nil)

	// Invoke callback if set
	if callback != nil {
		callback(totalProcessed, durationMs)
	}

	// Log reconciliation summary and completion
	m.logSyncReconciliation(totalProcessed, syncDuration)
	logging.Info().Int("records", totalProcessed).Dur("duration", syncDuration).Msg("Sync completed")
}

// logSyncReconciliation logs detailed reconciliation information at trace level
func (m *Manager) logSyncReconciliation(totalProcessed int, syncDuration time.Duration) {
	logging.Info().Msg("=== SYNC RECONCILIATION SUMMARY ===")
	logging.Trace().Msg("TRACE RECONCILE: Records fetched from API: (see TRACE API logs above)")
	logging.Trace().Int("processed", totalProcessed).Msg("TRACE RECONCILE: Records processed by sync")
	logging.Trace().Dur("duration", syncDuration).Msg("TRACE RECONCILE: Sync duration")
	logging.Trace().Msg("TRACE RECONCILE: If records are missing, check:")
	logging.Trace().Msg("TRACE RECONCILE:   1. TRACE API logs - records fetched from Tautulli")
	logging.Trace().Msg("TRACE RECONCILE:   2. TRACE SYNC logs - records entering sync manager")
	logging.Trace().Msg("TRACE RECONCILE:   3. TRACE RECORD logs - individual record processing")
	logging.Trace().Msg("TRACE RECONCILE:   4. TRACE PUBLISH logs - events published to NATS")
	logging.Trace().Msg("TRACE RECONCILE:   5. TRACE HANDLER logs - events received by handler")
	logging.Trace().Msg("TRACE RECONCILE:   6. TRACE APPENDER logs - events buffered")
	logging.Trace().Msg("TRACE RECONCILE:   7. TRACE STORE logs - events inserted to database")
	logging.Info().Msg("=================================")
}

// processBatch processes a batch of history records with batch geolocation lookups
// MEDIUM-2: Batch geolocation lookups for 10-20x performance improvement
// Returns number of successfully processed records
//
//nolint:unparam // error always nil by design
func (m *Manager) processBatch(ctx context.Context, records []tautulli.TautulliHistoryRecord) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	m.logBatchStart(records)

	// Step 1: Extract and sort unique IP addresses
	ipList := m.extractUniqueIPs(records)
	metrics.GeolocationBatchSize.Observe(float64(len(ipList)))

	// Step 2: Batch fetch geolocations from database
	geoMap, err := m.db.GetGeolocations(ctx, ipList)
	if err != nil {
		return m.processBatchWithFallback(ctx, records, err)
	}

	// Step 3: Fetch missing geolocations and update map
	metrics.GeolocationCacheHits.Add(float64(len(geoMap)))
	missingCount := m.fetchMissingGeolocations(ctx, ipList, geoMap)
	metrics.GeolocationCacheMisses.Add(float64(missingCount))

	// Step 4: Process all records using cached geolocation map
	return m.processRecordsWithGeoMap(ctx, records, geoMap), nil
}

// logBatchStart logs the batch processing start with trace-level details
func (m *Manager) logBatchStart(records []tautulli.TautulliHistoryRecord) {
	logging.Trace().Int("count", len(records)).Msg("TRACE SYNC: Processing batch of records")
	for i := range records {
		record := &records[i]
		sessionKey := getEffectiveSessionKey(record)
		if i < 3 || i >= len(records)-2 {
			logging.Trace().Int("index", i+1).Int("total", len(records)).Str("session", sessionKey).Str("user", record.User).Str("media", record.MediaType).Msg("TRACE SYNC")
		} else if i == 3 {
			logging.Trace().Int("omitted", len(records)-5).Msg("TRACE SYNC: ... (records omitted) ...")
		}
	}
}

// extractUniqueIPs extracts and sorts unique IP addresses from records
// DETERMINISM: Sorting ensures deterministic processing order across all operations
func (m *Manager) extractUniqueIPs(records []tautulli.TautulliHistoryRecord) []string {
	ipSet := make(map[string]bool)
	for i := range records {
		record := &records[i]
		if record.IPAddress != "" && record.IPAddress != "N/A" {
			ipSet[record.IPAddress] = true
		}
	}

	ipList := make([]string, 0, len(ipSet))
	for ip := range ipSet {
		ipList = append(ipList, ip)
	}
	sort.Strings(ipList)
	return ipList
}

// processBatchWithFallback handles batch processing when geolocation lookup fails
// Falls back to individual record processing with error collection
func (m *Manager) processBatchWithFallback(ctx context.Context, records []tautulli.TautulliHistoryRecord, lookupErr error) (int, error) {
	logging.Warn().Err(lookupErr).Msg("Batch geolocation lookup failed, falling back to individual lookups")

	var fallbackErrors []error
	processed := 0

	for i := range records {
		if recordErr := m.processHistoryRecord(ctx, &records[i]); recordErr != nil {
			// Only collect non-duplicate errors (session already processed is expected)
			if recordErr.Error() != "session already processed" {
				fallbackErrors = append(fallbackErrors, fmt.Errorf("record %d (session=%s): %w",
					i, getEffectiveSessionKey(&records[i]), recordErr))
			}
		} else {
			processed++
		}
	}

	if len(fallbackErrors) > 0 {
		logging.Warn().Int("errors", len(fallbackErrors)).Int("total_records", len(records)).Msg("Fallback processing completed with errors")
		return processed, errors.Join(fallbackErrors...)
	}
	return processed, nil
}

// fetchMissingGeolocations identifies and fetches geolocations not in the cache
// Returns the count of missing IPs that were fetched
func (m *Manager) fetchMissingGeolocations(ctx context.Context, ipList []string, geoMap map[string]*models.Geolocation) int {
	missingCount := 0
	for _, ip := range ipList {
		if _, exists := geoMap[ip]; exists {
			continue
		}
		missingCount++

		startTime := time.Now()
		geo := m.fetchOrCreateFallbackGeolocation(ctx, ip)
		metrics.GeolocationAPICallDuration.Observe(time.Since(startTime).Seconds())

		geoMap[ip] = geo
	}
	return missingCount
}

// fetchOrCreateFallbackGeolocation attempts to fetch geolocation, creating a fallback on failure
func (m *Manager) fetchOrCreateFallbackGeolocation(ctx context.Context, ip string) *models.Geolocation {
	geo, err := m.fetchAndCacheGeolocation(ctx, ip)
	if err != nil {
		logging.Warn().Err(err).Str("ip", ip).Msg("Failed to fetch geolocation - using unknown location")
		geo = m.createUnknownGeolocation(ip)
		if cacheErr := m.db.UpsertGeolocation(geo); cacheErr != nil {
			logging.Warn().Err(cacheErr).Str("ip", ip).Msg("Failed to cache unknown geolocation")
		}
	}
	return geo
}

// createUnknownGeolocation creates a fallback geolocation entry for unknown IPs
func (m *Manager) createUnknownGeolocation(ip string) *models.Geolocation {
	return &models.Geolocation{
		IPAddress:   ip,
		Latitude:    0,
		Longitude:   0,
		Country:     "Unknown",
		LastUpdated: time.Now(),
	}
}

// processRecordsWithGeoMap processes all records using a pre-fetched geolocation map
// Returns the count of successfully processed records
func (m *Manager) processRecordsWithGeoMap(ctx context.Context, records []tautulli.TautulliHistoryRecord, geoMap map[string]*models.Geolocation) int {
	processed := 0
	for i := range records {
		record := &records[i]
		if err := m.processHistoryRecordWithGeo(ctx, record, geoMap); err != nil {
			logging.Error().Err(err).Str("session", getEffectiveSessionKey(record)).Msg("Failed to process record")
			continue
		}
		processed++
	}
	return processed
}

// processHistoryRecordWithGeo processes a single record using pre-fetched geolocation map
// MEDIUM-2: Optimized version that skips individual geolocation lookups
//
// Data Flow depends on NATS_EVENT_SOURCING config:
//   - Event Sourcing Mode (true): Publish to NATS only, DuckDBConsumer writes to DB
//   - Notification Mode (false): Write to DB first, then publish to NATS for notifications
func (m *Manager) processHistoryRecordWithGeo(ctx context.Context, record *tautulli.TautulliHistoryRecord, geoMap map[string]*models.Geolocation) error {
	sessionKey := getEffectiveSessionKey(record)
	logging.Trace().Str("session", sessionKey).Str("user", record.User).Str("ip", record.IPAddress).Str("media", record.MediaType).Msg("TRACE RECORD: Processing")

	eventSourcingMode := m.cfg.NATS.Enabled && m.cfg.NATS.EventSourcing

	// Validate record based on mode
	if err := m.validateRecordForMode(ctx, record, eventSourcingMode); err != nil {
		return err
	}

	// Resolve and validate geolocation
	if _, err := m.resolveGeolocationFromMap(ctx, record, geoMap); err != nil {
		return err
	}

	// Build enriched playback event
	event := m.buildEnrichedEvent(record)

	// Persist event based on mode
	return m.persistEventForMode(ctx, event, eventSourcingMode)
}

// validateRecordForMode validates a record based on event sourcing or notification mode
func (m *Manager) validateRecordForMode(ctx context.Context, record *tautulli.TautulliHistoryRecord, eventSourcingMode bool) error {
	if !eventSourcingMode {
		return m.validateRecord(ctx, record)
	}

	// In event sourcing mode, only validate IP address
	if record.IPAddress == "" || record.IPAddress == "N/A" {
		return fmt.Errorf("invalid IP address for session %s", getEffectiveSessionKey(record))
	}
	return nil
}

// resolveGeolocationFromMap retrieves geolocation from map or fetches individually
func (m *Manager) resolveGeolocationFromMap(ctx context.Context, record *tautulli.TautulliHistoryRecord, geoMap map[string]*models.Geolocation) (*models.Geolocation, error) {
	geo, exists := geoMap[record.IPAddress]
	if !exists {
		logging.Warn().Str("ip", record.IPAddress).Msg("IP not in geolocation map, fetching individually")
		var err error
		geo, err = m.resolveGeolocation(ctx, record)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve geolocation: %w", err)
		}
	}

	if geo.Country == "" {
		return nil, fmt.Errorf("invalid geolocation: missing country for IP %s", record.IPAddress)
	}

	return geo, nil
}

// buildEnrichedEvent builds a playback event with all enrichments applied
func (m *Manager) buildEnrichedEvent(record *tautulli.TautulliHistoryRecord) *models.PlaybackEvent {
	event := m.buildCoreEvent(record)
	m.enrichEventWithMetadata(event, record)
	m.enrichEventWithQualityData(event, record)
	m.enrichEventWithExtendedData(event, record)
	return event
}

// persistEventForMode persists the event using event sourcing or notification mode
func (m *Manager) persistEventForMode(ctx context.Context, event *models.PlaybackEvent, eventSourcingMode bool) error {
	if eventSourcingMode {
		return m.persistEventWithEventSourcing(ctx, event)
	}
	return m.persistEventWithNotificationMode(ctx, event)
}

// persistEventWithEventSourcing persists event using NATS with DB fallback
func (m *Manager) persistEventWithEventSourcing(ctx context.Context, event *models.PlaybackEvent) error {
	// Event Sourcing Mode (v2.3): Use synchronous publish with automatic DB fallback
	// CRITICAL: publishEventWithFallback ensures ZERO data loss by:
	//   1. Attempting synchronous NATS publish
	//   2. On failure, immediately falling back to direct DB insert
	//   3. Generating correlation key before publish for consistent dedup
	if err := m.publishEventWithFallback(ctx, event); err != nil {
		return fmt.Errorf("failed to persist playback event: %w", err)
	}
	return nil
}

// persistEventWithNotificationMode persists event to DB first, then publishes notification
func (m *Manager) persistEventWithNotificationMode(ctx context.Context, event *models.PlaybackEvent) error {
	// Generate correlation key for deduplication
	if event.CorrelationKey == nil {
		correlationKey := generatePlaybackEventCorrelationKey(event)
		event.CorrelationKey = &correlationKey
	}

	m.logDedupKeyValues(event)

	if err := m.db.InsertPlaybackEvent(event); err != nil {
		return fmt.Errorf("failed to insert playback event: %w", err)
	}

	// Publish to NATS if enabled (v1.47: event-driven architecture)
	m.publishEvent(ctx, event)
	return nil
}

// logDedupKeyValues logs deduplication key values at trace level
func (m *Manager) logDedupKeyValues(event *models.PlaybackEvent) {
	if logging.GetLevel() > zerolog.TraceLevel {
		return
	}

	ratingKeyVal := "<nil>"
	if event.RatingKey != nil {
		ratingKeyVal = *event.RatingKey
	}
	logging.Trace().
		Str("session", event.SessionKey).
		Str("correlation", *event.CorrelationKey).
		Int("user_id", event.UserID).
		Str("rating_key", ratingKeyVal).
		Str("started_at", event.StartedAt.Format("2006-01-02T15:04:05")).
		Msg("dedup key values")
}

// validateRecord checks if a history record should be processed
// Returns true if the record is valid and should be processed
func (m *Manager) validateRecord(ctx context.Context, record *tautulli.TautulliHistoryRecord) error {
	// Use effective session key (with RowID fallback for NULL session_key from Tautulli API)
	sessionKey := getEffectiveSessionKey(record)

	// Check if session already exists
	exists, err := m.db.SessionKeyExists(ctx, sessionKey)
	if err != nil {
		return fmt.Errorf("failed to check if session exists: %w", err)
	}

	if exists {
		return fmt.Errorf("session already processed") // Special error type to skip gracefully
	}

	// Validate IP address
	if record.IPAddress == "" || record.IPAddress == "N/A" {
		return fmt.Errorf("invalid IP address for session %s", sessionKey)
	}

	return nil
}

// processHistoryRecord processes a single history record
// Refactored to use helper methods for clarity
//
// Data Flow depends on NATS_EVENT_SOURCING config:
//   - Event Sourcing Mode (true): Publish to NATS only, DuckDBConsumer writes to DB
//   - Notification Mode (false): Write to DB first, then publish to NATS for notifications
func (m *Manager) processHistoryRecord(ctx context.Context, record *tautulli.TautulliHistoryRecord) error {
	eventSourcingMode := m.cfg.NATS.Enabled && m.cfg.NATS.EventSourcing

	// Validate record based on mode
	if err := m.validateAndHandleRecord(ctx, record, eventSourcingMode); err != nil {
		if errors.Is(err, errSkipRecord) {
			return nil // Gracefully skip this record
		}
		return err
	}

	// Resolve geolocation (errors logged but don't fail sync)
	//nolint:errcheck // geolocation errors are logged but don't fail sync
	m.resolveGeolocation(ctx, record)

	// Build event with all field mappings
	event := m.buildEventWithAllFields(record)

	// Persist event based on mode
	return m.persistEventForMode(ctx, event, eventSourcingMode)
}

// validateAndHandleRecord validates a record and handles special cases like duplicates
func (m *Manager) validateAndHandleRecord(ctx context.Context, record *tautulli.TautulliHistoryRecord, eventSourcingMode bool) error {
	if err := m.validateRecordForMode(ctx, record, eventSourcingMode); err != nil {
		// In notification mode, gracefully skip already-processed sessions
		if !eventSourcingMode && err.Error() == "session already processed" {
			return errSkipRecord
		}
		return err
	}
	return nil
}

// buildEventWithAllFields builds a PlaybackEvent with all field mappings applied
func (m *Manager) buildEventWithAllFields(record *tautulli.TautulliHistoryRecord) *models.PlaybackEvent {
	event := m.buildCoreEvent(record)

	// Map optional fields using helper methods (groups related fields)
	m.mapStreamingQualityFields(record, event)
	m.mapLibraryFields(record, event)
	m.mapStreamDetailsFields(record, event)
	m.mapAudioFields(record, event)
	m.mapVideoFields(record, event)
	m.mapContainerSubtitleFields(record, event)
	m.mapConnectionFields(record, event)
	m.mapFileMetadataFields(record, event)
	m.mapEnrichmentFields(record, event)

	return event
}
