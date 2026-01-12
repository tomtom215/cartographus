// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliMetadata_JSONUnmarshal(t *testing.T) {
	t.Run("complete movie metadata", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"rating_key": "12345",
					"parent_rating_key": "",
					"grandparent_rating_key": "",
					"title": "Inception",
					"parent_title": "",
					"grandparent_title": "",
					"original_title": "Inception (Original)",
					"sort_title": "Inception",
					"media_index": 0,
					"parent_media_index": 0,
					"studio": "Warner Bros.",
					"content_rating": "PG-13",
					"summary": "A thief who steals corporate secrets through dream-sharing technology.",
					"tagline": "Your mind is the scene of the crime.",
					"rating": 8.8,
					"rating_image": "rottentomatoes://image.rating.ripe",
					"audience_rating": 9.1,
					"audience_rating_image": "rottentomatoes://image.rating.upright",
					"user_rating": 10.0,
					"duration": 8880000,
					"year": 2010,
					"thumb": "/library/metadata/12345/thumb",
					"parent_thumb": "",
					"grandparent_thumb": "",
					"art": "/library/metadata/12345/art",
					"banner": "/library/metadata/12345/banner",
					"originally_available_at": "2010-07-16",
					"added_at": 1609459200,
					"updated_at": 1640995200,
					"last_viewed_at": 1640908800,
					"guid": "plex://movie/12345",
					"guids": ["imdb://tt1375666", "tmdb://27205"],
					"directors": ["Christopher Nolan"],
					"writers": ["Christopher Nolan"],
					"actors": ["Leonardo DiCaprio", "Joseph Gordon-Levitt", "Ellen Page"],
					"genres": ["Action", "Science Fiction", "Thriller"],
					"labels": ["favorites"],
					"collections": ["Nolan Collection"],
					"media_info": [
						{
							"id": 1,
							"container": "mkv",
							"bitrate": 10000,
							"height": 2160,
							"width": 3840,
							"aspect_ratio": 2.39,
							"video_codec": "hevc",
							"video_resolution": "4k",
							"video_framerate": "23.976",
							"video_bit_depth": 10,
							"video_profile": "main 10",
							"audio_codec": "truehd",
							"audio_channels": 8,
							"audio_channel_layout": "7.1",
							"audio_bitrate": 1536,
							"optimized_version": 0
						}
					]
				}
			}
		}`

		var metadata TautulliMetadata
		if err := json.Unmarshal([]byte(jsonData), &metadata); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if metadata.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", metadata.Response.Result)
		}

		data := metadata.Response.Data
		if data.RatingKey != "12345" {
			t.Errorf("Expected rating_key '12345', got %q", data.RatingKey)
		}
		if data.Title != "Inception" {
			t.Errorf("Expected title 'Inception', got %q", data.Title)
		}
		if data.Studio != "Warner Bros." {
			t.Errorf("Expected studio 'Warner Bros.', got %q", data.Studio)
		}
		if data.ContentRating != "PG-13" {
			t.Errorf("Expected content_rating 'PG-13', got %q", data.ContentRating)
		}
		if data.Rating != 8.8 {
			t.Errorf("Expected rating 8.8, got %f", data.Rating)
		}
		if data.AudienceRating != 9.1 {
			t.Errorf("Expected audience_rating 9.1, got %f", data.AudienceRating)
		}
		if data.UserRating != 10.0 {
			t.Errorf("Expected user_rating 10.0, got %f", data.UserRating)
		}
		if data.Duration != 8880000 {
			t.Errorf("Expected duration 8880000, got %d", data.Duration)
		}
		if data.Year != 2010 {
			t.Errorf("Expected year 2010, got %d", data.Year)
		}
		if data.OriginallyAvailableAt != "2010-07-16" {
			t.Errorf("Expected originally_available_at '2010-07-16', got %q", data.OriginallyAvailableAt)
		}

		// Check arrays
		if len(data.Guids) != 2 {
			t.Errorf("Expected 2 guids, got %d", len(data.Guids))
		}
		if len(data.Directors) != 1 || data.Directors[0] != "Christopher Nolan" {
			t.Errorf("Directors not preserved correctly")
		}
		if len(data.Actors) != 3 {
			t.Errorf("Expected 3 actors, got %d", len(data.Actors))
		}
		if len(data.Genres) != 3 {
			t.Errorf("Expected 3 genres, got %d", len(data.Genres))
		}

		// Check media info
		if len(data.MediaInfo) != 1 {
			t.Fatalf("Expected 1 media info, got %d", len(data.MediaInfo))
		}
		media := data.MediaInfo[0]
		if media.Container != "mkv" {
			t.Errorf("Expected container 'mkv', got %q", media.Container)
		}
		if media.Width != 3840 {
			t.Errorf("Expected width 3840, got %d", media.Width)
		}
		if media.Height != 2160 {
			t.Errorf("Expected height 2160, got %d", media.Height)
		}
		if media.VideoCodec != "hevc" {
			t.Errorf("Expected video_codec 'hevc', got %q", media.VideoCodec)
		}
		if media.AudioChannels != 8 {
			t.Errorf("Expected audio_channels 8, got %d", media.AudioChannels)
		}
	})

	t.Run("episode metadata", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"rating_key": "67890",
					"parent_rating_key": "6789",
					"grandparent_rating_key": "678",
					"title": "Pilot",
					"parent_title": "Season 1",
					"grandparent_title": "Breaking Bad",
					"original_title": "",
					"sort_title": "Pilot",
					"media_index": 1,
					"parent_media_index": 1,
					"studio": "AMC",
					"content_rating": "TV-MA",
					"summary": "Walter White, a chemistry teacher, is diagnosed with cancer.",
					"tagline": "",
					"rating": 9.0,
					"rating_image": "",
					"audience_rating": 0.0,
					"audience_rating_image": "",
					"user_rating": 0.0,
					"duration": 3480000,
					"year": 2008,
					"thumb": "/library/metadata/67890/thumb",
					"parent_thumb": "/library/metadata/6789/thumb",
					"grandparent_thumb": "/library/metadata/678/thumb",
					"art": "",
					"banner": "",
					"originally_available_at": "2008-01-20",
					"added_at": 1609459200,
					"updated_at": 1640995200,
					"last_viewed_at": 0,
					"guid": "plex://episode/67890",
					"guids": [],
					"directors": ["Vince Gilligan"],
					"writers": ["Vince Gilligan"],
					"actors": [],
					"genres": [],
					"labels": [],
					"collections": [],
					"media_info": []
				}
			}
		}`

		var metadata TautulliMetadata
		if err := json.Unmarshal([]byte(jsonData), &metadata); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		data := metadata.Response.Data
		if data.ParentRatingKey != "6789" {
			t.Errorf("Expected parent_rating_key '6789', got %q", data.ParentRatingKey)
		}
		if data.GrandparentRatingKey != "678" {
			t.Errorf("Expected grandparent_rating_key '678', got %q", data.GrandparentRatingKey)
		}
		if data.ParentTitle != "Season 1" {
			t.Errorf("Expected parent_title 'Season 1', got %q", data.ParentTitle)
		}
		if data.GrandparentTitle != "Breaking Bad" {
			t.Errorf("Expected grandparent_title 'Breaking Bad', got %q", data.GrandparentTitle)
		}
		if data.MediaIndex != 1 {
			t.Errorf("Expected media_index 1, got %d", data.MediaIndex)
		}
		if data.ParentMediaIndex != 1 {
			t.Errorf("Expected parent_media_index 1, got %d", data.ParentMediaIndex)
		}
	})

	t.Run("error response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "error",
				"message": "Item not found",
				"data": {}
			}
		}`

		var metadata TautulliMetadata
		if err := json.Unmarshal([]byte(jsonData), &metadata); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if metadata.Response.Result != "error" {
			t.Errorf("Expected result 'error', got %q", metadata.Response.Result)
		}
		if metadata.Response.Message == nil || *metadata.Response.Message != "Item not found" {
			t.Error("Expected error message 'Item not found'")
		}
	})
}

func TestTautulliChildrenMetadata_JSONUnmarshal(t *testing.T) {
	t.Run("show with seasons", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"children_count": 5,
					"children_list": [
						{
							"media_type": "season",
							"section_id": 2,
							"library_name": "TV Shows",
							"rating_key": "1001",
							"parent_rating_key": "100",
							"title": "Season 1",
							"parent_title": "Breaking Bad",
							"media_index": 1,
							"year": 2008,
							"thumb": "/library/metadata/1001/thumb",
							"parent_thumb": "/library/metadata/100/thumb",
							"added_at": 1609459200,
							"updated_at": 1640995200,
							"guid": "plex://season/1001",
							"parent_guid": "plex://show/100"
						},
						{
							"media_type": "season",
							"section_id": 2,
							"library_name": "TV Shows",
							"rating_key": "1002",
							"parent_rating_key": "100",
							"title": "Season 2",
							"parent_title": "Breaking Bad",
							"media_index": 2,
							"year": 2009,
							"thumb": "/library/metadata/1002/thumb",
							"parent_thumb": "/library/metadata/100/thumb",
							"added_at": 1609459200,
							"updated_at": 1640995200,
							"guid": "plex://season/1002",
							"parent_guid": "plex://show/100"
						}
					]
				}
			}
		}`

		var children TautulliChildrenMetadata
		if err := json.Unmarshal([]byte(jsonData), &children); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		data := children.Response.Data
		if data.ChildrenCount != 5 {
			t.Errorf("Expected children_count 5, got %d", data.ChildrenCount)
		}
		if len(data.ChildrenList) != 2 {
			t.Fatalf("Expected 2 children, got %d", len(data.ChildrenList))
		}

		season1 := data.ChildrenList[0]
		if season1.MediaType != "season" {
			t.Errorf("Expected media_type 'season', got %q", season1.MediaType)
		}
		if season1.Title != "Season 1" {
			t.Errorf("Expected title 'Season 1', got %q", season1.Title)
		}
		if season1.ParentTitle != "Breaking Bad" {
			t.Errorf("Expected parent_title 'Breaking Bad', got %q", season1.ParentTitle)
		}
		if season1.MediaIndex != 1 {
			t.Errorf("Expected media_index 1, got %d", season1.MediaIndex)
		}
		if season1.LibraryName != "TV Shows" {
			t.Errorf("Expected library_name 'TV Shows', got %q", season1.LibraryName)
		}
	})

	t.Run("season with episodes", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"children_count": 7,
					"children_list": [
						{
							"media_type": "episode",
							"section_id": 2,
							"rating_key": "10001",
							"parent_rating_key": "1001",
							"grandparent_rating_key": "100",
							"title": "Pilot",
							"parent_title": "Season 1",
							"grandparent_title": "Breaking Bad",
							"original_title": "",
							"sort_title": "Pilot",
							"media_index": 1,
							"parent_media_index": 1,
							"year": 2008,
							"thumb": "/library/metadata/10001/thumb",
							"parent_thumb": "/library/metadata/1001/thumb",
							"grandparent_thumb": "/library/metadata/100/thumb",
							"added_at": 1609459200,
							"updated_at": 1640995200,
							"last_viewed_at": 1640908800,
							"guid": "plex://episode/10001",
							"parent_guid": "plex://season/1001",
							"grandparent_guid": "plex://show/100"
						}
					]
				}
			}
		}`

		var children TautulliChildrenMetadata
		if err := json.Unmarshal([]byte(jsonData), &children); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		episode := children.Response.Data.ChildrenList[0]
		if episode.MediaType != "episode" {
			t.Errorf("Expected media_type 'episode', got %q", episode.MediaType)
		}
		if episode.GrandparentRatingKey != "100" {
			t.Errorf("Expected grandparent_rating_key '100', got %q", episode.GrandparentRatingKey)
		}
		if episode.GrandparentTitle != "Breaking Bad" {
			t.Errorf("Expected grandparent_title 'Breaking Bad', got %q", episode.GrandparentTitle)
		}
		if episode.ParentMediaIndex != 1 {
			t.Errorf("Expected parent_media_index 1, got %d", episode.ParentMediaIndex)
		}
		if episode.LastViewedAt != 1640908800 {
			t.Errorf("Expected last_viewed_at 1640908800, got %d", episode.LastViewedAt)
		}
	})

	t.Run("empty children", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"children_count": 0,
					"children_list": []
				}
			}
		}`

		var children TautulliChildrenMetadata
		if err := json.Unmarshal([]byte(jsonData), &children); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if children.Response.Data.ChildrenCount != 0 {
			t.Errorf("Expected children_count 0, got %d", children.Response.Data.ChildrenCount)
		}
		if len(children.Response.Data.ChildrenList) != 0 {
			t.Errorf("Expected empty children_list, got %d", len(children.Response.Data.ChildrenList))
		}
	})
}

