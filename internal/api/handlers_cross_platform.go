// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
handlers_cross_platform.go - Cross-Platform Content and User Linking API Handlers

Phase 3 PRR: Provides HTTP endpoints for cross-platform content reconciliation
and user linking features, enabling unified analytics across Plex/Jellyfin/Emby.

Endpoints:
  - POST /api/v1/content/link - Create or update content mapping
  - GET  /api/v1/content/lookup - Lookup content by external ID
  - POST /api/v1/users/link - Create user link between platforms
  - GET  /api/v1/users/linked - Get all linked users for a user
  - DELETE /api/v1/users/link - Remove user link
  - GET  /api/v1/users/suggest-links - Suggest user links based on email matching
*/

package api

import (
	"net/http"
	"strconv"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/database"
)

// writeJSONResponse writes a JSON response with proper headers.
// This is a simple helper for cross-platform handlers.
func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	//nolint:errcheck // HTTP response write errors are not recoverable
	json.NewEncoder(w).Encode(data)
}

// ========================================
// Content Mapping Request/Response Types
// ========================================

// ContentMappingRequest represents a request to create/update content mapping.
type ContentMappingRequest struct {
	IMDbID         *string `json:"imdb_id,omitempty"`
	TMDbID         *int    `json:"tmdb_id,omitempty"`
	TVDbID         *int    `json:"tvdb_id,omitempty"`
	PlexRatingKey  *string `json:"plex_rating_key,omitempty"`
	JellyfinItemID *string `json:"jellyfin_item_id,omitempty"`
	EmbyItemID     *string `json:"emby_item_id,omitempty"`
	Title          string  `json:"title" validate:"required"`
	MediaType      string  `json:"media_type" validate:"required,oneof=movie show episode"`
	Year           *int    `json:"year,omitempty"`
}

// ContentMappingResponse represents a content mapping API response.
type ContentMappingResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message,omitempty"`
	Data    *database.ContentMapping `json:"data,omitempty"`
	Created bool                     `json:"created,omitempty"`
}

// ========================================
// User Linking Request/Response Types
// ========================================

// UserLinkRequest represents a request to create a user link.
type UserLinkRequest struct {
	PrimaryUserID int    `json:"primary_user_id" validate:"required,gt=0"`
	LinkedUserID  int    `json:"linked_user_id" validate:"required,gt=0"`
	LinkType      string `json:"link_type" validate:"required,oneof=manual email plex_home"`
}

// UserLinkResponse represents a user link API response.
type UserLinkResponse struct {
	Success bool               `json:"success"`
	Message string             `json:"message,omitempty"`
	Data    *database.UserLink `json:"data,omitempty"`
}

// LinkedUsersResponse represents a response containing linked users.
type LinkedUsersResponse struct {
	Success bool                       `json:"success"`
	UserIDs []int                      `json:"user_ids,omitempty"`
	Users   []*database.LinkedUserInfo `json:"users,omitempty"`
}

// SuggestedLinksResponse represents suggested user links based on email matching.
type SuggestedLinksResponse struct {
	Success     bool                                  `json:"success"`
	Suggestions map[string][]*database.LinkedUserInfo `json:"suggestions"`
}

// ========================================
// Content Mapping Handlers
// ========================================

// ContentMappingCreate creates or updates a content mapping.
//
// POST /api/v1/content/link
//
// Request body:
//
//	{
//	  "imdb_id": "tt1234567",        // At least one external ID required
//	  "tmdb_id": 12345,
//	  "tvdb_id": 67890,
//	  "plex_rating_key": "abc123",   // Optional platform-specific IDs
//	  "jellyfin_item_id": "uuid",
//	  "emby_item_id": "emby123",
//	  "title": "Movie Title",        // Required
//	  "media_type": "movie",         // Required: movie, show, episode
//	  "year": 2024                   // Optional
//	}
//
// Response: ContentMappingResponse with the created/existing mapping
func (h *Handler) ContentMappingCreate(w http.ResponseWriter, r *http.Request) {
	var req ContentMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Invalid JSON body",
		})
		return
	}

	// Validate required fields
	if req.Title == "" {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "title is required",
		})
		return
	}
	if req.MediaType == "" {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "media_type is required (movie, show, or episode)",
		})
		return
	}

	// Require at least one external ID
	if req.IMDbID == nil && req.TMDbID == nil && req.TVDbID == nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "At least one external ID (imdb_id, tmdb_id, or tvdb_id) is required",
		})
		return
	}

	lookup := &database.ContentMappingLookup{
		IMDbID:         req.IMDbID,
		TMDbID:         req.TMDbID,
		TVDbID:         req.TVDbID,
		PlexRatingKey:  req.PlexRatingKey,
		JellyfinItemID: req.JellyfinItemID,
		EmbyItemID:     req.EmbyItemID,
		Title:          req.Title,
		MediaType:      req.MediaType,
		Year:           req.Year,
	}

	mapping, created, err := h.db.GetOrCreateContentMapping(r.Context(), lookup)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	message := "Content mapping retrieved"
	if created {
		message = "Content mapping created"
	}

	writeJSONResponse(w, http.StatusOK, ContentMappingResponse{
		Success: true,
		Message: message,
		Data:    mapping,
		Created: created,
	})
}

