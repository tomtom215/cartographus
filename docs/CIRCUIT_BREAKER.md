# Circuit Breaker for Tautulli API

**Implementation Date:** 2025-11-23
**Optimization:** LOW-5 - Circuit Breaker for Tautulli API (4-6 hours)
**Status:** ✅ COMPLETE

---

## Overview

The Circuit Breaker pattern prevents cascading failures when the Tautulli API is unavailable, slow, or experiencing high error rates. It implements the "fail fast" principle by:

1. **Detecting failures** - Monitors API call success/failure rates
2. **Opening circuit** - Stops sending requests when failure threshold is exceeded
3. **Fast rejection** - Returns errors immediately when circuit is open (no waiting for timeouts)
4. **Automatic recovery** - Tests API health after timeout period
5. **Closing circuit** - Resumes normal operation when API recovers

This pattern is essential for production reliability, preventing:
- Resource exhaustion from retrying failed requests
- Thread pool saturation from blocked API calls
- Cascading failures across dependent services
- Poor user experience from hanging requests

---

## Architecture

### Circuit Breaker States

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
                                               │
                                               │
                                               │ Failure
                                               │
                                               ▼
                                        ┌─────────────┐
                                        │    OPEN     │
                                        │ (Rejecting) │
                                        └─────────────┘
```

### State Descriptions

| State | Description | Behavior |
|-------|-------------|----------|
| **Closed** | Normal operation | All requests pass through to Tautulli API |
| **Open** | API unavailable | All requests rejected immediately with `ErrOpenState` |
| **Half-Open** | Testing recovery | Limited requests (MaxRequests=3) allowed to test API health |

---

## Configuration

The circuit breaker is configured in `internal/sync/circuit_breaker.go` with the following settings:

```go
cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "tautulli-api",
    MaxRequests: 3,              // Max concurrent requests in half-open state
    Interval:    time.Minute,    // Reset counts after 1 minute in closed state
    Timeout:     2 * time.Minute, // Wait 2 minutes before transitioning to half-open

    // Opens when failure rate >= 60% with minimum 10 requests
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        if counts.Requests < 10 {
            return false // Need statistical significance
        }
        failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
        return failureRatio >= 0.6
    },
})
```

### Configuration Parameters

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| **MaxRequests** | 3 | Conservative limit in half-open state to avoid overwhelming recovering API |
| **Interval** | 1 minute | Rolling window for failure rate calculation in closed state |
| **Timeout** | 2 minutes | Reasonable recovery time for Tautulli server restarts or network issues |
| **Failure Threshold** | 60% | Tolerates transient errors while detecting sustained outages |
| **Minimum Requests** | 10 | Statistical significance threshold to avoid false positives |

### Why These Values?

1. **60% Failure Threshold**: Allows for transient network errors (40% tolerance) while still detecting serious outages quickly
2. **Minimum 10 Requests**: Prevents circuit from opening on first few errors (e.g., 1 failure out of 1 request = 100%)
3. **2 Minute Timeout**: Typical server restart time for Tautulli, plus network recovery time
4. **MaxRequests=3**: Small enough to avoid overwhelming recovering API, large enough for confidence in recovery

---

## Prometheus Metrics

The circuit breaker exposes 4 metrics for monitoring and alerting:

### 1. circuit_breaker_state (Gauge)

**Description**: Current circuit breaker state (0=closed, 1=half-open, 2=open)

**Labels**: `name` (circuit breaker name, e.g., "tautulli-api")

**Example Query**:
```promql
# Check if circuit is open
circuit_breaker_state{name="tautulli-api"} == 2

# Time in open state
time() - (circuit_breaker_state{name="tautulli-api"} == 2)
```

**Alerting Rule**:
```yaml
- alert: CircuitBreakerOpen
  expr: circuit_breaker_state{name="tautulli-api"} == 2
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Tautulli API circuit breaker is OPEN"
    description: "Circuit breaker has been open for >5 minutes, blocking all API requests"
```

### 2. circuit_breaker_requests_total (Counter)

**Description**: Total requests through circuit breaker by result

**Labels**: `name`, `result` (success, failure, rejected)

**Example Query**:
```promql
# Request rate by result
rate(circuit_breaker_requests_total[5m])

# Rejection rate
rate(circuit_breaker_requests_total{result="rejected"}[5m])

# Success rate in half-open state
rate(circuit_breaker_requests_total{result="success"}[1m])
  and circuit_breaker_state == 1
