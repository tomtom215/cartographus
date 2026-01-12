// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus application.
//
// handlers_plex_friends.go - Plex Friends and Sharing API Handlers
//
// This file contains HTTP handlers for managing Plex friends, library sharing,
// and managed users (Plex Home) through the plex.tv API.
//
// Endpoints:
//   - GET    /api/v1/plex/friends          - List Plex friends
//   - POST   /api/v1/plex/friends/invite   - Send friend invitation
//   - DELETE /api/v1/plex/friends/{id}     - Remove friend
//   - GET    /api/v1/plex/sharing          - List shared servers
//   - POST   /api/v1/plex/sharing          - Share libraries with user
//   - PUT    /api/v1/plex/sharing/{id}     - Update sharing settings
//   - DELETE /api/v1/plex/sharing/{id}     - Revoke sharing
//   - GET    /api/v1/plex/home/users       - List managed users
//   - POST   /api/v1/plex/home/users       - Create managed user
//   - DELETE /api/v1/plex/home/users/{id}  - Delete managed user
//   - PUT    /api/v1/plex/home/users/{id}  - Update managed user restrictions
//   - GET    /api/v1/plex/libraries        - List library sections for sharing UI
//   - GET    /api/v1/plex/identity         - Get server identity (machineIdentifier)
//
// Security:
//   - All endpoints require authentication AND admin role
//   - RBAC enforced via requirePlexAdmin() helper function
//   - Plex token is required in config
//   - All operations go through plex.tv API
//
// Observability:
//   - Prometheus metrics for Plex API operations
//   - Structured logging with correlation IDs
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"

	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/sync"
)

// validate is a reusable validator instance
var validate = validator.New()

// requirePlexAdmin is a helper that checks authentication and admin authorization.
// Returns true if the request should continue, false if an error response was sent.
// SECURITY: All Plex library management operations require admin role to prevent
// unauthorized users from modifying library access, sharing settings, or managed users.
func requirePlexAdmin(w http.ResponseWriter, hctx *HandlerContext) bool {
	if !hctx.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return false
	}
	if !hctx.IsAdmin {
		respondError(w, http.StatusForbidden, "ADMIN_REQUIRED", "Admin role required for Plex library management", nil)
		return false
	}
	return true
}

// getPlexTVClient creates a PlexTVClient from config
// Returns an error if Plex is not configured
func (h *Handler) getPlexTVClient() (*sync.PlexTVClient, error) {
	if !h.config.Plex.Enabled {
		return nil, ErrPlexNotEnabled
	}
	if h.config.Plex.Token == "" {
		return nil, ErrPlexTokenRequired
	}

	return sync.NewPlexTVClient(sync.PlexTVClientConfig{
		Token:      h.config.Plex.Token,
		MachineID:  h.config.Plex.ServerID,
		ClientID:   h.config.Plex.OAuthClientID,
		ClientName: "Cartographus",
	}), nil
}

