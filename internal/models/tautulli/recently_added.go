// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliRecentlyAdded represents the API response from get_recently_added endpoint
type TautulliRecentlyAdded struct {
	Response TautulliRecentlyAddedResponse `json:"response"`
}

type TautulliRecentlyAddedResponse struct {
	Result  string                    `json:"result"`
	Message *string                   `json:"message,omitempty"`
	Data    TautulliRecentlyAddedData `json:"data"`
}

type TautulliRecentlyAddedData struct {
	RecordsTotal  int                         `json:"records_total"`
	RecentlyAdded []TautulliRecentlyAddedItem `json:"recently_added"`
}

type TautulliRecentlyAddedItem struct {
	RatingKey            string `json:"rating_key"`
	ParentRatingKey      string `json:"parent_rating_key"`
	GrandparentRatingKey string `json:"grandparent_rating_key"`
	Title                string `json:"title"`
	ParentTitle          string `json:"parent_title"`
	GrandparentTitle     string `json:"grandparent_title"`
	MediaType            string `json:"media_type"`
	Year                 int    `json:"year"`
	Thumb                string `json:"thumb"`
	ParentThumb          string `json:"parent_thumb"`
	GrandparentThumb     string `json:"grandparent_thumb"`
	AddedAt              int64  `json:"added_at"`
	LibraryName          string `json:"library_name"`
	SectionID            int    `json:"section_id"`
}
