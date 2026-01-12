// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package middleware

import (
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// RequestMetrics tracks performance metrics for API requests
type RequestMetrics struct {
	Path       string
	Method     string
	DurationMS int64
	StatusCode int
	Timestamp  time.Time
	CacheHit   bool
	QueryCount int
}

// PerformanceMonitor tracks API performance metrics
type PerformanceMonitor struct {
	mu            sync.RWMutex
	metrics       []RequestMetrics
	maxMetrics    int
	requestCounts map[string]int64
	totalDuration map[string]int64
}

// EndpointStats contains aggregated statistics for an endpoint
type EndpointStats struct {
	Path         string
	RequestCount int64
	AvgDuration  float64
	P50Duration  int64
	P95Duration  int64
	P99Duration  int64
	MinDuration  int64
	MaxDuration  int64
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(maxMetrics int) *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics:       make([]RequestMetrics, 0, maxMetrics),
		maxMetrics:    maxMetrics,
		requestCounts: make(map[string]int64),
		totalDuration: make(map[string]int64),
	}
}

// RecordRequest adds a request metric
func (pm *PerformanceMonitor) RecordRequest(metric *RequestMetrics) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Add metric to sliding window
	pm.metrics = append(pm.metrics, *metric)
	if len(pm.metrics) > pm.maxMetrics {
		pm.metrics = pm.metrics[1:]
	}

	// Update aggregate stats
	key := metric.Method + " " + metric.Path
	pm.requestCounts[key]++
	pm.totalDuration[key] += metric.DurationMS
}

// GetStats returns aggregated statistics for all endpoints
func (pm *PerformanceMonitor) GetStats() []EndpointStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Group metrics by endpoint
	endpointMetrics := make(map[string][]int64)
	for _, m := range pm.metrics {
		key := m.Method + " " + m.Path
		endpointMetrics[key] = append(endpointMetrics[key], m.DurationMS)
	}

	// Calculate statistics for each endpoint
	stats := make([]EndpointStats, 0, len(endpointMetrics))
	for endpoint, durations := range endpointMetrics {
		if len(durations) == 0 {
			continue
		}

		// Sort durations for percentile calculations
		sorted := make([]int64, len(durations))
		copy(sorted, durations)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

		// Calculate statistics
		var sum int64
		for _, d := range sorted {
			sum += d
		}

		stat := EndpointStats{
			Path:         endpoint,
			RequestCount: int64(len(sorted)),
			AvgDuration:  float64(sum) / float64(len(sorted)),
			P50Duration:  percentile(sorted, 0.50),
			P95Duration:  percentile(sorted, 0.95),
			P99Duration:  percentile(sorted, 0.99),
			MinDuration:  sorted[0],
			MaxDuration:  sorted[len(sorted)-1],
		}

		stats = append(stats, stat)
	}

	// Sort by request count descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].RequestCount > stats[j].RequestCount
	})

	return stats
}

// GetRecentMetrics returns the most recent N metrics
func (pm *PerformanceMonitor) GetRecentMetrics(n int) []RequestMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if n > len(pm.metrics) {
		n = len(pm.metrics)
	}

	recent := make([]RequestMetrics, n)
	copy(recent, pm.metrics[len(pm.metrics)-n:])
	return recent
}

// LogSlowRequests logs requests that exceed the threshold
func (pm *PerformanceMonitor) LogSlowRequests(thresholdMS int64) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, m := range pm.metrics {
		if m.DurationMS > thresholdMS {
			logging.Warn().
				Str("method", m.Method).
				Str("path", m.Path).
				Int64("duration_ms", m.DurationMS).
				Int64("threshold_ms", thresholdMS).
				Msg("Slow request detected")
		}
	}
}

// Middleware creates an HTTP middleware for performance monitoring
func (pm *PerformanceMonitor) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapper := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start).Milliseconds()

		// Record metric
		pm.RecordRequest(&RequestMetrics{
			Path:       r.URL.Path,
			Method:     r.Method,
			DurationMS: duration,
			StatusCode: wrapper.statusCode,
			Timestamp:  time.Now(),
		})

		// Log slow requests (>1000ms)
		if duration > 1000 {
			logging.Warn().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int64("duration_ms", duration).
				Msg("Slow request detected")
		}
	})
}

// percentile calculates the percentile value from a sorted slice
func percentile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
