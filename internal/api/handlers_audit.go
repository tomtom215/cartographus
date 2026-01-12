// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/audit"
	"github.com/tomtom215/cartographus/internal/logging"
)

// AuditHandlers provides HTTP handlers for audit log endpoints.
// These endpoints expose the security audit trail to the Data Governance UI.
type AuditHandlers struct {
	logger *audit.Logger
	store  AuditStore
}

// AuditStore interface for dependency injection.
type AuditStore interface {
	Query(ctx context.Context, filter audit.QueryFilter) ([]audit.Event, error)
	Count(ctx context.Context, filter audit.QueryFilter) (int64, error)
	Get(ctx context.Context, id string) (*audit.Event, error)
	GetStats(ctx context.Context) (*audit.Stats, error)
}

// NewAuditHandlers creates new audit handlers.
func NewAuditHandlers(logger *audit.Logger, store AuditStore) *AuditHandlers {
	return &AuditHandlers{
		logger: logger,
		store:  store,
	}
}

// ListEvents handles GET /api/v1/audit/events
// Returns a paginated list of audit events with optional filtering.
func (h *AuditHandlers) ListEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := audit.DefaultQueryFilter()

	// Parse query parameters
	if v := r.URL.Query().Get("limit"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if offset, err := strconv.Atoi(v); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Parse type filter (can be multiple)
	if types := r.URL.Query()["type"]; len(types) > 0 {
		for _, t := range types {
			filter.Types = append(filter.Types, audit.EventType(t))
		}
	}

	// Parse severity filter (can be multiple)
	if severities := r.URL.Query()["severity"]; len(severities) > 0 {
		for _, s := range severities {
			filter.Severities = append(filter.Severities, audit.Severity(s))
		}
	}

	// Parse outcome filter (can be multiple)
	if outcomes := r.URL.Query()["outcome"]; len(outcomes) > 0 {
		for _, o := range outcomes {
			filter.Outcomes = append(filter.Outcomes, audit.Outcome(o))
		}
	}

	// Actor filters
	if v := r.URL.Query().Get("actor_id"); v != "" {
		filter.ActorID = v
	}
	if v := r.URL.Query().Get("actor_type"); v != "" {
		filter.ActorType = v
	}

	// Target filters
	if v := r.URL.Query().Get("target_id"); v != "" {
		filter.TargetID = v
	}
	if v := r.URL.Query().Get("target_type"); v != "" {
		filter.TargetType = v
	}

	// Source IP filter
	if v := r.URL.Query().Get("source_ip"); v != "" {
		filter.SourceIP = v
	}

	// Time range filter
	if v := r.URL.Query().Get("start_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.StartTime = &t
		}
	}
	if v := r.URL.Query().Get("end_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.EndTime = &t
		}
	}

	// Search text
	if v := r.URL.Query().Get("search"); v != "" {
		filter.SearchText = v
	}

	// Correlation and request ID
	if v := r.URL.Query().Get("correlation_id"); v != "" {
		filter.CorrelationID = v
	}
	if v := r.URL.Query().Get("request_id"); v != "" {
		filter.RequestID = v
	}

	// Ordering
	if v := r.URL.Query().Get("order_by"); v != "" {
		filter.OrderBy = v
	}
	filter.OrderDesc = r.URL.Query().Get("order_direction") != "asc"

	events, err := h.store.Query(ctx, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "AUDIT_ERROR", "Failed to fetch audit events", err)
		return
	}

	// Get total count for pagination
	count, err := h.store.Count(ctx, filter)
	if err != nil {
		logging.Warn().Err(err).Msg("Failed to get audit event count")
		count = int64(len(events))
	}

	response := map[string]interface{}{
		"events": events,
		"total":  count,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	}

	writeJSON(w, response)
}

// GetEvent handles GET /api/v1/audit/events/{id}
// Returns a single audit event by ID.
func (h *AuditHandlers) GetEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if id == "" {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Event ID is required", nil)
		return
	}

	event, err := h.store.Get(ctx, id)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found", err)
		return
	}

	writeJSON(w, event)
}

// GetStats handles GET /api/v1/audit/stats
// Returns aggregate statistics about audit events.
func (h *AuditHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.store.GetStats(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "AUDIT_ERROR", "Failed to get audit statistics", err)
		return
	}

	writeJSON(w, stats)
}

// GetTypes handles GET /api/v1/audit/types
// Returns the list of available audit event types.
func (h *AuditHandlers) GetTypes(w http.ResponseWriter, _ *http.Request) {
	types := []string{
		string(audit.EventTypeAuthSuccess),
		string(audit.EventTypeAuthFailure),
		string(audit.EventTypeAuthLockout),
		string(audit.EventTypeAuthUnlock),
		string(audit.EventTypeLogout),
		string(audit.EventTypeLogoutAll),
		string(audit.EventTypeSessionCreated),
		string(audit.EventTypeSessionExpired),
		string(audit.EventTypeTokenRevoked),
		string(audit.EventTypeAuthzGranted),
		string(audit.EventTypeAuthzDenied),
		string(audit.EventTypeDetectionAlert),
		string(audit.EventTypeDetectionAcknowledged),
		string(audit.EventTypeDetectionRuleChanged),
		string(audit.EventTypeUserCreated),
		string(audit.EventTypeUserModified),
		string(audit.EventTypeUserDeleted),
		string(audit.EventTypeRoleAssigned),
		string(audit.EventTypeRoleRevoked),
		string(audit.EventTypeConfigChanged),
		string(audit.EventTypeDataExport),
		string(audit.EventTypeDataImport),
		string(audit.EventTypeDataBackup),
		string(audit.EventTypeAdminAction),
	}

	writeJSON(w, map[string]interface{}{"types": types})
}

// GetSeverities handles GET /api/v1/audit/severities
// Returns the list of available audit severity levels.
func (h *AuditHandlers) GetSeverities(w http.ResponseWriter, _ *http.Request) {
	severities := []string{
		string(audit.SeverityDebug),
		string(audit.SeverityInfo),
		string(audit.SeverityWarning),
		string(audit.SeverityError),
		string(audit.SeverityCritical),
	}

	writeJSON(w, map[string]interface{}{"severities": severities})
}

// ExportEvents handles GET /api/v1/audit/export
// Exports audit events in JSON or CEF format.
func (h *AuditHandlers) ExportEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// Build filter from query params
	filter := audit.DefaultQueryFilter()
	filter.Limit = 10000 // Max export limit

	// Parse time range
	if v := r.URL.Query().Get("start_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.StartTime = &t
		}
	}
	if v := r.URL.Query().Get("end_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.EndTime = &t
		}
	}

	// Parse type filter
	if types := r.URL.Query()["type"]; len(types) > 0 {
		for _, t := range types {
			filter.Types = append(filter.Types, audit.EventType(t))
		}
	}

	events, err := h.store.Query(ctx, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "EXPORT_ERROR", "Failed to query events for export", err)
		return
	}

	var data []byte
	var contentType string
	var filename string

	switch format {
	case "cef":
		exporter := audit.NewCEFExporter()
		data, err = exporter.Export(events)
		contentType = "text/plain"
		filename = "audit-events.cef"
	default:
		data, err = json.MarshalIndent(events, "", "  ")
		contentType = "application/json"
		filename = "audit-events.json"
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "EXPORT_ERROR", "Failed to export events", err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		// Log error but don't respond since headers are already sent
		return
	}
}
