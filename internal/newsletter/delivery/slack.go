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

// SlackChannel implements Slack webhook delivery.
type SlackChannel struct {
	client *http.Client
}

// NewSlackChannel creates a new Slack delivery channel.
func NewSlackChannel() *SlackChannel {
	return &SlackChannel{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the channel identifier.
func (c *SlackChannel) Name() models.DeliveryChannel {
	return models.DeliveryChannelSlack
}

// SupportsHTML returns false as Slack uses its own mrkdwn format.
func (c *SlackChannel) SupportsHTML() bool {
	return false
}

// MaxContentLength returns Slack's block text limit.
func (c *SlackChannel) MaxContentLength() int {
	return 3000 // Slack section block text limit
}

// Validate checks if the Slack webhook configuration is valid.
func (c *SlackChannel) Validate(config *models.ChannelConfig) error {
	if config == nil {
		return fmt.Errorf("slack configuration is required")
	}
	if config.SlackWebhookURL == "" {
		return fmt.Errorf("slack webhook URL is required")
	}
	if err := ValidateWebhookURL(config.SlackWebhookURL); err != nil {
		return fmt.Errorf("invalid slack webhook URL: %w", err)
	}
	// Validate it looks like a Slack webhook
	if !strings.Contains(config.SlackWebhookURL, "hooks.slack.com/") {
		return fmt.Errorf("URL does not appear to be a slack webhook URL")
	}
	return nil
}

// SlackWebhookPayload represents the Slack webhook message structure.
type SlackWebhookPayload struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Text        string            `json:"text,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackBlock represents a Slack block element.
type SlackBlock struct {
	Type     string            `json:"type"`
	Text     *SlackTextObject  `json:"text,omitempty"`
	BlockID  string            `json:"block_id,omitempty"`
	Elements []SlackElement    `json:"elements,omitempty"`
	Fields   []SlackTextObject `json:"fields,omitempty"`
}

// SlackTextObject represents a Slack text object.
type SlackTextObject struct {
	Type  string `json:"type"` // plain_text or mrkdwn
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// SlackElement represents a Slack block element.
type SlackElement struct {
	Type string           `json:"type"`
	Text *SlackTextObject `json:"text,omitempty"`
	URL  string           `json:"url,omitempty"`
}

// SlackAttachment represents a Slack attachment (legacy but still useful for colors).
type SlackAttachment struct {
	Color      string `json:"color,omitempty"`
	Fallback   string `json:"fallback,omitempty"`
	Title      string `json:"title,omitempty"`
	TitleLink  string `json:"title_link,omitempty"`
	Text       string `json:"text,omitempty"`
	Footer     string `json:"footer,omitempty"`
	FooterIcon string `json:"footer_icon,omitempty"`
	Ts         int64  `json:"ts,omitempty"`
}

// Send delivers the newsletter via Slack webhook.
func (c *SlackChannel) Send(ctx context.Context, params *SendParams) (*DeliveryResult, error) {
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

	// Build Slack payload
	payload := c.buildPayload(params)

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to marshal payload: %v", err)
		result.ErrorCode = ErrorCodeUnknown
		return result, nil
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.Config.SlackWebhookURL, bytes.NewReader(jsonPayload))
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
	defer func() { _ = resp.Body.Close() }()

	result.ResponseCode = resp.StatusCode

	// Slack returns "ok" on success
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		body = []byte("(failed to read response)")
	}
	bodyStr := string(body)

	if resp.StatusCode == 200 && bodyStr == "ok" {
		now := time.Now()
		result.Success = true
		result.DeliveredAt = &now
		return result, nil
	}

	// Handle error
	result.ErrorMessage = fmt.Sprintf("Slack webhook returned %d: %s", resp.StatusCode, bodyStr)
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

// buildPayload constructs the Slack webhook payload.
func (c *SlackChannel) buildPayload(params *SendParams) SlackWebhookPayload {
	payload := SlackWebhookPayload{
		Channel:   params.Config.SlackChannel,
		Username:  params.Config.SlackUsername,
		IconEmoji: params.Config.SlackIconEmoji,
	}

	if payload.Username == "" && params.Metadata != nil {
		payload.Username = params.Metadata.ServerName
	}
	if payload.Username == "" {
		payload.Username = "Newsletter"
	}
	if payload.IconEmoji == "" {
		payload.IconEmoji = ":newspaper:"
	}

	// Get content - prefer plaintext for Slack
	content := params.BodyText
	if content == "" && params.BodyHTML != "" {
		content = HTMLToPlaintext(params.BodyHTML)
	}

	// Convert to Slack mrkdwn (basic conversion)
	content = c.convertToMrkdwn(content)

	// Truncate if needed
	content = TruncateContent(content, c.MaxContentLength())

	// Set fallback text
	payload.Text = params.Subject

	// Build blocks for rich formatting
	blocks := []SlackBlock{
		// Header block
		{
			Type: "header",
			Text: &SlackTextObject{
				Type: "plain_text",
				Text: params.Subject,
			},
		},
		// Divider
		{
			Type: "divider",
		},
		// Content section
		{
			Type: "section",
			Text: &SlackTextObject{
				Type: "mrkdwn",
				Text: content,
			},
		},
	}

	// Add context block with footer
	if params.Metadata != nil && params.Metadata.ServerName != "" {
		blocks = append(blocks, SlackBlock{
			Type: "context",
			Elements: []SlackElement{
				{
					Type: "mrkdwn",
					Text: &SlackTextObject{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*From %s* | %s", params.Metadata.ServerName, time.Now().Format("Jan 2, 2006")),
					},
				},
			},
		})
	}

	payload.Blocks = blocks

	// Add attachment for color accent
	payload.Attachments = []SlackAttachment{
		{
			Color: "#4A154B", // Slack purple
		},
	}

	return payload
}

// convertToMrkdwn converts plain text to Slack mrkdwn format.
func (c *SlackChannel) convertToMrkdwn(text string) string {
	// Slack mrkdwn uses:
	// *bold* _italic_ ~strikethrough~ `code` ```codeblock```
	// Links: <URL|text>

	// For now, just ensure proper line breaks
	text = strings.ReplaceAll(text, "\r\n", "\n")

	return text
}
