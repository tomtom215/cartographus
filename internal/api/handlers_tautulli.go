// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"

	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// This file contains all Tautulli API proxy endpoints (refactored)
// Total: 54 methods proxying Tautulli API calls
//
// Refactored from 2,627 lines to ~600 lines using configuration-driven approach
// Each handler now uses proxyTautulliRequest with a TautulliProxyConfig

// TautulliHomeStats handles Tautulli home statistics requests
func (h *Handler) TautulliHomeStats(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[HomeStatsParams, *tautulli.TautulliHomeStats]{
		CacheName:   "TautulliHomeStats",
		ParseParams: parseHomeStatsParams,
		CallClient: func(ctx context.Context, h *Handler, p HomeStatsParams) (*tautulli.TautulliHomeStats, error) {
			return h.client.GetHomeStats(ctx, p.TimeRange, p.StatsType, p.StatsCount)
		},
		ExtractData: func(resp *tautulli.TautulliHomeStats) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve home statistics from Tautulli",
	})
}

// TautulliPlaysByDate handles Tautulli plays by date requests
//
// @Summary Get playback data by date
// @Description Returns time series playback data organized by date from Tautulli
// @Tags Tautulli Analytics
// @Accept json
// @Produce json
// @Param time_range query int false "Time range in days" default(30)
// @Param y_axis query string false "Y-axis type (plays or duration)" default(plays)
// @Param user_id query int false "Filter by user ID" default(0)
// @Param grouping query int false "Group data (0 or 1)" default(0)
// @Success 200 {object} models.APIResponse{data=tautulli.TautulliPlaysByDateData} "Plays by date retrieved successfully"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /api/v1/tautulli/plays-by-date [get]
func (h *Handler) TautulliPlaysByDate(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliPlaysByDate]{
		CacheName:   "TautulliPlaysByDate",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliPlaysByDate, error) {
			return h.client.GetPlaysByDate(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliPlaysByDate) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve plays by date from Tautulli",
	})
}

// TautulliPlaysByDayOfWeek handles Tautulli plays by day of week requests
//
// @Summary Get playback data by day of week
// @Description Returns playback data organized by day of week from Tautulli
// @Tags Tautulli Analytics
// @Accept json
// @Produce json
// @Param time_range query int false "Time range in days" default(30)
// @Param y_axis query string false "Y-axis type (plays or duration)" default(plays)
// @Param user_id query int false "Filter by user ID" default(0)
// @Param grouping query int false "Group data (0 or 1)" default(0)
// @Success 200 {object} models.APIResponse{data=tautulli.TautulliPlaysByDayOfWeekData} "Plays by day of week retrieved successfully"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /api/v1/tautulli/plays-by-dayofweek [get]
func (h *Handler) TautulliPlaysByDayOfWeek(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliPlaysByDayOfWeek]{
		CacheName:   "TautulliPlaysByDayOfWeek",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliPlaysByDayOfWeek, error) {
			return h.client.GetPlaysByDayOfWeek(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliPlaysByDayOfWeek) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve plays by day of week from Tautulli",
	})
}

// TautulliPlaysByHourOfDay handles Tautulli plays by hour of day requests
//
// @Summary Get playback data by hour of day
// @Description Returns playback data organized by hour of day from Tautulli
// @Tags Tautulli Analytics
// @Accept json
// @Produce json
// @Param time_range query int false "Time range in days" default(30)
// @Param y_axis query string false "Y-axis type (plays or duration)" default(plays)
// @Param user_id query int false "Filter by user ID" default(0)
// @Param grouping query int false "Group data (0 or 1)" default(0)
// @Success 200 {object} models.APIResponse{data=tautulli.TautulliPlaysByHourOfDayData} "Plays by hour of day retrieved successfully"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /api/v1/tautulli/plays-by-hourofday [get]
func (h *Handler) TautulliPlaysByHourOfDay(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliPlaysByHourOfDay]{
		CacheName:   "TautulliPlaysByHourOfDay",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliPlaysByHourOfDay, error) {
			return h.client.GetPlaysByHourOfDay(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliPlaysByHourOfDay) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve plays by hour of day from Tautulli",
	})
}

