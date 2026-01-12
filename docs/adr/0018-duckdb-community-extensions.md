# ADR-0018: DuckDB Community Extensions (RapidFuzz + DataSketches)

**Date**: 2025-12-13
**Status**: Accepted

---

## Context

Following ADR-0001's adoption of DuckDB, an evaluation of DuckDB community extensions was conducted to identify extensions that could provide additional value for the Cartographus application.

### Requirements Evaluated

1. **Fuzzy Search**: Improve search UX by tolerating typos and partial matches
2. **Approximate Analytics**: Reduce query latency for large datasets using probabilistic data structures
3. **Vector Similarity**: Enable content-based recommendations (future consideration)

### Extensions Evaluated

| Extension | Type | Use Case | Verdict |
|-----------|------|----------|---------|
| **rapidfuzz** | Community | Fuzzy string matching | **Adopted** |
| **datasketches** | Community | Approximate distinct counts, percentiles | **Adopted** |
| **vss** | Community | Vector similarity search | Deferred (requires embedding infrastructure) |
| a5 | Community | Analytics (redundant with H3) | Not adopted |
| fuzzycomplete | Community | CLI autocomplete only | Not adopted |
| duckdb-nats-jetstream | Community | NATS integration (redundant) | Not adopted |
| radio | Community | Streaming (not applicable) | Not adopted |
| stochastic | Community | Probabilistic queries | Not adopted |
| duckdb_yaml | Community | YAML parsing (no use case) | Not adopted |

---

## Decision

Adopt **RapidFuzz** and **DataSketches** community extensions with graceful fallback when extensions are unavailable.

### RapidFuzz
- Provides `rapidfuzz_ratio()`, `rapidfuzz_token_set_ratio()` functions
- Enables typo-tolerant search across playback titles, usernames
- Falls back to exact `LIKE` matching when unavailable

### DataSketches
- Provides HyperLogLog (HLL) for approximate distinct counts (~2% error)
- Provides KLL sketches for approximate percentiles
- Falls back to exact `COUNT(DISTINCT)` and `PERCENTILE_CONT` when unavailable

### VSS (Deferred)
- Requires embedding vectors to be useful
- No current embedding generation infrastructure
- Revisit when content recommendation features are planned

---

## Consequences

### Positive

- **Improved Search UX**: Users can find content despite typos ("Braking Bad" finds "Breaking Bad")
- **Faster Dashboard Loads**: Approximate analytics reduce query time for large datasets
- **Graceful Degradation**: Features work even when extensions unavailable (exact calculations)
- **No Breaking Changes**: All new functionality is additive
- **Production Ready**: Extensions enabled by default for production deployments

### Negative

- **Extension Download Required**: Community extensions must be downloaded at startup
- **Increased Build Complexity**: Additional extensions to manage in setup scripts
- **Approximate Results**: DataSketches provides estimates, not exact values (acceptable for dashboards)

### Neutral

- **Extension Size**: ~10-15MB additional disk space for extension binaries
- **Memory Usage**: Minimal overhead for sketch data structures

---

## Implementation

### Extension Installation Pattern

All community extensions follow the same extensionSpec-based installation pattern with fallback:

```go
// internal/database/database_extensions.go
func (db *DB) installRapidFuzz(optional bool) error {
    spec := &extensionSpec{
        Name:              "rapidfuzz",
        Community:         true,
        VerifyQuery:       "SELECT rapidfuzz_ratio('hello', 'helo')",
        AvailabilityField: func(db *DB) *bool { return &db.rapidfuzzAvailable },
        WarningMessage:    "RapidFuzz extension unavailable, fuzzy search will use exact matching",
    }
    return db.installCommunityExtension(spec, optional)
}
```

### Fuzzy Search API

```go
// internal/database/search_fuzzy.go
func (db *DB) FuzzySearchPlaybacks(ctx context.Context, query string, minScore int, limit int) ([]FuzzySearchResult, error) {
    // Validate and set defaults
    if minScore <= 0 || minScore > 100 {
        minScore = 70
    }
    if limit <= 0 || limit > 100 {
        limit = 20
    }

    // Use fuzzy search if RapidFuzz available, otherwise fall back to exact
    if db.rapidfuzzAvailable {
        return db.fuzzySearchPlaybacksWithRapidFuzz(ctx, query, minScore, limit)
    }
    return db.fuzzySearchPlaybacksFallback(ctx, query, limit)
}
```

