// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ============================================================================
// Bandwidth Statistics Tests - GET /statistics/bandwidth
// ============================================================================

// TestPlexClientGetBandwidthStatistics tests successful bandwidth statistics retrieval
func TestPlexClientGetBandwidthStatistics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/statistics/bandwidth")
		verifyPlexHeaders(t, r)
		checkStringEqual(t, "Accept header", r.Header.Get("Accept"), "application/json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(bandwidthStatsResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	stats, err := client.GetBandwidthStatistics(context.Background(), nil)

	checkNoError(t, err)
	checkSliceLen(t, "Device", len(stats.MediaContainer.Device), 2)
	checkSliceLen(t, "Account", len(stats.MediaContainer.Account), 1)
	checkSliceLen(t, "StatisticsBandwidth", len(stats.MediaContainer.StatisticsBandwidth), 3)

	// Verify device data
	device := stats.MediaContainer.Device[0]
	checkStringEqual(t, "Device.Name", device.Name, "Roku Express")
	checkStringEqual(t, "Device.Platform", device.Platform, "Roku")

	// Verify account data
	account := stats.MediaContainer.Account[0]
	checkStringEqual(t, "Account.Name", account.Name, "TestUser")

	// Verify bandwidth record
	bw := stats.MediaContainer.StatisticsBandwidth[0]
	checkIntEqual(t, "StatisticsBandwidth.AccountID", bw.AccountID, 1)
	checkInt64Equal(t, "StatisticsBandwidth.Bytes", bw.Bytes, 1073741824)
	checkTrue(t, "StatisticsBandwidth.LAN should be true", bw.LAN)
}

const bandwidthStatsResponse = `{
	"MediaContainer": {
		"size": 3,
		"Device": [
			{
				"id": 1,
				"name": "Roku Express",
				"platform": "Roku",
				"clientIdentifier": "client-123",
				"createdAt": 1700000000
			},
			{
				"id": 2,
				"name": "iPhone",
				"platform": "iOS",
				"clientIdentifier": "client-456",
				"createdAt": 1700001000
			}
		],
		"Account": [
			{
				"id": 1,
				"key": "/accounts/1",
				"name": "TestUser",
				"defaultAudioLanguage": "en",
				"thumb": "https://plex.tv/users/abc/avatar"
			}
		],
		"StatisticsBandwidth": [
			{
				"accountID": 1,
				"deviceID": 1,
				"timespan": 3600,
				"at": 1700000000,
				"lan": true,
				"bytes": 1073741824
			},
			{
				"accountID": 1,
				"deviceID": 1,
				"timespan": 3600,
				"at": 1700003600,
				"lan": false,
				"bytes": 536870912
			},
			{
				"accountID": 1,
				"deviceID": 2,
				"timespan": 3600,
				"at": 1700007200,
				"lan": true,
				"bytes": 268435456
			}
		]
	}
}`

// TestPlexClientGetBandwidthStatisticsWithTimespan tests bandwidth with timespan filter
func TestPlexClientGetBandwidthStatisticsWithTimespan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "timespan parameter", r.URL.Query().Get("timespan"), "86400")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0, "StatisticsBandwidth": []}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	timespan := 86400 // 1 day
	_, err := client.GetBandwidthStatistics(context.Background(), &timespan)
	checkNoError(t, err)
}

// TestPlexClientGetBandwidthStatisticsError tests error handling
func TestPlexClientGetBandwidthStatisticsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	_, err := client.GetBandwidthStatistics(context.Background(), nil)
	checkError(t, err)
}

// ============================================================================
// Library Sections Tests - GET /library/sections
// ============================================================================

// TestPlexClientGetLibrarySections tests successful library sections retrieval
func TestPlexClientGetLibrarySections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/library/sections")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(librarySectionsResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	sections, err := client.GetLibrarySections(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Directory", len(sections.MediaContainer.Directory), 3)

	// Verify Movies section
	movies := sections.MediaContainer.Directory[0]
	checkStringEqual(t, "Movies.Title", movies.Title, "Movies")
	checkStringEqual(t, "Movies.Type", movies.Type, "movie")
	checkStringEqual(t, "Movies.Key", movies.Key, "1")
	checkTrue(t, "Movies.IsMovie()", movies.IsMovie())

	// Verify TV Shows section
	tvShows := sections.MediaContainer.Directory[1]
	checkStringEqual(t, "TV.Title", tvShows.Title, "TV Shows")
	checkStringEqual(t, "TV.Type", tvShows.Type, "show")
	checkTrue(t, "TV.IsTV()", tvShows.IsTV())

	// Verify Music section
	music := sections.MediaContainer.Directory[2]
	checkStringEqual(t, "Music.Title", music.Title, "Music")
	checkTrue(t, "Music.IsMusic()", music.IsMusic())
}

