// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus application.
//
// handlers_pat.go - Personal Access Token API Handlers
//
// This file contains HTTP handlers for managing Personal Access Tokens (PATs).
// PATs enable programmatic API access with scoped permissions.
//
// Endpoints:
//   - GET    /api/v1/user/tokens          - List user's PATs
//   - POST   /api/v1/user/tokens          - Create new PAT
//   - GET    /api/v1/user/tokens/{id}     - Get PAT details
//   - DELETE /api/v1/user/tokens/{id}     - Revoke/delete PAT
//   - POST   /api/v1/user/tokens/{id}/regenerate - Regenerate PAT secret
//   - GET    /api/v1/user/tokens/stats    - Get PAT usage statistics
//
// Security:
//   - All endpoints require authentication
//   - Users can only manage their own tokens
//   - Plaintext tokens are only shown once during creation
//
// Observability:
//   - Prometheus metrics for PAT operations
//   - Structured logging with correlation IDs
//   - Audit logging for security events
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/metrics"
	"github.com/tomtom215/cartographus/internal/models"
)

// PATList returns all PATs for the authenticated user.
//
// Method: GET
// Path: /api/v1/user/tokens
//
// Response: ListPATsResponse with all tokens for the user.
// Note: Token hashes are never included in the response.
//
// Authentication: Required
func (h *Handler) PATList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	hctx := GetHandlerContext(r)
	if !hctx.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	start := time.Now()

	patManager := auth.NewPATManager(h.db, &log.Logger)
	tokens, err := patManager.List(r.Context(), hctx.UserID)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", hctx.UserID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to list PATs")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list tokens", err)
		return
	}

	metrics.RecordPATOperation("list", true)

	response := models.ListPATsResponse{
		Tokens:     tokens,
		TotalCount: len(tokens),
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   response,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PATCreate creates a new PAT for the authenticated user.
//
// Method: POST
// Path: /api/v1/user/tokens
//
// Request Body: CreatePATRequest
//
// Response: CreatePATResponse including the plaintext token.
// IMPORTANT: The plaintext token is only shown once!
//
// Authentication: Required
func (h *Handler) PATCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	hctx := GetHandlerContext(r)
	if !hctx.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	// Parse request body
	var req models.CreatePATRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	// Validate request
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Token name is required", nil)
		return
	}
	if len(req.Scopes) == 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "At least one scope is required", nil)
		return
	}

	// Check for admin scope - only admins can create admin tokens
	for _, scope := range req.Scopes {
		if scope == models.ScopeAdmin && !hctx.IsAdmin {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "Only admins can create admin tokens", nil)
			return
		}
	}

	start := time.Now()

	patManager := auth.NewPATManager(h.db, &log.Logger)
	token, plaintextToken, err := patManager.Create(r.Context(), hctx.UserID, hctx.Username, &req)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", hctx.UserID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to create PAT")
		metrics.RecordPATOperation("create", false)
		respondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create token", err)
		return
	}

	metrics.RecordPATOperation("create", true)

	log.Info().
		Str("token_id", token.ID).
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Str("name", req.Name).
		Int("scopes_count", len(req.Scopes)).
		Msg("PAT created via API")

	response := models.CreatePATResponse{
		Token:          token,
		PlaintextToken: plaintextToken,
	}

	respondJSON(w, http.StatusCreated, &models.APIResponse{
		Status: "success",
		Data:   response,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PATGet retrieves a specific PAT by ID.
//
// Method: GET
// Path: /api/v1/user/tokens/{id}
//
// URL Parameters:
//   - id: The token ID
//
// Response: PersonalAccessToken (without hash)
//
// Authentication: Required
// Authorization: Users can only view their own tokens
func (h *Handler) PATGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	hctx := GetHandlerContext(r)
	if !hctx.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	tokenID := chi.URLParam(r, "id")
	if tokenID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Token ID is required", nil)
		return
	}

	start := time.Now()

	patManager := auth.NewPATManager(h.db, &log.Logger)
	token, err := patManager.Get(r.Context(), tokenID, hctx.UserID)
	if err != nil {
		if err.Error() == "access denied" {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil)
			return
		}
		log.Error().Err(err).
			Str("token_id", tokenID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get PAT")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get token", err)
		return
	}

	if token == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Token not found", nil)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   token,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PATRevoke revokes a PAT.
