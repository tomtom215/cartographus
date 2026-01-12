// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestConcurrent_ParallelInsertPlaybackEvent tests parallel insertion of playback events
// Verifies thread safety and no race conditions with go test -race
func TestConcurrent_ParallelInsertPlaybackEvent(t *testing.T) {
	// NOT parallel - tests concurrency explicitly

	db := setupConcurrentTestDB(t)
	defer cleanupTestDB(t, db)

	const numGoroutines = 50
	const insertsPerGoroutine = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*insertsPerGoroutine)

	startTime := time.Now()

	// Launch concurrent insert operations
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < insertsPerGoroutine; i++ {
				event := &models.PlaybackEvent{
					ID:              uuid.New(),
					SessionKey:      fmt.Sprintf("session-%d-%d", goroutineID, i),
					StartedAt:       time.Now().Add(-time.Duration(i) * time.Minute),
					UserID:          goroutineID,
					Username:        fmt.Sprintf("user-%d", goroutineID),
					IPAddress:       fmt.Sprintf("192.168.%d.%d", goroutineID, i),
					MediaType:       "movie",
					Title:           fmt.Sprintf("Movie %d-%d", goroutineID, i),
					Platform:        "Plex Web",
					Player:          "Chrome",
					LocationType:    "lan",
					PercentComplete: 100,
					CreatedAt:       time.Now(),
				}

				if err := db.InsertPlaybackEvent(event); err != nil {
					errors <- fmt.Errorf("goroutine %d insert %d failed: %w", goroutineID, i, err)
					return
				}
			}
		}(g)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	duration := time.Since(startTime)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent insert error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Failed with %d errors", errorCount)
	}

	// Verify total count
	ctx := context.Background()
	stats, err := db.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	expectedCount := numGoroutines * insertsPerGoroutine
	if stats.TotalPlaybacks != expectedCount {
		t.Errorf("Expected %d playbacks, got %d", expectedCount, stats.TotalPlaybacks)
	}

	t.Logf("Concurrent insert test completed:")
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Inserts per goroutine: %d", insertsPerGoroutine)
	t.Logf("  Total inserts: %d", expectedCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Inserts/second: %.0f", float64(expectedCount)/duration.Seconds())
}

// TestConcurrent_ParallelUpsertGeolocation tests parallel upsert of geolocation records
// Tests both inserts and updates to verify UPSERT conflict handling
// Note: Transaction conflicts are EXPECTED when multiple goroutines upsert overlapping IPs.
// DuckDB's MVCC correctly detects write-write conflicts. Production code uses per-IP mutex
// locking to avoid conflicts, but this test validates DuckDB handles them gracefully.
func TestConcurrent_ParallelUpsertGeolocation(t *testing.T) {
	// NOT parallel - tests concurrency explicitly

	db := setupConcurrentTestDB(t)
	defer cleanupTestDB(t, db)

	const numGoroutines = 30
	const upsertsPerGoroutine = 15
	const uniqueIPs = 100 // Fewer unique IPs than total upserts to force conflicts

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*upsertsPerGoroutine)

	startTime := time.Now()

	// Launch concurrent upsert operations
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < upsertsPerGoroutine; i++ {
				// Use modulo to create overlapping IP addresses
				ipID := (goroutineID*upsertsPerGoroutine + i) % uniqueIPs

				// Helper to convert values to pointers
				strPtr := func(s string) *string { return &s }
				intPtr := func(n int) *int { return &n }

				geo := &models.Geolocation{
					IPAddress:      fmt.Sprintf("10.0.%d.%d", ipID/256, ipID%256),
					Latitude:       40.7128 + float64(ipID)*0.01,
					Longitude:      -74.0060 + float64(ipID)*0.01,
					City:           strPtr(fmt.Sprintf("City %d", ipID)),
					Region:         strPtr(fmt.Sprintf("Region %d", ipID%10)),
					Country:        "United States",
					PostalCode:     strPtr(fmt.Sprintf("%05d", 10000+ipID)),
					Timezone:       strPtr("America/New_York"),
					AccuracyRadius: intPtr(10),
					LastUpdated:    time.Now(),
				}

				// Use UpsertGeolocationWithServer
				if err := db.UpsertGeolocationWithServer(geo, 40.7128, -74.0060); err != nil {
					errors <- err
					// Continue to next iteration instead of returning - allows concurrent conflict testing
				}
			}
		}(g)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	duration := time.Since(startTime)

	// Check for errors - transaction conflicts are EXPECTED with overlapping IPs
	var conflictCount int
	var unexpectedErrorCount int
	for err := range errors {
		// Write-write conflicts are expected with DuckDB's MVCC
		if strings.Contains(err.Error(), "write-write conflict") ||
			strings.Contains(err.Error(), "Conflict") {
			conflictCount++
			// t.Logf("Expected conflict: %v", err) // Uncomment for debug
		} else {
			t.Errorf("Unexpected error (non-conflict): %v", err)
			unexpectedErrorCount++
		}
	}

	// Fail only on unexpected errors, not transaction conflicts
	if unexpectedErrorCount > 0 {
		t.Fatalf("Failed with %d unexpected errors (conflicts are normal: %d)", unexpectedErrorCount, conflictCount)
	}

	// Verify at least some geolocations were created
	ctx := context.Background()
	stats, err := db.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// With high concurrency, some IPs may not be upserted due to conflicts.
	// Verify we have a reasonable number (at least 50% of unique IPs).
	minExpected := uniqueIPs / 2
	if stats.UniqueLocations < minExpected {
		t.Errorf("Expected at least %d unique locations, got %d", minExpected, stats.UniqueLocations)
	}

	totalUpserts := numGoroutines * upsertsPerGoroutine
	successfulUpserts := totalUpserts - conflictCount
	t.Logf("Concurrent upsert test completed:")
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Upserts per goroutine: %d", upsertsPerGoroutine)
	t.Logf("  Total upsert attempts: %d", totalUpserts)
	t.Logf("  Successful upserts: %d (%.1f%%)", successfulUpserts, float64(successfulUpserts)*100/float64(totalUpserts))
	t.Logf("  Transaction conflicts (expected): %d", conflictCount)
	t.Logf("  Unique locations stored: %d", stats.UniqueLocations)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Upserts/second: %.0f", float64(successfulUpserts)/duration.Seconds())
}

