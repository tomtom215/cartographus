// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/newsletter"
	"github.com/tomtom215/cartographus/internal/newsletter/delivery"
)

// mockStore implements SchedulerStore for testing.
type mockStore struct {
	mu                  sync.Mutex
	schedules           []models.NewsletterSchedule
	templates           map[string]*models.NewsletterTemplate
	deliveries          map[string]*models.NewsletterDelivery
	runStatusUpdates    []runStatusUpdate
	getDueForRunCalls   int
	createDeliveryCalls int
}

type runStatusUpdate struct {
	ID        string
	Status    models.DeliveryStatus
	NextRunAt *time.Time
}

func newMockStore() *mockStore {
	return &mockStore{
		templates:  make(map[string]*models.NewsletterTemplate),
		deliveries: make(map[string]*models.NewsletterDelivery),
	}
}

func (m *mockStore) GetSchedulesDueForRun(ctx context.Context) ([]models.NewsletterSchedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getDueForRunCalls++
	return m.schedules, nil
}

func (m *mockStore) GetNewsletterSchedule(ctx context.Context, id string) (*models.NewsletterSchedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.schedules {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *mockStore) UpdateScheduleRunStatus(ctx context.Context, id string, status models.DeliveryStatus, nextRunAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runStatusUpdates = append(m.runStatusUpdates, runStatusUpdate{
		ID:        id,
		Status:    status,
		NextRunAt: nextRunAt,
	})
	return nil
}

func (m *mockStore) GetNewsletterTemplate(ctx context.Context, id string) (*models.NewsletterTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.templates[id]; ok {
		return t, nil
	}
	return nil, nil
}

func (m *mockStore) CreateNewsletterDelivery(ctx context.Context, d *models.NewsletterDelivery) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveries[d.ID] = d
	m.createDeliveryCalls++
	return nil
}

func (m *mockStore) UpdateNewsletterDelivery(ctx context.Context, d *models.NewsletterDelivery) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.deliveries[d.ID]; ok {
		m.deliveries[d.ID] = d
	}
	return nil
}

func (m *mockStore) GetNewsletterDelivery(ctx context.Context, id string) (*models.NewsletterDelivery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.deliveries[id]; ok {
		return d, nil
	}
	return nil, nil
}