```

**Alerting Rule**:
```yaml
- alert: HighCircuitBreakerRejections
  expr: rate(circuit_breaker_requests_total{result="rejected"}[5m]) > 10
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "High circuit breaker rejection rate"
    description: "Circuit breaker rejecting >10 req/s, API likely unavailable"
```

### 3. circuit_breaker_consecutive_failures (Gauge)

**Description**: Current consecutive failure count

**Labels**: `name`

**Example Query**:
```promql
# Current consecutive failures
circuit_breaker_consecutive_failures{name="tautulli-api"}

# Approaching threshold (60% of 10 = 6 failures)
circuit_breaker_consecutive_failures{name="tautulli-api"} >= 6
```

**Alerting Rule**:
```yaml
- alert: CircuitBreakerHighFailures
  expr: circuit_breaker_consecutive_failures{name="tautulli-api"} >= 5
  for: 1m
  labels:
    severity: warning
  annotations:
    summary: "Circuit breaker approaching failure threshold"
    description: "{{ $value }} consecutive failures, may open soon (threshold: 6/10)"
```

### 4. circuit_breaker_transitions_total (Counter)

**Description**: Total state transitions (closed→open, open→half-open, etc.)

**Labels**: `name`, `from_state`, `to_state`

**Example Query**:
```promql
# Transition rate
rate(circuit_breaker_transitions_total[1h])

# Count of opens in last hour
increase(circuit_breaker_transitions_total{to_state="open"}[1h])

# Successful recoveries (half-open → closed)
increase(circuit_breaker_transitions_total{from_state="half-open",to_state="closed"}[1h])
```

**Alerting Rule**:
```yaml
- alert: FrequentCircuitBreakerTransitions
  expr: increase(circuit_breaker_transitions_total{to_state="open"}[1h]) > 5
  for: 1m
  labels:
    severity: warning
  annotations:
    summary: "Frequent circuit breaker state changes"
    description: "Circuit opened {{ $value }} times in last hour, API stability issues"
```

---

## Usage

### Integration in Application

The circuit breaker is automatically integrated into the application startup in `cmd/server/main.go`:

```go
// Initialize Tautulli client with circuit breaker for fault tolerance
tautulliClient := sync.NewCircuitBreakerClient(&cfg.Tautulli)

// Use as normal - circuit breaker is transparent
if err := tautulliClient.Ping(); err != nil {
    log.Printf("Warning: Failed to connect to Tautulli: %v", err)
}

// Pass to sync manager and API handlers
syncManager := sync.NewManager(db, tautulliClient, &cfg.Sync)
handler := api.NewHandler(db, syncManager, tautulliClient, cfg, jwtManager, wsHub)
```

### Error Handling

When the circuit is **open**, requests are rejected with `gobreaker.ErrOpenState`:

```go
_, err := tautulliClient.GetHistory(since, start, length)
if err != nil {
    if err == gobreaker.ErrOpenState {
        // Circuit is open - API unavailable
        log.Printf("Circuit breaker OPEN: Tautulli API unavailable")
        // Return cached data or graceful degradation
    } else if err == gobreaker.ErrTooManyRequests {
        // Circuit is half-open and MaxRequests exceeded
        log.Printf("Circuit breaker half-open: Too many concurrent requests")
    } else {
        // Actual API error
        log.Printf("Tautulli API error: %v", err)
    }
}
```

### State Transitions in Logs

Circuit breaker state changes are logged automatically:

```
[CIRCUIT BREAKER] State transition: closed -> open
[CIRCUIT BREAKER] Opening circuit after 7 failures (70.0% failure rate)
[CIRCUIT BREAKER] State transition: open -> half-open
[CIRCUIT BREAKER] State transition: half-open -> closed
```

---

## Testing

### Unit Tests

The circuit breaker includes comprehensive unit tests in `internal/sync/circuit_breaker_test.go`:

```bash
# Run circuit breaker tests
go test -v -race ./internal/sync -run TestCircuitBreaker

# Run specific test
go test -v ./internal/sync -run TestCircuitBreaker_OpensAfterFailures

