// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliPlaysBySourceResolution represents the API response from get_plays_by_source_resolution endpoint
type TautulliPlaysBySourceResolution struct {
	Response TautulliPlaysBySourceResolutionResponse `json:"response"`
}

type TautulliPlaysBySourceResolutionResponse struct {
	Result  string                              `json:"result"`
	Message *string                             `json:"message,omitempty"`
	Data    TautulliPlaysBySourceResolutionData `json:"data"`
}

type TautulliPlaysBySourceResolutionData struct {
	Categories []string                    `json:"categories"` // Array of resolution strings
	Series     []TautulliPlaysByDateSeries `json:"series"`     // Reuses existing series structure
}

// TautulliPlaysByStreamResolution represents the API response from get_plays_by_stream_resolution endpoint
type TautulliPlaysByStreamResolution struct {
	Response TautulliPlaysByStreamResolutionResponse `json:"response"`
}

type TautulliPlaysByStreamResolutionResponse struct {
	Result  string                              `json:"result"`
	Message *string                             `json:"message,omitempty"`
	Data    TautulliPlaysByStreamResolutionData `json:"data"`
}

type TautulliPlaysByStreamResolutionData struct {
	Categories []string                    `json:"categories"` // Array of resolution strings
	Series     []TautulliPlaysByDateSeries `json:"series"`     // Reuses existing series structure
}
