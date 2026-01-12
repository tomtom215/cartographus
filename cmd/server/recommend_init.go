// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package main

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/recommend"
	"github.com/tomtom215/cartographus/internal/recommend/algorithms"
	"github.com/tomtom215/cartographus/internal/recommend/reranking"
	"github.com/tomtom215/cartographus/internal/supervisor"
	"github.com/tomtom215/cartographus/internal/supervisor/services"
)

// RecommendComponents holds all recommendation-related components.
type RecommendComponents struct {
	Engine  *recommend.Engine
	Service *services.RecommendService
}

// algorithmRegistrar holds dependencies for algorithm registration.
type algorithmRegistrar struct {
	engine       *recommend.Engine
	cfg          *config.Config
	algorithmSet map[string]bool
	logger       zerolog.Logger
}

// initRecommend initializes the recommendation engine if enabled.
// Returns nil if recommendations are disabled in config.
//
//nolint:gocritic // hugeParam: logger passed by value for zerolog chaining
func initRecommend(cfg *config.Config, logger zerolog.Logger, tree *supervisor.SupervisorTree) *RecommendComponents {
	// Check if recommendations are disabled
	if !cfg.Recommend.Enabled {
		logger.Info().Msg("Recommendation engine disabled (RECOMMEND_ENABLED=false)")
		return nil
	}

	logger.Info().
		Strs("algorithms", cfg.Recommend.Algorithms).
		Dur("train_interval", cfg.Recommend.TrainInterval).
		Bool("train_on_startup", cfg.Recommend.TrainOnStartup).
		Int("min_interactions", cfg.Recommend.MinInteractions).
		Msg("initializing recommendation engine")

	// Create engine
	engine, err := recommend.NewEngine(buildEngineConfig(cfg), logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create recommendation engine")
		return nil
	}

	// Register algorithms based on configuration
	registrar := &algorithmRegistrar{
		engine:       engine,
		cfg:          cfg,
		algorithmSet: buildAlgorithmSet(cfg.Recommend.Algorithms),
		logger:       logger,
	}
	registrar.registerAllAlgorithms()

	// Register rerankers
	registerRerankers(engine, cfg, logger)

	// Create service for Suture
	serviceCfg := services.RecommendServiceConfig{
		TrainOnStartup:  cfg.Recommend.TrainOnStartup,
		TrainInterval:   cfg.Recommend.TrainInterval,
		MinInteractions: cfg.Recommend.MinInteractions,
	}
	service := services.NewRecommendService(engine, serviceCfg, logger)

	// Add to supervisor tree
	tree.AddMessagingService(service)
	logger.Info().
		Int("algorithms", len(cfg.Recommend.Algorithms)).
		Msg("recommendation service added to supervisor tree")

	return &RecommendComponents{
		Engine:  engine,
		Service: service,
	}
}

// buildEngineConfig creates the engine configuration from app config.
func buildEngineConfig(cfg *config.Config) *recommend.Config {
	return &recommend.Config{
		Seed: 42, // Deterministic for reproducibility
		Weights: recommend.AlgorithmWeights{
			CoVisit:    1.0,
			Content:    0.8,
			Popularity: 0.5,
			EASE:       1.2,
			ALS:        1.0,
			UserCF:     0.7,
			ItemCF:     0.7,
			FPMC:       0.6,
		},
		Limits: recommend.LimitsConfig{
			DefaultK:      10,
			MaxK:          100,
			MaxCandidates: cfg.Recommend.MaxCandidates,
		},
		Cache: recommend.CacheConfig{
			Enabled:           true,
			TTL:               cfg.Recommend.CacheTTL,
			MaxEntries:        10000,
			InvalidateOnTrain: true,
		},
		Training: recommend.TrainingConfig{
			MinInteractions: cfg.Recommend.MinInteractions,
		},
		Diversity: recommend.DiversityConfig{
			MMRLambda: cfg.Recommend.DiversityLambda,
		},
	}
}

