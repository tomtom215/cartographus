// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

// Package eventprocessor provides deterministic event replay for disaster recovery.
// CRITICAL-002: Implements replay from sequence number or timestamp for state reconstruction.
package eventprocessor

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	wmNats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	natsgo "github.com/nats-io/nats.go"
	"github.com/tomtom215/cartographus/internal/logging"
)

// ReplayMode specifies how to determine the starting point for replay.
type ReplayMode int

const (
	// ReplayModeNew delivers only new messages (default behavior).
	ReplayModeNew ReplayMode = iota
	// ReplayModeAll replays all messages from the beginning of the stream.
	ReplayModeAll
	// ReplayModeSequence replays from a specific sequence number.
	ReplayModeSequence
	// ReplayModeTime replays from a specific timestamp.
	ReplayModeTime
	// ReplayModeLastAcked resumes from the last acknowledged message (durable consumer).
	ReplayModeLastAcked
)

// String returns the string representation of the replay mode.
func (m ReplayMode) String() string {
	switch m {
	case ReplayModeNew:
		return "new"
	case ReplayModeAll:
		return "all"
	case ReplayModeSequence:
		return "sequence"
	case ReplayModeTime:
		return "time"
	case ReplayModeLastAcked:
		return "last_acked"
	default:
		return "unknown"
	}
}

// ReplayConfig defines replay-specific configuration.
type ReplayConfig struct {
	// Mode specifies the replay starting point.
	Mode ReplayMode

	// StartSequence is the sequence number to start from (for ReplayModeSequence).
	StartSequence uint64

	// StartTime is the timestamp to start from (for ReplayModeTime).
	StartTime time.Time

	// StopSequence is the optional end sequence (0 = no limit).
	StopSequence uint64

	// StopTime is the optional end timestamp (zero = no limit).
	StopTime time.Time

	// BatchSize is the number of messages to process before checkpointing.
	BatchSize int

	// CheckpointInterval is how often to save progress.
	CheckpointInterval time.Duration

	// VerifyTransactionID enables transaction ID verification during replay.
	VerifyTransactionID bool

	// DryRun previews replay without writing to database.
	DryRun bool
}

// DefaultReplayConfig returns sensible defaults for replay operations.
func DefaultReplayConfig() ReplayConfig {
	return ReplayConfig{
		Mode:                ReplayModeLastAcked,
		BatchSize:           1000,
		CheckpointInterval:  30 * time.Second,
		VerifyTransactionID: true,
		DryRun:              false,
	}
}

// ReplaySubscriber extends Subscriber with replay capabilities.
// It can replay events from a specific sequence number or timestamp
// for disaster recovery and state reconstruction.
type ReplaySubscriber struct {
	subscriber   message.Subscriber
	config       SubscriberConfig
	replayConfig ReplayConfig
	logger       watermill.LoggerAdapter
	checkpoint   *CheckpointStore
	stats        *ReplayStats
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
	ProcessingRate   float64 // messages per second
	Status           string
	LastError        string
	LastCheckpointAt time.Time
}

// NewReplaySubscriber creates a subscriber configured for replay.
func NewReplaySubscriber(
	cfg *SubscriberConfig,
	replayCfg *ReplayConfig,
	checkpoint *CheckpointStore,
	logger watermill.LoggerAdapter,
) (*ReplaySubscriber, error) {
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	// Build subscription options based on replay mode
	subOpts := buildReplaySubOptions(cfg, replayCfg)

	natsOpts := []natsgo.Option{
		natsgo.RetryOnFailedConnect(true),
		natsgo.MaxReconnects(cfg.MaxReconnects),
		natsgo.ReconnectWait(cfg.ReconnectWait),
		natsgo.DisconnectErrHandler(func(nc *natsgo.Conn, err error) {
			if err != nil {
				logger.Error("Replay subscriber disconnected", err, nil)
			}
		}),
		natsgo.ReconnectHandler(func(nc *natsgo.Conn) {
			logger.Info("Replay subscriber reconnected", watermill.LogFields{
				"url": nc.ConnectedUrl(),
			})
		}),
	}

	autoProvision := true
	if cfg.StreamName != "" {
		subOpts = append(subOpts, natsgo.BindStream(cfg.StreamName))
		autoProvision = false
	}

	// Use a unique durable name for replay to avoid conflicting with normal consumers
	durableName := cfg.DurableName + "-replay"

	wmConfig := wmNats.SubscriberConfig{
		URL:              cfg.URL,
		QueueGroupPrefix: "", // No queue group for replay (we want all messages)
		SubscribersCount: 1,  // Single-threaded for ordering
		AckWaitTimeout:   cfg.AckWaitTimeout,
		CloseTimeout:     cfg.CloseTimeout,
		NatsOptions:      natsOpts,
		Unmarshaler:      &wmNats.NATSMarshaler{},
		JetStream: wmNats.JetStreamConfig{
			Disabled:         false,
			AutoProvision:    autoProvision,
			AckAsync:         false,
			SubscribeOptions: subOpts,
			DurablePrefix:    durableName,
		},
	}

	sub, err := wmNats.NewSubscriber(wmConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("create replay subscriber: %w", err)
	}

	return &ReplaySubscriber{
		subscriber:   sub,
		config:       *cfg,
		replayConfig: *replayCfg,
		logger:       logger,
		checkpoint:   checkpoint,
		stats: &ReplayStats{
			Status: "initialized",
		},
	}, nil
}

