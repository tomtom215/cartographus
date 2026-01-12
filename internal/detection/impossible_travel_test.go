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

// mockEventHistory implements EventHistory for testing
// v2.1: Updated interface to include serverID parameter for multi-server support
type mockEventHistory struct {
	lastEvent             *DetectionEvent
	activeStreams         []DetectionEvent
	recentIPs             []string
	simultaneousLocations []DetectionEvent
	geolocation           *Geolocation
}

func (m *mockEventHistory) GetLastEventForUser(ctx context.Context, userID int, serverID string) (*DetectionEvent, error) {
	return m.lastEvent, nil
}

func (m *mockEventHistory) GetActiveStreamsForUser(ctx context.Context, userID int, serverID string) ([]DetectionEvent, error) {
	return m.activeStreams, nil
}

func (m *mockEventHistory) GetRecentIPsForDevice(ctx context.Context, machineID string, serverID string, window time.Duration) ([]string, error) {
	return m.recentIPs, nil
}

func (m *mockEventHistory) GetSimultaneousLocations(ctx context.Context, userID int, serverID string, window time.Duration) ([]DetectionEvent, error) {
	return m.simultaneousLocations, nil
}

func (m *mockEventHistory) GetGeolocation(ctx context.Context, ipAddress string) (*Geolocation, error) {
	return m.geolocation, nil
}

func TestImpossibleTravelDetector_Check(t *testing.T) {
	tests := []struct {
		name        string
		lastEvent   *DetectionEvent
		newEvent    *DetectionEvent
		expectAlert bool
		alertMsg    string
	}{
		{
			name:      "no previous event",
			lastEvent: nil,
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Latitude:  40.7128,
				Longitude: -74.0060, // NYC
				Timestamp: time.Now(),
			},
			expectAlert: false,
		},
		{
			name: "same location",
			lastEvent: &DetectionEvent{
				UserID:    1,
				Latitude:  40.7128,
				Longitude: -74.0060, // NYC
				Timestamp: time.Now().Add(-30 * time.Minute),
			},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Latitude:  40.7128,
				Longitude: -74.0060, // NYC
				Timestamp: time.Now(),
			},
			expectAlert: false,
		},
		{
			name: "nearby location (within threshold)",
			lastEvent: &DetectionEvent{
				UserID:    1,
				Latitude:  40.7128,
				Longitude: -74.0060, // NYC
				Timestamp: time.Now().Add(-30 * time.Minute),
			},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Latitude:  40.7484,
				Longitude: -73.9857, // Empire State Building (~6km)
				Timestamp: time.Now(),
			},
			expectAlert: false, // Within 100km threshold
		},
		{
			name: "impossible travel - NYC to London in 30 minutes",
			lastEvent: &DetectionEvent{
				UserID:    1,
				Latitude:  40.7128,
				Longitude: -74.0060, // NYC
				City:      "New York",
				Country:   "US",
				Timestamp: time.Now().Add(-30 * time.Minute),
			},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Latitude:  51.5074,
				Longitude: -0.1278, // London
				City:      "London",
				Country:   "UK",
				Timestamp: time.Now(),
			},
			expectAlert: true,
			alertMsg:    "would require", // ~5500km in 30 min = 11000 km/h
		},
		{
			name: "possible travel - NYC to Boston in 4 hours (by car)",
			lastEvent: &DetectionEvent{
				UserID:    1,
				Latitude:  40.7128,
				Longitude: -74.0060, // NYC
				Timestamp: time.Now().Add(-4 * time.Hour),
			},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Latitude:  42.3601,
				Longitude: -71.0589, // Boston
				Timestamp: time.Now(),
			},
			expectAlert: false, // ~350km in 4h = 87.5 km/h
		},
		{
			name: "possible travel - NYC to LA in 6 hours (by flight)",
			lastEvent: &DetectionEvent{
				UserID:    1,
				Latitude:  40.7128,
				Longitude: -74.0060, // NYC
				Timestamp: time.Now().Add(-6 * time.Hour),
			},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Latitude:  34.0522,
				Longitude: -118.2437, // LA
				Timestamp: time.Now(),
			},
			expectAlert: false, // ~4000km in 6h = 666 km/h (within 900 km/h limit)
		},
		{
			name: "events too close in time (< 5 min)",
			lastEvent: &DetectionEvent{
				UserID:    1,
				Latitude:  40.7128,
				Longitude: -74.0060, // NYC
				Timestamp: time.Now().Add(-2 * time.Minute),
			},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Latitude:  51.5074,
				Longitude: -0.1278, // London
				Timestamp: time.Now(),
			},
			expectAlert: false, // Too close in time, likely duplicate/out of order
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockEventHistory{lastEvent: tt.lastEvent}
			detector := NewImpossibleTravelDetector(mock)

			alert, err := detector.Check(context.Background(), tt.newEvent)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectAlert && alert == nil {
				t.Error("expected alert but got nil")
			}
			if !tt.expectAlert && alert != nil {
				t.Errorf("expected no alert but got: %s", alert.Message)
			}
			if alert != nil && tt.alertMsg != "" {
				if !containsSubstring(alert.Message, tt.alertMsg) {
					t.Errorf("alert message %q should contain %q", alert.Message, tt.alertMsg)
				}
			}
		})
	}
}

func TestImpossibleTravelDetector_Disabled(t *testing.T) {
	mock := &mockEventHistory{
		lastEvent: &DetectionEvent{
			UserID:    1,
			Latitude:  40.7128,
			Longitude: -74.0060,
			Timestamp: time.Now().Add(-30 * time.Minute),
		},
	}
	detector := NewImpossibleTravelDetector(mock)
	detector.SetEnabled(false)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Latitude:  51.5074,
		Longitude: -0.1278, // London - should trigger alert if enabled
		Timestamp: time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when detector is disabled")
	}
}

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name      string
		lat1      float64
		lon1      float64
		lat2      float64
		lon2      float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "NYC to London",
			lat1:      40.7128,
			lon1:      -74.0060,
			lat2:      51.5074,
			lon2:      -0.1278,
			expected:  5567,
			tolerance: 50, // Allow 50km tolerance
		},
		{
			name:      "NYC to LA",
			lat1:      40.7128,
			lon1:      -74.0060,
			lat2:      34.0522,
			lon2:      -118.2437,
			expected:  3940,
			tolerance: 50,
		},
		{
			name:      "Same point",
			lat1:      40.7128,
			lon1:      -74.0060,
			lat2:      40.7128,
			lon2:      -74.0060,
			expected:  0,
			tolerance: 0.1,
		},
		{
			name:      "Sydney to Tokyo",
			lat1:      -33.8688,
			lon1:      151.2093,
			lat2:      35.6762,
			lon2:      139.6503,
			expected:  7820,
			tolerance: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := haversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			diff := distance - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("distance = %.2f km, expected %.2f km (+/- %.2f)", distance, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestImpossibleTravelDetector_Configure(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewImpossibleTravelDetector(mock)

	// Valid configuration
	err := detector.Configure([]byte(`{"max_speed_kmh": 1000, "min_distance_km": 50, "min_time_delta_minutes": 10, "severity": "warning"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config := detector.Config()
	if config.MaxSpeedKmH != 1000 {
		t.Errorf("max_speed_kmh = %v, want 1000", config.MaxSpeedKmH)
	}
	if config.MinDistanceKm != 50 {
		t.Errorf("min_distance_km = %v, want 50", config.MinDistanceKm)
	}

	// Invalid configuration - negative speed
	err = detector.Configure([]byte(`{"max_speed_kmh": -100}`))
	if err == nil {
		t.Error("expected error for negative speed")
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
