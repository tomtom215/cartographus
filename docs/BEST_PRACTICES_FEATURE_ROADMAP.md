# Best Practices Feature Roadmap

**Created**: 2026-01-05
**Last Updated**: 2026-01-07 (Unraid Template Update complete)
**Last Verified**: 2026-01-11
**Status**: COMPLETE - All Features Implemented
**Purpose**: Production-grade features that add genuine competitive value
**Standards**: Testable, Observable, Traceable, Auditable, Secure, Mathematically Provable

> **Note**: This roadmap has been fully implemented. All 9 major features listed below are
> now complete and in production. This document is retained for historical reference and
> to document the implementation decisions made.

---

## Implementation Status

| Feature | Status | Completion Date | Notes |
|---------|--------|-----------------|-------|
| Personal Access Token System | **COMPLETED** | 2026-01-05 | Full implementation with 92%+ coverage |
| Annual Wrapped Reports | **COMPLETED** | 2026-01-06 | Backend + frontend, 20+ tests, RBAC, share tokens |
| Newsletter Generator | **COMPLETED** | 2026-01-06 | Full stack: scheduler, delivery, templates, ContentStore, UI |
| User Onboarding | **COMPLETED** | 2026-01-06 | Setup wizard, welcome tour, progressive tips (20+), WCAG 2.1 AA |
| Enhanced Recommendations | **COMPLETED** | 2026-01-06 | 13 algorithms, What's Next widget, tooltips, full API |
| Library Access Management | **COMPLETED** | 2026-01-06 | Plex friends/sharing/managed users, 12 handlers, admin RBAC, frontend UI |
| Advanced Charts | **COMPLETED** | 2026-01-06 | Sankey, Chord, Radar, Treemap, Calendar Heatmap, Bump Chart with RBAC |
| Grafana Dashboards | **COMPLETED** | 2026-01-07 | 6 dashboards (Overview, Performance, Streaming, Detection, Database, Auth), Docker Compose, Prometheus config |
| Unraid Template Update | **COMPLETED** | 2026-01-07 | 100+ config options: OIDC, Plex Auth, Casbin, Newsletter, Detection, NATS, Recommendations, VPN, GeoIP |

---

## Table of Contents