//
// Method: DELETE
// Path: /api/v1/user/tokens/{id}
//
// URL Parameters:
//   - id: The token ID
//
// Request Body (optional): RevokePATRequest
//
// Authentication: Required
// Authorization: Users can only revoke their own tokens
func (h *Handler) PATRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	hctx := GetHandlerContext(r)
	if !hctx.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	tokenID := chi.URLParam(r, "id")
	if tokenID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Token ID is required", nil)
		return
	}

	// Parse optional request body for reason
	var req models.RevokePATRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Ignore decode errors - reason is optional
			req.Reason = ""
		}
	}

	start := time.Now()

	// First verify ownership
	patManager := auth.NewPATManager(h.db, &log.Logger)
	token, err := patManager.Get(r.Context(), tokenID, hctx.UserID)
	if err != nil {
		if err.Error() == "access denied" {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil)
			return
		}
		log.Error().Err(err).
			Str("token_id", tokenID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get PAT for revocation")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify token ownership", err)
		return
	}

	if token == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Token not found", nil)
		return
	}

	// Revoke the token
	if err := patManager.Revoke(r.Context(), tokenID, hctx.UserID, req.Reason); err != nil {
		log.Error().Err(err).
			Str("token_id", tokenID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to revoke PAT")
		metrics.RecordPATOperation("revoke", false)
		respondError(w, http.StatusInternalServerError, "REVOKE_ERROR", "Failed to revoke token", err)
		return
	}

	metrics.RecordPATOperation("revoke", true)

	log.Info().
		Str("token_id", tokenID).
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Str("reason", req.Reason).
		Msg("PAT revoked via API")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]string{
			"message": "Token revoked successfully",
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PATRegenerate regenerates the secret for an existing PAT.
//
// Method: POST
// Path: /api/v1/user/tokens/{id}/regenerate
//
// URL Parameters:
//   - id: The token ID
//
// Response: CreatePATResponse including the new plaintext token.
// IMPORTANT: The new plaintext token is only shown once!
//
// Authentication: Required
// Authorization: Users can only regenerate their own tokens
func (h *Handler) PATRegenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	hctx := GetHandlerContext(r)
	if !hctx.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	tokenID := chi.URLParam(r, "id")
	if tokenID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Token ID is required", nil)
		return
	}

	start := time.Now()

	patManager := auth.NewPATManager(h.db, &log.Logger)
	token, plaintextToken, err := patManager.Regenerate(r.Context(), tokenID, hctx.UserID)
	if err != nil {
		if err.Error() == "access denied" {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil)
			return
		}
		if err.Error() == "token not found" {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Token not found", nil)
			return
		}
		log.Error().Err(err).
			Str("token_id", tokenID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to regenerate PAT")
		metrics.RecordPATOperation("regenerate", false)
		respondError(w, http.StatusInternalServerError, "REGENERATE_ERROR", "Failed to regenerate token", err)
		return
	}

	metrics.RecordPATOperation("regenerate", true)

	log.Info().
		Str("token_id", tokenID).
		Str("user_id", hctx.UserID).
		Str("request_id", hctx.RequestID).
		Msg("PAT regenerated via API")

	response := models.CreatePATResponse{
		Token:          token,
		PlaintextToken: plaintextToken,
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   response,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PATStats returns aggregated PAT statistics for the authenticated user.
//
// Method: GET
// Path: /api/v1/user/tokens/stats
//
// Response: PATStats with token counts and usage metrics.
//
// Authentication: Required
func (h *Handler) PATStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	hctx := GetHandlerContext(r)
	if !hctx.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	start := time.Now()

	patManager := auth.NewPATManager(h.db, &log.Logger)
	stats, err := patManager.GetStats(r.Context(), hctx.UserID)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", hctx.UserID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get PAT stats")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get token statistics", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   stats,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PATUsageLogs returns usage logs for a specific PAT.
//
// Method: GET
// Path: /api/v1/user/tokens/{id}/logs
//
// URL Parameters:
//   - id: The token ID
//
// Query Parameters:
//   - limit: Maximum number of log entries (default: 100, max: 1000)
//
// Response: Array of PATUsageLog entries.
//
// Authentication: Required
// Authorization: Users can only view logs for their own tokens
func (h *Handler) PATUsageLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	hctx := GetHandlerContext(r)
	if !hctx.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	tokenID := chi.URLParam(r, "id")
	if tokenID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Token ID is required", nil)
		return
	}

	// First verify ownership
	patManager := auth.NewPATManager(h.db, &log.Logger)
	token, err := patManager.Get(r.Context(), tokenID, hctx.UserID)
	if err != nil {
		if err.Error() == "access denied" {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil)
			return
		}
		log.Error().Err(err).
			Str("token_id", tokenID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to verify token ownership")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify token ownership", err)
		return
	}

	if token == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Token not found", nil)
		return
	}

	// Parse limit
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := parseInt(limitStr); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	start := time.Now()

	logs, err := patManager.GetUsageLogs(r.Context(), tokenID, limit)
	if err != nil {
		log.Error().Err(err).
			Str("token_id", tokenID).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get PAT usage logs")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get usage logs", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   logs,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// parseInt is a helper to parse string to int.
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
