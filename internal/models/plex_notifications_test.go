// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"testing"
)

// ===================================================================================================
// PlexPlayingNotification Tests
// ===================================================================================================

func TestPlexPlayingNotification_GetPlaybackState(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected string
	}{
		{"playing state", "playing", "Playing"},
		{"paused state", "paused", "Paused"},
		{"stopped state", "stopped", "Stopped"},
		{"buffering state", "buffering", "Buffering"},
		{"unknown state", "unknown", "Unknown"},
		{"empty state", "", "Unknown"},
		{"random string", "foo", "Unknown"},
		{"mixed case", "Playing", "Unknown"}, // Case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PlexPlayingNotification{State: tt.state}
			if got := p.GetPlaybackState(); got != tt.expected {
				t.Errorf("GetPlaybackState() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexPlayingNotification_IsBuffering(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected bool
	}{
		{"buffering state", "buffering", true},
		{"playing state", "playing", false},
		{"paused state", "paused", false},
		{"stopped state", "stopped", false},
		{"empty state", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PlexPlayingNotification{State: tt.state}
			if got := p.IsBuffering(); got != tt.expected {
				t.Errorf("IsBuffering() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexPlayingNotification_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected bool
	}{
		{"playing is active", "playing", true},
		{"paused is active", "paused", true},
		{"buffering is active", "buffering", true},
		{"stopped is not active", "stopped", false},
		{"empty is not active", "", false},
		{"unknown is not active", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PlexPlayingNotification{State: tt.state}
			if got := p.IsActive(); got != tt.expected {
				t.Errorf("IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ===================================================================================================
// PlexActivityNotification Tests
// ===================================================================================================

func TestPlexActivityNotification_GetActivityProgress(t *testing.T) {
	tests := []struct {
		name     string
		progress int
		expected int
	}{
		{"zero progress", 0, 0},
		{"partial progress", 50, 50},
		{"complete progress", 100, 100},
		{"negative progress", -1, -1}, // Edge case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &PlexActivityNotification{
				Activity: PlexActivityData{Progress: tt.progress},
			}
			if got := a.GetActivityProgress(); got != tt.expected {
				t.Errorf("GetActivityProgress() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestPlexActivityNotification_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		progress int
		expected bool
	}{
		{"ended event", "ended", 50, true},
		{"100% progress", "progress", 100, true},
		{"ended with 100%", "ended", 100, true},
		{"started event", "started", 0, false},
		{"progress event at 50%", "progress", 50, false},
		{"empty event at 100%", "", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &PlexActivityNotification{
				Event:    tt.event,
				Activity: PlexActivityData{Progress: tt.progress},
			}
			if got := a.IsComplete(); got != tt.expected {
				t.Errorf("IsComplete() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexActivityNotification_GetLibraryContext(t *testing.T) {
	tests := []struct {
		name       string
		sectionID  string
		expectNil  bool
		expectedID string
	}{
		{"with section ID", "123", false, "123"},
		{"empty section ID", "", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &PlexActivityNotification{
				Activity: PlexActivityData{
					Context: PlexActivityContext{
						LibrarySectionID:    tt.sectionID,
						LibrarySectionTitle: "Movies",
					},
				},
			}
			got := a.GetLibraryContext()
			if tt.expectNil {
				if got != nil {
					t.Error("GetLibraryContext() expected nil, got non-nil")
				}
			} else {
				if got == nil {
					t.Error("GetLibraryContext() expected non-nil, got nil")
				} else if got.LibrarySectionID != tt.expectedID {
					t.Errorf("GetLibraryContext().LibrarySectionID = %q, want %q", got.LibrarySectionID, tt.expectedID)
				}
			}
		})
	}
}

// ===================================================================================================
// PlexSession Tests
// ===================================================================================================

func TestPlexSession_IsTranscoding(t *testing.T) {
	tests := []struct {
		name     string
		session  PlexSession
		expected bool
	}{
		{
			name: "transcoding video",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{VideoDecision: "transcode"},
			},
			expected: true,
		},
		{
			name: "direct play",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{VideoDecision: "copy"},
			},
			expected: false,
		},
		{
			name:     "no transcode session",
			session:  PlexSession{TranscodeSession: nil},
			expected: false,
		},
		{
			name: "direct stream",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{VideoDecision: "direct play"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsTranscoding(); got != tt.expected {
				t.Errorf("IsTranscoding() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexSession_GetTranscodeProgress(t *testing.T) {
	tests := []struct {
		name     string
		session  PlexSession
		expected float64
	}{
		{
			name: "with progress",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{Progress: 75.5},
			},
			expected: 75.5,
		},
		{
			name: "zero progress",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{Progress: 0},
			},
			expected: 0,
		},
		{
			name:     "no transcode session",
			session:  PlexSession{TranscodeSession: nil},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.GetTranscodeProgress(); got != tt.expected {
				t.Errorf("GetTranscodeProgress() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexSession_IsHardwareAccelerated(t *testing.T) {
	tests := []struct {
		name     string
		session  PlexSession
		expected bool
	}{
		{
			name: "hw decoding",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwDecoding: "qsv"},
			},
			expected: true,
		},
		{
			name: "hw encoding",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwEncoding: "nvenc"},
			},
			expected: true,
		},
		{
			name: "both hw accel",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{
					TranscodeHwDecoding: "qsv",
					TranscodeHwEncoding: "qsv",
				},
			},
			expected: true,
		},
		{
			name: "software only",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{
					TranscodeHwDecoding: "",
					TranscodeHwEncoding: "",
				},
			},
			expected: false,
		},
		{
			name:     "no transcode session",
			session:  PlexSession{TranscodeSession: nil},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsHardwareAccelerated(); got != tt.expected {
				t.Errorf("IsHardwareAccelerated() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexSession_GetHardwareAccelerationType(t *testing.T) {
	tests := []struct {
		name     string
		session  PlexSession
		expected string
	}{
		{
			name:     "no transcode session",
			session:  PlexSession{TranscodeSession: nil},
			expected: "Direct Play",
		},
		{
			name: "software transcode",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{
					VideoDecision:       "transcode",
					TranscodeHwDecoding: "",
					TranscodeHwEncoding: "",
				},
			},
			expected: "Software",
		},
		{
			name: "direct play with empty transcode session",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{
					VideoDecision:       "copy",
					TranscodeHwDecoding: "",
					TranscodeHwEncoding: "",
				},
			},
			expected: "Direct Play",
		},
		{
			name: "Intel Quick Sync decoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwDecoding: "qsv"},
			},
			expected: "Quick Sync",
		},
		{
			name: "Intel Quick Sync encoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwEncoding: "qsv"},
			},
			expected: "Quick Sync",
		},
		{
			name: "NVIDIA NVENC decoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwDecoding: "nvenc"},
			},
			expected: "NVENC",
		},
		{
			name: "NVIDIA NVENC encoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwEncoding: "nvenc"},
			},
			expected: "NVENC",
		},
		{
			name: "VAAPI decoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwDecoding: "vaapi"},
			},
			expected: "VAAPI",
		},
		{
			name: "VAAPI encoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwEncoding: "vaapi"},
			},
			expected: "VAAPI",
		},
		{
			name: "Apple VideoToolbox decoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwDecoding: "videotoolbox"},
			},
			expected: "VideoToolbox",
		},
		{
			name: "Apple VideoToolbox encoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwEncoding: "videotoolbox"},
			},
			expected: "VideoToolbox",
		},
		{
			name: "Android MediaCodec decoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwDecoding: "mediacodec"},
			},
			expected: "MediaCodec",
		},
		{
			name: "Android MediaCodec encoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwEncoding: "mediacodec"},
			},
			expected: "MediaCodec",
		},
		{
			name: "Windows Media Foundation decoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwDecoding: "mf"},
			},
			expected: "MediaFoundation",
		},
		{
			name: "Windows Media Foundation encoder",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwEncoding: "mf"},
			},
			expected: "MediaFoundation",
		},
		{
			name: "unknown hw type",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{TranscodeHwDecoding: "someother"},
			},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.GetHardwareAccelerationType(); got != tt.expected {
				t.Errorf("GetHardwareAccelerationType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexSession_GetQualityTransition(t *testing.T) {
	tests := []struct {
		name     string
		session  PlexSession
		expected string
	}{
		{
			name:     "no media",
			session:  PlexSession{Media: nil},
			expected: "Unknown",
		},
		{
			name:     "empty media slice",
			session:  PlexSession{Media: []PlexMedia{}},
			expected: "Unknown",
		},
		{
			name: "direct play",
			session: PlexSession{
				Media:            []PlexMedia{{VideoResolution: "1080"}},
				TranscodeSession: nil,
			},
			expected: "1080",
		},
		{
			name: "same resolution",
			session: PlexSession{
				Media:            []PlexMedia{{VideoResolution: "1080p"}},
				TranscodeSession: &PlexTranscodeSession{Height: 1080},
			},
			expected: "1080p",
		},
		{
			name: "4K to 1080p",
			session: PlexSession{
				Media:            []PlexMedia{{VideoResolution: "4k"}},
				TranscodeSession: &PlexTranscodeSession{Height: 1080},
			},
			expected: "4k → 1080p",
		},
		{
			name: "1080p to 720p",
			session: PlexSession{
				Media:            []PlexMedia{{VideoResolution: "1080"}},
				TranscodeSession: &PlexTranscodeSession{Height: 720},
			},
			expected: "1080 → 720p",
		},
		{
			name: "1080p to 480p",
			session: PlexSession{
				Media:            []PlexMedia{{VideoResolution: "1080"}},
				TranscodeSession: &PlexTranscodeSession{Height: 480},
			},
			expected: "1080 → 480p",
		},
		{
			name: "1080p to SD",
			session: PlexSession{
				Media:            []PlexMedia{{VideoResolution: "1080"}},
				TranscodeSession: &PlexTranscodeSession{Height: 360},
			},
			expected: "1080 → SD",
		},
		{
			name: "1080p to 4K (upscale)",
			session: PlexSession{
				Media:            []PlexMedia{{VideoResolution: "1080"}},
				TranscodeSession: &PlexTranscodeSession{Height: 2160},
			},
			expected: "1080 → 4K",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.GetQualityTransition(); got != tt.expected {
				t.Errorf("GetQualityTransition() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexSession_GetCodecTransition(t *testing.T) {
	tests := []struct {
		name     string
		session  PlexSession
		expected string
	}{
		{
			name:     "no media",
			session:  PlexSession{Media: nil},
			expected: "Unknown",
		},
		{
			name:     "empty media slice",
			session:  PlexSession{Media: []PlexMedia{}},
			expected: "Unknown",
		},
		{
			name: "direct play hevc",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "hevc"}},
				TranscodeSession: nil,
			},
			expected: "HEVC",
		},
		{
			name: "direct play h264",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "h264"}},
				TranscodeSession: nil,
			},
			expected: "H.264",
		},
		{
			name: "same codec",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "h264"}},
				TranscodeSession: &PlexTranscodeSession{VideoCodec: "h264"},
			},
			expected: "H.264",
		},
		{
			name: "hevc to h264",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "hevc"}},
				TranscodeSession: &PlexTranscodeSession{VideoCodec: "h264"},
			},
			expected: "HEVC → H.264",
		},
		{
			name: "h265 to h264",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "h265"}},
				TranscodeSession: &PlexTranscodeSession{VideoCodec: "h264"},
			},
			expected: "HEVC → H.264",
		},
		{
			name: "vp9 to h264",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "vp9"}},
				TranscodeSession: &PlexTranscodeSession{VideoCodec: "h264"},
			},
			expected: "VP9 → H.264",
		},
		{
			name: "av1 to h264",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "av1"}},
				TranscodeSession: &PlexTranscodeSession{VideoCodec: "h264"},
			},
			expected: "AV1 → H.264",
		},
		{
			name: "mpeg4 to h264",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "mpeg4"}},
				TranscodeSession: &PlexTranscodeSession{VideoCodec: "h264"},
			},
			expected: "MPEG-4 → H.264",
		},
		{
			name: "mpeg2video to h264",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "mpeg2video"}},
				TranscodeSession: &PlexTranscodeSession{VideoCodec: "h264"},
			},
			expected: "MPEG-2 → H.264",
		},
		{
			name: "avc to h264 (same codec alias)",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "avc"}},
				TranscodeSession: &PlexTranscodeSession{VideoCodec: "h264"},
			},
			expected: "H.264 → H.264", // Both normalize to H.264 but different raw strings
		},
		{
			name: "unknown codec",
			session: PlexSession{
				Media:            []PlexMedia{{VideoCodec: "somecodec"}},
				TranscodeSession: &PlexTranscodeSession{VideoCodec: "h264"},
			},
			expected: "somecodec → H.264",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.GetCodecTransition(); got != tt.expected {
				t.Errorf("GetCodecTransition() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCodecToFriendlyName(t *testing.T) {
	tests := []struct {
		codec    string
		expected string
	}{
		{"hevc", "HEVC"},
		{"h265", "HEVC"},
		{"h264", "H.264"},
		{"avc", "H.264"},
		{"vp9", "VP9"},
		{"av1", "AV1"},
		{"mpeg4", "MPEG-4"},
		{"mpeg2video", "MPEG-2"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.codec, func(t *testing.T) {
			if got := codecToFriendlyName(tt.codec); got != tt.expected {
				t.Errorf("codecToFriendlyName(%q) = %q, want %q", tt.codec, got, tt.expected)
			}
		})
	}
}

func TestPlexSession_GetTranscodeSpeed(t *testing.T) {
	tests := []struct {
		name     string
		session  PlexSession
		expected string
	}{
		{
			name:     "no transcode session",
			session:  PlexSession{TranscodeSession: nil},
			expected: "N/A",
		},
		{
			name: "1x speed",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{Speed: 1.0},
			},
			expected: "1.0x",
		},
		{
			name: "1.5x speed",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{Speed: 1.5},
			},
			expected: "1.5x",
		},
		{
			name: "2.3x speed",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{Speed: 2.345},
			},
			expected: "2.3x",
		},
		{
			name: "zero speed",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{Speed: 0},
			},
			expected: "0.0x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.GetTranscodeSpeed(); got != tt.expected {
				t.Errorf("GetTranscodeSpeed() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexSession_IsThrottled(t *testing.T) {
	tests := []struct {
		name     string
		session  PlexSession
		expected bool
	}{
		{
			name:     "no transcode session",
			session:  PlexSession{TranscodeSession: nil},
			expected: false,
		},
		{
			name: "throttled",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{Throttled: true},
			},
			expected: true,
		},
		{
			name: "not throttled",
			session: PlexSession{
				TranscodeSession: &PlexTranscodeSession{Throttled: false},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsThrottled(); got != tt.expected {
				t.Errorf("IsThrottled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ===================================================================================================
// PlexWebhook Tests
// ===================================================================================================

func TestPlexWebhook_IsMediaEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		expected bool
	}{
		{"media.play", "media.play", true},
		{"media.pause", "media.pause", true},
		{"media.resume", "media.resume", true},
		{"media.stop", "media.stop", true},
		{"media.scrobble", "media.scrobble", true},
		{"media.rate", "media.rate", true},
		{"library.new", "library.new", false},
		{"admin.database.backup", "admin.database.backup", false},
		{"empty event", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &PlexWebhook{Event: tt.event}
			if got := w.IsMediaEvent(); got != tt.expected {
				t.Errorf("IsMediaEvent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexWebhook_IsLibraryEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		expected bool
	}{
		{"library.new", "library.new", true},
		{"library.on.deck", "library.on.deck", true},
		{"media.play", "media.play", false},
		{"admin.database.backup", "admin.database.backup", false},
		{"empty event", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &PlexWebhook{Event: tt.event}
			if got := w.IsLibraryEvent(); got != tt.expected {
				t.Errorf("IsLibraryEvent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexWebhook_IsAdminEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		expected bool
	}{
		{"admin.database.backup", "admin.database.backup", true},
		{"admin.database.corrupted", "admin.database.corrupted", true},
		{"device.new", "device.new", true},
		{"media.play", "media.play", false},
		{"library.new", "library.new", false},
		{"empty event", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &PlexWebhook{Event: tt.event}
			if got := w.IsAdminEvent(); got != tt.expected {
				t.Errorf("IsAdminEvent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexWebhook_GetUsername(t *testing.T) {
	tests := []struct {
		name     string
		account  PlexWebhookAccount
		expected string
	}{
		{
			name:     "regular username",
			account:  PlexWebhookAccount{Title: "testuser"},
			expected: "testuser",
		},
		{
			name:     "empty username",
			account:  PlexWebhookAccount{Title: ""},
			expected: "",
		},
		{
			name:     "special characters",
			account:  PlexWebhookAccount{Title: "user@domain.com"},
			expected: "user@domain.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &PlexWebhook{Account: tt.account}
			if got := w.GetUsername(); got != tt.expected {
				t.Errorf("GetUsername() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexWebhook_GetPlayerIP(t *testing.T) {
	tests := []struct {
		name     string
		player   PlexWebhookPlayer
		expected string
	}{
		{
			name:     "regular IP",
			player:   PlexWebhookPlayer{PublicAddress: "192.168.1.100"},
			expected: "192.168.1.100",
		},
		{
			name:     "empty IP",
			player:   PlexWebhookPlayer{PublicAddress: ""},
			expected: "",
		},
		{
			name:     "IPv6 address",
			player:   PlexWebhookPlayer{PublicAddress: "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
			expected: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &PlexWebhook{Player: tt.player}
			if got := w.GetPlayerIP(); got != tt.expected {
				t.Errorf("GetPlayerIP() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexWebhook_GetContentTitle(t *testing.T) {
	tests := []struct {
		name     string
		metadata *PlexWebhookMetadata
		expected string
	}{
		{
			name:     "nil metadata",
			metadata: nil,
			expected: "",
		},
		{
			name: "movie",
			metadata: &PlexWebhookMetadata{
				Title:            "Test Movie",
				GrandparentTitle: "",
			},
			expected: "Test Movie",
		},
		{
			name: "tv episode",
			metadata: &PlexWebhookMetadata{
				Title:            "Episode Title",
				GrandparentTitle: "Show Name",
				ParentIndex:      1,
				Index:            5,
			},
			expected: "Show Name - S01E05 - Episode Title",
		},
		{
			name: "tv episode season 10 ep 23",
			metadata: &PlexWebhookMetadata{
				Title:            "Episode Title",
				GrandparentTitle: "Show Name",
				ParentIndex:      10,
				Index:            23,
			},
			expected: "Show Name - S10E23 - Episode Title",
		},
		{
			name: "tv episode zero indexes",
			metadata: &PlexWebhookMetadata{
				Title:            "Pilot",
				GrandparentTitle: "New Show",
				ParentIndex:      0,
				Index:            0,
			},
			expected: "New Show - S00E00 - Pilot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &PlexWebhook{Metadata: tt.metadata}
			if got := w.GetContentTitle(); got != tt.expected {
				t.Errorf("GetContentTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}
