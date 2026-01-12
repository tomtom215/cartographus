// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/detection"
)

// mockAlertStore implements DetectionAlertStore for testing.
type mockAlertStore struct {
	alerts         []detection.Alert
	saveErr        error
	getErr         error
	listErr        error
	acknowledgeErr error
	countErr       error
	lastFilter     detection.AlertFilter
	acknowledgedID int64
	acknowledgedBy string
}

func (m *mockAlertStore) SaveAlert(ctx context.Context, alert *detection.Alert) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	alert.ID = int64(len(m.alerts) + 1)
	m.alerts = append(m.alerts, *alert)
	return nil
}

func (m *mockAlertStore) GetAlert(ctx context.Context, id int64) (*detection.Alert, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, a := range m.alerts {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, nil
}

func (m *mockAlertStore) ListAlerts(ctx context.Context, filter detection.AlertFilter) ([]detection.Alert, error) {
	m.lastFilter = filter
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.alerts, nil
}

func (m *mockAlertStore) AcknowledgeAlert(ctx context.Context, id int64, acknowledgedBy string) error {
	if m.acknowledgeErr != nil {
		return m.acknowledgeErr
	}
	m.acknowledgedID = id
	m.acknowledgedBy = acknowledgedBy
	return nil
}

func (m *mockAlertStore) GetAlertCount(ctx context.Context, filter detection.AlertFilter) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return len(m.alerts), nil
}

// mockRuleStore implements DetectionRuleStore for testing.
type mockRuleStore struct {
	rules      []detection.Rule
	getErr     error
	listErr    error
	saveErr    error
	enableErr  error
	lastSaved  *detection.Rule
	lastEnable struct {
		ruleType detection.RuleType
		enabled  bool
	}
}

func (m *mockRuleStore) GetRule(ctx context.Context, ruleType detection.RuleType) (*detection.Rule, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, r := range m.rules {
		if r.RuleType == ruleType {
			return &r, nil
		}
	}
	return nil, nil
}

func (m *mockRuleStore) ListRules(ctx context.Context) ([]detection.Rule, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.rules, nil
}

func (m *mockRuleStore) SaveRule(ctx context.Context, rule *detection.Rule) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.lastSaved = rule
	return nil
}

func (m *mockRuleStore) SetRuleEnabled(ctx context.Context, ruleType detection.RuleType, enabled bool) error {
	if m.enableErr != nil {
		return m.enableErr
	}
	m.lastEnable.ruleType = ruleType
	m.lastEnable.enabled = enabled
	return nil
}

// mockTrustStore implements DetectionTrustStore for testing.
type mockTrustStore struct {
	scores        []detection.TrustScore
	getErr        error
	listErr       error
	lastThreshold int
}

func (m *mockTrustStore) GetTrustScore(ctx context.Context, userID int) (*detection.TrustScore, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, s := range m.scores {
		if s.UserID == userID {
			return &s, nil
		}
	}
	// Return default score for unknown users
	return &detection.TrustScore{
		UserID: userID,
		Score:  100,
	}, nil
}

func (m *mockTrustStore) ListLowTrustUsers(ctx context.Context, threshold int) ([]detection.TrustScore, error) {
	m.lastThreshold = threshold
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []detection.TrustScore
	for _, s := range m.scores {
		if s.Score < threshold {
			result = append(result, s)
		}
	}
	return result, nil
}

func TestNewDetectionHandlers(t *testing.T) {
	alertStore := &mockAlertStore{}
	ruleStore := &mockRuleStore{}
	trustStore := &mockTrustStore{}

	handlers := NewDetectionHandlers(alertStore, ruleStore, trustStore, nil)

	if handlers == nil {
		t.Fatal("NewDetectionHandlers returned nil")
	}
	if handlers.alertStore != alertStore {
		t.Error("alertStore not set correctly")
	}
	if handlers.ruleStore != ruleStore {
		t.Error("ruleStore not set correctly")
	}
	if handlers.trustStore != trustStore {
		t.Error("trustStore not set correctly")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	writeJSON(w, data)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", w.Header().Get("Content-Type"))
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("Unexpected response: %v", result)
	}
}

