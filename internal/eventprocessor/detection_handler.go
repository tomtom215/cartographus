// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/detection"
)

// DetectionProcessor is the interface that the detection engine implements.
// This abstraction allows the handler to work without importing the full detection package.
type DetectionProcessor interface {
	// Process evaluates an event against all enabled detection rules.
	// Returns alerts if any violations are detected.
	Process(ctx context.Context, event *detection.DetectionEvent) ([]*detection.Alert, error)

	// Enabled returns whether the engine is currently enabled.
	Enabled() bool
}

// DetectionHandler processes media events for security anomaly detection.
// It converts MediaEvent to DetectionEvent and passes them to the detection engine.
//
// This handler is designed to work with the Watermill Router's middleware stack:
//   - Recoverer handles panics
//   - Retry handles transient failures
//   - PoisonQueue routes permanent failures to DLQ
//
// Detection processing is fire-and-forget - we don't want failed detection
// to block event consumption or cause message loss.
type DetectionHandler struct {
	processor DetectionProcessor
	logger    watermill.LoggerAdapter

	// Metrics
	messagesReceived  atomic.Int64
	messagesProcessed atomic.Int64
	detectionErrors   atomic.Int64
	alertsGenerated   atomic.Int64
	skippedDisabled   atomic.Int64
	parseErrors       atomic.Int64
	lastMessageTime   atomic.Value // stores time.Time
}

// DetectionHandlerConfig holds configuration for the detection handler.
type DetectionHandlerConfig struct {
	// ContinueOnError controls whether to ack messages even if detection fails.
	// If true (default), detection errors won't cause message retries.
	ContinueOnError bool
}

// DefaultDetectionHandlerConfig returns production defaults.
func DefaultDetectionHandlerConfig() DetectionHandlerConfig {
	return DetectionHandlerConfig{
		ContinueOnError: true, // Detection failures shouldn't block event flow
	}
}

// NewDetectionHandler creates a new handler for detection processing.
func NewDetectionHandler(processor DetectionProcessor, logger watermill.LoggerAdapter) (*DetectionHandler, error) {
	if processor == nil {
		return nil, ErrNilProcessor
	}
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	h := &DetectionHandler{
		processor: processor,
		logger:    logger,
	}
	h.lastMessageTime.Store(time.Time{})

	return h, nil
}

// Handle processes a single media event message for detection.
// This is the handler function passed to Router.AddNoPublisherHandler.
//
// Error handling:
//   - Parse errors return nil (ack - malformed messages can't be detected)
//   - Detection errors return nil (ack - we don't retry detection)
//   - All valid messages are processed through the detection engine
func (h *DetectionHandler) Handle(msg *message.Message) error {
	startTime := time.Now()
	h.messagesReceived.Add(1)
	h.lastMessageTime.Store(startTime)

	// Skip if detection is disabled
	if !h.processor.Enabled() {
		h.skippedDisabled.Add(1)
		return nil
	}

	// Deserialize event
	var event MediaEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		h.parseErrors.Add(1)
		h.logger.Error("Failed to parse message for detection", err, watermill.LogFields{
			"message_uuid": msg.UUID,
		})
		// Return nil to ack - can't detect on malformed JSON
		return nil
	}

	// Convert MediaEvent to DetectionEvent
	detectionEvent := h.mediaEventToDetectionEvent(&event)

	// Get context from message or create background context
	ctx := context.Background()
	if msgCtx := msg.Context(); msgCtx != nil {
		ctx = msgCtx
	}

	// Process through detection engine
	alerts, err := h.processor.Process(ctx, detectionEvent)
	if err != nil {
		h.detectionErrors.Add(1)
		h.logger.Error("Detection processing error", err, watermill.LogFields{
			"event_id": event.EventID,
			"user_id":  event.UserID,
		})
		// Return nil to ack - don't retry detection failures
		return nil
	}

	// Update metrics
	h.messagesProcessed.Add(1)
	if len(alerts) > 0 {
		h.alertsGenerated.Add(int64(len(alerts)))
		h.logger.Info("Detection alerts generated", watermill.LogFields{
			"event_id": event.EventID,
			"user_id":  event.UserID,
			"alerts":   len(alerts),
		})
	}

	return nil
}

// mediaEventToDetectionEvent converts a MediaEvent to DetectionEvent.
// The DetectionEvent is enriched with user and location information for detection rules.
func (h *DetectionHandler) mediaEventToDetectionEvent(event *MediaEvent) *detection.DetectionEvent {
	return &detection.DetectionEvent{
		// Event identification
		EventID:        event.EventID,
		SessionKey:     event.SessionKey,
		CorrelationKey: event.CorrelationKey,
		EventType:      determineEventType(event),
		Source:         event.Source,
		Timestamp:      event.Timestamp,

		// User information
		UserID:       event.UserID,
		Username:     event.Username,
		FriendlyName: event.FriendlyName,

		// Device information
		MachineID: event.MachineID,
		Platform:  event.Platform,
		Player:    event.Player,
		Device:    event.Device,

		// Media information
		MediaType:        event.MediaType,
		Title:            event.Title,
		GrandparentTitle: event.GrandparentTitle,

		// Network information
		IPAddress:    event.IPAddress,
		LocationType: event.LocationType,

		// Geolocation will be enriched by detection engine from geolocations table
		// The engine's Process method calls EventHistory.GetGeolocation
	}
}

// determineEventType determines the event type from MediaEvent fields.
// Returns "start", "stop", "pause", "resume", or "progress".
func determineEventType(event *MediaEvent) string {
	// If stopped_at is set, this is a stop event
	if event.StoppedAt != nil {
		return EventTypePlaybackStop
	}

	// Check for pause/resume based on paused counter changes
	// For now, default to start
	return EventTypePlaybackStart
}

// Stats returns current handler statistics.
func (h *DetectionHandler) Stats() DetectionHandlerStats {
	var lastTime time.Time
	if t, ok := h.lastMessageTime.Load().(time.Time); ok {
		lastTime = t
	}

	return DetectionHandlerStats{
		MessagesReceived:  h.messagesReceived.Load(),
		MessagesProcessed: h.messagesProcessed.Load(),
		DetectionErrors:   h.detectionErrors.Load(),
		AlertsGenerated:   h.alertsGenerated.Load(),
		SkippedDisabled:   h.skippedDisabled.Load(),
		ParseErrors:       h.parseErrors.Load(),
		LastMessageTime:   lastTime,
	}
}

// DetectionHandlerStats holds runtime statistics.
type DetectionHandlerStats struct {
	MessagesReceived  int64
	MessagesProcessed int64
	DetectionErrors   int64
	AlertsGenerated   int64
	SkippedDisabled   int64
	ParseErrors       int64
	LastMessageTime   time.Time
}

// ErrNilProcessor is returned when the detection processor is nil.
var ErrNilProcessor = NewPermanentError("detection processor required", nil)
