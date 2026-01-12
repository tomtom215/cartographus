// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestSyncEventPublisher_NewSyncEventPublisher verifies publisher creation.
func TestSyncEventPublisher_NewSyncEventPublisher(t *testing.T) {
	tests := []struct {
		name    string
		pub     *Publisher
		wantErr bool
	}{
		{
			name:    "nil publisher",
			pub:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSyncEventPublisher(tt.pub)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSyncEventPublisher() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSyncEventPublisher_PlaybackEventConversion verifies field mapping.
func TestSyncEventPublisher_PlaybackEventConversion(t *testing.T) {
	// Create a SyncEventPublisher with mock (can't use real Publisher without NATS)
	// We'll test the conversion method directly by making it accessible
	pub := &SyncEventPublisher{}

	now := time.Now()
	stoppedAt := now.Add(2 * time.Hour)
	id := uuid.New()

	// Build a comprehensive PlaybackEvent
	parentTitle := "Season 5"
	grandparentTitle := "Friends"
	ratingKey := "12345"
	transcodeDecision := "direct play"
	videoResolution := "1080"
	videoCodec := "hevc"
	dynamicRange := "HDR10"
	audioCodec := "eac3"
	playDuration := 7200
	audioChannels := "6"
	streamBitrate := 25000
	secure := 1
	local := 1

	event := &models.PlaybackEvent{
		ID:                      id,
		Source:                  "plex",
		SessionKey:              "test-session",
		StartedAt:               now,
		StoppedAt:               &stoppedAt,
		UserID:                  42,
		Username:                "testuser",
		MediaType:               "movie",
		Title:                   "Test Movie",
		ParentTitle:             &parentTitle,
		GrandparentTitle:        &grandparentTitle,
		RatingKey:               &ratingKey,
		TranscodeDecision:       &transcodeDecision,
		VideoResolution:         &videoResolution,
		VideoCodec:              &videoCodec,
		StreamVideoDynamicRange: &dynamicRange,
		AudioCodec:              &audioCodec,
		PlayDuration:            &playDuration,
		AudioChannels:           &audioChannels,
		StreamBitrate:           &streamBitrate,
		PercentComplete:         95,
		PausedCounter:           3,
		Platform:                "Windows",
		Player:                  "Plex for Windows",
		IPAddress:               "192.168.1.100",
		LocationType:            "lan",
		Secure:                  &secure,
		Local:                   &local,
	}

	mediaEvent := pub.playbackEventToMediaEvent(event)

	// Verify core fields
	if mediaEvent.EventID != id.String() {
		t.Errorf("EventID = %s, want %s", mediaEvent.EventID, id.String())
	}
	if mediaEvent.Source != "plex" {
		t.Errorf("Source = %s, want plex", mediaEvent.Source)
	}
	if mediaEvent.UserID != 42 {
		t.Errorf("UserID = %d, want 42", mediaEvent.UserID)
	}
	if mediaEvent.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", mediaEvent.Username)
	}
	if mediaEvent.MediaType != "movie" {
		t.Errorf("MediaType = %s, want movie", mediaEvent.MediaType)
	}
	if mediaEvent.Title != "Test Movie" {
		t.Errorf("Title = %s, want Test Movie", mediaEvent.Title)
	}

	// Verify optional string fields
	if mediaEvent.ParentTitle != "Season 5" {
		t.Errorf("ParentTitle = %s, want Season 5", mediaEvent.ParentTitle)
	}
	if mediaEvent.GrandparentTitle != "Friends" {
		t.Errorf("GrandparentTitle = %s, want Friends", mediaEvent.GrandparentTitle)
	}
	if mediaEvent.RatingKey != "12345" {
		t.Errorf("RatingKey = %s, want 12345", mediaEvent.RatingKey)
	}
	if mediaEvent.TranscodeDecision != "direct play" {
		t.Errorf("TranscodeDecision = %s, want direct play", mediaEvent.TranscodeDecision)
	}
	if mediaEvent.VideoResolution != "1080" {
		t.Errorf("VideoResolution = %s, want 1080", mediaEvent.VideoResolution)
	}
	if mediaEvent.VideoCodec != "hevc" {
		t.Errorf("VideoCodec = %s, want hevc", mediaEvent.VideoCodec)
	}
	if mediaEvent.VideoDynamicRange != "HDR10" {
		t.Errorf("VideoDynamicRange = %s, want HDR10", mediaEvent.VideoDynamicRange)
	}
	if mediaEvent.AudioCodec != "eac3" {
		t.Errorf("AudioCodec = %s, want eac3", mediaEvent.AudioCodec)
	}

	// Verify optional integer fields
	if mediaEvent.PlayDuration != 7200 {
		t.Errorf("PlayDuration = %d, want 7200", mediaEvent.PlayDuration)
	}
	if mediaEvent.AudioChannels != 6 {
		t.Errorf("AudioChannels = %d, want 6", mediaEvent.AudioChannels)
	}
	if mediaEvent.StreamBitrate != 25000 {
		t.Errorf("StreamBitrate = %d, want 25000", mediaEvent.StreamBitrate)
	}

	// Verify playback metrics
	if mediaEvent.PercentComplete != 95 {
		t.Errorf("PercentComplete = %d, want 95", mediaEvent.PercentComplete)
	}
	if mediaEvent.PausedCounter != 3 {
		t.Errorf("PausedCounter = %d, want 3", mediaEvent.PausedCounter)
	}

	// Verify platform fields
	if mediaEvent.Platform != "Windows" {
		t.Errorf("Platform = %s, want Windows", mediaEvent.Platform)
	}
	if mediaEvent.Player != "Plex for Windows" {
		t.Errorf("Player = %s, want Plex for Windows", mediaEvent.Player)
	}
	if mediaEvent.IPAddress != "192.168.1.100" {
		t.Errorf("IPAddress = %s, want 192.168.1.100", mediaEvent.IPAddress)
	}
	if mediaEvent.LocationType != "lan" {
		t.Errorf("LocationType = %s, want lan", mediaEvent.LocationType)
	}

	// Verify boolean fields
	if !mediaEvent.Secure {
		t.Error("Secure should be true")
	}
	if !mediaEvent.Local {
		t.Error("Local should be true")
	}
	if mediaEvent.Relayed {
		t.Error("Relayed should be false")
	}

	// Verify timing
	if mediaEvent.StoppedAt == nil {
		t.Error("StoppedAt should not be nil")
	}
}

// TestSyncEventPublisher_NilEvent verifies nil event handling.
func TestSyncEventPublisher_NilEvent(t *testing.T) {
	pub := &SyncEventPublisher{}

	ctx := context.Background()
	err := pub.PublishPlaybackEvent(ctx, nil)
	if err != nil {
		t.Errorf("PublishPlaybackEvent(nil) should return nil, got %v", err)
	}
}

// TestSyncEventPublisher_MinimalEvent verifies minimal event conversion.
func TestSyncEventPublisher_MinimalEvent(t *testing.T) {
	pub := &SyncEventPublisher{}

	event := &models.PlaybackEvent{
		ID:        uuid.New(),
		Source:    "tautulli",
		UserID:    1,
		Username:  "user",
		MediaType: "movie",
		Title:     "Title",
		Platform:  "iOS",
		Player:    "Plex",
		StartedAt: time.Now(),
	}

	mediaEvent := pub.playbackEventToMediaEvent(event)

	// Verify required fields are set
	if mediaEvent.EventID == "" {
		t.Error("EventID should not be empty")
	}
	if mediaEvent.Source != "tautulli" {
		t.Errorf("Source = %s, want tautulli", mediaEvent.Source)
	}

	// Verify optional fields are empty/zero when not set
	if mediaEvent.ParentTitle != "" {
		t.Errorf("ParentTitle = %s, want empty", mediaEvent.ParentTitle)
	}
	if mediaEvent.PlayDuration != 0 {
		t.Errorf("PlayDuration = %d, want 0", mediaEvent.PlayDuration)
	}
	if mediaEvent.Secure {
		t.Error("Secure should be false when not set")
	}
}

// TestSyncEventPublisher_AllOptionalUserFields verifies user optional field mapping.
func TestSyncEventPublisher_AllOptionalUserFields(t *testing.T) {
	pub := &SyncEventPublisher{}

	friendlyName := "John's TV"
	userThumb := "https://plex.tv/thumb/123"
	email := "john@example.com"

	event := &models.PlaybackEvent{
		ID:           uuid.New(),
		Source:       "plex",
		UserID:       1,
		Username:     "john",
		FriendlyName: &friendlyName,
		UserThumb:    &userThumb,
		Email:        &email,
		MediaType:    "movie",
		Title:        "Test",
		Platform:     "Web",
		Player:       "Plex",
		StartedAt:    time.Now(),
	}

	mediaEvent := pub.playbackEventToMediaEvent(event)

	if mediaEvent.FriendlyName != friendlyName {
		t.Errorf("FriendlyName = %s, want %s", mediaEvent.FriendlyName, friendlyName)
	}
	if mediaEvent.UserThumb != userThumb {
		t.Errorf("UserThumb = %s, want %s", mediaEvent.UserThumb, userThumb)
	}
	if mediaEvent.Email != email {
		t.Errorf("Email = %s, want %s", mediaEvent.Email, email)
	}
}

// TestSyncEventPublisher_PlatformOptionalFields verifies platform optional field mapping.
func TestSyncEventPublisher_PlatformOptionalFields(t *testing.T) {
	pub := &SyncEventPublisher{}

	platformName := "iOS"
	platformVersion := "17.0"
	product := "Plex"
	productVersion := "8.30.0"
	device := "iPhone 15 Pro"
	machineID := "abc123def456"

	event := &models.PlaybackEvent{
		ID:              uuid.New(),
		Source:          "plex",
		UserID:          1,
		Username:        "user",
		MediaType:       "movie",
		Title:           "Test",
		Platform:        "iOS",
		Player:          "Plex",
		PlatformName:    &platformName,
		PlatformVersion: &platformVersion,
		Product:         &product,
		ProductVersion:  &productVersion,
		Device:          &device,
		MachineID:       &machineID,
		StartedAt:       time.Now(),
	}

	mediaEvent := pub.playbackEventToMediaEvent(event)

	if mediaEvent.PlatformName != platformName {
		t.Errorf("PlatformName = %s, want %s", mediaEvent.PlatformName, platformName)
	}
	if mediaEvent.PlatformVersion != platformVersion {
		t.Errorf("PlatformVersion = %s, want %s", mediaEvent.PlatformVersion, platformVersion)
	}
	if mediaEvent.Product != product {
		t.Errorf("Product = %s, want %s", mediaEvent.Product, product)
	}
	if mediaEvent.ProductVersion != productVersion {
		t.Errorf("ProductVersion = %s, want %s", mediaEvent.ProductVersion, productVersion)
	}
	if mediaEvent.Device != device {
		t.Errorf("Device = %s, want %s", mediaEvent.Device, device)
	}
	if mediaEvent.MachineID != machineID {
		t.Errorf("MachineID = %s, want %s", mediaEvent.MachineID, machineID)
	}
}

// TestSyncEventPublisher_IntegerOptionalFields verifies integer optional field mapping.
func TestSyncEventPublisher_IntegerOptionalFields(t *testing.T) {
	pub := &SyncEventPublisher{}

	year := 2024
	bandwidth := 100000

	event := &models.PlaybackEvent{
		ID:        uuid.New(),
		Source:    "plex",
		UserID:    1,
		Username:  "user",
		MediaType: "movie",
		Title:     "Test",
		Platform:  "Web",
		Player:    "Plex",
		Year:      &year,
		Bandwidth: &bandwidth,
		StartedAt: time.Now(),
	}

	mediaEvent := pub.playbackEventToMediaEvent(event)

	if mediaEvent.Year != year {
		t.Errorf("Year = %d, want %d", mediaEvent.Year, year)
	}
	if mediaEvent.Bandwidth != bandwidth {
		t.Errorf("Bandwidth = %d, want %d", mediaEvent.Bandwidth, bandwidth)
	}
}

// TestSyncEventPublisher_BooleanFieldsFalse verifies boolean field mapping when 0.
func TestSyncEventPublisher_BooleanFieldsFalse(t *testing.T) {
	pub := &SyncEventPublisher{}

	secure := 0
	local := 0
	relayed := 0

	event := &models.PlaybackEvent{
		ID:        uuid.New(),
		Source:    "plex",
		UserID:    1,
		Username:  "user",
		MediaType: "movie",
		Title:     "Test",
		Platform:  "Web",
		Player:    "Plex",
		Secure:    &secure,
		Local:     &local,
		Relayed:   &relayed,
		StartedAt: time.Now(),
	}

	mediaEvent := pub.playbackEventToMediaEvent(event)

	if mediaEvent.Secure {
		t.Error("Secure should be false when value is 0")
	}
	if mediaEvent.Local {
		t.Error("Local should be false when value is 0")
	}
	if mediaEvent.Relayed {
		t.Error("Relayed should be false when value is 0")
	}
}

// TestSyncEventPublisher_AudioChannelsConversion verifies string to int conversion for audio channels.
func TestSyncEventPublisher_AudioChannelsConversion(t *testing.T) {
	pub := &SyncEventPublisher{}

	tests := []struct {
		name          string
		audioChannels string
		expected      int
	}{
		{"stereo", "2", 2},
		{"5.1", "6", 6},
		{"7.1", "8", 8},
		{"atmos", "16", 16},
		{"invalid", "not-a-number", 0}, // Should default to 0 on error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channels := tt.audioChannels
			event := &models.PlaybackEvent{
				ID:            uuid.New(),
				Source:        "plex",
				UserID:        1,
				Username:      "user",
				MediaType:     "movie",
				Title:         "Test",
				Platform:      "Web",
				Player:        "Plex",
				AudioChannels: &channels,
				StartedAt:     time.Now(),
			}

			mediaEvent := pub.playbackEventToMediaEvent(event)

			if mediaEvent.AudioChannels != tt.expected {
				t.Errorf("AudioChannels = %d, want %d for input %s", mediaEvent.AudioChannels, tt.expected, tt.audioChannels)
			}
		})
	}
}

// TestSyncEventPublisher_RelayedTrue verifies Relayed boolean field mapping when 1.
func TestSyncEventPublisher_RelayedTrue(t *testing.T) {
	pub := &SyncEventPublisher{}

	relayed := 1

	event := &models.PlaybackEvent{
		ID:        uuid.New(),
		Source:    "plex",
		UserID:    1,
		Username:  "user",
		MediaType: "movie",
		Title:     "Test",
		Platform:  "Web",
		Player:    "Plex",
		Relayed:   &relayed,
		StartedAt: time.Now(),
	}

	mediaEvent := pub.playbackEventToMediaEvent(event)

	if !mediaEvent.Relayed {
		t.Error("Relayed should be true when value is 1")
	}
}