// TautulliPlaysByStreamType handles Tautulli plays by stream type requests
//
// @Summary Get playback data by stream type
// @Description Returns playback data organized by stream type (Direct Play, Direct Stream, Transcode) from Tautulli
// @Tags Tautulli Analytics
// @Accept json
// @Produce json
// @Param time_range query int false "Time range in days" default(30)
// @Param y_axis query string false "Y-axis type (plays or duration)" default(plays)
// @Param user_id query int false "Filter by user ID" default(0)
// @Param grouping query int false "Group data (0 or 1)" default(0)
// @Success 200 {object} models.APIResponse{data=tautulli.TautulliPlaysByStreamTypeData} "Plays by stream type retrieved successfully"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /api/v1/tautulli/plays-by-stream-type [get]
func (h *Handler) TautulliPlaysByStreamType(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliPlaysByStreamType]{
		CacheName:   "TautulliPlaysByStreamType",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliPlaysByStreamType, error) {
			return h.client.GetPlaysByStreamType(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliPlaysByStreamType) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve plays by stream type from Tautulli",
	})
}

// TautulliConcurrentStreamsByStreamType handles Tautulli concurrent streams by stream type requests
//
// @Summary Get concurrent streams data by stream type
// @Description Returns concurrent streams data organized by stream type from Tautulli
// @Tags Tautulli Analytics
// @Accept json
// @Produce json
// @Param time_range query int false "Time range in days" default(30)
// @Param user_id query int false "Filter by user ID" default(0)
// @Success 200 {object} models.APIResponse{data=tautulli.TautulliConcurrentStreamsByStreamTypeData} "Concurrent streams by stream type retrieved successfully"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /api/v1/tautulli/concurrent-streams-by-stream-type [get]
func (h *Handler) TautulliConcurrentStreamsByStreamType(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[TwoIntParams, *tautulli.TautulliConcurrentStreamsByStreamType]{
		CacheName:   "TautulliConcurrentStreamsByStreamType",
		ParseParams: parseTimeRangeUserIDParams,
		CallClient: func(ctx context.Context, h *Handler, p TwoIntParams) (*tautulli.TautulliConcurrentStreamsByStreamType, error) {
			return h.client.GetConcurrentStreamsByStreamType(ctx, p.Param1, p.Param2)
		},
		ExtractData: func(resp *tautulli.TautulliConcurrentStreamsByStreamType) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve concurrent streams by stream type from Tautulli",
	})
}

// TautulliItemWatchTimeStats handles Tautulli item watch time statistics requests
func (h *Handler) TautulliItemWatchTimeStats(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[ItemWatchTimeParams, *tautulli.TautulliItemWatchTimeStats]{
		CacheName:   "TautulliItemWatchTimeStats",
		ParseParams: parseItemWatchTimeStatsParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p ItemWatchTimeParams) (*tautulli.TautulliItemWatchTimeStats, error) {
			return h.client.GetItemWatchTimeStats(ctx, p.RatingKey, p.Grouping, p.QueryDays)
		},
		ExtractData: func(resp *tautulli.TautulliItemWatchTimeStats) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve item watch time stats from Tautulli",
	})
}

// TautulliActivity handles Tautulli activity requests (no caching for real-time data)
func (h *Handler) TautulliActivity(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleStringParam, *tautulli.TautulliActivity]{
		CacheName:   "", // No caching for real-time activity
		ParseParams: parseActivityParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleStringParam) (*tautulli.TautulliActivity, error) {
			return h.client.GetActivity(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliActivity) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve activity from Tautulli",
	})
}

// TautulliMetadata handles Tautulli metadata requests
func (h *Handler) TautulliMetadata(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleStringParam, *tautulli.TautulliMetadata]{
		CacheName:   "TautulliMetadata",
		ParseParams: parseMetadataParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleStringParam) (*tautulli.TautulliMetadata, error) {
			return h.client.GetMetadata(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliMetadata) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve metadata from Tautulli",
	})
}

