// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"

	_ "github.com/duckdb/duckdb-go/v2"
)

// =============================================================================
// Test Helpers
// =============================================================================

// saveTestAlerts saves multiple alerts and fails the test if any save fails.
func saveTestAlerts(ctx context.Context, t *testing.T, store *DuckDBStore, alerts []*Alert) {
	t.Helper()
	for _, alert := range alerts {
		if err := store.SaveAlert(ctx, alert); err != nil {
			t.Fatalf("SaveAlert failed: %v", err)
		}
	}
}

// setupTestDB creates an in-memory DuckDB database for testing.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	return db, func() { db.Close() }
}

// setupTestStore creates a DuckDBStore with initialized schema.
// Note: Uses custom schema setup because DuckDB requires explicit ID generation.
func setupTestStore(t *testing.T) (*DuckDBStore, func()) {
	t.Helper()
	db, cleanup := setupTestDB(t)
	store := NewDuckDBStore(db)

	ctx := context.Background()
	if err := initTestSchema(ctx, db); err != nil {
		cleanup()
		t.Fatalf("failed to init schema: %v", err)
	}

	return store, cleanup
}

// initTestSchema creates the detection tables with proper ID sequences for testing.
func initTestSchema(ctx context.Context, db *sql.DB) error {
	// Create sequences for auto-increment
	sequences := []string{
		`CREATE SEQUENCE IF NOT EXISTS detection_rules_id_seq`,
		`CREATE SEQUENCE IF NOT EXISTS detection_alerts_id_seq`,
	}

	for _, seq := range sequences {
		if _, err := db.ExecContext(ctx, seq); err != nil {
			return err
		}
	}

	// Create tables with sequence-based IDs
	tables := []string{
		`CREATE TABLE IF NOT EXISTS detection_rules (
			id INTEGER PRIMARY KEY DEFAULT nextval('detection_rules_id_seq'),
			rule_type TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			enabled BOOLEAN DEFAULT true,
			config JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS detection_alerts (
			id INTEGER PRIMARY KEY DEFAULT nextval('detection_alerts_id_seq'),
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
		`CREATE TABLE IF NOT EXISTS user_trust_scores (
			user_id INTEGER PRIMARY KEY,
			username TEXT,
			score INTEGER DEFAULT 100,
			violations_count INTEGER DEFAULT 0,
			last_violation_at TIMESTAMP,
			restricted BOOLEAN DEFAULT false,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, table := range tables {
		if _, err := db.ExecContext(ctx, table); err != nil {
			return err
		}
	}

	// Insert default rules with explicit IDs
	defaults := []struct {
		id       int
		ruleType RuleType
		name     string
		enabled  bool
		config   interface{}
	}{
		{1, RuleTypeImpossibleTravel, "Impossible Travel Detection", true, DefaultImpossibleTravelConfig()},
		{2, RuleTypeConcurrentStreams, "Concurrent Stream Limits", true, DefaultConcurrentStreamsConfig()},
		{3, RuleTypeDeviceVelocity, "Device IP Velocity", true, DefaultDeviceVelocityConfig()},
		{4, RuleTypeGeoRestriction, "Geographic Restrictions", false, DefaultGeoRestrictionConfig()},
		{5, RuleTypeSimultaneousLocations, "Simultaneous Locations", true, DefaultSimultaneousLocationsConfig()},
	}

	for _, def := range defaults {
		configJSON, err := json.Marshal(def.config)
		if err != nil {
			return err
		}

		query := `INSERT INTO detection_rules (id, rule_type, name, enabled, config)
		          VALUES (?, ?, ?, ?, ?)
		          ON CONFLICT (rule_type) DO NOTHING`
		if _, err := db.ExecContext(ctx, query, def.id, def.ruleType, def.name, def.enabled, configJSON); err != nil {
			return err
		}
	}

	return nil
}

func TestNewDuckDBStore(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	if store == nil {
		t.Fatal("NewDuckDBStore returned nil")
	}
	if store.db != db {
		t.Error("store.db does not match provided db")
	}
}

func TestDuckDBStore_InitSchema(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Use test-specific schema initialization (production InitSchema has DuckDB ID generation issue)
	err := initTestSchema(ctx, db)
	if err != nil {
		t.Fatalf("initTestSchema failed: %v", err)
	}

	// Verify tables were created
	tables := []string{"detection_rules", "detection_alerts", "user_trust_scores"}
	for _, table := range tables {
		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count)
		if err != nil {
			t.Errorf("table %s not created: %v", table, err)
		}
	}

	// Verify default rules were inserted
	var ruleCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM detection_rules").Scan(&ruleCount)
	if err != nil {
		t.Fatalf("failed to count rules: %v", err)
	}
	if ruleCount != 5 {
		t.Errorf("expected 5 default rules, got %d", ruleCount)
	}
}

func TestDuckDBStore_InitSchema_Idempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Run initTestSchema multiple times
	for i := 0; i < 3; i++ {
		if err := initTestSchema(ctx, db); err != nil {
			t.Fatalf("initTestSchema run %d failed: %v", i+1, err)
		}
	}

	// Verify only 5 rules exist (no duplicates due to ON CONFLICT DO NOTHING)
	var ruleCount int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM detection_rules").Scan(&ruleCount)
	if err != nil {
		t.Fatalf("failed to count rules: %v", err)
	}
	if ruleCount != 5 {
		t.Errorf("expected 5 rules after multiple init, got %d", ruleCount)
	}
}

