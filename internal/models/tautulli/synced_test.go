// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliSyncedItems_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": [
				{
					"id": 1,
					"sync_id": "sync_abc123",
					"device_name": "iPhone 14",
					"platform": "iOS",
					"user_id": 12345,
					"username": "john_doe",
					"friendly_name": "John Doe",
					"sync_title": "Breaking Bad - Complete Series",
					"sync_media_type": "show",
					"rating_key": "98765",
					"state": "complete",
					"items_count": 62,
					"items_complete_count": 62,
					"items_downloaded_count": 62,
					"items_failed_count": 0,
					"total_size": 52428800000,
					"audio_bitrate": 256,
					"video_bitrate": 4000,
					"video_quality": 100,
					"photo_quality": 100,
					"client_id": "client_xyz789",
					"sync_version": 2,
					"root_title": "Breaking Bad",
					"metadata_type": "show",
					"content_type": "video"
				}
			]
		}
	}`

	var synced TautulliSyncedItems
	err := json.Unmarshal([]byte(jsonData), &synced)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSyncedItems: %v", err)
	}

	if synced.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", synced.Response.Result)
	}

	if len(synced.Response.Data) != 1 {
		t.Fatalf("Expected 1 synced item, got %d", len(synced.Response.Data))
	}

	item := synced.Response.Data[0]
	if item.ID != 1 {
		t.Errorf("Expected ID 1, got %d", item.ID)
	}
	if item.SyncID != "sync_abc123" {
		t.Errorf("Expected SyncID 'sync_abc123', got '%s'", item.SyncID)
	}
	if item.DeviceName != "iPhone 14" {
		t.Errorf("Expected DeviceName 'iPhone 14', got '%s'", item.DeviceName)
	}
	if item.Platform != "iOS" {
		t.Errorf("Expected Platform 'iOS', got '%s'", item.Platform)
	}
	if item.UserID != 12345 {
		t.Errorf("Expected UserID 12345, got %d", item.UserID)
	}
	if item.Username != "john_doe" {
		t.Errorf("Expected Username 'john_doe', got '%s'", item.Username)
	}
	if item.FriendlyName != "John Doe" {
		t.Errorf("Expected FriendlyName 'John Doe', got '%s'", item.FriendlyName)
	}
	if item.SyncTitle != "Breaking Bad - Complete Series" {
		t.Errorf("Expected SyncTitle mismatch, got '%s'", item.SyncTitle)
	}
	if item.SyncMediaType != "show" {
		t.Errorf("Expected SyncMediaType 'show', got '%s'", item.SyncMediaType)
	}
	if item.RatingKey != "98765" {
		t.Errorf("Expected RatingKey '98765', got '%s'", item.RatingKey)
	}
	if item.State != "complete" {
		t.Errorf("Expected State 'complete', got '%s'", item.State)
	}
	if item.ItemsCount != 62 {
		t.Errorf("Expected ItemsCount 62, got %d", item.ItemsCount)
	}
	if item.ItemsCompleteCount != 62 {
		t.Errorf("Expected ItemsCompleteCount 62, got %d", item.ItemsCompleteCount)
	}
	if item.ItemsDownloadedCount != 62 {
		t.Errorf("Expected ItemsDownloadedCount 62, got %d", item.ItemsDownloadedCount)
	}
	if item.ItemsFailedCount != 0 {
		t.Errorf("Expected ItemsFailedCount 0, got %d", item.ItemsFailedCount)
	}
	if item.TotalSize != 52428800000 {
		t.Errorf("Expected TotalSize 52428800000, got %d", item.TotalSize)
	}
	if item.AudioBitrate != 256 {
		t.Errorf("Expected AudioBitrate 256, got %d", item.AudioBitrate)
	}
	if item.VideoBitrate != 4000 {
		t.Errorf("Expected VideoBitrate 4000, got %d", item.VideoBitrate)
	}
	if item.VideoQuality != 100 {
		t.Errorf("Expected VideoQuality 100, got %d", item.VideoQuality)
	}
	if item.PhotoQuality != 100 {
		t.Errorf("Expected PhotoQuality 100, got %d", item.PhotoQuality)
	}
	if item.ClientID != "client_xyz789" {
		t.Errorf("Expected ClientID 'client_xyz789', got '%s'", item.ClientID)
	}
	if item.SyncVersion != 2 {
		t.Errorf("Expected SyncVersion 2, got %d", item.SyncVersion)
	}
	if item.RootTitle != "Breaking Bad" {
		t.Errorf("Expected RootTitle 'Breaking Bad', got '%s'", item.RootTitle)
	}
	if item.MetadataType != "show" {
		t.Errorf("Expected MetadataType 'show', got '%s'", item.MetadataType)
	}
	if item.ContentType != "video" {
		t.Errorf("Expected ContentType 'video', got '%s'", item.ContentType)
	}
}

func TestTautulliSyncedItems_MultipleSyncs(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": [
				{
					"id": 1,
					"sync_id": "sync_001",
					"device_name": "iPad Pro",
					"platform": "iOS",
					"user_id": 100,
					"username": "user1",
					"friendly_name": "User One",
					"sync_title": "Movie Playlist",
					"sync_media_type": "movie",
					"rating_key": "111",
					"state": "pending",
					"items_count": 10,
					"items_complete_count": 5,
					"items_downloaded_count": 5,
					"items_failed_count": 0,
					"total_size": 5000000000,
					"audio_bitrate": 128,
					"video_bitrate": 2000,
					"video_quality": 75,
					"photo_quality": 75,
					"client_id": "client_001",
					"sync_version": 1,
					"root_title": "My Movies",
					"metadata_type": "movie",
					"content_type": "video"
				},
				{
					"id": 2,
					"sync_id": "sync_002",
					"device_name": "Android Phone",
					"platform": "Android",
					"user_id": 101,
					"username": "user2",
					"friendly_name": "User Two",
					"sync_title": "Music Collection",
					"sync_media_type": "artist",
					"rating_key": "222",
					"state": "syncing",
					"items_count": 500,
					"items_complete_count": 250,
					"items_downloaded_count": 200,
					"items_failed_count": 5,
					"total_size": 2000000000,
					"audio_bitrate": 320,
					"video_bitrate": 0,
					"video_quality": 0,
					"photo_quality": 0,
					"client_id": "client_002",
					"sync_version": 2,
					"root_title": "Various Artists",
					"metadata_type": "artist",
					"content_type": "audio"
				}
			]
		}
	}`

	var synced TautulliSyncedItems
	err := json.Unmarshal([]byte(jsonData), &synced)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSyncedItems: %v", err)
	}

	if len(synced.Response.Data) != 2 {
		t.Fatalf("Expected 2 synced items, got %d", len(synced.Response.Data))
	}

	// First item
	if synced.Response.Data[0].Platform != "iOS" {
		t.Errorf("Expected first Platform 'iOS', got '%s'", synced.Response.Data[0].Platform)
	}
	if synced.Response.Data[0].State != "pending" {
		t.Errorf("Expected first State 'pending', got '%s'", synced.Response.Data[0].State)
	}

	// Second item
	if synced.Response.Data[1].Platform != "Android" {
		t.Errorf("Expected second Platform 'Android', got '%s'", synced.Response.Data[1].Platform)
	}
	if synced.Response.Data[1].State != "syncing" {
		t.Errorf("Expected second State 'syncing', got '%s'", synced.Response.Data[1].State)
	}
	if synced.Response.Data[1].ItemsFailedCount != 5 {
		t.Errorf("Expected second ItemsFailedCount 5, got %d", synced.Response.Data[1].ItemsFailedCount)
	}
}

