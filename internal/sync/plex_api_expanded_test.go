// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// Tests for New Plex API Client Methods (v1.13 - Standalone Operation)
// TDD: Tests written following Red-Green-Refactor pattern
// ============================================================================

// ============================================================================
// GetSessions Tests - GET /status/sessions
// ============================================================================

func TestPlexClientGetSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/status/sessions")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sessionsResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	sessions, err := client.GetSessions(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Metadata", len(sessions.MediaContainer.Metadata), 1)

	// Verify session data
	session := sessions.MediaContainer.Metadata[0]
	checkStringEqual(t, "Session.Title", session.Title, "Inception")
	checkStringEqual(t, "Session.Type", session.Type, "movie")
	checkStringEqual(t, "Session.SessionKey", session.SessionKey, "12345")
}

const sessionsResponse = `{
	"MediaContainer": {
		"size": 1,
		"Metadata": [
			{
				"sessionKey": "12345",
				"ratingKey": "1234",
				"key": "/library/metadata/1234",
				"type": "movie",
				"title": "Inception",
				"year": 2010,
				"duration": 8880000,
				"viewOffset": 1000000,
				"User": {
					"id": 1,
					"title": "TestUser",
					"thumb": "https://plex.tv/users/abc/avatar"
				},
				"Player": {
					"address": "192.168.1.100",
					"device": "Chrome",
					"machineIdentifier": "client-123",
					"model": "",
					"platform": "Chrome",
					"platformVersion": "120.0",
					"product": "Plex Web",
					"state": "playing",
					"title": "Chrome"
				},
				"Session": {
					"id": "session-abc",
					"bandwidth": 10000,
					"location": "lan"
				}
			}
		]
	}
}`

func TestPlexClientGetSessionsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	sessions, err := client.GetSessions(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Metadata", len(sessions.MediaContainer.Metadata), 0)
}

func TestPlexClientGetSessionsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	_, err := client.GetSessions(context.Background())
	checkError(t, err)
}

// ============================================================================
// GetIdentity Tests - GET /identity
// ============================================================================

func TestPlexClientGetIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/identity")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(identityResponseExpanded))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	identity, err := client.GetIdentity(context.Background())

	checkNoError(t, err)
	checkStringEqual(t, "Identity.MachineIdentifier", identity.MediaContainer.MachineIdentifier, "abc123def456")
	checkStringEqual(t, "Identity.Version", identity.MediaContainer.Version, "1.40.0.8395")
}

const identityResponseExpanded = `{
	"MediaContainer": {
		"size": 0,
		"claimed": true,
		"machineIdentifier": "abc123def456",
		"version": "1.40.0.8395"
	}
}`

func TestPlexClientGetIdentityError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	_, err := client.GetIdentity(context.Background())
	checkError(t, err)
}

// ============================================================================
// GetMetadata Tests - GET /library/metadata/{ratingKey}
// ============================================================================

func TestPlexClientGetMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/library/metadata/12345")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(metadataResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	metadata, err := client.GetMetadata(context.Background(), "12345")

	checkNoError(t, err)
	checkSliceLen(t, "Metadata", len(metadata.MediaContainer.Metadata), 1)

	item := metadata.MediaContainer.Metadata[0]
	checkStringEqual(t, "Item.Title", item.Title, "Inception")
	checkIntEqual(t, "Item.Year", item.Year, 2010)
	checkStringEqual(t, "Item.Type", item.Type, "movie")
}

const metadataResponse = `{
	"MediaContainer": {
		"size": 1,
		"librarySectionID": 1,
		"librarySectionTitle": "Movies",
		"Metadata": [
			{
				"ratingKey": "12345",
				"key": "/library/metadata/12345",
				"type": "movie",
				"title": "Inception",
				"year": 2010,
				"summary": "A thief who steals corporate secrets...",
				"rating": 8.8,
				"duration": 8880000,
				"thumb": "/library/metadata/12345/thumb",
				"Genre": [
					{"tag": "Action"},
					{"tag": "Sci-Fi"}
				],
				"Director": [
					{"tag": "Christopher Nolan"}
				],
				"Media": [
					{
						"id": 1,
						"duration": 8880000,
						"bitrate": 10000,
						"width": 1920,
						"height": 1080,
						"videoCodec": "h264",
						"audioCodec": "aac"
					}
				]
			}
		]
	}
}`

func TestPlexClientGetMetadataNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	_, err := client.GetMetadata(context.Background(), "99999")
	checkError(t, err)
}

