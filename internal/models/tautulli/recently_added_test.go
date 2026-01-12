// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliRecentlyAdded_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"records_total": 100,
				"recently_added": [
					{
						"rating_key": "12345",
						"parent_rating_key": "11111",
						"grandparent_rating_key": "10000",
						"title": "S01E01 - Pilot",
						"parent_title": "Season 1",
						"grandparent_title": "Breaking Bad",
						"media_type": "episode",
						"year": 2008,
						"thumb": "/library/metadata/12345/thumb",
						"parent_thumb": "/library/metadata/11111/thumb",
						"grandparent_thumb": "/library/metadata/10000/thumb",
						"added_at": 1700000000,
						"library_name": "TV Shows",
						"section_id": 2
					}
				]
			}
		}
	}`

	var recent TautulliRecentlyAdded
	err := json.Unmarshal([]byte(jsonData), &recent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	if recent.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", recent.Response.Result)
	}

	if recent.Response.Data.RecordsTotal != 100 {
		t.Errorf("Expected RecordsTotal 100, got %d", recent.Response.Data.RecordsTotal)
	}

	if len(recent.Response.Data.RecentlyAdded) != 1 {
		t.Fatalf("Expected 1 recently added item, got %d", len(recent.Response.Data.RecentlyAdded))
	}

	item := recent.Response.Data.RecentlyAdded[0]
	if item.RatingKey != "12345" {
		t.Errorf("Expected RatingKey '12345', got '%s'", item.RatingKey)
	}
	if item.ParentRatingKey != "11111" {
		t.Errorf("Expected ParentRatingKey '11111', got '%s'", item.ParentRatingKey)
	}
	if item.GrandparentRatingKey != "10000" {
		t.Errorf("Expected GrandparentRatingKey '10000', got '%s'", item.GrandparentRatingKey)
	}
	if item.Title != "S01E01 - Pilot" {
		t.Errorf("Expected Title 'S01E01 - Pilot', got '%s'", item.Title)
	}
	if item.ParentTitle != "Season 1" {
		t.Errorf("Expected ParentTitle 'Season 1', got '%s'", item.ParentTitle)
	}
	if item.GrandparentTitle != "Breaking Bad" {
		t.Errorf("Expected GrandparentTitle 'Breaking Bad', got '%s'", item.GrandparentTitle)
	}
	if item.MediaType != "episode" {
		t.Errorf("Expected MediaType 'episode', got '%s'", item.MediaType)
	}
	if item.Year != 2008 {
		t.Errorf("Expected Year 2008, got %d", item.Year)
	}
	if item.Thumb != "/library/metadata/12345/thumb" {
		t.Errorf("Expected Thumb mismatch, got '%s'", item.Thumb)
	}
	if item.ParentThumb != "/library/metadata/11111/thumb" {
		t.Errorf("Expected ParentThumb mismatch, got '%s'", item.ParentThumb)
	}
	if item.GrandparentThumb != "/library/metadata/10000/thumb" {
		t.Errorf("Expected GrandparentThumb mismatch, got '%s'", item.GrandparentThumb)
	}
	if item.AddedAt != 1700000000 {
		t.Errorf("Expected AddedAt 1700000000, got %d", item.AddedAt)
	}
	if item.LibraryName != "TV Shows" {
		t.Errorf("Expected LibraryName 'TV Shows', got '%s'", item.LibraryName)
	}
	if item.SectionID != 2 {
		t.Errorf("Expected SectionID 2, got %d", item.SectionID)
	}
}

func TestTautulliRecentlyAdded_Movie(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"records_total": 50,
				"recently_added": [
					{
						"rating_key": "99999",
						"parent_rating_key": "",
						"grandparent_rating_key": "",
						"title": "Inception",
						"parent_title": "",
						"grandparent_title": "",
						"media_type": "movie",
						"year": 2010,
						"thumb": "/library/metadata/99999/thumb",
						"parent_thumb": "",
						"grandparent_thumb": "",
						"added_at": 1700000000,
						"library_name": "Movies",
						"section_id": 1
					}
				]
			}
		}
	}`

	var recent TautulliRecentlyAdded
	err := json.Unmarshal([]byte(jsonData), &recent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	item := recent.Response.Data.RecentlyAdded[0]
	if item.MediaType != "movie" {
		t.Errorf("Expected MediaType 'movie', got '%s'", item.MediaType)
	}
	if item.Title != "Inception" {
		t.Errorf("Expected Title 'Inception', got '%s'", item.Title)
	}
	if item.ParentRatingKey != "" {
		t.Errorf("Expected empty ParentRatingKey, got '%s'", item.ParentRatingKey)
	}
	if item.GrandparentRatingKey != "" {
		t.Errorf("Expected empty GrandparentRatingKey, got '%s'", item.GrandparentRatingKey)
	}
}

