// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package recommend

import (
	"encoding/json"
	"fmt"
	"time"
)

// Config contains all configuration for the recommendation engine.
type Config struct {
	// Weights defines the relative contribution of each algorithm.
	// Weights are normalized at runtime, so they don't need to sum to 1.0.
	Weights AlgorithmWeights `json:"weights"`

	// EASE contains parameters for the EASE algorithm.
	EASE EASEConfig `json:"ease"`

	// ALS contains parameters for the ALS algorithm.
	ALS ALSConfig `json:"als"`

	// ContentBased contains parameters for content-based filtering.
	ContentBased ContentBasedConfig `json:"content_based"`

	// CoVisit contains parameters for co-visitation.
	CoVisit CoVisitConfig `json:"covisit"`

	// BPR contains parameters for Bayesian Personalized Ranking.
	BPR BPRConfigMain `json:"bpr"`

	// TimeAwareCF contains parameters for time-aware collaborative filtering.
	TimeAwareCF TimeAwareCFConfigMain `json:"time_aware_cf"`

	// MultiHopItemCF contains parameters for multi-hop item-based CF.
	MultiHopItemCF MultiHopItemCFConfigMain `json:"multihop_itemcf"`

	// MarkovChain contains parameters for simple Markov chain.
	MarkovChain MarkovChainConfigMain `json:"markov_chain"`

	// Diversity contains parameters for diversity reranking.
	Diversity DiversityConfig `json:"diversity"`

	// Training contains training schedule parameters.
	Training TrainingConfig `json:"training"`

	// Limits contains operational limits.
	Limits LimitsConfig `json:"limits"`

	// Cache contains caching parameters.
	Cache CacheConfig `json:"cache"`

	// Seed is the random seed for deterministic behavior.
	// If zero, a fixed default seed is used.
	Seed int64 `json:"seed"`
}

// AlgorithmWeights defines the relative contribution of each algorithm.
type AlgorithmWeights struct {
	// EASE is the weight for EASE algorithm.
	EASE float64 `json:"ease"`

	// ALS is the weight for ALS algorithm.
	ALS float64 `json:"als"`

	// UserCF is the weight for user-based collaborative filtering.
	UserCF float64 `json:"user_cf"`

	// ItemCF is the weight for item-based collaborative filtering.
	ItemCF float64 `json:"item_cf"`

	// Content is the weight for content-based filtering.
	Content float64 `json:"content"`

	// CoVisit is the weight for co-visitation.
	CoVisit float64 `json:"covisit"`

	// SASRec is the weight for sequential recommendation.
	SASRec float64 `json:"sasrec"`

	// FPMC is the weight for factorized personalized Markov chains.
	FPMC float64 `json:"fpmc"`

	// Popularity is the weight for popularity-based ranking.
	Popularity float64 `json:"popularity"`

	// Recency is the weight for recency-based ranking.
	Recency float64 `json:"recency"`

	// BPR is the weight for Bayesian Personalized Ranking.
	BPR float64 `json:"bpr"`

	// TimeAwareCF is the weight for time-aware collaborative filtering.
	TimeAwareCF float64 `json:"time_aware_cf"`

	// MultiHopItemCF is the weight for multi-hop item-based CF.
	MultiHopItemCF float64 `json:"multihop_itemcf"`

	// MarkovChain is the weight for simple Markov chain.
	MarkovChain float64 `json:"markov_chain"`
}

