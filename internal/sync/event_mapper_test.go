// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// TestBuildCoreEvent tests the buildCoreEvent method
func TestBuildCoreEvent(t *testing.T) {
	m := &Manager{}
	now := time.Now().Unix()

	tests := []struct {
		name   string
		record *tautulli.TautulliHistoryRecord
		verify func(t *testing.T, event *models.PlaybackEvent)
	}{
		{
			name: "basic fields are mapped correctly",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:      stringPtr("abc123"),
				Started:         now,
				UserID:          intPtr(1),
				User:            "testuser",
				IPAddress:       "192.168.1.100",
				MediaType:       "episode",
				Title:           "Test Episode",
				Platform:        "Chrome",
				Player:          "Plex Web",
				Location:        "lan",
				PercentComplete: intPtr(75),
				PausedCounter:   intPtr(30),
			},
			verify: verifyBasicFields,
		},
		{
			name: "stopped time is set when > 0",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey: stringPtr("xyz789"),
				Started:    now,
				Stopped:    now + 3600, // 1 hour later
			},
			verify: verifyStoppedTime(now + 3600),
		},
		{
			name: "parent title is set when non-empty",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:  stringPtr("parent123"),
				Started:     now,
				ParentTitle: stringPtr("Season 1"),
			},
			verify: verifyParentTitle("Season 1"),
		},
		{
			name: "grandparent title is set when non-empty",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:       stringPtr("grandparent123"),
				Started:          now,
				GrandparentTitle: stringPtr("Breaking Bad"),
			},
			verify: verifyGrandparentTitle("Breaking Bad"),
		},
		{
			name: "empty optional fields remain nil",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:       stringPtr("minimal"),
				Started:          now,
				ParentTitle:      nil,
				GrandparentTitle: nil,
				Stopped:          0,
			},
			verify: verifyEmptyOptionalFields,
		},
		{
			name: "UUID is generated",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey: stringPtr("uuid-test"),
				Started:    now,
			},
			verify: verifyUUIDGenerated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := m.buildCoreEvent(tt.record)
			if event == nil {
				t.Fatal("buildCoreEvent returned nil")
			}
			tt.verify(t, event)
		})
	}
}

// Verification functions for TestBuildCoreEvent - extracted to reduce cyclomatic complexity

func verifyBasicFields(t *testing.T, event *models.PlaybackEvent) {
	t.Helper()
	checkStringEqual(t, "SessionKey", event.SessionKey, "abc123")
	checkIntEqual(t, "UserID", event.UserID, 1)
	checkStringEqual(t, "Username", event.Username, "testuser")
	checkStringEqual(t, "IPAddress", event.IPAddress, "192.168.1.100")
	checkStringEqual(t, "MediaType", event.MediaType, "episode")
	checkStringEqual(t, "Title", event.Title, "Test Episode")
	checkStringEqual(t, "Platform", event.Platform, "Chrome")
	checkStringEqual(t, "Player", event.Player, "Plex Web")
	checkStringEqual(t, "LocationType", event.LocationType, "lan")
	checkIntEqual(t, "PercentComplete", event.PercentComplete, 75)
	checkIntEqual(t, "PausedCounter", event.PausedCounter, 30)
	checkNil(t, "StoppedAt", event.StoppedAt == nil)
}

func verifyStoppedTime(expectedStopped int64) func(t *testing.T, event *models.PlaybackEvent) {
	return func(t *testing.T, event *models.PlaybackEvent) {
		t.Helper()
		if event.StoppedAt == nil {
			t.Error("StoppedAt should not be nil when Stopped > 0")
			return
		}
		expected := time.Unix(expectedStopped, 0)
		if !event.StoppedAt.Equal(expected) {
			t.Errorf("StoppedAt: expected %v, got %v", expected, *event.StoppedAt)
		}
	}
}

func verifyParentTitle(expected string) func(t *testing.T, event *models.PlaybackEvent) {
	return func(t *testing.T, event *models.PlaybackEvent) {
		t.Helper()
		checkStringPtrEqual(t, "ParentTitle", event.ParentTitle, expected)
	}
}

func verifyGrandparentTitle(expected string) func(t *testing.T, event *models.PlaybackEvent) {
	return func(t *testing.T, event *models.PlaybackEvent) {
		t.Helper()
		checkStringPtrEqual(t, "GrandparentTitle", event.GrandparentTitle, expected)
	}
}

func verifyEmptyOptionalFields(t *testing.T, event *models.PlaybackEvent) {
	t.Helper()
	checkStringPtrNil(t, "ParentTitle", event.ParentTitle)
	checkStringPtrNil(t, "GrandparentTitle", event.GrandparentTitle)
	checkNil(t, "StoppedAt", event.StoppedAt == nil)
}

func verifyUUIDGenerated(t *testing.T, event *models.PlaybackEvent) {
	t.Helper()
	var zeroUUID [16]byte
	if event.ID == zeroUUID {
		t.Error("ID should be a valid UUID, got zero")
	}
}

// TestEnrichEventWithMetadata tests metadata enrichment
func TestEnrichEventWithMetadata(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:            stringPtr("meta123"),
		Started:               time.Now().Unix(),
		SectionID:             intPtr(5),
		LibraryName:           "Movies",
		ContentRating:         "PG-13",
		Duration:              intPtr(7200), // 2 hours in seconds
		Year:                  intPtr(2023),
		Guid:                  "plex://movie/12345",
		OriginalTitle:         stringPtr("Original Title"),
		FullTitle:             "Full Title - Extended",
		OriginallyAvailableAt: stringPtr("2023-05-15"),
		WatchedStatus:         float64Ptr(1.0),
		Thumb:                 "/library/metadata/12345/thumb",
		Directors:             "Christopher Nolan",
		Writers:               "Jonathan Nolan",
		Actors:                "Leonardo DiCaprio",
		Genres:                "Sci-Fi, Action",
		RatingKey:             intPtr(12345),
		ParentRatingKey:       intPtr(123),
		GrandparentRatingKey:  intPtr(456),
		MediaIndex:            intPtr(5),
		ParentMediaIndex:      intPtr(2),
	}

	event := m.buildCoreEvent(record)
	m.enrichEventWithMetadata(event, record)

	// Verify library fields
	checkIntPtrEqual(t, "SectionID", event.SectionID, 5)
	checkStringPtrEqual(t, "LibraryName", event.LibraryName, "Movies")
	checkStringPtrEqual(t, "ContentRating", event.ContentRating, "PG-13")
	checkIntPtrEqual(t, "PlayDuration", event.PlayDuration, 120) // 7200/60 = 120 minutes
	checkIntPtrEqual(t, "Year", event.Year, 2023)

	// Verify metadata enrichment fields
	checkStringPtrEqual(t, "Guid", event.Guid, "plex://movie/12345")
	checkStringPtrEqual(t, "OriginalTitle", event.OriginalTitle, "Original Title")
	checkStringPtrEqual(t, "FullTitle", event.FullTitle, "Full Title - Extended")
	checkStringPtrEqual(t, "OriginallyAvailableAt", event.OriginallyAvailableAt, "2023-05-15")
	checkIntPtrEqual(t, "WatchedStatus", event.WatchedStatus, 1)
	checkStringPtrEqual(t, "Thumb", event.Thumb, "/library/metadata/12345/thumb")
	checkStringPtrEqual(t, "Directors", event.Directors, "Christopher Nolan")
	checkStringPtrEqual(t, "Writers", event.Writers, "Jonathan Nolan")
	checkStringPtrEqual(t, "Actors", event.Actors, "Leonardo DiCaprio")
	checkStringPtrEqual(t, "Genres", event.Genres, "Sci-Fi, Action")
}