func TestDetectionHandlers_ListAlerts(t *testing.T) {
	now := time.Now()
	alerts := []detection.Alert{
		{ID: 1, RuleType: detection.RuleTypeImpossibleTravel, UserID: 1, Severity: detection.SeverityWarning, CreatedAt: now},
		{ID: 2, RuleType: detection.RuleTypeConcurrentStreams, UserID: 2, Severity: detection.SeverityCritical, CreatedAt: now},
	}

	tests := []struct {
		name           string
		query          string
		alerts         []detection.Alert
		listErr        error
		countErr       error
		wantStatus     int
		wantTotalField bool
	}{
		{
			name:           "success with alerts",
			query:          "",
			alerts:         alerts,
			wantStatus:     http.StatusOK,
			wantTotalField: true,
		},
		{
			name:           "with limit and offset",
			query:          "limit=10&offset=5",
			alerts:         alerts,
			wantStatus:     http.StatusOK,
			wantTotalField: true,
		},
		{
			name:           "with user_id filter",
			query:          "user_id=42",
			alerts:         nil,
			wantStatus:     http.StatusOK,
			wantTotalField: true,
		},
		{
			name:           "with acknowledged filter",
			query:          "acknowledged=true",
			alerts:         nil,
			wantStatus:     http.StatusOK,
			wantTotalField: true,
		},
		{
			name:           "with severity filter",
			query:          "severity=critical",
			alerts:         nil,
			wantStatus:     http.StatusOK,
			wantTotalField: true,
		},
		{
			name:           "with rule_type filter",
			query:          "rule_type=impossible_travel",
			alerts:         nil,
			wantStatus:     http.StatusOK,
			wantTotalField: true,
		},
		{
			name:           "with date filters",
			query:          "start_date=2024-01-01T00:00:00Z&end_date=2024-12-31T23:59:59Z",
			alerts:         nil,
			wantStatus:     http.StatusOK,
			wantTotalField: true,
		},
		{
			name:           "with ordering",
			query:          "order_by=severity&order_direction=asc",
			alerts:         nil,
			wantStatus:     http.StatusOK,
			wantTotalField: true,
		},
		{
			name:       "list error",
			query:      "",
			listErr:    errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:           "count error (still succeeds)",
			query:          "",
			alerts:         alerts,
			countErr:       errors.New("count error"),
			wantStatus:     http.StatusOK,
			wantTotalField: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alertStore := &mockAlertStore{
				alerts:   tt.alerts,
				listErr:  tt.listErr,
				countErr: tt.countErr,
			}
			handlers := NewDetectionHandlers(alertStore, nil, nil, nil)

			url := "/api/v1/detection/alerts"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handlers.ListAlerts(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantTotalField {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if _, ok := resp["total"]; !ok {
					t.Error("Response missing 'total' field")
				}
				if _, ok := resp["alerts"]; !ok {
					t.Error("Response missing 'alerts' field")
				}
			}
		})
	}
}

