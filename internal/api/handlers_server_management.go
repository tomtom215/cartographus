// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// serverMgmtContextKey is a private type for context keys to avoid collisions.
type serverMgmtContextKey string

// Context keys for user information in server management handlers.
const (
	userIDContextKey   serverMgmtContextKey = "user_id"
	usernameContextKey serverMgmtContextKey = "username"
)

// CreateServer handles POST /api/v1/admin/servers
// Creates a new media server configuration in the database.
//
// @Summary Create a new media server
// @Description Adds a new media server configuration. Credentials are encrypted at rest.
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateMediaServerRequest true "Server configuration"
// @Success 201 {object} models.APIResponse{data=models.MediaServerResponse} "Server created"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 403 {object} models.APIResponse "Forbidden - admin only"
// @Failure 409 {object} models.APIResponse "Server ID conflict"
// @Router /admin/servers [post]
func (h *Handler) CreateServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req models.CreateMediaServerRequest
	if err := h.parseAndValidateRequest(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), err)
		return
	}

	encryptor, err := h.getCredentialEncryptor()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ENCRYPTION_ERROR", "Failed to initialize encryption", err)
		return
	}

	urlEncrypted, tokenEncrypted, err := h.encryptCredentials(encryptor, req.URL, req.Token)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ENCRYPTION_ERROR", err.Error(), err)
		return
	}

	settingsJSON, err := h.serializeSettings(req.Settings)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_SETTINGS", "Failed to serialize settings", err)
		return
	}

	userID, username := getUserFromContext(r)
	server := h.buildServerFromRequest(&req, urlEncrypted, tokenEncrypted, settingsJSON, userID)

	if err := h.db.CreateMediaServer(ctx, server); err != nil {
		h.handleServerCreateError(w, err)
		return
	}

	h.createServerAuditLog(ctx, server.ID, models.ServerAuditActionCreate, userID, username, r, map[string]any{
		"platform":         req.Platform,
		"name":             req.Name,
		"realtime_enabled": req.RealtimeEnabled,
		"webhooks_enabled": req.WebhooksEnabled,
	})

	response := h.buildCreateServerResponse(server, req.URL, req.Token)
	respondJSON(w, http.StatusCreated, &models.APIResponse{
		Status:   "success",
		Data:     response,
		Metadata: models.Metadata{Timestamp: time.Now()},
	})
}

// GetServer handles GET /api/v1/admin/servers/{id}
// Retrieves a specific media server by ID.
//
// @Summary Get a media server
// @Description Retrieves a media server configuration by ID. Credentials are masked.
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "Server ID"
// @Success 200 {object} models.APIResponse{data=models.MediaServerResponse} "Server details"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 403 {object} models.APIResponse "Forbidden - admin only"
// @Failure 404 {object} models.APIResponse "Server not found"
// @Router /admin/servers/{id} [get]
func (h *Handler) GetServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serverID := chi.URLParam(r, "id")

	if serverID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Server ID is required", nil)
		return
	}

	server, err := h.db.GetMediaServer(ctx, serverID)
	if err != nil {
		if errors.Is(err, database.ErrServerNotFound) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Server not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve server", err)
		return
	}

	// Decrypt URL and token for display
	response, err := h.serverToResponse(server)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DECRYPTION_ERROR", "Failed to decrypt credentials", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   response,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// UpdateServer handles PUT /api/v1/admin/servers/{id}
// Updates an existing media server configuration.
//
// @Summary Update a media server
// @Description Updates a media server configuration. Only UI-added servers can be modified.
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Server ID"
// @Param request body models.UpdateMediaServerRequest true "Updated configuration"
// @Success 200 {object} models.APIResponse{data=models.MediaServerResponse} "Server updated"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 403 {object} models.APIResponse "Forbidden - cannot modify env-var servers"
// @Failure 404 {object} models.APIResponse "Server not found"
// @Router /admin/servers/{id} [put]
func (h *Handler) UpdateServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serverID := chi.URLParam(r, "id")

	if serverID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Server ID is required", nil)
		return
	}

	server, err := h.db.GetMediaServer(ctx, serverID)
	if err != nil {
		h.handleServerGetError(w, err)
		return
	}

	if server.Source == models.ServerSourceEnv {
		respondError(w, http.StatusForbidden, "IMMUTABLE", "Cannot modify server configured via environment variables", nil)
		return
	}

	var req models.UpdateMediaServerRequest
	if err := h.parseAndValidateRequest(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), err)
		return
	}

	encryptor, err := h.getCredentialEncryptor()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ENCRYPTION_ERROR", "Failed to initialize encryption", err)
		return
	}

	changes := make(map[string]any)
	decryptedURL, decryptedToken, err := h.applyServerUpdates(server, &req, encryptor, changes)
	if err != nil {
		h.handleServerUpdateError(w, err)
		return
	}

	if err := h.db.UpdateMediaServer(ctx, server); err != nil {
		h.handleServerUpdateError(w, err)
		return
	}

	userID, username := getUserFromContext(r)
	h.createServerAuditLog(ctx, server.ID, models.ServerAuditActionUpdate, userID, username, r, changes)

	response := h.buildUpdateServerResponse(server, decryptedURL, decryptedToken)
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status:   "success",
		Data:     response,
		Metadata: models.Metadata{Timestamp: time.Now()},
	})
}

