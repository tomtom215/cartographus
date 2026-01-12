// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliLibraryUserStats represents the API response from get_library_user_stats endpoint
type TautulliLibraryUserStats struct {
	Response TautulliLibraryUserStatsResponse `json:"response"`
}

type TautulliLibraryUserStatsResponse struct {
	Result  string                       `json:"result"`
	Message *string                      `json:"message,omitempty"`
	Data    []TautulliLibraryUserStatRow `json:"data"`
}

type TautulliLibraryUserStatRow struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	FriendlyName string `json:"friendly_name"`
	UserThumb    string `json:"user_thumb"`
	TotalPlays   int    `json:"total_plays"`
	TotalTime    int64  `json:"total_time"`
}

// TautulliLibraries represents the API response from get_libraries endpoint
type TautulliLibraries struct {
	Response TautulliLibrariesResponse `json:"response"`
}

type TautulliLibrariesResponse struct {
	Result  string                  `json:"result"`
	Message *string                 `json:"message,omitempty"`
	Data    []TautulliLibraryDetail `json:"data"`
}

type TautulliLibraryDetail struct {
	SectionID   int    `json:"section_id"`
	SectionName string `json:"section_name"`
	SectionType string `json:"section_type"`
	Count       int    `json:"count"`
	ParentCount int    `json:"parent_count"`
	ChildCount  int    `json:"child_count"`
	IsActive    int    `json:"is_active"`
	Thumb       string `json:"thumb"`
	Art         string `json:"art"`
}

// TautulliLibrary represents the API response from get_library endpoint
type TautulliLibrary struct {
	Response TautulliLibraryResponse `json:"response"`
}

type TautulliLibraryResponse struct {
	Result  string              `json:"result"`
	Message *string             `json:"message,omitempty"`
	Data    TautulliLibraryData `json:"data"`
}

type TautulliLibraryData struct {
	SectionID       int    `json:"section_id"`
	SectionName     string `json:"section_name"`
	SectionType     string `json:"section_type"`
	Agent           string `json:"agent"`
	Thumb           string `json:"thumb"`
	Art             string `json:"art"`
	Count           int    `json:"count"`
	ParentCount     int    `json:"parent_count"`
	ChildCount      int    `json:"child_count"`
	IsActive        int    `json:"is_active"`
	DoNotifyCreated int    `json:"do_notify_created"`
	KeepHistory     int    `json:"keep_history"`
	DeletedSection  int    `json:"deleted_section"`
	LastAccessed    int64  `json:"last_accessed"`
}

// TautulliLibrariesTable represents the API response from get_libraries_table endpoint
type TautulliLibrariesTable struct {
	Response TautulliLibrariesTableResponse `json:"response"`
}

type TautulliLibrariesTableResponse struct {
	Result  string                     `json:"result"`
	Message *string                    `json:"message,omitempty"`
	Data    TautulliLibrariesTableData `json:"data"`
}

type TautulliLibrariesTableData struct {
	RecordsFiltered int                         `json:"recordsFiltered"`
	RecordsTotal    int                         `json:"recordsTotal"`
	Draw            int                         `json:"draw,omitempty"`
	Data            []TautulliLibrariesTableRow `json:"data"`
}

type TautulliLibrariesTableRow struct {
	SectionID       int    `json:"section_id"`
	SectionName     string `json:"section_name"`
	SectionType     string `json:"section_type"`
	Agent           string `json:"agent,omitempty"`
	Thumb           string `json:"thumb,omitempty"`
	Art             string `json:"art,omitempty"`
	Count           int    `json:"count"` // Number of items in library
	ParentCount     int    `json:"parent_count,omitempty"`
	ChildCount      int    `json:"child_count,omitempty"`
	IsActive        int    `json:"is_active,omitempty"`
	DoNotify        int    `json:"do_notify,omitempty"`
	DoNotifyCreated int    `json:"do_notify_created,omitempty"`
	KeepHistory     int    `json:"keep_history,omitempty"`
	DeletedSection  int    `json:"deleted_section,omitempty"`
	Plays           int    `json:"plays,omitempty"`
	Duration        int    `json:"duration,omitempty"`
	LastAccessed    int64  `json:"last_accessed,omitempty"`
	LastPlayed      string `json:"last_played,omitempty"`
}

