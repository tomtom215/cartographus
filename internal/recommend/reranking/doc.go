// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package reranking implements post-processing algorithms for recommendation diversity.
//
// This package provides reranking algorithms that modify recommendation lists
// to achieve objectives beyond pure relevance, such as diversity, novelty,
// fairness, and calibration. Rerankers operate on already-scored recommendations
// and reorder them to balance multiple objectives.
//
// # Overview
//
// Reranking is applied after the initial ranking from recommendation algorithms:
//
//	Algorithms -> Initial Ranking -> Rerankers -> Final Ranking
//	(relevance)                      (diversity, fairness, etc.)
//
// # Available Rerankers
//
// Maximal Marginal Relevance (MMR):
//   - Balances relevance with diversity
//   - Penalizes items similar to already-selected items
//   - Lambda parameter controls relevance/diversity tradeoff
//
// Calibration:
//   - Ensures recommendation distribution matches user history
//   - Prevents over-representation of popular genres
//   - Useful for avoiding filter bubbles
//
// # Interface
//
// All rerankers implement the recommend.Reranker interface:
//
//	type Reranker interface {
//	    Name() string
//	    Rerank(ctx context.Context, items []ScoredItem, k int) []ScoredItem
//	}
//
// # Usage Example
//
// Applying MMR reranking:
//
//	// Get initial recommendations from algorithm
//	scores, err := algorithm.Predict(ctx, req)
//	items := convertToScoredItems(scores, itemCatalog)
//
//	// Apply MMR with 0.7 relevance / 0.3 diversity balance
//	mmr := reranking.NewMMR(0.7)
//	diversified := mmr.Rerank(ctx, items, 20)
//
// Chaining multiple rerankers:
//
//	// First diversify, then calibrate
//	mmr := reranking.NewMMR(0.8)
//	cal := reranking.NewCalibration(userProfile)
//
//	result := mmr.Rerank(ctx, items, 50)  // Over-fetch for calibration
//	result = cal.Rerank(ctx, result, 20)  // Final selection
//
// # MMR Algorithm
//
// Maximal Marginal Relevance iteratively selects items that are both
// relevant and dissimilar to already-selected items:
//
//	MMR = argmax[lambda * score(i) - (1-lambda) * max_similarity(i, selected)]
//
// Where:
//   - lambda: Balance parameter (1.0 = pure relevance, 0.0 = pure diversity)
//   - score(i): Original relevance score for item i
//   - max_similarity: Maximum similarity to any selected item
//
// Lambda Guidelines:
//   - 0.9-1.0: Mostly relevance, minimal diversity (safe default)
//   - 0.7-0.9: Balanced (recommended for most use cases)
//   - 0.5-0.7: Strong diversity push
//   - 0.0-0.5: Diversity-focused (may sacrifice relevance)
//
// # Similarity Metrics
//
// MMR uses genre-based Jaccard similarity for diversity:
//
//	sim(a, b) = |genres(a) intersection genres(b)| / |genres(a) union genres(b)|
//
// Items with identical genres have similarity 1.0, completely different
// genres have similarity 0.0.
//
// # Calibration Algorithm
//
// Calibration ensures the recommendation distribution matches the user's
// historical preferences:
//
//	target_dist = user genre distribution from history
//	actual_dist = current recommendation genre distribution
//	penalty = KL_divergence(actual_dist, target_dist)
//
// This prevents scenarios where a user who watches 30% action, 30% comedy,
// 40% drama gets recommendations that are 80% action.
//
// # Performance
//
// MMR Complexity:
//   - Time: O(k * n^2) where k = output size, n = input size
//   - Space: O(n^2) for similarity matrix
//
// For large catalogs, consider:
//   - Pre-filtering to top 100-200 items before reranking
//   - Approximate similarity using feature hashing
//
// # Thread Safety
//
// All rerankers are stateless and safe for concurrent use. The same
// reranker instance can process multiple requests simultaneously.
//
// # See Also
//
//   - internal/recommend/algorithms: Scoring algorithms
//   - internal/recommend: Engine that orchestrates reranking
//   - Carbonell & Goldstein (1998): "The Use of MMR" SIGIR paper
package reranking
