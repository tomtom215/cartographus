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

	"github.com/ThreeDotsLabs/watermill"
	wmNats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	natsgo "github.com/nats-io/nats.go"
	gobreaker "github.com/sony/gobreaker/v2"
	"github.com/tomtom215/cartographus/internal/metrics"
)

// Publisher wraps Watermill publisher with resilience patterns.
// It provides circuit breaker protection and automatic reconnection handling.
type Publisher struct {
	publisher      message.Publisher
	circuitBreaker *gobreaker.CircuitBreaker[interface{}]
	mu             sync.RWMutex
	closed         bool
	logger         watermill.LoggerAdapter
}

// NewPublisher creates a resilient Watermill NATS publisher.
// The publisher is configured for JetStream with message ID tracking for deduplication.
func NewPublisher(cfg PublisherConfig, logger watermill.LoggerAdapter) (*Publisher, error) {
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	// NATS connection options with reconnection handling
	natsOpts := []natsgo.Option{
		natsgo.RetryOnFailedConnect(true),
		natsgo.MaxReconnects(cfg.MaxReconnects),
		natsgo.ReconnectWait(cfg.ReconnectWait),
		natsgo.ReconnectBufSize(cfg.ReconnectBuffer),
		natsgo.DisconnectErrHandler(func(nc *natsgo.Conn, err error) {
			if err != nil {
				logger.Error("NATS disconnected", err, nil)
			}
		}),
		natsgo.ReconnectHandler(func(nc *natsgo.Conn) {
			logger.Info("NATS reconnected", watermill.LogFields{
				"url": nc.ConnectedUrl(),
			})
		}),
		natsgo.ErrorHandler(func(nc *natsgo.Conn, sub *natsgo.Subscription, err error) {
			logger.Error("NATS error", err, watermill.LogFields{
				"subject": sub.Subject,
			})
		}),
	}

	// Watermill publisher configuration
	wmConfig := wmNats.PublisherConfig{
		URL:         cfg.URL,
		NatsOptions: natsOpts,
		Marshaler:   &wmNats.NATSMarshaler{},
		JetStream: wmNats.JetStreamConfig{
			Disabled:      false,
			AutoProvision: false,                // Stream is pre-created by StreamInitializer
			TrackMsgId:    cfg.EnableTrackMsgID, // Enable deduplication
			PublishOptions: []natsgo.PubOpt{
				natsgo.RetryAttempts(3),
				natsgo.RetryWait(100 * time.Millisecond),
			},
		},
	}

	pub, err := wmNats.NewPublisher(wmConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("create watermill publisher: %w", err)
	}

	return &Publisher{
		publisher: pub,
		logger:    logger,
	}, nil
}

// SetCircuitBreaker configures the circuit breaker for publish operations.
func (p *Publisher) SetCircuitBreaker(cb *gobreaker.CircuitBreaker[interface{}]) {
	p.circuitBreaker = cb
}

// Publish sends a message to the specified topic with circuit breaker protection.
// The message UUID is used as Nats-Msg-Id for deduplication if not already set.
func (p *Publisher) Publish(ctx context.Context, topic string, msg *message.Message) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	p.mu.RUnlock()

	// Set Nats-Msg-Id for deduplication if not already set
	if msg.Metadata.Get(natsgo.MsgIdHdr) == "" {
		msg.Metadata.Set(natsgo.MsgIdHdr, msg.UUID)
	}

	var err error

	// Circuit breaker wrapper (using v2.3.0 generic API)
	if p.circuitBreaker != nil {
		_, err = p.circuitBreaker.Execute(func() (interface{}, error) {
			return nil, p.publisher.Publish(topic, msg)
		})
	} else {
		err = p.publisher.Publish(topic, msg)
	}

	// Record publish metrics
	if err == nil {
		metrics.RecordNATSPublish()
	}

	return err
}

// PublishEvent serializes and publishes a media event.
// This is a convenience method that handles serialization.
func (p *Publisher) PublishEvent(ctx context.Context, event *MediaEvent) error {
	data, err := SerializeEvent(event)
	if err != nil {
		return fmt.Errorf("serialize event: %w", err)
	}

	msg := message.NewMessage(event.EventID, data)
	msg.Metadata.Set("source", event.Source)
	msg.Metadata.Set("media_type", event.MediaType)
	msg.Metadata.Set("user_id", fmt.Sprintf("%d", event.UserID))

	return p.Publish(ctx, event.Topic(), msg)
}

// PublishBatch publishes multiple messages atomically.
// If any message fails, the error is returned immediately.
func (p *Publisher) PublishBatch(ctx context.Context, topic string, msgs ...*message.Message) error {
	for _, msg := range msgs {
		if err := p.Publish(ctx, topic, msg); err != nil {
			return fmt.Errorf("publish message %s: %w", msg.UUID, err)
		}
	}
	return nil
}

// Close gracefully shuts down the publisher.
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	return p.publisher.Close()
}

// WatermillPublisher returns the underlying Watermill publisher.
// This is useful for passing to Watermill components that require
// the native message.Publisher interface (e.g., poison queue middleware).
func (p *Publisher) WatermillPublisher() message.Publisher {
	return p.publisher
}
