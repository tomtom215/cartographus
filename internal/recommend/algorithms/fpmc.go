// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"math"
	"sort"
	"sync"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// FPMCConfig contains configuration for the FPMC algorithm.
type FPMCConfig struct {
	// NumFactors is the dimension of latent factor vectors.
	// Typical range: 32-128.
	NumFactors int

	// LearningRate is the SGD learning rate.
	// Typical range: 0.01-0.1.
	LearningRate float64

	// Regularization is the L2 regularization parameter.
	// Typical range: 0.001-0.1.
	Regularization float64

	// NumIterations is the number of training epochs.
	// Typical range: 10-50.
	NumIterations int

	// NegativeSamples is the number of negative samples per positive.
	// Typical range: 1-5.
	NegativeSamples int

	// MaxHistory is the maximum history length to consider.
	// Typical range: 5-20.
	MaxHistory int

	// NumWorkers for parallel training.
	NumWorkers int
}

// DefaultFPMCConfig returns default FPMC configuration.
func DefaultFPMCConfig() FPMCConfig {
	return FPMCConfig{
		NumFactors:      64,
		LearningRate:    0.01,
		Regularization:  0.001,
		NumIterations:   20,
		NegativeSamples: 3,
		MaxHistory:      10,
		NumWorkers:      4,
	}
}

// FPMC implements Factorized Personalized Markov Chains.
// Reference: "Factorizing Personalized Markov Chains for Next-Basket Recommendation" (Rendle et al., 2010)
//
// FPMC combines matrix factorization (for user preferences) with
// Markov chains (for sequential patterns) to predict the next item.
//
// The model factorizes:
// - User-Item matrix (general preferences): U * I_U
// - Item-Item transition matrix (Markov): I_L * I_I (last item to next item)
//
// Score: x(u, l, i) = <V_U^u, V_I_U^i> + <V_I_L^l, V_I_I^i>
type FPMC struct {
	BaseAlgorithm
	config FPMCConfig

	// VU is the user factor matrix (numUsers x numFactors)
	VU [][]float64

	// VIU is the item factor matrix for user preferences (numItems x numFactors)
	VIU [][]float64

	// VIL is the item factor matrix for "last item" (numItems x numFactors)
	VIL [][]float64

	// VII is the item factor matrix for "next item" in transitions (numItems x numFactors)
	VII [][]float64

	// userIndex maps user ID to matrix index
	userIndex map[int]int

	// itemIndex maps item ID to matrix index
	itemIndex map[int]int

	// indexToItem maps matrix index to item ID
	indexToItem []int

	// userHistory stores recent item sequences per user
	userHistory map[int][]int

	// allItems is the list of all item indices for negative sampling
	allItems []int

	mu sync.RWMutex
}

// NewFPMC creates a new FPMC algorithm.
func NewFPMC(cfg FPMCConfig) *FPMC {
	if cfg.NumFactors <= 0 {
		cfg.NumFactors = 64
	}
	if cfg.LearningRate <= 0 {
		cfg.LearningRate = 0.01
	}
	if cfg.Regularization <= 0 {
		cfg.Regularization = 0.001
	}
	if cfg.NumIterations <= 0 {
		cfg.NumIterations = 20
	}
	if cfg.NegativeSamples <= 0 {
		cfg.NegativeSamples = 3
	}
	if cfg.MaxHistory <= 0 {
		cfg.MaxHistory = 10
	}
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 4
	}

	return &FPMC{
		BaseAlgorithm: NewBaseAlgorithm("fpmc"),
		config:        cfg,
		userIndex:     make(map[int]int),
		itemIndex:     make(map[int]int),
		userHistory:   make(map[int][]int),
	}
}

