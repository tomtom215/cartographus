// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
analytics_advanced_charts.go - Advanced Chart Analytics Models

This file provides data models for advanced visualization charts:
  - Sankey Diagram: Content flow journeys (Show -> Season -> Episode)
  - Chord Diagram: User-user content overlap matrix
  - Radar Chart: User profile engagement dimensions
  - Treemap: Hierarchical library utilization

All models follow the production-grade standards with validation,
deterministic outputs, and comprehensive metadata.
*/

package models

// ============================================================================
// Sankey Diagram - Content Flow Journey
// ============================================================================

// SankeyNode represents a node in the Sankey diagram
type SankeyNode struct {
	ID    string `json:"id"`    // Unique identifier
	Name  string `json:"name"`  // Display name
	Depth int    `json:"depth"` // 0=Show, 1=Season, 2=Episode
	Value int64  `json:"value"` // Total watch count through this node
}

// SankeyLink represents a flow between two nodes
type SankeyLink struct {
	Source string `json:"source"` // Source node ID
	Target string `json:"target"` // Target node ID
	Value  int64  `json:"value"`  // Flow volume (watch count)
}

// ContentFlowJourney represents a single viewing journey path
type ContentFlowJourney struct {
	ShowTitle     string  `json:"showTitle"`
	SeasonNumber  int     `json:"seasonNumber"`
	EpisodeNumber int     `json:"episodeNumber"`
	EpisodeTitle  string  `json:"episodeTitle"`
	WatchCount    int64   `json:"watchCount"`
	AvgCompletion float64 `json:"avgCompletion"` // 0-100
}

// ContentFlowAnalytics is the response for content flow Sankey diagram
type ContentFlowAnalytics struct {
	Nodes       []SankeyNode         `json:"nodes"`
	Links       []SankeyLink         `json:"links"`
	Journeys    []ContentFlowJourney `json:"journeys"` // Top viewing paths
	TotalShows  int                  `json:"totalShows"`
	TotalFlows  int64                `json:"totalFlows"`  // Total viewing journeys
	DropOffRate float64              `json:"dropOffRate"` // % who start but don't finish
}

// ============================================================================
// Chord Diagram - User Content Overlap
// ============================================================================

// UserOverlapPair represents the content overlap between two users
type UserOverlapPair struct {
	User1ID         string  `json:"user1Id"`
	User1Name       string  `json:"user1Name"`
	User2ID         string  `json:"user2Id"`
	User2Name       string  `json:"user2Name"`
	SharedItems     int     `json:"sharedItems"`     // Number of items both watched
	OverlapPercent  float64 `json:"overlapPercent"`  // Jaccard similarity
	SharedWatchTime int64   `json:"sharedWatchTime"` // Total shared watch time (seconds)
	TopSharedGenre  string  `json:"topSharedGenre"`  // Most common shared genre
}

// UserOverlapMatrix represents the chord diagram data
type UserOverlapMatrix struct {
	Users  []string    `json:"users"`  // User names (matrix row/column labels)
	Matrix [][]float64 `json:"matrix"` // NxN overlap values (0-1 normalized)
}

// UserContentOverlapAnalytics is the response for user content overlap
type UserContentOverlapAnalytics struct {
	Matrix           UserOverlapMatrix `json:"matrix"`
	TopPairs         []UserOverlapPair `json:"topPairs"`   // Top overlapping user pairs
	AvgOverlap       float64           `json:"avgOverlap"` // Average overlap across all pairs
	TotalUsers       int               `json:"totalUsers"`
	TotalConnections int               `json:"totalConnections"` // Non-zero matrix cells
	ClusterCount     int               `json:"clusterCount"`     // Detected viewing clusters
}

// ============================================================================
// Radar Chart - User Profile Comparison
// ============================================================================

// UserProfileAxes defines the dimensions of user engagement
// These are the standard axes for the radar chart
var UserProfileAxes = []string{
	"Watch Time", // Total viewing hours
	"Completion", // Average completion rate
	"Diversity",  // Genre/content diversity
	"Quality",    // Average stream quality preference
	"Discovery",  // New content discovery rate
	"Social",     // Watch party participation
}

// UserProfileScore represents a user's score on the radar chart axes
type UserProfileScore struct {
	UserID   string    `json:"userId"`
	Username string    `json:"username"`
	Scores   []float64 `json:"scores"` // Values 0-100 for each axis
	Rank     int       `json:"rank"`   // Overall engagement rank
}

// UserProfileAnalytics is the response for user profile radar charts
type UserProfileAnalytics struct {
	Axes           []string           `json:"axes"`           // Dimension labels
	Profiles       []UserProfileScore `json:"profiles"`       // User scores
	AverageProfile []float64          `json:"averageProfile"` // Average scores across all users
	TopPerformers  []UserProfileScore `json:"topPerformers"`  // Users with highest overall scores
	TotalUsers     int                `json:"totalUsers"`
}

