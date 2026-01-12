// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliActivity_JSONUnmarshal(t *testing.T) {
	t.Run("complete response with multiple sessions", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"message": null,
				"data": {
					"lan_bandwidth": 50000,
					"wan_bandwidth": 25000,
					"total_bandwidth": 75000,
					"stream_count": 3,
					"stream_count_direct_play": 1,
					"stream_count_direct_stream": 1,
					"stream_count_transcode": 1,
					"sessions": [
						{
							"session_key": "abc123",
							"session_id": "session-001",
							"media_type": "episode",
							"rating_key": "12345",
							"parent_rating_key": "1234",
							"grandparent_rating_key": "123",
							"title": "Pilot",
							"parent_title": "Season 1",
							"grandparent_title": "Breaking Bad",
							"full_title": "Breaking Bad - S01E01 - Pilot",
							"original_title": "Original Pilot",
							"sort_title": "Pilot",
							"media_index": "1",
							"parent_media_index": "1",
							"year": 2008,
							"thumb": "/thumb/episode.jpg",
							"parent_thumb": "/thumb/season.jpg",
							"grandparent_thumb": "/thumb/show.jpg",
							"art": "/art/episode.jpg",
							"grandparent_art": "/art/show.jpg",
							"user": "testuser",
							"user_id": 42,
							"friendly_name": "Test User",
							"user_thumb": "/thumb/user.jpg",
							"email": "test@example.com",
							"is_admin": 0,
							"is_home_user": 1,
							"is_allow_sync": 1,
							"is_restricted": 0,
							"keep_history": 1,
							"deleted_user": 0,
							"do_notify": 1,
							"ip_address": "192.168.1.100",
							"ip_address_public": "203.0.113.50",
							"player": "Roku",
							"platform": "Roku",
							"platform_name": "Roku Ultra",
							"platform_version": "10.5",
							"product": "Plex for Roku",
							"product_version": "7.0.0",
							"device": "Roku Ultra",
							"machine_id": "ROKU123",
							"local": 1,
							"quality_profile": "1080p",
							"optimized_version": 0,
							"synced_version": 0,
							"state": "playing",
							"view_offset": 1200000,
							"duration": 3600000,
							"progress_percent": 33,
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
							"transcode_audio_codec": "aac",
							"transcode_audio_channels": 6,
							"width": 3840,
							"height": 2160,
							"container": "mkv",
							"video_codec": "hevc",
							"video_codec_level": "5.1",
							"video_bitrate": 10000000,
							"video_bit_depth": 10,
							"video_framerate": "23.976",
							"video_ref_frames": 4,
							"video_resolution": "4k",
							"video_full_resolution": "2160p",
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
							"audio_bitrate": 768000,
							"audio_bitrate_mode": "cbr",
							"audio_channels": 8,
							"audio_channel_layout": "7.1",
							"audio_sample_rate": 48000,
							"audio_language": "English",
							"audio_language_code": "eng",
							"audio_profile": "Dolby TrueHD",
							"subtitle_codec": "srt",
							"subtitle_container": "srt",
							"subtitle_format": "srt",
							"subtitle_language": "English",
							"subtitle_language_code": "eng",
							"subtitle_forced": 0,
							"subtitle_location": "external",
							"subtitles": 1,
							"stream_container": "mkv",
							"stream_bitrate": 8000000,
							"stream_video_codec": "h264",
							"stream_video_codec_level": "4.1",
							"stream_video_resolution": "1080",
							"stream_video_full_resolution": "1080p",
							"stream_video_decision": "transcode",
							"stream_video_bitrate": 6000000,
							"stream_video_bit_depth": 8,
							"stream_video_width": 1920,
							"stream_video_height": 1080,
							"stream_video_framerate": "23.976",
							"stream_video_profile": "high",
							"stream_video_scan_type": "progressive",
							"stream_video_language": "English",
							"stream_video_language_code": "eng",
							"stream_video_dynamic_range": "SDR",
							"stream_audio_codec": "aac",
							"stream_audio_channels": 6,
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
							"bitrate": 10000000,
							"bandwidth": 25000,
							"location": "wan",
							"secure": 1,
							"relayed": 0,
							"relay": 0,
							"section_id": "1",
							"library_name": "TV Shows",
							"content_rating": "TV-MA",
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
							"file": "/media/tv/breaking_bad/s01e01.mkv",
							"file_size": 8589934592,
							"guid": "plex://episode/12345",
							"parent_guid": "plex://season/1234",
							"grandparent_guid": "plex://show/123"
						}
					]
				}
			}
		}`

		var activity TautulliActivity
		if err := json.Unmarshal([]byte(jsonData), &activity); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if activity.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", activity.Response.Result)
		}

		data := activity.Response.Data
		if data.LANBandwidth != 50000 {
			t.Errorf("Expected lan_bandwidth 50000, got %d", data.LANBandwidth)
		}
		if data.WANBandwidth != 25000 {
			t.Errorf("Expected wan_bandwidth 25000, got %d", data.WANBandwidth)
		}
		if data.TotalBandwidth != 75000 {
			t.Errorf("Expected total_bandwidth 75000, got %d", data.TotalBandwidth)
		}
		if data.StreamCount != 3 {
			t.Errorf("Expected stream_count 3, got %d", data.StreamCount)
		}
		if data.StreamCountDirectPlay != 1 {
			t.Errorf("Expected stream_count_direct_play 1, got %d", data.StreamCountDirectPlay)
		}
		if data.StreamCountDirectStream != 1 {
			t.Errorf("Expected stream_count_direct_stream 1, got %d", data.StreamCountDirectStream)
		}
		if data.StreamCountTranscode != 1 {
			t.Errorf("Expected stream_count_transcode 1, got %d", data.StreamCountTranscode)
		}

		if len(data.Sessions) != 1 {
			t.Fatalf("Expected 1 session, got %d", len(data.Sessions))
		}

		session := data.Sessions[0]

		// Session identification
		if session.SessionKey != "abc123" {
			t.Errorf("Expected session_key 'abc123', got %q", session.SessionKey)
		}
		if session.SessionID != "session-001" {
			t.Errorf("Expected session_id 'session-001', got %q", session.SessionID)
		}

		// Media info
		if session.MediaType != "episode" {
			t.Errorf("Expected media_type 'episode', got %q", session.MediaType)
		}
		if session.Title != "Pilot" {
			t.Errorf("Expected title 'Pilot', got %q", session.Title)
		}
		if session.ParentTitle != "Season 1" {
			t.Errorf("Expected parent_title 'Season 1', got %q", session.ParentTitle)
		}
		if session.GrandparentTitle != "Breaking Bad" {
			t.Errorf("Expected grandparent_title 'Breaking Bad', got %q", session.GrandparentTitle)
		}
		if session.Year != 2008 {
			t.Errorf("Expected year 2008, got %d", session.Year)
		}

		// User info
		if session.User != "testuser" {
			t.Errorf("Expected user 'testuser', got %q", session.User)
		}
		if session.UserID != 42 {
			t.Errorf("Expected user_id 42, got %d", session.UserID)
		}
		if session.IsAdmin != 0 {
			t.Errorf("Expected is_admin 0, got %d", session.IsAdmin)
		}
		if session.IsHomeUser != 1 {
			t.Errorf("Expected is_home_user 1, got %d", session.IsHomeUser)
		}

		// Player info
		if session.Player != "Roku" {
			t.Errorf("Expected player 'Roku', got %q", session.Player)
		}
		if session.Platform != "Roku" {
			t.Errorf("Expected platform 'Roku', got %q", session.Platform)
		}
		if session.MachineID != "ROKU123" {
			t.Errorf("Expected machine_id 'ROKU123', got %q", session.MachineID)
		}

		// Playback state
		if session.State != "playing" {
			t.Errorf("Expected state 'playing', got %q", session.State)
		}
		if session.ViewOffset != 1200000 {
			t.Errorf("Expected view_offset 1200000, got %d", session.ViewOffset)
		}
		if session.Duration != 3600000 {
			t.Errorf("Expected duration 3600000, got %d", session.Duration)
		}
		if session.ProgressPercent != 33 {
			t.Errorf("Expected progress_percent 33, got %d", session.ProgressPercent)
		}

		// Transcode info
		if session.TranscodeDecision != "transcode" {
			t.Errorf("Expected transcode_decision 'transcode', got %q", session.TranscodeDecision)
		}
		if session.TranscodeHWDecoding != 1 {
			t.Errorf("Expected transcode_hw_decoding 1, got %d", session.TranscodeHWDecoding)
		}
		if session.TranscodeHWEncoding != 1 {
			t.Errorf("Expected transcode_hw_encoding 1, got %d", session.TranscodeHWEncoding)
		}
		if session.TranscodeHWDecodeTitle != "Intel Quick Sync" {
			t.Errorf("Expected transcode_hw_decode_title 'Intel Quick Sync', got %q", session.TranscodeHWDecodeTitle)
		}
		if session.TranscodeHWEncodeTitle != "NVIDIA NVENC" {
			t.Errorf("Expected transcode_hw_encode_title 'NVIDIA NVENC', got %q", session.TranscodeHWEncodeTitle)
		}

		// Video quality
		if session.Width != 3840 {
			t.Errorf("Expected width 3840, got %d", session.Width)
		}
		if session.Height != 2160 {
			t.Errorf("Expected height 2160, got %d", session.Height)
		}
		if session.VideoResolution != "4k" {
			t.Errorf("Expected video_resolution '4k', got %q", session.VideoResolution)
		}
		if session.VideoDynamicRange != "HDR10" {
			t.Errorf("Expected video_dynamic_range 'HDR10', got %q", session.VideoDynamicRange)
		}
		if session.VideoBitrate != 10000000 {
			t.Errorf("Expected video_bitrate 10000000, got %d", session.VideoBitrate)
		}
		if session.VideoBitDepth != 10 {
			t.Errorf("Expected video_bit_depth 10, got %d", session.VideoBitDepth)
		}

		// Audio quality
		if session.AudioCodec != "truehd" {
			t.Errorf("Expected audio_codec 'truehd', got %q", session.AudioCodec)
		}
		if session.AudioChannels != 8 {
			t.Errorf("Expected audio_channels 8, got %d", session.AudioChannels)
		}
		if session.AudioChannelLayout != "7.1" {
			t.Errorf("Expected audio_channel_layout '7.1', got %q", session.AudioChannelLayout)
		}

		// Stream output
		if session.StreamVideoResolution != "1080" {
			t.Errorf("Expected stream_video_resolution '1080', got %q", session.StreamVideoResolution)
		}
		if session.StreamVideoDynamicRange != "SDR" {
			t.Errorf("Expected stream_video_dynamic_range 'SDR', got %q", session.StreamVideoDynamicRange)
		}

		// Network
		if session.Location != "wan" {
			t.Errorf("Expected location 'wan', got %q", session.Location)
		}
		if session.Secure != 1 {
			t.Errorf("Expected secure 1, got %d", session.Secure)
		}
		if session.Bandwidth != 25000 {
			t.Errorf("Expected bandwidth 25000, got %d", session.Bandwidth)
		}

		// File info
		if session.FileSize != 8589934592 {
			t.Errorf("Expected file_size 8589934592, got %d", session.FileSize)
		}
	})

	t.Run("no active sessions", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"lan_bandwidth": 0,
					"wan_bandwidth": 0,
					"total_bandwidth": 0,
					"stream_count": 0,
					"stream_count_direct_play": 0,
					"stream_count_direct_stream": 0,
					"stream_count_transcode": 0,
					"sessions": []
				}
			}
		}`

		var activity TautulliActivity
		if err := json.Unmarshal([]byte(jsonData), &activity); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if activity.Response.Data.StreamCount != 0 {
			t.Errorf("Expected stream_count 0, got %d", activity.Response.Data.StreamCount)
		}
		if len(activity.Response.Data.Sessions) != 0 {
			t.Errorf("Expected empty sessions, got %d", len(activity.Response.Data.Sessions))
		}
	})

	t.Run("error response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "error",
				"message": "Server not responding",
				"data": {}
			}
		}`

		var activity TautulliActivity
		if err := json.Unmarshal([]byte(jsonData), &activity); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if activity.Response.Result != "error" {
			t.Errorf("Expected result 'error', got %q", activity.Response.Result)
		}
		if activity.Response.Message == nil {
			t.Error("Expected non-nil message")
		} else if *activity.Response.Message != "Server not responding" {
			t.Errorf("Expected message 'Server not responding', got %q", *activity.Response.Message)
		}
	})

	t.Run("live TV session", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"lan_bandwidth": 10000,
					"wan_bandwidth": 0,
					"total_bandwidth": 10000,
					"stream_count": 1,
					"stream_count_direct_play": 1,
					"stream_count_direct_stream": 0,
					"stream_count_transcode": 0,
					"sessions": [{
						"session_key": "live123",
						"session_id": "live-session-001",
						"media_type": "episode",
						"title": "Evening News",
						"user": "liveuser",
						"user_id": 10,
						"state": "playing",
						"live": 1,
						"live_uuid": "uuid-live-12345",
						"channel_stream": 42,
						"channel_call_sign": "NBC",
						"channel_identifier": "nbc-local-hd",
						"transcode_decision": "direct play"
					}]
				}
			}
		}`

		var activity TautulliActivity
		if err := json.Unmarshal([]byte(jsonData), &activity); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		session := activity.Response.Data.Sessions[0]
		if session.Live != 1 {
			t.Errorf("Expected live 1, got %d", session.Live)
		}
		if session.LiveUUID != "uuid-live-12345" {
			t.Errorf("Expected live_uuid 'uuid-live-12345', got %q", session.LiveUUID)
		}
		if session.ChannelStream != 42 {
			t.Errorf("Expected channel_stream 42, got %d", session.ChannelStream)
		}
		if session.ChannelCallSign != "NBC" {
			t.Errorf("Expected channel_call_sign 'NBC', got %q", session.ChannelCallSign)
		}
		if session.ChannelIdentifier != "nbc-local-hd" {
			t.Errorf("Expected channel_identifier 'nbc-local-hd', got %q", session.ChannelIdentifier)
		}
	})

	t.Run("paused session", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"lan_bandwidth": 0,
					"wan_bandwidth": 0,
					"total_bandwidth": 0,
					"stream_count": 1,
					"stream_count_direct_play": 0,
					"stream_count_direct_stream": 0,
					"stream_count_transcode": 1,
					"sessions": [{
						"session_key": "paused123",
						"session_id": "paused-session-001",
						"media_type": "movie",
						"title": "Inception",
						"user": "pauseduser",
						"user_id": 20,
						"state": "paused",
						"view_offset": 3600000,
						"duration": 7200000,
						"progress_percent": 50,
						"throttled": 1,
						"transcode_decision": "transcode",
						"transcode_throttled": 1,
						"transcode_progress": 75
					}]
				}
			}
		}`

		var activity TautulliActivity
		if err := json.Unmarshal([]byte(jsonData), &activity); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		session := activity.Response.Data.Sessions[0]
		if session.State != "paused" {
			t.Errorf("Expected state 'paused', got %q", session.State)
		}
		if session.ViewOffset != 3600000 {
			t.Errorf("Expected view_offset 3600000, got %d", session.ViewOffset)
		}
		if session.ProgressPercent != 50 {
			t.Errorf("Expected progress_percent 50, got %d", session.ProgressPercent)
		}
		if session.Throttled != 1 {
			t.Errorf("Expected throttled 1, got %d", session.Throttled)
		}
		if session.TranscodeThrottled != 1 {
			t.Errorf("Expected transcode_throttled 1, got %d", session.TranscodeThrottled)
		}
		if session.TranscodeProgress != 75 {
			t.Errorf("Expected transcode_progress 75, got %d", session.TranscodeProgress)
		}
	})

	t.Run("multiple sessions with different states", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"lan_bandwidth": 50000,
					"wan_bandwidth": 30000,
					"total_bandwidth": 80000,
					"stream_count": 3,
					"stream_count_direct_play": 1,
					"stream_count_direct_stream": 1,
					"stream_count_transcode": 1,
					"sessions": [
						{
							"session_key": "session1",
							"session_id": "s1",
							"media_type": "movie",
							"title": "Movie 1",
							"user": "user1",
							"user_id": 1,
							"state": "playing",
							"transcode_decision": "direct play"
						},
						{
							"session_key": "session2",
							"session_id": "s2",
							"media_type": "episode",
							"title": "Episode 1",
							"user": "user2",
							"user_id": 2,
							"state": "paused",
							"transcode_decision": "direct stream"
						},
						{
							"session_key": "session3",
							"session_id": "s3",
							"media_type": "track",
							"title": "Music Track",
							"user": "user3",
							"user_id": 3,
							"state": "buffering",
							"transcode_decision": "transcode"
						}
					]
				}
			}
		}`

		var activity TautulliActivity
		if err := json.Unmarshal([]byte(jsonData), &activity); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(activity.Response.Data.Sessions) != 3 {
			t.Fatalf("Expected 3 sessions, got %d", len(activity.Response.Data.Sessions))
		}

		// Verify each session has different state
		states := make(map[string]bool)
		for _, s := range activity.Response.Data.Sessions {
			states[s.State] = true
		}

		expectedStates := []string{"playing", "paused", "buffering"}
		for _, state := range expectedStates {
			if !states[state] {
				t.Errorf("Expected state '%s' to be present", state)
			}
		}

		// Verify stream counts match decisions
		if activity.Response.Data.StreamCountDirectPlay != 1 {
			t.Errorf("Expected 1 direct play, got %d", activity.Response.Data.StreamCountDirectPlay)
		}
		if activity.Response.Data.StreamCountDirectStream != 1 {
			t.Errorf("Expected 1 direct stream, got %d", activity.Response.Data.StreamCountDirectStream)
		}
		if activity.Response.Data.StreamCountTranscode != 1 {
			t.Errorf("Expected 1 transcode, got %d", activity.Response.Data.StreamCountTranscode)
		}
	})
}

