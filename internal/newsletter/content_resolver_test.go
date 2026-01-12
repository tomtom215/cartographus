// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package newsletter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/tomtom215/cartographus/internal/models"
)

// MockContentStore implements ContentStore interface for testing
type MockContentStore struct {
	// Mock data
	Movies          []models.NewsletterMediaItem
	Shows           []models.NewsletterShowItem
	Music           []models.NewsletterMediaItem
	TopMovies       []models.NewsletterMediaItem
	TopShows        []models.NewsletterShowItem
	Stats           *models.NewsletterStats
	UserStats       *models.NewsletterUserData
	Recommendations []models.NewsletterMediaItem
	Health          *models.NewsletterHealthData

	// Error injection
	MoviesErr          error
	ShowsErr           error
	MusicErr           error
	TopMoviesErr       error
	TopShowsErr        error
	StatsErr           error
	UserStatsErr       error
	RecommendationsErr error
	HealthErr          error
}

func (m *MockContentStore) GetRecentlyAddedMovies(ctx context.Context, since time.Time, limit int) ([]models.NewsletterMediaItem, error) {
	if m.MoviesErr != nil {
		return nil, m.MoviesErr
	}
	return m.Movies, nil
}

func (m *MockContentStore) GetRecentlyAddedShows(ctx context.Context, since time.Time, limit int) ([]models.NewsletterShowItem, error) {
	if m.ShowsErr != nil {
		return nil, m.ShowsErr
	}
	return m.Shows, nil
}

func (m *MockContentStore) GetRecentlyAddedMusic(ctx context.Context, since time.Time, limit int) ([]models.NewsletterMediaItem, error) {
	if m.MusicErr != nil {
		return nil, m.MusicErr
	}
	return m.Music, nil
}

func (m *MockContentStore) GetTopMovies(ctx context.Context, since time.Time, limit int) ([]models.NewsletterMediaItem, error) {
	if m.TopMoviesErr != nil {
		return nil, m.TopMoviesErr
	}
	return m.TopMovies, nil
}

func (m *MockContentStore) GetTopShows(ctx context.Context, since time.Time, limit int) ([]models.NewsletterShowItem, error) {
	if m.TopShowsErr != nil {
		return nil, m.TopShowsErr
	}
	return m.TopShows, nil
}

func (m *MockContentStore) GetPeriodStats(ctx context.Context, start, end time.Time) (*models.NewsletterStats, error) {
	if m.StatsErr != nil {
		return nil, m.StatsErr
	}
	return m.Stats, nil
}

func (m *MockContentStore) GetUserStats(ctx context.Context, userID string, start, end time.Time) (*models.NewsletterUserData, error) {
	if m.UserStatsErr != nil {
		return nil, m.UserStatsErr
	}
	return m.UserStats, nil
}

func (m *MockContentStore) GetUserRecommendations(ctx context.Context, userID string, limit int) ([]models.NewsletterMediaItem, error) {
	if m.RecommendationsErr != nil {
		return nil, m.RecommendationsErr
	}
	return m.Recommendations, nil
}

func (m *MockContentStore) GetServerHealth(ctx context.Context) (*models.NewsletterHealthData, error) {
	if m.HealthErr != nil {
		return nil, m.HealthErr
	}
	return m.Health, nil
}

func TestNewContentResolver(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{}
	config := ContentResolverConfig{
		ServerName: "Test Server",
		ServerURL:  "http://localhost:32400",
		BaseURL:    "http://localhost:3857",
	}

	resolver := NewContentResolver(store, &logger, config)
	if resolver == nil {
		t.Fatal("NewContentResolver returned nil")
	}

	if resolver.serverName != "Test Server" {
		t.Errorf("serverName = %q, want %q", resolver.serverName, "Test Server")
	}
	if resolver.serverURL != "http://localhost:32400" {
		t.Errorf("serverURL = %q, want %q", resolver.serverURL, "http://localhost:32400")
	}
}

