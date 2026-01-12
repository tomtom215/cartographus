// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build integration

package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/tomtom215/cartographus/internal/models"
)

// mockWebhookServer provides a mock HTTP server for testing webhook deliveries.
type mockWebhookServer struct {
	server   *httptest.Server
	captures []capturedRequest
	mu       sync.Mutex
	status   int
	body     []byte
}

type capturedRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
}

func newMockWebhookServer(t *testing.T) *mockWebhookServer {
	t.Helper()

	mws := &mockWebhookServer{
		status:   http.StatusOK,
		captures: make([]capturedRequest, 0),
	}

	mws.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, 0)
		if r.Body != nil {
			buf := make([]byte, 4096)
			for {
				n, err := r.Body.Read(buf)
				if n > 0 {
					body = append(body, buf[:n]...)
				}
				if err != nil {
					break
				}
			}
			r.Body.Close()
		}

		mws.mu.Lock()
		mws.captures = append(mws.captures, capturedRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Headers: r.Header.Clone(),
			Body:    body,
		})
		mws.mu.Unlock()

		w.WriteHeader(mws.status)
		if mws.body != nil {
			w.Write(mws.body) //nolint:errcheck
		}
	}))

	return mws
}

func (m *mockWebhookServer) url() string {
	return m.server.URL
}

func (m *mockWebhookServer) close() {
	m.server.Close()
}

func (m *mockWebhookServer) getCaptures() []capturedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]capturedRequest, len(m.captures))
	copy(result, m.captures)
	return result
}

func (m *mockWebhookServer) setResponse(status int, body []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = status
	m.body = body
}

// TestManager_Deliver_Integration tests the full delivery pipeline with mock webhook.
func TestManager_Deliver_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create mock webhook server
	mockServer := newMockWebhookServer(t)
	defer mockServer.close()

	// Set up success response
	mockServer.setResponse(http.StatusOK, []byte(`{"ok":true}`))

	// Create manager
	logger := zerolog.New(zerolog.NewTestWriter(t))
	manager := NewManager(&logger, ManagerConfig{
		MaxRetries:  3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Parallelism: 5,
	})

	// Create delivery request
	req := &DeliveryRequest{
		DeliveryID: "integration-test-001",
		ScheduleID: "schedule-001",
		Template: &models.NewsletterTemplate{
			ID:   "template-001",
			Name: "Test Newsletter",
			Type: models.NewsletterTypeWeeklyDigest,
		},
		Recipients: []models.NewsletterRecipient{
			{
				Type:   "webhook",
				Target: mockServer.url() + "/webhook/1",
				Name:   "Test Recipient 1",
			},
			{
				Type:   "webhook",
				Target: mockServer.url() + "/webhook/2",
				Name:   "Test Recipient 2",
			},
		},
		Channels: []models.DeliveryChannel{
			models.DeliveryChannelWebhook,
		},
		ChannelConfigs: map[models.DeliveryChannel]*models.ChannelConfig{
			models.DeliveryChannelWebhook: {
				// Webhook channel doesn't need specific config
			},
		},
		RenderedSubject: "Weekly Newsletter - Test",
		RenderedHTML:    "<h1>Test Newsletter</h1><p>Content here</p>",
		RenderedText:    "Test Newsletter\n\nContent here",
	}

	// Deliver
	report, err := manager.Deliver(ctx, req)
	if err != nil {
		t.Fatalf("Deliver failed: %v", err)
	}

	// Verify report
	if report.DeliveryID != req.DeliveryID {
		t.Errorf("DeliveryID = %s, want %s", report.DeliveryID, req.DeliveryID)
	}

	if report.TotalRecipients != 2 {
		t.Errorf("TotalRecipients = %d, want 2", report.TotalRecipients)
	}

	if report.SuccessfulDeliveries != 2 {
		t.Errorf("SuccessfulDeliveries = %d, want 2", report.SuccessfulDeliveries)
	}

	if report.FailedDeliveries != 0 {
		t.Errorf("FailedDeliveries = %d, want 0", report.FailedDeliveries)
	}

	if report.Status != models.DeliveryStatusDelivered {
		t.Errorf("Status = %s, want %s", report.Status, models.DeliveryStatusDelivered)
	}

	// Verify webhook was called
	captures := mockServer.getCaptures()
	if len(captures) != 2 {
		t.Errorf("Webhook called %d times, want 2", len(captures))
	}

	// Verify webhook payload
	for _, capture := range captures {
		if capture.Method != http.MethodPost {
			t.Errorf("Webhook method = %s, want POST", capture.Method)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(capture.Body, &payload); err != nil {
			t.Errorf("Failed to unmarshal webhook payload: %v", err)
			continue
		}

		// Verify content is present
		if _, ok := payload["html"]; !ok {
			t.Error("Webhook payload missing 'html' field")
		}
		if _, ok := payload["text"]; !ok {
			t.Error("Webhook payload missing 'text' field")
		}
	}

	t.Logf("Delivery completed in %dms", report.DurationMS)
}

