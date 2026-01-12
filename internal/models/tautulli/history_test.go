// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// TEST HELPERS
// =============================================================================

// assertIntPtrEqual checks that an int pointer has the expected value.
func assertIntPtrEqual(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil || *got != want {
		t.Errorf("Expected %s %d, got %v", name, want, got)
	}
}

// assertInt64PtrEqual checks that an int64 pointer has the expected value.
func assertInt64PtrEqual(t *testing.T, name string, got *int64, want int64) {
	t.Helper()
	if got == nil || *got != want {
		t.Errorf("Expected %s %d, got %v", name, want, got)
	}
}

// assertFloat64PtrEqual checks that a float64 pointer has the expected value.
func assertFloat64PtrEqual(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil || *got != want {
		t.Errorf("Expected %s %f, got %v", name, want, got)
	}
}

// assertStringPtrEqual checks that a string pointer has the expected value.
func assertStringPtrEqual(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Errorf("Expected %s %q, got %v", name, want, got)
	}
}

// assertStringEqual checks that a string has the expected value.
func assertStringEqual(t *testing.T, name string, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %s %q, got %q", name, want, got)
	}
}

// assertIntPtrNil checks that an int pointer is nil.
func assertIntPtrNil(t *testing.T, name string, got *int) {
	t.Helper()
	if got != nil {
		t.Errorf("Expected %s nil, got %v", name, got)
	}
}

// assertInt64PtrNil checks that an int64 pointer is nil.
func assertInt64PtrNil(t *testing.T, name string, got *int64) {
	t.Helper()
	if got != nil {
		t.Errorf("Expected %s nil, got %v", name, got)
	}
}

// assertStringPtrNil checks that a string pointer is nil.
func assertStringPtrNil(t *testing.T, name string, got *string) {
	t.Helper()
	if got != nil {
		t.Errorf("Expected %s nil, got %v", name, got)
	}
}

// assertIntPtrNotNilZero checks that an int pointer is not nil and contains zero.
func assertIntPtrNotNilZero(t *testing.T, name string, got *int) {
	t.Helper()
	if got == nil {
		t.Errorf("%s should not be nil for zero value", name)
	} else if *got != 0 {
		t.Errorf("Expected %s 0, got %d", name, *got)
	}
}

// assertInt64PtrNotNilZero checks that an int64 pointer is not nil and contains zero.
func assertInt64PtrNotNilZero(t *testing.T, name string, got *int64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s should not be nil for zero value", name)
	} else if *got != 0 {
		t.Errorf("Expected %s 0, got %d", name, *got)
	}
}

// assertFloat64PtrNotNilZero checks that a float64 pointer is not nil and contains zero.
func assertFloat64PtrNotNilZero(t *testing.T, name string, got *float64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s should not be nil for zero value", name)
	} else if *got != 0.0 {
		t.Errorf("Expected %s 0.0, got %f", name, *got)
	}
}

// assertStringPtrNotNilEmpty checks that a string pointer is not nil and contains empty string.
func assertStringPtrNotNilEmpty(t *testing.T, name string, got *string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s should not be nil for empty string", name)
	}
}

