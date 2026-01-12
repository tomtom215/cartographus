// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliPlaysByStreamType_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with series data", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"message": null,
				"data": {
					"categories": ["2021-12-01", "2021-12-02", "2021-12-03"],
					"series": [
						{
							"name": "Direct Play",
							"data": [10, 15, 12]
						},
						{
							"name": "Direct Stream",
							"data": [5, 8, 6]
						},
						{
							"name": "Transcode",
							"data": [3, 4, 2]
						}
					]
				}
			}
		}`

		var plays TautulliPlaysByStreamType
		if err := json.Unmarshal([]byte(jsonData), &plays); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if plays.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", plays.Response.Result)
		}
		if len(plays.Response.Data.Categories) != 3 {
			t.Errorf("Expected 3 categories, got %d", len(plays.Response.Data.Categories))
		}
		if len(plays.Response.Data.Series) != 3 {
			t.Errorf("Expected 3 series, got %d", len(plays.Response.Data.Series))
		}

		// Check Direct Play series
		directPlay := plays.Response.Data.Series[0]
		if directPlay.Name != "Direct Play" {
			t.Errorf("Expected name 'Direct Play', got %q", directPlay.Name)
		}
		if len(directPlay.Data) != 3 {
			t.Errorf("Expected 3 data points, got %d", len(directPlay.Data))
		}
	})

	t.Run("empty series data", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": [],
					"series": []
				}
			}
		}`

		var plays TautulliPlaysByStreamType
		if err := json.Unmarshal([]byte(jsonData), &plays); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(plays.Response.Data.Categories) != 0 {
			t.Errorf("Expected empty categories, got %d", len(plays.Response.Data.Categories))
		}
		if len(plays.Response.Data.Series) != 0 {
			t.Errorf("Expected empty series, got %d", len(plays.Response.Data.Series))
		}
	})
}

func TestTautulliConcurrentStreamsByStreamType_JSONUnmarshal(t *testing.T) {
	t.Run("complete response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": ["2021-12-01", "2021-12-02"],
					"series": [
						{
							"name": "Direct Play",
							"data": [5, 8]
						},
						{
							"name": "Total Concurrent Streams",
							"data": [10, 15]
						}
					]
				}
			}
		}`

		var concurrent TautulliConcurrentStreamsByStreamType
		if err := json.Unmarshal([]byte(jsonData), &concurrent); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(concurrent.Response.Data.Series) != 2 {
			t.Errorf("Expected 2 series, got %d", len(concurrent.Response.Data.Series))
		}

		// Find Total Concurrent Streams
		found := false
		for _, s := range concurrent.Response.Data.Series {
			if s.Name == "Total Concurrent Streams" {
				found = true
				if len(s.Data) != 2 {
					t.Errorf("Expected 2 data points, got %d", len(s.Data))
				}
			}
		}
		if !found {
			t.Error("Total Concurrent Streams series not found")
		}
	})
}

func TestTautulliStreamTypeByTop10Users_JSONUnmarshal(t *testing.T) {
	t.Run("complete response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": ["user1", "user2", "user3"],
					"series": [
						{
							"name": "Direct Play",
							"data": [100, 80, 60]
						},
						{
							"name": "Transcode",
							"data": [20, 30, 40]
						}
					]
				}
			}
		}`

		var top10 TautulliStreamTypeByTop10Users
		if err := json.Unmarshal([]byte(jsonData), &top10); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(top10.Response.Data.Categories) != 3 {
			t.Errorf("Expected 3 usernames, got %d", len(top10.Response.Data.Categories))
		}
		if top10.Response.Data.Categories[0] != "user1" {
			t.Errorf("Expected first user 'user1', got %q", top10.Response.Data.Categories[0])
		}
	})
}

func TestTautulliStreamTypeByTop10Platforms_JSONUnmarshal(t *testing.T) {
	t.Run("complete response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": ["Chrome", "iOS", "Android"],
					"series": [
						{
							"name": "Direct Play",
							"data": [50, 40, 30]
						}
					]
				}
			}
		}`

		var top10 TautulliStreamTypeByTop10Platforms
		if err := json.Unmarshal([]byte(jsonData), &top10); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(top10.Response.Data.Categories) != 3 {
			t.Errorf("Expected 3 platforms, got %d", len(top10.Response.Data.Categories))
		}
		if top10.Response.Data.Categories[1] != "iOS" {
			t.Errorf("Expected second platform 'iOS', got %q", top10.Response.Data.Categories[1])
		}
	})
}

func TestTautulliStreamTypeSeries_InterfaceData(t *testing.T) {
	t.Run("data can be integers or floats", func(t *testing.T) {
		// Test with integer data
		jsonDataInt := `{
			"name": "Test",
			"data": [1, 2, 3]
		}`

		var seriesInt TautulliStreamTypeSeries
		if err := json.Unmarshal([]byte(jsonDataInt), &seriesInt); err != nil {
			t.Fatalf("Failed to unmarshal int data: %v", err)
		}

		if len(seriesInt.Data) != 3 {
			t.Errorf("Expected 3 data points, got %d", len(seriesInt.Data))
		}

		// Test with float data
		jsonDataFloat := `{
			"name": "Test",
			"data": [1.5, 2.5, 3.5]
		}`

		var seriesFloat TautulliStreamTypeSeries
		if err := json.Unmarshal([]byte(jsonDataFloat), &seriesFloat); err != nil {
			t.Fatalf("Failed to unmarshal float data: %v", err)
		}

		if len(seriesFloat.Data) != 3 {
			t.Errorf("Expected 3 data points, got %d", len(seriesFloat.Data))
		}

		// Test with null values
		jsonDataNull := `{
			"name": "Test",
			"data": [1, null, 3]
		}`

		var seriesNull TautulliStreamTypeSeries
		if err := json.Unmarshal([]byte(jsonDataNull), &seriesNull); err != nil {
			t.Fatalf("Failed to unmarshal data with null: %v", err)
		}

		if len(seriesNull.Data) != 3 {
			t.Errorf("Expected 3 data points, got %d", len(seriesNull.Data))
		}
	})
}

func TestTautulliPlaysByStreamType_RoundTrip(t *testing.T) {
	original := TautulliPlaysByStreamType{
		Response: TautulliPlaysByStreamTypeResponse{
			Result: "success",
			Data: TautulliPlaysByStreamTypeData{
				Categories: []string{"2021-12-01", "2021-12-02"},
				Series: []TautulliStreamTypeSeries{
					{Name: "Direct Play", Data: []interface{}{10, 20}},
					{Name: "Transcode", Data: []interface{}{5, 10}},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliPlaysByStreamType
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(result.Response.Data.Categories) != 2 {
		t.Error("Categories not preserved in round-trip")
	}
	if len(result.Response.Data.Series) != 2 {
		t.Error("Series not preserved in round-trip")
	}
	if result.Response.Data.Series[0].Name != "Direct Play" {
		t.Error("Series name not preserved in round-trip")
	}
}
