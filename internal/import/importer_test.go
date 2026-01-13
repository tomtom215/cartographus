// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package tautulliimport

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/eventprocessor"
)

// --- Mock Implementations ---

// mockEventPublisher is a test double for EventPublisher.
type mockEventPublisher struct {
	mu           sync.Mutex
	events       []*eventprocessor.MediaEvent
	publishErr   error
	publishDelay time.Duration
	callCount    int32
}

func newMockEventPublisher() *mockEventPublisher {
	return &mockEventPublisher{
		events: make([]*eventprocessor.MediaEvent, 0),
	}
}

func (m *mockEventPublisher) PublishEvent(_ context.Context, event *eventprocessor.MediaEvent) error {
	atomic.AddInt32(&m.callCount, 1)

	if m.publishDelay > 0 {
		time.Sleep(m.publishDelay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.publishErr != nil {
		return m.publishErr
	}

	m.events = append(m.events, event)
	return nil
}

func (m *mockEventPublisher) getEvents() []*eventprocessor.MediaEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*eventprocessor.MediaEvent, len(m.events))
	copy(result, m.events)
	return result
}

func (m *mockEventPublisher) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishErr = err
}

// mockProgressTracker is a test double for ProgressTracker.
type mockProgressTracker struct {
	mu       sync.Mutex
	stats    *ImportStats
	saveErr  error
	loadErr  error
	clearErr error
}

func newMockProgressTracker() *mockProgressTracker {
	return &mockProgressTracker{}
}

func (m *mockProgressTracker) Save(_ context.Context, stats *ImportStats) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveErr != nil {
		return m.saveErr
	}

	statsCopy := *stats
	m.stats = &statsCopy
	return nil
}

func (m *mockProgressTracker) Load(_ context.Context) (*ImportStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.loadErr != nil {
		return nil, m.loadErr
	}

	if m.stats == nil {
		return nil, nil
	}

	statsCopy := *m.stats
	return &statsCopy, nil
}

func (m *mockProgressTracker) Clear(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clearErr != nil {
		return m.clearErr
	}

	m.stats = nil
	return nil
}

func (m *mockProgressTracker) setStats(stats *ImportStats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stats != nil {
		statsCopy := *stats
		m.stats = &statsCopy
	} else {
		m.stats = nil
	}
}

func (m *mockProgressTracker) getStats() *ImportStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stats == nil {
		return nil
	}
	statsCopy := *m.stats
	return &statsCopy
}

// --- Helper Functions ---

// createTestDatabaseWithRecords creates a test database and inserts records.
func createTestDatabaseWithRecords(t *testing.T, count int) (string, func()) {
	t.Helper()

	dbPath, cleanup := createTestDatabase(t)
	if count > 0 {
		insertTestRecords(t, dbPath, count)
	}
	return dbPath, cleanup
}

// createImportConfig creates a test import configuration.
func createImportConfig(dbPath string) *config.ImportConfig {
	return &config.ImportConfig{
		Enabled:   true,
		DBPath:    dbPath,
		BatchSize: 5,
		DryRun:    false,
		AutoStart: false,
	}
}

// testSetup holds common test dependencies for importer tests.
type testSetup struct {
	cfg       *config.ImportConfig
	publisher *mockEventPublisher
	progress  *mockProgressTracker
	importer  *Importer
	cleanup   func()
}

// setupImporter creates a test importer with configurable options and returns the setup.
func setupImporter(t *testing.T, recordCount int, opts ...func(*testSetup)) *testSetup {
	t.Helper()

	dbPath, cleanup := createTestDatabaseWithRecords(t, recordCount)

	setup := &testSetup{
		cfg:       createImportConfig(dbPath),
		publisher: newMockEventPublisher(),
		progress:  newMockProgressTracker(),
		cleanup:   cleanup,
	}

	for _, opt := range opts {
		opt(setup)
	}

	setup.importer = NewImporter(setup.cfg, setup.publisher, setup.progress)
	return setup
}

// withBatchSize sets the batch size option.
func withBatchSize(size int) func(*testSetup) {
	return func(s *testSetup) {
		s.cfg.BatchSize = size
	}
}

