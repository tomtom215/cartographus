// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliPlaysBySourceResolution_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with resolutions", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"message": null,
				"data": {
					"categories": ["4k", "1080p", "720p", "SD"],
					"series": [
						{
							"name": "Movies",
							"data": [30, 100, 50, 20]
						},
						{
							"name": "TV",
							"data": [10, 150, 80, 30]
						}
					]
				}
			}
		}`

		var resolution TautulliPlaysBySourceResolution
		if err := json.Unmarshal([]byte(jsonData), &resolution); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if resolution.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", resolution.Response.Result)
		}
		if len(resolution.Response.Data.Categories) != 4 {
			t.Errorf("Expected 4 resolutions, got %d", len(resolution.Response.Data.Categories))
		}
		if resolution.Response.Data.Categories[0] != "4k" {
			t.Errorf("Expected first resolution '4k', got %q", resolution.Response.Data.Categories[0])
		}
		if len(resolution.Response.Data.Series) != 2 {
			t.Errorf("Expected 2 series, got %d", len(resolution.Response.Data.Series))
		}
	})

	t.Run("empty data", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": [],
					"series": []
				}
			}
		}`

		var resolution TautulliPlaysBySourceResolution
		if err := json.Unmarshal([]byte(jsonData), &resolution); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(resolution.Response.Data.Categories) != 0 {
			t.Errorf("Expected empty categories, got %d", len(resolution.Response.Data.Categories))
		}
	})

	t.Run("error response with message", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "error",
				"message": "No resolution data available",
				"data": {}
			}
		}`

		var resolution TautulliPlaysBySourceResolution
		if err := json.Unmarshal([]byte(jsonData), &resolution); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if resolution.Response.Result != "error" {
			t.Errorf("Expected result 'error', got %q", resolution.Response.Result)
		}
		if resolution.Response.Message == nil {
			t.Error("Expected non-nil message")
		} else if *resolution.Response.Message != "No resolution data available" {
			t.Errorf("Expected message 'No resolution data available', got %q", *resolution.Response.Message)
		}
	})
}

func TestTautulliPlaysByStreamResolution_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with stream resolutions", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": ["1080p", "720p", "480p"],
					"series": [
						{
							"name": "Direct Play",
							"data": [80, 40, 10]
						},
						{
							"name": "Transcode",
							"data": [20, 30, 25]
						}
					]
				}
			}
		}`

		var resolution TautulliPlaysByStreamResolution
		if err := json.Unmarshal([]byte(jsonData), &resolution); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(resolution.Response.Data.Categories) != 3 {
			t.Errorf("Expected 3 resolutions, got %d", len(resolution.Response.Data.Categories))
		}
		if resolution.Response.Data.Categories[2] != "480p" {
			t.Errorf("Expected third resolution '480p', got %q", resolution.Response.Data.Categories[2])
		}
	})

	t.Run("single resolution", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"categories": ["1080p"],
					"series": [
						{
							"name": "All",
							"data": [100]
						}
					]
				}
			}
		}`

		var resolution TautulliPlaysByStreamResolution
		if err := json.Unmarshal([]byte(jsonData), &resolution); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(resolution.Response.Data.Categories) != 1 {
			t.Errorf("Expected 1 resolution, got %d", len(resolution.Response.Data.Categories))
		}
	})
}

func TestResolutionModels_RoundTrip(t *testing.T) {
	t.Run("TautulliPlaysBySourceResolution round trip", func(t *testing.T) {
		msg := "test message"
		original := TautulliPlaysBySourceResolution{
			Response: TautulliPlaysBySourceResolutionResponse{
				Result:  "success",
				Message: &msg,
				Data: TautulliPlaysBySourceResolutionData{
					Categories: []string{"4k", "1080p"},
					Series: []TautulliPlaysByDateSeries{
						{Name: "Movies", Data: []interface{}{15, 85}},
					},
				},
			},
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var result TautulliPlaysBySourceResolution
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(result.Response.Data.Categories) != 2 {
			t.Error("Categories not preserved in round-trip")
		}
		if result.Response.Message == nil || *result.Response.Message != "test message" {
			t.Error("Message not preserved in round-trip")
		}
	})

	t.Run("TautulliPlaysByStreamResolution round trip", func(t *testing.T) {
		original := TautulliPlaysByStreamResolution{
			Response: TautulliPlaysByStreamResolutionResponse{
				Result: "success",
				Data: TautulliPlaysByStreamResolutionData{
					Categories: []string{"720p"},
					Series: []TautulliPlaysByDateSeries{
						{Name: "Transcode", Data: []interface{}{50}},
					},
				},
			},
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var result TautulliPlaysByStreamResolution
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(result.Response.Data.Series) != 1 {
			t.Error("Series not preserved in round-trip")
		}
		if result.Response.Data.Series[0].Name != "Transcode" {
			t.Error("Series name not preserved in round-trip")
		}
	})
}

func TestResolutionData_ZeroValues(t *testing.T) {
	var sourceData TautulliPlaysBySourceResolutionData
	var streamData TautulliPlaysByStreamResolutionData

	if sourceData.Categories != nil {
		t.Error("Expected nil categories for zero value source data")
	}
	if sourceData.Series != nil {
		t.Error("Expected nil series for zero value source data")
	}
	if streamData.Categories != nil {
		t.Error("Expected nil categories for zero value stream data")
	}
}
