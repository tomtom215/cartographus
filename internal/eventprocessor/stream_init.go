// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

// JetStreamContext defines the subset of jetstream.JetStream used by StreamInitializer.
// This interface allows for testing with mock implementations.
type JetStreamContext interface {
	Stream(ctx context.Context, name string) (jetstream.Stream, error)
	CreateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error)
	UpdateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error)
	DeleteStream(ctx context.Context, name string) error
}

// StreamInitializer handles JetStream stream lifecycle management.
// It ensures streams are created with the correct configuration before
// publishers and subscribers start, guaranteeing reliable message delivery.
//
// Key responsibilities:
//   - Create stream if it doesn't exist
//   - Update stream configuration if it already exists
//   - Provide health check for stream availability
//   - Handle idempotent stream initialization
type StreamInitializer struct {
	js     JetStreamContext
	config StreamConfig
}

// NewStreamInitializer creates a new stream initializer with the given configuration.
// Returns an error if the JetStream context is nil or config is nil.
func NewStreamInitializer(js JetStreamContext, cfg *StreamConfig) (*StreamInitializer, error) {
	if js == nil {
		return nil, fmt.Errorf("JetStream context required")
	}
	if cfg == nil {
		return nil, fmt.Errorf("stream config required")
	}

	return &StreamInitializer{
		js:     js,
		config: *cfg,
	}, nil
}

// EnsureStream creates or updates the stream with the configured settings.
// This operation is idempotent - calling it multiple times is safe.
//
// The stream is configured with:
//   - File storage for durability
//   - LimitsPolicy retention (FIFO when limits reached)
//   - AllowDirect=true for efficient direct get operations
//   - Configurable deduplication window
//
// Returns the stream handle or an error if creation/update fails.
func (s *StreamInitializer) EnsureStream(ctx context.Context) (jetstream.Stream, error) {
	streamCfg := jetstream.StreamConfig{
		Name:        s.config.Name,
		Subjects:    s.config.Subjects,
		Retention:   jetstream.LimitsPolicy,
		MaxAge:      s.config.MaxAge,
		MaxBytes:    s.config.MaxBytes,
		MaxMsgs:     s.config.MaxMsgs,
		Duplicates:  s.config.DuplicateWindow,
		Replicas:    s.config.Replicas,
		Storage:     jetstream.FileStorage,
		AllowDirect: true, // Required for efficient direct get
		// Note: MirrorDirect is only valid for mirror streams, not primary streams
		Discard:     jetstream.DiscardOld,
		AllowRollup: true, // Allow message rollup for compaction
	}

	// Try to get existing stream
	_, err := s.js.Stream(ctx, s.config.Name)
	if err == nil {
		// Stream exists, update configuration
		stream, err := s.js.UpdateStream(ctx, streamCfg)
		if err != nil {
			return nil, fmt.Errorf("update stream %s: %w", s.config.Name, err)
		}
		return stream, nil
	}

	// Stream doesn't exist, create it
	if errors.Is(err, jetstream.ErrStreamNotFound) {
		stream, err := s.js.CreateStream(ctx, streamCfg)
		if err != nil {
			return nil, fmt.Errorf("create stream %s: %w", s.config.Name, err)
		}
		return stream, nil
	}

	// Unexpected error checking stream existence
	return nil, fmt.Errorf("check stream %s: %w", s.config.Name, err)
}

// GetStreamInfo retrieves current stream state and configuration.
// Returns an error if the stream doesn't exist.
func (s *StreamInitializer) GetStreamInfo(ctx context.Context) (*jetstream.StreamInfo, error) {
	stream, err := s.js.Stream(ctx, s.config.Name)
	if err != nil {
		return nil, fmt.Errorf("get stream %s: %w", s.config.Name, err)
	}
	return stream.Info(ctx)
}

// IsHealthy checks if the stream exists and is accessible.
// Returns true if the stream can be queried, false otherwise.
func (s *StreamInitializer) IsHealthy(ctx context.Context) bool {
	_, err := s.js.Stream(ctx, s.config.Name)
	return err == nil
}

// Config returns the current stream configuration.
func (s *StreamInitializer) Config() StreamConfig {
	return s.config
}
