// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package scheduler provides cron-based scheduling for newsletter delivery.
//
// scheduler.go - Newsletter Scheduler Service
//
// This file implements the scheduler service that:
//   - Runs on a configurable interval (default: 1 minute)
//   - Queries for schedules that are due to execute
//   - For each due schedule:
//     1. Fetches the template
//     2. Resolves content from the database
//     3. Renders HTML and plaintext versions
//     4. Delivers via configured channels
//     5. Updates schedule status and next run time
//
// The scheduler integrates with the supervisor tree for lifecycle management.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/newsletter"
	"github.com/tomtom215/cartographus/internal/newsletter/delivery"
)

// SchedulerStore defines the database operations required by the scheduler.
type SchedulerStore interface {
	// Schedule operations
	GetSchedulesDueForRun(ctx context.Context) ([]models.NewsletterSchedule, error)
	GetNewsletterSchedule(ctx context.Context, id string) (*models.NewsletterSchedule, error)
	UpdateScheduleRunStatus(ctx context.Context, id string, status models.DeliveryStatus, nextRunAt *time.Time) error

	// Template operations
	GetNewsletterTemplate(ctx context.Context, id string) (*models.NewsletterTemplate, error)

	// Delivery operations
	CreateNewsletterDelivery(ctx context.Context, delivery *models.NewsletterDelivery) error
	UpdateNewsletterDelivery(ctx context.Context, delivery *models.NewsletterDelivery) error
	GetNewsletterDelivery(ctx context.Context, id string) (*models.NewsletterDelivery, error)
}

// Config holds configuration for the newsletter scheduler.
type Config struct {
	// CheckInterval is how often to check for due schedules (default: 1 minute)
	CheckInterval time.Duration

	// MaxConcurrentDeliveries is the maximum number of newsletters to deliver concurrently
	MaxConcurrentDeliveries int

	// ExecutionTimeout is the maximum time allowed for a single newsletter execution
	ExecutionTimeout time.Duration

	// Enabled controls whether the scheduler is active
	Enabled bool
}

// DefaultConfig returns the default scheduler configuration.
func DefaultConfig() Config {
	return Config{
		CheckInterval:           time.Minute,
		MaxConcurrentDeliveries: 5,
		ExecutionTimeout:        5 * time.Minute,
		Enabled:                 true,
	}
}

// Scheduler manages cron-based newsletter delivery.
type Scheduler struct {
	store           SchedulerStore
	contentResolver *newsletter.ContentResolver
	templateEngine  *newsletter.TemplateEngine
	deliveryManager *delivery.Manager
	logger          zerolog.Logger
	config          Config

	// Runtime state
	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// NewScheduler creates a new newsletter scheduler.
func NewScheduler(
	store SchedulerStore,
	contentResolver *newsletter.ContentResolver,
	templateEngine *newsletter.TemplateEngine,
	deliveryManager *delivery.Manager,
	logger *zerolog.Logger,
	config Config,
) *Scheduler {
	if config.CheckInterval <= 0 {
		config.CheckInterval = time.Minute
	}
	if config.MaxConcurrentDeliveries <= 0 {
		config.MaxConcurrentDeliveries = 5
	}
	if config.ExecutionTimeout <= 0 {
		config.ExecutionTimeout = 5 * time.Minute
	}

	return &Scheduler{
		store:           store,
		contentResolver: contentResolver,
		templateEngine:  templateEngine,
		deliveryManager: deliveryManager,
		logger:          logger.With().Str("component", "newsletter-scheduler").Logger(),
		config:          config,
	}
}

// Start begins the scheduler loop.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	if !s.config.Enabled {
		s.logger.Info().Msg("Newsletter scheduler disabled")
		// Keep goroutine alive but don't do anything
		go func() {
			defer close(s.doneCh)
			<-s.stopCh
		}()
		return nil
	}

	s.logger.Info().
		Dur("check_interval", s.config.CheckInterval).
		Int("max_concurrent", s.config.MaxConcurrentDeliveries).
		Msg("Starting newsletter scheduler")

	go s.run(ctx)
	return nil
}

// Stop stops the scheduler loop and waits for it to complete.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	s.logger.Info().Msg("Stopping newsletter scheduler...")
	close(s.stopCh)
	<-s.doneCh

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	s.logger.Info().Msg("Newsletter scheduler stopped")
	return nil
}

