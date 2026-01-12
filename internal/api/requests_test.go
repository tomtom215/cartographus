// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"testing"

	"github.com/tomtom215/cartographus/internal/validation"
)

// ===================================================================================================
// PlaybacksRequest Tests
// ===================================================================================================

func TestPlaybacksRequest_Valid(t *testing.T) {
	tests := []struct {
		name    string
		request PlaybacksRequest
	}{
		{
			name: "default values",
			request: PlaybacksRequest{
				Limit:  100,
				Offset: 0,
			},
		},
		{
			name: "minimum limit",
			request: PlaybacksRequest{
				Limit:  1,
				Offset: 0,
			},
		},
		{
			name: "maximum limit",
			request: PlaybacksRequest{
				Limit:  1000,
				Offset: 0,
			},
		},
		{
			name: "maximum offset",
			request: PlaybacksRequest{
				Limit:  100,
				Offset: 1000000,
			},
		},
		{
			name: "with cursor",
			request: PlaybacksRequest{
				Limit:  100,
				Offset: 0,
				Cursor: "eyJzdGFydGVkX2F0IjoiMjAyNS0wMS0wMVQxMjowMDowMFoiLCJpZCI6ImFiYzEyMyJ9",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestPlaybacksRequest_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		request   PlaybacksRequest
		wantField string
	}{
		{
			name: "limit too low",
			request: PlaybacksRequest{
				Limit:  0,
				Offset: 0,
			},
			wantField: "Limit",
		},
		{
			name: "limit too high",
			request: PlaybacksRequest{
				Limit:  2000,
				Offset: 0,
			},
			wantField: "Limit",
		},
		{
			name: "negative offset",
			request: PlaybacksRequest{
				Limit:  100,
				Offset: -1,
			},
			wantField: "Offset",
		},
		{
			name: "offset too high",
			request: PlaybacksRequest{
				Limit:  100,
				Offset: 1000001,
			},
			wantField: "Offset",
		},
		{
			name: "invalid cursor",
			request: PlaybacksRequest{
				Limit:  100,
				Cursor: "not-valid-base64!!!",
			},
			wantField: "Cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err == nil {
				t.Fatal("ValidateStruct() should have returned an error")
			}

			errs := err.Errors()
			found := false
			for _, e := range errs {
				if e.Field() == tt.wantField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error on field %s, got: %v", tt.wantField, errs)
			}
		})
	}
}

// ===================================================================================================
// LocationsRequest Tests
// ===================================================================================================

func TestLocationsRequest_Valid(t *testing.T) {
	tests := []struct {
		name    string
		request LocationsRequest
	}{
		{
			name: "defaults",
			request: LocationsRequest{
				Limit: 100,
			},
		},
		{
			name: "with days",
			request: LocationsRequest{
				Limit: 100,
				Days:  30,
			},
		},
		{
			name: "max days",
			request: LocationsRequest{
				Limit: 100,
				Days:  3650,
			},
		},
		{
			name: "with dates",
			request: LocationsRequest{
				Limit:     100,
				StartDate: "2025-01-01T00:00:00Z",
				EndDate:   "2025-12-31T23:59:59Z",
			},
		},
		{
			name: "with users filter",
			request: LocationsRequest{
				Limit: 100,
				Users: "user1,user2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestLocationsRequest_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		request   LocationsRequest
		wantField string
	}{
		{
			name: "limit too low",
			request: LocationsRequest{
				Limit: 0,
			},
			wantField: "Limit",
		},
		{
			name: "days too high",
			request: LocationsRequest{
				Limit: 100,
				Days:  4000,
			},
			wantField: "Days",
		},
		{
			name: "invalid start date",
			request: LocationsRequest{
				Limit:     100,
				StartDate: "not-a-date",
			},
			wantField: "StartDate",
		},
		{
			name: "invalid end date",
			request: LocationsRequest{
				Limit:   100,
				EndDate: "2025/01/01",
			},
			wantField: "EndDate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err == nil {
				t.Fatal("ValidateStruct() should have returned an error")
			}

			errs := err.Errors()
			found := false
			for _, e := range errs {
				if e.Field() == tt.wantField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error on field %s, got: %v", tt.wantField, errs)
			}
		})
	}
}

