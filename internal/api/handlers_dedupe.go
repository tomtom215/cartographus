// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// dedupeMetadata creates a standard Metadata struct with current timestamp and query time.
func dedupeMetadata(queryStartTime time.Time) models.Metadata {
	return models.Metadata{
		Timestamp:   time.Now(),
		QueryTimeMS: time.Since(queryStartTime).Milliseconds(),
	}
}

// =============================================================================
// Dedupe Audit API Handlers (v2.2 - ADR-0022)
// =============================================================================

// DedupeAuditListResponse is the response for listing dedupe audit entries.
type DedupeAuditListResponse struct {
	Entries    []*models.DedupeAuditEntry `json:"entries"`
	TotalCount int64                      `json:"total_count"`
	Limit      int                        `json:"limit"`
	Offset     int                        `json:"offset"`
}

// DedupeAuditActionRequest is the request body for confirm/restore actions.
type DedupeAuditActionRequest struct {
	ResolvedBy string `json:"resolved_by"`
	Notes      string `json:"notes,omitempty"`
}

// DedupeAuditRestoreResponse is the response after restoring an event.
type DedupeAuditRestoreResponse struct {
	Success       bool                     `json:"success"`
	Message       string                   `json:"message"`
	RestoredEvent *uuid.UUID               `json:"restored_event_id,omitempty"`
	OriginalEntry *models.DedupeAuditEntry `json:"original_entry,omitempty"`
}

// DedupeAuditList handles GET /api/v1/dedupe/audit
// Lists dedupe audit entries with optional filtering.
//
// Query parameters:
//   - user_id: Filter by user ID
//   - source: Filter by source (tautulli, plex, jellyfin, emby)
//   - status: Filter by status (auto_dedupe, user_confirmed, user_restored)
//   - reason: Filter by dedupe reason
//   - layer: Filter by dedupe layer
//   - from: Filter from timestamp (RFC3339)
//   - to: Filter to timestamp (RFC3339)
//   - limit: Number of results (default 100, max 1000)
//   - offset: Pagination offset
func (h *Handler) DedupeAuditList(w http.ResponseWriter, r *http.Request) {
	queryStart := time.Now()
	ctx := r.Context()

	filter := database.DedupeAuditFilter{}

	// Parse query parameters
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			http.Error(w, "Invalid user_id parameter", http.StatusBadRequest)
			return
		}
		filter.UserID = &userID
	}

	filter.Source = r.URL.Query().Get("source")
	filter.Status = r.URL.Query().Get("status")
	filter.DedupeReason = r.URL.Query().Get("reason")
	filter.DedupeLayer = r.URL.Query().Get("layer")

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			http.Error(w, "Invalid 'from' timestamp (use RFC3339 format)", http.StatusBadRequest)
			return
		}
		filter.FromTime = &t
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			http.Error(w, "Invalid 'to' timestamp (use RFC3339 format)", http.StatusBadRequest)
			return
		}
		filter.ToTime = &t
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			http.Error(w, "Invalid limit (1-1000)", http.StatusBadRequest)
			return
		}
		filter.Limit = limit
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			http.Error(w, "Invalid offset", http.StatusBadRequest)
			return
		}
		filter.Offset = offset
	}

	entries, totalCount, err := h.db.ListDedupeAuditEntries(ctx, filter)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to list dedupe audit entries")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list dedupe audit entries", err)
		return
	}

	// Apply defaults for response
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	response := DedupeAuditListResponse{
		Entries:    entries,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     filter.Offset,
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status:   "success",
		Data:     response,
		Metadata: dedupeMetadata(queryStart),
	})
}

// DedupeAuditGet handles GET /api/v1/dedupe/audit/{id}
// Returns a specific dedupe audit entry by ID.
func (h *Handler) DedupeAuditGet(w http.ResponseWriter, r *http.Request) {
	queryStart := time.Now()
	ctx := r.Context()

	idStr := r.PathValue("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Missing entry ID", nil)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid entry ID format", err)
		return
	}

	entry, err := h.db.GetDedupeAuditEntry(ctx, id)
	if err != nil {
		logging.Error().Err(err).Str("id", id.String()).Msg("Failed to get dedupe audit entry")
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Dedupe audit entry not found", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status:   "success",
		Data:     entry,
		Metadata: dedupeMetadata(queryStart),
	})
}

