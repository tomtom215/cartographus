# ADR-0024: Hybrid Recommendation Engine for Media Content

**Date**: 2025-12-29
**Status**: Accepted

---

## Context

Cartographus aggregates playback data from multiple media servers (Plex, Jellyfin, Emby) via Tautulli and direct integrations. This rich dataset creates an opportunity to provide personalized content recommendations to users. Currently, users must manually browse their media libraries with no algorithmic assistance.

### Use Cases Requiring Recommendations

1. **Personalized Suggestions**: "What should I watch next?" based on viewing history
2. **Continue Watching**: Surface in-progress content with smart prioritization
3. **Similar Content**: "More like this" recommendations for specific titles
4. **Discovery/Exploration**: Surface content users might not find on their own
5. **Multi-User Households**: Per-user recommendations within shared libraries

### Requirements Evaluated

1. **Pure Go Implementation**: No external ML dependencies (TensorFlow, PyTorch)
2. **Deterministic Outputs**: Same inputs must produce identical results
3. **Consumer Hardware Target**: Must run on modest hardware (4-16GB RAM)
4. **Offline-First**: No external API calls or embedding services
5. **Multi-Server Support**: Unified recommendations across Plex, Jellyfin, Emby
6. **Cold Start Handling**: Recommendations for new users and new content

### Data Availability Assessment

Investigation of the existing Cartographus data model (`internal/models/playback.go`) confirmed extensive data availability:

| Category | Fields Available | Sufficiency |
|----------|------------------|-------------|
| **User-Item Interactions** | 280+ fields including percent_complete, play_duration, paused_counter | Exceeds requirements |
| **Content Metadata** | genres, directors, writers, actors, year, studio, rating, audience_rating | Full coverage |
| **Sequential Data** | media_index, parent_media_index, started_at for episode ordering | Full coverage |
| **User Behavior** | Binge patterns, completion rates, device preferences | Already analyzed |
| **Multi-Server Identity** | user_mappings table with unified internal_user_id | Supported |

### Existing Infrastructure

Cartographus already implements related functionality:

- **Binge Detection** (`internal/database/analytics_binge.go`): Identifies 3+ episode sessions within 6-hour windows
- **User Clustering** (`internal/database/analytics_user_network.go`): Connected components algorithm for user similarity
- **Content Discovery** (`internal/database/analytics_content_discovery.go`): Early adopter detection and discovery velocity
- **Fuzzy Matching** (`internal/database/search_fuzzy.go`): RapidFuzz integration for string similarity
- **Approximate Analytics** (`internal/database/analytics_approximate.go`): DataSketches for HLL/KLL

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **External Service (Spotify-like)** | Powerful algorithms | Requires network, not self-hosted |
| **Python ML Backend** | Full ecosystem (scikit-learn, PyTorch) | CGO complexity, deployment overhead |
| **Rule-Based Only** | Simple implementation | Limited personalization |
| **Hybrid Pure Go** | Self-contained, deterministic, offline | Implementation effort |

---

## Decision

Implement a **hybrid recommendation engine** in pure Go as an internal package (`internal/recommend`) that combines multiple algorithm families:

1. **Collaborative Filtering**: EASE, ALS, User-based CF, Item-based CF, BPR, Time-Aware CF, Multi-Hop ItemCF
2. **Content-Based Filtering**: Genre/cast/director similarity
3. **Sequential Models**: FPMC, Markov chains
4. **Contextual Bandits**: LinUCB for exploration/exploitation
5. **Diversity Reranking**: MMR, calibration for genre distribution

### Architecture