// run is the main scheduler loop.
func (s *Scheduler) run(ctx context.Context) {
	defer close(s.doneCh)

	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.checkAndExecute(ctx)

	for {
		select {
		case <-ticker.C:
			s.checkAndExecute(ctx)
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkAndExecute checks for due schedules and executes them.
func (s *Scheduler) checkAndExecute(ctx context.Context) {
	// Get schedules that are due
	schedules, err := s.store.GetSchedulesDueForRun(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get due schedules")
		return
	}

	if len(schedules) == 0 {
		s.logger.Debug().Msg("No schedules due for execution")
		return
	}

	s.logger.Info().Int("count", len(schedules)).Msg("Found schedules due for execution")

	// Execute schedules with concurrency limit
	sem := make(chan struct{}, s.config.MaxConcurrentDeliveries)
	var wg sync.WaitGroup

	for i := range schedules {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			execCtx, cancel := context.WithTimeout(ctx, s.config.ExecutionTimeout)
			defer cancel()

			s.executeSchedule(execCtx, &schedules[idx])
		}(i)
	}

	wg.Wait()
}

// executeSchedule executes a single newsletter schedule.
func (s *Scheduler) executeSchedule(ctx context.Context, schedule *models.NewsletterSchedule) {
	startTime := time.Now()
	logger := s.logger.With().
		Str("schedule_id", schedule.ID).
		Str("schedule_name", schedule.Name).
		Str("template_id", schedule.TemplateID).
		Logger()

	logger.Info().Msg("Executing newsletter schedule")

	// Use the first channel for the primary delivery record
	// (Each channel could have its own record, but for simplicity we track aggregate)
	primaryChannel := models.DeliveryChannelEmail
	if len(schedule.Channels) > 0 {
		primaryChannel = schedule.Channels[0]
	}

	// Create delivery record
	deliveryID := uuid.New().String()
	dlv := &models.NewsletterDelivery{
		ID:              deliveryID,
		ScheduleID:      schedule.ID,
		TemplateID:      schedule.TemplateID,
		Channel:         primaryChannel,
		Status:          models.DeliveryStatusPending,
		RecipientsTotal: len(schedule.Recipients) * len(schedule.Channels),
		StartedAt:       startTime,
		TriggeredBy:     "schedule",
	}

	if err := s.store.CreateNewsletterDelivery(ctx, dlv); err != nil {
		logger.Error().Err(err).Msg("Failed to create delivery record")
		s.updateScheduleStatus(ctx, schedule, models.DeliveryStatusFailed)
		return
	}

	// Fetch template
	tmpl, err := s.store.GetNewsletterTemplate(ctx, schedule.TemplateID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch template")
		s.failDelivery(ctx, deliveryID, "Failed to fetch template: "+err.Error())
		s.updateScheduleStatus(ctx, schedule, models.DeliveryStatusFailed)
		return
	}

	// Resolve content
	contentConfig := schedule.Config
	if contentConfig == nil {
		contentConfig = tmpl.DefaultConfig
	}

	content, err := s.contentResolver.ResolveContent(ctx, tmpl.Type, contentConfig, nil)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to resolve content")
		s.failDelivery(ctx, deliveryID, "Failed to resolve content: "+err.Error())
		s.updateScheduleStatus(ctx, schedule, models.DeliveryStatusFailed)
		return
	}

	// Render templates
	renderedSubject, err := s.templateEngine.RenderSubject(tmpl.Subject, content)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to render subject")
		s.failDelivery(ctx, deliveryID, "Failed to render subject: "+err.Error())
		s.updateScheduleStatus(ctx, schedule, models.DeliveryStatusFailed)
		return
	}

	renderedHTML, err := s.templateEngine.RenderHTML(tmpl.BodyHTML, content)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to render HTML")
		s.failDelivery(ctx, deliveryID, "Failed to render HTML: "+err.Error())
		s.updateScheduleStatus(ctx, schedule, models.DeliveryStatusFailed)
		return
	}

	renderedText, err := s.templateEngine.RenderText(tmpl.BodyText, content)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to render text")
		s.failDelivery(ctx, deliveryID, "Failed to render text: "+err.Error())
		s.updateScheduleStatus(ctx, schedule, models.DeliveryStatusFailed)
		return
	}

	// Update delivery with rendered content and status
	s.updateDeliveryStatus(ctx, deliveryID, models.DeliveryStatusSending, renderedSubject)

	// Prepare delivery request
	req := &delivery.DeliveryRequest{
		DeliveryID:      deliveryID,
		ScheduleID:      schedule.ID,
		Template:        tmpl,
		Recipients:      schedule.Recipients,
		Channels:        schedule.Channels,
		ChannelConfigs:  schedule.ChannelConfigs,
		RenderedSubject: renderedSubject,
		RenderedHTML:    renderedHTML,
		RenderedText:    renderedText,
	}

	// Deliver newsletter
	report, err := s.deliveryManager.Deliver(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("Delivery failed")
		s.failDelivery(ctx, deliveryID, "Delivery failed: "+err.Error())
		s.updateScheduleStatus(ctx, schedule, models.DeliveryStatusFailed)
		return
	}

	// Update delivery record with results
	completedAt := time.Now()
	s.completeDelivery(ctx, deliveryID, report, &completedAt)

	// Determine final status
	finalStatus := models.DeliveryStatusDelivered
	if report.FailedDeliveries > 0 {
		if report.SuccessfulDeliveries == 0 {
			finalStatus = models.DeliveryStatusFailed
		} else {
			finalStatus = models.DeliveryStatusPartial
		}
	}

	s.updateScheduleStatus(ctx, schedule, finalStatus)

	logger.Info().
		Str("status", string(finalStatus)).
		Int("successful", report.SuccessfulDeliveries).
		Int("failed", report.FailedDeliveries).
		Dur("duration", time.Since(startTime)).
		Msg("Newsletter schedule executed")
}