// DedupeAuditStats handles GET /api/v1/dedupe/audit/stats
// Returns aggregate statistics for the dedupe audit dashboard.
func (h *Handler) DedupeAuditStats(w http.ResponseWriter, r *http.Request) {
	queryStart := time.Now()
	ctx := r.Context()

	stats, err := h.db.GetDedupeAuditStats(ctx)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to get dedupe audit stats")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get dedupe statistics", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status:   "success",
		Data:     stats,
		Metadata: dedupeMetadata(queryStart),
	})
}

// DedupeAuditConfirm handles POST /api/v1/dedupe/audit/{id}/confirm
// Marks a dedupe decision as correct (confirmed by user).
func (h *Handler) DedupeAuditConfirm(w http.ResponseWriter, r *http.Request) {
	queryStart := time.Now()
	ctx := r.Context()

	idStr := r.PathValue("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Missing entry ID", nil)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid entry ID format", err)
		return
	}

	var req DedupeAuditActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body - use defaults
		req = DedupeAuditActionRequest{}
	}

	if req.ResolvedBy == "" {
		req.ResolvedBy = "api_user"
	}

	if err := h.db.UpdateDedupeAuditStatus(ctx, id, "user_confirmed", req.ResolvedBy, req.Notes); err != nil {
		logging.Error().Err(err).Str("id", id.String()).Msg("Failed to confirm dedupe audit entry")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to confirm dedupe entry", err)
		return
	}

	// Return updated entry
	entry, err := h.db.GetDedupeAuditEntry(ctx, id)
	if err != nil {
		logging.Error().Err(err).Str("id", id.String()).Msg("Failed to get updated entry")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Entry confirmed but failed to retrieve updated record", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status:   "success",
		Data:     entry,
		Metadata: dedupeMetadata(queryStart),
	})
}

// DedupeAuditRestore handles POST /api/v1/dedupe/audit/{id}/restore
// Restores a deduplicated event (inserts it back into playback_events).
func (h *Handler) DedupeAuditRestore(w http.ResponseWriter, r *http.Request) {
	queryStart := time.Now()
	ctx := r.Context()

	idStr := r.PathValue("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Missing entry ID", nil)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid entry ID format", err)
		return
	}

	var req DedupeAuditActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body - use defaults
		req = DedupeAuditActionRequest{}
	}

	if req.ResolvedBy == "" {
		req.ResolvedBy = "api_user"
	}

	// Get the audit entry to retrieve the raw payload
	entry, err := h.db.GetDedupeAuditEntry(ctx, id)
	if err != nil {
		logging.Error().Err(err).Str("id", id.String()).Msg("Failed to get dedupe audit entry")
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Dedupe audit entry not found", err)
		return
	}

	if entry.Status == "user_restored" {
		respondError(w, http.StatusConflict, "CONFLICT", "Event has already been restored", nil)
		return
	}

	if len(entry.DiscardedRawPayload) == 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "No raw payload available for restoration", nil)
		return
	}

	// Unmarshal the raw payload into a PlaybackEvent
	var event models.PlaybackEvent
	if err := json.Unmarshal(entry.DiscardedRawPayload, &event); err != nil {
		logging.Error().Err(err).Str("id", id.String()).Msg("Failed to unmarshal raw payload")
		respondError(w, http.StatusInternalServerError, "PARSE_ERROR", "Failed to parse stored event data", err)
		return
	}

	// Generate new unique ID and clear correlation key to avoid re-deduplication
	event.ID = uuid.New()
	event.CorrelationKey = nil

	// Add metadata about restoration
	restoredFromAudit := id.String()
	event.CorrelationKey = &restoredFromAudit

	// Insert the restored event
	if err := h.db.InsertPlaybackEvent(&event); err != nil {
		logging.Error().Err(err).Str("id", id.String()).Msg("Failed to insert restored event")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to restore event", err)
		return
	}

	// Update the audit entry status
	if err := h.db.UpdateDedupeAuditStatus(ctx, id, "user_restored", req.ResolvedBy, req.Notes); err != nil {
		logging.Warn().Err(err).Str("id", id.String()).Msg("Failed to update audit status after restore")
		// Don't return error - event was restored successfully
	}

	// Get updated entry for response
	updatedEntry, err := h.db.GetDedupeAuditEntry(ctx, id)
	if err != nil {
		logging.Warn().Err(err).Str("id", id.String()).Msg("Failed to get updated audit entry")
		// Don't fail - event was restored successfully, just log the error
	}

	response := DedupeAuditRestoreResponse{
		Success:       true,
		Message:       "Event successfully restored",
		RestoredEvent: &event.ID,
		OriginalEntry: updatedEntry,
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status:   "success",
		Data:     response,
		Metadata: dedupeMetadata(queryStart),
	})
}

