// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package tautulliimport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/eventprocessor"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// EventPublisher defines the interface for publishing events to NATS.
type EventPublisher interface {
	PublishEvent(ctx context.Context, event *eventprocessor.MediaEvent) error
}

// ProgressTracker defines the interface for tracking import progress.
type ProgressTracker interface {
	// Save persists the current import progress.
	Save(ctx context.Context, stats *ImportStats) error

	// Load retrieves the last saved import progress.
	Load(ctx context.Context) (*ImportStats, error)

	// Clear removes saved progress (for fresh imports).
	Clear(ctx context.Context) error
}

// Importer handles importing Tautulli database files.
type Importer struct {
	cfg       *config.ImportConfig
	publisher EventPublisher
	progress  ProgressTracker
	mapper    *Mapper

	// State
	mu       sync.RWMutex
	running  bool
	stats    *ImportStats
	stopChan chan struct{}
}

// NewImporter creates a new Tautulli database importer.
func NewImporter(cfg *config.ImportConfig, publisher EventPublisher, progress ProgressTracker) *Importer {
	return &Importer{
		cfg:       cfg,
		publisher: publisher,
		progress:  progress,
		mapper:    NewMapper(),
		stopChan:  make(chan struct{}),
	}
}

// Import performs the import operation.
// It reads records from the Tautulli SQLite database, converts them to
// PlaybackEvents, and publishes them to NATS JetStream.
func (i *Importer) Import(ctx context.Context) (*ImportStats, error) {
	i.mu.Lock()
	if i.running {
		i.mu.Unlock()
		return nil, fmt.Errorf("import already in progress")
	}
	i.running = true
	i.stats = &ImportStats{
		StartTime: time.Now(),
		DryRun:    i.cfg.DryRun,
	}
	i.mu.Unlock()

	defer func() {
		i.mu.Lock()
		i.running = false
		i.stats.EndTime = time.Now()
		i.mu.Unlock()
	}()

	// Open SQLite reader
	reader, err := NewSQLiteReader(i.cfg.DBPath)
	if err != nil {
		return i.GetStats(), fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			logging.Warn().Err(closeErr).Msg("Error closing SQLite reader")
		}
	}()

	// Get total record count
	total, err := reader.CountRecords(ctx)
	if err != nil {
		return i.GetStats(), fmt.Errorf("count records: %w", err)
	}

	i.mu.Lock()
	i.stats.TotalRecords = total
	i.mu.Unlock()

	logging.Info().Int64("total_records", total).Msg("Starting import")

	// Log database statistics
	if err := i.logDatabaseStats(ctx, reader); err != nil {
		logging.Warn().Err(err).Msg("Failed to get database stats")
	}

	// Determine starting point
	startID := i.cfg.ResumeFromID
	if startID == 0 && i.progress != nil {
		// Try to load previous progress
		if prevStats, err := i.progress.Load(ctx); err == nil && prevStats != nil {
			startID = prevStats.LastProcessedID
			logging.Info().Int64("start_id", startID).Msg("Resuming import from record ID")
		}
	}

	// Count remaining records
	remaining, err := reader.CountRecordsSince(ctx, startID)
	if err != nil {
		return i.GetStats(), fmt.Errorf("count remaining records: %w", err)
	}

	logging.Info().Int64("records_to_process", remaining).Int64("start_id", startID).Msg("Records to process")

	// Process all batches
	if err := i.processAllBatches(ctx, reader, startID); err != nil {
		return i.GetStats(), err
	}

	logging.Info().
		Int64("imported", i.stats.Imported).
		Int64("skipped", i.stats.Skipped).
		Int64("errors", i.stats.Errors).
		Dur("duration", i.stats.Duration()).
		Msg("Import completed")

	return i.GetStats(), nil
}

// processAllBatches processes all batches starting from the given ID.
func (i *Importer) processAllBatches(ctx context.Context, reader *SQLiteReader, startID int64) error {
	currentID := startID
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-i.stopChan:
			return fmt.Errorf("import canceled")
		default:
		}

		// Read batch
		records, err := reader.ReadBatch(ctx, currentID, i.cfg.BatchSize)
		if err != nil {
			return fmt.Errorf("read batch: %w", err)
		}

		if len(records) == 0 {
			return nil // Done
		}

		// Process batch and update stats
		currentID = i.processBatchAndUpdateStats(ctx, records)
	}
}

// processBatchAndUpdateStats processes a batch and updates statistics.
// Returns the last processed ID for the next iteration.
func (i *Importer) processBatchAndUpdateStats(ctx context.Context, records []TautulliRecord) int64 {
	imported, skipped, errors := i.processBatch(ctx, records)

	// Update stats
	i.mu.Lock()
	i.stats.Processed += int64(len(records))
	i.stats.Imported += int64(imported)
	i.stats.Skipped += int64(skipped)
	i.stats.Errors += int64(errors)
	lastID := records[len(records)-1].ID
	i.stats.LastProcessedID = lastID
	stats := *i.stats
	i.mu.Unlock()

	// Save progress
	if i.progress != nil && !i.cfg.DryRun {
		if err := i.progress.Save(ctx, &stats); err != nil {
			logging.Warn().Err(err).Msg("Failed to save progress")
		}
	}

	// Log progress
	logging.Info().
		Float64("progress_percent", stats.Progress()).
		Int64("processed", stats.Processed).
		Int64("total_records", stats.TotalRecords).
		Int64("imported", stats.Imported).
		Int64("skipped", stats.Skipped).
		Int64("errors", stats.Errors).
		Float64("records_per_second", stats.RecordsPerSecond()).
		Msg("Import progress")

	return lastID
}

