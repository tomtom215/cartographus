// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"sort"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// MultiHopItemCFConfig contains configuration for the Multi-Hop ItemCF algorithm.
type MultiHopItemCFConfig struct {
	// NumHops is the number of similarity propagation hops.
	// 1 = standard ItemCF, 2-3 = multi-hop expansion.
	// Higher values provide more coverage but may reduce precision.
	// Default: 2.
	NumHops int

	// TopKPerHop is the number of similar items to consider at each hop.
	// Smaller values are faster but may miss relevant items.
	// Default: 10.
	TopKPerHop int

	// DecayFactor controls how much scores decay per hop.
	// Score at hop h is multiplied by DecayFactor^h.
	// Default: 0.5.
	DecayFactor float64

	// MinSimilarity is the minimum similarity threshold.
	// Pairs below this threshold are not considered.
	// Default: 0.1.
	MinSimilarity float64

	// MinConfidence is the minimum interaction confidence threshold.
	// Default: 0.1.
	MinConfidence float64

	// MaxItemsPerUser limits memory usage for users with many interactions.
	// Default: 100.
	MaxItemsPerUser int
}

// DefaultMultiHopItemCFConfig returns default Multi-Hop ItemCF configuration.
func DefaultMultiHopItemCFConfig() MultiHopItemCFConfig {
	return MultiHopItemCFConfig{
		NumHops:         2,
		TopKPerHop:      10,
		DecayFactor:     0.5,
		MinSimilarity:   0.1,
		MinConfidence:   0.1,
		MaxItemsPerUser: 100,
	}
}

// MultiHopItemCF implements multi-hop item-based collaborative filtering.
//
// This algorithm extends traditional item-based CF by propagating similarity
// through multiple hops in the item-item similarity graph. This provides:
//
//   - Better coverage: Can recommend items that are 2-3 hops away from
//     what the user has watched, not just direct neighbors.
//   - Graph-like expansion: Approximates the effect of graph neural networks
//     (like LightGCN) but with much lower resource requirements.
//   - Serendipity: Discovers items through chains of similarity that would
//     be missed by single-hop methods.
//
// Example with 2 hops:
//
//	User watched: MovieA
//	Hop 1: MovieA similar to MovieB, MovieC (TopKPerHop = 2)
//	Hop 2: MovieB similar to MovieD, MovieE; MovieC similar to MovieF, MovieG
//	Final candidates: MovieB (decay^1), MovieC (decay^1),
//	                  MovieD (decay^2), MovieE (decay^2), MovieF (decay^2), MovieG (decay^2)
//
// The score for each candidate is:
//
//	score = sum over paths: (similarity_product * decay^hop_number * user_weight)
//
// Memory: O(K * items) for similarity matrix with TopK per item
// CPU: O(K^hops) per prediction, typically small due to limited K
type MultiHopItemCF struct {
	BaseAlgorithm
	config MultiHopItemCFConfig

	// itemIndex maps item ID to internal index
	itemIndex map[int]int

	// indexToItem maps internal index to item ID
	indexToItem []int

	// userIndex maps user ID to internal index
	userIndex map[int]int

	// userItems[userIdx] = map[itemIdx]weight
	userItems map[int]map[int]float64

	// itemSimilarity[itemIdx] = sorted list of (similar_item, similarity)
	// Only stores TopKPerHop most similar items per item
	itemSimilarity map[int][]itemSim
}

// itemSim represents an item similarity pair.
type itemSim struct {
	itemIdx    int
	similarity float64
}

// NewMultiHopItemCF creates a new Multi-Hop ItemCF algorithm.
func NewMultiHopItemCF(cfg MultiHopItemCFConfig) *MultiHopItemCF {
	if cfg.NumHops <= 0 {
		cfg.NumHops = 2
	}
	if cfg.TopKPerHop <= 0 {
		cfg.TopKPerHop = 10
	}
	if cfg.DecayFactor <= 0 || cfg.DecayFactor > 1 {
		cfg.DecayFactor = 0.5
	}
	if cfg.MinSimilarity <= 0 {
		cfg.MinSimilarity = 0.1
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = 0.1
	}
	if cfg.MaxItemsPerUser <= 0 {
		cfg.MaxItemsPerUser = 100
	}

	return &MultiHopItemCF{
		BaseAlgorithm:  NewBaseAlgorithm("multihop_itemcf"),
		config:         cfg,
		itemIndex:      make(map[int]int),
		userIndex:      make(map[int]int),
		userItems:      make(map[int]map[int]float64),
		itemSimilarity: make(map[int][]itemSim),
	}
}

