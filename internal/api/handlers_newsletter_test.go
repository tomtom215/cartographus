// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/models"
)

// addAuthContext adds authentication context to a request for testing.
func addAuthContext(req *http.Request, userID, username, role string) *http.Request {
	subject := &auth.AuthSubject{
		ID:       userID,
		Username: username,
		Roles:    []string{role},
	}
	ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
	return req.WithContext(ctx)
}

// addChiURLParamNewsletter adds a chi URL parameter to a request for testing.
func addChiURLParamNewsletter(req *http.Request, key, value string) *http.Request {
	// Get existing context or create new one
	rctx := chi.RouteContext(req.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

// ============================================================================
// Validation Function Tests
// ============================================================================

func TestValidateTemplateCreateRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		req         *models.CreateTemplateRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_request",
			req: &models.CreateTemplateRequest{
				Name:     "Test Template",
				Subject:  "Test Subject",
				BodyHTML: "<html><body>Test</body></html>",
				Type:     models.NewsletterTypeWeeklyDigest,
			},
			expectError: false,
		},
		{
			name: "missing_name",
			req: &models.CreateTemplateRequest{
				Subject:  "Test Subject",
				BodyHTML: "<html><body>Test</body></html>",
				Type:     models.NewsletterTypeWeeklyDigest,
			},
			expectError: true,
			errorMsg:    "Template name is required",
		},
		{
			name: "missing_subject",
			req: &models.CreateTemplateRequest{
				Name:     "Test Template",
				BodyHTML: "<html><body>Test</body></html>",
				Type:     models.NewsletterTypeWeeklyDigest,
			},
			expectError: true,
			errorMsg:    "Subject is required",
		},
		{
			name: "missing_body",
			req: &models.CreateTemplateRequest{
				Name:    "Test Template",
				Subject: "Test Subject",
				Type:    models.NewsletterTypeWeeklyDigest,
			},
			expectError: true,
			errorMsg:    "HTML body is required",
		},
		{
			name: "invalid_type",
			req: &models.CreateTemplateRequest{
				Name:     "Test Template",
				Subject:  "Test Subject",
				BodyHTML: "<html><body>Test</body></html>",
				Type:     "invalid_type",
			},
			expectError: true,
			errorMsg:    "Invalid newsletter type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateTemplateCreateRequest(tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateScheduleCreateRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		req         *models.CreateScheduleRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_request",
			req: &models.CreateScheduleRequest{
				Name:           "Weekly Digest",
				TemplateID:     "template-123",
				CronExpression: "0 8 * * 1",
				Timezone:       "America/New_York",
				Recipients:     []models.NewsletterRecipient{{Type: "user", Target: "user-1"}},
				Channels:       []models.DeliveryChannel{models.DeliveryChannelEmail},
			},
			expectError: false,
		},
		{
			name: "missing_name",
			req: &models.CreateScheduleRequest{
				TemplateID:     "template-123",
				CronExpression: "0 8 * * 1",
				Timezone:       "America/New_York",
				Recipients:     []models.NewsletterRecipient{{Type: "user", Target: "user-1"}},
				Channels:       []models.DeliveryChannel{models.DeliveryChannelEmail},
			},
			expectError: true,
			errorMsg:    "Schedule name is required",
		},
		{
			name: "missing_template_id",
			req: &models.CreateScheduleRequest{
				Name:           "Weekly Digest",
				CronExpression: "0 8 * * 1",
				Timezone:       "America/New_York",
				Recipients:     []models.NewsletterRecipient{{Type: "user", Target: "user-1"}},
				Channels:       []models.DeliveryChannel{models.DeliveryChannelEmail},
			},
			expectError: true,
			errorMsg:    "Template ID is required",
		},
		{
			name: "missing_cron",
			req: &models.CreateScheduleRequest{
				Name:       "Weekly Digest",
				TemplateID: "template-123",
				Timezone:   "America/New_York",
				Recipients: []models.NewsletterRecipient{{Type: "user", Target: "user-1"}},
				Channels:   []models.DeliveryChannel{models.DeliveryChannelEmail},
			},
			expectError: true,
			errorMsg:    "Cron expression is required",
		},
		{
			name: "missing_timezone",
			req: &models.CreateScheduleRequest{
				Name:           "Weekly Digest",
				TemplateID:     "template-123",
				CronExpression: "0 8 * * 1",
				Recipients:     []models.NewsletterRecipient{{Type: "user", Target: "user-1"}},
				Channels:       []models.DeliveryChannel{models.DeliveryChannelEmail},
			},
			expectError: true,
			errorMsg:    "Timezone is required",
		},
		{
			name: "no_recipients",
			req: &models.CreateScheduleRequest{
				Name:           "Weekly Digest",
				TemplateID:     "template-123",
				CronExpression: "0 8 * * 1",
				Timezone:       "America/New_York",
				Recipients:     []models.NewsletterRecipient{},
				Channels:       []models.DeliveryChannel{models.DeliveryChannelEmail},
			},
			expectError: true,
			errorMsg:    "At least one recipient is required",
		},
		{
			name: "no_channels",
			req: &models.CreateScheduleRequest{
				Name:           "Weekly Digest",
				TemplateID:     "template-123",
				CronExpression: "0 8 * * 1",
				Timezone:       "America/New_York",
				Recipients:     []models.NewsletterRecipient{{Type: "user", Target: "user-1"}},
				Channels:       []models.DeliveryChannel{},
			},
			expectError: true,
			errorMsg:    "At least one channel is required",
		},
		{
			name: "invalid_channel",
			req: &models.CreateScheduleRequest{
				Name:           "Weekly Digest",
				TemplateID:     "template-123",
				CronExpression: "0 8 * * 1",
				Timezone:       "America/New_York",
				Recipients:     []models.NewsletterRecipient{{Type: "user", Target: "user-1"}},
				Channels:       []models.DeliveryChannel{"invalid_channel"},
			},
			expectError: true,
			errorMsg:    "Invalid delivery channel: invalid_channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateScheduleCreateRequest(tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestParsePaginationParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		query          string
		defaultLimit   int
		maxLimit       int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "defaults",
			query:          "",
			defaultLimit:   50,
			maxLimit:       200,
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "valid_limit",
			query:          "limit=100",
			defaultLimit:   50,
			maxLimit:       200,
			expectedLimit:  100,
			expectedOffset: 0,
		},
		{
			name:           "valid_offset",
			query:          "offset=25",
			defaultLimit:   50,
			maxLimit:       200,
			expectedLimit:  50,
			expectedOffset: 25,
		},
		{
			name:           "both_params",
			query:          "limit=75&offset=50",
			defaultLimit:   50,
			maxLimit:       200,
			expectedLimit:  75,
			expectedOffset: 50,
		},
		{
			name:           "limit_exceeds_max",
			query:          "limit=500",
			defaultLimit:   50,
			maxLimit:       200,
			expectedLimit:  50, // Falls back to default
			expectedOffset: 0,
		},
		{
			name:           "negative_limit",
			query:          "limit=-10",
			defaultLimit:   50,
			maxLimit:       200,
			expectedLimit:  50, // Falls back to default
			expectedOffset: 0,
		},
		{
			name:           "negative_offset",
			query:          "offset=-10",
			defaultLimit:   50,
			maxLimit:       200,
			expectedLimit:  50,
			expectedOffset: 0, // Falls back to 0
		},
		{
			name:           "invalid_limit_string",
			query:          "limit=abc",
			defaultLimit:   50,
			maxLimit:       200,
			expectedLimit:  50, // Falls back to default
			expectedOffset: 0,
		},
		{
			name:           "zero_limit",
			query:          "limit=0",
			defaultLimit:   50,
			maxLimit:       200,
			expectedLimit:  50, // Falls back to default
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			limit, offset := parsePaginationParams(req, tt.defaultLimit, tt.maxLimit)

			if limit != tt.expectedLimit {
				t.Errorf("Expected limit %d, got %d", tt.expectedLimit, limit)
			}
			if offset != tt.expectedOffset {
				t.Errorf("Expected offset %d, got %d", tt.expectedOffset, offset)
			}
		})
	}
}

func TestParseBoolParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		param    string
		expected *bool
	}{
		{
			name:     "not_present",
			query:    "",
			param:    "active",
			expected: nil,
		},
		{
			name:     "true_value",
			query:    "active=true",
			param:    "active",
			expected: boolPtr(true),
		},
		{
			name:     "false_value",
			query:    "active=false",
			param:    "active",
			expected: boolPtr(false),
		},
		{
			name:     "any_other_value",
			query:    "active=yes",
			param:    "active",
			expected: boolPtr(false), // Only "true" is true
		},
		{
			name:     "empty_value",
			query:    "active=",
			param:    "active",
			expected: nil, // Empty string treated as not present
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			result := parseBoolParam(req, tt.param)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", *result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %v, got nil", *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("Expected %v, got %v", *tt.expected, *result)
				}
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestGetClientIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "from_remote_addr",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.100:12345",
			expected:   "192.168.1.100:12345",
		},
		{
			name: "from_x_forwarded_for",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
			},
			remoteAddr: "192.168.1.100:12345",
			expected:   "10.0.0.1",
		},
		{
			name: "from_x_forwarded_for_multiple",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1, 10.0.0.2, 10.0.0.3",
			},
			remoteAddr: "192.168.1.100:12345",
			expected:   "10.0.0.1",
		},
		{
			name: "from_x_real_ip",
			headers: map[string]string{
				"X-Real-IP": "172.16.0.50",
			},
			remoteAddr: "192.168.1.100:12345",
			expected:   "172.16.0.50",
		},
		{
			name: "xff_takes_precedence_over_xri",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
				"X-Real-IP":       "172.16.0.50",
			},
			remoteAddr: "192.168.1.100:12345",
			expected:   "10.0.0.1",
		},
		{
			name: "trim_whitespace_in_xff",
			headers: map[string]string{
				"X-Forwarded-For": "  10.0.0.1  ",
			},
			remoteAddr: "192.168.1.100:12345",
			expected:   "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result := getClientIP(req)

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// Builder Function Tests
// ============================================================================

func TestBuildTemplateFromRequest(t *testing.T) {
	t.Parallel()

	req := &models.CreateTemplateRequest{
		Name:        "Weekly Digest",
		Description: "A weekly digest template",
		Type:        models.NewsletterTypeWeeklyDigest,
		Subject:     "Your Weekly Digest",
		BodyHTML:    "<html><body>Content</body></html>",
		BodyText:    "Content",
		DefaultConfig: &models.TemplateConfig{
			TimeFrame:     7,
			MaxItems:      10,
			IncludeMovies: true,
			IncludeShows:  true,
		},
	}

	userID := "user-123"
	template := buildTemplateFromRequest(req, userID)

	if template.Name != req.Name {
		t.Errorf("Expected name %q, got %q", req.Name, template.Name)
	}
	if template.Description != req.Description {
		t.Errorf("Expected description %q, got %q", req.Description, template.Description)
	}
	if template.Type != req.Type {
		t.Errorf("Expected type %q, got %q", req.Type, template.Type)
	}
	if template.Subject != req.Subject {
		t.Errorf("Expected subject %q, got %q", req.Subject, template.Subject)
	}
	if template.BodyHTML != req.BodyHTML {
		t.Errorf("Expected body HTML %q, got %q", req.BodyHTML, template.BodyHTML)
	}
	if template.CreatedBy != userID {
		t.Errorf("Expected createdBy %q, got %q", userID, template.CreatedBy)
	}
	if template.Version != 1 {
		t.Errorf("Expected version 1, got %d", template.Version)
	}
	if template.IsBuiltIn {
		t.Error("Expected IsBuiltIn to be false")
	}
	if !template.IsActive {
		t.Error("Expected IsActive to be true")
	}
	if template.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if template.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestBuildScheduleFromRequest(t *testing.T) {
	t.Parallel()

	req := &models.CreateScheduleRequest{
		Name:           "Weekly Monday Digest",
		Description:    "Sent every Monday at 8am",
		TemplateID:     "template-abc",
		Recipients:     []models.NewsletterRecipient{{Type: "user", Target: "user-1"}},
		CronExpression: "0 8 * * 1",
		Timezone:       "America/New_York",
		Channels:       []models.DeliveryChannel{models.DeliveryChannelEmail},
		IsEnabled:      true,
	}

	userID := "admin-user"
	schedule := buildScheduleFromRequest(req, userID)

	if schedule.Name != req.Name {
		t.Errorf("Expected name %q, got %q", req.Name, schedule.Name)
	}
	if schedule.TemplateID != req.TemplateID {
		t.Errorf("Expected templateID %q, got %q", req.TemplateID, schedule.TemplateID)
	}
	if schedule.CronExpression != req.CronExpression {
		t.Errorf("Expected cron %q, got %q", req.CronExpression, schedule.CronExpression)
	}
	if schedule.Timezone != req.Timezone {
		t.Errorf("Expected timezone %q, got %q", req.Timezone, schedule.Timezone)
	}
	if schedule.CreatedBy != userID {
		t.Errorf("Expected createdBy %q, got %q", userID, schedule.CreatedBy)
	}
	if !schedule.IsEnabled {
		t.Error("Expected IsEnabled to be true")
	}
	if len(schedule.Recipients) != 1 {
		t.Errorf("Expected 1 recipient, got %d", len(schedule.Recipients))
	}
	if len(schedule.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(schedule.Channels))
	}
}

// ============================================================================
// Sample Data Generation Tests
// ============================================================================

func TestGenerateSampleContentData(t *testing.T) {
	t.Parallel()

	config := &models.TemplateConfig{
		TimeFrame:         7,
		MaxItems:          10,
		IncludeMovies:     true,
		IncludeShows:      true,
		IncludeTopContent: true,
		IncludeStats:      true,
	}

	data := generateSampleContentData(models.NewsletterTypeWeeklyDigest, config)

	if data.ServerName == "" {
		t.Error("Expected ServerName to be set")
	}
	if data.ServerURL == "" {
		t.Error("Expected ServerURL to be set")
	}
	if data.GeneratedAt.IsZero() {
		t.Error("Expected GeneratedAt to be set")
	}
	if data.DateRangeStart.IsZero() {
		t.Error("Expected DateRangeStart to be set")
	}
	if data.DateRangeEnd.IsZero() {
		t.Error("Expected DateRangeEnd to be set")
	}
	if data.DateRangeDisplay == "" {
		t.Error("Expected DateRangeDisplay to be set")
	}
	if len(data.NewMovies) == 0 {
		t.Error("Expected NewMovies to be populated")
	}
	if len(data.NewShows) == 0 {
		t.Error("Expected NewShows to be populated")
	}
	if len(data.TopMovies) == 0 {
		t.Error("Expected TopMovies to be populated")
	}
	if len(data.TopShows) == 0 {
		t.Error("Expected TopShows to be populated")
	}
	if data.Stats == nil {
		t.Error("Expected Stats to be populated")
	}
}

func TestGenerateSampleContentData_ServerHealthType(t *testing.T) {
	t.Parallel()

	config := &models.TemplateConfig{
		TimeFrame: 7,
	}

	data := generateSampleContentData(models.NewsletterTypeServerHealth, config)

	if data.Health == nil {
		t.Error("Expected Health to be populated for server_health type")
	}
	if data.Health.ServerStatus != "healthy" {
		t.Errorf("Expected server status 'healthy', got %q", data.Health.ServerStatus)
	}
}

func TestGenerateSampleContentData_NoMovies(t *testing.T) {
	t.Parallel()

	config := &models.TemplateConfig{
		TimeFrame:     7,
		IncludeMovies: false,
		IncludeShows:  true,
	}

	data := generateSampleContentData(models.NewsletterTypeWeeklyDigest, config)

	if len(data.NewMovies) > 0 {
		t.Error("Expected no movies when IncludeMovies is false")
	}
	if len(data.NewShows) == 0 {
		t.Error("Expected shows when IncludeShows is true")
	}
}

func TestGenerateSampleContentData_NoStats(t *testing.T) {
	t.Parallel()

	config := &models.TemplateConfig{
		TimeFrame:    7,
		IncludeStats: false,
	}

	data := generateSampleContentData(models.NewsletterTypeWeeklyDigest, config)

	if data.Stats != nil {
		t.Error("Expected no stats when IncludeStats is false")
	}
}

func TestGenerateSampleMovies(t *testing.T) {
	t.Parallel()

	now := time.Now()
	movies := generateSampleMovies(now)

	if len(movies) < 2 {
		t.Errorf("Expected at least 2 movies, got %d", len(movies))
	}

	// Check first movie
	movie := movies[0]
	if movie.Title == "" {
		t.Error("Expected movie title")
	}
	if movie.Year == 0 {
		t.Error("Expected movie year")
	}
	if movie.MediaType != "movie" {
		t.Errorf("Expected media type 'movie', got %q", movie.MediaType)
	}
}

func TestGenerateSampleShows(t *testing.T) {
	t.Parallel()

	now := time.Now()
	shows := generateSampleShows(now)

	if len(shows) < 1 {
		t.Errorf("Expected at least 1 show, got %d", len(shows))
	}

	// Check first show
	show := shows[0]
	if show.Title == "" {
		t.Error("Expected show title")
	}
	if show.Year == 0 {
		t.Error("Expected show year")
	}
	if show.NewEpisodesCount == 0 {
		t.Error("Expected new episodes count")
	}
	if len(show.Seasons) == 0 {
		t.Error("Expected seasons")
	}
}

// ============================================================================
// Audit Action Tests
// ============================================================================

func TestDetermineScheduleAuditAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *models.UpdateScheduleRequest
		expected string
	}{
		{
			name:     "nil_enabled",
			req:      &models.UpdateScheduleRequest{},
			expected: models.NewsletterAuditActionUpdate,
		},
		{
			name:     "enable_schedule",
			req:      &models.UpdateScheduleRequest{IsEnabled: boolPtr(true)},
			expected: models.NewsletterAuditActionEnable,
		},
		{
			name:     "disable_schedule",
			req:      &models.UpdateScheduleRequest{IsEnabled: boolPtr(false)},
			expected: models.NewsletterAuditActionDisable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := determineScheduleAuditAction(tt.req)

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// Config Resolution Tests
// ============================================================================

func TestResolveTemplateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestConfig  *models.TemplateConfig
		templateConfig *models.TemplateConfig
		expectedFrame  int
	}{
		{
			name:           "use_request_config",
			requestConfig:  &models.TemplateConfig{TimeFrame: 14},
			templateConfig: &models.TemplateConfig{TimeFrame: 7},
			expectedFrame:  14,
		},
		{
			name:           "use_template_config",
			requestConfig:  nil,
			templateConfig: &models.TemplateConfig{TimeFrame: 7},
			expectedFrame:  7,
		},
		{
			name:           "use_default",
			requestConfig:  nil,
			templateConfig: nil,
			expectedFrame:  7, // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := resolveTemplateConfig(tt.requestConfig, tt.templateConfig)

			if result.TimeFrame != tt.expectedFrame {
				t.Errorf("Expected TimeFrame %d, got %d", tt.expectedFrame, result.TimeFrame)
			}
		})
	}
}

