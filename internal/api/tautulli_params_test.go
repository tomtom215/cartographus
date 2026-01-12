// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"reflect"
	"testing"
)

// TestParamStructsExist verifies all parameter structs are defined correctly.
// These are simple data structs - we don't need to test Go's struct field assignment.
// Instead, we verify the struct types exist and have the expected fields.
func TestParamStructsExist(t *testing.T) {
	t.Parallel()

	// Define expected struct types and their fields
	tests := []struct {
		name           string
		structType     interface{}
		expectedFields []string
	}{
		{
			name:           "StandardTimeRangeParams",
			structType:     StandardTimeRangeParams{},
			expectedFields: []string{"TimeRange", "YAxis", "UserID", "Grouping"},
		},
		{
			name:           "HomeStatsParams",
			structType:     HomeStatsParams{},
			expectedFields: []string{"TimeRange", "StatsType", "StatsCount"},
		},
		{
			name:           "TwoIntParams",
			structType:     TwoIntParams{},
			expectedFields: []string{"Param1", "Param2"},
		},
		{
			name:           "SingleStringParam",
			structType:     SingleStringParam{},
			expectedFields: []string{"Value"},
		},
		{
			name:           "SingleIntParam",
			structType:     SingleIntParam{},
			expectedFields: []string{"Value"},
		},
		{
			name:           "ItemWatchTimeParams",
			structType:     ItemWatchTimeParams{},
			expectedFields: []string{"RatingKey", "Grouping", "QueryDays"},
		},
		{
			name:           "RecentlyAddedParams",
			structType:     RecentlyAddedParams{},
			expectedFields: []string{"Count", "Start", "MediaType", "SectionID"},
		},
		{
			name:           "TableParams",
			structType:     TableParams{},
			expectedFields: []string{"Grouping", "OrderColumn", "OrderDir", "Start", "Length", "Search"},
		},
		{
			name:           "LibraryMediaInfoParams",
			structType:     LibraryMediaInfoParams{},
			expectedFields: []string{"SectionID", "OrderColumn", "OrderDir", "Start", "Length"},
		},
		{
			name:           "ChildrenMetadataParams",
			structType:     ChildrenMetadataParams{},
			expectedFields: []string{"RatingKey", "MediaType"},
		},
		{
			name:           "StreamDataParams",
			structType:     StreamDataParams{},
			expectedFields: []string{"RowID", "SessionKey"},
		},
		{
			name:           "ExportMetadataParams",
			structType:     ExportMetadataParams{},
			expectedFields: []string{"SectionID", "ExportType", "UserID", "RatingKey", "FileFormat"},
		},
		{
			name:           "SearchParams",
			structType:     SearchParams{},
			expectedFields: []string{"Query", "Limit"},
		},
		{
			name:           "TerminateSessionParams",
			structType:     TerminateSessionParams{},
			expectedFields: []string{"SessionID", "Message"},
		},
		{
			name:           "NoParams",
			structType:     NoParams{},
			expectedFields: []string{},
		},
		{
			name:           "SyncedItemsParams",
			structType:     SyncedItemsParams{},
			expectedFields: []string{"MachineID", "UserID"},
		},
		{
			name:           "UserWatchTimeParams",
			structType:     UserWatchTimeParams{},
			expectedFields: []string{"UserID", "QueryDays"},
		},
		{
			name:           "CollectionsTableParams",
			structType:     CollectionsTableParams{},
			expectedFields: []string{"SectionID", "OrderColumn", "OrderDir", "Start", "Length", "Search"},
		},
		{
			name:           "PlaylistsTableParams",
			structType:     PlaylistsTableParams{},
			expectedFields: []string{"SectionID", "OrderColumn", "OrderDir", "Start", "Length", "Search"},
		},
		{
			name:           "ExportsTableParams",
			structType:     ExportsTableParams{},
			expectedFields: []string{"OrderColumn", "OrderDir", "Start", "Length", "Search"},
		},
		{
			name:           "LibraryWatchTimeParams",
			structType:     LibraryWatchTimeParams{},
			expectedFields: []string{"SectionID", "Grouping", "QueryDays"},
		},
		{
			name:           "UsersTableParams",
			structType:     UsersTableParams{},
			expectedFields: []string{"Grouping", "OrderColumn", "OrderDir", "Start", "Length", "Search"},
		},
		{
			name:           "UserLoginsParams",
			structType:     UserLoginsParams{},
			expectedFields: []string{"UserID", "OrderColumn", "OrderDir", "Start", "Length", "Search"},
		},
		{
			name:           "ItemUserStatsParams",
			structType:     ItemUserStatsParams{},
			expectedFields: []string{"RatingKey", "Grouping"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structType := reflect.TypeOf(tt.structType)

			// Verify each expected field exists
			for _, fieldName := range tt.expectedFields {
				if _, found := structType.FieldByName(fieldName); !found {
					t.Errorf("Struct %s missing expected field %s", tt.name, fieldName)
				}
			}

			// Verify field count matches
			if structType.NumField() != len(tt.expectedFields) {
				t.Errorf("Struct %s has %d fields, expected %d", tt.name, structType.NumField(), len(tt.expectedFields))
			}
		})
	}
}