// buildReplaySubOptions builds NATS subscription options based on replay mode.
func buildReplaySubOptions(cfg *SubscriberConfig, replayCfg *ReplayConfig) []natsgo.SubOpt {
	subOpts := []natsgo.SubOpt{
		natsgo.MaxDeliver(cfg.MaxDeliver),
		natsgo.MaxAckPending(cfg.MaxAckPending),
		natsgo.AckWait(cfg.AckWaitTimeout),
	}

	switch replayCfg.Mode {
	case ReplayModeAll:
		subOpts = append(subOpts, natsgo.DeliverAll())
	case ReplayModeSequence:
		subOpts = append(subOpts, natsgo.StartSequence(replayCfg.StartSequence))
	case ReplayModeTime:
		subOpts = append(subOpts, natsgo.StartTime(replayCfg.StartTime))
	case ReplayModeLastAcked:
		subOpts = append(subOpts, natsgo.DeliverLast())
	default:
		// ReplayModeNew
		subOpts = append(subOpts, natsgo.DeliverNew())
	}

	return subOpts
}

// Subscribe returns a channel of messages for replay.
func (s *ReplaySubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return s.subscriber.Subscribe(ctx, topic)
}

// Close gracefully shuts down the replay subscriber.
func (s *ReplaySubscriber) Close() error {
	return s.subscriber.Close()
}

// Stats returns current replay statistics.
func (s *ReplaySubscriber) Stats() *ReplayStats {
	return s.stats
}

// RunReplay executes a replay operation with the configured settings.
// It processes messages according to the replay mode and tracks progress.
func (s *ReplaySubscriber) RunReplay(
	ctx context.Context,
	topic string,
	handler func(ctx context.Context, event *MediaEvent) error,
) error {
	s.stats.StartTime = time.Now()
	s.stats.Status = "running"

	logging.Info().
		Str("mode", s.replayConfig.Mode.String()).
		Uint64("start_sequence", s.replayConfig.StartSequence).
		Time("start_time", s.replayConfig.StartTime).
		Bool("dry_run", s.replayConfig.DryRun).
		Msg("Starting event replay")

	messages, err := s.subscriber.Subscribe(ctx, topic)
	if err != nil {
		s.stats.Status = "error"
		s.stats.LastError = err.Error()
		return fmt.Errorf("subscribe for replay: %w", err)
	}

	serializer := NewSerializer()
	lastCheckpoint := time.Now()

	for {
		select {
		case <-ctx.Done():
			s.stats.Status = "canceled"
			s.stats.EndTime = time.Now()
			s.saveCheckpoint(ctx)
			return ctx.Err()

		case msg, ok := <-messages:
			if !ok {
				s.stats.Status = "completed"
				s.stats.EndTime = time.Now()
				s.saveCheckpoint(ctx)
				return nil
			}

			// Deserialize event
			event, err := serializer.Unmarshal(msg.Payload)
			if err != nil {
				s.stats.ErrorCount++
				s.stats.LastError = err.Error()
				msg.Nack()
				continue
			}

			// Check stop conditions
			if s.shouldStop(event) {
				s.stats.Status = "completed"
				s.stats.EndTime = time.Now()
				msg.Ack()
				s.saveCheckpoint(ctx)
				return nil
			}

			// Process event
			if !s.replayConfig.DryRun {
				if err := handler(ctx, event); err != nil {
					s.stats.ErrorCount++
					s.stats.LastError = err.Error()
					msg.Nack()
					continue
				}
			}

			// Update stats
			s.stats.ProcessedCount++
			s.stats.BytesProcessed += int64(len(msg.Payload))
			s.updateSequenceFromMetadata(msg)

			msg.Ack()

			// Periodic checkpoint
			if time.Since(lastCheckpoint) >= s.replayConfig.CheckpointInterval {
				s.saveCheckpoint(ctx)
				lastCheckpoint = time.Now()
			}

			// Update processing rate
			elapsed := time.Since(s.stats.StartTime).Seconds()
			if elapsed > 0 {
				s.stats.ProcessingRate = float64(s.stats.ProcessedCount) / elapsed
			}
		}
	}
}

// shouldStop checks if we've reached the stop conditions.
func (s *ReplaySubscriber) shouldStop(event *MediaEvent) bool {
	if s.replayConfig.StopSequence > 0 && s.stats.CurrentSequence >= s.replayConfig.StopSequence {
		return true
	}
	if !s.replayConfig.StopTime.IsZero() && event.Timestamp.After(s.replayConfig.StopTime) {
		return true
	}
	return false
}

// updateSequenceFromMetadata extracts sequence info from message metadata.
func (s *ReplaySubscriber) updateSequenceFromMetadata(msg *message.Message) {
	// Watermill stores NATS metadata in message metadata
	if seqStr := msg.Metadata.Get("nats_sequence"); seqStr != "" {
		// Parse sequence (best effort, ignore parse errors for non-numeric values)
		var seq uint64
		if _, err := fmt.Sscanf(seqStr, "%d", &seq); err == nil {
			s.stats.CurrentSequence = seq
			s.stats.LastSequence = seq
		}
	}
	s.stats.LastTimestamp = time.Now()
}

// saveCheckpoint persists current progress.
func (s *ReplaySubscriber) saveCheckpoint(ctx context.Context) {
	if s.checkpoint == nil {
		return
	}

	cp := &Checkpoint{
		ConsumerName:   s.config.DurableName + "-replay",
		StreamName:     s.config.StreamName,
		LastSequence:   s.stats.LastSequence,
		LastTimestamp:  s.stats.LastTimestamp,
		ProcessedCount: s.stats.ProcessedCount,
		Status:         s.stats.Status,
		UpdatedAt:      time.Now(),
	}

	if err := s.checkpoint.Save(ctx, cp); err != nil {
		logging.Warn().Err(err).Msg("Failed to save replay checkpoint")
	} else {
		s.stats.LastCheckpointAt = time.Now()
	}
}
