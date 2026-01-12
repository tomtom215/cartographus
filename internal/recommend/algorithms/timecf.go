// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"math"
	"time"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// TimeAwareCFConfig contains configuration for the Time-Aware CF algorithm.
type TimeAwareCFConfig struct {
	// DecayRate controls how fast older interactions lose weight.
	// Higher values mean faster decay. A value of 0.1 means interactions
	// lose ~10% of their weight every DecayUnit.
	// Default: 0.1.
	DecayRate float64

	// DecayUnit is the time unit for decay calculation.
	// Default: 24 hours (1 day).
	DecayUnit time.Duration

	// MaxLookback is the maximum age of interactions to consider.
	// Interactions older than this are ignored.
	// Default: 365 days.
	MaxLookback time.Duration

	// MinWeight is the minimum weight an interaction can have.
	// This prevents very old interactions from having zero contribution.
	// Default: 0.01.
	MinWeight float64

	// NumNeighbors is the number of similar users/items to consider.
	// Default: 50.
	NumNeighbors int

	// MinConfidence is the minimum confidence threshold for interactions.
	// Default: 0.1.
	MinConfidence float64

	// Mode determines whether to use user-based or item-based CF.
	// Default: "user".
	Mode string // "user" or "item"

	// ReferenceTime is used for calculating decay (for reproducibility).
	// If zero, uses time.Now() during training.
	ReferenceTime time.Time
}

// DefaultTimeAwareCFConfig returns default Time-Aware CF configuration.
func DefaultTimeAwareCFConfig() TimeAwareCFConfig {
	return TimeAwareCFConfig{
		DecayRate:     0.1,
		DecayUnit:     24 * time.Hour,
		MaxLookback:   365 * 24 * time.Hour,
		MinWeight:     0.01,
		NumNeighbors:  50,
		MinConfidence: 0.1,
		Mode:          "user",
	}
}

// TimeAwareCF implements time-weighted collaborative filtering.
//
// This algorithm extends traditional collaborative filtering by applying
// exponential decay to older interactions. The intuition is that user
// preferences evolve over time, so recent interactions are more indicative
// of current preferences.
//
// Weight function: w(t) = max(MinWeight, exp(-DecayRate * (now - t) / DecayUnit))
//
// For user-based CF:
// - Compute weighted similarity between users
// - Predict items based on what similar users have watched recently
//
// For item-based CF:
// - Compute weighted similarity between items
// - Predict items similar to what the user has watched recently
type TimeAwareCF struct {
	BaseAlgorithm
	config TimeAwareCFConfig

	// userIndex maps user ID to internal index
	userIndex map[int]int

	// itemIndex maps item ID to internal index
	itemIndex map[int]int

	// indexToUser maps internal index to user ID
	indexToUser []int

	// indexToItem maps internal index to item ID
	indexToItem []int

	// userItemWeights[user][item] = time-weighted confidence
	userItemWeights map[int]map[int]float64

	// userSimilarity[user1][user2] = similarity (for user-based mode)
	userSimilarity map[int]map[int]float64

	// itemSimilarity[item1][item2] = similarity (for item-based mode)
	itemSimilarity map[int]map[int]float64

	// itemUsers[item] = set of users who interacted with the item
	itemUsers map[int]map[int]float64

	// referenceTime used for decay calculation
	referenceTime time.Time
}

