// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliSearch_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"results_count": 5,
				"results": [
					{
						"type": "movie",
						"rating_key": "12345",
						"title": "Inception",
						"year": 2010,
						"thumb": "/library/metadata/12345/thumb",
						"score": 0.95,
						"library": "Movies",
						"library_id": 1,
						"media_type": "movie",
						"summary": "A thief who steals corporate secrets through dream-sharing technology..."
					}
				]
			}
		}
	}`

	var search TautulliSearch
	err := json.Unmarshal([]byte(jsonData), &search)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	if search.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", search.Response.Result)
	}

	if search.Response.Data.ResultsCount != 5 {
		t.Errorf("Expected ResultsCount 5, got %d", search.Response.Data.ResultsCount)
	}

	if len(search.Response.Data.Results) != 1 {
		t.Fatalf("Expected 1 search result, got %d", len(search.Response.Data.Results))
	}

	result := search.Response.Data.Results[0]
	if result.Type != "movie" {
		t.Errorf("Expected Type 'movie', got '%s'", result.Type)
	}
	if result.RatingKey != "12345" {
		t.Errorf("Expected RatingKey '12345', got '%s'", result.RatingKey)
	}
	if result.Title != "Inception" {
		t.Errorf("Expected Title 'Inception', got '%s'", result.Title)
	}
	if result.Year != 2010 {
		t.Errorf("Expected Year 2010, got %d", result.Year)
	}
	if result.Thumb != "/library/metadata/12345/thumb" {
		t.Errorf("Expected Thumb mismatch, got '%s'", result.Thumb)
	}
	if result.Score != 0.95 {
		t.Errorf("Expected Score 0.95, got %f", result.Score)
	}
	if result.Library != "Movies" {
		t.Errorf("Expected Library 'Movies', got '%s'", result.Library)
	}
	if result.LibraryID != 1 {
		t.Errorf("Expected LibraryID 1, got %d", result.LibraryID)
	}
	if result.MediaType != "movie" {
		t.Errorf("Expected MediaType 'movie', got '%s'", result.MediaType)
	}
}

func TestTautulliSearch_TVShowEpisode(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"results_count": 1,
				"results": [
					{
						"type": "show",
						"rating_key": "99999",
						"title": "S01E01 - Pilot",
						"year": 2008,
						"thumb": "/library/metadata/99999/thumb",
						"score": 0.88,
						"library": "TV Shows",
						"library_id": 2,
						"media_type": "episode",
						"summary": "The first episode of the series...",
						"grandparent_title": "Breaking Bad",
						"parent_title": "Season 1"
					}
				]
			}
		}
	}`

	var search TautulliSearch
	err := json.Unmarshal([]byte(jsonData), &search)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	result := search.Response.Data.Results[0]
	if result.Type != "show" {
		t.Errorf("Expected Type 'show', got '%s'", result.Type)
	}
	if result.MediaType != "episode" {
		t.Errorf("Expected MediaType 'episode', got '%s'", result.MediaType)
	}
	if result.GrandparentTitle != "Breaking Bad" {
		t.Errorf("Expected GrandparentTitle 'Breaking Bad', got '%s'", result.GrandparentTitle)
	}
	if result.ParentTitle != "Season 1" {
		t.Errorf("Expected ParentTitle 'Season 1', got '%s'", result.ParentTitle)
	}
}

func TestTautulliSearch_MultipleResults(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"results_count": 3,
				"results": [
					{
						"type": "movie",
						"rating_key": "1",
						"title": "The Matrix",
						"year": 1999,
						"score": 1.0,
						"library": "Movies",
						"library_id": 1,
						"media_type": "movie"
					},
					{
						"type": "movie",
						"rating_key": "2",
						"title": "The Matrix Reloaded",
						"year": 2003,
						"score": 0.9,
						"library": "Movies",
						"library_id": 1,
						"media_type": "movie"
					},
					{
						"type": "movie",
						"rating_key": "3",
						"title": "The Matrix Revolutions",
						"year": 2003,
						"score": 0.85,
						"library": "Movies",
						"library_id": 1,
						"media_type": "movie"
					}
				]
			}
		}
	}`

	var search TautulliSearch
	err := json.Unmarshal([]byte(jsonData), &search)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	if search.Response.Data.ResultsCount != 3 {
		t.Errorf("Expected ResultsCount 3, got %d", search.Response.Data.ResultsCount)
	}

	if len(search.Response.Data.Results) != 3 {
		t.Fatalf("Expected 3 search results, got %d", len(search.Response.Data.Results))
	}

	// Verify scores are in descending order
	scores := []float64{1.0, 0.9, 0.85}
	for i, result := range search.Response.Data.Results {
		if result.Score != scores[i] {
			t.Errorf("Expected Score %f, got %f", scores[i], result.Score)
		}
	}
}

func TestTautulliSearch_EmptyResults(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"results_count": 0,
				"results": []
			}
		}
	}`

	var search TautulliSearch
	err := json.Unmarshal([]byte(jsonData), &search)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	if search.Response.Data.ResultsCount != 0 {
		t.Errorf("Expected ResultsCount 0, got %d", search.Response.Data.ResultsCount)
	}
	if len(search.Response.Data.Results) != 0 {
		t.Errorf("Expected 0 search results, got %d", len(search.Response.Data.Results))
	}
}