func TestTautulliRecentlyAdded_MultipleItems(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"records_total": 3,
				"recently_added": [
					{
						"rating_key": "1",
						"title": "Movie 1",
						"parent_title": "",
						"grandparent_title": "",
						"media_type": "movie",
						"year": 2024,
						"thumb": "/thumb/1",
						"added_at": 1700300000,
						"library_name": "Movies",
						"section_id": 1
					},
					{
						"rating_key": "2",
						"title": "Episode 1",
						"parent_title": "Season 1",
						"grandparent_title": "Show A",
						"media_type": "episode",
						"year": 2023,
						"thumb": "/thumb/2",
						"added_at": 1700200000,
						"library_name": "TV Shows",
						"section_id": 2
					},
					{
						"rating_key": "3",
						"title": "Track 1",
						"parent_title": "Album 1",
						"grandparent_title": "Artist A",
						"media_type": "track",
						"year": 2022,
						"thumb": "/thumb/3",
						"added_at": 1700100000,
						"library_name": "Music",
						"section_id": 3
					}
				]
			}
		}
	}`

	var recent TautulliRecentlyAdded
	err := json.Unmarshal([]byte(jsonData), &recent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	if len(recent.Response.Data.RecentlyAdded) != 3 {
		t.Fatalf("Expected 3 recently added items, got %d", len(recent.Response.Data.RecentlyAdded))
	}

	expectedTypes := []string{"movie", "episode", "track"}
	for i, item := range recent.Response.Data.RecentlyAdded {
		if item.MediaType != expectedTypes[i] {
			t.Errorf("Expected MediaType '%s', got '%s'", expectedTypes[i], item.MediaType)
		}
	}
}

func TestTautulliRecentlyAdded_EmptyData(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"records_total": 0,
				"recently_added": []
			}
		}
	}`

	var recent TautulliRecentlyAdded
	err := json.Unmarshal([]byte(jsonData), &recent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	if recent.Response.Data.RecordsTotal != 0 {
		t.Errorf("Expected RecordsTotal 0, got %d", recent.Response.Data.RecordsTotal)
	}
	if len(recent.Response.Data.RecentlyAdded) != 0 {
		t.Errorf("Expected 0 recently added items, got %d", len(recent.Response.Data.RecentlyAdded))
	}
}

func TestTautulliRecentlyAdded_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Unable to fetch recently added",
			"data": {
				"records_total": 0,
				"recently_added": []
			}
		}
	}`

	var recent TautulliRecentlyAdded
	err := json.Unmarshal([]byte(jsonData), &recent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	if recent.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", recent.Response.Result)
	}
	if recent.Response.Message == nil || *recent.Response.Message != "Unable to fetch recently added" {
		t.Error("Expected Message 'Unable to fetch recently added'")
	}
}

func TestTautulliRecentlyAdded_JSONRoundTrip(t *testing.T) {
	original := TautulliRecentlyAdded{
		Response: TautulliRecentlyAddedResponse{
			Result: "success",
			Data: TautulliRecentlyAddedData{
				RecordsTotal: 10,
				RecentlyAdded: []TautulliRecentlyAddedItem{
					{
						RatingKey:            "test_key",
						ParentRatingKey:      "parent_key",
						GrandparentRatingKey: "grandparent_key",
						Title:                "Test Episode",
						ParentTitle:          "Test Season",
						GrandparentTitle:     "Test Show",
						MediaType:            "episode",
						Year:                 2024,
						Thumb:                "/thumb",
						ParentThumb:          "/parent_thumb",
						GrandparentThumb:     "/grandparent_thumb",
						AddedAt:              1700000000,
						LibraryName:          "TV Shows",
						SectionID:            2,
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliRecentlyAdded: %v", err)
	}

	var decoded TautulliRecentlyAdded
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	if decoded.Response.Data.RecordsTotal != original.Response.Data.RecordsTotal {
		t.Errorf("RecordsTotal mismatch")
	}
	if len(decoded.Response.Data.RecentlyAdded) != len(original.Response.Data.RecentlyAdded) {
		t.Errorf("RecentlyAdded length mismatch")
	}
	if decoded.Response.Data.RecentlyAdded[0].Title != original.Response.Data.RecentlyAdded[0].Title {
		t.Errorf("Title mismatch")
	}
}

func TestTautulliRecentlyAddedItem_MediaTypes(t *testing.T) {
	mediaTypes := []string{"movie", "show", "season", "episode", "artist", "album", "track"}

	for _, mType := range mediaTypes {
		t.Run("MediaType_"+mType, func(t *testing.T) {
			original := TautulliRecentlyAdded{
				Response: TautulliRecentlyAddedResponse{
					Result: "success",
					Data: TautulliRecentlyAddedData{
						RecordsTotal: 1,
						RecentlyAdded: []TautulliRecentlyAddedItem{
							{
								RatingKey:   "test",
								Title:       "Test",
								MediaType:   mType,
								AddedAt:     1700000000,
								LibraryName: "Test Library",
								SectionID:   1,
							},
						},
					},
				},
			}

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded TautulliRecentlyAdded
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Response.Data.RecentlyAdded[0].MediaType != mType {
				t.Errorf("Expected MediaType '%s', got '%s'", mType, decoded.Response.Data.RecentlyAdded[0].MediaType)
			}
		})
	}
}

func TestTautulliRecentlyAddedItem_AllLibraries(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"records_total": 4,
				"recently_added": [
					{
						"rating_key": "1",
						"title": "Movie",
						"media_type": "movie",
						"year": 2024,
						"added_at": 1700000000,
						"library_name": "Movies",
						"section_id": 1
					},
					{
						"rating_key": "2",
						"title": "Episode",
						"media_type": "episode",
						"year": 2024,
						"added_at": 1700000001,
						"library_name": "TV Shows",
						"section_id": 2
					},
					{
						"rating_key": "3",
						"title": "Track",
						"media_type": "track",
						"year": 2024,
						"added_at": 1700000002,
						"library_name": "Music",
						"section_id": 3
					},
					{
						"rating_key": "4",
						"title": "Photo",
						"media_type": "photo",
						"year": 2024,
						"added_at": 1700000003,
						"library_name": "Photos",
						"section_id": 4
					}
				]
			}
		}
	}`

	var recent TautulliRecentlyAdded
	err := json.Unmarshal([]byte(jsonData), &recent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	expectedLibraries := []string{"Movies", "TV Shows", "Music", "Photos"}
	for i, item := range recent.Response.Data.RecentlyAdded {
		if item.LibraryName != expectedLibraries[i] {
			t.Errorf("Expected LibraryName '%s', got '%s'", expectedLibraries[i], item.LibraryName)
		}
		if item.SectionID != i+1 {
			t.Errorf("Expected SectionID %d, got %d", i+1, item.SectionID)
		}
	}
}