// Train builds the item similarity matrix with top-K pruning.
//
//nolint:gocyclo,gocritic // gocyclo: ML training is complex; gocritic: rangeValCopy acceptable
func (m *MultiHopItemCF) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	m.acquireTrainLock()
	defer m.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Reset state
	m.itemIndex = make(map[int]int)
	m.indexToItem = nil
	m.userIndex = make(map[int]int)
	m.userItems = make(map[int]map[int]float64)
	m.itemSimilarity = make(map[int][]itemSim)

	// Build indices
	for _, inter := range interactions {
		if inter.Confidence < m.config.MinConfidence {
			continue
		}

		if _, ok := m.itemIndex[inter.ItemID]; !ok {
			m.itemIndex[inter.ItemID] = len(m.indexToItem)
			m.indexToItem = append(m.indexToItem, inter.ItemID)
		}

		if _, ok := m.userIndex[inter.UserID]; !ok {
			m.userIndex[inter.UserID] = len(m.userIndex)
			m.userItems[m.userIndex[inter.UserID]] = make(map[int]float64)
		}
	}

	numItems := len(m.indexToItem)
	if numItems == 0 {
		m.markTrained()
		return nil
	}

	// Build user-item interactions
	for _, inter := range interactions {
		if inter.Confidence < m.config.MinConfidence {
			continue
		}

		ui := m.userIndex[inter.UserID]
		ii := m.itemIndex[inter.ItemID]

		// Limit items per user to control memory
		if len(m.userItems[ui]) >= m.config.MaxItemsPerUser {
			continue
		}

		// Take max confidence for duplicates
		if inter.Confidence > m.userItems[ui][ii] {
			m.userItems[ui][ii] = inter.Confidence
		}
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Build item-user inverted index
	itemUsers := make(map[int]map[int]float64)
	for ui, items := range m.userItems {
		for ii, weight := range items {
			if itemUsers[ii] == nil {
				itemUsers[ii] = make(map[int]float64)
			}
			itemUsers[ii][ui] = weight
		}
	}

	// Compute item similarities (top-K per item)
	m.computeItemSimilarities(itemUsers, numItems)

	m.markTrained()
	return nil
}

// computeItemSimilarities calculates cosine similarity between items.
func (m *MultiHopItemCF) computeItemSimilarities(itemUsers map[int]map[int]float64, numItems int) {
	// Pre-compute item norms
	itemNorms := make([]float64, numItems)
	for ii, users := range itemUsers {
		var sumSq float64
		for _, weight := range users {
			sumSq += weight * weight
		}
		itemNorms[ii] = sqrt(sumSq)
	}

	// Compute similarities and keep top-K per item
	for i1 := 0; i1 < numItems; i1++ {
		users1 := itemUsers[i1]
		if len(users1) == 0 || itemNorms[i1] == 0 {
			continue
		}

		// Collect all similarities for this item
		var similarities []itemSim

		for i2 := 0; i2 < numItems; i2++ {
			if i1 == i2 {
				continue
			}

			users2 := itemUsers[i2]
			if len(users2) == 0 || itemNorms[i2] == 0 {
				continue
			}

			// Compute dot product (only overlapping users)
			var dotProduct float64
			for ui, w1 := range users1 {
				if w2, ok := users2[ui]; ok {
					dotProduct += w1 * w2
				}
			}

			if dotProduct <= 0 {
				continue
			}

			sim := dotProduct / (itemNorms[i1] * itemNorms[i2])
			if sim >= m.config.MinSimilarity {
				similarities = append(similarities, itemSim{i2, sim})
			}
		}

		// Sort by similarity (descending) and keep top-K
		sort.Slice(similarities, func(a, b int) bool {
			return similarities[a].similarity > similarities[b].similarity
		})

		if len(similarities) > m.config.TopKPerHop {
			similarities = similarities[:m.config.TopKPerHop]
		}

		m.itemSimilarity[i1] = similarities
	}
}

