// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// EASEConfig contains configuration for the EASE algorithm.
type EASEConfig struct {
	// L2Regularization is the L2 regularization parameter (lambda).
	// Higher values produce more conservative recommendations.
	// Typical range: 100-1000.
	L2Regularization float64

	// MinConfidence is the minimum confidence threshold for interactions.
	// Interactions below this threshold are treated as zero.
	MinConfidence float64
}

// DefaultEASEConfig returns default EASE configuration.
func DefaultEASEConfig() EASEConfig {
	return EASEConfig{
		L2Regularization: 500.0,
		MinConfidence:    0.1,
	}
}

// EASE implements the Embarrassingly Shallow Autoencoders algorithm.
// Reference: "Embarrassingly Shallow Autoencoders for Sparse Data" (Steck, 2019)
//
// EASE is a linear model that learns item-item similarity weights through
// closed-form optimization. It's computationally efficient and produces
// high-quality recommendations with minimal hyperparameter tuning.
//
// The model computes: score(u, i) = sum_j(X[u,j] * B[j,i]) for j != i
// where X is the user-item interaction matrix and B is the learned weight matrix.
type EASE struct {
	BaseAlgorithm
	config EASEConfig

	// B is the item-item weight matrix (B[i][j] = weight from item i to j)
	B [][]float64

	// itemIndex maps item ID to matrix index
	itemIndex map[int]int

	// indexToItem maps matrix index to item ID
	indexToItem []int

	// userVectors stores precomputed user interaction vectors
	userVectors map[int][]float64
}

// NewEASE creates a new EASE algorithm with the given configuration.
func NewEASE(cfg EASEConfig) *EASE {
	if cfg.L2Regularization <= 0 {
		cfg.L2Regularization = 500.0
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = 0.1
	}

	return &EASE{
		BaseAlgorithm: NewBaseAlgorithm("ease"),
		config:        cfg,
		itemIndex:     make(map[int]int),
		userVectors:   make(map[int][]float64),
	}
}

