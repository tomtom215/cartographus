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

// MarkovChainConfig contains configuration for the Markov Chain algorithm.
type MarkovChainConfig struct {
	// OrderK is the Markov chain order (1 = first-order, uses only last item).
	// Higher orders use more history but require more memory.
	// Default: 1.
	OrderK int

	// MinTransitionCount is the minimum number of times a transition must occur
	// to be included in the model. Helps filter noise.
	// Default: 2.
	MinTransitionCount int

	// MaxTransitionsPerItem limits memory usage by keeping only top transitions.
	// Default: 50.
	MaxTransitionsPerItem int

	// MinConfidence is the minimum interaction confidence threshold.
	// Default: 0.1.
	MinConfidence float64

	// SessionWindow is the maximum time between items to consider them sequential.
	// Items watched more than this apart are considered separate sessions.
	// Default: 6 hours (21600 seconds).
	SessionWindowSeconds int64

	// SmoothingAlpha is the Laplace smoothing parameter.
	// Adds this value to all transition counts to handle unseen transitions.
	// Default: 0.1.
	SmoothingAlpha float64
}

// DefaultMarkovChainConfig returns default Markov Chain configuration.
func DefaultMarkovChainConfig() MarkovChainConfig {
	return MarkovChainConfig{
		OrderK:                1,
		MinTransitionCount:    2,
		MaxTransitionsPerItem: 50,
		MinConfidence:         0.1,
		SessionWindowSeconds:  21600, // 6 hours
		SmoothingAlpha:        0.1,
	}
}

// MarkovChain implements a simple Markov chain for sequential recommendation.
//
// This algorithm models viewing behavior as a sequence where the probability
// of watching the next item depends only on the previous item(s). It answers
// the question: "Given that a user just watched item X, what should they watch next?"
//
// For first-order chains (OrderK=1):
//
//	P(next_item | last_item) = count(last_item -> next_item) / count(last_item -> any)
//
// Features:
//   - Simple and fast: O(1) prediction time after training
//   - Low memory: Only stores transition counts
//   - Session-aware: Groups viewing sequences into sessions
//   - Smoothing: Handles unseen transitions gracefully
//
// Use cases:
//   - "What to watch next" after completing content
//   - Episode progression (next episode in series)
//   - Content journey analysis
//
// Compared to FPMC: This is much simpler (no matrix factorization) but
// captures explicit sequential patterns. FPMC learns personalized
// latent transitions while this uses global transition probabilities.
type MarkovChain struct {
	BaseAlgorithm
	config MarkovChainConfig

	// itemIndex maps item ID to internal index
	itemIndex map[int]int

	// indexToItem maps internal index to item ID
	indexToItem []int

	// transitions[from_item] = sorted list of (to_item, probability)
	// Only stores top MaxTransitionsPerItem transitions per item
	transitions map[int][]transition

	// itemCounts tracks how many times each item appears as a "from" item
	itemCounts map[int]int

	// totalItems is the vocabulary size for smoothing
	totalItems int
}

// transition represents a transition probability.
type transition struct {
	toItem int
	prob   float64
	count  int
}

// NewMarkovChain creates a new Markov Chain algorithm.
func NewMarkovChain(cfg MarkovChainConfig) *MarkovChain {
	if cfg.OrderK <= 0 {
		cfg.OrderK = 1
	}
	if cfg.MinTransitionCount <= 0 {
		cfg.MinTransitionCount = 2
	}
	if cfg.MaxTransitionsPerItem <= 0 {
		cfg.MaxTransitionsPerItem = 50
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = 0.1
	}
	if cfg.SessionWindowSeconds <= 0 {
		cfg.SessionWindowSeconds = 21600 // 6 hours
	}
	if cfg.SmoothingAlpha <= 0 {
		cfg.SmoothingAlpha = 0.1
	}

	return &MarkovChain{
		BaseAlgorithm: NewBaseAlgorithm("markov_chain"),
		config:        cfg,
		itemIndex:     make(map[int]int),
		transitions:   make(map[int][]transition),
		itemCounts:    make(map[int]int),
	}
}

// Train builds the transition probability matrix from sequential interactions.
//
//nolint:gocyclo,gocritic // gocyclo: sequence processing is complex; gocritic: rangeValCopy acceptable
func (m *MarkovChain) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	m.acquireTrainLock()
	defer m.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Reset state
	m.itemIndex = make(map[int]int)
	m.indexToItem = nil
	m.transitions = make(map[int][]transition)
	m.itemCounts = make(map[int]int)

	// Filter and sort interactions by user and timestamp
	type userInteraction struct {
		userID     int
		itemID     int
		timestamp  int64 // Unix seconds
		confidence float64
	}

	var filtered []userInteraction
	for _, inter := range interactions {
		if inter.Confidence < m.config.MinConfidence {
			continue
		}

		// Register item
		if _, ok := m.itemIndex[inter.ItemID]; !ok {
			m.itemIndex[inter.ItemID] = len(m.indexToItem)
			m.indexToItem = append(m.indexToItem, inter.ItemID)
		}

		ts := int64(0)
		if !inter.Timestamp.IsZero() {
			ts = inter.Timestamp.Unix()
		}

		filtered = append(filtered, userInteraction{
			userID:     inter.UserID,
			itemID:     inter.ItemID,
			timestamp:  ts,
			confidence: inter.Confidence,
		})
	}

	m.totalItems = len(m.indexToItem)
	if m.totalItems == 0 {
		m.markTrained()
		return nil
	}

	// Sort by user, then by timestamp
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].userID != filtered[j].userID {
			return filtered[i].userID < filtered[j].userID
		}
		return filtered[i].timestamp < filtered[j].timestamp
	})

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Count transitions
	// transitionCounts[from][to] = count
	transitionCounts := make(map[int]map[int]int)

	// Process user sequences
	var prevUserID int
	var prevItemIdx int
	var prevTimestamp int64
	first := true

	for _, inter := range filtered {
		itemIdx := m.itemIndex[inter.itemID]

		if first || inter.userID != prevUserID {
			// New user, start fresh sequence
			prevUserID = inter.userID
			prevItemIdx = itemIdx
			prevTimestamp = inter.timestamp
			first = false
			continue
		}

		// Check if this is part of the same session
		timeDiff := inter.timestamp - prevTimestamp
		if timeDiff > m.config.SessionWindowSeconds {
			// Session break, don't count transition
			prevItemIdx = itemIdx
			prevTimestamp = inter.timestamp
			continue
		}

		// Count transition from prev to current
		if transitionCounts[prevItemIdx] == nil {
			transitionCounts[prevItemIdx] = make(map[int]int)
		}
		transitionCounts[prevItemIdx][itemIdx]++
		m.itemCounts[prevItemIdx]++

		prevItemIdx = itemIdx
		prevTimestamp = inter.timestamp
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Convert counts to probabilities and prune
	m.buildTransitionProbabilities(transitionCounts)

	m.markTrained()
	return nil
}