// TestManager_Deliver_PartialFailure tests handling of partial delivery failures.
func TestManager_Deliver_PartialFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create mock webhook server that alternates success/failure
	callCount := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count := callCount
		callCount++
		mu.Unlock()

		if count%2 == 0 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"server error"}`)) //nolint:errcheck
		}
	}))
	defer server.Close()

	// Create manager with fast retries
	logger := zerolog.New(zerolog.NewTestWriter(t))
	manager := NewManager(&logger, ManagerConfig{
		MaxRetries:  1, // Minimal retries for faster test
		BaseDelay:   5 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Parallelism: 1, // Sequential to ensure predictable order
	})

	// Create delivery request with multiple recipients
	req := &DeliveryRequest{
		DeliveryID: "integration-test-partial",
		Template: &models.NewsletterTemplate{
			ID:   "template-001",
			Name: "Test Newsletter",
			Type: models.NewsletterTypeWeeklyDigest,
		},
		Recipients: []models.NewsletterRecipient{
			{Type: "webhook", Target: server.URL + "/1", Name: "R1"},
			{Type: "webhook", Target: server.URL + "/2", Name: "R2"},
			{Type: "webhook", Target: server.URL + "/3", Name: "R3"},
			{Type: "webhook", Target: server.URL + "/4", Name: "R4"},
		},
		Channels: []models.DeliveryChannel{models.DeliveryChannelWebhook},
		ChannelConfigs: map[models.DeliveryChannel]*models.ChannelConfig{
			models.DeliveryChannelWebhook: {},
		},
		RenderedHTML: "<p>Test</p>",
		RenderedText: "Test",
	}

	// Deliver
	report, err := manager.Deliver(ctx, req)
	if err != nil {
		t.Fatalf("Deliver failed: %v", err)
	}

	// Verify partial status
	if report.Status != models.DeliveryStatusPartial {
		t.Errorf("Status = %s, want %s", report.Status, models.DeliveryStatusPartial)
	}

	// Should have mix of successes and failures
	if report.SuccessfulDeliveries == 0 {
		t.Error("Expected some successful deliveries")
	}
	if report.FailedDeliveries == 0 {
		t.Error("Expected some failed deliveries")
	}

	t.Logf("Partial delivery: %d succeeded, %d failed",
		report.SuccessfulDeliveries, report.FailedDeliveries)
}

// TestManager_Deliver_Timeout tests delivery timeout handling.
func TestManager_Deliver_Timeout(t *testing.T) {
	// Create a server that hangs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't respond - simulate timeout
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	manager := NewManager(&logger, ManagerConfig{
		MaxRetries:  1,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    50 * time.Millisecond,
		Parallelism: 1,
	})

	req := &DeliveryRequest{
		DeliveryID: "integration-test-timeout",
		Template: &models.NewsletterTemplate{
			ID:   "template-001",
			Name: "Test Newsletter",
		},
		Recipients: []models.NewsletterRecipient{
			{Type: "webhook", Target: server.URL, Name: "Timeout Test"},
		},
		Channels: []models.DeliveryChannel{models.DeliveryChannelWebhook},
		ChannelConfigs: map[models.DeliveryChannel]*models.ChannelConfig{
			models.DeliveryChannelWebhook: {},
		},
		RenderedHTML: "<p>Test</p>",
		RenderedText: "Test",
	}

	report, err := manager.Deliver(ctx, req)

	// Should complete (possibly with failures) or return error
	if err != nil {
		t.Logf("Delivery returned error (expected for timeout): %v", err)
	}

	if report != nil {
		t.Logf("Delivery status: %s, failed: %d", report.Status, report.FailedDeliveries)
		// With timeout, expect failures
		if report.SuccessfulDeliveries > 0 {
			t.Error("Expected no successful deliveries with timeout")
		}
	}
}

// TestManager_Deliver_Concurrency tests concurrent delivery processing.
func TestManager_Deliver_Concurrency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Track concurrent requests
	var maxConcurrent int32
	var currentConcurrent int32
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		currentConcurrent++
		if currentConcurrent > maxConcurrent {
			maxConcurrent = currentConcurrent
		}
		mu.Unlock()

		// Simulate some work
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		currentConcurrent--
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}))
	defer server.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	parallelism := 5
	manager := NewManager(&logger, ManagerConfig{
		MaxRetries:  1,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    50 * time.Millisecond,
		Parallelism: parallelism,
	})

	// Create many recipients to test parallelism
	recipients := make([]models.NewsletterRecipient, 20)
	for i := 0; i < 20; i++ {
		recipients[i] = models.NewsletterRecipient{
			Type:   "webhook",
			Target: server.URL,
			Name:   "Recipient",
		}
	}

	req := &DeliveryRequest{
		DeliveryID:     "integration-test-concurrency",
		Template:       &models.NewsletterTemplate{ID: "t1", Name: "Test"},
		Recipients:     recipients,
		Channels:       []models.DeliveryChannel{models.DeliveryChannelWebhook},
		ChannelConfigs: map[models.DeliveryChannel]*models.ChannelConfig{models.DeliveryChannelWebhook: {}},
		RenderedHTML:   "<p>Test</p>",
		RenderedText:   "Test",
	}

	report, err := manager.Deliver(ctx, req)
	if err != nil {
		t.Fatalf("Deliver failed: %v", err)
	}

	if report.SuccessfulDeliveries != 20 {
		t.Errorf("SuccessfulDeliveries = %d, want 20", report.SuccessfulDeliveries)
	}

	// Verify parallelism was used
	mu.Lock()
	max := maxConcurrent
	mu.Unlock()

	if max == 0 {
		t.Error("No concurrent requests detected")
	}
	if int(max) > parallelism {
		t.Errorf("Max concurrent = %d, exceeded parallelism limit %d", max, parallelism)
	}

	t.Logf("Concurrency test: max concurrent = %d (limit: %d)", max, parallelism)
}

// TestManager_InAppDelivery_Integration tests in-app notification delivery.
func TestManager_InAppDelivery_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create mock in-app store
	store := &mockInAppStore{
		notifications: make(map[string][]*InAppNotification),
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))
	manager := NewManager(&logger, DefaultManagerConfig())
	manager.SetInAppStore(store)

	req := &DeliveryRequest{
		DeliveryID: "integration-test-inapp",
		Template: &models.NewsletterTemplate{
			ID:   "template-001",
			Name: "In-App Test",
		},
		Recipients: []models.NewsletterRecipient{
			{Type: "user", Target: "user-001", Name: "User 1"},
			{Type: "user", Target: "user-002", Name: "User 2"},
		},
		Channels: []models.DeliveryChannel{models.DeliveryChannelInApp},
		ChannelConfigs: map[models.DeliveryChannel]*models.ChannelConfig{
			models.DeliveryChannelInApp: {},
		},
		RenderedSubject: "In-App Test Subject",
		RenderedHTML:    "<p>In-App content</p>",
		RenderedText:    "In-App content",
	}

	report, err := manager.Deliver(ctx, req)
	if err != nil {
		t.Fatalf("Deliver failed: %v", err)
	}

	if report.Status != models.DeliveryStatusDelivered {
		t.Errorf("Status = %s, want %s", report.Status, models.DeliveryStatusDelivered)
	}

	// Verify notifications were stored
	store.mu.Lock()
	defer store.mu.Unlock()

	if len(store.notifications["user-001"]) != 1 {
		t.Errorf("User user-001 has %d notifications, want 1", len(store.notifications["user-001"]))
	}
	if len(store.notifications["user-002"]) != 1 {
		t.Errorf("User user-002 has %d notifications, want 1", len(store.notifications["user-002"]))
	}

	t.Logf("In-app delivery: %d notifications stored", len(store.notifications))
}

// mockInAppStore implements InAppNotificationStore for testing.
type mockInAppStore struct {
	notifications map[string][]*InAppNotification
	mu            sync.Mutex
}

func (m *mockInAppStore) CreateNotification(ctx context.Context, notification *InAppNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications[notification.UserID] = append(m.notifications[notification.UserID], notification)
	return nil
}
