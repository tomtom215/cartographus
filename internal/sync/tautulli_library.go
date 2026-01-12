// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
tautulli_library.go - Tautulli Library and Metadata Methods

This file provides methods for retrieving library information and media
metadata from Tautulli.

Library Methods:
  - GetLibraries(): List all library sections
  - GetLibrary(): Single library details
  - GetLibraryMediaInfo(): Media items in a library (paginated)
  - GetLibraryUserStats(): Per-user library usage

Metadata Methods:
  - GetMetadata(): Detailed media metadata by rating key
  - GetChildrenMetadata(): Child items (episodes, tracks)
  - GetRecentlyAdded(): Recently added media items
  - GetCollectionsTable(): Library collections
  - GetPlaylistsTable(): User playlists

Media Types:
  - movie: Feature films
  - show: TV series
  - season: TV seasons
  - episode: TV episodes
  - artist: Music artists
  - album: Music albums
  - track: Music tracks

Validator Functions:
This file includes response validators for library types that check
the API response status and provide meaningful error messages when
requests fail.
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// Validator functions for library-related response types
func validateMetadataResponse(m *tautulli.TautulliMetadata) error {
	if m.Response.Result != "success" {
		msg := "unknown error"
		if m.Response.Message != nil {
			msg = *m.Response.Message
		}
		return fmt.Errorf("get_metadata request failed: %s", msg)
	}
	return nil
}

func validateLibraryUserStatsResponse(s *tautulli.TautulliLibraryUserStats) error {
	if s.Response.Result != "success" {
		msg := "unknown error"
		if s.Response.Message != nil {
			msg = *s.Response.Message
		}
		return fmt.Errorf("get_library_user_stats request failed: %s", msg)
	}
	return nil
}

func validateRecentlyAddedResponse(r *tautulli.TautulliRecentlyAdded) error {
	if r.Response.Result != "success" {
		msg := "unknown error"
		if r.Response.Message != nil {
			msg = *r.Response.Message
		}
		return fmt.Errorf("get_recently_added request failed: %s", msg)
	}
	return nil
}

func validateLibrariesResponse(l *tautulli.TautulliLibraries) error {
	if l.Response.Result != "success" {
		msg := "unknown error"
		if l.Response.Message != nil {
			msg = *l.Response.Message
		}
		return fmt.Errorf("get_libraries request failed: %s", msg)
	}
	return nil
}

func validateLibraryResponse(l *tautulli.TautulliLibrary) error {
	if l.Response.Result != "success" {
		msg := "unknown error"
		if l.Response.Message != nil {
			msg = *l.Response.Message
		}
		return fmt.Errorf("get_library request failed: %s", msg)
	}
	return nil
}

func validateLibrariesTableResponse(l *tautulli.TautulliLibrariesTable) error {
	if l.Response.Result != "success" {
		msg := "unknown error"
		if l.Response.Message != nil {
			msg = *l.Response.Message
		}
		return fmt.Errorf("get_libraries_table request failed: %s", msg)
	}
	return nil
}

func validateLibraryMediaInfoResponse(m *tautulli.TautulliLibraryMediaInfo) error {
	if m.Response.Result != "success" {
		msg := "unknown error"
		if m.Response.Message != nil {
			msg = *m.Response.Message
		}
		return fmt.Errorf("get_library_media_info request failed: %s", msg)
	}
	return nil
}

func validateLibraryWatchTimeStatsResponse(s *tautulli.TautulliLibraryWatchTimeStats) error {
	if s.Response.Result != "success" {
		msg := "unknown error"
		if s.Response.Message != nil {
			msg = *s.Response.Message
		}
		return fmt.Errorf("get_library_watch_time_stats request failed: %s", msg)
	}
	return nil
}

func validateChildrenMetadataResponse(m *tautulli.TautulliChildrenMetadata) error {
	if m.Response.Result != "success" {
		msg := "unknown error"
		if m.Response.Message != nil {
			msg = *m.Response.Message
		}
		return fmt.Errorf("get_children_metadata request failed: %s", msg)
	}
	return nil
}

