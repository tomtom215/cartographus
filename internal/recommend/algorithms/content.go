// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package algorithms

import (
	"context"
	"math"
	"strings"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// ContentBased implements content-based filtering using item metadata.
// It recommends items similar to those a user has previously enjoyed,
// based on attributes like genres, actors, directors, and year.
//
// This algorithm is particularly valuable for:
//   - Cold start: New users with little history
//   - New items: Items with no interaction history
//   - Interpretable recommendations: "Because you liked Action movies"
//
// The similarity between items is computed as a weighted combination:
//
//	sim(a, b) = w_genre * jaccard(genres_a, genres_b) +
//	            w_actor * jaccard(actors_a, actors_b) +
//	            w_director * jaccard(directors_a, directors_b) +
//	            w_year * year_similarity(year_a, year_b)
type ContentBased struct {
	BaseAlgorithm

	// Configuration
	genreWeight       float64
	actorWeight       float64
	directorWeight    float64
	yearWeight        float64
	maxYearDifference int

	// Trained model
	items        []recommend.Item
	itemIndex    map[int]int       // item_id -> index in items slice
	userProfiles map[int]*profile  // user_id -> preference profile
	itemFeatures map[int]*features // item_id -> feature vectors
}

// profile represents a user's content preferences.
type profile struct {
	genres    map[string]float64
	actors    map[string]float64
	directors map[string]float64
	avgYear   float64
	yearCount int
}

// features represents an item's feature vectors.
type features struct {
	genres    []string
	actors    []string
	directors []string
	year      int
}

// ContentBasedConfig contains configuration for content-based filtering.
type ContentBasedConfig struct {
	GenreWeight       float64
	ActorWeight       float64
	DirectorWeight    float64
	YearWeight        float64
	MaxYearDifference int
}

// NewContentBased creates a new content-based algorithm.
func NewContentBased(cfg ContentBasedConfig) *ContentBased {
	// Apply defaults
	if cfg.GenreWeight == 0 {
		cfg.GenreWeight = 0.4
	}
	if cfg.ActorWeight == 0 {
		cfg.ActorWeight = 0.3
	}
	if cfg.DirectorWeight == 0 {
		cfg.DirectorWeight = 0.2
	}
	if cfg.YearWeight == 0 {
		cfg.YearWeight = 0.1
	}
	if cfg.MaxYearDifference == 0 {
		cfg.MaxYearDifference = 20
	}

	// Normalize weights
	total := cfg.GenreWeight + cfg.ActorWeight + cfg.DirectorWeight + cfg.YearWeight
	if total > 0 {
		cfg.GenreWeight /= total
		cfg.ActorWeight /= total
		cfg.DirectorWeight /= total
		cfg.YearWeight /= total
	}

	return &ContentBased{
		BaseAlgorithm:     NewBaseAlgorithm("content"),
		genreWeight:       cfg.GenreWeight,
		actorWeight:       cfg.ActorWeight,
		directorWeight:    cfg.DirectorWeight,
		yearWeight:        cfg.YearWeight,
		maxYearDifference: cfg.MaxYearDifference,
		itemIndex:         make(map[int]int),
		userProfiles:      make(map[int]*profile),
		itemFeatures:      make(map[int]*features),
	}
}

// Train builds user preference profiles from interactions.
//
//nolint:gocritic // rangeValCopy: Item/Interaction passed by value in range, acceptable for clarity
func (c *ContentBased) Train(ctx context.Context, interactions []recommend.Interaction, items []recommend.Item) error {
	c.acquireTrainLock()
	defer c.releaseTrainLock()

	// Clear previous model
	c.items = items
	c.itemIndex = make(map[int]int, len(items))
	c.userProfiles = make(map[int]*profile)
	c.itemFeatures = make(map[int]*features, len(items))

	// Build item index and features
	for i, item := range items {
		c.itemIndex[item.ID] = i
		c.itemFeatures[item.ID] = &features{
			genres:    item.Genres,
			actors:    item.Actors,
			directors: item.Directors,
			year:      item.Year,
		}
	}

	if len(interactions) == 0 {
		c.markTrained()
		return nil
	}

	// Build user profiles from interactions
	for _, inter := range interactions {
		if ContextCancelled(ctx) {
			return ctx.Err()
		}

		// Only consider positive interactions
		if inter.Type == recommend.InteractionAbandoned {
			continue
		}

		feat, ok := c.itemFeatures[inter.ItemID]
		if !ok {
			continue
		}

		// Get or create user profile
		prof, ok := c.userProfiles[inter.UserID]
		if !ok {
			prof = &profile{
				genres:    make(map[string]float64),
				actors:    make(map[string]float64),
				directors: make(map[string]float64),
			}
			c.userProfiles[inter.UserID] = prof
		}

		// Weight by interaction confidence
		weight := inter.Confidence
		if weight <= 0 {
			weight = inter.Type.Confidence()
		}

		// Update genre preferences
		for _, genre := range feat.genres {
			prof.genres[strings.ToLower(genre)] += weight
		}

		// Update actor preferences
		for _, actor := range feat.actors {
			prof.actors[strings.ToLower(actor)] += weight
		}

		// Update director preferences
		for _, director := range feat.directors {
			prof.directors[strings.ToLower(director)] += weight
		}

		// Update year preference
		if feat.year > 0 {
			prof.avgYear = (prof.avgYear*float64(prof.yearCount) + float64(feat.year)) / float64(prof.yearCount+1)
			prof.yearCount++
		}
	}

	// Normalize user profiles
	for _, prof := range c.userProfiles {
		normalizeProfile(prof)
	}

	c.markTrained()
	return nil
}

// normalizeProfile normalizes a user profile to unit vectors.
func normalizeProfile(prof *profile) {
	normalizeMap(prof.genres)
	normalizeMap(prof.actors)
	normalizeMap(prof.directors)
}

// normalizeMap normalizes map values to sum to 1.
func normalizeMap(m map[string]float64) {
	var sum float64
	for _, v := range m {
		sum += v
	}
	if sum > 0 {
		for k := range m {
			m[k] /= sum
		}
	}
}

// Predict returns scores for candidate items based on user profile.
func (c *ContentBased) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	c.acquirePredictLock()
	defer c.releasePredictLock()

	if !c.trained {
		return nil, nil
	}

	prof, ok := c.userProfiles[userID]
	if !ok {
		// Cold start user - return nil for fallback to other algorithms
		return nil, nil
	}

	scores := make(map[int]float64, len(candidates))
	for _, candidateID := range candidates {
		if ContextCancelled(ctx) {
			return nil, ctx.Err()
		}

		feat, ok := c.itemFeatures[candidateID]
		if !ok {
			continue
		}

		score := c.computeUserItemScore(prof, feat)
		if score > 0 {
			scores[candidateID] = score
		}
	}

	return normalizeScores(scores), nil
}

