// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StreamManager handles JetStream stream lifecycle.
type StreamManager struct {
	js     jetstream.JetStream
	nc     *nats.Conn
	config StreamConfig
}

// NewStreamManager creates a stream manager with the given config.
func NewStreamManager(nc *nats.Conn, cfg *StreamConfig) (*StreamManager, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	return &StreamManager{
		js:     js,
		nc:     nc,
		config: *cfg,
	}, nil
}

// EnsureStream creates or updates the stream configuration.
func (m *StreamManager) EnsureStream(ctx context.Context) (jetstream.Stream, error) {
	// Single-instance stream for multi-source event ingestion (Plex, Tautulli, Jellyfin).
	// Subjects use hierarchy: events.plex.*, events.tautulli.*, events.jellyfin.*
	streamCfg := jetstream.StreamConfig{
		Name:       m.config.Name,
		Subjects:   m.config.Subjects,
		Retention:  jetstream.LimitsPolicy,
		MaxAge:     m.config.MaxAge,
		MaxBytes:   m.config.MaxBytes,
		MaxMsgs:    m.config.MaxMsgs,
		Duplicates: m.config.DuplicateWindow,
		Replicas:   m.config.Replicas,
		Storage:    jetstream.FileStorage,
		// AllowDirect enables direct get requests, required for DuckDB nats_js extension
		AllowDirect: true,
		// Discard old messages when limits reached
		Discard: jetstream.DiscardOld,
		// Allow message rollup for compaction
		AllowRollup: true,
	}

	// Try to get existing stream
	_, err := m.js.Stream(ctx, m.config.Name)
	if err == nil {
		// Update existing stream
		stream, err := m.js.UpdateStream(ctx, streamCfg)
		if err != nil {
			return nil, fmt.Errorf("update stream: %w", err)
		}
		return stream, nil
	}

	// Create new stream
	stream, err := m.js.CreateStream(ctx, streamCfg)
	if err != nil {
		return nil, fmt.Errorf("create stream: %w", err)
	}

	return stream, nil
}

// GetStreamInfo returns current stream state.
func (m *StreamManager) GetStreamInfo(ctx context.Context) (*jetstream.StreamInfo, error) {
	stream, err := m.js.Stream(ctx, m.config.Name)
	if err != nil {
		return nil, fmt.Errorf("get stream: %w", err)
	}
	return stream.Info(ctx)
}

// PurgeStream removes all messages (use with caution).
func (m *StreamManager) PurgeStream(ctx context.Context) error {
	stream, err := m.js.Stream(ctx, m.config.Name)
	if err != nil {
		return fmt.Errorf("get stream: %w", err)
	}
	return stream.Purge(ctx)
}

// DeleteStream removes the stream entirely.
func (m *StreamManager) DeleteStream(ctx context.Context) error {
	return m.js.DeleteStream(ctx, m.config.Name)
}