// TautulliUser handles Tautulli user requests
func (h *Handler) TautulliUser(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleIntParam, *tautulli.TautulliUser]{
		CacheName:   "TautulliUser",
		ParseParams: parseUserParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleIntParam) (*tautulli.TautulliUser, error) {
			return h.client.GetUser(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliUser) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve user from Tautulli",
	})
}

// TautulliUsers handles bulk user fetch requests - returns all users at once
// This is more efficient than calling TautulliUser for each user individually
func (h *Handler) TautulliUsers(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliUsers]{
		CacheName:   "TautulliUsers",
		ParseParams: parseUsersParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliUsers, error) {
			return h.client.GetUsers(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliUsers) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve users from Tautulli",
	})
}

// TautulliLibraryUserStats handles Tautulli library user statistics requests
func (h *Handler) TautulliLibraryUserStats(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[TwoIntParams, *tautulli.TautulliLibraryUserStats]{
		CacheName:   "TautulliLibraryUserStats",
		ParseParams: parseLibraryUserStatsParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p TwoIntParams) (*tautulli.TautulliLibraryUserStats, error) {
			return h.client.GetLibraryUserStats(ctx, p.Param1, p.Param2)
		},
		ExtractData: func(resp *tautulli.TautulliLibraryUserStats) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve library user stats from Tautulli",
	})
}

// TautulliRecentlyAdded handles Tautulli recently added media requests
func (h *Handler) TautulliRecentlyAdded(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[RecentlyAddedParams, *tautulli.TautulliRecentlyAdded]{
		CacheName:   "", // No caching for recently added (frequently changing)
		ParseParams: parseRecentlyAddedParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p RecentlyAddedParams) (*tautulli.TautulliRecentlyAdded, error) {
			return h.client.GetRecentlyAdded(ctx, p.Count, p.Start, p.MediaType, p.SectionID)
		},
		ExtractData: func(resp *tautulli.TautulliRecentlyAdded) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve recently added media from Tautulli",
	})
}

// TautulliLibraries handles Tautulli libraries list requests
func (h *Handler) TautulliLibraries(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliLibraries]{
		CacheName:   "TautulliLibraries",
		ParseParams: parseLibrariesParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliLibraries, error) {
			return h.client.GetLibraries(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliLibraries) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve libraries from Tautulli",
	})
}

// TautulliLibrary handles Tautulli single library requests
func (h *Handler) TautulliLibrary(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleIntParam, *tautulli.TautulliLibrary]{
		CacheName:   "TautulliLibrary",
		ParseParams: parseLibraryParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleIntParam) (*tautulli.TautulliLibrary, error) {
			return h.client.GetLibrary(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliLibrary) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve library from Tautulli",
	})
}

// TautulliServerInfo handles Tautulli server information requests
func (h *Handler) TautulliServerInfo(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliServerInfo]{
		CacheName:   "TautulliServerInfo",
		ParseParams: parseServerInfoParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliServerInfo, error) {
			return h.client.GetServerInfo(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliServerInfo) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve server info from Tautulli",
	})
}

// TautulliSyncedItems handles Tautulli synced items requests
func (h *Handler) TautulliSyncedItems(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SyncedItemsParams, *tautulli.TautulliSyncedItems]{
		CacheName:   "TautulliSyncedItems",
		ParseParams: parseSyncedItemsParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SyncedItemsParams) (*tautulli.TautulliSyncedItems, error) {
			return h.client.GetSyncedItems(ctx, p.MachineID, p.UserID)
		},
		ExtractData: func(resp *tautulli.TautulliSyncedItems) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve synced items from Tautulli",
	})
}

