// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides database operations for the Cartographus application.
//
// newsletter.go - Newsletter Generator Database Operations
//
// This file contains CRUD operations for the Newsletter Generator system:
//   - Template management (create, read, update, delete)
//   - Schedule management (create, read, update, delete, enable/disable)
//   - Delivery history (create, read, query)
//   - User preferences (create, read, update)
//   - Audit logging (append)
//   - Statistics and analytics
//
// All operations use parameterized queries to prevent SQL injection.
// JSON columns are handled with proper marshaling/unmarshaling.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/models"
)

// ============================================================================
// Newsletter Templates
// ============================================================================

// CreateNewsletterTemplate creates a new newsletter template.
func (db *DB) CreateNewsletterTemplate(ctx context.Context, template *models.NewsletterTemplate) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Generate ID if not provided
	if template.ID == "" {
		template.ID = uuid.New().String()
	}

	// Marshal JSON fields
	variablesJSON, err := json.Marshal(template.Variables)
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %w", err)
	}

	var defaultConfigJSON []byte
	if template.DefaultConfig != nil {
		defaultConfigJSON, err = json.Marshal(template.DefaultConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal default_config: %w", err)
		}
	}

	query := `
		INSERT INTO newsletter_templates (
			id, name, description, type, subject, body_html, body_text,
			variables, default_config, version, is_built_in, is_active,
			created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	template.CreatedAt = now
	template.UpdatedAt = now
	if template.Version == 0 {
		template.Version = 1
	}

	_, err = db.conn.ExecContext(ctx, query,
		template.ID,
		template.Name,
		template.Description,
		string(template.Type),
		template.Subject,
		template.BodyHTML,
		template.BodyText,
		string(variablesJSON),
		nullableJSON(defaultConfigJSON),
		template.Version,
		template.IsBuiltIn,
		template.IsActive,
		template.CreatedBy,
		template.CreatedAt,
		template.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert newsletter template: %w", err)
	}

	return nil
}

// GetNewsletterTemplate retrieves a newsletter template by ID.
func (db *DB) GetNewsletterTemplate(ctx context.Context, id string) (*models.NewsletterTemplate, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		SELECT
			id, name, description, type, subject, body_html, body_text,
			variables::VARCHAR, default_config::VARCHAR, version, is_built_in, is_active,
			created_by, created_at, updated_by, updated_at
		FROM newsletter_templates
		WHERE id = ?
	`

	row := db.conn.QueryRowContext(ctx, query, id)
	return db.scanNewsletterTemplate(row)
}

// scanNewsletterTemplate scans a newsletter template from a row scanner.
func (db *DB) scanNewsletterTemplate(row *sql.Row) (*models.NewsletterTemplate, error) {
	var template models.NewsletterTemplate
	var description, bodyText, updatedBy sql.NullString
	var variablesJSON, defaultConfigJSON sql.NullString

	err := row.Scan(
		&template.ID,
		&template.Name,
		&description,
		&template.Type,
		&template.Subject,
		&template.BodyHTML,
		&bodyText,
		&variablesJSON,
		&defaultConfigJSON,
		&template.Version,
		&template.IsBuiltIn,
		&template.IsActive,
		&template.CreatedBy,
		&template.CreatedAt,
		&updatedBy,
		&template.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan newsletter template: %w", err)
	}

	// Set nullable string fields
	template.Description = description.String
	template.BodyText = bodyText.String
	template.UpdatedBy = updatedBy.String

	// Parse JSON fields using helpers
	if err := parseJSONFieldInto(variablesJSON, &template.Variables, "variables"); err != nil {
		return nil, err
	}

	config, err := parseJSONField[models.TemplateConfig](defaultConfigJSON, "default_config")
	if err != nil {
		return nil, err
	}
	template.DefaultConfig = config

	return &template, nil
}

// ListNewsletterTemplates retrieves all newsletter templates with optional filtering.
// Parameters:
//   - templateType: filter by type (empty string for all)
//   - activeFilter: filter by active status (nil for all, *true for active only, *false for inactive only)
//   - limit: maximum results
//   - offset: pagination offset
func (db *DB) ListNewsletterTemplates(ctx context.Context, templateType string, activeFilter *bool, limit, offset int) ([]models.NewsletterTemplate, int, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build filter conditions using filter builder
	fb := newNewsletterFilterBuilder()
	fb.addFilter("type", templateType).addBoolFilter("is_active", activeFilter)
	whereClause, filterArgs := fb.buildWhere()

	// Count query
	countQuery := "SELECT COUNT(*) FROM newsletter_templates" + whereClause
	var totalCount int
	if err := db.conn.QueryRowContext(ctx, countQuery, filterArgs...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count newsletter templates: %w", err)
	}

	// Select query
	query := `
		SELECT
			id, name, description, type, subject, body_html, body_text,
			variables::VARCHAR, default_config::VARCHAR, version, is_built_in, is_active,
			created_by, created_at, updated_by, updated_at
		FROM newsletter_templates` + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"

	args := append(filterArgs, limit, offset)
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query newsletter templates: %w", err)
	}
	defer rows.Close()

	templates, err := db.scanTemplateRows(rows)
	if err != nil {
		return nil, 0, err
	}

	return templates, totalCount, nil
}