// Normalize returns a copy with weights normalized to sum to 1.0.
//
//nolint:gocritic // value receiver is intentional for immutable semantics
func (w AlgorithmWeights) Normalize() AlgorithmWeights {
	sum := w.EASE + w.ALS + w.UserCF + w.ItemCF + w.Content +
		w.CoVisit + w.SASRec + w.FPMC + w.Popularity + w.Recency +
		w.BPR + w.TimeAwareCF + w.MultiHopItemCF + w.MarkovChain

	if sum == 0 {
		// Return equal weights if all zero (14 algorithms, each gets 1/14 ≈ 0.0714)
		const equalWeight = 1.0 / 14.0
		return AlgorithmWeights{
			EASE: equalWeight, ALS: equalWeight, UserCF: equalWeight, ItemCF: equalWeight,
			Content: equalWeight, CoVisit: equalWeight, SASRec: equalWeight, FPMC: equalWeight,
			Popularity: equalWeight, Recency: equalWeight, BPR: equalWeight,
			TimeAwareCF: equalWeight, MultiHopItemCF: equalWeight, MarkovChain: equalWeight,
		}
	}

	return AlgorithmWeights{
		EASE:           w.EASE / sum,
		ALS:            w.ALS / sum,
		UserCF:         w.UserCF / sum,
		ItemCF:         w.ItemCF / sum,
		Content:        w.Content / sum,
		CoVisit:        w.CoVisit / sum,
		SASRec:         w.SASRec / sum,
		FPMC:           w.FPMC / sum,
		Popularity:     w.Popularity / sum,
		Recency:        w.Recency / sum,
		BPR:            w.BPR / sum,
		TimeAwareCF:    w.TimeAwareCF / sum,
		MultiHopItemCF: w.MultiHopItemCF / sum,
		MarkovChain:    w.MarkovChain / sum,
	}
}

// ToMap returns the weights as a string-keyed map.
//
//nolint:gocritic // value receiver is intentional for immutable semantics
func (w AlgorithmWeights) ToMap() map[string]float64 {
	return map[string]float64{
		"ease":            w.EASE,
		"als":             w.ALS,
		"user_cf":         w.UserCF,
		"item_cf":         w.ItemCF,
		"content":         w.Content,
		"covisit":         w.CoVisit,
		"sasrec":          w.SASRec,
		"fpmc":            w.FPMC,
		"popularity":      w.Popularity,
		"recency":         w.Recency,
		"bpr":             w.BPR,
		"time_aware_cf":   w.TimeAwareCF,
		"multihop_itemcf": w.MultiHopItemCF,
		"markov_chain":    w.MarkovChain,
	}
}

// EASEConfig contains parameters for the EASE algorithm.
type EASEConfig struct {
	// Lambda is the L2 regularization parameter.
	// Higher values produce sparser, more generalizable models.
	// Default: 500.0 (recommended for implicit feedback).
	Lambda float64 `json:"lambda"`

	// MaxItems is the maximum number of items to include in the model.
	// Memory usage scales quadratically with item count.
	// Default: 10000.
	MaxItems int `json:"max_items"`
}

// ALSConfig contains parameters for the ALS algorithm.
type ALSConfig struct {
	// Factors is the number of latent factors.
	// Higher values capture more nuance but increase memory.
	// Default: 64.
	Factors int `json:"factors"`

	// Lambda is the L2 regularization parameter.
	// Default: 0.1.
	Lambda float64 `json:"lambda"`

	// Alpha scales the confidence values.
	// Higher values make the model more sensitive to high-confidence interactions.
	// Default: 40.0.
	Alpha float64 `json:"alpha"`

	// Iterations is the number of training iterations.
	// Default: 15.
	Iterations int `json:"iterations"`
}

// ContentBasedConfig contains parameters for content-based filtering.
type ContentBasedConfig struct {
	// GenreWeight is the importance of genre similarity.
	// Default: 0.4.
	GenreWeight float64 `json:"genre_weight"`

	// ActorWeight is the importance of shared actors.
	// Default: 0.3.
	ActorWeight float64 `json:"actor_weight"`

	// DirectorWeight is the importance of shared directors.
	// Default: 0.2.
	DirectorWeight float64 `json:"director_weight"`

	// YearWeight is the importance of release year proximity.
	// Default: 0.1.
	YearWeight float64 `json:"year_weight"`

	// MaxYearDifference is the maximum year difference to consider.
	// Items beyond this are assigned zero year similarity.
	// Default: 20.
	MaxYearDifference int `json:"max_year_difference"`
}

