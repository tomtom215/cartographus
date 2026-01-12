// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"time"
)

// BingeSession represents a detected binge-watching session
type BingeSession struct {
	UserID           int       `json:"user_id"`
	Username         string    `json:"username"`
	ShowName         string    `json:"show_name"`
	EpisodeCount     int       `json:"episode_count"`
	FirstEpisodeTime time.Time `json:"first_episode_time"`
	LastEpisodeTime  time.Time `json:"last_episode_time"`
	TotalDuration    int       `json:"total_duration_minutes"`
	AvgCompletion    float64   `json:"avg_completion"`
}

// BingeAnalytics represents overall binge-watching analytics
type BingeAnalytics struct {
	TotalBingeSessions  int                 `json:"total_binge_sessions"`
	TotalEpisodesBinged int                 `json:"total_episodes_binged"`
	AvgEpisodesPerBinge float64             `json:"avg_episodes_per_binge"`
	AvgBingeDuration    float64             `json:"avg_binge_duration_minutes"`
	TopBingeShows       []BingeShowStats    `json:"top_binge_shows"`
	TopBingeWatchers    []BingeUserStats    `json:"top_binge_watchers"`
	RecentBingeSessions []BingeSession      `json:"recent_binge_sessions"`
	BingesByDay         []BingesByDayOfWeek `json:"binges_by_day"`
}

// BingeShowStats represents binge statistics for a specific show
type BingeShowStats struct {
	ShowName       string  `json:"show_name"`
	BingeCount     int     `json:"binge_count"`
	TotalEpisodes  int     `json:"total_episodes"`
	UniqueWatchers int     `json:"unique_watchers"`
	AvgEpisodes    float64 `json:"avg_episodes_per_binge"`
}

// BingeUserStats represents binge statistics for a specific user
type BingeUserStats struct {
	UserID        int     `json:"user_id"`
	Username      string  `json:"username"`
	BingeCount    int     `json:"binge_count"`
	TotalEpisodes int     `json:"total_episodes"`
	AvgEpisodes   float64 `json:"avg_episodes_per_binge"`
	FavoriteShow  string  `json:"favorite_show"`
}

// BingesByDayOfWeek represents binge session counts by day of week
type BingesByDayOfWeek struct {
	DayOfWeek   int `json:"day_of_week"` // 0 = Sunday, 6 = Saturday
	BingeCount  int `json:"binge_count"`
	AvgEpisodes int `json:"avg_episodes"`
}