// TestEnrichEventWithQualityData tests quality data enrichment
func TestEnrichEventWithQualityData(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:            stringPtr("quality123"),
		Started:               time.Now().Unix(),
		TranscodeDecision:     "transcode",
		VideoResolution:       "1080",
		VideoCodec:            "h264",
		AudioCodec:            "aac",
		StreamVideoResolution: "720",
		StreamAudioCodec:      "aac",
		StreamAudioChannels:   "2",
		StreamVideoDecision:   "transcode",
		StreamAudioDecision:   "copy",
		StreamContainer:       "mp4",
		StreamBitrate:         intPtr(5000),
	}

	event := m.buildCoreEvent(record)
	m.enrichEventWithQualityData(event, record)

	// Verify streaming quality fields
	checkStringPtrEqual(t, "TranscodeDecision", event.TranscodeDecision, "transcode")
	checkStringPtrEqual(t, "VideoResolution", event.VideoResolution, "1080")
	checkStringPtrEqual(t, "VideoCodec", event.VideoCodec, "h264")
	checkStringPtrEqual(t, "AudioCodec", event.AudioCodec, "aac")

	// Verify stream details fields
	checkStringPtrEqual(t, "StreamVideoResolution", event.StreamVideoResolution, "720")
	checkStringPtrEqual(t, "StreamAudioCodec", event.StreamAudioCodec, "aac")
	checkStringPtrEqual(t, "StreamAudioChannels", event.StreamAudioChannels, "2")
	checkStringPtrEqual(t, "StreamVideoDecision", event.StreamVideoDecision, "transcode")
	checkStringPtrEqual(t, "StreamAudioDecision", event.StreamAudioDecision, "copy")
	checkStringPtrEqual(t, "StreamContainer", event.StreamContainer, "mp4")
	checkIntPtrEqual(t, "StreamBitrate", event.StreamBitrate, 5000)
}

// TestMapAudioFields tests audio field mapping
func TestMapAudioFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:         stringPtr("audio123"),
		Started:            time.Now().Unix(),
		AudioChannels:      "6",
		AudioChannelLayout: "5.1",
		AudioBitrate:       intPtr(640),
		AudioSampleRate:    intPtr(48000),
		AudioLanguage:      "English",
	}

	event := m.buildCoreEvent(record)
	m.mapAudioFields(record, event)

	checkStringPtrEqual(t, "AudioChannels", event.AudioChannels, "6")
	checkStringPtrEqual(t, "AudioChannelLayout", event.AudioChannelLayout, "5.1")
	checkIntPtrEqual(t, "AudioBitrate", event.AudioBitrate, 640)
	checkIntPtrEqual(t, "AudioSampleRate", event.AudioSampleRate, 48000)
	checkStringPtrEqual(t, "AudioLanguage", event.AudioLanguage, "English")
}

// TestMapVideoFields tests video field mapping
func TestMapVideoFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:        stringPtr("video123"),
		Started:           time.Now().Unix(),
		VideoDynamicRange: "HDR",
		VideoFrameRate:    "24p",
		VideoBitrate:      intPtr(15000),
		VideoBitDepth:     intPtr(10),
		VideoWidth:        intPtr(3840),
		VideoHeight:       intPtr(2160),
	}

	event := m.buildCoreEvent(record)
	m.mapVideoFields(record, event)

	checkStringPtrEqual(t, "VideoDynamicRange", event.VideoDynamicRange, "HDR")
	checkStringPtrEqual(t, "VideoFrameRate", event.VideoFrameRate, "24p")
	checkIntPtrEqual(t, "VideoBitrate", event.VideoBitrate, 15000)
	checkIntPtrEqual(t, "VideoBitDepth", event.VideoBitDepth, 10)
	checkIntPtrEqual(t, "VideoWidth", event.VideoWidth, 3840)
	checkIntPtrEqual(t, "VideoHeight", event.VideoHeight, 2160)
}

// TestMapContainerSubtitleFields tests container and subtitle field mapping
func TestMapContainerSubtitleFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:       stringPtr("subtitle123"),
		Started:          time.Now().Unix(),
		Container:        "mkv",
		SubtitleCodec:    "srt",
		SubtitleLanguage: "English",
		Subtitles:        intPtr(1),
	}

	event := m.buildCoreEvent(record)
	m.mapContainerSubtitleFields(record, event)

	checkStringPtrEqual(t, "Container", event.Container, "mkv")
	checkStringPtrEqual(t, "SubtitleCodec", event.SubtitleCodec, "srt")
	checkStringPtrEqual(t, "SubtitleLanguage", event.SubtitleLanguage, "English")
	checkIntPtrEqual(t, "Subtitles", event.Subtitles, 1)
}

// TestMapConnectionFields tests connection field mapping
func TestMapConnectionFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("connection123"),
		Started:    time.Now().Unix(),
		Secure:     intPtr(1),
		Relayed:    intPtr(0),
		Local:      intPtr(1),
	}

	event := m.buildCoreEvent(record)
	m.mapConnectionFields(record, event)

	checkIntPtrEqual(t, "Secure", event.Secure, 1)
	checkIntPtrEqual(t, "Relayed", event.Relayed, 0)
	checkIntPtrEqual(t, "Local", event.Local, 1)
}

// TestMapFileMetadataFields tests file metadata field mapping
func TestMapFileMetadataFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:    stringPtr("file123"),
		Started:       time.Now().Unix(),
		FileSize:      int64Ptr(5000000000), // 5GB
		Bitrate:       intPtr(25000),
		StreamBitrate: intPtr(8000),
	}

	event := m.buildCoreEvent(record)
	m.mapFileMetadataFields(record, event)

	checkInt64PtrEqual(t, "FileSize", event.FileSize, 5000000000)
	checkIntPtrEqual(t, "Bitrate", event.Bitrate, 25000)
	checkIntPtrEqual(t, "SourceBitrate", event.SourceBitrate, 25000)
	checkIntPtrEqual(t, "TranscodeBitrate", event.TranscodeBitrate, 8000)
}

// TestMapEnrichmentFields tests enrichment field mapping
func TestMapEnrichmentFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:            stringPtr("enrich123"),
		Started:               time.Now().Unix(),
		RatingKey:             intPtr(12345),
		ParentRatingKey:       intPtr(123),
		GrandparentRatingKey:  intPtr(456),
		MediaIndex:            intPtr(5),
		ParentMediaIndex:      intPtr(2),
		Guid:                  "plex://movie/12345",
		OriginalTitle:         stringPtr("Original Title"),
		FullTitle:             "Full Title",
		OriginallyAvailableAt: stringPtr("2023-05-15"),
		WatchedStatus:         float64Ptr(1.0),
		Thumb:                 "/thumb",
		Directors:             "Director",
		Writers:               "Writer",
		Actors:                "Actor",
		Genres:                "Genre",
	}

	event := m.buildCoreEvent(record)
	m.mapEnrichmentFields(record, event)

	checkStringPtrEqual(t, "RatingKey", event.RatingKey, "12345")
	checkStringPtrEqual(t, "ParentRatingKey", event.ParentRatingKey, "123")
	checkStringPtrEqual(t, "GrandparentRatingKey", event.GrandparentRatingKey, "456")
	checkIntPtrEqual(t, "MediaIndex", event.MediaIndex, 5)
	checkIntPtrEqual(t, "ParentMediaIndex", event.ParentMediaIndex, 2)
}

// TestConvertPlexMetadataFields tests Plex metadata conversion
func TestConvertPlexMetadataFields(t *testing.T) {
	tests := []struct {
		name   string
		record *PlexMetadata
		verify func(t *testing.T, fields plexMetadataFields)
	}{
		{
			name: "all fields populated",
			record: &PlexMetadata{
				RatingKey:             "12345",
				ParentRatingKey:       "parent123",
				GrandparentRatingKey:  "gparent123",
				Index:                 5,
				ParentIndex:           2,
				Guid:                  "plex://movie/12345",
				OriginalTitle:         "Original",
				OriginallyAvailableAt: "2023-05-15",
				Thumb:                 "/thumb",
				Year:                  2023,
				Duration:              7200000, // 2 hours in milliseconds
				ParentTitle:           "Season 1",
				GrandparentTitle:      "Breaking Bad",
			},
			verify: verifyAllFieldsPopulated,
		},
		{
			name: "empty fields return nil",
			record: &PlexMetadata{
				RatingKey: "",
				Index:     0,
				Year:      0,
				Duration:  0,
			},
			verify: verifyEmptyFieldsNil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := convertPlexMetadataFields(tt.record)
			tt.verify(t, fields)
		})
	}
}

func verifyAllFieldsPopulated(t *testing.T, fields plexMetadataFields) {
	t.Helper()
	checkStringPtrEqual(t, "ratingKey", fields.ratingKey, "12345")
	checkStringPtrEqual(t, "parentRatingKey", fields.parentRatingKey, "parent123")
	checkStringPtrEqual(t, "grandparentRatingKey", fields.grandparentRatingKey, "gparent123")
	checkIntPtrEqual(t, "mediaIndex", fields.mediaIndex, 5)
	checkIntPtrEqual(t, "parentMediaIndex", fields.parentMediaIndex, 2)
	checkStringPtrEqual(t, "guid", fields.guid, "plex://movie/12345")
	checkStringPtrEqual(t, "originalTitle", fields.originalTitle, "Original")
	checkStringPtrEqual(t, "originallyAvailableAt", fields.originallyAvailableAt, "2023-05-15")
	checkStringPtrEqual(t, "thumb", fields.thumb, "/thumb")
	checkIntPtrEqual(t, "year", fields.year, 2023)
	checkIntPtrEqual(t, "playDuration", fields.playDuration, 7200) // milliseconds to seconds
	checkStringPtrEqual(t, "parentTitle", fields.parentTitle, "Season 1")
	checkStringPtrEqual(t, "grandparentTitle", fields.grandparentTitle, "Breaking Bad")
}

