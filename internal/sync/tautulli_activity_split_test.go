// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestTautulliClient_GetActivity(t *testing.T) {
	tests := []struct {
		name        string
		sessionKey  string
		mockData    tautulli.TautulliActivity
		expectError bool
	}{
		{
			name:       "get all activity",
			sessionKey: "",
			mockData: tautulli.TautulliActivity{
				Response: tautulli.TautulliActivityResponse{
					Result: "success",
					Data: tautulli.TautulliActivityData{
						StreamCount:           2,
						StreamCountDirectPlay: 1,
						StreamCountTranscode:  1,
						TotalBandwidth:        15000,
						LANBandwidth:          10000,
						WANBandwidth:          5000,
						Sessions:              []tautulli.TautulliActivitySession{},
					},
				},
			},
			expectError: false,
		},
		{
			name:       "filter by session key",
			sessionKey: "abc123",
			mockData: tautulli.TautulliActivity{
				Response: tautulli.TautulliActivityResponse{
					Result: "success",
					Data: tautulli.TautulliActivityData{
						StreamCount:    1,
						TotalBandwidth: 5000,
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cmd") != "get_activity" {
					t.Errorf("Expected cmd=get_activity, got %s", r.URL.Query().Get("cmd"))
				}
				if tt.sessionKey != "" && r.URL.Query().Get("session_key") != tt.sessionKey {
					t.Errorf("Expected session_key=%s, got %s", tt.sessionKey, r.URL.Query().Get("session_key"))
				}
				json.NewEncoder(w).Encode(tt.mockData)
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
			client := NewTautulliClient(cfg)

			activity, err := client.GetActivity(context.Background(), tt.sessionKey)
			if (err != nil) != tt.expectError {
				t.Errorf("GetActivity() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError && activity.Response.Data.StreamCount != tt.mockData.Response.Data.StreamCount {
				t.Errorf("Expected stream_count=%d, got %d", tt.mockData.Response.Data.StreamCount, activity.Response.Data.StreamCount)
			}
		})
	}
}

func TestTautulliClient_GetMetadata(t *testing.T) {
	tests := []struct {
		name        string
		ratingKey   string
		expectError bool
	}{
		{
			name:        "valid rating key",
			ratingKey:   "12345",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cmd") != "get_metadata" {
					t.Errorf("Expected cmd=get_metadata, got %s", r.URL.Query().Get("cmd"))
				}
				if r.URL.Query().Get("rating_key") != tt.ratingKey {
					t.Errorf("Expected rating_key=%s, got %s", tt.ratingKey, r.URL.Query().Get("rating_key"))
				}
				mockData := tautulli.TautulliMetadata{
					Response: tautulli.TautulliMetadataResponse{
						Result: "success",
						Data: tautulli.TautulliMetadataData{
							Title:     "Test Movie",
							Year:      2024,
							RatingKey: tt.ratingKey,
							Summary:   "Test summary",
							Rating:    8.5,
							Duration:  7200000,
							Genres:    []string{"Action", "Sci-Fi"},
							Directors: []string{"Director Name"},
							Writers:   []string{"Writer Name"},
							Actors:    []string{"Actor 1", "Actor 2"},
						},
					},
				}
				json.NewEncoder(w).Encode(mockData)
			}))
			defer server.Close()

			cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
			client := NewTautulliClient(cfg)

			metadata, err := client.GetMetadata(context.Background(), tt.ratingKey)
			if (err != nil) != tt.expectError {
				t.Errorf("GetMetadata() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError && metadata.Response.Data.Title != "Test Movie" {
				t.Errorf("Expected title='Test Movie', got '%s'", metadata.Response.Data.Title)
			}
		})
	}
}

func TestTautulliClient_GetStreamData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_stream_data" {
			t.Errorf("Expected cmd=get_stream_data, got %s", r.URL.Query().Get("cmd"))
		}

		// Verify session_key parameter
		if r.URL.Query().Get("session_key") != "test-session-123" {
			t.Errorf("Expected session_key=test-session-123, got %s", r.URL.Query().Get("session_key"))
		}

		response := tautulli.TautulliStreamData{
			Response: tautulli.TautulliStreamDataResponse{
				Result: "success",
				Data: tautulli.TautulliStreamDataInfo{
					SessionKey:              "test-session-123",
					TranscodeDecision:       "transcode",
					VideoDecision:           "transcode",
					AudioDecision:           "copy",
					SubtitleDecision:        "burn",
					Container:               "mkv",
					Bitrate:                 20000,
					VideoCodec:              "hevc",
					VideoResolution:         "4k",
					VideoWidth:              3840,
					VideoHeight:             2160,
					VideoFramerate:          "23.976 fps",
					VideoBitrate:            18000,
					AudioCodec:              "aac",
					AudioChannels:           6,
					AudioChannelLayout:      "5.1(side)",
					AudioBitrate:            640,
					AudioSampleRate:         48000,
					StreamContainer:         "mkv",
					StreamContainerDecision: "transcode",
					StreamBitrate:           8000,
					StreamVideoCodec:        "h264",
					StreamVideoResolution:   "1080p",
					StreamVideoBitrate:      7500,
					StreamVideoWidth:        1920,
					StreamVideoHeight:       1080,
					StreamVideoFramerate:    "23.976 fps",
					StreamAudioCodec:        "aac",
					StreamAudioChannels:     2,
					StreamAudioBitrate:      192,
					StreamAudioSampleRate:   48000,
					SubtitleCodec:           "srt",
					Optimized:               0,
					Throttled:               0,
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	streamData, err := client.GetStreamData(context.Background(), 0, "test-session-123")
	if err != nil {
		t.Fatalf("GetStreamData() error = %v", err)
	}
	if streamData.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", streamData.Response.Result)
	}
	if streamData.Response.Data.SessionKey != "test-session-123" {
		t.Errorf("Expected SessionKey='test-session-123', got '%s'", streamData.Response.Data.SessionKey)
	}
	if streamData.Response.Data.TranscodeDecision != "transcode" {
		t.Errorf("Expected TranscodeDecision='transcode', got '%s'", streamData.Response.Data.TranscodeDecision)
	}
	if streamData.Response.Data.VideoResolution != "4k" {
		t.Errorf("Expected VideoResolution='4k', got '%s'", streamData.Response.Data.VideoResolution)
	}
	if streamData.Response.Data.StreamVideoResolution != "1080p" {
		t.Errorf("Expected StreamVideoResolution='1080p', got '%s'", streamData.Response.Data.StreamVideoResolution)
	}
	if streamData.Response.Data.AudioChannels != 6 {
		t.Errorf("Expected AudioChannels=6, got %d", streamData.Response.Data.AudioChannels)
	}
	if streamData.Response.Data.StreamAudioChannels != 2 {
		t.Errorf("Expected StreamAudioChannels=2, got %d", streamData.Response.Data.StreamAudioChannels)
	}
}
