// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestTautulliClient_GetLibraryUserStats(t *testing.T) {
	tests := []struct {
		name        string
		sectionID   int
		grouping    int
		expectError bool
	}{
		{
			name:        "valid section with grouping",
			sectionID:   1,
			grouping:    1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cmd") != "get_library_user_stats" {
					t.Errorf("Expected cmd=get_library_user_stats, got %s", r.URL.Query().Get("cmd"))
				}
				mockData := tautulli.TautulliLibraryUserStats{
					Response: tautulli.TautulliLibraryUserStatsResponse{
						Result: "success",
						Data:   []tautulli.TautulliLibraryUserStatRow{},
					},
				}
				json.NewEncoder(w).Encode(mockData)
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
			client := NewTautulliClient(cfg)

			stats, err := client.GetLibraryUserStats(context.Background(), tt.sectionID, tt.grouping)
			if (err != nil) != tt.expectError {
				t.Errorf("GetLibraryUserStats() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && stats == nil {
				t.Error("Expected non-nil stats response")
			}
		})
	}
}

func TestTautulliClient_GetRecentlyAdded(t *testing.T) {
	tests := []struct {
		name        string
		count       int
		start       int
		mediaType   string
		sectionID   int
		expectError bool
	}{
		{
			name:        "get recent movies",
			count:       10,
			start:       0,
			mediaType:   "movie",
			sectionID:   1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cmd") != "get_recently_added" {
					t.Errorf("Expected cmd=get_recently_added, got %s", r.URL.Query().Get("cmd"))
				}
				mockData := tautulli.TautulliRecentlyAdded{
					Response: tautulli.TautulliRecentlyAddedResponse{
						Result: "success",
						Data: tautulli.TautulliRecentlyAddedData{
							RecentlyAdded: []tautulli.TautulliRecentlyAddedItem{},
						},
					},
				}
				json.NewEncoder(w).Encode(mockData)
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
			client := NewTautulliClient(cfg)

			items, err := client.GetRecentlyAdded(context.Background(), tt.count, tt.start, tt.mediaType, tt.sectionID)
			if (err != nil) != tt.expectError {
				t.Errorf("GetRecentlyAdded() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && items == nil {
				t.Error("Expected non-nil items response")
			}
		})
	}
}

func TestTautulliClient_GetLibraries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_libraries" {
			t.Errorf("Expected cmd=get_libraries, got %s", r.URL.Query().Get("cmd"))
		}
		mockData := tautulli.TautulliLibraries{
			Response: tautulli.TautulliLibrariesResponse{
				Result: "success",
				Data:   []tautulli.TautulliLibraryDetail{},
			},
		}
		json.NewEncoder(w).Encode(mockData)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	libraries, err := client.GetLibraries(context.Background())
	if err != nil {
		t.Errorf("GetLibraries() error = %v", err)
	}
	if libraries == nil {
		t.Error("Expected non-nil libraries response")
	}
}

func TestTautulliClient_GetLibrary(t *testing.T) {
	sectionID := 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_library" {
			t.Errorf("Expected cmd=get_library, got %s", r.URL.Query().Get("cmd"))
		}
		mockData := tautulli.TautulliLibrary{
			Response: tautulli.TautulliLibraryResponse{
				Result: "success",
				Data: tautulli.TautulliLibraryData{
					SectionID:   sectionID,
					SectionName: "Movies",
					SectionType: "movie",
					Count:       150,
				},
			},
		}
		json.NewEncoder(w).Encode(mockData)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	library, err := client.GetLibrary(context.Background(), sectionID)
	if err != nil {
		t.Errorf("GetLibrary() error = %v", err)
	}
	if library.Response.Data.SectionName != "Movies" {
		t.Errorf("Expected section_name='Movies', got '%s'", library.Response.Data.SectionName)
	}
}

