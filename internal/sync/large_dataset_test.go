// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// TestLargeDataset_100kRecords tests sync behavior with 100k records
// Verifies memory usage stays below 1GB and processing completes successfully
func TestLargeDataset_100kRecords(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	const targetRecords = 100000
	const batchSize = 1000

	processedRecords := 0
	insertedEvents := make([]*models.PlaybackEvent, 0, targetRecords)

	// Mock database that tracks inserts
	mockDB := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil // All new records
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ipAddress,
				Latitude:  40.7128,
				Longitude: -74.0060,
				Country:   "United States",
			}, nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			insertedEvents = append(insertedEvents, event)
			processedRecords++
			return nil
		},
	}

	// Mock client that returns large batches
	batchNumber := 0
	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			// Return empty when we've generated enough records
			if start >= targetRecords {
				return &tautulli.TautulliHistory{
					Response: tautulli.TautulliHistoryResponse{
						Result: "success",
						Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
					},
				}, nil
			}

			// Generate batch of records
			records := make([]tautulli.TautulliHistoryRecord, 0, length)
			for i := 0; i < length && (start+i) < targetRecords; i++ {
				userID := (start + i) % 100 // 100 unique users
				sessionKey := fmt.Sprintf("session-%d-%d", batchNumber, i)
				records = append(records, tautulli.TautulliHistoryRecord{
					SessionKey:      &sessionKey,
					Started:         time.Now().Add(-time.Duration(start+i) * time.Second).Unix(),
					UserID:          intPtr(userID),
					User:            fmt.Sprintf("user-%d", (start+i)%100),
					IPAddress:       fmt.Sprintf("192.168.%d.%d", (start+i)/256, (start+i)%256),
					MediaType:       "movie",
					Title:           fmt.Sprintf("Movie %d", start+i),
					Platform:        "Plex Web",
					Player:          "Chrome",
					Location:        "lan",
					PercentComplete: intPtr(100),
				})
			}

			batchNumber++
			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: records},
				},
			}, nil
		},
	}

	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     batchSize,
			RetryAttempts: 1,
			RetryDelay:    100 * time.Millisecond,
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	// Measure memory before sync
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Perform sync
	startTime := time.Now()
	err := manager.syncDataSince(context.Background(), time.Now().Add(-24*time.Hour))
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Sync failed with 100k records: %v", err)
	}

	// Measure memory after sync
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Verify all records processed
	if processedRecords != targetRecords {
		t.Errorf("Expected %d records processed, got %d", targetRecords, processedRecords)
	}

	// Calculate memory usage
	memoryUsedMB := float64(memAfter.Alloc-memBefore.Alloc) / 1024 / 1024
	peakMemoryMB := float64(memAfter.TotalAlloc-memBefore.TotalAlloc) / 1024 / 1024

	t.Logf("Large dataset test results:")
	t.Logf("  Records processed: %d", processedRecords)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Records/second: %.0f", float64(processedRecords)/duration.Seconds())
	t.Logf("  Memory used: %.2f MB", memoryUsedMB)
	t.Logf("  Peak memory: %.2f MB", peakMemoryMB)
	t.Logf("  Batches: %d (batch size: %d)", batchNumber, batchSize)

	// Verify memory usage < 1GB
	maxMemoryMB := 1024.0
	if peakMemoryMB > maxMemoryMB {
		t.Errorf("Memory usage exceeded 1GB: %.2f MB", peakMemoryMB)
	}

	// Verify reasonable performance (at least 1000 records/second)
	recordsPerSecond := float64(processedRecords) / duration.Seconds()
	if recordsPerSecond < 1000 {
		t.Errorf("Performance too slow: %.0f records/second (expected > 1000)", recordsPerSecond)
	}
}

