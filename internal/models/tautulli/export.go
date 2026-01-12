// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliExportMetadata represents the API response from Tautulli's export_metadata endpoint
type TautulliExportMetadata struct {
	Response TautulliExportMetadataResponse `json:"response"`
}

type TautulliExportMetadataResponse struct {
	Result  string                     `json:"result"`
	Message *string                    `json:"message,omitempty"`
	Data    TautulliExportMetadataData `json:"data"`
}

type TautulliExportMetadataData struct {
	ExportID     string `json:"export_id,omitempty"`
	Status       string `json:"status"`
	FileFormat   string `json:"file_format,omitempty"`
	RecordCount  int    `json:"record_count,omitempty"`
	DownloadURL  string `json:"download_url,omitempty"`
	ErrorMessage string `json:"error,omitempty"`
}

// TautulliExportFields represents the API response from get_export_fields endpoint
type TautulliExportFields struct {
	Response TautulliExportFieldsResponse `json:"response"`
}

type TautulliExportFieldsResponse struct {
	Result  string                    `json:"result"`
	Message *string                   `json:"message,omitempty"`
	Data    []TautulliExportFieldItem `json:"data"`
}

type TautulliExportFieldItem struct {
	FieldName    string `json:"field_name"`
	DisplayName  string `json:"display_name"`
	Description  string `json:"description,omitempty"`
	FieldType    string `json:"field_type,omitempty"`
	DefaultValue string `json:"default,omitempty"`
}

// TautulliExportsTable represents the API response from get_exports_table endpoint
type TautulliExportsTable struct {
	Response TautulliExportsTableResponse `json:"response"`
}

type TautulliExportsTableResponse struct {
	Result  string                   `json:"result"`
	Message *string                  `json:"message,omitempty"`
	Data    TautulliExportsTableData `json:"data"`
}

type TautulliExportsTableData struct {
	RecordsTotal    int                       `json:"recordsTotal"`
	RecordsFiltered int                       `json:"recordsFiltered"`
	Draw            int                       `json:"draw"`
	Data            []TautulliExportsTableRow `json:"data"`
	FilterDuration  string                    `json:"filter_duration,omitempty"`
}

type TautulliExportsTableRow struct {
	ID              int    `json:"id"`                         // Export ID
	Timestamp       int64  `json:"timestamp"`                  // Export creation timestamp
	SectionID       int    `json:"section_id"`                 // Library section ID
	UserID          int    `json:"user_id,omitempty"`          // User ID (if user-specific export)
	RatingKey       int    `json:"rating_key,omitempty"`       // Content rating key (if content-specific)
	FileFormat      string `json:"file_format"`                // Export format (csv, json, xml, m3u)
	ExportType      string `json:"export_type"`                // Type of export (library, user, playlist, etc.)
	MediaType       string `json:"media_type,omitempty"`       // Media type (movie, show, artist)
	Title           string `json:"title,omitempty"`            // Export title/description
	CustomFields    string `json:"custom_fields,omitempty"`    // Custom fields included
	IndividualFiles int    `json:"individual_files,omitempty"` // Number of individual files
	FileSize        int64  `json:"file_size"`                  // Export file size in bytes
	Complete        int    `json:"complete"`                   // Completion status (0 or 1)
}

// TautulliDownloadExport represents the API response from download_export endpoint
// Note: This endpoint returns binary file data, not JSON
type TautulliDownloadExport struct {
	Response TautulliDownloadExportResponse `json:"response"`
}

type TautulliDownloadExportResponse struct {
	Result  string                     `json:"result"`
	Message *string                    `json:"message,omitempty"`
	Data    TautulliDownloadExportData `json:"data"`
}

type TautulliDownloadExportData struct {
	FileData    []byte `json:"file_data,omitempty"` // Binary file data
	FileName    string `json:"file_name"`           // Filename for download
	FileSize    int64  `json:"file_size"`           // File size in bytes
	ContentType string `json:"content_type"`        // MIME type
}

// TautulliDeleteExport represents the API response from delete_export endpoint
type TautulliDeleteExport struct {
	Response TautulliDeleteExportResponse `json:"response"`
}

type TautulliDeleteExportResponse struct {
	Result  string  `json:"result"`
	Message *string `json:"message,omitempty"`
}
