// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package validation

import (
	"testing"
)

// ===================================================================================================
// Singleton Validator Tests
// ===================================================================================================

func TestGetValidator_Singleton(t *testing.T) {
	// Test that GetValidator returns the same instance
	v1 := GetValidator()
	v2 := GetValidator()

	if v1 != v2 {
		t.Error("GetValidator() should return the same singleton instance")
	}

	if v1 == nil {
		t.Error("GetValidator() should not return nil")
	}
}

// ===================================================================================================
// ValidateStruct Tests
// ===================================================================================================

// TestStruct for basic validation tests
type TestStruct struct {
	Name    string `validate:"required,min=1,max=100"`
	Age     int    `validate:"min=0,max=150"`
	Email   string `validate:"omitempty,email"`
	Limit   int    `validate:"min=1,max=1000"`
	Offset  int    `validate:"min=0,max=1000000"`
	Enabled bool
}

func TestValidateStruct_Valid(t *testing.T) {
	tests := []struct {
		name   string
		input  TestStruct
		errMsg string
	}{
		{
			name: "all valid fields",
			input: TestStruct{
				Name:   "John Doe",
				Age:    30,
				Email:  "john@example.com",
				Limit:  100,
				Offset: 0,
			},
		},
		{
			name: "minimum values",
			input: TestStruct{
				Name:   "A",
				Age:    0,
				Email:  "",
				Limit:  1,
				Offset: 0,
			},
		},
		{
			name: "maximum values",
			input: TestStruct{
				Name:   "A",
				Age:    150,
				Email:  "",
				Limit:  1000,
				Offset: 1000000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.input)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestValidateStruct_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		input     TestStruct
		wantField string
		wantTag   string
	}{
		{
			name: "missing required name",
			input: TestStruct{
				Name:  "",
				Limit: 100,
			},
			wantField: "Name",
			wantTag:   "required",
		},
		{
			name: "age too high",
			input: TestStruct{
				Name: "John",
				Age:  200,
			},
			wantField: "Age",
			wantTag:   "max",
		},
		{
			name: "invalid email",
			input: TestStruct{
				Name:  "John",
				Email: "not-an-email",
			},
			wantField: "Email",
			wantTag:   "email",
		},
		{
			name: "limit too low",
			input: TestStruct{
				Name:  "John",
				Limit: 0,
			},
			wantField: "Limit",
			wantTag:   "min",
		},
		{
			name: "limit too high",
			input: TestStruct{
				Name:  "John",
				Limit: 2000,
			},
			wantField: "Limit",
			wantTag:   "max",
		},
		{
			name: "negative offset",
			input: TestStruct{
				Name:   "John",
				Limit:  100,
				Offset: -1,
			},
			wantField: "Offset",
			wantTag:   "min",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.input)
			if err == nil {
				t.Fatal("ValidateStruct() should have returned an error")
			}

			errs := err.Errors()
			if len(errs) == 0 {
				t.Fatal("ValidationErrors should contain at least one error")
			}

			found := false
			for _, e := range errs {
				if e.Field() == tt.wantField && e.Tag() == tt.wantTag {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error on field %s with tag %s, got: %v", tt.wantField, tt.wantTag, errs)
			}
		})
	}
}

// ===================================================================================================
// ToAPIError Tests
// ===================================================================================================

func TestToAPIError_SingleError(t *testing.T) {
	input := TestStruct{
		Name:  "", // required field missing
		Limit: 100,
	}

	err := ValidateStruct(&input)
	if err == nil {
		t.Fatal("Expected validation error")
	}

	apiErr := err.ToAPIError()

	if apiErr.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected code VALIDATION_ERROR, got %s", apiErr.Code)
	}

	if apiErr.Message == "" {
		t.Error("Expected non-empty message")
	}

	// Should contain field name in details
	if apiErr.Details == nil {
		t.Error("Expected details to be set")
	}
}

