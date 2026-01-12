# DuckDB Extensions Implementation Plan

**Last Verified**: 2026-01-11
**Status**: PARTIAL - RapidFuzz and DataSketches Implemented

> **Implementation Status**: RapidFuzz and DataSketches extensions have been implemented
> (see `internal/database/search_fuzzy.go`, `internal/database/analytics_approximate.go`).
> VSS (Vector Similarity Search) remains a future enhancement.

## Overview

This document outlines the implementation plan for adding three DuckDB community extensions to Cartographus:

1. **RapidFuzz** - Fuzzy string matching for improved search UX
2. **DataSketches** - Approximate analytics for performance at scale
3. **VSS** - Vector similarity search for content recommendations

**Priority Order**: RapidFuzz > DataSketches > VSS

---

## Phase 1: RapidFuzz Extension

### Purpose
Enable fuzzy string matching for better search experience, data deduplication, and autocomplete functionality.

### Current State
- Search is proxied to Tautulli API (`handlers_tautulli.go:665-678`)
- Local DuckDB queries use exact `LIKE` pattern matching (`analytics_bandwidth.go:507-510`)
- Frontend `SearchManager.ts` handles search UI with debouncing

### Implementation Steps

#### Step 1.1: Extension Installation

**File**: `internal/database/database_extensions.go`

Add new extension installer function (following existing pattern):

```go
// installRapidFuzz installs the RapidFuzz community extension for fuzzy string matching
func (db *DB) installRapidFuzz(optional bool) error {
    ctx, cancel := extensionContext()
    defer cancel()

    if _, err := db.conn.ExecContext(ctx, "INSTALL rapidfuzz FROM community;"); err != nil {
        if _, loadErr := db.conn.ExecContext(ctx, "LOAD rapidfuzz;"); loadErr != nil {
            if _, forceErr := db.conn.ExecContext(ctx, "FORCE INSTALL rapidfuzz FROM community;"); forceErr != nil {
                if optional {
                    db.rapidfuzzAvailable = false
                    fmt.Printf("Warning: RapidFuzz extension unavailable, fuzzy search will be disabled\n")
                    return nil
                }
                return fmt.Errorf("failed to install rapidfuzz extension: %w", forceErr)
            }
        }
    }

    if db.rapidfuzzAvailable {
        if _, err := db.conn.ExecContext(ctx, "LOAD rapidfuzz;"); err != nil {
            if optional {
                db.rapidfuzzAvailable = false
                return nil
            }
            return fmt.Errorf("failed to load rapidfuzz extension: %w", err)
        }

        // Verify RapidFuzz functions are available
        var testScore int
        if err := db.conn.QueryRowContext(ctx,
            "SELECT rapidfuzz_ratio('hello', 'helo')").Scan(&testScore); err != nil {
            if optional {
                db.rapidfuzzAvailable = false
                return nil
            }
            return fmt.Errorf("rapidfuzz functions unavailable: %w", err)
        }
    }

    return nil
}
```

**File**: `internal/database/database.go`

Add field to DB struct:

```go
type DB struct {
    // ... existing fields ...
    rapidfuzzAvailable bool // Tracks whether rapidfuzz extension is loaded
}
```

Add availability check method:

```go
// IsRapidFuzzAvailable returns whether the rapidfuzz extension is available
func (db *DB) IsRapidFuzzAvailable() bool {
    return db.rapidfuzzAvailable
}
```

**File**: `scripts/setup-duckdb-extensions.sh`

Add to community extensions section:

```bash
# Download community extensions
download_extension "h3" "$COMMUNITY_REPO"
download_extension "rapidfuzz" "$COMMUNITY_REPO"  # NEW
```

#### Step 1.2: Create Fuzzy Search Database Methods

**New File**: `internal/database/search_fuzzy.go`