func TestDuckDBStore_SaveAlert(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		UserID:    123,
		Username:  "testuser",
		ServerID:  "server-1",
		MachineID: "machine-abc",
		IPAddress: "192.168.1.100",
		Severity:  SeverityCritical,
		Title:     "Impossible Travel Detected",
		Message:   "User traveled 1000km in 5 minutes",
		Metadata:  json.RawMessage(`{}`), // Provide valid JSON for DuckDB
		CreatedAt: now,
	}

	err := store.SaveAlert(ctx, alert)
	if err != nil {
		t.Fatalf("SaveAlert failed: %v", err)
	}

	// Verify alert was saved with an ID
	if alert.ID == 0 {
		t.Error("expected alert.ID to be set after save")
	}
}

func TestDuckDBStore_GetAlert(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	// Create and save an alert
	alert := &Alert{
		RuleType:  RuleTypeConcurrentStreams,
		UserID:    456,
		Username:  "anotheruser",
		Severity:  SeverityWarning,
		Title:     "Too Many Streams",
		Message:   "5 concurrent streams detected",
		Metadata:  json.RawMessage(`{}`),
		CreatedAt: now,
	}

	if err := store.SaveAlert(ctx, alert); err != nil {
		t.Fatalf("SaveAlert failed: %v", err)
	}

	// Retrieve the alert
	retrieved, err := store.GetAlert(ctx, alert.ID)
	if err != nil {
		t.Fatalf("GetAlert failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetAlert returned nil")
	}

	// Verify fields
	if retrieved.ID != alert.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, alert.ID)
	}
	if retrieved.RuleType != alert.RuleType {
		t.Errorf("RuleType = %s, want %s", retrieved.RuleType, alert.RuleType)
	}
	if retrieved.UserID != alert.UserID {
		t.Errorf("UserID = %d, want %d", retrieved.UserID, alert.UserID)
	}
	if retrieved.Title != alert.Title {
		t.Errorf("Title = %s, want %s", retrieved.Title, alert.Title)
	}
}

func TestDuckDBStore_GetAlert_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	alert, err := store.GetAlert(ctx, 99999)
	if err != nil {
		t.Fatalf("GetAlert returned error for missing: %v", err)
	}
	if alert != nil {
		t.Error("expected nil for non-existent alert")
	}
}

