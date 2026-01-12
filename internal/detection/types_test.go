// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"testing"
)

func TestIsUnknownLocation(t *testing.T) {
	tests := []struct {
		name     string
		lat      float64
		lon      float64
		expected bool
	}{
		{
			name:     "zero coordinates",
			lat:      0,
			lon:      0,
			expected: true,
		},
		{
			name:     "very small positive coordinates",
			lat:      1e-8,
			lon:      1e-8,
			expected: true, // Within epsilon
		},
		{
			name:     "very small negative coordinates",
			lat:      -1e-8,
			lon:      -1e-8,
			expected: true, // Within epsilon
		},
		{
			name:     "valid NYC coordinates",
			lat:      40.7128,
			lon:      -74.0060,
			expected: false,
		},
		{
			name:     "valid London coordinates",
			lat:      51.5074,
			lon:      -0.1278,
			expected: false,
		},
		{
			name:     "valid negative coordinates (Sydney)",
			lat:      -33.8688,
			lon:      151.2093,
			expected: false,
		},
		{
			name:     "only latitude is zero",
			lat:      0,
			lon:      50.0,
			expected: false,
		},
		{
			name:     "only longitude is zero",
			lat:      50.0,
			lon:      0,
			expected: false,
		},
		{
			name:     "at epsilon boundary positive",
			lat:      CoordinateEpsilon,
			lon:      CoordinateEpsilon,
			expected: false, // Exactly at epsilon is not within epsilon
		},
		{
			name:     "just below epsilon",
			lat:      CoordinateEpsilon * 0.5,
			lon:      CoordinateEpsilon * 0.5,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUnknownLocation(tt.lat, tt.lon)
			if result != tt.expected {
				t.Errorf("IsUnknownLocation(%v, %v) = %v, want %v", tt.lat, tt.lon, result, tt.expected)
			}
		})
	}
}

func TestHasValidCoordinates(t *testing.T) {
	tests := []struct {
		name     string
		lat      float64
		lon      float64
		expected bool
	}{
		{
			name:     "zero coordinates are invalid",
			lat:      0,
			lon:      0,
			expected: false,
		},
		{
			name:     "NYC coordinates are valid",
			lat:      40.7128,
			lon:      -74.0060,
			expected: true,
		},
		{
			name:     "small but non-zero lat",
			lat:      1e-6,
			lon:      0,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasValidCoordinates(tt.lat, tt.lon)
			if result != tt.expected {
				t.Errorf("HasValidCoordinates(%v, %v) = %v, want %v", tt.lat, tt.lon, result, tt.expected)
			}
		})
	}
}

func TestDefaultImpossibleTravelConfig(t *testing.T) {
	config := DefaultImpossibleTravelConfig()

	if config.MaxSpeedKmH != 900 {
		t.Errorf("MaxSpeedKmH = %v, want 900", config.MaxSpeedKmH)
	}
	if config.MinDistanceKm != 100 {
		t.Errorf("MinDistanceKm = %v, want 100", config.MinDistanceKm)
	}
	if config.MinTimeDeltaMinutes != 5 {
		t.Errorf("MinTimeDeltaMinutes = %v, want 5", config.MinTimeDeltaMinutes)
	}
	if config.Severity != SeverityCritical {
		t.Errorf("Severity = %v, want %v", config.Severity, SeverityCritical)
	}
}

func TestDefaultConcurrentStreamsConfig(t *testing.T) {
	config := DefaultConcurrentStreamsConfig()

	if config.DefaultLimit != 3 {
		t.Errorf("DefaultLimit = %v, want 3", config.DefaultLimit)
	}
	if config.UserLimits == nil {
		t.Error("UserLimits should not be nil")
	}
	if config.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want %v", config.Severity, SeverityWarning)
	}
}

func TestDefaultDeviceVelocityConfig(t *testing.T) {
	config := DefaultDeviceVelocityConfig()

	if config.WindowMinutes != 5 {
		t.Errorf("WindowMinutes = %v, want 5", config.WindowMinutes)
	}
	if config.MaxUniqueIPs != 3 {
		t.Errorf("MaxUniqueIPs = %v, want 3", config.MaxUniqueIPs)
	}
	if config.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want %v", config.Severity, SeverityWarning)
	}
}

func TestDefaultGeoRestrictionConfig(t *testing.T) {
	config := DefaultGeoRestrictionConfig()

	if len(config.BlockedCountries) != 0 {
		t.Errorf("BlockedCountries should be empty, got %v", config.BlockedCountries)
	}
	if len(config.AllowedCountries) != 0 {
		t.Errorf("AllowedCountries should be empty, got %v", config.AllowedCountries)
	}
	if config.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want %v", config.Severity, SeverityWarning)
	}
}

func TestDefaultSimultaneousLocationsConfig(t *testing.T) {
	config := DefaultSimultaneousLocationsConfig()

	if config.WindowMinutes != 30 {
		t.Errorf("WindowMinutes = %v, want 30", config.WindowMinutes)
	}
	if config.MinDistanceKm != 50 {
		t.Errorf("MinDistanceKm = %v, want 50", config.MinDistanceKm)
	}
	if config.Severity != SeverityCritical {
		t.Errorf("Severity = %v, want %v", config.Severity, SeverityCritical)
	}
}

func TestRuleTypeConstants(t *testing.T) {
	// Verify rule type constant values match expected strings
	tests := []struct {
		ruleType RuleType
		expected string
	}{
		{RuleTypeImpossibleTravel, "impossible_travel"},
		{RuleTypeConcurrentStreams, "concurrent_streams"},
		{RuleTypeDeviceVelocity, "device_velocity"},
		{RuleTypeGeoRestriction, "geo_restriction"},
		{RuleTypeSimultaneousLocations, "simultaneous_locations"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ruleType), func(t *testing.T) {
			if string(tt.ruleType) != tt.expected {
				t.Errorf("RuleType = %v, want %v", tt.ruleType, tt.expected)
			}
		})
	}
}

func TestSeverityConstants(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityCritical, "critical"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			if string(tt.severity) != tt.expected {
				t.Errorf("Severity = %v, want %v", tt.severity, tt.expected)
			}
		})
	}
}

func TestCoordinateEpsilonValue(t *testing.T) {
	// Verify epsilon is sensible (approximately 1cm at equator)
	// 1e-7 degrees * 111,111 m/degree ≈ 1.1cm
	if CoordinateEpsilon != 1e-7 {
		t.Errorf("CoordinateEpsilon = %v, want 1e-7", CoordinateEpsilon)
	}
}
