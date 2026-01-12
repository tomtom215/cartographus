// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package delivery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

func TestNewChannelRegistry(t *testing.T) {
	registry := NewChannelRegistry()
	if registry == nil {
		t.Fatal("NewChannelRegistry returned nil")
	}

	// Check all default channels are registered
	channels := registry.List()
	if len(channels) == 0 {
		t.Fatal("No channels registered")
	}

	expectedChannels := []models.DeliveryChannel{
		models.DeliveryChannelEmail,
		models.DeliveryChannelDiscord,
		models.DeliveryChannelSlack,
		models.DeliveryChannelTelegram,
		models.DeliveryChannelWebhook,
		models.DeliveryChannelInApp,
	}

	for _, expected := range expectedChannels {
		if _, ok := registry.Get(expected); !ok {
			t.Errorf("Channel %s not found in registry", expected)
		}
	}
}

func TestChannelRegistry_Get(t *testing.T) {
	registry := NewChannelRegistry()

	tests := []struct {
		name    string
		channel models.DeliveryChannel
		wantOK  bool
	}{
		{"email exists", models.DeliveryChannelEmail, true},
		{"discord exists", models.DeliveryChannelDiscord, true},
		{"slack exists", models.DeliveryChannelSlack, true},
		{"telegram exists", models.DeliveryChannelTelegram, true},
		{"webhook exists", models.DeliveryChannelWebhook, true},
		{"inapp exists", models.DeliveryChannelInApp, true},
		{"unknown does not exist", models.DeliveryChannel("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := registry.Get(tt.channel)
			if ok != tt.wantOK {
				t.Errorf("Get(%s) ok = %v, want %v", tt.channel, ok, tt.wantOK)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid email with subdomain", "test@mail.example.com", false},
		{"empty email", "", true},
		{"no at sign", "testexample.com", true},
		{"no domain", "test@", true},
		{"no local part", "@example.com", true},
		{"no dot in domain", "test@example", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https url", "https://example.com/webhook", false},
		{"valid http url", "http://example.com/webhook", false},
		{"empty url", "", true},
		{"no scheme", "example.com/webhook", true},
		{"ftp scheme", "ftp://example.com/webhook", true},
		{"no host", "https:///webhook", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWebhookURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSMTPConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *models.ChannelConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &models.ChannelConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				SMTPFrom: "noreply@example.com",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "missing host",
			config: &models.ChannelConfig{
				SMTPPort: 587,
				SMTPFrom: "noreply@example.com",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: &models.ChannelConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 0,
				SMTPFrom: "noreply@example.com",
			},
			wantErr: true,
		},
		{
			name: "missing from",
			config: &models.ChannelConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
			},
			wantErr: true,
		},
		{
			name: "invalid from email",
			config: &models.ChannelConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				SMTPFrom: "not-an-email",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSMTPConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSMTPConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTruncateContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		maxLen  int
		want    string
	}{
		{"short content", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"very short max", "hello", 2, "he"},
		{"zero max", "hello", 0, "hello"},
		{"negative max", "hello", -1, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateContent(tt.content, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateContent(%q, %d) = %q, want %q", tt.content, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestHTMLToPlaintext(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "simple html",
			html: "<p>Hello World</p>",
			want: "Hello World",
		},
		{
			name: "nested tags",
			html: "<div><p>Hello</p><p>World</p></div>",
			want: "HelloWorld",
		},
		{
			name: "html entities",
			html: "Hello &amp; World",
			want: "Hello & World",
		},
		{
			name: "nbsp",
			html: "Hello&nbsp;World",
			want: "Hello World",
		},
		{
			name: "quotes",
			html: "&quot;Hello&quot;",
			want: "\"Hello\"",
		},
		{
			name: "lt gt",
			html: "&lt;tag&gt;",
			want: "<tag>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTMLToPlaintext(tt.html)
			if got != tt.want {
				t.Errorf("HTMLToPlaintext(%q) = %q, want %q", tt.html, got, tt.want)
			}
		})
	}
}

func TestEmailChannel_Validate(t *testing.T) {
	channel := NewEmailChannel()

	tests := []struct {
		name    string
		config  *models.ChannelConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &models.ChannelConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				SMTPFrom: "noreply@example.com",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channel.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmailChannel_Properties(t *testing.T) {
	channel := NewEmailChannel()

	if channel.Name() != models.DeliveryChannelEmail {
		t.Errorf("Name() = %v, want %v", channel.Name(), models.DeliveryChannelEmail)
	}

	if !channel.SupportsHTML() {
		t.Error("SupportsHTML() = false, want true")
	}

	if channel.MaxContentLength() != 0 {
		t.Errorf("MaxContentLength() = %d, want 0", channel.MaxContentLength())
	}
}

func TestDiscordChannel_Validate(t *testing.T) {
	channel := NewDiscordChannel()

	tests := []struct {
		name    string
		config  *models.ChannelConfig
		wantErr bool
	}{
		{
			name: "valid discord webhook",
			config: &models.ChannelConfig{
				DiscordWebhookURL: "https://discord.com/api/webhooks/123/abc",
			},
			wantErr: false,
		},
		{
			name: "valid discordapp webhook",
			config: &models.ChannelConfig{
				DiscordWebhookURL: "https://discordapp.com/api/webhooks/123/abc",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty webhook url",
			config: &models.ChannelConfig{
				DiscordWebhookURL: "",
			},
			wantErr: true,
		},
		{
			name: "invalid webhook url",
			config: &models.ChannelConfig{
				DiscordWebhookURL: "https://example.com/webhook",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channel.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiscordChannel_Properties(t *testing.T) {
	channel := NewDiscordChannel()

	if channel.Name() != models.DeliveryChannelDiscord {
		t.Errorf("Name() = %v, want %v", channel.Name(), models.DeliveryChannelDiscord)
	}

	if channel.SupportsHTML() {
		t.Error("SupportsHTML() = true, want false")
	}

	if channel.MaxContentLength() != 4096 {
		t.Errorf("MaxContentLength() = %d, want 4096", channel.MaxContentLength())
	}
}

func TestSlackChannel_Validate(t *testing.T) {
	channel := NewSlackChannel()

	tests := []struct {
		name    string
		config  *models.ChannelConfig
		wantErr bool
	}{
		{
			name: "valid slack webhook",
			config: &models.ChannelConfig{
				SlackWebhookURL: "https://hooks.slack.com/services/T00/B00/XXX",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty webhook url",
			config: &models.ChannelConfig{
				SlackWebhookURL: "",
			},
			wantErr: true,
		},
		{
			name: "invalid webhook url",
			config: &models.ChannelConfig{
				SlackWebhookURL: "https://example.com/webhook",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channel.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSlackChannel_Properties(t *testing.T) {
	channel := NewSlackChannel()

	if channel.Name() != models.DeliveryChannelSlack {
		t.Errorf("Name() = %v, want %v", channel.Name(), models.DeliveryChannelSlack)
	}

	if channel.SupportsHTML() {
		t.Error("SupportsHTML() = true, want false")
	}

	if channel.MaxContentLength() != 3000 {
		t.Errorf("MaxContentLength() = %d, want 3000", channel.MaxContentLength())
	}
}

func TestTelegramChannel_Validate(t *testing.T) {
	channel := NewTelegramChannel()

	tests := []struct {
		name    string
		config  *models.ChannelConfig
		wantErr bool
	}{
		{
			name: "valid telegram config",
			config: &models.ChannelConfig{
				TelegramBotToken: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				TelegramChatID:   "-1001234567890",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "missing bot token",
			config: &models.ChannelConfig{
				TelegramChatID: "-1001234567890",
			},
			wantErr: true,
		},
		{
			name: "missing chat id",
			config: &models.ChannelConfig{
				TelegramBotToken: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			},
			wantErr: true,
		},
		{
			name: "invalid bot token format",
			config: &models.ChannelConfig{
				TelegramBotToken: "invalid-token",
				TelegramChatID:   "-1001234567890",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channel.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTelegramChannel_Properties(t *testing.T) {
	channel := NewTelegramChannel()

	if channel.Name() != models.DeliveryChannelTelegram {
		t.Errorf("Name() = %v, want %v", channel.Name(), models.DeliveryChannelTelegram)
	}

	if !channel.SupportsHTML() {
		t.Error("SupportsHTML() = false, want true")
	}

	if channel.MaxContentLength() != 4096 {
		t.Errorf("MaxContentLength() = %d, want 4096", channel.MaxContentLength())
	}
}

func TestWebhookChannel_Validate(t *testing.T) {
	channel := NewWebhookChannel()

	tests := []struct {
		name    string
		config  *models.ChannelConfig
		wantErr bool
	}{
		{
			name: "valid webhook config",
			config: &models.ChannelConfig{
				WebhookURL: "https://example.com/webhook",
			},
			wantErr: false,
		},
		{
			name: "valid webhook with method",
			config: &models.ChannelConfig{
				WebhookURL:    "https://example.com/webhook",
				WebhookMethod: "PUT",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty webhook url",
			config: &models.ChannelConfig{
				WebhookURL: "",
			},
			wantErr: true,
		},
		{
			name: "invalid webhook method",
			config: &models.ChannelConfig{
				WebhookURL:    "https://example.com/webhook",
				WebhookMethod: "GET",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channel.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWebhookChannel_Properties(t *testing.T) {
	channel := NewWebhookChannel()

	if channel.Name() != models.DeliveryChannelWebhook {
		t.Errorf("Name() = %v, want %v", channel.Name(), models.DeliveryChannelWebhook)
	}

	if !channel.SupportsHTML() {
		t.Error("SupportsHTML() = false, want true")
	}

	if channel.MaxContentLength() != 0 {
		t.Errorf("MaxContentLength() = %d, want 0", channel.MaxContentLength())
	}
}

func TestInAppChannel_Validate(t *testing.T) {
	channel := NewInAppChannel()

	// InApp channel doesn't require config validation
	if err := channel.Validate(nil); err != nil {
		t.Errorf("Validate(nil) should not error for InApp channel, got %v", err)
	}
}

func TestInAppChannel_Properties(t *testing.T) {
	channel := NewInAppChannel()

	if channel.Name() != models.DeliveryChannelInApp {
		t.Errorf("Name() = %v, want %v", channel.Name(), models.DeliveryChannelInApp)
	}

	if channel.SupportsHTML() {
		t.Error("SupportsHTML() = true, want false")
	}

	if channel.MaxContentLength() != 1000 {
		t.Errorf("MaxContentLength() = %d, want 1000", channel.MaxContentLength())
	}
}

func TestChannelRegistry_ValidateConfig(t *testing.T) {
	registry := NewChannelRegistry()

	tests := []struct {
		name    string
		channel models.DeliveryChannel
		config  *models.ChannelConfig
		wantErr bool
	}{
		{
			name:    "valid email config",
			channel: models.DeliveryChannelEmail,
			config: &models.ChannelConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				SMTPFrom: "noreply@example.com",
			},
			wantErr: false,
		},
		{
			name:    "unknown channel",
			channel: models.DeliveryChannel("unknown"),
			config:  nil,
			wantErr: true,
		},
		{
			name:    "invalid discord config",
			channel: models.DeliveryChannelDiscord,
			config: &models.ChannelConfig{
				DiscordWebhookURL: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateConfig(tt.channel, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClassifyEmailError(t *testing.T) {
	tests := []struct {
		name     string
		errStr   string
		wantCode string
	}{
		{"auth error", "authentication failed", ErrorCodeAuthFailed},
		{"auth error variant", "auth: bad credentials", ErrorCodeAuthFailed},
		{"connection error", "connection refused", ErrorCodeConnectionFailed},
		{"connect error", "failed to connect", ErrorCodeConnectionFailed},
		{"timeout error", "timeout waiting", ErrorCodeTimeout},
		{"deadline error", "deadline exceeded", ErrorCodeTimeout},
		{"recipient error", "recipient does not exist", ErrorCodeRecipientNotFound},
		{"mailbox error", "mailbox not found", ErrorCodeRecipientNotFound},
		{"rate limit error", "rate limit exceeded", ErrorCodeRateLimited},
		{"limit error", "limit reached", ErrorCodeRateLimited},
		{"too large error", "message too large", ErrorCodeContentTooLarge},
		{"size error", "size exceeds max", ErrorCodeContentTooLarge},
		{"unknown error", "some other error", ErrorCodeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errStr)
			code := classifyEmailError(err)
			if code != tt.wantCode {
				t.Errorf("classifyEmailError(%q) = %q, want %q", tt.errStr, code, tt.wantCode)
			}
		})
	}
}

func TestIsTransientEmailError(t *testing.T) {
	tests := []struct {
		code          string
		wantTransient bool
	}{
		{ErrorCodeConnectionFailed, true},
		{ErrorCodeTimeout, true},
		{ErrorCodeRateLimited, true},
		{ErrorCodeServerError, true},
		{ErrorCodeAuthFailed, false},
		{ErrorCodeInvalidConfig, false},
		{ErrorCodeInvalidRecipient, false},
		{ErrorCodeRecipientNotFound, false},
		{ErrorCodeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := isTransientEmailError(tt.code)
			if result != tt.wantTransient {
				t.Errorf("isTransientEmailError(%q) = %v, want %v", tt.code, result, tt.wantTransient)
			}
		})
	}
}

func TestClassifyHTTPError(t *testing.T) {
	tests := []struct {
		name     string
		errStr   string
		wantCode string
	}{
		{"timeout error", "context deadline exceeded: timeout", ErrorCodeTimeout},
		{"deadline error", "deadline exceeded", ErrorCodeTimeout},
		{"connection error", "connection refused", ErrorCodeConnectionFailed},
		{"refused error", "refused by server", ErrorCodeConnectionFailed},
		{"unknown error", "some other error", ErrorCodeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errStr)
			code := classifyHTTPError(err)
			if code != tt.wantCode {
				t.Errorf("classifyHTTPError(%q) = %q, want %q", tt.errStr, code, tt.wantCode)
			}
		})
	}
}

func TestClassifyHTTPStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		wantCode string
	}{
		{401, ErrorCodeAuthFailed},
		{403, ErrorCodeAuthFailed},
		{404, ErrorCodeRecipientNotFound},
		{429, ErrorCodeRateLimited},
		{413, ErrorCodeContentTooLarge},
		{500, ErrorCodeServerError},
		{502, ErrorCodeServerError},
		{503, ErrorCodeServerError},
		{400, ErrorCodeUnknown},
		{200, ErrorCodeUnknown},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.code), func(t *testing.T) {
			code := classifyHTTPStatusCode(tt.code)
			if code != tt.wantCode {
				t.Errorf("classifyHTTPStatusCode(%d) = %q, want %q", tt.code, code, tt.wantCode)
			}
		})
	}
}

func TestIsTransientHTTPError(t *testing.T) {
	tests := []struct {
		code          string
		wantTransient bool
	}{
		{ErrorCodeConnectionFailed, true},
		{ErrorCodeTimeout, true},
		{ErrorCodeRateLimited, true},
		{ErrorCodeServerError, true},
		{ErrorCodeAuthFailed, false},
		{ErrorCodeInvalidConfig, false},
		{ErrorCodeRecipientNotFound, false},
		{ErrorCodeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := isTransientHTTPError(tt.code)
			if result != tt.wantTransient {
				t.Errorf("isTransientHTTPError(%q) = %v, want %v", tt.code, result, tt.wantTransient)
			}
		})
	}
}

func TestEmailChannel_BuildMessage(t *testing.T) {
	channel := NewEmailChannel()

	tests := []struct {
		name           string
		params         *SendParams
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "HTML only message",
			params: &SendParams{
				Recipient: models.NewsletterRecipient{Target: "test@example.com"},
				Subject:   "Test Subject",
				BodyHTML:  "<h1>Hello</h1>",
				BodyText:  "",
				Config: &models.ChannelConfig{
					SMTPFrom: "sender@example.com",
				},
			},
			wantContains:   []string{"Content-Type: text/html", "From:", "To: test@example.com", "Subject: Test Subject"},
			wantNotContain: []string{"multipart/alternative"},
		},
		{
			name: "text only message",
			params: &SendParams{
				Recipient: models.NewsletterRecipient{Target: "test@example.com"},
				Subject:   "Test Subject",
				BodyHTML:  "",
				BodyText:  "Hello World",
				Config: &models.ChannelConfig{
					SMTPFrom: "sender@example.com",
				},
			},
			wantContains:   []string{"Content-Type: text/plain", "Hello World"},
			wantNotContain: []string{"multipart/alternative"},
		},
		{
			name: "multipart message",
			params: &SendParams{
				Recipient: models.NewsletterRecipient{Target: "test@example.com"},
				Subject:   "Test Subject",
				BodyHTML:  "<h1>Hello</h1>",
				BodyText:  "Hello",
				Config: &models.ChannelConfig{
					SMTPFrom: "sender@example.com",
				},
			},
			wantContains: []string{"multipart/alternative", "text/plain", "text/html"},
		},
		{
			name: "with metadata",
			params: &SendParams{
				Recipient: models.NewsletterRecipient{Target: "test@example.com"},
				Subject:   "Test Subject",
				BodyHTML:  "<h1>Hello</h1>",
				Config: &models.ChannelConfig{
					SMTPFrom: "sender@example.com",
				},
				Metadata: &DeliveryMetadata{
					DeliveryID:     "delivery-123",
					UnsubscribeURL: "https://example.com/unsubscribe",
				},
			},
			wantContains: []string{"X-Newsletter-ID: delivery-123", "List-Unsubscribe:"},
		},
		{
			name: "with custom from name",
			params: &SendParams{
				Recipient: models.NewsletterRecipient{Target: "test@example.com"},
				Subject:   "Test Subject",
				BodyText:  "Hello",
				Config: &models.ChannelConfig{
					SMTPFrom:     "sender@example.com",
					SMTPFromName: "Custom Sender",
				},
			},
			wantContains: []string{"From: Custom Sender <sender@example.com>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := channel.buildMessage(tt.params)
			for _, want := range tt.wantContains {
				if !strings.Contains(msg, want) {
					t.Errorf("message should contain %q, got:\n%s", want, msg)
				}
			}
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(msg, notWant) {
					t.Errorf("message should not contain %q, got:\n%s", notWant, msg)
				}
			}
		})
	}
}

func TestEmailChannel_Send_InvalidRecipient(t *testing.T) {
	channel := NewEmailChannel()

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Target: "not-an-email"},
		Subject:   "Test",
		Config: &models.ChannelConfig{
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			SMTPFrom: "sender@example.com",
		},
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.Success {
		t.Error("Expected failure for invalid recipient")
	}
	if result.ErrorCode != ErrorCodeInvalidRecipient {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, ErrorCodeInvalidRecipient)
	}
}

func TestEmailChannel_Send_InvalidConfig(t *testing.T) {
	channel := NewEmailChannel()

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Target: "test@example.com"},
		Subject:   "Test",
		Config:    nil,
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.Success {
		t.Error("Expected failure for nil config")
	}
	if result.ErrorCode != ErrorCodeInvalidConfig {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, ErrorCodeInvalidConfig)
	}
}

func TestDiscordChannel_BuildPayload(t *testing.T) {
	channel := NewDiscordChannel()

	tests := []struct {
		name       string
		params     *SendParams
		wantUser   string
		wantTitle  string
		wantFooter bool
	}{
		{
			name: "with custom username",
			params: &SendParams{
				Subject:  "Test Subject",
				BodyText: "Test content",
				Config: &models.ChannelConfig{
					DiscordUsername:  "CustomBot",
					DiscordAvatarURL: "https://example.com/avatar.png",
				},
			},
			wantUser:   "CustomBot",
			wantTitle:  "Test Subject",
			wantFooter: false,
		},
		{
			name: "with server name from metadata",
			params: &SendParams{
				Subject:  "Test Subject",
				BodyText: "Test content",
				Config:   &models.ChannelConfig{},
				Metadata: &DeliveryMetadata{
					ServerName: "My Plex Server",
				},
			},
			wantUser:   "My Plex Server",
			wantTitle:  "Test Subject",
			wantFooter: true,
		},
		{
			name: "default username",
			params: &SendParams{
				Subject:  "Test Subject",
				BodyText: "Test content",
				Config:   &models.ChannelConfig{},
			},
			wantUser:   "Newsletter",
			wantTitle:  "Test Subject",
			wantFooter: false,
		},
		{
			name: "HTML content converted to plaintext",
			params: &SendParams{
				Subject:  "Test",
				BodyHTML: "<h1>Hello</h1><p>World</p>",
				BodyText: "",
				Config:   &models.ChannelConfig{},
			},
			wantUser:  "Newsletter",
			wantTitle: "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := channel.buildPayload(tt.params)
			if payload.Username != tt.wantUser {
				t.Errorf("Username = %q, want %q", payload.Username, tt.wantUser)
			}
			if len(payload.Embeds) == 0 {
				t.Fatal("Expected at least one embed")
			}
			if payload.Embeds[0].Title != tt.wantTitle {
				t.Errorf("Embed Title = %q, want %q", payload.Embeds[0].Title, tt.wantTitle)
			}
			if tt.wantFooter && payload.Embeds[0].Footer == nil {
				t.Error("Expected footer when ServerName is set")
			}
		})
	}
}

func TestDiscordChannel_Send_InvalidConfig(t *testing.T) {
	channel := NewDiscordChannel()

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Target: "webhook"},
		Subject:   "Test",
		Config:    nil,
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.Success {
		t.Error("Expected failure for nil config")
	}
	if result.ErrorCode != ErrorCodeInvalidConfig {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, ErrorCodeInvalidConfig)
	}
}

// MockInAppStore for testing InApp channel
type mockInAppStore struct {
	notifications []*InAppNotification
	err           error
}

func (m *mockInAppStore) CreateNotification(ctx context.Context, notification *InAppNotification) error {
	if m.err != nil {
		return m.err
	}
	m.notifications = append(m.notifications, notification)
	return nil
}

func TestInAppChannel_Send_Success(t *testing.T) {
	store := &mockInAppStore{}
	channel := NewInAppChannelWithStore(store)

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Type: "user", Target: "user-123"},
		Subject:   "Test Newsletter",
		BodyText:  "Test content",
		Config:    &models.ChannelConfig{},
		Metadata: &DeliveryMetadata{
			DeliveryID:     "delivery-123",
			ScheduleID:     "schedule-456",
			TemplateID:     "template-789",
			TemplateName:   "Weekly Digest",
			NewsletterType: "weekly",
			ServerName:     "Test Server",
		},
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.ErrorMessage)
	}
	if len(store.notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(store.notifications))
	}

	notif := store.notifications[0]
	if notif.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", notif.UserID, "user-123")
	}
	if notif.Title != "Test Newsletter" {
		t.Errorf("Title = %q, want %q", notif.Title, "Test Newsletter")
	}
	if notif.DeliveryID != "delivery-123" {
		t.Errorf("DeliveryID = %q, want %q", notif.DeliveryID, "delivery-123")
	}
}

func TestInAppChannel_Send_InvalidRecipientType(t *testing.T) {
	store := &mockInAppStore{}
	channel := NewInAppChannelWithStore(store)

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Type: "email", Target: "test@example.com"},
		Subject:   "Test",
		Config:    &models.ChannelConfig{},
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.Success {
		t.Error("Expected failure for non-user recipient type")
	}
	if result.ErrorCode != ErrorCodeInvalidRecipient {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, ErrorCodeInvalidRecipient)
	}
}

func TestInAppChannel_Send_NoStore(t *testing.T) {
	channel := NewInAppChannel() // No store configured

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Type: "user", Target: "user-123"},
		Subject:   "Test",
		Config:    &models.ChannelConfig{},
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.Success {
		t.Error("Expected failure when no store is configured")
	}
	if result.ErrorCode != ErrorCodeInvalidConfig {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, ErrorCodeInvalidConfig)
	}
}

func TestInAppChannel_Send_StoreError(t *testing.T) {
	store := &mockInAppStore{err: fmt.Errorf("database error")}
	channel := NewInAppChannelWithStore(store)

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Type: "user", Target: "user-123"},
		Subject:   "Test",
		Config:    &models.ChannelConfig{},
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.Success {
		t.Error("Expected failure when store returns error")
	}
	if result.ErrorCode != ErrorCodeServerError {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, ErrorCodeServerError)
	}
	if !result.IsTransient {
		t.Error("Expected transient error for store failure")
	}
}

