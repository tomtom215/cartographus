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

func TestConcurrentStreamsDetector_Check(t *testing.T) {
	tests := []struct {
		name          string
		activeStreams []DetectionEvent
		newEvent      *DetectionEvent
		limit         int
		expectAlert   bool
	}{
		{
			name:          "no active streams",
			activeStreams: []DetectionEvent{},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				EventType: "start",
			},
			limit:       3,
			expectAlert: false,
		},
		{
			name: "below limit",
			activeStreams: []DetectionEvent{
				{SessionKey: "session1", UserID: 1},
			},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				EventType: "start",
			},
			limit:       3,
			expectAlert: false, // 1 active + 1 new = 2, below limit of 3
		},
		{
			name: "at limit",
			activeStreams: []DetectionEvent{
				{SessionKey: "session1", UserID: 1},
				{SessionKey: "session2", UserID: 1},
			},
			newEvent: &DetectionEvent{
				UserID:     1,
				Username:   "testuser",
				EventType:  "start",
				SessionKey: "session3",
			},
			limit:       3,
			expectAlert: false, // 2 active + 1 new = 3, at limit but not exceeding (allowed)
		},
		{
			name: "exceeding limit",
			activeStreams: []DetectionEvent{
				{SessionKey: "session1", UserID: 1},
				{SessionKey: "session2", UserID: 1},
				{SessionKey: "session3", UserID: 1},
			},
			newEvent: &DetectionEvent{
				UserID:     1,
				Username:   "testuser",
				EventType:  "start",
				SessionKey: "session4",
			},
			limit:       3,
			expectAlert: true, // 3 active + 1 new = 4, above limit
		},
		{
			name: "stop event - no check",
			activeStreams: []DetectionEvent{
				{SessionKey: "session1", UserID: 1},
				{SessionKey: "session2", UserID: 1},
				{SessionKey: "session3", UserID: 1},
			},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				EventType: "stop",
			},
			limit:       3,
			expectAlert: false, // Stop events don't trigger check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockEventHistory{activeStreams: tt.activeStreams}
			detector := NewConcurrentStreamsDetector(mock)

			// Set custom limit
			err := detector.Configure([]byte(`{"default_limit": ` + string(rune('0'+tt.limit)) + `}`))
			if err != nil {
				t.Fatalf("failed to configure: %v", err)
			}

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
		})
	}
}

func TestConcurrentStreamsDetector_UserLimits(t *testing.T) {
	mock := &mockEventHistory{
		activeStreams: []DetectionEvent{
			{SessionKey: "session1", UserID: 1},
			{SessionKey: "session2", UserID: 1},
			{SessionKey: "session3", UserID: 1},
			{SessionKey: "session4", UserID: 1},
		},
	}
	detector := NewConcurrentStreamsDetector(mock)

	// Set higher limit for user 1
	detector.SetUserLimit(1, 5)

	event := &DetectionEvent{
		UserID:     1,
		Username:   "testuser",
		EventType:  "start",
		SessionKey: "session5",
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert with increased user limit")
	}

	// Verify effective limit
	effectiveLimit := detector.GetUserLimit(1)
	if effectiveLimit != 5 {
		t.Errorf("effective limit = %d, want 5", effectiveLimit)
	}

	// Remove user limit and check again
	detector.RemoveUserLimit(1)
	effectiveLimit = detector.GetUserLimit(1)
	if effectiveLimit != 3 { // Default
		t.Errorf("effective limit after removal = %d, want 3", effectiveLimit)
	}
}

func TestConcurrentStreamsDetector_Disabled(t *testing.T) {
	mock := &mockEventHistory{
		activeStreams: []DetectionEvent{
			{SessionKey: "session1", UserID: 1},
			{SessionKey: "session2", UserID: 1},
			{SessionKey: "session3", UserID: 1},
		},
	}
	detector := NewConcurrentStreamsDetector(mock)
	detector.SetEnabled(false)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		EventType: "start",
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when detector is disabled")
	}
}

func TestDeviceVelocityDetector_Check(t *testing.T) {
	tests := []struct {
		name        string
		recentIPs   []string
		newEvent    *DetectionEvent
		maxIPs      int
		expectAlert bool
	}{
		{
			name:      "no recent IPs",
			recentIPs: []string{},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				MachineID: "device123",
				IPAddress: "1.2.3.4",
			},
			maxIPs:      3,
			expectAlert: false,
		},
		{
			name:      "below threshold",
			recentIPs: []string{"1.2.3.4", "1.2.3.5"},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				MachineID: "device123",
				IPAddress: "1.2.3.6",
			},
			maxIPs:      3,
			expectAlert: false, // 3 IPs total, at threshold
		},
		{
			name:      "exceeding threshold",
			recentIPs: []string{"1.2.3.4", "1.2.3.5", "1.2.3.6"},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				MachineID: "device123",
				IPAddress: "1.2.3.7",
			},
			maxIPs:      3,
			expectAlert: true, // 4 IPs total, above threshold
		},
		{
			name:      "no machine ID",
			recentIPs: []string{"1.2.3.4", "1.2.3.5", "1.2.3.6"},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				MachineID: "", // No machine ID
				IPAddress: "1.2.3.7",
			},
			maxIPs:      3,
			expectAlert: false, // Can't track without machine ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockEventHistory{recentIPs: tt.recentIPs}
			detector := NewDeviceVelocityDetector(mock)

			// Configure threshold
			err := detector.Configure([]byte(`{"window_minutes": 5, "max_unique_ips": ` + string(rune('0'+tt.maxIPs)) + `}`))
			if err != nil {
				t.Fatalf("failed to configure: %v", err)
			}

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
		})
	}
}

