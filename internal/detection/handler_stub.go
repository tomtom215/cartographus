// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package detection

// Topic returns the recommended NATS topic pattern for detection.
// Format: playback.> (subscribes to all playback events)
func Topic() string {
	return "playback.>"
}

// AlertTopic returns the topic for publishing detection alerts.
func AlertTopic() string {
	return "detection.alerts"
}