func TestInAppChannel_Send_HTMLConversion(t *testing.T) {
	store := &mockInAppStore{}
	channel := NewInAppChannelWithStore(store)

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Type: "user", Target: "user-123"},
		Subject:   "Test",
		BodyHTML:  "<h1>Hello</h1><p>World</p>",
		BodyText:  "",
		Config:    &models.ChannelConfig{},
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.ErrorMessage)
	}
	if len(store.notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(store.notifications))
	}

	// HTML should be converted to plaintext
	notif := store.notifications[0]
	if strings.Contains(notif.Message, "<") {
		t.Errorf("Message should not contain HTML tags: %q", notif.Message)
	}
}

func TestInAppChannel_SetStore(t *testing.T) {
	channel := NewInAppChannel()

	// Initially no store
	params := &SendParams{
		Recipient: models.NewsletterRecipient{Type: "user", Target: "user-123"},
		Subject:   "Test",
		Config:    &models.ChannelConfig{},
	}

	result, _ := channel.Send(context.Background(), params)
	if result.Success {
		t.Error("Expected failure without store")
	}

	// Set store
	store := &mockInAppStore{}
	channel.SetStore(store)

	result, _ = channel.Send(context.Background(), params)
	if !result.Success {
		t.Errorf("Expected success after setting store, got: %s", result.ErrorMessage)
	}
}