// CoVisitConfig contains parameters for co-visitation.
type CoVisitConfig struct {
	// MinCoOccurrence is the minimum number of users who watched both items.
	// Default: 2.
	MinCoOccurrence int `json:"min_co_occurrence"`

	// SessionWindowHours defines the session grouping window.
	// Items watched within this window are considered co-visited.
	// Default: 24.
	SessionWindowHours int `json:"session_window_hours"`

	// MaxPairs is the maximum number of co-visitation pairs to store.
	// Default: 100000.
	MaxPairs int `json:"max_pairs"`
}

// BPRConfigMain contains parameters for Bayesian Personalized Ranking.
type BPRConfigMain struct {
	// NumFactors is the dimension of the latent factor vectors.
	// Default: 64.
	NumFactors int `json:"num_factors"`

	// LearningRate is the SGD step size.
	// Default: 0.01.
	LearningRate float64 `json:"learning_rate"`

	// Regularization is the L2 regularization parameter.
	// Default: 0.01.
	Regularization float64 `json:"regularization"`

	// NumIterations is the number of training epochs.
	// Default: 100.
	NumIterations int `json:"num_iterations"`

	// NumNegativeSamples is how many negative samples per positive.
	// Default: 5.
	NumNegativeSamples int `json:"num_negative_samples"`
}

// TimeAwareCFConfigMain contains parameters for time-aware collaborative filtering.
type TimeAwareCFConfigMain struct {
	// DecayRate controls how fast older interactions lose weight.
	// Default: 0.1.
	DecayRate float64 `json:"decay_rate"`

	// DecayUnitHours is the time unit for decay calculation in hours.
	// Default: 24.
	DecayUnitHours int `json:"decay_unit_hours"`

	// MaxLookbackDays is the maximum age of interactions to consider.
	// Default: 365.
	MaxLookbackDays int `json:"max_lookback_days"`

	// MinWeight is the minimum weight an interaction can have.
	// Default: 0.01.
	MinWeight float64 `json:"min_weight"`

	// NumNeighbors is the number of similar users/items to consider.
	// Default: 50.
	NumNeighbors int `json:"num_neighbors"`

	// Mode determines whether to use user-based or item-based CF.
	// Default: "user".
	Mode string `json:"mode"`
}

// MultiHopItemCFConfigMain contains parameters for multi-hop item-based CF.
type MultiHopItemCFConfigMain struct {
	// NumHops is the number of similarity propagation hops.
	// Default: 2.
	NumHops int `json:"num_hops"`

	// TopKPerHop is the number of similar items to consider at each hop.
	// Default: 10.
	TopKPerHop int `json:"top_k_per_hop"`

	// DecayFactor controls how much scores decay per hop.
	// Default: 0.5.
	DecayFactor float64 `json:"decay_factor"`

	// MinSimilarity is the minimum similarity threshold.
	// Default: 0.1.
	MinSimilarity float64 `json:"min_similarity"`
}

// MarkovChainConfigMain contains parameters for simple Markov chain.
type MarkovChainConfigMain struct {
	// MinTransitionCount is the minimum number of times a transition must occur.
	// Default: 2.
	MinTransitionCount int `json:"min_transition_count"`

	// MaxTransitionsPerItem limits memory usage.
	// Default: 50.
	MaxTransitionsPerItem int `json:"max_transitions_per_item"`

	// SessionWindowHours defines session grouping window.
	// Default: 6.
	SessionWindowHours int `json:"session_window_hours"`

	// SmoothingAlpha is the Laplace smoothing parameter.
	// Default: 0.1.
	SmoothingAlpha float64 `json:"smoothing_alpha"`
}