// NewTimeAwareCF creates a new Time-Aware CF algorithm with the given configuration.
func NewTimeAwareCF(cfg *TimeAwareCFConfig) *TimeAwareCF {
	// Handle nil config with defaults
	if cfg == nil {
		cfg = &TimeAwareCFConfig{}
	}

	// Apply defaults for zero values
	config := *cfg
	if config.DecayRate <= 0 {
		config.DecayRate = 0.1
	}
	if config.DecayUnit <= 0 {
		config.DecayUnit = 24 * time.Hour
	}
	if config.MaxLookback <= 0 {
		config.MaxLookback = 365 * 24 * time.Hour
	}
	if config.MinWeight <= 0 {
		config.MinWeight = 0.01
	}
	if config.NumNeighbors <= 0 {
		config.NumNeighbors = 50
	}
	if config.MinConfidence <= 0 {
		config.MinConfidence = 0.1
	}
	if config.Mode == "" {
		config.Mode = "user"
	}

	return &TimeAwareCF{
		BaseAlgorithm:   NewBaseAlgorithm("time_aware_cf"),
		config:          config,
		userIndex:       make(map[int]int),
		itemIndex:       make(map[int]int),
		userItemWeights: make(map[int]map[int]float64),
		userSimilarity:  make(map[int]map[int]float64),
		itemSimilarity:  make(map[int]map[int]float64),
		itemUsers:       make(map[int]map[int]float64),
	}
}

// computeTimeWeight calculates the weight for an interaction based on age.
func (t *TimeAwareCF) computeTimeWeight(timestamp time.Time) float64 {
	if timestamp.IsZero() {
		return 1.0 // No timestamp means full weight
	}

	age := t.referenceTime.Sub(timestamp)
	if age < 0 {
		age = 0 // Future timestamps get full weight
	}

	if age > t.config.MaxLookback {
		return 0 // Too old
	}

	// Exponential decay: w = exp(-decay_rate * age_in_units)
	ageInUnits := float64(age) / float64(t.config.DecayUnit)
	weight := math.Exp(-t.config.DecayRate * ageInUnits)

	if weight < t.config.MinWeight {
		weight = t.config.MinWeight
	}

	return weight
}

// Train builds the time-weighted similarity matrices.
//
//nolint:gocyclo,gocritic // gocyclo: ML training algorithms are inherently complex; gocritic: rangeValCopy acceptable
func (t *TimeAwareCF) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	t.acquireTrainLock()
	defer t.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Set reference time
	if t.config.ReferenceTime.IsZero() {
		t.referenceTime = time.Now()
	} else {
		t.referenceTime = t.config.ReferenceTime
	}

	// Reset state
	t.userIndex = make(map[int]int)
	t.itemIndex = make(map[int]int)
	t.indexToUser = nil
	t.indexToItem = nil
	t.userItemWeights = make(map[int]map[int]float64)
	t.userSimilarity = make(map[int]map[int]float64)
	t.itemSimilarity = make(map[int]map[int]float64)
	t.itemUsers = make(map[int]map[int]float64)

	// Build indices and weighted interactions
	cutoffTime := t.referenceTime.Add(-t.config.MaxLookback)

	for _, inter := range interactions {
		if inter.Confidence < t.config.MinConfidence {
			continue
		}

		// Skip interactions older than max lookback
		if !inter.Timestamp.IsZero() && inter.Timestamp.Before(cutoffTime) {
			continue
		}

		// Register user
		if _, ok := t.userIndex[inter.UserID]; !ok {
			t.userIndex[inter.UserID] = len(t.indexToUser)
			t.indexToUser = append(t.indexToUser, inter.UserID)
		}

		// Register item
		if _, ok := t.itemIndex[inter.ItemID]; !ok {
			t.itemIndex[inter.ItemID] = len(t.indexToItem)
			t.indexToItem = append(t.indexToItem, inter.ItemID)
		}

		ui := t.userIndex[inter.UserID]
		ii := t.itemIndex[inter.ItemID]

		// Compute time-weighted score
		timeWeight := t.computeTimeWeight(inter.Timestamp)
		weight := inter.Confidence * timeWeight

		// Store user-item weight
		if t.userItemWeights[ui] == nil {
			t.userItemWeights[ui] = make(map[int]float64)
		}
		// Take max for duplicate interactions
		if weight > t.userItemWeights[ui][ii] {
			t.userItemWeights[ui][ii] = weight
		}

		// Store item-user mapping
		if t.itemUsers[ii] == nil {
			t.itemUsers[ii] = make(map[int]float64)
		}
		if weight > t.itemUsers[ii][ui] {
			t.itemUsers[ii][ui] = weight
		}
	}

	numUsers := len(t.indexToUser)
	numItems := len(t.indexToItem)

	if numUsers == 0 || numItems == 0 {
		t.markTrained()
		return nil
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Build similarity matrices based on mode
	if t.config.Mode == "item" {
		t.buildItemSimilarity(numItems)
	} else {
		t.buildUserSimilarity(numUsers)
	}

	t.markTrained()
	return nil
}