// scanTemplateRows scans multiple template rows into a slice.
func (db *DB) scanTemplateRows(rows *sql.Rows) ([]models.NewsletterTemplate, error) {
	var templates []models.NewsletterTemplate
	for rows.Next() {
		template, err := db.scanTemplateFromRows(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *template)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if templates == nil {
		templates = []models.NewsletterTemplate{}
	}

	return templates, nil
}

// scanTemplateFromRows scans a single template from rows.
func (db *DB) scanTemplateFromRows(rows *sql.Rows) (*models.NewsletterTemplate, error) {
	var template models.NewsletterTemplate
	var description, bodyText, updatedBy sql.NullString
	var variablesJSON, defaultConfigJSON sql.NullString

	err := rows.Scan(
		&template.ID,
		&template.Name,
		&description,
		&template.Type,
		&template.Subject,
		&template.BodyHTML,
		&bodyText,
		&variablesJSON,
		&defaultConfigJSON,
		&template.Version,
		&template.IsBuiltIn,
		&template.IsActive,
		&template.CreatedBy,
		&template.CreatedAt,
		&updatedBy,
		&template.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan newsletter template: %w", err)
	}

	// Set nullable string fields
	template.Description = description.String
	template.BodyText = bodyText.String
	template.UpdatedBy = updatedBy.String

	// Parse JSON fields using helpers
	if err := parseJSONFieldInto(variablesJSON, &template.Variables, "variables"); err != nil {
		return nil, err
	}

	config, err := parseJSONField[models.TemplateConfig](defaultConfigJSON, "default_config")
	if err != nil {
		return nil, err
	}
	template.DefaultConfig = config

	return &template, nil
}

// UpdateNewsletterTemplate updates an existing newsletter template using partial updates.
// Only non-nil fields in the request are updated.
func (db *DB) UpdateNewsletterTemplate(ctx context.Context, id string, req *models.UpdateTemplateRequest, userID string) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build dynamic update query using the update builder
	ub := newUpdateBuilder()
	ub.setString("name", req.Name).
		setString("description", req.Description).
		setString("subject", req.Subject).
		setString("body_html", req.BodyHTML).
		setString("body_text", req.BodyText).
		setBool("is_active", req.IsActive)

	// Handle JSON field
	if err := ub.setJSON("default_config", req.DefaultConfig, "default_config"); err != nil {
		return err
	}

	if ub.isEmpty() {
		return nil // Nothing to update
	}

	// Always update version, updated_by, updated_at
	ub.setRaw("version = version + 1").
		setString("updated_by", &userID).
		setTimestamp("updated_at", time.Now())

	setClause, args := ub.build(id)
	query := fmt.Sprintf("UPDATE newsletter_templates SET %s WHERE id = ?", setClause)

	result, err := db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update newsletter template: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("template not found")
	}

	return nil
}

// joinStrings joins strings with a separator (helper function).
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// DeleteNewsletterTemplate deletes a newsletter template by ID.
func (db *DB) DeleteNewsletterTemplate(ctx context.Context, id string) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `DELETE FROM newsletter_templates WHERE id = ?`

	result, err := db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete newsletter template: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("template not found")
	}

	return nil
}

// ============================================================================
// Newsletter Schedules
// ============================================================================

// CreateNewsletterSchedule creates a new newsletter schedule.
func (db *DB) CreateNewsletterSchedule(ctx context.Context, schedule *models.NewsletterSchedule) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Generate ID if not provided
	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}

	// Marshal JSON fields
	recipientsJSON, err := json.Marshal(schedule.Recipients)
	if err != nil {
		return fmt.Errorf("failed to marshal recipients: %w", err)
	}

	channelsJSON, err := json.Marshal(schedule.Channels)
	if err != nil {
		return fmt.Errorf("failed to marshal channels: %w", err)
	}

	var configJSON, channelConfigsJSON []byte
	if schedule.Config != nil {
		configJSON, err = json.Marshal(schedule.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
	}
	if schedule.ChannelConfigs != nil {
		channelConfigsJSON, err = json.Marshal(schedule.ChannelConfigs)
		if err != nil {
			return fmt.Errorf("failed to marshal channel_configs: %w", err)
		}
	}

	query := `
		INSERT INTO newsletter_schedules (
			id, name, description, template_id, recipients, cron_expression, timezone,
			config, channels, channel_configs, is_enabled, next_run_at,
			created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	_, err = db.conn.ExecContext(ctx, query,
		schedule.ID,
		schedule.Name,
		schedule.Description,
		schedule.TemplateID,
		string(recipientsJSON),
		schedule.CronExpression,
		schedule.Timezone,
		nullableJSON(configJSON),
		string(channelsJSON),
		nullableJSON(channelConfigsJSON),
		schedule.IsEnabled,
		schedule.NextRunAt,
		schedule.CreatedBy,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert newsletter schedule: %w", err)
	}

	return nil
}

// GetNewsletterSchedule retrieves a newsletter schedule by ID.
func (db *DB) GetNewsletterSchedule(ctx context.Context, id string) (*models.NewsletterSchedule, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		SELECT
			s.id, s.name, s.description, s.template_id, t.name as template_name,
			s.recipients::VARCHAR, s.cron_expression, s.timezone,
			s.config::VARCHAR, s.channels::VARCHAR, s.channel_configs::VARCHAR,
			s.is_enabled, s.last_run_at, s.next_run_at, s.last_run_status,
			s.run_count, s.success_count, s.failure_count,
			s.created_by, s.created_at, s.updated_by, s.updated_at
		FROM newsletter_schedules s
		LEFT JOIN newsletter_templates t ON s.template_id = t.id
		WHERE s.id = ?
	`

	row := db.conn.QueryRowContext(ctx, query, id)
	return db.scanNewsletterSchedule(row)
}

