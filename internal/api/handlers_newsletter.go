// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus application.
//
// handlers_newsletter.go - Newsletter Generator API Handlers
//
// This file contains HTTP handlers for the Newsletter Generator system.
// Newsletters enable automated delivery of media digests via multiple channels.
//
// Endpoints:
//
// Templates (require editor/admin role):
//   - GET    /api/v1/newsletter/templates          - List templates
//   - POST   /api/v1/newsletter/templates          - Create template
//   - GET    /api/v1/newsletter/templates/{id}     - Get template
//   - PUT    /api/v1/newsletter/templates/{id}     - Update template
//   - DELETE /api/v1/newsletter/templates/{id}     - Delete template
//   - POST   /api/v1/newsletter/templates/preview  - Preview template
//
// Schedules (require editor/admin role):
//   - GET    /api/v1/newsletter/schedules          - List schedules
//   - POST   /api/v1/newsletter/schedules          - Create schedule
//   - GET    /api/v1/newsletter/schedules/{id}     - Get schedule
//   - PUT    /api/v1/newsletter/schedules/{id}     - Update schedule
//   - DELETE /api/v1/newsletter/schedules/{id}     - Delete schedule
//   - POST   /api/v1/newsletter/schedules/{id}/trigger - Trigger immediate send
//
// Deliveries (require editor/admin role for send, viewer for list):
//   - GET    /api/v1/newsletter/deliveries         - List delivery history
//   - GET    /api/v1/newsletter/deliveries/{id}    - Get delivery details
//   - POST   /api/v1/newsletter/send               - Send newsletter immediately
//
// User Preferences (authenticated users):
//   - GET    /api/v1/user/newsletter/preferences   - Get user's preferences
//   - PUT    /api/v1/user/newsletter/preferences   - Update user's preferences
//   - POST   /api/v1/user/newsletter/unsubscribe   - Unsubscribe from all
//
// Stats (require viewer role):
//   - GET    /api/v1/newsletter/stats              - Get newsletter statistics
//
// Audit (require admin role):
//   - GET    /api/v1/newsletter/audit              - Get audit log
//
// Security:
//   - RBAC enforced on all endpoints
//   - Templates are sanitized to prevent XSS
//   - Credentials are encrypted at rest
//   - Webhook URLs are validated
//   - Audit logging for all operations
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/newsletter"
)

// ============================================================================
// Template Endpoints
// ============================================================================

