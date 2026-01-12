// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliHomeStats_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"message": null,
			"data": [
				{
					"stat_id": "top_movies",
					"stat_type": "top_movies",
					"stat_title": "Most Watched Movies",
					"rows": [
						{
							"title": "Inception",
							"total_plays": 150,
							"total_duration": 540000,
							"users_watched": 25,
							"rating_key": "12345",
							"last_play": 1700000000,
							"media_type": "movie",
							"library_name": "Movies",
							"section_id": 1,
							"thumb": "/library/metadata/12345/thumb",
							"year": 2010
						}
					],
					"rows_for_media": true,
					"total_duration": 540000,
					"total_plays": 150
				}
			]
		}
	}`

	var stats TautulliHomeStats
	err := json.Unmarshal([]byte(jsonData), &stats)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliHomeStats: %v", err)
	}

	if stats.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", stats.Response.Result)
	}

	if len(stats.Response.Data) != 1 {
		t.Fatalf("Expected 1 stat row, got %d", len(stats.Response.Data))
	}

	statRow := stats.Response.Data[0]
	if statRow.StatID != "top_movies" {
		t.Errorf("Expected StatID 'top_movies', got '%s'", statRow.StatID)
	}
	if statRow.StatType != "top_movies" {
		t.Errorf("Expected StatType 'top_movies', got '%s'", statRow.StatType)
	}
	if statRow.StatTitle != "Most Watched Movies" {
		t.Errorf("Expected StatTitle 'Most Watched Movies', got '%s'", statRow.StatTitle)
	}
	if !statRow.RowsForMedia {
		t.Error("Expected RowsForMedia to be true")
	}
	if statRow.TotalDuration != 540000 {
		t.Errorf("Expected TotalDuration 540000, got %d", statRow.TotalDuration)
	}
	if statRow.TotalPlays != 150 {
		t.Errorf("Expected TotalPlays 150, got %d", statRow.TotalPlays)
	}

	if len(statRow.Rows) != 1 {
		t.Fatalf("Expected 1 detail row, got %d", len(statRow.Rows))
	}

	detail := statRow.Rows[0]
	if detail.Title != "Inception" {
		t.Errorf("Expected Title 'Inception', got '%s'", detail.Title)
	}
	if detail.TotalPlays != 150 {
		t.Errorf("Expected TotalPlays 150, got %d", detail.TotalPlays)
	}
	if detail.TotalDuration != 540000 {
		t.Errorf("Expected TotalDuration 540000, got %d", detail.TotalDuration)
	}
	if detail.UsersWatched != 25 {
		t.Errorf("Expected UsersWatched 25, got %d", detail.UsersWatched)
	}
	if detail.RatingKey != "12345" {
		t.Errorf("Expected RatingKey '12345', got '%s'", detail.RatingKey)
	}
	if detail.Year != 2010 {
		t.Errorf("Expected Year 2010, got %d", detail.Year)
	}
}

func TestTautulliHomeStats_TopUsers(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": [
				{
					"stat_id": "top_users",
					"stat_type": "top_users",
					"stat_title": "Most Active Users",
					"rows": [
						{
							"user": "john_doe",
							"friendly_name": "John Doe",
							"total_plays": 500,
							"total_duration": 1800000,
							"platform_name": "Roku",
							"platform_type": "streaming_device",
							"player_name": "Living Room Roku"
						}
					],
					"rows_for_user": true
				}
			]
		}
	}`

	var stats TautulliHomeStats
	err := json.Unmarshal([]byte(jsonData), &stats)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliHomeStats: %v", err)
	}

	statRow := stats.Response.Data[0]
	if statRow.StatID != "top_users" {
		t.Errorf("Expected StatID 'top_users', got '%s'", statRow.StatID)
	}
	if !statRow.RowsForUser {
		t.Error("Expected RowsForUser to be true")
	}

	detail := statRow.Rows[0]
	if detail.User != "john_doe" {
		t.Errorf("Expected User 'john_doe', got '%s'", detail.User)
	}
	if detail.FriendlyName != "John Doe" {
		t.Errorf("Expected FriendlyName 'John Doe', got '%s'", detail.FriendlyName)
	}
	if detail.Platform != "Roku" {
		t.Errorf("Expected Platform 'Roku', got '%s'", detail.Platform)
	}
	if detail.PlatformType != "streaming_device" {
		t.Errorf("Expected PlatformType 'streaming_device', got '%s'", detail.PlatformType)
	}
	if detail.PlayerName != "Living Room Roku" {
		t.Errorf("Expected PlayerName 'Living Room Roku', got '%s'", detail.PlayerName)
	}
}

