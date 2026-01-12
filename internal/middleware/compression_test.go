// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompression_WithGzipAccept(t *testing.T) {
	// Create a test handler that returns a large response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write more than 1KB to trigger compression
		data := strings.Repeat("test data ", 200) // ~2KB
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(data))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap with compression middleware
	compressedHandler := Compression(handler)

	// Create test request with gzip accept header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	// Execute request
	compressedHandler(rec, req)

	// Verify Content-Encoding header is set
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Expected Content-Encoding: gzip, got: %s", rec.Header().Get("Content-Encoding"))
	}

	// Verify Content-Length header is removed
	if rec.Header().Get("Content-Length") != "" {
		t.Error("Expected Content-Length header to be removed")
	}

	// Verify response is actually gzipped
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read decompressed data: %v", err)
	}

	expected := strings.Repeat("test data ", 200)
	if string(decompressed) != expected {
		t.Error("Decompressed data doesn't match expected")
	}
}

func TestCompression_WithoutGzipAccept(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("uncompressed response"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap with compression middleware
	compressedHandler := Compression(handler)

	// Create test request WITHOUT gzip accept header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	compressedHandler(rec, req)

	// Verify Content-Encoding header is NOT set
	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Expected Content-Encoding to not be gzip when client doesn't accept it")
	}

	// Verify response is not compressed
	if rec.Body.String() != "uncompressed response" {
		t.Errorf("Expected uncompressed response, got: %s", rec.Body.String())
	}
}

func TestCompression_WebSocketConnection(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("websocket upgrade"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap with compression middleware
	compressedHandler := Compression(handler)

	// Create test request with WebSocket upgrade header
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Upgrade", "websocket")
	rec := httptest.NewRecorder()

	// Execute request
	compressedHandler(rec, req)

	// Verify Content-Encoding header is NOT set for WebSocket
	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Expected WebSocket connections to not be compressed")
	}

	// Verify response is not compressed
	if rec.Body.String() != "websocket upgrade" {
		t.Errorf("Expected uncompressed WebSocket response, got: %s", rec.Body.String())
	}
}

func TestCompression_PartialGzipAccept(t *testing.T) {
	// Test with Accept-Encoding that includes gzip among other encodings
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := strings.Repeat("data", 500) // ~2KB
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(data))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	compressedHandler := Compression(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "deflate, gzip, br")
	rec := httptest.NewRecorder()

	compressedHandler(rec, req)

	// Should still compress because gzip is in the list
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected gzip compression when Accept-Encoding includes gzip")
	}
}

func TestGzipResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	gz := gzip.NewWriter(rec)
	defer gz.Close()

	gzw := &gzipResponseWriter{
		Writer:         gz,
		ResponseWriter: rec,
		wroteHeader:    false,
	}

	// Test WriteHeader
	gzw.WriteHeader(http.StatusCreated)

	if !gzw.wroteHeader {
		t.Error("Expected wroteHeader to be true after WriteHeader")
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", rec.Code)
	}
}

func TestGzipResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	gz := gzip.NewWriter(rec)
	defer gz.Close()

	gzw := &gzipResponseWriter{
		Writer:         gz,
		ResponseWriter: rec,
		wroteHeader:    false,
	}

	// Write should set default status if not already set
	data := []byte("test data")
	n, err := gzw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	if !gzw.wroteHeader {
		t.Error("Expected wroteHeader to be true after Write")
	}
}

func TestCompression_EmptyResponse(t *testing.T) {
	// Test compression with empty response body
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	compressedHandler := Compression(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	compressedHandler(rec, req)

	// Even with empty body, gzip headers should be set
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected Content-Encoding: gzip even for empty response")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status code 204, got %d", rec.Code)
	}
}

func TestCompression_SmallResponse(t *testing.T) {
	// Test with small response (< 1KB)
	// Note: According to the comment, it should only compress > 1KB,
	// but the implementation doesn't actually enforce this
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("small"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	compressedHandler := Compression(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	compressedHandler(rec, req)

	// Currently the implementation compresses all responses
	// regardless of size when gzip is accepted
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected gzip compression for small response")
	}
}

func BenchmarkCompression(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := strings.Repeat("benchmark data ", 100)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(data))
	})

	compressedHandler := Compression(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		compressedHandler(rec, req)
	}
}

func BenchmarkCompressionWithoutGzip(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := strings.Repeat("benchmark data ", 100)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(data))
	})

	compressedHandler := Compression(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Accept-Encoding header

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		compressedHandler(rec, req)
	}
}