```
                    Recommendation Engine Architecture
                    ══════════════════════════════════

┌─────────────────────────────────────────────────────────────────────────┐
│                         Recommendation Engine                            │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                      Request Context                                 ││
│  │  (UserID, CurrentItem, Time, Device, K, ExcludeList)               ││
│  └─────────────────────────────────────────────────────────────────────┘│
│                                    │                                     │
│         ┌──────────────────────────┼──────────────────────────┐         │
│         │                          │                          │         │
│         ▼                          ▼                          ▼         │
│  ┌──────────────┐          ┌──────────────┐          ┌──────────────┐  │
│  │ Collaborative│          │ Content-Based│          │  Sequential  │  │
│  │   Filtering  │          │   Filtering  │          │    Models    │  │
│  │              │          │              │          │              │  │
│  │  - EASE      │          │  - Genre     │          │  - FPMC      │  │
│  │  - ALS       │          │  - Cast      │          │  - Markov    │  │
│  │  - User CF   │          │  - Director  │          │  - CoVisit   │  │
│  │  - Item CF   │          │  - Studio    │          │              │  │
│  │  - BPR       │          │  - Year      │          │              │  │
│  └──────────────┘          └──────────────┘          └──────────────┘  │
│         │                          │                          │         │
│         └──────────────────────────┼──────────────────────────┘         │
│                                    ▼                                     │
│                         ┌──────────────────┐                            │
│                         │  Score Fusion    │                            │
│                         │  (Weighted Sum)  │                            │
│                         └──────────────────┘                            │
│                                    │                                     │
│         ┌──────────────────────────┼──────────────────────────┐         │
│         ▼                          ▼                          ▼         │
│  ┌──────────────┐          ┌──────────────┐          ┌──────────────┐  │
│  │     MMR      │          │ Calibration  │          │   LinUCB     │  │
│  │  Diversity   │          │    Genre     │          │ Exploration  │  │
│  │  Reranking   │          │ Distribution │          │   Bonus      │  │
│  └──────────────┘          └──────────────┘          └──────────────┘  │
│                                    │                                     │
│                                    ▼                                     │
│                         ┌──────────────────┐                            │
│                         │ Final Rankings   │                            │
│                         │ (Top-K Items)    │                            │
│                         └──────────────────┘                            │
│                                    │                                     │
│         ┌──────────────────────────┼──────────────────────────┐         │
│         ▼                          ▼                          ▼         │
│  ┌──────────────┐          ┌──────────────┐          ┌──────────────┐  │
│  │   DuckDB     │          │  WebSocket   │          │  REST API    │  │
│  │   Storage    │          │  Broadcast   │          │  Response    │  │
│  └──────────────┘          └──────────────┘          └──────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

### Algorithm Selection

| Algorithm | Priority | Rationale | Data Requirement |
|-----------|----------|-----------|------------------|
| **EASE** | Tier 1 | Closed-form, no iteration, proven performance | User-item matrix |
| **ALS** | Tier 1 | Industry standard for implicit feedback | User-item matrix |
| **Content-Based** | Tier 1 | Cold start fallback, interpretable | genres, actors, directors columns |
| **Co-Visitation** | Tier 1 | O(1) lookup, session-based | Session groupings |
| **User CF** | Tier 1 | Extend existing user clustering | User network graph |
| **Item CF** | Tier 1 | Handles item cold start | Co-occurrence matrix |
| **FPMC** | Tier 2 | Markov transitions | Session sequences |
| **Markov Chain** | Tier 1 | Sequential viewing patterns | Session sequences |
| **BPR** | Tier 2 | Bayesian Personalized Ranking | User-item matrix |
| **Time-Aware CF** | Tier 2 | Temporal decay on interactions | Timestamped interactions |
| **Multi-Hop ItemCF** | Tier 2 | Graph-based item similarity | Co-occurrence graph |
| **MMR** | Tier 1 | Diversity via dissimilarity | Item similarity matrix |
| **Calibration** | Tier 1 | Genre distribution matching | User genre history |
| **LinUCB** | Tier 2 | Principled exploration | Context features |
| **Popularity** | Tier 1 | Baseline trending items | Play counts |

### Implicit Feedback Model

Unlike explicit rating systems, Cartographus uses behavioral signals as implicit feedback.
See `internal/recommend/types.go` for the full implementation:

```go
// ClassifyInteraction classifies an interaction based on completion percentage.
func ClassifyInteraction(percentComplete int) InteractionType {
    switch {
    case percentComplete >= 90:
        return InteractionCompleted  // Strong positive
    case percentComplete >= 50:
        return InteractionEngaged    // Moderate positive
    case percentComplete >= 10:
        return InteractionSampled    // Weak positive
    default:
        return InteractionAbandoned  // Potential negative
    }
}

