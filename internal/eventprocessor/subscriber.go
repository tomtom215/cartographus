// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	wmNats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	natsgo "github.com/nats-io/nats.go"
)

// Subscriber wraps Watermill subscriber with configuration.
// It provides durable JetStream consumption with exactly-once semantics.
type Subscriber struct {
	subscriber message.Subscriber
	config     SubscriberConfig
	logger     watermill.LoggerAdapter
}

// NewSubscriber creates a durable JetStream subscriber.
// The subscriber is configured for queue-based load balancing across multiple instances.
func NewSubscriber(cfg *SubscriberConfig, logger watermill.LoggerAdapter) (*Subscriber, error) {
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	natsOpts := []natsgo.Option{
		natsgo.RetryOnFailedConnect(true),
		natsgo.MaxReconnects(cfg.MaxReconnects),
		natsgo.ReconnectWait(cfg.ReconnectWait),
		natsgo.DisconnectErrHandler(func(nc *natsgo.Conn, err error) {
			if err != nil {
				logger.Error("Subscriber disconnected", err, nil)
			}
		}),
		natsgo.ReconnectHandler(func(nc *natsgo.Conn) {
			logger.Info("Subscriber reconnected", watermill.LogFields{
				"url": nc.ConnectedUrl(),
			})
		}),
	}

	// JetStream consumer options
	subOpts := []natsgo.SubOpt{
		natsgo.MaxDeliver(cfg.MaxDeliver),
		natsgo.MaxAckPending(cfg.MaxAckPending),
		natsgo.AckWait(cfg.AckWaitTimeout),
		// Deliver new messages only (use DeliverAll for replay)
		natsgo.DeliverNew(),
	}

	// When StreamName is configured, bind to the existing stream.
	// This is required for wildcard topics (e.g., "playback.>") because
	// NATS stream names cannot contain wildcards, and AutoProvision would
	// fail trying to create a stream named after the wildcard topic.
	autoProvision := true
	if cfg.StreamName != "" {
		subOpts = append(subOpts, natsgo.BindStream(cfg.StreamName))
		autoProvision = false
	}

	wmConfig := wmNats.SubscriberConfig{
		URL:              cfg.URL,
		QueueGroupPrefix: cfg.QueueGroup,
		SubscribersCount: cfg.SubscribersCount,
		AckWaitTimeout:   cfg.AckWaitTimeout,
		CloseTimeout:     cfg.CloseTimeout,
		NatsOptions:      natsOpts,
		Unmarshaler:      &wmNats.NATSMarshaler{},
		JetStream: wmNats.JetStreamConfig{
			Disabled:         false,
			AutoProvision:    autoProvision,
			AckAsync:         false, // Synchronous for exactly-once
			SubscribeOptions: subOpts,
			DurablePrefix:    cfg.DurableName,
		},
	}

	sub, err := wmNats.NewSubscriber(wmConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("create watermill subscriber: %w", err)
	}

	return &Subscriber{
		subscriber: sub,
		config:     *cfg,
		logger:     logger,
	}, nil
}

// Subscribe returns a channel of messages for the given topic.
// The channel is closed when the context is canceled or the subscriber is closed.
func (s *Subscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return s.subscriber.Subscribe(ctx, topic)
}

// Close gracefully shuts down the subscriber.
func (s *Subscriber) Close() error {
	return s.subscriber.Close()
}

// MessageHandler provides a fluent API for message processing.
type MessageHandler struct {
	subscriber *Subscriber
	topic      string
	handler    func(ctx context.Context, msg *message.Message) error
	logger     watermill.LoggerAdapter
}

// NewMessageHandler creates a handler for processing messages from the given topic.
func (s *Subscriber) NewMessageHandler(topic string) *MessageHandler {
	return &MessageHandler{
		subscriber: s,
		topic:      topic,
		logger:     s.logger,
	}
}

// Handle sets the message processing function.
// The function should return an error if processing fails (message will be nacked).
func (h *MessageHandler) Handle(fn func(ctx context.Context, msg *message.Message) error) *MessageHandler {
	h.handler = fn
	return h
}

// Run starts processing messages until context cancellation.
// Messages are acked on successful processing, nacked on error.
func (h *MessageHandler) Run(ctx context.Context) error {
	messages, err := h.subscriber.Subscribe(ctx, h.topic)
	if err != nil {
		return fmt.Errorf("subscribe to %s: %w", h.topic, err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-messages:
			if !ok {
				return nil
			}
			if err := h.processMessage(ctx, msg); err != nil {
				h.logger.Error("Message processing failed", err, watermill.LogFields{
					"message_uuid": msg.UUID,
					"topic":        h.topic,
				})
			}
		}
	}
}

func (h *MessageHandler) processMessage(ctx context.Context, msg *message.Message) error {
	if h.handler == nil {
		msg.Ack()
		return nil
	}

	if err := h.handler(ctx, msg); err != nil {
		msg.Nack()
		return err
	}

	msg.Ack()
	return nil
}

// EventHandler is a convenience type for handling MediaEvent messages.
type EventHandler struct {
	handler    *MessageHandler
	serializer *Serializer
}

// NewEventHandler creates a handler that automatically deserializes events.
func (s *Subscriber) NewEventHandler(topic string) *EventHandler {
	return &EventHandler{
		handler:    s.NewMessageHandler(topic),
		serializer: NewSerializer(),
	}
}

// Handle sets the event processing function.
func (h *EventHandler) Handle(fn func(ctx context.Context, event *MediaEvent) error) *EventHandler {
	h.handler.Handle(func(ctx context.Context, msg *message.Message) error {
		event, err := h.serializer.Unmarshal(msg.Payload)
		if err != nil {
			return fmt.Errorf("unmarshal event: %w", err)
		}
		return fn(ctx, event)
	})
	return h
}

// Run starts processing events until context cancellation.
func (h *EventHandler) Run(ctx context.Context) error {
	return h.handler.Run(ctx)
}