// ============================================================================
// Validation Error Tests
// ============================================================================

func TestErrValidation(t *testing.T) {
	t.Parallel()

	msg := "Test validation error"
	err := ErrValidation(msg)

	if err == nil {
		t.Fatal("Expected error to be non-nil")
	}

	if err.Error() != msg {
		t.Errorf("Expected error message %q, got %q", msg, err.Error())
	}
}

func TestValidationError_Type(t *testing.T) {
	t.Parallel()

	err := ErrValidation("test")

	// Should implement error interface
	var _ error = err

	// Check type assertion
	_, ok := err.(*validationError)
	if !ok {
		t.Error("Expected *validationError type")
	}
}

// ============================================================================
// Handler Authentication Tests
// ============================================================================

func TestNewsletterTemplateCreate_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/newsletter/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context
	req = addAuthContext(req, "user-1", "testuser", "editor")

	rec := httptest.NewRecorder()
	handler.NewsletterTemplateCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected error object in response, got %v", response)
	}
	if code, ok := errObj["code"].(string); !ok || code != "INVALID_JSON" {
		t.Errorf("Expected error code INVALID_JSON, got %v", errObj["code"])
	}
}

func TestNewsletterScheduleCreate_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/newsletter/schedules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context
	req = addAuthContext(req, "user-1", "testuser", "editor")

	rec := httptest.NewRecorder()
	handler.NewsletterScheduleCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterTemplateUpdate_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/newsletter/templates/test-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add chi URL params and auth context
	req = addChiURLParamNewsletter(req, "id", "test-id")
	req = addAuthContext(req, "user-1", "testuser", "editor")

	rec := httptest.NewRecorder()
	handler.NewsletterTemplateUpdate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterScheduleUpdate_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/newsletter/schedules/test-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add chi URL params and auth context
	req = addChiURLParamNewsletter(req, "id", "test-id")
	req = addAuthContext(req, "user-1", "testuser", "editor")

	rec := httptest.NewRecorder()
	handler.NewsletterScheduleUpdate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterUserPreferencesUpdate_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/user/newsletter/preferences", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context
	req = addAuthContext(req, "user-1", "testuser", "viewer")

	rec := httptest.NewRecorder()
	handler.NewsletterUserPreferencesUpdate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterTemplatePreview_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/newsletter/templates/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context
	req = addAuthContext(req, "user-1", "testuser", "editor")

	rec := httptest.NewRecorder()
	handler.NewsletterTemplatePreview(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterTemplatePreview_MissingTemplateID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/newsletter/templates/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context
	req = addAuthContext(req, "user-1", "testuser", "editor")

	rec := httptest.NewRecorder()
	handler.NewsletterTemplatePreview(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected error object in response, got %v", response)
	}
	if code, ok := errObj["code"].(string); !ok || code != "VALIDATION_ERROR" {
		t.Errorf("Expected error code VALIDATION_ERROR, got %v", errObj["code"])
	}
}

