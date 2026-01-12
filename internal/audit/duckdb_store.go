// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package audit provides security audit logging functionality with DuckDB persistence.
package audit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// DuckDBStore implements Store using DuckDB for persistent storage.
// This provides durable audit logging suitable for production use.
type DuckDBStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewDuckDBStore creates a new DuckDB-backed audit store.
// The caller is responsible for ensuring the audit_events table exists.
func NewDuckDBStore(db *sql.DB) *DuckDBStore {
	return &DuckDBStore{
		db: db,
	}
}

// countByColumn executes a GROUP BY query and returns counts per value.
// This helper properly handles rows.Close() via defer.
func (s *DuckDBStore) countByColumn(ctx context.Context, column string) (map[string]int64, error) {
	result := make(map[string]int64)
	query := fmt.Sprintf("SELECT %s, COUNT(*) FROM audit_events GROUP BY %s", column, column)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s counts: %w", column, err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err == nil {
			result[key] = count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating %s counts: %w", column, err)
	}
	return result, nil
}

// buildSliceCondition creates a SQL IN condition for a slice of string values.
func buildSliceCondition[T ~string](column string, values []T, args *[]interface{}) string {
	if len(values) == 0 {
		return ""
	}
	placeholders := make([]string, len(values))
	for i, v := range values {
		placeholders[i] = "?"
		*args = append(*args, string(v))
	}
	return fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ","))
}

// CreateTable creates the audit_events table if it doesn't exist.
// This should be called during database initialization.
func (s *DuckDBStore) CreateTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS audit_events (
			id TEXT PRIMARY KEY,
			timestamp TIMESTAMPTZ NOT NULL,
			type TEXT NOT NULL,
			severity TEXT NOT NULL,
			outcome TEXT NOT NULL,

			-- Actor information
			actor_id TEXT NOT NULL,
			actor_type TEXT NOT NULL,
			actor_name TEXT,
			actor_roles JSON,
			actor_session_id TEXT,
			actor_auth_method TEXT,

			-- Target information (optional)
			target_id TEXT,
			target_type TEXT,
			target_name TEXT,

			-- Source information
			source_ip TEXT NOT NULL,
			source_user_agent TEXT,
			source_hostname TEXT,
			source_port INTEGER,
			source_geo JSON,

			-- Event details
			action TEXT NOT NULL,
			description TEXT NOT NULL,
			metadata JSON,

			-- Correlation
			correlation_id TEXT,
			request_id TEXT,

			-- Audit metadata
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		-- Indexes for common query patterns
		CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_events(timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_events(type);
		CREATE INDEX IF NOT EXISTS idx_audit_severity ON audit_events(severity);
		CREATE INDEX IF NOT EXISTS idx_audit_outcome ON audit_events(outcome);
		CREATE INDEX IF NOT EXISTS idx_audit_actor_id ON audit_events(actor_id);
		CREATE INDEX IF NOT EXISTS idx_audit_actor_type ON audit_events(actor_type);
		CREATE INDEX IF NOT EXISTS idx_audit_target_id ON audit_events(target_id);
		CREATE INDEX IF NOT EXISTS idx_audit_source_ip ON audit_events(source_ip);
		CREATE INDEX IF NOT EXISTS idx_audit_correlation_id ON audit_events(correlation_id);
		CREATE INDEX IF NOT EXISTS idx_audit_request_id ON audit_events(request_id);
		CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_events(created_at DESC);
	`

	// Split and execute each statement
	statements := strings.Split(query, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute schema statement: %w", err)
		}
	}

	logging.Info().Msg("Audit events table created/verified")
	return nil
}

// Save persists an audit event to DuckDB.
func (s *DuckDBStore) Save(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	params := s.prepareEventParams(event)
	query := s.getInsertQuery()

	_, err := s.db.ExecContext(ctx, query, params...)
	if err != nil {
		return fmt.Errorf("failed to save audit event: %w", err)
	}

	return nil
}

// prepareEventParams prepares all parameters for event insertion.
func (s *DuckDBStore) prepareEventParams(event *Event) []interface{} {
	actorRolesStr := marshalActorRoles(event.Actor.Roles)
	sourceGeoStr := marshalSourceGeo(event.Source.Geo)
	targetID, targetType, targetName := extractTargetFields(event.Target)
	metadataStr := extractMetadata(event.Metadata)

	return []interface{}{
		event.ID,
		event.Timestamp,
		string(event.Type),
		string(event.Severity),
		string(event.Outcome),
		event.Actor.ID,
		event.Actor.Type,
		event.Actor.Name,
		actorRolesStr,
		event.Actor.SessionID,
		event.Actor.AuthMethod,
		targetID,
		targetType,
		targetName,
		event.Source.IPAddress,
		event.Source.UserAgent,
		event.Source.Hostname,
		event.Source.Port,
		sourceGeoStr,
		event.Action,
		event.Description,
		metadataStr,
		event.CorrelationID,
		event.RequestID,
		time.Now().UTC(),
	}
}

// marshalActorRoles marshals actor roles to JSON string for DuckDB.
func marshalActorRoles(roles []string) string {
	if len(roles) == 0 {
		return "[]"
	}
	if data, err := json.Marshal(roles); err == nil {
		return string(data)
	}
	return "[]"
}

// marshalSourceGeo marshals source geo to JSON string (if present).
func marshalSourceGeo(geo *GeoLocation) *string {
	if geo == nil {
		return nil
	}
	if data, err := json.Marshal(geo); err == nil {
		s := string(data)
		return &s
	}
	return nil
}

// extractTargetFields extracts target fields for database insertion.
func extractTargetFields(target *Target) (*string, *string, *string) {
	if target == nil {
		return nil, nil, nil
	}
	return &target.ID, &target.Type, &target.Name
}

// extractMetadata converts metadata to string for DuckDB JSON column.
func extractMetadata(metadata json.RawMessage) *string {
	if len(metadata) == 0 {
		return nil
	}
	s := string(metadata)
	return &s
}

// getInsertQuery returns the INSERT statement for audit events.
func (s *DuckDBStore) getInsertQuery() string {
	return `
		INSERT INTO audit_events (
			id, timestamp, type, severity, outcome,
			actor_id, actor_type, actor_name, actor_roles, actor_session_id, actor_auth_method,
			target_id, target_type, target_name,
			source_ip, source_user_agent, source_hostname, source_port, source_geo,
			action, description, metadata,
			correlation_id, request_id, created_at
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?
		)
	`
}

// Get retrieves an event by ID.
func (s *DuckDBStore) Get(ctx context.Context, id string) (*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Cast JSON columns to VARCHAR for proper scanning
	query := `
		SELECT
			id, timestamp, type, severity, outcome,
			actor_id, actor_type, actor_name,
			CAST(actor_roles AS VARCHAR) as actor_roles,
			actor_session_id, actor_auth_method,
			target_id, target_type, target_name,
			source_ip, source_user_agent, source_hostname, source_port,
			CAST(source_geo AS VARCHAR) as source_geo,
			action, description,
			CAST(metadata AS VARCHAR) as metadata,
			correlation_id, request_id
		FROM audit_events
		WHERE id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id)
	event, err := s.scanEvent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("event not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get audit event: %w", err)
	}

	return event, nil
}

