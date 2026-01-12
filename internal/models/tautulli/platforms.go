// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliPlaysByTop10Platforms represents the API response from get_plays_by_top_10_platforms endpoint
type TautulliPlaysByTop10Platforms struct {
	Response TautulliPlaysByTop10PlatformsResponse `json:"response"`
}

type TautulliPlaysByTop10PlatformsResponse struct {
	Result  string                            `json:"result"`
	Message *string                           `json:"message,omitempty"`
	Data    TautulliPlaysByTop10PlatformsData `json:"data"`
}

type TautulliPlaysByTop10PlatformsData struct {
	Categories []string                    `json:"categories"` // Array of platform names
	Series     []TautulliPlaysByDateSeries `json:"series"`     // Reuses existing series structure
}

// TautulliPlaysByTop10Users represents the API response from get_plays_by_top_10_users endpoint
type TautulliPlaysByTop10Users struct {
	Response TautulliPlaysByTop10UsersResponse `json:"response"`
}

type TautulliPlaysByTop10UsersResponse struct {
	Result  string                        `json:"result"`
	Message *string                       `json:"message,omitempty"`
	Data    TautulliPlaysByTop10UsersData `json:"data"`
}

type TautulliPlaysByTop10UsersData struct {
	Categories []string                    `json:"categories"` // Array of usernames
	Series     []TautulliPlaysByDateSeries `json:"series"`     // Reuses existing series structure
}

// TautulliPlaysPerMonth represents the API response from get_plays_per_month endpoint
type TautulliPlaysPerMonth struct {
	Response TautulliPlaysPerMonthResponse `json:"response"`
}

type TautulliPlaysPerMonthResponse struct {
	Result  string                    `json:"result"`
	Message *string                   `json:"message,omitempty"`
	Data    TautulliPlaysPerMonthData `json:"data"`
}

type TautulliPlaysPerMonthData struct {
	Categories []string                    `json:"categories"` // Array of month strings (YYYY-MM)
	Series     []TautulliPlaysByDateSeries `json:"series"`     // Reuses existing series structure
}