func TestTautulliSearch_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Search query too short",
			"data": {
				"results_count": 0,
				"results": []
			}
		}
	}`

	var search TautulliSearch
	err := json.Unmarshal([]byte(jsonData), &search)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	if search.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", search.Response.Result)
	}
	if search.Response.Message == nil || *search.Response.Message != "Search query too short" {
		t.Error("Expected Message 'Search query too short'")
	}
}

func TestTautulliSearch_JSONRoundTrip(t *testing.T) {
	original := TautulliSearch{
		Response: TautulliSearchResponse{
			Result: "success",
			Data: TautulliSearchData{
				ResultsCount: 2,
				Results: []TautulliSearchResult{
					{
						Type:      "movie",
						RatingKey: "test1",
						Title:     "Test Movie",
						Year:      2024,
						Thumb:     "/thumb/1",
						Score:     0.95,
						Library:   "Movies",
						LibraryID: 1,
						MediaType: "movie",
						Summary:   "Test summary",
					},
					{
						Type:             "show",
						RatingKey:        "test2",
						Title:            "Test Episode",
						Year:             2023,
						Thumb:            "/thumb/2",
						Score:            0.8,
						Library:          "TV Shows",
						LibraryID:        2,
						MediaType:        "episode",
						Summary:          "Episode summary",
						GrandparentTitle: "Test Show",
						ParentTitle:      "Season 1",
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliSearch: %v", err)
	}

	var decoded TautulliSearch
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	if decoded.Response.Data.ResultsCount != original.Response.Data.ResultsCount {
		t.Errorf("ResultsCount mismatch")
	}
	if len(decoded.Response.Data.Results) != len(original.Response.Data.Results) {
		t.Errorf("Results length mismatch")
	}
	if decoded.Response.Data.Results[0].Title != original.Response.Data.Results[0].Title {
		t.Errorf("Title mismatch")
	}
}

func TestTautulliSearchResult_MediaTypes(t *testing.T) {
	mediaTypes := []struct {
		resultType string
		mediaType  string
	}{
		{"movie", "movie"},
		{"show", "episode"},
		{"artist", "track"},
		{"album", "album"},
	}

	for _, mt := range mediaTypes {
		t.Run("Type_"+mt.resultType, func(t *testing.T) {
			original := TautulliSearch{
				Response: TautulliSearchResponse{
					Result: "success",
					Data: TautulliSearchData{
						ResultsCount: 1,
						Results: []TautulliSearchResult{
							{
								Type:      mt.resultType,
								RatingKey: "test",
								Title:     "Test",
								MediaType: mt.mediaType,
								Library:   "Test Library",
								LibraryID: 1,
							},
						},
					},
				},
			}

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded TautulliSearch
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Response.Data.Results[0].Type != mt.resultType {
				t.Errorf("Expected Type '%s', got '%s'", mt.resultType, decoded.Response.Data.Results[0].Type)
			}
			if decoded.Response.Data.Results[0].MediaType != mt.mediaType {
				t.Errorf("Expected MediaType '%s', got '%s'", mt.mediaType, decoded.Response.Data.Results[0].MediaType)
			}
		})
	}
}

func TestTautulliSearchResult_ScoreRange(t *testing.T) {
	scores := []float64{0.0, 0.25, 0.5, 0.75, 1.0}

	for _, score := range scores {
		t.Run("Score_"+string(rune(int(score*100))), func(t *testing.T) {
			original := TautulliSearch{
				Response: TautulliSearchResponse{
					Result: "success",
					Data: TautulliSearchData{
						ResultsCount: 1,
						Results: []TautulliSearchResult{
							{
								Type:      "movie",
								RatingKey: "test",
								Title:     "Test",
								Score:     score,
								Library:   "Movies",
								LibraryID: 1,
								MediaType: "movie",
							},
						},
					},
				},
			}

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded TautulliSearch
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Response.Data.Results[0].Score != score {
				t.Errorf("Expected Score %f, got %f", score, decoded.Response.Data.Results[0].Score)
			}
		})
	}
}

func TestTautulliSearchResult_SpecialCharacters(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"results_count": 2,
				"results": [
					{
						"type": "movie",
						"rating_key": "1",
						"title": "Movie: The Sequel (Part 2) - Director's Cut",
						"year": 2024,
						"score": 0.9,
						"library": "Movies",
						"library_id": 1,
						"media_type": "movie",
						"summary": "A movie with 'quotes' & \"special\" <characters>"
					},
					{
						"type": "artist",
						"rating_key": "2",
						"title": "Artist's \"Best\" Album",
						"year": 2023,
						"score": 0.8,
						"library": "Music",
						"library_id": 3,
						"media_type": "album"
					}
				]
			}
		}
	}`

	var search TautulliSearch
	err := json.Unmarshal([]byte(jsonData), &search)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	if search.Response.Data.Results[0].Title != "Movie: The Sequel (Part 2) - Director's Cut" {
		t.Errorf("Title mismatch, got '%s'", search.Response.Data.Results[0].Title)
	}
	if search.Response.Data.Results[0].Summary != "A movie with 'quotes' & \"special\" <characters>" {
		t.Errorf("Summary mismatch, got '%s'", search.Response.Data.Results[0].Summary)
	}
	if search.Response.Data.Results[1].Title != "Artist's \"Best\" Album" {
		t.Errorf("Title mismatch, got '%s'", search.Response.Data.Results[1].Title)
	}
}

