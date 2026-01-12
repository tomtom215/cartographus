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

func TestNewPerformanceMonitor(t *testing.T) {
	tests := []struct {
		name       string
		maxMetrics int
	}{
		{"small capacity", 10},
		{"medium capacity", 100},
		{"large capacity", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPerformanceMonitor(tt.maxMetrics)

			if pm == nil {
				t.Fatal("NewPerformanceMonitor returned nil")
			}

			if pm.maxMetrics != tt.maxMetrics {
				t.Errorf("Expected maxMetrics %d, got %d", tt.maxMetrics, pm.maxMetrics)
			}

			if pm.metrics == nil {
				t.Error("Expected metrics slice to be initialized")
			}

			if pm.requestCounts == nil {
				t.Error("Expected requestCounts map to be initialized")
			}

			if pm.totalDuration == nil {
				t.Error("Expected totalDuration map to be initialized")
			}
		})
	}
}

func TestPerformanceMonitor_RecordRequest(t *testing.T) {
	pm := NewPerformanceMonitor(10)

	metric := RequestMetrics{
		Path:       "/api/test",
		Method:     "GET",
		DurationMS: 50,
		StatusCode: 200,
		Timestamp:  time.Now(),
		CacheHit:   false,
		QueryCount: 1,
	}

	pm.RecordRequest(&metric)

	// Verify metric was added
	if len(pm.metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(pm.metrics))
	}

	// Verify request count was incremented
	key := "GET /api/test"
	if pm.requestCounts[key] != 1 {
		t.Errorf("Expected request count 1, got %d", pm.requestCounts[key])
	}

	// Verify total duration was recorded
	if pm.totalDuration[key] != 50 {
		t.Errorf("Expected total duration 50, got %d", pm.totalDuration[key])
	}
}

func TestPerformanceMonitor_RecordRequest_SlidingWindow(t *testing.T) {
	pm := NewPerformanceMonitor(5) // Small window for testing

	// Add more metrics than the window size
	for i := 0; i < 10; i++ {
		metric := RequestMetrics{
			Path:       "/api/test",
			Method:     "GET",
			DurationMS: int64(i * 10),
			StatusCode: 200,
			Timestamp:  time.Now(),
		}
		pm.RecordRequest(&metric)
	}

	// Verify sliding window keeps only last 5 metrics
	if len(pm.metrics) != 5 {
		t.Errorf("Expected 5 metrics (sliding window), got %d", len(pm.metrics))
	}

	// Verify request counts accumulate beyond window
	key := "GET /api/test"
	if pm.requestCounts[key] != 10 {
		t.Errorf("Expected request count 10, got %d", pm.requestCounts[key])
	}

	// Verify total duration accumulates
	expectedTotal := int64(0 + 10 + 20 + 30 + 40 + 50 + 60 + 70 + 80 + 90)
	if pm.totalDuration[key] != expectedTotal {
		t.Errorf("Expected total duration %d, got %d", expectedTotal, pm.totalDuration[key])
	}
}

