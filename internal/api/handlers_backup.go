// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/backup"
	"github.com/tomtom215/cartographus/internal/models"
)

// BackupManager is the interface for backup operations
type BackupManager interface {
	CreateBackup(ctx context.Context, backupType backup.BackupType, notes string) (*backup.Backup, error)
	ListBackups(opts backup.BackupListOptions) ([]*backup.Backup, error)
	GetBackup(backupID string) (*backup.Backup, error)
	DeleteBackup(backupID string) error
	ValidateBackup(backupID string) (*backup.ValidationResult, error)
	RestoreFromBackup(ctx context.Context, backupID string, opts backup.RestoreOptions) (*backup.RestoreResult, error)
	DownloadBackup(backupID string) (io.ReadCloser, *backup.Backup, error)
	ImportBackup(ctx context.Context, reader io.Reader, filename string) (*backup.Backup, error)
	GetStats() (*backup.BackupStats, error)
	ApplyRetentionPolicy(ctx context.Context) error
	GetRetentionPreview() (*backup.RetentionPreview, error)
	GetRetentionPolicy() backup.RetentionPolicy
	SetRetentionPolicy(policy backup.RetentionPolicy) error
	CleanupCorruptedBackups(ctx context.Context) (int, error)
	// Schedule configuration
	GetScheduleConfig() backup.ScheduleConfig
	SetScheduleConfig(ctx context.Context, schedule backup.ScheduleConfig) error
	TriggerScheduledBackup(ctx context.Context) (*backup.Backup, error)
}

// Helper functions to reduce cognitive complexity

// checkHTTPMethod validates the HTTP method and responds with error if invalid
func checkHTTPMethod(w http.ResponseWriter, r *http.Request, expectedMethod string) bool {
	if r.Method != expectedMethod {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", fmt.Sprintf("Only %s method is allowed", expectedMethod), nil)
		return false
	}
	return true
}

// checkBackupManagerAvailable checks if backup manager is available
func (h *Handler) checkBackupManagerAvailable(w http.ResponseWriter) bool {
	if h.backupManager == nil {
		respondError(w, http.StatusServiceUnavailable, "BACKUP_DISABLED", "Backup functionality is not enabled", nil)
		return false
	}
	return true
}

// getBackupIDFromQuery extracts and validates backup ID from query parameters
func getBackupIDFromQuery(w http.ResponseWriter, r *http.Request) (string, bool) {
	backupID := r.URL.Query().Get("id")
	if backupID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Backup ID is required", nil)
		return "", false
	}
	return backupID, true
}

// parseBackupType converts string to backup.BackupType
func parseBackupType(typeStr string) backup.BackupType {
	typeMap := map[string]backup.BackupType{
		"database": backup.TypeDatabase,
		"config":   backup.TypeConfig,
		"full":     backup.TypeFull,
		"":         backup.TypeFull,
	}

	if t, ok := typeMap[typeStr]; ok {
		return t
	}
	return backup.TypeFull
}

// respondBackupSuccess sends a successful backup response
func respondBackupSuccess(w http.ResponseWriter, statusCode int, data interface{}) {
	respondJSON(w, statusCode, &models.APIResponse{
		Status: "success",
		Data:   data,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// parseListOptions extracts and parses query parameters for backup listing
func parseListOptions(r *http.Request) backup.BackupListOptions {
	opts := backup.BackupListOptions{
		Limit:    100,
		Offset:   0,
		SortDesc: true,
	}

	query := r.URL.Query()

	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			opts.Limit = l
		}
	}

	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			opts.Offset = o
		}
	}

	if query.Get("sort") == "asc" {
		opts.SortDesc = false
	}

	if typeFilter := query.Get("type"); typeFilter != "" {
		t := backup.BackupType(typeFilter)
		opts.Type = &t
	}

	if statusFilter := query.Get("status"); statusFilter != "" {
		s := backup.BackupStatus(statusFilter)
		opts.Status = &s
	}

	return opts
}

// CreateBackupRequest is the request body for creating a backup
type CreateBackupRequest struct {
	Type  string `json:"type"`
	Notes string `json:"notes"`
}

// RestoreBackupRequest is the request body for restoring a backup
type RestoreBackupRequest struct {
	ValidateOnly           bool `json:"validate_only"`
	CreatePreRestoreBackup bool `json:"create_pre_restore_backup"`
	RestoreDatabase        bool `json:"restore_database"`
	RestoreConfig          bool `json:"restore_config"`
	ForceRestore           bool `json:"force_restore"`
	VerifyAfterRestore     bool `json:"verify_after_restore"`
}