// PredictSimilar returns items similar to the given item.
func (c *ContentBased) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	c.acquirePredictLock()
	defer c.releasePredictLock()

	if !c.trained {
		return nil, nil
	}

	sourceFeat, ok := c.itemFeatures[itemID]
	if !ok {
		return nil, nil
	}

	scores := make(map[int]float64, len(candidates))
	for _, candidateID := range candidates {
		if ContextCancelled(ctx) {
			return nil, ctx.Err()
		}

		if candidateID == itemID {
			continue
		}

		candidateFeat, ok := c.itemFeatures[candidateID]
		if !ok {
			continue
		}

		score := c.computeItemSimilarity(sourceFeat, candidateFeat)
		if score > 0 {
			scores[candidateID] = score
		}
	}

	return normalizeScores(scores), nil
}

// computeUserItemScore computes the score between a user profile and item.
func (c *ContentBased) computeUserItemScore(prof *profile, feat *features) float64 {
	var score float64

	// Genre match
	score += c.genreWeight * computeProfileMatch(prof.genres, feat.genres)

	// Actor match
	score += c.actorWeight * computeProfileMatch(prof.actors, feat.actors)

	// Director match
	score += c.directorWeight * computeProfileMatch(prof.directors, feat.directors)

	// Year preference
	if feat.year > 0 && prof.yearCount > 0 {
		yearDiff := math.Abs(float64(feat.year) - prof.avgYear)
		yearSim := 1.0 - yearDiff/float64(c.maxYearDifference)
		if yearSim < 0 {
			yearSim = 0
		}
		score += c.yearWeight * yearSim
	}

	return score
}

// computeProfileMatch computes how well item features match user preferences.
func computeProfileMatch(preferences map[string]float64, itemFeatures []string) float64 {
	if len(preferences) == 0 || len(itemFeatures) == 0 {
		return 0
	}

	var totalWeight float64
	for _, feat := range itemFeatures {
		if weight, ok := preferences[strings.ToLower(feat)]; ok {
			totalWeight += weight
		}
	}

	return totalWeight
}

// computeItemSimilarity computes similarity between two items.
func (c *ContentBased) computeItemSimilarity(a, b *features) float64 {
	var score float64

	// Genre similarity
	score += c.genreWeight * jaccardSimilarity(a.genres, b.genres)

	// Actor similarity
	score += c.actorWeight * jaccardSimilarity(a.actors, b.actors)

	// Director similarity
	score += c.directorWeight * jaccardSimilarity(a.directors, b.directors)

	// Year similarity
	if a.year > 0 && b.year > 0 {
		yearDiff := math.Abs(float64(a.year - b.year))
		yearSim := 1.0 - yearDiff/float64(c.maxYearDifference)
		if yearSim < 0 {
			yearSim = 0
		}
		score += c.yearWeight * yearSim
	}

	return score
}