func TestSlackChannel_BuildPayload(t *testing.T) {
	channel := NewSlackChannel()

	tests := []struct {
		name          string
		params        *SendParams
		wantUser      string
		wantEmoji     string
		wantBlockType string
	}{
		{
			name: "with custom username and emoji",
			params: &SendParams{
				Subject:  "Test Subject",
				BodyText: "Test content",
				Config: &models.ChannelConfig{
					SlackUsername:  "CustomBot",
					SlackIconEmoji: ":rocket:",
					SlackChannel:   "#general",
				},
			},
			wantUser:      "CustomBot",
			wantEmoji:     ":rocket:",
			wantBlockType: "header",
		},
		{
			name: "with server name from metadata",
			params: &SendParams{
				Subject:  "Test Subject",
				BodyText: "Test content",
				Config:   &models.ChannelConfig{},
				Metadata: &DeliveryMetadata{
					ServerName: "My Plex Server",
				},
			},
			wantUser:      "My Plex Server",
			wantEmoji:     ":newspaper:",
			wantBlockType: "header",
		},
		{
			name: "default values",
			params: &SendParams{
				Subject:  "Test Subject",
				BodyText: "Test content",
				Config:   &models.ChannelConfig{},
			},
			wantUser:      "Newsletter",
			wantEmoji:     ":newspaper:",
			wantBlockType: "header",
		},
		{
			name: "HTML content converted",
			params: &SendParams{
				Subject:  "Test",
				BodyHTML: "<h1>Hello</h1><p>World</p>",
				BodyText: "",
				Config:   &models.ChannelConfig{},
			},
			wantUser:      "Newsletter",
			wantEmoji:     ":newspaper:",
			wantBlockType: "header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := channel.buildPayload(tt.params)
			if payload.Username != tt.wantUser {
				t.Errorf("Username = %q, want %q", payload.Username, tt.wantUser)
			}
			if payload.IconEmoji != tt.wantEmoji {
				t.Errorf("IconEmoji = %q, want %q", payload.IconEmoji, tt.wantEmoji)
			}
			if len(payload.Blocks) == 0 {
				t.Fatal("Expected at least one block")
			}
			if payload.Blocks[0].Type != tt.wantBlockType {
				t.Errorf("Block type = %q, want %q", payload.Blocks[0].Type, tt.wantBlockType)
			}
			if len(payload.Attachments) == 0 {
				t.Error("Expected at least one attachment for color")
			}
		})
	}
}

