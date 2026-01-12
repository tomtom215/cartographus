// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliSearch represents the API response from Tautulli's search endpoint
type TautulliSearch struct {
	Response TautulliSearchResponse `json:"response"`
}

type TautulliSearchResponse struct {
	Result  string             `json:"result"`
	Message *string            `json:"message,omitempty"`
	Data    TautulliSearchData `json:"data"`
}

type TautulliSearchData struct {
	ResultsCount int                    `json:"results_count"`
	Results      []TautulliSearchResult `json:"results"`
}

type TautulliSearchResult struct {
	Type             string  `json:"type"`                        // "movie", "show", "artist", etc.
	RatingKey        string  `json:"rating_key"`                  // Plex rating key
	Title            string  `json:"title"`                       // Media title
	Year             int     `json:"year"`                        // Release year
	Thumb            string  `json:"thumb"`                       // Thumbnail URL
	Score            float64 `json:"score"`                       // Search relevance score
	Library          string  `json:"library"`                     // Library name
	LibraryID        int     `json:"library_id"`                  // Library ID
	MediaType        string  `json:"media_type"`                  // "movie", "episode", "track"
	Summary          string  `json:"summary"`                     // Content summary
	GrandparentTitle string  `json:"grandparent_title,omitempty"` // Show title for episodes
	ParentTitle      string  `json:"parent_title,omitempty"`      // Season title for episodes
}
