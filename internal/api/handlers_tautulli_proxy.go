// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models"
)

// TautulliProxyConfig defines configuration for a Tautulli API proxy endpoint
// P is the parameter type (e.g., StandardTimeRangeParams)
// R is the response type (e.g., *tautulli.TautulliPlaysByDate)
type TautulliProxyConfig[P any, R any] struct {
	// CacheName is the name used for cache key generation (empty = no caching)
	CacheName string

	// HTTPMethod is the allowed HTTP method (default: GET)
	HTTPMethod string

	// ParseParams extracts and validates parameters from the HTTP request
	// Returns strongly-typed parameters and an error if validation fails
	ParseParams func(r *http.Request) (P, error)

	// CallClient invokes the Tautulli client method with the parsed parameters
	// Returns the strongly-typed response and an error
	CallClient func(ctx context.Context, h *Handler, params P) (R, error)

	// ExtractData extracts the data field from the Tautulli response
	// Some responses use Response.Data, others use Response directly
	ExtractData func(response R) interface{}

	// ErrorMessage is the error message prefix for client call failures
	ErrorMessage string
}

// proxyTautulliRequest is a generic function for Tautulli API proxy endpoints
// It implements the common pattern: method check, parameter parsing, caching,
// client call, and JSON response
// P is the parameter type, R is the response type
func proxyTautulliRequest[P any, R any](h *Handler, w http.ResponseWriter, r *http.Request, config TautulliProxyConfig[P, R]) {
	// Set default HTTP method if not specified
	httpMethod := config.HTTPMethod
	if httpMethod == "" {
		httpMethod = http.MethodGet
	}

	// Check HTTP method
	if r.Method != httpMethod {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	start := time.Now()

	// Parse and validate parameters FIRST (validation errors = 400)
	params, err := config.ParseParams(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_PARAMETER", err.Error(), nil)
		return
	}

	// Check if Tautulli client is available AFTER validation (service errors = 503)
	if h.client == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Tautulli client not available", nil)
		return
	}

	// Check cache if caching is enabled AND cache is available
	if config.CacheName != "" && h.cache != nil {
		cacheKey := cache.GenerateKey(config.CacheName, params)
		if cached, found := h.cache.Get(cacheKey); found {
			// Type assert cached value safely
			if cachedResponse, ok := cached.(R); ok {
				respondJSON(w, http.StatusOK, &models.APIResponse{
					Status: "success",
					Data:   config.ExtractData(cachedResponse),
					Metadata: models.Metadata{
						Timestamp:   time.Now(),
						QueryTimeMS: time.Since(start).Milliseconds(),
					},
				})
				return
			}
			// If type assertion fails, fall through to fetch fresh data
		}
	}

	// Call Tautulli client
	response, err := config.CallClient(r.Context(), h, params)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "TAUTULLI_ERROR",
			config.ErrorMessage, err)
		return
	}

	// Check if response is nil (defensive check for nil pointer responses)
	// This uses reflection because R is a generic type parameter that could be a pointer
	if isNilResponse(response) {
		respondError(w, http.StatusInternalServerError, "TAUTULLI_ERROR",
			"Received nil response from Tautulli", nil)
		return
	}

	// Cache the result if caching is enabled AND cache is available
	if config.CacheName != "" && h.cache != nil {
		cacheKey := cache.GenerateKey(config.CacheName, params)
		h.cache.Set(cacheKey, response)
	}

	// Respond with JSON
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   config.ExtractData(response),
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// Common parameter parsing helpers

// parseStandardTimeRangeParams parses time_range, y_axis, user_id, grouping (most common pattern)
func parseStandardTimeRangeParams(r *http.Request) (StandardTimeRangeParams, error) {
	timeRange := parseIntParam(r.URL.Query().Get("time_range"), 30)
	yAxis := r.URL.Query().Get("y_axis")
	if yAxis == "" {
		yAxis = "plays"
	}
	userID := parseIntParam(r.URL.Query().Get("user_id"), 0)
	grouping := parseIntParam(r.URL.Query().Get("grouping"), 0)

	return StandardTimeRangeParams{
		TimeRange: timeRange,
		YAxis:     yAxis,
		UserID:    userID,
		Grouping:  grouping,
	}, nil
}

