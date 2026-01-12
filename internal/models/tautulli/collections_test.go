// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliCollectionsTable_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with multiple rows", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"message": null,
				"data": {
					"recordsTotal": 100,
					"recordsFiltered": 50,
					"draw": 1,
					"data": [
						{
							"rating_key": "12345",
							"title": "Marvel Movies",
							"section_id": 1,
							"section_name": "Movies",
							"section_type": "movie",
							"media_type": "collection",
							"content_rating": "PG-13",
							"summary": "All Marvel movies",
							"thumb": "/library/metadata/12345/thumb",
							"added_at": 1609459200,
							"updated_at": 1640995200,
							"child_count": 25,
							"last_played": 1640908800,
							"plays": 150
						},
						{
							"rating_key": "12346",
							"title": "Star Wars",
							"section_id": 1,
							"section_name": "Movies",
							"section_type": "movie",
							"media_type": "collection",
							"added_at": 1609459200,
							"child_count": 11
						}
					]
				}
			}
		}`

		var collections TautulliCollectionsTable
		if err := json.Unmarshal([]byte(jsonData), &collections); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if collections.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", collections.Response.Result)
		}
		if collections.Response.Data.RecordsTotal != 100 {
			t.Errorf("Expected recordsTotal 100, got %d", collections.Response.Data.RecordsTotal)
		}
		if collections.Response.Data.RecordsFiltered != 50 {
			t.Errorf("Expected recordsFiltered 50, got %d", collections.Response.Data.RecordsFiltered)
		}
		if len(collections.Response.Data.Data) != 2 {
			t.Fatalf("Expected 2 rows, got %d", len(collections.Response.Data.Data))
		}

		// Check first row
		row := collections.Response.Data.Data[0]
		if row.RatingKey != "12345" {
			t.Errorf("Expected rating_key '12345', got %q", row.RatingKey)
		}
		if row.Title != "Marvel Movies" {
			t.Errorf("Expected title 'Marvel Movies', got %q", row.Title)
		}
		if row.ChildCount != 25 {
			t.Errorf("Expected child_count 25, got %d", row.ChildCount)
		}
		if row.Plays != 150 {
			t.Errorf("Expected plays 150, got %d", row.Plays)
		}

		// Check second row (minimal fields)
		row2 := collections.Response.Data.Data[1]
		if row2.ContentRating != "" {
			t.Errorf("Expected empty content_rating, got %q", row2.ContentRating)
		}
		if row2.Summary != "" {
			t.Errorf("Expected empty summary, got %q", row2.Summary)
		}
	})

	t.Run("empty data array", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"recordsTotal": 0,
					"recordsFiltered": 0,
					"draw": 1,
					"data": []
				}
			}
		}`

		var collections TautulliCollectionsTable
		if err := json.Unmarshal([]byte(jsonData), &collections); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(collections.Response.Data.Data) != 0 {
			t.Errorf("Expected empty data array, got %d items", len(collections.Response.Data.Data))
		}
	})

	t.Run("error response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "error",
				"message": "Library not found",
				"data": {}
			}
		}`

		var collections TautulliCollectionsTable
		if err := json.Unmarshal([]byte(jsonData), &collections); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if collections.Response.Result != "error" {
			t.Errorf("Expected result 'error', got %q", collections.Response.Result)
		}
		if collections.Response.Message == nil {
			t.Error("Expected non-nil message")
		} else if *collections.Response.Message != "Library not found" {
			t.Errorf("Expected message 'Library not found', got %q", *collections.Response.Message)
		}
	})
}

func TestTautulliCollectionsTableRow_OmitEmpty(t *testing.T) {
	row := TautulliCollectionsTableRow{
		RatingKey:   "123",
		Title:       "Test",
		SectionID:   1,
		SectionName: "Movies",
		SectionType: "movie",
		MediaType:   "collection",
		AddedAt:     1609459200,
		ChildCount:  5,
		// Omitting optional fields
	}

	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify omitempty fields are not included when empty
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// These fields should be omitted when empty
	omitEmptyFields := []string{"content_rating", "summary", "thumb", "updated_at", "last_played", "plays"}
	for _, field := range omitEmptyFields {
		if _, exists := m[field]; exists {
			t.Errorf("Field %q should be omitted when empty, but was present", field)
		}
	}

	// These fields should always be present
	requiredFields := []string{"rating_key", "title", "section_id", "section_name", "section_type", "media_type", "added_at", "child_count"}
	for _, field := range requiredFields {
		if _, exists := m[field]; !exists {
			t.Errorf("Required field %q was not present in marshaled output", field)
		}
	}
}

func TestTautulliCollectionsTable_RoundTrip(t *testing.T) {
	msg := "test"
	original := TautulliCollectionsTable{
		Response: TautulliCollectionsTableResponse{
			Result:  "success",
			Message: &msg,
			Data: TautulliCollectionsTableData{
				RecordsTotal:    10,
				RecordsFiltered: 5,
				Draw:            2,
				Data: []TautulliCollectionsTableRow{
					{
						RatingKey:     "999",
						Title:         "Test Collection",
						SectionID:     3,
						SectionName:   "TV Shows",
						SectionType:   "show",
						MediaType:     "collection",
						ContentRating: "TV-14",
						Summary:       "A test collection",
						Thumb:         "/thumb/999",
						AddedAt:       1609459200,
						UpdatedAt:     1640995200,
						ChildCount:    10,
						LastPlayed:    1640908800,
						Plays:         50,
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliCollectionsTable
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Response.Data.RecordsTotal != original.Response.Data.RecordsTotal {
		t.Error("RecordsTotal not preserved in round-trip")
	}
	if len(result.Response.Data.Data) != 1 {
		t.Fatal("Data rows not preserved in round-trip")
	}
	if result.Response.Data.Data[0].Title != "Test Collection" {
		t.Error("Title not preserved in round-trip")
	}
}
