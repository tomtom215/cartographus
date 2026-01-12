// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewDiscordNotifier(t *testing.T) {
	config := DiscordConfig{
		WebhookURL:  "https://discord.com/api/webhooks/test",
		Enabled:     true,
		RateLimitMs: 500,
	}

	notifier := NewDiscordNotifier(config)

	if notifier == nil {
		t.Fatal("notifier should not be nil")
	}
	if notifier.Name() != "discord" {
		t.Errorf("Name() = %q, want %q", notifier.Name(), "discord")
	}
	if !notifier.Enabled() {
		t.Error("notifier should be enabled")
	}
}

func TestNewDiscordNotifier_DefaultRateLimit(t *testing.T) {
	config := DiscordConfig{
		WebhookURL:  "https://discord.com/api/webhooks/test",
		Enabled:     true,
		RateLimitMs: 0, // Should use default
	}

	notifier := NewDiscordNotifier(config)

	if notifier.rateLimit != 1*time.Second {
		t.Errorf("rateLimit = %v, want 1s", notifier.rateLimit)
	}
}

func TestDiscordNotifier_Enabled(t *testing.T) {
	tests := []struct {
		name     string
		config   DiscordConfig
		expected bool
	}{
		{
			name: "enabled with URL",
			config: DiscordConfig{
				WebhookURL: "https://discord.com/api/webhooks/test",
				Enabled:    true,
			},
			expected: true,
		},
		{
			name: "disabled",
			config: DiscordConfig{
				WebhookURL: "https://discord.com/api/webhooks/test",
				Enabled:    false,
			},
			expected: false,
		},
		{
			name: "enabled but no URL",
			config: DiscordConfig{
				WebhookURL: "",
				Enabled:    true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := NewDiscordNotifier(tt.config)
			if notifier.Enabled() != tt.expected {
				t.Errorf("Enabled() = %v, want %v", notifier.Enabled(), tt.expected)
			}
		})
	}
}

func TestDiscordNotifier_SetEnabled(t *testing.T) {
	notifier := NewDiscordNotifier(DiscordConfig{
		WebhookURL: "https://discord.com/api/webhooks/test",
		Enabled:    true,
	})

	notifier.SetEnabled(false)
	if notifier.enabled {
		t.Error("should be disabled after SetEnabled(false)")
	}

	notifier.SetEnabled(true)
	if !notifier.enabled {
		t.Error("should be enabled after SetEnabled(true)")
	}
}

func TestDiscordNotifier_SetWebhookURL(t *testing.T) {
	notifier := NewDiscordNotifier(DiscordConfig{
		WebhookURL: "https://discord.com/api/webhooks/old",
		Enabled:    true,
	})

	newURL := "https://discord.com/api/webhooks/new"
	notifier.SetWebhookURL(newURL)

	if notifier.webhookURL != newURL {
		t.Errorf("webhookURL = %q, want %q", notifier.webhookURL, newURL)
	}
}

func TestDiscordNotifier_Send_Disabled(t *testing.T) {
	notifier := NewDiscordNotifier(DiscordConfig{
		WebhookURL: "https://discord.com/api/webhooks/test",
		Enabled:    false,
	})

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		UserID:    1,
		Username:  "testuser",
		Severity:  SeverityCritical,
		Title:     "Test Alert",
		Message:   "Test message",
		CreatedAt: time.Now(),
	}

	err := notifier.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDiscordNotifier_Send_Success(t *testing.T) {
	var receivedPayload discordWebhookPayload
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Errorf("failed to decode payload: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewDiscordNotifier(DiscordConfig{
		WebhookURL:  server.URL,
		Enabled:     true,
		RateLimitMs: 10, // Low rate limit for testing
	})

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		UserID:    1,
		Username:  "testuser",
		IPAddress: "1.2.3.4",
		MachineID: "machine123",
		Severity:  SeverityCritical,
		Title:     "Test Alert",
		Message:   "Test message",
		CreatedAt: time.Now(),
	}

	err := notifier.Send(context.Background(), alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected 1 request, got %d", requestCount)
	}

	if len(receivedPayload.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(receivedPayload.Embeds))
	}

	embed := receivedPayload.Embeds[0]
	if embed.Title != "Test Alert" {
		t.Errorf("embed title = %q, want %q", embed.Title, "Test Alert")
	}
	if embed.Description != "Test message" {
		t.Errorf("embed description = %q, want %q", embed.Description, "Test message")
	}
}

func TestDiscordNotifier_Send_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewDiscordNotifier(DiscordConfig{
		WebhookURL:  server.URL,
		Enabled:     true,
		RateLimitMs: 10,
	})

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		Severity:  SeverityCritical,
		Title:     "Test Alert",
		Message:   "Test message",
		CreatedAt: time.Now(),
	}

	err := notifier.Send(context.Background(), alert)
	if err == nil {
		t.Error("expected error for 500 status")
	}
}