// withDryRun enables dry run mode.
func withDryRun() func(*testSetup) {
	return func(s *testSetup) {
		s.cfg.DryRun = true
	}
}

// withPublishDelay sets a publish delay on the mock publisher.
func withPublishDelay(delay time.Duration) func(*testSetup) {
	return func(s *testSetup) {
		s.publisher.publishDelay = delay
	}
}

// withPublishError sets an error on the mock publisher.
func withPublishError(err error) func(*testSetup) {
	return func(s *testSetup) {
		s.publisher.setError(err)
	}
}

// withExistingProgress sets up existing progress to simulate resume.
func withExistingProgress(stats *ImportStats) func(*testSetup) {
	return func(s *testSetup) {
		s.progress.setStats(stats)
	}
}

// assertImportCompleted verifies that import completed with expected results.
func assertImportCompleted(t *testing.T, stats *ImportStats, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if stats.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
	// Verify processed count matches total
	if stats.Imported+stats.Skipped+stats.Errors != stats.Processed {
		t.Errorf("Imported (%d) + Skipped (%d) + Errors (%d) != Processed (%d)",
			stats.Imported, stats.Skipped, stats.Errors, stats.Processed)
	}
}

// waitForRunning waits until the importer is running or timeout.
func waitForRunning(t *testing.T, importer *Importer, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !importer.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !importer.IsRunning() {
		t.Fatal("Import didn't start within timeout")
	}
}

// --- Tests ---

func TestNewImporter(t *testing.T) {
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	t.Run("creates importer with valid config", func(t *testing.T) {
		cfg := createImportConfig(dbPath)
		publisher := newMockEventPublisher()
		progress := newMockProgressTracker()

		importer := NewImporter(cfg, publisher, progress)

		if importer == nil {
			t.Fatal("NewImporter() returned nil")
		}
		if importer.cfg != cfg {
			t.Error("Importer config not set correctly")
		}
		if importer.publisher != publisher {
			t.Error("Importer publisher not set correctly")
		}
		if importer.progress != progress {
			t.Error("Importer progress tracker not set correctly")
		}
		if importer.mapper == nil {
			t.Error("Importer mapper not created")
		}
	})

	t.Run("creates importer with nil progress tracker", func(t *testing.T) {
		cfg := createImportConfig(dbPath)
		publisher := newMockEventPublisher()

		importer := NewImporter(cfg, publisher, nil)

		if importer == nil {
			t.Fatal("NewImporter() returned nil")
		}
		if importer.progress != nil {
			t.Error("Progress tracker should be nil")
		}
	})
}

func TestImporter_Import_Success(t *testing.T) {
	setup := setupImporter(t, 10, withBatchSize(3))
	defer setup.cleanup()

	stats, err := setup.importer.Import(context.Background())
	assertImportCompleted(t, stats, err)

	if stats.TotalRecords != 10 {
		t.Errorf("TotalRecords = %d, want 10", stats.TotalRecords)
	}
	if stats.Processed != 10 {
		t.Errorf("Processed = %d, want 10", stats.Processed)
	}

	// Note: EndTime is set in defer, which runs AFTER return value is computed
	finalStats := setup.importer.GetStats()
	if finalStats.EndTime.IsZero() {
		t.Error("EndTime should be set after completion (from GetStats)")
	}

	// Verify progress was saved
	savedStats := setup.progress.getStats()
	if savedStats == nil {
		t.Error("Progress should have been saved")
	}
}

func TestImporter_Import_AlreadyRunning(t *testing.T) {
	setup := setupImporter(t, 100, withBatchSize(1), withPublishDelay(50*time.Millisecond))
	defer setup.cleanup()

	// Start first import in background
	errCh := make(chan error, 1)
	go func() {
		_, err := setup.importer.Import(context.Background())
		errCh <- err
	}()

	// Wait for import to start
	waitForRunning(t, setup.importer, 5*time.Second)

	// Try to start second import
	_, err := setup.importer.Import(context.Background())
	if err == nil {
		t.Error("Expected error when import already in progress")
	}
	if err.Error() != "import already in progress" {
		t.Errorf("Expected 'import already in progress' error, got: %v", err)
	}

	// Verify IsRunning returns true
	if !setup.importer.IsRunning() {
		t.Error("IsRunning() should return true during import")
	}

	// Stop the import and wait for completion
	if err := setup.importer.Stop(); err != nil {
		t.Logf("Stop error (expected): %v", err)
	}

	<-errCh // Wait for first import to finish
}