// TestParamStructDefaultValues verifies structs have sensible zero values
func TestParamStructDefaultValues(t *testing.T) {
	t.Parallel()

	// Verify zero values don't cause issues when used
	t.Run("StandardTimeRangeParams zero value", func(t *testing.T) {
		var params StandardTimeRangeParams
		if params.TimeRange != 0 || params.YAxis != "" || params.UserID != 0 || params.Grouping != 0 {
			t.Error("Zero value should have all fields at their zero values")
		}
	})

	t.Run("HomeStatsParams zero value", func(t *testing.T) {
		var params HomeStatsParams
		if params.TimeRange != 0 || params.StatsType != "" || params.StatsCount != 0 {
			t.Error("Zero value should have all fields at their zero values")
		}
	})

	t.Run("TableParams zero value", func(t *testing.T) {
		var params TableParams
		if params.Grouping != 0 || params.OrderColumn != "" || params.Start != 0 || params.Length != 0 {
			t.Error("Zero value should have all fields at their zero values")
		}
	})

	t.Run("NoParams is empty", func(t *testing.T) {
		params := NoParams{}
		if reflect.TypeOf(params).NumField() != 0 {
			t.Error("NoParams should have no fields")
		}
	})
}

// TestParamStructFieldTypes verifies fields have the correct types for API usage
func TestParamStructFieldTypes(t *testing.T) {
	t.Parallel()

	// Test that numeric params use int (not int64, uint, etc.)
	checkIntField := func(t *testing.T, structType reflect.Type, fieldName string) {
		t.Helper()
		field, found := structType.FieldByName(fieldName)
		if !found {
			t.Errorf("Field %s not found", fieldName)
			return
		}
		if field.Type.Kind() != reflect.Int {
			t.Errorf("Field %s should be int, got %s", fieldName, field.Type.Kind())
		}
	}

	// Test that string params use string
	checkStringField := func(t *testing.T, structType reflect.Type, fieldName string) {
		t.Helper()
		field, found := structType.FieldByName(fieldName)
		if !found {
			t.Errorf("Field %s not found", fieldName)
			return
		}
		if field.Type.Kind() != reflect.String {
			t.Errorf("Field %s should be string, got %s", fieldName, field.Type.Kind())
		}
	}

	t.Run("StandardTimeRangeParams field types", func(t *testing.T) {
		structType := reflect.TypeOf(StandardTimeRangeParams{})
		checkIntField(t, structType, "TimeRange")
		checkStringField(t, structType, "YAxis")
		checkIntField(t, structType, "UserID")
		checkIntField(t, structType, "Grouping")
	})

	t.Run("TableParams field types", func(t *testing.T) {
		structType := reflect.TypeOf(TableParams{})
		checkIntField(t, structType, "Grouping")
		checkStringField(t, structType, "OrderColumn")
		checkStringField(t, structType, "OrderDir")
		checkIntField(t, structType, "Start")
		checkIntField(t, structType, "Length")
		checkStringField(t, structType, "Search")
	})

	t.Run("SearchParams field types", func(t *testing.T) {
		structType := reflect.TypeOf(SearchParams{})
		checkStringField(t, structType, "Query")
		checkIntField(t, structType, "Limit")
	})
}