// Train fits the EASE model on interaction data.
// This computes the closed-form solution: B = (X'X + λI)^(-1) * X'X
// with the diagonal set to zero (preventing self-recommendations).
//
//nolint:gocyclo,gocritic // gocyclo: ML training algorithms are inherently complex; gocritic: rangeValCopy is acceptable for clarity
func (e *EASE) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	e.acquireTrainLock()
	defer e.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Build item index
	e.itemIndex = make(map[int]int)
	e.indexToItem = make([]int, 0)

	itemSet := make(map[int]struct{})
	for _, inter := range interactions {
		if inter.Confidence >= e.config.MinConfidence {
			itemSet[inter.ItemID] = struct{}{}
		}
	}
	for _, item := range items {
		if _, ok := itemSet[item.ID]; ok {
			continue
		}
		itemSet[item.ID] = struct{}{}
	}

	for id := range itemSet {
		e.itemIndex[id] = len(e.indexToItem)
		e.indexToItem = append(e.indexToItem, id)
	}

	numItems := len(e.indexToItem)
	if numItems == 0 {
		e.markTrained()
		return nil
	}

	// Build user-item interaction matrix
	userItemMap := make(map[int]map[int]float64)
	for _, inter := range interactions {
		if inter.Confidence < e.config.MinConfidence {
			continue
		}
		if userItemMap[inter.UserID] == nil {
			userItemMap[inter.UserID] = make(map[int]float64)
		}
		// Keep highest confidence for duplicate interactions
		if c := userItemMap[inter.UserID][inter.ItemID]; inter.Confidence > c {
			userItemMap[inter.UserID][inter.ItemID] = inter.Confidence
		}
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Compute X'X (Gram matrix)
	G := make([][]float64, numItems)
	for i := range G {
		G[i] = make([]float64, numItems)
	}

	// Store user vectors for prediction
	e.userVectors = make(map[int][]float64)

	for userID, userItems := range userItemMap {
		// Build user vector
		userVec := make([]float64, numItems)
		for itemID, conf := range userItems {
			if idx, ok := e.itemIndex[itemID]; ok {
				userVec[idx] = conf
			}
		}
		e.userVectors[userID] = userVec

		// Add outer product to Gram matrix
		for i := 0; i < numItems; i++ {
			if userVec[i] == 0 {
				continue
			}
			for j := i; j < numItems; j++ {
				if userVec[j] == 0 {
					continue
				}
				G[i][j] += userVec[i] * userVec[j]
				if i != j {
					G[j][i] = G[i][j]
				}
			}
		}

		if ContextCancelled(ctx) {
			return ctx.Err()
		}
	}

	// Add L2 regularization to diagonal
	lambda := e.config.L2Regularization
	for i := 0; i < numItems; i++ {
		G[i][i] += lambda
	}

	// Compute inverse using Cholesky decomposition
	L, err := choleskyDecomposition(G)
	if err != nil {
		// Fall back to pseudo-inverse if Cholesky fails
		L = computePseudoInverseSimple(G)
	} else {
		L = choleskyInverse(L)
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Compute B = P * (G + λI)^(-1) where P = G (the Gram matrix before regularization)
	// But we need to set diagonal to zero and normalize
	e.B = make([][]float64, numItems)
	for i := range e.B {
		e.B[i] = make([]float64, numItems)
	}

	// Original Gram matrix (without regularization)
	origG := make([][]float64, numItems)
	for i := range origG {
		origG[i] = make([]float64, numItems)
		copy(origG[i], G[i])
		origG[i][i] -= lambda // Remove regularization from diagonal
	}

	// Multiply: B = origG * L (where L is the inverse)
	for i := 0; i < numItems; i++ {
		for j := 0; j < numItems; j++ {
			if i == j {
				e.B[i][j] = 0 // Self-similarity is zero
				continue
			}
			var sum float64
			for k := 0; k < numItems; k++ {
				sum += origG[i][k] * L[k][j]
			}
			e.B[i][j] = sum
		}
	}

	// Set diagonal to zero (self-recommendation prevention)
	for i := 0; i < numItems; i++ {
		e.B[i][i] = 0
	}

	e.markTrained()
	return nil
}

// Predict returns scores for candidate items for a user.
func (e *EASE) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	e.acquirePredictLock()
	defer e.releasePredictLock()

	if !e.trained || len(e.B) == 0 {
		return nil, nil
	}

	userVec, ok := e.userVectors[userID]
	if !ok {
		return nil, nil
	}

	scores := make(map[int]float64, len(candidates))
	for _, itemID := range candidates {
		itemIdx, ok := e.itemIndex[itemID]
		if !ok {
			continue
		}

		// score = sum_j(userVec[j] * B[j][itemIdx])
		var score float64
		for j, v := range userVec {
			if v > 0 {
				score += v * e.B[j][itemIdx]
			}
		}
		scores[itemID] = score
	}

	return normalizeScores(scores), nil
}

// PredictSimilar returns items similar to the given item.
func (e *EASE) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	e.acquirePredictLock()
	defer e.releasePredictLock()

	if !e.trained || len(e.B) == 0 {
		return nil, nil
	}

	sourceIdx, ok := e.itemIndex[itemID]
	if !ok {
		return nil, nil
	}

	scores := make(map[int]float64, len(candidates))
	for _, candidateID := range candidates {
		if candidateID == itemID {
			continue
		}
		candidateIdx, ok := e.itemIndex[candidateID]
		if !ok {
			continue
		}

		// Use symmetric similarity from B matrix
		score := (e.B[sourceIdx][candidateIdx] + e.B[candidateIdx][sourceIdx]) / 2
		if score > 0 {
			scores[candidateID] = score
		}
	}

	return normalizeScores(scores), nil
}

// GetWeightMatrix returns a copy of the learned weight matrix.
func (e *EASE) GetWeightMatrix() [][]float64 {
	e.acquirePredictLock()
	defer e.releasePredictLock()

	if e.B == nil {
		return nil
	}

	result := make([][]float64, len(e.B))
	for i := range e.B {
		result[i] = make([]float64, len(e.B[i]))
		copy(result[i], e.B[i])
	}
	return result
}

// GetItemIndex returns the item ID to index mapping.
func (e *EASE) GetItemIndex() map[int]int {
	e.acquirePredictLock()
	defer e.releasePredictLock()

	result := make(map[int]int, len(e.itemIndex))
	for k, v := range e.itemIndex {
		result[k] = v
	}
	return result
}

