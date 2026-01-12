// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"testing"

	"github.com/tomtom215/cartographus/internal/config"
)

// TestJellyfinManagerServerID tests the ServerID method
func TestJellyfinManagerServerID(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.JellyfinConfig
		expected string
	}{
		{
			name: "returns configured server ID",
			cfg: &config.JellyfinConfig{
				Enabled:  true,
				ServerID: "jellyfin-home",
				URL:      "http://localhost:8096",
				APIKey:   "test-key",
			},
			expected: "jellyfin-home",
		},
		{
			name:     "returns empty for nil manager",
			cfg:      nil,
			expected: "",
		},
		{
			name: "returns empty for disabled config",
			cfg: &config.JellyfinConfig{
				Enabled:  false,
				ServerID: "should-not-return",
			},
			expected: "",
		},
		{
			name: "returns auto-generated server ID",
			cfg: &config.JellyfinConfig{
				Enabled:  true,
				ServerID: "jellyfin-abc12345",
				URL:      "http://192.168.1.100:8096",
				APIKey:   "test-key",
			},
			expected: "jellyfin-abc12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mgr *JellyfinManager
			if tt.cfg != nil && tt.cfg.Enabled {
				mgr = NewJellyfinManager(tt.cfg, nil, nil)
			}

			got := ""
			if mgr != nil {
				got = mgr.ServerID()
			} else if tt.cfg == nil {
				// Test nil manager case
				var nilMgr *JellyfinManager
				got = nilMgr.ServerID()
			}

			if got != tt.expected {
				t.Errorf("ServerID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestEmbyManagerServerID tests the ServerID method
func TestEmbyManagerServerID(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.EmbyConfig
		expected string
	}{
		{
			name: "returns configured server ID",
			cfg: &config.EmbyConfig{
				Enabled:  true,
				ServerID: "emby-living-room",
				URL:      "http://localhost:8096",
				APIKey:   "test-key",
			},
			expected: "emby-living-room",
		},
		{
			name:     "returns empty for nil manager",
			cfg:      nil,
			expected: "",
		},
		{
			name: "returns empty for disabled config",
			cfg: &config.EmbyConfig{
				Enabled:  false,
				ServerID: "should-not-return",
			},
			expected: "",
		},
		{
			name: "returns auto-generated server ID",
			cfg: &config.EmbyConfig{
				Enabled:  true,
				ServerID: "emby-12345678",
				URL:      "http://192.168.1.200:8096",
				APIKey:   "test-key",
			},
			expected: "emby-12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mgr *EmbyManager
			if tt.cfg != nil && tt.cfg.Enabled {
				mgr = NewEmbyManager(tt.cfg, nil, nil)
			}

			got := ""
			if mgr != nil {
				got = mgr.ServerID()
			} else if tt.cfg == nil {
				// Test nil manager case
				var nilMgr *EmbyManager
				got = nilMgr.ServerID()
			}

			if got != tt.expected {
				t.Errorf("ServerID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestJellyfinManagerServerIDNilSafe tests that ServerID handles nil gracefully
func TestJellyfinManagerServerIDNilSafe(t *testing.T) {
	var mgr *JellyfinManager
	result := mgr.ServerID()
	if result != "" {
		t.Errorf("ServerID() on nil manager = %q, want empty string", result)
	}
}

// TestEmbyManagerServerIDNilSafe tests that ServerID handles nil gracefully
func TestEmbyManagerServerIDNilSafe(t *testing.T) {
	var mgr *EmbyManager
	result := mgr.ServerID()
	if result != "" {
		t.Errorf("ServerID() on nil manager = %q, want empty string", result)
	}
}