// ListNewsletterSchedules retrieves all newsletter schedules with optional filtering.
// Parameters:
//   - templateID: filter by template ID (empty string for all)
//   - enabledFilter: filter by enabled status (nil for all)
//   - limit: maximum results
//   - offset: pagination offset
func (db *DB) ListNewsletterSchedules(ctx context.Context, templateID string, enabledFilter *bool, limit, offset int) ([]models.NewsletterSchedule, int, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Count query
	countQuery := `SELECT COUNT(*) FROM newsletter_schedules WHERE 1=1`
	countArgs := make([]interface{}, 0)

	if templateID != "" {
		countQuery += " AND template_id = ?"
		countArgs = append(countArgs, templateID)
	}
	if enabledFilter != nil {
		countQuery += " AND is_enabled = ?"
		countArgs = append(countArgs, *enabledFilter)
	}

	var totalCount int
	if err := db.conn.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count newsletter schedules: %w", err)
	}

	query := `
		SELECT
			s.id, s.name, s.description, s.template_id, t.name as template_name,
			s.recipients::VARCHAR, s.cron_expression, s.timezone,
			s.config::VARCHAR, s.channels::VARCHAR, s.channel_configs::VARCHAR,
			s.is_enabled, s.last_run_at, s.next_run_at, s.last_run_status,
			s.run_count, s.success_count, s.failure_count,
			s.created_by, s.created_at, s.updated_by, s.updated_at
		FROM newsletter_schedules s
		LEFT JOIN newsletter_templates t ON s.template_id = t.id
		WHERE 1=1
	`
	args := make([]interface{}, 0)

	if templateID != "" {
		query += " AND s.template_id = ?"
		args = append(args, templateID)
	}
	if enabledFilter != nil {
		query += " AND s.is_enabled = ?"
		args = append(args, *enabledFilter)
	}

	query += " ORDER BY s.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query newsletter schedules: %w", err)
	}
	defer rows.Close()

	var schedules []models.NewsletterSchedule
	for rows.Next() {
		schedule, err := db.scanNewsletterScheduleRow(rows)
		if err != nil {
			return nil, 0, err
		}
		schedules = append(schedules, *schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if schedules == nil {
		schedules = []models.NewsletterSchedule{}
	}

	return schedules, totalCount, nil
}

// GetSchedulesDueForRun retrieves schedules that are due to run.
func (db *DB) GetSchedulesDueForRun(ctx context.Context) ([]models.NewsletterSchedule, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		SELECT
			s.id, s.name, s.description, s.template_id, t.name as template_name,
			s.recipients::VARCHAR, s.cron_expression, s.timezone,
			s.config::VARCHAR, s.channels::VARCHAR, s.channel_configs::VARCHAR,
			s.is_enabled, s.last_run_at, s.next_run_at, s.last_run_status,
			s.run_count, s.success_count, s.failure_count,
			s.created_by, s.created_at, s.updated_by, s.updated_at
		FROM newsletter_schedules s
		LEFT JOIN newsletter_templates t ON s.template_id = t.id
		WHERE s.is_enabled = TRUE
			AND s.next_run_at IS NOT NULL
			AND s.next_run_at <= ?
		ORDER BY s.next_run_at ASC
	`

	rows, err := db.conn.QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query due schedules: %w", err)
	}
	defer rows.Close()

	var schedules []models.NewsletterSchedule
	for rows.Next() {
		schedule, err := db.scanNewsletterScheduleRow(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, *schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if schedules == nil {
		schedules = []models.NewsletterSchedule{}
	}

	return schedules, nil
}

// UpdateNewsletterSchedule updates an existing newsletter schedule using partial updates.
// Only non-nil fields in the request are updated.
func (db *DB) UpdateNewsletterSchedule(ctx context.Context, id string, req *models.UpdateScheduleRequest, userID string) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build dynamic update query using the update builder
	ub := newUpdateBuilder()
	ub.setString("name", req.Name).
		setString("description", req.Description).
		setString("template_id", req.TemplateID).
		setString("cron_expression", req.CronExpression).
		setString("timezone", req.Timezone).
		setBool("is_enabled", req.IsEnabled)

	// Handle JSON fields
	if err := ub.setJSON("recipients", req.Recipients, "recipients"); err != nil {
		return err
	}
	if err := ub.setJSON("config", req.Config, "config"); err != nil {
		return err
	}
	if err := ub.setJSON("channels", req.Channels, "channels"); err != nil {
		return err
	}
	if err := ub.setJSON("channel_configs", req.ChannelConfigs, "channel_configs"); err != nil {
		return err
	}

	if ub.isEmpty() {
		return nil // Nothing to update
	}

	// Always update updated_by, updated_at
	ub.setString("updated_by", &userID).
		setTimestamp("updated_at", time.Now())

	setClause, args := ub.build(id)
	query := fmt.Sprintf("UPDATE newsletter_schedules SET %s WHERE id = ?", setClause)

	result, err := db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update newsletter schedule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("schedule not found")
	}

	return nil
}

// UpdateScheduleRunStatus updates the run status and statistics for a schedule.
func (db *DB) UpdateScheduleRunStatus(ctx context.Context, id string, status models.DeliveryStatus, nextRunAt *time.Time) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		UPDATE newsletter_schedules SET
			last_run_at = ?,
			last_run_status = ?,
			next_run_at = ?,
			run_count = run_count + 1,
			success_count = success_count + CASE WHEN ? = 'delivered' THEN 1 ELSE 0 END,
			failure_count = failure_count + CASE WHEN ? = 'failed' THEN 1 ELSE 0 END,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := db.conn.ExecContext(ctx, query,
		now,
		string(status),
		nextRunAt,
		string(status),
		string(status),
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update schedule run status: %w", err)
	}

	return nil
}

// DeleteNewsletterSchedule deletes a newsletter schedule by ID.
func (db *DB) DeleteNewsletterSchedule(ctx context.Context, id string) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `DELETE FROM newsletter_schedules WHERE id = ?`

	result, err := db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete newsletter schedule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("schedule not found")
	}

	return nil
}

// scanScheduleFromRow scans a newsletter schedule from any row scanner.
// This unified function works with both *sql.Row and *sql.Rows via the rowScanner interface.
func (db *DB) scanScheduleFromRow(scanner rowScanner) (*models.NewsletterSchedule, error) {
	var schedule models.NewsletterSchedule
	var description, templateName, updatedBy, lastRunStatus sql.NullString
	var recipientsJSON, configJSON, channelsJSON, channelConfigsJSON sql.NullString
	var lastRunAt, nextRunAt sql.NullTime

	err := scanner.Scan(
		&schedule.ID,
		&schedule.Name,
		&description,
		&schedule.TemplateID,
		&templateName,
		&recipientsJSON,
		&schedule.CronExpression,
		&schedule.Timezone,
		&configJSON,
		&channelsJSON,
		&channelConfigsJSON,
		&schedule.IsEnabled,
		&lastRunAt,
		&nextRunAt,
		&lastRunStatus,
		&schedule.RunCount,
		&schedule.SuccessCount,
		&schedule.FailureCount,
		&schedule.CreatedBy,
		&schedule.CreatedAt,
		&updatedBy,
		&schedule.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan newsletter schedule: %w", err)
	}

	// Set nullable string fields
	schedule.Description = description.String
	schedule.TemplateName = templateName.String
	schedule.UpdatedBy = updatedBy.String
	schedule.LastRunStatus = models.DeliveryStatus(lastRunStatus.String)

	// Set nullable time fields
	if lastRunAt.Valid {
		schedule.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		schedule.NextRunAt = &nextRunAt.Time
	}

	// Parse JSON fields using helpers
	if err := parseJSONFieldInto(recipientsJSON, &schedule.Recipients, "recipients"); err != nil {
		return nil, err
	}
	if err := parseJSONFieldInto(channelsJSON, &schedule.Channels, "channels"); err != nil {
		return nil, err
	}

	config, err := parseJSONField[models.TemplateConfig](configJSON, "config")
	if err != nil {
		return nil, err
	}
	schedule.Config = config

	channelConfigs, err := parseJSONField[map[models.DeliveryChannel]*models.ChannelConfig](channelConfigsJSON, "channel_configs")
	if err != nil {
		return nil, err
	}
	if channelConfigs != nil {
		schedule.ChannelConfigs = *channelConfigs
	}

	return &schedule, nil
}

// scanNewsletterSchedule scans a single schedule row (wrapper for backward compatibility).
func (db *DB) scanNewsletterSchedule(row *sql.Row) (*models.NewsletterSchedule, error) {
	schedule, err := db.scanScheduleFromRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return schedule, err
}

// scanNewsletterScheduleRow scans a schedule from rows (wrapper for backward compatibility).
func (db *DB) scanNewsletterScheduleRow(rows *sql.Rows) (*models.NewsletterSchedule, error) {
	return db.scanScheduleFromRow(rows)
}

// ============================================================================
// Newsletter Deliveries
// ============================================================================

// CreateNewsletterDelivery creates a new newsletter delivery record.
func (db *DB) CreateNewsletterDelivery(ctx context.Context, delivery *models.NewsletterDelivery) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Generate ID if not provided
	if delivery.ID == "" {
		delivery.ID = uuid.New().String()
	}

	// Marshal JSON fields
	var recipientDetailsJSON, contentStatsJSON, errorDetailsJSON []byte
	var err error

	if delivery.RecipientDetails != nil {
		recipientDetailsJSON, err = json.Marshal(delivery.RecipientDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal recipient_details: %w", err)
		}
	}
	if delivery.ContentStats != nil {
		contentStatsJSON, err = json.Marshal(delivery.ContentStats)
		if err != nil {
			return fmt.Errorf("failed to marshal content_stats: %w", err)
		}
	}
	if delivery.ErrorDetails != nil {
		errorDetailsJSON, err = json.Marshal(delivery.ErrorDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal error_details: %w", err)
		}
	}

	query := `
		INSERT INTO newsletter_deliveries (
			id, schedule_id, template_id, template_version, channel, status,
			recipients_total, recipients_delivered, recipients_failed,
			recipient_details, content_summary, content_stats,
			rendered_subject, rendered_body_size, started_at, completed_at, duration_ms,
			error_message, error_details, triggered_by, triggered_by_user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.conn.ExecContext(ctx, query,
		delivery.ID,
		nullableString(delivery.ScheduleID),
		delivery.TemplateID,
		delivery.TemplateVersion,
		string(delivery.Channel),
		string(delivery.Status),
		delivery.RecipientsTotal,
		delivery.RecipientsDelivered,
		delivery.RecipientsFailed,
		nullableJSON(recipientDetailsJSON),
		delivery.ContentSummary,
		nullableJSON(contentStatsJSON),
		delivery.RenderedSubject,
		delivery.RenderedBodySize,
		delivery.StartedAt,
		delivery.CompletedAt,
		delivery.DurationMS,
		delivery.ErrorMessage,
		nullableJSON(errorDetailsJSON),
		delivery.TriggeredBy,
		nullableString(delivery.TriggeredByUserID),
	)
	if err != nil {
		return fmt.Errorf("failed to insert newsletter delivery: %w", err)
	}

	return nil
}

// UpdateNewsletterDelivery updates an existing delivery record.
func (db *DB) UpdateNewsletterDelivery(ctx context.Context, delivery *models.NewsletterDelivery) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Marshal JSON fields
	var recipientDetailsJSON, errorDetailsJSON []byte
	var err error

	if delivery.RecipientDetails != nil {
		recipientDetailsJSON, err = json.Marshal(delivery.RecipientDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal recipient_details: %w", err)
		}
	}
	if delivery.ErrorDetails != nil {
		errorDetailsJSON, err = json.Marshal(delivery.ErrorDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal error_details: %w", err)
		}
	}

	query := `
		UPDATE newsletter_deliveries SET
			status = ?,
			recipients_delivered = ?,
			recipients_failed = ?,
			recipient_details = ?,
			completed_at = ?,
			duration_ms = ?,
			error_message = ?,
			error_details = ?
		WHERE id = ?
	`

	_, err = db.conn.ExecContext(ctx, query,
		string(delivery.Status),
		delivery.RecipientsDelivered,
		delivery.RecipientsFailed,
		nullableJSON(recipientDetailsJSON),
		delivery.CompletedAt,
		delivery.DurationMS,
		delivery.ErrorMessage,
		nullableJSON(errorDetailsJSON),
		delivery.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update newsletter delivery: %w", err)
	}

	return nil
}

// GetNewsletterDelivery retrieves a newsletter delivery by ID.
func (db *DB) GetNewsletterDelivery(ctx context.Context, id string) (*models.NewsletterDelivery, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		SELECT
			d.id, d.schedule_id, s.name as schedule_name, d.template_id, t.name as template_name,
			d.template_version, d.channel, d.status,
			d.recipients_total, d.recipients_delivered, d.recipients_failed,
			d.recipient_details::VARCHAR, d.content_summary, d.content_stats::VARCHAR,
			d.rendered_subject, d.rendered_body_size, d.started_at, d.completed_at, d.duration_ms,
			d.error_message, d.error_details::VARCHAR, d.triggered_by, d.triggered_by_user_id
		FROM newsletter_deliveries d
		LEFT JOIN newsletter_schedules s ON d.schedule_id = s.id
		LEFT JOIN newsletter_templates t ON d.template_id = t.id
		WHERE d.id = ?
	`

	row := db.conn.QueryRowContext(ctx, query, id)

	var delivery models.NewsletterDelivery
	var scheduleID, scheduleName, templateName, triggeredByUserID sql.NullString
	var contentSummary, renderedSubject, errorMessage sql.NullString
	var recipientDetailsJSON, contentStatsJSON, errorDetailsJSON sql.NullString
	var renderedBodySize sql.NullInt64
	var completedAt sql.NullTime
	var durationMS sql.NullInt64

	err := row.Scan(
		&delivery.ID,
		&scheduleID,
		&scheduleName,
		&delivery.TemplateID,
		&templateName,
		&delivery.TemplateVersion,
		&delivery.Channel,
		&delivery.Status,
		&delivery.RecipientsTotal,
		&delivery.RecipientsDelivered,
		&delivery.RecipientsFailed,
		&recipientDetailsJSON,
		&contentSummary,
		&contentStatsJSON,
		&renderedSubject,
		&renderedBodySize,
		&delivery.StartedAt,
		&completedAt,
		&durationMS,
		&errorMessage,
		&errorDetailsJSON,
		&delivery.TriggeredBy,
		&triggeredByUserID,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan newsletter delivery: %w", err)
	}

	delivery.ScheduleID = scheduleID.String
	delivery.ScheduleName = scheduleName.String
	delivery.TemplateName = templateName.String
	delivery.ContentSummary = contentSummary.String
	delivery.RenderedSubject = renderedSubject.String
	delivery.ErrorMessage = errorMessage.String
	delivery.TriggeredByUserID = triggeredByUserID.String
	delivery.RenderedBodySize = int(renderedBodySize.Int64)
	delivery.DurationMS = durationMS.Int64
	if completedAt.Valid {
		delivery.CompletedAt = &completedAt.Time
	}

	// Parse JSON fields
	if recipientDetailsJSON.Valid && recipientDetailsJSON.String != "" {
		if err := json.Unmarshal([]byte(recipientDetailsJSON.String), &delivery.RecipientDetails); err != nil {
			return nil, fmt.Errorf("failed to parse recipient_details: %w", err)
		}
	}
	if contentStatsJSON.Valid && contentStatsJSON.String != "" {
		var stats models.DeliveryContentStats
		if err := json.Unmarshal([]byte(contentStatsJSON.String), &stats); err != nil {
			return nil, fmt.Errorf("failed to parse content_stats: %w", err)
		}
		delivery.ContentStats = &stats
	}
	if errorDetailsJSON.Valid && errorDetailsJSON.String != "" {
		if err := json.Unmarshal([]byte(errorDetailsJSON.String), &delivery.ErrorDetails); err != nil {
			return nil, fmt.Errorf("failed to parse error_details: %w", err)
		}
	}

	return &delivery, nil
}

// ListNewsletterDeliveries retrieves delivery history with pagination.
// Parameters:
//   - scheduleID: filter by schedule ID (empty string for all)
//   - status: filter by delivery status (empty string for all)
//   - limit: maximum results
//   - offset: pagination offset
func (db *DB) ListNewsletterDeliveries(ctx context.Context, scheduleID string, status string, limit, offset int) ([]models.NewsletterDelivery, int, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build query
	baseQuery := `
		FROM newsletter_deliveries d
		LEFT JOIN newsletter_schedules s ON d.schedule_id = s.id
		LEFT JOIN newsletter_templates t ON d.template_id = t.id
		WHERE 1=1
	`
	args := make([]interface{}, 0)

	if scheduleID != "" {
		baseQuery += " AND d.schedule_id = ?"
		args = append(args, scheduleID)
	}
	if status != "" {
		baseQuery += " AND d.status = ?"
		args = append(args, status)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var totalCount int
	if err := db.conn.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count deliveries: %w", err)
	}

	// Get deliveries
	selectQuery := `
		SELECT
			d.id, d.schedule_id, s.name as schedule_name, d.template_id, t.name as template_name,
			d.template_version, d.channel, d.status,
			d.recipients_total, d.recipients_delivered, d.recipients_failed,
			d.content_summary, d.rendered_subject, d.started_at, d.completed_at, d.duration_ms,
			d.error_message, d.triggered_by, d.triggered_by_user_id
	` + baseQuery + " ORDER BY d.started_at DESC LIMIT ? OFFSET ?"

	args = append(args, limit, offset)

	rows, err := db.conn.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []models.NewsletterDelivery
	for rows.Next() {
		var delivery models.NewsletterDelivery
		var scheduleID, scheduleName, templateName, triggeredByUserID sql.NullString
		var contentSummary, renderedSubject, errorMessage sql.NullString
		var completedAt sql.NullTime
		var durationMS sql.NullInt64

		err := rows.Scan(
			&delivery.ID,
			&scheduleID,
			&scheduleName,
			&delivery.TemplateID,
			&templateName,
			&delivery.TemplateVersion,
			&delivery.Channel,
			&delivery.Status,
			&delivery.RecipientsTotal,
			&delivery.RecipientsDelivered,
			&delivery.RecipientsFailed,
			&contentSummary,
			&renderedSubject,
			&delivery.StartedAt,
			&completedAt,
			&durationMS,
			&errorMessage,
			&delivery.TriggeredBy,
			&triggeredByUserID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan delivery: %w", err)
		}

		delivery.ScheduleID = scheduleID.String
		delivery.ScheduleName = scheduleName.String
		delivery.TemplateName = templateName.String
		delivery.ContentSummary = contentSummary.String
		delivery.RenderedSubject = renderedSubject.String
		delivery.ErrorMessage = errorMessage.String
		delivery.TriggeredByUserID = triggeredByUserID.String
		delivery.DurationMS = durationMS.Int64
		if completedAt.Valid {
			delivery.CompletedAt = &completedAt.Time
		}

		deliveries = append(deliveries, delivery)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if deliveries == nil {
		deliveries = []models.NewsletterDelivery{}
	}

	return deliveries, totalCount, nil
}

// ============================================================================
// Newsletter User Preferences
// ============================================================================

// GetNewsletterUserPreferences retrieves user preferences.
func (db *DB) GetNewsletterUserPreferences(ctx context.Context, userID string) (*models.NewsletterUserPreferences, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		SELECT
			user_id, username, global_opt_out, global_opt_out_at,
			schedule_preferences::VARCHAR, preferred_channel, preferred_email, language, updated_at
		FROM newsletter_user_preferences
		WHERE user_id = ?
	`

	row := db.conn.QueryRowContext(ctx, query, userID)

	var prefs models.NewsletterUserPreferences
	var globalOptOutAt sql.NullTime
	var schedulePrefsJSON, preferredChannel, preferredEmail, language sql.NullString

	err := row.Scan(
		&prefs.UserID,
		&prefs.Username,
		&prefs.GlobalOptOut,
		&globalOptOutAt,
		&schedulePrefsJSON,
		&preferredChannel,
		&preferredEmail,
		&language,
		&prefs.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan user preferences: %w", err)
	}

	if globalOptOutAt.Valid {
		prefs.GlobalOptOutAt = &globalOptOutAt.Time
	}
	prefs.PreferredEmail = preferredEmail.String
	prefs.Language = language.String
	if preferredChannel.Valid && preferredChannel.String != "" {
		ch := models.DeliveryChannel(preferredChannel.String)
		prefs.PreferredChannel = &ch
	}

	// Parse JSON
	if schedulePrefsJSON.Valid && schedulePrefsJSON.String != "" {
		if err := json.Unmarshal([]byte(schedulePrefsJSON.String), &prefs.SchedulePreferences); err != nil {
			return nil, fmt.Errorf("failed to parse schedule_preferences: %w", err)
		}
	}

	return &prefs, nil
}

// UpsertNewsletterUserPreferences creates or updates user preferences.
func (db *DB) UpsertNewsletterUserPreferences(ctx context.Context, prefs *models.NewsletterUserPreferences) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Marshal JSON
	var schedulePrefsJSON []byte
	var err error
	if prefs.SchedulePreferences != nil {
		schedulePrefsJSON, err = json.Marshal(prefs.SchedulePreferences)
		if err != nil {
			return fmt.Errorf("failed to marshal schedule_preferences: %w", err)
		}
	}

	var preferredChannel *string
	if prefs.PreferredChannel != nil {
		s := string(*prefs.PreferredChannel)
		preferredChannel = &s
	}

	prefs.UpdatedAt = time.Now()

	query := `
		INSERT INTO newsletter_user_preferences (
			user_id, username, global_opt_out, global_opt_out_at,
			schedule_preferences, preferred_channel, preferred_email, language, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			username = EXCLUDED.username,
			global_opt_out = EXCLUDED.global_opt_out,
			global_opt_out_at = EXCLUDED.global_opt_out_at,
			schedule_preferences = EXCLUDED.schedule_preferences,
			preferred_channel = EXCLUDED.preferred_channel,
			preferred_email = EXCLUDED.preferred_email,
			language = EXCLUDED.language,
			updated_at = EXCLUDED.updated_at
	`

	_, err = db.conn.ExecContext(ctx, query,
		prefs.UserID,
		prefs.Username,
		prefs.GlobalOptOut,
		prefs.GlobalOptOutAt,
		nullableJSON(schedulePrefsJSON),
		preferredChannel,
		prefs.PreferredEmail,
		prefs.Language,
		prefs.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert user preferences: %w", err)
	}

	return nil
}