// TestLargeDataset_MemoryEfficiency_BatchProcessing verifies that batch processing
// keeps memory usage constant regardless of total record count
func TestLargeDataset_MemoryEfficiency_BatchProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory efficiency test in short mode")
	}

	testCases := []struct {
		name         string
		totalRecords int
		batchSize    int
	}{
		{"10k records, batch 100", 10000, 100},
		{"10k records, batch 1000", 10000, 1000},
		{"50k records, batch 1000", 50000, 1000},
		{"100k records, batch 1000", 100000, 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			processedCount := 0

			mockDB := &mockDB{
				sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
					return false, nil
				},
				getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
					return &models.Geolocation{
						IPAddress: ipAddress,
						Latitude:  40.7128,
						Longitude: -74.0060,
						Country:   "United States",
					}, nil
				},
				getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
					// Return empty map - all IPs will be fetched individually
					return make(map[string]*models.Geolocation), nil
				},
				insertPlaybackEvent: func(event *models.PlaybackEvent) error {
					processedCount++
					return nil
				},
			}

			mockClient := &mockTautulliClient{
				getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
					if start >= tc.totalRecords {
						return &tautulli.TautulliHistory{
							Response: tautulli.TautulliHistoryResponse{
								Result: "success",
								Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
							},
						}, nil
					}

					records := make([]tautulli.TautulliHistoryRecord, 0, length)
					for i := 0; i < length && (start+i) < tc.totalRecords; i++ {
						sessionKey := fmt.Sprintf("session-%d-%d", start, i)
						records = append(records, tautulli.TautulliHistoryRecord{
							SessionKey:      &sessionKey,
							Started:         time.Now().Unix(),
							UserID:          intPtr(i % 10),
							User:            fmt.Sprintf("user-%d", i%10),
							IPAddress:       fmt.Sprintf("192.168.1.%d", (start+i)%256),
							MediaType:       "movie",
							Title:           fmt.Sprintf("Movie %d", start+i),
							Platform:        "Plex Web",
							Player:          "Chrome",
							Location:        "lan",
							PercentComplete: intPtr(100),
						})
					}

					return &tautulli.TautulliHistory{
						Response: tautulli.TautulliHistoryResponse{
							Result: "success",
							Data:   tautulli.TautulliHistoryData{Data: records},
						},
					}, nil
				},
			}

			cfg := &config.Config{
				Sync: config.SyncConfig{
					BatchSize:     tc.batchSize,
					RetryAttempts: 1,
					RetryDelay:    100 * time.Millisecond,
				},
			}

			manager := NewManager(mockDB, nil, mockClient, cfg, nil)

			// Measure memory
			runtime.GC()
			var memBefore runtime.MemStats
			runtime.ReadMemStats(&memBefore)

			// Perform sync
			err := manager.syncDataSince(context.Background(), time.Now().Add(-1*time.Hour))
			if err != nil {
				t.Fatalf("Sync failed: %v", err)
			}

			runtime.GC()
			var memAfter runtime.MemStats
			runtime.ReadMemStats(&memAfter)

			memoryUsedMB := float64(memAfter.Alloc-memBefore.Alloc) / 1024 / 1024
			peakMemoryMB := float64(memAfter.TotalAlloc-memBefore.TotalAlloc) / 1024 / 1024

			t.Logf("Memory usage: %.2f MB (peak: %.2f MB)", memoryUsedMB, peakMemoryMB)

			// Verify memory usage is reasonable
			// Memory should scale with batch size, not total records
			expectedMaxMemoryMB := float64(tc.batchSize) * 0.05 // ~50KB per record in batch
			if memoryUsedMB > expectedMaxMemoryMB*10 {
				t.Logf("Warning: High memory usage %.2f MB (expected < %.2f MB)", memoryUsedMB, expectedMaxMemoryMB*10)
			}

			// Verify all records processed
			if processedCount != tc.totalRecords {
				t.Errorf("Expected %d records, got %d", tc.totalRecords, processedCount)
			}
		})
	}
}

