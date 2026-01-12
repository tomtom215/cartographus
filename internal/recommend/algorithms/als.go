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

// ALSConfig contains configuration for the ALS algorithm.
type ALSConfig struct {
	// NumFactors is the dimension of the latent factor vectors.
	// Typical range: 50-200.
	NumFactors int

	// NumIterations is the number of ALS iterations to run.
	// Typical range: 10-50.
	NumIterations int

	// Regularization is the L2 regularization parameter.
	// Higher values prevent overfitting but may underfit.
	// Typical range: 0.01-0.1.
	Regularization float64

	// Alpha scales the confidence transformation for implicit feedback.
	// c = 1 + alpha * r, where r is the implicit rating.
	// Typical range: 1-100.
	Alpha float64

	// MinConfidence is the minimum confidence threshold for interactions.
	MinConfidence float64

	// NumWorkers is the number of parallel workers for training.
	// If <= 0, defaults to 4.
	NumWorkers int
}

// DefaultALSConfig returns default ALS configuration.
func DefaultALSConfig() ALSConfig {
	return ALSConfig{
		NumFactors:     64,
		NumIterations:  15,
		Regularization: 0.01,
		Alpha:          40.0,
		MinConfidence:  0.1,
		NumWorkers:     4,
	}
}

// ALS implements the Alternating Least Squares algorithm for implicit feedback.
// Reference: "Collaborative Filtering for Implicit Feedback Datasets" (Hu, Koren, Volinsky, 2008)
//
// The algorithm factorizes the user-item interaction matrix into user and item
// latent factor matrices. For implicit feedback, it uses a confidence-weighted
// optimization where higher confidence interactions have more weight.
//
// The objective function minimizes:
// sum_{u,i} c_ui * (p_ui - x_u' * y_i)^2 + lambda * (||x_u||^2 + ||y_i||^2)
//
// where p_ui = 1 if user u interacted with item i, 0 otherwise,
// and c_ui = 1 + alpha * r_ui is the confidence.
type ALS struct {
	BaseAlgorithm
	config ALSConfig

	// X is the user factor matrix (numUsers x numFactors)
	X [][]float64

	// Y is the item factor matrix (numItems x numFactors)
	Y [][]float64

	// userIndex maps user ID to matrix row
	userIndex map[int]int

	// itemIndex maps item ID to matrix row
	itemIndex map[int]int

	// indexToUser maps matrix row to user ID
	indexToUser []int

	// indexToItem maps matrix row to item ID
	indexToItem []int
}

// NewALS creates a new ALS algorithm with the given configuration.
func NewALS(cfg ALSConfig) *ALS {
	if cfg.NumFactors <= 0 {
		cfg.NumFactors = 64
	}
	if cfg.NumIterations <= 0 {
		cfg.NumIterations = 15
	}
	if cfg.Regularization <= 0 {
		cfg.Regularization = 0.01
	}
	if cfg.Alpha <= 0 {
		cfg.Alpha = 40.0
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = 0.1
	}
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 4
	}

	return &ALS{
		BaseAlgorithm: NewBaseAlgorithm("als"),
		config:        cfg,
		userIndex:     make(map[int]int),
		itemIndex:     make(map[int]int),
	}
}

// Train fits the ALS model using alternating optimization.
//
//nolint:gocyclo,gocritic // gocyclo: ML training algorithms are inherently complex; gocritic: rangeValCopy is acceptable for clarity
func (a *ALS) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	a.acquireTrainLock()
	defer a.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Build user and item indices
	a.userIndex = make(map[int]int)
	a.itemIndex = make(map[int]int)
	a.indexToUser = nil
	a.indexToItem = nil

	for _, inter := range interactions {
		if inter.Confidence < a.config.MinConfidence {
			continue
		}
		if _, ok := a.userIndex[inter.UserID]; !ok {
			a.userIndex[inter.UserID] = len(a.indexToUser)
			a.indexToUser = append(a.indexToUser, inter.UserID)
		}
		if _, ok := a.itemIndex[inter.ItemID]; !ok {
			a.itemIndex[inter.ItemID] = len(a.indexToItem)
			a.indexToItem = append(a.indexToItem, inter.ItemID)
		}
	}

	numUsers := len(a.indexToUser)
	numItems := len(a.indexToItem)
	numFactors := a.config.NumFactors

	if numUsers == 0 || numItems == 0 {
		a.markTrained()
		return nil
	}

	// Build confidence matrix (sparse representation)
	// C[u][i] = 1 + alpha * r[u][i]
	userItems := make(map[int]map[int]float64)
	for _, inter := range interactions {
		if inter.Confidence < a.config.MinConfidence {
			continue
		}
		ui := a.userIndex[inter.UserID]
		ii := a.itemIndex[inter.ItemID]
		if userItems[ui] == nil {
			userItems[ui] = make(map[int]float64)
		}
		// Use max confidence for duplicates
		conf := 1.0 + a.config.Alpha*inter.Confidence
		if conf > userItems[ui][ii] {
			userItems[ui][ii] = conf
		}
	}

	// Transpose for item-to-user access
	itemUsers := make(map[int]map[int]float64)
	for ui, itemMap := range userItems {
		for ii, conf := range itemMap {
			if itemUsers[ii] == nil {
				itemUsers[ii] = make(map[int]float64)
			}
			itemUsers[ii][ui] = conf
		}
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Initialize factor matrices randomly
	a.X = make([][]float64, numUsers)
	a.Y = make([][]float64, numItems)

	for u := 0; u < numUsers; u++ {
		a.X[u] = make([]float64, numFactors)
		for f := 0; f < numFactors; f++ {
			// Small random initialization
			a.X[u][f] = 0.1 * (float64((u*numFactors+f)%1000)/1000.0 - 0.5)
		}
	}

	for i := 0; i < numItems; i++ {
		a.Y[i] = make([]float64, numFactors)
		for f := 0; f < numFactors; f++ {
			a.Y[i][f] = 0.1 * (float64((i*numFactors+f)%1000)/1000.0 - 0.5)
		}
	}

	// Alternating optimization
	lambda := a.config.Regularization

	for iter := 0; iter < a.config.NumIterations; iter++ {
		if ContextCancelled(ctx) {
			return ctx.Err()
		}

		// Update user factors (fix Y, solve for X)
		a.updateUserFactors(userItems, numUsers, numItems, numFactors, lambda)

		if ContextCancelled(ctx) {
			return ctx.Err()
		}

		// Update item factors (fix X, solve for Y)
		a.updateItemFactors(itemUsers, numUsers, numItems, numFactors, lambda)
	}

	a.markTrained()
	return nil
}

