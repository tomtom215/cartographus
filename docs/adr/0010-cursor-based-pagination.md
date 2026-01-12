# ADR-0010: Cursor-Based Pagination for API

**Date**: 2025-11-25
**Status**: Accepted

---

## Context

Cartographus API returns potentially large result sets (10,000+ playback events). Traditional offset-based pagination has issues:

1. **O(N) Performance**: `OFFSET 10000` still scans 10,000 rows
2. **Data Drift**: Results shift as new data is inserted
3. **Inconsistent Pages**: Same record can appear on multiple pages
4. **Memory Usage**: Database must materialize skipped rows

### Pagination Comparison

| Approach | Time Complexity | Consistency | Use Case |
|----------|-----------------|-------------|----------|
| **Offset-Based** | O(N) | Inconsistent | Small datasets |
| **Cursor-Based** | O(1) | Consistent | Large datasets |
| **Keyset** | O(1) | Consistent | Sorted data |

---

## Decision

Implement **cursor-based pagination** for all paginated API endpoints using encoded cursors:

```
/api/v1/playbacks?cursor=eyJzdGFydGVkX2F0IjoiMjAyNS0xMi0wMVQxMjowMDowMFoiLCJpZCI6ImFiYzEyMyJ9
```

Cursor encodes: `{"started_at": "2025-12-01T12:00:00Z", "id": "abc123"}` (timestamp + ID)

### Key Factors

1. **O(1) Performance**: Uses indexed seek instead of skip
2. **Stable Results**: No data drift between pages
3. **Opaque Token**: Clients don't need to understand cursor structure
4. **Bi-Directional**: Supports forward and backward navigation

---

## Consequences

### Positive

- **Consistent Performance**: 1M+ rows paginate in O(1)
- **Stable Pages**: No duplicate or missing records
- **Scalable**: Performance independent of page number
- **Client Simplicity**: Just pass cursor token

### Negative

- **No Random Access**: Can't jump to page 50 directly
- **Cursor Expiration**: Old cursors may become invalid
- **Implementation Complexity**: More complex than offset

### Neutral

- **Backward Compatibility**: Offset still supported for backward compatibility
- **Total Count**: Expensive, returned only on first page

---

## Implementation

### Cursor Structure

```go
// internal/models/api_responses.go
type PlaybackCursor struct {
    StartedAt time.Time `json:"started_at"` // Timestamp of last playback event
    ID        string    `json:"id"`         // UUID for tie-breaking
}

// internal/api/handlers_core.go
func encodeCursor(cursor *models.PlaybackCursor) string {
    data, err := json.Marshal(cursor)
    if err != nil {
        return ""
    }
    return base64.URLEncoding.EncodeToString(data)
}

func decodeCursor(encoded string) (*models.PlaybackCursor, error) {
    data, err := base64.URLEncoding.DecodeString(encoded)
    if err != nil {
        return nil, fmt.Errorf("invalid base64 encoding: %w", err)
    }
    var cursor models.PlaybackCursor
    if err := json.Unmarshal(data, &cursor); err != nil {
        return nil, fmt.Errorf("invalid cursor JSON: %w", err)
    }
    return &cursor, nil
}
```

### Database Query

```go
// internal/database/crud_playback.go
func (db *DB) GetPlaybackEventsWithCursor(ctx context.Context, limit int, cursor *models.PlaybackCursor) ([]models.PlaybackEvent, *models.PlaybackCursor, bool, error) {
    var query string
    var args []interface{}
    fetchLimit := limit + 1  // Fetch one extra to check for next page

    if cursor == nil {
        // First page - no cursor
        query = `
        SELECT id, session_key, started_at, stopped_at, user_id, username, ...
        FROM playback_events
        ORDER BY started_at DESC, id DESC
        LIMIT ?`
        args = []interface{}{fetchLimit}
    } else {
        // Subsequent page - use cursor for efficient seeking
        query = `
        SELECT id, session_key, started_at, stopped_at, user_id, username, ...
        FROM playback_events
        WHERE (started_at, id) < (?, CAST(? AS UUID))
        ORDER BY started_at DESC, id DESC
        LIMIT ?`
        args = []interface{}{cursor.StartedAt, cursor.ID, fetchLimit}
    }

    rows, err := db.conn.QueryContext(ctx, query, args...)
    // ... scan rows ...

    // Determine if there are more results
    hasMore := len(events) > limit
    if hasMore {
        events = events[:limit]
    }

    // Build next cursor from last item
    var nextCursor *models.PlaybackCursor
    if hasMore && len(events) > 0 {
        lastEvent := events[len(events)-1]
        nextCursor = &models.PlaybackCursor{
            StartedAt: lastEvent.StartedAt,
            ID:        lastEvent.ID.String(),
        }
    }

    return events, nextCursor, hasMore, nil
}
```

### API Response

