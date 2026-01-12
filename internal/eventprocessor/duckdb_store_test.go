// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// MockPlaybackInserter implements PlaybackEventInserter for testing.
type MockPlaybackInserter struct {
	mu          sync.Mutex
	events      []*models.PlaybackEvent
	insertErr   error
	insertCalls int
	errorAfterN int // Error after N successful inserts (0 = immediate error if insertErr set)
}

func NewMockPlaybackInserter() *MockPlaybackInserter {
	return &MockPlaybackInserter{
		events: make([]*models.PlaybackEvent, 0),
	}
}

func (m *MockPlaybackInserter) InsertPlaybackEvent(event *models.PlaybackEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.insertCalls++

	// Check if we should error after N calls
	if m.insertErr != nil && (m.errorAfterN == 0 || m.insertCalls > m.errorAfterN) {
		return m.insertErr
	}

	m.events = append(m.events, event)
	return nil
}

func (m *MockPlaybackInserter) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.insertErr = err
	m.errorAfterN = 0
}

// SetErrorAfterN configures the mock to error after N successful inserts.
func (m *MockPlaybackInserter) SetErrorAfterN(n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorAfterN = n
	m.insertErr = err
}

func (m *MockPlaybackInserter) GetEvents() []*models.PlaybackEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*models.PlaybackEvent, len(m.events))
	copy(result, m.events)
	return result
}

func (m *MockPlaybackInserter) GetInsertCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.insertCalls
}

// TestDuckDBStore_NewDuckDBStore verifies store creation.
func TestDuckDBStore_NewDuckDBStore(t *testing.T) {
	tests := []struct {
		name    string
		db      PlaybackEventInserter
		wantErr bool
	}{
		{
			name:    "valid database",
			db:      NewMockPlaybackInserter(),
			wantErr: false,
		},
		{
			name:    "nil database",
			db:      nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewDuckDBStore(tt.db)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDuckDBStore() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && store == nil {
				t.Error("NewDuckDBStore() returned nil store")
			}
		})
	}
}

// TestDuckDBStore_InsertMediaEvents_Single verifies single event insertion.
func TestDuckDBStore_InsertMediaEvents_Single(t *testing.T) {
	db := NewMockPlaybackInserter()
	store, err := NewDuckDBStore(db)
	if err != nil {
		t.Fatalf("NewDuckDBStore() error = %v", err)
	}

	event := NewMediaEvent(SourcePlex)
	event.UserID = 42
	event.Username = "testuser"
	event.MediaType = MediaTypeMovie
	event.Title = "Test Movie"
	event.StartedAt = time.Now()
	event.Platform = "Windows"
	event.Player = "Plex for Windows"
	event.IPAddress = "192.168.1.100"
	event.LocationType = LocationTypeLAN

	ctx := context.Background()
	if err := store.InsertMediaEvents(ctx, []*MediaEvent{event}); err != nil {
		t.Fatalf("InsertMediaEvents() error = %v", err)
	}

	events := db.GetEvents()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	playback := events[0]
	if playback.UserID != 42 {
		t.Errorf("UserID = %d, want 42", playback.UserID)
	}
	if playback.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", playback.Username)
	}
	if playback.MediaType != "movie" {
		t.Errorf("MediaType = %s, want movie", playback.MediaType)
	}
	if playback.Title != "Test Movie" {
		t.Errorf("Title = %s, want Test Movie", playback.Title)
	}
	if playback.Source != SourcePlex {
		t.Errorf("Source = %s, want %s", playback.Source, SourcePlex)
	}
}

// TestDuckDBStore_InsertMediaEvents_Batch verifies batch insertion.
func TestDuckDBStore_InsertMediaEvents_Batch(t *testing.T) {
	db := NewMockPlaybackInserter()
	store, err := NewDuckDBStore(db)
	if err != nil {
		t.Fatalf("NewDuckDBStore() error = %v", err)
	}

	events := make([]*MediaEvent, 10)
	for i := 0; i < 10; i++ {
		event := NewMediaEvent(SourceTautulli)
		event.UserID = i + 1
		event.Username = "user" + string(rune('A'+i))
		event.MediaType = MediaTypeEpisode
		event.Title = "Episode " + string(rune('1'+i))
		event.StartedAt = time.Now()
		event.Platform = "Android"
		event.Player = "Plex"
		event.IPAddress = "10.0.0." + string(rune('1'+i))
		event.LocationType = LocationTypeWAN
		events[i] = event
	}

	ctx := context.Background()
	if err := store.InsertMediaEvents(ctx, events); err != nil {
		t.Fatalf("InsertMediaEvents() error = %v", err)
	}

	if db.GetInsertCalls() != 10 {
		t.Errorf("InsertCalls = %d, want 10", db.GetInsertCalls())
	}

	storedEvents := db.GetEvents()
	if len(storedEvents) != 10 {
		t.Errorf("Stored events = %d, want 10", len(storedEvents))
	}
}

