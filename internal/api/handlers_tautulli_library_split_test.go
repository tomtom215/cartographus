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
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestTautulliLibraryUserStats_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetLibraryUserStatsFunc: func(ctx context.Context, sectionID int, grouping int) (*tautulli.TautulliLibraryUserStats, error) {
			return &tautulli.TautulliLibraryUserStats{
				Response: tautulli.TautulliLibraryUserStatsResponse{
					Result: "success",
					Data:   []tautulli.TautulliLibraryUserStatRow{},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/library-user-stats?section_id=1", nil)
	w := httptest.NewRecorder()

	handler.TautulliLibraryUserStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliRecentlyAdded_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetRecentlyAddedFunc: func(ctx context.Context, count int, start int, mediaType string, sectionID int) (*tautulli.TautulliRecentlyAdded, error) {
			return &tautulli.TautulliRecentlyAdded{
				Response: tautulli.TautulliRecentlyAddedResponse{
					Result: "success",
					Data: tautulli.TautulliRecentlyAddedData{
						RecordsTotal:  10,
						RecentlyAdded: []tautulli.TautulliRecentlyAddedItem{},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/recently-added?count=10", nil)
	w := httptest.NewRecorder()

	handler.TautulliRecentlyAdded(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliLibrary_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetLibraryFunc: func(ctx context.Context, sectionID int) (*tautulli.TautulliLibrary, error) {
			return &tautulli.TautulliLibrary{
				Response: tautulli.TautulliLibraryResponse{
					Result: "success",
					Data: tautulli.TautulliLibraryData{
						SectionID:   sectionID,
						SectionName: "Movies",
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/library?section_id=1", nil)
	w := httptest.NewRecorder()

	handler.TautulliLibrary(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliLibrariesTable_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetLibrariesTableFunc: func(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliLibrariesTable, error) {
			return &tautulli.TautulliLibrariesTable{
				Response: tautulli.TautulliLibrariesTableResponse{
					Result: "success",
					Data: tautulli.TautulliLibrariesTableData{
						RecordsTotal: 2,
						Data: []tautulli.TautulliLibrariesTableRow{
							{SectionID: 1, SectionName: "Movies", SectionType: "movie", Count: 100},
							{SectionID: 2, SectionName: "TV Shows", SectionType: "show", Count: 50},
						},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/libraries-table", nil)
	w := httptest.NewRecorder()

	handler.TautulliLibrariesTable(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliLibraryMediaInfo_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetLibraryMediaInfoFunc: func(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int) (*tautulli.TautulliLibraryMediaInfo, error) {
			return &tautulli.TautulliLibraryMediaInfo{
				Response: tautulli.TautulliLibraryMediaInfoResponse{
					Result: "success",
					Data: tautulli.TautulliLibraryMediaInfoData{
						RecordsTotal: 1,
						Data: []tautulli.TautulliLibraryMediaInfoRow{
							{RatingKey: "100", Title: "Test Movie", Year: 2025, Container: "mkv"},
						},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/library-media-info?section_id=1", nil)
	w := httptest.NewRecorder()

	handler.TautulliLibraryMediaInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliLibraryWatchTimeStats_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetLibraryWatchTimeStatsFunc: func(ctx context.Context, sectionID int, grouping int, queryDays string) (*tautulli.TautulliLibraryWatchTimeStats, error) {
			return &tautulli.TautulliLibraryWatchTimeStats{
				Response: tautulli.TautulliLibraryWatchTimeStatsResponse{
					Result: "success",
					Data: []tautulli.TautulliLibraryWatchTimeStatRow{
						{QueryDays: 7, TotalTime: 7200, TotalPlays: 20},
						{QueryDays: 30, TotalTime: 28800, TotalPlays: 80},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/library-watch-time-stats?section_id=1", nil)
	w := httptest.NewRecorder()

	handler.TautulliLibraryWatchTimeStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliChildrenMetadata_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetChildrenMetadataFunc: func(ctx context.Context, ratingKey string, mediaType string) (*tautulli.TautulliChildrenMetadata, error) {
			return &tautulli.TautulliChildrenMetadata{
				Response: tautulli.TautulliChildrenMetadataResponse{
					Result: "success",
					Data: tautulli.TautulliChildrenMetadataData{
						ChildrenCount: 2,
						ChildrenList: []tautulli.TautulliChildrenMetadataChild{
							{RatingKey: "123", Title: "Episode 1", MediaIndex: 1},
							{RatingKey: "124", Title: "Episode 2", MediaIndex: 2},
						},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/children-metadata?rating_key=12345", nil)
	w := httptest.NewRecorder()

	handler.TautulliChildrenMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliLibraryNames_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetLibraryNamesFunc: func(ctx context.Context) (*tautulli.TautulliLibraryNames, error) {
			return &tautulli.TautulliLibraryNames{
				Response: tautulli.TautulliLibraryNamesResponse{
					Result: "success",
					Data: []tautulli.TautulliLibraryNameItem{
						{
							SectionID:   1,
							SectionName: "Movies",
							SectionType: "movie",
						},
						{
							SectionID:   2,
							SectionName: "TV Shows",
							SectionType: "show",
						},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/library-names", nil)
	w := httptest.NewRecorder()

	handler.TautulliLibraryNames(w, req)

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
}