func TestTautulliHomeStats_EmptyData(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": []
		}
	}`

	var stats TautulliHomeStats
	err := json.Unmarshal([]byte(jsonData), &stats)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliHomeStats: %v", err)
	}

	if len(stats.Response.Data) != 0 {
		t.Errorf("Expected 0 data rows, got %d", len(stats.Response.Data))
	}
}

func TestTautulliHomeStats_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Invalid API key",
			"data": []
		}
	}`

	var stats TautulliHomeStats
	err := json.Unmarshal([]byte(jsonData), &stats)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliHomeStats: %v", err)
	}

	if stats.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", stats.Response.Result)
	}
	if stats.Response.Message == nil || *stats.Response.Message != "Invalid API key" {
		t.Error("Expected Message 'Invalid API key'")
	}
}

func TestTautulliHomeStats_JSONRoundTrip(t *testing.T) {
	original := TautulliHomeStats{
		Response: TautulliHomeStatsResponse{
			Result: "success",
			Data: []TautulliHomeStatRow{
				{
					StatID:        "top_tv",
					StatType:      "top_tv",
					StatTitle:     "Most Watched TV Shows",
					RowsForMedia:  true,
					TotalDuration: 3600000,
					TotalPlays:    250,
					Rows: []TautulliHomeStatDetail{
						{
							Title:            "Breaking Bad",
							GrandparentTitle: "Breaking Bad",
							TotalPlays:       100,
							TotalDuration:    1200000,
							UsersWatched:     15,
							RatingKey:        "98765",
							MediaType:        "show",
							Year:             2008,
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliHomeStats: %v", err)
	}

	var decoded TautulliHomeStats
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliHomeStats: %v", err)
	}

	if decoded.Response.Result != original.Response.Result {
		t.Errorf("Result mismatch: expected '%s', got '%s'", original.Response.Result, decoded.Response.Result)
	}
	if len(decoded.Response.Data) != len(original.Response.Data) {
		t.Errorf("Data length mismatch: expected %d, got %d", len(original.Response.Data), len(decoded.Response.Data))
	}
}

func TestTautulliPlaysByDate_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"categories": ["2024-01-01", "2024-01-02", "2024-01-03"],
				"series": [
					{
						"name": "Movies",
						"data": [10, 15, 8]
					},
					{
						"name": "TV",
						"data": [25, 30, 22]
					}
				]
			}
		}
	}`

	var plays TautulliPlaysByDate
	err := json.Unmarshal([]byte(jsonData), &plays)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByDate: %v", err)
	}

	if plays.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", plays.Response.Result)
	}

	if len(plays.Response.Data.Categories) != 3 {
		t.Fatalf("Expected 3 categories, got %d", len(plays.Response.Data.Categories))
	}
	if plays.Response.Data.Categories[0] != "2024-01-01" {
		t.Errorf("Expected first category '2024-01-01', got '%s'", plays.Response.Data.Categories[0])
	}

	if len(plays.Response.Data.Series) != 2 {
		t.Fatalf("Expected 2 series, got %d", len(plays.Response.Data.Series))
	}

	moviesSeries := plays.Response.Data.Series[0]
	if moviesSeries.Name != "Movies" {
		t.Errorf("Expected first series name 'Movies', got '%s'", moviesSeries.Name)
	}
	if len(moviesSeries.Data) != 3 {
		t.Fatalf("Expected 3 data points, got %d", len(moviesSeries.Data))
	}

	// Note: Data is []interface{}, so values come as float64 from JSON
	if val, ok := moviesSeries.Data[0].(float64); !ok || val != 10 {
		t.Errorf("Expected first data point 10, got %v", moviesSeries.Data[0])
	}
}

func TestTautulliPlaysByDate_EmptySeries(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"categories": [],
				"series": []
			}
		}
	}`

	var plays TautulliPlaysByDate
	err := json.Unmarshal([]byte(jsonData), &plays)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByDate: %v", err)
	}

	if len(plays.Response.Data.Categories) != 0 {
		t.Errorf("Expected 0 categories, got %d", len(plays.Response.Data.Categories))
	}
	if len(plays.Response.Data.Series) != 0 {
		t.Errorf("Expected 0 series, got %d", len(plays.Response.Data.Series))
	}
}