// NewsletterTemplateList returns all newsletter templates.
//
// Method: GET
// Path: /api/v1/newsletter/templates
//
// Query Parameters:
//   - type: Filter by newsletter type
//   - active: Filter by active status (true/false)
//   - limit: Maximum results (default: 50)
//   - offset: Pagination offset
//
// Response: ListTemplatesResponse
//
// Authentication: Required
// Authorization: Viewer role or higher
func (h *Handler) NewsletterTemplateList(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return
	}

	start := time.Now()

	// Parse query parameters
	templateType := r.URL.Query().Get("type")
	limit, offset := parsePaginationParams(r, 50, 200)
	activeFilter := parseBoolParam(r, "active")

	templates, totalCount, err := h.db.ListNewsletterTemplates(r.Context(), templateType, activeFilter, limit, offset)
	if err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Msg("Failed to list newsletter templates")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list templates", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: models.ListTemplatesResponse{
			Templates:  templates,
			TotalCount: totalCount,
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterTemplateCreate creates a new newsletter template.
//
// Method: POST
// Path: /api/v1/newsletter/templates
//
// Request Body: CreateTemplateRequest
//
// Response: NewsletterTemplate
//
// Authentication: Required
// Authorization: Editor role or higher
func (h *Handler) NewsletterTemplateCreate(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireEditor(w, r, "create templates")
	if hctx == nil {
		return
	}

	var req models.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	// Validate request
	if err := validateTemplateCreateRequest(&req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Validate template syntax
	engine := newsletter.NewTemplateEngine()
	if err := engine.ValidateTemplate(req.BodyHTML); err != nil {
		respondError(w, http.StatusBadRequest, "TEMPLATE_ERROR", "Invalid template syntax: "+err.Error(), nil)
		return
	}

	start := time.Now()

	// Create template
	template := buildTemplateFromRequest(&req, hctx.UserID)

	if err := h.db.CreateNewsletterTemplate(r.Context(), template); err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Str("user_id", hctx.UserID).
			Msg("Failed to create newsletter template")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create template", err)
		return
	}

	//nolint:errcheck // Audit log errors don't block the operation
	_ = h.auditNewsletter(r, hctx, models.NewsletterAuditActionCreate, models.NewsletterResourceTemplate, template.ID, template.Name, nil)

	log.Info().
		Str("template_id", template.ID).
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Str("name", template.Name).
		Str("type", string(template.Type)).
		Msg("Newsletter template created")

	respondJSON(w, http.StatusCreated, &models.APIResponse{
		Status: "success",
		Data:   template,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterTemplateGet retrieves a specific newsletter template.
//
// Method: GET
// Path: /api/v1/newsletter/templates/{id}
//
// URL Parameters:
//   - id: Template ID
//
// Response: NewsletterTemplate
//
// Authentication: Required
// Authorization: Viewer role or higher
func (h *Handler) NewsletterTemplateGet(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return
	}

	templateID := chi.URLParam(r, "id")
	if templateID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Template ID is required", nil)
		return
	}

	start := time.Now()

	template, err := h.db.GetNewsletterTemplate(r.Context(), templateID)
	if err != nil {
		log.Error().Err(err).
			Str("template_id", templateID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter template")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get template", err)
		return
	}

	if template == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Template not found", nil)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   template,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterTemplateUpdate updates a newsletter template.
//
// Method: PUT
// Path: /api/v1/newsletter/templates/{id}
//
// URL Parameters:
//   - id: Template ID
//
// Request Body: UpdateTemplateRequest
//
// Response: NewsletterTemplate
//
// Authentication: Required
// Authorization: Editor role or higher
func (h *Handler) NewsletterTemplateUpdate(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireEditor(w, r, "update templates")
	if hctx == nil {
		return
	}

	templateID := chi.URLParam(r, "id")
	if templateID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Template ID is required", nil)
		return
	}

	var req models.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	// Validate template syntax if body is being updated
	if req.BodyHTML != nil {
		engine := newsletter.NewTemplateEngine()
		if err := engine.ValidateTemplate(*req.BodyHTML); err != nil {
			respondError(w, http.StatusBadRequest, "TEMPLATE_ERROR", "Invalid template syntax: "+err.Error(), nil)
			return
		}
	}

	start := time.Now()

	// Get existing template
	existing, err := h.db.GetNewsletterTemplate(r.Context(), templateID)
	if err != nil {
		log.Error().Err(err).
			Str("template_id", templateID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter template for update")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get template", err)
		return
	}

	if existing == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Template not found", nil)
		return
	}

	// Prevent editing built-in templates
	if existing.IsBuiltIn {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "Cannot modify built-in templates", nil)
		return
	}

	// Update template
	if err := h.db.UpdateNewsletterTemplate(r.Context(), templateID, &req, hctx.UserID); err != nil {
		log.Error().Err(err).
			Str("template_id", templateID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to update newsletter template")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update template", err)
		return
	}

	// Get updated template (error logged but don't block the operation)
	updated, err := h.db.GetNewsletterTemplate(r.Context(), templateID)
	if err != nil {
		log.Warn().Err(err).Str("template_id", templateID).Msg("Failed to fetch updated template for response")
	}

	//nolint:errcheck // Audit log errors don't block the operation
	_ = h.auditNewsletter(r, hctx, models.NewsletterAuditActionUpdate, models.NewsletterResourceTemplate, templateID, existing.Name, nil)

	log.Info().
		Str("template_id", templateID).
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Msg("Newsletter template updated")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   updated,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterTemplateDelete deletes a newsletter template.
//
// Method: DELETE
// Path: /api/v1/newsletter/templates/{id}
//
// URL Parameters:
//   - id: Template ID
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) NewsletterTemplateDelete(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAdmin(w, r, "delete templates")
	if hctx == nil {
		return
	}

	templateID := chi.URLParam(r, "id")
	if templateID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Template ID is required", nil)
		return
	}

	start := time.Now()

	// Get existing template
	existing, err := h.db.GetNewsletterTemplate(r.Context(), templateID)
	if err != nil {
		log.Error().Err(err).
			Str("template_id", templateID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter template for deletion")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get template", err)
		return
	}

	if existing == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Template not found", nil)
		return
	}

	// Prevent deleting built-in templates
	if existing.IsBuiltIn {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "Cannot delete built-in templates", nil)
		return
	}

	// Delete template
	if err := h.db.DeleteNewsletterTemplate(r.Context(), templateID); err != nil {
		log.Error().Err(err).
			Str("template_id", templateID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to delete newsletter template")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete template", err)
		return
	}

	//nolint:errcheck // Audit log errors don't block the operation
	_ = h.auditNewsletter(r, hctx, models.NewsletterAuditActionDelete, models.NewsletterResourceTemplate, templateID, existing.Name, nil)

	log.Info().
		Str("template_id", templateID).
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Msg("Newsletter template deleted")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Template deleted successfully"},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterTemplatePreview previews a newsletter with sample data.
//
// Method: POST
// Path: /api/v1/newsletter/templates/preview
//
// Request Body: PreviewNewsletterRequest
//
// Response: PreviewNewsletterResponse
//
// Authentication: Required
// Authorization: Editor role or higher
func (h *Handler) NewsletterTemplatePreview(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireEditor(w, r, "preview templates")
	if hctx == nil {
		return
	}

	var req models.PreviewNewsletterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	if req.TemplateID == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Template ID is required", nil)
		return
	}

	start := time.Now()

	// Get template
	template, err := h.db.GetNewsletterTemplate(r.Context(), req.TemplateID)
	if err != nil {
		log.Error().Err(err).
			Str("template_id", req.TemplateID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter template for preview")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get template", err)
		return
	}

	if template == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Template not found", nil)
		return
	}

	// Get config and generate sample data
	config := resolveTemplateConfig(req.Config, template.DefaultConfig)
	sampleData := generateSampleContentData(template.Type, config)

	// Render template
	rendered, err := renderTemplatePreview(template, sampleData)
	if err != nil {
		respondError(w, http.StatusBadRequest, "RENDER_ERROR", err.Error(), nil)
		return
	}

	//nolint:errcheck // Audit log errors don't block the operation
	_ = h.auditNewsletter(r, hctx, models.NewsletterAuditActionPreview, models.NewsletterResourceTemplate, template.ID, template.Name, nil)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: models.PreviewNewsletterResponse{
			Subject:  rendered.Subject,
			BodyHTML: rendered.HTML,
			BodyText: rendered.Text,
			Data:     sampleData,
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// generateSampleContentData creates sample data for template preview.
func generateSampleContentData(newsletterType models.NewsletterType, config *models.TemplateConfig) *models.NewsletterContentData {
	now := time.Now()
	startDate := now.AddDate(0, 0, -config.TimeFrame)

	data := &models.NewsletterContentData{
		ServerName:       "Media Server",
		ServerURL:        "https://media.example.com",
		NewsletterURL:    "https://media.example.com/newsletter",
		UnsubscribeURL:   "https://media.example.com/unsubscribe",
		GeneratedAt:      now,
		DateRangeStart:   startDate,
		DateRangeEnd:     now,
		DateRangeDisplay: "Last " + strconv.Itoa(config.TimeFrame) + " days",
	}

	populateSampleMedia(data, config, now)
	populateSampleStats(data, config)
	populateHealthData(data, newsletterType, now)

	return data
}

// populateSampleMedia adds sample movies and shows to the newsletter data.
func populateSampleMedia(data *models.NewsletterContentData, config *models.TemplateConfig, now time.Time) {
	if config.IncludeMovies {
		data.NewMovies = generateSampleMovies(now)
	}

	if config.IncludeShows {
		data.NewShows = generateSampleShows(now)
	}

	if config.IncludeTopContent {
		data.TopMovies = generateTopMovies()
		data.TopShows = generateTopShows()
	}
}

// generateSampleMovies creates sample movie data.
func generateSampleMovies(now time.Time) []models.NewsletterMediaItem {
	addedAt := now.AddDate(0, 0, -2)
	return []models.NewsletterMediaItem{
		{
			RatingKey:     "1001",
			Title:         "The Matrix",
			Year:          1999,
			MediaType:     "movie",
			Summary:       "A computer hacker learns about the true nature of reality.",
			Genres:        []string{"Action", "Sci-Fi"},
			Duration:      136,
			ContentRating: "R",
			AddedAt:       &addedAt,
		},
		{
			RatingKey:     "1002",
			Title:         "Inception",
			Year:          2010,
			MediaType:     "movie",
			Summary:       "A thief who steals corporate secrets through dream-sharing technology.",
			Genres:        []string{"Action", "Sci-Fi", "Thriller"},
			Duration:      148,
			ContentRating: "PG-13",
			AddedAt:       &addedAt,
		},
	}
}

// generateSampleShows creates sample TV show data.
func generateSampleShows(now time.Time) []models.NewsletterShowItem {
	addedAt := now.AddDate(0, 0, -1)
	return []models.NewsletterShowItem{
		{
			RatingKey:        "2001",
			Title:            "Breaking Bad",
			Year:             2008,
			Summary:          "A high school chemistry teacher turned meth producer.",
			Genres:           []string{"Crime", "Drama", "Thriller"},
			ContentRating:    "TV-MA",
			NewEpisodesCount: 3,
			Seasons: []models.NewsletterSeasonItem{
				{
					SeasonNumber: 5,
					Episodes: []models.NewsletterEpisodeItem{
						{RatingKey: "2101", Title: "Blood Money", EpisodeNumber: 9, AddedAt: &addedAt},
						{RatingKey: "2102", Title: "Buried", EpisodeNumber: 10, AddedAt: &addedAt},
						{RatingKey: "2103", Title: "Confessions", EpisodeNumber: 11, AddedAt: &addedAt},
					},
				},
			},
		},
	}
}

// generateTopMovies creates sample top movie data.
func generateTopMovies() []models.NewsletterMediaItem {
	return []models.NewsletterMediaItem{
		{RatingKey: "3001", Title: "Pulp Fiction", Year: 1994, WatchCount: 45, WatchTime: 81.2},
		{RatingKey: "3002", Title: "The Shawshank Redemption", Year: 1994, WatchCount: 42, WatchTime: 79.8},
	}
}

// generateTopShows creates sample top show data.
func generateTopShows() []models.NewsletterShowItem {
	return []models.NewsletterShowItem{
		{RatingKey: "3101", Title: "The Office", Year: 2005, WatchCount: 234, WatchTime: 156.5},
		{RatingKey: "3102", Title: "Friends", Year: 1994, WatchCount: 189, WatchTime: 98.2},
	}
}

// populateSampleStats adds sample statistics to the newsletter data.
func populateSampleStats(data *models.NewsletterContentData, config *models.TemplateConfig) {
	if !config.IncludeStats {
		return
	}

	data.Stats = &models.NewsletterStats{
		TotalPlaybacks:      1547,
		TotalWatchTimeHours: 892.5,
		UniqueUsers:         42,
		UniqueContent:       234,
		MoviesWatched:       89,
		EpisodesWatched:     458,
		TopPlatforms: []models.PlatformStat{
			{Platform: "Roku", WatchCount: 523, WatchTime: 312.5, Percentage: 35.0},
			{Platform: "Web", WatchCount: 412, WatchTime: 245.2, Percentage: 27.5},
			{Platform: "Apple TV", WatchCount: 312, WatchTime: 198.3, Percentage: 20.0},
		},
		TopUsers: []models.UserStat{
			{UserID: "u1", Username: "alice", WatchCount: 156, WatchTime: 89.5},
			{UserID: "u2", Username: "bob", WatchCount: 134, WatchTime: 76.2},
			{UserID: "u3", Username: "carol", WatchCount: 98, WatchTime: 54.8},
		},
	}
}

// populateHealthData adds health monitoring data for server_health newsletters.
func populateHealthData(data *models.NewsletterContentData, newsletterType models.NewsletterType, now time.Time) {
	if newsletterType != models.NewsletterTypeServerHealth {
		return
	}

	syncTime := now.Add(-30 * time.Minute)
	data.Health = &models.NewsletterHealthData{
		ServerStatus:   "healthy",
		UptimePercent:  99.95,
		ActiveStreams:  3,
		TotalLibraries: 5,
		TotalContent:   15234,
		DatabaseSize:   "2.4 GB",
		LastSyncAt:     &syncTime,
		Warnings:       []string{},
	}
}

// ============================================================================
// Schedule Endpoints
// ============================================================================

// NewsletterScheduleList returns all newsletter schedules.
//
// Method: GET
// Path: /api/v1/newsletter/schedules
//
// Query Parameters:
//   - enabled: Filter by enabled status (true/false)
//   - template_id: Filter by template ID
//   - limit: Maximum results (default: 50)
//   - offset: Pagination offset
//
// Response: ListSchedulesResponse
//
// Authentication: Required
// Authorization: Viewer role or higher
func (h *Handler) NewsletterScheduleList(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return
	}

	start := time.Now()

	// Parse query parameters
	templateID := r.URL.Query().Get("template_id")
	limit, offset := parsePaginationParams(r, 50, 200)
	enabledFilter := parseBoolParam(r, "enabled")

	schedules, totalCount, err := h.db.ListNewsletterSchedules(r.Context(), templateID, enabledFilter, limit, offset)
	if err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Msg("Failed to list newsletter schedules")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list schedules", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: models.ListSchedulesResponse{
			Schedules:  schedules,
			TotalCount: totalCount,
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterScheduleCreate creates a new newsletter schedule.
//
// Method: POST
// Path: /api/v1/newsletter/schedules
//
// Request Body: CreateScheduleRequest
//
// Response: NewsletterSchedule
//
// Authentication: Required
// Authorization: Editor role or higher
func (h *Handler) NewsletterScheduleCreate(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireEditor(w, r, "create schedules")
	if hctx == nil {
		return
	}

	var req models.CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	// Validate request
	if err := validateScheduleCreateRequest(&req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Verify template exists
	if err := h.verifyTemplateExists(r.Context(), req.TemplateID); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	start := time.Now()

	// Create schedule
	schedule := buildScheduleFromRequest(&req, hctx.UserID)

	if err := h.db.CreateNewsletterSchedule(r.Context(), schedule); err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Str("user_id", hctx.UserID).
			Msg("Failed to create newsletter schedule")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create schedule", err)
		return
	}

	//nolint:errcheck // Audit log errors don't block the operation
	_ = h.auditNewsletter(r, hctx, models.NewsletterAuditActionCreate, models.NewsletterResourceSchedule, schedule.ID, schedule.Name, nil)

	log.Info().
		Str("schedule_id", schedule.ID).
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Str("name", schedule.Name).
		Str("cron", schedule.CronExpression).
		Msg("Newsletter schedule created")

	respondJSON(w, http.StatusCreated, &models.APIResponse{
		Status: "success",
		Data:   schedule,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterScheduleGet retrieves a specific newsletter schedule.
//
// Method: GET
// Path: /api/v1/newsletter/schedules/{id}
//
// URL Parameters:
//   - id: Schedule ID
//
// Response: NewsletterSchedule
//
// Authentication: Required
// Authorization: Viewer role or higher
func (h *Handler) NewsletterScheduleGet(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Schedule ID is required", nil)
		return
	}

	start := time.Now()

	schedule, err := h.db.GetNewsletterSchedule(r.Context(), scheduleID)
	if err != nil {
		log.Error().Err(err).
			Str("schedule_id", scheduleID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter schedule")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get schedule", err)
		return
	}

	if schedule == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Schedule not found", nil)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   schedule,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterScheduleUpdate updates a newsletter schedule.
//
// Method: PUT
// Path: /api/v1/newsletter/schedules/{id}
//
// URL Parameters:
//   - id: Schedule ID
//
// Request Body: UpdateScheduleRequest
//
// Response: NewsletterSchedule
//
// Authentication: Required
// Authorization: Editor role or higher
func (h *Handler) NewsletterScheduleUpdate(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireEditor(w, r, "update schedules")
	if hctx == nil {
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Schedule ID is required", nil)
		return
	}

	var req models.UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	start := time.Now()

	// Get existing schedule
	existing, err := h.db.GetNewsletterSchedule(r.Context(), scheduleID)
	if err != nil {
		log.Error().Err(err).
			Str("schedule_id", scheduleID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter schedule for update")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get schedule", err)
		return
	}

	if existing == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Schedule not found", nil)
		return
	}

	// Update schedule
	if err := h.db.UpdateNewsletterSchedule(r.Context(), scheduleID, &req, hctx.UserID); err != nil {
		log.Error().Err(err).
			Str("schedule_id", scheduleID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to update newsletter schedule")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update schedule", err)
		return
	}

	// Get updated schedule (error logged but don't block the operation)
	updated, err := h.db.GetNewsletterSchedule(r.Context(), scheduleID)
	if err != nil {
		log.Warn().Err(err).Str("schedule_id", scheduleID).Msg("Failed to fetch updated schedule for response")
	}

	// Audit the update
	action := determineScheduleAuditAction(&req)
	//nolint:errcheck // Audit log errors don't block the operation
	_ = h.auditNewsletter(r, hctx, action, models.NewsletterResourceSchedule, scheduleID, existing.Name, nil)

	log.Info().
		Str("schedule_id", scheduleID).
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Msg("Newsletter schedule updated")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   updated,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterScheduleDelete deletes a newsletter schedule.
//
// Method: DELETE
// Path: /api/v1/newsletter/schedules/{id}
//
// URL Parameters:
//   - id: Schedule ID
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) NewsletterScheduleDelete(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAdmin(w, r, "delete schedules")
	if hctx == nil {
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Schedule ID is required", nil)
		return
	}

	start := time.Now()

	// Get existing schedule
	existing, err := h.db.GetNewsletterSchedule(r.Context(), scheduleID)
	if err != nil {
		log.Error().Err(err).
			Str("schedule_id", scheduleID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter schedule for deletion")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get schedule", err)
		return
	}

	if existing == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Schedule not found", nil)
		return
	}

	// Delete schedule
	if err := h.db.DeleteNewsletterSchedule(r.Context(), scheduleID); err != nil {
		log.Error().Err(err).
			Str("schedule_id", scheduleID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to delete newsletter schedule")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete schedule", err)
		return
	}

	//nolint:errcheck // Audit log errors don't block the operation
	_ = h.auditNewsletter(r, hctx, models.NewsletterAuditActionDelete, models.NewsletterResourceSchedule, scheduleID, existing.Name, nil)

	log.Info().
		Str("schedule_id", scheduleID).
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Msg("Newsletter schedule deleted")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Schedule deleted successfully"},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterScheduleTrigger triggers immediate delivery of a schedule.
//
// Method: POST
// Path: /api/v1/newsletter/schedules/{id}/trigger
//
// URL Parameters:
//   - id: Schedule ID
//
// Response: NewsletterDelivery
//
// Authentication: Required
// Authorization: Editor role or higher
func (h *Handler) NewsletterScheduleTrigger(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireEditor(w, r, "trigger newsletters")
	if hctx == nil {
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Schedule ID is required", nil)
		return
	}

	start := time.Now()

	// Get schedule
	schedule, err := h.db.GetNewsletterSchedule(r.Context(), scheduleID)
	if err != nil {
		log.Error().Err(err).
			Str("schedule_id", scheduleID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter schedule for trigger")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get schedule", err)
		return
	}

	if schedule == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Schedule not found", nil)
		return
	}

	// Create delivery record
	delivery := &models.NewsletterDelivery{
		ScheduleID:        scheduleID,
		ScheduleName:      schedule.Name,
		TemplateID:        schedule.TemplateID,
		Status:            models.DeliveryStatusPending,
		RecipientsTotal:   len(schedule.Recipients) * len(schedule.Channels),
		StartedAt:         time.Now(),
		TriggeredBy:       "manual",
		TriggeredByUserID: hctx.UserID,
	}

	if err := h.db.CreateNewsletterDelivery(r.Context(), delivery); err != nil {
		log.Error().Err(err).
			Str("schedule_id", scheduleID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to create newsletter delivery")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create delivery", err)
		return
	}

	//nolint:errcheck // Audit log errors don't block the operation
	_ = h.auditNewsletter(r, hctx, models.NewsletterAuditActionSend, models.NewsletterResourceSchedule, scheduleID, schedule.Name, map[string]interface{}{
		"delivery_id": delivery.ID,
		"recipients":  len(schedule.Recipients),
		"channels":    len(schedule.Channels),
	})

	log.Info().
		Str("schedule_id", scheduleID).
		Str("delivery_id", delivery.ID).
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Msg("Newsletter delivery triggered manually")

	// Note: Actual delivery would be handled asynchronously by a background worker
	// For now, return the pending delivery record

	respondJSON(w, http.StatusAccepted, &models.APIResponse{
		Status: "success",
		Data:   delivery,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// ============================================================================
// Delivery Endpoints
// ============================================================================

// NewsletterDeliveryList returns delivery history.
//
// Method: GET
// Path: /api/v1/newsletter/deliveries
//
// Query Parameters:
//   - schedule_id: Filter by schedule ID
//   - status: Filter by status
//   - limit: Maximum results (default: 50)
//   - offset: Pagination offset
//
// Response: ListDeliveriesResponse
//
// Authentication: Required
// Authorization: Viewer role or higher
func (h *Handler) NewsletterDeliveryList(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return
	}

	start := time.Now()

	// Parse query parameters
	scheduleID := r.URL.Query().Get("schedule_id")
	status := r.URL.Query().Get("status")
	limit, offset := parsePaginationParams(r, 50, 200)

	deliveries, totalCount, err := h.db.ListNewsletterDeliveries(r.Context(), scheduleID, status, limit, offset)
	if err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Msg("Failed to list newsletter deliveries")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list deliveries", err)
		return
	}

	hasMore := offset+len(deliveries) < totalCount
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: models.ListDeliveriesResponse{
			Deliveries: deliveries,
			Pagination: models.PaginationInfo{
				Limit:      limit,
				HasMore:    hasMore,
				TotalCount: &totalCount,
			},
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterDeliveryGet retrieves a specific delivery record.
//
// Method: GET
// Path: /api/v1/newsletter/deliveries/{id}
//
// URL Parameters:
//   - id: Delivery ID
//
// Response: NewsletterDelivery
//
// Authentication: Required
// Authorization: Viewer role or higher
func (h *Handler) NewsletterDeliveryGet(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return
	}

	deliveryID := chi.URLParam(r, "id")
	if deliveryID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Delivery ID is required", nil)
		return
	}

	start := time.Now()

	delivery, err := h.db.GetNewsletterDelivery(r.Context(), deliveryID)
	if err != nil {
		log.Error().Err(err).
			Str("delivery_id", deliveryID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter delivery")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get delivery", err)
		return
	}

	if delivery == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Delivery not found", nil)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   delivery,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// ============================================================================
// Stats and Audit Endpoints
// ============================================================================

// NewsletterStats returns aggregated newsletter statistics.
//
// Method: GET
// Path: /api/v1/newsletter/stats
//
// Response: NewsletterStatsResponse
//
// Authentication: Required
// Authorization: Viewer role or higher
func (h *Handler) NewsletterStats(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return
	}

	start := time.Now()

	stats, err := h.db.GetNewsletterStats(r.Context())
	if err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter stats")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get statistics", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   stats,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterAuditLog returns the newsletter audit log.
//
// Method: GET
// Path: /api/v1/newsletter/audit
//
// Query Parameters:
//   - resource_type: Filter by resource type (template, schedule, delivery)
//   - resource_id: Filter by resource ID
//   - actor_id: Filter by actor ID
//   - action: Filter by action
//   - limit: Maximum results (default: 100)
//   - offset: Pagination offset
//
// Response: Array of NewsletterAuditEntry
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) NewsletterAuditLog(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAdmin(w, r, "view audit log")
	if hctx == nil {
		return
	}

	start := time.Now()

	// Parse query parameters
	resourceType := r.URL.Query().Get("resource_type")
	resourceID := r.URL.Query().Get("resource_id")
	actorID := r.URL.Query().Get("actor_id")
	action := r.URL.Query().Get("action")
	limit, offset := parsePaginationParams(r, 100, 1000)

	entries, totalCount, err := h.db.ListNewsletterAuditLog(r.Context(), resourceType, resourceID, actorID, action, limit, offset)
	if err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter audit log")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get audit log", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"entries":     entries,
			"total_count": totalCount,
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// ============================================================================
// User Preferences Endpoints
// ============================================================================

// NewsletterUserPreferencesGet returns the current user's newsletter preferences.
//
// Method: GET
// Path: /api/v1/user/newsletter/preferences
//
// Response: NewsletterUserPreferences
//
// Authentication: Required
func (h *Handler) NewsletterUserPreferencesGet(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return
	}

	start := time.Now()

	prefs, err := h.db.GetNewsletterUserPreferences(r.Context(), hctx.UserID)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", hctx.UserID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get newsletter preferences")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get preferences", err)
		return
	}

	// Return empty preferences if none exist
	if prefs == nil {
		prefs = &models.NewsletterUserPreferences{
			UserID:   hctx.UserID,
			Username: hctx.Username,
		}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   prefs,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterUserPreferencesUpdate updates the current user's newsletter preferences.
//
// Method: PUT
// Path: /api/v1/user/newsletter/preferences
//
// Request Body: NewsletterUserPreferences
//
// Response: NewsletterUserPreferences
//
// Authentication: Required
func (h *Handler) NewsletterUserPreferencesUpdate(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return
	}

	var prefs models.NewsletterUserPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	// Ensure user can only update their own preferences
	prefs.UserID = hctx.UserID
	prefs.Username = hctx.Username
	prefs.UpdatedAt = time.Now()

	start := time.Now()

	if err := h.db.UpsertNewsletterUserPreferences(r.Context(), &prefs); err != nil {
		log.Error().Err(err).
			Str("user_id", hctx.UserID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to update newsletter preferences")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update preferences", err)
		return
	}

	log.Info().
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Bool("global_opt_out", prefs.GlobalOptOut).
		Msg("Newsletter preferences updated")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   prefs,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// NewsletterUnsubscribe opts out the user from all newsletters.
//
// Method: POST
// Path: /api/v1/user/newsletter/unsubscribe
//
// Response: Success message
//
// Authentication: Required
func (h *Handler) NewsletterUnsubscribe(w http.ResponseWriter, r *http.Request) {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return
	}

	start := time.Now()
	now := time.Now()

	// Get existing preferences or create new
	prefs, err := h.db.GetNewsletterUserPreferences(r.Context(), hctx.UserID)
	if err != nil {
		prefs = &models.NewsletterUserPreferences{
			UserID:   hctx.UserID,
			Username: hctx.Username,
		}
	}

	// Set global opt-out
	prefs.GlobalOptOut = true
	prefs.GlobalOptOutAt = &now
	prefs.UpdatedAt = now

	if err := h.db.UpsertNewsletterUserPreferences(r.Context(), prefs); err != nil {
		log.Error().Err(err).
			Str("user_id", hctx.UserID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to unsubscribe from newsletters")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to unsubscribe", err)
		return
	}

	//nolint:errcheck // Audit log errors don't block the operation
	_ = h.auditNewsletter(r, hctx, models.NewsletterAuditActionOptOut, models.NewsletterResourcePreferences, hctx.UserID, hctx.Username, nil)

	log.Info().
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Msg("User unsubscribed from all newsletters")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Successfully unsubscribed from all newsletters"},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// ============================================================================
// Newsletter Handler Helpers
// ============================================================================

// requireAuth returns HandlerContext if authenticated, or sends error response and returns nil.
// This reduces repetitive authentication check boilerplate.
func (h *Handler) requireAuth(w http.ResponseWriter, r *http.Request) *HandlerContext {
	hctx := GetHandlerContext(r)
	if !hctx.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return nil
	}
	return hctx
}

// requireEditor returns HandlerContext if user has editor or admin role.
// Sends error response and returns nil if not authorized.
func (h *Handler) requireEditor(w http.ResponseWriter, r *http.Request, action string) *HandlerContext {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return nil
	}
	if !hctx.HasRole("editor") && !hctx.HasRole("admin") {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "Editor role required to "+action, nil)
		return nil
	}
	return hctx
}

