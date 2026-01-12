// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// IPv4 Private ranges (RFC 1918)
		{"10.0.0.0/8 start", "10.0.0.1", true},
		{"10.0.0.0/8 middle", "10.100.50.25", true},
		{"10.0.0.0/8 end", "10.255.255.255", true},
		{"172.16.0.0/12 start", "172.16.0.1", true},
		{"172.16.0.0/12 middle", "172.20.0.1", true},
		{"172.16.0.0/12 end", "172.31.255.255", true},
		{"192.168.0.0/16 start", "192.168.0.1", true},
		{"192.168.0.0/16 middle", "192.168.100.50", true},
		{"192.168.0.0/16 end", "192.168.255.255", true},

		// Loopback
		{"loopback IPv4", "127.0.0.1", true},
		{"loopback IPv4 other", "127.100.50.25", true},

		// Link-local
		{"link-local start", "169.254.0.1", true},
		{"link-local middle", "169.254.100.50", true},

		// IPv6 private/local
		{"IPv6 loopback", "::1", true},
		{"IPv6 link-local", "fe80::1", true},
		{"IPv6 unique local fc00", "fc00::1", true},
		{"IPv6 unique local fd00", "fd00::1234:5678", true},

		// Public IPs (should return false)
		{"public IP 1", "8.8.8.8", false},
		{"public IP 2", "1.1.1.1", false},
		{"public IP 3", "142.250.80.46", false},
		{"public IP 4", "93.184.216.34", false},
		{"public IPv6", "2001:4860:4860::8888", false},

		// Edge cases
		{"not in 172.16/12", "172.32.0.1", false},
		{"just below 172.16", "172.15.255.255", false},
		{"invalid IP", "not-an-ip", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPrivateIP(tt.ip)
			if result != tt.expected {
				t.Errorf("IsPrivateIP(%q) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsValidPublicIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"public IPv4", "8.8.8.8", true},
		{"public IPv6", "2001:4860:4860::8888", true},
		{"private IP", "192.168.1.1", false},
		{"loopback", "127.0.0.1", false},
		{"unspecified IPv4", "0.0.0.0", false},
		{"unspecified IPv6", "::", false},
		{"invalid IP", "not-an-ip", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidPublicIP(tt.ip)
			if result != tt.expected {
				t.Errorf("IsValidPublicIP(%q) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestNormalizeIPAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// IPv4 without port
		{"IPv4 simple", "192.168.1.1", "192.168.1.1"},
		{"IPv4 public", "8.8.8.8", "8.8.8.8"},

		// IPv4 with port
		{"IPv4 with port", "192.168.1.1:8096", "192.168.1.1"},
		{"IPv4 with port 32400", "10.0.0.1:32400", "10.0.0.1"},

		// IPv6 without port
		{"IPv6 loopback", "::1", "::1"},
		{"IPv6 full", "2001:4860:4860::8888", "2001:4860:4860::8888"},

		// IPv6 with port (bracketed)
		{"IPv6 bracketed with port", "[::1]:8096", "::1"},
		{"IPv6 full bracketed with port", "[2001:4860:4860::8888]:32400", "2001:4860:4860::8888"},

		// IPv6 bracketed without port
		{"IPv6 bracketed no port", "[::1]", "::1"},

		// Edge cases
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeIPAddress(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeIPAddress(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCreateLocalGeolocation(t *testing.T) {
	ip := "192.168.1.100"
	geo := CreateLocalGeolocation(ip)

	if geo == nil {
		t.Fatal("CreateLocalGeolocation returned nil")
	}
	if geo.IPAddress != ip {
		t.Errorf("IPAddress = %q, expected %q", geo.IPAddress, ip)
	}
	if geo.Latitude != 0 || geo.Longitude != 0 {
		t.Errorf("Coordinates = (%v, %v), expected (0, 0)", geo.Latitude, geo.Longitude)
	}
	if geo.Country != "Local" {
		t.Errorf("Country = %q, expected 'Local'", geo.Country)
	}
	if geo.City == nil || *geo.City != "Local Network" {
		t.Errorf("City = %v, expected 'Local Network'", geo.City)
	}
	if geo.LastUpdated.IsZero() {
		t.Error("LastUpdated should not be zero")
	}
}

func TestIPAPIProvider_Lookup(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse the IP from the URL path
		// Expected format: /json/8.8.8.8?fields=...
		response := ipAPIResponse{
			Status:      "success",
			Country:     "United States",
			CountryCode: "US",
			Region:      "CA",
			RegionName:  "California",
			City:        "Mountain View",
			Zip:         "94043",
			Lat:         37.4223,
			Lon:         -122.0848,
			Timezone:    "America/Los_Angeles",
			ISP:         "Google LLC",
			Query:       "8.8.8.8",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with mock server URL
	provider := &IPAPIProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		rateLimiter: newRateLimiter(45, time.Minute/45),
		baseURL:     server.URL + "/json",
	}

	ctx := context.Background()
	geo, err := provider.Lookup(ctx, "8.8.8.8")

	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}

	if geo == nil {
		t.Fatal("Lookup returned nil geolocation")
	}

	if geo.IPAddress != "8.8.8.8" {
		t.Errorf("IPAddress = %q, expected '8.8.8.8'", geo.IPAddress)
	}
	if geo.Country != "United States" {
		t.Errorf("Country = %q, expected 'United States'", geo.Country)
	}
	if geo.Latitude != 37.4223 {
		t.Errorf("Latitude = %v, expected 37.4223", geo.Latitude)
	}
	if geo.Longitude != -122.0848 {
		t.Errorf("Longitude = %v, expected -122.0848", geo.Longitude)
	}
	if geo.City == nil || *geo.City != "Mountain View" {
		t.Errorf("City = %v, expected 'Mountain View'", geo.City)
	}
	if geo.Region == nil || *geo.Region != "California" {
		t.Errorf("Region = %v, expected 'California'", geo.Region)
	}
	if geo.Timezone == nil || *geo.Timezone != "America/Los_Angeles" {
		t.Errorf("Timezone = %v, expected 'America/Los_Angeles'", geo.Timezone)
	}
}

func TestIPAPIProvider_Lookup_InvalidIP(t *testing.T) {
	provider := NewIPAPIProvider()

	ctx := context.Background()
	_, err := provider.Lookup(ctx, "not-an-ip")

	if err == nil {
		t.Error("Expected error for invalid IP, got nil")
	}
}

func TestIPAPIProvider_Lookup_FailedResponse(t *testing.T) {
	// Create a mock server that returns a failed response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ipAPIResponse{
			Status:  "fail",
			Message: "reserved range",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := &IPAPIProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		rateLimiter: newRateLimiter(45, time.Minute/45),
		baseURL:     server.URL + "/json",
	}

	ctx := context.Background()
	_, err := provider.Lookup(ctx, "192.168.1.1") // This will pass validation but server returns fail

	if err == nil {
		t.Error("Expected error for failed response, got nil")
	}
}

func TestIPAPIProvider_Name(t *testing.T) {
	provider := NewIPAPIProvider()
	if provider.Name() != "ip-api.com" {
		t.Errorf("Name() = %q, expected 'ip-api.com'", provider.Name())
	}
}

func TestIPAPIProvider_IsAvailable(t *testing.T) {
	provider := NewIPAPIProvider()
	if !provider.IsAvailable() {
		t.Error("IsAvailable() = false, expected true")
	}
}

func TestRateLimiter(t *testing.T) {
	// Create a rate limiter with 3 tokens
	rl := newRateLimiter(3, time.Second)

	// Should allow 3 requests
	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if rl.Allow() {
		t.Error("4th request should be denied")
	}
}

// ========================================
// MaxMind Provider Tests
// ========================================

func TestMaxMindProvider_Name(t *testing.T) {
	provider := NewMaxMindProvider("123456", "test-key")
	if provider.Name() != "maxmind-geolite2" {
		t.Errorf("Name() = %q, expected 'maxmind-geolite2'", provider.Name())
	}
}

func TestMaxMindProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name       string
		accountID  string
		licenseKey string
		want       bool
	}{
		{
			name:       "both credentials set",
			accountID:  "123456",
			licenseKey: "test-license-key",
			want:       true,
		},
		{
			name:       "missing account ID",
			accountID:  "",
			licenseKey: "test-license-key",
			want:       false,
		},
		{
			name:       "missing license key",
			accountID:  "123456",
			licenseKey: "",
			want:       false,
		},
		{
			name:       "both credentials missing",
			accountID:  "",
			licenseKey: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMaxMindProvider(tt.accountID, tt.licenseKey)
			if got := provider.IsAvailable(); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaxMindProvider_Lookup(t *testing.T) {
	// Create a mock server that simulates MaxMind GeoLite2 API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Basic Auth
		username, password, ok := r.BasicAuth()
		if !ok || username != "123456" || password != "test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"code":  "AUTHORIZATION_INVALID",
				"error": "Invalid credentials",
			})
			return
		}

		// MaxMind GeoLite2 response format
		response := maxMindResponse{
			City: struct {
				Names map[string]string `json:"names"`
			}{
				Names: map[string]string{"en": "Mountain View"},
			},
			Country: struct {
				ISOCode string            `json:"iso_code"`
				Names   map[string]string `json:"names"`
			}{
				ISOCode: "US",
				Names:   map[string]string{"en": "United States"},
			},
			Location: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
				TimeZone  string  `json:"time_zone"`
			}{
				Latitude:  37.386,
				Longitude: -122.0838,
				TimeZone:  "America/Los_Angeles",
			},
			Postal: struct {
				Code string `json:"code"`
			}{
				Code: "94035",
			},
			Subdivisions: []struct {
				ISOCode string            `json:"iso_code"`
				Names   map[string]string `json:"names"`
			}{
				{
					ISOCode: "CA",
					Names:   map[string]string{"en": "California"},
				},
			},
			Traits: struct {
				IPAddress string `json:"ip_address"`
			}{
				IPAddress: "8.8.8.8",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := &MaxMindProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		accountID:  "123456",
		licenseKey: "test-key",
		baseURL:    server.URL,
	}

	ctx := context.Background()
	geo, err := provider.Lookup(ctx, "8.8.8.8")

	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}

	if geo.IPAddress != "8.8.8.8" {
		t.Errorf("IPAddress = %q, expected '8.8.8.8'", geo.IPAddress)
	}
	if geo.Country != "United States" {
		t.Errorf("Country = %q, expected 'United States'", geo.Country)
	}
	if geo.Latitude != 37.386 {
		t.Errorf("Latitude = %v, expected 37.386", geo.Latitude)
	}
	if geo.Longitude != -122.0838 {
		t.Errorf("Longitude = %v, expected -122.0838", geo.Longitude)
	}
	if geo.City == nil || *geo.City != "Mountain View" {
		t.Errorf("City = %v, expected 'Mountain View'", geo.City)
	}
	if geo.Region == nil || *geo.Region != "California" {
		t.Errorf("Region = %v, expected 'California'", geo.Region)
	}
	if geo.PostalCode == nil || *geo.PostalCode != "94035" {
		t.Errorf("PostalCode = %v, expected '94035'", geo.PostalCode)
	}
	if geo.Timezone == nil || *geo.Timezone != "America/Los_Angeles" {
		t.Errorf("Timezone = %v, expected 'America/Los_Angeles'", geo.Timezone)
	}
}

func TestMaxMindProvider_Lookup_InvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"code":  "AUTHORIZATION_INVALID",
			"error": "Invalid credentials",
		})
	}))
	defer server.Close()

	provider := &MaxMindProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		accountID:  "wrong-id",
		licenseKey: "wrong-key",
		baseURL:    server.URL,
	}

	ctx := context.Background()
	_, err := provider.Lookup(ctx, "8.8.8.8")

	if err == nil {
		t.Error("Expected error for invalid credentials, got nil")
	}
	if !strings.Contains(err.Error(), "AUTHORIZATION_INVALID") {
		t.Errorf("Error should mention AUTHORIZATION_INVALID, got: %v", err)
	}
}

func TestMaxMindProvider_Lookup_NotConfigured(t *testing.T) {
	provider := NewMaxMindProvider("", "") // No credentials

	ctx := context.Background()
	_, err := provider.Lookup(ctx, "8.8.8.8")

	if err == nil {
		t.Error("Expected error when not configured, got nil")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Error should mention not configured, got: %v", err)
	}
}

func TestMaxMindProvider_Lookup_InvalidIP(t *testing.T) {
	provider := NewMaxMindProvider("123456", "test-key")

	ctx := context.Background()
	_, err := provider.Lookup(ctx, "not-an-ip")

	if err == nil {
		t.Error("Expected error for invalid IP, got nil")
	}
}
