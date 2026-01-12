// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

func TestDefaultForwarderConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultForwarderConfig()

	if cfg.PollInterval != 100*time.Millisecond {
		t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, 100*time.Millisecond)
	}
	if cfg.RetryDelay != time.Second {
		t.Errorf("RetryDelay = %v, want %v", cfg.RetryDelay, time.Second)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", cfg.BatchSize)
	}
}

func TestInMemoryOutboxStore(t *testing.T) {
	t.Parallel()

	store := NewInMemoryOutboxStore()
	ctx := context.Background()

	// Test Store
	msg := &OutboxMessage{
		ID:        "test-msg-1",
		Topic:     "test.topic",
		Payload:   []byte("test payload"),
		Metadata:  map[string]string{"key": "value"},
		CreatedAt: time.Now(),
	}

	err := store.Store(ctx, msg)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}

	// Test GetByID
	retrieved, err := store.GetByID(ctx, "test-msg-1")
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if retrieved.ID != msg.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, msg.ID)
	}

	// Test Size
	if store.Size() != 1 {
		t.Errorf("Size = %d, want 1", store.Size())
	}

	// Test GetPending
	pending, err := store.GetPending(ctx, 10)
	if err != nil {
		t.Fatalf("GetPending error: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("Pending count = %d, want 1", len(pending))
	}

	// Test MarkFailed
	testErr := errors.New("test error")
	err = store.MarkFailed(ctx, "test-msg-1", testErr)
	if err != nil {
		t.Fatalf("MarkFailed error: %v", err)
	}

	retrieved, _ = store.GetByID(ctx, "test-msg-1")
	if retrieved.RetryCount != 1 {
		t.Errorf("RetryCount = %d, want 1", retrieved.RetryCount)
	}
	if retrieved.LastError != "test error" {
		t.Errorf("LastError = %q, want %q", retrieved.LastError, "test error")
	}

	// Test MarkDelivered
	err = store.MarkDelivered(ctx, "test-msg-1")
	if err != nil {
		t.Fatalf("MarkDelivered error: %v", err)
	}
	if store.Size() != 0 {
		t.Errorf("Size after delivery = %d, want 0", store.Size())
	}
}

func TestInMemoryOutboxStore_NotFound(t *testing.T) {
	t.Parallel()

	store := NewInMemoryOutboxStore()
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Error("GetByID should error for nonexistent message")
	}
}

// MockForwarderPublisher tracks published messages.
type MockForwarderPublisher struct {
	publishedMessages []*message.Message
	publishedTopics   []string
	publishCount      atomic.Int32
	failNext          bool
	failErr           error
}

func (p *MockForwarderPublisher) Publish(topic string, messages ...*message.Message) error {
	if p.failNext {
		p.failNext = false
		return p.failErr
	}
	p.publishCount.Add(int32(len(messages)))
	for _, msg := range messages {
		p.publishedMessages = append(p.publishedMessages, msg)
		p.publishedTopics = append(p.publishedTopics, topic)
	}
	return nil
}

func (p *MockForwarderPublisher) Close() error {
	return nil
}

func TestNewForwarder_NilStore(t *testing.T) {
	t.Parallel()

	pub := &MockForwarderPublisher{}
	_, err := NewForwarder(nil, pub, DefaultForwarderConfig(), nil)
	if err == nil {
		t.Error("NewForwarder should error with nil store")
	}
}

func TestNewForwarder_NilPublisher(t *testing.T) {
	t.Parallel()

	store := NewInMemoryOutboxStore()
	_, err := NewForwarder(store, nil, DefaultForwarderConfig(), nil)
	if err == nil {
		t.Error("NewForwarder should error with nil publisher")
	}
}

func TestForwarder_StartStop(t *testing.T) {
	t.Parallel()

	store := NewInMemoryOutboxStore()
	pub := &MockForwarderPublisher{}
	cfg := DefaultForwarderConfig()
	cfg.PollInterval = 10 * time.Millisecond

	fwd, err := NewForwarder(store, pub, cfg, nil)
	if err != nil {
		t.Fatalf("NewForwarder error: %v", err)
	}

	ctx := context.Background()

	// Start
	if err := fwd.Start(ctx); err != nil {
		t.Fatalf("Start error: %v", err)
	}

	if !fwd.IsRunning() {
		t.Error("Forwarder should be running after Start")
	}

	// Double start should be safe
	if err := fwd.Start(ctx); err != nil {
		t.Errorf("Double Start error: %v", err)
	}

	// Stop
	if err := fwd.Stop(); err != nil {
		t.Errorf("Stop error: %v", err)
	}

	if fwd.IsRunning() {
		t.Error("Forwarder should not be running after Stop")
	}

	// Double stop should be safe
	if err := fwd.Stop(); err != nil {
		t.Errorf("Double Stop error: %v", err)
	}
}