const librarySectionsResponse = `{
	"MediaContainer": {
		"size": 3,
		"allowSync": true,
		"title1": "Plex Library",
		"Directory": [
			{
				"key": "1",
				"uuid": "abc-123",
				"title": "Movies",
				"type": "movie",
				"agent": "tv.plex.agents.movie",
				"scanner": "Plex Movie",
				"language": "en-US",
				"refreshing": false,
				"hidden": 0,
				"createdAt": 1600000000,
				"updatedAt": 1700000000,
				"scannedAt": 1700001000,
				"Location": [
					{"id": 1, "path": "/media/movies"}
				]
			},
			{
				"key": "2",
				"uuid": "def-456",
				"title": "TV Shows",
				"type": "show",
				"agent": "tv.plex.agents.series",
				"scanner": "Plex TV Series",
				"language": "en-US",
				"refreshing": true,
				"hidden": 0,
				"Location": [
					{"id": 2, "path": "/media/tv"}
				]
			},
			{
				"key": "3",
				"uuid": "ghi-789",
				"title": "Music",
				"type": "artist",
				"agent": "tv.plex.agents.music",
				"scanner": "Plex Music",
				"language": "en-US",
				"refreshing": false,
				"hidden": 0,
				"Location": [
					{"id": 3, "path": "/media/music"}
				]
			}
		]
	}
}`

// TestPlexClientGetLibrarySectionsError tests error handling
func TestPlexClientGetLibrarySectionsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	_, err := client.GetLibrarySections(context.Background())
	checkError(t, err)
}

// TestPlexClientGetLibrarySectionsEmpty tests empty sections response
func TestPlexClientGetLibrarySectionsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	sections, err := client.GetLibrarySections(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Directory", len(sections.MediaContainer.Directory), 0)
}

// ============================================================================
// Library Section Content Tests - GET /library/sections/{id}/all
// ============================================================================

// TestPlexClientGetLibrarySectionContent tests content retrieval
func TestPlexClientGetLibrarySectionContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkTrue(t, "path contains /library/sections/1/all",
			strings.HasPrefix(r.URL.Path, "/library/sections/1/all"))
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(librarySectionContentResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	content, err := client.GetLibrarySectionContent(context.Background(), "1", nil, nil)

	checkNoError(t, err)
	checkSliceLen(t, "Metadata", len(content.MediaContainer.Metadata), 2)
	checkIntEqual(t, "TotalSize", content.MediaContainer.TotalSize, 150)

	// Verify first movie
	movie := content.MediaContainer.Metadata[0]
	checkStringEqual(t, "Movie.Title", movie.Title, "Inception")
	checkStringEqual(t, "Movie.Type", movie.Type, "movie")
	checkIntEqual(t, "Movie.Year", movie.Year, 2010)
}

const librarySectionContentResponse = `{
	"MediaContainer": {
		"size": 2,
		"totalSize": 150,
		"offset": 0,
		"allowSync": true,
		"librarySectionID": 1,
		"librarySectionTitle": "Movies",
		"librarySectionUUID": "abc-123",
		"Metadata": [
			{
				"ratingKey": "12345",
				"key": "/library/metadata/12345",
				"type": "movie",
				"title": "Inception",
				"titleSort": "Inception",
				"contentRating": "PG-13",
				"summary": "A thief who steals corporate secrets...",
				"rating": 8.8,
				"audienceRating": 9.1,
				"year": 2010,
				"thumb": "/library/metadata/12345/thumb",
				"duration": 8880000,
				"addedAt": 1600000000,
				"updatedAt": 1700000000,
				"Media": [
					{
						"id": 1,
						"duration": 8880000,
						"bitrate": 10000,
						"width": 1920,
						"height": 1080,
						"videoCodec": "h264",
						"audioCodec": "aac",
						"videoResolution": "1080",
						"container": "mkv"
					}
				]
			},
			{
				"ratingKey": "12346",
				"key": "/library/metadata/12346",
				"type": "movie",
				"title": "The Matrix",
				"year": 1999,
				"thumb": "/library/metadata/12346/thumb",
				"duration": 8160000
			}
		]
	}
}`

// TestPlexClientGetLibrarySectionContentWithPagination tests pagination parameters
func TestPlexClientGetLibrarySectionContentWithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "start parameter", r.URL.Query().Get("X-Plex-Container-Start"), "10")
		checkStringEqual(t, "size parameter", r.URL.Query().Get("X-Plex-Container-Size"), "50")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0, "Metadata": []}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	start, size := 10, 50
	_, err := client.GetLibrarySectionContent(context.Background(), "1", &start, &size)
	checkNoError(t, err)
}

// TestPlexClientGetLibrarySectionContentError tests error handling
func TestPlexClientGetLibrarySectionContentError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	_, err := client.GetLibrarySectionContent(context.Background(), "999", nil, nil)
	checkError(t, err)
}

// ============================================================================
// Recently Added Tests - GET /library/sections/{id}/recentlyAdded
// ============================================================================

