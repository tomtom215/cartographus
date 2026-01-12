// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
tautulli_server.go - Tautulli Server and Export Methods

This file provides methods for retrieving server information, system status,
and data export operations from Tautulli.

Server Information:
  - GetServerInfo(): Tautulli and PMS version information
  - GetServerStatus(): Server health and connectivity
  - GetServerID(): Plex server machine identifier
  - GetSettings(): Tautulli configuration settings

Sync and Activity:
  - GetSyncedItems(): Media synced to devices
  - GetActivity(): Current active streams

Export Operations:
  - ExportMetadata(): Export media metadata to JSON/CSV
  - GetExportsTable(): List available exports
  - DownloadExport(): Retrieve export file
  - DeleteExport(): Remove export file

Admin Operations:
  - RestartTautulli(): Restart Tautulli service
  - GetNotificationLog(): Notification delivery history

Export Formats:
The export system supports multiple formats:
  - json: Structured JSON export
  - csv: Comma-separated values
  - xlsx: Excel spreadsheet

Export types include library metadata, user statistics, and playback history.
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func (c *TautulliClient) GetServerInfo(ctx context.Context) (*tautulli.TautulliServerInfo, error) {
	var result tautulli.TautulliServerInfo
	err := c.makeRequest(ctx, "get_server_info", nil, &result)
	return &result, err
}

// GetSyncedItems retrieves synced media from Tautulli
func (c *TautulliClient) GetSyncedItems(ctx context.Context, machineID string, userID int) (*tautulli.TautulliSyncedItems, error) {
	params := url.Values{}
	if machineID != "" {
		params.Set("machine_id", machineID)
	}
	if userID > 0 {
		params.Set("user_id", fmt.Sprintf("%d", userID))
	}

	var result tautulli.TautulliSyncedItems
	err := c.makeRequest(ctx, "get_synced_items", params, &result)
	return &result, err
}

// TerminateSession terminates an active playback session on Tautulli
func (c *TautulliClient) TerminateSession(ctx context.Context, sessionID string, message string) (*tautulli.TautulliTerminateSession, error) {
	params := url.Values{}
	params.Set("session_id", sessionID)
	if message != "" {
		params.Set("message", message)
	}

	var result tautulli.TautulliTerminateSession
	err := c.makeRequest(ctx, "terminate_session", params, &result)
	return &result, err
}

func (c *TautulliClient) GetStreamData(ctx context.Context, rowID int, sessionKey string) (*tautulli.TautulliStreamData, error) {
	params := url.Values{}
	if rowID > 0 {
		params.Set("row_id", fmt.Sprintf("%d", rowID))
	} else if sessionKey != "" {
		params.Set("session_key", sessionKey)
	} else {
		return nil, fmt.Errorf("either row_id or session_key must be provided")
	}

	var result tautulli.TautulliStreamData
	err := c.makeRequest(ctx, "get_stream_data", params, &result)
	return &result, err
}

// ExportMetadata initiates metadata export for a library or content
func (c *TautulliClient) ExportMetadata(ctx context.Context, sectionID int, exportType string, userID int, ratingKey string, fileFormat string) (*tautulli.TautulliExportMetadata, error) {
	params := url.Values{}
	if sectionID > 0 {
		params.Set("section_id", fmt.Sprintf("%d", sectionID))
	}
	if exportType != "" {
		params.Set("export_type", exportType)
	}
	if userID > 0 {
		params.Set("user_id", fmt.Sprintf("%d", userID))
	}
	if ratingKey != "" {
		params.Set("rating_key", ratingKey)
	}
	if fileFormat != "" {
		params.Set("file_format", fileFormat)
	}

	var result tautulli.TautulliExportMetadata
	err := c.makeRequest(ctx, "export_metadata", params, &result)
	return &result, err
}

// GetExportFields retrieves available export fields for a media type
func (c *TautulliClient) GetExportFields(ctx context.Context, mediaType string) (*tautulli.TautulliExportFields, error) {
	params := url.Values{}
	if mediaType != "" {
		params.Set("media_type", mediaType)
	}

	var result tautulli.TautulliExportFields
	err := c.makeRequest(ctx, "get_export_fields", params, &result)
	return &result, err
}

func (c *TautulliClient) Search(ctx context.Context, query string, limit int) (*tautulli.TautulliSearch, error) {
	params := url.Values{}
	params.Set("query", query)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	var result tautulli.TautulliSearch
	err := c.makeRequest(ctx, "search", params, &result)
	return &result, err
}

// GetNewRatingKeys retrieves updated rating key mappings after Plex database changes
func (c *TautulliClient) GetNewRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliNewRatingKeys, error) {
	params := url.Values{}
	if ratingKey != "" {
		params.Set("rating_key", ratingKey)
	}

	var result tautulli.TautulliNewRatingKeys
	err := c.makeRequest(ctx, "get_new_rating_keys", params, &result)
	return &result, err
}

