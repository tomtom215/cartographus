// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestManager_RetryWithBackoff(t *testing.T) {
	t.Parallel() // Safe - tests retry logic with isolated function

	cfg := newTestConfigWithRetries(3, 10*time.Millisecond)

	manager := NewManager(nil, nil, nil, cfg, nil)

	// Test successful retry on second attempt
	attemptCount := 0
	err := manager.retryWithBackoff(context.Background(), func() error {
		attemptCount++
		if attemptCount < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success after retry, got error: %v", err)
	}

	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", attemptCount)
	}

	// Test max retries exceeded
	attemptCount = 0
	err = manager.retryWithBackoff(context.Background(), func() error {
		attemptCount++
		return errors.New("persistent failure")
	})

	if err == nil {
		t.Error("Expected error after max retries")
	}

	if attemptCount != cfg.Sync.RetryAttempts {
		t.Errorf("Expected %d attempts, got %d", cfg.Sync.RetryAttempts, attemptCount)
	}

	if !strings.Contains(err.Error(), "max retry attempts reached") {
		t.Errorf("Expected 'max retry attempts reached' error, got: %v", err)
	}
}

func TestManager_RetryWithBackoff_Success(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	manager := NewManager(nil, nil, nil, cfg, nil)

	callCount := 0
	err := manager.retryWithBackoff(context.Background(), func() error {
		callCount++
		if callCount == 1 {
			return errors.New("first attempt fails")
		}
		return nil // Second attempt succeeds
	})

	if err != nil {
		t.Errorf("Expected success after retry, got error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", callCount)
	}
}

func BenchmarkManager_ProcessHistoryRecord(b *testing.B) {
	cfg := newTestConfig()

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ipAddress,
				Latitude:  37.7749,
				Longitude: -122.4194,
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

	manager := NewManager(mockDb, nil, &mockTautulliClient{}, cfg, nil)

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("bench-session"),
		UserID:     intPtr(1),
		User:       "benchuser",
		IPAddress:  "192.168.1.1",
		MediaType:  "movie",
		Title:      "Benchmark Movie",
		Platform:   "Chrome",
		Player:     "Plex Web",
		Started:    time.Now().Unix(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionKey := fmt.Sprintf("bench-session-%d", i)
		record.SessionKey = &sessionKey
		manager.processHistoryRecord(context.Background(), record)
	}
}

// TestManager_ConcurrentTriggerSync tests that concurrent TriggerSync calls don't cause race conditions