```go
package database

import (
    "context"
    "fmt"
)

// FuzzySearchResult represents a fuzzy search match
type FuzzySearchResult struct {
    ID              string  `json:"id"`
    Title           string  `json:"title"`
    ParentTitle     string  `json:"parent_title,omitempty"`
    GrandparentTitle string `json:"grandparent_title,omitempty"`
    MediaType       string  `json:"media_type"`
    Score           int     `json:"score"`
    Year            int     `json:"year,omitempty"`
}

// FuzzySearchPlaybacks searches playback events with fuzzy matching
// Uses rapidfuzz_ratio for overall similarity scoring
func (db *DB) FuzzySearchPlaybacks(ctx context.Context, query string, minScore int, limit int) ([]FuzzySearchResult, error) {
    if !db.rapidfuzzAvailable {
        return db.fallbackExactSearch(ctx, query, limit)
    }

    if minScore < 0 || minScore > 100 {
        minScore = 70 // Default threshold
    }
    if limit <= 0 || limit > 100 {
        limit = 20
    }

    sql := `
        WITH scored_results AS (
            SELECT DISTINCT
                rating_key as id,
                title,
                parent_title,
                grandparent_title,
                media_type,
                year,
                GREATEST(
                    rapidfuzz_ratio(LOWER(title), LOWER(?)),
                    COALESCE(rapidfuzz_ratio(LOWER(grandparent_title), LOWER(?)), 0),
                    rapidfuzz_token_set_ratio(
                        LOWER(COALESCE(grandparent_title, '') || ' ' || COALESCE(parent_title, '') || ' ' || title),
                        LOWER(?)
                    )
                ) as score
            FROM playback_events
            WHERE title IS NOT NULL
        )
        SELECT id, title, parent_title, grandparent_title, media_type, score, year
        FROM scored_results
        WHERE score >= ?
        ORDER BY score DESC, title ASC
        LIMIT ?
    `

    rows, err := db.conn.QueryContext(ctx, sql, query, query, query, minScore, limit)
    if err != nil {
        return nil, fmt.Errorf("fuzzy search failed: %w", err)
    }
    defer rows.Close()

    var results []FuzzySearchResult
    for rows.Next() {
        var r FuzzySearchResult
        var parentTitle, grandparentTitle sql.NullString
        var year sql.NullInt64

        if err := rows.Scan(&r.ID, &r.Title, &parentTitle, &grandparentTitle,
                           &r.MediaType, &r.Score, &year); err != nil {
            return nil, fmt.Errorf("scan error: %w", err)
        }

        if parentTitle.Valid {
            r.ParentTitle = parentTitle.String
        }
        if grandparentTitle.Valid {
            r.GrandparentTitle = grandparentTitle.String
        }
        if year.Valid {
            r.Year = int(year.Int64)
        }

        results = append(results, r)
    }

    return results, nil
}

// FuzzySearchUsers searches users with fuzzy matching on username
func (db *DB) FuzzySearchUsers(ctx context.Context, query string, minScore int, limit int) ([]UserSearchResult, error) {
    if !db.rapidfuzzAvailable {
        return db.fallbackUserSearch(ctx, query, limit)
    }

    sql := `
        SELECT DISTINCT
            user_id,
            username,
            friendly_name,
            rapidfuzz_ratio(LOWER(username), LOWER(?)) as score
        FROM playback_events
        WHERE username IS NOT NULL
          AND rapidfuzz_ratio(LOWER(username), LOWER(?)) >= ?
        ORDER BY score DESC
        LIMIT ?
    `

    // ... implementation similar to above
}

// fallbackExactSearch provides exact match when RapidFuzz unavailable
func (db *DB) fallbackExactSearch(ctx context.Context, query string, limit int) ([]FuzzySearchResult, error) {
    sql := `
        SELECT DISTINCT
            rating_key as id,
            title,
            parent_title,
            grandparent_title,
            media_type,
            100 as score,
            year
        FROM playback_events
        WHERE LOWER(title) LIKE LOWER('%' || ? || '%')
           OR LOWER(grandparent_title) LIKE LOWER('%' || ? || '%')
        ORDER BY title ASC
        LIMIT ?
    `
    // ... implementation
}
```

#### Step 1.3: Add API Endpoint

**File**: `internal/api/chi_router.go`

Add new route in analytics group:

```go
r.Route("/analytics", func(r chi.Router) {
    // ... existing routes ...
    r.Get("/search/fuzzy", h.FuzzySearch) // NEW
})
```

**New File**: `internal/api/handlers_search_fuzzy.go`

```go
package api

import (
    "net/http"
    "strconv"
)

// FuzzySearchParams holds fuzzy search parameters
type FuzzySearchParams struct {
    Query    string `validate:"required,min=1,max=200"`
    MinScore int    `validate:"omitempty,min=0,max=100"`
    Limit    int    `validate:"omitempty,min=1,max=100"`
}

// FuzzySearch handles fuzzy search requests against local DuckDB
func (h *Handler) FuzzySearch(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        respondError(w, http.StatusBadRequest, "Query parameter 'q' is required")
        return
    }

    minScore, _ := strconv.Atoi(r.URL.Query().Get("min_score"))
    if minScore == 0 {
        minScore = 70
    }

    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 {
        limit = 20
    }

    results, err := h.db.FuzzySearchPlaybacks(r.Context(), query, minScore, limit)
    if err != nil {
        h.logger.Error("Fuzzy search failed", "error", err)
        respondError(w, http.StatusInternalServerError, "Search failed")
        return
    }

    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status":  "success",
        "results": results,
        "count":   len(results),
        "fuzzy":   h.db.IsRapidFuzzAvailable(),
    })
}
```

#### Step 1.4: Frontend Integration

**File**: `web/src/lib/api.ts`

Add new method:

```typescript
async fuzzySearch(query: string, options?: { minScore?: number; limit?: number }): Promise<FuzzySearchResponse> {
    const params = new URLSearchParams({ q: query });
    if (options?.minScore) params.set('min_score', options.minScore.toString());
    if (options?.limit) params.set('limit', options.limit.toString());

    return this.get<FuzzySearchResponse>(`/api/v1/analytics/search/fuzzy?${params}`);
}
```

**File**: `web/src/app/SearchManager.ts`

Modify `performSearch` to use fuzzy endpoint:

```typescript
private async performSearch(): Promise<void> {
    // ... existing validation ...

    try {
        // Try local fuzzy search first, fall back to Tautulli
        const localResults = await this.api.fuzzySearch(trimmedQuery, { minScore: 70, limit: 50 });
        if (localResults.results.length > 0) {
            this.results = localResults.results.map(this.mapFuzzyToTautulli);
            this.resultsCount = localResults.count;
            this.isFuzzySearch = localResults.fuzzy;
        } else {
            // Fall back to Tautulli search
            const data = await this.api.search(trimmedQuery, 50);
            this.results = data.results || [];
            this.resultsCount = data.results_count || 0;
            this.isFuzzySearch = false;
        }
        // ... rest of method
    }
}
```