// ============================================================================
// GetDevices Tests - GET /devices
// ============================================================================

func TestPlexClientGetDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/devices")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(devicesResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	devices, err := client.GetDevices(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Device", len(devices.MediaContainer.Device), 2)

	device := devices.MediaContainer.Device[0]
	checkStringEqual(t, "Device.Name", device.Name, "Roku Express")
	checkStringEqual(t, "Device.Platform", device.Platform, "Roku")
}

const devicesResponse = `{
	"MediaContainer": {
		"size": 2,
		"Device": [
			{
				"id": 1,
				"name": "Roku Express",
				"product": "Plex for Roku",
				"productVersion": "8.0",
				"platform": "Roku",
				"platformVersion": "12.0",
				"device": "Roku Express",
				"clientIdentifier": "device-123",
				"createdAt": 1700000000,
				"lastSeenAt": 1700100000,
				"presence": true
			},
			{
				"id": 2,
				"name": "iPhone",
				"product": "Plex for iOS",
				"productVersion": "9.0",
				"platform": "iOS",
				"platformVersion": "17.0",
				"device": "iPhone 15",
				"clientIdentifier": "device-456",
				"createdAt": 1700001000,
				"lastSeenAt": 1700090000,
				"presence": false
			}
		]
	}
}`

func TestPlexClientGetDevicesEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	devices, err := client.GetDevices(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Device", len(devices.MediaContainer.Device), 0)
}

// ============================================================================
// GetAccounts Tests - GET /accounts
// ============================================================================

func TestPlexClientGetAccounts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/accounts")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(accountsResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	accounts, err := client.GetAccounts(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Account", len(accounts.MediaContainer.Account), 2)

	account := accounts.MediaContainer.Account[0]
	checkStringEqual(t, "Account.Name", account.Name, "Admin")
	checkIntEqual(t, "Account.ID", account.ID, 1)
}

const accountsResponse = `{
	"MediaContainer": {
		"size": 2,
		"Account": [
			{
				"id": 1,
				"key": "/accounts/1",
				"name": "Admin",
				"defaultAudioLanguage": "en",
				"thumb": "https://plex.tv/users/abc/avatar"
			},
			{
				"id": 2,
				"key": "/accounts/2",
				"name": "FamilyMember",
				"defaultAudioLanguage": "en",
				"autoSelectAudio": true
			}
		]
	}
}`

// ============================================================================
// GetOnDeck Tests - GET /library/onDeck
// ============================================================================

func TestPlexClientGetOnDeck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/library/onDeck")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(onDeckResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	onDeck, err := client.GetOnDeck(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Metadata", len(onDeck.MediaContainer.Metadata), 2)

	item := onDeck.MediaContainer.Metadata[0]
	checkStringEqual(t, "Item.Title", item.Title, "The Beginning")
	checkStringEqual(t, "Item.Type", item.Type, "episode")
}

const onDeckResponse = `{
	"MediaContainer": {
		"size": 2,
		"allowSync": true,
		"mixedParents": true,
		"Metadata": [
			{
				"ratingKey": "54321",
				"key": "/library/metadata/54321",
				"type": "episode",
				"title": "The Beginning",
				"parentTitle": "Season 1",
				"grandparentTitle": "Breaking Bad",
				"index": 1,
				"parentIndex": 1,
				"viewOffset": 1200000,
				"duration": 3600000
			},
			{
				"ratingKey": "54322",
				"key": "/library/metadata/54322",
				"type": "movie",
				"title": "The Matrix",
				"viewOffset": 3600000,
				"duration": 8160000
			}
		]
	}
}`

// ============================================================================
// GetPlaylists Tests - GET /playlists
// ============================================================================

func TestPlexClientGetPlaylists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/playlists")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(playlistsResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	playlists, err := client.GetPlaylists(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "Metadata", len(playlists.MediaContainer.Metadata), 2)

	playlist := playlists.MediaContainer.Metadata[0]
	checkStringEqual(t, "Playlist.Title", playlist.Title, "Favorites")
	checkStringEqual(t, "Playlist.PlaylistType", playlist.PlaylistType, "video")
}

const playlistsResponse = `{
	"MediaContainer": {
		"size": 2,
		"Metadata": [
			{
				"ratingKey": "playlist-1",
				"key": "/playlists/playlist-1/items",
				"type": "playlist",
				"title": "Favorites",
				"playlistType": "video",
				"smart": false,
				"leafCount": 15,
				"duration": 50000000
			},
			{
				"ratingKey": "playlist-2",
				"key": "/playlists/playlist-2/items",
				"type": "playlist",
				"title": "Recent Watches",
				"playlistType": "video",
				"smart": true,
				"leafCount": 10,
				"duration": 30000000
			}
		]
	}
}`