func TestTautulliSyncedItems_EmptyData(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": []
		}
	}`

	var synced TautulliSyncedItems
	err := json.Unmarshal([]byte(jsonData), &synced)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSyncedItems: %v", err)
	}

	if len(synced.Response.Data) != 0 {
		t.Errorf("Expected 0 synced items, got %d", len(synced.Response.Data))
	}
}

func TestTautulliSyncedItems_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Unable to fetch synced items",
			"data": []
		}
	}`

	var synced TautulliSyncedItems
	err := json.Unmarshal([]byte(jsonData), &synced)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSyncedItems: %v", err)
	}

	if synced.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", synced.Response.Result)
	}
	if synced.Response.Message == nil || *synced.Response.Message != "Unable to fetch synced items" {
		t.Error("Expected Message 'Unable to fetch synced items'")
	}
}

func TestTautulliSyncedItems_JSONRoundTrip(t *testing.T) {
	original := TautulliSyncedItems{
		Response: TautulliSyncedItemsResponse{
			Result: "success",
			Data: []TautulliSyncedItem{
				{
					ID:                   1,
					SyncID:               "test_sync",
					DeviceName:           "Test Device",
					Platform:             "macOS",
					UserID:               999,
					Username:             "testuser",
					FriendlyName:         "Test User",
					SyncTitle:            "Test Sync",
					SyncMediaType:        "movie",
					RatingKey:            "12345",
					State:                "complete",
					ItemsCount:           5,
					ItemsCompleteCount:   5,
					ItemsDownloadedCount: 5,
					ItemsFailedCount:     0,
					TotalSize:            1000000000,
					AudioBitrate:         192,
					VideoBitrate:         3000,
					VideoQuality:         90,
					PhotoQuality:         90,
					ClientID:             "test_client",
					SyncVersion:          1,
					RootTitle:            "Test Movie",
					MetadataType:         "movie",
					ContentType:          "video",
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliSyncedItems: %v", err)
	}

	var decoded TautulliSyncedItems
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSyncedItems: %v", err)
	}

	if len(decoded.Response.Data) != len(original.Response.Data) {
		t.Errorf("Data length mismatch")
	}
	if decoded.Response.Data[0].SyncID != original.Response.Data[0].SyncID {
		t.Errorf("SyncID mismatch")
	}
	if decoded.Response.Data[0].TotalSize != original.Response.Data[0].TotalSize {
		t.Errorf("TotalSize mismatch")
	}
}

func TestTautulliSyncedItem_SyncStates(t *testing.T) {
	states := []string{"pending", "syncing", "complete", "failed"}

	for _, state := range states {
		t.Run("State_"+state, func(t *testing.T) {
			original := TautulliSyncedItems{
				Response: TautulliSyncedItemsResponse{
					Result: "success",
					Data: []TautulliSyncedItem{
						{
							ID:       1,
							SyncID:   "sync_state_test",
							State:    state,
							Platform: "test",
						},
					},
				},
			}

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded TautulliSyncedItems
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Response.Data[0].State != state {
				t.Errorf("Expected State '%s', got '%s'", state, decoded.Response.Data[0].State)
			}
		})
	}
}

func TestTautulliTerminateSession_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"message": "Session terminated successfully"
		}
	}`

	var terminate TautulliTerminateSession
	err := json.Unmarshal([]byte(jsonData), &terminate)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliTerminateSession: %v", err)
	}

	if terminate.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", terminate.Response.Result)
	}
	if terminate.Response.Message == nil || *terminate.Response.Message != "Session terminated successfully" {
		t.Error("Expected Message 'Session terminated successfully'")
	}
}

func TestTautulliTerminateSession_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Session not found or already terminated"
		}
	}`

	var terminate TautulliTerminateSession
	err := json.Unmarshal([]byte(jsonData), &terminate)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliTerminateSession: %v", err)
	}

	if terminate.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", terminate.Response.Result)
	}
	if terminate.Response.Message == nil || *terminate.Response.Message != "Session not found or already terminated" {
		t.Error("Expected Message 'Session not found or already terminated'")
	}
}

