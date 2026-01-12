# ADR-0001: Use DuckDB for Analytics Database

**Date**: 2025-11-16
**Status**: Accepted

---

## Context

Cartographus requires a database to store and analyze Plex media server playback events with the following requirements:

1. **Spatial Queries**: Geographic aggregation for map visualization (clustering, distance calculations)
2. **Time-Series Analytics**: Trend analysis, hourly/daily aggregations, percentile calculations
3. **OLAP Workloads**: Complex aggregations across 100+ columns for 47 analytics charts
4. **Embedded Operation**: Single-binary deployment without external database dependencies
5. **Cross-Platform**: Support for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
6. **H3 Hexagonal Indexing**: Multi-resolution geographic aggregation for density visualization

### Alternatives Considered

| Database | Pros | Cons |
|----------|------|------|
| **SQLite** | Embedded, mature, portable | No spatial extension, slow OLAP, no PERCENTILE_CONT |
| **PostgreSQL + PostGIS** | Full-featured spatial, mature | External service, operational overhead |
| **ClickHouse** | Excellent OLAP | External service, no embedded mode |
| **DuckDB** | Embedded, OLAP-optimized, spatial extension, H3 support | Younger project, CGO required |

---

## Decision

Use **DuckDB** as the primary database with the following extensions:

- **spatial** (core): GEOMETRY types, ST_* functions, R-tree spatial indexes
- **h3** (community): Hexagonal hierarchical geospatial indexing (resolutions 6-8)
- **inet** (core): IP address handling
- **icu** (core): Internationalization support

### Key Factors

1. **Native OLAP Performance**: Columnar storage optimized for aggregation queries
2. **Embedded Architecture**: In-process database eliminates network latency
3. **Spatial Extension**: First-class spatial support comparable to PostGIS
4. **H3 Extension**: Uber's H3 hexagonal indexing for efficient geographic aggregation
5. **Modern SQL**: PERCENTILE_CONT, window functions, CTEs, lateral joins
6. **Zero Configuration**: No external services to manage

---

## Consequences

### Positive

- **Sub-30ms Query Performance**: P95 latency achieved for complex analytics queries
- **Single Binary Deployment**: No database setup required for users
- **Rich Spatial Capabilities**: Geographic clustering, distance calculations, bounding box queries
- **H3 Multi-Resolution**: Efficient hexagonal aggregation at multiple zoom levels
- **Parallel Query Execution**: Automatic parallelization with `runtime.NumCPU()` threads

### Negative

- **CGO Required**: DuckDB Go driver requires CGO, complicating cross-compilation
- **ARMv7 Not Supported**: DuckDB lacks pre-compiled binaries for 32-bit ARM
- **Memory Usage**: OLAP queries can consume significant memory (mitigated via DUCKDB_MAX_MEMORY)
- **Extension Loading**: Extensions must be pre-installed or downloaded at startup

### Neutral

- **Different Syntax from SQLite**: Teams familiar with SQLite need to learn DuckDB-specific SQL
- **File-Based Storage**: Single .duckdb file (suitable for this use case)

---

## Implementation

### Database Schema

```sql
-- Main playback events table (170+ columns)
CREATE TABLE playback_events (
    id UUID PRIMARY KEY,
    session_key TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    user_id INTEGER NOT NULL,
    ip_address TEXT NOT NULL,
    correlation_key TEXT UNIQUE,
    -- ... media metadata, streaming quality, technical details
);

-- Geographic locations with spatial data
-- Base table created in database_schema.go
CREATE TABLE geolocations (
    ip_address TEXT PRIMARY KEY,
    latitude DOUBLE NOT NULL,
    longitude DOUBLE NOT NULL,
    geom GEOMETRY NOT NULL,
    city TEXT,
    region TEXT,
    country TEXT NOT NULL,
    -- ...
);

-- H3 columns added via ALTER TABLE in spatial_optimization.go
ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS h3_index_6 UBIGINT;  -- ~36 km^2 hexagons
ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS h3_index_7 UBIGINT;  -- ~5 km^2 hexagons
ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS h3_index_8 UBIGINT;  -- ~0.74 km^2 hexagons

-- R-tree spatial index for geographic queries
CREATE INDEX IF NOT EXISTS idx_geolocation_spatial ON geolocations USING RTREE (geom);

-- H3 indexes for multi-resolution aggregation
CREATE INDEX IF NOT EXISTS idx_geolocation_h3_6 ON geolocations(h3_index_6);
CREATE INDEX IF NOT EXISTS idx_geolocation_h3_7 ON geolocations(h3_index_7);
CREATE INDEX IF NOT EXISTS idx_geolocation_h3_8 ON geolocations(h3_index_8);
```