// parseTimeRangeUserIDParams parses time_range and user_id
func parseTimeRangeUserIDParams(r *http.Request) (TwoIntParams, error) {
	timeRange := parseIntParam(r.URL.Query().Get("time_range"), 30)
	userID := parseIntParam(r.URL.Query().Get("user_id"), 0)

	return TwoIntParams{
		Param1: timeRange,
		Param2: userID,
	}, nil
}

// parseHomeStatsParams parses parameters for TautulliHomeStats
func parseHomeStatsParams(r *http.Request) (HomeStatsParams, error) {
	timeRange := parseIntParam(r.URL.Query().Get("time_range"), 30)
	statsType := r.URL.Query().Get("stats_type")
	if statsType == "" {
		statsType = "plays"
	}
	statsCount := parseIntParam(r.URL.Query().Get("stats_count"), 10)

	return HomeStatsParams{
		TimeRange:  timeRange,
		StatsType:  statsType,
		StatsCount: statsCount,
	}, nil
}

// Typed parsers for all parameter structs (alphabetically ordered)

// parseActivityParams parses parameters for TautulliActivity
func parseActivityParamsTyped(r *http.Request) (SingleStringParam, error) {
	sessionKey := r.URL.Query().Get("session_key")
	return SingleStringParam{Value: sessionKey}, nil
}

// parseChildrenMetadataParamsTyped parses parameters for TautulliChildrenMetadata
func parseChildrenMetadataParamsTyped(r *http.Request) (ChildrenMetadataParams, error) {
	ratingKey := r.URL.Query().Get("rating_key")
	if ratingKey == "" {
		return ChildrenMetadataParams{}, fmt.Errorf("rating_key parameter is required")
	}
	mediaType := r.URL.Query().Get("media_type")
	return ChildrenMetadataParams{RatingKey: ratingKey, MediaType: mediaType}, nil
}

// parseCollectionsTableParamsTyped parses parameters for TautulliCollectionsTable
func parseCollectionsTableParamsTyped(r *http.Request) (CollectionsTableParams, error) {
	sectionID := parseIntParam(r.URL.Query().Get("section_id"), 0)
	orderColumn := r.URL.Query().Get("order_column")
	if orderColumn == "" {
		orderColumn = "title"
	}
	orderDir := r.URL.Query().Get("order_dir")
	if orderDir == "" {
		orderDir = "asc"
	}
	start := parseIntParam(r.URL.Query().Get("start"), 0)
	length := parseIntParam(r.URL.Query().Get("length"), 25)
	search := r.URL.Query().Get("search")

	return CollectionsTableParams{
		SectionID:   sectionID,
		OrderColumn: orderColumn,
		OrderDir:    orderDir,
		Start:       start,
		Length:      length,
		Search:      search,
	}, nil
}

// parseDeleteExportParamsTyped parses parameters for TautulliDeleteExport
func parseDeleteExportParamsTyped(r *http.Request) (SingleIntParam, error) {
	exportID := parseIntParam(r.URL.Query().Get("export_id"), 0)
	if exportID == 0 {
		return SingleIntParam{}, fmt.Errorf("export_id parameter is required and must be valid")
	}
	return SingleIntParam{Value: exportID}, nil
}

// parseDownloadExportParamsTyped parses parameters for TautulliDownloadExport
func parseDownloadExportParamsTyped(r *http.Request) (SingleIntParam, error) {
	exportID := parseIntParam(r.URL.Query().Get("export_id"), 0)
	if exportID == 0 {
		return SingleIntParam{}, fmt.Errorf("export_id parameter is required and must be valid")
	}
	return SingleIntParam{Value: exportID}, nil
}

// parseExportFieldsParamsTyped parses parameters for TautulliExportFields
func parseExportFieldsParamsTyped(r *http.Request) (SingleStringParam, error) {
	mediaType := r.URL.Query().Get("media_type")
	return SingleStringParam{Value: mediaType}, nil
}