func TestTautulliHistory_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with all fields", func(t *testing.T) {
		sessionKey := "abc123"
		groupCount := 3
		state := "stopped"
		userID := 42
		parentTitle := "Season 1"
		grandparentTitle := "Breaking Bad"
		percentComplete := 95
		pausedCounter := 120
		duration := 3600
		sectionID := 1
		year := 2020
		ratingKey := 12345
		parentRatingKey := 1234
		grandparentRatingKey := 123
		originalTitle := "Original Movie Title"
		originallyAvailableAt := "2020-01-15"
		watchedStatus := 0.85
		rowID := 99999
		optVersion := 1
		syncedVersion := 0
		transcodeThrottled := 0
		transcodeProgress := 50
		transcodeHWRequested := 1
		transcodeHWDecoding := 1
		transcodeHWEncoding := 1
		transcodeHWFullPipeline := 0
		transcodeVideoWidth := 1920
		transcodeVideoHeight := 1080
		transcodeAudioChannels := 6
		streamBitrate := 8000000
		streamVideoBitrate := 6000000
		streamVideoBitDepth := 10
		streamVideoWidth := 1920
		streamVideoHeight := 1080
		streamAudioBitrate := 640000
		streamAudioSampleRate := 48000
		streamSubtitleForced := 0
		audioBitrate := 768000
		audioSampleRate := 48000
		videoBitrate := 10000000
		videoBitDepth := 10
		videoRefFrames := 4
		videoWidth := 3840
		videoHeight := 2160
		subtitleForced := 0
		subtitles := 1
		secure := 1
		relayed := 0
		relay := 0
		local := 1
		bandwidth := 25000
		fileSize := int64(8589934592)
		bitrate := 10000000
		isAdmin := 0
		isHomeUser := 1
		isAllowSync := 1
		isRestricted := 0
		keepHistory := 1
		deletedUser := 0
		doNotify := 1
		allowGuest := 1
		live := 0
		channelStream := 0
		throttled := 0
		mediaIndex := 5
		parentMediaIndex := 2
		referenceID := 88888

		jsonData := `{
			"response": {
				"result": "success",
				"message": null,
				"data": {
					"recordsFiltered": 100,
					"recordsTotal": 500,
					"data": [{
						"session_key": "abc123",
						"date": 1609459200,
						"started": 1609456800,
						"stopped": 1609460400,
						"group_count": 3,
						"group_ids": "1,2,3",
						"state": "stopped",
						"user_id": 42,
						"user": "testuser",
						"friendly_name": "Test User",
						"user_thumb": "/thumb/user.jpg",
						"email": "test@example.com",
						"ip_address": "192.168.1.100",
						"ip_address_public": "203.0.113.50",
						"is_admin": 0,
						"is_home_user": 1,
						"is_allow_sync": 1,
						"is_restricted": 0,
						"keep_history": 1,
						"deleted_user": 0,
						"do_notify": 1,
						"allow_guest": 1,
						"shared_libraries": "1;2;3",
						"media_type": "episode",
						"title": "Pilot",
						"parent_title": "Season 1",
						"grandparent_title": "Breaking Bad",
						"sort_title": "Pilot",
						"platform": "Roku",
						"platform_name": "Roku Ultra",
						"platform_version": "10.5",
						"player": "Roku",
						"product": "Plex for Roku",
						"product_version": "7.0.0",
						"device": "Roku Ultra",
						"machine_id": "ROKU123",
						"location": "lan",
						"quality_profile": "1080p",
						"optimized_version": 1,
						"synced_version": 0,
						"percent_complete": 95,
						"paused_counter": 120,
						"duration": 3600,
						"throttled": 0,
						"live": 0,
						"live_uuid": "",
						"channel_stream": 0,
						"channel_call_sign": "",
						"channel_identifier": "",
						"transcode_decision": "transcode",
						"video_decision": "transcode",
						"audio_decision": "copy",
						"subtitle_decision": "burn",
						"transcode_key": "TK123",
						"transcode_throttled": 0,
						"transcode_progress": 50,
						"transcode_speed": "2.5",
						"transcode_hw_requested": 1,
						"transcode_hw_decoding": 1,
						"transcode_hw_encoding": 1,
						"transcode_hw_full_pipeline": 0,
						"transcode_hw_decode": "hevc",
						"transcode_hw_decode_title": "Intel Quick Sync",
						"transcode_hw_encode": "h264",
						"transcode_hw_encode_title": "NVIDIA NVENC",
						"transcode_container": "mkv",
						"transcode_video_codec": "h264",
						"transcode_video_width": 1920,
						"transcode_video_height": 1080,
						"transcode_audio_codec": "aac",
						"transcode_audio_channels": 6,
						"video_resolution": "4k",
						"video_full_resolution": "2160p",
						"video_codec": "hevc",
						"video_codec_level": "5.1",
						"video_profile": "main 10",
						"video_scan_type": "progressive",
						"video_language": "English",
						"video_language_code": "eng",
						"aspect_ratio": "16:9",
						"video_dynamic_range": "HDR10",
						"video_color_primaries": "bt2020",
						"video_color_range": "tv",
						"video_color_space": "bt2020nc",
						"video_color_trc": "smpte2084",
						"video_chroma_subsampling": "4:2:0",
						"audio_codec": "truehd",
						"audio_profile": "Dolby TrueHD",
						"audio_language": "English",
						"audio_language_code": "eng",
						"section_id": 1,
						"library_name": "TV Shows",
						"content_rating": "TV-MA",
						"year": 2020,
						"studio": "AMC",
						"added_at": "1609459200",
						"updated_at": "1640995200",
						"last_viewed_at": "1640908800",
						"summary": "A chemistry teacher diagnosed with cancer.",
						"tagline": "Change is coming",
						"rating": "9.5",
						"audience_rating": "9.7",
						"user_rating": "10.0",
						"labels": "favorites,watched",
						"collections": "Breaking Bad Collection",
						"banner": "/banner.jpg",
						"rating_key": 12345,
						"parent_rating_key": 1234,
						"grandparent_rating_key": 123,
						"media_index": 5,
						"parent_media_index": 2,
						"guid": "plex://episode/12345",
						"original_title": "Original Movie Title",
						"full_title": "Breaking Bad - S01E01 - Pilot",
						"originally_available_at": "2020-01-15",
						"watched_status": 0.85,
						"row_id": 99999,
						"reference_id": 88888,
						"thumb": "/thumb/episode.jpg",
						"directors": "Vince Gilligan",
						"writers": "Vince Gilligan",
						"actors": "Bryan Cranston,Aaron Paul",
						"genres": "Drama,Crime",
						"stream_container": "mkv",
						"stream_bitrate": 8000000,
						"stream_video_codec": "h264",
						"stream_video_codec_level": "4.1",
						"stream_video_resolution": "1080",
						"stream_video_full_resolution": "1080p",
						"stream_video_decision": "transcode",
						"stream_video_bitrate": 6000000,
						"stream_video_bit_depth": 10,
						"stream_video_width": 1920,
						"stream_video_height": 1080,
						"stream_video_framerate": "23.976",
						"stream_video_profile": "high",
						"stream_video_scan_type": "progressive",
						"stream_video_language": "English",
						"stream_video_language_code": "eng",
						"stream_video_dynamic_range": "SDR",
						"stream_aspect_ratio": "16:9",
						"stream_audio_codec": "aac",
						"stream_audio_channels": "6",
						"stream_audio_channel_layout": "5.1",
						"stream_audio_bitrate": 640000,
						"stream_audio_bitrate_mode": "cbr",
						"stream_audio_sample_rate": 48000,
						"stream_audio_language": "English",
						"stream_audio_language_code": "eng",
						"stream_audio_profile": "lc",
						"stream_audio_decision": "copy",
						"stream_subtitle_codec": "srt",
						"stream_subtitle_container": "srt",
						"stream_subtitle_format": "srt",
						"stream_subtitle_forced": 0,
						"stream_subtitle_location": "external",
						"stream_subtitle_language": "English",
						"stream_subtitle_language_code": "eng",
						"stream_subtitle_decision": "burn",
						"subtitle_container": "srt",
						"audio_channels": "6",
						"audio_channel_layout": "5.1(side)",
						"audio_bitrate": 768000,
						"audio_bitrate_mode": "cbr",
						"audio_sample_rate": 48000,
						"video_framerate": "23.976",
						"video_bitrate": 10000000,
						"video_bit_depth": 10,
						"video_ref_frames": 4,
						"video_width": 3840,
						"video_height": 2160,
						"container": "mkv",
						"subtitle_codec": "srt",
						"subtitle_container_fmt": "srt",
						"subtitle_format": "srt",
						"subtitle_language": "English",
						"subtitle_language_code": "eng",
						"subtitle_forced": 0,
						"subtitle_location": "external",
						"subtitles": 1,
						"secure": 1,
						"relayed": 0,
						"relay": 0,
						"local": 1,
						"bandwidth": 25000,
						"file_size": 8589934592,
						"bitrate": 10000000,
						"file": "/media/tv/breaking_bad/s01e01.mkv",
						"parent_thumb": "/thumb/season.jpg",
						"grandparent_thumb": "/thumb/show.jpg",
						"art": "/art/episode.jpg",
						"grandparent_art": "/art/show.jpg",
						"parent_guid": "plex://season/1234",
						"grandparent_guid": "plex://show/123"
					}]
				}
			}
		}`

		var history TautulliHistory
		if err := json.Unmarshal([]byte(jsonData), &history); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if history.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", history.Response.Result)
		}
		if history.Response.Data.RecordsFiltered != 100 {
			t.Errorf("Expected recordsFiltered 100, got %d", history.Response.Data.RecordsFiltered)
		}
		if history.Response.Data.RecordsTotal != 500 {
			t.Errorf("Expected recordsTotal 500, got %d", history.Response.Data.RecordsTotal)
		}
		if len(history.Response.Data.Data) != 1 {
			t.Fatalf("Expected 1 record, got %d", len(history.Response.Data.Data))
		}

		record := history.Response.Data.Data[0]

		// Verify string pointer fields
		assertStringPtrEqual(t, "session_key", record.SessionKey, sessionKey)
		assertStringPtrEqual(t, "state", record.State, state)
		assertStringPtrEqual(t, "parent_title", record.ParentTitle, parentTitle)
		assertStringPtrEqual(t, "grandparent_title", record.GrandparentTitle, grandparentTitle)
		assertStringPtrEqual(t, "original_title", record.OriginalTitle, originalTitle)
		assertStringPtrEqual(t, "originally_available_at", record.OriginallyAvailableAt, originallyAvailableAt)

		// Verify int pointer fields
		assertIntPtrEqual(t, "group_count", record.GroupCount, groupCount)
		assertIntPtrEqual(t, "user_id", record.UserID, userID)
		assertIntPtrEqual(t, "percent_complete", record.PercentComplete, percentComplete)
		assertIntPtrEqual(t, "paused_counter", record.PausedCounter, pausedCounter)
		assertIntPtrEqual(t, "duration", record.Duration, duration)
		assertIntPtrEqual(t, "section_id", record.SectionID, sectionID)
		assertIntPtrEqual(t, "year", record.Year, year)
		assertIntPtrEqual(t, "rating_key", record.RatingKey, ratingKey)
		assertIntPtrEqual(t, "parent_rating_key", record.ParentRatingKey, parentRatingKey)
		assertIntPtrEqual(t, "grandparent_rating_key", record.GrandparentRatingKey, grandparentRatingKey)
		assertIntPtrEqual(t, "row_id", record.RowID, rowID)
		assertIntPtrEqual(t, "optimized_version", record.OptimizedVersion, optVersion)
		assertIntPtrEqual(t, "synced_version", record.SyncedVersion, syncedVersion)
		assertIntPtrEqual(t, "transcode_throttled", record.TranscodeThrottled, transcodeThrottled)
		assertIntPtrEqual(t, "transcode_progress", record.TranscodeProgress, transcodeProgress)
		assertIntPtrEqual(t, "transcode_hw_requested", record.TranscodeHWRequested, transcodeHWRequested)
		assertIntPtrEqual(t, "transcode_hw_decoding", record.TranscodeHWDecoding, transcodeHWDecoding)
		assertIntPtrEqual(t, "transcode_hw_encoding", record.TranscodeHWEncoding, transcodeHWEncoding)
		assertIntPtrEqual(t, "transcode_hw_full_pipeline", record.TranscodeHWFullPipeline, transcodeHWFullPipeline)
		assertIntPtrEqual(t, "transcode_video_width", record.TranscodeVideoWidth, transcodeVideoWidth)
		assertIntPtrEqual(t, "transcode_video_height", record.TranscodeVideoHeight, transcodeVideoHeight)
		assertIntPtrEqual(t, "transcode_audio_channels", record.TranscodeAudioChannels, transcodeAudioChannels)
		assertIntPtrEqual(t, "stream_bitrate", record.StreamBitrate, streamBitrate)
		assertIntPtrEqual(t, "stream_video_bitrate", record.StreamVideoBitrate, streamVideoBitrate)
		assertIntPtrEqual(t, "stream_video_bit_depth", record.StreamVideoBitDepth, streamVideoBitDepth)
		assertIntPtrEqual(t, "stream_video_width", record.StreamVideoWidth, streamVideoWidth)
		assertIntPtrEqual(t, "stream_video_height", record.StreamVideoHeight, streamVideoHeight)
		assertIntPtrEqual(t, "stream_audio_bitrate", record.StreamAudioBitrate, streamAudioBitrate)
		assertIntPtrEqual(t, "stream_audio_sample_rate", record.StreamAudioSampleRate, streamAudioSampleRate)
		assertIntPtrEqual(t, "stream_subtitle_forced", record.StreamSubtitleForced, streamSubtitleForced)
		assertIntPtrEqual(t, "audio_bitrate", record.AudioBitrate, audioBitrate)
		assertIntPtrEqual(t, "audio_sample_rate", record.AudioSampleRate, audioSampleRate)
		assertIntPtrEqual(t, "video_bitrate", record.VideoBitrate, videoBitrate)
		assertIntPtrEqual(t, "video_bit_depth", record.VideoBitDepth, videoBitDepth)
		assertIntPtrEqual(t, "video_ref_frames", record.VideoRefFrames, videoRefFrames)
		assertIntPtrEqual(t, "video_width", record.VideoWidth, videoWidth)
		assertIntPtrEqual(t, "video_height", record.VideoHeight, videoHeight)
		assertIntPtrEqual(t, "subtitle_forced", record.SubtitleForced, subtitleForced)
		assertIntPtrEqual(t, "subtitles", record.Subtitles, subtitles)
		assertIntPtrEqual(t, "secure", record.Secure, secure)
		assertIntPtrEqual(t, "relayed", record.Relayed, relayed)
		assertIntPtrEqual(t, "relay", record.Relay, relay)
		assertIntPtrEqual(t, "local", record.Local, local)
		assertIntPtrEqual(t, "bandwidth", record.Bandwidth, bandwidth)
		assertIntPtrEqual(t, "bitrate", record.Bitrate, bitrate)
		assertIntPtrEqual(t, "is_admin", record.IsAdmin, isAdmin)
		assertIntPtrEqual(t, "is_home_user", record.IsHomeUser, isHomeUser)
		assertIntPtrEqual(t, "is_allow_sync", record.IsAllowSync, isAllowSync)
		assertIntPtrEqual(t, "is_restricted", record.IsRestricted, isRestricted)
		assertIntPtrEqual(t, "keep_history", record.KeepHistory, keepHistory)
		assertIntPtrEqual(t, "deleted_user", record.DeletedUser, deletedUser)
		assertIntPtrEqual(t, "do_notify", record.DoNotify, doNotify)
		assertIntPtrEqual(t, "allow_guest", record.AllowGuest, allowGuest)
		assertIntPtrEqual(t, "live", record.Live, live)
		assertIntPtrEqual(t, "channel_stream", record.ChannelStream, channelStream)
		assertIntPtrEqual(t, "throttled", record.Throttled, throttled)
		assertIntPtrEqual(t, "media_index", record.MediaIndex, mediaIndex)
		assertIntPtrEqual(t, "parent_media_index", record.ParentMediaIndex, parentMediaIndex)
		assertIntPtrEqual(t, "reference_id", record.ReferenceID, referenceID)

		// Verify int64 pointer fields
		assertInt64PtrEqual(t, "file_size", record.FileSize, fileSize)

		// Verify float64 pointer fields
		assertFloat64PtrEqual(t, "watched_status", record.WatchedStatus, watchedStatus)

		// Verify non-pointer string fields
		assertStringEqual(t, "user", record.User, "testuser")
		assertStringEqual(t, "media_type", record.MediaType, "episode")
		assertStringEqual(t, "title", record.Title, "Pilot")
		assertStringEqual(t, "video_dynamic_range", record.VideoDynamicRange, "HDR10")
		assertStringEqual(t, "transcode_hw_decode_title", record.TranscodeHWDecodeTitle, "Intel Quick Sync")
		assertStringEqual(t, "transcode_hw_encode_title", record.TranscodeHWEncodeTitle, "NVIDIA NVENC")
	})

	t.Run("nullable fields as null", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"recordsFiltered": 1,
					"recordsTotal": 1,
					"data": [{
						"session_key": null,
						"date": 1609459200,
						"started": 1609456800,
						"stopped": 1609460400,
						"group_count": null,
						"state": null,
						"user_id": null,
						"user": "testuser",
						"media_type": "movie",
						"title": "Test Movie",
						"parent_title": null,
						"grandparent_title": null,
						"percent_complete": null,
						"paused_counter": null,
						"duration": null,
						"section_id": null,
						"year": null,
						"rating_key": null,
						"parent_rating_key": null,
						"grandparent_rating_key": null,
						"optimized_version": null,
						"synced_version": null,
						"is_admin": null,
						"is_home_user": null,
						"live": null,
						"video_bitrate": null,
						"audio_bitrate": null,
						"file_size": null,
						"secure": null,
						"local": null,
						"bandwidth": null
					}]
				}
			}
		}`

		var history TautulliHistory
		if err := json.Unmarshal([]byte(jsonData), &history); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		record := history.Response.Data.Data[0]

		// Verify string pointer fields are nil
		assertStringPtrNil(t, "session_key", record.SessionKey)
		assertStringPtrNil(t, "state", record.State)
		assertStringPtrNil(t, "parent_title", record.ParentTitle)
		assertStringPtrNil(t, "grandparent_title", record.GrandparentTitle)

		// Verify int pointer fields are nil
		assertIntPtrNil(t, "group_count", record.GroupCount)
		assertIntPtrNil(t, "user_id", record.UserID)
		assertIntPtrNil(t, "percent_complete", record.PercentComplete)
		assertIntPtrNil(t, "duration", record.Duration)
		assertIntPtrNil(t, "section_id", record.SectionID)
		assertIntPtrNil(t, "year", record.Year)
		assertIntPtrNil(t, "rating_key", record.RatingKey)
		assertIntPtrNil(t, "video_bitrate", record.VideoBitrate)
		assertIntPtrNil(t, "secure", record.Secure)
		assertIntPtrNil(t, "bandwidth", record.Bandwidth)

		// Verify int64 pointer fields are nil
		assertInt64PtrNil(t, "file_size", record.FileSize)

		// Non-pointer fields should have values
		assertStringEqual(t, "user", record.User, "testuser")
		assertStringEqual(t, "title", record.Title, "Test Movie")
	})

	t.Run("empty data array", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"recordsFiltered": 0,
					"recordsTotal": 0,
					"data": []
				}
			}
		}`

		var history TautulliHistory
		if err := json.Unmarshal([]byte(jsonData), &history); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(history.Response.Data.Data) != 0 {
			t.Errorf("Expected empty data array, got %d items", len(history.Response.Data.Data))
		}
	})

	t.Run("error response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "error",
				"message": "Invalid API key",
				"data": {}
			}
		}`

		var history TautulliHistory
		if err := json.Unmarshal([]byte(jsonData), &history); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if history.Response.Result != "error" {
			t.Errorf("Expected result 'error', got %q", history.Response.Result)
		}
		if history.Response.Message == nil {
			t.Error("Expected non-nil message")
		} else if *history.Response.Message != "Invalid API key" {
			t.Errorf("Expected message 'Invalid API key', got %q", *history.Response.Message)
		}
	})

	t.Run("movie record without episode fields", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"recordsFiltered": 1,
					"recordsTotal": 1,
					"data": [{
						"date": 1609459200,
						"started": 1609456800,
						"stopped": 1609460400,
						"user": "movieuser",
						"media_type": "movie",
						"title": "Inception",
						"parent_title": null,
						"grandparent_title": null,
						"media_index": null,
						"parent_media_index": null,
						"parent_rating_key": null,
						"grandparent_rating_key": null,
						"video_dynamic_range": "HDR",
						"transcode_decision": "direct play"
					}]
				}
			}
		}`

		var history TautulliHistory
		if err := json.Unmarshal([]byte(jsonData), &history); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		record := history.Response.Data.Data[0]
		assertStringEqual(t, "media_type", record.MediaType, "movie")
		assertStringEqual(t, "title", record.Title, "Inception")
		assertStringPtrNil(t, "parent_title", record.ParentTitle)
		assertStringPtrNil(t, "grandparent_title", record.GrandparentTitle)
		assertIntPtrNil(t, "media_index", record.MediaIndex)
		assertIntPtrNil(t, "parent_media_index", record.ParentMediaIndex)
		assertStringEqual(t, "transcode_decision", record.TranscodeDecision, "direct play")
	})

	t.Run("live TV record", func(t *testing.T) {
		wantLive := 1
		wantChannelStream := 42
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"recordsFiltered": 1,
					"recordsTotal": 1,
					"data": [{
						"date": 1609459200,
						"started": 1609456800,
						"stopped": 1609460400,
						"user": "tvuser",
						"media_type": "episode",
						"title": "Evening News",
						"live": 1,
						"live_uuid": "uuid-12345",
						"channel_stream": 42,
						"channel_call_sign": "NBC",
						"channel_identifier": "nbc-local",
						"duration": null
					}]
				}
			}
		}`

		var history TautulliHistory
		if err := json.Unmarshal([]byte(jsonData), &history); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		record := history.Response.Data.Data[0]
		assertIntPtrEqual(t, "live", record.Live, wantLive)
		assertIntPtrEqual(t, "channel_stream", record.ChannelStream, wantChannelStream)
		assertStringEqual(t, "channel_call_sign", record.ChannelCallSign, "NBC")
		assertStringEqual(t, "channel_identifier", record.ChannelIdentifier, "nbc-local")
		assertIntPtrNil(t, "duration", record.Duration)
	})
}

func TestTautulliHistory_RoundTrip(t *testing.T) {
	msg := "test"
	sessionKey := "session123"
	userID := 99
	parentTitle := "Season 2"
	grandparentTitle := "The Wire"
	percentComplete := 75
	duration := 2700
	sectionID := 5
	year := 2003
	ratingKey := 555
	originalTitle := "Original Title"
	watchedStatus := 0.5
	streamBitrate := 5000000
	videoBitrate := 4000000
	fileSize := int64(4294967296)
	secure := 1
	local := 0

	original := TautulliHistory{
		Response: TautulliHistoryResponse{
			Result:  "success",
			Message: &msg,
			Data: TautulliHistoryData{
				RecordsFiltered: 10,
				RecordsTotal:    100,
				Data: []TautulliHistoryRecord{
					{
						SessionKey:        &sessionKey,
						Date:              1609459200,
						Started:           1609456800,
						Stopped:           1609460400,
						UserID:            &userID,
						User:              "wireuser",
						FriendlyName:      "Wire User",
						MediaType:         "episode",
						Title:             "The Target",
						ParentTitle:       &parentTitle,
						GrandparentTitle:  &grandparentTitle,
						PercentComplete:   &percentComplete,
						Duration:          &duration,
						SectionID:         &sectionID,
						Year:              &year,
						RatingKey:         &ratingKey,
						OriginalTitle:     &originalTitle,
						WatchedStatus:     &watchedStatus,
						TranscodeDecision: "transcode",
						VideoDecision:     "transcode",
						AudioDecision:     "copy",
						StreamBitrate:     &streamBitrate,
						VideoBitrate:      &videoBitrate,
						FileSize:          &fileSize,
						Secure:            &secure,
						Local:             &local,
						VideoDynamicRange: "SDR",
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliHistory
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Response.Data.RecordsTotal != original.Response.Data.RecordsTotal {
		t.Error("RecordsTotal not preserved in round-trip")
	}
	if len(result.Response.Data.Data) != 1 {
		t.Fatal("Data rows not preserved in round-trip")
	}

	record := result.Response.Data.Data[0]
	if record.Title != "The Target" {
		t.Error("Title not preserved in round-trip")
	}
	if record.SessionKey == nil || *record.SessionKey != sessionKey {
		t.Error("SessionKey not preserved in round-trip")
	}
	if record.UserID == nil || *record.UserID != userID {
		t.Error("UserID not preserved in round-trip")
	}
	if record.ParentTitle == nil || *record.ParentTitle != parentTitle {
		t.Error("ParentTitle not preserved in round-trip")
	}
	if record.WatchedStatus == nil || *record.WatchedStatus != watchedStatus {
		t.Error("WatchedStatus not preserved in round-trip")
	}
	if record.StreamBitrate == nil || *record.StreamBitrate != streamBitrate {
		t.Error("StreamBitrate not preserved in round-trip")
	}
	if record.FileSize == nil || *record.FileSize != fileSize {
		t.Error("FileSize not preserved in round-trip")
	}
}

func TestTautulliHistoryRecord_ZeroValues(t *testing.T) {
	// Test that zero values for pointer fields are different from nil
	zero := 0
	zeroInt64 := int64(0)
	zeroFloat := 0.0
	emptyString := ""

	record := TautulliHistoryRecord{
		UserID:          &zero,
		PercentComplete: &zero,
		Duration:        &zero,
		Year:            &zero,
		RatingKey:       &zero,
		FileSize:        &zeroInt64,
		WatchedStatus:   &zeroFloat,
		OriginalTitle:   &emptyString,
	}

	// Zero values should NOT be nil and should contain zero
	assertIntPtrNotNilZero(t, "UserID", record.UserID)
	assertIntPtrNotNilZero(t, "PercentComplete", record.PercentComplete)
	assertIntPtrNotNilZero(t, "Duration", record.Duration)
	assertIntPtrNotNilZero(t, "Year", record.Year)
	assertIntPtrNotNilZero(t, "RatingKey", record.RatingKey)
	assertInt64PtrNotNilZero(t, "FileSize", record.FileSize)
	assertFloat64PtrNotNilZero(t, "WatchedStatus", record.WatchedStatus)
	assertStringPtrNotNilEmpty(t, "OriginalTitle", record.OriginalTitle)
}

func TestTautulliHistoryRecord_LargeValues(t *testing.T) {
	// Test boundary values and large numbers
	largeInt := 2147483647 // MaxInt32
	largeInt64 := int64(9223372036854775807)
	largeProgress := 100
	percent := 100

	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsFiltered": 1,
				"recordsTotal": 1,
				"data": [{
					"date": 9223372036854775807,
					"started": 9223372036854775807,
					"stopped": 9223372036854775807,
					"user": "largetest",
					"media_type": "movie",
					"title": "Large Values Test",
					"user_id": 2147483647,
					"rating_key": 2147483647,
					"percent_complete": 100,
					"duration": 2147483647,
					"file_size": 9223372036854775807,
					"video_bitrate": 2147483647,
					"bandwidth": 2147483647,
					"transcode_progress": 100
				}]
			}
		}
	}`

	var history TautulliHistory
	if err := json.Unmarshal([]byte(jsonData), &history); err != nil {
		t.Fatalf("Failed to unmarshal large values: %v", err)
	}

	record := history.Response.Data.Data[0]

	if record.Date != largeInt64 {
		t.Errorf("Expected date %d, got %d", largeInt64, record.Date)
	}
	assertIntPtrEqual(t, "user_id", record.UserID, largeInt)
	assertIntPtrEqual(t, "rating_key", record.RatingKey, largeInt)
	assertIntPtrEqual(t, "percent_complete", record.PercentComplete, percent)
	assertInt64PtrEqual(t, "file_size", record.FileSize, largeInt64)
	assertIntPtrEqual(t, "transcode_progress", record.TranscodeProgress, largeProgress)
}

