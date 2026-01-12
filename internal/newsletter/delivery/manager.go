// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package delivery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/tomtom215/cartographus/internal/models"
)

// Manager orchestrates newsletter delivery across multiple channels.
// It handles retry logic, parallel delivery, and result aggregation.
type Manager struct {
	registry    *ChannelRegistry
	logger      zerolog.Logger
	maxRetries  int
	baseDelay   time.Duration
	maxDelay    time.Duration
	parallelism int
}

// ManagerConfig contains configuration for the delivery manager.
type ManagerConfig struct {
	// MaxRetries is the maximum number of retry attempts for transient errors.
	MaxRetries int

	// BaseDelay is the initial delay between retries.
	BaseDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration

	// Parallelism is the maximum number of concurrent deliveries.
	Parallelism int
}

// DefaultManagerConfig returns a default manager configuration.
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		MaxRetries:  3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Parallelism: 10,
	}
}

// NewManager creates a new delivery manager.
func NewManager(logger *zerolog.Logger, config ManagerConfig) *Manager {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.BaseDelay <= 0 {
		config.BaseDelay = 1 * time.Second
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.Parallelism <= 0 {
		config.Parallelism = 10
	}

	return &Manager{
		registry:    NewChannelRegistry(),
		logger:      logger.With().Str("component", "newsletter-delivery").Logger(),
		maxRetries:  config.MaxRetries,
		baseDelay:   config.BaseDelay,
		maxDelay:    config.MaxDelay,
		parallelism: config.Parallelism,
	}
}

// SetInAppStore sets the store for in-app notifications.
func (m *Manager) SetInAppStore(store InAppNotificationStore) {
	if ch, ok := m.registry.Get(models.DeliveryChannelInApp); ok {
		if inAppCh, ok := ch.(*InAppChannel); ok {
			inAppCh.SetStore(store)
		}
	}
}

// DeliveryRequest contains all information needed to deliver a newsletter.
type DeliveryRequest struct {
	// DeliveryID is the unique identifier for this delivery.
	DeliveryID string

	// ScheduleID is the schedule that triggered this delivery (if any).
	ScheduleID string

	// Template is the template being used.
	Template *models.NewsletterTemplate

	// Recipients is the list of recipients.
	Recipients []models.NewsletterRecipient

	// Channels is the list of channels to use.
	Channels []models.DeliveryChannel

	// ChannelConfigs contains channel-specific configuration.
	ChannelConfigs map[models.DeliveryChannel]*models.ChannelConfig

	// RenderedSubject is the rendered subject line.
	RenderedSubject string

	// RenderedHTML is the rendered HTML content.
	RenderedHTML string

	// RenderedText is the rendered plaintext content.
	RenderedText string

	// Metadata contains additional delivery metadata.
	Metadata *DeliveryMetadata
}

// DeliveryReport contains the aggregated results of a delivery operation.
type DeliveryReport struct {
	// DeliveryID is the unique delivery identifier.
	DeliveryID string

	// Status is the overall delivery status.
	Status models.DeliveryStatus

	// TotalRecipients is the total number of recipients.
	TotalRecipients int

	// SuccessfulDeliveries is the count of successful deliveries.
	SuccessfulDeliveries int

	// FailedDeliveries is the count of failed deliveries.
	FailedDeliveries int

	// Results contains per-recipient delivery results.
	Results []DeliveryResult

	// StartedAt is when delivery started.
	StartedAt time.Time

	// CompletedAt is when delivery completed.
	CompletedAt time.Time

	// DurationMS is the total delivery duration in milliseconds.
	DurationMS int64
}

// Deliver sends a newsletter to all recipients across all channels.
func (m *Manager) Deliver(ctx context.Context, req *DeliveryRequest) (*DeliveryReport, error) {
	report := &DeliveryReport{
		DeliveryID:      req.DeliveryID,
		TotalRecipients: len(req.Recipients) * len(req.Channels),
		StartedAt:       time.Now(),
	}

	m.logger.Info().
		Str("delivery_id", req.DeliveryID).
		Int("recipients", len(req.Recipients)).
		Int("channels", len(req.Channels)).
		Msg("starting newsletter delivery")

	// Build list of all delivery jobs
	type deliveryJob struct {
		recipient models.NewsletterRecipient
		channel   models.DeliveryChannel
	}

	var jobs []deliveryJob
	for _, recipient := range req.Recipients {
		for _, channel := range req.Channels {
			jobs = append(jobs, deliveryJob{
				recipient: recipient,
				channel:   channel,
			})
		}
	}

	// Create result channel
	results := make(chan DeliveryResult, len(jobs))

	// Create worker pool
	jobChan := make(chan deliveryJob, len(jobs))
	var wg sync.WaitGroup

	// Start workers
	workerCount := m.parallelism
	if workerCount > len(jobs) {
		workerCount = len(jobs)
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobChan {
				result := m.deliverToRecipient(ctx, req, job.recipient, job.channel)
				results <- result
			}
		}()
	}

	// Send jobs to workers
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Collect results
	for result := range results {
		report.Results = append(report.Results, result)
		if result.Success {
			report.SuccessfulDeliveries++
		} else {
			report.FailedDeliveries++
		}
	}

	report.CompletedAt = time.Now()
	report.DurationMS = report.CompletedAt.Sub(report.StartedAt).Milliseconds()

	// Determine overall status
	if report.FailedDeliveries == 0 {
		report.Status = models.DeliveryStatusDelivered
	} else if report.SuccessfulDeliveries == 0 {
		report.Status = models.DeliveryStatusFailed
	} else {
		report.Status = models.DeliveryStatusPartial
	}

	m.logger.Info().
		Str("delivery_id", req.DeliveryID).
		Str("status", string(report.Status)).
		Int("successful", report.SuccessfulDeliveries).
		Int("failed", report.FailedDeliveries).
		Int64("duration_ms", report.DurationMS).
		Msg("newsletter delivery completed")

	return report, nil
}