// buildTransitionProbabilities converts counts to probabilities.
func (m *MarkovChain) buildTransitionProbabilities(transitionCounts map[int]map[int]int) {
	for fromIdx, toCounts := range transitionCounts {
		totalCount := m.itemCounts[fromIdx]
		if totalCount == 0 {
			continue
		}

		// Collect transitions above threshold
		var trans []transition
		for toIdx, count := range toCounts {
			if count < m.config.MinTransitionCount {
				continue
			}

			// Probability with Laplace smoothing
			// P(to|from) = (count + alpha) / (total + alpha * V)
			prob := (float64(count) + m.config.SmoothingAlpha) /
				(float64(totalCount) + m.config.SmoothingAlpha*float64(m.totalItems))

			trans = append(trans, transition{
				toItem: toIdx,
				prob:   prob,
				count:  count,
			})
		}

		// Sort by probability (descending)
		sort.Slice(trans, func(i, j int) bool {
			return trans[i].prob > trans[j].prob
		})

		// Keep top K
		if len(trans) > m.config.MaxTransitionsPerItem {
			trans = trans[:m.config.MaxTransitionsPerItem]
		}

		m.transitions[fromIdx] = trans
	}
}

// Predict returns scores for candidate items based on user's recent history.
// Uses the last item in user's history for first-order Markov prediction.
func (m *MarkovChain) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	m.acquirePredictLock()
	defer m.releasePredictLock()

	if !m.trained || len(m.transitions) == 0 {
		return nil, nil
	}

	// For personalized prediction, we need the user's recent items
	// Since this algorithm focuses on "what to watch next", we use
	// a global model. For personalization, see FPMC.
	// Return empty - this algorithm is better used with PredictNext
	return nil, nil
}

// PredictNext returns the most likely next items given a specific item.
// This is the primary use case for Markov chains.
func (m *MarkovChain) PredictNext(ctx context.Context, lastItemID int, candidates []int) (map[int]float64, error) {
	m.acquirePredictLock()
	defer m.releasePredictLock()

	if !m.trained {
		return nil, nil
	}

	lastIdx, ok := m.itemIndex[lastItemID]
	if !ok {
		return nil, nil
	}

	trans := m.transitions[lastIdx]
	if len(trans) == 0 {
		return nil, nil
	}

	// Build candidate set for filtering
	candidateSet := make(map[int]struct{}, len(candidates))
	for _, id := range candidates {
		if idx, ok := m.itemIndex[id]; ok {
			candidateSet[idx] = struct{}{}
		}
	}

	// Filter to candidates only
	scores := make(map[int]float64)
	for _, t := range trans {
		if _, ok := candidateSet[t.toItem]; ok {
			if t.toItem < len(m.indexToItem) {
				scores[m.indexToItem[t.toItem]] = t.prob
			}
		}
	}

	// Already probabilities, but normalize for consistency
	return normalizeScores(scores), nil
}

// PredictSimilar returns items that are likely to follow the given item.
func (m *MarkovChain) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	return m.PredictNext(ctx, itemID, candidates)
}

// GetTransitionCount returns the total number of stored transitions.
func (m *MarkovChain) GetTransitionCount() int {
	m.acquirePredictLock()
	defer m.releasePredictLock()

	count := 0
	for _, trans := range m.transitions {
		count += len(trans)
	}
	return count
}

// GetTopTransitions returns the top-K most common transitions for an item.
func (m *MarkovChain) GetTopTransitions(itemID int, k int) []struct {
	NextItem    int
	Probability float64
} {
	m.acquirePredictLock()
	defer m.releasePredictLock()

	idx, ok := m.itemIndex[itemID]
	if !ok {
		return nil
	}

	trans := m.transitions[idx]
	if len(trans) == 0 {
		return nil
	}

	if k > len(trans) {
		k = len(trans)
	}

	result := make([]struct {
		NextItem    int
		Probability float64
	}, k)

	for i := 0; i < k; i++ {
		if trans[i].toItem < len(m.indexToItem) {
			result[i].NextItem = m.indexToItem[trans[i].toItem]
			result[i].Probability = trans[i].prob
		}
	}

	return result
}

// GetOrderK returns the configured Markov chain order.
func (m *MarkovChain) GetOrderK() int {
	return m.config.OrderK
}

// Ensure interface compliance.
var _ recommend.Algorithm = (*MarkovChain)(nil)