func TestTautulliMediaInfo_RoundTrip(t *testing.T) {
	original := TautulliMediaInfo{
		ID:                 1,
		Container:          "mkv",
		Bitrate:            10000,
		Height:             2160,
		Width:              3840,
		AspectRatio:        2.39,
		VideoCodec:         "hevc",
		VideoResolution:    "4k",
		VideoFramerate:     "23.976",
		VideoBitDepth:      10,
		VideoProfile:       "main 10",
		AudioCodec:         "truehd",
		AudioChannels:      8,
		AudioChannelLayout: "7.1",
		AudioBitrate:       1536,
		OptimizedVersion:   0,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliMediaInfo
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Container != original.Container {
		t.Error("Container not preserved")
	}
	if result.Width != original.Width {
		t.Error("Width not preserved")
	}
	if result.Height != original.Height {
		t.Error("Height not preserved")
	}
	if result.AspectRatio != original.AspectRatio {
		t.Error("AspectRatio not preserved")
	}
	if result.VideoCodec != original.VideoCodec {
		t.Error("VideoCodec not preserved")
	}
	if result.AudioChannels != original.AudioChannels {
		t.Error("AudioChannels not preserved")
	}
}

func TestTautulliMetadataData_SpecialCharacters(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"rating_key": "12345",
				"parent_rating_key": "",
				"grandparent_rating_key": "",
				"title": "Movie: Part 1 - The \"Beginning\" & More",
				"parent_title": "",
				"grandparent_title": "",
				"original_title": "Titre Fran\u00e7ais",
				"sort_title": "Movie Part 1",
				"media_index": 0,
				"parent_media_index": 0,
				"studio": "O'Brien & Associates",
				"content_rating": "R",
				"summary": "A film about <special> characters & \"quotes\".",
				"tagline": "Coming soon\u2122",
				"rating": 0.0,
				"rating_image": "",
				"audience_rating": 0.0,
				"audience_rating_image": "",
				"user_rating": 0.0,
				"duration": 0,
				"year": 2020,
				"thumb": "",
				"parent_thumb": "",
				"grandparent_thumb": "",
				"art": "",
				"banner": "",
				"originally_available_at": "",
				"added_at": 0,
				"updated_at": 0,
				"last_viewed_at": 0,
				"guid": "",
				"guids": [],
				"directors": ["John O'Connor"],
				"writers": ["Mary-Jane \"MJ\" Smith"],
				"actors": [],
				"genres": ["Action & Adventure"],
				"labels": [],
				"collections": [],
				"media_info": []
			}
		}
	}`

	var metadata TautulliMetadata
	if err := json.Unmarshal([]byte(jsonData), &metadata); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	data := metadata.Response.Data
	if data.Title != "Movie: Part 1 - The \"Beginning\" & More" {
		t.Errorf("Title with special chars not preserved: %q", data.Title)
	}
	if data.Studio != "O'Brien & Associates" {
		t.Errorf("Studio with special chars not preserved: %q", data.Studio)
	}
	if len(data.Directors) != 1 || data.Directors[0] != "John O'Connor" {
		t.Errorf("Directors with special chars not preserved")
	}
	if len(data.Genres) != 1 || data.Genres[0] != "Action & Adventure" {
		t.Errorf("Genres with special chars not preserved")
	}
}

func TestTautulliMetadataData_ZeroRatings(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"rating_key": "12345",
				"parent_rating_key": "",
				"grandparent_rating_key": "",
				"title": "Unrated Movie",
				"parent_title": "",
				"grandparent_title": "",
				"original_title": "",
				"sort_title": "Unrated Movie",
				"media_index": 0,
				"parent_media_index": 0,
				"studio": "",
				"content_rating": "",
				"summary": "",
				"tagline": "",
				"rating": 0.0,
				"rating_image": "",
				"audience_rating": 0.0,
				"audience_rating_image": "",
				"user_rating": 0.0,
				"duration": 0,
				"year": 0,
				"thumb": "",
				"parent_thumb": "",
				"grandparent_thumb": "",
				"art": "",
				"banner": "",
				"originally_available_at": "",
				"added_at": 0,
				"updated_at": 0,
				"last_viewed_at": 0,
				"guid": "",
				"guids": [],
				"directors": [],
				"writers": [],
				"actors": [],
				"genres": [],
				"labels": [],
				"collections": [],
				"media_info": []
			}
		}
	}`

	var metadata TautulliMetadata
	if err := json.Unmarshal([]byte(jsonData), &metadata); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	data := metadata.Response.Data
	if data.Rating != 0.0 {
		t.Errorf("Expected rating 0.0, got %f", data.Rating)
	}
	if data.AudienceRating != 0.0 {
		t.Errorf("Expected audience_rating 0.0, got %f", data.AudienceRating)
	}
	if data.UserRating != 0.0 {
		t.Errorf("Expected user_rating 0.0, got %f", data.UserRating)
	}
	if data.Year != 0 {
		t.Errorf("Expected year 0, got %d", data.Year)
	}
	if data.Duration != 0 {
		t.Errorf("Expected duration 0, got %d", data.Duration)
	}
}

