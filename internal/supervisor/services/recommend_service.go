// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package services provides Suture service wrappers for various application components.
package services

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// RecommendEngine defines the interface for the recommendation engine.
// This allows the service to work with the engine without circular imports.
type RecommendEngine interface {
	// Train trains all registered algorithms.
	Train(ctx context.Context) error
}

// RecommendServiceConfig holds configuration for the recommendation service.
type RecommendServiceConfig struct {
	// TrainOnStartup triggers training when the service starts.
	TrainOnStartup bool

	// TrainInterval is how often to retrain models.
	TrainInterval time.Duration

	// MinInteractions is the minimum required before training.
	MinInteractions int
}

// RecommendService wraps the recommendation engine for Suture supervision.
// It manages the training lifecycle and periodic retraining.
type RecommendService struct {
	engine RecommendEngine
	config RecommendServiceConfig
	logger zerolog.Logger
	name   string
}

// NewRecommendService creates a new recommendation service.
//
//nolint:gocritic // logger passed by value is acceptable for zerolog
func NewRecommendService(engine RecommendEngine, cfg RecommendServiceConfig, logger zerolog.Logger) *RecommendService {
	return &RecommendService{
		engine: engine,
		config: cfg,
		logger: logger.With().Str("service", "recommend").Logger(),
		name:   "recommend-service",
	}
}

// Serve implements the suture.Service interface.
// It manages the training loop for the recommendation engine.
func (s *RecommendService) Serve(ctx context.Context) error {
	s.logger.Info().
		Bool("train_on_startup", s.config.TrainOnStartup).
		Dur("train_interval", s.config.TrainInterval).
		Msg("recommendation service starting")

	// Train on startup if configured
	if s.config.TrainOnStartup {
		s.logger.Info().Msg("training models on startup")
		if err := s.train(ctx); err != nil {
			s.logger.Warn().Err(err).Msg("initial training failed (will retry on schedule)")
		}
	}

	// Set up periodic retraining
	if s.config.TrainInterval <= 0 {
		s.config.TrainInterval = 24 * time.Hour
	}

	ticker := time.NewTicker(s.config.TrainInterval)
	defer ticker.Stop()

	s.logger.Info().Msg("recommendation service running")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("recommendation service shutting down")
			return ctx.Err()

		case <-ticker.C:
			s.logger.Debug().Msg("scheduled training triggered")
			if err := s.train(ctx); err != nil {
				s.logger.Warn().Err(err).Msg("scheduled training failed")
			}
		}
	}
}

// train performs a training cycle with proper context handling.
func (s *RecommendService) train(ctx context.Context) error {
	// Use a separate context with timeout for training
	trainCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	start := time.Now()
	s.logger.Info().Msg("starting model training")

	if err := s.engine.Train(trainCtx); err != nil {
		return err
	}

	s.logger.Info().
		Dur("duration", time.Since(start)).
		Msg("model training complete")

	return nil
}

// String returns the service name for logging.
func (s *RecommendService) String() string {
	return s.name
}