// DedupeAuditExport handles GET /api/v1/dedupe/audit/export
// Exports the dedupe audit log to CSV format.
func (h *Handler) DedupeAuditExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse the same filter parameters as list
	filter := database.DedupeAuditFilter{}

	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err == nil {
			filter.UserID = &userID
		}
	}

	filter.Source = r.URL.Query().Get("source")
	filter.Status = r.URL.Query().Get("status")
	filter.DedupeReason = r.URL.Query().Get("reason")
	filter.DedupeLayer = r.URL.Query().Get("layer")

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.FromTime = &t
		}
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.ToTime = &t
		}
	}

	// Get all matching entries (up to 10000 for export)
	filter.Limit = 10000
	filter.Offset = 0

	entries, _, err := h.db.ListDedupeAuditEntries(ctx, filter)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to list dedupe audit entries for export")
		http.Error(w, "Failed to export dedupe audit log", http.StatusInternalServerError)
		return
	}

	// Set CSV headers
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=dedupe_audit_log.csv")

	// Write CSV header
	csvHeader := "id,timestamp,discarded_event_id,discarded_source,matched_event_id,matched_source,dedupe_reason,dedupe_layer,user_id,username,media_type,title,status,resolved_by,resolved_at,resolution_notes\n"
	if _, err := w.Write([]byte(csvHeader)); err != nil {
		logging.Error().Err(err).Msg("Failed to write CSV header")
		return
	}

	// Write each entry
	for _, entry := range entries {
		resolvedAt := ""
		if entry.ResolvedAt != nil {
			resolvedAt = entry.ResolvedAt.Format(time.RFC3339)
		}

		// Escape fields that might contain commas/quotes
		line := escapeCSV(entry.ID.String()) + "," +
			escapeCSV(entry.Timestamp.Format(time.RFC3339)) + "," +
			escapeCSV(entry.DiscardedEventID) + "," +
			escapeCSV(entry.DiscardedSource) + "," +
			escapeCSV(entry.MatchedEventID) + "," +
			escapeCSV(entry.MatchedSource) + "," +
			escapeCSV(entry.DedupeReason) + "," +
			escapeCSV(entry.DedupeLayer) + "," +
			strconv.Itoa(entry.UserID) + "," +
			escapeCSV(entry.Username) + "," +
			escapeCSV(entry.MediaType) + "," +
			escapeCSV(entry.Title) + "," +
			escapeCSV(entry.Status) + "," +
			escapeCSV(entry.ResolvedBy) + "," +
			escapeCSV(resolvedAt) + "," +
			escapeCSV(entry.ResolutionNotes) + "\n"

		if _, err := w.Write([]byte(line)); err != nil {
			logging.Error().Err(err).Msg("Failed to write CSV line")
			return
		}
	}
}

// Note: escapeCSV is defined in handlers_helpers.go and shared across handlers