func TestTautulliRecentlyAddedItem_SpecialCharacters(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"records_total": 1,
				"recently_added": [
					{
						"rating_key": "special",
						"title": "Movie: The Sequel (Part 2) - Director's Cut",
						"parent_title": "Season 1 & 2",
						"grandparent_title": "Show with 'Quotes' <and> symbols",
						"media_type": "episode",
						"year": 2024,
						"added_at": 1700000000,
						"library_name": "TV Shows",
						"section_id": 1
					}
				]
			}
		}
	}`

	var recent TautulliRecentlyAdded
	err := json.Unmarshal([]byte(jsonData), &recent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	item := recent.Response.Data.RecentlyAdded[0]
	if item.Title != "Movie: The Sequel (Part 2) - Director's Cut" {
		t.Errorf("Title mismatch, got '%s'", item.Title)
	}
	if item.ParentTitle != "Season 1 & 2" {
		t.Errorf("ParentTitle mismatch, got '%s'", item.ParentTitle)
	}
	if item.GrandparentTitle != "Show with 'Quotes' <and> symbols" {
		t.Errorf("GrandparentTitle mismatch, got '%s'", item.GrandparentTitle)
	}
}

func TestTautulliRecentlyAdded_LargeRecordsTotal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"records_total": 999999,
				"recently_added": []
			}
		}
	}`

	var recent TautulliRecentlyAdded
	err := json.Unmarshal([]byte(jsonData), &recent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	if recent.Response.Data.RecordsTotal != 999999 {
		t.Errorf("Expected RecordsTotal 999999, got %d", recent.Response.Data.RecordsTotal)
	}
}

func TestTautulliRecentlyAdded_NullMessage(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"message": null,
			"data": {
				"records_total": 0,
				"recently_added": []
			}
		}
	}`

	var recent TautulliRecentlyAdded
	err := json.Unmarshal([]byte(jsonData), &recent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	if recent.Response.Message != nil {
		t.Errorf("Expected Message nil, got '%s'", *recent.Response.Message)
	}
}

func TestTautulliRecentlyAddedItem_OldContent(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"records_total": 1,
				"recently_added": [
					{
						"rating_key": "old_movie",
						"title": "Classic Film",
						"media_type": "movie",
						"year": 1950,
						"added_at": 1700000000,
						"library_name": "Classics",
						"section_id": 5
					}
				]
			}
		}
	}`

	var recent TautulliRecentlyAdded
	err := json.Unmarshal([]byte(jsonData), &recent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliRecentlyAdded: %v", err)
	}

	item := recent.Response.Data.RecentlyAdded[0]
	if item.Year != 1950 {
		t.Errorf("Expected Year 1950, got %d", item.Year)
	}
}