func verifyEmptyFieldsNil(t *testing.T, fields plexMetadataFields) {
	t.Helper()
	checkStringPtrNil(t, "ratingKey", fields.ratingKey)
	checkIntPtrNil(t, "mediaIndex", fields.mediaIndex)
	checkIntPtrNil(t, "year", fields.year)
	checkIntPtrNil(t, "playDuration", fields.playDuration)
}

// TestConvertPlexToPlaybackEvent tests Plex to PlaybackEvent conversion
func TestConvertPlexToPlaybackEvent(t *testing.T) {
	cfg := &config.Config{}
	m := &Manager{cfg: cfg}

	now := time.Now().Unix()
	record := &PlexMetadata{
		RatingKey:            "12345",
		ParentRatingKey:      "parent123",
		GrandparentRatingKey: "gparent123",
		Type:                 "episode",
		Title:                "Pilot",
		GrandparentTitle:     "Breaking Bad",
		ParentTitle:          "Season 1",
		ViewedAt:             now,
		Duration:             3600000, // 1 hour in milliseconds
		ViewOffset:           1800000, // 30 minutes in milliseconds
		AccountID:            42,
		User:                 &PlexUser{ID: 42, Title: "testuser"},
		Index:                1,
		ParentIndex:          1,
		Year:                 2008,
		Guid:                 "plex://show/12345/season/1/episode/1",
		Thumb:                "/library/metadata/12345/thumb",
	}

	event := m.convertPlexToPlaybackEvent(record)

	// Verify source
	checkStringEqual(t, "Source", event.Source, "plex")

	// Verify basic fields
	checkStringEqual(t, "Title", event.Title, "Pilot")
	checkStringEqual(t, "MediaType", event.MediaType, "episode")
	checkStringEqual(t, "Username", event.Username, "testuser")
	checkIntEqual(t, "UserID", event.UserID, 42)

	// Verify percent complete
	checkIntEqual(t, "PercentComplete", event.PercentComplete, 50)

	// Verify rating keys
	checkStringPtrEqual(t, "RatingKey", event.RatingKey, "12345")
	checkStringPtrEqual(t, "ParentRatingKey", event.ParentRatingKey, "parent123")
	checkStringPtrEqual(t, "GrandparentRatingKey", event.GrandparentRatingKey, "gparent123")

	// Verify episode numbering
	checkIntPtrEqual(t, "MediaIndex", event.MediaIndex, 1)
	checkIntPtrEqual(t, "ParentMediaIndex", event.ParentMediaIndex, 1)

	// Verify MISSING fields from Plex (should be empty/nil)
	checkStringEmpty(t, "IPAddress", event.IPAddress)
	checkStringEmpty(t, "Platform", event.Platform)
	checkStringEmpty(t, "Player", event.Player)
	checkStringEmpty(t, "LocationType", event.LocationType)
}

// TestConvertPlexToPlaybackEventNilUser tests conversion with nil user
func TestConvertPlexToPlaybackEventNilUser(t *testing.T) {
	cfg := &config.Config{}
	m := &Manager{cfg: cfg}

	record := &PlexMetadata{
		RatingKey: "12345",
		Type:      "movie",
		Title:     "Test Movie",
		ViewedAt:  time.Now().Unix(),
		User:      nil, // No user
	}

	event := m.convertPlexToPlaybackEvent(record)

	checkStringEmpty(t, "Username", event.Username)
}

// TestConvertPlexToPlaybackEventZeroDuration tests conversion with zero duration
func TestConvertPlexToPlaybackEventZeroDuration(t *testing.T) {
	cfg := &config.Config{}
	m := &Manager{cfg: cfg}

	record := &PlexMetadata{
		RatingKey:  "12345",
		Type:       "movie",
		Title:      "Test Movie",
		ViewedAt:   time.Now().Unix(),
		Duration:   0,
		ViewOffset: 1000,
	}

	event := m.convertPlexToPlaybackEvent(record)

	// Percent complete should be 0 when duration is 0
	checkIntEqual(t, "PercentComplete", event.PercentComplete, 0)

	// StoppedAt should be nil when duration is 0
	checkNil(t, "StoppedAt", event.StoppedAt == nil)
}

// ========================================
// Extended Field Mapping Tests (v1.43 - API Coverage Audit)
// ========================================

// TestMapUserFields tests user information field mapping
func TestMapUserFields(t *testing.T) {
	m := &Manager{}

	tests := []struct {
		name   string
		record *tautulli.TautulliHistoryRecord
		verify func(t *testing.T, event *models.PlaybackEvent)
	}{
		{
			name: "all user fields populated",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:      stringPtr("user123"),
				Started:         time.Now().Unix(),
				FriendlyName:    "John Doe",
				UserThumb:       "https://plex.tv/users/abc/avatar",
				Email:           "john@example.com",
				IPAddressPublic: "203.0.113.50",
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkStringPtrEqual(t, "FriendlyName", event.FriendlyName, "John Doe")
				checkStringPtrEqual(t, "UserThumb", event.UserThumb, "https://plex.tv/users/abc/avatar")
				checkStringPtrEqual(t, "Email", event.Email, "john@example.com")
				checkStringPtrEqual(t, "IPAddressPublic", event.IPAddressPublic, "203.0.113.50")
			},
		},
		{
			name: "empty user fields remain nil",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:      stringPtr("user-empty"),
				Started:         time.Now().Unix(),
				FriendlyName:    "",
				UserThumb:       "",
				Email:           "",
				IPAddressPublic: "",
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkStringPtrNil(t, "FriendlyName", event.FriendlyName)
				checkStringPtrNil(t, "UserThumb", event.UserThumb)
				checkStringPtrNil(t, "Email", event.Email)
				checkStringPtrNil(t, "IPAddressPublic", event.IPAddressPublic)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := m.buildCoreEvent(tt.record)
			m.mapUserFields(tt.record, event)
			tt.verify(t, event)
		})
	}
}

// TestMapClientDeviceFields tests client/device identification field mapping
func TestMapClientDeviceFields(t *testing.T) {
	m := &Manager{}

	tests := []struct {
		name   string
		record *tautulli.TautulliHistoryRecord
		verify func(t *testing.T, event *models.PlaybackEvent)
	}{
		{
			name: "all client device fields populated",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:       stringPtr("client123"),
				Started:          time.Now().Unix(),
				PlatformName:     "Windows",
				PlatformVersion:  "10.0.19041",
				Product:          "Plex for Windows",
				ProductVersion:   "1.40.0.7998",
				Device:           "Gaming PC",
				MachineID:        "abc123def456",
				QualityProfile:   "Original",
				OptimizedVersion: intPtr(0),
				SyncedVersion:    intPtr(0),
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkStringPtrEqual(t, "PlatformName", event.PlatformName, "Windows")
				checkStringPtrEqual(t, "PlatformVersion", event.PlatformVersion, "10.0.19041")
				checkStringPtrEqual(t, "Product", event.Product, "Plex for Windows")
				checkStringPtrEqual(t, "ProductVersion", event.ProductVersion, "1.40.0.7998")
				checkStringPtrEqual(t, "Device", event.Device, "Gaming PC")
				checkStringPtrEqual(t, "MachineID", event.MachineID, "abc123def456")
				checkStringPtrEqual(t, "QualityProfile", event.QualityProfile, "Original")
				checkIntPtrEqual(t, "OptimizedVersion", event.OptimizedVersion, 0)
				checkIntPtrEqual(t, "SyncedVersion", event.SyncedVersion, 0)
			},
		},
		{
			name: "optimized and synced versions set",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:       stringPtr("client-optimized"),
				Started:          time.Now().Unix(),
				OptimizedVersion: intPtr(1),
				SyncedVersion:    intPtr(1),
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkIntPtrEqual(t, "OptimizedVersion", event.OptimizedVersion, 1)
				checkIntPtrEqual(t, "SyncedVersion", event.SyncedVersion, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := m.buildCoreEvent(tt.record)
			m.mapClientDeviceFields(tt.record, event)
			tt.verify(t, event)
		})
	}
}