func TestImporter_Import_DatabaseError(t *testing.T) {
	// Use non-existent database path
	cfg := createImportConfig("/nonexistent/path/to/database.db")
	publisher := newMockEventPublisher()
	progress := newMockProgressTracker()

	importer := NewImporter(cfg, publisher, progress)

	_, err := importer.Import(context.Background())
	if err == nil {
		t.Error("Expected error for non-existent database")
	}
}

func TestImporter_Import_PublishError(t *testing.T) {
	setup := setupImporter(t, 5, withPublishError(errors.New("publish failed")))
	defer setup.cleanup()

	stats, err := setup.importer.Import(context.Background())
	if err != nil {
		t.Fatalf("Import() error = %v (should not return error for publish failures)", err)
	}

	// Import should complete but with errors recorded
	// Note: Some records may be skipped due to validation
	if stats.Errors == 0 && stats.Imported > 0 {
		t.Error("Expected errors to be recorded for publish failures")
	}
}

func TestImporter_Import_Resume(t *testing.T) {
	existingProgress := &ImportStats{
		TotalRecords:    10,
		Processed:       5,
		Imported:        5,
		LastProcessedID: 5,
		StartTime:       time.Now().Add(-time.Hour),
	}
	setup := setupImporter(t, 10, withBatchSize(5), withExistingProgress(existingProgress))
	defer setup.cleanup()

	stats, err := setup.importer.Import(context.Background())
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// Should have processed remaining records (6-10)
	if stats.LastProcessedID < 5 {
		t.Errorf("LastProcessedID = %d, should be > 5 after resume", stats.LastProcessedID)
	}
}

func TestImporter_Import_DryRun(t *testing.T) {
	setup := setupImporter(t, 10, withDryRun())
	defer setup.cleanup()

	stats, err := setup.importer.Import(context.Background())
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// Verify dry run flag
	if !stats.DryRun {
		t.Error("DryRun should be true in stats")
	}

	// Verify no events were published
	events := setup.publisher.getEvents()
	if len(events) > 0 {
		t.Errorf("DryRun should not publish events, got %d", len(events))
	}

	// Progress should not be saved in dry run
	savedStats := setup.progress.getStats()
	if savedStats != nil {
		t.Error("Progress should not be saved during dry run")
	}
}

func TestImporter_Import_ContextCancellation(t *testing.T) {
	setup := setupImporter(t, 100, withBatchSize(1), withPublishDelay(50*time.Millisecond))
	defer setup.cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after short delay
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	stats, err := setup.importer.Import(ctx)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}

	// Stats should always be non-nil, even on cancellation
	if stats == nil {
		t.Fatal("Expected stats to be non-nil")
	}
	// Note: stats.Processed may be 0 if cancellation happened before batch processing started
	// This is valid behavior - we just verify that stats is returned correctly
}

func TestImporter_Stop(t *testing.T) {
	t.Run("stop returns nil when import running", func(t *testing.T) {
		dbPath, cleanup := createTestDatabaseWithRecords(t, 100)
		defer cleanup()

		cfg := createImportConfig(dbPath)
		cfg.BatchSize = 1
		publisher := newMockEventPublisher()
		publisher.publishDelay = 100 * time.Millisecond
		progress := newMockProgressTracker()

		importer := NewImporter(cfg, publisher, progress)

		// Start import in background
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			importer.Import(ctx) //nolint:errcheck
		}()

		// Wait for import to start
		deadline := time.Now().Add(5 * time.Second)
		for !importer.IsRunning() && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}

		if !importer.IsRunning() {
			t.Fatal("Import didn't start within timeout")
		}

		// Stop() should succeed when import is running
		err := importer.Stop()
		if err != nil {
			t.Errorf("Stop() error = %v, want nil", err)
		}

		// Cancel context to ensure cleanup
		cancel()

		// Give time for cleanup
		time.Sleep(200 * time.Millisecond)
	})

	t.Run("stop when not running", func(t *testing.T) {
		dbPath, cleanup := createTestDatabase(t)
		defer cleanup()

		cfg := createImportConfig(dbPath)
		publisher := newMockEventPublisher()

		importer := NewImporter(cfg, publisher, nil)

		err := importer.Stop()
		if err == nil {
			t.Error("Expected error when no import in progress")
		}
		if err.Error() != "no import in progress" {
			t.Errorf("Expected 'no import in progress' error, got: %v", err)
		}
	})
}

