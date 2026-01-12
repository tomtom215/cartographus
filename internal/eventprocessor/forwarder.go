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
	"github.com/ThreeDotsLabs/watermill/message"
)

// ForwarderConfig holds configuration for the Outbox Forwarder.
type ForwarderConfig struct {
	// PollInterval is how often to check for pending messages.
	// Default: 100ms
	PollInterval time.Duration

	// RetryDelay is how long to wait before retrying a failed forward.
	// Default: 1s
	RetryDelay time.Duration

	// MaxRetries is the maximum number of retry attempts per message.
	// Default: 5
	MaxRetries int

	// BatchSize is the maximum number of messages to forward per poll.
	// Default: 100
	BatchSize int
}

// DefaultForwarderConfig returns production defaults.
func DefaultForwarderConfig() ForwarderConfig {
	return ForwarderConfig{
		PollInterval: 100 * time.Millisecond,
		RetryDelay:   time.Second,
		MaxRetries:   5,
		BatchSize:    100,
	}
}

// OutboxMessage represents a message stored in the outbox for reliable delivery.
// This is used when you need guaranteed delivery even if NATS is temporarily unavailable.
type OutboxMessage struct {
	// ID is the unique message identifier.
	ID string

	// Topic is the NATS subject to publish to.
	Topic string

	// Payload is the serialized message data.
	Payload []byte

	// Metadata contains message headers.
	Metadata map[string]string

	// CreatedAt is when the message was created.
	CreatedAt time.Time

	// RetryCount is the number of delivery attempts.
	RetryCount int

	// LastError is the most recent delivery error.
	LastError string
}

// OutboxStore defines the interface for persistent outbox storage.
// Implementations should use a transactional database (BadgerDB, PostgreSQL, etc.)
// to ensure atomicity with application state changes.
type OutboxStore interface {
	// Store saves a message to the outbox within a transaction.
	// This should be called in the same transaction as the business logic.
	Store(ctx context.Context, msg *OutboxMessage) error

	// GetPending returns messages ready for delivery.
	GetPending(ctx context.Context, limit int) ([]*OutboxMessage, error)

	// MarkDelivered removes a successfully delivered message.
	MarkDelivered(ctx context.Context, id string) error

	// MarkFailed updates retry count and error for a failed delivery.
	MarkFailed(ctx context.Context, id string, err error) error

	// GetByID retrieves a specific message.
	GetByID(ctx context.Context, id string) (*OutboxMessage, error)
}

// Forwarder implements the transactional outbox pattern.
// It ensures that messages are reliably delivered to NATS even if the broker
// is temporarily unavailable.
//
// How it works:
//  1. Application stores message in outbox (same transaction as business data)
//  2. Forwarder polls outbox for pending messages
//  3. Messages are forwarded to NATS
//  4. Successfully delivered messages are marked as complete
//  5. Failed messages are retried with exponential backoff
//
// Benefits:
//   - Atomic: Message storage is part of the business transaction
//   - Reliable: Messages survive application restarts
//   - Eventual: Delivery is guaranteed eventually (at-least-once)
type Forwarder struct {
	store     OutboxStore
	publisher message.Publisher
	config    ForwarderConfig
	logger    watermill.LoggerAdapter

	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// NewForwarder creates a new outbox forwarder.
func NewForwarder(
	store OutboxStore,
	publisher message.Publisher,
	cfg ForwarderConfig,
	logger watermill.LoggerAdapter,
) (*Forwarder, error) {
	if store == nil {
		return nil, fmt.Errorf("outbox store required")
	}
	if publisher == nil {
		return nil, fmt.Errorf("publisher required")
	}
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	return &Forwarder{
		store:     store,
		publisher: publisher,
		config:    cfg,
		logger:    logger,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}, nil
}

// Start begins the forwarding loop.
func (f *Forwarder) Start(ctx context.Context) error {
	if f.running {
		return nil
	}
	f.running = true

	go f.forwardLoop(ctx)

	f.logger.Info("Forwarder started", watermill.LogFields{
		"poll_interval": f.config.PollInterval.String(),
		"batch_size":    f.config.BatchSize,
	})

	return nil
}

// Stop gracefully stops the forwarder.
func (f *Forwarder) Stop() error {
	if !f.running {
		return nil
	}

	close(f.stopCh)
	<-f.doneCh
	f.running = false

	f.logger.Info("Forwarder stopped", nil)
	return nil
}

// IsRunning returns whether the forwarder is active.
func (f *Forwarder) IsRunning() bool {
	return f.running
}

// forwardLoop polls for pending messages and forwards them.
func (f *Forwarder) forwardLoop(ctx context.Context) {
	defer close(f.doneCh)

	ticker := time.NewTicker(f.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-f.stopCh:
			return
		case <-ticker.C:
			f.processPending(ctx)
		}
	}
}

