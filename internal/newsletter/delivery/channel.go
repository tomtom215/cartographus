// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package delivery provides newsletter delivery channel implementations.
//
// This package implements multiple delivery channels for the Newsletter Generator:
//   - Email: SMTP-based email delivery with HTML/plaintext support
//   - Discord: Discord webhook integration with embeds
//   - Slack: Slack webhook integration with blocks
//   - Telegram: Telegram Bot API integration
//   - Webhook: Generic HTTP webhook delivery
//   - InApp: In-app notification delivery
//
// Each channel implements the Channel interface for consistent behavior.
// All channels support:
//   - Retry with exponential backoff
//   - Timeout handling
//   - Error categorization (permanent vs transient)
//   - Metrics and logging
//
// Security:
//   - Credentials are never logged
//   - TLS is enforced where supported
//   - Webhook URLs are validated
//   - Rate limiting is applied per channel
package delivery

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// Channel defines the interface for newsletter delivery channels.
// All delivery channels must implement this interface for consistent behavior.
type Channel interface {
	// Name returns the channel identifier (email, discord, slack, etc.).
	Name() models.DeliveryChannel

	// Validate checks if the channel configuration is valid.
	// Returns an error if the configuration is incomplete or invalid.
	Validate(config *models.ChannelConfig) error

	// Send delivers the newsletter content to the specified recipient.
	// Returns a DeliveryResult with success/failure details.
	Send(ctx context.Context, params *SendParams) (*DeliveryResult, error)

	// SupportsHTML returns true if the channel supports HTML content.
	SupportsHTML() bool

	// MaxContentLength returns the maximum content length supported.
	// Returns 0 if there is no limit.
	MaxContentLength() int
}

// SendParams contains all parameters needed for newsletter delivery.
type SendParams struct {
	// Recipient is the target recipient information.
	Recipient models.NewsletterRecipient

	// Subject is the newsletter subject line.
	Subject string

	// BodyHTML is the HTML content (for channels that support it).
	BodyHTML string

	// BodyText is the plaintext content.
	BodyText string

	// Config is the channel-specific configuration.
	Config *models.ChannelConfig

	// Metadata contains additional delivery metadata.
	Metadata *DeliveryMetadata
}

// DeliveryMetadata contains metadata about the delivery for tracking.
type DeliveryMetadata struct {
	// DeliveryID is the unique delivery identifier.
	DeliveryID string

	// ScheduleID is the schedule that triggered this delivery.
	ScheduleID string

	// TemplateID is the template used.
	TemplateID string

	// TemplateName is the template name.
	TemplateName string

	// NewsletterType is the type of newsletter.
	NewsletterType models.NewsletterType

	// ServerName is the server name for branding.
	ServerName string

	// UnsubscribeURL is the opt-out link.
	UnsubscribeURL string
}

// DeliveryResult contains the result of a delivery attempt.
type DeliveryResult struct {
	// Success indicates if delivery was successful.
	Success bool

	// Recipient is the recipient identifier.
	Recipient string

	// RecipientType is the recipient type (user, email, webhook).
	RecipientType string

	// DeliveredAt is when delivery succeeded.
	DeliveredAt *time.Time

	// ErrorMessage contains error details if failed.
	ErrorMessage string

	// ErrorCode is a machine-readable error code.
	ErrorCode string

	// IsTransient indicates if the error is transient (can be retried).
	IsTransient bool

	// RetryAfter suggests when to retry (for rate limiting).
	RetryAfter *time.Duration

	// ExternalID is the external message ID (if provided by the service).
	ExternalID string

	// ResponseCode is the HTTP response code (for webhook-based channels).
	ResponseCode int

	// RetryCount is the number of retry attempts made.
	RetryCount int
}

// Error codes for delivery failures.
const (
	ErrorCodeInvalidConfig     = "INVALID_CONFIG"
	ErrorCodeInvalidRecipient  = "INVALID_RECIPIENT"
	ErrorCodeConnectionFailed  = "CONNECTION_FAILED"
	ErrorCodeAuthFailed        = "AUTH_FAILED"
	ErrorCodeRateLimited       = "RATE_LIMITED"
	ErrorCodeContentTooLarge   = "CONTENT_TOO_LARGE"
	ErrorCodeRecipientNotFound = "RECIPIENT_NOT_FOUND"
	ErrorCodeRecipientOptedOut = "RECIPIENT_OPTED_OUT"
	ErrorCodeServerError       = "SERVER_ERROR"
	ErrorCodeTimeout           = "TIMEOUT"
	ErrorCodeUnknown           = "UNKNOWN"
)

