// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// ========================================
// Tautulli Event Mapping
// ========================================

// buildCoreEvent creates a PlaybackEvent with core fields from a history record
func (m *Manager) buildCoreEvent(record *tautulli.TautulliHistoryRecord) *models.PlaybackEvent {
	// Handle nullable PercentComplete and PausedCounter fields
	percentComplete := 0
	if record.PercentComplete != nil {
		percentComplete = *record.PercentComplete
	}
	pausedCounter := 0
	if record.PausedCounter != nil {
		pausedCounter = *record.PausedCounter
	}

	// Handle nullable UserID field
	userID := 0
	if record.UserID != nil {
		userID = *record.UserID
	}

	event := &models.PlaybackEvent{
		ID:              uuid.New(),
		Source:          "tautulli",
		SessionKey:      getEffectiveSessionKey(record),
		StartedAt:       time.Unix(record.Started, 0),
		UserID:          userID,
		Username:        record.User,
		IPAddress:       record.IPAddress,
		MediaType:       record.MediaType,
		Title:           record.Title,
		Platform:        record.Platform,
		Player:          record.Player,
		LocationType:    record.Location,
		PercentComplete: percentComplete,
		PausedCounter:   pausedCounter,
		CreatedAt:       time.Now(),
	}

	// Optional core fields
	if record.Stopped > 0 {
		stopped := time.Unix(record.Stopped, 0)
		event.StoppedAt = &stopped
	}

	// ParentTitle and GrandparentTitle are now pointers due to nullable JSON
	if record.ParentTitle != nil && *record.ParentTitle != "" {
		event.ParentTitle = record.ParentTitle
	}

	if record.GrandparentTitle != nil && *record.GrandparentTitle != "" {
		event.GrandparentTitle = record.GrandparentTitle
	}

	return event
}

// enrichEventWithMetadata enriches the event with metadata fields (library, ratings, etc.)
func (m *Manager) enrichEventWithMetadata(event *models.PlaybackEvent, record *tautulli.TautulliHistoryRecord) {
	m.mapLibraryFields(record, event)

	// Map metadata enrichment fields (added in v1.8)
	mapStringField(record.Guid, &event.Guid)
	mapStringPtrField(record.OriginalTitle, &event.OriginalTitle) // OriginalTitle is now nullable
	mapStringField(record.FullTitle, &event.FullTitle)
	mapStringPtrField(record.OriginallyAvailableAt, &event.OriginallyAvailableAt) // OriginallyAvailableAt is now nullable
	mapFloat64ToIntPtrField(record.WatchedStatus, &event.WatchedStatus)           // WatchedStatus: float64 (0.75) -> int (0)
	mapStringField(record.Thumb, &event.Thumb)
	mapStringField(record.Directors, &event.Directors)
	mapStringField(record.Writers, &event.Writers)
	mapStringField(record.Actors, &event.Actors)
	mapStringField(record.Genres, &event.Genres)
	mapIntPtrToStringField(record.RatingKey, &event.RatingKey)
	mapIntPtrToStringField(record.ParentRatingKey, &event.ParentRatingKey)
	mapIntPtrToStringField(record.GrandparentRatingKey, &event.GrandparentRatingKey)
	mapSignedIntPtrField(record.MediaIndex, &event.MediaIndex)
	mapSignedIntPtrField(record.ParentMediaIndex, &event.ParentMediaIndex)
}

// enrichEventWithQualityData enriches the event with quality and streaming data
func (m *Manager) enrichEventWithQualityData(event *models.PlaybackEvent, record *tautulli.TautulliHistoryRecord) {
	m.mapStreamingQualityFields(record, event)
	m.mapStreamDetailsFields(record, event)
}

// mapStreamingQualityFields adds streaming quality fields to an event
func (m *Manager) mapStreamingQualityFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.TranscodeDecision, &event.TranscodeDecision)
	mapStringField(record.VideoResolution, &event.VideoResolution)
	mapStringField(record.VideoCodec, &event.VideoCodec)
	mapStringField(record.AudioCodec, &event.AudioCodec)
}