// ComputeConfidence computes the confidence score for an interaction.
// Higher confidence indicates stronger preference signal.
func ComputeConfidence(percentComplete, playDurationSeconds int) float64 {
    // Base confidence from completion
    c := 1.0 + 0.1*float64(percentComplete)

    // Boost for longer engagement (diminishing returns via log scale)
    if playDurationSeconds > 0 {
        c += 0.5 * (1.0 - 1.0/(1.0+float64(playDurationSeconds)/3600.0))
    }

    return c
}
```

### TV Series Specific Features

The engine includes specialized handling for episodic content:

1. **Continue Watching**: Score based on recency, completion momentum, episodes remaining
2. **Binge Propensity**: Leverage existing binge detection analytics
3. **Episode Ordering**: Use media_index/parent_media_index for sequence modeling
4. **Show-Level Aggregation**: Aggregate episode signals to show-level preferences

---

## Consequences

### Positive

- **Self-Contained**: No external dependencies, fully offline capable
- **Deterministic**: Reproducible results with seeded random operations
- **Multi-Server**: Unified recommendations across Plex, Jellyfin, Emby via user_mappings
- **Extensible**: Modular algorithm interface allows adding new methods
- **Efficient**: Leverages existing DuckDB infrastructure and extensions
- **Testable**: Pure Go enables comprehensive unit testing
- **Rich Signals**: 280+ fields of behavioral data exceed typical recommender inputs

### Negative

- **Development Effort**: Implementing 10+ algorithms requires significant work
- **Memory Usage**: Matrix factorization models consume memory (O(items^2) for EASE)
- **Training Latency**: Full model retraining may take minutes on large libraries
- **Cold Start**: New users require fallback to content-based or popularity

### Neutral

- **DuckDB Dependency**: Training queries leverage existing DuckDB connection
- **Periodic Retraining**: Models need scheduled updates (daily or on-demand)
- **Supervisor Integration**: Recommendation service added to Suture tree

---

## Implementation

### File Structure

```
internal/recommend/
├── doc.go                      # Package documentation
├── engine.go                   # Main recommendation engine
├── engine_test.go              # Engine tests
├── config.go                   # Algorithm weights and parameters
├── config_test.go              # Configuration tests
├── types.go                    # Core types and interfaces
├── types_test.go               # Types tests
│
├── algorithms/                 # Algorithm implementations
│   ├── doc.go                  # Package documentation
│   ├── interface.go            # Algorithm interface and utilities
│   ├── ease.go                 # EASE^R closed-form
│   ├── ease_test.go
│   ├── als.go                  # Alternating Least Squares
│   ├── als_test.go
│   ├── knn.go                  # User-CF and Item-CF (KNN-based)
│   ├── knn_test.go
│   ├── content.go              # Content-based filtering
│   ├── content_test.go
│   ├── covisit.go              # Co-visitation matrix
│   ├── covisit_test.go
│   ├── fpmc.go                 # Factorized Personalized Markov Chains
│   ├── fpmc_test.go
│   ├── markov.go               # Simple Markov chain
│   ├── markov_test.go
│   ├── bpr.go                  # Bayesian Personalized Ranking
│   ├── bpr_test.go
│   ├── timecf.go               # Time-aware collaborative filtering
│   ├── timecf_test.go
│   ├── multihop.go             # Multi-hop item-based CF
│   ├── multihop_test.go
│   ├── linucb.go               # Contextual bandits (LinUCB)
│   ├── linucb_test.go
│   └── popularity.go           # Popularity-based baseline
│
├── reranking/                  # Post-processing
│   ├── doc.go                  # Package documentation
│   ├── mmr.go                  # Maximal Marginal Relevance
│   ├── mmr_test.go
│   ├── calibration.go          # Genre distribution calibration
│   └── calibration_test.go
│
└── storage/                    # Model persistence
    ├── doc.go                  # Package documentation
    ├── models.go               # Model artifact storage (gob + gzip)
    └── models_test.go

internal/database/
├── analytics_recommend.go      # DuckDB queries for recommendations
└── analytics_recommend_test.go

internal/supervisor/services/
├── recommend_service.go        # Suture service wrapper
└── recommend_service_test.go

