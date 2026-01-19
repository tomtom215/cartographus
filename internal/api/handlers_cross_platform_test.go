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

func TestContentMappingCreate_MissingTitle(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{"media_type": "movie", "imdb_id": "tt1234567"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/link", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ContentMappingCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != false {
		t.Error("Expected success=false")
	}
	if response["error"] != "title is required" {
		t.Errorf("Expected error 'title is required', got %v", response["error"])
	}
}

func TestContentMappingCreate_MissingMediaType(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{"title": "Test Movie", "imdb_id": "tt1234567"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/link", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ContentMappingCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "media_type is required (movie, show, or episode)" {
		t.Errorf("Expected media_type error, got %v", response["error"])
	}
}

func TestContentMappingCreate_MissingExternalID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{"title": "Test Movie", "media_type": "movie"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/link", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ContentMappingCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.Contains(response["error"].(string), "At least one external ID") {
		t.Errorf("Expected external ID error, got %v", response["error"])
	}
}

func TestContentMappingCreate_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/link", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ContentMappingCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Invalid JSON body" {
		t.Errorf("Expected 'Invalid JSON body' error, got %v", response["error"])
	}
}

func TestContentMappingLookup_MissingType(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/content/lookup?id=tt1234567", nil)
	rec := httptest.NewRecorder()

	handler.ContentMappingLookup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.Contains(response["error"].(string), "Both 'type' and 'id'") {
		t.Errorf("Expected missing params error, got %v", response["error"])
	}
}

func TestContentMappingLookup_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/content/lookup?type=imdb", nil)
	rec := httptest.NewRecorder()

	handler.ContentMappingLookup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestContentMappingLookup_InvalidType(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/content/lookup?type=invalid&id=123", nil)
	rec := httptest.NewRecorder()

	handler.ContentMappingLookup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.Contains(response["error"].(string), "Invalid type") {
		t.Errorf("Expected invalid type error, got %v", response["error"])
	}
}

func TestContentMappingLookup_InvalidNumericID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		idType   string
		id       string
		expected int
	}{
		{name: "tmdb_non_numeric", idType: "tmdb", id: "abc", expected: http.StatusBadRequest},
		{name: "tvdb_non_numeric", idType: "tvdb", id: "xyz", expected: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/content/lookup?type="+tt.idType+"&id="+tt.id, nil)
			rec := httptest.NewRecorder()

			handler.ContentMappingLookup(rec, req)

			if rec.Code != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, rec.Code)
			}
		})
	}
}

func TestContentMappingLookup_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/content/lookup?type=imdb&id=tt9999999", nil)
	rec := httptest.NewRecorder()

	handler.ContentMappingLookup(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestContentMappingLookup_ValidTypes(t *testing.T) {
	t.Parallel()

	validTypes := []string{"imdb", "tmdb", "tvdb", "plex", "jellyfin", "emby"}

	for _, idType := range validTypes {
		t.Run(idType, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			id := "test123"
			if idType == "tmdb" || idType == "tvdb" {
				id = "12345"
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/content/lookup?type="+idType+"&id="+id, nil)
			rec := httptest.NewRecorder()

			handler.ContentMappingLookup(rec, req)

			// Should either return 200 (found) or 404 (not found), not 400 (bad request)
			if rec.Code == http.StatusBadRequest {
				t.Errorf("Type %s should be valid, got %d", idType, rec.Code)
			}
		})
	}
}

func TestUserLinkCreate_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{invalid`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/link", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UserLinkCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUserLinkCreate_InvalidUserIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "zero_primary", body: `{"primary_user_id": 0, "linked_user_id": 2, "link_type": "manual"}`},
		{name: "negative_primary", body: `{"primary_user_id": -1, "linked_user_id": 2, "link_type": "manual"}`},
		{name: "zero_linked", body: `{"primary_user_id": 1, "linked_user_id": 0, "link_type": "manual"}`},
		{name: "negative_linked", body: `{"primary_user_id": 1, "linked_user_id": -1, "link_type": "manual"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/users/link", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.UserLinkCreate(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if !strings.Contains(response["error"].(string), "positive integers") {
				t.Errorf("Expected positive integers error, got %v", response["error"])
			}
		})
	}
}

func TestUserLinkCreate_MissingLinkType(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{"primary_user_id": 1, "linked_user_id": 2}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/link", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UserLinkCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.Contains(response["error"].(string), "link_type is required") {
		t.Errorf("Expected link_type error, got %v", response["error"])
	}
}

