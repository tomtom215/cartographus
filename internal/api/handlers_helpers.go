// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/validation"
)

// sanitizeLogValue removes control characters from strings to prevent log injection attacks.
// This includes newlines, carriage returns, tabs, and other control characters that could
// allow attackers to forge log entries or corrupt log files.
func sanitizeLogValue(s string) string {
	var result strings.Builder
	result.Grow(len(s))
	for _, r := range s {
		// Replace control characters (0x00-0x1F and 0x7F) with a safe representation
		if r < 0x20 || r == 0x7F {
			result.WriteString(fmt.Sprintf("\\x%02x", r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// respondJSON sends a JSON response with proper headers
func respondJSON(w http.ResponseWriter, status int, response *models.APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("Vary", "Accept-Encoding")

	data, err := json.Marshal(response)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to marshal JSON response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	etag := generateETag(data)
	w.Header().Set("ETag", etag)

	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		logging.Error().Err(err).Msg("Failed to write JSON response")
	}
}

// generateETag creates a simple ETag from data using FNV-1a hash
func generateETag(data []byte) string {
	hash := uint32(2166136261)
	for _, b := range data {
		hash ^= uint32(b)
		hash *= 16777619
	}
	return strconv.FormatUint(uint64(hash), 16)
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, code, message string, err error) {
	if err != nil {
		// Sanitize error output to prevent log injection attacks
		logging.Error().Str("code", sanitizeLogValue(code)).Str("error", sanitizeLogValue(err.Error())).Msg("API Error")
	}

	respondJSON(w, status, &models.APIResponse{
		Status: "error",
		Data:   nil,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
		Error: &models.APIError{
			Code:    code,
			Message: message,
		},
	})
}

// validateRequest validates a struct using go-playground/validator.
// Returns nil if validation passes, or a models.APIError if validation fails.
// The returned error uses the VALIDATION_ERROR code consistent with existing API errors.
//
// Example:
//
//	req := PlaybacksRequest{
//	    Limit:  getIntParam(r, "limit", 100),
//	    Offset: getIntParam(r, "offset", 0),
//	}
//	if apiErr := validateRequest(&req); apiErr != nil {
//	    respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
//	    return
//	}
func validateRequest(v interface{}) *models.APIError {
	validationErr := validation.ValidateStruct(v)
	if validationErr == nil {
		return nil
	}

	// Convert validation error to API error format
	apiErr := validationErr.ToAPIError()
	return &models.APIError{
		Code:    apiErr.Code,
		Message: apiErr.Message,
		Details: apiErr.Details,
	}
}

// getIntParam extracts an integer query parameter with a default value
func getIntParam(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

// parseIntParam parses an integer from a string with a default value.
// Uses fmt.Sscanf for lenient parsing (handles floats like "3.14" → 3, spaces like " 10 " → 10).
func parseIntParam(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

// parseCommaSeparated parses a comma-separated string into a slice
func parseCommaSeparated(value string) []string {
	if value == "" {
		return nil
	}

	var result []string
	parts := strings.Split(value, ",")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseCommaSeparatedInts parses a comma-separated string into a slice of integers
func parseCommaSeparatedInts(value string) []int {
	if value == "" {
		return nil
	}

	var result []int
	parts := strings.Split(value, ",")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			if num, err := strconv.Atoi(trimmed); err == nil {
				result = append(result, num)
			}
		}
	}
	return result
}

// escapeCSV escapes a string for CSV format
func escapeCSV(s string) string {
	// If string contains comma, quote, or newline, wrap in quotes and escape internal quotes
	needsQuotes := false
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			needsQuotes = true
			break
		}
	}

	if !needsQuotes {
		return s
	}

	// Replace " with ""
	escaped := ""
	for _, c := range s {
		if c == '"' {
			escaped += "\"\""
		} else {
			escaped += string(c)
		}
	}

	return "\"" + escaped + "\""
}
