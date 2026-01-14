// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/eventprocessor"
)

// ========================================
// NewReplayHandlers Tests
// ========================================

func TestNewReplayHandlers(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)
	if handlers == nil {
		t.Error("Expected non-nil handlers")
	}
}

func TestNewReplayHandlers_WithStore(t *testing.T) {
	t.Parallel()

	store := &eventprocessor.CheckpointStore{}
	handlers := NewReplayHandlers(store)

	if handlers == nil {
		t.Error("Expected non-nil handlers")
	}
	if handlers.checkpointStore != store {
		t.Error("Expected checkpoint store to be set")
	}
}

// ========================================
// parseReplayMode Tests
// ========================================

func TestParseReplayMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected eventprocessor.ReplayMode
	}{
		{"all", eventprocessor.ReplayModeAll},
		{"sequence", eventprocessor.ReplayModeSequence},
		{"time", eventprocessor.ReplayModeTime},
		{"last_acked", eventprocessor.ReplayModeLastAcked},
		{"new", eventprocessor.ReplayModeNew},
		{"unknown", eventprocessor.ReplayModeNew}, // default
		{"", eventprocessor.ReplayModeNew},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseReplayMode(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ========================================
// checkpointToResponse Tests
// ========================================

func TestCheckpointToResponse(t *testing.T) {
	t.Parallel()

	now := time.Now()
	checkpoint := &eventprocessor.Checkpoint{
		ID:             1,
		ConsumerName:   "test-consumer",
		StreamName:     "test-stream",
		LastSequence:   100,
		LastTimestamp:  now,
		ProcessedCount: 50,
		ErrorCount:     2,
		Status:         "completed",
		ReplayMode:     "sequence",
		StartSequence:  10,
		StartTime:      now.Add(-time.Hour),
		CreatedAt:      now.Add(-2 * time.Hour),
		UpdatedAt:      now,
	}

	response := checkpointToResponse(checkpoint)

	if response.ID != 1 {
		t.Errorf("Expected ID=1, got %d", response.ID)
	}
	if response.ConsumerName != "test-consumer" {
		t.Errorf("Expected consumer_name=test-consumer, got %s", response.ConsumerName)
	}
	if response.StreamName != "test-stream" {
		t.Errorf("Expected stream_name=test-stream, got %s", response.StreamName)
	}
	if response.LastSequence != 100 {
		t.Errorf("Expected last_sequence=100, got %d", response.LastSequence)
	}
	if response.ProcessedCount != 50 {
		t.Errorf("Expected processed_count=50, got %d", response.ProcessedCount)
	}
	if response.ErrorCount != 2 {
		t.Errorf("Expected error_count=2, got %d", response.ErrorCount)
	}
	if response.Status != "completed" {
		t.Errorf("Expected status=completed, got %s", response.Status)
	}
	if response.ReplayMode != "sequence" {
		t.Errorf("Expected replay_mode=sequence, got %s", response.ReplayMode)
	}
	if response.StartSequence != 10 {
		t.Errorf("Expected start_sequence=10, got %d", response.StartSequence)
	}
}

// ========================================
// StartReplay Handler Tests (validation paths)
// ========================================

func TestStartReplay_InvalidJSON(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/replay/start", strings.NewReader("{invalid"))
	rec := httptest.NewRecorder()

	handlers.StartReplay(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestStartReplay_MissingStreamName(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	body := `{"topic": "test-topic", "mode": "all"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/replay/start", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.StartReplay(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["message"] != "stream_name is required" {
		t.Errorf("Expected 'stream_name is required', got %v", errObj["message"])
	}
}

func TestStartReplay_MissingTopic(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	body := `{"stream_name": "test-stream", "mode": "all"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/replay/start", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.StartReplay(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["message"] != "topic is required" {
		t.Errorf("Expected 'topic is required', got %v", errObj["message"])
	}
}

func TestStartReplay_SequenceModeNoStartSequence(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	body := `{"stream_name": "test-stream", "topic": "test-topic", "mode": "sequence"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/replay/start", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.StartReplay(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["message"] != "start_sequence is required for sequence mode" {
		t.Errorf("Expected sequence mode error, got %v", errObj["message"])
	}
}

func TestStartReplay_TimeModeNoStartTime(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	body := `{"stream_name": "test-stream", "topic": "test-topic", "mode": "time"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/replay/start", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.StartReplay(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["message"] != "start_time is required for time mode" {
		t.Errorf("Expected time mode error, got %v", errObj["message"])
	}
}

func TestStartReplay_InvalidStartTime(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	body := `{"stream_name": "test-stream", "topic": "test-topic", "mode": "time", "start_time": "invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/replay/start", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.StartReplay(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if !strings.Contains(errObj["message"].(string), "RFC3339") {
		t.Errorf("Expected RFC3339 format error, got %v", errObj["message"])
	}
}

func TestStartReplay_InvalidStopTime(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	body := `{"stream_name": "test-stream", "topic": "test-topic", "mode": "all", "stop_time": "bad-time"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/replay/start", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handlers.StartReplay(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ========================================
// GetCheckpoint Handler Tests
// ========================================

func TestGetCheckpoint_InvalidID(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/replay/checkpoints/invalid", nil)
	req.SetPathValue("id", "invalid")
	rec := httptest.NewRecorder()

	handlers.GetCheckpoint(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ========================================
// DeleteCheckpoint Handler Tests
// ========================================

func TestDeleteCheckpoint_InvalidID(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/replay/checkpoints/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	handlers.DeleteCheckpoint(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ========================================
// GetLastCheckpoint Handler Tests
// ========================================

func TestGetLastCheckpoint_MissingStream(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/replay/checkpoints/last", nil)
	rec := httptest.NewRecorder()

	handlers.GetLastCheckpoint(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["message"] != "stream parameter is required" {
		t.Errorf("Expected stream required error, got %v", errObj["message"])
	}
}

// ========================================
// CleanupOldCheckpoints Handler Tests
// ========================================

func TestCleanupOldCheckpoints_InvalidDuration(t *testing.T) {
	t.Parallel()

	handlers := NewReplayHandlers(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/replay/checkpoints/cleanup?older_than=invalid", nil)
	rec := httptest.NewRecorder()

	handlers.CleanupOldCheckpoints(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ========================================
// ReplayRequest/Response Type Tests
// ========================================

func TestReplayRequest_JSONMarshaling(t *testing.T) {
	t.Parallel()

	req := ReplayRequest{
		Mode:                "sequence",
		StartSequence:       100,
		StartTime:           "2024-01-01T00:00:00Z",
		StopSequence:        200,
		StopTime:            "2024-12-31T23:59:59Z",
		StreamName:          "test-stream",
		Topic:               "test-topic",
		DryRun:              true,
		VerifyTransactionID: true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ReplayRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Mode != "sequence" {
		t.Errorf("Expected mode=sequence, got %s", decoded.Mode)
	}
	if decoded.StartSequence != 100 {
		t.Errorf("Expected start_sequence=100, got %d", decoded.StartSequence)
	}
	if decoded.StreamName != "test-stream" {
		t.Errorf("Expected stream_name=test-stream, got %s", decoded.StreamName)
	}
	if !decoded.DryRun {
		t.Error("Expected dry_run=true")
	}
	if !decoded.VerifyTransactionID {
		t.Error("Expected verify_transaction_id=true")
	}
}

func TestReplayResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := ReplayResponse{
		ID:        123,
		Status:    "pending",
		Message:   "Replay queued",
		Processed: 50,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["id"].(float64) != 123 {
		t.Errorf("Expected id=123, got %v", decoded["id"])
	}
	if decoded["status"] != "pending" {
		t.Errorf("Expected status=pending, got %v", decoded["status"])
	}
	if decoded["message"] != "Replay queued" {
		t.Errorf("Expected message='Replay queued', got %v", decoded["message"])
	}
	if decoded["processed"].(float64) != 50 {
		t.Errorf("Expected processed=50, got %v", decoded["processed"])
	}
}

func TestCheckpointResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	now := time.Now()
	resp := CheckpointResponse{
		ID:             1,
		ConsumerName:   "test-consumer",
		StreamName:     "test-stream",
		LastSequence:   100,
		LastTimestamp:  now,
		ProcessedCount: 50,
		ErrorCount:     2,
		Status:         "completed",
		ReplayMode:     "all",
		StartSequence:  10,
		StartTime:      now.Add(-time.Hour),
		CreatedAt:      now.Add(-2 * time.Hour),
		UpdatedAt:      now,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["id"].(float64) != 1 {
		t.Errorf("Expected id=1, got %v", decoded["id"])
	}
	if decoded["consumer_name"] != "test-consumer" {
		t.Errorf("Expected consumer_name=test-consumer, got %v", decoded["consumer_name"])
	}
	if decoded["stream_name"] != "test-stream" {
		t.Errorf("Expected stream_name=test-stream, got %v", decoded["stream_name"])
	}
	if decoded["status"] != "completed" {
		t.Errorf("Expected status=completed, got %v", decoded["status"])
	}
}