func TestUserLinkDelete_MissingParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{name: "missing_both", query: ""},
		{name: "missing_linked", query: "primary_id=1"},
		{name: "missing_primary", query: "linked_id=2"},
		{name: "invalid_primary", query: "primary_id=abc&linked_id=2"},
		{name: "invalid_linked", query: "primary_id=1&linked_id=xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			url := "/api/v1/users/link"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			rec := httptest.NewRecorder()

			handler.UserLinkDelete(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestWriteJSONResponse(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()

	data := map[string]interface{}{
		"success": true,
		"message": "test",
	}

	writeJSONResponse(rec, http.StatusOK, data)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", rec.Header().Get("Content-Type"))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success=true")
	}
	if response["message"] != "test" {
		t.Error("Expected message='test'")
	}
}

func TestWriteJSONResponse_DifferentStatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
	}{
		{name: "ok", status: http.StatusOK},
		{name: "created", status: http.StatusCreated},
		{name: "bad_request", status: http.StatusBadRequest},
		{name: "not_found", status: http.StatusNotFound},
		{name: "internal_error", status: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			writeJSONResponse(rec, tt.status, map[string]string{"test": "value"})

			if rec.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, rec.Code)
			}
		})
	}
}

func TestContentMappingRequest_Fields(t *testing.T) {
	t.Parallel()

	// Test that ContentMappingRequest can be properly unmarshaled
	jsonData := `{
		"imdb_id": "tt1234567",
		"tmdb_id": 12345,
		"tvdb_id": 67890,
		"plex_rating_key": "abc123",
		"jellyfin_item_id": "uuid-string",
		"emby_item_id": "emby123",
		"title": "Test Movie",
		"media_type": "movie",
		"year": 2024
	}`

	var req ContentMappingRequest
	if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req.Title != "Test Movie" {
		t.Errorf("Expected title 'Test Movie', got %s", req.Title)
	}
	if req.MediaType != "movie" {
		t.Errorf("Expected media_type 'movie', got %s", req.MediaType)
	}
	if req.IMDbID == nil || *req.IMDbID != "tt1234567" {
		t.Error("Expected imdb_id 'tt1234567'")
	}
	if req.TMDbID == nil || *req.TMDbID != 12345 {
		t.Error("Expected tmdb_id 12345")
	}
	if req.Year == nil || *req.Year != 2024 {
		t.Error("Expected year 2024")
	}
}

func TestUserLinkRequest_Fields(t *testing.T) {
	t.Parallel()

	jsonData := `{
		"primary_user_id": 1,
		"linked_user_id": 2,
		"link_type": "manual"
	}`

	var req UserLinkRequest
	if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req.PrimaryUserID != 1 {
		t.Errorf("Expected primary_user_id 1, got %d", req.PrimaryUserID)
	}
	if req.LinkedUserID != 2 {
		t.Errorf("Expected linked_user_id 2, got %d", req.LinkedUserID)
	}
	if req.LinkType != "manual" {
		t.Errorf("Expected link_type 'manual', got %s", req.LinkType)
	}
}

func TestContentMappingResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := ContentMappingResponse{
		Success: true,
		Message: "Content mapping created",
		Created: true,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["success"] != true {
		t.Error("Expected success=true")
	}
	if decoded["message"] != "Content mapping created" {
		t.Errorf("Expected message, got %v", decoded["message"])
	}
	if decoded["created"] != true {
		t.Error("Expected created=true")
	}
}

func TestUserLinkResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := UserLinkResponse{
		Success: true,
		Message: "User link created",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["success"] != true {
		t.Error("Expected success=true")
	}
	if decoded["message"] != "User link created" {
		t.Errorf("Expected message, got %v", decoded["message"])
	}
}

func TestLinkedUsersResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := LinkedUsersResponse{
		Success: true,
		UserIDs: []int{1, 2, 3},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["success"] != true {
		t.Error("Expected success=true")
	}

	userIDs, ok := decoded["user_ids"].([]interface{})
	if !ok {
		t.Fatal("Expected user_ids array")
	}
	if len(userIDs) != 3 {
		t.Errorf("Expected 3 user IDs, got %d", len(userIDs))
	}
}

func TestSuggestedLinksResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := SuggestedLinksResponse{
		Success:     true,
		Suggestions: make(map[string][]*database.LinkedUserInfo),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["success"] != true {
		t.Error("Expected success=true")
	}

	_, ok := decoded["suggestions"].(map[string]interface{})
	if !ok {
		t.Error("Expected suggestions map")
	}
}

// ========================================
// ContentMappingLink* Tests (0% coverage)
// ========================================

