// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliStreamData_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"session_key": "abc123",
				"transcode_decision": "transcode",
				"video_decision": "transcode",
				"audio_decision": "copy",
				"subtitle_decision": "burn",
				"container": "mkv",
				"bitrate": 8000,
				"video_codec": "hevc",
				"video_resolution": "1080",
				"video_width": 1920,
				"video_height": 1080,
				"video_framerate": "24p",
				"video_bitrate": 6000,
				"audio_codec": "eac3",
				"audio_channels": 6,
				"audio_channel_layout": "5.1",
				"audio_bitrate": 640,
				"audio_sample_rate": 48000,
				"stream_container": "mpegts",
				"stream_container_decision": "transcode",
				"stream_bitrate": 4000,
				"stream_video_codec": "h264",
				"stream_video_resolution": "720",
				"stream_video_bitrate": 3000,
				"stream_video_width": 1280,
				"stream_video_height": 720,
				"stream_video_framerate": "24p",
				"stream_audio_codec": "aac",
				"stream_audio_channels": 2,
				"stream_audio_bitrate": 320,
				"stream_audio_sample_rate": 44100,
				"subtitle_codec": "srt",
				"optimized": 0,
				"throttled": 0
			}
		}
	}`

	var stream TautulliStreamData
	err := json.Unmarshal([]byte(jsonData), &stream)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliStreamData: %v", err)
	}

	if stream.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", stream.Response.Result)
	}

	data := stream.Response.Data
	if data.SessionKey != "abc123" {
		t.Errorf("Expected SessionKey 'abc123', got '%s'", data.SessionKey)
	}
	if data.TranscodeDecision != "transcode" {
		t.Errorf("Expected TranscodeDecision 'transcode', got '%s'", data.TranscodeDecision)
	}
	if data.VideoDecision != "transcode" {
		t.Errorf("Expected VideoDecision 'transcode', got '%s'", data.VideoDecision)
	}
	if data.AudioDecision != "copy" {
		t.Errorf("Expected AudioDecision 'copy', got '%s'", data.AudioDecision)
	}
	if data.SubtitleDecision != "burn" {
		t.Errorf("Expected SubtitleDecision 'burn', got '%s'", data.SubtitleDecision)
	}
	if data.Container != "mkv" {
		t.Errorf("Expected Container 'mkv', got '%s'", data.Container)
	}
	if data.Bitrate != 8000 {
		t.Errorf("Expected Bitrate 8000, got %d", data.Bitrate)
	}
	if data.VideoCodec != "hevc" {
		t.Errorf("Expected VideoCodec 'hevc', got '%s'", data.VideoCodec)
	}
	if data.VideoResolution != "1080" {
		t.Errorf("Expected VideoResolution '1080', got '%s'", data.VideoResolution)
	}
	if data.VideoWidth != 1920 {
		t.Errorf("Expected VideoWidth 1920, got %d", data.VideoWidth)
	}
	if data.VideoHeight != 1080 {
		t.Errorf("Expected VideoHeight 1080, got %d", data.VideoHeight)
	}
	if data.VideoFramerate != "24p" {
		t.Errorf("Expected VideoFramerate '24p', got '%s'", data.VideoFramerate)
	}
	if data.VideoBitrate != 6000 {
		t.Errorf("Expected VideoBitrate 6000, got %d", data.VideoBitrate)
	}
	if data.AudioCodec != "eac3" {
		t.Errorf("Expected AudioCodec 'eac3', got '%s'", data.AudioCodec)
	}
	if data.AudioChannels != 6 {
		t.Errorf("Expected AudioChannels 6, got %d", data.AudioChannels)
	}
	if data.AudioChannelLayout != "5.1" {
		t.Errorf("Expected AudioChannelLayout '5.1', got '%s'", data.AudioChannelLayout)
	}
	if data.AudioBitrate != 640 {
		t.Errorf("Expected AudioBitrate 640, got %d", data.AudioBitrate)
	}
	if data.AudioSampleRate != 48000 {
		t.Errorf("Expected AudioSampleRate 48000, got %d", data.AudioSampleRate)
	}
}

func TestTautulliStreamData_StreamFields(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"session_key": "xyz789",
				"transcode_decision": "direct play",
				"video_decision": "direct play",
				"audio_decision": "direct play",
				"container": "mp4",
				"bitrate": 10000,
				"video_codec": "h264",
				"video_resolution": "4k",
				"video_width": 3840,
				"video_height": 2160,
				"video_framerate": "60p",
				"video_bitrate": 9000,
				"audio_codec": "aac",
				"audio_channels": 2,
				"audio_bitrate": 256,
				"audio_sample_rate": 48000,
				"stream_container": "",
				"stream_container_decision": "",
				"stream_bitrate": 0,
				"stream_video_codec": "",
				"stream_video_resolution": "",
				"stream_video_bitrate": 0,
				"stream_video_width": 0,
				"stream_video_height": 0,
				"stream_video_framerate": "",
				"stream_audio_codec": "",
				"stream_audio_channels": 0,
				"stream_audio_bitrate": 0,
				"stream_audio_sample_rate": 0,
				"optimized": 1,
				"throttled": 0
			}
		}
	}`

	var stream TautulliStreamData
	err := json.Unmarshal([]byte(jsonData), &stream)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliStreamData: %v", err)
	}

	data := stream.Response.Data
	if data.TranscodeDecision != "direct play" {
		t.Errorf("Expected TranscodeDecision 'direct play', got '%s'", data.TranscodeDecision)
	}
	if data.VideoResolution != "4k" {
		t.Errorf("Expected VideoResolution '4k', got '%s'", data.VideoResolution)
	}
	if data.VideoWidth != 3840 {
		t.Errorf("Expected VideoWidth 3840, got %d", data.VideoWidth)
	}
	if data.VideoHeight != 2160 {
		t.Errorf("Expected VideoHeight 2160, got %d", data.VideoHeight)
	}
	if data.Optimized != 1 {
		t.Errorf("Expected Optimized 1, got %d", data.Optimized)
	}
	if data.StreamContainer != "" {
		t.Errorf("Expected empty StreamContainer, got '%s'", data.StreamContainer)
	}
}

