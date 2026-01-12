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

func TestTautulliExportMetadata_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		ExportMetadataFunc: func(ctx context.Context, sectionID int, exportType string, userID int, ratingKey string, fileFormat string) (*tautulli.TautulliExportMetadata, error) {
			return &tautulli.TautulliExportMetadata{
				Response: tautulli.TautulliExportMetadataResponse{
					Result: "success",
					Data: tautulli.TautulliExportMetadataData{
						ExportID:    "export-12345",
						Status:      "completed",
						FileFormat:  "csv",
						RecordCount: 1500,
						DownloadURL: "/exports/metadata-12345.csv",
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/export-metadata?section_id=1&file_format=csv", nil)
	w := httptest.NewRecorder()

	handler.TautulliExportMetadata(w, req)

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

func TestTautulliExportFields_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetExportFieldsFunc: func(ctx context.Context, mediaType string) (*tautulli.TautulliExportFields, error) {
			return &tautulli.TautulliExportFields{
				Response: tautulli.TautulliExportFieldsResponse{
					Result: "success",
					Data: []tautulli.TautulliExportFieldItem{
						{
							FieldName:    "title",
							DisplayName:  "Title",
							Description:  "Movie title",
							FieldType:    "string",
							DefaultValue: "",
						},
						{
							FieldName:    "year",
							DisplayName:  "Year",
							Description:  "Release year",
							FieldType:    "integer",
							DefaultValue: "0",
						},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/export-fields?media_type=movie", nil)
	w := httptest.NewRecorder()

	handler.TautulliExportFields(w, req)

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