// DiversityConfig contains parameters for diversity reranking.
type DiversityConfig struct {
	// MMRLambda balances relevance vs. diversity in MMR reranking.
	// 1.0 = pure relevance, 0.0 = pure diversity.
	// Default: 0.7.
	MMRLambda float64 `json:"mmr_lambda"`

	// CalibrationLambda balances score vs. genre distribution matching.
	// 1.0 = pure score, 0.0 = pure calibration.
	// Default: 0.5.
	CalibrationLambda float64 `json:"calibration_lambda"`

	// MinGenreEntropy is the minimum genre entropy threshold.
	// Recommendations below this trigger calibration.
	// Default: 1.5.
	MinGenreEntropy float64 `json:"min_genre_entropy"`
}

// TrainingConfig contains training schedule parameters.
type TrainingConfig struct {
	// Interval is the time between scheduled training runs.
	// Default: 24h.
	Interval time.Duration `json:"interval"`

	// MinInteractions is the minimum number of interactions required to train.
	// Training is skipped if below this threshold.
	// Default: 100.
	MinInteractions int `json:"min_interactions"`

	// MinUsers is the minimum number of unique users required to train.
	// Default: 5.
	MinUsers int `json:"min_users"`

	// MinItems is the minimum number of unique items required to train.
	// Default: 10.
	MinItems int `json:"min_items"`

	// Timeout is the maximum time allowed for a training run.
	// Default: 10m.
	Timeout time.Duration `json:"timeout"`

	// RetainVersions is the number of model versions to retain.
	// Default: 3.
	RetainVersions int `json:"retain_versions"`
}

// LimitsConfig contains operational limits.
type LimitsConfig struct {
	// MaxCandidates is the maximum number of candidate items to score.
	// Default: 1000.
	MaxCandidates int `json:"max_candidates"`

	// DefaultK is the default number of recommendations to return.
	// Default: 20.
	DefaultK int `json:"default_k"`

	// MaxK is the maximum allowed K value.
	// Default: 100.
	MaxK int `json:"max_k"`

	// PredictionTimeout is the maximum time for a single prediction.
	// Default: 5s.
	PredictionTimeout time.Duration `json:"prediction_timeout"`

	// MaxConcurrentRequests is the maximum concurrent recommendation requests.
	// Default: 100.
	MaxConcurrentRequests int `json:"max_concurrent_requests"`
}

// CacheConfig contains caching parameters.
type CacheConfig struct {
	// Enabled controls whether caching is active.
	// Default: true.
	Enabled bool `json:"enabled"`

	// TTL is the cache entry time-to-live.
	// Default: 5m.
	TTL time.Duration `json:"ttl"`

	// MaxEntries is the maximum number of cached entries.
	// Default: 10000.
	MaxEntries int `json:"max_entries"`

	// InvalidateOnTrain controls whether cache is cleared after training.
	// Default: true.
	InvalidateOnTrain bool `json:"invalidate_on_train"`
}

