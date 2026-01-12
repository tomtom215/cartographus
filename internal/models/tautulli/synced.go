// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliSyncedItems represents the API response from get_synced_items endpoint
type TautulliSyncedItems struct {
	Response TautulliSyncedItemsResponse `json:"response"`
}

type TautulliSyncedItemsResponse struct {
	Result  string               `json:"result"`
	Message *string              `json:"message,omitempty"`
	Data    []TautulliSyncedItem `json:"data"`
}

type TautulliSyncedItem struct {
	ID                   int    `json:"id"`
	SyncID               string `json:"sync_id"`
	DeviceName           string `json:"device_name"`
	Platform             string `json:"platform"`
	UserID               int    `json:"user_id"`
	Username             string `json:"username"`
	FriendlyName         string `json:"friendly_name"`
	SyncTitle            string `json:"sync_title"`
	SyncMediaType        string `json:"sync_media_type"`
	RatingKey            string `json:"rating_key"`
	State                string `json:"state"`
	ItemsCount           int    `json:"items_count"`
	ItemsCompleteCount   int    `json:"items_complete_count"`
	ItemsDownloadedCount int    `json:"items_downloaded_count"`
	ItemsFailedCount     int    `json:"items_failed_count"`
	TotalSize            int64  `json:"total_size"`
	AudioBitrate         int    `json:"audio_bitrate"`
	VideoBitrate         int    `json:"video_bitrate"`
	VideoQuality         int    `json:"video_quality"`
	PhotoQuality         int    `json:"photo_quality"`
	ClientID             string `json:"client_id"`
	SyncVersion          int    `json:"sync_version"`
	RootTitle            string `json:"root_title"`
	MetadataType         string `json:"metadata_type"`
	ContentType          string `json:"content_type"`
}

// TautulliTerminateSession represents the API response from terminate_session endpoint
type TautulliTerminateSession struct {
	Response TautulliTerminateSessionResponse `json:"response"`
}

type TautulliTerminateSessionResponse struct {
	Result  string  `json:"result"`
	Message *string `json:"message,omitempty"`
}