func TestToAPIError_MultipleErrors(t *testing.T) {
	input := TestStruct{
		Name:   "", // required field missing
		Age:    200,
		Limit:  0, // below minimum
		Offset: -1,
	}

	err := ValidateStruct(&input)
	if err == nil {
		t.Fatal("Expected validation error")
	}

	apiErr := err.ToAPIError()

	if apiErr.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected code VALIDATION_ERROR, got %s", apiErr.Code)
	}

	// Details should contain field information
	if apiErr.Details == nil {
		t.Error("Expected details to contain field information")
	}

	if _, ok := apiErr.Details["fields"]; !ok {
		t.Error("Expected details to contain 'fields' key")
	}
}

// ===================================================================================================
// Custom Validator Tests - Base64 Cursor
// ===================================================================================================

type CursorStruct struct {
	Cursor string `validate:"omitempty,base64url"`
}

func TestBase64URLValidation_Valid(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{"empty cursor", ""},
		{"valid base64url", "eyJzdGFydGVkX2F0IjoiMjAyNS0wMS0wMVQxMjowMDowMFoiLCJpZCI6ImFiYzEyMyJ9"},
		{"short cursor", "YWJj"},
		{"with padding", "YWJjZA=="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := CursorStruct{Cursor: tt.cursor}
			err := ValidateStruct(&input)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error for cursor %q: %v", tt.cursor, err)
			}
		})
	}
}

func TestBase64URLValidation_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{"invalid characters", "not-valid-base64!!!"},
		{"spaces", "abc def"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := CursorStruct{Cursor: tt.cursor}
			err := ValidateStruct(&input)
			if err == nil {
				t.Errorf("ValidateStruct() should have returned error for cursor %q", tt.cursor)
			}
		})
	}
}

// ===================================================================================================
// Datetime Validation Tests
// ===================================================================================================

type DateTimeStruct struct {
	StartDate string `validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	EndDate   string `validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

func TestDatetimeValidation_Valid(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		endDate   string
	}{
		{"empty dates", "", ""},
		{"valid RFC3339", "2025-01-15T10:30:00Z", "2025-12-31T23:59:59Z"},
		{"with timezone", "2025-01-15T10:30:00+05:00", ""},
		{"negative timezone", "2025-01-15T10:30:00-08:00", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := DateTimeStruct{
				StartDate: tt.startDate,
				EndDate:   tt.endDate,
			}
			err := ValidateStruct(&input)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestDatetimeValidation_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
	}{
		{"invalid format", "2025/01/15"},
		{"date only", "2025-01-15"},
		{"time only", "10:30:00"},
		{"garbage", "not-a-date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := DateTimeStruct{StartDate: tt.startDate}
			err := ValidateStruct(&input)
			if err == nil {
				t.Errorf("ValidateStruct() should have returned error for date %q", tt.startDate)
			}
		})
	}
}

// ===================================================================================================
// Oneof Validation Tests
// ===================================================================================================

type BackupTypeStruct struct {
	Type string `validate:"omitempty,oneof=full database config"`
}

func TestOneofValidation_Valid(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
	}{
		{"empty", ""},
		{"full", "full"},
		{"database", "database"},
		{"config", "config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := BackupTypeStruct{Type: tt.typeName}
			err := ValidateStruct(&input)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error for type %q: %v", tt.typeName, err)
			}
		})
	}
}

func TestOneofValidation_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
	}{
		{"invalid type", "invalid"},
		{"partial match", "fullx"},
		{"case sensitive", "Full"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := BackupTypeStruct{Type: tt.typeName}
			err := ValidateStruct(&input)
			if err == nil {
				t.Errorf("ValidateStruct() should have returned error for type %q", tt.typeName)
			}
		})
	}
}

// ===================================================================================================
// WithRequiredStructEnabled Tests
// ===================================================================================================

type NestedStruct struct {
	Inner InnerStruct `validate:"required"`
}