func TestTautulliMetadataData_MultipleMediaInfo(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"rating_key": "12345",
				"parent_rating_key": "",
				"grandparent_rating_key": "",
				"title": "Multi-Version Movie",
				"parent_title": "",
				"grandparent_title": "",
				"original_title": "",
				"sort_title": "Multi-Version Movie",
				"media_index": 0,
				"parent_media_index": 0,
				"studio": "",
				"content_rating": "",
				"summary": "",
				"tagline": "",
				"rating": 0.0,
				"rating_image": "",
				"audience_rating": 0.0,
				"audience_rating_image": "",
				"user_rating": 0.0,
				"duration": 7200000,
				"year": 2020,
				"thumb": "",
				"parent_thumb": "",
				"grandparent_thumb": "",
				"art": "",
				"banner": "",
				"originally_available_at": "",
				"added_at": 0,
				"updated_at": 0,
				"last_viewed_at": 0,
				"guid": "",
				"guids": [],
				"directors": [],
				"writers": [],
				"actors": [],
				"genres": [],
				"labels": [],
				"collections": [],
				"media_info": [
					{
						"id": 1,
						"container": "mkv",
						"bitrate": 50000,
						"height": 2160,
						"width": 3840,
						"aspect_ratio": 2.39,
						"video_codec": "hevc",
						"video_resolution": "4k",
						"video_framerate": "23.976",
						"video_bit_depth": 10,
						"video_profile": "main 10",
						"audio_codec": "truehd",
						"audio_channels": 8,
						"audio_channel_layout": "7.1",
						"audio_bitrate": 3000,
						"optimized_version": 0
					},
					{
						"id": 2,
						"container": "mp4",
						"bitrate": 8000,
						"height": 1080,
						"width": 1920,
						"aspect_ratio": 1.78,
						"video_codec": "h264",
						"video_resolution": "1080",
						"video_framerate": "23.976",
						"video_bit_depth": 8,
						"video_profile": "high",
						"audio_codec": "aac",
						"audio_channels": 2,
						"audio_channel_layout": "stereo",
						"audio_bitrate": 192,
						"optimized_version": 1
					}
				]
			}
		}
	}`

	var metadata TautulliMetadata
	if err := json.Unmarshal([]byte(jsonData), &metadata); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(metadata.Response.Data.MediaInfo) != 2 {
		t.Fatalf("Expected 2 media versions, got %d", len(metadata.Response.Data.MediaInfo))
	}

	uhd := metadata.Response.Data.MediaInfo[0]
	if uhd.VideoResolution != "4k" {
		t.Errorf("Expected UHD video_resolution '4k', got %q", uhd.VideoResolution)
	}
	if uhd.OptimizedVersion != 0 {
		t.Errorf("Expected UHD optimized_version 0, got %d", uhd.OptimizedVersion)
	}

	optimized := metadata.Response.Data.MediaInfo[1]
	if optimized.VideoResolution != "1080" {
		t.Errorf("Expected optimized video_resolution '1080', got %q", optimized.VideoResolution)
	}
	if optimized.OptimizedVersion != 1 {
		t.Errorf("Expected optimized optimized_version 1, got %d", optimized.OptimizedVersion)
	}
}