func validateLibraryNamesResponse(l *tautulli.TautulliLibraryNames) error {
	if l.Response.Result != "success" {
		msg := "unknown error"
		if l.Response.Message != nil {
			msg = *l.Response.Message
		}
		return fmt.Errorf("get_library_names request failed: %s", msg)
	}
	return nil
}

func validateCollectionsTableResponse(c *tautulli.TautulliCollectionsTable) error {
	if c.Response.Result != "success" {
		msg := "unknown error"
		if c.Response.Message != nil {
			msg = *c.Response.Message
		}
		return fmt.Errorf("get_collections_table request failed: %s", msg)
	}
	return nil
}

func validatePlaylistsTableResponse(p *tautulli.TautulliPlaylistsTable) error {
	if p.Response.Result != "success" {
		msg := "unknown error"
		if p.Response.Message != nil {
			msg = *p.Response.Message
		}
		return fmt.Errorf("get_playlists_table request failed: %s", msg)
	}
	return nil
}

func (c *TautulliClient) GetMetadata(ctx context.Context, ratingKey string) (*tautulli.TautulliMetadata, error) {
	params := url.Values{}
	params.Set("rating_key", ratingKey)
	return callTautulliAPI(ctx, c, "get_metadata", params, validateMetadataResponse)
}

func (c *TautulliClient) GetLibraryUserStats(ctx context.Context, sectionID int, grouping int) (*tautulli.TautulliLibraryUserStats, error) {
	params := url.Values{}
	params.Set("section_id", fmt.Sprintf("%d", sectionID))
	if grouping >= 0 {
		params.Set("grouping", fmt.Sprintf("%d", grouping))
	}
	return callTautulliAPI(ctx, c, "get_library_user_stats", params, validateLibraryUserStatsResponse)
}

// GetRecentlyAdded retrieves recently added content from Tautulli
func (c *TautulliClient) GetRecentlyAdded(ctx context.Context, count int, start int, mediaType string, sectionID int) (*tautulli.TautulliRecentlyAdded, error) {
	params := url.Values{}
	params.Set("count", fmt.Sprintf("%d", count))
	if start > 0 {
		params.Set("start", fmt.Sprintf("%d", start))
	}
	if mediaType != "" {
		params.Set("media_type", mediaType)
	}
	if sectionID > 0 {
		params.Set("section_id", fmt.Sprintf("%d", sectionID))
	}
	return callTautulliAPI(ctx, c, "get_recently_added", params, validateRecentlyAddedResponse)
}

// GetLibraries retrieves all library sections from Tautulli
func (c *TautulliClient) GetLibraries(ctx context.Context) (*tautulli.TautulliLibraries, error) {
	params := url.Values{}
	return callTautulliAPI(ctx, c, "get_libraries", params, validateLibrariesResponse)
}

// GetLibrary retrieves specific library details from Tautulli
func (c *TautulliClient) GetLibrary(ctx context.Context, sectionID int) (*tautulli.TautulliLibrary, error) {
	params := url.Values{}
	params.Set("section_id", fmt.Sprintf("%d", sectionID))
	return callTautulliAPI(ctx, c, "get_library", params, validateLibraryResponse)
}

func (c *TautulliClient) GetLibrariesTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliLibrariesTable, error) {
	params := url.Values{}
	if grouping >= 0 {
		params.Set("grouping", fmt.Sprintf("%d", grouping))
	}
	if orderColumn != "" {
		params.Set("order_column", orderColumn)
	}
	if orderDir != "" {
		params.Set("order_dir", orderDir)
	}
	if start >= 0 {
		params.Set("start", fmt.Sprintf("%d", start))
	}
	if length > 0 {
		params.Set("length", fmt.Sprintf("%d", length))
	}
	if search != "" {
		params.Set("search", search)
	}
	return callTautulliAPI(ctx, c, "get_libraries_table", params, validateLibrariesTableResponse)
}

