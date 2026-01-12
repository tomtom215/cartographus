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

// WebhookNotifier sends alerts to a generic webhook endpoint.
type WebhookNotifier struct {
	webhookURL string
	headers    map[string]string
	client     *http.Client
	enabled    bool
	mu         sync.RWMutex

	// Rate limiting
	lastSent  time.Time
	rateLimit time.Duration
}

// WebhookConfig configures the generic webhook notifier.
type WebhookConfig struct {
	WebhookURL  string            `json:"webhook_url"`
	Headers     map[string]string `json:"headers,omitempty"` // Custom headers (e.g., auth)
	Enabled     bool              `json:"enabled"`
	RateLimitMs int               `json:"rate_limit_ms"`
}

// WebhookPayload is the JSON payload sent to the webhook endpoint.
type WebhookPayload struct {
	Alert     *Alert    `json:"alert"`
	EventType string    `json:"event_type"` // detection_alert
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"` // cartographus
}

// NewWebhookNotifier creates a new generic webhook notifier.
func NewWebhookNotifier(config WebhookConfig) *WebhookNotifier {
	rateLimit := time.Duration(config.RateLimitMs) * time.Millisecond
	if rateLimit == 0 {
		rateLimit = 500 * time.Millisecond // Default 500ms rate limit
	}

	headers := make(map[string]string)
	for k, v := range config.Headers {
		headers[k] = v
	}

	return &WebhookNotifier{
		webhookURL: config.WebhookURL,
		headers:    headers,
		enabled:    config.Enabled,
		rateLimit:  rateLimit,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (n *WebhookNotifier) Name() string {
	return "webhook"
}

// Enabled returns whether this notifier is enabled.
func (n *WebhookNotifier) Enabled() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.enabled && n.webhookURL != ""
}

// SetEnabled enables or disables the notifier.
func (n *WebhookNotifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = enabled
}

// SetWebhookURL updates the webhook URL.
func (n *WebhookNotifier) SetWebhookURL(url string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.webhookURL = url
}

// SetHeaders updates the custom headers.
func (n *WebhookNotifier) SetHeaders(headers map[string]string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.headers = make(map[string]string)
	for k, v := range headers {
		n.headers[k] = v
	}
}

// Send delivers an alert to the webhook endpoint.
func (n *WebhookNotifier) Send(ctx context.Context, alert *Alert) error {
	n.mu.RLock()
	if !n.enabled || n.webhookURL == "" {
		n.mu.RUnlock()
		return nil
	}
	webhookURL := n.webhookURL
	headers := make(map[string]string)
	for k, v := range n.headers {
		headers[k] = v
	}
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

	// Build payload
	payload := WebhookPayload{
		Alert:     alert,
		EventType: "detection_alert",
		Timestamp: time.Now(),
		Source:    "cartographus",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set content type
	req.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Update last sent time
	n.mu.Lock()
	n.lastSent = time.Now()
	n.mu.Unlock()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