// mapLibraryFields adds library and content metadata fields to an event
func (m *Manager) mapLibraryFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	// SectionID is now a pointer due to nullable JSON
	if record.SectionID != nil && *record.SectionID > 0 {
		event.SectionID = record.SectionID
	}

	mapStringField(record.LibraryName, &event.LibraryName)
	mapStringField(record.ContentRating, &event.ContentRating)

	// Convert duration from seconds to minutes (Duration is now a pointer)
	if record.Duration != nil && *record.Duration > 0 {
		durationMinutes := *record.Duration / 60
		event.PlayDuration = &durationMinutes
	}

	// Year is now a pointer due to nullable JSON
	if record.Year != nil && *record.Year > 0 {
		event.Year = record.Year
	}
}

// mapStreamDetailsFields adds detailed stream quality comparison fields to an event
func (m *Manager) mapStreamDetailsFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.StreamVideoResolution, &event.StreamVideoResolution)
	mapStringField(record.StreamAudioCodec, &event.StreamAudioCodec)
	mapStringField(record.StreamAudioChannels, &event.StreamAudioChannels)
	mapStringField(record.StreamVideoDecision, &event.StreamVideoDecision)
	mapStringField(record.StreamAudioDecision, &event.StreamAudioDecision)
	mapStringField(record.StreamContainer, &event.StreamContainer)
	mapIntPtrField(record.StreamBitrate, &event.StreamBitrate) // StreamBitrate is now nullable
}

// mapAudioFields adds audio technical details to an event
func (m *Manager) mapAudioFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.AudioChannels, &event.AudioChannels)
	mapStringField(record.AudioChannelLayout, &event.AudioChannelLayout)
	mapIntPtrField(record.AudioBitrate, &event.AudioBitrate)       // AudioBitrate is now nullable
	mapIntPtrField(record.AudioSampleRate, &event.AudioSampleRate) // AudioSampleRate is now nullable
	mapStringField(record.AudioLanguage, &event.AudioLanguage)
}

// mapVideoFields adds video technical details to an event
func (m *Manager) mapVideoFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.VideoDynamicRange, &event.VideoDynamicRange)
	mapStringField(record.VideoFrameRate, &event.VideoFrameRate)
	mapIntPtrField(record.VideoBitrate, &event.VideoBitrate)   // VideoBitrate is now nullable
	mapIntPtrField(record.VideoBitDepth, &event.VideoBitDepth) // VideoBitDepth is now nullable
	mapIntPtrField(record.VideoWidth, &event.VideoWidth)       // VideoWidth is now nullable
	mapIntPtrField(record.VideoHeight, &event.VideoHeight)     // VideoHeight is now nullable
}

// mapContainerSubtitleFields adds container and subtitle information to an event
func (m *Manager) mapContainerSubtitleFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.Container, &event.Container)
	mapStringField(record.SubtitleCodec, &event.SubtitleCodec)
	mapStringField(record.SubtitleLanguage, &event.SubtitleLanguage)
	mapSignedIntPtrField(record.Subtitles, &event.Subtitles) // Subtitles is now nullable
}

// mapConnectionFields adds connection security and locality fields to an event
func (m *Manager) mapConnectionFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapSignedIntPtrField(record.Secure, &event.Secure)   // Secure is now nullable
	mapSignedIntPtrField(record.Relayed, &event.Relayed) // Relayed is now nullable
	mapSignedIntPtrField(record.Local, &event.Local)     // Local is now nullable
}

// mapFileMetadataFields adds file size and bitrate information to an event
func (m *Manager) mapFileMetadataFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapInt64PtrField(record.FileSize, &event.FileSize) // FileSize is now nullable
	mapIntPtrField(record.Bitrate, &event.Bitrate)     // Bitrate is now nullable

	// v1.42: Bitrate analytics fields (3-level tracking)
	mapIntPtrField(record.Bitrate, &event.SourceBitrate)          // Source file bitrate (nullable)
	mapIntPtrField(record.StreamBitrate, &event.TranscodeBitrate) // Transcoded stream bitrate (nullable)
	// NetworkBandwidth: Not provided by Tautulli, remains NULL (future enhancement)
}

