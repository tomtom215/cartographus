// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"time"
)

// WatchPartyParticipant represents a user participating in a watch party
type WatchPartyParticipant struct {
	UserID          int       `json:"user_id"`
	Username        string    `json:"username"`
	IPAddress       string    `json:"ip_address"`
	City            *string   `json:"city,omitempty"`
	Country         *string   `json:"country,omitempty"`
	StartedAt       time.Time `json:"started_at"`
	PercentComplete int       `json:"percent_complete"`
	PlayDuration    *int      `json:"play_duration_minutes,omitempty"`
}

// WatchParty represents a detected watch party (2+ users watching same content together)
type WatchParty struct {
	MediaType        string                  `json:"media_type"`
	Title            string                  `json:"title"`
	ParentTitle      *string                 `json:"parent_title,omitempty"`
	GrandparentTitle *string                 `json:"grandparent_title,omitempty"`
	PartyTime        time.Time               `json:"party_time"`
	ParticipantCount int                     `json:"participant_count"`
	Participants     []WatchPartyParticipant `json:"participants"`
	SameLocation     bool                    `json:"same_location"`
	LocationName     *string                 `json:"location_name,omitempty"`
	AvgCompletion    float64                 `json:"avg_completion"`
	TotalDuration    int                     `json:"total_duration_minutes"`
}

// WatchPartyAnalytics represents overall watch party analytics
type WatchPartyAnalytics struct {
	TotalWatchParties   int                      `json:"total_watch_parties"`
	TotalParticipants   int                      `json:"total_participants"`
	AvgParticipants     float64                  `json:"avg_participants_per_party"`
	SameLocationParties int                      `json:"same_location_parties"`
	TopContent          []WatchPartyContentStats `json:"top_content"`
	TopSocialUsers      []WatchPartyUserStats    `json:"top_social_users"`
	RecentWatchParties  []WatchParty             `json:"recent_watch_parties"`
	PartiesByDay        []WatchPartyByDay        `json:"parties_by_day"`
}

// WatchPartyContentStats represents watch party statistics for a specific piece of content
type WatchPartyContentStats struct {
	MediaType         string  `json:"media_type"`
	Title             string  `json:"title"`
	ParentTitle       *string `json:"parent_title,omitempty"`
	GrandparentTitle  *string `json:"grandparent_title,omitempty"`
	PartyCount        int     `json:"party_count"`
	TotalParticipants int     `json:"total_participants"`
	AvgParticipants   float64 `json:"avg_participants_per_party"`
	UniqueUsers       int     `json:"unique_users"`
}

// WatchPartyUserStats represents watch party statistics for a specific user
type WatchPartyUserStats struct {
	UserID            int     `json:"user_id"`
	Username          string  `json:"username"`
	PartyCount        int     `json:"party_count"`
	TotalCoWatchers   int     `json:"total_co_watchers"`
	AvgPartySize      float64 `json:"avg_party_size"`
	FavoriteContent   string  `json:"favorite_content"`
	SameLocationCount int     `json:"same_location_parties"`
}

// WatchPartyByDay represents watch party counts by day of week
type WatchPartyByDay struct {
	DayOfWeek       int     `json:"day_of_week"` // 0 = Sunday, 6 = Saturday
	PartyCount      int     `json:"party_count"`
	AvgParticipants float64 `json:"avg_participants"`
}