// TestMapTranscodeDecisionFields tests transcode decision field mapping
func TestMapTranscodeDecisionFields(t *testing.T) {
	m := &Manager{}

	tests := []struct {
		name   string
		record *tautulli.TautulliHistoryRecord
		verify func(t *testing.T, event *models.PlaybackEvent)
	}{
		{
			name: "all transcode decisions",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:       stringPtr("transcode123"),
				Started:          time.Now().Unix(),
				VideoDecision:    "transcode",
				AudioDecision:    "copy",
				SubtitleDecision: "burn",
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkStringPtrEqual(t, "VideoDecision", event.VideoDecision, "transcode")
				checkStringPtrEqual(t, "AudioDecision", event.AudioDecision, "copy")
				checkStringPtrEqual(t, "SubtitleDecision", event.SubtitleDecision, "burn")
			},
		},
		{
			name: "direct play decisions",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:       stringPtr("direct123"),
				Started:          time.Now().Unix(),
				VideoDecision:    "copy",
				AudioDecision:    "copy",
				SubtitleDecision: "",
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkStringPtrEqual(t, "VideoDecision", event.VideoDecision, "copy")
				checkStringPtrEqual(t, "AudioDecision", event.AudioDecision, "copy")
				checkStringPtrNil(t, "SubtitleDecision", event.SubtitleDecision)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := m.buildCoreEvent(tt.record)
			m.mapTranscodeDecisionFields(tt.record, event)
			tt.verify(t, event)
		})
	}
}

// TestMapHardwareTranscodeFields tests hardware transcode field mapping
// CRITICAL: Tests GPU utilization monitoring (NVIDIA NVENC, Intel Quick Sync, AMD VCE)
func TestMapHardwareTranscodeFields(t *testing.T) {
	m := &Manager{}

	tests := []struct {
		name   string
		record *tautulli.TautulliHistoryRecord
		verify func(t *testing.T, event *models.PlaybackEvent)
	}{
		{
			name: "full hardware transcode pipeline (NVIDIA NVENC)",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:              stringPtr("hw123"),
				Started:                 time.Now().Unix(),
				TranscodeKey:            "abc123",
				TranscodeThrottled:      intPtr(0),
				TranscodeProgress:       intPtr(75),
				TranscodeSpeed:          "2.5x",
				TranscodeHWRequested:    intPtr(1),
				TranscodeHWDecoding:     intPtr(1),
				TranscodeHWEncoding:     intPtr(1),
				TranscodeHWFullPipeline: intPtr(1),
				TranscodeHWDecode:       "nvdec",
				TranscodeHWDecodeTitle:  "NVIDIA NVDEC",
				TranscodeHWEncode:       "nvenc",
				TranscodeHWEncodeTitle:  "NVIDIA NVENC",
				TranscodeContainer:      "mkv",
				TranscodeVideoCodec:     "hevc",
				TranscodeAudioCodec:     "aac",
				TranscodeAudioChannels:  intPtr(6),
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkStringPtrEqual(t, "TranscodeKey", event.TranscodeKey, "abc123")
				checkIntPtrEqual(t, "TranscodeThrottled", event.TranscodeThrottled, 0)
				checkIntPtrEqual(t, "TranscodeProgress", event.TranscodeProgress, 75)
				checkStringPtrEqual(t, "TranscodeSpeed", event.TranscodeSpeed, "2.5x")
				checkIntPtrEqual(t, "TranscodeHWRequested", event.TranscodeHWRequested, 1)
				checkIntPtrEqual(t, "TranscodeHWDecoding", event.TranscodeHWDecoding, 1)
				checkIntPtrEqual(t, "TranscodeHWEncoding", event.TranscodeHWEncoding, 1)
				checkIntPtrEqual(t, "TranscodeHWFullPipeline", event.TranscodeHWFullPipeline, 1)
				checkStringPtrEqual(t, "TranscodeHWDecode", event.TranscodeHWDecode, "nvdec")
				checkStringPtrEqual(t, "TranscodeHWDecodeTitle", event.TranscodeHWDecodeTitle, "NVIDIA NVDEC")
				checkStringPtrEqual(t, "TranscodeHWEncode", event.TranscodeHWEncode, "nvenc")
				checkStringPtrEqual(t, "TranscodeHWEncodeTitle", event.TranscodeHWEncodeTitle, "NVIDIA NVENC")
				checkStringPtrEqual(t, "TranscodeContainer", event.TranscodeContainer, "mkv")
				checkStringPtrEqual(t, "TranscodeVideoCodec", event.TranscodeVideoCodec, "hevc")
				checkStringPtrEqual(t, "TranscodeAudioCodec", event.TranscodeAudioCodec, "aac")
				checkIntPtrEqual(t, "TranscodeAudioChannels", event.TranscodeAudioChannels, 6)
			},
		},
		{
			name: "Intel Quick Sync hardware transcode",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:              stringPtr("hw-intel"),
				Started:                 time.Now().Unix(),
				TranscodeHWRequested:    intPtr(1),
				TranscodeHWDecoding:     intPtr(1),
				TranscodeHWEncoding:     intPtr(1),
				TranscodeHWFullPipeline: intPtr(1),
				TranscodeHWDecode:       "qsv",
				TranscodeHWDecodeTitle:  "Intel Quick Sync Video",
				TranscodeHWEncode:       "qsv",
				TranscodeHWEncodeTitle:  "Intel Quick Sync Video",
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkStringPtrEqual(t, "TranscodeHWDecode", event.TranscodeHWDecode, "qsv")
				checkStringPtrEqual(t, "TranscodeHWDecodeTitle", event.TranscodeHWDecodeTitle, "Intel Quick Sync Video")
				checkStringPtrEqual(t, "TranscodeHWEncode", event.TranscodeHWEncode, "qsv")
				checkStringPtrEqual(t, "TranscodeHWEncodeTitle", event.TranscodeHWEncodeTitle, "Intel Quick Sync Video")
			},
		},
		{
			name: "software transcode (no hardware)",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:              stringPtr("hw-none"),
				Started:                 time.Now().Unix(),
				TranscodeHWRequested:    intPtr(0),
				TranscodeHWDecoding:     intPtr(0),
				TranscodeHWEncoding:     intPtr(0),
				TranscodeHWFullPipeline: intPtr(0),
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkIntPtrEqual(t, "TranscodeHWRequested", event.TranscodeHWRequested, 0)
				checkIntPtrEqual(t, "TranscodeHWDecoding", event.TranscodeHWDecoding, 0)
				checkIntPtrEqual(t, "TranscodeHWEncoding", event.TranscodeHWEncoding, 0)
				checkIntPtrEqual(t, "TranscodeHWFullPipeline", event.TranscodeHWFullPipeline, 0)
			},
		},
		{
			name: "throttled transcode",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:         stringPtr("hw-throttled"),
				Started:            time.Now().Unix(),
				TranscodeThrottled: intPtr(1),
				TranscodeSpeed:     "0.8x",
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkIntPtrEqual(t, "TranscodeThrottled", event.TranscodeThrottled, 1)
				checkStringPtrEqual(t, "TranscodeSpeed", event.TranscodeSpeed, "0.8x")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := m.buildCoreEvent(tt.record)
			m.mapHardwareTranscodeFields(tt.record, event)
			tt.verify(t, event)
		})
	}
}

