// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package delivery

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// EmailChannel implements email delivery via SMTP.
type EmailChannel struct {
	// defaultTimeout is the connection timeout.
	defaultTimeout time.Duration
}

// NewEmailChannel creates a new email delivery channel.
func NewEmailChannel() *EmailChannel {
	return &EmailChannel{
		defaultTimeout: 30 * time.Second,
	}
}

// Name returns the channel identifier.
func (c *EmailChannel) Name() models.DeliveryChannel {
	return models.DeliveryChannelEmail
}

// SupportsHTML returns true as email supports HTML content.
func (c *EmailChannel) SupportsHTML() bool {
	return true
}

// MaxContentLength returns 0 as email has no practical content limit.
func (c *EmailChannel) MaxContentLength() int {
	return 0 // No limit for email
}

// Validate checks if the SMTP configuration is valid.
func (c *EmailChannel) Validate(config *models.ChannelConfig) error {
	return ValidateSMTPConfig(config)
}

// Send delivers the newsletter via email.
func (c *EmailChannel) Send(ctx context.Context, params *SendParams) (*DeliveryResult, error) {
	result := &DeliveryResult{
		Recipient:     params.Recipient.Target,
		RecipientType: params.Recipient.Type,
	}

	// Validate recipient email
	if err := ValidateEmail(params.Recipient.Target); err != nil {
		result.ErrorMessage = err.Error()
		result.ErrorCode = ErrorCodeInvalidRecipient
		return result, nil //nolint:nilerr // Error is captured in result struct, not returned
	}

	// Validate config
	if err := c.Validate(params.Config); err != nil {
		result.ErrorMessage = err.Error()
		result.ErrorCode = ErrorCodeInvalidConfig
		return result, nil //nolint:nilerr // Error is captured in result struct, not returned
	}

	// Build email message
	msg := c.buildMessage(params)

	// Send email
	if err := c.sendSMTP(ctx, params.Config, params.Recipient.Target, msg); err != nil {
		result.ErrorMessage = err.Error()
		result.ErrorCode = classifyEmailError(err)
		result.IsTransient = isTransientEmailError(result.ErrorCode)
		return result, nil
	}

	now := time.Now()
	result.Success = true
	result.DeliveredAt = &now
	return result, nil
}

// buildMessage constructs the email message with headers.
func (c *EmailChannel) buildMessage(params *SendParams) string {
	var msg strings.Builder

	// Headers
	fromName := params.Config.SMTPFromName
	if fromName == "" {
		fromName = "Newsletter"
	}

	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, params.Config.SMTPFrom))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", params.Recipient.Target))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", params.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")

	// Add metadata headers if available
	if params.Metadata != nil {
		if params.Metadata.DeliveryID != "" {
			msg.WriteString(fmt.Sprintf("X-Newsletter-ID: %s\r\n", params.Metadata.DeliveryID))
		}
		if params.Metadata.UnsubscribeURL != "" {
			msg.WriteString(fmt.Sprintf("List-Unsubscribe: <%s>\r\n", params.Metadata.UnsubscribeURL))
			msg.WriteString("List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n")
		}
	}

	// Determine content type
	hasHTML := params.BodyHTML != ""
	hasText := params.BodyText != ""

	if hasHTML && hasText {
		// Multipart message with both HTML and text
		boundary := fmt.Sprintf("boundary_%d", time.Now().UnixNano())
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
		msg.WriteString("\r\n")

		// Plain text part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(params.BodyText)
		msg.WriteString("\r\n")

		// HTML part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(params.BodyHTML)
		msg.WriteString("\r\n")

		// End boundary
		msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if hasHTML {
		// HTML only
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(params.BodyHTML)
	} else {
		// Plain text only
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(params.BodyText)
	}

	return msg.String()
}

// sendSMTP sends the email via SMTP.
func (c *EmailChannel) sendSMTP(ctx context.Context, config *models.ChannelConfig, to, msg string) error {
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)

	// Create connection with timeout
	dialer := &net.Dialer{Timeout: c.defaultTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = conn.Close() }() //nolint:errcheck // Best effort cleanup

	// Create SMTP client
	client, err := smtp.NewClient(conn, config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer func() { _ = client.Close() }() //nolint:errcheck // Best effort cleanup

	// Start TLS if configured
	if config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: config.SMTPHost,
			MinVersion: tls.VersionTLS12,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate if credentials provided
	if config.SMTPUser != "" && config.SMTPPassword != "" {
		auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(config.SMTPFrom); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start message: %w", err)
	}

	if _, err := writer.Write([]byte(msg)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close message: %w", err)
	}

	// Quit gracefully
	if err := client.Quit(); err != nil {
		// Log but don't fail - message was sent
		return nil
	}

	return nil
}

// classifyEmailError classifies an error into an error code.
func classifyEmailError(err error) string {
	errStr := err.Error()

	if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "auth") {
		return ErrorCodeAuthFailed
	}
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "connect") {
		return ErrorCodeConnectionFailed
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return ErrorCodeTimeout
	}
	if strings.Contains(errStr, "recipient") || strings.Contains(errStr, "mailbox") {
		return ErrorCodeRecipientNotFound
	}
	if strings.Contains(errStr, "rate") || strings.Contains(errStr, "limit") {
		return ErrorCodeRateLimited
	}
	if strings.Contains(errStr, "too large") || strings.Contains(errStr, "size") {
		return ErrorCodeContentTooLarge
	}

	return ErrorCodeUnknown
}

// isTransientEmailError returns true if the error is transient and can be retried.
func isTransientEmailError(code string) bool {
	switch code {
	case ErrorCodeConnectionFailed, ErrorCodeTimeout, ErrorCodeRateLimited, ErrorCodeServerError:
		return true
	default:
		return false
	}
}
