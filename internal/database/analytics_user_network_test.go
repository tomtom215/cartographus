// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

func TestContainsString(t *testing.T) {

	tests := []struct {
		name     string
		slice    []string
		s        string
		expected bool
	}{
		{"empty slice returns false", []string{}, "test", false},
		{"finds existing element", []string{"a", "b", "c"}, "b", true},
		{"returns false for missing element", []string{"a", "b", "c"}, "d", false},
		{"single element found", []string{"test"}, "test", true},
		{"single element not found", []string{"test"}, "other", false},
		{"case sensitive", []string{"Test"}, "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsString(tt.slice, tt.s)
			if result != tt.expected {
				t.Errorf("containsString(%v, %q) = %v, want %v", tt.slice, tt.s, result, tt.expected)
			}
		})
	}
}

func TestGetClusterType(t *testing.T) {

	tests := []struct {
		name     string
		cluster  models.UserCluster
		expected string
	}{
		{
			name:     "isolated single user",
			cluster:  models.UserCluster{UserCount: 1},
			expected: "Isolated",
		},
		{
			name:     "pair of users",
			cluster:  models.UserCluster{UserCount: 2},
			expected: "Pair",
		},
		{
			name:     "tight-knit group with high density",
			cluster:  models.UserCluster{UserCount: 5, Density: 0.9},
			expected: "Tight-knit Group",
		},
		{
			name:     "large community",
			cluster:  models.UserCluster{UserCount: 15, Density: 0.3},
			expected: "Large Community",
		},
		{
			name:     "social group",
			cluster:  models.UserCluster{UserCount: 5, Density: 0.5},
			expected: "Social Group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getClusterType(&tt.cluster)
			if result != tt.expected {
				t.Errorf("getClusterType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetermineNetworkType(t *testing.T) {

	t.Run("fragmented when no connections", func(t *testing.T) {
		summary := models.NetworkSummary{TotalConnections: 0}
		clusters := []models.UserCluster{}

		result := determineNetworkType(&summary, clusters)

		if result != "fragmented" {
			t.Errorf("expected 'fragmented', got '%s'", result)
		}
	})

	t.Run("centralized when hub node exists", func(t *testing.T) {
		summary := models.NetworkSummary{
			TotalConnections:      10,
			MaxConnectionsCount:   20,
			AvgConnectionsPerUser: 2.0, // Max is 10x average
		}
		clusters := []models.UserCluster{}

		result := determineNetworkType(&summary, clusters)

		if result != "centralized" {
			t.Errorf("expected 'centralized', got '%s'", result)
		}
	})

	t.Run("fragmented when many small clusters", func(t *testing.T) {
		summary := models.NetworkSummary{
			TotalConnections:      5,
			TotalUsers:            15,
			MaxConnectionsCount:   3,
			AvgConnectionsPerUser: 2.0,
		}
		// 6 clusters for 15 users = many small groups
		clusters := make([]models.UserCluster, 6)

		result := determineNetworkType(&summary, clusters)

		if result != "fragmented" {
			t.Errorf("expected 'fragmented', got '%s'", result)
		}
	})

	t.Run("distributed when well-connected", func(t *testing.T) {
		summary := models.NetworkSummary{
			TotalConnections:      10,
			TotalUsers:            10,
			NetworkDensity:        0.5, // High density
			MaxConnectionsCount:   4,
			AvgConnectionsPerUser: 3.0,
		}
		clusters := []models.UserCluster{{UserCount: 10}}

		result := determineNetworkType(&summary, clusters)

		if result != "distributed" {
			t.Errorf("expected 'distributed', got '%s'", result)
		}
	})

	t.Run("hierarchical as default", func(t *testing.T) {
		summary := models.NetworkSummary{
			TotalConnections:      10,
			TotalUsers:            20,
			NetworkDensity:        0.1, // Low density
			MaxConnectionsCount:   5,
			AvgConnectionsPerUser: 3.0,
		}
		clusters := []models.UserCluster{{UserCount: 10}, {UserCount: 10}}

		result := determineNetworkType(&summary, clusters)

		if result != "hierarchical" {
			t.Errorf("expected 'hierarchical', got '%s'", result)
		}
	})
}

func TestFindClusterForNode(t *testing.T) {

	clusters := []models.UserCluster{
		{ID: 0, UserIDs: []string{"user1", "user2", "user3"}},
		{ID: 1, UserIDs: []string{"user4", "user5"}},
		{ID: 2, UserIDs: []string{"user6"}},
	}

	t.Run("finds node in first cluster", func(t *testing.T) {
		cluster, found := findClusterForNode("user2", clusters)

		if !found {
			t.Error("expected to find node in cluster")
		}
		if cluster.ID != 0 {
			t.Errorf("expected cluster ID 0, got %d", cluster.ID)
		}
	})

	t.Run("finds node in second cluster", func(t *testing.T) {
		cluster, found := findClusterForNode("user4", clusters)

		if !found {
			t.Error("expected to find node in cluster")
		}
		if cluster.ID != 1 {
			t.Errorf("expected cluster ID 1, got %d", cluster.ID)
		}
	})

	t.Run("returns false for missing node", func(t *testing.T) {
		_, found := findClusterForNode("user99", clusters)

		if found {
			t.Error("expected not to find missing node")
		}
	})

	t.Run("handles empty clusters", func(t *testing.T) {
		_, found := findClusterForNode("user1", []models.UserCluster{})

		if found {
			t.Error("expected not to find node in empty clusters")
		}
	})
}

func TestDetectClusters(t *testing.T) {

	t.Run("empty nodes returns empty clusters", func(t *testing.T) {
		clusters := detectClusters([]models.UserNode{}, []models.UserEdge{})

		if len(clusters) != 0 {
			t.Errorf("expected 0 clusters, got %d", len(clusters))
		}
	})

	t.Run("disconnected nodes become separate clusters", func(t *testing.T) {
		nodes := []models.UserNode{
			{ID: "user1", Username: "Alice"},
			{ID: "user2", Username: "Bob"},
			{ID: "user3", Username: "Charlie"},
		}
		edges := []models.UserEdge{} // No connections

		clusters := detectClusters(nodes, edges)

		if len(clusters) != 3 {
			t.Errorf("expected 3 separate clusters, got %d", len(clusters))
		}
		for _, cluster := range clusters {
			if cluster.UserCount != 1 {
				t.Errorf("expected 1 user per cluster, got %d", cluster.UserCount)
			}
		}
	})

	t.Run("connected nodes form single cluster", func(t *testing.T) {
		nodes := []models.UserNode{
			{ID: "user1", Username: "Alice"},
			{ID: "user2", Username: "Bob"},
			{ID: "user3", Username: "Charlie"},
		}
		edges := []models.UserEdge{
			{Source: "user1", Target: "user2"},
			{Source: "user2", Target: "user3"},
		}

		clusters := detectClusters(nodes, edges)

		if len(clusters) != 1 {
			t.Errorf("expected 1 cluster, got %d", len(clusters))
		}
		if clusters[0].UserCount != 3 {
			t.Errorf("expected 3 users in cluster, got %d", clusters[0].UserCount)
		}
	})

	t.Run("finds multiple connected components", func(t *testing.T) {
		nodes := []models.UserNode{
			{ID: "user1", Username: "Alice"},
			{ID: "user2", Username: "Bob"},
			{ID: "user3", Username: "Charlie"},
			{ID: "user4", Username: "Dave"},
		}
		edges := []models.UserEdge{
			{Source: "user1", Target: "user2"}, // Cluster 1
			{Source: "user3", Target: "user4"}, // Cluster 2
		}

		clusters := detectClusters(nodes, edges)

		if len(clusters) != 2 {
			t.Errorf("expected 2 clusters, got %d", len(clusters))
		}
		// Should be sorted by size
		for _, cluster := range clusters {
			if cluster.UserCount != 2 {
				t.Errorf("expected 2 users per cluster, got %d", cluster.UserCount)
			}
		}
	})

	t.Run("assigns cluster colors", func(t *testing.T) {
		nodes := []models.UserNode{
			{ID: "user1"},
			{ID: "user2"},
		}
		edges := []models.UserEdge{}

		clusters := detectClusters(nodes, edges)

		for _, cluster := range clusters {
			if cluster.ClusterColor == "" {
				t.Error("cluster should have a color assigned")
			}
			if cluster.ClusterColor[0] != '#' {
				t.Errorf("cluster color should be hex format, got %s", cluster.ClusterColor)
			}
		}
	})

	t.Run("calculates cluster density", func(t *testing.T) {
		// Triangle: 3 nodes, 3 edges = full connectivity
		nodes := []models.UserNode{
			{ID: "user1"},
			{ID: "user2"},
			{ID: "user3"},
		}
		edges := []models.UserEdge{
			{Source: "user1", Target: "user2"},
			{Source: "user2", Target: "user3"},
			{Source: "user1", Target: "user3"},
		}

		clusters := detectClusters(nodes, edges)

		if len(clusters) != 1 {
			t.Fatalf("expected 1 cluster, got %d", len(clusters))
		}
		// 3 nodes can have max 3 edges (3*2/2=3), all 3 present = density 1.0
		if clusters[0].Density != 1.0 {
			t.Errorf("expected density 1.0 for fully connected triangle, got %.2f", clusters[0].Density)
		}
	})
}

func TestCalculateNetworkSummary(t *testing.T) {

	t.Run("empty network", func(t *testing.T) {
		summary := calculateNetworkSummary([]models.UserNode{}, []models.UserEdge{}, []models.UserCluster{})

		if summary.TotalUsers != 0 {
			t.Errorf("expected 0 users, got %d", summary.TotalUsers)
		}
		if summary.NetworkType != "empty" {
			t.Errorf("expected 'empty' network type, got '%s'", summary.NetworkType)
		}
	})

	t.Run("calculates network density", func(t *testing.T) {
		nodes := []models.UserNode{
			{ID: "user1"},
			{ID: "user2"},
			{ID: "user3"},
		}
		edges := []models.UserEdge{
			{Source: "user1", Target: "user2"},
			{Source: "user2", Target: "user3"},
		}
		clusters := []models.UserCluster{}

		summary := calculateNetworkSummary(nodes, edges, clusters)

		// 3 nodes = max 3 edges, we have 2 = density 0.667
		expectedDensity := 2.0 / 3.0
		if summary.NetworkDensity < expectedDensity-0.01 || summary.NetworkDensity > expectedDensity+0.01 {
			t.Errorf("expected density ~%.3f, got %.3f", expectedDensity, summary.NetworkDensity)
		}
	})

	t.Run("identifies hub user", func(t *testing.T) {
		nodes := []models.UserNode{
			{ID: "user1", Username: "Alice", ConnectionCount: 5},
			{ID: "user2", Username: "Bob", ConnectionCount: 2},
			{ID: "user3", Username: "Charlie", ConnectionCount: 1},
		}
		edges := []models.UserEdge{}
		clusters := []models.UserCluster{}

		summary := calculateNetworkSummary(nodes, edges, clusters)

		if summary.MaxConnectionsUser != "Alice" {
			t.Errorf("expected max connections user 'Alice', got '%s'", summary.MaxConnectionsUser)
		}
		if summary.MaxConnectionsCount != 5 {
			t.Errorf("expected max connections 5, got %d", summary.MaxConnectionsCount)
		}
	})

	t.Run("counts isolated users", func(t *testing.T) {
		nodes := []models.UserNode{
			{ID: "user1", ConnectionCount: 5},
			{ID: "user2", ConnectionCount: 0}, // Isolated
			{ID: "user3", ConnectionCount: 0}, // Isolated
		}
		edges := []models.UserEdge{}
		clusters := []models.UserCluster{}

		summary := calculateNetworkSummary(nodes, edges, clusters)

		if summary.IsolatedUsers != 2 {
			t.Errorf("expected 2 isolated users, got %d", summary.IsolatedUsers)
		}
	})

	t.Run("counts shared sessions and watch parties", func(t *testing.T) {
		nodes := []models.UserNode{{ID: "user1"}, {ID: "user2"}}
		edges := []models.UserEdge{
			{Source: "user1", Target: "user2", ConnectionType: "shared_session", SharedSessions: 5},
			{Source: "user1", Target: "user2", ConnectionType: "watch_party", SharedSessions: 3},
		}
		clusters := []models.UserCluster{}

		summary := calculateNetworkSummary(nodes, edges, clusters)

		if summary.SharedSessionCount != 5 {
			t.Errorf("expected 5 shared sessions, got %d", summary.SharedSessionCount)
		}
		if summary.WatchPartyCount != 3 {
			t.Errorf("expected 3 watch parties, got %d", summary.WatchPartyCount)
		}
	})

	t.Run("identifies largest cluster", func(t *testing.T) {
		nodes := []models.UserNode{}
		edges := []models.UserEdge{}
		clusters := []models.UserCluster{
			{ID: 0, UserCount: 10}, // Largest first due to sorting
			{ID: 1, UserCount: 5},
		}

		summary := calculateNetworkSummary(nodes, edges, clusters)

		if summary.LargestClusterSize != 10 {
			t.Errorf("expected largest cluster size 10, got %d", summary.LargestClusterSize)
		}
	})
}

func TestBuildNetworkMetadata(t *testing.T) {

	t.Run("includes all parameters", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		filter := LocationStatsFilter{StartDate: &startDate}
		startTime := time.Now()

		metadata := buildNetworkMetadata(filter, 3, 0.5, 1000, startTime)

		if metadata.MinSharedSessions != 3 {
			t.Errorf("expected MinSharedSessions=3, got %d", metadata.MinSharedSessions)
		}
		if metadata.MinContentOverlap != 0.5 {
			t.Errorf("expected MinContentOverlap=0.5, got %f", metadata.MinContentOverlap)
		}
		if metadata.EventCount != 1000 {
			t.Errorf("expected EventCount=1000, got %d", metadata.EventCount)
		}
		if metadata.ClusteringAlgorithm != "connected_components" {
			t.Errorf("expected ClusteringAlgorithm='connected_components', got '%s'", metadata.ClusteringAlgorithm)
		}
	})

	t.Run("generates consistent query hash", func(t *testing.T) {
		startDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		filter := LocationStatsFilter{StartDate: &startDate}
		startTime := time.Now()

		metadata1 := buildNetworkMetadata(filter, 3, 0.5, 1000, startTime)
		metadata2 := buildNetworkMetadata(filter, 3, 0.5, 1000, startTime)

		if metadata1.QueryHash != metadata2.QueryHash {
			t.Errorf("same inputs should produce same hash: %s != %s", metadata1.QueryHash, metadata2.QueryHash)
		}
	})

	t.Run("different parameters produce different hashes", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		filter := LocationStatsFilter{StartDate: &startDate}
		startTime := time.Now()

		metadata1 := buildNetworkMetadata(filter, 3, 0.5, 1000, startTime)
		metadata2 := buildNetworkMetadata(filter, 5, 0.5, 1000, startTime)

		if metadata1.QueryHash == metadata2.QueryHash {
			t.Error("different parameters should produce different hashes")
		}
	})

	t.Run("hash is 16 characters", func(t *testing.T) {
		filter := LocationStatsFilter{}
		startTime := time.Now()

		metadata := buildNetworkMetadata(filter, 3, 0.5, 1000, startTime)

		if len(metadata.QueryHash) != 16 {
			t.Errorf("expected 16 char hash, got %d", len(metadata.QueryHash))
		}
	})
}

func TestClusterColors(t *testing.T) {

	// Verify cluster colors are valid hex codes
	for i, color := range clusterColors {
		if len(color) != 7 {
			t.Errorf("color %d should be 7 chars (including #), got %d", i, len(color))
		}
		if color[0] != '#' {
			t.Errorf("color %d should start with #, got %c", i, color[0])
		}
	}

	// Should have at least 10 colors
	if len(clusterColors) < 10 {
		t.Errorf("expected at least 10 cluster colors, got %d", len(clusterColors))
	}
}

func TestNetworkHashDeterminism(t *testing.T) {

	startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	filter := LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	}
	startTime := time.Now()

	firstMetadata := buildNetworkMetadata(filter, 5, 0.7, 5000, startTime)

	// Generate 100 times and verify consistency
	for i := 0; i < 100; i++ {
		metadata := buildNetworkMetadata(filter, 5, 0.7, 5000, startTime)
		if metadata.QueryHash != firstMetadata.QueryHash {
			t.Errorf("iteration %d produced different hash: %s != %s", i, metadata.QueryHash, firstMetadata.QueryHash)
		}
	}
}
