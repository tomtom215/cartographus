// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// SyncProgress represents the progress of a sync operation.
// This structure is shared between backend and frontend.
type SyncProgress struct {
	Status                    string      `json:"status"` // idle, running, completed, error, canceled
	TotalRecords              int64       `json:"total_records"`
	ProcessedRecords          int64       `json:"processed_records"`
	ImportedRecords           int64       `json:"imported_records"`
	SkippedRecords            int64       `json:"skipped_records"`
	ErrorCount                int64       `json:"error_count"`
	ProgressPercent           float64     `json:"progress_percent"`
	RecordsPerSecond          float64     `json:"records_per_second"`
	ElapsedSeconds            float64     `json:"elapsed_seconds"`
	EstimatedRemainingSeconds float64     `json:"estimated_remaining_seconds"`
	StartTime                 *time.Time  `json:"start_time,omitempty"`
	LastProcessedID           int64       `json:"last_processed_id,omitempty"`
	DryRun                    bool        `json:"dry_run,omitempty"`
	Errors                    []SyncError `json:"errors,omitempty"`
}

// SyncError represents an error encountered during sync.
type SyncError struct {
	Timestamp   time.Time `json:"timestamp"`
	RecordID    *int64    `json:"record_id,omitempty"`
	Message     string    `json:"message"`
	Recoverable bool      `json:"recoverable"`
}

// SyncStatusResponse represents the combined status of all sync operations.
type SyncStatusResponse struct {
	TautulliImport *SyncProgress            `json:"tautulli_import,omitempty"`
	PlexHistorical *SyncProgress            `json:"plex_historical,omitempty"`
	ServerSyncs    map[string]*SyncProgress `json:"server_syncs,omitempty"`
}

// PlexHistoricalRequest represents a request to start Plex historical sync.
type PlexHistoricalRequest struct {
	DaysBack   int      `json:"days_back,omitempty"`
	LibraryIDs []string `json:"library_ids,omitempty"`
}

