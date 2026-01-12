// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

// ContentAbandonmentAnalytics represents drop-off and completion rate analytics
type ContentAbandonmentAnalytics struct {
	Summary                 ContentAbandonmentSummary `json:"summary"`
	TopAbandoned            []AbandonedContent        `json:"top_abandoned"`            // Highest abandonment rates
	CompletionByMediaType   []MediaTypeCompletion     `json:"completion_by_media_type"` // Movie vs TV vs Music
	DropOffDistribution     []DropOffBucket           `json:"drop_off_distribution"`    // 0-25%, 25-50%, 50-75%, 75-100%
	AbandonmentByGenre      []GenreAbandonment        `json:"abandonment_by_genre,omitempty"`
	FirstEpisodeAbandonment []FirstEpisodeDropOff     `json:"first_episode_abandonment,omitempty"` // Shows abandoned after pilot
}

type ContentAbandonmentSummary struct {
	TotalPlaybacks     int     `json:"total_playbacks"`
	CompletedPlaybacks int     `json:"completed_playbacks"`   // >= 90% completion
	AbandonedPlaybacks int     `json:"abandoned_playbacks"`   // < 90% completion
	AverageCompletion  float64 `json:"average_completion"`    // Mean percent_complete
	MedianDropOffPoint float64 `json:"median_drop_off_point"` // Median abandonment point
	CompletionRate     float64 `json:"completion_rate"`       // % that reached 90%+
}

type AbandonedContent struct {
	Title              string  `json:"title"`
	MediaType          string  `json:"media_type"`
	GrandparentTitle   string  `json:"grandparent_title,omitempty"` // For episodes
	TotalStarts        int     `json:"total_starts"`
	Completions        int     `json:"completions"`           // >= 90%
	Abandonments       int     `json:"abandonments"`          // < 90%
	CompletionRate     float64 `json:"completion_rate"`       // %
	AverageCompletion  float64 `json:"average_completion"`    // Mean %
	MedianDropOffPoint float64 `json:"median_drop_off_point"` // Median abandonment %
	Genres             string  `json:"genres,omitempty"`
}

type MediaTypeCompletion struct {
	MediaType          string  `json:"media_type"`
	TotalPlaybacks     int     `json:"total_playbacks"`
	CompletedPlaybacks int     `json:"completed_playbacks"`
	CompletionRate     float64 `json:"completion_rate"`
	AverageCompletion  float64 `json:"average_completion"`
}

type DropOffBucket struct {
	Bucket            string  `json:"bucket"` // "0-25%", "25-50%", "50-75%", "75-90%", "90-100%"
	MinPercent        int     `json:"min_percent"`
	MaxPercent        int     `json:"max_percent"`
	PlaybackCount     int     `json:"playback_count"`
	PercentageOfTotal float64 `json:"percentage_of_total"`
}

type GenreAbandonment struct {
	Genre              string  `json:"genre"`
	TotalPlaybacks     int     `json:"total_playbacks"`
	CompletedPlaybacks int     `json:"completed_playbacks"`
	CompletionRate     float64 `json:"completion_rate"`
	AverageCompletion  float64 `json:"average_completion"`
}

type FirstEpisodeDropOff struct {
	ShowName            string  `json:"show_name"`
	FirstEpisodeTitle   string  `json:"first_episode_title,omitempty"`
	PilotStarts         int     `json:"pilot_starts"`
	PilotCompletions    int     `json:"pilot_completions"`
	SeriesContinuations int     `json:"series_continuations"` // Users who watched episode 2+
	DropOffRate         float64 `json:"drop_off_rate"`        // % who didn't continue
}