// parseExportMetadataParamsTyped parses parameters for TautulliExportMetadata
func parseExportMetadataParamsTyped(r *http.Request) (ExportMetadataParams, error) {
	sectionID := parseIntParam(r.URL.Query().Get("section_id"), 0)
	exportType := r.URL.Query().Get("export_type")
	userID := parseIntParam(r.URL.Query().Get("user_id"), 0)
	ratingKey := r.URL.Query().Get("rating_key")
	fileFormat := r.URL.Query().Get("file_format")

	return ExportMetadataParams{
		SectionID:  sectionID,
		ExportType: exportType,
		UserID:     userID,
		RatingKey:  ratingKey,
		FileFormat: fileFormat,
	}, nil
}

// parseExportsTableParamsTyped parses parameters for TautulliExportsTable
func parseExportsTableParamsTyped(r *http.Request) (ExportsTableParams, error) {
	orderColumn := r.URL.Query().Get("order_column")
	if orderColumn == "" {
		orderColumn = "timestamp"
	}
	orderDir := r.URL.Query().Get("order_dir")
	if orderDir == "" {
		orderDir = "desc"
	}
	start := parseIntParam(r.URL.Query().Get("start"), 0)
	length := parseIntParam(r.URL.Query().Get("length"), 25)
	search := r.URL.Query().Get("search")

	return ExportsTableParams{
		OrderColumn: orderColumn,
		OrderDir:    orderDir,
		Start:       start,
		Length:      length,
		Search:      search,
	}, nil
}

// parseItemUserStatsParamsTyped parses parameters for TautulliItemUserStats
func parseItemUserStatsParamsTyped(r *http.Request) (ItemUserStatsParams, error) {
	ratingKey := r.URL.Query().Get("rating_key")
	if ratingKey == "" {
		return ItemUserStatsParams{}, fmt.Errorf("rating_key parameter is required")
	}
	grouping := parseIntParam(r.URL.Query().Get("grouping"), 0)
	return ItemUserStatsParams{RatingKey: ratingKey, Grouping: grouping}, nil
}

// parseItemWatchTimeStatsParamsTyped parses parameters for TautulliItemWatchTimeStats
func parseItemWatchTimeStatsParamsTyped(r *http.Request) (ItemWatchTimeParams, error) {
	ratingKey := r.URL.Query().Get("rating_key")
	if ratingKey == "" {
		return ItemWatchTimeParams{}, fmt.Errorf("rating_key parameter is required")
	}
	grouping := parseIntParam(r.URL.Query().Get("grouping"), 0)
	queryDays := r.URL.Query().Get("query_days")
	if queryDays == "" {
		queryDays = "30"
	}

	return ItemWatchTimeParams{RatingKey: ratingKey, Grouping: grouping, QueryDays: queryDays}, nil
}

// parseLibrariesParamsTyped parses parameters for TautulliLibraries (no parameters)
func parseLibrariesParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// parseLibrariesTableParamsTyped parses parameters for TautulliLibrariesTable
func parseLibrariesTableParamsTyped(r *http.Request) (TableParams, error) {
	grouping := parseIntParam(r.URL.Query().Get("grouping"), 0)
	orderColumn := r.URL.Query().Get("order_column")
	if orderColumn == "" {
		orderColumn = "total_plays"
	}
	orderDir := r.URL.Query().Get("order_dir")
	if orderDir == "" {
		orderDir = "desc"
	}
	start := parseIntParam(r.URL.Query().Get("start"), 0)
	length := parseIntParam(r.URL.Query().Get("length"), 25)
	search := r.URL.Query().Get("search")

	return TableParams{
		Grouping:    grouping,
		OrderColumn: orderColumn,
		OrderDir:    orderDir,
		Start:       start,
		Length:      length,
		Search:      search,
	}, nil
}

