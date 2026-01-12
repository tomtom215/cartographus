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

func TestNewGeoRestrictionDetector(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)

	if detector == nil {
		t.Fatal("detector should not be nil")
	}
	if detector.Type() != RuleTypeGeoRestriction {
		t.Errorf("Type() = %v, want %v", detector.Type(), RuleTypeGeoRestriction)
	}
	if detector.Enabled() {
		t.Error("detector should be disabled by default")
	}
}

func TestGeoRestrictionDetector_Check_Disabled(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)

	event := &DetectionEvent{
		UserID:   1,
		Username: "testuser",
		Country:  "US",
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when detector is disabled")
	}
}

func TestGeoRestrictionDetector_Check_NoCountry(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)
	detector.SetEnabled(true)

	event := &DetectionEvent{
		UserID:   1,
		Username: "testuser",
		Country:  "", // No country data
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert != nil {
		t.Error("expected no alert when country is empty")
	}
}

func TestGeoRestrictionDetector_Check_Blocklist(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)
	detector.SetEnabled(true)

	// Configure blocklist
	err := detector.Configure([]byte(`{"blocked_countries": ["RU", "CN"], "severity": "warning"}`))
	if err != nil {
		t.Fatalf("failed to configure: %v", err)
	}

	tests := []struct {
		name        string
		country     string
		expectAlert bool
	}{
		{
			name:        "blocked country RU",
			country:     "RU",
			expectAlert: true,
		},
		{
			name:        "blocked country CN",
			country:     "CN",
			expectAlert: true,
		},
		{
			name:        "allowed country US",
			country:     "US",
			expectAlert: false,
		},
		{
			name:        "allowed country UK",
			country:     "UK",
			expectAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Country:   tt.country,
				IPAddress: "1.2.3.4",
				Timestamp: time.Now(),
			}

			alert, err := detector.Check(context.Background(), event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectAlert && alert == nil {
				t.Error("expected alert but got nil")
			}
			if !tt.expectAlert && alert != nil {
				t.Errorf("expected no alert but got: %s", alert.Message)
			}
			if alert != nil {
				if alert.RuleType != RuleTypeGeoRestriction {
					t.Errorf("alert RuleType = %v, want %v", alert.RuleType, RuleTypeGeoRestriction)
				}
				if alert.Severity != SeverityWarning {
					t.Errorf("alert Severity = %v, want %v", alert.Severity, SeverityWarning)
				}
			}
		})
	}
}

func TestGeoRestrictionDetector_Check_Allowlist(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)
	detector.SetEnabled(true)

	// Configure allowlist
	err := detector.Configure([]byte(`{"allowed_countries": ["US", "CA", "UK"], "severity": "critical"}`))
	if err != nil {
		t.Fatalf("failed to configure: %v", err)
	}

	tests := []struct {
		name        string
		country     string
		expectAlert bool
	}{
		{
			name:        "allowed country US",
			country:     "US",
			expectAlert: false,
		},
		{
			name:        "allowed country CA",
			country:     "CA",
			expectAlert: false,
		},
		{
			name:        "allowed country UK",
			country:     "UK",
			expectAlert: false,
		},
		{
			name:        "unauthorized country RU",
			country:     "RU",
			expectAlert: true,
		},
		{
			name:        "unauthorized country CN",
			country:     "CN",
			expectAlert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Country:   tt.country,
				IPAddress: "1.2.3.4",
				Timestamp: time.Now(),
			}

			alert, err := detector.Check(context.Background(), event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectAlert && alert == nil {
				t.Error("expected alert but got nil")
			}
			if !tt.expectAlert && alert != nil {
				t.Errorf("expected no alert but got: %s", alert.Message)
			}
			if alert != nil {
				if alert.Severity != SeverityCritical {
					t.Errorf("alert Severity = %v, want %v", alert.Severity, SeverityCritical)
				}
			}
		})
	}
}

