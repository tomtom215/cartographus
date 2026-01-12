// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package api

import (
	"context"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	tautulli_import "github.com/tomtom215/cartographus/internal/import"
)

// ImportController defines the interface for managing Tautulli imports.
// This interface is implemented by the import service.
type ImportController interface {
	// Import starts an import operation.
	Import(ctx context.Context) (*tautulli_import.ImportStats, error)

	// GetStats returns current import statistics.
	GetStats() *tautulli_import.ImportStats

	// IsRunning returns whether an import is in progress.
	IsRunning() bool

	// Stop cancels a running import.
	Stop() error
}

// ProgressController defines the interface for import progress tracking.
type ProgressController interface {
	// Load retrieves saved progress.
	Load(ctx context.Context) (*tautulli_import.ImportStats, error)

	// Clear removes saved progress.
	Clear(ctx context.Context) error
}

// ImportHandlers holds the import management handlers.
type ImportHandlers struct {
	importer ImportController
	progress ProgressController
}

// NewImportHandlers creates a new set of import handlers.
func NewImportHandlers(importer ImportController, progress ProgressController) *ImportHandlers {
	return &ImportHandlers{
		importer: importer,
		progress: progress,
	}
}

// ImportRequest represents a request to start an import.
type ImportRequest struct {
	// DBPath overrides the configured database path (optional).
	DBPath string `json:"db_path,omitempty"`

	// Resume continues from the last saved progress.
	Resume bool `json:"resume,omitempty"`

	// DryRun validates without actually importing.
	DryRun bool `json:"dry_run,omitempty"`
}

// ImportResponse represents the response from import operations.
type ImportResponse struct {
	Success bool                             `json:"success"`
	Message string                           `json:"message,omitempty"`
	Error   string                           `json:"error,omitempty"`
	Stats   *tautulli_import.ProgressSummary `json:"stats,omitempty"`
}

// HandleStartImport handles POST /api/v1/import/tautulli
//
// @Summary Start Tautulli database import
// @Description Starts importing playback history from a Tautulli SQLite database
// @Tags import
// @Accept json
// @Produce json
// @Param request body ImportRequest false "Import options"
// @Success 200 {object} ImportResponse
// @Failure 400 {object} ImportResponse "Invalid request"
// @Failure 409 {object} ImportResponse "Import already in progress"
// @Failure 500 {object} ImportResponse "Internal error"
// @Router /api/v1/import/tautulli [post]
func (h *ImportHandlers) HandleStartImport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if import is already running
	if h.importer.IsRunning() {
		stats := h.importer.GetStats()
		h.writeJSON(w, http.StatusConflict, ImportResponse{
			Success: false,
			Error:   "import already in progress",
			Stats:   stats.ToSummary(true),
		})
		return
	}

	// Parse optional request body
	var req ImportRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeJSON(w, http.StatusBadRequest, ImportResponse{
				Success: false,
				Error:   "invalid request body: " + err.Error(),
			})
			return
		}
	}

	// Clear previous progress if not resuming
	if !req.Resume && h.progress != nil {
		if err := h.progress.Clear(ctx); err != nil {
			h.writeJSON(w, http.StatusInternalServerError, ImportResponse{
				Success: false,
				Error:   "failed to clear previous progress: " + err.Error(),
			})
			return
		}
	}

	// Start import in background with timeout
	// Use a detached context with timeout to allow import to complete even if HTTP request is canceled,
	// but prevent indefinite hangs and respect server shutdown with a reasonable timeout
	importCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	go func() {
		defer cancel() // Clean up context resources
		//nolint:errcheck // Background import - errors are logged by the importer
		h.importer.Import(importCtx)
	}()

	h.writeJSON(w, http.StatusOK, ImportResponse{
		Success: true,
		Message: "import started",
		Stats:   h.importer.GetStats().ToSummary(true),
	})
}

// HandleGetImportStatus handles GET /api/v1/import/status
//
// @Summary Get import status
// @Description Returns the current status of a Tautulli database import
// @Tags import
// @Produce json
// @Success 200 {object} ImportResponse
// @Router /api/v1/import/status [get]
func (h *ImportHandlers) HandleGetImportStatus(w http.ResponseWriter, r *http.Request) {
	running := h.importer.IsRunning()
	stats := h.importer.GetStats()

	// If not running and no current stats, try to load from progress tracker
	if !running && stats.TotalRecords == 0 && h.progress != nil {
		if saved, err := h.progress.Load(r.Context()); err == nil && saved != nil {
			stats = saved
		}
	}

	h.writeJSON(w, http.StatusOK, ImportResponse{
		Success: true,
		Stats:   stats.ToSummary(running),
	})
}