// parseLibraryParamsTyped parses parameters for TautulliLibrary
func parseLibraryParamsTyped(r *http.Request) (SingleIntParam, error) {
	sectionID := parseIntParam(r.URL.Query().Get("section_id"), 0)
	if sectionID == 0 {
		return SingleIntParam{}, fmt.Errorf("section_id parameter is required and must be valid")
	}
	return SingleIntParam{Value: sectionID}, nil
}

// parseLibraryMediaInfoParamsTyped parses parameters for TautulliLibraryMediaInfo
func parseLibraryMediaInfoParamsTyped(r *http.Request) (LibraryMediaInfoParams, error) {
	sectionID := parseIntParam(r.URL.Query().Get("section_id"), 0)
	orderColumn := r.URL.Query().Get("order_column")
	if orderColumn == "" {
		orderColumn = "title"
	}
	orderDir := r.URL.Query().Get("order_dir")
	if orderDir == "" {
		orderDir = "asc"
	}
	start := parseIntParam(r.URL.Query().Get("start"), 0)
	length := parseIntParam(r.URL.Query().Get("length"), 25)

	return LibraryMediaInfoParams{
		SectionID:   sectionID,
		OrderColumn: orderColumn,
		OrderDir:    orderDir,
		Start:       start,
		Length:      length,
	}, nil
}

// parseLibraryNamesParamsTyped parses parameters for TautulliLibraryNames (no parameters)
func parseLibraryNamesParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// parseLibraryUserStatsParamsTyped parses parameters for TautulliLibraryUserStats
func parseLibraryUserStatsParamsTyped(r *http.Request) (TwoIntParams, error) {
	sectionID := parseIntParam(r.URL.Query().Get("section_id"), 0)
	grouping := parseIntParam(r.URL.Query().Get("grouping"), 0)
	return TwoIntParams{Param1: sectionID, Param2: grouping}, nil
}

// parseLibraryWatchTimeStatsParamsTyped parses parameters for TautulliLibraryWatchTimeStats
func parseLibraryWatchTimeStatsParamsTyped(r *http.Request) (LibraryWatchTimeParams, error) {
	sectionID := parseIntParam(r.URL.Query().Get("section_id"), 0)
	grouping := parseIntParam(r.URL.Query().Get("grouping"), 0)
	queryDays := r.URL.Query().Get("query_days")
	if queryDays == "" {
		queryDays = "30"
	}

	return LibraryWatchTimeParams{
		SectionID: sectionID,
		Grouping:  grouping,
		QueryDays: queryDays,
	}, nil
}

// parseMetadataParamsTyped parses parameters for TautulliMetadata
func parseMetadataParamsTyped(r *http.Request) (SingleStringParam, error) {
	ratingKey := r.URL.Query().Get("rating_key")
	if ratingKey == "" {
		return SingleStringParam{}, fmt.Errorf("rating_key parameter is required")
	}
	return SingleStringParam{Value: ratingKey}, nil
}

// parseNewRatingKeysParamsTyped parses parameters for TautulliNewRatingKeys
func parseNewRatingKeysParamsTyped(r *http.Request) (SingleStringParam, error) {
	ratingKey := r.URL.Query().Get("rating_key")
	return SingleStringParam{Value: ratingKey}, nil
}

// parseOldRatingKeysParamsTyped parses parameters for TautulliOldRatingKeys
func parseOldRatingKeysParamsTyped(r *http.Request) (SingleStringParam, error) {
	ratingKey := r.URL.Query().Get("rating_key")
	return SingleStringParam{Value: ratingKey}, nil
}

// parsePMSUpdateParamsTyped parses parameters for TautulliPMSUpdate (no parameters)
func parsePMSUpdateParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// parsePlaylistsTableParamsTyped parses parameters for TautulliPlaylistsTable
func parsePlaylistsTableParamsTyped(r *http.Request) (PlaylistsTableParams, error) {
	sectionID := parseIntParam(r.URL.Query().Get("section_id"), 0)
	orderColumn := r.URL.Query().Get("order_column")
	if orderColumn == "" {
		orderColumn = "title"
	}
	orderDir := r.URL.Query().Get("order_dir")
	if orderDir == "" {
		orderDir = "asc"
	}
	start := parseIntParam(r.URL.Query().Get("start"), 0)
	length := parseIntParam(r.URL.Query().Get("length"), 25)
	search := r.URL.Query().Get("search")

	return PlaylistsTableParams{
		SectionID:   sectionID,
		OrderColumn: orderColumn,
		OrderDir:    orderDir,
		Start:       start,
		Length:      length,
		Search:      search,
	}, nil
}

