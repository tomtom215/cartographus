// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// FallbackReader uses the native NATS Go client for stream queries.
// This is the fallback when the DuckDB nats_js extension is unavailable.
type FallbackReader struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	mu     sync.RWMutex
	closed bool
}

// NewFallbackReader creates a reader using the NATS Go client.
func NewFallbackReader(natsURL string) (*FallbackReader, error) {
	nc, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	return &FallbackReader{
		nc: nc,
		js: js,
	}, nil
}

// Query retrieves messages using JetStream Direct Get API.
//
//nolint:gocyclo // Query logic with time-based filtering and message iteration is inherently complex
func (r *FallbackReader) Query(ctx context.Context, streamName string, opts *QueryOptions) ([]StreamMessage, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, fmt.Errorf("reader is closed")
	}
	r.mu.RUnlock()

	stream, err := r.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("get stream: %w", err)
	}

	// Get stream info for bounds
	info, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("get stream info: %w", err)
	}

	// Determine sequence range
	startSeq := opts.StartSeq
	if startSeq == 0 {
		startSeq = info.State.FirstSeq
	}

	endSeq := opts.EndSeq
	if endSeq == 0 {
		endSeq = info.State.LastSeq
	}

	// Handle time-based queries with binary search
	if !opts.StartTime.IsZero() {
		if seq := r.findSequenceByTime(ctx, stream, info, opts.StartTime, true); seq > startSeq {
			startSeq = seq
		}
	}
	if !opts.EndTime.IsZero() {
		if seq := r.findSequenceByTime(ctx, stream, info, opts.EndTime, false); seq < endSeq {
			endSeq = seq
		}
	}

	// Fetch messages
	var messages []StreamMessage
	limit := opts.Limit
	if limit <= 0 {
		limit = 10000 // Default safety limit
	}

	for seq := startSeq; seq <= endSeq && len(messages) < limit; seq++ {
		msg, err := stream.GetMsg(ctx, seq)
		if err != nil {
			// Skip deleted or unavailable messages
			continue
		}

		// Subject filter
		if opts.Subject != "" && !matchSubject(msg.Subject, opts.Subject) {
			continue
		}

		messages = append(messages, StreamMessage{
			Sequence:  seq,
			Subject:   msg.Subject,
			Data:      msg.Data,
			Timestamp: msg.Time,
			Headers:   convertHeaders(msg.Header),
		})
	}

	return messages, nil
}

// GetMessage retrieves a single message by sequence using Direct Get.
func (r *FallbackReader) GetMessage(ctx context.Context, streamName string, seq uint64) (*StreamMessage, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, fmt.Errorf("reader is closed")
	}
	r.mu.RUnlock()

	stream, err := r.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("get stream: %w", err)
	}

	msg, err := stream.GetMsg(ctx, seq)
	if err != nil {
		return nil, fmt.Errorf("get message %d: %w", seq, err)
	}

	return &StreamMessage{
		Sequence:  seq,
		Subject:   msg.Subject,
		Data:      msg.Data,
		Timestamp: msg.Time,
		Headers:   convertHeaders(msg.Header),
	}, nil
}

// GetLastSequence returns the latest sequence number.
func (r *FallbackReader) GetLastSequence(ctx context.Context, streamName string) (uint64, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return 0, fmt.Errorf("reader is closed")
	}
	r.mu.RUnlock()

	stream, err := r.js.Stream(ctx, streamName)
	if err != nil {
		return 0, fmt.Errorf("get stream: %w", err)
	}

	info, err := stream.Info(ctx)
	if err != nil {
		return 0, fmt.Errorf("get stream info: %w", err)
	}

	return info.State.LastSeq, nil
}

// Health checks NATS connectivity.
func (r *FallbackReader) Health(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return fmt.Errorf("reader is closed")
	}
	if !r.nc.IsConnected() {
		return fmt.Errorf("not connected to NATS")
	}
	return nil
}

// Close releases the NATS connection.
func (r *FallbackReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	r.nc.Close()
	return nil
}

// findSequenceByTime performs binary search for sequence by timestamp (O(log n)).
// Returns the sequence number closest to the target time.
func (r *FallbackReader) findSequenceByTime(
	ctx context.Context,
	stream jetstream.Stream,
	info *jetstream.StreamInfo,
	targetTime time.Time,
	findFirst bool,
) uint64 {
	low := info.State.FirstSeq
	high := info.State.LastSeq
	result := low

	for low <= high {
		mid := low + (high-low)/2

		msg, err := stream.GetMsg(ctx, mid)
		if err != nil {
			// Skip to next available message
			low = mid + 1
			continue
		}

		if findFirst {
			// Looking for first message >= targetTime
			if msg.Time.Before(targetTime) {
				low = mid + 1
			} else {
				result = mid
				high = mid - 1
			}
		} else {
			// Looking for last message <= targetTime
			if msg.Time.After(targetTime) {
				high = mid - 1
			} else {
				result = mid
				low = mid + 1
			}
		}
	}

	return result
}

func matchSubject(subject, pattern string) bool {
	// Simple wildcard matching for NATS subjects
	if pattern == ">" || pattern == "*" {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '>' {
		prefix := pattern[:len(pattern)-1]
		return len(subject) >= len(prefix) && subject[:len(prefix)] == prefix
	}
	return subject == pattern
}

func convertHeaders(h nats.Header) map[string][]string {
	if h == nil {
		return nil
	}
	result := make(map[string][]string, len(h))
	for k, v := range h {
		result[k] = v
	}
	return result
}
