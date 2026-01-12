// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliGeoIP_JSONUnmarshal(t *testing.T) {
	t.Run("complete response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"message": null,
				"data": {
					"city": "New York",
					"region": "New York",
					"country": "United States",
					"postal_code": "10001",
					"timezone": "America/New_York",
					"latitude": 40.7128,
					"longitude": -74.0060,
					"accuracy_radius": 10
				}
			}
		}`

		var geoip TautulliGeoIP
		if err := json.Unmarshal([]byte(jsonData), &geoip); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if geoip.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", geoip.Response.Result)
		}
		if geoip.Response.Message != nil {
			t.Errorf("Expected nil message, got %v", geoip.Response.Message)
		}
		if geoip.Response.Data.City != "New York" {
			t.Errorf("Expected city 'New York', got %q", geoip.Response.Data.City)
		}
		if geoip.Response.Data.Latitude != 40.7128 {
			t.Errorf("Expected latitude 40.7128, got %f", geoip.Response.Data.Latitude)
		}
		if geoip.Response.Data.Longitude != -74.0060 {
			t.Errorf("Expected longitude -74.0060, got %f", geoip.Response.Data.Longitude)
		}
		if geoip.Response.Data.AccuracyRadius != 10 {
			t.Errorf("Expected accuracy_radius 10, got %d", geoip.Response.Data.AccuracyRadius)
		}
	})

	t.Run("with error message", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "error",
				"message": "IP not found",
				"data": {}
			}
		}`

		var geoip TautulliGeoIP
		if err := json.Unmarshal([]byte(jsonData), &geoip); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if geoip.Response.Result != "error" {
			t.Errorf("Expected result 'error', got %q", geoip.Response.Result)
		}
		if geoip.Response.Message == nil {
			t.Error("Expected message to be non-nil")
		} else if *geoip.Response.Message != "IP not found" {
			t.Errorf("Expected message 'IP not found', got %q", *geoip.Response.Message)
		}
	})

	t.Run("empty data fields", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"city": "",
					"region": "",
					"country": "",
					"postal_code": "",
					"timezone": "",
					"latitude": 0,
					"longitude": 0,
					"accuracy_radius": 0
				}
			}
		}`

		var geoip TautulliGeoIP
		if err := json.Unmarshal([]byte(jsonData), &geoip); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if geoip.Response.Data.City != "" {
			t.Errorf("Expected empty city, got %q", geoip.Response.Data.City)
		}
		if geoip.Response.Data.Latitude != 0 {
			t.Errorf("Expected latitude 0, got %f", geoip.Response.Data.Latitude)
		}
	})
}

func TestTautulliGeoIP_JSONMarshal(t *testing.T) {
	msg := "test message"
	geoip := TautulliGeoIP{
		Response: TautulliGeoIPResponse{
			Result:  "success",
			Message: &msg,
			Data: TautulliGeoIPData{
				City:           "London",
				Region:         "England",
				Country:        "United Kingdom",
				PostalCode:     "SW1A 1AA",
				Timezone:       "Europe/London",
				Latitude:       51.5074,
				Longitude:      -0.1278,
				AccuracyRadius: 5,
			},
		},
	}

	data, err := json.Marshal(geoip)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back to verify round-trip
	var result TautulliGeoIP
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Response.Data.City != "London" {
		t.Errorf("Expected city 'London', got %q", result.Response.Data.City)
	}
	if result.Response.Message == nil || *result.Response.Message != "test message" {
		t.Error("Message not preserved in round-trip")
	}
}

func TestTautulliGeoIPData_ZeroValue(t *testing.T) {
	var data TautulliGeoIPData

	if data.City != "" {
		t.Errorf("Expected empty city, got %q", data.City)
	}
	if data.Latitude != 0 {
		t.Errorf("Expected latitude 0, got %f", data.Latitude)
	}
	if data.AccuracyRadius != 0 {
		t.Errorf("Expected accuracy_radius 0, got %d", data.AccuracyRadius)
	}
}