// choleskyDecomposition computes the Cholesky decomposition L of a symmetric positive-definite matrix A
// such that A = L * L'.
//
//nolint:gocritic // A, L follow standard linear algebra notation
func choleskyDecomposition(A [][]float64) ([][]float64, error) {
	n := len(A)
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
					return nil, fmt.Errorf("matrix is not positive definite")
				}
				L[i][j] = math.Sqrt(sum)
			} else {
				if L[j][j] == 0 {
					return nil, fmt.Errorf("division by zero in Cholesky")
				}
				L[i][j] = sum / L[j][j]
			}
		}
	}

	return L, nil
}

// choleskyInverse computes the inverse of a matrix given its Cholesky decomposition.
//
//nolint:gocritic // L follows standard linear algebra notation
func choleskyInverse(L [][]float64) [][]float64 {
	n := len(L)

	// First compute L inverse
	Linv := make([][]float64, n)
	for i := range Linv {
		Linv[i] = make([]float64, n)
	}

	for i := 0; i < n; i++ {
		Linv[i][i] = 1.0 / L[i][i]
		for j := i + 1; j < n; j++ {
			var sum float64
			for k := i; k < j; k++ {
				sum -= L[j][k] * Linv[k][i]
			}
			Linv[j][i] = sum / L[j][j]
		}
	}

	// Compute A^(-1) = L^(-T) * L^(-1)
	inv := make([][]float64, n)
	for i := range inv {
		inv[i] = make([]float64, n)
	}

	for i := 0; i < n; i++ {
		for j := 0; j <= i; j++ {
			var sum float64
			for k := i; k < n; k++ {
				sum += Linv[k][i] * Linv[k][j]
			}
			inv[i][j] = sum
			inv[j][i] = sum
		}
	}

	return inv
}

// computePseudoInverseSimple computes a simplified pseudo-inverse using iterative refinement.
// This is a fallback when Cholesky fails.
//
//nolint:gocritic // A follows standard linear algebra notation
func computePseudoInverseSimple(A [][]float64) [][]float64 {
	n := len(A)

	// Start with scaled identity
	inv := make([][]float64, n)
	for i := range inv {
		inv[i] = make([]float64, n)
		// Use diagonal scaling for initial guess
		if A[i][i] != 0 {
			inv[i][i] = 1.0 / A[i][i]
		} else {
			inv[i][i] = 1.0
		}
	}

	// Newton-Schulz iteration: X_{k+1} = X_k * (2I - A*X_k)
	// Converges quadratically for suitable starting point
	temp := make([][]float64, n)
	for i := range temp {
		temp[i] = make([]float64, n)
	}

	for iter := 0; iter < 10; iter++ {
		// Compute A * inv
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				var sum float64
				for k := 0; k < n; k++ {
					sum += A[i][k] * inv[k][j]
				}
				temp[i][j] = sum
			}
		}

		// Compute 2I - A*inv
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if i == j {
					temp[i][j] = 2.0 - temp[i][j]
				} else {
					temp[i][j] = -temp[i][j]
				}
			}
		}

		// Compute inv * (2I - A*inv)
		newInv := make([][]float64, n)
		for i := range newInv {
			newInv[i] = make([]float64, n)
			for j := 0; j < n; j++ {
				var sum float64
				for k := 0; k < n; k++ {
					sum += inv[i][k] * temp[k][j]
				}
				newInv[i][j] = sum
			}
		}

		inv = newInv
	}

	return inv
}

// Ensure interface compliance.
var _ recommend.Algorithm = (*EASE)(nil)

// EASEParallel is a parallel version of EASE for larger datasets.
type EASEParallel struct {
	*EASE
	numWorkers int
}

// NewEASEParallel creates a new parallel EASE algorithm.
func NewEASEParallel(cfg EASEConfig, numWorkers int) *EASEParallel {
	if numWorkers <= 0 {
		numWorkers = 4
	}
	return &EASEParallel{
		EASE:       NewEASE(cfg),
		numWorkers: numWorkers,
	}
}

