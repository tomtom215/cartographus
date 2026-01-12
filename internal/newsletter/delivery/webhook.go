// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// WebhookChannel implements generic HTTP webhook delivery.
type WebhookChannel struct {
	client *http.Client
}

// NewWebhookChannel creates a new generic webhook delivery channel.
func NewWebhookChannel() *WebhookChannel {
	return &WebhookChannel{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the channel identifier.
func (c *WebhookChannel) Name() models.DeliveryChannel {
	return models.DeliveryChannelWebhook
}

// SupportsHTML returns true as webhooks can receive any content type.
func (c *WebhookChannel) SupportsHTML() bool {
	return true
}

// MaxContentLength returns 0 as webhooks have no standard limit.
func (c *WebhookChannel) MaxContentLength() int {
	return 0 // No limit
}

// Validate checks if the webhook configuration is valid.
func (c *WebhookChannel) Validate(config *models.ChannelConfig) error {
	if config == nil {
		return fmt.Errorf("webhook configuration is required")
	}
	if config.WebhookURL == "" {
		return fmt.Errorf("webhook URL is required")
	}
	if err := ValidateWebhookURL(config.WebhookURL); err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	// Validate method if specified
	method := strings.ToUpper(config.WebhookMethod)
	if method != "" && method != "POST" && method != "PUT" && method != "PATCH" {
		return fmt.Errorf("webhook method must be POST, PUT, or PATCH")
	}
	return nil
}

// WebhookPayload represents the generic webhook payload structure.
type WebhookPayload struct {
	// Event metadata
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`

	// Newsletter metadata
	Newsletter WebhookNewsletterMeta `json:"newsletter"`

	// Content
	Subject  string `json:"subject"`
	BodyHTML string `json:"body_html,omitempty"`
	BodyText string `json:"body_text,omitempty"`

	// Recipient information
	Recipient WebhookRecipient `json:"recipient"`
}

// WebhookNewsletterMeta contains newsletter metadata for the webhook.
type WebhookNewsletterMeta struct {
	DeliveryID   string `json:"delivery_id,omitempty"`
	ScheduleID   string `json:"schedule_id,omitempty"`
	TemplateID   string `json:"template_id,omitempty"`
	TemplateName string `json:"template_name,omitempty"`
	Type         string `json:"type,omitempty"`
	ServerName   string `json:"server_name,omitempty"`
}

// WebhookRecipient contains recipient information for the webhook.
type WebhookRecipient struct {
	Type   string `json:"type"`
	Target string `json:"target"`
	Name   string `json:"name,omitempty"`
}

// Send delivers the newsletter via generic HTTP webhook.
//
//nolint:gocyclo // Complexity from HTTP request building, retries, and response handling
func (c *WebhookChannel) Send(ctx context.Context, params *SendParams) (*DeliveryResult, error) {
	result := &DeliveryResult{
		Recipient:     params.Recipient.Target,
		RecipientType: params.Recipient.Type,
	}

	// Validate config
	if err := c.Validate(params.Config); err != nil {
		result.ErrorMessage = err.Error()
		result.ErrorCode = ErrorCodeInvalidConfig
		return result, nil
	}

	// Build webhook payload
	payload := c.buildPayload(params)

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to marshal payload: %v", err)
		result.ErrorCode = ErrorCodeUnknown
		return result, nil
	}

	// Determine HTTP method
	method := strings.ToUpper(params.Config.WebhookMethod)
	if method == "" {
		method = http.MethodPost
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, params.Config.WebhookURL, bytes.NewReader(jsonPayload))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		result.ErrorCode = ErrorCodeUnknown
		return result, nil
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Cartographus-Newsletter/1.0")

	// Add custom headers
	for key, value := range params.Config.WebhookHeaders {
		req.Header.Set(key, value)
	}

	// Add authentication if configured
	if params.Config.WebhookAuth != "" {
		req.Header.Set("Authorization", params.Config.WebhookAuth)
	}

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to send webhook: %v", err)
		result.ErrorCode = classifyHTTPError(err)
		result.IsTransient = isTransientHTTPError(result.ErrorCode)
		return result, nil
	}
	defer resp.Body.Close()

	result.ResponseCode = resp.StatusCode

	// Read response body for error details
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		body = []byte("(failed to read response)")
	}

	// Check for success (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		now := time.Now()
		result.Success = true
		result.DeliveredAt = &now

		// Try to extract external ID from response
		var respData map[string]interface{}
		if err := json.Unmarshal(body, &respData); err == nil {
			if id, ok := respData["id"].(string); ok {
				result.ExternalID = id
			} else if id, ok := respData["message_id"].(string); ok {
				result.ExternalID = id
			}
		}
		return result, nil
	}

	// Handle error
	result.ErrorMessage = fmt.Sprintf("webhook returned %d: %s", resp.StatusCode, string(body))
	result.ErrorCode = classifyHTTPStatusCode(resp.StatusCode)
	result.IsTransient = isTransientHTTPError(result.ErrorCode)

	// Check for rate limiting
	if resp.StatusCode == 429 {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
				result.RetryAfter = &seconds
			}
		}
	}

	return result, nil
}

// buildPayload constructs the webhook payload.
func (c *WebhookChannel) buildPayload(params *SendParams) WebhookPayload {
	payload := WebhookPayload{
		Event:     "newsletter.delivery",
		Timestamp: time.Now().UTC(),
		Subject:   params.Subject,
		BodyHTML:  params.BodyHTML,
		BodyText:  params.BodyText,
		Recipient: WebhookRecipient{
			Type:   params.Recipient.Type,
			Target: params.Recipient.Target,
			Name:   params.Recipient.Name,
		},
	}

	if params.Metadata != nil {
		payload.Newsletter = WebhookNewsletterMeta{
			DeliveryID:   params.Metadata.DeliveryID,
			ScheduleID:   params.Metadata.ScheduleID,
			TemplateID:   params.Metadata.TemplateID,
			TemplateName: params.Metadata.TemplateName,
			Type:         string(params.Metadata.NewsletterType),
			ServerName:   params.Metadata.ServerName,
		}
	}

	return payload
}
