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

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/detection"
	"github.com/tomtom215/cartographus/internal/logging"
)

// writeJSON encodes data as JSON and writes to the response.
// Logs errors but doesn't fail since headers are already sent.
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logging.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

// DetectionHandlers provides HTTP handlers for detection-related endpoints.
type DetectionHandlers struct {
	alertStore DetectionAlertStore
	ruleStore  DetectionRuleStore
	trustStore DetectionTrustStore
	engine     *detection.Engine
}

// DetectionAlertStore interface for dependency injection.
type DetectionAlertStore interface {
	SaveAlert(ctx context.Context, alert *detection.Alert) error
	GetAlert(ctx context.Context, id int64) (*detection.Alert, error)
	ListAlerts(ctx context.Context, filter detection.AlertFilter) ([]detection.Alert, error)
	AcknowledgeAlert(ctx context.Context, id int64, acknowledgedBy string) error
	GetAlertCount(ctx context.Context, filter detection.AlertFilter) (int, error)
}

// DetectionRuleStore interface for dependency injection.
type DetectionRuleStore interface {
	GetRule(ctx context.Context, ruleType detection.RuleType) (*detection.Rule, error)
	ListRules(ctx context.Context) ([]detection.Rule, error)
	SaveRule(ctx context.Context, rule *detection.Rule) error
	SetRuleEnabled(ctx context.Context, ruleType detection.RuleType, enabled bool) error
}

// DetectionTrustStore interface for dependency injection.
type DetectionTrustStore interface {
	GetTrustScore(ctx context.Context, userID int) (*detection.TrustScore, error)
	ListLowTrustUsers(ctx context.Context, threshold int) ([]detection.TrustScore, error)
}

// NewDetectionHandlers creates new detection handlers.
func NewDetectionHandlers(
	alertStore DetectionAlertStore,
	ruleStore DetectionRuleStore,
	trustStore DetectionTrustStore,
	engine *detection.Engine,
) *DetectionHandlers {
	return &DetectionHandlers{
		alertStore: alertStore,
		ruleStore:  ruleStore,
		trustStore: trustStore,
		engine:     engine,
	}
}

// ListAlerts handles GET /api/v1/detection/alerts
func (h *DetectionHandlers) ListAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := detection.AlertFilter{
		Limit: 100,
	}

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

	if v := r.URL.Query().Get("user_id"); v != "" {
		if userID, err := strconv.Atoi(v); err == nil {
			filter.UserID = &userID
		}
	}

	if v := r.URL.Query().Get("acknowledged"); v != "" {
		ack := v == "true"
		filter.Acknowledged = &ack
	}

	if v := r.URL.Query().Get("severity"); v != "" {
		filter.Severities = []detection.Severity{detection.Severity(v)}
	}

	if v := r.URL.Query().Get("rule_type"); v != "" {
		filter.RuleTypes = []detection.RuleType{detection.RuleType(v)}
	}

	if v := r.URL.Query().Get("start_date"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.StartDate = &t
		}
	}

	if v := r.URL.Query().Get("end_date"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.EndDate = &t
		}
	}

	filter.OrderBy = r.URL.Query().Get("order_by")
	if filter.OrderBy == "" {
		filter.OrderBy = "created_at"
	}
	filter.OrderDirection = r.URL.Query().Get("order_direction")
	if filter.OrderDirection == "" {
		filter.OrderDirection = "desc"
	}

	alerts, err := h.alertStore.ListAlerts(ctx, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETECTION_ERROR", "Failed to fetch alerts", err)
		return
	}

	// Get total count for pagination (best effort - don't fail if count fails)
	count, err := h.alertStore.GetAlertCount(ctx, filter)
	if err != nil {
		logging.Warn().Err(err).Msg("Failed to get alert count")
		count = len(alerts) // Fallback to current page count
	}

	response := map[string]interface{}{
		"alerts": alerts,
		"total":  count,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	}

	writeJSON(w, response)
}

// GetAlert handles GET /api/v1/detection/alerts/{id}
func (h *DetectionHandlers) GetAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid alert ID", err)
		return
	}

	alert, err := h.alertStore.GetAlert(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETECTION_ERROR", "Failed to fetch alert", err)
		return
	}
	if alert == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Alert not found", nil)
		return
	}

	writeJSON(w, alert)
}

