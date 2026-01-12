// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package eventprocessor

import (
	"fmt"

	"github.com/goccy/go-json"
)

// Serializer handles event encoding/decoding for NATS messages.
type Serializer struct{}

// NewSerializer creates a new serializer.
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Marshal converts an event to JSON bytes.
func (s *Serializer) Marshal(event *MediaEvent) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("validate event: %w", err)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}

	return data, nil
}

// Unmarshal converts JSON bytes to an event.
func (s *Serializer) Unmarshal(data []byte) (*MediaEvent, error) {
	var event MediaEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}

	return &event, nil
}

// SerializeEvent is a convenience function that marshals an event to JSON.
func SerializeEvent(event *MediaEvent) ([]byte, error) {
	return NewSerializer().Marshal(event)
}

// DeserializeEvent is a convenience function that unmarshals JSON to an event.
func DeserializeEvent(data []byte) (*MediaEvent, error) {
	return NewSerializer().Unmarshal(data)
}