// mapEnrichmentFields adds metadata enrichment fields (CRITICAL for binge detection and analytics)
func (m *Manager) mapEnrichmentFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapIntPtrToStringField(record.RatingKey, &event.RatingKey)
	mapIntPtrToStringField(record.ParentRatingKey, &event.ParentRatingKey)
	mapIntPtrToStringField(record.GrandparentRatingKey, &event.GrandparentRatingKey)
	mapSignedIntPtrField(record.MediaIndex, &event.MediaIndex)
	mapSignedIntPtrField(record.ParentMediaIndex, &event.ParentMediaIndex)
	mapStringField(record.Guid, &event.Guid)
	mapStringPtrField(record.OriginalTitle, &event.OriginalTitle) // OriginalTitle is now nullable
	mapStringField(record.FullTitle, &event.FullTitle)
	mapStringPtrField(record.OriginallyAvailableAt, &event.OriginallyAvailableAt) // OriginallyAvailableAt is now nullable
	mapFloat64ToIntPtrField(record.WatchedStatus, &event.WatchedStatus)           // WatchedStatus: float64 (0.75) -> int (0)
	mapStringField(record.Thumb, &event.Thumb)
	mapStringField(record.Directors, &event.Directors)
	mapStringField(record.Writers, &event.Writers)
	mapStringField(record.Actors, &event.Actors)
	mapStringField(record.Genres, &event.Genres)
}

// ========================================
// Extended Field Mapping (v1.43 - API Coverage Audit)
// ========================================

// mapUserFields adds user information fields (v1.43 - API Coverage Audit)
func (m *Manager) mapUserFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.FriendlyName, &event.FriendlyName)
	mapStringField(record.UserThumb, &event.UserThumb)
	mapStringField(record.Email, &event.Email)
	mapStringField(record.IPAddressPublic, &event.IPAddressPublic)
}

// mapClientDeviceFields adds client/device identification fields (v1.43 - API Coverage Audit)
func (m *Manager) mapClientDeviceFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.PlatformName, &event.PlatformName)
	mapStringField(record.PlatformVersion, &event.PlatformVersion)
	mapStringField(record.Product, &event.Product)
	mapStringField(record.ProductVersion, &event.ProductVersion)
	mapStringField(record.Device, &event.Device)
	mapStringField(record.MachineID, &event.MachineID)
	mapStringField(record.QualityProfile, &event.QualityProfile)
	mapSignedIntPtrField(record.OptimizedVersion, &event.OptimizedVersion) // OptimizedVersion is now nullable
	mapSignedIntPtrField(record.SyncedVersion, &event.SyncedVersion)       // SyncedVersion is now nullable
}

// mapTranscodeDecisionFields adds transcode decision fields (v1.43 - API Coverage Audit)
func (m *Manager) mapTranscodeDecisionFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.VideoDecision, &event.VideoDecision)
	mapStringField(record.AudioDecision, &event.AudioDecision)
	mapStringField(record.SubtitleDecision, &event.SubtitleDecision)
}

