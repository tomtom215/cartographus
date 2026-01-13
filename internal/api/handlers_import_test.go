// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

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
	tautulliimport "github.com/tomtom215/cartographus/internal/import"
)

// --- Mock Implementations ---

// mockImportController is a test double for ImportController.
type mockImportController struct {
	mu          sync.Mutex
	running     bool
	stats       *tautulliimport.ImportStats
	importErr   error
	stopErr     error
	importDelay time.Duration
}

func newMockImportController() *mockImportController {
	return &mockImportController{
		stats: &tautulliimport.ImportStats{},
	}
}

func (m *mockImportController) Import(ctx context.Context) (*tautulliimport.ImportStats, error) {
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	if m.importDelay > 0 {
		select {
		case <-time.After(m.importDelay):
		case <-ctx.Done():
			m.mu.Lock()
			m.running = false
			m.mu.Unlock()
			return m.stats, ctx.Err()
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false

	if m.importErr != nil {
		return nil, m.importErr
	}
	return m.stats, nil
}

func (m *mockImportController) GetStats() *tautulliimport.ImportStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stats == nil {
		return &tautulliimport.ImportStats{}
	}
	statsCopy := *m.stats
	return &statsCopy
}

func (m *mockImportController) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *mockImportController) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopErr != nil {
		return m.stopErr
	}
	if !m.running {
		return errors.New("no import in progress")
	}
	m.running = false
	return nil
}

func (m *mockImportController) setRunning(running bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = running
}

func (m *mockImportController) setStats(stats *tautulliimport.ImportStats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stats != nil {
		statsCopy := *stats
		m.stats = &statsCopy
	} else {
		m.stats = &tautulliimport.ImportStats{}
	}
}

// mockProgressController is a test double for ProgressController.
type mockProgressController struct {
	mu       sync.Mutex
	stats    *tautulliimport.ImportStats
	loadErr  error
	clearErr error
}

func newMockProgressController() *mockProgressController {
	return &mockProgressController{}
}

