// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
)

// DLQEntry represents a failed message in the Dead Letter Queue.
// This is a simplified view of the internal DLQ entry for API responses.
type DLQEntry struct {
	EventID       string    `json:"event_id"`
	MessageID     string    `json:"message_id"`
	Source        string    `json:"source"`
	Username      string    `json:"username,omitempty"`
	MediaTitle    string    `json:"media_title,omitempty"`
	OriginalError string    `json:"original_error"`
	LastError     string    `json:"last_error"`
	RetryCount    int       `json:"retry_count"`
	MaxRetries    int       `json:"max_retries"`
	FirstFailure  time.Time `json:"first_failure"`
	LastFailure   time.Time `json:"last_failure"`
	NextRetry     time.Time `json:"next_retry"`
	Category      string    `json:"category"`
	Status        string    `json:"status"` // "pending", "retrying", "permanent"
}

// DLQStats represents DLQ statistics.
type DLQStats struct {
	TotalEntries      int64            `json:"total_entries"`
	TotalAdded        int64            `json:"total_added"`
	TotalRemoved      int64            `json:"total_removed"`
	TotalRetries      int64            `json:"total_retries"`
	TotalExpired      int64            `json:"total_expired"`
	OldestEntryAge    *int64           `json:"oldest_entry_age_seconds,omitempty"`
	NewestEntryAge    *int64           `json:"newest_entry_age_seconds,omitempty"`
	EntriesByCategory map[string]int64 `json:"entries_by_category"`
	EntriesByStatus   map[string]int64 `json:"entries_by_status"`
}

// DLQEntriesResponse is the response for listing DLQ entries.
type DLQEntriesResponse struct {
	Entries []DLQEntry `json:"entries"`
	Total   int        `json:"total"`
	Limit   int        `json:"limit"`
	Offset  int        `json:"offset"`
}

// DLQRetryResponse is the response for retry operations.
type DLQRetryResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	RetriedCount int    `json:"retried_count,omitempty"`
}

// DLQStore interface for dependency injection.
// This allows the handlers to work with the event processor's DLQ.
type DLQStore interface {
	// ListEntries returns all DLQ entries.
	ListEntries() []DLQEntryInternal

	// GetEntry retrieves an entry by event ID.
	GetEntry(eventID string) *DLQEntryInternal

	// RemoveEntry removes an entry from the DLQ.
	RemoveEntry(eventID string) bool

	// GetPendingRetries returns entries ready for retry.
	GetPendingRetries() []DLQEntryInternal

	// Stats returns DLQ statistics.
	Stats() DLQStatsInternal

	// RetryEntry triggers a retry for a specific entry.
	RetryEntry(eventID string) error

	// RetryAllPending retries all entries ready for retry.
	RetryAllPending() (int, error)

	// GetMaxRetries returns the configured max retries.
	GetMaxRetries() int

	// Cleanup removes expired entries.
	Cleanup() int
}

// DLQEntryInternal is the internal representation from eventprocessor.
type DLQEntryInternal struct {
	EventID       string
	MessageID     string
	Source        string
	Username      string
	MediaTitle    string
	OriginalError string
	LastError     string
	RetryCount    int
	FirstFailure  time.Time
	LastFailure   time.Time
	NextRetry     time.Time
	Category      string
}

// DLQStatsInternal is the internal stats representation.
type DLQStatsInternal struct {
	TotalEntries      int64
	TotalAdded        int64
	TotalRemoved      int64
	TotalRetries      int64
	TotalExpired      int64
	OldestEntry       time.Time
	NewestEntry       time.Time
	EntriesByCategory map[string]int64
}

// DLQHandlers provides HTTP handlers for DLQ endpoints.
type DLQHandlers struct {
	store      DLQStore
	maxRetries int
}

// NewDLQHandlers creates new DLQ handlers.
func NewDLQHandlers(store DLQStore, maxRetries int) *DLQHandlers {
	return &DLQHandlers{
		store:      store,
		maxRetries: maxRetries,
	}
}

// ListEntries handles GET /api/v1/dlq/entries
// Returns a paginated list of DLQ entries.
func (h *DLQHandlers) ListEntries(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get optional filters
	categoryFilter := r.URL.Query().Get("category")
	statusFilter := r.URL.Query().Get("status")

	// Get all entries
	internalEntries := h.store.ListEntries()

	// Convert and filter entries
	var entries []DLQEntry
	for i := range internalEntries {
		entry := h.convertEntry(&internalEntries[i])

		// Apply filters
		if categoryFilter != "" && entry.Category != categoryFilter {
			continue
		}
		if statusFilter != "" && entry.Status != statusFilter {
			continue
		}

		entries = append(entries, entry)
	}

	total := len(entries)

	// Apply pagination
	start := offset
	if start > len(entries) {
		start = len(entries)
	}
	end := start + limit
	if end > len(entries) {
		end = len(entries)
	}
	paginatedEntries := entries[start:end]

	response := DLQEntriesResponse{
		Entries: paginatedEntries,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}

	writeJSON(w, response)
}