// TestPlexClientGetLibrarySectionRecentlyAdded tests recently added retrieval
func TestPlexClientGetLibrarySectionRecentlyAdded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkTrue(t, "path contains /library/sections/1/recentlyAdded",
			strings.HasPrefix(r.URL.Path, "/library/sections/1/recentlyAdded"))
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(recentlyAddedResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	content, err := client.GetLibrarySectionRecentlyAdded(context.Background(), "1", nil)

	checkNoError(t, err)
	checkSliceLen(t, "Metadata", len(content.MediaContainer.Metadata), 2)

	// Verify most recent item
	recent := content.MediaContainer.Metadata[0]
	checkStringEqual(t, "Recent.Title", recent.Title, "New Movie")
	checkInt64Equal(t, "Recent.AddedAt", recent.AddedAt, 1700050000)
}

const recentlyAddedResponse = `{
	"MediaContainer": {
		"size": 2,
		"librarySectionID": 1,
		"librarySectionTitle": "Movies",
		"Metadata": [
			{
				"ratingKey": "99999",
				"key": "/library/metadata/99999",
				"type": "movie",
				"title": "New Movie",
				"year": 2024,
				"addedAt": 1700050000,
				"duration": 7200000
			},
			{
				"ratingKey": "99998",
				"key": "/library/metadata/99998",
				"type": "movie",
				"title": "Another New Movie",
				"year": 2024,
				"addedAt": 1700040000,
				"duration": 6000000
			}
		]
	}
}`

// TestPlexClientGetLibrarySectionRecentlyAddedWithSize tests size parameter
func TestPlexClientGetLibrarySectionRecentlyAddedWithSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "size parameter", r.URL.Query().Get("X-Plex-Container-Size"), "20")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0, "Metadata": []}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	size := 20
	_, err := client.GetLibrarySectionRecentlyAdded(context.Background(), "1", &size)
	checkNoError(t, err)
}

// ============================================================================
// Activities Tests - GET /activities
// ============================================================================

// TestPlexClientGetActivities tests server activities retrieval
func TestPlexClientGetActivities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/activities")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(activitiesResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	activities, err := client.GetActivities(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Activity", len(activities.MediaContainer.Activity), 2)

	// Verify library scan activity
	scan := activities.MediaContainer.Activity[0]
	checkStringEqual(t, "Scan.UUID", scan.UUID, "activity-123")
	checkStringEqual(t, "Scan.Type", scan.Type, "library.refresh.items")
	checkStringEqual(t, "Scan.Title", scan.Title, "Scanning Movies")
	checkIntEqual(t, "Scan.Progress", scan.Progress, 45)
	checkTrue(t, "Scan.Cancellable", scan.Cancellable)
	checkTrue(t, "Scan.IsInProgress()", scan.IsInProgress())

	// Verify analysis activity
	analyze := activities.MediaContainer.Activity[1]
	checkStringEqual(t, "Analyze.Type", analyze.Type, "media.analyze")
	checkIntEqual(t, "Analyze.Progress", analyze.Progress, 100)
	checkTrue(t, "Analyze.IsComplete()", analyze.IsComplete())
}

const activitiesResponse = `{
	"MediaContainer": {
		"size": 2,
		"Activity": [
			{
				"uuid": "activity-123",
				"type": "library.refresh.items",
				"title": "Scanning Movies",
				"subtitle": "Processing new files...",
				"progress": 45,
				"userID": 1,
				"cancellable": true,
				"Context": {
					"librarySectionID": "1"
				}
			},
			{
				"uuid": "activity-456",
				"type": "media.analyze",
				"title": "Analyzing Media",
				"subtitle": "Complete",
				"progress": 100,
				"cancellable": false
			}
		]
	}
}`

// TestPlexClientGetActivitiesEmpty tests empty activities response
func TestPlexClientGetActivitiesEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	activities, err := client.GetActivities(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Activity", len(activities.MediaContainer.Activity), 0)
}

// TestPlexClientGetActivitiesError tests error handling
func TestPlexClientGetActivitiesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	_, err := client.GetActivities(context.Background())
	checkError(t, err)
}

// TestPlexClientGetActivitiesIndeterminate tests indeterminate progress activity
func TestPlexClientGetActivitiesIndeterminate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"MediaContainer": {
				"size": 1,
				"Activity": [
					{
						"uuid": "activity-indeterminate",
						"type": "library.update",
						"title": "Updating Library",
						"progress": -1,
						"cancellable": false
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	activities, err := client.GetActivities(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Activity", len(activities.MediaContainer.Activity), 1)

	activity := activities.MediaContainer.Activity[0]
	checkTrue(t, "Activity.IsIndeterminate()", activity.IsIndeterminate())
	checkTrue(t, "Activity should not be in progress", !activity.IsInProgress())
	checkTrue(t, "Activity should not be complete", !activity.IsComplete())
}

// ============================================================================
// Helper Functions
// ============================================================================

// checkIntEqual is defined in assert_helpers_test.go

func checkInt64Equal(t *testing.T, name string, got, want int64) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", name, got, want)
	}
}