// ============================================================================
// Newsletter Audit Log
// ============================================================================

// CreateNewsletterAuditEntry creates a new audit log entry.
func (db *DB) CreateNewsletterAuditEntry(ctx context.Context, entry *models.NewsletterAuditEntry) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Marshal details
	var detailsJSON []byte
	var err error
	if entry.Details != nil {
		detailsJSON, err = json.Marshal(entry.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	entry.Timestamp = time.Now()

	query := `
		INSERT INTO newsletter_audit_log (
			id, timestamp, actor_id, actor_username, action,
			resource_type, resource_id, resource_name, details, ip_address, user_agent
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.conn.ExecContext(ctx, query,
		entry.ID,
		entry.Timestamp,
		entry.ActorID,
		entry.ActorUsername,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		entry.ResourceName,
		nullableJSON(detailsJSON),
		entry.IPAddress,
		entry.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("failed to insert audit entry: %w", err)
	}

	return nil
}

// ListNewsletterAuditLog retrieves audit log entries with pagination.
// Parameters:
//   - resourceType: filter by resource type (empty string for all)
//   - resourceID: filter by resource ID (empty string for all)
//   - actorID: filter by actor ID (empty string for all)
//   - action: filter by action (empty string for all)
//   - limit: maximum results
//   - offset: pagination offset
func (db *DB) ListNewsletterAuditLog(ctx context.Context, resourceType, resourceID, actorID, action string, limit, offset int) ([]models.NewsletterAuditEntry, int, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build query
	baseQuery := " FROM newsletter_audit_log WHERE 1=1"
	args := make([]interface{}, 0)

	if resourceType != "" {
		baseQuery += " AND resource_type = ?"
		args = append(args, resourceType)
	}
	if resourceID != "" {
		baseQuery += " AND resource_id = ?"
		args = append(args, resourceID)
	}
	if actorID != "" {
		baseQuery += " AND actor_id = ?"
		args = append(args, actorID)
	}
	if action != "" {
		baseQuery += " AND action = ?"
		args = append(args, action)
	}

	// Get total count
	countQuery := "SELECT COUNT(*)" + baseQuery
	var totalCount int
	if err := db.conn.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit entries: %w", err)
	}

	// Get entries
	selectQuery := `
		SELECT
			id, timestamp, actor_id, actor_username, action,
			resource_type, resource_id, resource_name, details::VARCHAR, ip_address, user_agent
	` + baseQuery + " ORDER BY timestamp DESC LIMIT ? OFFSET ?"

	args = append(args, limit, offset)

	rows, err := db.conn.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	var entries []models.NewsletterAuditEntry
	for rows.Next() {
		var entry models.NewsletterAuditEntry
		var actorUsername, resourceName, detailsJSON, ipAddress, userAgent sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.ActorID,
			&actorUsername,
			&entry.Action,
			&entry.ResourceType,
			&entry.ResourceID,
			&resourceName,
			&detailsJSON,
			&ipAddress,
			&userAgent,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		entry.ActorUsername = actorUsername.String
		entry.ResourceName = resourceName.String
		entry.IPAddress = ipAddress.String
		entry.UserAgent = userAgent.String

		if detailsJSON.Valid && detailsJSON.String != "" {
			if err := json.Unmarshal([]byte(detailsJSON.String), &entry.Details); err != nil {
				return nil, 0, fmt.Errorf("failed to parse details: %w", err)
			}
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if entries == nil {
		entries = []models.NewsletterAuditEntry{}
	}

	return entries, totalCount, nil
}

// ============================================================================
// Newsletter Statistics
// ============================================================================

// GetNewsletterStats retrieves aggregated newsletter statistics.
func (db *DB) GetNewsletterStats(ctx context.Context) (*models.NewsletterStatsResponse, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	stats := &models.NewsletterStatsResponse{
		DeliveriesByChannel: make(map[string]int),
		DeliveriesByType:    make(map[string]int),
	}

	// Template counts
	templateQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN is_active THEN 1 END) as active
		FROM newsletter_templates
	`
	if err := db.conn.QueryRowContext(ctx, templateQuery).Scan(&stats.TotalTemplates, &stats.ActiveTemplates); err != nil {
		return nil, fmt.Errorf("failed to count templates: %w", err)
	}

	// Schedule counts
	scheduleQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN is_enabled THEN 1 END) as enabled
		FROM newsletter_schedules
	`
	if err := db.conn.QueryRowContext(ctx, scheduleQuery).Scan(&stats.TotalSchedules, &stats.EnabledSchedules); err != nil {
		return nil, fmt.Errorf("failed to count schedules: %w", err)
	}

	// Delivery counts
	deliveryQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'delivered' THEN 1 END) as successful,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM newsletter_deliveries
	`
	if err := db.conn.QueryRowContext(ctx, deliveryQuery).Scan(&stats.TotalDeliveries, &stats.SuccessfulDeliveries, &stats.FailedDeliveries); err != nil {
		return nil, fmt.Errorf("failed to count deliveries: %w", err)
	}

	// Deliveries by channel
	channelQuery := `
		SELECT channel, COUNT(*) as count
		FROM newsletter_deliveries
		GROUP BY channel
	`
	channelRows, err := db.conn.QueryContext(ctx, channelQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to count by channel: %w", err)
	}
	defer func() { _ = channelRows.Close() }()

	for channelRows.Next() {
		var channel string
		var count int
		if err := channelRows.Scan(&channel, &count); err != nil {
			return nil, err
		}
		stats.DeliveriesByChannel[channel] = count
	}
	if err := channelRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating channel rows: %w", err)
	}

	// Recent deliveries (last 7 and 30 days)
	recentQuery := `
		SELECT
			COUNT(CASE WHEN started_at >= ? THEN 1 END) as last_7_days,
			COUNT(CASE WHEN started_at >= ? THEN 1 END) as last_30_days
		FROM newsletter_deliveries
	`
	now := time.Now()
	last7Days := now.AddDate(0, 0, -7)
	last30Days := now.AddDate(0, 0, -30)

	if err := db.conn.QueryRowContext(ctx, recentQuery, last7Days, last30Days).Scan(&stats.Last7DaysDeliveries, &stats.Last30DaysDeliveries); err != nil {
		return nil, fmt.Errorf("failed to count recent deliveries: %w", err)
	}

	return stats, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// nullableJSON returns a sql.NullString for JSON data.