func TestTautulliSearchResult_OptionalFields(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"results_count": 1,
				"results": [
					{
						"type": "movie",
						"rating_key": "minimal",
						"title": "Minimal Movie",
						"year": 2024,
						"score": 0.5,
						"library": "Movies",
						"library_id": 1,
						"media_type": "movie"
					}
				]
			}
		}
	}`

	var search TautulliSearch
	err := json.Unmarshal([]byte(jsonData), &search)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	result := search.Response.Data.Results[0]
	// Optional fields should be empty strings
	if result.Thumb != "" {
		t.Errorf("Expected empty Thumb, got '%s'", result.Thumb)
	}
	if result.Summary != "" {
		t.Errorf("Expected empty Summary, got '%s'", result.Summary)
	}
	if result.GrandparentTitle != "" {
		t.Errorf("Expected empty GrandparentTitle, got '%s'", result.GrandparentTitle)
	}
	if result.ParentTitle != "" {
		t.Errorf("Expected empty ParentTitle, got '%s'", result.ParentTitle)
	}
}

func TestTautulliSearch_NullMessage(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"message": null,
			"data": {
				"results_count": 0,
				"results": []
			}
		}
	}`

	var search TautulliSearch
	err := json.Unmarshal([]byte(jsonData), &search)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	if search.Response.Message != nil {
		t.Errorf("Expected Message nil, got '%s'", *search.Response.Message)
	}
}

func TestTautulliSearch_MusicResults(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"results_count": 3,
				"results": [
					{
						"type": "artist",
						"rating_key": "artist1",
						"title": "The Beatles",
						"score": 1.0,
						"library": "Music",
						"library_id": 3,
						"media_type": "artist"
					},
					{
						"type": "album",
						"rating_key": "album1",
						"title": "Abbey Road",
						"year": 1969,
						"score": 0.95,
						"library": "Music",
						"library_id": 3,
						"media_type": "album",
						"grandparent_title": "The Beatles"
					},
					{
						"type": "track",
						"rating_key": "track1",
						"title": "Come Together",
						"year": 1969,
						"score": 0.9,
						"library": "Music",
						"library_id": 3,
						"media_type": "track",
						"grandparent_title": "The Beatles",
						"parent_title": "Abbey Road"
					}
				]
			}
		}
	}`

	var search TautulliSearch
	err := json.Unmarshal([]byte(jsonData), &search)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	expectedTypes := []string{"artist", "album", "track"}
	for i, result := range search.Response.Data.Results {
		if result.Type != expectedTypes[i] {
			t.Errorf("Expected Type '%s', got '%s'", expectedTypes[i], result.Type)
		}
		if result.MediaType != expectedTypes[i] {
			t.Errorf("Expected MediaType '%s', got '%s'", expectedTypes[i], result.MediaType)
		}
	}

	// Check hierarchy for track
	track := search.Response.Data.Results[2]
	if track.GrandparentTitle != "The Beatles" {
		t.Errorf("Expected GrandparentTitle 'The Beatles', got '%s'", track.GrandparentTitle)
	}
	if track.ParentTitle != "Abbey Road" {
		t.Errorf("Expected ParentTitle 'Abbey Road', got '%s'", track.ParentTitle)
	}
}

func TestTautulliSearch_LargeResultsCount(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"results_count": 9999,
				"results": []
			}
		}
	}`

	var search TautulliSearch
	err := json.Unmarshal([]byte(jsonData), &search)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSearch: %v", err)
	}

	if search.Response.Data.ResultsCount != 9999 {
		t.Errorf("Expected ResultsCount 9999, got %d", search.Response.Data.ResultsCount)
	}
}