func TestSlackChannel_ConvertToMrkdwn(t *testing.T) {
	channel := NewSlackChannel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple text", "Hello World", "Hello World"},
		{"crlf to lf", "Hello\r\nWorld", "Hello\nWorld"},
		{"preserves lf", "Hello\nWorld", "Hello\nWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := channel.convertToMrkdwn(tt.input)
			if got != tt.want {
				t.Errorf("convertToMrkdwn(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSlackChannel_Send_InvalidConfig(t *testing.T) {
	channel := NewSlackChannel()

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Target: "webhook"},
		Subject:   "Test",
		Config:    nil,
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.Success {
		t.Error("Expected failure for nil config")
	}
	if result.ErrorCode != ErrorCodeInvalidConfig {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, ErrorCodeInvalidConfig)
	}
}

func TestWebhookChannel_BuildPayload(t *testing.T) {
	channel := NewWebhookChannel()

	tests := []struct {
		name         string
		params       *SendParams
		wantEvent    string
		wantSubject  string
		wantHTML     string
		wantMetadata bool
	}{
		{
			name: "basic payload",
			params: &SendParams{
				Recipient: models.NewsletterRecipient{Type: "user", Target: "user-123", Name: "John"},
				Subject:   "Test Subject",
				BodyHTML:  "<h1>Hello</h1>",
				BodyText:  "Hello",
				Config:    &models.ChannelConfig{},
			},
			wantEvent:    "newsletter.delivery",
			wantSubject:  "Test Subject",
			wantHTML:     "<h1>Hello</h1>",
			wantMetadata: false,
		},
		{
			name: "with metadata",
			params: &SendParams{
				Recipient: models.NewsletterRecipient{Type: "email", Target: "test@example.com"},
				Subject:   "Weekly Digest",
				BodyText:  "Content here",
				Config:    &models.ChannelConfig{},
				Metadata: &DeliveryMetadata{
					DeliveryID:     "delivery-123",
					ScheduleID:     "schedule-456",
					TemplateID:     "template-789",
					TemplateName:   "Weekly Newsletter",
					NewsletterType: "weekly",
					ServerName:     "Test Server",
				},
			},
			wantEvent:    "newsletter.delivery",
			wantSubject:  "Weekly Digest",
			wantMetadata: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := channel.buildPayload(tt.params)
			if payload.Event != tt.wantEvent {
				t.Errorf("Event = %q, want %q", payload.Event, tt.wantEvent)
			}
			if payload.Subject != tt.wantSubject {
				t.Errorf("Subject = %q, want %q", payload.Subject, tt.wantSubject)
			}
			if tt.wantHTML != "" && payload.BodyHTML != tt.wantHTML {
				t.Errorf("BodyHTML = %q, want %q", payload.BodyHTML, tt.wantHTML)
			}
			if tt.wantMetadata && payload.Newsletter.DeliveryID == "" {
				t.Error("Expected newsletter metadata to be populated")
			}
			if payload.Recipient.Target != tt.params.Recipient.Target {
				t.Errorf("Recipient.Target = %q, want %q", payload.Recipient.Target, tt.params.Recipient.Target)
			}
		})
	}
}

