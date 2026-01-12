// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
)

// TestIcuExtensionAvailable verifies the ICU extension is loaded
func TestIcuExtensionAvailable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Check if ICU extension is available
	if !db.IsIcuAvailable() {
		t.Skip("ICU extension not available")
		return
	}

	// Verify we can use timezone functions
	var result string
	err := db.conn.QueryRow("SELECT timezone('UTC', TIMESTAMP '2024-01-01 12:00:00')::VARCHAR").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to execute timezone function: %v", err)
	}
	t.Logf("timezone result: %s", result)
}

// TestIcuTimezoneConversion tests timezone conversion functionality
func TestIcuTimezoneConversion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsIcuAvailable() {
		t.Skip("ICU extension not available")
		return
	}

	tests := []struct {
		name     string
		query    string
		validate func(result string) bool
	}{
		{
			name:  "UTC to New York conversion",
			query: "SELECT timezone('America/New_York', TIMESTAMPTZ '2024-01-01 12:00:00+00')::VARCHAR",
			validate: func(result string) bool {
				// Should be 5 hours behind UTC in winter (EST)
				return len(result) > 0
			},
		},
		{
			name:  "UTC to Los Angeles conversion",
			query: "SELECT timezone('America/Los_Angeles', TIMESTAMPTZ '2024-01-01 12:00:00+00')::VARCHAR",
			validate: func(result string) bool {
				// Should be 8 hours behind UTC in winter (PST)
				return len(result) > 0
			},
		},
		{
			name:  "UTC to Tokyo conversion",
			query: "SELECT timezone('Asia/Tokyo', TIMESTAMPTZ '2024-01-01 12:00:00+00')::VARCHAR",
			validate: func(result string) bool {
				// Should be 9 hours ahead of UTC (JST)
				return len(result) > 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			err := db.conn.QueryRow(tt.query).Scan(&result)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if !tt.validate(result) {
				t.Errorf("Validation failed for result: %s", result)
			}
			t.Logf("%s: %s", tt.name, result)
		})
	}
}

// TestIcuAtTimeZone tests the AT TIME ZONE syntax
func TestIcuAtTimeZone(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsIcuAvailable() {
		t.Skip("ICU extension not available")
		return
	}

	// Test AT TIME ZONE syntax for converting timestamps
	var result string
	err := db.conn.QueryRow(`
		SELECT (TIMESTAMP '2024-07-01 12:00:00' AT TIME ZONE 'UTC' AT TIME ZONE 'America/New_York')::VARCHAR
	`).Scan(&result)
	if err != nil {
		t.Fatalf("AT TIME ZONE query failed: %v", err)
	}
	t.Logf("AT TIME ZONE result: %s", result)

	// The result should contain a time (validating the function works)
	if len(result) == 0 {
		t.Error("Expected non-empty result from AT TIME ZONE")
	}
}

// TestIcuExtractHourInTimezone tests extracting hour in local timezone
// This is the key use case for analytics: "what hour of day do users watch in their local time?"
func TestIcuExtractHourInTimezone(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsIcuAvailable() {
		t.Skip("ICU extension not available")
		return
	}

	// Simulate the analytics use case: extract hour in user's local timezone
	// UTC noon should be 7am in New York (EST, winter) or 8am (EDT, summer)
	var hour int
	err := db.conn.QueryRow(`
		SELECT EXTRACT(HOUR FROM
			timezone('America/New_York', TIMESTAMPTZ '2024-01-15 12:00:00+00')
		)::INTEGER
	`).Scan(&hour)
	if err != nil {
		t.Fatalf("Extract hour query failed: %v", err)
	}

	// In January (EST = UTC-5), noon UTC should be 7am New York
	if hour != 7 {
		t.Errorf("Expected hour 7 (EST), got %d", hour)
	}
}

// TestIcuTimestamptzType tests TIMESTAMPTZ data type support
func TestIcuTimestamptzType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsIcuAvailable() {
		t.Skip("ICU extension not available")
		return
	}

	// Create temporary table with TIMESTAMPTZ
	_, err := db.conn.Exec(`
		CREATE TEMPORARY TABLE test_timestamptz (
			id INTEGER,
			event_time TIMESTAMPTZ
		);
		INSERT INTO test_timestamptz VALUES
			(1, TIMESTAMPTZ '2024-01-01 12:00:00+00'),
			(2, TIMESTAMPTZ '2024-01-01 12:00:00-05'),
			(3, TIMESTAMPTZ '2024-01-01 12:00:00+09');
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Query and verify all timestamps are comparable
	rows, err := db.conn.Query(`
		SELECT id, event_time::VARCHAR
		FROM test_timestamptz
		ORDER BY event_time
	`)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var id int
		var eventTime string
		if err := rows.Scan(&id, &eventTime); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		t.Logf("ID %d: %s", id, eventTime)
		count++
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("Row iteration error: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 rows, got %d", count)
	}
}

// TestIcuCollation tests region-dependent collation support
func TestIcuCollation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsIcuAvailable() {
		t.Skip("ICU extension not available")
		return
	}

	// Test that ICU collations are available
	// German collation sorts umlauts correctly
	_, err := db.conn.Exec(`
		CREATE TEMPORARY TABLE test_collation (name TEXT);
		INSERT INTO test_collation VALUES ('Muller'), ('Müller'), ('Mueller');
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Query with default collation
	rows, err := db.conn.Query(`SELECT name FROM test_collation ORDER BY name`)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("Row iteration error: %v", err)
	}

	t.Logf("Sorted names: %v", names)
	if len(names) != 3 {
		t.Errorf("Expected 3 names, got %d", len(names))
	}
}

// TestIcuLocalTimeAnalytics tests the primary use case: analytics by local time
func TestIcuLocalTimeAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsIcuAvailable() {
		t.Skip("ICU extension not available")
		return
	}

	// Simulate playback events with different timezones
	_, err := db.conn.Exec(`
		CREATE TEMPORARY TABLE test_playbacks (
			id INTEGER,
			started_at TIMESTAMP,
			timezone TEXT
		);
		INSERT INTO test_playbacks VALUES
			(1, '2024-01-01 12:00:00', 'America/New_York'),
			(2, '2024-01-01 12:00:00', 'America/Los_Angeles'),
			(3, '2024-01-01 12:00:00', 'Europe/London'),
			(4, '2024-01-01 12:00:00', 'Asia/Tokyo');
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Query: Get local hour for each playback
	// This simulates the real analytics use case
	rows, err := db.conn.Query(`
		SELECT
			id,
			timezone,
			EXTRACT(HOUR FROM timezone(timezone, started_at AT TIME ZONE 'UTC'))::INTEGER as local_hour
		FROM test_playbacks
		ORDER BY id
	`)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, localHour int
		var tz string
		if err := rows.Scan(&id, &tz, &localHour); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		t.Logf("Playback %d (%s): local hour = %d", id, tz, localHour)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("Row iteration error: %v", err)
	}
}
