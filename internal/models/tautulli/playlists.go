// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliPlaylistsTable represents the API response from Tautulli's get_playlists_table endpoint
type TautulliPlaylistsTable struct {
	Response TautulliPlaylistsTableResponse `json:"response"`
}

type TautulliPlaylistsTableResponse struct {
	Result  string                     `json:"result"`
	Message *string                    `json:"message,omitempty"`
	Data    TautulliPlaylistsTableData `json:"data"`
}

type TautulliPlaylistsTableData struct {
	RecordsTotal    int                         `json:"recordsTotal"`
	RecordsFiltered int                         `json:"recordsFiltered"`
	Draw            int                         `json:"draw"`
	Data            []TautulliPlaylistsTableRow `json:"data"`
}

type TautulliPlaylistsTableRow struct {
	RatingKey     string `json:"rating_key"`
	Title         string `json:"title"`
	PlaylistType  string `json:"playlist_type"` // audio, video, photo
	Composite     string `json:"composite,omitempty"`
	Summary       string `json:"summary,omitempty"`
	Smart         int    `json:"smart"`    // 0 or 1 (boolean)
	DurationTotal int    `json:"duration"` // Total duration in seconds
	AddedAt       int64  `json:"added_at"`
	UpdatedAt     int64  `json:"updated_at,omitempty"`
	ChildCount    int    `json:"child_count"` // Number of items in playlist
	LastPlayed    int64  `json:"last_played,omitempty"`
	Plays         int    `json:"plays,omitempty"`
}