// buildAlgorithmSet converts algorithm slice to set for O(1) lookup.
func buildAlgorithmSet(algs []string) map[string]bool {
	set := make(map[string]bool, len(algs))
	for _, alg := range algs {
		set[alg] = true
	}
	return set
}

// registerAllAlgorithms registers all enabled algorithms by phase.
func (r *algorithmRegistrar) registerAllAlgorithms() {
	r.registerLightweightAlgorithms()
	r.registerMatrixFactorization()
	r.registerCollaborativeFiltering()
	r.registerSequentialAlgorithms()
	r.registerAdvancedAlgorithms()
	r.registerBanditAlgorithms()
}

// registerLightweightAlgorithms registers Phase 1 algorithms.
func (r *algorithmRegistrar) registerLightweightAlgorithms() {
	if r.algorithmSet["covisit"] {
		r.engine.RegisterAlgorithm(algorithms.NewCoVisitation(algorithms.CoVisitConfig{
			MinCoOccurrence:    2,
			SessionWindowHours: 24,
			MaxPairs:           100000,
		}))
		r.logger.Debug().Msg("registered co-visitation algorithm")
	}

	if r.algorithmSet["content"] {
		r.engine.RegisterAlgorithm(algorithms.NewContentBased(algorithms.ContentBasedConfig{
			GenreWeight:       0.5,
			ActorWeight:       0.25,
			DirectorWeight:    0.15,
			YearWeight:        0.1,
			MaxYearDifference: 10,
		}))
		r.logger.Debug().Msg("registered content-based algorithm")
	}

	if r.algorithmSet["popularity"] {
		r.engine.RegisterAlgorithm(algorithms.NewPopularity(algorithms.PopularityConfig{
			UseTimeDecay:  true,
			DecayHalfLife: 30,
			MaxItems:      10000,
		}))
		r.logger.Debug().Msg("registered popularity algorithm")
	}
}

// registerMatrixFactorization registers Phase 2 algorithms.
func (r *algorithmRegistrar) registerMatrixFactorization() {
	if r.algorithmSet["ease"] {
		r.engine.RegisterAlgorithm(algorithms.NewEASE(algorithms.EASEConfig{
			L2Regularization: r.cfg.Recommend.EASE.L2Regularization,
			MinConfidence:    r.cfg.Recommend.EASE.MinConfidence,
		}))
		r.logger.Debug().Msg("registered EASE algorithm")
	}
}

// registerCollaborativeFiltering registers Phase 3 algorithms.
func (r *algorithmRegistrar) registerCollaborativeFiltering() {
	if r.algorithmSet["als"] {
		r.engine.RegisterAlgorithm(algorithms.NewALS(algorithms.ALSConfig{
			NumFactors:     r.cfg.Recommend.ALS.Factors,
			NumIterations:  r.cfg.Recommend.ALS.Iterations,
			Regularization: r.cfg.Recommend.ALS.Regularization,
			Alpha:          r.cfg.Recommend.ALS.Alpha,
			NumWorkers:     r.cfg.Recommend.ALS.NumWorkers,
		}))
		r.logger.Debug().Msg("registered ALS algorithm")
	}

	if r.algorithmSet["usercf"] {
		r.engine.RegisterAlgorithm(algorithms.NewUserBasedCF(algorithms.KNNConfig{
			K:                r.cfg.Recommend.KNN.Neighbors,
			SimilarityMetric: r.cfg.Recommend.KNN.Similarity,
			Shrinkage:        r.cfg.Recommend.KNN.Shrinkage,
		}))
		r.logger.Debug().Msg("registered user-based CF algorithm")
	}

	if r.algorithmSet["itemcf"] {
		r.engine.RegisterAlgorithm(algorithms.NewItemBasedCF(algorithms.KNNConfig{
			K:                r.cfg.Recommend.KNN.Neighbors,
			SimilarityMetric: r.cfg.Recommend.KNN.Similarity,
			Shrinkage:        r.cfg.Recommend.KNN.Shrinkage,
		}))
		r.logger.Debug().Msg("registered item-based CF algorithm")
	}
}

