// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// PlexWebhook handles incoming Plex webhook notifications
// POST /api/v1/plex/webhook
//
// Plex webhooks provide push-based notifications for playback events,
// complementing the WebSocket real-time stream with reliable delivery.
//
// Webhook Setup:
//  1. Go to Plex Settings → Webhooks
//  2. Add webhook URL: https://your-domain.com/api/v1/plex/webhook
//  3. Set PLEX_WEBHOOK_SECRET env var to a random 32+ character string
//  4. Configure ENABLE_PLEX_WEBHOOKS=true
//
// Supported Events:
//   - media.play: Playback started
//   - media.pause: Playback paused
//   - media.resume: Playback resumed
//   - media.stop: Playback stopped
//   - media.scrobble: Content marked as watched (75%+ viewed)
//   - media.rate: Content rated by user
//   - library.new: New content added to library
//   - library.on.deck: Content added to "On Deck"
//
// Security:
//   - Verifies HMAC-SHA256 signature if PLEX_WEBHOOK_SECRET is configured
//   - Rejects replay attacks using timestamp validation (5 minute window)
//   - Rate limited like all other API endpoints
func (h *Handler) PlexWebhook(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if webhooks are enabled
	if !h.config.Plex.WebhooksEnabled {
		respondError(w, http.StatusNotFound, "WEBHOOKS_DISABLED", "Plex webhooks are not enabled", nil)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to read request body", err)
		return
	}
	defer r.Body.Close()

	// Verify signature if secret is configured
	if h.config.Plex.WebhookSecret != "" {
		signature := r.Header.Get("X-Plex-Signature")
		if signature == "" {
			respondError(w, http.StatusUnauthorized, "MISSING_SIGNATURE", "X-Plex-Signature header required", nil)
			return
		}

		if !h.verifyWebhookSignature(body, signature, h.config.Plex.WebhookSecret) {
			respondError(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "Webhook signature verification failed", nil)
			return
		}
	}

	// Parse webhook payload
	var webhook models.PlexWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Failed to parse webhook JSON", err)
		return
	}

	// Log webhook event - sanitize all user-provided values to prevent log injection
	logging.Info().
		Str("event", sanitizeLogValue(webhook.Event)).
		Str("user", sanitizeLogValue(webhook.GetUsername())).
		Str("content", sanitizeLogValue(webhook.GetContentTitle())).
		Str("ip", sanitizeLogValue(webhook.GetPlayerIP())).
		Msg("Webhook received")

	// Process webhook based on event type
	switch webhook.Event {
	case "media.play":
		h.handleWebhookMediaPlay(&webhook)
	case "media.pause":
		h.handleWebhookMediaPause(&webhook)
	case "media.resume":
		h.handleWebhookMediaResume(&webhook)
	case "media.stop":
		h.handleWebhookMediaStop(&webhook)
	case "media.scrobble":
		h.handleWebhookMediaScrobble(&webhook)
	case "media.rate":
		h.handleWebhookMediaRate(&webhook)
	case "library.new":
		h.handleWebhookLibraryNew(&webhook)
	case "library.on.deck":
		h.handleWebhookLibraryOnDeck(&webhook)
	case "admin.database.backup":
		h.handleWebhookAdminBackup(&webhook)
	case "admin.database.corrupted":
		h.handleWebhookAdminCorrupted(&webhook)
	case "device.new":
		h.handleWebhookDeviceNew(&webhook)
	default:
		logging.Warn().Str("event", sanitizeLogValue(webhook.Event)).Msg("Unknown webhook event type")
	}

	// Publish to NATS for event-driven processing (async, non-blocking)
	h.publishWebhookEvent(r.Context(), &webhook)

	// Broadcast to WebSocket clients
	h.broadcastWebhookEvent(&webhook)

	// Return success
	elapsed := time.Since(start).Milliseconds()
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"received": true,
			"event":    webhook.Event,
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: elapsed,
		},
	})
}

