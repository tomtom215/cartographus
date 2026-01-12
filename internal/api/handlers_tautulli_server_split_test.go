// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestTautulliServerInfo_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetServerInfoFunc: func(ctx context.Context) (*tautulli.TautulliServerInfo, error) {
			return &tautulli.TautulliServerInfo{
				Response: tautulli.TautulliServerInfoResponse{
					Result: "success",
					Data: tautulli.TautulliServerInfoData{
						PMSName:    "Test Server",
						PMSVersion: "1.40.0",
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/server-info", nil)
	w := httptest.NewRecorder()

	handler.TautulliServerInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliSyncedItems_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetSyncedItemsFunc: func(ctx context.Context, machineID string, userID int) (*tautulli.TautulliSyncedItems, error) {
			return &tautulli.TautulliSyncedItems{
				Response: tautulli.TautulliSyncedItemsResponse{
					Result: "success",
					Data:   []tautulli.TautulliSyncedItem{},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/synced-items", nil)
	w := httptest.NewRecorder()

	handler.TautulliSyncedItems(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliTerminateSession_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		TerminateSessionFunc: func(ctx context.Context, sessionID string, message string) (*tautulli.TautulliTerminateSession, error) {
			return &tautulli.TautulliTerminateSession{
				Response: tautulli.TautulliTerminateSessionResponse{
					Result:  "success",
					Message: &message,
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tautulli/terminate-session?session_id=abc123&message=test", nil)
	w := httptest.NewRecorder()

	handler.TautulliTerminateSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliTerminateSession_MissingSessionID(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tautulli/terminate-session", nil)
	w := httptest.NewRecorder()

	handler.TautulliTerminateSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// ServerInfo endpoint tests

func TestServerInfo_Success(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Latitude:  40.7128,
			Longitude: -74.0060,
		},
	}

	handler := &Handler{
		config: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server-info", nil)
	w := httptest.NewRecorder()

	handler.ServerInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	if data["latitude"] != 40.7128 {
		t.Errorf("Expected latitude 40.7128, got %v", data["latitude"])
	}

	if data["longitude"] != -74.0060 {
		t.Errorf("Expected longitude -74.0060, got %v", data["longitude"])
	}

	if data["has_location"] != true {
		t.Errorf("Expected has_location true, got %v", data["has_location"])
	}
}

func TestServerInfo_NoLocationConfigured(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Latitude:  0.0,
			Longitude: 0.0,
		},
	}

	handler := &Handler{
		config: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server-info", nil)
	w := httptest.NewRecorder()

	handler.ServerInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	if data["has_location"] != false {
		t.Errorf("Expected has_location false when coordinates are 0,0, got %v", data["has_location"])
	}
}
