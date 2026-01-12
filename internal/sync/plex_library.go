// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
plex_library.go - Plex Library Content Methods

This file provides methods for accessing Plex Media Server library content
including sections, media items, playlists, and search functionality.

Library Structure:
  - Sections: Top-level library containers (Movies, TV Shows, Music, Photos)
  - Content: Media items within a section (with pagination)
  - Metadata: Detailed information for individual items
  - On Deck: Continue watching recommendations

API Methods:
  - GetLibrarySections(): List all library sections
  - GetLibrarySectionContent(): Paginated content listing
  - GetLibrarySectionRecentlyAdded(): Recent additions to library
  - GetOnDeck(): Continue watching recommendations
  - GetMetadata(): Detailed item metadata by rating key
  - GetPlaylists(): User playlists
  - Search(): Search within a library section

Pagination:
Library content endpoints support pagination via:
  - X-Plex-Container-Start: Starting index
  - X-Plex-Container-Size: Items per page

Content Types:
  - movie: Feature films
  - show: TV series
  - episode: TV episodes
  - track: Music tracks
  - artist: Music artists
  - album: Music albums
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetLibrarySections retrieves all library sections from Plex Media Server
//
// This endpoint returns all configured library sections (Movies, TV Shows, Music, Photos):
//   - Section metadata (name, type, UUID)
//   - Scanner and agent configuration
//   - Storage locations
//   - Last scan timestamps
//
// Endpoint: GET /library/sections
func (c *PlexClient) GetLibrarySections(ctx context.Context) (*models.PlexLibrarySectionsResponse, error) {
	var sectionsResp models.PlexLibrarySectionsResponse
	if err := c.doJSONRequest(ctx, "/library/sections", &sectionsResp); err != nil {
		return nil, err
	}
	return &sectionsResp, nil
}

// GetLibrarySectionContent retrieves all content from a library section
//
// This endpoint returns all media items in a specific library section with pagination:
//   - Full metadata (titles, ratings, thumbnails)
//   - Playback status (view count, last viewed)
//   - Media versions and file information
//
// Endpoint: GET /library/sections/{sectionKey}/all
func (c *PlexClient) GetLibrarySectionContent(ctx context.Context, sectionKey string, start, size *int) (*models.PlexLibrarySectionContentResponse, error) {
	query := url.Values{}
	if start != nil {
		query.Add("X-Plex-Container-Start", fmt.Sprintf("%d", *start))
	}
	if size != nil {
		query.Add("X-Plex-Container-Size", fmt.Sprintf("%d", *size))
	}

	var contentResp models.PlexLibrarySectionContentResponse
	if err := c.doJSONRequestWithQuery(ctx, "/library/sections/"+sectionKey+"/all", query, &contentResp); err != nil {
		return nil, err
	}
	return &contentResp, nil
}

// GetLibrarySectionRecentlyAdded retrieves recently added content from a library section
//
// This endpoint returns the most recently added items in a specific library section:
//   - Chronologically ordered by addition date
//   - Full metadata (titles, ratings, thumbnails)
//   - Useful for "New Arrivals" features
//
// Endpoint: GET /library/sections/{sectionKey}/recentlyAdded
func (c *PlexClient) GetLibrarySectionRecentlyAdded(ctx context.Context, sectionKey string, size *int) (*models.PlexLibrarySectionContentResponse, error) {
	query := url.Values{}
	if size != nil {
		query.Add("X-Plex-Container-Size", fmt.Sprintf("%d", *size))
	}

	var contentResp models.PlexLibrarySectionContentResponse
	if err := c.doJSONRequestWithQuery(ctx, "/library/sections/"+sectionKey+"/recentlyAdded", query, &contentResp); err != nil {
		return nil, err
	}
	return &contentResp, nil
}

// GetOnDeck retrieves on-deck content from Plex Media Server
//
// Endpoint: GET /library/onDeck
func (c *PlexClient) GetOnDeck(ctx context.Context) (*models.PlexOnDeckResponse, error) {
	var onDeckResp models.PlexOnDeckResponse
	if err := c.doJSONRequest(ctx, "/library/onDeck", &onDeckResp); err != nil {
		return nil, err
	}
	return &onDeckResp, nil
}

// GetMetadata retrieves detailed metadata for a specific media item
//
// Endpoint: GET /library/metadata/{ratingKey}
func (c *PlexClient) GetMetadata(ctx context.Context, ratingKey string) (*models.PlexMetadataResponse, error) {
	var metadataResp models.PlexMetadataResponse
	if err := c.doJSONRequest(ctx, "/library/metadata/"+ratingKey, &metadataResp); err != nil {
		return nil, err
	}
	return &metadataResp, nil
}

// GetPlaylists retrieves playlists from Plex Media Server
//
// Endpoint: GET /playlists
func (c *PlexClient) GetPlaylists(ctx context.Context) (*models.PlexPlaylistsResponse, error) {
	var playlistsResp models.PlexPlaylistsResponse
	if err := c.doJSONRequest(ctx, "/playlists", &playlistsResp); err != nil {
		return nil, err
	}
	return &playlistsResp, nil
}

// Search performs a search within a library section
//
// Endpoint: GET /library/sections/{key}/search
func (c *PlexClient) Search(ctx context.Context, sectionKey, searchQuery string, mediaType *int) (*models.PlexSearchResponse, error) {
	query := url.Values{}
	query.Add("query", searchQuery)
	if mediaType != nil {
		query.Add("type", fmt.Sprintf("%d", *mediaType))
	}

	var searchResp models.PlexSearchResponse
	if err := c.doJSONRequestWithQuery(ctx, "/library/sections/"+sectionKey+"/search", query, &searchResp); err != nil {
		return nil, err
	}
	return &searchResp, nil
}