// Train fits the FPMC model using BPR-based optimization.
//
//nolint:gocyclo,gocritic // gocyclo: ML training algorithms are inherently complex; gocritic: rangeValCopy is acceptable for clarity
func (f *FPMC) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	f.acquireTrainLock()
	defer f.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Build indices
	f.userIndex = make(map[int]int)
	f.itemIndex = make(map[int]int)
	f.indexToItem = nil
	f.allItems = nil

	for _, inter := range interactions {
		if _, ok := f.userIndex[inter.UserID]; !ok {
			f.userIndex[inter.UserID] = len(f.userIndex)
		}
		if _, ok := f.itemIndex[inter.ItemID]; !ok {
			idx := len(f.indexToItem)
			f.itemIndex[inter.ItemID] = idx
			f.indexToItem = append(f.indexToItem, inter.ItemID)
			f.allItems = append(f.allItems, idx)
		}
	}

	numUsers := len(f.userIndex)
	numItems := len(f.itemIndex)
	numFactors := f.config.NumFactors

	if numUsers == 0 || numItems == 0 {
		f.markTrained()
		return nil
	}

	// Build user sequences (ordered by timestamp)
	userSequences := make(map[int][]userItem)
	for _, inter := range interactions {
		uid := f.userIndex[inter.UserID]
		iid := f.itemIndex[inter.ItemID]
		userSequences[uid] = append(userSequences[uid], userItem{
			itemIdx:   iid,
			timestamp: inter.Timestamp.UnixNano(),
		})
	}

	// Sort sequences by timestamp
	for uid := range userSequences {
		seq := userSequences[uid]
		sort.Slice(seq, func(i, j int) bool {
			return seq[i].timestamp < seq[j].timestamp
		})
		userSequences[uid] = seq
	}

	// Store user history for prediction
	f.userHistory = make(map[int][]int)
	for userID, uid := range f.userIndex {
		seq := userSequences[uid]
		history := make([]int, 0, len(seq))
		for _, ui := range seq {
			history = append(history, ui.itemIdx)
		}
		// Keep only recent history
		if len(history) > f.config.MaxHistory {
			history = history[len(history)-f.config.MaxHistory:]
		}
		f.userHistory[userID] = history
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Initialize factor matrices
	f.VU = f.initMatrix(numUsers, numFactors)
	f.VIU = f.initMatrix(numItems, numFactors)
	f.VIL = f.initMatrix(numItems, numFactors)
	f.VII = f.initMatrix(numItems, numFactors)

	// Generate training samples (user, last_item, next_item)
	samples := make([]trainingSample, 0)
	for uid, seq := range userSequences {
		for i := 1; i < len(seq); i++ {
			samples = append(samples, trainingSample{
				userIdx: uid,
				lastIdx: seq[i-1].itemIdx,
				nextIdx: seq[i].itemIdx,
			})
		}
	}

	if len(samples) == 0 {
		f.markTrained()
		return nil
	}

	// BPR training
	for iter := 0; iter < f.config.NumIterations; iter++ {
		if ContextCancelled(ctx) {
			return ctx.Err()
		}

		// Shuffle samples (deterministic based on iteration)
		shuffleIdx := (iter * 31) % len(samples)
		for i := range samples {
			j := (i + shuffleIdx) % len(samples)
			samples[i], samples[j] = samples[j], samples[i]
		}

		for _, sample := range samples {
			// Positive sample
			f.updateBPR(sample.userIdx, sample.lastIdx, sample.nextIdx, true)

			// Negative samples
			for n := 0; n < f.config.NegativeSamples; n++ {
				negIdx := f.sampleNegative(sample.userIdx, userSequences)
				f.updateBPR(sample.userIdx, sample.lastIdx, negIdx, false)
			}
		}
	}

	f.markTrained()
	return nil
}

type userItem struct {
	itemIdx   int
	timestamp int64
}

type trainingSample struct {
	userIdx int
	lastIdx int
	nextIdx int
}

// initMatrix initializes a matrix with small random values.
func (f *FPMC) initMatrix(rows, cols int) [][]float64 {
	matrix := make([][]float64, rows)
	for i := range matrix {
		matrix[i] = make([]float64, cols)
		for j := range matrix[i] {
			// Deterministic initialization based on position
			matrix[i][j] = 0.1 * (float64((i*cols+j)%1000)/1000.0 - 0.5)
		}
	}
	return matrix
}

// sampleNegative samples a negative item (one the user hasn't interacted with).
func (f *FPMC) sampleNegative(userIdx int, userSequences map[int][]userItem) int {
	positives := make(map[int]struct{})
	for _, ui := range userSequences[userIdx] {
		positives[ui.itemIdx] = struct{}{}
	}

	// Simple sampling - pick item not in user's history
	for _, idx := range f.allItems {
		if _, ok := positives[idx]; !ok {
			return idx
		}
	}

	// Fallback: return random item
	return f.allItems[0]
}

// updateBPR performs one BPR update step.
func (f *FPMC) updateBPR(userIdx, lastIdx, itemIdx int, positive bool) {
	lr := f.config.LearningRate
	reg := f.config.Regularization

	// Compute score difference for BPR
	// x_uij = x(u, l, i) - x(u, l, j)
	score := f.computeScore(userIdx, lastIdx, itemIdx)

	// Sigmoid gradient: d/dx sigmoid(x) = sigmoid(x) * (1 - sigmoid(x))
	sigmoid := 1.0 / (1.0 + math.Exp(-score))
	if !positive {
		sigmoid = 1 - sigmoid
	}
	gradient := sigmoid * (1 - sigmoid)

	// Update factor matrices
	for k := range f.VU[userIdx] {
		// User preference factors
		vuGrad := gradient * f.VIU[itemIdx][k]
		viuGrad := gradient * f.VU[userIdx][k]

		f.VU[userIdx][k] += lr * (vuGrad - reg*f.VU[userIdx][k])
		f.VIU[itemIdx][k] += lr * (viuGrad - reg*f.VIU[itemIdx][k])

		// Markov transition factors
		vilGrad := gradient * f.VII[itemIdx][k]
		viiGrad := gradient * f.VIL[lastIdx][k]

		f.VIL[lastIdx][k] += lr * (vilGrad - reg*f.VIL[lastIdx][k])
		f.VII[itemIdx][k] += lr * (viiGrad - reg*f.VII[itemIdx][k])
	}
}