type InnerStruct struct {
	Value string `validate:"required"`
}

func TestNestedStructValidation(t *testing.T) {
	// Valid nested struct
	valid := NestedStruct{
		Inner: InnerStruct{Value: "test"},
	}

	err := ValidateStruct(&valid)
	if err != nil {
		t.Errorf("ValidateStruct() returned unexpected error for valid nested struct: %v", err)
	}

	// Invalid - missing inner value
	invalid := NestedStruct{
		Inner: InnerStruct{Value: ""},
	}

	err = ValidateStruct(&invalid)
	if err == nil {
		t.Error("ValidateStruct() should have returned error for invalid nested struct")
	}
}

// ===================================================================================================
// Latitude/Longitude Validation Tests
// ===================================================================================================

type CoordinatesStruct struct {
	Lat float64 `validate:"latitude"`
	Lon float64 `validate:"longitude"`
}

func TestCoordinateValidation_Valid(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{"origin", 0, 0},
		{"new york", 40.7128, -74.0060},
		{"tokyo", 35.6762, 139.6503},
		{"sydney", -33.8688, 151.2093},
		{"max lat", 90, 0},
		{"min lat", -90, 0},
		{"max lon", 0, 180},
		{"min lon", 0, -180},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := CoordinatesStruct{Lat: tt.lat, Lon: tt.lon}
			err := ValidateStruct(&input)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error for lat=%f, lon=%f: %v", tt.lat, tt.lon, err)
			}
		})
	}
}

func TestCoordinateValidation_Invalid(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{"lat too high", 91, 0},
		{"lat too low", -91, 0},
		{"lon too high", 0, 181},
		{"lon too low", 0, -181},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := CoordinatesStruct{Lat: tt.lat, Lon: tt.lon}
			err := ValidateStruct(&input)
			if err == nil {
				t.Errorf("ValidateStruct() should have returned error for lat=%f, lon=%f", tt.lat, tt.lon)
			}
		})
	}
}

// ===================================================================================================
// Integer Range Validation Tests
// ===================================================================================================

type RangeStruct struct {
	Days       int `validate:"omitempty,min=1,max=3650"`
	Resolution int `validate:"min=0,max=15"`
}

func TestRangeValidation_Valid(t *testing.T) {
	tests := []struct {
		name       string
		days       int
		resolution int
	}{
		{"zero values", 0, 0},
		{"typical values", 30, 7},
		{"max days", 3650, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := RangeStruct{Days: tt.days, Resolution: tt.resolution}
			err := ValidateStruct(&input)
			if err != nil {
				t.Errorf("ValidateStruct() returned unexpected error: %v", err)
			}
		})
	}
}

func TestRangeValidation_Invalid(t *testing.T) {
	tests := []struct {
		name       string
		days       int
		resolution int
		wantField  string
	}{
		{"days too high", 4000, 7, "Days"},
		{"days negative when set", -1, 7, "Days"},
		{"resolution too high", 30, 16, "Resolution"},
		{"resolution negative", 30, -1, "Resolution"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := RangeStruct{Days: tt.days, Resolution: tt.resolution}
			err := ValidateStruct(&input)
			if err == nil {
				t.Errorf("ValidateStruct() should have returned error for days=%d, resolution=%d", tt.days, tt.resolution)
			}
		})
	}
}

// ===================================================================================================
// Error Message Translation Tests
// ===================================================================================================

func TestErrorMessages(t *testing.T) {
	input := TestStruct{
		Name:  "",
		Limit: 0,
	}

	err := ValidateStruct(&input)
	if err == nil {
		t.Fatal("Expected validation error")
	}

	// Error message should be human-readable
	msg := err.Error()
	if msg == "" {
		t.Error("Error message should not be empty")
	}

	// Should contain field name
	if !containsSubstring(msg, "Name") && !containsSubstring(msg, "Limit") {
		t.Errorf("Error message should reference failed field: %s", msg)
	}
}

// helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