// mapHardwareTranscodeFields adds hardware transcode fields (v1.43 - API Coverage Audit)
// CRITICAL for GPU utilization monitoring (NVIDIA NVENC, Intel Quick Sync, AMD VCE)
func (m *Manager) mapHardwareTranscodeFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.TranscodeKey, &event.TranscodeKey)
	mapSignedIntPtrField(record.TranscodeThrottled, &event.TranscodeThrottled) // TranscodeThrottled is now nullable
	mapSignedIntPtrField(record.TranscodeProgress, &event.TranscodeProgress)   // TranscodeProgress is now nullable
	mapStringField(record.TranscodeSpeed, &event.TranscodeSpeed)
	mapSignedIntPtrField(record.TranscodeHWRequested, &event.TranscodeHWRequested)       // TranscodeHWRequested is now nullable
	mapSignedIntPtrField(record.TranscodeHWDecoding, &event.TranscodeHWDecoding)         // TranscodeHWDecoding is now nullable
	mapSignedIntPtrField(record.TranscodeHWEncoding, &event.TranscodeHWEncoding)         // TranscodeHWEncoding is now nullable
	mapSignedIntPtrField(record.TranscodeHWFullPipeline, &event.TranscodeHWFullPipeline) // TranscodeHWFullPipeline is now nullable
	mapStringField(record.TranscodeHWDecode, &event.TranscodeHWDecode)
	mapStringField(record.TranscodeHWDecodeTitle, &event.TranscodeHWDecodeTitle)
	mapStringField(record.TranscodeHWEncode, &event.TranscodeHWEncode)
	mapStringField(record.TranscodeHWEncodeTitle, &event.TranscodeHWEncodeTitle)
	mapStringField(record.TranscodeContainer, &event.TranscodeContainer)
	mapStringField(record.TranscodeVideoCodec, &event.TranscodeVideoCodec)
	mapStringField(record.TranscodeAudioCodec, &event.TranscodeAudioCodec)
	mapSignedIntPtrField(record.TranscodeAudioChannels, &event.TranscodeAudioChannels) // TranscodeAudioChannels is now nullable
}

// mapHDRColorMetadataFields adds HDR/color metadata fields (v1.43 - API Coverage Audit)
// CRITICAL for HDR10/HDR10+/Dolby Vision detection
func (m *Manager) mapHDRColorMetadataFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.VideoColorPrimaries, &event.VideoColorPrimaries)
	mapStringField(record.VideoColorRange, &event.VideoColorRange)
	mapStringField(record.VideoColorSpace, &event.VideoColorSpace)
	mapStringField(record.VideoColorTrc, &event.VideoColorTrc)
	mapStringField(record.VideoChromaSubsampling, &event.VideoChromaSubsampling)
}

// mapExtendedVideoFields adds extended video fields (v1.43 - API Coverage Audit)
func (m *Manager) mapExtendedVideoFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.VideoCodecLevel, &event.VideoCodecLevel)
	mapStringField(record.VideoProfile, &event.VideoProfile)
	mapStringField(record.VideoScanType, &event.VideoScanType)
	mapStringField(record.VideoLanguage, &event.VideoLanguage)
	mapStringField(record.AspectRatio, &event.AspectRatio)
	mapIntPtrField(record.VideoRefFrames, &event.VideoRefFrames) // VideoRefFrames is now nullable
}

// mapExtendedAudioFields adds extended audio fields (v1.43 - API Coverage Audit)
func (m *Manager) mapExtendedAudioFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.AudioProfile, &event.AudioProfile)
	mapStringField(record.AudioBitrateMode, &event.AudioBitrateMode)
}

// mapExtendedStreamFields adds extended stream output fields (v1.43 - API Coverage Audit)
func (m *Manager) mapExtendedStreamFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.StreamVideoCodec, &event.StreamVideoCodec)
	mapIntPtrField(record.StreamVideoBitrate, &event.StreamVideoBitrate) // StreamVideoBitrate is now nullable
	mapIntPtrField(record.StreamVideoWidth, &event.StreamVideoWidth)     // StreamVideoWidth is now nullable
	mapIntPtrField(record.StreamVideoHeight, &event.StreamVideoHeight)   // StreamVideoHeight is now nullable
	mapStringField(record.StreamVideoDynamicRange, &event.StreamVideoDynamicRange)
	mapIntPtrField(record.StreamAudioBitrate, &event.StreamAudioBitrate) // StreamAudioBitrate is now nullable
	mapStringField(record.StreamAudioChannelLayout, &event.StreamAudioChannelLayout)
}

