// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliLibraryUserStats_JSONUnmarshal(t *testing.T) {
	t.Run("multiple users", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": [
					{
						"user_id": 1,
						"username": "admin",
						"friendly_name": "Admin User",
						"user_thumb": "/thumb/admin.jpg",
						"total_plays": 500,
						"total_time": 1800000
					},
					{
						"user_id": 2,
						"username": "guest",
						"friendly_name": "Guest User",
						"user_thumb": "",
						"total_plays": 50,
						"total_time": 180000
					}
				]
			}
		}`

		var stats TautulliLibraryUserStats
		if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if stats.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", stats.Response.Result)
		}
		if len(stats.Response.Data) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(stats.Response.Data))
		}

		user1 := stats.Response.Data[0]
		if user1.UserID != 1 {
			t.Errorf("Expected user_id 1, got %d", user1.UserID)
		}
		if user1.TotalPlays != 500 {
			t.Errorf("Expected total_plays 500, got %d", user1.TotalPlays)
		}
		if user1.TotalTime != 1800000 {
			t.Errorf("Expected total_time 1800000, got %d", user1.TotalTime)
		}
	})

	t.Run("empty stats", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": []
			}
		}`

		var stats TautulliLibraryUserStats
		if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(stats.Response.Data) != 0 {
			t.Errorf("Expected empty data, got %d items", len(stats.Response.Data))
		}
	})
}

func TestTautulliLibraries_JSONUnmarshal(t *testing.T) {
	t.Run("multiple libraries", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": [
					{
						"section_id": 1,
						"section_name": "Movies",
						"section_type": "movie",
						"count": 500,
						"parent_count": 0,
						"child_count": 500,
						"is_active": 1,
						"thumb": "/thumb/movies.jpg",
						"art": "/art/movies.jpg"
					},
					{
						"section_id": 2,
						"section_name": "TV Shows",
						"section_type": "show",
						"count": 100,
						"parent_count": 200,
						"child_count": 2000,
						"is_active": 1,
						"thumb": "/thumb/tv.jpg",
						"art": "/art/tv.jpg"
					},
					{
						"section_id": 3,
						"section_name": "Music",
						"section_type": "artist",
						"count": 50,
						"parent_count": 100,
						"child_count": 1000,
						"is_active": 0,
						"thumb": "",
						"art": ""
					}
				]
			}
		}`

		var libs TautulliLibraries
		if err := json.Unmarshal([]byte(jsonData), &libs); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(libs.Response.Data) != 3 {
			t.Fatalf("Expected 3 libraries, got %d", len(libs.Response.Data))
		}

		lib1 := libs.Response.Data[0]
		if lib1.SectionID != 1 {
			t.Errorf("Expected section_id 1, got %d", lib1.SectionID)
		}
		if lib1.SectionName != "Movies" {
			t.Errorf("Expected section_name 'Movies', got %q", lib1.SectionName)
		}
		if lib1.SectionType != "movie" {
			t.Errorf("Expected section_type 'movie', got %q", lib1.SectionType)
		}
		if lib1.Count != 500 {
			t.Errorf("Expected count 500, got %d", lib1.Count)
		}
		if lib1.IsActive != 1 {
			t.Errorf("Expected is_active 1, got %d", lib1.IsActive)
		}

		lib3 := libs.Response.Data[2]
		if lib3.IsActive != 0 {
			t.Errorf("Expected is_active 0, got %d", lib3.IsActive)
		}
	})
}

func TestTautulliLibrary_JSONUnmarshal(t *testing.T) {
	t.Run("complete library data", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"section_id": 1,
					"section_name": "Movies",
					"section_type": "movie",
					"agent": "tv.plex.agents.movie",
					"thumb": "/thumb/movies.jpg",
					"art": "/art/movies.jpg",
					"count": 500,
					"parent_count": 0,
					"child_count": 500,
					"is_active": 1,
					"do_notify_created": 1,
					"keep_history": 1,
					"deleted_section": 0,
					"last_accessed": 1640995200
				}
			}
		}`

		var lib TautulliLibrary
		if err := json.Unmarshal([]byte(jsonData), &lib); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		data := lib.Response.Data
		if data.SectionID != 1 {
			t.Errorf("Expected section_id 1, got %d", data.SectionID)
		}
		if data.SectionName != "Movies" {
			t.Errorf("Expected section_name 'Movies', got %q", data.SectionName)
		}
		if data.Agent != "tv.plex.agents.movie" {
			t.Errorf("Expected agent 'tv.plex.agents.movie', got %q", data.Agent)
		}
		if data.DoNotifyCreated != 1 {
			t.Errorf("Expected do_notify_created 1, got %d", data.DoNotifyCreated)
		}
		if data.KeepHistory != 1 {
			t.Errorf("Expected keep_history 1, got %d", data.KeepHistory)
		}
		if data.DeletedSection != 0 {
			t.Errorf("Expected deleted_section 0, got %d", data.DeletedSection)
		}
		if data.LastAccessed != 1640995200 {
			t.Errorf("Expected last_accessed 1640995200, got %d", data.LastAccessed)
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

		var lib TautulliLibrary
		if err := json.Unmarshal([]byte(jsonData), &lib); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if lib.Response.Result != "error" {
			t.Errorf("Expected result 'error', got %q", lib.Response.Result)
		}
		if lib.Response.Message == nil || *lib.Response.Message != "Library not found" {
			t.Error("Expected error message 'Library not found'")
		}
	})
}

