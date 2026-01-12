// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package wal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/logging"
)

// Publisher is the interface for publishing WAL entries.
// Implementations should unmarshal the Entry.Payload and publish to their target.
type Publisher interface {
	// PublishEntry publishes a WAL entry. The implementation should unmarshal
	// Entry.Payload to the appropriate type and publish it.
	PublishEntry(ctx context.Context, entry *Entry) error
}

// PublisherFunc is a function type that implements Publisher.
// This allows using closures as publishers for flexibility.
type PublisherFunc func(ctx context.Context, entry *Entry) error

// PublishEntry implements Publisher.
func (f PublisherFunc) PublishEntry(ctx context.Context, entry *Entry) error {
	return f(ctx, entry)
}

// RecoveryResult contains the results of a recovery operation.
type RecoveryResult struct {
	// TotalPending is the number of pending entries found.
	TotalPending int

	// Recovered is the number of entries successfully published.
	Recovered int

	// Failed is the number of entries that failed to publish.
	Failed int

	// Expired is the number of entries that were too old and removed.
	Expired int

	// Skipped is the number of entries skipped because they were already
	// being processed by another goroutine (RACE CONDITION FIX v2.4).
	Skipped int

	// Errors contains any errors encountered during recovery.
	Errors []error

	// Duration is how long the recovery took.
	Duration time.Duration
}

// RecoverPending processes all pending WAL entries on startup.
// This is called during application initialization to ensure no events are lost
// from a previous run that may have crashed or been interrupted.
//
// The recovery process:
// 1. Gets all pending entries from the WAL
// 2. For each entry:
//   - If expired (older than EntryTTL), delete it
//   - If max retries exceeded, log and delete it
//   - Otherwise, attempt to publish
//   - If publish succeeds, confirm the entry
//   - If publish fails, update the attempt count
//
// Recovery is designed to be idempotent - calling it multiple times is safe.
func (w *BadgerWAL) RecoverPending(ctx context.Context, publisher Publisher) (*RecoveryResult, error) {
	if publisher == nil {
		return nil, fmt.Errorf("publisher cannot be nil")
	}

	start := time.Now()
	result := &RecoveryResult{}

	entries, err := w.GetPending(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pending entries: %w", err)
	}

	result.TotalPending = len(entries)
	if result.TotalPending == 0 {
		logging.Info().Msg("WAL recovery: no pending entries found")
		result.Duration = time.Since(start)
		return result, nil
	}

	logging.Info().Int("pending_entries", result.TotalPending).Msg("WAL recovery found pending entries")
	RecordWALRecoveredEntries(int64(result.TotalPending))

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, ctx.Err())
			result.Duration = time.Since(start)
			return result, ctx.Err()
		default:
		}

		// DURABLE LEASING: Try to claim exclusive processing rights using BadgerDB
		// Generate a unique lease holder for this recovery operation
		leaseHolder := fmt.Sprintf("recovery-%s", uuid.New().String()[:8])
		claimed, err := w.TryClaimEntryDurable(ctx, entry.ID, leaseHolder)
		if err != nil {
			logging.Error().Err(err).Str("entry_id", entry.ID).Msg("WAL recovery: error claiming entry")
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("claim entry %s: %w", entry.ID, err))
			continue
		}
		if !claimed {
			// Another processor has the lease, skip this entry
			result.Skipped++
			continue
		}

		// Process entry with durable lease held
		w.processRecoveryEntry(ctx, entry, publisher, result)
	}

	result.Duration = time.Since(start)

	logging.Info().
		Int("recovered", result.Recovered).
		Int("failed", result.Failed).
		Int("expired", result.Expired).
		Int("skipped", result.Skipped).
		Dur("duration", result.Duration).
		Msg("WAL recovery complete")

	return result, nil
}