func TestImporter_GetStats(t *testing.T) {
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	cfg := createImportConfig(dbPath)
	publisher := newMockEventPublisher()

	importer := NewImporter(cfg, publisher, nil)

	t.Run("returns empty stats before import", func(t *testing.T) {
		stats := importer.GetStats()
		if stats == nil {
			t.Fatal("GetStats() returned nil")
		}
		if stats.TotalRecords != 0 {
			t.Errorf("TotalRecords = %d, want 0", stats.TotalRecords)
		}
	})

	t.Run("returns copy of stats", func(t *testing.T) {
		// Insert some records and run import
		insertTestRecords(t, dbPath, 5)

		_, err := importer.Import(context.Background())
		if err != nil {
			t.Fatalf("Import() error = %v", err)
		}

		stats1 := importer.GetStats()
		stats2 := importer.GetStats()

		// Modify stats1
		stats1.TotalRecords = 999

		// stats2 should be unchanged
		if stats2.TotalRecords == 999 {
			t.Error("GetStats() should return a copy, not the original")
		}
	})
}

func TestImporter_IsRunning(t *testing.T) {
	t.Run("returns false before import", func(t *testing.T) {
		dbPath, cleanup := createTestDatabase(t)
		defer cleanup()

		cfg := createImportConfig(dbPath)
		publisher := newMockEventPublisher()
		importer := NewImporter(cfg, publisher, nil)

		if importer.IsRunning() {
			t.Error("IsRunning() should be false before import")
		}
	})

	t.Run("returns true during import", func(t *testing.T) {
		dbPath, cleanup := createTestDatabaseWithRecords(t, 50) // Enough records
		defer cleanup()

		cfg := createImportConfig(dbPath)
		cfg.BatchSize = 1
		publisher := newMockEventPublisher()
		publisher.publishDelay = 50 * time.Millisecond // Slow publishing
		importer := NewImporter(cfg, publisher, nil)

		// Start import in background
		done := make(chan struct{})
		go func() {
			defer close(done)
			importer.Import(context.Background()) //nolint:errcheck
		}()

		// Wait for import to start (poll until running)
		deadline := time.Now().Add(5 * time.Second)
		for !importer.IsRunning() && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}

		if !importer.IsRunning() {
			t.Error("IsRunning() should be true during import")
		}

		// Stop import to clean up
		importer.Stop() //nolint:errcheck

		// Wait with timeout
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Log("Warning: Import didn't stop within timeout")
		}
	})

	t.Run("returns false after import completes", func(t *testing.T) {
		dbPath, cleanup := createTestDatabaseWithRecords(t, 3) // Few records
		defer cleanup()

		cfg := createImportConfig(dbPath)
		cfg.BatchSize = 10 // Large batch - finishes quickly
		publisher := newMockEventPublisher()
		importer := NewImporter(cfg, publisher, nil)

		// Run import to completion
		_, err := importer.Import(context.Background())
		if err != nil {
			t.Fatalf("Import() error = %v", err)
		}

		if importer.IsRunning() {
			t.Error("IsRunning() should be false after import completes")
		}
	})
}