func TestTautulliLibrariesTable_JSONUnmarshal(t *testing.T) {
	t.Run("complete table response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"recordsFiltered": 3,
					"recordsTotal": 3,
					"draw": 1,
					"data": [
						{
							"section_id": 1,
							"section_name": "Movies",
							"section_type": "movie",
							"agent": "tv.plex.agents.movie",
							"thumb": "/thumb/movies.jpg",
							"art": "/art/movies.jpg",
							"count": 500,
							"parent_count": 0,
							"child_count": 500,
							"is_active": 1,
							"do_notify": 1,
							"do_notify_created": 1,
							"keep_history": 1,
							"deleted_section": 0,
							"plays": 1000,
							"duration": 3600000,
							"last_accessed": 1640995200,
							"last_played": "Inception"
						},
						{
							"section_id": 2,
							"section_name": "TV Shows",
							"section_type": "show",
							"count": 100,
							"plays": 5000,
							"duration": 18000000
						}
					]
				}
			}
		}`

		var table TautulliLibrariesTable
		if err := json.Unmarshal([]byte(jsonData), &table); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if table.Response.Data.RecordsTotal != 3 {
			t.Errorf("Expected recordsTotal 3, got %d", table.Response.Data.RecordsTotal)
		}
		if len(table.Response.Data.Data) != 2 {
			t.Fatalf("Expected 2 rows, got %d", len(table.Response.Data.Data))
		}

		row1 := table.Response.Data.Data[0]
		if row1.Plays != 1000 {
			t.Errorf("Expected plays 1000, got %d", row1.Plays)
		}
		if row1.Duration != 3600000 {
			t.Errorf("Expected duration 3600000, got %d", row1.Duration)
		}
		if row1.LastPlayed != "Inception" {
			t.Errorf("Expected last_played 'Inception', got %q", row1.LastPlayed)
		}
	})
}

func TestTautulliLibraryMediaInfo_JSONUnmarshal(t *testing.T) {
	t.Run("multiple media items", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"recordsFiltered": 100,
					"recordsTotal": 500,
					"draw": 1,
					"data": [
						{
							"section_id": 1,
							"section_type": "movie",
							"added_at": 1609459200,
							"media_type": "movie",
							"rating_key": "12345",
							"title": "Inception",
							"year": 2010,
							"thumb": "/thumb/inception.jpg",
							"container": "mkv",
							"bitrate": 10000,
							"video_codec": "hevc",
							"video_resolution": "4k",
							"video_framerate": "23.976",
							"audio_codec": "truehd",
							"audio_channels": "7.1",
							"file_size": 45000000000,
							"last_played": 1640995200,
							"play_count": 5
						},
						{
							"section_id": 2,
							"section_type": "show",
							"added_at": 1609459200,
							"media_type": "episode",
							"rating_key": "67890",
							"parent_rating_key": "6789",
							"grandparent_rating_key": "678",
							"title": "Pilot",
							"media_index": 1,
							"parent_media_index": 1,
							"container": "mp4",
							"video_codec": "h264",
							"video_resolution": "1080",
							"play_count": 0
						}
					]
				}
			}
		}`

		var info TautulliLibraryMediaInfo
		if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if info.Response.Data.RecordsTotal != 500 {
			t.Errorf("Expected recordsTotal 500, got %d", info.Response.Data.RecordsTotal)
		}
		if len(info.Response.Data.Data) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(info.Response.Data.Data))
		}

		movie := info.Response.Data.Data[0]
		if movie.Title != "Inception" {
			t.Errorf("Expected title 'Inception', got %q", movie.Title)
		}
		if movie.Year != 2010 {
			t.Errorf("Expected year 2010, got %d", movie.Year)
		}
		if movie.VideoCodec != "hevc" {
			t.Errorf("Expected video_codec 'hevc', got %q", movie.VideoCodec)
		}
		if movie.VideoResolution != "4k" {
			t.Errorf("Expected video_resolution '4k', got %q", movie.VideoResolution)
		}
		if movie.FileSize != 45000000000 {
			t.Errorf("Expected file_size 45000000000, got %d", movie.FileSize)
		}
		if movie.PlayCount != 5 {
			t.Errorf("Expected play_count 5, got %d", movie.PlayCount)
		}

		episode := info.Response.Data.Data[1]
		if episode.MediaIndex != 1 {
			t.Errorf("Expected media_index 1, got %d", episode.MediaIndex)
		}
		if episode.ParentMediaIndex != 1 {
			t.Errorf("Expected parent_media_index 1, got %d", episode.ParentMediaIndex)
		}
		if episode.ParentRatingKey != "6789" {
			t.Errorf("Expected parent_rating_key '6789', got %q", episode.ParentRatingKey)
		}
	})
}