1. [Annual Wrapped Reports (Spotify-Style)](#1-annual-wrapped-reports-spotify-style)
2. [Enhanced Recommendation Engine Algorithms](#2-enhanced-recommendation-engine-algorithms)
3. [Advanced Charts and Visualizations](#3-advanced-charts-and-visualizations)
4. [Newsletter and Digest Generator](#4-newsletter-and-digest-generator)
5. [Complete User Onboarding Experience](#5-complete-user-onboarding-experience)
6. [Production Grafana Dashboard](#6-production-grafana-dashboard)
7. [Personal Access Token System](#7-personal-access-token-system)
8. [Updated Unraid Template](#8-updated-unraid-template)
9. [Library Access Management](#9-library-access-management)
10. [Additional High-Value Features](#10-additional-high-value-features)

---

## 1. Annual Wrapped Reports (Spotify-Style)

**Priority**: HIGH
**Effort**: 5-7 days
**Competitive Advantage**: Unique in media server analytics space

### Overview

Generate personalized annual viewing reports for each user with engaging statistics, insights, and shareable summaries - similar to Spotify Wrapped but for media consumption.

### Features

#### 1.1 Per-User Annual Statistics

| Metric | Description | SQL Aggregation |
|--------|-------------|-----------------|
| Total Watch Time | Hours/days of content consumed | `SUM(play_duration) / 3600` |
| Content Count | Movies, episodes, tracks | `COUNT(DISTINCT rating_key)` |
| Completion Rate | % of started content finished | `AVG(CASE WHEN percent_complete >= 90 THEN 1 ELSE 0 END)` |
| Top Genres | Most-watched genre breakdown | `COUNT(*) GROUP BY genres` |
| Peak Viewing Hours | Most active times | `EXTRACT(HOUR FROM started_at)` |
| Binge Sessions | Marathon viewing events | Existing `GetBingeAnalytics()` |
| Quality Preference | Average resolution/bitrate | `AVG(stream_bitrate)` |
| Discovery Score | New content exploration rate | First-time views / total views |

#### 1.2 Personalized Insights

```
- "You watched 847 hours of content - that's equivalent to 35 days!"
- "Your favorite show was Breaking Bad with 12 rewatches"
- "You're in the top 5% of active users"
- "80% of your viewing was after 8 PM - true night owl!"
- "You discovered 45 new movies this year"
- "Your binge record: 11 episodes of The Office in one sitting"
```

#### 1.3 Comparative Rankings

- **Library Champion**: User who explored most diverse content
- **Binge Master**: Longest marathon sessions
- **Quality Enthusiast**: Highest average bitrate
- **Early Bird**: First to watch new additions
- **Night Owl**: Most late-night viewing

### Implementation

```go
// internal/database/analytics_wrapped.go
type WrappedReport struct {
    Year               int             `json:"year"`
    UserID             string          `json:"user_id"`
    Username           string          `json:"username"`
    GeneratedAt        time.Time       `json:"generated_at"`

    // Core Stats
    TotalWatchTimeHours float64        `json:"total_watch_time_hours"`
    TotalPlaybacks      int            `json:"total_playbacks"`
    UniqueContent       int            `json:"unique_content"`
    CompletionRate      float64        `json:"completion_rate"`

    // Content Breakdown
    TopMovies          []ContentRank   `json:"top_movies"`
    TopShows           []ContentRank   `json:"top_shows"`
    TopGenres          []GenreRank     `json:"top_genres"`
    TopActors          []PersonRank    `json:"top_actors"`
    TopDirectors       []PersonRank    `json:"top_directors"`

    // Viewing Patterns
    ViewingByHour      [24]int         `json:"viewing_by_hour"`
    ViewingByDay       [7]int          `json:"viewing_by_day"`
    ViewingByMonth     [12]int         `json:"viewing_by_month"`
    PeakHour           int             `json:"peak_hour"`
    PeakDay            string          `json:"peak_day"`

    // Binge Analysis
    BingeSessions      int             `json:"binge_sessions"`
    LongestBinge       BingeRecord     `json:"longest_binge"`
    TotalBingeHours    float64         `json:"total_binge_hours"`

    // Quality Metrics
    AvgBitrateMbps     float64         `json:"avg_bitrate_mbps"`
    DirectPlayRate     float64         `json:"direct_play_rate"`
    HDRViewingPercent  float64         `json:"hdr_viewing_percent"`

    // Discovery Metrics
    NewContentCount    int             `json:"new_content_count"`
    DiscoveryRate      float64         `json:"discovery_rate"`
    FirstWatchCount    int             `json:"first_watch_count"`

    // Achievements
    Achievements       []Achievement   `json:"achievements"`
    Percentiles        UserPercentiles `json:"percentiles"`

    // Shareable Summary
    ShareableText      string          `json:"shareable_text"`
    ShareableImage     string          `json:"shareable_image_url"`
}

func (db *DB) GenerateWrappedReport(ctx context.Context, userID string, year int) (*WrappedReport, error)
func (db *DB) GenerateAllWrappedReports(ctx context.Context, year int) ([]WrappedReport, error)
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/wrapped/{year}` | Server-wide wrapped stats |
| GET | `/api/v1/wrapped/{year}/user/{userID}` | Per-user wrapped report |
| GET | `/api/v1/wrapped/{year}/leaderboard` | Top users by various metrics |
| POST | `/api/v1/wrapped/{year}/generate` | Trigger report generation (admin) |
| GET | `/api/v1/wrapped/{year}/share/{userID}` | Shareable image/link |

### Frontend Components

- **WrappedDashboard**: Animated reveal of statistics (card-by-card)
- **WrappedTimeline**: Month-by-month viewing journey
- **WrappedShareCard**: Exportable image for social sharing
- **WrappedComparison**: Year-over-year comparison

### Database Schema

```sql
-- Store generated reports for caching
CREATE TABLE IF NOT EXISTS wrapped_reports (
    id INTEGER PRIMARY KEY,
    user_id TEXT NOT NULL,
    year INTEGER NOT NULL,
    report_json JSON NOT NULL,
    generated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    share_token TEXT UNIQUE,
    UNIQUE(user_id, year)
);

CREATE INDEX idx_wrapped_year ON wrapped_reports(year);
CREATE INDEX idx_wrapped_share ON wrapped_reports(share_token);
```

### Testing Requirements

- Unit tests for each aggregation query (90%+ coverage)
- Integration tests with sample data (multiple users, edge cases)
- E2E tests for frontend reveal animation
- Performance benchmarks (< 5s for 100k events)

### Observability

- Prometheus metrics: `wrapped_report_generation_duration_seconds`
- Audit log: Report generation events with user, year, duration
- Error tracking: Failed generations with query details

---

## 2. Enhanced Recommendation Engine Algorithms

**Priority**: HIGH
**Effort**: 7-10 days (Phase 2 of ADR-0024)
**Competitive Advantage**: No competitor offers ML recommendations
**Status**: **COMPLETED** (2026-01-06)

### Implementation Summary

The recommendation engine is fully implemented with 13 algorithms and complete frontend UI:

#### Backend Components

| Component | File | Status |
|-----------|------|--------|
| EASE Algorithm | `internal/recommend/algorithms/ease.go` | Complete |
| ALS Algorithm | `internal/recommend/algorithms/als.go` | Complete |
| UserCF Algorithm | `internal/recommend/algorithms/usercf.go` | Complete |
| ItemCF Algorithm | `internal/recommend/algorithms/itemcf.go` | Complete |
| Content-Based | `internal/recommend/algorithms/content.go` | Complete |
| Co-Visitation | `internal/recommend/algorithms/covisit.go` | Complete |
| Popularity | `internal/recommend/algorithms/popularity.go` | Complete |
| FPMC | `internal/recommend/algorithms/fpmc.go` | Complete |
| BPR | `internal/recommend/algorithms/bpr.go` | Complete |
| Time-Aware CF | `internal/recommend/algorithms/timecf.go` | Complete |
| Multi-Hop ItemCF | `internal/recommend/algorithms/multihop.go` | Complete |
| Markov Chain | `internal/recommend/algorithms/markov.go` | Complete |
| LinUCB | `internal/recommend/algorithms/linucb.go` | Complete |
| API Handlers | `internal/api/handlers_recommend.go` | Complete |
| Chi Routes | `internal/api/chi_router.go` | Complete |

#### Frontend Components

| Component | File | Status |
|-----------|------|--------|
| RecommendationManager | `web/src/app/RecommendationManager.ts` | Complete |
| TypeScript Types | `web/src/lib/types/recommend.ts` | Complete |
| API Client | `web/src/lib/api/recommend.ts` | Complete |
| CSS Styles | `web/src/styles/features/recommendations.css` | Complete |

#### Key Features Implemented

- **Tab-based UI**: Recommendations, What's Next, Algorithms tabs
- **What's Next Widget**: Markov chain predictions for "watch next"
- **Algorithm Tooltips**: Detailed explanations for all 13 algorithms
- **Algorithm Categories**: Basic, Matrix, Collaborative, Sequential, Advanced, Bandit
- **Algorithm Metrics**: Per-algorithm training status and latency
- **Full CRUD**: All recommendation endpoints with authentication

#### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/recommendations/status` | Training status |
| GET | `/api/v1/recommendations/config` | Engine configuration |
| PUT | `/api/v1/recommendations/config` | Update configuration |
| POST | `/api/v1/recommendations/train` | Trigger training |
| GET | `/api/v1/recommendations/algorithms` | Algorithm info |
| GET | `/api/v1/recommendations/algorithms/metrics` | Per-algorithm metrics |
| GET | `/api/v1/recommendations/user/{userID}` | User recommendations |
| GET | `/api/v1/recommendations/user/{userID}/continue` | Continue watching |
| GET | `/api/v1/recommendations/user/{userID}/explore` | Exploration mode |
| GET | `/api/v1/recommendations/similar/{itemID}` | Similar items |
| GET | `/api/v1/recommendations/next/{itemID}` | What's Next predictions |

### Overview

Extend the existing recommendation engine (ADR-0024) with additional algorithms and complete the implementation phases.

### Resource Constraints for Self-Hosted Environments

> **IMPORTANT**: Cartographus is designed for self-hosters running on consumer hardware.
> All algorithms MUST work within these constraints:

| Constraint | Requirement |
|------------|-------------|
| RAM | 4-8 GB total system memory |
| CPU | Consumer-grade x86-64 (no GPU required) |
| Storage | Standard SSD/HDD |
| Concurrent Load | Low (1-10 concurrent users) |

#### Algorithm Classification

| Algorithm | Memory Usage | CPU Usage | Status | Notes |
|-----------|--------------|-----------|--------|-------|
| EASE | Low (~50MB for 10k items) | O(n^2) but sparse | **RECOMMENDED** | Closed-form, no iterations |
| ItemCF | Low (~20MB) | Low | **RECOMMENDED** | Simple similarity matrix |
| UserCF | Low (~20MB) | Low | **RECOMMENDED** | Simple similarity matrix |
| Content-Based | Low (~10MB) | Low | **RECOMMENDED** | TF-IDF on metadata |
| Co-Visitation | Low (~30MB) | Low | **RECOMMENDED** | Simple co-occurrence counts |
| Time-Aware CF | Low (~20MB) | Low | **RECOMMENDED** | Weighted UserCF/ItemCF |
| BPR | Medium (~100MB) | Medium | **ENABLED** | SGD iterations, controllable |
| NCF | High (500MB+) | High | **DISABLED by default** | Neural network overhead |
| GRU4Rec | High (1GB+) | Very High | **DISABLED by default** | Recurrent neural network |
| LightGCN | Very High (2GB+) | Very High | **DISABLED by default** | Graph convolutions |

#### Resource Guards

```go
// internal/recommend/config.go
type ResourceConfig struct {
    // Maximum users before falling back to simpler algorithms
    MaxUsersForFullModel int `default:"5000"`

    // Maximum items before switching to approximate methods
    MaxItemsForFullModel int `default:"20000"`

    // Maximum memory budget for recommendation engine (MB)
    MaxMemoryMB int `default:"512"`

    // Enable resource-intensive algorithms (NCF, GRU4Rec, LightGCN)
    EnableAdvancedAlgorithms bool `default:"false"`
}
```

#### Configuration

```bash
# Environment variables for resource-constrained deployments

# Simple mode: Only lightweight algorithms (EASE, ItemCF, UserCF, Content-Based)
RECOMMENDATIONS_MODE=simple

# Full mode: All algorithms including neural networks (requires 8GB+ RAM)
RECOMMENDATIONS_MODE=full

# Auto mode: Detect available resources and select appropriate algorithms
RECOMMENDATIONS_MODE=auto
```

### Current State (ADR-0024 Design)

- Framework designed with modular `Algorithm` interface
- File structure defined in `internal/recommend/`
- Algorithms specified: EASE, ALS, UserCF, ItemCF, Content-Based, Co-Visitation, SASRec, FPMC
- Rerankers: MMR, Calibration, LinUCB

### Additional Algorithms to Implement

#### 2.1 BPR (Bayesian Personalized Ranking)

**Use Case**: Implicit feedback optimization
**Mathematical Foundation**: Maximum posterior estimator for pairwise preferences

```go
// internal/recommend/algorithms/bpr.go
type BPRAlgorithm struct {
    Factors     int           // Latent dimensions (default: 64)
    LearningRate float64      // SGD step size (default: 0.01)
    Lambda      float64       // L2 regularization (default: 0.01)
    Epochs      int           // Training iterations (default: 100)
    userFactors [][]float64   // User latent factors
    itemFactors [][]float64   // Item latent factors
}

// Pairwise ranking loss: ln(sigmoid(x_uij))
func (b *BPRAlgorithm) Train(ctx context.Context, interactions []Interaction) error
```

#### 2.2 Neural Collaborative Filtering (NCF)

> **WARNING: Resource-Intensive Algorithm**
> NCF is DISABLED by default. Requires `RECOMMENDATIONS_MODE=full` and 8GB+ RAM.
> For resource-constrained deployments, use BPR or EASE instead.

**Use Case**: Non-linear user-item interactions
**Implementation**: Pure Go neural network (no external dependencies)
**Memory**: ~500MB-1GB for medium libraries (10k users, 50k items)
**CPU**: High during training, medium during inference

```go
// internal/recommend/algorithms/ncf.go
type NCFAlgorithm struct {
    EmbedDim    int           // Embedding dimension (default: 32 for constrained, 64 for full)
    HiddenLayers []int        // MLP layer sizes [32, 16] for constrained, [64, 32, 16] for full
    DropoutRate float64       // Regularization
    MaxBatchSize int          // Limit batch size for memory control (default: 256)
}

// ResourceCheck validates system has sufficient resources
func (n *NCFAlgorithm) ResourceCheck() error {
    if !config.EnableAdvancedAlgorithms {
        return ErrAdvancedAlgorithmDisabled
    }
    // Check available memory before training
    return nil
}
```

**Simpler Alternative**: For most self-hosted use cases, BPR with matrix factorization provides 90% of NCF's quality with 10% of the resources.

#### 2.3 Session-Based Recommendations (GRU4Rec-inspired)

> **WARNING: Resource-Intensive Algorithm**
> GRU4Rec is DISABLED by default. Requires `RECOMMENDATIONS_MODE=full` and 8GB+ RAM.
> For resource-constrained deployments, use Markov Chain or Co-Visitation instead.

**Use Case**: "What to watch next" based on current session
**Implementation**: Simplified recurrent model in pure Go
**Memory**: ~1GB+ for training on 100k sessions
**CPU**: Very High during training (RNN backpropagation)

```go
// internal/recommend/algorithms/session.go
type SessionRecAlgorithm struct {
    HiddenSize    int           // Default: 64 for constrained, 128 for full
    SessionWindow time.Duration
    MaxSequenceLen int          // Limit sequence length (default: 20)
}

func (s *SessionRecAlgorithm) ResourceCheck() error {
    if !config.EnableAdvancedAlgorithms {
        return ErrAdvancedAlgorithmDisabled
    }
    return nil
}

func (s *SessionRecAlgorithm) PredictNext(ctx context.Context, sessionItems []int) ([]ScoredItem, error)
```

**Simpler Alternative - Markov Chain**: For "what to watch next", a simple first-order Markov chain (item-to-item transition probabilities) is highly effective with minimal resources:

```go
// internal/recommend/algorithms/markov.go
type MarkovChainAlgorithm struct {
    TransitionMatrix map[int]map[int]float64  // item -> next_item -> probability
}

// Memory: ~10MB for 50k items, CPU: O(1) inference
func (m *MarkovChainAlgorithm) PredictNext(ctx context.Context, lastItem int) ([]ScoredItem, error)
```

#### 2.4 Graph-Based Recommendations (LightGCN)

> **WARNING: Resource-Intensive Algorithm**
> LightGCN is DISABLED by default. Requires `RECOMMENDATIONS_MODE=full` and 16GB+ RAM.
> For resource-constrained deployments, use ItemCF with multi-hop expansion instead.

**Use Case**: Multi-hop user-item relationships
**Implementation**: Graph convolution without feature transformation
**Memory**: ~2GB+ for graph with 50k nodes
**CPU**: Very High (sparse matrix operations per layer)

```go
// internal/recommend/algorithms/lightgcn.go
type LightGCNAlgorithm struct {
    Layers      int           // Number of propagation layers (default: 2 for constrained, 3 for full)
    EmbedDim    int           // Embedding dimension (default: 32 for constrained, 64 for full)
    MaxNodes    int           // Maximum graph nodes before sampling (default: 10000)
}

func (l *LightGCNAlgorithm) ResourceCheck() error {
    if !config.EnableAdvancedAlgorithms {
        return ErrAdvancedAlgorithmDisabled
    }
    return nil
}
```

**Simpler Alternative - Multi-Hop ItemCF**: Approximate graph propagation by expanding ItemCF similarity through 2-3 hops:

```go
// internal/recommend/algorithms/multihop_itemcf.go
type MultiHopItemCFAlgorithm struct {
    Hops           int     // Number of expansion hops (default: 2)
    SimilarityTopK int     // Top-K similar items per hop (default: 10)
    DecayFactor    float64 // Score decay per hop (default: 0.5)
}

// Memory: ~30MB, CPU: O(k^hops) but typically small k
func (m *MultiHopItemCFAlgorithm) Recommend(ctx context.Context, userItems []int) ([]ScoredItem, error)
```

#### 2.5 Time-Aware Collaborative Filtering

**Use Case**: Account for temporal dynamics (recent preferences matter more)
**Implementation**: Time decay weighting

```go
// internal/recommend/algorithms/timecf.go
type TimeAwareCFAlgorithm struct {
    DecayRate   float64       // Exponential decay (default: 0.1)
    TimeWindow  time.Duration // Maximum lookback (default: 365 days)
}
```

### Recommendation Quality Metrics

| Metric | Description | Target |
|--------|-------------|--------|
| Precision@K | Relevant items in top-K | > 0.3 |
| Recall@K | Coverage of relevant items | > 0.2 |
| NDCG@K | Ranking quality | > 0.4 |
| Coverage | Unique items recommended | > 0.5 |
| Novelty | Discovery of long-tail content | > 0.3 |

### A/B Testing Framework

```go
// internal/recommend/experiment.go
type Experiment struct {
    ID          string
    Name        string
    Treatment   AlgorithmConfig
    Control     AlgorithmConfig
    TrafficSplit float64
    Metrics     []MetricCollector
}

func (e *Engine) RunExperiment(ctx context.Context, exp Experiment, userID string) (*RecommendResponse, error)
```

### Testing Requirements

- Unit tests for each algorithm (deterministic outputs with seeded random)
- Offline evaluation on historical data (train/test split)
- Integration tests with real database
- Performance benchmarks (inference < 50ms)

### Recommended Configurations by Hardware Tier

| Hardware Tier | RAM | Recommended Mode | Algorithms Enabled |
|---------------|-----|------------------|-------------------|
| Minimal | 4GB | `simple` | EASE, ItemCF, UserCF, Content-Based |
| Standard | 8GB | `simple` | + BPR, Co-Visitation, Time-Aware CF, Markov Chain |
| Performance | 16GB+ | `full` | + NCF, GRU4Rec (optional) |
| Enterprise | 32GB+ | `full` | All algorithms including LightGCN |

#### Typical Self-Hoster Configuration

```bash
# Recommended for most Unraid/Docker deployments (4-8GB RAM)
RECOMMENDATIONS_MODE=simple
RECOMMENDATIONS_MAX_USERS=5000
RECOMMENDATIONS_MAX_ITEMS=20000
RECOMMENDATIONS_MEMORY_MB=256
```

#### Why This Matters

Most self-hosters run Cartographus alongside other services (Plex, Sonarr, Radarr, etc.) on the same machine. The recommendation engine should:

1. **Never cause OOM**: Stay within memory budget even with large libraries
2. **Not starve other services**: Limit CPU usage during model training
3. **Degrade gracefully**: Fall back to simpler algorithms when resources are tight
4. **Provide value regardless**: Even simple algorithms (ItemCF, EASE) provide excellent recommendations

---

## 3. Advanced Charts and Visualizations

**Priority**: MEDIUM-HIGH
**Effort**: 5-7 days
**Competitive Advantage**: WebGL-powered visualizations unique to Cartographus
**Status**: **COMPLETED** (2026-01-06)

### Implementation Summary

The Advanced Charts feature is fully implemented with 6 new chart types and comprehensive RBAC enforcement:

#### Backend Components

| Component | File | Status |
|-----------|------|--------|
| Content Flow (Sankey) | `internal/database/analytics_advanced_charts.go` | Complete |
| User Overlap (Chord) | `internal/database/analytics_advanced_charts.go` | Complete |
| User Profile (Radar) | `internal/database/analytics_advanced_charts.go` | Complete |
| Library Utilization (Treemap) | `internal/database/analytics_advanced_charts.go` | Complete |
| Calendar Heatmap | `internal/database/analytics_advanced_charts.go` | Complete |
| Bump Chart | `internal/database/analytics_advanced_charts.go` | Complete |
| API Handlers | `internal/api/handlers_analytics_charts.go` | Complete |
| Analytics Executor | `internal/api/analytics_executor.go` | Complete |
| Handler Tests | `internal/api/handlers_analytics_charts_test.go` | Complete |

#### RBAC Enforcement

| Endpoint | Access Level | Description |
|----------|--------------|-------------|
| `/analytics/content-flow` | User-scoped | Shows only user's own viewing journeys |
| `/analytics/user-overlap` | Admin-only | Cross-user Jaccard similarity (admin sees all) |
| `/analytics/user-profile` | User-scoped | Multi-dimensional engagement scores |
| `/analytics/library-utilization` | User-scoped | Hierarchical usage filtered by user |
| `/analytics/calendar-heatmap` | User-scoped | Daily activity (user's own data) |
| `/analytics/bump-chart` | Admin-only | Global content ranking insights |

#### Frontend Components

| Component | File | Status |
|-----------|------|--------|
| TypeScript Types | `web/src/lib/types/analytics.ts` | Complete |
| API Client | `web/src/lib/api/analytics.ts` | Complete |

#### Key Features Implemented

- **ExecuteUserScoped()**: Auto-filters queries to user's own data for non-admins
- **ExecuteAdminOnly()**: Returns 401/403 for non-admin users
- **Cache key with UserScope**: Per-user caching for RBAC-aware responses
- **34+ tests**: Handler tests + RBAC enforcement tests
- **Deterministic**: Query hash in metadata for cache validation
- **Observable**: Full metadata with execution time and statistics

### Current State

- 47+ charts with 20 specialized ECharts renderers
- deck.gl 3D globe with WebGL
- MapLibre GL JS for 2D maps

### Additional Chart Types

#### 3.1 Sankey Diagram - Content Flow

**Use Case**: Visualize viewing journeys (TV Show -> Season -> Episodes)

```typescript
// web/src/lib/charts/renderers/SankeyChartRenderer.ts
interface SankeyData {
    nodes: { name: string; depth: number }[];
    links: { source: string; target: string; value: number }[];
}

// Example: Show -> Season -> Episode flow with drop-off at each stage
```

#### 3.2 Chord Diagram - User Content Overlap

**Use Case**: Show which users share similar viewing tastes

```typescript
// web/src/lib/charts/renderers/ChordChartRenderer.ts
// Matrix of user-user content overlap
```

#### 3.3 Radar Chart - User Profile Comparison

**Use Case**: Compare user engagement dimensions

```typescript
// web/src/lib/charts/renderers/RadarChartRenderer.ts
interface RadarProfile {
    axes: ['Watch Time', 'Completion', 'Diversity', 'Quality', 'Discovery', 'Social'];
    values: number[];  // 0-100 normalized scores
}
```

#### 3.4 Treemap - Library Utilization

**Use Case**: Hierarchical view of library usage

```typescript
// Library -> Section -> Content with size = watch time
```

#### 3.5 Calendar Heatmap - Yearly Activity

**Use Case**: GitHub-style contribution graph for viewing

```typescript
// web/src/lib/charts/renderers/CalendarHeatmapRenderer.ts
// 365-day grid with intensity based on watch time
```

#### 3.6 Parallel Coordinates - Multi-Dimensional Analysis

**Use Case**: Compare multiple metrics across users/content

```typescript
// Axes: Watch Time, Completion, Quality, Platform, Genre
// Lines: Each user/content item
```

#### 3.7 Network Graph - Content Similarity

**Use Case**: Visualize content relationships based on co-viewing

```typescript
// web/src/lib/charts/renderers/NetworkGraphRenderer.ts
// Force-directed graph of content nodes connected by shared viewers
```

#### 3.8 Spiral Timeline - Historical Trends

**Use Case**: Long-term viewing patterns in compact form

```typescript
// Archimedean spiral with radial position = time, color = activity
```

#### 3.9 Bump Chart - Ranking Changes Over Time

**Use Case**: Track content popularity ranking changes

```typescript
// Show how top 10 content ranking evolves week-by-week
```

#### 3.10 Stream Graph - Stacked Time Series

**Use Case**: Genre viewing over time with smooth transitions

```typescript
// Stacked area chart centered on baseline for aesthetic appeal
```

### Testing Requirements

- Visual regression tests with Percy/Chromatic
- Unit tests for data transformation
- E2E tests for interactivity (hover, click, zoom)
- Accessibility tests (keyboard navigation, screen reader)

---

## 4. Newsletter and Digest Generator

**Priority**: HIGH
**Effort**: 7-10 days
**Competitive Advantage**: Tautulli has this - achieving parity
**Status**: **COMPLETED** (2026-01-06) - Full Stack Implementation

### Implementation Summary

The Newsletter Generator is fully implemented with production-grade quality:

#### Backend Components

| Component | File | Status |
|-----------|------|--------|
| Cron Parser | `internal/newsletter/scheduler/cron.go` | Complete |
| Scheduler Service | `internal/newsletter/scheduler/scheduler.go` | Complete |
| Template Engine | `internal/newsletter/template_engine.go` | Complete |
| Content Resolver | `internal/newsletter/content_resolver.go` | Complete |
| ContentStore Interface | `internal/database/newsletter_content.go` | Complete |
| Delivery Manager | `internal/newsletter/delivery/manager.go` | Complete |
| Email Channel | `internal/newsletter/delivery/email.go` | Complete |
| Discord Channel | `internal/newsletter/delivery/discord.go` | Complete |
| Slack Channel | `internal/newsletter/delivery/slack.go` | Complete |
| Telegram Channel | `internal/newsletter/delivery/telegram.go` | Complete |
| Webhook Channel | `internal/newsletter/delivery/webhook.go` | Complete |
| In-App Channel | `internal/newsletter/delivery/inapp.go` | Complete |
| Config Integration | `internal/config/koanf.go` | Complete |
| Supervisor Integration | `cmd/server/newsletter_init.go` | Complete |
| Database Schema | `internal/database/database_schema.go` | Complete |
| API Handlers | `internal/api/handlers_newsletter.go` | Complete |

#### Frontend Components

| Component | File | Status |
|-----------|------|--------|
| Newsletter Manager | `web/src/app/NewsletterManager.ts` | Complete |
| Overview Renderer | `web/src/app/newsletter/OverviewRenderer.ts` | Complete |
| Templates Renderer | `web/src/app/newsletter/TemplatesRenderer.ts` | Complete |
| Schedules Renderer | `web/src/app/newsletter/SchedulesRenderer.ts` | Complete |
| Deliveries Renderer | `web/src/app/newsletter/DeliveriesRenderer.ts` | Complete |
| Base Renderer | `web/src/app/newsletter/BaseNewsletterRenderer.ts` | Complete |
| TypeScript Types | `web/src/lib/types/newsletter.ts` | Complete |
| API Client | `web/src/lib/api/newsletter.ts` | Complete |

**Key Features Implemented**:
- Cron-based scheduling with standard 5-field expressions + presets
- 6 delivery channels: Email, Discord, Slack, Telegram, Webhook, In-App
- ContentStore with 9 database methods for content resolution
- Template engine with HTML rendering and variable substitution
- Live template preview with iframe rendering
- Schedule management with enable/disable toggles
- Delivery history with pagination and detailed views
- Channel distribution statistics
- Full CRUD operations for templates and schedules
- Supervisor tree integration for lifecycle management
- Environment variable configuration (NEWSLETTER_*)

**ContentStore Methods**:
- `GetRecentlyAddedMovies` - Movies added since a given time
- `GetRecentlyAddedShows` - Shows with new episodes aggregated by show
- `GetRecentlyAddedMusic` - Recently added music tracks
- `GetTopMovies` - Most watched movies by play count
- `GetTopShows` - Most watched shows by episode count
- `GetPeriodStats` - Aggregate statistics for time periods
- `GetUserStats` - Per-user statistics for personalized newsletters
- `GetUserRecommendations` - Collaborative filtering recommendations
- `GetServerHealth` - Server health metrics for health newsletters

**Test Coverage**: 60+ tests across scheduler, delivery, ContentStore, and API modules

### Overview

Template-based newsletter system for periodic digests, similar to Tautulli's newsletter feature.

### Features

#### 4.1 Template Engine

- HTML template support with variable substitution
- Pre-built templates: Weekly Digest, New Content, User Activity
- Custom template editor with preview
- Template versioning and rollback

#### 4.2 Content Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.NewMovies}}` | Recently added movies | List with posters |
| `{{.NewShows}}` | Recently added TV shows | List with posters |
| `{{.TopContent}}` | Most-watched this period | Ranked list |
| `{{.UserStats}}` | Per-user statistics | Watch time, completions |
| `{{.ServerStats}}` | Server-wide metrics | Total streams, bandwidth |
| `{{.Recommendations}}` | Personalized suggestions | From recommendation engine |

#### 4.3 Delivery Channels

| Channel | Implementation | Configuration |
|---------|----------------|---------------|
| Email | SMTP integration | `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER` |
| Discord | Webhook | `DISCORD_WEBHOOK_URL` |
| Slack | Webhook | `SLACK_WEBHOOK_URL` |
| Telegram | Bot API | `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID` |
| Generic Webhook | HTTP POST | Custom URL with payload |

#### 4.4 Scheduling

```go
// internal/newsletter/scheduler.go
type Schedule struct {
    ID          string
    Name        string
    TemplateID  string
    Recipients  []Recipient
    Cron        string           // "0 9 * * 1" = Monday 9am
    Timezone    string
    LastRun     time.Time
    NextRun     time.Time
    Enabled     bool
}

type Recipient struct {
    Type        string           // "user", "email", "webhook"
    Target      string           // user_id, email@domain.com, webhook_url
    Preferences RecipientPrefs   // Opt-out, frequency preferences
}
```

### Implementation

```go
// internal/newsletter/
// ├── template.go         # Template parsing and rendering
// ├── variables.go        # Variable resolution from database
// ├── scheduler.go        # Cron-based scheduling
// ├── delivery/
// │   ├── email.go        # SMTP delivery
// │   ├── discord.go      # Discord webhook
// │   ├── slack.go        # Slack webhook
// │   ├── telegram.go     # Telegram bot
// │   └── webhook.go      # Generic HTTP
// └── templates/
//     ├── weekly_digest.html
//     ├── new_content.html
//     └── user_activity.html
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/newsletter/templates` | List templates |
| POST | `/api/v1/newsletter/templates` | Create template |
| PUT | `/api/v1/newsletter/templates/{id}` | Update template |
| DELETE | `/api/v1/newsletter/templates/{id}` | Delete template |
| POST | `/api/v1/newsletter/preview` | Preview with sample data |
| GET | `/api/v1/newsletter/schedules` | List schedules |
| POST | `/api/v1/newsletter/schedules` | Create schedule |
| PUT | `/api/v1/newsletter/schedules/{id}` | Update schedule |
| POST | `/api/v1/newsletter/send` | Send immediately |
| GET | `/api/v1/newsletter/history` | Delivery history |

### Database Schema

```sql
CREATE TABLE IF NOT EXISTS newsletter_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    subject TEXT,
    body_html TEXT NOT NULL,
    body_text TEXT,
    variables JSON,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS newsletter_schedules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    template_id TEXT NOT NULL REFERENCES newsletter_templates(id),
    recipients JSON NOT NULL,
    cron_expression TEXT NOT NULL,
    timezone TEXT DEFAULT 'UTC',
    enabled BOOLEAN DEFAULT true,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS newsletter_history (
    id INTEGER PRIMARY KEY,
    schedule_id TEXT REFERENCES newsletter_schedules(id),
    template_id TEXT NOT NULL,
    recipients_count INTEGER,
    delivered_count INTEGER,
    failed_count INTEGER,
    sent_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    errors JSON
);
```

### Testing Requirements

- Unit tests for template rendering (variable substitution)
- Integration tests for each delivery channel (mock servers)
- E2E tests for scheduling (time manipulation)
- Visual tests for rendered HTML

---

## 5. Complete User Onboarding Experience

**Priority**: MEDIUM-HIGH
**Effort**: 3-5 days
**Competitive Advantage**: Reduces churn, improves user adoption
**Status**: **COMPLETED** (2026-01-06)

### Implementation Summary

The User Onboarding Experience is fully implemented with three complementary components:

#### Components

| Component | File | Purpose |
|-----------|------|---------|
| Setup Wizard | `web/src/app/SetupWizardManager.ts` | Multi-step configuration wizard (5 steps) |
| Welcome Tour | `web/src/app/OnboardingManager.ts` | Feature introduction with guided tour |
| Progressive Tips | `web/src/app/ProgressiveOnboardingManager.ts` | Contextual feature discovery (20+ tips) |

#### Setup Wizard (5 Steps)
1. **Welcome** - Introduction to Cartographus
2. **Database** - Verify DuckDB connection status
3. **Data Sources** - Check Tautulli, Plex, Jellyfin, Emby connectivity
4. **Data** - Show playback event counts and sync options
5. **Complete** - Ready state with tour/explore options

#### Welcome Tour (8 Steps - Comprehensive Introduction)
1. **Interactive Map** - 2D, 3D globe, heatmap, hexagon visualizations
2. **Dashboard Navigation** - All 8 main sections introduced
3. **Filter Panel** - Date range, users, platforms, media types with presets
4. **Analytics Dashboard** - 47 charts across 10 analytics pages
5. **Export Options** - CSV, GeoJSON, GeoParquet formats
6. **Server Management** - Multi-server support (Tautulli/Plex/Jellyfin/Emby)
7. **Data Governance** - Backups, GDPR, audit logs, data quality
8. **Theme and Shortcuts** - Visual themes and keyboard navigation

#### Progressive Tips (55 Contextual Tips - Comprehensive Coverage)
- **Map tips (5)**: Mode switch, 2D/3D views, arc overlay, hexagon resolution, fullscreen
- **Navigation tabs (8)**: Maps, Live Activity, Analytics, Recently Added, Server, Cross-Platform, Data Governance, Newsletter
- **Analytics sub-tabs (10)**: Overview, Content, Users, Performance, Geographic, Advanced, Library, User Profiles, Tautulli, Wrapped
- **Chart tips (3)**: Export, maximize, quick insights
- **Filter tips (4)**: Date range, presets, save/clear
- **Export tips (3)**: CSV, GeoJSON, GeoParquet
- **Search tips (2)**: Global search, advanced search
- **Data Governance (5)**: Backup, retention, audit logs, GDPR, dedupe
- **Security (3)**: Detection rules, alerts, API tokens
- **Server Management (4)**: Sync status, manual sync, config, health
- **Newsletter (4)**: Templates, schedules, preview, channels
- **Recommendations (2)**: Panel, settings
- **Settings (4)**: Theme, refresh, shortcuts, notifications
- **Sidebar/UI (2)**: Collapse, breadcrumbs

#### Quality Features
- WCAG 2.1 AA compliant (contrast ratios, focus indicators)
- Keyboard navigation (Arrow keys, Escape, Enter)
- Reduced motion support (`prefers-reduced-motion`)
- Mobile responsive design
- SafeStorage for private browsing fallback
- E2E test coverage (14+ test cases)
- Non-invasive (all modals skippable/dismissible)
- Progress persistence via localStorage

#### CSS Files
- `web/src/styles/features/onboarding.css` - Welcome modal and tour
- `web/src/styles/features/setup-wizard.css` - Setup wizard
- `web/src/styles/features/progressive-onboarding.css` - Contextual tips

### Future Enhancements (Optional)

#### 5.1 Progressive Feature Discovery

```typescript
// web/src/app/ProgressiveOnboardingManager.ts (enhance existing)
interface OnboardingPhase {
    phase: 'setup' | 'exploration' | 'advanced' | 'power-user';
    steps: OnboardingStep[];
    triggers: PhaseTrigger[];
    completionCriteria: CompletionCriteria;
}

const PHASES: OnboardingPhase[] = [
    {
        phase: 'setup',
        steps: [
            { title: 'Connect Your Server', target: '.server-config' },
            { title: 'Import Historical Data', target: '.import-wizard' },
            { title: 'Configure Notifications', target: '.notification-settings' }
        ],
        triggers: [{ type: 'first_visit' }],
        completionCriteria: { serverConnected: true }
    },
    {
        phase: 'exploration',
        steps: [
            { title: 'Explore the Map', target: '#map' },
            { title: 'View Analytics', target: '.analytics-nav' },
            { title: 'Check Live Activity', target: '.live-activity' }
        ],
        triggers: [{ type: 'after_sync', delay: '1 day' }],
        completionCriteria: { pageVisits: { map: 1, analytics: 1 } }
    },
    {
        phase: 'advanced',
        steps: [
            { title: 'Create Custom Filters', target: '.filter-presets' },
            { title: 'Export Your Data', target: '.export-options' },
            { title: 'Configure Detection Rules', target: '.detection-config' }
        ],
        triggers: [{ type: 'after_phase', phase: 'exploration', delay: '3 days' }]
    },
    {
        phase: 'power-user',
        steps: [
            { title: 'API Access', target: '.api-tokens' },
            { title: 'Webhook Integrations', target: '.webhooks' },
            { title: 'Advanced Analytics', target: '.advanced-analytics' }
        ],
        triggers: [{ type: 'usage_threshold', threshold: { sessions: 10 } }]
    }
];
```

#### 5.2 Contextual Tooltips

```typescript
// Show tooltips based on user actions
interface ContextualTip {
    id: string;
    trigger: TipTrigger;
    content: string;
    position: 'top' | 'bottom' | 'left' | 'right';
    showOnce: boolean;
}

const CONTEXTUAL_TIPS: ContextualTip[] = [
    {
        id: 'first_filter',
        trigger: { type: 'element_visible', selector: '.filter-panel', delay: 3000 },
        content: 'Use filters to narrow down your data by date, users, or content type',
        position: 'right',
        showOnce: true
    },
    {
        id: 'globe_switch',
        trigger: { type: 'element_click', selector: '.map-mode-2d', count: 3 },
        content: 'Try the 3D globe view for a more immersive experience!',
        position: 'bottom',
        showOnce: true
    }
];
```

#### 5.3 Achievement System

```typescript
interface Achievement {
    id: string;
    name: string;
    description: string;
    icon: string;
    criteria: AchievementCriteria;
    points: number;
}

const ACHIEVEMENTS: Achievement[] = [
    { id: 'first_sync', name: 'Connected', description: 'Connected your first server', points: 10 },
    { id: 'globe_explorer', name: 'Global Viewer', description: 'Viewed the 3D globe', points: 5 },
    { id: 'filter_master', name: 'Filter Master', description: 'Created 5 filter presets', points: 15 },
    { id: 'export_pro', name: 'Data Exporter', description: 'Exported data in 3 formats', points: 10 },
    { id: 'wrapped_viewer', name: 'Year in Review', description: 'Viewed your annual wrapped report', points: 20 }
];
```

#### 5.4 Interactive Demos

- Guided demo mode with sample data
- Click-through tutorials for complex features
- Video walkthroughs embedded in help panels

### Testing Requirements

- E2E tests for each onboarding phase
- Unit tests for trigger conditions
- Accessibility tests (screen reader, keyboard)
- A/B tests for onboarding completion rates

---

## 6. Production Grafana Dashboard

**Priority**: MEDIUM
**Effort**: 2-3 days
**Competitive Advantage**: Enterprise observability

### Overview

Provide ready-to-use Grafana dashboards for Prometheus metrics.

### Dashboard Files

```
deploy/grafana/
├── dashboards/
│   ├── cartographus-overview.json       # Main overview dashboard
│   ├── cartographus-performance.json    # API latency, throughput
│   ├── cartographus-streaming.json      # Real-time streaming metrics
│   ├── cartographus-detection.json      # Security detection metrics
│   ├── cartographus-database.json       # DuckDB query performance
│   └── cartographus-auth.json           # OIDC authentication metrics
├── provisioning/
│   ├── datasources/
│   │   └── prometheus.yml
│   └── dashboards/
│       └── cartographus.yml
└── README.md
```

### Dashboard Panels

#### 6.1 Overview Dashboard

| Panel | Metrics | Visualization |
|-------|---------|---------------|
| Active Streams | `cartographus_active_streams` | Stat |
| Total Playbacks | `cartographus_playbacks_total` | Counter |
| API Latency | `cartographus_http_request_duration_seconds` | Histogram |
| Error Rate | `cartographus_http_errors_total` | Rate graph |
| Database Queries | `cartographus_db_query_duration_seconds` | Histogram |
| WebSocket Connections | `cartographus_ws_connections_active` | Gauge |

#### 6.2 Performance Dashboard

| Panel | Metrics | Visualization |
|-------|---------|---------------|
| p50/p90/p99 Latency | `histogram_quantile(0.99, ...)` | Time series |
| Throughput | `rate(cartographus_http_requests_total[5m])` | Time series |
| Slow Queries | `cartographus_db_slow_queries_total` | Counter |
| Cache Hit Rate | `cartographus_cache_hits_total / _total` | Gauge |

#### 6.3 Streaming Dashboard

| Panel | Metrics | Visualization |
|-------|---------|---------------|
| Concurrent Streams | `cartographus_concurrent_streams` | Gauge |
| Transcode Queue | `cartographus_transcode_active` | Gauge |
| Bandwidth Usage | `cartographus_bandwidth_bytes_total` | Rate graph |
| Buffer Health | `cartographus_buffer_health_score` | Heatmap |

### Docker Compose Integration

```yaml
# docker-compose.monitoring.yml
services:
  prometheus:
    image: prom/prometheus:v2.48.0
    volumes:
      - ./deploy/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:10.2.0
    volumes:
      - ./deploy/grafana/dashboards:/var/lib/grafana/dashboards
      - ./deploy/grafana/provisioning:/etc/grafana/provisioning
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

### Testing Requirements

- Validate JSON schema for all dashboards
- Integration test with Prometheus scrape
- Visual validation of rendered panels

---

## 7. Personal Access Token System

**Priority**: HIGH
**Effort**: 3-5 days
**Competitive Advantage**: Enables API integrations and automations
**Status**: **COMPLETED** (2026-01-05)

### Implementation Summary

The PAT system has been fully implemented with production-grade quality:

| Component | File | Coverage |
|-----------|------|----------|
| PAT Manager | `internal/auth/pat.go` | 92%+ |
| PAT Models | `internal/models/pat.go` | 100% |
| Database CRUD | `internal/database/pat.go` | 90%+ |
| API Handlers | `internal/api/handlers_pat.go` | 90%+ |
| API Routes | `internal/api/chi_router.go` | Integrated |

**Key Features Implemented**:
- SHA-256 + bcrypt token hashing (GitHub-style, handles 72-byte bcrypt limit)
- 12 granular token scopes (8 read, 3 write, 1 admin)
- IP allowlist validation
- Token expiration with configurable days
- Usage logging with audit trail
- Request ID tracing in all handlers
- Thread-safe validation with race condition fixes

**Test Coverage**:
- `CheckScope`: 100%
- `CheckAnyScope`: 100%
- `ValidateToken`: 92.7%
- `ExtractTokenFromHeader`: 100%
- `IsPATToken`: 100%

### Overview

Secure token system for programmatic API access, similar to GitHub PATs.

### Features

#### 7.1 Token Types

| Type | Permissions | Expiry | Use Case |
|------|-------------|--------|----------|
| Read-Only | GET endpoints only | Configurable | Monitoring, dashboards |
| Standard | GET, POST | Configurable | Integrations |
| Admin | All endpoints | Configurable | Automation |

#### 7.2 Token Scopes

```go
type TokenScope string

const (
    ScopeReadAnalytics    TokenScope = "read:analytics"
    ScopeReadUsers        TokenScope = "read:users"
    ScopeReadPlaybacks    TokenScope = "read:playbacks"
    ScopeWritePlaybacks   TokenScope = "write:playbacks"
    ScopeReadExport       TokenScope = "read:export"
    ScopeReadDetection    TokenScope = "read:detection"
    ScopeWriteDetection   TokenScope = "write:detection"
    ScopeAdmin            TokenScope = "admin"
)
```

#### 7.3 Token Format

```
carto_pat_<base64-encoded-token-id>_<random-secret>

Example: carto_pat_dXNlcjEyMw_a1b2c3d4e5f6g7h8
```

### Implementation

```go
// internal/auth/pat.go
type PersonalAccessToken struct {
    ID          string        `json:"id"`
    UserID      string        `json:"user_id"`
    Name        string        `json:"name"`
    TokenHash   string        `json:"-"`              // bcrypt hash
    Scopes      []TokenScope  `json:"scopes"`
    ExpiresAt   *time.Time    `json:"expires_at"`
    LastUsedAt  *time.Time    `json:"last_used_at"`
    LastUsedIP  string        `json:"last_used_ip"`
    CreatedAt   time.Time     `json:"created_at"`
    RevokedAt   *time.Time    `json:"revoked_at"`
}

func (m *PATManager) Create(ctx context.Context, userID string, req CreatePATRequest) (*PersonalAccessToken, string, error)
func (m *PATManager) Validate(ctx context.Context, token string) (*PersonalAccessToken, error)
func (m *PATManager) Revoke(ctx context.Context, tokenID string) error
func (m *PATManager) List(ctx context.Context, userID string) ([]PersonalAccessToken, error)
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/user/tokens` | List user's tokens |
| POST | `/api/v1/user/tokens` | Create new token |
| GET | `/api/v1/user/tokens/{id}` | Get token details |
| DELETE | `/api/v1/user/tokens/{id}` | Revoke token |
| POST | `/api/v1/user/tokens/{id}/regenerate` | Regenerate token |

### Database Schema

```sql
CREATE TABLE IF NOT EXISTS personal_access_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    scopes JSON NOT NULL,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    last_used_ip TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_pat_user ON personal_access_tokens(user_id);
CREATE INDEX idx_pat_hash ON personal_access_tokens(token_hash);
```

### Security Considerations

- Tokens hashed with bcrypt (cost 12)
- Rate limiting per token (separate from user limits)
- Automatic expiry enforcement
- Audit logging of all token usage
- IP allowlisting option per token
- Prometheus metrics for token usage patterns

### Testing Requirements

- Unit tests for token generation and validation
- Integration tests for scope enforcement
- Security tests (timing attacks, hash verification)
- E2E tests for token lifecycle

---

## 8. Updated Unraid Template

**Priority**: LOW
**Effort**: 1 day
**Competitive Advantage**: Better Unraid community support

### Current State

The existing template in `deploy/unraid/cartographus.xml` is functional but missing some new features.

### Updates Required

#### 8.1 New Configuration Options

```xml
<!-- OIDC Configuration (Zero Trust Auth) -->
<Config Name="OIDC Enabled" Target="OIDC_ENABLED" Default="false" Mode=""
        Description="Enable OpenID Connect authentication (recommended for production)"
        Type="Variable" Display="advanced" Required="false" Mask="false"/>
<Config Name="OIDC Issuer URL" Target="OIDC_ISSUER_URL" Default="" Mode=""
        Description="OIDC provider issuer URL (e.g., https://auth.example.com)"
        Type="Variable" Display="advanced" Required="false" Mask="false"/>
<Config Name="OIDC Client ID" Target="OIDC_CLIENT_ID" Default="" Mode=""
        Description="OIDC client identifier"
        Type="Variable" Display="advanced" Required="false" Mask="false"/>
<Config Name="OIDC Client Secret" Target="OIDC_CLIENT_SECRET" Default="" Mode=""
        Description="OIDC client secret"
        Type="Variable" Display="advanced" Required="true" Mask="true"/>

<!-- Newsletter Configuration -->
<Config Name="SMTP Host" Target="SMTP_HOST" Default="" Mode=""
        Description="SMTP server for newsletters (e.g., smtp.gmail.com)"
        Type="Variable" Display="advanced" Required="false" Mask="false"/>
<Config Name="SMTP Port" Target="SMTP_PORT" Default="587" Mode=""
        Description="SMTP port (587 for TLS, 465 for SSL)"
        Type="Variable" Display="advanced" Required="false" Mask="false"/>

<!-- Detection Configuration -->
<Config Name="Detection Enabled" Target="DETECTION_ENABLED" Default="true" Mode=""
        Description="Enable security detection rules"
        Type="Variable" Display="advanced" Required="false" Mask="false"/>

<!-- NATS Configuration -->
<Config Name="NATS Enabled" Target="NATS_ENABLED" Default="false" Mode=""
        Description="Enable NATS JetStream for event processing"
        Type="Variable" Display="advanced" Required="false" Mask="false"/>
<Config Name="NATS URL" Target="NATS_URL" Default="" Mode=""
        Description="NATS server URL (e.g., nats://localhost:4222)"
        Type="Variable" Display="advanced" Required="false" Mask="false"/>
```

#### 8.2 Updated Overview

```xml
<Overview>
Cartographus is a production-grade media server analytics platform that visualizes
playback activity on interactive maps. Features include:

- Real-time WebGL maps and 3D globe visualization
- 47+ analytics charts across 6 themed pages
- Security detection with 5+ anomaly rules
- Multi-server support (Plex, Jellyfin, Emby)
- Newsletter/digest system with email integration
- Annual Wrapped reports (Spotify-style)
- Personal API tokens for integrations
- OIDC/Zero Trust authentication support
- Prometheus metrics and Grafana dashboards

Track where your media content is being watched around the world with
beautiful visualizations and comprehensive analytics.
</Overview>
```

#### 8.3 Updated Categories

```xml
<Category>MediaServer:Other Tools:Productivity:</Category>
```

### Testing Requirements

- Validate XML schema
- Test template installation in Unraid (manual)
- Document all configuration options

---

## 9. Library Access Management (via Plex API)

**Priority**: MEDIUM
**Effort**: 3-4 days
**Competitive Advantage**: Unified UI for native Plex sharing features

### Overview

Provide a unified interface for managing library access using **native Plex APIs** for friends, sharing, and managed users. This ensures consistency with Plex and leverages existing infrastructure.

### Native Plex API Endpoints to Implement

These endpoints are documented in `docs/PLEX_API_GAP_ANALYSIS.md` (Part 5) as NOT YET IMPLEMENTED:

| Plex Endpoint | Method | Description |
|---------------|--------|-------------|
| `https://plex.tv/api/v2/friends` | GET | List friends with library access |
| `https://plex.tv/api/v2/friends/invite` | POST | Send friend invite |
| `https://plex.tv/api/v2/friends/{id}` | DELETE | Remove friend |
| `https://plex.tv/api/servers/{machineId}/shared_servers` | POST | Share libraries with friend |
| `https://plex.tv/api/servers/{machineId}/shared_servers/{id}` | DELETE | Revoke library sharing |
| `https://plex.tv/api/v2/home/users` | GET | List Plex Home managed users |
| `https://plex.tv/api/v2/home/users/restricted` | POST | Create managed user |

### Features

#### 9.1 Friends Management

```go
// internal/sync/plex_friends.go
type PlexFriend struct {
    ID                int       `json:"id"`
    UUID              string    `json:"uuid"`
    Username          string    `json:"username"`
    Email             string    `json:"email"`
    Thumb             string    `json:"thumb"`
    Server            bool      `json:"server"`          // Has server access
    Home              bool      `json:"home"`            // Is Plex Home member
    AllowSync         bool      `json:"allowSync"`       // Can sync content
    AllowCameraUpload bool      `json:"allowCameraUpload"`
    AllowChannels     bool      `json:"allowChannels"`
    SharedSections    []string  `json:"sharedSections"`  // Library section IDs
    Status            string    `json:"status"`          // "accepted", "pending"
}

type PlexFriendsClient struct {
    token   string
    baseURL string
    client  *http.Client
}

func (c *PlexFriendsClient) ListFriends(ctx context.Context) ([]PlexFriend, error)
func (c *PlexFriendsClient) InviteFriend(ctx context.Context, email string, sections []string) error
func (c *PlexFriendsClient) RemoveFriend(ctx context.Context, friendID int) error
func (c *PlexFriendsClient) UpdateSharing(ctx context.Context, friendID int, sections []string) error
```

#### 9.2 Managed Users (Plex Home)

```go
// internal/sync/plex_home.go
type PlexManagedUser struct {
    ID                 int    `json:"id"`
    UUID               string `json:"uuid"`
    Username           string `json:"username"`
    Thumb              string `json:"thumb"`
    Restricted         bool   `json:"restricted"`         // Content restrictions
    RestrictionProfile string `json:"restrictionProfile"` // "little_kid", "older_kid", "teen"
    Home               bool   `json:"home"`
    HomeAdmin          bool   `json:"homeAdmin"`
}

func (c *PlexHomeClient) ListManagedUsers(ctx context.Context) ([]PlexManagedUser, error)
func (c *PlexHomeClient) CreateManagedUser(ctx context.Context, name string, profile string) (*PlexManagedUser, error)
func (c *PlexHomeClient) DeleteManagedUser(ctx context.Context, userID int) error
func (c *PlexHomeClient) UpdateRestrictions(ctx context.Context, userID int, profile string) error
```

#### 9.3 Library Sharing Parameters

When sharing libraries via Plex API, these parameters are supported:

| Parameter | Type | Description |
|-----------|------|-------------|
| `allowSync` | bool | Allow offline sync/download |
| `allowCameraUpload` | bool | Allow camera roll uploads |
| `allowChannels` | bool | Allow channel access |
| `filterMovies` | string | Movie library filter (label-based) |
| `filterTelevision` | string | TV library filter (label-based) |
| `filterMusic` | string | Music library filter |

### Cartographus API Endpoints (Proxy to Plex)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/plex/friends` | List Plex friends |
| POST | `/api/v1/plex/friends/invite` | Send friend invitation |
| DELETE | `/api/v1/plex/friends/{id}` | Remove friend |
| PUT | `/api/v1/plex/friends/{id}/sharing` | Update library sharing |
| GET | `/api/v1/plex/home/users` | List managed users |
| POST | `/api/v1/plex/home/users` | Create managed user |
| DELETE | `/api/v1/plex/home/users/{id}` | Delete managed user |
| PUT | `/api/v1/plex/home/users/{id}/restrictions` | Update restrictions |

### Models

```go
// internal/models/plex_sharing.go

// PlexFriendsResponse represents GET https://plex.tv/api/v2/friends
type PlexFriendsResponse struct {
    Friends []PlexFriend `json:"friends"`
}

// PlexInviteRequest for POST https://plex.tv/api/v2/friends/invite
type PlexInviteRequest struct {
    Email         string   `json:"email" validate:"required,email"`
    LibrarySections []int  `json:"library_sections"`
    Settings      struct {
        AllowSync         bool `json:"allowSync"`
        AllowCameraUpload bool `json:"allowCameraUpload"`
        AllowChannels     bool `json:"allowChannels"`
    } `json:"settings"`
}

// PlexSharedServerRequest for POST https://plex.tv/api/servers/{id}/shared_servers
type PlexSharedServerRequest struct {
    InvitedEmail     string `json:"invitedEmail" validate:"required,email"`
    LibrarySectionIDs []int `json:"librarySectionIds"`
    Settings         struct {
        AllowSync     bool   `json:"allowSync"`
        FilterMovies  string `json:"filterMovies,omitempty"`
        FilterTV      string `json:"filterTelevision,omitempty"`
    } `json:"settings"`
}

// PlexHomeUsersResponse for GET https://plex.tv/api/v2/home/users
type PlexHomeUsersResponse struct {
    Users []PlexManagedUser `json:"users"`
}

// PlexCreateManagedUserRequest for POST https://plex.tv/api/v2/home/users/restricted
type PlexCreateManagedUserRequest struct {
    Name               string `json:"name" validate:"required,min=1,max=50"`
    RestrictionProfile string `json:"restrictionProfile" validate:"oneof=little_kid older_kid teen"`
}
```

### Implementation Notes

1. **Authentication**: All plex.tv API calls require the user's Plex token (obtained via Plex OAuth)
2. **Server ID**: Sharing operations require the Plex server's `machineIdentifier`
3. **Rate Limiting**: Plex.tv API has rate limits - implement with circuit breaker
4. **Caching**: Cache friend list with 5-minute TTL to reduce API calls
5. **Error Handling**: Map Plex API errors to user-friendly messages

### Frontend Components

```typescript
// web/src/app/FriendsManager.ts
class FriendsManager {
    async listFriends(): Promise<PlexFriend[]>
    async inviteFriend(email: string, sections: number[]): Promise<void>
    async removeFriend(friendId: number): Promise<void>
    async updateSharing(friendId: number, sections: number[]): Promise<void>
}

// UI Components
- FriendsList: Table of current friends with sharing status
- InviteFriendModal: Email input + library selection
- SharingEditor: Toggle libraries for a specific friend
- ManagedUsersPanel: Plex Home member management
```

### Testing Requirements

- Unit tests for Plex API client methods
- Integration tests with mock Plex.tv server
- E2E tests for friend invitation flow
- Error handling for API failures (expired token, rate limit)

---

## 10. Additional High-Value Features

### 10.1 Embeddable Widgets

**Priority**: LOW
**Effort**: 2-3 days

```html
<!-- Embed current activity widget on any webpage -->
<iframe src="https://cartographus.local/embed/activity?theme=dark"
        width="400" height="300"></iframe>

<!-- Embed stats widget -->
<iframe src="https://cartographus.local/embed/stats?period=week"
        width="300" height="200"></iframe>
```

### 10.2 Keyboard Shortcut System

**Priority**: LOW
**Effort**: 1-2 days

```typescript
const SHORTCUTS = {
    'g m': 'Go to Map',
    'g a': 'Go to Analytics',
    'g l': 'Go to Live Activity',
    'g s': 'Go to Settings',
    '/': 'Focus search',
    'f': 'Toggle filters',
    '?': 'Show shortcuts help',
    'Escape': 'Close modals/panels'
};
```

### 10.3 Command Palette (CMD+K)

**Priority**: LOW
**Effort**: 2-3 days

```typescript
// Quick access to any feature via keyboard
interface CommandPaletteItem {
    id: string;
    label: string;
    keywords: string[];
    action: () => void;
    shortcut?: string;
}
```

### 10.4 Data Retention Policies

**Priority**: MEDIUM
**Effort**: 3-4 days

```go
// internal/retention/policy.go
type RetentionPolicy struct {
    ID              string
    Name            string
    RetentionDays   int           // Delete after N days
    ArchiveFirst    bool          // Archive before delete
    Tables          []string      // Affected tables
    Enabled         bool
}

// Scheduled cleanup job
func (r *RetentionManager) RunCleanup(ctx context.Context) (CleanupReport, error)
```

### 10.5 Internationalization (i18n)

**Priority**: MEDIUM
**Effort**: 5-7 days

```typescript
// Language support for UI
const translations = {
    en: { 'nav.map': 'Map', 'nav.analytics': 'Analytics' },
    es: { 'nav.map': 'Mapa', 'nav.analytics': 'Analíticas' },
    de: { 'nav.map': 'Karte', 'nav.analytics': 'Analytik' },
    fr: { 'nav.map': 'Carte', 'nav.analytics': 'Analytiques' }
};
```

### 10.6 Mobile App (React Native)

**Priority**: LOW
**Effort**: 30+ days

Long-term consideration for native mobile experience matching Plex Dash.

### 10.7 Plugin/Extension System

**Priority**: LOW
**Effort**: 15-20 days

Allow community extensions for:
- Custom charts
- Additional data sources
- Notification integrations
- Theme customization

---

## Implementation Priority Matrix

| Feature | Impact | Effort | Priority | Status | Dependencies |
|---------|--------|--------|----------|--------|--------------|
| Personal Access Tokens | HIGH | 3-5 days | P1 | **DONE** | None |
| Annual Wrapped Reports | HIGH | 5-7 days | P1 | **DONE** | None |
| Newsletter Generator | HIGH | 7-10 days | P1 | **DONE** | None |
| Recommendation Algorithms | HIGH | 7-10 days | P2 | **DONE** | ADR-0024 |
| User Onboarding | MEDIUM-HIGH | 3-5 days | P2 | **DONE** | None |
| Library Access Management (Plex API) | MEDIUM | 3-4 days | P2 | **DONE** | Plex OAuth |
| Advanced Charts | MEDIUM-HIGH | 5-7 days | P2 | **DONE** | ECharts |
| Grafana Dashboards | MEDIUM | 2-3 days | P3 | **DONE** | Prometheus |
| Unraid Template Update | LOW | 1 day | P4 | **DONE** | None |
| Additional Features | LOW-MEDIUM | Varies | P4-P5 | Not Started | Varies |

---

## Quality Standards

All features MUST meet these criteria:

### Testability
- Unit test coverage: 90%+
- Integration tests for all API endpoints
- E2E tests for user-facing features
- Performance benchmarks with thresholds

### Observability
- Prometheus metrics for key operations
- Structured logging with correlation IDs
- Distributed tracing (OpenTelemetry ready)
- Health check endpoints

### Traceability
- Audit logs for sensitive operations
- Request correlation IDs
- Database query logging
- Version tracking for data changes

### Security
- Input validation with go-playground/validator
- SQL injection prevention (parameterized queries)
- XSS prevention (HTML escaping)
- CSRF protection for state-changing operations
- Rate limiting per endpoint/user

### Mathematical Correctness
- Documented algorithms with references
- Deterministic outputs (seeded randomness)
- Numeric precision handling (decimal types)
- Edge case handling (empty sets, division by zero)

---

## Conclusion

This roadmap provides a comprehensive set of production-grade features that would significantly enhance Cartographus's competitive position. The features are prioritized by impact and effort, with clear implementation guidelines and testing requirements.

### Progress Summary

**Completed (9/10)**:
- **Personal Access Tokens** - Full implementation with 92%+ auth coverage, SHA-256+bcrypt hashing, 12 scopes, IP allowlisting, usage logging, request tracing
- **Annual Wrapped Reports** - Backend + frontend, 20+ tests, RBAC, share tokens
- **Newsletter Generator** - Full stack: scheduler, delivery, templates, ContentStore, UI
- **User Onboarding** - Setup wizard, welcome tour, progressive tips (55+), WCAG 2.1 AA
- **Enhanced Recommendations** - 13 algorithms (EASE, ALS, UserCF, ItemCF, Content-Based, Co-Visitation, Popularity, FPMC, BPR, Time-Aware CF, Multi-Hop ItemCF, Markov Chain, LinUCB), What's Next widget, algorithm tooltips, full API with 11 endpoints
- **Library Access Management** - Plex friends/sharing/managed users via plex.tv API, 12 handlers with admin RBAC, frontend FriendsManager UI
- **Advanced Charts** - 6 chart types (Sankey, Chord, Radar, Treemap, Calendar Heatmap, Bump Chart), RBAC enforcement with ExecuteUserScoped/ExecuteAdminOnly, 34+ tests, per-user caching
- **Grafana Dashboards** - 6 production-grade dashboards (Overview, Performance, Streaming, Detection, Database, Auth), Prometheus configuration, Docker Compose monitoring stack
- **Unraid Template Update** - Comprehensive template with 100+ configuration options covering all features: OIDC/Zero Trust authentication, Plex OAuth, Casbin RBAC, Newsletter scheduler, Detection engine, Discord/Webhook notifications, NATS JetStream, Recommendation engine, VPN detection, GeoIP providers, database tuning, sync configuration, and more

**Recommended Next Steps:**
1. Additional High-Value Features - Email notifications, keyboard shortcuts, embeddable widgets
2. Plugin/Extension System - Community extensions for custom charts and integrations

Each feature should be implemented with full test coverage, documentation, and observability before moving to the next.

### Files Modified for PAT Implementation

| File | Purpose |
|------|---------|
| `internal/auth/pat.go` | PAT manager with create, validate, revoke, regenerate |
| `internal/auth/pat_test.go` | Comprehensive tests including error path coverage |
| `internal/models/pat.go` | PAT models, scopes, request/response types |
| `internal/models/pat_test.go` | Model validation tests |
| `internal/database/pat.go` | DuckDB CRUD with NULL handling for JSON columns |
| `internal/database/pat_test.go` | Database integration tests |
| `internal/database/database_schema.go` | PAT table schema |
| `internal/api/handlers_pat.go` | HTTP handlers with request ID tracing |
| `internal/api/handlers_pat_test.go` | Handler tests with mock store |
| `internal/api/chi_router.go` | 7 new PAT routes under `/api/v1/user/tokens` |

### Files Modified for Newsletter Generator Implementation

| File | Purpose |
|------|---------|
| `internal/newsletter/scheduler/cron.go` | Cron expression parser with standard 5-field format |
| `internal/newsletter/scheduler/cron_test.go` | Cron parser tests (17 tests) |
| `internal/newsletter/scheduler/scheduler.go` | Scheduler service with concurrent execution |
| `internal/newsletter/scheduler/scheduler_test.go` | Scheduler tests |
| `internal/newsletter/templates.go` | HTML template engine with variable substitution |
| `internal/newsletter/content.go` | ContentResolver for fetching newsletter data |
| `internal/newsletter/delivery/manager.go` | Delivery manager with retry logic |
| `internal/newsletter/delivery/manager_test.go` | Delivery manager tests |
| `internal/newsletter/delivery/channels.go` | Channel registry and common utilities |
| `internal/newsletter/delivery/channels_test.go` | Channel validation tests (23 tests) |
| `internal/newsletter/delivery/email.go` | SMTP email delivery |
| `internal/newsletter/delivery/discord.go` | Discord webhook delivery |
| `internal/newsletter/delivery/slack.go` | Slack webhook delivery |
| `internal/newsletter/delivery/telegram.go` | Telegram Bot API delivery |
| `internal/newsletter/delivery/webhook.go` | Generic webhook delivery |
| `internal/newsletter/delivery/inapp.go` | In-app notification delivery |
| `internal/database/newsletter_content.go` | ContentStore with 9 database methods |
| `internal/database/newsletter_content_test.go` | ContentStore tests (10 tests) |
| `internal/config/koanf.go` | Newsletter config defaults and env mappings |
| `internal/config/config.go` | NewsletterConfig struct |
| `cmd/server/newsletter_init.go` | Supervisor tree integration |
| `cmd/server/main.go` | initNewsletter() call |
| `internal/supervisor/services/newsletter.go` | Suture service wrapper |

### Files Modified for Library Access Management Implementation

| File | Purpose |
|------|---------|
| `internal/sync/plex_friends.go` | PlexTVClient for plex.tv API (friends, sharing, managed users) |
| `internal/sync/plex_friends_test.go` | Tests with full struct field assertions |
| `internal/api/handlers_plex_friends.go` | 12 HTTP handlers with admin authorization |
| `internal/api/errors.go` | ErrPlexNotEnabled, ErrPlexTokenRequired errors |
| `internal/api/chi_router.go` | 13 Plex routes (/api/v1/plex/friends, /sharing, /home/users, /libraries) |
| `internal/models/plex_sharing.go` | Request/response models with validation |
| `web/src/app/FriendsManager.ts` | Frontend UI component with tabs |
| `web/src/lib/types/plex.ts` | Friends/sharing TypeScript types |
| `web/src/lib/api/plex.ts` | 12 typed API methods for Plex operations |
| `web/src/lib/api/index.ts` | API facade exposing new Plex methods |

### Files Modified for Advanced Charts Implementation

| File | Purpose |
|------|---------|
| `internal/database/analytics_advanced_charts.go` | 6 database methods for chart data |
| `internal/api/handlers_analytics_charts.go` | 6 HTTP handlers with RBAC |
| `internal/api/handlers_analytics_charts_test.go` | 34+ tests including RBAC enforcement |
| `internal/api/analytics_executor.go` | ExecuteUserScoped, ExecuteAdminOnly, ExecuteWithParamUserScoped |
| `internal/api/handlers_analytics.go` | Updated all handlers to use ExecuteUserScoped |
| `internal/api/handlers_analytics_enhanced.go` | Updated all handlers to use ExecuteUserScoped |
| `internal/api/handlers_analytics_discovery.go` | Updated all handlers to use ExecuteUserScoped |
| `internal/api/chi_router.go` | 6 new routes under `/api/v1/analytics/` |
| `web/src/lib/types/analytics.ts` | 6 TypeScript types for chart responses |
| `web/src/lib/api/analytics.ts` | 6 API client methods for chart endpoints |
