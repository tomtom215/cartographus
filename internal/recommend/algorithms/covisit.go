// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"sort"
	"time"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// CoVisitation implements the co-visitation recommendation algorithm.
// It recommends items that are frequently watched together within a session.
//
// This is one of the simplest and most effective recommendation algorithms,
// particularly for session-based recommendations (e.g., "other users also watched").
//
// The algorithm builds a sparse matrix of co-occurrence counts:
//
//	covisit[item_a][item_b] = count of sessions where both items appeared
//
// For prediction, items are scored by their total co-occurrence with items
// the user has already watched.
type CoVisitation struct {
	BaseAlgorithm

	// Configuration
	minCoOccurrence    int
	sessionWindowHours int
	maxPairs           int

	// Trained model: item_a -> item_b -> count
	cooccurrence map[int]map[int]float64

	// Inverted index: item -> users who watched it
	itemUsers map[int][]int

	// Total occurrence counts for normalization
	itemCounts map[int]int
}

// CoVisitConfig contains configuration for the co-visitation algorithm.
type CoVisitConfig struct {
	// MinCoOccurrence is the minimum number of co-occurrences to store.
	MinCoOccurrence int

	// SessionWindowHours defines the session grouping window.
	SessionWindowHours int

	// MaxPairs is the maximum number of co-visitation pairs to store.
	MaxPairs int
}

// NewCoVisitation creates a new co-visitation algorithm.
func NewCoVisitation(cfg CoVisitConfig) *CoVisitation {
	if cfg.MinCoOccurrence < 1 {
		cfg.MinCoOccurrence = 2
	}
	if cfg.SessionWindowHours < 1 {
		cfg.SessionWindowHours = 24
	}
	if cfg.MaxPairs < 1 {
		cfg.MaxPairs = 100000
	}

	return &CoVisitation{
		BaseAlgorithm:      NewBaseAlgorithm("covisit"),
		minCoOccurrence:    cfg.MinCoOccurrence,
		sessionWindowHours: cfg.SessionWindowHours,
		maxPairs:           cfg.MaxPairs,
		cooccurrence:       make(map[int]map[int]float64),
		itemUsers:          make(map[int][]int),
		itemCounts:         make(map[int]int),
	}
}

// Train builds the co-visitation matrix from interactions.
//
//nolint:gocritic // rangeValCopy: Interaction passed by value in range, acceptable for clarity
func (c *CoVisitation) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	c.acquireTrainLock()
	defer c.releaseTrainLock()

	// Clear previous model
	c.cooccurrence = make(map[int]map[int]float64)
	c.itemUsers = make(map[int][]int)
	c.itemCounts = make(map[int]int)

	if len(interactions) == 0 {
		c.markTrained()
		return nil
	}

	// Group interactions by user
	userItems := make(map[int][]timedItem)
	for _, inter := range interactions {
		userItems[inter.UserID] = append(userItems[inter.UserID], timedItem{
			itemID:    inter.ItemID,
			timestamp: inter.Timestamp,
		})
	}

	// Build co-occurrence matrix
	sessionWindow := time.Duration(c.sessionWindowHours) * time.Hour
	cooccurrenceCounts := make(map[int]map[int]int)

	for userID, items := range userItems {
		if ContextCancelled(ctx) {
			return ctx.Err()
		}

		// Sort by timestamp
		sort.Slice(items, func(i, j int) bool {
			return items[i].timestamp.Before(items[j].timestamp)
		})

		// Group into sessions
		sessions := groupIntoSessions(items, sessionWindow)

		for _, session := range sessions {
			// Count item occurrences
			for _, ti := range session {
				c.itemCounts[ti.itemID]++
				c.itemUsers[ti.itemID] = append(c.itemUsers[ti.itemID], userID)
			}

			// Build co-occurrence pairs within session
			for i := 0; i < len(session); i++ {
				for j := i + 1; j < len(session); j++ {
					itemA, itemB := session[i].itemID, session[j].itemID

					// Ensure consistent ordering for symmetric pairs
					if itemA > itemB {
						itemA, itemB = itemB, itemA
					}

					if cooccurrenceCounts[itemA] == nil {
						cooccurrenceCounts[itemA] = make(map[int]int)
					}
					cooccurrenceCounts[itemA][itemB]++
				}
			}
		}
	}

	// Convert to normalized similarity scores
	c.cooccurrence = c.buildSimilarityMatrix(cooccurrenceCounts)

	c.markTrained()
	return nil
}

