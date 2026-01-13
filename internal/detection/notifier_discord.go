// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// DiscordNotifier sends alerts to Discord via webhooks.
type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
	enabled    bool
	mu         sync.RWMutex

	// Rate limiting
	lastSent  time.Time
	rateLimit time.Duration
}

// DiscordConfig configures the Discord notifier.
type DiscordConfig struct {
	WebhookURL  string `json:"webhook_url"`
	Enabled     bool   `json:"enabled"`
	RateLimitMs int    `json:"rate_limit_ms"` // Minimum ms between messages
}

// NewDiscordNotifier creates a new Discord notifier.
func NewDiscordNotifier(config DiscordConfig) *DiscordNotifier {
	rateLimit := time.Duration(config.RateLimitMs) * time.Millisecond
	if rateLimit == 0 {
		rateLimit = 1 * time.Second // Default 1 second rate limit
	}

	return &DiscordNotifier{
		webhookURL: config.WebhookURL,
		enabled:    config.Enabled,
		rateLimit:  rateLimit,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (n *DiscordNotifier) Name() string {
	return "discord"
}

// Enabled returns whether this notifier is enabled.
func (n *DiscordNotifier) Enabled() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.enabled && n.webhookURL != ""
}

// SetEnabled enables or disables the notifier.
func (n *DiscordNotifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = enabled
}

// SetWebhookURL updates the webhook URL.
func (n *DiscordNotifier) SetWebhookURL(url string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.webhookURL = url
}

// Send delivers an alert to Discord.
func (n *DiscordNotifier) Send(ctx context.Context, alert *Alert) error {
	n.mu.RLock()
	if !n.enabled || n.webhookURL == "" {
		n.mu.RUnlock()
		return nil
	}
	webhookURL := n.webhookURL
	rateLimit := n.rateLimit
	lastSent := n.lastSent
	n.mu.RUnlock()

	// Rate limiting with context cancellation support
	if time.Since(lastSent) < rateLimit {
		waitTime := rateLimit - time.Since(lastSent)
		select {
		case <-time.After(waitTime):
			// Continue after rate limit wait
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Build Discord embed
	embed := n.buildEmbed(alert)
	payload := discordWebhookPayload{
		Embeds: []discordEmbed{embed},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Discord webhook: %w", err)
	}
	defer resp.Body.Close()

	// Update last sent time
	n.mu.Lock()
	n.lastSent = time.Now()
	n.mu.Unlock()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// buildEmbed creates a Discord embed from an alert.
func (n *DiscordNotifier) buildEmbed(alert *Alert) discordEmbed {
	color := n.severityColor(alert.Severity)

	fields := []discordEmbedField{
		{Name: "User", Value: alert.Username, Inline: true},
		{Name: "Severity", Value: string(alert.Severity), Inline: true},
		{Name: "Rule Type", Value: string(alert.RuleType), Inline: true},
	}

	if alert.IPAddress != "" {
		fields = append(fields, discordEmbedField{
			Name:   "IP Address",
			Value:  alert.IPAddress,
			Inline: true,
		})
	}

	if alert.MachineID != "" {
		fields = append(fields, discordEmbedField{
			Name:   "Device",
			Value:  truncateMachineID(alert.MachineID),
			Inline: true,
		})
	}

	return discordEmbed{
		Title:       alert.Title,
		Description: alert.Message,
		Color:       color,
		Timestamp:   alert.CreatedAt.Format(time.RFC3339),
		Fields:      fields,
		Footer: discordEmbedFooter{
			Text: "Cartographus Detection Engine",
		},
	}
}

// severityColor returns the Discord embed color for a severity level.
func (n *DiscordNotifier) severityColor(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 0xFF0000 // Red
	case SeverityWarning:
		return 0xFFA500 // Orange
	case SeverityInfo:
		return 0x3498DB // Blue
	default:
		return 0x95A5A6 // Gray
	}
}

// Discord webhook structures
type discordWebhookPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []discordEmbed `json:"embeds,omitempty"`
}

type discordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Fields      []discordEmbedField `json:"fields,omitempty"`
	Footer      discordEmbedFooter  `json:"footer,omitempty"`
}

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type discordEmbedFooter struct {
	Text string `json:"text,omitempty"`
}