// TautulliTerminateSession handles Tautulli session termination requests
func (h *Handler) TautulliTerminateSession(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[TerminateSessionParams, *tautulli.TautulliTerminateSession]{
		HTTPMethod:  "POST",
		CacheName:   "", // No caching for write operations
		ParseParams: parseTerminateSessionParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p TerminateSessionParams) (*tautulli.TautulliTerminateSession, error) {
			return h.client.TerminateSession(ctx, p.SessionID, p.Message)
		},
		ExtractData: func(resp *tautulli.TautulliTerminateSession) interface{} {
			return resp.Response // Note: uses .Response not .Response.Data
		},
		ErrorMessage: "Failed to terminate session via Tautulli",
	})
}

// TautulliPlaysBySourceResolution handles Tautulli plays by source resolution requests
func (h *Handler) TautulliPlaysBySourceResolution(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliPlaysBySourceResolution]{
		CacheName:   "TautulliPlaysBySourceResolution",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliPlaysBySourceResolution, error) {
			return h.client.GetPlaysBySourceResolution(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliPlaysBySourceResolution) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve plays by source resolution from Tautulli",
	})
}

// TautulliPlaysByStreamResolution handles Tautulli plays by stream resolution requests
func (h *Handler) TautulliPlaysByStreamResolution(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliPlaysByStreamResolution]{
		CacheName:   "TautulliPlaysByStreamResolution",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliPlaysByStreamResolution, error) {
			return h.client.GetPlaysByStreamResolution(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliPlaysByStreamResolution) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve plays by stream resolution from Tautulli",
	})
}

// TautulliPlaysByTop10Platforms handles Tautulli plays by top 10 platforms requests
func (h *Handler) TautulliPlaysByTop10Platforms(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliPlaysByTop10Platforms]{
		CacheName:   "TautulliPlaysByTop10Platforms",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliPlaysByTop10Platforms, error) {
			return h.client.GetPlaysByTop10Platforms(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliPlaysByTop10Platforms) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve plays by top 10 platforms from Tautulli",
	})
}

// TautulliPlaysByTop10Users handles Tautulli plays by top 10 users requests
func (h *Handler) TautulliPlaysByTop10Users(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliPlaysByTop10Users]{
		CacheName:   "TautulliPlaysByTop10Users",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliPlaysByTop10Users, error) {
			return h.client.GetPlaysByTop10Users(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliPlaysByTop10Users) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve plays by top 10 users from Tautulli",
	})
}

// TautulliPlaysPerMonth handles Tautulli plays per month requests
func (h *Handler) TautulliPlaysPerMonth(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliPlaysPerMonth]{
		CacheName:   "TautulliPlaysPerMonth",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliPlaysPerMonth, error) {
			return h.client.GetPlaysPerMonth(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliPlaysPerMonth) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve plays per month from Tautulli",
	})
}

// TautulliUserPlayerStats handles Tautulli user player statistics requests
func (h *Handler) TautulliUserPlayerStats(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleIntParam, *tautulli.TautulliUserPlayerStats]{
		CacheName:   "TautulliUserPlayerStats",
		ParseParams: parseUserPlayerStatsParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleIntParam) (*tautulli.TautulliUserPlayerStats, error) {
			return h.client.GetUserPlayerStats(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliUserPlayerStats) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve user player stats from Tautulli",
	})
}

// TautulliUserWatchTimeStats handles Tautulli user watch time statistics requests
func (h *Handler) TautulliUserWatchTimeStats(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[UserWatchTimeParams, *tautulli.TautulliUserWatchTimeStats]{
		CacheName:   "TautulliUserWatchTimeStats",
		ParseParams: parseUserWatchTimeStatsParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p UserWatchTimeParams) (*tautulli.TautulliUserWatchTimeStats, error) {
			return h.client.GetUserWatchTimeStats(ctx, p.UserID, p.QueryDays)
		},
		ExtractData: func(resp *tautulli.TautulliUserWatchTimeStats) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve user watch time stats from Tautulli",
	})
}

