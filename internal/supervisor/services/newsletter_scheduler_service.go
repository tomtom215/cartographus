// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package services

import (
	"context"
	"fmt"
)

// NewsletterSchedulerManager interface matches the newsletter scheduler lifecycle.
//
// This interface abstracts the scheduler's Start/Stop pattern, allowing the
// NewsletterSchedulerService wrapper to adapt it to suture's Serve pattern
// without modifying the scheduler code.
//
// The interface is satisfied by *scheduler.Scheduler from internal/newsletter/scheduler/scheduler.go.
type NewsletterSchedulerManager interface {
	Start(ctx context.Context) error
	Stop() error
}

// NewsletterSchedulerService wraps the newsletter scheduler as a supervised service.
//
// It adapts the Start/Stop lifecycle pattern to suture's Serve pattern:
//  1. Calls Start(ctx) to begin the scheduler
//  2. Waits for context cancellation
//  3. Calls Stop() for graceful shutdown
type NewsletterSchedulerService struct {
	manager NewsletterSchedulerManager
	name    string
}

// NewNewsletterSchedulerService creates a new newsletter scheduler service wrapper.
//
// Example usage:
//
//	scheduler := scheduler.NewScheduler(store, contentResolver, templateEngine, deliveryManager, logger, cfg)
//	svc := services.NewNewsletterSchedulerService(scheduler)
//	tree.AddMessagingService(svc)
func NewNewsletterSchedulerService(manager NewsletterSchedulerManager) *NewsletterSchedulerService {
	return &NewsletterSchedulerService{
		manager: manager,
		name:    "newsletter-scheduler",
	}
}

// Serve implements suture.Service.
//
// This method:
//  1. Starts the scheduler (which spawns its internal loop)
//  2. Blocks until the context is canceled
//  3. Stops the scheduler gracefully
//
// If Start() fails, the error is returned immediately, causing suture to
// restart the service according to its backoff policy.
func (s *NewsletterSchedulerService) Serve(ctx context.Context) error {
	// Start the scheduler
	if err := s.manager.Start(ctx); err != nil {
		return fmt.Errorf("newsletter scheduler start failed: %w", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop the scheduler gracefully
	if err := s.manager.Stop(); err != nil {
		return fmt.Errorf("newsletter scheduler stop failed: %w", err)
	}

	return ctx.Err()
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify the service in log messages.
func (s *NewsletterSchedulerService) String() string {
	return s.name
}
