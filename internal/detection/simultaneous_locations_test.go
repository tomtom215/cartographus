// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"testing"
	"time"
)

func TestNewSimultaneousLocationsDetector(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewSimultaneousLocationsDetector(mock)

	if detector == nil {
		t.Fatal("detector should not be nil")
	}
	if detector.Type() != RuleTypeSimultaneousLocations {
		t.Errorf("Type() = %v, want %v", detector.Type(), RuleTypeSimultaneousLocations)
	}
	if !detector.Enabled() {
		t.Error("detector should be enabled by default")
	}
}

func TestSimultaneousLocationsDetector_Check_Disabled(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewSimultaneousLocationsDetector(mock)
	detector.SetEnabled(false)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when detector is disabled")
	}
}

func TestSimultaneousLocationsDetector_Check_NoGeoData(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewSimultaneousLocationsDetector(mock)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Latitude:  0,
		Longitude: 0, // Unknown location
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when geo data is missing")
	}
}

func TestSimultaneousLocationsDetector_Check_NoConcurrentEvents(t *testing.T) {
	mock := &mockEventHistory{
		simultaneousLocations: []DetectionEvent{}, // No concurrent events
	}
	detector := NewSimultaneousLocationsDetector(mock)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Latitude:  40.7128,
		Longitude: -74.0060,
		Timestamp: time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when no concurrent events")
	}
}

func TestSimultaneousLocationsDetector_Check_SameLocation(t *testing.T) {
	mock := &mockEventHistory{
		simultaneousLocations: []DetectionEvent{
			{
				SessionKey: "session2",
				UserID:     1,
				Latitude:   40.7128,  // Same location
				Longitude:  -74.0060, // Same location
				Timestamp:  time.Now(),
			},
		},
	}
	detector := NewSimultaneousLocationsDetector(mock)

	event := &DetectionEvent{
		SessionKey: "session1",
		UserID:     1,
		Username:   "testuser",
		Latitude:   40.7128,
		Longitude:  -74.0060,
		Timestamp:  time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when locations are the same")
	}
}

func TestSimultaneousLocationsDetector_Check_SameSession(t *testing.T) {
	mock := &mockEventHistory{
		simultaneousLocations: []DetectionEvent{
			{
				SessionKey: "session1", // Same session
				UserID:     1,
				Latitude:   51.5074,
				Longitude:  -0.1278,
				Timestamp:  time.Now(),
			},
		},
	}
	detector := NewSimultaneousLocationsDetector(mock)

	event := &DetectionEvent{
		SessionKey: "session1", // Same session
		UserID:     1,
		Username:   "testuser",
		Latitude:   40.7128,
		Longitude:  -74.0060,
		Timestamp:  time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when same session")
	}
}

