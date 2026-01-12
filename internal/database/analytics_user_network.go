// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains user network graph analytics for social viewing visualization.
//
// The user network graph helps understand:
//   - Who watches with whom (shared sessions, watch parties)
//   - Content taste similarity between users
//   - Social viewing patterns and communities
//   - Potential account sharing
package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// Cluster colors for visualization
var clusterColors = []string{
	"#7c3aed", // Violet
	"#3b82f6", // Blue
	"#10b981", // Emerald
	"#f59e0b", // Amber
	"#ef4444", // Red
	"#8b5cf6", // Purple
	"#06b6d4", // Cyan
	"#f97316", // Orange
	"#14b8a6", // Teal
	"#ec4899", // Pink
}

// GetUserNetworkGraph generates a user relationship network
func (db *DB) GetUserNetworkGraph(ctx context.Context, filter LocationStatsFilter, minSharedSessions int, minContentOverlap float64) (*models.UserNetworkGraph, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	startTime := time.Now()

	// Build filter conditions
	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := "1=1"
	if len(whereClauses) > 0 {
		whereClause = join(whereClauses, " AND ")
	}

	// Get user nodes
	nodes, err := db.getUserNodes(ctx, whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get user nodes: %w", err)
	}

	if len(nodes) == 0 {
		return &models.UserNetworkGraph{
			Nodes:    []models.UserNode{},
			Edges:    []models.UserEdge{},
			Clusters: []models.UserCluster{},
			Summary:  models.NetworkSummary{NetworkType: "empty"},
			Metadata: buildNetworkMetadata(filter, minSharedSessions, minContentOverlap, 0, startTime),
		}, nil
	}

	// Get edges (shared sessions)
	edges, err := db.getSharedSessionEdges(ctx, whereClause, args, minSharedSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared session edges: %w", err)
	}

	// Build node map for quick lookup
	nodeMap := make(map[string]*models.UserNode, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	// Update connection counts
	for i := range edges {
		if node, ok := nodeMap[edges[i].Source]; ok {
			node.ConnectionCount++
		}
		if node, ok := nodeMap[edges[i].Target]; ok {
			node.ConnectionCount++
		}
	}

	// Detect clusters using simple connected components
	clusters := detectClusters(nodes, edges)

	// Update nodes with cluster info
	for i := range nodes {
		if cluster, ok := findClusterForNode(nodes[i].ID, clusters); ok {
			nodes[i].ClusterID = cluster.ID
			nodes[i].NodeColor = cluster.ClusterColor
		}
	}

	// Calculate summary
	summary := calculateNetworkSummary(nodes, edges, clusters)

	// Get event count
	eventCount := getEventCount(ctx, db, whereClause, args)

	return &models.UserNetworkGraph{
		Nodes:    nodes,
		Edges:    edges,
		Clusters: clusters,
		Summary:  summary,
		Metadata: buildNetworkMetadata(filter, minSharedSessions, minContentOverlap, eventCount, startTime),
	}, nil
}

// getUserNodes retrieves all users as nodes
func (db *DB) getUserNodes(ctx context.Context, whereClause string, args []interface{}) ([]models.UserNode, error) {
	query := fmt.Sprintf(`
		SELECT
			CAST(user_id AS VARCHAR) AS id,
			username,
			COUNT(*) AS playback_count,
			SUM(COALESCE(play_duration, 0)) / 60.0 AS total_watch_hours,
			MAX(started_at) AS last_active
		FROM playback_events
		WHERE %s
		GROUP BY user_id, username
		HAVING COUNT(*) >= 3
		ORDER BY playback_count DESC
		LIMIT 100
	`, whereClause)

	var nodes []models.UserNode
	err := db.queryAndScan(ctx, query, args, func(rows *sql.Rows) error {
		var node models.UserNode
		if err := rows.Scan(
			&node.ID,
			&node.Username,
			&node.PlaybackCount,
			&node.TotalWatchHours,
			&node.LastActive,
		); err != nil {
			return err
		}

		// Calculate node size based on activity (logarithmic scale)
		node.NodeSize = 10 + (float64(node.PlaybackCount) / 10.0)
		if node.NodeSize > 50 {
			node.NodeSize = 50
		}

		// Default color (will be updated by cluster assignment)
		node.NodeColor = "#6b7280"

		nodes = append(nodes, node)
		return nil
	})

	return nodes, err
}

// getSharedSessionEdges finds users who watched together
func (db *DB) getSharedSessionEdges(ctx context.Context, whereClause string, args []interface{}, minSharedSessions int) ([]models.UserEdge, error) {
	// Find users who watched the same content within 5 minutes of each other
	// This indicates potential shared viewing or watch party
	query := fmt.Sprintf(`
		WITH user_sessions AS (
			SELECT
				CAST(user_id AS VARCHAR) AS user_id,
				username,
				rating_key,
				title,
				grandparent_title,
				started_at,
				ip_address
			FROM playback_events
			WHERE %s
				AND rating_key IS NOT NULL
		),
		shared_views AS (
			SELECT
				a.user_id AS user1,
				b.user_id AS user2,
				a.username AS username1,
				b.username AS username2,
				COALESCE(a.grandparent_title, a.title) AS content,
				CASE
					WHEN a.ip_address = b.ip_address THEN 'shared_session'
					ELSE 'watch_party'
				END AS connection_type,
				MIN(a.started_at) AS first_interaction,
				MAX(a.started_at) AS last_interaction
			FROM user_sessions a
			JOIN user_sessions b
				ON a.rating_key = b.rating_key
				AND a.user_id < b.user_id
				AND ABS(EPOCH(a.started_at) - EPOCH(b.started_at)) <= 300
			GROUP BY a.user_id, b.user_id, a.username, b.username, content, connection_type
		)
		SELECT
			user1,
			user2,
			username1,
			username2,
			connection_type,
			COUNT(*) AS shared_sessions,
			ARRAY_AGG(DISTINCT content) AS shared_content,
			MIN(first_interaction) AS first_interaction,
			MAX(last_interaction) AS last_interaction
		FROM shared_views
		GROUP BY user1, user2, username1, username2, connection_type
		HAVING COUNT(*) >= ?
		ORDER BY shared_sessions DESC
		LIMIT 500
	`, whereClause)

	fullArgs := append(args, minSharedSessions)

	var edges []models.UserEdge
	err := db.queryAndScan(ctx, query, fullArgs, func(rows *sql.Rows) error {
		var edge models.UserEdge
		var username1, username2 string
		var sharedContent []string

		if err := rows.Scan(
			&edge.Source,
			&edge.Target,
			&username1,
			&username2,
			&edge.ConnectionType,
			&edge.SharedSessions,
			&sharedContent,
			&edge.FirstInteraction,
			&edge.LastInteraction,
		); err != nil {
			return err
		}

		// Limit shared content to top 5
		if len(sharedContent) > 5 {
			edge.TopSharedContent = sharedContent[:5]
		} else {
			edge.TopSharedContent = sharedContent
		}

		// Calculate weight (logarithmic to avoid extreme values)
		edge.Weight = float64(edge.SharedSessions)
		edge.EdgeWidth = 1 + (float64(edge.SharedSessions) / 5.0)
		if edge.EdgeWidth > 10 {
			edge.EdgeWidth = 10
		}

		edges = append(edges, edge)
		return nil
	})

	return edges, err
}

// detectClusters performs simple connected component clustering
func detectClusters(nodes []models.UserNode, edges []models.UserEdge) []models.UserCluster {
	if len(nodes) == 0 {
		return []models.UserCluster{}
	}

	// Build adjacency map
	adj := make(map[string][]string)
	for i := range nodes {
		adj[nodes[i].ID] = []string{}
	}
	for i := range edges {
		adj[edges[i].Source] = append(adj[edges[i].Source], edges[i].Target)
		adj[edges[i].Target] = append(adj[edges[i].Target], edges[i].Source)
	}

	// Find connected components using DFS
	visited := make(map[string]bool)
	var clusters []models.UserCluster
	clusterID := 0

	for i := range nodes {
		if visited[nodes[i].ID] {
			continue
		}

		// DFS to find all nodes in this component
		var component []string
		stack := []string{nodes[i].ID}

		for len(stack) > 0 {
			curr := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			if visited[curr] {
				continue
			}
			visited[curr] = true
			component = append(component, curr)

			for _, neighbor := range adj[curr] {
				if !visited[neighbor] {
					stack = append(stack, neighbor)
				}
			}
		}

		// Create cluster
		cluster := models.UserCluster{
			ID:           clusterID,
			Name:         fmt.Sprintf("Cluster %d", clusterID+1),
			UserCount:    len(component),
			UserIDs:      component,
			ClusterColor: clusterColors[clusterID%len(clusterColors)],
		}

		// Calculate density
		possibleEdges := len(component) * (len(component) - 1) / 2
		if possibleEdges > 0 {
			actualEdges := 0
			for j := range edges {
				if containsString(component, edges[j].Source) && containsString(component, edges[j].Target) {
					actualEdges++
				}
			}
			cluster.Density = float64(actualEdges) / float64(possibleEdges)
		}

		// Assign characteristic type
		cluster.CharacteristicType = getClusterType(&cluster)

		clusters = append(clusters, cluster)
		clusterID++
	}

	// Sort by size descending
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].UserCount > clusters[j].UserCount
	})

	return clusters
}