func TestForwarder_ForwardsPendingMessages(t *testing.T) {
	t.Parallel()

	store := NewInMemoryOutboxStore()
	pub := &MockForwarderPublisher{}
	cfg := DefaultForwarderConfig()
	cfg.PollInterval = 10 * time.Millisecond

	fwd, err := NewForwarder(store, pub, cfg, nil)
	if err != nil {
		t.Fatalf("NewForwarder error: %v", err)
	}

	ctx := context.Background()

	// Add message to outbox
	msg := &OutboxMessage{
		ID:        "forward-test-1",
		Topic:     "test.topic",
		Payload:   []byte("test data"),
		Metadata:  map[string]string{"header": "value"},
		CreatedAt: time.Now(),
	}
	store.Store(ctx, msg)

	// Start forwarder
	fwd.Start(ctx)

	// Wait for message to be forwarded with polling (more reliable in CI under load)
	var publishCount int32
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		publishCount = pub.publishCount.Load()
		if publishCount >= 1 {
			break
		}
	}

	fwd.Stop()

	// Verify message was published
	if publishCount != 1 {
		t.Errorf("Publish count = %d, want 1", publishCount)
	}

	// Verify message was removed from outbox
	if store.Size() != 0 {
		t.Errorf("Outbox size = %d, want 0", store.Size())
	}
}

func TestForwarder_RetriesFailedMessages(t *testing.T) {
	t.Parallel()

	store := NewInMemoryOutboxStore()
	pub := &MockForwarderPublisher{}
	cfg := DefaultForwarderConfig()
	cfg.PollInterval = 10 * time.Millisecond
	cfg.MaxRetries = 3

	fwd, err := NewForwarder(store, pub, cfg, nil)
	if err != nil {
		t.Fatalf("NewForwarder error: %v", err)
	}

	ctx := context.Background()

	// Add message to outbox
	msg := &OutboxMessage{
		ID:        "retry-test-1",
		Topic:     "test.topic",
		Payload:   []byte("test data"),
		CreatedAt: time.Now(),
	}
	store.Store(ctx, msg)

	// Make publisher fail first attempt
	pub.failNext = true
	pub.failErr = errors.New("publish failed")

	// Start forwarder
	fwd.Start(ctx)

	// Wait for first attempt (which will fail)
	time.Sleep(30 * time.Millisecond)

	// Message should still be in outbox (first attempt failed)
	// Check if message exists and has retry count incremented
	retrieved, err := store.GetByID(ctx, "retry-test-1")
	if err == nil && retrieved != nil {
		// Message still in retry state
		if retrieved.RetryCount < 1 {
			t.Errorf("RetryCount = %d, want >= 1", retrieved.RetryCount)
		}
	}
	// If message is nil, it may have been delivered already (race condition)

	// Wait for successful retry with polling (more reliable in CI under load)
	var storeSize int
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		storeSize = store.Size()
		if storeSize == 0 {
			break
		}
	}

	fwd.Stop()

	// Message should be delivered now
	if storeSize != 0 {
		t.Errorf("Outbox size = %d, want 0", storeSize)
	}
}

func TestTransactionalPublisher(t *testing.T) {
	t.Parallel()

	store := NewInMemoryOutboxStore()
	pub, err := NewTransactionalPublisher(store, nil)
	if err != nil {
		t.Fatalf("NewTransactionalPublisher error: %v", err)
	}

	// Publish message
	msg := message.NewMessage("tx-msg-1", []byte("transaction data"))
	msg.Metadata.Set("header", "value")

	err = pub.Publish("tx.topic", msg)
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	// Verify message in outbox
	if store.Size() != 1 {
		t.Errorf("Outbox size = %d, want 1", store.Size())
	}

	ctx := context.Background()
	retrieved, _ := store.GetByID(ctx, "tx-msg-1")
	if retrieved.Topic != "tx.topic" {
		t.Errorf("Topic = %q, want %q", retrieved.Topic, "tx.topic")
	}
	if string(retrieved.Payload) != "transaction data" {
		t.Errorf("Payload = %q, want %q", string(retrieved.Payload), "transaction data")
	}
	if retrieved.Metadata["header"] != "value" {
		t.Errorf("Metadata[header] = %q, want %q", retrieved.Metadata["header"], "value")
	}
}

func TestTransactionalPublisher_NilStore(t *testing.T) {
	t.Parallel()

	_, err := NewTransactionalPublisher(nil, nil)
	if err == nil {
		t.Error("NewTransactionalPublisher should error with nil store")
	}
}

func TestTransactionalPublisher_Close(t *testing.T) {
	t.Parallel()

	store := NewInMemoryOutboxStore()
	pub, _ := NewTransactionalPublisher(store, nil)

	err := pub.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}
}