// SetRetentionPolicyRequest is the request body for setting retention policy
type SetRetentionPolicyRequest struct {
	MinCount             int `json:"min_count"`
	MaxCount             int `json:"max_count"`
	MaxAgeDays           int `json:"max_age_days"`
	KeepRecentHours      int `json:"keep_recent_hours"`
	KeepDailyForDays     int `json:"keep_daily_for_days"`
	KeepWeeklyForWeeks   int `json:"keep_weekly_for_weeks"`
	KeepMonthlyForMonths int `json:"keep_monthly_for_months"`
}

// SetScheduleConfigRequest is the request body for setting backup schedule
type SetScheduleConfigRequest struct {
	Enabled       bool   `json:"enabled"`
	IntervalHours int    `json:"interval_hours" validate:"min=1,max=720"`
	PreferredHour int    `json:"preferred_hour" validate:"min=0,max=23"`
	BackupType    string `json:"backup_type" validate:"omitempty,oneof=full database config"`
	PreSyncBackup bool   `json:"pre_sync_backup"`
}

// HandleCreateBackup creates a new backup
// POST /api/v1/backup
func (h *Handler) HandleCreateBackup(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodPost) || !h.checkBackupManagerAvailable(w) {
		return
	}

	var req CreateBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Type = "full" // Default to full backup if no body provided
	}

	// Use validator for struct validation (validates type oneof and notes max length)
	validationReq := CreateBackupRequestValidation(req)
	if apiErr := validateRequest(&validationReq); apiErr != nil {
		respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
		return
	}

	// Parse backup type (already validated by validator)
	backupType := parseBackupType(req.Type)

	// Create backup
	b, err := h.backupManager.CreateBackup(r.Context(), backupType, req.Notes)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "BACKUP_FAILED", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusCreated, b)
}

// HandleListBackups lists all backups with optional filtering
// GET /api/v1/backups
func (h *Handler) HandleListBackups(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodGet) || !h.checkBackupManagerAvailable(w) {
		return
	}

	// Parse query parameters
	opts := parseListOptions(r)

	// List backups
	backups, err := h.backupManager.ListBackups(opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusOK, map[string]interface{}{
		"backups": backups,
		"count":   len(backups),
		"limit":   opts.Limit,
		"offset":  opts.Offset,
	})
}

// HandleGetBackup gets a specific backup by ID
// GET /api/v1/backups/{id}
func (h *Handler) HandleGetBackup(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodGet) || !h.checkBackupManagerAvailable(w) {
		return
	}

	backupID, ok := getBackupIDFromQuery(w, r)
	if !ok {
		return
	}

	b, err := h.backupManager.GetBackup(backupID)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusOK, b)
}

// HandleDeleteBackup deletes a backup
// DELETE /api/v1/backups/{id}
func (h *Handler) HandleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodDelete) || !h.checkBackupManagerAvailable(w) {
		return
	}

	backupID, ok := getBackupIDFromQuery(w, r)
	if !ok {
		return
	}

	if err := h.backupManager.DeleteBackup(backupID); err != nil {
		respondError(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusOK, map[string]string{
		"message": "Backup deleted successfully",
	})
}

// HandleValidateBackup validates a backup's integrity
// GET /api/v1/backups/{id}/validate
func (h *Handler) HandleValidateBackup(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodGet) || !h.checkBackupManagerAvailable(w) {
		return
	}

	backupID, ok := getBackupIDFromQuery(w, r)
	if !ok {
		return
	}

	result, err := h.backupManager.ValidateBackup(backupID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "VALIDATION_FAILED", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusOK, result)
}

// HandleRestoreBackup restores from a backup
// POST /api/v1/backups/{id}/restore
func (h *Handler) HandleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodPost) || !h.checkBackupManagerAvailable(w) {
		return
	}

	backupID, ok := getBackupIDFromQuery(w, r)
	if !ok {
		return
	}

	var req RestoreBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use default options if no body provided
		req.CreatePreRestoreBackup = true
		req.VerifyAfterRestore = true
	}

	opts := backup.RestoreOptions{
		ValidateOnly:           req.ValidateOnly,
		CreatePreRestoreBackup: req.CreatePreRestoreBackup,
		RestoreDatabase:        req.RestoreDatabase,
		RestoreConfig:          req.RestoreConfig,
		ForceRestore:           req.ForceRestore,
		VerifyAfterRestore:     req.VerifyAfterRestore,
	}

	result, err := h.backupManager.RestoreFromBackup(r.Context(), backupID, opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "RESTORE_FAILED", err.Error(), err)
		return
	}

	statusCode := http.StatusOK
	if !result.Success {
		statusCode = http.StatusInternalServerError
	}

	respondBackupSuccess(w, statusCode, result)
}

