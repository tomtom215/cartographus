// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"sort"
	"sync"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// KNNConfig contains configuration for KNN-based algorithms.
type KNNConfig struct {
	// K is the number of neighbors to consider.
	// Typical range: 20-100.
	K int

	// MinSimilarity is the minimum similarity threshold.
	// Neighbors with lower similarity are ignored.
	// Typical range: 0.1-0.3.
	MinSimilarity float64

	// SimilarityMetric specifies which similarity function to use.
	// Options: "cosine", "pearson", "jaccard".
	SimilarityMetric string

	// Shrinkage adds a penalty for pairs with few co-ratings.
	// Regularizes similarity: sim = raw_sim * n / (n + shrinkage)
	// Typical range: 10-100.
	Shrinkage float64

	// MinCommonItems is the minimum number of common items/users
	// required for a valid similarity computation.
	MinCommonItems int

	// NumWorkers is the number of parallel workers.
	NumWorkers int
}

// DefaultKNNConfig returns default KNN configuration.
func DefaultKNNConfig() KNNConfig {
	return KNNConfig{
		K:                50,
		MinSimilarity:    0.1,
		SimilarityMetric: "cosine",
		Shrinkage:        100,
		MinCommonItems:   3,
		NumWorkers:       4,
	}
}

// neighbor represents a similar user or item with their similarity score.
type neighbor struct {
	ID         int
	Similarity float64
}

// ========== User-Based Collaborative Filtering ==========

// UserBasedCF implements user-based collaborative filtering.
// It recommends items that similar users have liked.
//
// For a target user u and candidate item i:
// score(u, i) = sum_{v in N(u)} sim(u, v) * r(v, i) / sum_{v in N(u)} |sim(u, v)|
//
// where N(u) is the set of k most similar users to u who have rated item i.
type UserBasedCF struct {
	BaseAlgorithm
	config KNNConfig

	// userVectors stores user interaction vectors (itemID -> confidence)
	userVectors map[int]map[int]float64

	// userSimilarity stores precomputed user-user similarities
	userSimilarity map[int][]neighbor

	// itemUsers stores which users interacted with each item
	itemUsers map[int][]int
}

// NewUserBasedCF creates a new user-based CF algorithm.
func NewUserBasedCF(cfg KNNConfig) *UserBasedCF {
	if cfg.K <= 0 {
		cfg.K = 50
	}
	if cfg.MinSimilarity <= 0 {
		cfg.MinSimilarity = 0.1
	}
	if cfg.SimilarityMetric == "" {
		cfg.SimilarityMetric = "cosine"
	}
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 4
	}

	return &UserBasedCF{
		BaseAlgorithm:  NewBaseAlgorithm("usercf"),
		config:         cfg,
		userVectors:    make(map[int]map[int]float64),
		userSimilarity: make(map[int][]neighbor),
		itemUsers:      make(map[int][]int),
	}
}

// Train fits the user-based CF model.
//
//nolint:gocritic // rangeValCopy: Interaction passed by value in range, acceptable for clarity
func (u *UserBasedCF) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	u.acquireTrainLock()
	defer u.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Build user vectors
	u.userVectors = make(map[int]map[int]float64)
	u.itemUsers = make(map[int][]int)

	for _, inter := range interactions {
		if u.userVectors[inter.UserID] == nil {
			u.userVectors[inter.UserID] = make(map[int]float64)
		}
		if c := u.userVectors[inter.UserID][inter.ItemID]; inter.Confidence > c {
			u.userVectors[inter.UserID][inter.ItemID] = inter.Confidence
		}
	}

	// Build item-user index
	for userID, itemMap := range u.userVectors {
		for itemID := range itemMap {
			u.itemUsers[itemID] = append(u.itemUsers[itemID], userID)
		}
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Precompute user similarities
	u.userSimilarity = make(map[int][]neighbor)
	userIDs := make([]int, 0, len(u.userVectors))
	for uid := range u.userVectors {
		userIDs = append(userIDs, uid)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	chunkSize := (len(userIDs) + u.config.NumWorkers - 1) / u.config.NumWorkers

	for w := 0; w < u.config.NumWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(userIDs) {
			end = len(userIDs)
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(users []int) {
			defer wg.Done()

			for _, uid := range users {
				if ContextCancelled(ctx) {
					return
				}

				neighbors := u.computeUserNeighbors(uid, userIDs)

				mu.Lock()
				u.userSimilarity[uid] = neighbors
				mu.Unlock()
			}
		}(userIDs[start:end])
	}

	wg.Wait()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	u.markTrained()
	return nil
}

// computeUserNeighbors computes the k most similar users for a given user.
func (u *UserBasedCF) computeUserNeighbors(userID int, allUsers []int) []neighbor {
	userVec := u.userVectors[userID]
	if len(userVec) == 0 {
		return nil
	}

	neighbors := make([]neighbor, 0, len(allUsers))

	for _, otherID := range allUsers {
		if otherID == userID {
			continue
		}

		otherVec := u.userVectors[otherID]
		sim := u.computeSimilarity(userVec, otherVec)

		if sim >= u.config.MinSimilarity {
			neighbors = append(neighbors, neighbor{ID: otherID, Similarity: sim})
		}
	}

	// Sort by similarity (descending) and take top K
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].Similarity > neighbors[j].Similarity
	})

	if len(neighbors) > u.config.K {
		neighbors = neighbors[:u.config.K]
	}

	return neighbors
}