func TestTautulliStreamData_DirectStream(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"session_key": "direct123",
				"transcode_decision": "direct stream",
				"video_decision": "copy",
				"audio_decision": "transcode",
				"container": "mkv",
				"bitrate": 15000,
				"video_codec": "hevc",
				"video_resolution": "1080",
				"video_width": 1920,
				"video_height": 1080,
				"video_framerate": "23.976",
				"video_bitrate": 12000,
				"audio_codec": "truehd",
				"audio_channels": 8,
				"audio_channel_layout": "7.1",
				"audio_bitrate": 3000,
				"audio_sample_rate": 48000,
				"stream_audio_codec": "aac",
				"stream_audio_channels": 2,
				"stream_audio_bitrate": 256
			}
		}
	}`

	var stream TautulliStreamData
	err := json.Unmarshal([]byte(jsonData), &stream)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliStreamData: %v", err)
	}

	data := stream.Response.Data
	if data.TranscodeDecision != "direct stream" {
		t.Errorf("Expected TranscodeDecision 'direct stream', got '%s'", data.TranscodeDecision)
	}
	if data.VideoDecision != "copy" {
		t.Errorf("Expected VideoDecision 'copy', got '%s'", data.VideoDecision)
	}
	if data.AudioDecision != "transcode" {
		t.Errorf("Expected AudioDecision 'transcode', got '%s'", data.AudioDecision)
	}
	if data.AudioChannels != 8 {
		t.Errorf("Expected AudioChannels 8, got %d", data.AudioChannels)
	}
	if data.AudioChannelLayout != "7.1" {
		t.Errorf("Expected AudioChannelLayout '7.1', got '%s'", data.AudioChannelLayout)
	}
	if data.StreamAudioCodec != "aac" {
		t.Errorf("Expected StreamAudioCodec 'aac', got '%s'", data.StreamAudioCodec)
	}
	if data.StreamAudioChannels != 2 {
		t.Errorf("Expected StreamAudioChannels 2, got %d", data.StreamAudioChannels)
	}
}

func TestTautulliStreamData_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Session not found",
			"data": {
				"session_key": "",
				"transcode_decision": "",
				"video_decision": "",
				"audio_decision": "",
				"container": "",
				"bitrate": 0,
				"video_codec": "",
				"video_resolution": "",
				"video_width": 0,
				"video_height": 0,
				"video_framerate": "",
				"video_bitrate": 0,
				"audio_codec": "",
				"audio_channels": 0,
				"audio_bitrate": 0,
				"audio_sample_rate": 0
			}
		}
	}`

	var stream TautulliStreamData
	err := json.Unmarshal([]byte(jsonData), &stream)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliStreamData: %v", err)
	}

	if stream.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", stream.Response.Result)
	}
	if stream.Response.Message == nil || *stream.Response.Message != "Session not found" {
		t.Error("Expected Message 'Session not found'")
	}
}