// findClusterForNode finds which cluster a node belongs to
func findClusterForNode(nodeID string, clusters []models.UserCluster) (models.UserCluster, bool) {
	for i := range clusters {
		for _, id := range clusters[i].UserIDs {
			if id == nodeID {
				return clusters[i], true
			}
		}
	}
	return models.UserCluster{}, false
}

// calculateNetworkSummary computes aggregate network statistics
func calculateNetworkSummary(nodes []models.UserNode, edges []models.UserEdge, clusters []models.UserCluster) models.NetworkSummary {
	summary := models.NetworkSummary{
		TotalUsers:       len(nodes),
		TotalConnections: len(edges),
		TotalClusters:    len(clusters),
	}

	// Set largest cluster size (even for empty networks)
	if len(clusters) > 0 {
		summary.LargestClusterSize = clusters[0].UserCount
	}

	if len(nodes) == 0 {
		summary.NetworkType = "empty"
		return summary
	}

	// Calculate network density
	possibleEdges := len(nodes) * (len(nodes) - 1) / 2
	if possibleEdges > 0 {
		summary.NetworkDensity = float64(len(edges)) / float64(possibleEdges)
	}

	// Calculate average connections
	totalConnections := 0
	maxConnections := 0
	var maxConnectionsUser string
	isolatedCount := 0

	for i := range nodes {
		totalConnections += nodes[i].ConnectionCount
		if nodes[i].ConnectionCount > maxConnections {
			maxConnections = nodes[i].ConnectionCount
			maxConnectionsUser = nodes[i].Username
		}
		if nodes[i].ConnectionCount == 0 {
			isolatedCount++
		}
	}

	summary.AvgConnectionsPerUser = float64(totalConnections) / float64(len(nodes))
	summary.MaxConnectionsUser = maxConnectionsUser
	summary.MaxConnectionsCount = maxConnections
	summary.IsolatedUsers = isolatedCount

	// Count shared sessions and watch parties
	for i := range edges {
		if edges[i].ConnectionType == "watch_party" {
			summary.WatchPartyCount += edges[i].SharedSessions
		} else {
			summary.SharedSessionCount += edges[i].SharedSessions
		}
	}

	// Determine network type
	summary.NetworkType = determineNetworkType(&summary, clusters)

	return summary
}

