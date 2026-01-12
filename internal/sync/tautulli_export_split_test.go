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

func TestTautulliClient_ExportMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "export_metadata" {
			t.Errorf("Expected cmd=export_metadata, got %s", r.URL.Query().Get("cmd"))
		}

		// Verify section_id parameter
		if r.URL.Query().Get("section_id") != "1" {
			t.Errorf("Expected section_id=1, got %s", r.URL.Query().Get("section_id"))
		}

		// Verify file_format parameter
		if r.URL.Query().Get("file_format") != "csv" {
			t.Errorf("Expected file_format=csv, got %s", r.URL.Query().Get("file_format"))
		}

		response := tautulli.TautulliExportMetadata{
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
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	exportMetadata, err := client.ExportMetadata(context.Background(), 1, "collection", 0, "", "csv")
	if err != nil {
		t.Fatalf("ExportMetadata() error = %v", err)
	}
	if exportMetadata.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", exportMetadata.Response.Result)
	}
	if exportMetadata.Response.Data.ExportID != "export-12345" {
		t.Errorf("Expected ExportID='export-12345', got '%s'", exportMetadata.Response.Data.ExportID)
	}
	if exportMetadata.Response.Data.Status != "completed" {
		t.Errorf("Expected Status='completed', got '%s'", exportMetadata.Response.Data.Status)
	}
	if exportMetadata.Response.Data.FileFormat != "csv" {
		t.Errorf("Expected FileFormat='csv', got '%s'", exportMetadata.Response.Data.FileFormat)
	}
	if exportMetadata.Response.Data.RecordCount != 1500 {
		t.Errorf("Expected RecordCount=1500, got %d", exportMetadata.Response.Data.RecordCount)
	}
}

func TestTautulliClient_GetExportFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") != "get_export_fields" {
			t.Errorf("Expected cmd=get_export_fields, got %s", r.URL.Query().Get("cmd"))
		}

		// Verify media_type parameter
		if r.URL.Query().Get("media_type") != "movie" {
			t.Errorf("Expected media_type=movie, got %s", r.URL.Query().Get("media_type"))
		}

		response := tautulli.TautulliExportFields{
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
					{
						FieldName:    "rating",
						DisplayName:  "Rating",
						Description:  "User rating",
						FieldType:    "float",
						DefaultValue: "0.0",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{URL: server.URL, APIKey: "test-key"}
	client := NewTautulliClient(cfg)

	exportFields, err := client.GetExportFields(context.Background(), "movie")
	if err != nil {
		t.Fatalf("GetExportFields() error = %v", err)
	}
	if exportFields.Response.Result != "success" {
		t.Errorf("Expected result='success', got '%s'", exportFields.Response.Result)
	}
	if len(exportFields.Response.Data) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(exportFields.Response.Data))
	}
	if exportFields.Response.Data[0].FieldName != "title" {
		t.Errorf("Expected FieldName='title', got '%s'", exportFields.Response.Data[0].FieldName)
	}
	if exportFields.Response.Data[1].FieldType != "integer" {
		t.Errorf("Expected FieldType='integer', got '%s'", exportFields.Response.Data[1].FieldType)
	}
	if exportFields.Response.Data[2].DisplayName != "Rating" {
		t.Errorf("Expected DisplayName='Rating', got '%s'", exportFields.Response.Data[2].DisplayName)
	}
}
