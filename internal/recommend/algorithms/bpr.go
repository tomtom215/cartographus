// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"math"
	"math/rand"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// BPRConfig contains configuration for the BPR algorithm.
type BPRConfig struct {
	// NumFactors is the dimension of the latent factor vectors.
	// Typical range: 32-128. Lower values use less memory.
	// Default: 64.
	NumFactors int

	// LearningRate is the SGD step size.
	// Typical range: 0.001-0.1.
	// Default: 0.01.
	LearningRate float64

	// Regularization is the L2 regularization parameter.
	// Higher values prevent overfitting.
	// Default: 0.01.
	Regularization float64

	// NumIterations is the number of training epochs.
	// Each epoch samples numInteractions triplets.
	// Default: 100.
	NumIterations int

	// NumNegativeSamples is how many negative samples per positive.
	// Higher values improve coverage but slow training.
	// Default: 5.
	NumNegativeSamples int

	// MinConfidence is the minimum confidence threshold for interactions.
	// Default: 0.1.
	MinConfidence float64

	// Seed for reproducible training.
	// If 0, uses a default seed.
	Seed int64
}

// DefaultBPRConfig returns default BPR configuration.
func DefaultBPRConfig() BPRConfig {
	return BPRConfig{
		NumFactors:         64,
		LearningRate:       0.01,
		Regularization:     0.01,
		NumIterations:      100,
		NumNegativeSamples: 5,
		MinConfidence:      0.1,
		Seed:               42,
	}
}

// BPR implements Bayesian Personalized Ranking for implicit feedback.
// Reference: "BPR: Bayesian Personalized Ranking from Implicit Feedback"
// (Rendle, Freudenthaler, Gantner, Schmidt-Thieme, 2009)
//
// BPR optimizes for pairwise ranking using SGD. For each user, it samples
// a positive item (interacted) and negative item (not interacted), then
// optimizes to rank the positive item higher.
//
// The objective maximizes: sum_{u,i,j} ln(sigmoid(x_uij)) - lambda*||theta||^2
// where x_uij = score(u,i) - score(u,j) and theta are model parameters.
//
// This implementation uses matrix factorization as the underlying model:
// score(u,i) = user_factors[u] dot item_factors[i]
type BPR struct {
	BaseAlgorithm
	config BPRConfig

	// userFactors is the user latent factor matrix (numUsers x numFactors)
	userFactors [][]float64

	// itemFactors is the item latent factor matrix (numItems x numFactors)
	itemFactors [][]float64

	// userIndex maps user ID to matrix row
	userIndex map[int]int

	// itemIndex maps item ID to matrix row
	itemIndex map[int]int

	// indexToUser maps matrix row to user ID
	indexToUser []int

	// indexToItem maps matrix row to item ID
	indexToItem []int

	// userItems stores the set of items each user has interacted with
	// Used for efficient negative sampling
	userItems map[int]map[int]struct{}

	// allItems is the list of all item indices for negative sampling
	allItems []int
}

// NewBPR creates a new BPR algorithm with the given configuration.
func NewBPR(cfg BPRConfig) *BPR {
	if cfg.NumFactors <= 0 {
		cfg.NumFactors = 64
	}
	if cfg.LearningRate <= 0 {
		cfg.LearningRate = 0.01
	}
	if cfg.Regularization <= 0 {
		cfg.Regularization = 0.01
	}
	if cfg.NumIterations <= 0 {
		cfg.NumIterations = 100
	}
	if cfg.NumNegativeSamples <= 0 {
		cfg.NumNegativeSamples = 5
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = 0.1
	}
	if cfg.Seed == 0 {
		cfg.Seed = 42
	}

	return &BPR{
		BaseAlgorithm: NewBaseAlgorithm("bpr"),
		config:        cfg,
		userIndex:     make(map[int]int),
		itemIndex:     make(map[int]int),
		userItems:     make(map[int]map[int]struct{}),
	}
}