func TestTautulliHistoryRecord_SpecialCharacters(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsFiltered": 1,
				"recordsTotal": 1,
				"data": [{
					"date": 1609459200,
					"started": 1609456800,
					"stopped": 1609460400,
					"user": "user<script>alert('xss')</script>",
					"friendly_name": "Test \"Quotes\" & <Brackets>",
					"media_type": "movie",
					"title": "Movie: Part 1 - The \"Beginning\" & More",
					"summary": "A film about \u0000null\u0000 bytes and unicode: \u00e9\u00e0\u00fc\u4e2d\u6587",
					"file": "/media/movies/Test's Movie (2020)/movie.mkv",
					"directors": "John O'Connor, Mary-Jane Smith",
					"genres": "Action & Adventure, Sci-Fi/Fantasy"
				}]
			}
		}
	}`

	var history TautulliHistory
	if err := json.Unmarshal([]byte(jsonData), &history); err != nil {
		t.Fatalf("Failed to unmarshal special characters: %v", err)
	}

	record := history.Response.Data.Data[0]

	assertStringEqual(t, "user", record.User, "user<script>alert('xss')</script>")
	assertStringEqual(t, "friendly_name", record.FriendlyName, "Test \"Quotes\" & <Brackets>")
	assertStringEqual(t, "title", record.Title, "Movie: Part 1 - The \"Beginning\" & More")
	assertStringEqual(t, "directors", record.Directors, "John O'Connor, Mary-Jane Smith")
	assertStringEqual(t, "genres", record.Genres, "Action & Adventure, Sci-Fi/Fantasy")
}