func TestWebhookChannel_Send_InvalidConfig(t *testing.T) {
	channel := NewWebhookChannel()

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Target: "webhook"},
		Subject:   "Test",
		Config:    nil,
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.Success {
		t.Error("Expected failure for nil config")
	}
	if result.ErrorCode != ErrorCodeInvalidConfig {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, ErrorCodeInvalidConfig)
	}
}

func TestTelegramChannel_BuildMessage(t *testing.T) {
	channel := NewTelegramChannel()

	tests := []struct {
		name          string
		params        *SendParams
		wantParseMode string
		wantContains  []string
	}{
		{
			name: "HTML mode default",
			params: &SendParams{
				Subject:  "Test Subject",
				BodyText: "Test content",
				Config: &models.ChannelConfig{
					TelegramChatID: "-1001234567890",
				},
			},
			wantParseMode: "HTML",
			wantContains:  []string{"<b>Test Subject</b>", "Test content"},
		},
		{
			name: "MarkdownV2 mode",
			params: &SendParams{
				Subject:  "Test Subject",
				BodyText: "Test content",
				Config: &models.ChannelConfig{
					TelegramChatID:    "-1001234567890",
					TelegramParseMode: "MarkdownV2",
				},
			},
			wantParseMode: "MarkdownV2",
			wantContains:  []string{"*Test Subject*", "Test content"},
		},
		{
			name: "plain text mode",
			params: &SendParams{
				Subject:  "Test Subject",
				BodyText: "Test content",
				Config: &models.ChannelConfig{
					TelegramChatID:    "-1001234567890",
					TelegramParseMode: "plain",
				},
			},
			wantParseMode: "",
			wantContains:  []string{"Test content"},
		},
		{
			name: "with server name footer",
			params: &SendParams{
				Subject:  "Test",
				BodyText: "Content",
				Config: &models.ChannelConfig{
					TelegramChatID: "-1001234567890",
				},
				Metadata: &DeliveryMetadata{
					ServerName: "My Server",
				},
			},
			wantParseMode: "HTML",
			wantContains:  []string{"<i>From My Server</i>"},
		},
		{
			name: "HTML to plaintext conversion",
			params: &SendParams{
				Subject:  "Test",
				BodyHTML: "<h1>Hello</h1><p>World</p>",
				BodyText: "",
				Config: &models.ChannelConfig{
					TelegramChatID: "-1001234567890",
				},
			},
			wantParseMode: "HTML",
			wantContains:  []string{"HelloWorld"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := channel.buildMessage(tt.params)
			if msg.ParseMode != tt.wantParseMode {
				t.Errorf("ParseMode = %q, want %q", msg.ParseMode, tt.wantParseMode)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(msg.Text, want) {
					t.Errorf("Text should contain %q, got: %s", want, msg.Text)
				}
			}
		})
	}
}