// updateUserFactors updates all user factor vectors.
func (a *ALS) updateUserFactors(userItems map[int]map[int]float64, numUsers, numItems, numFactors int, lambda float64) {
	// Precompute Y'Y
	YtY := make([][]float64, numFactors)
	for f := range YtY {
		YtY[f] = make([]float64, numFactors)
	}
	for i := 0; i < numItems; i++ {
		for f1 := 0; f1 < numFactors; f1++ {
			for f2 := f1; f2 < numFactors; f2++ {
				YtY[f1][f2] += a.Y[i][f1] * a.Y[i][f2]
				if f1 != f2 {
					YtY[f2][f1] = YtY[f1][f2]
				}
			}
		}
	}

	var wg sync.WaitGroup
	chunkSize := (numUsers + a.config.NumWorkers - 1) / a.config.NumWorkers

	for w := 0; w < a.config.NumWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > numUsers {
			end = numUsers
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(uStart, uEnd int) {
			defer wg.Done()

			for u := uStart; u < uEnd; u++ {
				a.updateSingleUser(u, userItems[u], YtY, numFactors, lambda)
			}
		}(start, end)
	}

	wg.Wait()
}

// updateSingleUser updates factors for a single user.
//
//nolint:gocritic // YtY follows standard linear algebra notation
func (a *ALS) updateSingleUser(u int, items map[int]float64, YtY [][]float64, numFactors int, lambda float64) {
	// A = Y' * C^u * Y + lambda * I
	// b = Y' * C^u * p^u
	// x_u = A^(-1) * b

	// Start with Y'Y + lambda*I
	A := make([][]float64, numFactors)
	for f := range A {
		A[f] = make([]float64, numFactors)
		copy(A[f], YtY[f])
		A[f][f] += lambda
	}

	// Add confidence-weighted contributions
	b := make([]float64, numFactors)
	for i, conf := range items {
		// A += (c_ui - 1) * y_i * y_i'
		// b += c_ui * y_i
		y := a.Y[i]
		cMinus1 := conf - 1.0

		for f1 := 0; f1 < numFactors; f1++ {
			for f2 := f1; f2 < numFactors; f2++ {
				delta := cMinus1 * y[f1] * y[f2]
				A[f1][f2] += delta
				if f1 != f2 {
					A[f2][f1] += delta
				}
			}
			b[f1] += conf * y[f1]
		}
	}

	// Solve A * x = b using Cholesky
	x := solveLinearSystem(A, b)
	a.X[u] = x
}

// updateItemFactors updates all item factor vectors.
func (a *ALS) updateItemFactors(itemUsers map[int]map[int]float64, numUsers, numItems, numFactors int, lambda float64) {
	// Precompute X'X
	XtX := make([][]float64, numFactors)
	for f := range XtX {
		XtX[f] = make([]float64, numFactors)
	}
	for u := 0; u < numUsers; u++ {
		for f1 := 0; f1 < numFactors; f1++ {
			for f2 := f1; f2 < numFactors; f2++ {
				XtX[f1][f2] += a.X[u][f1] * a.X[u][f2]
				if f1 != f2 {
					XtX[f2][f1] = XtX[f1][f2]
				}
			}
		}
	}

	var wg sync.WaitGroup
	chunkSize := (numItems + a.config.NumWorkers - 1) / a.config.NumWorkers

	for w := 0; w < a.config.NumWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > numItems {
			end = numItems
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(iStart, iEnd int) {
			defer wg.Done()

			for i := iStart; i < iEnd; i++ {
				a.updateSingleItem(i, itemUsers[i], XtX, numFactors, lambda)
			}
		}(start, end)
	}

	wg.Wait()
}

