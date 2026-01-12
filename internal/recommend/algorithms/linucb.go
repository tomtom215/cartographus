// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"math"
	"sync"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// LinUCBConfig contains configuration for the LinUCB algorithm.
type LinUCBConfig struct {
	// Alpha controls exploration vs exploitation.
	// Higher values = more exploration.
	// Typical range: 0.1-2.0.
	Alpha float64

	// NumFeatures is the dimension of context feature vectors.
	// Typical range: 10-100.
	NumFeatures int

	// ContextBuilder specifies how to build context features.
	// Options: "simple", "hybrid", "full".
	ContextBuilder string

	// MinObservations is the minimum observations needed before using UCB.
	// Below this, items are selected uniformly.
	MinObservations int

	// DecayRate is the rate at which old observations decay.
	// 0 = no decay, 1 = full decay. Typical: 0.001-0.01.
	DecayRate float64
}

// DefaultLinUCBConfig returns default LinUCB configuration.
func DefaultLinUCBConfig() LinUCBConfig {
	return LinUCBConfig{
		Alpha:           1.0,
		NumFeatures:     32,
		ContextBuilder:  "simple",
		MinObservations: 10,
		DecayRate:       0.001,
	}
}

// LinUCB implements the Linear Upper Confidence Bound algorithm for contextual bandits.
// Reference: "A Contextual-Bandit Approach to Personalized News Article Recommendation" (Li et al., 2010)
//
// LinUCB learns to balance exploration (trying new items) with exploitation
// (recommending known good items) using contextual information.
//
// For each item (arm), we maintain:
// - A matrix: inverse of the regularized design matrix
// - b vector: weighted sum of contexts times rewards
//
// The UCB score for item i given context x is:
// UCB(i) = theta_i' * x + alpha * sqrt(x' * A_i^(-1) * x)
//
// where theta_i = A_i^(-1) * b_i is the estimated weight vector.
type LinUCB struct {
	BaseAlgorithm
	config LinUCBConfig

	// Per-arm model parameters
	// A[itemID] is the design matrix (NumFeatures x NumFeatures)
	A map[int][][]float64

	// b[itemID] is the reward-weighted context sum (NumFeatures)
	b map[int][]float64

	// observations[itemID] counts observations for each item
	observations map[int]int

	// totalObservations counts all observations
	totalObservations int

	// userFeatures stores precomputed user feature vectors
	userFeatures map[int][]float64

	// itemFeatures stores precomputed item feature vectors
	itemFeatures map[int][]float64

	mu sync.RWMutex
}

// NewLinUCB creates a new LinUCB algorithm.
func NewLinUCB(cfg LinUCBConfig) *LinUCB {
	if cfg.Alpha <= 0 {
		cfg.Alpha = 1.0
	}
	if cfg.NumFeatures <= 0 {
		cfg.NumFeatures = 32
	}
	if cfg.MinObservations <= 0 {
		cfg.MinObservations = 10
	}

	return &LinUCB{
		BaseAlgorithm: NewBaseAlgorithm("linucb"),
		config:        cfg,
		A:             make(map[int][][]float64),
		b:             make(map[int][]float64),
		observations:  make(map[int]int),
		userFeatures:  make(map[int][]float64),
		itemFeatures:  make(map[int][]float64),
	}
}

// Train initializes the LinUCB model from historical interactions.
//
//nolint:gocritic // rangeValCopy: Item/Interaction passed by value in range, acceptable for clarity
func (l *LinUCB) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	l.acquireTrainLock()
	defer l.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	d := l.config.NumFeatures

	// Build feature vectors for items
	l.itemFeatures = make(map[int][]float64)
	for _, item := range items {
		l.itemFeatures[item.ID] = l.buildItemFeatures(item)
	}

	// Build user feature vectors from interactions
	l.userFeatures = make(map[int][]float64)
	userItems := make(map[int][]recommend.Interaction)
	for _, inter := range interactions {
		userItems[inter.UserID] = append(userItems[inter.UserID], inter)
	}

	for userID, inters := range userItems {
		l.userFeatures[userID] = l.buildUserFeatures(userID, inters, items)
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Initialize per-arm models
	l.A = make(map[int][][]float64)
	l.b = make(map[int][]float64)
	l.observations = make(map[int]int)
	l.totalObservations = 0

	// Warm-start from historical interactions
	for _, inter := range interactions {
		if ContextCancelled(ctx) {
			return ctx.Err()
		}

		itemID := inter.ItemID

		// Initialize arm if not exists
		if _, ok := l.A[itemID]; !ok {
			l.A[itemID] = identityMatrix(d)
			l.b[itemID] = make([]float64, d)
		}

		// Get context features
		userFeat := l.userFeatures[inter.UserID]
		if len(userFeat) == 0 {
			userFeat = make([]float64, d)
		}

		// Update with reward (confidence as reward signal)
		reward := inter.Confidence
		l.updateArm(itemID, userFeat, reward)
	}

	l.markTrained()
	return nil
}