// DefaultConfig returns a Config with sensible production defaults.
func DefaultConfig() *Config {
	return &Config{
		Weights: AlgorithmWeights{
			EASE:       0.15,
			ALS:        0.15,
			UserCF:     0.10,
			ItemCF:     0.10,
			Content:    0.15,
			CoVisit:    0.10,
			SASRec:     0.10,
			FPMC:       0.05,
			Popularity: 0.05,
			Recency:    0.05,
		},
		EASE: EASEConfig{
			Lambda:   500.0,
			MaxItems: 10000,
		},
		ALS: ALSConfig{
			Factors:    64,
			Lambda:     0.1,
			Alpha:      40.0,
			Iterations: 15,
		},
		ContentBased: ContentBasedConfig{
			GenreWeight:       0.4,
			ActorWeight:       0.3,
			DirectorWeight:    0.2,
			YearWeight:        0.1,
			MaxYearDifference: 20,
		},
		CoVisit: CoVisitConfig{
			MinCoOccurrence:    2,
			SessionWindowHours: 24,
			MaxPairs:           100000,
		},
		BPR: BPRConfigMain{
			NumFactors:         64,
			LearningRate:       0.01,
			Regularization:     0.01,
			NumIterations:      100,
			NumNegativeSamples: 5,
		},
		TimeAwareCF: TimeAwareCFConfigMain{
			DecayRate:       0.1,
			DecayUnitHours:  24,
			MaxLookbackDays: 365,
			MinWeight:       0.01,
			NumNeighbors:    50,
			Mode:            "user",
		},
		MultiHopItemCF: MultiHopItemCFConfigMain{
			NumHops:       2,
			TopKPerHop:    10,
			DecayFactor:   0.5,
			MinSimilarity: 0.1,
		},
		MarkovChain: MarkovChainConfigMain{
			MinTransitionCount:    2,
			MaxTransitionsPerItem: 50,
			SessionWindowHours:    6,
			SmoothingAlpha:        0.1,
		},
		Diversity: DiversityConfig{
			MMRLambda:         0.7,
			CalibrationLambda: 0.5,
			MinGenreEntropy:   1.5,
		},
		Training: TrainingConfig{
			Interval:        24 * time.Hour,
			MinInteractions: 100,
			MinUsers:        5,
			MinItems:        10,
			Timeout:         10 * time.Minute,
			RetainVersions:  3,
		},
		Limits: LimitsConfig{
			MaxCandidates:         1000,
			DefaultK:              20,
			MaxK:                  100,
			PredictionTimeout:     5 * time.Second,
			MaxConcurrentRequests: 100,
		},
		Cache: CacheConfig{
			Enabled:           true,
			TTL:               5 * time.Minute,
			MaxEntries:        10000,
			InvalidateOnTrain: true,
		},
		Seed: 42, // Default seed for determinism
	}
}

// Validate checks the configuration for errors.
//
//nolint:gocyclo // validation needs to check many fields
func (c *Config) Validate() error {
	if c.EASE.Lambda < 0 {
		return fmt.Errorf("ease.lambda must be non-negative, got %f", c.EASE.Lambda)
	}
	if c.EASE.MaxItems < 1 {
		return fmt.Errorf("ease.max_items must be positive, got %d", c.EASE.MaxItems)
	}

	if c.ALS.Factors < 1 {
		return fmt.Errorf("als.factors must be positive, got %d", c.ALS.Factors)
	}
	if c.ALS.Lambda < 0 {
		return fmt.Errorf("als.lambda must be non-negative, got %f", c.ALS.Lambda)
	}
	if c.ALS.Alpha < 0 {
		return fmt.Errorf("als.alpha must be non-negative, got %f", c.ALS.Alpha)
	}
	if c.ALS.Iterations < 1 {
		return fmt.Errorf("als.iterations must be positive, got %d", c.ALS.Iterations)
	}

	if c.Diversity.MMRLambda < 0 || c.Diversity.MMRLambda > 1 {
		return fmt.Errorf("diversity.mmr_lambda must be in [0, 1], got %f", c.Diversity.MMRLambda)
	}
	if c.Diversity.CalibrationLambda < 0 || c.Diversity.CalibrationLambda > 1 {
		return fmt.Errorf("diversity.calibration_lambda must be in [0, 1], got %f", c.Diversity.CalibrationLambda)
	}

	if c.Training.MinInteractions < 0 {
		return fmt.Errorf("training.min_interactions must be non-negative, got %d", c.Training.MinInteractions)
	}
	if c.Training.Timeout <= 0 {
		return fmt.Errorf("training.timeout must be positive, got %v", c.Training.Timeout)
	}

	if c.Limits.MaxCandidates < 1 {
		return fmt.Errorf("limits.max_candidates must be positive, got %d", c.Limits.MaxCandidates)
	}
	if c.Limits.DefaultK < 1 {
		return fmt.Errorf("limits.default_k must be positive, got %d", c.Limits.DefaultK)
	}
	if c.Limits.MaxK < c.Limits.DefaultK {
		return fmt.Errorf("limits.max_k must be >= limits.default_k, got %d < %d", c.Limits.MaxK, c.Limits.DefaultK)
	}

	return nil
}

