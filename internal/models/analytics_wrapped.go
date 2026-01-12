// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models provides data structures for the Cartographus application.
// This file contains models for Annual Wrapped Reports - Spotify-style yearly analytics.
package models

import (
	"time"
)

// WrappedReport represents a complete annual wrapped report for a user.
// This is the main response structure containing all yearly statistics and insights.
//
// The report is designed to be cached after generation and includes a share token
// for creating shareable links.
type WrappedReport struct {
	// Identification
	ID          string    `json:"id"`
	Year        int       `json:"year"`
	UserID      int       `json:"user_id"`
	Username    string    `json:"username"`
	GeneratedAt time.Time `json:"generated_at"`
	ShareToken  string    `json:"share_token,omitempty"`

	// Core Statistics
	TotalWatchTimeHours  float64 `json:"total_watch_time_hours"`
	TotalPlaybacks       int     `json:"total_playbacks"`
	UniqueContentCount   int     `json:"unique_content_count"`
	CompletionRate       float64 `json:"completion_rate"`         // 0-100 percentage
	DaysActive           int     `json:"days_active"`             // Days with at least one playback
	LongestStreakDays    int     `json:"longest_streak_days"`     // Consecutive days with playbacks
	AvgDailyWatchMinutes float64 `json:"avg_daily_watch_minutes"` // Average minutes on active days

	// Content Breakdown
	TopMovies    []WrappedContentRank `json:"top_movies"`
	TopShows     []WrappedContentRank `json:"top_shows"`
	TopEpisodes  []WrappedContentRank `json:"top_episodes,omitempty"`
	TopGenres    []WrappedGenreRank   `json:"top_genres"`
	TopActors    []WrappedPersonRank  `json:"top_actors,omitempty"`
	TopDirectors []WrappedPersonRank  `json:"top_directors,omitempty"`

	// Viewing Patterns
	ViewingByHour  [24]int          `json:"viewing_by_hour"`  // Playback count per hour (0-23)
	ViewingByDay   [7]int           `json:"viewing_by_day"`   // Playback count per day (0=Sunday)
	ViewingByMonth [12]int          `json:"viewing_by_month"` // Playback count per month (0=January)
	PeakHour       int              `json:"peak_hour"`        // Hour with most playbacks
	PeakDay        string           `json:"peak_day"`         // Day name with most playbacks
	PeakMonth      string           `json:"peak_month"`       // Month name with most playbacks
	MonthlyTrends  []WrappedMonthly `json:"monthly_trends"`

	// Binge Analysis
	BingeSessions     int               `json:"binge_sessions"`
	LongestBinge      *WrappedBingeInfo `json:"longest_binge,omitempty"`
	TotalBingeHours   float64           `json:"total_binge_hours"`
	FavoriteBingeShow string            `json:"favorite_binge_show,omitempty"`
	AvgBingeEpisodes  float64           `json:"avg_binge_episodes"`

	// Quality Metrics
	AvgBitrateMbps      float64 `json:"avg_bitrate_mbps"`
	DirectPlayRate      float64 `json:"direct_play_rate"`    // 0-100 percentage
	HDRViewingPercent   float64 `json:"hdr_viewing_percent"` // 0-100 percentage
	FourKViewingPercent float64 `json:"4k_viewing_percent"`  // 0-100 percentage
	PreferredPlatform   string  `json:"preferred_platform"`
	PreferredPlayer     string  `json:"preferred_player"`

	// Discovery Metrics
	NewContentCount  int     `json:"new_content_count"`   // First-time watches
	DiscoveryRate    float64 `json:"discovery_rate"`      // New content / total plays
	FirstWatchOfYear string  `json:"first_watch_of_year"` // First content watched in the year
	LastWatchOfYear  string  `json:"last_watch_of_year"`  // Last content watched in the year

	// Achievements
	Achievements []WrappedAchievement `json:"achievements"`
	Percentiles  WrappedPercentiles   `json:"percentiles"`

	// Shareable Summary
	ShareableText string `json:"shareable_text"`
}

// WrappedContentRank represents a ranked content item in the wrapped report.
type WrappedContentRank struct {
	Rank       int     `json:"rank"`
	Title      string  `json:"title"`
	RatingKey  string  `json:"rating_key,omitempty"`
	Thumb      string  `json:"thumb,omitempty"`
	WatchCount int     `json:"watch_count"`
	WatchTime  float64 `json:"watch_time_hours"`
	MediaType  string  `json:"media_type"` // movie, show, episode
}

// WrappedGenreRank represents a ranked genre in the wrapped report.
type WrappedGenreRank struct {
	Rank       int     `json:"rank"`
	Genre      string  `json:"genre"`
	WatchCount int     `json:"watch_count"`
	WatchTime  float64 `json:"watch_time_hours"`
	Percentage float64 `json:"percentage"` // 0-100 of total watch time
}

// WrappedPersonRank represents a ranked actor/director in the wrapped report.
type WrappedPersonRank struct {
	Rank       int     `json:"rank"`
	Name       string  `json:"name"`
	WatchCount int     `json:"watch_count"`
	WatchTime  float64 `json:"watch_time_hours"`
}

// WrappedMonthly represents monthly viewing statistics.
type WrappedMonthly struct {
	Month          int     `json:"month"` // 1-12
	MonthName      string  `json:"month_name"`
	PlaybackCount  int     `json:"playback_count"`
	WatchTimeHours float64 `json:"watch_time_hours"`
	UniqueContent  int     `json:"unique_content"`
	TopContent     string  `json:"top_content,omitempty"` // Most watched content that month
}

