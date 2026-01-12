// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats && integration

package eventprocessor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestIntegration_FullPipeline tests the complete event flow:
// Publisher -> Appender -> DuckDB Store
//
// This test verifies that events flow correctly through all components.
// It uses mocks for the actual NATS infrastructure but tests the integration
// between all the eventprocessor components.
func TestIntegration_FullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup: Create mock store
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     5,
		FlushInterval: 100 * time.Millisecond,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}
	defer appender.Close()

	// Start appender timer
	ctx := context.Background()
	if err := appender.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Test: Send multiple events through appender
	const numEvents = 12 // Should trigger 2 batch flushes + 2 remaining

	for i := 0; i < numEvents; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i + 1
		event.Username = "user" + string(rune('A'+i%26))
		event.MediaType = MediaTypeMovie
		event.Title = "Integration Test Movie"
		event.StartedAt = time.Now()

		if err := appender.Append(ctx, event); err != nil {
			t.Errorf("Append() event %d error = %v", i, err)
		}
	}

	// Wait for batch flushes
	time.Sleep(300 * time.Millisecond)

	// Verify: Check that events were flushed
	events := store.GetEvents()
	stats := appender.Stats()

	// Should have at least 2 batch flushes (10 events) plus timer flush for remaining
	if len(events) < 10 {
		t.Errorf("Store events = %d, want >= 10", len(events))
	}

	if stats.EventsReceived != int64(numEvents) {
		t.Errorf("Stats.EventsReceived = %d, want %d", stats.EventsReceived, numEvents)
	}

	if stats.FlushCount < 2 {
		t.Errorf("Stats.FlushCount = %d, want >= 2", stats.FlushCount)
	}
}

// TestIntegration_AppenderWithDuckDBStore tests Appender + DuckDBStore integration.
func TestIntegration_AppenderWithDuckDBStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup: Create mock database and DuckDB store
	mockDB := NewMockPlaybackInserter()
	duckDBStore, err := NewDuckDBStore(mockDB)
	if err != nil {
		t.Fatalf("NewDuckDBStore() error = %v", err)
	}

	cfg := AppenderConfig{
		BatchSize:     3,
		FlushInterval: time.Hour, // Won't trigger
	}

	appender, err := NewAppender(duckDBStore, cfg)
	if err != nil {
		t.Fatalf("NewAppender() error = %v", err)
	}
	defer appender.Close()

	ctx := context.Background()

	// Test: Send events that trigger batch flush
	for i := 0; i < 3; i++ {
		event := NewMediaEvent(SourceTautulli)
		event.UserID = i + 100
		event.Username = "integrationuser"
		event.MediaType = MediaTypeEpisode
		event.Title = "Episode " + string(rune('1'+i))
		event.StartedAt = time.Now()
		event.Platform = "iOS"
		event.Player = "Plex"

		if err := appender.Append(ctx, event); err != nil {
			t.Errorf("Append() error = %v", err)
		}
	}

	// Wait for async batch flush
	time.Sleep(100 * time.Millisecond)

	// Verify: Check PlaybackEvents were inserted
	playbacks := mockDB.GetEvents()
	if len(playbacks) != 3 {
		t.Fatalf("Expected 3 playback events, got %d", len(playbacks))
	}

	// Verify field mapping through the pipeline
	for i, p := range playbacks {
		if p.Source != SourceTautulli {
			t.Errorf("Event %d: Source = %s, want %s", i, p.Source, SourceTautulli)
		}
		if p.MediaType != MediaTypeEpisode {
			t.Errorf("Event %d: MediaType = %s, want %s", i, p.MediaType, MediaTypeEpisode)
		}
		if p.Username != "integrationuser" {
			t.Errorf("Event %d: Username = %s, want integrationuser", i, p.Username)
		}
	}
}

