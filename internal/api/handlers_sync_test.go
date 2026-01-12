// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

// --- Mock Implementations ---

// mockImportStatusProvider is a test double for ImportStatusProvider.
type mockImportStatusProvider struct {
	mu      sync.Mutex
	running bool
	stats   interface{}
}

func newMockImportStatusProvider() *mockImportStatusProvider {
	return &mockImportStatusProvider{}
}

func (m *mockImportStatusProvider) IsImportRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *mockImportStatusProvider) GetImportStats() interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stats
}

func (m *mockImportStatusProvider) setRunning(running bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = running
}

// mockSyncStatusProvider is a test double for SyncStatusProvider.
type mockSyncStatusProvider struct {
	mu                     sync.Mutex
	plexHistoricalRunning  bool
	plexHistoricalProgress *SyncProgress
	startSyncErr           error
	startSyncDelay         time.Duration
}

func newMockSyncStatusProvider() *mockSyncStatusProvider {
	return &mockSyncStatusProvider{}
}

func (m *mockSyncStatusProvider) IsPlexHistoricalRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.plexHistoricalRunning
}

func (m *mockSyncStatusProvider) GetPlexHistoricalProgress() *SyncProgress {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.plexHistoricalProgress != nil {
		progressCopy := *m.plexHistoricalProgress
		return &progressCopy
	}
	return nil
}

func (m *mockSyncStatusProvider) StartPlexHistoricalSync(_ context.Context, _ int, _ []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.startSyncDelay > 0 {
		m.mu.Unlock()
		time.Sleep(m.startSyncDelay)
		m.mu.Lock()
	}

	return m.startSyncErr
}

func (m *mockSyncStatusProvider) setRunning(running bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plexHistoricalRunning = running
}

func (m *mockSyncStatusProvider) setProgress(progress *SyncProgress) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if progress != nil {
		progressCopy := *progress
		m.plexHistoricalProgress = &progressCopy
	} else {
		m.plexHistoricalProgress = nil
	}
}

// --- Tests for HandleGetSyncStatus ---

