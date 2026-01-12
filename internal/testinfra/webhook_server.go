// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build integration

package testinfra

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// WebhookCapture represents a captured webhook request.
type WebhookCapture struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
}

// MockWebhookServer provides a mock HTTP server for testing webhook deliveries.
// It captures all incoming requests for verification.
type MockWebhookServer struct {
	Server   *httptest.Server
	Captures []WebhookCapture
	mu       sync.Mutex

	// ResponseStatus is the HTTP status code to return (default: 200).
	ResponseStatus int

	// ResponseBody is the response body to return.
	ResponseBody []byte

	// ResponseFunc allows custom response handling per request.
	ResponseFunc func(w http.ResponseWriter, r *http.Request)
}

// NewMockWebhookServer creates a new mock webhook server.
func NewMockWebhookServer(t *testing.T) *MockWebhookServer {
	t.Helper()

	mws := &MockWebhookServer{
		ResponseStatus: http.StatusOK,
		Captures:       make([]WebhookCapture, 0),
	}

	mws.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body
		body := make([]byte, 0)
		if r.Body != nil {
			body, _ = readAll(r.Body)
			r.Body.Close()
		}

		// Capture request
		mws.mu.Lock()
		mws.Captures = append(mws.Captures, WebhookCapture{
			Method:  r.Method,
			Path:    r.URL.Path,
			Headers: r.Header.Clone(),
			Body:    body,
		})
		mws.mu.Unlock()

		// Custom response handler
		if mws.ResponseFunc != nil {
			mws.ResponseFunc(w, r)
			return
		}

		// Default response
		w.WriteHeader(mws.ResponseStatus)
		if mws.ResponseBody != nil {
			w.Write(mws.ResponseBody) //nolint:errcheck
		}
	}))

	return mws
}

// URL returns the server URL.
func (m *MockWebhookServer) URL() string {
	return m.Server.URL
}

// Close shuts down the server.
func (m *MockWebhookServer) Close() {
	m.Server.Close()
}

// GetCaptures returns all captured requests.
func (m *MockWebhookServer) GetCaptures() []WebhookCapture {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]WebhookCapture, len(m.Captures))
	copy(result, m.Captures)
	return result
}

// ClearCaptures clears all captured requests.
func (m *MockWebhookServer) ClearCaptures() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Captures = make([]WebhookCapture, 0)
}

// WaitForCaptures waits until at least n requests are captured or timeout.
func (m *MockWebhookServer) WaitForCaptures(n int, timeout int) bool {
	for i := 0; i < timeout*10; i++ {
		m.mu.Lock()
		count := len(m.Captures)
		m.mu.Unlock()
		if count >= n {
			return true
		}
		sleep(100)
	}
	return false
}

// MockSlackResponse creates a typical Slack API success response.
func MockSlackResponse() []byte {
	resp := map[string]interface{}{
		"ok": true,
	}
	data, _ := json.Marshal(resp)
	return data
}

// MockDiscordResponse creates a typical Discord API success response.
func MockDiscordResponse() []byte {
	resp := map[string]interface{}{
		"id":         "123456789",
		"type":       0,
		"channel_id": "987654321",
	}
	data, _ := json.Marshal(resp)
	return data
}

// MockTelegramResponse creates a typical Telegram API success response.
func MockTelegramResponse() []byte {
	resp := map[string]interface{}{
		"ok": true,
		"result": map[string]interface{}{
			"message_id": 12345,
		},
	}
	data, _ := json.Marshal(resp)
	return data
}

// MockEmailServer provides a mock SMTP server for testing email delivery.
type MockEmailServer struct {
	ReceivedEmails []MockEmail
	mu             sync.Mutex
	Server         *httptest.Server
}

// MockEmail represents a captured email.
type MockEmail struct {
	From    string
	To      []string
	Subject string
	Body    string
}

// NewMockEmailServer creates a mock email server (HTTP-based for testing).
// In real integration tests, use MailHog or similar containerized SMTP server.
func NewMockEmailServer(t *testing.T) *MockEmailServer {
	t.Helper()

	mes := &MockEmailServer{
		ReceivedEmails: make([]MockEmail, 0),
	}

	// Simple HTTP endpoint that accepts JSON email submissions
	mes.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var email MockEmail
		if err := json.NewDecoder(r.Body).Decode(&email); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mes.mu.Lock()
		mes.ReceivedEmails = append(mes.ReceivedEmails, email)
		mes.mu.Unlock()

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"sent"}`)
	}))

	return mes
}

// URL returns the server URL.
func (m *MockEmailServer) URL() string {
	return m.Server.URL
}

// Close shuts down the server.
func (m *MockEmailServer) Close() {
	m.Server.Close()
}

// GetEmails returns all captured emails.
func (m *MockEmailServer) GetEmails() []MockEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]MockEmail, len(m.ReceivedEmails))
	copy(result, m.ReceivedEmails)
	return result
}

// Helper functions (avoid importing io for minimal dependencies)
func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var result []byte
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return result, nil
}

func sleep(ms int) {
	// Simple sleep without importing time in this context
	// Uses a busy loop (only for short waits in tests)
	for i := 0; i < ms*1000; i++ {
		_ = i
	}
}