func TestImporter_processBatch(t *testing.T) {
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	cfg := createImportConfig(dbPath)
	publisher := newMockEventPublisher()

	importer := NewImporter(cfg, publisher, nil)

	t.Run("processes valid records", func(t *testing.T) {
		records := []TautulliRecord{
			{
				ID:              1,
				SessionKey:      "session1",
				StartedAt:       time.Now(),
				UserID:          1,
				Username:        "user1",
				IPAddress:       "192.168.1.1",
				MediaType:       "movie",
				Title:           "Test Movie",
				Platform:        "Chrome",
				Player:          "Plex Web",
				PercentComplete: 100,
			},
			{
				ID:              2,
				SessionKey:      "session2",
				StartedAt:       time.Now(),
				UserID:          2,
				Username:        "user2",
				IPAddress:       "192.168.1.2",
				MediaType:       "episode",
				Title:           "Test Episode",
				Platform:        "Chrome",
				Player:          "Plex Web",
				PercentComplete: 50,
			},
		}

		imported, skipped, errors := importer.processBatch(context.Background(), records)

		if imported != 2 {
			t.Errorf("imported = %d, want 2", imported)
		}
		if skipped != 0 {
			t.Errorf("skipped = %d, want 0", skipped)
		}
		if errors != 0 {
			t.Errorf("errors = %d, want 0", errors)
		}

		events := publisher.getEvents()
		if len(events) != 2 {
			t.Errorf("Published %d events, want 2", len(events))
		}
	})

	t.Run("skips invalid records", func(t *testing.T) {
		// Reset publisher
		publisher = newMockEventPublisher()
		importer.publisher = publisher

		records := []TautulliRecord{
			{
				ID:         1,
				SessionKey: "session1",
				StartedAt:  time.Now(),
				UserID:     1,
				Username:   "user1",
				IPAddress:  "192.168.1.1",
				MediaType:  "movie",
				Title:      "Valid Movie",
			},
			{
				ID:         2,
				SessionKey: "", // Invalid: missing session key
				StartedAt:  time.Now(),
			},
		}

		imported, skipped, errors := importer.processBatch(context.Background(), records)

		if imported != 1 {
			t.Errorf("imported = %d, want 1", imported)
		}
		if skipped != 1 {
			t.Errorf("skipped = %d, want 1", skipped)
		}
		if errors != 0 {
			t.Errorf("errors = %d, want 0", errors)
		}
	})

	t.Run("handles publish errors", func(t *testing.T) {
		// Set up publisher to fail
		publisher = newMockEventPublisher()
		publisher.setError(errors.New("publish failed"))
		importer.publisher = publisher

		records := []TautulliRecord{
			{
				ID:              1,
				SessionKey:      "session1",
				StartedAt:       time.Now(),
				UserID:          1,
				Username:        "user1",
				IPAddress:       "192.168.1.1",
				MediaType:       "movie",
				Title:           "Test Movie",
				Platform:        "Chrome",
				Player:          "Plex Web",
				PercentComplete: 100,
			},
		}

		imported, _, errors := importer.processBatch(context.Background(), records)

		if imported != 0 {
			t.Errorf("imported = %d, want 0 (publish failed)", imported)
		}
		if errors != 1 {
			t.Errorf("errors = %d, want 1", errors)
		}
	})

	t.Run("handles dry run mode", func(t *testing.T) {
		// Set dry run mode
		cfg.DryRun = true
		publisher = newMockEventPublisher()
		importer = NewImporter(cfg, publisher, nil)

		records := []TautulliRecord{
			{
				ID:              1,
				SessionKey:      "session1",
				StartedAt:       time.Now(),
				UserID:          1,
				Username:        "user1",
				IPAddress:       "192.168.1.1",
				MediaType:       "movie",
				Title:           "Test Movie",
				Platform:        "Chrome",
				Player:          "Plex Web",
				PercentComplete: 100,
			},
		}

		imported, _, errors := importer.processBatch(context.Background(), records)

		if imported != 1 {
			t.Errorf("imported = %d, want 1 (counted in dry run)", imported)
		}
		if errors != 0 {
			t.Errorf("errors = %d, want 0", errors)
		}

		// No events should be published
		events := publisher.getEvents()
		if len(events) != 0 {
			t.Errorf("Published %d events in dry run, want 0", len(events))
		}
	})
}