// Query retrieves events matching the filter.
func (s *DuckDBStore) Query(ctx context.Context, filter QueryFilter) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query, args := s.buildQuery(filter, false)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		event, err := s.scanEventFromRows(rows)
		if err != nil {
			logging.Warn().Err(err).Msg("Failed to scan audit event row")
			continue
		}
		events = append(events, *event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit events: %w", err)
	}

	return events, nil
}

// Count returns the number of events matching the filter.
func (s *DuckDBStore) Count(ctx context.Context, filter QueryFilter) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query, args := s.buildQuery(filter, true)

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit events: %w", err)
	}

	return count, nil
}

// Delete removes events older than the given time.
func (s *DuckDBStore) Delete(ctx context.Context, olderThan time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `DELETE FROM audit_events WHERE timestamp < ?`

	result, err := s.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit events: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted count: %w", err)
	}

	if count > 0 {
		logging.Info().Int64("deleted", count).Time("older_than", olderThan).Msg("Deleted old audit events")
	}

	return count, nil
}

// GetStats returns statistics about the audit store.
func (s *DuckDBStore) GetStats(ctx context.Context) (*Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &Stats{
		EventsByType:     make(map[string]int64),
		EventsBySeverity: make(map[string]int64),
		EventsByOutcome:  make(map[string]int64),
	}

	// Get total count
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events").Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats.TotalEvents = total

	// Get counts by type using helper (handles defer properly)
	typeCounts, err := s.countByColumn(ctx, "type")
	if err != nil {
		return nil, err
	}
	stats.EventsByType = typeCounts

	// Get counts by severity
	severityCounts, err := s.countByColumn(ctx, "severity")
	if err != nil {
		return nil, err
	}
	stats.EventsBySeverity = severityCounts

	// Get counts by outcome
	outcomeCounts, err := s.countByColumn(ctx, "outcome")
	if err != nil {
		return nil, err
	}
	stats.EventsByOutcome = outcomeCounts

	// Get oldest and newest events
	s.setEventTimeRange(ctx, stats)

	return stats, nil
}

