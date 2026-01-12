// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package bandwidth

import (
	"testing"
)

func TestEstimateBandwidth(t *testing.T) {
	tests := []struct {
		name              string
		resolution        string
		transcodeDecision string
		wantMbps          float64
	}{
		// 4K resolutions - direct play
		{"4K direct play (4k)", "4k", "direct play", 25.0},
		{"4K direct play (2160p)", "2160p", "direct play", 25.0},
		{"4K direct play (3840x2160)", "3840x2160", "direct play", 25.0},
		{"4K direct play (4096x2160)", "4096x2160", "directplay", 25.0},

		// 4K resolutions - transcode
		{"4K transcode", "4k", "transcode", 15.0},
		{"4K copy", "2160p", "copy", 15.0},

		// 1080p resolutions - direct play
		{"1080p direct play", "1080p", "direct play", 10.0},
		{"1080p direct play (resolution)", "1920x1080", "direct play", 10.0},

		// 1080p resolutions - transcode
		{"1080p transcode", "1080p", "transcode", 8.0},
		{"1080p copy", "1920x1080", "copy", 8.0},

		// 720p resolutions - direct play
		{"720p direct play", "720p", "direct play", 5.0},
		{"720p direct play (resolution)", "1280x720", "direct play", 5.0},

		// 720p resolutions - transcode
		{"720p transcode", "720p", "transcode", 4.0},
		{"720p copy", "1280x720", "copy", 4.0},

		// SD resolutions - direct play
		{"SD direct play", "sd", "direct play", 2.0},
		{"480p direct play", "480p", "direct play", 2.0},
		{"SD direct play (720x480)", "720x480", "direct play", 2.0},
		{"SD direct play (640x480)", "640x480", "direct play", 2.0},

		// SD resolutions - transcode
		{"SD transcode", "sd", "transcode", 1.5},
		{"480p copy", "480p", "copy", 1.5},

		// Unknown/default resolutions
		{"Unknown resolution", "unknown", "direct play", 5.0},
		{"Empty resolution", "", "direct play", 5.0},
		{"Random resolution", "xyz", "transcode", 5.0},

		// Case insensitive
		{"Uppercase 1080P", "1080P", "direct play", 10.0},
		{"Mixed case 4K", "4K", "DIRECT PLAY", 25.0},
		{"Uppercase TRANSCODE", "720p", "TRANSCODE", 4.0},

		// Whitespace handling
		{"Resolution with spaces", " 1080p ", "direct play", 10.0},
		{"Decision with spaces", "720p", " transcode ", 4.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateBandwidth(tt.resolution, tt.transcodeDecision)
			if got != tt.wantMbps {
				t.Errorf("EstimateBandwidth(%q, %q) = %v, want %v",
					tt.resolution, tt.transcodeDecision, got, tt.wantMbps)
			}
		})
	}
}

func TestNormalizeResolution(t *testing.T) {
	tests := []struct {
		name       string
		resolution string
		want       string
	}{
		// 4K variants
		{"4k", "4k", "4k"},
		{"4K uppercase", "4K", "4k"},
		{"2160p", "2160p", "4k"},
		{"3840x2160", "3840x2160", "4k"},
		{"4096x2160", "4096x2160", "4k"},

		// 1080p variants
		{"1080p", "1080p", "1080p"},
		{"1080P uppercase", "1080P", "1080p"},
		{"1920x1080", "1920x1080", "1080p"},

		// 720p variants
		{"720p", "720p", "720p"},
		{"720P uppercase", "720P", "720p"},
		{"1280x720", "1280x720", "720p"},

		// SD variants
		{"sd", "sd", "sd"},
		{"SD uppercase", "SD", "sd"},
		{"480p", "480p", "sd"},
		{"720x480", "720x480", "sd"},
		{"640x480", "640x480", "sd"},

		// Unknown
		{"unknown", "unknown", "unknown"},
		{"empty", "", "unknown"},
		{"random", "random", "unknown"},
		{"360p", "360p", "unknown"},

		// Whitespace
		{"with spaces", " 1080p ", "1080p"},
		{"tabs", "\t720p\t", "720p"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeResolution(tt.resolution)
			if got != tt.want {
				t.Errorf("normalizeResolution(%q) = %v, want %v",
					tt.resolution, got, tt.want)
			}
		})
	}
}