// mapExtendedSubtitleFields adds extended subtitle fields (v1.43 - API Coverage Audit)
func (m *Manager) mapExtendedSubtitleFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapSignedIntPtrField(record.SubtitleForced, &event.SubtitleForced) // SubtitleForced is now nullable
	mapStringField(record.SubtitleLocation, &event.SubtitleLocation)
	mapStringField(record.StreamSubtitleCodec, &event.StreamSubtitleCodec)
	mapStringField(record.StreamSubtitleLanguage, &event.StreamSubtitleLanguage)
	mapStringField(record.SubtitleContainer, &event.SubtitleContainer)
}

// mapExtendedConnectionFields adds extended connection fields (v1.43 - API Coverage Audit)
func (m *Manager) mapExtendedConnectionFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapSignedIntPtrField(record.Relay, &event.Relay)         // Relay is now nullable
	mapSignedIntPtrField(record.Bandwidth, &event.Bandwidth) // Bandwidth is now nullable
	mapStringField(record.Location, &event.Location)
	// Note: BandwidthLAN/BandwidthWAN are server-level settings, not per-history record
}

// mapExtendedFileFields adds extended file metadata fields (v1.43 - API Coverage Audit)
func (m *Manager) mapExtendedFileFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.File, &event.File)
}

// mapThumbnailFields adds thumbnail and art fields (v1.43 - API Coverage Audit)
func (m *Manager) mapThumbnailFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.ParentThumb, &event.ParentThumb)
	mapStringField(record.GrandparentThumb, &event.GrandparentThumb)
	mapStringField(record.Art, &event.Art)
	mapStringField(record.GrandparentArt, &event.GrandparentArt)
}

// mapExtendedGUIDFields adds extended GUID fields for external ID matching (v1.43 - API Coverage Audit)
func (m *Manager) mapExtendedGUIDFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.ParentGuid, &event.ParentGuid)
	mapStringField(record.GrandparentGuid, &event.GrandparentGuid)
}

// enrichEventWithExtendedData enriches the event with all extended fields (v1.43 - API Coverage Audit)
// This method calls all the new mapping functions for comprehensive field coverage
func (m *Manager) enrichEventWithExtendedData(event *models.PlaybackEvent, record *tautulli.TautulliHistoryRecord) {
	m.mapUserFields(record, event)
	m.mapClientDeviceFields(record, event)
	m.mapTranscodeDecisionFields(record, event)
	m.mapHardwareTranscodeFields(record, event)
	m.mapHDRColorMetadataFields(record, event)
	m.mapExtendedVideoFields(record, event)
	m.mapExtendedAudioFields(record, event)
	m.mapExtendedStreamFields(record, event)
	m.mapExtendedSubtitleFields(record, event)
	m.mapExtendedConnectionFields(record, event)
	m.mapExtendedFileFields(record, event)
	m.mapThumbnailFields(record, event)
	m.mapExtendedGUIDFields(record, event)

	// v1.44 API Coverage Expansion
	m.mapUserPermissionFields(record, event)
	m.mapMediaMetadataFields(record, event)
	m.mapLanguageCodeFields(record, event)
	m.mapExtendedStreamV144Fields(record, event)

	// v1.45 API Coverage Expansion
	m.mapUserLibraryAccessFields(record, event)
	m.mapMediaIdentificationV145Fields(record, event)
	m.mapPlaybackStateV145Fields(record, event)
	m.mapLiveTVFields(record, event)

	// v1.46 API Coverage Expansion
	m.mapGroupedPlaybackFields(record, event)
	m.mapUserPermissionV146Fields(record, event)
	m.mapStreamAspectRatioFields(record, event)
	m.mapTranscodeOutputDimensionFields(record, event)
}

// ========================================
// v1.44 API Coverage Expansion Field Mapping
// ========================================

