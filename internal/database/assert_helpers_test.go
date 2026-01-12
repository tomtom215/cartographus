// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import "testing"

// Test assertion helpers with "check" prefix to avoid conflicts with existing helpers.
// Each helper encapsulates common validation patterns.
// Using t.Helper() ensures error messages point to the calling line.

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

// checkStringEqual checks that got equals want
func checkStringEqual(t *testing.T, fieldName, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected %q, got %q", fieldName, want, got)
	}
}

// checkStringNotEmpty checks that value is not empty
func checkStringNotEmpty(t *testing.T, fieldName, value string) {
	t.Helper()
	if value == "" {
		t.Errorf("%s should not be empty", fieldName)
	}
}

// checkIntPositive checks that value is positive (> 0)
func checkIntPositive(t *testing.T, fieldName string, value int) {
	t.Helper()
	if value <= 0 {
		t.Errorf("%s should be positive, got %d", fieldName, value)
	}
}

// checkIntNonNegative checks that value is non-negative (>= 0)
func checkIntNonNegative(t *testing.T, fieldName string, value int) {
	t.Helper()
	if value < 0 {
		t.Errorf("%s should be non-negative, got %d", fieldName, value)
	}
}

// checkIntInRange checks that value is in [minVal, maxVal] inclusive
func checkIntInRange(t *testing.T, fieldName string, value, minVal, maxVal int) {
	t.Helper()
	if value < minVal || value > maxVal {
		t.Errorf("%s: expected value in range [%d, %d], got %d", fieldName, minVal, maxVal, value)
	}
}

// checkSliceNotEmpty checks that slice length > 0
func checkSliceNotEmpty(t *testing.T, name string, length int) {
	t.Helper()
	if length == 0 {
		t.Errorf("%s should not be empty", name)
	}
}

// checkSliceEmpty checks that slice length == 0
func checkSliceEmpty(t *testing.T, name string, length int) {
	t.Helper()
	if length != 0 {
		t.Errorf("%s should be empty, got %d items", name, length)
	}
}

// checkSliceMaxLen checks that slice length <= maxLen
func checkSliceMaxLen(t *testing.T, name string, length, maxLen int) {
	t.Helper()
	if length > maxLen {
		t.Errorf("%s: expected at most %d items, got %d", name, maxLen, length)
	}
}

// checkSortedDescending checks that values are sorted in descending order
func checkSortedDescending(t *testing.T, name string, values []int) {
	t.Helper()
	for i := 1; i < len(values); i++ {
		if values[i-1] < values[i] {
			t.Errorf("%s not sorted descending: value at %d (%d) < value at %d (%d)",
				name, i-1, values[i-1], i, values[i])
			return
		}
	}
}

// checkUniqueStrings checks that all strings in the slice are unique
func checkUniqueStrings(t *testing.T, name string, values []string) {
	t.Helper()
	seen := make(map[string]bool)
	for _, v := range values {
		if seen[v] {
			t.Errorf("%s contains duplicate: %q", name, v)
			return
		}
		seen[v] = true
	}
}