// HandleStopImport handles DELETE /api/v1/import
//
// @Summary Stop import
// @Description Stops a running Tautulli database import
// @Tags import
// @Produce json
// @Success 200 {object} ImportResponse
// @Failure 400 {object} ImportResponse "No import in progress"
// @Router /api/v1/import [delete]
func (h *ImportHandlers) HandleStopImport(w http.ResponseWriter, r *http.Request) {
	if !h.importer.IsRunning() {
		h.writeJSON(w, http.StatusBadRequest, ImportResponse{
			Success: false,
			Error:   "no import in progress",
		})
		return
	}

	if err := h.importer.Stop(); err != nil {
		h.writeJSON(w, http.StatusInternalServerError, ImportResponse{
			Success: false,
			Error:   "failed to stop import: " + err.Error(),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, ImportResponse{
		Success: true,
		Message: "import stop requested",
		Stats:   h.importer.GetStats().ToSummary(false),
	})
}

// HandleClearProgress handles DELETE /api/v1/import/progress
//
// @Summary Clear import progress
// @Description Clears saved import progress to allow a fresh import
// @Tags import
// @Produce json
// @Success 200 {object} ImportResponse
// @Failure 400 {object} ImportResponse "Import in progress"
// @Failure 500 {object} ImportResponse "Internal error"
// @Router /api/v1/import/progress [delete]
func (h *ImportHandlers) HandleClearProgress(w http.ResponseWriter, r *http.Request) {
	if h.importer.IsRunning() {
		h.writeJSON(w, http.StatusBadRequest, ImportResponse{
			Success: false,
			Error:   "cannot clear progress while import is running",
		})
		return
	}

	if h.progress == nil {
		h.writeJSON(w, http.StatusOK, ImportResponse{
			Success: true,
			Message: "progress tracking not enabled",
		})
		return
	}

	if err := h.progress.Clear(r.Context()); err != nil {
		h.writeJSON(w, http.StatusInternalServerError, ImportResponse{
			Success: false,
			Error:   "failed to clear progress: " + err.Error(),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, ImportResponse{
		Success: true,
		Message: "progress cleared",
	})
}

// HandleValidateDatabase handles POST /api/v1/import/validate
//
// @Summary Validate Tautulli database
// @Description Validates a Tautulli database file without importing
// @Tags import
// @Accept json
// @Produce json
// @Param request body ValidateRequest true "Database path"
// @Success 200 {object} ValidateResponse
// @Failure 400 {object} ValidateResponse "Invalid request"
// @Failure 500 {object} ValidateResponse "Validation failed"
// @Router /api/v1/import/validate [post]
func (h *ImportHandlers) HandleValidateDatabase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DBPath string `json:"db_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid request body: " + err.Error(),
		})
		return
	}

	if req.DBPath == "" {
		h.writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "db_path is required",
		})
		return
	}

	// Try to open and validate the database
	reader, err := tautulli_import.NewSQLiteReader(req.DBPath)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "failed to open database: " + err.Error(),
		})
		return
	}
	defer reader.Close() //nolint:errcheck // best-effort cleanup

	ctx := r.Context()

	// Get statistics
	totalRecords, err := reader.CountRecords(ctx)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "failed to count records: " + err.Error(),
		})
		return
	}

	earliest, latest, err := reader.GetDateRange(ctx)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "failed to get date range: " + err.Error(),
		})
		return
	}

	userCount, _ := reader.GetUserStats(ctx)       //nolint:errcheck // optional stats, defaults to 0
	mediaTypes, _ := reader.GetMediaTypeStats(ctx) //nolint:errcheck // optional stats, defaults to nil

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"total_records": totalRecords,
		"date_range": map[string]string{
			"earliest": earliest.Format("2006-01-02"),
			"latest":   latest.Format("2006-01-02"),
		},
		"unique_users": userCount,
		"media_types":  mediaTypes,
	})
}

// writeJSON writes a JSON response.
func (h *ImportHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	//nolint:errcheck // HTTP response write errors are not recoverable
	json.NewEncoder(w).Encode(data)
}
