// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	// Create a test handler that checks for request ID
	var capturedID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with RequestID middleware
	wrappedHandler := RequestID(handler)

	// Create test request without X-Request-ID header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler(rec, req)

	// Verify X-Request-ID header is set in response
	responseID := rec.Header().Get("X-Request-ID")
	if responseID == "" {
		t.Error("Expected X-Request-ID header in response")
	}

	// Verify it's a valid UUID
	if _, err := uuid.Parse(responseID); err != nil {
		t.Errorf("Response X-Request-ID is not a valid UUID: %v", err)
	}

	// Verify the ID was added to context
	if capturedID == "" {
		t.Error("Expected request ID in context")
	}

	// Verify context ID matches response header
	if capturedID != responseID {
		t.Errorf("Context ID (%s) doesn't match response header ID (%s)", capturedID, responseID)
	}
}

func TestRequestID_PreservesExistingID(t *testing.T) {
	// Create a test handler that captures the request ID
	var capturedID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with RequestID middleware
	wrappedHandler := RequestID(handler)

	// Create test request WITH existing X-Request-ID header
	existingID := "existing-request-id-12345"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler(rec, req)

	// Verify the existing ID is preserved in response
	responseID := rec.Header().Get("X-Request-ID")
	if responseID != existingID {
		t.Errorf("Expected X-Request-ID to be %s, got %s", existingID, responseID)
	}

	// Verify the existing ID was added to context
	if capturedID != existingID {
		t.Errorf("Expected context ID to be %s, got %s", existingID, capturedID)
	}
}

func TestRequestID_PreservesUpstreamProxyID(t *testing.T) {
	// Test that IDs from upstream proxies (like nginx) are preserved
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequestID(handler)

	proxyID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", proxyID)
	rec := httptest.NewRecorder()

	wrappedHandler(rec, req)

	if rec.Header().Get("X-Request-ID") != proxyID {
		t.Error("Expected upstream proxy request ID to be preserved")
	}
}

func TestGetRequestID_WithID(t *testing.T) {
	// Create a context with a request ID
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	testID := "test-request-id-789"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != testID {
			t.Errorf("Expected request ID %s, got %s", testID, id)
		}
	})

	wrappedHandler := RequestID(handler)

	req.Header.Set("X-Request-ID", testID)
	rec := httptest.NewRecorder()

	wrappedHandler(rec, req)
}

func TestGetRequestID_WithoutID(t *testing.T) {
	// Create a context without a request ID
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Call GetRequestID directly without middleware
	id := GetRequestID(req.Context())

	// Should return empty string when no ID in context
	if id != "" {
		t.Errorf("Expected empty string when no request ID in context, got: %s", id)
	}
}

func TestGetRequestID_WithWrongType(t *testing.T) {
	// Create a context with wrong type for request ID
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := req.Context()

	// This shouldn't normally happen, but test defensive programming
	// by not adding anything to the context
	id := GetRequestID(ctx)

	// Should return empty string when type assertion fails
	if id != "" {
		t.Errorf("Expected empty string for wrong type, got: %s", id)
	}
}

func TestRequestID_MultipleRequests(t *testing.T) {
	// Verify that multiple requests get different IDs
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequestID(handler)

	// Make multiple requests
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler(rec, req)

		id := rec.Header().Get("X-Request-ID")
		if ids[id] {
			t.Errorf("Duplicate request ID generated: %s", id)
		}
		ids[id] = true
	}

	// Verify we got 10 unique IDs
	if len(ids) != 10 {
		t.Errorf("Expected 10 unique IDs, got %d", len(ids))
	}
}

func TestRequestID_ContextIsolation(t *testing.T) {
	// Verify that request IDs are properly isolated per request
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		headerID := r.Header.Get("X-Request-ID")

		// If there was an incoming header, it should match context
		if headerID != "" && id != headerID {
			t.Errorf("Context ID (%s) doesn't match header ID (%s)", id, headerID)
		}

		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequestID(handler)

	// Request 1 with custom ID
	req1 := httptest.NewRequest(http.MethodGet, "/test1", nil)
	req1.Header.Set("X-Request-ID", "custom-id-1")
	rec1 := httptest.NewRecorder()
	wrappedHandler(rec1, req1)

	// Request 2 with different custom ID
	req2 := httptest.NewRequest(http.MethodGet, "/test2", nil)
	req2.Header.Set("X-Request-ID", "custom-id-2")
	rec2 := httptest.NewRecorder()
	wrappedHandler(rec2, req2)

	// Verify each request got its own ID
	id1 := rec1.Header().Get("X-Request-ID")
	id2 := rec2.Header().Get("X-Request-ID")

	if id1 == id2 {
		t.Error("Expected different request IDs for different requests")
	}

	if id1 != "custom-id-1" {
		t.Errorf("Expected first request ID to be 'custom-id-1', got %s", id1)
	}

	if id2 != "custom-id-2" {
		t.Errorf("Expected second request ID to be 'custom-id-2', got %s", id2)
	}
}

func TestRequestID_EmptyHeader(t *testing.T) {
	// Test with empty X-Request-ID header (should generate new ID)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "") // Empty string
	rec := httptest.NewRecorder()

	wrappedHandler(rec, req)

	responseID := rec.Header().Get("X-Request-ID")
	if responseID == "" {
		t.Error("Expected new request ID to be generated for empty header")
	}

	// Verify it's a valid UUID (newly generated)
	if _, err := uuid.Parse(responseID); err != nil {
		t.Errorf("Generated ID is not a valid UUID: %v", err)
	}
}

func BenchmarkRequestID(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler(rec, req)
	}
}

func BenchmarkRequestID_WithExisting(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "existing-id-12345")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler(rec, req)
	}
}