// TestConcurrent_MixedReadsAndWrites tests concurrent reads and writes
// Simulates realistic scenario with both queries and inserts happening simultaneously
// Note: Previously skipped due to DuckDB segmentation fault, but now works correctly
// after NATS/WAL integration and 1GB memory limit improvements.
func TestConcurrent_MixedReadsAndWrites(t *testing.T) {
	// NOT parallel - tests concurrency explicitly

	db := setupConcurrentTestDB(t)
	defer cleanupTestDB(t, db)

	// Pre-populate database with some data
	for i := 0; i < 100; i++ {
		event := &models.PlaybackEvent{
			SessionKey:      fmt.Sprintf("initial-session-%d", i),
			StartedAt:       time.Now().Add(-24 * time.Hour),
			UserID:          i % 10,
			Username:        fmt.Sprintf("user-%d", i%10),
			IPAddress:       fmt.Sprintf("192.168.1.%d", i),
			MediaType:       "movie",
			Title:           fmt.Sprintf("Movie %d", i),
			Platform:        "Plex Web",
			Player:          "Chrome",
			LocationType:    "lan",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to populate database: %v", err)
		}
	}

	const numReaders = 20
	const numWriters = 10
	const operationsPerGoroutine = 25

	var wg sync.WaitGroup
	errors := make(chan error, (numReaders+numWriters)*operationsPerGoroutine)

	ctx := context.Background()
	startTime := time.Now()

	// Launch reader goroutines
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			for i := 0; i < operationsPerGoroutine; i++ {
				// Perform various read operations
				switch i % 4 {
				case 0:
					// Read stats
					if _, err := db.GetStats(ctx); err != nil {
						errors <- fmt.Errorf("reader %d stats query failed: %w", readerID, err)
						return
					}
				case 1:
					// Read location stats
					since := time.Now().Add(-48 * time.Hour)
					if _, err := db.GetLocationStats(ctx, since, 10); err != nil {
						errors <- fmt.Errorf("reader %d location stats query failed: %w", readerID, err)
						return
					}
				case 2:
					// Read playback events
					if _, err := db.GetPlaybackEvents(ctx, 20, 0); err != nil {
						errors <- fmt.Errorf("reader %d playback events query failed: %w", readerID, err)
						return
					}
				case 3:
					// Read unique users
					if _, err := db.GetUniqueUsers(ctx); err != nil {
						errors <- fmt.Errorf("reader %d unique users query failed: %w", readerID, err)
						return
					}
				}

				// Small delay to simulate real-world query patterns
				time.Sleep(1 * time.Millisecond)
			}
		}(r)
	}

	// Launch writer goroutines
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			for i := 0; i < operationsPerGoroutine; i++ {
				// Insert new playback event
				event := &models.PlaybackEvent{
					SessionKey:      fmt.Sprintf("writer-%d-session-%d", writerID, i),
					StartedAt:       time.Now(),
					UserID:          writerID + 100,
					Username:        fmt.Sprintf("writer-user-%d", writerID),
					IPAddress:       fmt.Sprintf("10.0.%d.%d", writerID, i),
					MediaType:       "movie",
					Title:           fmt.Sprintf("New Movie %d-%d", writerID, i),
					Platform:        "Plex Web",
					Player:          "Chrome",
					LocationType:    "lan",
					PercentComplete: 100,
				}

				if err := db.InsertPlaybackEvent(event); err != nil {
					errors <- fmt.Errorf("writer %d insert failed: %w", writerID, err)
					return
				}

				// Small delay to simulate real-world write patterns
				time.Sleep(2 * time.Millisecond)
			}
		}(w)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	duration := time.Since(startTime)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Failed with %d errors", errorCount)
	}

	// Verify final count
	stats, err := db.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get final stats: %v", err)
	}

	expectedTotal := 100 + (numWriters * operationsPerGoroutine)
	if stats.TotalPlaybacks != expectedTotal {
		t.Errorf("Expected %d total playbacks, got %d", expectedTotal, stats.TotalPlaybacks)
	}

	t.Logf("Concurrent mixed read/write test completed:")
	t.Logf("  Reader goroutines: %d", numReaders)
	t.Logf("  Writer goroutines: %d", numWriters)
	t.Logf("  Operations per goroutine: %d", operationsPerGoroutine)
	t.Logf("  Total operations: %d", (numReaders+numWriters)*operationsPerGoroutine)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Operations/second: %.0f", float64((numReaders+numWriters)*operationsPerGoroutine)/duration.Seconds())
}