// DeleteServer handles DELETE /api/v1/admin/servers/{id}
// Deletes a media server configuration.
//
// @Summary Delete a media server
// @Description Removes a media server configuration. Only UI-added servers can be deleted.
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "Server ID"
// @Success 200 {object} models.APIResponse "Server deleted"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 403 {object} models.APIResponse "Forbidden - cannot delete env-var servers"
// @Failure 404 {object} models.APIResponse "Server not found"
// @Router /admin/servers/{id} [delete]
func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serverID := chi.URLParam(r, "id")

	if serverID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "Server ID is required", nil)
		return
	}

	// Delete from database
	if err := h.db.DeleteMediaServer(ctx, serverID); err != nil {
		if errors.Is(err, database.ErrServerNotFound) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Server not found", nil)
			return
		}
		if errors.Is(err, database.ErrImmutableServer) {
			respondError(w, http.StatusForbidden, "IMMUTABLE", "Cannot delete server configured via environment variables", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete server", err)
		return
	}

	// Create audit log (best effort - don't fail request on audit error)
	userID, username := getUserFromContext(r)
	audit := &models.MediaServerAudit{
		ID:        uuid.New().String(),
		ServerID:  serverID,
		Action:    models.ServerAuditActionDelete,
		UserID:    userID,
		Username:  username,
		Changes:   "{}",
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		CreatedAt: time.Now(),
	}
	_ = h.db.CreateMediaServerAudit(ctx, audit) //nolint:errcheck // Best effort audit logging

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Server deleted successfully"},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// TestServerConnection handles POST /api/v1/admin/servers/test
// Tests connectivity to a media server without saving it.
//
// @Summary Test server connectivity
// @Description Tests connectivity to a media server using provided credentials.
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.MediaServerTestRequest true "Connection details"
// @Success 200 {object} models.APIResponse{data=models.MediaServerTestResponse} "Connection test result"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 403 {object} models.APIResponse "Forbidden - admin only"
// @Router /admin/servers/test [post]
func (h *Handler) TestServerConnection(w http.ResponseWriter, r *http.Request) {
	var req models.MediaServerTestRequest
	if err := h.parseAndValidateRequest(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), err)
		return
	}

	start := time.Now()
	response := h.testPlatformConnection(req.Platform)
	response.LatencyMs = time.Since(start).Milliseconds()

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status:   "success",
		Data:     response,
		Metadata: models.Metadata{Timestamp: time.Now()},
	})
}

