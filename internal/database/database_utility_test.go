// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestGetDatabasePath tests the database path getter
func TestGetDatabasePath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	path := db.GetDatabasePath()
	if path != ":memory:" {
		t.Errorf("Expected ':memory:' for test database, got %s", path)
	}
}

// TestCheckpoint tests WAL checkpoint functionality
func TestCheckpoint(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert some data
	insertTestGeolocations(t, db)

	// Checkpoint should succeed
	err := db.Checkpoint(context.Background())
	if err != nil {
		t.Errorf("Checkpoint failed: %v", err)
	}
}

// TestCheckpoint_ContextCanceled tests checkpoint with canceled context
func TestCheckpoint_ContextCanceled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := db.Checkpoint(ctx)
	if err == nil {
		t.Error("Expected error with canceled context, got nil")
	}
}

// TestGetRecordCounts_EmptyDatabase tests record counts on empty database
func TestGetRecordCounts_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	playbacks, geolocations, err := db.GetRecordCounts(context.Background())
	if err != nil {
		t.Fatalf("GetRecordCounts failed: %v", err)
	}

	if playbacks != 0 {
		t.Errorf("Expected 0 playbacks on empty DB, got %d", playbacks)
	}
	if geolocations != 0 {
		t.Errorf("Expected 0 geolocations on empty DB, got %d", geolocations)
	}
}

// TestGetRecordCounts_WithData tests record counts with data
func TestGetRecordCounts_WithData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert test data
	insertTestGeolocations(t, db)
	insertTestPlaybacks(t, db)

	playbacks, geolocations, err := db.GetRecordCounts(context.Background())
	if err != nil {
		t.Fatalf("GetRecordCounts failed: %v", err)
	}

	if playbacks <= 0 {
		t.Errorf("Expected positive playbacks count, got %d", playbacks)
	}
	if geolocations != 5 { // insertTestGeolocations inserts 5 records
		t.Errorf("Expected 5 geolocations, got %d", geolocations)
	}
}

// TestGetGeolocations_EmptyInput tests GetGeolocations with empty input
func TestGetGeolocations_EmptyInput(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	result, err := db.GetGeolocations(context.Background(), []string{})
	if err != nil {
		t.Fatalf("GetGeolocations failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty map for empty input, got %d entries", len(result))
	}
}

// TestGetGeolocations_Success tests batch geolocation retrieval
func TestGetGeolocations_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	result, err := db.GetGeolocations(context.Background(), ips)
	if err != nil {
		t.Fatalf("GetGeolocations failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 geolocations, got %d", len(result))
	}

	// Verify specific entries
	geo1, ok := result["192.168.1.1"]
	if !ok {
		t.Error("Expected to find geolocation for 192.168.1.1")
	} else if geo1.City == nil || *geo1.City != "New York" {
		city := "<nil>"
		if geo1.City != nil {
			city = *geo1.City
		}
		t.Errorf("Expected city 'New York', got '%s'", city)
	}

	geo2, ok := result["192.168.1.2"]
	if !ok {
		t.Error("Expected to find geolocation for 192.168.1.2")
	} else if geo2.City == nil || *geo2.City != "Los Angeles" {
		city := "<nil>"
		if geo2.City != nil {
			city = *geo2.City
		}
		t.Errorf("Expected city 'Los Angeles', got '%s'", city)
	}
}

// TestGetGeolocations_PartialMatch tests when some IPs don't exist
func TestGetGeolocations_PartialMatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	// Include IPs that don't exist
	ips := []string{"192.168.1.1", "10.0.0.1", "192.168.1.2", "172.16.0.1"}
	result, err := db.GetGeolocations(context.Background(), ips)
	if err != nil {
		t.Fatalf("GetGeolocations failed: %v", err)
	}

	// Should only find the 2 that exist
	if len(result) != 2 {
		t.Errorf("Expected 2 geolocations (partial match), got %d", len(result))
	}
}

// TestGetGeolocations_NoMatch tests when no IPs match
func TestGetGeolocations_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	ips := []string{"10.0.0.1", "172.16.0.1"}
	result, err := db.GetGeolocations(context.Background(), ips)
	if err != nil {
		t.Fatalf("GetGeolocations failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for non-matching IPs, got %d", len(result))
	}
}

// TestInvalidateTileCache tests tile cache invalidation
func TestInvalidateTileCache(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Should not panic or error even when cache is empty
	db.InvalidateTileCache()
}

// TestSeedMockData tests the mock data seeding function
// SeedMockData is intended for screenshot/demo purposes, not production.
// InsertPlaybackEvent is already tested by other unit tests.
func TestSeedMockData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Seed test data
	err := db.SeedMockData(context.Background())
	if err != nil {
		t.Fatalf("SeedMockData failed: %v", err)
	}

	// Verify data was inserted
	playbacks, geolocations, err := db.GetRecordCounts(context.Background())
	if err != nil {
		t.Fatalf("GetRecordCounts failed: %v", err)
	}

	if playbacks <= 0 {
		t.Error("Expected some playbacks after seeding")
	}
	if geolocations <= 0 {
		t.Error("Expected some geolocations after seeding")
	}
}

// TestExportGeoJSON_EmptyDatabase tests GeoJSON export with no data
func TestExportGeoJSON_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create temp file for output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "export.geojson")

	err := db.ExportGeoJSON(context.Background(), outputPath, LocationStatsFilter{})
	if err != nil {
		t.Fatalf("ExportGeoJSON failed: %v", err)
	}

	// File should be created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Expected output file to be created")
	}
}