// parseRecentlyAddedParamsTyped parses parameters for TautulliRecentlyAdded
func parseRecentlyAddedParamsTyped(r *http.Request) (RecentlyAddedParams, error) {
	count := parseIntParam(r.URL.Query().Get("count"), 25)
	start := parseIntParam(r.URL.Query().Get("start"), 0)
	mediaType := r.URL.Query().Get("media_type")
	sectionID := parseIntParam(r.URL.Query().Get("section_id"), 0)

	return RecentlyAddedParams{
		Count:     count,
		Start:     start,
		MediaType: mediaType,
		SectionID: sectionID,
	}, nil
}

// parseSearchParamsTyped parses parameters for TautulliSearch
func parseSearchParamsTyped(r *http.Request) (SearchParams, error) {
	query := r.URL.Query().Get("query")
	if query == "" {
		return SearchParams{}, fmt.Errorf("query parameter is required")
	}
	limit := parseIntParam(r.URL.Query().Get("limit"), 25)

	return SearchParams{Query: query, Limit: limit}, nil
}

// parseServerFriendlyNameParamsTyped parses parameters for TautulliServerFriendlyName (no parameters)
func parseServerFriendlyNameParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// parseServerIDParamsTyped parses parameters for TautulliServerID (no parameters)
func parseServerIDParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// parseServerIdentityParamsTyped parses parameters for TautulliServerIdentity (no parameters)
func parseServerIdentityParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// parseServerInfoParamsTyped parses parameters for TautulliServerInfo (no parameters)
func parseServerInfoParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// parseServerListParamsTyped parses parameters for TautulliServerList (no parameters)
func parseServerListParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// parseServerPrefParamsTyped parses parameters for TautulliServerPref
func parseServerPrefParamsTyped(r *http.Request) (SingleStringParam, error) {
	pref := r.URL.Query().Get("pref")
	if pref == "" {
		return SingleStringParam{}, fmt.Errorf("pref parameter is required")
	}
	return SingleStringParam{Value: pref}, nil
}

// parseServersInfoParamsTyped parses parameters for TautulliServersInfo (no parameters)
func parseServersInfoParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// parseStreamDataParamsTyped parses parameters for TautulliStreamData
func parseStreamDataParamsTyped(r *http.Request) (StreamDataParams, error) {
	rowID := parseIntParam(r.URL.Query().Get("row_id"), 0)
	sessionKey := r.URL.Query().Get("session_key")

	return StreamDataParams{RowID: rowID, SessionKey: sessionKey}, nil
}

// parseSyncedItemsParamsTyped parses parameters for TautulliSyncedItems
func parseSyncedItemsParamsTyped(r *http.Request) (SyncedItemsParams, error) {
	machineID := r.URL.Query().Get("machine_id")
	userID := parseIntParam(r.URL.Query().Get("user_id"), 0)
	return SyncedItemsParams{MachineID: machineID, UserID: userID}, nil
}

// parseTautulliInfoParamsTyped parses parameters for TautulliTautulliInfo (no parameters)
func parseTautulliInfoParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// parseTerminateSessionParamsTyped parses parameters for TautulliTerminateSession
func parseTerminateSessionParamsTyped(r *http.Request) (TerminateSessionParams, error) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		return TerminateSessionParams{}, fmt.Errorf("session_id parameter is required")
	}
	message := r.URL.Query().Get("message")
	return TerminateSessionParams{SessionID: sessionID, Message: message}, nil
}