// processBatch processes a batch of records.
// Returns counts of imported, skipped, and error records.
func (i *Importer) processBatch(ctx context.Context, records []TautulliRecord) (imported, skipped, errors int) {
	// Filter valid records
	validRecords, skipCount := i.mapper.FilterValidRecords(records)
	skipped = skipCount

	// Convert to PlaybackEvents
	events := i.mapper.ToPlaybackEvents(validRecords)

	// Publish to NATS (unless dry run)
	for _, event := range events {
		if i.cfg.DryRun {
			imported++
			continue
		}

		// Convert to MediaEvent for NATS publishing
		mediaEvent := playbackEventToMediaEvent(event)

		if err := i.publisher.PublishEvent(ctx, mediaEvent); err != nil {
			logging.Error().Err(err).Str("event_id", event.ID.String()).Msg("Failed to publish event")
			errors++
		} else {
			imported++
		}
	}

	return imported, skipped, errors
}

// logDatabaseStats logs statistics about the source database.
func (i *Importer) logDatabaseStats(ctx context.Context, reader *SQLiteReader) error {
	// Get date range
	earliest, latest, err := reader.GetDateRange(ctx)
	if err != nil {
		return err
	}
	logging.Info().
		Str("earliest", earliest.Format("2006-01-02")).
		Str("latest", latest.Format("2006-01-02")).
		Msg("Import date range")

	// Get user count
	userCount, err := reader.GetUserStats(ctx)
	if err != nil {
		return err
	}
	logging.Info().Int("unique_users", userCount).Msg("Unique users in database")

	// Get media type breakdown
	mediaTypes, err := reader.GetMediaTypeStats(ctx)
	if err != nil {
		return err
	}
	for mediaType, count := range mediaTypes {
		logging.Info().Str("media_type", mediaType).Int("count", count).Msg("Media type statistics")
	}

	return nil
}

// Stop cancels a running import operation.
func (i *Importer) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.running {
		return fmt.Errorf("no import in progress")
	}

	close(i.stopChan)
	i.stopChan = make(chan struct{}) // Reset for next import

	return nil
}

// GetStats returns the current import statistics.
func (i *Importer) GetStats() *ImportStats {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.stats == nil {
		return &ImportStats{}
	}

	// Return a copy
	stats := *i.stats
	return &stats
}

// IsRunning returns whether an import is currently in progress.
func (i *Importer) IsRunning() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.running
}

// playbackEventToMediaEvent converts a PlaybackEvent to a MediaEvent for NATS publishing.
func playbackEventToMediaEvent(pe *models.PlaybackEvent) *eventprocessor.MediaEvent {
	me := &eventprocessor.MediaEvent{
		EventID:    pe.ID.String(),
		SessionKey: pe.SessionKey,
		Source:     pe.Source,
		UserID:     pe.UserID,
		Username:   pe.Username,
		MediaType:  pe.MediaType,
		Title:      pe.Title,
		Platform:   pe.Platform,
		Player:     pe.Player,
		StartedAt:  pe.StartedAt,
		IPAddress:  pe.IPAddress,

		PercentComplete: pe.PercentComplete,
		PausedCounter:   pe.PausedCounter,
	}

	// Optional fields
	if pe.CorrelationKey != nil {
		me.CorrelationKey = *pe.CorrelationKey
	}
	if pe.FriendlyName != nil {
		me.FriendlyName = *pe.FriendlyName
	}
	if pe.MachineID != nil {
		me.MachineID = *pe.MachineID
	}
	if pe.StoppedAt != nil {
		me.StoppedAt = pe.StoppedAt
	}
	if pe.ParentTitle != nil {
		me.ParentTitle = *pe.ParentTitle
	}
	if pe.GrandparentTitle != nil {
		me.GrandparentTitle = *pe.GrandparentTitle
	}
	if pe.RatingKey != nil {
		me.RatingKey = *pe.RatingKey
	}
	// Note: ParentRatingKey, GrandparentRatingKey, MediaIndex, and ParentMediaIndex
	// are not included in MediaEvent but are stored in PlaybackEvent for binge detection.
	// These fields are handled by the DuckDBConsumer when writing to the database.
	if pe.TranscodeDecision != nil {
		me.TranscodeDecision = *pe.TranscodeDecision
	}
	if pe.VideoResolution != nil {
		me.VideoResolution = *pe.VideoResolution
	}
	if pe.AudioCodec != nil {
		me.AudioCodec = *pe.AudioCodec
	}
	if pe.StreamBitrate != nil {
		me.StreamBitrate = *pe.StreamBitrate
	}

	return me
}
