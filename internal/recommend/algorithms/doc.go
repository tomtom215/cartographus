// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package algorithms implements recommendation algorithms for the hybrid engine.
//
// This package provides multiple recommendation algorithm implementations that
// can be combined in an ensemble to produce personalized content recommendations.
// Each algorithm implements the recommend.Algorithm interface and can be
// independently trained, persisted, and queried.
//
// # Algorithm Categories
//
// Collaborative Filtering:
//   - EASE: Embarrassingly Shallow Autoencoders (linear, scalable)
//   - ALS: Alternating Least Squares matrix factorization
//   - UserBasedCF: User-user similarity collaborative filtering
//   - ItemBasedCF: Item-item similarity collaborative filtering
//
// Content-Based Filtering:
//   - ContentBased: Similarity based on metadata (genres, cast, directors)
//
// Sequential Patterns:
//   - CoVisitation: Item co-occurrence in sessions
//   - FPMC: Factorized Personalized Markov Chains
//
// Contextual Bandits:
//   - LinUCB: Linear Upper Confidence Bound for exploration/exploitation
//
// Baselines:
//   - Popularity: Global or time-decayed popularity ranking
//
// # Interface
//
// All algorithms implement the recommend.Algorithm interface:
//
//	type Algorithm interface {
//	    Name() string
//	    Train(ctx context.Context, data TrainingData) error
//	    Predict(ctx context.Context, req Request) (map[int]float64, error)
//	    IsTrained() bool
//	    Version() int
//	    LastTrainedAt() time.Time
//	}
//
// # Usage Example
//
// Training and using an algorithm:
//
//	// Create EASE algorithm
//	ease := algorithms.NewEASE(algorithms.EASEConfig{
//	    L2Regularization: 500.0,
//	    MinConfidence:    0.1,
//	})
//
//	// Train on interaction data
//	data := recommend.TrainingData{
//	    Interactions: interactions,
//	    Items:        items,
//	}
//	if err := ease.Train(ctx, data); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get recommendations
//	req := recommend.Request{
//	    UserID: 123,
//	    K:      20,
//	}
//	scores, err := ease.Predict(ctx, req)
//
// # Algorithm Selection Guide
//
// Choose algorithms based on data availability and use case:
//
// Cold-start users (new users with no history):
//   - ContentBased: Uses item metadata similarity
//   - Popularity: Global popularity fallback
//
// Cold-start items (new items with no interactions):
//   - ContentBased: Metadata-based similarity
//
// Established users with history:
//   - EASE: Fast, linear, interpretable (recommended default)
//   - ALS: Good for sparse data
//   - CoVisitation: Session-based patterns
//
// Sequential viewing patterns:
//   - FPMC: Markov chain with personalization
//   - CoVisitation: Simple co-occurrence
//
// # Training Data
//
// Algorithms require interaction data in this format:
//
//	type Interaction struct {
//	    UserID     int
//	    ItemID     int
//	    Confidence float64   // Implicit signal strength (e.g., watch time ratio)
//	    Timestamp  time.Time // For sequential algorithms
//	}
//
// # Thread Safety
//
// All algorithms are designed for concurrent use:
//   - Training: Acquires exclusive lock
//   - Prediction: Acquires shared lock
//   - Multiple predictions can run concurrently
//   - Training blocks predictions until complete
//
// # Base Algorithm
//
// The BaseAlgorithm type provides common functionality:
//
//	type BaseAlgorithm struct {
//	    name          string
//	    trained       bool
//	    version       int
//	    lastTrainedAt time.Time
//	    mu            sync.RWMutex
//	}
//
// All algorithm implementations embed BaseAlgorithm to inherit:
//   - Name() string
//   - IsTrained() bool
//   - Version() int
//   - LastTrainedAt() time.Time
//   - Lock management methods
//
// # Utility Functions
//
// The package provides helper functions for algorithm implementations:
//
//   - normalizeScores: Min-max normalization to [0, 1]
//   - cosineSimilarity: Vector similarity calculation
//   - jaccardSimilarity: Set-based similarity for categorical data
//   - ContextCancelled: Check for context cancellation
//
// # Performance Considerations
//
// Training Complexity:
//   - EASE: O(n^2) where n is number of items
//   - ALS: O(k * (m + n)) per iteration, k iterations
//   - ContentBased: O(n * m) where m is user count
//   - CoVisitation: O(s * w^2) where s=sessions, w=window size
//
// Prediction Complexity:
//   - EASE: O(n) per user
//   - ALS: O(k) per user
//   - ContentBased: O(n) per user
//   - CoVisitation: O(k) per user
//
// # See Also
//
//   - internal/recommend: Engine and interface definitions
//   - internal/recommend/storage: Model persistence
//   - internal/recommend/reranking: Diversity reranking
//   - docs/adr/0024-recommendation-engine.md: Architecture decision
package algorithms
