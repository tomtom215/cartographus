// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliItemWatchTimeStats_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with multiple periods", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"message": null,
				"data": [
					{
						"query_days": "1",
						"total_time": 3600,
						"total_plays": 2
					},
					{
						"query_days": "7",
						"total_time": 25200,
						"total_plays": 10
					},
					{
						"query_days": "30",
						"total_time": 108000,
						"total_plays": 45
					},
					{
						"query_days": "0",
						"total_time": 432000,
						"total_plays": 180
					}
				]
			}
		}`

		var stats TautulliItemWatchTimeStats
		if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if stats.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", stats.Response.Result)
		}
		if len(stats.Response.Data) != 4 {
			t.Fatalf("Expected 4 data items, got %d", len(stats.Response.Data))
		}

		// Check 7-day stats
		found := false
		for _, d := range stats.Response.Data {
			if d.QueryDays == "7" {
				found = true
				if d.TotalTime != 25200 {
					t.Errorf("Expected total_time 25200, got %d", d.TotalTime)
				}
				if d.TotalPlays != 10 {
					t.Errorf("Expected total_plays 10, got %d", d.TotalPlays)
				}
			}
		}
		if !found {
			t.Error("7-day stats not found")
		}

		// Check "all time" (query_days: "0")
		for _, d := range stats.Response.Data {
			if d.QueryDays == "0" {
				if d.TotalTime != 432000 {
					t.Errorf("Expected all-time total_time 432000, got %d", d.TotalTime)
				}
			}
		}
	})

	t.Run("empty data", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": []
			}
		}`

		var stats TautulliItemWatchTimeStats
		if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(stats.Response.Data) != 0 {
			t.Errorf("Expected empty data, got %d items", len(stats.Response.Data))
		}
	})
}

func TestTautulliItemUserStats_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with multiple users", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": [
					{
						"user_id": 1,
						"user": "john",
						"friendly_name": "John Doe",
						"total_plays": 50,
						"last_play": 1640995200,
						"last_played": "2021-12-31",
						"platform": "Chrome",
						"player": "Plex Web",
						"ip_address": "192.168.1.100",
						"percent_complete": 95,
						"watched_status": 1.0,
						"thumb": "/user/john/thumb"
					},
					{
						"user_id": 2,
						"user": "jane",
						"total_plays": 30,
						"last_play": 1640908800,
						"last_played": "2021-12-30"
					}
				]
			}
		}`

		var stats TautulliItemUserStats
		if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(stats.Response.Data) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(stats.Response.Data))
		}

		// Check first user with all fields
		user1 := stats.Response.Data[0]
		if user1.UserID != 1 {
			t.Errorf("Expected user_id 1, got %d", user1.UserID)
		}
		if user1.Username != "john" {
			t.Errorf("Expected user 'john', got %q", user1.Username)
		}
		if user1.FriendlyName != "John Doe" {
			t.Errorf("Expected friendly_name 'John Doe', got %q", user1.FriendlyName)
		}
		if user1.TotalPlays != 50 {
			t.Errorf("Expected total_plays 50, got %d", user1.TotalPlays)
		}
		if user1.Platform != "Chrome" {
			t.Errorf("Expected platform 'Chrome', got %q", user1.Platform)
		}
		if user1.WatchedStatus != 1.0 {
			t.Errorf("Expected watched_status 1.0, got %f", user1.WatchedStatus)
		}

		// Check second user with minimal fields
		user2 := stats.Response.Data[1]
		if user2.FriendlyName != "" {
			t.Errorf("Expected empty friendly_name, got %q", user2.FriendlyName)
		}
		if user2.Platform != "" {
			t.Errorf("Expected empty platform, got %q", user2.Platform)
		}
	})
}

func TestTautulliItemUserStatRow_OmitEmpty(t *testing.T) {
	row := TautulliItemUserStatRow{
		UserID:     1,
		Username:   "test",
		TotalPlays: 10,
		LastPlay:   1640995200,
		LastPlayed: "2021-12-31",
		// All optional fields left empty
	}

	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// These fields should be omitted when empty
	omitEmptyFields := []string{"friendly_name", "platform", "player", "ip_address", "percent_complete", "watched_status", "thumb"}
	for _, field := range omitEmptyFields {
		if _, exists := m[field]; exists {
			t.Errorf("Field %q should be omitted when empty, but was present", field)
		}
	}
}

func TestTautulliItemWatchTimeDetail_ZeroValues(t *testing.T) {
	var detail TautulliItemWatchTimeDetail

	if detail.QueryDays != "" {
		t.Errorf("Expected empty query_days, got %q", detail.QueryDays)
	}
	if detail.TotalTime != 0 {
		t.Errorf("Expected zero total_time, got %d", detail.TotalTime)
	}
	if detail.TotalPlays != 0 {
		t.Errorf("Expected zero total_plays, got %d", detail.TotalPlays)
	}
}

func TestTautulliItemWatchTimeStats_RoundTrip(t *testing.T) {
	original := TautulliItemWatchTimeStats{
		Response: TautulliItemWatchTimeStatsResponse{
			Result: "success",
			Data: []TautulliItemWatchTimeDetail{
				{QueryDays: "7", TotalTime: 3600, TotalPlays: 5},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliItemWatchTimeStats
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(result.Response.Data) != 1 {
		t.Fatal("Data not preserved in round-trip")
	}
	if result.Response.Data[0].TotalTime != 3600 {
		t.Error("TotalTime not preserved in round-trip")
	}
}