// TautulliItemUserStats handles Tautulli item user statistics requests
func (h *Handler) TautulliItemUserStats(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[ItemUserStatsParams, *tautulli.TautulliItemUserStats]{
		CacheName:   "TautulliItemUserStats",
		ParseParams: parseItemUserStatsParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p ItemUserStatsParams) (*tautulli.TautulliItemUserStats, error) {
			return h.client.GetItemUserStats(ctx, p.RatingKey, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliItemUserStats) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve item user stats from Tautulli",
	})
}

// TautulliLibrariesTable handles Tautulli libraries table requests
func (h *Handler) TautulliLibrariesTable(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[TableParams, *tautulli.TautulliLibrariesTable]{
		CacheName:   "TautulliLibrariesTable",
		ParseParams: parseLibrariesTableParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p TableParams) (*tautulli.TautulliLibrariesTable, error) {
			return h.client.GetLibrariesTable(ctx, p.Grouping, p.OrderColumn, p.OrderDir, p.Start, p.Length, p.Search)
		},
		ExtractData: func(resp *tautulli.TautulliLibrariesTable) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve libraries table from Tautulli",
	})
}

// TautulliLibraryMediaInfo handles Tautulli library media info requests
func (h *Handler) TautulliLibraryMediaInfo(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[LibraryMediaInfoParams, *tautulli.TautulliLibraryMediaInfo]{
		CacheName:   "TautulliLibraryMediaInfo",
		ParseParams: parseLibraryMediaInfoParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p LibraryMediaInfoParams) (*tautulli.TautulliLibraryMediaInfo, error) {
			return h.client.GetLibraryMediaInfo(ctx, p.SectionID, p.OrderColumn, p.OrderDir, p.Start, p.Length)
		},
		ExtractData: func(resp *tautulli.TautulliLibraryMediaInfo) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve library media info from Tautulli",
	})
}

// TautulliLibraryWatchTimeStats handles Tautulli library watch time statistics requests
func (h *Handler) TautulliLibraryWatchTimeStats(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[LibraryWatchTimeParams, *tautulli.TautulliLibraryWatchTimeStats]{
		CacheName:   "TautulliLibraryWatchTimeStats",
		ParseParams: parseLibraryWatchTimeStatsParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p LibraryWatchTimeParams) (*tautulli.TautulliLibraryWatchTimeStats, error) {
			return h.client.GetLibraryWatchTimeStats(ctx, p.SectionID, p.Grouping, p.QueryDays)
		},
		ExtractData: func(resp *tautulli.TautulliLibraryWatchTimeStats) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve library watch time stats from Tautulli",
	})
}

// TautulliChildrenMetadata handles Tautulli children metadata requests
func (h *Handler) TautulliChildrenMetadata(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[ChildrenMetadataParams, *tautulli.TautulliChildrenMetadata]{
		CacheName:   "TautulliChildrenMetadata",
		ParseParams: parseChildrenMetadataParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p ChildrenMetadataParams) (*tautulli.TautulliChildrenMetadata, error) {
			return h.client.GetChildrenMetadata(ctx, p.RatingKey, p.MediaType)
		},
		ExtractData: func(resp *tautulli.TautulliChildrenMetadata) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve children metadata from Tautulli",
	})
}

// TautulliUserIPs handles Tautulli user IPs requests
func (h *Handler) TautulliUserIPs(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleIntParam, *tautulli.TautulliUserIPs]{
		CacheName:   "TautulliUserIPs",
		ParseParams: parseUserIPsParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleIntParam) (*tautulli.TautulliUserIPs, error) {
			return h.client.GetUserIPs(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliUserIPs) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve user IPs from Tautulli",
	})
}

// TautulliUsersTable handles Tautulli users table requests
func (h *Handler) TautulliUsersTable(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[UsersTableParams, *tautulli.TautulliUsersTable]{
		CacheName:   "TautulliUsersTable",
		ParseParams: parseUsersTableParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p UsersTableParams) (*tautulli.TautulliUsersTable, error) {
			return h.client.GetUsersTable(ctx, p.Grouping, p.OrderColumn, p.OrderDir, p.Start, p.Length, p.Search)
		},
		ExtractData: func(resp *tautulli.TautulliUsersTable) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve users table from Tautulli",
	})
}