func nullableJSON(data []byte) interface{} {
	if len(data) == 0 {
		return nil
	}
	return string(data)
}

// nullableString returns a sql.NullString for optional string data.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ============================================================================
// JSON Parsing Helpers
// ============================================================================

// parseJSONField unmarshals a NullString JSON field into a destination pointer.
// Returns nil if the field is not valid or empty.
func parseJSONField[T any](field sql.NullString, fieldName string) (*T, error) {
	if !field.Valid || field.String == "" {
		return nil, nil
	}
	var result T
	if err := json.Unmarshal([]byte(field.String), &result); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", fieldName, err)
	}
	return &result, nil
}

// parseJSONFieldInto unmarshals a NullString JSON field into an existing destination.
// Returns nil error if the field is not valid or empty (no-op).
func parseJSONFieldInto(field sql.NullString, dest interface{}, fieldName string) error {
	if !field.Valid || field.String == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(field.String), dest); err != nil {
		return fmt.Errorf("failed to parse %s: %w", fieldName, err)
	}
	return nil
}

// marshalJSONField marshals a value to JSON bytes with error wrapping.
func marshalJSONField(v interface{}, fieldName string) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s: %w", fieldName, err)
	}
	return data, nil
}

// ============================================================================
// Newsletter Filter Builder Helpers
// ============================================================================