// Train fits the EASE model using parallel computation.
//
//nolint:gocyclo // ML training algorithms are inherently complex
func (e *EASEParallel) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	e.acquireTrainLock()
	defer e.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Build item index
	e.itemIndex = make(map[int]int)
	e.indexToItem = make([]int, 0)

	itemSet := make(map[int]struct{})
	for _, inter := range interactions {
		if inter.Confidence >= e.config.MinConfidence {
			itemSet[inter.ItemID] = struct{}{}
		}
	}

	for id := range itemSet {
		e.itemIndex[id] = len(e.indexToItem)
		e.indexToItem = append(e.indexToItem, id)
	}

	numItems := len(e.indexToItem)
	if numItems == 0 {
		e.markTrained()
		return nil
	}

	// Build user-item interaction matrix
	userItemMap := make(map[int]map[int]float64)
	for _, inter := range interactions {
		if inter.Confidence < e.config.MinConfidence {
			continue
		}
		if userItemMap[inter.UserID] == nil {
			userItemMap[inter.UserID] = make(map[int]float64)
		}
		if c := userItemMap[inter.UserID][inter.ItemID]; inter.Confidence > c {
			userItemMap[inter.UserID][inter.ItemID] = inter.Confidence
		}
	}

	// Compute X'X in parallel
	G := make([][]float64, numItems)
	for i := range G {
		G[i] = make([]float64, numItems)
	}

	e.userVectors = make(map[int][]float64)
	var mu sync.Mutex

	// Create user vector jobs
	users := make([]int, 0, len(userItemMap))
	for userID := range userItemMap {
		users = append(users, userID)
	}

	// Process users in parallel
	var wg sync.WaitGroup
	chunkSize := (len(users) + e.numWorkers - 1) / e.numWorkers

	for w := 0; w < e.numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(users) {
			end = len(users)
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(userSlice []int) {
			defer wg.Done()

			localG := make([][]float64, numItems)
			for i := range localG {
				localG[i] = make([]float64, numItems)
			}
			localVecs := make(map[int][]float64)

			for _, userID := range userSlice {
				if ContextCancelled(ctx) {
					return
				}

				userItems := userItemMap[userID]
				userVec := make([]float64, numItems)
				for itemID, conf := range userItems {
					if idx, ok := e.itemIndex[itemID]; ok {
						userVec[idx] = conf
					}
				}
				localVecs[userID] = userVec

				for i := 0; i < numItems; i++ {
					if userVec[i] == 0 {
						continue
					}
					for j := i; j < numItems; j++ {
						if userVec[j] == 0 {
							continue
						}
						localG[i][j] += userVec[i] * userVec[j]
					}
				}
			}

			// Merge results
			mu.Lock()
			for i := 0; i < numItems; i++ {
				for j := i; j < numItems; j++ {
					G[i][j] += localG[i][j]
				}
			}
			for userID, vec := range localVecs {
				e.userVectors[userID] = vec
			}
			mu.Unlock()
		}(users[start:end])
	}

	wg.Wait()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Fill lower triangle
	for i := 0; i < numItems; i++ {
		for j := 0; j < i; j++ {
			G[i][j] = G[j][i]
		}
	}

	// Add L2 regularization
	lambda := e.config.L2Regularization
	for i := 0; i < numItems; i++ {
		G[i][i] += lambda
	}

	// Compute inverse
	L, err := choleskyDecomposition(G)
	if err != nil {
		L = computePseudoInverseSimple(G)
	} else {
		L = choleskyInverse(L)
	}

	// Compute B matrix
	e.B = make([][]float64, numItems)
	for i := range e.B {
		e.B[i] = make([]float64, numItems)
	}

	// Restore original G (without regularization)
	for i := 0; i < numItems; i++ {
		G[i][i] -= lambda
	}

	// B = G * L with zero diagonal (parallel)
	wg = sync.WaitGroup{}
	rowChunk := (numItems + e.numWorkers - 1) / e.numWorkers

	for w := 0; w < e.numWorkers; w++ {
		start := w * rowChunk
		end := start + rowChunk
		if end > numItems {
			end = numItems
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(rowStart, rowEnd int) {
			defer wg.Done()

			for i := rowStart; i < rowEnd; i++ {
				for j := 0; j < numItems; j++ {
					if i == j {
						continue
					}
					var sum float64
					for k := 0; k < numItems; k++ {
						sum += G[i][k] * L[k][j]
					}
					e.B[i][j] = sum
				}
			}
		}(start, end)
	}

	wg.Wait()

	e.markTrained()
	return nil
}