func TestPerformanceMonitor_GetStats(t *testing.T) {
	pm := NewPerformanceMonitor(100)

	// Add multiple requests to the same endpoint
	for i := 0; i < 10; i++ {
		metric := RequestMetrics{
			Path:       "/api/users",
			Method:     "GET",
			DurationMS: int64(100 + i*10), // 100, 110, 120, ..., 190
			StatusCode: 200,
			Timestamp:  time.Now(),
		}
		pm.RecordRequest(&metric)
	}

	// Add requests to another endpoint
	for i := 0; i < 5; i++ {
		metric := RequestMetrics{
			Path:       "/api/posts",
			Method:     "GET",
			DurationMS: int64(50 + i*5), // 50, 55, 60, 65, 70
			StatusCode: 200,
			Timestamp:  time.Now(),
		}
		pm.RecordRequest(&metric)
	}

	stats := pm.GetStats()

	// Verify we got stats for 2 endpoints
	if len(stats) != 2 {
		t.Fatalf("Expected 2 endpoint stats, got %d", len(stats))
	}

	// Verify stats are sorted by request count (descending)
	// /api/users should come first (10 requests vs 5)
	if stats[0].RequestCount != 10 {
		t.Errorf("Expected first endpoint to have 10 requests, got %d", stats[0].RequestCount)
	}

	// Find the users endpoint stats
	var usersStats *EndpointStats
	for i := range stats {
		if stats[i].Path == "GET /api/users" {
			usersStats = &stats[i]
			break
		}
	}

	if usersStats == nil {
		t.Fatal("Expected to find stats for GET /api/users")
	}

	// Verify statistics calculations
	if usersStats.RequestCount != 10 {
		t.Errorf("Expected request count 10, got %d", usersStats.RequestCount)
	}

	// Average should be 145 ((100+110+120+130+140+150+160+170+180+190)/10)
	expectedAvg := 145.0
	if usersStats.AvgDuration != expectedAvg {
		t.Errorf("Expected average duration %.2f, got %.2f", expectedAvg, usersStats.AvgDuration)
	}

	// Min should be 100
	if usersStats.MinDuration != 100 {
		t.Errorf("Expected min duration 100, got %d", usersStats.MinDuration)
	}

	// Max should be 190
	if usersStats.MaxDuration != 190 {
		t.Errorf("Expected max duration 190, got %d", usersStats.MaxDuration)
	}

	// P50 should be around 145 (median of 10 values)
	if usersStats.P50Duration < 140 || usersStats.P50Duration > 150 {
		t.Errorf("Expected P50 around 145, got %d", usersStats.P50Duration)
	}
}

func TestPerformanceMonitor_GetRecentMetrics(t *testing.T) {
	pm := NewPerformanceMonitor(100)

	// Add 10 metrics
	for i := 0; i < 10; i++ {
		metric := RequestMetrics{
			Path:       "/api/test",
			Method:     "GET",
			DurationMS: int64(i),
			StatusCode: 200,
			Timestamp:  time.Now(),
		}
		pm.RecordRequest(&metric)
	}

	// Get recent 5 metrics
	recent := pm.GetRecentMetrics(5)

	if len(recent) != 5 {
		t.Errorf("Expected 5 recent metrics, got %d", len(recent))
	}

	// Verify we got the last 5 (durations 5, 6, 7, 8, 9)
	for i, metric := range recent {
		expectedDuration := int64(5 + i)
		if metric.DurationMS != expectedDuration {
			t.Errorf("Expected duration %d, got %d", expectedDuration, metric.DurationMS)
		}
	}
}

func TestPerformanceMonitor_GetRecentMetrics_MoreThanAvailable(t *testing.T) {
	pm := NewPerformanceMonitor(100)

	// Add only 3 metrics
	for i := 0; i < 3; i++ {
		metric := RequestMetrics{
			Path:       "/api/test",
			Method:     "GET",
			DurationMS: int64(i),
			StatusCode: 200,
			Timestamp:  time.Now(),
		}
		pm.RecordRequest(&metric)
	}

	// Request 10 metrics (more than available)
	recent := pm.GetRecentMetrics(10)

	// Should return all 3 available metrics
	if len(recent) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(recent))
	}
}

func TestPerformanceMonitor_Middleware(t *testing.T) {
	pm := NewPerformanceMonitor(100)

	// Create a handler that takes some time
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap with performance middleware
	wrappedHandler := pm.Middleware(handler)

	// Make a request
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	// Verify metric was recorded
	if len(pm.metrics) != 1 {
		t.Errorf("Expected 1 metric to be recorded, got %d", len(pm.metrics))
	}

	metric := pm.metrics[0]

	// Verify metric details
	if metric.Path != "/api/test" {
		t.Errorf("Expected path /api/test, got %s", metric.Path)
	}

	if metric.Method != "GET" {
		t.Errorf("Expected method GET, got %s", metric.Method)
	}

	if metric.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", metric.StatusCode)
	}

	// Duration should be at least 10ms
	if metric.DurationMS < 10 {
		t.Errorf("Expected duration >= 10ms, got %dms", metric.DurationMS)
	}
}

