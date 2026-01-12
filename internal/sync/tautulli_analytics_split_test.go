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

func TestTautulliClient_GetHomeStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_home_stats" {
			t.Errorf("Expected cmd=get_home_stats, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliHomeStats{
			Response: tautulli.TautulliHomeStatsResponse{
				Result: "success",
				Data: []tautulli.TautulliHomeStatRow{
					{
						StatID:    "top_movies",
						StatType:  "top_movies",
						StatTitle: "Most Watched Movies",
						Rows: []tautulli.TautulliHomeStatDetail{
							{Title: "Test Movie", TotalPlays: 10},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetHomeStats(context.Background(), 30, "top_movies", 10)
	if err != nil {
		t.Fatalf("GetHomeStats() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data) != 1 {
		t.Errorf("Expected 1 stat row, got %d", len(stats.Response.Data))
	}
}

func TestTautulliClient_GetPlaysByDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_plays_by_date" {
			t.Errorf("Expected cmd=get_plays_by_date, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliPlaysByDate{
			Response: tautulli.TautulliPlaysByDateResponse{
				Result: "success",
				Data: tautulli.TautulliPlaysByDateData{
					Categories: []string{"2025-01-01", "2025-01-02"},
					Series: []tautulli.TautulliPlaysByDateSeries{
						{Name: "Movies", Data: []interface{}{10, 15}},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	plays, err := client.GetPlaysByDate(context.Background(), 30, "plays", 0, 0)
	if err != nil {
		t.Fatalf("GetPlaysByDate() error = %v", err)
	}
	if plays.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", plays.Response.Result)
	}
	if len(plays.Response.Data.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(plays.Response.Data.Categories))
	}
}

func TestTautulliClient_GetPlaysByDayOfWeek(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_plays_by_dayofweek" {
			t.Errorf("Expected cmd=get_plays_by_dayofweek, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliPlaysByDayOfWeek{
			Response: tautulli.TautulliPlaysByDayOfWeekResponse{
				Result: "success",
				Data: tautulli.TautulliPlaysByDayOfWeekData{
					Categories: []string{"Sunday", "Monday", "Tuesday"},
					Series: []tautulli.TautulliPlaysByDateSeries{
						{Name: "Movies", Data: []interface{}{5, 10, 8}},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	plays, err := client.GetPlaysByDayOfWeek(context.Background(), 30, "plays", 0, 0)
	if err != nil {
		t.Fatalf("GetPlaysByDayOfWeek() error = %v", err)
	}
	if plays.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", plays.Response.Result)
	}
	if len(plays.Response.Data.Categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(plays.Response.Data.Categories))
	}
}

func TestTautulliClient_GetPlaysByHourOfDay(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_plays_by_hourofday" {
			t.Errorf("Expected cmd=get_plays_by_hourofday, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliPlaysByHourOfDay{
			Response: tautulli.TautulliPlaysByHourOfDayResponse{
				Result: "success",
				Data: tautulli.TautulliPlaysByHourOfDayData{
					Categories: []string{"00", "01", "02"},
					Series: []tautulli.TautulliPlaysByDateSeries{
						{Name: "Movies", Data: []interface{}{2, 1, 0}},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	plays, err := client.GetPlaysByHourOfDay(context.Background(), 30, "plays", 0, 0)
	if err != nil {
		t.Fatalf("GetPlaysByHourOfDay() error = %v", err)
	}
	if plays.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", plays.Response.Result)
	}
	if len(plays.Response.Data.Categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(plays.Response.Data.Categories))
	}
}

func TestTautulliClient_GetPlaysByStreamType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_plays_by_stream_type" {
			t.Errorf("Expected cmd=get_plays_by_stream_type, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliPlaysByStreamType{
			Response: tautulli.TautulliPlaysByStreamTypeResponse{
				Result: "success",
				Data: tautulli.TautulliPlaysByStreamTypeData{
					Categories: []string{"2025-01-01", "2025-01-02"},
					Series: []tautulli.TautulliStreamTypeSeries{
						{Name: "Direct Play", Data: []interface{}{8, 12}},
						{Name: "Transcode", Data: []interface{}{2, 3}},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	plays, err := client.GetPlaysByStreamType(context.Background(), 30, "plays", 0, 0)
	if err != nil {
		t.Fatalf("GetPlaysByStreamType() error = %v", err)
	}
	if plays.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", plays.Response.Result)
	}
	if len(plays.Response.Data.Series) != 2 {
		t.Errorf("Expected 2 series, got %d", len(plays.Response.Data.Series))
	}
}

func TestTautulliClient_GetConcurrentStreamsByStreamType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_concurrent_streams_by_stream_type" {
			t.Errorf("Expected cmd=get_concurrent_streams_by_stream_type, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliConcurrentStreamsByStreamType{
			Response: tautulli.TautulliConcurrentStreamsByStreamTypeResponse{
				Result: "success",
				Data: tautulli.TautulliConcurrentStreamsByStreamTypeData{
					Categories: []string{"2025-01-01", "2025-01-02"},
					Series: []tautulli.TautulliStreamTypeSeries{
						{Name: "Total Concurrent Streams", Data: []interface{}{5, 7}},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	streams, err := client.GetConcurrentStreamsByStreamType(context.Background(), 30, 0)
	if err != nil {
		t.Fatalf("GetConcurrentStreamsByStreamType() error = %v", err)
	}
	if streams.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", streams.Response.Result)
	}
}

func TestTautulliClient_GetItemWatchTimeStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_item_watch_time_stats" {
			t.Errorf("Expected cmd=get_item_watch_time_stats, got %s", r.URL.Query().Get("cmd"))
		}
		if r.URL.Query().Get("rating_key") != "12345" {
			t.Errorf("Expected rating_key=12345, got %s", r.URL.Query().Get("rating_key"))
		}

		response := tautulli.TautulliItemWatchTimeStats{
			Response: tautulli.TautulliItemWatchTimeStatsResponse{
				Result: "success",
				Data: []tautulli.TautulliItemWatchTimeDetail{
					{QueryDays: "7", TotalTime: 3600, TotalPlays: 10},
					{QueryDays: "30", TotalTime: 7200, TotalPlays: 25},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetItemWatchTimeStats(context.Background(), "12345", 0, "")
	if err != nil {
		t.Fatalf("GetItemWatchTimeStats() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data) != 2 {
		t.Errorf("Expected 2 time periods, got %d", len(stats.Response.Data))
	}
}

// Priority 1: Analytics Dashboard Completion Tests (8 endpoints)

func TestTautulliClient_GetPlaysBySourceResolution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_plays_by_source_resolution" {
			t.Errorf("Expected cmd=get_plays_by_source_resolution, got %s", r.URL.Query().Get("cmd"))
		}
		if r.URL.Query().Get("time_range") != "30" {
			t.Errorf("Expected time_range=30, got %s", r.URL.Query().Get("time_range"))
		}

		response := tautulli.TautulliPlaysBySourceResolution{
			Response: tautulli.TautulliPlaysBySourceResolutionResponse{
				Result: "success",
				Data: tautulli.TautulliPlaysBySourceResolutionData{
					Categories: []string{"4K", "1080p", "720p"},
					Series:     []tautulli.TautulliPlaysByDateSeries{{Name: "Plays", Data: []interface{}{10, 50, 30}}},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetPlaysBySourceResolution(context.Background(), 30, "plays", 0, 0)
	if err != nil {
		t.Fatalf("GetPlaysBySourceResolution() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data.Categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(stats.Response.Data.Categories))
	}
}

func TestTautulliClient_GetPlaysByStreamResolution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_plays_by_stream_resolution" {
			t.Errorf("Expected cmd=get_plays_by_stream_resolution, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliPlaysByStreamResolution{
			Response: tautulli.TautulliPlaysByStreamResolutionResponse{
				Result: "success",
				Data: tautulli.TautulliPlaysByStreamResolutionData{
					Categories: []string{"4K", "1080p", "720p", "SD"},
					Series:     []tautulli.TautulliPlaysByDateSeries{{Name: "Streams", Data: []interface{}{5, 40, 45, 10}}},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetPlaysByStreamResolution(context.Background(), 30, "plays", 0, 0)
	if err != nil {
		t.Fatalf("GetPlaysByStreamResolution() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data.Categories) != 4 {
		t.Errorf("Expected 4 categories, got %d", len(stats.Response.Data.Categories))
	}
}

func TestTautulliClient_GetPlaysByTop10Platforms(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_plays_by_top_10_platforms" {
			t.Errorf("Expected cmd=get_plays_by_top_10_platforms, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliPlaysByTop10Platforms{
			Response: tautulli.TautulliPlaysByTop10PlatformsResponse{
				Result: "success",
				Data: tautulli.TautulliPlaysByTop10PlatformsData{
					Categories: []string{"Chrome", "Roku", "Apple TV"},
					Series:     []tautulli.TautulliPlaysByDateSeries{{Name: "Plays", Data: []interface{}{100, 80, 60}}},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetPlaysByTop10Platforms(context.Background(), 30, "plays", 0, 0)
	if err != nil {
		t.Fatalf("GetPlaysByTop10Platforms() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data.Categories) != 3 {
		t.Errorf("Expected 3 platforms, got %d", len(stats.Response.Data.Categories))
	}
}

func TestTautulliClient_GetPlaysByTop10Users(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_plays_by_top_10_users" {
			t.Errorf("Expected cmd=get_plays_by_top_10_users, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliPlaysByTop10Users{
			Response: tautulli.TautulliPlaysByTop10UsersResponse{
				Result: "success",
				Data: tautulli.TautulliPlaysByTop10UsersData{
					Categories: []string{"user1", "user2", "user3"},
					Series:     []tautulli.TautulliPlaysByDateSeries{{Name: "Plays", Data: []interface{}{150, 120, 90}}},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetPlaysByTop10Users(context.Background(), 30, "plays", 0, 0)
	if err != nil {
		t.Fatalf("GetPlaysByTop10Users() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data.Categories) != 3 {
		t.Errorf("Expected 3 users, got %d", len(stats.Response.Data.Categories))
	}
}

func TestTautulliClient_GetPlaysPerMonth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_plays_per_month" {
			t.Errorf("Expected cmd=get_plays_per_month, got %s", r.URL.Query().Get("cmd"))
		}

		response := tautulli.TautulliPlaysPerMonth{
			Response: tautulli.TautulliPlaysPerMonthResponse{
				Result: "success",
				Data: tautulli.TautulliPlaysPerMonthData{
					Categories: []string{"2025-01", "2025-02", "2025-03"},
					Series:     []tautulli.TautulliPlaysByDateSeries{{Name: "Plays", Data: []interface{}{100, 150, 200}}},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetPlaysPerMonth(context.Background(), 365, "plays", 0, 0)
	if err != nil {
		t.Fatalf("GetPlaysPerMonth() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data.Categories) != 3 {
		t.Errorf("Expected 3 months, got %d", len(stats.Response.Data.Categories))
	}
}

func TestTautulliClient_GetItemUserStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_item_user_stats" {
			t.Errorf("Expected cmd=get_item_user_stats, got %s", r.URL.Query().Get("cmd"))
		}
		if r.URL.Query().Get("rating_key") != "12345" {
			t.Errorf("Expected rating_key=12345, got %s", r.URL.Query().Get("rating_key"))
		}

		response := tautulli.TautulliItemUserStats{
			Response: tautulli.TautulliItemUserStatsResponse{
				Result: "success",
				Data: []tautulli.TautulliItemUserStatRow{
					{UserID: 1, Username: "user1", TotalPlays: 5},
					{UserID: 2, Username: "user2", TotalPlays: 3},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	stats, err := client.GetItemUserStats(context.Background(), "12345", 0)
	if err != nil {
		t.Fatalf("GetItemUserStats() error = %v", err)
	}
	if stats.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", stats.Response.Result)
	}
	if len(stats.Response.Data) != 2 {
		t.Errorf("Expected 2 users, got %d", len(stats.Response.Data))
	}
}

// Priority 2: Library-Specific Analytics Tests (4 endpoints)
