// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import "testing"

func TestEmbySession_IsPlaying(t *testing.T) {
	tests := []struct {
		name     string
		session  EmbySession
		expected bool
	}{
		{
			name: "playing content",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{Name: "Movie"},
				PlayState:      &EmbyPlayState{IsPaused: false},
			},
			expected: true,
		},
		{
			name: "paused content",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{Name: "Movie"},
				PlayState:      &EmbyPlayState{IsPaused: true},
			},
			expected: false,
		},
		{
			name: "no playback item",
			session: EmbySession{
				PlayState: &EmbyPlayState{IsPaused: false},
			},
			expected: false,
		},
		{
			name: "no play state",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{Name: "Movie"},
			},
			expected: false,
		},
		{
			name:     "empty session",
			session:  EmbySession{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.IsPlaying()
			if result != tt.expected {
				t.Errorf("IsPlaying() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEmbySession_IsPaused(t *testing.T) {
	tests := []struct {
		name     string
		session  EmbySession
		expected bool
	}{
		{
			name: "paused content",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{Name: "Movie"},
				PlayState:      &EmbyPlayState{IsPaused: true},
			},
			expected: true,
		},
		{
			name: "playing content",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{Name: "Movie"},
				PlayState:      &EmbyPlayState{IsPaused: false},
			},
			expected: false,
		},
		{
			name:     "empty session",
			session:  EmbySession{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.IsPaused()
			if result != tt.expected {
				t.Errorf("IsPaused() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEmbySession_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		session  EmbySession
		expected bool
	}{
		{
			name: "has now playing item",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{Name: "Movie"},
			},
			expected: true,
		},
		{
			name:     "no now playing item",
			session:  EmbySession{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.IsActive()
			if result != tt.expected {
				t.Errorf("IsActive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEmbySession_GetIPAddress(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{"ipv4 with port", "192.168.1.100:52341", "192.168.1.100"},
		{"ipv4 without port", "192.168.1.100", "192.168.1.100"},
		{"ipv6 with port", "[2001:db8::1]:52341", "2001:db8::1"},
		{"ipv6 without port", "[2001:db8::1]", "2001:db8::1"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := EmbySession{RemoteEndPoint: tt.endpoint}
			result := session.GetIPAddress()
			if result != tt.expected {
				t.Errorf("GetIPAddress() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEmbySession_GetPositionSeconds(t *testing.T) {
	tests := []struct {
		name     string
		session  EmbySession
		expected int64
	}{
		{
			name: "valid position",
			session: EmbySession{
				PlayState: &EmbyPlayState{PositionTicks: 300000000}, // 30 seconds in ticks (30 * 10,000,000)
			},
			expected: 30,
		},
		{
			name:     "no play state",
			session:  EmbySession{},
			expected: 0,
		},
		{
			name: "zero position",
			session: EmbySession{
				PlayState: &EmbyPlayState{PositionTicks: 0},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.GetPositionSeconds()
			if result != tt.expected {
				t.Errorf("GetPositionSeconds() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestEmbySession_GetDurationSeconds(t *testing.T) {
	tests := []struct {
		name     string
		session  EmbySession
		expected int64
	}{
		{
			name: "valid duration",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{RunTimeTicks: 72000000000}, // 7200 seconds (2 hours) in ticks
			},
			expected: 7200,
		},
		{
			name:     "no now playing item",
			session:  EmbySession{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.GetDurationSeconds()
			if result != tt.expected {
				t.Errorf("GetDurationSeconds() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestEmbySession_GetPercentComplete(t *testing.T) {
	tests := []struct {
		name     string
		session  EmbySession
		expected int
	}{
		{
			name: "50 percent",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{RunTimeTicks: 1000000000000},
				PlayState:      &EmbyPlayState{PositionTicks: 500000000000},
			},
			expected: 50,
		},
		{
			name: "zero duration",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{RunTimeTicks: 0},
				PlayState:      &EmbyPlayState{PositionTicks: 0},
			},
			expected: 0,
		},
		{
			name:     "empty session",
			session:  EmbySession{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.GetPercentComplete()
			if result != tt.expected {
				t.Errorf("GetPercentComplete() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestEmbySession_GetTranscodeDecision(t *testing.T) {
	tests := []struct {
		name       string
		playMethod string
		expected   string
	}{
		{"direct play", "DirectPlay", "direct play"},
		{"direct stream", "DirectStream", "direct stream"},
		{"transcode", "Transcode", "transcode"},
		{"unknown method", "SomeOtherMethod", "SomeOtherMethod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := EmbySession{
				PlayState: &EmbyPlayState{PlayMethod: tt.playMethod},
			}
			result := session.GetTranscodeDecision()
			if result != tt.expected {
				t.Errorf("GetTranscodeDecision() = %q, want %q", result, tt.expected)
			}
		})
	}

	// Test nil play state
	t.Run("nil play state", func(t *testing.T) {
		session := EmbySession{}
		result := session.GetTranscodeDecision()
		if result != "" {
			t.Errorf("GetTranscodeDecision() = %q, want empty", result)
		}
	})
}

func TestEmbyNowPlayingItem_GetMediaType(t *testing.T) {
	tests := []struct {
		itemType string
		expected string
	}{
		{"Movie", "movie"},
		{"MOVIE", "movie"},
		{"Episode", "episode"},
		{"Audio", "track"},
		{"MusicVideo", "track"},
		{"Photo", "photo"},
		{"Unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.itemType, func(t *testing.T) {
			item := EmbyNowPlayingItem{Type: tt.itemType}
			result := item.GetMediaType()
			if result != tt.expected {
				t.Errorf("GetMediaType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEmbySession_GetContentTitle(t *testing.T) {
	tests := []struct {
		name     string
		session  EmbySession
		expected string
	}{
		{
			name: "movie",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{Name: "The Matrix"},
			},
			expected: "The Matrix",
		},
		{
			name: "tv show episode",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{
					Name:              "Pilot",
					SeriesName:        "Breaking Bad",
					ParentIndexNumber: 1,
					IndexNumber:       1,
				},
			},
			expected: "Breaking Bad - S01E01 - Pilot",
		},
		{
			name: "music track",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{
					Name:    "Bohemian Rhapsody",
					Album:   "A Night at the Opera",
					Artists: []string{"Queen"},
				},
			},
			expected: "Queen - Bohemian Rhapsody",
		},
		{
			name: "music track with album artist",
			session: EmbySession{
				NowPlayingItem: &EmbyNowPlayingItem{
					Name:        "Bohemian Rhapsody",
					Album:       "A Night at the Opera",
					AlbumArtist: "Queen",
				},
			},
			expected: "Queen - Bohemian Rhapsody",
		},
		{
			name:     "empty session",
			session:  EmbySession{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.GetContentTitle()
			if result != tt.expected {
				t.Errorf("GetContentTitle() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEmbySession_IsTranscoding(t *testing.T) {
	tests := []struct {
		name     string
		session  EmbySession
		expected bool
	}{
		{
			name: "transcoding",
			session: EmbySession{
				PlayState: &EmbyPlayState{PlayMethod: "Transcode"},
			},
			expected: true,
		},
		{
			name: "direct play",
			session: EmbySession{
				PlayState: &EmbyPlayState{PlayMethod: "DirectPlay"},
			},
			expected: false,
		},
		{
			name:     "no play state",
			session:  EmbySession{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.IsTranscoding()
			if result != tt.expected {
				t.Errorf("IsTranscoding() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEmbySession_GetHardwareAccelerationType(t *testing.T) {
	tests := []struct {
		name     string
		session  EmbySession
		expected string
	}{
		{
			name: "vaapi",
			session: EmbySession{
				TranscodingInfo: &EmbyTranscodingInfo{HardwareAccelerationType: "vaapi"},
			},
			expected: "VAAPI",
		},
		{
			name: "nvenc",
			session: EmbySession{
				TranscodingInfo: &EmbyTranscodingInfo{HardwareAccelerationType: "nvenc"},
			},
			expected: "NVENC",
		},
		{
			name: "qsv",
			session: EmbySession{
				TranscodingInfo: &EmbyTranscodingInfo{HardwareAccelerationType: "qsv"},
			},
			expected: "Quick Sync",
		},
		{
			name: "videotoolbox",
			session: EmbySession{
				TranscodingInfo: &EmbyTranscodingInfo{HardwareAccelerationType: "videotoolbox"},
			},
			expected: "VideoToolbox",
		},
		{
			name: "software transcode",
			session: EmbySession{
				TranscodingInfo: &EmbyTranscodingInfo{VideoCodec: "h264"},
			},
			expected: "Software",
		},
		{
			name: "unknown hw accel",
			session: EmbySession{
				TranscodingInfo: &EmbyTranscodingInfo{HardwareAccelerationType: "custom"},
			},
			expected: "custom",
		},
		{
			name: "direct play",
			session: EmbySession{
				PlayState: &EmbyPlayState{PlayMethod: "DirectPlay"},
			},
			expected: "Direct Play",
		},
		{
			name:     "empty session",
			session:  EmbySession{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.GetHardwareAccelerationType()
			if result != tt.expected {
				t.Errorf("GetHardwareAccelerationType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEmbyWebhook_IsPlaybackEvent(t *testing.T) {
	tests := []struct {
		event    string
		expected bool
	}{
		{"playback.start", true},
		{"playback.progress", true},
		{"playback.stop", true},
		{"library.new", false},
		{"system.updateavailable", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			webhook := EmbyWebhook{Event: tt.event}
			result := webhook.IsPlaybackEvent()
			if result != tt.expected {
				t.Errorf("IsPlaybackEvent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEmbyWebhook_GetMediaType(t *testing.T) {
	tests := []struct {
		itemType string
		expected string
	}{
		{"Movie", "movie"},
		{"Episode", "episode"},
		{"Audio", "track"},
		{"Other", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.itemType, func(t *testing.T) {
			webhook := EmbyWebhook{ItemType: tt.itemType}
			result := webhook.GetMediaType()
			if result != tt.expected {
				t.Errorf("GetMediaType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEmbyWebhook_GetContentTitle(t *testing.T) {
	tests := []struct {
		name     string
		webhook  EmbyWebhook
		expected string
	}{
		{
			name:     "movie",
			webhook:  EmbyWebhook{ItemName: "The Matrix"},
			expected: "The Matrix",
		},
		{
			name: "tv show",
			webhook: EmbyWebhook{
				ItemName:   "Pilot",
				SeriesName: "Breaking Bad",
				SeasonName: "Season 1",
			},
			expected: "Breaking Bad - Season 1 - Pilot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.webhook.GetContentTitle()
			if result != tt.expected {
				t.Errorf("GetContentTitle() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEmbyWebhook_GetPercentComplete(t *testing.T) {
	tests := []struct {
		name                  string
		runTimeTicks          int64
		playbackPositionTicks int64
		expected              int
	}{
		{"50 percent", 1000000000, 500000000, 50},
		{"zero runtime", 0, 500000000, 0},
		{"100 percent", 1000000000, 1000000000, 100},
		{"0 percent", 1000000000, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := EmbyWebhook{
				RunTimeTicks:          tt.runTimeTicks,
				PlaybackPositionTicks: tt.playbackPositionTicks,
			}
			result := webhook.GetPercentComplete()
			if result != tt.expected {
				t.Errorf("GetPercentComplete() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestEmbyWebhook_GetIPAddress(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{"ipv4 with port", "192.168.1.100:52341", "192.168.1.100"},
		{"ipv4 without port", "192.168.1.100", "192.168.1.100"},
		{"ipv6 with port", "[2001:db8::1]:52341", "2001:db8::1"},
		{"ipv6 without port", "[2001:db8::1]", "2001:db8::1"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := EmbyWebhook{RemoteEndPoint: tt.endpoint}
			result := webhook.GetIPAddress()
			if result != tt.expected {
				t.Errorf("GetIPAddress() = %q, want %q", result, tt.expected)
			}
		})
	}
}
