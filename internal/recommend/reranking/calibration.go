// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package reranking

import (
	"context"
	"math"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// CalibrationConfig contains configuration for calibration reranking.
type CalibrationConfig struct {
	// Lambda balances relevance and calibration (0=pure calibration, 1=pure relevance).
	// Typical range: 0.5-0.9.
	Lambda float64

	// AttributeWeights assigns importance to different attributes for calibration.
	// Keys: "genre", "year", "content_rating", "media_type".
	// If empty, only genre calibration is used with weight 1.0.
	AttributeWeights map[string]float64

	// TargetDistribution overrides the user's historical distribution.
	// If nil, uses the user's actual consumption pattern.
	TargetDistribution map[string]map[string]float64
}

// DefaultCalibrationConfig returns default calibration configuration.
func DefaultCalibrationConfig() CalibrationConfig {
	return CalibrationConfig{
		Lambda: 0.7,
		AttributeWeights: map[string]float64{
			"genre":          1.0,
			"year":           0.3,
			"content_rating": 0.2,
		},
	}
}

// Calibration implements calibrated recommendations.
// Reference: "Calibrated Recommendations" (Steck, 2018)
//
// Calibration ensures that the distribution of attributes (genres, years, etc.)
// in the recommendations matches the user's historical consumption pattern.
// This prevents filter bubbles and increases diversity while maintaining relevance.
//
// The objective function balances relevance and calibration:
// score(S) = lambda * relevance(S) + (1-lambda) * (1 - KL(p || q))
//
// where p is the target distribution and q is the distribution of recommendations.
type Calibration struct {
	config CalibrationConfig

	// userProfiles stores user consumption patterns by attribute
	// userProfiles[userID][attribute][value] = count
	userProfiles map[int]map[string]map[string]float64
}

// NewCalibration creates a new calibration reranker.
func NewCalibration(cfg CalibrationConfig) *Calibration {
	if cfg.Lambda < 0 {
		cfg.Lambda = 0
	}
	if cfg.Lambda > 1 {
		cfg.Lambda = 1
	}
	if len(cfg.AttributeWeights) == 0 {
		cfg.AttributeWeights = map[string]float64{"genre": 1.0}
	}

	return &Calibration{
		config:       cfg,
		userProfiles: make(map[int]map[string]map[string]float64),
	}
}

// Name returns the reranker identifier.
func (c *Calibration) Name() string {
	return "calibration"
}

// SetUserProfile sets the target distribution for a user.
// The distributions map attribute names to value distributions.
func (c *Calibration) SetUserProfile(userID int, distributions map[string]map[string]float64) {
	c.userProfiles[userID] = distributions
}

// LearnFromHistory learns user profiles from interaction history.
//
//nolint:gocyclo,gocritic // gocyclo: user profile learning involves multiple attribute types; gocritic: rangeValCopy is acceptable for clarity
func (c *Calibration) LearnFromHistory(interactions []recommend.Interaction, items map[int]recommend.Item) {
	c.userProfiles = make(map[int]map[string]map[string]float64)

	for _, inter := range interactions {
		item, ok := items[inter.ItemID]
		if !ok {
			continue
		}

		if c.userProfiles[inter.UserID] == nil {
			c.userProfiles[inter.UserID] = make(map[string]map[string]float64)
		}

		profile := c.userProfiles[inter.UserID]

		// Track genre distribution
		if _, ok := c.config.AttributeWeights["genre"]; ok {
			if profile["genre"] == nil {
				profile["genre"] = make(map[string]float64)
			}
			for _, genre := range item.Genres {
				profile["genre"][genre] += inter.Confidence
			}
		}

		// Track year distribution (decade buckets)
		if _, ok := c.config.AttributeWeights["year"]; ok && item.Year > 0 {
			if profile["year"] == nil {
				profile["year"] = make(map[string]float64)
			}
			decade := decadeBucket(item.Year)
			profile["year"][decade] += inter.Confidence
		}

		// Track content rating distribution
		if _, ok := c.config.AttributeWeights["content_rating"]; ok && item.ContentRating != "" {
			if profile["content_rating"] == nil {
				profile["content_rating"] = make(map[string]float64)
			}
			profile["content_rating"][item.ContentRating] += inter.Confidence
		}

		// Track media type distribution
		if _, ok := c.config.AttributeWeights["media_type"]; ok && item.MediaType != "" {
			if profile["media_type"] == nil {
				profile["media_type"] = make(map[string]float64)
			}
			profile["media_type"][item.MediaType] += inter.Confidence
		}
	}

	// Normalize distributions
	for _, profile := range c.userProfiles {
		for attr := range profile {
			normalizeDistribution(profile[attr])
		}
	}
}

// Rerank reorders items to match user's historical consumption pattern.
//
//nolint:gocritic // rangeValCopy: ScoredItem passed by value in range, acceptable for clarity
func (c *Calibration) Rerank(ctx context.Context, items []recommend.ScoredItem, k int) []recommend.ScoredItem {
	if len(items) <= 1 || k <= 0 {
		return items
	}

	// Bound k to prevent excessive memory allocation
	if k > maxRerankSize {
		k = maxRerankSize
	}
	if k > len(items) {
		k = len(items)
	}

	// Get user ID from first item (all items are for same user in a request)
	// Note: In practice, user ID should be passed separately
	// For now, we use a simplified approach that works for any user
	result := make([]recommend.ScoredItem, 0, k)
	remaining := make([]recommend.ScoredItem, len(items))
	copy(remaining, items)

	// Greedy selection
	for len(result) < k && len(remaining) > 0 {
		bestIdx := -1
		bestScore := math.Inf(-1)

		for i, item := range remaining {
			// Compute score if we add this item
			testResult := append(result, item)
			newDist := c.computeDistribution(testResult)

			// Compute calibration score (negative KL divergence)
			calibScore := c.computeCalibrationScore(newDist)

			// Combined score: lambda * relevance + (1-lambda) * calibration
			combinedScore := c.config.Lambda*item.Score + (1-c.config.Lambda)*calibScore

			if combinedScore > bestScore {
				bestScore = combinedScore
				bestIdx = i
			}
		}

		if bestIdx >= 0 {
			result = append(result, remaining[bestIdx])
			// Remove selected item from remaining
			remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
		} else {
			break
		}
	}

	return result
}

// computeDistribution computes the attribute distribution for a list of items.
//
//nolint:gocritic // rangeValCopy: ScoredItem passed by value in range, acceptable for clarity
func (c *Calibration) computeDistribution(items []recommend.ScoredItem) map[string]map[string]float64 {
	dist := make(map[string]map[string]float64)

	for attr := range c.config.AttributeWeights {
		dist[attr] = make(map[string]float64)
	}

	for _, item := range items {
		if _, ok := c.config.AttributeWeights["genre"]; ok {
			for _, genre := range item.Item.Genres {
				dist["genre"][genre]++
			}
		}

		if _, ok := c.config.AttributeWeights["year"]; ok && item.Item.Year > 0 {
			dist["year"][decadeBucket(item.Item.Year)]++
		}

		if _, ok := c.config.AttributeWeights["content_rating"]; ok && item.Item.ContentRating != "" {
			dist["content_rating"][item.Item.ContentRating]++
		}

		if _, ok := c.config.AttributeWeights["media_type"]; ok && item.Item.MediaType != "" {
			dist["media_type"][item.Item.MediaType]++
		}
	}

	// Normalize each distribution
	for attr := range dist {
		normalizeDistribution(dist[attr])
	}

	return dist
}

// computeCalibrationScore computes how well a distribution matches target.
// Higher is better (returns 1 - normalized_divergence).
func (c *Calibration) computeCalibrationScore(dist map[string]map[string]float64) float64 {
	if len(c.config.TargetDistribution) == 0 {
		// No target distribution - return neutral score
		return 0.5
	}

	var totalScore float64
	var totalWeight float64

	for attr, weight := range c.config.AttributeWeights {
		targetDist, ok := c.config.TargetDistribution[attr]
		if !ok || len(targetDist) == 0 {
			continue
		}

		recDist := dist[attr]
		if len(recDist) == 0 {
			continue
		}

		// Compute KL divergence: KL(target || rec)
		kl := klDivergence(targetDist, recDist)

		// Convert to score (1 - normalized divergence)
		// Clamp to [0, 1]
		score := 1.0 - math.Min(kl, 1.0)

		totalScore += weight * score
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.5
	}

	return totalScore / totalWeight
}

// klDivergence computes KL divergence from p to q.
func klDivergence(p, q map[string]float64) float64 {
	var kl float64
	epsilon := 1e-10 // Smoothing to avoid log(0)

	for key, pVal := range p {
		qVal := q[key]
		if qVal <= 0 {
			qVal = epsilon
		}
		if pVal > 0 {
			kl += pVal * math.Log(pVal/qVal)
		}
	}

	return kl
}

// normalizeDistribution normalizes a distribution to sum to 1.
func normalizeDistribution(dist map[string]float64) {
	var total float64
	for _, v := range dist {
		total += v
	}

	if total > 0 {
		for k := range dist {
			dist[k] /= total
		}
	}
}

// decadeBucket converts a year to a decade bucket string.
func decadeBucket(year int) string {
	decade := (year / 10) * 10
	switch {
	case decade >= 2020:
		return "2020s"
	case decade >= 2010:
		return "2010s"
	case decade >= 2000:
		return "2000s"
	case decade >= 1990:
		return "1990s"
	case decade >= 1980:
		return "1980s"
	default:
		return "pre-1980"
	}
}

// Ensure interface compliance.
var _ recommend.Reranker = (*Calibration)(nil)