func TestSimultaneousLocationsDetector_Check(t *testing.T) {
	tests := []struct {
		name                  string
		simultaneousLocations []DetectionEvent
		newEvent              *DetectionEvent
		minDistanceKm         float64
		expectAlert           bool
	}{
		{
			name:                  "no simultaneous sessions",
			simultaneousLocations: []DetectionEvent{},
			newEvent: &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			minDistanceKm: 50,
			expectAlert:   false,
		},
		{
			name: "same location sessions",
			simultaneousLocations: []DetectionEvent{
				{
					SessionKey: "session1",
					UserID:     1,
					Latitude:   40.7128,
					Longitude:  -74.0060,
				},
			},
			newEvent: &DetectionEvent{
				UserID:     1,
				Username:   "testuser",
				SessionKey: "session2",
				Latitude:   40.7128,
				Longitude:  -74.0060,
			},
			minDistanceKm: 50,
			expectAlert:   false,
		},
		{
			name: "distant location sessions",
			simultaneousLocations: []DetectionEvent{
				{
					SessionKey: "session1",
					UserID:     1,
					Latitude:   40.7128,
					Longitude:  -74.0060, // NYC
					City:       "New York",
					Country:    "US",
					Timestamp:  time.Now().Add(-5 * time.Minute),
				},
			},
			newEvent: &DetectionEvent{
				UserID:     1,
				Username:   "testuser",
				SessionKey: "session2",
				Latitude:   34.0522,
				Longitude:  -118.2437, // LA
				City:       "Los Angeles",
				Country:    "US",
				Timestamp:  time.Now(),
			},
			minDistanceKm: 50,
			expectAlert:   true, // ~4000km apart
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockEventHistory{simultaneousLocations: tt.simultaneousLocations}
			detector := NewSimultaneousLocationsDetector(mock)

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
		})
	}
}

func TestGeoRestrictionDetector_Check(t *testing.T) {
	tests := []struct {
		name             string
		blockedCountries []string
		allowedCountries []string
		eventCountry     string
		expectAlert      bool
	}{
		{
			name:             "allowed country in blocklist mode",
			blockedCountries: []string{"CN", "RU"},
			eventCountry:     "US",
			expectAlert:      false,
		},
		{
			name:             "blocked country in blocklist mode",
			blockedCountries: []string{"CN", "RU"},
			eventCountry:     "CN",
			expectAlert:      true,
		},
		{
			name:             "allowed country in allowlist mode",
			allowedCountries: []string{"US", "CA", "GB"},
			eventCountry:     "US",
			expectAlert:      false,
		},
		{
			name:             "blocked country in allowlist mode",
			allowedCountries: []string{"US", "CA", "GB"},
			eventCountry:     "CN",
			expectAlert:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockEventHistory{}
			detector := NewGeoRestrictionDetector(mock)
			detector.SetEnabled(true)

			config := GeoRestrictionConfig{
				BlockedCountries: tt.blockedCountries,
				AllowedCountries: tt.allowedCountries,
				Severity:         SeverityWarning,
			}

			if len(tt.blockedCountries) > 0 || len(tt.allowedCountries) > 0 {
				configBytes, _ := jsonMarshal(config)
				detector.Configure(configBytes)
			}

			event := &DetectionEvent{
				UserID:   1,
				Username: "testuser",
				Country:  tt.eventCountry,
			}

			alert, err := detector.Check(context.Background(), event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectAlert && alert == nil {
				t.Error("expected alert but got nil")
			}
			if !tt.expectAlert && alert != nil {
				t.Errorf("expected no alert but got: %s", alert.Message)
			}
		})
	}
}

// jsonMarshal is a simple JSON marshal helper for tests
func jsonMarshal(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case GeoRestrictionConfig:
		var result string
		result = `{"severity":"` + string(val.Severity) + `"`
		if len(val.BlockedCountries) > 0 {
			result += `,"blocked_countries":[`
			for i, c := range val.BlockedCountries {
				if i > 0 {
					result += ","
				}
				result += `"` + c + `"`
			}
			result += `]`
		}
		if len(val.AllowedCountries) > 0 {
			result += `,"allowed_countries":[`
			for i, c := range val.AllowedCountries {
				if i > 0 {
					result += ","
				}
				result += `"` + c + `"`
			}
			result += `]`
		}
		result += `}`
		return []byte(result), nil
	default:
		return nil, nil
	}
}
