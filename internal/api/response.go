// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides standardized API response handling.
// Phase 3: All API endpoints use consistent response format.
package api

import (
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// APIResponse is the standardized response wrapper for all API endpoints.
// This provides consistent structure across all responses for better client handling.
type APIResponse struct {
	// Success indicates whether the request was successful
	Success bool `json:"success"`

	// Data contains the response payload (null on error)
	Data interface{} `json:"data,omitempty"`

	// Error contains error details (null on success)
	Error *APIError `json:"error,omitempty"`

	// Meta contains optional metadata about the response
	Meta *APIMeta `json:"meta,omitempty"`
}

// APIError represents an error response.
type APIError struct {
	// Code is a machine-readable error code
	Code string `json:"code"`

	// Message is a human-readable error message
	Message string `json:"message"`

	// Details contains additional error details (optional)
	Details interface{} `json:"details,omitempty"`

	// RequestID is the request ID for tracing
	RequestID string `json:"request_id,omitempty"`
}

// APIMeta contains optional response metadata.
type APIMeta struct {
	// RequestID is the unique request identifier for tracing
	RequestID string `json:"request_id,omitempty"`

	// Timestamp is when the response was generated
	Timestamp time.Time `json:"timestamp"`

	// Duration is the request processing time in milliseconds
	DurationMs int64 `json:"duration_ms,omitempty"`

	// Pagination contains pagination info for list responses
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

// PaginationMeta contains pagination information for list responses.
type PaginationMeta struct {
	// Total is the total number of items
	Total int64 `json:"total,omitempty"`

	// Count is the number of items in this response
	Count int `json:"count"`

	// Offset is the offset used
	Offset int `json:"offset,omitempty"`

	// Limit is the limit used
	Limit int `json:"limit,omitempty"`

	// HasMore indicates if there are more items
	HasMore bool `json:"has_more"`

	// NextCursor is the cursor for the next page (for cursor-based pagination)
	NextCursor string `json:"next_cursor,omitempty"`
}

// Error codes for API responses
const (
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeTooManyRequests     = "TOO_MANY_REQUESTS"
	ErrCodeInternalError       = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
	ErrCodeDatabaseError       = "DATABASE_ERROR"
	ErrCodeExternalServiceFail = "EXTERNAL_SERVICE_FAILED"
)

// ResponseWriter provides methods for writing standardized API responses.
type ResponseWriter struct {
	w         http.ResponseWriter
	r         *http.Request
	startTime time.Time
}

// NewResponseWriter creates a new response writer.
func NewResponseWriter(w http.ResponseWriter, r *http.Request) *ResponseWriter {
	return &ResponseWriter{
		w:         w,
		r:         r,
		startTime: time.Now(),
	}
}

// Success writes a successful response with data.
func (rw *ResponseWriter) Success(data interface{}) {
	rw.SuccessWithMeta(data, nil)
}

// SuccessWithMeta writes a successful response with data and metadata.
func (rw *ResponseWriter) SuccessWithMeta(data interface{}, meta *APIMeta) {
	if meta == nil {
		meta = &APIMeta{}
	}
	meta.Timestamp = time.Now()
	meta.DurationMs = time.Since(rw.startTime).Milliseconds()
	meta.RequestID = logging.RequestIDFromContext(rw.r.Context())

	response := APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	}

	rw.writeJSON(http.StatusOK, response)
}

// SuccessWithPagination writes a successful paginated response.
func (rw *ResponseWriter) SuccessWithPagination(data interface{}, pagination *PaginationMeta) {
	meta := &APIMeta{
		Pagination: pagination,
	}
	rw.SuccessWithMeta(data, meta)
}

// Created writes a 201 Created response.
func (rw *ResponseWriter) Created(data interface{}) {
	meta := &APIMeta{
		Timestamp:  time.Now(),
		DurationMs: time.Since(rw.startTime).Milliseconds(),
		RequestID:  logging.RequestIDFromContext(rw.r.Context()),
	}

	response := APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	}

	rw.writeJSON(http.StatusCreated, response)
}

// NoContent writes a 204 No Content response.
func (rw *ResponseWriter) NoContent() {
	rw.w.WriteHeader(http.StatusNoContent)
}

// Error writes an error response with the given status code.
func (rw *ResponseWriter) Error(statusCode int, code, message string) {
	rw.ErrorWithDetails(statusCode, code, message, nil)
}