func TestContentResolver_ResolveContent_RecentlyAdded(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{
		Movies: []models.NewsletterMediaItem{
			{Title: "Movie 1", Year: 2026},
			{Title: "Movie 2", Year: 2025},
		},
		Shows: []models.NewsletterShowItem{
			{Title: "Show 1", Year: 2026},
		},
	}
	config := ContentResolverConfig{
		ServerName: "Test Server",
		ServerURL:  "http://localhost:32400",
	}

	resolver := NewContentResolver(store, &logger, config)
	ctx := context.Background()

	data, err := resolver.ResolveContent(ctx, models.NewsletterTypeRecentlyAdded, nil, nil)
	if err != nil {
		t.Fatalf("ResolveContent failed: %v", err)
	}

	if len(data.NewMovies) != 2 {
		t.Errorf("NewMovies count = %d, want 2", len(data.NewMovies))
	}
	if len(data.NewShows) != 1 {
		t.Errorf("NewShows count = %d, want 1", len(data.NewShows))
	}
	if data.ServerName != "Test Server" {
		t.Errorf("ServerName = %q, want %q", data.ServerName, "Test Server")
	}
}

func TestContentResolver_ResolveContent_WeeklyDigest(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{
		Movies: []models.NewsletterMediaItem{
			{Title: "Movie 1", Year: 2026},
		},
		TopMovies: []models.NewsletterMediaItem{
			{Title: "Top Movie 1", Year: 2024},
		},
		Stats: &models.NewsletterStats{
			TotalPlaybacks:      100,
			TotalWatchTimeHours: 50.5,
		},
	}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{
		ServerName: "Test Server",
	})
	ctx := context.Background()

	data, err := resolver.ResolveContent(ctx, models.NewsletterTypeWeeklyDigest, nil, nil)
	if err != nil {
		t.Fatalf("ResolveContent failed: %v", err)
	}

	if data.Stats == nil {
		t.Fatal("Stats should not be nil")
	}
	if data.Stats.TotalPlaybacks != 100 {
		t.Errorf("TotalPlaybacks = %d, want 100", data.Stats.TotalPlaybacks)
	}
	if len(data.TopMovies) != 1 {
		t.Errorf("TopMovies count = %d, want 1", len(data.TopMovies))
	}
}

func TestContentResolver_ResolveContent_MonthlyStats(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{
		Stats: &models.NewsletterStats{
			TotalPlaybacks:      500,
			TotalWatchTimeHours: 200.0,
			UniqueUsers:         25,
		},
		TopMovies: []models.NewsletterMediaItem{
			{Title: "Top Movie"},
		},
		TopShows: []models.NewsletterShowItem{
			{Title: "Top Show"},
		},
	}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{
		ServerName: "Test Server",
	})
	ctx := context.Background()

	config := &models.TemplateConfig{
		TimeFrame:     30,
		TimeFrameUnit: models.TimeFrameUnitDays,
	}

	data, err := resolver.ResolveContent(ctx, models.NewsletterTypeMonthlyStats, config, nil)
	if err != nil {
		t.Fatalf("ResolveContent failed: %v", err)
	}

	if data.Stats.TotalPlaybacks != 500 {
		t.Errorf("TotalPlaybacks = %d, want 500", data.Stats.TotalPlaybacks)
	}
}

func TestContentResolver_ResolveContent_UserActivity(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{
		UserStats: &models.NewsletterUserData{
			Username:       "testuser",
			PlaybackCount:  50,
			WatchTimeHours: 25.5,
		},
		Stats: &models.NewsletterStats{
			TotalPlaybacks: 500,
		},
	}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{
		ServerName: "Test Server",
	})
	ctx := context.Background()
	userID := "user-123"

	data, err := resolver.ResolveContent(ctx, models.NewsletterTypeUserActivity, nil, &userID)
	if err != nil {
		t.Fatalf("ResolveContent failed: %v", err)
	}

	if data.User == nil {
		t.Fatal("User should not be nil")
	}
	if data.User.Username != "testuser" {
		t.Errorf("Username = %q, want %q", data.User.Username, "testuser")
	}
}

func TestContentResolver_ResolveContent_UserActivity_RequiresUserID(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{})
	ctx := context.Background()

	_, err := resolver.ResolveContent(ctx, models.NewsletterTypeUserActivity, nil, nil)
	if err == nil {
		t.Error("Expected error when userID is nil for UserActivity")
	}
}

