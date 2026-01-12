// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// EventPublisher defines the interface for publishing playback events to NATS.
// This mirrors sync.EventPublisher to allow the same publisher instance to be
// used by both sync.Manager and api.Handler.
type EventPublisher interface {
	// PublishPlaybackEvent publishes a playback event to the event bus.
	// Returns nil if publishing is not enabled or succeeds.
	// Errors should be logged but not block webhook processing.
	PublishPlaybackEvent(ctx context.Context, event *models.PlaybackEvent) error
}

// SetEventPublisher sets the optional event publisher for NATS integration.
// When set, webhook media events will be published after successful processing.
// The publisher is optional - passing nil disables event publishing.
//
// Thread Safety: Safe for concurrent access but should be called once during startup.
func (h *Handler) SetEventPublisher(publisher EventPublisher) {
	h.eventPublisher = publisher
}

// publishWebhookEvent publishes a webhook event to NATS if a publisher is configured.
// Only media events (media.play, media.stop, etc.) are published since they contain
// meaningful playback data. Admin and library events are skipped.
//
// Errors are logged but don't block webhook processing since NATS is optional.
// Publishing is done asynchronously to avoid blocking the webhook response.
func (h *Handler) publishWebhookEvent(ctx context.Context, webhook *models.PlexWebhook) {
	if h.eventPublisher == nil {
		return
	}

	// Only publish media events that have metadata
	if webhook.Metadata == nil {
		return
	}

	// Only publish media playback events
	if !webhook.IsMediaEvent() {
		return
	}

	// Convert webhook to PlaybackEvent
	event := h.webhookToPlaybackEvent(webhook)

	// Publish asynchronously to avoid blocking webhook response
	go func() {
		if err := h.eventPublisher.PublishPlaybackEvent(ctx, event); err != nil {
			logging.Warn().Err(err).Msg("Failed to publish webhook event to NATS")
		}
	}()
}

// webhookToPlaybackEvent converts a PlexWebhook to a PlaybackEvent for NATS publishing.
// This creates a minimal event suitable for real-time streaming to WebSocket clients.
//
// IMPORTANT: Plex webhooks don't include a session key like WebSocket notifications do.
// We construct a pseudo-session key using the Player UUID for basic deduplication.
// Database-aware deduplication in DuckDBConsumer handles collision with Tautulli events.
func (h *Handler) webhookToPlaybackEvent(webhook *models.PlexWebhook) *models.PlaybackEvent {
	// Construct a pseudo-session key for webhook deduplication.
	// Format: "webhook:{player_uuid}:{rating_key}" - enables dedup of duplicate webhook deliveries.
	// This won't match Tautulli session keys, but database-aware deduplication handles that.
	sessionKey := "webhook:" + webhook.Player.UUID
	if webhook.Metadata != nil && webhook.Metadata.RatingKey != "" {
		sessionKey += ":" + webhook.Metadata.RatingKey
	}

	event := &models.PlaybackEvent{
		ID:         uuid.New(),
		Source:     "plex",
		SessionKey: sessionKey, // Pseudo-session key for webhook deduplication
		UserID:     webhook.Account.ID,
		Username:   webhook.Account.Title,
		StartedAt:  time.Now(),
		IPAddress:  webhook.Player.PublicAddress,
		Player:     webhook.Player.Title,
	}

	// CRITICAL (v1.47): Set MachineID for cross-source deduplication
	// The Player.UUID uniquely identifies the device, which is essential for
	// generating accurate CorrelationKeys in multi-device shared account scenarios.
	// Without this, events from different devices would have empty MachineID,
	// resulting in false positive deduplication.
	if webhook.Player.UUID != "" {
		event.MachineID = &webhook.Player.UUID
	}

	// Set location type based on local flag
	if webhook.Player.Local {
		event.LocationType = "lan"
	} else {
		event.LocationType = "wan"
	}

	// Media information from metadata
	if webhook.Metadata != nil {
		event.MediaType = webhook.Metadata.Type
		event.Title = webhook.Metadata.Title

		// Parent/Grandparent titles for TV episodes
		if webhook.Metadata.ParentTitle != "" {
			event.ParentTitle = &webhook.Metadata.ParentTitle
		}
		if webhook.Metadata.GrandparentTitle != "" {
			event.GrandparentTitle = &webhook.Metadata.GrandparentTitle
		}

		// Rating key for content identification
		if webhook.Metadata.RatingKey != "" {
			event.RatingKey = &webhook.Metadata.RatingKey
		}
	}

	// Platform from player device info
	event.Platform = webhook.Player.Title

	return event
}