// registerSequentialAlgorithms registers Phase 4 algorithms.
func (r *algorithmRegistrar) registerSequentialAlgorithms() {
	if r.algorithmSet["fpmc"] {
		r.engine.RegisterAlgorithm(algorithms.NewFPMC(algorithms.FPMCConfig{
			NumFactors:      r.cfg.Recommend.FPMC.Factors,
			LearningRate:    r.cfg.Recommend.FPMC.LearningRate,
			Regularization:  r.cfg.Recommend.FPMC.Regularization,
			NumIterations:   r.cfg.Recommend.FPMC.Epochs,
			NegativeSamples: r.cfg.Recommend.FPMC.NegativeSamples,
		}))
		r.logger.Debug().Msg("registered FPMC algorithm")
	}

	if r.algorithmSet["markov"] {
		r.engine.RegisterAlgorithm(algorithms.NewMarkovChain(algorithms.MarkovChainConfig{
			OrderK:                1,
			MinTransitionCount:    2,
			MaxTransitionsPerItem: 50,
			SessionWindowSeconds:  21600, // 6 hours
			SmoothingAlpha:        0.1,
		}))
		r.logger.Debug().Msg("registered Markov Chain algorithm")
	}
}

// registerAdvancedAlgorithms registers Phase 5 algorithms.
func (r *algorithmRegistrar) registerAdvancedAlgorithms() {
	if r.algorithmSet["bpr"] {
		r.engine.RegisterAlgorithm(algorithms.NewBPR(algorithms.BPRConfig{
			NumFactors:         64,
			LearningRate:       0.01,
			Regularization:     0.01,
			NumIterations:      100,
			NumNegativeSamples: 5,
			Seed:               42,
		}))
		r.logger.Debug().Msg("registered BPR algorithm")
	}

	if r.algorithmSet["timeaware"] {
		r.engine.RegisterAlgorithm(algorithms.NewTimeAwareCF(&algorithms.TimeAwareCFConfig{
			DecayRate:    0.1,
			DecayUnit:    24 * time.Hour,
			MaxLookback:  365 * 24 * time.Hour,
			MinWeight:    0.01,
			NumNeighbors: 50,
			Mode:         "user",
		}))
		r.logger.Debug().Msg("registered Time-Aware CF algorithm")
	}

	if r.algorithmSet["multihop"] {
		r.engine.RegisterAlgorithm(algorithms.NewMultiHopItemCF(algorithms.MultiHopItemCFConfig{
			NumHops:       2,
			TopKPerHop:    10,
			DecayFactor:   0.5,
			MinSimilarity: 0.1,
		}))
		r.logger.Debug().Msg("registered Multi-Hop ItemCF algorithm")
	}
}

// registerBanditAlgorithms registers Phase 6 algorithms.
func (r *algorithmRegistrar) registerBanditAlgorithms() {
	if r.algorithmSet["linucb"] {
		r.engine.RegisterAlgorithm(algorithms.NewLinUCB(algorithms.LinUCBConfig{
			Alpha:       r.cfg.Recommend.LinUCB.Alpha,
			NumFeatures: r.cfg.Recommend.LinUCB.NumFeatures,
			DecayRate:   r.cfg.Recommend.LinUCB.DecayRate,
		}))
		r.logger.Debug().Msg("registered LinUCB algorithm")
	}
}

// registerRerankers registers all reranking strategies.
//
//nolint:gocritic // hugeParam: logger passed by value for zerolog chaining
func registerRerankers(engine *recommend.Engine, cfg *config.Config, logger zerolog.Logger) {
	mmr := reranking.NewMMR(cfg.Recommend.DiversityLambda)
	engine.RegisterReranker(mmr)
	logger.Debug().Float64("lambda", cfg.Recommend.DiversityLambda).Msg("registered MMR reranker")

	if cfg.Recommend.CalibrationEnabled {
		calibration := reranking.NewCalibration(reranking.CalibrationConfig{
			Lambda: 0.5,
			AttributeWeights: map[string]float64{
				"genre": 0.6,
				"year":  0.4,
			},
		})
		engine.RegisterReranker(calibration)
		logger.Debug().Msg("registered calibration reranker")
	}
}