// GetLibraryMediaInfo retrieves media information for a specific library
func (c *TautulliClient) GetLibraryMediaInfo(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int) (*tautulli.TautulliLibraryMediaInfo, error) {
	params := url.Values{}
	if sectionID > 0 {
		params.Set("section_id", fmt.Sprintf("%d", sectionID))
	}
	if orderColumn != "" {
		params.Set("order_column", orderColumn)
	}
	if orderDir != "" {
		params.Set("order_dir", orderDir)
	}
	if start >= 0 {
		params.Set("start", fmt.Sprintf("%d", start))
	}
	if length > 0 {
		params.Set("length", fmt.Sprintf("%d", length))
	}
	return callTautulliAPI(ctx, c, "get_library_media_info", params, validateLibraryMediaInfoResponse)
}

// GetLibraryWatchTimeStats retrieves watch time statistics for a specific library
func (c *TautulliClient) GetLibraryWatchTimeStats(ctx context.Context, sectionID int, grouping int, queryDays string) (*tautulli.TautulliLibraryWatchTimeStats, error) {
	params := url.Values{}
	if sectionID > 0 {
		params.Set("section_id", fmt.Sprintf("%d", sectionID))
	}
	if grouping >= 0 {
		params.Set("grouping", fmt.Sprintf("%d", grouping))
	}
	if queryDays != "" {
		params.Set("query_days", queryDays)
	}
	return callTautulliAPI(ctx, c, "get_library_watch_time_stats", params, validateLibraryWatchTimeStatsResponse)
}

// GetChildrenMetadata retrieves metadata for child items (e.g., episodes of a season)
func (c *TautulliClient) GetChildrenMetadata(ctx context.Context, ratingKey string, mediaType string) (*tautulli.TautulliChildrenMetadata, error) {
	params := url.Values{}
	if ratingKey != "" {
		params.Set("rating_key", ratingKey)
	}
	if mediaType != "" {
		params.Set("media_type", mediaType)
	}
	return callTautulliAPI(ctx, c, "get_children_metadata", params, validateChildrenMetadataResponse)
}

func (c *TautulliClient) GetLibraryNames(ctx context.Context) (*tautulli.TautulliLibraryNames, error) {
	params := url.Values{}
	return callTautulliAPI(ctx, c, "get_library_names", params, validateLibraryNamesResponse)
}

func (c *TautulliClient) GetCollectionsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliCollectionsTable, error) {
	params := url.Values{}
	if sectionID > 0 {
		params.Set("section_id", fmt.Sprintf("%d", sectionID))
	}
	if orderColumn != "" {
		params.Set("order_column", orderColumn)
	}
	if orderDir != "" {
		params.Set("order_dir", orderDir)
	}
	if start >= 0 {
		params.Set("start", fmt.Sprintf("%d", start))
	}
	if length > 0 {
		params.Set("length", fmt.Sprintf("%d", length))
	}
	if search != "" {
		params.Set("search", search)
	}
	return callTautulliAPI(ctx, c, "get_collections_table", params, validateCollectionsTableResponse)
}

// GetPlaylistsTable retrieves paginated playlist data with sorting and filtering
func (c *TautulliClient) GetPlaylistsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliPlaylistsTable, error) {
	params := url.Values{}
	if sectionID > 0 {
		params.Set("section_id", fmt.Sprintf("%d", sectionID))
	}
	if orderColumn != "" {
		params.Set("order_column", orderColumn)
	}
	if orderDir != "" {
		params.Set("order_dir", orderDir)
	}
	if start >= 0 {
		params.Set("start", fmt.Sprintf("%d", start))
	}
	if length > 0 {
		params.Set("length", fmt.Sprintf("%d", length))
	}
	if search != "" {
		params.Set("search", search)
	}
	return callTautulliAPI(ctx, c, "get_playlists_table", params, validatePlaylistsTableResponse)
}