// requireAdmin returns HandlerContext if user has admin role.
// Sends error response and returns nil if not authorized.
func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request, action string) *HandlerContext {
	hctx := h.requireAuth(w, r)
	if hctx == nil {
		return nil
	}
	if !hctx.HasRole("admin") {
		respondError(w, http.StatusForbidden, "FORBIDDEN", "Admin role required to "+action, nil)
		return nil
	}
	return hctx
}

// auditNewsletter creates an audit log entry for newsletter operations.
// Returns error if audit logging fails, but callers typically ignore this.
func (h *Handler) auditNewsletter(r *http.Request, hctx *HandlerContext, action, resourceType, resourceID, resourceName string, details map[string]interface{}) error {
	entry := &models.NewsletterAuditEntry{
		Timestamp:     time.Now(),
		ActorID:       hctx.UserID,
		ActorUsername: hctx.Username,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		ResourceName:  resourceName,
		Details:       details,
		IPAddress:     getClientIP(r),
		UserAgent:     r.UserAgent(),
	}
	return h.db.CreateNewsletterAuditEntry(r.Context(), entry)
}

// parsePaginationParams extracts limit and offset from query parameters.
func parsePaginationParams(r *http.Request, defaultLimit, maxLimit int) (limit, offset int) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit = defaultLimit
	offset = 0
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= maxLimit {
		limit = l
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}
	return limit, offset
}