// mapUserPermissionFields adds user permission/status fields (v1.44 - API Coverage Expansion)
func (m *Manager) mapUserPermissionFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapSignedIntPtrField(record.IsAdmin, &event.IsAdmin)           // IsAdmin is now nullable
	mapSignedIntPtrField(record.IsHomeUser, &event.IsHomeUser)     // IsHomeUser is now nullable
	mapSignedIntPtrField(record.IsAllowSync, &event.IsAllowSync)   // IsAllowSync is now nullable
	mapSignedIntPtrField(record.IsRestricted, &event.IsRestricted) // IsRestricted is now nullable
	mapSignedIntPtrField(record.KeepHistory, &event.KeepHistory)   // KeepHistory is now nullable
	mapSignedIntPtrField(record.DeletedUser, &event.DeletedUser)   // DeletedUser is now nullable
	mapSignedIntPtrField(record.DoNotify, &event.DoNotify)         // DoNotify is now nullable
}

// mapMediaMetadataFields adds media metadata fields (v1.44 - API Coverage Expansion)
func (m *Manager) mapMediaMetadataFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.Studio, &event.Studio)
	mapStringField(record.AddedAt, &event.AddedAt)
	mapStringField(record.UpdatedAt, &event.UpdatedAt)
	mapStringField(record.LastViewedAt, &event.LastViewedAt)
	mapStringField(record.Summary, &event.Summary)
	mapStringField(record.Tagline, &event.Tagline)
	mapStringField(record.Rating, &event.Rating)
	mapStringField(record.AudienceRating, &event.AudienceRating)
	mapStringField(record.UserRating, &event.UserRating)
	mapStringField(record.Labels, &event.Labels)
	mapStringField(record.Collections, &event.Collections)
	mapStringField(record.Banner, &event.Banner)
}

// mapLanguageCodeFields adds language code fields (v1.44 - API Coverage Expansion)
func (m *Manager) mapLanguageCodeFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.VideoLanguageCode, &event.VideoLanguageCode)
	mapStringField(record.VideoFullResolution, &event.VideoFullResolution)
	mapStringField(record.AudioLanguageCode, &event.AudioLanguageCode)
	mapStringField(record.SubtitleLanguageCode, &event.SubtitleLanguageCode)
	mapStringField(record.SubtitleContainer, &event.SubtitleContainerFmt)
	mapStringField(record.SubtitleFormat, &event.SubtitleFormat)
}

// mapExtendedStreamV144Fields adds extended stream output fields (v1.44 - API Coverage Expansion)
func (m *Manager) mapExtendedStreamV144Fields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	// Extended stream video fields
	mapStringField(record.StreamVideoCodecLevel, &event.StreamVideoCodecLevel)
	mapStringField(record.StreamVideoFullResolution, &event.StreamVideoFullResolution)
	mapIntPtrField(record.StreamVideoBitDepth, &event.StreamVideoBitDepth) // StreamVideoBitDepth is now nullable
	mapStringField(record.StreamVideoFramerate, &event.StreamVideoFramerate)
	mapStringField(record.StreamVideoProfile, &event.StreamVideoProfile)
	mapStringField(record.StreamVideoScanType, &event.StreamVideoScanType)
	mapStringField(record.StreamVideoLanguage, &event.StreamVideoLanguage)
	mapStringField(record.StreamVideoLanguageCode, &event.StreamVideoLanguageCode)

	// Extended stream audio fields
	mapStringField(record.StreamAudioBitrateMode, &event.StreamAudioBitrateMode)
	mapIntPtrField(record.StreamAudioSampleRate, &event.StreamAudioSampleRate) // StreamAudioSampleRate is now nullable
	mapStringField(record.StreamAudioLanguage, &event.StreamAudioLanguage)
	mapStringField(record.StreamAudioLanguageCode, &event.StreamAudioLanguageCode)
	mapStringField(record.StreamAudioProfile, &event.StreamAudioProfile)

	// Extended stream subtitle fields
	mapStringField(record.StreamSubtitleContainer, &event.StreamSubtitleContainer)
	mapStringField(record.StreamSubtitleFormat, &event.StreamSubtitleFormat)
	mapSignedIntPtrField(record.StreamSubtitleForced, &event.StreamSubtitleForced) // StreamSubtitleForced is now nullable
	mapStringField(record.StreamSubtitleLocation, &event.StreamSubtitleLocation)
	mapStringField(record.StreamSubtitleLanguageCode, &event.StreamSubtitleLanguageCode)
	mapStringField(record.StreamSubtitleDecision, &event.StreamSubtitleDecision)
}