// computeSimilarity computes similarity between two user vectors.
func (u *UserBasedCF) computeSimilarity(a, b map[int]float64) float64 {
	// Find common items
	var commonItems []int
	for item := range a {
		if _, ok := b[item]; ok {
			commonItems = append(commonItems, item)
		}
	}

	if len(commonItems) < u.config.MinCommonItems {
		return 0
	}

	// Compute similarity based on metric
	var sim float64
	switch u.config.SimilarityMetric {
	case "cosine":
		sim = u.cosineSim(a, b, commonItems)
	case "pearson":
		sim = u.pearsonSim(a, b, commonItems)
	case "jaccard":
		sim = float64(len(commonItems)) / float64(len(a)+len(b)-len(commonItems))
	default:
		sim = u.cosineSim(a, b, commonItems)
	}

	// Apply shrinkage
	if u.config.Shrinkage > 0 {
		sim = sim * float64(len(commonItems)) / (float64(len(commonItems)) + u.config.Shrinkage)
	}

	return sim
}

func (u *UserBasedCF) cosineSim(a, b map[int]float64, common []int) float64 {
	var dot, normA, normB float64
	for _, item := range common {
		dot += a[item] * b[item]
	}
	for _, v := range a {
		normA += v * v
	}
	for _, v := range b {
		normB += v * v
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (sqrt(normA) * sqrt(normB))
}

func (u *UserBasedCF) pearsonSim(a, b map[int]float64, common []int) float64 {
	if len(common) == 0 {
		return 0
	}

	// Compute means over common items
	var sumA, sumB float64
	for _, item := range common {
		sumA += a[item]
		sumB += b[item]
	}
	meanA := sumA / float64(len(common))
	meanB := sumB / float64(len(common))

	// Compute Pearson correlation
	var num, denA, denB float64
	for _, item := range common {
		diffA := a[item] - meanA
		diffB := b[item] - meanB
		num += diffA * diffB
		denA += diffA * diffA
		denB += diffB * diffB
	}

	if denA == 0 || denB == 0 {
		return 0
	}

	return num / (sqrt(denA) * sqrt(denB))
}

// Predict returns scores for candidate items for a user.
func (u *UserBasedCF) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	u.acquirePredictLock()
	defer u.releasePredictLock()

	if !u.trained {
		return nil, nil
	}

	neighbors, ok := u.userSimilarity[userID]
	if !ok || len(neighbors) == 0 {
		return nil, nil
	}

	scores := make(map[int]float64, len(candidates))

	for _, itemID := range candidates {
		var num, den float64

		for _, n := range neighbors {
			if rating, ok := u.userVectors[n.ID][itemID]; ok {
				num += n.Similarity * rating
				den += abs(n.Similarity)
			}
		}

		if den > 0 {
			scores[itemID] = num / den
		}
	}

	return normalizeScores(scores), nil
}

// PredictSimilar returns items similar to the given item.
func (u *UserBasedCF) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	u.acquirePredictLock()
	defer u.releasePredictLock()

	if !u.trained {
		return nil, nil
	}

	// Get users who liked this item
	users := u.itemUsers[itemID]
	if len(users) == 0 {
		return nil, nil
	}

	// Score candidates based on how many similar users liked them
	scores := make(map[int]float64, len(candidates))

	for _, candidateID := range candidates {
		if candidateID == itemID {
			continue
		}

		candidateUsers := u.itemUsers[candidateID]
		if len(candidateUsers) == 0 {
			continue
		}

		// Jaccard similarity between user sets
		userSetA := make(map[int]struct{}, len(users))
		for _, uid := range users {
			userSetA[uid] = struct{}{}
		}

		intersection := 0
		for _, uid := range candidateUsers {
			if _, ok := userSetA[uid]; ok {
				intersection++
			}
		}

		union := len(users) + len(candidateUsers) - intersection
		if union > 0 {
			scores[candidateID] = float64(intersection) / float64(union)
		}
	}

	return normalizeScores(scores), nil
}

