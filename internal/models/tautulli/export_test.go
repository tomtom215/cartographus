// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliExportMetadata_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"export_id": "export_12345",
				"status": "complete",
				"file_format": "csv",
				"record_count": 1500,
				"download_url": "/api/v2?cmd=download_export&export_id=export_12345"
			}
		}
	}`

	var export TautulliExportMetadata
	err := json.Unmarshal([]byte(jsonData), &export)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliExportMetadata: %v", err)
	}

	if export.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", export.Response.Result)
	}

	data := export.Response.Data
	if data.ExportID != "export_12345" {
		t.Errorf("Expected ExportID 'export_12345', got '%s'", data.ExportID)
	}
	if data.Status != "complete" {
		t.Errorf("Expected Status 'complete', got '%s'", data.Status)
	}
	if data.FileFormat != "csv" {
		t.Errorf("Expected FileFormat 'csv', got '%s'", data.FileFormat)
	}
	if data.RecordCount != 1500 {
		t.Errorf("Expected RecordCount 1500, got %d", data.RecordCount)
	}
	if data.DownloadURL != "/api/v2?cmd=download_export&export_id=export_12345" {
		t.Errorf("Expected DownloadURL mismatch, got '%s'", data.DownloadURL)
	}
}

func TestTautulliExportMetadata_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Export failed",
			"data": {
				"status": "failed",
				"error": "Unable to export library data"
			}
		}
	}`

	var export TautulliExportMetadata
	err := json.Unmarshal([]byte(jsonData), &export)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliExportMetadata: %v", err)
	}

	if export.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", export.Response.Result)
	}
	if export.Response.Message == nil || *export.Response.Message != "Export failed" {
		t.Error("Expected Message 'Export failed'")
	}
	if export.Response.Data.Status != "failed" {
		t.Errorf("Expected Status 'failed', got '%s'", export.Response.Data.Status)
	}
	if export.Response.Data.ErrorMessage != "Unable to export library data" {
		t.Errorf("Expected ErrorMessage mismatch, got '%s'", export.Response.Data.ErrorMessage)
	}
}

func TestTautulliExportMetadata_JSONRoundTrip(t *testing.T) {
	original := TautulliExportMetadata{
		Response: TautulliExportMetadataResponse{
			Result: "success",
			Data: TautulliExportMetadataData{
				ExportID:    "exp_001",
				Status:      "complete",
				FileFormat:  "json",
				RecordCount: 500,
				DownloadURL: "/download/exp_001",
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliExportMetadata: %v", err)
	}

	var decoded TautulliExportMetadata
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliExportMetadata: %v", err)
	}

	if decoded.Response.Data.ExportID != original.Response.Data.ExportID {
		t.Errorf("ExportID mismatch")
	}
	if decoded.Response.Data.RecordCount != original.Response.Data.RecordCount {
		t.Errorf("RecordCount mismatch")
	}
}

func TestTautulliExportFields_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": [
				{
					"field_name": "title",
					"display_name": "Title",
					"description": "The title of the media item",
					"field_type": "string",
					"default": ""
				},
				{
					"field_name": "year",
					"display_name": "Year",
					"description": "Release year",
					"field_type": "integer",
					"default": "0"
				},
				{
					"field_name": "duration",
					"display_name": "Duration",
					"description": "Duration in milliseconds",
					"field_type": "integer"
				}
			]
		}
	}`

	var fields TautulliExportFields
	err := json.Unmarshal([]byte(jsonData), &fields)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliExportFields: %v", err)
	}

	if fields.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", fields.Response.Result)
	}

	if len(fields.Response.Data) != 3 {
		t.Fatalf("Expected 3 fields, got %d", len(fields.Response.Data))
	}

	field := fields.Response.Data[0]
	if field.FieldName != "title" {
		t.Errorf("Expected FieldName 'title', got '%s'", field.FieldName)
	}
	if field.DisplayName != "Title" {
		t.Errorf("Expected DisplayName 'Title', got '%s'", field.DisplayName)
	}
	if field.Description != "The title of the media item" {
		t.Errorf("Expected Description mismatch, got '%s'", field.Description)
	}
	if field.FieldType != "string" {
		t.Errorf("Expected FieldType 'string', got '%s'", field.FieldType)
	}
}

func TestTautulliExportFields_EmptyData(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": []
		}
	}`

	var fields TautulliExportFields
	err := json.Unmarshal([]byte(jsonData), &fields)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliExportFields: %v", err)
	}

	if len(fields.Response.Data) != 0 {
		t.Errorf("Expected 0 fields, got %d", len(fields.Response.Data))
	}
}