```go
// internal/models/api_responses.go
type PlaybacksResponse struct {
    Events     []PlaybackEvent `json:"events"`
    Pagination PaginationInfo  `json:"pagination"`
}

type PaginationInfo struct {
    Limit      int     `json:"limit"`
    HasMore    bool    `json:"has_more"`
    NextCursor *string `json:"next_cursor,omitempty"`
    PrevCursor *string `json:"prev_cursor,omitempty"`
    TotalCount *int    `json:"total_count,omitempty"` // Optional, expensive for large datasets
}

// internal/api/handlers_core.go
func (h *Handler) Playbacks(w http.ResponseWriter, r *http.Request) {
    params, err := h.parsePlaybacksParams(r)
    if err != nil {
        respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), err)
        return
    }

    if params.cursorParam != "" {
        h.handleCursorPagination(w, r, params.limit, params.cursor, start)
        return
    }
    // ... offset pagination fallback ...
}

func (h *Handler) handleCursorPagination(w http.ResponseWriter, r *http.Request, limit int, cursor *models.PlaybackCursor, start time.Time) {
    events, nextCursor, hasMore, err := h.db.GetPlaybackEventsWithCursor(r.Context(), limit, cursor)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve playback events", err)
        return
    }

    response := buildPlaybacksResponse(events, limit, hasMore, nextCursor)
    h.respondWithPlaybacks(w, response, start)
}

func buildPlaybacksResponse(events []models.PlaybackEvent, limit int, hasMore bool, nextCursor *models.PlaybackCursor) models.PlaybacksResponse {
    var nextCursorStr *string
    if nextCursor != nil {
        encoded := encodeCursor(nextCursor)
        nextCursorStr = &encoded
    }

    return models.PlaybacksResponse{
        Events: events,
        Pagination: models.PaginationInfo{
            Limit:      limit,
            HasMore:    hasMore,
            NextCursor: nextCursorStr,
        },
    }
}
```

### DuckDB Optimization

```sql
-- Primary index for time-ordered queries (internal/database/database_schema.go)
CREATE INDEX IF NOT EXISTS idx_playback_started_at ON playback_events(started_at DESC);

-- Query uses tuple comparison for efficient cursor-based seek
-- DuckDB optimizes (started_at, id) < (?, ?) with the started_at index
SELECT id, session_key, started_at, stopped_at, ...
FROM playback_events
WHERE (started_at, id) < (?, CAST(? AS UUID))
ORDER BY started_at DESC, id DESC
LIMIT 51;  -- Fetch limit+1 to detect hasMore
```

Note: DuckDB uses the `idx_playback_started_at` index for the ORDER BY clause. The tuple comparison `(started_at, id) < (?, ?)` enables efficient keyset pagination without requiring a compound index on both columns.

### API Examples

```bash
# First page
curl "/api/v1/playbacks?limit=50"
{
  "status": "success",
  "data": {
    "events": [...],
    "pagination": {
      "limit": 50,
      "has_more": true,
      "next_cursor": "eyJzdGFydGVkX2F0IjoiMjAyNS0xMi0wMVQxMjowMDowMFoiLCJpZCI6ImFiYzEyMyJ9"
    }
  },
  "metadata": {
    "timestamp": "2025-12-01T12:00:00Z",
    "query_time_ms": 15
  }
}

# Next page
curl "/api/v1/playbacks?limit=50&cursor=eyJzdGFydGVkX2F0IjoiMjAyNS0xMi0wMVQxMjowMDowMFoiLCJpZCI6ImFiYzEyMyJ9"
{
  "status": "success",
  "data": {
    "events": [...],
    "pagination": {
      "limit": 50,
      "has_more": true,
      "next_cursor": "eyJzdGFydGVkX2F0IjoiMjAyNS0xMS0zMFQxODowMDowMFoiLCJpZCI6ImRlZjQ1NiJ9"
    }
  },
  "metadata": {
    "timestamp": "2025-12-01T12:01:00Z",
    "query_time_ms": 12
  }
}
```

### Code References

| Component | File | Notes |
|-----------|------|-------|
| PlaybackCursor struct | `internal/models/api_responses.go` | Cursor with StartedAt + ID fields |
| PaginationInfo struct | `internal/models/api_responses.go` | Pagination metadata |
| PlaybacksResponse struct | `internal/models/api_responses.go` | Response wrapper with events + pagination |
| encodeCursor/decodeCursor | `internal/api/handlers_core.go` | Base64 URL encoding |
| Playbacks handler | `internal/api/handlers_core.go` | Routes to cursor/offset pagination |
| handleCursorPagination | `internal/api/handlers_core.go` | Cursor-based pagination handler |
| GetPlaybackEventsWithCursor | `internal/database/crud_playback.go` | Database query with cursor |
| idx_playback_started_at | `internal/database/database_schema.go` | Index for ORDER BY optimization |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| Cursor-based pagination | `internal/api/handlers_core.go:197-206` | Yes |
| Base64 URL encoding | `internal/api/handlers_core.go:244-266` | Yes |
| Tuple comparison query | `internal/database/crud_playback.go:612-702` | Yes |
| idx_playback_started_at index | `internal/database/database_schema.go:787` | Yes |

### Test Coverage

- Cursor encoding tests: `internal/api/handlers_core_cursor_test.go`
- Playbacks handler tests: `internal/api/handlers_core_enhanced_test.go`
- Database pagination tests: `internal/database/crud_pagination_split_test.go`
- Coverage target: 90%+ for pagination functions

---

## Related ADRs

- [ADR-0001](0001-use-duckdb-for-analytics.md): DuckDB index support
- [ADR-0002](0002-frontend-technology-stack.md): Frontend pagination handling

---

## References

- [Cursor-Based Pagination](https://use-the-index-luke.com/no-offset)
- [Slack's Cursor Pagination](https://api.slack.com/docs/pagination)
- [GraphQL Cursor Connections](https://graphql.org/learn/pagination/)