// parseBoolParam parses a boolean query parameter, returning nil if not present.
func parseBoolParam(r *http.Request, name string) *bool {
	val := r.URL.Query().Get(name)
	if val == "" {
		return nil
	}
	result := val == "true"
	return &result
}

// ============================================================================
// Validation and Builder Helpers
// ============================================================================

// validateTemplateCreateRequest validates a template creation request.
func validateTemplateCreateRequest(req *models.CreateTemplateRequest) error {
	if req.Name == "" {
		return ErrValidation("Template name is required")
	}
	if req.Subject == "" {
		return ErrValidation("Subject is required")
	}
	if req.BodyHTML == "" {
		return ErrValidation("HTML body is required")
	}
	if !models.IsValidNewsletterType(req.Type) {
		return ErrValidation("Invalid newsletter type")
	}
	return nil
}

// buildTemplateFromRequest constructs a NewsletterTemplate from a request.
func buildTemplateFromRequest(req *models.CreateTemplateRequest, userID string) *models.NewsletterTemplate {
	now := time.Now()
	return &models.NewsletterTemplate{
		Name:          req.Name,
		Description:   req.Description,
		Type:          req.Type,
		Subject:       req.Subject,
		BodyHTML:      req.BodyHTML,
		BodyText:      req.BodyText,
		DefaultConfig: req.DefaultConfig,
		Version:       1,
		IsBuiltIn:     false,
		IsActive:      true,
		CreatedBy:     userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// validateScheduleCreateRequest validates a schedule creation request.
func validateScheduleCreateRequest(req *models.CreateScheduleRequest) error {
	if req.Name == "" {
		return ErrValidation("Schedule name is required")
	}
	if req.TemplateID == "" {
		return ErrValidation("Template ID is required")
	}
	if req.CronExpression == "" {
		return ErrValidation("Cron expression is required")
	}
	if req.Timezone == "" {
		return ErrValidation("Timezone is required")
	}
	if len(req.Recipients) == 0 {
		return ErrValidation("At least one recipient is required")
	}
	if len(req.Channels) == 0 {
		return ErrValidation("At least one channel is required")
	}

	// Validate channels
	for _, channel := range req.Channels {
		if !models.IsValidDeliveryChannel(channel) {
			return ErrValidation("Invalid delivery channel: " + string(channel))
		}
	}

	return nil
}

// buildScheduleFromRequest constructs a NewsletterSchedule from a request.
func buildScheduleFromRequest(req *models.CreateScheduleRequest, userID string) *models.NewsletterSchedule {
	now := time.Now()
	return &models.NewsletterSchedule{
		Name:           req.Name,
		Description:    req.Description,
		TemplateID:     req.TemplateID,
		Recipients:     req.Recipients,
		CronExpression: req.CronExpression,
		Timezone:       req.Timezone,
		Config:         req.Config,
		Channels:       req.Channels,
		ChannelConfigs: req.ChannelConfigs,
		IsEnabled:      req.IsEnabled,
		CreatedBy:      userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// verifyTemplateExists checks if a template exists in the database.
func (h *Handler) verifyTemplateExists(ctx context.Context, templateID string) error {
	template, err := h.db.GetNewsletterTemplate(ctx, templateID)
	if err != nil || template == nil {
		return ErrValidation("Template not found")
	}
	return nil
}

// determineScheduleAuditAction determines the appropriate audit action for a schedule update.
func determineScheduleAuditAction(req *models.UpdateScheduleRequest) string {
	if req.IsEnabled == nil {
		return models.NewsletterAuditActionUpdate
	}
	if *req.IsEnabled {
		return models.NewsletterAuditActionEnable
	}
	return models.NewsletterAuditActionDisable
}

// resolveTemplateConfig resolves the template config from request, template default, or system default.
func resolveTemplateConfig(requestConfig, templateDefaultConfig *models.TemplateConfig) *models.TemplateConfig {
	if requestConfig != nil {
		return requestConfig
	}
	if templateDefaultConfig != nil {
		return templateDefaultConfig
	}
	return &models.TemplateConfig{
		TimeFrame:     7,
		TimeFrameUnit: models.TimeFrameUnitDays,
		MaxItems:      10,
		IncludeMovies: true,
		IncludeShows:  true,
	}
}

// renderedTemplate holds the results of template rendering.
type renderedTemplate struct {
	Subject string
	HTML    string
	Text    string
}

// renderTemplatePreview renders a template with sample data.
func renderTemplatePreview(template *models.NewsletterTemplate, data *models.NewsletterContentData) (*renderedTemplate, error) {
	engine := newsletter.NewTemplateEngine()

	renderedSubject, err := engine.RenderSubject(template.Subject, data)
	if err != nil {
		return nil, ErrValidation("Failed to render subject: " + err.Error())
	}

	renderedHTML, err := engine.RenderHTML(template.BodyHTML, data)
	if err != nil {
		return nil, ErrValidation("Failed to render HTML: " + err.Error())
	}

	renderedText := ""
	if template.BodyText != "" {
		renderedText, err = engine.RenderText(template.BodyText, data)
		if err != nil {
			return nil, ErrValidation("Failed to render text: " + err.Error())
		}
	}

	return &renderedTemplate{
		Subject: renderedSubject,
		HTML:    renderedHTML,
		Text:    renderedText,
	}, nil
}

// ErrValidation creates a validation error with the given message.
func ErrValidation(msg string) error {
	return &validationError{message: msg}
}

type validationError struct {
	message string
}

func (e *validationError) Error() string {
	return e.message
}