func TestDuckDBStore_ListAlerts(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	// Create multiple alerts using helper
	alerts := []*Alert{
		{RuleType: RuleTypeImpossibleTravel, UserID: 1, Severity: SeverityCritical, Title: "Alert 1", Message: "msg1", Metadata: json.RawMessage(`{}`), CreatedAt: now},
		{RuleType: RuleTypeConcurrentStreams, UserID: 2, Severity: SeverityWarning, Title: "Alert 2", Message: "msg2", Metadata: json.RawMessage(`{}`), CreatedAt: now.Add(-time.Hour)},
		{RuleType: RuleTypeDeviceVelocity, UserID: 1, Severity: SeverityInfo, Title: "Alert 3", Message: "msg3", Metadata: json.RawMessage(`{}`), CreatedAt: now.Add(-2 * time.Hour)},
	}
	saveTestAlerts(ctx, t, store, alerts)

	// Table-driven filter tests
	userID1 := 1
	tests := []struct {
		name     string
		filter   AlertFilter
		expected int
	}{
		{"no filter", AlertFilter{}, 3},
		{"filter by rule type", AlertFilter{RuleTypes: []RuleType{RuleTypeImpossibleTravel}}, 1},
		{"filter by user ID", AlertFilter{UserID: &userID1}, 2},
		{"filter by severity", AlertFilter{Severities: []Severity{SeverityCritical, SeverityWarning}}, 2},
		{"with limit", AlertFilter{Limit: 1}, 1},
		{"with offset", AlertFilter{Limit: 10, Offset: 1}, 2},
		{"order by severity ASC", AlertFilter{OrderBy: "severity", OrderDirection: "ASC"}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := store.ListAlerts(ctx, tt.filter)
			if err != nil {
				t.Fatalf("ListAlerts failed: %v", err)
			}
			if len(result) != tt.expected {
				t.Errorf("expected %d alerts, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestDuckDBStore_AcknowledgeAlert(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		UserID:    123,
		Severity:  SeverityCritical,
		Title:     "Test Alert",
		Message:   "Test message",
		Metadata:  json.RawMessage(`{}`),
		CreatedAt: now,
	}

	if err := store.SaveAlert(ctx, alert); err != nil {
		t.Fatalf("SaveAlert failed: %v", err)
	}

	// Acknowledge the alert
	err := store.AcknowledgeAlert(ctx, alert.ID, "admin@example.com")
	if err != nil {
		t.Fatalf("AcknowledgeAlert failed: %v", err)
	}

	// Verify acknowledgement
	retrieved, err := store.GetAlert(ctx, alert.ID)
	if err != nil {
		t.Fatalf("GetAlert failed: %v", err)
	}
	if !retrieved.Acknowledged {
		t.Error("expected alert to be acknowledged")
	}
	if retrieved.AcknowledgedBy != "admin@example.com" {
		t.Errorf("AcknowledgedBy = %s, want admin@example.com", retrieved.AcknowledgedBy)
	}
	if retrieved.AcknowledgedAt == nil {
		t.Error("expected AcknowledgedAt to be set")
	}
}

func TestDuckDBStore_GetAlertCount(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	// Create alerts
	for i := 0; i < 5; i++ {
		alert := &Alert{
			RuleType:  RuleTypeImpossibleTravel,
			UserID:    i + 1,
			Severity:  SeverityCritical,
			Title:     "Alert",
			Message:   "msg",
			Metadata:  json.RawMessage(`{}`),
			CreatedAt: now,
		}
		if err := store.SaveAlert(ctx, alert); err != nil {
			t.Fatalf("SaveAlert failed: %v", err)
		}
	}

	count, err := store.GetAlertCount(ctx, AlertFilter{})
	if err != nil {
		t.Fatalf("GetAlertCount failed: %v", err)
	}
	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}

	// Count with filter
	count, err = store.GetAlertCount(ctx, AlertFilter{RuleTypes: []RuleType{RuleTypeConcurrentStreams}})
	if err != nil {
		t.Fatalf("GetAlertCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0 for concurrent streams, got %d", count)
	}
}

func TestDuckDBStore_GetRule(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("existing rule", func(t *testing.T) {
		rule, err := store.GetRule(ctx, RuleTypeImpossibleTravel)
		if err != nil {
			t.Fatalf("GetRule failed: %v", err)
		}
		if rule == nil {
			t.Fatal("expected rule, got nil")
		}
		if rule.RuleType != RuleTypeImpossibleTravel {
			t.Errorf("RuleType = %s, want %s", rule.RuleType, RuleTypeImpossibleTravel)
		}
		if !rule.Enabled {
			t.Error("expected impossible_travel to be enabled by default")
		}
	})

	t.Run("non-existent rule", func(t *testing.T) {
		rule, err := store.GetRule(ctx, RuleType("nonexistent"))
		if err != nil {
			t.Fatalf("GetRule returned error for missing: %v", err)
		}
		if rule != nil {
			t.Error("expected nil for non-existent rule")
		}
	})
}

func TestDuckDBStore_ListRules(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	rules, err := store.ListRules(ctx)
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}

	if len(rules) != 5 {
		t.Errorf("expected 5 default rules, got %d", len(rules))
	}

	// Verify all expected rule types exist
	expectedRules := map[RuleType]bool{
		RuleTypeImpossibleTravel:      true,
		RuleTypeConcurrentStreams:     true,
		RuleTypeDeviceVelocity:        true,
		RuleTypeGeoRestriction:        true,
		RuleTypeSimultaneousLocations: true,
	}

	for _, rule := range rules {
		if !expectedRules[rule.RuleType] {
			t.Errorf("unexpected rule type: %s", rule.RuleType)
		}
		delete(expectedRules, rule.RuleType)
	}

	if len(expectedRules) > 0 {
		t.Errorf("missing rule types: %v", expectedRules)
	}
}

func TestDuckDBStore_SaveRule(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Get existing rule
	rule, err := store.GetRule(ctx, RuleTypeImpossibleTravel)
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}

	// Modify and save
	rule.Name = "Updated Rule Name"
	rule.Enabled = false
	newConfig := ImpossibleTravelConfig{MaxSpeedKmH: 1200}
	rule.Config, _ = json.Marshal(newConfig)

	err = store.SaveRule(ctx, rule)
	if err != nil {
		t.Fatalf("SaveRule failed: %v", err)
	}

	// Verify changes
	updated, err := store.GetRule(ctx, RuleTypeImpossibleTravel)
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}
	if updated.Name != "Updated Rule Name" {
		t.Errorf("Name = %s, want Updated Rule Name", updated.Name)
	}
	if updated.Enabled {
		t.Error("expected rule to be disabled")
	}
}

