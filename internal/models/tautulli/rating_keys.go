// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliNewRatingKeys represents the API response from get_new_rating_keys endpoint
type TautulliNewRatingKeys struct {
	Response TautulliNewRatingKeysResponse `json:"response"`
}

type TautulliNewRatingKeysResponse struct {
	Result  string                    `json:"result"`
	Message *string                   `json:"message,omitempty"`
	Data    TautulliNewRatingKeysData `json:"data"`
}

type TautulliNewRatingKeysData struct {
	RatingKeys []TautulliRatingKeyMapping `json:"rating_keys"`
}

type TautulliRatingKeyMapping struct {
	OldRatingKey string `json:"old_rating_key"` // Previous rating key
	NewRatingKey string `json:"new_rating_key"` // Current rating key
	Title        string `json:"title"`          // Media title
	MediaType    string `json:"media_type"`     // "movie", "show", etc.
	UpdatedAt    int64  `json:"updated_at"`     // Unix timestamp of mapping update
}

// TautulliOldRatingKeys represents the API response from get_old_rating_keys endpoint
type TautulliOldRatingKeys struct {
	Response TautulliOldRatingKeysResponse `json:"response"`
}

type TautulliOldRatingKeysResponse struct {
	Result  string                    `json:"result"`
	Message *string                   `json:"message,omitempty"`
	Data    TautulliOldRatingKeysData `json:"data"`
}

type TautulliOldRatingKeysData struct {
	RatingKeys []TautulliRatingKeyMapping `json:"rating_keys"` // Reuses TautulliRatingKeyMapping
}
