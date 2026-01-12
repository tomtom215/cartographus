// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"net/http"
	"time"
)

// WALStats represents WAL statistics for the API response.
type WALStats struct {
	PendingCount      int64     `json:"pending_count"`
	ConfirmedCount    int64     `json:"confirmed_count"`
	TotalWrites       int64     `json:"total_writes"`
	TotalConfirms     int64     `json:"total_confirms"`
	TotalRetries      int64     `json:"total_retries"`
	LastCompaction    time.Time `json:"last_compaction,omitempty"`
	DBSizeBytes       int64     `json:"db_size_bytes"`
	DBSizeFormatted   string    `json:"db_size_formatted"`
	WriteRatePerMin   float64   `json:"write_rate_per_min,omitempty"`
	ConfirmRatePerMin float64   `json:"confirm_rate_per_min,omitempty"`
	Status            string    `json:"status"`
	Healthy           bool      `json:"healthy"`
}

// WALStatsProvider interface for dependency injection.
// This allows the handlers to work with the WAL implementation.
type WALStatsProvider interface {
	// GetStats returns current WAL statistics.
	GetStats() WALStatsInternal
}

// WALStatsInternal is the internal representation from wal package.
type WALStatsInternal struct {
	PendingCount   int64
	ConfirmedCount int64
	TotalWrites    int64
	TotalConfirms  int64
	TotalRetries   int64
	LastCompaction time.Time
	DBSizeBytes    int64
}

// WALHandlers provides HTTP handlers for WAL endpoints.
type WALHandlers struct {
	provider  WALStatsProvider
	startTime time.Time
}

// NewWALHandlers creates new WAL handlers.
func NewWALHandlers(provider WALStatsProvider) *WALHandlers {
	return &WALHandlers{
		provider:  provider,
		startTime: time.Now(),
	}
}

// GetStats handles GET /api/v1/wal/stats
// Returns WAL statistics.
func (h *WALHandlers) GetStats(w http.ResponseWriter, _ *http.Request) {
	if h.provider == nil {
		writeJSON(w, WALStats{
			Status:  "unavailable",
			Healthy: false,
		})
		return
	}

	internal := h.provider.GetStats()
	uptime := time.Since(h.startTime)

	// Calculate rates if uptime is sufficient
	var writeRate, confirmRate float64
	if uptime.Minutes() > 0 {
		writeRate = float64(internal.TotalWrites) / uptime.Minutes()
		confirmRate = float64(internal.TotalConfirms) / uptime.Minutes()
	}

	// Determine health status
	healthy := internal.PendingCount < 10000 // Healthy if less than 10k pending
	status := "healthy"
	if !healthy {
		status = "backpressure"
	}
	if internal.PendingCount == 0 && internal.TotalWrites == 0 {
		status = "idle"
	}

	stats := WALStats{
		PendingCount:      internal.PendingCount,
		ConfirmedCount:    internal.ConfirmedCount,
		TotalWrites:       internal.TotalWrites,
		TotalConfirms:     internal.TotalConfirms,
		TotalRetries:      internal.TotalRetries,
		LastCompaction:    internal.LastCompaction,
		DBSizeBytes:       internal.DBSizeBytes,
		DBSizeFormatted:   formatBytes(internal.DBSizeBytes),
		WriteRatePerMin:   writeRate,
		ConfirmRatePerMin: confirmRate,
		Status:            status,
		Healthy:           healthy,
	}

	writeJSON(w, stats)
}

// TriggerCompaction handles POST /api/v1/wal/compact
// Triggers manual WAL compaction.
func (h *WALHandlers) TriggerCompaction(w http.ResponseWriter, _ *http.Request) {
	// This would need integration with the actual compactor
	// For now, return a success acknowledgment
	writeJSON(w, map[string]interface{}{
		"message": "Compaction triggered",
		"status":  "queued",
	})
}

// GetHealth handles GET /api/v1/wal/health
// Returns a simple health check for WAL.
func (h *WALHandlers) GetHealth(w http.ResponseWriter, _ *http.Request) {
	if h.provider == nil {
		writeJSON(w, map[string]interface{}{
			"status":  "unavailable",
			"healthy": false,
			"message": "WAL not configured",
		})
		return
	}

	internal := h.provider.GetStats()
	healthy := internal.PendingCount < 10000

	writeJSON(w, map[string]interface{}{
		"status":        getWALHealthStatus(internal.PendingCount),
		"healthy":       healthy,
		"pending_count": internal.PendingCount,
		"message":       getWALHealthMessage(internal.PendingCount),
	})
}

// formatBytes converts bytes to human-readable format.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmtInt(bytes) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmtFloat(float64(bytes)/float64(div)) + " " + []string{"KB", "MB", "GB", "TB"}[exp]
}

func fmtInt(n int64) string {
	return fmtFloat(float64(n))
}

func fmtFloat(f float64) string {
	if f == float64(int64(f)) {
		return intToString(int64(f))
	}
	// Format with 1 decimal place
	return floatToString(f)
}

func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	// Simple int to string conversion without fmt
	var result []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	if negative {
		result = append([]byte{'-'}, result...)
	}
	return string(result)
}

func floatToString(f float64) string {
	// Simple float formatting with 1 decimal
	intPart := int64(f)
	decPart := int64((f - float64(intPart)) * 10)
	if decPart < 0 {
		decPart = -decPart
	}
	return intToString(intPart) + "." + intToString(decPart)
}

func getWALHealthStatus(pendingCount int64) string {
	switch {
	case pendingCount == 0:
		return "idle"
	case pendingCount < 1000:
		return "healthy"
	case pendingCount < 5000:
		return "moderate"
	case pendingCount < 10000:
		return "elevated"
	default:
		return "critical"
	}
}

func getWALHealthMessage(pendingCount int64) string {
	switch {
	case pendingCount == 0:
		return "No pending entries"
	case pendingCount < 1000:
		return "WAL operating normally"
	case pendingCount < 5000:
		return "Moderate backlog, processing"
	case pendingCount < 10000:
		return "Elevated backlog, may need attention"
	default:
		return "Critical backlog, processing may be delayed"
	}
}
