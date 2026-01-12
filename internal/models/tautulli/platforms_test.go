// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliPlaysByTop10Platforms_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with platforms", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"message": null,
				"data": {
					"categories": ["Chrome", "iOS", "Android", "Roku", "Apple TV"],
					"series": [
						{
							"name": "TV",
							"data": [100, 80, 60, 40, 20]
						},
						{
							"name": "Movies",
							"data": [50, 40, 30, 20, 10]
						}
					]
				}
			}
		}`

		var platforms TautulliPlaysByTop10Platforms
		if err := json.Unmarshal([]byte(jsonData), &platforms); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if platforms.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", platforms.Response.Result)
		}
		if len(platforms.Response.Data.Categories) != 5 {
			t.Errorf("Expected 5 platforms, got %d", len(platforms.Response.Data.Categories))
		}
		if platforms.Response.Data.Categories[0] != "Chrome" {
			t.Errorf("Expected first platform 'Chrome', got %q", platforms.Response.Data.Categories[0])
		}
		if len(platforms.Response.Data.Series) != 2 {
			t.Errorf("Expected 2 series, got %d", len(platforms.Response.Data.Series))
		}
	})

	t.Run("empty platforms", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": [],
					"series": []
				}
			}
		}`

		var platforms TautulliPlaysByTop10Platforms
		if err := json.Unmarshal([]byte(jsonData), &platforms); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(platforms.Response.Data.Categories) != 0 {
			t.Errorf("Expected empty categories, got %d", len(platforms.Response.Data.Categories))
		}
	})
}

func TestTautulliPlaysByTop10Users_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with users", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": ["user1", "user2", "user3"],
					"series": [
						{
							"name": "Movies",
							"data": [200, 150, 100]
						}
					]
				}
			}
		}`

		var users TautulliPlaysByTop10Users
		if err := json.Unmarshal([]byte(jsonData), &users); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(users.Response.Data.Categories) != 3 {
			t.Errorf("Expected 3 users, got %d", len(users.Response.Data.Categories))
		}
		if users.Response.Data.Categories[1] != "user2" {
			t.Errorf("Expected second user 'user2', got %q", users.Response.Data.Categories[1])
		}
	})

	t.Run("error response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "error",
				"message": "No data available",
				"data": {}
			}
		}`

		var users TautulliPlaysByTop10Users
		if err := json.Unmarshal([]byte(jsonData), &users); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if users.Response.Result != "error" {
			t.Errorf("Expected result 'error', got %q", users.Response.Result)
		}
		if users.Response.Message == nil {
			t.Error("Expected non-nil message")
		}
	})
}

func TestTautulliPlaysPerMonth_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with monthly data", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": ["2021-01", "2021-02", "2021-03", "2021-04"],
					"series": [
						{
							"name": "Movies",
							"data": [50, 45, 60, 55]
						},
						{
							"name": "TV",
							"data": [100, 120, 110, 130]
						}
					]
				}
			}
		}`

		var monthly TautulliPlaysPerMonth
		if err := json.Unmarshal([]byte(jsonData), &monthly); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(monthly.Response.Data.Categories) != 4 {
			t.Errorf("Expected 4 months, got %d", len(monthly.Response.Data.Categories))
		}
		if monthly.Response.Data.Categories[0] != "2021-01" {
			t.Errorf("Expected first month '2021-01', got %q", monthly.Response.Data.Categories[0])
		}
	})

	t.Run("single month", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": ["2021-12"],
					"series": [
						{
							"name": "Movies",
							"data": [75]
						}
					]
				}
			}
		}`

		var monthly TautulliPlaysPerMonth
		if err := json.Unmarshal([]byte(jsonData), &monthly); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(monthly.Response.Data.Categories) != 1 {
			t.Errorf("Expected 1 month, got %d", len(monthly.Response.Data.Categories))
		}
	})
}

func TestPlatformModels_RoundTrip(t *testing.T) {
	t.Run("TautulliPlaysByTop10Platforms round trip", func(t *testing.T) {
		original := TautulliPlaysByTop10Platforms{
			Response: TautulliPlaysByTop10PlatformsResponse{
				Result: "success",
				Data: TautulliPlaysByTop10PlatformsData{
					Categories: []string{"Chrome", "iOS"},
					Series: []TautulliPlaysByDateSeries{
						{Name: "Movies", Data: []interface{}{10, 20}},
					},
				},
			},
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var result TautulliPlaysByTop10Platforms
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(result.Response.Data.Categories) != 2 {
			t.Error("Categories not preserved in round-trip")
		}
	})
}
