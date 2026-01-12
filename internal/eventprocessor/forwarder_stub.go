// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"fmt"
	"time"
)

// ForwarderConfig holds configuration for the Outbox Forwarder.
type ForwarderConfig struct {
	PollInterval time.Duration
	RetryDelay   time.Duration
	MaxRetries   int
	BatchSize    int
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

// OutboxMessage represents a message stored in the outbox.
type OutboxMessage struct {
	ID         string
	Topic      string
	Payload    []byte
	Metadata   map[string]string
	CreatedAt  time.Time
	RetryCount int
	LastError  string
}

// OutboxStore defines the interface for persistent outbox storage.
type OutboxStore interface {
	Store(ctx context.Context, msg *OutboxMessage) error
	GetPending(ctx context.Context, limit int) ([]*OutboxMessage, error)
	MarkDelivered(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, err error) error
	GetByID(ctx context.Context, id string) (*OutboxMessage, error)
}

// Forwarder is a stub for non-NATS builds.
type Forwarder struct{}

// Start is a stub for non-NATS builds.
func (f *Forwarder) Start(_ context.Context) error {
	return ErrNATSNotEnabled
}

// Stop is a stub for non-NATS builds.
func (f *Forwarder) Stop() error {
	return nil
}

// IsRunning is a stub for non-NATS builds.
func (f *Forwarder) IsRunning() bool {
	return false
}

// TransactionalPublisher is a stub for non-NATS builds.
type TransactionalPublisher struct{}

// Publish is a stub for non-NATS builds.
func (p *TransactionalPublisher) Publish(_ string, _ ...interface{}) error {
	return ErrNATSNotEnabled
}

// Close is a stub for non-NATS builds.
func (p *TransactionalPublisher) Close() error {
	return nil
}

// InMemoryOutboxStore provides an in-memory implementation.
type InMemoryOutboxStore struct {
	messages map[string]*OutboxMessage
}

// NewInMemoryOutboxStore creates a new in-memory outbox store.
func NewInMemoryOutboxStore() *InMemoryOutboxStore {
	return &InMemoryOutboxStore{
		messages: make(map[string]*OutboxMessage),
	}
}

// Store saves a message.
func (s *InMemoryOutboxStore) Store(_ context.Context, msg *OutboxMessage) error {
	s.messages[msg.ID] = msg
	return nil
}

// GetPending returns pending messages.
func (s *InMemoryOutboxStore) GetPending(_ context.Context, limit int) ([]*OutboxMessage, error) {
	result := make([]*OutboxMessage, 0, limit)
	for _, msg := range s.messages {
		result = append(result, msg)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

// MarkDelivered removes a message.
func (s *InMemoryOutboxStore) MarkDelivered(_ context.Context, id string) error {
	delete(s.messages, id)
	return nil
}

// MarkFailed updates a message's retry count.
func (s *InMemoryOutboxStore) MarkFailed(_ context.Context, id string, err error) error {
	if msg, ok := s.messages[id]; ok {
		msg.RetryCount++
		msg.LastError = err.Error()
	}
	return nil
}

// GetByID retrieves a message.
func (s *InMemoryOutboxStore) GetByID(_ context.Context, id string) (*OutboxMessage, error) {
	if msg, ok := s.messages[id]; ok {
		return msg, nil
	}
	return nil, fmt.Errorf("message not found: %s", id)
}

// Size returns the number of pending messages.
func (s *InMemoryOutboxStore) Size() int {
	return len(s.messages)
}