// Predict returns scores for candidate items using multi-hop expansion.
func (m *MultiHopItemCF) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	m.acquirePredictLock()
	defer m.releasePredictLock()

	if !m.trained {
		return nil, nil
	}

	ui, ok := m.userIndex[userID]
	if !ok {
		return nil, nil
	}

	userItemSet := m.userItems[ui]
	if len(userItemSet) == 0 {
		return nil, nil
	}

	// Build candidate set for efficient lookup
	candidateSet := make(map[int]struct{}, len(candidates))
	for _, itemID := range candidates {
		if ii, ok := m.itemIndex[itemID]; ok {
			candidateSet[ii] = struct{}{}
		}
	}

	// Multi-hop expansion
	// scores[itemIdx] = accumulated score
	scores := make(map[int]float64)

	// currentFrontier: items to expand from, with their accumulated weights
	type frontierItem struct {
		itemIdx int
		weight  float64
	}

	// Start from user's watched items
	currentFrontier := make([]frontierItem, 0, len(userItemSet))
	visited := make(map[int]struct{})

	for ii, weight := range userItemSet {
		currentFrontier = append(currentFrontier, frontierItem{ii, weight})
		visited[ii] = struct{}{}
	}

	// Expand through hops
	for hop := 1; hop <= m.config.NumHops; hop++ {
		if len(currentFrontier) == 0 {
			break
		}

		decay := m.pow(m.config.DecayFactor, hop)
		nextFrontier := make([]frontierItem, 0)

		for _, frontier := range currentFrontier {
			similarities := m.itemSimilarity[frontier.itemIdx]

			for _, sim := range similarities {
				neighborIdx := sim.itemIdx

				// Skip already visited items
				if _, ok := visited[neighborIdx]; ok {
					continue
				}

				// Calculate contribution
				contribution := frontier.weight * sim.similarity * decay

				// If this is a candidate, add to scores
				if _, isCandidate := candidateSet[neighborIdx]; isCandidate {
					scores[neighborIdx] += contribution
				}

				// Add to next frontier for further expansion
				// Mark as visited to avoid cycles
				visited[neighborIdx] = struct{}{}
				nextFrontier = append(nextFrontier, frontierItem{neighborIdx, frontier.weight * sim.similarity})
			}
		}

		currentFrontier = nextFrontier
	}

	// Convert internal indices to item IDs
	result := make(map[int]float64, len(scores))
	for ii, score := range scores {
		if ii < len(m.indexToItem) {
			result[m.indexToItem[ii]] = score
		}
	}

	return normalizeScores(result), nil
}

// pow computes x^n using repeated squaring.
func (m *MultiHopItemCF) pow(x float64, n int) float64 {
	if n == 0 {
		return 1.0
	}
	result := 1.0
	for n > 0 {
		if n%2 == 1 {
			result *= x
		}
		x *= x
		n /= 2
	}
	return result
}

// PredictSimilar returns items similar to the given item using multi-hop.
func (m *MultiHopItemCF) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	m.acquirePredictLock()
	defer m.releasePredictLock()

	if !m.trained {
		return nil, nil
	}

	sourceIdx, ok := m.itemIndex[itemID]
	if !ok {
		return nil, nil
	}

	// Build candidate set
	candidateSet := make(map[int]struct{}, len(candidates))
	for _, id := range candidates {
		if id != itemID {
			if ii, ok := m.itemIndex[id]; ok {
				candidateSet[ii] = struct{}{}
			}
		}
	}

	// Multi-hop expansion from source item
	scores := make(map[int]float64)

	type frontierItem struct {
		itemIdx int
		weight  float64
	}

	currentFrontier := []frontierItem{{sourceIdx, 1.0}}
	visited := map[int]struct{}{sourceIdx: {}}

	for hop := 1; hop <= m.config.NumHops; hop++ {
		if len(currentFrontier) == 0 {
			break
		}

		decay := m.pow(m.config.DecayFactor, hop)
		nextFrontier := make([]frontierItem, 0)

		for _, frontier := range currentFrontier {
			similarities := m.itemSimilarity[frontier.itemIdx]

			for _, sim := range similarities {
				neighborIdx := sim.itemIdx

				if _, ok := visited[neighborIdx]; ok {
					continue
				}

				contribution := frontier.weight * sim.similarity * decay

				if _, isCandidate := candidateSet[neighborIdx]; isCandidate {
					scores[neighborIdx] += contribution
				}

				visited[neighborIdx] = struct{}{}
				nextFrontier = append(nextFrontier, frontierItem{neighborIdx, frontier.weight * sim.similarity})
			}
		}

		currentFrontier = nextFrontier
	}

	// Convert to item IDs
	result := make(map[int]float64, len(scores))
	for ii, score := range scores {
		if ii < len(m.indexToItem) {
			result[m.indexToItem[ii]] = score
		}
	}

	return normalizeScores(result), nil
}

// GetNumHops returns the configured number of hops.
func (m *MultiHopItemCF) GetNumHops() int {
	return m.config.NumHops
}

// GetDecayFactor returns the configured decay factor.
func (m *MultiHopItemCF) GetDecayFactor() float64 {
	return m.config.DecayFactor
}

// GetSimilarityCount returns the number of item similarity pairs stored.
func (m *MultiHopItemCF) GetSimilarityCount() int {
	m.acquirePredictLock()
	defer m.releasePredictLock()

	count := 0
	for _, sims := range m.itemSimilarity {
		count += len(sims)
	}
	return count
}

// Ensure interface compliance.
var _ recommend.Algorithm = (*MultiHopItemCF)(nil)
