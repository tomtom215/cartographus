// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package reranking

import (
	"context"
	"testing"

	"github.com/tomtom215/cartographus/internal/recommend"
)

func TestNewMMR(t *testing.T) {
	tests := []struct {
		name       string
		lambda     float64
		wantLambda float64
	}{
		{"normal value", 0.7, 0.7},
		{"zero value", 0.0, 0.0},
		{"one value", 1.0, 1.0},
		{"negative clamped to zero", -0.5, 0.0},
		{"above one clamped to one", 1.5, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmr := NewMMR(tt.lambda)
			if mmr == nil {
				t.Fatal("NewMMR() returned nil")
			}
			if mmr.lambda != tt.wantLambda {
				t.Errorf("lambda = %f, want %f", mmr.lambda, tt.wantLambda)
			}
		})
	}
}

func TestMMR_Name(t *testing.T) {
	mmr := NewMMR(0.7)
	if mmr.Name() != "mmr" {
		t.Errorf("Name() = %q, want %q", mmr.Name(), "mmr")
	}
}

func TestMMR_Rerank(t *testing.T) {
	// Create items with varying genres for diversity testing
	items := []recommend.ScoredItem{
		{Item: recommend.Item{ID: 1, Genres: []string{"Action"}}, Score: 1.0},
		{Item: recommend.Item{ID: 2, Genres: []string{"Action"}}, Score: 0.9},
		{Item: recommend.Item{ID: 3, Genres: []string{"Comedy"}}, Score: 0.85},
		{Item: recommend.Item{ID: 4, Genres: []string{"Action"}}, Score: 0.8},
		{Item: recommend.Item{ID: 5, Genres: []string{"Drama"}}, Score: 0.75},
		{Item: recommend.Item{ID: 6, Genres: []string{"Comedy"}}, Score: 0.7},
	}

	tests := []struct {
		name        string
		lambda      float64
		k           int
		wantLen     int
		description string
	}{
		{
			name:        "pure relevance (lambda=1)",
			lambda:      1.0,
			k:           3,
			wantLen:     3,
			description: "returns top 3 by score",
		},
		{
			name:        "balanced (lambda=0.7)",
			lambda:      0.7,
			k:           3,
			wantLen:     3,
			description: "balances relevance and diversity",
		},
		{
			name:        "k larger than items",
			lambda:      0.7,
			k:           10,
			wantLen:     6,
			description: "returns all items when k > len",
		},
		{
			name:        "k zero returns input",
			lambda:      0.7,
			k:           0,
			wantLen:     6,
			description: "returns original list for k=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmr := NewMMR(tt.lambda)
			result := mmr.Rerank(context.Background(), items, tt.k)

			if len(result) != tt.wantLen {
				t.Errorf("len(result) = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestMMR_Rerank_DiversityEffect(t *testing.T) {
	// All high-scoring items are Action, lower-scoring are Comedy
	items := []recommend.ScoredItem{
		{Item: recommend.Item{ID: 1, Genres: []string{"Action"}}, Score: 1.0},
		{Item: recommend.Item{ID: 2, Genres: []string{"Action"}}, Score: 0.95},
		{Item: recommend.Item{ID: 3, Genres: []string{"Action"}}, Score: 0.9},
		{Item: recommend.Item{ID: 4, Genres: []string{"Comedy"}}, Score: 0.5},
		{Item: recommend.Item{ID: 5, Genres: []string{"Drama"}}, Score: 0.4},
	}

	t.Run("pure relevance keeps all Action", func(t *testing.T) {
		mmr := NewMMR(1.0)
		result := mmr.Rerank(context.Background(), items, 3)

		for _, item := range result {
			if len(item.Item.Genres) > 0 && item.Item.Genres[0] != "Action" {
				t.Errorf("pure relevance should only select Action items, got %v", item.Item.Genres)
			}
		}
	})

	t.Run("low lambda promotes diversity", func(t *testing.T) {
		mmr := NewMMR(0.3) // Strong diversity preference
		result := mmr.Rerank(context.Background(), items, 3)

		// With strong diversity, should include non-Action items
		genresSeen := make(map[string]bool)
		for _, item := range result {
			for _, g := range item.Item.Genres {
				genresSeen[g] = true
			}
		}

		if len(genresSeen) < 2 {
			t.Errorf("expected genre diversity, only saw %v", genresSeen)
		}
	})
}

func TestMMR_Rerank_EmptyInput(t *testing.T) {
	mmr := NewMMR(0.7)

	t.Run("empty items", func(t *testing.T) {
		result := mmr.Rerank(context.Background(), nil, 5)
		if len(result) != 0 {
			t.Errorf("expected empty result for empty input, got %d items", len(result))
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		result := mmr.Rerank(context.Background(), []recommend.ScoredItem{}, 5)
		if len(result) != 0 {
			t.Errorf("expected empty result for empty slice, got %d items", len(result))
		}
	})
}

func TestMMR_Rerank_SingleItem(t *testing.T) {
	mmr := NewMMR(0.7)
	items := []recommend.ScoredItem{
		{Item: recommend.Item{ID: 1, Genres: []string{"Action"}}, Score: 1.0},
	}

	result := mmr.Rerank(context.Background(), items, 5)

	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
	if result[0].Item.ID != 1 {
		t.Errorf("expected item ID 1, got %d", result[0].Item.ID)
	}
}

func TestComputeGenreSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected float64
	}{
		{
			name:     "identical genres",
			a:        []string{"Action", "Sci-Fi"},
			b:        []string{"Action", "Sci-Fi"},
			expected: 1.0,
		},
		{
			name:     "no overlap",
			a:        []string{"Action"},
			b:        []string{"Comedy"},
			expected: 0.0,
		},
		{
			name:     "partial overlap",
			a:        []string{"Action", "Sci-Fi"},
			b:        []string{"Action", "Drama"},
			expected: 1.0 / 3.0, // 1 intersection, 3 union
		},
		{
			name:     "both empty",
			a:        nil,
			b:        nil,
			expected: 0.0,
		},
		{
			name:     "one empty",
			a:        []string{"Action"},
			b:        nil,
			expected: 0.0,
		},
		{
			name:     "case insensitive",
			a:        []string{"ACTION"},
			b:        []string{"action"},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeGenreSimilarity(tt.a, tt.b)
			// Allow small floating point tolerance
			if result < tt.expected-0.01 || result > tt.expected+0.01 {
				t.Errorf("computeGenreSimilarity(%v, %v) = %f, want %f", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