func TestNewScheduler(t *testing.T) {
	logger := zerolog.Nop()
	store := newMockStore()

	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "default config",
			config: DefaultConfig(),
		},
		{
			name: "custom config",
			config: Config{
				CheckInterval:           2 * time.Minute,
				MaxConcurrentDeliveries: 10,
				ExecutionTimeout:        10 * time.Minute,
				Enabled:                 true,
			},
		},
		{
			name:   "zero config uses defaults",
			config: Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := NewScheduler(store, nil, nil, nil, &logger, tt.config)
			if scheduler == nil {
				t.Fatal("NewScheduler() returned nil")
			}
			if scheduler.store != store {
				t.Error("Scheduler store not set correctly")
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.CheckInterval != time.Minute {
		t.Errorf("CheckInterval = %v, want %v", cfg.CheckInterval, time.Minute)
	}
	if cfg.MaxConcurrentDeliveries != 5 {
		t.Errorf("MaxConcurrentDeliveries = %d, want 5", cfg.MaxConcurrentDeliveries)
	}
	if cfg.ExecutionTimeout != 5*time.Minute {
		t.Errorf("ExecutionTimeout = %v, want %v", cfg.ExecutionTimeout, 5*time.Minute)
	}
	if !cfg.Enabled {
		t.Error("Enabled should be true by default")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	logger := zerolog.Nop()
	store := newMockStore()
	config := Config{
		CheckInterval: 100 * time.Millisecond,
		Enabled:       true,
	}

	scheduler := NewScheduler(store, nil, nil, nil, &logger, config)

	ctx := context.Background()

	// Start should succeed
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !scheduler.IsRunning() {
		t.Error("IsRunning() should be true after Start")
	}

	// Double start should fail
	if err := scheduler.Start(ctx); err == nil {
		t.Error("Second Start() should return error")
	}

	// Wait a bit for at least one check cycle
	time.Sleep(150 * time.Millisecond)

	// Stop should succeed
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if scheduler.IsRunning() {
		t.Error("IsRunning() should be false after Stop")
	}

	// Double stop should not fail
	if err := scheduler.Stop(); err != nil {
		t.Errorf("Second Stop() should not error, got %v", err)
	}
}

func TestScheduler_Disabled(t *testing.T) {
	logger := zerolog.Nop()
	store := newMockStore()
	config := Config{
		CheckInterval: 50 * time.Millisecond,
		Enabled:       false,
	}

	scheduler := NewScheduler(store, nil, nil, nil, &logger, config)

	ctx := context.Background()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Should not have called getDueForRun when disabled
	store.mu.Lock()
	calls := store.getDueForRunCalls
	store.mu.Unlock()

	if calls != 0 {
		t.Errorf("GetSchedulesDueForRun called %d times when disabled, want 0", calls)
	}

	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestScheduler_CheckAndExecute_NoSchedules(t *testing.T) {
	logger := zerolog.Nop()
	store := newMockStore()
	config := Config{
		CheckInterval: 100 * time.Millisecond,
		Enabled:       true,
	}

	scheduler := NewScheduler(store, nil, nil, nil, &logger, config)
	ctx := context.Background()

	// Call checkAndExecute directly
	scheduler.checkAndExecute(ctx)

	store.mu.Lock()
	calls := store.getDueForRunCalls
	store.mu.Unlock()

	if calls != 1 {
		t.Errorf("GetSchedulesDueForRun called %d times, want 1", calls)
	}
}

func TestScheduler_CheckInterval(t *testing.T) {
	logger := zerolog.Nop()
	store := newMockStore()
	config := Config{
		CheckInterval: 50 * time.Millisecond,
		Enabled:       true,
	}

	scheduler := NewScheduler(store, nil, nil, nil, &logger, config)
	ctx := context.Background()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for multiple check cycles
	time.Sleep(180 * time.Millisecond)

	store.mu.Lock()
	calls := store.getDueForRunCalls
	store.mu.Unlock()

	// Should have been called multiple times (initial + at least 2-3 ticks)
	if calls < 3 {
		t.Errorf("GetSchedulesDueForRun called %d times, want at least 3", calls)
	}

	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestScheduler_UpdateScheduleStatus(t *testing.T) {
	logger := zerolog.Nop()
	store := newMockStore()
	config := DefaultConfig()

	scheduler := NewScheduler(store, nil, nil, nil, &logger, config)
	ctx := context.Background()

	schedule := &models.NewsletterSchedule{
		ID:             "sched-1",
		CronExpression: "0 9 * * *",
		Timezone:       "UTC",
	}

	scheduler.updateScheduleStatus(ctx, schedule, models.DeliveryStatusDelivered)

	store.mu.Lock()
	updates := store.runStatusUpdates
	store.mu.Unlock()

	if len(updates) != 1 {
		t.Fatalf("Expected 1 status update, got %d", len(updates))
	}

	if updates[0].ID != "sched-1" {
		t.Errorf("Update ID = %s, want sched-1", updates[0].ID)
	}

	if updates[0].Status != models.DeliveryStatusDelivered {
		t.Errorf("Update Status = %v, want Delivered", updates[0].Status)
	}

	if updates[0].NextRunAt == nil {
		t.Error("NextRunAt should not be nil")
	}
}

// Integration test for the scheduler wrapper
func TestNewsletterSchedulerServiceInterface(t *testing.T) {
	logger := zerolog.Nop()
	store := newMockStore()
	config := Config{
		CheckInterval: 50 * time.Millisecond,
		Enabled:       true,
	}

	scheduler := NewScheduler(store, nil, nil, nil, &logger, config)

	// Verify scheduler implements the interface expected by the service wrapper
	var _ interface {
		Start(ctx context.Context) error
		Stop() error
	} = scheduler
}

func TestContentResolver_ResolveContent(t *testing.T) {
	// Create a minimal content resolver
	logger := zerolog.Nop()

	// The real content resolver needs a ContentStore, but we're just testing the interface
	// This test ensures the ContentResolver interface is correctly defined
	resolver := newsletter.NewContentResolver(nil, &logger, newsletter.ContentResolverConfig{
		ServerName: "Test Server",
		ServerURL:  "http://localhost",
		BaseURL:    "/",
	})

	if resolver == nil {
		t.Error("NewContentResolver returned nil")
	}
}

func TestDeliveryManager_NewManager(t *testing.T) {
	logger := zerolog.Nop()
	config := delivery.DefaultManagerConfig()

	manager := delivery.NewManager(&logger, config)

	if manager == nil {
		t.Error("NewManager returned nil")
	}
}