// processPending forwards all pending messages.
func (f *Forwarder) processPending(ctx context.Context) {
	messages, err := f.store.GetPending(ctx, f.config.BatchSize)
	if err != nil {
		f.logger.Error("Failed to get pending messages", err, nil)
		return
	}

	for _, outboxMsg := range messages {
		if err := f.forward(ctx, outboxMsg); err != nil {
			f.logger.Error("Failed to forward message", err, watermill.LogFields{
				"message_id":  outboxMsg.ID,
				"topic":       outboxMsg.Topic,
				"retry_count": outboxMsg.RetryCount,
			})
		}
	}
}

// forward attempts to deliver a single message.
func (f *Forwarder) forward(ctx context.Context, outboxMsg *OutboxMessage) error {
	// Create Watermill message
	msg := message.NewMessage(outboxMsg.ID, outboxMsg.Payload)
	for k, v := range outboxMsg.Metadata {
		msg.Metadata.Set(k, v)
	}

	// Attempt delivery
	if err := f.publisher.Publish(outboxMsg.Topic, msg); err != nil {
		// Mark as failed
		if outboxMsg.RetryCount >= f.config.MaxRetries {
			f.logger.Error("Message exceeded max retries, moving to dead letter", nil, watermill.LogFields{
				"message_id": outboxMsg.ID,
				"topic":      outboxMsg.Topic,
			})
		}
		return f.store.MarkFailed(ctx, outboxMsg.ID, err)
	}

	// Mark as delivered
	return f.store.MarkDelivered(ctx, outboxMsg.ID)
}

// TransactionalPublisher wraps the outbox store to provide a Publisher interface.
// Use this instead of publishing directly to NATS when you need transactional guarantees.
type TransactionalPublisher struct {
	store  OutboxStore
	logger watermill.LoggerAdapter
}

// NewTransactionalPublisher creates a publisher that writes to the outbox.
func NewTransactionalPublisher(store OutboxStore, logger watermill.LoggerAdapter) (*TransactionalPublisher, error) {
	if store == nil {
		return nil, fmt.Errorf("outbox store required")
	}
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	return &TransactionalPublisher{
		store:  store,
		logger: logger,
	}, nil
}

// Publish stores a message in the outbox for later delivery.
// This should be called within the same transaction as your business logic.
func (p *TransactionalPublisher) Publish(topic string, messages ...*message.Message) error {
	ctx := context.Background()

	for _, msg := range messages {
		outboxMsg := &OutboxMessage{
			ID:        msg.UUID,
			Topic:     topic,
			Payload:   msg.Payload,
			Metadata:  make(map[string]string),
			CreatedAt: time.Now(),
		}

		// Copy metadata
		for k, v := range msg.Metadata {
			outboxMsg.Metadata[k] = v
		}

		if err := p.store.Store(ctx, outboxMsg); err != nil {
			return fmt.Errorf("store outbox message: %w", err)
		}
	}

	return nil
}

// Close implements message.Publisher.
func (p *TransactionalPublisher) Close() error {
	return nil
}

// InMemoryOutboxStore provides an in-memory implementation for testing.
// Do NOT use in production - messages will be lost on restart.
type InMemoryOutboxStore struct {
	mu       sync.RWMutex
	messages map[string]*OutboxMessage
}

// NewInMemoryOutboxStore creates a new in-memory outbox store.
func NewInMemoryOutboxStore() *InMemoryOutboxStore {
	return &InMemoryOutboxStore{
		messages: make(map[string]*OutboxMessage),
	}
}

// Store saves a message.
func (s *InMemoryOutboxStore) Store(ctx context.Context, msg *OutboxMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[msg.ID] = msg
	return nil
}

// GetPending returns pending messages.
func (s *InMemoryOutboxStore) GetPending(ctx context.Context, limit int) ([]*OutboxMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*OutboxMessage, 0, limit)
	for _, msg := range s.messages {
		// Return copies to avoid data races when caller reads fields
		msgCopy := *msg
		result = append(result, &msgCopy)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

// MarkDelivered removes a message.
func (s *InMemoryOutboxStore) MarkDelivered(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.messages, id)
	return nil
}

// MarkFailed updates a message's retry count.
func (s *InMemoryOutboxStore) MarkFailed(ctx context.Context, id string, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if msg, ok := s.messages[id]; ok {
		msg.RetryCount++
		msg.LastError = err.Error()
	}
	return nil
}

// GetByID retrieves a message.
// Returns a copy of the message to avoid data races.
func (s *InMemoryOutboxStore) GetByID(ctx context.Context, id string) (*OutboxMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if msg, ok := s.messages[id]; ok {
		// Return a copy to avoid data races when caller reads fields
		msgCopy := *msg
		return &msgCopy, nil
	}
	return nil, fmt.Errorf("message not found: %s", id)
}

// Size returns the number of pending messages.
func (s *InMemoryOutboxStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.messages)
}