// ============================================================================
// Missing ID Tests
// ============================================================================

func TestNewsletterTemplateGet_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/templates/", nil)
	req = addChiURLParamNewsletter(req, "id", "")
	req = addAuthContext(req, "user-1", "testuser", "viewer")

	rec := httptest.NewRecorder()
	handler.NewsletterTemplateGet(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterTemplateUpdate_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/newsletter/templates/", strings.NewReader(`{}`))
	req = addChiURLParamNewsletter(req, "id", "")
	req = addAuthContext(req, "user-1", "testuser", "editor")

	rec := httptest.NewRecorder()
	handler.NewsletterTemplateUpdate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterTemplateDelete_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/newsletter/templates/", nil)
	req = addChiURLParamNewsletter(req, "id", "")
	req = addAuthContext(req, "user-1", "testuser", "admin")

	rec := httptest.NewRecorder()
	handler.NewsletterTemplateDelete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterScheduleGet_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/schedules/", nil)
	req = addChiURLParamNewsletter(req, "id", "")
	req = addAuthContext(req, "user-1", "testuser", "viewer")

	rec := httptest.NewRecorder()
	handler.NewsletterScheduleGet(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterScheduleUpdate_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/newsletter/schedules/", strings.NewReader(`{}`))
	req = addChiURLParamNewsletter(req, "id", "")
	req = addAuthContext(req, "user-1", "testuser", "editor")

	rec := httptest.NewRecorder()
	handler.NewsletterScheduleUpdate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterScheduleDelete_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/newsletter/schedules/", nil)
	req = addChiURLParamNewsletter(req, "id", "")
	req = addAuthContext(req, "user-1", "testuser", "admin")

	rec := httptest.NewRecorder()
	handler.NewsletterScheduleDelete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterScheduleTrigger_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/newsletter/schedules//trigger", nil)
	req = addChiURLParamNewsletter(req, "id", "")
	req = addAuthContext(req, "user-1", "testuser", "editor")

	rec := httptest.NewRecorder()
	handler.NewsletterScheduleTrigger(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewsletterDeliveryGet_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/deliveries/", nil)
	req = addChiURLParamNewsletter(req, "id", "")
	req = addAuthContext(req, "user-1", "testuser", "viewer")

	rec := httptest.NewRecorder()
	handler.NewsletterDeliveryGet(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ========================================
// NewsletterTemplateList Tests (0% coverage)
// ========================================

func TestNewsletterTemplateList_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/templates", nil)
	req = addAuthContext(req, "user-1", "testuser", "admin")
	rec := httptest.NewRecorder()

	handler.NewsletterTemplateList(rec, req)

	// Should succeed (possibly empty list)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNewsletterTemplateList_WithPagination(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/templates?limit=10&offset=0", nil)
	req = addAuthContext(req, "user-1", "testuser", "admin")
	rec := httptest.NewRecorder()

	handler.NewsletterTemplateList(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// ========================================
// NewsletterScheduleList Tests (0% coverage)
// ========================================

func TestNewsletterScheduleList_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/schedules", nil)
	req = addAuthContext(req, "user-1", "testuser", "admin")
	rec := httptest.NewRecorder()

	handler.NewsletterScheduleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNewsletterScheduleList_WithFilters(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/schedules?enabled=true&limit=50", nil)
	req = addAuthContext(req, "user-1", "testuser", "admin")
	rec := httptest.NewRecorder()

	handler.NewsletterScheduleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// ========================================
// NewsletterDeliveryList Tests (0% coverage)
// ========================================

func TestNewsletterDeliveryList_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/deliveries", nil)
	req = addAuthContext(req, "user-1", "testuser", "admin")
	rec := httptest.NewRecorder()

	handler.NewsletterDeliveryList(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNewsletterDeliveryList_WithFilters(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/deliveries?status=delivered&limit=50", nil)
	req = addAuthContext(req, "user-1", "testuser", "admin")
	rec := httptest.NewRecorder()

	handler.NewsletterDeliveryList(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// ========================================
// NewsletterStats Tests (0% coverage)
// ========================================

func TestNewsletterStats_Unauthorized(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	// No auth context - should be unauthorized
	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/stats", nil)
	rec := httptest.NewRecorder()

	handler.NewsletterStats(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestNewsletterStats_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/stats", nil)
	req = addAuthContext(req, "user-1", "testuser", "viewer") // Any authenticated user works
	rec := httptest.NewRecorder()

	handler.NewsletterStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// ========================================
// NewsletterAuditLog Tests (0% coverage)
// ========================================

func TestNewsletterAuditLog_Unauthorized(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/audit", nil)
	req = addAuthContext(req, "user-1", "testuser", "viewer") // Not admin
	rec := httptest.NewRecorder()

	handler.NewsletterAuditLog(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestNewsletterAuditLog_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/audit", nil)
	req = addAuthContext(req, "user-1", "testuser", "admin")
	rec := httptest.NewRecorder()

	handler.NewsletterAuditLog(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNewsletterAuditLog_WithFilters(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/audit?resource_type=template&action=create&limit=50", nil)
	req = addAuthContext(req, "user-1", "testuser", "admin")
	rec := httptest.NewRecorder()

	handler.NewsletterAuditLog(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// ========================================
// NewsletterUserPreferencesGet Tests (0% coverage)
// ========================================

func TestNewsletterUserPreferencesGet_Unauthorized(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	// Test without auth context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/preferences", nil)
	rec := httptest.NewRecorder()

	handler.NewsletterUserPreferencesGet(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestNewsletterUserPreferencesGet_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/newsletter/preferences", nil)
	req = addAuthContext(req, "user-1", "testuser", "viewer")
	rec := httptest.NewRecorder()

	handler.NewsletterUserPreferencesGet(rec, req)

	// Should succeed or return not found (both acceptable)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 200 or 404, got %d", rec.Code)
	}
}

// ========================================
// NewsletterUnsubscribe Tests (0% coverage)
// ========================================

func TestNewsletterUnsubscribe_Unauthorized(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	// No auth context - should be unauthorized
	req := httptest.NewRequest(http.MethodPost, "/api/v1/newsletter/unsubscribe", nil)
	rec := httptest.NewRecorder()

	handler.NewsletterUnsubscribe(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