// TestMapHDRColorMetadataFields tests HDR/color metadata field mapping
// CRITICAL: Tests HDR10/HDR10+/Dolby Vision detection
func TestMapHDRColorMetadataFields(t *testing.T) {
	m := &Manager{}

	tests := []struct {
		name   string
		record *tautulli.TautulliHistoryRecord
		verify func(t *testing.T, event *models.PlaybackEvent)
	}{
		{
			name: "HDR10 content (bt2020nc + smpte2084)",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:             stringPtr("hdr10"),
				Started:                time.Now().Unix(),
				VideoColorPrimaries:    "bt2020",
				VideoColorRange:        "tv",
				VideoColorSpace:        "bt2020nc",
				VideoColorTrc:          "smpte2084",
				VideoChromaSubsampling: "4:2:0",
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkStringPtrEqual(t, "VideoColorPrimaries", event.VideoColorPrimaries, "bt2020")
				checkStringPtrEqual(t, "VideoColorRange", event.VideoColorRange, "tv")
				checkStringPtrEqual(t, "VideoColorSpace", event.VideoColorSpace, "bt2020nc")
				checkStringPtrEqual(t, "VideoColorTrc", event.VideoColorTrc, "smpte2084")
				checkStringPtrEqual(t, "VideoChromaSubsampling", event.VideoChromaSubsampling, "4:2:0")
			},
		},
		{
			name: "Dolby Vision content",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:          stringPtr("dv"),
				Started:             time.Now().Unix(),
				VideoColorPrimaries: "bt2020",
				VideoColorRange:     "tv",
				VideoColorSpace:     "bt2020nc",
				VideoColorTrc:       "arib-std-b67", // HLG for DV
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkStringPtrEqual(t, "VideoColorTrc", event.VideoColorTrc, "arib-std-b67")
			},
		},
		{
			name: "SDR content",
			record: &tautulli.TautulliHistoryRecord{
				SessionKey:             stringPtr("sdr"),
				Started:                time.Now().Unix(),
				VideoColorPrimaries:    "bt709",
				VideoColorRange:        "tv",
				VideoColorSpace:        "bt709",
				VideoColorTrc:          "bt709",
				VideoChromaSubsampling: "4:2:0",
			},
			verify: func(t *testing.T, event *models.PlaybackEvent) {
				t.Helper()
				checkStringPtrEqual(t, "VideoColorPrimaries", event.VideoColorPrimaries, "bt709")
				checkStringPtrEqual(t, "VideoColorSpace", event.VideoColorSpace, "bt709")
				checkStringPtrEqual(t, "VideoColorTrc", event.VideoColorTrc, "bt709")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := m.buildCoreEvent(tt.record)
			m.mapHDRColorMetadataFields(tt.record, event)
			tt.verify(t, event)
		})
	}
}

// TestMapExtendedVideoFields tests extended video field mapping
func TestMapExtendedVideoFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:      stringPtr("video-ext"),
		Started:         time.Now().Unix(),
		VideoCodecLevel: "5.1",
		VideoProfile:    "High",
		VideoScanType:   "progressive",
		VideoLanguage:   "English",
		AspectRatio:     "2.39",
		VideoRefFrames:  intPtr(4),
	}

	event := m.buildCoreEvent(record)
	m.mapExtendedVideoFields(record, event)

	checkStringPtrEqual(t, "VideoCodecLevel", event.VideoCodecLevel, "5.1")
	checkStringPtrEqual(t, "VideoProfile", event.VideoProfile, "High")
	checkStringPtrEqual(t, "VideoScanType", event.VideoScanType, "progressive")
	checkStringPtrEqual(t, "VideoLanguage", event.VideoLanguage, "English")
	checkStringPtrEqual(t, "AspectRatio", event.AspectRatio, "2.39")
	checkIntPtrEqual(t, "VideoRefFrames", event.VideoRefFrames, 4)
}

// TestMapExtendedAudioFields tests extended audio field mapping
func TestMapExtendedAudioFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:       stringPtr("audio-ext"),
		Started:          time.Now().Unix(),
		AudioProfile:     "LC",
		AudioBitrateMode: "CBR",
	}

	event := m.buildCoreEvent(record)
	m.mapExtendedAudioFields(record, event)

	checkStringPtrEqual(t, "AudioProfile", event.AudioProfile, "LC")
	checkStringPtrEqual(t, "AudioBitrateMode", event.AudioBitrateMode, "CBR")
}

// TestMapExtendedStreamFields tests extended stream output field mapping
func TestMapExtendedStreamFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:               stringPtr("stream-ext"),
		Started:                  time.Now().Unix(),
		StreamVideoCodec:         "hevc",
		StreamVideoWidth:         intPtr(1920),
		StreamVideoHeight:        intPtr(1080),
		StreamAudioBitrate:       intPtr(384),
		StreamAudioChannelLayout: "5.1",
		StreamVideoDynamicRange:  "HDR",
	}

	event := m.buildCoreEvent(record)
	m.mapExtendedStreamFields(record, event)

	checkStringPtrEqual(t, "StreamVideoCodec", event.StreamVideoCodec, "hevc")
	checkIntPtrEqual(t, "StreamVideoWidth", event.StreamVideoWidth, 1920)
	checkIntPtrEqual(t, "StreamVideoHeight", event.StreamVideoHeight, 1080)
	checkIntPtrEqual(t, "StreamAudioBitrate", event.StreamAudioBitrate, 384)
	checkStringPtrEqual(t, "StreamAudioChannelLayout", event.StreamAudioChannelLayout, "5.1")
	checkStringPtrEqual(t, "StreamVideoDynamicRange", event.StreamVideoDynamicRange, "HDR")
}

// TestMapExtendedSubtitleFields tests extended subtitle field mapping
func TestMapExtendedSubtitleFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:             stringPtr("subtitle-ext"),
		Started:                time.Now().Unix(),
		StreamSubtitleCodec:    "ass",
		StreamSubtitleLanguage: "Japanese",
		SubtitleContainer:      "mkv",
		SubtitleForced:         intPtr(1),
	}

	event := m.buildCoreEvent(record)
	m.mapExtendedSubtitleFields(record, event)

	checkStringPtrEqual(t, "StreamSubtitleCodec", event.StreamSubtitleCodec, "ass")
	checkStringPtrEqual(t, "StreamSubtitleLanguage", event.StreamSubtitleLanguage, "Japanese")
	checkStringPtrEqual(t, "SubtitleContainer", event.SubtitleContainer, "mkv")
	checkIntPtrEqual(t, "SubtitleForced", event.SubtitleForced, 1)
}

// TestMapExtendedConnectionFields tests extended connection field mapping
func TestMapExtendedConnectionFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("conn-ext"),
		Started:    time.Now().Unix(),
		Bandwidth:  intPtr(50000),
		Location:   "lan",
		Relay:      intPtr(1),
	}

	event := m.buildCoreEvent(record)
	m.mapExtendedConnectionFields(record, event)

	checkIntPtrEqual(t, "Bandwidth", event.Bandwidth, 50000)
	checkStringPtrEqual(t, "Location", event.Location, "lan")
	checkIntPtrEqual(t, "Relay", event.Relay, 1)
	// Note: BandwidthLAN/BandwidthWAN are server-level settings, not per-history record
}

// TestMapExtendedFileFields tests extended file metadata field mapping
func TestMapExtendedFileFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("file-ext"),
		Started:    time.Now().Unix(),
		File:       "/media/movies/Movie.mkv",
	}

	event := m.buildCoreEvent(record)
	m.mapExtendedFileFields(record, event)

	checkStringPtrEqual(t, "File", event.File, "/media/movies/Movie.mkv")
}

// TestMapThumbnailFields tests thumbnail field mapping
func TestMapThumbnailFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:       stringPtr("thumb123"),
		Started:          time.Now().Unix(),
		ParentThumb:      "/library/metadata/12345/thumb",
		GrandparentThumb: "/library/metadata/67890/thumb",
		Art:              "/library/metadata/12345/art",
		GrandparentArt:   "/library/metadata/67890/art",
	}

	event := m.buildCoreEvent(record)
	m.mapThumbnailFields(record, event)

	checkStringPtrEqual(t, "ParentThumb", event.ParentThumb, "/library/metadata/12345/thumb")
	checkStringPtrEqual(t, "GrandparentThumb", event.GrandparentThumb, "/library/metadata/67890/thumb")
	checkStringPtrEqual(t, "Art", event.Art, "/library/metadata/12345/art")
	checkStringPtrEqual(t, "GrandparentArt", event.GrandparentArt, "/library/metadata/67890/art")
}

// TestMapExtendedGUIDFields tests extended GUID field mapping
func TestMapExtendedGUIDFields(t *testing.T) {
	m := &Manager{}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey:      stringPtr("guid123"),
		Started:         time.Now().Unix(),
		ParentGuid:      "plex://season/12345",
		GrandparentGuid: "plex://show/67890",
	}

	event := m.buildCoreEvent(record)
	m.mapExtendedGUIDFields(record, event)

	checkStringPtrEqual(t, "ParentGuid", event.ParentGuid, "plex://season/12345")
	checkStringPtrEqual(t, "GrandparentGuid", event.GrandparentGuid, "plex://show/67890")
}

