// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliCollectionsTable represents the API response from Tautulli's get_collections_table endpoint
type TautulliCollectionsTable struct {
	Response TautulliCollectionsTableResponse `json:"response"`
}

type TautulliCollectionsTableResponse struct {
	Result  string                       `json:"result"`
	Message *string                      `json:"message,omitempty"`
	Data    TautulliCollectionsTableData `json:"data"`
}

type TautulliCollectionsTableData struct {
	RecordsTotal    int                           `json:"recordsTotal"`
	RecordsFiltered int                           `json:"recordsFiltered"`
	Draw            int                           `json:"draw"`
	Data            []TautulliCollectionsTableRow `json:"data"`
}

type TautulliCollectionsTableRow struct {
	RatingKey     string `json:"rating_key"`
	Title         string `json:"title"`
	SectionID     int    `json:"section_id"`
	SectionName   string `json:"section_name"`
	SectionType   string `json:"section_type"`
	MediaType     string `json:"media_type"`
	ContentRating string `json:"content_rating,omitempty"`
	Summary       string `json:"summary,omitempty"`
	Thumb         string `json:"thumb,omitempty"`
	AddedAt       int64  `json:"added_at"`
	UpdatedAt     int64  `json:"updated_at,omitempty"`
	ChildCount    int    `json:"child_count"` // Number of items in collection
	LastPlayed    int64  `json:"last_played,omitempty"`
	Plays         int    `json:"plays,omitempty"`
}