// ========== Item-Based Collaborative Filtering ==========

// ItemBasedCF implements item-based collaborative filtering.
// It recommends items similar to what the user has liked before.
//
// For a target user u and candidate item i:
// score(u, i) = sum_{j in N(i)} sim(i, j) * r(u, j) / sum_{j in N(i)} |sim(i, j)|
//
// where N(i) is the set of k most similar items to i that user u has rated.
type ItemBasedCF struct {
	BaseAlgorithm
	config KNNConfig

	// itemVectors stores item interaction vectors (userID -> confidence)
	itemVectors map[int]map[int]float64

	// itemSimilarity stores precomputed item-item similarities
	itemSimilarity map[int][]neighbor

	// userItems stores which items each user interacted with
	userItems map[int][]int
}

// NewItemBasedCF creates a new item-based CF algorithm.
func NewItemBasedCF(cfg KNNConfig) *ItemBasedCF {
	if cfg.K <= 0 {
		cfg.K = 50
	}
	if cfg.MinSimilarity <= 0 {
		cfg.MinSimilarity = 0.1
	}
	if cfg.SimilarityMetric == "" {
		cfg.SimilarityMetric = "cosine"
	}
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 4
	}

	return &ItemBasedCF{
		BaseAlgorithm:  NewBaseAlgorithm("itemcf"),
		config:         cfg,
		itemVectors:    make(map[int]map[int]float64),
		itemSimilarity: make(map[int][]neighbor),
		userItems:      make(map[int][]int),
	}
}

// Train fits the item-based CF model.
//
//nolint:gocritic // rangeValCopy: Interaction passed by value in range, acceptable for clarity
func (i *ItemBasedCF) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	i.acquireTrainLock()
	defer i.releaseTrainLock()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Build item vectors (transposed view)
	i.itemVectors = make(map[int]map[int]float64)
	i.userItems = make(map[int][]int)

	for _, inter := range interactions {
		if i.itemVectors[inter.ItemID] == nil {
			i.itemVectors[inter.ItemID] = make(map[int]float64)
		}
		if c := i.itemVectors[inter.ItemID][inter.UserID]; inter.Confidence > c {
			i.itemVectors[inter.ItemID][inter.UserID] = inter.Confidence
		}
	}

	// Build user-item index
	for itemID, userMap := range i.itemVectors {
		for userID := range userMap {
			i.userItems[userID] = append(i.userItems[userID], itemID)
		}
	}

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	// Precompute item similarities
	i.itemSimilarity = make(map[int][]neighbor)
	itemIDs := make([]int, 0, len(i.itemVectors))
	for iid := range i.itemVectors {
		itemIDs = append(itemIDs, iid)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	chunkSize := (len(itemIDs) + i.config.NumWorkers - 1) / i.config.NumWorkers

	for w := 0; w < i.config.NumWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(itemIDs) {
			end = len(itemIDs)
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(itemSlice []int) {
			defer wg.Done()

			for _, iid := range itemSlice {
				if ContextCancelled(ctx) {
					return
				}

				neighbors := i.computeItemNeighbors(iid, itemIDs)

				mu.Lock()
				i.itemSimilarity[iid] = neighbors
				mu.Unlock()
			}
		}(itemIDs[start:end])
	}

	wg.Wait()

	if ContextCancelled(ctx) {
		return ctx.Err()
	}

	i.markTrained()
	return nil
}

// computeItemNeighbors computes the k most similar items for a given item.
func (i *ItemBasedCF) computeItemNeighbors(itemID int, allItems []int) []neighbor {
	itemVec := i.itemVectors[itemID]
	if len(itemVec) == 0 {
		return nil
	}

	neighbors := make([]neighbor, 0, len(allItems))

	for _, otherID := range allItems {
		if otherID == itemID {
			continue
		}

		otherVec := i.itemVectors[otherID]
		sim := i.computeSimilarity(itemVec, otherVec)

		if sim >= i.config.MinSimilarity {
			neighbors = append(neighbors, neighbor{ID: otherID, Similarity: sim})
		}
	}

	// Sort by similarity (descending) and take top K
	sort.Slice(neighbors, func(a, b int) bool {
		return neighbors[a].Similarity > neighbors[b].Similarity
	})

	if len(neighbors) > i.config.K {
		neighbors = neighbors[:i.config.K]
	}

	return neighbors
}