func TestIsTranscodeDecision(t *testing.T) {
	tests := []struct {
		name     string
		decision string
		want     bool
	}{
		// Transcode decisions (true)
		{"transcode lowercase", "transcode", true},
		{"transcode uppercase", "TRANSCODE", true},
		{"transcode mixed case", "Transcode", true},
		{"copy lowercase", "copy", true},
		{"copy uppercase", "COPY", true},
		{"copy mixed case", "Copy", true},

		// Direct play decisions (false)
		{"direct play lowercase", "direct play", false},
		{"direct play uppercase", "DIRECT PLAY", false},
		{"directplay no space", "directplay", false},
		{"directplay uppercase", "DIRECTPLAY", false},

		// Unknown decisions (false)
		{"unknown", "unknown", false},
		{"empty", "", false},
		{"random", "random", false},

		// Whitespace
		{"transcode with spaces", " transcode ", true},
		{"direct play with spaces", " direct play ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTranscodeDecision(tt.decision)
			if got != tt.want {
				t.Errorf("isTranscodeDecision(%q) = %v, want %v",
					tt.decision, got, tt.want)
			}
		})
	}
}

func TestCalculateBandwidthGB(t *testing.T) {
	tests := []struct {
		name            string
		bandwidthMbps   float64
		durationSeconds int
		wantGB          float64
	}{
		{
			name:            "1080p direct play for 1 hour",
			bandwidthMbps:   10.0,
			durationSeconds: 3600, // 1 hour
			wantGB:          4.39453125,
		},
		{
			name:            "4K direct play for 2 hours",
			bandwidthMbps:   25.0,
			durationSeconds: 7200,       // 2 hours
			wantGB:          21.9726562, // Approximately 22 GB
		},
		{
			name:            "720p transcode for 30 minutes",
			bandwidthMbps:   4.0,
			durationSeconds: 1800, // 30 minutes
			wantGB:          0.87890625,
		},
		{
			name:            "SD for 10 minutes",
			bandwidthMbps:   1.5,
			durationSeconds: 600, // 10 minutes
			wantGB:          0.1098632812,
		},
		{
			name:            "Zero duration",
			bandwidthMbps:   10.0,
			durationSeconds: 0,
			wantGB:          0.0,
		},
		{
			name:            "Zero bandwidth",
			bandwidthMbps:   0.0,
			durationSeconds: 3600,
			wantGB:          0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateBandwidthGB(tt.bandwidthMbps, tt.durationSeconds)

			// Use tolerance for floating point comparison
			tolerance := 0.00001
			diff := got - tt.wantGB
			if diff < 0 {
				diff = -diff
			}

			if diff > tolerance {
				t.Errorf("CalculateBandwidthGB(%v, %v) = %v, want %v (diff: %v)",
					tt.bandwidthMbps, tt.durationSeconds, got, tt.wantGB, diff)
			}
		})
	}
}

// TestEstimateBandwidthRealWorld tests realistic scenarios
func TestEstimateBandwidthRealWorld(t *testing.T) {
	tests := []struct {
		name              string
		resolution        string
		transcodeDecision string
		durationMinutes   int
		wantGB            float64
		description       string
	}{
		{
			name:              "Movie: 4K direct play 2h",
			resolution:        "4k",
			transcodeDecision: "direct play",
			durationMinutes:   120,
			wantGB:            21.97,
			description:       "Typical 4K movie direct play",
		},
		{
			name:              "TV Show: 1080p transcode 45min",
			resolution:        "1080p",
			transcodeDecision: "transcode",
			durationMinutes:   45,
			wantGB:            2.64,
			description:       "TV episode transcoded from 1080p",
		},
		{
			name:              "Anime: 720p copy 24min",
			resolution:        "720p",
			transcodeDecision: "copy",
			durationMinutes:   24,
			wantGB:            0.70,
			description:       "Anime episode with copy transcode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mbps := EstimateBandwidth(tt.resolution, tt.transcodeDecision)
			durationSeconds := tt.durationMinutes * 60
			gotGB := CalculateBandwidthGB(mbps, durationSeconds)

			// Allow 1% tolerance for real-world scenarios
			tolerance := tt.wantGB * 0.01
			diff := gotGB - tt.wantGB
			if diff < 0 {
				diff = -diff
			}

			if diff > tolerance {
				t.Logf("%s: Expected ~%.2f GB, got %.2f GB (%.1f Mbps for %d min)",
					tt.description, tt.wantGB, gotGB, mbps, tt.durationMinutes)
				t.Errorf("Real-world scenario failed: diff %.2f GB exceeds tolerance %.2f GB",
					diff, tolerance)
			}
		})
	}
}

// BenchmarkEstimateBandwidth measures performance of bandwidth estimation
func BenchmarkEstimateBandwidth(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = EstimateBandwidth("1080p", "direct play")
	}
}

// BenchmarkCalculateBandwidthGB measures performance of GB calculation
func BenchmarkCalculateBandwidthGB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CalculateBandwidthGB(10.0, 3600)
	}
}
