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

// DiscordChannel implements Discord webhook delivery.
type DiscordChannel struct {
	client *http.Client
}

// NewDiscordChannel creates a new Discord delivery channel.
func NewDiscordChannel() *DiscordChannel {
	return &DiscordChannel{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the channel identifier.
func (c *DiscordChannel) Name() models.DeliveryChannel {
	return models.DeliveryChannelDiscord
}

// SupportsHTML returns false as Discord uses its own markdown format.
func (c *DiscordChannel) SupportsHTML() bool {
	return false
}

// MaxContentLength returns Discord's embed description limit.
func (c *DiscordChannel) MaxContentLength() int {
	return 4096 // Discord embed description limit
}

// Validate checks if the Discord webhook configuration is valid.
func (c *DiscordChannel) Validate(config *models.ChannelConfig) error {
	if config == nil {
		return fmt.Errorf("Discord configuration is required")
	}
	if config.DiscordWebhookURL == "" {
		return fmt.Errorf("Discord webhook URL is required")
	}
	if err := ValidateWebhookURL(config.DiscordWebhookURL); err != nil {
		return fmt.Errorf("invalid Discord webhook URL: %w", err)
	}
	// Validate it looks like a Discord webhook
	if !strings.Contains(config.DiscordWebhookURL, "discord.com/api/webhooks/") &&
		!strings.Contains(config.DiscordWebhookURL, "discordapp.com/api/webhooks/") {
		return fmt.Errorf("URL does not appear to be a Discord webhook URL")
	}
	return nil
}

// DiscordWebhookPayload represents the Discord webhook message structure.
type DiscordWebhookPayload struct {
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Content   string         `json:"content,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed object.
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Author      *DiscordEmbedAuthor `json:"author,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
}

// DiscordEmbedFooter represents the footer of a Discord embed.
type DiscordEmbedFooter struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedAuthor represents the author of a Discord embed.
type DiscordEmbedAuthor struct {
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedField represents a field in a Discord embed.
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// Send delivers the newsletter via Discord webhook.
func (c *DiscordChannel) Send(ctx context.Context, params *SendParams) (*DeliveryResult, error) {
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

	// Build Discord payload
	payload := c.buildPayload(params)

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to marshal payload: %v", err)
		result.ErrorCode = ErrorCodeUnknown
		return result, nil
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.Config.DiscordWebhookURL, bytes.NewReader(jsonPayload))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		result.ErrorCode = ErrorCodeUnknown
		return result, nil
	}
	req.Header.Set("Content-Type", "application/json")

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

	// Check response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		now := time.Now()
		result.Success = true
		result.DeliveredAt = &now
		return result, nil
	}

	// Read error response
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		body = []byte("(failed to read response)")
	}
	result.ErrorMessage = fmt.Sprintf("Discord webhook returned %d: %s", resp.StatusCode, string(body))
	result.ErrorCode = classifyHTTPStatusCode(resp.StatusCode)
	result.IsTransient = isTransientHTTPError(result.ErrorCode)

	// Check for rate limiting
	if resp.StatusCode == 429 {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Discord returns retry-after in seconds
			if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
				result.RetryAfter = &seconds
			}
		}
	}

	return result, nil
}

// buildPayload constructs the Discord webhook payload.
func (c *DiscordChannel) buildPayload(params *SendParams) DiscordWebhookPayload {
	payload := DiscordWebhookPayload{
		Username:  params.Config.DiscordUsername,
		AvatarURL: params.Config.DiscordAvatarURL,
	}

	if payload.Username == "" && params.Metadata != nil {
		payload.Username = params.Metadata.ServerName
	}
	if payload.Username == "" {
		payload.Username = "Newsletter"
	}

	// Get content - prefer plaintext for Discord
	content := params.BodyText
	if content == "" && params.BodyHTML != "" {
		content = HTMLToPlaintext(params.BodyHTML)
	}

	// Truncate if needed
	content = TruncateContent(content, c.MaxContentLength()-200) // Leave room for embed chrome

	// Create embed
	embed := DiscordEmbed{
		Title:       params.Subject,
		Description: content,
		Color:       0x5865F2, // Discord blurple
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	// Add footer
	if params.Metadata != nil && params.Metadata.ServerName != "" {
		embed.Footer = &DiscordEmbedFooter{
			Text: fmt.Sprintf("From %s", params.Metadata.ServerName),
		}
	}

	payload.Embeds = []DiscordEmbed{embed}
	return payload
}

// =============================================================================
// HTTP Error Helpers (shared across webhook-based channels)
// =============================================================================

// classifyHTTPError classifies an HTTP error into an error code.
func classifyHTTPError(err error) string {
	errStr := err.Error()

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return ErrorCodeTimeout
	}
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "refused") {
		return ErrorCodeConnectionFailed
	}

	return ErrorCodeUnknown
}

// classifyHTTPStatusCode classifies an HTTP status code into an error code.
func classifyHTTPStatusCode(code int) string {
	switch {
	case code == 401 || code == 403:
		return ErrorCodeAuthFailed
	case code == 404:
		return ErrorCodeRecipientNotFound
	case code == 429:
		return ErrorCodeRateLimited
	case code == 413:
		return ErrorCodeContentTooLarge
	case code >= 500:
		return ErrorCodeServerError
	default:
		return ErrorCodeUnknown
	}
}

// isTransientHTTPError returns true if the error is transient and can be retried.
func isTransientHTTPError(code string) bool {
	switch code {
	case ErrorCodeConnectionFailed, ErrorCodeTimeout, ErrorCodeRateLimited, ErrorCodeServerError:
		return true
	default:
		return false
	}
}