// PlexFriendsList returns all Plex friends.
//
// Method: GET
// Path: /api/v1/plex/friends
//
// Response: PlexFriendsListResponse with all friends.
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexFriendsList(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	start := time.Now()

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	friends, err := client.ListFriends(r.Context())
	if err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Msg("Failed to list Plex friends")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to list friends", err)
		return
	}

	// Convert to response type
	friendResponses := make([]models.PlexFriendResponse, len(friends))
	for i := range friends {
		friendResponses[i] = models.PlexFriendResponse{
			ID:                friends[i].ID,
			UUID:              friends[i].UUID,
			Username:          friends[i].Username,
			Email:             friends[i].Email,
			Thumb:             friends[i].Thumb,
			Title:             friends[i].Title,
			Server:            friends[i].Server,
			Home:              friends[i].Home,
			AllowSync:         friends[i].AllowSync,
			AllowCameraUpload: friends[i].AllowCameraUpload,
			AllowChannels:     friends[i].AllowChannels,
			SharedSections:    friends[i].SharedSections,
			FilterMovies:      friends[i].FilterMovies,
			FilterTelevision:  friends[i].FilterTelevision,
			FilterMusic:       friends[i].FilterMusic,
			Status:            friends[i].Status,
		}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: models.PlexFriendsListResponse{
			Friends: friendResponses,
			Total:   len(friendResponses),
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexFriendsInvite sends a friend invitation.
//
// Method: POST
// Path: /api/v1/plex/friends/invite
//
// Request Body: PlexInviteFriendRequest
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexFriendsInvite(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	var req models.PlexInviteFriendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	if err := validate.Struct(req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request", err)
		return
	}

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	inviteReq := sync.PlexInviteRequest{
		Email:             req.Email,
		AllowSync:         req.AllowSync,
		AllowCameraUpload: req.AllowCameraUpload,
		AllowChannels:     req.AllowChannels,
	}

	if err := client.InviteFriend(r.Context(), inviteReq); err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Str("email", req.Email).
			Msg("Failed to invite friend")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to send invitation", err)
		return
	}

	log.Info().
		Str("request_id", hctx.RequestID).
		Str("email", req.Email).
		Msg("Friend invitation sent")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Friend invitation sent"},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// PlexFriendsRemove removes a friend.
//
// Method: DELETE
// Path: /api/v1/plex/friends/{id}
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexFriendsRemove(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	friendIDStr := chi.URLParam(r, "id")
	friendID, err := strconv.ParseInt(friendIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid friend ID", err)
		return
	}

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	if err := client.RemoveFriend(r.Context(), friendID); err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Int64("friend_id", friendID).
			Msg("Failed to remove friend")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to remove friend", err)
		return
	}

	log.Info().
		Str("request_id", hctx.RequestID).
		Int64("friend_id", friendID).
		Msg("Friend removed")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Friend removed"},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// PlexSharingList returns all shared server entries.
//
// Method: GET
// Path: /api/v1/plex/sharing
//
// Response: PlexSharedServersListResponse
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexSharingList(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	start := time.Now()

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	sharedServers, err := client.ListSharedServers(r.Context())
	if err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Msg("Failed to list shared servers")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to list shared servers", err)
		return
	}

	// Convert to response type
	responses := make([]models.PlexSharedServerResponse, len(sharedServers))
	for i := range sharedServers {
		responses[i] = models.PlexSharedServerResponse{
			ID:                sharedServers[i].ID,
			UserID:            sharedServers[i].UserID,
			Username:          sharedServers[i].Username,
			Email:             sharedServers[i].Email,
			Thumb:             sharedServers[i].Thumb,
			InvitedEmail:      sharedServers[i].InvitedEmail,
			AllowSync:         sharedServers[i].AllowSync,
			AllowCameraUpload: sharedServers[i].AllowCameraUpload,
			AllowChannels:     sharedServers[i].AllowChannels,
			FilterMovies:      sharedServers[i].FilterMovies,
			FilterTelevision:  sharedServers[i].FilterTelevision,
			FilterMusic:       sharedServers[i].FilterMusic,
		}
		if sharedServers[i].AcceptedAt > 0 {
			responses[i].AcceptedAt = time.Unix(sharedServers[i].AcceptedAt, 0)
		}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: models.PlexSharedServersListResponse{
			SharedServers: responses,
			Total:         len(responses),
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexSharingCreate shares libraries with a user.
//
// Method: POST
// Path: /api/v1/plex/sharing
//
// Request Body: PlexShareLibrariesRequest
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexSharingCreate(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	var req models.PlexShareLibrariesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	if err := validate.Struct(req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request", err)
		return
	}

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	shareReq := sync.PlexShareRequest{
		InvitedEmail:      req.Email,
		LibrarySectionIDs: req.LibrarySectionIDs,
		AllowSync:         req.AllowSync,
		AllowCameraUpload: req.AllowCameraUpload,
		AllowChannels:     req.AllowChannels,
		FilterMovies:      req.FilterMovies,
		FilterTelevision:  req.FilterTelevision,
		FilterMusic:       req.FilterMusic,
	}

	if err := client.ShareLibraries(r.Context(), &shareReq); err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Str("email", req.Email).
			Ints("sections", req.LibrarySectionIDs).
			Msg("Failed to share libraries")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to share libraries", err)
		return
	}

	log.Info().
		Str("request_id", hctx.RequestID).
		Str("email", req.Email).
		Ints("sections", req.LibrarySectionIDs).
		Msg("Libraries shared")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Libraries shared successfully"},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// PlexSharingUpdate updates sharing settings for a user.
//
// Method: PUT
// Path: /api/v1/plex/sharing/{id}
//
// Request Body: PlexUpdateSharingRequest
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexSharingUpdate(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	sharedServerIDStr := chi.URLParam(r, "id")
	sharedServerID, err := strconv.ParseInt(sharedServerIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid shared server ID", err)
		return
	}

	var req models.PlexUpdateSharingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	if err := validate.Struct(req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request", err)
		return
	}

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	if err := client.UpdateSharing(r.Context(), sharedServerID, req.LibrarySectionIDs); err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Int64("shared_server_id", sharedServerID).
			Msg("Failed to update sharing")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to update sharing", err)
		return
	}

	log.Info().
		Str("request_id", hctx.RequestID).
		Int64("shared_server_id", sharedServerID).
		Msg("Sharing updated")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Sharing settings updated"},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// PlexSharingRevoke revokes sharing for a user.
//
// Method: DELETE
// Path: /api/v1/plex/sharing/{id}
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexSharingRevoke(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	sharedServerIDStr := chi.URLParam(r, "id")
	sharedServerID, err := strconv.ParseInt(sharedServerIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid shared server ID", err)
		return
	}

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	if err := client.RevokeSharing(r.Context(), sharedServerID); err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Int64("shared_server_id", sharedServerID).
			Msg("Failed to revoke sharing")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to revoke sharing", err)
		return
	}

	log.Info().
		Str("request_id", hctx.RequestID).
		Int64("shared_server_id", sharedServerID).
		Msg("Sharing revoked")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Sharing revoked"},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// PlexManagedUsersList returns all managed users in Plex Home.
//
// Method: GET
// Path: /api/v1/plex/home/users
//
// Response: PlexManagedUsersListResponse
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexManagedUsersList(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	start := time.Now()

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	users, err := client.ListManagedUsers(r.Context())
	if err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Msg("Failed to list managed users")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to list managed users", err)
		return
	}

	// Convert to response type
	responses := make([]models.PlexManagedUserResponse, len(users))
	for i := range users {
		responses[i] = models.PlexManagedUserResponse{
			ID:                 users[i].ID,
			UUID:               users[i].UUID,
			Username:           users[i].Username,
			Title:              users[i].Title,
			Thumb:              users[i].Thumb,
			Restricted:         users[i].Restricted,
			RestrictionProfile: users[i].RestrictionProfile,
			Home:               users[i].Home,
			HomeAdmin:          users[i].HomeAdmin,
			Guest:              users[i].Guest,
			Protected:          users[i].Protected,
		}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: models.PlexManagedUsersListResponse{
			Users: responses,
			Total: len(responses),
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexManagedUsersCreate creates a new managed user.
//
// Method: POST
// Path: /api/v1/plex/home/users
//
// Request Body: PlexCreateManagedUserRequest
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexManagedUsersCreate(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	var req models.PlexCreateManagedUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	if err := validate.Struct(req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request", err)
		return
	}

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	createReq := sync.PlexCreateManagedUserRequest{
		Name:               req.Name,
		RestrictionProfile: req.RestrictionProfile,
	}

	user, err := client.CreateManagedUser(r.Context(), createReq)
	if err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Str("name", req.Name).
			Msg("Failed to create managed user")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to create managed user", err)
		return
	}

	log.Info().
		Str("request_id", hctx.RequestID).
		Str("name", req.Name).
		Int64("user_id", user.ID).
		Msg("Managed user created")

	respondJSON(w, http.StatusCreated, &models.APIResponse{
		Status: "success",
		Data: models.PlexManagedUserResponse{
			ID:                 user.ID,
			UUID:               user.UUID,
			Username:           user.Username,
			Title:              user.Title,
			Thumb:              user.Thumb,
			Restricted:         user.Restricted,
			RestrictionProfile: user.RestrictionProfile,
			Home:               user.Home,
			HomeAdmin:          user.HomeAdmin,
			Guest:              user.Guest,
			Protected:          user.Protected,
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// PlexManagedUsersDelete deletes a managed user.
//
// Method: DELETE
// Path: /api/v1/plex/home/users/{id}
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexManagedUsersDelete(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID", err)
		return
	}

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	if err := client.DeleteManagedUser(r.Context(), userID); err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Int64("user_id", userID).
			Msg("Failed to delete managed user")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to delete managed user", err)
		return
	}

	log.Info().
		Str("request_id", hctx.RequestID).
		Int64("user_id", userID).
		Msg("Managed user deleted")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Managed user deleted"},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// PlexManagedUsersUpdate updates a managed user's restrictions.
//
// Method: PUT
// Path: /api/v1/plex/home/users/{id}
//
// Request Body: PlexUpdateManagedUserRequest
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexManagedUsersUpdate(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID", err)
		return
	}

	var req models.PlexUpdateManagedUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
		return
	}

	if err := validate.Struct(req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request", err)
		return
	}

	client, err := h.getPlexTVClient()
	if err != nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", err.Error(), err)
		return
	}

	if err := client.UpdateManagedUserRestrictions(r.Context(), userID, req.RestrictionProfile); err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Int64("user_id", userID).
			Str("profile", req.RestrictionProfile).
			Msg("Failed to update managed user restrictions")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to update restrictions", err)
		return
	}

	log.Info().
		Str("request_id", hctx.RequestID).
		Int64("user_id", userID).
		Str("profile", req.RestrictionProfile).
		Msg("Managed user restrictions updated")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Restrictions updated"},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// PlexLibrariesList returns library sections for the sharing UI.
//
// Method: GET
// Path: /api/v1/plex/libraries
//
// Response: PlexLibrarySectionsListResponse
//
// Authentication: Required
// Authorization: Admin role required
func (h *Handler) PlexLibrariesList(w http.ResponseWriter, r *http.Request) {
	hctx := GetHandlerContext(r)
	if !requirePlexAdmin(w, hctx) {
		return
	}

	start := time.Now()

	// Use the existing Plex sync client to get library sections
	if h.sync == nil {
		respondError(w, http.StatusBadRequest, "PLEX_NOT_CONFIGURED", "Plex sync not configured", nil)
		return
	}

	// Get libraries from Plex via sync manager
	sectionsResp, err := h.sync.GetPlexLibrarySections(r.Context())
	if err != nil {
		log.Error().Err(err).
			Str("request_id", hctx.RequestID).
			Msg("Failed to get library sections")
		respondError(w, http.StatusInternalServerError, "PLEX_API_ERROR", "Failed to get libraries", err)
		return
	}

	// Convert to response type
	sections := sectionsResp.MediaContainer.Directory
	responses := make([]models.PlexLibrarySectionResponse, len(sections))
	for i := range sections {
		// Parse key as integer for ID
		id := 0
		if sections[i].Key != "" {
			if parsed, err := strconv.Atoi(sections[i].Key); err == nil {
				id = parsed
			}
		}
		responses[i] = models.PlexLibrarySectionResponse{
			ID:    id,
			Key:   sections[i].Key,
			Type:  sections[i].Type,
			Title: sections[i].Title,
			Thumb: sections[i].Thumb,
		}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: models.PlexLibrarySectionsListResponse{
			Sections: responses,
			Total:    len(responses),
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}