func TestDetectionHandlers_GetAlert(t *testing.T) {
	now := time.Now()
	alert := detection.Alert{
		ID:        1,
		RuleType:  detection.RuleTypeImpossibleTravel,
		UserID:    42,
		Severity:  detection.SeverityWarning,
		Title:     "Test Alert",
		CreatedAt: now,
	}

	tests := []struct {
		name       string
		alertID    string
		alerts     []detection.Alert
		getErr     error
		wantStatus int
	}{
		{
			name:       "success",
			alertID:    "1",
			alerts:     []detection.Alert{alert},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid ID",
			alertID:    "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			alertID:    "999",
			alerts:     []detection.Alert{alert},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "database error",
			alertID:    "1",
			getErr:     errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alertStore := &mockAlertStore{
				alerts: tt.alerts,
				getErr: tt.getErr,
			}
			handlers := NewDetectionHandlers(alertStore, nil, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/detection/alerts/"+tt.alertID, nil)
			req.SetPathValue("id", tt.alertID)
			w := httptest.NewRecorder()

			handlers.GetAlert(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestDetectionHandlers_AcknowledgeAlert(t *testing.T) {
	tests := []struct {
		name           string
		alertID        string
		body           string
		acknowledgeErr error
		wantStatus     int
		wantAckBy      string
	}{
		{
			name:       "success with user",
			alertID:    "1",
			body:       `{"acknowledged_by": "admin"}`,
			wantStatus: http.StatusOK,
			wantAckBy:  "admin",
		},
		{
			name:       "success without body (decode error triggers default)",
			alertID:    "1",
			body:       "invalid json",
			wantStatus: http.StatusOK,
			wantAckBy:  "system",
		},
		{
			name:       "invalid ID",
			alertID:    "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:           "database error",
			alertID:        "1",
			body:           `{"acknowledged_by": "admin"}`,
			acknowledgeErr: errors.New("database error"),
			wantStatus:     http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alertStore := &mockAlertStore{
				acknowledgeErr: tt.acknowledgeErr,
			}
			handlers := NewDetectionHandlers(alertStore, nil, nil, nil)

			body := bytes.NewBufferString(tt.body)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/detection/alerts/"+tt.alertID+"/acknowledge", body)
			req.SetPathValue("id", tt.alertID)
			w := httptest.NewRecorder()

			handlers.AcknowledgeAlert(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK && tt.wantAckBy != "" {
				if alertStore.acknowledgedBy != tt.wantAckBy {
					t.Errorf("AcknowledgedBy = %q, want %q", alertStore.acknowledgedBy, tt.wantAckBy)
				}
			}
		})
	}
}

func TestDetectionHandlers_ListRules(t *testing.T) {
	rules := []detection.Rule{
		{ID: 1, RuleType: detection.RuleTypeImpossibleTravel, Name: "Impossible Travel", Enabled: true},
		{ID: 2, RuleType: detection.RuleTypeConcurrentStreams, Name: "Concurrent Streams", Enabled: false},
	}

	tests := []struct {
		name       string
		rules      []detection.Rule
		listErr    error
		wantStatus int
	}{
		{
			name:       "success",
			rules:      rules,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty list",
			rules:      nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "database error",
			listErr:    errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleStore := &mockRuleStore{
				rules:   tt.rules,
				listErr: tt.listErr,
			}
			handlers := NewDetectionHandlers(nil, ruleStore, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/detection/rules", nil)
			w := httptest.NewRecorder()

			handlers.ListRules(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if _, ok := resp["rules"]; !ok {
					t.Error("Response missing 'rules' field")
				}
			}
		})
	}
}

func TestDetectionHandlers_GetRule(t *testing.T) {
	rule := detection.Rule{
		ID:       1,
		RuleType: detection.RuleTypeImpossibleTravel,
		Name:     "Impossible Travel",
		Enabled:  true,
	}

	tests := []struct {
		name       string
		ruleType   string
		rules      []detection.Rule
		getErr     error
		wantStatus int
	}{
		{
			name:       "success",
			ruleType:   "impossible_travel",
			rules:      []detection.Rule{rule},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			ruleType:   "unknown_type",
			rules:      []detection.Rule{rule},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "database error",
			ruleType:   "impossible_travel",
			getErr:     errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleStore := &mockRuleStore{
				rules:  tt.rules,
				getErr: tt.getErr,
			}
			handlers := NewDetectionHandlers(nil, ruleStore, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/detection/rules/"+tt.ruleType, nil)
			req.SetPathValue("type", tt.ruleType)
			w := httptest.NewRecorder()

			handlers.GetRule(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestDetectionHandlers_UpdateRule(t *testing.T) {
	rule := detection.Rule{
		ID:       1,
		RuleType: detection.RuleTypeImpossibleTravel,
		Name:     "Impossible Travel",
		Enabled:  true,
		Config:   []byte(`{"max_speed_kmh": 800}`),
	}

	tests := []struct {
		name       string
		ruleType   string
		body       string
		rules      []detection.Rule
		getErr     error
		saveErr    error
		wantStatus int
	}{
		{
			name:       "success",
			ruleType:   "impossible_travel",
			body:       `{"enabled": false, "config": {"max_speed_kmh": 1000}}`,
			rules:      []detection.Rule{rule},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid body",
			ruleType:   "impossible_travel",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rule not found",
			ruleType:   "unknown_type",
			body:       `{"enabled": false}`,
			rules:      []detection.Rule{rule},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "get error",
			ruleType:   "impossible_travel",
			body:       `{"enabled": false}`,
			getErr:     errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "save error",
			ruleType:   "impossible_travel",
			body:       `{"enabled": false}`,
			rules:      []detection.Rule{rule},
			saveErr:    errors.New("save error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleStore := &mockRuleStore{
				rules:   tt.rules,
				getErr:  tt.getErr,
				saveErr: tt.saveErr,
			}
			handlers := NewDetectionHandlers(nil, ruleStore, nil, nil)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/detection/rules/"+tt.ruleType, strings.NewReader(tt.body))
			req.SetPathValue("type", tt.ruleType)
			w := httptest.NewRecorder()

			handlers.UpdateRule(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestDetectionHandlers_SetRuleEnabled(t *testing.T) {
	tests := []struct {
		name       string
		ruleType   string
		body       string
		enableErr  error
		wantStatus int
	}{
		{
			name:       "enable success",
			ruleType:   "impossible_travel",
			body:       `{"enabled": true}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "disable success",
			ruleType:   "impossible_travel",
			body:       `{"enabled": false}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid body",
			ruleType:   "impossible_travel",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "database error",
			ruleType:   "impossible_travel",
			body:       `{"enabled": true}`,
			enableErr:  errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleStore := &mockRuleStore{
				enableErr: tt.enableErr,
			}
			handlers := NewDetectionHandlers(nil, ruleStore, nil, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/detection/rules/"+tt.ruleType+"/enable", strings.NewReader(tt.body))
			req.SetPathValue("type", tt.ruleType)
			w := httptest.NewRecorder()

			handlers.SetRuleEnabled(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestDetectionHandlers_GetUserTrustScore(t *testing.T) {
	score := detection.TrustScore{
		UserID:   42,
		Username: "testuser",
		Score:    75,
	}

	tests := []struct {
		name       string
		userID     string
		scores     []detection.TrustScore
		getErr     error
		wantStatus int
	}{
		{
			name:       "success with existing user",
			userID:     "42",
			scores:     []detection.TrustScore{score},
			wantStatus: http.StatusOK,
		},
		{
			name:       "success with new user (default score)",
			userID:     "999",
			scores:     nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid user ID",
			userID:     "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "database error",
			userID:     "42",
			getErr:     errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trustStore := &mockTrustStore{
				scores: tt.scores,
				getErr: tt.getErr,
			}
			handlers := NewDetectionHandlers(nil, nil, trustStore, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/detection/users/"+tt.userID+"/trust", nil)
			req.SetPathValue("id", tt.userID)
			w := httptest.NewRecorder()

			handlers.GetUserTrustScore(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestDetectionHandlers_ListLowTrustUsers(t *testing.T) {
	scores := []detection.TrustScore{
		{UserID: 1, Username: "user1", Score: 30},
		{UserID: 2, Username: "user2", Score: 45},
		{UserID: 3, Username: "user3", Score: 60},
	}

	tests := []struct {
		name          string
		query         string
		scores        []detection.TrustScore
		listErr       error
		wantStatus    int
		wantThreshold int
	}{
		{
			name:          "success with default threshold",
			query:         "",
			scores:        scores,
			wantStatus:    http.StatusOK,
			wantThreshold: 50,
		},
		{
			name:          "success with custom threshold",
			query:         "threshold=70",
			scores:        scores,
			wantStatus:    http.StatusOK,
			wantThreshold: 70,
		},
		{
			name:          "invalid threshold ignored",
			query:         "threshold=invalid",
			scores:        scores,
			wantStatus:    http.StatusOK,
			wantThreshold: 50,
		},
		{
			name:          "threshold out of range ignored",
			query:         "threshold=150",
			scores:        scores,
			wantStatus:    http.StatusOK,
			wantThreshold: 50,
		},
		{
			name:       "database error",
			query:      "",
			listErr:    errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trustStore := &mockTrustStore{
				scores:  tt.scores,
				listErr: tt.listErr,
			}
			handlers := NewDetectionHandlers(nil, nil, trustStore, nil)

			url := "/api/v1/detection/users/low-trust"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handlers.ListLowTrustUsers(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				if trustStore.lastThreshold != tt.wantThreshold {
					t.Errorf("Threshold = %d, want %d", trustStore.lastThreshold, tt.wantThreshold)
				}

				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if _, ok := resp["users"]; !ok {
					t.Error("Response missing 'users' field")
				}
				if _, ok := resp["threshold"]; !ok {
					t.Error("Response missing 'threshold' field")
				}
			}
		})
	}
}

func TestDetectionHandlers_GetEngineMetrics(t *testing.T) {
	t.Run("engine not available", func(t *testing.T) {
		handlers := NewDetectionHandlers(nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/detection/metrics", nil)
		w := httptest.NewRecorder()

		handlers.GetEngineMetrics(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("engine available", func(t *testing.T) {
		// Create engine with nil stores (for metrics test only)
		engine := detection.NewEngine(nil, nil, nil, nil)
		handlers := NewDetectionHandlers(nil, nil, nil, engine)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/detection/metrics", nil)
		w := httptest.NewRecorder()

		handlers.GetEngineMetrics(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

func TestDetectionHandlers_GetAlertStats(t *testing.T) {
	alerts := []detection.Alert{
		{ID: 1, RuleType: detection.RuleTypeImpossibleTravel, Severity: detection.SeverityCritical},
		{ID: 2, RuleType: detection.RuleTypeConcurrentStreams, Severity: detection.SeverityWarning},
		{ID: 3, RuleType: detection.RuleTypeDeviceVelocity, Severity: detection.SeverityInfo},
	}

	t.Run("success", func(t *testing.T) {
		alertStore := &mockAlertStore{
			alerts: alerts,
		}
		handlers := NewDetectionHandlers(alertStore, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/detection/stats", nil)
		w := httptest.NewRecorder()

		handlers.GetAlertStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if _, ok := resp["by_severity"]; !ok {
			t.Error("Response missing 'by_severity' field")
		}
		if _, ok := resp["by_rule_type"]; !ok {
			t.Error("Response missing 'by_rule_type' field")
		}
		if _, ok := resp["unacknowledged"]; !ok {
			t.Error("Response missing 'unacknowledged' field")
		}
		if _, ok := resp["total"]; !ok {
			t.Error("Response missing 'total' field")
		}
	})

	t.Run("with count errors (still succeeds)", func(t *testing.T) {
		alertStore := &mockAlertStore{
			alerts:   alerts,
			countErr: errors.New("count error"),
		}
		handlers := NewDetectionHandlers(alertStore, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/detection/stats", nil)
		w := httptest.NewRecorder()

		handlers.GetAlertStats(w, req)

		// Should still succeed even with count errors
		if w.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}