// Train fits the BPR model using stochastic gradient descent.
//
//nolint:gocyclo,gocritic // gocyclo: ML training algorithms are inherently complex; gocritic: rangeValCopy is acceptable
func (b *BPR) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	b.acquireTrainLock()
	defer b.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Build user and item indices
	b.userIndex = make(map[int]int)
	b.itemIndex = make(map[int]int)
	b.indexToUser = nil
	b.indexToItem = nil
	b.userItems = make(map[int]map[int]struct{})
	b.allItems = nil

	// First pass: collect users and items
	for _, inter := range interactions {
		if inter.Confidence < b.config.MinConfidence {
			continue
		}

		if _, ok := b.userIndex[inter.UserID]; !ok {
			b.userIndex[inter.UserID] = len(b.indexToUser)
			b.indexToUser = append(b.indexToUser, inter.UserID)
			b.userItems[b.userIndex[inter.UserID]] = make(map[int]struct{})
		}
		if _, ok := b.itemIndex[inter.ItemID]; !ok {
			b.itemIndex[inter.ItemID] = len(b.indexToItem)
			b.indexToItem = append(b.indexToItem, inter.ItemID)
		}
	}

	numUsers := len(b.indexToUser)
	numItems := len(b.indexToItem)
	numFactors := b.config.NumFactors

	if numUsers == 0 || numItems == 0 {
		b.markTrained()
		return nil
	}

	// Build allItems list for negative sampling
	b.allItems = make([]int, numItems)
	for i := 0; i < numItems; i++ {
		b.allItems[i] = i
	}

	// Second pass: build user-item interaction sets
	for _, inter := range interactions {
		if inter.Confidence < b.config.MinConfidence {
			continue
		}
		ui := b.userIndex[inter.UserID]
		ii := b.itemIndex[inter.ItemID]
		b.userItems[ui][ii] = struct{}{}
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Initialize factor matrices with small random values
	//nolint:gosec // G404: math/rand is acceptable for ML initialization (not security)
	rng := rand.New(rand.NewSource(b.config.Seed))

	b.userFactors = make([][]float64, numUsers)
	for u := 0; u < numUsers; u++ {
		b.userFactors[u] = make([]float64, numFactors)
		for f := 0; f < numFactors; f++ {
			b.userFactors[u][f] = (rng.Float64() - 0.5) * 0.01
		}
	}

	b.itemFactors = make([][]float64, numItems)
	for i := 0; i < numItems; i++ {
		b.itemFactors[i] = make([]float64, numFactors)
		for f := 0; f < numFactors; f++ {
			b.itemFactors[i][f] = (rng.Float64() - 0.5) * 0.01
		}
	}

	// Build list of all (user, positive_item) pairs for sampling
	type userItemPair struct {
		userIdx int
		itemIdx int
	}
	var positivePairs []userItemPair
	for ui, items := range b.userItems {
		for ii := range items {
			positivePairs = append(positivePairs, userItemPair{ui, ii})
		}
	}

	if len(positivePairs) == 0 {
		b.markTrained()
		return nil
	}

	// Training loop
	lr := b.config.LearningRate
	reg := b.config.Regularization
	numSamples := len(positivePairs)

	for epoch := 0; epoch < b.config.NumIterations; epoch++ {
		if ContextCancelled(ctx) {
			return ctx.Err()
		}

		// Shuffle training pairs each epoch
		rng.Shuffle(len(positivePairs), func(i, j int) {
			positivePairs[i], positivePairs[j] = positivePairs[j], positivePairs[i]
		})

		// Process each positive pair
		for _, pair := range positivePairs {
			u := pair.userIdx
			i := pair.itemIdx
			userItemSet := b.userItems[u]

			// Sample negative items
			for ns := 0; ns < b.config.NumNegativeSamples; ns++ {
				// Sample a random item not in user's history
				var j int
				for tries := 0; tries < 100; tries++ {
					j = rng.Intn(numItems)
					if _, ok := userItemSet[j]; !ok {
						break
					}
				}

				// Skip if we couldn't find a negative sample
				if _, ok := userItemSet[j]; ok {
					continue
				}

				// Compute score difference: x_uij = x_ui - x_uj
				var xui, xuj float64
				for f := 0; f < numFactors; f++ {
					xui += b.userFactors[u][f] * b.itemFactors[i][f]
					xuj += b.userFactors[u][f] * b.itemFactors[j][f]
				}
				xuij := xui - xuj

				// Compute sigmoid gradient
				// d/d_theta ln(sigmoid(x)) = 1 / (1 + exp(x))
				expXuij := math.Exp(xuij)
				sigmoid := 1.0 / (1.0 + expXuij)

				// If sigmoid is too close to 1, skip to avoid numerical issues
				if sigmoid < 1e-10 {
					continue
				}

				// Update user and item factors
				for f := 0; f < numFactors; f++ {
					wuf := b.userFactors[u][f]
					hif := b.itemFactors[i][f]
					hjf := b.itemFactors[j][f]

					// Gradient with respect to user factors
					// d_W_uf = sigmoid * (H_if - H_jf) - reg * W_uf
					b.userFactors[u][f] += lr * (sigmoid*(hif-hjf) - reg*wuf)

					// Gradient with respect to positive item factors
					// d_H_if = sigmoid * W_uf - reg * H_if
					b.itemFactors[i][f] += lr * (sigmoid*wuf - reg*hif)

					// Gradient with respect to negative item factors
					// d_H_jf = -sigmoid * W_uf - reg * H_jf
					b.itemFactors[j][f] += lr * (-sigmoid*wuf - reg*hjf)
				}
			}
		}

		// Adaptive learning rate decay (optional)
		if epoch > 0 && epoch%10 == 0 {
			lr *= 0.95
		}
	}

	_ = numSamples // suppress unused warning
	b.markTrained()
	return nil
}