// parseUserIPsParamsTyped parses parameters for TautulliUserIPs
func parseUserIPsParamsTyped(r *http.Request) (SingleIntParam, error) {
	userID := parseIntParam(r.URL.Query().Get("user_id"), 0)
	return SingleIntParam{Value: userID}, nil
}

// parseUserLoginsParamsTyped parses parameters for TautulliUserLogins
func parseUserLoginsParamsTyped(r *http.Request) (UserLoginsParams, error) {
	userID := parseIntParam(r.URL.Query().Get("user_id"), 0)
	orderColumn := r.URL.Query().Get("order_column")
	if orderColumn == "" {
		orderColumn = "date"
	}
	orderDir := r.URL.Query().Get("order_dir")
	if orderDir == "" {
		orderDir = "desc"
	}
	start := parseIntParam(r.URL.Query().Get("start"), 0)
	length := parseIntParam(r.URL.Query().Get("length"), 25)
	search := r.URL.Query().Get("search")

	return UserLoginsParams{
		UserID:      userID,
		OrderColumn: orderColumn,
		OrderDir:    orderDir,
		Start:       start,
		Length:      length,
		Search:      search,
	}, nil
}

// parseUserParamsTyped parses parameters for TautulliUser
func parseUserParamsTyped(r *http.Request) (SingleIntParam, error) {
	userID := parseIntParam(r.URL.Query().Get("user_id"), 0)
	if userID == 0 {
		return SingleIntParam{}, fmt.Errorf("user_id parameter is required and must be valid")
	}
	return SingleIntParam{Value: userID}, nil
}

// parseUserPlayerStatsParamsTyped parses parameters for TautulliUserPlayerStats
func parseUserPlayerStatsParamsTyped(r *http.Request) (SingleIntParam, error) {
	userID := parseIntParam(r.URL.Query().Get("user_id"), 0)
	return SingleIntParam{Value: userID}, nil
}

// parseUsersTableParamsTyped parses parameters for TautulliUsersTable
func parseUsersTableParamsTyped(r *http.Request) (UsersTableParams, error) {
	grouping := parseIntParam(r.URL.Query().Get("grouping"), 0)
	orderColumn := r.URL.Query().Get("order_column")
	if orderColumn == "" {
		orderColumn = "friendly_name"
	}
	orderDir := r.URL.Query().Get("order_dir")
	if orderDir == "" {
		orderDir = "asc"
	}
	start := parseIntParam(r.URL.Query().Get("start"), 0)
	length := parseIntParam(r.URL.Query().Get("length"), 25)
	search := r.URL.Query().Get("search")

	return UsersTableParams{
		Grouping:    grouping,
		OrderColumn: orderColumn,
		OrderDir:    orderDir,
		Start:       start,
		Length:      length,
		Search:      search,
	}, nil
}

// parseUserWatchTimeStatsParamsTyped parses parameters for TautulliUserWatchTimeStats
func parseUserWatchTimeStatsParamsTyped(r *http.Request) (UserWatchTimeParams, error) {
	userID := parseIntParam(r.URL.Query().Get("user_id"), 0)
	queryDays := r.URL.Query().Get("query_days")
	if queryDays == "" {
		queryDays = "30"
	}
	return UserWatchTimeParams{UserID: userID, QueryDays: queryDays}, nil
}

// parseUsersParamsTyped parses parameters for TautulliUsers (bulk user fetch)
// This endpoint takes no parameters
func parseUsersParamsTyped(r *http.Request) (NoParams, error) {
	return NoParams{}, nil
}

// Common data extraction helpers
//
// Note: The ExtractData function in TautulliProxyConfig should be a closure
// that knows the specific type and extracts the appropriate field.
// Most Tautulli responses use response.Response.Data
// Some (like TerminateSession) use response.Response directly

// isNilResponse checks if a generic response value is nil.
// This is necessary because Go's generics don't allow direct nil comparison
// for type parameters. When R is a pointer type like *SomeStruct, we need
// reflection to properly detect a nil pointer value.
func isNilResponse(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return rv.IsNil()
	}
	return false
}
