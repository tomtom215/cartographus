// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tomtom215/cartographus/internal/database"
)

// ========================================
// FuzzySearch Handler Tests
// ========================================

func TestFuzzySearch_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(method, "/api/v1/search/fuzzy?q=test", nil)
			rec := httptest.NewRecorder()

			handler.FuzzySearch(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}
}

func TestFuzzySearch_MissingQuery(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/fuzzy", nil)
	rec := httptest.NewRecorder()

	handler.FuzzySearch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["message"] != "Query parameter 'q' is required" {
		t.Errorf("Expected query required error, got %v", errObj["message"])
	}
}

func TestFuzzySearch_QueryTooLong(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	// Create a query string > 200 characters
	longQuery := strings.Repeat("a", 201)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/fuzzy?q="+longQuery, nil)
	rec := httptest.NewRecorder()

	handler.FuzzySearch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if !strings.Contains(errObj["message"].(string), "200 characters") {
		t.Errorf("Expected character limit error, got %v", errObj["message"])
	}
}

func TestFuzzySearch_InvalidMinScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		minScore string
	}{
		{name: "non_numeric", minScore: "abc"},
		{name: "negative", minScore: "-1"},
		{name: "over_100", minScore: "101"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/search/fuzzy?q=test&min_score="+tt.minScore, nil)
			rec := httptest.NewRecorder()

			handler.FuzzySearch(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			errObj, ok := response["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected error object in response")
			}

			if !strings.Contains(errObj["message"].(string), "min_score") {
				t.Errorf("Expected min_score error, got %v", errObj["message"])
			}
		})
	}
}

func TestFuzzySearch_InvalidLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit string
	}{
		{name: "non_numeric", limit: "xyz"},
		{name: "zero", limit: "0"},
		{name: "over_100", limit: "101"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/search/fuzzy?q=test&limit="+tt.limit, nil)
			rec := httptest.NewRecorder()

			handler.FuzzySearch(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			errObj, ok := response["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected error object in response")
			}

			if !strings.Contains(errObj["message"].(string), "limit") {
				t.Errorf("Expected limit error, got %v", errObj["message"])
			}
		})
	}
}

func TestFuzzySearch_ValidParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{name: "simple_query", query: "q=test"},
		{name: "with_min_score", query: "q=movie&min_score=50"},
		{name: "with_limit", query: "q=movie&limit=10"},
		{name: "all_params", query: "q=movie&min_score=80&limit=50"},
		{name: "min_score_boundary_0", query: "q=test&min_score=0"},
		{name: "min_score_boundary_100", query: "q=test&min_score=100"},
		{name: "limit_boundary_1", query: "q=test&limit=1"},
		{name: "limit_boundary_100", query: "q=test&limit=100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/search/fuzzy?"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.FuzzySearch(rec, req)

			// Should either succeed or return database error (not validation error)
			if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 200 or 500, got %d", rec.Code)
			}
		})
	}
}

func TestFuzzySearch_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/fuzzy?q=test", nil)
	rec := httptest.NewRecorder()

	handler.FuzzySearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("Expected status=success, got %v", response["status"])
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data object in response")
	}

	// Results should be an array (possibly empty)
	_, ok = data["results"].([]interface{})
	if !ok {
		t.Error("Expected results array")
	}

	// Count should be present
	_, ok = data["count"].(float64)
	if !ok {
		t.Error("Expected count field")
	}
}

// ========================================
// FuzzySearchUsers Handler Tests
// ========================================

func TestFuzzySearchUsers_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(method, "/api/v1/search/users?q=john", nil)
			rec := httptest.NewRecorder()

			handler.FuzzySearchUsers(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}
}

func TestFuzzySearchUsers_MissingQuery(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/users", nil)
	rec := httptest.NewRecorder()

	handler.FuzzySearchUsers(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["message"] != "Query parameter 'q' is required" {
		t.Errorf("Expected query required error, got %v", errObj["message"])
	}
}

func TestFuzzySearchUsers_QueryTooLong(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	longQuery := strings.Repeat("b", 201)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/users?q="+longQuery, nil)
	rec := httptest.NewRecorder()

	handler.FuzzySearchUsers(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestFuzzySearchUsers_InvalidMinScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		minScore string
	}{
		{name: "non_numeric", minScore: "bad"},
		{name: "negative", minScore: "-5"},
		{name: "over_100", minScore: "150"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/search/users?q=john&min_score="+tt.minScore, nil)
			rec := httptest.NewRecorder()

			handler.FuzzySearchUsers(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestFuzzySearchUsers_InvalidLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit string
	}{
		{name: "non_numeric", limit: "bad"},
		{name: "zero", limit: "0"},
		{name: "negative", limit: "-1"},
		{name: "over_100", limit: "200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/search/users?q=john&limit="+tt.limit, nil)
			rec := httptest.NewRecorder()

			handler.FuzzySearchUsers(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestFuzzySearchUsers_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/users?q=john", nil)
	rec := httptest.NewRecorder()

	handler.FuzzySearchUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("Expected status=success, got %v", response["status"])
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data object in response")
	}

	_, ok = data["results"].([]interface{})
	if !ok {
		t.Error("Expected results array")
	}

	_, ok = data["count"].(float64)
	if !ok {
		t.Error("Expected count field")
	}
}

// ========================================
// Response Type Tests
// ========================================

func TestFuzzySearchResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := FuzzySearchResponse{
		Results:     []database.FuzzySearchResult{},
		Count:       0,
		FuzzySearch: true,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["fuzzy_search"] != true {
		t.Error("Expected fuzzy_search=true")
	}
	if decoded["count"].(float64) != 0 {
		t.Errorf("Expected count=0, got %v", decoded["count"])
	}
}

func TestFuzzyUserSearchResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := FuzzyUserSearchResponse{
		Results:     []database.UserSearchResult{},
		Count:       0,
		FuzzySearch: false,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["fuzzy_search"] != false {
		t.Error("Expected fuzzy_search=false")
	}
	if decoded["count"].(float64) != 0 {
		t.Errorf("Expected count=0, got %v", decoded["count"])
	}
}