// TautulliLibraryMediaInfo represents the API response from get_library_media_info endpoint
type TautulliLibraryMediaInfo struct {
	Response TautulliLibraryMediaInfoResponse `json:"response"`
}

type TautulliLibraryMediaInfoResponse struct {
	Result  string                       `json:"result"`
	Message *string                      `json:"message,omitempty"`
	Data    TautulliLibraryMediaInfoData `json:"data"`
}

type TautulliLibraryMediaInfoData struct {
	RecordsFiltered int                           `json:"recordsFiltered"`
	RecordsTotal    int                           `json:"recordsTotal"`
	Draw            int                           `json:"draw,omitempty"`
	Data            []TautulliLibraryMediaInfoRow `json:"data"`
}

type TautulliLibraryMediaInfoRow struct {
	SectionID            int    `json:"section_id"`
	SectionType          string `json:"section_type"`
	AddedAt              int64  `json:"added_at"`
	MediaType            string `json:"media_type"`
	RatingKey            string `json:"rating_key"`
	ParentRatingKey      string `json:"parent_rating_key,omitempty"`
	GrandparentRatingKey string `json:"grandparent_rating_key,omitempty"`
	Title                string `json:"title"`
	Year                 int    `json:"year,omitempty"`
	MediaIndex           int    `json:"media_index,omitempty"`
	ParentMediaIndex     int    `json:"parent_media_index,omitempty"`
	Thumb                string `json:"thumb,omitempty"`
	Container            string `json:"container,omitempty"`
	Bitrate              int    `json:"bitrate,omitempty"`
	VideoCodec           string `json:"video_codec,omitempty"`
	VideoResolution      string `json:"video_resolution,omitempty"`
	VideoFrameRate       string `json:"video_framerate,omitempty"`
	AudioCodec           string `json:"audio_codec,omitempty"`
	AudioChannels        string `json:"audio_channels,omitempty"`
	FileSize             int64  `json:"file_size,omitempty"`
	LastPlayed           int64  `json:"last_played,omitempty"`
	PlayCount            int    `json:"play_count,omitempty"`
}

// TautulliLibraryWatchTimeStats represents the API response from get_library_watch_time_stats endpoint
type TautulliLibraryWatchTimeStats struct {
	Response TautulliLibraryWatchTimeStatsResponse `json:"response"`
}

type TautulliLibraryWatchTimeStatsResponse struct {
	Result  string                            `json:"result"`
	Message *string                           `json:"message,omitempty"`
	Data    []TautulliLibraryWatchTimeStatRow `json:"data"`
}

type TautulliLibraryWatchTimeStatRow struct {
	QueryDays   int    `json:"query_days"`
	TotalTime   int    `json:"total_time"`  // Total watch time in seconds
	TotalPlays  int    `json:"total_plays"` // Total number of plays
	SectionID   int    `json:"section_id"`
	SectionName string `json:"section_name,omitempty"`
	SectionType string `json:"section_type,omitempty"`
}

// TautulliLibraryNames represents the API response from get_library_names endpoint
type TautulliLibraryNames struct {
	Response TautulliLibraryNamesResponse `json:"response"`
}

type TautulliLibraryNamesResponse struct {
	Result  string                    `json:"result"`
	Message *string                   `json:"message,omitempty"`
	Data    []TautulliLibraryNameItem `json:"data"`
}

type TautulliLibraryNameItem struct {
	SectionID   int    `json:"section_id"`
	SectionName string `json:"section_name"`
	SectionType string `json:"section_type,omitempty"`
}