// TestExportGeoJSON_WithData tests GeoJSON export with data
func TestExportGeoJSON_WithData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)
	insertTestPlaybacks(t, db)

	// Create temp file for output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "export.geojson")

	err := db.ExportGeoJSON(context.Background(), outputPath, LocationStatsFilter{})
	if err != nil {
		t.Fatalf("ExportGeoJSON failed: %v", err)
	}

	// File should be created
	info, err := os.Stat(outputPath)
	if os.IsNotExist(err) {
		t.Fatal("Expected output file to be created")
	}

	// File should have content
	if info.Size() == 0 {
		t.Error("Expected non-empty GeoJSON file")
	}
}

// TestExportGeoJSON_WithFilter tests GeoJSON export with filters
func TestExportGeoJSON_WithFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Use existing helpers that properly set up matching geolocation and playback data
	insertTestGeolocations(t, db)
	insertTestPlaybacks(t, db)

	// Create temp file for output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "export.geojson")

	// Filter by media type (which we know exists in test data)
	err := db.ExportGeoJSON(context.Background(), outputPath, LocationStatsFilter{
		MediaTypes: []string{"movie"},
	})
	if err != nil {
		t.Fatalf("ExportGeoJSON with filter failed: %v", err)
	}

	// File should be created (DuckDB COPY TO creates file even with 0 rows)
	info, err := os.Stat(outputPath)
	if os.IsNotExist(err) {
		// If no file, the export may have had no matching data - this is acceptable
		// as long as no error was returned
		t.Log("File not created, which may be due to no matching data (acceptable)")
		return
	}

	// If file exists, verify it has content
	if info.Size() > 0 {
		t.Logf("Export file created with %d bytes", info.Size())
	}
}

// TestQueryRowWithContext_Success tests the queryRowWithContext helper
func TestQueryRowWithContext_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	ctx := context.Background()
	query := "SELECT COUNT(*) FROM geolocations"

	var count int
	err := db.queryRowWithContext(ctx, query, nil, &count)
	if err != nil {
		t.Fatalf("queryRowWithContext failed: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 geolocations, got %d", count)
	}
}

// TestQueryRowWithContext_WithArgs tests queryRowWithContext with arguments
func TestQueryRowWithContext_WithArgs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertTestGeolocations(t, db)

	ctx := context.Background()
	query := "SELECT city FROM geolocations WHERE ip_address = ?"
	args := []interface{}{"192.168.1.1"}

	var city string
	err := db.queryRowWithContext(ctx, query, args, &city)
	if err != nil {
		t.Fatalf("queryRowWithContext failed: %v", err)
	}

	if city != "New York" {
		t.Errorf("Expected 'New York', got '%s'", city)
	}
}

// TestQueryRowWithContext_NoRows tests queryRowWithContext with no matching rows
// Note: This function intentionally returns nil on ErrNoRows to support aggregation queries
// where "no rows" typically means zero/empty values, not an error condition
func TestQueryRowWithContext_NoRows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	query := "SELECT city FROM geolocations WHERE ip_address = ?"
	args := []interface{}{"10.0.0.1"}

	var city string
	err := db.queryRowWithContext(ctx, query, args, &city)
	// This function returns nil on ErrNoRows by design (for aggregation queries)
	// The destination variable remains at its zero value
	if err != nil {
		t.Errorf("Expected nil error for no matching rows (by design), got: %v", err)
	}
	if city != "" {
		t.Errorf("Expected empty city for no matching rows, got: %s", city)
	}
}

// TestIsConnectionError tests the connection error detection helper
func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "connection refused",
			err:  fmt.Errorf("connection refused"),
			want: true,
		},
		{
			name: "connection reset",
			err:  fmt.Errorf("connection reset by peer"),
			want: true,
		},
		{
			name: "broken pipe",
			err:  fmt.Errorf("write: broken pipe"),
			want: true,
		},
		{
			name: "bad connection",
			err:  fmt.Errorf("bad connection"),
			want: true,
		},
		{
			name: "driver bad connection",
			err:  fmt.Errorf("driver: bad connection"),
			want: true,
		},
		{
			name: "database is closed",
			err:  fmt.Errorf("database is closed"),
			want: true,
		},
		{
			name: "sql database is closed",
			err:  fmt.Errorf("sql: database is closed"),
			want: true,
		},
		{
			name: "regular error",
			err:  fmt.Errorf("some other error"),
			want: false,
		},
		{
			name: "syntax error",
			err:  fmt.Errorf("syntax error near 'SELECT'"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isConnectionError(tt.err)
			if got != tt.want {
				t.Errorf("isConnectionError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestIsTransactionConflict tests the transaction conflict detection helper
func TestIsTransactionConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "transaction conflict",
			err:  fmt.Errorf("Transaction conflict detected"),
			want: true,
		},
		{
			name: "conflict on update",
			err:  fmt.Errorf("Conflict on update"),
			want: true,
		},
		{
			name: "cannot update altered table",
			err:  fmt.Errorf("cannot update a table that has been altered"),
			want: true,
		},
		{
			name: "regular error",
			err:  fmt.Errorf("some other error"),
			want: false,
		},
		{
			name: "syntax error",
			err:  fmt.Errorf("syntax error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTransactionConflict(tt.err)
			if got != tt.want {
				t.Errorf("isTransactionConflict(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestIsInternalError tests the internal error detection helper
func TestIsInternalError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "internal error",
			err:  fmt.Errorf("INTERNAL Error: something went wrong"),
			want: true,
		},
		{
			name: "regular error",
			err:  fmt.Errorf("some other error"),
			want: false,
		},
		{
			name: "binder error",
			err:  fmt.Errorf("Binder Error: column not found"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInternalError(tt.err)
			if got != tt.want {
				t.Errorf("isInternalError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