# Benchmarks
go test -bench=BenchmarkCircuitBreaker ./internal/sync
```

### Test Coverage

| Test | Description |
|------|-------------|
| `TestCircuitBreaker_OpensAfterFailures` | Verifies circuit opens after 60% failure rate (7/10 failures) |
| `TestCircuitBreaker_DoesNotOpenBelowThreshold` | Verifies circuit stays closed at 50% failure rate |
| `TestCircuitBreaker_RequiresMinimumRequests` | Verifies minimum 10 requests required |
| `TestCircuitBreaker_TransitionsToHalfOpen` | Verifies timeout transitions to half-open |
| `TestCircuitBreaker_ClosesAfterSuccessInHalfOpen` | Verifies recovery closes circuit |
| `TestCircuitBreaker_MetricsUpdated` | Verifies Prometheus metrics |
| `TestCircuitBreaker_RealAPICall` | Integration test with actual client method |
| `TestCircuitBreaker_MaxRequestsInHalfOpen` | Verifies MaxRequests limit |
| `BenchmarkCircuitBreaker_ClosedState` | Benchmarks normal operation overhead |
| `BenchmarkCircuitBreaker_OpenState` | Benchmarks fast rejection performance |

### Manual Testing

To manually test circuit breaker behavior:

1. **Start application** with Tautulli API unavailable (wrong URL or API key):
   ```bash
   TAUTULLI_URL=http://invalid-url:8181 ./cartographus
   ```

2. **Trigger API calls** (automatic sync will trigger calls):
   ```bash
   curl http://localhost:3857/api/v1/sync
   ```

3. **Monitor logs** for circuit breaker state transitions:
   ```
   [CIRCUIT BREAKER] State transition: closed -> open
   [CIRCUIT BREAKER] Opening circuit after 10 failures (100.0% failure rate)
   ```

4. **Check Prometheus metrics**:
   ```bash
   curl http://localhost:3857/metrics | grep circuit_breaker
   ```

5. **Fix Tautulli connection** and wait 2 minutes for recovery:
   ```
   [CIRCUIT BREAKER] State transition: open -> half-open
   [CIRCUIT BREAKER] State transition: half-open -> closed
   ```

---

## Troubleshooting

### Circuit Keeps Opening

**Symptom**: Circuit breaker repeatedly opens during normal operation

**Possible Causes**:
1. Tautulli API is actually slow or timing out
2. Network connectivity issues
3. Rate limiting from Tautulli (too many requests)
4. Threshold too sensitive (60% may be too low)

**Solutions**:
- Check Tautulli logs for errors
- Verify network connectivity: `ping <tautulli-host>`
- Increase sync interval to reduce request rate
- Check Prometheus metrics: `sync_errors_total{error_type="tautulli_api"}`
- Adjust threshold in `circuit_breaker.go` (e.g., 80% instead of 60%)

### Circuit Never Opens

**Symptom**: Circuit stays closed even when Tautulli is down

**Possible Causes**:
1. Not enough requests to meet minimum threshold (need 10)
2. Errors are being caught and not returned from API calls
3. Circuit breaker not properly wrapping calls

**Solutions**:
- Check request count: `circuit_breaker_requests_total`
- Verify failures are being tracked: `circuit_breaker_consecutive_failures`
- Enable debug logging in `circuit_breaker.go`
- Check that all API methods use `execute()` wrapper

### Circuit Stuck in Half-Open

**Symptom**: Circuit transitions to half-open but never closes

**Possible Causes**:
1. MaxRequests=3 are all failing
2. API is partially recovered (intermittent errors)
3. Slow API responses causing timeouts

**Solutions**:
- Monitor half-open metrics: `circuit_breaker_requests_total{result="success"} and circuit_breaker_state == 1`
- Check consecutive failures gauge
- Increase MaxRequests to allow more test requests
- Investigate Tautulli performance issues

### High Rejection Rate

**Symptom**: Many requests rejected with `ErrOpenState`

**Possible Causes**:
1. Tautulli API is down (expected behavior)
2. Circuit not recovering fast enough
3. Timeout too long (2 minutes)

**Solutions**:
- **If API is down**: This is correct behavior! Circuit is protecting your application
- Check Tautulli health: `curl http://<tautulli>/api/v2?cmd=status`
- Reduce timeout to recover faster (e.g., 1 minute instead of 2)
- Implement graceful degradation (cached data, reduced features)

---

## Performance Impact

### Overhead in Closed State

- **Per-request overhead**: ~5-10 microseconds (gobreaker state check + metrics)
- **Memory overhead**: ~200 bytes per circuit breaker instance
- **CPU overhead**: Negligible (<0.1% in typical workloads)

