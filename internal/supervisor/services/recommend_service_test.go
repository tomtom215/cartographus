// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// mockRecommendEngine is a mock implementation for testing.
type mockRecommendEngine struct {
	mu         sync.Mutex
	trainCalls int
	trainErr   error
	trainDelay time.Duration
}

func (m *mockRecommendEngine) Train(ctx context.Context) error {
	m.mu.Lock()
	m.trainCalls++
	m.mu.Unlock()

	if m.trainDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.trainDelay):
		}
	}

	return m.trainErr
}

func (m *mockRecommendEngine) getTrainCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.trainCalls
}

func TestRecommendService_String(t *testing.T) {
	logger := zerolog.Nop()
	engine := &mockRecommendEngine{}
	cfg := RecommendServiceConfig{
		TrainInterval: time.Hour,
	}

	service := NewRecommendService(engine, cfg, logger)

	if got := service.String(); got != "recommend-service" {
		t.Errorf("String() = %q, want %q", got, "recommend-service")
	}
}

func TestRecommendService_TrainOnStartup(t *testing.T) {
	logger := zerolog.Nop()
	engine := &mockRecommendEngine{}
	cfg := RecommendServiceConfig{
		TrainOnStartup: true,
		TrainInterval:  time.Hour, // Long interval to avoid scheduled training
	}

	service := NewRecommendService(engine, cfg, logger)

	// Run service briefly
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = service.Serve(ctx)

	// Should have trained once on startup
	if got := engine.getTrainCalls(); got != 1 {
		t.Errorf("Train() called %d times, want 1", got)
	}
}

func TestRecommendService_NoTrainOnStartup(t *testing.T) {
	logger := zerolog.Nop()
	engine := &mockRecommendEngine{}
	cfg := RecommendServiceConfig{
		TrainOnStartup: false,
		TrainInterval:  time.Hour, // Long interval to avoid scheduled training
	}

	service := NewRecommendService(engine, cfg, logger)

	// Run service briefly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = service.Serve(ctx)

	// Should not have trained
	if got := engine.getTrainCalls(); got != 0 {
		t.Errorf("Train() called %d times, want 0", got)
	}
}

func TestRecommendService_ScheduledTraining(t *testing.T) {
	logger := zerolog.Nop()
	engine := &mockRecommendEngine{}
	cfg := RecommendServiceConfig{
		TrainOnStartup: false,
		TrainInterval:  50 * time.Millisecond, // Short interval for testing
	}

	service := NewRecommendService(engine, cfg, logger)

	// Run service long enough for 2 scheduled trainings
	ctx, cancel := context.WithTimeout(context.Background(), 130*time.Millisecond)
	defer cancel()

	_ = service.Serve(ctx)

	// Should have trained at least twice (at 50ms and 100ms)
	if got := engine.getTrainCalls(); got < 2 {
		t.Errorf("Train() called %d times, want >= 2", got)
	}
}

func TestRecommendService_GracefulShutdown(t *testing.T) {
	logger := zerolog.Nop()
	engine := &mockRecommendEngine{
		trainDelay: 50 * time.Millisecond,
	}
	cfg := RecommendServiceConfig{
		TrainOnStartup: true,
		TrainInterval:  time.Hour,
	}

	service := NewRecommendService(engine, cfg, logger)

	// Create a context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- service.Serve(ctx)
	}()

	// Wait for training to start, then cancel
	time.Sleep(20 * time.Millisecond)
	cancel()

	// Should complete gracefully
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Serve() returned %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not complete in time")
	}
}

func TestRecommendService_TrainingError(t *testing.T) {
	logger := zerolog.Nop()
	engine := &mockRecommendEngine{
		trainErr: context.DeadlineExceeded,
	}
	cfg := RecommendServiceConfig{
		TrainOnStartup: true,
		TrainInterval:  time.Hour,
	}

	service := NewRecommendService(engine, cfg, logger)

	// Run service briefly - should continue despite training error
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = service.Serve(ctx)

	// Should have attempted training despite error
	if got := engine.getTrainCalls(); got != 1 {
		t.Errorf("Train() called %d times, want 1", got)
	}
}