#### Step 1.5: Tests

**New File**: `internal/database/search_fuzzy_test.go`

```go
package database

import (
    "context"
    "testing"
)

func TestFuzzySearchPlaybacks(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Insert test data
    insertTestPlaybacks(t, db, []testPlayback{
        {Title: "Breaking Bad", MediaType: "show"},
        {Title: "Breaking Dawn", MediaType: "movie"},
        {Title: "Better Call Saul", MediaType: "show"},
    })

    tests := []struct {
        name      string
        query     string
        minScore  int
        wantCount int
        wantFirst string
    }{
        {"exact match", "Breaking Bad", 90, 1, "Breaking Bad"},
        {"typo match", "Braking Bad", 70, 1, "Breaking Bad"},
        {"partial match", "Breaking", 70, 2, "Breaking Bad"},
        {"no match", "xyznonexistent", 70, 0, ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            results, err := db.FuzzySearchPlaybacks(context.Background(), tt.query, tt.minScore, 10)
            if err != nil {
                t.Fatalf("FuzzySearchPlaybacks() error = %v", err)
            }
            if len(results) != tt.wantCount {
                t.Errorf("got %d results, want %d", len(results), tt.wantCount)
            }
            if tt.wantCount > 0 && results[0].Title != tt.wantFirst {
                t.Errorf("first result = %s, want %s", results[0].Title, tt.wantFirst)
            }
        })
    }
}
```

### Deliverables for Phase 1
- [ ] Extension installation in `database_extensions.go`
- [ ] DB struct update with `rapidfuzzAvailable` flag
- [ ] Setup script update for extension download
- [ ] `search_fuzzy.go` with database methods
- [ ] API handler for fuzzy search endpoint
- [ ] Chi router update
- [ ] Frontend API method
- [ ] SearchManager integration
- [ ] Unit tests (90%+ coverage)
- [ ] E2E test for fuzzy search

---

## Phase 2: DataSketches Extension

### Purpose
Enable approximate analytics using probabilistic data structures for improved performance at scale (10,000+ locations, millions of events).

### Current State
- Exact `COUNT(DISTINCT)` used extensively (50+ occurrences in analytics)
- Exact `PERCENTILE_CONT` for median calculations (10+ occurrences)
- Analytics queries can be slow on large datasets

### Key Functions to Leverage
- **HyperLogLog** (`hll_count_distinct`) - ~10x faster approximate distinct counts
- **KLL Sketches** (`kll_quantile`) - Streaming percentile estimation
- **Theta Sketches** - Set operations on large datasets

### Implementation Steps

#### Step 2.1: Extension Installation

**File**: `internal/database/database_extensions.go`

```go
// installDataSketches installs the DataSketches community extension
func (db *DB) installDataSketches(optional bool) error {
    ctx, cancel := extensionContext()
    defer cancel()

    if _, err := db.conn.ExecContext(ctx, "INSTALL datasketches FROM community;"); err != nil {
        if _, loadErr := db.conn.ExecContext(ctx, "LOAD datasketches;"); loadErr != nil {
            if optional {
                db.datasketchesAvailable = false
                fmt.Printf("Warning: DataSketches extension unavailable, using exact analytics\n")
                return nil
            }
            return fmt.Errorf("failed to install datasketches: %w", err)
        }
    }

    if _, err := db.conn.ExecContext(ctx, "LOAD datasketches;"); err != nil {
        if optional {
            db.datasketchesAvailable = false
            return nil
        }
        return fmt.Errorf("failed to load datasketches: %w", err)
    }

    // Verify HLL functions
    var testCount int64
    if err := db.conn.QueryRowContext(ctx,
        "SELECT datasketch_hll_count(datasketch_hll_create())").Scan(&testCount); err != nil {
        if optional {
            db.datasketchesAvailable = false
            return nil
        }
        return fmt.Errorf("datasketches functions unavailable: %w", err)
    }

    return nil
}
```

#### Step 2.2: Create Approximate Analytics Methods

**New File**: `internal/database/analytics_approximate.go`