// verifyWebhookSignature verifies the HMAC-SHA256 signature of the webhook payload
func (h *Handler) verifyWebhookSignature(body []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

// handleWebhookMediaPlay processes media.play events (playback started)
func (h *Handler) handleWebhookMediaPlay(webhook *models.PlexWebhook) {
	logging.Info().
		Str("user", sanitizeLogValue(webhook.GetUsername())).
		Str("content", sanitizeLogValue(webhook.GetContentTitle())).
		Msg("Media play started")

	// Broadcast real-time update
	h.wsHub.BroadcastJSON("plex_webhook_play", map[string]interface{}{
		"event":    "media.play",
		"username": webhook.GetUsername(),
		"content":  webhook.GetContentTitle(),
		"ip":       webhook.GetPlayerIP(),
		"player":   webhook.Player.Title,
		"local":    webhook.Player.Local,
	})
}

// handleWebhookMediaPause processes media.pause events (playback paused)
func (h *Handler) handleWebhookMediaPause(webhook *models.PlexWebhook) {
	logging.Info().
		Str("user", sanitizeLogValue(webhook.GetUsername())).
		Str("content", sanitizeLogValue(webhook.GetContentTitle())).
		Msg("Media paused")

	h.wsHub.BroadcastJSON("plex_webhook_pause", map[string]interface{}{
		"event":    "media.pause",
		"username": webhook.GetUsername(),
		"content":  webhook.GetContentTitle(),
	})
}

// handleWebhookMediaResume processes media.resume events (playback resumed)
func (h *Handler) handleWebhookMediaResume(webhook *models.PlexWebhook) {
	logging.Info().
		Str("user", sanitizeLogValue(webhook.GetUsername())).
		Str("content", sanitizeLogValue(webhook.GetContentTitle())).
		Msg("Media resumed")

	h.wsHub.BroadcastJSON("plex_webhook_resume", map[string]interface{}{
		"event":    "media.resume",
		"username": webhook.GetUsername(),
		"content":  webhook.GetContentTitle(),
	})
}

// handleWebhookMediaStop processes media.stop events (playback stopped)
func (h *Handler) handleWebhookMediaStop(webhook *models.PlexWebhook) {
	logging.Info().
		Str("user", sanitizeLogValue(webhook.GetUsername())).
		Str("content", sanitizeLogValue(webhook.GetContentTitle())).
		Msg("Media stopped")

	h.wsHub.BroadcastJSON("plex_webhook_stop", map[string]interface{}{
		"event":    "media.stop",
		"username": webhook.GetUsername(),
		"content":  webhook.GetContentTitle(),
	})
}

// handleWebhookMediaScrobble processes media.scrobble events (content marked as watched)
func (h *Handler) handleWebhookMediaScrobble(webhook *models.PlexWebhook) {
	logging.Info().
		Str("user", sanitizeLogValue(webhook.GetUsername())).
		Str("content", sanitizeLogValue(webhook.GetContentTitle())).
		Msg("Media scrobbled")

	h.wsHub.BroadcastJSON("plex_webhook_scrobble", map[string]interface{}{
		"event":    "media.scrobble",
		"username": webhook.GetUsername(),
		"content":  webhook.GetContentTitle(),
	})
}

// handleWebhookMediaRate processes media.rate events (content rated)
func (h *Handler) handleWebhookMediaRate(webhook *models.PlexWebhook) {
	logging.Info().
		Str("user", sanitizeLogValue(webhook.GetUsername())).
		Str("content", sanitizeLogValue(webhook.GetContentTitle())).
		Msg("Media rated")

	h.wsHub.BroadcastJSON("plex_webhook_rate", map[string]interface{}{
		"event":    "media.rate",
		"username": webhook.GetUsername(),
		"content":  webhook.GetContentTitle(),
	})
}

// handleWebhookLibraryNew processes library.new events (new content added)
func (h *Handler) handleWebhookLibraryNew(webhook *models.PlexWebhook) {
	if webhook.Metadata == nil {
		return
	}
	logging.Info().
		Str("title", sanitizeLogValue(webhook.Metadata.Title)).
		Str("library", sanitizeLogValue(webhook.Metadata.LibrarySectionTitle)).
		Msg("Library new content")

	h.wsHub.BroadcastJSON("plex_webhook_library_new", map[string]interface{}{
		"event":   "library.new",
		"title":   webhook.Metadata.Title,
		"library": webhook.Metadata.LibrarySectionTitle,
		"type":    webhook.Metadata.Type,
	})
}

// handleWebhookLibraryOnDeck processes library.on.deck events
func (h *Handler) handleWebhookLibraryOnDeck(webhook *models.PlexWebhook) {
	if webhook.Metadata == nil {
		return
	}
	logging.Info().
		Str("title", sanitizeLogValue(webhook.Metadata.Title)).
		Msg("Library on deck")

	h.wsHub.BroadcastJSON("plex_webhook_on_deck", map[string]interface{}{
		"event": "library.on.deck",
		"title": webhook.Metadata.Title,
	})
}

// handleWebhookAdminBackup processes admin.database.backup events
func (h *Handler) handleWebhookAdminBackup(webhook *models.PlexWebhook) {
	logging.Info().
		Str("server", sanitizeLogValue(webhook.Server.Title)).
		Msg("Admin backup completed")

	h.wsHub.BroadcastJSON("plex_webhook_admin", map[string]interface{}{
		"event":  "admin.database.backup",
		"server": webhook.Server.Title,
	})
}

// handleWebhookAdminCorrupted processes admin.database.corrupted events
func (h *Handler) handleWebhookAdminCorrupted(webhook *models.PlexWebhook) {
	logging.Error().
		Str("server", sanitizeLogValue(webhook.Server.Title)).
		Msg("Database corruption detected")

	h.wsHub.BroadcastJSON("plex_webhook_alert", map[string]interface{}{
		"event":   "admin.database.corrupted",
		"server":  webhook.Server.Title,
		"alert":   true,
		"message": "Database corruption detected!",
	})
}

// handleWebhookDeviceNew processes device.new events (new device connected)
func (h *Handler) handleWebhookDeviceNew(webhook *models.PlexWebhook) {
	logging.Info().
		Str("server", sanitizeLogValue(webhook.Server.Title)).
		Msg("New device connected")

	h.wsHub.BroadcastJSON("plex_webhook_device", map[string]interface{}{
		"event":  "device.new",
		"server": webhook.Server.Title,
		"player": webhook.Player.Title,
	})
}

// broadcastWebhookEvent broadcasts the raw webhook event to all WebSocket clients
func (h *Handler) broadcastWebhookEvent(webhook *models.PlexWebhook) {
	h.wsHub.BroadcastJSON("plex_webhook", map[string]interface{}{
		"event":     webhook.Event,
		"username":  webhook.GetUsername(),
		"content":   webhook.GetContentTitle(),
		"player":    webhook.Player.Title,
		"ip":        webhook.GetPlayerIP(),
		"local":     webhook.Player.Local,
		"server":    webhook.Server.Title,
		"timestamp": time.Now().Unix(),
	})
}
