// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OAuth flow handlers.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// PlexLogin initiates the Plex PIN-based authentication flow.
// GET /api/auth/plex/login
func (h *FlowHandlers) PlexLogin(w http.ResponseWriter, r *http.Request) {
	if h.plexFlow == nil {
		http.Error(w, "Plex authentication not configured", http.StatusServiceUnavailable)
		return
	}

	// Request a PIN from Plex
	pin, err := h.plexFlow.RequestPIN(r.Context())
	if err != nil {
		logging.Error().Err(err).Msg("Failed to request Plex PIN")
		http.Error(w, "Failed to initiate Plex login", http.StatusInternalServerError)
		return
	}

	// Return PIN info as JSON (frontend will redirect user to Plex auth page)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"pin_id":   pin.ID,
		"pin_code": pin.Code,
		"auth_url": pin.AuthURL,
		"expires":  pin.ExpiresAt,
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode PIN response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// PlexPoll checks if a Plex PIN has been authorized.
// GET /api/auth/plex/poll?pin_id=12345
func (h *FlowHandlers) PlexPoll(w http.ResponseWriter, r *http.Request) {
	if h.plexFlow == nil {
		http.Error(w, "Plex authentication not configured", http.StatusServiceUnavailable)
		return
	}

	// Get PIN ID from query
	pinIDStr := r.URL.Query().Get("pin_id")
	if pinIDStr == "" {
		http.Error(w, "Missing pin_id parameter", http.StatusBadRequest)
		return
	}

	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin_id parameter", http.StatusBadRequest)
		return
	}

	// Check PIN status
	result, err := h.plexFlow.CheckPIN(r.Context(), pinID)
	if err != nil {
		if errors.Is(err, ErrPINNotFound) {
			http.Error(w, "PIN expired or not found", http.StatusNotFound)
			return
		}
		logging.Error().Err(err).Msg("Failed to check Plex PIN")
		http.Error(w, "Failed to check PIN status", http.StatusInternalServerError)
		return
	}

	// Return status
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"authorized": result.Authorized,
		"expires":    result.ExpiresAt,
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode poll response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// PlexCallbackRequest is the request body for Plex callback.
type PlexCallbackRequest struct {
	PinID             int    `json:"pin_id"`
	PostLoginRedirect string `json:"redirect_uri,omitempty"`
}

// PlexCallback completes the Plex authentication after PIN authorization.
// POST /api/auth/plex/callback
func (h *FlowHandlers) PlexCallback(w http.ResponseWriter, r *http.Request) {
	if h.plexFlow == nil {
		http.Error(w, "Plex authentication not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse request body
	var req PlexCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PinID == 0 {
		http.Error(w, "Missing pin_id", http.StatusBadRequest)
		return
	}

	// Check PIN status
	result, err := h.plexFlow.CheckPIN(r.Context(), req.PinID)
	if err != nil {
		if errors.Is(err, ErrPINNotFound) {
			http.Error(w, "PIN expired or not found", http.StatusNotFound)
			return
		}
		logging.Error().Err(err).Msg("Failed to check Plex PIN")
		http.Error(w, "Failed to verify PIN", http.StatusInternalServerError)
		return
	}

	if !result.Authorized {
		http.Error(w, "PIN not yet authorized", http.StatusBadRequest)
		return
	}

	// Get user info with the auth token
	subject, err := h.plexFlow.GetUserInfo(r.Context(), result.AuthToken)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to get Plex user info")
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Store auth token in metadata
	if subject.Metadata == nil {
		subject.Metadata = make(map[string]string)
	}
	subject.Metadata["plex_token"] = result.AuthToken

	// Create session and return success
	h.completePlexCallback(w, r, subject, req.PostLoginRedirect)
}

// completePlexCallback creates a session and returns success response.
func (h *FlowHandlers) completePlexCallback(w http.ResponseWriter, r *http.Request, subject *AuthSubject, postLoginRedirect string) {
	// Create session
	session, err := h.sessionMiddleware.CreateSession(r.Context(), w, subject)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to create session")
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	logging.Info().Str("user", subject.Username).Str("session_id", session.ID).Msg("Plex login successful")

	// Return success with redirect URL (validate to prevent open redirect attacks)
	redirectURL := validateRedirectURI(postLoginRedirect)
	if redirectURL == "" {
		redirectURL = h.config.DefaultPostLoginRedirect
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"redirect_url": redirectURL,
		"user": map[string]interface{}{
			"id":       subject.ID,
			"username": subject.Username,
			"email":    subject.Email,
			"roles":    subject.Roles,
		},
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode callback response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