func TestTautulliPlaysByDate_ErrorResponse(t *testing.T) {
	errorMsg := "Database error"
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Database error",
			"data": {
				"categories": [],
				"series": []
			}
		}
	}`

	var plays TautulliPlaysByDate
	err := json.Unmarshal([]byte(jsonData), &plays)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByDate: %v", err)
	}

	if plays.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", plays.Response.Result)
	}
	if plays.Response.Message == nil || *plays.Response.Message != errorMsg {
		t.Errorf("Expected Message '%s', got '%v'", errorMsg, plays.Response.Message)
	}
}

func TestTautulliPlaysByDate_JSONRoundTrip(t *testing.T) {
	original := TautulliPlaysByDate{
		Response: TautulliPlaysByDateResponse{
			Result: "success",
			Data: TautulliPlaysByDateData{
				Categories: []string{"2024-01-01", "2024-01-02"},
				Series: []TautulliPlaysByDateSeries{
					{Name: "Movies", Data: []interface{}{float64(5), float64(10)}},
					{Name: "TV", Data: []interface{}{float64(15), float64(20)}},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliPlaysByDate: %v", err)
	}

	var decoded TautulliPlaysByDate
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByDate: %v", err)
	}

	if len(decoded.Response.Data.Categories) != len(original.Response.Data.Categories) {
		t.Errorf("Categories length mismatch")
	}
	if len(decoded.Response.Data.Series) != len(original.Response.Data.Series) {
		t.Errorf("Series length mismatch")
	}
}

func TestTautulliPlaysByDayOfWeek_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"categories": ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"],
				"series": [
					{
						"name": "Movies",
						"data": [50, 30, 25, 28, 35, 45, 60]
					}
				]
			}
		}
	}`

	var plays TautulliPlaysByDayOfWeek
	err := json.Unmarshal([]byte(jsonData), &plays)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByDayOfWeek: %v", err)
	}

	if plays.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", plays.Response.Result)
	}

	if len(plays.Response.Data.Categories) != 7 {
		t.Fatalf("Expected 7 categories (days), got %d", len(plays.Response.Data.Categories))
	}
	if plays.Response.Data.Categories[0] != "Sunday" {
		t.Errorf("Expected first category 'Sunday', got '%s'", plays.Response.Data.Categories[0])
	}
	if plays.Response.Data.Categories[6] != "Saturday" {
		t.Errorf("Expected last category 'Saturday', got '%s'", plays.Response.Data.Categories[6])
	}

	if len(plays.Response.Data.Series) != 1 {
		t.Fatalf("Expected 1 series, got %d", len(plays.Response.Data.Series))
	}

	series := plays.Response.Data.Series[0]
	if len(series.Data) != 7 {
		t.Fatalf("Expected 7 data points, got %d", len(series.Data))
	}
}