```go
package database

import (
    "context"
    "fmt"
)

// ApproximateStats contains approximate statistics
type ApproximateStats struct {
    ApproxUniqueUsers    int64   `json:"approx_unique_users"`
    ApproxUniqueSessions int64   `json:"approx_unique_sessions"`
    ApproxUniqueIPs      int64   `json:"approx_unique_ips"`
    MedianDuration       float64 `json:"median_duration"`
    P95Duration          float64 `json:"p95_duration"`
    IsApproximate        bool    `json:"is_approximate"`
}

// GetApproximateStats returns approximate statistics using DataSketches
// Falls back to exact queries if extension unavailable
func (db *DB) GetApproximateStats(ctx context.Context) (*ApproximateStats, error) {
    if db.datasketchesAvailable {
        return db.getApproximateStatsWithSketches(ctx)
    }
    return db.getExactStats(ctx)
}

func (db *DB) getApproximateStatsWithSketches(ctx context.Context) (*ApproximateStats, error) {
    sql := `
        SELECT
            datasketch_hll_count(datasketch_hll_merge(user_sketch)) as approx_users,
            datasketch_hll_count(datasketch_hll_merge(session_sketch)) as approx_sessions,
            datasketch_hll_count(datasketch_hll_merge(ip_sketch)) as approx_ips
        FROM (
            SELECT
                datasketch_hll_create(user_id) as user_sketch,
                datasketch_hll_create(session_key) as session_sketch,
                datasketch_hll_create(ip_address) as ip_sketch
            FROM playback_events
        ) sketches
    `

    var stats ApproximateStats
    stats.IsApproximate = true

    err := db.conn.QueryRowContext(ctx, sql).Scan(
        &stats.ApproxUniqueUsers,
        &stats.ApproxUniqueSessions,
        &stats.ApproxUniqueIPs,
    )
    if err != nil {
        return nil, fmt.Errorf("approximate stats query failed: %w", err)
    }

    // Get percentiles using KLL sketches
    percentileSql := `
        SELECT
            datasketch_kll_quantile(duration_sketch, 0.5) as median,
            datasketch_kll_quantile(duration_sketch, 0.95) as p95
        FROM (
            SELECT datasketch_kll_create(COALESCE(play_duration, 0)::DOUBLE) as duration_sketch
            FROM playback_events
        )
    `

    err = db.conn.QueryRowContext(ctx, percentileSql).Scan(
        &stats.MedianDuration,
        &stats.P95Duration,
    )
    if err != nil {
        return nil, fmt.Errorf("percentile query failed: %w", err)
    }

    return &stats, nil
}

func (db *DB) getExactStats(ctx context.Context) (*ApproximateStats, error) {
    sql := `
        SELECT
            COUNT(DISTINCT user_id) as unique_users,
            COUNT(DISTINCT session_key) as unique_sessions,
            COUNT(DISTINCT ip_address) as unique_ips,
            COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY COALESCE(play_duration, 0)), 0) as median,
            COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY COALESCE(play_duration, 0)), 0) as p95
        FROM playback_events
    `

    var stats ApproximateStats
    stats.IsApproximate = false

    err := db.conn.QueryRowContext(ctx, sql).Scan(
        &stats.ApproxUniqueUsers,
        &stats.ApproxUniqueSessions,
        &stats.ApproxUniqueIPs,
        &stats.MedianDuration,
        &stats.P95Duration,
    )
    if err != nil {
        return nil, fmt.Errorf("exact stats query failed: %w", err)
    }

    return &stats, nil
}

// GetApproximateUniqueUsers returns approximate unique user count per time bucket
// Useful for trend charts with large datasets
func (db *DB) GetApproximateUniqueUsersTrend(ctx context.Context, days int) ([]TrendPoint, error) {
    if !db.datasketchesAvailable {
        return db.GetExactUniqueUsersTrend(ctx, days)
    }

    sql := `
        SELECT
            strftime('%Y-%m-%d', started_at) as date,
            datasketch_hll_count(datasketch_hll_create(user_id)) as unique_users
        FROM playback_events
        WHERE started_at >= CURRENT_DATE - INTERVAL ? DAY
        GROUP BY strftime('%Y-%m-%d', started_at)
        ORDER BY date ASC
    `

    rows, err := db.conn.QueryContext(ctx, sql, days)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []TrendPoint
    for rows.Next() {
        var point TrendPoint
        if err := rows.Scan(&point.Date, &point.Value); err != nil {
            return nil, err
        }
        results = append(results, point)
    }

    return results, nil
}
```

#### Step 2.3: Pre-computed Sketch Tables (Optional Optimization)

For very large datasets, consider pre-computing sketches:

**New File**: `internal/database/schema_sketches.go`

