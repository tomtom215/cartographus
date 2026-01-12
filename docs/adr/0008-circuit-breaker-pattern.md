# ADR-0008: Circuit Breaker for Tautulli API

**Date**: 2025-11-23
**Status**: Accepted

---

## Context

Cartographus depends on the Tautulli API for playback history synchronization. When Tautulli is unavailable:

1. **Cascading Failures**: Retry storms can exhaust resources
2. **Thread Pool Saturation**: Blocked goroutines waiting for timeouts
3. **Poor UX**: Users wait 30+ seconds for requests to timeout
4. **Resource Waste**: Continued attempts against known-failing endpoint

### Requirements

- Fast failure when Tautulli is unavailable
- Automatic recovery when Tautulli recovers
- Configurable thresholds for different network conditions
- Prometheus metrics for monitoring
- No external dependencies (embedded)

### Alternatives Considered

| Library | Pros | Cons |
|---------|------|------|
| **Manual implementation** | Full control | Error-prone, maintenance |
| **hystrix-go** | Netflix-proven | Archived, no longer maintained |
| **gobreaker** | Sony production-proven | Additional dependency |
| **resilience4go** | Feature-rich | Heavier API |

---

## Decision

Use **Sony gobreaker v2** for circuit breaker implementation with the following configuration:

```go
cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "tautulli-api",
    MaxRequests: 3,              // Half-open state limit
    Interval:    time.Minute,    // Reset window
    Timeout:     2 * time.Minute, // Open → Half-open
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        if counts.Requests < 10 {
            return false // Need statistical significance
        }
        failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
        return failureRatio >= 0.6 // 60% failure threshold
    },
})
```

### State Machine

```
┌─────────────┐
│   CLOSED    │ ◄─────────────────────────────┐
│  (Normal)   │                                │
└──────┬──────┘                                │
       │                                       │
       │ Failure threshold exceeded            │ Success in half-open
       │ (60% failure rate, min 10 requests)   │ (MaxRequests succeed)
       │                                       │
       ▼                                       │
┌─────────────┐                         ┌─────────────┐
│    OPEN     │ ───────────────────────▶│  HALF-OPEN  │
│ (Rejecting) │   After timeout (2 min) │  (Testing)  │
└─────────────┘                         └─────────────┘
       ▲                                       │
       │                                       │
       └───────────────────────────────────────┘
                    Failure in half-open
```

### Key Factors

1. **Production-Proven**: Sony uses gobreaker in production systems
2. **Simple API**: Wraps existing functions transparently
3. **Low Overhead**: Minimal performance impact in closed state
4. **Configurable**: Thresholds tunable for network conditions

---

## Consequences

### Positive

- **Fast Failure**: 0.1µs rejection vs 10-30s timeout
- **Resource Protection**: No blocked goroutines on failures
- **Automatic Recovery**: Tests API health after timeout
- **Observability**: Prometheus metrics for monitoring
- **Graceful Degradation**: Cached data used when circuit open

### Negative

- **Additional Dependency**: `github.com/sony/gobreaker/v2`
- **Configuration Complexity**: Thresholds require tuning
- **Transient Failures**: May reject valid requests during half-open

### Neutral

- **State Management**: Circuit state stored in memory
- **Threshold Tuning**: 60% failure rate is a starting point

---

## Implementation

### Circuit Breaker Client

```go
// internal/sync/circuit_breaker.go
type CircuitBreakerClient struct {
    client *TautulliClient
    cb     *gobreaker.CircuitBreaker[interface{}]
    name   string
}

func NewCircuitBreakerClient(cfg *config.TautulliConfig) *CircuitBreakerClient {
    client := NewTautulliClient(cfg)
    cbName := "tautulli-api"

    // Initialize circuit breaker state metrics
    metrics.CircuitBreakerState.WithLabelValues(cbName).Set(0)
    metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(cbName).Set(0)

    cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
        Name:        cbName,
        MaxRequests: 3,
        Interval:    time.Minute,
        Timeout:     2 * time.Minute,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            if counts.Requests < 10 {
                return false
            }
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return failureRatio >= 0.6
        },
        OnStateChange: func(name string, from, to gobreaker.State) {
            fromStr := stateToString(from)
            toStr := stateToString(to)
            logging.Info().Str("from", fromStr).Str("to", toStr).Msg("[CIRCUIT BREAKER] State transition")
            metrics.CircuitBreakerState.WithLabelValues(name).Set(stateToFloat(to))
            metrics.CircuitBreakerTransitions.WithLabelValues(name, fromStr, toStr).Inc()
        },
    })

    return &CircuitBreakerClient{
        client: client,
        cb:     cb,
        name:   cbName,
    }
}

func (cbc *CircuitBreakerClient) execute(fn func() (interface{}, error)) (interface{}, error) {
    result, err := cbc.cb.Execute(fn)

    if err != nil {
        if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
            metrics.CircuitBreakerRequests.WithLabelValues(cbc.name, "rejected").Inc()
        } else {
            metrics.CircuitBreakerRequests.WithLabelValues(cbc.name, "failure").Inc()
            counts := cbc.cb.Counts()
            metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(cbc.name).Set(float64(counts.ConsecutiveFailures))
        }
        return nil, err
    }

    metrics.CircuitBreakerRequests.WithLabelValues(cbc.name, "success").Inc()
    metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(cbc.name).Set(0)
    return result, nil
}
```