// ErrorWithDetails writes an error response with additional details.
func (rw *ResponseWriter) ErrorWithDetails(statusCode int, code, message string, details interface{}) {
	requestID := logging.RequestIDFromContext(rw.r.Context())

	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: requestID,
		},
		Meta: &APIMeta{
			Timestamp:  time.Now(),
			DurationMs: time.Since(rw.startTime).Milliseconds(),
			RequestID:  requestID,
		},
	}

	rw.writeJSON(statusCode, response)
}

// BadRequest writes a 400 Bad Request error.
func (rw *ResponseWriter) BadRequest(message string) {
	rw.Error(http.StatusBadRequest, ErrCodeBadRequest, message)
}

// BadRequestWithDetails writes a 400 Bad Request error with details.
func (rw *ResponseWriter) BadRequestWithDetails(message string, details interface{}) {
	rw.ErrorWithDetails(http.StatusBadRequest, ErrCodeBadRequest, message, details)
}

// Unauthorized writes a 401 Unauthorized error.
func (rw *ResponseWriter) Unauthorized(message string) {
	rw.Error(http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// Forbidden writes a 403 Forbidden error.
func (rw *ResponseWriter) Forbidden(message string) {
	rw.Error(http.StatusForbidden, ErrCodeForbidden, message)
}

// NotFound writes a 404 Not Found error.
func (rw *ResponseWriter) NotFound(message string) {
	rw.Error(http.StatusNotFound, ErrCodeNotFound, message)
}

// Conflict writes a 409 Conflict error.
func (rw *ResponseWriter) Conflict(message string) {
	rw.Error(http.StatusConflict, ErrCodeConflict, message)
}

// TooManyRequests writes a 429 Too Many Requests error.
func (rw *ResponseWriter) TooManyRequests(message string) {
	rw.Error(http.StatusTooManyRequests, ErrCodeTooManyRequests, message)
}

// InternalError writes a 500 Internal Server Error.
func (rw *ResponseWriter) InternalError(message string) {
	rw.Error(http.StatusInternalServerError, ErrCodeInternalError, message)
}

// ServiceUnavailable writes a 503 Service Unavailable error.
func (rw *ResponseWriter) ServiceUnavailable(message string) {
	rw.Error(http.StatusServiceUnavailable, ErrCodeServiceUnavailable, message)
}

// ValidationError writes a 400 error with validation details.
func (rw *ResponseWriter) ValidationError(message string, validationErrors interface{}) {
	rw.ErrorWithDetails(http.StatusBadRequest, ErrCodeValidationFailed, message, validationErrors)
}

// DatabaseError writes a 500 error for database failures.
func (rw *ResponseWriter) DatabaseError(err error) {
	logging.Error().Err(err).Msg("Database error")
	rw.Error(http.StatusInternalServerError, ErrCodeDatabaseError, "A database error occurred")
}

// ExternalServiceError writes a 502 error for external service failures.
func (rw *ResponseWriter) ExternalServiceError(service string, err error) {
	logging.Error().Err(err).Str("service", service).Msg("External service error")
	rw.Error(http.StatusBadGateway, ErrCodeExternalServiceFail, "External service unavailable: "+service)
}

// writeJSON writes JSON response with proper headers.
func (rw *ResponseWriter) writeJSON(statusCode int, data interface{}) {
	rw.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.w.WriteHeader(statusCode)

	if err := json.NewEncoder(rw.w).Encode(data); err != nil {
		logging.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

// WriteSuccess is a convenience function for writing success responses.
// For use in handlers that don't need the full ResponseWriter functionality.
func WriteSuccess(w http.ResponseWriter, r *http.Request, data interface{}) {
	NewResponseWriter(w, r).Success(data)
}

// WriteError is a convenience function for writing error responses.
func WriteError(w http.ResponseWriter, r *http.Request, statusCode int, code, message string) {
	NewResponseWriter(w, r).Error(statusCode, code, message)
}

// WriteBadRequest is a convenience function for 400 errors.
func WriteBadRequest(w http.ResponseWriter, r *http.Request, message string) {
	NewResponseWriter(w, r).BadRequest(message)
}

// WriteNotFound is a convenience function for 404 errors.
func WriteNotFound(w http.ResponseWriter, r *http.Request, message string) {
	NewResponseWriter(w, r).NotFound(message)
}

// WriteInternalError is a convenience function for 500 errors.
func WriteInternalError(w http.ResponseWriter, r *http.Request, message string) {
	NewResponseWriter(w, r).InternalError(message)
}

// WriteDatabaseError is a convenience function for database errors.
func WriteDatabaseError(w http.ResponseWriter, r *http.Request, err error) {
	NewResponseWriter(w, r).DatabaseError(err)
}