func TestPlaybackEventToMediaEvent(t *testing.T) {
	// Create a test TautulliRecord with full data
	ratingKey := "12345"
	machineID := "machine123"
	friendlyName := "Test User"
	parentTitle := "Season 1"
	grandparentTitle := "Test Show"
	transcodeDecision := "direct play"
	videoResolution := "1080"
	audioCodec := "aac"
	streamBitrate := 10000

	record := &TautulliRecord{
		ID:                1,
		SessionKey:        "test-session",
		StartedAt:         time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		StoppedAt:         time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		UserID:            42,
		Username:          "testuser",
		FriendlyName:      &friendlyName,
		IPAddress:         "192.168.1.100",
		Platform:          "Chrome",
		Player:            "Plex Web",
		MachineID:         &machineID,
		MediaType:         "episode",
		Title:             "Pilot",
		ParentTitle:       &parentTitle,
		GrandparentTitle:  &grandparentTitle,
		RatingKey:         &ratingKey,
		TranscodeDecision: &transcodeDecision,
		VideoResolution:   &videoResolution,
		AudioCodec:        &audioCodec,
		StreamBitrate:     &streamBitrate,
		PercentComplete:   75,
		PausedCounter:     120,
	}

	mapper := NewMapper()
	playbackEvent := mapper.ToPlaybackEvent(record)
	mediaEvent := playbackEventToMediaEvent(playbackEvent)

	// Verify core fields
	if mediaEvent.EventID == "" {
		t.Error("EventID should be set")
	}
	if mediaEvent.SessionKey != "test-session" {
		t.Errorf("SessionKey = %s, want test-session", mediaEvent.SessionKey)
	}
	if mediaEvent.Source != "tautulli-import" {
		t.Errorf("Source = %s, want tautulli-import", mediaEvent.Source)
	}
	if mediaEvent.UserID != 42 {
		t.Errorf("UserID = %d, want 42", mediaEvent.UserID)
	}
	if mediaEvent.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", mediaEvent.Username)
	}
	if mediaEvent.FriendlyName != "Test User" {
		t.Errorf("FriendlyName = %s, want Test User", mediaEvent.FriendlyName)
	}
	if mediaEvent.MachineID != "machine123" {
		t.Errorf("MachineID = %s, want machine123", mediaEvent.MachineID)
	}

	// Verify media fields
	if mediaEvent.MediaType != "episode" {
		t.Errorf("MediaType = %s, want episode", mediaEvent.MediaType)
	}
	if mediaEvent.Title != "Pilot" {
		t.Errorf("Title = %s, want Pilot", mediaEvent.Title)
	}
	if mediaEvent.ParentTitle != "Season 1" {
		t.Errorf("ParentTitle = %s, want Season 1", mediaEvent.ParentTitle)
	}
	if mediaEvent.GrandparentTitle != "Test Show" {
		t.Errorf("GrandparentTitle = %s, want Test Show", mediaEvent.GrandparentTitle)
	}
	if mediaEvent.RatingKey != "12345" {
		t.Errorf("RatingKey = %s, want 12345", mediaEvent.RatingKey)
	}

	// Verify streaming quality fields
	if mediaEvent.TranscodeDecision != "direct play" {
		t.Errorf("TranscodeDecision = %s, want direct play", mediaEvent.TranscodeDecision)
	}
	if mediaEvent.VideoResolution != "1080" {
		t.Errorf("VideoResolution = %s, want 1080", mediaEvent.VideoResolution)
	}
	if mediaEvent.AudioCodec != "aac" {
		t.Errorf("AudioCodec = %s, want aac", mediaEvent.AudioCodec)
	}
	if mediaEvent.StreamBitrate != 10000 {
		t.Errorf("StreamBitrate = %d, want 10000", mediaEvent.StreamBitrate)
	}

	// Verify playback metrics
	if mediaEvent.PercentComplete != 75 {
		t.Errorf("PercentComplete = %d, want 75", mediaEvent.PercentComplete)
	}
	if mediaEvent.PausedCounter != 120 {
		t.Errorf("PausedCounter = %d, want 120", mediaEvent.PausedCounter)
	}

	// Verify timestamps
	if !mediaEvent.StartedAt.Equal(record.StartedAt) {
		t.Errorf("StartedAt = %v, want %v", mediaEvent.StartedAt, record.StartedAt)
	}
	if mediaEvent.StoppedAt == nil {
		t.Error("StoppedAt should be set")
	} else if !mediaEvent.StoppedAt.Equal(record.StoppedAt) {
		t.Errorf("StoppedAt = %v, want %v", mediaEvent.StoppedAt, record.StoppedAt)
	}

	// Verify correlation key is set
	if mediaEvent.CorrelationKey == "" {
		t.Error("CorrelationKey should be set")
	}
}