func TestTautulliActivity_RoundTrip(t *testing.T) {
	msg := "test"
	original := TautulliActivity{
		Response: TautulliActivityResponse{
			Result:  "success",
			Message: &msg,
			Data: TautulliActivityData{
				LANBandwidth:            100000,
				WANBandwidth:            50000,
				TotalBandwidth:          150000,
				StreamCount:             2,
				StreamCountDirectPlay:   1,
				StreamCountDirectStream: 0,
				StreamCountTranscode:    1,
				Sessions: []TautulliActivitySession{
					{
						SessionKey:            "roundtrip123",
						SessionID:             "rt-session",
						MediaType:             "movie",
						RatingKey:             "99999",
						Title:                 "Round Trip Movie",
						User:                  "rtuser",
						UserID:                50,
						State:                 "playing",
						ViewOffset:            600000,
						Duration:              7200000,
						ProgressPercent:       8,
						TranscodeDecision:     "transcode",
						VideoDecision:         "transcode",
						AudioDecision:         "copy",
						Width:                 1920,
						Height:                1080,
						VideoCodec:            "h264",
						VideoResolution:       "1080",
						VideoDynamicRange:     "SDR",
						AudioCodec:            "ac3",
						AudioChannels:         6,
						StreamVideoResolution: "720",
						Bandwidth:             20000,
						Location:              "wan",
						Secure:                1,
						FileSize:              4294967296,
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliActivity
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Response.Data.TotalBandwidth != original.Response.Data.TotalBandwidth {
		t.Error("TotalBandwidth not preserved in round-trip")
	}
	if result.Response.Data.StreamCount != original.Response.Data.StreamCount {
		t.Error("StreamCount not preserved in round-trip")
	}
	if len(result.Response.Data.Sessions) != 1 {
		t.Fatal("Sessions not preserved in round-trip")
	}

	session := result.Response.Data.Sessions[0]
	if session.Title != "Round Trip Movie" {
		t.Error("Title not preserved in round-trip")
	}
	if session.SessionKey != "roundtrip123" {
		t.Error("SessionKey not preserved in round-trip")
	}
	if session.ViewOffset != 600000 {
		t.Error("ViewOffset not preserved in round-trip")
	}
	if session.VideoDynamicRange != "SDR" {
		t.Error("VideoDynamicRange not preserved in round-trip")
	}
	if session.FileSize != 4294967296 {
		t.Error("FileSize not preserved in round-trip")
	}
}

func TestTautulliActivitySession_ZeroValues(t *testing.T) {
	session := TautulliActivitySession{
		SessionKey:          "test",
		SessionID:           "test-id",
		UserID:              0, // Explicitly zero
		ViewOffset:          0,
		Duration:            0,
		ProgressPercent:     0,
		Width:               0,
		Height:              0,
		VideoBitrate:        0,
		AudioBitrate:        0,
		AudioChannels:       0,
		AudioSampleRate:     0,
		StreamBitrate:       0,
		FileSize:            0,
		Bandwidth:           0,
		Secure:              0,
		Local:               0,
		Live:                0,
		Throttled:           0,
		TranscodeHWDecoding: 0,
		TranscodeHWEncoding: 0,
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliActivitySession
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// All zero values should be preserved
	if result.UserID != 0 {
		t.Errorf("Expected user_id 0, got %d", result.UserID)
	}
	if result.ViewOffset != 0 {
		t.Errorf("Expected view_offset 0, got %d", result.ViewOffset)
	}
	if result.ProgressPercent != 0 {
		t.Errorf("Expected progress_percent 0, got %d", result.ProgressPercent)
	}
	if result.Width != 0 {
		t.Errorf("Expected width 0, got %d", result.Width)
	}
	if result.VideoBitrate != 0 {
		t.Errorf("Expected video_bitrate 0, got %d", result.VideoBitrate)
	}
	if result.AudioChannels != 0 {
		t.Errorf("Expected audio_channels 0, got %d", result.AudioChannels)
	}
	if result.FileSize != 0 {
		t.Errorf("Expected file_size 0, got %d", result.FileSize)
	}
	if result.Live != 0 {
		t.Errorf("Expected live 0, got %d", result.Live)
	}
}

func TestTautulliActivitySession_LargeValues(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"lan_bandwidth": 2147483647,
				"wan_bandwidth": 2147483647,
				"total_bandwidth": 2147483647,
				"stream_count": 100,
				"stream_count_direct_play": 50,
				"stream_count_direct_stream": 25,
				"stream_count_transcode": 25,
				"sessions": [{
					"session_key": "large123",
					"session_id": "large-session",
					"media_type": "movie",
					"title": "Large Values",
					"user": "largeuser",
					"user_id": 2147483647,
					"view_offset": 2147483647,
					"duration": 2147483647,
					"progress_percent": 100,
					"width": 7680,
					"height": 4320,
					"video_bitrate": 100000000,
					"video_bit_depth": 12,
					"audio_bitrate": 1536000,
					"audio_channels": 16,
					"audio_sample_rate": 192000,
					"file_size": 9223372036854775807,
					"bandwidth": 2147483647,
					"stream_bitrate": 100000000
				}]
			}
		}
	}`

	var activity TautulliActivity
	if err := json.Unmarshal([]byte(jsonData), &activity); err != nil {
		t.Fatalf("Failed to unmarshal large values: %v", err)
	}

	data := activity.Response.Data
	if data.LANBandwidth != 2147483647 {
		t.Errorf("Expected lan_bandwidth 2147483647, got %d", data.LANBandwidth)
	}
	if data.StreamCount != 100 {
		t.Errorf("Expected stream_count 100, got %d", data.StreamCount)
	}

	session := data.Sessions[0]
	if session.UserID != 2147483647 {
		t.Errorf("Expected user_id 2147483647, got %d", session.UserID)
	}
	if session.Width != 7680 {
		t.Errorf("Expected width 7680 (8K), got %d", session.Width)
	}
	if session.Height != 4320 {
		t.Errorf("Expected height 4320 (8K), got %d", session.Height)
	}
	if session.VideoBitDepth != 12 {
		t.Errorf("Expected video_bit_depth 12, got %d", session.VideoBitDepth)
	}
	if session.AudioChannels != 16 {
		t.Errorf("Expected audio_channels 16, got %d", session.AudioChannels)
	}
	if session.FileSize != 9223372036854775807 {
		t.Errorf("Expected file_size MaxInt64, got %d", session.FileSize)
	}
}

func TestTautulliActivitySession_SpecialCharacters(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"lan_bandwidth": 0,
				"wan_bandwidth": 0,
				"total_bandwidth": 0,
				"stream_count": 1,
				"stream_count_direct_play": 1,
				"stream_count_direct_stream": 0,
				"stream_count_transcode": 0,
				"sessions": [{
					"session_key": "special123",
					"session_id": "special-session",
					"media_type": "movie",
					"title": "Movie: Part 1 - The \"Beginning\" & More",
					"user": "user<script>alert('xss')</script>",
					"friendly_name": "Test \"Quotes\" & <Brackets>",
					"user_id": 1,
					"file": "/media/movies/Test's Movie (2020)/movie.mkv",
					"summary": "A film about unicode: \u00e9\u00e0\u00fc\u4e2d\u6587",
					"tagline": "Coming soon\u2122",
					"studio": "O'Brien & Associates"
				}]
			}
		}
	}`

	var activity TautulliActivity
	if err := json.Unmarshal([]byte(jsonData), &activity); err != nil {
		t.Fatalf("Failed to unmarshal special characters: %v", err)
	}

	session := activity.Response.Data.Sessions[0]

	if session.User != "user<script>alert('xss')</script>" {
		t.Errorf("User with HTML not preserved correctly: %q", session.User)
	}
	if session.FriendlyName != "Test \"Quotes\" & <Brackets>" {
		t.Errorf("FriendlyName with quotes not preserved: %q", session.FriendlyName)
	}
	if session.Title != "Movie: Part 1 - The \"Beginning\" & More" {
		t.Errorf("Title with special chars not preserved: %q", session.Title)
	}
	if session.File != "/media/movies/Test's Movie (2020)/movie.mkv" {
		t.Errorf("File path with apostrophe not preserved: %q", session.File)
	}
	if session.Studio != "O'Brien & Associates" {
		t.Errorf("Studio with special chars not preserved: %q", session.Studio)
	}
}

func TestTautulliActivityData_BandwidthCalculations(t *testing.T) {
	testCases := []struct {
		name          string
		lan           int
		wan           int
		expectedTotal int
	}{
		{"LAN only", 50000, 0, 50000},
		{"WAN only", 0, 30000, 30000},
		{"Both LAN and WAN", 50000, 30000, 80000},
		{"Zero bandwidth", 0, 0, 0},
		{"High bandwidth", 1000000, 500000, 1500000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := TautulliActivityData{
				LANBandwidth:   tc.lan,
				WANBandwidth:   tc.wan,
				TotalBandwidth: tc.expectedTotal,
			}

			if data.LANBandwidth+data.WANBandwidth != data.TotalBandwidth {
				t.Errorf("Bandwidth calculation mismatch: LAN(%d) + WAN(%d) should equal Total(%d)",
					data.LANBandwidth, data.WANBandwidth, data.TotalBandwidth)
			}
		})
	}
}
