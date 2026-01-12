// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides admin handlers for event replay management.
// CRITICAL-002: Admin API for triggering deterministic event replay.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/tomtom215/cartographus/internal/eventprocessor"
	"github.com/tomtom215/cartographus/internal/logging"
)

// ReplayHandlers provides HTTP handlers for event replay operations.
type ReplayHandlers struct {
	checkpointStore *eventprocessor.CheckpointStore
}

// NewReplayHandlers creates new replay handlers.
func NewReplayHandlers(store *eventprocessor.CheckpointStore) *ReplayHandlers {
	return &ReplayHandlers{
		checkpointStore: store,
	}
}

// ReplayRequest is the request body for starting a replay.
type ReplayRequest struct {
	// Mode specifies how to determine the starting point.
	// Options: "new", "all", "sequence", "time", "last_acked"
	Mode string `json:"mode"`

	// StartSequence is the sequence number to start from (for mode "sequence").
	StartSequence uint64 `json:"start_sequence,omitempty"`

	// StartTime is the ISO8601 timestamp to start from (for mode "time").
	StartTime string `json:"start_time,omitempty"`

	// StopSequence is the optional end sequence (0 = no limit).
	StopSequence uint64 `json:"stop_sequence,omitempty"`

	// StopTime is the optional ISO8601 end timestamp.
	StopTime string `json:"stop_time,omitempty"`

	// StreamName is the NATS stream to replay from.
	StreamName string `json:"stream_name"`

	// Topic is the NATS topic to subscribe to.
	Topic string `json:"topic"`

	// DryRun previews replay without writing to database.
	DryRun bool `json:"dry_run,omitempty"`

	// VerifyTransactionID enables transaction ID verification.
	VerifyTransactionID bool `json:"verify_transaction_id,omitempty"`
}

// ReplayResponse is the response for replay operations.
type ReplayResponse struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Processed int64  `json:"processed,omitempty"`
}