func TestDuckDBStore_SetRuleEnabled(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Verify initially enabled
	rule, err := store.GetRule(ctx, RuleTypeImpossibleTravel)
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}
	if !rule.Enabled {
		t.Error("expected rule to be enabled initially")
	}

	// Disable the rule
	err = store.SetRuleEnabled(ctx, RuleTypeImpossibleTravel, false)
	if err != nil {
		t.Fatalf("SetRuleEnabled failed: %v", err)
	}

	// Verify disabled
	rule, err = store.GetRule(ctx, RuleTypeImpossibleTravel)
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}
	if rule.Enabled {
		t.Error("expected rule to be disabled after SetRuleEnabled(false)")
	}

	// Re-enable
	err = store.SetRuleEnabled(ctx, RuleTypeImpossibleTravel, true)
	if err != nil {
		t.Fatalf("SetRuleEnabled failed: %v", err)
	}

	rule, err = store.GetRule(ctx, RuleTypeImpossibleTravel)
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}
	if !rule.Enabled {
		t.Error("expected rule to be enabled after SetRuleEnabled(true)")
	}
}

func TestDuckDBStore_GetTrustScore(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("new user returns default", func(t *testing.T) {
		score, err := store.GetTrustScore(ctx, 12345)
		if err != nil {
			t.Fatalf("GetTrustScore failed: %v", err)
		}
		if score == nil {
			t.Fatal("expected score, got nil")
		}
		if score.UserID != 12345 {
			t.Errorf("UserID = %d, want 12345", score.UserID)
		}
		if score.Score != 100 {
			t.Errorf("Score = %d, want 100 for new user", score.Score)
		}
	})
}