### Extension Installation

```go
// internal/database/database_extensions.go
func (db *DB) installExtensions() error {
    // Core extensions (required unless DUCKDB_SPATIAL_OPTIONAL=true)
    coreExtensions := []extensionInstaller{
        db.installSpatial,
        db.installH3,
        db.installInet,
        db.installICU,
        db.installJSON,
    }
    for _, installer := range coreExtensions {
        if err := installExtension(installer, spatialOptional); err != nil {
            return err
        }
    }
    // Community extensions loaded if locally available
    // ...
    return nil
}

// internal/database/database_extensions.go - H3 installation
func (db *DB) installH3(optional bool) error {
    spec := &extensionSpec{
        Name:             "h3",
        Community:        true,
        DependsOnSpatial: true,
        VerifyQuery:      "SELECT h3_latlng_to_cell(0.0, 0.0, 0)",
        WarningMessage:   "H3 extension unavailable, H3 indexing will be disabled",
    }
    return db.installCommunityExtension(spec, optional)
}
```

### Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `DUCKDB_PATH` | `/data/cartographus.duckdb` | Database file path |
| `DUCKDB_MAX_MEMORY` | `2GB` | Maximum memory for queries |
| `DUCKDB_THREADS` | `runtime.NumCPU()` | Parallel query threads |

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Database initialization | `internal/database/database.go` | Core DB lifecycle |
| Extension loading | `internal/database/database_extensions.go` | Extension installation logic |
| Extension core logic | `internal/database/database_extensions_core.go` | Table-driven extension installer |
| Schema creation | `internal/database/database_schema.go` | Table definitions |
| Spatial optimization | `internal/database/spatial_optimization.go` | H3 columns, R-tree index, distance calculations |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| DuckDB v1.4.3 (engine) | `internal/database/database_extensions.go:88` | Yes |
| Go driver v2.5.4 | `go.mod:11` | Yes |
| Spatial extension | `internal/database/database_extensions.go:335-342` | Yes |
| H3 community extension | `internal/database/database_extensions.go:345-354` | Yes |
| GEOMETRY type | `internal/database/database_schema.go:347` | Yes |
| R-tree spatial index | `internal/database/spatial_optimization.go:47` | Yes |
| H3 column definitions | `internal/database/spatial_optimization.go:21-25` | Yes |
| Sub-30ms P95 latency | `ARCHITECTURE.md:189` | Yes |

### Test Coverage

- Database tests: `internal/database/*_test.go` (54 test files)
- Spatial tests: `internal/database/spatial_test.go`
- Spatial optimization tests: `internal/database/spatial_optimization_test.go`
- Benchmark tests: `internal/database/database_bench_test.go`
- Coverage target: 90%+ for database package

---

## Related ADRs

- [ADR-0005](0005-nats-jetstream-event-processing.md): Event ingestion into DuckDB
- [ADR-0007](0007-event-sourcing-architecture.md): DuckDB as materialized view

---

## References

- [DuckDB Documentation](https://duckdb.org/docs/)
- [DuckDB Spatial Extension](https://duckdb.org/docs/extensions/spatial.html)
- [H3 Hexagonal Hierarchical Index](https://h3geo.org/)
- [DuckDB Go Driver](https://github.com/duckdb/duckdb-go)