// ========================================
// v1.45 API Coverage Expansion Field Mapping
// ========================================

// mapUserLibraryAccessFields adds user library access fields (v1.45 - API Coverage Expansion)
func (m *Manager) mapUserLibraryAccessFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.SharedLibraries, &event.SharedLibraries)
}

// mapMediaIdentificationV145Fields adds media identification fields (v1.45 - API Coverage Expansion)
func (m *Manager) mapMediaIdentificationV145Fields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.SortTitle, &event.SortTitle)
}

// mapPlaybackStateV145Fields adds playback state fields (v1.45 - API Coverage Expansion)
func (m *Manager) mapPlaybackStateV145Fields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapSignedIntPtrField(record.Throttled, &event.Throttled) // Throttled is now nullable
}

// mapLiveTVFields adds Live TV fields (v1.45 - API Coverage Expansion)
func (m *Manager) mapLiveTVFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapSignedIntPtrField(record.Live, &event.Live) // Live is now nullable
	mapStringField(record.LiveUUID, &event.LiveUUID)
	mapSignedIntPtrField(record.ChannelStream, &event.ChannelStream) // ChannelStream is now nullable
	mapStringField(record.ChannelCallSign, &event.ChannelCallSign)
	mapStringField(record.ChannelIdentifier, &event.ChannelIdentifier)
}

// ========================================
// v1.46 API Coverage Expansion Field Mapping
// ========================================

// mapGroupedPlaybackFields adds grouped playback fields (v1.46 - API Coverage Expansion)
func (m *Manager) mapGroupedPlaybackFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapSignedIntPtrField(record.GroupCount, &event.GroupCount) // GroupCount is now nullable
	mapStringField(record.GroupIDs, &event.GroupIDs)
	mapStringPtrField(record.State, &event.State) // State is now nullable
}

// mapUserPermissionV146Fields adds user permission fields (v1.46 - API Coverage Expansion)
func (m *Manager) mapUserPermissionV146Fields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapSignedIntPtrField(record.AllowGuest, &event.AllowGuest) // AllowGuest is now nullable
}

// mapStreamAspectRatioFields adds stream aspect ratio field (v1.46 - API Coverage Expansion)
func (m *Manager) mapStreamAspectRatioFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapStringField(record.StreamAspectRatio, &event.StreamAspectRatio)
}

// mapTranscodeOutputDimensionFields adds transcode output dimension fields (v1.46 - API Coverage Expansion)
func (m *Manager) mapTranscodeOutputDimensionFields(record *tautulli.TautulliHistoryRecord, event *models.PlaybackEvent) {
	mapSignedIntPtrField(record.TranscodeVideoWidth, &event.TranscodeVideoWidth)   // TranscodeVideoWidth is now nullable
	mapSignedIntPtrField(record.TranscodeVideoHeight, &event.TranscodeVideoHeight) // TranscodeVideoHeight is now nullable
}

// ========================================
// Plex Event Mapping
// ========================================

// plexMetadataFields holds converted Plex metadata fields as pointers
type plexMetadataFields struct {
	ratingKey             *string
	parentRatingKey       *string
	grandparentRatingKey  *string
	mediaIndex            *int
	parentMediaIndex      *int
	guid                  *string
	originalTitle         *string
	originallyAvailableAt *string
	thumb                 *string
	year                  *int
	playDuration          *int
	parentTitle           *string
	grandparentTitle      *string
}