// GetOldRatingKeys retrieves historical rating key mappings
func (c *TautulliClient) GetOldRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliOldRatingKeys, error) {
	params := url.Values{}
	if ratingKey != "" {
		params.Set("rating_key", ratingKey)
	}

	var result tautulli.TautulliOldRatingKeys
	err := c.makeRequest(ctx, "get_old_rating_keys", params, &result)
	return &result, err
}

func (c *TautulliClient) GetServerFriendlyName(ctx context.Context) (*tautulli.TautulliServerFriendlyName, error) {
	var result tautulli.TautulliServerFriendlyName
	err := c.makeRequest(ctx, "get_server_friendly_name", nil, &result)
	return &result, err
}

// GetServerID retrieves the Plex Media Server unique identifier
func (c *TautulliClient) GetServerID(ctx context.Context) (*tautulli.TautulliServerID, error) {
	var result tautulli.TautulliServerID
	err := c.makeRequest(ctx, "get_server_id", nil, &result)
	return &result, err
}

// GetServerIdentity retrieves the Plex Media Server machine identity and platform information
func (c *TautulliClient) GetServerIdentity(ctx context.Context) (*tautulli.TautulliServerIdentity, error) {
	var result tautulli.TautulliServerIdentity
	err := c.makeRequest(ctx, "get_server_identity", nil, &result)
	return &result, err
}

// GetExportsTable retrieves paginated export history table from Tautulli
func (c *TautulliClient) GetExportsTable(ctx context.Context, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliExportsTable, error) {
	params := url.Values{}
	if orderColumn != "" {
		params.Set("order_column", orderColumn)
	}
	if orderDir != "" {
		params.Set("order_dir", orderDir)
	}
	if start > 0 {
		params.Set("start", fmt.Sprintf("%d", start))
	}
	if length > 0 {
		params.Set("length", fmt.Sprintf("%d", length))
	}
	if search != "" {
		params.Set("search", search)
	}

	var result tautulli.TautulliExportsTable
	err := c.makeRequest(ctx, "get_exports_table", params, &result)
	return &result, err
}

// DownloadExport retrieves export file data from Tautulli
func (c *TautulliClient) DownloadExport(ctx context.Context, exportID int) (*tautulli.TautulliDownloadExport, error) {
	params := url.Values{}
	params.Set("export_id", fmt.Sprintf("%d", exportID))

	var result tautulli.TautulliDownloadExport
	err := c.makeRequest(ctx, "download_export", params, &result)
	return &result, err
}

// DeleteExport deletes an export file from Tautulli
func (c *TautulliClient) DeleteExport(ctx context.Context, exportID int) (*tautulli.TautulliDeleteExport, error) {
	params := url.Values{}
	params.Set("export_id", fmt.Sprintf("%d", exportID))

	var result tautulli.TautulliDeleteExport
	err := c.makeRequest(ctx, "delete_export", params, &result)
	return &result, err
}

// GetTautulliInfo retrieves Tautulli version and platform information
func (c *TautulliClient) GetTautulliInfo(ctx context.Context) (*tautulli.TautulliTautulliInfo, error) {
	var result tautulli.TautulliTautulliInfo
	err := c.makeRequest(ctx, "get_tautulli_info", nil, &result)
	return &result, err
}

// GetServerPref retrieves a specific server preference from Tautulli
func (c *TautulliClient) GetServerPref(ctx context.Context, pref string) (*tautulli.TautulliServerPref, error) {
	params := url.Values{}
	params.Set("pref", pref)

	var result tautulli.TautulliServerPref
	err := c.makeRequest(ctx, "get_server_pref", params, &result)
	return &result, err
}

// GetServerList retrieves list of all Plex servers from Tautulli
func (c *TautulliClient) GetServerList(ctx context.Context) (*tautulli.TautulliServerList, error) {
	var result tautulli.TautulliServerList
	err := c.makeRequest(ctx, "get_server_list", nil, &result)
	return &result, err
}

// GetServersInfo retrieves detailed information for all Plex servers from Tautulli
func (c *TautulliClient) GetServersInfo(ctx context.Context) (*tautulli.TautulliServersInfo, error) {
	var result tautulli.TautulliServersInfo
	err := c.makeRequest(ctx, "get_servers_info", nil, &result)
	return &result, err
}

// GetPMSUpdate retrieves Plex Media Server update status from Tautulli
func (c *TautulliClient) GetPMSUpdate(ctx context.Context) (*tautulli.TautulliPMSUpdate, error) {
	var result tautulli.TautulliPMSUpdate
	err := c.makeRequest(ctx, "get_pms_update", nil, &result)
	return &result, err
}