// updateSingleItem updates factors for a single item.
//
//nolint:gocritic // XtX follows standard linear algebra notation
func (a *ALS) updateSingleItem(i int, users map[int]float64, XtX [][]float64, numFactors int, lambda float64) {
	// A = X' * C^i * X + lambda * I
	// b = X' * C^i * p^i
	// y_i = A^(-1) * b

	A := make([][]float64, numFactors)
	for f := range A {
		A[f] = make([]float64, numFactors)
		copy(A[f], XtX[f])
		A[f][f] += lambda
	}

	b := make([]float64, numFactors)
	for u, conf := range users {
		x := a.X[u]
		cMinus1 := conf - 1.0

		for f1 := 0; f1 < numFactors; f1++ {
			for f2 := f1; f2 < numFactors; f2++ {
				delta := cMinus1 * x[f1] * x[f2]
				A[f1][f2] += delta
				if f1 != f2 {
					A[f2][f1] += delta
				}
			}
			b[f1] += conf * x[f1]
		}
	}

	y := solveLinearSystem(A, b)
	a.Y[i] = y
}

// solveLinearSystem solves A*x = b using Cholesky decomposition.
//
//nolint:gocritic // A, L follow standard linear algebra notation
func solveLinearSystem(A [][]float64, b []float64) []float64 {
	n := len(b)

	// Cholesky decomposition: A = L * L'
	L := make([][]float64, n)
	for i := range L {
		L[i] = make([]float64, n)
	}

	for i := 0; i < n; i++ {
		for j := 0; j <= i; j++ {
			sum := A[i][j]
			for k := 0; k < j; k++ {
				sum -= L[i][k] * L[j][k]
			}

			if i == j {
				if sum <= 0 {
					// Add regularization if not positive definite
					sum = 1e-10
				}
				L[i][j] = math.Sqrt(sum)
			} else {
				if L[j][j] != 0 {
					L[i][j] = sum / L[j][j]
				}
			}
		}
	}

	// Solve L * z = b (forward substitution)
	z := make([]float64, n)
	for i := 0; i < n; i++ {
		sum := b[i]
		for j := 0; j < i; j++ {
			sum -= L[i][j] * z[j]
		}
		if L[i][i] != 0 {
			z[i] = sum / L[i][i]
		}
	}

	// Solve L' * x = z (back substitution)
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		sum := z[i]
		for j := i + 1; j < n; j++ {
			sum -= L[j][i] * x[j]
		}
		if L[i][i] != 0 {
			x[i] = sum / L[i][i]
		}
	}

	return x
}

// Predict returns scores for candidate items for a user.
func (a *ALS) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	a.acquirePredictLock()
	defer a.releasePredictLock()

	if !a.trained || len(a.X) == 0 || len(a.Y) == 0 {
		return nil, nil
	}

	ui, ok := a.userIndex[userID]
	if !ok {
		return nil, nil
	}

	userVec := a.X[ui]
	scores := make(map[int]float64, len(candidates))

	for _, itemID := range candidates {
		ii, ok := a.itemIndex[itemID]
		if !ok {
			continue
		}

		// score = x_u' * y_i
		var score float64
		for f := range userVec {
			score += userVec[f] * a.Y[ii][f]
		}
		scores[itemID] = score
	}

	return normalizeScores(scores), nil
}

// PredictSimilar returns items similar to the given item.
func (a *ALS) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	a.acquirePredictLock()
	defer a.releasePredictLock()

	if !a.trained || len(a.Y) == 0 {
		return nil, nil
	}

	sourceIdx, ok := a.itemIndex[itemID]
	if !ok {
		return nil, nil
	}

	sourceVec := a.Y[sourceIdx]
	scores := make(map[int]float64, len(candidates))

	for _, candidateID := range candidates {
		if candidateID == itemID {
			continue
		}

		candidateIdx, ok := a.itemIndex[candidateID]
		if !ok {
			continue
		}

		// Cosine similarity between item vectors
		score := cosineSimilarity(sourceVec, a.Y[candidateIdx])
		if score > 0 {
			scores[candidateID] = score
		}
	}

	return normalizeScores(scores), nil
}

// GetUserFactors returns a copy of user factors.
func (a *ALS) GetUserFactors() [][]float64 {
	a.acquirePredictLock()
	defer a.releasePredictLock()

	if a.X == nil {
		return nil
	}

	result := make([][]float64, len(a.X))
	for i := range a.X {
		result[i] = make([]float64, len(a.X[i]))
		copy(result[i], a.X[i])
	}
	return result
}

// GetItemFactors returns a copy of item factors.
func (a *ALS) GetItemFactors() [][]float64 {
	a.acquirePredictLock()
	defer a.releasePredictLock()

	if a.Y == nil {
		return nil
	}

	result := make([][]float64, len(a.Y))
	for i := range a.Y {
		result[i] = make([]float64, len(a.Y[i]))
		copy(result[i], a.Y[i])
	}
	return result
}

// Ensure interface compliance.
var _ recommend.Algorithm = (*ALS)(nil)
