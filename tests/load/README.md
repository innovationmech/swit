# Security Load Testing

This directory contains load and stress tests for the security components of the swit framework.

## Overview

The load tests verify that the security middleware stack (JWT authentication, OPA policy evaluation, OAuth2 token cache) can handle the target performance requirements:

- **Target RPS**: 10,000 requests per second
- **P99 Latency**: < 15ms for full middleware stack
- **Error Rate**: < 1%
- **Performance Impact**: < 5%

## Test Files

### Go Load Tests (`security_load_test.go`)

Native Go load tests using the standard testing framework. These tests provide:

- **Token Cache Load Tests**: High-throughput cache operations
- **JWT Validation Tests**: Token validation performance
- **OPA Policy Evaluation Tests**: Policy evaluation latency
- **Middleware Stack Tests**: Full security middleware stack
- **Stability Tests**: Long-running sustained load
- **Stress Tests**: Beyond-normal-limits testing
- **Performance Impact Assessment**: Baseline comparison

#### Running Go Load Tests

```bash
# Run all load tests
go test -v ./tests/load/... -count=1

# Run specific test
go test -v ./tests/load/... -run TestLoadOAuth2TokenCache -count=1

# Run with race detection
go test -v -race ./tests/load/... -count=1

# Skip load tests (short mode)
go test -v -short ./tests/load/...

# Run stability test (longer duration)
go test -v ./tests/load/... -run TestStabilityLongRunning -count=1 -timeout 5m
```

### k6 Load Tests (`k6-script.js`)

JavaScript-based load tests using [k6](https://k6.io/). These tests provide:

- **Smoke Test**: Quick validation
- **Load Test**: Target 10,000 RPS
- **Stress Test**: Beyond-limits testing
- **Stability Test**: Long-running sustained load

#### Prerequisites

Install k6:

```bash
# macOS
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6
```

#### Running k6 Load Tests

```bash
# Run default load test
k6 run tests/load/k6-script.js

# Run smoke test
k6 run -e SCENARIO=smoke tests/load/k6-script.js

# Run stress test
k6 run -e SCENARIO=stress tests/load/k6-script.js

# Run stability test
k6 run -e SCENARIO=stability tests/load/k6-script.js

# Custom configuration
k6 run --vus 100 --duration 30s tests/load/k6-script.js

# Target specific service
k6 run -e BASE_URL=http://localhost:8080 tests/load/k6-script.js

# With Docker
docker run -i grafana/k6 run - < tests/load/k6-script.js
```

## Test Scenarios

### Smoke Test
Quick validation that the system works correctly under minimal load.
- Duration: 30 seconds
- VUs: 1
- Purpose: Functionality verification

### Load Test
Standard load test targeting 10,000 RPS.
- Duration: ~8 minutes
- Target RPS: 10,000
- Ramp-up: Gradual increase to target
- Purpose: Performance validation

### Stress Test
Push the system beyond normal limits to find breaking points.
- Duration: ~5 minutes
- Target RPS: 20,000
- Purpose: Identify system limits

### Stability Test
Long-running test to verify system stability over time.
- Duration: 30 minutes
- Sustained RPS: 5,000
- Purpose: Memory leaks, degradation detection

## Performance Targets

| Component | Target | Metric |
|-----------|--------|--------|
| OAuth2 Token Cache | < 10ms | P99 Latency |
| JWT Validation | < 1ms | P99 Latency |
| OPA Policy Evaluation | < 5ms | P99 Latency |
| Full Middleware Stack | < 15ms | P99 Latency |
| Error Rate | < 1% | Request Failures |
| Performance Impact | < 5% | Overhead vs Baseline |

## Output

### Go Tests
Results are printed to stdout with detailed metrics including:
- Total/Success/Failed requests
- RPS achieved
- Latency percentiles (P50, P90, P99)
- Memory usage
- GC pauses

### k6 Tests
Results are saved to `tests/load/results/`:
- `summary.json`: Machine-readable summary
- `report.txt`: Human-readable report

## Interpreting Results

### Success Criteria
- All latency thresholds met
- Error rate below target
- No memory leaks in stability tests
- Consistent performance over time

### Common Issues

1. **High P99 Latency**
   - Check for GC pauses
   - Verify cache hit rates
   - Review connection pooling

2. **High Error Rate**
   - Check service health
   - Verify token validity
   - Review rate limiting configuration

3. **Memory Growth**
   - Check for cache eviction
   - Review connection cleanup
   - Monitor goroutine counts

## Integration with CI/CD

Add to your CI pipeline:

```yaml
# GitHub Actions example
- name: Run Load Tests
  run: |
    go test -v ./tests/load/... -count=1 -timeout 10m
```

For k6 in CI:

```yaml
- name: Run k6 Load Tests
  uses: grafana/k6-action@v0.3.0
  with:
    filename: tests/load/k6-script.js
    flags: -e SCENARIO=smoke
```

## Related Documentation

- [Performance Monitoring Guide](../../docs/performance-monitoring-alerting.md)
- [Security Best Practices](../../docs/security-best-practices.md)
- [Benchmark Tests](../../pkg/security/oauth2/benchmark_test.go)







