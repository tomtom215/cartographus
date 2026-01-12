// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliPlaysByStreamType represents the API response from get_plays_by_stream_type endpoint
type TautulliPlaysByStreamType struct {
	Response TautulliPlaysByStreamTypeResponse `json:"response"`
}

type TautulliPlaysByStreamTypeResponse struct {
	Result  string                        `json:"result"`
	Message *string                       `json:"message,omitempty"`
	Data    TautulliPlaysByStreamTypeData `json:"data"`
}

type TautulliPlaysByStreamTypeData struct {
	Categories []string                   `json:"categories"` // Array of date strings
	Series     []TautulliStreamTypeSeries `json:"series"`     // Direct Play, Direct Stream, Transcode
}

type TautulliStreamTypeSeries struct {
	Name string        `json:"name"` // "Direct Play", "Direct Stream", "Transcode"
	Data []interface{} `json:"data"` // Array of counts
}

// TautulliConcurrentStreamsByStreamType represents the API response from get_concurrent_streams_by_stream_type endpoint
type TautulliConcurrentStreamsByStreamType struct {
	Response TautulliConcurrentStreamsByStreamTypeResponse `json:"response"`
}

type TautulliConcurrentStreamsByStreamTypeResponse struct {
	Result  string                                    `json:"result"`
	Message *string                                   `json:"message,omitempty"`
	Data    TautulliConcurrentStreamsByStreamTypeData `json:"data"`
}

type TautulliConcurrentStreamsByStreamTypeData struct {
	Categories []string                   `json:"categories"` // Array of date strings
	Series     []TautulliStreamTypeSeries `json:"series"`     // Including "Total Concurrent Streams"
}

// TautulliStreamTypeByTop10Users represents the API response from get_stream_type_by_top_10_users endpoint
type TautulliStreamTypeByTop10Users struct {
	Response TautulliStreamTypeByTop10UsersResponse `json:"response"`
}

type TautulliStreamTypeByTop10UsersResponse struct {
	Result  string                             `json:"result"`
	Message *string                            `json:"message,omitempty"`
	Data    TautulliStreamTypeByTop10UsersData `json:"data"`
}

type TautulliStreamTypeByTop10UsersData struct {
	Categories []string                    `json:"categories"` // Array of usernames
	Series     []TautulliPlaysByDateSeries `json:"series"`     // Stream type series (direct play, transcode, etc.)
}

// TautulliStreamTypeByTop10Platforms represents the API response from get_stream_type_by_top_10_platforms endpoint
type TautulliStreamTypeByTop10Platforms struct {
	Response TautulliStreamTypeByTop10PlatformsResponse `json:"response"`
}

type TautulliStreamTypeByTop10PlatformsResponse struct {
	Result  string                                 `json:"result"`
	Message *string                                `json:"message,omitempty"`
	Data    TautulliStreamTypeByTop10PlatformsData `json:"data"`
}

type TautulliStreamTypeByTop10PlatformsData struct {
	Categories []string                    `json:"categories"` // Array of platform names
	Series     []TautulliPlaysByDateSeries `json:"series"`     // Stream type series
}
