// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package reranking implements post-processing algorithms for recommendation diversity.
package reranking

import (
	"context"
	"strings"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// maxRerankSize limits slice allocations to prevent excessive memory usage.
// This is a defense-in-depth measure; k is also bounded by len(items).
const maxRerankSize = 10000

// MMR implements Maximal Marginal Relevance reranking.
// It balances relevance and diversity by iteratively selecting items
// that are both relevant and dissimilar to already selected items.
//
// The MMR formula is:
//
//	MMR = argmax[lambda * score(i) - (1-lambda) * max(sim(i, s)) for s in selected]
//
// Where:
//   - lambda: balance parameter (1.0 = pure relevance, 0.0 = pure diversity)
//   - score(i): original relevance score for item i
//   - sim(i, s): similarity between item i and selected item s
//
// Reference:
// Carbonell, J., & Goldstein, J. (1998). "The Use of MMR, Diversity-Based
// Reranking for Reordering Documents and Producing Summaries." SIGIR 1998.
type MMR struct {
	// Lambda balances relevance vs. diversity (0.0 to 1.0)
	lambda float64
}

// NewMMR creates a new MMR reranker.
func NewMMR(lambda float64) *MMR {
	if lambda < 0 {
		lambda = 0
	}
	if lambda > 1 {
		lambda = 1
	}
	return &MMR{lambda: lambda}
}

// Name returns the reranker identifier.
func (m *MMR) Name() string {
	return "mmr"
}

// Rerank applies MMR reranking to diversify the recommendation list.
//
//nolint:gocritic // rangeValCopy: ScoredItem passed by value in range, acceptable for clarity
func (m *MMR) Rerank(ctx context.Context, items []recommend.ScoredItem, k int) []recommend.ScoredItem {
	if len(items) == 0 || k <= 0 {
		return items
	}

	// Bound k to prevent excessive memory allocation
	if k > maxRerankSize {
		k = maxRerankSize
	}
	if k > len(items) {
		k = len(items)
	}

	// Early return if lambda is 1.0 (pure relevance)
	if m.lambda >= 1.0 {
		if len(items) > k {
			return items[:k]
		}
		return items
	}

	// Build similarity matrix for genre-based diversity
	// This is a simplified version using genre Jaccard similarity
	similarities := m.buildSimilarityMatrix(items)

	// Greedy MMR selection
	selected := make([]recommend.ScoredItem, 0, k)
	selectedIndices := make(map[int]struct{})

	for len(selected) < k {
		bestIdx := -1
		bestMMR := -1.0

		for i, item := range items {
			if _, ok := selectedIndices[i]; ok {
				continue // Already selected
			}

			// Compute MMR score
			relevance := item.Score
			maxSim := 0.0

			for j := range selectedIndices {
				sim := similarities[i][j]
				if sim > maxSim {
					maxSim = sim
				}
			}

			mmrScore := m.lambda*relevance - (1-m.lambda)*maxSim

			if mmrScore > bestMMR {
				bestMMR = mmrScore
				bestIdx = i
			}
		}

		if bestIdx < 0 {
			break
		}

		selected = append(selected, items[bestIdx])
		selectedIndices[bestIdx] = struct{}{}
	}

	return selected
}

// buildSimilarityMatrix computes pairwise genre-based similarity.
func (m *MMR) buildSimilarityMatrix(items []recommend.ScoredItem) [][]float64 {
	n := len(items)
	similarities := make([][]float64, n)
	for i := range similarities {
		similarities[i] = make([]float64, n)
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			sim := computeGenreSimilarity(items[i].Item.Genres, items[j].Item.Genres)
			similarities[i][j] = sim
			similarities[j][i] = sim
		}
	}

	return similarities
}

// computeGenreSimilarity computes Jaccard similarity between genre lists.
func computeGenreSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}

	setA := make(map[string]struct{}, len(a))
	for _, g := range a {
		setA[strings.ToLower(g)] = struct{}{}
	}

	setB := make(map[string]struct{}, len(b))
	for _, g := range b {
		setB[strings.ToLower(g)] = struct{}{}
	}

	// Compute intersection
	intersection := 0
	for g := range setA {
		if _, ok := setB[g]; ok {
			intersection++
		}
	}

	// Compute union
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// Ensure MMR implements the interface.
var _ recommend.Reranker = (*MMR)(nil)
