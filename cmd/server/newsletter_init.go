// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package main

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/newsletter"
	"github.com/tomtom215/cartographus/internal/newsletter/delivery"
	"github.com/tomtom215/cartographus/internal/newsletter/scheduler"
	"github.com/tomtom215/cartographus/internal/supervisor"
	"github.com/tomtom215/cartographus/internal/supervisor/services"
)

// NewsletterComponents holds all newsletter-related components.
type NewsletterComponents struct {
	Scheduler       *scheduler.Scheduler
	ContentResolver *newsletter.ContentResolver
	TemplateEngine  *newsletter.TemplateEngine
	DeliveryManager *delivery.Manager
}

// initNewsletter initializes the newsletter scheduler service if enabled.
// Returns nil if newsletters are disabled in config.
//
// The newsletter scheduler:
//   - Runs on a configurable interval (default: 1 minute)
//   - Queries for schedules that are due to execute
//   - Resolves content, renders templates, and delivers via configured channels
//   - Updates schedule status and calculates next run time
func initNewsletter(cfg *config.Config, db *database.DB, logger *zerolog.Logger, tree *supervisor.SupervisorTree) *NewsletterComponents {
	// Check if newsletter scheduler is disabled
	if !cfg.Newsletter.Enabled {
		logger.Info().Msg("Newsletter scheduler disabled (NEWSLETTER_ENABLED=false)")
		return nil
	}

	logging.Info().
		Dur("check_interval", cfg.Newsletter.CheckInterval).
		Int("max_concurrent", cfg.Newsletter.MaxConcurrentDeliveries).
		Dur("execution_timeout", cfg.Newsletter.ExecutionTimeout).
		Msg("Initializing newsletter scheduler")

	// Create content resolver for fetching newsletter content
	// The database implements the ContentStore interface
	contentResolverConfig := newsletter.ContentResolverConfig{
		ServerName: getServerName(cfg),
		ServerURL:  getServerURL(cfg),
		BaseURL:    "/",
	}
	contentResolver := newsletter.NewContentResolver(db, logger, contentResolverConfig)

	// Create template engine for rendering newsletters
	templateEngine := newsletter.NewTemplateEngine()

	// Create delivery manager for sending newsletters
	deliveryManagerConfig := delivery.ManagerConfig{
		MaxRetries:  3,
		BaseDelay:   cfg.Newsletter.CheckInterval / 10, // Retry quickly
		MaxDelay:    cfg.Newsletter.ExecutionTimeout / 3,
		Parallelism: cfg.Newsletter.MaxConcurrentDeliveries,
	}
	deliveryManager := delivery.NewManager(logger, deliveryManagerConfig)

	// Set up in-app notification store for in-app delivery channel
	// Note: This requires db to implement InAppNotificationStore
	// deliveryManager.SetInAppStore(db)

	// Create scheduler configuration
	schedulerConfig := scheduler.Config{
		CheckInterval:           cfg.Newsletter.CheckInterval,
		MaxConcurrentDeliveries: cfg.Newsletter.MaxConcurrentDeliveries,
		ExecutionTimeout:        cfg.Newsletter.ExecutionTimeout,
		Enabled:                 cfg.Newsletter.Enabled,
	}

	// Create the scheduler with all components
	sched := scheduler.NewScheduler(
		db, // implements SchedulerStore
		contentResolver,
		templateEngine,
		deliveryManager,
		logger,
		schedulerConfig,
	)

	// Create supervisor service wrapper
	service := services.NewNewsletterSchedulerService(sched)

	// Add to supervisor tree (messaging layer - handles events)
	tree.AddMessagingService(service)
	logging.Info().Msg("Newsletter scheduler service added to supervisor tree")

	return &NewsletterComponents{
		Scheduler:       sched,
		ContentResolver: contentResolver,
		TemplateEngine:  templateEngine,
		DeliveryManager: deliveryManager,
	}
}

// getServerName returns the server name for newsletters.
func getServerName(cfg *config.Config) string {
	// Use environment or a sensible default
	if cfg.Server.Host != "" && cfg.Server.Host != "0.0.0.0" {
		return cfg.Server.Host
	}
	return "Cartographus"
}

// getServerURL constructs the server URL for newsletters.
func getServerURL(cfg *config.Config) string {
	host := cfg.Server.Host
	if host == "" || host == "0.0.0.0" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%d", host, cfg.Server.Port)
}
