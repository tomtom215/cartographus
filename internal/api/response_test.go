// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"
)

func TestResponseWriter_Success(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	data := map[string]string{"message": "hello"}
	NewResponseWriter(w, r).Success(data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}

	if response.Error != nil {
		t.Error("Expected Error to be nil")
	}

	if response.Meta == nil {
		t.Error("Expected Meta to not be nil")
	}

	if response.Meta.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set")
	}
}

func TestResponseWriter_SuccessWithPagination(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	data := []string{"item1", "item2"}
	pagination := &PaginationMeta{
		Total:   100,
		Count:   2,
		Offset:  0,
		Limit:   10,
		HasMore: true,
	}

	NewResponseWriter(w, r).SuccessWithPagination(data, pagination)

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Meta == nil || response.Meta.Pagination == nil {
		t.Fatal("Expected pagination metadata")
	}

	if response.Meta.Pagination.Total != 100 {
		t.Errorf("Expected Total 100, got %d", response.Meta.Pagination.Total)
	}

	if !response.Meta.Pagination.HasMore {
		t.Error("Expected HasMore to be true")
	}
}

func TestResponseWriter_Created(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)

	data := map[string]int{"id": 123}
	NewResponseWriter(w, r).Created(data)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}
}

func TestResponseWriter_NoContent(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/test", nil)

	NewResponseWriter(w, r).NoContent()

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	if w.Body.Len() != 0 {
		t.Error("Expected empty body for NoContent")
	}
}

func TestResponseWriter_BadRequest(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)

	NewResponseWriter(w, r).BadRequest("invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("Expected Success to be false")
	}

	if response.Error == nil {
		t.Fatal("Expected Error to not be nil")
	}

	if response.Error.Code != ErrCodeBadRequest {
		t.Errorf("Expected code %s, got %s", ErrCodeBadRequest, response.Error.Code)
	}

	if response.Error.Message != "invalid input" {
		t.Errorf("Expected message 'invalid input', got '%s'", response.Error.Message)
	}
}

func TestResponseWriter_Unauthorized(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	NewResponseWriter(w, r).Unauthorized("token expired")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != ErrCodeUnauthorized {
		t.Errorf("Expected code %s, got %s", ErrCodeUnauthorized, response.Error.Code)
	}
}

func TestResponseWriter_Forbidden(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	NewResponseWriter(w, r).Forbidden("access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != ErrCodeForbidden {
		t.Errorf("Expected code %s, got %s", ErrCodeForbidden, response.Error.Code)
	}
}

func TestResponseWriter_NotFound(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test/123", nil)

	NewResponseWriter(w, r).NotFound("resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != ErrCodeNotFound {
		t.Errorf("Expected code %s, got %s", ErrCodeNotFound, response.Error.Code)
	}
}

func TestResponseWriter_Conflict(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)

	NewResponseWriter(w, r).Conflict("resource already exists")

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

func TestResponseWriter_TooManyRequests(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	NewResponseWriter(w, r).TooManyRequests("rate limit exceeded")

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != ErrCodeTooManyRequests {
		t.Errorf("Expected code %s, got %s", ErrCodeTooManyRequests, response.Error.Code)
	}
}

func TestResponseWriter_InternalError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	NewResponseWriter(w, r).InternalError("something went wrong")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestResponseWriter_ServiceUnavailable(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	NewResponseWriter(w, r).ServiceUnavailable("service down")

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestResponseWriter_ValidationError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)

	validationErrors := map[string]string{
		"email": "invalid email format",
		"name":  "required",
	}

	NewResponseWriter(w, r).ValidationError("validation failed", validationErrors)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != ErrCodeValidationFailed {
		t.Errorf("Expected code %s, got %s", ErrCodeValidationFailed, response.Error.Code)
	}

	if response.Error.Details == nil {
		t.Error("Expected validation details")
	}
}

func TestResponseWriter_ErrorWithDetails(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	details := map[string]interface{}{
		"field":   "username",
		"reason":  "duplicate",
		"suggest": "try username123",
	}

	NewResponseWriter(w, r).ErrorWithDetails(
		http.StatusConflict,
		ErrCodeConflict,
		"username already taken",
		details,
	)

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Details == nil {
		t.Error("Expected error details")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	t.Parallel()

	t.Run("WriteSuccess", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)

		WriteSuccess(w, r, "data")

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("WriteError", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)

		WriteError(w, r, http.StatusTeapot, "CUSTOM", "I'm a teapot")

		if w.Code != http.StatusTeapot {
			t.Errorf("Expected 418, got %d", w.Code)
		}
	})

	t.Run("WriteBadRequest", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)

		WriteBadRequest(w, r, "bad")

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("WriteNotFound", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)

		WriteNotFound(w, r, "not found")

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	t.Run("WriteInternalError", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)

		WriteInternalError(w, r, "error")

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

func TestResponseWriter_ContentType(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	NewResponseWriter(w, r).Success("test")

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected 'application/json; charset=utf-8', got '%s'", contentType)
	}
}