// ChannelRegistry manages registered delivery channels.
type ChannelRegistry struct {
	channels map[models.DeliveryChannel]Channel
}

// NewChannelRegistry creates a new channel registry with all default channels.
func NewChannelRegistry() *ChannelRegistry {
	registry := &ChannelRegistry{
		channels: make(map[models.DeliveryChannel]Channel),
	}

	// Register default channels
	registry.Register(NewEmailChannel())
	registry.Register(NewDiscordChannel())
	registry.Register(NewSlackChannel())
	registry.Register(NewTelegramChannel())
	registry.Register(NewWebhookChannel())
	registry.Register(NewInAppChannel())

	return registry
}

// Register adds a channel to the registry.
func (r *ChannelRegistry) Register(channel Channel) {
	r.channels[channel.Name()] = channel
}

// Get retrieves a channel by name.
func (r *ChannelRegistry) Get(name models.DeliveryChannel) (Channel, bool) {
	channel, ok := r.channels[name]
	return channel, ok
}

// List returns all registered channel names.
func (r *ChannelRegistry) List() []models.DeliveryChannel {
	names := make([]models.DeliveryChannel, 0, len(r.channels))
	for name := range r.channels {
		names = append(names, name)
	}
	return names
}

// ValidateConfig validates configuration for a specific channel.
func (r *ChannelRegistry) ValidateConfig(channel models.DeliveryChannel, config *models.ChannelConfig) error {
	ch, ok := r.Get(channel)
	if !ok {
		return fmt.Errorf("unknown delivery channel: %s", channel)
	}
	return ch.Validate(config)
}

// =============================================================================
// Validation Helpers
// =============================================================================

// ValidateEmail validates an email address format.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email address is required")
	}
	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email address format: %s", email)
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid email address format: %s", email)
	}
	if !strings.Contains(parts[1], ".") {
		return fmt.Errorf("invalid email domain: %s", parts[1])
	}
	return nil
}

// ValidateWebhookURL validates a webhook URL.
func ValidateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("webhook URL is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("webhook URL must use http or https scheme")
	}
	if parsed.Host == "" {
		return fmt.Errorf("webhook URL must have a host")
	}
	return nil
}

// ValidateSMTPConfig validates SMTP configuration.
func ValidateSMTPConfig(config *models.ChannelConfig) error {
	if config == nil {
		return fmt.Errorf("SMTP configuration is required")
	}
	if config.SMTPHost == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if config.SMTPPort <= 0 || config.SMTPPort > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", config.SMTPPort)
	}
	if config.SMTPFrom == "" {
		return fmt.Errorf("SMTP from address is required")
	}
	if err := ValidateEmail(config.SMTPFrom); err != nil {
		return fmt.Errorf("invalid SMTP from address: %w", err)
	}
	return nil
}

// =============================================================================
// Content Helpers
// =============================================================================

// TruncateContent truncates content to the specified length with ellipsis.
func TruncateContent(content string, maxLen int) string {
	if maxLen <= 0 || len(content) <= maxLen {
		return content
	}
	if maxLen <= 3 {
		return content[:maxLen]
	}
	return content[:maxLen-3] + "..."
}

// HTMLToPlaintext converts HTML to plain text for channels that don't support HTML.
// This is a simple conversion that strips tags; for complex HTML, consider a proper library.
func HTMLToPlaintext(html string) string {
	// Remove HTML tags
	var result strings.Builder
	inTag := false
	for _, r := range html {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result.WriteRune(r)
			}
		}
	}

	// Clean up whitespace
	text := result.String()
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")

	// Collapse multiple whitespace
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// FormatForChannel prepares content for a specific channel.
func FormatForChannel(channel Channel, subject, bodyHTML, bodyText string) (string, string) {
	// If channel doesn't support HTML, convert to plaintext
	if !channel.SupportsHTML() && bodyText == "" && bodyHTML != "" {
		bodyText = HTMLToPlaintext(bodyHTML)
	}

	// Apply length limits
	maxLen := channel.MaxContentLength()
	if maxLen > 0 {
		if bodyHTML != "" {
			bodyHTML = TruncateContent(bodyHTML, maxLen)
		}
		if bodyText != "" {
			bodyText = TruncateContent(bodyText, maxLen)
		}
	}

	return bodyHTML, bodyText
}