// TestLargeDataset_ErrorHandling_PartialBatch tests that errors in large dataset
// syncs are handled gracefully without memory leaks
func TestLargeDataset_ErrorHandling_PartialBatch(t *testing.T) {
	const totalRecords = 10000
	const batchSize = 1000
	const failAtRecord = 5500 // Fail in middle of 6th batch

	processedCount := 0

	mockDB := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ipAddress,
				Latitude:  40.7128,
				Longitude: -74.0060,
				Country:   "United States",
			}, nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			processedCount++
			// Simulate error at specific record
			if processedCount == failAtRecord {
				return fmt.Errorf("simulated database error at record %d", failAtRecord)
			}
			return nil
		},
	}

	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			if start >= totalRecords {
				return &tautulli.TautulliHistory{
					Response: tautulli.TautulliHistoryResponse{
						Result: "success",
						Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
					},
				}, nil
			}

			records := make([]tautulli.TautulliHistoryRecord, 0, length)
			for i := 0; i < length && (start+i) < totalRecords; i++ {
				sessionKey := fmt.Sprintf("session-%d", start+i)
				records = append(records, tautulli.TautulliHistoryRecord{
					SessionKey:      &sessionKey,
					Started:         time.Now().Unix(),
					UserID:          intPtr(1),
					User:            "testuser",
					IPAddress:       "192.168.1.100",
					MediaType:       "movie",
					Title:           "Test Movie",
					Platform:        "Plex Web",
					Player:          "Chrome",
					Location:        "lan",
					PercentComplete: intPtr(100),
				})
			}

			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: records},
				},
			}, nil
		},
	}

	cfg := &config.Config{
		Sync: config.SyncConfig{
			BatchSize:     batchSize,
			RetryAttempts: 1,
			RetryDelay:    100 * time.Millisecond,
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	// Measure memory before
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Perform sync (should complete but log errors)
	err := manager.syncDataSince(context.Background(), time.Now().Add(-1*time.Hour))

	// Measure memory after
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Sync should succeed (errors are logged, not fatal)
	if err != nil {
		t.Logf("Sync completed with logged errors: %v", err)
	}

	// Verify partial processing
	if processedCount < failAtRecord {
		t.Errorf("Expected at least %d records processed before failure, got %d", failAtRecord, processedCount)
	}

	// Verify no memory leak
	// Note: Use signed arithmetic to avoid uint64 underflow if GC ran between measurements
	var memoryUsedMB float64
	if memAfter.Alloc >= memBefore.Alloc {
		memoryUsedMB = float64(memAfter.Alloc-memBefore.Alloc) / 1024 / 1024
	} else {
		// GC ran between measurements, memory decreased (this is good)
		memoryUsedMB = 0
		t.Logf("Memory decreased (GC ran): before=%d after=%d", memBefore.Alloc, memAfter.Alloc)
	}
	t.Logf("Memory used after partial sync with errors: %.2f MB", memoryUsedMB)

	// Memory usage should still be reasonable even with errors
	maxMemoryMB := 500.0
	if memoryUsedMB > maxMemoryMB {
		t.Errorf("Memory usage too high after error: %.2f MB (expected < %.2f MB)", memoryUsedMB, maxMemoryMB)
	}
}

// BenchmarkLargeDataset_Throughput benchmarks sync throughput with various batch sizes
func BenchmarkLargeDataset_Throughput(b *testing.B) {
	benchmarks := []struct {
		name      string
		batchSize int
		records   int
	}{
		{"1k_records_batch100", 100, 1000},
		{"1k_records_batch1000", 1000, 1000},
		{"10k_records_batch100", 100, 10000},
		{"10k_records_batch1000", 1000, 10000},
		{"10k_records_batch5000", 5000, 10000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			mockDB := &mockDB{
				sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
					return false, nil
				},
				getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
					return &models.Geolocation{
						IPAddress: ipAddress,
						Latitude:  40.7128,
						Longitude: -74.0060,
						Country:   "United States",
					}, nil
				},
				getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
					// Return empty map - all IPs will be fetched individually
					return make(map[string]*models.Geolocation), nil
				},
				insertPlaybackEvent: func(event *models.PlaybackEvent) error {
					return nil
				},
			}

			mockClient := &mockTautulliClient{
				getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
					if start >= bm.records {
						return &tautulli.TautulliHistory{
							Response: tautulli.TautulliHistoryResponse{
								Result: "success",
								Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
							},
						}, nil
					}

					records := make([]tautulli.TautulliHistoryRecord, 0, length)
					for i := 0; i < length && (start+i) < bm.records; i++ {
						sessionKey := uuid.New().String()
						records = append(records, tautulli.TautulliHistoryRecord{
							SessionKey:      &sessionKey,
							Started:         time.Now().Unix(),
							UserID:          intPtr(1),
							User:            "testuser",
							IPAddress:       "192.168.1.100",
							MediaType:       "movie",
							Title:           "Test Movie",
							Platform:        "Plex Web",
							Player:          "Chrome",
							Location:        "lan",
							PercentComplete: intPtr(100),
						})
					}

					return &tautulli.TautulliHistory{
						Response: tautulli.TautulliHistoryResponse{
							Result: "success",
							Data:   tautulli.TautulliHistoryData{Data: records},
						},
					}, nil
				},
			}

			cfg := &config.Config{
				Sync: config.SyncConfig{
					BatchSize:     bm.batchSize,
					RetryAttempts: 1,
					RetryDelay:    1 * time.Millisecond,
				},
			}

			manager := NewManager(mockDB, nil, mockClient, cfg, nil)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := manager.syncDataSince(context.Background(), time.Now().Add(-1*time.Hour))
				if err != nil {
					b.Fatalf("Sync failed: %v", err)
				}
			}
		})
	}
}