func TestTautulliLibraryWatchTimeStats_JSONUnmarshal(t *testing.T) {
	t.Run("multiple time periods", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": [
					{
						"query_days": 1,
						"total_time": 7200,
						"total_plays": 3,
						"section_id": 1,
						"section_name": "Movies",
						"section_type": "movie"
					},
					{
						"query_days": 7,
						"total_time": 36000,
						"total_plays": 15,
						"section_id": 1,
						"section_name": "Movies",
						"section_type": "movie"
					},
					{
						"query_days": 30,
						"total_time": 180000,
						"total_plays": 60,
						"section_id": 1,
						"section_name": "Movies",
						"section_type": "movie"
					}
				]
			}
		}`

		var stats TautulliLibraryWatchTimeStats
		if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(stats.Response.Data) != 3 {
			t.Fatalf("Expected 3 stats, got %d", len(stats.Response.Data))
		}

		stat1 := stats.Response.Data[0]
		if stat1.QueryDays != 1 {
			t.Errorf("Expected query_days 1, got %d", stat1.QueryDays)
		}
		if stat1.TotalTime != 7200 {
			t.Errorf("Expected total_time 7200, got %d", stat1.TotalTime)
		}
		if stat1.TotalPlays != 3 {
			t.Errorf("Expected total_plays 3, got %d", stat1.TotalPlays)
		}

		stat3 := stats.Response.Data[2]
		if stat3.QueryDays != 30 {
			t.Errorf("Expected query_days 30, got %d", stat3.QueryDays)
		}
	})
}

func TestTautulliLibraryNames_JSONUnmarshal(t *testing.T) {
	t.Run("multiple libraries", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": [
					{
						"section_id": 1,
						"section_name": "Movies",
						"section_type": "movie"
					},
					{
						"section_id": 2,
						"section_name": "TV Shows",
						"section_type": "show"
					},
					{
						"section_id": 3,
						"section_name": "Music"
					}
				]
			}
		}`

		var names TautulliLibraryNames
		if err := json.Unmarshal([]byte(jsonData), &names); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(names.Response.Data) != 3 {
			t.Fatalf("Expected 3 libraries, got %d", len(names.Response.Data))
		}

		lib1 := names.Response.Data[0]
		if lib1.SectionID != 1 {
			t.Errorf("Expected section_id 1, got %d", lib1.SectionID)
		}
		if lib1.SectionName != "Movies" {
			t.Errorf("Expected section_name 'Movies', got %q", lib1.SectionName)
		}
		if lib1.SectionType != "movie" {
			t.Errorf("Expected section_type 'movie', got %q", lib1.SectionType)
		}

		// Third library has no section_type
		lib3 := names.Response.Data[2]
		if lib3.SectionType != "" {
			t.Errorf("Expected empty section_type, got %q", lib3.SectionType)
		}
	})
}

