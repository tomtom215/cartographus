// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliPlaylistsTable_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsTotal": 10,
				"recordsFiltered": 5,
				"draw": 1,
				"data": [
					{
						"rating_key": "12345",
						"title": "My Favorite Movies",
						"playlist_type": "video",
						"composite": "/library/sections/1/composite/12345",
						"summary": "A collection of my all-time favorite movies",
						"smart": 0,
						"duration": 36000,
						"added_at": 1700000000,
						"updated_at": 1700100000,
						"child_count": 25,
						"last_played": 1700200000,
						"plays": 50
					}
				]
			}
		}
	}`

	var playlists TautulliPlaylistsTable
	err := json.Unmarshal([]byte(jsonData), &playlists)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaylistsTable: %v", err)
	}

	if playlists.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", playlists.Response.Result)
	}

	data := playlists.Response.Data
	if data.RecordsTotal != 10 {
		t.Errorf("Expected RecordsTotal 10, got %d", data.RecordsTotal)
	}
	if data.RecordsFiltered != 5 {
		t.Errorf("Expected RecordsFiltered 5, got %d", data.RecordsFiltered)
	}
	if data.Draw != 1 {
		t.Errorf("Expected Draw 1, got %d", data.Draw)
	}

	if len(data.Data) != 1 {
		t.Fatalf("Expected 1 playlist, got %d", len(data.Data))
	}

	playlist := data.Data[0]
	if playlist.RatingKey != "12345" {
		t.Errorf("Expected RatingKey '12345', got '%s'", playlist.RatingKey)
	}
	if playlist.Title != "My Favorite Movies" {
		t.Errorf("Expected Title 'My Favorite Movies', got '%s'", playlist.Title)
	}
	if playlist.PlaylistType != "video" {
		t.Errorf("Expected PlaylistType 'video', got '%s'", playlist.PlaylistType)
	}
	if playlist.Composite != "/library/sections/1/composite/12345" {
		t.Errorf("Expected Composite mismatch, got '%s'", playlist.Composite)
	}
	if playlist.Summary != "A collection of my all-time favorite movies" {
		t.Errorf("Expected Summary mismatch, got '%s'", playlist.Summary)
	}
	if playlist.Smart != 0 {
		t.Errorf("Expected Smart 0, got %d", playlist.Smart)
	}
	if playlist.DurationTotal != 36000 {
		t.Errorf("Expected DurationTotal 36000, got %d", playlist.DurationTotal)
	}
	if playlist.AddedAt != 1700000000 {
		t.Errorf("Expected AddedAt 1700000000, got %d", playlist.AddedAt)
	}
	if playlist.UpdatedAt != 1700100000 {
		t.Errorf("Expected UpdatedAt 1700100000, got %d", playlist.UpdatedAt)
	}
	if playlist.ChildCount != 25 {
		t.Errorf("Expected ChildCount 25, got %d", playlist.ChildCount)
	}
	if playlist.LastPlayed != 1700200000 {
		t.Errorf("Expected LastPlayed 1700200000, got %d", playlist.LastPlayed)
	}
	if playlist.Plays != 50 {
		t.Errorf("Expected Plays 50, got %d", playlist.Plays)
	}
}

func TestTautulliPlaylistsTable_MultiplePlaylists(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsTotal": 3,
				"recordsFiltered": 3,
				"draw": 1,
				"data": [
					{
						"rating_key": "111",
						"title": "Movie Playlist",
						"playlist_type": "video",
						"smart": 0,
						"duration": 10800,
						"added_at": 1700000000,
						"child_count": 10
					},
					{
						"rating_key": "222",
						"title": "Music Playlist",
						"playlist_type": "audio",
						"smart": 1,
						"duration": 3600,
						"added_at": 1700000001,
						"child_count": 50
					},
					{
						"rating_key": "333",
						"title": "Photo Slideshow",
						"playlist_type": "photo",
						"smart": 0,
						"duration": 600,
						"added_at": 1700000002,
						"child_count": 100
					}
				]
			}
		}
	}`

	var playlists TautulliPlaylistsTable
	err := json.Unmarshal([]byte(jsonData), &playlists)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaylistsTable: %v", err)
	}

	if len(playlists.Response.Data.Data) != 3 {
		t.Fatalf("Expected 3 playlists, got %d", len(playlists.Response.Data.Data))
	}

	expectedTypes := []string{"video", "audio", "photo"}
	for i, playlist := range playlists.Response.Data.Data {
		if playlist.PlaylistType != expectedTypes[i] {
			t.Errorf("Expected PlaylistType '%s', got '%s'", expectedTypes[i], playlist.PlaylistType)
		}
	}

	// Check smart playlist
	if playlists.Response.Data.Data[1].Smart != 1 {
		t.Errorf("Expected second playlist Smart 1, got %d", playlists.Response.Data.Data[1].Smart)
	}
}