// newsletterFilterBuilder helps construct dynamic SQL WHERE clauses for newsletter queries.
type newsletterFilterBuilder struct {
	clauses []string
	args    []interface{}
}

// newNewsletterFilterBuilder creates a new filter builder.
func newNewsletterFilterBuilder() *newsletterFilterBuilder {
	return &newsletterFilterBuilder{
		clauses: make([]string, 0),
		args:    make([]interface{}, 0),
	}
}

// addFilter adds a filter clause if the value is non-empty (for strings).
func (fb *newsletterFilterBuilder) addFilter(column string, value string) *newsletterFilterBuilder {
	if value != "" {
		fb.clauses = append(fb.clauses, column+" = ?")
		fb.args = append(fb.args, value)
	}
	return fb
}

// addBoolFilter adds a filter clause if the bool pointer is non-nil.
func (fb *newsletterFilterBuilder) addBoolFilter(column string, value *bool) *newsletterFilterBuilder {
	if value != nil {
		fb.clauses = append(fb.clauses, column+" = ?")
		fb.args = append(fb.args, *value)
	}
	return fb
}

// buildWhere returns the WHERE clause string and arguments.
func (fb *newsletterFilterBuilder) buildWhere() (string, []interface{}) {
	if len(fb.clauses) == 0 {
		return " WHERE 1=1", fb.args
	}
	where := " WHERE 1=1"
	for _, clause := range fb.clauses {
		where += " AND " + clause
	}
	return where, fb.args
}