// TestEnrichEventWithExtendedData tests the complete extended data enrichment
func TestEnrichEventWithExtendedData(t *testing.T) {
	m := &Manager{}

	// Create a comprehensive record with all extended fields
	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("complete123"),
		Started:    time.Now().Unix(),
		// User fields
		FriendlyName:    "Test User",
		UserThumb:       "https://plex.tv/users/test/avatar",
		Email:           "test@example.com",
		IPAddressPublic: "203.0.113.100",
		// Client device fields
		PlatformName:    "Android",
		PlatformVersion: "13",
		Product:         "Plex for Android",
		ProductVersion:  "9.5.0",
		Device:          "Pixel 7",
		MachineID:       "device123",
		QualityProfile:  "Original",
		// Transcode decision fields
		VideoDecision:    "transcode",
		AudioDecision:    "copy",
		SubtitleDecision: "burn",
		// Hardware transcode fields
		TranscodeHWRequested:    intPtr(1),
		TranscodeHWDecoding:     intPtr(1),
		TranscodeHWEncoding:     intPtr(1),
		TranscodeHWFullPipeline: intPtr(1),
		TranscodeHWDecode:       "nvdec",
		TranscodeHWEncode:       "nvenc",
		// HDR color metadata
		VideoColorPrimaries: "bt2020",
		VideoColorSpace:     "bt2020nc",
		VideoColorTrc:       "smpte2084",
		// Extended video fields
		VideoCodecLevel: "5.1",
		VideoProfile:    "Main 10",
		// Extended audio fields
		AudioProfile: "LC",
		// Extended stream fields
		StreamVideoCodec:  "hevc",
		StreamVideoWidth:  intPtr(3840),
		StreamVideoHeight: intPtr(2160),
		// Thumbnail fields
		ParentThumb:      "/library/metadata/parent/thumb",
		GrandparentThumb: "/library/metadata/grandparent/thumb",
		// GUID fields
		ParentGuid:      "plex://season/parent",
		GrandparentGuid: "plex://show/grandparent",
	}

	event := m.buildCoreEvent(record)
	m.enrichEventWithExtendedData(event, record)

	// Verify user fields
	checkStringPtrEqual(t, "FriendlyName", event.FriendlyName, "Test User")
	checkStringPtrEqual(t, "IPAddressPublic", event.IPAddressPublic, "203.0.113.100")

	// Verify client device fields
	checkStringPtrEqual(t, "PlatformName", event.PlatformName, "Android")
	checkStringPtrEqual(t, "MachineID", event.MachineID, "device123")

	// Verify transcode decision fields
	checkStringPtrEqual(t, "VideoDecision", event.VideoDecision, "transcode")
	checkStringPtrEqual(t, "AudioDecision", event.AudioDecision, "copy")

	// Verify hardware transcode fields
	checkIntPtrEqual(t, "TranscodeHWDecoding", event.TranscodeHWDecoding, 1)
	checkIntPtrEqual(t, "TranscodeHWEncoding", event.TranscodeHWEncoding, 1)
	checkStringPtrEqual(t, "TranscodeHWDecode", event.TranscodeHWDecode, "nvdec")
	checkStringPtrEqual(t, "TranscodeHWEncode", event.TranscodeHWEncode, "nvenc")

	// Verify HDR color metadata
	checkStringPtrEqual(t, "VideoColorPrimaries", event.VideoColorPrimaries, "bt2020")
	checkStringPtrEqual(t, "VideoColorTrc", event.VideoColorTrc, "smpte2084")

	// Verify extended video fields
	checkStringPtrEqual(t, "VideoCodecLevel", event.VideoCodecLevel, "5.1")
	checkStringPtrEqual(t, "VideoProfile", event.VideoProfile, "Main 10")

	// Verify extended stream fields
	checkStringPtrEqual(t, "StreamVideoCodec", event.StreamVideoCodec, "hevc")
	checkIntPtrEqual(t, "StreamVideoWidth", event.StreamVideoWidth, 3840)
	checkIntPtrEqual(t, "StreamVideoHeight", event.StreamVideoHeight, 2160)

	// Verify thumbnail fields
	checkStringPtrEqual(t, "ParentThumb", event.ParentThumb, "/library/metadata/parent/thumb")

	// Verify GUID fields
	checkStringPtrEqual(t, "ParentGuid", event.ParentGuid, "plex://season/parent")
	checkStringPtrEqual(t, "GrandparentGuid", event.GrandparentGuid, "plex://show/grandparent")
}

// ========================================
// v1.44 API Coverage Expansion Tests
// ========================================

// TestMapUserPermissionFields tests the v1.44 user permission/status field mapping
func TestMapUserPermissionFields(t *testing.T) {
	m := &Manager{}
	record := &tautulli.TautulliHistoryRecord{
		SessionKey:   stringPtr("perm123"),
		Started:      time.Now().Unix(),
		IsAdmin:      intPtr(1),
		IsHomeUser:   intPtr(0),
		IsAllowSync:  intPtr(1),
		IsRestricted: intPtr(0),
		KeepHistory:  intPtr(1),
		DeletedUser:  intPtr(0),
		DoNotify:     intPtr(1),
	}

	event := m.buildCoreEvent(record)
	m.mapUserPermissionFields(record, event)

	checkIntPtrEqual(t, "IsAdmin", event.IsAdmin, 1)
	checkIntPtrEqual(t, "IsHomeUser", event.IsHomeUser, 0)
	checkIntPtrEqual(t, "IsAllowSync", event.IsAllowSync, 1)
	checkIntPtrEqual(t, "IsRestricted", event.IsRestricted, 0)
	checkIntPtrEqual(t, "KeepHistory", event.KeepHistory, 1)
	checkIntPtrEqual(t, "DeletedUser", event.DeletedUser, 0)
	checkIntPtrEqual(t, "DoNotify", event.DoNotify, 1)
}

// TestMapMediaMetadataFields tests the v1.44 media metadata field mapping
func TestMapMediaMetadataFields(t *testing.T) {
	m := &Manager{}
	record := &tautulli.TautulliHistoryRecord{
		SessionKey:     stringPtr("media123"),
		Started:        time.Now().Unix(),
		Studio:         "Warner Bros.",
		AddedAt:        "1698000000",
		UpdatedAt:      "1699000000",
		LastViewedAt:   "1700000000",
		Summary:        "An exciting action movie with stunning visuals.",
		Tagline:        "Every ending has a beginning.",
		Rating:         "8.5",
		AudienceRating: "9.0",
		UserRating:     "4.5",
		Labels:         "action,blockbuster",
		Collections:    "Marvel,Avengers",
		Banner:         "/library/metadata/123/banner",
	}

	event := m.buildCoreEvent(record)
	m.mapMediaMetadataFields(record, event)

	checkStringPtrEqual(t, "Studio", event.Studio, "Warner Bros.")
	checkStringPtrEqual(t, "AddedAt", event.AddedAt, "1698000000")
	checkStringPtrEqual(t, "UpdatedAt", event.UpdatedAt, "1699000000")
	checkStringPtrEqual(t, "LastViewedAt", event.LastViewedAt, "1700000000")
	checkStringPtrEqual(t, "Summary", event.Summary, "An exciting action movie with stunning visuals.")
	checkStringPtrEqual(t, "Tagline", event.Tagline, "Every ending has a beginning.")
	checkStringPtrEqual(t, "Rating", event.Rating, "8.5")
	checkStringPtrEqual(t, "AudienceRating", event.AudienceRating, "9.0")
	checkStringPtrEqual(t, "UserRating", event.UserRating, "4.5")
	checkStringPtrEqual(t, "Labels", event.Labels, "action,blockbuster")
	checkStringPtrEqual(t, "Collections", event.Collections, "Marvel,Avengers")
	checkStringPtrEqual(t, "Banner", event.Banner, "/library/metadata/123/banner")
}

// TestMapLanguageCodeFields tests the v1.44 language code field mapping
func TestMapLanguageCodeFields(t *testing.T) {
	m := &Manager{}
	record := &tautulli.TautulliHistoryRecord{
		SessionKey:           stringPtr("lang123"),
		Started:              time.Now().Unix(),
		VideoLanguageCode:    "eng",
		VideoFullResolution:  "1080p",
		AudioLanguageCode:    "jpn",
		SubtitleLanguageCode: "spa",
		SubtitleContainer:    "srt",
		SubtitleFormat:       "srt",
	}

	event := m.buildCoreEvent(record)
	m.mapLanguageCodeFields(record, event)

	checkStringPtrEqual(t, "VideoLanguageCode", event.VideoLanguageCode, "eng")
	checkStringPtrEqual(t, "VideoFullResolution", event.VideoFullResolution, "1080p")
	checkStringPtrEqual(t, "AudioLanguageCode", event.AudioLanguageCode, "jpn")
	checkStringPtrEqual(t, "SubtitleLanguageCode", event.SubtitleLanguageCode, "spa")
	checkStringPtrEqual(t, "SubtitleContainerFmt", event.SubtitleContainerFmt, "srt")
	checkStringPtrEqual(t, "SubtitleFormat", event.SubtitleFormat, "srt")
}

