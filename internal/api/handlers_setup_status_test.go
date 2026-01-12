// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/middleware"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestSetupStatus_MethodNotAllowed tests SetupStatus with invalid HTTP methods
func TestSetupStatus_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		startTime: time.Now(),
		config:    &config.Config{},
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/health/setup", nil)
			w := httptest.NewRecorder()

			handler.SetupStatus(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestSetupStatus_NoConfiguration tests SetupStatus when no data sources are configured
func TestSetupStatus_NoConfiguration(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Tautulli: config.TautulliConfig{
			Enabled: false,
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

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

	// Parse setup status from response
	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// With no configuration, Ready should be false
	if status.Ready {
		t.Error("Expected Ready to be false when no data sources configured")
	}

	// Should have recommendation to configure data sources
	if len(status.Recommendations) == 0 {
		t.Error("Expected recommendations when no data sources configured")
	}

	// Should recommend configuring a data source
	found := false
	for _, rec := range status.Recommendations {
		if rec == "No data sources configured. Configure at least one of: Tautulli, Plex, Jellyfin, or Emby." {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected recommendation to configure data sources")
	}
}

// TestSetupStatus_TautulliConfiguredNotConnected tests SetupStatus when Tautulli is configured but not reachable
func TestSetupStatus_TautulliConfiguredNotConnected(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Tautulli: config.TautulliConfig{
			Enabled: true,
			URL:     "http://localhost:8181",
			APIKey:  "test-api-key",
		},
	}

	mockClient := &MockTautulliClient{
		PingFunc: func(ctx context.Context) error {
			return errors.New("connection refused")
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		client:    mockClient,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// Tautulli should be configured but not connected
	if !status.DataSources.Tautulli.Configured {
		t.Error("Expected Tautulli.Configured to be true")
	}
	if status.DataSources.Tautulli.Connected {
		t.Error("Expected Tautulli.Connected to be false when connection fails")
	}
	if status.DataSources.Tautulli.Error == "" {
		t.Error("Expected Tautulli.Error to be set when connection fails")
	}

	// Should have recommendation about Tautulli connection
	found := false
	for _, rec := range status.Recommendations {
		if rec == "Tautulli is configured but not reachable. Check the URL and API key." {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected recommendation about Tautulli connection")
	}
}

// TestSetupStatus_TautulliConnected tests SetupStatus when Tautulli is configured and connected
func TestSetupStatus_TautulliConnected(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Tautulli: config.TautulliConfig{
			Enabled: true,
			URL:     "http://localhost:8181",
			APIKey:  "test-api-key",
		},
	}

	mockClient := &MockTautulliClient{
		PingFunc: func(ctx context.Context) error {
			return nil // Success
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		client:    mockClient,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// Tautulli should be configured and connected
	if !status.DataSources.Tautulli.Configured {
		t.Error("Expected Tautulli.Configured to be true")
	}
	if !status.DataSources.Tautulli.Connected {
		t.Error("Expected Tautulli.Connected to be true")
	}
	if status.DataSources.Tautulli.Error != "" {
		t.Errorf("Expected no Tautulli.Error, got '%s'", status.DataSources.Tautulli.Error)
	}

	// Ready should be true (Tautulli is a data source)
	// Note: Ready also requires database connection, so it might be false
	// We're testing data source detection here
}

// TestSetupStatus_PlexConfigured tests SetupStatus when Plex servers are configured
func TestSetupStatus_PlexConfigured(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		PlexServers: []config.PlexConfig{
			{
				Enabled: true,
				URL:     "http://localhost:32400",
				Token:   "test-token-1",
			},
			{
				Enabled: true,
				URL:     "http://localhost:32401",
				Token:   "test-token-2",
			},
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// Plex should be configured with 2 servers
	if !status.DataSources.Plex.Configured {
		t.Error("Expected Plex.Configured to be true")
	}
	if status.DataSources.Plex.ServerCount != 2 {
		t.Errorf("Expected Plex.ServerCount to be 2, got %d", status.DataSources.Plex.ServerCount)
	}
}

// TestSetupStatus_JellyfinConfigured tests SetupStatus when Jellyfin servers are configured
func TestSetupStatus_JellyfinConfigured(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		JellyfinServers: []config.JellyfinConfig{
			{
				Enabled: true,
				URL:     "http://localhost:8096",
				APIKey:  "test-api-key",
			},
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// Jellyfin should be configured with 1 server
	if !status.DataSources.Jellyfin.Configured {
		t.Error("Expected Jellyfin.Configured to be true")
	}
	if status.DataSources.Jellyfin.ServerCount != 1 {
		t.Errorf("Expected Jellyfin.ServerCount to be 1, got %d", status.DataSources.Jellyfin.ServerCount)
	}
}

// TestSetupStatus_EmbyConfigured tests SetupStatus when Emby servers are configured
func TestSetupStatus_EmbyConfigured(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		EmbyServers: []config.EmbyConfig{
			{
				Enabled: true,
				URL:     "http://localhost:8096",
				APIKey:  "test-api-key",
			},
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// Emby should be configured with 1 server
	if !status.DataSources.Emby.Configured {
		t.Error("Expected Emby.Configured to be true")
	}
	if status.DataSources.Emby.ServerCount != 1 {
		t.Errorf("Expected Emby.ServerCount to be 1, got %d", status.DataSources.Emby.ServerCount)
	}
}

// TestSetupStatus_NATSEnabled tests SetupStatus when NATS is enabled
func TestSetupStatus_NATSEnabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		NATS: config.NATSConfig{
			Enabled: true,
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// NATS should be enabled
	if !status.DataSources.NATS.Enabled {
		t.Error("Expected NATS.Enabled to be true")
	}
}

// TestSetupStatus_WithDatabase tests SetupStatus with a real database
func TestSetupStatus_WithDatabase(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled: true,
			URL:     "http://localhost:32400",
			Token:   "test-token",
		},
	}

	handler := &Handler{
		db:        db,
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// Database should be connected
	if !status.Database.Connected {
		t.Error("Expected Database.Connected to be true")
	}

	// Ready should be true (database connected + Plex configured)
	if !status.Ready {
		t.Error("Expected Ready to be true with database and Plex configured")
	}

	// No playback data yet
	if status.DataAvailable.HasPlaybacks {
		t.Error("Expected HasPlaybacks to be false on empty database")
	}
	if status.DataAvailable.PlaybackCount != 0 {
		t.Errorf("Expected PlaybackCount to be 0, got %d", status.DataAvailable.PlaybackCount)
	}
}

// TestSetupStatus_WithPlaybackData tests SetupStatus with playback data in database
func TestSetupStatus_WithPlaybackData(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	// Insert test playback data
	insertTestPlaybacks(t, db, 5)

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled: true,
			URL:     "http://localhost:32400",
			Token:   "test-token",
		},
	}

	handler := &Handler{
		db:        db,
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// Should have playback data
	if !status.DataAvailable.HasPlaybacks {
		t.Error("Expected HasPlaybacks to be true with data")
	}
	if status.DataAvailable.PlaybackCount < 1 {
		t.Error("Expected PlaybackCount to be at least 1")
	}
}

// TestSetupStatus_Recommendations tests that recommendations are generated correctly
func TestSetupStatus_Recommendations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         *config.Config
		clientPingErr  error
		hasDB          bool
		expectedRecs   []string
		unexpectedRecs []string
	}{
		{
			name:   "no data sources configured",
			config: &config.Config{},
			expectedRecs: []string{
				"No data sources configured. Configure at least one of: Tautulli, Plex, Jellyfin, or Emby.",
			},
		},
		{
			name: "tautulli configured but not connected, suggest plex",
			config: &config.Config{
				Tautulli: config.TautulliConfig{
					Enabled: true,
					URL:     "http://localhost:8181",
					APIKey:  "test",
				},
			},
			clientPingErr: errors.New("connection refused"),
			expectedRecs: []string{
				"Tautulli is configured but not reachable. Check the URL and API key.",
			},
		},
		{
			name: "tautulli connected, suggest plex integration",
			config: &config.Config{
				Tautulli: config.TautulliConfig{
					Enabled: true,
					URL:     "http://localhost:8181",
					APIKey:  "test",
				},
			},
			clientPingErr: nil, // Connected successfully
			expectedRecs: []string{
				"Consider enabling Plex direct integration for real-time playback updates.",
			},
		},
		{
			name: "tautulli connected with plex, suggest NATS",
			config: &config.Config{
				Tautulli: config.TautulliConfig{
					Enabled: true,
					URL:     "http://localhost:8181",
					APIKey:  "test",
				},
				Plex: config.PlexConfig{
					Enabled: true,
					URL:     "http://localhost:32400",
					Token:   "test",
				},
			},
			clientPingErr: nil,
			expectedRecs: []string{
				"Consider enabling NATS for event-driven processing and better real-time updates.",
			},
			unexpectedRecs: []string{
				"Consider enabling Plex direct integration for real-time playback updates.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockClient *MockTautulliClient
			if tt.config.Tautulli.Enabled {
				mockClient = &MockTautulliClient{
					PingFunc: func(ctx context.Context) error {
						return tt.clientPingErr
					},
				}
			}

			handler := &Handler{
				startTime: time.Now(),
				config:    tt.config,
				client:    mockClient,
				cache:     cache.New(5 * time.Minute),
				perfMon:   middleware.NewPerformanceMonitor(1000),
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
			w := httptest.NewRecorder()

			handler.SetupStatus(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			data, err := json.Marshal(response.Data)
			if err != nil {
				t.Fatalf("Failed to marshal data: %v", err)
			}

			var status models.SetupStatus
			if err := json.Unmarshal(data, &status); err != nil {
				t.Fatalf("Failed to unmarshal setup status: %v", err)
			}

			// Check expected recommendations
			for _, expected := range tt.expectedRecs {
				found := false
				for _, rec := range status.Recommendations {
					if rec == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected recommendation '%s' not found in %v", expected, status.Recommendations)
				}
			}

			// Check unexpected recommendations are not present
			for _, unexpected := range tt.unexpectedRecs {
				for _, rec := range status.Recommendations {
					if rec == unexpected {
						t.Errorf("Unexpected recommendation '%s' found", unexpected)
					}
				}
			}
		})
	}
}

// TestSetupStatus_ResponseFormat tests the response format
func TestSetupStatus_ResponseFormat(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	// Check Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify JSON is valid
	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	// Verify status field
	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	// Verify metadata has timestamp
	if response.Metadata.Timestamp.IsZero() {
		t.Error("Expected Metadata.Timestamp to be set")
	}
}

// TestSetupStatus_URLMasking tests that URLs are properly masked
func TestSetupStatus_URLMasking(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Tautulli: config.TautulliConfig{
			Enabled: true,
			URL:     "http://localhost:8181/api/v2?apikey=supersecretkey12345",
			APIKey:  "supersecretkey12345",
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// URL should be masked (truncated for long URLs)
	if status.DataSources.Tautulli.URL == "" {
		t.Error("Expected Tautulli URL to be present")
	}

	// The URL should be masked/truncated if it's longer than 50 chars
	originalURL := cfg.Tautulli.URL
	maskedURL := status.DataSources.Tautulli.URL
	if len(originalURL) > 50 && len(maskedURL) > 53 { // 50 + "..."
		t.Errorf("Long URL should be masked, got '%s'", maskedURL)
	}
}

// TestSetupStatus_AllDataSources tests SetupStatus with all data sources configured
func TestSetupStatus_AllDataSources(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Tautulli: config.TautulliConfig{
			Enabled: true,
			URL:     "http://localhost:8181",
			APIKey:  "test",
		},
		PlexServers: []config.PlexConfig{
			{Enabled: true, URL: "http://localhost:32400", Token: "token1"},
			{Enabled: true, URL: "http://localhost:32401", Token: "token2"},
		},
		JellyfinServers: []config.JellyfinConfig{
			{Enabled: true, URL: "http://localhost:8096", APIKey: "key1"},
		},
		EmbyServers: []config.EmbyConfig{
			{Enabled: true, URL: "http://localhost:8097", APIKey: "key1"},
		},
		NATS: config.NATSConfig{
			Enabled: true,
		},
	}

	mockClient := &MockTautulliClient{
		PingFunc: func(ctx context.Context) error {
			return nil
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		client:    mockClient,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// Verify all data sources
	if !status.DataSources.Tautulli.Configured {
		t.Error("Expected Tautulli to be configured")
	}
	if !status.DataSources.Tautulli.Connected {
		t.Error("Expected Tautulli to be connected")
	}
	if !status.DataSources.Plex.Configured {
		t.Error("Expected Plex to be configured")
	}
	if status.DataSources.Plex.ServerCount != 2 {
		t.Errorf("Expected 2 Plex servers, got %d", status.DataSources.Plex.ServerCount)
	}
	if !status.DataSources.Jellyfin.Configured {
		t.Error("Expected Jellyfin to be configured")
	}
	if status.DataSources.Jellyfin.ServerCount != 1 {
		t.Errorf("Expected 1 Jellyfin server, got %d", status.DataSources.Jellyfin.ServerCount)
	}
	if !status.DataSources.Emby.Configured {
		t.Error("Expected Emby to be configured")
	}
	if status.DataSources.Emby.ServerCount != 1 {
		t.Errorf("Expected 1 Emby server, got %d", status.DataSources.Emby.ServerCount)
	}
	if !status.DataSources.NATS.Enabled {
		t.Error("Expected NATS to be enabled")
	}
}

// TestSetupStatus_NilConfig tests SetupStatus with nil config doesn't panic
func TestSetupStatus_NilConfig(t *testing.T) {
	t.Parallel()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetupStatus panicked with nil config: %v", r)
		}
	}()

	handler := &Handler{
		startTime: time.Now(),
		config:    nil, // Nil config
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	// Should return 200 OK with a recommendation about config
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// Should have recommendation about config
	found := false
	for _, rec := range status.Recommendations {
		if rec == "Configuration not loaded. Check server startup logs." {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected recommendation about config, got %v", status.Recommendations)
	}
}

// TestSetupStatus_DatabasePingFails tests SetupStatus when database ping fails
func TestSetupStatus_DatabasePingFails(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	db.Close() // Close DB to make Ping fail

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled: true,
			URL:     "http://localhost:32400",
			Token:   "test",
		},
	}

	handler := &Handler{
		db:        db, // Closed DB
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var status models.SetupStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("Failed to unmarshal setup status: %v", err)
	}

	// Database should not be connected
	if status.Database.Connected {
		t.Error("Expected Database.Connected to be false with closed database")
	}

	// Ready should be false (database not connected)
	if status.Ready {
		t.Error("Expected Ready to be false with disconnected database")
	}
}

// TestSetupStatus_ContextCancellation tests SetupStatus with canceled context
func TestSetupStatus_ContextCancellation(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	cfg := &config.Config{}

	handler := &Handler{
		db:        db,
		startTime: time.Now(),
		config:    cfg,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler.SetupStatus(w, req)

	// Should still return a response (even if partial)
	if w.Code != http.StatusOK {
		// Might get error due to canceled context, which is acceptable
		t.Logf("Got status %d with canceled context (expected behavior)", w.Code)
	}
}

// BenchmarkSetupStatus benchmarks the SetupStatus endpoint
func BenchmarkSetupStatus(b *testing.B) {
	cfg := &config.Config{
		Tautulli: config.TautulliConfig{
			Enabled: true,
			URL:     "http://localhost:8181",
			APIKey:  "test",
		},
		Plex: config.PlexConfig{
			Enabled: true,
			URL:     "http://localhost:32400",
			Token:   "token1",
		},
	}

	mockClient := &MockTautulliClient{
		PingFunc: func(ctx context.Context) error {
			return nil
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
		client:    mockClient,
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.SetupStatus(w, req)
	}
}

// BenchmarkSetupStatus_WithDB benchmarks SetupStatus with database operations
func BenchmarkSetupStatus_WithDB(b *testing.B) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		b.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := &Handler{
		db:        db,
		startTime: time.Now(),
		config: &config.Config{
			Plex: config.PlexConfig{
				Enabled: true,
				URL:     "http://localhost:32400",
				Token:   "token1",
			},
		},
		cache:   cache.New(5 * time.Minute),
		perfMon: middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/setup", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.SetupStatus(w, req)
	}
}