// ContentMappingLookup looks up a content mapping by external ID.
//
// GET /api/v1/content/lookup?type=imdb&id=tt1234567
// GET /api/v1/content/lookup?type=tmdb&id=12345
// GET /api/v1/content/lookup?type=plex&id=abc123
//
// Query parameters:
//   - type: ID type (imdb, tmdb, tvdb, plex, jellyfin, emby)
//   - id: The external ID value
//
// Response: ContentMappingResponse with the found mapping
func (h *Handler) ContentMappingLookup(w http.ResponseWriter, r *http.Request) {
	idType := r.URL.Query().Get("type")
	id := r.URL.Query().Get("id")

	if idType == "" || id == "" {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Both 'type' and 'id' query parameters are required",
		})
		return
	}

	// Validate type
	validTypes := map[string]bool{"imdb": true, "tmdb": true, "tvdb": true, "plex": true, "jellyfin": true, "emby": true}
	if !validTypes[idType] {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Invalid type. Must be: imdb, tmdb, tvdb, plex, jellyfin, or emby",
		})
		return
	}

	// Convert numeric IDs
	var lookupID interface{} = id
	if idType == "tmdb" || idType == "tvdb" {
		numID, err := strconv.Atoi(id)
		if err != nil {
			writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "ID must be numeric for tmdb/tvdb types",
			})
			return
		}
		lookupID = numID
	}

	mapping, err := h.db.GetContentMappingByExternalID(r.Context(), idType, lookupID)
	if err != nil {
		writeJSONResponse(w, http.StatusNotFound, map[string]interface{}{
			"success": false,
			"error":   "Content mapping not found",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, ContentMappingResponse{
		Success: true,
		Data:    mapping,
	})
}

// ContentMappingLinkPlex links a Plex rating_key to an existing content mapping.
//
// POST /api/v1/content/{id}/link/plex
//
// Request body: {"rating_key": "12345"}
func (h *Handler) ContentMappingLinkPlex(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Invalid mapping ID",
		})
		return
	}

	var req struct {
		RatingKey string `json:"rating_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RatingKey == "" {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "rating_key is required",
		})
		return
	}

	if err := h.db.LinkPlexContent(r.Context(), id, req.RatingKey); err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Plex content linked successfully",
	})
}

// ContentMappingLinkJellyfin links a Jellyfin item ID to an existing content mapping.
//
// POST /api/v1/content/{id}/link/jellyfin
//
// Request body: {"item_id": "uuid-string"}
func (h *Handler) ContentMappingLinkJellyfin(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Invalid mapping ID",
		})
		return
	}

	var req struct {
		ItemID string `json:"item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ItemID == "" {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "item_id is required",
		})
		return
	}

	if err := h.db.LinkJellyfinContent(r.Context(), id, req.ItemID); err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Jellyfin content linked successfully",
	})
}