// computeScore computes the FPMC score for (user, last_item, next_item).
func (f *FPMC) computeScore(userIdx, lastIdx, itemIdx int) float64 {
	var userScore, transScore float64

	for k := range f.VU[userIdx] {
		userScore += f.VU[userIdx][k] * f.VIU[itemIdx][k]
		transScore += f.VIL[lastIdx][k] * f.VII[itemIdx][k]
	}

	return userScore + transScore
}

// Predict returns scores for candidate items for a user.
func (f *FPMC) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	f.acquirePredictLock()
	defer f.releasePredictLock()

	if !f.trained || len(f.VU) == 0 {
		return nil, nil
	}

	userIdx, ok := f.userIndex[userID]
	if !ok {
		return nil, nil
	}

	history := f.userHistory[userID]
	if len(history) == 0 {
		return nil, nil
	}

	// Use the last item in history for Markov component
	lastIdx := history[len(history)-1]

	scores := make(map[int]float64, len(candidates))

	for _, itemID := range candidates {
		itemIdx, ok := f.itemIndex[itemID]
		if !ok {
			continue
		}

		score := f.computeScore(userIdx, lastIdx, itemIdx)
		scores[itemID] = score
	}

	return normalizeScores(scores), nil
}

// PredictSimilar returns items similar to the given item.
// Uses item-item transition similarity.
func (f *FPMC) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	f.acquirePredictLock()
	defer f.releasePredictLock()

	if !f.trained || len(f.VIL) == 0 {
		return nil, nil
	}

	sourceIdx, ok := f.itemIndex[itemID]
	if !ok {
		return nil, nil
	}

	scores := make(map[int]float64, len(candidates))

	for _, candidateID := range candidates {
		if candidateID == itemID {
			continue
		}

		candidateIdx, ok := f.itemIndex[candidateID]
		if !ok {
			continue
		}

		// Transition score: how likely to go from source to candidate
		var score float64
		for k := range f.VIL[sourceIdx] {
			score += f.VIL[sourceIdx][k] * f.VII[candidateIdx][k]
		}

		if score > 0 {
			scores[candidateID] = score
		}
	}

	return normalizeScores(scores), nil
}

// PredictNext returns the most likely next items given recent history.
func (f *FPMC) PredictNext(ctx context.Context, userID int, recentItems []int, candidates []int) (map[int]float64, error) {
	f.acquirePredictLock()
	defer f.releasePredictLock()

	if !f.trained || len(f.VU) == 0 {
		return nil, nil
	}

	userIdx, ok := f.userIndex[userID]
	if !ok {
		// New user - use only Markov component
		return f.predictNewUser(recentItems, candidates), nil
	}

	if len(recentItems) == 0 {
		return nil, nil
	}

	// Get last item index
	lastItemID := recentItems[len(recentItems)-1]
	lastIdx, ok := f.itemIndex[lastItemID]
	if !ok {
		return nil, nil
	}

	scores := make(map[int]float64, len(candidates))

	for _, itemID := range candidates {
		itemIdx, ok := f.itemIndex[itemID]
		if !ok {
			continue
		}

		score := f.computeScore(userIdx, lastIdx, itemIdx)
		scores[itemID] = score
	}

	return normalizeScores(scores), nil
}

// predictNewUser predicts for new users using only the Markov component.
func (f *FPMC) predictNewUser(recentItems []int, candidates []int) map[int]float64 {
	if len(recentItems) == 0 {
		return nil
	}

	lastItemID := recentItems[len(recentItems)-1]
	lastIdx, ok := f.itemIndex[lastItemID]
	if !ok {
		return nil
	}

	scores := make(map[int]float64, len(candidates))

	for _, itemID := range candidates {
		itemIdx, ok := f.itemIndex[itemID]
		if !ok {
			continue
		}

		// Only Markov component
		var score float64
		for k := range f.VIL[lastIdx] {
			score += f.VIL[lastIdx][k] * f.VII[itemIdx][k]
		}

		if score > 0 {
			scores[itemID] = score
		}
	}

	return normalizeScores(scores)
}

// GetUserHistory returns the stored history for a user.
func (f *FPMC) GetUserHistory(userID int) []int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if history, ok := f.userHistory[userID]; ok {
		result := make([]int, len(history))
		// Convert indices back to item IDs
		for i, idx := range history {
			if idx < len(f.indexToItem) {
				result[i] = f.indexToItem[idx]
			}
		}
		return result
	}
	return nil
}

// Ensure interface compliance.
var _ recommend.Algorithm = (*FPMC)(nil)
