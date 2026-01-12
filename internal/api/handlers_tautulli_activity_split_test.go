// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestTautulliActivity_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetActivityFunc: func(ctx context.Context, sessionKey string) (*tautulli.TautulliActivity, error) {
			return &tautulli.TautulliActivity{
				Response: tautulli.TautulliActivityResponse{
					Result: "success",
					Data: tautulli.TautulliActivityData{
						StreamCount:    2,
						TotalBandwidth: 10000,
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/activity", nil)
	w := httptest.NewRecorder()

	handler.TautulliActivity(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

func TestTautulliMetadata_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetMetadataFunc: func(ctx context.Context, ratingKey string) (*tautulli.TautulliMetadata, error) {
			return &tautulli.TautulliMetadata{
				Response: tautulli.TautulliMetadataResponse{
					Result: "success",
					Data: tautulli.TautulliMetadataData{
						Title: "Test Movie",
						Year:  2024,
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/metadata?rating_key=12345", nil)
	w := httptest.NewRecorder()

	handler.TautulliMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTautulliMetadata_MissingRatingKey(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/metadata", nil)
	w := httptest.NewRecorder()

	handler.TautulliMetadata(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestTautulliStreamData_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetStreamDataFunc: func(ctx context.Context, rowID int, sessionKey string) (*tautulli.TautulliStreamData, error) {
			return &tautulli.TautulliStreamData{
				Response: tautulli.TautulliStreamDataResponse{
					Result: "success",
					Data: tautulli.TautulliStreamDataInfo{
						SessionKey:            "test-session-123",
						TranscodeDecision:     "transcode",
						VideoDecision:         "transcode",
						AudioDecision:         "copy",
						Container:             "mkv",
						Bitrate:               20000,
						VideoCodec:            "hevc",
						VideoResolution:       "4k",
						VideoWidth:            3840,
						VideoHeight:           2160,
						AudioCodec:            "aac",
						AudioChannels:         6,
						StreamVideoCodec:      "h264",
						StreamVideoResolution: "1080p",
						StreamVideoWidth:      1920,
						StreamVideoHeight:     1080,
						StreamAudioChannels:   2,
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/stream-data?session_key=test-session-123", nil)
	w := httptest.NewRecorder()

	handler.TautulliStreamData(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}
