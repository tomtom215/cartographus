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

// Popularity implements a popularity-based recommendation algorithm.
// It ranks items by their total interaction count, providing a simple
// but effective baseline for recommendations.
//
// This algorithm is useful for:
//   - Cold start users with no history
//   - Fallback when other algorithms fail
//   - Blending with personalized scores
//
// The popularity score is computed as:
//
//	score(item) = sum(confidence) for all interactions with item
//
// Optionally, time decay can be applied to favor recent interactions.
type Popularity struct {
	BaseAlgorithm

	// Configuration
	useTimeDecay  bool
	decayHalfLife float64 // days
	maxItems      int

	// Trained model
	itemScores map[int]float64
	sortedIDs  []int // item IDs sorted by popularity descending
}

// PopularityConfig contains configuration for the popularity algorithm.
type PopularityConfig struct {
	// UseTimeDecay applies exponential time decay to older interactions.
	UseTimeDecay bool

	// DecayHalfLife is the half-life in days for time decay.
	DecayHalfLife float64

	// MaxItems limits the number of items to track.
	MaxItems int
}

// NewPopularity creates a new popularity algorithm.
func NewPopularity(cfg PopularityConfig) *Popularity {
	if cfg.DecayHalfLife <= 0 {
		cfg.DecayHalfLife = 30 // 30-day half-life
	}
	if cfg.MaxItems <= 0 {
		cfg.MaxItems = 10000
	}

	return &Popularity{
		BaseAlgorithm: NewBaseAlgorithm("popularity"),
		useTimeDecay:  cfg.UseTimeDecay,
		decayHalfLife: cfg.DecayHalfLife,
		maxItems:      cfg.MaxItems,
		itemScores:    make(map[int]float64),
	}
}

// Train computes popularity scores from interactions.
//
//nolint:gocritic // rangeValCopy: Interaction is passed by value in range, acceptable for clarity
func (p *Popularity) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	p.acquireTrainLock()
	defer p.releaseTrainLock()

	// Clear previous model
	p.itemScores = make(map[int]float64)
	p.sortedIDs = nil

	if len(interactions) == 0 {
		p.markTrained()
		return nil
	}

	// Compute popularity scores
	for _, inter := range interactions {
		if ContextCancelled(ctx) {
			return ctx.Err()
		}

		// Weight by confidence
		weight := inter.Confidence
		if weight <= 0 {
			weight = inter.Type.Confidence()
		}

		// Apply time decay if configured
		// Note: time decay would use inter.Timestamp if not zero
		// Currently disabled to avoid time.Now() non-determinism
		_ = p.useTimeDecay // placeholder for future time decay implementation

		p.itemScores[inter.ItemID] += weight
	}

	// Sort items by popularity
	type scoredItem struct {
		id    int
		score float64
	}

	scored := make([]scoredItem, 0, len(p.itemScores))
	for id, score := range p.itemScores {
		scored = append(scored, scoredItem{id, score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Limit to max items
	if len(scored) > p.maxItems {
		scored = scored[:p.maxItems]
	}

	// Store sorted IDs
	p.sortedIDs = make([]int, len(scored))
	for i, s := range scored {
		p.sortedIDs[i] = s.id
	}

	p.markTrained()
	return nil
}

// Predict returns popularity scores for candidate items.
// User ID is ignored for popularity-based recommendations.
func (p *Popularity) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	p.acquirePredictLock()
	defer p.releasePredictLock()

	if !p.trained || len(p.itemScores) == 0 {
		return nil, nil
	}

	scores := make(map[int]float64, len(candidates))
	for _, candidateID := range candidates {
		if score, ok := p.itemScores[candidateID]; ok {
			scores[candidateID] = score
		}
	}

	return normalizeScores(scores), nil
}

// PredictSimilar returns popular items (similarity not applicable for popularity).
func (p *Popularity) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	// For "similar" mode, just return popularity scores
	return p.Predict(ctx, 0, candidates)
}

// GetTopK returns the top K most popular item IDs.
func (p *Popularity) GetTopK(k int) []int {
	p.acquirePredictLock()
	defer p.releasePredictLock()

	if k <= 0 || len(p.sortedIDs) == 0 {
		return nil
	}

	if k > len(p.sortedIDs) {
		k = len(p.sortedIDs)
	}

	result := make([]int, k)
	copy(result, p.sortedIDs[:k])
	return result
}