// CheckpointResponse is the response for checkpoint queries.
type CheckpointResponse struct {
	ID             int64     `json:"id"`
	ConsumerName   string    `json:"consumer_name"`
	StreamName     string    `json:"stream_name"`
	LastSequence   uint64    `json:"last_sequence"`
	LastTimestamp  time.Time `json:"last_timestamp,omitempty"`
	ProcessedCount int64     `json:"processed_count"`
	ErrorCount     int64     `json:"error_count"`
	Status         string    `json:"status"`
	ReplayMode     string    `json:"replay_mode,omitempty"`
	StartSequence  uint64    `json:"start_sequence,omitempty"`
	StartTime      time.Time `json:"start_time,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ListCheckpoints returns all replay checkpoints.
// @Summary List replay checkpoints
// @Description Returns all replay checkpoints with optional status filter
// @Tags Admin, Replay
// @Accept json
// @Produce json
// @Param status query string false "Filter by status (running, completed, error, canceled)"
// @Success 200 {array} CheckpointResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/replay/checkpoints [get]
func (h *ReplayHandlers) ListCheckpoints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")

	checkpoints, err := h.checkpointStore.List(ctx, status)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to list checkpoints")
		respondError(w, http.StatusInternalServerError, "REPLAY_ERROR", "Failed to list checkpoints", err)
		return
	}

	// Convert to response format
	response := make([]CheckpointResponse, 0, len(checkpoints))
	for _, cp := range checkpoints {
		response = append(response, checkpointToResponse(cp))
	}

	writeJSON(w, response)
}

// GetCheckpoint returns a specific checkpoint by ID.
// @Summary Get replay checkpoint
// @Description Returns a specific replay checkpoint
// @Tags Admin, Replay
// @Accept json
// @Produce json
// @Param id path int true "Checkpoint ID"
// @Success 200 {object} CheckpointResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/replay/checkpoints/{id} [get]
func (h *ReplayHandlers) GetCheckpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid checkpoint ID", err)
		return
	}

	checkpoint, err := h.checkpointStore.GetByID(ctx, id)
	if err != nil {
		logging.Error().Err(err).Int64("id", id).Msg("Failed to get checkpoint")
		respondError(w, http.StatusInternalServerError, "REPLAY_ERROR", "Failed to get checkpoint", err)
		return
	}

	if checkpoint == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Checkpoint not found", nil)
		return
	}

	writeJSON(w, checkpointToResponse(checkpoint))
}

// DeleteCheckpoint removes a checkpoint by ID.
// @Summary Delete replay checkpoint
// @Description Removes a specific replay checkpoint
// @Tags Admin, Replay
// @Accept json
// @Produce json
// @Param id path int true "Checkpoint ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/replay/checkpoints/{id} [delete]
func (h *ReplayHandlers) DeleteCheckpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid checkpoint ID", err)
		return
	}

	if err := h.checkpointStore.Delete(ctx, id); err != nil {
		logging.Error().Err(err).Int64("id", id).Msg("Failed to delete checkpoint")
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Checkpoint not found", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetLastCheckpoint returns the most recent checkpoint for a stream.
// @Summary Get last checkpoint for stream
// @Description Returns the most recent checkpoint for a specific stream
// @Tags Admin, Replay
// @Accept json
// @Produce json
// @Param stream query string true "Stream name"
// @Success 200 {object} CheckpointResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/replay/checkpoints/last [get]
func (h *ReplayHandlers) GetLastCheckpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	streamName := r.URL.Query().Get("stream")

	if streamName == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "stream parameter is required", nil)
		return
	}

	checkpoint, err := h.checkpointStore.GetLastForStream(ctx, streamName)
	if err != nil {
		logging.Error().Err(err).Str("stream", streamName).Msg("Failed to get last checkpoint")
		respondError(w, http.StatusInternalServerError, "REPLAY_ERROR", "Failed to get last checkpoint", err)
		return
	}

	if checkpoint == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "No checkpoints found for stream", nil)
		return
	}

	writeJSON(w, checkpointToResponse(checkpoint))
}

// CleanupOldCheckpoints removes old completed checkpoints.
// @Summary Cleanup old checkpoints
// @Description Removes completed/error/canceled checkpoints older than specified duration
// @Tags Admin, Replay
// @Accept json
// @Produce json
// @Param older_than query string false "Duration string (e.g., '24h', '7d')" default(168h)
// @Success 200 {object} map[string]int64
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/replay/checkpoints/cleanup [post]
func (h *ReplayHandlers) CleanupOldCheckpoints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	olderThanStr := r.URL.Query().Get("older_than")
	if olderThanStr == "" {
		olderThanStr = "168h" // Default: 7 days
	}

	olderThan, err := time.ParseDuration(olderThanStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid duration format", err)
		return
	}

	deleted, err := h.checkpointStore.DeleteOld(ctx, olderThan)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to cleanup checkpoints")
		respondError(w, http.StatusInternalServerError, "REPLAY_ERROR", "Failed to cleanup checkpoints", err)
		return
	}

	writeJSON(w, map[string]int64{"deleted": deleted})
}

// StartReplay initiates an event replay operation.
// Note: This is a placeholder - full replay requires NATS connection.
// @Summary Start event replay
// @Description Initiates an event replay from the specified starting point
// @Tags Admin, Replay
// @Accept json
// @Produce json
// @Param request body ReplayRequest true "Replay configuration"
// @Success 202 {object} ReplayResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/replay/start [post]
func (h *ReplayHandlers) StartReplay(w http.ResponseWriter, r *http.Request) {
	var req ReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err)
		return
	}

	// Validate required fields
	if req.StreamName == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "stream_name is required", nil)
		return
	}
	if req.Topic == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "topic is required", nil)
		return
	}

	// Parse mode
	mode := parseReplayMode(req.Mode)

	// Validate mode-specific requirements
	if mode == eventprocessor.ReplayModeSequence && req.StartSequence == 0 {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "start_sequence is required for sequence mode", nil)
		return
	}
	if mode == eventprocessor.ReplayModeTime && req.StartTime == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "start_time is required for time mode", nil)
		return
	}

	// Parse timestamps if provided
	var startTime, stopTime time.Time
	if req.StartTime != "" {
		var err error
		startTime, err = time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid start_time format (use RFC3339)", err)
			return
		}
	}
	if req.StopTime != "" {
		var err error
		stopTime, err = time.Parse(time.RFC3339, req.StopTime)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid stop_time format (use RFC3339)", err)
			return
		}
	}

	// Create checkpoint for this replay operation
	ctx := r.Context()
	checkpoint := &eventprocessor.Checkpoint{
		ConsumerName:  "replay-" + time.Now().Format("20060102-150405"),
		StreamName:    req.StreamName,
		Status:        "pending",
		ReplayMode:    req.Mode,
		StartSequence: req.StartSequence,
		StartTime:     startTime,
	}

	if err := h.checkpointStore.Save(ctx, checkpoint); err != nil {
		logging.Error().Err(err).Msg("Failed to create replay checkpoint")
		respondError(w, http.StatusInternalServerError, "REPLAY_ERROR", "Failed to initialize replay", err)
		return
	}

	// Get the saved checkpoint to return ID
	saved, err := h.checkpointStore.Get(ctx, checkpoint.ConsumerName, checkpoint.StreamName)
	if err != nil || saved == nil {
		respondError(w, http.StatusInternalServerError, "REPLAY_ERROR", "Failed to retrieve checkpoint", err)
		return
	}

	// Log the replay request
	logging.Info().
		Int64("checkpoint_id", saved.ID).
		Str("stream", req.StreamName).
		Str("mode", req.Mode).
		Uint64("start_sequence", req.StartSequence).
		Time("start_time", startTime).
		Time("stop_time", stopTime).
		Bool("dry_run", req.DryRun).
		Msg("Replay operation initiated")

	// Note: Actual replay execution requires NATS connection and is handled
	// by a background worker that processes pending replay checkpoints.
	// This endpoint only creates the replay request.

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, ReplayResponse{
		ID:      saved.ID,
		Status:  "pending",
		Message: "Replay operation queued. Use GET /api/v1/admin/replay/checkpoints/{id} to check status.",
	})
}

// parseReplayMode converts string mode to ReplayMode.
func parseReplayMode(mode string) eventprocessor.ReplayMode {
	switch mode {
	case "all":
		return eventprocessor.ReplayModeAll
	case "sequence":
		return eventprocessor.ReplayModeSequence
	case "time":
		return eventprocessor.ReplayModeTime
	case "last_acked":
		return eventprocessor.ReplayModeLastAcked
	default:
		return eventprocessor.ReplayModeNew
	}
}

// checkpointToResponse converts a Checkpoint to CheckpointResponse.
func checkpointToResponse(cp *eventprocessor.Checkpoint) CheckpointResponse {
	return CheckpointResponse{
		ID:             cp.ID,
		ConsumerName:   cp.ConsumerName,
		StreamName:     cp.StreamName,
		LastSequence:   cp.LastSequence,
		LastTimestamp:  cp.LastTimestamp,
		ProcessedCount: cp.ProcessedCount,
		ErrorCount:     cp.ErrorCount,
		Status:         cp.Status,
		ReplayMode:     cp.ReplayMode,
		StartSequence:  cp.StartSequence,
		StartTime:      cp.StartTime,
		CreatedAt:      cp.CreatedAt,
		UpdatedAt:      cp.UpdatedAt,
	}
}