// buildItemFeatures creates feature vector for an item.
//
//nolint:gocritic // hugeParam: Item passed by value for immutability
func (l *LinUCB) buildItemFeatures(item recommend.Item) []float64 {
	d := l.config.NumFeatures
	features := make([]float64, d)

	// Simple feature encoding
	// First few features: one-hot for common genres
	genreMap := map[string]int{
		"Action": 0, "Comedy": 1, "Drama": 2, "Horror": 3,
		"Sci-Fi": 4, "Romance": 5, "Thriller": 6, "Documentary": 7,
	}

	for _, genre := range item.Genres {
		if idx, ok := genreMap[genre]; ok && idx < d/4 {
			features[idx] = 1.0
		}
	}

	// Year features (normalized)
	if item.Year > 0 {
		normalizedYear := float64(item.Year-1970) / 60.0 // Normalize to ~[0,1]
		if d > 8 {
			features[8] = normalizedYear
		}
	}

	// Rating features
	if d > 9 && item.Rating > 0 {
		features[9] = item.Rating / 10.0
	}
	if d > 10 && item.AudienceRating > 0 {
		features[10] = item.AudienceRating / 10.0
	}

	return features
}

// buildUserFeatures creates feature vector for a user based on their history.
//
//nolint:gocritic // rangeValCopy: Item passed by value in range, acceptable for clarity
func (l *LinUCB) buildUserFeatures(_ int, interactions []recommend.Interaction, items []recommend.Item) []float64 {
	d := l.config.NumFeatures
	features := make([]float64, d)

	// Build item lookup
	itemMap := make(map[int]recommend.Item)
	for _, item := range items {
		itemMap[item.ID] = item
	}

	// Aggregate item features weighted by confidence
	var totalWeight float64
	for _, inter := range interactions {
		item, ok := itemMap[inter.ItemID]
		if !ok {
			continue
		}

		itemFeat := l.buildItemFeatures(item)
		weight := inter.Confidence
		totalWeight += weight

		for i := range features {
			features[i] += weight * itemFeat[i]
		}
	}

	// Normalize
	if totalWeight > 0 {
		for i := range features {
			features[i] /= totalWeight
		}
	}

	return features
}

// updateArm updates the model for a single arm with a new observation.
func (l *LinUCB) updateArm(itemID int, context []float64, reward float64) {
	d := l.config.NumFeatures
	if len(context) != d {
		return
	}

	// Apply decay if configured
	if l.config.DecayRate > 0 {
		decay := 1.0 - l.config.DecayRate
		for i := range l.A[itemID] {
			for j := range l.A[itemID][i] {
				if i == j {
					l.A[itemID][i][j] = decay*(l.A[itemID][i][j]-1.0) + 1.0
				} else {
					l.A[itemID][i][j] *= decay
				}
			}
		}
		for i := range l.b[itemID] {
			l.b[itemID][i] *= decay
		}
	}

	// A = A + x * x'
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			l.A[itemID][i][j] += context[i] * context[j]
		}
	}

	// b = b + reward * x
	for i := 0; i < d; i++ {
		l.b[itemID][i] += reward * context[i]
	}

	l.observations[itemID]++
	l.totalObservations++
}

// Predict returns scores for candidate items for a user.
func (l *LinUCB) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	l.acquirePredictLock()
	defer l.releasePredictLock()

	if !l.trained {
		return nil, nil
	}

	// Get user context
	context := l.userFeatures[userID]
	if len(context) == 0 {
		context = make([]float64, l.config.NumFeatures)
	}

	scores := make(map[int]float64, len(candidates))

	for _, itemID := range candidates {
		score := l.computeUCB(itemID, context)
		if score > 0 {
			scores[itemID] = score
		}
	}

	return normalizeScores(scores), nil
}

