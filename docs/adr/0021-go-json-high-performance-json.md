# ADR-0021: High-Performance JSON with goccy/go-json

**Date**: 2025-12-18
**Status**: Accepted

---

## Context

Cartographus makes extensive use of JSON encoding/decoding throughout the application:

| Component | Usage |
|-----------|-------|
| API Handlers | 302 endpoints returning JSON responses |
| Event Processing | NATS JetStream message serialization |
| WebSocket | Real-time message broadcasting |
| Media Server Sync | Plex, Jellyfin, Emby API communication |
| Detection Engine | Configuration via `json.RawMessage` |
| Session Storage | BadgerDB session serialization |

A codebase audit identified **169 files** importing `encoding/json` with heavy use of:
- `json.Marshal` / `json.Unmarshal`
- `json.NewEncoder` / `json.NewDecoder`
- `json.RawMessage` for dynamic configuration
- Custom `MarshalJSON` / `UnmarshalJSON` methods

### Libraries Evaluated

| Library | Performance | Compatibility | CGO | Go 1.24 Support |
|---------|-------------|---------------|-----|-----------------|
| encoding/json (stdlib) | Baseline | N/A | No | Yes |
| **goccy/go-json** | ~2-3x faster | Full drop-in | No | Yes |
| bytedance/sonic | ~2.5-5x faster | Partial | Yes | **Workaround required** |
| json-iterator/go | ~2x faster | Partial | No | Yes |
| easyjson | ~3-5x faster | None (codegen) | No | Yes |

### Why goccy/go-json

1. **True Drop-in Replacement**: Same API as `encoding/json` - only import path changes
2. **No CGO Required**: Pure Go with `unsafe` optimizations (unlike Sonic)
3. **No Code Generation**: Unlike easyjson, no build step required
4. **Full Compatibility**: Supports `RawMessage`, custom marshalers, struct tags
5. **No Go Version Issues**: Works with Go 1.24+ without workarounds
6. **Production Proven**: Used by major projects including CockroachDB

### Why Not Sonic

While Sonic offers higher performance (2.5-5x), it was rejected due to:
- CGO requirement adds build complexity alongside DuckDB's CGO
- Go 1.24 compatibility requires `--ldflags="-checklinkname=0"` workaround
- Partial API compatibility (HTML escaping disabled by default)

---

## Decision

Replace `encoding/json` with `github.com/goccy/go-json` across the entire codebase.

### Implementation

Simple import replacement:
```go
// Before
import "encoding/json"

// After
import "github.com/goccy/go-json"
```

No other code changes required - all existing code works unchanged.

---

## Consequences

### Positive

- **~2-3x Faster JSON Operations**: Reduced latency for API responses, event processing
- **Zero Code Changes**: Only import statements modified
- **No Build Complexity**: Pure Go, no CGO, no code generation
- **Future-Proof**: Active maintenance, full stdlib compatibility
- **Lower Memory Allocations**: More efficient memory usage during marshaling

### Negative

- **Additional Dependency**: One more external package to track
- **Subtle Behavior Differences**: Extremely rare edge cases may differ from stdlib

### Neutral

- **No API Changes**: All existing code patterns continue to work
- **No Configuration Required**: Works identically to stdlib

---

## Implementation Details

### Files Modified

- **169 Go files**: Import statement updated from `encoding/json` to `github.com/goccy/go-json`
- **go.mod**: Added `github.com/goccy/go-json v0.10.5`

### Verification

The following patterns were verified to work unchanged:
- `json.Marshal()` / `json.Unmarshal()`
- `json.NewEncoder()` / `json.NewDecoder()`
- `json.RawMessage` type
- Custom `MarshalJSON()` / `UnmarshalJSON()` methods
- All struct tags (`json:"name,omitempty"`)

### Rollback Plan

If issues arise, revert by:
```bash
find . -name "*.go" -exec sed -i 's|"github.com/goccy/go-json"|"encoding/json"|g' {} \;
# Remove from go.mod
go mod tidy
```

---

## Performance Benchmarks

From goccy/go-json documentation (medium-sized JSON ~13KB):

| Operation | encoding/json | goccy/go-json | Improvement |
|-----------|---------------|---------------|-------------|
| Marshal | 792 MB/s | 2,079 MB/s | 2.6x |
| Unmarshal | 117 MB/s | 400 MB/s | 3.4x |
| Parallel Marshal | 3,486 MB/s | 8,810 MB/s | 2.5x |

---

## References

- [goccy/go-json GitHub](https://github.com/goccy/go-json)
- [Sonic](https://github.com/bytedance/sonic) - Rejected due to CGO requirement and Go 1.24+ compatibility issues
- [ADR-0001: DuckDB for Analytics](./0001-use-duckdb-for-analytics.md)
