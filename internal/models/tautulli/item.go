// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliItemWatchTimeStats represents the API response from get_item_watch_time_stats endpoint
type TautulliItemWatchTimeStats struct {
	Response TautulliItemWatchTimeStatsResponse `json:"response"`
}

type TautulliItemWatchTimeStatsResponse struct {
	Result  string                        `json:"result"`
	Message *string                       `json:"message,omitempty"`
	Data    []TautulliItemWatchTimeDetail `json:"data"`
}

type TautulliItemWatchTimeDetail struct {
	QueryDays  string `json:"query_days"`  // "1", "7", "30", "0" (all time)
	TotalTime  int64  `json:"total_time"`  // Total watch time in seconds
	TotalPlays int    `json:"total_plays"` // Number of plays
}

// TautulliItemUserStats represents the API response from get_item_user_stats endpoint
type TautulliItemUserStats struct {
	Response TautulliItemUserStatsResponse `json:"response"`
}

type TautulliItemUserStatsResponse struct {
	Result  string                    `json:"result"`
	Message *string                   `json:"message,omitempty"`
	Data    []TautulliItemUserStatRow `json:"data"`
}

type TautulliItemUserStatRow struct {
	UserID          int     `json:"user_id"`
	Username        string  `json:"user"`
	FriendlyName    string  `json:"friendly_name,omitempty"`
	TotalPlays      int     `json:"total_plays"`
	LastPlay        int64   `json:"last_play"`
	LastPlayed      string  `json:"last_played"`
	Platform        string  `json:"platform,omitempty"`
	Player          string  `json:"player,omitempty"`
	IPAddress       string  `json:"ip_address,omitempty"`
	PercentComplete int     `json:"percent_complete,omitempty"`
	WatchedStatus   float64 `json:"watched_status,omitempty"`
	Thumb           string  `json:"thumb,omitempty"`
}