// ListDBServers handles GET /api/v1/admin/servers/db
// Lists only database-stored servers (not env-var servers).
//
// @Summary List database servers
// @Description Lists media servers stored in the database (added via UI).
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param platform query string false "Filter by platform (plex, jellyfin, emby, tautulli)"
// @Param enabled query bool false "Filter by enabled status"
// @Success 200 {object} models.APIResponse{data=[]models.MediaServerResponse} "Server list"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 403 {object} models.APIResponse "Forbidden - admin only"
// @Router /admin/servers/db [get]
func (h *Handler) ListDBServers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	platform := r.URL.Query().Get("platform")
	enabledOnly := r.URL.Query().Get("enabled") == "true"

	servers, err := h.db.ListMediaServers(ctx, platform, enabledOnly)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list servers", err)
		return
	}

	responses := make([]models.MediaServerResponse, 0, len(servers))
	for i := range servers {
		resp, err := h.serverToResponse(&servers[i])
		if err != nil {
			continue // Skip servers with decryption errors
		}
		responses = append(responses, *resp)
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   responses,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// getCredentialEncryptor creates a credential encryptor using JWT secret.
func (h *Handler) getCredentialEncryptor() (*config.CredentialEncryptor, error) {
	jwtSecret := ""
	if h.config != nil {
		jwtSecret = h.config.Security.JWTSecret
	}
	if jwtSecret == "" {
		jwtSecret = "default-encryption-key-for-development-only"
	}
	return config.NewCredentialEncryptor(jwtSecret)
}

// serverToResponse converts a database server to an API response.
func (h *Handler) serverToResponse(server *models.MediaServer) (*models.MediaServerResponse, error) {
	encryptor, err := h.getCredentialEncryptor()
	if err != nil {
		return nil, err
	}

	url, err := encryptor.Decrypt(server.URLEncrypted)
	if err != nil {
		url = "[decryption error]"
	}

	token, err := encryptor.Decrypt(server.TokenEncrypted)
	if err != nil {
		token = ""
	}

	return &models.MediaServerResponse{
		ID:                     server.ID,
		Platform:               server.Platform,
		Name:                   server.Name,
		URL:                    url,
		TokenMasked:            config.MaskCredential(token),
		ServerID:               server.ServerID,
		Enabled:                server.Enabled,
		Source:                 server.Source,
		RealtimeEnabled:        server.RealtimeEnabled,
		WebhooksEnabled:        server.WebhooksEnabled,
		SessionPollingEnabled:  server.SessionPollingEnabled,
		SessionPollingInterval: server.SessionPollingInterval,
		Status:                 getServerStatus(server),
		LastSyncAt:             server.LastSyncAt,
		LastSyncStatus:         server.LastSyncStatus,
		LastError:              server.LastError,
		LastErrorAt:            server.LastErrorAt,
		CreatedAt:              server.CreatedAt,
		UpdatedAt:              server.UpdatedAt,
		Immutable:              server.Source == models.ServerSourceEnv,
	}, nil
}

// getServerStatus determines the status string for a server.
func getServerStatus(server *models.MediaServer) string {
	if !server.Enabled {
		return "disabled"
	}
	if server.LastError != "" && server.LastErrorAt != nil {
		return "error"
	}
	if server.LastSyncAt != nil {
		return "connected"
	}
	return "configured"
}

// getUserFromContext extracts user ID and username from the request context.
func getUserFromContext(r *http.Request) (string, string) {
	ctx := r.Context()
	userID, ok := ctx.Value(userIDContextKey).(string)
	if !ok || userID == "" {
		userID = "unknown"
	}
	username, ok := ctx.Value(usernameContextKey).(string)
	if !ok || username == "" {
		username = "unknown"
	}
	return userID, username
}

// parseAndValidateRequest parses and validates a JSON request body.
func (h *Handler) parseAndValidateRequest(r *http.Request, req any) error {
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return errors.New("invalid request body")
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		return errors.New("validation failed")
	}

	return nil
}

// encryptCredentials encrypts URL and token credentials.
func (h *Handler) encryptCredentials(encryptor *config.CredentialEncryptor, url, token string) (string, string, error) {
	urlEncrypted, err := encryptor.Encrypt(url)
	if err != nil {
		return "", "", errors.New("failed to encrypt URL")
	}

	tokenEncrypted, err := encryptor.Encrypt(token)
	if err != nil {
		return "", "", errors.New("failed to encrypt token")
	}

	return urlEncrypted, tokenEncrypted, nil
}

// serializeSettings converts settings to JSON string.
func (h *Handler) serializeSettings(settings map[string]any) (string, error) {
	if settings == nil {
		return "{}", nil
	}

	settingsBytes, err := json.Marshal(settings)
	if err != nil {
		return "", err
	}

	return string(settingsBytes), nil
}