// GetEntry handles GET /api/v1/dlq/entries/{id}
// Returns a single DLQ entry by event ID.
func (h *DLQHandlers) GetEntry(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Event ID is required", nil)
		return
	}

	entry := h.store.GetEntry(eventID)
	if entry == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "DLQ entry not found", nil)
		return
	}

	writeJSON(w, h.convertEntry(entry))
}

// RetryEntry handles POST /api/v1/dlq/entries/{id}/retry
// Triggers a retry for a specific DLQ entry.
func (h *DLQHandlers) RetryEntry(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Event ID is required", nil)
		return
	}

	entry := h.store.GetEntry(eventID)
	if entry == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "DLQ entry not found", nil)
		return
	}

	err := h.store.RetryEntry(eventID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "RETRY_ERROR", "Failed to retry entry", err)
		return
	}

	writeJSON(w, DLQRetryResponse{
		Success:      true,
		Message:      "Entry queued for retry",
		RetriedCount: 1,
	})
}

// DeleteEntry handles DELETE /api/v1/dlq/entries/{id}
// Removes an entry from the DLQ.
func (h *DLQHandlers) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Event ID is required", nil)
		return
	}

	if !h.store.RemoveEntry(eventID) {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "DLQ entry not found", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetStats handles GET /api/v1/dlq/stats
// Returns DLQ statistics.
func (h *DLQHandlers) GetStats(w http.ResponseWriter, _ *http.Request) {
	internalStats := h.store.Stats()

	stats := DLQStats{
		TotalEntries:      internalStats.TotalEntries,
		TotalAdded:        internalStats.TotalAdded,
		TotalRemoved:      internalStats.TotalRemoved,
		TotalRetries:      internalStats.TotalRetries,
		TotalExpired:      internalStats.TotalExpired,
		EntriesByCategory: internalStats.EntriesByCategory,
		EntriesByStatus:   make(map[string]int64),
	}

	// Calculate oldest and newest entry ages
	if !internalStats.OldestEntry.IsZero() {
		age := int64(time.Since(internalStats.OldestEntry).Seconds())
		stats.OldestEntryAge = &age
	}
	if !internalStats.NewestEntry.IsZero() {
		age := int64(time.Since(internalStats.NewestEntry).Seconds())
		stats.NewestEntryAge = &age
	}

	// Count entries by status
	entries := h.store.ListEntries()
	for i := range entries {
		status := h.getStatus(entries[i].RetryCount)
		stats.EntriesByStatus[status]++
	}

	writeJSON(w, stats)
}

// RetryAllPending handles POST /api/v1/dlq/retry-all
// Retries all entries that are ready for retry.
func (h *DLQHandlers) RetryAllPending(w http.ResponseWriter, _ *http.Request) {
	count, err := h.store.RetryAllPending()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "RETRY_ERROR", "Failed to retry entries", err)
		return
	}

	writeJSON(w, DLQRetryResponse{
		Success:      true,
		Message:      "Pending entries queued for retry",
		RetriedCount: count,
	})
}

// Cleanup handles POST /api/v1/dlq/cleanup
// Removes expired entries from the DLQ.
func (h *DLQHandlers) Cleanup(w http.ResponseWriter, _ *http.Request) {
	count := h.store.Cleanup()

	writeJSON(w, map[string]interface{}{
		"cleaned_count": count,
		"message":       "Expired entries cleaned up",
	})
}

// GetCategories handles GET /api/v1/dlq/categories
// Returns the list of available error categories.
func (h *DLQHandlers) GetCategories(w http.ResponseWriter, _ *http.Request) {
	categories := []string{
		"unknown",
		"connection",
		"timeout",
		"validation",
		"database",
		"capacity",
	}

	writeJSON(w, map[string]interface{}{"categories": categories})
}

// convertEntry converts an internal DLQ entry to the API representation.
func (h *DLQHandlers) convertEntry(internal *DLQEntryInternal) DLQEntry {
	return DLQEntry{
		EventID:       internal.EventID,
		MessageID:     internal.MessageID,
		Source:        internal.Source,
		Username:      internal.Username,
		MediaTitle:    internal.MediaTitle,
		OriginalError: internal.OriginalError,
		LastError:     internal.LastError,
		RetryCount:    internal.RetryCount,
		MaxRetries:    h.maxRetries,
		FirstFailure:  internal.FirstFailure,
		LastFailure:   internal.LastFailure,
		NextRetry:     internal.NextRetry,
		Category:      internal.Category,
		Status:        h.getStatus(internal.RetryCount),
	}
}

// getStatus determines the status based on retry count.
func (h *DLQHandlers) getStatus(retryCount int) string {
	if retryCount >= h.maxRetries {
		return "permanent"
	}
	if retryCount > 0 {
		return "retrying"
	}
	return "pending"
}

// Ensure json package is used
var _ = json.Marshal
