// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package websocket

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// MediaEvent mirrors eventprocessor.MediaEvent to avoid circular imports.
// This is the same structure as defined in internal/eventprocessor/events.go.
type MediaEvent struct {
	// Identification
	EventID    string    `json:"event_id"`
	SessionKey string    `json:"session_key,omitempty"`
	Source     string    `json:"source"`
	Timestamp  time.Time `json:"timestamp"`

	// User information
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	FriendlyName string `json:"friendly_name,omitempty"`
	UserThumb    string `json:"user_thumb,omitempty"`
	Email        string `json:"email,omitempty"`

	// Media identification
	MediaType        string `json:"media_type"`
	Title            string `json:"title"`
	ParentTitle      string `json:"parent_title,omitempty"`
	GrandparentTitle string `json:"grandparent_title,omitempty"`
	RatingKey        string `json:"rating_key,omitempty"`
	Year             int    `json:"year,omitempty"`
	MediaDuration    int    `json:"media_duration,omitempty"`

	// Playback timing
	StartedAt       time.Time  `json:"started_at"`
	StoppedAt       *time.Time `json:"stopped_at,omitempty"`
	PercentComplete int        `json:"percent_complete,omitempty"`
	PlayDuration    int        `json:"play_duration,omitempty"`
	PausedCounter   int        `json:"paused_counter,omitempty"`

	// Platform information
	Platform        string `json:"platform,omitempty"`
	PlatformName    string `json:"platform_name,omitempty"`
	PlatformVersion string `json:"platform_version,omitempty"`
	Player          string `json:"player,omitempty"`
	Product         string `json:"product,omitempty"`
	ProductVersion  string `json:"product_version,omitempty"`
	Device          string `json:"device,omitempty"`
	MachineID       string `json:"machine_id,omitempty"`
	IPAddress       string `json:"ip_address,omitempty"`
	LocationType    string `json:"location_type,omitempty"`

	// Streaming quality
	TranscodeDecision string `json:"transcode_decision,omitempty"`
	VideoResolution   string `json:"video_resolution,omitempty"`
	VideoCodec        string `json:"video_codec,omitempty"`
	VideoDynamicRange string `json:"video_dynamic_range,omitempty"`
	AudioCodec        string `json:"audio_codec,omitempty"`
	AudioChannels     int    `json:"audio_channels,omitempty"`
	StreamBitrate     int    `json:"stream_bitrate,omitempty"`
	Bandwidth         int    `json:"bandwidth,omitempty"`

	// Connection details
	Secure  bool `json:"secure,omitempty"`
	Local   bool `json:"local,omitempty"`
	Relayed bool `json:"relayed,omitempty"`
}

// NATSMessageHandler defines the interface for receiving NATS messages.
// This allows the WebSocket subscriber to work with any message source.
type NATSMessageHandler interface {
	// Subscribe subscribes to a topic and returns a channel of messages.
	Subscribe(ctx context.Context, topic string) (<-chan []byte, error)
	// Close releases resources.
	Close() error
}

// NATSSubscriber bridges NATS events to WebSocket broadcasts.
// It subscribes to NATS topics and forwards events to the WebSocket hub.
type NATSSubscriber struct {
	hub     *Hub
	handler NATSMessageHandler
	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// NewNATSSubscriber creates a new NATS to WebSocket bridge.
func NewNATSSubscriber(hub *Hub, handler NATSMessageHandler) *NATSSubscriber {
	return &NATSSubscriber{
		hub:     hub,
		handler: handler,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
}

// Start begins listening for NATS events and forwarding to WebSocket.
// Subscribes to the "playback.>" wildcard to receive all playback events.
func (s *NATSSubscriber) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	// Subscribe to all playback events
	messages, err := s.handler.Subscribe(ctx, "playback.>")
	if err != nil {
		return err
	}

	go s.processMessages(ctx, messages)

	logging.Info().Msg("NATS to WebSocket subscriber started")
	return nil
}

// Stop stops the subscriber.
func (s *NATSSubscriber) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	<-s.doneCh
	logging.Info().Msg("NATS to WebSocket subscriber stopped")
}

// processMessages handles incoming NATS messages.
func (s *NATSSubscriber) processMessages(ctx context.Context, messages <-chan []byte) {
	defer close(s.doneCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case data, ok := <-messages:
			if !ok {
				return
			}
			s.handleMessage(data)
		}
	}
}