// ============================================================================
// Search Tests - GET /library/sections/{key}/search
// ============================================================================

func TestPlexClientSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkTrue(t, "path contains /library/sections/1/search",
			r.URL.Path == "/library/sections/1/search")
		checkStringEqual(t, "query parameter", r.URL.Query().Get("query"), "inception")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(searchResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	results, err := client.Search(context.Background(), "1", "inception", nil)

	checkNoError(t, err)
	checkSliceLen(t, "Metadata", len(results.MediaContainer.Metadata), 1)

	item := results.MediaContainer.Metadata[0]
	checkStringEqual(t, "Item.Title", item.Title, "Inception")
}

const searchResponse = `{
	"MediaContainer": {
		"size": 1,
		"librarySectionID": 1,
		"librarySectionTitle": "Movies",
		"Metadata": [
			{
				"ratingKey": "12345",
				"key": "/library/metadata/12345",
				"type": "movie",
				"title": "Inception",
				"year": 2010
			}
		]
	}
}`

func TestPlexClientSearchWithType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "query parameter", r.URL.Query().Get("query"), "test")
		checkStringEqual(t, "type parameter", r.URL.Query().Get("type"), "1")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0, "Metadata": []}}`))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	mediaType := 1 // movie
	_, err := client.Search(context.Background(), "1", "test", &mediaType)
	checkNoError(t, err)
}

// ============================================================================
// GetTranscodeSessionsDetailed Tests - GET /transcode/sessions
// ============================================================================

func TestPlexClientGetTranscodeSessionsDetailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/transcode/sessions")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(transcodeSessionsDetailedResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	sessions, err := client.GetTranscodeSessionsDetailed(context.Background())

	checkNoError(t, err)
	checkSliceLen(t, "TranscodeSession", len(sessions.MediaContainer.TranscodeSession), 1)

	session := sessions.MediaContainer.TranscodeSession[0]
	checkStringEqual(t, "Session.Key", session.Key, "transcode-123")
	checkStringEqual(t, "Session.VideoDecision", session.VideoDecision, "transcode")
}

const transcodeSessionsDetailedResponse = `{
	"MediaContainer": {
		"size": 1,
		"TranscodeSession": [
			{
				"key": "transcode-123",
				"throttled": false,
				"complete": false,
				"progress": 45.5,
				"size": 500000000,
				"speed": 2.5,
				"error": false,
				"duration": 8880000,
				"remaining": 100,
				"context": "streaming",
				"sourceVideoCodec": "hevc",
				"sourceAudioCodec": "truehd",
				"videoDecision": "transcode",
				"audioDecision": "transcode",
				"protocol": "http",
				"container": "mkv",
				"videoCodec": "h264",
				"audioCodec": "aac",
				"audioChannels": 2,
				"transcodeHwRequested": true,
				"transcodeHwDecoding": "qsv",
				"transcodeHwEncoding": "qsv",
				"transcodeHwFullPipeline": true
			}
		]
	}
}`

// ============================================================================
// CancelTranscode Tests - DELETE /transcode/sessions/{sessionKey}
// ============================================================================

func TestPlexClientCancelTranscode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/transcode/sessions/transcode-123")
		checkStringEqual(t, "method", r.Method, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	err := client.CancelTranscode(context.Background(), "transcode-123")
	checkNoError(t, err)
}

func TestPlexClientCancelTranscodeOK(t *testing.T) {
	// Some Plex versions return 200 OK instead of 204 No Content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	err := client.CancelTranscode(context.Background(), "transcode-123")
	checkNoError(t, err)
}

func TestPlexClientCancelTranscodeNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	err := client.CancelTranscode(context.Background(), "invalid-key")
	checkError(t, err)
}

// ============================================================================
// GetServerCapabilities Tests - GET /
// ============================================================================

func TestPlexClientGetServerCapabilities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkStringEqual(t, "path", r.URL.Path, "/")
		verifyPlexHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(serverCapabilitiesResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	capabilities, err := client.GetServerCapabilities(context.Background())

	checkNoError(t, err)
	checkStringEqual(t, "FriendlyName", capabilities.MediaContainer.FriendlyName, "Plex Media Server")
	checkStringEqual(t, "MachineIdentifier", capabilities.MediaContainer.MachineIdentifier, "abc123def456")
	checkStringEqual(t, "Version", capabilities.MediaContainer.Version, "1.40.0.8395")
	checkStringEqual(t, "Platform", capabilities.MediaContainer.Platform, "Linux")

	// Verify feature flags
	checkTrue(t, "TranscoderVideo", capabilities.MediaContainer.TranscoderVideo)
	checkTrue(t, "TranscoderAudio", capabilities.MediaContainer.TranscoderAudio)
	checkTrue(t, "MyPlexSubscription", capabilities.MediaContainer.MyPlexSubscription)
	checkTrue(t, "AllowSync", capabilities.MediaContainer.AllowSync)
}

const serverCapabilitiesResponse = `{
	"MediaContainer": {
		"size": 25,
		"friendlyName": "Plex Media Server",
		"machineIdentifier": "abc123def456",
		"version": "1.40.0.8395",
		"platform": "Linux",
		"platformVersion": "Ubuntu 22.04",
		"claimed": true,
		"myPlex": true,
		"myPlexMappingState": "mapped",
		"myPlexSigninState": "ok",
		"myPlexSubscription": true,
		"myPlexUsername": "testuser@example.com",
		"allowCameraUpload": true,
		"allowChannelAccess": true,
		"allowMediaDeletion": true,
		"allowSharing": true,
		"allowSync": true,
		"allowTuners": true,
		"backgroundProcessing": true,
		"certificate": true,
		"companionProxy": true,
		"countryCode": "US",
		"eventStream": true,
		"hubSearch": true,
		"livetv": true,
		"mediaProviders": true,
		"multiuser": true,
		"photoAutoTag": true,
		"pluginHost": true,
		"pushNotifications": true,
		"streamingBrainABRVersion": 3,
		"streamingBrainVersion": 2,
		"sync": true,
		"transcoderActiveVideoSessions": 2,
		"transcoderAudio": true,
		"transcoderLyrics": true,
		"transcoderPhoto": true,
		"transcoderSubtitles": true,
		"transcoderVideo": true,
		"transcoderVideoBitrates": "64,96,208,320,720,1500,2000,3000,4000,8000,10000,12000,20000",
		"transcoderVideoQualities": "0,1,2,3,4,5,6,7,8,9,10,11,12",
		"transcoderVideoResolutions": "128,128,160,240,320,480,720,720,1080,1080,1080,1080,1080",
		"updatedAt": 1700000000,
		"updater": true,
		"voiceSearch": true,
		"Directory": [
			{
				"count": 5,
				"key": "activities",
				"title": "activities"
			},
			{
				"count": 1,
				"key": "channels",
				"title": "channels"
			},
			{
				"count": 3,
				"key": "library",
				"title": "library"
			}
		]
	}
}`

func TestPlexClientGetServerCapabilitiesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	_, err := client.GetServerCapabilities(context.Background())
	checkError(t, err)
}

func TestPlexClientGetServerCapabilitiesHelperMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(serverCapabilitiesResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	capabilities, err := client.GetServerCapabilities(context.Background())
	checkNoError(t, err)

	// Test helper methods
	checkTrue(t, "HasPlexPass", capabilities.MediaContainer.HasPlexPass())
	checkTrue(t, "SupportsHardwareTranscoding", capabilities.MediaContainer.SupportsHardwareTranscoding())
	checkTrue(t, "SupportsLiveTV", capabilities.MediaContainer.SupportsLiveTV())
	checkTrue(t, "SupportsSync", capabilities.MediaContainer.SupportsSync())
	checkTrue(t, "IsClaimedAndConnected", capabilities.MediaContainer.IsClaimedAndConnected())
	checkIntEqual(t, "GetActiveTranscodeSessions", capabilities.MediaContainer.GetActiveTranscodeSessions(), 2)
}

func TestPlexClientGetServerCapabilitiesDirectories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(serverCapabilitiesResponse))
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	capabilities, err := client.GetServerCapabilities(context.Background())
	checkNoError(t, err)

	// Verify directories are parsed
	checkSliceLen(t, "Directory", len(capabilities.MediaContainer.Directory), 3)

	dir := capabilities.MediaContainer.Directory[0]
	checkStringEqual(t, "Directory.Key", dir.Key, "activities")
	checkStringEqual(t, "Directory.Title", dir.Title, "activities")
	checkIntEqual(t, "Directory.Count", dir.Count, 5)
}