// WrappedBingeInfo represents information about a binge-watching session.
type WrappedBingeInfo struct {
	ShowName      string    `json:"show_name"`
	EpisodeCount  int       `json:"episode_count"`
	DurationHours float64   `json:"duration_hours"`
	Date          time.Time `json:"date"`
}

// WrappedAchievement represents a user achievement/badge in the wrapped report.
type WrappedAchievement struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Tier        string `json:"tier,omitempty"` // bronze, silver, gold, platinum
	EarnedAt    string `json:"earned_at,omitempty"`
}

// WrappedPercentiles represents how the user compares to other users.
type WrappedPercentiles struct {
	WatchTime      int `json:"watch_time"`      // Percentile for total watch time (0-100)
	ContentCount   int `json:"content_count"`   // Percentile for unique content
	BingeCount     int `json:"binge_count"`     // Percentile for binge sessions
	CompletionRate int `json:"completion_rate"` // Percentile for completion rate
	EarlyAdopter   int `json:"early_adopter"`   // Percentile for watching new content quickly
}

// WrappedSummary is a lightweight summary for the wrapped reports list.
type WrappedSummary struct {
	Year                int       `json:"year"`
	UserID              int       `json:"user_id"`
	Username            string    `json:"username"`
	TotalWatchTimeHours float64   `json:"total_watch_time_hours"`
	TotalPlaybacks      int       `json:"total_playbacks"`
	TopGenre            string    `json:"top_genre"`
	GeneratedAt         time.Time `json:"generated_at"`
}

// WrappedLeaderboardEntry represents an entry in the wrapped leaderboard.
type WrappedLeaderboardEntry struct {
	Rank                int     `json:"rank"`
	UserID              int     `json:"user_id"`
	Username            string  `json:"username"`
	TotalWatchTimeHours float64 `json:"total_watch_time_hours"`
	TotalPlaybacks      int     `json:"total_playbacks"`
	UniqueContent       int     `json:"unique_content"`
	CompletionRate      float64 `json:"completion_rate"`
}

// WrappedServerStats represents server-wide wrapped statistics.
type WrappedServerStats struct {
	Year                 int       `json:"year"`
	TotalUsers           int       `json:"total_users"`
	TotalWatchTimeHours  float64   `json:"total_watch_time_hours"`
	TotalPlaybacks       int       `json:"total_playbacks"`
	UniqueContentWatched int       `json:"unique_content_watched"`
	TopMovie             string    `json:"top_movie"`
	TopShow              string    `json:"top_show"`
	TopGenre             string    `json:"top_genre"`
	PeakMonth            string    `json:"peak_month"`
	AvgCompletionRate    float64   `json:"avg_completion_rate"`
	GeneratedAt          time.Time `json:"generated_at"`
}

// WrappedGenerateRequest represents a request to generate wrapped reports.
type WrappedGenerateRequest struct {
	Year   int  `json:"year" validate:"required,min=2000,max=2100"`
	UserID *int `json:"user_id,omitempty"` // If nil, generate for all users
	Force  bool `json:"force"`             // Regenerate even if exists
}

// WrappedGenerateResponse represents the response after generating wrapped reports.
type WrappedGenerateResponse struct {
	Year             int       `json:"year"`
	ReportsGenerated int       `json:"reports_generated"`
	DurationMS       int64     `json:"duration_ms"`
	GeneratedAt      time.Time `json:"generated_at"`
}

// DayNames maps day of week integers to day names.
var DayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// MonthNames maps month integers (1-12) to month names.
var MonthNames = []string{"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

// Achievement IDs for the wrapped report.
const (
	AchievementBingemaster       = "bingemaster"
	AchievementNightOwl          = "night_owl"
	AchievementEarlyBird         = "early_bird"
	AchievementWeekendWarrior    = "weekend_warrior"
	AchievementMovieBuff         = "movie_buff"
	AchievementSeriesAddict      = "series_addict"
	AchievementQualityEnthusiast = "quality_enthusiast"
	AchievementExplorer          = "explorer"
	AchievementMarathoner        = "marathoner"
	AchievementConsistent        = "consistent"
)

// DefaultAchievements returns the list of possible achievements for wrapped reports.
func DefaultAchievements() []WrappedAchievement {
	return []WrappedAchievement{
		{ID: AchievementBingemaster, Name: "Bingemaster", Description: "Completed 10+ binge sessions", Icon: "tv"},
		{ID: AchievementNightOwl, Name: "Night Owl", Description: "50%+ of viewing after 10 PM", Icon: "moon"},
		{ID: AchievementEarlyBird, Name: "Early Bird", Description: "50%+ of viewing before 9 AM", Icon: "sun"},
		{ID: AchievementWeekendWarrior, Name: "Weekend Warrior", Description: "60%+ of viewing on weekends", Icon: "calendar"},
		{ID: AchievementMovieBuff, Name: "Movie Buff", Description: "Watched 50+ unique movies", Icon: "film"},
		{ID: AchievementSeriesAddict, Name: "Series Addict", Description: "Watched 10+ complete series", Icon: "layers"},
		{ID: AchievementQualityEnthusiast, Name: "Quality Enthusiast", Description: "80%+ direct play rate", Icon: "award"},
		{ID: AchievementExplorer, Name: "Explorer", Description: "Watched content in 10+ genres", Icon: "compass"},
		{ID: AchievementMarathoner, Name: "Marathoner", Description: "500+ hours watched in a year", Icon: "clock"},
		{ID: AchievementConsistent, Name: "Consistent", Description: "30+ day viewing streak", Icon: "trending-up"},
	}
}