func TestTautulliPlaysByDayOfWeek_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "No data available",
			"data": {
				"categories": [],
				"series": []
			}
		}
	}`

	var plays TautulliPlaysByDayOfWeek
	err := json.Unmarshal([]byte(jsonData), &plays)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByDayOfWeek: %v", err)
	}

	if plays.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", plays.Response.Result)
	}
}

func TestTautulliPlaysByDayOfWeek_JSONRoundTrip(t *testing.T) {
	original := TautulliPlaysByDayOfWeek{
		Response: TautulliPlaysByDayOfWeekResponse{
			Result: "success",
			Data: TautulliPlaysByDayOfWeekData{
				Categories: []string{"Sunday", "Monday", "Tuesday"},
				Series: []TautulliPlaysByDateSeries{
					{Name: "TV", Data: []interface{}{float64(100), float64(80), float64(90)}},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliPlaysByDayOfWeek: %v", err)
	}

	var decoded TautulliPlaysByDayOfWeek
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByDayOfWeek: %v", err)
	}

	if decoded.Response.Result != original.Response.Result {
		t.Errorf("Result mismatch")
	}
}

func TestTautulliPlaysByHourOfDay_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"categories": ["00", "01", "02", "03", "04", "05", "06", "07", "08", "09", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23"],
				"series": [
					{
						"name": "Movies",
						"data": [2, 1, 0, 0, 0, 1, 2, 5, 8, 10, 12, 15, 18, 20, 22, 25, 28, 30, 35, 40, 45, 35, 20, 10]
					}
				]
			}
		}
	}`

	var plays TautulliPlaysByHourOfDay
	err := json.Unmarshal([]byte(jsonData), &plays)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByHourOfDay: %v", err)
	}

	if plays.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", plays.Response.Result)
	}

	if len(plays.Response.Data.Categories) != 24 {
		t.Fatalf("Expected 24 categories (hours), got %d", len(plays.Response.Data.Categories))
	}
	if plays.Response.Data.Categories[0] != "00" {
		t.Errorf("Expected first category '00', got '%s'", plays.Response.Data.Categories[0])
	}
	if plays.Response.Data.Categories[23] != "23" {
		t.Errorf("Expected last category '23', got '%s'", plays.Response.Data.Categories[23])
	}

	if len(plays.Response.Data.Series) != 1 {
		t.Fatalf("Expected 1 series, got %d", len(plays.Response.Data.Series))
	}

	series := plays.Response.Data.Series[0]
	if len(series.Data) != 24 {
		t.Fatalf("Expected 24 data points, got %d", len(series.Data))
	}
}

func TestTautulliPlaysByHourOfDay_MultipleSeries(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"categories": ["00", "01", "02"],
				"series": [
					{"name": "Movies", "data": [5, 3, 2]},
					{"name": "TV", "data": [10, 8, 6]},
					{"name": "Music", "data": [2, 1, 0]},
					{"name": "Live TV", "data": [0, 0, 1]}
				]
			}
		}
	}`

	var plays TautulliPlaysByHourOfDay
	err := json.Unmarshal([]byte(jsonData), &plays)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByHourOfDay: %v", err)
	}

	if len(plays.Response.Data.Series) != 4 {
		t.Fatalf("Expected 4 series, got %d", len(plays.Response.Data.Series))
	}

	expectedNames := []string{"Movies", "TV", "Music", "Live TV"}
	for i, series := range plays.Response.Data.Series {
		if series.Name != expectedNames[i] {
			t.Errorf("Expected series name '%s', got '%s'", expectedNames[i], series.Name)
		}
	}
}

func TestTautulliPlaysByHourOfDay_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Time period too short",
			"data": {
				"categories": [],
				"series": []
			}
		}
	}`

	var plays TautulliPlaysByHourOfDay
	err := json.Unmarshal([]byte(jsonData), &plays)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByHourOfDay: %v", err)
	}

	if plays.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", plays.Response.Result)
	}
	if plays.Response.Message == nil || *plays.Response.Message != "Time period too short" {
		t.Error("Expected Message 'Time period too short'")
	}
}