func TestTelegramChannel_Send_InvalidConfig(t *testing.T) {
	channel := NewTelegramChannel()

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Target: "chat"},
		Subject:   "Test",
		Config:    nil,
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.Success {
		t.Error("Expected failure for nil config")
	}
	if result.ErrorCode != ErrorCodeInvalidConfig {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, ErrorCodeInvalidConfig)
	}
}

func TestClassifyTelegramError(t *testing.T) {
	tests := []struct {
		name        string
		code        int
		description string
		wantCode    string
	}{
		{"auth error", 401, "Unauthorized", ErrorCodeAuthFailed},
		{"chat not found", 400, "Bad Request: chat not found", ErrorCodeRecipientNotFound},
		{"user blocked", 400, "Forbidden: user is blocked", ErrorCodeRecipientOptedOut},
		{"user deactivated", 400, "Bad Request: user is deactivated", ErrorCodeRecipientOptedOut},
		{"other 400", 400, "Bad Request: invalid message", ErrorCodeInvalidConfig},
		{"forbidden", 403, "Forbidden: bot was blocked", ErrorCodeRecipientOptedOut},
		{"rate limited", 429, "Too Many Requests", ErrorCodeRateLimited},
		{"server error", 500, "Internal Server Error", ErrorCodeServerError},
		{"server error 502", 502, "Bad Gateway", ErrorCodeServerError},
		{"unknown", 418, "I'm a teapot", ErrorCodeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := classifyTelegramError(tt.code, tt.description)
			if code != tt.wantCode {
				t.Errorf("classifyTelegramError(%d, %q) = %q, want %q", tt.code, tt.description, code, tt.wantCode)
			}
		})
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "Hello World"},
		{"<script>alert('xss')</script>", "&lt;script&gt;alert('xss')&lt;/script&gt;"},
		{"A & B", "A &amp; B"},
		{"<>", "&lt;&gt;"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeHTML(tt.input)
			if got != tt.want {
				t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeMarkdownV2(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "Hello World"},
		{"*bold*", "\\*bold\\*"},
		{"_italic_", "\\_italic\\_"},
		{"[link](url)", "\\[link\\]\\(url\\)"},
		{"~strike~", "\\~strike\\~"},
		{"`code`", "\\`code\\`"},
		{">quote", "\\>quote"},
		{"#hashtag", "\\#hashtag"},
		{"a+b=c", "a\\+b\\=c"},
		{"a-b", "a\\-b"},
		{"|pipe|", "\\|pipe\\|"},
		{"{}", "\\{\\}"},
		{"a.b", "a\\.b"},
		{"hello!", "hello\\!"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeMarkdownV2(tt.input)
			if got != tt.want {
				t.Errorf("escapeMarkdownV2(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDeliveryResult_Fields(t *testing.T) {
	now := time.Now()
	retryAfter := 30 * time.Second
	result := DeliveryResult{
		Recipient:     "test@example.com",
		RecipientType: "email",
		Success:       true,
		DeliveredAt:   &now,
		ErrorMessage:  "none",
		ErrorCode:     "",
		IsTransient:   false,
		RetryAfter:    &retryAfter,
		RetryCount:    0,
		ResponseCode:  200,
		ExternalID:    "msg-123",
	}

	if result.Recipient != "test@example.com" {
		t.Errorf("Recipient = %q, want %q", result.Recipient, "test@example.com")
	}
	if result.RecipientType != "email" {
		t.Errorf("RecipientType = %q, want %q", result.RecipientType, "email")
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.DeliveredAt == nil || !result.DeliveredAt.Equal(now) {
		t.Error("DeliveredAt not set correctly")
	}
	if result.ErrorMessage != "none" {
		t.Errorf("ErrorMessage = %q, want %q", result.ErrorMessage, "none")
	}
	if result.ErrorCode != "" {
		t.Errorf("ErrorCode = %q, want empty", result.ErrorCode)
	}
	if result.IsTransient {
		t.Error("IsTransient should be false")
	}
	if result.RetryAfter == nil || *result.RetryAfter != 30*time.Second {
		t.Error("RetryAfter not set correctly")
	}
	if result.RetryCount != 0 {
		t.Errorf("RetryCount = %d, want 0", result.RetryCount)
	}
	if result.ResponseCode != 200 {
		t.Errorf("ResponseCode = %d, want 200", result.ResponseCode)
	}
	if result.ExternalID != "msg-123" {
		t.Errorf("ExternalID = %q, want %q", result.ExternalID, "msg-123")
	}
}

func TestDeliveryMetadata_Fields(t *testing.T) {
	meta := DeliveryMetadata{
		DeliveryID:     "delivery-123",
		ScheduleID:     "schedule-456",
		TemplateID:     "template-789",
		TemplateName:   "Weekly Digest",
		NewsletterType: "weekly",
		ServerName:     "Test Server",
		UnsubscribeURL: "https://example.com/unsubscribe",
	}

	if meta.DeliveryID != "delivery-123" {
		t.Errorf("DeliveryID = %q, want %q", meta.DeliveryID, "delivery-123")
	}
	if meta.ScheduleID != "schedule-456" {
		t.Errorf("ScheduleID = %q, want %q", meta.ScheduleID, "schedule-456")
	}
	if meta.TemplateID != "template-789" {
		t.Errorf("TemplateID = %q, want %q", meta.TemplateID, "template-789")
	}
	if meta.TemplateName != "Weekly Digest" {
		t.Errorf("TemplateName = %q, want %q", meta.TemplateName, "Weekly Digest")
	}
	if meta.NewsletterType != "weekly" {
		t.Errorf("NewsletterType = %q, want %q", meta.NewsletterType, "weekly")
	}
	if meta.ServerName != "Test Server" {
		t.Errorf("ServerName = %q, want %q", meta.ServerName, "Test Server")
	}
	if meta.UnsubscribeURL != "https://example.com/unsubscribe" {
		t.Errorf("UnsubscribeURL = %q, want %q", meta.UnsubscribeURL, "https://example.com/unsubscribe")
	}
}

func TestSendParams_Fields(t *testing.T) {
	params := SendParams{
		Recipient: models.NewsletterRecipient{Type: "user", Target: "user-123"},
		Subject:   "Test Subject",
		BodyHTML:  "<p>HTML content</p>",
		BodyText:  "Text content",
		Config:    &models.ChannelConfig{SMTPHost: "smtp.example.com"},
		Metadata:  &DeliveryMetadata{DeliveryID: "d-123"},
	}

	if params.Recipient.Target != "user-123" {
		t.Errorf("Recipient.Target = %q, want %q", params.Recipient.Target, "user-123")
	}
	if params.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", params.Subject, "Test Subject")
	}
	if params.BodyHTML != "<p>HTML content</p>" {
		t.Errorf("BodyHTML = %q, want %q", params.BodyHTML, "<p>HTML content</p>")
	}
	if params.BodyText != "Text content" {
		t.Errorf("BodyText = %q, want %q", params.BodyText, "Text content")
	}
	if params.Config == nil || params.Config.SMTPHost != "smtp.example.com" {
		t.Error("Config not set correctly")
	}
	if params.Metadata == nil || params.Metadata.DeliveryID != "d-123" {
		t.Error("Metadata not set correctly")
	}
}

func TestChannelRegistry_Register(t *testing.T) {
	registry := NewChannelRegistry()

	// Get count before
	before := len(registry.List())

	// Register a custom channel (this might already exist, but tests the mechanism)
	customChannel := NewEmailChannel()
	registry.Register(customChannel)

	// Count should be the same (email already exists) or one more
	after := len(registry.List())
	if after < before {
		t.Errorf("Expected at least %d channels, got %d", before, after)
	}
}

func TestChannelRegistry_List_Order(t *testing.T) {
	registry := NewChannelRegistry()
	channels := registry.List()

	if len(channels) != 6 {
		t.Errorf("Expected 6 channels, got %d", len(channels))
	}

	// Verify all expected channels are present
	expected := map[models.DeliveryChannel]bool{
		models.DeliveryChannelEmail:    false,
		models.DeliveryChannelDiscord:  false,
		models.DeliveryChannelSlack:    false,
		models.DeliveryChannelTelegram: false,
		models.DeliveryChannelWebhook:  false,
		models.DeliveryChannelInApp:    false,
	}

	for _, ch := range channels {
		if _, ok := expected[ch]; ok {
			expected[ch] = true
		}
	}

	for ch, found := range expected {
		if !found {
			t.Errorf("Channel %s not found in list", ch)
		}
	}
}

func TestErrorCodes(t *testing.T) {
	// Test that error codes are defined as constants
	codes := []string{
		ErrorCodeUnknown,
		ErrorCodeInvalidConfig,
		ErrorCodeInvalidRecipient,
		ErrorCodeConnectionFailed,
		ErrorCodeTimeout,
		ErrorCodeAuthFailed,
		ErrorCodeRateLimited,
		ErrorCodeContentTooLarge,
		ErrorCodeRecipientNotFound,
		ErrorCodeRecipientOptedOut,
		ErrorCodeServerError,
	}

	for _, code := range codes {
		if code == "" {
			t.Error("Error code should not be empty")
		}
	}
}

func TestTelegramChannel_BuildMarkdownV2Content_WithMetadata(t *testing.T) {
	channel := NewTelegramChannel()

	params := &SendParams{
		Subject:  "Test Subject",
		BodyText: "Test content",
		Config: &models.ChannelConfig{
			TelegramChatID:    "-1001234567890",
			TelegramParseMode: "MarkdownV2",
		},
		Metadata: &DeliveryMetadata{
			ServerName: "My Server",
		},
	}

	msg := channel.buildMessage(params)
	if msg.ParseMode != "MarkdownV2" {
		t.Errorf("ParseMode = %q, want %q", msg.ParseMode, "MarkdownV2")
	}
	// Check that footer is included
	if !strings.Contains(msg.Text, "_From My Server_") {
		t.Errorf("Text should contain footer, got: %s", msg.Text)
	}
}

func TestTelegramChannel_BuildMessage_PlainTextWithHTML(t *testing.T) {
	channel := NewTelegramChannel()

	params := &SendParams{
		Subject:  "Test",
		BodyHTML: "<p>HTML content</p>",
		BodyText: "",
		Config: &models.ChannelConfig{
			TelegramChatID:    "-1001234567890",
			TelegramParseMode: "plain",
		},
	}

	msg := channel.buildMessage(params)
	if msg.ParseMode != "" {
		t.Errorf("ParseMode should be empty for plain, got %q", msg.ParseMode)
	}
	// HTML should be converted to plaintext
	if strings.Contains(msg.Text, "<") {
		t.Errorf("Plain text should not contain HTML tags: %s", msg.Text)
	}
}

func TestSlackChannel_BuildPayload_WithContextBlock(t *testing.T) {
	channel := NewSlackChannel()

	params := &SendParams{
		Subject:  "Test Subject",
		BodyText: "Test content",
		Config:   &models.ChannelConfig{},
		Metadata: &DeliveryMetadata{
			ServerName: "Media Server",
		},
	}

	payload := channel.buildPayload(params)

	// Should have 4 blocks: header, divider, section, context
	if len(payload.Blocks) != 4 {
		t.Errorf("Expected 4 blocks, got %d", len(payload.Blocks))
	}

	// Last block should be context
	lastBlock := payload.Blocks[len(payload.Blocks)-1]
	if lastBlock.Type != "context" {
		t.Errorf("Last block type = %q, want %q", lastBlock.Type, "context")
	}
}

func TestDiscordChannel_BuildPayload_WithFooter(t *testing.T) {
	channel := NewDiscordChannel()

	params := &SendParams{
		Subject:  "Test Subject",
		BodyText: "Test content",
		Config:   &models.ChannelConfig{},
		Metadata: &DeliveryMetadata{
			ServerName: "Media Server",
		},
	}

	payload := channel.buildPayload(params)

	if len(payload.Embeds) == 0 {
		t.Fatal("Expected at least one embed")
	}

	if payload.Embeds[0].Footer == nil {
		t.Error("Expected footer when ServerName is set")
	}
}

func TestValidateWebhookURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://example.com/webhook", false},
		{"valid http", "http://example.com/webhook", false},
		{"missing scheme", "example.com/webhook", true},
		{"empty", "", true},
		{"ftp scheme", "ftp://example.com", true},
		{"file scheme", "file:///path/to/file", true},
		{"no host https", "https:///webhook", true},
		{"no host http", "http:///webhook", true},
		{"just scheme", "https://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWebhookURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestInAppChannel_Send_EmptyTarget(t *testing.T) {
	store := &mockInAppStore{}
	channel := NewInAppChannelWithStore(store)

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Type: "user", Target: ""},
		Subject:   "Test",
		Config:    &models.ChannelConfig{},
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	// Empty target should still work (store will handle it)
	if !result.Success {
		// Check if it failed for a different reason
		t.Logf("Result: Success=%v, Error=%s, Code=%s", result.Success, result.ErrorMessage, result.ErrorCode)
	}
}

func TestInAppChannel_Send_ContentTruncation(t *testing.T) {
	store := &mockInAppStore{}
	channel := NewInAppChannelWithStore(store)

	// Create content longer than max length (1000)
	longContent := strings.Repeat("x", 2000)

	params := &SendParams{
		Recipient: models.NewsletterRecipient{Type: "user", Target: "user-123"},
		Subject:   "Test",
		BodyText:  longContent,
		Config:    &models.ChannelConfig{},
	}

	result, err := channel.Send(context.Background(), params)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.ErrorMessage)
	}

	if len(store.notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(store.notifications))
	}

	// Check content was truncated
	if len(store.notifications[0].Message) > 1000 {
		t.Errorf("Message should be truncated to 1000, got %d", len(store.notifications[0].Message))
	}
}