// updateScheduleStatus updates the schedule run status and calculates next run time.
func (s *Scheduler) updateScheduleStatus(ctx context.Context, schedule *models.NewsletterSchedule, status models.DeliveryStatus) {
	// Calculate next run time
	var nextRunAt *time.Time
	if schedule.CronExpression != "" {
		nextTime, err := CalculateNextRun(schedule.CronExpression, time.Now(), schedule.Timezone)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("schedule_id", schedule.ID).
				Str("cron", schedule.CronExpression).
				Msg("Failed to calculate next run time")
		} else {
			nextRunAt = &nextTime
		}
	}

	if err := s.store.UpdateScheduleRunStatus(ctx, schedule.ID, status, nextRunAt); err != nil {
		s.logger.Error().
			Err(err).
			Str("schedule_id", schedule.ID).
			Msg("Failed to update schedule run status")
	}
}

// failDelivery marks a delivery as failed with an error message.
func (s *Scheduler) failDelivery(ctx context.Context, deliveryID, errorMsg string) {
	dlv, err := s.store.GetNewsletterDelivery(ctx, deliveryID)
	if err != nil {
		s.logger.Error().Err(err).Str("delivery_id", deliveryID).Msg("Failed to get delivery for update")
		return
	}

	now := time.Now()
	dlv.Status = models.DeliveryStatusFailed
	dlv.CompletedAt = &now
	dlv.ErrorMessage = errorMsg
	dlv.DurationMS = now.Sub(dlv.StartedAt).Milliseconds()

	if err := s.store.UpdateNewsletterDelivery(ctx, dlv); err != nil {
		s.logger.Error().Err(err).Str("delivery_id", deliveryID).Msg("Failed to update delivery status")
	}
}

// updateDeliveryStatus updates delivery status to sending.
func (s *Scheduler) updateDeliveryStatus(ctx context.Context, deliveryID string, status models.DeliveryStatus, renderedSubject string) {
	dlv, err := s.store.GetNewsletterDelivery(ctx, deliveryID)
	if err != nil {
		s.logger.Error().Err(err).Str("delivery_id", deliveryID).Msg("Failed to get delivery for update")
		return
	}

	dlv.Status = status
	dlv.RenderedSubject = renderedSubject

	if err := s.store.UpdateNewsletterDelivery(ctx, dlv); err != nil {
		s.logger.Error().Err(err).Str("delivery_id", deliveryID).Msg("Failed to update delivery status")
	}
}

// completeDelivery marks a delivery as complete with the delivery report.
func (s *Scheduler) completeDelivery(ctx context.Context, deliveryID string, report *delivery.DeliveryReport, completedAt *time.Time) {
	dlv, err := s.store.GetNewsletterDelivery(ctx, deliveryID)
	if err != nil {
		s.logger.Error().Err(err).Str("delivery_id", deliveryID).Msg("Failed to get delivery for completion")
		return
	}

	dlv.Status = report.Status
	dlv.CompletedAt = completedAt
	dlv.RecipientsDelivered = report.SuccessfulDeliveries
	dlv.RecipientsFailed = report.FailedDeliveries
	if completedAt != nil {
		dlv.DurationMS = completedAt.Sub(dlv.StartedAt).Milliseconds()
	}

	if err := s.store.UpdateNewsletterDelivery(ctx, dlv); err != nil {
		s.logger.Error().Err(err).Str("delivery_id", deliveryID).Msg("Failed to update delivery completion")
	}
}

// IsRunning returns whether the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