// HandleDownloadBackup downloads a backup file
// GET /api/v1/backups/{id}/download
func (h *Handler) HandleDownloadBackup(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodGet) || !h.checkBackupManagerAvailable(w) {
		return
	}

	backupID, ok := getBackupIDFromQuery(w, r)
	if !ok {
		return
	}

	reader, b, err := h.backupManager.DownloadBackup(backupID)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), err)
		return
	}
	defer reader.Close()

	// Set headers for file download
	setDownloadHeaders(w, b)

	// Stream the file to response
	//nolint:errcheck // Error is intentionally ignored as headers are already sent and we can't return an error response
	io.Copy(w, reader)
}

// setDownloadHeaders sets HTTP headers for backup file download
func setDownloadHeaders(w http.ResponseWriter, b *backup.Backup) {
	filename := fmt.Sprintf("cartographus-backup-%s.tar.gz", b.CreatedAt.Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Length", strconv.FormatInt(b.FileSize, 10))
	w.Header().Set("X-Backup-ID", b.ID)
	w.Header().Set("X-Backup-Type", string(b.Type))
	w.Header().Set("X-Backup-Checksum", b.Checksum)
}

// HandleUploadBackup uploads and imports a backup file
// POST /api/v1/backups/upload
func (h *Handler) HandleUploadBackup(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodPost) || !h.checkBackupManagerAvailable(w) {
		return
	}

	// Parse multipart form (max 500MB)
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "PARSE_ERROR", "Failed to parse upload: "+err.Error(), err)
		return
	}

	file, header, err := r.FormFile("backup")
	if err != nil {
		respondError(w, http.StatusBadRequest, "FILE_ERROR", "No backup file provided", err)
		return
	}
	defer file.Close()

	b, err := h.backupManager.ImportBackup(r.Context(), file, header.Filename)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "IMPORT_FAILED", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusCreated, b)
}

// HandleGetBackupStats gets backup statistics
// GET /api/v1/backup/stats
func (h *Handler) HandleGetBackupStats(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodGet) || !h.checkBackupManagerAvailable(w) {
		return
	}

	stats, err := h.backupManager.GetStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STATS_FAILED", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusOK, stats)
}

// HandleGetRetentionPolicy gets the current retention policy
// GET /api/v1/backup/retention
func (h *Handler) HandleGetRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodGet) || !h.checkBackupManagerAvailable(w) {
		return
	}

	policy := h.backupManager.GetRetentionPolicy()
	respondBackupSuccess(w, http.StatusOK, policy)
}

// HandleSetRetentionPolicy sets the retention policy
// PUT /api/v1/backup/retention
func (h *Handler) HandleSetRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodPut) || !h.checkBackupManagerAvailable(w) {
		return
	}

	var req SetRetentionPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	// Use validator for struct validation (validates all fields >= 0)
	validationReq := SetRetentionPolicyRequestValidation(req)
	if apiErr := validateRequest(&validationReq); apiErr != nil {
		respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
		return
	}

	policy := backup.RetentionPolicy{
		MinCount:             req.MinCount,
		MaxCount:             req.MaxCount,
		MaxAgeDays:           req.MaxAgeDays,
		KeepRecentHours:      req.KeepRecentHours,
		KeepDailyForDays:     req.KeepDailyForDays,
		KeepWeeklyForWeeks:   req.KeepWeeklyForWeeks,
		KeepMonthlyForMonths: req.KeepMonthlyForMonths,
	}

	if err := h.backupManager.SetRetentionPolicy(policy); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_POLICY", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusOK, policy)
}

// HandleRetentionPreview shows what would be deleted by retention policy
// GET /api/v1/backup/retention/preview
func (h *Handler) HandleRetentionPreview(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodGet) || !h.checkBackupManagerAvailable(w) {
		return
	}

	preview, err := h.backupManager.GetRetentionPreview()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PREVIEW_FAILED", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusOK, preview)
}

// HandleApplyRetention manually applies retention policy
// POST /api/v1/backup/retention/apply
func (h *Handler) HandleApplyRetention(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodPost) || !h.checkBackupManagerAvailable(w) {
		return
	}

	// Get preview before applying (used for reporting)
	preview, previewErr := h.backupManager.GetRetentionPreview()

	if err := h.backupManager.ApplyRetentionPolicy(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "APPLY_FAILED", err.Error(), err)
		return
	}

	// Build response data with preview info if available
	responseData := buildRetentionResponseData(preview, previewErr)
	respondBackupSuccess(w, http.StatusOK, responseData)
}

