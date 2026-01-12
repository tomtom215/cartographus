// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import "testing"

func TestPlexTranscodeSessionInfo_GetHardwareAccelerationName(t *testing.T) {
	tests := []struct {
		name          string
		transcodeInfo PlexTranscodeSessionInfo
		expected      string
	}{
		{
			name: "decoding title",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwDecodingTitle: "Intel QuickSync Decoding",
			},
			expected: "Intel QuickSync Decoding",
		},
		{
			name: "encoding title",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwEncodingTitle: "NVIDIA NVENC Encoding",
			},
			expected: "NVIDIA NVENC Encoding",
		},
		{
			name: "qsv decoding",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwDecoding: "qsv",
			},
			expected: "Intel QuickSync",
		},
		{
			name: "qsv encoding",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwEncoding: "qsv",
			},
			expected: "Intel QuickSync",
		},
		{
			name: "nvenc",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwDecoding: "nvenc",
			},
			expected: "NVIDIA NVENC",
		},
		{
			name: "nvdec",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwDecoding: "nvdec",
			},
			expected: "NVIDIA NVENC",
		},
		{
			name: "vaapi",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwDecoding: "vaapi",
			},
			expected: "VAAPI",
		},
		{
			name: "videotoolbox",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwDecoding: "videotoolbox",
			},
			expected: "VideoToolbox",
		},
		{
			name: "mediacodec",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwDecoding: "mediacodec",
			},
			expected: "MediaCodec",
		},
		{
			name: "mediafoundation",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwDecoding: "mf",
			},
			expected: "MediaFoundation",
		},
		{
			name: "unknown hw accel",
			transcodeInfo: PlexTranscodeSessionInfo{
				TranscodeHwDecoding: "custom_hw",
			},
			expected: "custom_hw",
		},
		{
			name:          "software",
			transcodeInfo: PlexTranscodeSessionInfo{},
			expected:      "Software",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.transcodeInfo.GetHardwareAccelerationName()
			if result != tt.expected {
				t.Errorf("GetHardwareAccelerationName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPlexServerCapabilitiesContainer_HasPlexPass(t *testing.T) {
	tests := []struct {
		name         string
		subscription bool
		expected     bool
	}{
		{"with subscription", true, true},
		{"without subscription", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := PlexServerCapabilitiesContainer{MyPlexSubscription: tt.subscription}
			result := caps.HasPlexPass()
			if result != tt.expected {
				t.Errorf("HasPlexPass() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPlexServerCapabilitiesContainer_SupportsHardwareTranscoding(t *testing.T) {
	tests := []struct {
		name            string
		transcoderVideo bool
		expected        bool
	}{
		{"supports hw transcode", true, true},
		{"no hw transcode", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := PlexServerCapabilitiesContainer{TranscoderVideo: tt.transcoderVideo}
			result := caps.SupportsHardwareTranscoding()
			if result != tt.expected {
				t.Errorf("SupportsHardwareTranscoding() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPlexServerCapabilitiesContainer_SupportsLiveTV(t *testing.T) {
	tests := []struct {
		name     string
		liveTV   bool
		expected bool
	}{
		{"supports live tv", true, true},
		{"no live tv", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := PlexServerCapabilitiesContainer{LiveTV: tt.liveTV}
			result := caps.SupportsLiveTV()
			if result != tt.expected {
				t.Errorf("SupportsLiveTV() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPlexServerCapabilitiesContainer_SupportsSync(t *testing.T) {
	tests := []struct {
		name      string
		sync      bool
		allowSync bool
		expected  bool
	}{
		{"sync enabled", true, true, true},
		{"sync disabled", true, false, false},
		{"allowsync disabled", false, true, false},
		{"both disabled", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := PlexServerCapabilitiesContainer{
				Sync:      tt.sync,
				AllowSync: tt.allowSync,
			}
			result := caps.SupportsSync()
			if result != tt.expected {
				t.Errorf("SupportsSync() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPlexServerCapabilitiesContainer_IsClaimedAndConnected(t *testing.T) {
	tests := []struct {
		name        string
		claimed     bool
		myPlex      bool
		signinState string
		expected    bool
	}{
		{"fully connected", true, true, "ok", true},
		{"not claimed", false, true, "ok", false},
		{"no myplex", true, false, "ok", false},
		{"bad signin state", true, true, "failed", false},
		{"signin pending", true, true, "pending", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := PlexServerCapabilitiesContainer{
				Claimed:           tt.claimed,
				MyPlex:            tt.myPlex,
				MyPlexSigninState: tt.signinState,
			}
			result := caps.IsClaimedAndConnected()
			if result != tt.expected {
				t.Errorf("IsClaimedAndConnected() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPlexServerCapabilitiesContainer_GetActiveTranscodeSessions(t *testing.T) {
	tests := []struct {
		name     string
		sessions int
	}{
		{"no sessions", 0},
		{"one session", 1},
		{"multiple sessions", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := PlexServerCapabilitiesContainer{TranscoderActiveVideoSessions: tt.sessions}
			result := caps.GetActiveTranscodeSessions()
			if result != tt.sessions {
				t.Errorf("GetActiveTranscodeSessions() = %d, want %d", result, tt.sessions)
			}
		})
	}
}
