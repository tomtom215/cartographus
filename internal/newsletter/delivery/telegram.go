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

// TelegramChannel implements Telegram Bot API delivery.
type TelegramChannel struct {
	client  *http.Client
	baseURL string
}

// NewTelegramChannel creates a new Telegram delivery channel.
func NewTelegramChannel() *TelegramChannel {
	return &TelegramChannel{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.telegram.org",
	}
}

// Name returns the channel identifier.
func (c *TelegramChannel) Name() models.DeliveryChannel {
	return models.DeliveryChannelTelegram
}

// SupportsHTML returns true as Telegram supports HTML formatting.
func (c *TelegramChannel) SupportsHTML() bool {
	return true
}

// MaxContentLength returns Telegram's message length limit.
func (c *TelegramChannel) MaxContentLength() int {
	return 4096 // Telegram message limit
}

// Validate checks if the Telegram configuration is valid.
func (c *TelegramChannel) Validate(config *models.ChannelConfig) error {
	if config == nil {
		return fmt.Errorf("Telegram configuration is required")
	}
	if config.TelegramBotToken == "" {
		return fmt.Errorf("Telegram bot token is required")
	}
	if config.TelegramChatID == "" {
		return fmt.Errorf("Telegram chat ID is required")
	}
	// Validate bot token format (numbers:alphanumeric)
	parts := strings.Split(config.TelegramBotToken, ":")
	if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
		return fmt.Errorf("invalid Telegram bot token format")
	}
	return nil
}

// TelegramSendMessageRequest represents the Telegram sendMessage API request.
type TelegramSendMessageRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool   `json:"disable_notification,omitempty"`
}

// TelegramAPIResponse represents a Telegram API response.
type TelegramAPIResponse struct {
	OK          bool                   `json:"ok"`
	Result      map[string]interface{} `json:"result,omitempty"`
	ErrorCode   int                    `json:"error_code,omitempty"`
	Description string                 `json:"description,omitempty"`
	Parameters  *TelegramParameters    `json:"parameters,omitempty"`
}

// TelegramParameters contains additional response parameters.
type TelegramParameters struct {
	RetryAfter int `json:"retry_after,omitempty"`
}

// Send delivers the newsletter via Telegram Bot API.
func (c *TelegramChannel) Send(ctx context.Context, params *SendParams) (*DeliveryResult, error) {
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

	// Build Telegram message
	message := c.buildMessage(params)

	// Create request
	url := fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, params.Config.TelegramBotToken)

	jsonPayload, err := json.Marshal(message)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to marshal payload: %v", err)
		result.ErrorCode = ErrorCodeUnknown
		return result, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonPayload))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		result.ErrorCode = ErrorCodeUnknown
		return result, nil
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to send message: %v", err)
		result.ErrorCode = classifyHTTPError(err)
		result.IsTransient = isTransientHTTPError(result.ErrorCode)
		return result, nil
	}
	defer resp.Body.Close()

	result.ResponseCode = resp.StatusCode

	// Parse response
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to read response: %v", err)
		result.ErrorCode = ErrorCodeUnknown
		return result, nil
	}
	var apiResp TelegramAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to parse response: %v", err)
		result.ErrorCode = ErrorCodeUnknown
		return result, nil
	}

	if apiResp.OK {
		now := time.Now()
		result.Success = true
		result.DeliveredAt = &now
		// Extract message ID if available
		if msgID, ok := apiResp.Result["message_id"]; ok {
			result.ExternalID = fmt.Sprintf("%v", msgID)
		}
		return result, nil
	}

	// Handle error
	result.ErrorMessage = apiResp.Description
	result.ErrorCode = classifyTelegramError(apiResp.ErrorCode, apiResp.Description)
	result.IsTransient = isTransientHTTPError(result.ErrorCode)

	// Check for rate limiting
	if apiResp.Parameters != nil && apiResp.Parameters.RetryAfter > 0 {
		retryAfter := time.Duration(apiResp.Parameters.RetryAfter) * time.Second
		result.RetryAfter = &retryAfter
	}

	return result, nil
}

// buildMessage constructs the Telegram message.
func (c *TelegramChannel) buildMessage(params *SendParams) TelegramSendMessageRequest {
	msg := TelegramSendMessageRequest{
		ChatID:                params.Config.TelegramChatID,
		DisableWebPagePreview: true,
	}

	// Determine parse mode
	parseMode := params.Config.TelegramParseMode
	if parseMode == "" {
		parseMode = "HTML" // Default to HTML
	}
	msg.ParseMode = parseMode

	// Build content based on parse mode
	var content string
	switch parseMode {
	case "HTML":
		content = c.buildHTMLContent(params)
	case "MarkdownV2":
		content = c.buildMarkdownV2Content(params)
	default:
		// Plain text
		content = params.BodyText
		if content == "" && params.BodyHTML != "" {
			content = HTMLToPlaintext(params.BodyHTML)
		}
		msg.ParseMode = ""
	}

	// Truncate if needed
	content = TruncateContent(content, c.MaxContentLength())
	msg.Text = content

	return msg
}

// buildHTMLContent builds Telegram HTML formatted content.
func (c *TelegramChannel) buildHTMLContent(params *SendParams) string {
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("<b>%s</b>\n\n", escapeHTML(params.Subject)))

	// Content - use plaintext and convert to Telegram HTML
	content := params.BodyText
	if content == "" && params.BodyHTML != "" {
		content = HTMLToPlaintext(params.BodyHTML)
	}

	// Escape HTML entities and preserve line breaks
	content = escapeHTML(content)
	sb.WriteString(content)

	// Footer
	if params.Metadata != nil && params.Metadata.ServerName != "" {
		sb.WriteString(fmt.Sprintf("\n\n<i>From %s</i>", escapeHTML(params.Metadata.ServerName)))
	}

	return sb.String()
}

// buildMarkdownV2Content builds Telegram MarkdownV2 formatted content.
func (c *TelegramChannel) buildMarkdownV2Content(params *SendParams) string {
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("*%s*\n\n", escapeMarkdownV2(params.Subject)))

	// Content
	content := params.BodyText
	if content == "" && params.BodyHTML != "" {
		content = HTMLToPlaintext(params.BodyHTML)
	}

	// Escape MarkdownV2 special characters
	content = escapeMarkdownV2(content)
	sb.WriteString(content)

	// Footer
	if params.Metadata != nil && params.Metadata.ServerName != "" {
		sb.WriteString(fmt.Sprintf("\n\n_From %s_", escapeMarkdownV2(params.Metadata.ServerName)))
	}

	return sb.String()
}

// escapeHTML escapes HTML special characters for Telegram.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// escapeMarkdownV2 escapes MarkdownV2 special characters for Telegram.
func escapeMarkdownV2(s string) string {
	// Characters that need escaping in MarkdownV2
	special := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range special {
		s = strings.ReplaceAll(s, char, "\\"+char)
	}
	return s
}

// classifyTelegramError classifies a Telegram error into an error code.
func classifyTelegramError(code int, description string) string {
	switch code {
	case 401:
		return ErrorCodeAuthFailed
	case 400:
		if strings.Contains(description, "chat not found") {
			return ErrorCodeRecipientNotFound
		}
		if strings.Contains(description, "blocked") || strings.Contains(description, "deactivated") {
			return ErrorCodeRecipientOptedOut
		}
		return ErrorCodeInvalidConfig
	case 403:
		return ErrorCodeRecipientOptedOut // Bot was blocked by user
	case 429:
		return ErrorCodeRateLimited
	default:
		if code >= 500 {
			return ErrorCodeServerError
		}
		return ErrorCodeUnknown
	}
}