func TestTautulliPlaysByHourOfDay_JSONRoundTrip(t *testing.T) {
	original := TautulliPlaysByHourOfDay{
		Response: TautulliPlaysByHourOfDayResponse{
			Result: "success",
			Data: TautulliPlaysByHourOfDayData{
				Categories: []string{"00", "01", "02", "03"},
				Series: []TautulliPlaysByDateSeries{
					{Name: "Movies", Data: []interface{}{float64(1), float64(2), float64(3), float64(4)}},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliPlaysByHourOfDay: %v", err)
	}

	var decoded TautulliPlaysByHourOfDay
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaysByHourOfDay: %v", err)
	}

	if decoded.Response.Result != original.Response.Result {
		t.Errorf("Result mismatch")
	}
	if len(decoded.Response.Data.Categories) != len(original.Response.Data.Categories) {
		t.Errorf("Categories length mismatch")
	}
}

func TestTautulliHomeStatDetail_AllFields(t *testing.T) {
	jsonData := `{
		"title": "The Matrix",
		"user": "neo",
		"friendly_name": "Neo Anderson",
		"total_plays": 42,
		"total_duration": 86400,
		"users_watched": 10,
		"rating_key": "99999",
		"last_play": 1700000000,
		"grandparent_title": "The Matrix Trilogy",
		"media_type": "movie",
		"platform_name": "Web",
		"platform_type": "browser",
		"player_name": "Chrome",
		"library_name": "Movies",
		"section_id": 5,
		"thumb": "/library/metadata/99999/thumb",
		"year": 1999
	}`

	var detail TautulliHomeStatDetail
	err := json.Unmarshal([]byte(jsonData), &detail)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliHomeStatDetail: %v", err)
	}

	if detail.Title != "The Matrix" {
		t.Errorf("Expected Title 'The Matrix', got '%s'", detail.Title)
	}
	if detail.User != "neo" {
		t.Errorf("Expected User 'neo', got '%s'", detail.User)
	}
	if detail.FriendlyName != "Neo Anderson" {
		t.Errorf("Expected FriendlyName 'Neo Anderson', got '%s'", detail.FriendlyName)
	}
	if detail.TotalPlays != 42 {
		t.Errorf("Expected TotalPlays 42, got %d", detail.TotalPlays)
	}
	if detail.TotalDuration != 86400 {
		t.Errorf("Expected TotalDuration 86400, got %d", detail.TotalDuration)
	}
	if detail.UsersWatched != 10 {
		t.Errorf("Expected UsersWatched 10, got %d", detail.UsersWatched)
	}
	if detail.RatingKey != "99999" {
		t.Errorf("Expected RatingKey '99999', got '%s'", detail.RatingKey)
	}
	if detail.LastPlay != 1700000000 {
		t.Errorf("Expected LastPlay 1700000000, got %d", detail.LastPlay)
	}
	if detail.GrandparentTitle != "The Matrix Trilogy" {
		t.Errorf("Expected GrandparentTitle 'The Matrix Trilogy', got '%s'", detail.GrandparentTitle)
	}
	if detail.MediaType != "movie" {
		t.Errorf("Expected MediaType 'movie', got '%s'", detail.MediaType)
	}
	if detail.Platform != "Web" {
		t.Errorf("Expected Platform 'Web', got '%s'", detail.Platform)
	}
	if detail.PlatformType != "browser" {
		t.Errorf("Expected PlatformType 'browser', got '%s'", detail.PlatformType)
	}
	if detail.PlayerName != "Chrome" {
		t.Errorf("Expected PlayerName 'Chrome', got '%s'", detail.PlayerName)
	}
	if detail.LibraryName != "Movies" {
		t.Errorf("Expected LibraryName 'Movies', got '%s'", detail.LibraryName)
	}
	if detail.SectionID != 5 {
		t.Errorf("Expected SectionID 5, got %d", detail.SectionID)
	}
	if detail.ThumbURL != "/library/metadata/99999/thumb" {
		t.Errorf("Expected ThumbURL '/library/metadata/99999/thumb', got '%s'", detail.ThumbURL)
	}
	if detail.Year != 1999 {
		t.Errorf("Expected Year 1999, got %d", detail.Year)
	}
}