func TestTautulliExportFields_JSONRoundTrip(t *testing.T) {
	original := TautulliExportFields{
		Response: TautulliExportFieldsResponse{
			Result: "success",
			Data: []TautulliExportFieldItem{
				{
					FieldName:    "rating",
					DisplayName:  "Rating",
					Description:  "Content rating",
					FieldType:    "float",
					DefaultValue: "0.0",
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliExportFields: %v", err)
	}

	var decoded TautulliExportFields
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliExportFields: %v", err)
	}

	if len(decoded.Response.Data) != 1 {
		t.Errorf("Expected 1 field, got %d", len(decoded.Response.Data))
	}
}

func TestTautulliExportsTable_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsTotal": 50,
				"recordsFiltered": 25,
				"draw": 1,
				"data": [
					{
						"id": 1,
						"timestamp": 1700000000,
						"section_id": 1,
						"user_id": 0,
						"rating_key": 12345,
						"file_format": "csv",
						"export_type": "library",
						"media_type": "movie",
						"title": "Movies Library Export",
						"custom_fields": "title,year,duration",
						"individual_files": 1,
						"file_size": 1024000,
						"complete": 1
					}
				],
				"filter_duration": "0.05s"
			}
		}
	}`

	var table TautulliExportsTable
	err := json.Unmarshal([]byte(jsonData), &table)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliExportsTable: %v", err)
	}

	if table.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", table.Response.Result)
	}

	data := table.Response.Data
	if data.RecordsTotal != 50 {
		t.Errorf("Expected RecordsTotal 50, got %d", data.RecordsTotal)
	}
	if data.RecordsFiltered != 25 {
		t.Errorf("Expected RecordsFiltered 25, got %d", data.RecordsFiltered)
	}
	if data.Draw != 1 {
		t.Errorf("Expected Draw 1, got %d", data.Draw)
	}
	if data.FilterDuration != "0.05s" {
		t.Errorf("Expected FilterDuration '0.05s', got '%s'", data.FilterDuration)
	}

	if len(data.Data) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(data.Data))
	}

	row := data.Data[0]
	if row.ID != 1 {
		t.Errorf("Expected ID 1, got %d", row.ID)
	}
	if row.Timestamp != 1700000000 {
		t.Errorf("Expected Timestamp 1700000000, got %d", row.Timestamp)
	}
	if row.SectionID != 1 {
		t.Errorf("Expected SectionID 1, got %d", row.SectionID)
	}
	if row.RatingKey != 12345 {
		t.Errorf("Expected RatingKey 12345, got %d", row.RatingKey)
	}
	if row.FileFormat != "csv" {
		t.Errorf("Expected FileFormat 'csv', got '%s'", row.FileFormat)
	}
	if row.ExportType != "library" {
		t.Errorf("Expected ExportType 'library', got '%s'", row.ExportType)
	}
	if row.MediaType != "movie" {
		t.Errorf("Expected MediaType 'movie', got '%s'", row.MediaType)
	}
	if row.Title != "Movies Library Export" {
		t.Errorf("Expected Title 'Movies Library Export', got '%s'", row.Title)
	}
	if row.CustomFields != "title,year,duration" {
		t.Errorf("Expected CustomFields 'title,year,duration', got '%s'", row.CustomFields)
	}
	if row.IndividualFiles != 1 {
		t.Errorf("Expected IndividualFiles 1, got %d", row.IndividualFiles)
	}
	if row.FileSize != 1024000 {
		t.Errorf("Expected FileSize 1024000, got %d", row.FileSize)
	}
	if row.Complete != 1 {
		t.Errorf("Expected Complete 1, got %d", row.Complete)
	}
}

func TestTautulliExportsTable_EmptyData(t *testing.T) {
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

	var table TautulliExportsTable
	err := json.Unmarshal([]byte(jsonData), &table)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliExportsTable: %v", err)
	}

	if len(table.Response.Data.Data) != 0 {
		t.Errorf("Expected 0 rows, got %d", len(table.Response.Data.Data))
	}
}

func TestTautulliExportsTable_MultipleFormats(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"recordsTotal": 4,
				"recordsFiltered": 4,
				"draw": 1,
				"data": [
					{"id": 1, "file_format": "csv", "export_type": "library", "file_size": 1000, "complete": 1, "timestamp": 1700000000, "section_id": 1},
					{"id": 2, "file_format": "json", "export_type": "user", "file_size": 2000, "complete": 1, "timestamp": 1700000001, "section_id": 1},
					{"id": 3, "file_format": "xml", "export_type": "playlist", "file_size": 1500, "complete": 1, "timestamp": 1700000002, "section_id": 2},
					{"id": 4, "file_format": "m3u", "export_type": "library", "file_size": 500, "complete": 0, "timestamp": 1700000003, "section_id": 2}
				]
			}
		}
	}`

	var table TautulliExportsTable
	err := json.Unmarshal([]byte(jsonData), &table)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliExportsTable: %v", err)
	}

	if len(table.Response.Data.Data) != 4 {
		t.Fatalf("Expected 4 rows, got %d", len(table.Response.Data.Data))
	}

	expectedFormats := []string{"csv", "json", "xml", "m3u"}
	for i, row := range table.Response.Data.Data {
		if row.FileFormat != expectedFormats[i] {
			t.Errorf("Expected FileFormat '%s', got '%s'", expectedFormats[i], row.FileFormat)
		}
	}
}