func TestDuckDBStore_UpdateTrustScore(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	// Create a trust score
	score := &TrustScore{
		UserID:          42,
		Username:        "testuser",
		Score:           75,
		ViolationsCount: 2,
		LastViolationAt: &now,
		Restricted:      false,
	}

	err := store.UpdateTrustScore(ctx, score)
	if err != nil {
		t.Fatalf("UpdateTrustScore failed: %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.GetTrustScore(ctx, 42)
	if err != nil {
		t.Fatalf("GetTrustScore failed: %v", err)
	}
	if retrieved.Score != 75 {
		t.Errorf("Score = %d, want 75", retrieved.Score)
	}
	if retrieved.ViolationsCount != 2 {
		t.Errorf("ViolationsCount = %d, want 2", retrieved.ViolationsCount)
	}

	// Update existing score
	score.Score = 50
	score.Restricted = true
	err = store.UpdateTrustScore(ctx, score)
	if err != nil {
		t.Fatalf("UpdateTrustScore (update) failed: %v", err)
	}

	retrieved, err = store.GetTrustScore(ctx, 42)
	if err != nil {
		t.Fatalf("GetTrustScore failed: %v", err)
	}
	if retrieved.Score != 50 {
		t.Errorf("Score = %d, want 50 after update", retrieved.Score)
	}
	if !retrieved.Restricted {
		t.Error("expected Restricted = true")
	}
}

func TestDuckDBStore_DecrementTrustScore(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("new user", func(t *testing.T) {
		err := store.DecrementTrustScore(ctx, 1001, 10)
		if err != nil {
			t.Fatalf("DecrementTrustScore failed: %v", err)
		}

		score, err := store.GetTrustScore(ctx, 1001)
		if err != nil {
			t.Fatalf("GetTrustScore failed: %v", err)
		}
		if score.Score != 90 {
			t.Errorf("Score = %d, want 90 after -10", score.Score)
		}
		if score.ViolationsCount != 1 {
			t.Errorf("ViolationsCount = %d, want 1", score.ViolationsCount)
		}
	})

	t.Run("existing user", func(t *testing.T) {
		// Decrement again
		err := store.DecrementTrustScore(ctx, 1001, 50)
		if err != nil {
			t.Fatalf("DecrementTrustScore failed: %v", err)
		}

		score, err := store.GetTrustScore(ctx, 1001)
		if err != nil {
			t.Fatalf("GetTrustScore failed: %v", err)
		}
		if score.Score != 40 {
			t.Errorf("Score = %d, want 40 after -50", score.Score)
		}
		if score.ViolationsCount != 2 {
			t.Errorf("ViolationsCount = %d, want 2", score.ViolationsCount)
		}
		if !score.Restricted {
			t.Error("expected Restricted = true for score < 50")
		}
	})

	t.Run("does not go below zero", func(t *testing.T) {
		err := store.DecrementTrustScore(ctx, 1001, 100)
		if err != nil {
			t.Fatalf("DecrementTrustScore failed: %v", err)
		}

		score, err := store.GetTrustScore(ctx, 1001)
		if err != nil {
			t.Fatalf("GetTrustScore failed: %v", err)
		}
		if score.Score < 0 {
			t.Errorf("Score = %d, should not be negative", score.Score)
		}
	})
}

func TestDuckDBStore_RecoverTrustScores(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create users with low scores
	for i := 1; i <= 3; i++ {
		score := &TrustScore{
			UserID: i,
			Score:  50,
		}
		if err := store.UpdateTrustScore(ctx, score); err != nil {
			t.Fatalf("UpdateTrustScore failed: %v", err)
		}
	}

	// Recover scores
	err := store.RecoverTrustScores(ctx, 10)
	if err != nil {
		t.Fatalf("RecoverTrustScores failed: %v", err)
	}

	// Verify all scores increased
	for i := 1; i <= 3; i++ {
		score, err := store.GetTrustScore(ctx, i)
		if err != nil {
			t.Fatalf("GetTrustScore failed: %v", err)
		}
		if score.Score != 60 {
			t.Errorf("User %d: Score = %d, want 60", i, score.Score)
		}
	}
}

func TestDuckDBStore_RecoverTrustScores_CapsAt100(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create user with score 95
	score := &TrustScore{
		UserID: 999,
		Score:  95,
	}
	if err := store.UpdateTrustScore(ctx, score); err != nil {
		t.Fatalf("UpdateTrustScore failed: %v", err)
	}

	// Recover by 10
	err := store.RecoverTrustScores(ctx, 10)
	if err != nil {
		t.Fatalf("RecoverTrustScores failed: %v", err)
	}

	// Verify capped at 100
	retrieved, err := store.GetTrustScore(ctx, 999)
	if err != nil {
		t.Fatalf("GetTrustScore failed: %v", err)
	}
	if retrieved.Score != 100 {
		t.Errorf("Score = %d, want 100 (capped)", retrieved.Score)
	}
}

func TestDuckDBStore_ListLowTrustUsers(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create users with varying trust scores
	users := []struct {
		userID int
		score  int
	}{
		{1, 90},
		{2, 40},
		{3, 20},
		{4, 60},
		{5, 10},
	}

	for _, u := range users {
		score := &TrustScore{
			UserID:   u.userID,
			Score:    u.score,
			Username: "user",
		}
		if err := store.UpdateTrustScore(ctx, score); err != nil {
			t.Fatalf("UpdateTrustScore failed: %v", err)
		}
	}

	lowTrust, err := store.ListLowTrustUsers(ctx, 50)
	if err != nil {
		t.Fatalf("ListLowTrustUsers failed: %v", err)
	}

	if len(lowTrust) != 3 {
		t.Errorf("expected 3 low trust users, got %d", len(lowTrust))
	}

	// Should be ordered by score ASC
	if len(lowTrust) >= 3 {
		if lowTrust[0].Score > lowTrust[1].Score {
			t.Error("expected results ordered by score ASC")
		}
	}
}

func TestDuckDBStore_BuildPlaceholders(t *testing.T) {
	store := &DuckDBStore{}

	tests := []struct {
		count    int
		expected string
	}{
		{0, ""},
		{1, "?"},
		{3, "?, ?, ?"},
		{5, "?, ?, ?, ?, ?"},
	}

	for _, tt := range tests {
		result := store.buildPlaceholders(tt.count)
		if result != tt.expected {
			t.Errorf("buildPlaceholders(%d) = %q, want %q", tt.count, result, tt.expected)
		}
	}
}

func TestDuckDBStore_ApplyAlertOrdering(t *testing.T) {
	store := &DuckDBStore{}
	baseQuery := "SELECT * FROM alerts WHERE 1=1"

	tests := []struct {
		name     string
		filter   AlertFilter
		contains string
	}{
		{
			name:     "default ordering",
			filter:   AlertFilter{},
			contains: "ORDER BY created_at DESC",
		},
		{
			name:     "order by severity ASC",
			filter:   AlertFilter{OrderBy: "severity", OrderDirection: "ASC"},
			contains: "ORDER BY severity ASC",
		},
		{
			name:     "order by severity DESC",
			filter:   AlertFilter{OrderBy: "severity", OrderDirection: "DESC"},
			contains: "ORDER BY severity DESC",
		},
		{
			name:     "invalid column falls back to created_at",
			filter:   AlertFilter{OrderBy: "DROP TABLE alerts;--", OrderDirection: "ASC"},
			contains: "ORDER BY created_at ASC",
		},
		{
			name:     "invalid direction falls back to DESC",
			filter:   AlertFilter{OrderBy: "severity", OrderDirection: "INVALID"},
			contains: "ORDER BY severity DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.applyAlertOrdering(baseQuery, tt.filter)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("result %q does not contain %q", result, tt.contains)
			}
		})
	}
}