// TestIntegration_SyncPublisherConversion tests PlaybackEvent -> MediaEvent conversion.
func TestIntegration_SyncPublisherConversion(t *testing.T) {
	// This tests the roundtrip conversion:
	// PlaybackEvent -> MediaEvent -> PlaybackEvent
	// Ensures data is preserved through the conversion.

	pub := &SyncEventPublisher{}

	// Create a comprehensive PlaybackEvent
	parentTitle := "Season 3"
	grandparentTitle := "Breaking Bad"
	ratingKey := "98765"
	transcodeDecision := "transcode"
	videoResolution := "4k"
	videoCodec := "h265"
	dynamicRange := "Dolby Vision"
	audioCodec := "truehd"
	playDuration := 3600
	audioChannels := 8
	streamBitrate := 50000
	secure := 1
	local := 0

	original := &models.PlaybackEvent{
		ID:                      uuid.New(),
		Source:                  SourcePlex,
		SessionKey:              "session-123",
		StartedAt:               time.Now(),
		UserID:                  1001,
		Username:                "heisenberg",
		MediaType:               MediaTypeEpisode,
		Title:                   "Ozymandias",
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
		PercentComplete:         100,
		PausedCounter:           0,
		Platform:                "Apple TV",
		Player:                  "Plex",
		IPAddress:               "10.0.0.50",
		LocationType:            LocationTypeLAN,
		Secure:                  &secure,
		Local:                   &local,
	}

	// Convert to MediaEvent
	mediaEvent := pub.playbackEventToMediaEvent(original)

	// Convert back to PlaybackEvent using DuckDBStore
	store := &DuckDBStore{}
	restored := store.mediaEventToPlaybackEvent(mediaEvent)

	// Verify roundtrip preservation
	if restored.Source != original.Source {
		t.Errorf("Source: got %s, want %s", restored.Source, original.Source)
	}
	if restored.UserID != original.UserID {
		t.Errorf("UserID: got %d, want %d", restored.UserID, original.UserID)
	}
	if restored.Username != original.Username {
		t.Errorf("Username: got %s, want %s", restored.Username, original.Username)
	}
	if restored.MediaType != original.MediaType {
		t.Errorf("MediaType: got %s, want %s", restored.MediaType, original.MediaType)
	}
	if restored.Title != original.Title {
		t.Errorf("Title: got %s, want %s", restored.Title, original.Title)
	}
	if restored.PercentComplete != original.PercentComplete {
		t.Errorf("PercentComplete: got %d, want %d", restored.PercentComplete, original.PercentComplete)
	}
	if restored.Platform != original.Platform {
		t.Errorf("Platform: got %s, want %s", restored.Platform, original.Platform)
	}
	if restored.IPAddress != original.IPAddress {
		t.Errorf("IPAddress: got %s, want %s", restored.IPAddress, original.IPAddress)
	}

	// Verify optional string fields
	if restored.ParentTitle == nil || *restored.ParentTitle != *original.ParentTitle {
		t.Errorf("ParentTitle: got %v, want %v", restored.ParentTitle, original.ParentTitle)
	}
	if restored.GrandparentTitle == nil || *restored.GrandparentTitle != *original.GrandparentTitle {
		t.Errorf("GrandparentTitle: got %v, want %v", restored.GrandparentTitle, original.GrandparentTitle)
	}
	if restored.TranscodeDecision == nil || *restored.TranscodeDecision != *original.TranscodeDecision {
		t.Errorf("TranscodeDecision: got %v, want %v", restored.TranscodeDecision, original.TranscodeDecision)
	}

	// Verify optional integer fields
	if restored.PlayDuration == nil || *restored.PlayDuration != *original.PlayDuration {
		t.Errorf("PlayDuration: got %v, want %v", restored.PlayDuration, original.PlayDuration)
	}
	if restored.StreamBitrate == nil || *restored.StreamBitrate != *original.StreamBitrate {
		t.Errorf("StreamBitrate: got %v, want %v", restored.StreamBitrate, original.StreamBitrate)
	}

	// Verify boolean conversion (both directions)
	if restored.Secure == nil || *restored.Secure != *original.Secure {
		t.Errorf("Secure: got %v, want %v", restored.Secure, original.Secure)
	}
	// Local was 0, so should be nil after roundtrip (false -> nil)
	if restored.Local != nil && *restored.Local != 0 {
		t.Errorf("Local: got %v, want nil or 0", restored.Local)
	}
}

// TestIntegration_ConcurrentAppenders tests multiple concurrent appenders.
func TestIntegration_ConcurrentAppenders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     10,
		FlushInterval: time.Hour,
	}

	const numAppenders = 5
	const eventsPerAppender = 20

	var wg sync.WaitGroup
	appenders := make([]*Appender, numAppenders)

	// Create multiple appenders sharing the same store
	for i := 0; i < numAppenders; i++ {
		appender, err := NewAppender(store, cfg)
		if err != nil {
			t.Fatalf("NewAppender() %d error = %v", i, err)
		}
		appenders[i] = appender
	}

	ctx := context.Background()

	// Run concurrent appends
	wg.Add(numAppenders)
	for i := 0; i < numAppenders; i++ {
		go func(appenderID int) {
			defer wg.Done()
			appender := appenders[appenderID]

			for j := 0; j < eventsPerAppender; j++ {
				event := NewMediaEvent(SourcePlex)
				event.UserID = appenderID*1000 + j
				event.MediaType = MediaTypeMovie
				event.Title = "Concurrent Test"
				event.StartedAt = time.Now()

				if err := appender.Append(ctx, event); err != nil {
					t.Errorf("Appender %d: Append() error = %v", appenderID, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Close all appenders to flush remaining events
	for i, appender := range appenders {
		if err := appender.Close(); err != nil {
			t.Errorf("Appender %d: Close() error = %v", i, err)
		}
	}

	// Verify all events were stored
	totalExpected := numAppenders * eventsPerAppender
	events := store.GetEvents()
	if len(events) != totalExpected {
		t.Errorf("Store events = %d, want %d", len(events), totalExpected)
	}

	// Verify uniqueness by checking user IDs
	userIDs := make(map[int]bool)
	for _, e := range events {
		userIDs[e.UserID] = true
	}
	if len(userIDs) != totalExpected {
		t.Errorf("Unique user IDs = %d, want %d", len(userIDs), totalExpected)
	}
}

// BenchmarkIntegration_Pipeline benchmarks the full pipeline throughput.
func BenchmarkIntegration_Pipeline(b *testing.B) {
	store := NewMockEventStore()
	cfg := AppenderConfig{
		BatchSize:     1000,
		FlushInterval: time.Second,
	}

	appender, err := NewAppender(store, cfg)
	if err != nil {
		b.Fatalf("NewAppender() error = %v", err)
	}
	defer appender.Close()

	ctx := context.Background()
	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.MediaType = MediaTypeMovie
	event.Title = "Benchmark"
	event.StartedAt = time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = appender.Append(ctx, event)
	}
	b.StopTimer()

	// Ensure all events are flushed
	_ = appender.Close()
}
