// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package bandwidth

import "strings"

// ResolutionProfile defines bandwidth estimates for a resolution tier
type ResolutionProfile struct {
	DirectPlayMbps float64 // Bandwidth for direct play (no transcoding)
	TranscodeMbps  float64 // Bandwidth for transcoded streams
}

// resolutionProfiles maps normalized resolution names to bandwidth profiles
// These estimates are based on typical streaming quality requirements
var resolutionProfiles = map[string]ResolutionProfile{
	"4k": {
		DirectPlayMbps: 25.0, // 25 Mbps for direct play 4K
		TranscodeMbps:  15.0, // 15 Mbps for transcoded 4K
	},
	"1080p": {
		DirectPlayMbps: 10.0, // 10 Mbps for direct play 1080p
		TranscodeMbps:  8.0,  // 8 Mbps for transcoded 1080p
	},
	"720p": {
		DirectPlayMbps: 5.0, // 5 Mbps for direct play 720p
		TranscodeMbps:  4.0, // 4 Mbps for transcoded 720p
	},
	"sd": {
		DirectPlayMbps: 2.0, // 2 Mbps for direct play SD
		TranscodeMbps:  1.5, // 1.5 Mbps for transcoded SD
	},
}

// defaultBandwidthMbps is the fallback bandwidth estimate for unknown resolutions
const defaultBandwidthMbps = 5.0

// EstimateBandwidth calculates estimated bandwidth in Mbps based on resolution and transcode decision
//
// Parameters:
//   - resolution: Video resolution (e.g., "1080p", "720p", "4k", "sd")
//   - transcodeDecision: Transcode decision ("direct play", "transcode", "copy", etc.)
//
// Returns:
//   - Estimated bandwidth in Mbps
//
// Examples:
//   - EstimateBandwidth("1080p", "direct play") = 10.0 Mbps
//   - EstimateBandwidth("1080p", "transcode") = 8.0 Mbps
//   - EstimateBandwidth("unknown", "direct play") = 5.0 Mbps (default)
func EstimateBandwidth(resolution string, transcodeDecision string) float64 {
	normalizedResolution := normalizeResolution(resolution)
	isTranscode := isTranscodeDecision(transcodeDecision)

	profile, ok := resolutionProfiles[normalizedResolution]
	if !ok {
		return defaultBandwidthMbps
	}

	if isTranscode {
		return profile.TranscodeMbps
	}
	return profile.DirectPlayMbps
}

// normalizeResolution converts various resolution formats to standard names
//
// Supported formats:
//   - "4k", "2160p", "3840x2160", "4096x2160" → "4k"
//   - "1080p", "1920x1080" → "1080p"
//   - "720p", "1280x720" → "720p"
//   - "sd", "480p", "720x480", "640x480" → "sd"
//   - "unknown", "" → "unknown"
func normalizeResolution(resolution string) string {
	normalized := strings.ToLower(strings.TrimSpace(resolution))

	switch normalized {
	case "4k", "2160p", "3840x2160", "4096x2160":
		return "4k"
	case "1080p", "1920x1080":
		return "1080p"
	case "720p", "1280x720":
		return "720p"
	case "sd", "480p", "720x480", "640x480":
		return "sd"
	default:
		return "unknown"
	}
}

// isTranscodeDecision determines if a transcode decision represents transcoding
//
// Transcode decisions include: "transcode", "copy"
// Direct play decisions include: "direct play", "directplay"
func isTranscodeDecision(decision string) bool {
	normalized := strings.ToLower(strings.TrimSpace(decision))
	return normalized == "transcode" || normalized == "copy"
}

// CalculateBandwidthGB converts bandwidth in Mbps and duration to total bandwidth in GB
//
// Parameters:
//   - bandwidthMbps: Bandwidth in megabits per second
//   - durationSeconds: Duration in seconds
//
// Returns:
//   - Total bandwidth in gigabytes
//
// Formula: (Mbps * seconds) / 8 (bits to bytes) / 1024 (MB to GB)
func CalculateBandwidthGB(bandwidthMbps float64, durationSeconds int) float64 {
	return (bandwidthMbps * float64(durationSeconds)) / 8.0 / 1024.0
}