### Approximate Analytics API

```go
// internal/database/analytics_approximate.go
func (db *DB) GetApproximateStats(ctx context.Context, filter ApproximateStatsFilter) (*ApproximateStats, error) {
    start := time.Now()

    if db.datasketchesAvailable {
        stats, err := db.getApproximateStatsWithSketches(ctx, filter)
        if err == nil {
            stats.QueryTimeMS = time.Since(start).Milliseconds()
            return stats, nil
        }
        // Fall through to exact calculation on error
        logging.Warn().Err(err).Msg("DataSketches query failed, falling back to exact")
    }

    // Fall back to exact calculation
    stats, err := db.getExactStats(ctx, filter)
    if err != nil {
        return nil, err
    }
    stats.QueryTimeMS = time.Since(start).Milliseconds()
    return stats, nil
}
```

### API Endpoints Added

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/search/fuzzy` | Fuzzy content search |
| `GET /api/v1/search/users` | Fuzzy user search |
| `GET /api/v1/analytics/approximate` | Approximate stats (HLL + KLL) |
| `GET /api/v1/analytics/approximate/distinct` | Approximate distinct count |
| `GET /api/v1/analytics/approximate/percentile` | Approximate percentile |

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DUCKDB_SPATIAL_OPTIONAL` | `false` | Allow startup without spatial extensions |
| `DUCKDB_DATASKETCHES_ENABLED` | `false` | Enable DataSketches extension for approximate analytics |

DataSketches is installed but disabled by default. Enable via `DUCKDB_DATASKETCHES_ENABLED=true` to use HyperLogLog and KLL sketches. When extensions are unavailable, the application falls back to exact calculations (COUNT DISTINCT, PERCENTILE_CONT).

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Extension installation | `internal/database/database_extensions.go` | extensionSpec-based patterns |
| RapidFuzz availability | `internal/database/database.go:35` | `rapidfuzzAvailable` field |
| DataSketches availability | `internal/database/database.go:36` | `datasketchesAvailable` field |
| Fuzzy search methods | `internal/database/search_fuzzy.go` | Database methods |
| Fuzzy search tests | `internal/database/search_fuzzy_test.go` | Unit tests |
| Fuzzy search handler | `internal/api/handlers_search_fuzzy.go` | API handler |
| Approximate analytics | `internal/database/analytics_approximate.go` | Database methods |
| Approximate analytics tests | `internal/database/analytics_approximate_test.go` | Unit tests |
| Approximate analytics handler | `internal/api/handlers_analytics_approximate.go` | API handler |
| Router registration | `internal/api/chi_router.go:144-147,167-168` | Route definitions |
| Setup script | `scripts/setup-duckdb-extensions.sh:607-608` | Extension download |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| RapidFuzz fallback works | `internal/database/search_fuzzy_test.go` | Yes |
| DataSketches fallback works | `internal/database/analytics_approximate_test.go` | Yes |
| Extensions load at startup | `internal/database/database_extensions.go` | Yes |
| API endpoints registered | `internal/api/chi_router.go:144-147,167-168` | Yes |
| Setup script downloads extensions | `scripts/setup-duckdb-extensions.sh:607-608` | Yes |

### Test Coverage

- Fuzzy search tests: `internal/database/search_fuzzy_test.go`
- Approximate analytics tests: `internal/database/analytics_approximate_test.go`
- Both test suites verify fallback behavior when extensions unavailable
- Coverage target: 90%+ for new database methods

---

## Related ADRs

- [ADR-0001](0001-use-duckdb-for-analytics.md): Original DuckDB adoption decision
- [ADR-0016](0016-chi-router-adoption.md): Chi router for new API endpoints

---

## References

- [DuckDB Community Extensions](https://community-extensions.duckdb.org/)
- [RapidFuzz Extension](https://community-extensions.duckdb.org/extensions/rapidfuzz.html)
- [DataSketches Extension](https://community-extensions.duckdb.org/extensions/datasketches.html)
- [Apache DataSketches](https://datasketches.apache.org/)
- [HyperLogLog Algorithm](https://en.wikipedia.org/wiki/HyperLogLog)
