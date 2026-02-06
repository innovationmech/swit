# Security Testing Guide | 安全测试指南

This comprehensive guide covers testing strategies for the security components of the Swit framework, including unit testing, integration testing, performance testing, security scanning, and best practices.

本综合指南涵盖 Swit 框架安全组件的测试策略，包括单元测试、集成测试、性能测试、安全扫描和最佳实践。

---

## Table of Contents | 目录

- [Testing Architecture | 测试架构](#testing-architecture--测试架构)
- [Test Environment Setup | 测试环境搭建](#test-environment-setup--测试环境搭建)
- [Running Tests | 运行测试](#running-tests--运行测试)
- [Unit Testing | 单元测试](#unit-testing--单元测试)
- [Integration Testing | 集成测试](#integration-testing--集成测试)
- [Performance Testing | 性能测试](#performance-testing--性能测试)
- [Security Testing | 安全测试](#security-testing--安全测试)
- [Load Testing | 负载测试](#load-testing--负载测试)
- [CI/CD Integration | CI/CD 集成](#cicd-integration--cicd-集成)
- [Troubleshooting | 故障排除](#troubleshooting--故障排除)

---

## Testing Architecture | 测试架构

### Overview | 概述

The security testing architecture is organized into multiple layers:

安全测试架构分为多个层次：

```
tests/
├── load/                    # Load and stress tests | 负载和压力测试
│   ├── security_load_test.go
│   ├── k6-script.js
│   └── README.md
└── security/                # Security test configurations | 安全测试配置
    ├── docker-compose.test.yml
    ├── keycloak/
    └── policies/

pkg/security/
├── jwt/                     # JWT validation tests | JWT 验证测试
│   ├── *_test.go
│   └── validator_bench_test.go
├── oauth2/                  # OAuth2 tests | OAuth2 测试
│   ├── *_test.go
│   ├── benchmark_test.go
│   └── integration_test.go
├── opa/                     # OPA policy tests | OPA 策略测试
│   ├── *_test.go
│   ├── benchmark_test.go
│   └── integration_test.go
├── scanner/                 # Security scanner tests | 安全扫描测试
│   └── *_test.go
├── secrets/                 # Secrets management tests | 密钥管理测试
│   └── *_test.go
├── testing/                 # Security-specific tests | 安全专项测试
│   ├── dos_test.go
│   ├── fuzz_test.go
│   ├── penetration_test.go
│   ├── timing_test.go
│   └── security_test.go
├── tls/                     # TLS configuration tests | TLS 配置测试
│   └── *_test.go
└── vulnerability/           # Vulnerability checker tests | 漏洞检查测试
    └── *_test.go
```

### Test Categories | 测试分类

| Category | Purpose | Location |
|----------|---------|----------|
| Unit Tests | Test individual components | `pkg/security/*_test.go` |
| Integration Tests | Test component interactions | `pkg/security/*/integration_test.go` |
| Benchmark Tests | Measure performance | `pkg/security/*/benchmark_test.go` |
| Load Tests | Verify scalability | `tests/load/` |
| Security Tests | Verify security properties | `pkg/security/testing/` |

| 分类 | 目的 | 位置 |
|------|------|------|
| 单元测试 | 测试独立组件 | `pkg/security/*_test.go` |
| 集成测试 | 测试组件交互 | `pkg/security/*/integration_test.go` |
| 基准测试 | 测量性能 | `pkg/security/*/benchmark_test.go` |
| 负载测试 | 验证可扩展性 | `tests/load/` |
| 安全测试 | 验证安全属性 | `pkg/security/testing/` |

### Component Coverage | 组件覆盖

```
┌─────────────────────────────────────────────────────────────────┐
│                    Security Testing Stack                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   OAuth2    │  │    JWT      │  │    OPA      │              │
│  │   Tests     │  │   Tests     │  │   Tests     │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│         │                │                │                      │
│         ▼                ▼                ▼                      │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              Integration Tests                               ││
│  │   - OAuth2 Flow Tests                                       ││
│  │   - JWT + OPA Middleware Tests                              ││
│  │   - Full Authentication Chain                               ││
│  └─────────────────────────────────────────────────────────────┘│
│         │                │                │                      │
│         ▼                ▼                ▼                      │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              Performance Tests                               ││
│  │   - Benchmark Tests (Go native)                             ││
│  │   - Load Tests (k6)                                         ││
│  │   - Stress Tests                                            ││
│  └─────────────────────────────────────────────────────────────┘│
│         │                │                │                      │
│         ▼                ▼                ▼                      │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              Security Tests                                  ││
│  │   - Penetration Tests                                       ││
│  │   - Fuzz Tests                                              ││
│  │   - Timing Attack Tests                                     ││
│  │   - DoS Protection Tests                                    ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

---

## Test Environment Setup | 测试环境搭建

### Prerequisites | 前置条件

#### Required Tools | 必需工具

```bash
# Go 1.23+
go version

# Docker (for integration tests)
docker --version
docker-compose --version

# k6 (for load tests)
# macOS
brew install k6

# Linux
sudo apt-get install k6

# gosec (for security scanning)
go install github.com/securego/gosec/v2/cmd/gosec@latest

# govulncheck (for vulnerability scanning)
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### Docker Test Environment | Docker 测试环境

For integration tests requiring external services:

对于需要外部服务的集成测试：

```bash
# Start test environment
cd tests/security
docker-compose -f docker-compose.test.yml up -d

# Verify services are running
docker-compose -f docker-compose.test.yml ps

# Stop test environment
docker-compose -f docker-compose.test.yml down
```

### Environment Variables | 环境变量

```bash
# Test configuration
export SWIT_TEST_MODE=true
export SWIT_LOG_LEVEL=debug

# OAuth2 test configuration (if using external provider)
export OAUTH2_TEST_CLIENT_ID=test-client
export OAUTH2_TEST_CLIENT_SECRET=test-secret
export OAUTH2_TEST_ISSUER_URL=http://localhost:8080/realms/test

# OPA test configuration
export OPA_TEST_POLICY_DIR=/path/to/test/policies

# Skip long-running tests
export SWIT_SKIP_LONG_TESTS=true
```

---

## Running Tests | 运行测试

### Quick Start | 快速开始

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test -v ./pkg/security/... -count=1

# Run with race detection
go test -v -race ./pkg/security/... -count=1
```

### Test Commands Reference | 测试命令参考

| Command | Description | 描述 |
|---------|-------------|------|
| `make test` | Run all tests | 运行所有测试 |
| `make test-coverage` | Generate coverage report | 生成覆盖率报告 |
| `make test-race` | Run tests with race detection | 使用竞态检测运行测试 |
| `go test -short ./...` | Skip long-running tests | 跳过长时间运行的测试 |
| `go test -v ./pkg/security/jwt/...` | Test JWT package | 测试 JWT 包 |
| `go test -v ./pkg/security/oauth2/...` | Test OAuth2 package | 测试 OAuth2 包 |
| `go test -v ./pkg/security/opa/...` | Test OPA package | 测试 OPA 包 |

### Running Specific Test Types | 运行特定类型的测试

```bash
# Unit tests only
go test -v ./pkg/security/... -run "^Test[^Integration]" -count=1

# Integration tests only
go test -v ./pkg/security/... -run "Integration" -count=1

# Benchmark tests
go test -v ./pkg/security/... -bench=. -benchmem

# Security tests
go test -v ./pkg/security/testing/... -count=1

# Load tests
go test -v ./tests/load/... -count=1 -timeout 10m
```

---

## Unit Testing | 单元测试

### JWT Validation Tests | JWT 验证测试

```go
// Example: Testing JWT validator
func TestJWTValidator_ValidateToken(t *testing.T) {
    secret := "test-secret-key-32-bytes-long!!"
    
    config := &jwt.Config{
        Secret: secret,
    }
    
    validator, err := jwt.NewValidator(config)
    require.NoError(t, err)
    
    // Create a valid token
    claims := jwtlib.MapClaims{
        "sub":   "user123",
        "exp":   time.Now().Add(time.Hour).Unix(),
        "iat":   time.Now().Unix(),
        "roles": []string{"admin"},
    }
    
    token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(secret))
    require.NoError(t, err)
    
    // Validate the token
    parsedToken, err := validator.ValidateToken(tokenString)
    require.NoError(t, err)
    assert.True(t, parsedToken.Valid)
}
```

### OAuth2 Cache Tests | OAuth2 缓存测试

```go
// Example: Testing token cache
func TestMemoryTokenCache_SetGet(t *testing.T) {
    cache := oauth2.NewMemoryTokenCacheWithSize(time.Hour, 1000)
    defer cache.Close()
    
    ctx := context.Background()
    
    token := &oauth2lib.Token{
        AccessToken: "test-access-token",
        TokenType:   "Bearer",
        Expiry:      time.Now().Add(time.Hour),
    }
    
    // Set token
    err := cache.Set(ctx, "test-key", token)
    require.NoError(t, err)
    
    // Get token
    retrieved, err := cache.Get(ctx, "test-key")
    require.NoError(t, err)
    assert.Equal(t, token.AccessToken, retrieved.AccessToken)
}
```

### OPA Policy Tests | OPA 策略测试

```go
// Example: Testing OPA policy evaluation
func TestOPAClient_Evaluate(t *testing.T) {
    ctx := context.Background()
    tmpDir := t.TempDir()
    
    config := &opa.Config{
        Mode: opa.ModeEmbedded,
        EmbeddedConfig: &opa.EmbeddedConfig{
            PolicyDir: tmpDir,
        },
        DefaultDecisionPath: "authz.allow",
    }
    
    client, err := opa.NewClient(ctx, config)
    require.NoError(t, err)
    defer client.Close(ctx)
    
    // Load test policy
    policy := `
package authz
default allow = false
allow { input.user.roles[_] == "admin" }
`
    err = client.LoadPolicy(ctx, "authz.rego", policy)
    require.NoError(t, err)
    
    // Test evaluation
    input := map[string]interface{}{
        "user": map[string]interface{}{
            "roles": []string{"admin"},
        },
    }
    
    result, err := client.Evaluate(ctx, "authz.allow", input)
    require.NoError(t, err)
    assert.True(t, result.Allowed)
}
```

### Test Naming Conventions | 测试命名约定

Follow these naming patterns:

遵循以下命名模式：

```go
// Unit tests: Test<Function>_<Scenario>
func TestValidator_ValidToken(t *testing.T) {}
func TestValidator_ExpiredToken(t *testing.T) {}
func TestValidator_InvalidSignature(t *testing.T) {}

// Table-driven tests: Test<Function>
func TestValidator(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    bool
        wantErr bool
    }{
        {"valid token", validToken, true, false},
        {"expired token", expiredToken, false, true},
    }
    // ...
}

// Benchmark tests: Benchmark<Function>_<Scenario>
func BenchmarkValidator_ValidateToken(b *testing.B) {}
func BenchmarkCache_Get(b *testing.B) {}
```

---

## Integration Testing | 集成测试

### OAuth2 Integration Tests | OAuth2 集成测试

```go
// Full OAuth2 flow test
func TestOAuth2Integration_AuthorizationCodeFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Setup test server
    config := &oauth2.Config{
        ClientID:     "test-client",
        ClientSecret: "test-secret",
        // ... other config
    }
    
    client, err := oauth2.NewClient(config)
    require.NoError(t, err)
    
    // Test authorization URL generation
    authURL := client.GetAuthorizationURL("test-state", "openid profile")
    assert.Contains(t, authURL, "response_type=code")
    
    // Test token exchange (requires mock server)
    // ...
}
```

### Middleware Integration Tests | 中间件集成测试

```go
func TestMiddlewareIntegration_FullStack(t *testing.T) {
    ctx := context.Background()
    
    // Setup JWT validator
    jwtConfig := &jwt.Config{Secret: "test-secret"}
    jwtValidator, _ := jwt.NewValidator(jwtConfig)
    
    // Setup OPA client
    opaConfig := &opa.Config{
        Mode: opa.ModeEmbedded,
        // ...
    }
    opaClient, _ := opa.NewClient(ctx, opaConfig)
    defer opaClient.Close(ctx)
    
    // Create middleware stack
    router := gin.New()
    router.Use(middleware.OAuth2Middleware(nil, jwtValidator))
    router.Use(middleware.OPAMiddleware(opaClient))
    router.GET("/api/test", func(c *gin.Context) {
        c.String(http.StatusOK, "OK")
    })
    
    // Test with valid token
    token := createTestToken("test-secret")
    req := httptest.NewRequest("GET", "/api/test", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### Docker-based Integration Tests | 基于 Docker 的集成测试

```yaml
# tests/security/docker-compose.test.yml
version: '3.8'
services:
  keycloak:
    image: quay.io/keycloak/keycloak:latest
    environment:
      KEYCLOAK_ADMIN: admin
      KEYCLOAK_ADMIN_PASSWORD: admin
    ports:
      - "8080:8080"
    command: start-dev
    
  opa:
    image: openpolicyagent/opa:latest
    ports:
      - "8181:8181"
    command: run --server --addr :8181
```

```bash
# Run integration tests with Docker
docker-compose -f tests/security/docker-compose.test.yml up -d
go test -v ./pkg/security/... -run Integration -count=1
docker-compose -f tests/security/docker-compose.test.yml down
```

---

## Performance Testing | 性能测试

### Benchmark Tests | 基准测试

```bash
# Run all benchmarks
go test -v ./pkg/security/... -bench=. -benchmem

# Run specific benchmarks
go test -v ./pkg/security/jwt/... -bench=BenchmarkValidator -benchmem
go test -v ./pkg/security/oauth2/... -bench=BenchmarkCache -benchmem
go test -v ./pkg/security/opa/... -bench=BenchmarkEvaluate -benchmem

# Run benchmarks with profiling
go test -v ./pkg/security/jwt/... -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof
go tool pprof cpu.prof
```

### Performance Targets | 性能目标

| Component | Target P99 | Target RPS |
|-----------|------------|------------|
| JWT Validation | < 1ms | 100,000+ |
| OAuth2 Token Cache | < 10ms | 50,000+ |
| OPA Policy Evaluation | < 5ms | 50,000+ |
| Full Middleware Stack | < 15ms | 10,000+ |

| 组件 | 目标 P99 | 目标 RPS |
|------|----------|----------|
| JWT 验证 | < 1ms | 100,000+ |
| OAuth2 令牌缓存 | < 10ms | 50,000+ |
| OPA 策略评估 | < 5ms | 50,000+ |
| 完整中间件栈 | < 15ms | 10,000+ |

### Benchmark Example Output | 基准测试示例输出

```
BenchmarkValidator_ValidateToken-8        500000      2341 ns/op      1024 B/op      12 allocs/op
BenchmarkCache_Get-8                     1000000      1023 ns/op       256 B/op       4 allocs/op
BenchmarkOPAClient_Evaluate-8             200000      6234 ns/op      2048 B/op      24 allocs/op
```

---

## Security Testing | 安全测试

### Static Security Analysis | 静态安全分析

```bash
# Run gosec
gosec -fmt=json -out=gosec-report.json ./pkg/security/...

# Run govulncheck
govulncheck ./pkg/security/...

# Run all security checks
make quality-advanced OPERATION=security
```

### Security Test Categories | 安全测试分类

#### 1. Penetration Tests | 渗透测试

```go
// pkg/security/testing/penetration_test.go
func TestPenetration_TokenForging(t *testing.T) {
    // Test that forged tokens are rejected
    validator, _ := jwt.NewValidator(&jwt.Config{Secret: "secret"})
    
    forgedToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.forged.signature"
    _, err := validator.ValidateToken(forgedToken)
    
    assert.Error(t, err, "Forged token should be rejected")
}
```

#### 2. Timing Attack Tests | 时间侧信道攻击测试

```go
// pkg/security/testing/timing_test.go
func TestTiming_ConstantTimeComparison(t *testing.T) {
    // Verify that token comparison is constant-time
    secret := "test-secret"
    
    validToken := createToken(secret)
    invalidToken := "invalid-token"
    
    // Measure validation time for valid and invalid tokens
    validDuration := measureValidationTime(validToken)
    invalidDuration := measureValidationTime(invalidToken)
    
    // Times should be similar (within tolerance)
    tolerance := 100 * time.Microsecond
    diff := abs(validDuration - invalidDuration)
    assert.Less(t, diff, tolerance, "Timing should be constant")
}
```

#### 3. Fuzz Tests | 模糊测试

```go
// pkg/security/testing/fuzz_test.go
func FuzzJWTValidation(f *testing.F) {
    validator, _ := jwt.NewValidator(&jwt.Config{Secret: "secret"})
    
    // Add seed corpus
    f.Add("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.signature")
    f.Add("")
    f.Add("invalid")
    
    f.Fuzz(func(t *testing.T, input string) {
        // Should not panic
        _, _ = validator.ValidateToken(input)
    })
}
```

#### 4. DoS Protection Tests | DoS 防护测试

```go
// pkg/security/testing/dos_test.go
func TestDoS_LargePayload(t *testing.T) {
    // Test that large payloads are rejected
    largePayload := strings.Repeat("a", 10*1024*1024) // 10MB
    
    _, err := processPayload(largePayload)
    assert.Error(t, err, "Large payload should be rejected")
}

func TestDoS_RateLimiting(t *testing.T) {
    // Test rate limiting
    limiter := NewRateLimiter(100) // 100 requests/second
    
    for i := 0; i < 200; i++ {
        allowed := limiter.Allow()
        if i >= 100 {
            assert.False(t, allowed, "Should be rate limited")
        }
    }
}
```

### Security Checklist | 安全检查清单

- [ ] Token validation rejects expired tokens
- [ ] Token validation rejects invalid signatures
- [ ] Token validation rejects tampered payloads
- [ ] Rate limiting prevents DoS attacks
- [ ] Sensitive data is not logged
- [ ] TLS is properly configured
- [ ] Secrets are not hardcoded
- [ ] Input validation prevents injection attacks

- [ ] 令牌验证拒绝过期令牌
- [ ] 令牌验证拒绝无效签名
- [ ] 令牌验证拒绝篡改的载荷
- [ ] 速率限制防止 DoS 攻击
- [ ] 敏感数据未被记录
- [ ] TLS 配置正确
- [ ] 密钥未硬编码
- [ ] 输入验证防止注入攻击

---

## Load Testing | 负载测试

### Go Native Load Tests | Go 原生负载测试

```bash
# Run all load tests
go test -v ./tests/load/... -count=1 -timeout 10m

# Run specific load test
go test -v ./tests/load/... -run TestLoadOAuth2TokenCache -count=1

# Run stress test
go test -v ./tests/load/... -run TestStressHighConcurrency -count=1

# Run stability test
go test -v ./tests/load/... -run TestStabilityLongRunning -count=1 -timeout 5m
```

### k6 Load Tests | k6 负载测试

```bash
# Install k6
brew install k6  # macOS
# or
sudo apt-get install k6  # Linux

# Run smoke test
k6 run -e SCENARIO=smoke tests/load/k6-script.js

# Run load test
k6 run -e SCENARIO=load tests/load/k6-script.js

# Run stress test
k6 run -e SCENARIO=stress tests/load/k6-script.js

# Run stability test
k6 run -e SCENARIO=stability tests/load/k6-script.js

# Custom configuration
k6 run --vus 100 --duration 30s tests/load/k6-script.js
```

### Load Test Scenarios | 负载测试场景

| Scenario | Duration | Target RPS | Purpose |
|----------|----------|------------|---------|
| Smoke | 30s | 1 VU | Functionality verification |
| Load | 8 min | 10,000 | Performance validation |
| Stress | 5 min | 20,000 | Find breaking points |
| Stability | 30 min | 5,000 | Memory leak detection |

| 场景 | 持续时间 | 目标 RPS | 目的 |
|------|----------|----------|------|
| 冒烟测试 | 30秒 | 1 VU | 功能验证 |
| 负载测试 | 8分钟 | 10,000 | 性能验证 |
| 压力测试 | 5分钟 | 20,000 | 发现断点 |
| 稳定性测试 | 30分钟 | 5,000 | 内存泄漏检测 |

---

## CI/CD Integration | CI/CD 集成

### GitHub Actions | GitHub Actions

```yaml
# .github/workflows/security-tests.yml
name: Security Tests

on:
  push:
    branches: [master, main]
  pull_request:
    branches: [master, main]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Run Unit Tests
        run: go test -v -race ./pkg/security/... -count=1
      
      - name: Run Coverage
        run: |
          go test -coverprofile=coverage.out ./pkg/security/...
          go tool cover -html=coverage.out -o coverage.html
      
      - name: Upload Coverage
        uses: actions/upload-artifact@v4
        with:
          name: coverage-report
          path: coverage.html

  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Install Security Tools
        run: |
          go install github.com/securego/gosec/v2/cmd/gosec@latest
          go install golang.org/x/vuln/cmd/govulncheck@latest
      
      - name: Run gosec
        run: gosec -fmt=json -out=gosec-report.json ./pkg/security/...
        continue-on-error: true
      
      - name: Run govulncheck
        run: govulncheck ./pkg/security/...
      
      - name: Upload Security Report
        uses: actions/upload-artifact@v4
        with:
          name: security-report
          path: gosec-report.json

  load-tests:
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Run Load Tests
        run: go test -v ./tests/load/... -run "^TestLoad" -count=1 -timeout 10m

  k6-tests:
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
      - uses: actions/checkout@v4
      
      - name: Run k6 Smoke Test
        uses: grafana/k6-action@v0.3.0
        with:
          filename: tests/load/k6-script.js
          flags: -e SCENARIO=smoke
```

### GitLab CI | GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - test
  - security
  - performance

unit-tests:
  stage: test
  image: golang:1.23
  script:
    - go test -v -race ./pkg/security/... -count=1
    - go test -coverprofile=coverage.out ./pkg/security/...
  artifacts:
    paths:
      - coverage.out

security-scan:
  stage: security
  image: golang:1.23
  script:
    - go install github.com/securego/gosec/v2/cmd/gosec@latest
    - go install golang.org/x/vuln/cmd/govulncheck@latest
    - gosec -fmt=json -out=gosec-report.json ./pkg/security/...
    - govulncheck ./pkg/security/...
  artifacts:
    paths:
      - gosec-report.json
  allow_failure: true

load-tests:
  stage: performance
  image: golang:1.23
  script:
    - go test -v ./tests/load/... -count=1 -timeout 10m
  only:
    - master
    - main
```

### Coverage Requirements | 覆盖率要求

| Component | Minimum Coverage |
|-----------|------------------|
| JWT | 85% |
| OAuth2 | 85% |
| OPA | 85% |
| Middleware | 80% |
| Overall | 85% |

| 组件 | 最低覆盖率 |
|------|-----------|
| JWT | 85% |
| OAuth2 | 85% |
| OPA | 85% |
| 中间件 | 80% |
| 总体 | 85% |

---

## Troubleshooting | 故障排除

### Common Issues | 常见问题

#### 1. Tests Timeout | 测试超时

```bash
# Increase timeout
go test -v ./... -timeout 10m

# Skip long-running tests
go test -v -short ./...
```

#### 2. Race Conditions | 竞态条件

```bash
# Run with race detector
go test -v -race ./pkg/security/...

# If race detected, check for:
# - Shared state without synchronization
# - Concurrent map access
# - Goroutine leaks
```

#### 3. Docker Connection Issues | Docker 连接问题

```bash
# Check Docker is running
docker info

# Check containers are up
docker-compose -f tests/security/docker-compose.test.yml ps

# View container logs
docker-compose -f tests/security/docker-compose.test.yml logs
```

#### 4. Coverage Report Issues | 覆盖率报告问题

```bash
# Generate coverage for specific package
go test -coverprofile=coverage.out -covermode=atomic ./pkg/security/jwt/...

# View coverage in browser
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out | grep total
```

### Debug Tips | 调试技巧

```bash
# Verbose output
go test -v ./pkg/security/... -count=1

# Run single test
go test -v ./pkg/security/jwt/... -run TestValidator_ValidToken -count=1

# Print test logs
go test -v ./pkg/security/... 2>&1 | tee test.log

# Use delve for debugging
dlv test ./pkg/security/jwt/... -- -test.run TestValidator_ValidToken
```

### Performance Debugging | 性能调试

```bash
# CPU profiling
go test -v ./pkg/security/jwt/... -bench=. -cpuprofile=cpu.prof
go tool pprof -http=:8080 cpu.prof

# Memory profiling
go test -v ./pkg/security/jwt/... -bench=. -memprofile=mem.prof
go tool pprof -http=:8080 mem.prof

# Trace
go test -v ./pkg/security/jwt/... -trace=trace.out
go tool trace trace.out
```

---

## Best Practices | 最佳实践

### Test Writing | 编写测试

1. **Use table-driven tests** for comprehensive coverage
2. **Mock external dependencies** for unit tests
3. **Use test fixtures** for consistent test data
4. **Clean up resources** in test teardown
5. **Use meaningful assertions** with clear error messages

1. **使用表驱动测试** 以获得全面覆盖
2. **模拟外部依赖** 用于单元测试
3. **使用测试夹具** 获得一致的测试数据
4. **清理资源** 在测试拆卸时
5. **使用有意义的断言** 带有清晰的错误消息

### Test Organization | 测试组织

1. Keep test files alongside source files (`*_test.go`)
2. Use separate packages for integration tests (`_test` suffix)
3. Group related tests using subtests
4. Use test helpers for common operations
5. Document complex test scenarios

1. 将测试文件与源文件放在一起 (`*_test.go`)
2. 为集成测试使用单独的包（`_test` 后缀）
3. 使用子测试分组相关测试
4. 为常见操作使用测试助手
5. 记录复杂的测试场景

### CI/CD | CI/CD

1. Run fast tests first (unit tests)
2. Run security scans on every PR
3. Run load tests only on main branch
4. Store test artifacts for debugging
5. Set coverage thresholds

1. 首先运行快速测试（单元测试）
2. 在每个 PR 上运行安全扫描
3. 仅在主分支上运行负载测试
4. 存储测试工件用于调试
5. 设置覆盖率阈值

---

## Related Documentation | 相关文档

- [Security Best Practices](./security-best-practices.md)
- [Security Configuration Reference](./security-configuration-reference.md)
- [Performance Monitoring Guide](./performance-monitoring-alerting.md)
- [Load Testing Guide](../tests/load/README.md)
- [OPA Policy Guide](./opa-policy-guide.md)
- [OAuth2 Integration Guide](./oauth2-integration-guide.md)

---

*Last updated: November 2025*

*最后更新：2025年11月*