internal/api/
└── handlers_recommend.go       # HTTP API handlers
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/recommendations/status` | Training status and metrics |
| GET | `/api/v1/recommendations/config` | Get algorithm weights |
| PUT | `/api/v1/recommendations/config` | Update algorithm weights |
| POST | `/api/v1/recommendations/train` | Trigger model retraining (admin) |
| GET | `/api/v1/recommendations/algorithms` | List available algorithms |
| GET | `/api/v1/recommendations/algorithms/metrics` | Per-algorithm metrics |
| GET | `/api/v1/recommendations/user/{userID}` | Personalized recommendations |
| GET | `/api/v1/recommendations/user/{userID}/continue` | Continue watching list |
| GET | `/api/v1/recommendations/user/{userID}/explore` | Discovery/exploration mode |
| GET | `/api/v1/recommendations/similar/{itemID}` | Similar content |
| GET | `/api/v1/recommendations/next/{itemID}` | Sequential "what's next" predictions |

### Core Interfaces

```go
// Algorithm defines the interface all recommendation algorithms must implement.
// See internal/recommend/types.go for the full definition.
type Algorithm interface {
    // Name returns the algorithm identifier (e.g., "ease", "content", "covisit")
    Name() string

    // Train fits the model on interaction data and item metadata
    Train(ctx context.Context, interactions []Interaction, items []Item) error

    // Predict returns scores for candidate items for a user
    Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error)

    // PredictSimilar returns items similar to the given item
    PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error)

    // IsTrained returns whether the model has been trained
    IsTrained() bool

    // Version returns the model version (incremented on each train)
    Version() int

    // LastTrainedAt returns when the model was last trained
    LastTrainedAt() time.Time
}

// Reranker modifies a ranked list for diversity or other objectives.
// See internal/recommend/types.go for the full definition.
type Reranker interface {
    // Name returns the reranker identifier (e.g., "mmr", "calibration")
    Name() string

    // Rerank modifies the order of scored items to optimize a secondary objective
    Rerank(ctx context.Context, items []ScoredItem, k int) []ScoredItem
}

// DataProvider defines the interface for fetching training and prediction data.
// Implemented by database.RecommendationDataProvider.
type DataProvider interface {
    GetInteractions(ctx context.Context, since time.Time) ([]Interaction, error)
    GetItems(ctx context.Context) ([]Item, error)
    GetUserHistory(ctx context.Context, userID int) ([]int, error)
    GetCandidates(ctx context.Context, userID int, limit int) ([]int, error)
}

// Engine coordinates multiple algorithms and produces final recommendations.
// See internal/recommend/engine.go for the full implementation.
type Engine struct {
    config       *Config
    algorithms   []Algorithm
    rerankers    []Reranker
    dataProvider DataProvider
    // ... additional fields for caching, metrics, and concurrency control
}

// Recommend generates recommendations for a user.
func (e *Engine) Recommend(ctx context.Context, req Request) (*Response, error) {
    // 1. Check cache for existing response
    // 2. Get candidate items (excluding user's history)
    // 3. Score candidates with each algorithm in parallel
    // 4. Fuse scores using weighted combination
    // 5. Apply rerankers (MMR, calibration)
    // 6. Return top-K with metadata
}
```

### Database Schema Additions

```sql
-- Model artifact storage (extends existing model_artifacts table pattern)
CREATE TABLE IF NOT EXISTS recommendation_models (
    name TEXT NOT NULL,              -- Algorithm name
    version INTEGER NOT NULL,        -- Model version
    data BLOB NOT NULL,              -- Serialized model (gob encoded)
    item_count INTEGER,              -- Number of items in model
    user_count INTEGER,              -- Number of users in model
    training_time_ms INTEGER,        -- Training duration
    metrics JSON,                    -- Evaluation metrics
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (name, version)
);

