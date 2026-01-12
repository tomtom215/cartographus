// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"fmt"

	gobreaker "github.com/sony/gobreaker/v2"
)

// Publisher is a stub when NATS dependencies are not available.
// Build with -tags=nats to enable full Watermill publisher support.
type Publisher struct {
	circuitBreaker *gobreaker.CircuitBreaker[interface{}]
}

// NewPublisher returns an error when NATS dependencies are not available.
// Build with -tags=nats to enable full Watermill publisher support.
func NewPublisher(cfg PublisherConfig, logger interface{}) (*Publisher, error) {
	return nil, fmt.Errorf("NATS publisher not available: build with -tags=nats")
}

// SetCircuitBreaker configures the circuit breaker for publish operations.
func (p *Publisher) SetCircuitBreaker(cb *gobreaker.CircuitBreaker[interface{}]) {
	p.circuitBreaker = cb
}

// Publish is a stub that returns an error.
func (p *Publisher) Publish(ctx context.Context, topic string, msg interface{}) error {
	return fmt.Errorf("NATS publisher not available: build with -tags=nats")
}

// PublishEvent is a stub that returns an error.
func (p *Publisher) PublishEvent(ctx context.Context, event *MediaEvent) error {
	return fmt.Errorf("NATS publisher not available: build with -tags=nats")
}

// PublishBatch is a stub that returns an error.
func (p *Publisher) PublishBatch(ctx context.Context, topic string, msgs ...interface{}) error {
	return fmt.Errorf("NATS publisher not available: build with -tags=nats")
}

// Close is a no-op stub.
func (p *Publisher) Close() error {
	return nil
}

// WatermillPublisher returns nil for the stub implementation.
// This method is only used when NATS is enabled, so the stub returns nil.
func (p *Publisher) WatermillPublisher() interface{} {
	return nil
}
