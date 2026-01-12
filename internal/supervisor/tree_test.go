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

func TestSupervisorTreeConstruction(t *testing.T) {
	t.Run("creates hierarchical supervisor tree", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, err := NewSupervisorTree(logger, TreeConfig{
			FailureThreshold: 5,
			FailureBackoff:   time.Second,
			ShutdownTimeout:  10 * time.Second,
		})
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		if tree.Root() == nil {
			t.Error("root supervisor should not be nil")
		}
	})

	t.Run("applies default values for zero config", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, err := NewSupervisorTree(logger, TreeConfig{})
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		if tree.config.FailureThreshold != 5.0 {
			t.Errorf("expected default FailureThreshold 5.0, got %f", tree.config.FailureThreshold)
		}
		if tree.config.FailureDecay != 30.0 {
			t.Errorf("expected default FailureDecay 30.0, got %f", tree.config.FailureDecay)
		}
		if tree.config.FailureBackoff != 15*time.Second {
			t.Errorf("expected default FailureBackoff 15s, got %v", tree.config.FailureBackoff)
		}
		if tree.config.ShutdownTimeout != 10*time.Second {
			t.Errorf("expected default ShutdownTimeout 10s, got %v", tree.config.ShutdownTimeout)
		}
	})
}

func TestSupervisorTreeLifecycle(t *testing.T) {
	t.Run("tree starts and stops gracefully", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, err := NewSupervisorTree(logger, TreeConfig{
			FailureThreshold: 5,
			FailureBackoff:   100 * time.Millisecond,
			ShutdownTimeout:  time.Second,
		})
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Add mock services to each layer
		tree.AddDataService(NewMockService("mock-data"))
		tree.AddMessagingService(NewMockService("mock-messaging"))
		tree.AddAPIService(NewMockService("mock-api"))

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			errCh <- tree.Serve(ctx)
		}()

		// Let it run briefly
		time.Sleep(100 * time.Millisecond)
		cancel()

		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("unexpected error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("tree did not shut down in time")
		}
	})

	t.Run("ServeBackground returns channel", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, _ := NewSupervisorTree(logger, TreeConfig{ShutdownTimeout: time.Second})

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		errCh := tree.ServeBackground(ctx)

		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("unexpected error: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("did not receive from error channel")
		}
	})
}

func TestSupervisorTreeServiceManagement(t *testing.T) {
	t.Run("services in data layer are started", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, _ := NewSupervisorTree(logger, TreeConfig{ShutdownTimeout: time.Second})

		dataSvc := NewMockService("data-service")
		tree.AddDataService(dataSvc)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go tree.Serve(ctx)
		time.Sleep(100 * time.Millisecond)

		if dataSvc.StartCount() < 1 {
			t.Error("data service was not started")
		}
	})

	t.Run("services in messaging layer are started", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, _ := NewSupervisorTree(logger, TreeConfig{ShutdownTimeout: time.Second})

		msgSvc := NewMockService("messaging-service")
		tree.AddMessagingService(msgSvc)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go tree.Serve(ctx)
		time.Sleep(100 * time.Millisecond)

		if msgSvc.StartCount() < 1 {
			t.Error("messaging service was not started")
		}
	})

	t.Run("services in api layer are started", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, _ := NewSupervisorTree(logger, TreeConfig{ShutdownTimeout: time.Second})

		apiSvc := NewMockService("api-service")
		tree.AddAPIService(apiSvc)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go tree.Serve(ctx)
		time.Sleep(100 * time.Millisecond)

		if apiSvc.StartCount() < 1 {
			t.Error("api service was not started")
		}
	})

	// Note: Remove/RemoveAndWait on tree.Root() only works for services
	// added directly to root. Services added to child supervisors (data,
	// messaging, api) must be removed from those supervisors directly.
	// This is a limitation of suture's service token design.
}

func TestSupervisorTreeFailureHandling(t *testing.T) {
	t.Run("failing service in one layer is restarted", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

		tree, _ := NewSupervisorTree(logger, TreeConfig{
			FailureThreshold: 10,
			FailureBackoff:   10 * time.Millisecond,
			ShutdownTimeout:  time.Second,
		})

		failingSvc := NewMockService("failing")
		failingSvc.SetFailCount(2) // Fail twice, then succeed

		stableSvc := NewMockService("stable")

		tree.AddMessagingService(failingSvc)
		tree.AddAPIService(stableSvc)

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		go tree.Serve(ctx)
		time.Sleep(200 * time.Millisecond)

		// Failing service should have been restarted
		if failingSvc.StartCount() < 3 {
			t.Errorf("expected at least 3 starts for failing service, got %d", failingSvc.StartCount())
		}

		// Stable service should have started normally
		if stableSvc.StartCount() < 1 {
			t.Error("stable service was not started")
		}
	})
}

func TestDefaultTreeConfig(t *testing.T) {
	config := DefaultTreeConfig()

	if config.FailureThreshold != 5.0 {
		t.Errorf("expected FailureThreshold 5.0, got %f", config.FailureThreshold)
	}
	if config.FailureDecay != 30.0 {
		t.Errorf("expected FailureDecay 30.0, got %f", config.FailureDecay)
	}
	if config.FailureBackoff != 15*time.Second {
		t.Errorf("expected FailureBackoff 15s, got %v", config.FailureBackoff)
	}
	if config.ShutdownTimeout != 10*time.Second {
		t.Errorf("expected ShutdownTimeout 10s, got %v", config.ShutdownTimeout)
	}
}