// Predict returns scores for candidate items for a user.
func (b *BPR) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	b.acquirePredictLock()
	defer b.releasePredictLock()

	if !b.trained || len(b.userFactors) == 0 || len(b.itemFactors) == 0 {
		return nil, nil
	}

	ui, ok := b.userIndex[userID]
	if !ok {
		return nil, nil
	}

	userVec := b.userFactors[ui]
	scores := make(map[int]float64, len(candidates))

	for _, itemID := range candidates {
		ii, ok := b.itemIndex[itemID]
		if !ok {
			continue
		}

		// score = user_factors[u] dot item_factors[i]
		var score float64
		itemVec := b.itemFactors[ii]
		for f := range userVec {
			score += userVec[f] * itemVec[f]
		}
		scores[itemID] = score
	}

	return normalizeScores(scores), nil
}

// PredictSimilar returns items similar to the given item.
func (b *BPR) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	b.acquirePredictLock()
	defer b.releasePredictLock()

	if !b.trained || len(b.itemFactors) == 0 {
		return nil, nil
	}

	sourceIdx, ok := b.itemIndex[itemID]
	if !ok {
		return nil, nil
	}

	sourceVec := b.itemFactors[sourceIdx]
	scores := make(map[int]float64, len(candidates))

	for _, candidateID := range candidates {
		if candidateID == itemID {
			continue
		}

		candidateIdx, ok := b.itemIndex[candidateID]
		if !ok {
			continue
		}

		// Cosine similarity between item vectors
		score := cosineSimilarity(sourceVec, b.itemFactors[candidateIdx])
		if score > 0 {
			scores[candidateID] = score
		}
	}

	return normalizeScores(scores), nil
}

// GetUserFactors returns a copy of user factors (for testing/debugging).
func (b *BPR) GetUserFactors() [][]float64 {
	b.acquirePredictLock()
	defer b.releasePredictLock()

	if b.userFactors == nil {
		return nil
	}

	result := make([][]float64, len(b.userFactors))
	for i := range b.userFactors {
		result[i] = make([]float64, len(b.userFactors[i]))
		copy(result[i], b.userFactors[i])
	}
	return result
}

// GetItemFactors returns a copy of item factors (for testing/debugging).
func (b *BPR) GetItemFactors() [][]float64 {
	b.acquirePredictLock()
	defer b.releasePredictLock()

	if b.itemFactors == nil {
		return nil
	}

	result := make([][]float64, len(b.itemFactors))
	for i := range b.itemFactors {
		result[i] = make([]float64, len(b.itemFactors[i]))
		copy(result[i], b.itemFactors[i])
	}
	return result
}

// Ensure interface compliance.
var _ recommend.Algorithm = (*BPR)(nil)