// timedItem associates an item with a timestamp.
type timedItem struct {
	itemID    int
	timestamp time.Time
}

// groupIntoSessions groups items into sessions based on time window.
func groupIntoSessions(items []timedItem, window time.Duration) [][]timedItem {
	if len(items) == 0 {
		return nil
	}

	var sessions [][]timedItem
	currentSession := []timedItem{items[0]}

	for i := 1; i < len(items); i++ {
		if items[i].timestamp.Sub(currentSession[len(currentSession)-1].timestamp) > window {
			// New session
			sessions = append(sessions, currentSession)
			currentSession = []timedItem{items[i]}
		} else {
			currentSession = append(currentSession, items[i])
		}
	}

	// Add last session
	if len(currentSession) > 0 {
		sessions = append(sessions, currentSession)
	}

	return sessions
}

// buildSimilarityMatrix converts co-occurrence counts to similarity scores.
func (c *CoVisitation) buildSimilarityMatrix(counts map[int]map[int]int) map[int]map[int]float64 {
	similarity := make(map[int]map[int]float64)

	// Collect all pairs that meet minimum threshold
	type pair struct {
		a, b  int
		count int
	}
	var pairs []pair

	for itemA, bCounts := range counts {
		for itemB, count := range bCounts {
			if count >= c.minCoOccurrence {
				pairs = append(pairs, pair{itemA, itemB, count})
			}
		}
	}

	// Sort by count descending and take top maxPairs
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	if len(pairs) > c.maxPairs {
		pairs = pairs[:c.maxPairs]
	}

	// Build similarity matrix using Jaccard-like coefficient
	for _, p := range pairs {
		// Similarity = co-occurrence / (count_a + count_b - co-occurrence)
		countA := c.itemCounts[p.a]
		countB := c.itemCounts[p.b]
		union := countA + countB - p.count

		var sim float64
		if union > 0 {
			sim = float64(p.count) / float64(union)
		}

		// Store symmetric
		if similarity[p.a] == nil {
			similarity[p.a] = make(map[int]float64)
		}
		if similarity[p.b] == nil {
			similarity[p.b] = make(map[int]float64)
		}

		similarity[p.a][p.b] = sim
		similarity[p.b][p.a] = sim
	}

	return similarity
}

// Predict returns scores for candidate items based on user history.
func (c *CoVisitation) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	c.acquirePredictLock()
	defer c.releasePredictLock()

	if !c.trained || len(c.cooccurrence) == 0 {
		return nil, nil
	}

	// Get items the user has interacted with
	userHistory := c.getUserHistory(userID)
	if len(userHistory) == 0 {
		return nil, nil
	}

	// Score candidates by co-occurrence with user history
	scores := make(map[int]float64)
	candidateSet := make(map[int]struct{}, len(candidates))
	for _, id := range candidates {
		candidateSet[id] = struct{}{}
	}

	for candidateID := range candidateSet {
		if ContextCancelled(ctx) {
			return nil, ctx.Err()
		}

		var totalScore float64
		for _, historyItem := range userHistory {
			if sim, ok := c.cooccurrence[historyItem][candidateID]; ok {
				totalScore += sim
			}
		}

		if totalScore > 0 {
			scores[candidateID] = totalScore
		}
	}

	return normalizeScores(scores), nil
}

// PredictSimilar returns items similar to the given item.
func (c *CoVisitation) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	c.acquirePredictLock()
	defer c.releasePredictLock()

	if !c.trained || len(c.cooccurrence) == 0 {
		return nil, nil
	}

	itemSimilarities, ok := c.cooccurrence[itemID]
	if !ok {
		return nil, nil
	}

	// Filter to candidates
	scores := make(map[int]float64)
	for _, candidateID := range candidates {
		if sim, ok := itemSimilarities[candidateID]; ok {
			scores[candidateID] = sim
		}
	}

	return normalizeScores(scores), nil
}

// getUserHistory returns items the user has interacted with.
func (c *CoVisitation) getUserHistory(userID int) []int {
	history := make(map[int]struct{})
	for itemID, users := range c.itemUsers {
		for _, uid := range users {
			if uid == userID {
				history[itemID] = struct{}{}
				break
			}
		}
	}

	result := make([]int, 0, len(history))
	for itemID := range history {
		result = append(result, itemID)
	}
	return result
}