### Wrapped API Calls

```go
func (cbc *CircuitBreakerClient) GetHistorySince(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
    return castResult[tautulli.TautulliHistory](cbc.execute(func() (interface{}, error) {
        return cbc.client.GetHistorySince(ctx, since, start, length)
    }))
}

func (cbc *CircuitBreakerClient) Ping(ctx context.Context) error {
    _, err := cbc.execute(func() (interface{}, error) {
        return nil, cbc.client.Ping(ctx)
    })
    return err
}
```

### Prometheus Metrics

```go
// internal/metrics/metrics.go
var (
    CircuitBreakerState = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "circuit_breaker_state",
            Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
        },
        []string{"name"},
    )

    CircuitBreakerRequests = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "circuit_breaker_requests_total",
            Help: "Total number of requests through circuit breaker",
        },
        []string{"name", "result"},
    )

    CircuitBreakerConsecutiveFailures = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "circuit_breaker_consecutive_failures",
            Help: "Current number of consecutive failures",
        },
        []string{"name"},
    )

    CircuitBreakerTransitions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "circuit_breaker_state_transitions_total",
            Help: "Total number of circuit breaker state transitions",
        },
        []string{"name", "from_state", "to_state"},
    )
)
```

### Error Handling

The circuit breaker wrapper handles errors automatically. When using the client:

```go
// Example error handling pattern
history, err := tautulliClient.GetHistorySince(ctx, since, 0, 1000)
if err != nil {
    if errors.Is(err, gobreaker.ErrOpenState) {
        logging.Warn().Msg("Sync skipped: Tautulli API circuit breaker OPEN")
        // Use cached data, don't retry
        return nil
    }
    if errors.Is(err, gobreaker.ErrTooManyRequests) {
        logging.Warn().Msg("Sync throttled: Half-open state, too many requests")
        return nil
    }
    return fmt.Errorf("sync failed: %w", err)
}
```

### Configuration

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| MaxRequests | 3 | Conservative limit in half-open |
| Interval | 1 minute | Rolling window for failure rate |
| Timeout | 2 minutes | Typical server restart time |
| Failure Threshold | 60% | Tolerates transient errors |
| Minimum Requests | 10 | Statistical significance |

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Circuit breaker client | `internal/sync/circuit_breaker.go` | gobreaker v2 wrapper |
| Prometheus metrics | `internal/metrics/metrics.go` | Gauges and counters (lines 249-279) |
| Settings struct | `internal/sync/circuit_breaker.go:44-83` | Circuit breaker configuration |
| Unit tests | `internal/sync/circuit_breaker_test.go` | Comprehensive test coverage |
| Benchmarks | `internal/sync/circuit_breaker_test.go` | `BenchmarkCircuitBreaker_*` functions |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| gobreaker v2.4.0 | `go.mod:22` | Yes |
| 60% failure threshold | `internal/sync/circuit_breaker.go:58` | Yes |
| 2 minute timeout | `internal/sync/circuit_breaker.go:48` | Yes |
| Prometheus metrics | `internal/metrics/metrics.go:249-279` | Yes |
| State transitions logged | `internal/sync/circuit_breaker.go:72` | Yes |

### Test Coverage

- Circuit breaker tests: `internal/sync/circuit_breaker_test.go`
- Benchmark tests: `BenchmarkCircuitBreaker_*`
- Coverage target: 90%+ for circuit breaker functions

---

## Related ADRs

- [ADR-0005](0005-nats-jetstream-event-processing.md): Alternative event source
- [ADR-0009](0009-plex-direct-integration.md): Plex direct fallback

---

## References

- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html)
- [Sony gobreaker](https://github.com/sony/gobreaker)
- [docs/CIRCUIT_BREAKER.md](../CIRCUIT_BREAKER.md)
