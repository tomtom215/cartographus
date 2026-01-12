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

func TestNewWebhookNotifier(t *testing.T) {
	config := WebhookConfig{
		WebhookURL:  "https://example.com/webhook",
		Headers:     map[string]string{"Authorization": "Bearer token"},
		Enabled:     true,
		RateLimitMs: 500,
	}

	notifier := NewWebhookNotifier(config)

	if notifier == nil {
		t.Fatal("notifier should not be nil")
	}
	if notifier.Name() != "webhook" {
		t.Errorf("Name() = %q, want %q", notifier.Name(), "webhook")
	}
	if !notifier.Enabled() {
		t.Error("notifier should be enabled")
	}
}

func TestNewWebhookNotifier_DefaultRateLimit(t *testing.T) {
	config := WebhookConfig{
		WebhookURL:  "https://example.com/webhook",
		Enabled:     true,
		RateLimitMs: 0, // Should use default
	}

	notifier := NewWebhookNotifier(config)

	if notifier.rateLimit != 500*time.Millisecond {
		t.Errorf("rateLimit = %v, want 500ms", notifier.rateLimit)
	}
}

func TestWebhookNotifier_Enabled(t *testing.T) {
	tests := []struct {
		name     string
		config   WebhookConfig
		expected bool
	}{
		{
			name: "enabled with URL",
			config: WebhookConfig{
				WebhookURL: "https://example.com/webhook",
				Enabled:    true,
			},
			expected: true,
		},
		{
			name: "disabled",
			config: WebhookConfig{
				WebhookURL: "https://example.com/webhook",
				Enabled:    false,
			},
			expected: false,
		},
		{
			name: "enabled but no URL",
			config: WebhookConfig{
				WebhookURL: "",
				Enabled:    true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := NewWebhookNotifier(tt.config)
			if notifier.Enabled() != tt.expected {
				t.Errorf("Enabled() = %v, want %v", notifier.Enabled(), tt.expected)
			}
		})
	}
}

func TestWebhookNotifier_SetEnabled(t *testing.T) {
	notifier := NewWebhookNotifier(WebhookConfig{
		WebhookURL: "https://example.com/webhook",
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

func TestWebhookNotifier_SetWebhookURL(t *testing.T) {
	notifier := NewWebhookNotifier(WebhookConfig{
		WebhookURL: "https://example.com/old",
		Enabled:    true,
	})

	newURL := "https://example.com/new"
	notifier.SetWebhookURL(newURL)

	if notifier.webhookURL != newURL {
		t.Errorf("webhookURL = %q, want %q", notifier.webhookURL, newURL)
	}
}

func TestWebhookNotifier_SetHeaders(t *testing.T) {
	notifier := NewWebhookNotifier(WebhookConfig{
		WebhookURL: "https://example.com/webhook",
		Headers:    map[string]string{"Old-Header": "old"},
		Enabled:    true,
	})

	newHeaders := map[string]string{
		"Authorization": "Bearer new-token",
		"X-Custom":      "value",
	}
	notifier.SetHeaders(newHeaders)

	if notifier.headers["Authorization"] != "Bearer new-token" {
		t.Errorf("Authorization header = %q, want %q", notifier.headers["Authorization"], "Bearer new-token")
	}
	if notifier.headers["X-Custom"] != "value" {
		t.Errorf("X-Custom header = %q, want %q", notifier.headers["X-Custom"], "value")
	}
	if _, exists := notifier.headers["Old-Header"]; exists {
		t.Error("Old-Header should not exist after SetHeaders")
	}
}

func TestWebhookNotifier_Send_Disabled(t *testing.T) {
	notifier := NewWebhookNotifier(WebhookConfig{
		WebhookURL: "https://example.com/webhook",
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

func TestWebhookNotifier_Send_Success(t *testing.T) {
	var receivedPayload WebhookPayload
	var receivedHeaders http.Header
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		receivedHeaders = r.Header.Clone()

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

	notifier := NewWebhookNotifier(WebhookConfig{
		WebhookURL: server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
			"X-Custom":      "custom-value",
		},
		Enabled:     true,
		RateLimitMs: 10,
	})

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		UserID:    42,
		Username:  "testuser",
		IPAddress: "1.2.3.4",
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

	// Verify headers
	if receivedHeaders.Get("Authorization") != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", receivedHeaders.Get("Authorization"), "Bearer test-token")
	}
	if receivedHeaders.Get("X-Custom") != "custom-value" {
		t.Errorf("X-Custom header = %q, want %q", receivedHeaders.Get("X-Custom"), "custom-value")
	}

	// Verify payload
	if receivedPayload.EventType != "detection_alert" {
		t.Errorf("EventType = %q, want %q", receivedPayload.EventType, "detection_alert")
	}
	if receivedPayload.Source != "cartographus" {
		t.Errorf("Source = %q, want %q", receivedPayload.Source, "cartographus")
	}
	if receivedPayload.Alert == nil {
		t.Error("Alert should not be nil")
	}
	if receivedPayload.Alert.Username != "testuser" {
		t.Errorf("Alert.Username = %q, want %q", receivedPayload.Alert.Username, "testuser")
	}
}

func TestWebhookNotifier_Send_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(WebhookConfig{
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
		t.Error("expected error for 502 status")
	}
}

func TestWebhookNotifier_Send_ClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(WebhookConfig{
		WebhookURL:  server.URL,
		Enabled:     true,
		RateLimitMs: 10,
	})

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		Severity:  SeverityInfo,
		Title:     "Test",
		Message:   "Test",
		CreatedAt: time.Now(),
	}

	err := notifier.Send(context.Background(), alert)
	if err == nil {
		t.Error("expected error for 400 status")
	}
}

func TestWebhookNotifier_RateLimiting(t *testing.T) {
	var requestTimes []time.Time

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(WebhookConfig{
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

func TestWebhookNotifier_HeadersCopy(t *testing.T) {
	originalHeaders := map[string]string{
		"Authorization": "Bearer token",
	}

	notifier := NewWebhookNotifier(WebhookConfig{
		WebhookURL: "https://example.com/webhook",
		Headers:    originalHeaders,
		Enabled:    true,
	})

	// Modify original headers
	originalHeaders["New-Header"] = "value"

	// Notifier should not be affected
	if _, exists := notifier.headers["New-Header"]; exists {
		t.Error("notifier headers should be a copy, not reference")
	}
}

func TestWebhookNotifier_ConcurrentSend(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		time.Sleep(10 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(WebhookConfig{
		WebhookURL:  server.URL,
		Enabled:     true,
		RateLimitMs: 1, // Very low rate limit for testing
	})

	// Send multiple alerts concurrently
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func(id int) {
			alert := &Alert{
				RuleType:  RuleTypeImpossibleTravel,
				UserID:    id,
				Severity:  SeverityInfo,
				Title:     "Test",
				Message:   "Test",
				CreatedAt: time.Now(),
			}
			done <- notifier.Send(context.Background(), alert)
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 3; i++ {
		if err := <-done; err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}

	if atomic.LoadInt32(&requestCount) != 3 {
		t.Errorf("expected 3 requests, got %d", requestCount)
	}
}
