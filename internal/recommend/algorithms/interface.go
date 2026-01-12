// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package algorithms implements recommendation algorithms for the hybrid engine.
//
// Each algorithm implements the recommend.Algorithm interface and can be
// registered with the recommendation engine.
//
// # Algorithm Categories
//
//   - Collaborative Filtering: EASE, ALS, UserCF, ItemCF
//   - Content-Based: Content similarity based on metadata
//   - Sequential: Co-visitation, Markov chains
//   - Popularity: Baseline popularity ranking
//
// # Thread Safety
//
// All algorithms are designed to be safe for concurrent use. Training
// acquires an exclusive lock while prediction uses a shared lock.
package algorithms

import (
	"context"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// BaseAlgorithm provides common functionality for all algorithms.
type BaseAlgorithm struct {
	name          string
	trained       bool
	version       int
	lastTrainedAt time.Time
	mu            sync.RWMutex
}

// NewBaseAlgorithm creates a new base algorithm with the given name.
func NewBaseAlgorithm(name string) BaseAlgorithm {
	return BaseAlgorithm{
		name: name,
	}
}

// Name returns the algorithm identifier.
func (b *BaseAlgorithm) Name() string {
	return b.name
}

// IsTrained returns whether the model has been trained.
func (b *BaseAlgorithm) IsTrained() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.trained
}

// Version returns the model version.
func (b *BaseAlgorithm) Version() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.version
}

// LastTrainedAt returns when the model was last trained.
func (b *BaseAlgorithm) LastTrainedAt() time.Time {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastTrainedAt
}

// markTrained updates the trained state.
// Must be called while holding the training lock (acquireTrainLock).
func (b *BaseAlgorithm) markTrained() {
	// Lock is already held by caller via acquireTrainLock()
	b.trained = true
	b.version++
	b.lastTrainedAt = time.Now()
}

// acquireTrainLock acquires the exclusive training lock.
func (b *BaseAlgorithm) acquireTrainLock() {
	b.mu.Lock()
}

// releaseTrainLock releases the exclusive training lock.
func (b *BaseAlgorithm) releaseTrainLock() {
	b.mu.Unlock()
}

// acquirePredictLock acquires the shared prediction lock.
func (b *BaseAlgorithm) acquirePredictLock() {
	b.mu.RLock()
}

// releasePredictLock releases the shared prediction lock.
func (b *BaseAlgorithm) releasePredictLock() {
	b.mu.RUnlock()
}

// normalizeScores normalizes scores to [0, 1] range using min-max normalization.
func normalizeScores(scores map[int]float64) map[int]float64 {
	if len(scores) == 0 {
		return scores
	}

	// Find min and max
	var minScore, maxScore float64
	first := true
	for _, score := range scores {
		if first {
			minScore, maxScore = score, score
			first = false
			continue
		}
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
	}

	// Avoid division by zero
	rang := maxScore - minScore
	if rang == 0 {
		// All scores are equal - return 0.5 for all
		for id := range scores {
			scores[id] = 0.5
		}
		return scores
	}

	// Normalize
	for id, score := range scores {
		scores[id] = (score - minScore) / rang
	}

	return scores
}

// filterCandidates filters scores to only include candidate items.
//
//nolint:unused // utility function for future use
func filterCandidates(scores map[int]float64, candidates []int) map[int]float64 {
	candidateSet := make(map[int]struct{}, len(candidates))
	for _, id := range candidates {
		candidateSet[id] = struct{}{}
	}

	filtered := make(map[int]float64, len(candidates))
	for id, score := range scores {
		if _, ok := candidateSet[id]; ok {
			filtered[id] = score
		}
	}

	return filtered
}

// buildItemIndex creates a mapping from item ID to index.
//
//nolint:unused,gocritic // unused: utility function for future use; gocritic: rangeValCopy is acceptable
func buildItemIndex(items []recommend.Item) map[int]int {
	index := make(map[int]int, len(items))
	for i, item := range items {
		index[item.ID] = i
	}
	return index
}

// buildUserIndex creates a mapping from user ID to index.
//
//nolint:unused // utility function for future use
func buildUserIndex(interactions []recommend.Interaction) map[int]int {
	seen := make(map[int]struct{})
	index := make(map[int]int)

	for _, inter := range interactions {
		if _, ok := seen[inter.UserID]; !ok {
			index[inter.UserID] = len(index)
			seen[inter.UserID] = struct{}{}
		}
	}

	return index
}

// buildInteractionMatrix creates a user-item interaction matrix.
// Returns the matrix and mappings for user/item indices.
//
//nolint:unused // utility function for future use
func buildInteractionMatrix(interactions []recommend.Interaction) ([][]float64, map[int]int, map[int]int) {
	userIndex := buildUserIndex(interactions)
	itemSet := make(map[int]struct{})
	for _, inter := range interactions {
		itemSet[inter.ItemID] = struct{}{}
	}

	itemIndex := make(map[int]int, len(itemSet))
	i := 0
	for itemID := range itemSet {
		itemIndex[itemID] = i
		i++
	}

	// Create matrix
	matrix := make([][]float64, len(userIndex))
	for i := range matrix {
		matrix[i] = make([]float64, len(itemIndex))
	}

	// Fill matrix with confidence values
	for _, inter := range interactions {
		ui := userIndex[inter.UserID]
		ii := itemIndex[inter.ItemID]
		matrix[ui][ii] = inter.Confidence
	}

	return matrix, userIndex, itemIndex
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt returns the square root using Newton's method.
// This avoids importing math for a simple operation.
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}

	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// jaccardSimilarity computes Jaccard similarity between two sets.
func jaccardSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}

	setA := make(map[string]struct{}, len(a))
	for _, s := range a {
		setA[s] = struct{}{}
	}

	setB := make(map[string]struct{}, len(b))
	for _, s := range b {
		setB[s] = struct{}{}
	}

	// Compute intersection
	intersection := 0
	for s := range setA {
		if _, ok := setB[s]; ok {
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

// Ensure all algorithms implement the interface.
var (
	_ recommend.Algorithm = (*CoVisitation)(nil)
	_ recommend.Algorithm = (*ContentBased)(nil)
	_ recommend.Algorithm = (*Popularity)(nil)
	_ recommend.Algorithm = (*EASE)(nil)
	_ recommend.Algorithm = (*ALS)(nil)
	_ recommend.Algorithm = (*UserBasedCF)(nil)
	_ recommend.Algorithm = (*ItemBasedCF)(nil)
	_ recommend.Algorithm = (*FPMC)(nil)
	_ recommend.Algorithm = (*LinUCB)(nil)
	_ recommend.Algorithm = (*BPR)(nil)
	_ recommend.Algorithm = (*TimeAwareCF)(nil)
	_ recommend.Algorithm = (*MultiHopItemCF)(nil)
	_ recommend.Algorithm = (*MarkovChain)(nil)
)

// ContextCancelled checks if the context has been canceled.
func ContextCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
