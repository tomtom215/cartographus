// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import "testing"

// Test assertion helpers with "check" prefix to avoid conflicts with existing helpers.
// Each helper encapsulates common nil-check + value comparison patterns.
// Using t.Helper() ensures error messages point to the calling line.

// checkStringEqual checks that got equals want, failing if not
func checkStringEqual(t *testing.T, fieldName, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected %q, got %q", fieldName, want, got)
	}
}

// checkStringEmpty checks that value is empty
func checkStringEmpty(t *testing.T, fieldName, value string) {
	t.Helper()
	if value != "" {
		t.Errorf("%s should be empty, got %q", fieldName, value)
	}
}

// checkIntEqual checks that got equals want
func checkIntEqual(t *testing.T, fieldName string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected %d, got %d", fieldName, want, got)
	}
}

// checkStringPtrNil checks that ptr is nil
func checkStringPtrNil(t *testing.T, fieldName string, ptr *string) {
	t.Helper()
	if ptr != nil {
		t.Errorf("%s should be nil, got %q", fieldName, *ptr)
	}
}

// checkStringPtrEqual checks that ptr is not nil and equals want
func checkStringPtrEqual(t *testing.T, fieldName string, ptr *string, want string) {
	t.Helper()
	if ptr == nil {
		t.Errorf("%s should not be nil, expected %q", fieldName, want)
		return
	}
	if *ptr != want {
		t.Errorf("%s: expected %q, got %q", fieldName, want, *ptr)
	}
}

// checkIntPtrNil checks that ptr is nil
func checkIntPtrNil(t *testing.T, fieldName string, ptr *int) {
	t.Helper()
	if ptr != nil {
		t.Errorf("%s should be nil, got %d", fieldName, *ptr)
	}
}

// checkIntPtrEqual checks that ptr is not nil and equals want
func checkIntPtrEqual(t *testing.T, fieldName string, ptr *int, want int) {
	t.Helper()
	if ptr == nil {
		t.Errorf("%s should not be nil, expected %d", fieldName, want)
		return
	}
	if *ptr != want {
		t.Errorf("%s: expected %d, got %d", fieldName, want, *ptr)
	}
}

// checkInt64PtrEqual checks that ptr is not nil and equals want
func checkInt64PtrEqual(t *testing.T, fieldName string, ptr *int64, want int64) {
	t.Helper()
	if ptr == nil {
		t.Errorf("%s should not be nil, expected %d", fieldName, want)
		return
	}
	if *ptr != want {
		t.Errorf("%s: expected %d, got %d", fieldName, want, *ptr)
	}
}

// checkFloat64PtrEqual checks that ptr is not nil and equals want
func checkFloat64PtrEqual(t *testing.T, fieldName string, ptr *float64, want float64) {
	t.Helper()
	if ptr == nil {
		t.Errorf("%s should not be nil, expected %f", fieldName, want)
		return
	}
	if *ptr != want {
		t.Errorf("%s: expected %f, got %f", fieldName, want, *ptr)
	}
}

// checkNil checks that ptr value represents nil
func checkNil(t *testing.T, fieldName string, isNil bool) {
	t.Helper()
	if !isNil {
		t.Errorf("%s should be nil", fieldName)
	}
}

// checkNoError fails the test if err is not nil
func checkNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// checkError fails the test if err is nil
func checkError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// checkErrorContains fails the test if err is nil or doesn't contain substr
func checkErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if got := err.Error(); len(got) < len(substr) || !containsString(got, substr) {
		t.Errorf("expected error containing %q, got %q", substr, got)
	}
}

// containsString checks if s contains substr (simple implementation to avoid import)
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// checkSliceLen checks that slice has expected length
func checkSliceLen(t *testing.T, name string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected length %d, got %d", name, want, got)
	}
}

// checkTrue checks that condition is true
func checkTrue(t *testing.T, description string, condition bool) {
	t.Helper()
	if !condition {
		t.Errorf("expected %s to be true", description)
	}
}