// computeSimilarity computes similarity between two item vectors.
func (i *ItemBasedCF) computeSimilarity(a, b map[int]float64) float64 {
	// Find common users
	var commonUsers []int
	for user := range a {
		if _, ok := b[user]; ok {
			commonUsers = append(commonUsers, user)
		}
	}

	if len(commonUsers) < i.config.MinCommonItems {
		return 0
	}

	// Compute similarity based on metric
	var sim float64
	switch i.config.SimilarityMetric {
	case "cosine":
		sim = i.cosineSim(a, b, commonUsers)
	case "pearson":
		sim = i.pearsonSim(a, b, commonUsers)
	case "jaccard":
		sim = float64(len(commonUsers)) / float64(len(a)+len(b)-len(commonUsers))
	default:
		sim = i.cosineSim(a, b, commonUsers)
	}

	// Apply shrinkage
	if i.config.Shrinkage > 0 {
		sim = sim * float64(len(commonUsers)) / (float64(len(commonUsers)) + i.config.Shrinkage)
	}

	return sim
}

func (i *ItemBasedCF) cosineSim(a, b map[int]float64, common []int) float64 {
	var dot, normA, normB float64
	for _, user := range common {
		dot += a[user] * b[user]
	}
	for _, v := range a {
		normA += v * v
	}
	for _, v := range b {
		normB += v * v
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (sqrt(normA) * sqrt(normB))
}

func (i *ItemBasedCF) pearsonSim(a, b map[int]float64, common []int) float64 {
	if len(common) == 0 {
		return 0
	}

	var sumA, sumB float64
	for _, user := range common {
		sumA += a[user]
		sumB += b[user]
	}
	meanA := sumA / float64(len(common))
	meanB := sumB / float64(len(common))

	var num, denA, denB float64
	for _, user := range common {
		diffA := a[user] - meanA
		diffB := b[user] - meanB
		num += diffA * diffB
		denA += diffA * diffA
		denB += diffB * diffB
	}

	if denA == 0 || denB == 0 {
		return 0
	}

	return num / (sqrt(denA) * sqrt(denB))
}

// Predict returns scores for candidate items for a user.
func (i *ItemBasedCF) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	i.acquirePredictLock()
	defer i.releasePredictLock()

	if !i.trained {
		return nil, nil
	}

	userItemsList := i.userItems[userID]
	if len(userItemsList) == 0 {
		return nil, nil
	}

	// Build user ratings map for quick lookup
	userRatings := make(map[int]float64)
	for _, itemID := range userItemsList {
		if vec, ok := i.itemVectors[itemID]; ok {
			if rating, ok := vec[userID]; ok {
				userRatings[itemID] = rating
			}
		}
	}

	scores := make(map[int]float64, len(candidates))

	for _, candidateID := range candidates {
		neighbors := i.itemSimilarity[candidateID]
		if len(neighbors) == 0 {
			continue
		}

		var num, den float64
		for _, n := range neighbors {
			if rating, ok := userRatings[n.ID]; ok {
				num += n.Similarity * rating
				den += abs(n.Similarity)
			}
		}

		if den > 0 {
			scores[candidateID] = num / den
		}
	}

	return normalizeScores(scores), nil
}

// PredictSimilar returns items similar to the given item.
func (i *ItemBasedCF) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	i.acquirePredictLock()
	defer i.releasePredictLock()

	if !i.trained {
		return nil, nil
	}

	neighbors := i.itemSimilarity[itemID]
	if len(neighbors) == 0 {
		return nil, nil
	}

	// Build neighbor set for quick lookup
	neighborSim := make(map[int]float64)
	for _, n := range neighbors {
		neighborSim[n.ID] = n.Similarity
	}

	scores := make(map[int]float64, len(candidates))
	for _, candidateID := range candidates {
		if candidateID == itemID {
			continue
		}
		if sim, ok := neighborSim[candidateID]; ok {
			scores[candidateID] = sim
		}
	}

	return normalizeScores(scores), nil
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Ensure interface compliance.
var (
	_ recommend.Algorithm = (*UserBasedCF)(nil)
	_ recommend.Algorithm = (*ItemBasedCF)(nil)
)
