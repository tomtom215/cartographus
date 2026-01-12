// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package newsletter provides newsletter generation and delivery functionality.
//
// content_resolver.go - Newsletter Content Resolution
//
// This file implements the content resolver for newsletters:
//   - Fetches recently added content from the database
//   - Calculates viewing statistics for the specified period
//   - Retrieves top content rankings
//   - Supports user-specific personalization
//   - Generates content recommendations
package newsletter

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/tomtom215/cartographus/internal/models"
)

// ContentStore defines the database operations required for content resolution.
type ContentStore interface {
	// Recently added content
	GetRecentlyAddedMovies(ctx context.Context, since time.Time, limit int) ([]models.NewsletterMediaItem, error)
	GetRecentlyAddedShows(ctx context.Context, since time.Time, limit int) ([]models.NewsletterShowItem, error)
	GetRecentlyAddedMusic(ctx context.Context, since time.Time, limit int) ([]models.NewsletterMediaItem, error)

	// Top content
	GetTopMovies(ctx context.Context, since time.Time, limit int) ([]models.NewsletterMediaItem, error)
	GetTopShows(ctx context.Context, since time.Time, limit int) ([]models.NewsletterShowItem, error)

	// Statistics
	GetPeriodStats(ctx context.Context, start, end time.Time) (*models.NewsletterStats, error)

	// User-specific
	GetUserStats(ctx context.Context, userID string, start, end time.Time) (*models.NewsletterUserData, error)
	GetUserRecommendations(ctx context.Context, userID string, limit int) ([]models.NewsletterMediaItem, error)

	// Server health
	GetServerHealth(ctx context.Context) (*models.NewsletterHealthData, error)
}

// ContentResolver resolves content data for newsletter templates.
type ContentResolver struct {
	store  ContentStore
	logger zerolog.Logger

	// Configuration
	serverName string
	serverURL  string
	baseURL    string
}

// ContentResolverConfig holds configuration for the content resolver.
type ContentResolverConfig struct {
	ServerName string
	ServerURL  string
	BaseURL    string
}

// NewContentResolver creates a new content resolver.
func NewContentResolver(store ContentStore, logger *zerolog.Logger, config ContentResolverConfig) *ContentResolver {
	return &ContentResolver{
		store:      store,
		logger:     logger.With().Str("component", "content_resolver").Logger(),
		serverName: config.ServerName,
		serverURL:  config.ServerURL,
		baseURL:    config.BaseURL,
	}
}

