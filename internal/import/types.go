// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulliimport

import (
	"time"
)

// ImportStats holds statistics about an import operation.
type ImportStats struct {
	// TotalRecords is the total number of records in the source database.
	TotalRecords int64

	// Processed is the number of records processed (including skipped).
	Processed int64

	// Imported is the number of records successfully published to NATS.
	Imported int64

	// Skipped is the number of records skipped due to validation failures.
	Skipped int64

	// Errors is the number of records that failed to import.
	Errors int64

	// StartTime is when the import started.
	StartTime time.Time

	// EndTime is when the import completed (zero if still running).
	EndTime time.Time

	// LastProcessedID is the ID of the last successfully processed record.
	LastProcessedID int64

	// DryRun indicates if this was a dry run (no actual imports).
	DryRun bool
}

// Duration returns the duration of the import operation.
func (s *ImportStats) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// Progress returns the import progress as a percentage (0-100).
func (s *ImportStats) Progress() float64 {
	if s.TotalRecords == 0 {
		return 0
	}
	return float64(s.Processed) / float64(s.TotalRecords) * 100
}

// RecordsPerSecond returns the import rate.
func (s *ImportStats) RecordsPerSecond() float64 {
	duration := s.Duration().Seconds()
	if duration == 0 {
		return 0
	}
	return float64(s.Processed) / duration
}

// ProgressSummary provides a human-readable summary of import progress.
type ProgressSummary struct {
	Status          string    `json:"status"`
	Progress        float64   `json:"progress"`
	TotalRecords    int64     `json:"total_records"`
	Processed       int64     `json:"processed"`
	Imported        int64     `json:"imported"`
	Skipped         int64     `json:"skipped"`
	Errors          int64     `json:"errors"`
	RecordsPerSec   float64   `json:"records_per_second"`
	ElapsedSeconds  float64   `json:"elapsed_seconds"`
	EstimatedRemain float64   `json:"estimated_remaining_seconds"`
	StartTime       time.Time `json:"start_time"`
	LastProcessedID int64     `json:"last_processed_id"`
	DryRun          bool      `json:"dry_run"`
}

// ToSummary converts ImportStats to a ProgressSummary with calculated fields.
func (s *ImportStats) ToSummary(running bool) *ProgressSummary {
	summary := &ProgressSummary{
		Progress:        s.Progress(),
		TotalRecords:    s.TotalRecords,
		Processed:       s.Processed,
		Imported:        s.Imported,
		Skipped:         s.Skipped,
		Errors:          s.Errors,
		RecordsPerSec:   s.RecordsPerSecond(),
		ElapsedSeconds:  s.Duration().Seconds(),
		StartTime:       s.StartTime,
		LastProcessedID: s.LastProcessedID,
		DryRun:          s.DryRun,
	}

	// Set status
	if running {
		summary.Status = "running"
	} else if s.EndTime.IsZero() {
		summary.Status = "pending"
	} else {
		summary.Status = "completed"
	}

	// Estimate remaining time
	if running && summary.RecordsPerSec > 0 {
		remaining := s.TotalRecords - s.Processed
		summary.EstimatedRemain = float64(remaining) / summary.RecordsPerSec
	}

	return summary
}
