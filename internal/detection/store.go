// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// DuckDBStore implements AlertStore, RuleStore, TrustStore, and EventHistory
// using DuckDB as the backend storage.
type DuckDBStore struct {
	db *sql.DB
}

// NewDuckDBStore creates a new DuckDB-backed store.
func NewDuckDBStore(db *sql.DB) *DuckDBStore {
	return &DuckDBStore{db: db}
}

// Common SQL query fragments to reduce duplication
const (
	detectionEventSelectColumns = `
		p.session_key,
		p.started_at,
		p.user_id,
		p.username,
		COALESCE(p.friendly_name, '') as friendly_name,
		COALESCE(p.server_id, '') as server_id,
		COALESCE(p.machine_id, '') as machine_id,
		COALESCE(p.platform, '') as platform,
		COALESCE(p.player, '') as player,
		COALESCE(p.device, '') as device,
		p.media_type,
		p.title,
		COALESCE(p.grandparent_title, '') as grandparent_title,
		p.ip_address,
		COALESCE(p.location_type, '') as location_type,
		COALESCE(g.latitude, 0) as latitude,
		COALESCE(g.longitude, 0) as longitude,
		COALESCE(g.city, '') as city,
		COALESCE(g.region, '') as region,
		COALESCE(g.country, '') as country`

	detectionEventFromClause = `FROM playback_events p
		LEFT JOIN geolocations g ON p.ip_address = g.ip_address`
)

// Helper functions to reduce cognitive complexity

// buildServerFilter adds server_id filter clause and argument if serverID is provided.
func buildServerFilter(serverID string, baseArgs []interface{}) (filterClause string, args []interface{}) {
	args = baseArgs
	if serverID != "" {
		filterClause = " AND p.server_id = ?"
		args = append(args, serverID)
	}
	return filterClause, args
}

// scanDetectionEvent scans a single row into a DetectionEvent struct.
func scanDetectionEvent(scanner interface {
	Scan(dest ...interface{}) error
}, event *DetectionEvent) error {
	return scanner.Scan(
		&event.SessionKey,
		&event.Timestamp,
		&event.UserID,
		&event.Username,
		&event.FriendlyName,
		&event.ServerID,
		&event.MachineID,
		&event.Platform,
		&event.Player,
		&event.Device,
		&event.MediaType,
		&event.Title,
		&event.GrandparentTitle,
		&event.IPAddress,
		&event.LocationType,
		&event.Latitude,
		&event.Longitude,
		&event.City,
		&event.Region,
		&event.Country,
	)
}