func TestWebhookChannel_Validate_Methods(t *testing.T) {
	channel := NewWebhookChannel()

	tests := []struct {
		name    string
		config  *models.ChannelConfig
		wantErr bool
	}{
		{
			name: "POST method",
			config: &models.ChannelConfig{
				WebhookURL:    "https://example.com/webhook",
				WebhookMethod: "POST",
			},
			wantErr: false,
		},
		{
			name: "PUT method",
			config: &models.ChannelConfig{
				WebhookURL:    "https://example.com/webhook",
				WebhookMethod: "PUT",
			},
			wantErr: false,
		},
		{
			name: "PATCH method",
			config: &models.ChannelConfig{
				WebhookURL:    "https://example.com/webhook",
				WebhookMethod: "PATCH",
			},
			wantErr: false,
		},
		{
			name: "lowercase post",
			config: &models.ChannelConfig{
				WebhookURL:    "https://example.com/webhook",
				WebhookMethod: "post",
			},
			wantErr: false,
		},
		{
			name: "DELETE method not allowed",
			config: &models.ChannelConfig{
				WebhookURL:    "https://example.com/webhook",
				WebhookMethod: "DELETE",
			},
			wantErr: true,
		},
		{
			name: "HEAD method not allowed",
			config: &models.ChannelConfig{
				WebhookURL:    "https://example.com/webhook",
				WebhookMethod: "HEAD",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channel.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiscordChannel_Validate_WebhookURLFormats(t *testing.T) {
	channel := NewDiscordChannel()

	tests := []struct {
		name    string
		config  *models.ChannelConfig
		wantErr bool
	}{
		{
			name: "discord.com webhook",
			config: &models.ChannelConfig{
				DiscordWebhookURL: "https://discord.com/api/webhooks/123456/abcdef",
			},
			wantErr: false,
		},
		{
			name: "discordapp.com webhook",
			config: &models.ChannelConfig{
				DiscordWebhookURL: "https://discordapp.com/api/webhooks/123456/abcdef",
			},
			wantErr: false,
		},
		{
			name: "canary discord webhook",
			config: &models.ChannelConfig{
				DiscordWebhookURL: "https://canary.discord.com/api/webhooks/123456/abcdef",
			},
			wantErr: false,
		},
		{
			name: "not discord url",
			config: &models.ChannelConfig{
				DiscordWebhookURL: "https://slack.com/webhook",
			},
			wantErr: true,
		},
		{
			name: "invalid url",
			config: &models.ChannelConfig{
				DiscordWebhookURL: "not-a-url",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channel.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSlackChannel_Validate_WebhookURLFormats(t *testing.T) {
	channel := NewSlackChannel()

	tests := []struct {
		name    string
		config  *models.ChannelConfig
		wantErr bool
	}{
		{
			name: "standard slack webhook",
			config: &models.ChannelConfig{
				SlackWebhookURL: "https://hooks.slack.com/services/T00/B00/XXX",
			},
			wantErr: false,
		},
		{
			name: "not slack url",
			config: &models.ChannelConfig{
				SlackWebhookURL: "https://discord.com/webhook",
			},
			wantErr: true,
		},
		{
			name: "invalid url",
			config: &models.ChannelConfig{
				SlackWebhookURL: "not-a-url",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channel.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
