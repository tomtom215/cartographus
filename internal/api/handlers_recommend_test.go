// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/tomtom215/cartographus/internal/logging"
)

func TestGetAlgorithms(t *testing.T) {
	t.Parallel()

	// Initialize logging for test
	logging.Init(logging.Config{})

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/algorithms", nil)
	rec := httptest.NewRecorder()

	// Create a minimal handler to test - we can test algorithmInfoMap directly
	// since GetAlgorithms just returns the static data
	h := &RecommendHandler{}
	h.GetAlgorithms(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("Expected status 'success', got %v", response["status"])
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatal("Expected data to be an array")
	}

	// Should have all algorithms from algorithmInfoMap
	if len(data) != len(algorithmInfoMap) {
		t.Errorf("Expected %d algorithms, got %d", len(algorithmInfoMap), len(data))
	}
}

func TestGetAlgorithms_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/algorithms", nil)
	rec := httptest.NewRecorder()

	h := &RecommendHandler{}
	h.GetAlgorithms(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestGetRecommendations_InvalidUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userID string
	}{
		{name: "non_numeric", userID: "abc"},
		{name: "empty", userID: ""},
		{name: "special_chars", userID: "!@#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/user/"+tt.userID, nil)
			rec := httptest.NewRecorder()

			// Set up chi context with URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h := &RecommendHandler{}
			h.GetRecommendations(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestGetRecommendations_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/user/1", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userID", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h := &RecommendHandler{}
	h.GetRecommendations(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestGetContinueWatching_InvalidUserID(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/user/invalid/continue", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userID", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h := &RecommendHandler{}
	h.GetContinueWatching(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGetContinueWatching_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/user/1/continue", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userID", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h := &RecommendHandler{}
	h.GetContinueWatching(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestGetSimilar_InvalidItemID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		itemID string
	}{
		{name: "non_numeric", itemID: "xyz"},
		{name: "empty", itemID: ""},
		{name: "negative_text", itemID: "negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/similar/"+tt.itemID, nil)
			rec := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("itemID", tt.itemID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h := &RecommendHandler{}
			h.GetSimilar(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestGetSimilar_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/similar/1", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("itemID", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h := &RecommendHandler{}
	h.GetSimilar(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestGetRecommendationStatus_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/status", nil)
	rec := httptest.NewRecorder()

	h := &RecommendHandler{}
	h.GetRecommendationStatus(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestGetRecommendationConfig_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/config", nil)
	rec := httptest.NewRecorder()

	h := &RecommendHandler{}
	h.GetRecommendationConfig(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestUpdateRecommendationConfig_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/config", nil)
	rec := httptest.NewRecorder()

	h := &RecommendHandler{}
	h.UpdateRecommendationConfig(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestUpdateRecommendationConfig_InvalidJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/recommendations/config", strings.NewReader("invalid json"))
	rec := httptest.NewRecorder()

	h := &RecommendHandler{}
	h.UpdateRecommendationConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTriggerTraining_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/train", nil)
	rec := httptest.NewRecorder()

	h := &RecommendHandler{}
	h.TriggerTraining(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestGetExploreRecommendations_InvalidUserID(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/user/invalid/explore", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userID", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h := &RecommendHandler{}
	h.GetExploreRecommendations(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGetExploreRecommendations_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/user/1/explore", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userID", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h := &RecommendHandler{}
	h.GetExploreRecommendations(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestGetWhatsNext_InvalidItemID(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/next/invalid", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("itemID", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h := &RecommendHandler{}
	h.GetWhatsNext(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGetWhatsNext_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/next/1", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("itemID", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h := &RecommendHandler{}
	h.GetWhatsNext(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestGetAlgorithmMetrics_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/algorithms/metrics", nil)
	rec := httptest.NewRecorder()

	h := &RecommendHandler{}
	h.GetAlgorithmMetrics(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestAlgorithmInfoMap_ContainsExpectedAlgorithms(t *testing.T) {
	t.Parallel()

	expectedAlgorithms := []string{
		"covisit", "content", "popularity", "ease", "als",
		"usercf", "itemcf", "fpmc", "markov", "bpr",
		"timeaware", "multihop", "linucb",
	}

	algIDMap := make(map[string]bool)
	for _, alg := range algorithmInfoMap {
		algIDMap[alg.ID] = true
	}

	for _, expected := range expectedAlgorithms {
		if !algIDMap[expected] {
			t.Errorf("Expected algorithm %q not found in algorithmInfoMap", expected)
		}
	}
}

func TestAlgorithmInfoMap_AllFieldsPopulated(t *testing.T) {
	t.Parallel()

	for _, alg := range algorithmInfoMap {
		if alg.ID == "" {
			t.Error("Algorithm ID is empty")
		}
		if alg.Name == "" {
			t.Errorf("Algorithm %s has empty Name", alg.ID)
		}
		if alg.Description == "" {
			t.Errorf("Algorithm %s has empty Description", alg.ID)
		}
		if alg.Tooltip == "" {
			t.Errorf("Algorithm %s has empty Tooltip", alg.ID)
		}
		if alg.Category == "" {
			t.Errorf("Algorithm %s has empty Category", alg.ID)
		}
	}
}

func TestAlgorithmCategories_Valid(t *testing.T) {
	t.Parallel()

	validCategories := map[string]bool{
		"basic":         true,
		"matrix":        true,
		"collaborative": true,
		"sequential":    true,
		"advanced":      true,
		"bandit":        true,
	}

	for _, alg := range algorithmInfoMap {
		if !validCategories[alg.Category] {
			t.Errorf("Algorithm %s has invalid category: %s", alg.ID, alg.Category)
		}
	}
}

func TestQueryParamParsing_K(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{name: "default", query: "", expected: 20},
		{name: "valid_5", query: "k=5", expected: 5},
		{name: "valid_100", query: "k=100", expected: 100},
		{name: "invalid_string", query: "k=abc", expected: 20},
		{name: "invalid_zero", query: "k=0", expected: 20},
		{name: "invalid_negative", query: "k=-5", expected: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Parse k parameter similar to how GetRecommendations does it
			k := 20
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			if kStr := req.URL.Query().Get("k"); kStr != "" {
				if parsed, err := parsePositiveInt(kStr); err == nil {
					k = parsed
				}
			}

			if k != tt.expected {
				t.Errorf("Expected k=%d, got k=%d", tt.expected, k)
			}
		})
	}
}

// parsePositiveInt is a helper to parse positive integers, returning error for invalid values.
func parsePositiveInt(s string) (int, error) {
	var val int
	_, err := strings.NewReader(s).Read(nil)
	if err != nil && err.Error() != "EOF" {
		return 0, err
	}

	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, http.ErrNotSupported
		}
		val = val*10 + int(c-'0')
	}

	if val <= 0 {
		return 0, http.ErrNotSupported
	}

	return val, nil
}