// scanDetectionEvents scans multiple rows into DetectionEvent structs.
func scanDetectionEvents(rows *sql.Rows) ([]DetectionEvent, error) {
	var events []DetectionEvent
	for rows.Next() {
		var event DetectionEvent
		if err := scanDetectionEvent(rows, &event); err != nil {
			return nil, fmt.Errorf("failed to scan detection event: %w", err)
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// scanAlertRow scans a single alert row with nullable fields handling.
func scanAlertRow(scanner interface {
	Scan(dest ...interface{}) error
}, alert *Alert) error {
	var serverID, acknowledgedBy sql.NullString
	var metadata interface{} // DuckDB returns JSON as map[string]interface{}

	if err := scanner.Scan(
		&alert.ID,
		&alert.RuleType,
		&alert.UserID,
		&alert.Username,
		&serverID,
		&alert.MachineID,
		&alert.IPAddress,
		&alert.Severity,
		&alert.Title,
		&alert.Message,
		&metadata,
		&alert.Acknowledged,
		&acknowledgedBy,
		&alert.AcknowledgedAt,
		&alert.CreatedAt,
	); err != nil {
		return err
	}

	// Handle nullable fields
	if serverID.Valid {
		alert.ServerID = serverID.String
	}
	if acknowledgedBy.Valid {
		alert.AcknowledgedBy = acknowledgedBy.String
	}

	// Convert metadata back to JSON bytes
	if metadata != nil {
		if metadataBytes, err := json.Marshal(metadata); err == nil {
			alert.Metadata = metadataBytes
		}
	}

	return nil
}

// scanRuleRow scans a single rule row with JSON config handling.
func scanRuleRow(scanner interface {
	Scan(dest ...interface{}) error
}, rule *Rule) error {
	var config interface{} // DuckDB returns JSON as map[string]interface{}

	if err := scanner.Scan(
		&rule.ID,
		&rule.RuleType,
		&rule.Name,
		&rule.Enabled,
		&config,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	); err != nil {
		return err
	}

	// Convert config back to JSON bytes
	if config != nil {
		if configBytes, err := json.Marshal(config); err == nil {
			rule.Config = configBytes
		}
	}

	return nil
}

// scanTrustScoreRow scans a single trust score row with nullable fields handling.
func scanTrustScoreRow(scanner interface {
	Scan(dest ...interface{}) error
}, score *TrustScore) error {
	var username sql.NullString // username can be NULL in the database

	if err := scanner.Scan(
		&score.UserID,
		&username,
		&score.Score,
		&score.ViolationsCount,
		&score.LastViolationAt,
		&score.Restricted,
		&score.UpdatedAt,
	); err != nil {
		return err
	}

	// Handle nullable field
	if username.Valid {
		score.Username = username.String
	}

	return nil
}

// InitSchema creates the detection-related tables if they don't exist.
func (s *DuckDBStore) InitSchema(ctx context.Context) error {
	queries := []string{
		// Detection rules configuration
		`CREATE TABLE IF NOT EXISTS detection_rules (
			id INTEGER PRIMARY KEY,
			rule_type TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			enabled BOOLEAN DEFAULT true,
			config JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Detection alerts
		`CREATE TABLE IF NOT EXISTS detection_alerts (
			id INTEGER PRIMARY KEY,
			rule_type TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT,
			server_id TEXT,
			machine_id TEXT,
			ip_address TEXT,
			severity TEXT NOT NULL,
			title TEXT NOT NULL,
			message TEXT NOT NULL,
			metadata JSON,
			acknowledged BOOLEAN DEFAULT false,
			acknowledged_by TEXT,
			acknowledged_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// User trust scores
		`CREATE TABLE IF NOT EXISTS user_trust_scores (
			user_id INTEGER PRIMARY KEY,
			username TEXT,
			score INTEGER DEFAULT 100,
			violations_count INTEGER DEFAULT 0,
			last_violation_at TIMESTAMP,
			restricted BOOLEAN DEFAULT false,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_alerts_user_id ON detection_alerts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_rule_type ON detection_alerts(rule_type)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_severity ON detection_alerts(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON detection_alerts(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_acknowledged ON detection_alerts(acknowledged)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_server_id ON detection_alerts(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_trust_score ON user_trust_scores(score)`,

		// v2.1: Migration - add server_id column to existing tables
		`ALTER TABLE detection_alerts ADD COLUMN IF NOT EXISTS server_id TEXT`,
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	// Insert default rules if not exist
	if err := s.insertDefaultRules(ctx); err != nil {
		return fmt.Errorf("failed to insert default rules: %w", err)
	}

	// Force a checkpoint after creating tables to flush the WAL.
	// This prevents DuckDB WAL replay issues on restart.
	if _, err := s.db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		logging.Warn().Err(err).Msg("Failed to checkpoint after detection schema initialization")
	}

	return nil
}

// insertDefaultRules creates default detection rule configurations.
func (s *DuckDBStore) insertDefaultRules(ctx context.Context) error {
	defaults := []struct {
		ruleType RuleType
		name     string
		enabled  bool
		config   interface{}
	}{
		{RuleTypeImpossibleTravel, "Impossible Travel Detection", true, DefaultImpossibleTravelConfig()},
		{RuleTypeConcurrentStreams, "Concurrent Stream Limits", true, DefaultConcurrentStreamsConfig()},
		{RuleTypeDeviceVelocity, "Device IP Velocity", true, DefaultDeviceVelocityConfig()},
		{RuleTypeGeoRestriction, "Geographic Restrictions", false, DefaultGeoRestrictionConfig()},
		{RuleTypeSimultaneousLocations, "Simultaneous Locations", true, DefaultSimultaneousLocationsConfig()},
		{RuleTypeUserAgentAnomaly, "User Agent Anomaly Detection", true, DefaultUserAgentAnomalyConfig()},
		{RuleTypeVPNUsage, "VPN Usage Detection", true, DefaultVPNUsageConfig()},
	}

	for _, def := range defaults {
		configJSON, err := json.Marshal(def.config)
		if err != nil {
			return fmt.Errorf("failed to marshal config for %s: %w", def.ruleType, err)
		}

		// DuckDB-native: ON CONFLICT (rule_type) DO NOTHING handles unique constraint on rule_type
		query := `INSERT INTO detection_rules (rule_type, name, enabled, config)
		          VALUES (?, ?, ?, ?)
		          ON CONFLICT (rule_type) DO NOTHING`
		if _, err := s.db.ExecContext(ctx, query, def.ruleType, def.name, def.enabled, configJSON); err != nil {
			return fmt.Errorf("failed to insert rule %s: %w", def.ruleType, err)
		}
	}

	return nil
}

// SaveAlert persists a new alert.
func (s *DuckDBStore) SaveAlert(ctx context.Context, alert *Alert) error {
	// Use RETURNING to get the generated ID (DuckDB doesn't support LastInsertId with sequences)
	query := `INSERT INTO detection_alerts
		(rule_type, user_id, username, server_id, machine_id, ip_address, severity, title, message, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`

	// Cast Metadata to []byte to avoid DuckDB driver issue with json.RawMessage
	// (DuckDB rejects json.Marshaler interface but accepts []byte)
	var metadata []byte
	if alert.Metadata != nil {
		metadata = []byte(alert.Metadata)
	}

	err := s.db.QueryRowContext(ctx, query,
		alert.RuleType,
		alert.UserID,
		alert.Username,
		alert.ServerID,
		alert.MachineID,
		alert.IPAddress,
		alert.Severity,
		alert.Title,
		alert.Message,
		metadata,
		alert.CreatedAt,
	).Scan(&alert.ID)
	if err != nil {
		return fmt.Errorf("failed to insert alert: %w", err)
	}

	return nil
}

// GetAlert retrieves an alert by ID.
func (s *DuckDBStore) GetAlert(ctx context.Context, id int64) (*Alert, error) {
	query := `SELECT id, rule_type, user_id, username, server_id, machine_id, ip_address,
		severity, title, message, metadata, acknowledged, acknowledged_by, acknowledged_at, created_at
		FROM detection_alerts WHERE id = ?`

	alert := &Alert{}
	err := scanAlertRow(s.db.QueryRowContext(ctx, query, id), alert)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	return alert, nil
}

// ListAlerts retrieves alerts with optional filtering.
// Security: Query uses parameterized values (?) and ORDER BY columns are whitelisted
// via validAlertOrderColumns map. See buildAlertQuery, applyAlertFilters, applyAlertOrdering.
func (s *DuckDBStore) ListAlerts(ctx context.Context, filter AlertFilter) ([]Alert, error) {
	query, args := s.buildAlertQuery(filter)

	// codeql[go/sql-injection]: False positive - all user values use parameterized queries (?),
	// ORDER BY columns are validated against validAlertOrderColumns whitelist
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	return s.scanAlerts(rows)
}

// buildAlertQuery constructs the SQL query and args for alert filtering.
func (s *DuckDBStore) buildAlertQuery(filter AlertFilter) (string, []interface{}) {
	query := `SELECT id, rule_type, user_id, username, server_id, machine_id, ip_address,
		severity, title, message, metadata, acknowledged, acknowledged_by, acknowledged_at, created_at
		FROM detection_alerts WHERE 1=1`
	args := make([]interface{}, 0)

	// Apply filters
	query, args = s.applyAlertFilters(query, args, filter)

	// Ordering
	query = s.applyAlertOrdering(query, filter)

	// Pagination
	query, args = s.applyAlertPagination(query, args, filter)

	return query, args
}

// applyAlertFilters adds WHERE clauses for alert filtering.
func (s *DuckDBStore) applyAlertFilters(query string, args []interface{}, filter AlertFilter) (string, []interface{}) {
	if len(filter.RuleTypes) > 0 {
		placeholders := s.buildPlaceholders(len(filter.RuleTypes))
		query += fmt.Sprintf(" AND rule_type IN (%s)", placeholders)
		for _, rt := range filter.RuleTypes {
			args = append(args, rt)
		}
	}

	if len(filter.Severities) > 0 {
		placeholders := s.buildPlaceholders(len(filter.Severities))
		query += fmt.Sprintf(" AND severity IN (%s)", placeholders)
		for _, sev := range filter.Severities {
			args = append(args, sev)
		}
	}

	if filter.UserID != nil {
		query += " AND user_id = ?"
		args = append(args, *filter.UserID)
	}

	if filter.ServerID != nil {
		query += " AND server_id = ?"
		args = append(args, *filter.ServerID)
	}

	if filter.Acknowledged != nil {
		query += " AND acknowledged = ?"
		args = append(args, *filter.Acknowledged)
	}

	if filter.StartDate != nil {
		query += " AND created_at >= ?"
		args = append(args, *filter.StartDate)
	}

	if filter.EndDate != nil {
		query += " AND created_at <= ?"
		args = append(args, *filter.EndDate)
	}

	return query, args
}

// applyAlertOrdering adds ORDER BY clause.
// validAlertOrderColumns is a whitelist of columns that can be used for ordering alerts.
// This prevents SQL injection by only allowing known safe column names.
var validAlertOrderColumns = map[string]bool{
	"id":              true,
	"rule_type":       true,
	"user_id":         true,
	"username":        true,
	"server_id":       true,
	"severity":        true,
	"acknowledged":    true,
	"acknowledged_at": true,
	"created_at":      true,
}

func (s *DuckDBStore) applyAlertOrdering(query string, filter AlertFilter) string {
	// Default to created_at if not specified or invalid (prevents SQL injection)
	orderBy := "created_at"
	if filter.OrderBy != "" && validAlertOrderColumns[filter.OrderBy] {
		orderBy = filter.OrderBy
	}

	// Only allow ASC or DESC (case-insensitive, prevents SQL injection)
	orderDir := "DESC"
	if filter.OrderDirection != "" {
		upperDir := strings.ToUpper(filter.OrderDirection)
		if upperDir == "ASC" || upperDir == "DESC" {
			orderDir = upperDir
		}
	}

	return query + fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)
}

// applyAlertPagination adds LIMIT and OFFSET clauses.
func (s *DuckDBStore) applyAlertPagination(query string, args []interface{}, filter AlertFilter) (string, []interface{}) {
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	} else {
		query += " LIMIT 100"
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	return query, args
}

// buildPlaceholders creates a comma-separated string of ? placeholders.
func (s *DuckDBStore) buildPlaceholders(count int) string {
	if count == 0 {
		return ""
	}
	placeholders := "?"
	for i := 1; i < count; i++ {
		placeholders += ", ?"
	}
	return placeholders
}

// scanAlerts scans rows into Alert structs.
func (s *DuckDBStore) scanAlerts(rows *sql.Rows) ([]Alert, error) {
	var alerts []Alert
	for rows.Next() {
		var alert Alert
		if err := scanAlertRow(rows, &alert); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
}

// AcknowledgeAlert marks an alert as acknowledged.
func (s *DuckDBStore) AcknowledgeAlert(ctx context.Context, id int64, acknowledgedBy string) error {
	query := `UPDATE detection_alerts
		SET acknowledged = true, acknowledged_by = ?, acknowledged_at = ?
		WHERE id = ?`

	_, err := s.db.ExecContext(ctx, query, acknowledgedBy, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	return nil
}

// GetAlertCount returns the count of alerts matching the filter.
func (s *DuckDBStore) GetAlertCount(ctx context.Context, filter AlertFilter) (int, error) {
	query := `SELECT COUNT(*) FROM detection_alerts WHERE 1=1`
	args := make([]interface{}, 0)

	// Apply filters (reuse the filter logic)
	query, args = s.applyAlertFilters(query, args, filter)

	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count alerts: %w", err)
	}

	return count, nil
}

// GetRule retrieves a rule by type.
func (s *DuckDBStore) GetRule(ctx context.Context, ruleType RuleType) (*Rule, error) {
	query := `SELECT id, rule_type, name, enabled, config, created_at, updated_at
		FROM detection_rules WHERE rule_type = ?`

	rule := &Rule{}
	err := scanRuleRow(s.db.QueryRowContext(ctx, query, ruleType), rule)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	return rule, nil
}

// ListRules retrieves all rules.
func (s *DuckDBStore) ListRules(ctx context.Context) ([]Rule, error) {
	query := `SELECT id, rule_type, name, enabled, config, created_at, updated_at
		FROM detection_rules ORDER BY id`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	return s.scanRules(rows)
}

// scanRules scans multiple rule rows.
func (s *DuckDBStore) scanRules(rows *sql.Rows) ([]Rule, error) {
	var rules []Rule
	for rows.Next() {
		var rule Rule
		if err := scanRuleRow(rows, &rule); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// SaveRule persists a rule configuration.
func (s *DuckDBStore) SaveRule(ctx context.Context, rule *Rule) error {
	query := `UPDATE detection_rules
		SET name = ?, enabled = ?, config = ?, updated_at = ?
		WHERE rule_type = ?`

	// Cast Config to []byte to avoid DuckDB driver issue with json.RawMessage
	var config []byte
	if rule.Config != nil {
		config = []byte(rule.Config)
	}

	_, err := s.db.ExecContext(ctx, query, rule.Name, rule.Enabled, config, time.Now(), rule.RuleType)
	if err != nil {
		return fmt.Errorf("failed to save rule: %w", err)
	}

	return nil
}

// SetRuleEnabled enables or disables a rule.
func (s *DuckDBStore) SetRuleEnabled(ctx context.Context, ruleType RuleType, enabled bool) error {
	query := `UPDATE detection_rules SET enabled = ?, updated_at = ? WHERE rule_type = ?`
	_, err := s.db.ExecContext(ctx, query, enabled, time.Now(), ruleType)
	if err != nil {
		return fmt.Errorf("failed to set rule enabled: %w", err)
	}

	return nil
}

// GetTrustScore retrieves a user's trust score.
func (s *DuckDBStore) GetTrustScore(ctx context.Context, userID int) (*TrustScore, error) {
	query := `SELECT user_id, username, score, violations_count, last_violation_at, restricted, updated_at
		FROM user_trust_scores WHERE user_id = ?`

	score := &TrustScore{}
	err := scanTrustScoreRow(s.db.QueryRowContext(ctx, query, userID), score)
	if errors.Is(err, sql.ErrNoRows) {
		// Return default trust score for new users
		return &TrustScore{
			UserID:    userID,
			Score:     100,
			UpdatedAt: time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get trust score: %w", err)
	}

	return score, nil
}

// UpdateTrustScore updates a user's trust score.
func (s *DuckDBStore) UpdateTrustScore(ctx context.Context, score *TrustScore) error {
	query := `INSERT INTO user_trust_scores (user_id, username, score, violations_count, last_violation_at, restricted, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			username = EXCLUDED.username,
			score = EXCLUDED.score,
			violations_count = EXCLUDED.violations_count,
			last_violation_at = EXCLUDED.last_violation_at,
			restricted = EXCLUDED.restricted,
			updated_at = EXCLUDED.updated_at`

	_, err := s.db.ExecContext(ctx, query,
		score.UserID,
		score.Username,
		score.Score,
		score.ViolationsCount,
		score.LastViolationAt,
		score.Restricted,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update trust score: %w", err)
	}

	return nil
}

// DecrementTrustScore decreases a user's trust score by the given amount.
func (s *DuckDBStore) DecrementTrustScore(ctx context.Context, userID int, amount int) error {
	// First ensure the user has a record
	now := time.Now()
	query := `INSERT INTO user_trust_scores (user_id, score, violations_count, last_violation_at, restricted, updated_at)
		VALUES (?, 100 - ?, 1, ?, ? < 50, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			score = GREATEST(0, user_trust_scores.score - ?),
			violations_count = user_trust_scores.violations_count + 1,
			last_violation_at = ?,
			restricted = GREATEST(0, user_trust_scores.score - ?) < 50,
			updated_at = ?`

	_, err := s.db.ExecContext(ctx, query,
		userID, amount, now, 100-amount, now, // INSERT values
		amount, now, amount, now, // UPDATE values
	)
	if err != nil {
		return fmt.Errorf("failed to decrement trust score: %w", err)
	}

	return nil
}

// RecoverTrustScores increases all users' trust scores (daily job).
func (s *DuckDBStore) RecoverTrustScores(ctx context.Context, amount int) error {
	query := `UPDATE user_trust_scores
		SET score = LEAST(100, score + ?),
		    restricted = LEAST(100, score + ?) >= 50,
		    updated_at = ?
		WHERE score < 100`

	_, err := s.db.ExecContext(ctx, query, amount, amount, time.Now())
	if err != nil {
		return fmt.Errorf("failed to recover trust scores: %w", err)
	}

	return nil
}

// ListLowTrustUsers returns users with trust scores below threshold.
func (s *DuckDBStore) ListLowTrustUsers(ctx context.Context, threshold int) ([]TrustScore, error) {
	query := `SELECT user_id, username, score, violations_count, last_violation_at, restricted, updated_at
		FROM user_trust_scores WHERE score < ? ORDER BY score ASC`

	rows, err := s.db.QueryContext(ctx, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query low trust users: %w", err)
	}
	defer rows.Close()

	return s.scanTrustScores(rows)
}

// scanTrustScores scans multiple trust score rows.
func (s *DuckDBStore) scanTrustScores(rows *sql.Rows) ([]TrustScore, error) {
	var scores []TrustScore
	for rows.Next() {
		var score TrustScore
		if err := scanTrustScoreRow(rows, &score); err != nil {
			return nil, fmt.Errorf("failed to scan trust score: %w", err)
		}
		scores = append(scores, score)
	}
	return scores, rows.Err()
}

// =============================================================================
// EventHistory Interface Implementation
// =============================================================================

// GetLastEventForUser retrieves the most recent event for a user on a specific server.
// v2.1: Added serverID parameter for multi-server support. Pass empty string for all servers.
func (s *DuckDBStore) GetLastEventForUser(ctx context.Context, userID int, serverID string) (*DetectionEvent, error) {
	serverFilter, args := buildServerFilter(serverID, []interface{}{userID})

	query := fmt.Sprintf(`
		SELECT %s
		%s
		WHERE p.user_id = ?%s
		ORDER BY p.started_at DESC
		LIMIT 1`, detectionEventSelectColumns, detectionEventFromClause, serverFilter)

	event := &DetectionEvent{}
	err := scanDetectionEvent(s.db.QueryRowContext(ctx, query, args...), event)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last event: %w", err)
	}

	return event, nil
}

// GetActiveStreamsForUser retrieves currently active streams for a user on a specific server.
// Active streams are those that started within the last 4 hours and haven't stopped.
// v2.1: Added serverID parameter for multi-server support. Pass empty string for all servers.
func (s *DuckDBStore) GetActiveStreamsForUser(ctx context.Context, userID int, serverID string) ([]DetectionEvent, error) {
	serverFilter, args := buildServerFilter(serverID, []interface{}{userID})

	query := fmt.Sprintf(`
		SELECT %s
		%s
		WHERE p.user_id = ?
		  AND p.stopped_at IS NULL
		  AND p.started_at >= CURRENT_TIMESTAMP - INTERVAL '4 hours'%s
		ORDER BY p.started_at DESC`, detectionEventSelectColumns, detectionEventFromClause, serverFilter)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query active streams: %w", err)
	}
	defer rows.Close()

	return scanDetectionEvents(rows)
}

// GetRecentIPsForDevice retrieves recent IPs for a device within window on a specific server.
// v2.1: Added serverID parameter for multi-server support. Pass empty string for all servers.
func (s *DuckDBStore) GetRecentIPsForDevice(ctx context.Context, machineID string, serverID string, window time.Duration) ([]string, error) {
	windowStart := time.Now().Add(-window)

	// Build query with optional server_id filter (note: this uses playback_events table directly, not the p alias)
	baseArgs := []interface{}{machineID, windowStart}
	serverFilter := ""
	args := baseArgs
	if serverID != "" {
		serverFilter = " AND server_id = ?"
		args = append(args, serverID)
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT ip_address
		FROM playback_events
		WHERE machine_id = ?
		  AND started_at >= ?%s
		ORDER BY started_at DESC`, serverFilter)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent IPs: %w", err)
	}
	defer rows.Close()

	return s.scanIPAddresses(rows)
}

// scanIPAddresses scans IP addresses from query results.
func (s *DuckDBStore) scanIPAddresses(rows *sql.Rows) ([]string, error) {
	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, fmt.Errorf("failed to scan IP: %w", err)
		}
		ips = append(ips, ip)
	}
	return ips, rows.Err()
}

// GetSimultaneousLocations retrieves concurrent sessions at different locations on a specific server.
// v2.1: Added serverID parameter for multi-server support. Pass empty string for all servers.
func (s *DuckDBStore) GetSimultaneousLocations(ctx context.Context, userID int, serverID string, window time.Duration) ([]DetectionEvent, error) {
	windowStart := time.Now().Add(-window)
	serverFilter, args := buildServerFilter(serverID, []interface{}{userID, windowStart})

	query := fmt.Sprintf(`
		SELECT %s
		%s
		WHERE p.user_id = ?
		  AND p.stopped_at IS NULL
		  AND p.started_at >= ?
		  AND g.latitude IS NOT NULL
		  AND g.longitude IS NOT NULL%s
		ORDER BY p.started_at DESC`, detectionEventSelectColumns, detectionEventFromClause, serverFilter)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query simultaneous locations: %w", err)
	}
	defer rows.Close()

	return scanDetectionEvents(rows)
}

// GetGeolocation retrieves geolocation for an IP address.
func (s *DuckDBStore) GetGeolocation(ctx context.Context, ipAddress string) (*Geolocation, error) {
	query := `
		SELECT ip_address, latitude, longitude,
			COALESCE(city, '') as city,
			COALESCE(region, '') as region,
			country
		FROM geolocations
		WHERE ip_address = ?`

	geo := &Geolocation{}
	err := s.db.QueryRowContext(ctx, query, ipAddress).Scan(
		&geo.IPAddress,
		&geo.Latitude,
		&geo.Longitude,
		&geo.City,
		&geo.Region,
		&geo.Country,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get geolocation: %w", err)
	}

	return geo, nil
}