// Clone returns a deep copy of the configuration.
func (c *Config) Clone() *Config {
	// Direct field copy - all nested structs contain only value types (no pointers/slices)
	return &Config{
		Weights:        c.Weights,
		EASE:           c.EASE,
		ALS:            c.ALS,
		ContentBased:   c.ContentBased,
		CoVisit:        c.CoVisit,
		BPR:            c.BPR,
		TimeAwareCF:    c.TimeAwareCF,
		MultiHopItemCF: c.MultiHopItemCF,
		MarkovChain:    c.MarkovChain,
		Diversity:      c.Diversity,
		Training:       c.Training,
		Limits:         c.Limits,
		Cache:          c.Cache,
		Seed:           c.Seed,
	}
}

// MarshalJSON implements custom JSON marshaling for duration fields.
func (c *Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	return json.Marshal(&struct {
		*Alias
		Training struct {
			Interval        string `json:"interval"`
			MinInteractions int    `json:"min_interactions"`
			MinUsers        int    `json:"min_users"`
			MinItems        int    `json:"min_items"`
			Timeout         string `json:"timeout"`
			RetainVersions  int    `json:"retain_versions"`
		} `json:"training"`
		Limits struct {
			MaxCandidates         int    `json:"max_candidates"`
			DefaultK              int    `json:"default_k"`
			MaxK                  int    `json:"max_k"`
			PredictionTimeout     string `json:"prediction_timeout"`
			MaxConcurrentRequests int    `json:"max_concurrent_requests"`
		} `json:"limits"`
		Cache struct {
			Enabled           bool   `json:"enabled"`
			TTL               string `json:"ttl"`
			MaxEntries        int    `json:"max_entries"`
			InvalidateOnTrain bool   `json:"invalidate_on_train"`
		} `json:"cache"`
	}{
		Alias: (*Alias)(c),
		Training: struct {
			Interval        string `json:"interval"`
			MinInteractions int    `json:"min_interactions"`
			MinUsers        int    `json:"min_users"`
			MinItems        int    `json:"min_items"`
			Timeout         string `json:"timeout"`
			RetainVersions  int    `json:"retain_versions"`
		}{
			Interval:        c.Training.Interval.String(),
			MinInteractions: c.Training.MinInteractions,
			MinUsers:        c.Training.MinUsers,
			MinItems:        c.Training.MinItems,
			Timeout:         c.Training.Timeout.String(),
			RetainVersions:  c.Training.RetainVersions,
		},
		Limits: struct {
			MaxCandidates         int    `json:"max_candidates"`
			DefaultK              int    `json:"default_k"`
			MaxK                  int    `json:"max_k"`
			PredictionTimeout     string `json:"prediction_timeout"`
			MaxConcurrentRequests int    `json:"max_concurrent_requests"`
		}{
			MaxCandidates:         c.Limits.MaxCandidates,
			DefaultK:              c.Limits.DefaultK,
			MaxK:                  c.Limits.MaxK,
			PredictionTimeout:     c.Limits.PredictionTimeout.String(),
			MaxConcurrentRequests: c.Limits.MaxConcurrentRequests,
		},
		Cache: struct {
			Enabled           bool   `json:"enabled"`
			TTL               string `json:"ttl"`
			MaxEntries        int    `json:"max_entries"`
			InvalidateOnTrain bool   `json:"invalidate_on_train"`
		}{
			Enabled:           c.Cache.Enabled,
			TTL:               c.Cache.TTL.String(),
			MaxEntries:        c.Cache.MaxEntries,
			InvalidateOnTrain: c.Cache.InvalidateOnTrain,
		},
	})
}