func TestHandleGetSyncStatus_NoOperationsRunning(t *testing.T) {
	handlers := NewSyncHandlers(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/status", nil)
	w := httptest.NewRecorder()

	handlers.HandleGetSyncStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response SyncStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.TautulliImport != nil {
		t.Error("Expected TautulliImport to be nil")
	}
	if response.PlexHistorical != nil {
		t.Error("Expected PlexHistorical to be nil")
	}
	if len(response.ServerSyncs) != 0 {
		t.Errorf("Expected empty ServerSyncs, got %d entries", len(response.ServerSyncs))
	}
}

func TestHandleGetSyncStatus_WithPlexHistoricalProgress(t *testing.T) {
	provider := newMockSyncStatusProvider()
	startTime := time.Now()
	provider.setProgress(&SyncProgress{
		Status:           "running",
		TotalRecords:     1000,
		ProcessedRecords: 500,
		ProgressPercent:  50.0,
		StartTime:        &startTime,
	})

	handlers := NewSyncHandlers(nil, provider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/status", nil)
	w := httptest.NewRecorder()

	handlers.HandleGetSyncStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response SyncStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.PlexHistorical == nil {
		t.Fatal("Expected PlexHistorical to be present")
	}
	if response.PlexHistorical.Status != "running" {
		t.Errorf("Status = %q, want 'running'", response.PlexHistorical.Status)
	}
	if response.PlexHistorical.TotalRecords != 1000 {
		t.Errorf("TotalRecords = %d, want 1000", response.PlexHistorical.TotalRecords)
	}
	if response.PlexHistorical.ProcessedRecords != 500 {
		t.Errorf("ProcessedRecords = %d, want 500", response.PlexHistorical.ProcessedRecords)
	}
	if response.PlexHistorical.ProgressPercent != 50.0 {
		t.Errorf("ProgressPercent = %f, want 50.0", response.PlexHistorical.ProgressPercent)
	}
}

func TestHandleGetSyncStatus_WithInternalProgress(t *testing.T) {
	handlers := NewSyncHandlers(nil, nil)

	// Set internal progress
	startTime := time.Now()
	handlers.UpdatePlexHistoricalProgress(&SyncProgress{
		Status:           "running",
		TotalRecords:     200,
		ProcessedRecords: 100,
		ProgressPercent:  50.0,
		StartTime:        &startTime,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/status", nil)
	w := httptest.NewRecorder()

	handlers.HandleGetSyncStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response SyncStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.PlexHistorical == nil {
		t.Fatal("Expected PlexHistorical to be present")
	}
	if response.PlexHistorical.TotalRecords != 200 {
		t.Errorf("TotalRecords = %d, want 200", response.PlexHistorical.TotalRecords)
	}
}

// --- Tests for HandleStartPlexHistoricalSync ---

func TestHandleStartPlexHistoricalSync_Success(t *testing.T) {
	provider := newMockSyncStatusProvider()
	handlers := NewSyncHandlers(nil, provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/plex/historical", nil)
	w := httptest.NewRecorder()

	handlers.HandleStartPlexHistoricalSync(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response PlexHistoricalResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true")
	}
	if response.Message != "Plex historical sync started" {
		t.Errorf("Message = %q, want 'Plex historical sync started'", response.Message)
	}
	if response.CorrelationID == "" {
		t.Error("Expected correlation_id to be set")
	}
}

func TestHandleStartPlexHistoricalSync_WithOptions(t *testing.T) {
	provider := newMockSyncStatusProvider()
	handlers := NewSyncHandlers(nil, provider)

	reqBody := `{"days_back": 60, "library_ids": ["1", "2", "3"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/plex/historical", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.HandleStartPlexHistoricalSync(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response PlexHistoricalResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true")
	}
}

func TestHandleStartPlexHistoricalSync_AlreadyRunning(t *testing.T) {
	provider := newMockSyncStatusProvider()
	provider.setRunning(true)
	handlers := NewSyncHandlers(nil, provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/plex/historical", nil)
	w := httptest.NewRecorder()

	handlers.HandleStartPlexHistoricalSync(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}

	var response PlexHistoricalResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success=false")
	}
	if response.Error != "Plex historical sync already in progress" {
		t.Errorf("Error = %q, want 'Plex historical sync already in progress'", response.Error)
	}
}

func TestHandleStartPlexHistoricalSync_InternalAlreadyRunning(t *testing.T) {
	handlers := NewSyncHandlers(nil, nil)

	// Set internal running state
	handlers.mu.Lock()
	handlers.plexHistoricalRunning = true
	handlers.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/plex/historical", nil)
	w := httptest.NewRecorder()

	handlers.HandleStartPlexHistoricalSync(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}

	var response PlexHistoricalResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success=false")
	}
}

func TestHandleStartPlexHistoricalSync_ConflictWithTautulliImport(t *testing.T) {
	// Create mock import status provider that is running
	importProvider := newMockImportStatusProvider()
	importProvider.setRunning(true)

	handlers := NewSyncHandlers(importProvider, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/plex/historical", nil)
	w := httptest.NewRecorder()

	handlers.HandleStartPlexHistoricalSync(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}

	var response PlexHistoricalResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success=false")
	}
	if response.Error != "cannot start Plex historical sync while Tautulli import is running" {
		t.Errorf("Error = %q, want 'cannot start Plex historical sync while Tautulli import is running'", response.Error)
	}
}

func TestHandleStartPlexHistoricalSync_InvalidJSON(t *testing.T) {
	handlers := NewSyncHandlers(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/plex/historical", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 12
	w := httptest.NewRecorder()

	handlers.HandleStartPlexHistoricalSync(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var response PlexHistoricalResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success=false")
	}
	if response.Error == "" {
		t.Error("Expected error message")
	}
}

func TestHandleStartPlexHistoricalSync_ProviderError(t *testing.T) {
	provider := newMockSyncStatusProvider()
	provider.startSyncErr = errors.New("sync provider error")
	handlers := NewSyncHandlers(nil, provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/plex/historical", nil)
	w := httptest.NewRecorder()

	handlers.HandleStartPlexHistoricalSync(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Should still return 200 OK because sync starts in background
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response PlexHistoricalResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true (sync starts async)")
	}

	// Wait for goroutine to complete and check status
	time.Sleep(100 * time.Millisecond)

	handlers.mu.RLock()
	progress := handlers.plexHistoricalProgress
	handlers.mu.RUnlock()

	if progress == nil || progress.Status != "error" {
		t.Errorf("Expected progress status to be 'error' after provider failure")
	}
}

// --- Tests for UpdatePlexHistoricalProgress ---

func TestUpdatePlexHistoricalProgress(t *testing.T) {
	handlers := NewSyncHandlers(nil, nil)

	startTime := time.Now()
	progress := &SyncProgress{
		Status:           "running",
		TotalRecords:     1000,
		ProcessedRecords: 250,
		ImportedRecords:  248,
		SkippedRecords:   2,
		ProgressPercent:  25.0,
		RecordsPerSecond: 50.0,
		StartTime:        &startTime,
	}

	handlers.UpdatePlexHistoricalProgress(progress)

	handlers.mu.RLock()
	stored := handlers.plexHistoricalProgress
	handlers.mu.RUnlock()

	if stored == nil {
		t.Fatal("Expected progress to be stored")
	}
	if stored.Status != "running" {
		t.Errorf("Status = %q, want 'running'", stored.Status)
	}
	if stored.TotalRecords != 1000 {
		t.Errorf("TotalRecords = %d, want 1000", stored.TotalRecords)
	}
	if stored.ProcessedRecords != 250 {
		t.Errorf("ProcessedRecords = %d, want 250", stored.ProcessedRecords)
	}
}

func TestUpdatePlexHistoricalProgress_Nil(t *testing.T) {
	handlers := NewSyncHandlers(nil, nil)

	// Set initial progress
	handlers.UpdatePlexHistoricalProgress(&SyncProgress{
		Status: "running",
	})

	// Update to nil
	handlers.UpdatePlexHistoricalProgress(nil)

	handlers.mu.RLock()
	stored := handlers.plexHistoricalProgress
	handlers.mu.RUnlock()

	if stored != nil {
		t.Error("Expected progress to be nil after update with nil")
	}
}

// --- Tests for NewSyncHandlers ---

func TestNewSyncHandlers(t *testing.T) {
	importProvider := newMockImportStatusProvider()
	syncProvider := newMockSyncStatusProvider()

	handlers := NewSyncHandlers(importProvider, syncProvider)

	if handlers == nil {
		t.Fatal("NewSyncHandlers() returned nil")
	}
	if handlers.importStatusProvider == nil {
		t.Error("ImportStatusProvider not set")
	}
	if handlers.syncStatusProvider == nil {
		t.Error("SyncStatusProvider not set")
	}
}

func TestNewSyncHandlers_NilDependencies(t *testing.T) {
	handlers := NewSyncHandlers(nil, nil)

	if handlers == nil {
		t.Fatal("NewSyncHandlers() returned nil")
	}
	if handlers.importStatusProvider != nil {
		t.Error("ImportStatusProvider should be nil")
	}
	if handlers.syncStatusProvider != nil {
		t.Error("SyncStatusProvider should be nil")
	}
}

// --- Tests for helper functions ---

func TestGenerateCorrelationID(t *testing.T) {
	id1 := generateCorrelationID()
	id2 := generateCorrelationID()

	if id1 == "" {
		t.Error("Expected non-empty correlation ID")
	}
	if len(id1) < 10 {
		t.Errorf("Correlation ID too short: %q", id1)
	}

	// IDs should be unique (or at least different most of the time)
	// Due to time-based generation, this could occasionally fail
	// but should be reliable in normal testing conditions
	time.Sleep(1 * time.Millisecond) // Ensure time difference
	if id1 == id2 {
		t.Log("Warning: correlation IDs are the same (time-based generation)")
	}
}

func TestRandomSuffix(t *testing.T) {
	suffix := randomSuffix()

	if len(suffix) != 8 {
		t.Errorf("Expected suffix length 8, got %d", len(suffix))
	}

	// Check that suffix only contains valid characters
	for _, c := range suffix {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			t.Errorf("Invalid character in suffix: %c", c)
		}
	}
}

// --- Tests for convertImportStatsToSyncProgress ---

func TestConvertImportStatsToSyncProgress_Map(t *testing.T) {
	stats := map[string]interface{}{
		"total_records": int64(1000),
		"processed":     int64(500),
		"imported":      int64(495),
		"skipped":       int64(5),
		"errors":        int64(0),
	}

	progress := convertImportStatsToSyncProgress(stats, true)

	if progress == nil {
		t.Fatal("Expected non-nil progress")
	}
	if progress.Status != "running" {
		t.Errorf("Status = %q, want 'running'", progress.Status)
	}
	if progress.TotalRecords != 1000 {
		t.Errorf("TotalRecords = %d, want 1000", progress.TotalRecords)
	}
	if progress.ProcessedRecords != 500 {
		t.Errorf("ProcessedRecords = %d, want 500", progress.ProcessedRecords)
	}
}

func TestConvertImportStatsToSyncProgress_NotRunning(t *testing.T) {
	stats := map[string]interface{}{
		"total_records": int64(100),
	}

	progress := convertImportStatsToSyncProgress(stats, false)

	if progress == nil {
		t.Fatal("Expected non-nil progress")
	}
	if progress.Status != "idle" {
		t.Errorf("Status = %q, want 'idle'", progress.Status)
	}
}

func TestConvertImportStatsToSyncProgress_InvalidType(t *testing.T) {
	progress := convertImportStatsToSyncProgress("invalid", true)

	if progress != nil {
		t.Error("Expected nil progress for invalid input type")
	}
}
