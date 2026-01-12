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

func TestTautulliClient_GetGeoIPLookup(t *testing.T) {
	tests := []struct {
		name        string
		ipAddress   string
		mockData    tautulli.TautulliGeoIP
		expectError bool
	}{
		{
			name:      "valid IP with full data",
			ipAddress: "8.8.8.8",
			mockData: tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Country:        "United States",
						Region:         "California",
						City:           "Mountain View",
						PostalCode:     "94043",
						Timezone:       "America/Los_Angeles",
						Latitude:       37.386,
						Longitude:      -122.084,
						AccuracyRadius: 1000,
					},
				},
			},
			expectError: false,
		},
		{
			name:      "valid IP with minimal data",
			ipAddress: "1.1.1.1",
			mockData: tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Country:   "Australia",
						Latitude:  -33.494,
						Longitude: 143.2104,
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cmd") != "get_geoip_lookup" {
					t.Errorf("Expected cmd=get_geoip_lookup, got %s", r.URL.Query().Get("cmd"))
				}
				if r.URL.Query().Get("ip_address") != tt.ipAddress {
					t.Errorf("Expected ip_address=%s, got %s", tt.ipAddress, r.URL.Query().Get("ip_address"))
				}

				json.NewEncoder(w).Encode(tt.mockData)
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{
				URL:    server.URL,
				APIKey: "test-key",
			}
			client := NewTautulliClient(cfg)

			geoIP, err := client.GetGeoIPLookup(context.Background(), tt.ipAddress)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if geoIP == nil {
				t.Fatal("Expected non-nil geoIP")
			}

			if geoIP.Response.Data.Country != tt.mockData.Response.Data.Country {
				t.Errorf("Expected country %s, got %s", tt.mockData.Response.Data.Country, geoIP.Response.Data.Country)
			}

			if geoIP.Response.Data.Latitude != tt.mockData.Response.Data.Latitude {
				t.Errorf("Expected latitude %f, got %f", tt.mockData.Response.Data.Latitude, geoIP.Response.Data.Latitude)
			}

			if geoIP.Response.Data.Longitude != tt.mockData.Response.Data.Longitude {
				t.Errorf("Expected longitude %f, got %f", tt.mockData.Response.Data.Longitude, geoIP.Response.Data.Longitude)
			}
		})
	}
}