// TestMapExtendedStreamV144Fields tests the v1.44 extended stream field mapping
func TestMapExtendedStreamV144Fields(t *testing.T) {
	m := &Manager{}
	record := &tautulli.TautulliHistoryRecord{
		SessionKey:                 stringPtr("stream144"),
		Started:                    time.Now().Unix(),
		StreamVideoCodecLevel:      "5.1",
		StreamVideoFullResolution:  "2160p",
		StreamVideoBitDepth:        intPtr(10),
		StreamVideoFramerate:       "23.976",
		StreamVideoProfile:         "Main 10",
		StreamVideoScanType:        "progressive",
		StreamVideoLanguage:        "English",
		StreamVideoLanguageCode:    "eng",
		StreamAudioBitrateMode:     "vbr",
		StreamAudioSampleRate:      intPtr(48000),
		StreamAudioLanguage:        "Japanese",
		StreamAudioLanguageCode:    "jpn",
		StreamAudioProfile:         "lc",
		StreamSubtitleContainer:    "srt",
		StreamSubtitleFormat:       "srt",
		StreamSubtitleForced:       intPtr(0),
		StreamSubtitleLocation:     "external",
		StreamSubtitleLanguageCode: "eng",
		StreamSubtitleDecision:     "burn",
	}

	event := m.buildCoreEvent(record)
	m.mapExtendedStreamV144Fields(record, event)

	// Extended stream video fields
	checkStringPtrEqual(t, "StreamVideoCodecLevel", event.StreamVideoCodecLevel, "5.1")
	checkStringPtrEqual(t, "StreamVideoFullResolution", event.StreamVideoFullResolution, "2160p")
	checkIntPtrEqual(t, "StreamVideoBitDepth", event.StreamVideoBitDepth, 10)
	checkStringPtrEqual(t, "StreamVideoFramerate", event.StreamVideoFramerate, "23.976")
	checkStringPtrEqual(t, "StreamVideoProfile", event.StreamVideoProfile, "Main 10")
	checkStringPtrEqual(t, "StreamVideoScanType", event.StreamVideoScanType, "progressive")
	checkStringPtrEqual(t, "StreamVideoLanguage", event.StreamVideoLanguage, "English")
	checkStringPtrEqual(t, "StreamVideoLanguageCode", event.StreamVideoLanguageCode, "eng")

	// Extended stream audio fields
	checkStringPtrEqual(t, "StreamAudioBitrateMode", event.StreamAudioBitrateMode, "vbr")
	checkIntPtrEqual(t, "StreamAudioSampleRate", event.StreamAudioSampleRate, 48000)
	checkStringPtrEqual(t, "StreamAudioLanguage", event.StreamAudioLanguage, "Japanese")
	checkStringPtrEqual(t, "StreamAudioLanguageCode", event.StreamAudioLanguageCode, "jpn")
	checkStringPtrEqual(t, "StreamAudioProfile", event.StreamAudioProfile, "lc")

	// Extended stream subtitle fields
	checkStringPtrEqual(t, "StreamSubtitleContainer", event.StreamSubtitleContainer, "srt")
	checkStringPtrEqual(t, "StreamSubtitleFormat", event.StreamSubtitleFormat, "srt")
	checkIntPtrEqual(t, "StreamSubtitleForced", event.StreamSubtitleForced, 0)
	checkStringPtrEqual(t, "StreamSubtitleLocation", event.StreamSubtitleLocation, "external")
	checkStringPtrEqual(t, "StreamSubtitleLanguageCode", event.StreamSubtitleLanguageCode, "eng")
	checkStringPtrEqual(t, "StreamSubtitleDecision", event.StreamSubtitleDecision, "burn")
}

// TestEnrichEventWithExtendedDataV144 tests that v1.44 fields are included in full enrichment
func TestEnrichEventWithExtendedDataV144(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("full144"),
		Started:    time.Now().Unix(),
		// v1.44 User permission fields
		IsAdmin:     intPtr(1),
		IsHomeUser:  intPtr(0),
		KeepHistory: intPtr(1),
		// v1.44 Media metadata fields
		Studio:  "Universal",
		Summary: "Test summary",
		Rating:  "7.5",
		// v1.44 Language code fields
		VideoLanguageCode: "eng",
		AudioLanguageCode: "spa",
		// v1.44 Extended stream fields
		StreamVideoCodecLevel:   "4.1",
		StreamAudioLanguageCode: "fra",
		StreamSubtitleDecision:  "embed",
	}

	event := m.buildCoreEvent(record)
	m.enrichEventWithExtendedData(event, record)

	// Verify v1.44 fields are enriched
	checkIntPtrEqual(t, "IsAdmin", event.IsAdmin, 1)
	checkIntPtrEqual(t, "IsHomeUser", event.IsHomeUser, 0)
	checkIntPtrEqual(t, "KeepHistory", event.KeepHistory, 1)
	checkStringPtrEqual(t, "Studio", event.Studio, "Universal")
	checkStringPtrEqual(t, "Summary", event.Summary, "Test summary")
	checkStringPtrEqual(t, "Rating", event.Rating, "7.5")
	checkStringPtrEqual(t, "VideoLanguageCode", event.VideoLanguageCode, "eng")
	checkStringPtrEqual(t, "AudioLanguageCode", event.AudioLanguageCode, "spa")
	checkStringPtrEqual(t, "StreamVideoCodecLevel", event.StreamVideoCodecLevel, "4.1")
	checkStringPtrEqual(t, "StreamAudioLanguageCode", event.StreamAudioLanguageCode, "fra")
	checkStringPtrEqual(t, "StreamSubtitleDecision", event.StreamSubtitleDecision, "embed")
}

// ========================================
// v1.45 API Coverage Expansion Tests
// ========================================

// TestMapUserLibraryAccessFields tests the v1.45 user library access field mapping
func TestMapUserLibraryAccessFields(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		SharedLibraries: "1;2;5;10",
	}

	event := &models.PlaybackEvent{}
	m.mapUserLibraryAccessFields(record, event)

	checkStringPtrEqual(t, "SharedLibraries", event.SharedLibraries, "1;2;5;10")
}

// TestMapUserLibraryAccessFieldsEmpty tests empty shared_libraries handling
func TestMapUserLibraryAccessFieldsEmpty(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		SharedLibraries: "",
	}

	event := &models.PlaybackEvent{}
	m.mapUserLibraryAccessFields(record, event)

	if event.SharedLibraries != nil {
		t.Errorf("SharedLibraries should be nil for empty string, got %v", *event.SharedLibraries)
	}
}

// TestMapMediaIdentificationV145Fields tests the v1.45 media identification field mapping
func TestMapMediaIdentificationV145Fields(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		SortTitle: "Matrix, The",
	}

	event := &models.PlaybackEvent{}
	m.mapMediaIdentificationV145Fields(record, event)

	checkStringPtrEqual(t, "SortTitle", event.SortTitle, "Matrix, The")
}

// TestMapMediaIdentificationV145FieldsEmpty tests empty sort_title handling
func TestMapMediaIdentificationV145FieldsEmpty(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		SortTitle: "",
	}

	event := &models.PlaybackEvent{}
	m.mapMediaIdentificationV145Fields(record, event)

	if event.SortTitle != nil {
		t.Errorf("SortTitle should be nil for empty string, got %v", *event.SortTitle)
	}
}

// TestMapPlaybackStateV145Fields tests the v1.45 playback state field mapping
func TestMapPlaybackStateV145Fields(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	tests := []struct {
		name      string
		throttled *int
		expected  int
	}{
		{"throttled active", intPtr(1), 1},
		{"not throttled", intPtr(0), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &tautulli.TautulliHistoryRecord{
				Throttled: tt.throttled,
			}
			event := &models.PlaybackEvent{}
			m.mapPlaybackStateV145Fields(record, event)
			checkIntPtrEqual(t, "Throttled", event.Throttled, tt.expected)
		})
	}
}

// TestMapLiveTVFields tests the v1.45 Live TV field mapping
func TestMapLiveTVFields(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		Live:              intPtr(1),
		LiveUUID:          "abc123-def456-789",
		ChannelStream:     intPtr(5),
		ChannelCallSign:   "NBC",
		ChannelIdentifier: "nbc-hd",
	}

	event := &models.PlaybackEvent{}
	m.mapLiveTVFields(record, event)

	checkIntPtrEqual(t, "Live", event.Live, 1)
	checkStringPtrEqual(t, "LiveUUID", event.LiveUUID, "abc123-def456-789")
	checkIntPtrEqual(t, "ChannelStream", event.ChannelStream, 5)
	checkStringPtrEqual(t, "ChannelCallSign", event.ChannelCallSign, "NBC")
	checkStringPtrEqual(t, "ChannelIdentifier", event.ChannelIdentifier, "nbc-hd")
}

