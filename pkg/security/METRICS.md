# Security Metrics

本包提供全面的安全相关 Prometheus 指标采集功能，用于监控认证、授权、安全事件、漏洞扫描和 TLS 连接等安全操作。

## 功能特性

- **认证指标**：监控认证尝试、令牌验证、令牌刷新
- **授权指标**：跟踪策略评估、访问拒绝事件
- **安全事件**：记录各类安全事件（如暴力破解、可疑活动）
- **漏洞指标**：追踪漏洞扫描结果和发现的漏洞数量
- **TLS/连接指标**：监控 TLS 连接建立和握手性能
- **会话指标**：跟踪活跃会话、会话创建和撤销

## 快速开始

### 基本使用

```go
package main

import (
	"time"
	"github.com/innovationmech/swit/pkg/security"
)

func main() {
	// 创建安全指标收集器（使用默认配置）
	metrics, err := security.NewSecurityMetrics(nil)
	if err != nil {
		panic(err)
	}

	// 记录认证尝试
	start := time.Now()
	metrics.RecordAuthAttempt("jwt", "success")
	metrics.RecordAuthDuration("jwt", time.Since(start).Seconds())

	// 记录策略评估
	metrics.RecordPolicyEvaluation("rbac", "allow")
	
	// 记录 TLS 连接
	metrics.RecordTLSConnection("1.3", "TLS_AES_128_GCM_SHA256")
}
```

### 自定义配置

```go
config := &security.SecurityMetricsConfig{
	Namespace: "myapp",
	Subsystem: "security",
	Registry:  prometheus.NewRegistry(),
	DurationBuckets: []float64{0.001, 0.01, 0.1, 1.0, 10.0},
}

metrics, err := security.NewSecurityMetrics(config)
```

### 暴露 Prometheus 指标端点

```go
import (
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

http.Handle("/metrics", promhttp.HandlerFor(
	metrics.GetRegistry(),
	promhttp.HandlerOpts{},
))

http.ListenAndServe(":9090", nil)
```

## 指标清单

### 认证指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `security_auth_attempts_total` | Counter | `result`, `method` | 认证尝试总数 |
| `security_auth_duration_seconds` | Histogram | `method` | 认证操作耗时 |
| `security_token_validations_total` | Counter | `result` | 令牌验证总数 |
| `security_token_refreshes_total` | Counter | `result` | 令牌刷新总数 |

**标签值示例：**
- `result`: `success`, `failure`, `error`, `expired`, `invalid`
- `method`: `jwt`, `oauth2`, `basic`, `api_key`

### 授权指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `security_policy_evaluations_total` | Counter | `decision`, `policy` | 策略评估总数 |
| `security_policy_evaluation_duration_seconds` | Histogram | `policy` | 策略评估耗时 |
| `security_access_denied_total` | Counter | `reason`, `resource` | 访问拒绝总数 |

**标签值示例：**
- `decision`: `allow`, `deny`, `error`
- `policy`: `rbac`, `abac`, `acl`
- `reason`: `insufficient_permissions`, `invalid_token`, `rate_limit`

### 安全事件指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `security_security_events_total` | Counter | `type`, `severity` | 安全事件总数 |

**标签值示例：**
- `type`: `brute_force`, `suspicious_activity`, `unauthorized_access`
- `severity`: `low`, `medium`, `high`, `critical`

### 漏洞指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `security_vulnerabilities_found` | Gauge | `severity`, `tool` | 当前发现的漏洞数量 |
| `security_security_scans_total` | Counter | `tool`, `result` | 安全扫描执行总数 |

**标签值示例：**
- `severity`: `low`, `medium`, `high`, `critical`
- `tool`: `gosec`, `govulncheck`, `trivy`
- `result`: `success`, `failure`, `error`

### TLS/连接指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `security_tls_connections_total` | Counter | `version`, `cipher_suite` | TLS 连接总数 |
| `security_tls_handshake_duration_seconds` | Histogram | `version` | TLS 握手耗时 |
| `security_certificate_expiry_seconds` | Gauge | `common_name` | 证书到期时间（秒） |

**标签值示例：**
- `version`: `1.2`, `1.3`
- `cipher_suite`: `TLS_AES_128_GCM_SHA256`, `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`

### 会话指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `security_active_sessions` | Gauge | - | 当前活跃会话数 |
| `security_session_created_total` | Counter | `type` | 会话创建总数 |
| `security_session_revoked_total` | Counter | `reason` | 会话撤销总数 |

**标签值示例：**
- `type`: `user`, `service`, `temporary`
- `reason`: `logout`, `timeout`, `admin_action`, `security_breach`

## API 参考

### 认证方法

```go
// 记录认证尝试
RecordAuthAttempt(method, result string)

// 记录认证耗时
RecordAuthDuration(method string, durationSeconds float64)

// 记录令牌验证
RecordTokenValidation(result string)

// 记录令牌刷新
RecordTokenRefresh(result string)
```

### 授权方法

```go
// 记录策略评估
RecordPolicyEvaluation(policy, decision string)

// 记录策略评估耗时
RecordPolicyEvaluationDuration(policy string, durationSeconds float64)

// 记录访问拒绝
RecordAccessDenied(reason, resource string)
```

### 安全事件方法

```go
// 记录安全事件
RecordSecurityEvent(eventType, severity string)
```

### 漏洞方法

```go
// 设置发现的漏洞数量
SetVulnerabilitiesFound(severity, tool string, count float64)

// 记录安全扫描
RecordSecurityScan(tool, result string)
```