func TestTautulliLibraryData_RoundTrip(t *testing.T) {
	original := TautulliLibraryData{
		SectionID:       99,
		SectionName:     "Test Library",
		SectionType:     "movie",
		Agent:           "tv.plex.agents.movie",
		Thumb:           "/thumb/test.jpg",
		Art:             "/art/test.jpg",
		Count:           1000,
		ParentCount:     0,
		ChildCount:      1000,
		IsActive:        1,
		DoNotifyCreated: 1,
		KeepHistory:     1,
		DeletedSection:  0,
		LastAccessed:    1640995200,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliLibraryData
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.SectionID != original.SectionID {
		t.Error("SectionID not preserved")
	}
	if result.SectionName != original.SectionName {
		t.Error("SectionName not preserved")
	}
	if result.Count != original.Count {
		t.Error("Count not preserved")
	}
	if result.LastAccessed != original.LastAccessed {
		t.Error("LastAccessed not preserved")
	}
}

func TestTautulliLibraryDetail_SpecialCharacters(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": [
				{
					"section_id": 1,
					"section_name": "Movies & TV (All Genres)",
					"section_type": "movie",
					"count": 100,
					"parent_count": 0,
					"child_count": 100,
					"is_active": 1,
					"thumb": "/thumb/test's-image.jpg",
					"art": "/art/test\"image.jpg"
				}
			]
		}
	}`

	var libs TautulliLibraries
	if err := json.Unmarshal([]byte(jsonData), &libs); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	lib := libs.Response.Data[0]
	if lib.SectionName != "Movies & TV (All Genres)" {
		t.Errorf("SectionName with special chars not preserved: %q", lib.SectionName)
	}
	if lib.Thumb != "/thumb/test's-image.jpg" {
		t.Errorf("Thumb with apostrophe not preserved: %q", lib.Thumb)
	}
}

func TestTautulliLibraryMediaInfoRow_ZeroValues(t *testing.T) {
	row := TautulliLibraryMediaInfoRow{
		SectionID:   1,
		SectionType: "movie",
		AddedAt:     0,
		MediaType:   "movie",
		RatingKey:   "12345",
		Title:       "Test",
		Year:        0,
		Bitrate:     0,
		FileSize:    0,
		PlayCount:   0,
	}

	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliLibraryMediaInfoRow
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Year != 0 {
		t.Errorf("Expected year 0, got %d", result.Year)
	}
	if result.Bitrate != 0 {
		t.Errorf("Expected bitrate 0, got %d", result.Bitrate)
	}
	if result.FileSize != 0 {
		t.Errorf("Expected file_size 0, got %d", result.FileSize)
	}
	if result.PlayCount != 0 {
		t.Errorf("Expected play_count 0, got %d", result.PlayCount)
	}
}

func TestTautulliLibraryMediaInfoRow_LargeValues(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsFiltered": 1,
				"recordsTotal": 1,
				"data": [
					{
						"section_id": 1,
						"section_type": "movie",
						"added_at": 9223372036854775807,
						"media_type": "movie",
						"rating_key": "12345",
						"title": "Large File",
						"bitrate": 100000,
						"file_size": 107374182400,
						"last_played": 9223372036854775807,
						"play_count": 2147483647
					}
				]
			}
		}
	}`

	var info TautulliLibraryMediaInfo
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	row := info.Response.Data.Data[0]
	if row.FileSize != 107374182400 {
		t.Errorf("Expected file_size 107374182400 (100GB), got %d", row.FileSize)
	}
	if row.PlayCount != 2147483647 {
		t.Errorf("Expected play_count MaxInt32, got %d", row.PlayCount)
	}
}
