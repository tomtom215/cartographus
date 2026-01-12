// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
	"time"
)

// RouterConfig holds configuration for the Watermill Router.
// This is a stub for non-NATS builds.
type RouterConfig struct {
	CloseTimeout         time.Duration
	RetryMaxRetries      int
	RetryInitialInterval time.Duration
	RetryMaxInterval     time.Duration
	RetryMultiplier      float64
	ThrottlePerSecond    int64
	PoisonQueueTopic     string
	DeduplicationEnabled bool
	DeduplicationTTL     time.Duration
}

// DefaultRouterConfig returns production defaults for the Router.
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		CloseTimeout:         30 * time.Second,
		RetryMaxRetries:      5,
		RetryInitialInterval: time.Second,
		RetryMaxInterval:     time.Minute,
		RetryMultiplier:      2.0,
		ThrottlePerSecond:    0,
		PoisonQueueTopic:     "dlq.playback",
		DeduplicationEnabled: false,
		DeduplicationTTL:     5 * time.Minute,
	}
}

// Router is a stub for non-NATS builds.
type Router struct{}

// NewRouter is a stub for non-NATS builds.
func NewRouter(_ *RouterConfig, _ interface{}, _ interface{}) (*Router, error) {
	return nil, ErrNATSNotEnabled
}

// AddConsumerHandler is a stub for non-NATS builds.
func (r *Router) AddConsumerHandler(_ string, _ string, _ interface{}, _ interface{}) interface{} {
	return nil
}

// AddHandler is a stub for non-NATS builds.
func (r *Router) AddHandler(_ string, _ string, _ interface{}, _ string, _ interface{}, _ interface{}) interface{} {
	return nil
}

// AddHandlerMiddleware is a stub for non-NATS builds.
func (r *Router) AddHandlerMiddleware(_ string, _ ...interface{}) error {
	return ErrNATSNotEnabled
}

// RouterMetrics holds runtime metrics for the Router.
type RouterMetrics struct {
	MessagesReceived     int64
	MessagesProcessed    int64
	MessagesFailed       int64
	MessagesRetried      int64
	MessagesPoisoned     int64
	MessagesDeduplicated int64
}

// InMemoryDeduplicator is a stub for non-NATS builds.
type InMemoryDeduplicator struct{}

// NewInMemoryDeduplicator is a stub for non-NATS builds.
func NewInMemoryDeduplicator(_ time.Duration) *InMemoryDeduplicator {
	return nil
}

// IsDuplicate is a stub for non-NATS builds.
func (d *InMemoryDeduplicator) IsDuplicate(_ context.Context, _ string) (bool, error) {
	return false, ErrNATSNotEnabled
}

// Run is a stub for non-NATS builds.
func (r *Router) Run(_ context.Context) error {
	return ErrNATSNotEnabled
}

// RunAsync is a stub for non-NATS builds.
func (r *Router) RunAsync(_ context.Context) <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// Running is a stub for non-NATS builds.
func (r *Router) Running() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// Close is a stub for non-NATS builds.
func (r *Router) Close() error {
	return nil
}

// IsRunning is a stub for non-NATS builds.
func (r *Router) IsRunning() bool {
	return false
}

// Metrics is a stub for non-NATS builds.
func (r *Router) Metrics() *RouterMetrics {
	return &RouterMetrics{}
}

// HealthCheck is a stub for non-NATS builds.
func (r *Router) HealthCheck(_ context.Context) ComponentHealth {
	return ComponentHealth{
		Name:      "router",
		Healthy:   false,
		Error:     "NATS not enabled",
		LastCheck: time.Now(),
	}
}