func TestSimultaneousLocationsDetector_Check_DistantLocations(t *testing.T) {
	mock := &mockEventHistory{
		simultaneousLocations: []DetectionEvent{
			{
				SessionKey: "session2",
				UserID:     1,
				IPAddress:  "5.6.7.8",
				Latitude:   51.5074, // London
				Longitude:  -0.1278,
				City:       "London",
				Country:    "UK",
				Title:      "Movie 2",
				Timestamp:  time.Now().Add(-5 * time.Minute),
			},
		},
	}
	detector := NewSimultaneousLocationsDetector(mock)

	event := &DetectionEvent{
		SessionKey: "session1",
		UserID:     1,
		Username:   "testuser",
		IPAddress:  "1.2.3.4",
		ServerID:   "server1",
		MachineID:  "machine1",
		Latitude:   40.7128, // NYC
		Longitude:  -74.0060,
		City:       "New York",
		Country:    "US",
		Title:      "Movie 1",
		Timestamp:  time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert == nil {
		t.Fatal("expected alert but got nil")
	}

	if alert.RuleType != RuleTypeSimultaneousLocations {
		t.Errorf("RuleType = %v, want %v", alert.RuleType, RuleTypeSimultaneousLocations)
	}
	if alert.UserID != 1 {
		t.Errorf("UserID = %d, want 1", alert.UserID)
	}
	if alert.Title != "Simultaneous Locations Detected" {
		t.Errorf("Title = %s, want Simultaneous Locations Detected", alert.Title)
	}
}

func TestSimultaneousLocationsDetector_Check_CloseLocations(t *testing.T) {
	mock := &mockEventHistory{
		simultaneousLocations: []DetectionEvent{
			{
				SessionKey: "session2",
				UserID:     1,
				Latitude:   40.7484,  // Empire State Building
				Longitude:  -73.9857, // ~6km from Times Square
				Timestamp:  time.Now(),
			},
		},
	}
	detector := NewSimultaneousLocationsDetector(mock)

	event := &DetectionEvent{
		SessionKey: "session1",
		UserID:     1,
		Username:   "testuser",
		Latitude:   40.7580, // Times Square
		Longitude:  -73.9855,
		Timestamp:  time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert for close locations (< 50km)")
	}
}

func TestSimultaneousLocationsDetector_Check_ConcurrentWithNoGeo(t *testing.T) {
	mock := &mockEventHistory{
		simultaneousLocations: []DetectionEvent{
			{
				SessionKey: "session2",
				UserID:     1,
				Latitude:   0, // Unknown location
				Longitude:  0,
				Timestamp:  time.Now(),
			},
		},
	}
	detector := NewSimultaneousLocationsDetector(mock)

	event := &DetectionEvent{
		SessionKey: "session1",
		UserID:     1,
		Username:   "testuser",
		Latitude:   40.7128,
		Longitude:  -74.0060,
		Timestamp:  time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when concurrent has no geo data")
	}
}

func TestSimultaneousLocationsDetector_Configure(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewSimultaneousLocationsDetector(mock)

	tests := []struct {
		name        string
		config      string
		expectError bool
	}{
		{
			name:        "valid configuration",
			config:      `{"window_minutes": 60, "min_distance_km": 100, "severity": "warning"}`,
			expectError: false,
		},
		{
			name:        "invalid json",
			config:      `{invalid}`,
			expectError: true,
		},
		{
			name:        "zero window",
			config:      `{"window_minutes": 0, "min_distance_km": 50}`,
			expectError: true,
		},
		{
			name:        "negative window",
			config:      `{"window_minutes": -30, "min_distance_km": 50}`,
			expectError: true,
		},
		{
			name:        "negative distance",
			config:      `{"window_minutes": 30, "min_distance_km": -10}`,
			expectError: true,
		},
		{
			name:        "zero distance is valid",
			config:      `{"window_minutes": 30, "min_distance_km": 0}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detector.Configure([]byte(tt.config))
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSimultaneousLocationsDetector_EnableDisable(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewSimultaneousLocationsDetector(mock)

	if !detector.Enabled() {
		t.Error("detector should be enabled by default")
	}

	detector.SetEnabled(false)
	if detector.Enabled() {
		t.Error("detector should be disabled after SetEnabled(false)")
	}

	detector.SetEnabled(true)
	if !detector.Enabled() {
		t.Error("detector should be enabled after SetEnabled(true)")
	}
}

func TestSimultaneousLocationsDetector_Config(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewSimultaneousLocationsDetector(mock)

	config := detector.Config()
	if config.WindowMinutes != 30 {
		t.Errorf("WindowMinutes = %d, want 30", config.WindowMinutes)
	}
	if config.MinDistanceKm != 50 {
		t.Errorf("MinDistanceKm = %v, want 50", config.MinDistanceKm)
	}
	if config.Severity != SeverityCritical {
		t.Errorf("Severity = %v, want %v", config.Severity, SeverityCritical)
	}
}

func TestFormatLocationSummary(t *testing.T) {
	tests := []struct {
		name      string
		locations []LocationInfo
		expected  string
	}{
		{
			name:      "empty locations",
			locations: []LocationInfo{},
			expected:  "",
		},
		{
			name: "single location",
			locations: []LocationInfo{
				{City: "New York", Country: "US"},
			},
			expected: "New York, US",
		},
		{
			name: "multiple locations",
			locations: []LocationInfo{
				{City: "New York", Country: "US"},
				{City: "London", Country: "UK"},
			},
			expected: "New York, US, London, UK",
		},
		{
			name: "city only",
			locations: []LocationInfo{
				{City: "Paris", Country: ""},
			},
			expected: "Paris",
		},
		{
			name: "country only",
			locations: []LocationInfo{
				{City: "", Country: "DE"},
			},
			expected: "DE",
		},
		{
			name: "unknown location",
			locations: []LocationInfo{
				{City: "", Country: ""},
			},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLocationSummary(tt.locations)
			if result != tt.expected {
				t.Errorf("formatLocationSummary() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSimultaneousLocationsDetector_MultipleDistantLocations(t *testing.T) {
	mock := &mockEventHistory{
		simultaneousLocations: []DetectionEvent{
			{
				SessionKey: "session2",
				UserID:     1,
				IPAddress:  "5.6.7.8",
				Latitude:   51.5074, // London
				Longitude:  -0.1278,
				City:       "London",
				Country:    "UK",
				Timestamp:  time.Now().Add(-5 * time.Minute),
			},
			{
				SessionKey: "session3",
				UserID:     1,
				IPAddress:  "9.10.11.12",
				Latitude:   35.6762, // Tokyo
				Longitude:  139.6503,
				City:       "Tokyo",
				Country:    "JP",
				Timestamp:  time.Now().Add(-3 * time.Minute),
			},
		},
	}
	detector := NewSimultaneousLocationsDetector(mock)

	event := &DetectionEvent{
		SessionKey: "session1",
		UserID:     1,
		Username:   "suspicious_user",
		IPAddress:  "1.2.3.4",
		Latitude:   40.7128, // NYC
		Longitude:  -74.0060,
		City:       "New York",
		Country:    "US",
		Timestamp:  time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert == nil {
		t.Fatal("expected alert but got nil")
	}

	// Verify message mentions 3 locations
	if !containsSubstring(alert.Message, "3 locations") {
		t.Errorf("alert message should mention 3 locations: %s", alert.Message)
	}
}