// setEventTimeRange populates the oldest and newest event timestamps.
func (s *DuckDBStore) setEventTimeRange(ctx context.Context, stats *Stats) {
	var oldest, newest sql.NullTime
	err := s.db.QueryRowContext(ctx, "SELECT MIN(timestamp), MAX(timestamp) FROM audit_events").Scan(&oldest, &newest)
	if err == nil {
		if oldest.Valid {
			stats.OldestEvent = &oldest.Time
		}
		if newest.Valid {
			stats.NewestEvent = &newest.Time
		}
	}
}

// buildQuery constructs the SQL query based on the filter.
func (s *DuckDBStore) buildQuery(filter QueryFilter, countOnly bool) (string, []interface{}) {
	conditions, args := s.buildFilterConditions(filter)

	// Build base query
	query := s.getBaseQuery(countOnly)

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if !countOnly {
		query = s.appendOrderAndLimit(query, filter)
	}

	return query, args
}

// buildFilterConditions builds WHERE clause conditions from a QueryFilter.
func (s *DuckDBStore) buildFilterConditions(filter QueryFilter) ([]string, []interface{}) {
	var args []interface{}
	var conditions []string

	// Slice conditions using helper
	if cond := buildSliceCondition("type", filter.Types, &args); cond != "" {
		conditions = append(conditions, cond)
	}
	if cond := buildSliceCondition("severity", filter.Severities, &args); cond != "" {
		conditions = append(conditions, cond)
	}
	if cond := buildSliceCondition("outcome", filter.Outcomes, &args); cond != "" {
		conditions = append(conditions, cond)
	}

	// String equality conditions
	conditions, args = appendStringCondition(conditions, args, "actor_id", filter.ActorID)
	conditions, args = appendStringCondition(conditions, args, "actor_type", filter.ActorType)
	conditions, args = appendStringCondition(conditions, args, "target_id", filter.TargetID)
	conditions, args = appendStringCondition(conditions, args, "target_type", filter.TargetType)
	conditions, args = appendStringCondition(conditions, args, "source_ip", filter.SourceIP)
	conditions, args = appendStringCondition(conditions, args, "correlation_id", filter.CorrelationID)
	conditions, args = appendStringCondition(conditions, args, "request_id", filter.RequestID)

	// Time range conditions
	if filter.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filter.StartTime)
	}
	if filter.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filter.EndTime)
	}

	// Search text condition
	if filter.SearchText != "" {
		conditions = append(conditions, "(LOWER(description) LIKE ? OR LOWER(action) LIKE ?)")
		searchPattern := "%" + strings.ToLower(filter.SearchText) + "%"
		args = append(args, searchPattern, searchPattern)
	}

	return conditions, args
}

// appendStringCondition adds a string equality condition if value is non-empty.
func appendStringCondition(conditions []string, args []interface{}, column, value string) ([]string, []interface{}) {
	if value != "" {
		conditions = append(conditions, column+" = ?")
		args = append(args, value)
	}
	return conditions, args
}

// getBaseQuery returns the SELECT statement for audit events.
func (s *DuckDBStore) getBaseQuery(countOnly bool) string {
	if countOnly {
		return "SELECT COUNT(*) FROM audit_events"
	}
	// Cast JSON columns to VARCHAR for proper scanning
	return `
		SELECT
			id, timestamp, type, severity, outcome,
			actor_id, actor_type, actor_name,
			CAST(actor_roles AS VARCHAR) as actor_roles,
			actor_session_id, actor_auth_method,
			target_id, target_type, target_name,
			source_ip, source_user_agent, source_hostname, source_port,
			CAST(source_geo AS VARCHAR) as source_geo,
			action, description,
			CAST(metadata AS VARCHAR) as metadata,
			correlation_id, request_id
		FROM audit_events
	`
}