// buildRetentionResponseData creates response data for retention policy application
func buildRetentionResponseData(preview *backup.RetentionPreview, previewErr error) map[string]interface{} {
	responseData := map[string]interface{}{
		"message": "Retention policy applied successfully",
	}

	if previewErr == nil && preview != nil {
		responseData["message"] = fmt.Sprintf("Retention policy applied. Deleted %d backups.", preview.DeletedCount)
		responseData["deleted_count"] = preview.DeletedCount
		responseData["deleted_size"] = preview.TotalDeletedSize
	}

	return responseData
}

// HandleCleanupCorrupted cleans up corrupted backups
// POST /api/v1/backup/cleanup
func (h *Handler) HandleCleanupCorrupted(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodPost) || !h.checkBackupManagerAvailable(w) {
		return
	}

	count, err := h.backupManager.CleanupCorruptedBackups(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "CLEANUP_FAILED", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusOK, map[string]interface{}{
		"message":       fmt.Sprintf("Cleaned up %d corrupted backups", count),
		"cleaned_count": count,
	})
}

// HandleQuickBackup creates a quick backup with default settings
// This is a simplified endpoint for easy backup creation
// POST /api/v1/backup/quick
func (h *Handler) HandleQuickBackup(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodPost) || !h.checkBackupManagerAvailable(w) {
		return
	}

	// Create a quick full backup with automatic timestamp note
	notes := fmt.Sprintf("Quick backup created at %s", time.Now().Format(time.RFC3339))
	b, err := h.backupManager.CreateBackup(r.Context(), backup.TypeFull, notes)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "BACKUP_FAILED", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusCreated, b)
}

// HandleGetScheduleConfig gets the current backup schedule configuration
// GET /api/v1/backup/schedule
func (h *Handler) HandleGetScheduleConfig(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodGet) || !h.checkBackupManagerAvailable(w) {
		return
	}

	schedule := h.backupManager.GetScheduleConfig()
	responseData := buildScheduleResponseData(schedule)
	respondBackupSuccess(w, http.StatusOK, responseData)
}

// buildScheduleResponseData converts ScheduleConfig to response format
func buildScheduleResponseData(schedule backup.ScheduleConfig) map[string]interface{} {
	return map[string]interface{}{
		"enabled":         schedule.Enabled,
		"interval_hours":  int(schedule.Interval.Hours()),
		"preferred_hour":  schedule.PreferredHour,
		"backup_type":     string(schedule.BackupType),
		"pre_sync_backup": schedule.PreSyncBackup,
	}
}

// HandleSetScheduleConfig updates the backup schedule configuration
// PUT /api/v1/backup/schedule
func (h *Handler) HandleSetScheduleConfig(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodPut) || !h.checkBackupManagerAvailable(w) {
		return
	}

	var req SetScheduleConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	// Use validator for struct validation
	validationReq := SetScheduleConfigRequestValidation(req)
	if apiErr := validateRequest(&validationReq); apiErr != nil {
		respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
		return
	}

	schedule := backup.ScheduleConfig{
		Enabled:       req.Enabled,
		Interval:      time.Duration(req.IntervalHours) * time.Hour,
		PreferredHour: req.PreferredHour,
		BackupType:    parseBackupType(req.BackupType),
		PreSyncBackup: req.PreSyncBackup,
	}

	if err := h.backupManager.SetScheduleConfig(r.Context(), schedule); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_SCHEDULE", err.Error(), err)
		return
	}

	// Return the updated schedule
	responseData := map[string]interface{}{
		"enabled":         schedule.Enabled,
		"interval_hours":  req.IntervalHours,
		"preferred_hour":  schedule.PreferredHour,
		"backup_type":     string(schedule.BackupType),
		"pre_sync_backup": schedule.PreSyncBackup,
	}

	respondBackupSuccess(w, http.StatusOK, responseData)
}

// HandleTriggerScheduledBackup triggers a backup using the scheduled backup settings
// POST /api/v1/backup/schedule/trigger
func (h *Handler) HandleTriggerScheduledBackup(w http.ResponseWriter, r *http.Request) {
	if !checkHTTPMethod(w, r, http.MethodPost) || !h.checkBackupManagerAvailable(w) {
		return
	}

	b, err := h.backupManager.TriggerScheduledBackup(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "BACKUP_FAILED", err.Error(), err)
		return
	}

	respondBackupSuccess(w, http.StatusCreated, b)
}