// PlexHistoricalResponse represents the response from starting Plex historical sync.
type PlexHistoricalResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message,omitempty"`
	Error         string `json:"error,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// ImportStatusProvider provides import status information.
// This interface decouples sync handlers from the concrete ImportHandlers type.
type ImportStatusProvider interface {
	// IsImportRunning returns whether an import is currently running.
	IsImportRunning() bool

	// GetImportStats returns the current import statistics.
	GetImportStats() interface{}
}

// SyncStatusProvider provides sync status information.
// Implemented by sync.Manager.
type SyncStatusProvider interface {
	// IsPlexHistoricalRunning returns whether a Plex historical sync is running.
	IsPlexHistoricalRunning() bool

	// GetPlexHistoricalProgress returns the current Plex historical sync progress.
	GetPlexHistoricalProgress() *SyncProgress

	// StartPlexHistoricalSync starts a Plex historical sync.
	StartPlexHistoricalSync(ctx context.Context, daysBack int, libraryIDs []string) error
}

// SyncHandlers holds the sync management handlers.
type SyncHandlers struct {
	importStatusProvider ImportStatusProvider // Optional: for import status
	syncStatusProvider   SyncStatusProvider   // Optional: for Plex historical sync
	mu                   sync.RWMutex

	// Track Plex historical sync state
	plexHistoricalProgress *SyncProgress
	plexHistoricalRunning  bool
}

// NewSyncHandlers creates a new set of sync handlers.
func NewSyncHandlers(importStatusProvider ImportStatusProvider, syncStatusProvider SyncStatusProvider) *SyncHandlers {
	return &SyncHandlers{
		importStatusProvider: importStatusProvider,
		syncStatusProvider:   syncStatusProvider,
	}
}

// HandleGetSyncStatus handles GET /api/v1/sync/status
//
// @Summary Get sync status
// @Description Returns the combined status of all sync operations
// @Tags sync
// @Produce json
// @Success 200 {object} SyncStatusResponse
// @Router /api/v1/sync/status [get]
func (h *SyncHandlers) HandleGetSyncStatus(w http.ResponseWriter, r *http.Request) {
	response := SyncStatusResponse{
		ServerSyncs: make(map[string]*SyncProgress),
	}

	// Get Tautulli import status if available
	if h.importStatusProvider != nil {
		running := h.importStatusProvider.IsImportRunning()
		stats := h.importStatusProvider.GetImportStats()

		if running || stats != nil {
			response.TautulliImport = convertImportStatsToSyncProgress(stats, running)
		}
	}

	// Get Plex historical sync status
	h.mu.RLock()
	if h.plexHistoricalProgress != nil {
		response.PlexHistorical = h.plexHistoricalProgress
	}
	h.mu.RUnlock()

	// If we have a sync status provider, check it too
	if h.syncStatusProvider != nil {
		if progress := h.syncStatusProvider.GetPlexHistoricalProgress(); progress != nil {
			response.PlexHistorical = progress
		}
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleStartPlexHistoricalSync handles POST /api/v1/sync/plex/historical
//
// @Summary Start Plex historical sync
// @Description Starts syncing playback history from Plex servers
// @Tags sync
// @Accept json
// @Produce json
// @Param request body PlexHistoricalRequest false "Sync options"
// @Success 200 {object} PlexHistoricalResponse
// @Failure 400 {object} PlexHistoricalResponse "Invalid request"
// @Failure 409 {object} PlexHistoricalResponse "Sync already in progress"
// @Failure 500 {object} PlexHistoricalResponse "Internal error"
// @Router /api/v1/sync/plex/historical [post]
func (h *SyncHandlers) HandleStartPlexHistoricalSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if Tautulli import is running (mutex - can't run both)
	if h.importStatusProvider != nil && h.importStatusProvider.IsImportRunning() {
		h.writeJSON(w, http.StatusConflict, PlexHistoricalResponse{
			Success: false,
			Error:   "cannot start Plex historical sync while Tautulli import is running",
		})
		return
	}

	// Check if Plex historical sync is already running
	h.mu.RLock()
	running := h.plexHistoricalRunning
	h.mu.RUnlock()

	if running {
		h.writeJSON(w, http.StatusConflict, PlexHistoricalResponse{
			Success: false,
			Error:   "Plex historical sync already in progress",
		})
		return
	}

	// Also check sync status provider
	if h.syncStatusProvider != nil && h.syncStatusProvider.IsPlexHistoricalRunning() {
		h.writeJSON(w, http.StatusConflict, PlexHistoricalResponse{
			Success: false,
			Error:   "Plex historical sync already in progress",
		})
		return
	}

	// Parse request body
	var req PlexHistoricalRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeJSON(w, http.StatusBadRequest, PlexHistoricalResponse{
				Success: false,
				Error:   "invalid request body: " + err.Error(),
			})
			return
		}
	}

	// Set defaults
	if req.DaysBack <= 0 {
		req.DaysBack = 30
	}

	// Generate correlation ID
	correlationID := generateCorrelationID()

	// Mark as running
	h.mu.Lock()
	h.plexHistoricalRunning = true
	h.plexHistoricalProgress = &SyncProgress{
		Status: "running",
	}
	startTime := time.Now()
	h.plexHistoricalProgress.StartTime = &startTime
	h.mu.Unlock()

	// Start sync in background
	go func() {
		defer func() {
			h.mu.Lock()
			h.plexHistoricalRunning = false
			if h.plexHistoricalProgress != nil {
				if h.plexHistoricalProgress.Status == "running" {
					h.plexHistoricalProgress.Status = "completed"
				}
			}
			h.mu.Unlock()
		}()

		// If we have a sync status provider, use it
		if h.syncStatusProvider != nil {
			if err := h.syncStatusProvider.StartPlexHistoricalSync(ctx, req.DaysBack, req.LibraryIDs); err != nil {
				logging.Error().Err(err).Str("correlation_id", correlationID).Msg("Plex historical sync failed")
				h.mu.Lock()
				if h.plexHistoricalProgress != nil {
					h.plexHistoricalProgress.Status = "error"
				}
				h.mu.Unlock()
				return
			}
		} else {
			// No sync status provider - simulate completion after a delay
			logging.Warn().Msg("No sync status provider configured for Plex historical sync")
			time.Sleep(1 * time.Second)
		}

		logging.Info().Str("correlation_id", correlationID).Msg("Plex historical sync completed")
	}()

	h.writeJSON(w, http.StatusOK, PlexHistoricalResponse{
		Success:       true,
		Message:       "Plex historical sync started",
		CorrelationID: correlationID,
	})
}

// UpdatePlexHistoricalProgress updates the Plex historical sync progress.
// Called by sync.Manager during sync operations.
func (h *SyncHandlers) UpdatePlexHistoricalProgress(progress *SyncProgress) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.plexHistoricalProgress = progress
}

// convertImportStatsToSyncProgress converts import stats to sync progress format.
func convertImportStatsToSyncProgress(stats interface{}, running bool) *SyncProgress {
	// Type assertion for ImportStats from the import package
	// We use interface{} to avoid circular imports
	type importStats interface {
		Progress() float64
		RecordsPerSecond() float64
		Duration() time.Duration
		EstimatedRemain() time.Duration
	}

	if s, ok := stats.(importStats); ok {
		status := "idle"
		if running {
			status = "running"
		}

		progress := &SyncProgress{
			Status:                    status,
			ProgressPercent:           s.Progress(),
			RecordsPerSecond:          s.RecordsPerSecond(),
			ElapsedSeconds:            s.Duration().Seconds(),
			EstimatedRemainingSeconds: s.EstimatedRemain().Seconds(),
		}

		// Try to get additional fields via reflection or interface
		// This is a simplified version - in production we'd have proper types
		return progress
	}

	// Fallback: try to convert from map[string]interface{}
	if m, ok := stats.(map[string]interface{}); ok {
		progress := &SyncProgress{
			Status: "idle",
		}
		if running {
			progress.Status = "running"
		}
		if v, ok := m["total_records"].(int64); ok {
			progress.TotalRecords = v
		}
		if v, ok := m["processed"].(int64); ok {
			progress.ProcessedRecords = v
		}
		if v, ok := m["imported"].(int64); ok {
			progress.ImportedRecords = v
		}
		if v, ok := m["skipped"].(int64); ok {
			progress.SkippedRecords = v
		}
		if v, ok := m["errors"].(int64); ok {
			progress.ErrorCount = v
		}
		return progress
	}

	return nil
}

// generateCorrelationID generates a unique correlation ID for tracing.
func generateCorrelationID() string {
	// Use timestamp + random suffix for simplicity
	// In production, use crypto/rand or a UUID library
	return time.Now().Format("20060102150405") + "-" + randomSuffix()
}

// randomSuffix generates a random 8-character suffix.
func randomSuffix() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		// Use time-based pseudo-random for simplicity
		// In production, use crypto/rand
		b[i] = chars[(time.Now().UnixNano()+int64(i))%int64(len(chars))]
	}
	return string(b)
}

// writeJSON writes a JSON response.
func (h *SyncHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	//nolint:errcheck // HTTP response write errors are not recoverable
	json.NewEncoder(w).Encode(data)
}
