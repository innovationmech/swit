# Saga Recovery Monitoring and Alerting Guide

本指南介绍如何使用 Saga 恢复系统的监控和告警功能，包括 Prometheus 指标集成、告警规则配置以及故障排查方法。

## 目录

- [概述](#概述)
- [Prometheus 指标](#prometheus-指标)
- [告警配置](#告警配置)
- [监控最佳实践](#监控最佳实践)
- [故障排查](#故障排查)

## 概述

Saga 恢复系统提供了完整的监控和告警功能，用于跟踪恢复操作的性能和可靠性。主要功能包括:

- **Prometheus 指标**: 记录恢复尝试、成功率、耗时等关键指标
- **自动告警**: 基于指标阈值的自动告警机制
- **恢复事件**: 发布恢复生命周期事件供外部系统订阅
- **健康检查**: 定期检查 Saga 实例的健康状态

## Prometheus 指标

### 可用指标

恢复系统导出以下 Prometheus 指标：

#### 恢复尝试计数器
```
saga_recovery_attempts_total{strategy="resume|compensate|retry", saga_type="<saga_definition_id>"}
```
记录恢复尝试的总次数，按策略和 Saga 类型分类。

#### 恢复成功计数器
```
saga_recovery_success_total{strategy="resume|compensate|retry", saga_type="<saga_definition_id>"}
```
记录成功恢复的总次数。

#### 恢复失败计数器
```
saga_recovery_failure_total{strategy="resume|compensate|retry", saga_type="<saga_definition_id>", error_type="timeout|max_attempts_exceeded|no_strategy|execution_error"}
```
记录失败恢复的总次数，包含错误类型。

#### 恢复耗时直方图
```
saga_recovery_duration_seconds{strategy="resume|compensate|retry", saga_type="<saga_definition_id>"}
```
记录恢复操作的耗时分布，单位为秒。

**默认分桶**: 0.5, 1.0, 2.5, 5.0, 10.0, 15.0, 20.0, 30.0, 60.0, 120.0

#### 恢复中 Gauge
```
saga_recovery_in_progress{strategy="resume|compensate|retry"}
```
当前正在恢复的 Saga 数量。

#### 检测到的故障计数器
```
saga_recovery_detected_failures_total{failure_type="timeout|stuck|compensating|inconsistent"}
```
按类型记录检测到的故障数量。

#### 手动干预计数器
```
saga_recovery_manual_interventions_total
```
记录手动干预操作的总次数。

#### 恢复成功率 Summary
```
saga_recovery_success_rate
```
恢复成功率的摘要统计（P50, P90, P99）。

### 指标查询示例

**查看恢复成功率**:
```promql
rate(saga_recovery_success_total[5m]) / rate(saga_recovery_attempts_total[5m])
```

**查看平均恢复时间**:
```promql
rate(saga_recovery_duration_seconds_sum[5m]) / rate(saga_recovery_duration_seconds_count[5m])
```

**查看 P95 恢复时间**:
```promql
histogram_quantile(0.95, rate(saga_recovery_duration_seconds_bucket[5m]))
```

**按策略查看恢复尝试次数**:
```promql
sum(rate(saga_recovery_attempts_total[5m])) by (strategy)
```

## 告警配置

### 内置告警规则

恢复系统提供以下内置告警规则：

1. **HighRecoveryFailureRate**: 恢复失败率超过阈值
2. **TooManyStuckSagas**: 卡住的 Saga 数量过多
3. **SlowRecovery**: 平均恢复时间过长
4. **HighManualInterventionRate**: 手动干预率过高

### 告警配置参数

```yaml
recovery:
  alerting:
    enabled: true
    check_interval: 30s
    deduplication_window: 5m
    max_alerts_per_minute: 10
    high_failure_rate_threshold: 0.1  # 10%
    too_many_stuck_sagas_threshold: 10
    slow_recovery_threshold: 30s
```

### Prometheus 告警规则示例

创建 `alerts/saga-recovery.yml`:

```yaml
groups:
  - name: saga_recovery
    interval: 30s
    rules:
      # 高失败率告警
      - alert: SagaRecoveryHighFailureRate
        expr: |
          (
            rate(saga_recovery_failure_total[5m])
            / 
            rate(saga_recovery_attempts_total[5m])
          ) > 0.1
        for: 5m
        labels:
          severity: warning
          component: saga_recovery
        annotations:
          summary: "Saga 恢复失败率过高"
          description: "过去5分钟恢复失败率超过 10%: {{ $value | humanizePercentage }}"
          
      # 太多卡住的 Sagas
      - alert: SagaTooManyStuckSagas
        expr: saga_recovery_detected_failures_total{failure_type="stuck"} > 10
        for: 10m
        labels:
          severity: critical
          component: saga_recovery
        annotations:
          summary: "卡住的 Saga 数量过多"
          description: "当前有 {{ $value }} 个 Saga 处于卡住状态"
          
      # 恢复时间过长
      - alert: SagaRecoverySlowPerformance
        expr: |
          (
            rate(saga_recovery_duration_seconds_sum[5m])
            /
            rate(saga_recovery_duration_seconds_count[5m])
          ) > 30
        for: 10m
        labels:
          severity: warning
          component: saga_recovery
        annotations:
          summary: "Saga 恢复性能下降"
          description: "平均恢复时间超过 30 秒: {{ $value }}s"
          
      # P95 恢复时间过长
      - alert: SagaRecoveryP95Slow
        expr: |
          histogram_quantile(0.95, 
            rate(saga_recovery_duration_seconds_bucket[5m])
          ) > 60
        for: 10m
        labels:
          severity: warning
          component: saga_recovery
        annotations:
          summary: "Saga 恢复 P95 时间过长"
          description: "P95 恢复时间超过 60 秒: {{ $value }}s"
          
      # 恢复中的 Saga 数量异常
      - alert: SagaTooManyRecovering
        expr: saga_recovery_in_progress > 50
        for: 15m
        labels:
          severity: warning
          component: saga_recovery
        annotations:
          summary: "同时恢复的 Saga 过多"
          description: "当前有 {{ $value }} 个 Saga 正在恢复"
          
      # 手动干预率过高
      - alert: SagaHighManualInterventionRate
        expr: |
          (
            rate(saga_recovery_manual_interventions_total[1h])
            /
            rate(saga_recovery_attempts_total[1h])
          ) > 0.2
        for: 1h
        labels:
          severity: info
          component: saga_recovery
        annotations:
          summary: "手动干预率过高"
          description: "过去1小时手动干预率超过 20%"
```

### 自定义告警规则

可以通过代码添加自定义告警规则：

```go
package main

import (
	"github.com/innovationmech/swit/pkg/saga/state"
	"go.uber.org/zap"
)

func setupCustomAlerts(recoveryManager *state.RecoveryManager, logger *zap.Logger) {
	// 获取告警管理器
	alertingManager := recoveryManager.GetAlertingManager()
	if alertingManager == nil {
		return
	}

	// 添加自定义规则
	alertingManager.AddRule(&state.AlertRule{
		Name:        "CustomHighTimeoutRate",
		Description: "超时率超过阈值",
		Severity:    state.AlertSeverityWarning,
		Enabled:     true,
		CheckFunc: func(snapshot *state.RecoveryMetricsSnapshot) (bool, interface{}, interface{}) {
			if snapshot.TotalAttempts < 10 {
				return false, nil, nil
			}
			timeoutRate := float64(snapshot.DetectedTimeouts) / float64(snapshot.TotalAttempts)
			threshold := 0.15 // 15%
			return timeoutRate > threshold, timeoutRate, threshold
		},
	})
}
```

### 告警通知

实现自定义告警通知器：

```go
package main

import (
	"context"
	"fmt"

	"github.com/innovationmech/swit/pkg/saga/state"
)

// SlackNotifier 发送告警到 Slack
type SlackNotifier struct {
	webhookURL string
}

func (n *SlackNotifier) Notify(ctx context.Context, alert *state.RecoveryAlert) error {
	message := fmt.Sprintf(
		"🚨 *%s*\n"+
			"严重程度: %s\n"+
			"描述: %s\n"+
			"当前值: %v\n"+
			"阈值: %v",
		alert.Name,
		alert.Severity,
		alert.Description,
		alert.Value,
		alert.Threshold,
	)
	
	// 发送到 Slack (实现略)
	return nil
}

// 注册通知器
func setupNotifiers(recoveryManager *state.RecoveryManager) {
	notifier := &SlackNotifier{
		webhookURL: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
	}
	recoveryManager.AddAlertNotifier(notifier)
}
```

## 监控最佳实践

### 1. 设置合理的阈值

根据业务特性设置告警阈值：

- **失败率阈值**: 通常设置为 5-10%
- **恢复时间阈值**: 根据 SLA 要求设置，一般 10-30 秒
- **卡住 Saga 阈值**: 根据系统规模设置，如 10-50 个

### 2. 使用 Grafana 仪表盘

创建 Grafana 仪表盘监控关键指标：

```json
{
  "dashboard": {
    "title": "Saga Recovery Monitoring",
    "panels": [
      {
        "title": "Recovery Success Rate",
        "targets": [{
          "expr": "rate(saga_recovery_success_total[5m]) / rate(saga_recovery_attempts_total[5m])"
        }]
      },
      {
        "title": "Recovery Duration P95",
        "targets": [{
          "expr": "histogram_quantile(0.95, rate(saga_recovery_duration_seconds_bucket[5m]))"
        }]
      },
      {
        "title": "Detected Failures by Type",
        "targets": [{
          "expr": "sum(rate(saga_recovery_detected_failures_total[5m])) by (failure_type)"
        }]
      }
    ]
  }
}
```

### 3. 监控趋势

不仅监控当前值，还要关注趋势变化：

```promql
# 恢复成功率的1小时环比变化
(
  rate(saga_recovery_success_total[5m])
  / 
  rate(saga_recovery_attempts_total[5m])
)
- 
(
  rate(saga_recovery_success_total[5m] offset 1h)
  / 
  rate(saga_recovery_attempts_total[5m] offset 1h)
)
```

### 4. 关联日志和追踪

将指标与日志和分布式追踪关联：

- 在日志中包含 saga_id、recovery_strategy 等标签
- 使用 OpenTelemetry 追踪恢复操作
- 在告警中包含相关日志查询链接

## 故障排查

### 高失败率排查

1. **检查失败类型分布**:
   ```promql
   sum(rate(saga_recovery_failure_total[5m])) by (error_type)
   ```

2. **查看失败的 Saga 类型**:
   ```promql
   sum(rate(saga_recovery_failure_total[5m])) by (saga_type)
   ```

3. **检查日志**:
   ```bash
   kubectl logs <pod-name> | grep "saga recovery failed"
   ```

### 恢复时间过长排查

1. **按策略分析**:
   ```promql
   histogram_quantile(0.95, 
     rate(saga_recovery_duration_seconds_bucket[5m])
   ) by (strategy)
   ```

2. **检查资源使用**:
   - CPU 使用率
   - 内存使用率
   - 数据库连接池

3. **分析慢查询**:
   - 检查状态存储的查询性能
   - 检查 Saga 步骤执行时间

### 卡住 Saga 排查

1. **检查卡住原因**:
   ```bash
   # 查看健康检查报告
   curl http://localhost:8080/admin/recovery/health
   ```

2. **手动恢复**:
   ```bash
   # 强制补偿
   curl -X POST http://localhost:8080/admin/recovery/compensate/<saga_id>
   
   # 标记为失败
   curl -X POST http://localhost:8080/admin/recovery/fail/<saga_id>
   ```

3. **检查依赖服务**:
   - 数据库连接状态
   - 消息队列状态
   - 下游服务可用性

## 配置示例

### 完整配置示例

```yaml
# swit.yaml
saga:
  recovery:
    # 基本配置
    enabled: true
    check_interval: 30s
    recovery_timeout: 30s
    max_concurrent_recoveries: 10
    max_recovery_attempts: 3
    recovery_backoff: 5s
    enable_auto_recovery: true
    
    # 检测配置
    detection:
      timeout_threshold: 30s
      stuck_threshold: 5m
      enable_inconsistency_check: true
      max_results_per_scan: 100
      
    # 告警配置
    alerting:
      enabled: true
      check_interval: 30s
      deduplication_window: 5m
      max_alerts_per_minute: 10
      high_failure_rate_threshold: 0.1
      too_many_stuck_sagas_threshold: 10
      slow_recovery_threshold: 30s
      
  # Prometheus 指标配置
  metrics:
    enabled: true
    namespace: "saga"
    subsystem: "recovery"
    duration_buckets: [0.5, 1.0, 2.5, 5.0, 10.0, 15.0, 20.0, 30.0, 60.0, 120.0]
```

## 参考资料

- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)
- [Saga 恢复机制设计](./saga-user-guide.md)
- [性能监控最佳实践](./performance-monitoring-alerting.md)