-- Library catalog for cold start (optional enhancement)
CREATE TABLE IF NOT EXISTS library_catalog (
    rating_key TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    media_type TEXT NOT NULL,
    year INTEGER,
    genres TEXT,                     -- Comma-separated
    directors TEXT,                  -- Comma-separated
    actors TEXT,                     -- Comma-separated
    studio TEXT,
    content_rating TEXT,
    guid TEXT,                       -- IMDB/TMDB/TVDB
    added_at TIMESTAMPTZ,
    last_synced_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_genres ON library_catalog(genres);
CREATE INDEX idx_catalog_year ON library_catalog(year);
```

### Configuration

The configuration is defined in `internal/recommend/config.go`. Key sections include:

```go
type Config struct {
    // Algorithm weights (normalized at runtime, don't need to sum to 1.0)
    Weights AlgorithmWeights `json:"weights"`

    // Per-algorithm configuration
    EASE           EASEConfig           `json:"ease"`
    ALS            ALSConfig            `json:"als"`
    ContentBased   ContentBasedConfig   `json:"content_based"`
    CoVisit        CoVisitConfig        `json:"covisit"`
    BPR            BPRConfigMain        `json:"bpr"`
    TimeAwareCF    TimeAwareCFConfigMain `json:"time_aware_cf"`
    MultiHopItemCF MultiHopItemCFConfigMain `json:"multihop_itemcf"`
    MarkovChain    MarkovChainConfigMain `json:"markov_chain"`

    // Diversity parameters
    Diversity DiversityConfig `json:"diversity"`

    // Training schedule
    Training TrainingConfig `json:"training"`

    // Operational limits
    Limits LimitsConfig `json:"limits"`

    // Caching
    Cache CacheConfig `json:"cache"`

    // Seed for deterministic behavior (default: 42)
    Seed int64 `json:"seed"`
}

// AlgorithmWeights defines relative contribution of each algorithm.
type AlgorithmWeights struct {
    EASE           float64 `json:"ease"`        // default: 0.15
    ALS            float64 `json:"als"`         // default: 0.15
    UserCF         float64 `json:"user_cf"`     // default: 0.10
    ItemCF         float64 `json:"item_cf"`     // default: 0.10
    Content        float64 `json:"content"`     // default: 0.15
    CoVisit        float64 `json:"covisit"`     // default: 0.10
    SASRec         float64 `json:"sasrec"`      // default: 0.10
    FPMC           float64 `json:"fpmc"`        // default: 0.05
    Popularity     float64 `json:"popularity"`  // default: 0.05
    Recency        float64 `json:"recency"`     // default: 0.05
    BPR            float64 `json:"bpr"`
    TimeAwareCF    float64 `json:"time_aware_cf"`
    MultiHopItemCF float64 `json:"multihop_itemcf"`
    MarkovChain    float64 `json:"markov_chain"`
}
```

See `internal/recommend/config.go` for complete configuration options including algorithm-specific parameters.

### Usage Example

```go
// In handlers_recommend.go - creating the handler with engine
func NewRecommendHandler(db *database.DB) (*RecommendHandler, error) {
    cfg := recommend.DefaultConfig()
    logger := logging.Logger()

    engine, err := recommend.NewEngine(cfg, logger)
    if err != nil {
        return nil, err
    }

    // Set up data provider for training and prediction
    dataProvider := database.NewRecommendationDataProvider(db)
    engine.SetDataProvider(dataProvider)

    return &RecommendHandler{
        engine:       engine,
        dataProvider: dataProvider,
        db:           db,
    }, nil
}

// Register algorithms (typically done in engine initialization)
engine.RegisterAlgorithm(algorithms.NewEASE(cfg.EASE))
engine.RegisterAlgorithm(algorithms.NewALS(cfg.ALS))
engine.RegisterAlgorithm(algorithms.NewUserBasedCF(algorithms.DefaultKNNConfig()))
engine.RegisterAlgorithm(algorithms.NewItemBasedCF(algorithms.DefaultKNNConfig()))
engine.RegisterAlgorithm(algorithms.NewContentBased(cfg.ContentBased))
engine.RegisterAlgorithm(algorithms.NewCoVisitation(cfg.CoVisit))

// Register rerankers
engine.RegisterReranker(reranking.NewMMR(cfg.Diversity.MMRLambda))
engine.RegisterReranker(reranking.NewCalibration(reranking.DefaultCalibrationConfig()))

// Add to supervisor (in supervisor setup)
tree.AddService(services.NewRecommendService(engine, serviceCfg, logger))

// API handler usage
func (h *RecommendHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
    userID, _ := strconv.Atoi(chi.URLParam(r, "userID"))

    req := recommend.Request{
        UserID:    userID,
        K:         20,
        Mode:      recommend.ModePersonalized,
        RequestID: r.Header.Get("X-Request-ID"),
    }

    resp, err := h.engine.Recommend(r.Context(), req)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "RECOMMENDATION_ERROR", "Failed", err)
        return
    }

    respondJSON(w, http.StatusOK, resp)
}
```

---

## Verification

### Data Availability Verified

| Claim | Source | Verified |
|-------|--------|----------|
| PlaybackEvent has 280+ fields | `internal/models/playback.go` | Yes |
| Genres, actors, directors columns exist | `internal/models/playback.go` (lines 219-222) | Yes |
| percent_complete available | `internal/models/playback.go` (line 125) | Yes |
| media_index for episode ordering | `internal/models/playback.go` (line 209) | Yes |
| User mapping for multi-server | `internal/models/user_mapping.go` | Yes |
| Binge detection exists | `internal/database/analytics_binge.go` | Yes |
| User clustering exists | `internal/database/analytics_user_network.go` | Yes |
| Content discovery analytics | `internal/database/analytics_content_discovery.go` | Yes |
| RapidFuzz available | `internal/database/search_fuzzy.go` | Yes |
| DataSketches available | `internal/database/analytics_approximate.go` | Yes |

### Existing Patterns Verified

| Pattern | Source | Verified |
|---------|--------|----------|
| CTE-based analytics queries | `internal/database/analytics_binge.go` | Yes |
| Window functions for temporal | `internal/database/analytics_binge.go` (LAG, SUM OVER) | Yes |
| Connected components algorithm | `internal/database/analytics_user_network.go` (detectClusters) | Yes |
| Table-driven tests | `internal/database/analytics_binge_test.go` | Yes |
| Context timeout pattern | Used throughout `internal/database/` | Yes |
| Graceful fallback pattern | `internal/database/analytics_approximate.go` | Yes |

### Test Coverage Target

- Algorithm implementations: 90%+
- Engine integration: 85%+
- API handlers: 90%+

---

## Implementation Status

The recommendation engine has been fully implemented with the following components:

### Completed (All Phases)

**Core Infrastructure:**
- Package structure (`internal/recommend/`)
- Core interfaces (Algorithm, Reranker, Engine, DataProvider)
- Configuration system with validation
- Model persistence with gob + gzip compression
- Suture service integration for supervised training

**Algorithms Implemented (13 total):**
1. EASE - Closed-form matrix factorization
2. ALS - Alternating Least Squares
3. User-Based CF - KNN collaborative filtering
4. Item-Based CF - KNN collaborative filtering
5. Content-Based - Genre/actor/director similarity
6. Co-Visitation - Session-based co-occurrence
7. FPMC - Factorized Personalized Markov Chains
8. Markov Chain - Simple sequential patterns
9. BPR - Bayesian Personalized Ranking
10. Time-Aware CF - Temporal decay weighting
11. Multi-Hop ItemCF - Graph-based similarity propagation
12. LinUCB - Contextual bandits
13. Popularity - Baseline trending items

**Reranking:**
- MMR (Maximal Marginal Relevance)
- Calibration (genre distribution matching)

**API Endpoints:**
- All 11 endpoints implemented
- Continue watching functionality
- Sequential "what's next" predictions

**Database Integration:**
- RecommendationDataProvider for training data
- Continue watching queries
- Candidate generation

---

## Related ADRs

- [ADR-0001](0001-use-duckdb-for-analytics.md): DuckDB for analytics (query infrastructure)
- [ADR-0004](0004-process-supervision-with-suture.md): Suture supervision for training service
- [ADR-0016](0016-chi-router-adoption.md): Chi router for API endpoints
- [ADR-0018](0018-duckdb-community-extensions.md): RapidFuzz for content similarity
- [ADR-0020](0020-detection-rules-engine.md): Similar pattern for pluggable engine

---

## References

### Academic Papers

- Steck, H. (2019). "Embarrassingly Shallow Autoencoders for Sparse Data." WWW 2019. [EASE]
- Hu, Y., Koren, Y., & Volinsky, C. (2008). "Collaborative Filtering for Implicit Feedback Datasets." ICDM 2008. [ALS]
- Rendle, S., Freudenthaler, C., Gantner, Z., & Schmidt-Thieme, L. (2009). "BPR: Bayesian Personalized Ranking from Implicit Feedback." UAI 2009. [BPR]
- Rendle, S., Freudenthaler, C., & Schmidt-Thieme, L. (2010). "Factorizing Personalized Markov Chains for Next-Basket Recommendation." WWW 2010. [FPMC]
- Carbonell, J., & Goldstein, J. (1998). "The Use of MMR, Diversity-Based Reranking for Reordering Documents and Producing Summaries." SIGIR 1998. [MMR]
- Steck, H. (2018). "Calibrated Recommendations." RecSys 2018. [Calibration]
- Li, L., et al. (2010). "A Contextual-Bandit Approach to Personalized News Article Recommendation." WWW 2010. [LinUCB]

### Internal Documentation

- [docs/archive/working/RECOMMENDATION_FEASIBILITY_ANALYSIS.md](../archive/working/RECOMMENDATION_FEASIBILITY_ANALYSIS.md): Detailed feasibility analysis (archived)