// TestDuckDBStore_InsertMediaEvents_Error verifies error handling.
func TestDuckDBStore_InsertMediaEvents_Error(t *testing.T) {
	db := NewMockPlaybackInserter()
	store, err := NewDuckDBStore(db)
	if err != nil {
		t.Fatalf("NewDuckDBStore() error = %v", err)
	}

	// Set error to occur on insert
	insertErr := errors.New("database connection failed")
	db.SetError(insertErr)

	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.MediaType = MediaTypeMovie
	event.Title = "Test"
	event.StartedAt = time.Now()

	ctx := context.Background()
	err = store.InsertMediaEvents(ctx, []*MediaEvent{event})
	if err == nil {
		t.Fatal("InsertMediaEvents() should return error")
	}
	if !errors.Is(err, insertErr) {
		t.Errorf("Error should wrap original: %v", err)
	}
}

// TestDuckDBStore_InsertMediaEvents_PartialBatchError verifies partial batch error.
func TestDuckDBStore_InsertMediaEvents_PartialBatchError(t *testing.T) {
	db := NewMockPlaybackInserter()
	store, err := NewDuckDBStore(db)
	if err != nil {
		t.Fatalf("NewDuckDBStore() error = %v", err)
	}

	events := make([]*MediaEvent, 5)
	for i := 0; i < 5; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i + 1
		event.MediaType = MediaTypeMovie
		event.Title = "Movie " + string(rune('A'+i))
		event.StartedAt = time.Now()
		events[i] = event
	}

	// Set error after 3 successful inserts
	db.SetErrorAfterN(3, errors.New("connection lost"))

	ctx := context.Background()
	err = store.InsertMediaEvents(ctx, events)
	if err == nil {
		t.Fatal("InsertMediaEvents() should return error on partial failure")
	}

	// First 3 events should have been inserted
	if len(db.GetEvents()) != 3 {
		t.Errorf("Expected 3 events before failure, got %d", len(db.GetEvents()))
	}
}

