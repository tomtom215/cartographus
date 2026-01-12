// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package detection

import (
	"testing"
)

func TestTopic(t *testing.T) {
	expected := "playback.>"
	result := Topic()
	if result != expected {
		t.Errorf("Topic() = %q, want %q", result, expected)
	}
}

func TestAlertTopic(t *testing.T) {
	expected := "detection.alerts"
	result := AlertTopic()
	if result != expected {
		t.Errorf("AlertTopic() = %q, want %q", result, expected)
	}
}