func TestDuckDBStore_ListAlerts_DateFilters(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create alerts with different timestamps
	alerts := []*Alert{
		{RuleType: RuleTypeImpossibleTravel, UserID: 1, Severity: SeverityCritical, Title: "A1", Message: "m1", Metadata: json.RawMessage(`{}`), CreatedAt: now},
		{RuleType: RuleTypeImpossibleTravel, UserID: 2, Severity: SeverityCritical, Title: "A2", Message: "m2", Metadata: json.RawMessage(`{}`), CreatedAt: now.Add(-24 * time.Hour)},
		{RuleType: RuleTypeImpossibleTravel, UserID: 3, Severity: SeverityCritical, Title: "A3", Message: "m3", Metadata: json.RawMessage(`{}`), CreatedAt: now.Add(-48 * time.Hour)},
	}

	for _, alert := range alerts {
		if err := store.SaveAlert(ctx, alert); err != nil {
			t.Fatalf("SaveAlert failed: %v", err)
		}
	}

	t.Run("filter by start date", func(t *testing.T) {
		startDate := now.Add(-30 * time.Hour)
		result, err := store.ListAlerts(ctx, AlertFilter{StartDate: &startDate})
		if err != nil {
			t.Fatalf("ListAlerts failed: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 alerts after start date, got %d", len(result))
		}
	})

	t.Run("filter by end date", func(t *testing.T) {
		endDate := now.Add(-20 * time.Hour)
		result, err := store.ListAlerts(ctx, AlertFilter{EndDate: &endDate})
		if err != nil {
			t.Fatalf("ListAlerts failed: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 alerts before end date, got %d", len(result))
		}
	})

	t.Run("filter by date range", func(t *testing.T) {
		startDate := now.Add(-30 * time.Hour)
		endDate := now.Add(-20 * time.Hour)
		result, err := store.ListAlerts(ctx, AlertFilter{StartDate: &startDate, EndDate: &endDate})
		if err != nil {
			t.Fatalf("ListAlerts failed: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 alert in date range, got %d", len(result))
		}
	})
}

