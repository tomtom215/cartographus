// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"testing"

	"github.com/goccy/go-json"
)

// ============================================================================
// PlexStatisticsBandwidth Tests
// ============================================================================

func TestPlexStatisticsBandwidth_IsLANBandwidth(t *testing.T) {
	tests := []struct {
		name string
		bw   PlexStatisticsBandwidth
		want bool
	}{
		{"LAN traffic", PlexStatisticsBandwidth{LAN: true}, true},
		{"WAN traffic", PlexStatisticsBandwidth{LAN: false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bw.IsLANBandwidth(); got != tt.want {
				t.Errorf("IsLANBandwidth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlexStatisticsBandwidth_GetBandwidthMB(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  float64
	}{
		{"1 GB", 1073741824, 1024.0},
		{"512 MB", 536870912, 512.0},
		{"0 bytes", 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bw := PlexStatisticsBandwidth{Bytes: tt.bytes}
			if got := bw.GetBandwidthMB(); got != tt.want {
				t.Errorf("GetBandwidthMB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlexStatisticsBandwidth_GetBandwidthGB(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  float64
	}{
		{"1 GB", 1073741824, 1.0},
		{"2 GB", 2147483648, 2.0},
		{"512 MB", 536870912, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bw := PlexStatisticsBandwidth{Bytes: tt.bytes}
			if got := bw.GetBandwidthGB(); got != tt.want {
				t.Errorf("GetBandwidthGB() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// PlexLibrarySection Tests
// ============================================================================

func TestPlexLibrarySection_IsRefreshing(t *testing.T) {
	tests := []struct {
		name       string
		refreshing bool
		want       bool
	}{
		{"currently refreshing", true, true},
		{"not refreshing", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := PlexLibrarySection{Refreshing: tt.refreshing}
			if got := section.IsRefreshing(); got != tt.want {
				t.Errorf("IsRefreshing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlexLibrarySection_IsHidden(t *testing.T) {
	tests := []struct {
		name   string
		hidden int
		want   bool
	}{
		{"visible (0)", 0, false},
		{"hidden (1)", 1, true},
		{"hidden (2)", 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := PlexLibrarySection{Hidden: tt.hidden}
			if got := section.IsHidden(); got != tt.want {
				t.Errorf("IsHidden() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlexLibrarySection_TypeChecks(t *testing.T) {
	tests := []struct {
		name    string
		section PlexLibrarySection
		isMovie bool
		isTV    bool
		isMusic bool
		isPhoto bool
	}{
		{"movie section", PlexLibrarySection{Type: "movie"}, true, false, false, false},
		{"show section", PlexLibrarySection{Type: "show"}, false, true, false, false},
		{"artist section", PlexLibrarySection{Type: "artist"}, false, false, true, false},
		{"photo section", PlexLibrarySection{Type: "photo"}, false, false, false, true},
		{"unknown type", PlexLibrarySection{Type: "other"}, false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.section.IsMovie(); got != tt.isMovie {
				t.Errorf("IsMovie() = %v, want %v", got, tt.isMovie)
			}
			if got := tt.section.IsTV(); got != tt.isTV {
				t.Errorf("IsTV() = %v, want %v", got, tt.isTV)
			}
			if got := tt.section.IsMusic(); got != tt.isMusic {
				t.Errorf("IsMusic() = %v, want %v", got, tt.isMusic)
			}
			if got := tt.section.IsPhoto(); got != tt.isPhoto {
				t.Errorf("IsPhoto() = %v, want %v", got, tt.isPhoto)
			}
		})
	}
}

// ============================================================================
// PlexActivity Tests
// ============================================================================

func TestPlexActivity_IsInProgress(t *testing.T) {
	tests := []struct {
		name     string
		progress int
		want     bool
	}{
		{"0% progress", 0, true},
		{"50% progress", 50, true},
		{"99% progress", 99, true},
		{"100% complete", 100, false},
		{"indeterminate (-1)", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activity := PlexActivity{Progress: tt.progress}
			if got := activity.IsInProgress(); got != tt.want {
				t.Errorf("IsInProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlexActivity_IsIndeterminate(t *testing.T) {
	tests := []struct {
		name     string
		progress int
		want     bool
	}{
		{"indeterminate (-1)", -1, true},
		{"0% progress", 0, false},
		{"50% progress", 50, false},
		{"100% complete", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activity := PlexActivity{Progress: tt.progress}
			if got := activity.IsIndeterminate(); got != tt.want {
				t.Errorf("IsIndeterminate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlexActivity_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		progress int
		want     bool
	}{
		{"100% complete", 100, true},
		{"99% progress", 99, false},
		{"0% progress", 0, false},
		{"indeterminate (-1)", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activity := PlexActivity{Progress: tt.progress}
			if got := activity.IsComplete(); got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlexActivity_GetLibrarySectionID(t *testing.T) {
	tests := []struct {
		name    string
		context map[string]interface{}
		want    string
	}{
		{
			name:    "with section ID",
			context: map[string]interface{}{"librarySectionID": "1"},
			want:    "1",
		},
		{
			name:    "without section ID",
			context: map[string]interface{}{"other": "value"},
			want:    "",
		},
		{
			name:    "nil context",
			context: nil,
			want:    "",
		},
		{
			name:    "empty context",
			context: map[string]interface{}{},
			want:    "",
		},
		{
			name:    "wrong type",
			context: map[string]interface{}{"librarySectionID": 123},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activity := PlexActivity{Context: tt.context}
			if got := activity.GetLibrarySectionID(); got != tt.want {
				t.Errorf("GetLibrarySectionID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// JSON Marshaling/Unmarshaling Tests
// ============================================================================

func TestPlexBandwidthResponse_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"MediaContainer": {
			"size": 1,
			"Device": [{"id": 1, "name": "Test Device"}],
			"Account": [{"id": 1, "name": "Test User"}],
			"StatisticsBandwidth": [{"accountID": 1, "deviceID": 1, "bytes": 1024, "lan": true}]
		}
	}`

	var resp PlexBandwidthResponse
	if err := unmarshalJSON([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(resp.MediaContainer.Device) != 1 {
		t.Errorf("Expected 1 device, got %d", len(resp.MediaContainer.Device))
	}
	if resp.MediaContainer.Device[0].Name != "Test Device" {
		t.Errorf("Expected device name 'Test Device', got '%s'", resp.MediaContainer.Device[0].Name)
	}
}

func TestPlexActivitiesResponse_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"MediaContainer": {
			"size": 1,
			"Activity": [{
				"uuid": "test-uuid",
				"type": "library.scan",
				"title": "Scanning",
				"progress": 50,
				"cancellable": true
			}]
		}
	}`

	var resp PlexActivitiesResponse
	if err := unmarshalJSON([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(resp.MediaContainer.Activity) != 1 {
		t.Errorf("Expected 1 activity, got %d", len(resp.MediaContainer.Activity))
	}
	if resp.MediaContainer.Activity[0].UUID != "test-uuid" {
		t.Errorf("Expected UUID 'test-uuid', got '%s'", resp.MediaContainer.Activity[0].UUID)
	}
}

// Helper function for JSON unmarshaling in tests
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
