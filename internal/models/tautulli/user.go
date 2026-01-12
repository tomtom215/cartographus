// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliUser represents the API response from Tautulli's get_user endpoint
type TautulliUser struct {
	Response TautulliUserResponse `json:"response"`
}

type TautulliUserResponse struct {
	Result  string           `json:"result"`
	Message *string          `json:"message,omitempty"`
	Data    TautulliUserData `json:"data"`
}

type TautulliUserData struct {
	UserID          int      `json:"user_id"`
	Username        string   `json:"username"`
	FriendlyName    string   `json:"friendly_name"`
	UserThumb       string   `json:"user_thumb"`
	Email           string   `json:"email"`
	IsHomeUser      int      `json:"is_home_user"`
	IsAllowSync     int      `json:"is_allow_sync"`
	IsRestricted    int      `json:"is_restricted"`
	DoNotify        int      `json:"do_notify"`
	KeepHistory     int      `json:"keep_history"`
	DeletedUser     int      `json:"deleted_user"`
	AllowGuest      int      `json:"allow_guest"`
	ServerToken     string   `json:"server_token"`
	SharedLibraries []string `json:"shared_libraries"`
	FilterAll       string   `json:"filter_all"`
	FilterMovies    string   `json:"filter_movies"`
	FilterTV        string   `json:"filter_tv"`
	FilterMusic     string   `json:"filter_music"`
	FilterPhotos    string   `json:"filter_photos"`
}

// TautulliUsersTable represents the API response from get_users_table endpoint
type TautulliUsersTable struct {
	Response TautulliUsersTableResponse `json:"response"`
}

type TautulliUsersTableResponse struct {
	Result  string                 `json:"result"`
	Message *string                `json:"message,omitempty"`
	Data    TautulliUsersTableData `json:"data"`
}

type TautulliUsersTableData struct {
	RecordsTotal    int                     `json:"recordsTotal"`
	RecordsFiltered int                     `json:"recordsFiltered"`
	Draw            int                     `json:"draw"`
	Data            []TautulliUsersTableRow `json:"data"`
}

type TautulliUsersTableRow struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"user"`
	FriendlyName string `json:"friendly_name"`
	UserThumb    string `json:"user_thumb"`
	Plays        int    `json:"plays"`
	Duration     int    `json:"duration"`
	LastSeen     int64  `json:"last_seen"`
	LastPlayed   string `json:"last_played"`
	IPAddress    string `json:"ip_address,omitempty"`
	PlatformName string `json:"platform,omitempty"`
	PlayerName   string `json:"player,omitempty"`
	MediaType    string `json:"media_type,omitempty"`
	Title        string `json:"title,omitempty"`
	Thumb        string `json:"thumb,omitempty"`
	RatingKey    string `json:"rating_key,omitempty"`
}

// TautulliUserIPs represents the API response from get_user_ips endpoint
type TautulliUserIPs struct {
	Response TautulliUserIPsResponse `json:"response"`
}

type TautulliUserIPsResponse struct {
	Result  string               `json:"result"`
	Message *string              `json:"message,omitempty"`
	Data    []TautulliUserIPData `json:"data"`
}

type TautulliUserIPData struct {
	FriendlyName string `json:"friendly_name"`
	IPAddress    string `json:"ip_address"`
	LastSeen     int64  `json:"last_seen"`
	LastPlayed   string `json:"last_played"`
	PlayCount    int    `json:"play_count"`
	PlatformName string `json:"platform_name,omitempty"`
	PlayerName   string `json:"player_name,omitempty"`
	UserID       int    `json:"user_id"`
}

// TautulliUserLogins represents the API response from get_user_logins endpoint
type TautulliUserLogins struct {
	Response TautulliUserLoginsResponse `json:"response"`
}

type TautulliUserLoginsResponse struct {
	Result  string                 `json:"result"`
	Message *string                `json:"message,omitempty"`
	Data    TautulliUserLoginsData `json:"data"`
}

type TautulliUserLoginsData struct {
	RecordsTotal    int                     `json:"recordsTotal"`
	RecordsFiltered int                     `json:"recordsFiltered"`
	Draw            int                     `json:"draw"`
	Data            []TautulliUserLoginsRow `json:"data"`
}

type TautulliUserLoginsRow struct {
	Timestamp    int64  `json:"timestamp"`
	Time         string `json:"time"`
	UserID       int    `json:"user_id"`
	Username     string `json:"user"`
	FriendlyName string `json:"friendly_name"`
	IPAddress    string `json:"ip_address"`
	Host         string `json:"host,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
	OS           string `json:"os,omitempty"`
	Browser      string `json:"browser,omitempty"`
	Success      int    `json:"success"`
}

// TautulliUserPlayerStats represents the API response from get_user_player_stats endpoint
type TautulliUserPlayerStats struct {
	Response TautulliUserPlayerStatsResponse `json:"response"`
}

type TautulliUserPlayerStatsResponse struct {
	Result  string                      `json:"result"`
	Message *string                     `json:"message,omitempty"`
	Data    []TautulliUserPlayerStatRow `json:"data"`
}

type TautulliUserPlayerStatRow struct {
	PlayerName   string `json:"player_name"`
	PlatformName string `json:"platform_name"`
	PlatformType string `json:"platform_type"`
	ResultID     int    `json:"result_id"`
	RowID        int    `json:"row_id"`
	TotalPlays   int    `json:"total_plays"`
	LastPlay     int64  `json:"last_play"`
	LastPlayed   string `json:"last_played"`
	MediaType    string `json:"media_type,omitempty"`
	RatingKey    string `json:"rating_key,omitempty"`
	Thumb        string `json:"thumb,omitempty"`
	Title        string `json:"title,omitempty"`
	UserID       int    `json:"user_id,omitempty"`
}

// TautulliUserWatchTimeStats represents the API response from get_user_watch_time_stats endpoint
type TautulliUserWatchTimeStats struct {
	Response TautulliUserWatchTimeStatsResponse `json:"response"`
}

type TautulliUserWatchTimeStatsResponse struct {
	Result  string                         `json:"result"`
	Message *string                        `json:"message,omitempty"`
	Data    []TautulliUserWatchTimeStatRow `json:"data"`
}

type TautulliUserWatchTimeStatRow struct {
	QueryDays  int    `json:"query_days"`
	TotalTime  int    `json:"total_time"`  // Total watch time in seconds
	TotalPlays int    `json:"total_plays"` // Total number of plays
	MediaType  string `json:"media_type"`  // movie, episode, track, etc.
}

// TautulliUsers represents the API response from Tautulli's get_users endpoint
// Returns a list of all users that have accessed the Plex server
type TautulliUsers struct {
	Response TautulliUsersResponse `json:"response"`
}

type TautulliUsersResponse struct {
	Result  string             `json:"result"`
	Message *string            `json:"message,omitempty"`
	Data    []TautulliUserData `json:"data"`
}