// processRecoveryEntry processes a single entry during recovery.
//
// DURABLE LEASING: Caller must hold the durable lease via TryClaimEntryDurable.
// The lease is automatically released when the entry is Confirmed or DeleteEntry is called.
// If processing fails, the lease naturally expires allowing retry.
func (w *BadgerWAL) processRecoveryEntry(ctx context.Context, entry *Entry, publisher Publisher, result *RecoveryResult) {
	// Note: No defer release needed - Confirm/DeleteEntry removes entry, or lease expires naturally

	// Check if entry is expired
	if time.Since(entry.CreatedAt) > w.config.EntryTTL {
		logging.Info().
			Str("entry_id", entry.ID).
			Dur("age", time.Since(entry.CreatedAt)).
			Msg("WAL recovery: entry expired, removing")
		if err := w.DeleteEntry(ctx, entry.ID); err != nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("delete expired entry %s: %w", entry.ID, err))
		}
		result.Expired++
		RecordWALExpiredEntry()
		return
	}

	// Check if max retries exceeded
	if entry.Attempts >= w.config.MaxRetries {
		logging.Info().
			Str("entry_id", entry.ID).
			Int("attempts", entry.Attempts).
			Msg("WAL recovery: entry exceeded max retries, removing")
		if err := w.DeleteEntry(ctx, entry.ID); err != nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("delete max-retried entry %s: %w", entry.ID, err))
		}
		result.Failed++
		RecordWALMaxRetriesExceeded()
		return
	}

	// Attempt to publish
	if err := publisher.PublishEntry(ctx, entry); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("WAL recovery: failed to publish entry")
		if updateErr := w.UpdateAttempt(ctx, entry.ID, err.Error()); updateErr != nil {
			// RACE CONDITION FIX: ErrEntryNotFound means another goroutine already processed it
			if errors.Is(updateErr, ErrEntryNotFound) {
				logging.Debug().Str("entry_id", entry.ID).Msg("WAL recovery: entry already processed by another goroutine")
			} else {
				result.Errors = append(result.Errors,
					fmt.Errorf("update attempt for %s: %w", entry.ID, updateErr))
			}
		}
		result.Failed++
		RecordWALNATSPublishFailure()
		return
	}

	// Confirm successful publish
	if err := w.Confirm(ctx, entry.ID); err != nil {
		// RACE CONDITION FIX: ErrEntryNotFound means another goroutine already confirmed it
		if errors.Is(err, ErrEntryNotFound) {
			logging.Debug().Str("entry_id", entry.ID).Msg("WAL recovery: entry already confirmed by another goroutine")
		} else {
			logging.Error().Err(err).Str("entry_id", entry.ID).Msg("WAL recovery: failed to confirm entry")
			result.Errors = append(result.Errors,
				fmt.Errorf("confirm entry %s: %w", entry.ID, err))
			result.Failed++
		}
		return
	}

	result.Recovered++
}

// RecoverPendingStream processes pending WAL entries using streaming for memory efficiency.
// This is preferred for large WALs where loading all entries into memory would be prohibitive.
// See RecoverPending for the standard recovery method.
func (w *BadgerWAL) RecoverPendingStream(ctx context.Context, publisher Publisher) (*RecoveryResult, error) {
	if publisher == nil {
		return nil, fmt.Errorf("publisher cannot be nil")
	}

	start := time.Now()
	result := &RecoveryResult{}

	entryCh, errCh := w.GetPendingStream(ctx)

	for entry := range entryCh {
		result.TotalPending++

		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, ctx.Err())
			result.Duration = time.Since(start)
			return result, ctx.Err()
		default:
		}

		// DURABLE LEASING: Try to claim exclusive processing rights using BadgerDB
		leaseHolder := fmt.Sprintf("recovery-stream-%s", uuid.New().String()[:8])
		claimed, err := w.TryClaimEntryDurable(ctx, entry.ID, leaseHolder)
		if err != nil {
			logging.Error().Err(err).Str("entry_id", entry.ID).Msg("WAL recovery: error claiming entry")
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("claim entry %s: %w", entry.ID, err))
			continue
		}
		if !claimed {
			// Another processor has the lease, skip this entry
			result.Skipped++
			continue
		}

		// Process entry with durable lease held
		w.processRecoveryEntry(ctx, entry, publisher, result)
	}

	// Check for stream errors
	if err := <-errCh; err != nil {
		result.Errors = append(result.Errors, err)
	}

	result.Duration = time.Since(start)

	if result.TotalPending > 0 {
		RecordWALRecoveredEntries(int64(result.TotalPending))
		logging.Info().
			Int("recovered", result.Recovered).
			Int("failed", result.Failed).
			Int("expired", result.Expired).
			Int("skipped", result.Skipped).
			Dur("duration", result.Duration).
			Msg("WAL recovery (stream) complete")
	} else {
		logging.Info().Msg("WAL recovery: no pending entries found")
	}

	return result, nil
}