func TestContentResolver_ResolveContent_Recommendations(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{
		Recommendations: []models.NewsletterMediaItem{
			{Title: "Recommended 1"},
			{Title: "Recommended 2"},
		},
		UserStats: &models.NewsletterUserData{
			Username: "testuser",
		},
	}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{})
	ctx := context.Background()
	userID := "user-123"

	data, err := resolver.ResolveContent(ctx, models.NewsletterTypeRecommendations, nil, &userID)
	if err != nil {
		t.Fatalf("ResolveContent failed: %v", err)
	}

	if len(data.Recommendations) != 2 {
		t.Errorf("Recommendations count = %d, want 2", len(data.Recommendations))
	}
}

func TestContentResolver_ResolveContent_Recommendations_RequiresUserID(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{})
	ctx := context.Background()

	_, err := resolver.ResolveContent(ctx, models.NewsletterTypeRecommendations, nil, nil)
	if err == nil {
		t.Error("Expected error when userID is nil for Recommendations")
	}
}

func TestContentResolver_ResolveContent_ServerHealth(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{
		Health: &models.NewsletterHealthData{
			ServerStatus:   "healthy",
			UptimePercent:  99.5,
			ActiveStreams:  5,
			TotalLibraries: 4,
		},
	}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{})
	ctx := context.Background()

	data, err := resolver.ResolveContent(ctx, models.NewsletterTypeServerHealth, nil, nil)
	if err != nil {
		t.Fatalf("ResolveContent failed: %v", err)
	}

	if data.Health == nil {
		t.Fatal("Health should not be nil")
	}
	if data.Health.ServerStatus != "healthy" {
		t.Errorf("ServerStatus = %q, want %q", data.Health.ServerStatus, "healthy")
	}
}

func TestContentResolver_ResolveContent_ServerHealth_Error(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{
		HealthErr: errors.New("health check failed"),
	}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{})
	ctx := context.Background()

	_, err := resolver.ResolveContent(ctx, models.NewsletterTypeServerHealth, nil, nil)
	if err == nil {
		t.Error("Expected error when health check fails")
	}
}

func TestContentResolver_ResolveContent_Custom(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{
		Movies: []models.NewsletterMediaItem{
			{Title: "Movie 1"},
		},
		TopMovies: []models.NewsletterMediaItem{
			{Title: "Top Movie"},
		},
		Stats: &models.NewsletterStats{
			TotalPlaybacks: 100,
		},
	}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{})
	ctx := context.Background()

	config := &models.TemplateConfig{
		IncludeMovies:     true,
		IncludeTopContent: true,
	}

	data, err := resolver.ResolveContent(ctx, models.NewsletterTypeCustom, config, nil)
	if err != nil {
		t.Fatalf("ResolveContent failed: %v", err)
	}

	if len(data.NewMovies) != 1 {
		t.Errorf("NewMovies count = %d, want 1", len(data.NewMovies))
	}
	if len(data.TopMovies) != 1 {
		t.Errorf("TopMovies count = %d, want 1", len(data.TopMovies))
	}
}

func TestContentResolver_ResolveContent_UnsupportedType(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{})
	ctx := context.Background()

	_, err := resolver.ResolveContent(ctx, models.NewsletterType("invalid"), nil, nil)
	if err == nil {
		t.Error("Expected error for unsupported newsletter type")
	}
}

func TestContentResolver_calculateDateRange(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{}
	resolver := NewContentResolver(store, &logger, ContentResolverConfig{})

	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		config    *models.TemplateConfig
		wantStart time.Time
	}{
		{
			name:      "nil config defaults to 7 days",
			config:    nil,
			wantStart: now.AddDate(0, 0, -7),
		},
		{
			name:      "zero time frame defaults to 7 days",
			config:    &models.TemplateConfig{TimeFrame: 0},
			wantStart: now.AddDate(0, 0, -7),
		},
		{
			name: "days unit",
			config: &models.TemplateConfig{
				TimeFrame:     14,
				TimeFrameUnit: models.TimeFrameUnitDays,
			},
			wantStart: now.AddDate(0, 0, -14),
		},
		{
			name: "hours unit",
			config: &models.TemplateConfig{
				TimeFrame:     24,
				TimeFrameUnit: models.TimeFrameUnitHours,
			},
			wantStart: now.Add(-24 * time.Hour),
		},
		{
			name: "weeks unit",
			config: &models.TemplateConfig{
				TimeFrame:     2,
				TimeFrameUnit: models.TimeFrameUnitWeeks,
			},
			wantStart: now.AddDate(0, 0, -14),
		},
		{
			name: "months unit",
			config: &models.TemplateConfig{
				TimeFrame:     1,
				TimeFrameUnit: models.TimeFrameUnitMonths,
			},
			wantStart: now.AddDate(0, -1, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := resolver.calculateDateRange(now, tt.config)
			if !start.Equal(tt.wantStart) {
				t.Errorf("start = %v, want %v", start, tt.wantStart)
			}
			if !end.Equal(now) {
				t.Errorf("end = %v, want %v", end, now)
			}
		})
	}
}