// determineNetworkType classifies the network structure
func determineNetworkType(summary *models.NetworkSummary, clusters []models.UserCluster) string {
	if summary.TotalConnections == 0 {
		return "fragmented"
	}

	// Check if hub-spoke (one node with many more connections than others)
	if float64(summary.MaxConnectionsCount) > summary.AvgConnectionsPerUser*3 {
		return "centralized"
	}

	// Check if fragmented (many small clusters)
	if len(clusters) > summary.TotalUsers/3 {
		return "fragmented"
	}

	// Check if well-connected
	if summary.NetworkDensity > 0.3 {
		return "distributed"
	}

	return "hierarchical"
}

// getClusterType determines the characteristic of a cluster
func getClusterType(cluster *models.UserCluster) string {
	if cluster.UserCount == 1 {
		return "Isolated"
	}
	if cluster.UserCount == 2 {
		return "Pair"
	}
	if cluster.Density > 0.8 {
		return "Tight-knit Group"
	}
	if cluster.UserCount > 10 {
		return "Large Community"
	}
	return "Social Group"
}

// getEventCount retrieves total event count for metadata
func getEventCount(ctx context.Context, db *DB, whereClause string, args []interface{}) int64 {
	query := fmt.Sprintf("SELECT COUNT(*) FROM playback_events WHERE %s", whereClause)
	var count int64
	if err := db.conn.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0
	}
	return count
}

// buildNetworkMetadata creates metadata for the response
func buildNetworkMetadata(filter LocationStatsFilter, minSharedSessions int, minContentOverlap float64, eventCount int64, startTime time.Time) models.NetworkMetadata {
	dataRangeStart, dataRangeEnd := getDataRange(filter)

	canonical := fmt.Sprintf("user_network|min_sessions=%d|min_overlap=%.2f|",
		minSharedSessions, minContentOverlap)
	if filter.StartDate != nil {
		canonical += fmt.Sprintf("start=%s|", filter.StartDate.Format(time.RFC3339))
	}
	if filter.EndDate != nil {
		canonical += fmt.Sprintf("end=%s|", filter.EndDate.Format(time.RFC3339))
	}
	hash := sha256.Sum256([]byte(canonical))

	return models.NetworkMetadata{
		QueryHash:           hex.EncodeToString(hash[:8]),
		DataRangeStart:      dataRangeStart,
		DataRangeEnd:        dataRangeEnd,
		MinSharedSessions:   minSharedSessions,
		MinContentOverlap:   minContentOverlap,
		ClusteringAlgorithm: "connected_components",
		EventCount:          eventCount,
		GeneratedAt:         time.Now(),
		QueryTimeMs:         time.Since(startTime).Milliseconds(),
		Cached:              false,
	}
}

// containsString checks if a slice contains a string
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