// deliverToRecipient handles delivery to a single recipient on a single channel.
func (m *Manager) deliverToRecipient(ctx context.Context, req *DeliveryRequest, recipient models.NewsletterRecipient, channelName models.DeliveryChannel) DeliveryResult {
	// Get channel
	channel, ok := m.registry.Get(channelName)
	if !ok {
		return DeliveryResult{
			Recipient:     recipient.Target,
			RecipientType: recipient.Type,
			ErrorMessage:  fmt.Sprintf("unknown channel: %s", channelName),
			ErrorCode:     ErrorCodeInvalidConfig,
		}
	}

	// Get channel config
	config := req.ChannelConfigs[channelName]

	// Prepare content for this channel
	bodyHTML, bodyText := FormatForChannel(channel, req.RenderedSubject, req.RenderedHTML, req.RenderedText)

	// Build send params
	params := &SendParams{
		Recipient: recipient,
		Subject:   req.RenderedSubject,
		BodyHTML:  bodyHTML,
		BodyText:  bodyText,
		Config:    config,
		Metadata:  req.Metadata,
	}

	// Attempt delivery with retries
	var lastResult *DeliveryResult
	for attempt := 0; attempt <= m.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay
			delay := m.calculateBackoff(attempt, lastResult)
			m.logger.Debug().
				Str("delivery_id", req.DeliveryID).
				Str("recipient", recipient.Target).
				Str("channel", string(channelName)).
				Int("attempt", attempt).
				Dur("delay", delay).
				Msg("retrying delivery after delay")

			select {
			case <-ctx.Done():
				return DeliveryResult{
					Recipient:     recipient.Target,
					RecipientType: recipient.Type,
					ErrorMessage:  "delivery canceled",
					ErrorCode:     ErrorCodeTimeout,
				}
			case <-time.After(delay):
			}
		}

		result, err := channel.Send(ctx, params)
		if err != nil {
			m.logger.Error().
				Err(err).
				Str("delivery_id", req.DeliveryID).
				Str("recipient", recipient.Target).
				Str("channel", string(channelName)).
				Int("attempt", attempt).
				Msg("channel send error")
			continue
		}

		lastResult = result

		if result.Success {
			m.logger.Debug().
				Str("delivery_id", req.DeliveryID).
				Str("recipient", recipient.Target).
				Str("channel", string(channelName)).
				Msg("delivery successful")
			return *result
		}

		// Check if error is transient and can be retried
		if !result.IsTransient {
			m.logger.Warn().
				Str("delivery_id", req.DeliveryID).
				Str("recipient", recipient.Target).
				Str("channel", string(channelName)).
				Str("error", result.ErrorMessage).
				Str("error_code", result.ErrorCode).
				Msg("permanent delivery error, not retrying")
			return *result
		}

		m.logger.Debug().
			Str("delivery_id", req.DeliveryID).
			Str("recipient", recipient.Target).
			Str("channel", string(channelName)).
			Str("error", result.ErrorMessage).
			Int("attempt", attempt).
			Msg("transient delivery error")
	}

	// All retries exhausted
	if lastResult != nil {
		lastResult.RetryCount = m.maxRetries
		return *lastResult
	}

	return DeliveryResult{
		Recipient:     recipient.Target,
		RecipientType: recipient.Type,
		ErrorMessage:  "delivery failed after retries",
		ErrorCode:     ErrorCodeUnknown,
		RetryCount:    m.maxRetries,
	}
}

// calculateBackoff calculates the delay before the next retry attempt.
func (m *Manager) calculateBackoff(attempt int, lastResult *DeliveryResult) time.Duration {
	// If server specified retry-after, use it
	if lastResult != nil && lastResult.RetryAfter != nil {
		return *lastResult.RetryAfter
	}

	// Exponential backoff: baseDelay * 2^attempt
	delay := m.baseDelay * (1 << uint(attempt-1))
	if delay > m.maxDelay {
		delay = m.maxDelay
	}

	return delay
}

// ValidateChannelConfigs validates all channel configurations.
func (m *Manager) ValidateChannelConfigs(channels []models.DeliveryChannel, configs map[models.DeliveryChannel]*models.ChannelConfig) error {
	for _, channelName := range channels {
		channel, ok := m.registry.Get(channelName)
		if !ok {
			return fmt.Errorf("unknown channel: %s", channelName)
		}

		config := configs[channelName]
		if err := channel.Validate(config); err != nil {
			return fmt.Errorf("invalid %s configuration: %w", channelName, err)
		}
	}
	return nil
}

// GetAvailableChannels returns all available delivery channels.
func (m *Manager) GetAvailableChannels() []models.DeliveryChannel {
	return m.registry.List()
}