// ===================================================================================================
// LoginRequest Tests
// ===================================================================================================

func TestLoginRequestValidation_Valid(t *testing.T) {
	tests := []struct {
		name    string
		request LoginRequestValidation
	}{
		{
			name: "valid login",
			request: LoginRequestValidation{
				Username: "admin",
				Password: "password123",
			},
		},
		{
			name: "with remember me",
			request: LoginRequestValidation{
				Username:   "admin",
				Password:   "password123",
				RememberMe: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestLoginRequestValidation_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		request   LoginRequestValidation
		wantField string
	}{
		{
			name: "missing username",
			request: LoginRequestValidation{
				Password: "password123",
			},
			wantField: "Username",
		},
		{
			name: "missing password",
			request: LoginRequestValidation{
				Username: "admin",
			},
			wantField: "Password",
		},
		{
			name: "both empty",
			request: LoginRequestValidation{
				Username: "",
				Password: "",
			},
			wantField: "Username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err == nil {
				t.Fatal("ValidateStruct() should have returned an error")
			}

			errs := err.Errors()
			found := false
			for _, e := range errs {
				if e.Field() == tt.wantField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error on field %s, got: %v", tt.wantField, errs)
			}
		})
	}
}

// ===================================================================================================
// CreateBackupRequestValidation Tests
// ===================================================================================================

func TestCreateBackupRequestValidation_Valid(t *testing.T) {
	tests := []struct {
		name    string
		request CreateBackupRequestValidation
	}{
		{
			name:    "empty type defaults",
			request: CreateBackupRequestValidation{},
		},
		{
			name: "full backup",
			request: CreateBackupRequestValidation{
				Type: "full",
			},
		},
		{
			name: "database backup",
			request: CreateBackupRequestValidation{
				Type: "database",
			},
		},
		{
			name: "config backup",
			request: CreateBackupRequestValidation{
				Type: "config",
			},
		},
		{
			name: "with notes",
			request: CreateBackupRequestValidation{
				Type:  "full",
				Notes: "Pre-upgrade backup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestCreateBackupRequestValidation_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		request   CreateBackupRequestValidation
		wantField string
	}{
		{
			name: "invalid type",
			request: CreateBackupRequestValidation{
				Type: "invalid",
			},
			wantField: "Type",
		},
		{
			name: "notes too long",
			request: CreateBackupRequestValidation{
				Notes: string(make([]byte, 600)), // 600 chars > 500 max
			},
			wantField: "Notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err == nil {
				t.Fatal("ValidateStruct() should have returned an error")
			}

			errs := err.Errors()
			found := false
			for _, e := range errs {
				if e.Field() == tt.wantField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error on field %s, got: %v", tt.wantField, errs)
			}
		})
	}
}

// ===================================================================================================
// SetRetentionPolicyRequestValidation Tests
// ===================================================================================================

func TestSetRetentionPolicyRequestValidation_Valid(t *testing.T) {
	tests := []struct {
		name    string
		request SetRetentionPolicyRequestValidation
	}{
		{
			name:    "all zeros",
			request: SetRetentionPolicyRequestValidation{},
		},
		{
			name: "typical values",
			request: SetRetentionPolicyRequestValidation{
				MinCount:             3,
				MaxCount:             10,
				MaxAgeDays:           30,
				KeepRecentHours:      24,
				KeepDailyForDays:     7,
				KeepWeeklyForWeeks:   4,
				KeepMonthlyForMonths: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestSetRetentionPolicyRequestValidation_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		request   SetRetentionPolicyRequestValidation
		wantField string
	}{
		{
			name: "negative min count",
			request: SetRetentionPolicyRequestValidation{
				MinCount: -1,
			},
			wantField: "MinCount",
		},
		{
			name: "negative max count",
			request: SetRetentionPolicyRequestValidation{
				MaxCount: -1,
			},
			wantField: "MaxCount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err == nil {
				t.Fatal("ValidateStruct() should have returned an error")
			}

			errs := err.Errors()
			found := false
			for _, e := range errs {
				if e.Field() == tt.wantField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error on field %s, got: %v", tt.wantField, errs)
			}
		})
	}
}