// ============================================================================
// Partial Update Builder
// ============================================================================

// updateBuilder helps construct dynamic UPDATE queries.
type updateBuilder struct {
	setClauses []string
	args       []interface{}
}

// newUpdateBuilder creates a new update builder.
func newUpdateBuilder() *updateBuilder {
	return &updateBuilder{
		setClauses: make([]string, 0),
		args:       make([]interface{}, 0),
	}
}

// setString adds a string field update if the pointer is non-nil.
func (ub *updateBuilder) setString(column string, value *string) *updateBuilder {
	if value != nil {
		ub.setClauses = append(ub.setClauses, column+" = ?")
		ub.args = append(ub.args, *value)
	}
	return ub
}

// setBool adds a bool field update if the pointer is non-nil.
//
//nolint:unparam // Returns *updateBuilder for API consistency with other builder methods.
func (ub *updateBuilder) setBool(column string, value *bool) *updateBuilder {
	if value != nil {
		ub.setClauses = append(ub.setClauses, column+" = ?")
		ub.args = append(ub.args, *value)
	}
	return ub
}

// setJSON marshals and adds a JSON field update if the value is non-nil.
func (ub *updateBuilder) setJSON(column string, value interface{}, fieldName string) error {
	if value == nil {
		return nil
	}
	data, err := marshalJSONField(value, fieldName)
	if err != nil {
		return err
	}
	ub.setClauses = append(ub.setClauses, column+" = ?")
	ub.args = append(ub.args, string(data))
	return nil
}

// setTimestamp adds a timestamp field update.
//
//nolint:unparam // Returns *updateBuilder for API consistency with other builder methods.
func (ub *updateBuilder) setTimestamp(column string, t time.Time) *updateBuilder {
	ub.setClauses = append(ub.setClauses, column+" = ?")
	ub.args = append(ub.args, t)
	return ub
}

// setRaw adds a raw SET clause.
func (ub *updateBuilder) setRaw(clause string) *updateBuilder {
	ub.setClauses = append(ub.setClauses, clause)
	return ub
}

// isEmpty returns true if no updates have been added.
func (ub *updateBuilder) isEmpty() bool {
	return len(ub.setClauses) == 0
}

// build returns the SET clause string and all arguments including the whereArg.
func (ub *updateBuilder) build(whereArg interface{}) (string, []interface{}) {
	args := append(ub.args, whereArg)
	return joinStrings(ub.setClauses, ", "), args
}

// ============================================================================
// Row Scanner Interface
// ============================================================================

// rowScanner is an interface that both sql.Row and sql.Rows satisfy.
type rowScanner interface {
	Scan(dest ...interface{}) error
}