func TestTautulliStreamData_JSONRoundTrip(t *testing.T) {
	original := TautulliStreamData{
		Response: TautulliStreamDataResponse{
			Result: "success",
			Data: TautulliStreamDataInfo{
				SessionKey:         "session456",
				TranscodeDecision:  "transcode",
				VideoDecision:      "transcode",
				AudioDecision:      "copy",
				Container:          "mkv",
				Bitrate:            5000,
				VideoCodec:         "h264",
				VideoResolution:    "720",
				VideoWidth:         1280,
				VideoHeight:        720,
				VideoFramerate:     "30p",
				VideoBitrate:       4000,
				AudioCodec:         "aac",
				AudioChannels:      2,
				AudioChannelLayout: "stereo",
				AudioBitrate:       256,
				AudioSampleRate:    48000,
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliStreamData: %v", err)
	}

	var decoded TautulliStreamData
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliStreamData: %v", err)
	}

	if decoded.Response.Data.SessionKey != original.Response.Data.SessionKey {
		t.Errorf("SessionKey mismatch")
	}
	if decoded.Response.Data.Bitrate != original.Response.Data.Bitrate {
		t.Errorf("Bitrate mismatch")
	}
	if decoded.Response.Data.VideoWidth != original.Response.Data.VideoWidth {
		t.Errorf("VideoWidth mismatch")
	}
}

func TestTautulliStreamData_ThrottledStream(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"session_key": "throttled123",
				"transcode_decision": "transcode",
				"video_decision": "transcode",
				"audio_decision": "transcode",
				"container": "mkv",
				"bitrate": 20000,
				"video_codec": "hevc",
				"video_resolution": "4k",
				"video_width": 3840,
				"video_height": 2160,
				"video_framerate": "24p",
				"video_bitrate": 18000,
				"audio_codec": "dts-hd",
				"audio_channels": 8,
				"audio_bitrate": 2000,
				"audio_sample_rate": 48000,
				"throttled": 1
			}
		}
	}`

	var stream TautulliStreamData
	err := json.Unmarshal([]byte(jsonData), &stream)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliStreamData: %v", err)
	}

	if stream.Response.Data.Throttled != 1 {
		t.Errorf("Expected Throttled 1, got %d", stream.Response.Data.Throttled)
	}
}

func TestTautulliStreamData_SubtitleHandling(t *testing.T) {
	testCases := []struct {
		name             string
		subtitleDecision string
		subtitleCodec    string
	}{
		{"Burn Subtitles", "burn", "ass"},
		{"Copy Subtitles", "copy", "srt"},
		{"Transcode Subtitles", "transcode", "pgs"},
		{"No Subtitles", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := TautulliStreamData{
				Response: TautulliStreamDataResponse{
					Result: "success",
					Data: TautulliStreamDataInfo{
						SessionKey:       "sub_test",
						SubtitleDecision: tc.subtitleDecision,
						SubtitleCodec:    tc.subtitleCodec,
					},
				},
			}

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded TautulliStreamData
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Response.Data.SubtitleDecision != tc.subtitleDecision {
				t.Errorf("Expected SubtitleDecision '%s', got '%s'", tc.subtitleDecision, decoded.Response.Data.SubtitleDecision)
			}
			if decoded.Response.Data.SubtitleCodec != tc.subtitleCodec {
				t.Errorf("Expected SubtitleCodec '%s', got '%s'", tc.subtitleCodec, decoded.Response.Data.SubtitleCodec)
			}
		})
	}
}

func TestTautulliStreamDataInfo_AllOptionalFieldsEmpty(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"session_key": "minimal",
				"transcode_decision": "direct play",
				"video_decision": "direct play",
				"audio_decision": "direct play",
				"container": "mp4",
				"bitrate": 5000,
				"video_codec": "h264",
				"video_resolution": "1080",
				"video_width": 1920,
				"video_height": 1080,
				"video_framerate": "24p",
				"video_bitrate": 4500,
				"audio_codec": "aac",
				"audio_channels": 2,
				"audio_bitrate": 256,
				"audio_sample_rate": 44100
			}
		}
	}`

	var stream TautulliStreamData
	err := json.Unmarshal([]byte(jsonData), &stream)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliStreamData: %v", err)
	}

	data := stream.Response.Data
	// Check that optional fields default to zero values
	if data.SubtitleDecision != "" {
		t.Errorf("Expected empty SubtitleDecision, got '%s'", data.SubtitleDecision)
	}
	if data.AudioChannelLayout != "" {
		t.Errorf("Expected empty AudioChannelLayout, got '%s'", data.AudioChannelLayout)
	}
	if data.StreamContainer != "" {
		t.Errorf("Expected empty StreamContainer, got '%s'", data.StreamContainer)
	}
	if data.Optimized != 0 {
		t.Errorf("Expected Optimized 0, got %d", data.Optimized)
	}
	if data.Throttled != 0 {
		t.Errorf("Expected Throttled 0, got %d", data.Throttled)
	}
}
