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

func TestNewDeviceVelocityDetector(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewDeviceVelocityDetector(mock)

	if detector == nil {
		t.Fatal("detector should not be nil")
	}
	if detector.Type() != RuleTypeDeviceVelocity {
		t.Errorf("Type() = %v, want %v", detector.Type(), RuleTypeDeviceVelocity)
	}
	if !detector.Enabled() {
		t.Error("detector should be enabled by default")
	}
}

func TestDeviceVelocityDetector_Check_Disabled(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewDeviceVelocityDetector(mock)
	detector.SetEnabled(false)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		MachineID: "machine123",
		IPAddress: "1.2.3.4",
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when detector is disabled")
	}
}

func TestDeviceVelocityDetector_Check_NoMachineID(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewDeviceVelocityDetector(mock)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		MachineID: "", // No machine ID
		IPAddress: "1.2.3.4",
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when machine ID is empty")
	}
}

func TestDeviceVelocityDetector_Check_BelowThreshold(t *testing.T) {
	mock := &mockEventHistory{
		recentIPs: []string{"1.2.3.4", "1.2.3.5"}, // 2 IPs
	}
	detector := NewDeviceVelocityDetector(mock)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		MachineID: "machine123",
		IPAddress: "1.2.3.6", // 3rd unique IP, but threshold is 3
		Timestamp: time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when below threshold")
	}
}

func TestDeviceVelocityDetector_Check_ExceedsThreshold(t *testing.T) {
	mock := &mockEventHistory{
		recentIPs: []string{"1.2.3.4", "1.2.3.5", "1.2.3.6"}, // 3 IPs
	}
	detector := NewDeviceVelocityDetector(mock)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		MachineID: "machine123abc456",
		IPAddress: "1.2.3.7", // 4th unique IP, exceeds threshold of 3
		ServerID:  "server1",
		Timestamp: time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert == nil {
		t.Fatal("expected alert but got nil")
	}

	if alert.RuleType != RuleTypeDeviceVelocity {
		t.Errorf("RuleType = %v, want %v", alert.RuleType, RuleTypeDeviceVelocity)
	}
	if alert.UserID != 1 {
		t.Errorf("UserID = %d, want 1", alert.UserID)
	}
	if alert.ServerID != "server1" {
		t.Errorf("ServerID = %s, want server1", alert.ServerID)
	}
	if alert.Title != "Device IP Velocity Alert" {
		t.Errorf("Title = %s, want Device IP Velocity Alert", alert.Title)
	}
}

func TestDeviceVelocityDetector_Check_SameIPInList(t *testing.T) {
	mock := &mockEventHistory{
		recentIPs: []string{"1.2.3.4", "1.2.3.5"}, // 2 IPs
	}
	detector := NewDeviceVelocityDetector(mock)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		MachineID: "machine123",
		IPAddress: "1.2.3.4", // Same as in list
		Timestamp: time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when IP is already in list")
	}
}

func TestDeviceVelocityDetector_Configure(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewDeviceVelocityDetector(mock)

	tests := []struct {
		name        string
		config      string
		expectError bool
	}{
		{
			name:        "valid configuration",
			config:      `{"window_minutes": 10, "max_unique_ips": 5, "severity": "critical"}`,
			expectError: false,
		},
		{
			name:        "invalid json",
			config:      `{invalid}`,
			expectError: true,
		},
		{
			name:        "zero window",
			config:      `{"window_minutes": 0, "max_unique_ips": 5}`,
			expectError: true,
		},
		{
			name:        "negative window",
			config:      `{"window_minutes": -5, "max_unique_ips": 5}`,
			expectError: true,
		},
		{
			name:        "zero max IPs",
			config:      `{"window_minutes": 5, "max_unique_ips": 0}`,
			expectError: true,
		},
		{
			name:        "negative max IPs",
			config:      `{"window_minutes": 5, "max_unique_ips": -3}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detector.Configure([]byte(tt.config))
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	// Verify valid config was applied
	config := detector.Config()
	if config.WindowMinutes != 10 {
		t.Errorf("WindowMinutes = %d, want 10", config.WindowMinutes)
	}
	if config.MaxUniqueIPs != 5 {
		t.Errorf("MaxUniqueIPs = %d, want 5", config.MaxUniqueIPs)
	}
}

func TestDeviceVelocityDetector_EnableDisable(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewDeviceVelocityDetector(mock)

	if !detector.Enabled() {
		t.Error("detector should be enabled by default")
	}

	detector.SetEnabled(false)
	if detector.Enabled() {
		t.Error("detector should be disabled after SetEnabled(false)")
	}

	detector.SetEnabled(true)
	if !detector.Enabled() {
		t.Error("detector should be enabled after SetEnabled(true)")
	}
}

func TestTruncateMachineID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "short",
			expected: "short",
		},
		{
			input:    "exactly12ch",
			expected: "exactly12ch",
		},
		{
			input:    "verylongmachineid123456789",
			expected: "verylong...",
		},
		{
			input:    "",
			expected: "",
		},
		{
			input:    "12345678",
			expected: "12345678",
		},
		{
			input:    "123456789", // 9 chars, still within 12-char threshold
			expected: "123456789",
		},
		{
			input:    "1234567890123", // 13 chars, above 12-char threshold
			expected: "12345678...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateMachineID(tt.input)
			if result != tt.expected {
				t.Errorf("truncateMachineID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDeviceVelocityDetector_CustomConfig(t *testing.T) {
	mock := &mockEventHistory{
		recentIPs: []string{"1.2.3.1", "1.2.3.2", "1.2.3.3", "1.2.3.4", "1.2.3.5"}, // 5 IPs
	}
	detector := NewDeviceVelocityDetector(mock)

	// Configure with higher threshold
	err := detector.Configure([]byte(`{"window_minutes": 10, "max_unique_ips": 10, "severity": "info"}`))
	if err != nil {
		t.Fatalf("failed to configure: %v", err)
	}

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		MachineID: "machine123",
		IPAddress: "1.2.3.6", // 6th IP, but threshold is now 10
		Timestamp: time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert with higher threshold")
	}
}