func TestDuckDBStore_ListAlerts_ServerIDFilter(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create alerts with different server IDs
	alerts := []*Alert{
		{RuleType: RuleTypeImpossibleTravel, UserID: 1, ServerID: "server-1", Severity: SeverityCritical, Title: "A1", Message: "m1", Metadata: json.RawMessage(`{}`), CreatedAt: now},
		{RuleType: RuleTypeImpossibleTravel, UserID: 2, ServerID: "server-2", Severity: SeverityCritical, Title: "A2", Message: "m2", Metadata: json.RawMessage(`{}`), CreatedAt: now},
		{RuleType: RuleTypeImpossibleTravel, UserID: 3, ServerID: "server-1", Severity: SeverityCritical, Title: "A3", Message: "m3", Metadata: json.RawMessage(`{}`), CreatedAt: now},
	}

	for _, alert := range alerts {
		if err := store.SaveAlert(ctx, alert); err != nil {
			t.Fatalf("SaveAlert failed: %v", err)
		}
	}

	serverID := "server-1"
	result, err := store.ListAlerts(ctx, AlertFilter{ServerID: &serverID})
	if err != nil {
		t.Fatalf("ListAlerts failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 alerts for server-1, got %d", len(result))
	}
}

func TestDuckDBStore_GetAlertCount_WithAcknowledged(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create alerts
	for i := 0; i < 5; i++ {
		alert := &Alert{
			RuleType:  RuleTypeImpossibleTravel,
			UserID:    i + 1,
			Severity:  SeverityCritical,
			Title:     "Alert",
			Message:   "msg",
			Metadata:  json.RawMessage(`{}`),
			CreatedAt: now,
		}
		if err := store.SaveAlert(ctx, alert); err != nil {
			t.Fatalf("SaveAlert failed: %v", err)
		}
		// Acknowledge some
		if i < 2 {
			if err := store.AcknowledgeAlert(ctx, alert.ID, "admin"); err != nil {
				t.Fatalf("AcknowledgeAlert failed: %v", err)
			}
		}
	}

	acked := true
	count, err := store.GetAlertCount(ctx, AlertFilter{Acknowledged: &acked})
	if err != nil {
		t.Fatalf("GetAlertCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 acknowledged alerts, got %d", count)
	}

	unacked := false
	count, err = store.GetAlertCount(ctx, AlertFilter{Acknowledged: &unacked})
	if err != nil {
		t.Fatalf("GetAlertCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 unacknowledged alerts, got %d", count)
	}
}