// convertPlexMetadataFields extracts and converts Plex metadata fields to pointers
func convertPlexMetadataFields(record *PlexMetadata) plexMetadataFields {
	var fields plexMetadataFields

	// Convert rating keys
	fields.ratingKey = stringToPtr(record.RatingKey)
	fields.parentRatingKey = stringToPtr(record.ParentRatingKey)
	fields.grandparentRatingKey = stringToPtr(record.GrandparentRatingKey)

	// Convert episode/season numbers
	fields.mediaIndex = intToPtr(record.Index)
	fields.parentMediaIndex = intToPtr(record.ParentIndex)

	// Convert metadata fields
	fields.guid = stringToPtr(record.Guid)
	fields.originalTitle = stringToPtr(record.OriginalTitle)
	fields.originallyAvailableAt = stringToPtr(record.OriginallyAvailableAt)
	fields.thumb = stringToPtr(record.Thumb)
	fields.year = intToPtr(record.Year)

	// Convert duration to seconds (Plex returns milliseconds)
	if record.Duration > 0 {
		durationSec := int(record.Duration / 1000)
		fields.playDuration = &durationSec
	}

	// Convert titles
	fields.parentTitle = stringToPtr(record.ParentTitle)
	fields.grandparentTitle = stringToPtr(record.GrandparentTitle)

	return fields
}

// convertPlexToPlaybackEvent converts Plex API metadata to PlaybackEvent model
//
// Plex history is INCOMPLETE compared to Tautulli - missing critical fields:
//   - ❌ IP addresses (no geolocation possible)
//   - ❌ Platform/Player information (device/app unknown)
//   - ❌ Quality metrics (resolution, codec, transcode decision unknown)
//   - ❌ Stream details (bitrate, audio channels, HDR unknown)
//   - ❌ Connection security flags (secure, relayed, local unknown)
//
// These fields will be NULL in database for Plex-sourced events
func (m *Manager) convertPlexToPlaybackEvent(record *PlexMetadata) *models.PlaybackEvent {
	startedAt := time.Unix(record.ViewedAt, 0)

	// Calculate percent complete (if duration available)
	var percentComplete int
	if record.Duration > 0 {
		pct := float64(record.ViewOffset) / float64(record.Duration) * 100
		percentComplete = int(pct) // Round to integer percentage
	}

	// Calculate stopped_at (started + duration)
	var stoppedAt *time.Time
	if record.Duration > 0 {
		stopped := startedAt.Add(time.Duration(record.Duration) * time.Millisecond)
		stoppedAt = &stopped
	}

	// Extract username from User object
	username := ""
	if record.User != nil {
		username = record.User.Title
	}

	// Convert all metadata fields using helper
	fields := convertPlexMetadataFields(record)

	return &models.PlaybackEvent{
		Source:  "plex", // Mark as Plex source
		PlexKey: fields.ratingKey,

		// SessionKey: Generated by database (UUID)
		// ID: Generated by database (UUID)

		// Basic playback metadata (COMPLETE from Plex)
		RatingKey:            fields.ratingKey,
		ParentRatingKey:      fields.parentRatingKey,
		GrandparentRatingKey: fields.grandparentRatingKey,
		UserID:               record.AccountID,
		Username:             username,
		Title:                record.Title,
		GrandparentTitle:     fields.grandparentTitle,
		ParentTitle:          fields.parentTitle,
		MediaType:            record.Type,
		StartedAt:            startedAt,
		StoppedAt:            stoppedAt,
		PlayDuration:         fields.playDuration,
		PercentComplete:      percentComplete,

		// Episode/season numbering (for binge detection)
		MediaIndex:       fields.mediaIndex,
		ParentMediaIndex: fields.parentMediaIndex,

		// Extended metadata
		Guid:                  fields.guid,
		OriginalTitle:         fields.originalTitle,
		OriginallyAvailableAt: fields.originallyAvailableAt,
		Thumb:                 fields.thumb,
		Year:                  fields.year,

		// Fields MISSING from Plex (will be NULL in database):
		IPAddress:    "", // ❌ No IP address from Plex
		Platform:     "", // ❌ No platform info
		Player:       "", // ❌ No player info
		LocationType: "", // ❌ No location type
		// All quality/stream fields will be NULL
		// All geolocation fields will be NULL
		// All connection security fields will be NULL
	}
}