### TLS/连接方法

```go
// 记录 TLS 连接
RecordTLSConnection(version, cipherSuite string)

// 记录 TLS 握手耗时
RecordTLSHandshakeDuration(version string, durationSeconds float64)

// 设置证书到期时间
SetCertificateExpiry(commonName string, secondsUntilExpiry float64)
```

### 会话方法

```go
// 设置活跃会话数
SetActiveSessions(count float64)

// 增加活跃会话数
IncreaseActiveSessions()

// 减少活跃会话数
DecreaseActiveSessions()

// 记录会话创建
RecordSessionCreated(sessionType string)

// 记录会话撤销
RecordSessionRevoked(reason string)
```

## 集成示例

### JWT 认证集成

```go
func (v *JWTValidator) ValidateToken(tokenString string) (*Claims, error) {
	start := time.Now()
	defer func() {
		v.metrics.RecordAuthDuration("jwt", time.Since(start).Seconds())
	}()

	claims, err := v.parseToken(tokenString)
	if err != nil {
		v.metrics.RecordAuthAttempt("jwt", "failure")
		v.metrics.RecordTokenValidation("invalid")
		return nil, err
	}

	if claims.ExpiresAt.Before(time.Now()) {
		v.metrics.RecordTokenValidation("expired")
		return nil, ErrTokenExpired
	}

	v.metrics.RecordAuthAttempt("jwt", "success")
	v.metrics.RecordTokenValidation("success")
	return claims, nil
}
```

### OPA 策略评估集成

```go
func (e *Evaluator) Evaluate(ctx context.Context, policy string, input interface{}) (bool, error) {
	start := time.Now()
	defer func() {
		e.metrics.RecordPolicyEvaluationDuration(policy, time.Since(start).Seconds())
	}()

	result, err := e.client.Evaluate(ctx, policy, input)
	if err != nil {
		e.metrics.RecordPolicyEvaluation(policy, "error")
		return false, err
	}

	decision := "deny"
	if result {
		decision = "allow"
	}
	e.metrics.RecordPolicyEvaluation(policy, decision)

	if !result {
		e.metrics.RecordAccessDenied("policy_denied", extractResource(input))
	}

	return result, nil
}
```

### TLS 连接监控集成

```go
func (s *Server) handleConnection(conn *tls.Conn) {
	start := time.Now()
	
	if err := conn.Handshake(); err != nil {
		s.logger.Error("TLS handshake failed", zap.Error(err))
		return
	}
	
	state := conn.ConnectionState()
	version := getTLSVersion(state.Version)
	cipherSuite := tls.CipherSuiteName(state.CipherSuite)
	
	s.metrics.RecordTLSConnection(version, cipherSuite)
	s.metrics.RecordTLSHandshakeDuration(version, time.Since(start).Seconds())
	
	// Handle connection...
}
```

### 漏洞扫描集成

```go
func (s *Scanner) RunScan() error {
	results, err := s.runGosec()
	if err != nil {
		s.metrics.RecordSecurityScan("gosec", "failure")
		return err
	}

	s.metrics.RecordSecurityScan("gosec", "success")
	
	// 按严重程度分类漏洞
	severityCounts := make(map[string]float64)
	for _, vuln := range results {
		severityCounts[vuln.Severity]++
	}

	for severity, count := range severityCounts {
		s.metrics.SetVulnerabilitiesFound(severity, "gosec", count)
	}

	return nil
}
```

## 最佳实践

### 1. 指标粒度

- 使用合适的标签来分类指标，但避免高基数标签（如用户 ID）
- 对于高频操作，使用采样来减少开销

### 2. 性能考虑

- 指标记录操作是线程安全的但会有轻微性能开销
- 对于关键路径，考虑异步记录指标
- 使用合适的直方图桶配置

### 3. 告警配置

建议在 Prometheus 中配置以下告警：

```yaml
groups:
  - name: security_alerts
    rules:
      # 高认证失败率
      - alert: HighAuthFailureRate
        expr: rate(security_auth_attempts_total{result="failure"}[5m]) > 10
        annotations:
          summary: "High authentication failure rate detected"

      # 访问拒绝激增
      - alert: AccessDeniedSpike
        expr: rate(security_access_denied_total[5m]) > 50
        annotations:
          summary: "Unusual number of access denied events"

      # 发现高危漏洞
      - alert: CriticalVulnerabilitiesFound
        expr: security_vulnerabilities_found{severity="critical"} > 0
        annotations:
          summary: "Critical vulnerabilities detected"

      # 证书即将过期
      - alert: CertificateExpiringSoon
        expr: security_certificate_expiry_seconds < 604800  # 7 days
        annotations:
          summary: "TLS certificate expiring within 7 days"
```

### 4. 仪表板配置

使用 Grafana 创建安全监控仪表板：

- **认证面板**：显示成功/失败率、延迟分布
- **授权面板**：策略评估决策比例、访问拒绝趋势
- **安全事件面板**：按严重程度分类的事件时间线
- **漏洞面板**：当前漏洞分布、扫描历史
- **TLS 面板**：连接数、握手延迟、证书到期警告
- **会话面板**：活跃会话趋势、会话创建/撤销率

## 参考

- [Issue #808](https://github.com/innovationmech/swit/issues/808)
- [Epic #777](https://github.com/innovationmech/swit/issues/777)
- [Prometheus 最佳实践](https://prometheus.io/docs/practices/)
- [示例代码](../../examples/security/metrics-example.go)