// TestDuckDBStore_MediaEventConversion verifies complete field mapping.
func TestDuckDBStore_MediaEventConversion(t *testing.T) {
	db := NewMockPlaybackInserter()
	store, err := NewDuckDBStore(db)
	if err != nil {
		t.Fatalf("NewDuckDBStore() error = %v", err)
	}

	now := time.Now()
	stoppedAt := now.Add(2 * time.Hour)

	event := &MediaEvent{
		EventID:           "test-event-123",
		Source:            SourcePlex,
		Timestamp:         now,
		UserID:            100,
		Username:          "moviefan",
		MediaType:         MediaTypeEpisode,
		Title:             "The One Where They Code",
		ParentTitle:       "Season 5",
		GrandparentTitle:  "Friends Reboot",
		RatingKey:         "54321",
		StartedAt:         now,
		StoppedAt:         &stoppedAt,
		PercentComplete:   95,
		PlayDuration:      7200,
		PausedCounter:     3,
		Platform:          "macOS",
		Player:            "Plex Web",
		IPAddress:         "192.168.1.50",
		LocationType:      LocationTypeLAN,
		TranscodeDecision: TranscodeDecisionDirectPlay,
		VideoResolution:   "1080",
		VideoCodec:        "hevc",
		VideoDynamicRange: "HDR10",
		AudioCodec:        "eac3",
		AudioChannels:     6,
		StreamBitrate:     25000,
		Secure:            true,
		Local:             true,
		Relayed:           false,
	}

	ctx := context.Background()
	if err := store.InsertMediaEvents(ctx, []*MediaEvent{event}); err != nil {
		t.Fatalf("InsertMediaEvents() error = %v", err)
	}

	events := db.GetEvents()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	p := events[0]

	// Verify core fields
	if p.Source != SourcePlex {
		t.Errorf("Source = %s, want %s", p.Source, SourcePlex)
	}
	if p.UserID != 100 {
		t.Errorf("UserID = %d, want 100", p.UserID)
	}
	if p.Username != "moviefan" {
		t.Errorf("Username = %s, want moviefan", p.Username)
	}
	if p.MediaType != MediaTypeEpisode {
		t.Errorf("MediaType = %s, want %s", p.MediaType, MediaTypeEpisode)
	}
	if p.Title != "The One Where They Code" {
		t.Errorf("Title = %s, want The One Where They Code", p.Title)
	}

	// Verify optional string fields
	if p.ParentTitle == nil || *p.ParentTitle != "Season 5" {
		t.Errorf("ParentTitle = %v, want Season 5", p.ParentTitle)
	}
	if p.GrandparentTitle == nil || *p.GrandparentTitle != "Friends Reboot" {
		t.Errorf("GrandparentTitle = %v, want Friends Reboot", p.GrandparentTitle)
	}
	if p.RatingKey == nil || *p.RatingKey != "54321" {
		t.Errorf("RatingKey = %v, want 54321", p.RatingKey)
	}
	if p.TranscodeDecision == nil || *p.TranscodeDecision != TranscodeDecisionDirectPlay {
		t.Errorf("TranscodeDecision = %v, want %s", p.TranscodeDecision, TranscodeDecisionDirectPlay)
	}
	if p.VideoResolution == nil || *p.VideoResolution != "1080" {
		t.Errorf("VideoResolution = %v, want 1080", p.VideoResolution)
	}
	if p.VideoCodec == nil || *p.VideoCodec != "hevc" {
		t.Errorf("VideoCodec = %v, want hevc", p.VideoCodec)
	}
	if p.StreamVideoDynamicRange == nil || *p.StreamVideoDynamicRange != "HDR10" {
		t.Errorf("StreamVideoDynamicRange = %v, want HDR10", p.StreamVideoDynamicRange)
	}
	if p.AudioCodec == nil || *p.AudioCodec != "eac3" {
		t.Errorf("AudioCodec = %v, want eac3", p.AudioCodec)
	}

	// Verify optional integer fields
	if p.PlayDuration == nil || *p.PlayDuration != 7200 {
		t.Errorf("PlayDuration = %v, want 7200", p.PlayDuration)
	}
	if p.AudioChannels == nil || *p.AudioChannels != "6" {
		t.Errorf("AudioChannels = %v, want 6", p.AudioChannels)
	}
	if p.StreamBitrate == nil || *p.StreamBitrate != 25000 {
		t.Errorf("StreamBitrate = %v, want 25000", p.StreamBitrate)
	}

	// Verify boolean to int conversion
	if p.Secure == nil || *p.Secure != 1 {
		t.Errorf("Secure = %v, want 1", p.Secure)
	}
	if p.Local == nil || *p.Local != 1 {
		t.Errorf("Local = %v, want 1", p.Local)
	}
	// Relayed is false, so should be nil
	if p.Relayed != nil {
		t.Errorf("Relayed = %v, want nil", p.Relayed)
	}

	// Verify timing
	if p.PercentComplete != 95 {
		t.Errorf("PercentComplete = %d, want 95", p.PercentComplete)
	}
	if p.PausedCounter != 3 {
		t.Errorf("PausedCounter = %d, want 3", p.PausedCounter)
	}
	if p.StoppedAt == nil {
		t.Error("StoppedAt should not be nil")
	}
}

// TestDuckDBStore_EmptyBatch verifies empty batch handling.
func TestDuckDBStore_EmptyBatch(t *testing.T) {
	db := NewMockPlaybackInserter()
	store, err := NewDuckDBStore(db)
	if err != nil {
		t.Fatalf("NewDuckDBStore() error = %v", err)
	}

	ctx := context.Background()
	if err := store.InsertMediaEvents(ctx, []*MediaEvent{}); err != nil {
		t.Errorf("InsertMediaEvents() with empty batch should not error: %v", err)
	}

	if db.GetInsertCalls() != 0 {
		t.Errorf("InsertCalls = %d, want 0 for empty batch", db.GetInsertCalls())
	}
}

// BenchmarkDuckDBStore_InsertMediaEvents benchmarks batch insertion.
func BenchmarkDuckDBStore_InsertMediaEvents(b *testing.B) {
	db := NewMockPlaybackInserter()
	store, err := NewDuckDBStore(db)
	if err != nil {
		b.Fatalf("NewDuckDBStore() error = %v", err)
	}

	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.MediaType = MediaTypeMovie
	event.Title = "Benchmark Movie"
	event.StartedAt = time.Now()

	events := []*MediaEvent{event}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.InsertMediaEvents(ctx, events)
	}
}
