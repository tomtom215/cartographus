// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// ========================================
// Plex API Integration (v1.37)
// Hybrid Plex + Tautulli Data Architecture
// ========================================

// syncPlexHistorical performs one-time backfill of all Plex playback history
//
// This method fetches historical playback data from Plex Media Server for the
// configured time range (PLEX_SYNC_DAYS_BACK). It's designed to be run ONCE
// during initial setup to backfill history from BEFORE Tautulli was deployed.
//
// Workflow:
//  1. Test Plex connectivity (fail fast if unreachable)
//  2. Fetch ALL history from Plex (sorted oldest first for chronological insertion)
//  3. Filter to configured date range (last N days)
//  4. Process in batches of 1000 records
//  5. Deduplicate against existing Tautulli data (automatic via UNIQUE constraint)
//  6. Insert new events with source='plex'
//
// Deduplication:
// - Uses (rating_key, user_id, started_at) UNIQUE constraint
// - Tautulli data always wins (inserted first, Plex can't overwrite)
// - Skips duplicates automatically without error
//
// Performance:
// - Large libraries (10k+ events): 10-30 minutes initial sync
// - Batch size: 1000 records per transaction
// - Memory efficient: processes one batch at a time
func (m *Manager) syncPlexHistorical(ctx context.Context) error {
	if m.plexClient == nil {
		return fmt.Errorf("plex client not initialized")
	}

	// Test connectivity first (fail fast)
	logging.Info().Msg("Testing Plex server connectivity...")
	if err := m.plexClient.Ping(ctx); err != nil {
		return fmt.Errorf("plex connectivity check failed: %w", err)
	}
	logging.Info().Msg("Plex server is reachable")

	// Fetch all history (sorted oldest first for chronological insertion)
	logging.Info().Msg("Fetching Plex history (last  days)...")
	history, err := m.plexClient.GetHistoryAll(ctx, "viewedAt", nil)
	if err != nil {
		return fmt.Errorf("fetch history: %w", err)
	}
	logging.Info().Msg("Retrieved  Plex history records")

	// Filter by date range (last N days)
	cutoffTime := time.Now().AddDate(0, 0, -m.cfg.Plex.SyncDaysBack).Unix()
	var filteredHistory []PlexMetadata
	for i := range history {
		if history[i].ViewedAt >= cutoffTime {
			filteredHistory = append(filteredHistory, history[i])
		}
	}
	logging.Info().Msg("Processing  records within  day window")

	// Process in batches
	batchSize := 1000
	inserted := 0
	skipped := 0

	for i := 0; i < len(filteredHistory); i += batchSize {
		end := i + batchSize
		if end > len(filteredHistory) {
			end = len(filteredHistory)
		}

		batch := filteredHistory[i:end]

		for j := range batch {
			// Convert Plex metadata to PlaybackEvent
			event := m.convertPlexToPlaybackEvent(&batch[j])

			// Insert with automatic deduplication via UNIQUE constraint
			// Database will silently skip duplicates (Tautulli events already exist)
			if err := m.db.InsertPlaybackEvent(event); err != nil {
				// Check if it's a duplicate (UNIQUE constraint violation)
				// DuckDB returns "Constraint Error" for duplicates
				if strings.Contains(err.Error(), "Constraint") || strings.Contains(err.Error(), "UNIQUE") {
					skipped++
					continue
				}

				// Real error - log and continue
				logging.Error().Err(err).Msg("Insert Plex event")
				continue
			}

			// Publish to NATS if enabled (v1.47: event-driven architecture)
			m.publishEvent(ctx, event)

			inserted++
		}

		if (i/batchSize)%10 == 0 { // Log every 10 batches
			logging.Info().Int("processed", end).Int("total", len(filteredHistory)).Int("inserted", inserted).Int("skipped", skipped).Msg("Progress")
		}
	}

	logging.Info().Msg("Plex historical sync complete: inserted  new events, skipped  duplicates")
	return nil
}

// syncPlexRecent checks for recent playback events Tautulli might have missed
//
// This method runs periodically (configured by PLEX_SYNC_INTERVAL) to catch
// events that Tautulli might have missed due to:
// - Tautulli server downtime
// - API rate limiting
// - Network issues
//
// Workflow:
//  1. Fetch events from last sync interval (e.g., last 24 hours)
//  2. Process newest events first (sorted descending)
//  3. Stop when encountering events older than sync interval
//  4. Deduplicate automatically (same UNIQUE constraint as historical sync)
//
// Performance:
// - Typically processes <100 events per run (only recent activity)
// - Fast execution: <1 second for most deployments
// - Minimal database overhead (deduplication is index-based)
func (m *Manager) syncPlexRecent(ctx context.Context) error {
	if m.plexClient == nil {
		return fmt.Errorf("plex client not initialized")
	}

	// Fetch events from last sync interval (newest first)
	history, err := m.plexClient.GetHistoryAll(ctx, "-viewedAt", nil) // Descending order
	if err != nil {
		return fmt.Errorf("fetch recent history: %w", err)
	}

	// Only process events from last sync interval
	cutoffTime := time.Now().Add(-m.cfg.Plex.SyncInterval).Unix()

	inserted := 0
	for i := range history {
		// Stop when we reach old events (records are sorted newest first)
		if history[i].ViewedAt < cutoffTime {
			break
		}

		event := m.convertPlexToPlaybackEvent(&history[i])

		// Insert with automatic deduplication
		if err := m.db.InsertPlaybackEvent(event); err != nil {
			// Skip duplicates silently
			if strings.Contains(err.Error(), "Constraint") || strings.Contains(err.Error(), "UNIQUE") {
				continue
			}

			logging.Error().Err(err).Msg("Insert Plex event")
			continue
		}

		// Publish to NATS if enabled (v1.47: event-driven architecture)
		m.publishEvent(ctx, event)

		inserted++
	}

	if inserted > 0 {
		logging.Info().Msg("Plex sync: inserted  new events missed by Tautulli")
	}

	return nil
}

// runPlexSyncLoop periodically checks for missed events (periodic sync mode)
//
// This goroutine runs when PLEX_HISTORICAL_SYNC=false and implements the
// periodic sync strategy to catch events Tautulli missed.
//
// Runs at interval configured by PLEX_SYNC_INTERVAL (default: 24 hours)
func (m *Manager) runPlexSyncLoop(ctx context.Context) {
	defer m.wg.Done()

	m.plexSyncTicker = time.NewTicker(m.cfg.Plex.SyncInterval)
	defer m.plexSyncTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			logging.Info().Msg("Plex sync loop stopping (context canceled)")
			return
		case <-m.stopChan:
			logging.Info().Msg("Plex sync loop stopping (stop signal received)")
			return
		case <-m.plexSyncTicker.C:
			logging.Info().Msg("Starting periodic Plex sync check...")
			if err := m.syncPlexRecent(ctx); err != nil {
				logging.Error().Err(err).Msg("Plex sync")
			}
		}
	}
}
