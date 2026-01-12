// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package supervisor

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

// TestSupervisorTreeIntegration tests the complete supervisor tree behavior
// with multiple services across all layers, simulating a real application.
func TestSupervisorTreeIntegration(t *testing.T) {
	t.Run("full tree with services in all layers", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, err := NewSupervisorTree(logger, TreeConfig{
			FailureThreshold: 5,
			FailureBackoff:   50 * time.Millisecond,
			ShutdownTimeout:  500 * time.Millisecond,
		})
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Create services for all layers
		dataSvc := NewMockService("database")
		wsSvc := NewMockService("websocket-hub")
		syncSvc := NewMockService("sync-manager")
		httpSvc := NewMockService("http-server")

		// Add services to appropriate layers
		tree.AddDataService(dataSvc)
		tree.AddMessagingService(wsSvc)
		tree.AddMessagingService(syncSvc)
		tree.AddAPIService(httpSvc)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		errCh := tree.ServeBackground(ctx)

		// Wait for services to start with polling (more reliable in CI under load)
		var allStarted bool
		for i := 0; i < 10; i++ {
			time.Sleep(20 * time.Millisecond)
			if dataSvc.StartCount() >= 1 && wsSvc.StartCount() >= 1 &&
				syncSvc.StartCount() >= 1 && httpSvc.StartCount() >= 1 {
				allStarted = true
				break
			}
		}

		// Verify all services started
		if !allStarted {
			if dataSvc.StartCount() < 1 {
				t.Error("data service was not started")
			}
			if wsSvc.StartCount() < 1 {
				t.Error("websocket service was not started")
			}
			if syncSvc.StartCount() < 1 {
				t.Error("sync service was not started")
			}
			if httpSvc.StartCount() < 1 {
				t.Error("http service was not started")
			}
		}

		// Wait for context timeout to trigger shutdown
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("unexpected error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("tree did not shut down")
		}
	})

	t.Run("cascade failure isolation", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, _ := NewSupervisorTree(logger, TreeConfig{
			FailureThreshold: 10,
			FailureBackoff:   10 * time.Millisecond,
			ShutdownTimeout:  500 * time.Millisecond,
		})

		// Create a failing service in messaging layer
		failingSvc := NewMockService("failing-messaging")
		failingSvc.SetFailCount(3) // Fail 3 times then succeed

		// Create stable services in other layers
		stableData := NewMockService("stable-data")
		stableAPI := NewMockService("stable-api")

		tree.AddDataService(stableData)
		tree.AddMessagingService(failingSvc)
		tree.AddAPIService(stableAPI)

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		errCh := tree.ServeBackground(ctx)

		// Wait for restarts to happen
		time.Sleep(150 * time.Millisecond)

		// Failing service should have been restarted at least 3 times
		if failingSvc.StartCount() < 3 {
			t.Errorf("failing service should have been restarted at least 3 times, got %d", failingSvc.StartCount())
		}

		// Other services should still be running (started once)
		if stableData.StartCount() < 1 {
			t.Error("stable data service should have started")
		}
		if stableAPI.StartCount() < 1 {
			t.Error("stable API service should have started")
		}

		// Wait for shutdown
		select {
		case <-errCh:
			// Success
		case <-time.After(2 * time.Second):
			t.Error("tree did not shut down")
		}
	})
}

// TestSupervisorTreeConcurrency tests concurrent operations on the supervisor tree.
func TestSupervisorTreeConcurrency(t *testing.T) {
	t.Run("concurrent service additions are safe", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, _ := NewSupervisorTree(logger, TreeConfig{
			ShutdownTimeout: 500 * time.Millisecond,
		})

		// Add services from multiple goroutines before starting
		done := make(chan struct{})
		for i := 0; i < 10; i++ {
			go func(idx int) {
				svc := NewMockService("concurrent-svc")
				switch idx % 3 {
				case 0:
					tree.AddDataService(svc)
				case 1:
					tree.AddMessagingService(svc)
				case 2:
					tree.AddAPIService(svc)
				}
			}(i)
		}

		// Short delay to let goroutines complete (100ms for CI reliability under load)
		time.Sleep(100 * time.Millisecond)
		close(done)

		// Start and stop the tree to verify no data races
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		errCh := tree.ServeBackground(ctx)

		select {
		case <-errCh:
			// Success
		case <-time.After(2 * time.Second):
			t.Error("tree did not shut down")
		}
	})
}

// TestSupervisorTreeEdgeCases tests edge cases and error conditions.
func TestSupervisorTreeEdgeCases(t *testing.T) {
	t.Run("empty tree starts and stops gracefully", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, _ := NewSupervisorTree(logger, TreeConfig{
			ShutdownTimeout: 500 * time.Millisecond,
		})

		// Don't add any services
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		errCh := tree.ServeBackground(ctx)

		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("unexpected error: %v", err)
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("tree did not shut down")
		}
	})

	t.Run("root accessor returns non-nil", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, _ := NewSupervisorTree(logger, TreeConfig{})

		if tree.Root() == nil {
			t.Error("Root() should return non-nil supervisor")
		}
	})
}