// handleMessage processes a single NATS message.
func (s *NATSSubscriber) handleMessage(data []byte) {
	var event MediaEvent
	if err := json.Unmarshal(data, &event); err != nil {
		logging.Warn().Err(err).Msg("failed to unmarshal NATS event")
		return
	}

	// Convert to PlaybackEvent and broadcast
	playback := s.mediaEventToPlaybackEvent(&event)
	s.hub.BroadcastNewPlayback(playback)
}

// mediaEventToPlaybackEvent converts a MediaEvent to a PlaybackEvent.
//
//nolint:gocyclo // Data mapping function with many optional fields requires conditional checks
func (s *NATSSubscriber) mediaEventToPlaybackEvent(event *MediaEvent) *models.PlaybackEvent {
	// Use SessionKey if available, fallback to EventID
	sessionKey := event.SessionKey
	if sessionKey == "" {
		sessionKey = event.EventID
	}

	playback := &models.PlaybackEvent{
		ID:              parseOrGenerateUUID(event.EventID),
		Source:          event.Source,
		SessionKey:      sessionKey,
		StartedAt:       event.StartedAt,
		StoppedAt:       event.StoppedAt,
		UserID:          event.UserID,
		Username:        event.Username,
		MediaType:       event.MediaType,
		Title:           event.Title,
		Platform:        event.Platform,
		Player:          event.Player,
		IPAddress:       event.IPAddress,
		LocationType:    event.LocationType,
		PercentComplete: event.PercentComplete,
		PausedCounter:   event.PausedCounter,
		CreatedAt:       time.Now(),
	}

	// User optional string fields
	if event.FriendlyName != "" {
		playback.FriendlyName = &event.FriendlyName
	}
	if event.UserThumb != "" {
		playback.UserThumb = &event.UserThumb
	}
	if event.Email != "" {
		playback.Email = &event.Email
	}

	// Media optional string fields
	if event.ParentTitle != "" {
		playback.ParentTitle = &event.ParentTitle
	}
	if event.GrandparentTitle != "" {
		playback.GrandparentTitle = &event.GrandparentTitle
	}
	if event.RatingKey != "" {
		playback.RatingKey = &event.RatingKey
	}

	// Media optional integer fields
	if event.Year > 0 {
		playback.Year = &event.Year
	}
	// Note: MediaDuration not mapped - PlaybackEvent doesn't have media duration field

	// Platform optional fields
	if event.PlatformName != "" {
		playback.PlatformName = &event.PlatformName
	}
	if event.PlatformVersion != "" {
		playback.PlatformVersion = &event.PlatformVersion
	}
	if event.Product != "" {
		playback.Product = &event.Product
	}
	if event.ProductVersion != "" {
		playback.ProductVersion = &event.ProductVersion
	}
	if event.Device != "" {
		playback.Device = &event.Device
	}
	if event.MachineID != "" {
		playback.MachineID = &event.MachineID
	}

	// Streaming quality optional fields
	if event.TranscodeDecision != "" {
		playback.TranscodeDecision = &event.TranscodeDecision
	}
	if event.VideoResolution != "" {
		playback.VideoResolution = &event.VideoResolution
	}
	if event.VideoCodec != "" {
		playback.VideoCodec = &event.VideoCodec
	}
	if event.VideoDynamicRange != "" {
		playback.StreamVideoDynamicRange = &event.VideoDynamicRange
	}
	if event.AudioCodec != "" {
		playback.AudioCodec = &event.AudioCodec
	}

	// Optional integer fields
	if event.PlayDuration > 0 {
		playback.PlayDuration = &event.PlayDuration
	}
	if event.AudioChannels > 0 {
		audioChannels := strconv.Itoa(event.AudioChannels)
		playback.AudioChannels = &audioChannels
	}
	if event.StreamBitrate > 0 {
		playback.StreamBitrate = &event.StreamBitrate
	}
	if event.Bandwidth > 0 {
		playback.Bandwidth = &event.Bandwidth
	}

	// Boolean to int conversions
	if event.Secure {
		secure := 1
		playback.Secure = &secure
	}
	if event.Local {
		local := 1
		playback.Local = &local
	}
	if event.Relayed {
		relayed := 1
		playback.Relayed = &relayed
	}

	return playback
}

// parseOrGenerateUUID attempts to parse the string as a UUID.
func parseOrGenerateUUID(s string) uuid.UUID {
	if id, err := uuid.Parse(s); err == nil {
		return id
	}
	return uuid.New()
}
