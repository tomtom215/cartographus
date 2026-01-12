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

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestTautulliPlaysBySourceResolution_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetPlaysBySourceResolutionFunc: func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysBySourceResolution, error) {
			return &tautulli.TautulliPlaysBySourceResolution{
				Response: tautulli.TautulliPlaysBySourceResolutionResponse{
					Result: "success",
					Data: tautulli.TautulliPlaysBySourceResolutionData{
						Categories: []string{"4K", "1080p", "720p"},
						Series:     []tautulli.TautulliPlaysByDateSeries{{Name: "Plays", Data: []interface{}{10, 50, 30}}},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/plays-by-source-resolution?time_range=30", nil)
	w := httptest.NewRecorder()

	handler.TautulliPlaysBySourceResolution(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliPlaysByStreamResolution_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetPlaysByStreamResolutionFunc: func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamResolution, error) {
			return &tautulli.TautulliPlaysByStreamResolution{
				Response: tautulli.TautulliPlaysByStreamResolutionResponse{
					Result: "success",
					Data: tautulli.TautulliPlaysByStreamResolutionData{
						Categories: []string{"4K", "1080p", "720p", "SD"},
						Series:     []tautulli.TautulliPlaysByDateSeries{{Name: "Streams", Data: []interface{}{5, 40, 45, 10}}},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/plays-by-stream-resolution?time_range=30", nil)
	w := httptest.NewRecorder()

	handler.TautulliPlaysByStreamResolution(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliPlaysByTop10Platforms_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetPlaysByTop10PlatformsFunc: func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Platforms, error) {
			return &tautulli.TautulliPlaysByTop10Platforms{
				Response: tautulli.TautulliPlaysByTop10PlatformsResponse{
					Result: "success",
					Data: tautulli.TautulliPlaysByTop10PlatformsData{
						Categories: []string{"Chrome", "Roku", "Apple TV"},
						Series:     []tautulli.TautulliPlaysByDateSeries{{Name: "Plays", Data: []interface{}{100, 80, 60}}},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/plays-by-top-10-platforms?time_range=30", nil)
	w := httptest.NewRecorder()

	handler.TautulliPlaysByTop10Platforms(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliPlaysByTop10Users_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetPlaysByTop10UsersFunc: func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Users, error) {
			return &tautulli.TautulliPlaysByTop10Users{
				Response: tautulli.TautulliPlaysByTop10UsersResponse{
					Result: "success",
					Data: tautulli.TautulliPlaysByTop10UsersData{
						Categories: []string{"user1", "user2", "user3"},
						Series:     []tautulli.TautulliPlaysByDateSeries{{Name: "Plays", Data: []interface{}{150, 120, 90}}},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/plays-by-top-10-users?time_range=30", nil)
	w := httptest.NewRecorder()

	handler.TautulliPlaysByTop10Users(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliPlaysPerMonth_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetPlaysPerMonthFunc: func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysPerMonth, error) {
			return &tautulli.TautulliPlaysPerMonth{
				Response: tautulli.TautulliPlaysPerMonthResponse{
					Result: "success",
					Data: tautulli.TautulliPlaysPerMonthData{
						Categories: []string{"2025-01", "2025-02", "2025-03"},
						Series:     []tautulli.TautulliPlaysByDateSeries{{Name: "Plays", Data: []interface{}{100, 150, 200}}},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/plays-per-month?time_range=365", nil)
	w := httptest.NewRecorder()

	handler.TautulliPlaysPerMonth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