func TestTautulliExportsTable_JSONRoundTrip(t *testing.T) {
	original := TautulliExportsTable{
		Response: TautulliExportsTableResponse{
			Result: "success",
			Data: TautulliExportsTableData{
				RecordsTotal:    10,
				RecordsFiltered: 5,
				Draw:            2,
				FilterDuration:  "0.1s",
				Data: []TautulliExportsTableRow{
					{
						ID:         1,
						Timestamp:  1700000000,
						SectionID:  1,
						FileFormat: "csv",
						ExportType: "library",
						FileSize:   1024,
						Complete:   1,
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliExportsTable: %v", err)
	}

	var decoded TautulliExportsTable
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliExportsTable: %v", err)
	}

	if decoded.Response.Data.RecordsTotal != original.Response.Data.RecordsTotal {
		t.Errorf("RecordsTotal mismatch")
	}
}

func TestTautulliDownloadExport_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"file_name": "movies_export_2024.csv",
				"file_size": 524288,
				"content_type": "text/csv"
			}
		}
	}`

	var download TautulliDownloadExport
	err := json.Unmarshal([]byte(jsonData), &download)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliDownloadExport: %v", err)
	}

	if download.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", download.Response.Result)
	}

	data := download.Response.Data
	if data.FileName != "movies_export_2024.csv" {
		t.Errorf("Expected FileName 'movies_export_2024.csv', got '%s'", data.FileName)
	}
	if data.FileSize != 524288 {
		t.Errorf("Expected FileSize 524288, got %d", data.FileSize)
	}
	if data.ContentType != "text/csv" {
		t.Errorf("Expected ContentType 'text/csv', got '%s'", data.ContentType)
	}
}

func TestTautulliDownloadExport_WithFileData(t *testing.T) {
	// Note: In practice, file_data would be base64 encoded or handled differently
	// This test verifies the structure handles the field
	original := TautulliDownloadExport{
		Response: TautulliDownloadExportResponse{
			Result: "success",
			Data: TautulliDownloadExportData{
				FileData:    []byte("title,year,rating\nInception,2010,8.8"),
				FileName:    "movies.csv",
				FileSize:    35,
				ContentType: "text/csv",
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliDownloadExport: %v", err)
	}

	var decoded TautulliDownloadExport
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliDownloadExport: %v", err)
	}

	if decoded.Response.Data.FileName != original.Response.Data.FileName {
		t.Errorf("FileName mismatch")
	}
}

func TestTautulliDownloadExport_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Export not found",
			"data": {
				"file_name": "",
				"file_size": 0,
				"content_type": ""
			}
		}
	}`

	var download TautulliDownloadExport
	err := json.Unmarshal([]byte(jsonData), &download)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliDownloadExport: %v", err)
	}

	if download.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", download.Response.Result)
	}
	if download.Response.Message == nil || *download.Response.Message != "Export not found" {
		t.Error("Expected Message 'Export not found'")
	}
}

