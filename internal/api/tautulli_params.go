// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

// Parameter structs for Tautulli API proxy endpoints
// These replace []interface{} with strongly-typed structs for compile-time type safety

// StandardTimeRangeParams is used by most time-series analytics endpoints
type StandardTimeRangeParams struct {
	TimeRange int
	YAxis     string
	UserID    int
	Grouping  int
}

// HomeStatsParams is used by TautulliHomeStats
type HomeStatsParams struct {
	TimeRange  int
	StatsType  string
	StatsCount int
}

// TwoIntParams is used by endpoints that take two integer parameters
type TwoIntParams struct {
	Param1 int
	Param2 int
}

// SingleStringParam is used by endpoints that take a single string parameter
type SingleStringParam struct {
	Value string
}

// SingleIntParam is used by endpoints that take a single integer parameter
type SingleIntParam struct {
	Value int
}

// ItemWatchTimeParams is used by TautulliItemWatchTimeStats
type ItemWatchTimeParams struct {
	RatingKey string
	Grouping  int
	QueryDays string
}

// RecentlyAddedParams is used by TautulliRecentlyAdded
type RecentlyAddedParams struct {
	Count     int
	Start     int
	MediaType string
	SectionID int
}

// TableParams is used by endpoints that return paginated table data
type TableParams struct {
	Grouping    int
	OrderColumn string
	OrderDir    string
	Start       int
	Length      int
	Search      string
}

// LibraryMediaInfoParams is used by TautulliLibraryMediaInfo
type LibraryMediaInfoParams struct {
	SectionID   int
	OrderColumn string
	OrderDir    string
	Start       int
	Length      int
}

// LibraryWatchTimeParams is used by TautulliLibraryWatchTimeStats
type LibraryWatchTimeParams struct {
	SectionID int
	Grouping  int
	QueryDays string
}

// ChildrenMetadataParams is used by TautulliChildrenMetadata
type ChildrenMetadataParams struct {
	RatingKey string
	MediaType string
}

// UsersTableParams is used by TautulliUsersTable
type UsersTableParams struct {
	Grouping    int
	OrderColumn string
	OrderDir    string
	Start       int
	Length      int
	Search      string
}

// UserLoginsParams is used by TautulliUserLogins
type UserLoginsParams struct {
	UserID      int
	OrderColumn string
	OrderDir    string
	Start       int
	Length      int
	Search      string
}

// StreamDataParams is used by TautulliStreamData
type StreamDataParams struct {
	RowID      int
	SessionKey string
}

// ExportMetadataParams is used by TautulliExportMetadata
type ExportMetadataParams struct {
	SectionID  int
	ExportType string
	UserID     int
	RatingKey  string
	FileFormat string
}

// SearchParams is used by TautulliSearch
type SearchParams struct {
	Query string
	Limit int
}

// CollectionsTableParams is used by TautulliCollectionsTable
type CollectionsTableParams struct {
	SectionID   int
	OrderColumn string
	OrderDir    string
	Start       int
	Length      int
	Search      string
}

// PlaylistsTableParams is used by TautulliPlaylistsTable
type PlaylistsTableParams struct {
	SectionID   int
	OrderColumn string
	OrderDir    string
	Start       int
	Length      int
	Search      string
}

// ExportsTableParams is used by TautulliExportsTable
type ExportsTableParams struct {
	OrderColumn string
	OrderDir    string
	Start       int
	Length      int
	Search      string
}

// UserWatchTimeParams is used by TautulliUserWatchTimeStats
type UserWatchTimeParams struct {
	UserID    int
	QueryDays string
}

// ItemUserStatsParams is used by TautulliItemUserStats
type ItemUserStatsParams struct {
	RatingKey string
	Grouping  int
}

// SyncedItemsParams is used by TautulliSyncedItems
type SyncedItemsParams struct {
	MachineID string
	UserID    int
}

// TerminateSessionParams is used by TautulliTerminateSession
type TerminateSessionParams struct {
	SessionID string
	Message   string
}

// NoParams is used by endpoints that don't take any parameters
type NoParams struct{}
