// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliHomeStats represents the API response from Tautulli's get_home_stats endpoint
type TautulliHomeStats struct {
	Response TautulliHomeStatsResponse `json:"response"`
}

type TautulliHomeStatsResponse struct {
	Result  string                `json:"result"`
	Message *string               `json:"message,omitempty"`
	Data    []TautulliHomeStatRow `json:"data"`
}

type TautulliHomeStatRow struct {
	StatID        string                   `json:"stat_id"`
	StatType      string                   `json:"stat_type"` // "top_movies", "top_tv", "top_users", etc.
	StatTitle     string                   `json:"stat_title"`
	Rows          []TautulliHomeStatDetail `json:"rows"`
	RowsForMedia  bool                     `json:"rows_for_media,omitempty"`
	RowsForUser   bool                     `json:"rows_for_user,omitempty"`
	TotalDuration int64                    `json:"total_duration,omitempty"`
	TotalPlays    int                      `json:"total_plays,omitempty"`
}

type TautulliHomeStatDetail struct {
	// Common fields
	Title            string `json:"title,omitempty"`
	User             string `json:"user,omitempty"`
	FriendlyName     string `json:"friendly_name,omitempty"`
	TotalPlays       int    `json:"total_plays,omitempty"`
	TotalDuration    int64  `json:"total_duration,omitempty"`
	UsersWatched     int    `json:"users_watched,omitempty"`
	RatingKey        string `json:"rating_key,omitempty"`
	LastPlay         int64  `json:"last_play,omitempty"`
	GrandparentTitle string `json:"grandparent_title,omitempty"`
	MediaType        string `json:"media_type,omitempty"`
	Platform         string `json:"platform_name,omitempty"`
	PlatformType     string `json:"platform_type,omitempty"`
	PlayerName       string `json:"player_name,omitempty"`
	LibraryName      string `json:"library_name,omitempty"`
	SectionID        int    `json:"section_id,omitempty"`
	ThumbURL         string `json:"thumb,omitempty"`
	Year             int    `json:"year,omitempty"`
}

// TautulliPlaysByDate represents the API response from get_plays_by_date endpoint
type TautulliPlaysByDate struct {
	Response TautulliPlaysByDateResponse `json:"response"`
}

type TautulliPlaysByDateResponse struct {
	Result  string                  `json:"result"`
	Message *string                 `json:"message,omitempty"`
	Data    TautulliPlaysByDateData `json:"data"`
}

type TautulliPlaysByDateData struct {
	Categories []string                    `json:"categories"` // Array of date strings
	Series     []TautulliPlaysByDateSeries `json:"series"`     // Series data by media type
}

type TautulliPlaysByDateSeries struct {
	Name string        `json:"name"` // "Movies", "TV", "Music", "Live TV"
	Data []interface{} `json:"data"` // Array of counts (int or float)
}

// TautulliPlaysByDayOfWeek represents the API response from get_plays_by_dayofweek endpoint
type TautulliPlaysByDayOfWeek struct {
	Response TautulliPlaysByDayOfWeekResponse `json:"response"`
}

type TautulliPlaysByDayOfWeekResponse struct {
	Result  string                       `json:"result"`
	Message *string                      `json:"message,omitempty"`
	Data    TautulliPlaysByDayOfWeekData `json:"data"`
}

type TautulliPlaysByDayOfWeekData struct {
	Categories []string                    `json:"categories"` // ["Sunday", "Monday", ...]
	Series     []TautulliPlaysByDateSeries `json:"series"`     // Same structure as plays by date
}

// TautulliPlaysByHourOfDay represents the API response from get_plays_by_hourofday endpoint
type TautulliPlaysByHourOfDay struct {
	Response TautulliPlaysByHourOfDayResponse `json:"response"`
}

type TautulliPlaysByHourOfDayResponse struct {
	Result  string                       `json:"result"`
	Message *string                      `json:"message,omitempty"`
	Data    TautulliPlaysByHourOfDayData `json:"data"`
}

type TautulliPlaysByHourOfDayData struct {
	Categories []string                    `json:"categories"` // ["00", "01", ..., "23"]
	Series     []TautulliPlaysByDateSeries `json:"series"`     // Same structure as plays by date
}