// TestMapLiveTVFieldsNotLive tests Live TV field mapping for non-Live TV content
func TestMapLiveTVFieldsNotLive(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		Live:              intPtr(0),
		LiveUUID:          "",
		ChannelStream:     intPtr(0),
		ChannelCallSign:   "",
		ChannelIdentifier: "",
	}

	event := &models.PlaybackEvent{}
	m.mapLiveTVFields(record, event)

	checkIntPtrEqual(t, "Live", event.Live, 0)
	if event.LiveUUID != nil {
		t.Errorf("LiveUUID should be nil for empty string, got %v", *event.LiveUUID)
	}
	if event.ChannelCallSign != nil {
		t.Errorf("ChannelCallSign should be nil for empty string, got %v", *event.ChannelCallSign)
	}
	if event.ChannelIdentifier != nil {
		t.Errorf("ChannelIdentifier should be nil for empty string, got %v", *event.ChannelIdentifier)
	}
}

// TestEnrichEventWithExtendedDataV145 tests that v1.45 fields are included in full enrichment
func TestEnrichEventWithExtendedDataV145(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("full145"),
		Started:    time.Now().Unix(),
		// v1.45 User library access fields
		SharedLibraries: "1;2;3",
		// v1.45 Media identification fields
		SortTitle: "Avengers, The",
		// v1.45 Playback state fields
		Throttled: intPtr(1),
		// v1.45 Live TV fields
		Live:              intPtr(1),
		LiveUUID:          "live-session-uuid",
		ChannelStream:     intPtr(7),
		ChannelCallSign:   "ESPN",
		ChannelIdentifier: "espn-hd",
	}

	event := m.buildCoreEvent(record)
	m.enrichEventWithExtendedData(event, record)

	// Verify v1.45 fields are enriched
	checkStringPtrEqual(t, "SharedLibraries", event.SharedLibraries, "1;2;3")
	checkStringPtrEqual(t, "SortTitle", event.SortTitle, "Avengers, The")
	checkIntPtrEqual(t, "Throttled", event.Throttled, 1)
	checkIntPtrEqual(t, "Live", event.Live, 1)
	checkStringPtrEqual(t, "LiveUUID", event.LiveUUID, "live-session-uuid")
	checkIntPtrEqual(t, "ChannelStream", event.ChannelStream, 7)
	checkStringPtrEqual(t, "ChannelCallSign", event.ChannelCallSign, "ESPN")
	checkStringPtrEqual(t, "ChannelIdentifier", event.ChannelIdentifier, "espn-hd")
}

// ========================================
// v1.46 API Coverage Expansion Tests
// ========================================

// TestMapGroupedPlaybackFields tests the v1.46 grouped playback field mapping
func TestMapGroupedPlaybackFields(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		GroupCount: intPtr(5),
		GroupIDs:   "1,2,3,4,5",
		State:      stringPtr("playing"),
	}

	event := &models.PlaybackEvent{}
	m.mapGroupedPlaybackFields(record, event)

	checkIntPtrEqual(t, "GroupCount", event.GroupCount, 5)
	checkStringPtrEqual(t, "GroupIDs", event.GroupIDs, "1,2,3,4,5")
	checkStringPtrEqual(t, "State", event.State, "playing")
}

// TestMapGroupedPlaybackFieldsEmpty tests empty grouped playback field handling
func TestMapGroupedPlaybackFieldsEmpty(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		GroupCount: intPtr(0),
		GroupIDs:   "",
		State:      nil,
	}

	event := &models.PlaybackEvent{}
	m.mapGroupedPlaybackFields(record, event)

	// GroupCount of 0 should still be mapped (it's a valid value)
	checkIntPtrEqual(t, "GroupCount", event.GroupCount, 0)
	if event.GroupIDs != nil {
		t.Errorf("GroupIDs should be nil for empty string, got %v", *event.GroupIDs)
	}
	if event.State != nil {
		t.Errorf("State should be nil for empty string, got %v", *event.State)
	}
}

// TestMapUserPermissionV146Fields tests the v1.46 user permission field mapping
func TestMapUserPermissionV146Fields(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	tests := []struct {
		name       string
		allowGuest *int
		expected   int
	}{
		{"guest allowed", intPtr(1), 1},
		{"guest not allowed", intPtr(0), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &tautulli.TautulliHistoryRecord{
				AllowGuest: tt.allowGuest,
			}
			event := &models.PlaybackEvent{}
			m.mapUserPermissionV146Fields(record, event)
			checkIntPtrEqual(t, "AllowGuest", event.AllowGuest, tt.expected)
		})
	}
}

// TestMapStreamAspectRatioFields tests the v1.46 stream aspect ratio field mapping
func TestMapStreamAspectRatioFields(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	tests := []struct {
		name              string
		streamAspectRatio string
		expectNil         bool
	}{
		{"widescreen 16:9", "1.78", false},
		{"cinemascope 2.39:1", "2.39", false},
		{"4:3 standard", "1.33", false},
		{"empty aspect ratio", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &tautulli.TautulliHistoryRecord{
				StreamAspectRatio: tt.streamAspectRatio,
			}
			event := &models.PlaybackEvent{}
			m.mapStreamAspectRatioFields(record, event)

			if tt.expectNil {
				if event.StreamAspectRatio != nil {
					t.Errorf("StreamAspectRatio should be nil for empty string, got %v", *event.StreamAspectRatio)
				}
			} else {
				checkStringPtrEqual(t, "StreamAspectRatio", event.StreamAspectRatio, tt.streamAspectRatio)
			}
		})
	}
}

// TestMapTranscodeOutputDimensionFields tests the v1.46 transcode output dimension field mapping
func TestMapTranscodeOutputDimensionFields(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	tests := []struct {
		name                 string
		transcodeVideoWidth  int
		transcodeVideoHeight int
	}{
		{"4K output", 3840, 2160},
		{"1080p output", 1920, 1080},
		{"720p output", 1280, 720},
		{"480p output", 854, 480},
		{"no transcode (zeros)", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &tautulli.TautulliHistoryRecord{
				TranscodeVideoWidth:  intPtr(tt.transcodeVideoWidth),
				TranscodeVideoHeight: intPtr(tt.transcodeVideoHeight),
			}
			event := &models.PlaybackEvent{}
			m.mapTranscodeOutputDimensionFields(record, event)

			checkIntPtrEqual(t, "TranscodeVideoWidth", event.TranscodeVideoWidth, tt.transcodeVideoWidth)
			checkIntPtrEqual(t, "TranscodeVideoHeight", event.TranscodeVideoHeight, tt.transcodeVideoHeight)
		})
	}
}

// TestEnrichEventWithExtendedDataV146 tests that v1.46 fields are included in full enrichment
func TestEnrichEventWithExtendedDataV146(t *testing.T) {
	m := &Manager{
		cfg: &config.Config{},
	}

	record := &tautulli.TautulliHistoryRecord{
		SessionKey: stringPtr("full146"),
		Started:    time.Now().Unix(),
		// v1.46 Grouped playback fields
		GroupCount: intPtr(3),
		GroupIDs:   "101,102,103",
		State:      stringPtr("paused"),
		// v1.46 User permission fields
		AllowGuest: intPtr(1),
		// v1.46 Stream aspect ratio
		StreamAspectRatio: "2.35",
		// v1.46 Transcode output dimensions
		TranscodeVideoWidth:  intPtr(1920),
		TranscodeVideoHeight: intPtr(800),
	}

	event := m.buildCoreEvent(record)
	m.enrichEventWithExtendedData(event, record)

	// Verify v1.46 fields are enriched
	checkIntPtrEqual(t, "GroupCount", event.GroupCount, 3)
	checkStringPtrEqual(t, "GroupIDs", event.GroupIDs, "101,102,103")
	checkStringPtrEqual(t, "State", event.State, "paused")
	checkIntPtrEqual(t, "AllowGuest", event.AllowGuest, 1)
	checkStringPtrEqual(t, "StreamAspectRatio", event.StreamAspectRatio, "2.35")
	checkIntPtrEqual(t, "TranscodeVideoWidth", event.TranscodeVideoWidth, 1920)
	checkIntPtrEqual(t, "TranscodeVideoHeight", event.TranscodeVideoHeight, 800)
}