func TestTautulliPlaylistsTable_SmartPlaylist(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsTotal": 1,
				"recordsFiltered": 1,
				"draw": 1,
				"data": [
					{
						"rating_key": "smart_001",
						"title": "Recently Added Movies",
						"playlist_type": "video",
						"summary": "Automatically updates with recently added movies",
						"smart": 1,
						"duration": 7200,
						"added_at": 1700000000,
						"updated_at": 1700500000,
						"child_count": 15,
						"plays": 0
					}
				]
			}
		}
	}`

	var playlists TautulliPlaylistsTable
	err := json.Unmarshal([]byte(jsonData), &playlists)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaylistsTable: %v", err)
	}

	playlist := playlists.Response.Data.Data[0]
	if playlist.Smart != 1 {
		t.Errorf("Expected Smart 1, got %d", playlist.Smart)
	}
	if playlist.Title != "Recently Added Movies" {
		t.Errorf("Expected Title 'Recently Added Movies', got '%s'", playlist.Title)
	}
}

func TestTautulliPlaylistsTable_EmptyData(t *testing.T) {
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

	var playlists TautulliPlaylistsTable
	err := json.Unmarshal([]byte(jsonData), &playlists)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaylistsTable: %v", err)
	}

	if len(playlists.Response.Data.Data) != 0 {
		t.Errorf("Expected 0 playlists, got %d", len(playlists.Response.Data.Data))
	}
	if playlists.Response.Data.RecordsTotal != 0 {
		t.Errorf("Expected RecordsTotal 0, got %d", playlists.Response.Data.RecordsTotal)
	}
}

func TestTautulliPlaylistsTable_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Unable to fetch playlists",
			"data": {
				"recordsTotal": 0,
				"recordsFiltered": 0,
				"draw": 0,
				"data": []
			}
		}
	}`

	var playlists TautulliPlaylistsTable
	err := json.Unmarshal([]byte(jsonData), &playlists)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaylistsTable: %v", err)
	}

	if playlists.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", playlists.Response.Result)
	}
	if playlists.Response.Message == nil || *playlists.Response.Message != "Unable to fetch playlists" {
		t.Error("Expected Message 'Unable to fetch playlists'")
	}
}