func TestImporter_Deduplication(t *testing.T) {
	// Test that reimporting the same records produces the same event IDs
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	// Create test data manually with known values
	insertTestRecordsWithKnownData(t, dbPath)

	cfg := createImportConfig(dbPath)
	publisher1 := newMockEventPublisher()
	importer1 := NewImporter(cfg, publisher1, nil)

	// First import
	_, err := importer1.Import(context.Background())
	if err != nil {
		t.Fatalf("First import error: %v", err)
	}

	events1 := publisher1.getEvents()

	// Second import with new publisher
	publisher2 := newMockEventPublisher()
	importer2 := NewImporter(cfg, publisher2, nil)

	_, err = importer2.Import(context.Background())
	if err != nil {
		t.Fatalf("Second import error: %v", err)
	}

	events2 := publisher2.getEvents()

	// Verify same event IDs were generated
	if len(events1) != len(events2) {
		t.Fatalf("Different number of events: %d vs %d", len(events1), len(events2))
	}

	for i := range events1 {
		if events1[i].EventID != events2[i].EventID {
			t.Errorf("Event %d: EventID mismatch: %s vs %s",
				i, events1[i].EventID, events2[i].EventID)
		}
		if events1[i].CorrelationKey != events2[i].CorrelationKey {
			t.Errorf("Event %d: CorrelationKey mismatch: %s vs %s",
				i, events1[i].CorrelationKey, events2[i].CorrelationKey)
		}
	}
}

// insertTestRecordsWithKnownData inserts records with deterministic data for deduplication testing.
func insertTestRecordsWithKnownData(t *testing.T, dbPath string) {
	t.Helper()

	tmpDir := filepath.Dir(dbPath)
	// Ensure the directory exists (created by createTestDatabase)
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Fatalf("Directory does not exist: %s", tmpDir)
	}

	db, ctx := setupDuckDBWithSQLiteScanner(t)
	defer db.Close()

	attachSQLiteDB(t, db, ctx, dbPath, "tautulli")

	// Fixed timestamp for deterministic event IDs
	started := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC).Unix()
	stopped := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix()

	// Insert session history with known values
	_, err := db.ExecContext(ctx, `
		INSERT INTO tautulli.session_history (id, session_key, started, stopped, user_id, user, ip_address, platform, player, percent_complete, paused_counter, location)
		VALUES (1, 'known-session-1', ?, ?, 42, 'testuser', '192.168.1.100', 'Chrome', 'Plex Web', 100, 0, 'lan')
	`, started, stopped)
	if err != nil {
		t.Fatalf("Failed to insert session_history: %v", err)
	}

	// Insert metadata
	_, err = db.ExecContext(ctx, `
		INSERT INTO tautulli.session_history_metadata (id, media_type, title, year)
		VALUES (1, 'movie', 'Test Movie', 2024)
	`)
	if err != nil {
		t.Fatalf("Failed to insert session_history_metadata: %v", err)
	}

	// Insert media info
	_, err = db.ExecContext(ctx, `
		INSERT INTO tautulli.session_history_media_info (id, video_resolution, video_codec, transcode_decision)
		VALUES (1, '1080', 'h264', 'direct play')
	`)
	if err != nil {
		t.Fatalf("Failed to insert session_history_media_info: %v", err)
	}

	detachSQLiteDB(t, db, ctx, "tautulli")
}