// TautulliUserLogins handles Tautulli user logins requests
func (h *Handler) TautulliUserLogins(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[UserLoginsParams, *tautulli.TautulliUserLogins]{
		CacheName:   "TautulliUserLogins",
		ParseParams: parseUserLoginsParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p UserLoginsParams) (*tautulli.TautulliUserLogins, error) {
			return h.client.GetUserLogins(ctx, p.UserID, p.OrderColumn, p.OrderDir, p.Start, p.Length, p.Search)
		},
		ExtractData: func(resp *tautulli.TautulliUserLogins) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve user logins from Tautulli",
	})
}

// TautulliStreamData handles Tautulli stream data requests
func (h *Handler) TautulliStreamData(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StreamDataParams, *tautulli.TautulliStreamData]{
		CacheName:   "", // No caching for stream data (real-time)
		ParseParams: parseStreamDataParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p StreamDataParams) (*tautulli.TautulliStreamData, error) {
			return h.client.GetStreamData(ctx, p.RowID, p.SessionKey)
		},
		ExtractData: func(resp *tautulli.TautulliStreamData) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve stream data from Tautulli",
	})
}

// TautulliLibraryNames handles Tautulli library names requests
func (h *Handler) TautulliLibraryNames(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliLibraryNames]{
		CacheName:   "TautulliLibraryNames",
		ParseParams: parseLibraryNamesParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliLibraryNames, error) {
			return h.client.GetLibraryNames(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliLibraryNames) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve library names from Tautulli",
	})
}

// TautulliExportMetadata handles Tautulli export metadata requests
func (h *Handler) TautulliExportMetadata(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[ExportMetadataParams, *tautulli.TautulliExportMetadata]{
		CacheName:   "", // No caching for exports
		ParseParams: parseExportMetadataParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p ExportMetadataParams) (*tautulli.TautulliExportMetadata, error) {
			return h.client.ExportMetadata(ctx, p.SectionID, p.ExportType, p.UserID, p.RatingKey, p.FileFormat)
		},
		ExtractData: func(resp *tautulli.TautulliExportMetadata) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to export metadata from Tautulli",
	})
}

// TautulliExportFields handles Tautulli export fields requests
func (h *Handler) TautulliExportFields(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleStringParam, *tautulli.TautulliExportFields]{
		CacheName:   "TautulliExportFields",
		ParseParams: parseExportFieldsParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleStringParam) (*tautulli.TautulliExportFields, error) {
			return h.client.GetExportFields(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliExportFields) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve export fields from Tautulli",
	})
}

// TautulliStreamTypeByTop10Users handles Tautulli stream type by top 10 users requests
func (h *Handler) TautulliStreamTypeByTop10Users(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliStreamTypeByTop10Users]{
		CacheName:   "TautulliStreamTypeByTop10Users",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliStreamTypeByTop10Users, error) {
			return h.client.GetStreamTypeByTop10Users(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliStreamTypeByTop10Users) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve stream type by top 10 users from Tautulli",
	})
}

// TautulliStreamTypeByTop10Platforms handles Tautulli stream type by top 10 platforms requests
func (h *Handler) TautulliStreamTypeByTop10Platforms(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[StandardTimeRangeParams, *tautulli.TautulliStreamTypeByTop10Platforms]{
		CacheName:   "TautulliStreamTypeByTop10Platforms",
		ParseParams: parseStandardTimeRangeParams,
		CallClient: func(ctx context.Context, h *Handler, p StandardTimeRangeParams) (*tautulli.TautulliStreamTypeByTop10Platforms, error) {
			return h.client.GetStreamTypeByTop10Platforms(ctx, p.TimeRange, p.YAxis, p.UserID, p.Grouping)
		},
		ExtractData: func(resp *tautulli.TautulliStreamTypeByTop10Platforms) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve stream type by top 10 platforms from Tautulli",
	})
}