// computeUCB computes the Upper Confidence Bound for an item.
func (l *LinUCB) computeUCB(itemID int, context []float64) float64 {
	d := l.config.NumFeatures

	A, okA := l.A[itemID]
	b, okB := l.b[itemID]

	if !okA || !okB {
		// New item - use exploration bonus only
		return l.config.Alpha * math.Sqrt(float64(l.totalObservations+1))
	}

	// Check if context is all zeros (cold-start user)
	isZeroContext := true
	for _, v := range context {
		if v != 0 {
			isZeroContext = false
			break
		}
	}

	if isZeroContext {
		// Cold-start user: provide exploration bonus based on item uncertainty
		// Use inverse of observation count as uncertainty proxy
		obs := l.observations[itemID]
		if obs > 0 {
			// More observations = lower uncertainty = lower score
			// This encourages exploration of less-observed items
			return l.config.Alpha * math.Sqrt(float64(l.totalObservations+1)/float64(obs+1))
		}
		return l.config.Alpha * math.Sqrt(float64(l.totalObservations+1))
	}

	// Compute A^(-1) (we should cache this, but for simplicity compute inline)
	Ainv := invertMatrix(A)
	if Ainv == nil {
		return 0
	}

	// theta = A^(-1) * b
	theta := make([]float64, d)
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			theta[i] += Ainv[i][j] * b[j]
		}
	}

	// Expected reward: theta' * x
	var expectedReward float64
	for i := 0; i < d; i++ {
		expectedReward += theta[i] * context[i]
	}

	// Exploration bonus: alpha * sqrt(x' * A^(-1) * x)
	var variance float64
	for i := 0; i < d; i++ {
		var temp float64
		for j := 0; j < d; j++ {
			temp += Ainv[i][j] * context[j]
		}
		variance += context[i] * temp
	}

	explorationBonus := l.config.Alpha * math.Sqrt(math.Max(variance, 0))

	return expectedReward + explorationBonus
}

// PredictSimilar returns items similar to the given item.
func (l *LinUCB) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	l.acquirePredictLock()
	defer l.releasePredictLock()

	if !l.trained {
		return nil, nil
	}

	sourceFeatures, ok := l.itemFeatures[itemID]
	if !ok {
		return nil, nil
	}

	scores := make(map[int]float64, len(candidates))

	for _, candidateID := range candidates {
		if candidateID == itemID {
			continue
		}

		candidateFeatures, ok := l.itemFeatures[candidateID]
		if !ok {
			continue
		}

		// Cosine similarity between item features
		sim := cosineSimilarity(sourceFeatures, candidateFeatures)
		if sim > 0 {
			scores[candidateID] = sim
		}
	}

	return normalizeScores(scores), nil
}

// RecordFeedback updates the model with new feedback.
// This enables online learning.
func (l *LinUCB) RecordFeedback(userID int, itemID int, reward float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	d := l.config.NumFeatures

	// Initialize arm if not exists
	if _, ok := l.A[itemID]; !ok {
		l.A[itemID] = identityMatrix(d)
		l.b[itemID] = make([]float64, d)
	}

	// Get user context
	context := l.userFeatures[userID]
	if len(context) == 0 {
		context = make([]float64, d)
	}

	l.updateArm(itemID, context, reward)
}

// GetExplorationRate returns the current exploration rate.
func (l *LinUCB) GetExplorationRate() float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.totalObservations == 0 {
		return 1.0
	}

	// Exploration rate decreases with more observations
	return l.config.Alpha / math.Sqrt(float64(l.totalObservations))
}

// identityMatrix creates an n x n identity matrix.
func identityMatrix(n int) [][]float64 {
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n)
		m[i][i] = 1.0
	}
	return m
}

// invertMatrix computes the inverse of a matrix using Gaussian elimination.
//
//nolint:gocritic // A follows standard linear algebra notation
func invertMatrix(A [][]float64) [][]float64 {
	n := len(A)
	if n == 0 {
		return nil
	}

	// Create augmented matrix [A|I]
	augmented := make([][]float64, n)
	for i := range augmented {
		augmented[i] = make([]float64, 2*n)
		copy(augmented[i], A[i])
		augmented[i][n+i] = 1.0
	}

	// Forward elimination
	for i := 0; i < n; i++ {
		// Find pivot
		maxRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(augmented[k][i]) > math.Abs(augmented[maxRow][i]) {
				maxRow = k
			}
		}
		augmented[i], augmented[maxRow] = augmented[maxRow], augmented[i]

		// Check for singular matrix
		if math.Abs(augmented[i][i]) < 1e-10 {
			// Add regularization
			augmented[i][i] = 1e-10
		}

		// Eliminate column
		for k := i + 1; k < n; k++ {
			factor := augmented[k][i] / augmented[i][i]
			for j := i; j < 2*n; j++ {
				augmented[k][j] -= factor * augmented[i][j]
			}
		}
	}

	// Back substitution
	for i := n - 1; i >= 0; i-- {
		pivot := augmented[i][i]
		for j := i; j < 2*n; j++ {
			augmented[i][j] /= pivot
		}
		for k := 0; k < i; k++ {
			factor := augmented[k][i]
			for j := i; j < 2*n; j++ {
				augmented[k][j] -= factor * augmented[i][j]
			}
		}
	}

	// Extract inverse
	inv := make([][]float64, n)
	for i := range inv {
		inv[i] = make([]float64, n)
		copy(inv[i], augmented[i][n:])
	}

	return inv
}

// Ensure interface compliance.
var _ recommend.Algorithm = (*LinUCB)(nil)