func TestTautulliDownloadExport_ContentTypes(t *testing.T) {
	testCases := []struct {
		name        string
		contentType string
	}{
		{"CSV", "text/csv"},
		{"JSON", "application/json"},
		{"XML", "application/xml"},
		{"M3U", "audio/x-mpegurl"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := TautulliDownloadExport{
				Response: TautulliDownloadExportResponse{
					Result: "success",
					Data: TautulliDownloadExportData{
						FileName:    "export." + tc.name,
						FileSize:    1024,
						ContentType: tc.contentType,
					},
				},
			}

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded TautulliDownloadExport
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Response.Data.ContentType != tc.contentType {
				t.Errorf("Expected ContentType '%s', got '%s'", tc.contentType, decoded.Response.Data.ContentType)
			}
		})
	}
}

func TestTautulliDeleteExport_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"message": "Export deleted successfully"
		}
	}`

	var del TautulliDeleteExport
	err := json.Unmarshal([]byte(jsonData), &del)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliDeleteExport: %v", err)
	}

	if del.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", del.Response.Result)
	}
	if del.Response.Message == nil || *del.Response.Message != "Export deleted successfully" {
		t.Error("Expected Message 'Export deleted successfully'")
	}
}

func TestTautulliDeleteExport_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Export not found or already deleted"
		}
	}`

	var del TautulliDeleteExport
	err := json.Unmarshal([]byte(jsonData), &del)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliDeleteExport: %v", err)
	}

	if del.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", del.Response.Result)
	}
	if del.Response.Message == nil || *del.Response.Message != "Export not found or already deleted" {
		t.Error("Expected Message 'Export not found or already deleted'")
	}
}

func TestTautulliDeleteExport_NullMessage(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"message": null
		}
	}`

	var del TautulliDeleteExport
	err := json.Unmarshal([]byte(jsonData), &del)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliDeleteExport: %v", err)
	}

	if del.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", del.Response.Result)
	}
	if del.Response.Message != nil {
		t.Errorf("Expected Message nil, got '%s'", *del.Response.Message)
	}
}

func TestTautulliDeleteExport_JSONRoundTrip(t *testing.T) {
	msg := "Deleted"
	original := TautulliDeleteExport{
		Response: TautulliDeleteExportResponse{
			Result:  "success",
			Message: &msg,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliDeleteExport: %v", err)
	}

	var decoded TautulliDeleteExport
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliDeleteExport: %v", err)
	}

	if decoded.Response.Result != original.Response.Result {
		t.Errorf("Result mismatch")
	}
	if decoded.Response.Message == nil || *decoded.Response.Message != *original.Response.Message {
		t.Errorf("Message mismatch")
	}
}
