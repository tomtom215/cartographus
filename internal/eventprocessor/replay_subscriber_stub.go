// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

// Package eventprocessor provides stub implementations for non-NATS builds.
package eventprocessor

import (
	"context"
	"errors"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ReplayMode specifies how to determine the starting point for replay.
type ReplayMode int

const (
	ReplayModeNew ReplayMode = iota
	ReplayModeAll
	ReplayModeSequence
	ReplayModeTime
	ReplayModeLastAcked
)

// String returns the string representation of the replay mode.
func (m ReplayMode) String() string {
	return "disabled"
}

// ReplayConfig defines replay-specific configuration.
type ReplayConfig struct {
	Mode                ReplayMode
	StartSequence       uint64
	StartTime           time.Time
	StopSequence        uint64
	StopTime            time.Time
	BatchSize           int
	CheckpointInterval  time.Duration
	VerifyTransactionID bool
	DryRun              bool
}

// DefaultReplayConfig returns sensible defaults for replay operations.
func DefaultReplayConfig() ReplayConfig {
	return ReplayConfig{}
}

// ReplayStats tracks replay progress.
type ReplayStats struct {
	StartTime        time.Time
	EndTime          time.Time
	TotalMessages    int64
	ProcessedCount   int64
	SkippedCount     int64
	ErrorCount       int64
	LastSequence     uint64
	LastTimestamp    time.Time
	CurrentSequence  uint64
	EstimatedRemain  int64
	BytesProcessed   int64
	ProcessingRate   float64
	Status           string
	LastError        string
	LastCheckpointAt time.Time
}

// ReplaySubscriber is a stub for non-NATS builds.
type ReplaySubscriber struct{}

// NewReplaySubscriber returns an error for non-NATS builds.
func NewReplaySubscriber(
	_ *SubscriberConfig,
	_ *ReplayConfig,
	_ *CheckpointStore,
	_ watermill.LoggerAdapter,
) (*ReplaySubscriber, error) {
	return nil, errors.New("replay subscriber requires NATS build tag")
}

// Subscribe is a stub.
func (s *ReplaySubscriber) Subscribe(_ context.Context, _ string) (<-chan *message.Message, error) {
	return nil, errors.New("replay subscriber requires NATS build tag")
}

// Close is a stub.
func (s *ReplaySubscriber) Close() error {
	return nil
}

// Stats returns nil for stub.
func (s *ReplaySubscriber) Stats() *ReplayStats {
	return nil
}

// RunReplay is a stub.
func (s *ReplaySubscriber) RunReplay(
	_ context.Context,
	_ string,
	_ func(context.Context, *MediaEvent) error,
) error {
	return errors.New("replay subscriber requires NATS build tag")
}