func TestContentMappingLinkPlex_InvalidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{"rating_key": "12345"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/abc/link/plex", strings.NewReader(body))
	req.SetPathValue("id", "abc")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ContentMappingLinkPlex(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Invalid mapping ID" {
		t.Errorf("Expected 'Invalid mapping ID' error, got %v", response["error"])
	}
}

func TestContentMappingLinkPlex_MissingRatingKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "empty_body", body: `{}`},
		{name: "empty_rating_key", body: `{"rating_key": ""}`},
		{name: "invalid_json", body: `{invalid`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/content/1/link/plex", strings.NewReader(tt.body))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ContentMappingLinkPlex(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != "rating_key is required" {
				t.Errorf("Expected 'rating_key is required' error, got %v", response["error"])
			}
		})
	}
}

func TestContentMappingLinkJellyfin_InvalidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{"item_id": "abc-uuid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/invalid/link/jellyfin", strings.NewReader(body))
	req.SetPathValue("id", "invalid")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ContentMappingLinkJellyfin(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Invalid mapping ID" {
		t.Errorf("Expected 'Invalid mapping ID' error, got %v", response["error"])
	}
}

func TestContentMappingLinkJellyfin_MissingItemID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "empty_body", body: `{}`},
		{name: "empty_item_id", body: `{"item_id": ""}`},
		{name: "invalid_json", body: `{bad`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/content/1/link/jellyfin", strings.NewReader(tt.body))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ContentMappingLinkJellyfin(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != "item_id is required" {
				t.Errorf("Expected 'item_id is required' error, got %v", response["error"])
			}
		})
	}
}

func TestContentMappingLinkEmby_InvalidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	body := `{"item_id": "emby-123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/xyz/link/emby", strings.NewReader(body))
	req.SetPathValue("id", "xyz")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ContentMappingLinkEmby(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Invalid mapping ID" {
		t.Errorf("Expected 'Invalid mapping ID' error, got %v", response["error"])
	}
}

func TestContentMappingLinkEmby_MissingItemID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "empty_body", body: `{}`},
		{name: "empty_item_id", body: `{"item_id": ""}`},
		{name: "invalid_json", body: `{nope`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/content/1/link/emby", strings.NewReader(tt.body))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ContentMappingLinkEmby(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != "item_id is required" {
				t.Errorf("Expected 'item_id is required' error, got %v", response["error"])
			}
		})
	}
}

// ========================================
// UserLinkedGet Tests (0% coverage)
// ========================================

func TestUserLinkedGet_InvalidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/invalid/linked", nil)
	req.SetPathValue("id", "invalid")
	rec := httptest.NewRecorder()

	handler.UserLinkedGet(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Invalid user ID" {
		t.Errorf("Expected 'Invalid user ID' error, got %v", response["error"])
	}
}

func TestUserLinkedGet_ValidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1/linked", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.UserLinkedGet(rec, req)

	// Should succeed or return empty results (not error on valid ID)
	if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", rec.Code)
	}
}

// ========================================
// UserSuggestLinks Tests (0% coverage)
// ========================================

func TestUserSuggestLinks_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/suggest-links", nil)
	rec := httptest.NewRecorder()

	handler.UserSuggestLinks(rec, req)

	// Should succeed (empty suggestions is fine)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success=true")
	}
}

// ========================================
// CrossPlatform Analytics Tests (0% coverage)
// ========================================

func TestCrossPlatformUserStats_InvalidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cross-platform/user/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	handler.CrossPlatformUserStats(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Invalid user ID" {
		t.Errorf("Expected 'Invalid user ID' error, got %v", response["error"])
	}
}

func TestCrossPlatformUserStats_ValidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cross-platform/user/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.CrossPlatformUserStats(rec, req)

	// Should succeed or return error based on data availability
	if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", rec.Code)
	}
}

func TestCrossPlatformContentStats_InvalidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cross-platform/content/invalid", nil)
	req.SetPathValue("id", "invalid")
	rec := httptest.NewRecorder()

	handler.CrossPlatformContentStats(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Invalid content mapping ID" {
		t.Errorf("Expected 'Invalid content mapping ID' error, got %v", response["error"])
	}
}

func TestCrossPlatformContentStats_ValidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cross-platform/content/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.CrossPlatformContentStats(rec, req)

	// Should succeed or return not found/error based on data
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200, 404, or 500, got %d", rec.Code)
	}
}

func TestCrossPlatformSummary_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/cross-platform/summary", nil)
	rec := httptest.NewRecorder()

	handler.CrossPlatformSummary(rec, req)

	// Should succeed with empty or populated summary
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success=true")
	}
}
