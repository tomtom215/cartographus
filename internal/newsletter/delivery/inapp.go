// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package delivery

import (
	"context"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// InAppChannel implements in-app notification delivery.
// This channel stores notifications in the database for retrieval via the API.
type InAppChannel struct {
	store InAppNotificationStore
}

// InAppNotificationStore defines the interface for storing in-app notifications.
type InAppNotificationStore interface {
	// CreateNotification creates a new in-app notification.
	CreateNotification(ctx context.Context, notification *InAppNotification) error
}

// InAppNotification represents an in-app notification.
type InAppNotification struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	Type       string                 `json:"type"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Read       bool                   `json:"read"`
	ReadAt     *time.Time             `json:"read_at,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
	DeliveryID string                 `json:"delivery_id,omitempty"`
	ScheduleID string                 `json:"schedule_id,omitempty"`
	TemplateID string                 `json:"template_id,omitempty"`
}

// NewInAppChannel creates a new in-app notification delivery channel.
func NewInAppChannel() *InAppChannel {
	return &InAppChannel{}
}

// NewInAppChannelWithStore creates a new in-app notification channel with a store.
func NewInAppChannelWithStore(store InAppNotificationStore) *InAppChannel {
	return &InAppChannel{
		store: store,
	}
}

// SetStore sets the notification store (for dependency injection).
func (c *InAppChannel) SetStore(store InAppNotificationStore) {
	c.store = store
}

// Name returns the channel identifier.
func (c *InAppChannel) Name() models.DeliveryChannel {
	return models.DeliveryChannelInApp
}

// SupportsHTML returns false as in-app notifications use plain text.
func (c *InAppChannel) SupportsHTML() bool {
	return false
}

// MaxContentLength returns the maximum notification message length.
func (c *InAppChannel) MaxContentLength() int {
	return 1000 // Reasonable limit for in-app notifications
}

// Validate checks if the in-app configuration is valid.
// In-app notifications require a user recipient type.
func (c *InAppChannel) Validate(config *models.ChannelConfig) error {
	// In-app channel doesn't require special configuration
	// Validation is done on the recipient instead
	return nil
}

// Send delivers the newsletter as an in-app notification.
func (c *InAppChannel) Send(ctx context.Context, params *SendParams) (*DeliveryResult, error) {
	result := &DeliveryResult{
		Recipient:     params.Recipient.Target,
		RecipientType: params.Recipient.Type,
	}

	// Validate recipient type - must be a user
	if params.Recipient.Type != "user" {
		result.ErrorMessage = "in-app notifications require a user recipient"
		result.ErrorCode = ErrorCodeInvalidRecipient
		return result, nil
	}

	// Validate store is configured
	if c.store == nil {
		result.ErrorMessage = "in-app notification store not configured"
		result.ErrorCode = ErrorCodeInvalidConfig
		return result, nil
	}

	// Get content - prefer plaintext
	content := params.BodyText
	if content == "" && params.BodyHTML != "" {
		content = HTMLToPlaintext(params.BodyHTML)
	}

	// Truncate if needed
	content = TruncateContent(content, c.MaxContentLength())

	// Build notification
	notification := &InAppNotification{
		UserID:    params.Recipient.Target,
		Type:      "newsletter",
		Title:     params.Subject,
		Message:   content,
		Read:      false,
		CreatedAt: time.Now(),
	}

	// Add metadata
	if params.Metadata != nil {
		notification.DeliveryID = params.Metadata.DeliveryID
		notification.ScheduleID = params.Metadata.ScheduleID
		notification.TemplateID = params.Metadata.TemplateID
		notification.Data = map[string]interface{}{
			"newsletter_type": string(params.Metadata.NewsletterType),
			"template_name":   params.Metadata.TemplateName,
			"server_name":     params.Metadata.ServerName,
		}
	}

	// Store notification
	if err := c.store.CreateNotification(ctx, notification); err != nil {
		result.ErrorMessage = err.Error()
		result.ErrorCode = ErrorCodeServerError
		result.IsTransient = true
		return result, nil //nolint:nilerr // Error is captured in result struct, not returned
	}

	now := time.Now()
	result.Success = true
	result.DeliveredAt = &now
	result.ExternalID = notification.ID
	return result, nil
}