// ===================================================================================================
// SpatialViewportRequest Tests
// ===================================================================================================

func TestSpatialViewportRequest_Valid(t *testing.T) {
	tests := []struct {
		name    string
		request SpatialViewportRequest
	}{
		{
			name: "simple viewport",
			request: SpatialViewportRequest{
				West:  -74.0060,
				South: 40.7128,
				East:  -73.9,
				North: 40.8,
			},
		},
		{
			name: "world bounds",
			request: SpatialViewportRequest{
				West:  -180,
				South: -90,
				East:  180,
				North: 90,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestSpatialViewportRequest_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		request   SpatialViewportRequest
		wantField string
	}{
		{
			name: "west out of range",
			request: SpatialViewportRequest{
				West:  -181,
				South: 40,
				East:  -73,
				North: 41,
			},
			wantField: "West",
		},
		{
			name: "south out of range",
			request: SpatialViewportRequest{
				West:  -74,
				South: -91,
				East:  -73,
				North: 41,
			},
			wantField: "South",
		},
		{
			name: "east out of range",
			request: SpatialViewportRequest{
				West:  -74,
				South: 40,
				East:  181,
				North: 41,
			},
			wantField: "East",
		},
		{
			name: "north out of range",
			request: SpatialViewportRequest{
				West:  -74,
				South: 40,
				East:  -73,
				North: 91,
			},
			wantField: "North",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err == nil {
				t.Fatal("ValidateStruct() should have returned an error")
			}

			errs := err.Errors()
			found := false
			for _, e := range errs {
				if e.Field() == tt.wantField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error on field %s, got: %v", tt.wantField, errs)
			}
		})
	}
}

// ===================================================================================================
// SpatialHexagonsRequest Tests
// ===================================================================================================

func TestSpatialHexagonsRequest_Valid(t *testing.T) {
	tests := []struct {
		name    string
		request SpatialHexagonsRequest
	}{
		{
			name: "default resolution",
			request: SpatialHexagonsRequest{
				Resolution: 7,
			},
		},
		{
			name: "min resolution",
			request: SpatialHexagonsRequest{
				Resolution: 0,
			},
		},
		{
			name: "max resolution",
			request: SpatialHexagonsRequest{
				Resolution: 15,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestSpatialHexagonsRequest_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		request   SpatialHexagonsRequest
		wantField string
	}{
		{
			name: "resolution too low",
			request: SpatialHexagonsRequest{
				Resolution: -1,
			},
			wantField: "Resolution",
		},
		{
			name: "resolution too high",
			request: SpatialHexagonsRequest{
				Resolution: 16,
			},
			wantField: "Resolution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err == nil {
				t.Fatal("ValidateStruct() should have returned an error")
			}

			errs := err.Errors()
			found := false
			for _, e := range errs {
				if e.Field() == tt.wantField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error on field %s, got: %v", tt.wantField, errs)
			}
		})
	}
}

// ===================================================================================================
// ExportPlaybacksCSVRequest Tests
// ===================================================================================================

func TestExportPlaybacksCSVRequest_Valid(t *testing.T) {
	tests := []struct {
		name    string
		request ExportPlaybacksCSVRequest
	}{
		{
			name: "default",
			request: ExportPlaybacksCSVRequest{
				Limit: 10000,
			},
		},
		{
			name: "min limit",
			request: ExportPlaybacksCSVRequest{
				Limit: 1,
			},
		},
		{
			name: "max limit",
			request: ExportPlaybacksCSVRequest{
				Limit: 100000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestExportPlaybacksCSVRequest_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		request   ExportPlaybacksCSVRequest
		wantField string
	}{
		{
			name: "limit too low",
			request: ExportPlaybacksCSVRequest{
				Limit: 0,
			},
			wantField: "Limit",
		},
		{
			name: "limit too high",
			request: ExportPlaybacksCSVRequest{
				Limit: 100001,
			},
			wantField: "Limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStruct(&tt.request)
			if err == nil {
				t.Fatal("ValidateStruct() should have returned an error")
			}

			errs := err.Errors()
			found := false
			for _, e := range errs {
				if e.Field() == tt.wantField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error on field %s, got: %v", tt.wantField, errs)
			}
		})
	}
}