// appendOrderAndLimit adds ORDER BY, LIMIT, and OFFSET clauses.
func (s *DuckDBStore) appendOrderAndLimit(query string, filter QueryFilter) string {
	// ORDER BY with validation
	orderBy := "timestamp"
	validFields := map[string]bool{
		"timestamp": true, "type": true, "severity": true,
		"outcome": true, "actor_id": true, "created_at": true,
	}
	if filter.OrderBy != "" && validFields[filter.OrderBy] {
		orderBy = filter.OrderBy
	}

	if filter.OrderDesc {
		query += fmt.Sprintf(" ORDER BY %s DESC", orderBy)
	} else {
		query += fmt.Sprintf(" ORDER BY %s ASC", orderBy)
	}

	// LIMIT and OFFSET
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	return query
}

// scannedEventData holds raw scanned values from database.
type scannedEventData struct {
	event      Event
	eventType  string
	severity   string
	outcome    string
	actorRoles sql.NullString
	sourceGeo  sql.NullString
	metadata   sql.NullString
	targetID   sql.NullString
	targetType sql.NullString
	targetName sql.NullString
	sourcePort sql.NullInt64
}

// scanDestinations returns pointers to all fields for scanning.
func (d *scannedEventData) scanDestinations() []interface{} {
	return []interface{}{
		&d.event.ID,
		&d.event.Timestamp,
		&d.eventType,
		&d.severity,
		&d.outcome,
		&d.event.Actor.ID,
		&d.event.Actor.Type,
		&d.event.Actor.Name,
		&d.actorRoles,
		&d.event.Actor.SessionID,
		&d.event.Actor.AuthMethod,
		&d.targetID,
		&d.targetType,
		&d.targetName,
		&d.event.Source.IPAddress,
		&d.event.Source.UserAgent,
		&d.event.Source.Hostname,
		&d.sourcePort,
		&d.sourceGeo,
		&d.event.Action,
		&d.event.Description,
		&d.metadata,
		&d.event.CorrelationID,
		&d.event.RequestID,
	}
}

// toEvent converts scanned data to a fully populated Event.
func (d *scannedEventData) toEvent() *Event {
	d.event.Type = EventType(d.eventType)
	d.event.Severity = Severity(d.severity)
	d.event.Outcome = Outcome(d.outcome)

	d.parseActorRoles()
	d.parseSourceGeo()
	d.parseSourcePort()
	d.parseTarget()
	d.parseMetadata()

	return &d.event
}

// parseActorRoles parses actor roles from JSON string.
func (d *scannedEventData) parseActorRoles() {
	if !d.actorRoles.Valid || d.actorRoles.String == "" {
		return
	}
	if err := json.Unmarshal([]byte(d.actorRoles.String), &d.event.Actor.Roles); err != nil {
		logging.Debug().Err(err).Str("roles", d.actorRoles.String).Msg("Failed to parse actor roles JSON")
	}
}

// parseSourceGeo parses source geo from JSON string.
func (d *scannedEventData) parseSourceGeo() {
	if !d.sourceGeo.Valid || d.sourceGeo.String == "" {
		return
	}
	var geo GeoLocation
	if err := json.Unmarshal([]byte(d.sourceGeo.String), &geo); err == nil {
		d.event.Source.Geo = &geo
	}
}

// parseSourcePort parses source port from nullable int.
func (d *scannedEventData) parseSourcePort() {
	if d.sourcePort.Valid {
		d.event.Source.Port = int(d.sourcePort.Int64)
	}
}

// parseTarget sets target if present.
func (d *scannedEventData) parseTarget() {
	if d.targetID.Valid {
		d.event.Target = &Target{
			ID:   d.targetID.String,
			Type: d.targetType.String,
			Name: d.targetName.String,
		}
	}
}

// parseMetadata sets metadata from JSON string.
func (d *scannedEventData) parseMetadata() {
	if d.metadata.Valid && d.metadata.String != "" {
		d.event.Metadata = json.RawMessage(d.metadata.String)
	}
}

// scanEvent scans a single row into an Event.
func (s *DuckDBStore) scanEvent(row *sql.Row) (*Event, error) {
	var data scannedEventData
	if err := row.Scan(data.scanDestinations()...); err != nil {
		return nil, err
	}
	return data.toEvent(), nil
}

// scanEventFromRows scans a row from sql.Rows into an Event.
func (s *DuckDBStore) scanEventFromRows(rows *sql.Rows) (*Event, error) {
	var data scannedEventData
	if err := rows.Scan(data.scanDestinations()...); err != nil {
		return nil, err
	}
	return data.toEvent(), nil
}
