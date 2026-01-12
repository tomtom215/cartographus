// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package recommend implements a hybrid recommendation engine for media content.
//
// # Architecture
//
// The recommendation engine combines multiple algorithm families to produce
// personalized content recommendations:
//
//   - Collaborative Filtering: EASE, ALS, User-based CF, Item-based CF
//   - Content-Based Filtering: Genre/cast/director similarity
//   - Sequential Models: Co-visitation, Markov chains
//   - Diversity Reranking: MMR (Maximal Marginal Relevance)
//
// # Design Principles
//
// The engine is designed with the following production-grade requirements:
//
//   - Deterministic: Same inputs produce identical outputs (seeded RNG)
//   - Reproducible: Results are consistent across runs
//   - Auditable: All operations are logged with structured fields
//   - Observable: Metrics exposed for monitoring
//   - Durable: Model state persisted to DuckDB
//   - Traceable: Request IDs propagated through context
//
// # Algorithm Selection
//
// Algorithms are selected based on data availability and cold-start conditions:
//
//   - New users: Content-based + popularity fallback
//   - New items: Content-based similarity
//   - Established users: Full hybrid ensemble
//
// # Usage
//
//	cfg := recommend.DefaultConfig()
//	engine := recommend.NewEngine(db, cfg, logger)
//
//	// Register algorithms
//	engine.RegisterAlgorithm(algorithms.NewEASE(cfg.EASE))
//	engine.RegisterAlgorithm(algorithms.NewContentBased())
//
//	// Get recommendations
//	recs, err := engine.Recommend(ctx, recommend.Request{
//	    UserID: userID,
//	    K:      20,
//	})
//
// # Thread Safety
//
// The engine is safe for concurrent use. Training operations acquire an
// exclusive lock, while prediction operations use a shared lock. This
// allows concurrent reads during normal operation while ensuring
// consistency during model updates.
//
// # References
//
// See ADR-0024 for architecture decisions and algorithm selection rationale.
package recommend