func TestDiscordNotifier_SeverityColor(t *testing.T) {
	notifier := NewDiscordNotifier(DiscordConfig{})

	tests := []struct {
		severity Severity
		expected int
	}{
		{SeverityCritical, 0xFF0000},
		{SeverityWarning, 0xFFA500},
		{SeverityInfo, 0x3498DB},
		{Severity("unknown"), 0x95A5A6},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			color := notifier.severityColor(tt.severity)
			if color != tt.expected {
				t.Errorf("severityColor(%q) = 0x%X, want 0x%X", tt.severity, color, tt.expected)
			}
		})
	}
}

func TestDiscordNotifier_BuildEmbed(t *testing.T) {
	notifier := NewDiscordNotifier(DiscordConfig{})

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		UserID:    42,
		Username:  "testuser",
		IPAddress: "1.2.3.4",
		MachineID: "verylongmachineid12345678",
		Severity:  SeverityWarning,
		Title:     "Test Title",
		Message:   "Test Description",
		CreatedAt: time.Now(),
	}

	embed := notifier.buildEmbed(alert)

	if embed.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", embed.Title, "Test Title")
	}
	if embed.Description != "Test Description" {
		t.Errorf("Description = %q, want %q", embed.Description, "Test Description")
	}
	if embed.Color != 0xFFA500 {
		t.Errorf("Color = 0x%X, want 0x%X", embed.Color, 0xFFA500)
	}
	if embed.Footer.Text != "Cartographus Detection Engine" {
		t.Errorf("Footer = %q, want %q", embed.Footer.Text, "Cartographus Detection Engine")
	}

	// Check fields
	if len(embed.Fields) < 3 {
		t.Fatalf("expected at least 3 fields, got %d", len(embed.Fields))
	}

	// Verify required fields exist
	foundUser := false
	foundSeverity := false
	foundRuleType := false
	foundIP := false
	foundDevice := false

	for _, field := range embed.Fields {
		switch field.Name {
		case "User":
			foundUser = true
			if field.Value != "testuser" {
				t.Errorf("User field value = %q, want %q", field.Value, "testuser")
			}
		case "Severity":
			foundSeverity = true
			if field.Value != "warning" {
				t.Errorf("Severity field value = %q, want %q", field.Value, "warning")
			}
		case "Rule Type":
			foundRuleType = true
			if field.Value != "impossible_travel" {
				t.Errorf("Rule Type field value = %q, want %q", field.Value, "impossible_travel")
			}
		case "IP Address":
			foundIP = true
			if field.Value != "1.2.3.4" {
				t.Errorf("IP Address field value = %q, want %q", field.Value, "1.2.3.4")
			}
		case "Device":
			foundDevice = true
		}
	}

	if !foundUser {
		t.Error("missing User field")
	}
	if !foundSeverity {
		t.Error("missing Severity field")
	}
	if !foundRuleType {
		t.Error("missing Rule Type field")
	}
	if !foundIP {
		t.Error("missing IP Address field")
	}
	if !foundDevice {
		t.Error("missing Device field")
	}
}

func TestDiscordNotifier_BuildEmbed_NoOptionalFields(t *testing.T) {
	notifier := NewDiscordNotifier(DiscordConfig{})

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		UserID:    42,
		Username:  "testuser",
		IPAddress: "", // No IP
		MachineID: "", // No machine ID
		Severity:  SeverityInfo,
		Title:     "Test Title",
		Message:   "Test Description",
		CreatedAt: time.Now(),
	}

	embed := notifier.buildEmbed(alert)

	// Should only have 3 required fields
	if len(embed.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(embed.Fields))
	}

	// Verify no IP or Device fields
	for _, field := range embed.Fields {
		if field.Name == "IP Address" {
			t.Error("should not have IP Address field when empty")
		}
		if field.Name == "Device" {
			t.Error("should not have Device field when empty")
		}
	}
}

func TestDiscordNotifier_RateLimiting(t *testing.T) {
	var requestTimes []time.Time

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewDiscordNotifier(DiscordConfig{
		WebhookURL:  server.URL,
		Enabled:     true,
		RateLimitMs: 100, // 100ms rate limit
	})

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		Severity:  SeverityInfo,
		Title:     "Test",
		Message:   "Test",
		CreatedAt: time.Now(),
	}

	// Send first request
	start := time.Now()
	err := notifier.Send(context.Background(), alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Send second request immediately (should be rate limited)
	err = notifier.Send(context.Background(), alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	if len(requestTimes) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requestTimes))
	}

	// Second request should be delayed by at least 80ms (allowing some tolerance)
	if elapsed < 80*time.Millisecond {
		t.Errorf("rate limiting not working: elapsed = %v, expected >= 80ms", elapsed)
	}
}