// ResolveContent resolves content for a newsletter based on type and configuration.
func (cr *ContentResolver) ResolveContent(ctx context.Context, newsletterType models.NewsletterType, config *models.TemplateConfig, userID *string) (*models.NewsletterContentData, error) {
	now := time.Now()

	// Calculate date range
	start, end := cr.calculateDateRange(now, config)

	// Initialize content data
	data := &models.NewsletterContentData{
		ServerName:       cr.serverName,
		ServerURL:        cr.serverURL,
		GeneratedAt:      now,
		DateRangeStart:   start,
		DateRangeEnd:     end,
		DateRangeDisplay: formatDateRangeDisplay(start, end),
	}

	// Resolve content based on newsletter type
	var err error
	switch newsletterType {
	case models.NewsletterTypeRecentlyAdded:
		err = cr.resolveRecentlyAdded(ctx, data, config, start)

	case models.NewsletterTypeWeeklyDigest:
		err = cr.resolveWeeklyDigest(ctx, data, config, start, end)

	case models.NewsletterTypeMonthlyStats:
		err = cr.resolveMonthlyStats(ctx, data, config, start, end)

	case models.NewsletterTypeUserActivity:
		if userID == nil {
			return nil, fmt.Errorf("user_id required for user activity newsletter")
		}
		err = cr.resolveUserActivity(ctx, data, config, *userID, start, end)

	case models.NewsletterTypeRecommendations:
		if userID == nil {
			return nil, fmt.Errorf("user_id required for recommendations newsletter")
		}
		err = cr.resolveRecommendations(ctx, data, config, *userID)

	case models.NewsletterTypeServerHealth:
		err = cr.resolveServerHealth(ctx, data)

	case models.NewsletterTypeCustom:
		// Custom newsletters may use any combination of content
		err = cr.resolveCustom(ctx, data, config, start, end, userID)

	default:
		return nil, fmt.Errorf("unsupported newsletter type: %s", newsletterType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to resolve content for %s: %w", newsletterType, err)
	}

	return data, nil
}

// calculateDateRange calculates the date range based on configuration.
func (cr *ContentResolver) calculateDateRange(now time.Time, config *models.TemplateConfig) (time.Time, time.Time) {
	end := now

	if config == nil {
		// Default: last 7 days
		return now.AddDate(0, 0, -7), end
	}

	timeFrame := config.TimeFrame
	if timeFrame <= 0 {
		timeFrame = 7
	}

	var start time.Time
	switch config.TimeFrameUnit {
	case models.TimeFrameUnitHours:
		start = now.Add(-time.Duration(timeFrame) * time.Hour)
	case models.TimeFrameUnitDays, "":
		start = now.AddDate(0, 0, -timeFrame)
	case models.TimeFrameUnitWeeks:
		start = now.AddDate(0, 0, -timeFrame*7)
	case models.TimeFrameUnitMonths:
		start = now.AddDate(0, -timeFrame, 0)
	default:
		start = now.AddDate(0, 0, -timeFrame)
	}

	return start, end
}

// resolveRecentlyAdded fetches recently added content.
func (cr *ContentResolver) resolveRecentlyAdded(ctx context.Context, data *models.NewsletterContentData, config *models.TemplateConfig, since time.Time) error {
	maxItems := getMaxItems(config, 10)

	// Fetch movies if included
	if config == nil || config.IncludeMovies {
		movies, err := cr.store.GetRecentlyAddedMovies(ctx, since, maxItems)
		if err != nil {
			cr.logger.Warn().Err(err).Msg("Failed to fetch recently added movies")
		} else {
			data.NewMovies = movies
		}
	}

	// Fetch shows if included
	if config == nil || config.IncludeShows {
		shows, err := cr.store.GetRecentlyAddedShows(ctx, since, maxItems)
		if err != nil {
			cr.logger.Warn().Err(err).Msg("Failed to fetch recently added shows")
		} else {
			data.NewShows = shows
		}
	}

	// Fetch music if included
	if config != nil && config.IncludeMusic {
		music, err := cr.store.GetRecentlyAddedMusic(ctx, since, maxItems)
		if err != nil {
			cr.logger.Warn().Err(err).Msg("Failed to fetch recently added music")
		} else {
			data.NewMusic = music
		}
	}

	return nil
}

// resolveWeeklyDigest fetches weekly digest content.
func (cr *ContentResolver) resolveWeeklyDigest(ctx context.Context, data *models.NewsletterContentData, config *models.TemplateConfig, start, end time.Time) error {
	// Get statistics
	stats, err := cr.store.GetPeriodStats(ctx, start, end)
	if err != nil {
		cr.logger.Warn().Err(err).Msg("Failed to fetch period stats")
	} else {
		data.Stats = stats
	}

	// Get recently added content
	if err := cr.resolveRecentlyAdded(ctx, data, config, start); err != nil {
		return err
	}

	// Get top content if requested
	if config == nil || config.IncludeTopContent {
		maxItems := getMaxItems(config, 5)

		topMovies, err := cr.store.GetTopMovies(ctx, start, maxItems)
		if err != nil {
			cr.logger.Warn().Err(err).Msg("Failed to fetch top movies")
		} else {
			data.TopMovies = topMovies
		}

		topShows, err := cr.store.GetTopShows(ctx, start, maxItems)
		if err != nil {
			cr.logger.Warn().Err(err).Msg("Failed to fetch top shows")
		} else {
			data.TopShows = topShows
		}
	}

	return nil
}

// resolveMonthlyStats fetches monthly statistics.
func (cr *ContentResolver) resolveMonthlyStats(ctx context.Context, data *models.NewsletterContentData, config *models.TemplateConfig, start, end time.Time) error {
	// Get comprehensive statistics
	stats, err := cr.store.GetPeriodStats(ctx, start, end)
	if err != nil {
		return fmt.Errorf("failed to fetch period stats: %w", err)
	}
	data.Stats = stats

	// Get top content
	maxItems := getMaxItems(config, 10)

	topMovies, err := cr.store.GetTopMovies(ctx, start, maxItems)
	if err != nil {
		cr.logger.Warn().Err(err).Msg("Failed to fetch top movies")
	} else {
		data.TopMovies = topMovies
	}

	topShows, err := cr.store.GetTopShows(ctx, start, maxItems)
	if err != nil {
		cr.logger.Warn().Err(err).Msg("Failed to fetch top shows")
	} else {
		data.TopShows = topShows
	}

	return nil
}

// resolveUserActivity fetches user-specific activity data.
func (cr *ContentResolver) resolveUserActivity(ctx context.Context, data *models.NewsletterContentData, config *models.TemplateConfig, userID string, start, end time.Time) error {
	// Get user stats
	userStats, err := cr.store.GetUserStats(ctx, userID, start, end)
	if err != nil {
		return fmt.Errorf("failed to fetch user stats: %w", err)
	}
	data.User = userStats

	// Get overall stats for comparison
	stats, err := cr.store.GetPeriodStats(ctx, start, end)
	if err != nil {
		cr.logger.Warn().Err(err).Msg("Failed to fetch period stats")
	} else {
		data.Stats = stats
	}

	// Get recommendations if personalization is enabled
	if config != nil && config.PersonalizeForUser {
		recommendations, err := cr.store.GetUserRecommendations(ctx, userID, getMaxItems(config, 5))
		if err != nil {
			cr.logger.Warn().Err(err).Msg("Failed to fetch recommendations")
		} else {
			data.Recommendations = recommendations
		}
	}

	return nil
}

// resolveRecommendations fetches personalized recommendations.
func (cr *ContentResolver) resolveRecommendations(ctx context.Context, data *models.NewsletterContentData, config *models.TemplateConfig, userID string) error {
	maxItems := getMaxItems(config, 10)

	recommendations, err := cr.store.GetUserRecommendations(ctx, userID, maxItems)
	if err != nil {
		return fmt.Errorf("failed to fetch recommendations: %w", err)
	}
	data.Recommendations = recommendations

	// Get user info
	userStats, err := cr.store.GetUserStats(ctx, userID, time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		cr.logger.Warn().Err(err).Msg("Failed to fetch user stats")
	} else {
		data.User = userStats
	}

	return nil
}

// resolveServerHealth fetches server health data.
func (cr *ContentResolver) resolveServerHealth(ctx context.Context, data *models.NewsletterContentData) error {
	health, err := cr.store.GetServerHealth(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch server health: %w", err)
	}
	data.Health = health
	return nil
}

// resolveCustom fetches content based on custom configuration.
func (cr *ContentResolver) resolveCustom(ctx context.Context, data *models.NewsletterContentData, config *models.TemplateConfig, start, end time.Time, userID *string) error {
	// Stats are always potentially useful for custom templates
	stats, err := cr.store.GetPeriodStats(ctx, start, end)
	if err != nil {
		cr.logger.Warn().Err(err).Msg("Failed to fetch period stats")
	} else {
		data.Stats = stats
	}

	// Resolve based on configuration flags
	if config != nil {
		if config.IncludeMovies || config.IncludeShows || config.IncludeMusic {
			if err := cr.resolveRecentlyAdded(ctx, data, config, start); err != nil {
				return err
			}
		}

		if config.IncludeTopContent {
			maxItems := getMaxItems(config, 5)

			topMovies, err := cr.store.GetTopMovies(ctx, start, maxItems)
			if err != nil {
				cr.logger.Warn().Err(err).Msg("Failed to fetch top movies")
			} else {
				data.TopMovies = topMovies
			}

			topShows, err := cr.store.GetTopShows(ctx, start, maxItems)
			if err != nil {
				cr.logger.Warn().Err(err).Msg("Failed to fetch top shows")
			} else {
				data.TopShows = topShows
			}
		}

		if config.PersonalizeForUser && userID != nil {
			userStats, err := cr.store.GetUserStats(ctx, *userID, start, end)
			if err != nil {
				cr.logger.Warn().Err(err).Msg("Failed to fetch user stats")
			} else {
				data.User = userStats
			}

			recommendations, err := cr.store.GetUserRecommendations(ctx, *userID, getMaxItems(config, 5))
			if err != nil {
				cr.logger.Warn().Err(err).Msg("Failed to fetch recommendations")
			} else {
				data.Recommendations = recommendations
			}
		}
	}

	return nil
}

// getMaxItems returns the max items from config or a default value.
func getMaxItems(config *models.TemplateConfig, defaultMax int) int {
	if config != nil && config.MaxItems > 0 {
		return config.MaxItems
	}
	return defaultMax
}

// formatDateRangeDisplay formats the date range for display.
func formatDateRangeDisplay(start, end time.Time) string {
	if start.Year() == end.Year() {
		if start.Month() == end.Month() {
			return fmt.Sprintf("%s %d - %d, %d", start.Month().String(), start.Day(), end.Day(), start.Year())
		}
		return fmt.Sprintf("%s %d - %s %d, %d", start.Month().String()[:3], start.Day(), end.Month().String()[:3], end.Day(), start.Year())
	}
	return fmt.Sprintf("%s %d, %d - %s %d, %d", start.Month().String()[:3], start.Day(), start.Year(), end.Month().String()[:3], end.Day(), end.Year())
}