func TestTautulliClient_GetLibrariesTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_libraries_table" {
			t.Errorf("Expected cmd=get_libraries_table, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliLibrariesTable{
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
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetLibrariesTable(context.Background(), 0, "section_name", "asc", 0, 25, "")
	if err != nil {
		t.Fatalf("GetLibrariesTable() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data.Data) != 2 {
		t.Errorf("Expected 2 libraries, got %d", len(stats.Response.Data.Data))
	}
}

func TestTautulliClient_GetLibraryMediaInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_library_media_info" {
			t.Errorf("Expected cmd=get_library_media_info, got %s", r.URL.Query().Get("cmd"))
		}
		if r.URL.Query().Get("section_id") != "1" {
			t.Errorf("Expected section_id=1, got %s", r.URL.Query().Get("section_id"))
		}

		response := tautulli.TautulliLibraryMediaInfo{
			Response: tautulli.TautulliLibraryMediaInfoResponse{
				Result: "success",
				Data: tautulli.TautulliLibraryMediaInfoData{
					RecordsTotal: 1,
					Data: []tautulli.TautulliLibraryMediaInfoRow{
						{RatingKey: "100", Title: "Test Movie", Year: 2025, Container: "mkv"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetLibraryMediaInfo(context.Background(), 1, "title", "asc", 0, 25)
	if err != nil {
		t.Fatalf("GetLibraryMediaInfo() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data.Data) != 1 {
		t.Errorf("Expected 1 media item, got %d", len(stats.Response.Data.Data))
	}
}

func TestTautulliClient_GetLibraryWatchTimeStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_library_watch_time_stats" {
			t.Errorf("Expected cmd=get_library_watch_time_stats, got %s", r.URL.Query().Get("cmd"))
		}
		if r.URL.Query().Get("section_id") != "1" {
			t.Errorf("Expected section_id=1, got %s", r.URL.Query().Get("section_id"))
		}

		response := tautulli.TautulliLibraryWatchTimeStats{
			Response: tautulli.TautulliLibraryWatchTimeStatsResponse{
				Result: "success",
				Data: []tautulli.TautulliLibraryWatchTimeStatRow{
					{QueryDays: 7, TotalTime: 7200, TotalPlays: 20},
					{QueryDays: 30, TotalTime: 28800, TotalPlays: 80},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetLibraryWatchTimeStats(context.Background(), 1, 0, "")
	if err != nil {
		t.Fatalf("GetLibraryWatchTimeStats() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data) != 2 {
		t.Errorf("Expected 2 time periods, got %d", len(stats.Response.Data))
	}
}

func TestTautulliClient_GetChildrenMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_children_metadata" {
			t.Errorf("Expected cmd=get_children_metadata, got %s", r.URL.Query().Get("cmd"))
		}
		if r.URL.Query().Get("rating_key") != "12345" {
			t.Errorf("Expected rating_key=12345, got %s", r.URL.Query().Get("rating_key"))
		}

		response := tautulli.TautulliChildrenMetadata{
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
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetChildrenMetadata(context.Background(), "12345", "season")
	if err != nil {
		t.Fatalf("GetChildrenMetadata() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data.ChildrenList) != 2 {
		t.Errorf("Expected 2 children, got %d", len(stats.Response.Data.ChildrenList))
	}
}

func TestTautulliClient_GetLibraryNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_library_names" {
			t.Errorf("Expected cmd=get_library_names, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliLibraryNames{
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
					{
						SectionID:   3,
						SectionName: "Music",
						SectionType: "artist",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	libraryNames, err := client.GetLibraryNames(context.Background())
	if err != nil {
		t.Fatalf("GetLibraryNames() error = %v", err)
	}
	if libraryNames.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", libraryNames.Response.Result)
	}
	if len(libraryNames.Response.Data) != 3 {
		t.Errorf("Expected 3 libraries, got %d", len(libraryNames.Response.Data))
	}
	if libraryNames.Response.Data[0].SectionName != "Movies" {
		t.Errorf("Expected SectionName='Movies', got '%s'", libraryNames.Response.Data[0].SectionName)
	}
	if libraryNames.Response.Data[1].SectionType != "show" {
		t.Errorf("Expected SectionType='show', got '%s'", libraryNames.Response.Data[1].SectionType)
	}
	if libraryNames.Response.Data[2].SectionID != 3 {
		t.Errorf("Expected SectionID=3, got %d", libraryNames.Response.Data[2].SectionID)
	}
}