func TestTautulliPlaylistsTable_JSONRoundTrip(t *testing.T) {
	original := TautulliPlaylistsTable{
		Response: TautulliPlaylistsTableResponse{
			Result: "success",
			Data: TautulliPlaylistsTableData{
				RecordsTotal:    5,
				RecordsFiltered: 3,
				Draw:            2,
				Data: []TautulliPlaylistsTableRow{
					{
						RatingKey:     "test_playlist",
						Title:         "Test Playlist",
						PlaylistType:  "video",
						Composite:     "/composite/test",
						Summary:       "Test summary",
						Smart:         0,
						DurationTotal: 7200,
						AddedAt:       1700000000,
						UpdatedAt:     1700100000,
						ChildCount:    10,
						LastPlayed:    1700200000,
						Plays:         25,
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliPlaylistsTable: %v", err)
	}

	var decoded TautulliPlaylistsTable
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaylistsTable: %v", err)
	}

	if decoded.Response.Data.RecordsTotal != original.Response.Data.RecordsTotal {
		t.Errorf("RecordsTotal mismatch")
	}
	if len(decoded.Response.Data.Data) != len(original.Response.Data.Data) {
		t.Errorf("Data length mismatch")
	}
	if decoded.Response.Data.Data[0].Title != original.Response.Data.Data[0].Title {
		t.Errorf("Title mismatch")
	}
}

func TestTautulliPlaylistsTableRow_OptionalFields(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsTotal": 1,
				"recordsFiltered": 1,
				"draw": 1,
				"data": [
					{
						"rating_key": "minimal",
						"title": "Minimal Playlist",
						"playlist_type": "audio",
						"smart": 0,
						"duration": 1800,
						"added_at": 1700000000,
						"child_count": 5
					}
				]
			}
		}
	}`

	var playlists TautulliPlaylistsTable
	err := json.Unmarshal([]byte(jsonData), &playlists)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaylistsTable: %v", err)
	}

	playlist := playlists.Response.Data.Data[0]
	// Optional fields should be zero values
	if playlist.Composite != "" {
		t.Errorf("Expected empty Composite, got '%s'", playlist.Composite)
	}
	if playlist.Summary != "" {
		t.Errorf("Expected empty Summary, got '%s'", playlist.Summary)
	}
	if playlist.UpdatedAt != 0 {
		t.Errorf("Expected UpdatedAt 0, got %d", playlist.UpdatedAt)
	}
	if playlist.LastPlayed != 0 {
		t.Errorf("Expected LastPlayed 0, got %d", playlist.LastPlayed)
	}
	if playlist.Plays != 0 {
		t.Errorf("Expected Plays 0, got %d", playlist.Plays)
	}
}

func TestTautulliPlaylistsTableRow_PlaylistTypes(t *testing.T) {
	playlistTypes := []string{"video", "audio", "photo"}

	for _, pType := range playlistTypes {
		t.Run("Type_"+pType, func(t *testing.T) {
			original := TautulliPlaylistsTable{
				Response: TautulliPlaylistsTableResponse{
					Result: "success",
					Data: TautulliPlaylistsTableData{
						RecordsTotal:    1,
						RecordsFiltered: 1,
						Draw:            1,
						Data: []TautulliPlaylistsTableRow{
							{
								RatingKey:    "type_test",
								Title:        "Type Test",
								PlaylistType: pType,
								AddedAt:      1700000000,
								ChildCount:   1,
							},
						},
					},
				},
			}

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded TautulliPlaylistsTable
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Response.Data.Data[0].PlaylistType != pType {
				t.Errorf("Expected PlaylistType '%s', got '%s'", pType, decoded.Response.Data.Data[0].PlaylistType)
			}
		})
	}
}

func TestTautulliPlaylistsTable_LongDuration(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsTotal": 1,
				"recordsFiltered": 1,
				"draw": 1,
				"data": [
					{
						"rating_key": "long_playlist",
						"title": "Complete TV Series",
						"playlist_type": "video",
						"smart": 0,
						"duration": 864000,
						"added_at": 1700000000,
						"child_count": 500
					}
				]
			}
		}
	}`

	var playlists TautulliPlaylistsTable
	err := json.Unmarshal([]byte(jsonData), &playlists)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaylistsTable: %v", err)
	}

	playlist := playlists.Response.Data.Data[0]
	// 864000 seconds = 10 days
	if playlist.DurationTotal != 864000 {
		t.Errorf("Expected DurationTotal 864000, got %d", playlist.DurationTotal)
	}
	if playlist.ChildCount != 500 {
		t.Errorf("Expected ChildCount 500, got %d", playlist.ChildCount)
	}
}

func TestTautulliPlaylistsTable_SpecialCharacters(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsTotal": 1,
				"recordsFiltered": 1,
				"draw": 1,
				"data": [
					{
						"rating_key": "special_chars",
						"title": "Movies: Action & Adventure (2024)",
						"playlist_type": "video",
						"summary": "Collection of action movies with 'special' characters & symbols <test>",
						"smart": 0,
						"duration": 7200,
						"added_at": 1700000000,
						"child_count": 20
					}
				]
			}
		}
	}`

	var playlists TautulliPlaylistsTable
	err := json.Unmarshal([]byte(jsonData), &playlists)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliPlaylistsTable: %v", err)
	}

	playlist := playlists.Response.Data.Data[0]
	if playlist.Title != "Movies: Action & Adventure (2024)" {
		t.Errorf("Title with special characters mismatch, got '%s'", playlist.Title)
	}
	if playlist.Summary != "Collection of action movies with 'special' characters & symbols <test>" {
		t.Errorf("Summary with special characters mismatch, got '%s'", playlist.Summary)
	}
}