func (m *mockProgressController) Load(_ context.Context) (*tautulliimport.ImportStats, error) {
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

func (m *mockProgressController) Clear(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clearErr != nil {
		return m.clearErr
	}
	m.stats = nil
	return nil
}

func (m *mockProgressController) setStats(stats *tautulliimport.ImportStats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stats != nil {
		statsCopy := *stats
		m.stats = &statsCopy
	} else {
		m.stats = nil
	}
}

// --- Tests ---

func TestHandleStartImport_Success(t *testing.T) {
	importer := newMockImportController()
	importer.setStats(&tautulliimport.ImportStats{
		TotalRecords: 100,
		StartTime:    time.Now(),
	})
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/tautulli", nil)
	w := httptest.NewRecorder()

	handlers.HandleStartImport(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true")
	}
	if response.Message != "import started" {
		t.Errorf("Message = %q, want 'import started'", response.Message)
	}
}

func TestHandleStartImport_AlreadyRunning(t *testing.T) {
	importer := newMockImportController()
	importer.setRunning(true)
	importer.setStats(&tautulliimport.ImportStats{
		TotalRecords: 100,
		Processed:    50,
		StartTime:    time.Now(),
	})
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/tautulli", nil)
	w := httptest.NewRecorder()

	handlers.HandleStartImport(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success=false")
	}
	if response.Error != "import already in progress" {
		t.Errorf("Error = %q, want 'import already in progress'", response.Error)
	}
}

func TestHandleStartImport_InvalidRequest(t *testing.T) {
	importer := newMockImportController()
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/tautulli", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 12 // Set content length to trigger body parsing
	w := httptest.NewRecorder()

	handlers.HandleStartImport(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleStartImport_Resume(t *testing.T) {
	importer := newMockImportController()
	importer.setStats(&tautulliimport.ImportStats{
		TotalRecords: 100,
		StartTime:    time.Now(),
	})
	progress := newMockProgressController()
	progress.setStats(&tautulliimport.ImportStats{
		TotalRecords:    100,
		Processed:       50,
		LastProcessedID: 50,
	})

	handlers := NewImportHandlers(importer, progress)

	reqBody := `{"resume": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/tautulli", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.HandleStartImport(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify progress was NOT cleared (resume=true)
	savedStats, _ := progress.Load(context.Background())
	if savedStats == nil {
		t.Error("Progress should not be cleared when resume=true")
	}
}

func TestHandleStartImport_DryRun(t *testing.T) {
	importer := newMockImportController()
	importer.setStats(&tautulliimport.ImportStats{
		TotalRecords: 100,
		DryRun:       true,
		StartTime:    time.Now(),
	})
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	reqBody := `{"dry_run": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/tautulli", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.HandleStartImport(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleGetImportStatus_Running(t *testing.T) {
	importer := newMockImportController()
	importer.setRunning(true)
	importer.setStats(&tautulliimport.ImportStats{
		TotalRecords: 100,
		Processed:    50,
		Imported:     48,
		Skipped:      2,
		StartTime:    time.Now().Add(-time.Minute),
	})
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/import/status", nil)
	w := httptest.NewRecorder()

	handlers.HandleGetImportStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true")
	}
	if response.Stats == nil {
		t.Fatal("Expected stats in response")
	}
	if response.Stats.Status != "running" {
		t.Errorf("Status = %q, want 'running'", response.Stats.Status)
	}
	if response.Stats.TotalRecords != 100 {
		t.Errorf("TotalRecords = %d, want 100", response.Stats.TotalRecords)
	}
	if response.Stats.Processed != 50 {
		t.Errorf("Processed = %d, want 50", response.Stats.Processed)
	}
}

func TestHandleGetImportStatus_Completed(t *testing.T) {
	importer := newMockImportController()
	importer.setRunning(false)
	startTime := time.Now().Add(-time.Hour)
	endTime := startTime.Add(10 * time.Minute)
	importer.setStats(&tautulliimport.ImportStats{
		TotalRecords: 100,
		Processed:    100,
		Imported:     100,
		StartTime:    startTime,
		EndTime:      endTime,
	})
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/import/status", nil)
	w := httptest.NewRecorder()

	handlers.HandleGetImportStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Stats.Status != "completed" {
		t.Errorf("Status = %q, want 'completed'", response.Stats.Status)
	}
}

func TestHandleGetImportStatus_NoImport(t *testing.T) {
	importer := newMockImportController()
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/import/status", nil)
	w := httptest.NewRecorder()

	handlers.HandleGetImportStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Stats.Status != "pending" {
		t.Errorf("Status = %q, want 'pending'", response.Stats.Status)
	}
}

func TestHandleGetImportStatus_LoadsSavedProgress(t *testing.T) {
	importer := newMockImportController()
	importer.setRunning(false)
	// No current stats
	importer.setStats(&tautulliimport.ImportStats{})

	progress := newMockProgressController()
	progress.setStats(&tautulliimport.ImportStats{
		TotalRecords:    200,
		Processed:       150,
		Imported:        148,
		Skipped:         2,
		LastProcessedID: 150,
		StartTime:       time.Now().Add(-time.Hour),
	})

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/import/status", nil)
	w := httptest.NewRecorder()

	handlers.HandleGetImportStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should load saved progress
	if response.Stats.TotalRecords != 200 {
		t.Errorf("TotalRecords = %d, want 200 (from saved progress)", response.Stats.TotalRecords)
	}
}

func TestHandleStopImport_Success(t *testing.T) {
	importer := newMockImportController()
	importer.setRunning(true)
	importer.setStats(&tautulliimport.ImportStats{
		TotalRecords: 100,
		Processed:    50,
		StartTime:    time.Now(),
	})
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/import", nil)
	w := httptest.NewRecorder()

	handlers.HandleStopImport(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true")
	}
	if response.Message != "import stop requested" {
		t.Errorf("Message = %q, want 'import stop requested'", response.Message)
	}
}

func TestHandleStopImport_NotRunning(t *testing.T) {
	importer := newMockImportController()
	importer.setRunning(false)
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/import", nil)
	w := httptest.NewRecorder()

	handlers.HandleStopImport(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success=false")
	}
	if response.Error != "no import in progress" {
		t.Errorf("Error = %q, want 'no import in progress'", response.Error)
	}
}

func TestHandleClearProgress_Success(t *testing.T) {
	importer := newMockImportController()
	importer.setRunning(false)
	progress := newMockProgressController()
	progress.setStats(&tautulliimport.ImportStats{
		TotalRecords: 100,
		Processed:    100,
	})

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/import/progress", nil)
	w := httptest.NewRecorder()

	handlers.HandleClearProgress(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true")
	}
	if response.Message != "progress cleared" {
		t.Errorf("Message = %q, want 'progress cleared'", response.Message)
	}

	// Verify progress was cleared
	savedStats, _ := progress.Load(context.Background())
	if savedStats != nil {
		t.Error("Progress should have been cleared")
	}
}

func TestHandleClearProgress_WhileRunning(t *testing.T) {
	importer := newMockImportController()
	importer.setRunning(true)
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/import/progress", nil)
	w := httptest.NewRecorder()

	handlers.HandleClearProgress(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success=false")
	}
	if response.Error != "cannot clear progress while import is running" {
		t.Errorf("Error = %q, want 'cannot clear progress while import is running'", response.Error)
	}
}

func TestHandleClearProgress_NoProgressTracker(t *testing.T) {
	importer := newMockImportController()
	importer.setRunning(false)

	handlers := NewImportHandlers(importer, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/import/progress", nil)
	w := httptest.NewRecorder()

	handlers.HandleClearProgress(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var response ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true")
	}
	if response.Message != "progress tracking not enabled" {
		t.Errorf("Message = %q, want 'progress tracking not enabled'", response.Message)
	}
}

func TestHandleValidateDatabase_InvalidPath(t *testing.T) {
	importer := newMockImportController()
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	reqBody := `{"db_path": "/nonexistent/path/to/database.db"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/validate", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.HandleValidateDatabase(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleValidateDatabase_MissingPath(t *testing.T) {
	importer := newMockImportController()
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	reqBody := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/validate", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.HandleValidateDatabase(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "db_path is required" {
		t.Errorf("Error = %q, want 'db_path is required'", response["error"])
	}
}

func TestHandleValidateDatabase_InvalidJSON(t *testing.T) {
	importer := newMockImportController()
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/validate", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.HandleValidateDatabase(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestNewImportHandlers(t *testing.T) {
	importer := newMockImportController()
	progress := newMockProgressController()

	handlers := NewImportHandlers(importer, progress)

	if handlers == nil {
		t.Fatal("NewImportHandlers() returned nil")
	}
	if handlers.importer == nil {
		t.Error("Importer not set")
	}
	if handlers.progress == nil {
		t.Error("Progress not set")
	}
}

func TestNewImportHandlers_NilProgress(t *testing.T) {
	importer := newMockImportController()

	handlers := NewImportHandlers(importer, nil)

	if handlers == nil {
		t.Fatal("NewImportHandlers() returned nil")
	}
	if handlers.progress != nil {
		t.Error("Progress should be nil")
	}
}
