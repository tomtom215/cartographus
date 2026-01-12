// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models provides data structures for the Cartographus application.
// This file contains user network graph models for social viewing visualization.
//
// The user network graph visualizes relationships between users based on:
// - Shared viewing sessions (watching same content at similar times)
// - Content overlap (similar taste profiles)
// - Watch party participation
//
// This enables insights into:
// - Social viewing patterns
// - Account sharing detection
// - User clusters/communities
// - Content recommendation opportunities
package models

import "time"

// UserNetworkGraph represents the complete user relationship network
type UserNetworkGraph struct {
	// Nodes represent individual users
	Nodes []UserNode `json:"nodes"`

	// Edges represent relationships between users
	Edges []UserEdge `json:"edges"`

	// Clusters represent detected user communities
	Clusters []UserCluster `json:"clusters"`

	// Summary provides aggregate network statistics
	Summary NetworkSummary `json:"summary"`

	// Metadata provides query provenance
	Metadata NetworkMetadata `json:"metadata"`
}

// UserNode represents a user in the network graph
type UserNode struct {
	// ID is the unique user identifier
	ID string `json:"id"`

	// Username for display
	Username string `json:"username"`

	// PlaybackCount is total playbacks by this user
	PlaybackCount int `json:"playback_count"`

	// ConnectionCount is the number of edges connected to this node
	ConnectionCount int `json:"connection_count"`

	// ClusterID is the detected community/cluster this user belongs to
	ClusterID int `json:"cluster_id"`

	// TotalWatchHours is cumulative watch time in hours
	TotalWatchHours float64 `json:"total_watch_hours"`

	// TopGenre is this user's most-watched genre
	TopGenre string `json:"top_genre,omitempty"`

	// LastActive is when this user last played content
	LastActive time.Time `json:"last_active"`

	// IsCentral indicates if this user is a hub in the network
	IsCentral bool `json:"is_central"`

	// NodeSize is a computed size based on activity (for visualization)
	NodeSize float64 `json:"node_size"`

	// NodeColor is the assigned color based on cluster (hex format)
	NodeColor string `json:"node_color"`
}

// UserEdge represents a relationship between two users
type UserEdge struct {
	// Source is the ID of the first user
	Source string `json:"source"`

	// Target is the ID of the second user
	Target string `json:"target"`

	// ConnectionType categorizes the relationship
	// Types: "shared_session" (same IP, same time), "content_overlap" (similar taste),
	//        "watch_party" (synchronized viewing), "sequential" (back-to-back same content)
	ConnectionType string `json:"connection_type"`

	// Weight represents the strength of the connection (higher = stronger)
	Weight float64 `json:"weight"`

	// SharedSessions is the count of co-viewing events (if type is shared_session/watch_party)
	SharedSessions int `json:"shared_sessions,omitempty"`

	// ContentOverlap is the Jaccard similarity of content watched (0-1)
	ContentOverlap float64 `json:"content_overlap,omitempty"`

	// TopSharedContent lists titles commonly watched by both users
	TopSharedContent []string `json:"top_shared_content,omitempty"`

	// FirstInteraction is when these users first had a connection
	FirstInteraction time.Time `json:"first_interaction,omitempty"`

	// LastInteraction is the most recent connection event
	LastInteraction time.Time `json:"last_interaction,omitempty"`

	// EdgeWidth is computed for visualization (based on weight)
	EdgeWidth float64 `json:"edge_width"`
}

// UserCluster represents a detected community of users
type UserCluster struct {
	// ID is the cluster identifier
	ID int `json:"id"`

	// Name is an auto-generated or user-defined cluster name
	Name string `json:"name"`

	// UserCount is the number of users in this cluster
	UserCount int `json:"user_count"`

	// UserIDs lists all user IDs in this cluster
	UserIDs []string `json:"user_ids"`

	// Density is the internal connectivity (edges / possible edges)
	Density float64 `json:"density"`

	// TopContent is the most popular content within this cluster
	TopContent []string `json:"top_content,omitempty"`

	// TopGenres are the most popular genres within this cluster
	TopGenres []string `json:"top_genres,omitempty"`

	// AvgWatchHours is the average watch time per user in this cluster
	AvgWatchHours float64 `json:"avg_watch_hours"`

	// ClusterColor is the assigned color for visualization (hex)
	ClusterColor string `json:"cluster_color"`

	// CharacteristicType describes the cluster (e.g., "Family", "Power Users", "Casual")
	CharacteristicType string `json:"characteristic_type"`
}

// NetworkSummary provides aggregate statistics about the network
type NetworkSummary struct {
	// TotalUsers is the count of users (nodes) in the network
	TotalUsers int `json:"total_users"`

	// TotalConnections is the count of relationships (edges)
	TotalConnections int `json:"total_connections"`

	// TotalClusters is the number of detected communities
	TotalClusters int `json:"total_clusters"`

	// NetworkDensity is total edges / possible edges
	// Density close to 1 means highly connected, close to 0 means sparse
	NetworkDensity float64 `json:"network_density"`

	// AvgConnectionsPerUser is mean degree centrality
	AvgConnectionsPerUser float64 `json:"avg_connections_per_user"`

	// MaxConnectionsUser is the most connected user
	MaxConnectionsUser string `json:"max_connections_user"`

	// MaxConnectionsCount is their connection count
	MaxConnectionsCount int `json:"max_connections_count"`

	// IsolatedUsers is the count of users with no connections
	IsolatedUsers int `json:"isolated_users"`

	// SharedSessionCount is total co-viewing events detected
	SharedSessionCount int `json:"shared_session_count"`

	// WatchPartyCount is detected watch party events
	WatchPartyCount int `json:"watch_party_count"`

	// LargestClusterSize is the user count in the biggest cluster
	LargestClusterSize int `json:"largest_cluster_size"`

	// NetworkType describes the overall structure
	// Types: "fragmented" (many small clusters), "centralized" (hub-spoke),
	//        "distributed" (even connections), "hierarchical" (nested clusters)
	NetworkType string `json:"network_type"`
}

// NetworkMetadata provides provenance information
type NetworkMetadata struct {
	// QueryHash for reproducibility
	QueryHash string `json:"query_hash"`

	// DataRangeStart is the earliest data analyzed
	DataRangeStart time.Time `json:"data_range_start"`

	// DataRangeEnd is the latest data analyzed
	DataRangeEnd time.Time `json:"data_range_end"`

	// MinSharedSessions is the threshold for creating an edge
	MinSharedSessions int `json:"min_shared_sessions"`

	// MinContentOverlap is the threshold for content similarity edges
	MinContentOverlap float64 `json:"min_content_overlap"`

	// ClusteringAlgorithm used (e.g., "louvain", "label_propagation")
	ClusteringAlgorithm string `json:"clustering_algorithm"`

	// EventCount is total events analyzed
	EventCount int64 `json:"event_count"`

	// GeneratedAt is when this was generated
	GeneratedAt time.Time `json:"generated_at"`

	// QueryTimeMs is execution time
	QueryTimeMs int64 `json:"query_time_ms"`

	// Cached indicates if from cache
	Cached bool `json:"cached"`
}