// TautulliSearch handles Tautulli search requests
func (h *Handler) TautulliSearch(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SearchParams, *tautulli.TautulliSearch]{
		CacheName:   "", // No caching for search (user-specific queries)
		ParseParams: parseSearchParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SearchParams) (*tautulli.TautulliSearch, error) {
			return h.client.Search(ctx, p.Query, p.Limit)
		},
		ExtractData: func(resp *tautulli.TautulliSearch) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to perform search via Tautulli",
	})
}

// TautulliNewRatingKeys handles Tautulli new rating keys requests
func (h *Handler) TautulliNewRatingKeys(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleStringParam, *tautulli.TautulliNewRatingKeys]{
		CacheName:   "TautulliNewRatingKeys",
		ParseParams: parseNewRatingKeysParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleStringParam) (*tautulli.TautulliNewRatingKeys, error) {
			return h.client.GetNewRatingKeys(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliNewRatingKeys) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve new rating keys from Tautulli",
	})
}

// TautulliOldRatingKeys handles Tautulli old rating keys requests
func (h *Handler) TautulliOldRatingKeys(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleStringParam, *tautulli.TautulliOldRatingKeys]{
		CacheName:   "TautulliOldRatingKeys",
		ParseParams: parseOldRatingKeysParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleStringParam) (*tautulli.TautulliOldRatingKeys, error) {
			return h.client.GetOldRatingKeys(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliOldRatingKeys) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve old rating keys from Tautulli",
	})
}

// TautulliCollectionsTable handles Tautulli collections table requests
func (h *Handler) TautulliCollectionsTable(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[CollectionsTableParams, *tautulli.TautulliCollectionsTable]{
		CacheName:   "TautulliCollectionsTable",
		ParseParams: parseCollectionsTableParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p CollectionsTableParams) (*tautulli.TautulliCollectionsTable, error) {
			return h.client.GetCollectionsTable(ctx, p.SectionID, p.OrderColumn, p.OrderDir, p.Start, p.Length, p.Search)
		},
		ExtractData: func(resp *tautulli.TautulliCollectionsTable) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve collections table from Tautulli",
	})
}

// TautulliPlaylistsTable handles Tautulli playlists table requests
func (h *Handler) TautulliPlaylistsTable(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[PlaylistsTableParams, *tautulli.TautulliPlaylistsTable]{
		CacheName:   "TautulliPlaylistsTable",
		ParseParams: parsePlaylistsTableParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p PlaylistsTableParams) (*tautulli.TautulliPlaylistsTable, error) {
			return h.client.GetPlaylistsTable(ctx, p.SectionID, p.OrderColumn, p.OrderDir, p.Start, p.Length, p.Search)
		},
		ExtractData: func(resp *tautulli.TautulliPlaylistsTable) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve playlists table from Tautulli",
	})
}

// TautulliServerFriendlyName handles Tautulli server friendly name requests
func (h *Handler) TautulliServerFriendlyName(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliServerFriendlyName]{
		CacheName:   "TautulliServerFriendlyName",
		ParseParams: parseServerFriendlyNameParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliServerFriendlyName, error) {
			return h.client.GetServerFriendlyName(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliServerFriendlyName) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve server friendly name from Tautulli",
	})
}

// TautulliServerID handles Tautulli server ID requests
func (h *Handler) TautulliServerID(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliServerID]{
		CacheName:   "TautulliServerID",
		ParseParams: parseServerIDParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliServerID, error) {
			return h.client.GetServerID(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliServerID) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve server ID from Tautulli",
	})
}

// TautulliServerIdentity handles Tautulli server identity requests
func (h *Handler) TautulliServerIdentity(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliServerIdentity]{
		CacheName:   "TautulliServerIdentity",
		ParseParams: parseServerIdentityParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliServerIdentity, error) {
			return h.client.GetServerIdentity(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliServerIdentity) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve server identity from Tautulli",
	})
}