// ContentMappingLinkEmby links an Emby item ID to an existing content mapping.
//
// POST /api/v1/content/{id}/link/emby
//
// Request body: {"item_id": "emby-item-id"}
func (h *Handler) ContentMappingLinkEmby(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Invalid mapping ID",
		})
		return
	}

	var req struct {
		ItemID string `json:"item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ItemID == "" {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "item_id is required",
		})
		return
	}

	if err := h.db.LinkEmbyContent(r.Context(), id, req.ItemID); err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Emby content linked successfully",
	})
}

// ========================================
// User Linking Handlers
// ========================================

// UserLinkCreate creates a link between two user identities.
//
// POST /api/v1/users/link
//
// Request body:
//
//	{
//	  "primary_user_id": 1,      // Internal user ID from user_mappings
//	  "linked_user_id": 2,       // Internal user ID to link
//	  "link_type": "manual"      // manual, email, or plex_home
//	}
//
// Response: UserLinkResponse with the created link
func (h *Handler) UserLinkCreate(w http.ResponseWriter, r *http.Request) {
	var req UserLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Invalid JSON body",
		})
		return
	}

	// Validate required fields
	if req.PrimaryUserID <= 0 || req.LinkedUserID <= 0 {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "primary_user_id and linked_user_id must be positive integers",
		})
		return
	}
	if req.LinkType == "" {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "link_type is required (manual, email, or plex_home)",
		})
		return
	}

	// Get username from context for audit trail
	var createdBy *string
	if username := r.Context().Value("username"); username != nil {
		if u, ok := username.(string); ok {
			createdBy = &u
		}
	}

	link, err := h.db.CreateUserLink(r.Context(), req.PrimaryUserID, req.LinkedUserID, req.LinkType, createdBy)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, UserLinkResponse{
		Success: true,
		Message: "User link created",
		Data:    link,
	})
}

// UserLinkedGet retrieves all users linked to a specific user.
//
// GET /api/v1/users/{id}/linked
//
// Path parameters:
//   - id: Internal user ID
//
// Response: LinkedUsersResponse with linked user IDs and details
func (h *Handler) UserLinkedGet(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Invalid user ID",
		})
		return
	}

	// Get all linked user IDs
	userIDs, err := h.db.GetAllLinkedUserIDs(r.Context(), id)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get linked user details
	users, err := h.db.GetLinkedUsers(r.Context(), id)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, LinkedUsersResponse{
		Success: true,
		UserIDs: userIDs,
		Users:   users,
	})
}

// UserLinkDelete removes a link between two users.
//
// DELETE /api/v1/users/link?primary_id=1&linked_id=2
//
// Query parameters:
//   - primary_id: First user's internal ID
//   - linked_id: Second user's internal ID
//
// Response: Success/error message
func (h *Handler) UserLinkDelete(w http.ResponseWriter, r *http.Request) {
	primaryIDStr := r.URL.Query().Get("primary_id")
	linkedIDStr := r.URL.Query().Get("linked_id")

	primaryID, err1 := strconv.Atoi(primaryIDStr)
	linkedID, err2 := strconv.Atoi(linkedIDStr)

	if err1 != nil || err2 != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Both primary_id and linked_id query parameters are required and must be integers",
		})
		return
	}

	if err := h.db.DeleteUserLink(r.Context(), primaryID, linkedID); err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "User link deleted",
	})
}

// UserSuggestLinks suggests potential user links based on email matching.
//
// GET /api/v1/users/suggest-links
//
// Response: SuggestedLinksResponse with groups of users sharing email addresses
func (h *Handler) UserSuggestLinks(w http.ResponseWriter, r *http.Request) {
	suggestions, err := h.db.FindUsersByEmail(r.Context())
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, SuggestedLinksResponse{
		Success:     true,
		Suggestions: suggestions,
	})
}

// ========================================
// Cross-Platform Analytics Handlers
// ========================================

// CrossPlatformUserStats returns aggregated watch statistics for linked users.
//
// GET /api/v1/analytics/cross-platform/user/{id}
//
// Returns watch statistics aggregated across all linked user identities,
// enabling unified analytics for users who use multiple media servers.
//
// Path parameters:
//   - id: Internal user ID (will include all linked users)
//
// Query parameters:
//   - start_date: Filter start date (RFC3339)
//   - end_date: Filter end date (RFC3339)
//
// Response: Aggregated watch statistics across all linked identities
func (h *Handler) CrossPlatformUserStats(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Invalid user ID",
		})
		return
	}

	// Get all linked user IDs
	linkedIDs, err := h.db.GetAllLinkedUserIDs(r.Context(), userID)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Build filter with all linked user IDs
	filter := h.buildFilter(r)

	// Convert int slice to string slice for users filter
	usernames := make([]string, 0)

	// Get stats for each linked user and aggregate
	var totalPlays, totalDuration int
	platformStats := make(map[string]int)

	for _, linkedID := range linkedIDs {
		// Get user mappings to find usernames
		mappings, err := h.db.GetUserMappingByInternal(r.Context(), linkedID)
		if err != nil {
			continue
		}
		for _, m := range mappings {
			if m.Username != nil {
				usernames = append(usernames, *m.Username)
				platformStats[m.Source]++
			}
		}
	}

	// Query with all linked usernames
	filter.Users = usernames
	stats, err := h.db.GetUserPlayStats(r.Context(), filter)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if stats != nil {
		totalPlays = stats.TotalPlays
		totalDuration = stats.TotalDuration
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success":           true,
		"user_id":           userID,
		"linked_user_ids":   linkedIDs,
		"total_plays":       totalPlays,
		"total_duration":    totalDuration,
		"platforms_used":    platformStats,
		"linked_identities": len(linkedIDs),
	})
}

// CrossPlatformContentStats returns watch statistics for content across all platforms.
//
// GET /api/v1/analytics/cross-platform/content/{id}
//
// Returns watch statistics for content that has been linked across multiple
// media servers, enabling unified "most watched" analytics.
//
// Path parameters:
//   - id: Content mapping ID
//
// Response: Aggregated content watch statistics
func (h *Handler) CrossPlatformContentStats(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	mappingID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Invalid content mapping ID",
		})
		return
	}

	// Get the content mapping
	mapping, err := h.db.GetContentMappingByID(r.Context(), mappingID)
	if err != nil {
		writeJSONResponse(w, http.StatusNotFound, map[string]interface{}{
			"success": false,
			"error":   "Content mapping not found",
		})
		return
	}

	// Get cross-platform watch count
	totalPlays, err := h.db.GetCrossplatformWatchCount(r.Context(), mappingID)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Build platform availability map
	platforms := make(map[string]bool)
	if mapping.PlexRatingKey != nil {
		platforms["plex"] = true
	}
	if mapping.JellyfinItemID != nil {
		platforms["jellyfin"] = true
	}
	if mapping.EmbyItemID != nil {
		platforms["emby"] = true
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success":             true,
		"content_mapping_id":  mappingID,
		"title":               mapping.Title,
		"media_type":          mapping.MediaType,
		"year":                mapping.Year,
		"total_plays":         totalPlays,
		"platforms_available": platforms,
		"external_ids": map[string]interface{}{
			"imdb": mapping.IMDbID,
			"tmdb": mapping.TMDbID,
			"tvdb": mapping.TVDbID,
		},
	})
}

// CrossPlatformSummary provides a high-level summary of cross-platform usage.
//
// GET /api/v1/analytics/cross-platform/summary
//
// Returns summary statistics about cross-platform content and user linking.
// Non-critical stats default to 0 on query errors for graceful degradation.
func (h *Handler) CrossPlatformSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Count content mappings (defaults to 0 on error)
	var contentMappingsCount int
	row := h.db.QueryRow(ctx, "SELECT COUNT(*) FROM content_mappings")
	//nolint:errcheck // Non-critical summary stat, defaults to 0
	_ = row.Scan(&contentMappingsCount)

	// Count user links (defaults to 0 on error)
	var userLinksCount int
	row = h.db.QueryRow(ctx, "SELECT COUNT(*) FROM user_links")
	//nolint:errcheck // Non-critical summary stat, defaults to 0
	_ = row.Scan(&userLinksCount)

	// Count content by platform availability (defaults to 0 on error)
	var plexContent, jellyfinContent, embyContent int
	row = h.db.QueryRow(ctx, "SELECT COUNT(*) FROM content_mappings WHERE plex_rating_key IS NOT NULL")
	//nolint:errcheck // Non-critical summary stat, defaults to 0
	_ = row.Scan(&plexContent)
	row = h.db.QueryRow(ctx, "SELECT COUNT(*) FROM content_mappings WHERE jellyfin_item_id IS NOT NULL")
	//nolint:errcheck // Non-critical summary stat, defaults to 0
	_ = row.Scan(&jellyfinContent)
	row = h.db.QueryRow(ctx, "SELECT COUNT(*) FROM content_mappings WHERE emby_item_id IS NOT NULL")
	//nolint:errcheck // Non-critical summary stat, defaults to 0
	_ = row.Scan(&embyContent)

	// Count multi-platform content (linked to 2+ platforms, defaults to 0 on error)
	var multiPlatformContent int
	row = h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM content_mappings
		WHERE (CASE WHEN plex_rating_key IS NOT NULL THEN 1 ELSE 0 END +
		       CASE WHEN jellyfin_item_id IS NOT NULL THEN 1 ELSE 0 END +
		       CASE WHEN emby_item_id IS NOT NULL THEN 1 ELSE 0 END) >= 2
	`)
	//nolint:errcheck // Non-critical summary stat, defaults to 0
	_ = row.Scan(&multiPlatformContent)

	// Get user mapping stats (nil on error for graceful degradation)
	//nolint:errcheck // Non-critical summary stat, nil on error
	userStats, _ := h.db.GetUserMappingStats(ctx)

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"content_mappings": map[string]interface{}{
			"total":          contentMappingsCount,
			"plex":           plexContent,
			"jellyfin":       jellyfinContent,
			"emby":           embyContent,
			"multi_platform": multiPlatformContent,
		},
		"user_links": map[string]interface{}{
			"total": userLinksCount,
		},
		"user_mappings": userStats,
	})
}