// ============================================================================
// Treemap - Library Utilization
// ============================================================================

// TreemapNode represents a node in the hierarchical treemap
type TreemapNode struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Value    int64          `json:"value"` // Watch time in seconds or play count
	Children []TreemapNode  `json:"children,omitempty"`
	ItemType string         `json:"itemType"` // "library", "section", "genre", "content"
	Metrics  TreemapMetrics `json:"metrics,omitempty"`
}

// TreemapMetrics contains additional metrics for treemap nodes
type TreemapMetrics struct {
	PlayCount      int64   `json:"playCount"`
	UniqueViewers  int     `json:"uniqueViewers"`
	AvgCompletion  float64 `json:"avgCompletion"`
	TotalDuration  int64   `json:"totalDuration"`  // Media duration in seconds
	UtilizationPct float64 `json:"utilizationPct"` // Watched/Total duration %
}

// LibraryUtilizationAnalytics is the response for library treemap
type LibraryUtilizationAnalytics struct {
	Root            TreemapNode `json:"root"`            // Hierarchical data
	TotalWatchTime  int64       `json:"totalWatchTime"`  // Total watch time seconds
	TotalContent    int         `json:"totalContent"`    // Total content items
	UtilizedContent int         `json:"utilizedContent"` // Content with at least 1 play
	UtilizationRate float64     `json:"utilizationRate"` // % of content watched
	TopUnwatched    []string    `json:"topUnwatched"`    // Content with zero plays
	MostPopularPath string      `json:"mostPopularPath"` // Most-watched hierarchy path
}

// ============================================================================
// Calendar Heatmap - Activity Calendar
// ============================================================================

// CalendarDayActivity represents activity for a single day
type CalendarDayActivity struct {
	Date      string  `json:"date"`      // ISO date (YYYY-MM-DD)
	WatchTime int64   `json:"watchTime"` // Seconds watched
	PlayCount int     `json:"playCount"` // Number of plays
	Intensity float64 `json:"intensity"` // 0-1 normalized activity level
}

// CalendarHeatmapAnalytics is the response for calendar heatmap
type CalendarHeatmapAnalytics struct {
	Days           []CalendarDayActivity `json:"days"` // 365 days of activity
	TotalWatchTime int64                 `json:"totalWatchTime"`
	TotalPlayCount int                   `json:"totalPlayCount"`
	MaxDaily       int64                 `json:"maxDaily"`      // Max watch time in a day
	AvgDaily       float64               `json:"avgDaily"`      // Average daily watch time
	ActiveDays     int                   `json:"activeDays"`    // Days with any activity
	LongestStreak  int                   `json:"longestStreak"` // Consecutive active days
	CurrentStreak  int                   `json:"currentStreak"` // Current streak
}

// ============================================================================
// Bump Chart - Ranking Changes
// ============================================================================

// RankingEntry represents a content's rank at a point in time
type RankingEntry struct {
	ContentID    string `json:"contentId"`
	ContentTitle string `json:"contentTitle"`
	Rank         int    `json:"rank"` // 1-based rank
	PlayCount    int64  `json:"playCount"`
	Period       string `json:"period"` // Week/Month identifier
}

// BumpChartAnalytics is the response for ranking bump chart
type BumpChartAnalytics struct {
	Periods      []string         `json:"periods"`    // Time period labels
	Rankings     [][]RankingEntry `json:"rankings"`   // Rankings per period
	TopMovers    []ContentMover   `json:"topMovers"`  // Biggest rank changes
	NewEntries   []RankingEntry   `json:"newEntries"` // Content that entered top 10
	Exits        []RankingEntry   `json:"exits"`      // Content that exited top 10
	TotalPeriods int              `json:"totalPeriods"`
}

// ContentMover represents content with significant rank changes
type ContentMover struct {
	ContentID    string `json:"contentId"`
	ContentTitle string `json:"contentTitle"`
	StartRank    int    `json:"startRank"`
	EndRank      int    `json:"endRank"`
	RankChange   int    `json:"rankChange"` // Positive = improved
}

// ============================================================================
// Stream Graph - Genre Stacked Time Series
// ============================================================================

// StreamGraphPoint represents a data point in the stream graph
type StreamGraphPoint struct {
	Timestamp string           `json:"timestamp"` // ISO date
	Values    map[string]int64 `json:"values"`    // Genre -> watch time
}

// StreamGraphAnalytics is the response for genre stream graph
type StreamGraphAnalytics struct {
	Points        []StreamGraphPoint `json:"points"`
	Genres        []string           `json:"genres"`        // All genres in the data
	TotalTime     int64              `json:"totalTime"`     // Total watch time
	DominantGenre string             `json:"dominantGenre"` // Most-watched genre overall
	TrendingUp    []string           `json:"trendingUp"`    // Genres with increasing share
	TrendingDown  []string           `json:"trendingDown"`  // Genres with decreasing share
}