// TautulliExportsTable handles Tautulli exports table requests
func (h *Handler) TautulliExportsTable(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[ExportsTableParams, *tautulli.TautulliExportsTable]{
		CacheName:   "", // No caching for exports list (may change frequently)
		ParseParams: parseExportsTableParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p ExportsTableParams) (*tautulli.TautulliExportsTable, error) {
			return h.client.GetExportsTable(ctx, p.OrderColumn, p.OrderDir, p.Start, p.Length, p.Search)
		},
		ExtractData: func(resp *tautulli.TautulliExportsTable) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve exports table from Tautulli",
	})
}

// TautulliDownloadExport handles Tautulli download export requests
func (h *Handler) TautulliDownloadExport(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleIntParam, *tautulli.TautulliDownloadExport]{
		CacheName:   "", // No caching for downloads
		ParseParams: parseDownloadExportParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleIntParam) (*tautulli.TautulliDownloadExport, error) {
			return h.client.DownloadExport(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliDownloadExport) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to download export from Tautulli",
	})
}

// TautulliDeleteExport handles Tautulli delete export requests
func (h *Handler) TautulliDeleteExport(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleIntParam, *tautulli.TautulliDeleteExport]{
		CacheName:   "", // No caching for delete operations
		ParseParams: parseDeleteExportParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleIntParam) (*tautulli.TautulliDeleteExport, error) {
			return h.client.DeleteExport(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliDeleteExport) interface{} {
			return resp.Response // Note: uses .Response not .Response.Data
		},
		ErrorMessage: "Failed to delete export from Tautulli",
	})
}

// TautulliTautulliInfo handles Tautulli info requests
func (h *Handler) TautulliTautulliInfo(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliTautulliInfo]{
		CacheName:   "TautulliTautulliInfo",
		ParseParams: parseTautulliInfoParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliTautulliInfo, error) {
			return h.client.GetTautulliInfo(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliTautulliInfo) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve Tautulli info from Tautulli",
	})
}

// TautulliServerPref handles Tautulli server preference requests
func (h *Handler) TautulliServerPref(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[SingleStringParam, *tautulli.TautulliServerPref]{
		CacheName:   "TautulliServerPref",
		ParseParams: parseServerPrefParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p SingleStringParam) (*tautulli.TautulliServerPref, error) {
			return h.client.GetServerPref(ctx, p.Value)
		},
		ExtractData: func(resp *tautulli.TautulliServerPref) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve server preference from Tautulli",
	})
}

// TautulliServerList handles Tautulli server list requests
func (h *Handler) TautulliServerList(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliServerList]{
		CacheName:   "TautulliServerList",
		ParseParams: parseServerListParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliServerList, error) {
			return h.client.GetServerList(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliServerList) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve server list from Tautulli",
	})
}

// TautulliServersInfo handles Tautulli servers info requests
func (h *Handler) TautulliServersInfo(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliServersInfo]{
		CacheName:   "TautulliServersInfo",
		ParseParams: parseServersInfoParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliServersInfo, error) {
			return h.client.GetServersInfo(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliServersInfo) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve servers info from Tautulli",
	})
}

// TautulliPMSUpdate handles Tautulli PMS update requests
func (h *Handler) TautulliPMSUpdate(w http.ResponseWriter, r *http.Request) {
	proxyTautulliRequest(h, w, r, TautulliProxyConfig[NoParams, *tautulli.TautulliPMSUpdate]{
		CacheName:   "", // No caching for update checks
		ParseParams: parsePMSUpdateParamsTyped,
		CallClient: func(ctx context.Context, h *Handler, p NoParams) (*tautulli.TautulliPMSUpdate, error) {
			return h.client.GetPMSUpdate(ctx)
		},
		ExtractData: func(resp *tautulli.TautulliPMSUpdate) interface{} {
			return resp.Response.Data
		},
		ErrorMessage: "Failed to retrieve PMS update info from Tautulli",
	})
}