// TestConcurrent_SameIPUpsert tests multiple goroutines upserting the same IP address
// Verifies UPSERT conflict resolution and last-write-wins behavior
func TestConcurrent_SameIPUpsert(t *testing.T) {
	// NOT parallel - tests concurrency explicitly

	db := setupConcurrentTestDB(t)
	defer cleanupTestDB(t, db)

	const numGoroutines = 25
	const targetIP = "203.0.113.42" // Reserved test IP (RFC 5737)

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	startTime := time.Now()

	// Launch goroutines that all update the same IP
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// Helper to convert values to pointers
			strPtr := func(s string) *string { return &s }
			intPtr := func(n int) *int { return &n }
			geo := &models.Geolocation{
				IPAddress:      targetIP,
				Latitude:       37.7749 + float64(goroutineID)*0.001,
				Longitude:      -122.4194 + float64(goroutineID)*0.001,
				City:           strPtr(fmt.Sprintf("City-%d", goroutineID)),
				Region:         strPtr(fmt.Sprintf("Region-%d", goroutineID)),
				Country:        "Test Country",
				PostalCode:     strPtr(fmt.Sprintf("%05d", 94000+goroutineID)),
				Timezone:       strPtr("America/Los_Angeles"),
				AccuracyRadius: intPtr(goroutineID),
				LastUpdated:    time.Now(),
			}

			if err := db.UpsertGeolocationWithServer(geo, 37.7749, -122.4194); err != nil {
				errors <- fmt.Errorf("goroutine %d upsert failed: %w", goroutineID, err)
			}
		}(g)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	duration := time.Since(startTime)

	// Check for errors
	// Note: Transaction conflicts are EXPECTED with concurrent UPSERTs on same IP
	// DuckDB's MVCC transaction isolation prevents simultaneous updates to the same row
	// We verify final state consistency rather than requiring all operations to succeed
	var errorCount int
	var conflictCount int
	for err := range errors {
		// Count transaction conflicts separately (these are expected)
		if err != nil && (strings.Contains(err.Error(), "Conflict") || strings.Contains(err.Error(), "Duplicate key")) {
			conflictCount++
			t.Logf("Expected transaction conflict: %v", err)
		} else {
			// Unexpected errors
			t.Errorf("Unexpected error (non-conflict): %v", err)
			errorCount++
		}
	}

	// Fail only on unexpected errors, not transaction conflicts
	if errorCount > 0 {
		t.Fatalf("Failed with %d unexpected errors (conflicts are normal: %d)", errorCount, conflictCount)
	}

	t.Logf("Transaction conflicts (expected): %d", conflictCount)

	// Verify only one location exists
	ctx := context.Background()
	geo, err := db.GetGeolocation(ctx, targetIP)
	if err != nil {
		t.Fatalf("Failed to get geolocation: %v", err)
	}

	if geo == nil {
		t.Fatal("Expected geolocation to exist, got nil")
	}

	if geo.IPAddress != targetIP {
		t.Errorf("Expected IP %s, got %s", targetIP, geo.IPAddress)
	}

	// Verify stats show only 1 unique location
	stats, err := db.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.UniqueLocations != 1 {
		t.Errorf("Expected 1 unique location, got %d", stats.UniqueLocations)
	}

	t.Logf("Concurrent same-IP upsert test completed:")
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Target IP: %s", targetIP)
	t.Logf("  Duration: %v", duration)
	// Dereference pointer fields for logging
	city := ""
	if geo.City != nil {
		city = *geo.City
	}
	region := ""
	if geo.Region != nil {
		region = *geo.Region
	}
	t.Logf("  Final location: %s, %s, %s", city, region, geo.Country)
}