```go
// createSketchTables creates tables for pre-computed sketches
func (db *DB) createSketchTables() error {
    if !db.datasketchesAvailable {
        return nil
    }

    sql := `
        CREATE TABLE IF NOT EXISTS daily_sketches (
            date DATE PRIMARY KEY,
            user_hll BLOB,      -- HyperLogLog sketch for users
            session_hll BLOB,   -- HyperLogLog sketch for sessions
            ip_hll BLOB,        -- HyperLogLog sketch for IPs
            duration_kll BLOB,  -- KLL sketch for duration percentiles
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `

    _, err := db.conn.ExecContext(context.Background(), sql)
    return err
}

// UpdateDailySketches updates the pre-computed sketches for a given date
func (db *DB) UpdateDailySketches(ctx context.Context, date string) error {
    if !db.datasketchesAvailable {
        return nil
    }

    sql := `
        INSERT OR REPLACE INTO daily_sketches (date, user_hll, session_hll, ip_hll, duration_kll)
        SELECT
            ?::DATE,
            datasketch_hll_create(user_id),
            datasketch_hll_create(session_key),
            datasketch_hll_create(ip_address),
            datasketch_kll_create(COALESCE(play_duration, 0)::DOUBLE)
        FROM playback_events
        WHERE strftime('%Y-%m-%d', started_at) = ?
    `

    _, err := db.conn.ExecContext(ctx, sql, date, date)
    return err
}
```

#### Step 2.4: API Integration

**File**: `internal/api/handlers_analytics.go`

Add endpoint for approximate stats:

```go
// ApproximateStats returns approximate statistics using DataSketches
func (h *Handler) ApproximateStats(w http.ResponseWriter, r *http.Request) {
    stats, err := h.db.GetApproximateStats(r.Context())
    if err != nil {
        h.logger.Error("Failed to get approximate stats", "error", err)
        respondError(w, http.StatusInternalServerError, "Failed to retrieve statistics")
        return
    }

    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status": "success",
        "data":   stats,
    })
}
```

#### Step 2.5: Gradual Migration Strategy

Rather than replacing all exact queries, use approximate queries for:

1. **Dashboard Overview** - Quick stats display
2. **Trend Charts** - Historical unique counts
3. **Large Time Ranges** - Queries spanning >90 days
4. **Export Operations** - Pre-flight counts

Keep exact queries for:
1. **User Tables** - Need precise pagination
2. **Drill-down Views** - Detailed analysis
3. **Small Datasets** - <10k records

### Deliverables for Phase 2
- [ ] Extension installation in `database_extensions.go`
- [ ] `analytics_approximate.go` with sketch-based methods
- [ ] Optional pre-computed sketch tables
- [ ] API endpoint for approximate stats
- [ ] Dashboard integration for quick stats
- [ ] Unit tests with performance benchmarks
- [ ] Documentation for when to use approximate vs exact

---

## Phase 3: VSS (Vector Similarity Search) Extension

### Purpose
Enable content-based recommendations ("Users who watched X also watched Y") and similar content discovery.

### Current State
- No recommendation system exists
- Content metadata available (title, genre, year, actors)
- User viewing history tracked in playback_events

### Architecture Decision
VSS requires embeddings. Options:

1. **Pre-computed Embeddings** - Generate embeddings externally, store in DuckDB
2. **Simple Feature Vectors** - Create vectors from metadata (genre, year, duration)
3. **Viewing Pattern Vectors** - Create vectors from user behavior

**Recommendation**: Start with option 2 (metadata vectors), expand to option 1 later.

### Implementation Steps

#### Step 3.1: Extension Installation

**File**: `internal/database/database_extensions.go`

```go
// installVSS installs the Vector Similarity Search extension
func (db *DB) installVSS(optional bool) error {
    ctx, cancel := extensionContext()
    defer cancel()

    if _, err := db.conn.ExecContext(ctx, "INSTALL vss FROM community;"); err != nil {
        if _, loadErr := db.conn.ExecContext(ctx, "LOAD vss;"); loadErr != nil {
            if optional {
                db.vssAvailable = false
                fmt.Printf("Warning: VSS extension unavailable, recommendations will be disabled\n")
                return nil
            }
            return fmt.Errorf("failed to install vss: %w", err)
        }
    }

    if _, err := db.conn.ExecContext(ctx, "LOAD vss;"); err != nil {
        if optional {
            db.vssAvailable = false
            return nil
        }
        return fmt.Errorf("failed to load vss: %w", err)
    }

    return nil
}
```

#### Step 3.2: Content Metadata Vectors Schema

**New File**: `internal/database/schema_embeddings.go`

```go
package database

import "context"

// createEmbeddingsTables creates tables for content embeddings
func (db *DB) createEmbeddingsTables() error {
    if !db.vssAvailable {
        return nil
    }

    // Content embeddings table
    sql := `
        CREATE TABLE IF NOT EXISTS content_embeddings (
            rating_key VARCHAR PRIMARY KEY,
            title VARCHAR NOT NULL,
            media_type VARCHAR,
            year INTEGER,
            genre_vector FLOAT[20],      -- Genre encoding (20 common genres)
            metadata_vector FLOAT[50],   -- Combined metadata features
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );

        -- Create HNSW index for fast similarity search
        CREATE INDEX IF NOT EXISTS idx_content_metadata_vector
        ON content_embeddings
        USING HNSW (metadata_vector)
        WITH (metric = 'cosine');
    `

    _, err := db.conn.ExecContext(context.Background(), sql)
    if err != nil {
        return err
    }

    // User preference embeddings
    sqlUser := `
        CREATE TABLE IF NOT EXISTS user_embeddings (
            user_id VARCHAR PRIMARY KEY,
            username VARCHAR,
            preference_vector FLOAT[50],  -- Aggregated from watched content
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );

        CREATE INDEX IF NOT EXISTS idx_user_preference_vector
        ON user_embeddings
        USING HNSW (preference_vector)
        WITH (metric = 'cosine');
    `

    _, err = db.conn.ExecContext(context.Background(), sqlUser)
    return err
}
```

#### Step 3.3: Embedding Generation

**New File**: `internal/embeddings/generator.go`

```go
package embeddings

import (
    "strings"
)

// GenreList defines the standard genre encoding
var GenreList = []string{
    "action", "adventure", "animation", "comedy", "crime",
    "documentary", "drama", "family", "fantasy", "history",
    "horror", "music", "mystery", "romance", "sci-fi",
    "thriller", "war", "western", "biography", "sport",
}

// GenerateGenreVector creates a one-hot encoded genre vector
func GenerateGenreVector(genres []string) []float32 {
    vector := make([]float32, 20)
    genreLower := make(map[string]bool)
    for _, g := range genres {
        genreLower[strings.ToLower(g)] = true
    }

    for i, genre := range GenreList {
        if genreLower[genre] {
            vector[i] = 1.0
        }
    }

    return vector
}

// GenerateMetadataVector creates a combined feature vector from content metadata
func GenerateMetadataVector(content ContentMetadata) []float32 {
    vector := make([]float32, 50)

    // Indices 0-19: Genre encoding
    genreVec := GenerateGenreVector(content.Genres)
    copy(vector[0:20], genreVec)

    // Indices 20-24: Year encoding (normalized decades)
    if content.Year > 0 {
        decade := (content.Year - 1900) / 10
        if decade >= 0 && decade < 5 {
            vector[20+decade] = 1.0
        }
    }

    // Indices 25-29: Duration buckets (for movies)
    if content.Duration > 0 {
        bucket := content.Duration / 30 // 30-minute buckets
        if bucket < 5 {
            vector[25+bucket] = 1.0
        }
    }

    // Indices 30-34: Rating buckets
    if content.Rating > 0 {
        bucket := int(content.Rating / 2) // 0-2, 2-4, 4-6, 6-8, 8-10
        if bucket < 5 {
            vector[30+bucket] = 1.0
        }
    }

    // Indices 35-39: Content type encoding
    typeMap := map[string]int{"movie": 0, "show": 1, "episode": 2, "album": 3, "track": 4}
    if idx, ok := typeMap[content.MediaType]; ok {
        vector[35+idx] = 1.0
    }

    // Indices 40-49: Reserved for future features

    return vector
}

// ContentMetadata holds metadata for embedding generation
type ContentMetadata struct {
    RatingKey string
    Title     string
    Genres    []string
    Year      int
    Duration  int     // minutes
    Rating    float64 // 0-10
    MediaType string
}
```

#### Step 3.4: Recommendation Engine

**New File**: `internal/database/recommendations.go`

```go
package database

import (
    "context"
    "fmt"
)

// Recommendation represents a content recommendation
type Recommendation struct {
    RatingKey  string  `json:"rating_key"`
    Title      string  `json:"title"`
    MediaType  string  `json:"media_type"`
    Year       int     `json:"year,omitempty"`
    Similarity float64 `json:"similarity"`
    Reason     string  `json:"reason,omitempty"`
}

// GetSimilarContent finds content similar to the given rating key
func (db *DB) GetSimilarContent(ctx context.Context, ratingKey string, limit int) ([]Recommendation, error) {
    if !db.vssAvailable {
        return nil, fmt.Errorf("VSS extension not available")
    }

    sql := `
        WITH target AS (
            SELECT metadata_vector, title, media_type
            FROM content_embeddings
            WHERE rating_key = ?
        )
        SELECT
            ce.rating_key,
            ce.title,
            ce.media_type,
            ce.year,
            1 - array_cosine_distance(ce.metadata_vector, t.metadata_vector) as similarity
        FROM content_embeddings ce, target t
        WHERE ce.rating_key != ?
        ORDER BY array_cosine_distance(ce.metadata_vector, t.metadata_vector) ASC
        LIMIT ?
    `

    rows, err := db.conn.QueryContext(ctx, sql, ratingKey, ratingKey, limit)
    if err != nil {
        return nil, fmt.Errorf("similarity search failed: %w", err)
    }
    defer rows.Close()

    var results []Recommendation
    for rows.Next() {
        var r Recommendation
        var year *int
        if err := rows.Scan(&r.RatingKey, &r.Title, &r.MediaType, &year, &r.Similarity); err != nil {
            return nil, err
        }
        if year != nil {
            r.Year = *year
        }
        r.Reason = "Similar genre and characteristics"
        results = append(results, r)
    }

    return results, nil
}

// GetRecommendationsForUser gets personalized recommendations based on viewing history
func (db *DB) GetRecommendationsForUser(ctx context.Context, userID string, limit int) ([]Recommendation, error) {
    if !db.vssAvailable {
        return db.getFallbackRecommendations(ctx, userID, limit)
    }

    sql := `
        WITH user_pref AS (
            SELECT preference_vector
            FROM user_embeddings
            WHERE user_id = ?
        ),
        watched AS (
            SELECT DISTINCT rating_key
            FROM playback_events
            WHERE user_id = ?
        )
        SELECT
            ce.rating_key,
            ce.title,
            ce.media_type,
            ce.year,
            1 - array_cosine_distance(ce.metadata_vector, up.preference_vector) as similarity
        FROM content_embeddings ce, user_pref up
        WHERE ce.rating_key NOT IN (SELECT rating_key FROM watched)
        ORDER BY array_cosine_distance(ce.metadata_vector, up.preference_vector) ASC
        LIMIT ?
    `

    rows, err := db.conn.QueryContext(ctx, sql, userID, userID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []Recommendation
    for rows.Next() {
        var r Recommendation
        var year *int
        if err := rows.Scan(&r.RatingKey, &r.Title, &r.MediaType, &year, &r.Similarity); err != nil {
            return nil, err
        }
        if year != nil {
            r.Year = *year
        }
        r.Reason = "Based on your viewing history"
        results = append(results, r)
    }

    return results, nil
}

// UpdateUserEmbedding aggregates user viewing history into preference vector
func (db *DB) UpdateUserEmbedding(ctx context.Context, userID string) error {
    if !db.vssAvailable {
        return nil
    }

    sql := `
        INSERT OR REPLACE INTO user_embeddings (user_id, username, preference_vector)
        SELECT
            p.user_id,
            MAX(p.username),
            -- Average of all watched content vectors, weighted by completion
            list_aggregate(
                list_transform(
                    list(ce.metadata_vector),
                    x -> list_transform(x, v -> v * COALESCE(p.percent_complete, 50) / 100)
                ),
                'avg'
            )
        FROM playback_events p
        JOIN content_embeddings ce ON p.rating_key = ce.rating_key
        WHERE p.user_id = ?
        GROUP BY p.user_id
    `

    _, err := db.conn.ExecContext(ctx, sql, userID)
    return err
}

// getFallbackRecommendations provides simple recommendations when VSS unavailable
func (db *DB) getFallbackRecommendations(ctx context.Context, userID string, limit int) ([]Recommendation, error) {
    // Fall back to "most popular content not yet watched by user"
    sql := `
        WITH user_watched AS (
            SELECT DISTINCT rating_key
            FROM playback_events
            WHERE user_id = ?
        ),
        popular AS (
            SELECT
                rating_key,
                title,
                media_type,
                year,
                COUNT(*) as watch_count
            FROM playback_events
            WHERE rating_key NOT IN (SELECT rating_key FROM user_watched)
            GROUP BY rating_key, title, media_type, year
            ORDER BY watch_count DESC
            LIMIT ?
        )
        SELECT rating_key, title, media_type, year,
               1.0 - (ROW_NUMBER() OVER () * 0.05) as similarity
        FROM popular
    `

    rows, err := db.conn.QueryContext(ctx, sql, userID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []Recommendation
    for rows.Next() {
        var r Recommendation
        var year *int
        if err := rows.Scan(&r.RatingKey, &r.Title, &r.MediaType, &year, &r.Similarity); err != nil {
            return nil, err
        }
        if year != nil {
            r.Year = *year
        }
        r.Reason = "Popular with other users"
        results = append(results, r)
    }

    return results, nil
}
```

#### Step 3.5: API Endpoints

**File**: `internal/api/handlers_recommendations.go`

```go
package api

import (
    "net/http"
    "strconv"

    "github.com/go-chi/chi/v5"
)

// SimilarContent returns content similar to the given item
func (h *Handler) SimilarContent(w http.ResponseWriter, r *http.Request) {
    ratingKey := chi.URLParam(r, "ratingKey")
    if ratingKey == "" {
        respondError(w, http.StatusBadRequest, "Rating key required")
        return
    }

    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit <= 0 || limit > 50 {
        limit = 10
    }

    results, err := h.db.GetSimilarContent(r.Context(), ratingKey, limit)
    if err != nil {
        h.logger.Error("Failed to get similar content", "error", err)
        respondError(w, http.StatusInternalServerError, "Failed to find similar content")
        return
    }

    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status":          "success",
        "recommendations": results,
        "count":           len(results),
    })
}

// UserRecommendations returns personalized recommendations for a user
func (h *Handler) UserRecommendations(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "userID")
    if userID == "" {
        respondError(w, http.StatusBadRequest, "User ID required")
        return
    }

    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit <= 0 || limit > 50 {
        limit = 10
    }

    results, err := h.db.GetRecommendationsForUser(r.Context(), userID, limit)
    if err != nil {
        h.logger.Error("Failed to get recommendations", "error", err)
        respondError(w, http.StatusInternalServerError, "Failed to generate recommendations")
        return
    }

    respondJSON(w, http.StatusOK, map[string]interface{}{
        "status":          "success",
        "recommendations": results,
        "count":           len(results),
    })
}
```

**File**: `internal/api/chi_router.go`

Add routes:

```go
r.Route("/recommendations", func(r chi.Router) {
    r.Get("/similar/{ratingKey}", h.SimilarContent)
    r.Get("/user/{userID}", h.UserRecommendations)
})
```

#### Step 3.6: Embedding Sync Job

**New File**: `internal/sync/embedding_sync.go`

```go
package sync

import (
    "context"
    "time"

    "github.com/tomtom215/cartographus/internal/database"
    "github.com/tomtom215/cartographus/internal/embeddings"
)

// EmbeddingSyncManager keeps content and user embeddings up to date
type EmbeddingSyncManager struct {
    db       *database.DB
    interval time.Duration
    stopCh   chan struct{}
}

// NewEmbeddingSyncManager creates a new embedding sync manager
func NewEmbeddingSyncManager(db *database.DB, interval time.Duration) *EmbeddingSyncManager {
    return &EmbeddingSyncManager{
        db:       db,
        interval: interval,
        stopCh:   make(chan struct{}),
    }
}

// Start begins the periodic sync
func (m *EmbeddingSyncManager) Start(ctx context.Context) error {
    ticker := time.NewTicker(m.interval)
    defer ticker.Stop()

    // Initial sync
    if err := m.syncEmbeddings(ctx); err != nil {
        // Log but don't fail - embeddings are optional
        fmt.Printf("Warning: Initial embedding sync failed: %v\n", err)
    }

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-m.stopCh:
            return nil
        case <-ticker.C:
            if err := m.syncEmbeddings(ctx); err != nil {
                fmt.Printf("Warning: Embedding sync failed: %v\n", err)
            }
        }
    }
}

func (m *EmbeddingSyncManager) syncEmbeddings(ctx context.Context) error {
    // Sync content embeddings for new/updated content
    // Sync user embeddings for users with recent activity
    // Implementation details...
    return nil
}
```

#### Step 3.7: Frontend Integration

**New File**: `web/src/app/RecommendationsManager.ts`

```typescript
/**
 * Recommendations Manager
 * Displays personalized content recommendations and similar content
 */

import { API } from '../lib/api';
import type { Recommendation } from '../lib/types';

export class RecommendationsManager {
    private api: API;
    private containerId: string;

    constructor(api: API, containerId: string) {
        this.api = api;
        this.containerId = containerId;
    }

    async loadSimilarContent(ratingKey: string): Promise<void> {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        container.innerHTML = '<div class="loading">Finding similar content...</div>';

        try {
            const response = await this.api.getSimilarContent(ratingKey, 10);
            this.renderRecommendations(container, response.recommendations, 'Similar Content');
        } catch (error) {
            container.innerHTML = '<div class="error">Failed to load recommendations</div>';
        }
    }

    async loadUserRecommendations(userId: string): Promise<void> {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        container.innerHTML = '<div class="loading">Generating recommendations...</div>';

        try {
            const response = await this.api.getUserRecommendations(userId, 10);
            this.renderRecommendations(container, response.recommendations, 'Recommended for You');
        } catch (error) {
            container.innerHTML = '<div class="error">Failed to load recommendations</div>';
        }
    }

    private renderRecommendations(container: HTMLElement, recommendations: Recommendation[], title: string): void {
        if (recommendations.length === 0) {
            container.innerHTML = '<div class="empty">No recommendations available</div>';
            return;
        }

        container.innerHTML = `
            <div class="recommendations-panel">
                <h3>${title}</h3>
                <div class="recommendations-list">
                    ${recommendations.map(r => `
                        <div class="recommendation-item" data-rating-key="${r.rating_key}">
                            <div class="rec-title">${this.escapeHtml(r.title)}</div>
                            <div class="rec-meta">
                                <span class="rec-type">${r.media_type}</span>
                                ${r.year ? `<span class="rec-year">${r.year}</span>` : ''}
                                <span class="rec-score">${Math.round(r.similarity * 100)}% match</span>
                            </div>
                            <div class="rec-reason">${this.escapeHtml(r.reason || '')}</div>
                        </div>
                    `).join('')}
                </div>
            </div>
        `;
    }

    private escapeHtml(text: string): string {
        return text.replace(/[&<>"']/g, (c) => ({
            '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;'
        }[c] || c));
    }
}
```

### Deliverables for Phase 3
- [ ] VSS extension installation
- [ ] Content embeddings schema
- [ ] User embeddings schema
- [ ] Embedding generator utilities
- [ ] Recommendation database methods
- [ ] API endpoints for recommendations
- [ ] Background sync job for embeddings
- [ ] Frontend RecommendationsManager
- [ ] Integration with metadata detail views
- [ ] Unit and integration tests

---

## Testing Strategy

### Unit Tests
Each phase includes unit tests with table-driven patterns:

```go
func TestFuzzySearch(t *testing.T) {
    tests := []struct {
        name     string
        query    string
        minScore int
        want     int
    }{
        // Test cases...
    }
    // ...
}
```

### Integration Tests
Test extension availability detection and fallback behavior:

```go
func TestExtensionFallback(t *testing.T) {
    // Test that features degrade gracefully when extensions unavailable
}
```

### E2E Tests
Add Playwright tests for new UI features:

```typescript
test('fuzzy search finds typo matches', async ({ page }) => {
    await page.fill('#search-input', 'Braking Bad');
    await expect(page.locator('[data-testid="search-result"]')).toContainText('Breaking Bad');
});
```

### Performance Benchmarks
Compare exact vs approximate analytics:

```go
func BenchmarkExactDistinctCount(b *testing.B) { /* ... */ }
func BenchmarkApproximateDistinctCount(b *testing.B) { /* ... */ }
```

---

## Migration Notes

### Backwards Compatibility
- All new extensions are **optional** with graceful fallback
- Existing queries continue to work unchanged
- New features are additive, not replacing existing functionality

### Database Schema Changes
- New tables created only when extensions available
- No migration of existing data required for Phase 1 and 2
- Phase 3 requires embedding generation job to populate vectors

### Configuration
Add new config options:

```yaml
database:
  extensions:
    rapidfuzz: true        # Enable fuzzy search (default: true)
    datasketches: true     # Enable approximate analytics (default: true)
    vss: false             # Enable recommendations (default: false, opt-in)
```

---

## Timeline Considerations

**Phase 1 (RapidFuzz)**: Smallest scope, most immediate value
- Extension installation: straightforward
- Database methods: 1 new file
- API changes: 1 new endpoint
- Frontend: modifications to existing SearchManager

**Phase 2 (DataSketches)**: Medium scope, performance focus
- Extension installation: straightforward
- Database methods: 1 new file + optional schema
- API changes: 1 new endpoint
- Frontend: optional indicator for approximate data

**Phase 3 (VSS)**: Largest scope, new feature
- Extension installation: straightforward
- Database methods: 2 new files (schema + recommendations)
- New package: embeddings generator
- Background job: embedding sync
- API changes: 2 new endpoints
- Frontend: new RecommendationsManager

---

## Appendix: Extension Versions and Compatibility

| Extension | DuckDB Version | Community |
|-----------|----------------|-----------|
| rapidfuzz | 1.4.x+         | Yes       |
| datasketches | 1.4.x+      | Yes       |
| vss       | 1.4.x+         | Yes       |

All extensions compatible with current DuckDB v1.4.2 used by the project.
