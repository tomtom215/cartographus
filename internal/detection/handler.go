// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package detection

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// WatermillHandler handles playback events from the NATS event stream.
// It integrates with the existing Watermill router to process events
// through the detection engine.
type WatermillHandler struct {
	engine *Engine
}

// NewWatermillHandler creates a new Watermill handler for detection.
func NewWatermillHandler(engine *Engine) *WatermillHandler {
	return &WatermillHandler{engine: engine}
}

// Handle processes a playback event through the detection engine.
// This method implements the Watermill HandlerFunc signature.
func (h *WatermillHandler) Handle(msg *message.Message) ([]*message.Message, error) {
	if h.engine == nil || !h.engine.Enabled() {
		msg.Ack()
		return nil, nil
	}

	// Parse the event from the message payload
	event, err := h.parseEvent(msg)
	if err != nil {
		logging.Warn().Err(err).Msg("failed to parse event")
		msg.Ack() // Ack to prevent reprocessing invalid messages
		return nil, nil
	}

	// Process through detection engine
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	alerts, err := h.engine.Process(ctx, event)
	if err != nil {
		logging.Error().Err(err).Msg("error processing event")
		// Don't return error - we don't want to retry detection failures
	}

	if len(alerts) > 0 {
		logging.Info().Int("count", len(alerts)).Str("username", event.Username).Msg("generated alerts")
	}

	msg.Ack()

	// Optionally publish alerts to a separate topic
	// This allows other systems to react to detection events
	var outputMsgs []*message.Message
	for _, alert := range alerts {
		alertJSON, err := json.Marshal(alert)
		if err != nil {
			logging.Error().Err(err).Msg("failed to marshal alert")
			continue
		}

		outputMsg := message.NewMessage(alert.Title, alertJSON)
		outputMsg.Metadata.Set("rule_type", string(alert.RuleType))
		outputMsg.Metadata.Set("severity", string(alert.Severity))
		outputMsg.Metadata.Set("user_id", fmt.Sprintf("%d", alert.UserID))
		// v2.1: Multi-server support - include server_id in alert metadata
		if alert.ServerID != "" {
			outputMsg.Metadata.Set("server_id", alert.ServerID)
		}
		outputMsgs = append(outputMsgs, outputMsg)
	}

	return outputMsgs, nil
}

// HandleNoPublish processes events without publishing output messages.
// This implements the Watermill NoPublishHandlerFunc signature.
func (h *WatermillHandler) HandleNoPublish(msg *message.Message) error {
	if h.engine == nil || !h.engine.Enabled() {
		msg.Ack()
		return nil
	}

	event, err := h.parseEvent(msg)
	if err != nil {
		logging.Warn().Err(err).Msg("failed to parse event")
		msg.Ack()
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	alerts, err := h.engine.Process(ctx, event)
	if err != nil {
		logging.Error().Err(err).Msg("error processing event")
	}

	if len(alerts) > 0 {
		logging.Info().Int("count", len(alerts)).Str("username", event.Username).Msg("generated alerts")
	}

	msg.Ack()
	return nil
}

// parseEvent converts a Watermill message to a DetectionEvent.
func (h *WatermillHandler) parseEvent(msg *message.Message) (*DetectionEvent, error) {
	// The message payload is JSON from the event processor
	var rawEvent map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &rawEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	event := &DetectionEvent{
		EventID:   msg.UUID,
		Timestamp: time.Now(),
	}

	// Extract fields from the raw event
	// This handles both MediaEvent format and playback_events format
	parseEventIdentification(rawEvent, event)
	parseEventTimestamp(rawEvent, event)
	parseEventUser(rawEvent, event)
	parseEventDevice(rawEvent, event)
	parseEventMedia(rawEvent, event)
	parseEventNetwork(rawEvent, event)
	parseEventGeolocation(rawEvent, event)

	return event, nil
}

// parseEventIdentification extracts event identification fields.
func parseEventIdentification(raw map[string]interface{}, event *DetectionEvent) {
	if v, ok := raw["session_key"].(string); ok {
		event.SessionKey = v
	}
	if v, ok := raw["correlation_key"].(string); ok {
		event.CorrelationKey = v
	}
	if v, ok := raw["event_type"].(string); ok {
		event.EventType = v
	}
	if v, ok := raw["source"].(string); ok {
		event.Source = v
	}
	// v2.1: Multi-server support - extract server_id for server-scoped detection
	if v, ok := raw["server_id"].(string); ok {
		event.ServerID = v
	}
}

// parseEventTimestamp extracts and parses timestamp fields.
func parseEventTimestamp(raw map[string]interface{}, event *DetectionEvent) {
	if v, ok := raw["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			event.Timestamp = t
		}
	}
	if v, ok := raw["started_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			event.Timestamp = t
		}
	}
}

// parseEventUser extracts user information fields.
func parseEventUser(raw map[string]interface{}, event *DetectionEvent) {
	if v, ok := raw["user_id"].(float64); ok {
		event.UserID = int(v)
	}
	if v, ok := raw["username"].(string); ok {
		event.Username = v
	}
	if v, ok := raw["friendly_name"].(string); ok {
		event.FriendlyName = v
	}
}

// parseEventDevice extracts device information fields.
func parseEventDevice(raw map[string]interface{}, event *DetectionEvent) {
	if v, ok := raw["machine_id"].(string); ok {
		event.MachineID = v
	}
	if v, ok := raw["platform"].(string); ok {
		event.Platform = v
	}
	if v, ok := raw["player"].(string); ok {
		event.Player = v
	}
	if v, ok := raw["device"].(string); ok {
		event.Device = v
	}
}

// parseEventMedia extracts media information fields.
func parseEventMedia(raw map[string]interface{}, event *DetectionEvent) {
	if v, ok := raw["media_type"].(string); ok {
		event.MediaType = v
	}
	if v, ok := raw["title"].(string); ok {
		event.Title = v
	}
	if v, ok := raw["grandparent_title"].(string); ok {
		event.GrandparentTitle = v
	}
}

// parseEventNetwork extracts network information fields.
func parseEventNetwork(raw map[string]interface{}, event *DetectionEvent) {
	if v, ok := raw["ip_address"].(string); ok {
		event.IPAddress = v
	}
	if v, ok := raw["location_type"].(string); ok {
		event.LocationType = v
	}
}

// parseEventGeolocation extracts geolocation fields (if already enriched).
func parseEventGeolocation(raw map[string]interface{}, event *DetectionEvent) {
	if v, ok := raw["latitude"].(float64); ok {
		event.Latitude = v
	}
	if v, ok := raw["longitude"].(float64); ok {
		event.Longitude = v
	}
	if v, ok := raw["city"].(string); ok {
		event.City = v
	}
	if v, ok := raw["region"].(string); ok {
		event.Region = v
	}
	if v, ok := raw["country"].(string); ok {
		event.Country = v
	}
}

// Topic returns the recommended NATS topic pattern for detection.
// Format: playback.> (subscribes to all playback events)
func Topic() string {
	return "playback.>"
}

// AlertTopic returns the topic for publishing detection alerts.
func AlertTopic() string {
	return "detection.alerts"
}