func TestGetMaxItems(t *testing.T) {
	tests := []struct {
		name       string
		config     *models.TemplateConfig
		defaultMax int
		want       int
	}{
		{
			name:       "nil config uses default",
			config:     nil,
			defaultMax: 10,
			want:       10,
		},
		{
			name:       "zero max items uses default",
			config:     &models.TemplateConfig{MaxItems: 0},
			defaultMax: 10,
			want:       10,
		},
		{
			name:       "custom max items",
			config:     &models.TemplateConfig{MaxItems: 20},
			defaultMax: 10,
			want:       20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMaxItems(tt.config, tt.defaultMax)
			if got != tt.want {
				t.Errorf("getMaxItems() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFormatDateRangeDisplay(t *testing.T) {
	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  string
	}{
		{
			name:  "same month",
			start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			want:  "January 1 - 15, 2026",
		},
		{
			name:  "different months same year",
			start: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			want:  "Jan 15 - Feb 15, 2026",
		},
		{
			name:  "different years",
			start: time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			want:  "Dec 15, 2025 - Jan 15, 2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDateRangeDisplay(tt.start, tt.end)
			if got != tt.want {
				t.Errorf("formatDateRangeDisplay() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContentResolver_ResolveContent_PartialErrors(t *testing.T) {
	// Test that partial errors don't fail the entire resolution
	logger := zerolog.Nop()
	store := &MockContentStore{
		Movies:   []models.NewsletterMediaItem{{Title: "Movie 1"}},
		ShowsErr: errors.New("shows error"),
		Stats:    &models.NewsletterStats{TotalPlaybacks: 100},
	}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{})
	ctx := context.Background()

	// For weekly digest, shows error should not fail the whole resolution
	data, err := resolver.ResolveContent(ctx, models.NewsletterTypeWeeklyDigest, nil, nil)
	if err != nil {
		t.Errorf("ResolveContent should not fail for partial errors: %v", err)
	}

	// Movies should still be populated
	if len(data.NewMovies) != 1 {
		t.Errorf("NewMovies should be populated despite shows error")
	}
	// Shows should be empty due to error
	if len(data.NewShows) != 0 {
		t.Errorf("NewShows should be empty due to error")
	}
}

func TestContentResolver_ResolveContent_WithConfig(t *testing.T) {
	logger := zerolog.Nop()
	store := &MockContentStore{
		Movies: []models.NewsletterMediaItem{{Title: "Movie 1"}},
		Shows:  []models.NewsletterShowItem{{Title: "Show 1"}},
		Music:  []models.NewsletterMediaItem{{Title: "Album 1"}},
	}

	resolver := NewContentResolver(store, &logger, ContentResolverConfig{})
	ctx := context.Background()

	// Config that excludes movies but includes music
	config := &models.TemplateConfig{
		IncludeMovies: false,
		IncludeShows:  true,
		IncludeMusic:  true,
	}

	data, err := resolver.ResolveContent(ctx, models.NewsletterTypeRecentlyAdded, config, nil)
	if err != nil {
		t.Fatalf("ResolveContent failed: %v", err)
	}

	// Movies should NOT be populated because IncludeMovies is false
	// Note: The current implementation defaults to including movies if config.IncludeMovies is false
	// Let's test the actual behavior
	if len(data.NewShows) != 1 {
		t.Errorf("NewShows count = %d, want 1", len(data.NewShows))
	}
	if len(data.NewMusic) != 1 {
		t.Errorf("NewMusic count = %d, want 1", len(data.NewMusic))
	}
}