func TestGeoRestrictionDetector_Configure_Invalid(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)

	tests := []struct {
		name   string
		config string
	}{
		{
			name:   "invalid json",
			config: `{invalid}`,
		},
		{
			name:   "no countries configured",
			config: `{"severity": "warning"}`,
		},
		{
			name:   "both blocklist and allowlist",
			config: `{"blocked_countries": ["RU"], "allowed_countries": ["US"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detector.Configure([]byte(tt.config))
			if err == nil {
				t.Error("expected error but got nil")
			}
		})
	}
}

func TestGeoRestrictionDetector_AddBlockedCountry(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)

	// Add a blocked country
	err := detector.AddBlockedCountry("RU")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it was added
	config := detector.Config()
	if len(config.BlockedCountries) != 1 || config.BlockedCountries[0] != "RU" {
		t.Errorf("BlockedCountries = %v, want [RU]", config.BlockedCountries)
	}

	// Add same country again (should be no-op)
	err = detector.AddBlockedCountry("RU")
	if err != nil {
		t.Fatalf("unexpected error on duplicate: %v", err)
	}
	config = detector.Config()
	if len(config.BlockedCountries) != 1 {
		t.Errorf("BlockedCountries should still have 1 entry, got %d", len(config.BlockedCountries))
	}

	// Add another country
	err = detector.AddBlockedCountry("CN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	config = detector.Config()
	if len(config.BlockedCountries) != 2 {
		t.Errorf("BlockedCountries should have 2 entries, got %d", len(config.BlockedCountries))
	}
}

func TestGeoRestrictionDetector_AddBlockedCountry_AllowlistActive(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)

	// Configure allowlist first
	err := detector.Configure([]byte(`{"allowed_countries": ["US"]}`))
	if err != nil {
		t.Fatalf("failed to configure: %v", err)
	}

	// Try to add blocked country - should fail
	err = detector.AddBlockedCountry("RU")
	if err == nil {
		t.Error("expected error when allowlist is active")
	}
}

func TestGeoRestrictionDetector_RemoveBlockedCountry(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)

	// Add some countries
	detector.AddBlockedCountry("RU")
	detector.AddBlockedCountry("CN")
	detector.AddBlockedCountry("KP")

	// Remove one
	detector.RemoveBlockedCountry("CN")

	config := detector.Config()
	if len(config.BlockedCountries) != 2 {
		t.Errorf("BlockedCountries should have 2 entries, got %d", len(config.BlockedCountries))
	}

	// Verify CN is removed
	for _, c := range config.BlockedCountries {
		if c == "CN" {
			t.Error("CN should have been removed")
		}
	}

	// Remove non-existent country (should be no-op)
	detector.RemoveBlockedCountry("XX")
	config = detector.Config()
	if len(config.BlockedCountries) != 2 {
		t.Errorf("BlockedCountries should still have 2 entries, got %d", len(config.BlockedCountries))
	}
}

func TestGeoRestrictionDetector_IsCountryBlocked_Blocklist(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)

	detector.AddBlockedCountry("RU")
	detector.AddBlockedCountry("CN")

	tests := []struct {
		country  string
		expected bool
	}{
		{"RU", true},
		{"CN", true},
		{"US", false},
		{"UK", false},
	}

	for _, tt := range tests {
		t.Run(tt.country, func(t *testing.T) {
			result := detector.IsCountryBlocked(tt.country)
			if result != tt.expected {
				t.Errorf("IsCountryBlocked(%s) = %v, want %v", tt.country, result, tt.expected)
			}
		})
	}
}

func TestGeoRestrictionDetector_IsCountryBlocked_Allowlist(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)

	// Configure allowlist
	err := detector.Configure([]byte(`{"allowed_countries": ["US", "CA"]}`))
	if err != nil {
		t.Fatalf("failed to configure: %v", err)
	}

	tests := []struct {
		country  string
		expected bool // true = blocked (not in allowlist)
	}{
		{"US", false},
		{"CA", false},
		{"RU", true},
		{"UK", true},
	}

	for _, tt := range tests {
		t.Run(tt.country, func(t *testing.T) {
			result := detector.IsCountryBlocked(tt.country)
			if result != tt.expected {
				t.Errorf("IsCountryBlocked(%s) = %v, want %v", tt.country, result, tt.expected)
			}
		})
	}
}

func TestGeoRestrictionDetector_EnableDisable(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)

	// Initially disabled
	if detector.Enabled() {
		t.Error("detector should be disabled by default")
	}

	// Enable
	detector.SetEnabled(true)
	if !detector.Enabled() {
		t.Error("detector should be enabled after SetEnabled(true)")
	}

	// Disable
	detector.SetEnabled(false)
	if detector.Enabled() {
		t.Error("detector should be disabled after SetEnabled(false)")
	}
}

func TestGeoRestrictionDetector_AlertMetadata(t *testing.T) {
	mock := &mockEventHistory{}
	detector := NewGeoRestrictionDetector(mock)
	detector.SetEnabled(true)

	// Configure blocklist
	err := detector.Configure([]byte(`{"blocked_countries": ["RU"], "severity": "critical"}`))
	if err != nil {
		t.Fatalf("failed to configure: %v", err)
	}

	event := &DetectionEvent{
		UserID:    42,
		Username:  "blocked_user",
		Country:   "RU",
		City:      "Moscow",
		IPAddress: "1.2.3.4",
		ServerID:  "server1",
		MachineID: "machine1",
		Timestamp: time.Now(),
	}

	alert, err := detector.Check(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert == nil {
		t.Fatal("expected alert but got nil")
	}

	// Verify alert fields
	if alert.UserID != 42 {
		t.Errorf("UserID = %d, want 42", alert.UserID)
	}
	if alert.Username != "blocked_user" {
		t.Errorf("Username = %s, want blocked_user", alert.Username)
	}
	if alert.ServerID != "server1" {
		t.Errorf("ServerID = %s, want server1", alert.ServerID)
	}
	if alert.IPAddress != "1.2.3.4" {
		t.Errorf("IPAddress = %s, want 1.2.3.4", alert.IPAddress)
	}
	if alert.Title != "Geographic Restriction Violation" {
		t.Errorf("Title = %s, want Geographic Restriction Violation", alert.Title)
	}
	if len(alert.Metadata) == 0 {
		t.Error("Metadata should not be empty")
	}
}