// buildUserSimilarity computes weighted cosine similarity between users.
func (t *TimeAwareCF) buildUserSimilarity(numUsers int) {
	// Pre-compute user vector norms
	userNorms := make([]float64, numUsers)
	for ui, itemWeights := range t.userItemWeights {
		var sumSq float64
		for _, weight := range itemWeights {
			sumSq += weight * weight
		}
		userNorms[ui] = math.Sqrt(sumSq)
	}

	// Compute similarities
	for u1 := 0; u1 < numUsers; u1++ {
		items1 := t.userItemWeights[u1]
		if len(items1) == 0 || userNorms[u1] == 0 {
			continue
		}

		t.userSimilarity[u1] = make(map[int]float64)

		for u2 := u1 + 1; u2 < numUsers; u2++ {
			items2 := t.userItemWeights[u2]
			if len(items2) == 0 || userNorms[u2] == 0 {
				continue
			}

			// Compute dot product
			var dotProduct float64
			for ii, w1 := range items1 {
				if w2, ok := items2[ii]; ok {
					dotProduct += w1 * w2
				}
			}

			if dotProduct > 0 {
				sim := dotProduct / (userNorms[u1] * userNorms[u2])
				t.userSimilarity[u1][u2] = sim
				if t.userSimilarity[u2] == nil {
					t.userSimilarity[u2] = make(map[int]float64)
				}
				t.userSimilarity[u2][u1] = sim
			}
		}
	}
}

// buildItemSimilarity computes weighted cosine similarity between items.
func (t *TimeAwareCF) buildItemSimilarity(numItems int) {
	// Pre-compute item vector norms
	itemNorms := make([]float64, numItems)
	for ii, userWeights := range t.itemUsers {
		var sumSq float64
		for _, weight := range userWeights {
			sumSq += weight * weight
		}
		itemNorms[ii] = math.Sqrt(sumSq)
	}

	// Compute similarities
	for i1 := 0; i1 < numItems; i1++ {
		users1 := t.itemUsers[i1]
		if len(users1) == 0 || itemNorms[i1] == 0 {
			continue
		}

		t.itemSimilarity[i1] = make(map[int]float64)

		for i2 := i1 + 1; i2 < numItems; i2++ {
			users2 := t.itemUsers[i2]
			if len(users2) == 0 || itemNorms[i2] == 0 {
				continue
			}

			// Compute dot product
			var dotProduct float64
			for ui, w1 := range users1 {
				if w2, ok := users2[ui]; ok {
					dotProduct += w1 * w2
				}
			}

			if dotProduct > 0 {
				sim := dotProduct / (itemNorms[i1] * itemNorms[i2])
				t.itemSimilarity[i1][i2] = sim
				if t.itemSimilarity[i2] == nil {
					t.itemSimilarity[i2] = make(map[int]float64)
				}
				t.itemSimilarity[i2][i1] = sim
			}
		}
	}
}

// Predict returns scores for candidate items for a user.
func (t *TimeAwareCF) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	t.acquirePredictLock()
	defer t.releasePredictLock()

	if !t.trained {
		return nil, nil
	}

	ui, ok := t.userIndex[userID]
	if !ok {
		return nil, nil
	}

	if t.config.Mode == "item" {
		return t.predictItemBased(ui, candidates), nil
	}
	return t.predictUserBased(ui, candidates), nil
}