func TestTautulliTerminateSession_NullMessage(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"message": null
		}
	}`

	var terminate TautulliTerminateSession
	err := json.Unmarshal([]byte(jsonData), &terminate)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliTerminateSession: %v", err)
	}

	if terminate.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", terminate.Response.Result)
	}
	if terminate.Response.Message != nil {
		t.Errorf("Expected Message nil, got '%s'", *terminate.Response.Message)
	}
}

func TestTautulliTerminateSession_JSONRoundTrip(t *testing.T) {
	msg := "Terminated"
	original := TautulliTerminateSession{
		Response: TautulliTerminateSessionResponse{
			Result:  "success",
			Message: &msg,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliTerminateSession: %v", err)
	}

	var decoded TautulliTerminateSession
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliTerminateSession: %v", err)
	}

	if decoded.Response.Result != original.Response.Result {
		t.Errorf("Result mismatch")
	}
	if decoded.Response.Message == nil || *decoded.Response.Message != *original.Response.Message {
		t.Errorf("Message mismatch")
	}
}

func TestTautulliSyncedItem_PhotoSync(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": [
				{
					"id": 1,
					"sync_id": "photo_sync",
					"device_name": "iPhone",
					"platform": "iOS",
					"user_id": 1,
					"username": "photographer",
					"friendly_name": "Photo Fan",
					"sync_title": "Vacation Photos",
					"sync_media_type": "photo",
					"rating_key": "55555",
					"state": "complete",
					"items_count": 1000,
					"items_complete_count": 1000,
					"items_downloaded_count": 1000,
					"items_failed_count": 0,
					"total_size": 10000000000,
					"audio_bitrate": 0,
					"video_bitrate": 0,
					"video_quality": 0,
					"photo_quality": 100,
					"client_id": "photo_client",
					"sync_version": 1,
					"root_title": "Photos",
					"metadata_type": "photo",
					"content_type": "image"
				}
			]
		}
	}`

	var synced TautulliSyncedItems
	err := json.Unmarshal([]byte(jsonData), &synced)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliSyncedItems: %v", err)
	}

	item := synced.Response.Data[0]
	if item.SyncMediaType != "photo" {
		t.Errorf("Expected SyncMediaType 'photo', got '%s'", item.SyncMediaType)
	}
	if item.ContentType != "image" {
		t.Errorf("Expected ContentType 'image', got '%s'", item.ContentType)
	}
	if item.AudioBitrate != 0 {
		t.Errorf("Expected AudioBitrate 0, got %d", item.AudioBitrate)
	}
	if item.VideoBitrate != 0 {
		t.Errorf("Expected VideoBitrate 0, got %d", item.VideoBitrate)
	}
	if item.PhotoQuality != 100 {
		t.Errorf("Expected PhotoQuality 100, got %d", item.PhotoQuality)
	}
}
