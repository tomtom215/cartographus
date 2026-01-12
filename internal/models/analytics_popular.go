// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"time"
)

// PopularContent represents a single piece of popular content with statistics
type PopularContent struct {
	MediaType        string    `json:"media_type"`
	Title            string    `json:"title"`
	ParentTitle      *string   `json:"parent_title,omitempty"`
	GrandparentTitle *string   `json:"grandparent_title,omitempty"`
	PlaybackCount    int       `json:"playback_count"`
	UniqueUsers      int       `json:"unique_users"`
	AvgCompletion    float64   `json:"avg_completion"`
	FirstPlayed      time.Time `json:"first_played"`
	LastPlayed       time.Time `json:"last_played"`
	Year             *int      `json:"year,omitempty"`
	ContentRating    *string   `json:"content_rating,omitempty"`
	TotalWatchTime   int       `json:"total_watch_time_minutes"`
}

// PopularAnalytics represents popular content analytics
type PopularAnalytics struct {
	TopMovies   []PopularContent `json:"top_movies"`
	TopShows    []PopularContent `json:"top_shows"`
	TopEpisodes []PopularContent `json:"top_episodes"`
}
