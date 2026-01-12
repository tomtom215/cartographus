// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPrometheusMetrics(t *testing.T) {
	t.Parallel()

	t.Run("records metrics for successful request", func(t *testing.T) {
		t.Parallel()
		handler := PrometheusMetrics(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("records metrics for error response", func(t *testing.T) {
		t.Parallel()
		handler := PrometheusMetrics(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		})

		req := httptest.NewRequest("POST", "/api/v1/test", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rec.Code)
		}
	})

	t.Run("records metrics for various HTTP methods", func(t *testing.T) {
		t.Parallel()
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				handler := PrometheusMetrics(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				req := httptest.NewRequest(method, "/api/v1/test", nil)
				rec := httptest.NewRecorder()

				handler(rec, req)

				if rec.Code != http.StatusOK {
					t.Errorf("Expected status 200 for %s, got %d", method, rec.Code)
				}
			})
		}
	})

	t.Run("records metrics for various status codes", func(t *testing.T) {
		t.Parallel()
		statusCodes := []int{
			http.StatusOK,
			http.StatusCreated,
			http.StatusNoContent,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusInternalServerError,
			http.StatusServiceUnavailable,
		}

		for _, code := range statusCodes {
			t.Run(http.StatusText(code), func(t *testing.T) {
				handler := PrometheusMetrics(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(code)
				})

				req := httptest.NewRequest("GET", "/api/v1/test", nil)
				rec := httptest.NewRecorder()

				handler(rec, req)

				if rec.Code != code {
					t.Errorf("Expected status %d, got %d", code, rec.Code)
				}
			})
		}
	})

	t.Run("defaults to 200 when WriteHeader not called", func(t *testing.T) {
		t.Parallel()
		handler := PrometheusMetrics(func(w http.ResponseWriter, r *http.Request) {
			// Just write body without explicit WriteHeader
			w.Write([]byte("Hello"))
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected default status 200, got %d", rec.Code)
		}
	})

	t.Run("measures request duration", func(t *testing.T) {
		t.Parallel()
		var duration time.Duration

		handler := PrometheusMetrics(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		start := time.Now()
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)
		duration = time.Since(start)

		if duration < 10*time.Millisecond {
			t.Errorf("Expected duration >= 10ms, got %v", duration)
		}
	})

	t.Run("handles various URL paths", func(t *testing.T) {
		t.Parallel()
		paths := []string{
			"/api/v1/stats",
			"/api/v1/playbacks",
			"/api/v1/locations",
			"/api/v1/analytics/geographic",
			"/health",
			"/metrics",
		}

		for _, path := range paths {
			t.Run(path, func(t *testing.T) {
				handler := PrometheusMetrics(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				req := httptest.NewRequest("GET", path, nil)
				rec := httptest.NewRecorder()

				handler(rec, req)

				if rec.Code != http.StatusOK {
					t.Errorf("Expected status 200 for path %s, got %d", path, rec.Code)
				}
			})
		}
	})

	t.Run("tracks active requests", func(t *testing.T) {
		t.Parallel()
		started := make(chan struct{})
		done := make(chan struct{})

		handler := PrometheusMetrics(func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-done // Wait until test says to finish
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		rec := httptest.NewRecorder()

		go func() {
			handler(rec, req)
		}()

		<-started   // Wait for handler to start
		close(done) // Allow handler to finish

		// Small wait for goroutine to complete
		time.Sleep(10 * time.Millisecond)
	})
}

func TestMetricsResponseWriter(t *testing.T) {
	t.Parallel()

	t.Run("captures status code", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		wrapper := &metricsResponseWriter{
			ResponseWriter: rec,
			statusCode:     http.StatusOK,
		}

		wrapper.WriteHeader(http.StatusNotFound)

		if wrapper.statusCode != http.StatusNotFound {
			t.Errorf("Expected status code 404, got %d", wrapper.statusCode)
		}
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected underlying recorder status 404, got %d", rec.Code)
		}
	})

	t.Run("WriteHeader sets status code", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		wrapper := &metricsResponseWriter{
			ResponseWriter: rec,
			statusCode:     http.StatusOK, // Default
		}

		wrapper.WriteHeader(http.StatusCreated)

		if wrapper.statusCode != http.StatusCreated {
			t.Errorf("Expected status code 201, got %d", wrapper.statusCode)
		}
	})

	t.Run("preserves ResponseWriter functionality", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		wrapper := &metricsResponseWriter{
			ResponseWriter: rec,
		}

		// Test Header
		wrapper.Header().Set("Content-Type", "application/json")
		if wrapper.Header().Get("Content-Type") != "application/json" {
			t.Error("Header should be preserved")
		}

		// Test Write
		n, err := wrapper.Write([]byte("test body"))
		if err != nil {
			t.Errorf("Write error: %v", err)
		}
		if n != 9 {
			t.Errorf("Expected 9 bytes written, got %d", n)
		}

		// Verify body was written to underlying recorder
		if rec.Body.String() != "test body" {
			t.Errorf("Body not written: %s", rec.Body.String())
		}
	})

	t.Run("default status code is 200", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		wrapper := &metricsResponseWriter{
			ResponseWriter: rec,
			statusCode:     http.StatusOK,
		}

		// Don't call WriteHeader, just write
		wrapper.Write([]byte("test"))

		// Status code should still be the default
		if wrapper.statusCode != http.StatusOK {
			t.Errorf("Expected default status 200, got %d", wrapper.statusCode)
		}
	})
}

// Benchmark tests
func BenchmarkPrometheusMetrics(b *testing.B) {
	handler := PrometheusMetrics(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler(rec, req)
	}
}

func BenchmarkMetricsResponseWriter_WriteHeader(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrapper := &metricsResponseWriter{
			ResponseWriter: rec,
			statusCode:     http.StatusOK,
		}
		wrapper.WriteHeader(http.StatusOK)
	}
}

func BenchmarkMetricsResponseWriter_Write(b *testing.B) {
	data := []byte("Hello, World!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrapper := &metricsResponseWriter{
			ResponseWriter: rec,
		}
		wrapper.Write(data)
	}
}
