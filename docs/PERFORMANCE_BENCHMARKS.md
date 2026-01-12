# Performance Benchmarks

Comprehensive performance testing and optimization guide for Cartographus.

**Related Documentation**:
- [DEVELOPMENT.md](./DEVELOPMENT.md) - Development workflow
- [ARCHITECTURE.md](./ARCHITECTURE.md) - System architecture
- [CONFIGURATION_REFERENCE.md](./CONFIGURATION_REFERENCE.md) - Tuning parameters

---

## Table of Contents

1. [Overview](#overview)
2. [Running Benchmarks](#running-benchmarks)
3. [Benchmark Categories](#benchmark-categories)
4. [Performance Baselines](#performance-baselines)
5. [Optimization Techniques](#optimization-techniques)
6. [Profiling](#profiling)
7. [Production Monitoring](#production-monitoring)

---

## Overview

Cartographus includes **191 benchmark tests** covering:

- Database operations (DuckDB queries, spatial operations)
- API handler performance
- Caching layer efficiency
- Event processing throughput
- WebSocket messaging
- Authentication and authorization

### Test Environment

Benchmarks are designed to run on:
- **Minimum**: 4 cores, 8GB RAM
- **Recommended**: 8+ cores, 32GB RAM
- **CI Environment**: Self-hosted runner (8 cores, 32GB RAM)

---

## Running Benchmarks

### All Benchmarks

```bash
# Set up environment first
export GOTOOLCHAIN=local
export no_proxy="localhost,127.0.0.1"
export NO_PROXY="localhost,127.0.0.1"

# Run all benchmarks
go test -bench=. -benchmem ./...

# Run with specific iterations
go test -bench=. -benchmem -count=5 ./...

# Generate benchstat-compatible output
go test -bench=. -benchmem -count=10 ./... | tee benchmark.txt
```

### Specific Package Benchmarks

```bash
# Database benchmarks
go test -bench=. -benchmem ./internal/database/...

# API handler benchmarks
go test -bench=. -benchmem ./internal/api/...

# Cache benchmarks
go test -bench=. -benchmem ./internal/cache/...

# Spatial benchmarks (requires DuckDB extensions)
go test -bench=. -benchmem ./internal/database/ -run=NONE -bench=Spatial

# Event processor benchmarks (requires NATS build tag)
go test -tags nats -bench=. -benchmem ./internal/eventprocessor/...
```

### Stress Tests

```bash
# Run stress tests (skipped in short mode)
go test -v -run=TestStress ./internal/eventprocessor/...

# Run with race detector
go test -race -v -run=TestStress ./internal/eventprocessor/...
```

### Comparing Benchmarks

```bash
# Before optimization
go test -bench=. -benchmem -count=10 ./internal/database/ > before.txt

# After optimization
go test -bench=. -benchmem -count=10 ./internal/database/ > after.txt

# Compare with benchstat
benchstat before.txt after.txt
```

---

## Benchmark Categories

### Database Benchmarks

Located in `internal/database/`:

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `BenchmarkGetStats` | Overall statistics query | <5ms |
| `BenchmarkGetPlaybackTrends` | Time-series trend analysis | <10ms |
| `BenchmarkGetMediaTypeDistribution` | Media type breakdown | <5ms |
| `BenchmarkGetTopCities` | Geographic aggregation | <15ms |
| `BenchmarkGetLibraryStats` | Per-library statistics | <10ms |
| `BenchmarkGetDurationStats` | Duration analytics | <5ms |
| `BenchmarkParallelQueries` | Concurrent query execution | <20ms |

**Spatial Benchmarks** (`internal/database/spatial_bench_test.go`):

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `BenchmarkGetH3AggregatedHexagons` | H3 hexagon aggregation | <50ms |
| `BenchmarkGetDistanceWeightedArcs` | Arc distance calculations | <30ms |
| `BenchmarkGetLocationsInViewport` | Viewport queries | <20ms |
| `BenchmarkGetTemporalSpatialDensity` | Time-space density | <100ms |
| `BenchmarkGetNearbyLocations` | Proximity search | <15ms |
| `BenchmarkSpatialIndexPerformance` | Spatial index benefit | <30ms |

### API Handler Benchmarks

Located in `internal/api/`:

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `BenchmarkRouterSetup` | Router initialization | <1ms |
| `BenchmarkRouterHandleRequest` | Request routing | <0.1ms |
| `BenchmarkStats_WithDB` | Full stats endpoint | <10ms |
| `BenchmarkPlaybacks_WithDB` | Playbacks list | <15ms |
| `BenchmarkRespondJSON_LargePayload` | JSON serialization | <5ms |
| `BenchmarkEncodeCursor` | Cursor pagination encoding | <0.01ms |
| `BenchmarkDecodeCursor` | Cursor pagination decoding | <0.01ms |
| `BenchmarkCheckCacheAndReturnIfHit` | Cache hit path | <0.05ms |

### Cache Benchmarks

Located in `internal/cache/`:

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `BenchmarkCacheSet` | Cache write | <0.1ms |
| `BenchmarkCacheGet` | Cache read | <0.05ms |
| `BenchmarkCacheCleanup` | Cache eviction | <1ms |
| `BenchmarkLFUCache_Set` | LFU cache write | <0.1ms |
| `BenchmarkLFUCache_Get` | LFU cache read | <0.05ms |
| `BenchmarkLRUCache_Add` | LRU cache add | <0.05ms |
| `BenchmarkBloomFilter_Add` | Bloom filter insert | <0.01ms |
| `BenchmarkBloomFilter_Test` | Bloom filter lookup | <0.01ms |
| `BenchmarkTrie_Autocomplete` | Autocomplete search | <0.5ms |

### Event Processor Benchmarks

Located in `internal/eventprocessor/` (requires `-tags nats`):

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `BenchmarkAppender_Append` | Single event append | <0.1ms |
| `BenchmarkAppender_Concurrent` | Concurrent appends | <0.2ms |
| `BenchmarkDuckDBStore_InsertMediaEvents` | Batch insert | <10ms |
| `BenchmarkIntegration_Pipeline` | Full pipeline | <50ms |
| `BenchmarkStress_Throughput` | Maximum throughput | >100K events/sec |

### Stress Tests

| Test | Description | Verification |
|------|-------------|--------------|
| `TestStress_HighMessageVolume` | 10,000+ events | >1,000 events/sec |
| `TestStress_ConcurrentConsumerGroups` | 10 groups x 1,000 events | No race conditions |
| `TestStress_Backpressure` | Slow store simulation | Graceful backpressure |
| `TestStress_BurstTraffic` | 5 bursts x 200 events | Buffer handling |
| `TestStress_LongRunning` | Extended timer flushes | No memory leaks |
| `TestStress_RaceConditions` | Concurrent writes/reads | Race-free |

### WebSocket Benchmarks

Located in `internal/websocket/`:

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `BenchmarkHub_BroadcastJSON` | Broadcast to clients | <1ms |
| `BenchmarkHub_RegisterUnregister` | Client registration | <0.1ms |
| `BenchmarkClient_SendMessage` | Single message send | <0.05ms |

### Auth Benchmarks

Located in `internal/auth/` and `internal/authz/`:

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `BenchmarkTokenEncryptor_Encrypt` | Token encryption | <0.5ms |
| `BenchmarkTokenEncryptor_Decrypt` | Token decryption | <0.5ms |
| `BenchmarkOIDCAuthenticate` | OIDC auth flow | <5ms |
| `BenchmarkCache_Set` | AuthZ cache set | <0.1ms |
| `BenchmarkCache_Get` | AuthZ cache get | <0.05ms |

---

## Performance Baselines

### Database Query Performance

With 100,000 playback records and 10,000 geolocations:

| Query Type | P50 | P95 | P99 |
|------------|-----|-----|-----|
| GetStats | 2ms | 5ms | 10ms |
| GetPlaybackTrends (30 days) | 8ms | 15ms | 25ms |
| GetGeographic (full) | 15ms | 30ms | 50ms |
| GetTopCities (10) | 5ms | 10ms | 20ms |
| H3 Aggregation (res 7) | 20ms | 40ms | 80ms |
| Viewport Query (NYC) | 8ms | 15ms | 30ms |

### API Response Times

| Endpoint | P50 | P95 | P99 |
|----------|-----|-----|-----|
| `/api/v1/health/live` | 0.2ms | 0.5ms | 1ms |
| `/api/v1/stats` | 5ms | 15ms | 30ms |
| `/api/v1/playbacks?limit=100` | 10ms | 25ms | 50ms |
| `/api/v1/analytics/trends` | 15ms | 40ms | 80ms |
| `/api/v1/analytics/geographic` | 25ms | 60ms | 100ms |
| `/api/v1/spatial/hexagons` | 30ms | 70ms | 120ms |

### Throughput Targets

| Component | Minimum | Expected | Optimal |
|-----------|---------|----------|---------|
| API requests/sec | 500 | 1,000 | 5,000 |
| Events/sec (ingestion) | 1,000 | 10,000 | 100,000 |
| WebSocket broadcasts/sec | 100 | 500 | 1,000 |
| Cache operations/sec | 10,000 | 50,000 | 100,000 |

---

## Optimization Techniques

### DuckDB Optimizations

1. **Memory Configuration**
   ```bash
   # Increase memory limit for large datasets
   DATABASE_MAX_MEMORY=4GB
   ```

2. **Connection Pooling**
   ```go
   // Connection pool is managed internally by DuckDB
   // Single writer, multiple readers
   ```

3. **Query Patterns**
   - Use parameterized queries (avoid string interpolation)
   - Leverage DuckDB's columnar storage
   - Use appropriate indexes (spatial, H3)

### Cache Optimization

1. **TTL Configuration**
   ```bash
   CACHE_TTL=5m           # Analytics cache TTL
   CACHE_MAX_SIZE=10000   # Maximum cache entries
   ```

2. **Cache Strategy**
   - LRU for frequently accessed data
   - LFU for hot-spot detection
   - Bloom filter for deduplication

### API Optimization

1. **Response Compression**
   ```bash
   COMPRESSION_ENABLED=true
   COMPRESSION_LEVEL=5    # Balance speed/size
   ```

2. **Rate Limiting**
   ```bash
   RATE_LIMIT_DEFAULT=100/min
   RATE_LIMIT_ANALYTICS=1000/min
   RATE_LIMIT_AUTH=5/min
   ```

3. **Pagination**
   - Use cursor-based pagination for large result sets
   - Limit default page size

### Event Processing

1. **Batch Configuration**
   ```bash
   EVENT_BATCH_SIZE=1000
   EVENT_FLUSH_INTERVAL=1s
   ```

2. **Backpressure Handling**
   - Buffered channels for event queue
   - Graceful degradation under load

---

## Profiling

### CPU Profiling

```bash
# Generate CPU profile during benchmarks
go test -bench=BenchmarkGetStats -cpuprofile=cpu.prof ./internal/database/

# Analyze profile
go tool pprof -http=:8080 cpu.prof

# Generate flame graph (requires graphviz)
go tool pprof -svg cpu.prof > cpu.svg
```

### Memory Profiling

```bash
# Generate memory profile
go test -bench=BenchmarkGetStats -memprofile=mem.prof ./internal/database/

# Analyze allocations
go tool pprof -alloc_space mem.prof

# Show top allocators
go tool pprof -top mem.prof
```

### Trace Profiling

```bash
# Generate execution trace
go test -bench=BenchmarkGetStats -trace=trace.out ./internal/database/

# View trace in browser
go tool trace trace.out
```

### Continuous Profiling

```go
// Runtime profiling endpoints (debug builds)
import _ "net/http/pprof"

// Access at http://localhost:3857/debug/pprof/
```

---

## Production Monitoring

### Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `cartographus_http_request_duration_seconds` | Histogram | Request latency |
| `cartographus_http_requests_total` | Counter | Total requests |
| `cartographus_db_query_duration_seconds` | Histogram | Query latency |
| `cartographus_db_connections_active` | Gauge | Active DB connections |
| `cartographus_cache_hits_total` | Counter | Cache hits |
| `cartographus_cache_misses_total` | Counter | Cache misses |
| `cartographus_events_processed_total` | Counter | Events processed |

### Grafana Dashboard Queries

```promql
# Request latency P95
histogram_quantile(0.95,
  sum(rate(cartographus_http_request_duration_seconds_bucket[5m])) by (le, handler)
)

# Requests per second
sum(rate(cartographus_http_requests_total[1m])) by (handler)

# Cache hit ratio
sum(rate(cartographus_cache_hits_total[5m])) /
(sum(rate(cartographus_cache_hits_total[5m])) + sum(rate(cartographus_cache_misses_total[5m])))

# Database query latency
histogram_quantile(0.99,
  sum(rate(cartographus_db_query_duration_seconds_bucket[5m])) by (le, query_type)
)
```

### Performance Alerts

```yaml
groups:
  - name: cartographus-performance
    rules:
      - alert: HighLatencyP95
        expr: histogram_quantile(0.95, sum(rate(cartographus_http_request_duration_seconds_bucket[5m])) by (le)) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API P95 latency above 500ms"

      - alert: LowCacheHitRate
        expr: sum(rate(cartographus_cache_hits_total[5m])) / (sum(rate(cartographus_cache_hits_total[5m])) + sum(rate(cartographus_cache_misses_total[5m]))) < 0.8
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Cache hit rate below 80%"

      - alert: HighQueryLatency
        expr: histogram_quantile(0.99, sum(rate(cartographus_db_query_duration_seconds_bucket[5m])) by (le)) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database P99 latency above 1 second"
```

---

## Benchmark Reference

### Running Specific Benchmarks

```bash
# Single benchmark
go test -bench=BenchmarkGetStats -benchmem ./internal/database/

# Pattern matching
go test -bench=BenchmarkCache.* -benchmem ./internal/cache/

# Exclude benchmarks
go test -bench=. -benchmem -run=NONE ./...

# With timeout
go test -bench=. -benchmem -timeout=30m ./...
```

### Interpreting Results

```
BenchmarkGetStats-8    12345    96543 ns/op    12345 B/op    123 allocs/op
```

- `BenchmarkGetStats-8`: Name and GOMAXPROCS
- `12345`: Iterations run
- `96543 ns/op`: Time per operation (96.5us)
- `12345 B/op`: Bytes allocated per operation
- `123 allocs/op`: Allocations per operation

### Performance Regression Testing

```bash
# CI workflow integration
# Compare against main branch baseline
benchstat main.txt pr.txt

# Exit with error if regression detected
benchstat -delta-test=ttest main.txt pr.txt | grep -q "+.*%" && exit 1
```

---

## Best Practices

1. **Run benchmarks on quiet system** - Avoid running other workloads
2. **Use -count flag** - Multiple iterations for stable results
3. **Disable CPU scaling** - `sudo cpupower frequency-set -g performance`
4. **Clear cache between runs** - For accurate cache benchmarks
5. **Profile hotspots** - Focus optimization on measured bottlenecks
6. **Track over time** - Monitor performance trends in CI