**Benchmark Results** (typical):
```
BenchmarkCircuitBreaker_ClosedState-8     1000000    1.2 µs/op
BenchmarkCircuitBreaker_OpenState-8      10000000    0.1 µs/op
```

### Benefits in Open State

- **Fast rejection**: 100x faster than waiting for API timeout (0.1µs vs 10-30s)
- **Resource savings**: No blocked goroutines, no thread pool saturation
- **Improved UX**: Instant error response instead of hanging requests

---

## Integration with Other Optimizations

### Works with MEDIUM-2: Batch Geolocation Lookups

Circuit breaker protects geolocation API calls in `internal/sync/manager.go`:

```go
// Circuit breaker automatically wraps GetGeoIPLookup()
geo, err := m.client.GetGeoIPLookup(ip)
if err == gobreaker.ErrOpenState {
    // Use default location or skip this IP
    log.Printf("Geolocation lookup skipped (circuit open): %s", ip)
}
```

### Works with LOW-4: Prometheus Metrics

All circuit breaker metrics are exposed on `/metrics` endpoint:

```bash
# View all circuit breaker metrics
curl http://localhost:3857/metrics | grep circuit_breaker

# Example output:
circuit_breaker_state{name="tautulli-api"} 0
circuit_breaker_requests_total{name="tautulli-api",result="success"} 1234
circuit_breaker_requests_total{name="tautulli-api",result="failure"} 56
circuit_breaker_requests_total{name="tautulli-api",result="rejected"} 0
circuit_breaker_consecutive_failures{name="tautulli-api"} 0
circuit_breaker_transitions_total{name="tautulli-api",from_state="closed",to_state="open"} 2
circuit_breaker_transitions_total{name="tautulli-api",from_state="open",to_state="half-open"} 2
circuit_breaker_transitions_total{name="tautulli-api",from_state="half-open",to_state="closed"} 2
```

---

## Best Practices

### 1. Monitor Circuit Breaker State

Always monitor circuit breaker state in production:

```yaml
# Grafana dashboard panel
- title: "Circuit Breaker State"
  targets:
    - expr: circuit_breaker_state{name="tautulli-api"}
      legendFormat: "State (0=closed, 1=half-open, 2=open)"
  alert:
    name: "Circuit Breaker Open"
    condition: "state == 2 for 5m"
```

### 2. Implement Graceful Degradation

When circuit is open, use cached data or reduced features:

```go
data, err := tautulliClient.GetHistory(...)
if err == gobreaker.ErrOpenState {
    // Use cached data with warning
    cachedData := cache.Get("recent_history")
    if cachedData != nil {
        log.Println("Using cached data (Tautulli API unavailable)")
        return cachedData, nil
    }
    // Or return reduced dataset
    return emptyHistory, nil
}
```

### 3. Alert on Repeated Opens

Circuit opening once is normal, but repeated opens indicate problems:

```promql
increase(circuit_breaker_transitions_total{to_state="open"}[1h]) > 5
```

### 4. Test Circuit Breaker in Staging

Before deploying, test circuit breaker behavior:

```bash
# Temporarily break Tautulli connection
iptables -A OUTPUT -d <tautulli-ip> -j DROP

# Monitor circuit breaker state
watch 'curl -s http://localhost:3857/metrics | grep circuit_breaker_state'

# Restore connection
iptables -D OUTPUT -d <tautulli-ip> -j DROP

# Verify recovery
```

---

## Future Enhancements

Potential improvements for future versions:

1. **Dynamic Thresholds**: Adjust failure threshold based on historical error rates
2. **Per-Endpoint Circuit Breakers**: Separate circuits for different API endpoints
3. **Adaptive Timeout**: Automatically adjust timeout based on recovery patterns
4. **Circuit Breaker Dashboard**: Real-time state visualization
5. **Failure Classification**: Different thresholds for network vs application errors
6. **Health Checks**: Active health probing in half-open state instead of waiting for requests

---

## References

- **Circuit Breaker Pattern**: [Martin Fowler's Circuit Breaker](https://martinfowler.com/bliki/CircuitBreaker.html)
- **gobreaker Library**: [sony/gobreaker](https://github.com/sony/gobreaker)
- **Prometheus Metrics**: See `docs/PROMETHEUS_METRICS.md`
- **Optimization Progress**: See `docs/OPTIMIZATION_PROGRESS.md`

---

**End of Circuit Breaker Documentation**
**Last Updated:** 2025-11-23