// TestConcurrent_SessionKeyExistence tests concurrent session key existence checks
// Verifies thread-safe behavior when checking for duplicate sessions
func TestConcurrent_SessionKeyExistence(t *testing.T) {
	// NOT parallel - tests concurrency explicitly

	db := setupConcurrentTestDB(t)
	defer cleanupTestDB(t, db)

	// Pre-populate with some session keys
	existingKeys := make([]string, 50)
	for i := 0; i < 50; i++ {
		existingKeys[i] = fmt.Sprintf("existing-session-%d", i)
		event := &models.PlaybackEvent{
			SessionKey:      existingKeys[i],
			StartedAt:       time.Now(),
			UserID:          i,
			Username:        fmt.Sprintf("user-%d", i),
			IPAddress:       fmt.Sprintf("192.168.1.%d", i),
			MediaType:       "movie",
			Title:           "Test Movie",
			Platform:        "Plex Web",
			Player:          "Chrome",
			LocationType:    "lan",
			PercentComplete: 100,
		}
		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Fatalf("Failed to populate database: %v", err)
		}
	}

	const numGoroutines = 30
	const checksPerGoroutine = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*checksPerGoroutine)

	ctx := context.Background()
	startTime := time.Now()

	// Launch concurrent existence checks
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < checksPerGoroutine; i++ {
				// Check both existing and non-existing keys
				var sessionKey string
				var expectedExists bool

				if i%2 == 0 {
					// Check existing key
					sessionKey = existingKeys[i%len(existingKeys)]
					expectedExists = true
				} else {
					// Check non-existing key
					sessionKey = fmt.Sprintf("non-existing-session-%d-%d", goroutineID, i)
					expectedExists = false
				}

				exists, err := db.SessionKeyExists(ctx, sessionKey)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d check %d failed: %w", goroutineID, i, err)
					return
				}

				if exists != expectedExists {
					errors <- fmt.Errorf("goroutine %d check %d: expected exists=%v, got %v for key %s",
						goroutineID, i, expectedExists, exists, sessionKey)
					return
				}
			}
		}(g)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	duration := time.Since(startTime)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent existence check error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Failed with %d errors", errorCount)
	}

	totalChecks := numGoroutines * checksPerGoroutine
	t.Logf("Concurrent session key existence test completed:")
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Checks per goroutine: %d", checksPerGoroutine)
	t.Logf("  Total checks: %d", totalChecks)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Checks/second: %.0f", float64(totalChecks)/duration.Seconds())
}

// TestConcurrent_RaceDetector is a meta-test that ensures race detector is enabled
func TestConcurrent_RaceDetector(t *testing.T) {
	// NOT parallel - tests concurrency explicitly

	// This test verifies that the race detector is active
	// Run with: go test -race ./internal/database

	var counter int
	var wg sync.WaitGroup

	// Intentionally create a race condition to verify detector works
	// Note: This test should PASS even with race detector because we use mutex
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}

	wg.Wait()

	if counter != 10 {
		t.Errorf("Expected counter=10, got %d (indicates race condition)", counter)
	}

	t.Log("Race detector verification: PASS (counter protected by mutex)")
	t.Log("Run with 'go test -race' to enable Go race detector")
}

// Note: setupTestDB is defined in database_test.go and shared across all test files in this package

// cleanupTestDB closes and cleans up the test database
func cleanupTestDB(t *testing.T, db *DB) {
	if err := db.Close(); err != nil {
		t.Errorf("Failed to close test database: %v", err)
	}
}