// AcknowledgeAlert handles POST /api/v1/detection/alerts/{id}/acknowledge
func (h *DetectionHandlers) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid alert ID", err)
		return
	}

	// Get acknowledger from request (could be from auth context)
	var req struct {
		AcknowledgedBy string `json:"acknowledged_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.AcknowledgedBy = "system" // Default
	}

	if err := h.alertStore.AcknowledgeAlert(ctx, id, req.AcknowledgedBy); err != nil {
		respondError(w, http.StatusInternalServerError, "DETECTION_ERROR", "Failed to acknowledge alert", err)
		return
	}

	writeJSON(w, map[string]string{"status": "acknowledged"})
}

// ListRules handles GET /api/v1/detection/rules
func (h *DetectionHandlers) ListRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rules, err := h.ruleStore.ListRules(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETECTION_ERROR", "Failed to fetch rules", err)
		return
	}

	writeJSON(w, map[string]interface{}{"rules": rules})
}

// GetRule handles GET /api/v1/detection/rules/{type}
func (h *DetectionHandlers) GetRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ruleType := detection.RuleType(r.PathValue("type"))

	rule, err := h.ruleStore.GetRule(ctx, ruleType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETECTION_ERROR", "Failed to fetch rule", err)
		return
	}
	if rule == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Rule not found", nil)
		return
	}

	writeJSON(w, rule)
}

// UpdateRule handles PUT /api/v1/detection/rules/{type}
func (h *DetectionHandlers) UpdateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ruleType := detection.RuleType(r.PathValue("type"))

	var req struct {
		Enabled bool            `json:"enabled"`
		Config  json.RawMessage `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err)
		return
	}

	// Get existing rule
	rule, err := h.ruleStore.GetRule(ctx, ruleType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETECTION_ERROR", "Failed to fetch rule", err)
		return
	}
	if rule == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Rule not found", nil)
		return
	}

	// Update rule
	rule.Enabled = req.Enabled
	if len(req.Config) > 0 {
		rule.Config = req.Config
	}

	if err := h.ruleStore.SaveRule(ctx, rule); err != nil {
		respondError(w, http.StatusInternalServerError, "DETECTION_ERROR", "Failed to update rule", err)
		return
	}

	// Update in-memory detector if engine is available
	if h.engine != nil {
		if len(req.Config) > 0 {
			if err := h.engine.ConfigureDetector(ruleType, req.Config); err != nil {
				logging.Warn().Err(err).Str("rule_type", sanitizeLogValue(string(ruleType))).Msg("Failed to configure detector")
			}
		}
		if err := h.engine.SetDetectorEnabled(ruleType, req.Enabled); err != nil {
			logging.Warn().Err(err).Str("rule_type", sanitizeLogValue(string(ruleType))).Msg("Failed to set detector enabled state")
		}
	}

	writeJSON(w, rule)
}

// SetRuleEnabled handles POST /api/v1/detection/rules/{type}/enable
func (h *DetectionHandlers) SetRuleEnabled(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ruleType := detection.RuleType(r.PathValue("type"))

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err)
		return
	}

	if err := h.ruleStore.SetRuleEnabled(ctx, ruleType, req.Enabled); err != nil {
		respondError(w, http.StatusInternalServerError, "DETECTION_ERROR", "Failed to update rule", err)
		return
	}

	// Update in-memory detector
	if h.engine != nil {
		if err := h.engine.SetDetectorEnabled(ruleType, req.Enabled); err != nil {
			logging.Warn().Err(err).Str("rule_type", sanitizeLogValue(string(ruleType))).Msg("Failed to set detector enabled state")
		}
	}

	writeJSON(w, map[string]bool{"enabled": req.Enabled})
}

// GetUserTrustScore handles GET /api/v1/detection/users/{id}/trust
func (h *DetectionHandlers) GetUserTrustScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.PathValue("id")
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid user ID", err)
		return
	}

	score, err := h.trustStore.GetTrustScore(ctx, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETECTION_ERROR", "Failed to fetch trust score", err)
		return
	}

	writeJSON(w, score)
}

// ListLowTrustUsers handles GET /api/v1/detection/users/low-trust
func (h *DetectionHandlers) ListLowTrustUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	threshold := 50 // Default threshold
	if v := r.URL.Query().Get("threshold"); v != "" {
		if t, err := strconv.Atoi(v); err == nil && t >= 0 && t <= 100 {
			threshold = t
		}
	}

	scores, err := h.trustStore.ListLowTrustUsers(ctx, threshold)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETECTION_ERROR", "Failed to fetch low trust users", err)
		return
	}

	writeJSON(w, map[string]interface{}{
		"users":     scores,
		"threshold": threshold,
	})
}

// GetEngineMetrics handles GET /api/v1/detection/metrics
func (h *DetectionHandlers) GetEngineMetrics(w http.ResponseWriter, r *http.Request) {
	if h.engine == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Detection engine not available", nil)
		return
	}

	metrics := h.engine.Metrics()

	// Use pointer to avoid copying the RWMutex embedded in EngineMetrics
	writeJSON(w, &metrics)
}

// GetAlertStats handles GET /api/v1/detection/stats
func (h *DetectionHandlers) GetAlertStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Helper to get count with error logging (returns 0 on error)
	getCount := func(filter detection.AlertFilter) int {
		count, err := h.alertStore.GetAlertCount(ctx, filter)
		if err != nil {
			logging.Warn().Err(err).Msg("Failed to get alert count")
			return 0
		}
		return count
	}

	// Get counts by severity
	criticalCount := getCount(detection.AlertFilter{
		Severities: []detection.Severity{detection.SeverityCritical},
	})
	warningCount := getCount(detection.AlertFilter{
		Severities: []detection.Severity{detection.SeverityWarning},
	})
	infoCount := getCount(detection.AlertFilter{
		Severities: []detection.Severity{detection.SeverityInfo},
	})

	// Get unacknowledged count
	unackFalse := false
	unacknowledgedCount := getCount(detection.AlertFilter{
		Acknowledged: &unackFalse,
	})

	// Get counts by rule type
	impossibleTravelCount := getCount(detection.AlertFilter{
		RuleTypes: []detection.RuleType{detection.RuleTypeImpossibleTravel},
	})
	concurrentStreamsCount := getCount(detection.AlertFilter{
		RuleTypes: []detection.RuleType{detection.RuleTypeConcurrentStreams},
	})
	deviceVelocityCount := getCount(detection.AlertFilter{
		RuleTypes: []detection.RuleType{detection.RuleTypeDeviceVelocity},
	})

	stats := map[string]interface{}{
		"by_severity": map[string]int{
			"critical": criticalCount,
			"warning":  warningCount,
			"info":     infoCount,
		},
		"by_rule_type": map[string]int{
			"impossible_travel":  impossibleTravelCount,
			"concurrent_streams": concurrentStreamsCount,
			"device_velocity":    deviceVelocityCount,
		},
		"unacknowledged": unacknowledgedCount,
		"total":          criticalCount + warningCount + infoCount,
	}

	writeJSON(w, stats)
}