func TestPerformanceMonitor_Middleware_CapturesStatusCode(t *testing.T) {
	pm := NewPerformanceMonitor(100)

	tests := []struct {
		name       string
		statusCode int
	}{
		{"OK", http.StatusOK},
		{"Created", http.StatusCreated},
		{"BadRequest", http.StatusBadRequest},
		{"NotFound", http.StatusNotFound},
		{"InternalError", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous metrics
			pm.metrics = []RequestMetrics{}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			wrappedHandler := pm.Middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			if len(pm.metrics) != 1 {
				t.Fatalf("Expected 1 metric, got %d", len(pm.metrics))
			}

			if pm.metrics[0].StatusCode != tt.statusCode {
				t.Errorf("Expected status code %d, got %d", tt.statusCode, pm.metrics[0].StatusCode)
			}
		})
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	rw.WriteHeader(http.StatusCreated)

	if rw.statusCode != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", rw.statusCode)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected recorder code 201, got %d", rec.Code)
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		data   []int64
		p      float64
		expect int64
	}{
		{
			name:   "P50 of odd number of elements",
			data:   []int64{10, 20, 30, 40, 50},
			p:      0.50,
			expect: 30,
		},
		{
			name:   "P95 of dataset",
			data:   []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			p:      0.95,
			expect: 9, // index = int(float64(10-1) * 0.95) = 8, so data[8] = 9
		},
		{
			name:   "P99 of dataset",
			data:   []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			p:      0.99,
			expect: 9, // index = int(float64(10-1) * 0.99) = 8, so data[8] = 9
		},
		{
			name:   "P0 (minimum)",
			data:   []int64{10, 20, 30, 40, 50},
			p:      0.0,
			expect: 10,
		},
		{
			name:   "P100 (maximum)",
			data:   []int64{10, 20, 30, 40, 50},
			p:      1.0,
			expect: 50,
		},
		{
			name:   "single element",
			data:   []int64{42},
			p:      0.5,
			expect: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := percentile(tt.data, tt.p)
			if result != tt.expect {
				t.Errorf("Expected %d, got %d", tt.expect, result)
			}
		})
	}
}

func TestPercentile_EmptySlice(t *testing.T) {
	result := percentile([]int64{}, 0.5)
	if result != 0 {
		t.Errorf("Expected 0 for empty slice, got %d", result)
	}
}

func TestPerformanceMonitor_ConcurrentAccess(t *testing.T) {
	pm := NewPerformanceMonitor(1000)

	done := make(chan bool)

	// Spawn multiple goroutines recording metrics
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				metric := RequestMetrics{
					Path:       "/api/test",
					Method:     "GET",
					DurationMS: int64(j),
					StatusCode: 200,
					Timestamp:  time.Now(),
				}
				pm.RecordRequest(&metric)
			}
			done <- true
		}(i)
	}

	// Spawn goroutines reading stats
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				pm.GetStats()
				pm.GetRecentMetrics(10)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	// If we get here without deadlock or panic, the test passes
	stats := pm.GetStats()
	if len(stats) == 0 {
		t.Error("Expected stats to be recorded")
	}
}

func BenchmarkPerformanceMonitor_RecordRequest(b *testing.B) {
	pm := NewPerformanceMonitor(10000)

	metric := RequestMetrics{
		Path:       "/api/test",
		Method:     "GET",
		DurationMS: 50,
		StatusCode: 200,
		Timestamp:  time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.RecordRequest(&metric)
	}
}

func BenchmarkPerformanceMonitor_GetStats(b *testing.B) {
	pm := NewPerformanceMonitor(10000)

	// Pre-populate with metrics
	for i := 0; i < 1000; i++ {
		metric := RequestMetrics{
			Path:       "/api/test",
			Method:     "GET",
			DurationMS: int64(i),
			StatusCode: 200,
			Timestamp:  time.Now(),
		}
		pm.RecordRequest(&metric)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.GetStats()
	}
}