// predictUserBased uses user similarity for prediction.
func (t *TimeAwareCF) predictUserBased(userIdx int, candidates []int) map[int]float64 {
	// Get similar users
	neighbors := t.getTopNeighbors(t.userSimilarity[userIdx], t.config.NumNeighbors)
	if len(neighbors) == 0 {
		return nil
	}

	scores := make(map[int]float64, len(candidates))

	for _, itemID := range candidates {
		ii, ok := t.itemIndex[itemID]
		if !ok {
			continue
		}

		// Score = sum of (similarity * neighbor's weight for item)
		var sumScore, sumSim float64
		for neighborIdx, sim := range neighbors {
			if weight, ok := t.userItemWeights[neighborIdx][ii]; ok {
				sumScore += sim * weight
				sumSim += sim
			}
		}

		if sumSim > 0 {
			scores[itemID] = sumScore / sumSim
		}
	}

	return normalizeScores(scores)
}

// predictItemBased uses item similarity for prediction.
func (t *TimeAwareCF) predictItemBased(userIdx int, candidates []int) map[int]float64 {
	userItems := t.userItemWeights[userIdx]
	if len(userItems) == 0 {
		return nil
	}

	scores := make(map[int]float64, len(candidates))

	for _, itemID := range candidates {
		ii, ok := t.itemIndex[itemID]
		if !ok {
			continue
		}

		// Skip items user already has
		if _, ok := userItems[ii]; ok {
			continue
		}

		// Score = sum of (similarity * user's weight for similar item)
		var sumScore, sumSim float64

		for userItemIdx, userWeight := range userItems {
			var sim float64
			if sims, ok := t.itemSimilarity[ii]; ok {
				sim = sims[userItemIdx]
			}
			if sim > 0 {
				sumScore += sim * userWeight
				sumSim += sim
			}
		}

		if sumSim > 0 {
			scores[itemID] = sumScore / sumSim
		}
	}

	return normalizeScores(scores)
}

// getTopNeighbors returns the top-K most similar neighbors.
func (t *TimeAwareCF) getTopNeighbors(similarities map[int]float64, k int) map[int]float64 {
	if len(similarities) == 0 {
		return nil
	}

	// Convert to slice for sorting
	type neighbor struct {
		idx int
		sim float64
	}
	neighbors := make([]neighbor, 0, len(similarities))
	for idx, sim := range similarities {
		neighbors = append(neighbors, neighbor{idx, sim})
	}

	// Simple selection of top-K (insertion sort for small K)
	for i := 0; i < len(neighbors) && i < k; i++ {
		maxIdx := i
		for j := i + 1; j < len(neighbors); j++ {
			if neighbors[j].sim > neighbors[maxIdx].sim {
				maxIdx = j
			}
		}
		neighbors[i], neighbors[maxIdx] = neighbors[maxIdx], neighbors[i]
	}

	// Convert back to map
	result := make(map[int]float64, k)
	for i := 0; i < len(neighbors) && i < k; i++ {
		result[neighbors[i].idx] = neighbors[i].sim
	}

	return result
}

// PredictSimilar returns items similar to the given item.
func (t *TimeAwareCF) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	t.acquirePredictLock()
	defer t.releasePredictLock()

	if !t.trained {
		return nil, nil
	}

	sourceIdx, ok := t.itemIndex[itemID]
	if !ok {
		return nil, nil
	}

	scores := make(map[int]float64, len(candidates))

	for _, candidateID := range candidates {
		if candidateID == itemID {
			continue
		}

		candidateIdx, ok := t.itemIndex[candidateID]
		if !ok {
			continue
		}

		// Get similarity from precomputed matrix
		var sim float64
		if sims, ok := t.itemSimilarity[sourceIdx]; ok {
			sim = sims[candidateIdx]
		}
		if sim > 0 {
			scores[candidateID] = sim
		}
	}

	return normalizeScores(scores), nil
}

// GetDecayRate returns the current decay rate.
func (t *TimeAwareCF) GetDecayRate() float64 {
	return t.config.DecayRate
}

// GetMode returns the current CF mode ("user" or "item").
func (t *TimeAwareCF) GetMode() string {
	return t.config.Mode
}

// Ensure interface compliance.
var _ recommend.Algorithm = (*TimeAwareCF)(nil)