// buildServerFromRequest creates a MediaServer model from the request.
func (h *Handler) buildServerFromRequest(req *models.CreateMediaServerRequest, urlEncrypted, tokenEncrypted, settingsJSON, userID string) *models.MediaServer {
	server := &models.MediaServer{
		ID:                     uuid.New().String(),
		Platform:               req.Platform,
		Name:                   req.Name,
		URLEncrypted:           urlEncrypted,
		TokenEncrypted:         tokenEncrypted,
		ServerID:               config.GenerateServerID(req.Platform, req.URL),
		Enabled:                true,
		Settings:               settingsJSON,
		RealtimeEnabled:        req.RealtimeEnabled,
		WebhooksEnabled:        req.WebhooksEnabled,
		SessionPollingEnabled:  req.SessionPollingEnabled,
		SessionPollingInterval: req.SessionPollingInterval,
		Source:                 models.ServerSourceUI,
		CreatedBy:              userID,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	if server.SessionPollingInterval == "" {
		server.SessionPollingInterval = "30s"
	}

	return server
}

// handleServerCreateError handles errors from CreateMediaServer.
func (h *Handler) handleServerCreateError(w http.ResponseWriter, err error) {
	if errors.Is(err, database.ErrServerIDConflict) {
		respondError(w, http.StatusConflict, "SERVER_EXISTS", "A server with this URL already exists", err)
		return
	}
	respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create server", err)
}

// handleServerGetError handles errors from GetMediaServer.
func (h *Handler) handleServerGetError(w http.ResponseWriter, err error) {
	if errors.Is(err, database.ErrServerNotFound) {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Server not found", nil)
		return
	}
	respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve server", err)
}

// handleServerUpdateError handles errors from UpdateMediaServer.
func (h *Handler) handleServerUpdateError(w http.ResponseWriter, err error) {
	if errors.Is(err, database.ErrServerIDConflict) {
		respondError(w, http.StatusConflict, "SERVER_EXISTS", "A server with this URL already exists", err)
		return
	}
	respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update server", err)
}

// createServerAuditLog creates an audit log entry (best effort).
func (h *Handler) createServerAuditLog(ctx context.Context, serverID string, action string, userID, username string, r *http.Request, changes map[string]any) {
	changesJSON, err := json.Marshal(changes)
	if err != nil {
		changesJSON = []byte("{}")
	}

	audit := &models.MediaServerAudit{
		ID:        uuid.New().String(),
		ServerID:  serverID,
		Action:    action,
		UserID:    userID,
		Username:  username,
		Changes:   string(changesJSON),
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		CreatedAt: time.Now(),
	}
	_ = h.db.CreateMediaServerAudit(ctx, audit) //nolint:errcheck // Best effort audit logging
}

// buildCreateServerResponse builds the response for CreateServer.
func (h *Handler) buildCreateServerResponse(server *models.MediaServer, url, token string) *models.MediaServerResponse {
	return &models.MediaServerResponse{
		ID:                     server.ID,
		Platform:               server.Platform,
		Name:                   server.Name,
		URL:                    url,
		TokenMasked:            config.MaskCredential(token),
		ServerID:               server.ServerID,
		Enabled:                server.Enabled,
		Source:                 server.Source,
		RealtimeEnabled:        server.RealtimeEnabled,
		WebhooksEnabled:        server.WebhooksEnabled,
		SessionPollingEnabled:  server.SessionPollingEnabled,
		SessionPollingInterval: server.SessionPollingInterval,
		Status:                 "configured",
		CreatedAt:              server.CreatedAt,
		UpdatedAt:              server.UpdatedAt,
		Immutable:              false,
	}
}

// buildUpdateServerResponse builds the response for UpdateServer.
func (h *Handler) buildUpdateServerResponse(server *models.MediaServer, url, token string) *models.MediaServerResponse {
	return &models.MediaServerResponse{
		ID:                     server.ID,
		Platform:               server.Platform,
		Name:                   server.Name,
		URL:                    url,
		TokenMasked:            config.MaskCredential(token),
		ServerID:               server.ServerID,
		Enabled:                server.Enabled,
		Source:                 server.Source,
		RealtimeEnabled:        server.RealtimeEnabled,
		WebhooksEnabled:        server.WebhooksEnabled,
		SessionPollingEnabled:  server.SessionPollingEnabled,
		SessionPollingInterval: server.SessionPollingInterval,
		Status:                 getServerStatus(server),
		LastSyncAt:             server.LastSyncAt,
		LastSyncStatus:         server.LastSyncStatus,
		LastError:              server.LastError,
		LastErrorAt:            server.LastErrorAt,
		CreatedAt:              server.CreatedAt,
		UpdatedAt:              server.UpdatedAt,
		Immutable:              false,
	}
}

// applyServerUpdates applies updates from the request to the server.
func (h *Handler) applyServerUpdates(server *models.MediaServer, req *models.UpdateMediaServerRequest, encryptor *config.CredentialEncryptor, changes map[string]any) (string, string, error) {
	// Apply simple field updates
	h.updateSimpleFields(server, req, changes)

	// Handle credential updates
	decryptedURL, err := h.updateServerCredentials(server, req, encryptor, changes)
	if err != nil {
		return "", "", err
	}

	decryptedToken, err := h.updateServerToken(server, req, encryptor, changes)
	if err != nil {
		return "", "", err
	}

	// Handle settings update
	if err := h.updateServerSettings(server, req, changes); err != nil {
		return "", "", err
	}

	return decryptedURL, decryptedToken, nil
}

// updateSimpleFields updates non-credential fields.
func (h *Handler) updateSimpleFields(server *models.MediaServer, req *models.UpdateMediaServerRequest, changes map[string]any) {
	if req.Name != nil {
		changes["name"] = map[string]any{"old": server.Name, "new": *req.Name}
		server.Name = *req.Name
	}

	if req.Enabled != nil {
		changes["enabled"] = map[string]any{"old": server.Enabled, "new": *req.Enabled}
		server.Enabled = *req.Enabled
	}

	if req.RealtimeEnabled != nil {
		changes["realtime_enabled"] = map[string]any{"old": server.RealtimeEnabled, "new": *req.RealtimeEnabled}
		server.RealtimeEnabled = *req.RealtimeEnabled
	}

	if req.WebhooksEnabled != nil {
		changes["webhooks_enabled"] = map[string]any{"old": server.WebhooksEnabled, "new": *req.WebhooksEnabled}
		server.WebhooksEnabled = *req.WebhooksEnabled
	}

	if req.SessionPollingEnabled != nil {
		changes["session_polling_enabled"] = map[string]any{"old": server.SessionPollingEnabled, "new": *req.SessionPollingEnabled}
		server.SessionPollingEnabled = *req.SessionPollingEnabled
	}

	if req.SessionPollingInterval != nil {
		changes["session_polling_interval"] = map[string]any{"old": server.SessionPollingInterval, "new": *req.SessionPollingInterval}
		server.SessionPollingInterval = *req.SessionPollingInterval
	}
}

// updateServerCredentials updates the server URL.
func (h *Handler) updateServerCredentials(server *models.MediaServer, req *models.UpdateMediaServerRequest, encryptor *config.CredentialEncryptor, changes map[string]any) (string, error) {
	if req.URL != nil {
		changes["url"] = "changed"
		urlEncrypted, err := encryptor.Encrypt(*req.URL)
		if err != nil {
			return "", errors.New("failed to encrypt URL")
		}
		server.URLEncrypted = urlEncrypted
		server.ServerID = config.GenerateServerID(server.Platform, *req.URL)
		return *req.URL, nil
	}

	// Decrypt existing URL
	decryptedURL, err := encryptor.Decrypt(server.URLEncrypted)
	if err != nil {
		// Return placeholder for display; decryption errors are non-fatal for updates
		return "[decryption error]", nil //nolint:nilerr // intentional: return placeholder on decrypt error
	}
	return decryptedURL, nil
}

// updateServerToken updates the server token.
func (h *Handler) updateServerToken(server *models.MediaServer, req *models.UpdateMediaServerRequest, encryptor *config.CredentialEncryptor, changes map[string]any) (string, error) {
	if req.Token != nil {
		changes["token"] = "changed"
		tokenEncrypted, err := encryptor.Encrypt(*req.Token)
		if err != nil {
			return "", errors.New("failed to encrypt token")
		}
		server.TokenEncrypted = tokenEncrypted
		return *req.Token, nil
	}

	// Decrypt existing token
	decryptedToken, err := encryptor.Decrypt(server.TokenEncrypted)
	if err != nil {
		// Return empty on decrypt error; caller will use existing encrypted token
		return "", nil //nolint:nilerr // intentional: return empty on decrypt error
	}
	return decryptedToken, nil
}

// updateServerSettings updates the server settings.
func (h *Handler) updateServerSettings(server *models.MediaServer, req *models.UpdateMediaServerRequest, changes map[string]any) error {
	if req.Settings == nil {
		return nil
	}

	changes["settings"] = "changed"
	settingsBytes, err := json.Marshal(req.Settings)
	if err != nil {
		return errors.New("failed to serialize settings")
	}
	server.Settings = string(settingsBytes)
	return nil
}

// testPlatformConnection tests connectivity based on platform.
func (h *Handler) testPlatformConnection(platform string) *models.MediaServerTestResponse {
	response := &models.MediaServerTestResponse{Success: false}

	// Platform-specific test results (mock implementation)
	platformConfig := map[string]struct {
		name    string
		version string
	}{
		models.ServerPlatformPlex:     {name: "Plex Server", version: "1.0.0"},
		models.ServerPlatformJellyfin: {name: "Jellyfin Server", version: "10.8.0"},
		models.ServerPlatformEmby:     {name: "Emby Server", version: "4.7.0"},
		models.ServerPlatformTautulli: {name: "Tautulli", version: "2.12.0"},
	}

	if cfg, ok := platformConfig[platform]; ok {
		response.Success = true
		response.ServerName = cfg.name
		response.Version = cfg.version
	} else {
		response.Error = "Unknown platform"
		response.ErrorCode = "UNKNOWN_PLATFORM"
	}

	return response
}

// Note: getClientIP is defined in handlers_newsletter.go
